package session

import (
	"github.com/XIAZY/libghostty-vt-go/vt"
)

// wordBoundaries are the characters that end a word for SelectWordAt.
var wordBoundaries = []uint32{' ', '\t', '\'', '"', '`', '|', ':', ';', ',', '(', ')', '[', ']', '{', '}', '<', '>', '$', 0x2502}

// pin is the place in the terminal's pages of a cell of the view (the
// visible rows, wherever the view is scrolled to), or false outside it.
func (s *Session) pin(x, y int) (vt.Pin, bool) {
	if x < 0 || y < 0 || x >= int(s.term.Cols) || y >= int(s.term.Rows) {
		return vt.Pin{}, false
	}
	p := s.term.Screens.Active.Pages.Pin(vt.Point{Tag: vt.PointTagViewport, Viewport: vt.Coordinate{X: uint16(x), Y: uint32(y)}})
	return p.Value, p.Valid
}

// Select selects the text from one cell of the view to another, both
// included (the order does not matter). The selection stays with the text
// when the screen scrolls, and is shown by the next frame. Cells outside the
// view are ignored.
func (s *Session) Select(x0, y0, x1, y1 int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	a, ok1 := s.pin(x0, y0)
	b, ok2 := s.pin(x1, y1)
	if !ok1 || !ok2 {
		return nil
	}
	err := s.term.Screens.Active.Select(vt.Some(vt.NewSelection(a, b, false)))
	s.signal()
	return err
}

// SelectWordAt selects the word at a cell of the view; it reports whether
// there was one.
func (s *Session) SelectWordAt(x, y int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false, ErrClosed
	}
	p, ok := s.pin(x, y)
	if !ok {
		return false, nil
	}
	sel := s.term.Screens.Active.SelectWord(p, wordBoundaries)
	if !sel.Valid {
		return false, nil
	}
	err := s.term.Screens.Active.Select(sel)
	s.signal()
	return err == nil, err
}

// Selected reports whether a cell of the view is part of the selection.
func (s *Session) Selected(x, y int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	scr := s.term.Screens.Active
	p, ok := s.pin(x, y)
	return ok && scr.Selection.Valid && scr.Selection.Value.Contains(scr, p)
}

// SelectAll selects everything, the scrollback included.
func (s *Session) SelectAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	sel := s.term.Screens.Active.SelectAll()
	if !sel.Valid {
		return nil
	}
	err := s.term.Screens.Active.Select(sel)
	s.signal()
	return err
}

// ClearSelection removes the selection.
func (s *Session) ClearSelection() {
	s.mu.Lock()
	if !s.closed {
		s.term.Screens.Active.ClearSelection()
	}
	s.mu.Unlock()
	s.signal()
}

// SelectionText returns the selected text, with trailing blanks trimmed
// from each line; ok is false when nothing is selected.
func (s *Session) SelectionText() (text string, ok bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return "", false, ErrClosed
	}
	scr := s.term.Screens.Active
	if !scr.Selection.Valid {
		return "", false, nil
	}
	b, err := scr.SelectionString(vt.ScreenSelectionString{Sel: scr.Selection.Value, Trim: true})
	if err != nil {
		return "", false, err
	}
	return string(b), true, nil
}
