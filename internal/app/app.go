// Package app is the program behind the page: it runs the shells and tells
// the page what to show.
//
// The page is a column of Cascades text labels, one per terminal row. It
// measures its monospace font and says how many cells fit; the app answers
// with the rows that changed, as rich text. Key presses, the key bar and the
// menu come back as commands.
package app

import (
	"encoding/json"
	"log"

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
	a.event(map[string]any{"notice": text})
}

// event sends the page a one-off message.
func (a *App) event(v map[string]any) {
	b, _ := json.Marshal(v) // a map of strings, numbers and booleans always encodes
	a.pub.Event(string(b))
}

func num(v any) int {
	f, _ := v.(float64)
	return int(f)
}

func (a *App) state(v map[string]any) {
	b, _ := json.Marshal(v) // as in event
	a.pub.State(string(b))
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
