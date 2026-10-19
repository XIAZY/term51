// Tests translated one for one from src/terminal/modes.zig (Ghostty v1.3.1
// plus the term51 additional tests). Each Go test carries the exact Zig title
// in a "// Zig: test" comment.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualStrings keep Zig's names and Zig's
//     (expected, actual) argument order (helpers in Parser_test.go).
//   - `ModePacked` is one bit per mode in Go as in Zig (bit i for entry i).
//     term51EBits (`@bitCast(p)` to u40) is sm1_modesBits and the reverse
//     `@bitCast` from u40 is sm1_modesFromBits. `p.name` is
//     sm1_modesField(&p, "name") and `p.name = v` is sm1_modesFieldSet;
//     sm1_modesFieldNames lists the names in declaration order. The layout
//     test checks that the bit of each name is its position in that list,
//     the Go counterpart of `@bitOffsetOf`.
//   - `ModePacked = .{}` and `ModeState = .{}` are ModesModePackedInit() and
//     ModesModeStateInit(): a Go zero value is all false, while six modes
//     default to true.
//   - `@typeInfo(Mode).@"enum".fields` is ModesModeValues() (name by String(),
//     value by integer conversion).
//   - The Zig field `default` is `default_` (Go keyword); the field
//     `132_column` is `x132_column`.

package terminal

import (
	"math/bits"
	"reflect"
	"testing"
	"unsafe"
)

// sm1_modeRow is term51EModeRow.
type sm1_modeRow struct {
	mode     ModesMode
	name     string
	value    uint16
	ansi     bool
	default_ bool
}

// sm1_modeRows is term51E_mode_rows: every mode in declaration order (the
// order of `entries`). The index of a row is the bit position of the mode in
// ModePacked.
var sm1_modeRows = [...]sm1_modeRow{
	{ModesModeDisableKeyboard, "disable_keyboard", 2, true, false},
	{ModesModeInsert, "insert", 4, true, false},
	{ModesModeSendReceiveMode, "send_receive_mode", 12, true, true},
	{ModesModeLinefeed, "linefeed", 20, true, false},
	{ModesModeCursorKeys, "cursor_keys", 1, false, false},
	{ModesMode132Column, "132_column", 3, false, false},
	{ModesModeSlowScroll, "slow_scroll", 4, false, false},
	{ModesModeReverseColors, "reverse_colors", 5, false, false},
	{ModesModeOrigin, "origin", 6, false, false},
	{ModesModeWraparound, "wraparound", 7, false, true},
	{ModesModeAutorepeat, "autorepeat", 8, false, false},
	{ModesModeMouseEventX10, "mouse_event_x10", 9, false, false},
	{ModesModeCursorBlinking, "cursor_blinking", 12, false, false},
	{ModesModeCursorVisible, "cursor_visible", 25, false, true},
	{ModesModeEnableMode3, "enable_mode_3", 40, false, false},
	{ModesModeReverseWrap, "reverse_wrap", 45, false, false},
	{ModesModeAltScreenLegacy, "alt_screen_legacy", 47, false, false},
	{ModesModeKeypadKeys, "keypad_keys", 66, false, false},
	{ModesModeEnableLeftAndRightMargin, "enable_left_and_right_margin", 69, false, false},
	{ModesModeMouseEventNormal, "mouse_event_normal", 1000, false, false},
	{ModesModeMouseEventButton, "mouse_event_button", 1002, false, false},
	{ModesModeMouseEventAny, "mouse_event_any", 1003, false, false},
	{ModesModeFocusEvent, "focus_event", 1004, false, false},
	{ModesModeMouseFormatUtf8, "mouse_format_utf8", 1005, false, false},
	{ModesModeMouseFormatSgr, "mouse_format_sgr", 1006, false, false},
	{ModesModeMouseAlternateScroll, "mouse_alternate_scroll", 1007, false, true},
	{ModesModeMouseFormatUrxvt, "mouse_format_urxvt", 1015, false, false},
	{ModesModeMouseFormatSgrPixels, "mouse_format_sgr_pixels", 1016, false, false},
	{ModesModeIgnoreKeypadWithNumlock, "ignore_keypad_with_numlock", 1035, false, true},
	{ModesModeAltEscPrefix, "alt_esc_prefix", 1036, false, true},
	{ModesModeAltSendsEscape, "alt_sends_escape", 1039, false, false},
	{ModesModeReverseWrapExtended, "reverse_wrap_extended", 1045, false, false},
	{ModesModeAltScreen, "alt_screen", 1047, false, false},
	{ModesModeSaveCursor, "save_cursor", 1048, false, false},
	{ModesModeAltScreenSaveCursorClearEnter, "alt_screen_save_cursor_clear_enter", 1049, false, false},
	{ModesModeBracketedPaste, "bracketed_paste", 2004, false, false},
	{ModesModeSynchronizedOutput, "synchronized_output", 2026, false, false},
	{ModesModeGraphemeCluster, "grapheme_cluster", 2027, false, false},
	{ModesModeReportColorScheme, "report_color_scheme", 2031, false, false},
	{ModesModeInBandSizeReports, "in_band_size_reports", 2048, false, false},
}

// sm1_modesFieldNames is the fields of ModePacked in declaration order (the
// order of `entries`), by their Zig names.
var sm1_modesFieldNames = [40]string{
	"disable_keyboard",
	"insert",
	"send_receive_mode",
	"linefeed",
	"cursor_keys",
	"132_column",
	"slow_scroll",
	"reverse_colors",
	"origin",
	"wraparound",
	"autorepeat",
	"mouse_event_x10",
	"cursor_blinking",
	"cursor_visible",
	"enable_mode_3",
	"reverse_wrap",
	"alt_screen_legacy",
	"keypad_keys",
	"enable_left_and_right_margin",
	"mouse_event_normal",
	"mouse_event_button",
	"mouse_event_any",
	"focus_event",
	"mouse_format_utf8",
	"mouse_format_sgr",
	"mouse_alternate_scroll",
	"mouse_format_urxvt",
	"mouse_format_sgr_pixels",
	"ignore_keypad_with_numlock",
	"alt_esc_prefix",
	"alt_sends_escape",
	"reverse_wrap_extended",
	"alt_screen",
	"save_cursor",
	"alt_screen_save_cursor_clear_enter",
	"bracketed_paste",
	"synchronized_output",
	"grapheme_cluster",
	"report_color_scheme",
	"in_band_size_reports",
}

// sm1_modesField is `@field(p, name)`: the value of the mode Zig calls name.
func sm1_modesField(p *ModesModePacked, name string) bool {
	return p.get(modesModePackedFieldBit([]uint8(name)))
}

// sm1_modesFieldSet is `@field(p, name) = value`.
func sm1_modesFieldSet(p *ModesModePacked, name string, value bool) {
	p.set(modesModePackedFieldBit([]uint8(name)), value)
}

// sm1_modesBits is term51EBits: the u40 `@bitCast` of a ModePacked (bit i is
// the i-th field in declaration order).
func sm1_modesBits(p ModesModePacked) uint64 {
	return p.bits
}

// sm1_modesFromBits is `@bitCast(@as(u40, b))` to a ModePacked.
func sm1_modesFromBits(b uint64) ModesModePacked {
	return ModesModePacked{bits: b}
}

// sm1_modesDefaultBits is term51EDefaultBits.
func sm1_modesDefaultBits() uint64 {
	var b uint64 = 0
	for i, row := range sm1_modeRows {
		if row.default_ {
			b |= uint64(1) << uint(i)
		}
	}
	return b
}

// sm1_modesReference is term51EReference.
func sm1_modesReference(v uint16, ansi bool) Optional[ModesMode] {
	for _, row := range sm1_modeRows {
		if row.value == v && row.ansi == ansi {
			return Some(row.mode)
		}
	}
	return None[ModesMode]()
}

// Not translated, Zig test (unnamed, modes.zig:79): see scripts/not-translated.txt (size of ModePacked).

// Zig: test "order"
func TestModes_order(t *testing.T) {
	tag := ModesModeTag{Value: 1}
	int_ := tag.Int()
	expectEqual(t, ModesModeTagBacking(1), int_)
}

// Not translated, Zig test (unnamed, modes.zig:235): see scripts/not-translated.txt (only forces analysis of Mode and ModePacked).

// Zig: test modeFromInt
func TestModes_modeFromInt(t *testing.T) {
	expect(t, zigUnwrap(t, ModesModeFromInt(4, true)) == ModesModeInsert, "modeFromInt(4, true).? == .insert")
	expect(t, !ModesModeFromInt(9, true).Valid, "modeFromInt(9, true) == null")
	expect(t, zigUnwrap(t, ModesModeFromInt(9, false)) == ModesModeMouseEventX10, "modeFromInt(9, false).? == .mouse_event_x10")
	expect(t, !ModesModeFromInt(14, true).Valid, "modeFromInt(14, true) == null")
}

// Zig: test ModeState
func TestModes_ModeState(t *testing.T) {
	state := ModesModeStateInit()

	// Normal set/get
	expect(t, !state.Get(ModesModeCursorKeys), "!state.get(.cursor_keys)")
	state.Set(ModesModeCursorKeys, true)
	expect(t, state.Get(ModesModeCursorKeys), "state.get(.cursor_keys)")

	// Save/restore
	state.Save(ModesModeCursorKeys)
	state.Set(ModesModeCursorKeys, false)
	expect(t, !state.Get(ModesModeCursorKeys), "!state.get(.cursor_keys)")
	expect(t, state.Restore(ModesModeCursorKeys), "state.restore(.cursor_keys)")
	expect(t, state.Get(ModesModeCursorKeys), "state.get(.cursor_keys)")
}

// Zig: test "term51: Mode: every mode has the documented tag value and order"
func TestModes_term51_Mode_every_mode_has_the_documented_tag_value_and_order(t *testing.T) {
	fields := ModesModeValues()
	expectEqual(t, 40, len(fields))
	expectEqual(t, 40, len(sm1_modeRows))
	expectEqual(t, 40, len(modesEntries))
	expect(t, reflect.TypeOf(ModesMode(0)).Kind() == reflect.Uint16, "@typeInfo(Mode).@\"enum\".tag_type == u16")
	// Zig-only: expect(@typeInfo(Mode).@"enum".is_exhaustive)

	for i, row := range sm1_modeRows {
		expectEqualStrings(t, row.name, fields[i].String())
		expectEqualStrings(t, row.name, row.mode.String())
		want := row.value
		if row.ansi {
			want |= 0x8000
		}
		expectEqual(t, want, uint16(row.mode))
		expectEqual(t, want, uint16(fields[i]))
	}

	// No duplicated (value, ansi) pairs: all tags are distinct.
	for i, a := range sm1_modeRows {
		for _, b := range sm1_modeRows[i+1:] {
			expect(t, uint16(a.mode) != uint16(b.mode), "@intFromEnum(a.mode) != @intFromEnum(b.mode)")
			expect(t, a.name != b.name, "!std.mem.eql(u8, a.name, b.name)")
		}
	}
}

// Zig: test "term51: Mode: ANSI modes are exactly the four documented"
func TestModes_term51_Mode_ANSI_modes_are_exactly_the_four_documented(t *testing.T) {
	n := 0
	for _, row := range sm1_modeRows {
		if row.ansi {
			n += 1
			expect(t, uint16(row.mode)&0x8000 != 0, "@intFromEnum(row.mode) & 0x8000 != 0")
		} else {
			expect(t, uint16(row.mode)&0x8000 == 0, "@intFromEnum(row.mode) & 0x8000 == 0")
		}
	}
	expectEqual(t, 4, n)
	expectEqual(t, uint16(0x8002), uint16(ModesModeDisableKeyboard))
	expectEqual(t, uint16(0x8004), uint16(ModesModeInsert))
	expectEqual(t, uint16(0x800C), uint16(ModesModeSendReceiveMode))
	expectEqual(t, uint16(0x8014), uint16(ModesModeLinefeed))
	// The DEC modes sharing a number with an ANSI one differ in the ansi bit.
	expectEqual(t, uint16(4), uint16(ModesModeSlowScroll))
	expectEqual(t, uint16(12), uint16(ModesModeCursorBlinking))
}

// Zig: test "term51: ModeTag: layout of the value and ansi bit"
func TestModes_term51_ModeTag_layout_of_the_value_and_ansi_bit(t *testing.T) {
	expect(t, reflect.TypeOf(ModesModeTagBacking(0)).Kind() == reflect.Uint16, "ModeTag.Backing == u16")
	expectEqual(t, uint16(1), ModesModeTag{Value: 1}.Int())
	expectEqual(t, uint16(0x8001), ModesModeTag{Value: 1, Ansi: true}.Int())
	expectEqual(t, uint16(0x7FFF), ModesModeTag{Value: 0x7FFF}.Int())
	expectEqual(t, uint16(0xFFFF), ModesModeTag{Value: 0x7FFF, Ansi: true}.Int())
	tag := ModesModeTagFromInt(0x8004)
	expectEqual(t, uint16(4), tag.Value)
	expect(t, tag.Ansi, "t.ansi")
}

// Zig: test "term51: ModePacked: layout one bit per mode in declaration order"
func TestModes_term51_ModePacked_layout_one_bit_per_mode_in_declaration_order(t *testing.T) {
	// Zig: @bitSizeOf(ModePacked) == 40 and @sizeOf(ModePacked) == 8. Go:
	// one bit per entry in a 64-bit integer; the entry count stands in for
	// the bit size.
	expectEqual(t, 40, len(modesEntries))
	expectEqual(t, uintptr(8), unsafe.Sizeof(ModesModePacked{}))

	for i, row := range sm1_modeRows {
		// Zig: @bitOffsetOf(ModePacked, row.name) == i, and
		// @FieldType(ModePacked, row.name) == bool (a single bit in Go).
		expectEqualStrings(t, row.name, sm1_modesFieldNames[i])
		expectEqual(t, uint8(i), modesModePackedFieldBit([]uint8(row.name)))

		// Only that bit is set when the field is true and the rest are false.
		p := sm1_modesFromBits(0)
		sm1_modesFieldSet(&p, row.name, true)
		expectEqual(t, uint64(1)<<uint(i), sm1_modesBits(p))
	}
}

// Zig: test "term51: ModePacked: default values"
func TestModes_term51_ModePacked_default_values(t *testing.T) {
	p := ModesModePackedInit()
	expectEqual(t, sm1_modesDefaultBits(), sm1_modesBits(p))
	// Exactly six modes default to true.
	expectEqual(t, 6, bits.OnesCount64(sm1_modesBits(p)))
	n := 0
	for i, row := range sm1_modeRows {
		expectEqual(t, row.default_, sm1_modesField(&p, sm1_modesFieldNames[i]))
		if row.default_ {
			n += 1
		}
	}
	expectEqual(t, 6, n)
	expect(t, sm1_modesField(&p, "send_receive_mode"), "p.send_receive_mode")
	expect(t, sm1_modesField(&p, "wraparound"), "p.wraparound")
	expect(t, sm1_modesField(&p, "cursor_visible"), "p.cursor_visible")
	expect(t, sm1_modesField(&p, "mouse_alternate_scroll"), "p.mouse_alternate_scroll")
	expect(t, sm1_modesField(&p, "ignore_keypad_with_numlock"), "p.ignore_keypad_with_numlock")
	expect(t, sm1_modesField(&p, "alt_esc_prefix"), "p.alt_esc_prefix")
	expect(t, !sm1_modesField(&p, "cursor_keys"), "!p.cursor_keys")
	expect(t, !sm1_modesField(&p, "in_band_size_reports"), "!p.in_band_size_reports")
}

// Zig: test "term51: entryForMode: returns the entry for every mode"
func TestModes_term51_entryForMode_returns_the_entry_for_every_mode(t *testing.T) {
	for _, row := range sm1_modeRows {
		e := modesEntryForMode(row.mode)
		expectEqualStrings(t, row.name, string(e.name))
		expectEqual(t, int(row.value), e.value)
		expectEqual(t, row.ansi, e.ansi)
		expectEqual(t, row.default_, e.default_)
		expectEqual(t, false, e.disabled)
	}
}

// Zig: test "term51: modeFromInt: exhaustive over every u16 and both namespaces"
func TestModes_term51_modeFromInt_exhaustive_over_every_u16_and_both_namespaces(t *testing.T) {
	found := 0
	for v := uint32(0); v <= 0xFFFF; v++ {
		n := uint16(v)
		for _, ansi := range [...]bool{false, true} {
			got := ModesModeFromInt(n, ansi)
			want := sm1_modesReference(n, ansi)
			expectEqual(t, want, got)
			if got.Valid {
				found += 1
			}
		}
	}
	expectEqual(t, 40, found)
}

// Zig: test "term51: modeFromInt: every mode round trips and the other namespace is separate"
func TestModes_term51_modeFromInt_every_mode_round_trips_and_the_other_namespace_is_separate(t *testing.T) {
	for _, row := range sm1_modeRows {
		m := zigUnwrap(t, ModesModeFromInt(row.value, row.ansi))
		expectEqual(t, row.mode, m)
		// The returned tag carries the value and the ansi bit.
		tag := ModesModeTagFromInt(ModesModeTagBacking(m))
		expectEqual(t, row.value, tag.Value)
		expectEqual(t, row.ansi, tag.Ansi)
	}

	// Numbers that exist in both namespaces map to different modes.
	expectEqual(t, ModesModeInsert, zigUnwrap(t, ModesModeFromInt(4, true)))
	expectEqual(t, ModesModeSlowScroll, zigUnwrap(t, ModesModeFromInt(4, false)))
	expectEqual(t, ModesModeSendReceiveMode, zigUnwrap(t, ModesModeFromInt(12, true)))
	expectEqual(t, ModesModeCursorBlinking, zigUnwrap(t, ModesModeFromInt(12, false)))

	// Numbers that exist in only one namespace.
	expect(t, !ModesModeFromInt(2, false).Valid, "modeFromInt(2, false) == null")
	expectEqual(t, ModesModeDisableKeyboard, zigUnwrap(t, ModesModeFromInt(2, true)))
	expect(t, !ModesModeFromInt(20, false).Valid, "modeFromInt(20, false) == null")
	expectEqual(t, ModesModeLinefeed, zigUnwrap(t, ModesModeFromInt(20, true)))
	expect(t, !ModesModeFromInt(1, true).Valid, "modeFromInt(1, true) == null")
	expectEqual(t, ModesModeCursorKeys, zigUnwrap(t, ModesModeFromInt(1, false)))
	expect(t, !ModesModeFromInt(1049, true).Valid, "modeFromInt(1049, true) == null")
	expectEqual(t, ModesModeAltScreenSaveCursorClearEnter, zigUnwrap(t, ModesModeFromInt(1049, false)))
}

// Zig: test "term51: modeFromInt: values just outside the defined ones"
func TestModes_term51_modeFromInt_values_just_outside_the_defined_ones(t *testing.T) {
	// Below, between and above the DEC numbers.
	expect(t, !ModesModeFromInt(0, false).Valid, "modeFromInt(0, false) == null")
	expect(t, !ModesModeFromInt(0, true).Valid, "modeFromInt(0, true) == null")
	expect(t, !ModesModeFromInt(10, false).Valid, "modeFromInt(10, false) == null")
	expect(t, !ModesModeFromInt(2, false).Valid, "modeFromInt(2, false) == null")
	expect(t, !ModesModeFromInt(24, false).Valid, "modeFromInt(24, false) == null")
	expect(t, !ModesModeFromInt(26, false).Valid, "modeFromInt(26, false) == null")
	expect(t, !ModesModeFromInt(39, false).Valid, "modeFromInt(39, false) == null")
	expect(t, !ModesModeFromInt(41, false).Valid, "modeFromInt(41, false) == null")
	expect(t, !ModesModeFromInt(46, false).Valid, "modeFromInt(46, false) == null")
	expect(t, !ModesModeFromInt(65, false).Valid, "modeFromInt(65, false) == null")
	expect(t, !ModesModeFromInt(67, false).Valid, "modeFromInt(67, false) == null")
	expect(t, !ModesModeFromInt(68, false).Valid, "modeFromInt(68, false) == null")
	expect(t, !ModesModeFromInt(70, false).Valid, "modeFromInt(70, false) == null")
	expect(t, !ModesModeFromInt(999, false).Valid, "modeFromInt(999, false) == null")
	expect(t, !ModesModeFromInt(1001, false).Valid, "modeFromInt(1001, false) == null")
	expect(t, !ModesModeFromInt(1008, false).Valid, "modeFromInt(1008, false) == null")
	expect(t, !ModesModeFromInt(1014, false).Valid, "modeFromInt(1014, false) == null")
	expect(t, !ModesModeFromInt(1017, false).Valid, "modeFromInt(1017, false) == null")
	expect(t, !ModesModeFromInt(1034, false).Valid, "modeFromInt(1034, false) == null")
	expect(t, !ModesModeFromInt(1037, false).Valid, "modeFromInt(1037, false) == null")
	expect(t, !ModesModeFromInt(1038, false).Valid, "modeFromInt(1038, false) == null")
	expect(t, !ModesModeFromInt(1046, false).Valid, "modeFromInt(1046, false) == null")
	expect(t, !ModesModeFromInt(1050, false).Valid, "modeFromInt(1050, false) == null")
	expect(t, !ModesModeFromInt(2003, false).Valid, "modeFromInt(2003, false) == null")
	expect(t, !ModesModeFromInt(2005, false).Valid, "modeFromInt(2005, false) == null")
	expect(t, !ModesModeFromInt(2025, false).Valid, "modeFromInt(2025, false) == null")
	expect(t, !ModesModeFromInt(2028, false).Valid, "modeFromInt(2028, false) == null")
	expect(t, !ModesModeFromInt(2030, false).Valid, "modeFromInt(2030, false) == null")
	expect(t, !ModesModeFromInt(2032, false).Valid, "modeFromInt(2032, false) == null")
	expect(t, !ModesModeFromInt(2047, false).Valid, "modeFromInt(2047, false) == null")
	expect(t, !ModesModeFromInt(2049, false).Valid, "modeFromInt(2049, false) == null")
	// The first and last defined DEC numbers.
	expectEqual(t, ModesModeCursorKeys, zigUnwrap(t, ModesModeFromInt(1, false)))
	expectEqual(t, ModesModeInBandSizeReports, zigUnwrap(t, ModesModeFromInt(2048, false)))

	// Around the ANSI numbers.
	expect(t, !ModesModeFromInt(1, true).Valid, "modeFromInt(1, true) == null")
	expect(t, !ModesModeFromInt(3, true).Valid, "modeFromInt(3, true) == null")
	expect(t, !ModesModeFromInt(5, true).Valid, "modeFromInt(5, true) == null")
	expect(t, !ModesModeFromInt(11, true).Valid, "modeFromInt(11, true) == null")
	expect(t, !ModesModeFromInt(13, true).Valid, "modeFromInt(13, true) == null")
	expect(t, !ModesModeFromInt(19, true).Valid, "modeFromInt(19, true) == null")
	expect(t, !ModesModeFromInt(21, true).Valid, "modeFromInt(21, true) == null")

	// The top bit of the u16 is not a namespace selector for the input.
	expect(t, !ModesModeFromInt(0x8002, true).Valid, "modeFromInt(0x8002, true) == null")
	expect(t, !ModesModeFromInt(0x8004, false).Valid, "modeFromInt(0x8004, false) == null")
	expect(t, !ModesModeFromInt(32770, true).Valid, "modeFromInt(32770, true) == null")
	expect(t, !ModesModeFromInt(65535, true).Valid, "modeFromInt(65535, true) == null")
	expect(t, !ModesModeFromInt(65535, false).Valid, "modeFromInt(65535, false) == null")
	expect(t, !ModesModeFromInt(0x7FFF, false).Valid, "modeFromInt(0x7FFF, false) == null")
	expect(t, !ModesModeFromInt(0x8000, false).Valid, "modeFromInt(0x8000, false) == null")
}

// Zig: test "term51: modeFromInt: no entry is disabled"
func TestModes_term51_modeFromInt_no_entry_is_disabled(t *testing.T) {
	// The disabled branch of modeFromInt cannot be reached with the shipped
	// table: every entry is enabled, so every mode is reachable.
	for _, e := range modesEntries {
		expect(t, !e.disabled, "!e.disabled")
	}
	for _, row := range sm1_modeRows {
		expect(t, ModesModeFromInt(row.value, row.ansi).Valid, "modeFromInt(row.value, row.ansi) != null")
	}
}

// Zig: test "term51: ModeState: defaults for values, saved and default"
func TestModes_term51_ModeState_defaults_for_values_saved_and_default(t *testing.T) {
	s := ModesModeStateInit()
	expectEqual(t, sm1_modesDefaultBits(), sm1_modesBits(s.Values))
	expectEqual(t, sm1_modesDefaultBits(), sm1_modesBits(s.Saved))
	expectEqual(t, sm1_modesDefaultBits(), sm1_modesBits(s.Default))
	for _, row := range sm1_modeRows {
		expectEqual(t, row.default_, s.Get(row.mode))
	}
}

// Zig: test "term51: ModeState.get: reads every mode from the values field"
func TestModes_term51_ModeState_get_reads_every_mode_from_the_values_field(t *testing.T) {
	for i, row := range sm1_modeRows {
		s := ModesModeStateInit()
		// Flip exactly one bit directly in values.
		s.Values = sm1_modesFromBits(sm1_modesDefaultBits() ^ (uint64(1) << uint(i)))
		for _, other := range sm1_modeRows {
			var want bool
			if other.mode == row.mode {
				want = !other.default_
			} else {
				want = other.default_
			}
			expectEqual(t, want, s.Get(other.mode))
		}
	}
}

// Zig: test "term51: ModeState.set: every mode sets and clears only its own bit"
func TestModes_term51_ModeState_set_every_mode_sets_and_clears_only_its_own_bit(t *testing.T) {
	for i, row := range sm1_modeRows {
		bit := uint64(1) << uint(i)
		s := ModesModeStateInit()

		s.Set(row.mode, true)
		expect(t, s.Get(row.mode), "s.get(row.mode)")
		expectEqual(t, sm1_modesDefaultBits()|bit, sm1_modesBits(s.Values))

		s.Set(row.mode, true)
		expectEqual(t, sm1_modesDefaultBits()|bit, sm1_modesBits(s.Values))

		s.Set(row.mode, false)
		expect(t, !s.Get(row.mode), "!s.get(row.mode)")
		expectEqual(t, sm1_modesDefaultBits()&^bit, sm1_modesBits(s.Values))

		s.Set(row.mode, false)
		expectEqual(t, sm1_modesDefaultBits()&^bit, sm1_modesBits(s.Values))

		// set does not touch saved or default.
		expectEqual(t, sm1_modesDefaultBits(), sm1_modesBits(s.Saved))
		expectEqual(t, sm1_modesDefaultBits(), sm1_modesBits(s.Default))
	}
}

// Zig: test "term51: ModeState.set: all modes can be set and cleared together"
func TestModes_term51_ModeState_set_all_modes_can_be_set_and_cleared_together(t *testing.T) {
	s := ModesModeStateInit()
	for _, row := range sm1_modeRows {
		s.Set(row.mode, true)
	}
	expectEqual(t, uint64(0xFF_FFFF_FFFF), sm1_modesBits(s.Values))
	for _, row := range sm1_modeRows {
		expect(t, s.Get(row.mode), "s.get(row.mode)")
	}
	for _, row := range sm1_modeRows {
		s.Set(row.mode, false)
	}
	expectEqual(t, uint64(0), sm1_modesBits(s.Values))
	for _, row := range sm1_modeRows {
		expect(t, !s.Get(row.mode), "!s.get(row.mode)")
	}
}

// Zig: test "term51: ModeState.save: copies one mode into saved"
func TestModes_term51_ModeState_save_copies_one_mode_into_saved(t *testing.T) {
	for i, row := range sm1_modeRows {
		bit := uint64(1) << uint(i)
		s := ModesModeStateInit()

		// Saving the default leaves saved equal to the default pattern.
		s.Save(row.mode)
		expectEqual(t, sm1_modesDefaultBits(), sm1_modesBits(s.Saved))

		// Saving the flipped value changes only this mode's saved bit.
		s.Set(row.mode, !row.default_)
		s.Save(row.mode)
		expectEqual(t, sm1_modesDefaultBits()^bit, sm1_modesBits(s.Saved))
		// values are unchanged by save.
		expectEqual(t, sm1_modesDefaultBits()^bit, sm1_modesBits(s.Values))

		// Saving again overwrites the single saved slot.
		s.Set(row.mode, row.default_)
		s.Save(row.mode)
		expectEqual(t, sm1_modesDefaultBits(), sm1_modesBits(s.Saved))
	}
}

// Zig: test "term51: ModeState.restore: returns the restored value and leaves other modes alone"
func TestModes_term51_ModeState_restore_returns_the_restored_value_and_leaves_other_modes_alone(t *testing.T) {
	for i, row := range sm1_modeRows {
		bit := uint64(1) << uint(i)
		for _, saved_value := range [...]bool{false, true} {
			s := ModesModeStateInit()
			s.Set(row.mode, saved_value)
			s.Save(row.mode)
			s.Set(row.mode, !saved_value)
			expectEqual(t, !saved_value, s.Get(row.mode))

			expectEqual(t, saved_value, s.Restore(row.mode))
			expectEqual(t, saved_value, s.Get(row.mode))

			// Only this mode's bit may differ from the default pattern.
			var want uint64
			if saved_value {
				want = sm1_modesDefaultBits() | bit
			} else {
				want = sm1_modesDefaultBits() &^ bit
			}
			expectEqual(t, want, sm1_modesBits(s.Values))

			// The saved slot is not consumed: restoring again gives the same.
			s.Set(row.mode, !saved_value)
			expectEqual(t, saved_value, s.Restore(row.mode))
			expectEqual(t, saved_value, s.Get(row.mode))
		}
	}
}

// Zig: test "term51: ModeState.restore: without a prior save restores the default value"
func TestModes_term51_ModeState_restore_without_a_prior_save_restores_the_default_value(t *testing.T) {
	// `saved` starts as the default pattern, so an unmatched restore
	// resets the mode to its default.
	for _, row := range sm1_modeRows {
		s := ModesModeStateInit()
		s.Set(row.mode, !row.default_)
		expectEqual(t, row.default_, s.Restore(row.mode))
		expectEqual(t, row.default_, s.Get(row.mode))
	}
}

// Zig: test "term51: ModeState.restore: modes are saved independently"
func TestModes_term51_ModeState_restore_modes_are_saved_independently(t *testing.T) {
	s := ModesModeStateInit()
	s.Set(ModesModeCursorKeys, true)
	s.Set(ModesModeOrigin, true)
	s.Save(ModesModeCursorKeys)
	s.Set(ModesModeCursorKeys, false)
	s.Set(ModesModeOrigin, false)

	// origin was never saved so it restores to its default (false), not true.
	expect(t, s.Restore(ModesModeCursorKeys), "s.restore(.cursor_keys)")
	expect(t, !s.Restore(ModesModeOrigin), "!s.restore(.origin)")
	expect(t, s.Get(ModesModeCursorKeys), "s.get(.cursor_keys)")
	expect(t, !s.Get(ModesModeOrigin), "!s.get(.origin)")

	// Saved defaults of default-true modes come back as true.
	s.Set(ModesModeWraparound, false)
	expect(t, s.Restore(ModesModeWraparound), "s.restore(.wraparound)")

	// ANSI and DEC modes sharing a number do not alias.
	s.Set(ModesModeInsert, true)
	s.Save(ModesModeInsert)
	s.Set(ModesModeInsert, false)
	s.Set(ModesModeSlowScroll, true)
	expect(t, s.Restore(ModesModeInsert), "s.restore(.insert)")
	expect(t, s.Get(ModesModeSlowScroll), "s.get(.slow_scroll)")
}

// Zig: test "term51: ModeState.reset: restores defaults and clears saved for every mode"
func TestModes_term51_ModeState_reset_restores_defaults_and_clears_saved_for_every_mode(t *testing.T) {
	s := ModesModeStateInit()
	for _, row := range sm1_modeRows {
		s.Set(row.mode, !row.default_)
		s.Save(row.mode)
	}
	expectEqual(t, ^sm1_modesDefaultBits()&0xFF_FFFF_FFFF, sm1_modesBits(s.Values))
	expectEqual(t, ^sm1_modesDefaultBits()&0xFF_FFFF_FFFF, sm1_modesBits(s.Saved))

	s.Reset()
	expectEqual(t, sm1_modesDefaultBits(), sm1_modesBits(s.Values))
	expectEqual(t, sm1_modesDefaultBits(), sm1_modesBits(s.Saved))
	expectEqual(t, sm1_modesDefaultBits(), sm1_modesBits(s.Default))
	for _, row := range sm1_modeRows {
		expectEqual(t, row.default_, s.Get(row.mode))
		// Saved was cleared: restore yields the default again.
		expectEqual(t, row.default_, s.Restore(row.mode))
	}
}

// Zig: test "term51: ModeState.reset: uses the configured default values"
func TestModes_term51_ModeState_reset_uses_the_configured_default_values(t *testing.T) {
	s := ModesModeStateInit()
	// Custom defaults: every mode on.
	s.Default = sm1_modesFromBits(0xFF_FFFF_FFFF)
	s.Set(ModesModeCursorKeys, false)
	s.Save(ModesModeCursorKeys)

	s.Reset()
	expectEqual(t, uint64(0xFF_FFFF_FFFF), sm1_modesBits(s.Values))
	expect(t, s.Get(ModesModeCursorKeys), "s.get(.cursor_keys)")
	// saved is reset to the type's default pattern, not to `default`.
	expectEqual(t, sm1_modesDefaultBits(), sm1_modesBits(s.Saved))
	expect(t, !s.Restore(ModesModeCursorKeys), "!s.restore(.cursor_keys)")
	expect(t, s.Restore(ModesModeWraparound), "s.restore(.wraparound)")

	// Custom defaults: every mode off.
	s.Default = sm1_modesFromBits(0)
	s.Reset()
	expectEqual(t, uint64(0), sm1_modesBits(s.Values))
	for _, row := range sm1_modeRows {
		expect(t, !s.Get(row.mode), "!s.get(row.mode)")
	}

	// reset is idempotent.
	s.Reset()
	expectEqual(t, uint64(0), sm1_modesBits(s.Values))
}
