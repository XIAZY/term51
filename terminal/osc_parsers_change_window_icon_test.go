// Tests translated one for one from src/terminal/osc/parsers/change_window_icon.zig
// (Ghostty v1.3.1 plus the term51 additional tests). Each Go test carries the
// exact Zig title in a "// Zig: test" comment. The helpers expect, expectEqual,
// expectEqualSlices, expectEqualStrings and zigUnwrap are defined in
// Parser_test.go.
//
// Helpers shared by the osc_parsers_*_test.go files of this group (all named
// oscA_...) are defined here:
//   - oscA_term51Feed is Zig's term51Feed, which is the same function in
//     every file under osc/parsers/;
//   - oscA_testingAllocator stands in for std.testing.allocator: the wireframe
//     keeps only whether OSCParser.alloc is nil (Zig: null) or not;
//   - oscA_unwrapPtr is `parse(...).?` / `p.end(...).?` (a ?*Command), and
//     the Zig `.?.*` is `*oscA_unwrapPtr(...)`;
//   - oscA_ptrAt is Zig's `s.ptr[i]`: it reads the byte i of the memory a
//     slice points at, which for a [:0] slice reaches the NUL terminator at
//     s.len. In Go that is a reslice up to the capacity, so it needs the
//     implementation to return slices into the parser buffer (as Zig does)
//     with the NUL still behind them.
//
// Translation notes:
//   - `.init(null)` is OSCParserInit(nil) and `.init(testing.allocator)` is
//     OSCParserInit(oscA_testingAllocator); `defer p.deinit()` is
//     `defer p.Deinit()`.
//   - `p.next(c)` is p.Next(c), `p.end(x)` is p.End(Some(x)) or
//     p.End(None[uint8]()), and a null ?*Command is a nil *OSCCommand.
//   - `cmd == .change_window_icon` is cmd.Tag == OSCCommandKeyChangeWindowIcon.
//   - `inline for` over a tuple of ?u8 values is a loop over a []Optional[uint8].

package terminal

import (
	"strings"
	"testing"
)

// oscA_term51Feed is the term51 test helper term51Feed (the same function in
// every file under osc/parsers/).
func oscA_term51Feed(p *OSCParser, input string) {
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}
}

// oscA_testingAllocator stands in for std.testing.allocator: any non-nil
// value is "an allocator was provided".
var oscA_testingAllocator any = struct{}{}

// oscA_unwrapPtr is `o.?` on a ?*Command: it fails the test if o is nil.
func oscA_unwrapPtr(t *testing.T, o *OSCCommand) *OSCCommand {
	t.Helper()
	if o == nil {
		t.Fatalf("unwrapped a null optional")
	}
	return o
}

// oscA_ptrAt is Zig's `s.ptr[i]`: byte i of the memory behind a slice, even
// at or past its length.
func oscA_ptrAt(s []uint8, i int) uint8 {
	return s[:i+1][i]
}

// oscA_terms is the Zig tuple `.{ @as(?u8, null), @as(?u8, 0x07), @as(?u8, 0x1b) }`.
var oscA_terms = []Optional[uint8]{None[uint8](), Some[uint8](0x07), Some[uint8](0x1b)}

// Zig: test "OSC 1: change_window_icon"
func TestOSCParsersChangeWindowIcon_OSC_1_change_window_icon(t *testing.T) {
	p := OSCParserInit(nil)
	p.Next('1')
	p.Next(';')
	p.Next('a')
	p.Next('b')
	cmd := *oscA_unwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyChangeWindowIcon, "cmd == .change_window_icon")
	expectEqualStrings(t, "ab", string(cmd.ChangeWindowIcon))
}

// Zig: test "term51: parse: basic icon name"
func TestOSCParsersChangeWindowIcon_term51_parse_basic_icon_name(t *testing.T) {
	for _, term := range oscA_terms {
		p := OSCParserInit(nil)
		oscA_term51Feed(&p, "1;icon")
		cmd := *oscA_unwrapPtr(t, OSCParsersChangeWindowIconParse(&p, term))
		expect(t, cmd.Tag == OSCCommandKeyChangeWindowIcon, "cmd == .change_window_icon")
		expectEqualStrings(t, "icon", string(cmd.ChangeWindowIcon))
		expectEqual(t, uint8(0), oscA_ptrAt(cmd.ChangeWindowIcon, 4))
	}
}

// Zig: test "term51: parse: missing separator is invalid"
func TestOSCParsersChangeWindowIcon_term51_parse_missing_separator_is_invalid(t *testing.T) {
	p := OSCParserInit(nil)
	oscA_term51Feed(&p, "1")
	expect(t, p.writer == nil, "p.writer == null")
	expect(t, OSCParsersChangeWindowIconParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
	expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	expect(t, p.command.Tag == OSCCommandKeyInvalid, "p.command == .invalid")
}

// Zig: test "term51: parse: empty icon name"
func TestOSCParsersChangeWindowIcon_term51_parse_empty_icon_name(t *testing.T) {
	p := OSCParserInit(nil)
	oscA_term51Feed(&p, "1;")
	cmd := *oscA_unwrapPtr(t, OSCParsersChangeWindowIconParse(&p, None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyChangeWindowIcon, "cmd == .change_window_icon")
	expectEqual(t, 0, len(cmd.ChangeWindowIcon))
}

// Zig: test "term51: parse: semicolons and arbitrary bytes are kept"
func TestOSCParsersChangeWindowIcon_term51_parse_semicolons_and_arbitrary_bytes_are_kept(t *testing.T) {
	p := OSCParserInit(nil)
	oscA_term51Feed(&p, "1;a;b\x00\x01\xff")
	cmd := *oscA_unwrapPtr(t, OSCParsersChangeWindowIconParse(&p, None[uint8]()))
	expectEqualSlices(t, []uint8("a;b\x00\x01\xff"), cmd.ChangeWindowIcon)
}

// Zig: test "term51: parse: buffer limit"
func TestOSCParsersChangeWindowIcon_term51_parse_buffer_limit(t *testing.T) {
	// One byte is always reserved for the NUL terminator.
	{
		p := OSCParserInit(nil)
		oscA_term51Feed(&p, "1;"+strings.Repeat("a", OSCParserMAX_BUF-1))
		cmd := *oscA_unwrapPtr(t, OSCParsersChangeWindowIconParse(&p, None[uint8]()))
		expectEqual(t, OSCParserMAX_BUF-1, len(cmd.ChangeWindowIcon))
	}
	{
		p := OSCParserInit(nil)
		oscA_term51Feed(&p, "1;"+strings.Repeat("a", OSCParserMAX_BUF))
		expect(t, p.state == OSCParserState1, "p.state == .@\"1\"")
		expect(t, OSCParsersChangeWindowIconParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
	{
		// An allocator does not change the limit.
		p := OSCParserInit(oscA_testingAllocator)
		defer p.Deinit()
		oscA_term51Feed(&p, "1;"+strings.Repeat("a", OSCParserMAX_BUF))
		expect(t, OSCParsersChangeWindowIconParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
	}
}
