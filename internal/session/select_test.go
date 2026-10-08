package session

import (
	"testing"

	"github.com/XIAZY/libghostty-vt-go/vt"
)

func TestSelection(t *testing.T) {
	s := start(t, `printf 'hello brave world\r\nsecond line here\r\n'; sleep 5`)
	waitText(t, s, "here")

	if _, ok, _ := s.SelectionText(); ok {
		t.Error("something is selected at the start")
	}

	// From "brave" on the first row to "second" on the next, given end
	// first: the order of the two cells does not matter.
	if err := s.Select(5, 1, 6, 0); err != nil {
		t.Fatal(err)
	}
	text, ok, err := s.SelectionText()
	if err != nil || !ok || text != "brave world\nsecond" {
		t.Errorf("selection: %q, %v, %v", text, ok, err)
	}

	// The next snapshot marks the selected cells for drawing.
	rs := vt.RenderStateEmpty()
	if err := s.Snapshot(&rs); err != nil {
		t.Fatal(err)
	}
	if sel := rs.RowData[0].Selection; !sel.Valid || sel.Value[0] != 6 {
		t.Errorf("row 0 selection: %+v", sel)
	}
	if sel := rs.RowData[1].Selection; !sel.Valid || sel.Value != [2]uint16{0, 5} {
		t.Errorf("row 1 selection: %+v", sel)
	}
	if rs.RowData[2].Selection.Valid {
		t.Error("row 2 is marked selected")
	}

	if found, err := s.SelectWordAt(8, 0); err != nil || !found {
		t.Fatalf("SelectWordAt: %v, %v", found, err)
	}
	if text, _, _ := s.SelectionText(); text != "brave" {
		t.Errorf("word: %q", text)
	}
	if !s.Selected(6, 0) || !s.Selected(10, 0) || s.Selected(5, 0) || s.Selected(11, 0) || s.Selected(6, 1) {
		t.Error("Selected does not agree with the selected word, cells 6 to 10 of row 0")
	}
	if found, _ := s.SelectWordAt(30, 5); found {
		t.Error("a word was found on an empty cell")
	}

	if err := s.SelectAll(); err != nil {
		t.Fatal(err)
	}
	if text, _, _ := s.SelectionText(); text != "hello brave world\nsecond line here" {
		t.Errorf("all: %q", text)
	}

	s.ClearSelection()
	if _, ok, _ := s.SelectionText(); ok {
		t.Error("still selected after ClearSelection")
	}

	// Cells outside the view are ignored.
	if err := s.Select(-1, 0, 500, 500); err != nil {
		t.Errorf("Select outside the view: %v", err)
	}
	if _, ok, _ := s.SelectionText(); ok {
		t.Error("a selection outside the view was made")
	}
}
