// Tests translated one for one from src/terminal/Parser.zig that expect an
// osc_dispatch action, or drive osc_parser and expect a command back. They
// are in the same order as in Parser.zig. They fail with the not-implemented
// panic until osc.zig is ported.
//
// The helpers (expect, expectEqual, zigUnwrap, zigPayload, term51Feed, ...)
// are in Parser_test.go.

package terminal

import (
	"testing"
)

// Zig: test "osc: change window title"
func TestParser_osc_change_window_title(t *testing.T) {
	p := ParserInit()
	_ = p.Next(0x1B)
	_ = p.Next(']')
	_ = p.Next('0')
	_ = p.Next(';')
	_ = p.Next('a')
	_ = p.Next('b')
	_ = p.Next('c')

	{
		a := p.Next(0x07) // BEL
		expect(t, p.state == ParserStateGround, "p.state == .ground")
		expect(t, zigUnwrap(t, a[0]).Tag == ParserActionOscDispatch, "a[0].? == .osc_dispatch")
		expect(t, !a[1].Valid, "a[1] == null")
		expect(t, !a[2].Valid, "a[2] == null")

		cmd := zigPayload(t, a[0], ParserActionOscDispatch).OscDispatch
		expect(t, cmd.Tag == OSCCommandKeyChangeWindowTitle, "cmd == .change_window_title")
		expectEqualStrings(t, "abc", string(cmd.ChangeWindowTitle))
	}
}

// Zig: test "osc: change window title (end in esc)"
func TestParser_osc_change_window_title_end_in_esc(t *testing.T) {
	p := ParserInit()
	_ = p.Next(0x1B)
	_ = p.Next(']')
	_ = p.Next('0')
	_ = p.Next(';')
	_ = p.Next('a')
	_ = p.Next('b')
	_ = p.Next('c')

	{
		a := p.Next(0x1B)
		_ = p.Next('\\')
		expect(t, p.state == ParserStateGround, "p.state == .ground")
		expect(t, zigUnwrap(t, a[0]).Tag == ParserActionOscDispatch, "a[0].? == .osc_dispatch")
		expect(t, !a[1].Valid, "a[1] == null")
		expect(t, !a[2].Valid, "a[2] == null")

		cmd := zigPayload(t, a[0], ParserActionOscDispatch).OscDispatch
		expect(t, cmd.Tag == OSCCommandKeyChangeWindowTitle, "cmd == .change_window_title")
		expectEqualStrings(t, "abc", string(cmd.ChangeWindowTitle))
	}
}

// https://github.com/darrenstarr/VtNetCore/pull/14
// Saw this on HN, decided to add a test case because why not.
//
// Zig: test "osc: 112 incomplete sequence"
func TestParser_osc_112_incomplete_sequence(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	p.osc_parser.alloc = struct{}{} // Zig: p.osc_parser.alloc = std.testing.allocator;

	_ = p.Next(0x1B)
	_ = p.Next(']')
	_ = p.Next('1')
	_ = p.Next('1')
	_ = p.Next('2')

	{
		a := p.Next(0x07)
		expect(t, p.state == ParserStateGround, "p.state == .ground")
		expect(t, zigUnwrap(t, a[0]).Tag == ParserActionOscDispatch, "a[0].? == .osc_dispatch")
		expect(t, !a[1].Valid, "a[1] == null")
		expect(t, !a[2].Valid, "a[2] == null")

		cmd := zigPayload(t, a[0], ParserActionOscDispatch).OscDispatch
		expect(t, cmd.Tag == OSCCommandKeyColorOperation, "cmd == .color_operation")
		// Zig passes these two in this order: (actual, expected).
		expectEqual(t, cmd.ColorOperation.terminator, OSCTerminatorBel)
		expect(t, cmd.ColorOperation.op == OSCParsersColorOperationOsc112, "cmd.color_operation.op == .osc_112")
		expect(t, cmd.ColorOperation.requests.count() == 1, "cmd.color_operation.requests.count() == 1")
		it := cmd.ColorOperation.requests.constIterator(0)
		{
			op := it.next()
			expect(t, op != nil, "it.next().?")
			expect(t, op.Tag == OSCParsersColorRequestReset, "op.* == .reset")
			expectEqual(t,
				OSCParsersColorRequest{Tag: OSCParsersColorRequestReset, Reset: OSCParsersColorTarget{Tag: OSCParsersColorTargetDynamic, Dynamic: ColorDynamicCursor}},
				*op,
			)
		}
		expect(t, it.next() == nil, "it.next() == null")
	}
}

// Zig: test "osc: 104 empty"
func TestParser_osc_104_empty(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	p.osc_parser.alloc = struct{}{} // Zig: p.osc_parser.alloc = std.testing.allocator;

	_ = p.Next(0x1B)
	_ = p.Next(']')
	_ = p.Next('1')
	_ = p.Next('0')
	_ = p.Next('4')

	{
		a := p.Next(0x07)
		expect(t, p.state == ParserStateGround, "p.state == .ground")
		expect(t, zigUnwrap(t, a[0]).Tag == ParserActionOscDispatch, "a[0].? == .osc_dispatch")
		expect(t, !a[1].Valid, "a[1] == null")
		expect(t, !a[2].Valid, "a[2] == null")

		cmd := zigPayload(t, a[0], ParserActionOscDispatch).OscDispatch
		expect(t, cmd.Tag == OSCCommandKeyColorOperation, "cmd == .color_operation")
		// Zig passes these two in this order: (actual, expected).
		expectEqual(t, cmd.ColorOperation.terminator, OSCTerminatorBel)
		expect(t, cmd.ColorOperation.op == OSCParsersColorOperationOsc104, "cmd.color_operation.op == .osc_104")
		expect(t, cmd.ColorOperation.requests.count() == 1, "cmd.color_operation.requests.count() == 1")
		it := cmd.ColorOperation.requests.constIterator(0)
		{
			op := it.next()
			expect(t, op != nil, "it.next().?")
			expect(t, op.Tag == OSCParsersColorRequestResetPalette, "op.* == .reset_palette")
		}
		expect(t, it.next() == nil, "it.next() == null")
	}
}

// Zig: test "term51: next: exit, transition and entry actions occupy slots 0, 1 and 2"
func TestParser_term51_next_exit_transition_and_entry_actions_occupy_slots_0_1_and_2(t *testing.T) {
	// Leaving dcs_passthrough yields dcs_unhook in slot 0.
	{
		p := ParserInit()
		defer p.Deinit()
		hook := term51Feed(&p, "\x1bPq")
		expect(t, !hook[0].Valid, "hook[0] == null")
		expect(t, !hook[1].Valid, "hook[1] == null")
		expect(t, zigUnwrap(t, hook[2]).Tag == ParserActionDcsHook, "hook[2].? == .dcs_hook")
		expectEqual(t, uint8('q'), zigPayload(t, hook[2], ParserActionDcsHook).DcsHook.Final)
		a := p.Next(0x18)
		expect(t, zigUnwrap(t, a[0]).Tag == ParserActionDcsUnhook, "a[0].? == .dcs_unhook")
		expect(t, zigUnwrap(t, a[1]).Tag == ParserActionExecute, "a[1].? == .execute")
		expect(t, !a[2].Valid, "a[2] == null")
	}
	// Entering and leaving the APC string state.
	{
		p := ParserInit()
		defer p.Deinit()
		start := term51Feed(&p, "\x1b_")
		expect(t, !start[0].Valid, "start[0] == null")
		expect(t, zigUnwrap(t, start[2]).Tag == ParserActionApcStart, "start[2].? == .apc_start")
		put := p.Next('G')
		expect(t, !put[0].Valid, "put[0] == null")
		expect(t, zigUnwrap(t, put[1]).Tag == ParserActionApcPut, "put[1].? == .apc_put")
		expectEqual(t, uint8('G'), zigPayload(t, put[1], ParserActionApcPut).ApcPut)
		expect(t, !put[2].Valid, "put[2] == null")
		end := p.Next(0x1B)
		expect(t, zigUnwrap(t, end[0]).Tag == ParserActionApcEnd, "end[0].? == .apc_end")
		expectEqual(t, ParserStateEscape, p.state)
	}
	// Leaving osc_string dispatches the OSC in slot 0.
	{
		p := ParserInit()
		defer p.Deinit()
		_ = term51Feed(&p, "\x1b]2;ab")
		a := p.Next(0x07)
		expect(t, zigUnwrap(t, a[0]).Tag == ParserActionOscDispatch, "a[0].? == .osc_dispatch")
		expect(t, zigPayload(t, a[0], ParserActionOscDispatch).OscDispatch.Tag == OSCCommandKeyChangeWindowTitle, "a[0].?.osc_dispatch == .change_window_title")
		expectEqualStrings(t, "ab", string(zigPayload(t, a[0], ParserActionOscDispatch).OscDispatch.ChangeWindowTitle))
		expectEqual(t, ParserStateGround, p.state)
	}
}

// Zig: test "term51: doAction: osc_put feeds the OSC parser"
func TestParser_term51_doAction_osc_put_feeds_the_OSC_parser(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	p.osc_parser.Reset()
	expect(t, !p.doAction(ParserTransitionActionOscPut, '2').Valid, "p.doAction(.osc_put, '2') == null")
	expect(t, !p.doAction(ParserTransitionActionOscPut, ';').Valid, "p.doAction(.osc_put, ';') == null")
	expect(t, !p.doAction(ParserTransitionActionOscPut, 'z').Valid, "p.doAction(.osc_put, 'z') == null")
	// Zig: const cmd = p.osc_parser.end(0x07).?; a null *OSCCommand is nil.
	cmd := p.osc_parser.End(Some[uint8](0x07))
	expect(t, cmd != nil, "p.osc_parser.end(0x07).?")
	expect(t, cmd.Tag == OSCCommandKeyChangeWindowTitle, "cmd.* == .change_window_title")
	expectEqualStrings(t, "z", string(cmd.ChangeWindowTitle))
}
