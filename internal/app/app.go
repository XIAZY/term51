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

// Config is what the app starts with.
type Config struct {
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
	return Config{FontPx: 34, Shell: "/bin/sh", MaxScrollback: 8 << 20}
}

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

// Run waits for QML to give the window's place and size, opens the window
// and the session, and serves them until the shell exits. It returns the
// reason it ended.
func (a *App) Run() error {
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
	a.state(map[string]any{"cols": cols, "rows": rows, "ctrl": false, "alt": false})

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
	key := func(raw keys.Raw) {
		ev, ok := tr.Translate(raw)
		if !ok {
			return
		}
		sess.ScrollToBottom()
		if err := sess.Key(ev); err != nil {
			log.Printf("app: key: %v", err)
		}
		ctrl, alt := tr.Sticky()
		a.state(map[string]any{"ctrl": ctrl, "alt": alt})
	}

	dirty := true
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
				case "key":
					// A Cascades key event (the page's KeyListener).
					text, _ := c.Args["unicode"].(string)
					down, _ := c.Args["down"].(bool)
					alt, _ := c.Args["alt"].(bool)
					shift, _ := c.Args["shift"].(bool)
					ctrl, _ := c.Args["ctrl"].(bool)
					key(keys.FromCascades(num(c.Args["key"]), num(c.Args["keycap"]), text, down, alt, shift, ctrl))
				case "bar":
					// The key bar.
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
					ctrl, alt := tr.Sticky()
					a.state(map[string]any{"ctrl": ctrl, "alt": alt})
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

		// The window's own events. The wait is short because the session's
		// news arrives on a Go channel, which this call cannot wait on.
		ev := win.NextEvent(8 * time.Millisecond)
		switch ev.Type {
		case ui.EventKey:
			key(keys.FromScreen(ev.KeyFlags, ev.KeySym, ev.KeyCap, ev.KeyMods))
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

		if dirty {
			dirty = false
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
