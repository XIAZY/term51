package app

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
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

	"term51/internal/busybox"
	"term51/internal/cells"
	"term51/internal/pwsh"
	"term51/internal/selbar"
	"term51/internal/settings"
)

// page stands in for the QML page: it keeps the merged state, the clipboard
// and the notices. The terminal itself is on the screen (scr).
type page struct {
	mu        sync.Mutex
	state     map[string]any
	clipboard string
	notices   []string
	pads      []string            // the arrow keys the app said it sent for the arrow pad
	saved     []settings.Settings // each time the app saved the settings
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
		Pad       *string `json:"pad"`
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
	if e.Pad != nil {
		p.pads = append(p.pads, *e.Pad)
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

func (s *screen) Frame(table []byte, lo, hi int) error {
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
	}, fonts, func(s settings.Settings) error {
		p.mu.Lock()
		defer p.mu.Unlock()
		p.saved = append(p.saved, s)
		return nil
	})
	a.PowerShell = &fakePowerShell{a: a, st: pwsh.State{Phase: pwsh.Absent}}
	done := make(chan error, 1)
	go func() { done <- a.Run() }()
	// The page's place for the terminal: 40 by 8 cells and a few pixels over.
	a.Send(Command{Name: "window", Args: map[string]any{"group": "g", "id": "term51", "w": float64(40*scr.cw + 3), "h": float64(8*scr.ch + 5)}})
	wait(t, "the first prompt", func() bool { return strings.Contains(p.screen(), "$") })
	return a, p, done
}

// fakePowerShell stands in for the downloadable PowerShell (pwsh.Store):
// each call succeeds at once, or fails with fail, and the app is told the
// new state as the store would tell it.
type fakePowerShell struct {
	a    *App
	mu   sync.Mutex
	st   pwsh.State
	fail string
	n    map[string]int
}

func (f *fakePowerShell) State() pwsh.State {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.st
}

func (f *fakePowerShell) to(call string, st pwsh.State) error {
	f.mu.Lock()
	if f.n == nil {
		f.n = map[string]int{}
	}
	f.n[call]++
	var err error
	if f.fail != "" {
		st, err = pwsh.State{Phase: pwsh.Failed, Reason: f.fail}, errors.New(f.fail)
	}
	f.st = st
	f.mu.Unlock()
	f.a.PowerShellChanged(st)
	return err
}

func (f *fakePowerShell) calls(call string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.n[call]
}

func (f *fakePowerShell) Download(context.Context) error {
	return f.to("download", pwsh.State{Phase: pwsh.Ready, Version: "7.6.6"})
}
func (f *fakePowerShell) Remove() error { return f.to("remove", pwsh.State{Phase: pwsh.Absent}) }
func (f *fakePowerShell) Check(context.Context) error {
	return f.to("check", pwsh.State{Phase: pwsh.Ready, Version: "7.6.6", Available: "7.6.7"})
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

	// The second tab will ring its bell in a moment, when it is no longer
	// the one shown.
	typeLine(a, "sleep 1; printf '\\a'")
	rang := func(i int) bool {
		r, _ := p.get("rang").([]any)
		return len(r) > i && r[i] == true
	}

	// Output in a tab that is not shown does not reach the page; switching
	// back shows that tab as it is now.
	a.Send(Command{Name: "tab", Args: map[string]any{"index": 0.0}})
	wait(t, "the first tab again", func() bool {
		s := p.screen()
		return strings.Contains(s, "in-first-tab") && !strings.Contains(s, "in-second-tab")
	})

	// The bell of the tab that is not shown marks it, until it is shown.
	wait(t, "the second tab marked as having rung", func() bool { return rang(1) && !rang(0) })
	a.Send(Command{Name: "tab", Args: map[string]any{"index": 1.0}})
	wait(t, "the mark gone", func() bool { return p.get("current") == 1.0 && !rang(1) })
	a.Send(Command{Name: "tab", Args: map[string]any{"index": 0.0}})
	wait(t, "the first tab once more", func() bool { return p.get("current") == 0.0 })

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

// A setting changed on the settings sheet is saved at once; the text size
// also changes what is on screen. A value that makes no sense changes
// nothing and says so.
func TestSettingsFromThePage(t *testing.T) {
	a, p, done := startApp(t)
	setting := func(name string) any {
		m, _ := p.get("settings").(map[string]any)
		return m[name]
	}
	lastSaved := func() (settings.Settings, int) {
		p.mu.Lock()
		defer p.mu.Unlock()
		if len(p.saved) == 0 {
			return settings.Settings{}, 0
		}
		return p.saved[len(p.saved)-1], len(p.saved)
	}
	wait(t, "the settings on the page", func() bool { return setting("fontPt") == "7" && setting("shell") == "/bin/bash" })
	if p.get("cols") != 40.0 {
		t.Fatalf("columns at the start: %v", p.get("cols"))
	}

	// Twice the text size: about half the columns in the same window (19:
	// the cell is 17 pixels wide, the window 323).
	a.Send(Command{Name: "set", Args: map[string]any{"name": "fontPt", "value": "14"}})
	wait(t, "the larger text", func() bool {
		s, n := lastSaved()
		cols, _ := p.scr.size()
		return n == 1 && s.FontPt == 14 && setting("fontPt") == "14" && p.get("cols") == 19.0 && cols == 19
	})

	a.Send(Command{Name: "set", Args: map[string]any{"name": "fontPt", "value": "400"}})
	wait(t, "the refusal", func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return len(p.notices) == 1 && strings.Contains(p.notices[0], "fontPt")
	})
	if _, n := lastSaved(); n != 1 || setting("fontPt") != "14" || p.get("cols") != 19.0 {
		t.Errorf("a refused value changed something: saved %d times, fontPt %v, %v columns", n, setting("fontPt"), p.get("cols"))
	}

	// The program is for tabs opened from now on.
	a.Send(Command{Name: "set", Args: map[string]any{"name": "shell", "value": "busybox"}})
	wait(t, "the shell saved", func() bool {
		s, n := lastSaved()
		return n == 2 && s.Shell == settings.ShellBusyBox && setting("shell") == "busybox"
	})

	// A theme and the cursor's shape change what is on screen: the last
	// cell (blank) gets the theme's background, and some cell the bar.
	lastCell := func() []byte {
		p.scr.mu.Lock()
		defer p.scr.mu.Unlock()
		if len(p.scr.table) < 16 {
			return make([]byte, 16)
		}
		return append([]byte(nil), p.scr.table[len(p.scr.table)-16:]...)
	}
	hasFlag := func(flag byte) bool {
		p.scr.mu.Lock()
		defer p.scr.mu.Unlock()
		for i := 0; i+16 <= len(p.scr.table); i += 16 {
			if p.scr.table[i+2]&flag != 0 {
				return true
			}
		}
		return false
	}
	a.Send(Command{Name: "set", Args: map[string]any{"name": "theme", "value": "light"}})
	wait(t, "the light theme on screen", func() bool {
		c := lastCell()
		return setting("theme") == "light" && c[8] == 0xfa && c[9] == 0xfa && c[10] == 0xf5
	})
	if hasFlag(cells.FlagCursorBar) {
		t.Error("a bar cursor before it was chosen")
	}
	a.Send(Command{Name: "set", Args: map[string]any{"name": "cursor", "value": "bar"}})
	wait(t, "the bar cursor on screen", func() bool { return hasFlag(cells.FlagCursorBar) })
	if s, n := lastSaved(); n != 4 || s.Cursor != "bar" || s.ThemeName() != "light" {
		t.Errorf("saved %d times; cursor %q, theme %q", n, s.Cursor, s.ThemeName())
	}

	a.Quit()
	if err := <-done; err != nil {
		t.Errorf("Run returned %v", err)
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

	// A press and hold on text that is already selected keeps the
	// selection (it only opens the menu): after Select all, Copy still
	// copies everything, not the one word under the finger.
	typeLine(a, "echo second-mark")
	wait(t, "the second line", func() bool { return strings.Contains(p.screen(), "second-mark") })
	a.Send(Command{Name: "selectall"})
	a.Send(Command{Name: "selectword", Args: map[string]any{"x": 0.0, "y": 0.0}})
	a.Send(Command{Name: "copy"})
	wait(t, "everything in the clipboard", func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return strings.Contains(p.clipboard, "copy-me") && strings.Contains(p.clipboard, "second-mark")
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

// Stretching a selection, end to end: the word a hold selected is stretched
// by a drag, the page is told that there is a selection, Copy takes it and
// leaves it shown as a finished one, and the next touch clears it. Held at
// the top edge, the screen scrolls and the selection runs into the
// scrollback. Select Line takes the whole line.
func TestStretchSelection(t *testing.T) {
	a, p, done := startApp(t)
	clipboard := func() string {
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.clipboard
	}
	typeLine(a, "clear; for i in $(seq 1 30); do echo line-$i end; done")
	wait(t, "the output", func() bool { return strings.Contains(p.screen(), "line-30 end") })
	// The view is 8 rows: lines 24 to 30 and the prompt. "line-30" is in row 6.
	if p.get("selected") != nil && p.get("selected") != "" {
		t.Fatalf("selected at the start: %v", p.get("selected"))
	}
	a.Send(Command{Name: "selectword", Args: map[string]any{"x": 2.0, "y": 6.0}})
	wait(t, "the page told of the selection", func() bool { return p.get("selected") == "open" })

	// A finger down to the right of the word, dragged to the end of the row.
	a.Send(Command{Name: "stretch", Args: map[string]any{"phase": "down", "x": 9.5, "y": 6.5}})
	a.Send(Command{Name: "stretch", Args: map[string]any{"phase": "move", "x": 10.5, "y": 6.5}})
	a.Send(Command{Name: "stretch", Args: map[string]any{"phase": "up", "x": 10.5, "y": 6.5}})
	a.Send(Command{Name: "copy"})
	wait(t, "the stretched selection copied", func() bool { return clipboard() == "line-30 end" })
	wait(t, "the selection finished but shown", func() bool { return p.get("selected") == "done" })
	a.Send(Command{Name: "unselect"})
	wait(t, "the selection gone", func() bool { return p.get("selected") == "" })

	// Select Line from a word.
	a.Send(Command{Name: "selectword", Args: map[string]any{"x": 9.0, "y": 5.0}}) // "end" of line-29
	a.Send(Command{Name: "selectline"})
	a.Send(Command{Name: "copy"})
	wait(t, "the line copied", func() bool { return clipboard() == "line-29 end" })
	a.Send(Command{Name: "unselect"})

	// The start of a word dragged up to the top edge and held there: the
	// screen scrolls back by itself and the selection grows with it.
	a.Send(Command{Name: "selectword", Args: map[string]any{"x": 2.0, "y": 6.0}})
	a.Send(Command{Name: "stretch", Args: map[string]any{"phase": "down", "x": 0.5, "y": 5.5}})
	a.Send(Command{Name: "stretch", Args: map[string]any{"phase": "move", "x": 0.5, "y": 0.2}})
	wait(t, "the screen scrolled back", func() bool { return strings.Contains(p.screen(), "line-15 end") })
	a.Send(Command{Name: "stretch", Args: map[string]any{"phase": "up", "x": 0.5, "y": 0.2}})
	a.Send(Command{Name: "copy"})
	wait(t, "the selection into the scrollback copied", func() bool {
		c := clipboard()
		return strings.Contains(c, "line-15 end\nline-16 end") && strings.HasSuffix(c, "line-29 end\nline-30")
	})
	// And it no longer scrolls once the finger has lifted. (The frame
	// that shows the selection as copied is let through first.)
	wait(t, "the selection finished", func() bool { return p.get("selected") == "done" })
	time.Sleep(100 * time.Millisecond)
	before := p.screen()
	time.Sleep(300 * time.Millisecond)
	if p.screen() != before {
		t.Error("the screen went on scrolling after the finger lifted")
	}
	a.Send(Command{Name: "unselect"})

	// A finger that slides off the terminal (onto the key bar below it)
	// selects to the last row, and held there the selection keeps growing
	// as the screen scrolls. The view is still scrolled back, so there is
	// text below it.
	wait(t, "the selection gone", func() bool { return p.get("selected") == "" })
	top := strings.Fields(p.screen())[0]
	if !strings.HasPrefix(top, "line-") || strings.Contains(p.screen(), "line-30") {
		t.Fatalf("the view is not scrolled back onto the lines:\n%s", p.screen())
	}
	a.Send(Command{Name: "selectword", Args: map[string]any{"x": 2.0, "y": 0.0}})
	a.Send(Command{Name: "stretch", Args: map[string]any{"phase": "down", "x": 9.5, "y": 0.5}})
	a.Send(Command{Name: "stretch", Args: map[string]any{"phase": "move", "x": 40.0, "y": 9.5}})
	wait(t, "the screen scrolled forwards", func() bool { return strings.Contains(p.screen(), "line-30 end") })
	a.Send(Command{Name: "stretch", Args: map[string]any{"phase": "up", "x": 40.0, "y": 9.5}})
	a.Send(Command{Name: "copy"})
	wait(t, "the selection grown past the bottom copied", func() bool {
		c := clipboard()
		return strings.HasPrefix(c, top+" end\n") && strings.Contains(c, "line-29 end\nline-30 end")
	})
	a.Quit()
	if err := <-done; err != nil {
		t.Errorf("Run returned %v", err)
	}
}

// The selection's bar of buttons: the page is told where it goes, by the
// rows of the view that have selected text in them (internal/selbar), and
// again whenever that changes.
func TestSelectionBar(t *testing.T) {
	a, p, done := startApp(t)
	// Room for the bar: 80 by 30 cells of 8 by 17 pixels.
	a.Send(Command{Name: "window", Args: map[string]any{"group": "g", "id": "term51", "w": 80.0 * 8, "h": 30.0 * 17}})
	wait(t, "the larger view", func() bool { c, r := p.scr.size(); return c == 80 && r == 30 })
	typeLine(a, "clear; for i in $(seq 1 40); do printf '%-40s%s\\n' line-$i end-$i; done")
	wait(t, "the output", func() bool { return strings.Contains(p.screen(), "end-40") })
	// Rows 0 to 28 are lines 12 to 40: line-32 is row 20, and its "end-32"
	// columns 40 to 45.
	bar := func(x, y int) func() bool {
		return func() bool { return p.get("selBarX") == float64(x) && p.get("selBarY") == float64(y) }
	}
	const above = 20*17 - selbar.Gap - selbar.Height
	const right = 80*8 - selbar.Width - selbar.Margin

	ends := func(sx, sy int, son bool, ex, ey int, eon bool) func() bool {
		return func() bool {
			return p.get("selStartOn") == son && (!son || p.get("selStartX") == float64(sx) && p.get("selStartY") == float64(sy)) &&
				p.get("selEndOn") == eon && (!eon || p.get("selEndX") == float64(ex) && p.get("selEndY") == float64(ey))
		}
	}

	// A word: the bar above it and centred over it, a handle at each end.
	a.Send(Command{Name: "selectword", Args: map[string]any{"x": 42.0, "y": 20.0}})
	wait(t, "the bar over the word", bar((40+46)*8/2-selbar.Width/2, above))
	wait(t, "the word's ends", ends(40, 20, true, 45, 20, true))

	// Stretched into the next row, the first row is selected from the word
	// to its end: the bar is over that part, and stays inside the right
	// side.
	a.Send(Command{Name: "stretch", Args: map[string]any{"phase": "down", "x": 46.5, "y": 20.5}})
	a.Send(Command{Name: "stretch", Args: map[string]any{"phase": "move", "x": 42.5, "y": 21.5}})
	a.Send(Command{Name: "stretch", Args: map[string]any{"phase": "up", "x": 42.5, "y": 21.5}})
	wait(t, "the bar over the first row's part", bar(right, above))
	wait(t, "the ends of two rows", ends(40, 20, true, 42, 21, true))

	// The start's handle dragged: the finger is taken to come down on the
	// start's own cell, so the start is the end that moves, and the other
	// stays.
	a.Send(Command{Name: "stretch", Args: map[string]any{"phase": "down", "x": 40.5, "y": 20.5}})
	a.Send(Command{Name: "stretch", Args: map[string]any{"phase": "move", "x": 2.5, "y": 19.5}})
	a.Send(Command{Name: "stretch", Args: map[string]any{"phase": "up", "x": 2.5, "y": 19.5}})
	wait(t, "the start moved, the end left", ends(2, 19, true, 42, 21, true))
	const left = (2+80)*8/2 - selbar.Width/2
	wait(t, "the bar over the new first row", bar(left, 19*17-selbar.Gap-selbar.Height))

	// Scrolled back nine rows, the selection's last row is below the view,
	// and its end with it; one more and only its first row is left; one
	// more and none of it.
	a.Send(Command{Name: "scroll", Args: map[string]any{"rows": -9.0}})
	wait(t, "the end out of the view", ends(2, 28, true, 0, 0, false))
	wait(t, "the bar still over the first row", bar(left, 28*17-selbar.Gap-selbar.Height))
	a.Send(Command{Name: "scroll", Args: map[string]any{"rows": -1.0}})
	wait(t, "the bar over the row left in the view", bar(left, 29*17-selbar.Gap-selbar.Height))
	a.Send(Command{Name: "scroll", Args: map[string]any{"rows": -1.0}})
	wait(t, "the bar at the top", bar(80*8/2-selbar.Width/2, selbar.Gap))
	wait(t, "both ends out of the view", ends(0, 0, false, 0, 0, false))

	a.Send(Command{Name: "unselect"})
	wait(t, "no selection", func() bool { return p.get("selected") == "" })
	a.Quit()
	if err := <-done; err != nil {
		t.Errorf("Run returned %v", err)
	}
}

// The arrow pad: after a press and hold, the finger past the threshold sends
// that arrow key once at once, repeats it while it stays out there, and
// stops when it lifts. The page is told of each key.
func TestArrowPad(t *testing.T) {
	a, p, done := startApp(t)
	pads := func() []string {
		p.mu.Lock()
		defer p.mu.Unlock()
		return append([]string(nil), p.pads...)
	}
	a.Send(Command{Name: "text", Args: map[string]any{"text": "echo abcdefgh"}})
	wait(t, "the typed line", func() bool { return strings.Contains(p.screen(), "echo abcdefgh") })

	// Not far enough: nothing. Then past the threshold to the left: one
	// Left, so that an X typed now lands before the h.
	a.Send(Command{Name: "pad", Args: map[string]any{"phase": "hold"}})
	a.Send(Command{Name: "pad", Args: map[string]any{"phase": "move", "dx": -40.0, "dy": 3.0}})
	a.Send(Command{Name: "pad", Args: map[string]any{"phase": "move", "dx": -70.0, "dy": 3.0}})
	a.Send(Command{Name: "pad", Args: map[string]any{"phase": "lift"}})
	a.Send(Command{Name: "text", Args: map[string]any{"text": "X"}})
	wait(t, "the X before the h", func() bool { return strings.Contains(p.screen(), "echo abcdefgXh") })
	if got := pads(); len(got) != 1 || got[0] != "left" {
		t.Errorf("the page was told %v, want one left", got)
	}

	// Held out there, it repeats until the finger lifts: several more
	// Lefts, and none after the lift.
	a.Send(Command{Name: "pad", Args: map[string]any{"phase": "hold"}})
	a.Send(Command{Name: "pad", Args: map[string]any{"phase": "move", "dx": -200.0, "dy": 0.0}})
	wait(t, "the repeats", func() bool { return len(pads()) >= 5 })
	a.Send(Command{Name: "pad", Args: map[string]any{"phase": "lift"}})
	time.Sleep(50 * time.Millisecond) // the loop takes the lift
	n := len(pads())
	time.Sleep(300 * time.Millisecond)
	if after := len(pads()); after != n {
		t.Errorf("%d keys were sent after the finger lifted", after-n)
	}
	a.Quit()
	if err := <-done; err != nil {
		t.Errorf("Run returned %v", err)
	}
}

// A character cell of the key bar types its character like one from the
// input field; with Ctrl armed it sends the control character and Ctrl is
// no longer armed.
func TestBarCharacters(t *testing.T) {
	a, p, done := startApp(t)
	typeLine(a, "cat -v")
	// What a "$" cell sends, then Ctrl and what a "_" cell sends: cat shows
	// the dollar as it is and the control character as ^_.
	a.Send(Command{Name: "text", Args: map[string]any{"text": "$"}})
	a.Send(Command{Name: "bar", Args: map[string]any{"name": "ctrl"}})
	wait(t, "Ctrl shown as armed", func() bool { return p.get("ctrl") == true })
	a.Send(Command{Name: "text", Args: map[string]any{"text": "_"}})
	wait(t, "Ctrl used up", func() bool { return p.get("ctrl") == false })
	a.Send(Command{Name: "key", Args: map[string]any{"down": true, "key": 13.0, "keycap": 13.0}})
	wait(t, "cat's echo of both", func() bool { return strings.Contains(p.screen(), "$^_") })
	a.Quit()
	if err := <-done; err != nil {
		t.Errorf("Run returned %v", err)
	}
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

// The downloadable PowerShell, from the settings sheet: the page is told
// where it is; it cannot be chosen as the shell, and no tab opens with it,
// until it is downloaded; a download that does not work is a notice with
// the reason.
func TestPowerShellFromThePage(t *testing.T) {
	a, p, done := startApp(t)
	// bash stands in for pwsh, at the place the download would put it.
	a.Bundled.PowerShell = "/bin/bash"
	fake := a.PowerShell.(*fakePowerShell)
	told := func(key string) any {
		m, _ := p.get("pwsh").(map[string]any)
		return m[key]
	}
	notices := func() []string {
		p.mu.Lock()
		defer p.mu.Unlock()
		return append([]string(nil), p.notices...)
	}
	setting := func(name string) any {
		m, _ := p.get("settings").(map[string]any)
		return m[name]
	}
	wait(t, "the page told there is none", func() bool { return told("phase") == "absent" })

	// Not downloaded: the shell cannot be chosen.
	a.Send(Command{Name: "set", Args: map[string]any{"name": "shell", "value": "powershell"}})
	wait(t, "the refusal", func() bool { n := notices(); return len(n) == 1 && n[0] == noPowerShell })
	if setting("shell") == "powershell" {
		t.Error("the PowerShell shell was set although it is not downloaded")
	}

	// A download that does not work: the reason is shown.
	fake.fail = "the list of releases could not be read"
	a.Send(Command{Name: "pwsh", Args: map[string]any{"action": "download"}})
	wait(t, "the failure told", func() bool {
		n := notices()
		return told("phase") == "failed" && len(n) == 2 && n[1] == "PowerShell could not be downloaded: the list of releases could not be read"
	})

	// One that works: ready, with its version; now it can be chosen, and
	// a new tab runs it.
	fake.fail = ""
	a.Send(Command{Name: "pwsh", Args: map[string]any{"action": "download"}})
	wait(t, "ready", func() bool { return told("phase") == "ready" && told("version") == "7.6.6" })
	a.Send(Command{Name: "set", Args: map[string]any{"name": "shell", "value": "powershell"}})
	wait(t, "the shell set", func() bool { return setting("shell") == "powershell" })
	a.Send(Command{Name: "newtab"})
	wait(t, "a PowerShell tab", func() bool {
		tabs, _ := p.get("tabs").([]any)
		return len(tabs) == 2 && tabs[1] == "bash" // the program the test runs as pwsh
	})

	// The sheet being opened asks for a newer release.
	a.Send(Command{Name: "pwsh", Args: map[string]any{"action": "check"}})
	wait(t, "a newer release told", func() bool { return told("available") == "7.6.7" })

	// Removed: the setting stays, but no tab opens with it.
	a.Send(Command{Name: "pwsh", Args: map[string]any{"action": "remove"}})
	wait(t, "removed", func() bool { return told("phase") == "absent" && fake.calls("remove") == 1 })
	a.Send(Command{Name: "newtab"})
	wait(t, "the second refusal", func() bool { n := notices(); return len(n) == 3 && n[2] == noPowerShell })
	if tabs, _ := p.get("tabs").([]any); len(tabs) != 2 {
		t.Errorf("%d tabs, want the two there were", len(tabs))
	}
	a.Quit()
	if err := <-done; err != nil {
		t.Errorf("Run returned %v", err)
	}
}

// The BusyBox shell: with the shell setting on it, a new tab runs the ash
// of the tools folder, and that folder is at the end of its PATH. The
// machine's own BusyBox stands in for the one in the package.
func TestBusyBoxShell(t *testing.T) {
	program, err := exec.LookPath("busybox")
	if err != nil {
		t.Skip("no busybox on this machine")
	}
	tools := filepath.Join(t.TempDir(), "t51tools")
	if err := busybox.Links(program, tools); err != nil {
		t.Fatal(err)
	}
	a, p, done := startApp(t)
	a.Bundled.Tools = tools

	a.Send(Command{Name: "set", Args: map[string]any{"name": "shell", "value": "busybox"}})
	a.Send(Command{Name: "newtab"})
	wait(t, "the second tab", func() bool {
		tabs, _ := p.get("tabs").([]any)
		return len(tabs) == 2 && tabs[0] == "bash" && tabs[1] == "ash"
	})
	// What runs is BusyBox's ash (its help names it), found through the
	// folder that is last on the PATH.
	a.Send(Command{Name: "text", Args: map[string]any{"text": "clear; ash --help 2>&1 | head -n 1; echo ${PATH##*/} | tr a-z A-Z"}})
	a.Send(Command{Name: "key", Args: map[string]any{"down": true, "key": 13.0, "keycap": 13.0}})
	wait(t, "the BusyBox shell's answer", func() bool {
		s := p.screen()
		return strings.Contains(s, "BusyBox v") && strings.Contains(s, "\nT51TOOLS")
	})
	a.Quit()
	if err := <-done; err != nil {
		t.Errorf("Run returned %v", err)
	}
}

// Without the tools folder (the links could not be made at the start), the
// BusyBox shell does not start: no tab is opened, and the page is told why.
func TestBusyBoxShellWithoutItsTools(t *testing.T) {
	a, p, done := startApp(t)
	a.Send(Command{Name: "set", Args: map[string]any{"name": "shell", "value": "busybox"}})
	a.Send(Command{Name: "newtab"})
	wait(t, "the notice", func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return len(p.notices) == 1 && p.notices[0] == "The BusyBox shell could not be started: its tools could not be set up (see the log)"
	})
	if tabs, _ := p.get("tabs").([]any); len(tabs) != 1 {
		t.Errorf("%d tabs, want the one there was", len(tabs))
	}
	a.Quit()
	if err := <-done; err != nil {
		t.Errorf("Run returned %v", err)
	}
}
