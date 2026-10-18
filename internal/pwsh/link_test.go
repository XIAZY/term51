package pwsh

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncLink(t *testing.T) {
	home := t.TempDir()
	link := filepath.Join(home, ".local/bin/pwsh")
	program := filepath.Join(home, "powershell/pwsh/pwsh")

	// Ready: the link is made, its folder too.
	if err := SyncLink(link, program, State{Phase: Ready}); err != nil {
		t.Fatalf("ready: %v", err)
	}
	if got, err := os.Readlink(link); err != nil || got != program {
		t.Fatalf("after ready: %q, %v; want a link to %s", got, err, program)
	}
	// Ready again: nothing changes.
	if err := SyncLink(link, program, State{Phase: Ready}); err != nil {
		t.Fatalf("ready again: %v", err)
	}

	// A download in progress leaves it.
	for _, phase := range []string{Downloading, Unpacking} {
		if err := SyncLink(link, program, State{Phase: phase}); err != nil {
			t.Fatalf("%s: %v", phase, err)
		}
		if _, err := os.Lstat(link); err != nil {
			t.Errorf("%s removed the link: %v", phase, err)
		}
	}

	// A program at another place: the link follows.
	other := filepath.Join(home, "elsewhere/pwsh")
	if err := SyncLink(link, other, State{Phase: Ready}); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.Readlink(link); got != other {
		t.Errorf("after a move: the link points to %q", got)
	}

	// No PowerShell: the link goes, and going again is no error.
	for _, phase := range []string{Absent, Failed} {
		if err := SyncLink(link, program, State{Phase: Ready}); err != nil {
			t.Fatal(err)
		}
		if err := SyncLink(link, program, State{Phase: phase}); err != nil {
			t.Fatalf("%s: %v", phase, err)
		}
		if _, err := os.Lstat(link); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s left the link: %v", phase, err)
		}
		if err := SyncLink(link, program, State{Phase: phase}); err != nil {
			t.Errorf("%s with no link: %v", phase, err)
		}
	}
}
