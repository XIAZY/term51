// WIREFRAME for src/input/kitty.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. The file has no functions; its two
// tables are left at their zero value. The implementation is added only
// after the project owner and the project lead both approve it. Do not fill
// in the tables before then. See the header of color.go for the rules.
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
// table").
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
//
// Wireframe: left nil; the implementation fills it at start-up from
// inputKittyRawEntries (81 entries, same order).
var InputKittyEntries []InputKittyEntry

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
//
// Wireframe: left nil; the implementation fills it with upstream's 81
// entries in upstream's order (escape 27 'u' through alt_right 57449 'u').
var inputKittyRawEntries []inputKittyRawEntry
