// Port of src/input/kitty.zig (Ghostty v1.3.1). The file has no
// functions, only two tables.
//
// File naming: kitty.zig lives in src/input/, so the file is
// `input_kitty.go` and every declaration takes the prefix `InputKitty`
// (`inputKitty` for private ones): `kitty.Entry` is `InputKittyEntry`,
// `kitty.entries` is `InputKittyEntries`, the private `RawEntry` and
// `raw_entries` are `inputKittyRawEntry` and `inputKittyRawEntries`. This
// is the input-side kitty keymap, not src/terminal/kitty/ (kitty_key.go,
// kitty_color.go).
//
// Declarations appear in the same order as in kitty.zig.
//
// # Comptime
//
// `entries` is built at compile time from `raw_entries`, one Entry per raw
// tuple in the same order. In Go both are package-level variables that the
// implementation initialises once at start-up (docs/mapping.md, "comptime
// table"): `raw_entries` is a Go slice literal generated mechanically from
// kitty.zig by a throwaway script (81 entries, same order), and `entries`
// is built from it by inputKittyEntriesInit, the comptime block.
//
// # Declarations that are not Zig declarations
//
// None. `RawEntry` is a Zig tuple struct `struct { key.Key, u21, u8, bool }`;
// its fields "0" to "3" begin with a digit and so take the prefix `x` (as
// `x132_column` in modes.go): `x0`, `x1`, `x2`, `x3`. A test's `raw[0]` is
// `raw.x0`.
//
// # Omitted
//
// The import `key` (nothing to declare). The file-level `test { _ =
// entries; }` only forces comptime evaluation.
// No C ABI declaration exists in this file.
//
// # Zig defaults a Go literal does not give
//
// None: Entry has no field defaults.

package terminal

// InputKittyEntry is kitty.Entry: a single entry in the kitty keymap data.
// There are only ~100 entries so the recommendation is to just use a linear
// search to find the entry for a given key.
type InputKittyEntry struct {
	key InputKeyKey
	// Zig: u21.
	code     uint32
	final    uint8
	modifier bool
}

// InputKittyEntries is kitty.entries: the full list of entries for the
// current platform.
var InputKittyEntries = inputKittyEntriesInit()

// inputKittyEntriesInit is the comptime block `entries: { ... }` that
// initialises kitty.entries.
func inputKittyEntriesInit() []InputKittyEntry {
	result := make([]InputKittyEntry, len(inputKittyRawEntries))
	for i, raw := range inputKittyRawEntries {
		result[i] = InputKittyEntry{
			key:      raw.x0,
			code:     raw.x1,
			final:    raw.x2,
			modifier: raw.x3,
		}
	}

	final := result
	return final
}

// inputKittyRawEntry is kitty.RawEntry: raw entry is the tuple form of an
// entry for easy human management. This should never be used in a real
// program so it is not pub. For real programs, use `entries` which has
// properly typed, structured data.
//
// Zig: struct { key.Key, u21, u8, bool }.
type inputKittyRawEntry struct {
	x0 InputKeyKey
	// Zig: u21.
	x1 uint32
	x2 uint8
	x3 bool
}

// inputKittyRawEntries is kitty.raw_entries: the raw data for how to map
// keys to Kitty data. Based on the information:
// https://sw.kovidgoyal.net/kitty/keyboard-protocol/#functional-key-definitions
// And the exact table is ported from Foot:
// https://codeberg.org/dnkl/foot/src/branch/master/kitty-keymap.h
//
// Note that we currently don't support all the same keysyms as Kitty,
// but we can add them as we add support.
//
// These are stored in order of the above Kitty link, so it is easy to
// find the entry for a given key.
var inputKittyRawEntries = []inputKittyRawEntry{
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
