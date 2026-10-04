// Tests translated one for one from src/input/kitty.zig (Ghostty v1.3.1 plus
// the term51 additional tests). Each Go test carries the exact Zig title in a
// "// Zig: test" comment. Helpers are prefixed kp.
//
// Translation notes:
//   - `raw[0]` .. `raw[3]` are the fields x0 .. x3 of inputKittyRawEntry.
//   - The Zig tuple type `struct { key.Key, u21, u8, bool }` of
//     term51ADKittyExpected is kpKittyWant (named fields).
//   - `inline for (@typeInfo(key.Key).@"enum".fields)` is a loop over
//     InputKeyKeyValues().
//   - `u21` is uint32.
//   - Tests that were not translated are listed in
//     scripts/not-translated-kp.txt.

package terminal

import "testing"

// kpKittyWant is the anonymous tuple `struct { key.Key, u21, u8, bool }` of
// term51ADKittyExpected: key, code, final byte, modifier flag.
type kpKittyWant struct {
	key      InputKeyKey
	code     uint32
	final    uint8
	modifier bool
}

// kpKittyExpected is term51ADKittyExpected: every entry of the table,
// transcribed by hand from the source in the same order.
var kpKittyExpected = []kpKittyWant{
	{InputKeyKeyEscape, 27, 'u', false},
	{InputKeyKeyEnter, 13, 'u', false},
	{InputKeyKeyTab, 9, 'u', false},
	{InputKeyKeyBackspace, 127, 'u', false},
	{InputKeyKeyInsert, 2, '~', false},
	{InputKeyKeyDelete, 3, '~', false},
	{InputKeyKeyArrowLeft, 1, 'D', false},
	{InputKeyKeyArrowRight, 1, 'C', false},
	{InputKeyKeyArrowUp, 1, 'A', false},
	{InputKeyKeyArrowDown, 1, 'B', false},
	{InputKeyKeyPageUp, 5, '~', false},
	{InputKeyKeyPageDown, 6, '~', false},
	{InputKeyKeyHome, 1, 'H', false},
	{InputKeyKeyEnd, 1, 'F', false},
	{InputKeyKeyCapsLock, 57358, 'u', true},
	{InputKeyKeyScrollLock, 57359, 'u', false},
	{InputKeyKeyNumLock, 57360, 'u', true},
	{InputKeyKeyPrintScreen, 57361, 'u', false},
	{InputKeyKeyPause, 57362, 'u', false},
	{InputKeyKeyF1, 1, 'P', false},
	{InputKeyKeyF2, 1, 'Q', false},
	{InputKeyKeyF3, 13, '~', false},
	{InputKeyKeyF4, 1, 'S', false},
	{InputKeyKeyF5, 15, '~', false},
	{InputKeyKeyF6, 17, '~', false},
	{InputKeyKeyF7, 18, '~', false},
	{InputKeyKeyF8, 19, '~', false},
	{InputKeyKeyF9, 20, '~', false},
	{InputKeyKeyF10, 21, '~', false},
	{InputKeyKeyF11, 23, '~', false},
	{InputKeyKeyF12, 24, '~', false},
	{InputKeyKeyF13, 57376, 'u', false},
	{InputKeyKeyF14, 57377, 'u', false},
	{InputKeyKeyF15, 57378, 'u', false},
	{InputKeyKeyF16, 57379, 'u', false},
	{InputKeyKeyF17, 57380, 'u', false},
	{InputKeyKeyF18, 57381, 'u', false},
	{InputKeyKeyF19, 57382, 'u', false},
	{InputKeyKeyF20, 57383, 'u', false},
	{InputKeyKeyF21, 57384, 'u', false},
	{InputKeyKeyF22, 57385, 'u', false},
	{InputKeyKeyF23, 57386, 'u', false},
	{InputKeyKeyF24, 57387, 'u', false},
	{InputKeyKeyF25, 57388, 'u', false},
	{InputKeyKeyNumpad0, 57399, 'u', false},
	{InputKeyKeyNumpad1, 57400, 'u', false},
	{InputKeyKeyNumpad2, 57401, 'u', false},
	{InputKeyKeyNumpad3, 57402, 'u', false},
	{InputKeyKeyNumpad4, 57403, 'u', false},
	{InputKeyKeyNumpad5, 57404, 'u', false},
	{InputKeyKeyNumpad6, 57405, 'u', false},
	{InputKeyKeyNumpad7, 57406, 'u', false},
	{InputKeyKeyNumpad8, 57407, 'u', false},
	{InputKeyKeyNumpad9, 57408, 'u', false},
	{InputKeyKeyNumpadDecimal, 57409, 'u', false},
	{InputKeyKeyNumpadDivide, 57410, 'u', false},
	{InputKeyKeyNumpadMultiply, 57411, 'u', false},
	{InputKeyKeyNumpadSubtract, 57412, 'u', false},
	{InputKeyKeyNumpadAdd, 57413, 'u', false},
	{InputKeyKeyNumpadEnter, 57414, 'u', false},
	{InputKeyKeyNumpadEqual, 57415, 'u', false},
	{InputKeyKeyNumpadSeparator, 57416, 'u', false},
	{InputKeyKeyNumpadLeft, 57417, 'u', false},
	{InputKeyKeyNumpadRight, 57418, 'u', false},
	{InputKeyKeyNumpadUp, 57419, 'u', false},
	{InputKeyKeyNumpadDown, 57420, 'u', false},
	{InputKeyKeyNumpadPageUp, 57421, 'u', false},
	{InputKeyKeyNumpadPageDown, 57422, 'u', false},
	{InputKeyKeyNumpadHome, 57423, 'u', false},
	{InputKeyKeyNumpadEnd, 57424, 'u', false},
	{InputKeyKeyNumpadInsert, 57425, 'u', false},
	{InputKeyKeyNumpadDelete, 57426, 'u', false},
	{InputKeyKeyNumpadBegin, 57427, 'u', false},
	{InputKeyKeyShiftLeft, 57441, 'u', true},
	{InputKeyKeyShiftRight, 57447, 'u', true},
	{InputKeyKeyControlLeft, 57442, 'u', true},
	{InputKeyKeyControlRight, 57448, 'u', true},
	{InputKeyKeyMetaLeft, 57444, 'u', true},
	{InputKeyKeyMetaRight, 57450, 'u', true},
	{InputKeyKeyAltLeft, 57443, 'u', true},
	{InputKeyKeyAltRight, 57449, 'u', true},
}

// kpKittyFind is term51ADKittyFind: linear search, as the Entry doc comment
// recommends.
func kpKittyFind(k InputKeyKey) Optional[InputKittyEntry] {
	for _, entry := range InputKittyEntries {
		if entry.key == k {
			return Optional[InputKittyEntry]{Valid: true, Value: entry}
		}
	}
	return Optional[InputKittyEntry]{}
}

// kpKittyExpect is term51ADKittyExpect.
func kpKittyExpect(tt *testing.T, k InputKeyKey, code uint32, final uint8, modifier bool) {
	tt.Helper()
	e := kpKittyFind(k)
	if !e.Valid {
		tt.Fatalf("no entry for key %d (error.TestUnexpectedResult)", int32(k))
	}
	expectEqual(tt, k, e.Value.key)
	expectEqual(tt, code, e.Value.code)
	expectEqual(tt, final, e.Value.final)
	expectEqual(tt, modifier, e.Value.modifier)
}

// Zig: test (unnamed, kitty.zig:131)
func TestKpKitty_unnamed_force_comptime(t *testing.T) {
	// To force comptime to test it
	_ = InputKittyEntries
}

// Zig: test "term51: entries: count"
func TestKpKitty_entries_count(t *testing.T) {
	expectEqual(t, 81, len(InputKittyEntries))
	expectEqual(t, 81, len(inputKittyRawEntries))
	expectEqual(t, 81, len(kpKittyExpected))
}

// Zig: test "term51: entries: matches raw_entries in order"
func TestKpKitty_entries_matches_raw_entries_in_order(t *testing.T) {
	// Go-only guard: a loop over an empty table would pass without checking anything.
	expect(t, len(InputKittyEntries) > 0, "len(entries) > 0")
	expectEqual(t, len(inputKittyRawEntries), len(InputKittyEntries))
	for i, raw := range inputKittyRawEntries {
		expectEqual(t, raw.x0, InputKittyEntries[i].key)
		expectEqual(t, raw.x1, InputKittyEntries[i].code)
		expectEqual(t, raw.x2, InputKittyEntries[i].final)
		expectEqual(t, raw.x3, InputKittyEntries[i].modifier)
	}
}

// Zig: test "term51: entries: complete table-driven check of every entry"
func TestKpKitty_entries_complete_table_driven_check_of_every_entry(t *testing.T) {
	expectEqual(t, len(kpKittyExpected), len(InputKittyEntries))
	for i, want := range kpKittyExpected {
		expectEqual(t, want.key, InputKittyEntries[i].key)
		expectEqual(t, want.code, InputKittyEntries[i].code)
		expectEqual(t, want.final, InputKittyEntries[i].final)
		expectEqual(t, want.modifier, InputKittyEntries[i].modifier)
	}
}

// Zig: test "term51: entries: keys are unique"
func TestKpKitty_entries_keys_are_unique(t *testing.T) {
	// Go-only guard: a loop over an empty table would pass without checking anything.
	expect(t, len(InputKittyEntries) > 0, "len(entries) > 0")
	for i, a := range InputKittyEntries {
		for _, b := range InputKittyEntries[i+1:] {
			expect(t, a.key != b.key, "a.key != b.key")
		}
	}
}

// Zig: test "term51: entries: code and final pairs are unique"
func TestKpKitty_entries_code_and_final_pairs_are_unique(t *testing.T) {
	// Go-only guard: a loop over an empty table would pass without checking anything.
	expect(t, len(InputKittyEntries) > 0, "len(entries) > 0")
	for i, a := range InputKittyEntries {
		for _, b := range InputKittyEntries[i+1:] {
			expect(t, a.code != b.code || a.final != b.final, "a.code != b.code or a.final != b.final")
		}
	}
}

// Zig: test "term51: entries: final bytes and code classes"
func TestKpKitty_entries_final_bytes_and_code_classes(t *testing.T) {
	var u_count, tilde_count, letter_count int
	for _, e := range InputKittyEntries {
		switch e.final {
		case 'u':
			u_count += 1
		case '~':
			tilde_count += 1
		case 'A', 'B', 'C', 'D', 'H', 'F', 'P', 'Q', 'S':
			letter_count += 1
			// Every letter-final entry uses code 1.
			expectEqual(t, uint32(1), e.code)
		default:
			t.Fatalf("error.TestUnexpectedResult: unexpected final byte %q", e.final)
		}

		// Codes with a '~' final are the legacy numeric ones.
		if e.final == '~' {
			expect(t, e.code < 25, "e.code < 25")
		}
	}
	expectEqual(t, 59, u_count)
	expectEqual(t, 13, tilde_count)
	expectEqual(t, 9, letter_count)
}

// Zig: test "term51: entries: modifier flag set only on lock and modifier keys"
func TestKpKitty_entries_modifier_flag_set_only_on_lock_and_modifier_keys(t *testing.T) {
	count := 0
	for _, e := range InputKittyEntries {
		var expected bool
		switch e.key {
		case InputKeyKeyCapsLock,
			InputKeyKeyNumLock,
			InputKeyKeyShiftLeft,
			InputKeyKeyShiftRight,
			InputKeyKeyControlLeft,
			InputKeyKeyControlRight,
			InputKeyKeyMetaLeft,
			InputKeyKeyMetaRight,
			InputKeyKeyAltLeft,
			InputKeyKeyAltRight:
			expected = true
		default:
			expected = false
		}
		expectEqual(t, expected, e.modifier)
		if e.modifier {
			count += 1
		}
	}
	expectEqual(t, 10, count)
}

// Zig: test "term51: entries: modifier entries use the private use area codes"
func TestKpKitty_entries_modifier_entries_use_the_private_use_area_codes(t *testing.T) {
	for _, e := range InputKittyEntries {
		if e.modifier && e.key != InputKeyKeyCapsLock && e.key != InputKeyKeyNumLock {
			expect(t, e.code >= 57441 && e.code <= 57450, "e.code >= 57441 and e.code <= 57450")
		}
	}
	// The left/right pairs are 6 apart: shift 57441/57447, ctrl 57442/57448,
	// alt 57443/57449, meta 57444/57450.
	expectEqual(
		t,
		zigUnwrap(t, kpKittyFind(InputKeyKeyShiftLeft)).code+6,
		zigUnwrap(t, kpKittyFind(InputKeyKeyShiftRight)).code,
	)
	expectEqual(
		t,
		zigUnwrap(t, kpKittyFind(InputKeyKeyControlLeft)).code+6,
		zigUnwrap(t, kpKittyFind(InputKeyKeyControlRight)).code,
	)
	expectEqual(
		t,
		zigUnwrap(t, kpKittyFind(InputKeyKeyAltLeft)).code+6,
		zigUnwrap(t, kpKittyFind(InputKeyKeyAltRight)).code,
	)
	expectEqual(
		t,
		zigUnwrap(t, kpKittyFind(InputKeyKeyMetaLeft)).code+6,
		zigUnwrap(t, kpKittyFind(InputKeyKeyMetaRight)).code,
	)
}

// Zig: test "term51: entries: functional keys one by one"
func TestKpKitty_entries_functional_keys_one_by_one(t *testing.T) {
	kpKittyExpect(t, InputKeyKeyEscape, 27, 'u', false)
	kpKittyExpect(t, InputKeyKeyEnter, 13, 'u', false)
	kpKittyExpect(t, InputKeyKeyTab, 9, 'u', false)
	kpKittyExpect(t, InputKeyKeyBackspace, 127, 'u', false)
	kpKittyExpect(t, InputKeyKeyInsert, 2, '~', false)
	kpKittyExpect(t, InputKeyKeyDelete, 3, '~', false)
	kpKittyExpect(t, InputKeyKeyArrowLeft, 1, 'D', false)
	kpKittyExpect(t, InputKeyKeyArrowRight, 1, 'C', false)
	kpKittyExpect(t, InputKeyKeyArrowUp, 1, 'A', false)
	kpKittyExpect(t, InputKeyKeyArrowDown, 1, 'B', false)
	kpKittyExpect(t, InputKeyKeyPageUp, 5, '~', false)
	kpKittyExpect(t, InputKeyKeyPageDown, 6, '~', false)
	kpKittyExpect(t, InputKeyKeyHome, 1, 'H', false)
	kpKittyExpect(t, InputKeyKeyEnd, 1, 'F', false)
	kpKittyExpect(t, InputKeyKeyCapsLock, 57358, 'u', true)
	kpKittyExpect(t, InputKeyKeyScrollLock, 57359, 'u', false)
	kpKittyExpect(t, InputKeyKeyNumLock, 57360, 'u', true)
	kpKittyExpect(t, InputKeyKeyPrintScreen, 57361, 'u', false)
	kpKittyExpect(t, InputKeyKeyPause, 57362, 'u', false)
}

// Zig: test "term51: entries: function keys one by one"
func TestKpKitty_entries_function_keys_one_by_one(t *testing.T) {
	kpKittyExpect(t, InputKeyKeyF1, 1, 'P', false)
	kpKittyExpect(t, InputKeyKeyF2, 1, 'Q', false)
	kpKittyExpect(t, InputKeyKeyF3, 13, '~', false)
	kpKittyExpect(t, InputKeyKeyF4, 1, 'S', false)
	kpKittyExpect(t, InputKeyKeyF5, 15, '~', false)
	kpKittyExpect(t, InputKeyKeyF6, 17, '~', false)
	kpKittyExpect(t, InputKeyKeyF7, 18, '~', false)
	kpKittyExpect(t, InputKeyKeyF8, 19, '~', false)
	kpKittyExpect(t, InputKeyKeyF9, 20, '~', false)
	kpKittyExpect(t, InputKeyKeyF10, 21, '~', false)
	kpKittyExpect(t, InputKeyKeyF11, 23, '~', false)
	kpKittyExpect(t, InputKeyKeyF12, 24, '~', false)
	kpKittyExpect(t, InputKeyKeyF13, 57376, 'u', false)
	kpKittyExpect(t, InputKeyKeyF25, 57388, 'u', false)
}

// Zig: test "term51: entries: numpad and modifier keys one by one"
func TestKpKitty_entries_numpad_and_modifier_keys_one_by_one(t *testing.T) {
	kpKittyExpect(t, InputKeyKeyNumpad0, 57399, 'u', false)
	kpKittyExpect(t, InputKeyKeyNumpad9, 57408, 'u', false)
	kpKittyExpect(t, InputKeyKeyNumpadDecimal, 57409, 'u', false)
	kpKittyExpect(t, InputKeyKeyNumpadEnter, 57414, 'u', false)
	kpKittyExpect(t, InputKeyKeyNumpadBegin, 57427, 'u', false)
	kpKittyExpect(t, InputKeyKeyShiftLeft, 57441, 'u', true)
	kpKittyExpect(t, InputKeyKeyShiftRight, 57447, 'u', true)
	kpKittyExpect(t, InputKeyKeyControlLeft, 57442, 'u', true)
	kpKittyExpect(t, InputKeyKeyControlRight, 57448, 'u', true)
	kpKittyExpect(t, InputKeyKeyMetaLeft, 57444, 'u', true)
	kpKittyExpect(t, InputKeyKeyMetaRight, 57450, 'u', true)
	kpKittyExpect(t, InputKeyKeyAltLeft, 57443, 'u', true)
	kpKittyExpect(t, InputKeyKeyAltRight, 57449, 'u', true)
}

// Zig: test "term51: entries: keys without an entry"
func TestKpKitty_entries_keys_without_an_entry(t *testing.T) {
	// Everything that is not one of the 81 table keys is absent.
	present := 0
	absent := 0
	for _, k := range InputKeyKeyValues() {
		if kpKittyFind(k).Valid {
			present += 1
		} else {
			absent += 1
		}
	}
	expectEqual(t, 81, present)
	expectEqual(t, 176-81, absent)

	expect(t, !kpKittyFind(InputKeyKeyUnidentified).Valid, "find(.unidentified) == null")
	expect(t, !kpKittyFind(InputKeyKeyKeyA).Valid, "find(.key_a) == null")
	expect(t, !kpKittyFind(InputKeyKeySpace).Valid, "find(.space) == null")
	expect(t, !kpKittyFind(InputKeyKeyContextMenu).Valid, "find(.context_menu) == null")
	expect(t, !kpKittyFind(InputKeyKeyNumpadClear).Valid, "find(.numpad_clear) == null")
	expect(t, !kpKittyFind(InputKeyKeyFn).Valid, "find(.@\"fn\") == null")
	expect(t, !kpKittyFind(InputKeyKeyCopy).Valid, "find(.copy) == null")
}
