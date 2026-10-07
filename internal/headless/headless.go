// Package headless runs the app's whole pipeline without a window: a real
// shell in a session, keys translated as the phone's are, and the view
// drawing frames into pixel buffers that are handed out in turn, as a
// window's are. It exists to test and measure the app on a machine that has
// no phone attached; cmd/t51shot is its command line.
package headless

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"time"

	"term51/internal/keys"
	"term51/internal/render"
	"term51/internal/replies"
	"term51/internal/session"
	"term51/internal/view"
)

// Options describes the window to imitate and the program to run in it.
type Options struct {
	// Shell and Args are the program; Env its environment (nil: the
	// session's default).
	Shell string
	Args  []string
	Env   []string

	// W and H are the size of the drawing area in pixels, FontPx the font
	// size in pixels.
	W, H   int
	FontPx float64

	// Buffers is how many pixel buffers the frames alternate between (a
	// Screen window has two). Zero means two.
	Buffers int
}

// Window is a running session with its view and buffers.
type Window struct {
	Session *session.Session
	View    *view.View
	Keys    keys.Translator

	w, h   int
	bufs   [][]uint32
	next   int // the buffer the next frame is drawn into
	shown  int // the buffer last "posted"; -1 before the first frame
	frames int
	rows   int // pixel rows painted, over all frames
}

// Open starts the program and draws the first frame.
func Open(o Options) (*Window, error) {
	if o.W <= 0 || o.H <= 0 {
		return nil, fmt.Errorf("headless: bad size %dx%d", o.W, o.H)
	}
	if o.Buffers <= 0 {
		o.Buffers = 2
	}
	r, err := render.New(o.FontPx)
	if err != nil {
		return nil, err
	}
	cols, rows := r.Grid(o.W, o.H)
	cw, ch := r.CellSize()
	sess, err := session.Start(session.Config{
		Shell: o.Shell, Args: o.Args, Env: o.Env,
		Cols: cols, Rows: rows, CellW: cw, CellH: ch,
		MaxScrollback: 1 << 20,
		Replies:       replies.DefaultConfig(),
	})
	if err != nil {
		return nil, err
	}
	win := &Window{Session: sess, View: view.New(sess, r), w: o.W, h: o.H, shown: -1}
	for i := 0; i < o.Buffers; i++ {
		win.bufs = append(win.bufs, make([]uint32, o.W*o.H))
	}
	if _, _, err := win.View.Resize(o.W, o.H); err != nil {
		sess.Close()
		return nil, err
	}
	if err := win.Frame(); err != nil {
		sess.Close()
		return nil, err
	}
	return win, nil
}

// Frame draws one frame into the next buffer and, if anything was painted,
// "posts" it: that buffer becomes the one shown and the next frame goes to
// the following buffer, as with a window's buffers.
func (w *Window) Frame() error {
	y0, y1, err := w.View.Frame(w.next, w.bufs[w.next], w.w, w.w, w.h)
	if err != nil {
		return err
	}
	if y1 > y0 {
		w.shown = w.next
		w.next = (w.next + 1) % len(w.bufs)
		w.frames++
		w.rows += y1 - y0
	}
	return nil
}

// Stats is how many frames were posted and how many pixel rows they painted
// in total.
func (w *Window) Stats() (frames, pixelRows int) { return w.frames, w.rows }

// Settle draws frames as the terminal changes, until it has been quiet for
// quiet or limit has passed.
func (w *Window) Settle(quiet, limit time.Duration) error {
	deadline := time.After(limit)
	for {
		select {
		case <-w.Session.Changed():
			if err := w.Frame(); err != nil {
				return err
			}
		case <-time.After(quiet):
			return w.Frame()
		case <-deadline:
			return w.Frame()
		}
	}
}

// WaitText draws frames until the screen's text contains want.
func (w *Window) WaitText(want string, limit time.Duration) error {
	deadline := time.After(limit)
	for {
		text, err := w.Session.Text()
		if err != nil {
			return err
		}
		if strings.Contains(text, want) {
			return w.Frame()
		}
		select {
		case <-w.Session.Changed():
			if err := w.Frame(); err != nil {
				return err
			}
		case <-time.After(20 * time.Millisecond):
		case <-deadline:
			return fmt.Errorf("headless: the screen never contained %q; it is %q", want, text)
		}
	}
}

// Key sends one key event as the phone would deliver it.
func (w *Window) Key(r keys.Raw) error {
	ev, ok := w.Keys.Translate(r)
	if !ok {
		return nil
	}
	return w.Session.Key(ev)
}

// Type presses and releases the key of each character of s. '\n' is the
// Enter key and '\t' the Tab key; an upper-case letter is typed with Shift.
func (w *Window) Type(s string) error {
	for _, c := range s {
		r := keys.Raw{Down: true, Sym: c, Cap: c}
		switch {
		case c == '\n':
			r.Sym, r.Cap = keys.CodeReturn, keys.CodeReturn
		case c == '\t':
			r.Sym, r.Cap = keys.CodeTab, keys.CodeTab
		case c >= 'A' && c <= 'Z':
			r.Cap = c + 'a' - 'A'
			r.Mods = keys.ModShift
		}
		if err := w.Key(r); err != nil {
			return err
		}
		r.Down = false
		if err := w.Key(r); err != nil {
			return err
		}
	}
	return nil
}

// Image is the buffer last shown, as an image; nil before the first frame.
func (w *Window) Image() *image.RGBA {
	if w.shown < 0 {
		return nil
	}
	img := image.NewRGBA(image.Rect(0, 0, w.w, w.h))
	for i, p := range w.bufs[w.shown] {
		img.SetRGBA(i%w.w, i/w.w, color.RGBA{R: uint8(p >> 16), G: uint8(p >> 8), B: uint8(p), A: 0xFF})
	}
	return img
}

// Pixels is the buffer last shown (0x00RRGGBB per pixel, W pixels per row);
// nil before the first frame.
func (w *Window) Pixels() []uint32 {
	if w.shown < 0 {
		return nil
	}
	return w.bufs[w.shown]
}

// Close ends the session.
func (w *Window) Close() error { return w.Session.Close() }
