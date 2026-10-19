// Tests translated one for one from src/terminal/osc/parsers/kitty_color.zig
// (Ghostty v1.3.1 plus the term51 additional tests). Each Go test carries the
// exact Zig title in a "// Zig: test" comment. The helpers expect,
// expectEqual, expectEqualSlices, expectEqualStrings and zigUnwrap are
// defined in Parser_test.go.
//
// Translation notes:
//   - `.init(testing.allocator)` is OSCParserInit(true), any non-nil
//     value; `.init(null)` is OSCParserInit(false). The leak check of
//     testing.allocator has no Go counterpart; the `defer p.deinit()` calls
//     are kept.
//   - `p.next(ch)` is p.Next(ch); `p.end('\x1b')` is p.End(Some[uint8](0x1b));
//     `p.end(null)` is p.End(None[uint8]()).
//   - `parse(&p, null).?.*` is oscB_kc_unwrap(t, OSCParsersKittyColorParse(...)),
//     which fails on nil and copies the command out.
//   - `cmd.kitty_color_protocol.list.items` is
//     cmd.KittyColorProtocol.list.
//   - `item == .query` is item.Tag == KittyColorOSCRequestQuery;
//     `item.query`, `item.set` and `item.reset` are the safety-checked union
//     accesses oscB_kc_query, oscB_kc_set and oscB_kc_reset.
//   - expectEqual on a Kind union compares the tags and then the payload of
//     the active tag (oscB_kc_expectKindEqual).
//   - Zig's `defer p.deinit()` inside a loop body runs at the end of each
//     iteration; Go's defer runs when the test function returns.
//   - `inline for` over cases is a loop over a slice.
//   - `"x" ** n` is strings.Repeat.

package terminal

import (
	"strings"
	"testing"
)

// oscB_kc_red is Zig's term51_red.
var oscB_kc_red = ColorRGB{R: 255, G: 0, B: 0}

// oscB_kc_unwrap is Zig's `ptr.?.*` on the result of parse or end.
func oscB_kc_unwrap(t *testing.T, c *OSCCommand) OSCCommand {
	t.Helper()
	if c == nil {
		t.Fatalf("unwrapped a null optional")
	}
	return *c
}

// oscB_kc_term51Feed is Zig's term51Feed.
func oscB_kc_term51Feed(p *OSCParser, input string) {
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
}

// oscB_kc_query is the safety-checked union access `item.query`.
func oscB_kc_query(t *testing.T, item KittyColorOSCRequest) KittyColorKind {
	t.Helper()
	if item.Tag != KittyColorOSCRequestQuery {
		t.Fatalf("access of union field query while field %v is active", item.Tag)
	}
	return item.Query
}

// oscB_kc_set is the safety-checked union access `item.set`.
func oscB_kc_set(t *testing.T, item KittyColorOSCRequest) KittyColorOSCRequestSetPayload {
	t.Helper()
	if item.Tag != KittyColorOSCRequestSet {
		t.Fatalf("access of union field set while field %v is active", item.Tag)
	}
	return item.Set
}

// oscB_kc_reset is the safety-checked union access `item.reset`.
func oscB_kc_reset(t *testing.T, item KittyColorOSCRequest) KittyColorKind {
	t.Helper()
	if item.Tag != KittyColorOSCRequestReset {
		t.Fatalf("access of union field reset while field %v is active", item.Tag)
	}
	return item.Reset
}

// oscB_kc_expectKindEqual is Zig's `try testing.expectEqual(expected, actual)`
// on two Kind unions: the tags and the payloads of the active tag are
// compared.
func oscB_kc_expectKindEqual(t *testing.T, want KittyColorKind, got KittyColorKind) {
	t.Helper()
	if want.Tag != got.Tag {
		t.Fatalf("got tag %v, want tag %v", got.Tag, want.Tag)
	}
	switch want.Tag {
	case KittyColorKindPalette:
		expectEqual(t, want.Palette, got.Palette)
	case KittyColorKindSpecial:
		expectEqual(t, want.Special, got.Special)
	}
}

// oscB_kc_special is Zig's `Kind{ .special = s }`.
func oscB_kc_special(s KittyColorSpecial) KittyColorKind {
	return KittyColorKind{Tag: KittyColorKindSpecial, Special: s}
}

// oscB_kc_palette is Zig's `Kind{ .palette = n }`.
func oscB_kc_palette(n uint8) KittyColorKind {
	return KittyColorKind{Tag: KittyColorKindPalette, Palette: n}
}

const oscB_kc_longInput = "21;foreground=?;background=rgb:f0/f8/ff;cursor=aliceblue;cursor_text;visual_bell=;selection_foreground=#xxxyyzz;selection_background=?;selection_background=#aabbcc;2=?;3=rgbi:1.0/1.0/1.0"

// Zig: test "OSC 21: kitty color protocol"
func TestOSCParsersKittyColor_OSC_21_kitty_color_protocol(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()

	input := oscB_kc_longInput
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}

	cmd := oscB_kc_unwrap(t, p.End(Some[uint8](0x1b)))
	expect(t, cmd.Tag == OSCCommandKeyKittyColorProtocol, "cmd == .kitty_color_protocol")
	expectEqual(t, 9, len(cmd.KittyColorProtocol.List))
	{
		item := cmd.KittyColorProtocol.List[0]
		expect(t, item.Tag == KittyColorOSCRequestQuery, "item == .query")
		oscB_kc_expectKindEqual(t, oscB_kc_special(KittyColorSpecialForeground), oscB_kc_query(t, item))
	}
	{
		item := cmd.KittyColorProtocol.List[1]
		expect(t, item.Tag == KittyColorOSCRequestSet, "item == .set")
		set := oscB_kc_set(t, item)
		oscB_kc_expectKindEqual(t, oscB_kc_special(KittyColorSpecialBackground), set.Key)
		expectEqual(t, uint8(0xf0), set.Color.R)
		expectEqual(t, uint8(0xf8), set.Color.G)
		expectEqual(t, uint8(0xff), set.Color.B)
	}
	{
		item := cmd.KittyColorProtocol.List[2]
		expect(t, item.Tag == KittyColorOSCRequestSet, "item == .set")
		set := oscB_kc_set(t, item)
		oscB_kc_expectKindEqual(t, oscB_kc_special(KittyColorSpecialCursor), set.Key)
		expectEqual(t, uint8(0xf0), set.Color.R)
		expectEqual(t, uint8(0xf8), set.Color.G)
		expectEqual(t, uint8(0xff), set.Color.B)
	}
	{
		item := cmd.KittyColorProtocol.List[3]
		expect(t, item.Tag == KittyColorOSCRequestReset, "item == .reset")
		oscB_kc_expectKindEqual(t, oscB_kc_special(KittyColorSpecialCursorText), oscB_kc_reset(t, item))
	}
	{
		item := cmd.KittyColorProtocol.List[4]
		expect(t, item.Tag == KittyColorOSCRequestReset, "item == .reset")
		oscB_kc_expectKindEqual(t, oscB_kc_special(KittyColorSpecialVisualBell), oscB_kc_reset(t, item))
	}
	{
		item := cmd.KittyColorProtocol.List[5]
		expect(t, item.Tag == KittyColorOSCRequestQuery, "item == .query")
		oscB_kc_expectKindEqual(t, oscB_kc_special(KittyColorSpecialSelectionBackground), oscB_kc_query(t, item))
	}
	{
		item := cmd.KittyColorProtocol.List[6]
		expect(t, item.Tag == KittyColorOSCRequestSet, "item == .set")
		set := oscB_kc_set(t, item)
		oscB_kc_expectKindEqual(t, oscB_kc_special(KittyColorSpecialSelectionBackground), set.Key)
		expectEqual(t, uint8(0xaa), set.Color.R)
		expectEqual(t, uint8(0xbb), set.Color.G)
		expectEqual(t, uint8(0xcc), set.Color.B)
	}
	{
		item := cmd.KittyColorProtocol.List[7]
		expect(t, item.Tag == KittyColorOSCRequestQuery, "item == .query")
		oscB_kc_expectKindEqual(t, oscB_kc_palette(2), oscB_kc_query(t, item))
	}
	{
		item := cmd.KittyColorProtocol.List[8]
		expect(t, item.Tag == KittyColorOSCRequestSet, "item == .set")
		set := oscB_kc_set(t, item)
		oscB_kc_expectKindEqual(t, oscB_kc_palette(3), set.Key)
		expectEqual(t, uint8(0xff), set.Color.R)
		expectEqual(t, uint8(0xff), set.Color.G)
		expectEqual(t, uint8(0xff), set.Color.B)
	}
}

// Zig: test "OSC 21: kitty color protocol without allocator"
func TestOSCParsersKittyColor_OSC_21_kitty_color_protocol_without_allocator(t *testing.T) {
	p := OSCParserInit(false)
	defer p.Deinit()

	input := "21;foreground=?"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}
	expect(t, p.End(Some[uint8](0x1b)) == nil, "p.end('\\x1b') == null")
}

// Zig: test "OSC 21: kitty color protocol double reset"
func TestOSCParsersKittyColor_OSC_21_kitty_color_protocol_double_reset(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()

	input := oscB_kc_longInput
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}

	cmd := oscB_kc_unwrap(t, p.End(Some[uint8](0x1b)))
	expect(t, cmd.Tag == OSCCommandKeyKittyColorProtocol, "cmd == .kitty_color_protocol")

	p.Reset()
	p.Reset()
}

// Zig: test "OSC 21: kitty color protocol reset after invalid"
func TestOSCParsersKittyColor_OSC_21_kitty_color_protocol_reset_after_invalid(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()

	input := oscB_kc_longInput
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}

	cmd := oscB_kc_unwrap(t, p.End(Some[uint8](0x1b)))
	expect(t, cmd.Tag == OSCCommandKeyKittyColorProtocol, "cmd == .kitty_color_protocol")

	p.Reset()

	expectEqual(t, OSCParserStateStart, p.state)
	p.Next('X')
	expectEqual(t, OSCParserStateInvalid, p.state)

	p.Reset()
}

// Zig: test "OSC 21: kitty color protocol no key"
func TestOSCParsersKittyColor_OSC_21_kitty_color_protocol_no_key(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()

	input := "21;"
	for _, ch := range []uint8(input) {
		p.Next(ch)
	}

	cmd := oscB_kc_unwrap(t, p.End(Some[uint8](0x1b)))
	expect(t, cmd.Tag == OSCCommandKeyKittyColorProtocol, "cmd == .kitty_color_protocol")
	expectEqual(t, 0, len(cmd.KittyColorProtocol.List))
}

// Zig: test "term51: parse: no allocator is invalid"
func TestOSCParsersKittyColor_term51_parse_no_allocator_is_invalid(t *testing.T) {
	p := OSCParserInit(false)
	oscB_kc_term51Feed(&p, "21")
	expect(t, p.state == OSCParserState21, "p.state == .@\"21\"")
	expect(t, OSCParsersKittyColorParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
	expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	expect(t, p.command.Tag == OSCCommandKeyInvalid, "p.command == .invalid")
}

// Zig: test "term51: parse: missing separator after 21 is invalid"
func TestOSCParsersKittyColor_term51_parse_missing_separator_after_21_is_invalid(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	oscB_kc_term51Feed(&p, "21")
	expect(t, p.state == OSCParserState21, "p.state == .@\"21\"")
	expect(t, OSCParsersKittyColorParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
	expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	expect(t, p.command.Tag == OSCCommandKeyInvalid, "p.command == .invalid")
}

// Zig: test "term51: parse: empty body yields an empty list"
func TestOSCParsersKittyColor_term51_parse_empty_body_yields_an_empty_list(t *testing.T) {
	for _, input := range []string{"21;", "21;;", "21;;;;"} {
		p := OSCParserInit(true)
		defer p.Deinit()
		oscB_kc_term51Feed(&p, input)
		cmd := oscB_kc_unwrap(t, OSCParsersKittyColorParse(&p, None[uint8]()))
		expect(t, cmd.Tag == OSCCommandKeyKittyColorProtocol, "cmd == .kitty_color_protocol")
		expectEqual(t, 0, len(cmd.KittyColorProtocol.List))
	}
}

// Zig: test "term51: parse: terminator is recorded"
func TestOSCParsersKittyColor_term51_parse_terminator_is_recorded(t *testing.T) {
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
	} {
		p := OSCParserInit(true)
		defer p.Deinit()
		oscB_kc_term51Feed(&p, "21;foreground=?")
		cmd := oscB_kc_unwrap(t, OSCParsersKittyColorParse(&p, c.term))
		expect(t, cmd.KittyColorProtocol.Terminator == c.want, "cmd.kitty_color_protocol.terminator == case[1]")
		expectEqual(t, 1, len(cmd.KittyColorProtocol.List))
	}
}

// Zig: test "term51: parse: query, set and reset"
func TestOSCParsersKittyColor_term51_parse_query_set_and_reset(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	oscB_kc_term51Feed(&p, "21;foreground=?;background=red;cursor=;cursor_text")
	cmd := oscB_kc_unwrap(t, OSCParsersKittyColorParse(&p, None[uint8]()))
	items := cmd.KittyColorProtocol.List
	expectEqual(t, 4, len(items))
	expect(t, items[0].Tag == KittyColorOSCRequestQuery, "items[0] == .query")
	oscB_kc_expectKindEqual(t, oscB_kc_special(KittyColorSpecialForeground), oscB_kc_query(t, items[0]))
	expect(t, items[1].Tag == KittyColorOSCRequestSet, "items[1] == .set")
	oscB_kc_expectKindEqual(t, oscB_kc_special(KittyColorSpecialBackground), oscB_kc_set(t, items[1]).Key)
	expectEqual(t, oscB_kc_red, oscB_kc_set(t, items[1]).Color)
	// "key=" and a bare "key" are both resets.
	expect(t, items[2].Tag == KittyColorOSCRequestReset, "items[2] == .reset")
	oscB_kc_expectKindEqual(t, oscB_kc_special(KittyColorSpecialCursor), oscB_kc_reset(t, items[2]))
	expect(t, items[3].Tag == KittyColorOSCRequestReset, "items[3] == .reset")
	oscB_kc_expectKindEqual(t, oscB_kc_special(KittyColorSpecialCursorText), oscB_kc_reset(t, items[3]))
}

// Zig: test "term51: parse: every special key"
func TestOSCParsersKittyColor_term51_parse_every_special_key(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	oscB_kc_term51Feed(&p, "21;foreground=?;background=?;selection_foreground=?;selection_background=?;cursor=?;cursor_text=?;visual_bell=?;second_transparent_background=?")
	cmd := oscB_kc_unwrap(t, OSCParsersKittyColorParse(&p, None[uint8]()))
	items := cmd.KittyColorProtocol.List
	expectEqual(t, 8, len(items))
	oscB_kc_expectKindEqual(t, oscB_kc_special(KittyColorSpecialForeground), oscB_kc_query(t, items[0]))
	oscB_kc_expectKindEqual(t, oscB_kc_special(KittyColorSpecialBackground), oscB_kc_query(t, items[1]))
	oscB_kc_expectKindEqual(t, oscB_kc_special(KittyColorSpecialSelectionForeground), oscB_kc_query(t, items[2]))
	oscB_kc_expectKindEqual(t, oscB_kc_special(KittyColorSpecialSelectionBackground), oscB_kc_query(t, items[3]))
	oscB_kc_expectKindEqual(t, oscB_kc_special(KittyColorSpecialCursor), oscB_kc_query(t, items[4]))
	oscB_kc_expectKindEqual(t, oscB_kc_special(KittyColorSpecialCursorText), oscB_kc_query(t, items[5]))
	oscB_kc_expectKindEqual(t, oscB_kc_special(KittyColorSpecialVisualBell), oscB_kc_query(t, items[6]))
	oscB_kc_expectKindEqual(t, oscB_kc_special(KittyColorSpecialSecondTransparentBackground), oscB_kc_query(t, items[7]))
}

// Zig: test "term51: parse: palette key boundaries"
func TestOSCParsersKittyColor_term51_parse_palette_key_boundaries(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	// 256, -1, +1 and the non-numeric keys are skipped. "1_0" is accepted
	// as 10 and "007" as 7 because the key is parsed with
	// std.fmt.parseUnsigned.
	oscB_kc_term51Feed(&p, "21;0=?;255=?;256=?;-1=?;+1=?;1_0=?;007=?;0x1=?;1.0=?;_1=?;1_=?; 1=?;1 =?")
	cmd := oscB_kc_unwrap(t, OSCParsersKittyColorParse(&p, None[uint8]()))
	items := cmd.KittyColorProtocol.List
	expectEqual(t, 4, len(items))
	oscB_kc_expectKindEqual(t, oscB_kc_palette(0), oscB_kc_query(t, items[0]))
	oscB_kc_expectKindEqual(t, oscB_kc_palette(255), oscB_kc_query(t, items[1]))
	oscB_kc_expectKindEqual(t, oscB_kc_palette(10), oscB_kc_query(t, items[2]))
	oscB_kc_expectKindEqual(t, oscB_kc_palette(7), oscB_kc_query(t, items[3]))
}

// Zig: test "term51: parse: unknown, empty and differently cased keys are skipped"
func TestOSCParsersKittyColor_term51_parse_unknown_empty_and_differently_cased_keys_are_skipped(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	oscB_kc_term51Feed(&p, "21;bogus=?;=?;Foreground=?;FOREGROUND=?; foreground=?;foreground =?;;cursor=?")
	cmd := oscB_kc_unwrap(t, OSCParsersKittyColorParse(&p, None[uint8]()))
	items := cmd.KittyColorProtocol.List
	expectEqual(t, 1, len(items))
	oscB_kc_expectKindEqual(t, oscB_kc_special(KittyColorSpecialCursor), oscB_kc_query(t, items[0]))
}

// Zig: test "term51: parse: only spaces are trimmed from values"
func TestOSCParsersKittyColor_term51_parse_only_spaces_are_trimmed_from_values(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	oscB_kc_term51Feed(&p, "21;foreground=  ?  ;background=   ;cursor=  red  ;cursor_text=\t?;visual_bell=\t")
	cmd := oscB_kc_unwrap(t, OSCParsersKittyColorParse(&p, None[uint8]()))
	items := cmd.KittyColorProtocol.List
	// A tab is not trimmed, so "\t?" and "\t" are invalid colors and are
	// skipped.
	expectEqual(t, 3, len(items))
	expect(t, items[0].Tag == KittyColorOSCRequestQuery, "items[0] == .query")
	oscB_kc_expectKindEqual(t, oscB_kc_special(KittyColorSpecialForeground), oscB_kc_query(t, items[0]))
	expect(t, items[1].Tag == KittyColorOSCRequestReset, "items[1] == .reset")
	oscB_kc_expectKindEqual(t, oscB_kc_special(KittyColorSpecialBackground), oscB_kc_reset(t, items[1]))
	expect(t, items[2].Tag == KittyColorOSCRequestSet, "items[2] == .set")
	oscB_kc_expectKindEqual(t, oscB_kc_special(KittyColorSpecialCursor), oscB_kc_set(t, items[2]).Key)
	expectEqual(t, oscB_kc_red, oscB_kc_set(t, items[2]).Color)
}

// Zig: test "term51: parse: invalid colors are skipped and later entries still parsed"
func TestOSCParsersKittyColor_term51_parse_invalid_colors_are_skipped_and_later_entries_still_parsed(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	// The value is everything after the first '=', so "red=blue" and "??"
	// are invalid colors.
	oscB_kc_term51Feed(&p, "21;foreground=bogus;background=red=blue;cursor=??;1=#ff0000;2=rgb:ff/00/00")
	cmd := oscB_kc_unwrap(t, OSCParsersKittyColorParse(&p, None[uint8]()))
	items := cmd.KittyColorProtocol.List
	expectEqual(t, 2, len(items))
	oscB_kc_expectKindEqual(t, oscB_kc_palette(1), oscB_kc_set(t, items[0]).Key)
	expectEqual(t, oscB_kc_red, oscB_kc_set(t, items[0]).Color)
	oscB_kc_expectKindEqual(t, oscB_kc_palette(2), oscB_kc_set(t, items[1]).Key)
	expectEqual(t, oscB_kc_red, oscB_kc_set(t, items[1]).Color)
}

// Zig: test "term51: parse: repeated keys are all kept in order"
func TestOSCParsersKittyColor_term51_parse_repeated_keys_are_all_kept_in_order(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	oscB_kc_term51Feed(&p, "21;1=?;1=red;1=;1=?")
	cmd := oscB_kc_unwrap(t, OSCParsersKittyColorParse(&p, None[uint8]()))
	items := cmd.KittyColorProtocol.List
	expectEqual(t, 4, len(items))
	oscB_kc_expectKindEqual(t, oscB_kc_palette(1), oscB_kc_query(t, items[0]))
	oscB_kc_expectKindEqual(t, oscB_kc_palette(1), oscB_kc_set(t, items[1]).Key)
	oscB_kc_expectKindEqual(t, oscB_kc_palette(1), oscB_kc_reset(t, items[2]))
	oscB_kc_expectKindEqual(t, oscB_kc_palette(1), oscB_kc_query(t, items[3]))
}

// Zig: test "term51: parse: limit on the number of requests"
func TestOSCParsersKittyColor_term51_parse_limit_on_the_number_of_requests(t *testing.T) {
	// The limit is twice the number of distinct keys: (255 + 8) * 2.
	const limit = 526
	expectEqual(t, limit, KittyColorKindMax*2)

	{
		// Exactly at the limit.
		p := OSCParserInit(true)
		defer p.Deinit()
		oscB_kc_term51Feed(&p, "21;"+strings.Repeat("0;", limit-1)+"0")
		cmd := oscB_kc_unwrap(t, OSCParsersKittyColorParse(&p, None[uint8]()))
		expectEqual(t, limit, len(cmd.KittyColorProtocol.List))
		oscB_kc_expectKindEqual(t, oscB_kc_palette(0), oscB_kc_reset(t, cmd.KittyColorProtocol.List[limit-1]))
	}
	{
		// One more request.
		p := OSCParserInit(true)
		defer p.Deinit()
		oscB_kc_term51Feed(&p, "21;"+strings.Repeat("0;", limit)+"0")
		expect(t, OSCParsersKittyColorParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
	{
		// The limit is checked before the entry is looked at, so even an
		// empty trailing entry (a trailing ';') after a full list is fatal.
		p := OSCParserInit(true)
		defer p.Deinit()
		oscB_kc_term51Feed(&p, "21;"+strings.Repeat("0;", limit))
		expect(t, OSCParsersKittyColorParse(&p, None[uint8]()) == nil, "parse(&p, null) == null")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
	}
	{
		// Skipped entries do not count towards the limit.
		p := OSCParserInit(true)
		defer p.Deinit()
		oscB_kc_term51Feed(&p, "21;"+strings.Repeat("x;", 400)+strings.Repeat("0;", limit-1)+"0")
		cmd := oscB_kc_unwrap(t, OSCParsersKittyColorParse(&p, None[uint8]()))
		expectEqual(t, limit, len(cmd.KittyColorProtocol.List))
	}
}

// Zig: test "term51: parse: buffer limit"
func TestOSCParsersKittyColor_term51_parse_buffer_limit(t *testing.T) {
	// No NUL terminator is needed, so the body may fill the whole buffer.
	// An allocator is required but does not lift the limit.
	body := strings.Repeat("1=?;", OSCParserMAX_BUF/4)
	expectEqual(t, OSCParserMAX_BUF, len(body))
	{
		p := OSCParserInit(true)
		defer p.Deinit()
		oscB_kc_term51Feed(&p, "21;"+body)
		expect(t, p.state == OSCParserState21, "p.state == .@\"21\"")
		cmd := oscB_kc_unwrap(t, OSCParsersKittyColorParse(&p, None[uint8]()))
		expectEqual(t, OSCParserMAX_BUF/4, len(cmd.KittyColorProtocol.List))
	}
	{
		p := OSCParserInit(true)
		defer p.Deinit()
		oscB_kc_term51Feed(&p, "21;"+body+"1")
		expect(t, p.state == OSCParserStateInvalid, "p.state == .invalid")
		expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
	}
}
