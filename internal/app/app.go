//go:build qnx

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
	"runtime"
	"time"

	"term51/internal/keys"
	"term51/internal/render"
	"term51/internal/replies"
	"term51/internal/session"
	"term51/internal/ui"
	"term51/internal/view"
)

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
	pub  ui.Publisher
}

// New makes the app; Run starts it.
func New(cfg Config) *App {
	return &App{cfg: cfg, cmds: make(chan Command, 256)}
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

// Run serves the configured front end until the shell exits. It returns the
// reason it ended.
func (a *App) Run() error {
	if a.cfg.Frontend == FrontendPixels {
		return a.runPixels()
	}
	return a.runText()
}

// runPixels waits for QML to give the window's place and size, opens the
// window and the session, and serves them.
func (a *App) runPixels() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// The window: QML asks for it once its ForeignWindowControl is laid out.
	var win *ui.Window
	for win == nil {
		c := <-a.cmds
		if c.Name != "window" {
			continue
		}
		group, _ := c.Args["group"].(string)
		id, _ := c.Args["id"].(string)
		var err error
		win, err = ui.OpenWindow(group, id, num(c.Args["w"]), num(c.Args["h"]))
		if err != nil {
			log.Printf("app: %v", err)
			win = nil
		}
	}
	defer win.Close()

	r, err := render.New(a.cfg.FontPx)
	if err != nil {
		return err
	}
	cw, ch := r.CellSize()
	cols, rows := r.Grid(win.Width, win.Height)
	sess, err := session.Start(session.Config{
		Shell: a.cfg.Shell, Args: a.cfg.Args,
		Cols: cols, Rows: rows, CellW: cw, CellH: ch,
		MaxScrollback: a.cfg.MaxScrollback,
		Replies:       replies.DefaultConfig(),
	})
	if err != nil {
		return err
	}
	defer sess.Close()
	v := view.New(sess, r)
	if _, _, err := v.Resize(win.Width, win.Height); err != nil {
		log.Printf("app: %v", err)
	}
	var tr keys.Translator
	a.state(map[string]any{"cols": cols, "rows": rows, "ctrl": false, "alt": false, "rawKeys": a.cfg.RawKeys})

	resize := func(w, h int) {
		if w <= 0 || h <= 0 || (w == win.Width && h == win.Height) {
			return
		}
		if err := win.Resize(w, h); err != nil {
			log.Printf("app: %v", err)
			return
		}
		v.Invalidate() // the window's buffers are new
		if c, rw, err := v.Resize(w, h); err != nil {
			log.Printf("app: %v", err)
		} else {
			a.state(map[string]any{"cols": c, "rows": rw})
		}
	}
	stickyCtrl, stickyAlt := false, false
	sticky := func() {
		if c, al := tr.Sticky(); c != stickyCtrl || al != stickyAlt {
			stickyCtrl, stickyAlt = c, al
			a.state(map[string]any{"ctrl": c, "alt": al})
		}
	}
	key := func(raw keys.Raw) {
		ev, ok := tr.Translate(raw)
		if !ok {
			return
		}
		if raw.Down && !keys.IsModifier(raw) {
			sess.ScrollToBottom()
		}
		if err := sess.Key(ev); err != nil {
			log.Printf("app: key: %v", err)
		}
		sticky()
	}

	dirty := true
	lastActive := time.Now()
	var lastFrame time.Time
	var touchY int
	touching := false
	for {
		// Commands from QML.
		for drained := false; !drained; {
			select {
			case c := <-a.cmds:
				switch c.Name {
				case "size":
					resize(num(c.Args["w"]), num(c.Args["h"]))
					dirty = true
					lastActive = time.Now()
				case "key":
					// A Cascades key event (the page's KeyListener).
					if a.cfg.RawKeys {
						break
					}
					text, _ := c.Args["unicode"].(string)
					down, _ := c.Args["down"].(bool)
					alt, _ := c.Args["alt"].(bool)
					shift, _ := c.Args["shift"].(bool)
					ctrl, _ := c.Args["ctrl"].(bool)
					key(keys.FromCascades(num(c.Args["key"]), num(c.Args["keycap"]), text, down, alt, shift, ctrl))
				case "bar":
					a.bar(sess, &tr, key, c)
					sticky()
				case "paste":
					if text, _ := c.Args["text"].(string); text != "" {
						sess.ScrollToBottom()
						_ = sess.Paste([]byte(text))
					}
				}
			default:
				drained = true
			}
		}

		// The window's own events. The session's news arrives on a Go
		// channel, which this call cannot wait on, so the wait is short
		// while things are happening and longer once they have been quiet
		// for a while (a phone should not wake 125 times a second to look
		// at an idle screen).
		wait := activeWait
		if time.Since(lastActive) > idleAfter {
			wait = idleWait
		}
		ev := win.NextEvent(wait)
		if ev.Type != ui.EventNone {
			lastActive = time.Now()
		}
		switch ev.Type {
		case ui.EventKey:
			if a.cfg.RawKeys {
				key(keys.FromScreen(ev.KeyFlags, ev.KeySym, ev.KeyCap, ev.KeyMods))
			}
		case ui.EventSize:
			resize(ev.W, ev.H)
			dirty = true
		case ui.EventTouch:
			// A vertical drag scrolls through the scrollback, a cell at a
			// time.
			switch ev.Touch {
			case 1:
				touching, touchY = true, ev.Y
			case 2:
				if touching {
					if d := (ev.Y - touchY) / ch; d != 0 {
						sess.ScrollViewport(-d)
						touchY += d * ch
					}
				}
			case 3:
				touching = false
			}
		case ui.EventClose:
			return nil
		}

		// The session's news.
		for more := true; more; {
			select {
			case <-sess.Changed():
				dirty = true
				lastActive = time.Now()
			case e := <-sess.Events():
				switch e.Kind {
				case session.EventTitle:
					a.state(map[string]any{"title": e.Text})
				case session.EventBell:
					a.pub.Event(`{"bell":true}`)
				}
			case <-sess.Done():
				return sess.ExitErr()
			default:
				more = false
			}
		}

		// At most one frame per frameGap: under a flood of output, drawing
		// every change would cost more than the output itself.
		if dirty && time.Since(lastFrame) >= frameGap {
			dirty = false
			lastFrame = time.Now()
			err := win.Frame(func(id int, pix []uint32, stride int) (int, int) {
				y0, y1, err := v.Frame(id, pix, stride, win.Width, win.Height)
				if err != nil {
					log.Printf("app: frame: %v", err)
				}
				return y0, y1
			})
			if err != nil {
				log.Printf("app: %v", err)
			}
		}
	}
}
