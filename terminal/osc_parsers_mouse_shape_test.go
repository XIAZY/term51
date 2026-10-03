// Tests translated one for one from src/terminal/osc/parsers/mouse_shape.zig
// (Ghostty v1.3.1 plus the term51 additional tests). Each Go test carries the
// exact Zig title in a "// Zig: test" comment. The helpers expect,
// expectEqual, expectEqualSlices, expectEqualStrings and zigUnwrap are
// defined in Parser_test.go.
//
// Translation notes:
//   - `.init(null)` is OSCParserInit(nil); `p.next(ch)` is p.Next(ch);
//     `p.end(null)` is p.End(None[uint8]()).
//   - `parse(&p, null).?.*` is oscB_mouse_shape_unwrap(t, OSCParsersMouseShapeParse(...)),
//     which fails on nil and copies the command out.
//   - `inline for` over the optional terminators is a loop over a slice.
//   - `value.ptr[n]` (the sentinel byte after the slice) is
//     oscB_mouse_shape_at, which reads one byte past the length of the slice
//     (up to its capacity).

package terminal

import (
	"strings"
	"testing"
)

// oscB_mouse_shape_unwrap is Zig's `ptr.?.*` on the result of parse or end.
func oscB_mouse_shape_unwrap(t *testing.T, c *OSCCommand) OSCCommand {
	t.Helper()
	if c == nil {
		t.Fatalf("unwrapped a null optional")
	}
	return *c
}

// oscB_mouse_shape_term51Feed is Zig's term51Feed.
func oscB_mouse_shape_term51Feed(p *OSCParser, input []uint8) {
	for _, ch := range input {
		p.Next(ch)
	}
}

// oscB_mouse_shape_at is Zig's `s.ptr[i]`: the byte at index i of the memory
// behind s, which may lie past its length (the sentinel).
func oscB_mouse_shape_at(s []uint8, i int) uint8 {
	return s[:i+1][i]
}

// Zig: test "OSC 22: pointer cursor"
func TestOSCParsersMouseShape_OSC_22_pointer_cursor(t *testing.T) {
	p := OSCParserInit(nil)

	input := "22;pointer"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}

	cmd := oscB_mouse_shape_unwrap(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyMouseShape, "cmd == .mouse_shape")
	expectEqualStrings(t, "pointer", string(cmd.MouseShape.value))
}

// Zig: test "term51: parse: basic shape name"
func TestOSCParsersMouseShape_term51_parse_basic_shape_name(t *testing.T) {
	for _, term := range []Optional[uint8]{None[uint8](), Some[uint8](0x07), Some[uint8](0x1b)} {
		p := OSCParserInit(nil)
		oscB_mouse_shape_term51Feed(&p, []uint8("22;text"))
		cmd := oscB_mouse_shape_unwrap(t, OSCParsersMouseShapeParse(&p, term))
		expect(t, cmd.Tag == OSCCommandKeyMouseShape, "cmd == .mouse_shape")
		expectEqualStrings(t, "text", string(cmd.MouseShape.value))
		expectEqual(t, uint8(0), oscB_mouse_shape_at(cmd.MouseShape.value, 4))
	}
}

// Zig: test "term51: parse: missing separator is invalid"
func TestOSCParsersMouseShape_term51_parse_missing_separator_is_invalid(t *testing.T) {
	p := OSCParserInit(nil)
	oscB_mouse_shape_term51Feed(&p, []uint8("22"))
	expect(t, p.state == OSCParserState22, "p.state == .@\"22\"")
	expect(t, OSCParsersMouseShapeParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
	expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
}

// Zig: test "term51: parse: empty shape name"
func TestOSCParsersMouseShape_term51_parse_empty_shape_name(t *testing.T) {
	p := OSCParserInit(nil)
	oscB_mouse_shape_term51Feed(&p, []uint8("22;"))
	cmd := oscB_mouse_shape_unwrap(t, OSCParsersMouseShapeParse(&p, None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyMouseShape, "cmd == .mouse_shape")
	expectEqual(t, 0, len(cmd.MouseShape.value))
}

// Zig: test "term51: parse: the value is not validated"
func TestOSCParsersMouseShape_term51_parse_the_value_is_not_validated(t *testing.T) {
	// Unknown names, upper case, semicolons and invalid UTF-8 are all
	// returned as-is.
	p := OSCParserInit(nil)
	oscB_mouse_shape_term51Feed(&p, []uint8("22;Not A Cursor;x\xff"))
	cmd := oscB_mouse_shape_unwrap(t, OSCParsersMouseShapeParse(&p, None[uint8]()))
	expectEqualSlices(t, []uint8("Not A Cursor;x\xff"), cmd.MouseShape.value)
}

// Zig: test "term51: parse: buffer limit"
func TestOSCParsersMouseShape_term51_parse_buffer_limit(t *testing.T) {
	{
		p := OSCParserInit(nil)
		oscB_mouse_shape_term51Feed(&p, []uint8("22;"+strings.Repeat("a", OSCParserMAX_BUF-1)))
		cmd := oscB_mouse_shape_unwrap(t, OSCParsersMouseShapeParse(&p, None[uint8]()))
		expectEqual(t, OSCParserMAX_BUF-1, len(cmd.MouseShape.value))
	}
	{
		p := OSCParserInit(nil)
		oscB_mouse_shape_term51Feed(&p, []uint8("22;"+strings.Repeat("a", OSCParserMAX_BUF)))
		expect(t, p.state == OSCParserState22, "p.state == .@\"22\"")
		expect(t, OSCParsersMouseShapeParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
}
