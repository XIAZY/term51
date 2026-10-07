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
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/XIAZY/libghostty-vt-go/vt"
	"github.com/XIAZY/libghostty-vt-go/vt/input"

	"term51/internal/pty"
	"term51/internal/replies"
)

// Config describes the session to start.
type Config struct {
	// Shell is the program to run, with its arguments and environment (nil
	// Env: the process's own). Dir is its working directory.
	Shell string
	Args  []string
	Env   []string
	Dir   string

	// The terminal's size in cells and the size of one cell in pixels
	// (zero when unknown).
	Cols, Rows   int
	CellW, CellH int

	// MaxScrollback is the scrollback limit in bytes, as the library
	// counts it.
	MaxScrollback uint64

	// Replies configures the answers to programs' queries.
	Replies replies.Config
}

// EventKind is the kind of an Event.
type EventKind int

const (
	EventTitle     EventKind = iota // the window title changed; Text is the title
	EventBell                       // the bell rang
	EventPwd                        // the shell reported its directory; Text is the URL
	EventClipboard                  // a program set the clipboard; Text is the data as sent
	EventExit                       // the shell exited; Err is its exit error, if any
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

	// mu guards the terminal, the stream and the sizes.
	mu     sync.Mutex
	term   vt.Terminal
	stream vt.Stream[*handler]
	h      handler
	cellW  int
	cellH  int

	// wmu serialises writes to the shell (keys from the UI, answers from
	// the reader).
	wmu sync.Mutex

	changed chan struct{}
	events  chan Event
	done    chan struct{}
}

// Start starts the shell and the reader.
func Start(cfg Config) (*Session, error) {
	if cfg.Cols <= 0 || cfg.Rows <= 0 || cfg.Cols > 0xFFFF || cfg.Rows > 0xFFFF {
		return nil, fmt.Errorf("session: bad size %dx%d", cfg.Cols, cfg.Rows)
	}
	term, err := vt.NewTerminal(vt.TerminalOptions{
		Cols:          uint16(cfg.Cols),
		Rows:          uint16(cfg.Rows),
		MaxScrollback: cfg.MaxScrollback,
		Colors:        vt.TerminalColorsDefault(),
		DefaultModes:  vt.NewModePacked(),
	})
	if err != nil {
		return nil, err
	}
	s := &Session{
		cfg:     cfg.Replies,
		term:    term,
		cellW:   cfg.CellW,
		cellH:   cfg.CellH,
		changed: make(chan struct{}, 1),
		events:  make(chan Event, 64),
		done:    make(chan struct{}),
	}
	s.h = handler{s: s, ro: vt.NewReadonlyHandler(&s.term)}
	s.stream = vt.StreamInitAlloc(&s.h)

	cmd := exec.Command(cfg.Shell, cfg.Args...)
	cmd.Env = cfg.Env
	cmd.Dir = cfg.Dir
	master, err := pty.Start(cmd, s.ptySize(cfg.Cols, cfg.Rows))
	if err != nil {
		s.term.Deinit()
		return nil, err
	}
	s.cmd, s.master = cmd, master
	go s.read()
	return s, nil
}

func (s *Session) ptySize(cols, rows int) pty.Size {
	clamp := func(v int) uint16 {
		if v > 0xFFFF {
			return 0xFFFF
		}
		return uint16(v)
	}
	return pty.Size{Rows: uint16(rows), Cols: uint16(cols), XPixel: clamp(cols * s.cellW), YPixel: clamp(rows * s.cellH)}
}

// read feeds the shell's output to the terminal until the shell's side of
// the pseudo-terminal closes, then waits for the shell.
func (s *Session) read() {
	buf := make([]byte, 64<<10)
	for {
		n, err := s.master.Read(buf)
		if n > 0 {
			s.mu.Lock()
			s.h.out.Reset()
			ferr := s.stream.NextSlice(buf[:n])
			answers := append([]byte(nil), s.h.out.Bytes()...)
			evs := s.h.events
			s.h.events = nil
			s.mu.Unlock()
			if ferr != nil {
				// The library reports a sequence it could not apply; the
				// terminal stays usable, so keep reading.
				_ = ferr
			}
			if len(answers) > 0 {
				s.write(answers)
			}
			for _, ev := range evs {
				s.emit(ev)
			}
			select {
			case s.changed <- struct{}{}:
			default:
			}
		}
		if err != nil {
			break
		}
	}
	werr := s.cmd.Wait()
	s.emit(Event{Kind: EventExit, Err: werr})
	close(s.done)
}

// emit queues an event; when the UI is not reading, the oldest kind of loss
// is acceptable, so a full queue drops the event.
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

// Events delivers titles, bells and the shell's exit.
func (s *Session) Events() <-chan Event { return s.events }

// Done is closed after the shell has exited and been waited for.
func (s *Session) Done() <-chan struct{} { return s.done }

// Send writes bytes to the shell as they are.
func (s *Session) Send(b []byte) { s.write(b) }

// Key encodes a key event with the terminal's current modes and sends it.
func (s *Session) Key(ev input.KeyEvent) error {
	s.mu.Lock()
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
	if cols <= 0 || rows <= 0 || cols > 0xFFFF || rows > 0xFFFF {
		return fmt.Errorf("session: bad size %dx%d", cols, rows)
	}
	s.mu.Lock()
	s.cellW, s.cellH = cellW, cellH
	err := s.term.Resize(uint16(cols), uint16(rows))
	size := s.ptySize(cols, rows)
	s.mu.Unlock()
	if err != nil {
		return err
	}
	return pty.SetSize(s.master, size)
}

// ScrollViewport moves the view into the scrollback by delta rows (negative:
// towards older output).
func (s *Session) ScrollViewport(delta int) {
	s.mu.Lock()
	s.term.ScrollViewport(vt.TerminalScrollViewport{Tag: vt.TerminalScrollViewportDelta, Delta: int64(delta)})
	s.mu.Unlock()
	select {
	case s.changed <- struct{}{}:
	default:
	}
}

// ScrollToBottom returns the view to the live screen.
func (s *Session) ScrollToBottom() {
	s.mu.Lock()
	s.term.ScrollViewport(vt.TerminalScrollViewport{Tag: vt.TerminalScrollViewportBottom})
	s.mu.Unlock()
	select {
	case s.changed <- struct{}{}:
	default:
	}
}

// Snapshot updates rs to the terminal's current state, for drawing. The
// caller owns rs and reuses it between frames.
func (s *Session) Snapshot(rs *vt.RenderState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return rs.Update(&s.term)
}

// Text returns the screen as plain text (for tests and for copying).
func (s *Session) Text() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := s.term.PlainString()
	return string(b), err
}

// Close hangs up the shell and waits for it.
func (s *Session) Close() error {
	if s.cmd.Process != nil {
		// The shell's process group is its session (pty.Start).
		_ = syscall.Kill(-s.cmd.Process.Pid, syscall.SIGHUP)
	}
	err := s.master.Close()
	<-s.done
	s.mu.Lock()
	s.stream.Deinit()
	s.term.Deinit()
	s.mu.Unlock()
	if errors.Is(err, os.ErrClosed) {
		return nil
	}
	return err
}

var _ io.Writer = (*bytes.Buffer)(nil)
