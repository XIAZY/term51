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
