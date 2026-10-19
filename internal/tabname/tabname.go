// Package tabname decides what a tab is called. It follows macOS
// Terminal.app's tab title, "Working directory or document — Active process
// name": "~ — ash" at the shell's prompt at home, "term51 — vi" with vi open
// in that directory, "pwsh" for PowerShell, which reports no directory. The
// first part is the title a program set (OSC 0 or 2), as Terminal.app puts
// a document there, or else the last part of the directory the shell
// reported (OSC 7, ~ for home); the second is the file name of the program
// in the foreground of the tab's terminal, the shell included. With no
// first part the program stands alone.
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

// Separator is Terminal.app's, between the directory and the program.
const Separator = " — "

// Source gives the path of the program in the foreground of a tab's
// terminal.
type Source func() (string, error)

// Tab is one tab's name.
type Tab struct {
	Title string // the title a program set; empty when none is set

	home    string
	dir     string // the directory the shell last reported; empty if none
	program string // the foreground program's file name
	due     time.Time
}

// New is the name of a tab whose shell is the program at shell, in the
// home folder home.
func New(home, shell string) Tab {
	return Tab{home: home, program: filepath.Base(shell)}
}

// Name is what the tab is called.
func (t *Tab) Name() string {
	first := t.Title
	switch {
	case first != "":
	case t.dir == "":
	case t.dir == t.home:
		first = "~"
	default:
		first = filepath.Base(t.dir)
	}
	if first == "" {
		return t.program
	}
	return first + Separator + t.program
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
	path, err := src()
	if err != nil {
		return err
	}
	t.program = filepath.Base(path)
	return nil
}
