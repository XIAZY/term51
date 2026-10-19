// Ported from src/terminal/osc/encoding.zig (Ghostty v1.3.1).
//
// Specialized encodings used in some OSC protocols.
//
// std.unicode.Utf8View.init validates the whole string first; Go's
// unicode/utf8.Valid accepts exactly the same strings (no overlong forms,
// no surrogates, nothing above U+10FFFF, no truncated sequences), and
// ranging over the validated string then yields the same code points as
// Utf8View's iterator. See docs/mapping.md.
//
// File naming: `osc/encoding.zig` becomes `osc_encoding.go` in package
// terminal, and its declarations take the prefix `OSCEncoding`. The file has
// no C ABI declarations.

package terminal

import "unicode/utf8"

// OSCEncodingIsSafeUtf8 is osc.encoding.isSafeUtf8.
//
// Kitty defines "Escape code safe UTF-8" as valid UTF-8 with the additional
// requirement of not containing any C0 escape codes (0x00-0x1f), DEL (0x7f)
// and C1 escape codes (0x80-0x9f).
//
// Used by OSC 66 (text sizing) and OSC 99 (Kitty notifications).
//
// See: https://sw.kovidgoyal.net/kitty/desktop-notifications/#safe-utf8
func OSCEncodingIsSafeUtf8(s []uint8) bool {
	// Zig: std.unicode.Utf8View.init(s) catch return false.
	if !utf8.Valid(s) {
		return false
	}

	for _, cp := range string(s) {
		switch {
		case cp >= 0x00 && cp <= 0x1f, cp == 0x7f, cp >= 0x80 && cp <= 0x9f:
			return false
		default:
		}
	}

	return true
}
