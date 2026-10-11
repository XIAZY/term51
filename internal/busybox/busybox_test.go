package busybox

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The machine's own BusyBox stands in for the phone's.
func program(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("busybox")
	if err != nil {
		t.Skip("no busybox on this machine")
	}
	// A copy, so that the test can change the file.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	copied := filepath.Join(t.TempDir(), "busybox")
	if err := os.WriteFile(copied, data, 0o755); err != nil {
		t.Fatal(err)
	}
	return copied
}

func TestLinks(t *testing.T) {
	prog := program(t)
	dir := filepath.Join(t.TempDir(), "bin")

	if err := Links(prog, dir); err != nil {
		t.Fatal(err)
	}
	// The tools are there by name and run.
	out, err := exec.Command(filepath.Join(dir, "tr"), "a-z", "A-Z").CombinedOutput()
	if err == nil {
		// tr with no input prints nothing; what counts is that it ran.
		_ = out
	} else {
		t.Fatalf("tr through its link: %v: %s", err, out)
	}
	if target, err := os.Readlink(filepath.Join(dir, "ash")); err != nil || target != prog {
		t.Errorf("ash links to %q (%v), want %q", target, err, prog)
	}

	// A second call leaves the folder alone: a file put there stays.
	mark := filepath.Join(dir, "mark")
	if err := os.WriteFile(mark, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Links(prog, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(mark); err != nil {
		t.Error("the links were made again although nothing changed")
	}

	// Another program file: the links are made again.
	f, err := os.OpenFile(prog, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("\n")
	f.Close()
	if err := Links(prog, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(mark); err == nil {
		t.Error("the links were not made again for a changed program")
	}
	if _, err := os.Lstat(filepath.Join(dir, "head")); err != nil {
		t.Errorf("head is missing after the links were made again: %v", err)
	}

	// The same file at another place: made again, pointing there.
	moved := filepath.Join(t.TempDir(), "busybox")
	data, _ := os.ReadFile(prog)
	if err := os.WriteFile(moved, data, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Links(moved, dir); err != nil {
		t.Fatal(err)
	}
	if target, _ := os.Readlink(filepath.Join(dir, "ash")); target != moved {
		t.Errorf("after the program moved, ash links to %q, want %q", target, moved)
	}
}

func TestLinksWithoutTheProgram(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "bin")
	err := Links(filepath.Join(t.TempDir(), "none"), dir)
	if err == nil || !strings.HasPrefix(err.Error(), "busybox: ") {
		t.Errorf("no error for a missing program: %v", err)
	}
	if _, statErr := os.Stat(dir); statErr == nil {
		t.Error("a links folder was made without a program")
	}
}
