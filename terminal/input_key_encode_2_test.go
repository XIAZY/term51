// Tests translated one for one from src/input/key_encode.zig (Ghostty
// v1.3.1 plus the term51 additional tests), lines 2319 to 4048: the last
// legacy tests (hu layout, backspace with DEL), the ctrlSeq tests, and the
// term51 tests from "Options.fromTerminal" through "encode: disabled kitty
// flags use legacy, any flag uses kitty" (tests 69 to 136). Each Go test
// carries the exact Zig title in a "// Zig: test" comment. Helpers are
// prefixed ke2.
//
// Translation notes:
//   - A KeyEvent's `action` defaults to .press in Zig, which is not the Go
//     zero value: every InputKeyKeyEvent literal in this file says
//     `action: InputKeyActionPress` (or the explicit action of the Zig
//     literal). All encoder options default to the Go zero value.
//   - `term51ABExpect`, `term51ABExpectSeq`, `term51ABRunCases` and the
//     table helpers are ke2Expect, ke2ExpectSeq, ke2RunCases, ke2Rows, ke2Row,
//     ke2Mods and ke2KittyValue; the term51AB* option constants are the
//     package variables ke2Dis, ke2Events, ... ke2Full.
//   - `kitty`, `legacy` and `encode` are inputKeyEncodeKitty,
//     inputKeyEncodeLegacy and InputKeyEncodeEncode; `try testing.expect*`
//     are expect / expectEqual / expectEqualStrings (Zig's argument order),
//     `x.?` is zigUnwrap, `try f()` on an error result is zigTry.
//   - `@bitCast` of an integer is the ...FromInt stub of the wireframe,
//     `.int()` is Int. Zig's `u3`, `u4`, `u5` and `u9` integers are uint8
//     (uint16 for u9).
//   - The Terminal that Zig's tests call `t` is called `term`, because the
//     testing parameter must be `t`.
//   - Darwin: the two tests that are skipped off macOS are not translated;
//     the two tests that branch on `builtin.os.tag == .macos` follow the
//     non-Darwin branch, with `// Zig-only (Darwin):` comments for the lines
//     that exist only there. See scripts/not-translated-ke2.txt.

package terminal

import (
	"errors"
	"fmt"
	"testing"
)

// ke2Func is the shape of `kitty`, `legacy` and `encode`, the functions that
// term51ABExpect takes as `comptime f: anytype`.
type ke2Func func(writer *oscWriter, event InputKeyKeyEvent, opts InputKeyEncodeOptions) error

// ke2ExpectWriteFailed is `try std.testing.expectError(error.WriteFailed, err)`.
func ke2ExpectWriteFailed(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, ErrWriteFailed) {
		t.Fatalf("expected error.WriteFailed, found %v", err)
	}
}

// ke2Expect is term51ABExpect: runs f into a roomy buffer and checks the
// exact output, then re-runs it with every smaller capacity (0..len-1) and
// expects error.WriteFailed, so that every write point of the encoder is
// checked for a too-small buffer.
func ke2Expect(t *testing.T, f ke2Func, expected string, event InputKeyKeyEvent, opts InputKeyEncodeOptions) {
	t.Helper()
	var big [512]uint8
	w := oscWriterFixed(big[:])
	zigTry(t, f(&w, event, opts))
	expectEqualStrings(t, expected, string(w.buffered()))

	for capacity := 0; capacity <= len(expected); capacity++ {
		var small [512]uint8
		w2 := oscWriterFixed(small[0:capacity])
		if capacity < len(expected) {
			ke2ExpectWriteFailed(t, f(&w2, event, opts))
		} else {
			zigTry(t, f(&w2, event, opts))
			expectEqualStrings(t, expected, string(w2.buffered()))
		}
	}
}

// ke2ExpectSeq is term51ABExpectSeq: the same as ke2Expect but for a
// KittySequence.
func ke2ExpectSeq(t *testing.T, expected string, seq inputKeyEncodeKittySequence) {
	t.Helper()
	var big [512]uint8
	w := oscWriterFixed(big[:])
	zigTry(t, seq.Encode(&w))
	expectEqualStrings(t, expected, string(w.buffered()))

	for capacity := 0; capacity <= len(expected); capacity++ {
		var small [512]uint8
		w2 := oscWriterFixed(small[0:capacity])
		if capacity < len(expected) {
			ke2ExpectWriteFailed(t, seq.Encode(&w2))
		} else {
			zigTry(t, seq.Encode(&w2))
			expectEqualStrings(t, expected, string(w2.buffered()))
		}
	}
}

// ke2Case is term51ABCase.
type ke2Case struct {
	ev   InputKeyKeyEvent
	opts InputKeyEncodeOptions
	want string
}

// ke2RunCases is term51ABRunCases: on a failure it reports the case index,
// as Zig's std.debug.print does.
func ke2RunCases(t *testing.T, f ke2Func, cases []ke2Case) {
	t.Helper()
	for i, c := range cases {
		ok := false
		func() {
			defer func() {
				if !ok {
					t.Logf("term51AB case index %d failed", i)
				}
			}()
			ke2Expect(t, f, c.want, c.ev, c.opts)
			ok = true
		}()
	}
}

// ke2Terminal is `var t: Terminal = try .init(alloc, .{ .cols = 80, .rows = 24 })`.
func ke2Terminal(t *testing.T) *Terminal {
	t.Helper()
	term, err := TerminalInit(terminalOptionsDefault(80, 24))
	zigTry(t, err)
	return &term
}

// ke2AltPrevents is term51ABAltPrevents: on macOS the Option key only
// prevents associated text when it is configured as alt (for the matching
// side); elsewhere alt always does.
func ke2AltPrevents(oa InputConfigOptionAsAlt, side InputKeyModsModSide) bool {
	// Zig-only (Darwin): if (comptime builtin.os.tag == .macos) { return switch (oa) {
	// .false => false, .true => true, .left => side == .left, .right => side == .right } }
	return true
}

// ke2KeyRow is term51ABKeyRow.
type ke2KeyRow struct {
	k        InputKeyKey
	seq      string
	modifier bool
}

// ke2Row is term51ABRow.
func ke2Row(k InputKeyKey) (ke2KeyRow, bool) {
	for _, r := range ke2Rows {
		if r.k == k {
			return r, true
		}
	}
	return ke2KeyRow{}, false
}

// ke2Mods is term51ABMods.
func ke2Mods(bits uint8) InputKeyMods {
	return InputKeyMods{
		shift:     (bits & 1) != 0,
		ctrl:      (bits & 2) != 0,
		alt:       (bits & 4) != 0,
		super:     (bits & 8) != 0,
		caps_lock: (bits & 16) != 0,
		num_lock:  (bits & 32) != 0,
	}
}

// ke2KittyValue is term51ABKittyValue: the Kitty "modifiers" value (1 +
// bitmask) expected for ke2Mods(bits).
func ke2KittyValue(bits uint8) uint16 {
	var v uint16 = 1
	if (bits & 1) != 0 {
		v += 1 // shift
	}
	if (bits & 2) != 0 {
		v += 4 // ctrl
	}
	if (bits & 4) != 0 {
		v += 2 // alt
	}
	if (bits & 8) != 0 {
		v += 8 // super
	}
	if (bits & 16) != 0 {
		v += 64 // caps lock
	}
	if (bits & 32) != 0 {
		v += 128 // num lock
	}
	return v
}

// The term51AB option constants.

var ke2Dis = InputKeyEncodeOptions{kitty_flags: KittyKeyFlags{disambiguate: true}}
var ke2Events = InputKeyEncodeOptions{kitty_flags: KittyKeyFlags{report_events: true}}
var ke2AltsOnly = InputKeyEncodeOptions{kitty_flags: KittyKeyFlags{report_alternates: true}}
var ke2AllOnly = InputKeyEncodeOptions{kitty_flags: KittyKeyFlags{report_all: true}}
var ke2AssocOnly = InputKeyEncodeOptions{kitty_flags: KittyKeyFlags{report_associated: true}}
var ke2DisAll = InputKeyEncodeOptions{kitty_flags: KittyKeyFlags{disambiguate: true, report_all: true}}
var ke2DisEvents = InputKeyEncodeOptions{kitty_flags: KittyKeyFlags{disambiguate: true, report_events: true}}
var ke2EventsAll = InputKeyEncodeOptions{kitty_flags: KittyKeyFlags{disambiguate: true, report_events: true, report_all: true}}
var ke2Alts = InputKeyEncodeOptions{kitty_flags: KittyKeyFlags{disambiguate: true, report_alternates: true, report_all: true}}
var ke2Assoc = InputKeyEncodeOptions{kitty_flags: KittyKeyFlags{disambiguate: true, report_associated: true, report_all: true}}
var ke2Full = InputKeyEncodeOptions{kitty_flags: KittyKeyFlagsTrue}

// ke2Rows is term51ABRows: expected report-all encoding (no mods) of every key in the Kitty table.
var ke2Rows = []ke2KeyRow{
	{k: InputKeyKeyEscape, seq: "\x1b[27u", modifier: false},
	{k: InputKeyKeyEnter, seq: "\x1b[13u", modifier: false},
	{k: InputKeyKeyTab, seq: "\x1b[9u", modifier: false},
	{k: InputKeyKeyBackspace, seq: "\x1b[127u", modifier: false},
	{k: InputKeyKeyInsert, seq: "\x1b[2~", modifier: false},
	{k: InputKeyKeyDelete, seq: "\x1b[3~", modifier: false},
	{k: InputKeyKeyArrowLeft, seq: "\x1b[D", modifier: false},
	{k: InputKeyKeyArrowRight, seq: "\x1b[C", modifier: false},
	{k: InputKeyKeyArrowUp, seq: "\x1b[A", modifier: false},
	{k: InputKeyKeyArrowDown, seq: "\x1b[B", modifier: false},
	{k: InputKeyKeyPageUp, seq: "\x1b[5~", modifier: false},
	{k: InputKeyKeyPageDown, seq: "\x1b[6~", modifier: false},
	{k: InputKeyKeyHome, seq: "\x1b[H", modifier: false},
	{k: InputKeyKeyEnd, seq: "\x1b[F", modifier: false},
	{k: InputKeyKeyCapsLock, seq: "\x1b[57358u", modifier: true},
	{k: InputKeyKeyScrollLock, seq: "\x1b[57359u", modifier: false},
	{k: InputKeyKeyNumLock, seq: "\x1b[57360u", modifier: true},
	{k: InputKeyKeyPrintScreen, seq: "\x1b[57361u", modifier: false},
	{k: InputKeyKeyPause, seq: "\x1b[57362u", modifier: false},
	{k: InputKeyKeyF1, seq: "\x1b[P", modifier: false},
	{k: InputKeyKeyF2, seq: "\x1b[Q", modifier: false},
	{k: InputKeyKeyF3, seq: "\x1b[13~", modifier: false},
	{k: InputKeyKeyF4, seq: "\x1b[S", modifier: false},
	{k: InputKeyKeyF5, seq: "\x1b[15~", modifier: false},
	{k: InputKeyKeyF6, seq: "\x1b[17~", modifier: false},
	{k: InputKeyKeyF7, seq: "\x1b[18~", modifier: false},
	{k: InputKeyKeyF8, seq: "\x1b[19~", modifier: false},
	{k: InputKeyKeyF9, seq: "\x1b[20~", modifier: false},
	{k: InputKeyKeyF10, seq: "\x1b[21~", modifier: false},
	{k: InputKeyKeyF11, seq: "\x1b[23~", modifier: false},
	{k: InputKeyKeyF12, seq: "\x1b[24~", modifier: false},
	{k: InputKeyKeyF13, seq: "\x1b[57376u", modifier: false},
	{k: InputKeyKeyF14, seq: "\x1b[57377u", modifier: false},
	{k: InputKeyKeyF15, seq: "\x1b[57378u", modifier: false},
	{k: InputKeyKeyF16, seq: "\x1b[57379u", modifier: false},
	{k: InputKeyKeyF17, seq: "\x1b[57380u", modifier: false},
	{k: InputKeyKeyF18, seq: "\x1b[57381u", modifier: false},
	{k: InputKeyKeyF19, seq: "\x1b[57382u", modifier: false},
	{k: InputKeyKeyF20, seq: "\x1b[57383u", modifier: false},
	{k: InputKeyKeyF21, seq: "\x1b[57384u", modifier: false},
	{k: InputKeyKeyF22, seq: "\x1b[57385u", modifier: false},
	{k: InputKeyKeyF23, seq: "\x1b[57386u", modifier: false},
	{k: InputKeyKeyF24, seq: "\x1b[57387u", modifier: false},
	{k: InputKeyKeyF25, seq: "\x1b[57388u", modifier: false},
	{k: InputKeyKeyNumpad0, seq: "\x1b[57399u", modifier: false},
	{k: InputKeyKeyNumpad1, seq: "\x1b[57400u", modifier: false},
	{k: InputKeyKeyNumpad2, seq: "\x1b[57401u", modifier: false},
	{k: InputKeyKeyNumpad3, seq: "\x1b[57402u", modifier: false},
	{k: InputKeyKeyNumpad4, seq: "\x1b[57403u", modifier: false},
	{k: InputKeyKeyNumpad5, seq: "\x1b[57404u", modifier: false},
	{k: InputKeyKeyNumpad6, seq: "\x1b[57405u", modifier: false},
	{k: InputKeyKeyNumpad7, seq: "\x1b[57406u", modifier: false},
	{k: InputKeyKeyNumpad8, seq: "\x1b[57407u", modifier: false},
	{k: InputKeyKeyNumpad9, seq: "\x1b[57408u", modifier: false},
	{k: InputKeyKeyNumpadDecimal, seq: "\x1b[57409u", modifier: false},
	{k: InputKeyKeyNumpadDivide, seq: "\x1b[57410u", modifier: false},
	{k: InputKeyKeyNumpadMultiply, seq: "\x1b[57411u", modifier: false},
	{k: InputKeyKeyNumpadSubtract, seq: "\x1b[57412u", modifier: false},
	{k: InputKeyKeyNumpadAdd, seq: "\x1b[57413u", modifier: false},
	{k: InputKeyKeyNumpadEnter, seq: "\x1b[57414u", modifier: false},
	{k: InputKeyKeyNumpadEqual, seq: "\x1b[57415u", modifier: false},
	{k: InputKeyKeyNumpadSeparator, seq: "\x1b[57416u", modifier: false},
	{k: InputKeyKeyNumpadLeft, seq: "\x1b[57417u", modifier: false},
	{k: InputKeyKeyNumpadRight, seq: "\x1b[57418u", modifier: false},
	{k: InputKeyKeyNumpadUp, seq: "\x1b[57419u", modifier: false},
	{k: InputKeyKeyNumpadDown, seq: "\x1b[57420u", modifier: false},
	{k: InputKeyKeyNumpadPageUp, seq: "\x1b[57421u", modifier: false},
	{k: InputKeyKeyNumpadPageDown, seq: "\x1b[57422u", modifier: false},
	{k: InputKeyKeyNumpadHome, seq: "\x1b[57423u", modifier: false},
	{k: InputKeyKeyNumpadEnd, seq: "\x1b[57424u", modifier: false},
	{k: InputKeyKeyNumpadInsert, seq: "\x1b[57425u", modifier: false},
	{k: InputKeyKeyNumpadDelete, seq: "\x1b[57426u", modifier: false},
	{k: InputKeyKeyNumpadBegin, seq: "\x1b[57427u", modifier: false},
	{k: InputKeyKeyShiftLeft, seq: "\x1b[57441u", modifier: true},
	{k: InputKeyKeyShiftRight, seq: "\x1b[57447u", modifier: true},
	{k: InputKeyKeyControlLeft, seq: "\x1b[57442u", modifier: true},
	{k: InputKeyKeyControlRight, seq: "\x1b[57448u", modifier: true},
	{k: InputKeyKeyMetaLeft, seq: "\x1b[57444u", modifier: true},
	{k: InputKeyKeyMetaRight, seq: "\x1b[57450u", modifier: true},
	{k: InputKeyKeyAltLeft, seq: "\x1b[57443u", modifier: true},
	{k: InputKeyKeyAltRight, seq: "\x1b[57449u", modifier: true},
}

// Zig: test "legacy: hu layout ctrl+ő sends proper codepoint"
func TestKe2_legacy_hu_layout_ctrl_sends_proper_codepoint(t *testing.T) {
	var buf [128]uint8
	writer := oscWriterFixed(buf[:])
	zigTry(t, inputKeyEncodeLegacy(&writer, InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyBracketLeft, mods: InputKeyMods{ctrl: true}, utf8: []uint8("ő"), unshifted_codepoint: 337}, InputKeyEncodeOptions{}))
	actual := writer.buffered()
	expectEqualStrings(t, "[337;5u", string(actual[1:]))
}

// Zig: test "legacy: backspace with DEL utf8"
func TestKe2_legacy_backspace_with_DEL_utf8(t *testing.T) {
	var buf [128]uint8
	writer := oscWriterFixed(buf[:])
	zigTry(t, inputKeyEncodeLegacy(&writer, InputKeyKeyEvent{
		action:              InputKeyActionPress,
		key:                 InputKeyKeyBackspace,
		utf8:                []uint8{0x7F},
		unshifted_codepoint: 0x08,
	}, InputKeyEncodeOptions{}))
	expectEqualStrings(t, "\x7F", string(writer.buffered()))
}

// Zig: test "ctrlseq: normal ctrl c"
func TestKe2_ctrlseq_normal_ctrl_c(t *testing.T) {
	seq := inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("c"), 'c', InputKeyMods{ctrl: true})
	expectEqual(t, uint8(0x03), zigUnwrap(t, seq))
}

// Zig: test "ctrlseq: normal ctrl c, right control"
func TestKe2_ctrlseq_normal_ctrl_c_right_control(t *testing.T) {
	seq := inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("c"), 'c', InputKeyMods{ctrl: true, sides: InputKeyModsModsSide{ctrl: InputKeyModsModSideRight}})
	expectEqual(t, uint8(0x03), zigUnwrap(t, seq))
}

// Zig: test "ctrlseq: alt should be allowed"
func TestKe2_ctrlseq_alt_should_be_allowed(t *testing.T) {
	seq := inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("c"), 'c', InputKeyMods{alt: true, ctrl: true})
	expectEqual(t, uint8(0x03), zigUnwrap(t, seq))
}

// Zig: test "ctrlseq: no ctrl does nothing"
func TestKe2_ctrlseq_no_ctrl_does_nothing(t *testing.T) {
	expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("c"), 'c', InputKeyMods{}).Valid, "ctrlSeq(.unidentified, \"c\", 'c', .{}) == null")
}

// Zig: test "ctrlseq: shifted non-character"
func TestKe2_ctrlseq_shifted_non_character(t *testing.T) {
	seq := inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("_"), '-', InputKeyMods{ctrl: true, shift: true})
	expectEqual(t, uint8(0x1F), zigUnwrap(t, seq))
}

// Zig: test "ctrlseq: caps ascii letter"
func TestKe2_ctrlseq_caps_ascii_letter(t *testing.T) {
	seq := inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("C"), 'c', InputKeyMods{ctrl: true, caps_lock: true})
	expectEqual(t, uint8(0x03), zigUnwrap(t, seq))
}

// Zig: test "ctrlseq: shift does not generate ctrl seq"
func TestKe2_ctrlseq_shift_does_not_generate_ctrl_seq(t *testing.T) {
	expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("C"), 'c', InputKeyMods{shift: true}).Valid, "ctrlSeq(.unidentified, \"C\", 'c', .{ .shift = true }) == null")
	expect(t, !inputKeyEncodeCtrlSeq(InputKeyKeyUnidentified, []uint8("C"), 'c', InputKeyMods{shift: true, ctrl: true}).Valid, "ctrlSeq(.unidentified, \"C\", 'c', .{ .shift = true, .ctrl = true }) == null")
}

// Zig: test "ctrlseq: russian ctrl c"
func TestKe2_ctrlseq_russian_ctrl_c(t *testing.T) {
	seq := inputKeyEncodeCtrlSeq(InputKeyKeyKeyC, []uint8("с"), 0x0441, InputKeyMods{ctrl: true})
	expectEqual(t, uint8(0x03), zigUnwrap(t, seq))
}

// Zig: test "ctrlseq: russian shifted ctrl c"
func TestKe2_ctrlseq_russian_shifted_ctrl_c(t *testing.T) {
	seq := inputKeyEncodeCtrlSeq(InputKeyKeyKeyC, []uint8("с"), 0x0441, InputKeyMods{ctrl: true, shift: true})
	expect(t, !seq.Valid, "seq == null")
}

// Zig: test "ctrlseq: russian alt ctrl c"
func TestKe2_ctrlseq_russian_alt_ctrl_c(t *testing.T) {
	seq := inputKeyEncodeCtrlSeq(InputKeyKeyKeyC, []uint8("с"), 0x0441, InputKeyMods{ctrl: true, alt: true})
	expectEqual(t, uint8(0x03), zigUnwrap(t, seq))
}

// Zig: test "ctrlseq: right ctrl c"
func TestKe2_ctrlseq_right_ctrl_c(t *testing.T) {
	seq := inputKeyEncodeCtrlSeq(InputKeyKeyKeyC, []uint8("с"), 'c', InputKeyMods{ctrl: true, sides: InputKeyModsModsSide{ctrl: InputKeyModsModSideRight}})
	expectEqual(t, uint8(0x03), zigUnwrap(t, seq))
}

// Zig: test "term51: Options.fromTerminal: terminal defaults"
func TestKe2_term51_Options_fromTerminal_terminal_defaults(t *testing.T) {
	term := ke2Terminal(t)
	defer term.Deinit()

	o := InputKeyEncodeOptionsFromTerminal(term)
	// Modes 1036 and 1035 default to on, the others to off.
	expect(t, o.alt_esc_prefix, "o.alt_esc_prefix")
	expect(t, o.ignore_keypad_with_numlock, "o.ignore_keypad_with_numlock")
	expect(t, !o.cursor_key_application, "!o.cursor_key_application")
	expect(t, !o.keypad_key_application, "!o.keypad_key_application")
	expect(t, !o.modify_other_keys_state_2, "!o.modify_other_keys_state_2")
	expectEqual(t, uint8(0), o.kitty_flags.Int())
	expectEqual(t, InputConfigOptionAsAltFalse, o.macos_option_as_alt)
}

// Zig: test "term51: Options.fromTerminal: Options.default values"
func TestKe2_term51_Options_fromTerminal_Options_default_values(t *testing.T) {
	d := InputKeyEncodeOptionsDefault
	expect(t, !d.cursor_key_application, "!d.cursor_key_application")
	expect(t, !d.keypad_key_application, "!d.keypad_key_application")
	expect(t, !d.ignore_keypad_with_numlock, "!d.ignore_keypad_with_numlock")
	expect(t, !d.alt_esc_prefix, "!d.alt_esc_prefix")
	expect(t, !d.modify_other_keys_state_2, "!d.modify_other_keys_state_2")
	expectEqual(t, uint8(0), d.kitty_flags.Int())
	expectEqual(t, InputConfigOptionAsAltFalse, d.macos_option_as_alt)

	// The struct field defaults match Options.default.
	e := InputKeyEncodeOptions{}
	expectEqual(t, d, e)
}

// Zig: test "term51: Options.fromTerminal: each mode maps to its own field"
func TestKe2_term51_Options_fromTerminal_each_mode_maps_to_its_own_field(t *testing.T) {
	term := ke2Terminal(t)
	defer term.Deinit()
	// cursor_keys -> cursor_key_application only
	term.modes.Set(ModesModeCursorKeys, true)
	o := InputKeyEncodeOptionsFromTerminal(term)
	expect(t, o.cursor_key_application, "o.cursor_key_application")
	expect(t, !o.keypad_key_application, "!o.keypad_key_application")
	expect(t, o.alt_esc_prefix, "o.alt_esc_prefix")
	expect(t, o.ignore_keypad_with_numlock, "o.ignore_keypad_with_numlock")
	expect(t, !o.modify_other_keys_state_2, "!o.modify_other_keys_state_2")
	term.modes.Set(ModesModeCursorKeys, false)
	// keypad_keys -> keypad_key_application only
	term.modes.Set(ModesModeKeypadKeys, true)
	o = InputKeyEncodeOptionsFromTerminal(term)
	expect(t, !o.cursor_key_application, "!o.cursor_key_application")
	expect(t, o.keypad_key_application, "o.keypad_key_application")
	expect(t, o.alt_esc_prefix, "o.alt_esc_prefix")
	expect(t, o.ignore_keypad_with_numlock, "o.ignore_keypad_with_numlock")
	term.modes.Set(ModesModeKeypadKeys, false)
	// ignore_keypad_with_numlock (off)
	term.modes.Set(ModesModeIgnoreKeypadWithNumlock, false)
	o = InputKeyEncodeOptionsFromTerminal(term)
	expect(t, !o.ignore_keypad_with_numlock, "!o.ignore_keypad_with_numlock")
	expect(t, o.alt_esc_prefix, "o.alt_esc_prefix")
	expect(t, !o.cursor_key_application, "!o.cursor_key_application")
	expect(t, !o.keypad_key_application, "!o.keypad_key_application")
	term.modes.Set(ModesModeIgnoreKeypadWithNumlock, true)
	// alt_esc_prefix (off)
	term.modes.Set(ModesModeAltEscPrefix, false)
	o = InputKeyEncodeOptionsFromTerminal(term)
	expect(t, !o.alt_esc_prefix, "!o.alt_esc_prefix")
	expect(t, o.ignore_keypad_with_numlock, "o.ignore_keypad_with_numlock")
	term.modes.Set(ModesModeAltEscPrefix, true)
	// modifyOtherKeys state 2 comes from the terminal flags, not a mode
	term.flags.modify_other_keys_2 = true
	o = InputKeyEncodeOptionsFromTerminal(term)
	expect(t, o.modify_other_keys_state_2, "o.modify_other_keys_state_2")
	expect(t, !o.cursor_key_application, "!o.cursor_key_application")
	expect(t, !o.keypad_key_application, "!o.keypad_key_application")
	term.flags.modify_other_keys_2 = false
	o = InputKeyEncodeOptionsFromTerminal(term)
	expect(t, !o.modify_other_keys_state_2, "!o.modify_other_keys_state_2")
	// All together
	term.modes.Set(ModesModeCursorKeys, true)
	term.modes.Set(ModesModeKeypadKeys, true)
	term.modes.Set(ModesModeIgnoreKeypadWithNumlock, false)
	term.modes.Set(ModesModeAltEscPrefix, false)
	term.flags.modify_other_keys_2 = true
	o = InputKeyEncodeOptionsFromTerminal(term)
	expect(t, o.cursor_key_application, "o.cursor_key_application")
	expect(t, o.keypad_key_application, "o.keypad_key_application")
	expect(t, !o.ignore_keypad_with_numlock, "!o.ignore_keypad_with_numlock")
	expect(t, !o.alt_esc_prefix, "!o.alt_esc_prefix")
	expect(t, o.modify_other_keys_state_2, "o.modify_other_keys_state_2")
	expectEqual(t, InputConfigOptionAsAltFalse, o.macos_option_as_alt)
}

// Zig: test "term51: Options.fromTerminal: kitty flags follow the active screen stack"
func TestKe2_term51_Options_fromTerminal_kitty_flags_follow_the_active_screen_stack(t *testing.T) {
	term := ke2Terminal(t)
	defer term.Deinit()

	// Each flag alone.
	flags := []KittyKeyFlags{
		{disambiguate: true},
		{report_events: true},
		{report_alternates: true},
		{report_all: true},
		{report_associated: true},
	}
	for i, f := range flags {
		term.screens.active.kitty_keyboard.Set(KittyKeySetModeSet, f)
		o := InputKeyEncodeOptionsFromTerminal(term)
		expectEqual(t, f, o.kitty_flags)
		expectEqual(t, uint8(1)<<i, o.kitty_flags.Int())
	}

	// All flags.
	term.screens.active.kitty_keyboard.Set(KittyKeySetModeSet, KittyKeyFlagsTrue)
	expectEqual(t, uint8(0b11111), InputKeyEncodeOptionsFromTerminal(term).kitty_flags.Int())

	// Push/pop changes the current value.
	term.screens.active.kitty_keyboard.Push(KittyKeyFlags{report_all: true})
	expectEqual(t,
		uint8(0b01000),
		InputKeyEncodeOptionsFromTerminal(term).kitty_flags.Int(),
	)
	term.screens.active.kitty_keyboard.Pop(1)
	expectEqual(t, uint8(0b11111), InputKeyEncodeOptionsFromTerminal(term).kitty_flags.Int())

	// The alternate screen has its own stack.
	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	expectEqual(t, uint8(0), InputKeyEncodeOptionsFromTerminal(term).kitty_flags.Int())
	term.screens.active.kitty_keyboard.Set(KittyKeySetModeSet, KittyKeyFlags{report_events: true})
	expectEqual(t, uint8(0b10), InputKeyEncodeOptionsFromTerminal(term).kitty_flags.Int())
	_, err = term.SwitchScreen(ScreenSetKeyPrimary)
	zigTry(t, err)
	expectEqual(t, uint8(0b11111), InputKeyEncodeOptionsFromTerminal(term).kitty_flags.Int())

	// The option feeds straight into the encoder.
	var buf [32]uint8
	w := oscWriterFixed(buf[:])
	zigTry(t, InputKeyEncodeEncode(&w, InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEscape}, InputKeyEncodeOptionsFromTerminal(term)))
	expectEqualStrings(t, "\x1b[27u", string(w.buffered()))
}

// Zig: test "term51: CsiUMods.fromInput: every shift/ctrl/alt combination"
func TestKe2_term51_CsiUMods_fromInput_every_shift_ctrl_alt_combination(t *testing.T) {
	for bits := uint8(0); ; bits++ {
		mods := InputKeyMods{
			shift: (bits & 1) != 0,
			alt:   (bits & 2) != 0,
			ctrl:  (bits & 4) != 0,
		}
		c := inputKeyEncodeCsiUModsFromInput(mods)
		expectEqual(t, mods.shift, c.shift)
		expectEqual(t, mods.alt, c.alt)
		expectEqual(t, mods.ctrl, c.ctrl)
		// The bit layout of CsiUMods is shift=1, alt=2, ctrl=4.
		expectEqual(t, bits, c.Int())
		if bits == 7 {
			break
		}
	}
}

// Zig: test "term51: CsiUMods.fromInput: ignores super, locks and sides"
func TestKe2_term51_CsiUMods_fromInput_ignores_super_locks_and_sides(t *testing.T) {
	c := inputKeyEncodeCsiUModsFromInput(InputKeyMods{
		super:     true,
		caps_lock: true,
		num_lock:  true,
		sides:     InputKeyModsModsSide{shift: InputKeyModsModSideRight, ctrl: InputKeyModsModSideRight, alt: InputKeyModsModSideRight, super: InputKeyModsModSideRight},
	})
	expectEqual(t, inputKeyEncodeCsiUMods{}, c)
	expectEqual(t, uint8(0), c.Int())
	expectEqual(t, uint8(1), c.SeqInt())

	// Sides do not alter the set bits either.
	d := inputKeyEncodeCsiUModsFromInput(InputKeyMods{
		shift: true,
		ctrl:  true,
		sides: InputKeyModsModsSide{shift: InputKeyModsModSideRight, ctrl: InputKeyModsModSideRight},
	})
	expectEqual(t, uint8(0b101), d.Int())

	// Every bit pattern of the full 16 bit mods struct maps only 3 bits.
	all := InputKeyModsModsFromInt(0xFFFF)
	expectEqual(t, uint8(7), inputKeyEncodeCsiUModsFromInput(all).Int())
}

// Zig: test "term51: CsiUMods.int: bit positions"
func TestKe2_term51_CsiUMods_int_bit_positions(t *testing.T) {
	expectEqual(t, uint8(0), (inputKeyEncodeCsiUMods{}).Int())
	expectEqual(t, uint8(1), (inputKeyEncodeCsiUMods{shift: true}).Int())
	expectEqual(t, uint8(2), (inputKeyEncodeCsiUMods{alt: true}).Int())
	expectEqual(t, uint8(4), (inputKeyEncodeCsiUMods{ctrl: true}).Int())
	expectEqual(t, uint8(3), (inputKeyEncodeCsiUMods{shift: true, alt: true}).Int())
	expectEqual(t, uint8(5), (inputKeyEncodeCsiUMods{shift: true, ctrl: true}).Int())
	expectEqual(t, uint8(6), (inputKeyEncodeCsiUMods{alt: true, ctrl: true}).Int())
	expectEqual(t, uint8(7), (inputKeyEncodeCsiUMods{shift: true, alt: true, ctrl: true}).Int())
}

// Zig: test "term51: CsiUMods.seqInt: all eight values"
func TestKe2_term51_CsiUMods_seqInt_all_eight_values(t *testing.T) {
	for raw := uint8(0); ; raw++ {
		c := inputKeyEncodeCsiUModsFromInt(raw)
		expectEqual(t, raw+1, c.SeqInt())
		if raw == 7 {
			break
		}
	}
	// Maximum value is 8 (not 7+1 overflowing u3).
	expectEqual(t, uint8(8), (inputKeyEncodeCsiUMods{shift: true, alt: true, ctrl: true}).SeqInt())
}

// Zig: test "term51: KittyMods.fromInput: each modifier alone"
func TestKe2_term51_KittyMods_fromInput_each_modifier_alone(t *testing.T) {
	none := inputKeyEncodeKittyModsFromInput(InputKeyActionPress, InputKeyKeyKeyA, InputKeyMods{})
	expectEqual(t, uint8(0), none.Int())

	expectEqual(t, uint8(1), inputKeyEncodeKittyModsFromInput(InputKeyActionPress, InputKeyKeyKeyA, InputKeyMods{shift: true}).Int())
	expectEqual(t, uint8(2), inputKeyEncodeKittyModsFromInput(InputKeyActionPress, InputKeyKeyKeyA, InputKeyMods{alt: true}).Int())
	// Note: input Mods has ctrl before alt, but Kitty has alt before ctrl.
	expectEqual(t, uint8(4), inputKeyEncodeKittyModsFromInput(InputKeyActionPress, InputKeyKeyKeyA, InputKeyMods{ctrl: true}).Int())
	expectEqual(t, uint8(8), inputKeyEncodeKittyModsFromInput(InputKeyActionPress, InputKeyKeyKeyA, InputKeyMods{super: true}).Int())
	expectEqual(t, uint8(64), inputKeyEncodeKittyModsFromInput(InputKeyActionPress, InputKeyKeyKeyA, InputKeyMods{caps_lock: true}).Int())
	expectEqual(t, uint8(128), inputKeyEncodeKittyModsFromInput(InputKeyActionPress, InputKeyKeyKeyA, InputKeyMods{num_lock: true}).Int())

	s := inputKeyEncodeKittyModsFromInput(InputKeyActionPress, InputKeyKeyKeyA, InputKeyMods{shift: true})
	expect(t, s.shift && !s.alt && !s.ctrl && !s.super, "s.shift and !s.alt and !s.ctrl and !s.super")
	expect(t, !s.hyper && !s.meta && !s.caps_lock && !s.num_lock, "!s.hyper and !s.meta and !s.caps_lock and !s.num_lock")
}

// Zig: test "term51: KittyMods.fromInput: all 64 combinations, hyper and meta never set"
func TestKe2_term51_KittyMods_fromInput_all_64_combinations_hyper_and_meta_never_set(t *testing.T) {
	for bits := uint8(0); bits < 64; bits++ {
		b := bits
		m := inputKeyEncodeKittyModsFromInput(InputKeyActionPress, InputKeyKeyKeyA, ke2Mods(b))
		expectEqual(t, ke2KittyValue(b)-1, uint16(m.Int()))
		expect(t, !m.hyper, "!m.hyper")
		expect(t, !m.meta, "!m.meta")
	}
}

// Zig: test "term51: KittyMods.fromInput: ignores action, key, sides and padding"
func TestKe2_term51_KittyMods_fromInput_ignores_action_key_sides_and_padding(t *testing.T) {
	mods := InputKeyMods{ctrl: true, shift: true}
	base := inputKeyEncodeKittyModsFromInput(InputKeyActionPress, InputKeyKeyKeyA, mods)
	expectEqual(t, uint8(5), base.Int())

	for _, a := range []InputKeyAction{InputKeyActionRelease, InputKeyActionPress, InputKeyActionRepeat} {
		for _, k := range []InputKeyKey{InputKeyKeyUnidentified, InputKeyKeyShiftLeft, InputKeyKeyCapsLock, InputKeyKeyEnter, InputKeyKeyF1, InputKeyKeyCopy} {
			expectEqual(t, base, inputKeyEncodeKittyModsFromInput(a, k, mods))
		}
	}

	// Sides do not matter.
	sided := mods
	sided.sides = InputKeyModsModsSide{shift: InputKeyModsModSideRight, ctrl: InputKeyModsModSideRight, alt: InputKeyModsModSideRight, super: InputKeyModsModSideRight}
	expectEqual(t, base, inputKeyEncodeKittyModsFromInput(InputKeyActionPress, InputKeyKeyKeyA, sided))

	// Padding bits are ignored.
	padded := InputKeyModsModsFromInt(0xFFFF)
	expectEqual(t, uint8(0b1100_1111), inputKeyEncodeKittyModsFromInput(InputKeyActionPress, InputKeyKeyKeyA, padded).Int())
}

// Zig: test "term51: KittyMods.int: bit positions"
func TestKe2_term51_KittyMods_int_bit_positions(t *testing.T) {
	expectEqual(t, uint8(0), (inputKeyEncodeKittyMods{}).Int())
	expectEqual(t, uint8(1), (inputKeyEncodeKittyMods{shift: true}).Int())
	expectEqual(t, uint8(2), (inputKeyEncodeKittyMods{alt: true}).Int())
	expectEqual(t, uint8(4), (inputKeyEncodeKittyMods{ctrl: true}).Int())
	expectEqual(t, uint8(8), (inputKeyEncodeKittyMods{super: true}).Int())
	expectEqual(t, uint8(16), (inputKeyEncodeKittyMods{hyper: true}).Int())
	expectEqual(t, uint8(32), (inputKeyEncodeKittyMods{meta: true}).Int())
	expectEqual(t, uint8(64), (inputKeyEncodeKittyMods{caps_lock: true}).Int())
	expectEqual(t, uint8(128), (inputKeyEncodeKittyMods{num_lock: true}).Int())
	expectEqual(t, uint8(255), (inputKeyEncodeKittyMods{
		shift:     true,
		alt:       true,
		ctrl:      true,
		super:     true,
		hyper:     true,
		meta:      true,
		caps_lock: true,
		num_lock:  true,
	}).Int())
}

// Zig: test "term51: KittyMods.seqInt: every bit pattern is the bitmask plus one"
func TestKe2_term51_KittyMods_seqInt_every_bit_pattern_is_the_bitmask_plus_one(t *testing.T) {
	for raw := uint16(0); raw < 256; raw++ {
		m := inputKeyEncodeKittyModsFromInt(uint8(raw))
		expectEqual(t, raw, uint16(m.Int()))
		expectEqual(t, raw+1, m.SeqInt())
	}
	// 256 is only representable thanks to the u9 return type.
	expectEqual(t, uint16(256), (inputKeyEncodeKittyMods{
		shift:     true,
		alt:       true,
		ctrl:      true,
		super:     true,
		hyper:     true,
		meta:      true,
		caps_lock: true,
		num_lock:  true,
	}).SeqInt())
	expectEqual(t, uint16(65), (inputKeyEncodeKittyMods{caps_lock: true}).SeqInt())
	expectEqual(t, uint16(129), (inputKeyEncodeKittyMods{num_lock: true}).SeqInt())
	expectEqual(t, uint16(9), (inputKeyEncodeKittyMods{super: true}).SeqInt())
	expectEqual(t, uint16(17), (inputKeyEncodeKittyMods{hyper: true}).SeqInt())
	expectEqual(t, uint16(33), (inputKeyEncodeKittyMods{meta: true}).SeqInt())
}

// Zig: test "term51: KittyMods.preventsText: each modifier"
func TestKe2_term51_KittyMods_preventsText_each_modifier(t *testing.T) {
	cases := []struct {
		m         inputKeyEncodeKittyMods
		alt_true  bool
		alt_false bool
	}{
		{m: inputKeyEncodeKittyMods{}, alt_true: false, alt_false: false},
		{m: inputKeyEncodeKittyMods{shift: true}, alt_true: false, alt_false: false},
		{m: inputKeyEncodeKittyMods{caps_lock: true}, alt_true: false, alt_false: false},
		{m: inputKeyEncodeKittyMods{num_lock: true}, alt_true: false, alt_false: false},
		{m: inputKeyEncodeKittyMods{shift: true, caps_lock: true, num_lock: true}, alt_true: false, alt_false: false},
		{m: inputKeyEncodeKittyMods{alt: true}, alt_true: true, alt_false: false},
		{m: inputKeyEncodeKittyMods{alt: true, shift: true}, alt_true: true, alt_false: false},
		{m: inputKeyEncodeKittyMods{ctrl: true}, alt_true: true, alt_false: true},
		{m: inputKeyEncodeKittyMods{super: true}, alt_true: true, alt_false: true},
		{m: inputKeyEncodeKittyMods{hyper: true}, alt_true: true, alt_false: true},
		{m: inputKeyEncodeKittyMods{meta: true}, alt_true: true, alt_false: true},
		{m: inputKeyEncodeKittyMods{alt: true, ctrl: true}, alt_true: true, alt_false: true},
		{m: inputKeyEncodeKittyMods{alt: true, super: true}, alt_true: true, alt_false: true},
	}
	for _, c := range cases {
		expectEqual(t, c.alt_true, c.m.PreventsText(true))
		expectEqual(t, c.alt_false, c.m.PreventsText(false))
	}
}

// Zig: test "term51: KittyMods.preventsText: all 256 bit patterns"
func TestKe2_term51_KittyMods_preventsText_all_256_bit_patterns(t *testing.T) {
	for raw := uint16(0); raw < 256; raw++ {
		r := uint8(raw)
		m := inputKeyEncodeKittyModsFromInt(r)
		// ctrl (4), super (8), hyper (16), meta (32) always prevent text.
		always := (r & 0b0011_1100) != 0
		alt := (r & 0b10) != 0
		expectEqual(t, always || alt, m.PreventsText(true))
		expectEqual(t, always, m.PreventsText(false))
	}
}

// Zig: test "term51: KittySequence.encode: dispatches on final byte"
func TestKe2_term51_KittySequence_encode_dispatches_on_final_byte(t *testing.T) {
	// 'u' and '~' use the full form.
	ke2ExpectSeq(t, "\x1b[97u", inputKeyEncodeKittySequence{key: 97, final: 'u'})
	ke2ExpectSeq(t, "\x1b[3~", inputKeyEncodeKittySequence{key: 3, final: '~'})
	// Every other final uses the special form, which ignores the key code.
	ke2ExpectSeq(t, "\x1b[A", inputKeyEncodeKittySequence{key: 1, final: 'A'})
	ke2ExpectSeq(t, "\x1b[A", inputKeyEncodeKittySequence{key: 57, final: 'A'})
	ke2ExpectSeq(t, "\x1b[x", inputKeyEncodeKittySequence{key: 5, final: 'x'})
	ke2ExpectSeq(t, "\x1b[U", inputKeyEncodeKittySequence{key: 5, final: 'U'})
	ke2ExpectSeq(t, "\x1b[1;2x", inputKeyEncodeKittySequence{key: 5, final: 'x', mods: inputKeyEncodeKittyMods{shift: true}})
	ke2ExpectSeq(t, "\x1b[1;2~", inputKeyEncodeKittySequence{key: 1, final: '~', mods: inputKeyEncodeKittyMods{shift: true}})
}

// Zig: test "term51: KittySequence.encodeFull: key codes"
func TestKe2_term51_KittySequence_encodeFull_key_codes(t *testing.T) {
	ke2ExpectSeq(t, "\x1b[0u", inputKeyEncodeKittySequence{key: 0, final: 'u'})
	ke2ExpectSeq(t, "\x1b[1u", inputKeyEncodeKittySequence{key: 1, final: 'u'})
	ke2ExpectSeq(t, "\x1b[1114111u", inputKeyEncodeKittySequence{key: 0x10FFFF, final: 'u'})
	ke2ExpectSeq(t, "\x1b[1114112u", inputKeyEncodeKittySequence{key: 0x110000, final: 'u'})
	ke2ExpectSeq(t, "\x1b[2097151u", inputKeyEncodeKittySequence{key: 0x1FFFFF, final: 'u'})
	ke2ExpectSeq(t, "\x1b[57441u", inputKeyEncodeKittySequence{key: 57441, final: 'u'})
	ke2ExpectSeq(t, "\x1b[24~", inputKeyEncodeKittySequence{key: 24, final: '~'})
}

// Zig: test "term51: KittySequence.encodeFull: modifiers"
func TestKe2_term51_KittySequence_encodeFull_modifiers(t *testing.T) {
	// mods == 1 (none) is omitted, otherwise written as the seqInt value.
	ke2ExpectSeq(t, "\x1b[97;2u", inputKeyEncodeKittySequence{key: 97, final: 'u', mods: inputKeyEncodeKittyMods{shift: true}})
	ke2ExpectSeq(t, "\x1b[97;3u", inputKeyEncodeKittySequence{key: 97, final: 'u', mods: inputKeyEncodeKittyMods{alt: true}})
	ke2ExpectSeq(t, "\x1b[97;5u", inputKeyEncodeKittySequence{key: 97, final: 'u', mods: inputKeyEncodeKittyMods{ctrl: true}})
	ke2ExpectSeq(t, "\x1b[97;9u", inputKeyEncodeKittySequence{key: 97, final: 'u', mods: inputKeyEncodeKittyMods{super: true}})
	ke2ExpectSeq(t, "\x1b[97;17u", inputKeyEncodeKittySequence{key: 97, final: 'u', mods: inputKeyEncodeKittyMods{hyper: true}})
	ke2ExpectSeq(t, "\x1b[97;33u", inputKeyEncodeKittySequence{key: 97, final: 'u', mods: inputKeyEncodeKittyMods{meta: true}})
	ke2ExpectSeq(t, "\x1b[97;65u", inputKeyEncodeKittySequence{key: 97, final: 'u', mods: inputKeyEncodeKittyMods{caps_lock: true}})
	ke2ExpectSeq(t, "\x1b[97;129u", inputKeyEncodeKittySequence{key: 97, final: 'u', mods: inputKeyEncodeKittyMods{num_lock: true}})
	ke2ExpectSeq(t, "\x1b[97;256u", inputKeyEncodeKittySequence{key: 97, final: 'u', mods: inputKeyEncodeKittyModsFromInt(255)})
	ke2ExpectSeq(t, "\x1b[3;6~", inputKeyEncodeKittySequence{key: 3, final: '~', mods: inputKeyEncodeKittyMods{shift: true, ctrl: true}})
}

// Zig: test "term51: KittySequence.encodeFull: event types"
func TestKe2_term51_KittySequence_encodeFull_event_types(t *testing.T) {
	// none and press with no modifiers emit no modifier section at all.
	ke2ExpectSeq(t, "\x1b[97u", inputKeyEncodeKittySequence{key: 97, final: 'u', event: inputKeyEncodeKittySequenceEventNone})
	ke2ExpectSeq(t, "\x1b[97u", inputKeyEncodeKittySequence{key: 97, final: 'u', event: inputKeyEncodeKittySequenceEventPress})
	// press does not emit ":1" but the modifiers still go out.
	ke2ExpectSeq(t, "\x1b[97;2u", inputKeyEncodeKittySequence{key: 97, final: 'u', event: inputKeyEncodeKittySequenceEventPress, mods: inputKeyEncodeKittyMods{shift: true}})
	ke2ExpectSeq(t, "\x1b[97;2u", inputKeyEncodeKittySequence{key: 97, final: 'u', event: inputKeyEncodeKittySequenceEventNone, mods: inputKeyEncodeKittyMods{shift: true}})
	// repeat and release always emit the modifier value (even 1) and the event.
	ke2ExpectSeq(t, "\x1b[97;1:2u", inputKeyEncodeKittySequence{key: 97, final: 'u', event: inputKeyEncodeKittySequenceEventRepeat})
	ke2ExpectSeq(t, "\x1b[97;1:3u", inputKeyEncodeKittySequence{key: 97, final: 'u', event: inputKeyEncodeKittySequenceEventRelease})
	ke2ExpectSeq(t, "\x1b[97;5:2u", inputKeyEncodeKittySequence{key: 97, final: 'u', event: inputKeyEncodeKittySequenceEventRepeat, mods: inputKeyEncodeKittyMods{ctrl: true}})
	ke2ExpectSeq(t, "\x1b[97;5:3u", inputKeyEncodeKittySequence{key: 97, final: 'u', event: inputKeyEncodeKittySequenceEventRelease, mods: inputKeyEncodeKittyMods{ctrl: true}})
	ke2ExpectSeq(t, "\x1b[2;1:3~", inputKeyEncodeKittySequence{key: 2, final: '~', event: inputKeyEncodeKittySequenceEventRelease})
	expectEqual(t, uint8(0), uint8(inputKeyEncodeKittySequenceEventNone))
	expectEqual(t, uint8(1), uint8(inputKeyEncodeKittySequenceEventPress))
	expectEqual(t, uint8(2), uint8(inputKeyEncodeKittySequenceEventRepeat))
	expectEqual(t, uint8(3), uint8(inputKeyEncodeKittySequenceEventRelease))
}

// Zig: test "term51: KittySequence.encodeFull: alternates"
func TestKe2_term51_KittySequence_encodeFull_alternates(t *testing.T) {
	ke2ExpectSeq(t, "\x1b[97:65u", inputKeyEncodeKittySequence{key: 97, final: 'u', alternates: [2]Optional[uint32]{Some(uint32(65)), {}}})
	ke2ExpectSeq(t, "\x1b[97::98u", inputKeyEncodeKittySequence{key: 97, final: 'u', alternates: [2]Optional[uint32]{{}, Some(uint32(98))}})
	ke2ExpectSeq(t, "\x1b[97:65:98u", inputKeyEncodeKittySequence{key: 97, final: 'u', alternates: [2]Optional[uint32]{Some(uint32(65)), Some(uint32(98))}})
	ke2ExpectSeq(t, "\x1b[97:0u", inputKeyEncodeKittySequence{key: 97, final: 'u', alternates: [2]Optional[uint32]{Some(uint32(0)), {}}})
	ke2ExpectSeq(t, "\x1b[97::0u", inputKeyEncodeKittySequence{key: 97, final: 'u', alternates: [2]Optional[uint32]{{}, Some(uint32(0))}})
	ke2ExpectSeq(t, "\x1b[97:2097151:2097151u", inputKeyEncodeKittySequence{key: 97, final: 'u', alternates: [2]Optional[uint32]{Some(uint32(0x1FFFFF)), Some(uint32(0x1FFFFF))}})
	ke2ExpectSeq(t, "\x1b[97u", inputKeyEncodeKittySequence{key: 97, final: 'u', alternates: [2]Optional[uint32]{{}, {}}})
	// With modifiers, an event and '~'.
	ke2ExpectSeq(t, "\x1b[97:65:98;2:3u", inputKeyEncodeKittySequence{key: 97, final: 'u', alternates: [2]Optional[uint32]{Some(uint32(65)), Some(uint32(98))}, mods: inputKeyEncodeKittyMods{shift: true}, event: inputKeyEncodeKittySequenceEventRelease})
	ke2ExpectSeq(t, "\x1b[3::98;5~", inputKeyEncodeKittySequence{key: 3, final: '~', alternates: [2]Optional[uint32]{{}, Some(uint32(98))}, mods: inputKeyEncodeKittyMods{ctrl: true}})
}

// Zig: test "term51: KittySequence.encodeFull: associated text"
func TestKe2_term51_KittySequence_encodeFull_associated_text(t *testing.T) {
	// No modifier section was emitted so two semicolons introduce the text.
	ke2ExpectSeq(t, "\x1b[97;;97u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("a")})
	ke2ExpectSeq(t, "\x1b[97;;97u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("a"), event: inputKeyEncodeKittySequenceEventPress})
	// After a modifier section just one.
	ke2ExpectSeq(t, "\x1b[97;2;65u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("A"), mods: inputKeyEncodeKittyMods{shift: true}})
	ke2ExpectSeq(t, "\x1b[97;1:2;97u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("a"), event: inputKeyEncodeKittySequenceEventRepeat})
	ke2ExpectSeq(t, "\x1b[97;1:3;97u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("a"), event: inputKeyEncodeKittySequenceEventRelease})
	// Multiple codepoints are colon separated.
	ke2ExpectSeq(t, "\x1b[97;;97:98u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("ab")})
	ke2ExpectSeq(t, "\x1b[97;;97:98:99u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("abc")})
	// Multi-byte codepoints (2, 3 and 4 byte UTF-8, max scalar).
	ke2ExpectSeq(t, "\x1b[97;;233u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("é")})
	ke2ExpectSeq(t, "\x1b[97;;8721u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("∑")})
	ke2ExpectSeq(t, "\x1b[97;;128512u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("😀")})
	ke2ExpectSeq(t, "\x1b[97;;1114111u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("\xF4\x8F\xBF\xBF")})
	// Space is printable, the largest control (0x1F) and DEL are skipped.
	ke2ExpectSeq(t, "\x1b[97;;32u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8(" ")})
	ke2ExpectSeq(t, "\x1b[97u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("\x1f")})
	ke2ExpectSeq(t, "\x1b[97u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("\x7f")})
	// U+0080 is a C1 control but isControl only knows C0 and DEL.
	ke2ExpectSeq(t, "\x1b[97;;128u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("\xC2\x80")})
	ke2ExpectSeq(t, "\x1b[97;;32:97u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8(" a")})
}

// Zig: test "term51: KittySequence.encodeFull: control characters in text"
func TestKe2_term51_KittySequence_encodeFull_control_characters_in_text(t *testing.T) {
	// Controls are skipped, the rest keeps its separators.
	ke2ExpectSeq(t, "\x1b[97;;97:98u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("a\x01b")})
	ke2ExpectSeq(t, "\x1b[97;;97u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("\x01a")})
	ke2ExpectSeq(t, "\x1b[97;;97u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("a\x7f")})
	ke2ExpectSeq(t, "\x1b[97;;97:98u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("\x1b\n\ra\tb\x00")})
	// Only controls: no text section and no stray semicolons.
	ke2ExpectSeq(t, "\x1b[97u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("\x01\x02")})
	ke2ExpectSeq(t, "\x1b[97u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("\x00")})
	// The modifier section stays when the text vanishes.
	ke2ExpectSeq(t, "\x1b[97;2u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("\x01"), mods: inputKeyEncodeKittyMods{shift: true}})
	ke2ExpectSeq(t, "\x1b[97;1:3u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("\r"), event: inputKeyEncodeKittySequenceEventRelease})
}

// Zig: test "term51: KittySequence.encodeFull: invalid or empty text is dropped"
func TestKe2_term51_KittySequence_encodeFull_invalid_or_empty_text_is_dropped(t *testing.T) {
	ke2ExpectSeq(t, "\x1b[97u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("")})
	ke2ExpectSeq(t, "\x1b[97u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("\xff")})
	ke2ExpectSeq(t, "\x1b[97u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("a\xff")})
	ke2ExpectSeq(t, "\x1b[97u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("\xC0\x80")})
	ke2ExpectSeq(t, "\x1b[97u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("\xED\xA0\x80")})
	ke2ExpectSeq(t, "\x1b[97u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("\xE3\x81")})
	ke2ExpectSeq(t, "\x1b[97u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("\xF4\x90\x80\x80")})
	ke2ExpectSeq(t, "\x1b[97;2u", inputKeyEncodeKittySequence{key: 97, final: 'u', text: []uint8("\xff"), mods: inputKeyEncodeKittyMods{shift: true}})
}

// Zig: test "term51: KittySequence.encodeFull: everything together"
func TestKe2_term51_KittySequence_encodeFull_everything_together(t *testing.T) {
	ke2ExpectSeq(t, "\x1b[97:65:98;2:2;65:66u", inputKeyEncodeKittySequence{key: 97, final: 'u', alternates: [2]Optional[uint32]{Some(uint32(65)), Some(uint32(98))}, mods: inputKeyEncodeKittyMods{shift: true}, event: inputKeyEncodeKittySequenceEventRepeat, text: []uint8("AB")})
	ke2ExpectSeq(t, "\x1b[3:1:2;2;120~", inputKeyEncodeKittySequence{key: 3, final: '~', alternates: [2]Optional[uint32]{Some(uint32(1)), Some(uint32(2))}, mods: inputKeyEncodeKittyMods{shift: true}, event: inputKeyEncodeKittySequenceEventPress, text: []uint8("x")})
	ke2ExpectSeq(t, "\x1b[97::98;;97u", inputKeyEncodeKittySequence{key: 97, final: 'u', alternates: [2]Optional[uint32]{{}, Some(uint32(98))}, event: inputKeyEncodeKittySequenceEventPress, text: []uint8("a")})
}

// Zig: test "term51: KittySequence.encodeSpecial: forms"
func TestKe2_term51_KittySequence_encodeSpecial_forms(t *testing.T) {
	// No mods, no event.
	ke2ExpectSeq(t, "\x1b[A", inputKeyEncodeKittySequence{key: 1, final: 'A'})
	ke2ExpectSeq(t, "\x1b[H", inputKeyEncodeKittySequence{key: 1, final: 'H'})
	// Mods only.
	ke2ExpectSeq(t, "\x1b[1;2A", inputKeyEncodeKittySequence{key: 1, final: 'A', mods: inputKeyEncodeKittyMods{shift: true}})
	ke2ExpectSeq(t, "\x1b[1;9A", inputKeyEncodeKittySequence{key: 1, final: 'A', mods: inputKeyEncodeKittyMods{super: true}})
	ke2ExpectSeq(t, "\x1b[1;256A", inputKeyEncodeKittySequence{key: 1, final: 'A', mods: inputKeyEncodeKittyModsFromInt(255)})
	// Any event (including press) forces the full "1;mods:event" form.
	ke2ExpectSeq(t, "\x1b[1;1:1A", inputKeyEncodeKittySequence{key: 1, final: 'A', event: inputKeyEncodeKittySequenceEventPress})
	ke2ExpectSeq(t, "\x1b[1;1:2A", inputKeyEncodeKittySequence{key: 1, final: 'A', event: inputKeyEncodeKittySequenceEventRepeat})
	ke2ExpectSeq(t, "\x1b[1;1:3A", inputKeyEncodeKittySequence{key: 1, final: 'A', event: inputKeyEncodeKittySequenceEventRelease})
	ke2ExpectSeq(t, "\x1b[1;2:1A", inputKeyEncodeKittySequence{key: 1, final: 'A', event: inputKeyEncodeKittySequenceEventPress, mods: inputKeyEncodeKittyMods{shift: true}})
	ke2ExpectSeq(t, "\x1b[1;2:3A", inputKeyEncodeKittySequence{key: 1, final: 'A', event: inputKeyEncodeKittySequenceEventRelease, mods: inputKeyEncodeKittyMods{shift: true}})
	ke2ExpectSeq(t, "\x1b[1;5:2D", inputKeyEncodeKittySequence{key: 1, final: 'D', event: inputKeyEncodeKittySequenceEventRepeat, mods: inputKeyEncodeKittyMods{ctrl: true}})
}

// Zig: test "term51: KittySequence.encodeSpecial: ignores key code, alternates and text"
func TestKe2_term51_KittySequence_encodeSpecial_ignores_key_code_alternates_and_text(t *testing.T) {
	ke2ExpectSeq(t, "\x1b[A", inputKeyEncodeKittySequence{key: 99999, final: 'A', alternates: [2]Optional[uint32]{Some(uint32(65)), Some(uint32(66))}, text: []uint8("abc")})
	ke2ExpectSeq(t, "\x1b[1;2:3A", inputKeyEncodeKittySequence{key: 0, final: 'A', alternates: [2]Optional[uint32]{Some(uint32(65)), {}}, text: []uint8("abc"), mods: inputKeyEncodeKittyMods{shift: true}, event: inputKeyEncodeKittySequenceEventRelease})
	// Non-ASCII final byte is written as a raw byte.
	ke2ExpectSeq(t, "\x1b[\xff", inputKeyEncodeKittySequence{key: 1, final: 0xFF})
}

// Zig: test "term51: kitty: disabled flags fall back to legacy"
func TestKe2_term51_kitty_disabled_flags_fall_back_to_legacy(t *testing.T) {
	none := InputKeyEncodeOptions{}
	ke2RunCases(t, inputKeyEncodeKitty, []ke2Case{
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEscape}, opts: none, want: "\x1b"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyEscape}, opts: none, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionRepeat, key: InputKeyKeyEscape}, opts: none, want: "\x1b"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyArrowUp}, opts: none, want: "\x1b[A"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyArrowUp}, opts: InputKeyEncodeOptions{cursor_key_application: true}, want: "\x1bOA"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a"), unshifted_codepoint: 'a'}, opts: none, want: "a"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, mods: InputKeyMods{ctrl: true}, utf8: []uint8("a"), unshifted_codepoint: 'a'}, opts: none, want: "\x01"},
		// Legacy with a control character as the text of ctrl+a falls to CSI u.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, mods: InputKeyMods{ctrl: true}, utf8: []uint8("\x01"), unshifted_codepoint: 'a'}, opts: none, want: "\x1b[1;5u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, composing: true, utf8: []uint8("a")}, opts: none, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyF1}, opts: none, want: "\x1bOP"},
	})
}

// Zig: test "term51: kitty: release events need report_events"
func TestKe2_term51_kitty_release_events_need_report_events(t *testing.T) {
	a_rel := InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyKeyA, utf8: []uint8("a"), unshifted_codepoint: 'a'}
	// Without report_events nothing is sent, whatever else is set.
	ke2RunCases(t, inputKeyEncodeKitty, []ke2Case{
		{ev: a_rel, opts: ke2Dis, want: ""},
		{ev: a_rel, opts: ke2AllOnly, want: ""},
		{ev: a_rel, opts: ke2DisAll, want: ""},
		{ev: a_rel, opts: ke2Alts, want: ""},
		{ev: a_rel, opts: ke2Assoc, want: ""},
		{ev: a_rel, opts: ke2AltsOnly, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyEscape}, opts: ke2DisAll, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyArrowUp}, opts: ke2DisAll, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyEnter}, opts: ke2DisAll, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyShiftLeft}, opts: ke2DisAll, want: ""},
		// With report_events they are encoded, always with an explicit event.
		{ev: a_rel, opts: ke2Events, want: "\x1b[97;1:3u"},
		{ev: a_rel, opts: ke2DisEvents, want: "\x1b[97;1:3u"},
		{ev: a_rel, opts: ke2EventsAll, want: "\x1b[97;1:3u"},
		{ev: a_rel, opts: ke2Full, want: "\x1b[97;1:3u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyEscape}, opts: ke2Events, want: "\x1b[27;1:3u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyArrowUp}, opts: ke2Events, want: "\x1b[1;1:3A"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyDelete}, opts: ke2Events, want: "\x1b[3;1:3~"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyArrowUp, mods: InputKeyMods{shift: true}}, opts: ke2Events, want: "\x1b[1;2:3A"},
	})
}

// Zig: test "term51: kitty: release of enter, backspace and tab needs report_all"
func TestKe2_term51_kitty_release_of_enter_backspace_and_tab_needs_report_all(t *testing.T) {
	for _, k := range []InputKeyKey{InputKeyKeyEnter, InputKeyKeyBackspace, InputKeyKeyTab} {
		ev := InputKeyKeyEvent{action: InputKeyActionRelease, key: k}
		ke2Expect(t, inputKeyEncodeKitty, "", ev, ke2Events)
		ke2Expect(t, inputKeyEncodeKitty, "", ev, ke2DisEvents)
		ke2Expect(t, inputKeyEncodeKitty, "", ev, InputKeyEncodeOptions{kitty_flags: KittyKeyFlags{report_events: true, report_alternates: true, report_associated: true}})
	}
	ke2RunCases(t, inputKeyEncodeKitty, []ke2Case{
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyEnter}, opts: ke2EventsAll, want: "\x1b[13;1:3u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyBackspace}, opts: ke2EventsAll, want: "\x1b[127;1:3u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyTab}, opts: ke2EventsAll, want: "\x1b[9;1:3u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyEnter, mods: InputKeyMods{shift: true}}, opts: ke2EventsAll, want: "\x1b[13;2:3u"},
		// Other keys that merely look like them still encode.
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyNumpadEnter}, opts: ke2Events, want: "\x1b[57414;1:3u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyEscape}, opts: ke2Events, want: "\x1b[27;1:3u"},
	})
}

// Zig: test "term51: kitty: press and repeat events"
func TestKe2_term51_kitty_press_and_repeat_events(t *testing.T) {
	ke2RunCases(t, inputKeyEncodeKitty, []ke2Case{
		// report_events: press has no ":1" in the full form, repeat does.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEscape}, opts: ke2Events, want: "\x1b[27u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRepeat, key: InputKeyKeyEscape}, opts: ke2Events, want: "\x1b[27;1:2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEscape, mods: InputKeyMods{ctrl: true, shift: true}}, opts: ke2Events, want: "\x1b[27;6u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRepeat, key: InputKeyKeyEscape, mods: InputKeyMods{ctrl: true, shift: true}}, opts: ke2Events, want: "\x1b[27;6:2u"},
		// The special form always carries the event.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyArrowUp}, opts: ke2Events, want: "\x1b[1;1:1A"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRepeat, key: InputKeyKeyArrowUp}, opts: ke2Events, want: "\x1b[1;1:2A"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyDelete}, opts: ke2Events, want: "\x1b[3~"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRepeat, key: InputKeyKeyDelete}, opts: ke2Events, want: "\x1b[3;1:2~"},
		// Without report_events repeat looks like press.
		{ev: InputKeyKeyEvent{action: InputKeyActionRepeat, key: InputKeyKeyEscape}, opts: ke2Dis, want: "\x1b[27u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRepeat, key: InputKeyKeyArrowUp}, opts: ke2Dis, want: "\x1b[A"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRepeat, key: InputKeyKeyArrowUp, mods: InputKeyMods{shift: true}}, opts: ke2Dis, want: "\x1b[1;2A"},
		// Plain text is unaffected by the action (press and repeat).
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a"), unshifted_codepoint: 'a'}, opts: ke2Events, want: "a"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRepeat, key: InputKeyKeyKeyA, utf8: []uint8("a"), unshifted_codepoint: 'a'}, opts: ke2Events, want: "a"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRepeat, key: InputKeyKeyKeyA, utf8: []uint8("a"), unshifted_codepoint: 'a'}, opts: ke2EventsAll, want: "\x1b[97;1:2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a"), unshifted_codepoint: 'a'}, opts: ke2EventsAll, want: "\x1b[97u"},
	})
}

// Zig: test "term51: kitty: every key of the Kitty table with report_all"
func TestKe2_term51_kitty_every_key_of_the_Kitty_table_with_report_all(t *testing.T) {
	for _, r := range ke2Rows {
		ke2Expect(t, inputKeyEncodeKitty, r.seq, InputKeyKeyEvent{action: InputKeyActionPress, key: r.k}, ke2DisAll)
		ke2Expect(t, inputKeyEncodeKitty, r.seq, InputKeyKeyEvent{action: InputKeyActionPress, key: r.k}, ke2AllOnly)
		// The unshifted codepoint is ignored for table keys.
		ke2Expect(t, inputKeyEncodeKitty, r.seq, InputKeyKeyEvent{action: InputKeyActionPress, key: r.k, unshifted_codepoint: 'x'}, ke2DisAll)
	}
	expectEqual(t, 81, len(ke2Rows))
}

// Zig: test "term51: kitty: every key of the Kitty table with disambiguate only"
func TestKe2_term51_kitty_every_key_of_the_Kitty_table_with_disambiguate_only(t *testing.T) {
	for _, r := range ke2Rows {
		var want string
		if r.modifier {
			want = ""
		} else {
			switch r.k {
			case InputKeyKeyEnter:
				want = "\r"
			case InputKeyKeyTab:
				want = "\t"
			case InputKeyKeyBackspace:
				want = "\x7f"
			default:
				want = r.seq
			}
		}
		ke2Expect(t, inputKeyEncodeKitty, want, InputKeyKeyEvent{action: InputKeyActionPress, key: r.k}, ke2Dis)
		ke2Expect(t, inputKeyEncodeKitty, want, InputKeyKeyEvent{action: InputKeyActionPress, key: r.k, unshifted_codepoint: 'x'}, ke2Dis)
	}
}

// Zig: test "term51: kitty: every key of the Kitty table with ctrl"
func TestKe2_term51_kitty_every_key_of_the_Kitty_table_with_ctrl(t *testing.T) {
	for _, r := range ke2Rows {
		last := r.seq[len(r.seq)-1]
		var want string
		if last == 'u' || last == '~' {
			want = fmt.Sprintf("%s;5%c", r.seq[0:len(r.seq)-1], last)
		} else {
			want = fmt.Sprintf("\x1b[1;5%c", last)
		}
		// Modifier keys are only reported with report_all.
		if !r.modifier {
			ke2Expect(t, inputKeyEncodeKitty, want, InputKeyKeyEvent{action: InputKeyActionPress, key: r.k, mods: InputKeyMods{ctrl: true}}, ke2Dis)
		}
		ke2Expect(t, inputKeyEncodeKitty, want, InputKeyKeyEvent{action: InputKeyActionPress, key: r.k, mods: InputKeyMods{ctrl: true}}, ke2DisAll)
	}
}

// Zig: test "term51: kitty: keys outside the table use the unshifted codepoint"
func TestKe2_term51_kitty_keys_outside_the_table_use_the_unshifted_codepoint(t *testing.T) {
	for _, k := range InputKeyKeyValues() {
		if _, ok := ke2Row(k); ok {
			continue
		}
		// No unshifted codepoint, no text: nothing to send.
		ke2Expect(t, inputKeyEncodeKitty, "", InputKeyKeyEvent{action: InputKeyActionPress, key: k}, ke2DisAll)
		ke2Expect(t, inputKeyEncodeKitty, "", InputKeyKeyEvent{action: InputKeyActionPress, key: k}, ke2Dis)
		ke2Expect(t, inputKeyEncodeKitty, "", InputKeyKeyEvent{action: InputKeyActionPress, key: k}, ke2Full)
		// With an unshifted codepoint it is the key number.
		ke2Expect(t, inputKeyEncodeKitty, "\x1b[120u", InputKeyKeyEvent{action: InputKeyActionPress, key: k, unshifted_codepoint: 'x'}, ke2DisAll)
		ke2Expect(t, inputKeyEncodeKitty, "\x1b[120u", InputKeyKeyEvent{action: InputKeyActionPress, key: k, unshifted_codepoint: 'x'}, ke2Dis)
		ke2Expect(t, inputKeyEncodeKitty, "\x1b[120;5u", InputKeyKeyEvent{action: InputKeyActionPress, key: k, unshifted_codepoint: 'x', mods: InputKeyMods{ctrl: true}}, ke2Dis)
	}
}

// Zig: test "term51: kitty: unshifted codepoint boundaries"
func TestKe2_term51_kitty_unshifted_codepoint_boundaries(t *testing.T) {
	ke2RunCases(t, inputKeyEncodeKitty, []ke2Case{
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA}, opts: ke2DisAll, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, unshifted_codepoint: 1}, opts: ke2DisAll, want: "\x1b[1u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, unshifted_codepoint: 0x10FFFF}, opts: ke2DisAll, want: "\x1b[1114111u"},
		// One past the last Unicode scalar: encoded as is.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, unshifted_codepoint: 0x110000}, opts: ke2DisAll, want: "\x1b[1114112u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, unshifted_codepoint: 0x1FFFFF}, opts: ke2DisAll, want: "\x1b[2097151u"},
		// A table key with a shadowing unshifted codepoint still uses the table.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, unshifted_codepoint: 'x'}, opts: ke2DisAll, want: "\x1b[13u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEscape, unshifted_codepoint: 0x1FFFFF}, opts: ke2DisAll, want: "\x1b[27u"},
	})
}

// Zig: test "term51: kitty: composing only passes modifier keys with report_all"
func TestKe2_term51_kitty_composing_only_passes_modifier_keys_with_report_all(t *testing.T) {
	ke2RunCases(t, inputKeyEncodeKitty, []ke2Case{
		// Modifier key while composing: needs report_all.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyShiftLeft, composing: true}, opts: ke2Dis, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyShiftLeft, composing: true}, opts: ke2DisAll, want: "\x1b[57441u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyShiftLeft, composing: true, mods: InputKeyMods{shift: true}}, opts: ke2DisAll, want: "\x1b[57441;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyCapsLock, composing: true, mods: InputKeyMods{caps_lock: true}}, opts: ke2DisAll, want: "\x1b[57358;65u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyNumLock, composing: true}, opts: ke2DisAll, want: "\x1b[57360u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyControlRight, composing: true}, opts: ke2AllOnly, want: "\x1b[57448u"},
		// Preedit text on a modifier key is only sent with report_associated.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyShiftLeft, composing: true, utf8: []uint8("x")}, opts: ke2DisAll, want: "\x1b[57441u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyShiftLeft, composing: true, utf8: []uint8("x")}, opts: ke2Assoc, want: "\x1b[57441;;120u"},
		// Everything else is swallowed.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, composing: true}, opts: ke2DisAll, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, composing: true, unshifted_codepoint: 'a'}, opts: ke2DisAll, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, composing: true, utf8: []uint8("a"), unshifted_codepoint: 'a'}, opts: ke2DisAll, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, composing: true, utf8: []uint8("a"), unshifted_codepoint: 'a'}, opts: ke2Dis, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, composing: true, utf8: []uint8("あ")}, opts: ke2Full, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEscape, composing: true}, opts: ke2DisAll, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyArrowUp, composing: true}, opts: ke2DisAll, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, composing: true, utf8: []uint8("x")}, opts: ke2DisAll, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, composing: true}, opts: ke2Dis, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyBackspace, composing: true, utf8: []uint8("x")}, opts: ke2Dis, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyUnidentified, composing: true, utf8: []uint8("x")}, opts: ke2DisAll, want: ""},
		// The release check precedes composing.
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyShiftLeft, composing: true}, opts: ke2DisAll, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyShiftLeft, composing: true}, opts: ke2EventsAll, want: "\x1b[57441;1:3u"},
	})
}

// Zig: test "term51: kitty: enter and backspace with utf8 text"
func TestKe2_term51_kitty_enter_and_backspace_with_utf8_text(t *testing.T) {
	ke2RunCases(t, inputKeyEncodeKitty, []ke2Case{
		// Enter with committed text sends the text (IME confirmation).
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, utf8: []uint8("ab")}, opts: ke2Dis, want: "ab"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, utf8: []uint8("あ")}, opts: ke2Dis, want: "あ"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, utf8: []uint8("x"), mods: InputKeyMods{ctrl: true}}, opts: ke2Dis, want: "x"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, utf8: []uint8("\x01\x02")}, opts: ke2Dis, want: "\x01\x02"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, utf8: []uint8("\x1b[")}, opts: ke2Dis, want: "\x1b["},
		// A single byte >= 0x80 is not a control character.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, utf8: []uint8("\x80")}, opts: ke2Dis, want: "\x80"},
		// This happens even for report_all and for release.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, utf8: []uint8("x")}, opts: ke2DisAll, want: "x"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, utf8: []uint8("x")}, opts: ke2Assoc, want: "x"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyEnter, utf8: []uint8("x")}, opts: ke2EventsAll, want: "x"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRepeat, key: InputKeyKeyEnter, utf8: []uint8("x")}, opts: ke2EventsAll, want: "x"},
		// A single control character is ignored as text.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, utf8: []uint8("\r")}, opts: ke2Dis, want: "\r"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, utf8: []uint8("\n")}, opts: ke2Dis, want: "\r"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, utf8: []uint8("\x1b")}, opts: ke2Dis, want: "\r"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, utf8: []uint8("\x7f")}, opts: ke2Dis, want: "\r"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, utf8: []uint8("\x1f")}, opts: ke2Dis, want: "\r"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, utf8: []uint8("\r")}, opts: ke2DisAll, want: "\x1b[13u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, utf8: []uint8("\r")}, opts: ke2Assoc, want: "\x1b[13u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyEnter, utf8: []uint8("\r")}, opts: ke2EventsAll, want: "\x1b[13;1:3u"},
		// Backspace with text sends nothing (preedit editing).
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyBackspace, utf8: []uint8("ab")}, opts: ke2Dis, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyBackspace, utf8: []uint8("あ")}, opts: ke2Dis, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyBackspace, utf8: []uint8("x")}, opts: ke2DisAll, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyBackspace, utf8: []uint8("x"), mods: InputKeyMods{ctrl: true}}, opts: ke2Dis, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyBackspace, utf8: []uint8("\x01\x02")}, opts: ke2Dis, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyBackspace, utf8: []uint8("\x80")}, opts: ke2Dis, want: ""},
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyBackspace, utf8: []uint8("x")}, opts: ke2EventsAll, want: ""},
		// ... unless the text is one control character.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyBackspace, utf8: []uint8("\x08")}, opts: ke2Dis, want: "\x7f"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyBackspace, utf8: []uint8("\x7f")}, opts: ke2Dis, want: "\x7f"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyBackspace, utf8: []uint8("\x7f")}, opts: ke2DisAll, want: "\x1b[127u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyBackspace, utf8: []uint8("\x7f")}, opts: ke2Assoc, want: "\x1b[127u"},
		// Tab is not special-cased for text.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyTab, utf8: []uint8("x")}, opts: ke2Dis, want: "\t"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyTab, utf8: []uint8("x")}, opts: ke2DisAll, want: "\x1b[9u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyTab, utf8: []uint8("x")}, opts: ke2Assoc, want: "\x1b[9;;120u"},
		// Numpad enter is not special-cased either.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyNumpadEnter, utf8: []uint8("x")}, opts: ke2Dis, want: "x"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyNumpadEnter, utf8: []uint8("\r")}, opts: ke2Dis, want: "\x1b[57414u"},
	})
}

// Zig: test "term51: kitty: enter, tab and backspace in legacy form need empty binding mods"
func TestKe2_term51_kitty_enter_tab_and_backspace_in_legacy_form_need_empty_binding_mods(t *testing.T) {
	ke2RunCases(t, inputKeyEncodeKitty, []ke2Case{
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter}, opts: ke2Dis, want: "\r"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyTab}, opts: ke2Dis, want: "\t"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyBackspace}, opts: ke2Dis, want: "\x7f"},
		// Locks and sides are not binding mods.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, mods: InputKeyMods{caps_lock: true}}, opts: ke2Dis, want: "\r"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyTab, mods: InputKeyMods{num_lock: true}}, opts: ke2Dis, want: "\t"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyBackspace, mods: InputKeyMods{sides: InputKeyModsModsSide{shift: InputKeyModsModSideRight, alt: InputKeyModsModSideRight}}}, opts: ke2Dis, want: "\x7f"},
		// Any real modifier switches to CSI u, with all modifiers listed.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, mods: InputKeyMods{shift: true}}, opts: ke2Dis, want: "\x1b[13;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, mods: InputKeyMods{ctrl: true}}, opts: ke2Dis, want: "\x1b[13;5u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyTab, mods: InputKeyMods{shift: true}}, opts: ke2Dis, want: "\x1b[9;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyTab, mods: InputKeyMods{alt: true}}, opts: ke2Dis, want: "\x1b[9;3u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyBackspace, mods: InputKeyMods{alt: true}}, opts: ke2Dis, want: "\x1b[127;3u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyBackspace, mods: InputKeyMods{super: true}}, opts: ke2Dis, want: "\x1b[127;9u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, mods: InputKeyMods{shift: true, caps_lock: true}}, opts: ke2Dis, want: "\x1b[13;66u"},
		// The consumed mods are only honoured when there is text.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, mods: InputKeyMods{shift: true}, consumed_mods: InputKeyMods{shift: true}}, opts: ke2Dis, want: "\x1b[13;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, mods: InputKeyMods{shift: true}, consumed_mods: InputKeyMods{shift: true}, utf8: []uint8("\r")}, opts: ke2Dis, want: "\r"},
		// report_all always uses CSI u.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter}, opts: ke2DisAll, want: "\x1b[13u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyTab}, opts: ke2AllOnly, want: "\x1b[9u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyBackspace}, opts: ke2AllOnly, want: "\x1b[127u"},
	})
}

// Zig: test "term51: kitty: plain text is sent directly"
func TestKe2_term51_kitty_plain_text_is_sent_directly(t *testing.T) {
	a := InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a"), unshifted_codepoint: 'a'}
	ke2RunCases(t, inputKeyEncodeKitty, []ke2Case{
		{ev: a, opts: ke2Dis, want: "a"},
		{ev: a, opts: ke2Events, want: "a"},
		{ev: a, opts: ke2AltsOnly, want: "a"},
		{ev: a, opts: ke2AssocOnly, want: "a"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("abcd"), unshifted_codepoint: 'a'}, opts: ke2Dis, want: "abcd"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("é")}, opts: ke2Dis, want: "é"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("あ")}, opts: ke2Dis, want: "あ"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("😀")}, opts: ke2Dis, want: "😀"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("\xC2\x80")}, opts: ke2Dis, want: "\xC2\x80"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeySpace, utf8: []uint8(" "), unshifted_codepoint: ' '}, opts: ke2Dis, want: " "},
		{ev: InputKeyKeyEvent{action: InputKeyActionRepeat, key: InputKeyKeyKeyA, utf8: []uint8("a"), unshifted_codepoint: 'a'}, opts: ke2Dis, want: "a"},
		// Non-binding mods do not prevent plain text.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("A"), mods: InputKeyMods{caps_lock: true}, unshifted_codepoint: 'a'}, opts: ke2Dis, want: "A"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a"), mods: InputKeyMods{num_lock: true}, unshifted_codepoint: 'a'}, opts: ke2Dis, want: "a"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a"), mods: InputKeyMods{sides: InputKeyModsModsSide{alt: InputKeyModsModSideRight}}, unshifted_codepoint: 'a'}, opts: ke2Dis, want: "a"},
		// Shift consumed to produce the text: still plain text.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("A"), mods: InputKeyMods{shift: true}, consumed_mods: InputKeyMods{shift: true}, unshifted_codepoint: 'a'}, opts: ke2Dis, want: "A"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("\xC3\xA5"), mods: InputKeyMods{alt: true}, consumed_mods: InputKeyMods{alt: true}, unshifted_codepoint: 'a'}, opts: ke2Dis, want: "\xC3\xA5"},
		// Shift not consumed: CSI u.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("A"), mods: InputKeyMods{shift: true}, unshifted_codepoint: 'a'}, opts: ke2Dis, want: "\x1b[97;2u"},
		// Mods other than shift always make CSI u (the all-mods value is used).
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a"), mods: InputKeyMods{alt: true}, unshifted_codepoint: 'a'}, opts: ke2Dis, want: "\x1b[97;3u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a"), mods: InputKeyMods{super: true}, unshifted_codepoint: 'a'}, opts: ke2Dis, want: "\x1b[97;9u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("\x01"), mods: InputKeyMods{ctrl: true}, unshifted_codepoint: 'a'}, opts: ke2Dis, want: "\x1b[97;5u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a"), mods: InputKeyMods{ctrl: true, caps_lock: true}, unshifted_codepoint: 'a'}, opts: ke2Dis, want: "\x1b[97;69u"},
		// Consumed mods are ignored without text.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, mods: InputKeyMods{ctrl: true}, consumed_mods: InputKeyMods{ctrl: true}, unshifted_codepoint: 'a'}, opts: ke2Dis, want: "\x1b[97;5u"},
		// With report_all plain text is encoded as CSI u instead.
		{ev: a, opts: ke2DisAll, want: "\x1b[97u"},
		{ev: a, opts: ke2AllOnly, want: "\x1b[97u"},
	})
}

// Zig: test "term51: kitty: control characters and invalid utf8 are not plain text"
func TestKe2_term51_kitty_control_characters_and_invalid_utf8_are_not_plain_text(t *testing.T) {
	k_a := InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, unshifted_codepoint: 'a'}
	texts := []string{
		"\x00", "\x01", "\x1b", "\x1f", "\x7f", "a\x01", "\x01a", "ab\x7f",
		// invalid UTF-8
		"\xff", "a\xff", "\xC0\x80", "\xED\xA0\x80", "\xE3\x81", "\xF4\x90\x80\x80", "\x80",
	}
	for _, text := range texts {
		ev := k_a
		ev.utf8 = []uint8(text)
		ke2Expect(t, inputKeyEncodeKitty, "\x1b[97u", ev, ke2Dis)
	}
	// First printable at 0x20 and DEL boundary.
	ev := k_a
	ev.utf8 = []uint8(" ")
	ke2Expect(t, inputKeyEncodeKitty, " ", ev, ke2Dis)
	ev.utf8 = []uint8("\x7e")
	ke2Expect(t, inputKeyEncodeKitty, "\x7e", ev, ke2Dis)
	ev.utf8 = []uint8("\x7f")
	ke2Expect(t, inputKeyEncodeKitty, "\x1b[97u", ev, ke2Dis)
	ev.utf8 = []uint8("\x1f")
	ke2Expect(t, inputKeyEncodeKitty, "\x1b[97u", ev, ke2Dis)
	// With report_associated a control char or invalid text just disappears.
	ev.utf8 = []uint8("\x01")
	ke2Expect(t, inputKeyEncodeKitty, "\x1b[97u", ev, ke2Assoc)
	ev.utf8 = []uint8("\xff")
	ke2Expect(t, inputKeyEncodeKitty, "\x1b[97u", ev, ke2Assoc)
	// Mixed controls are dropped from the associated text.
	ev.utf8 = []uint8("a\x01b")
	ke2Expect(t, inputKeyEncodeKitty, "\x1b[97;;97:98u", ev, ke2Assoc)
}

// Zig: test "term51: kitty: no table entry and no unshifted codepoint passes text through"
func TestKe2_term51_kitty_no_table_entry_and_no_unshifted_codepoint_passes_text_through(t *testing.T) {
	opts_list := []InputKeyEncodeOptions{ke2Dis, ke2DisAll, ke2Assoc, ke2Full, ke2Alts}
	for _, o := range opts_list {
		ke2Expect(t, inputKeyEncodeKitty, "x", InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("x")}, o)
		ke2Expect(t, inputKeyEncodeKitty, "ab", InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("ab")}, o)
		ke2Expect(t, inputKeyEncodeKitty, "あ", InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyUnidentified, utf8: []uint8("あ")}, o)
		ke2Expect(t, inputKeyEncodeKitty, "", InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA}, o)
		ke2Expect(t, inputKeyEncodeKitty, "", InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyUnidentified}, o)
		// Modifiers do not change it.
		ke2Expect(t, inputKeyEncodeKitty, "x", InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("x"), mods: InputKeyMods{ctrl: true}}, o)
		ke2Expect(t, inputKeyEncodeKitty, "x", InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("x"), mods: InputKeyMods{shift: true, alt: true}}, o)
		// Control characters and even invalid UTF-8 are written verbatim.
		ke2Expect(t, inputKeyEncodeKitty, "\x01", InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("\x01"), mods: InputKeyMods{ctrl: true}}, o)
		ke2Expect(t, inputKeyEncodeKitty, "\xff", InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("\xff")}, o)
	}
	// Even for a release event (a text-bearing release of a key without an
	// unshifted codepoint writes its text).
	ke2Expect(t, inputKeyEncodeKitty, "x", InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyKeyA, utf8: []uint8("x")}, ke2EventsAll)
	ke2Expect(t, inputKeyEncodeKitty, "x", InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyKeyA, utf8: []uint8("x")}, ke2Full)
	ke2Expect(t, inputKeyEncodeKitty, "x", InputKeyKeyEvent{action: InputKeyActionRepeat, key: InputKeyKeyKeyA, utf8: []uint8("x")}, ke2Full)
}

// Zig: test "term51: kitty: modifier keys need report_all"
func TestKe2_term51_kitty_modifier_keys_need_report_all(t *testing.T) {
	for _, r := range ke2Rows {
		if !r.modifier {
			continue
		}
		ke2Expect(t, inputKeyEncodeKitty, "", InputKeyKeyEvent{action: InputKeyActionPress, key: r.k}, ke2Dis)
		ke2Expect(t, inputKeyEncodeKitty, "", InputKeyKeyEvent{action: InputKeyActionPress, key: r.k}, ke2Events)
		ke2Expect(t, inputKeyEncodeKitty, "", InputKeyKeyEvent{action: InputKeyActionPress, key: r.k}, ke2DisEvents)
		ke2Expect(t, inputKeyEncodeKitty, "", InputKeyKeyEvent{action: InputKeyActionPress, key: r.k}, ke2AltsOnly)
		ke2Expect(t, inputKeyEncodeKitty, "", InputKeyKeyEvent{action: InputKeyActionPress, key: r.k}, ke2AssocOnly)
		ke2Expect(t, inputKeyEncodeKitty, r.seq, InputKeyKeyEvent{action: InputKeyActionPress, key: r.k}, ke2AllOnly)
	}
	// Modifiers on the modifier key itself.
	ke2RunCases(t, inputKeyEncodeKitty, []ke2Case{
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyShiftLeft, mods: InputKeyMods{shift: true}}, opts: ke2DisAll, want: "\x1b[57441;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyShiftRight, mods: InputKeyMods{shift: true, sides: InputKeyModsModsSide{shift: InputKeyModsModSideRight}}}, opts: ke2DisAll, want: "\x1b[57447;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyControlLeft, mods: InputKeyMods{ctrl: true}}, opts: ke2DisAll, want: "\x1b[57442;5u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyAltLeft, mods: InputKeyMods{alt: true}}, opts: ke2DisAll, want: "\x1b[57443;3u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyMetaLeft, mods: InputKeyMods{super: true}}, opts: ke2DisAll, want: "\x1b[57444;9u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyMetaRight, mods: InputKeyMods{super: true}}, opts: ke2DisAll, want: "\x1b[57450;9u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyCapsLock, mods: InputKeyMods{caps_lock: true}}, opts: ke2DisAll, want: "\x1b[57358;65u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyNumLock, mods: InputKeyMods{num_lock: true}}, opts: ke2DisAll, want: "\x1b[57360;129u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyShiftLeft, mods: InputKeyMods{shift: true}}, opts: ke2EventsAll, want: "\x1b[57441;2:3u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRepeat, key: InputKeyKeyShiftLeft}, opts: ke2EventsAll, want: "\x1b[57441;1:2u"},
		// Without report_all a release of a modifier key is dropped too.
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyShiftLeft}, opts: ke2DisEvents, want: ""},
		// A modifier key with text and no report_all falls to plain text.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyShiftLeft, utf8: []uint8("x")}, opts: ke2Dis, want: "x"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyShiftLeft, utf8: []uint8("x")}, opts: ke2DisAll, want: "\x1b[57441u"},
	})
}

// Zig: test "term51: kitty: modifier combinations with report_all"
func TestKe2_term51_kitty_modifier_combinations_with_report_all(t *testing.T) {
	for bits := uint8(0); bits < 64; bits++ {
		b := bits
		m := ke2KittyValue(b)
		mods := ke2Mods(b)

		var esc string
		if m == 1 {
			esc = fmt.Sprintf("\x1b[27u")
		} else {
			esc = fmt.Sprintf("\x1b[27;%du", m)
		}
		ke2Expect(t, inputKeyEncodeKitty, esc, InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEscape, mods: mods}, ke2DisAll)
		ke2Expect(t, inputKeyEncodeKitty, esc, InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEscape, mods: mods}, ke2AllOnly)

		// All of the modifier bits flow through unchanged for the other forms.
		var arrow string
		if m == 1 {
			arrow = fmt.Sprintf("\x1b[A")
		} else {
			arrow = fmt.Sprintf("\x1b[1;%dA", m)
		}
		ke2Expect(t, inputKeyEncodeKitty, arrow, InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyArrowUp, mods: mods}, ke2DisAll)

		// Left/right sides are never reported.
		sided := mods
		sided.sides = InputKeyModsModsSide{shift: InputKeyModsModSideRight, ctrl: InputKeyModsModSideRight, alt: InputKeyModsModSideRight, super: InputKeyModsModSideRight}
		ke2Expect(t, inputKeyEncodeKitty, esc, InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEscape, mods: sided}, ke2DisAll)
	}

	// Spot checks with literal values.
	ke2RunCases(t, inputKeyEncodeKitty, []ke2Case{
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEscape, mods: InputKeyMods{shift: true}}, opts: ke2DisAll, want: "\x1b[27;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEscape, mods: InputKeyMods{alt: true}}, opts: ke2DisAll, want: "\x1b[27;3u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEscape, mods: InputKeyMods{ctrl: true}}, opts: ke2DisAll, want: "\x1b[27;5u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEscape, mods: InputKeyMods{super: true}}, opts: ke2DisAll, want: "\x1b[27;9u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEscape, mods: InputKeyMods{caps_lock: true}}, opts: ke2DisAll, want: "\x1b[27;65u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEscape, mods: InputKeyMods{num_lock: true}}, opts: ke2DisAll, want: "\x1b[27;129u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEscape, mods: InputKeyMods{shift: true, ctrl: true, alt: true, super: true, caps_lock: true, num_lock: true}}, opts: ke2DisAll, want: "\x1b[27;208u"},
		// Padding bits do not leak.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEscape, mods: InputKeyModsModsFromInt(0xFFFF)}, opts: ke2DisAll, want: "\x1b[27;208u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyArrowUp, mods: InputKeyModsModsFromInt(0xFFFF)}, opts: ke2DisAll, want: "\x1b[1;208A"},
	})
}

// Zig: test "term51: kitty: consumed mods only affect the plain text decisions"
func TestKe2_term51_kitty_consumed_mods_only_affect_the_plain_text_decisions(t *testing.T) {
	ke2RunCases(t, inputKeyEncodeKitty, []ke2Case{
		// The CSI u modifier value always uses all mods, never subtracting
		// the consumed ones.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("A"), mods: InputKeyMods{shift: true}, consumed_mods: InputKeyMods{shift: true}, unshifted_codepoint: 'a'}, opts: ke2DisAll, want: "\x1b[97;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("A"), mods: InputKeyMods{shift: true}, consumed_mods: InputKeyMods{shift: true}, unshifted_codepoint: 'a'}, opts: ke2Assoc, want: "\x1b[97;2;65u"},
		// Consuming a mod the event does not have changes nothing.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a"), consumed_mods: InputKeyMods{ctrl: true, shift: true}, unshifted_codepoint: 'a'}, opts: ke2Dis, want: "a"},
		// Consumed ctrl+shift: both removed so plain text goes out.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("x"), mods: InputKeyMods{ctrl: true, shift: true}, consumed_mods: InputKeyMods{ctrl: true, shift: true}, unshifted_codepoint: 'a'}, opts: ke2Dis, want: "x"},
		// Consuming only one of two still leaves a binding mod.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("x"), mods: InputKeyMods{ctrl: true, shift: true}, consumed_mods: InputKeyMods{shift: true}, unshifted_codepoint: 'a'}, opts: ke2Dis, want: "\x1b[97;6u"},
		// Consumed mods do not matter without text.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, mods: InputKeyMods{ctrl: true}, consumed_mods: InputKeyMods{ctrl: true}}, opts: ke2Dis, want: "\x1b[13;5u"},
		// Consumed mods do not change the associated text rule (alt prevents).
		// Zig-only (Darwin): on macOS this case wants "\x1b[119;3;8721u" (want = if (builtin.os.tag == .macos) ... else ...).
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyW, utf8: []uint8("\xE2\x88\x91"), mods: InputKeyMods{alt: true}, consumed_mods: InputKeyMods{alt: true}, unshifted_codepoint: 'w'}, opts: ke2Assoc, want: "\x1b[119;3u"},
	})
}

// Zig: test "term51: kitty: report_events special form for the table with report_all"
func TestKe2_term51_kitty_report_events_special_form_for_the_table_with_report_all(t *testing.T) {
	for _, r := range ke2Rows {
		last := r.seq[len(r.seq)-1]
		var press string
		if last == 'u' || last == '~' {
			press = r.seq
		} else {
			press = fmt.Sprintf("\x1b[1;1:1%c", last)
		}
		ke2Expect(t, inputKeyEncodeKitty, press, InputKeyKeyEvent{action: InputKeyActionPress, key: r.k}, ke2EventsAll)
		var repeat string
		if last == 'u' || last == '~' {
			repeat = fmt.Sprintf("%s;1:2%c", r.seq[0:len(r.seq)-1], last)
		} else {
			repeat = fmt.Sprintf("\x1b[1;1:2%c", last)
		}
		ke2Expect(t, inputKeyEncodeKitty, repeat, InputKeyKeyEvent{action: InputKeyActionRepeat, key: r.k}, ke2EventsAll)
		var release string
		if last == 'u' || last == '~' {
			release = fmt.Sprintf("%s;1:3%c", r.seq[0:len(r.seq)-1], last)
		} else {
			release = fmt.Sprintf("\x1b[1;1:3%c", last)
		}
		ke2Expect(t, inputKeyEncodeKitty, release, InputKeyKeyEvent{action: InputKeyActionRelease, key: r.k}, ke2EventsAll)
	}
}

// Zig: test "term51: kitty: alternate keys"
func TestKe2_term51_kitty_alternate_keys(t *testing.T) {
	ke2RunCases(t, inputKeyEncodeKitty, []ke2Case{
		// No alternates when nothing differs.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a"), unshifted_codepoint: 'a'}, opts: ke2Alts, want: "\x1b[97u"},
		// Shifted alternate needs shift and a differing first codepoint.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("A"), mods: InputKeyMods{shift: true}, unshifted_codepoint: 'a'}, opts: ke2Alts, want: "\x1b[97:65;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a"), mods: InputKeyMods{shift: true}, unshifted_codepoint: 'a'}, opts: ke2Alts, want: "\x1b[97;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("A"), unshifted_codepoint: 'a'}, opts: ke2Alts, want: "\x1b[97u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("A"), mods: InputKeyMods{caps_lock: true}, unshifted_codepoint: 'a'}, opts: ke2Alts, want: "\x1b[97;65u"},
		// A consumed shift is still reported as shift.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("A"), mods: InputKeyMods{shift: true}, consumed_mods: InputKeyMods{shift: true}, unshifted_codepoint: 'a'}, opts: ke2Alts, want: "\x1b[97:65;2u"},
		// Base layout key: a Cyrillic layout on the physical 'a' key.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("ф"), unshifted_codepoint: 1092}, opts: ke2Alts, want: "\x1b[1092::97u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("Ф"), mods: InputKeyMods{shift: true}, unshifted_codepoint: 1092}, opts: ke2Alts, want: "\x1b[1092:1060:97;2u"},
		// The base key is skipped when the text equals it.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a"), unshifted_codepoint: 1092}, opts: ke2Alts, want: "\x1b[1092u"},
		// Base key vs a different physical key.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyB, utf8: []uint8("a"), unshifted_codepoint: 'a'}, opts: ke2Alts, want: "\x1b[97::98u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyB, utf8: []uint8("A"), mods: InputKeyMods{shift: true}, unshifted_codepoint: 'a'}, opts: ke2Alts, want: "\x1b[97:65:98;2u"},
		// More than one codepoint of text: no base key; shifted still reported.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("фф"), mods: InputKeyMods{shift: true}, unshifted_codepoint: 1092}, opts: ke2Alts, want: "\x1b[1092;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("ФФ"), mods: InputKeyMods{shift: true}, unshifted_codepoint: 1092}, opts: ke2Alts, want: "\x1b[1092:1060;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("ФФ"), unshifted_codepoint: 1092}, opts: ke2Alts, want: "\x1b[1092u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("ab"), mods: InputKeyMods{shift: true}, unshifted_codepoint: 'a'}, opts: ke2Alts, want: "\x1b[97;2u"},
		// No text: only the base layout key.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, unshifted_codepoint: 1092}, opts: ke2Alts, want: "\x1b[1092::97u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, unshifted_codepoint: 1092, mods: InputKeyMods{shift: true}}, opts: ke2Alts, want: "\x1b[1092::97;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, unshifted_codepoint: 'a'}, opts: ke2Alts, want: "\x1b[97u"},
		// Keys without a codepoint have no base key.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyUnidentified, unshifted_codepoint: 1092}, opts: ke2Alts, want: "\x1b[1092u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyF13, utf8: []uint8("x")}, opts: ke2Alts, want: "\x1b[57376u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyF13, utf8: []uint8("x"), mods: InputKeyMods{shift: true}}, opts: ke2Alts, want: "\x1b[57376:120;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyF13}, opts: ke2Alts, want: "\x1b[57376u"},
		// Keypad keys: the table code differs from the key's codepoint.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyNumpad1}, opts: ke2Alts, want: "\x1b[57400::49u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyNumpad1, utf8: []uint8("1")}, opts: ke2Alts, want: "\x1b[57400u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyNumpadDecimal}, opts: ke2Alts, want: "\x1b[57409::46u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyNumpadMultiply}, opts: ke2Alts, want: "\x1b[57411::42u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyNumpadEnter}, opts: ke2Alts, want: "\x1b[57414u"},
		// Space.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeySpace, utf8: []uint8(" "), unshifted_codepoint: ' '}, opts: ke2Alts, want: "\x1b[32u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeySpace, utf8: []uint8(" "), mods: InputKeyMods{shift: true}, unshifted_codepoint: ' '}, opts: ke2Alts, want: "\x1b[32;2u"},
		// Invalid text: no alternates at all.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("\xff"), unshifted_codepoint: 1092}, opts: ke2Alts, want: "\x1b[1092u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("\xED\xA0\x80"), mods: InputKeyMods{shift: true}, unshifted_codepoint: 1092}, opts: ke2Alts, want: "\x1b[1092;2u"},
		// Not enabled: nothing.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("Ф"), mods: InputKeyMods{shift: true}, unshifted_codepoint: 1092}, opts: ke2DisAll, want: "\x1b[1092;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, unshifted_codepoint: 1092}, opts: ke2DisAll, want: "\x1b[1092u"},
		// Alone (no report_all) with no text and with plain text.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, unshifted_codepoint: 1092}, opts: ke2AltsOnly, want: "\x1b[1092::97u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("ф"), unshifted_codepoint: 1092}, opts: ke2AltsOnly, want: "ф"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("Ф"), mods: InputKeyMods{shift: true}, unshifted_codepoint: 1092}, opts: ke2AltsOnly, want: "\x1b[1092:1060:97;2u"},
	})
}

// Zig: test "term51: kitty: alternate keys are skipped for control key codes"
func TestKe2_term51_kitty_alternate_keys_are_skipped_for_control_key_codes(t *testing.T) {
	ke2RunCases(t, inputKeyEncodeKitty, []ke2Case{
		// Table keys whose code is a control (<0x20 or 0x7f): never alternates.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEscape, utf8: []uint8("x"), mods: InputKeyMods{shift: true}}, opts: ke2Alts, want: "\x1b[27;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyTab, utf8: []uint8("x"), mods: InputKeyMods{shift: true}}, opts: ke2Alts, want: "\x1b[9;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyBackspace, mods: InputKeyMods{shift: true}}, opts: ke2Alts, want: "\x1b[127;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, mods: InputKeyMods{shift: true}}, opts: ke2Alts, want: "\x1b[13;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyF3, utf8: []uint8("x"), mods: InputKeyMods{shift: true}}, opts: ke2Alts, want: "\x1b[13;2~"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyInsert, utf8: []uint8("x"), mods: InputKeyMods{shift: true}}, opts: ke2Alts, want: "\x1b[2;2~"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyArrowUp, utf8: []uint8("x"), mods: InputKeyMods{shift: true}}, opts: ke2Alts, want: "\x1b[1;2A"},
		// Unshifted codepoints around the control boundary.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("A"), mods: InputKeyMods{shift: true}, unshifted_codepoint: 0x1F}, opts: ke2Alts, want: "\x1b[31;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("A"), mods: InputKeyMods{shift: true}, unshifted_codepoint: 0x20}, opts: ke2Alts, want: "\x1b[32:65:97;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("A"), mods: InputKeyMods{shift: true}, unshifted_codepoint: 0x7E}, opts: ke2Alts, want: "\x1b[126:65:97;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("A"), mods: InputKeyMods{shift: true}, unshifted_codepoint: 0x7F}, opts: ke2Alts, want: "\x1b[127;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("A"), mods: InputKeyMods{shift: true}, unshifted_codepoint: 0x80}, opts: ke2Alts, want: "\x1b[128:65:97;2u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, unshifted_codepoint: 0x01}, opts: ke2Alts, want: "\x1b[1u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, unshifted_codepoint: 0x20}, opts: ke2Alts, want: "\x1b[32::97u"},
	})
}

// Zig: test "term51: kitty: associated text"
func TestKe2_term51_kitty_associated_text(t *testing.T) {
	ke2RunCases(t, inputKeyEncodeKitty, []ke2Case{
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a"), unshifted_codepoint: 'a'}, opts: ke2Assoc, want: "\x1b[97;;97u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("A"), mods: InputKeyMods{shift: true}, unshifted_codepoint: 'a'}, opts: ke2Assoc, want: "\x1b[97;2;65u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("A"), mods: InputKeyMods{caps_lock: true}, unshifted_codepoint: 'a'}, opts: ke2Assoc, want: "\x1b[97;65;65u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a"), mods: InputKeyMods{num_lock: true}, unshifted_codepoint: 'a'}, opts: ke2Assoc, want: "\x1b[97;129;97u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("A"), mods: InputKeyMods{shift: true, caps_lock: true, num_lock: true}, unshifted_codepoint: 'a'}, opts: ke2Assoc, want: "\x1b[97;194;65u"},
		// Multiple codepoints, multi-byte and a control char among them.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("ab"), unshifted_codepoint: 'a'}, opts: ke2Assoc, want: "\x1b[97;;97:98u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("ф"), unshifted_codepoint: 1092}, opts: ke2Assoc, want: "\x1b[1092;;1092u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a\x01b"), unshifted_codepoint: 'a'}, opts: ke2Assoc, want: "\x1b[97;;97:98u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("\x01"), unshifted_codepoint: 'a'}, opts: ke2Assoc, want: "\x1b[97u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, unshifted_codepoint: 'a'}, opts: ke2Assoc, want: "\x1b[97u"},
		// Text-preventing modifiers: alt (on this platform, see below), ctrl, super.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a"), mods: InputKeyMods{ctrl: true}, unshifted_codepoint: 'a'}, opts: ke2Assoc, want: "\x1b[97;5u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a"), mods: InputKeyMods{super: true}, unshifted_codepoint: 'a'}, opts: ke2Assoc, want: "\x1b[97;9u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a"), mods: InputKeyMods{ctrl: true, shift: true}, unshifted_codepoint: 'a'}, opts: ke2Assoc, want: "\x1b[97;6u"},
		// Without report_associated no text.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a"), unshifted_codepoint: 'a'}, opts: ke2DisAll, want: "\x1b[97u"},
		// report_associated alone behaves like plain text without modifiers.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a"), unshifted_codepoint: 'a'}, opts: ke2AssocOnly, want: "a"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("A"), mods: InputKeyMods{shift: true}, unshifted_codepoint: 'a'}, opts: ke2AssocOnly, want: "\x1b[97;2;65u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a"), mods: InputKeyMods{ctrl: true}, unshifted_codepoint: 'a'}, opts: ke2AssocOnly, want: "\x1b[97;5u"},
		// Events: press omits ":1", repeat includes text, release drops text.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a"), unshifted_codepoint: 'a'}, opts: ke2Full, want: "\x1b[97;;97u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRepeat, key: InputKeyKeyKeyA, utf8: []uint8("a"), unshifted_codepoint: 'a'}, opts: ke2Full, want: "\x1b[97;1:2;97u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyKeyA, utf8: []uint8("a"), unshifted_codepoint: 'a'}, opts: ke2Full, want: "\x1b[97;1:3u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("A"), mods: InputKeyMods{shift: true}, unshifted_codepoint: 'a'}, opts: ke2Full, want: "\x1b[97:65;2;65u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRepeat, key: InputKeyKeyKeyA, utf8: []uint8("A"), mods: InputKeyMods{shift: true}, unshifted_codepoint: 'a'}, opts: ke2Full, want: "\x1b[97:65;2:2;65u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyKeyA, utf8: []uint8("A"), mods: InputKeyMods{shift: true}, unshifted_codepoint: 'a'}, opts: ke2Full, want: "\x1b[97:65;2:3u"},
		// Special keys ignore the text, the '~' keys carry it.
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyArrowUp, utf8: []uint8("x")}, opts: ke2Assoc, want: "\x1b[A"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyArrowUp, utf8: []uint8("x")}, opts: ke2Full, want: "\x1b[1;1:1A"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyDelete, utf8: []uint8("x")}, opts: ke2Assoc, want: "\x1b[3;;120~"},
		{ev: InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyDelete, utf8: []uint8("x")}, opts: ke2Full, want: "\x1b[3;1:3~"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEscape, utf8: []uint8("\x1b")}, opts: ke2Assoc, want: "\x1b[27u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEscape, utf8: []uint8("\x1b")}, opts: ke2Dis, want: "\x1b[27u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyF13, utf8: []uint8("x")}, opts: ke2Assoc, want: "\x1b[57376;;120u"},
		// Modifier keys during composing carry preedit text (see composing test).
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyNumpad1, utf8: []uint8("1")}, opts: ke2Full, want: "\x1b[57400;;49u"},
	})
}

// Zig: test "term51: kitty: macos_option_as_alt and associated text with alt"
func TestKe2_term51_kitty_macos_option_as_alt_and_associated_text_with_alt(t *testing.T) {
	for _, oa := range []InputConfigOptionAsAlt{InputConfigOptionAsAltFalse, InputConfigOptionAsAltTrue, InputConfigOptionAsAltLeft, InputConfigOptionAsAltRight} {
		for _, sides := range []InputKeyModsModsSide{{}, {alt: InputKeyModsModSideRight}} {
			opts := InputKeyEncodeOptions{
				kitty_flags:         KittyKeyFlags{disambiguate: true, report_all: true, report_associated: true},
				macos_option_as_alt: oa,
			}
			ev := InputKeyKeyEvent{
				action:              InputKeyActionPress,
				key:                 InputKeyKeyKeyW,
				mods:                InputKeyMods{alt: true, sides: sides},
				utf8:                []uint8("∑"),
				unshifted_codepoint: 119,
			}
			var want string
			if ke2AltPrevents(oa, sides.alt) {
				want = "\x1b[119;3u"
			} else {
				want = "\x1b[119;3;8721u"
			}
			ke2Expect(t, inputKeyEncodeKitty, want, ev, opts)
			ke2Expect(t, InputKeyEncodeEncode, want, ev, opts)

			// Other text-preventing modifiers are never affected.
			ev2 := ev
			ev2.mods.alt = false
			ev2.mods.ctrl = true
			ke2Expect(t, inputKeyEncodeKitty, "\x1b[119;5u", ev2, opts)
			ev2.mods.ctrl = false
			ev2.mods.super = true
			ke2Expect(t, inputKeyEncodeKitty, "\x1b[119;9u", ev2, opts)
			// Without alt the text is always reported.
			ev2.mods = InputKeyMods{shift: true, sides: sides}
			ke2Expect(t, inputKeyEncodeKitty, "\x1b[119;2;8721u", ev2, opts)
		}
	}
	// On non-macOS builds every value behaves the same (alt prevents text).
	ke2Expect(t, inputKeyEncodeKitty, "\x1b[119;3u", InputKeyKeyEvent{
		action:              InputKeyActionPress,
		key:                 InputKeyKeyKeyW,
		mods:                InputKeyMods{alt: true},
		utf8:                []uint8("∑"),
		unshifted_codepoint: 119,
	}, InputKeyEncodeOptions{
		kitty_flags:         KittyKeyFlags{disambiguate: true, report_all: true, report_associated: true},
		macos_option_as_alt: InputConfigOptionAsAltFalse,
	})
}

// Zig: test "term51: kitty: option flags other than kitty_flags are ignored"
func TestKe2_term51_kitty_option_flags_other_than_kitty_flags_are_ignored(t *testing.T) {
	base := ke2DisAll
	o := base
	o.cursor_key_application = true
	o.keypad_key_application = true
	o.ignore_keypad_with_numlock = true
	o.alt_esc_prefix = true
	o.modify_other_keys_state_2 = true
	ke2RunCases(t, inputKeyEncodeKitty, []ke2Case{
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyArrowUp}, opts: o, want: "\x1b[A"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyNumpad1, utf8: []uint8("1")}, opts: o, want: "\x1b[57400u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyC, utf8: []uint8("c"), mods: InputKeyMods{alt: true}, unshifted_codepoint: 'c'}, opts: o, want: "\x1b[99;3u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyDigit8, utf8: []uint8("8"), mods: InputKeyMods{alt: true}, unshifted_codepoint: '8'}, opts: o, want: "\x1b[56;3u"},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyH, utf8: []uint8("H"), mods: InputKeyMods{ctrl: true, shift: true}, unshifted_codepoint: 'h'}, opts: o, want: "\x1b[104;6u"},
	})
}

// Zig: test "term51: kitty: sides and padding bits never change the output"
func TestKe2_term51_kitty_sides_and_padding_bits_never_change_the_output(t *testing.T) {
	var sides_all InputKeyModsModsSide
	sides_all = InputKeyModsModsSideFromInt(0xF)
	ev := InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("A"), unshifted_codepoint: 'a', mods: InputKeyMods{shift: true, ctrl: true}}
	ev2 := ev
	ev2.mods.sides = sides_all
	ev2.consumed_mods.sides = sides_all
	ke2Expect(t, inputKeyEncodeKitty, "\x1b[97:65;6u", ev, ke2Alts)
	ke2Expect(t, inputKeyEncodeKitty, "\x1b[97:65;6u", ev2, ke2Alts)
	// Consumed side bits do not cancel the real modifiers, since binding
	// strips sides: only the consumed shift/ctrl bits matter.
	ke2Expect(t, inputKeyEncodeKitty, "A", InputKeyKeyEvent{
		action:              InputKeyActionPress,
		key:                 InputKeyKeyKeyA,
		utf8:                []uint8("A"),
		unshifted_codepoint: 'a',
		mods:                InputKeyMods{shift: true, sides: sides_all},
		consumed_mods:       InputKeyMods{shift: true},
	}, ke2Dis)
}

// Zig: test "term51: kitty: write errors with a too small buffer"
func TestKe2_term51_kitty_write_errors_with_a_too_small_buffer(t *testing.T) {
	// An empty buffer fails for every non-empty output, in every code path.
	var empty [0]uint8
	cases := []struct {
		ev   InputKeyKeyEvent
		opts InputKeyEncodeOptions
	}{
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter}, opts: ke2Dis},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyTab}, opts: ke2Dis},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyBackspace}, opts: ke2Dis},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEnter, utf8: []uint8("xy")}, opts: ke2Dis},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a")}, opts: ke2Dis},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a")}, opts: ke2AltsOnly},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("x")}, opts: ke2DisAll},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEscape}, opts: ke2Dis},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyArrowUp}, opts: ke2Dis},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, utf8: []uint8("a"), unshifted_codepoint: 'a'}, opts: ke2Assoc},
		{ev: InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEscape}, opts: InputKeyEncodeOptions{}},
	}
	for _, c := range cases {
		w := oscWriterFixed(empty[:])
		ke2ExpectWriteFailed(t, inputKeyEncodeKitty(&w, c.ev, c.opts))
	}
	// And no error (and no output) for events that emit nothing.
	w := oscWriterFixed(empty[:])
	zigTry(t, inputKeyEncodeKitty(&w, InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyShiftLeft}, ke2Dis))
	zigTry(t, inputKeyEncodeKitty(&w, InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA, composing: true}, ke2DisAll))
	zigTry(t, inputKeyEncodeKitty(&w, InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyEscape}, ke2Dis))
	zigTry(t, inputKeyEncodeKitty(&w, InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyKeyA}, ke2DisAll))
	zigTry(t, inputKeyEncodeKitty(&w, InputKeyKeyEvent{action: InputKeyActionRelease, key: InputKeyKeyEscape}, InputKeyEncodeOptions{}))
	expectEqual(t, 0, len(w.buffered()))

	// Cut off at each position of a long sequence.
	ke2Expect(t, inputKeyEncodeKitty, "\x1b[97:65;2:2;65:66u", InputKeyKeyEvent{action: InputKeyActionRepeat, key: InputKeyKeyKeyB, utf8: []uint8("AB"), mods: InputKeyMods{shift: true}, unshifted_codepoint: 'a'}, ke2Full)
}

// Zig: test "term51: encode: disabled kitty flags use legacy, any flag uses kitty"
func TestKe2_term51_encode_disabled_kitty_flags_use_legacy_any_flag_uses_kitty(t *testing.T) {
	for bits := uint8(0); bits < 32; bits++ {
		flags := KittyKeyFlagsFromInt(bits)
		opts := InputKeyEncodeOptions{kitty_flags: flags}
		esc_want := "\x1b[27u"
		if bits == 0 {
			esc_want = "\x1b"
		}
		ke2Expect(t, InputKeyEncodeEncode, esc_want, InputKeyKeyEvent{action: InputKeyActionPress, key: InputKeyKeyEscape}, opts)

		// ctrl+a: C0 byte in legacy, CSI u in kitty.
		ctrl_a := InputKeyKeyEvent{
			action:              InputKeyActionPress,
			key:                 InputKeyKeyKeyA,
			mods:                InputKeyMods{ctrl: true},
			utf8:                []uint8("a"),
			unshifted_codepoint: 'a',
		}
		ca_want := "\x1b[97;5u"
		if bits == 0 {
			ca_want = "\x01"
		}
		ke2Expect(t, InputKeyEncodeEncode, ca_want, ctrl_a, opts)
	}
}
