package app

import (
	"bytes"
	"os"
	"strconv"
	"strings"
	"time"
)

// A DEVELOPMENT AID, to be removed before release (with the `remote`
// setting, the bridge's screenshot and the two permissions it needs): the
// page can only be seen on the phone, so with `remote = true` in the
// settings file the app runs commands written to a file, and can take a
// picture of the display. With these the developers look at a page change
// themselves before the owner does.
//
// The file is read a few times a second and removed; one command per line.
// A file that cannot be removed is not run at all, so that it can never run
// twice. The phone asks the user's leave for every picture, so only the
// first "shot" of a file is run:
//
//	drawer open | drawer close   the tabs drawer
//	menu                         the menu from the three dots
//	settings | back              the settings sheet, and closing it
//	tab N                        show tab N (the first is 1)
//	type TEXT                    type TEXT into the shell (\n is Enter)
//	shot NAME                    a picture of the display, data/NAME.png
//	wait N                       wait N milliseconds before the next line
//
// The commands the page carries out reach it as "remote" events.

// remotePoll is how often the command file is looked at.
const remotePoll = 250 * time.Millisecond

// remote reads the command file until stop is closed, and sends each line to
// the loop as a "remote" command.
func (a *App) remote(path string, stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			return
		case <-time.After(remotePoll):
		}
		data, err := os.ReadFile(path)
		if err != nil || len(bytes.TrimSpace(data)) == 0 {
			continue
		}
		if os.Remove(path) != nil {
			continue
		}
		shot := false
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "shot ") {
				if shot {
					continue
				}
				shot = true
			}
			if rest, ok := strings.CutPrefix(line, "wait "); ok {
				ms, _ := strconv.Atoi(rest)
				select {
				case <-stop:
					return
				case <-time.After(time.Duration(ms) * time.Millisecond):
				}
				continue
			}
			a.Send(Command{Name: "remote", Args: map[string]any{"line": line}})
		}
	}
}
