package keys

import (
	"bytes"
	"testing"

	"github.com/XIAZY/term51/libghostty-vt-go/vt"
	"github.com/XIAZY/term51/libghostty-vt-go/vt/input"
)

// term is a terminal to take the encoder's options from.
func term(t *testing.T, setup string) *vt.Terminal {
	t.Helper()
	tm, err := vt.NewTerminal(vt.TerminalOptions{Cols: 80, Rows: 24, Colors: vt.TerminalColorsDefault(), DefaultModes: vt.NewModePacked()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(tm.Deinit)
	if setup != "" {
		s := tm.VtStream()
		if err := s.NextSlice([]byte(setup)); err != nil {
			t.Fatal(err)
		}
	}
	return &tm
}

// send translates r and returns what the library's encoder sends for it.
func send(t *testing.T, tr *Translator, tm *vt.Terminal, r Raw) string {
	t.Helper()
	ev, ok := tr.Translate(r)
	if !ok {
		return ""
	}
	var b bytes.Buffer
	if err := input.EncodeKey(&b, ev, input.KeyEncodeOptionsFromTerminal(tm)); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func down(sym, cap rune, mods int) Raw { return Raw{Down: true, Sym: sym, Cap: cap, Mods: mods} }

func TestWhatEachKeySends(t *testing.T) {
	tm := term(t, "")
	cases := []struct {
		name string
		r    Raw
		want string
	}{
		{"a letter", down('a', 'a', 0), "a"},
		{"Shift and a letter", down('A', 'a', ModShift), "A"},
		{"a digit", down('1', '1', 0), "1"},
		{"space", down(' ', ' ', 0), " "},
		{"a shifted symbol", down('!', '1', ModShift), "!"},
		{"a non-ASCII character", down('é', 0, 0), "é"},

		// The phone's Alt selects the symbol above a letter: no ESC prefix.
		{"Alt for the symbol layer", down('#', 'q', ModAlt), "#"},
		{"Alt for a digit", down('1', 'w', ModAlt), "1"},
		// Alt with a key that produced its own character is a terminal Alt.
		{"Alt held with a plain key", down('x', 'x', ModAlt), "\x1bx"},
		{"Alt and Shift with a letter", down('X', 'x', ModAlt|ModShift), "\x1bX"},

		{"Enter", down(CodeReturn, CodeReturn, 0), "\r"},
		{"Backspace", down(CodeBackspace, CodeBackspace, 0), "\x7f"},
		{"Tab", down(CodeTab, CodeTab, 0), "\t"},
		{"back tab", down(CodeBackTab, CodeTab, 0), "\x1b[Z"},
		{"Shift and Tab", down(CodeTab, CodeTab, ModShift), "\x1b[Z"},
		{"Escape", down(CodeEscape, CodeEscape, 0), "\x1b"},
		{"Delete", down(CodeDelete, CodeDelete, 0), "\x1b[3~"},
		{"Up", down(CodeUp, CodeUp, 0), "\x1b[A"},
		{"Down", down(CodeDown, CodeDown, 0), "\x1b[B"},
		{"Right", down(CodeRight, CodeRight, 0), "\x1b[C"},
		{"Left", down(CodeLeft, CodeLeft, 0), "\x1b[D"},
		{"Home", down(CodeHome, CodeHome, 0), "\x1b[H"},
		{"End", down(CodeEnd, CodeEnd, 0), "\x1b[F"},
		{"Page Up", down(CodePgUp, CodePgUp, 0), "\x1b[5~"},
		{"Page Down", down(CodePgDown, CodePgDown, 0), "\x1b[6~"},
		{"F1", down(CodeF1, CodeF1, 0), "\x1bOP"},
		{"F5", down(CodeF1+4, CodeF1+4, 0), "\x1b[15~"},
		{"F12", down(CodeF12, CodeF12, 0), "\x1b[24~"},
		{"Ctrl and Up", down(CodeUp, CodeUp, ModCtrl), "\x1b[1;5A"},

		// Ctrl from a keyboard that has the key.
		{"Ctrl and c", down('c', 'c', ModCtrl), "\x03"},
		{"Ctrl and c, already applied", down(0x03, 'c', ModCtrl), "\x03"},
		{"Ctrl and ]", down(']', ']', ModCtrl), "\x1d"},
		{"Ctrl and space", down(' ', ' ', ModCtrl), "\x00"},
		// The library, like upstream, does not send a bare ESC for Ctrl+[:
		// it sends the unambiguous form. Esc is on the key bar.
		{"Ctrl and [", down('[', '[', ModCtrl), "\x1b[91;5u"},

		// Cascades reports Enter and Backspace as control characters.
		{"Enter as 13", down(13, 13, 0), "\r"},
		{"Backspace as 8", down(8, 8, 0), "\x7f"},

		// Nothing is sent for these.
		{"a release", Raw{Down: false, Sym: 'a', Cap: 'a'}, ""},
		{"Shift alone", down(CodeLeftShift, CodeLeftShift, ModShift), ""},
		{"Alt alone", down(CodeLeftAlt, CodeLeftAlt, ModAlt), ""},
		{"no key at all", Raw{Down: true}, ""},
	}
	for _, c := range cases {
		var tr Translator
		if got := send(t, &tr, tm, c.r); got != c.want {
			t.Errorf("%s: sent %q, want %q", c.name, got, c.want)
		}
	}
}

func TestTheProgramsModesChangeWhatIsSent(t *testing.T) {
	var tr Translator
	// Application cursor keys (DECCKM).
	if got := send(t, &tr, term(t, "\x1b[?1h"), down(CodeUp, CodeUp, 0)); got != "\x1bOA" {
		t.Errorf("Up in application cursor mode: sent %q", got)
	}
}

func TestTheKeyBarsStickyModifiers(t *testing.T) {
	tm := term(t, "")
	var tr Translator

	tr.StickyCtrl(true)
	if ctrl, alt := tr.Sticky(); !ctrl || alt {
		t.Errorf("Sticky() = %v, %v after arming Ctrl", ctrl, alt)
	}
	// A modifier key by itself does not use the sticky modifier up.
	send(t, &tr, tm, down(CodeLeftShift, CodeLeftShift, ModShift))
	if ctrl, _ := tr.Sticky(); !ctrl {
		t.Error("a Shift press used up the sticky Ctrl")
	}
	if got := send(t, &tr, tm, down('c', 'c', 0)); got != "\x03" {
		t.Errorf("sticky Ctrl then c: sent %q, want %q", got, "\x03")
	}
	if ctrl, _ := tr.Sticky(); ctrl {
		t.Error("the sticky Ctrl is still armed after a key")
	}
	if got := send(t, &tr, tm, down('c', 'c', 0)); got != "c" {
		t.Errorf("the next c: sent %q, want %q", got, "c")
	}

	// Sticky Alt is a terminal Alt even with a character from the phone's
	// symbol layer.
	tr.StickyAlt(true)
	if got := send(t, &tr, tm, down('x', 'x', 0)); got != "\x1bx" {
		t.Errorf("sticky Alt then x: sent %q, want %q", got, "\x1bx")
	}
	tr.StickyAlt(true)
	if got := send(t, &tr, tm, down('#', 'q', ModAlt)); got != "\x1b#" {
		t.Errorf("sticky Alt then the phone's Alt+Q: sent %q, want %q", got, "\x1b#")
	}

	// Both, with an arrow.
	tr.StickyCtrl(true)
	tr.StickyAlt(true)
	if got := send(t, &tr, tm, down(CodeLeft, CodeLeft, 0)); got != "\x1b[1;7D" {
		t.Errorf("sticky Ctrl and Alt then Left: sent %q, want %q", got, "\x1b[1;7D")
	}

	// A release does not use them up; the press does.
	tr.StickyCtrl(true)
	send(t, &tr, tm, Raw{Down: false, Sym: 'z', Cap: 'z'})
	if ctrl, _ := tr.Sticky(); !ctrl {
		t.Error("a release used up the sticky Ctrl")
	}
	tr.StickyCtrl(false)

	// The key bar's Esc.
	var b bytes.Buffer
	if err := input.EncodeKey(&b, tr.Escape(), input.KeyEncodeOptionsFromTerminal(tm)); err != nil || b.String() != "\x1b" {
		t.Errorf("Escape(): sent %q, %v", b.String(), err)
	}
}
