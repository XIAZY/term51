// Tests translated one for one from src/unicode/grapheme.zig (Ghostty
// v1.3.1). Each Go test carries the exact Zig title in a "// Zig: test"
// comment. The helper expect is defined in Parser_test.go.
//
// Not translated: "term51: graphemeBreak: dump exact data when
// TERM51_UNICODE_DUMP_DIR is set" (term51 build): it only writes the data
// files under data/unicode/, which corpus_unicode_test.go compares against.
//
// Translation notes:
//   - `var state: uucode.grapheme.BreakState = .default` is
//     `state := UucodeGraphemeBreakStateDefault`.
//   - `uucode.utf8.Iterator.init(s)` and `it.next() orelse unreachable` are
//     unicodeGraphemeTestIterator below: it decodes the (valid UTF-8)
//     string one codepoint at a time and panics when it is exhausted.

package terminal

import (
	"testing"
	"unicode/utf8"
)

// unicodeGraphemeTestIterator stands in for uucode.utf8.Iterator over a
// valid UTF-8 string.
type unicodeGraphemeTestIterator struct {
	s string
}

// next returns the next codepoint; `orelse unreachable` on null is the
// panic.
func (self *unicodeGraphemeTestIterator) next() uint32 {
	if len(self.s) == 0 {
		panic("unreachable")
	}
	r, n := utf8.DecodeRuneInString(self.s)
	self.s = self.s[n:]
	return uint32(r)
}

// Zig: test "grapheme break: emoji modifier"
func TestUnicodeGrapheme_grapheme_break_emoji_modifier(t *testing.T) {
	// Emoji and modifier
	{
		state := UucodeGraphemeBreakStateDefault
		expect(t, !UnicodeGraphemeBreak(0x261D, 0x1F3FF, &state), "!graphemeBreak(0x261D, 0x1F3FF, &state)")
	}

	// Non-emoji and emoji modifier
	{
		state := UucodeGraphemeBreakStateDefault
		expect(t, UnicodeGraphemeBreak(0x22, 0x1F3FF, &state), "graphemeBreak(0x22, 0x1F3FF, &state)")
	}
}

// Zig: test "long emoji zwj sequences"
func TestUnicodeGrapheme_long_emoji_zwj_sequences(t *testing.T) {
	state := UucodeGraphemeBreakStateDefault
	// 👩‍👩‍👧‍👦 (family: woman, woman, girl, boy)
	it := unicodeGraphemeTestIterator{s: "\U0001F469‍\U0001F469‍\U0001F467‍\U0001F466_"}
	cp1 := it.next()
	cp2 := it.next()
	expect(t, cp1 == 0x1F469, "cp1 == 0x1F469") // 👩
	expect(t, !UnicodeGraphemeBreak(cp1, cp2, &state), "!graphemeBreak(cp1, cp2, &state)")

	cp1 = cp2
	cp2 = it.next()
	expect(t, cp1 == 0x200D, "cp1 == 0x200D")
	expect(t, !UnicodeGraphemeBreak(cp1, cp2, &state), "!graphemeBreak(cp1, cp2, &state)")

	cp1 = cp2
	cp2 = it.next()
	expect(t, cp1 == 0x1F469, "cp1 == 0x1F469") // 👩
	expect(t, !UnicodeGraphemeBreak(cp1, cp2, &state), "!graphemeBreak(cp1, cp2, &state)")

	cp1 = cp2
	cp2 = it.next()
	expect(t, cp1 == 0x200D, "cp1 == 0x200D")
	expect(t, !UnicodeGraphemeBreak(cp1, cp2, &state), "!graphemeBreak(cp1, cp2, &state)")

	cp1 = cp2
	cp2 = it.next()
	expect(t, cp1 == 0x1F467, "cp1 == 0x1F467") // 👧
	expect(t, !UnicodeGraphemeBreak(cp1, cp2, &state), "!graphemeBreak(cp1, cp2, &state)")

	cp1 = cp2
	cp2 = it.next()
	expect(t, cp1 == 0x200D, "cp1 == 0x200D")
	expect(t, !UnicodeGraphemeBreak(cp1, cp2, &state), "!graphemeBreak(cp1, cp2, &state)")

	cp1 = cp2
	cp2 = it.next()
	expect(t, cp1 == 0x1F466, "cp1 == 0x1F466")                                          // 👦
	expect(t, UnicodeGraphemeBreak(cp1, cp2, &state), "graphemeBreak(cp1, cp2, &state)") // break
}

// unicodeExpectPanic fails unless f panics.
func unicodeExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic")
		}
	}()
	f()
}

// Go-only: panic test: graphemeBreak: a codepoint above maxInt(u21) (cannot exist in Zig: cp1 and cp2 are u21; table.get asserts the range)
func TestUnicodeGrapheme_panic_graphemeBreak_codepoint_above_u21(t *testing.T) {
	state := UucodeGraphemeBreakStateDefault
	unicodeExpectPanic(t, func() { UnicodeGraphemeBreak('a', 0x200000, &state) })
	unicodeExpectPanic(t, func() { UnicodeGraphemeBreak(0x200000, 'a', &state) })
}
