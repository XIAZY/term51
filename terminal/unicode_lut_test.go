// Tests for src/unicode/lut.zig (Ghostty v1.3.1). lut.zig has no Zig
// tests; the table reader is checked against the pinned build by
// corpus_unicode_test.go (table.get for every u21 codepoint, writeZig).
// unicodeExpectPanic is defined in unicode_grapheme_test.go.

package terminal

import "testing"

// Go-only: panic test: lut.Tables.get: a codepoint above maxInt(u21) (cannot exist in Zig: cp is a u21; without the assert the index would read past stage1)
func TestUnicodeLut_panic_Tables_get_codepoint_above_u21(t *testing.T) {
	unicodeExpectPanic(t, func() { UnicodeTable.Get(0x200000) })
	unicodeExpectPanic(t, func() { UnicodeTable.Get(0xFFFFFFFF) })
}
