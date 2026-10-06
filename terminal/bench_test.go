package terminal

// Throughput benchmarks: the baseline for docs/post-port-optimizations.md.
//
// Two paths, three inputs each:
//
//   - BenchmarkStream*: the parser and stream with a handler that does
//     nothing, so only parsing and dispatch are measured;
//   - BenchmarkTerminal*: the same bytes through a real Terminal (80x24,
//     the default scrollback), which is what an application pays for.
//
// Inputs: plain ASCII lines; coloured text (SGR sequences); and a mixed
// stream with cursor movement, erases and CJK wide characters. Each is about
// 128 KB. b.SetBytes makes `go test -bench` report MB/s.
//
// The page self-checks (pageSlowRuntimeSafety) are switched off for the
// Terminal benchmarks: TestMain switches them on for the tests, but a release
// build of the library runs without them, and that is what is measured.
//
//	go test ./terminal -run '^$' -bench . -benchtime 2s

import (
	"bytes"
	"testing"
)

// benchNopHandler is a stream handler that does nothing.
type benchNopHandler struct{ n int }

func (h *benchNopHandler) Vt(StreamActionKey, StreamAction) error { h.n++; return nil }
func (h *benchNopHandler) Deinit()                                {}

const benchInputSize = 128 << 10

func benchRepeat(line string) []byte {
	var b bytes.Buffer
	for b.Len() < benchInputSize {
		b.WriteString(line)
	}
	return b.Bytes()
}

// benchInputASCII is plain text lines.
func benchInputASCII() []byte {
	return benchRepeat("The quick brown fox jumps over the lazy dog 0123456789 abcdefghij\r\n")
}

// benchInputSGR is coloured text, as `ls --color` or a compiler prints.
func benchInputSGR() []byte {
	return benchRepeat("\x1b[1;32mhello world \x1b[0m some plain text here \x1b[38;5;208m1234567890\x1b[0m \x1b[48;2;10;20;30mrgb\x1b[0m\r\n")
}

// benchInputMixed is a full-screen program's output: cursor addressing,
// erases, scrolling, and CJK wide characters.
func benchInputMixed() []byte {
	return benchRepeat("\x1b[H\x1b[2J\x1b[5;10Hstatus: \x1b[7m OK \x1b[0m\x1b[K" +
		"\x1b[7;1H日本語のテキスト wide text 中文\x1b[K" +
		"\x1b[10;1Hline one\r\nline two\r\n\x1b[3A\x1b[10C\x1b[1mbold\x1b[0m\x1b[B\x1b[4D...." +
		"\x1b[24;1H\x1b[S\x1b[1;24r\x1b[r\r\n")
}

func benchStream(b *testing.B, in []byte) {
	h := &benchNopHandler{}
	s := StreamInit(h)
	defer s.Deinit()
	b.SetBytes(int64(len(in)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := s.NextSlice(in); err != nil {
			b.Fatal(err)
		}
	}
}

func benchTerminal(b *testing.B, in []byte) {
	old := pageSlowRuntimeSafety
	pageSlowRuntimeSafety = false
	defer func() { pageSlowRuntimeSafety = old }()

	t, err := TerminalInit(terminalOptionsDefault(80, 24))
	if err != nil {
		b.Fatal(err)
	}
	defer t.Deinit()
	s := t.VtStream()
	defer s.Deinit()
	b.SetBytes(int64(len(in)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := s.NextSlice(in); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStreamASCII(b *testing.B) { benchStream(b, benchInputASCII()) }
func BenchmarkStreamSGR(b *testing.B)   { benchStream(b, benchInputSGR()) }
func BenchmarkStreamMixed(b *testing.B) { benchStream(b, benchInputMixed()) }

func BenchmarkTerminalASCII(b *testing.B) { benchTerminal(b, benchInputASCII()) }
func BenchmarkTerminalSGR(b *testing.B)   { benchTerminal(b, benchInputSGR()) }
func BenchmarkTerminalMixed(b *testing.B) { benchTerminal(b, benchInputMixed()) }
