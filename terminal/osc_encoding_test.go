// Tests translated one for one from src/terminal/osc/encoding.zig (Ghostty
// v1.3.1 plus the term51 additional tests). Each Go test carries the exact Zig
// title in a "// Zig: test" comment. The helpers expect and expectEqual are
// defined in Parser_test.go.
//
// Translation notes:
//   - `isSafeUtf8(&.{c})` is OSCEncodingIsSafeUtf8([]uint8{c}).

package terminal

import (
	"testing"
)

// Zig: test isSafeUtf8
func TestOSCEncoding_isSafeUtf8(t *testing.T) {
	expect(t, OSCEncodingIsSafeUtf8([]uint8("Hello world!")), "isSafeUtf8(\"Hello world!\")")
	expect(t, OSCEncodingIsSafeUtf8([]uint8("安全的ユニコード☀️")), "isSafeUtf8(\"安全的ユニコード☀️\")")
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("No linebreaks\nallowed")), "!isSafeUtf8(\"No linebreaks\\nallowed\")")
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("\x07no bells")), "!isSafeUtf8(\"\\x07no bells\")")
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("\x1b]9;no OSCs\x1b\\\x1b[m")), "!isSafeUtf8(\"\\x1b]9;no OSCs\\x1b\\\\\\x1b[m\")")
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("\x9f8-bit escapes are clever, but no")), "!isSafeUtf8(\"\\x9f8-bit escapes are clever, but no\")")
}

// ---------------------------------------------------------------------------
// term51: additional tests. Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

// Zig: test "term51: isSafeUtf8: empty string"
func TestOSCEncoding_term51_isSafeUtf8_empty_string(t *testing.T) {
	expect(t, OSCEncodingIsSafeUtf8([]uint8("")), "isSafeUtf8(\"\")")
}

// Zig: test "term51: isSafeUtf8: every single byte"
func TestOSCEncoding_term51_isSafeUtf8_every_single_byte(t *testing.T) {
	// As a one-byte string only 0x20-0x7e are safe: below are the C0
	// controls, 0x7f is DEL and 0x80 and above are not valid UTF-8 alone.
	for i := 0; i < 256; i++ {
		c := uint8(i)
		expectEqual(t, c >= 0x20 && c <= 0x7e, OSCEncodingIsSafeUtf8([]uint8{c}))
		// One unsafe byte anywhere makes the whole string unsafe.
		expectEqual(t, c >= 0x20 && c <= 0x7e, OSCEncodingIsSafeUtf8([]uint8{'a', c, 'b'}))
	}
}

// Zig: test "term51: isSafeUtf8: C1 controls encoded as UTF-8"
func TestOSCEncoding_term51_isSafeUtf8_C1_controls_encoded_as_UTF_8(t *testing.T) {
	// U+0080 to U+009F are rejected, U+00A0 is the first accepted code
	// point after them.
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("\xc2\x80")), "!isSafeUtf8(\"\\xc2\\x80\")")
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("\xc2\x85")), "!isSafeUtf8(\"\\xc2\\x85\")")
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("\xc2\x9f")), "!isSafeUtf8(\"\\xc2\\x9f\")")
	expect(t, OSCEncodingIsSafeUtf8([]uint8("\xc2\xa0")), "isSafeUtf8(\"\\xc2\\xa0\")")
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("ok\xc2\x9bok")), "!isSafeUtf8(\"ok\\xc2\\x9bok\")")
}

// Zig: test "term51: isSafeUtf8: multi-byte boundaries"
func TestOSCEncoding_term51_isSafeUtf8_multi_byte_boundaries(t *testing.T) {
	// largest two-byte, smallest and largest three-byte, smallest and
	// largest four-byte code points
	expect(t, OSCEncodingIsSafeUtf8([]uint8("\xdf\xbf")), "isSafeUtf8(\"\\xdf\\xbf\")")
	expect(t, OSCEncodingIsSafeUtf8([]uint8("\xe0\xa0\x80")), "isSafeUtf8(\"\\xe0\\xa0\\x80\")")
	expect(t, OSCEncodingIsSafeUtf8([]uint8("\xef\xbf\xbf")), "isSafeUtf8(\"\\xef\\xbf\\xbf\")")
	expect(t, OSCEncodingIsSafeUtf8([]uint8("\xf0\x90\x80\x80")), "isSafeUtf8(\"\\xf0\\x90\\x80\\x80\")")
	expect(t, OSCEncodingIsSafeUtf8([]uint8("\xf4\x8f\xbf\xbf")), "isSafeUtf8(\"\\xf4\\x8f\\xbf\\xbf\")")
	// just outside the surrogate range
	expect(t, OSCEncodingIsSafeUtf8([]uint8("\xed\x9f\xbf")), "isSafeUtf8(\"\\xed\\x9f\\xbf\")")
	expect(t, OSCEncodingIsSafeUtf8([]uint8("\xee\x80\x80")), "isSafeUtf8(\"\\xee\\x80\\x80\")")
	// Other invisible or formatting code points are not rejected.
	expect(t, OSCEncodingIsSafeUtf8([]uint8("\xe2\x80\x8b")), "isSafeUtf8(\"\\xe2\\x80\\x8b\")")
	expect(t, OSCEncodingIsSafeUtf8([]uint8("\xe2\x80\xa8")), "isSafeUtf8(\"\\xe2\\x80\\xa8\")")
	expect(t, OSCEncodingIsSafeUtf8([]uint8("\xef\xbb\xbf")), "isSafeUtf8(\"\\xef\\xbb\\xbf\")")
}

// Zig: test "term51: isSafeUtf8: invalid UTF-8"
func TestOSCEncoding_term51_isSafeUtf8_invalid_UTF_8(t *testing.T) {
	// lone continuation byte and lone lead bytes
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("\x80")), "!isSafeUtf8(\"\\x80\")")
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("\xc2")), "!isSafeUtf8(\"\\xc2\")")
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("\xe2\x82")), "!isSafeUtf8(\"\\xe2\\x82\")")
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("\xf0\x9f\x98")), "!isSafeUtf8(\"\\xf0\\x9f\\x98\")")
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("a\xe2\x82")), "!isSafeUtf8(\"a\\xe2\\x82\")")
	// lead byte followed by a non-continuation byte
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("\xc2\x20")), "!isSafeUtf8(\"\\xc2\\x20\")")
	// overlong encodings
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("\xc0\x80")), "!isSafeUtf8(\"\\xc0\\x80\")")
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("\xc1\xbf")), "!isSafeUtf8(\"\\xc1\\xbf\")")
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("\xe0\x9f\xbf")), "!isSafeUtf8(\"\\xe0\\x9f\\xbf\")")
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("\xf0\x8f\xbf\xbf")), "!isSafeUtf8(\"\\xf0\\x8f\\xbf\\xbf\")")
	// surrogates
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("\xed\xa0\x80")), "!isSafeUtf8(\"\\xed\\xa0\\x80\")")
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("\xed\xbf\xbf")), "!isSafeUtf8(\"\\xed\\xbf\\xbf\")")
	// beyond U+10FFFF and bytes that never appear in UTF-8
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("\xf4\x90\x80\x80")), "!isSafeUtf8(\"\\xf4\\x90\\x80\\x80\")")
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("\xf5\x80\x80\x80")), "!isSafeUtf8(\"\\xf5\\x80\\x80\\x80\")")
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("\xfe")), "!isSafeUtf8(\"\\xfe\")")
	expect(t, !OSCEncodingIsSafeUtf8([]uint8("\xff")), "!isSafeUtf8(\"\\xff\")")
}
