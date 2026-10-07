// Tests translated one for one from src/input/function_keys.zig (Ghostty
// v1.3.1 plus the term51 additional tests). Each Go test carries the exact
// Zig title in a "// Zig: test" comment. Helpers are prefixed kp.
//
// Translation notes:
//   - Zig's `.{ .sequence = "x" }` is an Entry with mods_empty_is_any TRUE
//     (the Zig default, not the Go zero value): every Entry literal here goes
//     through kpFKEntry, which sets it. Only kpDefault's entry has it false,
//     and the tests compare it through kpFKExpectEntry.
//   - `[]const u8` sequences and comptime string parameters are written as
//     Go strings in the helpers and converted with []uint8(...) at the call.
//   - `key.Mods.int()` is Int(), `equal` is Equal().
//   - `keys.get(k)` is InputFunctionKeysKeys.Get(k); `keys.values` is
//     InputFunctionKeysKeys.values; `std.enums.values(key.Key)` is
//     InputKeyKeyValues().
//   - `@typeInfo(E).@"enum".fields.len` and `fields[i].name` are the
//     inputFunctionKeys<E>Count constants and the String methods of the
//     values (the wireframe declares both for this purpose).
//   - `term51AEFreeze` (comptime copy of a comptime-built list) has no Go
//     counterpart: the helper functions return a fresh slice on each call.
//   - `std.fmt.allocPrint(alloc, "...{d}...", .{n})` is fmt.Sprintf.
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt.

package terminal

import (
	"fmt"
	"strconv"
	"testing"
)

// kpFKEntry is `.{ .mods = mods, .modify_other_keys = mok, .sequence = seq }`
// with every other field at its Zig default (mods_empty_is_any TRUE, cursor
// and keypad .any).
func kpFKEntry(mods InputKeyMods, mok InputFunctionKeysModifyKeys, seq string) InputFunctionKeysEntry {
	return InputFunctionKeysEntry{
		mods:              mods,
		mods_empty_is_any: true,
		modify_other_keys: mok,
		sequence:          []uint8(seq),
	}
}

// kpFKMods is term51AEMods.
func kpFKMods(shift bool, ctrl bool, alt bool, super bool) InputKeyMods {
	return InputKeyMods{Shift: shift, Ctrl: ctrl, Alt: alt, Super: super}
}

// kpFKBits is the xterm bit mask of the four standard modifiers (shift=1,
// alt=2, ctrl=4, super=8); the xterm modifier code is bits + 1.
func kpFKBits(m InputKeyMods) int {
	b := func(v bool) int {
		if v {
			return 1
		}
		return 0
	}
	return b(m.Shift) + 2*b(m.Alt) + 4*b(m.Ctrl) + 8*b(m.Super)
}

// kpFKExpectEntry is term51AEExpectEntry.
func kpFKExpectEntry(
	tt *testing.T,
	e InputFunctionKeysEntry,
	mods InputKeyMods,
	any bool,
	cursor InputFunctionKeysCursorMode,
	keypad InputFunctionKeysKeypadMode,
	mok InputFunctionKeysModifyKeys,
	seq string,
) {
	tt.Helper()
	expectEqual(tt, mods.Int(), e.mods.Int())
	expectEqual(tt, any, e.mods_empty_is_any)
	expectEqual(tt, cursor, e.cursor)
	expectEqual(tt, keypad, e.keypad)
	expectEqual(tt, mok, e.modify_other_keys)
	expectEqualStrings(tt, seq, string(e.sequence))
}

// kpFKExpectEntries is term51AEExpectEntries.
func kpFKExpectEntries(tt *testing.T, expected []InputFunctionKeysEntry, actual []InputFunctionKeysEntry) {
	tt.Helper()
	expectEqual(tt, len(expected), len(actual))
	for i, ex := range expected {
		kpFKExpectEntry(
			tt,
			actual[i],
			ex.mods,
			ex.mods_empty_is_any,
			ex.cursor,
			ex.keypad,
			ex.modify_other_keys,
			string(ex.sequence),
		)
	}
}

// kpFKModTable is term51AEModTable: the 15 modifier combinations written out
// as (shift, ctrl, alt, super), in table order; the xterm modifier code of
// entry i is i + 2.
var kpFKModTable = [15][4]bool{
	{true, false, false, false},
	{false, false, true, false},
	{true, false, true, false},
	{false, true, false, false},
	{true, true, false, false},
	{false, true, true, false},
	{true, true, true, false},
	{false, false, false, true},
	{true, false, false, true},
	{false, false, true, true},
	{true, false, true, true},
	{false, true, false, true},
	{true, true, false, true},
	{false, true, true, true},
	{true, true, true, true},
}

// kpFKExpectPc is term51AEExpectPc: asserts that the first 15 entries are the
// pcStyle block for the given prefix/suffix: entry i has modifiers
// kpFKModTable[i], the sequence prefix ++ (i + 2) ++ suffix, and the given
// keypad mode.
func kpFKExpectPc(
	tt *testing.T,
	entries []InputFunctionKeysEntry,
	prefix string,
	suffix string,
	keypad InputFunctionKeysKeypadMode,
) {
	tt.Helper()
	expect(tt, len(entries) >= 15, "entries.len >= 15")
	for i, m := range kpFKModTable {
		seq := prefix + strconv.Itoa(i+2) + suffix
		kpFKExpectEntry(
			tt,
			entries[i],
			kpFKMods(m[0], m[1], m[2], m[3]),
			true,
			InputFunctionKeysCursorModeAny,
			keypad,
			InputFunctionKeysModifyKeysAny,
			seq,
		)
	}
}

// kpFKByte is `&.{c}` for a single byte c.
func kpFKByte(c uint8) string { return string([]uint8{c}) }

// kpFKBackspace is term51AEBackspace.
var kpFKBackspace = []InputFunctionKeysEntry{
	kpFKEntry(InputKeyMods{Shift: true}, InputFunctionKeysModifyKeysSet, "\x7f"),
	kpFKEntry(InputKeyMods{Alt: true}, InputFunctionKeysModifyKeysSet, "\x1b\x7f"),
	kpFKEntry(InputKeyMods{Alt: true, Shift: true}, InputFunctionKeysModifyKeysSet, "\x1b\x7f"),
	kpFKEntry(InputKeyMods{Ctrl: true, Shift: true}, InputFunctionKeysModifyKeysSet, "\x08"),
	kpFKEntry(InputKeyMods{Alt: true, Ctrl: true}, InputFunctionKeysModifyKeysSet, "\x1b\x08"),
	kpFKEntry(InputKeyMods{Super: true}, InputFunctionKeysModifyKeysSet, "\x7f"),
	kpFKEntry(InputKeyMods{Super: true, Shift: true}, InputFunctionKeysModifyKeysSet, "\x7f"),
	kpFKEntry(InputKeyMods{Alt: true, Super: true}, InputFunctionKeysModifyKeysSet, "\x1b\x7f"),
	kpFKEntry(InputKeyMods{Alt: true, Super: true, Shift: true}, InputFunctionKeysModifyKeysSet, "\x1b\x7f"),
	kpFKEntry(InputKeyMods{Super: true, Ctrl: true}, InputFunctionKeysModifyKeysSet, "\x08"),
	kpFKEntry(InputKeyMods{Super: true, Ctrl: true, Shift: true}, InputFunctionKeysModifyKeysSet, "\x08"),
	kpFKEntry(InputKeyMods{Alt: true, Super: true, Ctrl: true}, InputFunctionKeysModifyKeysSet, "\x1b\x08"),
	kpFKEntry(InputKeyMods{Alt: true, Super: true, Ctrl: true, Shift: true}, InputFunctionKeysModifyKeysSet, "\x1b\x08"),
	kpFKEntry(InputKeyMods{Shift: true}, InputFunctionKeysModifyKeysSetOther, "\x1b[27;2;127~"),
	kpFKEntry(InputKeyMods{Alt: true}, InputFunctionKeysModifyKeysSetOther, "\x1b[27;3;127~"),
	kpFKEntry(InputKeyMods{Alt: true, Shift: true}, InputFunctionKeysModifyKeysSetOther, "\x1b[27;4;127~"),
	kpFKEntry(InputKeyMods{Ctrl: true, Shift: true}, InputFunctionKeysModifyKeysSetOther, "\x1b[27;6;127~"),
	kpFKEntry(InputKeyMods{Alt: true, Ctrl: true}, InputFunctionKeysModifyKeysSetOther, "\x1b[27;7;127~"),
	kpFKEntry(InputKeyMods{Alt: true, Shift: true, Ctrl: true}, InputFunctionKeysModifyKeysSetOther, "\x1b[27;8;127~"),
	kpFKEntry(InputKeyMods{Super: true}, InputFunctionKeysModifyKeysSetOther, "\x1b[27;9;127~"),
	kpFKEntry(InputKeyMods{Super: true, Shift: true}, InputFunctionKeysModifyKeysSetOther, "\x1b[27;10;127~"),
	kpFKEntry(InputKeyMods{Alt: true, Super: true}, InputFunctionKeysModifyKeysSetOther, "\x1b[27;11;127~"),
	kpFKEntry(InputKeyMods{Alt: true, Super: true, Shift: true}, InputFunctionKeysModifyKeysSetOther, "\x1b[27;12;127~"),
	kpFKEntry(InputKeyMods{Super: true, Ctrl: true}, InputFunctionKeysModifyKeysSetOther, "\x1b[27;13;127~"),
	kpFKEntry(InputKeyMods{Super: true, Ctrl: true, Shift: true}, InputFunctionKeysModifyKeysSetOther, "\x1b[27;14;127~"),
	kpFKEntry(InputKeyMods{Alt: true, Super: true, Ctrl: true}, InputFunctionKeysModifyKeysSetOther, "\x1b[27;15;127~"),
	kpFKEntry(InputKeyMods{Alt: true, Super: true, Ctrl: true, Shift: true}, InputFunctionKeysModifyKeysSetOther, "\x1b[27;16;127~"),
	kpFKEntry(InputKeyMods{Ctrl: true}, InputFunctionKeysModifyKeysAny, "\x08"),
	kpFKEntry(InputKeyMods{}, InputFunctionKeysModifyKeysAny, "\x7f"),
}

// kpFKTab is term51AETab.
var kpFKTab = []InputFunctionKeysEntry{
	kpFKEntry(InputKeyMods{Shift: true}, InputFunctionKeysModifyKeysSet, "\x1b[Z"),
	kpFKEntry(InputKeyMods{Alt: true}, InputFunctionKeysModifyKeysSet, "\x1b\t"),
	kpFKEntry(InputKeyMods{Shift: true}, InputFunctionKeysModifyKeysSetOther, "\x1b[27;2;9~"),
	kpFKEntry(InputKeyMods{Alt: true}, InputFunctionKeysModifyKeysSetOther, "\x1b[27;3;9~"),
	kpFKEntry(InputKeyMods{Alt: true, Shift: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;4;9~"),
	kpFKEntry(InputKeyMods{Ctrl: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;5;9~"),
	kpFKEntry(InputKeyMods{Ctrl: true, Shift: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;6;9~"),
	kpFKEntry(InputKeyMods{Alt: true, Ctrl: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;7;9~"),
	kpFKEntry(InputKeyMods{Alt: true, Ctrl: true, Shift: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;8;9~"),
	kpFKEntry(InputKeyMods{Super: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;9;9~"),
	kpFKEntry(InputKeyMods{Super: true, Shift: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;10;9~"),
	kpFKEntry(InputKeyMods{Alt: true, Super: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;11;9~"),
	kpFKEntry(InputKeyMods{Alt: true, Super: true, Shift: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;12;9~"),
	kpFKEntry(InputKeyMods{Super: true, Ctrl: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;13;9~"),
	kpFKEntry(InputKeyMods{Super: true, Ctrl: true, Shift: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;14;9~"),
	kpFKEntry(InputKeyMods{Alt: true, Super: true, Ctrl: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;15;9~"),
	kpFKEntry(InputKeyMods{Alt: true, Super: true, Ctrl: true, Shift: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;16;9~"),
	kpFKEntry(InputKeyMods{}, InputFunctionKeysModifyKeysAny, "\t"),
}

// kpFKEnter is term51AEEnter.
var kpFKEnter = []InputFunctionKeysEntry{
	kpFKEntry(InputKeyMods{Shift: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;2;13~"),
	kpFKEntry(InputKeyMods{Alt: true}, InputFunctionKeysModifyKeysSet, "\x1b\r"),
	kpFKEntry(InputKeyMods{Alt: true}, InputFunctionKeysModifyKeysSetOther, "\x1b[27;3;13~"),
	kpFKEntry(InputKeyMods{Alt: true, Shift: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;4;13~"),
	kpFKEntry(InputKeyMods{Ctrl: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;5;13~"),
	kpFKEntry(InputKeyMods{Ctrl: true, Shift: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;6;13~"),
	kpFKEntry(InputKeyMods{Alt: true, Ctrl: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;7;13~"),
	kpFKEntry(InputKeyMods{Alt: true, Ctrl: true, Shift: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;8;13~"),
	kpFKEntry(InputKeyMods{Super: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;9;13~"),
	kpFKEntry(InputKeyMods{Super: true, Shift: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;10;13~"),
	kpFKEntry(InputKeyMods{Alt: true, Super: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;11;13~"),
	kpFKEntry(InputKeyMods{Alt: true, Super: true, Shift: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;12;13~"),
	kpFKEntry(InputKeyMods{Super: true, Ctrl: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;13;13~"),
	kpFKEntry(InputKeyMods{Super: true, Ctrl: true, Shift: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;14;13~"),
	kpFKEntry(InputKeyMods{Alt: true, Super: true, Ctrl: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;15;13~"),
	kpFKEntry(InputKeyMods{Alt: true, Super: true, Ctrl: true, Shift: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;16;13~"),
	kpFKEntry(InputKeyMods{}, InputFunctionKeysModifyKeysAny, "\r"),
}

// kpFKEscape is term51AEEscape.
var kpFKEscape = []InputFunctionKeysEntry{
	kpFKEntry(InputKeyMods{Shift: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;2;27~"),
	kpFKEntry(InputKeyMods{Alt: true}, InputFunctionKeysModifyKeysAny, "\x1b\x1b"),
	kpFKEntry(InputKeyMods{Alt: true, Shift: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;4;27~"),
	kpFKEntry(InputKeyMods{Ctrl: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;5;27~"),
	kpFKEntry(InputKeyMods{Ctrl: true, Shift: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;6;27~"),
	kpFKEntry(InputKeyMods{Alt: true, Ctrl: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;7;27~"),
	kpFKEntry(InputKeyMods{Alt: true, Ctrl: true, Shift: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;8;27~"),
	kpFKEntry(InputKeyMods{Super: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;9;27~"),
	kpFKEntry(InputKeyMods{Super: true, Shift: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;10;27~"),
	kpFKEntry(InputKeyMods{Alt: true, Super: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;11;27~"),
	kpFKEntry(InputKeyMods{Alt: true, Super: true, Shift: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;12;27~"),
	kpFKEntry(InputKeyMods{Super: true, Ctrl: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;13;27~"),
	kpFKEntry(InputKeyMods{Super: true, Ctrl: true, Shift: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;14;27~"),
	kpFKEntry(InputKeyMods{Alt: true, Super: true, Ctrl: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;15;27~"),
	kpFKEntry(InputKeyMods{Alt: true, Super: true, Ctrl: true, Shift: true}, InputFunctionKeysModifyKeysAny, "\x1b[27;16;27~"),
	kpFKEntry(InputKeyMods{}, InputFunctionKeysModifyKeysAny, "\x1b"),
}

// Zig: test "term51: Entry: field defaults and enum values"
func TestKpFunctionKeys_Entry_field_defaults_and_enum_values(t *testing.T) {
	e := InputFunctionKeysEntry{mods_empty_is_any: true, sequence: []uint8("x")}
	kpFKExpectEntry(t, e, InputKeyMods{}, true, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysAny, "x")
	expectEqual(t, uint16(0), e.mods.Int())

	expectEqual(t, 3, inputFunctionKeysCursorModeCount)
	expectEqualStrings(t, "any", InputFunctionKeysCursorMode(0).String())
	expectEqualStrings(t, "normal", InputFunctionKeysCursorMode(1).String())
	expectEqualStrings(t, "application", InputFunctionKeysCursorMode(2).String())
	expectEqual(t, 3, inputFunctionKeysKeypadModeCount)
	expectEqualStrings(t, "any", InputFunctionKeysKeypadMode(0).String())
	expectEqualStrings(t, "normal", InputFunctionKeysKeypadMode(1).String())
	expectEqualStrings(t, "application", InputFunctionKeysKeypadMode(2).String())
	expectEqual(t, 3, inputFunctionKeysModifyKeysCount)
	expectEqualStrings(t, "any", InputFunctionKeysModifyKeys(0).String())
	expectEqualStrings(t, "set", InputFunctionKeysModifyKeys(1).String())
	expectEqualStrings(t, "set_other", InputFunctionKeysModifyKeys(2).String())
}

// Zig: test "term51: modifiers: every entry of the modifier table"
func TestKpFunctionKeys_modifiers_every_entry_of_the_modifier_table(t *testing.T) {
	expectEqual(t, 15, len(InputFunctionKeysModifiers))
	for i, m := range kpFKModTable {
		actual := InputFunctionKeysModifiers[i]
		expectEqual(t, kpFKMods(m[0], m[1], m[2], m[3]).Int(), actual.Int())
		// Only the four standard modifier bits may be set (no locks/sides).
		expect(t, actual.Int()&^uint16(0b1111) == 0, "actual.int() & ~@as(u16, 0b1111) == 0")
	}

	// The xterm modifier code is 1 + the bitmask (shift=1, alt=2, ctrl=4,
	// super=8), and the table index is code - 2.
	for i, m := range InputFunctionKeysModifiers {
		code := i + 2
		expectEqual(t, code, kpFKBits(m)+1)
	}

	// Spot check the literal table.
	expect(t, InputFunctionKeysModifiers[0].Equal(InputKeyMods{Shift: true}), "modifiers[0].equal(.{ .shift = true })")
	expect(t, InputFunctionKeysModifiers[1].Equal(InputKeyMods{Alt: true}), "modifiers[1].equal(.{ .alt = true })")
	expect(t, InputFunctionKeysModifiers[3].Equal(InputKeyMods{Ctrl: true}), "modifiers[3].equal(.{ .ctrl = true })")
	expect(t, InputFunctionKeysModifiers[7].Equal(InputKeyMods{Super: true}), "modifiers[7].equal(.{ .super = true })")
	expect(t, InputFunctionKeysModifiers[14].Equal(InputKeyMods{Shift: true, Alt: true, Ctrl: true, Super: true}), "modifiers[14].equal(.{ .shift = true, .alt = true, .ctrl = true, .super = true })")
}

// Zig: test "term51: keys: cursor keys (arrows, home, end, keypad cursor keys)"
func TestKpFunctionKeys_keys_cursor_keys_arrows_home_end_keypad_cursor_keys(t *testing.T) {
	cases := [...]struct {
		x0 InputKeyKey
		x1 uint8
	}{
		{InputKeyKeyArrowUp, 'A'},
		{InputKeyKeyArrowDown, 'B'},
		{InputKeyKeyArrowRight, 'C'},
		{InputKeyKeyArrowLeft, 'D'},
		{InputKeyKeyHome, 'H'},
		{InputKeyKeyEnd, 'F'},
		{InputKeyKeyNumpadUp, 'A'},
		{InputKeyKeyNumpadDown, 'B'},
		{InputKeyKeyNumpadRight, 'C'},
		{InputKeyKeyNumpadLeft, 'D'},
		{InputKeyKeyNumpadBegin, 'E'},
		{InputKeyKeyNumpadHome, 'H'},
		{InputKeyKeyNumpadEnd, 'F'},
	}
	for _, c := range cases {
		entries := InputFunctionKeysKeys.Get(c.x0)
		expectEqual(t, 17, len(entries))
		kpFKExpectPc(t, entries, "\x1b[1;", kpFKByte(c.x1), InputFunctionKeysKeypadModeAny)
		kpFKExpectEntry(t, entries[15], InputKeyMods{}, true, InputFunctionKeysCursorModeNormal, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysAny, string([]uint8{0x1b, '[', c.x1}))
		kpFKExpectEntry(t, entries[16], InputKeyMods{}, true, InputFunctionKeysCursorModeApplication, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysAny, string([]uint8{0x1b, 'O', c.x1}))
	}

	// Explicit literal spot checks.
	expectEqualStrings(t, "\x1b[1;2A", string(InputFunctionKeysKeys.Get(InputKeyKeyArrowUp)[0].sequence))
	expectEqualStrings(t, "\x1b[1;16A", string(InputFunctionKeysKeys.Get(InputKeyKeyArrowUp)[14].sequence))
	expectEqualStrings(t, "\x1b[A", string(InputFunctionKeysKeys.Get(InputKeyKeyArrowUp)[15].sequence))
	expectEqualStrings(t, "\x1bOA", string(InputFunctionKeysKeys.Get(InputKeyKeyArrowUp)[16].sequence))
	expectEqualStrings(t, "\x1b[1;5E", string(InputFunctionKeysKeys.Get(InputKeyKeyNumpadBegin)[3].sequence))
	expectEqualStrings(t, "\x1b[E", string(InputFunctionKeysKeys.Get(InputKeyKeyNumpadBegin)[15].sequence))
	expectEqualStrings(t, "\x1bOE", string(InputFunctionKeysKeys.Get(InputKeyKeyNumpadBegin)[16].sequence))
}

// Zig: test "term51: keys: editing keys (insert, delete, page up/down and keypad variants)"
func TestKpFunctionKeys_keys_editing_keys_insert_delete_page_up_down_and_keypad_variants(t *testing.T) {
	cases := [...]struct {
		x0 InputKeyKey
		x1 string
		x2 string
	}{
		{InputKeyKeyInsert, "\x1b[2;", "\x1B[2~"},
		{InputKeyKeyDelete, "\x1b[3;", "\x1B[3~"},
		{InputKeyKeyPageUp, "\x1b[5;", "\x1B[5~"},
		{InputKeyKeyPageDown, "\x1b[6;", "\x1B[6~"},
		{InputKeyKeyNumpadInsert, "\x1b[2;", "\x1B[2~"},
		{InputKeyKeyNumpadDelete, "\x1b[3;", "\x1B[3~"},
		{InputKeyKeyNumpadPageUp, "\x1b[5;", "\x1B[5~"},
		{InputKeyKeyNumpadPageDown, "\x1b[6;", "\x1B[6~"},
	}
	for _, c := range cases {
		entries := InputFunctionKeysKeys.Get(c.x0)
		expectEqual(t, 16, len(entries))
		kpFKExpectPc(t, entries, c.x1, "~", InputFunctionKeysKeypadModeAny)
		kpFKExpectEntry(t, entries[15], InputKeyMods{}, true, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysAny, c.x2)
	}
	expectEqualStrings(t, "\x1b[2;2~", string(InputFunctionKeysKeys.Get(InputKeyKeyInsert)[0].sequence))
	expectEqualStrings(t, "\x1b[6;16~", string(InputFunctionKeysKeys.Get(InputKeyKeyPageDown)[14].sequence))
}

// Zig: test "term51: keys: function keys F1 to F12"
func TestKpFunctionKeys_keys_function_keys_F1_to_F12(t *testing.T) {
	cases := [...]struct {
		x0 InputKeyKey
		x1 string
		x2 string
		x3 string
	}{
		{InputKeyKeyF1, "\x1b[1;", "P", "\x1BOP"},
		{InputKeyKeyF2, "\x1b[1;", "Q", "\x1BOQ"},
		{InputKeyKeyF3, "\x1b[13;", "~", "\x1BOR"},
		{InputKeyKeyF4, "\x1b[1;", "S", "\x1BOS"},
		{InputKeyKeyF5, "\x1b[15;", "~", "\x1B[15~"},
		{InputKeyKeyF6, "\x1b[17;", "~", "\x1B[17~"},
		{InputKeyKeyF7, "\x1b[18;", "~", "\x1B[18~"},
		{InputKeyKeyF8, "\x1b[19;", "~", "\x1B[19~"},
		{InputKeyKeyF9, "\x1b[20;", "~", "\x1B[20~"},
		{InputKeyKeyF10, "\x1b[21;", "~", "\x1B[21~"},
		{InputKeyKeyF11, "\x1b[23;", "~", "\x1B[23~"},
		{InputKeyKeyF12, "\x1b[24;", "~", "\x1B[24~"},
	}
	for _, c := range cases {
		entries := InputFunctionKeysKeys.Get(c.x0)
		expectEqual(t, 16, len(entries))
		kpFKExpectPc(t, entries, c.x1, c.x2, InputFunctionKeysKeypadModeAny)
		kpFKExpectEntry(t, entries[15], InputKeyMods{}, true, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysAny, c.x3)
	}
	expectEqualStrings(t, "\x1b[1;2P", string(InputFunctionKeysKeys.Get(InputKeyKeyF1)[0].sequence))
	expectEqualStrings(t, "\x1b[13;16~", string(InputFunctionKeysKeys.Get(InputKeyKeyF3)[14].sequence))
	expectEqualStrings(t, "\x1BOR", string(InputFunctionKeysKeys.Get(InputKeyKeyF3)[15].sequence))
}

// Zig: test "term51: keys: F13 to F25 have no entries"
func TestKpFunctionKeys_keys_F13_to_F25_have_no_entries(t *testing.T) {
	// The source only defines F1-F12 ("todo: f13-f35"); the later function
	// keys fall through to the empty default.
	later := [...]InputKeyKey{
		InputKeyKeyF13, InputKeyKeyF14, InputKeyKeyF15, InputKeyKeyF16, InputKeyKeyF17, InputKeyKeyF18, InputKeyKeyF19,
		InputKeyKeyF20, InputKeyKeyF21, InputKeyKeyF22, InputKeyKeyF23, InputKeyKeyF24, InputKeyKeyF25,
	}
	for _, k := range later {
		expectEqual(t, 0, len(InputFunctionKeysKeys.Get(k)))
	}
}

// Zig: test "term51: keys: keypad keys"
func TestKpFunctionKeys_keys_keypad_keys(t *testing.T) {
	cases := [...]struct {
		x0 InputKeyKey
		x1 uint8
	}{
		{InputKeyKeyNumpad0, 'p'},
		{InputKeyKeyNumpad1, 'q'},
		{InputKeyKeyNumpad2, 'r'},
		{InputKeyKeyNumpad3, 's'},
		{InputKeyKeyNumpad4, 't'},
		{InputKeyKeyNumpad5, 'u'},
		{InputKeyKeyNumpad6, 'v'},
		{InputKeyKeyNumpad7, 'w'},
		{InputKeyKeyNumpad8, 'x'},
		{InputKeyKeyNumpad9, 'y'},
		{InputKeyKeyNumpadDecimal, 'n'},
		{InputKeyKeyNumpadDivide, 'o'},
		{InputKeyKeyNumpadMultiply, 'j'},
		{InputKeyKeyNumpadSubtract, 'm'},
		{InputKeyKeyNumpadAdd, 'k'},
		{InputKeyKeyNumpadEnter, 'M'},
	}
	for _, c := range cases {
		entries := InputFunctionKeysKeys.Get(c.x0)
		want_len := 16
		if c.x0 == InputKeyKeyNumpadEnter {
			want_len = 17
		}
		expectEqual(t, want_len, len(entries))

		// Entry 0: application keypad with no modifiers at all.
		kpFKExpectEntry(t, entries[0], InputKeyMods{}, false, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeApplication, InputFunctionKeysModifyKeysAny, string([]uint8{0x1b, 'O', c.x1}))
		// Entries 1..15: modifier variants, all requiring application keypad.
		kpFKExpectPc(t, entries[1:], "\x1bO", kpFKByte(c.x1), InputFunctionKeysKeypadModeApplication)
	}

	// numpad_enter additionally has the plain "\r" fallback.
	kpFKExpectEntry(t, InputFunctionKeysKeys.Get(InputKeyKeyNumpadEnter)[16], InputKeyMods{}, true, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysAny, "\r")

	// Literal spot checks.
	expectEqualStrings(t, "\x1bOp", string(InputFunctionKeysKeys.Get(InputKeyKeyNumpad0)[0].sequence))
	expectEqualStrings(t, "\x1bO2p", string(InputFunctionKeysKeys.Get(InputKeyKeyNumpad0)[1].sequence))
	expectEqualStrings(t, "\x1bO16p", string(InputFunctionKeysKeys.Get(InputKeyKeyNumpad0)[15].sequence))
	expectEqualStrings(t, "\x1bO5M", string(InputFunctionKeysKeys.Get(InputKeyKeyNumpadEnter)[4].sequence))
}

// Zig: test "term51: keys: only the expected keys have entries and the counts match"
func TestKpFunctionKeys_keys_only_the_expected_keys_have_entries_and_the_counts_match(t *testing.T) {
	nonempty := 0
	total_entries := 0
	for _, k := range InputKeyKeyValues() {
		entries := InputFunctionKeysKeys.Get(k)
		var expect_len int
		switch k {
		case InputKeyKeyArrowUp, InputKeyKeyArrowDown, InputKeyKeyArrowRight, InputKeyKeyArrowLeft:
			expect_len = 17
		case InputKeyKeyHome, InputKeyKeyEnd:
			expect_len = 17
		case InputKeyKeyInsert, InputKeyKeyDelete, InputKeyKeyPageUp, InputKeyKeyPageDown:
			expect_len = 16
		case InputKeyKeyF1, InputKeyKeyF2, InputKeyKeyF3, InputKeyKeyF4, InputKeyKeyF5, InputKeyKeyF6:
			expect_len = 16
		case InputKeyKeyF7, InputKeyKeyF8, InputKeyKeyF9, InputKeyKeyF10, InputKeyKeyF11, InputKeyKeyF12:
			expect_len = 16
		case InputKeyKeyNumpad0, InputKeyKeyNumpad1, InputKeyKeyNumpad2, InputKeyKeyNumpad3, InputKeyKeyNumpad4:
			expect_len = 16
		case InputKeyKeyNumpad5, InputKeyKeyNumpad6, InputKeyKeyNumpad7, InputKeyKeyNumpad8, InputKeyKeyNumpad9:
			expect_len = 16
		case InputKeyKeyNumpadDecimal, InputKeyKeyNumpadDivide, InputKeyKeyNumpadMultiply:
			expect_len = 16
		case InputKeyKeyNumpadSubtract, InputKeyKeyNumpadAdd:
			expect_len = 16
		case InputKeyKeyNumpadEnter:
			expect_len = 17
		case InputKeyKeyNumpadUp, InputKeyKeyNumpadDown, InputKeyKeyNumpadRight, InputKeyKeyNumpadLeft:
			expect_len = 17
		case InputKeyKeyNumpadBegin, InputKeyKeyNumpadHome, InputKeyKeyNumpadEnd:
			expect_len = 17
		case InputKeyKeyNumpadInsert, InputKeyKeyNumpadDelete:
			expect_len = 16
		case InputKeyKeyNumpadPageUp, InputKeyKeyNumpadPageDown:
			expect_len = 16
		case InputKeyKeyBackspace:
			expect_len = 29
		case InputKeyKeyTab:
			expect_len = 18
		case InputKeyKeyEnter:
			expect_len = 17
		case InputKeyKeyEscape:
			expect_len = 16
		default:
			expect_len = 0
		}
		expectEqual(t, expect_len, len(entries))
		if len(entries) > 0 {
			nonempty += 1
		}
		total_entries += len(entries)
	}
	expectEqual(t, 53, nonempty)

	// 6 cursor (17) + 4 editing (16) + 12 F (16) + 15 keypad (16) +
	// numpad_enter (17) + 7 keypad cursor (17) + 4 keypad editing (16)
	// + backspace 29 + tab 18 + enter 17 + escape 16
	expectEqual(
		t,
		6*17+4*16+12*16+15*16+17+7*17+4*16+29+18+17+16,
		total_entries,
	)

	// Keys that are not special-cased at all.
	expectEqual(t, 0, len(InputFunctionKeysKeys.Get(InputKeyKeyUnidentified)))
	expectEqual(t, 0, len(InputFunctionKeysKeys.Get(InputKeyKeySpace)))
	expectEqual(t, 0, len(InputFunctionKeysKeys.Get(InputKeyKeyNumpadEqual)))
	expectEqual(t, 0, len(InputFunctionKeysKeys.Get(InputKeyKeyNumpadBackspace)))
	expectEqual(t, 0, len(InputFunctionKeysKeys.Get(InputKeyKeyNumpadSeparator)))
}

// Zig: test "term51: keys: backspace entries"
func TestKpFunctionKeys_keys_backspace_entries(t *testing.T) {
	kpFKExpectEntries(t, kpFKBackspace, InputFunctionKeysKeys.Get(InputKeyKeyBackspace))
	// Spot checks against explicit values.
	e := InputFunctionKeysKeys.Get(InputKeyKeyBackspace)
	kpFKExpectEntry(t, e[0], InputKeyMods{Shift: true}, true, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysSet, "\x7f")
	kpFKExpectEntry(t, e[3], kpFKMods(true, true, false, false), true, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysSet, "\x08")
	kpFKExpectEntry(t, e[13], InputKeyMods{Shift: true}, true, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysSetOther, "\x1b[27;2;127~")
	kpFKExpectEntry(t, e[26], kpFKMods(true, true, true, true), true, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysSetOther, "\x1b[27;16;127~")
	kpFKExpectEntry(t, e[27], InputKeyMods{Ctrl: true}, true, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysAny, "\x08")
	kpFKExpectEntry(t, e[28], InputKeyMods{}, true, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysAny, "\x7f")
}

// Zig: test "term51: keys: backspace modify-other-keys structure"
func TestKpFunctionKeys_keys_backspace_modify_other_keys_structure(t *testing.T) {
	e := InputFunctionKeysKeys.Get(InputKeyKeyBackspace)
	// 13 entries for modify_other_keys=set, 14 for set_other, 2 fallbacks.
	for _, x := range e[0:13] {
		expectEqual(t, InputFunctionKeysModifyKeysSet, x.modify_other_keys)
	}
	for _, x := range e[13:27] {
		expectEqual(t, InputFunctionKeysModifyKeysSetOther, x.modify_other_keys)
	}
	for _, x := range e[27:29] {
		expectEqual(t, InputFunctionKeysModifyKeysAny, x.modify_other_keys)
	}

	// The set_other entries emit CSI 27;<code>;127~ where code matches the
	// modifier combination, in modifier-table order but skipping code 5
	// (ctrl alone, which is the plain-ctrl fallback entry below).
	codes := [...]int{2, 3, 4, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	for i, x := range e[13:27] {
		code := codes[i]
		want := fmt.Sprintf("\x1b[27;%d;127~", code)
		expectEqualStrings(t, want, string(x.sequence))
		// The modifier set decodes to that xterm code.
		expectEqual(t, code, kpFKBits(x.mods)+1)
	}
	// Plain ctrl (code 5) has no set_other entry; it uses the fallback.
	for _, x := range e[13:27] {
		expect(t, !x.mods.Equal(InputKeyMods{Ctrl: true}), "!x.mods.equal(.{ .ctrl = true })")
	}

	// No entry in the whole backspace table sets lock/side bits.
	for _, x := range e {
		expect(t, x.mods.Int()&^uint16(0b1111) == 0, "x.mods.int() & ~@as(u16, 0b1111) == 0")
	}
	// Every entry has any-cursor, any-keypad, mods_empty_is_any true.
	for _, x := range e {
		expect(t, x.mods_empty_is_any, "x.mods_empty_is_any")
		expectEqual(t, InputFunctionKeysCursorModeAny, x.cursor)
		expectEqual(t, InputFunctionKeysKeypadModeAny, x.keypad)
	}
}

// Zig: test "term51: keys: tab entries"
func TestKpFunctionKeys_keys_tab_entries(t *testing.T) {
	kpFKExpectEntries(t, kpFKTab, InputFunctionKeysKeys.Get(InputKeyKeyTab))
	e := InputFunctionKeysKeys.Get(InputKeyKeyTab)
	kpFKExpectEntry(t, e[0], InputKeyMods{Shift: true}, true, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysSet, "\x1b[Z")
	kpFKExpectEntry(t, e[1], InputKeyMods{Alt: true}, true, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysSet, "\x1b\t")
	kpFKExpectEntry(t, e[2], InputKeyMods{Shift: true}, true, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysSetOther, "\x1b[27;2;9~")
	kpFKExpectEntry(t, e[3], InputKeyMods{Alt: true}, true, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysSetOther, "\x1b[27;3;9~")
	kpFKExpectEntry(t, e[4], kpFKMods(true, false, true, false), true, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysAny, "\x1b[27;4;9~")
	kpFKExpectEntry(t, e[17], InputKeyMods{}, true, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysAny, "\t")

	// Entries 4..16 are modifier codes 4..16 except 2 and 3, in table order.
	for i, x := range e[4:17] {
		code := 4 + i
		want := fmt.Sprintf("\x1b[27;%d;9~", code)
		expectEqualStrings(t, want, string(x.sequence))
		expectEqual(t, InputFunctionKeysModifyKeysAny, x.modify_other_keys)
	}
}

// Zig: test "term51: keys: enter entries"
func TestKpFunctionKeys_keys_enter_entries(t *testing.T) {
	kpFKExpectEntries(t, kpFKEnter, InputFunctionKeysKeys.Get(InputKeyKeyEnter))
	e := InputFunctionKeysKeys.Get(InputKeyKeyEnter)
	kpFKExpectEntry(t, e[0], InputKeyMods{Shift: true}, true, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysAny, "\x1b[27;2;13~")
	kpFKExpectEntry(t, e[1], InputKeyMods{Alt: true}, true, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysSet, "\x1b\r")
	kpFKExpectEntry(t, e[2], InputKeyMods{Alt: true}, true, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysSetOther, "\x1b[27;3;13~")
	kpFKExpectEntry(t, e[3], kpFKMods(true, false, true, false), true, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysAny, "\x1b[27;4;13~")
	kpFKExpectEntry(t, e[15], kpFKMods(true, true, true, true), true, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysAny, "\x1b[27;16;13~")
	kpFKExpectEntry(t, e[16], InputKeyMods{}, true, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysAny, "\r")

	// Entries 3..15 are modifier codes 4..16 minus none: code = index + 1.
	for i, x := range e[3:16] {
		code := 4 + i
		want := fmt.Sprintf("\x1b[27;%d;13~", code)
		expectEqualStrings(t, want, string(x.sequence))
	}
}

// Zig: test "term51: keys: escape entries"
func TestKpFunctionKeys_keys_escape_entries(t *testing.T) {
	kpFKExpectEntries(t, kpFKEscape, InputFunctionKeysKeys.Get(InputKeyKeyEscape))
	e := InputFunctionKeysKeys.Get(InputKeyKeyEscape)
	expectEqual(t, 16, len(e))
	// Entries 0..14 follow the modifier table; code 3 (alt) is the only one
	// that is not CSI 27;<code>;27~.
	for i, x := range e[0:15] {
		m := InputFunctionKeysModifiers[i]
		code := i + 2
		expectEqual(t, m.Int(), x.mods.Int())
		expectEqual(t, InputFunctionKeysModifyKeysAny, x.modify_other_keys)
		if code == 3 {
			expectEqualStrings(t, "\x1b\x1b", string(x.sequence))
		} else {
			want := fmt.Sprintf("\x1b[27;%d;27~", code)
			expectEqualStrings(t, want, string(x.sequence))
		}
	}
	kpFKExpectEntry(t, e[15], InputKeyMods{}, true, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysAny, "\x1b")
}

// Zig: test "term51: keys: every non-fallback entry of non-pc keys has non-empty distinct modifiers"
func TestKpFunctionKeys_keys_every_non_fallback_entry_of_non_pc_keys_has_non_empty_distinct_modifiers(t *testing.T) {
	// Within backspace/tab/enter/escape, (mods, modify_other_keys) pairs are
	// unique, so first-match lookup is unambiguous.
	lists := [...][]InputFunctionKeysEntry{
		InputFunctionKeysKeys.Get(InputKeyKeyBackspace), InputFunctionKeysKeys.Get(InputKeyKeyTab),
		InputFunctionKeysKeys.Get(InputKeyKeyEnter), InputFunctionKeysKeys.Get(InputKeyKeyEscape),
	}
	for _, l := range lists {
		for i, a := range l {
			expect(t, len(a.sequence) > 0, "a.sequence.len > 0")
			for _, b := range l[i+1:] {
				expect(t, !(a.mods.Int() == b.mods.Int() &&
					a.modify_other_keys == b.modify_other_keys), "!(a.mods.int() == b.mods.int() and a.modify_other_keys == b.modify_other_keys)")
			}
		}
	}
}

// Zig: test "term51: keys: sequence lengths and library build options"
func TestKpFunctionKeys_keys_sequence_lengths_and_library_build_options(t *testing.T) {
	// The upstream "keys" test is skipped in the library build because
	// it needs termio.zig; that part cannot be compiled here (see
	// findings). Here we assert the same bound with the constant from
	// src/termio/message.zig (`MessageData(u8, 38)`).
	// Zig-only: `@tagName(@import("terminal_options").artifact) == "lib"`;
	// Go has only the library build, so there is no build option to check.
	const termio_small_max = 38
	longest := 0
	for _, entries := range InputFunctionKeysKeys.values {
		for _, entry := range entries {
			expect(t, len(entry.sequence) <= termio_small_max, "entry.sequence.len <= termio_small_max")
			longest = max(longest, len(entry.sequence))
		}
	}
	// "\x1b[27;16;127~" is the longest sequence in the table.
	expectEqual(t, 12, longest)
}

// Zig: test "term51: pcStyle: expands one hole for each modifier combination"
func TestKpFunctionKeys_pcStyle_expands_one_hole_for_each_modifier_combination(t *testing.T) {
	r := inputFunctionKeysPcStyle([]uint8("\x1b[1;{}A"))
	expectEqual(t, 15, len(r))
	kpFKExpectPc(t, r, "\x1b[1;", "A", InputFunctionKeysKeypadModeAny)
	expectEqualStrings(t, "\x1b[1;2A", string(r[0].sequence))
	expectEqualStrings(t, "\x1b[1;16A", string(r[14].sequence))

	r2 := inputFunctionKeysPcStyle([]uint8("{}"))
	expectEqual(t, 15, len(r2))
	kpFKExpectPc(t, r2, "", "", InputFunctionKeysKeypadModeAny)
	expectEqualStrings(t, "2", string(r2[0].sequence))
	expectEqualStrings(t, "9", string(r2[7].sequence))
	expectEqualStrings(t, "10", string(r2[8].sequence))
	expectEqualStrings(t, "16", string(r2[14].sequence))

	r3 := inputFunctionKeysPcStyle([]uint8("ab{}cd{{}}"))
	// `{{` and `}}` are literal braces in comptimePrint formats.
	expectEqualStrings(t, "ab2cd{}", string(r3[0].sequence))
	expectEqualStrings(t, "ab16cd{}", string(r3[14].sequence))

	// Zig-only: `@typeInfo(@TypeOf(pcStyle)).@"fn".return_type.? == []Entry`
	// (the Go signature returns a mutable []InputFunctionKeysEntry).
}

// Zig: test "term51: pcStyle: entries carry only mods and sequence"
func TestKpFunctionKeys_pcStyle_entries_carry_only_mods_and_sequence(t *testing.T) {
	r := inputFunctionKeysPcStyle([]uint8("\x1b[2;{}~"))
	for _, e := range r {
		expect(t, e.mods_empty_is_any, "e.mods_empty_is_any")
		expectEqual(t, InputFunctionKeysCursorModeAny, e.cursor)
		expectEqual(t, InputFunctionKeysKeypadModeAny, e.keypad)
		expectEqual(t, InputFunctionKeysModifyKeysAny, e.modify_other_keys)
	}
}

// Zig: test "term51: kpDefault: application keypad entry with no modifiers"
func TestKpFunctionKeys_kpDefault_application_keypad_entry_with_no_modifiers(t *testing.T) {
	r := inputFunctionKeysKpDefault([]uint8("p"))
	expectEqual(t, 1, len(r))
	kpFKExpectEntry(t, r[0], InputKeyMods{}, false, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeApplication, InputFunctionKeysModifyKeysAny, "\x1bOp")

	r2 := inputFunctionKeysKpDefault([]uint8(""))
	expectEqual(t, 1, len(r2))
	kpFKExpectEntry(t, r2[0], InputKeyMods{}, false, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeApplication, InputFunctionKeysModifyKeysAny, "\x1bO")

	r3 := inputFunctionKeysKpDefault([]uint8("xy"))
	kpFKExpectEntry(t, r3[0], InputKeyMods{}, false, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeApplication, InputFunctionKeysModifyKeysAny, "\x1bOxy")

	r4 := inputFunctionKeysKpDefault([]uint8("M"))
	kpFKExpectEntry(t, r4[0], InputKeyMods{}, false, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeApplication, InputFunctionKeysModifyKeysAny, "\x1bOM")
}

// Zig: test "term51: kpKeys: default entry followed by application-keypad modifier entries"
func TestKpFunctionKeys_kpKeys_default_entry_followed_by_application_keypad_modifier_entries(t *testing.T) {
	r := inputFunctionKeysKpKeys([]uint8("r"))
	expectEqual(t, 16, len(r))
	kpFKExpectEntry(t, r[0], InputKeyMods{}, false, InputFunctionKeysCursorModeAny, InputFunctionKeysKeypadModeApplication, InputFunctionKeysModifyKeysAny, "\x1bOr")
	kpFKExpectPc(t, r[1:], "\x1bO", "r", InputFunctionKeysKeypadModeApplication)
	expectEqualStrings(t, "\x1bO2r", string(r[1].sequence))
	expectEqualStrings(t, "\x1bO16r", string(r[15].sequence))
	for _, e := range r[1:] {
		expect(t, e.mods_empty_is_any, "e.mods_empty_is_any")
		expectEqual(t, InputFunctionKeysCursorModeAny, e.cursor)
		expectEqual(t, InputFunctionKeysModifyKeysAny, e.modify_other_keys)
	}

	r2 := inputFunctionKeysKpKeys([]uint8("M"))
	expectEqual(t, 16, len(r2))
	expectEqualStrings(t, "\x1bOM", string(r2[0].sequence))
	expectEqualStrings(t, "\x1bO3M", string(r2[2].sequence))

	// kpKeys sets .keypad on the array it got from its own pcStyle call; a
	// separate pcStyle call with the same format string yields entries that
	// still have keypad .any (pins observed behaviour: the mutation is not
	// visible through a second call).
	plain := inputFunctionKeysPcStyle([]uint8("\x1bO{}r"))
	for _, e := range plain {
		expectEqual(t, InputFunctionKeysKeypadModeAny, e.keypad)
	}
}

// Zig: test "term51: cursorKey: normal and application entries"
func TestKpFunctionKeys_cursorKey_normal_and_application_entries(t *testing.T) {
	r := inputFunctionKeysCursorKey([]uint8("\x1b[A"), []uint8("\x1bOA"))
	expectEqual(t, 2, len(r))
	kpFKExpectEntry(t, r[0], InputKeyMods{}, true, InputFunctionKeysCursorModeNormal, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysAny, "\x1b[A")
	kpFKExpectEntry(t, r[1], InputKeyMods{}, true, InputFunctionKeysCursorModeApplication, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysAny, "\x1bOA")

	r2 := inputFunctionKeysCursorKey([]uint8(""), []uint8("z"))
	expectEqual(t, 2, len(r2))
	kpFKExpectEntry(t, r2[0], InputKeyMods{}, true, InputFunctionKeysCursorModeNormal, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysAny, "")
	kpFKExpectEntry(t, r2[1], InputKeyMods{}, true, InputFunctionKeysCursorModeApplication, InputFunctionKeysKeypadModeAny, InputFunctionKeysModifyKeysAny, "z")
}
