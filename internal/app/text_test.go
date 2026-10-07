package app

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"term51/internal/settings"
)

// page stands in for the QML page: it keeps the merged state and the text of
// every row, as the labels would.
type page struct {
	mu        sync.Mutex
	state     map[string]any
	rows      map[int]string
	count     int
	clipboard string
	notices   []string
}

func (p *page) State(js string) {
	var m map[string]any
	if json.Unmarshal([]byte(js), &m) != nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for k, v := range m {
		p.state[k] = v
	}
}

func (p *page) Event(js string) {
	var e struct {
		Rows []struct {
			Y    int    `json:"y"`
			HTML string `json:"html"`
		} `json:"rows"`
		Count     int     `json:"count"`
		Clipboard *string `json:"clipboard"`
		Notice    *string `json:"notice"`
	}
	if json.Unmarshal([]byte(js), &e) != nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if e.Clipboard != nil {
		p.clipboard = *e.Clipboard
	}
	if e.Notice != nil {
		p.notices = append(p.notices, *e.Notice)
	}
	if e.Rows != nil {
		if e.Count != p.count {
			p.count = e.Count
			p.rows = map[int]string{}
		}
		for _, r := range e.Rows {
			p.rows[r.Y] = r.HTML
		}
	}
}

// screen is the page's text with the markup removed.
func (p *page) screen() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	var sb strings.Builder
	for y := 0; y < p.count; y++ {
		inTag := false
		for _, r := range p.rows[y] {
			switch {
			case r == '<':
				inTag = true
			case r == '>':
				inTag = false
			case !inTag && r == ' ':
				sb.WriteByte(' ')
			case !inTag:
				sb.WriteRune(r)
			}
		}
		sb.WriteByte('\n')
	}
	return strings.NewReplacer("&lt;", "<", "&gt;", ">", "&amp;", "&", "&quot;", `"`).Replace(sb.String())
}

func (p *page) get(key string) any {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state[key]
}

var debugPage *page

func wait(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			if debugPage != nil {
				t.Logf("screen:\n%s\nstate: %v", debugPage.screen(), debugPage.state)
			}
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func startApp(t *testing.T) (*App, *page, chan error) {
	t.Helper()
	p := &page{state: map[string]any{}, rows: map[int]string{}}
	debugPage = p
	cfg := settings.Default()
	cfg.Shell, cfg.Args = "/bin/bash", []string{"--norc", "--noprofile"}
	a := New(cfg, p)
	done := make(chan error, 1)
	go func() { done <- a.Run() }()
	a.Send(Command{Name: "grid", Args: map[string]any{"cols": 40.0, "rows": 8.0, "cw": 10.0, "ch": 20.0, "mono": true}})
	wait(t, "the first prompt", func() bool { return strings.Contains(p.screen(), "$") })
	return a, p, done
}

// typeLine pastes a line and presses Enter. (The Enter is a key: bash turns
// bracketed paste on, and a pasted newline is then text, not Enter.)
func typeLine(a *App, s string) {
	a.Send(Command{Name: "paste", Args: map[string]any{"text": s}})
	a.Send(Command{Name: "key", Args: map[string]any{"down": true, "key": 13.0, "keycap": 13.0}})
}

func TestTextFrontEndShowsTheShell(t *testing.T) {
	a, p, done := startApp(t)
	typeLine(a, "echo one-$((40+2))")
	wait(t, "the command's output", func() bool { return strings.Contains(p.screen(), "one-42") })
	if p.count != 8 {
		t.Errorf("the page has %d rows, want 8", p.count)
	}

	// The key bar: sticky Ctrl, then a "d" from the keyboard, ends the shell,
	// and with the last shell the app.
	a.Send(Command{Name: "bar", Args: map[string]any{"name": "ctrl"}})
	wait(t, "Ctrl shown as armed", func() bool { return p.get("ctrl") == true })
	a.Send(Command{Name: "key", Args: map[string]any{"down": true, "key": float64('d'), "keycap": float64('d'), "unicode": "d"}})
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the app did not end when its last shell exited")
	}
	if p.get("ctrl") != false {
		t.Error("Ctrl is still shown as armed after the key")
	}
}

func TestTabs(t *testing.T) {
	a, p, done := startApp(t)
	typeLine(a, "echo in-first-tab")
	wait(t, "the first tab's output", func() bool { return strings.Contains(p.screen(), "in-first-tab") })

	a.Send(Command{Name: "newtab"})
	wait(t, "a second tab", func() bool {
		tabs, _ := p.get("tabs").([]any)
		return len(tabs) == 2 && p.get("current") == 1.0
	})
	wait(t, "the new tab's empty screen", func() bool { return !strings.Contains(p.screen(), "in-first-tab") })
	typeLine(a, "echo in-second-tab")
	wait(t, "the second tab's output", func() bool { return strings.Contains(p.screen(), "in-second-tab") })

	// Output in a tab that is not shown does not reach the page; switching
	// back shows that tab as it is now.
	a.Send(Command{Name: "tab", Args: map[string]any{"index": 0.0}})
	wait(t, "the first tab again", func() bool {
		s := p.screen()
		return strings.Contains(s, "in-first-tab") && !strings.Contains(s, "in-second-tab")
	})

	// Closing the shown tab shows the other one.
	a.Send(Command{Name: "closetab"})
	wait(t, "one tab left", func() bool {
		tabs, _ := p.get("tabs").([]any)
		return len(tabs) == 1 && p.get("current") == 0.0
	})
	wait(t, "the remaining tab's screen", func() bool { return strings.Contains(p.screen(), "in-second-tab") })

	// Its shell exiting ends the app.
	typeLine(a, "exit")
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the app did not end when its last shell exited")
	}
}

func TestCopyAndResize(t *testing.T) {
	a, p, done := startApp(t)
	typeLine(a, "echo copy-me")
	wait(t, "output", func() bool { return strings.Contains(p.screen(), "copy-me") })

	// Copy with nothing selected copies nothing and says so.
	a.Send(Command{Name: "copy"})
	wait(t, "the notice", func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return len(p.notices) == 1 && p.clipboard == ""
	})
	a.Send(Command{Name: "selectall"})
	a.Send(Command{Name: "copy"})
	wait(t, "the clipboard", func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return strings.Contains(p.clipboard, "copy-me")
	})

	a.Send(Command{Name: "grid", Args: map[string]any{"cols": 30.0, "rows": 5.0, "cw": 10.0, "ch": 20.0}})
	wait(t, "the resized page", func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.count == 5 && p.state["cols"] == 30.0
	})
	typeLine(a, "stty size")
	wait(t, "the shell's new size", func() bool { return strings.Contains(p.screen(), "5 30") })

	// Quit (the user closing the app) hangs up every shell and ends the
	// loop, also with a shell that ignores the hang-up.
	a.Send(Command{Name: "newtab"})
	wait(t, "a second tab", func() bool { return p.get("current") == 1.0 })
	typeLine(a, "trap '' HUP; echo armed")
	wait(t, "the trap", func() bool { return strings.Contains(p.screen(), "armed") })
	began := time.Now()
	a.Quit()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run returned %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Quit did not end the loop")
	}
	if d := time.Since(began); d > 5*time.Second {
		t.Errorf("Quit took %v", d)
	}
}

// A long selection is cut to its end before it goes to the page.
func TestCopyIsCapped(t *testing.T) {
	a, p, _ := startApp(t)
	typeLine(a, "head -c 700000 /dev/zero | tr '\\0' 'x' | fold -w 39; echo THE-END")
	wait(t, "the output", func() bool { return strings.Contains(p.screen(), "THE-END") })
	a.Send(Command{Name: "selectall"})
	a.Send(Command{Name: "copy"})
	wait(t, "the clipboard", func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.clipboard != ""
	})
	p.mu.Lock()
	n, end, notices := len(p.clipboard), strings.Contains(p.clipboard, "THE-END"), len(p.notices)
	p.mu.Unlock()
	if n > copyLimit || n < copyLimit/2 || !end || notices != 1 {
		t.Errorf("copied %d bytes (limit %d), end kept: %v, notices: %d", n, copyLimit, end, notices)
	}
	a.Quit()
}

// Characters typed into the page's input field arrive as "text"; the key
// bar's sticky Ctrl applies to the first of them.
func TestTextFromTheInputField(t *testing.T) {
	a, p, _ := startApp(t)
	a.Send(Command{Name: "text", Args: map[string]any{"text": "echo $((6*7))-typed"}})
	a.Send(Command{Name: "key", Args: map[string]any{"down": true, "key": 13.0, "keycap": 13.0}})
	wait(t, "the output", func() bool { return strings.Contains(p.screen(), "42-typed") })

	typeLine(a, "cat")
	a.Send(Command{Name: "bar", Args: map[string]any{"name": "ctrl"}})
	a.Send(Command{Name: "text", Args: map[string]any{"text": "d"}})
	typeLine(a, "echo after-cat")
	wait(t, "cat ended by Ctrl-D", func() bool { return strings.Contains(p.screen(), "\nafter-cat ") })
	a.Quit()
}
