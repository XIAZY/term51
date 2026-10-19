// Tests translated one for one from src/os/string_encoding.zig (Ghostty
// v1.3.1): upstream's, then the term51 build's after the term51 banner.
// Each Go test carries the exact Zig title in a "// Zig: test" comment. The helpers expect,
// expectEqual, expectEqualStrings and zigTry are defined in Parser_test.go
// and dcs_test.go.
//
// Translation notes:
//   - `var w: std.Io.Writer.Allocating = .init(std.testing.allocator)` is
//     `w := oscWriterAllocatingInit()` (the allocator is dropped),
//     `defer w.deinit()` is kept, `&w.writer` is `&w.writer` and
//     `w.written()` is `w.written()`.
//   - `try x` on an error union is zigTry on the error, and
//     `expectError(error.DecodeError, x)` compares the error with
//     ErrDecodeError.
//   - printf_q 8-10 copy the input into an unused NUL-terminated array
//     (`var src: [s.len:0]u8`); the copy is kept as a Go slice one byte
//     longer than s.
//   - `std.fmt.bufPrintZ(&buf_, "%{x:0>2}", .{c})` is
//     fmt.Sprintf("%%%02x", c) (and %02X for "{X:0>2}"); the result is at
//     most 3 bytes, so the [4]u8 buffer of the Zig test never overflows.
//   - Zig's defer inside a loop body runs at the end of each iteration; Go's
//     defer runs when the test function returns.

package terminal

import (
	"fmt"
	"testing"
)

// Zig: test "printf_q 1"
func TestOsStringEncoding_printf_q_1(t *testing.T) {
	w := oscWriterAllocatingInit()
	defer w.deinit()

	s := []uint8("bobr\\ kurwa")

	zigTry(t, OsStringEncodingPrintfQDecode(&w.writer, s))
	expectEqualStrings(t, "bobr kurwa", string(w.written()))
}

// Zig: test "printf_q 2"
func TestOsStringEncoding_printf_q_2(t *testing.T) {
	w := oscWriterAllocatingInit()
	defer w.deinit()

	s := []uint8("bobr\\nkurwa")

	zigTry(t, OsStringEncodingPrintfQDecode(&w.writer, s))
	expectEqualStrings(t, "bobr\nkurwa", string(w.written()))
}

// Zig: test "printf_q 3"
func TestOsStringEncoding_printf_q_3(t *testing.T) {
	w := oscWriterAllocatingInit()
	defer w.deinit()

	s := []uint8("bobr\\dkurwa")

	expectEqual(t, error(ErrDecodeError), OsStringEncodingPrintfQDecode(&w.writer, s))
}

// Zig: test "printf_q 4"
func TestOsStringEncoding_printf_q_4(t *testing.T) {
	w := oscWriterAllocatingInit()
	defer w.deinit()

	s := []uint8("bobr kurwa\\")

	expectEqual(t, error(ErrDecodeError), OsStringEncodingPrintfQDecode(&w.writer, s))
}

// Zig: test "printf_q 5"
func TestOsStringEncoding_printf_q_5(t *testing.T) {
	w := oscWriterAllocatingInit()
	defer w.deinit()

	s := []uint8("$'bobr kurwa'")

	zigTry(t, OsStringEncodingPrintfQDecode(&w.writer, s))
	expectEqualStrings(t, "bobr kurwa", string(w.written()))
}

// Zig: test "printf_q 6"
func TestOsStringEncoding_printf_q_6(t *testing.T) {
	w := oscWriterAllocatingInit()
	defer w.deinit()

	s := []uint8("'bobr kurwa'")

	zigTry(t, OsStringEncodingPrintfQDecode(&w.writer, s))
	expectEqualStrings(t, "bobr kurwa", string(w.written()))
}

// Zig: test "printf_q 7"
func TestOsStringEncoding_printf_q_7(t *testing.T) {
	w := oscWriterAllocatingInit()
	defer w.deinit()

	s := []uint8("$'bobr kurwa")

	expectEqual(t, error(ErrDecodeError), OsStringEncodingPrintfQDecode(&w.writer, s))
}

// Zig: test "printf_q 8"
func TestOsStringEncoding_printf_q_8(t *testing.T) {
	w := oscWriterAllocatingInit()
	defer w.deinit()
	s := []uint8("$'")
	src := make([]uint8, len(s)+1) // var src: [s.len:0]u8 = undefined;
	copy(src, s)                   // @memcpy(&src, s);
	_ = src
	expectEqual(t, error(ErrDecodeError), OsStringEncodingPrintfQDecode(&w.writer, s))
}

// Zig: test "printf_q 9"
func TestOsStringEncoding_printf_q_9(t *testing.T) {
	w := oscWriterAllocatingInit()
	defer w.deinit()
	s := []uint8("'bobr kurwa")
	src := make([]uint8, len(s)+1) // var src: [s.len:0]u8 = undefined;
	copy(src, s)                   // @memcpy(&src, s);
	_ = src
	expectEqual(t, error(ErrDecodeError), OsStringEncodingPrintfQDecode(&w.writer, s))
}

// Zig: test "printf_q 10"
func TestOsStringEncoding_printf_q_10(t *testing.T) {
	w := oscWriterAllocatingInit()
	defer w.deinit()
	s := []uint8("'")
	src := make([]uint8, len(s)+1) // var src: [s.len:0]u8 = undefined;
	copy(src, s)                   // @memcpy(&src, s);
	_ = src
	expectEqual(t, error(ErrDecodeError), OsStringEncodingPrintfQDecode(&w.writer, s))
}

// Zig: test "singles percent"
func TestOsStringEncoding_singles_percent(t *testing.T) {
	for c := 0; c < 255; c++ {
		w := oscWriterAllocatingInit()
		defer w.deinit()

		buf := []uint8(fmt.Sprintf("%%%02x", c))

		zigTry(t, OsStringEncodingUrlPercentDecode(&w.writer, buf))
		decoded := w.written()

		expectEqual(t, 1, len(decoded))
		expectEqual(t, uint8(c), decoded[0])
	}
	for c := 0; c < 255; c++ {
		w := oscWriterAllocatingInit()
		defer w.deinit()

		buf := []uint8(fmt.Sprintf("%%%02X", c))

		zigTry(t, OsStringEncodingUrlPercentDecode(&w.writer, buf))
		decoded := w.written()

		expectEqual(t, 1, len(decoded))
		expectEqual(t, uint8(c), decoded[0])
	}
}

// Zig: test "percent 1"
func TestOsStringEncoding_percent_1(t *testing.T) {
	w := oscWriterAllocatingInit()
	defer w.deinit()

	s := []uint8("bobr%20kurwa")

	zigTry(t, OsStringEncodingUrlPercentDecode(&w.writer, s))
	expectEqualStrings(t, "bobr kurwa", string(w.written()))
}

// Zig: test "percent 2"
func TestOsStringEncoding_percent_2(t *testing.T) {
	w := oscWriterAllocatingInit()
	defer w.deinit()

	s := []uint8("bobr%2kurwa")

	expectEqual(t, error(ErrDecodeError), OsStringEncodingUrlPercentDecode(&w.writer, s))
}

// Zig: test "percent 3"
func TestOsStringEncoding_percent_3(t *testing.T) {
	w := oscWriterAllocatingInit()
	defer w.deinit()

	s := []uint8("bobr%kurwa")

	expectEqual(t, error(ErrDecodeError), OsStringEncodingUrlPercentDecode(&w.writer, s))
}

// Zig: test "percent 4"
func TestOsStringEncoding_percent_4(t *testing.T) {
	w := oscWriterAllocatingInit()
	defer w.deinit()

	s := []uint8("bobr%%kurwa")

	expectEqual(t, error(ErrDecodeError), OsStringEncodingUrlPercentDecode(&w.writer, s))
}

// Zig: test "percent 5"
func TestOsStringEncoding_percent_5(t *testing.T) {
	w := oscWriterAllocatingInit()
	defer w.deinit()

	s := []uint8("bobr%20kurwa%20")

	zigTry(t, OsStringEncodingUrlPercentDecode(&w.writer, s))
	expectEqualStrings(t, "bobr kurwa ", string(w.written()))
}

// Zig: test "percent 6"
func TestOsStringEncoding_percent_6(t *testing.T) {
	w := oscWriterAllocatingInit()
	defer w.deinit()

	s := []uint8("bobr%20kurwa%2")

	expectEqual(t, error(ErrDecodeError), OsStringEncodingUrlPercentDecode(&w.writer, s))
}

// Zig: test "percent 7"
func TestOsStringEncoding_percent_7(t *testing.T) {
	w := oscWriterAllocatingInit()
	defer w.deinit()

	s := []uint8("bobr%20kurwa%")

	expectEqual(t, error(ErrDecodeError), OsStringEncodingUrlPercentDecode(&w.writer, s))
}

// ---------------------------------------------------------------------------
// term51: additional tests. Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

// term51ExpectPrintfQ is term51ExpectPrintfQ: decode with printfQDecode
// into an allocating writer and expect success with the given output.
func term51ExpectPrintfQ(t *testing.T, input string, expected string) {
	t.Helper()
	w := oscWriterAllocatingInit()
	defer w.deinit()
	zigTry(t, OsStringEncodingPrintfQDecode(&w.writer, []uint8(input)))
	expectEqualSlices(t, []uint8(expected), w.written())
}

// term51ExpectPrintfQError is term51ExpectPrintfQError: decode with
// printfQDecode and expect DecodeError after the given bytes were written.
func term51ExpectPrintfQError(t *testing.T, input string, written string) {
	t.Helper()
	w := oscWriterAllocatingInit()
	defer w.deinit()
	expectEqual(t, error(ErrDecodeError), OsStringEncodingPrintfQDecode(&w.writer, []uint8(input)))
	expectEqualSlices(t, []uint8(written), w.written())
}

// term51ExpectPercent is term51ExpectPercent: decode with urlPercentDecode
// into an allocating writer and expect success with the given output.
func term51ExpectPercent(t *testing.T, input string, expected string) {
	t.Helper()
	w := oscWriterAllocatingInit()
	defer w.deinit()
	zigTry(t, OsStringEncodingUrlPercentDecode(&w.writer, []uint8(input)))
	expectEqualSlices(t, []uint8(expected), w.written())
}

// term51ExpectPercentError is term51ExpectPercentError: decode with
// urlPercentDecode and expect DecodeError after the given bytes were
// written.
func term51ExpectPercentError(t *testing.T, input string, written string) {
	t.Helper()
	w := oscWriterAllocatingInit()
	defer w.deinit()
	expectEqual(t, error(ErrDecodeError), OsStringEncodingUrlPercentDecode(&w.writer, []uint8(input)))
	expectEqualSlices(t, []uint8(written), w.written())
}

// term51ExpectEncode is term51ExpectEncode: encode with urlPercentEncode
// into an allocating writer and expect the given output.
func term51ExpectEncode(t *testing.T, input string, expected string) {
	t.Helper()
	w := oscWriterAllocatingInit()
	defer w.deinit()
	zigTry(t, OsStringEncodingUrlPercentEncode(&w.writer, []uint8(input)))
	expectEqualStrings(t, expected, string(w.written()))
}

// term51IsHexDigit is term51IsHexDigit.
func term51IsHexDigit(c uint8) bool {
	switch {
	case '0' <= c && c <= '9', 'a' <= c && c <= 'f', 'A' <= c && c <= 'F':
		return true
	default:
		return false
	}
}

// Zig: test "term51: printfQDecode: empty input decodes to nothing"
func TestOsStringEncoding_term51_printfQDecode_empty_input_decodes_to_nothing(t *testing.T) {
	term51ExpectPrintfQ(t, "", "")
}

// Zig: test "term51: printfQDecode: empty quoted strings decode to nothing"
func TestOsStringEncoding_term51_printfQDecode_empty_quoted_strings_decode_to_nothing(t *testing.T) {
	term51ExpectPrintfQ(t, "''", "")
	term51ExpectPrintfQ(t, "$''", "")
}

// Zig: test "term51: printfQDecode: every supported escape"
func TestOsStringEncoding_term51_printfQDecode_every_supported_escape(t *testing.T) {
	term51ExpectPrintfQ(t, "\\ ", " ")
	term51ExpectPrintfQ(t, "\\\\", "\\")
	term51ExpectPrintfQ(t, "\\\"", "\"")
	term51ExpectPrintfQ(t, "\\'", "'")
	term51ExpectPrintfQ(t, "\\$", "$")
	term51ExpectPrintfQ(t, "\\e", "\x1b")
	term51ExpectPrintfQ(t, "\\n", "\n")
	term51ExpectPrintfQ(t, "\\r", "\r")
	term51ExpectPrintfQ(t, "\\t", "\t")
	term51ExpectPrintfQ(t, "\\v", "\x0b")
}

// Zig: test "term51: printfQDecode: every other byte after a backslash is an error"
func TestOsStringEncoding_term51_printfQDecode_every_other_byte_after_a_backslash_is_an_error(t *testing.T) {
	for i := 0; i < 256; i++ {
		c := uint8(i)
		switch c {
		case ' ', '\\', '"', '\'', '$', 'e', 'n', 'r', 't', 'v':
			continue
		}
		input := []uint8{'a', '\\', c, 'b'}
		term51ExpectPrintfQError(t, string(input), "a")
	}
}

// Zig: test "term51: printfQDecode: escapes at the start and the end of the data"
func TestOsStringEncoding_term51_printfQDecode_escapes_at_the_start_and_the_end_of_the_data(t *testing.T) {
	term51ExpectPrintfQ(t, "\\nab", "\nab")
	term51ExpectPrintfQ(t, "ab\\n", "ab\n")
	term51ExpectPrintfQ(t, "$'\\nab\\t'", "\nab\t")
	term51ExpectPrintfQ(t, "'\\nab\\t'", "\nab\t")
}

// Zig: test "term51: printfQDecode: consecutive escapes"
func TestOsStringEncoding_term51_printfQDecode_consecutive_escapes(t *testing.T) {
	term51ExpectPrintfQ(t, "\\\\\\\\", "\\\\")
	term51ExpectPrintfQ(t, "\\\\n", "\\n")
	term51ExpectPrintfQ(t, "\\e\\e\\n\\r", "\x1b\x1b\n\r")
}

// Zig: test "term51: printfQDecode: a lone trailing backslash is an error"
func TestOsStringEncoding_term51_printfQDecode_a_lone_trailing_backslash_is_an_error(t *testing.T) {
	term51ExpectPrintfQError(t, "\\", "")
	term51ExpectPrintfQError(t, "ab\\", "ab")
	term51ExpectPrintfQError(t, "\\\\\\", "\\")
	// Inside quoting the backslash before the closing quote is the last
	// byte of the data, so the quote cannot be escaped there.
	term51ExpectPrintfQError(t, "$'ab\\'", "ab")
	term51ExpectPrintfQError(t, "'ab\\'", "ab")
}

// Zig: test "term51: printfQDecode: an escaped quote before the closing quote"
func TestOsStringEncoding_term51_printfQDecode_an_escaped_quote_before_the_closing_quote(t *testing.T) {
	term51ExpectPrintfQ(t, "'ab\\''", "ab'")
	term51ExpectPrintfQ(t, "$'ab\\''", "ab'")
}

// Zig: test "term51: printfQDecode: quoting is only stripped when it starts the buffer"
func TestOsStringEncoding_term51_printfQDecode_quoting_is_only_stripped_when_it_starts_the_buffer(t *testing.T) {
	term51ExpectPrintfQ(t, "a'", "a'")
	term51ExpectPrintfQ(t, "ab''", "ab''")
	term51ExpectPrintfQ(t, "$a'", "$a'")
	term51ExpectPrintfQ(t, "$", "$")
	term51ExpectPrintfQ(t, "a$'b'", "a$'b'")
}

// Zig: test "term51: printfQDecode: only the outer quotes are stripped"
func TestOsStringEncoding_term51_printfQDecode_only_the_outer_quotes_are_stripped(t *testing.T) {
	term51ExpectPrintfQ(t, "'$'x''", "$'x'")
	term51ExpectPrintfQ(t, "$''x''", "'x'")
	term51ExpectPrintfQ(t, "''''", "''")
	term51ExpectPrintfQ(t, "$'$''", "$'")
}

// Zig: test "term51: printfQDecode: unterminated quoting is an error and writes nothing"
func TestOsStringEncoding_term51_printfQDecode_unterminated_quoting_is_an_error_and_writes_nothing(t *testing.T) {
	term51ExpectPrintfQError(t, "'", "")
	term51ExpectPrintfQError(t, "$'", "")
	term51ExpectPrintfQError(t, "'a", "")
	term51ExpectPrintfQError(t, "$'a", "")
	term51ExpectPrintfQError(t, "$'a\\n", "")
}

// Zig: test "term51: printfQDecode: unescaped bytes pass through unchanged"
func TestOsStringEncoding_term51_printfQDecode_unescaped_bytes_pass_through_unchanged(t *testing.T) {
	var all [256]uint8
	var n int
	for i := 0; i < 256; i++ {
		c := uint8(i)
		if c == '\\' || c == '\'' || c == '$' {
			continue
		}
		all[n] = c
		n += 1
	}
	term51ExpectPrintfQ(t, string(all[0:n]), string(all[0:n]))
	// Quotes and dollar signs in the middle need no escaping either.
	term51ExpectPrintfQ(t, "a'b$c\"d", "a'b$c\"d")
	// Non-ASCII and invalid UTF-8 bytes are not validated.
	term51ExpectPrintfQ(t, "\xc3\xa9\xff\x80", "\xc3\xa9\xff\x80")
}

// Zig: test "term51: printfQDecode: bytes before an invalid escape are written"
func TestOsStringEncoding_term51_printfQDecode_bytes_before_an_invalid_escape_are_written(t *testing.T) {
	term51ExpectPrintfQError(t, "ab\\ncd\\x", "ab\ncd")
	term51ExpectPrintfQError(t, "'ab\\q'", "ab")
}

// Zig: test "term51: printfQDecode: a full fixed writer fails with WriteFailed"
func TestOsStringEncoding_term51_printfQDecode_a_full_fixed_writer_fails_with_WriteFailed(t *testing.T) {
	{
		var buf [2]uint8
		w := oscWriterFixed(buf[:])
		expectEqual(t, error(ErrWriteFailed), OsStringEncodingPrintfQDecode(&w, []uint8("abc")))
		expectEqualStrings(t, "ab", string(w.buffered()))
	}
	{
		var buf [2]uint8
		w := oscWriterFixed(buf[:])
		expectEqual(t, error(ErrWriteFailed), OsStringEncodingPrintfQDecode(&w, []uint8("ab\\n")))
		expectEqualStrings(t, "ab", string(w.buffered()))
	}
	{
		// Exactly full succeeds.
		var buf [3]uint8
		w := oscWriterFixed(buf[:])
		zigTry(t, OsStringEncodingPrintfQDecode(&w, []uint8("'a\\tb'")))
		expectEqualStrings(t, "a\tb", string(w.buffered()))
	}
	{
		// A decode error is reported even when the writer is full.
		var buf [0]uint8
		w := oscWriterFixed(buf[:])
		expectEqual(t, error(ErrDecodeError), OsStringEncodingPrintfQDecode(&w, []uint8("'")))
	}
}

// Zig: test "term51: urlPercentDecode: empty input decodes to nothing"
func TestOsStringEncoding_term51_urlPercentDecode_empty_input_decodes_to_nothing(t *testing.T) {
	term51ExpectPercent(t, "", "")
}

// Zig: test "term51: urlPercentDecode: bytes other than percent pass through unchanged"
func TestOsStringEncoding_term51_urlPercentDecode_bytes_other_than_percent_pass_through_unchanged(t *testing.T) {
	var all [255]uint8
	var n int
	for i := 0; i < 256; i++ {
		c := uint8(i)
		if c == '%' {
			continue
		}
		all[n] = c
		n += 1
	}
	term51ExpectPercent(t, string(all[0:n]), string(all[0:n]))
	// A plus sign is not a space.
	term51ExpectPercent(t, "a+b", "a+b")
}

// Zig: test "term51: urlPercentDecode: byte 255 in both cases"
func TestOsStringEncoding_term51_urlPercentDecode_byte_255_in_both_cases(t *testing.T) {
	// Upstream's "singles percent" loops over 0..255, which excludes 255.
	term51ExpectPercent(t, "%ff", "\xff")
	term51ExpectPercent(t, "%FF", "\xff")
	term51ExpectPercent(t, "%fF", "\xff")
	term51ExpectPercent(t, "%Ff", "\xff")
}

// Zig: test "term51: urlPercentDecode: every pair of bytes after a percent sign"
func TestOsStringEncoding_term51_urlPercentDecode_every_pair_of_bytes_after_a_percent_sign(t *testing.T) {
	for i := 0; i < 256; i++ {
		for j := 0; j < 256; j++ {
			a := uint8(i)
			b := uint8(j)
			input := []uint8{'x', '%', a, b, 'y'}
			if term51IsHexDigit(a) && term51IsHexDigit(b) {
				v := osStringEncodingHex(a)<<4 | osStringEncodingHex(b)
				term51ExpectPercent(t, string(input), string([]uint8{'x', v, 'y'}))
			} else {
				term51ExpectPercentError(t, string(input), "x")
			}
		}
	}
}

// Zig: test "term51: urlPercentDecode: escapes cut short at the end"
func TestOsStringEncoding_term51_urlPercentDecode_escapes_cut_short_at_the_end(t *testing.T) {
	term51ExpectPercentError(t, "%", "")
	term51ExpectPercentError(t, "%2", "")
	term51ExpectPercentError(t, "%g", "")
	term51ExpectPercentError(t, "ab%", "ab")
	term51ExpectPercentError(t, "ab%4", "ab")
	term51ExpectPercentError(t, "%41%", "A")
	term51ExpectPercentError(t, "%41%4", "A")
	// A complete escape as the last three bytes is fine.
	term51ExpectPercent(t, "%41", "A")
	term51ExpectPercent(t, "ab%41", "abA")
}

// Zig: test "term51: urlPercentDecode: invalid hex digits"
func TestOsStringEncoding_term51_urlPercentDecode_invalid_hex_digits(t *testing.T) {
	term51ExpectPercentError(t, "%g0", "")
	term51ExpectPercentError(t, "%0g", "")
	term51ExpectPercentError(t, "%G0", "")
	term51ExpectPercentError(t, "% 1", "")
	term51ExpectPercentError(t, "%1 ", "")
	term51ExpectPercentError(t, "%+1", "")
	term51ExpectPercentError(t, "%-1", "")
	term51ExpectPercentError(t, "%x1", "")
	term51ExpectPercentError(t, "%%41", "")
	term51ExpectPercentError(t, "a%2%41", "a")
}

// Zig: test "term51: urlPercentDecode: decoded bytes are not decoded again"
func TestOsStringEncoding_term51_urlPercentDecode_decoded_bytes_are_not_decoded_again(t *testing.T) {
	term51ExpectPercent(t, "%2541", "%41")
	term51ExpectPercent(t, "%25%25", "%%")
	term51ExpectPercent(t, "%00", "\x00")
}

// Zig: test "term51: urlPercentDecode: a full fixed writer fails with WriteFailed"
func TestOsStringEncoding_term51_urlPercentDecode_a_full_fixed_writer_fails_with_WriteFailed(t *testing.T) {
	{
		var buf [2]uint8
		w := oscWriterFixed(buf[:])
		expectEqual(t, error(ErrWriteFailed), OsStringEncodingUrlPercentDecode(&w, []uint8("ab%41")))
		expectEqualStrings(t, "ab", string(w.buffered()))
	}
	{
		var buf [3]uint8
		w := oscWriterFixed(buf[:])
		zigTry(t, OsStringEncodingUrlPercentDecode(&w, []uint8("a%20b")))
		expectEqualStrings(t, "a b", string(w.buffered()))
	}
	{
		var buf [0]uint8
		w := oscWriterFixed(buf[:])
		expectEqual(t, error(ErrDecodeError), OsStringEncodingUrlPercentDecode(&w, []uint8("%4")))
	}
}

// Zig: test "term51: hex: every hex digit"
func TestOsStringEncoding_term51_hex_every_hex_digit(t *testing.T) {
	const digits = "0123456789abcdef"
	for i := 0; i < len(digits); i++ {
		c := digits[i]
		expectEqual(t, uint8(i), osStringEncodingHex(c))
		// std.ascii.toUpper
		expectEqual(t, uint8(i), osStringEncodingHex(term51ToUpper(c)))
	}
}

// term51ToUpper is std.ascii.toUpper, used by the test above.
func term51ToUpper(c uint8) uint8 {
	if 'a' <= c && c <= 'z' {
		return c - 32
	}
	return c
}

// Zig: test "term51: isValidChar: every byte"
func TestOsStringEncoding_term51_isValidChar_every_byte(t *testing.T) {
	for i := 0; i < 256; i++ {
		c := uint8(i)
		expected := c >= 0x21 && c <= 0x7e && c != ';' && c != '='
		expectEqual(t, expected, osStringEncodingIsValidChar(c))
	}
	expect(t, !osStringEncodingIsValidChar(' '), "!isValidChar(' ')")
	expect(t, !osStringEncodingIsValidChar(';'), "!isValidChar(';')")
	expect(t, !osStringEncodingIsValidChar('='), "!isValidChar('=')")
	expect(t, !osStringEncodingIsValidChar(0x7f), "!isValidChar(0x7f)")
	expect(t, !osStringEncodingIsValidChar(0x80), "!isValidChar(0x80)")
	expect(t, osStringEncodingIsValidChar('%'), "isValidChar('%')")
	expect(t, osStringEncodingIsValidChar('~'), "isValidChar('~')")
	expect(t, osStringEncodingIsValidChar('!'), "isValidChar('!')")
}

// Zig: test "term51: urlPercentEncode: empty input encodes to nothing"
func TestOsStringEncoding_term51_urlPercentEncode_empty_input_encodes_to_nothing(t *testing.T) {
	term51ExpectEncode(t, "", "")
}

// Zig: test "term51: urlPercentEncode: valid characters pass through"
func TestOsStringEncoding_term51_urlPercentEncode_valid_characters_pass_through(t *testing.T) {
	term51ExpectEncode(t, "abc/:?#&%+~", "abc/:?#&%+~")
}

// Zig: test "term51: urlPercentEncode: invalid characters use upper-case two-digit escapes"
func TestOsStringEncoding_term51_urlPercentEncode_invalid_characters_use_upper_case_two_digit_escapes(t *testing.T) {
	term51ExpectEncode(t, " ;=", "%20%3B%3D")
	term51ExpectEncode(t, "\x00", "%00")
	term51ExpectEncode(t, "\x0a", "%0A")
	term51ExpectEncode(t, "\x7f", "%7F")
	term51ExpectEncode(t, "\xff", "%FF")
	term51ExpectEncode(t, "\xc3\xa9", "%C3%A9")
}

// Zig: test "term51: urlPercentEncode: escapes at the start, middle and end"
func TestOsStringEncoding_term51_urlPercentEncode_escapes_at_the_start_middle_and_end(t *testing.T) {
	term51ExpectEncode(t, " ab", "%20ab")
	term51ExpectEncode(t, "a b", "a%20b")
	term51ExpectEncode(t, "ab ", "ab%20")
	term51ExpectEncode(t, "a  b", "a%20%20b")
}

// Zig: test "term51: urlPercentEncode: every byte"
func TestOsStringEncoding_term51_urlPercentEncode_every_byte(t *testing.T) {
	for i := 0; i < 256; i++ {
		c := uint8(i)
		var expected string
		if osStringEncodingIsValidChar(c) {
			expected = string([]uint8{c})
		} else {
			// std.fmt.bufPrint(&expected_buf, "%{X:0>2}", .{c})
			expected = fmt.Sprintf("%%%02X", c)
		}
		w := oscWriterAllocatingInit()
		defer w.deinit()
		zigTry(t, OsStringEncodingUrlPercentEncode(&w.writer, []uint8{c}))
		expectEqualStrings(t, expected, string(w.written()))
	}
}

// Zig: test "term51: urlPercentEncode: decoding the encoding gives back every byte except the percent sign"
func TestOsStringEncoding_term51_urlPercentEncode_decoding_the_encoding_gives_back_every_byte_except_the_percent_sign(t *testing.T) {
	var all [255]uint8
	var n int
	for i := 0; i < 256; i++ {
		c := uint8(i)
		if c == '%' {
			continue
		}
		all[n] = c
		n += 1
	}
	enc := oscWriterAllocatingInit()
	defer enc.deinit()
	zigTry(t, OsStringEncodingUrlPercentEncode(&enc.writer, all[0:n]))
	dec := oscWriterAllocatingInit()
	defer dec.deinit()
	zigTry(t, OsStringEncodingUrlPercentDecode(&dec.writer, enc.written()))
	expectEqualSlices(t, all[0:n], dec.written())
}

// Zig: test "term51: urlPercentEncode: the percent sign is not escaped, so the result may not decode"
func TestOsStringEncoding_term51_urlPercentEncode_the_percent_sign_is_not_escaped_so_the_result_may_not_decode(t *testing.T) {
	// isValidChar accepts '%' (it is printable), so urlPercentEncode writes
	// it unescaped and urlPercentDecode then reads it as an escape.
	term51ExpectEncode(t, "%", "%")
	term51ExpectEncode(t, "100% sure", "100%%20sure")
	term51ExpectPercentError(t, "100%%20sure", "100")
	term51ExpectEncode(t, "%41", "%41")
	term51ExpectPercent(t, "%41", "A")
}

// Zig: test "term51: urlPercentEncode: a full fixed writer fails with WriteFailed"
func TestOsStringEncoding_term51_urlPercentEncode_a_full_fixed_writer_fails_with_WriteFailed(t *testing.T) {
	{
		var buf [4]uint8
		w := oscWriterFixed(buf[:])
		expectEqual(t, error(ErrWriteFailed), OsStringEncodingUrlPercentEncode(&w, []uint8("ab c")))
		expectEqualStrings(t, "ab%2", string(w.buffered()))
	}
	{
		var buf [2]uint8
		w := oscWriterFixed(buf[:])
		expectEqual(t, error(ErrWriteFailed), OsStringEncodingUrlPercentEncode(&w, []uint8("abc")))
		expectEqualStrings(t, "ab", string(w.buffered()))
	}
	{
		var buf [5]uint8
		w := oscWriterFixed(buf[:])
		expectEqual(t, error(ErrWriteFailed), OsStringEncodingUrlPercentEncode(&w, []uint8("ab c")))
		expectEqualStrings(t, "ab%20", string(w.buffered()))
	}
	{
		// Exactly full succeeds.
		var buf [6]uint8
		w := oscWriterFixed(buf[:])
		zigTry(t, OsStringEncodingUrlPercentEncode(&w, []uint8("ab c")))
		expectEqualStrings(t, "ab%20c", string(w.buffered()))
	}
}
