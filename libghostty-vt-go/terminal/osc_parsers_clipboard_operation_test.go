// Tests translated one for one from src/terminal/osc/parsers/clipboard_operation.zig
// (Ghostty v1.3.1 plus the term51 additional tests). Each Go test carries the
// exact Zig title in a "// Zig: test" comment. The helpers (expect,
// expectEqual, ...) are in Parser_test.go; oscA_term51Feed, oscA_unwrapPtr,
// oscA_ptrAt and oscA_terms are in
// osc_parsers_change_window_icon_test.go, where the translation notes are.
//
// `.clipboard_contents.kind` is cmd.ClipboardContents.kind and `.data` is
// cmd.ClipboardContents.data (the payload fields keep Zig's names).

package terminal

import (
	"strings"
	"testing"
)

// Zig: test "OSC 52: get/set clipboard"
func TestOSCParsersClipboardOperation_OSC_52_get_set_clipboard(t *testing.T) {
	p := OSCParserInit(false)

	input := "52;s;?"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyClipboardContents, "cmd == .clipboard_contents")
	expect(t, cmd.ClipboardContents.Kind == 's', "cmd.clipboard_contents.kind == 's'")
	expectEqualStrings(t, "?", string(cmd.ClipboardContents.Data))
}

// Zig: test "OSC 52: get/set clipboard (optional parameter)"
func TestOSCParsersClipboardOperation_OSC_52_get_set_clipboard_optional_parameter(t *testing.T) {
	p := OSCParserInit(false)

	input := "52;;?"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyClipboardContents, "cmd == .clipboard_contents")
	expect(t, cmd.ClipboardContents.Kind == 'c', "cmd.clipboard_contents.kind == 'c'")
	expectEqualStrings(t, "?", string(cmd.ClipboardContents.Data))
}

// Zig: test "OSC 52: get/set clipboard with allocator"
func TestOSCParsersClipboardOperation_OSC_52_get_set_clipboard_with_allocator(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()

	input := "52;s;?"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyClipboardContents, "cmd == .clipboard_contents")
	expect(t, cmd.ClipboardContents.Kind == 's', "cmd.clipboard_contents.kind == 's'")
	expectEqualStrings(t, "?", string(cmd.ClipboardContents.Data))
}

// Zig: test "OSC 52: clear clipboard"
func TestOSCParsersClipboardOperation_OSC_52_clear_clipboard(t *testing.T) {
	p := OSCParserInit(false)
	defer p.Deinit()

	input := "52;;"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(None[uint8]()))
	expect(t, cmd.Tag == OSCCommandKeyClipboardContents, "cmd == .clipboard_contents")
	expect(t, cmd.ClipboardContents.Kind == 'c', "cmd.clipboard_contents.kind == 'c'")
	expectEqualStrings(t, "", string(cmd.ClipboardContents.Data))
}

// Zig: test "term51: parse: kind and data"
func TestOSCParsersClipboardOperation_term51_parse_kind_and_data(t *testing.T) {
	for _, term := range oscA_terms {
		p := OSCParserInit(false)
		oscA_term51Feed(&p, "52;p;aGVsbG8=")
		cmd := *oscA_unwrapPtr(t, OSCParsersClipboardOperationParse(&p, term))
		expect(t, cmd.Tag == OSCCommandKeyClipboardContents, "cmd == .clipboard_contents")
		expectEqual(t, uint8('p'), cmd.ClipboardContents.Kind)
		expectEqualStrings(t, "aGVsbG8=", string(cmd.ClipboardContents.Data))
		expectEqual(t, uint8(0), oscA_ptrAt(cmd.ClipboardContents.Data, 8))
	}
}

// Zig: test "term51: parse: missing separator after 52 is invalid"
func TestOSCParsersClipboardOperation_term51_parse_missing_separator_after_52_is_invalid(t *testing.T) {
	p := OSCParserInit(false)
	oscA_term51Feed(&p, "52")
	expect(t, p.state == OSCParserState52, "p.state == .@\"52\"")
	expect(t, OSCParsersClipboardOperationParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
	expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
}

// Zig: test "term51: parse: empty data is invalid"
func TestOSCParsersClipboardOperation_term51_parse_empty_data_is_invalid(t *testing.T) {
	p := OSCParserInit(false)
	oscA_term51Feed(&p, "52;")
	expect(t, OSCParsersClipboardOperationParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
	expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
}

// Zig: test "term51: parse: empty kind defaults to c"
func TestOSCParsersClipboardOperation_term51_parse_empty_kind_defaults_to_c(t *testing.T) {
	{
		p := OSCParserInit(false)
		oscA_term51Feed(&p, "52;;")
		cmd := *oscA_unwrapPtr(t, OSCParsersClipboardOperationParse(&p, None[uint8]()))
		expectEqual(t, uint8('c'), cmd.ClipboardContents.Kind)
		expectEqualStrings(t, "", string(cmd.ClipboardContents.Data))
	}
	{
		p := OSCParserInit(false)
		oscA_term51Feed(&p, "52;;YQ==")
		cmd := *oscA_unwrapPtr(t, OSCParsersClipboardOperationParse(&p, None[uint8]()))
		expectEqual(t, uint8('c'), cmd.ClipboardContents.Kind)
		expectEqualStrings(t, "YQ==", string(cmd.ClipboardContents.Data))
	}
	{
		// Only the first semicolon is consumed.
		p := OSCParserInit(false)
		oscA_term51Feed(&p, "52;;;x")
		cmd := *oscA_unwrapPtr(t, OSCParsersClipboardOperationParse(&p, None[uint8]()))
		expectEqual(t, uint8('c'), cmd.ClipboardContents.Kind)
		expectEqualStrings(t, ";x", string(cmd.ClipboardContents.Data))
	}
}

// Zig: test "term51: parse: kind without a following separator is invalid"
func TestOSCParsersClipboardOperation_term51_parse_kind_without_a_following_separator_is_invalid(t *testing.T) {
	for _, input := range []string{
		// single kind byte and nothing else
		"52;c",
		// xterm allows several selection characters, this parser allows one
		"52;cs;YQ==",
		"52;pc;?",
		"52;abc",
	} {
		p := OSCParserInit(false)
		oscA_term51Feed(&p, input)
		expect(t, OSCParsersClipboardOperationParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
}

// Zig: test "term51: parse: any single byte is accepted as the kind"
func TestOSCParsersClipboardOperation_term51_parse_any_single_byte_is_accepted_as_the_kind(t *testing.T) {
	for _, kind := range []string{"c", "s", "p", "q", "0", "7", "Z", " ", "\x00", "\xff"} {
		p := OSCParserInit(false)
		oscA_term51Feed(&p, "52;"+kind+";?")
		cmd := *oscA_unwrapPtr(t, OSCParsersClipboardOperationParse(&p, None[uint8]()))
		expect(t, cmd.Tag == OSCCommandKeyClipboardContents, "cmd == .clipboard_contents")
		expectEqual(t, kind[0], cmd.ClipboardContents.Kind)
		expectEqualStrings(t, "?", string(cmd.ClipboardContents.Data))
	}
}

// Zig: test "term51: parse: data is not validated and keeps semicolons"
func TestOSCParsersClipboardOperation_term51_parse_data_is_not_validated_and_keeps_semicolons(t *testing.T) {
	p := OSCParserInit(false)
	oscA_term51Feed(&p, "52;c;not base64!;x\x01\xff")
	cmd := *oscA_unwrapPtr(t, OSCParsersClipboardOperationParse(&p, None[uint8]()))
	expectEqual(t, uint8('c'), cmd.ClipboardContents.Kind)
	expectEqualSlices(t, []uint8("not base64!;x\x01\xff"), cmd.ClipboardContents.Data)
}

// Zig: test "term51: parse: buffer limit without allocator"
func TestOSCParsersClipboardOperation_term51_parse_buffer_limit_without_allocator(t *testing.T) {
	// Without an allocator the fixed buffer is used: "c;" plus the data
	// plus the NUL terminator must fit in MAX_BUF.
	{
		data := strings.Repeat("A", OSCParserMAX_BUF-3)
		p := OSCParserInit(false)
		oscA_term51Feed(&p, "52;c;"+data)
		cmd := *oscA_unwrapPtr(t, OSCParsersClipboardOperationParse(&p, None[uint8]()))
		expectEqual(t, uint8('c'), cmd.ClipboardContents.Kind)
		expectEqualStrings(t, data, string(cmd.ClipboardContents.Data))
	}
	{
		data := strings.Repeat("A", OSCParserMAX_BUF-2)
		p := OSCParserInit(false)
		oscA_term51Feed(&p, "52;c;"+data)
		expect(t, p.state == OSCParserState52, "p.state == .@\"52\"")
		expect(t, OSCParsersClipboardOperationParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
}

// Zig: test "term51: parse: no buffer limit with allocator"
func TestOSCParsersClipboardOperation_term51_parse_no_buffer_limit_with_allocator(t *testing.T) {
	for _, n := range []int{OSCParserMAX_BUF - 2, OSCParserMAX_BUF, OSCParserMAX_BUF*4 + 1} {
		data := strings.Repeat("A", n)
		p := OSCParserInit(true)
		defer p.Deinit()
		oscA_term51Feed(&p, "52;s;"+data)
		cmd := *oscA_unwrapPtr(t, OSCParsersClipboardOperationParse(&p, None[uint8]()))
		expectEqual(t, uint8('s'), cmd.ClipboardContents.Kind)
		expectEqualStrings(t, data, string(cmd.ClipboardContents.Data))
	}
}
