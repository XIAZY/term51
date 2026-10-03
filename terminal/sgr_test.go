// Tests translated one for one from src/terminal/sgr.zig (Ghostty v1.3.1 plus
// the term51 additional tests). Each Go test carries the exact Zig title in a
// "// Zig: test" comment. The helpers expect, expectEqual and
// expectEqualSlices and zigUnwrap are defined in Parser_test.go.
//
// Not translated (C ABI, see docs/mapping.md):
//   - test "sgr: Attribute C compat"
//   - test "term51: Attribute.Unknown.cval: exposes both slices as pointer and length"
//   - test "term51: Attribute.Unknown.cval: empty slices"
//   - test "term51: Attribute.Underline.cval: numeric value of each style"
//
// Translation notes:
//   - `try testing.expect(v == .bold)` is expect(t, v.Tag == SGRAttributeTagBold, ...).
//   - `p.next() == null` is `!p.Next().Valid` and `p.next().?` is zigUnwrap.
//   - Reading a union payload without a preceding tag check (`x.?.underline`)
//     goes through sgr_field, which checks the tag as Zig's safety-checked
//     union field access does.
//   - `.{ .params = ... }` is SGRParser{params: ...}; `.empty` is
//     SGRParserEmpty. A `sep: { var list = SepList.initEmpty(); ... }` block
//     is written out with StaticBitSet24InitEmpty and set.
//   - The colon list of term51Parser is []uint here (Zig: []const usize).

package terminal

import (
	"testing"
)

// sgr_field is Zig's safety-checked union field access `a.<tag>`: it fails if
// the active tag is not tag. The caller then reads the matching Go field.
func sgr_field(t *testing.T, a SGRAttribute, tag SGRAttributeTag) SGRAttribute {
	t.Helper()
	if a.Tag != tag {
		t.Fatalf("access of union field %v while field %v is active", tag, a.Tag)
	}
	return a
}

// sgr_testParse is Zig's testParse.
func sgr_testParse(t *testing.T, params []uint16) SGRAttribute {
	t.Helper()
	p := SGRParser{params: params}
	return zigUnwrap(t, p.Next())
}

// sgr_testParseColon is Zig's testParseColon.
func sgr_testParseColon(t *testing.T, params []uint16) SGRAttribute {
	t.Helper()
	p := SGRParser{params: params}
	// Mark all parameters except the last as having a colon after.
	for i := 0; i < len(params)-1; i++ {
		p.params_sep.set(i)
	}
	return zigUnwrap(t, p.Next())
}

// sgr_term51Parser is the term51 test helper term51Parser: a parser whose
// separators after the given parameter indexes are colons.
func sgr_term51Parser(params []uint16, colons []uint) SGRParser {
	p := SGRParser{params: params}
	for _, i := range colons {
		p.params_sep.set(int(i))
	}
	return p
}

// Zig: test "sgr: Parser"
func TestSGR_sgr_Parser(t *testing.T) {
	expect(t, sgr_testParse(t, []uint16{}).Tag == SGRAttributeTagUnset, "testParse(&[_]u16{}) == .unset")
	expect(t, sgr_testParse(t, []uint16{0}).Tag == SGRAttributeTagUnset, "testParse(&[_]u16{0}) == .unset")

	{
		v := sgr_testParse(t, []uint16{38, 2, 40, 44, 52})
		expect(t, v.Tag == SGRAttributeTagDirectColorFg, "v == .direct_color_fg")
		expectEqual(t, uint8(40), v.DirectColorFg.r)
		expectEqual(t, uint8(44), v.DirectColorFg.g)
		expectEqual(t, uint8(52), v.DirectColorFg.b)
	}

	expect(t, sgr_testParse(t, []uint16{38, 2, 44, 52}).Tag == SGRAttributeTagUnknown, "testParse(&[_]u16{ 38, 2, 44, 52 }) == .unknown")

	{
		v := sgr_testParse(t, []uint16{48, 2, 40, 44, 52})
		expect(t, v.Tag == SGRAttributeTagDirectColorBg, "v == .direct_color_bg")
		expectEqual(t, uint8(40), v.DirectColorBg.r)
		expectEqual(t, uint8(44), v.DirectColorBg.g)
		expectEqual(t, uint8(52), v.DirectColorBg.b)
	}

	expect(t, sgr_testParse(t, []uint16{48, 2, 44, 52}).Tag == SGRAttributeTagUnknown, "testParse(&[_]u16{ 48, 2, 44, 52 }) == .unknown")
}

// Zig: test "sgr: Parser multiple"
func TestSGR_sgr_Parser_multiple(t *testing.T) {
	p := SGRParser{params: []uint16{0, 38, 2, 40, 44, 52}}
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagUnset, "p.next().? == .unset")
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagDirectColorFg, "p.next().? == .direct_color_fg")
	expect(t, !p.Next().Valid, "p.next() == null")
	expect(t, !p.Next().Valid, "p.next() == null")
}

// Zig: test "sgr: unsupported with colon"
func TestSGR_sgr_unsupported_with_colon(t *testing.T) {
	sep := StaticBitSet24InitEmpty()
	sep.set(0)
	p := SGRParser{
		params:     []uint16{0, 4, 1},
		params_sep: sep,
	}
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagUnknown, "p.next().? == .unknown")
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagBold, "p.next().? == .bold")
	expect(t, !p.Next().Valid, "p.next() == null")
}

// Zig: test "sgr: unsupported with multiple colon"
func TestSGR_sgr_unsupported_with_multiple_colon(t *testing.T) {
	sep := StaticBitSet24InitEmpty()
	sep.set(0)
	sep.set(1)
	p := SGRParser{
		params:     []uint16{0, 4, 2, 1},
		params_sep: sep,
	}
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagUnknown, "p.next().? == .unknown")
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagBold, "p.next().? == .bold")
	expect(t, !p.Next().Valid, "p.next() == null")
}

// Zig: test "sgr: bold"
func TestSGR_sgr_bold(t *testing.T) {
	{
		v := sgr_testParse(t, []uint16{1})
		expect(t, v.Tag == SGRAttributeTagBold, "v == .bold")
	}

	{
		v := sgr_testParse(t, []uint16{22})
		expect(t, v.Tag == SGRAttributeTagResetBold, "v == .reset_bold")
	}
}

// Zig: test "sgr: italic"
func TestSGR_sgr_italic(t *testing.T) {
	{
		v := sgr_testParse(t, []uint16{3})
		expect(t, v.Tag == SGRAttributeTagItalic, "v == .italic")
	}

	{
		v := sgr_testParse(t, []uint16{23})
		expect(t, v.Tag == SGRAttributeTagResetItalic, "v == .reset_italic")
	}
}

// Zig: test "sgr: underline"
func TestSGR_sgr_underline(t *testing.T) {
	{
		v := sgr_testParse(t, []uint16{4})
		expect(t, v.Tag == SGRAttributeTagUnderline, "v == .underline")
	}

	{
		v := sgr_testParse(t, []uint16{24})
		expect(t, v.Tag == SGRAttributeTagUnderline, "v == .underline")
		expect(t, v.Underline == SGRAttributeUnderlineNone, "v.underline == .none")
	}
}

// Zig: test "sgr: underline styles"
func TestSGR_sgr_underline_styles(t *testing.T) {
	{
		v := sgr_testParseColon(t, []uint16{4, 2})
		expect(t, v.Tag == SGRAttributeTagUnderline, "v == .underline")
		expect(t, v.Underline == SGRAttributeUnderlineDouble, "v.underline == .double")
	}

	{
		v := sgr_testParseColon(t, []uint16{4, 0})
		expect(t, v.Tag == SGRAttributeTagUnderline, "v == .underline")
		expect(t, v.Underline == SGRAttributeUnderlineNone, "v.underline == .none")
	}

	{
		v := sgr_testParseColon(t, []uint16{4, 1})
		expect(t, v.Tag == SGRAttributeTagUnderline, "v == .underline")
		expect(t, v.Underline == SGRAttributeUnderlineSingle, "v.underline == .single")
	}

	{
		v := sgr_testParseColon(t, []uint16{4, 3})
		expect(t, v.Tag == SGRAttributeTagUnderline, "v == .underline")
		expect(t, v.Underline == SGRAttributeUnderlineCurly, "v.underline == .curly")
	}

	{
		v := sgr_testParseColon(t, []uint16{4, 4})
		expect(t, v.Tag == SGRAttributeTagUnderline, "v == .underline")
		expect(t, v.Underline == SGRAttributeUnderlineDotted, "v.underline == .dotted")
	}

	{
		v := sgr_testParseColon(t, []uint16{4, 5})
		expect(t, v.Tag == SGRAttributeTagUnderline, "v == .underline")
		expect(t, v.Underline == SGRAttributeUnderlineDashed, "v.underline == .dashed")
	}
}

// Zig: test "sgr: underline style with more"
func TestSGR_sgr_underline_style_with_more(t *testing.T) {
	sep := StaticBitSet24InitEmpty()
	sep.set(0)
	p := SGRParser{
		params:     []uint16{4, 2, 1},
		params_sep: sep,
	}

	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagUnderline, "p.next().? == .underline")
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagBold, "p.next().? == .bold")
	expect(t, !p.Next().Valid, "p.next() == null")
}

// Zig: test "sgr: underline style with too many colons"
func TestSGR_sgr_underline_style_with_too_many_colons(t *testing.T) {
	sep := StaticBitSet24InitEmpty()
	sep.set(0)
	sep.set(1)
	p := SGRParser{
		params:     []uint16{4, 2, 3, 1},
		params_sep: sep,
	}

	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagUnknown, "p.next().? == .unknown")
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagBold, "p.next().? == .bold")
	expect(t, !p.Next().Valid, "p.next() == null")
}

// Zig: test "sgr: blink"
func TestSGR_sgr_blink(t *testing.T) {
	{
		v := sgr_testParse(t, []uint16{5})
		expect(t, v.Tag == SGRAttributeTagBlink, "v == .blink")
	}

	{
		v := sgr_testParse(t, []uint16{6})
		expect(t, v.Tag == SGRAttributeTagBlink, "v == .blink")
	}

	{
		v := sgr_testParse(t, []uint16{25})
		expect(t, v.Tag == SGRAttributeTagResetBlink, "v == .reset_blink")
	}
}

// Zig: test "sgr: inverse"
func TestSGR_sgr_inverse(t *testing.T) {
	{
		v := sgr_testParse(t, []uint16{7})
		expect(t, v.Tag == SGRAttributeTagInverse, "v == .inverse")
	}

	{
		v := sgr_testParse(t, []uint16{27})
		expect(t, v.Tag == SGRAttributeTagResetInverse, "v == .reset_inverse")
	}
}

// Zig: test "sgr: strikethrough"
func TestSGR_sgr_strikethrough(t *testing.T) {
	{
		v := sgr_testParse(t, []uint16{9})
		expect(t, v.Tag == SGRAttributeTagStrikethrough, "v == .strikethrough")
	}

	{
		v := sgr_testParse(t, []uint16{29})
		expect(t, v.Tag == SGRAttributeTagResetStrikethrough, "v == .reset_strikethrough")
	}
}

// Zig: test "sgr: 8 color"
func TestSGR_sgr_8_color(t *testing.T) {
	p := SGRParser{params: []uint16{31, 43, 90, 103}}

	{
		v := zigUnwrap(t, p.Next())
		expect(t, v.Tag == SGRAttributeTag8Fg, `v == .@"8_fg"`)
		expect(t, v.X8Fg == ColorNameRed, `v.@"8_fg" == .red`)
	}

	{
		v := zigUnwrap(t, p.Next())
		expect(t, v.Tag == SGRAttributeTag8Bg, `v == .@"8_bg"`)
		expect(t, v.X8Bg == ColorNameYellow, `v.@"8_bg" == .yellow`)
	}

	{
		v := zigUnwrap(t, p.Next())
		expect(t, v.Tag == SGRAttributeTag8BrightFg, `v == .@"8_bright_fg"`)
		expect(t, v.X8BrightFg == ColorNameBrightBlack, `v.@"8_bright_fg" == .bright_black`)
	}

	{
		v := zigUnwrap(t, p.Next())
		expect(t, v.Tag == SGRAttributeTag8BrightBg, `v == .@"8_bright_bg"`)
		expect(t, v.X8BrightBg == ColorNameBrightYellow, `v.@"8_bright_bg" == .bright_yellow`)
	}
}

// Zig: test "sgr: 256 color"
func TestSGR_sgr_256_color(t *testing.T) {
	p := SGRParser{params: []uint16{38, 5, 161, 48, 5, 236}}
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTag256Fg, `p.next().? == .@"256_fg"`)
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTag256Bg, `p.next().? == .@"256_bg"`)
	expect(t, !p.Next().Valid, "p.next() == null")
}

// Zig: test "sgr: 256 color underline"
func TestSGR_sgr_256_color_underline(t *testing.T) {
	p := SGRParser{params: []uint16{58, 5, 9}}
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTag256UnderlineColor, `p.next().? == .@"256_underline_color"`)
	expect(t, !p.Next().Valid, "p.next() == null")
}

// Zig: test "sgr: 24-bit bg color"
func TestSGR_sgr_24_bit_bg_color(t *testing.T) {
	{
		v := sgr_testParseColon(t, []uint16{48, 2, 1, 2, 3})
		expect(t, v.Tag == SGRAttributeTagDirectColorBg, "v == .direct_color_bg")
		expectEqual(t, uint8(1), v.DirectColorBg.r)
		expectEqual(t, uint8(2), v.DirectColorBg.g)
		expectEqual(t, uint8(3), v.DirectColorBg.b)
	}
}

// Zig: test "sgr: underline color"
func TestSGR_sgr_underline_color(t *testing.T) {
	{
		v := sgr_testParseColon(t, []uint16{58, 2, 1, 2, 3})
		expect(t, v.Tag == SGRAttributeTagUnderlineColor, "v == .underline_color")
		expectEqual(t, uint8(1), v.UnderlineColor.r)
		expectEqual(t, uint8(2), v.UnderlineColor.g)
		expectEqual(t, uint8(3), v.UnderlineColor.b)
	}

	{
		v := sgr_testParseColon(t, []uint16{58, 2, 0, 1, 2, 3})
		expect(t, v.Tag == SGRAttributeTagUnderlineColor, "v == .underline_color")
		expectEqual(t, uint8(1), v.UnderlineColor.r)
		expectEqual(t, uint8(2), v.UnderlineColor.g)
		expectEqual(t, uint8(3), v.UnderlineColor.b)
	}
}

// Zig: test "sgr: reset underline color"
func TestSGR_sgr_reset_underline_color(t *testing.T) {
	p := SGRParser{params: []uint16{59}}
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagResetUnderlineColor, "p.next().? == .reset_underline_color")
}

// Zig: test "sgr: invisible"
func TestSGR_sgr_invisible(t *testing.T) {
	p := SGRParser{params: []uint16{8, 28}}
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagInvisible, "p.next().? == .invisible")
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagResetInvisible, "p.next().? == .reset_invisible")
}

// Zig: test "sgr: underline, bg, and fg"
func TestSGR_sgr_underline_bg_and_fg(t *testing.T) {
	p := SGRParser{
		params: []uint16{4, 38, 2, 255, 247, 219, 48, 2, 242, 93, 147, 4},
	}
	{
		v := zigUnwrap(t, p.Next())
		expect(t, v.Tag == SGRAttributeTagUnderline, "v == .underline")
		expectEqual(t, SGRAttributeUnderlineSingle, v.Underline)
	}
	{
		v := zigUnwrap(t, p.Next())
		expect(t, v.Tag == SGRAttributeTagDirectColorFg, "v == .direct_color_fg")
		expectEqual(t, uint8(255), v.DirectColorFg.r)
		expectEqual(t, uint8(247), v.DirectColorFg.g)
		expectEqual(t, uint8(219), v.DirectColorFg.b)
	}
	{
		v := zigUnwrap(t, p.Next())
		expect(t, v.Tag == SGRAttributeTagDirectColorBg, "v == .direct_color_bg")
		expectEqual(t, uint8(242), v.DirectColorBg.r)
		expectEqual(t, uint8(93), v.DirectColorBg.g)
		expectEqual(t, uint8(147), v.DirectColorBg.b)
	}
	{
		v := zigUnwrap(t, p.Next())
		expect(t, v.Tag == SGRAttributeTagUnderline, "v == .underline")
		expectEqual(t, SGRAttributeUnderlineSingle, v.Underline)
	}
}

// Zig: test "sgr: direct color fg missing color"
func TestSGR_sgr_direct_color_fg_missing_color(t *testing.T) {
	// This used to crash
	p := SGRParser{params: []uint16{38, 5}}
	for p.Next().Valid {
	}
}

// Zig: test "sgr: direct color bg missing color"
func TestSGR_sgr_direct_color_bg_missing_color(t *testing.T) {
	// This used to crash
	p := SGRParser{params: []uint16{48, 5}}
	for p.Next().Valid {
	}
}

// Zig: test "sgr: direct fg/bg/underline ignore optional color space"
func TestSGR_sgr_direct_fg_bg_underline_ignore_optional_color_space(t *testing.T) {
	// These behaviors have been verified against xterm.

	// Colon version should skip the optional color space identifier
	{
		// 3 8 : 2 : Pi : Pr : Pg : Pb
		v := sgr_testParseColon(t, []uint16{38, 2, 0, 1, 2, 3})
		expect(t, v.Tag == SGRAttributeTagDirectColorFg, "v == .direct_color_fg")
		expectEqual(t, uint8(1), v.DirectColorFg.r)
		expectEqual(t, uint8(2), v.DirectColorFg.g)
		expectEqual(t, uint8(3), v.DirectColorFg.b)
	}
	{
		// 4 8 : 2 : Pi : Pr : Pg : Pb
		v := sgr_testParseColon(t, []uint16{48, 2, 0, 1, 2, 3})
		expect(t, v.Tag == SGRAttributeTagDirectColorBg, "v == .direct_color_bg")
		expectEqual(t, uint8(1), v.DirectColorBg.r)
		expectEqual(t, uint8(2), v.DirectColorBg.g)
		expectEqual(t, uint8(3), v.DirectColorBg.b)
	}
	{
		// 5 8 : 2 : Pi : Pr : Pg : Pb
		v := sgr_testParseColon(t, []uint16{58, 2, 0, 1, 2, 3})
		expect(t, v.Tag == SGRAttributeTagUnderlineColor, "v == .underline_color")
		expectEqual(t, uint8(1), v.UnderlineColor.r)
		expectEqual(t, uint8(2), v.UnderlineColor.g)
		expectEqual(t, uint8(3), v.UnderlineColor.b)
	}

	// Semicolon version should not parse optional color space identifier
	{
		// 3 8 ; 2 ; Pr ; Pg ; Pb
		v := sgr_testParse(t, []uint16{38, 2, 0, 1, 2, 3})
		expect(t, v.Tag == SGRAttributeTagDirectColorFg, "v == .direct_color_fg")
		expectEqual(t, uint8(0), v.DirectColorFg.r)
		expectEqual(t, uint8(1), v.DirectColorFg.g)
		expectEqual(t, uint8(2), v.DirectColorFg.b)
	}
	{
		// 4 8 ; 2 ; Pr ; Pg ; Pb
		v := sgr_testParse(t, []uint16{48, 2, 0, 1, 2, 3})
		expect(t, v.Tag == SGRAttributeTagDirectColorBg, "v == .direct_color_bg")
		expectEqual(t, uint8(0), v.DirectColorBg.r)
		expectEqual(t, uint8(1), v.DirectColorBg.g)
		expectEqual(t, uint8(2), v.DirectColorBg.b)
	}
	{
		// 5 8 ; 2 ; Pr ; Pg ; Pb
		v := sgr_testParse(t, []uint16{58, 2, 0, 1, 2, 3})
		expect(t, v.Tag == SGRAttributeTagUnderlineColor, "v == .underline_color")
		expectEqual(t, uint8(0), v.UnderlineColor.r)
		expectEqual(t, uint8(1), v.UnderlineColor.g)
		expectEqual(t, uint8(2), v.UnderlineColor.b)
	}
}

// Zig: test "sgr: direct fg colon with too many colons"
func TestSGR_sgr_direct_fg_colon_with_too_many_colons(t *testing.T) {
	sep := StaticBitSet24InitEmpty()
	for idx := 0; idx < 6; idx++ {
		sep.set(idx)
	}
	p := SGRParser{
		params:     []uint16{38, 2, 0, 1, 2, 3, 4, 1},
		params_sep: sep,
	}

	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagUnknown, "p.next().? == .unknown")
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagBold, "p.next().? == .bold")
	expect(t, !p.Next().Valid, "p.next() == null")
}

// Zig: test "sgr: direct fg colon with colorspace and extra param"
func TestSGR_sgr_direct_fg_colon_with_colorspace_and_extra_param(t *testing.T) {
	sep := StaticBitSet24InitEmpty()
	for idx := 0; idx < 5; idx++ {
		sep.set(idx)
	}
	p := SGRParser{
		params:     []uint16{38, 2, 0, 1, 2, 3, 1},
		params_sep: sep,
	}

	{
		v := zigUnwrap(t, p.Next())
		expect(t, v.Tag == SGRAttributeTagDirectColorFg, "v == .direct_color_fg")
		expectEqual(t, uint8(1), v.DirectColorFg.r)
		expectEqual(t, uint8(2), v.DirectColorFg.g)
		expectEqual(t, uint8(3), v.DirectColorFg.b)
	}

	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagBold, "p.next().? == .bold")
	expect(t, !p.Next().Valid, "p.next() == null")
}

// Zig: test "sgr: direct fg colon no colorspace and extra param"
func TestSGR_sgr_direct_fg_colon_no_colorspace_and_extra_param(t *testing.T) {
	sep := StaticBitSet24InitEmpty()
	for idx := 0; idx < 4; idx++ {
		sep.set(idx)
	}
	p := SGRParser{
		params:     []uint16{38, 2, 1, 2, 3, 1},
		params_sep: sep,
	}

	{
		v := zigUnwrap(t, p.Next())
		expect(t, v.Tag == SGRAttributeTagDirectColorFg, "v == .direct_color_fg")
		expectEqual(t, uint8(1), v.DirectColorFg.r)
		expectEqual(t, uint8(2), v.DirectColorFg.g)
		expectEqual(t, uint8(3), v.DirectColorFg.b)
	}

	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagBold, "p.next().? == .bold")
	expect(t, !p.Next().Valid, "p.next() == null")
}

// Kakoune sent this complex SGR sequence that caused invalid behavior.
//
// Zig: test "sgr: kakoune input"
func TestSGR_sgr_kakoune_input(t *testing.T) {
	// This used to crash
	sep := StaticBitSet24InitEmpty()
	sep.set(1)
	sep.set(8)
	sep.set(9)
	sep.set(10)
	sep.set(11)
	sep.set(12)
	p := SGRParser{
		params:     []uint16{0, 4, 3, 38, 2, 175, 175, 215, 58, 2, 0, 190, 80, 70},
		params_sep: sep,
	}

	{
		v := zigUnwrap(t, p.Next())
		expect(t, v.Tag == SGRAttributeTagUnset, "v == .unset")
	}
	{
		v := zigUnwrap(t, p.Next())
		expect(t, v.Tag == SGRAttributeTagUnderline, "v == .underline")
		expectEqual(t, SGRAttributeUnderlineCurly, v.Underline)
	}
	{
		v := zigUnwrap(t, p.Next())
		expect(t, v.Tag == SGRAttributeTagDirectColorFg, "v == .direct_color_fg")
		expectEqual(t, uint8(175), v.DirectColorFg.r)
		expectEqual(t, uint8(175), v.DirectColorFg.g)
		expectEqual(t, uint8(215), v.DirectColorFg.b)
	}
	{
		v := zigUnwrap(t, p.Next())
		expect(t, v.Tag == SGRAttributeTagUnderlineColor, "v == .underline_color")
		expectEqual(t, uint8(190), v.UnderlineColor.r)
		expectEqual(t, uint8(80), v.UnderlineColor.g)
		expectEqual(t, uint8(70), v.UnderlineColor.b)
	}

	//try testing.expect(p.next() == null);
}

// Discussion #5930, another input sent by kakoune
//
// Zig: test "sgr: kakoune input issue underline, fg, and bg"
func TestSGR_sgr_kakoune_input_issue_underline_fg_and_bg(t *testing.T) {
	// echo -e "\033[4:3;38;2;51;51;51;48;2;170;170;170;58;2;255;97;136mset everything in one sequence, broken\033[m"

	// This used to crash
	sep := StaticBitSet24InitEmpty()
	sep.set(0)
	p := SGRParser{
		params:     []uint16{4, 3, 38, 2, 51, 51, 51, 48, 2, 170, 170, 170, 58, 2, 255, 97, 136},
		params_sep: sep,
	}

	{
		v := zigUnwrap(t, p.Next())
		expect(t, v.Tag == SGRAttributeTagUnderline, "v == .underline")
		expectEqual(t, SGRAttributeUnderlineCurly, v.Underline)
	}

	{
		v := zigUnwrap(t, p.Next())
		expect(t, v.Tag == SGRAttributeTagDirectColorFg, "v == .direct_color_fg")
		expectEqual(t, uint8(51), v.DirectColorFg.r)
		expectEqual(t, uint8(51), v.DirectColorFg.g)
		expectEqual(t, uint8(51), v.DirectColorFg.b)
	}

	{
		v := zigUnwrap(t, p.Next())
		expect(t, v.Tag == SGRAttributeTagDirectColorBg, "v == .direct_color_bg")
		expectEqual(t, uint8(170), v.DirectColorBg.r)
		expectEqual(t, uint8(170), v.DirectColorBg.g)
		expectEqual(t, uint8(170), v.DirectColorBg.b)
	}

	{
		v := zigUnwrap(t, p.Next())
		expect(t, v.Tag == SGRAttributeTagUnderlineColor, "v == .underline_color")
		expectEqual(t, uint8(255), v.UnderlineColor.r)
		expectEqual(t, uint8(97), v.UnderlineColor.g)
		expectEqual(t, uint8(136), v.UnderlineColor.b)
	}

	expect(t, !p.Next().Valid, "p.next() == null")
}

// Fuzz crash: afl-out/stream/default/crashes/id:000021
// Input "ESC [ 5 8 : 4 : m" produces params [58, 4] with colon
// separator bits set at indices 0 and 1. The trailing colon causes
// the second iteration to see param 4 (underline) with a colon,
// triggering assert(slice.len >= 2) with slice.len == 1.
//
// Zig: test "sgr: underline colon with trailing separator and short slice"
func TestSGR_sgr_underline_colon_with_trailing_separator_and_short_slice(t *testing.T) {
	sep := StaticBitSet24InitEmpty()
	sep.set(0)
	sep.set(1)
	p := SGRParser{
		params:     []uint16{58, 4},
		params_sep: sep,
	}

	// 58:4 is not a valid underline color (sub-param 4 is not 2 or 5),
	// so it falls through as unknown.
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagUnknown, "p.next().? == .unknown")

	// Param 4 with a trailing colon but no sub-param is malformed,
	// so it also falls through as unknown rather than panicking.
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagUnknown, "p.next().? == .unknown")

	expect(t, !p.Next().Valid, "p.next() == null")
}

// ---------------------------------------------------------------------------
// term51: additional tests. Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

// Zig: test "term51: Parser.next: every single parameter from 0 to 110"
func TestSGR_term51_Parser_next_every_single_parameter_from_0_to_110(t *testing.T) {
	for v := uint16(0); v <= 110; v++ {
		p := SGRParser{params: []uint16{v}}
		a := zigUnwrap(t, p.Next())
		// Zig: const tag: Attribute.Tag = a;  (union to its tag enum coercion)
		tag := a.Tag
		var want SGRAttributeTag
		switch {
		case v == 0:
			want = SGRAttributeTagUnset
		case v == 1:
			want = SGRAttributeTagBold
		case v == 2:
			want = SGRAttributeTagFaint
		case v == 3:
			want = SGRAttributeTagItalic
		case v == 4:
			want = SGRAttributeTagUnderline
		case v == 5, v == 6:
			want = SGRAttributeTagBlink
		case v == 7:
			want = SGRAttributeTagInverse
		case v == 8:
			want = SGRAttributeTagInvisible
		case v == 9:
			want = SGRAttributeTagStrikethrough
		case v == 21:
			want = SGRAttributeTagUnderline
		case v == 22:
			want = SGRAttributeTagResetBold
		case v == 23:
			want = SGRAttributeTagResetItalic
		case v == 24:
			want = SGRAttributeTagUnderline
		case v == 25:
			want = SGRAttributeTagResetBlink
		case v == 27:
			want = SGRAttributeTagResetInverse
		case v == 28:
			want = SGRAttributeTagResetInvisible
		case v == 29:
			want = SGRAttributeTagResetStrikethrough
		case v >= 30 && v <= 37:
			want = SGRAttributeTag8Fg
		case v == 39:
			want = SGRAttributeTagResetFg
		case v >= 40 && v <= 47:
			want = SGRAttributeTag8Bg
		case v == 49:
			want = SGRAttributeTagResetBg
		case v == 53:
			want = SGRAttributeTagOverline
		case v == 55:
			want = SGRAttributeTagResetOverline
		case v == 59:
			want = SGRAttributeTagResetUnderlineColor
		case v >= 90 && v <= 97:
			want = SGRAttributeTag8BrightFg
		case v >= 100 && v <= 107:
			want = SGRAttributeTag8BrightBg
		default:
			// 38, 48 and 58 need more parameters; everything else is
			// not an attribute upstream knows
			want = SGRAttributeTagUnknown
		}
		expectEqual(t, want, tag)
		// a single parameter yields exactly one attribute
		expect(t, !p.Next().Valid, "p.next() == null")
	}
}

// Zig: test "term51: Parser.next: values of the single-parameter attributes"
func TestSGR_term51_Parser_next_values_of_the_single_parameter_attributes(t *testing.T) {
	expectEqual(t, SGRAttributeUnderlineSingle, sgr_field(t, sgr_testParse(t, []uint16{4}), SGRAttributeTagUnderline).Underline)
	expectEqual(t, SGRAttributeUnderlineDouble, sgr_field(t, sgr_testParse(t, []uint16{21}), SGRAttributeTagUnderline).Underline)
	expectEqual(t, SGRAttributeUnderlineNone, sgr_field(t, sgr_testParse(t, []uint16{24}), SGRAttributeTagUnderline).Underline)
	expectEqual(t, ColorNameBlack, sgr_field(t, sgr_testParse(t, []uint16{30}), SGRAttributeTag8Fg).X8Fg)
	expectEqual(t, ColorNameWhite, sgr_field(t, sgr_testParse(t, []uint16{37}), SGRAttributeTag8Fg).X8Fg)
	expectEqual(t, ColorNameBlack, sgr_field(t, sgr_testParse(t, []uint16{40}), SGRAttributeTag8Bg).X8Bg)
	expectEqual(t, ColorNameWhite, sgr_field(t, sgr_testParse(t, []uint16{47}), SGRAttributeTag8Bg).X8Bg)
	expectEqual(t, ColorNameBrightBlack, sgr_field(t, sgr_testParse(t, []uint16{90}), SGRAttributeTag8BrightFg).X8BrightFg)
	expectEqual(t, ColorNameBrightWhite, sgr_field(t, sgr_testParse(t, []uint16{97}), SGRAttributeTag8BrightFg).X8BrightFg)
	expectEqual(t, ColorNameBrightBlack, sgr_field(t, sgr_testParse(t, []uint16{100}), SGRAttributeTag8BrightBg).X8BrightBg)
	expectEqual(t, ColorNameBrightWhite, sgr_field(t, sgr_testParse(t, []uint16{107}), SGRAttributeTag8BrightBg).X8BrightBg)
}

// Zig: test "term51: Parser.next: unknown parameters report the full list and the remainder"
func TestSGR_term51_Parser_next_unknown_parameters_report_the_full_list_and_the_remainder(t *testing.T) {
	p := SGRParser{params: []uint16{1, 26, 3}}
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagBold, "p.next().? == .bold")
	u := zigUnwrap(t, p.Next())
	expect(t, u.Tag == SGRAttributeTagUnknown, "u == .unknown")
	expectEqualSlices(t, []uint16{1, 26, 3}, u.Unknown.full)
	// partial is everything from the unknown parameter on
	expectEqualSlices(t, []uint16{26, 3}, u.Unknown.partial)
	// parsing continues with the next parameter
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagItalic, "p.next().? == .italic")
	expect(t, !p.Next().Valid, "p.next() == null")
}

// Zig: test "term51: Parser.next: large and boundary parameter values are unknown"
func TestSGR_term51_Parser_next_large_and_boundary_parameter_values_are_unknown(t *testing.T) {
	for _, v := range []uint16{10, 20, 26, 38, 48, 50, 52, 54, 56, 57, 58, 60, 89, 98, 99, 108, 255, 256, 1000, 65535} {
		p := SGRParser{params: []uint16{v}}
		a := zigUnwrap(t, p.Next())
		expect(t, a.Tag == SGRAttributeTagUnknown, "a == .unknown")
		expectEqualSlices(t, []uint16{v}, a.Unknown.partial)
	}
}

// Zig: test "term51: Parser.next: an empty list is one unset and then nothing"
func TestSGR_term51_Parser_next_an_empty_list_is_one_unset_and_then_nothing(t *testing.T) {
	p := SGRParserEmpty
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagUnset, "p.next().? == .unset")
	expect(t, !p.Next().Valid, "p.next() == null")
	expect(t, !p.Next().Valid, "p.next() == null")
	expect(t, !p.Next().Valid, "p.next() == null")
}

// Zig: test "term51: Parser.next: Parser.empty has no parameters and no colons"
func TestSGR_term51_Parser_next_Parser_empty_has_no_parameters_and_no_colons(t *testing.T) {
	p := SGRParserEmpty
	expectEqual(t, uint(0), uint(len(p.params)))
	expectEqual(t, uint(0), uint(p.params_sep.count()))
	expectEqual(t, uint(0), p.idx)
}

// Zig: test "term51: Parser.next: 256-colour index is truncated to 8 bits"
func TestSGR_term51_Parser_next_256_colour_index_is_truncated_to_8_bits(t *testing.T) {
	expectEqual(t, uint8(0), sgr_field(t, sgr_testParse(t, []uint16{38, 5, 0}), SGRAttributeTag256Fg).X256Fg)
	expectEqual(t, uint8(255), sgr_field(t, sgr_testParse(t, []uint16{38, 5, 255}), SGRAttributeTag256Fg).X256Fg)
	expectEqual(t, uint8(0), sgr_field(t, sgr_testParse(t, []uint16{38, 5, 256}), SGRAttributeTag256Fg).X256Fg)
	expectEqual(t, uint8(44), sgr_field(t, sgr_testParse(t, []uint16{38, 5, 300}), SGRAttributeTag256Fg).X256Fg)
	expectEqual(t, uint8(255), sgr_field(t, sgr_testParse(t, []uint16{48, 5, 65535}), SGRAttributeTag256Bg).X256Bg)
	expectEqual(t, uint8(1), sgr_field(t, sgr_testParse(t, []uint16{58, 5, 257}), SGRAttributeTag256UnderlineColor).X256UnderlineColor)
}

// Zig: test "term51: Parser.next: 256-colour consumes three parameters"
func TestSGR_term51_Parser_next_256_colour_consumes_three_parameters(t *testing.T) {
	p := SGRParser{params: []uint16{38, 5, 9, 1}}
	expectEqual(t, uint8(9), sgr_field(t, zigUnwrap(t, p.Next()), SGRAttributeTag256Fg).X256Fg)
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagBold, "p.next().? == .bold")
	expect(t, !p.Next().Valid, "p.next() == null")
}

// Zig: test "term51: Parser.next: extended colour with missing parts is unknown"
func TestSGR_term51_Parser_next_extended_colour_with_missing_parts_is_unknown(t *testing.T) {
	// (parameters, expected partial)
	cases := [][]uint16{
		{38}, {48}, {58},
		{38, 5}, {48, 5}, {58, 5},
		{38, 2}, {38, 2, 1}, {38, 2, 1, 2},
		{38, 0}, {38, 1, 5}, {38, 3, 1, 2, 3},
		{38, 4, 1, 2, 3, 4},
	}
	for _, params := range cases {
		p := SGRParser{params: params}
		a := zigUnwrap(t, p.Next())
		expect(t, a.Tag == SGRAttributeTagUnknown, "a == .unknown")
		expectEqualSlices(t, params, a.Unknown.partial)
	}
}

// Zig: test "term51: Parser.next: after an unknown extended colour the following parameters are parsed on their own"
func TestSGR_term51_Parser_next_after_an_unknown_extended_colour_the_following_parameters_are_parsed_on_their_own(t *testing.T) {
	// 38 with an unsupported type: 38 is unknown, then 0 resets, then 1 is bold
	p := SGRParser{params: []uint16{38, 0, 1}}
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagUnknown, "p.next().? == .unknown")
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagUnset, "p.next().? == .unset")
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagBold, "p.next().? == .bold")
	expect(t, !p.Next().Valid, "p.next() == null")
}

// Zig: test "term51: Parser.next: underline with a colon selects the style"
func TestSGR_term51_Parser_next_underline_with_a_colon_selects_the_style(t *testing.T) {
	styles := []struct {
		a uint16
		b SGRAttributeUnderline
	}{
		{0, SGRAttributeUnderlineNone}, {1, SGRAttributeUnderlineSingle}, {2, SGRAttributeUnderlineDouble},
		{3, SGRAttributeUnderlineCurly}, {4, SGRAttributeUnderlineDotted}, {5, SGRAttributeUnderlineDashed},
		// out of range falls back to single
		{6, SGRAttributeUnderlineSingle}, {7, SGRAttributeUnderlineSingle}, {255, SGRAttributeUnderlineSingle},
		{65535, SGRAttributeUnderlineSingle},
	}
	for _, s := range styles {
		p := sgr_term51Parser([]uint16{4, s.a}, []uint{0})
		expectEqual(t, s.b, sgr_field(t, zigUnwrap(t, p.Next()), SGRAttributeTagUnderline).Underline)
		expect(t, !p.Next().Valid, "p.next() == null")
	}
}

// Zig: test "term51: Parser.next: underline with a semicolon is two attributes"
func TestSGR_term51_Parser_next_underline_with_a_semicolon_is_two_attributes(t *testing.T) {
	p := SGRParser{params: []uint16{4, 3}}
	expectEqual(t, SGRAttributeUnderlineSingle, sgr_field(t, zigUnwrap(t, p.Next()), SGRAttributeTagUnderline).Underline)
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagItalic, "p.next().? == .italic")
	expect(t, !p.Next().Valid, "p.next() == null")
}

// Zig: test "term51: Parser.next: underline colon with nothing after it is unknown"
func TestSGR_term51_Parser_next_underline_colon_with_nothing_after_it_is_unknown(t *testing.T) {
	p := sgr_term51Parser([]uint16{4}, []uint{0})
	a := zigUnwrap(t, p.Next())
	expect(t, a.Tag == SGRAttributeTagUnknown, "a == .unknown")
	expectEqualSlices(t, []uint16{4}, a.Unknown.partial)
	expect(t, !p.Next().Valid, "p.next() == null")
}

// Zig: test "term51: Parser.next: a colon after a parameter that takes none makes it unknown"
func TestSGR_term51_Parser_next_a_colon_after_a_parameter_that_takes_none_makes_it_unknown(t *testing.T) {
	// 1:2 then ;3
	p := sgr_term51Parser([]uint16{1, 2, 3}, []uint{0})
	a := zigUnwrap(t, p.Next())
	expect(t, a.Tag == SGRAttributeTagUnknown, "a == .unknown")
	expectEqualSlices(t, []uint16{1, 2, 3}, a.Unknown.full)
	expectEqualSlices(t, []uint16{1, 2}, a.Unknown.partial)
	// the colon group is skipped as a whole
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagItalic, "p.next().? == .italic")
	expect(t, !p.Next().Valid, "p.next() == null")
}

// Zig: test "term51: Parser.next: colon group on an unsupported parameter is skipped to its end"
func TestSGR_term51_Parser_next_colon_group_on_an_unsupported_parameter_is_skipped_to_its_end(t *testing.T) {
	// 1:2:3:4 then ;9
	p := sgr_term51Parser([]uint16{1, 2, 3, 4, 9}, []uint{0, 1, 2})
	a := zigUnwrap(t, p.Next())
	expect(t, a.Tag == SGRAttributeTagUnknown, "a == .unknown")
	expectEqualSlices(t, []uint16{1, 2, 3, 4}, a.Unknown.partial)
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagStrikethrough, "p.next().? == .strikethrough")
	expect(t, !p.Next().Valid, "p.next() == null")
}

// Zig: test "term51: Parser.parseDirectColor: semicolon form takes the next three values"
func TestSGR_term51_Parser_parseDirectColor_semicolon_form_takes_the_next_three_values(t *testing.T) {
	p := SGRParser{params: []uint16{38, 2, 1, 2, 3, 4}}
	p.idx = 1 // as next() leaves it after reading 38
	a := zigUnwrap(t, p.parseDirectColor(SGRAttributeTagDirectColorFg, p.params, false))
	expectEqual(t, uint8(1), sgr_field(t, a, SGRAttributeTagDirectColorFg).DirectColorFg.r)
	expectEqual(t, uint8(2), sgr_field(t, a, SGRAttributeTagDirectColorFg).DirectColorFg.g)
	expectEqual(t, uint8(3), sgr_field(t, a, SGRAttributeTagDirectColorFg).DirectColorFg.b)
	expectEqual(t, uint(5), p.idx)
}

// Zig: test "term51: Parser.parseDirectColor: each target tag"
func TestSGR_term51_Parser_parseDirectColor_each_target_tag(t *testing.T) {
	{
		p := SGRParser{params: []uint16{48, 2, 10, 20, 30}}
		p.idx = 1
		a := zigUnwrap(t, p.parseDirectColor(SGRAttributeTagDirectColorBg, p.params, false))
		expect(t, a.Tag == SGRAttributeTagDirectColorBg, "a == .direct_color_bg")
		expectEqual(t, uint8(30), a.DirectColorBg.b)
	}
	{
		p := SGRParser{params: []uint16{58, 2, 10, 20, 30}}
		p.idx = 1
		a := zigUnwrap(t, p.parseDirectColor(SGRAttributeTagUnderlineColor, p.params, false))
		expect(t, a.Tag == SGRAttributeTagUnderlineColor, "a == .underline_color")
		expectEqual(t, uint8(10), a.UnderlineColor.r)
	}
}

// Zig: test "term51: Parser.parseDirectColor: fewer than five values is not a colour"
func TestSGR_term51_Parser_parseDirectColor_fewer_than_five_values_is_not_a_colour(t *testing.T) {
	for _, params := range [][]uint16{{38, 2}, {38, 2, 1}, {38, 2, 1, 2}} {
		p := SGRParser{params: params}
		p.idx = 1
		expect(t, !p.parseDirectColor(SGRAttributeTagDirectColorFg, params, false).Valid, "p.parseDirectColor(.direct_color_fg, params, false) == null")
		// nothing consumed
		expectEqual(t, uint(1), p.idx)
	}
}

// Zig: test "term51: Parser.parseDirectColor: components are truncated to 8 bits"
func TestSGR_term51_Parser_parseDirectColor_components_are_truncated_to_8_bits(t *testing.T) {
	p := SGRParser{params: []uint16{38, 2, 256, 257, 65535}}
	p.idx = 1
	a := zigUnwrap(t, p.parseDirectColor(SGRAttributeTagDirectColorFg, p.params, false))
	expectEqual(t, uint8(0), sgr_field(t, a, SGRAttributeTagDirectColorFg).DirectColorFg.r)
	expectEqual(t, uint8(1), sgr_field(t, a, SGRAttributeTagDirectColorFg).DirectColorFg.g)
	expectEqual(t, uint8(255), sgr_field(t, a, SGRAttributeTagDirectColorFg).DirectColorFg.b)
}

// Zig: test "term51: Parser.parseDirectColor: colon form without a colour space"
func TestSGR_term51_Parser_parseDirectColor_colon_form_without_a_colour_space(t *testing.T) {
	// 38:2:1:2:3
	p := sgr_term51Parser([]uint16{38, 2, 1, 2, 3}, []uint{0, 1, 2, 3})
	p.idx = 1
	a := zigUnwrap(t, p.parseDirectColor(SGRAttributeTagDirectColorFg, p.params, true))
	expectEqual(t, uint8(1), sgr_field(t, a, SGRAttributeTagDirectColorFg).DirectColorFg.r)
	expectEqual(t, uint8(2), sgr_field(t, a, SGRAttributeTagDirectColorFg).DirectColorFg.g)
	expectEqual(t, uint8(3), sgr_field(t, a, SGRAttributeTagDirectColorFg).DirectColorFg.b)
	expectEqual(t, uint(5), p.idx)
}

// Zig: test "term51: Parser.parseDirectColor: colon form with a colour space skips it"
func TestSGR_term51_Parser_parseDirectColor_colon_form_with_a_colour_space_skips_it(t *testing.T) {
	// 38:2:99:1:2:3  (99 is the ignored colour space id)
	p := sgr_term51Parser([]uint16{38, 2, 99, 1, 2, 3}, []uint{0, 1, 2, 3, 4})
	p.idx = 1
	a := zigUnwrap(t, p.parseDirectColor(SGRAttributeTagDirectColorFg, p.params, true))
	expectEqual(t, uint8(1), sgr_field(t, a, SGRAttributeTagDirectColorFg).DirectColorFg.r)
	expectEqual(t, uint8(2), sgr_field(t, a, SGRAttributeTagDirectColorFg).DirectColorFg.g)
	expectEqual(t, uint8(3), sgr_field(t, a, SGRAttributeTagDirectColorFg).DirectColorFg.b)
	expectEqual(t, uint(6), p.idx)
}

// Zig: test "term51: Parser.parseDirectColor: too many colon parts is not a colour and the group is consumed"
func TestSGR_term51_Parser_parseDirectColor_too_many_colon_parts_is_not_a_colour_and_the_group_is_consumed(t *testing.T) {
	// 38:2:0:1:2:3:4 then ;1
	p := sgr_term51Parser([]uint16{38, 2, 0, 1, 2, 3, 4, 1}, []uint{0, 1, 2, 3, 4, 5})
	p.idx = 1
	expect(t, !p.parseDirectColor(SGRAttributeTagDirectColorFg, p.params, true).Valid, "p.parseDirectColor(.direct_color_fg, p.params, true) == null")
	// idx now points at the parameter after the colon group
	expectEqual(t, uint(7), p.idx)
}

// Zig: test "term51: Parser.parseDirectColor: through next, semicolon colour followed by another attribute"
func TestSGR_term51_Parser_parseDirectColor_through_next_semicolon_colour_followed_by_another_attribute(t *testing.T) {
	p := SGRParser{params: []uint16{38, 2, 1, 2, 3, 1}}
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagDirectColorFg, "p.next().? == .direct_color_fg")
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagBold, "p.next().? == .bold")
	expect(t, !p.Next().Valid, "p.next() == null")
}

// Zig: test "term51: Parser.isColon: reports the separator after the current index"
func TestSGR_term51_Parser_isColon_reports_the_separator_after_the_current_index(t *testing.T) {
	p := sgr_term51Parser([]uint16{4, 3, 1}, []uint{0})
	p.idx = 0
	expect(t, p.isColon(), "p.isColon()")
	p.idx = 1
	expect(t, !p.isColon(), "!p.isColon()")
	p.idx = 2
	expect(t, !p.isColon(), "!p.isColon()")
}

// Zig: test "term51: Parser.isColon: false with no colons at all"
func TestSGR_term51_Parser_isColon_false_with_no_colons_at_all(t *testing.T) {
	p := SGRParser{params: []uint16{1, 2, 3}}
	for i := uint(0); i < 3; i++ {
		p.idx = i
		expect(t, !p.isColon(), "!p.isColon()")
	}
}

// Zig: test "term51: Parser.countColon: counts consecutive colons from the current index"
func TestSGR_term51_Parser_countColon_counts_consecutive_colons_from_the_current_index(t *testing.T) {
	p := sgr_term51Parser([]uint16{38, 2, 1, 2, 3, 9}, []uint{0, 1, 2, 3})
	p.idx = 0
	expectEqual(t, uint(4), p.countColon())
	p.idx = 1
	expectEqual(t, uint(3), p.countColon())
	p.idx = 3
	expectEqual(t, uint(1), p.countColon())
	p.idx = 4
	expectEqual(t, uint(0), p.countColon())
}

// Zig: test "term51: Parser.countColon: stops at the first semicolon"
func TestSGR_term51_Parser_countColon_stops_at_the_first_semicolon(t *testing.T) {
	// 1:2;3:4
	p := sgr_term51Parser([]uint16{1, 2, 3, 4}, []uint{0, 2})
	p.idx = 0
	expectEqual(t, uint(1), p.countColon())
	p.idx = 1
	expectEqual(t, uint(0), p.countColon())
	p.idx = 2
	expectEqual(t, uint(1), p.countColon())
}

// Zig: test "term51: Parser.countColon: never counts a separator after the last parameter"
func TestSGR_term51_Parser_countColon_never_counts_a_separator_after_the_last_parameter(t *testing.T) {
	// a colon bit set on the last parameter is ignored
	p := sgr_term51Parser([]uint16{4, 3}, []uint{0, 1})
	p.idx = 0
	expectEqual(t, uint(1), p.countColon())
	p.idx = 1
	expectEqual(t, uint(0), p.countColon())
}

// Zig: test "term51: Parser.consumeUnknownColon: skips the colon group and one more"
func TestSGR_term51_Parser_consumeUnknownColon_skips_the_colon_group_and_one_more(t *testing.T) {
	// 4:3:1 then ;7
	p := sgr_term51Parser([]uint16{4, 3, 1, 7}, []uint{0, 1})
	p.idx = 1
	p.consumeUnknownColon()
	expectEqual(t, uint(3), p.idx)
	expect(t, zigUnwrap(t, p.Next()).Tag == SGRAttributeTagInverse, "p.next().? == .inverse")
}

// Zig: test "term51: Parser.consumeUnknownColon: with no colon advances by one"
func TestSGR_term51_Parser_consumeUnknownColon_with_no_colon_advances_by_one(t *testing.T) {
	p := SGRParser{params: []uint16{1, 2, 3}}
	p.idx = 1
	p.consumeUnknownColon()
	expectEqual(t, uint(2), p.idx)
}
