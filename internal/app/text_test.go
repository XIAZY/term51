package app

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/XIAZY/libghostty-vt-go/vt"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonobolditalic"
	"golang.org/x/image/font/gofont/gomonoitalic"
	"golang.org/x/image/font/opentype"

	"term51/internal/cells"
	"term51/internal/settings"
)

// page stands in for the QML page: it keeps the merged state, the clipboard
// and the notices. The terminal itself is on the screen (scr).
type page struct {
	mu        sync.Mutex
	state     map[string]any
	clipboard string
	notices   []string
	scr       *screen
}

func (p *page) State(js string) {
	var m map[string]any
	if json.Unmarshal([]byte(js), &m) != nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for k, v := range m {
		p.state[k] = v
	}
}

func (p *page) Event(js string) {
	var e struct {
		Clipboard *string `json:"clipboard"`
		Notice    *string `json:"notice"`
	}
	if json.Unmarshal([]byte(js), &e) != nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if e.Clipboard != nil {
		p.clipboard = *e.Clipboard
	}
	if e.Notice != nil {
		p.notices = append(p.notices, *e.Notice)
	}
}

// screen is what the page shows of the terminal: the text on the screen.
func (p *page) screen() string { return p.scr.text() }

// screen stands in for the window: it keeps what the app hands over, the
// character sheet and the cell table, exactly as the GPU would hold them.
type screen struct {
	mu         sync.Mutex
	w, h       int
	cols, rows int
	cw, ch     int
	sheet      []byte
	table      []byte
	frames     int
	resizes    int
	shapes     map[string]rune // a slot's pixels -> the character, for text
}

// testDPI makes the default 7 point font 14 pixels.
const testDPI = 144

func testFonts(t *testing.T) cells.Fonts {
	t.Helper()
	parse := func(ttf []byte) *opentype.Font {
		f, err := opentype.Parse(ttf)
		if err != nil {
			t.Fatal(err)
		}
		return f
	}
	return cells.Fonts{Faces: [4]*opentype.Font{parse(gomono.TTF), parse(gomonobold.TTF), parse(gomonoitalic.TTF), parse(gomonobolditalic.TTF)}}
}

// newScreen makes the stand-in. To read text back from what it is handed,
// it learns what each printable ASCII character looks like: it draws them
// through a sheet and table of its own, with the same fonts and size as the
// app's, and remembers each one's pixels.
func newScreen(t *testing.T, fonts cells.Fonts, sizePx float64) *screen {
	t.Helper()
	ref, err := cells.NewSheet(fonts, sizePx)
	if err != nil {
		t.Fatal(err)
	}
	s := &screen{sheet: make([]byte, cells.SheetSize*cells.SheetSize), shapes: map[string]rune{}}
	s.cw, s.ch = ref.CellSize()

	var printable []byte
	for c := byte('!'); c <= '~'; c++ {
		printable = append(printable, c)
	}
	term, err := vt.NewTerminal(vt.TerminalOptions{Cols: 100, Rows: 1, Colors: vt.TerminalColorsDefault(), DefaultModes: vt.NewModePacked()})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Deinit()
	stream := term.VtStream()
	if err := stream.NextSlice(printable); err != nil {
		t.Fatal(err)
	}
	rs := vt.RenderStateEmpty()
	if err := rs.Update(&term); err != nil {
		t.Fatal(err)
	}
	var table cells.Table
	table.Update(&rs, ref)
	for i, c := range printable {
		slot := int(table.Data[i*cells.RecordSize]) | int(table.Data[i*cells.RecordSize+1])<<8
		s.shapes[slotPixels(ref.Pix, slot, ref.PerRow, s.cw, s.ch)] = rune(c)
	}
	return s
}

// slotPixels is the pixels of one slot of a sheet.
func slotPixels(sheet []byte, slot, perRow, cw, ch int) string {
	x0, y0 := slot%perRow*cw, slot/perRow*ch
	var b []byte
	for y := y0; y < y0+ch; y++ {
		b = append(b, sheet[y*cells.SheetSize+x0:y*cells.SheetSize+x0+cw]...)
	}
	return string(b)
}

func (s *screen) Resize(w, h int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.w, s.h = w, h
	s.resizes++
	return nil
}

func (s *screen) DPI() int { return testDPI }

func (s *screen) Sizes() (window, buffer, source [2]int) { return }

func (s *screen) Grid(cols, rows, cellW, cellH int, r, g, b uint8) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cols, s.rows, s.cw, s.ch = cols, rows, cellW, cellH
	s.table = make([]byte, cols*rows*cells.RecordSize) // new storage, as on the GPU
}

func (s *screen) Sheet(sheet []byte, x, y, w, h int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for row := y; row < y+h; row++ {
		copy(s.sheet[row*cells.SheetSize+x:row*cells.SheetSize+x+w], sheet[row*cells.SheetSize+x:])
	}
}

func (s *screen) Frame(table []byte, lo, hi int, shot []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copy(s.table[lo:hi], table[lo:hi])
	s.frames++
	return nil
}

func (s *screen) Close() {}

// size is the grid as last set.
func (s *screen) size() (cols, rows int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cols, s.rows
}

// text reads the screen: each cell's slot is looked up in the sheet the
// screen was handed, and its pixels matched with the characters it knows.
// A shape it does not know reads as "?".
func (s *screen) text() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var sb strings.Builder
	for y := 0; y < s.rows; y++ {
		for x := 0; x < s.cols; x++ {
			rec := s.table[(y*s.cols+x)*cells.RecordSize:]
			slot := int(rec[0]) | int(rec[1])<<8
			if slot == 0 {
				sb.WriteByte(' ')
				continue
			}
			if c, ok := s.shapes[slotPixels(s.sheet, slot, cells.SheetSize/s.cw, s.cw, s.ch)]; ok {
				sb.WriteRune(c)
			} else {
				sb.WriteByte('?')
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func (p *page) get(key string) any {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state[key]
}

var debugPage *page

func wait(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			if debugPage != nil {
				t.Logf("screen:\n%s\nstate: %v", debugPage.screen(), debugPage.state)
			}
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func startApp(t *testing.T) (*App, *page, chan error) {
	t.Helper()
	cfg := settings.Default()
	cfg.Shell, cfg.Args = "/bin/bash", []string{"--norc", "--noprofile"}
	fonts := testFonts(t)
	scr := newScreen(t, fonts, cfg.FontPt*testDPI/72)
	p := &page{state: map[string]any{}, scr: scr}
	debugPage = p
	a := New(cfg, p, func(group, id string, w, h int) (Screen, error) {
		scr.Resize(w, h)
		return scr, nil
	}, fonts)
	done := make(chan error, 1)
	go func() { done <- a.Run() }()
	// The page's place for the terminal: 40 by 8 cells and a few pixels over.
	a.Send(Command{Name: "window", Args: map[string]any{"group": "g", "id": "term51", "w": float64(40*scr.cw + 3), "h": float64(8*scr.ch + 5)}})
	wait(t, "the first prompt", func() bool { return strings.Contains(p.screen(), "$") })
	return a, p, done
}

// typeLine pastes a line and presses Enter. (The Enter is a key: bash turns
// bracketed paste on, and a pasted newline is then text, not Enter.)
func typeLine(a *App, s string) {
	a.Send(Command{Name: "paste", Args: map[string]any{"text": s}})
	a.Send(Command{Name: "key", Args: map[string]any{"down": true, "key": 13.0, "keycap": 13.0}})
}

func TestTextFrontEndShowsTheShell(t *testing.T) {
	a, p, done := startApp(t)
	typeLine(a, "echo one-$((40+2))")
	wait(t, "the command's output", func() bool { return strings.Contains(p.screen(), "one-42") })
	if cols, rows := p.scr.size(); cols != 40 || rows != 8 {
		t.Errorf("the screen is %d by %d cells, want 40 by 8", cols, rows)
	}

	// The key bar: sticky Ctrl, then a "d" from the keyboard, ends the shell,
	// and with the last shell the app.
	a.Send(Command{Name: "bar", Args: map[string]any{"name": "ctrl"}})
	wait(t, "Ctrl shown as armed", func() bool { return p.get("ctrl") == true })
	a.Send(Command{Name: "key", Args: map[string]any{"down": true, "key": float64('d'), "keycap": float64('d'), "unicode": "d"}})
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the app did not end when its last shell exited")
	}
	if p.get("ctrl") != false {
		t.Error("Ctrl is still shown as armed after the key")
	}
}

func TestTabs(t *testing.T) {
	a, p, done := startApp(t)
	typeLine(a, "echo in-first-tab")
	wait(t, "the first tab's output", func() bool { return strings.Contains(p.screen(), "in-first-tab") })

	a.Send(Command{Name: "newtab"})
	wait(t, "a second tab", func() bool {
		tabs, _ := p.get("tabs").([]any)
		return len(tabs) == 2 && p.get("current") == 1.0
	})
	wait(t, "the new tab's empty screen", func() bool { return !strings.Contains(p.screen(), "in-first-tab") })
	typeLine(a, "echo in-second-tab")
	wait(t, "the second tab's output", func() bool { return strings.Contains(p.screen(), "in-second-tab") })

	// Output in a tab that is not shown does not reach the page; switching
	// back shows that tab as it is now.
	a.Send(Command{Name: "tab", Args: map[string]any{"index": 0.0}})
	wait(t, "the first tab again", func() bool {
		s := p.screen()
		return strings.Contains(s, "in-first-tab") && !strings.Contains(s, "in-second-tab")
	})

	// Closing the shown tab shows the other one.
	a.Send(Command{Name: "closetab"})
	wait(t, "one tab left", func() bool {
		tabs, _ := p.get("tabs").([]any)
		return len(tabs) == 1 && p.get("current") == 0.0
	})
	wait(t, "the remaining tab's screen", func() bool { return strings.Contains(p.screen(), "in-second-tab") })

	// Its shell exiting ends the app.
	typeLine(a, "exit")
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the app did not end when its last shell exited")
	}
}

func TestCopyAndResize(t *testing.T) {
	a, p, done := startApp(t)
	typeLine(a, "echo copy-me")
	wait(t, "output", func() bool { return strings.Contains(p.screen(), "copy-me") })

	// Copy with nothing selected copies nothing and says so.
	a.Send(Command{Name: "copy"})
	wait(t, "the notice", func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return len(p.notices) == 1 && p.clipboard == ""
	})
	a.Send(Command{Name: "selectall"})
	a.Send(Command{Name: "copy"})
	wait(t, "the clipboard", func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return strings.Contains(p.clipboard, "copy-me")
	})

	// The sizes of one slide of the on-screen rows, close together, are
	// one resize: to the last of them.
	p.scr.mu.Lock()
	before := p.scr.resizes
	p.scr.mu.Unlock()
	for _, rows := range []int{9, 7, 5} {
		a.Send(Command{Name: "size", Args: map[string]any{"w": float64(30 * p.scr.cw), "h": float64(rows * p.scr.ch)}})
	}
	wait(t, "the resized screen", func() bool {
		cols, rows := p.scr.size()
		return cols == 30 && rows == 5 && p.get("cols") == 30.0
	})
	p.scr.mu.Lock()
	if n := p.scr.resizes - before; n != 1 {
		t.Errorf("three sizes close together resized the screen %d times, want once", n)
	}
	p.scr.mu.Unlock()
	typeLine(a, "stty size")
	wait(t, "the shell's new size", func() bool { return strings.Contains(p.screen(), "5 30") })

	// Quit (the user closing the app) hangs up every shell and ends the
	// loop, also with a shell that ignores the hang-up.
	a.Send(Command{Name: "newtab"})
	wait(t, "a second tab", func() bool { return p.get("current") == 1.0 })
	typeLine(a, "trap '' HUP; echo armed")
	wait(t, "the trap", func() bool { return strings.Contains(p.screen(), "armed") })
	began := time.Now()
	a.Quit()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run returned %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Quit did not end the loop")
	}
	if d := time.Since(began); d > 5*time.Second {
		t.Errorf("Quit took %v", d)
	}
}

// A long selection is cut to its end before it goes to the page.
func TestCopyIsCapped(t *testing.T) {
	a, p, _ := startApp(t)
	typeLine(a, "head -c 700000 /dev/zero | tr '\\0' 'x' | fold -w 39; echo THE-END")
	wait(t, "the output", func() bool { return strings.Contains(p.screen(), "THE-END") })
	a.Send(Command{Name: "selectall"})
	a.Send(Command{Name: "copy"})
	wait(t, "the clipboard", func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.clipboard != ""
	})
	p.mu.Lock()
	n, end, notices := len(p.clipboard), strings.Contains(p.clipboard, "THE-END"), len(p.notices)
	p.mu.Unlock()
	if n > copyLimit || n < copyLimit/2 || !end || notices != 1 {
		t.Errorf("copied %d bytes (limit %d), end kept: %v, notices: %d", n, copyLimit, end, notices)
	}
	a.Quit()
}

// Characters typed into the page's input field arrive as "text"; the key
// bar's sticky Ctrl applies to the next of them.
func TestTextFromTheInputField(t *testing.T) {
	a, p, done := startApp(t)
	a.Send(Command{Name: "text", Args: map[string]any{"text": "echo $((6*7))-typed"}})
	a.Send(Command{Name: "key", Args: map[string]any{"down": true, "key": 13.0, "keycap": 13.0}})
	wait(t, "the output", func() bool { return strings.Contains(p.screen(), "42-typed") })

	// Sticky Ctrl, then "d" from the input field: the shell ends, and with
	// it the app.
	a.Send(Command{Name: "bar", Args: map[string]any{"name": "ctrl"}})
	wait(t, "Ctrl shown as armed", func() bool { return p.get("ctrl") == true })
	a.Send(Command{Name: "text", Args: map[string]any{"text": "d"}})
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Ctrl-D from the input field did not end the shell")
	}
}
