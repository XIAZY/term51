// Tests translated one for one from src/terminal/Parser.zig (Ghostty v1.3.1
// plus the term51 additional tests). Each Go test carries the exact Zig title
// in a "// Zig: test" comment. Tests that need a working OSC parser are in
// Parser_osc_test.go.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualSlices / expectEqualStrings keep Zig's names
//     and Zig's (expected, actual) argument order. Like `try`, they stop the
//     test at the first failure.
//   - `x.?` is zigUnwrap, and reading a union payload (`x.?.csi_dispatch`) is
//     zigPayload / zigField, which check the tag as Zig's safety-checked
//     union field access does.
//   - Zig's `defer p.deinit()` inside a loop body runs at the end of each
//     iteration; Go's defer runs when the test function returns.

package terminal

import (
	"testing"
)

// expect is Zig's `try testing.expect(ok)`. zig is the Zig expression.
func expect(t *testing.T, ok bool, zig string) {
	t.Helper()
	if !ok {
		t.Fatalf("expect failed: %s", zig)
	}
}

// expectEqual is Zig's `try testing.expectEqual(expected, actual)`.
func expectEqual[T comparable](t *testing.T, want T, got T) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// expectEqualSlices is Zig's `try testing.expectEqualSlices(T, expected, actual)`.
func expectEqualSlices[T comparable](t *testing.T, want []T, got []T) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v (len %d), want %v (len %d)", got, len(got), want, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v (first difference at index %d)", got, want, i)
		}
	}
}

// expectEqualStrings is Zig's `try testing.expectEqualStrings(expected, actual)`.
func expectEqualStrings(t *testing.T, want string, got string) {
	t.Helper()
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// zigUnwrap is Zig's `o.?`: the value of an optional, failing if it is null.
func zigUnwrap[T any](t *testing.T, o Optional[T]) T {
	t.Helper()
	if !o.Valid {
		t.Fatalf("unwrapped a null optional")
	}
	return o.Value
}

// zigField is Zig's safety-checked union field access `a.<tag>`: it fails if
// the active tag is not tag. The caller then reads the matching Go field.
func zigField(t *testing.T, a ParserAction, tag ParserActionTag) ParserAction {
	t.Helper()
	if a.Tag != tag {
		t.Fatalf("access of union field %v while field %v is active", tag, a.Tag)
	}
	return a
}

// zigPayload is Zig's `o.?.<tag>`: zigUnwrap followed by zigField.
func zigPayload(t *testing.T, o Optional[ParserAction], tag ParserActionTag) ParserAction {
	t.Helper()
	return zigField(t, zigUnwrap(t, o), tag)
}

// Zig: test (unnamed, Parser.zig:417)
func TestParser_unnamed(t *testing.T) {
	p := ParserInit()
	_ = p.Next(0x9E)
	expect(t, p.state == ParserStateSosPmApcString, "p.state == .sos_pm_apc_string")
	_ = p.Next(0x9C)
	expect(t, p.state == ParserStateGround, "p.state == .ground")

	{
		a := p.Next('a')
		expect(t, p.state == ParserStateGround, "p.state == .ground")
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, zigUnwrap(t, a[1]).Tag == ParserActionPrint, "a[1].? == .print")
		expect(t, !a[2].Valid, "a[2] == null")
	}

	{
		a := p.Next(0x19)
		expect(t, p.state == ParserStateGround, "p.state == .ground")
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, zigUnwrap(t, a[1]).Tag == ParserActionExecute, "a[1].? == .execute")
		expect(t, !a[2].Valid, "a[2] == null")
	}
}

// Zig: test "esc: ESC ( B"
func TestParser_esc_ESC_B(t *testing.T) {
	p := ParserInit()
	_ = p.Next(0x1B)
	_ = p.Next('(')

	{
		a := p.Next('B')
		expect(t, p.state == ParserStateGround, "p.state == .ground")
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, zigUnwrap(t, a[1]).Tag == ParserActionEscDispatch, "a[1].? == .esc_dispatch")
		expect(t, !a[2].Valid, "a[2] == null")

		d := zigPayload(t, a[1], ParserActionEscDispatch).EscDispatch
		expect(t, d.Final == 'B', "d.final == 'B'")
		expect(t, len(d.Intermediates) == 1, "d.intermediates.len == 1")
		expect(t, d.Intermediates[0] == '(', "d.intermediates[0] == '('")
	}
}

// Zig: test "csi: ESC [ H"
func TestParser_csi_ESC_H(t *testing.T) {
	p := ParserInit()
	_ = p.Next(0x1B)
	_ = p.Next(0x5B)

	{
		a := p.Next(0x48)
		expect(t, p.state == ParserStateGround, "p.state == .ground")
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, zigUnwrap(t, a[1]).Tag == ParserActionCsiDispatch, "a[1].? == .csi_dispatch")
		expect(t, !a[2].Valid, "a[2] == null")

		d := zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch
		expect(t, d.Final == 0x48, "d.final == 0x48")
		expect(t, len(d.Params) == 0, "d.params.len == 0")
	}
}

// Zig: test "csi: ESC [ 1 ; 4 H"
func TestParser_csi_ESC_1_4_H(t *testing.T) {
	p := ParserInit()
	_ = p.Next(0x1B)
	_ = p.Next(0x5B)
	_ = p.Next(0x31) // 1
	_ = p.Next(0x3B) // ;
	_ = p.Next(0x34) // 4

	{
		a := p.Next(0x48) // H
		expect(t, p.state == ParserStateGround, "p.state == .ground")
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, zigUnwrap(t, a[1]).Tag == ParserActionCsiDispatch, "a[1].? == .csi_dispatch")
		expect(t, !a[2].Valid, "a[2] == null")

		d := zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch
		expect(t, d.Final == 'H', "d.final == 'H'")
		expect(t, len(d.Params) == 2, "d.params.len == 2")
		expectEqual(t, uint16(1), d.Params[0])
		expectEqual(t, uint16(4), d.Params[1])
	}
}

// Zig: test "csi: SGR ESC [ 38 : 2 m"
func TestParser_csi_SGR_ESC_38_2_m(t *testing.T) {
	p := ParserInit()
	_ = p.Next(0x1B)
	_ = p.Next('[')
	_ = p.Next('3')
	_ = p.Next('8')
	_ = p.Next(':')
	_ = p.Next('2')

	{
		a := p.Next('m')
		expect(t, p.state == ParserStateGround, "p.state == .ground")
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, zigUnwrap(t, a[1]).Tag == ParserActionCsiDispatch, "a[1].? == .csi_dispatch")
		expect(t, !a[2].Valid, "a[2] == null")

		d := zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch
		expect(t, d.Final == 'm', "d.final == 'm'")
		expect(t, len(d.Params) == 2, "d.params.len == 2")
		expectEqual(t, uint16(38), d.Params[0])
		expect(t, d.ParamsSep.isSet(0), "d.params_sep.isSet(0)")
		expectEqual(t, uint16(2), d.Params[1])
		expect(t, !d.ParamsSep.isSet(1), "!d.params_sep.isSet(1)")
	}
}

// Zig: test "csi: SGR colon followed by semicolon"
func TestParser_csi_SGR_colon_followed_by_semicolon(t *testing.T) {
	p := ParserInit()
	_ = p.Next(0x1B)
	for _, c := range []uint8("[48:2") {
		a := p.Next(c)
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, !a[1].Valid, "a[1] == null")
		expect(t, !a[2].Valid, "a[2] == null")
	}

	{
		a := p.Next('m')
		expect(t, p.state == ParserStateGround, "p.state == .ground")
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, zigUnwrap(t, a[1]).Tag == ParserActionCsiDispatch, "a[1].? == .csi_dispatch")
		expect(t, !a[2].Valid, "a[2] == null")
	}

	_ = p.Next(0x1B)
	_ = p.Next('[')
	{
		a := p.Next('H')
		expect(t, p.state == ParserStateGround, "p.state == .ground")
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, zigUnwrap(t, a[1]).Tag == ParserActionCsiDispatch, "a[1].? == .csi_dispatch")
		expect(t, !a[2].Valid, "a[2] == null")
	}
}

// Zig: test "csi: SGR mixed colon and semicolon"
func TestParser_csi_SGR_mixed_colon_and_semicolon(t *testing.T) {
	p := ParserInit()
	_ = p.Next(0x1B)
	for _, c := range []uint8("[38:5:1;48:5:0") {
		a := p.Next(c)
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, !a[1].Valid, "a[1] == null")
		expect(t, !a[2].Valid, "a[2] == null")
	}

	{
		a := p.Next('m')
		expect(t, p.state == ParserStateGround, "p.state == .ground")
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, zigUnwrap(t, a[1]).Tag == ParserActionCsiDispatch, "a[1].? == .csi_dispatch")
		expect(t, !a[2].Valid, "a[2] == null")
	}
}

// Zig: test "csi: SGR ESC [ 48 : 2 m"
func TestParser_csi_SGR_ESC_48_2_m(t *testing.T) {
	p := ParserInit()
	_ = p.Next(0x1B)
	for _, c := range []uint8("[48:2:240:143:104") {
		a := p.Next(c)
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, !a[1].Valid, "a[1] == null")
		expect(t, !a[2].Valid, "a[2] == null")
	}

	{
		a := p.Next('m')
		expect(t, p.state == ParserStateGround, "p.state == .ground")
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, zigUnwrap(t, a[1]).Tag == ParserActionCsiDispatch, "a[1].? == .csi_dispatch")
		expect(t, !a[2].Valid, "a[2] == null")

		d := zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch
		expect(t, d.Final == 'm', "d.final == 'm'")
		expect(t, len(d.Params) == 5, "d.params.len == 5")
		expectEqual(t, uint16(48), d.Params[0])
		expect(t, d.ParamsSep.isSet(0), "d.params_sep.isSet(0)")
		expectEqual(t, uint16(2), d.Params[1])
		expect(t, d.ParamsSep.isSet(1), "d.params_sep.isSet(1)")
		expectEqual(t, uint16(240), d.Params[2])
		expect(t, d.ParamsSep.isSet(2), "d.params_sep.isSet(2)")
		expectEqual(t, uint16(143), d.Params[3])
		expect(t, d.ParamsSep.isSet(3), "d.params_sep.isSet(3)")
		expectEqual(t, uint16(104), d.Params[4])
		expect(t, !d.ParamsSep.isSet(4), "!d.params_sep.isSet(4)")
	}
}

// Zig: test "csi: SGR ESC [4:3m colon"
func TestParser_csi_SGR_ESC_4_3m_colon(t *testing.T) {
	p := ParserInit()
	_ = p.Next(0x1B)
	_ = p.Next('[')
	_ = p.Next('4')
	_ = p.Next(':')
	_ = p.Next('3')

	{
		a := p.Next('m')
		expect(t, p.state == ParserStateGround, "p.state == .ground")
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, zigUnwrap(t, a[1]).Tag == ParserActionCsiDispatch, "a[1].? == .csi_dispatch")
		expect(t, !a[2].Valid, "a[2] == null")

		d := zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch
		expect(t, d.Final == 'm', "d.final == 'm'")
		expect(t, len(d.Params) == 2, "d.params.len == 2")
		expectEqual(t, uint16(4), d.Params[0])
		expect(t, d.ParamsSep.isSet(0), "d.params_sep.isSet(0)")
		expectEqual(t, uint16(3), d.Params[1])
		expect(t, !d.ParamsSep.isSet(1), "!d.params_sep.isSet(1)")
	}
}

// Zig: test "csi: SGR with many blank and colon"
func TestParser_csi_SGR_with_many_blank_and_colon(t *testing.T) {
	p := ParserInit()
	_ = p.Next(0x1B)
	for _, c := range []uint8("[58:2::240:143:104") {
		a := p.Next(c)
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, !a[1].Valid, "a[1] == null")
		expect(t, !a[2].Valid, "a[2] == null")
	}

	{
		a := p.Next('m')
		expect(t, p.state == ParserStateGround, "p.state == .ground")
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, zigUnwrap(t, a[1]).Tag == ParserActionCsiDispatch, "a[1].? == .csi_dispatch")
		expect(t, !a[2].Valid, "a[2] == null")

		d := zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch
		expect(t, d.Final == 'm', "d.final == 'm'")
		expect(t, len(d.Params) == 6, "d.params.len == 6")
		expectEqual(t, uint16(58), d.Params[0])
		expect(t, d.ParamsSep.isSet(0), "d.params_sep.isSet(0)")
		expectEqual(t, uint16(2), d.Params[1])
		expect(t, d.ParamsSep.isSet(1), "d.params_sep.isSet(1)")
		expectEqual(t, uint16(0), d.Params[2])
		expect(t, d.ParamsSep.isSet(2), "d.params_sep.isSet(2)")
		expectEqual(t, uint16(240), d.Params[3])
		expect(t, d.ParamsSep.isSet(3), "d.params_sep.isSet(3)")
		expectEqual(t, uint16(143), d.Params[4])
		expect(t, d.ParamsSep.isSet(4), "d.params_sep.isSet(4)")
		expectEqual(t, uint16(104), d.Params[5])
		expect(t, !d.ParamsSep.isSet(5), "!d.params_sep.isSet(5)")
	}
}

// This is from a Kakoune actual SGR sequence.
//
// Zig: test "csi: SGR mixed colon and semicolon with blank"
func TestParser_csi_SGR_mixed_colon_and_semicolon_with_blank(t *testing.T) {
	p := ParserInit()
	_ = p.Next(0x1B)
	for _, c := range []uint8("[;4:3;38;2;175;175;215;58:2::190:80:70") {
		a := p.Next(c)
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, !a[1].Valid, "a[1] == null")
		expect(t, !a[2].Valid, "a[2] == null")
	}

	{
		a := p.Next('m')
		expect(t, p.state == ParserStateGround, "p.state == .ground")
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, zigUnwrap(t, a[1]).Tag == ParserActionCsiDispatch, "a[1].? == .csi_dispatch")
		expect(t, !a[2].Valid, "a[2] == null")

		d := zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch
		expect(t, d.Final == 'm', "d.final == 'm'")
		expectEqual(t, 14, len(d.Params))
		expectEqual(t, uint16(0), d.Params[0])
		expect(t, !d.ParamsSep.isSet(0), "!d.params_sep.isSet(0)")
		expectEqual(t, uint16(4), d.Params[1])
		expect(t, d.ParamsSep.isSet(1), "d.params_sep.isSet(1)")
		expectEqual(t, uint16(3), d.Params[2])
		expect(t, !d.ParamsSep.isSet(2), "!d.params_sep.isSet(2)")
		expectEqual(t, uint16(38), d.Params[3])
		expect(t, !d.ParamsSep.isSet(3), "!d.params_sep.isSet(3)")
		expectEqual(t, uint16(2), d.Params[4])
		expect(t, !d.ParamsSep.isSet(4), "!d.params_sep.isSet(4)")
		expectEqual(t, uint16(175), d.Params[5])
		expect(t, !d.ParamsSep.isSet(5), "!d.params_sep.isSet(5)")
		expectEqual(t, uint16(175), d.Params[6])
		expect(t, !d.ParamsSep.isSet(6), "!d.params_sep.isSet(6)")
		expectEqual(t, uint16(215), d.Params[7])
		expect(t, !d.ParamsSep.isSet(7), "!d.params_sep.isSet(7)")
		expectEqual(t, uint16(58), d.Params[8])
		expect(t, d.ParamsSep.isSet(8), "d.params_sep.isSet(8)")
		expectEqual(t, uint16(2), d.Params[9])
		expect(t, d.ParamsSep.isSet(9), "d.params_sep.isSet(9)")
		expectEqual(t, uint16(0), d.Params[10])
		expect(t, d.ParamsSep.isSet(10), "d.params_sep.isSet(10)")
		expectEqual(t, uint16(190), d.Params[11])
		expect(t, d.ParamsSep.isSet(11), "d.params_sep.isSet(11)")
		expectEqual(t, uint16(80), d.Params[12])
		expect(t, d.ParamsSep.isSet(12), "d.params_sep.isSet(12)")
		expectEqual(t, uint16(70), d.Params[13])
		expect(t, !d.ParamsSep.isSet(13), "!d.params_sep.isSet(13)")
	}
}

// This is from a Kakoune actual SGR sequence also.
//
// Zig: test "csi: SGR mixed colon and semicolon setting underline, bg, fg"
func TestParser_csi_SGR_mixed_colon_and_semicolon_setting_underline_bg_fg(t *testing.T) {
	p := ParserInit()
	_ = p.Next(0x1B)
	for _, c := range []uint8("[4:3;38;2;51;51;51;48;2;170;170;170;58;2;255;97;136") {
		a := p.Next(c)
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, !a[1].Valid, "a[1] == null")
		expect(t, !a[2].Valid, "a[2] == null")
	}

	{
		a := p.Next('m')
		expect(t, p.state == ParserStateGround, "p.state == .ground")
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, zigUnwrap(t, a[1]).Tag == ParserActionCsiDispatch, "a[1].? == .csi_dispatch")
		expect(t, !a[2].Valid, "a[2] == null")

		d := zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch
		expect(t, d.Final == 'm', "d.final == 'm'")
		expectEqual(t, 17, len(d.Params))
		expectEqual(t, uint16(4), d.Params[0])
		expect(t, d.ParamsSep.isSet(0), "d.params_sep.isSet(0)")
		expectEqual(t, uint16(3), d.Params[1])
		expect(t, !d.ParamsSep.isSet(1), "!d.params_sep.isSet(1)")
		expectEqual(t, uint16(38), d.Params[2])
		expect(t, !d.ParamsSep.isSet(2), "!d.params_sep.isSet(2)")
		expectEqual(t, uint16(2), d.Params[3])
		expect(t, !d.ParamsSep.isSet(3), "!d.params_sep.isSet(3)")
		expectEqual(t, uint16(51), d.Params[4])
		expect(t, !d.ParamsSep.isSet(4), "!d.params_sep.isSet(4)")
		expectEqual(t, uint16(51), d.Params[5])
		expect(t, !d.ParamsSep.isSet(5), "!d.params_sep.isSet(5)")
		expectEqual(t, uint16(51), d.Params[6])
		expect(t, !d.ParamsSep.isSet(6), "!d.params_sep.isSet(6)")
		expectEqual(t, uint16(48), d.Params[7])
		expect(t, !d.ParamsSep.isSet(7), "!d.params_sep.isSet(7)")
		expectEqual(t, uint16(2), d.Params[8])
		expect(t, !d.ParamsSep.isSet(8), "!d.params_sep.isSet(8)")
		expectEqual(t, uint16(170), d.Params[9])
		expect(t, !d.ParamsSep.isSet(9), "!d.params_sep.isSet(9)")
		expectEqual(t, uint16(170), d.Params[10])
		expect(t, !d.ParamsSep.isSet(10), "!d.params_sep.isSet(10)")
		expectEqual(t, uint16(170), d.Params[11])
		expect(t, !d.ParamsSep.isSet(11), "!d.params_sep.isSet(11)")
		expectEqual(t, uint16(58), d.Params[12])
		expect(t, !d.ParamsSep.isSet(12), "!d.params_sep.isSet(12)")
		expectEqual(t, uint16(2), d.Params[13])
		expect(t, !d.ParamsSep.isSet(13), "!d.params_sep.isSet(13)")
		expectEqual(t, uint16(255), d.Params[14])
		expect(t, !d.ParamsSep.isSet(14), "!d.params_sep.isSet(14)")
		expectEqual(t, uint16(97), d.Params[15])
		expect(t, !d.ParamsSep.isSet(15), "!d.params_sep.isSet(15)")
		expectEqual(t, uint16(136), d.Params[16])
		expect(t, !d.ParamsSep.isSet(16), "!d.params_sep.isSet(16)")
	}
}

// Zig: test "csi: colon for non-m final"
func TestParser_csi_colon_for_non_m_final(t *testing.T) {
	p := ParserInit()
	_ = p.Next(0x1B)
	for _, c := range []uint8("[38:2h") {
		a := p.Next(c)
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, !a[1].Valid, "a[1] == null")
		expect(t, !a[2].Valid, "a[2] == null")
	}

	expect(t, p.state == ParserStateGround, "p.state == .ground")
}

// Zig: test "csi: request mode decrqm"
func TestParser_csi_request_mode_decrqm(t *testing.T) {
	p := ParserInit()
	_ = p.Next(0x1B)
	for _, c := range []uint8("[?2026$") {
		a := p.Next(c)
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, !a[1].Valid, "a[1] == null")
		expect(t, !a[2].Valid, "a[2] == null")
	}

	{
		a := p.Next('p')
		expect(t, p.state == ParserStateGround, "p.state == .ground")
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, zigUnwrap(t, a[1]).Tag == ParserActionCsiDispatch, "a[1].? == .csi_dispatch")
		expect(t, !a[2].Valid, "a[2] == null")

		d := zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch
		expect(t, d.Final == 'p', "d.final == 'p'")
		expectEqual(t, 2, len(d.Intermediates))
		expectEqual(t, 1, len(d.Params))
		// Zig compares the u8 intermediates against @as(u16, ...).
		expectEqual(t, uint16('?'), uint16(d.Intermediates[0]))
		expectEqual(t, uint16('$'), uint16(d.Intermediates[1]))
		expectEqual(t, uint16(2026), d.Params[0])
	}
}

// Zig: test "csi: change cursor"
func TestParser_csi_change_cursor(t *testing.T) {
	p := ParserInit()
	_ = p.Next(0x1B)
	for _, c := range []uint8("[3 ") {
		a := p.Next(c)
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, !a[1].Valid, "a[1] == null")
		expect(t, !a[2].Valid, "a[2] == null")
	}

	{
		a := p.Next('q')
		expect(t, p.state == ParserStateGround, "p.state == .ground")
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, zigUnwrap(t, a[1]).Tag == ParserActionCsiDispatch, "a[1].? == .csi_dispatch")
		expect(t, !a[2].Valid, "a[2] == null")

		d := zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch
		expect(t, d.Final == 'q', "d.final == 'q'")
		expectEqual(t, 1, len(d.Intermediates))
		expectEqual(t, 1, len(d.Params))
		// Zig compares the u8 intermediate against @as(u16, ' ').
		expectEqual(t, uint16(' '), uint16(d.Intermediates[0]))
		expectEqual(t, uint16(3), d.Params[0])
	}
}

// The four upstream "osc: ..." tests that follow "csi: change cursor" in
// Parser.zig are in Parser_osc_test.go.

// Zig: test "csi: too many params"
func TestParser_csi_too_many_params(t *testing.T) {
	p := ParserInit()
	_ = p.Next(0x1B)
	_ = p.Next('[')
	for range 100 {
		_ = p.Next('1')
		_ = p.Next(';')
	}
	_ = p.Next('1')

	{
		a := p.Next('C')
		expect(t, p.state == ParserStateGround, "p.state == .ground")
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, !a[1].Valid, "a[1] == null")
		expect(t, !a[2].Valid, "a[2] == null")
	}
}

// Zig: test "csi: sgr with up to our max parameters"
func TestParser_csi_sgr_with_up_to_our_max_parameters(t *testing.T) {
	for max := 1; max < MAX_PARAMS+1; max++ {
		p := ParserInit()
		_ = p.Next(0x1B)
		_ = p.Next('[')

		for range max - 1 {
			_ = p.Next('1')
			_ = p.Next(';')
		}
		_ = p.Next('2')

		{
			a := p.Next('H')
			expect(t, p.state == ParserStateGround, "p.state == .ground")
			expect(t, !a[0].Valid, "a[0] == null")
			expect(t, zigUnwrap(t, a[1]).Tag == ParserActionCsiDispatch, "a[1].? == .csi_dispatch")
			expect(t, !a[2].Valid, "a[2] == null")

			csi := zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch
			expectEqual(t, max, len(csi.Params))
			expectEqual(t, uint16(2), csi.Params[max-1])
		}
	}
}

// Zig: test "csi: sgr beyond our max drops it"
func TestParser_csi_sgr_beyond_our_max_drops_it(t *testing.T) {
	// Has to be +2 for the loops below
	const max = MAX_PARAMS + 2

	p := ParserInit()
	_ = p.Next(0x1B)
	_ = p.Next('[')

	for range max - 1 {
		_ = p.Next('1')
		_ = p.Next(';')
	}
	_ = p.Next('2')

	{
		a := p.Next('H')
		expect(t, p.state == ParserStateGround, "p.state == .ground")
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, !a[1].Valid, "a[1] == null")
		expect(t, !a[2].Valid, "a[2] == null")
	}
}

// Zig: test "dcs: XTGETTCAP"
func TestParser_dcs_XTGETTCAP(t *testing.T) {
	p := ParserInit()
	_ = p.Next(0x1B)
	for _, c := range []uint8("P+") {
		a := p.Next(c)
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, !a[1].Valid, "a[1] == null")
		expect(t, !a[2].Valid, "a[2] == null")
	}

	{
		a := p.Next('q')
		expect(t, p.state == ParserStateDcsPassthrough, "p.state == .dcs_passthrough")
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, !a[1].Valid, "a[1] == null")
		expect(t, zigUnwrap(t, a[2]).Tag == ParserActionDcsHook, "a[2].? == .dcs_hook")

		hook := zigPayload(t, a[2], ParserActionDcsHook).DcsHook
		expectEqualSlices(t, []uint8{'+'}, hook.Intermediates)
		expectEqualSlices(t, []uint16{}, hook.Params)
		expectEqual(t, 'q', hook.Final)
	}
}

// Zig: test "dcs: params"
func TestParser_dcs_params(t *testing.T) {
	p := ParserInit()
	_ = p.Next(0x1B)
	for _, c := range []uint8("P1000") {
		a := p.Next(c)
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, !a[1].Valid, "a[1] == null")
		expect(t, !a[2].Valid, "a[2] == null")
	}

	{
		a := p.Next('p')
		expect(t, p.state == ParserStateDcsPassthrough, "p.state == .dcs_passthrough")
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, !a[1].Valid, "a[1] == null")
		expect(t, zigUnwrap(t, a[2]).Tag == ParserActionDcsHook, "a[2].? == .dcs_hook")

		hook := zigPayload(t, a[2], ParserActionDcsHook).DcsHook
		expectEqualSlices(t, []uint16{1000}, hook.Params)
		expectEqual(t, 'p', hook.Final)
	}
}

// Zig: test "dcs: too many params"
func TestParser_dcs_too_many_params(t *testing.T) {
	// Regression test for a crash found by fuzzing (afl). When a DCS
	// sequence has more than MAX_PARAMS parameters and param_acc_idx > 0,
	// entering dcs_passthrough wrote to params[params_idx] without a
	// bounds check, causing an out-of-bounds access.
	p := ParserInit()
	_ = p.Next(0x1B) // ESC
	_ = p.Next('P')  // DCS entry

	// Feed a digit then MAX_PARAMS semicolons to fill all param slots.
	_ = p.Next('6')
	for range MAX_PARAMS {
		_ = p.Next(';')
	}
	// Feed another digit so param_acc_idx > 0 while params_idx == MAX_PARAMS.
	_ = p.Next('7')

	// A final byte triggers entry to dcs_passthrough. The DCS should
	// be dropped entirely, consistent with how CSI handles overflow.
	a := p.Next('p')
	expect(t, !a[0].Valid, "a[0] == null")
	expect(t, !a[1].Valid, "a[1] == null")
	expect(t, !a[2].Valid, "a[2] == null")
}

// ---------------------------------------------------------------------------
// term51: additional tests. Not part of upstream Ghostty.
//
// Each test targets one function, named in the test title, and covers its
// boundaries. They are characterization tests: they pin down what the pinned
// upstream source does, including behaviour that looks like a defect.
// ---------------------------------------------------------------------------

// term51 test helper: feed bytes, return the actions of the last byte.
func term51Feed(p *Parser, bytes string) [3]Optional[ParserAction] {
	last := [3]Optional[ParserAction]{None[ParserAction](), None[ParserAction](), None[ParserAction]()}
	for _, c := range []uint8(bytes) {
		last = p.Next(c)
	}
	return last
}

// Zig: test "term51: init: starts in ground with empty accumulators"
func TestParser_term51_init_starts_in_ground_with_empty_accumulators(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	expectEqual(t, ParserStateGround, p.state)
	expectEqual(t, uint8(0), p.intermediates_idx)
	expectEqual(t, uint8(0), p.params_idx)
	expectEqual(t, 0, p.params_sep.count())
	expectEqual(t, uint16(0), p.param_acc)
	expectEqual(t, uint8(0), p.param_acc_idx)
	expect(t, !p.osc_parser.alloc, "p.osc_parser.alloc == null")
}

// Zig: test "term51: init: limits"
func TestParser_term51_init_limits(t *testing.T) {
	expectEqual(t, 4, MAX_INTERMEDIATE)
	expectEqual(t, 24, MAX_PARAMS)
	// SepList must be able to hold one bit per parameter
	expectEqual(t, MAX_PARAMS, StaticBitSet24BitLength)
}

// Zig: test "term51: deinit: releases an OSC buffer that was never terminated"
func TestParser_term51_deinit_releases_an_OSC_buffer_that_was_never_terminated(t *testing.T) {
	p := ParserInit()
	p.osc_parser.alloc = true // Zig: p.osc_parser.alloc = testing.allocator;
	// A title longer than the fixed buffer forces the allocating writer.
	_ = term51Feed(&p, "\x1b]2;")
	for range 4096 {
		_ = p.Next('x')
	}
	expectEqual(t, ParserStateOscString, p.state)
	// testing.allocator fails the test if this leaks
	// (Go has no leak check; the deinit call itself is kept.)
	p.Deinit()
}

// Zig: test "term51: deinit: safe on a fresh parser"
func TestParser_term51_deinit_safe_on_a_fresh_parser(t *testing.T) {
	p := ParserInit()
	p.Deinit()
}

// Zig: test "term51: next: every printable byte prints in ground"
func TestParser_term51_next_every_printable_byte_prints_in_ground(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	for c := uint8(0x20); c <= 0x7E; c += 1 {
		a := p.Next(c)
		expect(t, !a[0].Valid, "a[0] == null")
		expect(t, zigUnwrap(t, a[1]).Tag == ParserActionPrint, "a[1].? == .print")
		expectEqual(t, uint32(c), zigPayload(t, a[1], ParserActionPrint).Print)
		expect(t, !a[2].Valid, "a[2] == null")
		expectEqual(t, ParserStateGround, p.state)
	}
}

// Zig: test "term51: next: DEL in ground"
func TestParser_term51_next_DEL_in_ground(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	a := p.Next(0x7F)
	expectEqual(t, ParserStateGround, p.state)
	expect(t, !a[0].Valid, "a[0] == null")
	expect(t, !a[2].Valid, "a[2] == null")
	// The parser itself prints DEL in ground; it is the stream that
	// discards it later.
	expect(t, zigUnwrap(t, a[1]).Tag == ParserActionPrint, "a[1].? == .print")
	expectEqual(t, uint32(0x7F), zigPayload(t, a[1], ParserActionPrint).Print)
}

// Zig: test "term51: next: C0 controls execute in ground"
func TestParser_term51_next_C0_controls_execute_in_ground(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	for c := uint8(0); c < 0x20; c += 1 {
		if c == 0x1B {
			continue // ESC starts a sequence
		}
		a := p.Next(c)
		expectEqual(t, ParserStateGround, p.state)
		expect(t, zigUnwrap(t, a[1]).Tag == ParserActionExecute, "a[1].? == .execute")
		expectEqual(t, c, zigPayload(t, a[1], ParserActionExecute).Execute)
	}
}

// Zig: test "term51: next: CAN and SUB return to ground from every state and execute"
func TestParser_term51_next_CAN_and_SUB_return_to_ground_from_every_state_and_execute(t *testing.T) {
	prefixes := []string{
		"", "\x1b", "\x1b ", "\x1b[", "\x1b[1", "\x1b[1 ", "\x1b[1 1",
		"\x1bP", "\x1bP1", "\x1bP1 ", "\x1bP1 q", "\x1bP1 1", "\x1b]2;t", "\x1bX",
		"\x1b_G", "\x1b^",
	}
	for _, prefix := range prefixes {
		for _, c := range []uint8{0x18, 0x1A} {
			p := ParserInit()
			defer p.Deinit()
			_ = term51Feed(&p, prefix)
			a := p.Next(c)
			expectEqual(t, ParserStateGround, p.state)
			expect(t, zigUnwrap(t, a[1]).Tag == ParserActionExecute, "a[1].? == .execute")
			expectEqual(t, c, zigPayload(t, a[1], ParserActionExecute).Execute)
		}
	}
}

// Zig: test "term51: next: ESC enters escape from every state and clears"
func TestParser_term51_next_ESC_enters_escape_from_every_state_and_clears(t *testing.T) {
	prefixes := []string{
		"", "\x1b", "\x1b ", "\x1b[", "\x1b[1;2", "\x1b[1 ", "\x1b[1 1",
		"\x1bP", "\x1bP1", "\x1bP1 ", "\x1bP1 q", "\x1bP1 1", "\x1b]2;t", "\x1bX",
		"\x1b_G", "\x1b^",
	}
	for _, prefix := range prefixes {
		p := ParserInit()
		defer p.Deinit()
		_ = term51Feed(&p, prefix)
		_ = p.Next(0x1B)
		expectEqual(t, ParserStateEscape, p.state)
		expectEqual(t, uint8(0), p.intermediates_idx)
		expectEqual(t, uint8(0), p.params_idx)
		expectEqual(t, uint8(0), p.param_acc_idx)
		expectEqual(t, uint16(0), p.param_acc)
	}
}

// "term51: next: exit, transition and entry actions occupy slots 0, 1 and 2"
// follows here in Parser.zig; its last block expects an osc_dispatch, so the
// test is in Parser_osc_test.go.

// Zig: test "term51: next: SOS and PM strings also report apc_start and apc_end"
func TestParser_term51_next_SOS_and_PM_strings_also_report_apc_start_and_apc_end(t *testing.T) {
	for _, intro := range []uint8{'X', '^'} {
		p := ParserInit()
		defer p.Deinit()
		_ = p.Next(0x1B)
		start := p.Next(intro)
		expectEqual(t, ParserStateSosPmApcString, p.state)
		expect(t, zigUnwrap(t, start[2]).Tag == ParserActionApcStart, "start[2].? == .apc_start")
		end := term51Feed(&p, "x\x1b")
		expect(t, zigUnwrap(t, end[0]).Tag == ParserActionApcEnd, "end[0].? == .apc_end")
	}
}

// Zig: test "term51: next: parameter count boundary at MAX_PARAMS"
func TestParser_term51_next_parameter_count_boundary_at_MAX_PARAMS(t *testing.T) {
	// Exactly MAX_PARAMS parameters dispatch.
	{
		p := ParserInit()
		defer p.Deinit()
		_ = term51Feed(&p, "\x1b[")
		for range MAX_PARAMS - 1 {
			_ = term51Feed(&p, "1;")
		}
		a := term51Feed(&p, "7H")
		expect(t, zigUnwrap(t, a[1]).Tag == ParserActionCsiDispatch, "a[1].? == .csi_dispatch")
		expectEqual(t, MAX_PARAMS, len(zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch.Params))
		expectEqual(t, uint16(7), zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch.Params[MAX_PARAMS-1])
	}
	// One more parameter and the whole sequence is dropped.
	{
		p := ParserInit()
		defer p.Deinit()
		_ = term51Feed(&p, "\x1b[")
		for range MAX_PARAMS {
			_ = term51Feed(&p, "1;")
		}
		a := term51Feed(&p, "7H")
		expect(t, !a[1].Valid, "a[1] == null")
		expectEqual(t, ParserStateGround, p.state)
	}
	// MAX_PARAMS values followed by a trailing separator is also dropped.
	{
		p := ParserInit()
		defer p.Deinit()
		_ = term51Feed(&p, "\x1b[")
		for range MAX_PARAMS {
			_ = term51Feed(&p, "1;")
		}
		a := p.Next('H')
		expect(t, !a[1].Valid, "a[1] == null")
	}
}

// Zig: test "term51: next: parameter value saturates at 65535"
func TestParser_term51_next_parameter_value_saturates_at_65535(t *testing.T) {
	// Zig: an array of tuples struct { []const u8, u16 }; f0 and f1 are
	// case[0] and case[1].
	cases := []struct {
		f0 string
		f1 uint16
	}{
		{"0", 0},
		{"65534", 65534},
		{"65535", 65535},
		{"65536", 65535},
		{"99999", 65535},
		{"4294967296", 65535},
		{"00000000000000000007", 7},
	}
	for _, case_ := range cases {
		p := ParserInit()
		defer p.Deinit()
		_ = term51Feed(&p, "\x1b[")
		_ = term51Feed(&p, case_.f0)
		a := p.Next('A')
		expectEqual(t, 1, len(zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch.Params))
		expectEqual(t, case_.f1, zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch.Params[0])
	}
}

// Zig: test "term51: next: a final parameter of exactly 256 digits is dropped"
func TestParser_term51_next_a_final_parameter_of_exactly_256_digits_is_dropped(t *testing.T) {
	// The digit counter is a u8 that wraps. A final parameter is only
	// appended when the counter is non-zero, so a parameter whose digit
	// count is a multiple of 256 disappears.
	type Case struct {
		digits int
		params int
	}
	for _, case_ := range []Case{
		{digits: 1, params: 1},
		{digits: 255, params: 1},
		{digits: 256, params: 0},
		{digits: 257, params: 1},
		{digits: 511, params: 1},
		{digits: 512, params: 0},
		{digits: 513, params: 1},
	} {
		p := ParserInit()
		defer p.Deinit()
		_ = term51Feed(&p, "\x1b[")
		for range case_.digits {
			_ = p.Next('1')
		}
		a := p.Next('A')
		expect(t, zigUnwrap(t, a[1]).Tag == ParserActionCsiDispatch, "a[1].? == .csi_dispatch")
		expectEqual(t, case_.params, len(zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch.Params))
		if case_.params == 1 {
			var want uint16
			if case_.digits == 1 {
				want = 1
			} else {
				want = 65535
			}
			expectEqual(t, want, zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch.Params[0])
		}
	}
}

// Zig: test "term51: next: a non-final parameter of 256 digits is kept"
func TestParser_term51_next_a_non_final_parameter_of_256_digits_is_kept(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	_ = term51Feed(&p, "\x1b[")
	for range 256 {
		_ = p.Next('1')
	}
	a := term51Feed(&p, ";2H")
	expectEqual(t, 2, len(zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch.Params))
	expectEqual(t, uint16(65535), zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch.Params[0])
	expectEqual(t, uint16(2), zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch.Params[1])
}

// Zig: test "term51: next: empty parameters are zero"
func TestParser_term51_next_empty_parameters_are_zero(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	a := term51Feed(&p, "\x1b[;;5H")
	csi := zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch
	expectEqual(t, 3, len(csi.Params))
	expectEqualSlices(t, []uint16{0, 0, 5}, csi.Params)

	// A trailing separator does not add a parameter.
	p2 := ParserInit()
	defer p2.Deinit()
	b := term51Feed(&p2, "\x1b[5;H")
	expectEqualSlices(t, []uint16{5}, zigPayload(t, b[1], ParserActionCsiDispatch).CsiDispatch.Params)

	// No parameters at all.
	p3 := ParserInit()
	defer p3.Deinit()
	c := term51Feed(&p3, "\x1b[H")
	expectEqual(t, 0, len(zigPayload(t, c[1], ParserActionCsiDispatch).CsiDispatch.Params))
}

// Zig: test "term51: next: colon separators are recorded per parameter and only allowed for m"
func TestParser_term51_next_colon_separators_are_recorded_per_parameter_and_only_allowed_for_m(t *testing.T) {
	{
		p := ParserInit()
		defer p.Deinit()
		a := term51Feed(&p, "\x1b[1;4:3;31m")
		csi := zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch
		expectEqualSlices(t, []uint16{1, 4, 3, 31}, csi.Params)
		expect(t, !csi.ParamsSep.isSet(0), "!csi.params_sep.isSet(0)")
		expect(t, csi.ParamsSep.isSet(1), "csi.params_sep.isSet(1)")
		expect(t, !csi.ParamsSep.isSet(2), "!csi.params_sep.isSet(2)")
		expectEqual(t, 1, csi.ParamsSep.count())
	}
	// Any other final byte with a colon drops the sequence.
	for final := uint8(0x40); final <= 0x7E; final += 1 {
		if final == 'm' {
			continue
		}
		p := ParserInit()
		defer p.Deinit()
		_ = term51Feed(&p, "\x1b[1:2")
		a := p.Next(final)
		expect(t, !a[1].Valid, "a[1] == null")
		expectEqual(t, ParserStateGround, p.state)
	}
}

// Zig: test "term51: next: DCS hook carries params and intermediates, and is dropped past MAX_PARAMS"
func TestParser_term51_next_DCS_hook_carries_params_and_intermediates_and_is_dropped_past_MAX_PARAMS(t *testing.T) {
	{
		p := ParserInit()
		defer p.Deinit()
		a := term51Feed(&p, "\x1bP1;2$q")
		hook := zigPayload(t, a[2], ParserActionDcsHook).DcsHook
		expectEqualSlices(t, []uint16{1, 2}, hook.Params)
		expectEqualStrings(t, "$", string(hook.Intermediates))
		expectEqual(t, uint8('q'), hook.Final)
	}
	{
		p := ParserInit()
		defer p.Deinit()
		_ = term51Feed(&p, "\x1bP")
		for range MAX_PARAMS {
			_ = term51Feed(&p, "1;")
		}
		a := p.Next('q')
		expectEqual(t, ParserStateDcsPassthrough, p.state)
		expect(t, !a[2].Valid, "a[2] == null")
	}
}

// Zig: test "term51: next: every byte from every state lands in the state the table names"
func TestParser_term51_next_every_byte_from_every_state_lands_in_the_state_the_table_names(t *testing.T) {
	// Zig: an array of tuples struct { State, []const u8 }; f0 and f1 are
	// entry[0] and entry[1].
	prefixes := []struct {
		f0 ParserState
		f1 string
	}{
		{ParserStateGround, ""},
		{ParserStateEscape, "\x1b"},
		{ParserStateEscapeIntermediate, "\x1b "},
		{ParserStateCsiEntry, "\x1b["},
		{ParserStateCsiParam, "\x1b[1"},
		{ParserStateCsiIntermediate, "\x1b[1 "},
		{ParserStateCsiIgnore, "\x1b[1 1"},
		{ParserStateDcsEntry, "\x1bP"},
		{ParserStateDcsParam, "\x1bP1"},
		{ParserStateDcsIntermediate, "\x1bP1 "},
		{ParserStateDcsPassthrough, "\x1bP1 q"},
		{ParserStateDcsIgnore, "\x1bP1 1"},
		{ParserStateOscString, "\x1b]2;t"},
		{ParserStateSosPmApcString, "\x1bX"},
	}
	for _, entry := range prefixes {
		for byte_ := 0; byte_ < 256; byte_++ {
			p := ParserInit()
			defer p.Deinit()
			_ = term51Feed(&p, entry.f1)
			expectEqual(t, entry.f0, p.state)
			c := uint8(byte_)
			want := table[c][int(entry.f0)].state
			_ = p.Next(c)
			expectEqual(t, want, p.state)
		}
	}
}

// Zig: test "term51: collect: stores up to MAX_INTERMEDIATE bytes and ignores the rest"
func TestParser_term51_collect_stores_up_to_MAX_INTERMEDIATE_bytes_and_ignores_the_rest(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	p.Collect('a')
	expectEqual(t, uint8(1), p.intermediates_idx)
	expectEqual(t, uint8('a'), p.intermediates[0])
	p.Collect('b')
	p.Collect('c')
	p.Collect('d')
	expectEqual(t, uint8(MAX_INTERMEDIATE), p.intermediates_idx)
	expectEqualStrings(t, "abcd", string(p.intermediates[0:p.intermediates_idx]))
	// One past the limit: ignored, earlier bytes untouched.
	p.Collect('e')
	p.Collect('f')
	expectEqual(t, uint8(MAX_INTERMEDIATE), p.intermediates_idx)
	expectEqualStrings(t, "abcd", string(p.intermediates[0:p.intermediates_idx]))
}

// Zig: test "term51: collect: through next, a fifth intermediate is dropped but the sequence still dispatches"
func TestParser_term51_collect_through_next_a_fifth_intermediate_is_dropped_but_the_sequence_still_dispatches(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	a := term51Feed(&p, "\x1b[1 !\"#$q")
	csi := zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch
	expectEqualStrings(t, " !\"#", string(csi.Intermediates))
	expectEqual(t, uint8('q'), csi.Final)
}

// Zig: test "term51: doAction: none and ignore produce nothing"
func TestParser_term51_doAction_none_and_ignore_produce_nothing(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	expect(t, !p.doAction(ParserTransitionActionNone, 'x').Valid, "p.doAction(.none, 'x') == null")
	expect(t, !p.doAction(ParserTransitionActionIgnore, 'x').Valid, "p.doAction(.ignore, 'x') == null")
	expectEqual(t, uint8(0), p.intermediates_idx)
	expectEqual(t, uint8(0), p.params_idx)
}

// Zig: test "term51: doAction: print, execute, put and apc_put carry the byte"
func TestParser_term51_doAction_print_execute_put_and_apc_put_carry_the_byte(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	expectEqual(t, uint32('x'), zigPayload(t, p.doAction(ParserTransitionActionPrint, 'x'), ParserActionPrint).Print)
	expectEqual(t, uint32(0xFF), zigPayload(t, p.doAction(ParserTransitionActionPrint, 0xFF), ParserActionPrint).Print)
	expectEqual(t, uint8(0x07), zigPayload(t, p.doAction(ParserTransitionActionExecute, 0x07), ParserActionExecute).Execute)
	expectEqual(t, uint8('d'), zigPayload(t, p.doAction(ParserTransitionActionPut, 'd'), ParserActionDcsPut).DcsPut)
	expectEqual(t, uint8('a'), zigPayload(t, p.doAction(ParserTransitionActionApcPut, 'a'), ParserActionApcPut).ApcPut)
}

// Zig: test "term51: doAction: collect appends an intermediate and returns nothing"
func TestParser_term51_doAction_collect_appends_an_intermediate_and_returns_nothing(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	expect(t, !p.doAction(ParserTransitionActionCollect, '?').Valid, "p.doAction(.collect, '?') == null")
	expectEqualStrings(t, "?", string(p.intermediates[0:p.intermediates_idx]))
}

// Zig: test "term51: doAction: param accumulates digits and commits on a separator"
func TestParser_term51_doAction_param_accumulates_digits_and_commits_on_a_separator(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	expect(t, !p.doAction(ParserTransitionActionParam, '1').Valid, "p.doAction(.param, '1') == null")
	expect(t, !p.doAction(ParserTransitionActionParam, '2').Valid, "p.doAction(.param, '2') == null")
	expectEqual(t, uint16(12), p.param_acc)
	expectEqual(t, uint8(2), p.param_acc_idx)
	expectEqual(t, uint8(0), p.params_idx)

	expect(t, !p.doAction(ParserTransitionActionParam, ';').Valid, "p.doAction(.param, ';') == null")
	expectEqual(t, uint8(1), p.params_idx)
	expectEqual(t, uint16(12), p.params[0])
	expectEqual(t, uint16(0), p.param_acc)
	expectEqual(t, uint8(0), p.param_acc_idx)
	expect(t, !p.params_sep.isSet(0), "!p.params_sep.isSet(0)")

	_ = p.doAction(ParserTransitionActionParam, '7')
	expect(t, !p.doAction(ParserTransitionActionParam, ':').Valid, "p.doAction(.param, ':') == null")
	expectEqual(t, uint8(2), p.params_idx)
	expectEqual(t, uint16(7), p.params[1])
	expect(t, p.params_sep.isSet(1), "p.params_sep.isSet(1)")
}

// Zig: test "term51: doAction: param separators past MAX_PARAMS are ignored"
func TestParser_term51_doAction_param_separators_past_MAX_PARAMS_are_ignored(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	for range MAX_PARAMS {
		_ = p.doAction(ParserTransitionActionParam, ';')
	}
	expectEqual(t, uint8(MAX_PARAMS), p.params_idx)
	_ = p.doAction(ParserTransitionActionParam, '9')
	_ = p.doAction(ParserTransitionActionParam, ';')
	expectEqual(t, uint8(MAX_PARAMS), p.params_idx)
	// The pending value is left in the accumulator, not committed.
	expectEqual(t, uint16(9), p.param_acc)
}

// Zig: test "term51: doAction: csi_dispatch commits the pending parameter"
func TestParser_term51_doAction_csi_dispatch_commits_the_pending_parameter(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	_ = p.doAction(ParserTransitionActionCollect, '?')
	_ = p.doAction(ParserTransitionActionParam, '2')
	_ = p.doAction(ParserTransitionActionParam, '5')
	a := zigUnwrap(t, p.doAction(ParserTransitionActionCsiDispatch, 'h'))
	expect(t, a.Tag == ParserActionCsiDispatch, "a == .csi_dispatch")
	expectEqualStrings(t, "?", string(zigField(t, a, ParserActionCsiDispatch).CsiDispatch.Intermediates))
	expectEqualSlices(t, []uint16{25}, zigField(t, a, ParserActionCsiDispatch).CsiDispatch.Params)
	expectEqual(t, uint8('h'), zigField(t, a, ParserActionCsiDispatch).CsiDispatch.Final)
}

// Zig: test "term51: doAction: esc_dispatch carries intermediates and final"
func TestParser_term51_doAction_esc_dispatch_carries_intermediates_and_final(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	_ = p.doAction(ParserTransitionActionCollect, '(')
	a := zigUnwrap(t, p.doAction(ParserTransitionActionEscDispatch, '0'))
	expect(t, a.Tag == ParserActionEscDispatch, "a == .esc_dispatch")
	expectEqualStrings(t, "(", string(zigField(t, a, ParserActionEscDispatch).EscDispatch.Intermediates))
	expectEqual(t, uint8('0'), zigField(t, a, ParserActionEscDispatch).EscDispatch.Final)
}

// "term51: doAction: osc_put feeds the OSC parser" follows here in
// Parser.zig; it expects a command back from the OSC parser, so the test is
// in Parser_osc_test.go.

// Zig: test "term51: clear: resets accumulators but not the state"
func TestParser_term51_clear_resets_accumulators_but_not_the_state(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	_ = term51Feed(&p, "\x1b[?1;2:3")
	expectEqual(t, ParserStateCsiParam, p.state)
	expect(t, p.intermediates_idx > 0, "p.intermediates_idx > 0")
	expect(t, p.params_idx > 0, "p.params_idx > 0")
	expect(t, p.params_sep.count() > 0, "p.params_sep.count() > 0")
	expect(t, p.param_acc_idx > 0, "p.param_acc_idx > 0")

	p.Clear()
	expectEqual(t, ParserStateCsiParam, p.state)
	expectEqual(t, uint8(0), p.intermediates_idx)
	expectEqual(t, uint8(0), p.params_idx)
	expectEqual(t, 0, p.params_sep.count())
	expectEqual(t, uint16(0), p.param_acc)
	expectEqual(t, uint8(0), p.param_acc_idx)
}

// Zig: test "term51: clear: idempotent on a fresh parser"
func TestParser_term51_clear_idempotent_on_a_fresh_parser(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	p.Clear()
	p.Clear()
	expectEqual(t, ParserStateGround, p.state)
	expectEqual(t, uint8(0), p.params_idx)
}

// Zig: test "term51: Action.CSI.format: prints intermediates, params and final"
func TestParser_term51_Action_CSI_format_prints_intermediates_params_and_final(t *testing.T) {
	inter := [...]uint8{'?'}
	params := [...]uint16{1, 22}
	csi := ParserActionCSI{
		Intermediates: inter[:],
		Params:        params[:],
		ParamsSep:     StaticBitSet24InitEmpty(),
		Final:         'h',
	}
	// Zig: std.fmt.bufPrint(&buf, "{f}", .{csi}) into a [128]u8 buffer.
	s := csi.Format()
	expectEqualStrings(t, "ESC [ ? { 1, 22 } h", s)
}

// Zig: test "term51: Action.CSI.format: no intermediates and no params"
func TestParser_term51_Action_CSI_format_no_intermediates_and_no_params(t *testing.T) {
	csi := ParserActionCSI{
		Intermediates: []uint8{},
		Params:        []uint16{},
		ParamsSep:     StaticBitSet24InitEmpty(),
		Final:         'H',
	}
	// Zig: std.fmt.bufPrint(&buf, "{f}", .{csi}) into a [128]u8 buffer.
	s := csi.Format()
	expectEqualStrings(t, "ESC [  {  } H", s)
}

// Zig: test "term51: Action.ESC.format: prints intermediates and final"
func TestParser_term51_Action_ESC_format_prints_intermediates_and_final(t *testing.T) {
	inter := [...]uint8{'('}
	esc := ParserActionESC{Intermediates: inter[:], Final: '0'}
	// Zig: std.fmt.bufPrint(&buf, "{f}", .{esc}) into a [64]u8 buffer.
	expectEqualStrings(t, "ESC ( 0", esc.Format())

	bare := ParserActionESC{Intermediates: []uint8{}, Final: 'M'}
	expectEqualStrings(t, "ESC  M", bare.Format())
}

// Zig: test "term51: Action.format: print, execute and payloads without a custom format"
func TestParser_term51_Action_format_print_execute_and_payloads_without_a_custom_format(t *testing.T) {
	// Zig: std.fmt.bufPrint(&buf, "{f}", .{a}) into a [256]u8 buffer.
	{
		a := ParserAction{Tag: ParserActionPrint, Print: 'A'}
		expectEqualStrings(t,
			"terminal.Parser.Action{ .print = 'A' (U+41) }",
			a.Format(),
		)
	}
	{
		a := ParserAction{Tag: ParserActionPrint, Print: 0x4E2D}
		expectEqualStrings(t,
			"terminal.Parser.Action{ .print = '中' (U+4E2D) }",
			a.Format(),
		)
	}
	{
		a := ParserAction{Tag: ParserActionExecute, Execute: 0x07}
		expectEqualStrings(t,
			"terminal.Parser.Action{ .execute = 0x7 }",
			a.Format(),
		)
	}
	{
		a := ParserAction{Tag: ParserActionDcsPut, DcsPut: 0xFF}
		expectEqualStrings(t,
			"terminal.Parser.Action{ .dcs_put = 0xff }",
			a.Format(),
		)
	}
	{
		// A nested CSI is printed field by field, not through CSI.format.
		params := [...]uint16{5}
		a := ParserAction{Tag: ParserActionCsiDispatch, CsiDispatch: ParserActionCSI{
			Intermediates: []uint8{},
			Params:        params[:],
			ParamsSep:     StaticBitSet24InitEmpty(),
			Final:         'A',
		}}
		expectEqualStrings(t,
			"terminal.Parser.Action{ .csi_dispatch = .{ .intermediates = {  }, .params = { 5 }, .params_sep = .{ .mask = 0 }, .final = 65 } }",
			a.Format(),
		)
	}
	{
		a := ParserAction{Tag: ParserActionDcsUnhook}
		expectEqualStrings(t,
			"terminal.Parser.Action{ .dcs_unhook = void }",
			a.Format(),
		)
	}
}

// term51: boundary cases requested in review.

// Zig: test "term51: next: a leading colon invalidates the sequence"
func TestParser_term51_next_a_leading_colon_invalidates_the_sequence(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	_ = term51Feed(&p, "\x1b[:")
	expectEqual(t, ParserStateCsiIgnore, p.state)
	a := term51Feed(&p, "1m")
	expect(t, !a[1].Valid, "a[1] == null")
	expectEqual(t, ParserStateGround, p.state)
}

// Zig: test "term51: next: a trailing colon marks the last parameter and adds no empty one"
func TestParser_term51_next_a_trailing_colon_marks_the_last_parameter_and_adds_no_empty_one(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	a := term51Feed(&p, "\x1b[4:m")
	csi := zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch
	expectEqualSlices(t, []uint16{4}, csi.Params)
	expect(t, csi.ParamsSep.isSet(0), "csi.params_sep.isSet(0)")
	expectEqual(t, 1, csi.ParamsSep.count())
}

// Zig: test "term51: next: two colons in a row give an empty parameter between them"
func TestParser_term51_next_two_colons_in_a_row_give_an_empty_parameter_between_them(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	a := term51Feed(&p, "\x1b[38::5m")
	csi := zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch
	expectEqualSlices(t, []uint16{38, 0, 5}, csi.Params)
	expect(t, csi.ParamsSep.isSet(0), "csi.params_sep.isSet(0)")
	expect(t, csi.ParamsSep.isSet(1), "csi.params_sep.isSet(1)")
	expect(t, !csi.ParamsSep.isSet(2), "!csi.params_sep.isSet(2)")
}

// Zig: test "term51: next: a colon after a semicolon-separated parameter"
func TestParser_term51_next_a_colon_after_a_semicolon_separated_parameter(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	a := term51Feed(&p, "\x1b[1;4:3m")
	csi := zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch
	expectEqualSlices(t, []uint16{1, 4, 3}, csi.Params)
	expect(t, !csi.ParamsSep.isSet(0), "!csi.params_sep.isSet(0)")
	expect(t, csi.ParamsSep.isSet(1), "csi.params_sep.isSet(1)")
}

// Zig: test "term51: next: a colon at the last allowed parameter position"
func TestParser_term51_next_a_colon_at_the_last_allowed_parameter_position(t *testing.T) {
	// MAX_PARAMS - 1 semicolons, then "7:" fills the last slot with a colon.
	p := ParserInit()
	defer p.Deinit()
	_ = term51Feed(&p, "\x1b[")
	for range MAX_PARAMS - 1 {
		_ = term51Feed(&p, "1;")
	}
	_ = term51Feed(&p, "7:")
	expectEqual(t, uint8(MAX_PARAMS), p.params_idx)
	expect(t, p.params_sep.isSet(MAX_PARAMS-1), "p.params_sep.isSet(MAX_PARAMS - 1)")
	// The list is full, so the sequence is dropped at dispatch.
	a := p.Next('m')
	expect(t, !a[1].Valid, "a[1] == null")
}

// Zig: test "term51: next: a colon one past the last allowed parameter position is ignored"
func TestParser_term51_next_a_colon_one_past_the_last_allowed_parameter_position_is_ignored(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	_ = term51Feed(&p, "\x1b[")
	for range MAX_PARAMS {
		_ = term51Feed(&p, "1;")
	}
	expectEqual(t, uint8(MAX_PARAMS), p.params_idx)
	before := p.params_sep.count()
	_ = term51Feed(&p, "7:")
	// no slot left: neither the value nor the colon is recorded
	expectEqual(t, uint8(MAX_PARAMS), p.params_idx)
	expectEqual(t, before, p.params_sep.count())
	a := p.Next('m')
	expect(t, !a[1].Valid, "a[1] == null")
}

// Zig: test "term51: next: the last parameter with a colon before it still dispatches at exactly MAX_PARAMS"
func TestParser_term51_next_the_last_parameter_with_a_colon_before_it_still_dispatches_at_exactly_MAX_PARAMS(t *testing.T) {
	// MAX_PARAMS - 2 semicolons, then "4:3": 24 parameters, colon at index 22.
	p := ParserInit()
	defer p.Deinit()
	_ = term51Feed(&p, "\x1b[")
	for range MAX_PARAMS - 2 {
		_ = term51Feed(&p, "1;")
	}
	a := term51Feed(&p, "4:3m")
	csi := zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch
	expectEqual(t, MAX_PARAMS, len(csi.Params))
	expect(t, csi.ParamsSep.isSet(MAX_PARAMS-2), "csi.params_sep.isSet(MAX_PARAMS - 2)")
	expectEqual(t, uint16(3), csi.Params[MAX_PARAMS-1])
}

// Zig: test "term51: next: a 256-digit parameter followed by a colon is kept"
func TestParser_term51_next_a_256_digit_parameter_followed_by_a_colon_is_kept(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	_ = term51Feed(&p, "\x1b[")
	for range 256 {
		_ = p.Next('1')
	}
	a := term51Feed(&p, ":3m")
	csi := zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch
	expectEqualSlices(t, []uint16{65535, 3}, csi.Params)
	expect(t, csi.ParamsSep.isSet(0), "csi.params_sep.isSet(0)")
}

// Zig: test "term51: next: a 256-digit final parameter after a colon is dropped"
func TestParser_term51_next_a_256_digit_final_parameter_after_a_colon_is_dropped(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	_ = term51Feed(&p, "\x1b[4:")
	for range 256 {
		_ = p.Next('1')
	}
	a := p.Next('m')
	csi := zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch
	// only the 4 survives; its colon is still recorded
	expectEqualSlices(t, []uint16{4}, csi.Params)
	expect(t, csi.ParamsSep.isSet(0), "csi.params_sep.isSet(0)")
}

// Zig: test "term51: next: digit-count boundaries of the final parameter"
func TestParser_term51_next_digit_count_boundaries_of_the_final_parameter(t *testing.T) {
	type Case struct {
		digits int
		kept   bool
	}
	for _, case_ := range []Case{
		{digits: 254, kept: true},
		{digits: 255, kept: true},
		{digits: 256, kept: false},
		{digits: 257, kept: true},
		{digits: 511, kept: true},
		{digits: 512, kept: false},
		{digits: 513, kept: true},
		{digits: 768, kept: false},
	} {
		p := ParserInit()
		defer p.Deinit()
		_ = term51Feed(&p, "\x1b[")
		for range case_.digits {
			_ = p.Next('9')
		}
		a := p.Next('H')
		csi := zigPayload(t, a[1], ParserActionCsiDispatch).CsiDispatch
		if case_.kept {
			expectEqualSlices(t, []uint16{65535}, csi.Params)
		} else {
			expectEqual(t, 0, len(csi.Params))
		}
	}
}

// Zig: test "term51: next: the 256-digit drop also applies to the DCS hook"
func TestParser_term51_next_the_256_digit_drop_also_applies_to_the_DCS_hook(t *testing.T) {
	type Case struct {
		digits int
		kept   bool
	}
	for _, case_ := range []Case{
		{digits: 255, kept: true},
		{digits: 256, kept: false},
		{digits: 257, kept: true},
		{digits: 512, kept: false},
	} {
		p := ParserInit()
		defer p.Deinit()
		_ = term51Feed(&p, "\x1bP")
		for range case_.digits {
			_ = p.Next('1')
		}
		a := p.Next('q')
		hook := zigPayload(t, a[2], ParserActionDcsHook).DcsHook
		// Zig: @as(usize, if (case.kept) 1 else 0)
		var want int
		if case_.kept {
			want = 1
		} else {
			want = 0
		}
		expectEqual(t, want, len(hook.Params))
		if case_.kept {
			expectEqual(t, uint16(65535), hook.Params[0])
		}
	}
}

// Zig: test "term51: next: a 256-digit DCS parameter followed by a separator is kept"
func TestParser_term51_next_a_256_digit_DCS_parameter_followed_by_a_separator_is_kept(t *testing.T) {
	p := ParserInit()
	defer p.Deinit()
	_ = term51Feed(&p, "\x1bP")
	for range 256 {
		_ = p.Next('1')
	}
	a := term51Feed(&p, ";2q")
	expectEqualSlices(t, []uint16{65535, 2}, zigPayload(t, a[2], ParserActionDcsHook).DcsHook.Params)
}

// Zig: test "term51: next: a colon in DCS parameters invalidates the sequence"
func TestParser_term51_next_a_colon_in_DCS_parameters_invalidates_the_sequence(t *testing.T) {
	// Unlike CSI, the DCS states have no colon separator: both dcs_entry
	// and dcs_param go to dcs_ignore, and no hook is produced.
	for _, prefix := range []string{"\x1bP:", "\x1bP1:", "\x1bP1;2:"} {
		p := ParserInit()
		defer p.Deinit()
		_ = term51Feed(&p, prefix)
		expectEqual(t, ParserStateDcsIgnore, p.state)
		a := term51Feed(&p, "2q")
		expect(t, !a[2].Valid, "a[2] == null")
		expectEqual(t, ParserStateDcsIgnore, p.state)
		// the string still ends at ST
		_ = term51Feed(&p, "\x1b\\")
		expectEqual(t, ParserStateGround, p.state)
	}
}
