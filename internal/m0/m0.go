// Package m0 is milestone 0's proof app: Go draws into the native window
// inside the Cascades page, and key presses from both possible paths reach
// Go and are shown on screen. It answers two questions before the terminal is
// built on top: does the embedded window work through cgo, and do keys arrive
// better as Cascades key events or as raw Screen events forwarded to the
// window. It is replaced by the terminal in M1.
package m0

import (
	"fmt"
	"image"
	"log"
	"runtime"
	"sync"
	"time"

	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"term51/internal/ui"
)

// Command is a command from QML.
type Command struct {
	Name string
	Args map[string]any
}

// App owns the window on its own OS thread.
type App struct {
	cmds chan Command

	mu    sync.Mutex
	lines []string // newest last
}

func New() *App { return &App{cmds: make(chan Command, 64)} }

// Send queues a command from QML. It never blocks the UI thread: commands
// that don't fit are dropped and logged.
func (a *App) Send(c Command) {
	select {
	case a.cmds <- c:
	default:
		log.Printf("m0: command queue full, dropped %s", c.Name)
	}
}

func (a *App) logf(format string, args ...any) {
	s := fmt.Sprintf(format, args...)
	log.Print(s)
	a.mu.Lock()
	a.lines = append(a.lines, s)
	if len(a.lines) > 200 {
		a.lines = a.lines[len(a.lines)-200:]
	}
	a.mu.Unlock()
}

// Run waits for QML to ask for the window, then owns it until the app exits.
func (a *App) Run() {
	runtime.LockOSThread()
	var win *ui.Window
	for win == nil {
		c := <-a.cmds
		if c.Name != "window" {
			a.handle(nil, c)
			continue
		}
		group, _ := c.Args["group"].(string)
		id, _ := c.Args["id"].(string)
		w, h := num(c.Args["w"]), num(c.Args["h"])
		var err error
		win, err = ui.OpenWindow(group, id, w, h)
		if err != nil {
			a.logf("open window (group %q, id %q, %dx%d): %v", group, id, w, h, err)
			win = nil
			continue
		}
		a.logf("window open: %dx%d, group %q, id %q", w, h, group, id)
	}
	defer win.Close()

	frames := 0
	start := time.Now()
	dirty := true
	for {
		for drained := false; !drained; {
			select {
			case c := <-a.cmds:
				a.handle(win, c)
				dirty = true
			default:
				drained = true
			}
		}
		ev := win.NextEvent(15 * time.Millisecond)
		switch ev.Type {
		case ui.EventKey:
			a.logf("raw key: flags=%#x sym=%#x cap=%#x mods=%#x scan=%d",
				ev.KeyFlags, ev.KeySym, ev.KeyCap, ev.KeyMods, ev.KeyScan)
			dirty = true
		case ui.EventSize:
			a.logf("size event: %dx%d", ev.W, ev.H)
			if err := win.Resize(ev.W, ev.H); err != nil {
				a.logf("%v", err)
			}
			dirty = true
		case ui.EventTouch:
			if ev.Touch != 2 {
				a.logf("touch %d at %d,%d", ev.Touch, ev.X, ev.Y)
				dirty = true
			}
		case ui.EventClose:
			a.logf("window closed")
			return
		}
		if dirty {
			t0 := time.Now()
			if err := win.Frame(func(pix []uint32, stride int) { a.draw(pix, stride, win.Width, win.Height) }); err != nil {
				a.logf("%v", err)
			}
			frames++
			if frames%50 == 1 {
				log.Printf("frame %d: %v to draw, %.1f frames/s overall", frames, time.Since(t0),
					float64(frames)/time.Since(start).Seconds())
			}
			dirty = false
		}
	}
}

func (a *App) handle(win *ui.Window, c Command) {
	switch c.Name {
	case "size":
		w, h := num(c.Args["w"]), num(c.Args["h"])
		a.logf("size from QML: %dx%d", w, h)
		if win != nil {
			if err := win.Resize(w, h); err != nil {
				a.logf("%v", err)
			}
		}
	case "key":
		a.logf("cascades key: %v", c.Args)
	default:
		a.logf("command %s %v", c.Name, c.Args)
	}
}

func num(v any) int {
	f, _ := v.(float64)
	return int(f)
}

// The palette of the test pattern: the 16 ANSI colours (xterm defaults).
var ansi = [16]uint32{
	0x000000, 0xcd0000, 0x00cd00, 0xcdcd00, 0x0000ee, 0xcd00cd, 0x00cdcd, 0xe5e5e5,
	0x7f7f7f, 0xff0000, 0x00ff00, 0xffff00, 0x5c5cff, 0xff00ff, 0x00ffff, 0xffffff,
}

const scale = 3 // basicfont is 7x13; tripled for the Passport's ~450 ppi

func (a *App) draw(pix []uint32, stride, w, h int) {
	for y := 0; y < h; y++ {
		row := pix[y*stride : y*stride+w]
		for x := range row {
			row[x] = 0x101418
		}
	}
	// A bar of the 16 colours across the top.
	bar := 24
	for y := 0; y < bar && y < h; y++ {
		for x := 0; x < w; x++ {
			pix[y*stride+x] = ansi[x*16/w]
		}
	}
	lineH := 13 * scale
	y := bar + lineH
	a.text(pix, stride, w, h, 8, y, fmt.Sprintf("term51 M0  %dx%d", w, h), 0xffffff)
	y += lineH
	a.mu.Lock()
	lines := a.lines
	n := (h - y) / lineH
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	for _, s := range lines {
		y += lineH
		a.text(pix, stride, w, h, 8, y, s, 0xc0c0c0)
	}
	a.mu.Unlock()
}

// text draws s with its baseline at y, scaled by scale.
func (a *App) text(pix []uint32, stride, w, h, x, y int, s string, color uint32) {
	face := basicfont.Face7x13
	for _, r := range s {
		dr, mask, mp, adv, ok := face.Glyph(fixed.P(0, 0), r)
		if !ok {
			dr, mask, mp, adv, _ = face.Glyph(fixed.P(0, 0), '?')
		}
		for gy := dr.Min.Y; gy < dr.Max.Y; gy++ {
			for gx := dr.Min.X; gx < dr.Max.X; gx++ {
				_, _, _, al := mask.At(mp.X+gx-dr.Min.X, mp.Y+gy-dr.Min.Y).RGBA()
				if al < 0x8000 {
					continue
				}
				fill(pix, stride, w, h, image.Rect(x+gx*scale, y+gy*scale, x+(gx+1)*scale, y+(gy+1)*scale), color)
			}
		}
		x += adv.Round() * scale
		if x >= w {
			return
		}
	}
}

func fill(pix []uint32, stride, w, h int, r image.Rectangle, color uint32) {
	r = r.Intersect(image.Rect(0, 0, w, h))
	for y := r.Min.Y; y < r.Max.Y; y++ {
		row := pix[y*stride+r.Min.X : y*stride+r.Max.X]
		for i := range row {
			row[i] = color
		}
	}
}
