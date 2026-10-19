package terminal

import (
	"bytes"
	"errors"
	"testing"
)

// Go-only tests of the io.Writer forms (api_writer.go): each writes exactly
// what the internal function writes into a growing Zig-style writer.

func apiWriterTerminal(t *testing.T) *Terminal {
	t.Helper()
	term, err := TerminalInit(TerminalOptions{Cols: 20, Rows: 4, MaxScrollback: 10_000, Colors: TerminalColorsDefault, DefaultModes: ModesModePackedInit()})
	if err != nil {
		t.Fatal(err)
	}
	s := term.VtStream()
	if err := s.NextSlice([]byte("hello \x1b[1;31mworld\x1b[0m\r\nline 2 \xe4\xb8\xad\xe6\x96\x87")); err != nil {
		t.Fatal(err)
	}
	return &term
}

func TestApiWriter_formatters_match_the_internal_writer(t *testing.T) {
	term := apiWriterTerminal(t)
	defer term.Deinit()
	for _, opts := range []FormatterOptions{FormatterOptionsPlain, FormatterOptionsVt, FormatterOptionsHtml} {
		f := FormatterTerminalFormatterInit(term, opts)
		aw := oscWriterAllocatingInit()
		if err := f.format(&aw.writer); err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		if err := f.Format(&buf); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(buf.Bytes(), aw.written()) || buf.Len() == 0 {
			t.Errorf("emit %v: got %q, want %q", opts.Emit, buf.Bytes(), aw.written())
		}

		sf := FormatterScreenFormatterInit(term.Screens.Active, opts)
		aw = oscWriterAllocatingInit()
		if err := sf.format(&aw.writer); err != nil {
			t.Fatal(err)
		}
		buf.Reset()
		if err := sf.Format(&buf); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(buf.Bytes(), aw.written()) {
			t.Errorf("screen, emit %v: got %q, want %q", opts.Emit, buf.Bytes(), aw.written())
		}
	}
}

func TestApiWriter_DumpString(t *testing.T) {
	term := apiWriterTerminal(t)
	defer term.Deinit()
	var buf bytes.Buffer
	if err := term.Screens.Active.DumpString(&buf, ScreenDumpStringOptions{Tl: term.Screens.Active.Pages.Pin(Point{Tag: PointTagScreen}).Value}); err != nil {
		t.Fatal(err)
	}
	if want := "hello world\nline 2 中文"; buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestApiWriter_InputKeyEncodeEncode(t *testing.T) {
	var buf bytes.Buffer
	if err := InputKeyEncodeEncode(&buf, InputKeyKeyEvent{Action: InputKeyActionPress, Key: InputKeyKeyEscape}, InputKeyEncodeOptionsDefault); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "\x1b" {
		t.Errorf("escape: got %q", buf.String())
	}
}

type apiWriterFailing struct{ err error }

func (f apiWriterFailing) Write(p []byte) (int, error) { return 0, f.err }

// A failing io.Writer's error is returned.
func TestApiWriter_write_error_is_returned(t *testing.T) {
	term := apiWriterTerminal(t)
	defer term.Deinit()
	want := errors.New("disk full")
	f := FormatterTerminalFormatterInit(term, FormatterOptionsPlain)
	if err := f.Format(apiWriterFailing{want}); err != want {
		t.Errorf("got %v, want %v", err, want)
	}
}
