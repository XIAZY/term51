package pty

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// readUntil reads from r until the accumulated output contains want or the
// timeout passes, and returns everything read.
func readUntil(t *testing.T, r io.Reader, want string, timeout time.Duration) string {
	t.Helper()
	got := make(chan string, 1)
	go func() {
		var b strings.Builder
		br := bufio.NewReader(r)
		buf := make([]byte, 256)
		for {
			n, err := br.Read(buf)
			b.Write(buf[:n])
			if strings.Contains(b.String(), want) || err != nil {
				got <- b.String()
				return
			}
		}
	}()
	select {
	case s := <-got:
		if !strings.Contains(s, want) {
			t.Fatalf("output %q does not contain %q", s, want)
		}
		return s
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for %q", want)
		return ""
	}
}

func shell(t *testing.T) string {
	t.Helper()
	for _, p := range []string{"/bin/sh", "/base/bin/sh"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	t.Skip("no /bin/sh")
	return ""
}

// The child sees the pty as its terminal, at the size given to Start.
func TestStartChildHasControllingTerminalAndSize(t *testing.T) {
	cmd := exec.Command(shell(t), "-c", "[ -t 0 ] && echo TTY0; [ -t 1 ] && echo TTY1; stty size; echo DONE")
	m, err := Start(cmd, Size{Rows: 24, Cols: 80})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	out := readUntil(t, m, "DONE", 10*time.Second)
	for _, w := range []string{"TTY0", "TTY1", "24 80"} {
		if !strings.Contains(out, w) {
			t.Errorf("output %q lacks %q", out, w)
		}
	}
	if err := cmd.Wait(); err != nil {
		t.Errorf("wait: %v", err)
	}
}

// SetSize reaches the child.
func TestSetSize(t *testing.T) {
	cmd := exec.Command(shell(t), "-c", "echo READY; read x; stty size; echo DONE")
	m, err := Start(cmd, Size{Rows: 24, Cols: 80})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	readUntil(t, m, "READY", 10*time.Second)
	if err := SetSize(m, Size{Rows: 30, Cols: 100}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Write([]byte("\n")); err != nil {
		t.Fatal(err)
	}
	readUntil(t, m, "30 100", 10*time.Second)
	cmd.Wait()
}

// Input written to the master reaches the child as typed text, echoed by the
// line discipline.
func TestInputReachesChild(t *testing.T) {
	cmd := exec.Command(shell(t), "-c", "read line; echo GOT:$line")
	m, err := Start(cmd, Size{Rows: 24, Cols: 80})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if _, err := m.Write([]byte("hello\n")); err != nil {
		t.Fatal(err)
	}
	readUntil(t, m, "GOT:hello", 10*time.Second)
	cmd.Wait()
}

func TestStartRejectsPresetStdio(t *testing.T) {
	cmd := exec.Command("/bin/true")
	cmd.Stdout = os.Stdout
	if _, err := Start(cmd, Size{Rows: 24, Cols: 80}); err == nil {
		t.Fatal("Start accepted a cmd with Stdout set")
	}
}
