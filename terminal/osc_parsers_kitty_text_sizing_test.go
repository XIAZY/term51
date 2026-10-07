// Tests translated one for one from src/terminal/osc/parsers/kitty_text_sizing.zig
// (Ghostty v1.3.1 plus the term51 additional tests). Each Go test carries the
// exact Zig title in a "// Zig: test" comment. The helpers expect,
// expectEqual, expectEqualSlices, expectEqualStrings, zigTry and zigUnwrap are
// defined in Parser_test.go and dcs_test.go.
//
// Translation notes:
//   - `.init(testing.allocator)` is OSCParserInit(true), any non-nil
//     value; `.init(null)` is OSCParserInit(false). The leak check of
//     testing.allocator has no Go counterpart; the `defer p.deinit()` calls
//     are kept.
//   - `p.next(ch)` is p.Next(ch); `p.end('\x1b')` is p.End(Some[uint8](0x1b)).
//   - `parse(&p, null).?.*` is oscB_kts_unwrap(t, OSCParsersKittyTextSizingParse(...)),
//     which fails on nil and copies the command out.
//   - `cmd.kitty_text_sizing` is cmd.KittyTextSizing.
//   - `const osc: OSC = .{ .text = ... }` is oscB_kts_newOSC(...): the Go zero
//     value of scale is 0, Zig's default is 1, so every place that Zig builds
//     an OSC by default sets scale to 1 explicitly.
//   - `try osc.update(k, v)` is zigTry(t, osc.update(k, v)); `try
//     testing.expectError(error.X, osc.update(k, v))` compares the returned
//     error with ErrX.
//   - `value.ptr[n]` (the sentinel byte after the slice) is oscB_kts_at.
//   - `inline for` over cases is a loop over a slice; `"x" ** n` is
//     strings.Repeat.

package terminal

import (
	"strings"
	"testing"
)

// oscB_kts_alloc stands for `testing.allocator`: any non-nil value for
// OSCParserInit.
var oscB_kts_alloc any = "testing.allocator"

// oscB_kts_unwrap is Zig's `ptr.?.*` on the result of parse or end.
func oscB_kts_unwrap(t *testing.T, c *OSCCommand) OSCCommand {
	t.Helper()
	if c == nil {
		t.Fatalf("unwrapped a null optional")
	}
	return *c
}

// oscB_kts_term51Feed is Zig's term51Feed.
func oscB_kts_term51Feed(p *OSCParser, input string) {
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
}

// oscB_kts_at is Zig's `s.ptr[i]`: the byte at index i of the memory behind
// s, which may lie past its length (the sentinel).
func oscB_kts_at(s []uint8, i int) uint8 {
	return s[:i+1][i]
}

// oscB_kts_newOSC is Zig's `.{ .text = text }`: every field has its Zig
// default, which for scale is 1.
func oscB_kts_newOSC(text string) OSCParsersKittyTextSizingOSC {
	return OSCParsersKittyTextSizingOSC{Scale: 1, Text: []uint8(text)}
}

// oscB_kts_term51ExpectFields is Zig's term51ExpectFields: check every field
// except text.
func oscB_kts_term51ExpectFields(
	t *testing.T,
	osc OSCParsersKittyTextSizingOSC,
	scale uint8,
	width uint8,
	numerator uint8,
	denominator uint8,
	valign OSCParsersKittyTextSizingVAlign,
	halign OSCParsersKittyTextSizingHAlign,
) {
	t.Helper()
	expectEqual(t, scale, osc.Scale)
	expectEqual(t, width, osc.Width)
	expectEqual(t, numerator, osc.Numerator)
	expectEqual(t, denominator, osc.Denominator)
	expectEqual(t, valign, osc.Valign)
	expectEqual(t, halign, osc.Halign)
}

// Zig: test "OSC 66: empty parameters"
func TestOSCParsersKittyTextSizing_OSC_66_empty_parameters(t *testing.T) {
	p := OSCParserInit(false)

	input := "66;;bobr"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}

	cmd := oscB_kts_unwrap(t, p.End(Some[uint8](0x1b)))
	expect(t, cmd.Tag == OSCCommandKeyKittyTextSizing, "cmd == .kitty_text_sizing")
	expectEqual(t, uint8(1), cmd.KittyTextSizing.Scale)
	expectEqualStrings(t, "bobr", string(cmd.KittyTextSizing.Text))
}

// Zig: test "OSC 66: single parameter"
func TestOSCParsersKittyTextSizing_OSC_66_single_parameter(t *testing.T) {
	p := OSCParserInit(false)

	input := "66;s=2;kurwa"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}

	cmd := oscB_kts_unwrap(t, p.End(Some[uint8](0x1b)))
	expect(t, cmd.Tag == OSCCommandKeyKittyTextSizing, "cmd == .kitty_text_sizing")
	expectEqual(t, uint8(2), cmd.KittyTextSizing.Scale)
	expectEqualStrings(t, "kurwa", string(cmd.KittyTextSizing.Text))
}

// Zig: test "OSC 66: multiple parameters"
func TestOSCParsersKittyTextSizing_OSC_66_multiple_parameters(t *testing.T) {
	p := OSCParserInit(false)

	input := "66;s=2:w=7:n=13:d=15:v=1:h=2;long"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}

	cmd := oscB_kts_unwrap(t, p.End(Some[uint8](0x1b)))
	expect(t, cmd.Tag == OSCCommandKeyKittyTextSizing, "cmd == .kitty_text_sizing")
	expectEqual(t, uint8(2), cmd.KittyTextSizing.Scale)
	expectEqual(t, uint8(7), cmd.KittyTextSizing.Width)
	expectEqual(t, uint8(13), cmd.KittyTextSizing.Numerator)
	expectEqual(t, uint8(15), cmd.KittyTextSizing.Denominator)
	expectEqual(t, OSCParsersKittyTextSizingVAlignBottom, cmd.KittyTextSizing.Valign)
	expectEqual(t, OSCParsersKittyTextSizingHAlignCenter, cmd.KittyTextSizing.Halign)
	expectEqualStrings(t, "long", string(cmd.KittyTextSizing.Text))
}

// Zig: test "OSC 66: scale is zero"
func TestOSCParsersKittyTextSizing_OSC_66_scale_is_zero(t *testing.T) {
	p := OSCParserInit(false)

	input := "66;s=0;nope"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	cmd := oscB_kts_unwrap(t, p.End(Some[uint8](0x1b)))

	expect(t, cmd.Tag == OSCCommandKeyKittyTextSizing, "cmd == .kitty_text_sizing")
	expectEqual(t, uint8(1), cmd.KittyTextSizing.Scale)
}

// Zig: test "OSC 66: invalid parameters"
func TestOSCParsersKittyTextSizing_OSC_66_invalid_parameters(t *testing.T) {
	p := OSCParserInit(false)

	for _, ch := range []uint8("66;w=8:v=3:n=16;") {
		p.Next(ch)
	}
	cmd := oscB_kts_unwrap(t, p.End(Some[uint8](0x1b)))

	expect(t, cmd.Tag == OSCCommandKeyKittyTextSizing, "cmd == .kitty_text_sizing")
	expectEqual(t, uint8(0), cmd.KittyTextSizing.Width)
	expect(t, cmd.KittyTextSizing.Valign == OSCParsersKittyTextSizingVAlignTop, "cmd.kitty_text_sizing.valign == .top")
	expectEqual(t, uint8(0), cmd.KittyTextSizing.Numerator)
}

// Zig: test "OSC 66: UTF-8"
func TestOSCParsersKittyTextSizing_OSC_66_UTF_8(t *testing.T) {
	p := OSCParserInit(false)

	input := "66;;👻魑魅魍魉ゴースッティ"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}

	cmd := oscB_kts_unwrap(t, p.End(Some[uint8](0x1b)))
	expect(t, cmd.Tag == OSCCommandKeyKittyTextSizing, "cmd == .kitty_text_sizing")
	expectEqualStrings(t, "👻魑魅魍魉ゴースッティ", string(cmd.KittyTextSizing.Text))
}

// Zig: test "OSC 66: unsafe UTF-8"
func TestOSCParsersKittyTextSizing_OSC_66_unsafe_UTF_8(t *testing.T) {
	p := OSCParserInit(false)

	input := "66;;\n"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}

	expect(t, p.End(Some[uint8](0x1b)) == nil, "p.end('\\x1b') == null")
}

// Zig: test "OSC 66: overlong UTF-8"
func TestOSCParsersKittyTextSizing_OSC_66_overlong_UTF_8(t *testing.T) {
	p := OSCParserInit(false)

	input := "66;;" + strings.Repeat("bobr", 1025)
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}

	expect(t, p.End(Some[uint8](0x1b)) == nil, "p.end('\\x1b') == null")
}

// Zig: test "term51: OSC.update: defaults before any update"
func TestOSCParsersKittyTextSizing_term51_OSC_update_defaults_before_any_update(t *testing.T) {
	osc := oscB_kts_newOSC("")
	oscB_kts_term51ExpectFields(t, osc, 1, 0, 0, 0, OSCParsersKittyTextSizingVAlignTop, OSCParsersKittyTextSizingHAlignLeft)
}

// Zig: test "term51: OSC.update: scale range is 1 to 7"
func TestOSCParsersKittyTextSizing_term51_OSC_update_scale_range_is_1_to_7(t *testing.T) {
	osc := oscB_kts_newOSC("")
	expectEqual(t, ErrInvalidValue, osc.update('s', []uint8("0")))
	expectEqual(t, uint8(1), osc.Scale)
	zigTry(t, osc.update('s', []uint8("1")))
	expectEqual(t, uint8(1), osc.Scale)
	zigTry(t, osc.update('s', []uint8("7")))
	expectEqual(t, uint8(7), osc.Scale)
	// One past the limit leaves the previous value in place.
	expectEqual(t, ErrInvalidValue, osc.update('s', []uint8("8")))
	expectEqual(t, ErrInvalidValue, osc.update('s', []uint8("15")))
	expectEqual(t, ErrInvalidValue, osc.update('s', []uint8("16")))
	expectEqual(t, uint8(7), osc.Scale)
	oscB_kts_term51ExpectFields(t, osc, 7, 0, 0, 0, OSCParsersKittyTextSizingVAlignTop, OSCParsersKittyTextSizingHAlignLeft)
}

// Zig: test "term51: OSC.update: width range is 0 to 7"
func TestOSCParsersKittyTextSizing_term51_OSC_update_width_range_is_0_to_7(t *testing.T) {
	osc := oscB_kts_newOSC("")
	zigTry(t, osc.update('w', []uint8("7")))
	expectEqual(t, uint8(7), osc.Width)
	expectEqual(t, ErrInvalidValue, osc.update('w', []uint8("8")))
	expectEqual(t, ErrInvalidValue, osc.update('w', []uint8("16")))
	expectEqual(t, uint8(7), osc.Width)
	zigTry(t, osc.update('w', []uint8("0")))
	expectEqual(t, uint8(0), osc.Width)
	oscB_kts_term51ExpectFields(t, osc, 1, 0, 0, 0, OSCParsersKittyTextSizingVAlignTop, OSCParsersKittyTextSizingHAlignLeft)
}

// Zig: test "term51: OSC.update: numerator and denominator range is 0 to 15"
func TestOSCParsersKittyTextSizing_term51_OSC_update_numerator_and_denominator_range_is_0_to_15(t *testing.T) {
	osc := oscB_kts_newOSC("")
	zigTry(t, osc.update('n', []uint8("15")))
	zigTry(t, osc.update('d', []uint8("15")))
	oscB_kts_term51ExpectFields(t, osc, 1, 0, 15, 15, OSCParsersKittyTextSizingVAlignTop, OSCParsersKittyTextSizingHAlignLeft)
	expectEqual(t, ErrInvalidValue, osc.update('n', []uint8("16")))
	expectEqual(t, ErrInvalidValue, osc.update('d', []uint8("16")))
	oscB_kts_term51ExpectFields(t, osc, 1, 0, 15, 15, OSCParsersKittyTextSizingVAlignTop, OSCParsersKittyTextSizingHAlignLeft)
	zigTry(t, osc.update('n', []uint8("0")))
	zigTry(t, osc.update('d', []uint8("0")))
	oscB_kts_term51ExpectFields(t, osc, 1, 0, 0, 0, OSCParsersKittyTextSizingVAlignTop, OSCParsersKittyTextSizingHAlignLeft)
	// The numerator may be larger than the denominator; nothing is checked.
	zigTry(t, osc.update('n', []uint8("9")))
	zigTry(t, osc.update('d', []uint8("2")))
	oscB_kts_term51ExpectFields(t, osc, 1, 0, 9, 2, OSCParsersKittyTextSizingVAlignTop, OSCParsersKittyTextSizingHAlignLeft)
}

// Zig: test "term51: OSC.update: vertical alignment range is 0 to 2"
func TestOSCParsersKittyTextSizing_term51_OSC_update_vertical_alignment_range_is_0_to_2(t *testing.T) {
	osc := oscB_kts_newOSC("")
	zigTry(t, osc.update('v', []uint8("1")))
	expectEqual(t, OSCParsersKittyTextSizingVAlignBottom, osc.Valign)
	zigTry(t, osc.update('v', []uint8("2")))
	expectEqual(t, OSCParsersKittyTextSizingVAlignCenter, osc.Valign)
	expectEqual(t, ErrInvalidValue, osc.update('v', []uint8("3")))
	expectEqual(t, ErrInvalidValue, osc.update('v', []uint8("15")))
	expectEqual(t, OSCParsersKittyTextSizingVAlignCenter, osc.Valign)
	zigTry(t, osc.update('v', []uint8("0")))
	expectEqual(t, OSCParsersKittyTextSizingVAlignTop, osc.Valign)
	oscB_kts_term51ExpectFields(t, osc, 1, 0, 0, 0, OSCParsersKittyTextSizingVAlignTop, OSCParsersKittyTextSizingHAlignLeft)
}

// Zig: test "term51: OSC.update: horizontal alignment range is 0 to 2"
func TestOSCParsersKittyTextSizing_term51_OSC_update_horizontal_alignment_range_is_0_to_2(t *testing.T) {
	osc := oscB_kts_newOSC("")
	zigTry(t, osc.update('h', []uint8("1")))
	expectEqual(t, OSCParsersKittyTextSizingHAlignRight, osc.Halign)
	zigTry(t, osc.update('h', []uint8("2")))
	expectEqual(t, OSCParsersKittyTextSizingHAlignCenter, osc.Halign)
	expectEqual(t, ErrInvalidValue, osc.update('h', []uint8("3")))
	expectEqual(t, ErrInvalidValue, osc.update('h', []uint8("15")))
	expectEqual(t, OSCParsersKittyTextSizingHAlignCenter, osc.Halign)
	zigTry(t, osc.update('h', []uint8("0")))
	expectEqual(t, OSCParsersKittyTextSizingHAlignLeft, osc.Halign)
	oscB_kts_term51ExpectFields(t, osc, 1, 0, 0, 0, OSCParsersKittyTextSizingVAlignTop, OSCParsersKittyTextSizingHAlignLeft)
}

// Zig: test "term51: OSC.update: unknown keys"
func TestOSCParsersKittyTextSizing_term51_OSC_update_unknown_keys(t *testing.T) {
	osc := oscB_kts_newOSC("")
	expectEqual(t, ErrUnknownKey, osc.update('x', []uint8("1")))
	// Keys are case sensitive.
	expectEqual(t, ErrUnknownKey, osc.update('S', []uint8("2")))
	expectEqual(t, ErrUnknownKey, osc.update('W', []uint8("2")))
	expectEqual(t, ErrUnknownKey, osc.update(0, []uint8("2")))
	expectEqual(t, ErrUnknownKey, osc.update('=', []uint8("2")))
	// The value is parsed before the key is looked at, so an unknown key
	// with a bad value reports the value.
	expectEqual(t, ErrInvalidValue, osc.update('x', []uint8("abc")))
	expectEqual(t, ErrInvalidValue, osc.update('x', []uint8("16")))
	expectEqual(t, ErrInvalidValue, osc.update('x', []uint8("")))
	oscB_kts_term51ExpectFields(t, osc, 1, 0, 0, 0, OSCParsersKittyTextSizingVAlignTop, OSCParsersKittyTextSizingHAlignLeft)
}

// Zig: test "term51: OSC.update: invalid values"
func TestOSCParsersKittyTextSizing_term51_OSC_update_invalid_values(t *testing.T) {
	osc := oscB_kts_newOSC("")
	for _, value := range []string{"", " ", "a", "-1", "1.0", " 1", "1 ", "0x1", "16", "99", "_1", "1_", "+", "-"} {
		for _, key := range []uint8{'s', 'w', 'n', 'd', 'v', 'h'} {
			expectEqual(t, ErrInvalidValue, osc.update(key, []uint8(value)))
		}
	}
	oscB_kts_term51ExpectFields(t, osc, 1, 0, 0, 0, OSCParsersKittyTextSizingVAlignTop, OSCParsersKittyTextSizingHAlignLeft)
}

// Zig: test "term51: OSC.update: number formats accepted by std.fmt.parseInt"
func TestOSCParsersKittyTextSizing_term51_OSC_update_number_formats_accepted_by_std_fmt_parseInt(t *testing.T) {
	osc := oscB_kts_newOSC("")
	// A leading '+', leading zeros, '_' between digits and "-0" are all
	// accepted.
	zigTry(t, osc.update('s', []uint8("+2")))
	zigTry(t, osc.update('w', []uint8("007")))
	zigTry(t, osc.update('n', []uint8("1_0")))
	zigTry(t, osc.update('d', []uint8("0_9")))
	oscB_kts_term51ExpectFields(t, osc, 2, 7, 10, 9, OSCParsersKittyTextSizingVAlignTop, OSCParsersKittyTextSizingHAlignLeft)
	zigTry(t, osc.update('w', []uint8("-0")))
	zigTry(t, osc.update('n', []uint8("-0")))
	oscB_kts_term51ExpectFields(t, osc, 2, 0, 0, 9, OSCParsersKittyTextSizingVAlignTop, OSCParsersKittyTextSizingHAlignLeft)
}

// Zig: test "term51: parse: missing separators are invalid"
func TestOSCParsersKittyTextSizing_term51_parse_missing_separators_are_invalid(t *testing.T) {
	// No ';' after 66, or no ';' between the metadata and the payload.
	for _, input := range []string{"66", "66;", "66;s=2", "66;text"} {
		p := OSCParserInit(false)
		oscB_kts_term51Feed(&p, input)
		expect(t, p.state == OSCParserState66, "p.state == .@\"66\"")
		expect(t, OSCParsersKittyTextSizingParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
}

// Zig: test "term51: parse: empty metadata and empty text"
func TestOSCParsersKittyTextSizing_term51_parse_empty_metadata_and_empty_text(t *testing.T) {
	for _, term := range []Optional[uint8]{None[uint8](), Some[uint8](0x07), Some[uint8](0x1b)} {
		p := OSCParserInit(false)
		oscB_kts_term51Feed(&p, "66;;")
		cmd := oscB_kts_unwrap(t, OSCParsersKittyTextSizingParse(&p, term))
		expect(t, cmd.Tag == OSCCommandKeyKittyTextSizing, "cmd == .kitty_text_sizing")
		oscB_kts_term51ExpectFields(t, cmd.KittyTextSizing, 1, 0, 0, 0, OSCParsersKittyTextSizingVAlignTop, OSCParsersKittyTextSizingHAlignLeft)
		expectEqual(t, 0, len(cmd.KittyTextSizing.Text))
		expectEqual(t, uint8(0), oscB_kts_at(cmd.KittyTextSizing.Text, 0))
	}
}

// Zig: test "term51: parse: every parameter"
func TestOSCParsersKittyTextSizing_term51_parse_every_parameter(t *testing.T) {
	{
		p := OSCParserInit(false)
		oscB_kts_term51Feed(&p, "66;s=7:w=7:n=15:d=15:v=2:h=2;x")
		cmd := oscB_kts_unwrap(t, OSCParsersKittyTextSizingParse(&p, None[uint8]()))
		oscB_kts_term51ExpectFields(t, cmd.KittyTextSizing, 7, 7, 15, 15, OSCParsersKittyTextSizingVAlignCenter, OSCParsersKittyTextSizingHAlignCenter)
		expectEqualStrings(t, "x", string(cmd.KittyTextSizing.Text))
	}
	{
		// The order of parameters does not matter.
		p := OSCParserInit(false)
		oscB_kts_term51Feed(&p, "66;h=1:v=1:d=3:n=2:w=4:s=5;x")
		cmd := oscB_kts_unwrap(t, OSCParsersKittyTextSizingParse(&p, None[uint8]()))
		oscB_kts_term51ExpectFields(t, cmd.KittyTextSizing, 5, 4, 2, 3, OSCParsersKittyTextSizingVAlignBottom, OSCParsersKittyTextSizingHAlignRight)
	}
}

// Zig: test "term51: parse: out of range parameters keep the defaults"
func TestOSCParsersKittyTextSizing_term51_parse_out_of_range_parameters_keep_the_defaults(t *testing.T) {
	p := OSCParserInit(false)
	oscB_kts_term51Feed(&p, "66;s=8:w=8:n=16:d=16:v=3:h=3;x")
	cmd := oscB_kts_unwrap(t, OSCParsersKittyTextSizingParse(&p, None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyKittyTextSizing, "cmd == .kitty_text_sizing")
	oscB_kts_term51ExpectFields(t, cmd.KittyTextSizing, 1, 0, 0, 0, OSCParsersKittyTextSizingVAlignTop, OSCParsersKittyTextSizingHAlignLeft)
	expectEqualStrings(t, "x", string(cmd.KittyTextSizing.Text))
}

// Zig: test "term51: parse: malformed parameters are skipped"
func TestOSCParsersKittyTextSizing_term51_parse_malformed_parameters_are_skipped(t *testing.T) {
	for _, input := range []string{
		// no '='
		"66;s;x",
		"66;s2;x",
		// key is not exactly one byte
		"66;=2;x",
		"66;ss=2;x",
		"66; s=2;x",
		// unknown or upper case key
		"66;x=2;x",
		"66;S=2;x",
		// bad value
		"66;s=;x",
		"66;s= 2;x",
		"66;s=two;x",
		// empty parameters
		"66;:;x",
		"66;:::;x",
	} {
		p := OSCParserInit(false)
		oscB_kts_term51Feed(&p, input)
		cmd := oscB_kts_unwrap(t, OSCParsersKittyTextSizingParse(&p, None[uint8]()))
		expect(t, cmd.Tag == OSCCommandKeyKittyTextSizing, "cmd == .kitty_text_sizing")
		oscB_kts_term51ExpectFields(t, cmd.KittyTextSizing, 1, 0, 0, 0, OSCParsersKittyTextSizingVAlignTop, OSCParsersKittyTextSizingHAlignLeft)
		expectEqualStrings(t, "x", string(cmd.KittyTextSizing.Text))
	}
}

// Zig: test "term51: parse: a malformed parameter does not affect the others"
func TestOSCParsersKittyTextSizing_term51_parse_a_malformed_parameter_does_not_affect_the_others(t *testing.T) {
	p := OSCParserInit(false)
	oscB_kts_term51Feed(&p, "66;:bogus:s=3::x=1:w=9:n=4:;x")
	cmd := oscB_kts_unwrap(t, OSCParsersKittyTextSizingParse(&p, None[uint8]()))
	oscB_kts_term51ExpectFields(t, cmd.KittyTextSizing, 3, 0, 4, 0, OSCParsersKittyTextSizingVAlignTop, OSCParsersKittyTextSizingHAlignLeft)
}

// Zig: test "term51: parse: repeated parameters"
func TestOSCParsersKittyTextSizing_term51_parse_repeated_parameters(t *testing.T) {
	{
		// The last valid value wins.
		p := OSCParserInit(false)
		oscB_kts_term51Feed(&p, "66;s=2:s=3;x")
		cmd := oscB_kts_unwrap(t, OSCParsersKittyTextSizingParse(&p, None[uint8]()))
		expectEqual(t, uint8(3), cmd.KittyTextSizing.Scale)
	}
	{
		// An invalid later value does not undo an earlier one.
		p := OSCParserInit(false)
		oscB_kts_term51Feed(&p, "66;s=2:s=9:s=0:s=;x")
		cmd := oscB_kts_unwrap(t, OSCParsersKittyTextSizingParse(&p, None[uint8]()))
		expectEqual(t, uint8(2), cmd.KittyTextSizing.Scale)
	}
}

// Zig: test "term51: parse: only the text between the first and second equals is the value"
func TestOSCParsersKittyTextSizing_term51_parse_only_the_text_between_the_first_and_second_equals_is_the_value(t *testing.T) {
	p := OSCParserInit(false)
	oscB_kts_term51Feed(&p, "66;s=2=3:w=4=junk;x")
	cmd := oscB_kts_unwrap(t, OSCParsersKittyTextSizingParse(&p, None[uint8]()))
	oscB_kts_term51ExpectFields(t, cmd.KittyTextSizing, 2, 4, 0, 0, OSCParsersKittyTextSizingVAlignTop, OSCParsersKittyTextSizingHAlignLeft)
}

// Zig: test "term51: parse: later semicolons belong to the text"
func TestOSCParsersKittyTextSizing_term51_parse_later_semicolons_belong_to_the_text(t *testing.T) {
	p := OSCParserInit(false)
	oscB_kts_term51Feed(&p, "66;s=2;a;b:c=d;")
	cmd := oscB_kts_unwrap(t, OSCParsersKittyTextSizingParse(&p, None[uint8]()))
	expectEqual(t, uint8(2), cmd.KittyTextSizing.Scale)
	expectEqualStrings(t, "a;b:c=d;", string(cmd.KittyTextSizing.Text))
}

// Zig: test "term51: parse: text must be escape code safe UTF-8"
func TestOSCParsersKittyTextSizing_term51_parse_text_must_be_escape_code_safe_UTF_8(t *testing.T) {
	// Accepted: printable ASCII, U+00A0 (first code point after C1) and
	// multi-byte sequences.
	for _, text := range []string{" ", "~", "\xc2\xa0", "\xe2\x82\xac", "\xf4\x8f\xbf\xbf"} {
		p := OSCParserInit(false)
		oscB_kts_term51Feed(&p, "66;;"+text)
		cmd := oscB_kts_unwrap(t, OSCParsersKittyTextSizingParse(&p, None[uint8]()))
		expectEqualStrings(t, text, string(cmd.KittyTextSizing.Text))
	}

	// Rejected: C0 controls (including NUL, tab and newline), DEL, C1
	// controls encoded as UTF-8, and invalid UTF-8.
	for _, text := range []string{
		"\x00",
		"a\x00b",
		"\x01",
		"\t",
		"\n",
		"\x1f",
		"\x7f",
		"\xc2\x80",
		"\xc2\x9f",
		"\x80",
		"\xff",
		"\xe2\x82",
		"\xc0\x80",
		"\xed\xa0\x80",
		"\xf4\x90\x80\x80",
	} {
		p := OSCParserInit(false)
		oscB_kts_term51Feed(&p, "66;s=2;"+text)
		expect(t, OSCParsersKittyTextSizingParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
}

// Zig: test "term51: parse: payload length limit with allocator"
func TestOSCParsersKittyTextSizing_term51_parse_payload_length_limit_with_allocator(t *testing.T) {
	expectEqual(t, 4096, OSCParsersKittyTextSizingMaxPayloadLength)
	{
		text := strings.Repeat("a", OSCParsersKittyTextSizingMaxPayloadLength)
		p := OSCParserInit(true)
		defer p.Deinit()
		oscB_kts_term51Feed(&p, "66;s=2;"+text)
		cmd := oscB_kts_unwrap(t, OSCParsersKittyTextSizingParse(&p, None[uint8]()))
		expectEqual(t, uint8(2), cmd.KittyTextSizing.Scale)
		expectEqualStrings(t, text, string(cmd.KittyTextSizing.Text))
	}
	{
		text := strings.Repeat("a", OSCParsersKittyTextSizingMaxPayloadLength+1)
		p := OSCParserInit(true)
		defer p.Deinit()
		oscB_kts_term51Feed(&p, "66;s=2;"+text)
		expect(t, p.state == OSCParserState66, "p.state == .@\"66\"")
		expect(t, OSCParsersKittyTextSizingParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
	{
		// The limit is on bytes, not on code points, and only on the
		// payload: long metadata does not count.
		text := strings.Repeat("\xe2\x82\xac", 1365) + "a"
		expectEqual(t, OSCParsersKittyTextSizingMaxPayloadLength, len(text))
		p := OSCParserInit(true)
		defer p.Deinit()
		oscB_kts_term51Feed(&p, "66;"+strings.Repeat("s=2:", 100)+"w=3;"+text)
		cmd := oscB_kts_unwrap(t, OSCParsersKittyTextSizingParse(&p, None[uint8]()))
		oscB_kts_term51ExpectFields(t, cmd.KittyTextSizing, 2, 3, 0, 0, OSCParsersKittyTextSizingVAlignTop, OSCParsersKittyTextSizingHAlignLeft)
		expectEqual(t, OSCParsersKittyTextSizingMaxPayloadLength, len(cmd.KittyTextSizing.Text))
	}
}

// Zig: test "term51: parse: buffer limit without allocator"
func TestOSCParsersKittyTextSizing_term51_parse_buffer_limit_without_allocator(t *testing.T) {
	// Without an allocator the fixed buffer is used: the ';', the payload
	// and the NUL terminator must fit in MAX_BUF, which is less than
	// max_payload_length.
	{
		text := strings.Repeat("a", OSCParserMAX_BUF-2)
		p := OSCParserInit(false)
		oscB_kts_term51Feed(&p, "66;;"+text)
		cmd := oscB_kts_unwrap(t, OSCParsersKittyTextSizingParse(&p, None[uint8]()))
		expectEqualStrings(t, text, string(cmd.KittyTextSizing.Text))
	}
	{
		text := strings.Repeat("a", OSCParserMAX_BUF-1)
		p := OSCParserInit(false)
		oscB_kts_term51Feed(&p, "66;;"+text)
		expect(t, p.state == OSCParserState66, "p.state == .@\"66\"")
		expect(t, OSCParsersKittyTextSizingParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
}
