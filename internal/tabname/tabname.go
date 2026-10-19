// Package tabname decides what a tab is called, as the live terminals do
// (Terminal.app, iTerm2, Konsole, WezTerm, Ghostty): a title that a program
// set (OSC 0 or 2) is the name; without one, the name of the program in the
// foreground of the tab's terminal; and while that is the shell itself,
// waiting at its prompt, the last part of the directory the shell reported
// (OSC 7, ~ for home), or the shell's name as the settings call it when it
// reported none.
//
// The program is looked up again when the tab's output has gone quiet
// after a burst, since a program that starts or ends almost always writes
// to the screen, and not while the tab is idle.
package tabname

import (
	"net/url"
	"path/filepath"
	"time"
)

// Quiet is how long a tab's output must have stopped before its program is
// looked up again.
const Quiet = 300 * time.Millisecond

// Program is what runs in the foreground of a terminal.
type Program struct {
	Shell bool   // it is the tab's shell itself
	Path  string // the path it was started by, when it is not the shell
}

// Source finds the program in the foreground of a tab's terminal.
type Source func() (Program, error)

// Tab is one tab's name.
type Tab struct {
	Shell string // the shell's name, as the settings call it
	Home  string // the home folder, shown as ~
	Title string // the title a program set; empty when none is set

	dir     string // the directory the shell last reported; empty if none
	program string // the foreground program's name, empty for the shell
	due     time.Time
}

// Name is what the tab is called.
func (t *Tab) Name() string {
	switch {
	case t.Title != "":
		return t.Title
	case t.program != "":
		return t.program
	case t.dir == "":
		return t.Shell
	case t.dir == t.Home:
		return "~"
	}
	return filepath.Base(t.dir)
}

// Pwd takes the directory the shell reported, a file URL (OSC 7). A report
// that is not one is ignored.
func (t *Tab) Pwd(report string) {
	u, err := url.Parse(report)
	if err != nil || u.Scheme != "file" || u.Path == "" {
		return
	}
	t.dir = filepath.Clean(u.Path)
}

// Output notes that the tab wrote to its screen at now: its program is due
// to be looked up Quiet after the last output.
func (t *Tab) Output(now time.Time) { t.due = now.Add(Quiet) }

// Due says whether the program is due to be looked up at now, and if it is
// not, how long until it will be (zero when nothing is due).
func (t *Tab) Due(now time.Time) (bool, time.Duration) {
	if t.due.IsZero() {
		return false, 0
	}
	if wait := t.due.Sub(now); wait > 0 {
		return false, wait
	}
	return true, 0
}

// Refresh looks the program up with src. A lookup that fails leaves the
// name as it was.
func (t *Tab) Refresh(src Source) error {
	t.due = time.Time{}
	p, err := src()
	if err != nil {
		return err
	}
	t.program = ""
	if !p.Shell && p.Path != "" {
		t.program = filepath.Base(p.Path)
	}
	return nil
}
