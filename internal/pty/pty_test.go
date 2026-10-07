package pty

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

// openBSD scans dir for a free ptyXY master and opens the matching ttyXY.
// The test builds a fake /dev with plain files: an absent pair, a master
// that cannot be opened (no permission, as one in use refuses), and a free
// pair, which must be the one chosen. On a phone with real BSD ptys,
// TestOpenBSDRealDevices exercises the actual devices.
func TestOpenBSDScansForAFreePair(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores the permission bits this test relies on")
	}
	dir := t.TempDir()
	mk := func(name string, mode os.FileMode) {
		if err := os.WriteFile(dir+"/"+name, nil, mode); err != nil {
			t.Fatal(err)
		}
	}
	mk("ptyp1", 0o000) // refuses to open, like a master in use
	mk("ttyp1", 0o600)
	mk("ptyp2", 0o600)
	mk("ttyp2", 0o600)
	m, s, err := openBSD(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	defer s.Close()
	if m.Name() != dir+"/ptyp2" || s.Name() != dir+"/ttyp2" {
		t.Fatalf("got %s and %s, want ptyp2 and ttyp2", m.Name(), s.Name())
	}
}

func TestOpenBSDNoDevices(t *testing.T) {
	if _, _, err := openBSD(t.TempDir()); err == nil {
		t.Fatal("openBSD found a pseudo-terminal in an empty directory")
	}
}

// On a system with real BSD pseudo-terminals (BlackBerry 10), the scan opens
// a working pair: bytes written to the slave come out of the master.
func TestOpenBSDRealDevices(t *testing.T) {
	if _, err := os.Stat("/dev/ptyp0"); err != nil {
		t.Skip("no BSD pseudo-terminals here")
	}
	m, s, err := openBSD("/dev")
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	defer s.Close()
	if _, err := s.Write([]byte("ping\n")); err != nil {
		t.Fatal(err)
	}
	readUntil(t, m, "ping", 10*time.Second)
}

// The program is a session leader whose controlling terminal is the pty,
// runs in the directory asked for, and gets its own arguments.
func TestStartSessionDirectoryAndArguments(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command(shell(t), "-c", `echo "pwd=$(pwd)"; echo "arg=$1"; (: </dev/tty) && echo CTTY; echo DONE`, "sh", "two words")
	cmd.Dir = dir
	m, err := Start(cmd, Size{Rows: 24, Cols: 80})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	out := readUntil(t, m, "DONE", 10*time.Second)
	for _, w := range []string{"pwd=" + dir, "arg=two words", "CTTY"} {
		if !strings.Contains(out, w) {
			t.Errorf("output %q lacks %q", out, w)
		}
	}
	if err := cmd.Wait(); err != nil {
		t.Errorf("wait: %v", err)
	}
}

// A program that is not there is an error from Start.
func TestStartOfAMissingProgramIsAnError(t *testing.T) {
	if m, err := Start(exec.Command("/no/such/shell"), Size{Rows: 24, Cols: 80}); err == nil {
		m.Close()
		t.Fatal("no error")
	} else if !strings.Contains(err.Error(), "/no/such/shell") {
		t.Errorf("the error does not name the program: %v", err)
	}
}

// A program that is there but cannot be run says so in the terminal, and
// the process ends with a failure.
func TestAProgramThatCannotBeRunIsReportedInTheTerminal(t *testing.T) {
	notExecutable := filepath.Join(t.TempDir(), "shell")
	if err := os.WriteFile(notExecutable, []byte("not a program\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(notExecutable)
	m, err := Start(cmd, Size{Rows: 24, Cols: 80})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	out := readUntil(t, m, "permission denied", 10*time.Second)
	if !strings.Contains(out, "term51: starting "+notExecutable) {
		t.Errorf("the terminal shows %q", out)
	}
	if err := cmd.Wait(); err == nil {
		t.Error("the process ended without a failure")
	}
}
