// Tests translated one for one from src/terminal/osc.zig (Ghostty v1.3.1 plus
// the term51 additional tests). Each Go test carries the exact Zig title in a
// "// Zig: test" comment. The helpers expect, expectEqual,
// expectEqualStrings and zigUnwrap are defined in Parser_test.go.
//
// Translation notes:
//   - `var p: Parser = .init(null)` is OSCParserInit(false) and
//     `.init(testing.allocator)` is OSCParserInit(true), a non-nil
//     value (docs/mapping.md: the field `alloc` keeps only whether an
//     allocator was provided). Go has no leak check; the lines that exist
//     only for it are kept as calls and marked "Zig-only".
//   - `p.end(x).?` is a nil check on the returned *OSCCommand; `cmd.* ==
//     .tag` is `cmd.Tag == OSCCommandKey<Tag>`; `p.fixed != null` is
//     `p.fixed.Valid`; `p.writer == null` is `p.writer == nil`.
//   - `w.buffered()` of the parser's writer is osc0_buffered(w): the wireframe's
//     oscWriter has the fields buffer and end only, and buffered() is
//     buffer[:end].
//   - Zig's `defer p.deinit()` inside a loop body runs at the end of each
//     iteration; Go's defer runs when the test function returns.

package terminal

import (
	"testing"
)

// osc0_allocator stands in for std.testing.allocator: any non-nil value.
var osc0_allocator any = struct{}{}

// osc0_feed is term51Feed of osc.zig: feed a string to an OSC parser.
func osc0_feed(p *OSCParser, bytes string) {
	for _, c := range []uint8(bytes) {
		p.Next(c)
	}
}

// osc0_buffered is Zig's `writer.buffered()`: buffer[:end].
func osc0_buffered(w *oscWriter) []uint8 {
	return w.buffer[:w.end]
}

// osc0_writer is Zig's `p.writer.?`: the active writer, failing if it is null.
func osc0_writer(t *testing.T, p *OSCParser) *oscWriter {
	t.Helper()
	if p.writer == nil {
		t.Fatalf("unwrapped a null optional")
	}
	return p.writer
}

// Not translated (Zig: test "ghostty.h Command.ProgressReport.State"): see scripts/not-translated.txt.

// Not translated (Zig: test (unnamed, osc.zig:773)): see scripts/not-translated.txt.

// ---------------------------------------------------------------------------
// term51: additional tests. Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

// Zig: test "term51: Terminator.init: only BEL selects bel"
func TestOSC_term51_Terminator_init_only_BEL_selects_bel(t *testing.T) {
	expectEqual(t, OSCTerminatorSt, OSCTerminatorInit(None[uint8]()))
	expectEqual(t, OSCTerminatorBel, OSCTerminatorInit(Some[uint8](0x07)))
	// every other byte, including ESC and the 8-bit ST, is st
	for c := 0; c < 256; c++ {
		if c == 0x07 {
			continue
		}
		expectEqual(t, OSCTerminatorSt, OSCTerminatorInit(Some(uint8(c))))
	}
}

// Zig: test "term51: Terminator.string: the bytes to end a reply with"
func TestOSC_term51_Terminator_string_the_bytes_to_end_a_reply_with(t *testing.T) {
	expectEqualStrings(t, "\x1b\\", string(OSCTerminatorSt.String()))
	expectEqualStrings(t, "\x07", string(OSCTerminatorBel.String()))
}

// Not translated (Zig: test "term51: Terminator.cval: st is 0 and bel is 1"): see scripts/not-translated.txt.

// Zig: test "term51: Terminator.format: writes the terminator bytes"
func TestOSC_term51_Terminator_format_writes_the_terminator_bytes(t *testing.T) {
	// Zig-only: var buf: [8]u8 = undefined;
	{
		// Zig-only: var w: std.Io.Writer = .fixed(&buf);
		// Zig-only: try Terminator.st.format("", .{}, &w);
		expectEqualStrings(t, "\x1b\\", OSCTerminatorSt.Format())
	}
	{
		// Zig-only: var w: std.Io.Writer = .fixed(&buf);
		// Zig-only: try Terminator.bel.format("", .{}, &w);
		expectEqualStrings(t, "\x07", OSCTerminatorBel.Format())
	}
}

// Not translated (Zig: test "term51: Command.ProgressReport.cval: state number and clamped progress"): see scripts/not-translated.txt.

// Zig: test "term51: Parser.init: starts empty, with or without an allocator"
func TestOSC_term51_Parser_init_starts_empty_with_or_without_an_allocator(t *testing.T) {
	{
		p := OSCParserInit(false)
		defer p.Deinit()
		expect(t, !p.alloc, "p.alloc == null")
		expectEqual(t, OSCParserStateStart, p.state)
		expect(t, !p.fixed.Valid, "p.fixed == null")
		expect(t, !p.allocating.Valid, "p.allocating == null")
		expect(t, p.writer == nil, "p.writer == null")
		expect(t, p.command.Tag == OSCCommandKeyInvalid, "p.command == .invalid")
	}
	{
		p := OSCParserInit(true)
		defer p.Deinit()
		expect(t, p.alloc, "p.alloc != null")
		expectEqual(t, OSCParserStateStart, p.state)
		expect(t, p.writer == nil, "p.writer == null")
	}
}

// Zig: test "term51: Parser.init: buffer size"
func TestOSC_term51_Parser_init_buffer_size(t *testing.T) {
	expectEqual(t, 2048, OSCParserMAX_BUF)
	p := OSCParserInit(false)
	expectEqual(t, OSCParserMAX_BUF, len(p.buffer))
}

// Zig: test "term51: Parser.deinit: frees an allocating buffer that was never ended"
func TestOSC_term51_Parser_deinit_frees_an_allocating_buffer_that_was_never_ended(t *testing.T) {
	p := OSCParserInit(true)
	osc0_feed(&p, "52;c;")
	for i := 0; i < 5000; i++ {
		p.Next('A')
	}
	expect(t, p.allocating.Valid, "p.allocating != null")
	// Zig-only: testing.allocator reports a leak if this does not free it
	p.Deinit()
	expectEqual(t, OSCParserStateStart, p.state)
}

// Zig: test "term51: Parser.deinit: safe on a fresh parser and when called twice"
func TestOSC_term51_Parser_deinit_safe_on_a_fresh_parser_and_when_called_twice(t *testing.T) {
	p := OSCParserInit(false)
	p.Deinit()
	p.Deinit()
	expectEqual(t, OSCParserStateStart, p.state)
}

// Zig: test "term51: Parser.reset: returns to start and drops the writers and command"
func TestOSC_term51_Parser_reset_returns_to_start_and_drops_the_writers_and_command(t *testing.T) {
	p := OSCParserInit(false)
	defer p.Deinit()
	osc0_feed(&p, "2;title")
	expect(t, p.writer != nil, "p.writer != null")
	expect(t, p.fixed.Valid, "p.fixed != null")
	expect(t, p.End(None[uint8]()) != nil, "p.end(null).?")
	expect(t, p.command.Tag == OSCCommandKeyChangeWindowTitle, "p.command == .change_window_title")

	p.Reset()
	expectEqual(t, OSCParserStateStart, p.state)
	expect(t, !p.fixed.Valid, "p.fixed == null")
	expect(t, !p.allocating.Valid, "p.allocating == null")
	expect(t, p.writer == nil, "p.writer == null")
	expect(t, p.command.Tag == OSCCommandKeyInvalid, "p.command == .invalid")
}

// Zig: test "term51: Parser.reset: leaves the invalid state"
func TestOSC_term51_Parser_reset_leaves_the_invalid_state(t *testing.T) {
	p := OSCParserInit(false)
	defer p.Deinit()
	osc0_feed(&p, "x")
	expectEqual(t, OSCParserStateInvalid, p.state)
	p.Reset()
	expectEqual(t, OSCParserStateStart, p.state)
	// and the parser is usable again
	osc0_feed(&p, "0;ok")
	cmd := p.End(None[uint8]())
	expect(t, cmd != nil, "p.end(null).?")
	expectEqualStrings(t, "ok", string(cmd.ChangeWindowTitle))
}

// Zig: test "term51: Parser.reset: frees the allocating writer"
func TestOSC_term51_Parser_reset_frees_the_allocating_writer(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	osc0_feed(&p, "52;c;AAAA")
	expect(t, p.allocating.Valid, "p.allocating != null")
	p.Reset()
	expect(t, !p.allocating.Valid, "p.allocating == null")
	expect(t, p.writer == nil, "p.writer == null")
}

// Zig: test "term51: Parser.reset: frees a kitty colour command"
func TestOSC_term51_Parser_reset_frees_a_kitty_colour_command(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	osc0_feed(&p, "21;foreground=?;background=red")
	cmd := p.End(Some[uint8](0x07))
	expect(t, cmd != nil, "p.end(0x07).?")
	expect(t, cmd.Tag == OSCCommandKeyKittyColorProtocol, "cmd.* == .kitty_color_protocol")
	// Zig-only: the command owns a list; reset must release it (leak check)
	p.Reset()
	expect(t, p.command.Tag == OSCCommandKeyInvalid, "p.command == .invalid")
}

// Zig: test "term51: Parser.ensureAllocator: true with an allocator, and the state is untouched"
func TestOSC_term51_Parser_ensureAllocator_true_with_an_allocator_and_the_state_is_untouched(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	p.state = OSCParserState4
	expect(t, p.ensureAllocator(), "p.ensureAllocator()")
	expectEqual(t, OSCParserState4, p.state)
}

// Zig: test "term51: Parser.ensureAllocator: false without one, and the parser becomes invalid"
func TestOSC_term51_Parser_ensureAllocator_false_without_one_and_the_parser_becomes_invalid(t *testing.T) {
	p := OSCParserInit(false)
	defer p.Deinit()
	p.state = OSCParserState4
	expect(t, !p.ensureAllocator(), "!p.ensureAllocator()")
	expectEqual(t, OSCParserStateInvalid, p.state)
}

// Zig: test "term51: Parser.writeToFixed: later bytes land in the fixed buffer"
func TestOSC_term51_Parser_writeToFixed_later_bytes_land_in_the_fixed_buffer(t *testing.T) {
	p := OSCParserInit(false)
	defer p.Deinit()
	p.writeToFixed()
	expect(t, p.fixed.Valid, "p.fixed != null")
	expect(t, !p.allocating.Valid, "p.allocating == null")
	expect(t, p.writer != nil, "p.writer != null")
	expect(t, &p.fixed.Value == p.writer, "@intFromPtr(&p.fixed.?) == @intFromPtr(p.writer.?)")
	p.Next('h')
	p.Next('i')
	expectEqualStrings(t, "hi", string(osc0_buffered(osc0_writer(t, &p))))
	expectEqualStrings(t, "hi", string(p.buffer[0:2]))
}

// Zig: test "term51: Parser.writeToAllocating: uses the heap when there is an allocator"
func TestOSC_term51_Parser_writeToAllocating_uses_the_heap_when_there_is_an_allocator(t *testing.T) {
	p := OSCParserInit(true)
	defer p.Deinit()
	p.writeToAllocating()
	expect(t, p.allocating.Valid, "p.allocating != null")
	expect(t, !p.fixed.Valid, "p.fixed == null")
	expect(t, &p.allocating.Value.writer == osc0_writer(t, &p), "@intFromPtr(&p.allocating.?.writer) == @intFromPtr(p.writer.?)")
	// more than the fixed buffer could hold
	for i := 0; i < OSCParserMAX_BUF*3; i++ {
		p.Next('z')
	}
	expectEqual(t, OSCParserMAX_BUF*3, len(osc0_buffered(osc0_writer(t, &p))))
	expect(t, p.state != OSCParserStateInvalid, "p.state != .invalid")
}

// Zig: test "term51: Parser.writeToAllocating: falls back to the fixed buffer without an allocator"
func TestOSC_term51_Parser_writeToAllocating_falls_back_to_the_fixed_buffer_without_an_allocator(t *testing.T) {
	p := OSCParserInit(false)
	defer p.Deinit()
	p.writeToAllocating()
	expect(t, !p.allocating.Valid, "p.allocating == null")
	expect(t, p.fixed.Valid, "p.fixed != null")
	expect(t, &p.fixed.Value == osc0_writer(t, &p), "@intFromPtr(&p.fixed.?) == @intFromPtr(p.writer.?)")
}

// Zig: test "term51: Parser.next: every number prefix reaches its state"
func TestOSC_term51_Parser_next_every_number_prefix_reaches_its_state(t *testing.T) {
	cases := []struct {
		in    string
		state OSCParserState
	}{
		{"", OSCParserStateStart}, {"0", OSCParserState0}, {"1", OSCParserState1},
		{"2", OSCParserState2}, {"3", OSCParserState3}, {"4", OSCParserState4},
		{"5", OSCParserState5}, {"6", OSCParserState6}, {"7", OSCParserState7},
		{"8", OSCParserState8}, {"9", OSCParserState9}, {"10", OSCParserState10},
		{"11", OSCParserState11}, {"12", OSCParserState12}, {"13", OSCParserState13},
		{"14", OSCParserState14}, {"15", OSCParserState15}, {"16", OSCParserState16},
		{"17", OSCParserState17}, {"18", OSCParserState18}, {"19", OSCParserState19},
		{"21", OSCParserState21}, {"22", OSCParserState22}, {"30", OSCParserState30},
		{"300", OSCParserState300}, {"3008", OSCParserState3008}, {"52", OSCParserState52},
		{"55", OSCParserState55}, {"552", OSCParserState552}, {"5522", OSCParserState5522},
		{"66", OSCParserState66}, {"77", OSCParserState77}, {"777", OSCParserState777},
		{"104", OSCParserState104}, {"110", OSCParserState110}, {"111", OSCParserState111},
		{"112", OSCParserState112}, {"113", OSCParserState113}, {"114", OSCParserState114},
		{"115", OSCParserState115}, {"116", OSCParserState116}, {"117", OSCParserState117},
		{"118", OSCParserState118}, {"119", OSCParserState119}, {"133", OSCParserState133},
		{"1337", OSCParserState1337},
	}
	for _, case_ := range cases {
		p := OSCParserInit(false)
		defer p.Deinit()
		osc0_feed(&p, case_.in)
		expectEqual(t, case_.state, p.state)
		expect(t, p.writer == nil, "p.writer == null")
	}
}

// Zig: test "term51: Parser.next: numbers upstream does not know are invalid"
func TestOSC_term51_Parser_next_numbers_upstream_does_not_know_are_invalid(t *testing.T) {
	cases := []string{
		"a", ";", " ", "00", "01", "20", "23", "29",
		"31", "301", "3000", "3009", "30080", "40", "50", "51",
		"53", "550", "553", "5520", "5523", "55220", "60", "65",
		"67", "660", "70", "76", "78", "770", "776", "778",
		"7770", "80", "90", "99", "100", "101", "105", "1040",
		"1100", "120", "130", "132", "134", "1330", "1336", "1338",
		"13370", "140", "190", "210", "220", "1x", "-1", "0x",
	}
	for _, input := range cases {
		p := OSCParserInit(true)
		defer p.Deinit()
		osc0_feed(&p, input)
		expectEqual(t, OSCParserStateInvalid, p.state)
		expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
	}
}

// Zig: test "term51: Parser.next: a semicolon starts the payload in the fixed buffer"
func TestOSC_term51_Parser_next_a_semicolon_starts_the_payload_in_the_fixed_buffer(t *testing.T) {
	// these need no allocator
	for _, num := range []string{"0", "1", "2", "7", "8", "9", "22", "133", "777", "1337", "3008"} {
		p := OSCParserInit(false)
		defer p.Deinit()
		osc0_feed(&p, num)
		before := p.state
		p.Next(';')
		// the state is unchanged; only the writer is set up
		expectEqual(t, before, p.state)
		expect(t, p.fixed.Valid, "p.fixed != null")
		expect(t, !p.allocating.Valid, "p.allocating == null")
		expect(t, p.writer != nil, "p.writer != null")
		expectEqual(t, 0, len(osc0_buffered(osc0_writer(t, &p))))
	}
}

// Zig: test "term51: Parser.next: colour sequences need an allocator"
func TestOSC_term51_Parser_next_colour_sequences_need_an_allocator(t *testing.T) {
	nums := []string{
		"4", "5", "10", "11", "12", "13", "14", "15", "16",
		"17", "18", "19", "21", "104", "110", "111", "112", "113",
		"114", "115", "116", "117", "118", "119",
	}
	for _, num := range nums {
		{
			p := OSCParserInit(false)
			defer p.Deinit()
			osc0_feed(&p, num)
			p.Next(';')
			expectEqual(t, OSCParserStateInvalid, p.state)
			expect(t, p.writer == nil, "p.writer == null")
		}
		{
			p := OSCParserInit(true)
			defer p.Deinit()
			osc0_feed(&p, num)
			before := p.state
			p.Next(';')
			expectEqual(t, before, p.state)
			// the payload still goes to the fixed buffer
			expect(t, p.fixed.Valid, "p.fixed != null")
			expect(t, !p.allocating.Valid, "p.allocating == null")
		}
	}
}

// Zig: test "term51: Parser.next: clipboard, text sizing and kitty clipboard payloads go to the heap"
func TestOSC_term51_Parser_next_clipboard_text_sizing_and_kitty_clipboard_payloads_go_to_the_heap(t *testing.T) {
	for _, num := range []string{"52", "66", "5522"} {
		{
			p := OSCParserInit(true)
			defer p.Deinit()
			osc0_feed(&p, num)
			p.Next(';')
			expect(t, p.allocating.Valid, "p.allocating != null")
			expect(t, !p.fixed.Valid, "p.fixed == null")
		}
		{
			// without an allocator they fall back to the fixed buffer
			p := OSCParserInit(false)
			defer p.Deinit()
			osc0_feed(&p, num)
			p.Next(';')
			expect(t, !p.allocating.Valid, "p.allocating == null")
			expect(t, p.fixed.Valid, "p.fixed != null")
			expect(t, p.state != OSCParserStateInvalid, "p.state != .invalid")
		}
	}
}

// Zig: test "term51: Parser.next: prefixes with no semicolon form are invalid at a semicolon"
func TestOSC_term51_Parser_next_prefixes_with_no_semicolon_form_are_invalid_at_a_semicolon(t *testing.T) {
	for _, num := range []string{"3", "30", "300", "6", "55", "552", "77"} {
		p := OSCParserInit(true)
		defer p.Deinit()
		osc0_feed(&p, num)
		p.Next(';')
		expectEqual(t, OSCParserStateInvalid, p.state)
	}
}

// Zig: test "term51: Parser.next: once a payload has started, every byte is payload"
func TestOSC_term51_Parser_next_once_a_payload_has_started_every_byte_is_payload(t *testing.T) {
	p := OSCParserInit(false)
	defer p.Deinit()
	osc0_feed(&p, "2;")
	// digits and semicolons no longer drive the state machine
	osc0_feed(&p, "1337;\x00\xff;")
	expectEqual(t, OSCParserState2, p.state)
	expectEqualStrings(t, "1337;\x00\xff;", string(osc0_buffered(osc0_writer(t, &p))))
}

// Zig: test "term51: Parser.next: the fixed buffer holds exactly MAX_BUF bytes"
func TestOSC_term51_Parser_next_the_fixed_buffer_holds_exactly_MAX_BUF_bytes(t *testing.T) {
	p := OSCParserInit(false)
	defer p.Deinit()
	osc0_feed(&p, "2;")
	for i := 0; i < OSCParserMAX_BUF; i++ {
		p.Next('x')
	}
	expectEqual(t, OSCParserState2, p.state)
	expectEqual(t, OSCParserMAX_BUF, len(osc0_buffered(osc0_writer(t, &p))))
	// one more byte does not fit
	p.Next('y')
	expectEqual(t, OSCParserStateInvalid, p.state)
}

// Zig: test "term51: Parser.next: input is ignored in the invalid state"
func TestOSC_term51_Parser_next_input_is_ignored_in_the_invalid_state(t *testing.T) {
	p := OSCParserInit(false)
	defer p.Deinit()
	osc0_feed(&p, "x")
	expectEqual(t, OSCParserStateInvalid, p.state)
	osc0_feed(&p, "2;title")
	expectEqual(t, OSCParserStateInvalid, p.state)
	expect(t, p.writer == nil, "p.writer == null")
	expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
}

// Zig: test "term51: Parser.end: nothing to end"
func TestOSC_term51_Parser_end_nothing_to_end(t *testing.T) {
	p := OSCParserInit(false)
	defer p.Deinit()
	// start
	expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
	expect(t, p.End(Some[uint8](0x07)) == nil, "p.end(0x07) == null")
}

// Zig: test "term51: Parser.end: prefixes that are not complete commands"
func TestOSC_term51_Parser_end_prefixes_that_are_not_complete_commands(t *testing.T) {
	for _, num := range []string{"3", "30", "300", "55", "552", "6", "77"} {
		p := OSCParserInit(true)
		defer p.Deinit()
		osc0_feed(&p, num)
		expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
	}
}

// Zig: test "term51: Parser.end: each number dispatches to its command"
func TestOSC_term51_Parser_end_each_number_dispatches_to_its_command(t *testing.T) {
	cases := []struct {
		in  string
		tag OSCCommandKey
	}{
		{"0;t", OSCCommandKeyChangeWindowTitle},
		{"1;t", OSCCommandKeyChangeWindowIcon},
		{"2;t", OSCCommandKeyChangeWindowTitle},
		{"4;1;red", OSCCommandKeyColorOperation},
		{"5;1;red", OSCCommandKeyColorOperation},
		{"10;red", OSCCommandKeyColorOperation},
		{"11;red", OSCCommandKeyColorOperation},
		{"12;red", OSCCommandKeyColorOperation},
		{"19;red", OSCCommandKeyColorOperation},
		{"104;1", OSCCommandKeyColorOperation},
		{"110;", OSCCommandKeyColorOperation},
		{"119;", OSCCommandKeyColorOperation},
		{"7;file:///tmp", OSCCommandKeyReportPwd},
		{"8;;http://a", OSCCommandKeyHyperlinkStart},
		{"8;;", OSCCommandKeyHyperlinkEnd},
		{"9;hello", OSCCommandKeyShowDesktopNotification},
		{"21;foreground=?", OSCCommandKeyKittyColorProtocol},
		{"22;pointer", OSCCommandKeyMouseShape},
		{"52;c;aGk=", OSCCommandKeyClipboardContents},
		{"133;A", OSCCommandKeySemanticPrompt},
		{"777;notify;a;b", OSCCommandKeyShowDesktopNotification},
	}
	for _, case_ := range cases {
		p := OSCParserInit(true)
		defer p.Deinit()
		osc0_feed(&p, case_.in)
		cmd := p.End(Some[uint8](0x07))
		if cmd == nil {
			t.Fatalf("no command for %s", case_.in)
		}
		expectEqual(t, case_.tag, cmd.Tag)
		// the returned pointer is the parser's own command slot
		expect(t, &p.command == cmd, "@intFromPtr(&p.command) == @intFromPtr(cmd)")
	}
}

// Zig: test "term51: Parser.end: a number without a payload has no writer and yields nothing"
func TestOSC_term51_Parser_end_a_number_without_a_payload_has_no_writer_and_yields_nothing(t *testing.T) {
	for _, num := range []string{"0", "1", "2", "7", "8", "9", "22", "133", "777", "1337"} {
		p := OSCParserInit(true)
		defer p.Deinit()
		osc0_feed(&p, num)
		expect(t, p.writer == nil, "p.writer == null")
		expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
	}
}

// Zig: test "term51: Parser.end: title length boundary of the fixed buffer"
func TestOSC_term51_Parser_end_title_length_boundary_of_the_fixed_buffer(t *testing.T) {
	// The title parser appends a NUL, so the longest title is MAX_BUF - 1.
	{
		p := OSCParserInit(false)
		defer p.Deinit()
		osc0_feed(&p, "2;")
		for i := 0; i < OSCParserMAX_BUF-1; i++ {
			p.Next('x')
		}
		cmd := p.End(None[uint8]())
		expect(t, cmd != nil, "p.end(null).?")
		expectEqual(t, OSCParserMAX_BUF-1, len(cmd.ChangeWindowTitle))
	}
	{
		p := OSCParserInit(false)
		defer p.Deinit()
		osc0_feed(&p, "2;")
		for i := 0; i < OSCParserMAX_BUF; i++ {
			p.Next('x')
		}
		expect(t, p.End(None[uint8]()) == nil, "p.end(null) == null")
		expectEqual(t, OSCParserStateInvalid, p.state)
	}
}

// Zig: test "term51: Parser.end: an empty title is a command with an empty string"
func TestOSC_term51_Parser_end_an_empty_title_is_a_command_with_an_empty_string(t *testing.T) {
	p := OSCParserInit(false)
	defer p.Deinit()
	osc0_feed(&p, "2;")
	cmd := p.End(None[uint8]())
	expect(t, cmd != nil, "p.end(null).?")
	expect(t, cmd.Tag == OSCCommandKeyChangeWindowTitle, "cmd.* == .change_window_title")
	expectEqualStrings(t, "", string(cmd.ChangeWindowTitle))
}

// Zig: test "term51: Parser.end: the terminator is recorded by commands that reply"
func TestOSC_term51_Parser_end_the_terminator_is_recorded_by_commands_that_reply(t *testing.T) {
	{
		p := OSCParserInit(true)
		defer p.Deinit()
		osc0_feed(&p, "4;1;?")
		cmd := p.End(Some[uint8](0x07))
		expect(t, cmd != nil, "p.end(0x07).?")
		expectEqual(t, OSCTerminatorBel, cmd.ColorOperation.Terminator)
	}
	{
		p := OSCParserInit(true)
		defer p.Deinit()
		osc0_feed(&p, "4;1;?")
		cmd := p.End(Some[uint8](0x1b))
		expect(t, cmd != nil, "p.end(0x1b).?")
		expectEqual(t, OSCTerminatorSt, cmd.ColorOperation.Terminator)
	}
	{
		p := OSCParserInit(true)
		defer p.Deinit()
		osc0_feed(&p, "4;1;?")
		cmd := p.End(None[uint8]())
		expect(t, cmd != nil, "p.end(null).?")
		expectEqual(t, OSCTerminatorSt, cmd.ColorOperation.Terminator)
	}
}
