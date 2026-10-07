// Package app is the program on the phone: it owns the native window and
// connects it to one terminal session.
//
// One goroutine, locked to its OS thread, owns the window. It takes the
// commands QML sends (the window's size, key presses, the key bar), the
// window's own events (raw keys, touch) and the session's news, and draws a
// frame when something changed.
package app

import (
	"encoding/json"
	"log"
	"time"

	"term51/internal/keys"
	"term51/internal/session"
	"term51/internal/settings"
)

// Publisher is the way to the page: State replaces what the page knows
// (merged by key), Event is a one-off message. Both take JSON and may be
// called from any goroutine. On the phone it is the Cascades bridge.
type Publisher interface {
	State(json string)
	Event(json string)
}

// Command is a command from QML: a name and its arguments.
type Command struct {
	Name string
	Args map[string]any
}

// The pixel loop's timing.
const (
	frameGap   = 16 * time.Millisecond  // at most one frame per gap
	activeWait = 8 * time.Millisecond   // the wait for window events while busy
	idleWait   = 50 * time.Millisecond  // and once it has been quiet
	idleAfter  = 500 * time.Millisecond // for this long
)

// App is the running program.
type App struct {
	cfg  settings.Settings
	cmds chan Command
	pub  Publisher
}

// New makes the app; Run starts it.
func New(cfg settings.Settings, pub Publisher) *App {
	return &App{cfg: cfg, cmds: make(chan Command, 256), pub: pub}
}

// Send queues a command from QML. It is called on the UI thread and never
// blocks: a command that does not fit is dropped and logged.
func (a *App) Send(c Command) {
	select {
	case a.cmds <- c:
	default:
		log.Printf("app: command queue full, dropped %s", c.Name)
	}
}

// Quit asks the running loop to hang up every shell and end; Run returns
// when that is done. The page's own exit calls it, so that no shell outlives
// the app.
func (a *App) Quit() {
	a.cmds <- Command{Name: "quit"}
}

// notice shows the user a short message (a toast on the phone).
func (a *App) notice(text string) {
	if b, err := json.Marshal(map[string]any{"notice": text}); err == nil {
		a.pub.Event(string(b))
	}
}

func num(v any) int {
	f, _ := v.(float64)
	return int(f)
}

func (a *App) state(v map[string]any) {
	b, err := json.Marshal(v)
	if err == nil {
		a.pub.State(string(b))
	}
}

// Page is the QML page of the configured front end.
func (a *App) Page() string {
	if a.cfg.Frontend == settings.FrontendPixels {
		return "pixels"
	}
	return "text"
}

// bar handles a key of the key bar.
func (a *App) bar(sess *session.Session, tr *keys.Translator, key func(keys.Raw), c Command) {
	switch name, _ := c.Args["name"].(string); name {
	case "ctrl":
		on, _ := tr.Sticky()
		tr.StickyCtrl(!on)
	case "alt":
		_, on := tr.Sticky()
		tr.StickyAlt(!on)
	case "esc":
		sess.ScrollToBottom()
		_ = sess.Key(tr.Escape())
	case "tab":
		key(keys.Raw{Down: true, Sym: keys.CodeTab, Cap: keys.CodeTab})
	case "up":
		key(keys.Raw{Down: true, Sym: keys.CodeUp, Cap: keys.CodeUp})
	case "down":
		key(keys.Raw{Down: true, Sym: keys.CodeDown, Cap: keys.CodeDown})
	case "left":
		key(keys.Raw{Down: true, Sym: keys.CodeLeft, Cap: keys.CodeLeft})
	case "right":
		key(keys.Raw{Down: true, Sym: keys.CodeRight, Cap: keys.CodeRight})
	}
}
