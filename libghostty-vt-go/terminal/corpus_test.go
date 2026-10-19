package terminal

// Tests against the recorded corpus: outputs of the pinned upstream
// libghostty-vt, kept in the reference-tests repository (term51 on the
// project's machine; not published). The corpus location is
// LIBGHOSTTY_VT_CORPUS, defaulting to a checkout named term51 beside the
// repository this library is part of. A missing corpus fails the tests; it
// never skips them.

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
		dir = filepath.Join("..", "..", "..", "term51")
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

// The parser trace of every byte of every parser suite (the old
// TestCorpusParserTrace, with its PENDING exclusion of the traces that
// contain an OSC dispatch) is now the `parser` kind of TestCorpusReplayParser
// (corpus_replay_trace_test.go: parserFeed), without any exclusion.
