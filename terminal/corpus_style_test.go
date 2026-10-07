package terminal

// Corpus test for style.zig: Style.hash over the recorded styles of
// data/sets/style_hash.txt. The generator is the Zig dump test
// `term51: Style.hash: dump exact data when TERM51_SET_DUMP_DIR is set` at the
// end of style.zig. Each line is
//
//	n fg bg ul flags hash
//
// where a colour is `none`, `pal:N` (N decimal) or `rgb:rrggbb`, flags is the
// 16-bit Style.Flags value in hex (four digits) and hash is Style.hash in 16
// hex digits. Style.Flags layout (style.zig): bold is bit 0, italic 1, faint
// 2, blink 3, inverse 4, invisible 5, strikethrough 6, overline 7, underline
// bits 8..10, _padding bits 11..15.

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// sm3_corpusColor parses a colour of style_hash.txt.
func sm3_corpusColor(t *testing.T, s string) StyleColor {
	t.Helper()
	switch {
	case s == "none":
		return StyleColor{}
	case strings.HasPrefix(s, "pal:"):
		v, err := strconv.ParseUint(s[4:], 10, 8)
		if err != nil {
			t.Fatalf("bad colour %q: %v", s, err)
		}
		return sm3_pal(uint8(v))
	case strings.HasPrefix(s, "rgb:") && len(s) == 10:
		v, err := strconv.ParseUint(s[4:], 16, 24)
		if err != nil {
			t.Fatalf("bad colour %q: %v", s, err)
		}
		return sm3_rgb(uint8(v>>16), uint8(v>>8), uint8(v))
	}
	t.Fatalf("bad colour %q", s)
	return StyleColor{}
}

// sm3_corpusFlags is `@bitCast` of a u16 to Style.Flags.
func sm3_corpusFlags(v uint16) StyleFlags {
	return StyleFlags{
		Bold:          v&(1<<0) != 0,
		Italic:        v&(1<<1) != 0,
		Faint:         v&(1<<2) != 0,
		Blink:         v&(1<<3) != 0,
		Inverse:       v&(1<<4) != 0,
		Invisible:     v&(1<<5) != 0,
		Strikethrough: v&(1<<6) != 0,
		Overline:      v&(1<<7) != 0,
		Underline:     SGRAttributeUnderline((v >> 8) & 0x7),
		_padding:      uint8(v >> 11),
	}
}

// Style.hash must equal upstream's for every one of the 4,096 recorded
// styles.
func TestStyle_corpus_hash(t *testing.T) {
	lines := corpusLines(t, "data/sets/style_hash.txt")
	if len(lines) != 4096 {
		t.Fatalf("got %d lines, want 4096", len(lines))
	}
	failures := 0
	for i, line := range lines {
		f := strings.Fields(line)
		if len(f) != 6 {
			t.Fatalf("line %d: %d fields, want 6: %q", i+1, len(f), line)
		}
		n, err := strconv.Atoi(f[0])
		if err != nil || n != i {
			t.Fatalf("line %d: bad index %q", i+1, f[0])
		}
		flags, err := strconv.ParseUint(f[4], 16, 16)
		if err != nil {
			t.Fatalf("line %d: bad flags %q: %v", i+1, f[4], err)
		}
		s := Style{
			FgColor:         sm3_corpusColor(t, f[1]),
			BgColor:         sm3_corpusColor(t, f[2]),
			underline_color: sm3_corpusColor(t, f[3]),
			Flags:           sm3_corpusFlags(uint16(flags)),
		}
		got := fmt.Sprintf("%016x", s.Hash())
		if got != f[5] {
			failures++
			if failures <= 10 {
				t.Errorf("line %d (%s): got %s, want %s", i+1, line, got, f[5])
			}
		}
	}
	if failures > 0 {
		t.Errorf("%d of %d styles differ", failures, len(lines))
	}
}
