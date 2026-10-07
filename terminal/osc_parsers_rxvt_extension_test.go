// Tests translated one for one from src/terminal/osc/parsers/rxvt_extension.zig
// (Ghostty v1.3.1 plus the term51 additional tests). Each Go test carries the
// exact Zig title in a "// Zig: test" comment. The helpers expect,
// expectEqual, expectEqualSlices, expectEqualStrings and zigUnwrap are
// defined in Parser_test.go.
//
// Translation notes:
//   - `.init(null)` is OSCParserInit(nil); `p.next(ch)` is p.Next(ch);
//     `p.end('\x1b')` is p.End(Some[uint8](0x1b)).
//   - `parse(&p, null).?.*` is oscB_rxvt_unwrap(t, OSCParsersRxvtExtensionParse(...)),
//     which fails on nil and copies the command out.
//   - `inline for` over the optional terminators or over the inputs is a
//     loop over a slice.
//   - `value.ptr[n]` (the sentinel byte after the slice) is oscB_rxvt_at,
//     which reads one byte past the length of the slice (up to its
//     capacity).
//   - `p.writer == null` is `p.writer == nil`.

package terminal

import (
	"strings"
	"testing"
)

// oscB_rxvt_unwrap is Zig's `ptr.?.*` on the result of parse or end.
func oscB_rxvt_unwrap(t *testing.T, c *OSCCommand) OSCCommand {
	t.Helper()
	if c == nil {
		t.Fatalf("unwrapped a null optional")
	}
	return *c
}

// oscB_rxvt_term51Feed is Zig's term51Feed.
func oscB_rxvt_term51Feed(p *OSCParser, input []uint8) {
	for _, ch := range input {
		p.Next(ch)
	}
}

// oscB_rxvt_at is Zig's `s.ptr[i]`: the byte at index i of the memory behind
// s, which may lie past its length (the sentinel).
func oscB_rxvt_at(s []uint8, i int) uint8 {
	return s[:i+1][i]
}

// Zig: test "OSC: OSC 777 show desktop notification with title"
func TestOSCParsersRxvtExtension_OSC_OSC_777_show_desktop_notification_with_title(t *testing.T) {
	p := OSCParserInit(nil)

	input := "777;notify;Title;Body"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}

	cmd := oscB_rxvt_unwrap(t, p.End(Some[uint8](0x1b)))
	expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
	expectEqualStrings(t, string(cmd.ShowDesktopNotification.Title), "Title")
	expectEqualStrings(t, string(cmd.ShowDesktopNotification.Body), "Body")
}

// Zig: test "term51: parse: notify with title and body"
func TestOSCParsersRxvtExtension_term51_parse_notify_with_title_and_body(t *testing.T) {
	for _, term := range []Optional[uint8]{None[uint8](), Some[uint8](0x07), Some[uint8](0x1b)} {
		p := OSCParserInit(nil)
		oscB_rxvt_term51Feed(&p, []uint8("777;notify;Title;Body"))
		cmd := oscB_rxvt_unwrap(t, OSCParsersRxvtExtensionParse(&p, term))
		expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
		expectEqualStrings(t, "Title", string(cmd.ShowDesktopNotification.Title))
		expectEqualStrings(t, "Body", string(cmd.ShowDesktopNotification.Body))
		expectEqual(t, uint8(0), oscB_rxvt_at(cmd.ShowDesktopNotification.Title, 5))
		expectEqual(t, uint8(0), oscB_rxvt_at(cmd.ShowDesktopNotification.Body, 4))
	}
}

// Zig: test "term51: parse: missing separator after 777 is invalid"
func TestOSCParsersRxvtExtension_term51_parse_missing_separator_after_777_is_invalid(t *testing.T) {
	p := OSCParserInit(nil)
	oscB_rxvt_term51Feed(&p, []uint8("777"))
	expect(t, p.writer == nil, "p.writer == null")
	expect(t, OSCParsersRxvtExtensionParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
	expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
}

// Zig: test "term51: parse: missing extension, title or body separators are invalid"
func TestOSCParsersRxvtExtension_term51_parse_missing_extension_title_or_body_separators_are_invalid(t *testing.T) {
	for _, input := range []string{
		// empty data: no ';' at all
		"777;",
		// extension name only
		"777;notify",
		// extension and separator but no title separator
		"777;notify;",
		"777;notify;Title",
	} {
		p := OSCParserInit(nil)
		oscB_rxvt_term51Feed(&p, []uint8(input))
		expect(t, OSCParsersRxvtExtensionParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
}

// Zig: test "term51: parse: unknown, empty and differently cased extensions are invalid"
func TestOSCParsersRxvtExtension_term51_parse_unknown_empty_and_differently_cased_extensions_are_invalid(t *testing.T) {
	for _, input := range []string{
		"777;other;Title;Body",
		"777;;Title;Body",
		"777;Notify;Title;Body",
		"777;NOTIFY;Title;Body",
		"777; notify;Title;Body",
		"777;notify ;Title;Body",
	} {
		p := OSCParserInit(nil)
		oscB_rxvt_term51Feed(&p, []uint8(input))
		expect(t, OSCParsersRxvtExtensionParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
}

// Zig: test "term51: parse: empty title and body"
func TestOSCParsersRxvtExtension_term51_parse_empty_title_and_body(t *testing.T) {
	{
		p := OSCParserInit(nil)
		oscB_rxvt_term51Feed(&p, []uint8("777;notify;;"))
		cmd := oscB_rxvt_unwrap(t, OSCParsersRxvtExtensionParse(&p, None[uint8]()))
		expect(t, cmd.Tag == OSCCommandKeyShowDesktopNotification, "cmd == .show_desktop_notification")
		expectEqual(t, 0, len(cmd.ShowDesktopNotification.Title))
		expectEqual(t, 0, len(cmd.ShowDesktopNotification.Body))
	}
	{
		p := OSCParserInit(nil)
		oscB_rxvt_term51Feed(&p, []uint8("777;notify;T;"))
		cmd := oscB_rxvt_unwrap(t, OSCParsersRxvtExtensionParse(&p, None[uint8]()))
		expectEqualStrings(t, "T", string(cmd.ShowDesktopNotification.Title))
		expectEqualStrings(t, "", string(cmd.ShowDesktopNotification.Body))
	}
	{
		p := OSCParserInit(nil)
		oscB_rxvt_term51Feed(&p, []uint8("777;notify;;B"))
		cmd := oscB_rxvt_unwrap(t, OSCParsersRxvtExtensionParse(&p, None[uint8]()))
		expectEqualStrings(t, "", string(cmd.ShowDesktopNotification.Title))
		expectEqualStrings(t, "B", string(cmd.ShowDesktopNotification.Body))
	}
}

// Zig: test "term51: parse: further semicolons belong to the body"
func TestOSCParsersRxvtExtension_term51_parse_further_semicolons_belong_to_the_body(t *testing.T) {
	p := OSCParserInit(nil)
	oscB_rxvt_term51Feed(&p, []uint8("777;notify;T;a;b;;c"))
	cmd := oscB_rxvt_unwrap(t, OSCParsersRxvtExtensionParse(&p, None[uint8]()))
	expectEqualStrings(t, "T", string(cmd.ShowDesktopNotification.Title))
	expectEqualStrings(t, "a;b;;c", string(cmd.ShowDesktopNotification.Body))
}

// Zig: test "term51: parse: title and body bytes are not validated"
func TestOSCParsersRxvtExtension_term51_parse_title_and_body_bytes_are_not_validated(t *testing.T) {
	p := OSCParserInit(nil)
	oscB_rxvt_term51Feed(&p, []uint8("777;notify;T\x01\xff;B\x00\x80"))
	cmd := oscB_rxvt_unwrap(t, OSCParsersRxvtExtensionParse(&p, None[uint8]()))
	expectEqualSlices(t, []uint8("T\x01\xff"), cmd.ShowDesktopNotification.Title)
	expectEqualSlices(t, []uint8("B\x00\x80"), cmd.ShowDesktopNotification.Body)
}

// Zig: test "term51: parse: buffer limit"
func TestOSCParsersRxvtExtension_term51_parse_buffer_limit(t *testing.T) {
	// "notify;T;" is 9 bytes; one byte is reserved for the NUL terminator.
	const prefix = "notify;T;"
	{
		body := strings.Repeat("b", OSCParserMAX_BUF-1-len(prefix))
		p := OSCParserInit(nil)
		oscB_rxvt_term51Feed(&p, []uint8("777;"+prefix+body))
		cmd := oscB_rxvt_unwrap(t, OSCParsersRxvtExtensionParse(&p, None[uint8]()))
		expectEqualStrings(t, "T", string(cmd.ShowDesktopNotification.Title))
		expectEqualStrings(t, body, string(cmd.ShowDesktopNotification.Body))
	}
	{
		body := strings.Repeat("b", OSCParserMAX_BUF-len(prefix))
		p := OSCParserInit(nil)
		oscB_rxvt_term51Feed(&p, []uint8("777;"+prefix+body))
		expect(t, p.state == OSCParserState777, "p.state == .@\"777\"")
		expect(t, OSCParsersRxvtExtensionParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
}
