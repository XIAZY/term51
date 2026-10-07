package session

import (
	"strings"
	"testing"
	"time"

	"github.com/XIAZY/libghostty-vt-go/vt/input"

	"term51/internal/replies"
)

func start(t *testing.T, script string) *Session {
	t.Helper()
	s, err := Start(Config{
		Shell: "/bin/bash", Args: []string{"-c", script},
		Env:  []string{"TERM=xterm-256color", "PATH=/usr/bin:/bin"},
		Cols: 40, Rows: 10, CellW: 9, CellH: 18,
		MaxScrollback: 1 << 20,
		Replies:       replies.DefaultConfig(),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// waitText waits until the screen contains want.
func waitText(t *testing.T, s *Session, want string) string {
	t.Helper()
	deadline := time.After(10 * time.Second)
	for {
		text, err := s.Text()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(text, want) {
			return text
		}
		select {
		case <-s.Changed():
		case <-time.After(50 * time.Millisecond):
		case <-deadline:
			t.Fatalf("screen never contained %q; it is %q", want, text)
		}
	}
}

func TestOutputReachesTheScreen(t *testing.T) {
	s := start(t, `printf 'hello \033[1;31mworld\033[0m\r\nline 2'; sleep 5`)
	waitText(t, s, "line 2")
	if text, _ := s.Text(); text != "hello world\nline 2" {
		t.Errorf("screen: %q", text)
	}
}

// A program asks where the cursor is and what the terminal is, and gets the
// answers typed back.
func TestQueriesAreAnswered(t *testing.T) {
	s := start(t, `stty raw -echo
printf 'abc\033[6n'; IFS= read -r -d R pos
printf '\033[c'; IFS= read -r -d c da
printf '\r\npos=%s da=%s' "${pos#??}" "${da#??}"; sleep 5`)
	text := waitText(t, s, "da=")
	if !strings.Contains(text, "pos=1;4 da=?62;22") {
		t.Errorf("screen: %q", text)
	}
}

func TestKeysAndPasteReachTheShell(t *testing.T) {
	s := start(t, `stty -echo; IFS= read -r line; printf 'got<%s>' "$line"; sleep 5`)
	time.Sleep(200 * time.Millisecond)
	for _, r := range "hi" {
		if err := s.Key(input.KeyEvent{Action: input.KeyActionPress, UTF8: []byte(string(r))}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Paste([]byte(" there")); err != nil {
		t.Fatal(err)
	}
	if err := s.Key(input.KeyEvent{Action: input.KeyActionPress, Key: input.KeyEnter}); err != nil {
		t.Fatal(err)
	}
	waitText(t, s, "got<hi there>")
}

func TestResizeReachesTheShell(t *testing.T) {
	s := start(t, `trap 'printf "size=%s " "$(stty size)"' WINCH; printf ready; while :; do sleep 0.1; done`)
	waitText(t, s, "ready")
	if err := s.Resize(60, 20, 9, 18); err != nil {
		t.Fatal(err)
	}
	waitText(t, s, "size=20 60")
	if err := s.Resize(0, 20, 9, 18); err == nil {
		t.Error("a zero size was accepted")
	}
}

func TestTitleBellAndExitAreEvents(t *testing.T) {
	s := start(t, `printf '\033]2;my title\007\007done'; exit 3`)
	var title, bell, exit bool
	deadline := time.After(10 * time.Second)
	for !exit {
		select {
		case ev := <-s.Events():
			switch ev.Kind {
			case EventTitle:
				title = ev.Text == "my title"
			case EventBell:
				bell = true
			case EventExit:
				exit = true
				if ev.Err == nil || !strings.Contains(ev.Err.Error(), "3") {
					t.Errorf("exit error: %v", ev.Err)
				}
			}
		case <-deadline:
			t.Fatal("no exit event")
		}
	}
	if !title || !bell {
		t.Errorf("title %v, bell %v", title, bell)
	}
}

// Close returns promptly even when the shell ignores the hang-up signal, and
// a second Close and later calls are safe.
func TestCloseDoesNotHang(t *testing.T) {
	s := start(t, `trap '' HUP; printf ready; sleep 30`)
	waitText(t, s, "ready")
	began := time.Now()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if d := time.Since(began); d > closeGrace+closeLimit+time.Second {
		t.Errorf("Close took %v", d)
	}
	select {
	case <-s.Done():
	default:
		t.Error("Done is not closed after Close")
	}
	if err := s.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
	if _, err := s.Text(); err != ErrClosed {
		t.Errorf("Text after Close: %v", err)
	}
	if err := s.Key(input.KeyEvent{Action: input.KeyActionPress, UTF8: []byte("x")}); err != ErrClosed {
		t.Errorf("Key after Close: %v", err)
	}
}

// The shell's exit is reported when the shell exits, even while a background
// job still holds the pseudo-terminal open.
func TestExitWithABackgroundJob(t *testing.T) {
	s := start(t, `(trap '' HUP; sleep 30) & printf started; exit 0`)
	select {
	case <-s.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("Done did not close while a background job held the terminal")
	}
	if err := s.ExitErr(); err != nil {
		t.Errorf("exit error: %v", err)
	}
}

func TestTermIsSet(t *testing.T) {
	got := withTerm([]string{"A=1"})
	if len(got) != 2 || got[1] != "TERM="+DefaultTerm {
		t.Errorf("TERM added: %v", got)
	}
	if got := withTerm([]string{"TERM=vt100"}); len(got) != 1 || got[0] != "TERM=vt100" {
		t.Errorf("TERM kept: %v", got)
	}
	found := 0
	for _, kv := range withTerm(nil) {
		if strings.HasPrefix(kv, "TERM=") {
			found++
			if kv != "TERM="+DefaultTerm {
				t.Errorf("inherited environment: %s", kv)
			}
		}
	}
	if found != 1 {
		t.Errorf("TERM entries in the inherited environment: %d", found)
	}
}

// Output larger than one feed chunk arrives whole.
func TestLargeOutput(t *testing.T) {
	s := start(t, `head -c 300000 /dev/zero | tr '\0' 'x'; printf '\r\nEND'; sleep 5`)
	waitText(t, s, "END")
}

// The theme's colours are the terminal's defaults, so a program that asks
// for the background colour gets it.
func TestThemeColoursAreReported(t *testing.T) {
	s := start(t, `stty raw -echo
printf '\033]11;?\033\\'; IFS= read -r -d '\' bg
printf '\r\nbg=%s' "${bg#??}"; sleep 5`)
	text := waitText(t, s, "bg=")
	if !strings.Contains(text, "bg=11;rgb:1010/1414/1818") {
		t.Errorf("screen: %q", text)
	}
}

func TestThemePalette(t *testing.T) {
	red := RGB{0xAA, 0, 0}
	c := Theme{Foreground: RGB{1, 2, 3}, Background: RGB{4, 5, 6}, Cursor: &red, Palette: []RGB{{9, 9, 9}}}.colors()
	if fg := c.Foreground.Get(); !fg.Valid || fg.Value.R != 1 || fg.Value.B != 3 {
		t.Errorf("foreground %+v", fg)
	}
	if cur := c.Cursor.Get(); !cur.Valid || cur.Value.R != 0xAA {
		t.Errorf("cursor %+v", cur)
	}
	if p := c.Palette.Current[0]; p.R != 9 {
		t.Errorf("palette[0] %+v", p)
	}
	if p, lib := c.Palette.Current[1], (Theme{}).colors().Palette.Current[1]; p != lib {
		t.Errorf("palette[1] %+v, the library's is %+v", p, lib)
	}
}
