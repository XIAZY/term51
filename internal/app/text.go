package app

import (
	"log"
	"reflect"
	"runtime"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/XIAZY/libghostty-vt-go/vt"

	"term51/internal/arrowpad"
	"term51/internal/cells"
	"term51/internal/keys"
	"term51/internal/selbar"
	"term51/internal/session"
	"term51/internal/stretch"
)

// copyLimit is the most text Copy hands to the page: the text travels as
// one message that the page's thread parses, and "select all" over a long
// scrollback can be megabytes. The end of the selection is kept.
const copyLimit = 512 << 10

// textFrame is how often, at most, the screen is drawn: bursts of output
// are merged into one frame.
const textFrame = 33 * time.Millisecond

// sizeQuiet is how long the page's size must have stayed the same before
// the window and the shells are resized: the page reports two or three
// sizes while the on-screen keyboard rows slide in or out, and each resize
// makes the window's buffers again and reflows every shell.
const sizeQuiet = 150 * time.Millisecond

// tab is one shell.
type tab struct {
	sess  *session.Session
	tr    keys.Translator
	title string
	rang  bool // its bell rang while another tab was shown
	// copied: its selection has been copied. Such a selection is finished:
	// it stays shown until the next touch, and is not stretched any more.
	copied bool
	stop   chan struct{} // closed when the tab is closed
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

// Run serves the page until the last shell has exited. The page says where
// the terminal goes and how big it is ("window", then "size"); the loop
// opens the screen there, makes the character sheet for the display, and
// from then on draws the shown shell's terminal whenever it changes. It
// holds any number of tabs, one shown. It returns the reason it ended.
func (a *App) Run() error {
	// The screen's OpenGL context belongs to one thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	var scr Screen
	defer func() {
		if scr != nil {
			scr.Close()
		}
	}()
	var sheet *cells.Sheet // the character shapes, shared by all tabs
	var table cells.Table  // the shown tab's cells
	bg := a.cfg.SessionTheme().Background
	winW, winH := 0, 0 // the window in pixels

	var tabs []*tab
	cur := -1
	inbox := make(chan news, 16)
	rs := vt.RenderStateEmpty()
	cols, rows, cw, ch := 0, 0, 0, 0
	sentW, sentH := 0, 0 // the cell size the shells were last told

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
		titles, rang := make([]string, len(tabs)), make([]bool, len(tabs))
		for i, t := range tabs {
			titles[i], rang[i] = t.title, t.rang
		}
		a.state(map[string]any{"tabs": titles, "rang": rang, "current": cur, "cols": cols, "rows": rows, "cw": cw, "ch": ch, "ctrl": stickyCtrl, "alt": stickyAlt,
			// For the page's drawing of the arrow pad: the same numbers the
			// rule uses.
			"padThreshold": arrowpad.Threshold, "padAxis": arrowpad.Axis})
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
	toldSel := ""                // what the page was last told about the selection,
	toldBarX, toldBarY := -1, -1 // and about the place of its bar
	post := func() {
		lastPost = time.Now()
		if cur < 0 {
			return
		}
		if err := tabs[cur].sess.Snapshot(&rs); err != nil {
			return
		}
		// The page is told with every frame whether there is a selection,
		// and whether it is still open to being stretched: a selection can
		// go away by itself, with its text.
		sel := ""
		if tabs[cur].sess.HasSelection() {
			sel = "open"
			if tabs[cur].copied {
				sel = "done"
			}
		} else {
			tabs[cur].copied = false
		}
		if sel != toldSel {
			toldSel = sel
			a.state(map[string]any{"selected": sel})
		}
		// And where its bar of buttons goes (internal/selbar), by the rows
		// of the view that have selected text in them.
		if sel == "open" {
			var first, last selbar.Row
			seen := false
			for y := range rs.RowData {
				if part := rs.RowData[y].Selection; part.Valid {
					last = selbar.Row{Y: y, Left: int(part.Value[0]), Right: int(part.Value[1])}
					if !seen {
						first, seen = last, true
					}
				}
			}
			if x, y := selbar.Place(first, last, seen, cols, rows, cw, ch); x != toldBarX || y != toldBarY {
				toldBarX, toldBarY = x, y
				a.state(map[string]any{"selBarX": x, "selBarY": y})
			}
		}
		// Update draws the characters it meets for the first time into
		// the sheet, so the sheet's new part is taken after it.
		first, end := table.Update(&rs, sheet)
		fresh := sheet.TakeDirty()
		if !fresh.Empty() {
			scr.Sheet(sheet.Pix, fresh.Min.X, fresh.Min.Y, fresh.Dx(), fresh.Dy())
		}
		if first == end && fresh.Empty() {
			return
		}
		if err := scr.Frame(table.Data, first, end); err != nil {
			log.Printf("app: %v", err)
		}
	}
	// show makes tab i the one on screen: its whole table is drawn.
	show := func(i int) {
		cur = i
		tabs[i].rang = false
		table = cells.Table{}
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

	// font makes the character sheet for the settings' text size. The size
	// is in points: real ones, by the display's dots per inch.
	font := func() error {
		s, err := cells.NewSheet(a.fonts, a.cfg.FontPt*float64(scr.DPI())/72)
		if err != nil {
			return err
		}
		sheet = s
		cw, ch = sheet.CellSize()
		return nil
	}
	// grid lays the cells out in the window and makes the shells fit.
	grid := func() {
		nc, nr := sheet.Grid(winW, winH)
		scr.Grid(nc, nr, cw, ch, bg.R, bg.G, bg.B)
		table = cells.Table{} // the screen's table is new: all of it goes again
		switch {
		case len(tabs) == 0:
			// The first size starts the first shell.
			cols, rows = nc, nr
			if t := open(); t != nil {
				tabs = append(tabs, t)
				show(0)
			}
		case nc != cols || nr != rows || cw != sentW || ch != sentH:
			cols, rows = nc, nr
			for _, t := range tabs {
				if err := t.sess.Resize(cols, rows, cw, ch); err != nil {
					log.Printf("app: %v", err)
				}
			}
			tell()
			post() // the reflowed screen, without waiting for output
		default:
			post()
		}
		sentW, sentH = cw, ch
	}
	// settings tells the page every setting's value, for its settings
	// sheet.
	settings := func() { a.state(map[string]any{"settings": a.cfg.Values(), "bell": a.cfg.Bell}) }

	// fit makes the window, the grid and the shells fit the place the page
	// last reported (want).
	var want Command
	fit := func() error {
		c := want
		w, h := num(c.Args["w"]), num(c.Args["h"])
		if scr != nil && w == winW && h == winH {
			return nil // the page came back to the size the window has
		}
		winW, winH = w, h
		if scr == nil {
			group, _ := c.Args["group"].(string)
			id, _ := c.Args["id"].(string)
			s, err := a.open(group, id, w, h)
			if err != nil {
				log.Printf("app: %v", err)
				a.notice("The screen could not be opened: " + err.Error())
				return nil
			}
			scr = s
			if err := font(); err != nil {
				return err
			}
			settings()
		} else if err := scr.Resize(w, h); err != nil {
			log.Printf("app: %v", err)
			return nil
		}
		grid()
		return nil
	}

	// arrows asks the arrow pad's rule what the finger's place means now:
	// a key to send, and when to ask again if the finger stays where it is.
	// The page is told of each key sent, to show it on the pad.
	var pad arrowpad.Pad
	var padX, padY float64
	var padDue <-chan time.Time
	arrows := func() {
		padDue = nil
		if cur < 0 {
			return
		}
		name, again := pad.Update(padX, padY, time.Now())
		if name != "" {
			a.bar(tabs[cur].sess, &tabs[cur].tr, key, Command{Name: "bar", Args: map[string]any{"name": name}})
			sticky()
			a.event(map[string]any{"pad": name})
		}
		if again > 0 {
			padDue = time.After(again)
		}
	}

	// scrollEdge asks the edge-scrolling rule what the stretching finger's
	// place means now: rows to scroll, after which the selection runs to the
	// cell that is under the finger then, and when to ask again if the
	// finger stays where it is.
	var edge stretch.Edge
	var stretching bool
	var stretchX, stretchY float64
	var stretchDue <-chan time.Time
	// stretchCell is the cell the stretching finger selects to: the one
	// under it, or the nearest one of the view when the finger has slid off
	// the terminal (onto the key bar below it, say).
	stretchCell := func() (int, int) {
		return min(max(int(stretchX), 0), cols-1), min(max(int(stretchY), 0), rows-1)
	}
	scrollEdge := func() {
		stretchDue = nil
		if cur < 0 || !stretching {
			return
		}
		n, again := edge.Update(stretchY, rows, time.Now())
		if n != 0 {
			tabs[cur].sess.ScrollViewport(n)
			tabs[cur].sess.StretchTo(stretchCell())
		}
		if again > 0 {
			stretchDue = time.After(again)
		}
	}

	var pending <-chan time.Time // set while an update is due
	var quiet <-chan time.Time   // set while a new size waits to be used
	for {
		select {
		case c := <-a.cmds:
			if c.Name == "window" || c.Name == "size" {
				// The page's place for the terminal, in pixels. The first
				// one creates the window at once; a later one resizes it
				// when no other has followed for sizeQuiet.
				want = c
				if scr == nil {
					if err := fit(); err != nil {
						return err
					}
				} else {
					quiet = time.After(sizeQuiet)
				}
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
			case "pad":
				// The arrow pad (internal/arrowpad): the terminal was
				// pressed and held ("hold"), the finger is at dx, dy from
				// where it went down ("move"), or it has lifted ("lift").
				switch phase, _ := c.Args["phase"].(string); phase {
				case "hold":
					pad, padX, padY, padDue = arrowpad.Pad{}, 0, 0, nil
				case "move":
					padX, _ = c.Args["dx"].(float64)
					padY, _ = c.Args["dy"].(float64)
					arrows()
				case "lift":
					padDue = nil
				}
			case "paste":
				if text, _ := c.Args["text"].(string); text != "" {
					sess.ScrollToBottom()
					_ = sess.Paste([]byte(text))
				}
			case "selectword":
				// A press and hold selects the word under the finger; on
				// text that is already selected it keeps the selection, so
				// that the menu it opens can copy it (after Select all, for
				// one).
				x, y := num(c.Args["x"]), num(c.Args["y"])
				if sess.Selected(x, y) {
					break
				}
				tabs[cur].copied = false
				if found, _ := sess.SelectWordAt(x, y); !found {
					sess.ClearSelection()
				}
			case "select":
				tabs[cur].copied = false
				_ = sess.Select(num(c.Args["x0"]), num(c.Args["y0"]), num(c.Args["x1"]), num(c.Args["y1"]))
			case "selectall":
				tabs[cur].copied = false
				_ = sess.SelectAll()
			case "selectline":
				// The selection widened to its whole line, a wrapped
				// command being one line.
				tabs[cur].copied = false
				sess.SelectLine()
			case "unselect":
				sess.ClearSelection()
			case "stretch":
				// A finger is stretching the selection: it came down on a
				// cell ("down": the nearer end of the selection is the one
				// it moves), is now over another ("move"), or has lifted
				// ("up"). x and y are in cells and may lie between two.
				// Near the top or bottom edge the screen scrolls by itself
				// (internal/stretch).
				stretchX, _ = c.Args["x"].(float64)
				stretchY, _ = c.Args["y"].(float64)
				switch phase, _ := c.Args["phase"].(string); phase {
				case "down":
					edge, stretchDue = stretch.Edge{}, nil
					stretching = sess.StretchStart(stretchCell())
					tabs[cur].copied = false
				case "move":
					if stretching {
						sess.StretchTo(stretchCell())
						scrollEdge()
					}
				case "up":
					sess.StretchEnd()
					stretching, stretchDue = false, nil
				}
			case "set":
				// One setting changed on the settings sheet. It is written
				// to the file at once. The text size, the colours and the
				// cursor's shape apply to what is on screen; the rest to
				// tabs opened from now on.
				name, _ := c.Args["name"].(string)
				value, _ := c.Args["value"].(string)
				before := a.cfg
				if err := a.cfg.Set(name, value); err != nil {
					a.notice(err.Error())
				} else if err := a.save(a.cfg); err != nil {
					log.Printf("app: %v", err)
					a.notice("The settings could not be saved: " + err.Error())
				}
				redraw := false
				if a.cfg.FontPt != before.FontPt {
					if err := font(); err != nil {
						return err
					}
					redraw = true
				}
				if theme := a.cfg.SessionTheme(); !reflect.DeepEqual(theme, before.SessionTheme()) {
					bg = theme.Background
					for _, t := range tabs {
						t.sess.SetTheme(theme)
					}
					redraw = true
				}
				if a.cfg.Cursor != before.Cursor {
					for _, t := range tabs {
						t.sess.SetCursor(a.cfg.SessionCursor())
					}
					redraw = true
				}
				if redraw {
					grid()
				}
				// Also after a refused value, so that the sheet shows
				// what is in force.
				settings()
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
				// The selection stays shown, as a finished one: the page
				// clears it at the next touch.
				tabs[cur].copied = true
				post()
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
					// A tab that is not shown is marked in the list of
					// tabs until it is looked at.
					for i, t := range tabs {
						if t == n.t {
							a.event(map[string]any{"bell": true, "tab": i, "shown": i == cur})
							if i != cur {
								t.rang = true
								tell()
							}
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
		case <-quiet:
			if len(a.cmds) > 0 {
				// More from the page is waiting, perhaps a newer size:
				// that first.
				quiet = time.After(sizeQuiet)
				break
			}
			quiet = nil
			if err := fit(); err != nil {
				return err
			}
		case <-padDue:
			arrows()
		case <-stretchDue:
			scrollEdge()
		case <-pending:
			pending = nil
			post()
		}
	}
}
