package textgrid

import (
	"strings"
	"testing"

	"github.com/XIAZY/libghostty-vt-go/vt"
	"github.com/XIAZY/libghostty-vt-go/vt/color"
)

type term struct {
	t  *vt.Terminal
	s  vt.ReadonlyStream
	rs vt.RenderState
}

func newTerm(t *testing.T, cols, rows uint16) *term {
	t.Helper()
	colors := vt.TerminalColorsDefault()
	colors.Foreground = color.NewDynamicRGB(color.RGB{R: 0xD0, G: 0xD0, B: 0xD0})
	colors.Background = color.NewDynamicRGB(color.RGB{R: 0x10, G: 0x14, B: 0x18})
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

const nb = " "

func TestRowsAsRichText(t *testing.T) {
	x := newTerm(t, 8, 2)
	var g Grid
	rows := g.Update(x.feed(t, "a <\x1b[1;38;2;255;0;0;48;2;0;0;255mB\x1b[0m\x1b[?25l"))
	if len(rows) != 2 {
		t.Fatalf("first update: %d rows, want 2", len(rows))
	}
	want := `<span style="color:#d0d0d0">a` + nb + `&lt;</span>` +
		`<span style="color:#ff0000;background-color:#0000ff;font-weight:bold">B</span>` +
		`<span style="color:#d0d0d0">` + strings.Repeat(nb, 4) + `</span>`
	if rows[0].Y != 0 || rows[0].HTML != want {
		t.Errorf("row 0:\n got %q\nwant %q", rows[0].HTML, want)
	}
	if want := `<span style="color:#d0d0d0">` + strings.Repeat(nb, 8) + `</span>`; rows[1].HTML != want {
		t.Errorf("row 1:\n got %q\nwant %q", rows[1].HTML, want)
	}
}

func TestOnlyChangedRowsAreReturned(t *testing.T) {
	x := newTerm(t, 8, 3)
	var g Grid
	g.Update(x.feed(t, "one\r\ntwo\r\nthree\x1b[?25l"))
	if rows := g.Update(x.feed(t, "")); len(rows) != 0 {
		t.Errorf("nothing changed, %d rows returned", len(rows))
	}
	rows := g.Update(x.feed(t, "\x1b[2;1HTWO"))
	if len(rows) != 1 || rows[0].Y != 1 || !strings.Contains(rows[0].HTML, "TWO") {
		t.Errorf("after changing row 1: %+v", rows)
	}
}

func TestCursorAndInverse(t *testing.T) {
	x := newTerm(t, 4, 1)
	var g Grid
	// "ab", the cursor on the third cell: that cell is shown inverted.
	rows := g.Update(x.feed(t, "ab"))
	want := `<span style="color:#d0d0d0">ab</span>` +
		`<span style="color:#101418;background-color:#d0d0d0">` + nb + `</span>` +
		`<span style="color:#d0d0d0">` + nb + `</span>`
	if rows[0].HTML != want {
		t.Errorf("cursor:\n got %q\nwant %q", rows[0].HTML, want)
	}
	// Moving the cursor changes the row.
	rows = g.Update(x.feed(t, "\x1b[1;1H"))
	if len(rows) != 1 || !strings.HasPrefix(rows[0].HTML, `<span style="color:#101418;background-color:#d0d0d0">a</span>`) {
		t.Errorf("cursor moved: %+v", rows)
	}
}

func TestWideCharactersAndDecorations(t *testing.T) {
	x := newTerm(t, 8, 1)
	var g Grid
	rows := g.Update(x.feed(t, "\xe4\xb8\xad\x1b[4;9mx\x1b[0m\x1b[3my\x1b[?25l"))
	want := `<span style="color:#d0d0d0">中</span>` +
		`<span style="color:#d0d0d0;text-decoration:underline line-through">x</span>` +
		`<span style="color:#d0d0d0;font-style:italic">y</span>` +
		`<span style="color:#d0d0d0">` + strings.Repeat(nb, 4) + `</span>`
	if rows[0].HTML != want {
		t.Errorf("row:\n got %q\nwant %q", rows[0].HTML, want)
	}
}
