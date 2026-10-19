package terminal

import "testing"

// Go-only tests: a size of zero columns or rows is an error, not a panic,
// and leaves the terminal as it was.

func TestApiArgs_zero_size_is_an_error(t *testing.T) {
	opts := func(c, r SizeCellCountInt) TerminalOptions {
		return TerminalOptions{Cols: c, Rows: r, MaxScrollback: 10_000, Colors: TerminalColorsDefault, DefaultModes: ModesModePackedInit()}
	}
	if _, err := TerminalInit(opts(0, 24)); err != ErrZeroColCount {
		t.Errorf("TerminalInit, 0 columns: %v", err)
	}
	if _, err := TerminalInit(opts(80, 0)); err != ErrZeroRowCount {
		t.Errorf("TerminalInit, 0 rows: %v", err)
	}
	if _, err := TerminalInit(TerminalOptions{}); err != ErrZeroColCount {
		t.Errorf("TerminalInit, zero options: %v", err)
	}
	if _, err := ScreenInit(ScreenOptions{Cols: 0, Rows: 5}); err != ErrZeroColCount {
		t.Errorf("ScreenInit, 0 columns: %v", err)
	}
	if _, err := PageListInit(5, 0, None[uint64]()); err != ErrZeroRowCount {
		t.Errorf("PageListInit, 0 rows: %v", err)
	}

	term, err := TerminalInit(opts(20, 5))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Deinit()
	s := term.VtStream()
	if err := s.NextSlice([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	for _, size := range [][2]SizeCellCountInt{{0, 0}, {0, 5}, {5, 0}} {
		err := term.Resize(size[0], size[1])
		if err != ErrZeroColCount && err != ErrZeroRowCount {
			t.Errorf("Resize(%d, %d): %v", size[0], size[1], err)
		}
		if term.Cols != 20 || term.Rows != 5 {
			t.Errorf("Resize(%d, %d) changed the size to %dx%d", size[0], size[1], term.Cols, term.Rows)
		}
	}
	if err := term.Screens.Active.Resize(ScreenResize{Cols: 0, Rows: 5}); err != ErrZeroColCount {
		t.Errorf("Screen.Resize, 0 columns: %v", err)
	}
	got, err := term.PlainString()
	if err != nil || string(got) != "hello" {
		t.Errorf("after the refused resizes: %q, %v", got, err)
	}
	if err := term.Resize(1, 1); err != nil {
		t.Errorf("Resize(1, 1): %v", err)
	}
}
