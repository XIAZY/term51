// Package cells prepares what the GPU draws the terminal from: a table with
// one record per cell, and a sheet holding the shape of every character used
// so far. Both are plain Go memory with no dependency on the window or on
// OpenGL, so the package runs and is tested on any machine.
package cells

import (
	"image"
	"image/draw"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// SheetSize is the width and height of the character sheet in pixels.
const SheetSize = 2048

// Fonts are the fonts the characters are drawn from.
type Fonts struct {
	// Faces are the monospace font's regular, bold, italic and bold italic
	// styles. The regular one decides the size of a cell.
	Faces [4]*opentype.Font
	// Others are searched, in order, for a character the monospace font
	// lacks (Chinese, for one). They have one style each, used for all four.
	Others []*opentype.Font
}

// Sheet is the character sheet: one 8-bit coverage picture cut into slots
// the size of a cell. Slot n is at column n % PerRow, row n / PerRow. A wide
// character takes two slots following each other, its left and right half.
// Slot 0 is never given out: in the table it means "nothing to draw".
//
// A Sheet is not safe for use from several goroutines.
type Sheet struct {
	// Pix is the picture, SheetSize by SheetSize bytes, row by row.
	Pix []byte
	// PerRow is how many slots there are across the sheet.
	PerRow int

	faces  [4][]font.Face // per style: the monospace face, then the others
	cellW  int
	cellH  int
	ascent int
	slots  map[glyphKey]uint16
	next   int
	total  int
	gen    int
	dirty  image.Rectangle
}

type glyphKey struct {
	text  string // the cell's character and its combining marks
	style uint8
	wide  bool
}

// NewSheet makes an empty sheet for the fonts at the given size in pixels
// (the height of the font's em square).
func NewSheet(fonts Fonts, sizePx float64) (*Sheet, error) {
	s := &Sheet{Pix: make([]byte, SheetSize*SheetSize), slots: map[glyphKey]uint16{}, next: 1}
	opts := &opentype.FaceOptions{Size: sizePx, DPI: 72, Hinting: font.HintingFull}
	var others []font.Face
	for _, f := range fonts.Others {
		face, err := opentype.NewFace(f, opts)
		if err != nil {
			return nil, err
		}
		others = append(others, face)
	}
	for i, f := range fonts.Faces {
		face, err := opentype.NewFace(f, opts)
		if err != nil {
			return nil, err
		}
		s.faces[i] = append([]font.Face{face}, others...)
	}
	regular := s.faces[0][0]
	m := regular.Metrics()
	adv, _ := regular.GlyphAdvance('M')
	s.cellW = int(math.Ceil(float64(adv) / 64))
	s.ascent = m.Ascent.Ceil()
	s.cellH = s.ascent + m.Descent.Ceil()
	s.PerRow = SheetSize / s.cellW
	s.total = s.PerRow * (SheetSize / s.cellH)
	return s, nil
}

// CellSize is the size of one cell, and of one slot, in pixels.
func (s *Sheet) CellSize() (w, h int) { return s.cellW, s.cellH }

// Grid is how many columns and rows fit in w by h pixels.
func (s *Sheet) Grid(w, h int) (cols, rows int) { return w / s.cellW, h / s.cellH }

// Generation changes when the sheet filled up and was started again: every
// slot number handed out before is then meaningless.
func (s *Sheet) Generation() int { return s.gen }

// TakeDirty returns the part of Pix that changed since the last call (empty
// when nothing did) and forgets it.
func (s *Sheet) TakeDirty() image.Rectangle {
	d := s.dirty
	s.dirty = image.Rectangle{}
	return d
}

// slot returns the slot holding the character text in the given style (bit 0
// bold, bit 1 italic), drawing it first if this is its first use. A wide
// character's right half is in the slot after the one returned.
func (s *Sheet) slot(text []rune, style uint8, wide bool) uint16 {
	key := glyphKey{string(text), style, wide}
	if n, ok := s.slots[key]; ok {
		return n
	}
	need := 1
	if wide {
		need = 2
	}
	if s.next+need > s.total {
		// Full: a session that goes through a large script would otherwise
		// need a sheet without bound. Start again; the table notices by the
		// generation and asks for every character on screen anew.
		clear(s.slots)
		clear(s.Pix)
		s.next = 1
		s.gen++
		s.dirty = image.Rect(0, 0, SheetSize, SheetSize)
	}
	n := s.next
	s.next += need

	w := s.cellW * need
	img := image.NewAlpha(image.Rect(0, 0, w, s.cellH))
	faces := s.faces[style]
	base := s.face(faces, text[0])
	if base == nil {
		// No font has the character: a hollow box, so that it is visibly
		// there and takes its cells.
		box := image.Rect(1, 2, w-1, s.cellH-2)
		draw.Draw(img, box, image.Opaque, image.Point{}, draw.Src)
		draw.Draw(img, box.Inset(1), image.Transparent, image.Point{}, draw.Src)
	} else {
		// A wide character is centred in its two cells; its font's advance
		// is not twice the monospace font's.
		x := fixed.Int26_6(0)
		if adv, _ := base.GlyphAdvance(text[0]); wide && adv < fixed.I(w) {
			x = (fixed.I(w) - adv) / 2
		}
		for _, r := range text {
			// Combining marks go on top of the character, from the same
			// origin. A mark no font has is left out, which keeps the
			// character readable.
			if f := s.face(faces, r); f != nil {
				d := font.Drawer{Dst: img, Src: image.Opaque, Face: f, Dot: fixed.Point26_6{X: x, Y: fixed.I(s.ascent)}}
				d.DrawString(string(r))
			}
		}
	}
	for i := 0; i < need; i++ {
		at := s.rect(n + i)
		for y := 0; y < s.cellH; y++ {
			copy(s.Pix[(at.Min.Y+y)*SheetSize+at.Min.X:][:s.cellW], img.Pix[y*img.Stride+i*s.cellW:])
		}
		s.dirty = s.dirty.Union(at)
	}
	s.slots[key] = uint16(n)
	return uint16(n)
}

// face is the first of faces that has the character, or nil.
func (s *Sheet) face(faces []font.Face, r rune) font.Face {
	for _, f := range faces {
		if _, ok := f.GlyphAdvance(r); ok {
			return f
		}
	}
	return nil
}

// rect is where slot n is in Pix.
func (s *Sheet) rect(n int) image.Rectangle {
	x, y := n%s.PerRow*s.cellW, n/s.PerRow*s.cellH
	return image.Rect(x, y, x+s.cellW, y+s.cellH)
}
