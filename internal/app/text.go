//go:build qnx

package app

import (
	"encoding/json"
	"log"
	"time"

	"github.com/XIAZY/libghostty-vt-go/vt"

	"term51/internal/keys"
	"term51/internal/replies"
	"term51/internal/session"
	"term51/internal/textgrid"
)

// textFrame is how often, at most, the labels are updated: Cascades lays a
// label out again for every change, so bursts of output are merged.
const textFrame = 33 * time.Millisecond

// runText serves the text front end: QML measures its monospace font and
// says how many cells fit ("grid"); Go answers with the rows that changed,
// as rich text for the page's labels.
func (a *App) runText() error {
	var sess *session.Session
	var grid textgrid.Grid
	var tr keys.Translator
	rs := vt.RenderStateEmpty()
	cols, rows, cw, ch := 0, 0, 0, 0

	// The session is started by the first "grid" command.
	for sess == nil {
		c := <-a.cmds
		if c.Name != "grid" {
			continue
		}
		if mono, ok := c.Args["mono"].(bool); ok && !mono {
			log.Printf("app: the page's font is not monospace: the grid will not line up")
		}
		cols, rows, cw, ch = num(c.Args["cols"]), num(c.Args["rows"]), num(c.Args["cw"]), num(c.Args["ch"])
		var err error
		sess, err = session.Start(session.Config{
			Shell: a.cfg.Shell, Args: a.cfg.Args,
			Cols: cols, Rows: rows, CellW: cw, CellH: ch,
			MaxScrollback: a.cfg.MaxScrollback,
			Replies:       replies.DefaultConfig(),
		})
		if err != nil {
			log.Printf("app: %v", err)
			sess = nil
		}
	}
	defer sess.Close()
	a.state(map[string]any{"cols": cols, "rows": rows, "ctrl": false, "alt": false})

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
	var lastPost time.Time
	post := func() {
		lastPost = time.Now()
		if err := sess.Snapshot(&rs); err != nil {
			return
		}
		changed := grid.Update(&rs)
		if len(changed) == 0 {
			return
		}
		b, err := json.Marshal(map[string]any{"rows": changed, "count": len(rs.RowData)})
		if err == nil {
			a.pub.Event(string(b))
		}
	}

	post()
	var pending <-chan time.Time // set while an update is due
	for {
		select {
		case c := <-a.cmds:
			switch c.Name {
			case "grid":
				if mono, ok := c.Args["mono"].(bool); ok && !mono {
					log.Printf("app: the page's font is not monospace: the grid will not line up")
				}
				nc, nr := num(c.Args["cols"]), num(c.Args["rows"])
				if nc == cols && nr == rows {
					break
				}
				if err := sess.Resize(nc, nr, num(c.Args["cw"]), num(c.Args["ch"])); err != nil {
					log.Printf("app: %v", err)
					break
				}
				cols, rows = nc, nr
				a.state(map[string]any{"cols": cols, "rows": rows})
				post() // the reflowed screen, without waiting for output
			case "key":
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
			case "scroll":
				sess.ScrollViewport(num(c.Args["rows"]))
			}
		case <-sess.Changed():
			// The first change after a quiet moment is shown at once (the
			// echo of a typed key must not wait); changes that follow
			// closely are merged until the frame time has passed.
			if pending != nil {
				break
			}
			if since := time.Since(lastPost); since >= textFrame {
				post()
			} else {
				pending = time.After(textFrame - since)
			}
		case <-pending:
			pending = nil
			post()
		case e := <-sess.Events():
			switch e.Kind {
			case session.EventTitle:
				a.state(map[string]any{"title": e.Text})
			case session.EventBell:
				a.pub.Event(`{"bell":true}`)
			}
		case <-sess.Done():
			post()
			return sess.ExitErr()
		}
	}
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
