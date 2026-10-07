package app

import (
	"log"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/XIAZY/libghostty-vt-go/vt"

	"term51/internal/keys"
	"term51/internal/session"
	"term51/internal/textgrid"
)

// copyLimit is the most text Copy hands to the page: the text travels as
// one message that the page's thread parses, and "select all" over a long
// scrollback can be megabytes. The end of the selection is kept.
const copyLimit = 512 << 10

// textFrame is how often, at most, the labels are updated: Cascades lays a
// label out again for every change, so bursts of output are merged.
const textFrame = 33 * time.Millisecond

// tab is one shell.
type tab struct {
	sess  *session.Session
	tr    keys.Translator
	title string
	stop  chan struct{} // closed when the tab is closed
}

// news is something a tab's session reported.
type news struct {
	t       *tab
	changed bool
	ev      *session.Event
	done    bool
}

// follow forwards a session's news to the loop until the tab is closed or
// its shell has exited.
func (t *tab) follow(out chan<- news) {
	send := func(n news) bool {
		select {
		case out <- n:
			return true
		case <-t.stop:
			return false
		}
	}
	for {
		select {
		case <-t.sess.Changed():
			if !send(news{t: t, changed: true}) {
				return
			}
		case e := <-t.sess.Events():
			if !send(news{t: t, ev: &e}) {
				return
			}
		case <-t.sess.Done():
			send(news{t: t, done: true})
			return
		case <-t.stop:
			return
		}
	}
}

// Run serves the page until the last shell has exited: QML measures its
// monospace font and says how many cells fit ("grid"); Go answers with the
// rows that changed, as rich text for the page's labels. It holds any
// number of tabs, one shown. It returns the reason it ended.
func (a *App) Run() error {
	var tabs []*tab
	cur := -1
	inbox := make(chan news, 16)
	var grid textgrid.Grid
	rs := vt.RenderStateEmpty()
	cols, rows, cw, ch := 0, 0, 0, 0

	open := func() *tab {
		sess, err := session.Start(a.cfg.Session(cols, rows, cw, ch, nil))
		if err != nil {
			log.Printf("app: %v", err)
			a.notice("The shell could not be started: " + err.Error())
			return nil
		}
		t := &tab{sess: sess, title: a.cfg.Shell, stop: make(chan struct{})}
		go t.follow(inbox)
		return t
	}
	// tell publishes what the page shows about the tabs and the key bar.
	stickyCtrl, stickyAlt := false, false
	tell := func() {
		titles := make([]string, len(tabs))
		for i, t := range tabs {
			titles[i] = t.title
		}
		st := map[string]any{"tabs": titles, "current": cur, "cols": cols, "rows": rows, "ctrl": stickyCtrl, "alt": stickyAlt}
		if cur >= 0 {
			st["title"] = tabs[cur].title
		}
		a.state(st)
	}
	sticky := func() {
		if cur < 0 {
			return
		}
		if c, al := tabs[cur].tr.Sticky(); c != stickyCtrl || al != stickyAlt {
			stickyCtrl, stickyAlt = c, al
			a.state(map[string]any{"ctrl": c, "alt": al})
		}
	}
	var lastPost time.Time
	post := func() {
		lastPost = time.Now()
		if cur < 0 {
			return
		}
		if err := tabs[cur].sess.Snapshot(&rs); err != nil {
			return
		}
		changed := grid.Update(&rs)
		if len(changed) == 0 {
			return
		}
		a.event(map[string]any{"rows": changed, "count": len(rs.RowData)})
	}
	// show makes tab i the one on screen: every row is sent again.
	show := func(i int) {
		cur = i
		grid = textgrid.Grid{}
		sticky()
		tell()
		post()
	}
	// closing counts the shells being hung up in the background.
	var closing sync.WaitGroup
	// shutdown hangs up every shell and waits for them (each within the
	// session's bound of a few seconds). The app must not exit before: a
	// shell left behind may outlive it.
	shutdown := func() {
		for _, t := range tabs {
			close(t.stop)
			closing.Add(1)
			go func(t *tab) { defer closing.Done(); t.sess.Close() }(t)
		}
		tabs, cur = nil, -1
		closing.Wait()
	}
	// drop removes a tab; its shell is hung up in the background (Close
	// can take a moment and must not hold the loop).
	drop := func(t *tab) {
		for i, x := range tabs {
			if x != t {
				continue
			}
			close(t.stop)
			closing.Add(1)
			go func() { defer closing.Done(); t.sess.Close() }()
			tabs = append(tabs[:i], tabs[i+1:]...)
			switch {
			case len(tabs) == 0:
				cur = -1
			case i < cur:
				// The shown tab moved down one place; its screen is the
				// same.
				cur--
				tell()
			case cur >= len(tabs):
				show(cur - 1)
			case i == cur:
				show(cur)
			default:
				tell()
			}
			return
		}
	}
	key := func(raw keys.Raw) {
		if cur < 0 {
			return
		}
		t := tabs[cur]
		ev, ok := t.tr.Translate(raw)
		if !ok {
			return
		}
		if raw.Down && !keys.IsModifier(raw) {
			t.sess.ScrollToBottom()
		}
		if err := t.sess.Key(ev); err != nil {
			log.Printf("app: key: %v", err)
		}
		sticky()
	}

	var pending <-chan time.Time // set while an update is due
	for {
		select {
		case c := <-a.cmds:
			if c.Name == "grid" {
				nc, nr := num(c.Args["cols"]), num(c.Args["rows"])
				if nc == cols && nr == rows && len(tabs) > 0 {
					break
				}
				cols, rows, cw, ch = nc, nr, num(c.Args["cw"]), num(c.Args["ch"])
				if len(tabs) == 0 {
					// The first size starts the first shell.
					if t := open(); t != nil {
						tabs = append(tabs, t)
						show(0)
					}
					break
				}
				for _, t := range tabs {
					if err := t.sess.Resize(cols, rows, cw, ch); err != nil {
						log.Printf("app: %v", err)
					}
				}
				tell()
				post() // the reflowed screen, without waiting for output
				break
			}
			if c.Name == "ready" {
				// The page is up: give it what it needs before it measures.
				a.state(map[string]any{"fontPt": a.cfg.FontPt})
				break
			}
			if c.Name == "quit" {
				shutdown()
				return nil
			}
			if cur < 0 {
				break // nothing to act on before the first shell
			}
			sess := tabs[cur].sess
			if a.cfg.LogInput {
				log.Printf("input: %s %v", c.Name, c.Args)
			}
			switch c.Name {
			case "text":
				// Characters from the page's input field (the physical keys
				// and the on-screen rows alike), one key press each, so that
				// the key bar's sticky modifiers apply.
				text, _ := c.Args["text"].(string)
				for _, r := range text {
					key(keys.FromCascades(int(r), 0, string(r), true, false, false, false))
				}
			case "key":
				text, _ := c.Args["unicode"].(string)
				down, _ := c.Args["down"].(bool)
				alt, _ := c.Args["alt"].(bool)
				shift, _ := c.Args["shift"].(bool)
				ctrl, _ := c.Args["ctrl"].(bool)
				key(keys.FromCascades(num(c.Args["key"]), num(c.Args["keycap"]), text, down, alt, shift, ctrl))
			case "bar":
				a.bar(sess, &tabs[cur].tr, key, c)
				sticky()
			case "paste":
				if text, _ := c.Args["text"].(string); text != "" {
					sess.ScrollToBottom()
					_ = sess.Paste([]byte(text))
				}
			case "selectword":
				if found, _ := sess.SelectWordAt(num(c.Args["x"]), num(c.Args["y"])); !found {
					sess.ClearSelection()
				}
			case "select":
				_ = sess.Select(num(c.Args["x0"]), num(c.Args["y0"]), num(c.Args["x1"]), num(c.Args["y1"]))
			case "selectall":
				_ = sess.SelectAll()
			case "unselect":
				sess.ClearSelection()
			case "copy":
				// The selection goes back to the page, which owns the
				// clipboard.
				text, ok, _ := sess.SelectionText()
				if !ok {
					a.notice("Nothing is selected")
					break
				}
				msg := map[string]any{"clipboard": text}
				if len(text) > copyLimit {
					cut := len(text) - copyLimit
					for cut < len(text) && !utf8.RuneStart(text[cut]) {
						cut++
					}
					msg["clipboard"] = text[cut:]
					a.notice("The selection is very long: its last 512 KB were copied")
				}
				a.event(msg)
				sess.ClearSelection()
			case "scroll":
				sess.ScrollViewport(num(c.Args["rows"]))
			case "newtab":
				if t := open(); t != nil {
					tabs = append(tabs, t)
					show(len(tabs) - 1)
				}
			case "tab":
				if i := num(c.Args["index"]); i >= 0 && i < len(tabs) && i != cur {
					show(i)
				}
			case "closetab":
				i := cur
				if v, ok := c.Args["index"]; ok {
					i = num(v)
				}
				if i >= 0 && i < len(tabs) {
					drop(tabs[i])
					if len(tabs) == 0 {
						closing.Wait()
						return nil
					}
				}
			}
		case n := <-inbox:
			switch {
			case n.done:
				err := n.t.sess.ExitErr()
				if cur >= 0 && tabs[cur] == n.t {
					post() // the shell's last words
				}
				drop(n.t)
				if len(tabs) == 0 {
					closing.Wait()
					return err
				}
			case n.ev != nil:
				switch n.ev.Kind {
				case session.EventTitle:
					n.t.title = n.ev.Text
					tell()
				case session.EventBell:
					// With the tab, so that the page can mark a tab that
					// is not shown instead of beeping for it.
					for i, t := range tabs {
						if t == n.t {
							a.event(map[string]any{"bell": true, "tab": i, "shown": i == cur})
						}
					}
				}
			case n.changed:
				if cur < 0 || tabs[cur] != n.t || pending != nil {
					break
				}
				// The first change after a quiet moment is shown at once
				// (the echo of a typed key must not wait); changes that
				// follow closely are merged until the frame time has
				// passed.
				if since := time.Since(lastPost); since >= textFrame {
					post()
				} else {
					pending = time.After(textFrame - since)
				}
			}
		case <-pending:
			pending = nil
			post()
		}
	}
}
