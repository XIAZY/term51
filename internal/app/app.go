// Package app is the program behind the page: it runs the shells and draws
// the one that is shown.
//
// The page has a key bar, a menu, an input field and a place for the
// terminal. The terminal is drawn in a window of the app's own (Screen):
// the loop turns the shown shell's terminal into a table of cells and a
// sheet of character shapes (internal/cells) and hands both to the screen.
// Key presses, the key bar and the menu come from the page as commands.
package app

import (
	"encoding/json"
	"log"

	"term51/internal/cells"
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

// Screen is the window the terminal is drawn in: internal/gpu on the phone.
// It is used only from the goroutine that runs Run, which is locked to its
// thread.
type Screen interface {
	Resize(w, h int) error
	// DPI is the display's dots per inch.
	DPI() int
	Grid(cols, rows, cellW, cellH int, r, g, b uint8)
	Sheet(sheet []byte, x, y, w, h int)
	// Frame draws the table, whose bytes lo <= i < hi changed.
	Frame(table []byte, lo, hi int) error
	Close()
}

// OpenScreen creates the window for the page's ForeignWindowControl, which
// has the given window group and id, w by h pixels.
type OpenScreen func(group, id string, w, h int) (Screen, error)

// Command is a command from QML: a name and its arguments.
type Command struct {
	Name string
	Args map[string]any
}

// App is the running program.
type App struct {
	cfg   settings.Settings
	cmds  chan Command
	pub   Publisher
	open  OpenScreen
	fonts cells.Fonts
	save  func(settings.Settings) error
}

// New makes the app; Run starts it. save writes the settings when the user
// changes one.
func New(cfg settings.Settings, pub Publisher, open OpenScreen, fonts cells.Fonts, save func(settings.Settings) error) *App {
	return &App{cfg: cfg, cmds: make(chan Command, 256), pub: pub, open: open, fonts: fonts, save: save}
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
