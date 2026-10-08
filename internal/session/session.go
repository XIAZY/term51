// Package session runs one terminal: a shell on a pseudo-terminal, the
// terminal state it writes into, and the way in for keys, pastes and
// resizes.
//
// One goroutine reads the shell's output and feeds it to the terminal
// library. The library never calls the UI: the UI is told that something
// changed (Changed) and then copies what it needs to draw (Snapshot). The
// queries programs send ("where is the cursor", "what terminal are you")
// are answered here, with package replies, by writing to the shell.
package session

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/XIAZY/libghostty-vt-go/vt"
	"github.com/XIAZY/libghostty-vt-go/vt/color"
	"github.com/XIAZY/libghostty-vt-go/vt/input"

	"term51/internal/pty"
	"term51/internal/replies"
)

// Config describes the session to start.
type Config struct {
	// Shell is the program to run, with its arguments and environment (nil
	// Env: the process's own). Dir is its working directory. TERM is set
	// to DefaultTerm unless Env names one: the app's own TERM, if it has
	// one, says nothing about this terminal.
	Shell string
	Args  []string
	Env   []string
	Dir   string

	// The terminal's size in cells and the size of one cell in pixels (the
	// page measures both before the first shell starts). A size of zero
	// columns or rows is the library's error.
	Cols, Rows   int
	CellW, CellH int

	// MaxScrollback is the scrollback limit in bytes, as the library
	// counts it.
	MaxScrollback uint64

	// Theme is the terminal's default colours.
	Theme Theme

	// Cursor is the cursor's shape while no program has asked for one.
	Cursor CursorShape

	// Replies configures the answers to programs' queries.
	Replies replies.Config
}

// RGB is a colour.
type RGB struct{ R, G, B uint8 }

// CursorShape is the shape of the cursor.
type CursorShape int

const (
	CursorBlock CursorShape = iota
	CursorBar
	CursorUnderline
)

func (c CursorShape) style() vt.CursorStyle {
	switch c {
	case CursorBar:
		return vt.CursorStyleBar
	case CursorUnderline:
		return vt.CursorStyleUnderline
	}
	return vt.CursorStyleBlock
}

// Theme is the colours the terminal starts with. The library leaves the
// default text and background colours unset, so an application has to give
// them; the zero Theme is DefaultTheme. (So a theme of black text on a black
// background with nothing else set cannot be asked for; nobody wants it.)
type Theme struct {
	Foreground, Background RGB
	// Cursor is the cursor's colour; nil draws it in the text colour.
	Cursor *RGB
	// Palette replaces the first colours of the 256-colour palette (the 16
	// ANSI colours, usually); nil keeps the library's.
	Palette []RGB
}

// Dark reports whether the theme is a dark one: whether its background is
// darker than middle grey. It is what a program asking for the colour scheme
// is told.
func (t Theme) Dark() bool {
	bg := t.Background
	if t.Foreground == (RGB{}) && bg == (RGB{}) && t.Cursor == nil && t.Palette == nil {
		bg = DefaultTheme.Background
	}
	// Perceived brightness, in thousandths of full white.
	return 299*int(bg.R)+587*int(bg.G)+114*int(bg.B) < 500*255
}

// DefaultTheme is light grey on near black.
var DefaultTheme = Theme{
	Foreground: RGB{0xD0, 0xD0, 0xD0},
	Background: RGB{0x10, 0x14, 0x18},
}

func (t Theme) colors() vt.TerminalColors {
	if t.Foreground == (RGB{}) && t.Background == (RGB{}) && t.Cursor == nil && t.Palette == nil {
		t = DefaultTheme
	}
	c := vt.TerminalColorsDefault()
	c.Foreground = color.NewDynamicRGB(color.RGB{R: t.Foreground.R, G: t.Foreground.G, B: t.Foreground.B})
	c.Background = color.NewDynamicRGB(color.RGB{R: t.Background.R, G: t.Background.G, B: t.Background.B})
	if t.Cursor != nil {
		c.Cursor = color.NewDynamicRGB(color.RGB{R: t.Cursor.R, G: t.Cursor.G, B: t.Cursor.B})
	}
	if len(t.Palette) > 0 {
		pal := c.Palette.Current
		for i, p := range t.Palette {
			if i >= len(pal) {
				break
			}
			pal[i] = color.RGB{R: p.R, G: p.G, B: p.B}
		}
		c.Palette = color.NewDynamicPalette(pal)
	}
	return c
}

// DefaultTerm is the TERM the shell gets when Config.Env does not set one:
// xterm, which the phone's own terminal database has (it has no
// xterm-256color).
const DefaultTerm = "xterm"

// feedChunk is how much output is fed to the terminal per hold of the lock,
// so that drawing (Snapshot) is never locked out for long: at the phone's
// 2 to 4 MB/s, 4 KB is 1 to 2 ms.
const feedChunk = 4 << 10

// How long Close waits for the shell after the hang-up signal before it
// kills the shell's process group, and how long it waits after that.
const (
	closeGrace = 500 * time.Millisecond
	closeLimit = 2 * time.Second
)

// EventKind is the kind of an Event.
type EventKind int

const (
	EventTitle     EventKind = iota // the window title changed; Text is the title
	EventBell                       // the bell rang
	EventPwd                        // the shell reported its directory; Text is the URL
	EventClipboard                  // a program set the clipboard; Text is the data as sent
	EventExit                       // the shell exited; Err is its exit error, if any (see Done)
)

// Event is something the UI may want to show or act on.
type Event struct {
	Kind EventKind
	Text string
	Err  error
}

// Session is one running terminal.
type Session struct {
	cfg    replies.Config
	cmd    *exec.Cmd
	master *os.File

	// mu guards the terminal, the stream, the sizes and closed.
	mu     sync.Mutex
	closed bool
	term   vt.Terminal
	stream vt.Stream[*handler]
	h      handler
	cellW  int
	cellH  int

	// wmu serialises writes to the shell (keys from the UI, answers from
	// the reader).
	wmu sync.Mutex

	changed      chan struct{}
	events       chan Event
	readerDone   chan struct{} // the reader saw the end of the shell's output
	done         chan struct{} // the shell has exited and been waited for
	exitErr      error         // set before done is closed
	cursor       CursorShape   // the user's shape
	cursorChosen bool          // a program asked for a shape (DECSCUSR)
	closeOnce    sync.Once
}

// Start starts the shell and the reader.
func Start(cfg Config) (*Session, error) {
	term, err := vt.NewTerminal(vt.TerminalOptions{
		Cols:          uint16(cfg.Cols),
		Rows:          uint16(cfg.Rows),
		MaxScrollback: cfg.MaxScrollback,
		Colors:        cfg.Theme.colors(),
		DefaultModes:  vt.NewModePacked(),
	})
	if err != nil {
		return nil, err
	}
	s := &Session{
		cfg:        cfg.Replies,
		term:       term,
		cellW:      cfg.CellW,
		cellH:      cfg.CellH,
		cursor:     cfg.Cursor,
		changed:    make(chan struct{}, 1),
		events:     make(chan Event, 64),
		readerDone: make(chan struct{}),
		done:       make(chan struct{}),
	}
	s.h = handler{s: s, ro: vt.NewReadonlyHandler(&s.term)}
	s.stream = vt.NewStream(&s.h)

	cmd := exec.Command(cfg.Shell, cfg.Args...)
	cmd.Env = withTerm(cfg.Env)
	cmd.Dir = cfg.Dir
	master, err := pty.Start(cmd, s.ptySize(cfg.Cols, cfg.Rows))
	if err != nil {
		s.term.Deinit()
		return nil, err
	}
	s.cmd, s.master = cmd, master
	go s.read()
	go s.wait()
	return s, nil
}

// withTerm returns env (nil: the process's own environment) with TERM set to
// DefaultTerm unless env itself names one.
func withTerm(env []string) []string {
	if env == nil {
		for _, kv := range os.Environ() {
			if !strings.HasPrefix(kv, "TERM=") {
				env = append(env, kv)
			}
		}
		return append(env, "TERM="+DefaultTerm)
	}
	for _, kv := range env {
		if strings.HasPrefix(kv, "TERM=") {
			return env
		}
	}
	return append(append([]string(nil), env...), "TERM="+DefaultTerm)
}

func (s *Session) ptySize(cols, rows int) pty.Size {
	return pty.Size{Rows: uint16(rows), Cols: uint16(cols), XPixel: uint16(cols * s.cellW), YPixel: uint16(rows * s.cellH)}
}

// read feeds the shell's output to the terminal until the shell's side of
// the pseudo-terminal closes.
func (s *Session) read() {
	defer close(s.readerDone)
	buf := make([]byte, 64<<10)
	for {
		n, err := s.master.Read(buf)
		for off := 0; off < n; off += feedChunk {
			end := min(off+feedChunk, n)
			if !s.feed(buf[off:end]) {
				return
			}
		}
		if n > 0 {
			s.signal()
		}
		if err != nil {
			return
		}
	}
}

// feed gives one piece of output to the terminal, then sends the answers to
// the queries in it and its events. It reports false after Close.
func (s *Session) feed(b []byte) bool {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return false
	}
	s.h.out.Reset()
	// An error is a sequence the library could not apply; the terminal
	// stays usable, so reading goes on.
	_ = s.stream.NextSlice(b)
	answers := append([]byte(nil), s.h.out.Bytes()...)
	evs := s.h.events
	s.h.events = nil
	s.mu.Unlock()

	if len(answers) > 0 {
		s.write(answers)
	}
	for _, ev := range evs {
		s.emit(ev)
	}
	return true
}

// wait waits for the shell itself. The end of its output is not a reliable
// sign of its exit: a background job can keep the pseudo-terminal open
// after the shell has gone. The reader gets a moment to take in the last
// output first.
func (s *Session) wait() {
	err := s.cmd.Wait()
	select {
	case <-s.readerDone:
	case <-time.After(300 * time.Millisecond):
	}
	s.exitErr = err
	close(s.done)
	s.emit(Event{Kind: EventExit, Err: err})
}

func (s *Session) signal() {
	select {
	case s.changed <- struct{}{}:
	default:
	}
}

// emit queues an event. When the UI is not reading and the queue is full the
// event is dropped, the exit event included: Done is the reliable sign of
// the shell's exit.
func (s *Session) emit(ev Event) {
	select {
	case s.events <- ev:
	default:
	}
}

func (s *Session) write(b []byte) {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	for len(b) > 0 {
		n, err := s.master.Write(b)
		if err != nil {
			return // the shell is gone; the reader reports it
		}
		b = b[n:]
	}
}

// Changed delivers a value after the terminal changed. Signals are merged:
// at most one is pending.
func (s *Session) Changed() <-chan struct{} { return s.changed }

// Events delivers titles, bells and the shell's exit. An event is dropped
// when 64 are waiting unread.
func (s *Session) Events() <-chan Event { return s.events }

// Done is closed after the shell has exited and been waited for.
func (s *Session) Done() <-chan struct{} { return s.done }

// ExitErr is the shell's exit error (nil for a clean exit). It is valid
// after Done is closed.
func (s *Session) ExitErr() error {
	select {
	case <-s.done:
		return s.exitErr
	default:
		return nil
	}
}

// Send writes bytes to the shell as they are.
func (s *Session) Send(b []byte) { s.write(b) }

// Key encodes a key event with the terminal's current modes and sends it.
func (s *Session) Key(ev input.KeyEvent) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrClosed
	}
	opts := input.KeyEncodeOptionsFromTerminal(&s.term)
	s.mu.Unlock()
	var b bytes.Buffer
	if err := input.EncodeKey(&b, ev, opts); err != nil {
		return err
	}
	if b.Len() > 0 {
		s.write(b.Bytes())
	}
	return nil
}

// Paste sends pasted text, bracketed when the program asked for that. The
// data is copied before it is changed.
func (s *Session) Paste(data []byte) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrClosed
	}
	bracketed := s.term.Modes.Get(vt.ModeBracketedPaste)
	s.mu.Unlock()
	parts := input.EncodePasteMutable(append([]byte(nil), data...), input.PasteOptions{Bracketed: bracketed})
	for _, p := range parts {
		if len(p) > 0 {
			s.write(p)
		}
	}
	return nil
}

// Resize changes the terminal's size and tells the shell.
func (s *Session) Resize(cols, rows, cellW, cellH int) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrClosed
	}
	s.cellW, s.cellH = cellW, cellH
	err := s.term.Resize(uint16(cols), uint16(rows))
	size := s.ptySize(cols, rows)
	s.mu.Unlock()
	if err != nil {
		return err
	}
	return pty.SetSize(s.master, size)
}

// SetTheme changes the terminal's default colours, for what is on screen
// too: the next frame is drawn with them. Colours a program has set since
// the start are replaced as well, and a program asking for the colour scheme
// is told the new theme's.
func (s *Session) SetTheme(t Theme) {
	s.mu.Lock()
	if !s.closed {
		s.term.Colors = t.colors()
		s.cfg.Dark = t.Dark()
	}
	s.mu.Unlock()
	s.signal()
}

// SetCursor changes the cursor's shape for as long as no program asks for
// one.
func (s *Session) SetCursor(c CursorShape) {
	s.mu.Lock()
	s.cursor = c
	s.mu.Unlock()
	s.signal()
}

// ScrollViewport moves the view into the scrollback by delta rows (negative:
// towards older output).
func (s *Session) ScrollViewport(delta int) {
	s.mu.Lock()
	if !s.closed {
		s.term.ScrollViewport(vt.TerminalScrollViewport{Tag: vt.TerminalScrollViewportDelta, Delta: int64(delta)})
	}
	s.mu.Unlock()
	s.signal()
}

// ScrollToBottom returns the view to the live screen.
func (s *Session) ScrollToBottom() {
	s.mu.Lock()
	if !s.closed {
		s.term.ScrollViewport(vt.TerminalScrollViewport{Tag: vt.TerminalScrollViewportBottom})
	}
	s.mu.Unlock()
	s.signal()
}

// Snapshot updates rs to the terminal's current state, for drawing. The
// caller owns rs and reuses it between frames.
func (s *Session) Snapshot(rs *vt.RenderState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	if err := rs.Update(&s.term); err != nil {
		return err
	}
	// The library's cursor is a block until a program asks for a shape;
	// until then it has the shape the user chose.
	if !s.cursorChosen && rs.Cursor.VisualStyle == vt.CursorStyleBlock {
		rs.Cursor.VisualStyle = s.cursor.style()
	}
	return nil
}

// Text returns the screen as plain text (for tests and for copying).
func (s *Session) Text() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return "", ErrClosed
	}
	b, err := s.term.PlainString()
	return string(b), err
}

// ErrClosed is returned by a Session that has been closed.
var ErrClosed = errors.New("session: closed")

// Close ends the session: it hangs up the shell, kills its process group
// if it has not gone after closeGrace, and releases the terminal. It does
// not wait longer than closeGrace + closeLimit. Calling it again does
// nothing.
func (s *Session) Close() error {
	s.closeOnce.Do(func() {
		// The shell's process group is its session (pty.Start).
		group := -s.cmd.Process.Pid
		_ = syscall.Kill(group, syscall.SIGHUP)
		select {
		case <-s.done:
		case <-time.After(closeGrace):
			_ = syscall.Kill(group, syscall.SIGKILL)
			select {
			case <-s.done:
			case <-time.After(closeLimit):
			}
		}

		// From here the reader feeds nothing more, whether or not it is
		// still blocked in a read (Go cannot interrupt a read on a
		// blocking descriptor; it ends when the last process holding the
		// pseudo-terminal goes).
		s.mu.Lock()
		s.closed = true
		s.stream.Deinit()
		s.term.Deinit()
		s.mu.Unlock()
		_ = s.master.Close()
	})
	return nil
}
