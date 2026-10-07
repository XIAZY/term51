package render

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"github.com/XIAZY/libghostty-vt-go/vt"
	vtcolor "github.com/XIAZY/libghostty-vt-go/vt/color"
)

// screen feeds input to a new terminal and returns its render state.
func screen(t *testing.T, cols, rows uint16, input string) *vt.RenderState {
	t.Helper()
	// The library's default colours are unset; an app gives its theme's.
	colors := vt.TerminalColorsDefault()
	colors.Foreground = vtcolor.NewDynamicRGB(vtcolor.RGB{R: 0xD0, G: 0xD0, B: 0xD0})
	colors.Background = vtcolor.NewDynamicRGB(vtcolor.RGB{R: 0x10, G: 0x14, B: 0x18})
	term, err := vt.NewTerminal(vt.TerminalOptions{Cols: cols, Rows: rows, MaxScrollback: 1 << 20, Colors: colors, DefaultModes: vt.NewModePacked()})
	if err != nil {
		t.Fatal(err)
	}
	s := term.VtStream()
	if err := s.NextSlice([]byte(input)); err != nil {
		t.Fatal(err)
	}
	rs := vt.RenderStateEmpty()
	if err := rs.Update(&term); err != nil {
		t.Fatal(err)
	}
	return &rs
}

type canvas struct {
	pix  []uint32
	w, h int
}

func paint(t *testing.T, r *Renderer, rs *vt.RenderState) canvas {
	cw, ch := r.CellSize()
	c := canvas{w: int(rs.Cols)*cw + 7, h: int(rs.Rows)*ch + 5}
	c.pix = make([]uint32, c.w*c.h)
	r.Draw(rs, c.pix, c.w, c.w, c.h)
	return c
}

// count returns how many pixels of the cell at column x, row y have the
// colour col.
func (c canvas) count(r *Renderer, x, y int, col uint32) int {
	cw, ch := r.CellSize()
	n := 0
	for py := y * ch; py < (y+1)*ch; py++ {
		for px := x * cw; px < (x+1)*cw; px++ {
			if c.pix[py*c.w+px] == col {
				n++
			}
		}
	}
	return n
}

func (c canvas) save(t *testing.T, name string) {
	dir := os.Getenv("TERM51_RENDER_OUT")
	if dir == "" {
		return
	}
	img := image.NewRGBA(image.Rect(0, 0, c.w, c.h))
	for i, p := range c.pix {
		img.Set(i%c.w, i/c.w, color.RGBA{uint8(p >> 16), uint8(p >> 8), uint8(p), 255})
	}
	f, err := os.Create(dir + "/" + name + ".png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestTextColoursAndCursor(t *testing.T) {
	r, err := New(28)
	if err != nil {
		t.Fatal(err)
	}
	cw, ch := r.CellSize()
	if cw < 10 || ch < 20 {
		t.Fatalf("cell size %dx%d", cw, ch)
	}
	// Row 0: "A", a space, "B" in truecolour red on blue. The cursor ends
	// on row 1 after "x".
	rs := screen(t, 10, 3, "A \x1b[38;2;255;0;0;48;2;0;0;255mB\x1b[0m\r\nx")
	c := paint(t, r, rs)
	c.save(t, "text")

	bg, fg := rgb(rs.Colors.Background), rgb(rs.Colors.Foreground)
	if bg != 0x101418 || fg != 0xD0D0D0 {
		t.Fatalf("default colours %06x on %06x", fg, bg)
	}
	cell := cw * ch
	if n := c.count(r, 0, 0, fg); n < cell/20 {
		t.Errorf("cell A: %d foreground pixels", n)
	}
	if n := c.count(r, 1, 0, bg); n != cell {
		t.Errorf("empty cell: %d of %d background pixels", n, cell)
	}
	if n := c.count(r, 2, 0, 0xFF0000); n < cell/20 {
		t.Errorf("cell B: %d red pixels", n)
	}
	if n := c.count(r, 2, 0, 0x0000FF); n < cell/2 {
		t.Errorf("cell B: %d blue pixels", n)
	}
	if n := c.count(r, 2, 0, bg); n != 0 {
		t.Errorf("cell B: %d default-background pixels", n)
	}
	// The block cursor fills its cell with the cursor colour.
	if n := c.count(r, 1, 1, fg); n != cell {
		t.Errorf("cursor cell: %d of %d pixels", n, cell)
	}
	// Outside the grid: background.
	if p := c.pix[len(c.pix)-1]; p != bg {
		t.Errorf("margin pixel %06x", p)
	}
}

func TestAttributesWideAndInverse(t *testing.T) {
	r, err := New(28)
	if err != nil {
		t.Fatal(err)
	}
	cw, ch := r.CellSize()
	rs := screen(t, 20, 4, "\x1b[1mbold\x1b[0m \x1b[3mital\x1b[0m \x1b[4munder\x1b[0m\r\n"+
		"\x1b[7minv\x1b[0m \x1b[9mstrike\x1b[0m \x1b[2mfaint\x1b[0m\r\n"+
		"\xe4\xb8\xad\xe6\x96\x87 e\xcc\x81 \x1b[32mgreen\x1b[0m \x1b[?25l")
	c := paint(t, r, rs)
	c.save(t, "attributes")

	bg, fg := rgb(rs.Colors.Background), rgb(rs.Colors.Foreground)
	cell := cw * ch
	// Inverse: the cell is mostly foreground colour.
	if n := c.count(r, 0, 1, fg); n < cell/2 {
		t.Errorf("inverse cell: %d foreground pixels of %d", n, cell)
	}
	// A wide character takes two cells (the fallback box has ink in both).
	if a, b := cell-c.count(r, 0, 2, bg), cell-c.count(r, 1, 2, bg); a == 0 || b == 0 {
		t.Errorf("wide character: ink %d and %d in its two cells", a, b)
	}
	// Bold has more ink than regular for the same letter.
	plain := paint(t, r, screen(t, 4, 1, "b\x1b[?25l"))
	bold := paint(t, r, screen(t, 4, 1, "\x1b[1mb\x1b[?25l"))
	if p, b := cell-plain.count(r, 0, 0, bg), cell-bold.count(r, 0, 0, bg); b <= p {
		t.Errorf("bold ink %d, regular ink %d", b, p)
	}
	// The cursor is hidden: its cell is all background.
	hidden := paint(t, r, screen(t, 4, 1, "\x1b[?25l"))
	if n := hidden.count(r, 0, 0, bg); n != cell {
		t.Errorf("hidden cursor: %d of %d background pixels", n, cell)
	}
}

func TestGrid(t *testing.T) {
	r, err := New(28)
	if err != nil {
		t.Fatal(err)
	}
	cw, ch := r.CellSize()
	if cols, rows := r.Grid(cw*80+3, ch*24+1); cols != 80 || rows != 24 {
		t.Errorf("grid %dx%d", cols, rows)
	}
	if cols, rows := r.Grid(1, 1); cols != 1 || rows != 1 {
		t.Errorf("tiny grid %dx%d", cols, rows)
	}
}

func BenchmarkDraw(b *testing.B) {
	r, _ := New(28)
	term, _ := vt.NewTerminal(vt.TerminalOptions{Cols: 80, Rows: 40, Colors: vt.TerminalColorsDefault(), DefaultModes: vt.NewModePacked()})
	s := term.VtStream()
	for i := 0; i < 40; i++ {
		s.NextSlice([]byte("\x1b[32mdrwxr-xr-x\x1b[0m  2 user group  4096 Oct  7 03:00 \x1b[1;34msome-directory\x1b[0m and more text\r\n"))
	}
	rs := vt.RenderStateEmpty()
	rs.Update(&term)
	cw, ch := r.CellSize()
	w, h := 80*cw, 40*ch
	pix := make([]uint32, w*h)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Draw(&rs, pix, w, w, h)
	}
}
