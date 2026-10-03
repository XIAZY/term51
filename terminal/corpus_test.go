package terminal

// Tests against the recorded corpus: outputs of the pinned upstream
// libghostty-vt, kept in the term51 repository. The corpus location is
// LIBGHOSTTY_VT_CORPUS, defaulting to a term51 checkout next to this
// repository. A missing corpus fails the tests; it never skips them.

import (
	"bufio"
	"compress/gzip"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func corpusDir(t testing.TB) string {
	t.Helper()
	dir := os.Getenv("LIBGHOSTTY_VT_CORPUS")
	if dir == "" {
		dir = filepath.Join("..", "..", "term51")
	}
	if _, err := os.Stat(filepath.Join(dir, "golden", "MANIFEST")); err != nil {
		t.Fatalf("corpus not found at %q (set LIBGHOSTTY_VT_CORPUS): %v", dir, err)
	}
	return dir
}

// corpusLines returns the lines of a corpus file, transparently
// decompressing ".gz" files.
func corpusLines(t testing.TB, rel string) []string {
	t.Helper()
	f, err := os.Open(filepath.Join(corpusDir(t), rel))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var r io.Reader = f
	if strings.HasSuffix(rel, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			t.Fatal(err)
		}
		defer gz.Close()
		r = gz
	}
	var out []string
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<28)
	for sc.Scan() {
		out = append(out, sc.Text())
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// corpusCase is one case of a suite file: its name and the bytes of each
// feed op, in order.
type corpusCase struct {
	name  string
	feeds [][]byte
}

// corpusSuite reads the cases of cases/<rel>.
func corpusSuite(t testing.TB, rel string) []corpusCase {
	t.Helper()
	var out []corpusCase
	for _, line := range corpusLines(t, filepath.Join("cases", rel)) {
		switch {
		case strings.HasPrefix(line, "case "):
			out = append(out, corpusCase{name: line[5:]})
		case strings.HasPrefix(line, "feed"):
			b, err := hex.DecodeString(strings.TrimSpace(line[4:]))
			if err != nil {
				t.Fatalf("%s: bad feed line: %v", rel, err)
			}
			out[len(out)-1].feeds = append(out[len(out)-1].feeds, b)
		}
	}
	return out
}

// corpusOutputs splits a recorded output file into its per-case sections.
func corpusOutputs(t testing.TB, rel string) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	name := ""
	for _, line := range corpusLines(t, filepath.Join("golden", rel)) {
		if strings.HasPrefix(line, "## case ") {
			name = line[8:]
			out[name] = []string{}
			continue
		}
		out[name] = append(out[name], line)
	}
	return out
}

// The parser state table must equal upstream's, entry by entry.
func TestCorpusParseTable(t *testing.T) {
	lines := corpusLines(t, filepath.Join("golden", "tables", "parse_table.txt"))
	if len(lines) != 256*parserStateCount {
		t.Fatalf("got %d lines, want %d", len(lines), 256*parserStateCount)
	}
	i := 0
	for s := 0; s < parserStateCount; s++ {
		for c := 0; c < 256; c++ {
			tr := table[c][s]
			got := fmt.Sprintf("%s %02x -> %s %s", ParserState(s), c, tr.state, tr.action)
			if got != lines[i] {
				t.Errorf("line %d:\n got  %s\n want %s", i+1, got, lines[i])
			}
			i++
		}
	}
}

// The decoder must produce exactly the codepoints upstream printed for
// every input of the utf8_decode suite (valid, truncated, overlong,
// surrogate and random malformed sequences).
func TestCorpusUTF8Decode(t *testing.T) {
	cases := corpusSuite(t, filepath.Join("parser", "utf8_decode.case"))
	want := corpusOutputs(t, filepath.Join("parser", "utf8_decode.actions.gz"))
	if len(cases) < 60000 {
		t.Fatalf("only %d cases", len(cases))
	}
	failures := 0
	for _, c := range cases {
		lines, ok := want[c.name]
		if !ok || len(lines) != 1 || !strings.HasPrefix(lines[0], "@print ") {
			t.Fatalf("%s: unexpected recorded trace %q", c.name, lines)
		}
		var wantCps []uint32
		for _, h := range strings.Split(lines[0][7:], ",") {
			v, err := strconv.ParseUint(h, 16, 32)
			if err != nil {
				t.Fatal(err)
			}
			wantCps = append(wantCps, uint32(v))
		}

		// Decode the way the stream does: re-feed a byte that the
		// decoder reports as not consumed.
		var d UTF8Decoder
		var got []uint32
		for _, feed := range c.feeds {
			for _, b := range feed {
				consumed := false
				for !consumed {
					var cp Optional[uint32]
					cp, consumed = d.Next(b)
					if cp.Valid {
						got = append(got, cp.Value)
					}
				}
			}
		}
		if fmt.Sprint(got) != fmt.Sprint(wantCps) {
			failures++
			if failures <= 10 {
				t.Errorf("%s: input % x\n got  %x\n want %x", c.name, c.feeds, got, wantCps)
			}
		}
	}
	if failures > 0 {
		t.Errorf("%d of %d cases differ", failures, len(cases))
	}
}

// dumpEscaped mirrors the recorder's writeEscaped: printable ASCII verbatim,
// everything else (and backslash) as \xNN.
func dumpEscaped(b []uint8) string {
	var sb strings.Builder
	for _, c := range b {
		if c >= 0x20 && c < 0x7f && c != '\\' {
			sb.WriteByte(c)
		} else {
			fmt.Fprintf(&sb, "\\x%02x", c)
		}
	}
	return sb.String()
}

func dumpU16s(v []uint16) string {
	parts := make([]string, len(v))
	for i, x := range v {
		parts[i] = strconv.Itoa(int(x))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// dumpParserAction renders an action exactly as the recorder's parser trace
// does (harness/src/trace.zig parserFeed).
func dumpParserAction(a ParserAction) []string {
	tag := a.Tag.String()
	out := []string{"@" + tag, " v=" + tag}
	p := " v." + tag
	switch a.Tag {
	case ParserActionPrint:
		out = append(out, fmt.Sprintf("%s=%d", p, a.Print))
	case ParserActionExecute:
		out = append(out, fmt.Sprintf("%s=%d", p, a.Execute))
	case ParserActionDcsPut:
		out = append(out, fmt.Sprintf("%s=%d", p, a.DcsPut))
	case ParserActionApcPut:
		out = append(out, fmt.Sprintf("%s=%d", p, a.ApcPut))
	case ParserActionCsiDispatch:
		v := a.CsiDispatch
		out = append(out,
			p+".intermediates="+dumpEscaped(v.Intermediates),
			p+".params="+dumpU16s(v.Params),
			fmt.Sprintf("%s.params_sep.mask=%d", p, v.ParamsSep.mask),
			fmt.Sprintf("%s.final=%d", p, v.Final),
		)
	case ParserActionEscDispatch:
		v := a.EscDispatch
		out = append(out,
			p+".intermediates="+dumpEscaped(v.Intermediates),
			fmt.Sprintf("%s.final=%d", p, v.Final),
		)
	case ParserActionDcsHook:
		v := a.DcsHook
		out = append(out,
			p+".intermediates="+dumpEscaped(v.Intermediates),
			p+".params="+dumpU16s(v.Params),
			fmt.Sprintf("%s.final=%d", p, v.Final),
		)
	case ParserActionDcsUnhook, ParserActionApcStart, ParserActionApcEnd:
		// void payload: nothing more
	case ParserActionOscDispatch:
		out = append(out, p+"=<osc: not yet ported>")
	}
	return out
}

// Every byte of every parser suite must leave the parser in the state
// upstream recorded and produce the same actions.
//
// PENDING: cases whose recorded trace contains an OSC dispatch are not
// compared until osc.zig is ported (terminal/osc.go is a placeholder). The
// test logs how many that is, and fails if the number is ever zero while the
// exclusion is still in place.
func TestCorpusParserTrace(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join(corpusDir(t), "golden", "parser"))
	if err != nil {
		t.Fatal(err)
	}
	total, pending, failures := 0, 0, 0
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".parser.gz") {
			continue
		}
		suite := strings.TrimSuffix(e.Name(), ".parser.gz")
		cases := corpusSuite(t, filepath.Join("parser", suite+".case"))
		want := corpusOutputs(t, filepath.Join("parser", e.Name()))
		for _, c := range cases {
			wantLines := want[c.name]
			total++
			hasOSC := false
			for _, l := range wantLines {
				if l == "@osc_dispatch" {
					hasOSC = true
					break
				}
			}
			if hasOSC {
				pending++
				continue
			}

			p := ParserInit()
			var got []string
			for _, feed := range c.feeds {
				for _, b := range feed {
					actions := p.Next(b)
					got = append(got, fmt.Sprintf("> %02x state=%s", b, p.state))
					for _, a := range actions {
						if a.Valid {
							got = append(got, dumpParserAction(a.Value)...)
						}
					}
				}
			}
			p.Deinit()

			if strings.Join(got, "\n") != strings.Join(wantLines, "\n") {
				failures++
				if failures <= 5 {
					for i := 0; i < len(got) || i < len(wantLines); i++ {
						var g, w string
						if i < len(got) {
							g = got[i]
						}
						if i < len(wantLines) {
							w = wantLines[i]
						}
						if g != w {
							t.Errorf("%s/%s line %d:\n got  %q\n want %q", suite, c.name, i+1, g, w)
							break
						}
					}
				}
			}
		}
	}
	t.Logf("%d parser-trace cases: %d compared, %d pending on the OSC port, %d differ", total, total-pending, pending, failures)
	if total < 8000 {
		t.Errorf("only %d cases found", total)
	}
	if pending == 0 {
		t.Errorf("no case is pending on OSC: remove the exclusion from this test")
	}
	if failures > 0 {
		t.Errorf("%d cases differ", failures)
	}
}
