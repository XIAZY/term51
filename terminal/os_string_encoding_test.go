// Tests translated one for one from src/os/string_encoding.zig (Ghostty
// v1.3.1; the term51 build adds no tests to this file). Each Go test carries
// the exact Zig title in a "// Zig: test" comment. The helpers expect,
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
