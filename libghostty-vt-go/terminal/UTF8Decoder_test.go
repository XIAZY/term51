// Tests translated one for one from src/terminal/UTF8Decoder.zig (Ghostty
// v1.3.1 plus the term51 additional tests). Each Go test carries the exact
// Zig title in a "// Zig: test" comment.
//
// The helpers expect, expectEqual, expectEqualSlices, expectEqualStrings and
// zigUnwrap are in Parser_test.go; they stand for Zig's testing functions and
// for `x.?`.
//
// Zig's `d.next(byte)` returns a tuple `res`; Go returns two values, named
// res0 and res1 here for res[0] and res[1].

package terminal

import (
	"testing"
	"unicode/utf8"
)

// Zig: test "ASCII"
func TestUTF8Decoder_ASCII(t *testing.T) {
	var d UTF8Decoder
	var out [13]uint8
	for i, byte_ := range []uint8("Hello, World!") {
		res0, res1 := d.Next(byte_)
		expect(t, res1, "res[1]")
		if res0.Valid {
			codepoint := res0.Value
			out[i] = uint8(codepoint)
		}
	}

	// Zig passes these two in this order.
	expectEqualStrings(t, string(out[:]), "Hello, World!")
}

// Zig: test "Well formed utf-8"
func TestUTF8Decoder_Well_formed_utf_8(t *testing.T) {
	var d UTF8Decoder
	var out [4]uint32
	i := 0
	// 4 bytes, 3 bytes, 2 bytes, 1 byte
	for _, byte_ := range []uint8("😄✤ÁA") {
		consumed := false
		for !consumed {
			res0, res1 := d.Next(byte_)
			consumed = res1
			// There are no errors in this sequence, so
			// every byte should be consumed first try.
			expect(t, consumed == true, "consumed == true")
			if res0.Valid {
				codepoint := res0.Value
				out[i] = codepoint
				i += 1
			}
		}
	}

	expect(t, out == [4]uint32{0x1F604, 0x2724, 0xC1, 0x41},
		"std.mem.eql(u21, &out, &[_]u21{ 0x1F604, 0x2724, 0xC1, 0x41 })")
}

// Zig: test "Partially invalid utf-8"
func TestUTF8Decoder_Partially_invalid_utf_8(t *testing.T) {
	var d UTF8Decoder
	var out [5]uint32
	i := 0
	// Illegally terminated sequence, valid sequence, illegal surrogate pair.
	for _, byte_ := range []uint8("\xF0\x9F😄\xED\xA0\x80") {
		consumed := false
		for !consumed {
			res0, res1 := d.Next(byte_)
			consumed = res1
			if res0.Valid {
				codepoint := res0.Value
				out[i] = codepoint
				i += 1
			}
		}
	}

	expect(t, out == [5]uint32{0xFFFD, 0x1F604, 0xFFFD, 0xFFFD, 0xFFFD},
		"std.mem.eql(u21, &out, &[_]u21{ 0xFFFD, 0x1F604, 0xFFFD, 0xFFFD, 0xFFFD })")
}

// ---------------------------------------------------------------------------
// term51: additional tests. Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

// term51 test helper: decode a whole byte string the way the stream does,
// re-feeding a byte when the decoder reports it was not consumed.
func term51Decode(bytes string, out []uint32) []uint32 {
	var d UTF8Decoder
	n := 0
	for _, byte_ := range []uint8(bytes) {
		consumed := false
		for !consumed {
			res0, res1 := d.Next(byte_)
			consumed = res1
			if res0.Valid {
				cp := res0.Value
				out[n] = cp
				n += 1
			}
		}
	}
	return out[0:n]
}

// Zig: test "term51: next: every scalar value round-trips"
func TestUTF8Decoder_term51_next_every_scalar_value_round_trips(t *testing.T) {
	var d UTF8Decoder
	var cp uint32 = 0
	for ; true; cp += 1 {
		if cp >= 0xD800 && cp <= 0xDFFF {
			if cp == 0x10FFFF {
				break
			}
			continue
		}
		var buf [4]uint8
		// Zig: const len = try std.unicode.utf8Encode(cp, &buf);
		len_ := utf8.EncodeRune(buf[:], rune(cp))
		for i, byte_ := range buf[0:len_] {
			res0, res1 := d.Next(byte_)
			expect(t, res1, "res[1]")
			if i+1 < len_ {
				expect(t, !res0.Valid, "res[0] == null")
			} else {
				expectEqual(t, cp, zigUnwrap(t, res0))
			}
		}
		expectEqual(t, uint8(ACCEPT_STATE), d.state)
		expectEqual(t, uint32(0), d.accumulator)
		if cp == 0x10FFFF {
			break
		}
	}
}

// Zig: test "term51: next: encoding length boundaries"
func TestUTF8Decoder_term51_next_encoding_length_boundaries(t *testing.T) {
	var out [8]uint32
	expectEqualSlices(t, []uint32{0x7F}, term51Decode("\x7f", out[:]))
	expectEqualSlices(t, []uint32{0x80}, term51Decode("\xc2\x80", out[:]))
	expectEqualSlices(t, []uint32{0x7FF}, term51Decode("\xdf\xbf", out[:]))
	expectEqualSlices(t, []uint32{0x800}, term51Decode("\xe0\xa0\x80", out[:]))
	expectEqualSlices(t, []uint32{0xD7FF}, term51Decode("\xed\x9f\xbf", out[:]))
	expectEqualSlices(t, []uint32{0xE000}, term51Decode("\xee\x80\x80", out[:]))
	expectEqualSlices(t, []uint32{0xFFFF}, term51Decode("\xef\xbf\xbf", out[:]))
	expectEqualSlices(t, []uint32{0x10000}, term51Decode("\xf0\x90\x80\x80", out[:]))
	expectEqualSlices(t, []uint32{0x10FFFF}, term51Decode("\xf4\x8f\xbf\xbf", out[:]))
}

// Zig: test "term51: next: a byte that can never start a sequence is consumed and replaced"
func TestUTF8Decoder_term51_next_a_byte_that_can_never_start_a_sequence_is_consumed_and_replaced(t *testing.T) {
	for _, byte_ := range []uint8{0x80, 0x8F, 0x90, 0xBF, 0xC0, 0xC1, 0xF5, 0xF8, 0xFC, 0xFE, 0xFF} {
		var d UTF8Decoder
		res0, res1 := d.Next(byte_)
		expectEqual(t, uint32(0xFFFD), zigUnwrap(t, res0))
		expect(t, res1, "res[1]") // consumed
		expectEqual(t, uint8(ACCEPT_STATE), d.state)
		expectEqual(t, uint32(0), d.accumulator)
	}
}

// Zig: test "term51: next: a bad continuation is replaced and not consumed"
func TestUTF8Decoder_term51_next_a_bad_continuation_is_replaced_and_not_consumed(t *testing.T) {
	for _, prefix := range []string{"\xc3", "\xe4", "\xe4\xb8", "\xf0", "\xf0\x9f", "\xf0\x9f\x98"} {
		var d UTF8Decoder
		for _, byte_ := range []uint8(prefix) {
			res0, res1 := d.Next(byte_)
			expect(t, !res0.Valid, "res[0] == null")
			expect(t, res1, "res[1]")
		}
		res0, res1 := d.Next('A')
		expectEqual(t, uint32(0xFFFD), zigUnwrap(t, res0))
		expect(t, !res1, "!res[1]") // not consumed: caller must feed it again
		expectEqual(t, uint8(ACCEPT_STATE), d.state)
		again0, again1 := d.Next('A')
		expectEqual(t, uint32('A'), zigUnwrap(t, again0))
		expect(t, again1, "again[1]")
	}
}

// Zig: test "term51: next: one replacement per truncated sequence, not per byte"
func TestUTF8Decoder_term51_next_one_replacement_per_truncated_sequence_not_per_byte(t *testing.T) {
	var out [16]uint32
	expectEqualSlices(t, []uint32{0xFFFD, '|'}, term51Decode("\xe4\xb8|", out[:]))
	expectEqualSlices(t, []uint32{0xFFFD, '|'}, term51Decode("\xf0\x9f\x98|", out[:]))
	expectEqualSlices(t, []uint32{0xFFFD, '|'}, term51Decode("\xc3|", out[:]))
	// a truncated sequence followed by a valid one
	expectEqualSlices(t, []uint32{0xFFFD, 0xE9}, term51Decode("\xe4\xb8\xc3\xa9", out[:]))
	// two truncated sequences in a row
	expectEqualSlices(t, []uint32{0xFFFD, 0xFFFD, '|'}, term51Decode("\xe4\xb8\xe4\xb8|", out[:]))
}

// Zig: test "term51: next: overlong encodings are rejected"
func TestUTF8Decoder_term51_next_overlong_encodings_are_rejected(t *testing.T) {
	var out [16]uint32
	// C0 and C1 can never be valid lead bytes
	expectEqualSlices(t, []uint32{0xFFFD, 0xFFFD}, term51Decode("\xc0\x80", out[:]))
	expectEqualSlices(t, []uint32{0xFFFD, 0xFFFD}, term51Decode("\xc1\xbf", out[:]))
	// E0 followed by a continuation below A0
	expectEqualSlices(t, []uint32{0xFFFD, 0xFFFD, 0xFFFD}, term51Decode("\xe0\x80\x80", out[:]))
	expectEqualSlices(t, []uint32{0xFFFD, 0xFFFD, 0xFFFD}, term51Decode("\xe0\x9f\xbf", out[:]))
	// F0 followed by a continuation below 90
	expectEqualSlices(t, []uint32{0xFFFD, 0xFFFD, 0xFFFD, 0xFFFD}, term51Decode("\xf0\x80\x80\x80", out[:]))
	expectEqualSlices(t, []uint32{0xFFFD, 0xFFFD, 0xFFFD, 0xFFFD}, term51Decode("\xf0\x8f\xbf\xbf", out[:]))
}

// Zig: test "term51: next: surrogates and values above U+10FFFF are rejected"
func TestUTF8Decoder_term51_next_surrogates_and_values_above_U_10FFFF_are_rejected(t *testing.T) {
	var out [16]uint32
	expectEqualSlices(t, []uint32{0xFFFD, 0xFFFD, 0xFFFD}, term51Decode("\xed\xa0\x80", out[:]))
	expectEqualSlices(t, []uint32{0xFFFD, 0xFFFD, 0xFFFD}, term51Decode("\xed\xbf\xbf", out[:]))
	expectEqualSlices(t, []uint32{0xFFFD, 0xFFFD, 0xFFFD, 0xFFFD}, term51Decode("\xf4\x90\x80\x80", out[:]))
	expectEqualSlices(t, []uint32{0xFFFD, 0xFFFD, 0xFFFD, 0xFFFD}, term51Decode("\xf5\x80\x80\x80", out[:]))
}

// Zig: test "term51: next: the example from the SIMD comparison"
func TestUTF8Decoder_term51_next_the_example_from_the_SIMD_comparison(t *testing.T) {
	var out [16]uint32
	expectEqualSlices(t, []uint32{0xFFFD, 0xFFFD, 0xFFFD, 0x09}, term51Decode("\xf6\xa2\xca\x09", out[:]))
}

// Zig: test "term51: next: a lead byte interrupting a sequence starts a new one"
func TestUTF8Decoder_term51_next_a_lead_byte_interrupting_a_sequence_starts_a_new_one(t *testing.T) {
	var out [16]uint32
	expectEqualSlices(t, []uint32{0xFFFD, 0xE9}, term51Decode("\xc3\xc3\xa9", out[:]))
	expectEqualSlices(t, []uint32{0xFFFD, 0x4E2D}, term51Decode("\xe4\xe4\xb8\xad", out[:]))
}

// Zig: test "term51: next: control bytes interrupting a sequence are not swallowed"
func TestUTF8Decoder_term51_next_control_bytes_interrupting_a_sequence_are_not_swallowed(t *testing.T) {
	var out [16]uint32
	expectEqualSlices(t, []uint32{0xFFFD, 0x1B}, term51Decode("\xe4\x1b", out[:]))
	expectEqualSlices(t, []uint32{0xFFFD, 0x0A}, term51Decode("\xe4\xb8\x0a", out[:]))
	expectEqualSlices(t, []uint32{0xFFFD, 0x00}, term51Decode("\xc3\x00", out[:]))
}

// Zig: test "term51: next: decoder is reusable after an error"
func TestUTF8Decoder_term51_next_decoder_is_reusable_after_an_error(t *testing.T) {
	var d UTF8Decoder
	_, _ = d.Next(0xFF)
	a0, _ := d.Next('a')
	expectEqual(t, uint32('a'), zigUnwrap(t, a0))
	_, _ = d.Next(0xE4)
	_, _ = d.Next(0x28) // rejected, not consumed
	b0, _ := d.Next(0xC3)
	expect(t, !b0.Valid, "b[0] == null")
	c0, _ := d.Next(0xA9)
	expectEqual(t, uint32(0xE9), zigUnwrap(t, c0))
}

// Zig: test "term51: next: character class table covers every byte"
func TestUTF8Decoder_term51_next_character_class_table_covers_every_byte(t *testing.T) {
	expectEqual(t, 256, len(char_classes))
	for b := 0; b < 0x80; b++ {
		expectEqual(t, uint8(0), char_classes[b])
	}
	for b := 0x80; b < 0x90; b++ {
		expectEqual(t, uint8(1), char_classes[b])
	}
	for b := 0x90; b < 0xA0; b++ {
		expectEqual(t, uint8(9), char_classes[b])
	}
	for b := 0xA0; b < 0xC0; b++ {
		expectEqual(t, uint8(7), char_classes[b])
	}
	expectEqual(t, uint8(8), char_classes[0xC0])
	expectEqual(t, uint8(8), char_classes[0xC1])
	for b := 0xC2; b < 0xE0; b++ {
		expectEqual(t, uint8(2), char_classes[b])
	}
	expectEqual(t, uint8(10), char_classes[0xE0])
	expectEqual(t, uint8(4), char_classes[0xED])
	expectEqual(t, uint8(11), char_classes[0xF0])
	expectEqual(t, uint8(5), char_classes[0xF4])
	for b := 0xF5; b < 0x100; b++ {
		expectEqual(t, uint8(8), char_classes[b])
	}
}
