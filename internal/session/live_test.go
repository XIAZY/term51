package session

import (
	"testing"

	"github.com/XIAZY/libghostty-vt-go/vt"
)

// The theme and the cursor's shape can change while the shell runs.
func TestThemeAndCursorChangeLive(t *testing.T) {
	// The script asks for a bar cursor, then for the default shape again.
	s := start(t, `printf 'one\n'; read x; printf '\033[5 qtwo\n'; read x; printf '\033[0 qthree\n'; sleep 5`)
	waitText(t, s, "one")
	rs := vt.RenderStateEmpty()
	snap := func() {
		t.Helper()
		if err := s.Snapshot(&rs); err != nil {
			t.Fatal(err)
		}
	}

	snap()
	if rs.Cursor.VisualStyle != vt.CursorStyleBlock {
		t.Errorf("the cursor at the start: %v, want a block", rs.Cursor.VisualStyle)
	}
	s.SetCursor(CursorUnderline)
	snap()
	if rs.Cursor.VisualStyle != vt.CursorStyleUnderline {
		t.Errorf("after SetCursor: %v, want an underline", rs.Cursor.VisualStyle)
	}

	green := RGB{R: 1, G: 200, B: 3}
	s.SetTheme(Theme{Foreground: RGB{R: 9, G: 9, B: 9}, Background: green})
	snap()
	if bg := rs.Colors.Background; bg.R != 1 || bg.G != 200 || bg.B != 3 {
		t.Errorf("the background after SetTheme: %+v", bg)
	}

	// A shape a program asks for wins over the user's, until the program
	// asks for the default again.
	s.Send([]byte("\n"))
	waitText(t, s, "two")
	snap()
	if rs.Cursor.VisualStyle != vt.CursorStyleBar {
		t.Errorf("after the program asked for a bar: %v", rs.Cursor.VisualStyle)
	}
	s.Send([]byte("\n"))
	waitText(t, s, "three")
	snap()
	if rs.Cursor.VisualStyle != vt.CursorStyleUnderline {
		t.Errorf("after the program asked for the default: %v, want the user's underline", rs.Cursor.VisualStyle)
	}
}
