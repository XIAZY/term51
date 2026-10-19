// Ported from Zig 0.15.2's lib/std/unicode.zig: only the function the port
// reaches (render.zig's RenderState.string calls
// std.unicode.utf8CodepointSequenceLength). Go's utf8.RuneLen is not the
// same function: it returns -1 for surrogate halves, for which Zig returns
// 3. std's unicode.zig has no test block for this function (it is only
// exercised through utf8Encode); there is no matching test file.

package terminal

import "errors"

// ErrCodepointTooLarge is std.unicode's error.CodepointTooLarge.
var ErrCodepointTooLarge = errors.New("CodepointTooLarge")

// stdUnicodeUtf8CodepointSequenceLength is
// std.unicode.utf8CodepointSequenceLength.
//
// Returns how many bytes the UTF-8 representation would require
// for the given codepoint.
//
// Zig: pub fn utf8CodepointSequenceLength(c: u21) !u3.
func stdUnicodeUtf8CodepointSequenceLength(c uint32) (uint8, error) {
	if c < 0x80 {
		return 1, nil
	}
	if c < 0x800 {
		return 2, nil
	}
	if c < 0x10000 {
		return 3, nil
	}
	if c < 0x110000 {
		return 4, nil
	}
	return 0, ErrCodepointTooLarge
}
