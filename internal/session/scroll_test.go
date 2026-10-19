package session

import (
	"strings"
	"testing"
	"time"

	"github.com/XIAZY/term51/libghostty-vt-go/vt/input"
)

// scrollReport starts a program that sets modes, then shows in hex all it is
// sent until an "x"; the test scrolls, then types the x.
func scrollReport(t *testing.T, modes string, scroll func(s *Session)) string {
	t.Helper()
	s := start(t, `stty raw -echo; printf '`+modes+`ready'; IFS= read -r -d x got; printf 'got<%s>' "$(printf %s "$got" | od -An -v -tx1 | tr -d ' \n')"; sleep 5`)
	waitText(t, s, "ready")
	time.Sleep(100 * time.Millisecond)
	scroll(s)
	if err := s.Key(input.KeyEvent{Action: input.KeyActionPress, UTF8: []byte("x")}); err != nil {
		t.Fatal(err)
	}
	// The report may wrap onto the next row of the 40-column screen.
	text := strings.ReplaceAll(waitText(t, s, "got<"), "\n", "")
	for !strings.Contains(text, ">") {
		time.Sleep(50 * time.Millisecond)
		text, _ = s.Text()
		text = strings.ReplaceAll(text, "\n", "")
	}
	_, after, _ := strings.Cut(text, "got<")
	got, _, _ := strings.Cut(after, ">")
	return got
}

// The cells are 9 by 18 pixels (start): the finger at 50, 60 is in column 5,
// row 3, 0-based.
const fingerX, fingerY = 50, 60

func TestScrollSendsWheelReportsWhenTheProgramAsksForTheMouse(t *testing.T) {
	// Normal tracking and SGR reports: two rows of the finger moving up are
	// two presses of button 5 at its cell, 1-based.
	got := scrollReport(t, `\033[?1000h\033[?1006h`, func(s *Session) { s.Scroll(2, fingerX, fingerY) })
	want := hex("\x1b[<65;6;4M\x1b[<65;6;4M")
	if got != want {
		t.Errorf("sent %s, want %s", got, want)
	}
}

func TestScrollSendsArrowsOnTheAlternateScreen(t *testing.T) {
	got := scrollReport(t, `\033[?1049h`, func(s *Session) { s.Scroll(2, fingerX, fingerY) })
	if want := hex("\x1b[B\x1b[B"); got != want {
		t.Errorf("down: sent %s, want %s", got, want)
	}
	got = scrollReport(t, `\033[?1049h`, func(s *Session) { s.Scroll(-1, fingerX, fingerY) })
	if want := hex("\x1b[A"); got != want {
		t.Errorf("up: sent %s, want %s", got, want)
	}
	// The program's cursor-key mode is honoured.
	got = scrollReport(t, `\033[?1049h\033[?1h`, func(s *Session) { s.Scroll(2, fingerX, fingerY) })
	if want := hex("\x1bOB\x1bOB"); got != want {
		t.Errorf("application cursor keys: sent %s, want %s", got, want)
	}
}

func TestScrollSendsNothingWithAlternateScrollOff(t *testing.T) {
	got := scrollReport(t, `\033[?1049h\033[?1007l`, func(s *Session) { s.Scroll(3, fingerX, fingerY) })
	if got != "" {
		t.Errorf("sent %s, want nothing", got)
	}
}

// Mouse reporting wins over the alternate screen.
func TestScrollPrefersTheMouseOnTheAlternateScreen(t *testing.T) {
	got := scrollReport(t, `\033[?1049h\033[?1000h\033[?1006h`, func(s *Session) { s.Scroll(-1, fingerX, fingerY) })
	if want := hex("\x1b[<64;6;4M"); got != want {
		t.Errorf("sent %s, want %s", got, want)
	}
}

// X10 mode reports only button presses, not the wheel.
func TestScrollSendsNothingInX10Mode(t *testing.T) {
	got := scrollReport(t, `\033[?9h`, func(s *Session) { s.Scroll(2, fingerX, fingerY) })
	if got != "" {
		t.Errorf("sent %s, want nothing", got)
	}
}

// A plain shell: the view moves into the scrollback and nothing is sent.
func TestScrollMovesTheViewOtherwise(t *testing.T) {
	s := start(t, `stty raw -echo; for i in $(seq 1 30); do printf 'line-%s\r\n' $i; done; IFS= read -r -d x got; printf 'got<%s>' "$(printf %s "$got" | od -An -v -tx1 | tr -d ' \n')"; sleep 5`)
	waitText(t, s, "line-30")
	time.Sleep(100 * time.Millisecond)
	s.Scroll(-10, fingerX, fingerY) // ten rows back: the view starts at line 12
	if found, _ := s.SelectWordAt(2, 0); !found {
		t.Fatal("no word at the top of the view")
	}
	if got := selected(t, s); got != "line-12" {
		t.Errorf("the top of the view after scrolling: %q, want line-12", got)
	}
	s.ClearSelection()
	s.ScrollToBottom()
	if err := s.Key(input.KeyEvent{Action: input.KeyActionPress, UTF8: []byte("x")}); err != nil {
		t.Fatal(err)
	}
	if text := waitText(t, s, "got<"); !strings.Contains(text, "got<>") {
		t.Errorf("the shell was sent something: %q", text)
	}
}

// With the user's setting off, a drag only moves the view: the program is
// sent nothing, whether it reads the mouse or fills the alternate screen.
func TestScrollProgramOff(t *testing.T) {
	for _, modes := range []string{`\033[?1000h\033[?1006h`, `\033[?1049h`, `\033[?1049h\033[?1000h`} {
		got := scrollReport(t, modes, func(s *Session) {
			s.SetScrollProgram(false)
			s.Scroll(3, fingerX, fingerY)
		})
		if got != "" {
			t.Errorf("%q: sent %s, want nothing", modes, got)
		}
	}
	// And the view moves, as in a plain shell.
	s := start(t, `for i in $(seq 1 30); do printf 'line-%s\r\n' $i; done; printf '\033[?1000h'; sleep 5`)
	waitText(t, s, "line-30")
	time.Sleep(100 * time.Millisecond)
	s.SetScrollProgram(false)
	s.Scroll(-10, fingerX, fingerY)
	if found, _ := s.SelectWordAt(2, 0); !found {
		t.Fatal("no word at the top of the view")
	}
	if got := selected(t, s); got != "line-12" {
		t.Errorf("the top of the view: %q, want line-12", got)
	}
}

func hex(s string) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 0, 2*len(s))
	for i := range len(s) {
		out = append(out, digits[s[i]>>4], digits[s[i]&15])
	}
	return string(out)
}
