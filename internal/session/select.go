package session

import (
	"github.com/XIAZY/term51/libghostty-vt-go/vt"
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

// HasSelection reports whether anything is selected. A selection goes away
// by itself when its text does (the screen is cleared, the text leaves the
// scrollback).
func (s *Session) HasSelection() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.closed && s.term.Screens.Active.Selection.Valid
}

// End is one end of the selection: the cell of the view it is on, or that
// it is not in the view (Seen is false).
type End struct {
	X, Y int
	Seen bool
}

// SelectionEnds gives the two ends of the selection, the first in the text
// and the last, both part of it. With nothing selected neither is seen.
func (s *Session) SelectionEnds() (first, last End) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	scr := s.term.Screens.Active
	if !scr.Selection.Valid {
		return
	}
	// A pin above the view has no point in it; one below it has a point
	// past the last row.
	end := func(p vt.Pin) End {
		pt := scr.Pages.PointFromPin(vt.PointTagViewport, p)
		if !pt.Valid || int(pt.Value.Viewport.Y) >= int(s.term.Rows) {
			return End{}
		}
		return End{X: int(pt.Value.Viewport.X), Y: int(pt.Value.Viewport.Y), Seen: true}
	}
	return end(scr.Selection.Value.TopLeft(scr)), end(scr.Selection.Value.BottomRight(scr))
}

// order is a pin's place in the text, for comparing two pins: its row
// counted from the top of the scrollback, and its column.
func (s *Session) order(scr *vt.Screen, p vt.Pin) (row int64, col int64) {
	pt := scr.Pages.PointFromPin(vt.PointTagScreen, p)
	return int64(pt.Value.Screen.Y), int64(pt.Value.Screen.X)
}

// StretchStart begins stretching the selection by a finger that has come
// down on the cell x, y of the view. The end of the selection nearer to that
// cell is the one the finger will move; the other end stays where it is in
// the text (the anchor), however the screen scrolls meanwhile. It reports
// false when nothing is selected or the cell is outside the view.
func (s *Session) StretchStart(x, y int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stretchEnd()
	if s.closed {
		return false
	}
	scr := s.term.Screens.Active
	at, ok := s.pin(x, y)
	if !ok || !scr.Selection.Valid {
		return false
	}
	// How far a pin is from the finger, in cells along the text.
	cols := int64(s.term.Cols)
	ar, ac := s.order(scr, at)
	far := func(p vt.Pin) int64 {
		r, c := s.order(scr, p)
		d := (r-ar)*cols + (c - ac)
		if d < 0 {
			return -d
		}
		return d
	}
	start, end := scr.Selection.Value.Start(), scr.Selection.Value.End()
	anchor := start
	if far(start) < far(end) {
		anchor = end
	}
	tracked, err := scr.Pages.TrackPin(anchor)
	if err != nil {
		return false
	}
	s.anchor, s.anchorScreen = tracked, scr
	return true
}

// StretchTo makes the selection run from the anchor to the cell x, y of the
// view, both included. Outside a stretch, or for a cell outside the view,
// it does nothing.
func (s *Session) StretchTo(x, y int) {
	s.mu.Lock()
	if !s.closed && s.anchor != nil && s.anchorScreen == s.term.Screens.Active {
		if at, ok := s.pin(x, y); ok {
			_ = s.anchorScreen.Select(vt.Some(vt.NewSelection(*s.anchor, at, false)))
		}
	}
	s.mu.Unlock()
	s.signal()
}

// StretchEnd ends the stretch: the selection stays as it is.
func (s *Session) StretchEnd() {
	s.mu.Lock()
	s.stretchEnd()
	s.mu.Unlock()
}

// stretchEnd lets go of the anchor. The caller holds mu.
func (s *Session) stretchEnd() {
	if s.anchor != nil && !s.closed {
		s.anchorScreen.Pages.UntrackPin(s.anchor)
	}
	s.anchor, s.anchorScreen = nil, nil
}

// SelectLine widens the selection to the whole line its start is in: a line
// as it was written, so a long command that was wrapped onto several rows
// is one line. It reports whether there was a selection to widen.
func (s *Session) SelectLine() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	scr := s.term.Screens.Active
	if !scr.Selection.Valid {
		return false
	}
	line := scr.SelectLine(vt.ScreenSelectLine{Pin: scr.Selection.Value.Start(), Whitespace: vt.Some([]uint32{0, ' ', '\t'})})
	if !line.Valid {
		return false
	}
	_ = scr.Select(line)
	s.signal()
	return true
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
