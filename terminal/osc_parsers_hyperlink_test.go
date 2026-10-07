// Tests translated one for one from src/terminal/osc/parsers/hyperlink.zig
// (Ghostty v1.3.1 plus the term51 additional tests). Each Go test carries the
// exact Zig title in a "// Zig: test" comment. The helpers (expect,
// expectEqual, ...) are in Parser_test.go; oscA_term51Feed, oscA_unwrapPtr,
// oscA_ptrAt and oscA_terms are in
// osc_parsers_change_window_icon_test.go, where the translation notes are.
//
// `cmd.hyperlink_start.id` is cmd.HyperlinkStart.id (an Optional[[]uint8]):
// `.id == null` is !id.Valid, `.id.?` is zigUnwrap, and
// `testing.expectEqual(null, ...id)` is an expect on !Valid. `.uri` is
// cmd.HyperlinkStart.uri. `.hyperlink_end` has no payload.

package terminal

import (
	"strings"
	"testing"
)

// Zig: test "OSC 8: hyperlink"
func TestOSCParsersHyperlink_OSC_8_hyperlink(t *testing.T) {
	p := OSCParserInit(false)

	input := "8;;http://example.com"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(Some[uint8](0x1b)))
	expect(t, cmd.Tag == OSCCommandKeyHyperlinkStart, "cmd == .hyperlink_start")
	expectEqualStrings(t, string(cmd.HyperlinkStart.URI), "http://example.com")
}

// Zig: test "OSC 8: hyperlink with id set"
func TestOSCParsersHyperlink_OSC_8_hyperlink_with_id_set(t *testing.T) {
	p := OSCParserInit(false)

	input := "8;id=foo;http://example.com"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(Some[uint8](0x1b)))
	expect(t, cmd.Tag == OSCCommandKeyHyperlinkStart, "cmd == .hyperlink_start")
	expectEqualStrings(t, string(zigUnwrap(t, cmd.HyperlinkStart.ID)), "foo")
	expectEqualStrings(t, string(cmd.HyperlinkStart.URI), "http://example.com")
}

// Zig: test "OSC 8: hyperlink with empty id"
func TestOSCParsersHyperlink_OSC_8_hyperlink_with_empty_id(t *testing.T) {
	p := OSCParserInit(false)

	input := "8;id=;http://example.com"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(Some[uint8](0x1b)))
	expect(t, cmd.Tag == OSCCommandKeyHyperlinkStart, "cmd == .hyperlink_start")
	expect(t, !cmd.HyperlinkStart.ID.Valid, "cmd.hyperlink_start.id == null")
	expectEqualStrings(t, string(cmd.HyperlinkStart.URI), "http://example.com")
}

// Zig: test "OSC 8: hyperlink with incomplete key"
func TestOSCParsersHyperlink_OSC_8_hyperlink_with_incomplete_key(t *testing.T) {
	p := OSCParserInit(false)

	input := "8;id;http://example.com"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(Some[uint8](0x1b)))
	expect(t, cmd.Tag == OSCCommandKeyHyperlinkStart, "cmd == .hyperlink_start")
	expect(t, !cmd.HyperlinkStart.ID.Valid, "cmd.hyperlink_start.id == null")
	expectEqualStrings(t, string(cmd.HyperlinkStart.URI), "http://example.com")
}

// Zig: test "OSC 8: hyperlink with empty key"
func TestOSCParsersHyperlink_OSC_8_hyperlink_with_empty_key(t *testing.T) {
	p := OSCParserInit(false)

	input := "8;=value;http://example.com"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(Some[uint8](0x1b)))
	expect(t, cmd.Tag == OSCCommandKeyHyperlinkStart, "cmd == .hyperlink_start")
	expect(t, !cmd.HyperlinkStart.ID.Valid, "cmd.hyperlink_start.id == null")
	expectEqualStrings(t, string(cmd.HyperlinkStart.URI), "http://example.com")
}

// Zig: test "OSC 8: hyperlink with empty key and id"
func TestOSCParsersHyperlink_OSC_8_hyperlink_with_empty_key_and_id(t *testing.T) {
	p := OSCParserInit(false)

	input := "8;=value:id=foo;http://example.com"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(Some[uint8](0x1b)))
	expect(t, cmd.Tag == OSCCommandKeyHyperlinkStart, "cmd == .hyperlink_start")
	expectEqualStrings(t, string(zigUnwrap(t, cmd.HyperlinkStart.ID)), "foo")
	expectEqualStrings(t, string(cmd.HyperlinkStart.URI), "http://example.com")
}

// Zig: test "OSC 8: hyperlink with empty uri"
func TestOSCParsersHyperlink_OSC_8_hyperlink_with_empty_uri(t *testing.T) {
	p := OSCParserInit(false)

	input := "8;id=foo;"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := p.End(Some[uint8](0x1b))
	expect(t, cmd == nil, "cmd == null")
}

// Zig: test "OSC 8: hyperlink end"
func TestOSCParsersHyperlink_OSC_8_hyperlink_end(t *testing.T) {
	p := OSCParserInit(false)

	input := "8;;"
	for i := 0; i < len(input); i++ {
		p.Next(input[i])
	}

	cmd := *oscA_unwrapPtr(t, p.End(Some[uint8](0x1b)))
	expect(t, cmd.Tag == OSCCommandKeyHyperlinkEnd, "cmd == .hyperlink_end")
}

// Zig: test "term51: parse: start without id"
func TestOSCParsersHyperlink_term51_parse_start_without_id(t *testing.T) {
	for _, term := range oscA_terms {
		p := OSCParserInit(false)
		oscA_term51Feed(&p, "8;;http://example.com")
		cmd := *oscA_unwrapPtr(t, OSCParsersHyperlinkParse(&p, term))
		expect(t, cmd.Tag == OSCCommandKeyHyperlinkStart, "cmd == .hyperlink_start")
		expect(t, !cmd.HyperlinkStart.ID.Valid, "cmd.hyperlink_start.id == null")
		expectEqualStrings(t, "http://example.com", string(cmd.HyperlinkStart.URI))
		expectEqual(t, uint8(0), oscA_ptrAt(cmd.HyperlinkStart.URI, len(cmd.HyperlinkStart.URI)))
	}
}

// Zig: test "term51: parse: missing separators are invalid"
func TestOSCParsersHyperlink_term51_parse_missing_separators_are_invalid(t *testing.T) {
	{
		// No ';' after the 8.
		p := OSCParserInit(false)
		oscA_term51Feed(&p, "8")
		expect(t, p.writer == nil, "p.writer == null")
		expect(t, OSCParsersHyperlinkParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
	for _, input := range []string{"8;", "8;id=foo", "8;http://example.com"} {
		// No ';' between the parameters and the URI.
		p := OSCParserInit(false)
		oscA_term51Feed(&p, input)
		expect(t, OSCParsersHyperlinkParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
}

// Zig: test "term51: parse: end"
func TestOSCParsersHyperlink_term51_parse_end(t *testing.T) {
	for _, input := range []string{
		"8;;",
		// parameters other than a non-empty id do not prevent an end
		"8;foo=bar;",
		"8;id=;",
		"8;id;",
		"8;ID=foo;",
	} {
		p := OSCParserInit(false)
		oscA_term51Feed(&p, input)
		cmd := *oscA_unwrapPtr(t, OSCParsersHyperlinkParse(&p, None[uint8]()))
		expect(t, cmd.Tag == OSCCommandKeyHyperlinkEnd, "cmd == .hyperlink_end")
	}
}

// Zig: test "term51: parse: empty uri with an id is invalid"
func TestOSCParsersHyperlink_term51_parse_empty_uri_with_an_id_is_invalid(t *testing.T) {
	for _, input := range []string{"8;id=foo;", "8;a=b:id=foo;"} {
		p := OSCParserInit(false)
		oscA_term51Feed(&p, input)
		expect(t, OSCParsersHyperlinkParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
}

// Zig: test "term51: parse: id among other parameters"
func TestOSCParsersHyperlink_term51_parse_id_among_other_parameters(t *testing.T) {
	for _, input := range []string{
		"8;id=foo;u",
		"8;a=b:id=foo;u",
		"8;id=foo:a=b;u",
		"8;a=b:id=foo:c=d;u",
		// a trailing ':' is harmless
		"8;id=foo:;u",
	} {
		p := OSCParserInit(false)
		oscA_term51Feed(&p, input)
		cmd := *oscA_unwrapPtr(t, OSCParsersHyperlinkParse(&p, None[uint8]()))
		expect(t, cmd.Tag == OSCCommandKeyHyperlinkStart, "cmd == .hyperlink_start")
		expectEqualStrings(t, "foo", string(zigUnwrap(t, cmd.HyperlinkStart.ID)))
		expectEqual(t, uint8(0), oscA_ptrAt(zigUnwrap(t, cmd.HyperlinkStart.ID), 3))
		expectEqualStrings(t, "u", string(cmd.HyperlinkStart.URI))
	}
}

// Zig: test "term51: parse: repeated id"
func TestOSCParsersHyperlink_term51_parse_repeated_id(t *testing.T) {
	{
		// The last non-empty id wins.
		p := OSCParserInit(false)
		oscA_term51Feed(&p, "8;id=a:id=b;u")
		cmd := *oscA_unwrapPtr(t, OSCParsersHyperlinkParse(&p, None[uint8]()))
		expectEqualStrings(t, "b", string(zigUnwrap(t, cmd.HyperlinkStart.ID)))
	}
	{
		// An empty id does not clear an earlier one.
		p := OSCParserInit(false)
		oscA_term51Feed(&p, "8;id=a:id=;u")
		cmd := *oscA_unwrapPtr(t, OSCParsersHyperlinkParse(&p, None[uint8]()))
		expectEqualStrings(t, "a", string(zigUnwrap(t, cmd.HyperlinkStart.ID)))
	}
}

// Zig: test "term51: parse: id key is case sensitive"
func TestOSCParsersHyperlink_term51_parse_id_key_is_case_sensitive(t *testing.T) {
	for _, input := range []string{"8;ID=foo;u", "8;Id=foo;u", "8; id=foo;u", "8;id =foo;u"} {
		p := OSCParserInit(false)
		oscA_term51Feed(&p, input)
		cmd := *oscA_unwrapPtr(t, OSCParsersHyperlinkParse(&p, None[uint8]()))
		expect(t, cmd.Tag == OSCCommandKeyHyperlinkStart, "cmd == .hyperlink_start")
		expect(t, !cmd.HyperlinkStart.ID.Valid, "cmd.hyperlink_start.id == null")
		expectEqualStrings(t, "u", string(cmd.HyperlinkStart.URI))
	}
}

// Zig: test "term51: parse: id value keeps equals signs and arbitrary bytes"
func TestOSCParsersHyperlink_term51_parse_id_value_keeps_equals_signs_and_arbitrary_bytes(t *testing.T) {
	p := OSCParserInit(false)
	oscA_term51Feed(&p, "8;id=a=b \x01\xff;u")
	cmd := *oscA_unwrapPtr(t, OSCParsersHyperlinkParse(&p, None[uint8]()))
	expectEqualSlices(t, []uint8("a=b \x01\xff"), zigUnwrap(t, cmd.HyperlinkStart.ID))
}

// Zig: test "term51: parse: a parameter without equals stops parameter parsing"
func TestOSCParsersHyperlink_term51_parse_a_parameter_without_equals_stops_parameter_parsing(t *testing.T) {
	// "foo" has no '=', so the loop stops and the id after it is never seen.
	{
		p := OSCParserInit(false)
		oscA_term51Feed(&p, "8;foo:id=x;u")
		cmd := *oscA_unwrapPtr(t, OSCParsersHyperlinkParse(&p, None[uint8]()))
		expect(t, cmd.Tag == OSCCommandKeyHyperlinkStart, "cmd == .hyperlink_start")
		expect(t, !cmd.HyperlinkStart.ID.Valid, "cmd.hyperlink_start.id == null")
	}
	// An id before it is kept.
	{
		p := OSCParserInit(false)
		oscA_term51Feed(&p, "8;id=x:foo;u")
		cmd := *oscA_unwrapPtr(t, OSCParsersHyperlinkParse(&p, None[uint8]()))
		expectEqualStrings(t, "x", string(zigUnwrap(t, cmd.HyperlinkStart.ID)))
	}
}

// Zig: test "term51: parse: an empty parameter hides the id that follows it"
func TestOSCParsersHyperlink_term51_parse_an_empty_parameter_hides_the_id_that_follows_it(t *testing.T) {
	// The ':' separators are replaced by NUL and the scan for the end of a
	// parameter starts one byte after its start, so an empty parameter is
	// merged with the next one and its key becomes "\x00id".
	{
		p := OSCParserInit(false)
		oscA_term51Feed(&p, "8;:id=x;u")
		cmd := *oscA_unwrapPtr(t, OSCParsersHyperlinkParse(&p, None[uint8]()))
		expect(t, cmd.Tag == OSCCommandKeyHyperlinkStart, "cmd == .hyperlink_start")
		expect(t, !cmd.HyperlinkStart.ID.Valid, "cmd.hyperlink_start.id == null")
		expectEqualStrings(t, "u", string(cmd.HyperlinkStart.URI))
	}
	{
		p := OSCParserInit(false)
		oscA_term51Feed(&p, "8;id=a::id=b;u")
		cmd := *oscA_unwrapPtr(t, OSCParsersHyperlinkParse(&p, None[uint8]()))
		expectEqualStrings(t, "a", string(zigUnwrap(t, cmd.HyperlinkStart.ID)))
	}
	{
		// With an empty URI this is an end, not an error, because no id
		// was recognised.
		p := OSCParserInit(false)
		oscA_term51Feed(&p, "8;:id=x;")
		cmd := *oscA_unwrapPtr(t, OSCParsersHyperlinkParse(&p, None[uint8]()))
		expect(t, cmd.Tag == OSCCommandKeyHyperlinkEnd, "cmd == .hyperlink_end")
	}
}

// Zig: test "term51: parse: uri keeps semicolons, colons and arbitrary bytes"
func TestOSCParsersHyperlink_term51_parse_uri_keeps_semicolons_colons_and_arbitrary_bytes(t *testing.T) {
	p := OSCParserInit(false)
	oscA_term51Feed(&p, "8;id=i;http://h:80/a;b=c:d\x01\xff")
	cmd := *oscA_unwrapPtr(t, OSCParsersHyperlinkParse(&p, None[uint8]()))
	expectEqualStrings(t, "i", string(zigUnwrap(t, cmd.HyperlinkStart.ID)))
	expectEqualSlices(t, []uint8("http://h:80/a;b=c:d\x01\xff"), cmd.HyperlinkStart.URI)
}

// Zig: test "term51: parse: buffer limit"
func TestOSCParsersHyperlink_term51_parse_buffer_limit(t *testing.T) {
	// ";" plus the URI plus the NUL terminator must fit in MAX_BUF.
	{
		uri := strings.Repeat("u", OSCParserMAX_BUF-2)
		p := OSCParserInit(false)
		oscA_term51Feed(&p, "8;;"+uri)
		cmd := *oscA_unwrapPtr(t, OSCParsersHyperlinkParse(&p, None[uint8]()))
		expect(t, cmd.Tag == OSCCommandKeyHyperlinkStart, "cmd == .hyperlink_start")
		expectEqualStrings(t, uri, string(cmd.HyperlinkStart.URI))
	}
	{
		uri := strings.Repeat("u", OSCParserMAX_BUF-1)
		p := OSCParserInit(false)
		oscA_term51Feed(&p, "8;;"+uri)
		expect(t, p.state == OSCParserState8, "p.state == .@\"8\"")
		expect(t, OSCParsersHyperlinkParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
	{
		// An allocator does not change the limit.
		uri := strings.Repeat("u", OSCParserMAX_BUF-1)
		p := OSCParserInit(true)
		defer p.Deinit()
		oscA_term51Feed(&p, "8;;"+uri)
		expect(t, OSCParsersHyperlinkParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
	}
}
