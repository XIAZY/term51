package vt_test

import (
	"testing"

	"github.com/XIAZY/term51/libghostty-vt-go/vt"
	"github.com/XIAZY/term51/libghostty-vt-go/vt/osc"
)

// A program outside the library creates a terminal, feeds it output and reads
// the screen back, using only the public packages.
func TestTerminalFromOutside(t *testing.T) {
	term, err := vt.NewTerminal(vt.TerminalOptions{
		Cols:          20,
		Rows:          3,
		MaxScrollback: 10_000,
		Colors:        vt.TerminalColorsDefault(),
		DefaultModes:  vt.NewModePacked(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Deinit()
	stream := term.VtStream()
	defer stream.Deinit()
	if err := stream.NextSlice([]byte("hello \x1b[1mworld\x1b[0m\r\nline 2")); err != nil {
		t.Fatal(err)
	}
	got, err := term.PlainString()
	if err != nil {
		t.Fatal(err)
	}
	if want := "hello world\nline 2"; string(got) != want {
		t.Errorf("screen: got %q, want %q", got, want)
	}
}

type queryHandler struct {
	ro      vt.ReadonlyHandler
	queries int
}

func (h *queryHandler) Deinit() {}

func (h *queryHandler) Vt(action vt.StreamActionKey, value *vt.StreamAction) error {
	if action == vt.StreamActionKeyColorOperation {
		reqs := &value.ColorOperation.Requests
		for i := 0; i < reqs.Count(); i++ {
			if reqs.At(i).Tag == osc.ParsersColorRequestQuery {
				h.queries++
			}
		}
	}
	return h.ro.Vt(action, value)
}

// A handler outside the library can read the requests of a colour operation
// (OSC 4, 10, 11, ...), which it needs to answer colour queries.
func TestColourRequestsFromOutside(t *testing.T) {
	term, err := vt.NewTerminal(vt.TerminalOptions{Cols: 20, Rows: 3, Colors: vt.TerminalColorsDefault(), DefaultModes: vt.NewModePacked()})
	if err != nil {
		t.Fatal(err)
	}
	h := &queryHandler{ro: vt.NewReadonlyHandler(&term)}
	stream := vt.NewStream(h)
	if err := stream.NextSlice([]byte("\x1b]11;?\x07\x1b]4;1;?;2;rgb:ff/00/00\x1b\\")); err != nil {
		t.Fatal(err)
	}
	if h.queries != 2 {
		t.Errorf("queries seen: %d, want 2", h.queries)
	}
}
