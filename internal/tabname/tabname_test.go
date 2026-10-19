package tabname

import (
	"errors"
	"testing"
	"time"
)

func source(path string, err error, calls *int) Source {
	return func() (string, error) {
		*calls++
		return path, err
	}
}

// As in Terminal.app: "directory — program", a program's title in the
// directory's place, and the program alone while no directory is known.
func TestName(t *testing.T) {
	calls := 0
	tab := New("/app/data", "/app/data/.local/share/term51/bin/ash")
	if got := tab.Name(); got != "ash" {
		t.Errorf("a new tab is %q", got)
	}

	tab.Pwd("file:///app/data")
	if got := tab.Name(); got != "~ — ash" {
		t.Errorf("the shell at home: %q", got)
	}
	tab.Pwd("file:///app/data/projects/term51")
	if got := tab.Name(); got != "term51 — ash" {
		t.Errorf("the shell in a directory: %q", got)
	}
	tab.Pwd("file:///")
	if got := tab.Name(); got != "/ — ash" {
		t.Errorf("the shell at the root: %q", got)
	}
	tab.Pwd("file:///app/data/my%20files")
	if got := tab.Name(); got != "my files — ash" {
		t.Errorf("an escaped directory: %q", got)
	}
	tab.Pwd("file:///app/data/a b")
	if got := tab.Name(); got != "a b — ash" {
		t.Errorf("a raw space: %q", got)
	}
	tab.Pwd("http://example.com/x")
	if got := tab.Name(); got != "a b — ash" {
		t.Errorf("a report that is not a file URL changed the name to %q", got)
	}

	tab.Pwd("file:///app/data/projects/term51")
	tab.Refresh(source("/app/data/.local/share/term51/bin/vi", nil, &calls))
	if got := tab.Name(); got != "term51 — vi" {
		t.Errorf("vi in the foreground: %q", got)
	}
	tab.Title = "README.md"
	if got := tab.Name(); got != "README.md — vi" {
		t.Errorf("a program's title: %q", got)
	}
	tab.Title = ""
	tab.Refresh(source("/app/data/.local/share/term51/bin/ash", nil, &calls))
	if got := tab.Name(); got != "term51 — ash" {
		t.Errorf("back at the shell: %q", got)
	}

	// PowerShell reports no directory: the program alone.
	ps := New("/app/data", "/app/data/.local/share/term51/powershell/pwsh/pwsh")
	if got := ps.Name(); got != "pwsh" {
		t.Errorf("PowerShell at its prompt: %q", got)
	}
	ps.Refresh(source("/app/data/.local/bin/pi", nil, &calls))
	if got := ps.Name(); got != "pi" {
		t.Errorf("pi from PowerShell: %q", got)
	}
}

// The program is looked up once the output has been quiet for Quiet after
// the last of a burst, not while it continues and not while the tab is
// idle; a lookup that fails keeps the name.
func TestRefreshWhenQuietAfterABurst(t *testing.T) {
	calls := 0
	tab := New("/home", "/bin/ash")
	t0 := time.Unix(1000, 0)

	if due, wait := tab.Due(t0); due || wait != 0 {
		t.Fatalf("an idle tab is due (%v, %v)", due, wait)
	}
	tab.Output(t0)
	tab.Output(t0.Add(100 * time.Millisecond))
	tab.Output(t0.Add(200 * time.Millisecond))
	if due, wait := tab.Due(t0.Add(400 * time.Millisecond)); due || wait != 100*time.Millisecond {
		t.Errorf("during the burst's quiet time: due %v, wait %v", due, wait)
	}
	due, _ := tab.Due(t0.Add(500 * time.Millisecond))
	if !due {
		t.Fatal("not due once the output has been quiet for Quiet")
	}
	if err := tab.Refresh(source("/usr/bin/pi", nil, &calls)); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || tab.Name() != "pi" {
		t.Errorf("after the refresh: %d lookups, name %q", calls, tab.Name())
	}
	if due, _ := tab.Due(t0.Add(time.Hour)); due {
		t.Error("due again without new output")
	}

	tab.Output(t0.Add(time.Hour))
	failed := errors.New("no such process")
	if err := tab.Refresh(source("", failed, &calls)); !errors.Is(err, failed) {
		t.Errorf("the lookup's error: %v", err)
	}
	if tab.Name() != "pi" {
		t.Errorf("a failed lookup changed the name to %q", tab.Name())
	}
}
