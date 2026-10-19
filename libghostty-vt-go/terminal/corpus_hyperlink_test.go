package terminal

// Corpus test for hyperlink.zig: PageEntry.hash over the recorded entries of
// data/sets/hyperlink_hash.txt. The generator is the Zig dump test
// `term51: PageEntry.hash: dump exact data when TERM51_SET_DUMP_DIR is set` at
// the end of hyperlink.zig, which builds each entry in a page. Each line is
//
//	n kind id uri hash
//
// where kind is `implicit` or `explicit`, id is the implicit u32 in decimal or
// the explicit id's bytes in hex (`-` when empty), uri is the uri's bytes in
// hex (`-` when empty) and hash is PageEntry.hash in 16 hex digits. The Go
// entry holds its slices directly (no page memory), so the entries are built
// here without a page.

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// hl_corpusBytes parses a hex field of hyperlink_hash.txt (`-` is empty).
func hl_corpusBytes(t *testing.T, line int, s string) []uint8 {
	t.Helper()
	if s == "-" {
		return []uint8{}
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("line %d: bad hex %q: %v", line, s, err)
	}
	return b
}

// Go-only: corpus (PageEntry.hash must equal upstream's for every one of the 3,000 recorded entries; the Zig dump test is not translated)
func TestHyperlink_corpus_hash(t *testing.T) {
	lines := corpusLines(t, "data/sets/hyperlink_hash.txt")
	if len(lines) != 3000 {
		t.Fatalf("got %d lines, want 3000", len(lines))
	}
	failures := 0
	for i, line := range lines {
		f := strings.Fields(line)
		if len(f) != 5 {
			t.Fatalf("line %d: %d fields, want 5: %q", i+1, len(f), line)
		}
		n, err := strconv.Atoi(f[0])
		if err != nil || n != i {
			t.Fatalf("line %d: bad index %q", i+1, f[0])
		}
		var e HyperlinkPageEntry
		switch f[1] {
		case "implicit":
			v, err := strconv.ParseUint(f[2], 10, 32)
			if err != nil {
				t.Fatalf("line %d: bad implicit id %q: %v", i+1, f[2], err)
			}
			e.id = HyperlinkPageEntryId{Tag: HyperlinkPageEntryIdImplicit, Implicit: uint32(v)}
		case "explicit":
			e.id = HyperlinkPageEntryId{Tag: HyperlinkPageEntryIdExplicit, Explicit: hl_corpusBytes(t, i+1, f[2])}
		default:
			t.Fatalf("line %d: bad kind %q", i+1, f[1])
		}
		e.uri = hl_corpusBytes(t, i+1, f[3])
		got := fmt.Sprintf("%016x", e.Hash())
		if got != f[4] {
			failures++
			if failures <= 10 {
				t.Errorf("line %d (%s): got %s, want %s", i+1, line, got, f[4])
			}
		}
	}
	if failures > 0 {
		t.Errorf("%d of %d entries differ", failures, len(lines))
	}
}
