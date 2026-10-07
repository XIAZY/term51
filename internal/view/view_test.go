package view

import (
	"strings"
	"testing"
	"time"

	"term51/internal/render"
	"term51/internal/replies"
	"term51/internal/session"
)

func start(t *testing.T, script string) (*session.Session, *render.Renderer) {
	t.Helper()
	r, err := render.New(20)
	if err != nil {
		t.Fatal(err)
	}
	cw, ch := r.CellSize()
	s, err := session.Start(session.Config{
		Shell: "/bin/bash", Args: []string{"-c", script},
		Env:  []string{"PATH=/usr/bin:/bin"},
		Cols: 20, Rows: 4, CellW: cw, CellH: ch,
		MaxScrollback: 1 << 20,
		Replies:       replies.DefaultConfig(),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s, r
}

func waitText(t *testing.T, s *session.Session, want string) {
	t.Helper()
	deadline := time.After(10 * time.Second)
	for {
		text, err := s.Text()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(text, want) {
			return
		}
		select {
		case <-s.Changed():
		case <-time.After(50 * time.Millisecond):
		case <-deadline:
			t.Fatalf("screen never contained %q; it is %q", want, text)
		}
	}
}

// Frames drawn in turn into two buffers, each redrawing only its stale rows,
// end up identical to a full redraw of the final state.
func TestTwoBuffersConvergeOnTheFullPicture(t *testing.T) {
	s, r := start(t, `stty -echo; printf 'first\r\n'; IFS= read -r a; printf 'second %s\r\n' "$a"; IFS= read -r b; printf 'third %s' "$b"; sleep 5`)
	cw, ch := r.CellSize()
	w, h := 20*cw+3, 4*ch+2
	v := New(s, r)
	if cols, rows, err := v.Resize(w, h); err != nil || cols != 20 || rows != 4 {
		t.Fatalf("Resize: %dx%d, %v", cols, rows, err)
	}
	bufs := [2][]uint32{make([]uint32, w*h), make([]uint32, w*h)}
	turn := 0
	frame := func() bool {
		drawn, err := v.Frame(bufs[turn], w, w, h)
		if err != nil {
			t.Fatal(err)
		}
		if drawn {
			turn ^= 1 // a window swaps buffers when a frame is shown
		}
		return drawn
	}

	waitText(t, s, "first")
	if !frame() {
		t.Error("the first frame drew nothing")
	}
	frame() // the second buffer's first frame is a full one too
	if frame() {
		t.Error("a frame was drawn although nothing changed")
	}

	s.Send([]byte("x\n"))
	waitText(t, s, "second x")
	frame()
	s.Send([]byte("y\n"))
	waitText(t, s, "third y")
	frame()
	frame()
	frame()

	want := make([]uint32, w*h)
	v2 := New(s, r)
	if _, err := v2.Frame(want, w, w, h); err != nil {
		t.Fatal(err)
	}
	for k := range bufs {
		for i := range want {
			if bufs[k][i] != want[i] {
				t.Fatalf("buffer %d: pixel %d,%d differs from a full redraw", k, i%w, i/w)
			}
		}
	}
}

func TestResizeFitsTheTerminalToTheArea(t *testing.T) {
	s, r := start(t, `trap 'printf "size=%s " "$(stty size)"' WINCH; printf ready; while :; do sleep 0.1; done`)
	waitText(t, s, "ready")
	cw, ch := r.CellSize()
	v := New(s, r)
	if cols, rows, err := v.Resize(30*cw+5, 6*ch+1); err != nil || cols != 30 || rows != 6 {
		t.Fatalf("Resize: %dx%d, %v", cols, rows, err)
	}
	waitText(t, s, "size=6 30")
	// The same area again changes nothing.
	if cols, rows, err := v.Resize(30*cw, 6*ch); err != nil || cols != 30 || rows != 6 {
		t.Fatalf("second Resize: %dx%d, %v", cols, rows, err)
	}
}
