package app

import (
	"strings"
	"testing"
)

// A program that asks for application cursor keys (as PowerShell does at
// its prompt) gets the arrow pad's and the bar's keys in that form: ESC O A,
// not ESC [ A. The keys go through the terminal library's encoder, which
// follows the terminal's modes.
func TestArrowPadFollowsTheCursorKeyMode(t *testing.T) {
	a, p, done := startApp(t)
	// cat -v shows what arrives; the two printf set the modes PowerShell
	// sets: application cursor keys and application keypad.
	typeLine(a, `clear; printf '\033[?1h\033='; stty raw -echo; echo RE''ADY; cat -v`)
	wait(t, "cat running", func() bool { return strings.Contains(p.screen(), "READY") })
	a.Send(Command{Name: "pad", Args: map[string]any{"phase": "hold"}})
	a.Send(Command{Name: "pad", Args: map[string]any{"phase": "move", "dx": 0.0, "dy": -70.0}})
	a.Send(Command{Name: "pad", Args: map[string]any{"phase": "lift"}})
	a.Send(Command{Name: "pad", Args: map[string]any{"phase": "hold"}})
	a.Send(Command{Name: "pad", Args: map[string]any{"phase": "move", "dx": -70.0, "dy": 0.0}})
	a.Send(Command{Name: "pad", Args: map[string]any{"phase": "lift"}})
	wait(t, "the two arrows in application form", func() bool { return strings.Contains(p.screen(), "^[OA^[OD") })
	if strings.Contains(p.screen(), "^[[A") {
		t.Errorf("an arrow was sent in the normal form:\n%s", p.screen())
	}
	a.Quit()
	if err := <-done; err != nil {
		t.Errorf("Run returned %v", err)
	}
}
