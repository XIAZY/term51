package cells

import (
	"encoding/binary"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonobolditalic"
	"golang.org/x/image/font/gofont/gomonoitalic"
	"golang.org/x/image/font/opentype"

	"github.com/XIAZY/libghostty-vt-go/vt"
	"github.com/XIAZY/libghostty-vt-go/vt/color"
)

// wideFont is a font with Chinese and Japanese characters on the development
// machine; the phone's is another file of the same kind.
const wideFont = "/usr/share/fonts/opentype/ipafont-gothic/ipag.ttf"

var (
	fg = color.RGB{R: 0xD0, G: 0xD0, B: 0xD0}
	bg = color.RGB{R: 0x10, G: 0x14, B: 0x18}
)

func parse(t *testing.T, ttf []byte) *opentype.Font {
	t.Helper()
	f, err := opentype.Parse(ttf)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func read(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func testFonts(t *testing.T) Fonts {
	t.Helper()
	return Fonts{
		Faces:  [4]*opentype.Font{parse(t, gomono.TTF), parse(t, gomonobold.TTF), parse(t, gomonoitalic.TTF), parse(t, gomonobolditalic.TTF)},
		Others: []*opentype.Font{parse(t, read(t, wideFont))},
	}
}

func newSheet(t *testing.T, sizePx float64) *Sheet {
	t.Helper()
	s, err := NewSheet(testFonts(t), sizePx)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

type term struct {
	t  *vt.Terminal
	s  vt.ReadonlyStream
	rs vt.RenderState
}

func newTerm(t *testing.T, cols, rows uint16) *term {
	t.Helper()
	colors := vt.TerminalColorsDefault()
	colors.Foreground = color.NewDynamicRGB(fg)
	colors.Background = color.NewDynamicRGB(bg)
	tm, err := vt.NewTerminal(vt.TerminalOptions{Cols: cols, Rows: rows, Colors: colors, DefaultModes: vt.NewModePacked()})
	if err != nil {
		t.Fatal(err)
	}
	x := &term{t: &tm, rs: vt.RenderStateEmpty()}
	x.s = x.t.VtStream()
	return x
}

func (x *term) feed(t *testing.T, in string) *vt.RenderState {
	t.Helper()
	if err := x.s.NextSlice([]byte(in)); err != nil {
		t.Fatal(err)
	}
	if err := x.rs.Update(x.t); err != nil {
		t.Fatal(err)
	}
	return &x.rs
}

// record is one cell's record, decoded.
type record struct {
	slot, flags  uint16
	fg, bg, line color.RGB
}

func (tb *Table) at(t *testing.T, x, y int) record {
	t.Helper()
	b := tb.Data[(y*tb.Cols+x)*RecordSize:][:RecordSize]
	rgb := func(b []byte) color.RGB {
		if b[3] != 255 {
			t.Errorf("cell %d,%d: a colour's fourth byte is %d, want 255", x, y, b[3])
		}
		return color.RGB{R: b[0], G: b[1], B: b[2]}
	}
	return record{
		slot: binary.LittleEndian.Uint16(b), flags: binary.LittleEndian.Uint16(b[2:]),
		fg: rgb(b[4:]), bg: rgb(b[8:]), line: rgb(b[12:]),
	}
}

// ink is the sum of the coverage in a slot.
func (s *Sheet) ink(n uint16) int {
	sum := 0
	r := s.rect(int(n))
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			sum += int(s.Pix[y*SheetSize+x])
		}
	}
	return sum
}

func TestSheetDrawsACharacterOnceIntoItsSlot(t *testing.T) {
	s := newSheet(t, 28)
	w, h := s.CellSize()
	if w < 10 || w > 28 || h < 28 || h > 48 {
		t.Fatalf("cell size %d by %d for a 28 pixel font", w, h)
	}
	if s.PerRow != SheetSize/w {
		t.Fatalf("PerRow = %d, want %d", s.PerRow, SheetSize/w)
	}
	if d := s.TakeDirty(); !d.Empty() {
		t.Fatalf("a new sheet is dirty: %v", d)
	}

	a := s.slot([]rune{'A'}, 0, false)
	if a != 1 {
		t.Fatalf("first slot = %d, want 1 (0 means no character)", a)
	}
	if s.ink(a) == 0 {
		t.Fatal("'A' left its slot empty")
	}
	if d := s.TakeDirty(); d != s.rect(1) {
		t.Fatalf("dirty = %v, want the slot %v", d, s.rect(1))
	}
	total := 0
	for _, p := range s.Pix {
		total += int(p)
	}
	if total != s.ink(a) {
		t.Fatal("'A' was drawn outside its slot")
	}

	if again := s.slot([]rune{'A'}, 0, false); again != a {
		t.Fatalf("second use of 'A': slot %d, want %d", again, a)
	}
	if d := s.TakeDirty(); !d.Empty() {
		t.Fatalf("a character already in the sheet made it dirty: %v", d)
	}

	bold := s.slot([]rune{'A'}, 1, false)
	if bold == a {
		t.Fatal("bold 'A' shares the regular one's slot")
	}
	if s.ink(bold) <= s.ink(a) {
		t.Errorf("bold 'A' has no more ink (%d) than the regular one (%d)", s.ink(bold), s.ink(a))
	}
	if s.Generation() != 0 {
		t.Fatalf("generation = %d", s.Generation())
	}
}

func TestSheetWideCharacterTakesTwoSlots(t *testing.T) {
	s := newSheet(t, 28)
	n := s.slot([]rune{'中'}, 0, true)
	if s.ink(n) == 0 || s.ink(n+1) == 0 {
		t.Fatalf("ink in the two halves of a wide character: %d and %d", s.ink(n), s.ink(n+1))
	}
	if next := s.slot([]rune{'B'}, 0, false); next != n+2 {
		t.Fatalf("slot after a wide character = %d, want %d", next, n+2)
	}
	// The vertical stroke of 中 is in the middle of the two cells: the
	// columns next to the seam hold more ink than the outer ones.
	w, h := s.CellSize()
	col := func(slot uint16, x int) int {
		r := s.rect(int(slot))
		sum := 0
		for y := 0; y < h; y++ {
			sum += int(s.Pix[(r.Min.Y+y)*SheetSize+r.Min.X+x])
		}
		return sum
	}
	seam := col(n, w-1) + col(n, w-2) + col(n+1, 0) + col(n+1, 1)
	edge := col(n, 0) + col(n, 1) + col(n+1, w-1) + col(n+1, w-2)
	if seam <= edge {
		t.Errorf("中 is not centred in its two cells: ink at the seam %d, at the edges %d", seam, edge)
	}
}

func TestSheetCombiningMarkAndMissingCharacter(t *testing.T) {
	s := newSheet(t, 28)
	plain := s.slot([]rune{'e'}, 0, false)
	marked := s.slot([]rune{'e', 0x0301}, 0, false)
	if marked == plain {
		t.Fatal("'e' with an accent shares the slot of 'e'")
	}
	if s.ink(marked) <= s.ink(plain) {
		t.Errorf("the accent added no ink: %d against %d", s.ink(marked), s.ink(plain))
	}

	// A private-use character no font has: a hollow box.
	box := s.slot([]rune{0xF8FF0}, 0, false)
	r := s.rect(int(box))
	w, h := s.CellSize()
	if s.Pix[(r.Min.Y+2)*SheetSize+r.Min.X+1] != 255 {
		t.Error("the box of a missing character has no corner")
	}
	if s.Pix[(r.Min.Y+h/2)*SheetSize+r.Min.X+w/2] != 0 {
		t.Error("the box of a missing character is filled")
	}
}

func TestSheetStartsAgainWhenFull(t *testing.T) {
	s := newSheet(t, 400) // a handful of slots
	total := s.PerRow * (SheetSize / func() int { _, h := s.CellSize(); return h }())
	if total < 3 || total > 64 {
		t.Fatalf("%d slots at this size; the test wants a few", total)
	}
	for i := 0; i < total-1; i++ {
		if n := s.slot([]rune{rune('a' + i)}, 0, false); int(n) != i+1 {
			t.Fatalf("character %d: slot %d", i, n)
		}
	}
	if s.Generation() != 0 {
		t.Fatal("the sheet started again before it was full")
	}
	s.TakeDirty()
	n := s.slot([]rune{'Z'}, 0, false)
	if n != 1 || s.Generation() != 1 {
		t.Fatalf("after filling up: slot %d, generation %d; want 1 and 1", n, s.Generation())
	}
	if d := s.TakeDirty(); d != image.Rect(0, 0, SheetSize, SheetSize) {
		t.Fatalf("dirty after starting again = %v, want the whole sheet", d)
	}
	if s.ink(2) != 0 {
		t.Fatal("an old character is still in the sheet")
	}
}

func TestTableRecords(t *testing.T) {
	s := newSheet(t, 28)
	x := newTerm(t, 8, 2)
	var tb Table
	first, end := tb.Update(x.feed(t, "a \x1b[1;4;38;2;255;0;0;48;2;0;0;255;58;2;0;255;0mB\x1b[0;7;9mc\x1b[0;8md\x1b[0m\x1b[?25l"), s)
	if tb.Cols != 8 || tb.Rows != 2 || len(tb.Data) != 8*2*RecordSize {
		t.Fatalf("table %d by %d, %d bytes", tb.Cols, tb.Rows, len(tb.Data))
	}
	if first != 0 || end != len(tb.Data) {
		t.Fatalf("first update changed [%d, %d), want everything", first, end)
	}

	a := tb.at(t, 0, 0)
	if a.slot == 0 || a.flags != 0 || a.fg != fg || a.bg != bg || a.line != fg {
		t.Errorf("'a': %+v", a)
	}
	if sp := tb.at(t, 1, 0); sp.slot != 0 || sp.bg != bg {
		t.Errorf("the space: %+v", sp)
	}
	b := tb.at(t, 2, 0)
	red, blue, green := color.RGB{R: 255}, color.RGB{B: 255}, color.RGB{G: 255}
	if b.slot == 0 || b.slot == a.slot || b.flags != FlagUnderline || b.fg != red || b.bg != blue || b.line != green {
		t.Errorf("bold underlined 'B': %+v", b)
	}
	if b.slot != s.slot([]rune{'B'}, 1, false) {
		t.Error("bold 'B' is not drawn from the bold font")
	}
	c := tb.at(t, 3, 0)
	if c.slot == 0 || c.flags != FlagStrikethrough || c.fg != bg || c.bg != fg {
		t.Errorf("reverse struck 'c': %+v", c)
	}
	if d := tb.at(t, 4, 0); d.slot != 0 {
		t.Errorf("invisible 'd' has a character: %+v", d)
	}
	if e := tb.at(t, 7, 1); e.slot != 0 || e.flags != 0 || e.fg != fg || e.bg != bg {
		t.Errorf("an empty cell: %+v", e)
	}
}

func TestTableWideCharacter(t *testing.T) {
	s := newSheet(t, 28)
	x := newTerm(t, 6, 1)
	var tb Table
	tb.Update(x.feed(t, "\x1b[41m中\x1b[0mx\x1b[?25l"), s)
	head, tail, after := tb.at(t, 0, 0), tb.at(t, 1, 0), tb.at(t, 2, 0)
	if head.slot == 0 || tail.slot != head.slot+1 {
		t.Fatalf("slots of the two halves: %d and %d", head.slot, tail.slot)
	}
	if tail.bg != head.bg || tail.fg != head.fg || head.bg == bg {
		t.Errorf("the halves differ in look: %+v and %+v", head, tail)
	}
	if after.slot == 0 || after.slot == tail.slot || after.bg != bg {
		t.Errorf("the character after it: %+v", after)
	}
}

func TestTableCursor(t *testing.T) {
	s := newSheet(t, 28)
	cursor := fg // no cursor colour set: the text's default colour

	for _, tc := range []struct {
		name, shape string
		check       func(t *testing.T, r record)
	}{
		{"block", "\x1b[2 q", func(t *testing.T, r record) {
			// The text's underline stays, in the colour the character has.
			if r.slot == 0 || r.flags != FlagUnderline || r.fg != bg || r.bg != cursor || r.line != bg {
				t.Errorf("%+v", r)
			}
		}},
		{"bar", "\x1b[6 q", func(t *testing.T, r record) {
			// One lines' colour per cell: the underline takes the cursor's.
			if r.flags != FlagUnderline|FlagCursorBar || r.line != cursor || r.fg != (color.RGB{R: 255}) || r.bg != bg {
				t.Errorf("%+v", r)
			}
		}},
		{"underline", "\x1b[4 q", func(t *testing.T, r record) {
			if r.flags != FlagCursorUnderline || r.line != cursor {
				t.Errorf("%+v", r)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := newTerm(t, 4, 1)
			var tb Table
			// A red underlined 'a' with the cursor moved back onto it.
			tb.Update(x.feed(t, tc.shape+"\x1b[4;38;2;255;0;0ma\x1b[0mb\x1b[1G"), s)
			tc.check(t, tb.at(t, 0, 0))
			if b := tb.at(t, 1, 0); b.flags != 0 || b.fg != fg || b.bg != bg {
				t.Errorf("the cell beside the cursor: %+v", b)
			}

			// Hidden, the cell is the text's own again.
			tb.Update(x.feed(t, "\x1b[?25l"), s)
			if r := tb.at(t, 0, 0); r.flags != FlagUnderline || r.fg != (color.RGB{R: 255}) || r.bg != bg || r.line != r.fg {
				t.Errorf("with the cursor hidden: %+v", r)
			}
		})
	}

	t.Run("on a wide character", func(t *testing.T) {
		x := newTerm(t, 4, 1)
		var tb Table
		tb.Update(x.feed(t, "\x1b[2 q中\x1b[1G"), s)
		if h, tl := tb.at(t, 0, 0), tb.at(t, 1, 0); h.bg != cursor || tl.bg != cursor || tl.slot != h.slot+1 {
			t.Errorf("block: %+v and %+v", h, tl)
		}
		tb.Update(x.feed(t, "\x1b[6 q"), s)
		if h, tl := tb.at(t, 0, 0), tb.at(t, 1, 0); h.flags != FlagCursorBar || tl.flags != 0 {
			t.Errorf("bar: flags %d and %d, want it on the left half only", h.flags, tl.flags)
		}
	})
}

func TestTableSelection(t *testing.T) {
	s := newSheet(t, 28)
	x := newTerm(t, 6, 1)
	var tb Table
	tb.Update(x.feed(t, "ab\x1b[7mc\x1b[0md\x1b[?25l"), s)

	pin := func(col uint16) vt.Pin {
		p := x.t.Screens.Active.Pages.Pin(vt.Point{Tag: vt.PointTagViewport, Viewport: vt.Coordinate{X: col}})
		if !p.Valid {
			t.Fatalf("no pin for column %d", col)
		}
		return p.Value
	}
	if err := x.t.Screens.Active.Select(vt.Some(vt.NewSelection(pin(1), pin(2), false))); err != nil {
		t.Fatal(err)
	}
	first, end := tb.Update(x.feed(t, ""), s)
	if first != 1*RecordSize || end != 3*RecordSize {
		t.Errorf("selecting two cells changed [%d, %d), want [%d, %d)", first, end, RecordSize, 3*RecordSize)
	}
	if a := tb.at(t, 0, 0); a.fg != fg || a.bg != bg {
		t.Errorf("outside the selection: %+v", a)
	}
	if b := tb.at(t, 1, 0); b.fg != bg || b.bg != fg {
		t.Errorf("selected: %+v", b)
	}
	// Reverse video and selected: still reversed, with a tinted background.
	if c := tb.at(t, 2, 0); c.fg != bg || c.bg != mix(fg, selectionTint) {
		t.Errorf("selected reverse video: %+v", c)
	}
}

func TestTableReportsOnlyWhatChanged(t *testing.T) {
	s := newSheet(t, 28)
	x := newTerm(t, 10, 4)
	var tb Table
	tb.Update(x.feed(t, "\x1b[?25lhello"), s)

	if first, end := tb.Update(x.feed(t, ""), s); first != end {
		t.Fatalf("nothing changed, but [%d, %d) reported", first, end)
	}

	// One character on row 2, one on row 3: from the first to the last.
	first, end := tb.Update(x.feed(t, "\x1b[3;4Hx\x1b[4;2Hy"), s)
	if want0, want1 := (2*10+3)*RecordSize, (3*10+1+1)*RecordSize; first != want0 || end != want1 {
		t.Fatalf("changed [%d, %d), want [%d, %d)", first, end, want0, want1)
	}

	// A new size: a new table, all of it.
	if err := x.t.Resize(12, 3); err != nil {
		t.Fatal(err)
	}
	first, end = tb.Update(x.feed(t, ""), s)
	if tb.Cols != 12 || tb.Rows != 3 || first != 0 || end != 12*3*RecordSize {
		t.Fatalf("after a resize: %d by %d, changed [%d, %d)", tb.Cols, tb.Rows, first, end)
	}
}

func TestTableAfterTheSheetStartedAgain(t *testing.T) {
	s := newSheet(t, 400)
	total := s.PerRow * (SheetSize / func() int { _, h := s.CellSize(); return h }())
	x := newTerm(t, 4, 1)
	var tb Table
	tb.Update(x.feed(t, "\x1b[?25lab"), s)
	a, b := tb.at(t, 0, 0).slot, tb.at(t, 1, 0).slot
	if a != 1 || b != 2 {
		t.Fatalf("slots %d and %d", a, b)
	}

	// Fill the sheet from elsewhere (another tab's table would), so that the
	// next new character starts it again.
	for i := 0; s.Generation() == 0; i++ {
		if i > total {
			t.Fatal("the sheet never filled up")
		}
		s.slot([]rune{rune('0' + i)}, 0, false)
	}
	first, end := tb.Update(x.feed(t, ""), s)
	if first != 0 || end != len(tb.Data) {
		t.Fatalf("after the sheet started again, changed [%d, %d), want everything", first, end)
	}
	a, b = tb.at(t, 0, 0).slot, tb.at(t, 1, 0).slot
	if s.ink(a) == 0 || s.ink(b) == 0 || a == b {
		t.Fatalf("slots %d and %d after starting again", a, b)
	}
	if a != s.slot([]rune{'a'}, 0, false) || b != s.slot([]rune{'b'}, 0, false) {
		t.Fatal("the table's slots are not the sheet's")
	}
}

// compose paints the table from the sheet the way the GPU program does (the
// lines aside), as a reference and to look at: TERM51_CELLS_OUT=dir writes
// the pictures there.
func compose(tb *Table, s *Sheet) *image.RGBA {
	w, h := s.CellSize()
	img := image.NewRGBA(image.Rect(0, 0, tb.Cols*w, tb.Rows*h))
	for i := 0; i < tb.Cols*tb.Rows; i++ {
		rec := tb.Data[i*RecordSize:][:RecordSize]
		at := s.rect(int(binary.LittleEndian.Uint16(rec)))
		x0, y0 := i%tb.Cols*w, i/tb.Cols*h
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				a := int(s.Pix[(at.Min.Y+y)*SheetSize+at.Min.X+x])
				p := img.Pix[img.PixOffset(x0+x, y0+y):]
				for c := 0; c < 3; c++ {
					p[c] = uint8((int(rec[4+c])*a + int(rec[8+c])*(255-a)) / 255)
				}
				p[3] = 255
			}
		}
	}
	return img
}

func TestPicture(t *testing.T) {
	s := newSheet(t, 28)
	x := newTerm(t, 64, 6)
	var tb Table
	tb.Update(x.feed(t, "\x1b[2 q$ ls \x1b[1mbold\x1b[0m \x1b[3mitalic\x1b[0m \x1b[1;3mboth\x1b[0m\r\n"+
		"\x1b[31mred \x1b[32mgreen \x1b[34;47mblue on white\x1b[0m \x1b[7mreverse\x1b[0m\r\n"+
		"中文 wide 日本語 é \U000F8FF0\r\n"+
		"prompt> \x1b[5;1H┌─┬─┐ ╔═╦═╗ ╒═╤═╕ ╓─╥─╖ ┏━┳━┓ ╭─╮ ▁▂▃▄▅▆▇█ ░▒▓\r\n"+
		"├─┼─┤ ╠═╬═╣ ╞═╪═╡ ╟─╫─╢ ┣━╋━┫ ╰─╯ ▏▎▍▌▋▊▉█ ▖▗▘▙▚▛▜▝▞▟\x1b[4;9H"), s)
	img := compose(&tb, s)

	// The block cursor after the prompt is a cell in the text's colour.
	w, h := s.CellSize()
	if got := img.RGBAAt(8*w+w/2, 3*h+h/2); got.R != fg.R || got.G != fg.G || got.B != fg.B {
		t.Errorf("the cursor's cell is %v", got)
	}
	// A slot never used is empty, so an empty cell is pure background.
	if got := img.RGBAAt(60*w+w/2, 5*h+h/2); got.R != bg.R || got.G != bg.G || got.B != bg.B {
		t.Errorf("an empty cell is %v", got)
	}

	dir := os.Getenv("TERM51_CELLS_OUT")
	if dir == "" {
		return
	}
	for name, im := range map[string]image.Image{
		"screen.png": img,
		"sheet.png":  &image.Gray{Pix: s.Pix, Stride: SheetSize, Rect: image.Rect(0, 0, SheetSize, 4*h)},
	} {
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, im); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

// TestPhoneFonts draws from the phone's own font files, which are not ours to
// keep in the repository: TERM51_PHONE_FONTS=dir names a directory holding
// copies of the phone's DejaVuSansMono*.ttf and MHeiM.GB18030_200910.ttf.
func TestPhoneFonts(t *testing.T) {
	dir := os.Getenv("TERM51_PHONE_FONTS")
	if dir == "" {
		t.Skip("TERM51_PHONE_FONTS is not set")
	}
	var f Fonts
	for i, name := range []string{"DejaVuSansMono.ttf", "DejaVuSansMono-Bold.ttf", "DejaVuSansMono-Oblique.ttf", "DejaVuSansMono-BoldOblique.ttf"} {
		f.Faces[i] = parse(t, read(t, filepath.Join(dir, name)))
	}
	f.Others = []*opentype.Font{parse(t, read(t, filepath.Join(dir, "MHeiM.GB18030_200910.ttf")))}
	s, err := NewSheet(f, 28)
	if err != nil {
		t.Fatal(err)
	}
	for style := uint8(0); style < 4; style++ {
		if n := s.slot([]rune{'A'}, style, false); s.ink(n) == 0 {
			t.Errorf("style %d: 'A' is empty", style)
		}
	}
	for _, r := range "中文日本語한" {
		n := s.slot([]rune{r}, 0, true)
		if s.ink(n) == 0 || s.ink(n+1) == 0 {
			t.Errorf("%c: ink %d and %d", r, s.ink(n), s.ink(n+1))
		}
		t.Logf("%c: found=%v", r, s.face(s.faces[0], r) != nil)
	}
	x := newTerm(t, 40, 3)
	var tb Table
	tb.Update(x.feed(t, "$ ls \x1b[1mbold\x1b[0m \x1b[3mitalic\x1b[0m \x1b[1;3mboth\x1b[0m\r\n中文 wide 日本語 한국어 e\u0301 │┌─┐ ✓ →\r\nprompt> "), s)
	if out := os.Getenv("TERM51_CELLS_OUT"); out != "" {
		file, err := os.Create(filepath.Join(out, "phone.png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(file, compose(&tb, s)); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
