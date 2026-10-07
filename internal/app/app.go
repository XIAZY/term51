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

// The two front ends. Which one is better on the phone (an exact grid, the
// phone's own fonts, speed) is to be found out there.
const (
	// FrontendText shows the terminal as a column of Cascades text labels,
	// one per row (assets/text.qml, internal/textgrid).
	FrontendText = "text"
	// FrontendPixels draws the terminal into a native window inside the
	// page (assets/pixels.qml, internal/render).
	FrontendPixels = "pixels"
)

// Config is what the app starts with.
type Config struct {
	// Frontend is FrontendText or FrontendPixels.
	Frontend string
	// RawKeys chooses where the pixel front end takes the keyboard from:
	// false, the page's Cascades key events; true, raw Screen key events
	// forwarded to the native window. Only one is served, so that a press
	// the phone delivers both ways is not typed twice. Which one the
	// phone needs is to be found out there.
	RawKeys bool
	// FontPx is the font size in pixels.
	FontPx float64
	// Shell is the program to run, with its arguments.
	Shell string
	Args  []string
	// MaxScrollback is the scrollback limit in bytes.
	MaxScrollback uint64
}

// DefaultConfig suits the Passport: a 1440 pixel wide screen at about 450
// pixels per inch.
func DefaultConfig() Config {
	return Config{Frontend: FrontendText, FontPx: 34, Shell: "/bin/sh", MaxScrollback: 8 << 20}
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
	cfg  Config
	cmds chan Command
	pub  Publisher
}

// New makes the app; Run starts it.
func New(cfg Config, pub Publisher) *App {
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
	if a.cfg.Frontend == FrontendPixels {
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
