// Tests translated one for one from src/input/paste.zig (Ghostty v1.3.1 plus
// the term51 additional tests). Each Go test carries the exact Zig title in a
// "// Zig: test" comment. Helpers are prefixed kp.
//
// Translation notes:
//   - Zig's generic `encode` is two Go functions (see input_paste.go):
//     a call whose argument is `[]const u8` (a string literal, or
//     `@as([]const u8, ...)`, or a variable of that type) is
//     InputPasteEncode; a call whose argument is a `[]u8` (a dupe, or a
//     mutable buffer) is InputPasteEncodeMutable. Go has no const slices, so
//     a Zig `[]const u8` is a []uint8 that the test never expects to change
//     (the tests that check it say so).
//   - `try testing.expectError(Error.MutableRequired, encode(...))` is
//     kpPasteExpectMutableRequired.
//   - `std.testing.allocator.dupe(u8, s)` is `[]uint8(s)` (a copy);
//     `defer free` leak checks are dropped with `// Zig-only:` comments.
//   - `@intFromPtr(a.ptr) == @intFromPtr(b.ptr)` is kpPasteSamePtr
//     (unsafe.SliceData).
//   - `var t = try Terminal.init(alloc, .{ .cols = 10, .rows = 5 })` is
//     `term` (the testing parameter is `t`), started from
//     terminalOptionsDefault; `.default_modes = .{ .bracketed_paste = true }`
//     is ModesModePackedInit() with bracketed_paste set (the other modes keep
//     their Zig defaults).
//   - Tests that were not translated are listed in
//     scripts/not-translated-kp.txt.

package terminal

import (
	"testing"
	"unsafe"
)

// kpPasteIsStrip is term51AEIsStrip: bytes that encode() replaces with a
// space (copied from the source list).
func kpPasteIsStrip(b uint8) bool {
	switch b {
	case 0x00, 0x08, 0x05, 0x04, 0x1B, 0x7F:
		return true
	case 0x03, 0x1C, 0x15, 0x1A, 0x11, 0x13, 0x17, 0x16, 0x12, 0x0F:
		return true
	}
	return false
}

// kpPasteStripList is term51AEStripList.
var kpPasteStripList = [...]uint8{
	0x00, 0x08, 0x05, 0x04, 0x1B, 0x7F, 0x03, 0x1C,
	0x15, 0x1A, 0x11, 0x13, 0x17, 0x16, 0x12, 0x0F,
}

// kpPasteJoin is term51AEJoin: concatenates the three slices of an encode
// result.
func kpPasteJoin(r [3][]uint8) []uint8 {
	var out []uint8
	out = append(out, r[0]...)
	out = append(out, r[1]...)
	out = append(out, r[2]...)
	return out
}

// kpPasteSamePtr is `@intFromPtr(a.ptr) == @intFromPtr(b.ptr)`.
func kpPasteSamePtr(a []uint8, b []uint8) bool {
	return unsafe.SliceData(a) == unsafe.SliceData(b)
}

// kpPasteExpectMutableRequired is `try testing.expectError(
// Error.MutableRequired, encode(data, opts))` for `[]const u8` data.
func kpPasteExpectMutableRequired(tt *testing.T, data []uint8, bracketed bool) {
	tt.Helper()
	_, err := InputPasteEncode(data, InputPasteOptions{bracketed: bracketed})
	if err != ErrMutableRequired {
		tt.Fatalf("expected error MutableRequired, found %v", err)
	}
}

// kpPasteExpectMutable is term51AEExpectMutable: runs encode on a mutable
// copy of `input`; asserts the three slices.
func kpPasteExpectMutable(
	tt *testing.T,
	input string,
	bracketed bool,
	want0 string,
	want1 string,
	want2 string,
) {
	tt.Helper()
	data := []uint8(input)
	// Zig-only: defer std.testing.allocator.free(data)
	r := InputPasteEncodeMutable(data, InputPasteOptions{bracketed: bracketed})
	expectEqualStrings(tt, want0, string(r[0]))
	expectEqualStrings(tt, want1, string(r[1]))
	expectEqualStrings(tt, want2, string(r[2]))
	// The middle slice is the caller's own buffer, edited in place.
	expect(tt, kpPasteSamePtr(data, r[1]), "@intFromPtr(data.ptr) == @intFromPtr(r[1].ptr)")
	expectEqual(tt, len(data), len(r[1]))
}

// kpPasteTerminal is `try Terminal.init(alloc, .{ .cols = 10, .rows = 5 })`
// with the Zig defaults, plus any option change made by modify.
func kpPasteTerminal(tt *testing.T, modify func(opts *TerminalOptions)) *Terminal {
	tt.Helper()
	opts := terminalOptionsDefault(10, 5)
	if modify != nil {
		modify(&opts)
	}
	term, err := TerminalInit(opts)
	zigTry(tt, err)
	return &term
}

// Zig: test isSafe
func TestKpPaste_isSafe(t *testing.T) {
	expect(t, InputPasteIsSafe([]uint8("hello")), `isSafe("hello")`)
	expect(t, !InputPasteIsSafe([]uint8("hello\n")), `!isSafe("hello\n")`)
	expect(t, !InputPasteIsSafe([]uint8("hello\nworld")), `!isSafe("hello\nworld")`)
	expect(t, !InputPasteIsSafe([]uint8("he\x1b[201~llo")), `!isSafe("he\x1b[201~llo")`)
}

// Zig: test "encode bracketed"
func TestKpPaste_encode_bracketed(t *testing.T) {
	result, err := InputPasteEncode(
		[]uint8("hello"),
		InputPasteOptions{bracketed: true},
	)
	zigTry(t, err)
	expectEqualStrings(t, "\x1b[200~", string(result[0]))
	expectEqualStrings(t, "hello", string(result[1]))
	expectEqualStrings(t, "\x1b[201~", string(result[2]))
}

// Zig: test "encode unbracketed no newlines"
func TestKpPaste_encode_unbracketed_no_newlines(t *testing.T) {
	result, err := InputPasteEncode(
		[]uint8("hello"),
		InputPasteOptions{bracketed: false},
	)
	zigTry(t, err)
	expectEqualStrings(t, "", string(result[0]))
	expectEqualStrings(t, "hello", string(result[1]))
	expectEqualStrings(t, "", string(result[2]))
}

// Zig: test "encode unbracketed newlines const"
func TestKpPaste_encode_unbracketed_newlines_const(t *testing.T) {
	kpPasteExpectMutableRequired(t, []uint8("hello\nworld"), false)
}

// Zig: test "encode unbracketed newlines"
func TestKpPaste_encode_unbracketed_newlines(t *testing.T) {
	data := []uint8("hello\nworld")
	// Zig-only: defer testing.allocator.free(data)
	result := InputPasteEncodeMutable(data, InputPasteOptions{bracketed: false})
	expectEqualStrings(t, "", string(result[0]))
	expectEqualStrings(t, "hello\rworld", string(result[1]))
	expectEqualStrings(t, "", string(result[2]))
}

// Zig: test "encode unbracketed windows-stye newline"
func TestKpPaste_encode_unbracketed_windows_stye_newline(t *testing.T) {
	data := []uint8("hello\r\nworld")
	// Zig-only: defer testing.allocator.free(data)
	result := InputPasteEncodeMutable(data, InputPasteOptions{bracketed: false})
	expectEqualStrings(t, "", string(result[0]))
	expectEqualStrings(t, "hello\r\rworld", string(result[1]))
	expectEqualStrings(t, "", string(result[2]))
}

// Zig: test "encode strip unsafe bytes const"
func TestKpPaste_encode_strip_unsafe_bytes_const(t *testing.T) {
	kpPasteExpectMutableRequired(t, []uint8("hello\x00world"), true)
}

// Zig: test "encode strip unsafe bytes mutable bracketed"
func TestKpPaste_encode_strip_unsafe_bytes_mutable_bracketed(t *testing.T) {
	data := []uint8("hel\x1blo\x00world")
	// Zig-only: defer testing.allocator.free(data)
	result := InputPasteEncodeMutable(data, InputPasteOptions{bracketed: true})
	expectEqualStrings(t, "\x1b[200~", string(result[0]))
	expectEqualStrings(t, "hel lo world", string(result[1]))
	expectEqualStrings(t, "\x1b[201~", string(result[2]))
}

// Zig: test "encode strip unsafe bytes mutable unbracketed"
func TestKpPaste_encode_strip_unsafe_bytes_mutable_unbracketed(t *testing.T) {
	data := []uint8("hel\x03lo")
	// Zig-only: defer testing.allocator.free(data)
	result := InputPasteEncodeMutable(data, InputPasteOptions{bracketed: false})
	expectEqualStrings(t, "", string(result[0]))
	expectEqualStrings(t, "hel lo", string(result[1]))
	expectEqualStrings(t, "", string(result[2]))
}

// Zig: test "encode strip multiple unsafe bytes"
func TestKpPaste_encode_strip_multiple_unsafe_bytes(t *testing.T) {
	data := []uint8("\x00\x08\x7f")
	// Zig-only: defer testing.allocator.free(data)
	result := InputPasteEncodeMutable(data, InputPasteOptions{bracketed: true})
	expectEqualStrings(t, "   ", string(result[1]))
}

// Zig: test "term51: Options.fromTerminal: default terminal has bracketed paste off"
func TestKpPaste_Options_fromTerminal_default_terminal_has_bracketed_paste_off(t *testing.T) {
	term := kpPasteTerminal(t, nil)
	defer term.Deinit()
	expectEqual(t, false, InputPasteOptionsFromTerminal(term).bracketed)
}

// Zig: test "term51: Options.fromTerminal: follows the bracketed paste mode"
func TestKpPaste_Options_fromTerminal_follows_the_bracketed_paste_mode(t *testing.T) {
	term := kpPasteTerminal(t, nil)
	defer term.Deinit()

	term.modes.Set(ModesModeBracketedPaste, true)
	expectEqual(t, true, InputPasteOptionsFromTerminal(term).bracketed)
	// Idempotent.
	term.modes.Set(ModesModeBracketedPaste, true)
	expectEqual(t, true, InputPasteOptionsFromTerminal(term).bracketed)

	term.modes.Set(ModesModeBracketedPaste, false)
	expectEqual(t, false, InputPasteOptionsFromTerminal(term).bracketed)

	term.modes.Set(ModesModeBracketedPaste, true)
	expectEqual(t, true, InputPasteOptionsFromTerminal(term).bracketed)
}

// Zig: test "term51: Options.fromTerminal: other modes do not affect it"
func TestKpPaste_Options_fromTerminal_other_modes_do_not_affect_it(t *testing.T) {
	term := kpPasteTerminal(t, nil)
	defer term.Deinit()

	term.modes.Set(ModesModeFocusEvent, true)
	term.modes.Set(ModesModeMouseEventNormal, true)
	term.modes.Set(ModesModeCursorKeys, true)
	term.modes.Set(ModesModeAltScreen, true)
	expectEqual(t, false, InputPasteOptionsFromTerminal(term).bracketed)

	term.modes.Set(ModesModeBracketedPaste, true)
	term.modes.Set(ModesModeFocusEvent, false)
	expectEqual(t, true, InputPasteOptionsFromTerminal(term).bracketed)
}

// Zig: test "term51: Options.fromTerminal: default_modes and reset"
func TestKpPaste_Options_fromTerminal_default_modes_and_reset(t *testing.T) {
	// Default on: starts on, a reset restores on, and clearing works until reset.
	{
		term := kpPasteTerminal(t, func(opts *TerminalOptions) {
			opts.default_modes.bracketed_paste = true
		})
		defer term.Deinit()
		expectEqual(t, true, InputPasteOptionsFromTerminal(term).bracketed)
		term.modes.Set(ModesModeBracketedPaste, false)
		expectEqual(t, false, InputPasteOptionsFromTerminal(term).bracketed)
		term.FullReset()
		expectEqual(t, true, InputPasteOptionsFromTerminal(term).bracketed)
	}

	// Default off: set on, a reset turns it back off.
	{
		term := kpPasteTerminal(t, nil)
		defer term.Deinit()
		term.modes.Set(ModesModeBracketedPaste, true)
		expectEqual(t, true, InputPasteOptionsFromTerminal(term).bracketed)
		term.FullReset()
		expectEqual(t, false, InputPasteOptionsFromTerminal(term).bracketed)
	}
}

// Zig: test "term51: Options.fromTerminal: result drives encode"
func TestKpPaste_Options_fromTerminal_result_drives_encode(t *testing.T) {
	term := kpPasteTerminal(t, nil)
	defer term.Deinit()

	off, err := InputPasteEncode([]uint8("hi"), InputPasteOptionsFromTerminal(term))
	zigTry(t, err) // `catch unreachable`
	expectEqualStrings(t, "", string(off[0]))
	expectEqualStrings(t, "hi", string(off[1]))
	expectEqualStrings(t, "", string(off[2]))

	term.modes.Set(ModesModeBracketedPaste, true)
	on, err := InputPasteEncode([]uint8("hi"), InputPasteOptionsFromTerminal(term))
	zigTry(t, err) // `catch unreachable`
	expectEqualStrings(t, "\x1b[200~", string(on[0]))
	expectEqualStrings(t, "hi", string(on[1]))
	expectEqualStrings(t, "\x1b[201~", string(on[2]))
}

// Zig: test "term51: encode: error set and return types"
func TestKpPaste_encode_error_set_and_return_types(t *testing.T) {
	// The error set has the one member MutableRequired: the sentinel.
	expectEqualStrings(t, "MutableRequired", ErrMutableRequired.Error())
	// Zig-only: `@typeInfo(Error).error_set.?.len == 1`; Go has no error-set types.

	// The return types: `[]u8` data gives `[3][]const u8`, `[]const u8`
	// data gives `Error![3][]const u8` (checked at compile time; neither
	// call is evaluated, as with `@TypeOf`).
	var _ func([]uint8, InputPasteOptions) [3][]uint8 = InputPasteEncodeMutable
	var _ func([]uint8, InputPasteOptions) ([3][]uint8, error) = InputPasteEncode
}

// Zig: test "term51: encode: bracketed const clean data is passed through without a copy"
func TestKpPaste_encode_bracketed_const_clean_data_is_passed_through_without_a_copy(t *testing.T) {
	c := []uint8("hello world")
	r, err := InputPasteEncode(c, InputPasteOptions{bracketed: true})
	zigTry(t, err)
	expectEqualStrings(t, "\x1b[200~", string(r[0]))
	expectEqualStrings(t, "hello world", string(r[1]))
	expectEqualStrings(t, "\x1b[201~", string(r[2]))
	expect(t, kpPasteSamePtr(c, r[1]), "@intFromPtr(c.ptr) == @intFromPtr(r[1].ptr)")
	expectEqual(t, len(c), len(r[1]))

	joined := kpPasteJoin(r)
	// Zig-only: defer std.testing.allocator.free(joined)
	expectEqualStrings(t, "\x1b[200~hello world\x1b[201~", string(joined))
}

// Zig: test "term51: encode: unbracketed const clean data has empty fences"
func TestKpPaste_encode_unbracketed_const_clean_data_has_empty_fences(t *testing.T) {
	c := []uint8("hello world")
	r, err := InputPasteEncode(c, InputPasteOptions{bracketed: false})
	zigTry(t, err)
	expectEqual(t, 0, len(r[0]))
	expectEqual(t, 0, len(r[2]))
	expectEqualStrings(t, "hello world", string(r[1]))
	expect(t, kpPasteSamePtr(c, r[1]), "@intFromPtr(c.ptr) == @intFromPtr(r[1].ptr)")

	joined := kpPasteJoin(r)
	// Zig-only: defer std.testing.allocator.free(joined)
	expectEqualStrings(t, "hello world", string(joined))
}

// Zig: test "term51: encode: empty data"
func TestKpPaste_encode_empty_data(t *testing.T) {
	c := []uint8("")
	rb, err := InputPasteEncode(c, InputPasteOptions{bracketed: true})
	zigTry(t, err)
	expectEqualStrings(t, "\x1b[200~", string(rb[0]))
	expectEqual(t, 0, len(rb[1]))
	expectEqualStrings(t, "\x1b[201~", string(rb[2]))

	ru, err := InputPasteEncode(c, InputPasteOptions{bracketed: false})
	zigTry(t, err)
	expectEqual(t, 0, len(ru[0]))
	expectEqual(t, 0, len(ru[1]))
	expectEqual(t, 0, len(ru[2]))

	kpPasteExpectMutable(t, "", true, "\x1b[200~", "", "\x1b[201~")
	kpPasteExpectMutable(t, "", false, "", "", "")
}

// Zig: test "term51: encode: every byte value, const and mutable, bracketed and not"
func TestKpPaste_encode_every_byte_value_const_and_mutable_bracketed_and_not(t *testing.T) {
	for b := 0; b < 256; b++ {
		byte_ := uint8(b)
		strip := kpPasteIsStrip(byte_)
		nl := byte_ == '\n'

		for _, bracketed := range [...]bool{true, false} {
			// Const: error iff a strip byte, or (unbracketed) a newline.
			needs_mut := strip || (!bracketed && nl)
			cbuf := [...]uint8{byte_}
			cdata := cbuf[:]
			if r, err := InputPasteEncode(cdata, InputPasteOptions{bracketed: bracketed}); err == nil {
				expect(t, !needs_mut, "!needs_mut")
				expectEqual(t, byte_, r[1][0])
				expectEqual(t, 1, len(r[1]))
			} else {
				expect(t, needs_mut, "needs_mut")
				expectEqual(t, error(ErrMutableRequired), err)
				// A failed const encode must not modify the data.
				expectEqual(t, byte_, cbuf[0])
			}

			// Mutable.
			mbuf := [...]uint8{byte_}
			mdata := mbuf[:]
			r := InputPasteEncodeMutable(mdata, InputPasteOptions{bracketed: bracketed})
			var want uint8
			if strip {
				want = ' '
			} else if !bracketed && nl {
				want = '\r'
			} else {
				want = byte_
			}
			expectEqual(t, want, mbuf[0])
			expectEqual(t, 1, len(r[1]))
			expectEqual(t, want, r[1][0])
			if bracketed {
				expectEqualStrings(t, "\x1b[200~", string(r[0]))
				expectEqualStrings(t, "\x1b[201~", string(r[2]))
			} else {
				expectEqual(t, 0, len(r[0]))
				expectEqual(t, 0, len(r[2]))
			}
		}
	}
}

// Zig: test "term51: encode: each stripped control byte becomes a space"
func TestKpPaste_encode_each_stripped_control_byte_becomes_a_space(t *testing.T) {
	// Same list as the source, spelled out with names.
	names := [...]struct {
		x0 uint8
		x1 string
	}{
		{0x00, "NUL"},
		{0x08, "BS"},
		{0x05, "ENQ"},
		{0x04, "EOT"},
		{0x1B, "ESC"},
		{0x7F, "DEL"},
		{0x03, "VINTR"},
		{0x1C, "VQUIT"},
		{0x15, "VKILL"},
		{0x1A, "VSUSP"},
		{0x11, "VSTART"},
		{0x13, "VSTOP"},
		{0x17, "VWERASE"},
		{0x16, "VLNEXT"},
		{0x12, "VREPRINT"},
		{0x0F, "VDISCARD"},
	}
	expectEqual(t, len(kpPasteStripList), len(names))
	for i, n := range names {
		expectEqual(t, kpPasteStripList[i], n.x0)
	}

	for _, n := range names {
		input := [...]uint8{'a', n.x0, 'b'}
		kpPasteExpectMutable(t, string(input[:]), true, "\x1b[200~", "a b", "\x1b[201~")
		kpPasteExpectMutable(t, string(input[:]), false, "", "a b", "")

		c := input[:]
		kpPasteExpectMutableRequired(t, c, true)
		kpPasteExpectMutableRequired(t, c, false)
	}
}

// Zig: test "term51: encode: control characters that are not stripped are kept"
func TestKpPaste_encode_control_characters_that_are_not_stripped_are_kept(t *testing.T) {
	// TAB, VT, FF, CR and the other C0 values not in the strip list.
	kept := [...]uint8{
		0x01, 0x02, 0x06, 0x07, 0x09, 0x0B, 0x0C, 0x0D,
		0x0E, 0x10, 0x14, 0x18, 0x19, 0x1D, 0x1E, 0x1F,
	}
	for _, b := range kept {
		expect(t, !kpPasteIsStrip(b), "!isStrip(b)")
		input := [...]uint8{'a', b, 'b'}
		kpPasteExpectMutable(t, string(input[:]), true, "\x1b[200~", string(input[:]), "\x1b[201~")
		kpPasteExpectMutable(t, string(input[:]), false, "", string(input[:]), "")

		c := input[:]
		rb, err := InputPasteEncode(c, InputPasteOptions{bracketed: true})
		zigTry(t, err)
		expectEqualSlices(t, input[:], rb[1])
		ru, err := InputPasteEncode(c, InputPasteOptions{bracketed: false})
		zigTry(t, err)
		expectEqualSlices(t, input[:], ru[1])
	}
}

// Zig: test "term51: encode: boundary bytes around the C0 range, DEL and 8-bit bytes"
func TestKpPaste_encode_boundary_bytes_around_the_C0_range_DEL_and_8_bit_bytes(t *testing.T) {
	// 0x20 (space) and 0x7E (~) are kept; 0x7F is stripped; 0x80, 0x9B
	// (C1 CSI), 0xA0 and 0xFF are kept (no UTF-8 validation).
	input := [...]uint8{0x20, 0x7E, 0x7F, 0x80, 0x9B, 0xA0, 0xFF, 0x1F}
	want := [...]uint8{0x20, 0x7E, ' ', 0x80, 0x9B, 0xA0, 0xFF, 0x1F}
	kpPasteExpectMutable(t, string(input[:]), true, "\x1b[200~", string(want[:]), "\x1b[201~")
	kpPasteExpectMutable(t, string(input[:]), false, "", string(want[:]), "")

	// Valid UTF-8 multibyte text is left alone, const and mutable.
	text := "h\xc3\xa9llo \xe2\x82\xac \xf0\x9f\x98\x80"
	c := []uint8(text)
	r, err := InputPasteEncode(c, InputPasteOptions{bracketed: false})
	zigTry(t, err)
	expectEqualStrings(t, text, string(r[1]))
	kpPasteExpectMutable(t, text, true, "\x1b[200~", text, "\x1b[201~")
	kpPasteExpectMutable(t, text, false, "", text, "")

	// Invalid UTF-8 (lone continuation / truncated lead byte) is left alone.
	bad := "a\x80b\xc3"
	kpPasteExpectMutable(t, bad, false, "", bad, "")
}

// Zig: test "term51: encode: newline handling bracketed"
func TestKpPaste_encode_newline_handling_bracketed(t *testing.T) {
	// Bracketed: newlines are kept as LF, const data does not need a copy.
	c := []uint8("a\nb\r\nc\n")
	r, err := InputPasteEncode(c, InputPasteOptions{bracketed: true})
	zigTry(t, err)
	expectEqualStrings(t, "a\nb\r\nc\n", string(r[1]))
	expect(t, kpPasteSamePtr(c, r[1]), "@intFromPtr(c.ptr) == @intFromPtr(r[1].ptr)")
	kpPasteExpectMutable(t, "a\nb\r\nc\n", true, "\x1b[200~", "a\nb\r\nc\n", "\x1b[201~")
}

// Zig: test "term51: encode: newline handling unbracketed"
func TestKpPaste_encode_newline_handling_unbracketed(t *testing.T) {
	kpPasteExpectMutable(t, "\n", false, "", "\r", "")
	kpPasteExpectMutable(t, "\n\n\n", false, "", "\r\r\r", "")
	kpPasteExpectMutable(t, "\nabc", false, "", "\rabc", "")
	kpPasteExpectMutable(t, "abc\n", false, "", "abc\r", "")
	kpPasteExpectMutable(t, "a\nb\nc", false, "", "a\rb\rc", "")
	kpPasteExpectMutable(t, "a\r\nb", false, "", "a\r\rb", "")
	kpPasteExpectMutable(t, "a\rb", false, "", "a\rb", "")
	kpPasteExpectMutable(t, "\r\n\r\n", false, "", "\r\r\r\r", "")

	c := []uint8("a\nb")
	kpPasteExpectMutableRequired(t, c, false)
	c2 := []uint8("a\r\nb")
	kpPasteExpectMutableRequired(t, c2, false)
	// CR alone does not need conversion, so const data is fine.
	c3 := []uint8("a\rb")
	r, err := InputPasteEncode(c3, InputPasteOptions{bracketed: false})
	zigTry(t, err)
	expectEqualStrings(t, "a\rb", string(r[1]))
}

// Zig: test "term51: encode: strip and newline together"
func TestKpPaste_encode_strip_and_newline_together(t *testing.T) {
	kpPasteExpectMutable(t, "a\x00\nb\x1b\n", false, "", "a \rb \r", "")
	kpPasteExpectMutable(t, "a\x00\nb\x1b\n", true, "\x1b[200~", "a \nb \n", "\x1b[201~")

	// Const: either condition alone is enough for an error when unbracketed;
	// bracketed only the strip bytes are.
	strip_only := []uint8("a\x00b")
	kpPasteExpectMutableRequired(t, strip_only, true)
	kpPasteExpectMutableRequired(t, strip_only, false)
	nl_only := []uint8("a\nb")
	_, err := InputPasteEncode(nl_only, InputPasteOptions{bracketed: true})
	zigTry(t, err)
	kpPasteExpectMutableRequired(t, nl_only, false)
	both := []uint8("a\x00\nb")
	kpPasteExpectMutableRequired(t, both, true)
	kpPasteExpectMutableRequired(t, both, false)
}

// Zig: test "term51: encode: const data is never modified when an error is returned"
func TestKpPaste_encode_const_data_is_never_modified_when_an_error_is_returned(t *testing.T) {
	buf := []uint8("a\x00b\nc\x1bd")
	c := buf
	kpPasteExpectMutableRequired(t, c, false)
	expectEqualStrings(t, "a\x00b\nc\x1bd", string(buf))
	kpPasteExpectMutableRequired(t, c, true)
	expectEqualStrings(t, "a\x00b\nc\x1bd", string(buf))
}

// Zig: test "term51: encode: consecutive and edge-positioned strip bytes"
func TestKpPaste_encode_consecutive_and_edge_positioned_strip_bytes(t *testing.T) {
	kpPasteExpectMutable(t, "\x00", true, "\x1b[200~", " ", "\x1b[201~")
	kpPasteExpectMutable(t, "\x00\x00", true, "\x1b[200~", "  ", "\x1b[201~")
	kpPasteExpectMutable(t, "\x00abc", false, "", " abc", "")
	kpPasteExpectMutable(t, "abc\x7f", false, "", "abc ", "")
	kpPasteExpectMutable(t, "\x00a\x08b\x05c\x04d\x1be\x7ff", true, "\x1b[200~", " a b c d e f", "\x1b[201~")
	kpPasteExpectMutable(t, "\x03\x1c\x15\x1a\x11\x13\x17\x16\x12\x0f", false, "", "          ", "")
	// Spaces already present stay.
	kpPasteExpectMutable(t, " \x00 ", false, "", "   ", "")
}

// Zig: test "term51: encode: embedded bracketed paste end sequence is defanged by ESC stripping"
func TestKpPaste_encode_embedded_bracketed_paste_end_sequence_is_defanged_by_ESC_stripping(t *testing.T) {
	// Bracketed: ESC in the data becomes a space, so the paste can't end early.
	kpPasteExpectMutable(t, "a\x1b[201~b", true, "\x1b[200~", "a [201~b", "\x1b[201~")
	// Unbracketed: same.
	kpPasteExpectMutable(t, "a\x1b[201~b", false, "", "a [201~b", "")
	// Embedded start sequence.
	kpPasteExpectMutable(t, "\x1b[200~x\x1b[201~", true, "\x1b[200~", " [200~x [201~", "\x1b[201~")
	// After encode the middle slice is safe by isSafe's standard.
	{
		data := []uint8("a\x1b[201~b")
		// Zig-only: defer std.testing.allocator.free(data)
		expect(t, !InputPasteIsSafe(data), "!isSafe(data)")
		r := InputPasteEncodeMutable(data, InputPasteOptions{bracketed: true})
		expect(t, InputPasteIsSafe(r[1]), "isSafe(r[1])")
	}

	// Const data with the sequence is an error in either mode.
	c := []uint8("a\x1b[201~b")
	kpPasteExpectMutableRequired(t, c, true)
	kpPasteExpectMutableRequired(t, c, false)

	// The text without ESC is not a problem for const data.
	c2 := []uint8("a[201~b")
	r2, err := InputPasteEncode(c2, InputPasteOptions{bracketed: true})
	zigTry(t, err)
	expectEqualStrings(t, "a[201~b", string(r2[1]))
}

// Zig: test "term51: encode: large data"
func TestKpPaste_encode_large_data(t *testing.T) {
	const n = 1 << 20

	// Clean const data: no copy.
	{
		buf := make([]uint8, n)
		// Zig-only: defer std.testing.allocator.free(buf)
		for i := range buf {
			buf[i] = 'x'
		}
		c := buf
		r, err := InputPasteEncode(c, InputPasteOptions{bracketed: false})
		zigTry(t, err)
		expectEqual(t, n, len(r[1]))
		expect(t, kpPasteSamePtr(buf, r[1]), "@intFromPtr(buf.ptr) == @intFromPtr(r[1].ptr)")
		// Last byte newline: error when unbracketed, fine when bracketed.
		buf[n-1] = '\n'
		kpPasteExpectMutableRequired(t, c, false)
		rb, err := InputPasteEncode(c, InputPasteOptions{bracketed: true})
		zigTry(t, err)
		expectEqual(t, n, len(rb[1]))
		// Last byte ESC: error when bracketed too.
		buf[n-1] = 0x1b
		kpPasteExpectMutableRequired(t, c, true)
	}

	// Mutable data with strip bytes and newlines throughout.
	{
		buf := make([]uint8, n)
		// Zig-only: defer std.testing.allocator.free(buf)
		for i := range buf {
			switch i % 4 {
			case 0:
				buf[i] = 'a'
			case 1:
				buf[i] = 0x00
			case 2:
				buf[i] = '\n'
			default:
				buf[i] = 0x80
			}
		}
		r := InputPasteEncodeMutable(buf, InputPasteOptions{bracketed: false})
		expectEqual(t, n, len(r[1]))
		for i, b := range r[1] {
			var want uint8
			switch i % 4 {
			case 0:
				want = 'a'
			case 1:
				want = ' '
			case 2:
				want = '\r'
			default:
				want = 0x80
			}
			expectEqual(t, want, b)
		}
	}

	// Large data that is all strip bytes (every position replaced).
	{
		buf := make([]uint8, n)
		// Zig-only: defer std.testing.allocator.free(buf)
		for i := range buf {
			buf[i] = 0x1b
		}
		r := InputPasteEncodeMutable(buf, InputPasteOptions{bracketed: true})
		expectEqual(t, n, len(r[1]))
		for _, b := range r[1] {
			expectEqual(t, uint8(' '), b)
		}
	}
}

// Zig: test "term51: encode: three slices concatenate to the pty bytes"
func TestKpPaste_encode_three_slices_concatenate_to_the_pty_bytes(t *testing.T) {
	data := []uint8("ls\x03 -l\nexit\n")
	// Zig-only: defer std.testing.allocator.free(data)
	r := InputPasteEncodeMutable(data, InputPasteOptions{bracketed: true})
	joined := kpPasteJoin(r)
	// Zig-only: defer std.testing.allocator.free(joined)
	expectEqualStrings(t, "\x1b[200~ls  -l\nexit\n\x1b[201~", string(joined))

	data2 := []uint8("ls\x03 -l\nexit\n")
	// Zig-only: defer std.testing.allocator.free(data2)
	r2 := InputPasteEncodeMutable(data2, InputPasteOptions{bracketed: false})
	joined2 := kpPasteJoin(r2)
	// Zig-only: defer std.testing.allocator.free(joined2)
	expectEqualStrings(t, "ls  -l\rexit\r", string(joined2))
}

// Zig: test "term51: isSafe: empty and plain data"
func TestKpPaste_isSafe_empty_and_plain_data(t *testing.T) {
	expect(t, InputPasteIsSafe([]uint8("")), `isSafe("")`)
	expect(t, InputPasteIsSafe([]uint8("a")), `isSafe("a")`)
	expect(t, InputPasteIsSafe([]uint8("hello world")), `isSafe("hello world")`)
	expect(t, InputPasteIsSafe([]uint8("tab\there")), `isSafe("tab\there")`)
	expect(t, InputPasteIsSafe([]uint8("carriage\rreturn")), `isSafe("carriage\rreturn")`)
}

// Zig: test "term51: isSafe: every single byte value"
func TestKpPaste_isSafe_every_single_byte_value(t *testing.T) {
	for b := 0; b < 256; b++ {
		byte_ := uint8(b)
		buf := [...]uint8{byte_}
		// Only newline (0x0A) is unsafe by itself.
		expectEqual(t, byte_ != '\n', InputPasteIsSafe(buf[:]))
	}
}

// Zig: test "term51: isSafe: boundary bytes"
func TestKpPaste_isSafe_boundary_bytes(t *testing.T) {
	expect(t, InputPasteIsSafe([]uint8("\x00")), `isSafe("\x00")`)
	expect(t, InputPasteIsSafe([]uint8("\x1f")), `isSafe("\x1f")`)
	expect(t, InputPasteIsSafe([]uint8("\x20")), `isSafe("\x20")`)
	expect(t, InputPasteIsSafe([]uint8("\x7e")), `isSafe("\x7e")`)
	expect(t, InputPasteIsSafe([]uint8("\x7f")), `isSafe("\x7f")`)
	expect(t, InputPasteIsSafe([]uint8("\x80")), `isSafe("\x80")`)
	expect(t, InputPasteIsSafe([]uint8("\xff")), `isSafe("\xff")`)
	expect(t, InputPasteIsSafe([]uint8("\x09")), `isSafe("\x09")`)
	expect(t, !InputPasteIsSafe([]uint8("\x0a")), `!isSafe("\x0a")`)
	expect(t, InputPasteIsSafe([]uint8("\x0b")), `isSafe("\x0b")`)
	expect(t, InputPasteIsSafe([]uint8("\x0d")), `isSafe("\x0d")`)
	// A NUL byte does not terminate the scan.
	expect(t, !InputPasteIsSafe([]uint8("a\x00\nb")), `!isSafe("a\x00\nb")`)
	expect(t, !InputPasteIsSafe([]uint8("a\x00\x1b[201~")), `!isSafe("a\x00\x1b[201~")`)
}

// Zig: test "term51: isSafe: control and C1 bytes other than newline are safe"
func TestKpPaste_isSafe_control_and_C1_bytes_other_than_newline_are_safe(t *testing.T) {
	for b := 0; b < 0x20; b++ {
		if b == '\n' {
			continue
		}
		buf := [...]uint8{'a', uint8(b), 'b'}
		expect(t, InputPasteIsSafe(buf[:]), "isSafe(&buf)")
	}
	for b := 0x7f; b < 0xa0; b++ {
		buf := [...]uint8{'a', uint8(b), 'b'}
		expect(t, InputPasteIsSafe(buf[:]), "isSafe(&buf)")
	}
	// C1 CSI (0x9b) followed by 201~ is not recognized.
	expect(t, InputPasteIsSafe([]uint8("\x9b201~")), `isSafe("\x9b201~")`)
	// ESC alone, ESC [ and the bracketed paste start are all safe.
	expect(t, InputPasteIsSafe([]uint8("\x1b")), `isSafe("\x1b")`)
	expect(t, InputPasteIsSafe([]uint8("\x1b[")), `isSafe("\x1b[")`)
	expect(t, InputPasteIsSafe([]uint8("\x1b[200~")), `isSafe("\x1b[200~")`)
	expect(t, InputPasteIsSafe([]uint8("\x1b]201~")), `isSafe("\x1b]201~")`)
}

// Zig: test "term51: isSafe: newline anywhere is unsafe"
func TestKpPaste_isSafe_newline_anywhere_is_unsafe(t *testing.T) {
	expect(t, !InputPasteIsSafe([]uint8("\n")), `!isSafe("\n")`)
	expect(t, !InputPasteIsSafe([]uint8("\nabc")), `!isSafe("\nabc")`)
	expect(t, !InputPasteIsSafe([]uint8("abc\n")), `!isSafe("abc\n")`)
	expect(t, !InputPasteIsSafe([]uint8("a\nb")), `!isSafe("a\nb")`)
	expect(t, !InputPasteIsSafe([]uint8("a\r\nb")), `!isSafe("a\r\nb")`)
	expect(t, !InputPasteIsSafe([]uint8("\n\n")), `!isSafe("\n\n")`)
	expect(t, !InputPasteIsSafe([]uint8("\x1b[200~a\n\x1b[201")), `!isSafe("\x1b[200~a\n\x1b[201")`)
}

// Zig: test "term51: isSafe: bracketed paste end sequence"
func TestKpPaste_isSafe_bracketed_paste_end_sequence(t *testing.T) {
	expect(t, !InputPasteIsSafe([]uint8("\x1b[201~")), `!isSafe("\x1b[201~")`)
	expect(t, !InputPasteIsSafe([]uint8("\x1b[201~abc")), `!isSafe("\x1b[201~abc")`)
	expect(t, !InputPasteIsSafe([]uint8("abc\x1b[201~")), `!isSafe("abc\x1b[201~")`)
	expect(t, !InputPasteIsSafe([]uint8("ab\x1b[201~cd")), `!isSafe("ab\x1b[201~cd")`)
	expect(t, !InputPasteIsSafe([]uint8("\x1b[201~\x1b[201~")), `!isSafe("\x1b[201~\x1b[201~")`)
	expect(t, !InputPasteIsSafe([]uint8("\x1b\x1b[201~")), `!isSafe("\x1b\x1b[201~")`)
	expect(t, !InputPasteIsSafe([]uint8("\x1b[200~\x1b[201~")), `!isSafe("\x1b[200~\x1b[201~")`)
	expect(t, !InputPasteIsSafe([]uint8("\x1b[2\x1b[201~")), `!isSafe("\x1b[2\x1b[201~")`)

	// Near misses are safe.
	expect(t, InputPasteIsSafe([]uint8("\x1b[201")), `isSafe("\x1b[201")`)
	expect(t, InputPasteIsSafe([]uint8("\x1b[20~")), `isSafe("\x1b[20~")`)
	expect(t, InputPasteIsSafe([]uint8("\x1b[2011~")), `isSafe("\x1b[2011~")`)
	expect(t, InputPasteIsSafe([]uint8("\x1b[ 201~")), `isSafe("\x1b[ 201~")`)
	expect(t, InputPasteIsSafe([]uint8("[201~")), `isSafe("[201~")`)
	expect(t, InputPasteIsSafe([]uint8("\x1b[202~")), `isSafe("\x1b[202~")`)
	expect(t, InputPasteIsSafe([]uint8("\x1b[200~")), `isSafe("\x1b[200~")`)
	expect(t, InputPasteIsSafe([]uint8("\x1b[201")), `isSafe("\x1b[201")`)
	expect(t, InputPasteIsSafe([]uint8("\x1b[20")), `isSafe("\x1b[20")`)
	expect(t, InputPasteIsSafe([]uint8("1~")), `isSafe("1~")`)
	expect(t, InputPasteIsSafe([]uint8("\x1b[201 ~")), `isSafe("\x1b[201 ~")`)
	// Split by another byte.
	expect(t, InputPasteIsSafe([]uint8("\x1b[20\x001~")), `isSafe("\x1b[20\x001~")`)
}

// Zig: test "term51: isSafe: long data"
func TestKpPaste_isSafe_long_data(t *testing.T) {
	const n = 1 << 20
	buf := make([]uint8, n)
	// Zig-only: defer std.testing.allocator.free(buf)
	fill := func(b uint8) {
		for i := range buf {
			buf[i] = b
		}
	}
	fill('a')
	expect(t, InputPasteIsSafe(buf), "isSafe(buf)")

	// Newline as the last byte.
	buf[n-1] = '\n'
	expect(t, !InputPasteIsSafe(buf), "!isSafe(buf)")
	buf[n-1] = 'a'

	// End sequence flush against the end of the data.
	copy(buf[n-6:], "\x1b[201~")
	expect(t, !InputPasteIsSafe(buf), "!isSafe(buf)")

	// End sequence one byte short at the end is safe.
	copy(buf[n-6:], "\x1b[201a")
	expect(t, InputPasteIsSafe(buf), "isSafe(buf)")

	// End sequence in the middle.
	fill('a')
	copy(buf[n/2:][:6], "\x1b[201~")
	expect(t, !InputPasteIsSafe(buf), "!isSafe(buf)")

	// At the very start.
	fill('a')
	copy(buf[0:6], "\x1b[201~")
	expect(t, !InputPasteIsSafe(buf), "!isSafe(buf)")

	// High bytes only.
	fill(0xff)
	expect(t, InputPasteIsSafe(buf), "isSafe(buf)")
}
