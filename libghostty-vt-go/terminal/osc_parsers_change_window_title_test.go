// Tests translated one for one from src/terminal/osc/parsers/change_window_title.zig
// (Ghostty v1.3.1 plus the term51 additional tests). Each Go test carries the
// exact Zig title in a "// Zig: test" comment. The helpers (expect,
// expectEqual, ...) are in Parser_test.go; oscA_term51Feed, oscA_unwrapPtr,
// oscA_ptrAt and oscA_terms are in
// osc_parsers_change_window_icon_test.go, where the translation notes are.

package terminal

import (
	"strings"
	"testing"
)

// Zig: test "OSC 0: change_window_title"
func TestOSCParsersChangeWindowTitle_OSC_0_change_window_title(t *testing.T) {
	p := OSCParserInit(false)
	p.Next('0')
	p.Next(';')
	p.Next('a')
	p.Next('b')
	cmd := *oscA_unwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyChangeWindowTitle, "cmd == .change_window_title")
	expectEqualStrings(t, "ab", string(cmd.ChangeWindowTitle))
}

// Zig: test "OSC 0: longer than buffer"
func TestOSCParsersChangeWindowTitle_OSC_0_longer_than_buffer(t *testing.T) {
	p := OSCParserInit(false)

	input := "0;" + strings.Repeat("a", OSCParserMAX_BUF+2)
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
}

// Zig: test "OSC 0: one shorter than buffer length"
func TestOSCParsersChangeWindowTitle_OSC_0_one_shorter_than_buffer_length(t *testing.T) {
	p := OSCParserInit(false)

	prefix := "0;"
	title := strings.Repeat("a", OSCParserMAX_BUF-1)
	input := prefix + title
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyChangeWindowTitle, "cmd == .change_window_title")
	expectEqualStrings(t, title, string(cmd.ChangeWindowTitle))
}

// Zig: test "OSC 0: exactly at buffer length"
func TestOSCParsersChangeWindowTitle_OSC_0_exactly_at_buffer_length(t *testing.T) {
	p := OSCParserInit(false)

	prefix := "0;"
	title := strings.Repeat("a", OSCParserMAX_BUF)
	input := prefix + title
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	// This should be null because we always reserve space for a null terminator.
	expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
}

// Zig: test "OSC 2: change_window_title with 2"
func TestOSCParsersChangeWindowTitle_OSC_2_change_window_title_with_2(t *testing.T) {
	p := OSCParserInit(false)
	p.Next('2')
	p.Next(';')
	p.Next('a')
	p.Next('b')
	cmd := *oscA_unwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyChangeWindowTitle, "cmd == .change_window_title")
	expectEqualStrings(t, "ab", string(cmd.ChangeWindowTitle))
}

// Zig: test "OSC 2: change_window_title with utf8"
func TestOSCParsersChangeWindowTitle_OSC_2_change_window_title_with_utf8(t *testing.T) {
	p := OSCParserInit(false)
	p.Next('2')
	p.Next(';')
	// '—' EM DASH U+2014 (E2 80 94)
	p.Next(0xE2)
	p.Next(0x80)
	p.Next(0x94)

	p.Next(' ')
	// '‐' HYPHEN U+2010 (E2 80 90)
	// Intententionally chosen to conflict with the 0x90 C1 control
	p.Next(0xE2)
	p.Next(0x80)
	p.Next(0x90)
	cmd := *oscA_unwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyChangeWindowTitle, "cmd == .change_window_title")
	expectEqualStrings(t, "— ‐", string(cmd.ChangeWindowTitle))
}

// Zig: test "OSC 2: change_window_title empty"
func TestOSCParsersChangeWindowTitle_OSC_2_change_window_title_empty(t *testing.T) {
	p := OSCParserInit(false)
	p.Next('2')
	p.Next(';')
	cmd := *oscA_unwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyChangeWindowTitle, "cmd == .change_window_title")
	expectEqualStrings(t, "", string(cmd.ChangeWindowTitle))
}

// Zig: test "term51: parse: OSC 0 and OSC 2 both produce change_window_title"
func TestOSCParsersChangeWindowTitle_term51_parse_OSC_0_and_OSC_2_both_produce_change_window_title(t *testing.T) {
	for _, input := range []string{"0;ab", "2;ab"} {
		p := OSCParserInit(false)
		oscA_term51Feed(&p, input)
		cmd := *oscA_unwrapPtr(t, OSCParsersChangeWindowTitleParse(&p, None[uint8]()))
		expect(t, cmd.Tag == OSCCommandKeyChangeWindowTitle, "cmd == .change_window_title")
		expectEqualStrings(t, "ab", string(cmd.ChangeWindowTitle))
		// The title is NUL terminated inside the parser buffer.
		expectEqual(t, uint8(0), oscA_ptrAt(cmd.ChangeWindowTitle, 2))
	}
}

// Zig: test "term51: parse: terminator is ignored"
func TestOSCParsersChangeWindowTitle_term51_parse_terminator_is_ignored(t *testing.T) {
	for _, term := range []Optional[uint8]{None[uint8](), Some[uint8](0x07), Some[uint8](0x1b), Some[uint8]('x')} {
		p := OSCParserInit(false)
		oscA_term51Feed(&p, "0;title")
		cmd := *oscA_unwrapPtr(t, OSCParsersChangeWindowTitleParse(&p, term))
		expect(t, cmd.Tag == OSCCommandKeyChangeWindowTitle, "cmd == .change_window_title")
		expectEqualStrings(t, "title", string(cmd.ChangeWindowTitle))
	}
}

// Zig: test "term51: parse: missing separator is invalid"
func TestOSCParsersChangeWindowTitle_term51_parse_missing_separator_is_invalid(t *testing.T) {
	for _, input := range []string{"0", "2"} {
		p := OSCParserInit(false)
		oscA_term51Feed(&p, input)
		// No ';' was seen so no writer has been set up.
		expect(t, p.writer == nil, "p.writer == null")
		expect(t, OSCParsersChangeWindowTitleParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
		expect(t, p.command.Tag == OSCCommandKeyInvalid, "p.command == .invalid")
	}
}

// Zig: test "term51: parse: empty title"
func TestOSCParsersChangeWindowTitle_term51_parse_empty_title(t *testing.T) {
	p := OSCParserInit(false)
	oscA_term51Feed(&p, "0;")
	cmd := *oscA_unwrapPtr(t, OSCParsersChangeWindowTitleParse(&p, Some[uint8](0x07)))
	expect(t, cmd.Tag == OSCCommandKeyChangeWindowTitle, "cmd == .change_window_title")
	expectEqual(t, 0, len(cmd.ChangeWindowTitle))
}

// Zig: test "term51: parse: semicolons after the first are part of the title"
func TestOSCParsersChangeWindowTitle_term51_parse_semicolons_after_the_first_are_part_of_the_title(t *testing.T) {
	p := OSCParserInit(false)
	oscA_term51Feed(&p, "2;a;b;;c")
	cmd := *oscA_unwrapPtr(t, OSCParsersChangeWindowTitleParse(&p, None[uint8]()))
	expectEqualStrings(t, "a;b;;c", string(cmd.ChangeWindowTitle))
}

// Zig: test "term51: parse: bytes are not validated"
func TestOSCParsersChangeWindowTitle_term51_parse_bytes_are_not_validated(t *testing.T) {
	// Control bytes, an embedded NUL and invalid UTF-8 are all passed
	// through unchanged; the slice length includes the embedded NUL.
	p := OSCParserInit(false)
	oscA_term51Feed(&p, "0;a\x00b\x01\xff\x80")
	cmd := *oscA_unwrapPtr(t, OSCParsersChangeWindowTitleParse(&p, None[uint8]()))
	expectEqualSlices(t, []uint8("a\x00b\x01\xff\x80"), cmd.ChangeWindowTitle)
}

// Zig: test "term51: parse: allocator does not lift the buffer limit"
func TestOSCParsersChangeWindowTitle_term51_parse_allocator_does_not_lift_the_buffer_limit(t *testing.T) {
	// OSC 0 always uses the fixed buffer, even if an allocator is available.
	{
		p := OSCParserInit(true)
		defer p.Deinit()
		oscA_term51Feed(&p, "0;"+strings.Repeat("a", OSCParserMAX_BUF-1))
		cmd := *oscA_unwrapPtr(t, OSCParsersChangeWindowTitleParse(&p, None[uint8]()))
		expectEqual(t, OSCParserMAX_BUF-1, len(cmd.ChangeWindowTitle))
	}
	{
		p := OSCParserInit(true)
		defer p.Deinit()
		oscA_term51Feed(&p, "0;"+strings.Repeat("a", OSCParserMAX_BUF))
		// The buffer is exactly full: the bytes were all accepted...
		expect(t, p.state == OSCParserState0, "p.state == .@\"0\"")
		// ...but there is no room for the NUL terminator.
		expect(t, OSCParsersChangeWindowTitleParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
	{
		p := OSCParserInit(true)
		defer p.Deinit()
		oscA_term51Feed(&p, "0;"+strings.Repeat("a", OSCParserMAX_BUF+1))
		// The overflow was detected while feeding.
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
		expect(t, OSCParsersChangeWindowTitleParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
	}
}
