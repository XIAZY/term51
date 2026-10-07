package vt_test

import (
	"testing"

	"github.com/XIAZY/libghostty-vt-go/vt"
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
