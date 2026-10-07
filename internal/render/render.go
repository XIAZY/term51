// Package render draws the terminal's render state into a pixel buffer.
//
// It is plain Go on a []uint32 (0x00RRGGBB), with no dependency on the
// window it ends up in, so it runs and is tested on any machine. Glyphs are
// rasterised once per (character, weight, slant) into cell-sized coverage
// masks and blended from the cache afterwards.
package render

import (
	"image"
	"image/draw"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonobolditalic"
	"golang.org/x/image/font/gofont/gomonoitalic"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/XIAZY/libghostty-vt-go/vt"
	"github.com/XIAZY/libghostty-vt-go/vt/color"
	"github.com/XIAZY/libghostty-vt-go/vt/page"
	"github.com/XIAZY/libghostty-vt-go/vt/sgr"
)

// Renderer holds the font and the glyph cache. It is not safe for use from
// several goroutines.
type Renderer struct {
	faces  [4]font.Face // regular, bold, italic, bold italic
	cellW  int
	cellH  int
	ascent int
	cache  map[glyphKey][]uint8
}

type glyphKey struct {
	r     rune
	face  uint8
	cells uint8 // 1, or 2 for a wide character
}

// New makes a renderer with the built-in monospace font at the given size in
// pixels (the height of the font's em square).
func New(sizePx float64) (*Renderer, error) {
	r := &Renderer{cache: map[glyphKey][]uint8{}}
	for i, ttf := range [][]byte{gomono.TTF, gomonobold.TTF, gomonoitalic.TTF, gomonobolditalic.TTF} {
		f, err := opentype.Parse(ttf)
		if err != nil {
			return nil, err
		}
		face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: sizePx, DPI: 72, Hinting: font.HintingFull})
		if err != nil {
			return nil, err
		}
		r.faces[i] = face
	}
	m := r.faces[0].Metrics()
	adv, _ := r.faces[0].GlyphAdvance('M')
	r.cellW = int(math.Ceil(float64(adv) / 64))
	r.ascent = m.Ascent.Ceil()
	r.cellH = r.ascent + m.Descent.Ceil()
	if r.cellW <= 0 || r.cellH <= 0 {
		r.cellW, r.cellH = 1, 1
	}
	return r, nil
}

// CellSize is the size of one cell in pixels.
func (r *Renderer) CellSize() (w, h int) { return r.cellW, r.cellH }

// Grid is how many columns and rows fit in w by h pixels (at least 1 by 1).
func (r *Renderer) Grid(w, h int) (cols, rows int) {
	return max(w/r.cellW, 1), max(h/r.cellH, 1)
}

// glyph returns the coverage mask of a character: cells*cellW by cellH
// bytes, or nil for a character that draws nothing.
func (r *Renderer) glyph(ch rune, face uint8, cells uint8) []uint8 {
	if ch == 0 || ch == ' ' {
		return nil
	}
	key := glyphKey{ch, face, cells}
	if m, ok := r.cache[key]; ok {
		return m
	}
	w := r.cellW * int(cells)
	img := image.NewAlpha(image.Rect(0, 0, w, r.cellH))
	f := r.faces[face]
	if _, ok := f.GlyphAdvance(ch); ok {
		d := font.Drawer{Dst: img, Src: image.Opaque, Face: f, Dot: fixed.P(0, r.ascent)}
		d.DrawString(string(ch))
	} else {
		// No glyph in the font: a hollow box, so the character is visibly
		// there and takes its cells.
		box := image.Rect(1, 2, w-1, r.cellH-2)
		draw.Draw(img, box, image.Opaque, image.Point{}, draw.Src)
		draw.Draw(img, box.Inset(1), image.Transparent, image.Point{}, draw.Src)
	}
	r.cache[key] = img.Pix
	return img.Pix
}

func rgb(c color.RGB) uint32 { return uint32(c.R)<<16 | uint32(c.G)<<8 | uint32(c.B) }

// blend mixes fg over bg with coverage a (0..255).
func blend(bg, fg uint32, a uint32) uint32 {
	if a == 255 {
		return fg
	}
	ia := 255 - a
	rb := ((fg&0xFF00FF)*a + (bg&0xFF00FF)*ia + 0x800080) >> 8 & 0xFF00FF
	g := ((fg&0x00FF00)*a + (bg&0x00FF00)*ia + 0x008000) >> 8 & 0x00FF00
	return rb | g
}

type target struct {
	pix    []uint32
	stride int
	w, h   int
}

func (t *target) fill(x0, y0, x1, y1 int, c uint32) {
	x0, y0, x1, y1 = max(x0, 0), max(y0, 0), min(x1, t.w), min(y1, t.h)
	for y := y0; y < y1; y++ {
		row := t.pix[y*t.stride+x0 : y*t.stride+x1]
		for i := range row {
			row[i] = c
		}
	}
}

func (t *target) mask(x0, y0, mw, mh int, m []uint8, fg uint32) {
	for my := 0; my < mh; my++ {
		y := y0 + my
		if y < 0 || y >= t.h {
			continue
		}
		for mx := 0; mx < mw; mx++ {
			a := uint32(m[my*mw+mx])
			x := x0 + mx
			if a == 0 || x < 0 || x >= t.w {
				continue
			}
			p := &t.pix[y*t.stride+x]
			*p = blend(*p, fg, a)
		}
	}
}

// Draw paints the whole render state into pix: w by h pixels, stride
// pixels per row. Pixels outside the grid get the background colour.
func (r *Renderer) Draw(rs *vt.RenderState, pix []uint32, stride, w, h int) {
	t := &target{pix, stride, w, h}
	defBg, defFg := rgb(rs.Colors.Background), rgb(rs.Colors.Foreground)
	t.fill(0, 0, w, h, defBg)

	for y := range rs.RowData {
		row := &rs.RowData[y]
		py := y * r.cellH
		if py >= h {
			break
		}
		for x := range row.Cells {
			cell := &row.Cells[x]
			wide := cell.Raw.Wide()
			if wide == page.CellWideSpacerTail {
				continue // drawn with the wide character before it
			}
			cells := 1
			if wide == page.CellWideWide {
				cells = 2
			}
			px := x * r.cellW

			var style vt.Style
			if cell.Raw.HasStyling() {
				style = cell.Style
			}
			fg := rgb(style.Fg(vt.StyleFg{Default: rs.Colors.Foreground, Palette: &rs.Colors.Palette}))
			bg := defBg
			if c := style.Bg(&cell.Raw, &rs.Colors.Palette); c.Valid {
				bg = rgb(c.Value)
			}
			if style.Flags.Inverse {
				fg, bg = bg, fg
			}
			if sel := row.Selection; sel.Valid && x >= int(sel.Value[0]) && x <= int(sel.Value[1]) {
				fg, bg = bg, fg
			}
			if style.Flags.Faint {
				fg = blend(bg, fg, 128)
			}
			if bg != defBg {
				t.fill(px, py, px+cells*r.cellW, py+r.cellH, bg)
			}
			if style.Flags.Invisible || !cell.Raw.HasText() {
				continue
			}
			face := uint8(0)
			if style.Flags.Bold {
				face |= 1
			}
			if style.Flags.Italic {
				face |= 2
			}
			if m := r.glyph(rune(cell.Raw.Codepoint()), face, uint8(cells)); m != nil {
				t.mask(px, py, cells*r.cellW, r.cellH, m, fg)
			}
			// Combining marks of the cell's grapheme go on top.
			for _, cp := range cell.Grapheme {
				if _, ok := r.faces[face].GlyphAdvance(rune(cp)); !ok {
					continue // no glyph for the mark: leave the base character readable
				}
				if m := r.glyph(rune(cp), face, uint8(cells)); m != nil {
					t.mask(px, py, cells*r.cellW, r.cellH, m, fg)
				}
			}
			thick := max(r.cellH/14, 1)
			if style.Flags.Underline != sgr.AttributeUnderlineNone {
				uc := fg
				if c := style.UnderlineColor(&rs.Colors.Palette); c.Valid {
					uc = rgb(c.Value)
				}
				uy := py + r.ascent + thick
				t.fill(px, uy, px+cells*r.cellW, uy+thick, uc)
				if style.Flags.Underline == sgr.AttributeUnderlineDouble {
					t.fill(px, uy+2*thick, px+cells*r.cellW, uy+3*thick, uc)
				}
			}
			if style.Flags.Strikethrough {
				sy := py + r.ascent*2/3
				t.fill(px, sy, px+cells*r.cellW, sy+thick, fg)
			}
			if style.Flags.Overline {
				t.fill(px, py, px+cells*r.cellW, py+thick, fg)
			}
		}
	}
	r.cursor(rs, t, defBg, defFg)
}

// cursor draws the cursor when it is visible and inside the viewport.
func (r *Renderer) cursor(rs *vt.RenderState, t *target, defBg, defFg uint32) {
	c := &rs.Cursor
	if !c.Visible || !c.Viewport.Valid {
		return
	}
	color := defFg
	if rs.Colors.Cursor.Valid {
		color = rgb(rs.Colors.Cursor.Value)
	}
	x := int(c.Viewport.Value.X)
	if c.Viewport.Value.WideTail && x > 0 {
		x--
	}
	cells := 1
	if c.Cell.Wide() == page.CellWideWide {
		cells = 2
	}
	px, py := x*r.cellW, int(c.Viewport.Value.Y)*r.cellH
	cw := cells * r.cellW
	thick := max(r.cellW/6, 2)
	switch c.VisualStyle {
	case vt.CursorStyleBar:
		t.fill(px, py, px+thick, py+r.cellH, color)
	case vt.CursorStyleUnderline:
		t.fill(px, py+r.cellH-thick, px+cw, py+r.cellH, color)
	case vt.CursorStyleBlockHollow:
		t.fill(px, py, px+cw, py+1, color)
		t.fill(px, py+r.cellH-1, px+cw, py+r.cellH, color)
		t.fill(px, py, px+1, py+r.cellH, color)
		t.fill(px+cw-1, py, px+cw, py+r.cellH, color)
	default: // block: the cell in the cursor colour, its character in the background colour
		t.fill(px, py, px+cw, py+r.cellH, color)
		if c.Cell.HasText() {
			face := uint8(0)
			if c.Style.Flags.Bold {
				face |= 1
			}
			if c.Style.Flags.Italic {
				face |= 2
			}
			if m := r.glyph(rune(c.Cell.Codepoint()), face, uint8(cells)); m != nil {
				t.mask(px, py, cw, r.cellH, m, defBg)
			}
		}
	}
}
