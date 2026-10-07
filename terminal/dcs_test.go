// Tests translated one for one from src/terminal/dcs.zig (Ghostty v1.3.1 plus
// the term51 additional tests). Each Go test carries the exact Zig title in a
// "// Zig: test" comment. The helpers expect, expectEqual,
// expectEqualStrings and zigUnwrap are defined in Parser_test.go.
//
// Translation notes:
//   - Allocator arguments (`alloc`, `testing.allocator`) have no Go
//     counterpart (docs/mapping.md), so `h.hook(alloc, dcs)` is h.Hook(dcs)
//     and the `const alloc = testing.allocator;` lines are dropped.
//   - `var h: Handler = .{}` is DCSHandlerInit(); `.{ .max_bytes = n }` sets
//     the field after init.
//   - `try x` on a call that returns an error union is zigTry on the error.
//   - `expectError(error.OutOfMemory, x)` compares the error with
//     ErrOutOfMemory.
//   - `cmd == .xtgettcap` is `cmd.tag == DCSCommandXtgettcap`, and
//     `h.state == .ignore` is `h.state.tag == DCSStateIgnore`.
//   - Zig's `defer` inside a loop body runs at the end of each iteration;
//     Go's defer runs when the test function returns.
//   - Go has no leak check; calls that only exist to be leak-checked in Zig
//     are kept.
//   - tmux control mode is compiled out of the library build, as in the Zig
//     library build the tests are written for.

package terminal

import (
	"testing"
)

// zigTry is the error half of Zig's `try`: it stops the test when the call
// returned an error.
func zigTry(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("try failed: %v", err)
	}
}

// Zig: test "unknown DCS command"
func TestDCS_unknown_DCS_command(t *testing.T) {
	h := DCSHandlerInit()
	defer h.Deinit()
	expect(t, !h.Hook(ParserActionDCS{Final: 'A'}).Valid, "h.hook(alloc, .{ .final = 'A' }) == null")
	expect(t, h.state.tag == DCSStateIgnore, "h.state == .ignore")
	expect(t, !h.Unhook().Valid, "h.unhook() == null")
	expect(t, h.state.tag == DCSStateInactive, "h.state == .inactive")
}

// Zig: test "XTGETTCAP command"
func TestDCS_XTGETTCAP_command(t *testing.T) {
	h := DCSHandlerInit()
	defer h.Deinit()
	expect(t, !h.Hook(ParserActionDCS{Intermediates: []uint8("+"), Final: 'q'}).Valid, "h.hook(alloc, .{ .intermediates = \"+\", .final = 'q' }) == null")
	for _, byte_ := range []uint8("536D756C78") {
		_ = h.Put(byte_)
	}
	cmd := zigUnwrap(t, h.Unhook())
	defer cmd.Deinit()
	expect(t, cmd.Tag == DCSCommandXtgettcap, "cmd == .xtgettcap")
	expectEqualStrings(t, "536D756C78", string(zigUnwrap(t, cmd.Xtgettcap.Next())))
	expect(t, !cmd.Xtgettcap.Next().Valid, "cmd.xtgettcap.next() == null")
}

// Zig: test "XTGETTCAP mixed case"
func TestDCS_XTGETTCAP_mixed_case(t *testing.T) {
	h := DCSHandlerInit()
	defer h.Deinit()
	expect(t, !h.Hook(ParserActionDCS{Intermediates: []uint8("+"), Final: 'q'}).Valid, "h.hook(alloc, .{ .intermediates = \"+\", .final = 'q' }) == null")
	for _, byte_ := range []uint8("536d756C78") {
		_ = h.Put(byte_)
	}
	cmd := zigUnwrap(t, h.Unhook())
	defer cmd.Deinit()
	expect(t, cmd.Tag == DCSCommandXtgettcap, "cmd == .xtgettcap")
	expectEqualStrings(t, "536D756C78", string(zigUnwrap(t, cmd.Xtgettcap.Next())))
	expect(t, !cmd.Xtgettcap.Next().Valid, "cmd.xtgettcap.next() == null")
}

// Zig: test "XTGETTCAP command multiple keys"
func TestDCS_XTGETTCAP_command_multiple_keys(t *testing.T) {
	h := DCSHandlerInit()
	defer h.Deinit()
	expect(t, !h.Hook(ParserActionDCS{Intermediates: []uint8("+"), Final: 'q'}).Valid, "h.hook(alloc, .{ .intermediates = \"+\", .final = 'q' }) == null")
	for _, byte_ := range []uint8("536D756C78;536D756C78") {
		_ = h.Put(byte_)
	}
	cmd := zigUnwrap(t, h.Unhook())
	defer cmd.Deinit()
	expect(t, cmd.Tag == DCSCommandXtgettcap, "cmd == .xtgettcap")
	expectEqualStrings(t, "536D756C78", string(zigUnwrap(t, cmd.Xtgettcap.Next())))
	expectEqualStrings(t, "536D756C78", string(zigUnwrap(t, cmd.Xtgettcap.Next())))
	expect(t, !cmd.Xtgettcap.Next().Valid, "cmd.xtgettcap.next() == null")
}

// Zig: test "XTGETTCAP command invalid data"
func TestDCS_XTGETTCAP_command_invalid_data(t *testing.T) {
	h := DCSHandlerInit()
	defer h.Deinit()
	expect(t, !h.Hook(ParserActionDCS{Intermediates: []uint8("+"), Final: 'q'}).Valid, "h.hook(alloc, .{ .intermediates = \"+\", .final = 'q' }) == null")
	for _, byte_ := range []uint8("who;536D756C78") {
		_ = h.Put(byte_)
	}
	cmd := zigUnwrap(t, h.Unhook())
	defer cmd.Deinit()
	expect(t, cmd.Tag == DCSCommandXtgettcap, "cmd == .xtgettcap")
	expectEqualStrings(t, "WHO", string(zigUnwrap(t, cmd.Xtgettcap.Next())))
	expectEqualStrings(t, "536D756C78", string(zigUnwrap(t, cmd.Xtgettcap.Next())))
	expect(t, !cmd.Xtgettcap.Next().Valid, "cmd.xtgettcap.next() == null")
}

// Zig: test "DECRQSS command"
func TestDCS_DECRQSS_command(t *testing.T) {
	h := DCSHandlerInit()
	defer h.Deinit()
	expect(t, !h.Hook(ParserActionDCS{Intermediates: []uint8("$"), Final: 'q'}).Valid, "h.hook(alloc, .{ .intermediates = \"$\", .final = 'q' }) == null")
	_ = h.Put('m')
	cmd := zigUnwrap(t, h.Unhook())
	defer cmd.Deinit()
	expect(t, cmd.Tag == DCSCommandDecrqss, "cmd == .decrqss")
	expect(t, cmd.Decrqss == DCSCommandDECRQSSSgr, "cmd.decrqss == .sgr")
}

// Zig: test "DECRQSS invalid command"
func TestDCS_DECRQSS_invalid_command(t *testing.T) {
	h := DCSHandlerInit()
	defer h.Deinit()
	expect(t, !h.Hook(ParserActionDCS{Intermediates: []uint8("$"), Final: 'q'}).Valid, "h.hook(alloc, .{ .intermediates = \"$\", .final = 'q' }) == null")
	_ = h.Put('z')
	cmd := zigUnwrap(t, h.Unhook())
	defer cmd.Deinit()
	expect(t, cmd.Tag == DCSCommandDecrqss, "cmd == .decrqss")
	expect(t, cmd.Decrqss == DCSCommandDECRQSSNone, "cmd.decrqss == .none")

	h.discard()

	expect(t, !h.Hook(ParserActionDCS{Intermediates: []uint8("$"), Final: 'q'}).Valid, "h.hook(alloc, .{ .intermediates = \"$\", .final = 'q' }) == null")
	_ = h.Put('"')
	_ = h.Put(' ')
	_ = h.Put('q')
	expect(t, !h.Unhook().Valid, "h.unhook() == null")
}

// Zig: test "tmux enter and implicit exit"
func TestDCS_tmux_enter_and_implicit_exit(t *testing.T) {
	// Zig: if (comptime !build_options.tmux_control_mode) return error.SkipZigTest;
	t.Skip("tmux control mode is not compiled into the library build")
}

// ---------------------------------------------------------------------------
// term51: additional tests. Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

// term51 test helper: hook, put every byte, unhook.
func term51Run(h *DCSHandler, dcs ParserActionDCS, data string) Optional[DCSCommand] {
	_ = h.Hook(dcs)
	for _, byte_ := range []uint8(data) {
		_ = h.Put(byte_)
	}
	return h.Unhook()
}

// Zig: test "term51: Handler.tryHook: recognised requests select their state and carry no command"
func TestDCS_term51_Handler_tryHook_recognised_requests_select_their_state_and_carry_no_command(t *testing.T) {
	h := DCSHandlerInit()
	{
		hk_, err := h.tryHook(ParserActionDCS{Intermediates: []uint8("$"), Final: 'q'})
		zigTry(t, err)
		hk := zigUnwrap(t, hk_)
		defer hk.state.Deinit()
		expect(t, hk.state.tag == DCSStateDecrqss, "hk.state == .decrqss")
		expect(t, !hk.command.Valid, "hk.command == null")
		expectEqual(t, uint8(0), hk.state.decrqss.len)
	}
	{
		hk_, err := h.tryHook(ParserActionDCS{Intermediates: []uint8("+"), Final: 'q'})
		zigTry(t, err)
		hk := zigUnwrap(t, hk_)
		defer hk.state.Deinit()
		expect(t, hk.state.tag == DCSStateXtgettcap, "hk.state == .xtgettcap")
		expect(t, !hk.command.Valid, "hk.command == null")
		expectEqual(t, 0, len(hk.state.xtgettcap.written()))
	}
}

// Zig: test "term51: Handler.tryHook: everything else is unknown"
func TestDCS_term51_Handler_tryHook_everything_else_is_unknown(t *testing.T) {
	h := DCSHandlerInit()
	cases := []ParserActionDCS{
		{Final: 'q'},
		{Final: 'p'},
		{Final: 'x'},
		{Intermediates: []uint8("$"), Final: 'p'},
		{Intermediates: []uint8("$"), Final: 'r'},
		{Intermediates: []uint8("+"), Final: 'p'},
		{Intermediates: []uint8("+"), Final: 'Q'},
		{Intermediates: []uint8("!"), Final: 'q'},
		{Intermediates: []uint8("$$"), Final: 'q'},
		{Intermediates: []uint8("+$"), Final: 'q'},
		{Intermediates: []uint8("$"), Params: []uint16{1}, Final: 'x'},
	}
	for _, dcs := range cases {
		hk, err := h.tryHook(dcs)
		zigTry(t, err)
		expect(t, !hk.Valid, "(try h.tryHook(alloc, dcs)) == null")
	}
}

// Zig: test "term51: Handler.tryHook: tmux control mode is not part of the library build"
func TestDCS_term51_Handler_tryHook_tmux_control_mode_is_not_part_of_the_library_build(t *testing.T) {
	h := DCSHandlerInit()
	// Zig: if (comptime build_options.tmux_control_mode) return error.SkipZigTest;
	// (tmux control mode is compiled out, so the test runs.)
	{
		hk, err := h.tryHook(ParserActionDCS{Params: []uint16{1000}, Final: 'p'})
		zigTry(t, err)
		expect(t, !hk.Valid, "(try h.tryHook(alloc, .{ .params = &.{1000}, .final = 'p' })) == null")
	}
	{
		hk, err := h.tryHook(ParserActionDCS{Params: []uint16{999}, Final: 'p'})
		zigTry(t, err)
		expect(t, !hk.Valid, "(try h.tryHook(alloc, .{ .params = &.{999}, .final = 'p' })) == null")
	}
	{
		hk, err := h.tryHook(ParserActionDCS{Params: []uint16{1000, 1}, Final: 'p'})
		zigTry(t, err)
		expect(t, !hk.Valid, "(try h.tryHook(alloc, .{ .params = &.{ 1000, 1 }, .final = 'p' })) == null")
	}
}

// Zig: test "term51: Handler.tryHook: parameters do not matter for decrqss and xtgettcap"
func TestDCS_term51_Handler_tryHook_parameters_do_not_matter_for_decrqss_and_xtgettcap(t *testing.T) {
	h := DCSHandlerInit()
	a_, err := h.tryHook(ParserActionDCS{Intermediates: []uint8("$"), Params: []uint16{1, 2, 3}, Final: 'q'})
	zigTry(t, err)
	a := zigUnwrap(t, a_)
	defer a.state.Deinit()
	expect(t, a.state.tag == DCSStateDecrqss, "a.state == .decrqss")
	b_, err := h.tryHook(ParserActionDCS{Intermediates: []uint8("+"), Params: []uint16{9}, Final: 'q'})
	zigTry(t, err)
	b := zigUnwrap(t, b_)
	defer b.state.Deinit()
	expect(t, b.state.tag == DCSStateXtgettcap, "b.state == .xtgettcap")
}

// Zig: test "term51: Handler.hook: an unknown request puts the handler in ignore"
func TestDCS_term51_Handler_hook_an_unknown_request_puts_the_handler_in_ignore(t *testing.T) {
	h := DCSHandlerInit()
	defer h.Deinit()
	expect(t, !h.Hook(ParserActionDCS{Final: 'x'}).Valid, "h.hook(term51_testing.allocator, .{ .final = 'x' }) == null")
	expect(t, h.state.tag == DCSStateIgnore, "h.state == .ignore")
	// bytes are swallowed and the unhook yields nothing
	expect(t, !h.Put('a').Valid, "h.put('a') == null")
	expect(t, !h.Unhook().Valid, "h.unhook() == null")
	expect(t, h.state.tag == DCSStateInactive, "h.state == .inactive")
}

// Zig: test "term51: Handler.hook: a recognised request returns no command at hook time"
func TestDCS_term51_Handler_hook_a_recognised_request_returns_no_command_at_hook_time(t *testing.T) {
	h := DCSHandlerInit()
	defer h.Deinit()
	expect(t, !h.Hook(ParserActionDCS{Intermediates: []uint8("$"), Final: 'q'}).Valid, "h.hook(term51_testing.allocator, .{ .intermediates = \"$\", .final = 'q' }) == null")
	expect(t, h.state.tag == DCSStateDecrqss, "h.state == .decrqss")
	_ = h.Unhook()
	expect(t, !h.Hook(ParserActionDCS{Intermediates: []uint8("+"), Final: 'q'}).Valid, "h.hook(term51_testing.allocator, .{ .intermediates = \"+\", .final = 'q' }) == null")
	expect(t, h.state.tag == DCSStateXtgettcap, "h.state == .xtgettcap")
}

// Zig: test "term51: Handler.tryPut: inactive and ignore swallow bytes"
func TestDCS_term51_Handler_tryPut_inactive_and_ignore_swallow_bytes(t *testing.T) {
	h := DCSHandlerInit()
	defer h.Deinit()
	{
		cmd, err := h.tryPut('a')
		zigTry(t, err)
		expect(t, !cmd.Valid, "(try h.tryPut('a')) == null")
	}
	expect(t, h.state.tag == DCSStateInactive, "h.state == .inactive")
	h.state = DCSState{tag: DCSStateIgnore}
	{
		cmd, err := h.tryPut('a')
		zigTry(t, err)
		expect(t, !cmd.Valid, "(try h.tryPut('a')) == null")
	}
	expect(t, h.state.tag == DCSStateIgnore, "h.state == .ignore")
}

// Zig: test "term51: Handler.tryPut: decrqss holds two bytes and fails on the third"
func TestDCS_term51_Handler_tryPut_decrqss_holds_two_bytes_and_fails_on_the_third(t *testing.T) {
	h := DCSHandlerInit()
	defer h.Deinit()
	_ = h.Hook(ParserActionDCS{Intermediates: []uint8("$"), Final: 'q'})
	{
		cmd, err := h.tryPut(' ')
		zigTry(t, err)
		expect(t, !cmd.Valid, "(try h.tryPut(' ')) == null")
	}
	expectEqual(t, uint8(1), h.state.decrqss.len)
	{
		cmd, err := h.tryPut('q')
		zigTry(t, err)
		expect(t, !cmd.Valid, "(try h.tryPut('q')) == null")
	}
	expectEqual(t, uint8(2), h.state.decrqss.len)
	expectEqualStrings(t, " q", string(h.state.decrqss.data[0:2]))
	{
		_, err := h.tryPut('x')
		expectEqual(t, error(ErrOutOfMemory), err)
	}
	// the failed byte changed nothing
	expectEqual(t, uint8(2), h.state.decrqss.len)
}

// Zig: test "term51: Handler.tryPut: xtgettcap accepts up to max_bytes and fails on the next"
func TestDCS_term51_Handler_tryPut_xtgettcap_accepts_up_to_max_bytes_and_fails_on_the_next(t *testing.T) {
	h := DCSHandlerInit()
	h.max_bytes = 4
	defer h.Deinit()
	_ = h.Hook(ParserActionDCS{Intermediates: []uint8("+"), Final: 'q'})
	for _, byte_ := range []uint8("abcd") {
		cmd, err := h.tryPut(byte_)
		zigTry(t, err)
		expect(t, !cmd.Valid, "(try h.tryPut(byte)) == null")
	}
	expectEqualStrings(t, "abcd", string(h.state.xtgettcap.written()))
	{
		_, err := h.tryPut('e')
		expectEqual(t, error(ErrOutOfMemory), err)
	}
	expectEqualStrings(t, "abcd", string(h.state.xtgettcap.written()))
}

// Zig: test "term51: Handler.put: overflowing decrqss discards the request"
func TestDCS_term51_Handler_put_overflowing_decrqss_discards_the_request(t *testing.T) {
	h := DCSHandlerInit()
	defer h.Deinit()
	_ = h.Hook(ParserActionDCS{Intermediates: []uint8("$"), Final: 'q'})
	expect(t, !h.Put(' ').Valid, "h.put(' ') == null")
	expect(t, !h.Put('q').Valid, "h.put('q') == null")
	expect(t, h.state.tag == DCSStateDecrqss, "h.state == .decrqss")
	expect(t, !h.Put('x').Valid, "h.put('x') == null")
	expect(t, h.state.tag == DCSStateIgnore, "h.state == .ignore")
	expect(t, !h.Unhook().Valid, "h.unhook() == null")
}

// Zig: test "term51: Handler.put: overflowing xtgettcap frees the buffer and ignores the rest"
func TestDCS_term51_Handler_put_overflowing_xtgettcap_frees_the_buffer_and_ignores_the_rest(t *testing.T) {
	h := DCSHandlerInit()
	h.max_bytes = 3
	defer h.Deinit()
	_ = h.Hook(ParserActionDCS{Intermediates: []uint8("+"), Final: 'q'})
	for _, byte_ := range []uint8("abc") {
		expect(t, !h.Put(byte_).Valid, "h.put(byte) == null")
	}
	expect(t, h.state.tag == DCSStateXtgettcap, "h.state == .xtgettcap")
	expect(t, !h.Put('d').Valid, "h.put('d') == null")
	expect(t, h.state.tag == DCSStateIgnore, "h.state == .ignore")
	expect(t, !h.Put('e').Valid, "h.put('e') == null")
	expect(t, !h.Unhook().Valid, "h.unhook() == null")
}

// Zig: test "term51: Handler.put: default limit"
func TestDCS_term51_Handler_put_default_limit(t *testing.T) {
	h := DCSHandlerInit()
	expectEqual(t, uint(1024*1024), h.max_bytes)
	expect(t, h.state.tag == DCSStateInactive, "h.state == .inactive")
}

// Zig: test "term51: Handler.unhook: nothing hooked"
func TestDCS_term51_Handler_unhook_nothing_hooked(t *testing.T) {
	h := DCSHandlerInit()
	defer h.Deinit()
	expect(t, !h.Unhook().Valid, "h.unhook() == null")
	expect(t, h.state.tag == DCSStateInactive, "h.state == .inactive")
}

// Zig: test "term51: Handler.unhook: every decrqss request"
func TestDCS_term51_Handler_unhook_every_decrqss_request(t *testing.T) {
	const (
		none     = DCSCommandDECRQSSNone
		sgr      = DCSCommandDECRQSSSgr
		decstbm  = DCSCommandDECRQSSDecstbm
		decslrm  = DCSCommandDECRQSSDecslrm
		decscusr = DCSCommandDECRQSSDecscusr
	)
	cases := []struct {
		data string
		want DCSCommandDECRQSS
	}{
		{"", none}, {"m", sgr}, {"r", decstbm},
		{"s", decslrm}, {" q", decscusr}, {"q", none},
		{"x", none}, {" ", none}, {" x", none},
		{"mm", none}, {"m ", none}, {"\"q", none},
		{"\"p", none}, {"t", none}, {"$}", none},
	}
	for _, case_ := range cases {
		h := DCSHandlerInit()
		defer h.Deinit()
		cmd := zigUnwrap(t, term51Run(&h, ParserActionDCS{Intermediates: []uint8("$"), Final: 'q'}, case_.data))
		defer cmd.Deinit()
		expect(t, cmd.Tag == DCSCommandDecrqss, "cmd == .decrqss")
		expectEqual(t, case_.want, cmd.Decrqss)
		expect(t, h.state.tag == DCSStateInactive, "h.state == .inactive")
	}
}

// Zig: test "term51: Handler.unhook: xtgettcap data is upper-cased"
func TestDCS_term51_Handler_unhook_xtgettcap_data_is_upper_cased(t *testing.T) {
	h := DCSHandlerInit()
	defer h.Deinit()
	cmd := zigUnwrap(t, term51Run(&h, ParserActionDCS{Intermediates: []uint8("+"), Final: 'q'}, "544e;636f;aBcDef09"))
	defer cmd.Deinit()
	expect(t, cmd.Tag == DCSCommandXtgettcap, "cmd == .xtgettcap")
	expectEqualStrings(t, "544E;636F;ABCDEF09", string(cmd.Xtgettcap.data.written()))
	expect(t, h.state.tag == DCSStateInactive, "h.state == .inactive")
}

// Zig: test "term51: Handler.unhook: an empty xtgettcap request still yields a command"
func TestDCS_term51_Handler_unhook_an_empty_xtgettcap_request_still_yields_a_command(t *testing.T) {
	h := DCSHandlerInit()
	defer h.Deinit()
	cmd := zigUnwrap(t, term51Run(&h, ParserActionDCS{Intermediates: []uint8("+"), Final: 'q'}, ""))
	defer cmd.Deinit()
	expect(t, cmd.Tag == DCSCommandXtgettcap, "cmd == .xtgettcap")
	expect(t, !cmd.Xtgettcap.Next().Valid, "cmd.xtgettcap.next() == null")
}

// Zig: test "term51: Handler.unhook: the handler can be hooked again afterwards"
func TestDCS_term51_Handler_unhook_the_handler_can_be_hooked_again_afterwards(t *testing.T) {
	h := DCSHandlerInit()
	defer h.Deinit()
	a := zigUnwrap(t, term51Run(&h, ParserActionDCS{Intermediates: []uint8("$"), Final: 'q'}, "m"))
	a.Deinit()
	b := zigUnwrap(t, term51Run(&h, ParserActionDCS{Intermediates: []uint8("$"), Final: 'q'}, "r"))
	b.Deinit()
	expectEqual(t, DCSCommandDECRQSSDecstbm, b.Decrqss)
}

// Zig: test "term51: Handler.discard: frees a pending xtgettcap buffer"
func TestDCS_term51_Handler_discard_frees_a_pending_xtgettcap_buffer(t *testing.T) {
	h := DCSHandlerInit()
	_ = h.Hook(ParserActionDCS{Intermediates: []uint8("+"), Final: 'q'})
	for _, byte_ := range []uint8("abcdef") {
		_ = h.Put(byte_)
	}
	expect(t, h.state.tag == DCSStateXtgettcap, "h.state == .xtgettcap")
	h.discard() // leak check (Go has no leak check; the call is kept)
	expect(t, h.state.tag == DCSStateInactive, "h.state == .inactive")
	// idempotent
	h.discard()
	expect(t, h.state.tag == DCSStateInactive, "h.state == .inactive")
}

// Zig: test "term51: Handler.deinit: frees a pending request"
func TestDCS_term51_Handler_deinit_frees_a_pending_request(t *testing.T) {
	h := DCSHandlerInit()
	_ = h.Hook(ParserActionDCS{Intermediates: []uint8("+"), Final: 'q'})
	for _, byte_ := range []uint8("abcdef") {
		_ = h.Put(byte_)
	}
	h.Deinit() // leak check (Go has no leak check; the call is kept)
	expect(t, h.state.tag == DCSStateInactive, "h.state == .inactive")
}

// Zig: test "term51: Command.deinit: frees xtgettcap data and is a no-op for decrqss"
func TestDCS_term51_Command_deinit_frees_xtgettcap_data_and_is_a_no_op_for_decrqss(t *testing.T) {
	h := DCSHandlerInit()
	defer h.Deinit()
	a := zigUnwrap(t, term51Run(&h, ParserActionDCS{Intermediates: []uint8("+"), Final: 'q'}, "544e"))
	a.Deinit() // leak check (Go has no leak check; the call is kept)
	b := DCSCommand{Tag: DCSCommandDecrqss, Decrqss: DCSCommandDECRQSSSgr}
	b.Deinit()
	expectEqual(t, DCSCommandDECRQSSSgr, b.Decrqss)
}

// Zig: test "term51: Command.XTGETTCAP.next: splits on semicolons"
func TestDCS_term51_Command_XTGETTCAP_next_splits_on_semicolons(t *testing.T) {
	cases := []struct {
		data string
		want []string
	}{
		{"", []string{}},
		{"a", []string{"A"}},
		{"a;b", []string{"A", "B"}},
		{"544e;636f;5463", []string{"544E", "636F", "5463"}},
		// a trailing separator adds nothing
		{"a;", []string{"A"}},
		// empty fields are returned as empty strings
		{";a", []string{"", "A"}},
		{"a;;b", []string{"A", "", "B"}},
		{";", []string{""}},
		{";;", []string{"", ""}},
	}
	for _, case_ := range cases {
		h := DCSHandlerInit()
		defer h.Deinit()
		cmd := zigUnwrap(t, term51Run(&h, ParserActionDCS{Intermediates: []uint8("+"), Final: 'q'}, case_.data))
		defer cmd.Deinit()
		for _, want := range case_.want {
			expectEqualStrings(t, want, string(zigUnwrap(t, cmd.Xtgettcap.Next())))
		}
		expect(t, !cmd.Xtgettcap.Next().Valid, "cmd.xtgettcap.next() == null")
		// and stays exhausted
		expect(t, !cmd.Xtgettcap.Next().Valid, "cmd.xtgettcap.next() == null")
	}
}

// Zig: test "term51: State.deinit: every state"
func TestDCS_term51_State_deinit_every_state(t *testing.T) {
	a := DCSState{tag: DCSStateInactive}
	a.Deinit()
	b := DCSState{tag: DCSStateIgnore}
	b.Deinit()
	c := DCSState{tag: DCSStateDecrqss}
	c.Deinit()
	// Zig-only: .initCapacity(term51_testing.allocator, 16) (capacity argument; Go slices grow on demand)
	d := DCSState{tag: DCSStateXtgettcap, xtgettcap: &dcsWriterAllocating{}}
	d.Deinit() // leak check (Go has no leak check; the call is kept)
}
