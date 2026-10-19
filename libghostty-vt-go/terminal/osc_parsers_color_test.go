// Tests translated one for one from src/terminal/osc/parsers/color.zig
// (Ghostty v1.3.1 plus the term51 additional tests). Each Go test carries the
// exact Zig title in a "// Zig: test" comment. The helpers (expect,
// expectEqual, ...) are in Parser_test.go and zigTry is in dcs_test.go;
// oscA_term51Feed, oscA_unwrapPtr, oscA_ptrAt and oscA_terms are in
// osc_parsers_change_window_icon_test.go, where
// the translation notes are.
//
// Not translated (std.testing.FailingAllocator has no Go counterpart, see
// scripts/not-translated.txt):
//   - test "term51: parse: allocation failure yields an empty request list"
//   - test "term51: parseGetSetAnsiColor: allocation failure frees the partial list"
//   - test "term51: parseResetAnsiColor: allocation failure frees the partial list"
//   - test "term51: parseGetSetDynamicColor: allocation failure leaks the partial list"
//
// Translation notes:
//   - `try parseColor(alloc, op, body)` is oscA_parseColor(t, op, body), which
//     is oscParsersColorParseColor followed by zigTry; the same for the other
//     parse functions (oscA_parseGetSetAnsiColor and so on). `defer
//     list.deinit(alloc)` is `defer list.deinit()` and `list.at(0).*` is
//     `*list.at(0)`.
//   - Request, Target and ColoredTarget literals such as
//     `Request{ .set = .{ .target = .{ .palette = 1 }, .color = c } }` are
//     written with the helpers oscA_colSet, oscA_colQuery, oscA_colReset (a
//     Request) and oscA_colPalette, oscA_colSpecial, oscA_colDynamic (a
//     Target); `.reset_palette = {}` is the literal
//     OSCParsersColorRequest{Tag: OSCParsersColorRequestResetPalette}.
//   - `std.mem.tokenizeScalar(u8, buf, ';')` is oscA_term51Tokens(buf), a
//     literal of the wireframe's oscParsersColorTokenIterator.
//   - `inline for (@typeInfo(DynamicColor).fields)` is a loop over
//     oscA_colDynamics: each dynamic color with its OSC 10-19 and OSC 110-119
//     operations (Zig builds the operation name from the field value).
//   - `0..std.math.maxInt(u8)` is `for idx := 0; idx < 255; idx++` (the range
//     excludes 255).

package terminal

import (
	"fmt"
	"strings"
	"testing"
)

var oscA_colRed = ColorRGB{R: 255, G: 0, B: 0}
var oscA_colBlue = ColorRGB{R: 0, G: 0, B: 255}

// oscA_colSpecials is every value of color.Special in order (Zig:
// `0..@typeInfo(SpecialColor).@"enum".fields.len` and intToEnum).
var oscA_colSpecials = []ColorSpecial{
	ColorSpecialBold,
	ColorSpecialUnderline,
	ColorSpecialBlink,
	ColorSpecialReverse,
	ColorSpecialItalic,
}

// oscA_colDynamics is every value of color.Dynamic in field order with the
// Operation that gets/sets it (osc_10..osc_19) and the one that resets it
// (osc_110..osc_119).
var oscA_colDynamics = []struct {
	color ColorDynamic
	get   OSCParsersColorOperation
	reset OSCParsersColorOperation
}{
	{ColorDynamicForeground, OSCParsersColorOperationOsc10, OSCParsersColorOperationOsc110},
	{ColorDynamicBackground, OSCParsersColorOperationOsc11, OSCParsersColorOperationOsc111},
	{ColorDynamicCursor, OSCParsersColorOperationOsc12, OSCParsersColorOperationOsc112},
	{ColorDynamicPointerForeground, OSCParsersColorOperationOsc13, OSCParsersColorOperationOsc113},
	{ColorDynamicPointerBackground, OSCParsersColorOperationOsc14, OSCParsersColorOperationOsc114},
	{ColorDynamicTektronixForeground, OSCParsersColorOperationOsc15, OSCParsersColorOperationOsc115},
	{ColorDynamicTektronixBackground, OSCParsersColorOperationOsc16, OSCParsersColorOperationOsc116},
	{ColorDynamicHighlightBackground, OSCParsersColorOperationOsc17, OSCParsersColorOperationOsc117},
	{ColorDynamicTektronixCursor, OSCParsersColorOperationOsc18, OSCParsersColorOperationOsc118},
	{ColorDynamicHighlightForeground, OSCParsersColorOperationOsc19, OSCParsersColorOperationOsc119},
}

// Request literals.
func oscA_colSet(target OSCParsersColorTarget, color ColorRGB) OSCParsersColorRequest {
	return OSCParsersColorRequest{Tag: OSCParsersColorRequestSet, Set: OSCParsersColorColoredTarget{Target: target, Color: color}}
}

func oscA_colQuery(target OSCParsersColorTarget) OSCParsersColorRequest {
	return OSCParsersColorRequest{Tag: OSCParsersColorRequestQuery, Query: target}
}

func oscA_colReset(target OSCParsersColorTarget) OSCParsersColorRequest {
	return OSCParsersColorRequest{Tag: OSCParsersColorRequestReset, Reset: target}
}

// Target literals.
func oscA_colPalette(idx uint8) OSCParsersColorTarget {
	return OSCParsersColorTarget{Tag: OSCParsersColorTargetPalette, Palette: idx}
}

func oscA_colSpecial(s ColorSpecial) OSCParsersColorTarget {
	return OSCParsersColorTarget{Tag: OSCParsersColorTargetSpecial, Special: s}
}

func oscA_colDynamic(d ColorDynamic) OSCParsersColorTarget {
	return OSCParsersColorTarget{Tag: OSCParsersColorTargetDynamic, Dynamic: d}
}

// oscA_term51Tokens is the term51 helper term51Tokens.
func oscA_term51Tokens(buf string) oscParsersColorTokenIterator {
	return oscParsersColorTokenIterator{buffer: []uint8(buf), delimiter: ';'}
}

// oscA_parseColor is `try parseColor(alloc, op, buf)`.
func oscA_parseColor(t *testing.T, op OSCParsersColorOperation, buf string) OSCParsersColorList {
	t.Helper()
	list, err := oscParsersColorParseColor(op, []uint8(buf))
	zigTry(t, err)
	return list
}

// oscA_parseGetSetAnsiColor is `try parseGetSetAnsiColor(alloc, op, &it)`.
func oscA_parseGetSetAnsiColor(t *testing.T, op OSCParsersColorOperation, it *oscParsersColorTokenIterator) OSCParsersColorList {
	t.Helper()
	list, err := oscParsersColorParseGetSetAnsiColor(op, it)
	zigTry(t, err)
	return list
}

// oscA_parseResetAnsiColor is `try parseResetAnsiColor(alloc, op, &it)`.
func oscA_parseResetAnsiColor(t *testing.T, op OSCParsersColorOperation, it *oscParsersColorTokenIterator) OSCParsersColorList {
	t.Helper()
	list, err := oscParsersColorParseResetAnsiColor(op, it)
	zigTry(t, err)
	return list
}

// oscA_parseGetSetDynamicColor is `try parseGetSetDynamicColor(alloc, start, &it)`.
func oscA_parseGetSetDynamicColor(t *testing.T, start ColorDynamic, it *oscParsersColorTokenIterator) OSCParsersColorList {
	t.Helper()
	list, err := oscParsersColorParseGetSetDynamicColor(start, it)
	zigTry(t, err)
	return list
}

// oscA_parseResetDynamicColor is `try parseResetDynamicColor(alloc, color, &it)`.
func oscA_parseResetDynamicColor(t *testing.T, color ColorDynamic, it *oscParsersColorTokenIterator) OSCParsersColorList {
	t.Helper()
	list, err := oscParsersColorParseResetDynamicColor(color, it)
	zigTry(t, err)
	return list
}

// Zig: test "OSC 4: empty param"
func TestOSCParsersColor_OSC_4_empty_param(t *testing.T) {
	p := OSCParserInit(false)

	input := "4;;"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := p.End(Some[uint8](0x1b))
	expect(t, cmd == nil, "cmd == null")
}

// Zig: test "OSC 4:"
func TestOSCParsersColor_OSC_4(t *testing.T) {
	// Test every palette index
	for idx := 0; idx < 255; idx++ {
		// Simple color set
		// printf '\e]4;0;red\\'
		{
			body := fmt.Sprintf("%d;red", idx)

			list := oscA_parseColor(t, OSCParsersColorOperationOsc4, body)
			defer list.deinit()
			expectEqual(t, 1, list.Count())
			expectEqual(t, oscA_colSet(oscA_colPalette(uint8(idx)), oscA_colRed), *list.At(0))
		}

		// Simple color query
		// printf '\e]4;0;?\\'
		{
			body := fmt.Sprintf("%d;?", idx)

			list := oscA_parseColor(t, OSCParsersColorOperationOsc4, body)
			defer list.deinit()
			expectEqual(t, 1, list.Count())
			expectEqual(t, oscA_colQuery(oscA_colPalette(uint8(idx))), *list.At(0))
		}

		// Trailing invalid data produces results up to that point
		// printf '\e]4;0;red;\e\\'
		{
			body := fmt.Sprintf("%d;red;", idx)

			list := oscA_parseColor(t, OSCParsersColorOperationOsc4, body)
			defer list.deinit()
			expectEqual(t, 1, list.Count())
			expectEqual(t, oscA_colSet(oscA_colPalette(uint8(idx)), oscA_colRed), *list.At(0))
		}

		// Whitespace doesn't produce a working value in xterm but we
		// allow it because Kitty does and it seems harmless.
		//
		// printf '\e]4;0;red \e\\'
		{
			body := fmt.Sprintf("%d;red ", idx)

			list := oscA_parseColor(t, OSCParsersColorOperationOsc4, body)
			defer list.deinit()
			expectEqual(t, 1, list.Count())
			expectEqual(t, oscA_colSet(oscA_colPalette(uint8(idx)), oscA_colRed), *list.At(0))
		}
	}

	// Test every special color
	for i := 0; i < len(oscA_colSpecials); i++ {
		special := oscA_colSpecials[i]

		// Simple color set
		// printf '\e]4;256;red\\'
		{
			body := fmt.Sprintf("%d;red", 256+i)

			list := oscA_parseColor(t, OSCParsersColorOperationOsc4, body)
			defer list.deinit()
			expectEqual(t, 1, list.Count())
			expectEqual(t, oscA_colSet(oscA_colSpecial(special), oscA_colRed), *list.At(0))
		}
	}
}

// Zig: test "OSC 5:"
func TestOSCParsersColor_OSC_5(t *testing.T) {
	// Test every special color
	for i := 0; i < len(oscA_colSpecials); i++ {
		special := oscA_colSpecials[i]

		// Simple color set
		// printf '\e]4;256;red\\'
		{
			body := fmt.Sprintf("%d;red", i)

			list := oscA_parseColor(t, OSCParsersColorOperationOsc5, body)
			defer list.deinit()
			expectEqual(t, 1, list.Count())
			expectEqual(t, oscA_colSet(oscA_colSpecial(special), oscA_colRed), *list.At(0))
		}
	}
}

// Zig: test "OSC 4: multiple requests"
func TestOSCParsersColor_OSC_4_multiple_requests(t *testing.T) {
	// printf '\e]4;0;red;1;blue\e\\'
	{
		list := oscA_parseColor(t, OSCParsersColorOperationOsc4, "0;red;1;blue")
		defer list.deinit()
		expectEqual(t, 2, list.Count())
		expectEqual(t, oscA_colSet(oscA_colPalette(0), ColorRGB{R: 255, G: 0, B: 0}), *list.At(0))
		expectEqual(t, oscA_colSet(oscA_colPalette(1), ColorRGB{R: 0, G: 0, B: 255}), *list.At(1))
	}

	// Multiple requests with same index overwrite each other
	// printf '\e]4;0;red;0;blue\e\\'
	{
		list := oscA_parseColor(t, OSCParsersColorOperationOsc4, "0;red;0;blue")
		defer list.deinit()
		expectEqual(t, 2, list.Count())
		expectEqual(t, oscA_colSet(oscA_colPalette(0), ColorRGB{R: 255, G: 0, B: 0}), *list.At(0))
		expectEqual(t, oscA_colSet(oscA_colPalette(0), ColorRGB{R: 0, G: 0, B: 255}), *list.At(1))
	}
}

// Zig: test "OSC 104:"
func TestOSCParsersColor_OSC_104(t *testing.T) {
	// Test every palette index
	for idx := 0; idx < 255; idx++ {
		// Simple color set
		// printf '\e]104;0\\'
		{
			body := fmt.Sprintf("%d", idx)

			list := oscA_parseColor(t, OSCParsersColorOperationOsc104, body)
			defer list.deinit()
			expectEqual(t, 1, list.Count())
			expectEqual(t, oscA_colReset(oscA_colPalette(uint8(idx))), *list.At(0))
		}
	}

	// Test every special color
	for i := 0; i < len(oscA_colSpecials); i++ {
		special := oscA_colSpecials[i]

		// Simple color set
		// printf '\e]104;256\\'
		{
			body := fmt.Sprintf("%d", 256+i)

			list := oscA_parseColor(t, OSCParsersColorOperationOsc104, body)
			defer list.deinit()
			expectEqual(t, 1, list.Count())
			expectEqual(t, oscA_colReset(oscA_colSpecial(special)), *list.At(0))
		}
	}
}

// Zig: test "OSC 104: empty index"
func TestOSCParsersColor_OSC_104_empty_index(t *testing.T) {
	list := oscA_parseColor(t, OSCParsersColorOperationOsc104, "0;;1")
	defer list.deinit()
	expectEqual(t, 2, list.Count())
	expectEqual(t, oscA_colReset(oscA_colPalette(0)), *list.At(0))
	expectEqual(t, oscA_colReset(oscA_colPalette(1)), *list.At(1))
}

// Zig: test "OSC 104: invalid index"
func TestOSCParsersColor_OSC_104_invalid_index(t *testing.T) {
	list := oscA_parseColor(t, OSCParsersColorOperationOsc104, "ffff;1")
	defer list.deinit()
	expectEqual(t, 1, list.Count())
	expectEqual(t, oscA_colReset(oscA_colPalette(1)), *list.At(0))
}

// Zig: test "OSC 104: reset all"
func TestOSCParsersColor_OSC_104_reset_all(t *testing.T) {
	list := oscA_parseColor(t, OSCParsersColorOperationOsc104, "")
	defer list.deinit()
	expectEqual(t, 1, list.Count())
	expectEqual(t, OSCParsersColorRequest{Tag: OSCParsersColorRequestResetPalette}, *list.At(0))
}

// Zig: test "OSC 105: reset all"
func TestOSCParsersColor_OSC_105_reset_all(t *testing.T) {
	list := oscA_parseColor(t, OSCParsersColorOperationOsc105, "")
	defer list.deinit()
	expectEqual(t, 1, list.Count())
	expectEqual(t, OSCParsersColorRequest{Tag: OSCParsersColorRequestResetSpecial}, *list.At(0))
}

// Zig: test "OSC 10: OSC 11: OSC 12: OSC: 13: OSC 14: OSC 15: OSC: 16: OSC 17: OSC 18: OSC 19: dynamic"
func TestOSCParsersColor_OSC_10_OSC_11_OSC_12_OSC_13_OSC_14_OSC_15_OSC_16_OSC_17_OSC_18_OSC_19_dynamic(t *testing.T) {
	// OSC 10-19: Get/Set Dynamic Colors
	for _, d := range oscA_colDynamics {
		color := d.color
		op := d.get

		// Example script:
		// printf '\e]10;red\e\\'
		{
			list := oscA_parseColor(t, op, "red")
			defer list.deinit()
			expectEqual(t, 1, list.Count())
			expectEqual(t, oscA_colSet(oscA_colDynamic(color), ColorRGB{R: 255, G: 0, B: 0}), *list.At(0))
		}
	}
}

// Zig: test "OSC 10: OSC 11: OSC 12: OSC: 13: OSC 14: OSC 15: OSC: 16: OSC 17: OSC 18: OSC 19: dynamic multiple"
func TestOSCParsersColor_OSC_10_OSC_11_OSC_12_OSC_13_OSC_14_OSC_15_OSC_16_OSC_17_OSC_18_OSC_19_dynamic_multiple(t *testing.T) {
	// Example script:
	// printf '\e]11;red;blue\e\\'
	{
		list := oscA_parseColor(t, OSCParsersColorOperationOsc11, "red;blue")
		defer list.deinit()
		expectEqual(t, 2, list.Count())
		expectEqual(t, oscA_colSet(oscA_colDynamic(ColorDynamicBackground), ColorRGB{R: 255, G: 0, B: 0}), *list.At(0))
		expectEqual(t, oscA_colSet(oscA_colDynamic(ColorDynamicCursor), ColorRGB{R: 0, G: 0, B: 255}), *list.At(1))
	}
}

// Zig: test "OSC 110: OSC 111: OSC 112: OSC: 113: OSC 114: OSC 115: OSC: 116: OSC 117: OSC 118: OSC 119: reset dynamic"
func TestOSCParsersColor_OSC_110_OSC_111_OSC_112_OSC_113_OSC_114_OSC_115_OSC_116_OSC_117_OSC_118_OSC_119_reset_dynamic(t *testing.T) {
	// OSC 110-119: Reset Dynamic Colors
	for _, d := range oscA_colDynamics {
		color := d.color
		op := d.reset

		// Example script:
		// printf '\e]110\e\\'
		{
			list := oscA_parseColor(t, op, "")
			defer list.deinit()
			expectEqual(t, 1, list.Count())
			expectEqual(t, oscA_colReset(oscA_colDynamic(color)), *list.At(0))
		}

		// xterm allows a trailing semicolon. script to verify:
		//
		// printf '\e]110;\e\\'
		{
			list := oscA_parseColor(t, op, ";")
			defer list.deinit()
			expectEqual(t, 1, list.Count())
			expectEqual(t, oscA_colReset(oscA_colDynamic(color)), *list.At(0))
		}

		// xterm does NOT allow any whitespace
		//
		// printf '\e]110 \e\\'
		{
			list := oscA_parseColor(t, op, " ")
			defer list.deinit()
			expectEqual(t, 0, list.Count())
		}
	}
}

// Zig: test "term51: parse: no allocator is invalid"
func TestOSCParsersColor_term51_parse_no_allocator_is_invalid(t *testing.T) {
	p := OSCParserInit(false)
	oscA_term51Feed(&p, "4")
	expect(t, p.state == OSCParserState4, "p.state == .@\"4\"")
	expect(t, OSCParsersColorParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
	expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	expect(t, p.command.Tag == OSCCommandKeyInvalid, "p.command == .invalid")
}

// Zig: test "term51: parse: a state that is not a color OSC is invalid"
func TestOSCParsersColor_term51_parse_a_state_that_is_not_a_color_OSC_is_invalid(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	oscA_term51Feed(&p, "0;red")
	expect(t, p.state == OSCParserState0, "p.state == .@\"0\"")
	expect(t, OSCParsersColorParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
	expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	expect(t, p.command.Tag == OSCCommandKeyInvalid, "p.command == .invalid")
}

// Zig: test "term51: parse: operation for every state, without a separator"
func TestOSCParsersColor_term51_parse_operation_for_every_state_without_a_separator(t *testing.T) {
	// Without a ';' no data was collected and the body is the empty string:
	// get/set operations yield no requests and reset operations reset
	// everything they cover.
	none := None[OSCParsersColorRequest]()
	for _, c := range []struct {
		input string
		op    OSCParsersColorOperation
		req   Optional[OSCParsersColorRequest]
	}{
		{"4", OSCParsersColorOperationOsc4, none},
		{"5", OSCParsersColorOperationOsc5, none},
		{"10", OSCParsersColorOperationOsc10, none},
		{"11", OSCParsersColorOperationOsc11, none},
		{"12", OSCParsersColorOperationOsc12, none},
		{"13", OSCParsersColorOperationOsc13, none},
		{"14", OSCParsersColorOperationOsc14, none},
		{"15", OSCParsersColorOperationOsc15, none},
		{"16", OSCParsersColorOperationOsc16, none},
		{"17", OSCParsersColorOperationOsc17, none},
		{"18", OSCParsersColorOperationOsc18, none},
		{"19", OSCParsersColorOperationOsc19, none},
		{"104", OSCParsersColorOperationOsc104, Some(OSCParsersColorRequest{Tag: OSCParsersColorRequestResetPalette})},
		{"110", OSCParsersColorOperationOsc110, Some(oscA_colReset(oscA_colDynamic(ColorDynamicForeground)))},
		{"111", OSCParsersColorOperationOsc111, Some(oscA_colReset(oscA_colDynamic(ColorDynamicBackground)))},
		{"112", OSCParsersColorOperationOsc112, Some(oscA_colReset(oscA_colDynamic(ColorDynamicCursor)))},
		{"113", OSCParsersColorOperationOsc113, Some(oscA_colReset(oscA_colDynamic(ColorDynamicPointerForeground)))},
		{"114", OSCParsersColorOperationOsc114, Some(oscA_colReset(oscA_colDynamic(ColorDynamicPointerBackground)))},
		{"115", OSCParsersColorOperationOsc115, Some(oscA_colReset(oscA_colDynamic(ColorDynamicTektronixForeground)))},
		{"116", OSCParsersColorOperationOsc116, Some(oscA_colReset(oscA_colDynamic(ColorDynamicTektronixBackground)))},
		{"117", OSCParsersColorOperationOsc117, Some(oscA_colReset(oscA_colDynamic(ColorDynamicHighlightBackground)))},
		{"118", OSCParsersColorOperationOsc118, Some(oscA_colReset(oscA_colDynamic(ColorDynamicTektronixCursor)))},
		{"119", OSCParsersColorOperationOsc119, Some(oscA_colReset(oscA_colDynamic(ColorDynamicHighlightForeground)))},
	} {
		p := OSCParserInit(true)
		defer p.Deinit()
		oscA_term51Feed(&p, c.input)
		expect(t, p.writer == nil, "p.writer == null")
		cmd := oscA_unwrapPtr(t, OSCParsersColorParse(&p, None[uint8]()))
		defer cmd.ColorOperation.Requests.deinit()
		expect(t, cmd.Tag == OSCCommandKeyColorOperation, "cmd.* == .color_operation")
		expectEqual(t, c.op, cmd.ColorOperation.Op)
		if !c.req.Valid {
			expectEqual(t, 0, cmd.ColorOperation.Requests.Count())
		} else {
			expectEqual(t, 1, cmd.ColorOperation.Requests.Count())
			expectEqual(t, c.req.Value, *cmd.ColorOperation.Requests.At(0))
		}
	}
}

// Zig: test "term51: parse: a separator with no body behaves like no separator"
func TestOSCParsersColor_term51_parse_a_separator_with_no_body_behaves_like_no_separator(t *testing.T) {
	{
		p := OSCParserInit(true)
		defer p.Deinit()
		oscA_term51Feed(&p, "104;")
		cmd := oscA_unwrapPtr(t, OSCParsersColorParse(&p, None[uint8]()))
		defer cmd.ColorOperation.Requests.deinit()
		expectEqual(t, OSCParsersColorOperationOsc104, cmd.ColorOperation.Op)
		expectEqual(t, 1, cmd.ColorOperation.Requests.Count())
		expectEqual(t, OSCParsersColorRequest{Tag: OSCParsersColorRequestResetPalette}, *cmd.ColorOperation.Requests.At(0))
	}
	{
		p := OSCParserInit(true)
		defer p.Deinit()
		oscA_term51Feed(&p, "4;")
		cmd := oscA_unwrapPtr(t, OSCParsersColorParse(&p, None[uint8]()))
		defer cmd.ColorOperation.Requests.deinit()
		expectEqual(t, OSCParsersColorOperationOsc4, cmd.ColorOperation.Op)
		expectEqual(t, 0, cmd.ColorOperation.Requests.Count())
	}
}

// Zig: test "term51: parse: terminator is recorded"
func TestOSCParsersColor_term51_parse_terminator_is_recorded(t *testing.T) {
	for _, c := range []struct {
		term Optional[uint8]
		want OSCTerminator
	}{
		{Some[uint8](0x07), OSCTerminatorBel},
		{Some[uint8](0x1b), OSCTerminatorSt},
		{None[uint8](), OSCTerminatorSt},
		// Anything that is not BEL is treated as ST.
		{Some[uint8]('\\'), OSCTerminatorSt},
		{Some[uint8](0x9c), OSCTerminatorSt},
		{Some[uint8](0), OSCTerminatorSt},
	} {
		p := OSCParserInit(true)
		defer p.Deinit()
		oscA_term51Feed(&p, "10;?")
		cmd := oscA_unwrapPtr(t, OSCParsersColorParse(&p, c.term))
		defer cmd.ColorOperation.Requests.deinit()
		expect(t, cmd.ColorOperation.Terminator == c.want, "cmd.color_operation.terminator == case[1]")
		expectEqual(t, OSCParsersColorOperationOsc10, cmd.ColorOperation.Op)
		expectEqual(t, 1, cmd.ColorOperation.Requests.Count())
		expectEqual(t, oscA_colQuery(oscA_colDynamic(ColorDynamicForeground)), *cmd.ColorOperation.Requests.At(0))
	}
}

// Zig: test "term51: parse: OSC 4 with several requests"
func TestOSCParsersColor_term51_parse_OSC_4_with_several_requests(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	oscA_term51Feed(&p, "4;1;red;2;?;256;#0000ff")
	cmd := oscA_unwrapPtr(t, OSCParsersColorParse(&p, Some[uint8](0x07)))
	defer cmd.ColorOperation.Requests.deinit()
	reqs := &cmd.ColorOperation.Requests
	expectEqual(t, OSCParsersColorOperationOsc4, cmd.ColorOperation.Op)
	expectEqual(t, 3, reqs.Count())
	expectEqual(t, oscA_colSet(oscA_colPalette(1), oscA_colRed), *reqs.At(0))
	expectEqual(t, oscA_colQuery(oscA_colPalette(2)), *reqs.At(1))
	expectEqual(t, oscA_colSet(oscA_colSpecial(ColorSpecialBold), oscA_colBlue), *reqs.At(2))
}

// Zig: test "term51: parse: OSC 5 and OSC 104 through the parser"
func TestOSCParsersColor_term51_parse_OSC_5_and_OSC_104_through_the_parser(t *testing.T) {
	{
		p := OSCParserInit(true)
		defer p.Deinit()
		oscA_term51Feed(&p, "5;4;?")
		cmd := oscA_unwrapPtr(t, OSCParsersColorParse(&p, None[uint8]()))
		defer cmd.ColorOperation.Requests.deinit()
		expectEqual(t, OSCParsersColorOperationOsc5, cmd.ColorOperation.Op)
		expectEqual(t, 1, cmd.ColorOperation.Requests.Count())
		expectEqual(t, oscA_colQuery(oscA_colSpecial(ColorSpecialItalic)), *cmd.ColorOperation.Requests.At(0))
	}
	{
		p := OSCParserInit(true)
		defer p.Deinit()
		oscA_term51Feed(&p, "104;7;260")
		cmd := oscA_unwrapPtr(t, OSCParsersColorParse(&p, None[uint8]()))
		defer cmd.ColorOperation.Requests.deinit()
		expectEqual(t, OSCParsersColorOperationOsc104, cmd.ColorOperation.Op)
		expectEqual(t, 2, cmd.ColorOperation.Requests.Count())
		expectEqual(t, oscA_colReset(oscA_colPalette(7)), *cmd.ColorOperation.Requests.At(0))
		expectEqual(t, oscA_colReset(oscA_colSpecial(ColorSpecialItalic)), *cmd.ColorOperation.Requests.At(1))
	}
}

// Zig: test "term51: parse: OSC 105 is not reachable through the parser"
func TestOSCParsersColor_term51_parse_OSC_105_is_not_reachable_through_the_parser(t *testing.T) {
	// Operation.osc_105 exists and parseColor handles it, but the state
	// machine has no "105" state: the '5' after "10" is rejected.
	p := OSCParserInit(true)
	defer p.Deinit()
	oscA_term51Feed(&p, "105")
	expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	expect(t, OSCParsersColorParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
	expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
}

// Zig: test "term51: parse: unparseable body still yields a command"
func TestOSCParsersColor_term51_parse_unparseable_body_still_yields_a_command(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	oscA_term51Feed(&p, "4;bogus;data")
	cmd := oscA_unwrapPtr(t, OSCParsersColorParse(&p, None[uint8]()))
	defer cmd.ColorOperation.Requests.deinit()
	expect(t, cmd.Tag == OSCCommandKeyColorOperation, "cmd.* == .color_operation")
	expectEqual(t, 0, cmd.ColorOperation.Requests.Count())
}

// Zig: test "term51: parse: buffer limit"
func TestOSCParsersColor_term51_parse_buffer_limit(t *testing.T) {
	// No NUL terminator is needed, so the body may fill the whole buffer.
	// An allocator is required but does not lift the limit.
	body := strings.Repeat("0;?;", OSCParserMAX_BUF/4)
	assert(len(body) == OSCParserMAX_BUF)
	{
		p := OSCParserInit(true)
		defer p.Deinit()
		oscA_term51Feed(&p, "4;"+body)
		expect(t, p.state == OSCParserState4, "p.state == .@\"4\"")
		cmd := oscA_unwrapPtr(t, OSCParsersColorParse(&p, None[uint8]()))
		defer cmd.ColorOperation.Requests.deinit()
		expectEqual(t, OSCParserMAX_BUF/4, cmd.ColorOperation.Requests.Count())
		expectEqual(t, oscA_colQuery(oscA_colPalette(0)), *cmd.ColorOperation.Requests.At(OSCParserMAX_BUF/4 - 1))
	}
	{
		p := OSCParserInit(true)
		defer p.Deinit()
		oscA_term51Feed(&p, "4;"+body+"1")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
		expect(t, OSCParsersColorParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
	}
}

// Zig: test "term51: parseGetSetAnsiColor: empty input"
func TestOSCParsersColor_term51_parseGetSetAnsiColor_empty_input(t *testing.T) {
	for _, body := range []string{"", ";", ";;;"} {
		{
			it := oscA_term51Tokens(body)
			list := oscA_parseGetSetAnsiColor(t, OSCParsersColorOperationOsc4, &it)
			defer list.deinit()
			expectEqual(t, 0, list.Count())
		}
		{
			it := oscA_term51Tokens(body)
			list := oscA_parseGetSetAnsiColor(t, OSCParsersColorOperationOsc5, &it)
			defer list.deinit()
			expectEqual(t, 0, list.Count())
		}
	}
}

// Zig: test "term51: parseGetSetAnsiColor: index without a spec"
func TestOSCParsersColor_term51_parseGetSetAnsiColor_index_without_a_spec(t *testing.T) {
	{
		it := oscA_term51Tokens("0")
		list := oscA_parseGetSetAnsiColor(t, OSCParsersColorOperationOsc4, &it)
		defer list.deinit()
		expectEqual(t, 0, list.Count())
	}
	{
		// A complete pair followed by a dangling index.
		it := oscA_term51Tokens("0;red;1")
		list := oscA_parseGetSetAnsiColor(t, OSCParsersColorOperationOsc4, &it)
		defer list.deinit()
		expectEqual(t, 1, list.Count())
		expectEqual(t, oscA_colSet(oscA_colPalette(0), oscA_colRed), *list.At(0))
	}
}

// Zig: test "term51: parseGetSetAnsiColor: OSC 4 index boundaries"
func TestOSCParsersColor_term51_parseGetSetAnsiColor_OSC_4_index_boundaries(t *testing.T) {
	// Valid indices: 0-255 are palette entries, 256-260 are special colors.
	for _, c := range []struct {
		body   string
		target OSCParsersColorTarget
	}{
		{"0;?", oscA_colPalette(0)},
		{"255;?", oscA_colPalette(255)},
		{"256;?", oscA_colSpecial(ColorSpecialBold)},
		{"257;?", oscA_colSpecial(ColorSpecialUnderline)},
		{"258;?", oscA_colSpecial(ColorSpecialBlink)},
		{"259;?", oscA_colSpecial(ColorSpecialReverse)},
		{"260;?", oscA_colSpecial(ColorSpecialItalic)},
	} {
		it := oscA_term51Tokens(c.body)
		list := oscA_parseGetSetAnsiColor(t, OSCParsersColorOperationOsc4, &it)
		defer list.deinit()
		expectEqual(t, 1, list.Count())
		expectEqual(t, oscA_colQuery(c.target), *list.At(0))
	}

	// Invalid indices produce nothing.
	for _, body := range []string{
		// 261-263 fit in the u3 offset but are not special colors
		"261;?",
		"263;?",
		// 264-511 fit in the u9 but not in the u3 offset
		"264;?",
		"511;?",
		// 512 and up do not fit in a u9
		"512;?",
		"65536;?",
		"-1;?",
		"abc;?",
		" 1;?",
		"1 ;?",
		"0x1;?",
		"1.0;?",
	} {
		it := oscA_term51Tokens(body)
		list := oscA_parseGetSetAnsiColor(t, OSCParsersColorOperationOsc4, &it)
		defer list.deinit()
		expectEqual(t, 0, list.Count())
	}
}

// Zig: test "term51: parseGetSetAnsiColor: OSC 5 index boundaries"
func TestOSCParsersColor_term51_parseGetSetAnsiColor_OSC_5_index_boundaries(t *testing.T) {
	for _, c := range []struct {
		body    string
		special ColorSpecial
	}{
		{"0;?", ColorSpecialBold},
		{"1;?", ColorSpecialUnderline},
		{"2;?", ColorSpecialBlink},
		{"3;?", ColorSpecialReverse},
		{"4;?", ColorSpecialItalic},
	} {
		it := oscA_term51Tokens(c.body)
		list := oscA_parseGetSetAnsiColor(t, OSCParsersColorOperationOsc5, &it)
		defer list.deinit()
		expectEqual(t, 1, list.Count())
		expectEqual(t, oscA_colQuery(oscA_colSpecial(c.special)), *list.At(0))
	}

	// 5-7 fit in a u3 but are not special colors, 8 and up do not fit.
	// The OSC 4 offsets (256-260) are not accepted by OSC 5.
	for _, body := range []string{"5;?", "7;?", "8;?", "255;?", "256;?", "260;?", "511;?", "512;?", "-1;?", "bold;?"} {
		it := oscA_term51Tokens(body)
		list := oscA_parseGetSetAnsiColor(t, OSCParsersColorOperationOsc5, &it)
		defer list.deinit()
		expectEqual(t, 0, list.Count())
	}
}

// Zig: test "term51: parseGetSetAnsiColor: index number formats accepted by std.fmt.parseInt"
func TestOSCParsersColor_term51_parseGetSetAnsiColor_index_number_formats_accepted_by_std_fmt_parseInt(t *testing.T) {
	// The index is parsed with std.fmt.parseInt, which accepts a leading
	// '+', a leading '-' for zero, leading zeros and '_' between digits.
	for _, c := range []struct {
		body string
		idx  uint8
	}{
		{"+1;?", 1},
		{"-0;?", 0},
		{"007;?", 7},
		{"1_0;?", 10},
		{"2_5_5;?", 255},
	} {
		it := oscA_term51Tokens(c.body)
		list := oscA_parseGetSetAnsiColor(t, OSCParsersColorOperationOsc4, &it)
		defer list.deinit()
		expectEqual(t, 1, list.Count())
		expectEqual(t, oscA_colQuery(oscA_colPalette(c.idx)), *list.At(0))
	}

	// A leading or trailing underscore and a bare sign are rejected.
	for _, body := range []string{"_1;?", "1_;?", "+;?", "-;?", "_;?"} {
		it := oscA_term51Tokens(body)
		list := oscA_parseGetSetAnsiColor(t, OSCParsersColorOperationOsc4, &it)
		defer list.deinit()
		expectEqual(t, 0, list.Count())
	}
}

// Zig: test "term51: parseGetSetAnsiColor: query and set specs"
func TestOSCParsersColor_term51_parseGetSetAnsiColor_query_and_set_specs(t *testing.T) {
	for _, body := range []string{"1;red", "1;#ff0000", "1;#f00", "1;rgb:ff/00/00"} {
		it := oscA_term51Tokens(body)
		list := oscA_parseGetSetAnsiColor(t, OSCParsersColorOperationOsc4, &it)
		defer list.deinit()
		expectEqual(t, 1, list.Count())
		expectEqual(t, oscA_colSet(oscA_colPalette(1), oscA_colRed), *list.At(0))
	}

	// Only exactly "?" is a query; anything else must be a color.
	for _, body := range []string{"1;??", "1; ?", "1;? ", "1;bogus", "1;#ff", "1;rgb:ff/00"} {
		it := oscA_term51Tokens(body)
		list := oscA_parseGetSetAnsiColor(t, OSCParsersColorOperationOsc4, &it)
		defer list.deinit()
		expectEqual(t, 0, list.Count())
	}
}

// Zig: test "term51: parseGetSetAnsiColor: the first error ends parsing and keeps earlier requests"
func TestOSCParsersColor_term51_parseGetSetAnsiColor_the_first_error_ends_parsing_and_keeps_earlier_requests(t *testing.T) {
	for _, body := range []string{
		// invalid index
		"0;red;x;#0000ff;1;#0000ff",
		// out of range index
		"0;red;512;#0000ff;1;#0000ff",
		// invalid spec
		"0;red;1;bogus;2;#0000ff",
	} {
		it := oscA_term51Tokens(body)
		list := oscA_parseGetSetAnsiColor(t, OSCParsersColorOperationOsc4, &it)
		defer list.deinit()
		expectEqual(t, 1, list.Count())
		expectEqual(t, oscA_colSet(oscA_colPalette(0), oscA_colRed), *list.At(0))
	}
}

// Zig: test "term51: parseGetSetAnsiColor: empty fields are skipped, not treated as errors"
func TestOSCParsersColor_term51_parseGetSetAnsiColor_empty_fields_are_skipped_not_treated_as_errors(t *testing.T) {
	// The tokenizer never yields empty tokens, so "0;;red" pairs 0 with red.
	it := oscA_term51Tokens(";;0;;red;;1;?;;")
	list := oscA_parseGetSetAnsiColor(t, OSCParsersColorOperationOsc4, &it)
	defer list.deinit()
	expectEqual(t, 2, list.Count())
	expectEqual(t, oscA_colSet(oscA_colPalette(0), oscA_colRed), *list.At(0))
	expectEqual(t, oscA_colQuery(oscA_colPalette(1)), *list.At(1))
}

// Zig: test "term51: parseGetSetAnsiColor: many requests keep their order and duplicates"
func TestOSCParsersColor_term51_parseGetSetAnsiColor_many_requests_keep_their_order_and_duplicates(t *testing.T) {
	it := oscA_term51Tokens("3;red;3;?;3;#0000ff;4;?;256;?;255;red;0;?")
	list := oscA_parseGetSetAnsiColor(t, OSCParsersColorOperationOsc4, &it)
	defer list.deinit()
	expectEqual(t, 7, list.Count())
	expectEqual(t, oscA_colSet(oscA_colPalette(3), oscA_colRed), *list.At(0))
	expectEqual(t, oscA_colQuery(oscA_colPalette(3)), *list.At(1))
	expectEqual(t, oscA_colSet(oscA_colPalette(3), oscA_colBlue), *list.At(2))
	expectEqual(t, oscA_colQuery(oscA_colPalette(4)), *list.At(3))
	expectEqual(t, oscA_colQuery(oscA_colSpecial(ColorSpecialBold)), *list.At(4))
	expectEqual(t, oscA_colSet(oscA_colPalette(255), oscA_colRed), *list.At(5))
	expectEqual(t, oscA_colQuery(oscA_colPalette(0)), *list.At(6))
}

// Zig: test "term51: parseResetAnsiColor: no parameters resets the whole table"
func TestOSCParsersColor_term51_parseResetAnsiColor_no_parameters_resets_the_whole_table(t *testing.T) {
	for _, body := range []string{"", ";", ";;;"} {
		{
			it := oscA_term51Tokens(body)
			list := oscA_parseResetAnsiColor(t, OSCParsersColorOperationOsc104, &it)
			defer list.deinit()
			expectEqual(t, 1, list.Count())
			expectEqual(t, OSCParsersColorRequest{Tag: OSCParsersColorRequestResetPalette}, *list.At(0))
		}
		{
			it := oscA_term51Tokens(body)
			list := oscA_parseResetAnsiColor(t, OSCParsersColorOperationOsc105, &it)
			defer list.deinit()
			expectEqual(t, 1, list.Count())
			expectEqual(t, OSCParsersColorRequest{Tag: OSCParsersColorRequestResetSpecial}, *list.At(0))
		}
	}
}

// Zig: test "term51: parseResetAnsiColor: only invalid parameters also resets the whole table"
func TestOSCParsersColor_term51_parseResetAnsiColor_only_invalid_parameters_also_resets_the_whole_table(t *testing.T) {
	// Invalid parameters are skipped. If that leaves nothing, the result
	// is the same as if no parameters had been given at all.
	for _, body := range []string{"abc", "512", "261", "-1", "abc;999; 1"} {
		it := oscA_term51Tokens(body)
		list := oscA_parseResetAnsiColor(t, OSCParsersColorOperationOsc104, &it)
		defer list.deinit()
		expectEqual(t, 1, list.Count())
		expectEqual(t, OSCParsersColorRequest{Tag: OSCParsersColorRequestResetPalette}, *list.At(0))
	}
	for _, body := range []string{"abc", "5", "8", "256"} {
		it := oscA_term51Tokens(body)
		list := oscA_parseResetAnsiColor(t, OSCParsersColorOperationOsc105, &it)
		defer list.deinit()
		expectEqual(t, 1, list.Count())
		expectEqual(t, OSCParsersColorRequest{Tag: OSCParsersColorRequestResetSpecial}, *list.At(0))
	}
}

// Zig: test "term51: parseResetAnsiColor: OSC 104 index boundaries"
func TestOSCParsersColor_term51_parseResetAnsiColor_OSC_104_index_boundaries(t *testing.T) {
	// Invalid entries (261, 264, 511, 512) are skipped and parsing goes on.
	it := oscA_term51Tokens("0;255;256;260;261;264;511;512;257;258;259;1")
	list := oscA_parseResetAnsiColor(t, OSCParsersColorOperationOsc104, &it)
	defer list.deinit()
	expectEqual(t, 8, list.Count())
	expectEqual(t, oscA_colReset(oscA_colPalette(0)), *list.At(0))
	expectEqual(t, oscA_colReset(oscA_colPalette(255)), *list.At(1))
	expectEqual(t, oscA_colReset(oscA_colSpecial(ColorSpecialBold)), *list.At(2))
	expectEqual(t, oscA_colReset(oscA_colSpecial(ColorSpecialItalic)), *list.At(3))
	expectEqual(t, oscA_colReset(oscA_colSpecial(ColorSpecialUnderline)), *list.At(4))
	expectEqual(t, oscA_colReset(oscA_colSpecial(ColorSpecialBlink)), *list.At(5))
	expectEqual(t, oscA_colReset(oscA_colSpecial(ColorSpecialReverse)), *list.At(6))
	expectEqual(t, oscA_colReset(oscA_colPalette(1)), *list.At(7))
}

// Zig: test "term51: parseResetAnsiColor: OSC 105 index boundaries"
func TestOSCParsersColor_term51_parseResetAnsiColor_OSC_105_index_boundaries(t *testing.T) {
	// 5-7 fit in a u3 but are not special colors; 8 and up do not fit.
	it := oscA_term51Tokens("0;1;2;3;4;5;7;8;256;511;512;0")
	list := oscA_parseResetAnsiColor(t, OSCParsersColorOperationOsc105, &it)
	defer list.deinit()
	expectEqual(t, 6, list.Count())
	expectEqual(t, oscA_colReset(oscA_colSpecial(ColorSpecialBold)), *list.At(0))
	expectEqual(t, oscA_colReset(oscA_colSpecial(ColorSpecialUnderline)), *list.At(1))
	expectEqual(t, oscA_colReset(oscA_colSpecial(ColorSpecialBlink)), *list.At(2))
	expectEqual(t, oscA_colReset(oscA_colSpecial(ColorSpecialReverse)), *list.At(3))
	expectEqual(t, oscA_colReset(oscA_colSpecial(ColorSpecialItalic)), *list.At(4))
	// Duplicates are kept.
	expectEqual(t, oscA_colReset(oscA_colSpecial(ColorSpecialBold)), *list.At(5))
}

// Zig: test "term51: parseResetAnsiColor: index number formats accepted by std.fmt.parseInt"
func TestOSCParsersColor_term51_parseResetAnsiColor_index_number_formats_accepted_by_std_fmt_parseInt(t *testing.T) {
	it := oscA_term51Tokens("+1;-0;007;1_0;_1;1_;+;0x1;1.0;2 ")
	list := oscA_parseResetAnsiColor(t, OSCParsersColorOperationOsc104, &it)
	defer list.deinit()
	expectEqual(t, 4, list.Count())
	expectEqual(t, oscA_colReset(oscA_colPalette(1)), *list.At(0))
	expectEqual(t, oscA_colReset(oscA_colPalette(0)), *list.At(1))
	expectEqual(t, oscA_colReset(oscA_colPalette(7)), *list.At(2))
	expectEqual(t, oscA_colReset(oscA_colPalette(10)), *list.At(3))
}

// Zig: test "term51: parseGetSetDynamicColor: empty input"
func TestOSCParsersColor_term51_parseGetSetDynamicColor_empty_input(t *testing.T) {
	for _, body := range []string{"", ";", ";;;"} {
		it := oscA_term51Tokens(body)
		list := oscA_parseGetSetDynamicColor(t, ColorDynamicForeground, &it)
		defer list.deinit()
		expectEqual(t, 0, list.Count())
	}
}

// Zig: test "term51: parseGetSetDynamicColor: query and set for every starting color"
func TestOSCParsersColor_term51_parseGetSetDynamicColor_query_and_set_for_every_starting_color(t *testing.T) {
	for _, d := range oscA_colDynamics {
		color := d.color
		{
			it := oscA_term51Tokens("?")
			list := oscA_parseGetSetDynamicColor(t, color, &it)
			defer list.deinit()
			expectEqual(t, 1, list.Count())
			expectEqual(t, oscA_colQuery(oscA_colDynamic(color)), *list.At(0))
		}
		{
			it := oscA_term51Tokens("#0000ff")
			list := oscA_parseGetSetDynamicColor(t, color, &it)
			defer list.deinit()
			expectEqual(t, 1, list.Count())
			expectEqual(t, oscA_colSet(oscA_colDynamic(color), oscA_colBlue), *list.At(0))
		}
	}
}

// Zig: test "term51: parseGetSetDynamicColor: successive values use successive colors"
func TestOSCParsersColor_term51_parseGetSetDynamicColor_successive_values_use_successive_colors(t *testing.T) {
	// Ten values starting at the foreground cover all ten dynamic colors;
	// the eleventh value is ignored.
	it := oscA_term51Tokens("red;?;#0000ff;?;?;?;?;?;?;red;#0000ff")
	list := oscA_parseGetSetDynamicColor(t, ColorDynamicForeground, &it)
	defer list.deinit()
	expectEqual(t, 10, list.Count())
	expectEqual(t, oscA_colSet(oscA_colDynamic(ColorDynamicForeground), oscA_colRed), *list.At(0))
	expectEqual(t, oscA_colQuery(oscA_colDynamic(ColorDynamicBackground)), *list.At(1))
	expectEqual(t, oscA_colSet(oscA_colDynamic(ColorDynamicCursor), oscA_colBlue), *list.At(2))
	expectEqual(t, oscA_colQuery(oscA_colDynamic(ColorDynamicPointerForeground)), *list.At(3))
	expectEqual(t, oscA_colQuery(oscA_colDynamic(ColorDynamicPointerBackground)), *list.At(4))
	expectEqual(t, oscA_colQuery(oscA_colDynamic(ColorDynamicTektronixForeground)), *list.At(5))
	expectEqual(t, oscA_colQuery(oscA_colDynamic(ColorDynamicTektronixBackground)), *list.At(6))
	expectEqual(t, oscA_colQuery(oscA_colDynamic(ColorDynamicHighlightBackground)), *list.At(7))
	expectEqual(t, oscA_colQuery(oscA_colDynamic(ColorDynamicTektronixCursor)), *list.At(8))
	expectEqual(t, oscA_colSet(oscA_colDynamic(ColorDynamicHighlightForeground), oscA_colRed), *list.At(9))
}

// Zig: test "term51: parseGetSetDynamicColor: values past the last dynamic color are ignored"
func TestOSCParsersColor_term51_parseGetSetDynamicColor_values_past_the_last_dynamic_color_are_ignored(t *testing.T) {
	{
		it := oscA_term51Tokens("red;#0000ff")
		list := oscA_parseGetSetDynamicColor(t, ColorDynamicHighlightForeground, &it)
		defer list.deinit()
		expectEqual(t, 1, list.Count())
		expectEqual(t, oscA_colSet(oscA_colDynamic(ColorDynamicHighlightForeground), oscA_colRed), *list.At(0))
		// The second value was never consumed.
		expectEqualStrings(t, "#0000ff", string(zigUnwrap(t, it.next())))
	}
	{
		// Even an invalid value after the last color is not an error.
		it := oscA_term51Tokens("?;?;bogus")
		list := oscA_parseGetSetDynamicColor(t, ColorDynamicTektronixCursor, &it)
		defer list.deinit()
		expectEqual(t, 2, list.Count())
		expectEqual(t, oscA_colQuery(oscA_colDynamic(ColorDynamicTektronixCursor)), *list.At(0))
		expectEqual(t, oscA_colQuery(oscA_colDynamic(ColorDynamicHighlightForeground)), *list.At(1))
	}
}

// Zig: test "term51: parseGetSetDynamicColor: the first invalid value ends parsing"
func TestOSCParsersColor_term51_parseGetSetDynamicColor_the_first_invalid_value_ends_parsing(t *testing.T) {
	{
		it := oscA_term51Tokens("red;bogus;#0000ff")
		list := oscA_parseGetSetDynamicColor(t, ColorDynamicForeground, &it)
		defer list.deinit()
		expectEqual(t, 1, list.Count())
		expectEqual(t, oscA_colSet(oscA_colDynamic(ColorDynamicForeground), oscA_colRed), *list.At(0))
	}
	// Only exactly "?" is a query.
	for _, body := range []string{"bogus;red", "??;red", " ?;red", "#ff;red"} {
		it := oscA_term51Tokens(body)
		list := oscA_parseGetSetDynamicColor(t, ColorDynamicForeground, &it)
		defer list.deinit()
		expectEqual(t, 0, list.Count())
	}
}

// Zig: test "term51: parseGetSetDynamicColor: empty fields do not advance to the next color"
func TestOSCParsersColor_term51_parseGetSetDynamicColor_empty_fields_do_not_advance_to_the_next_color(t *testing.T) {
	// The tokenizer never yields empty tokens, so "red;;blue" sets the
	// foreground and the background, not the foreground and the cursor.
	it := oscA_term51Tokens(";red;;#0000ff;")
	list := oscA_parseGetSetDynamicColor(t, ColorDynamicForeground, &it)
	defer list.deinit()
	expectEqual(t, 2, list.Count())
	expectEqual(t, oscA_colSet(oscA_colDynamic(ColorDynamicForeground), oscA_colRed), *list.At(0))
	expectEqual(t, oscA_colSet(oscA_colDynamic(ColorDynamicBackground), oscA_colBlue), *list.At(1))
}

// Zig: test "term51: parseResetDynamicColor: no parameters resets the color"
func TestOSCParsersColor_term51_parseResetDynamicColor_no_parameters_resets_the_color(t *testing.T) {
	for _, d := range oscA_colDynamics {
		color := d.color
		for _, body := range []string{"", ";", ";;;"} {
			it := oscA_term51Tokens(body)
			list := oscA_parseResetDynamicColor(t, color, &it)
			defer list.deinit()
			expectEqual(t, 1, list.Count())
			expectEqual(t, oscA_colReset(oscA_colDynamic(color)), *list.At(0))
		}
	}
}

// Zig: test "term51: parseResetDynamicColor: any parameter yields nothing"
func TestOSCParsersColor_term51_parseResetDynamicColor_any_parameter_yields_nothing(t *testing.T) {
	for _, body := range []string{" ", "x", "?", "red", "10", ";x;", "a;b"} {
		it := oscA_term51Tokens(body)
		list := oscA_parseResetDynamicColor(t, ColorDynamicBackground, &it)
		defer list.deinit()
		expectEqual(t, 0, list.Count())
	}
}

// Zig: test "term51: parseResetDynamicColor: consumes exactly one token"
func TestOSCParsersColor_term51_parseResetDynamicColor_consumes_exactly_one_token(t *testing.T) {
	it := oscA_term51Tokens("a;b;c")
	list := oscA_parseResetDynamicColor(t, ColorDynamicCursor, &it)
	defer list.deinit()
	expectEqual(t, 0, list.Count())
	expectEqualStrings(t, "b", string(zigUnwrap(t, it.next())))
}
