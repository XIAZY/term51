package terminal

// Tests against the recorded data of the pinned upstream build for
// src/unicode (data/unicode/ in the term51 corpus). The files are written by
// two gated Zig tests in the term51 build ("term51: table: dump exact data
// when TERM51_UNICODE_DUMP_DIR is set" in props_table.zig and "term51:
// graphemeBreak: dump exact data when TERM51_UNICODE_DUMP_DIR is set" in
// grapheme.zig), run by term51's tools/dump_unicode.sh, which also records
// the Unicode version of uucode's data. These tests regenerate the same
// lines with the Go port and compare them exactly; every check is
// exhaustive:
//
//   - the embedded res/ files are the dumped files byte for byte;
//   - table.get for every u21 codepoint (0..0x1FFFFF, all four Properties
//     fields): props_runs.txt (runs of equal values) and props_sha256.txt
//     (SHA-256 of each 64K block's bit patterns);
//   - lut.Tables.writeZig against the Zig file the build generated;
//   - every byte of Precompute.data, and Precompute.data[Key.index()] for
//     every key built from valid enum values;
//   - graphemeBreak for every state and every pair of classes, with the
//     lowest and highest codepoint of each class (2x2 pairs each).
//
// The test names use a lower-case "corpus" so that the corpus tests of the
// other files (-run Corpus) do not select them. corpusDir and corpusLines
// are in corpus_test.go; col_corpusExpectLines in corpus_color_test.go.

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

// unicodeTestVersion is res/unicode_version.txt: the Unicode version of the
// embedded tables' data (uucode's UCD files), recorded by
// tools/dump_unicode.sh.
//
//go:embed res/unicode_version.txt
var unicodeTestVersion string

func unicodeCorpusFile(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(corpusDir(t), "data", "unicode", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func unicodeCorpusHex(v uint32) string { return strconv.FormatUint(uint64(v), 16) }

func unicodeCorpusBit(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// The embedded files are the dumped files, byte for byte.
func TestUnicode_corpus_res_files(t *testing.T) {
	expect(t, bytes.Equal(unicodePropsTableData, unicodeCorpusFile(t, "unicode_tables.bin")),
		"res/unicode_tables.bin == data/unicode/unicode_tables.bin")
	expect(t, bytes.Equal(unicodeGraphemePrecomputeBin, unicodeCorpusFile(t, "grapheme_precompute.bin")),
		"res/grapheme_precompute.bin == data/unicode/grapheme_precompute.bin")
	expect(t, unicodeTestVersion == string(unicodeCorpusFile(t, "unicode_version.txt")),
		"res/unicode_version.txt == data/unicode/unicode_version.txt")
}

// The Unicode version of the embedded data (uucode 0.2.0's UCD) is the
// version of Go's unicode package: the port's day-one decision to use Go's
// Unicode version still holds. A mismatch means one side moved: rerun
// tools/dump_unicode.sh for a new uucode, or review the decision for a new
// Go.
func TestUnicode_version_matches_go(t *testing.T) {
	expectEqual(t, unicode.Version, strings.TrimSuffix(unicodeTestVersion, "\n"))
}

// table.get for every u21 codepoint, as runs of equal values.
func TestUnicode_corpus_props_runs(t *testing.T) {
	want := corpusLines(t, filepath.Join("data", "unicode", "props_runs.txt"))
	var got []string
	line := func(first, last uint32, p UnicodeProperties) string {
		return unicodeCorpusHex(first) + " " + unicodeCorpusHex(last) + " " +
			itoa(uint64(p.width)) + " " + unicodeCorpusBit(p.width_zero_in_grapheme) + " " +
			p.grapheme_break.String() + " " + unicodeCorpusBit(p.emoji_vs_base)
	}
	start := uint32(0)
	prev := UnicodeTable.Get(0)
	for cp := uint32(1); cp <= 0x1FFFFF; cp++ {
		p := UnicodeTable.Get(cp)
		if !p.Eql(prev) {
			got = append(got, line(start, cp-1, prev))
			start = cp
			prev = p
		}
	}
	got = append(got, line(start, 0x1FFFFF, prev))
	col_corpusExpectLines(t, "props_runs.txt", want, got)
}

// table.get for every u21 codepoint: SHA-256 per 64K block of the
// Properties bit patterns (u16 little endian).
func TestUnicode_corpus_props_sha256(t *testing.T) {
	want := corpusLines(t, filepath.Join("data", "unicode", "props_sha256.txt"))
	var got []string
	var b [2 * 65536]byte
	for block := uint32(0); block < 0x200000; block += 65536 {
		for i := uint32(0); i < 65536; i++ {
			binary.LittleEndian.PutUint16(b[2*i:], UnicodeTable.Get(block+i).Int())
		}
		d := sha256.Sum256(b[:])
		h := strconv.FormatUint(uint64(block), 16)
		got = append(got, strings.Repeat("0", 6-len(h))+h+" "+hex.EncodeToString(d[:]))
	}
	col_corpusExpectLines(t, "props_sha256.txt", want, got)
}

// lut.Tables.writeZig reproduces the Zig file the build generated (and
// compiled as the `unicode_tables` module).
func TestUnicode_corpus_write_zig(t *testing.T) {
	w := oscWriterAllocatingInit()
	defer w.deinit()
	zigTry(t, UnicodeTable.WriteZig(&w.writer))
	want := unicodeCorpusFile(t, "unicode_tables.zig.txt")
	if !bytes.Equal(w.written(), want) {
		col_corpusExpectLines(t, "unicode_tables.zig.txt",
			strings.Split(string(want), "\n"), strings.Split(string(w.written()), "\n"))
		t.Fatal("unicode_tables.zig.txt differs")
	}
}

// Every byte of Precompute.data decodes to a Value whose bit pattern is that
// byte, and Precompute.data[Key.index()] for every key of valid enum values
// is what Zig reads.
func TestUnicode_corpus_precompute(t *testing.T) {
	raw := unicodeCorpusFile(t, "grapheme_precompute.bin")
	expectEqual(t, 8192, len(raw))
	for i, v := range unicodeGraphemePrecomputeData {
		b := uint8(v.state) << 1
		if v.result {
			b |= 1
		}
		if b != raw[i] {
			t.Fatalf("Precompute.data[%d]: port %#x, Zig %#x", i, b, raw[i])
		}
	}

	want := corpusLines(t, filepath.Join("data", "unicode", "grapheme_precompute.txt"))
	var got []string
	for s := range len(uucodeGraphemeBreakStateNames) {
		for g1 := range len(uucodeXTypesGraphemeBreakNoControlNames) {
			for g2 := range len(uucodeXTypesGraphemeBreakNoControlNames) {
				key := unicodeGraphemePrecomputeKey{
					state: UucodeGraphemeBreakState(s),
					gb1:   UucodeXTypesGraphemeBreakNoControl(g1),
					gb2:   UucodeXTypesGraphemeBreakNoControl(g2),
				}
				v := unicodeGraphemePrecomputeData[key.index()]
				got = append(got, itoa(uint64(key.index()))+" "+key.state.String()+" "+
					key.gb1.String()+" "+key.gb2.String()+" -> "+strconv.FormatBool(v.result)+" "+v.state.String())
			}
		}
	}
	col_corpusExpectLines(t, "grapheme_precompute.txt", want, got)
}

// graphemeBreak for every state and pair of classes, called once with that
// state on the lowest and highest codepoint of each class (2x2 pairs).
func TestUnicode_corpus_grapheme_break(t *testing.T) {
	want := corpusLines(t, filepath.Join("data", "unicode", "grapheme_break.txt"))
	n := len(uucodeXTypesGraphemeBreakNoControlNames)
	first := make([]Optional[uint32], n)
	last := make([]Optional[uint32], n)
	for cp := uint32(0); cp <= 0x1FFFFF; cp++ {
		c := UnicodeTable.Get(cp).grapheme_break
		if !first[c].Valid {
			first[c] = Some(cp)
		}
		last[c] = Some(cp)
	}
	var got []string
	for c := range n {
		if first[c].Valid {
			got = append(got, "sample "+UucodeXTypesGraphemeBreakNoControl(c).String()+" "+
				unicodeCorpusHex(first[c].Value)+" "+unicodeCorpusHex(last[c].Value))
		}
	}
	for s := range len(uucodeGraphemeBreakStateNames) {
		for g1 := range n {
			for g2 := range n {
				if !first[g1].Valid || !first[g2].Valid {
					continue
				}
				for _, cp1 := range []uint32{first[g1].Value, last[g1].Value} {
					for _, cp2 := range []uint32{first[g2].Value, last[g2].Value} {
						state := UucodeGraphemeBreakState(s)
						r := UnicodeGraphemeBreak(cp1, cp2, &state)
						got = append(got, UucodeGraphemeBreakState(s).String()+" "+unicodeCorpusHex(cp1)+" "+
							unicodeCorpusHex(cp2)+" -> "+strconv.FormatBool(r)+" "+state.String())
					}
				}
			}
		}
	}
	col_corpusExpectLines(t, "grapheme_break.txt", want, got)
}
