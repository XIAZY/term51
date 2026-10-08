package session

import (
	"strings"
	"testing"
)

func selected(t *testing.T, s *Session) string {
	t.Helper()
	text, ok, err := s.SelectionText()
	if err != nil || !ok {
		t.Fatalf("no selection: %v, %v", ok, err)
	}
	return text
}

// A selection is stretched by moving one of its ends: the one nearer to
// where the finger comes down; the other stays.
func TestStretch(t *testing.T) {
	s := start(t, `printf 'alpha beta gamma\r\nsecond line here\r\n'; sleep 5`)
	waitText(t, s, "here")

	if s.HasSelection() || s.StretchStart(3, 0) {
		t.Fatal("a stretch began with nothing selected")
	}
	s.SelectWordAt(7, 0) // "beta", cells 6 to 9 of row 0
	if !s.HasSelection() {
		t.Fatal("HasSelection is false after selecting a word")
	}

	// A finger to the right of the word moves its end; the start stays.
	if !s.StretchStart(12, 0) {
		t.Fatal("StretchStart refused")
	}
	s.StretchTo(15, 0)
	if got := selected(t, s); got != "beta gamma" {
		t.Errorf("stretched to the end of the row: %q", got)
	}
	s.StretchTo(5, 1)
	if got := selected(t, s); got != "beta gamma\nsecond" {
		t.Errorf("stretched into the next row: %q", got)
	}
	// Back across the anchor: the selection turns round on it.
	s.StretchTo(2, 0)
	if got := selected(t, s); got != "pha b" {
		t.Errorf("stretched back across the anchor: %q", got)
	}
	s.StretchTo(5, 1)
	s.StretchEnd()

	// A finger near the start moves the start; the end stays.
	if !s.StretchStart(3, 0) {
		t.Fatal("StretchStart refused")
	}
	s.StretchTo(0, 0)
	if got := selected(t, s); got != "alpha beta gamma\nsecond" {
		t.Errorf("the start stretched to the top left: %q", got)
	}
	s.StretchEnd()

	// Outside a stretch, StretchTo changes nothing.
	s.StretchTo(15, 1)
	if got := selected(t, s); got != "alpha beta gamma\nsecond" {
		t.Errorf("StretchTo outside a stretch changed the selection: %q", got)
	}
}

// The anchor is tied to its text: when the view scrolls during a stretch the
// selection runs from the same text to the cell now under the finger, also
// when that is in the scrollback.
func TestStretchIntoTheScrollback(t *testing.T) {
	s := start(t, `for i in $(seq 1 30); do echo "line-$i"; done; sleep 5`)
	waitText(t, s, "line-30")
	// The view shows lines 22 to 30 and an empty row; "line-30" is row 8.
	if found, _ := s.SelectWordAt(2, 8); !found {
		t.Fatal("no word on the last line")
	}
	if got := selected(t, s); got != "line-30" {
		t.Fatalf("the word: %q", got)
	}
	if !s.StretchStart(0, 7) { // above the word: its start moves
		t.Fatal("StretchStart refused")
	}
	s.ScrollViewport(-10) // ten rows back: the view starts at line 12
	s.StretchTo(0, 0)
	s.StretchEnd()
	got := selected(t, s)
	if !strings.HasPrefix(got, "line-12\nline-13\n") || !strings.HasSuffix(got, "line-29\nline-30") {
		t.Errorf("stretched into the scrollback: %q", got)
	}
}

// Select Line takes the whole line as it was written: a command wrapped
// onto several rows is one line. Wide characters count as the two cells
// they take.
func TestSelectLineAndWideCharacters(t *testing.T) {
	long := strings.Repeat("abcdefghij", 6) // 60 characters in a 40-column terminal
	s := start(t, `printf 'short one\r\n`+long+`\r\n中文 wide 字\r\n'; sleep 5`)
	waitText(t, s, "wide")

	if s.SelectLine() {
		t.Error("Select Line with nothing selected")
	}
	s.SelectWordAt(5, 2) // in the second row of the wrapped line
	if !s.SelectLine() {
		t.Fatal("SelectLine refused")
	}
	if got := selected(t, s); got != long {
		t.Errorf("the wrapped line: %q", got)
	}
	s.SelectWordAt(2, 0)
	s.SelectLine()
	if got := selected(t, s); got != "short one" {
		t.Errorf("the short line: %q", got)
	}

	// "wide" is cells 5 to 8 of row 3; the two wide characters before it
	// are cells 0 to 3. Stretching the start onto the second half of the
	// first of them takes that character whole.
	s.SelectWordAt(6, 3)
	if !s.StretchStart(4, 3) {
		t.Fatal("StretchStart refused")
	}
	s.StretchTo(1, 3)
	s.StretchEnd()
	if got := selected(t, s); got != "中文 wide" {
		t.Errorf("stretched over wide characters: %q", got)
	}
}

// The two ends of the selection are told by their cells of the view, the
// first in the text first whichever way it was selected, and an end that is
// out of the view is told as not seen.
func TestSelectionEnds(t *testing.T) {
	s := start(t, `for i in $(seq 1 30); do echo "line-$i end"; done; sleep 5`)
	waitText(t, s, "line-30 end")
	ends := func(what string, first, last End) {
		t.Helper()
		if f, l := s.SelectionEnds(); f != first || l != last {
			t.Errorf("%s: the ends are %+v and %+v, want %+v and %+v", what, f, l, first, last)
		}
	}
	// The view shows lines 22 to 30 and an empty row; "line-30" is row 8.
	ends("nothing selected", End{}, End{})
	s.SelectWordAt(9, 8) // "end", cells 8 to 10
	ends("a word", End{8, 8, true}, End{10, 8, true})

	// Stretched backwards into the row above: the anchor is now the last.
	s.StretchStart(8, 8)
	s.StretchTo(5, 7)
	s.StretchEnd()
	ends("two rows, selected backwards", End{5, 7, true}, End{10, 8, true})

	// Scrolled back, the text moves down the view, and out of it.
	s.ScrollViewport(-1)
	ends("one row down", End{5, 8, true}, End{10, 9, true})
	s.ScrollViewport(-1)
	ends("the last end below the view", End{5, 9, true}, End{})
	s.ScrollViewport(-1)
	ends("both below the view", End{}, End{})

	// And up out of it: selected further back, then scrolled forwards.
	s.ScrollViewport(-2)
	if err := s.Select(3, 0, 4, 1); err != nil {
		t.Fatal(err)
	}
	ends("two rows at the top", End{3, 0, true}, End{4, 1, true})
	s.ScrollViewport(1)
	ends("the first end above the view", End{}, End{4, 0, true})
	s.ScrollViewport(1)
	ends("both above the view", End{}, End{})
}
