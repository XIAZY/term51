// WIREFRAME for src/input/function_keys.zig (Ghostty v1.3.1). NOT YET
// PORTED.
//
// This is the list of "PC style function keys" that xterm supports for
// the legacy keyboard protocols. These always take priority since at the
// time of writing this, even the most modern keyboard protocols still
// are backwards compatible with regards to these sequences.
//
// This is based on a variety of sources cross-referenced but mostly
// based on foot's keymap.h: https://codeberg.org/dnkl/foot/src/branch/master/keymap.h
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented) and every table is left at its zero value. The
// implementation is added only after the project owner and the project lead
// both approve it. Do not fill in bodies before then. See the header of
// color.go for the rules.
//
// File naming: function_keys.zig lives in src/input/, so the file is
// `input_function_keys.go` and every declaration takes the prefix
// `InputFunctionKeys` (`inputFunctionKeys` for private ones):
// `function_keys.CursorMode` is `InputFunctionKeysCursorMode`,
// `KeypadMode` is `InputFunctionKeysKeypadMode`, `ModifyKeys` is
// `InputFunctionKeysModifyKeys`, `Entry` is `InputFunctionKeysEntry`,
// `modifiers` is `InputFunctionKeysModifiers`, `KeyEntryArray` is
// `InputFunctionKeysKeyEntryArray`, `keys` is `InputFunctionKeysKeys`; the
// private `kpDefault`, `kpKeys`, `cursorKey`, `pcStyle` are
// `inputFunctionKeysKpDefault`, `inputFunctionKeysKpKeys`,
// `inputFunctionKeysCursorKey`, `inputFunctionKeysPcStyle`.
//
// Declarations appear in the same order as in function_keys.zig.
//
// # Comptime
//
// `keys` is built at compile time with `pcStyle`, `cursorKey`, `kpKeys` and
// `kpDefault`, which take `comptime` string parameters and are only ever
// evaluated at compile time. In Go `keys` is a package-level variable that
// the implementation initialises once at start-up (docs/mapping.md,
// "comptime table"), and the four helpers are ordinary functions that
// return a fresh slice on each call (each Zig call yields its own array;
// `kpKeys` sets `.keypad` on the entries of its own `pcStyle` call only).
// `pcStyle`'s format has exactly one "{}" hole for the xterm modifier code
// (2 to 16) and uses std.fmt syntax: "{{" and "}}" are literal braces (a
// project test uses "ab{}cd{{}}").
//
// `KeyEntryArray` is `std.EnumArray(key.Key, []const Entry)`: one slice of
// entries per Key value. Go: a struct with the array `values` (Zig's field
// name, which a test reads) indexed by the key's integer, plus the std
// operations function_keys.zig and the tests use: `InitFill`, `Get`, `Set`.
//
// # Declarations that are not Zig declarations
//
//   - `InputFunctionKeysKeyEntryArrayInitFill` and the methods `Get` and
//     `Set` (std.EnumArray's `initFill`, `get`, `set`);
//   - the `...Count` constants and the `String` methods of the enums.
//
// # Omitted
//
// The import aliases `std` and `key` (nothing to declare). The upstream
// test "keys" needs termio.zig (it returns SkipZigTest in the library
// build).
// No C ABI declaration exists in this file.
//
// # Zig defaults a Go literal does not give
//
// A Go composite literal leaves an omitted field at the Go zero value. The
// Zig default that is NOT the Go zero value, which a translated literal
// must write out:
//   - `Entry.mods_empty_is_any` defaults to TRUE.
// So Zig's `.{ .sequence = "x" }` is
// `InputFunctionKeysEntry{mods_empty_is_any: true, sequence: []uint8("x")}`,
// and every Entry literal of upstream's tables (backspace, tab, enter,
// escape, cursorKey, pcStyle) has `mods_empty_is_any: true`; only
// `kpDefault` sets it false. All other Entry defaults (`mods` .{}, `cursor`
// .any, `keypad` .any, `modify_other_keys` .any) are the Go zero value.
// `sequence` has no default.

package terminal

// InputFunctionKeysCursorMode is function_keys.CursorMode.
//
// Zig: enum, values numbered from zero in the order listed.
type InputFunctionKeysCursorMode uint8

const (
	InputFunctionKeysCursorModeAny InputFunctionKeysCursorMode = iota
	InputFunctionKeysCursorModeNormal
	InputFunctionKeysCursorModeApplication

	inputFunctionKeysCursorModeCount = iota
)

// String returns the Zig tag name (@tagName).
func (self InputFunctionKeysCursorMode) String() string { panic(notImplemented) }

// InputFunctionKeysKeypadMode is function_keys.KeypadMode.
//
// Zig: enum, values numbered from zero in the order listed.
type InputFunctionKeysKeypadMode uint8

const (
	InputFunctionKeysKeypadModeAny InputFunctionKeysKeypadMode = iota
	InputFunctionKeysKeypadModeNormal
	InputFunctionKeysKeypadModeApplication

	inputFunctionKeysKeypadModeCount = iota
)

// String returns the Zig tag name (@tagName).
func (self InputFunctionKeysKeypadMode) String() string { panic(notImplemented) }

// InputFunctionKeysModifyKeys is function_keys.ModifyKeys.
//
// A bit confusing so I'll document this one: this is the "modify other keys"
// setting. We only change behavior for "set_other" which is ESC [ > 4; 2 m.
// So this can be "any" which means we don't care what's going on. Or it
// can be "set" which means modify keys must be set EXCEPT FOR "other keys"
// mode, and "set_other" which means modify keys must be set to "other keys"
// mode.
//
// See: https://invisible-island.net/xterm/modified-keys.html
//
// Zig: enum, values numbered from zero in the order listed.
type InputFunctionKeysModifyKeys uint8

const (
	InputFunctionKeysModifyKeysAny InputFunctionKeysModifyKeys = iota
	InputFunctionKeysModifyKeysSet
	InputFunctionKeysModifyKeysSetOther

	inputFunctionKeysModifyKeysCount = iota
)

// String returns the Zig tag name (@tagName).
func (self InputFunctionKeysModifyKeys) String() string { panic(notImplemented) }

// InputFunctionKeysEntry is function_keys.Entry: a single entry in the
// table of keys.
//
// See the file header: `mods_empty_is_any` defaults to true in Zig, which
// is NOT the Go zero value.
type InputFunctionKeysEntry struct {
	// The exact set of modifiers that must be active for this entry to match.
	// If mods_empty_is_any is true then empty mods means any set of mods can
	// match. Otherwise, empty mods means no mods must be active.
	//
	// Zig default: .{} (the zero value).
	mods InputKeyMods
	// Zig default: TRUE (NOT the Go zero value).
	mods_empty_is_any bool

	// The state required for cursor/keypad mode.
	//
	// Zig defaults: .any (the zero value).
	cursor InputFunctionKeysCursorMode
	keypad InputFunctionKeysKeypadMode

	// Whether or not this entry should be used
	//
	// Zig default: .any (the zero value).
	modify_other_keys InputFunctionKeysModifyKeys

	// The sequence to send to the pty if this entry matches.
	//
	// Zig: []const u8, no default.
	sequence []uint8
}

// InputFunctionKeysModifiers is function_keys.modifiers: the list of
// modifier combinations for modify other key sequences. The mode value is
// index + 2.
//
// Wireframe: left nil; the implementation fills it with upstream's 15
// entries in upstream's order (shift; alt; shift+alt; ctrl; shift+ctrl;
// alt+ctrl; shift+alt+ctrl; super; ... ; shift+alt+ctrl+super).
var InputFunctionKeysModifiers []InputKeyMods

// InputFunctionKeysKeyEntryArray is function_keys.KeyEntryArray: this is
// the array of entries for the PC style function keys as mapped to our set
// of possible key codes. Not every key has any entries.
//
// Zig: std.EnumArray(key.Key, []const Entry). See the file header.
type InputFunctionKeysKeyEntryArray struct {
	// Zig: values: [Indexer.count][]const Entry, indexed by the key's
	// integer value.
	values [inputKeyKeyCount][]InputFunctionKeysEntry
}

// InputFunctionKeysKeyEntryArrayInitFill is std.EnumArray.initFill: every
// key gets value.
func InputFunctionKeysKeyEntryArrayInitFill(value []InputFunctionKeysEntry) InputFunctionKeysKeyEntryArray {
	panic(notImplemented)
}

// Get is std.EnumArray.get: the entries of key.
func (self *InputFunctionKeysKeyEntryArray) Get(key InputKeyKey) []InputFunctionKeysEntry {
	panic(notImplemented)
}

// Set is std.EnumArray.set: replace the entries of key.
func (self *InputFunctionKeysKeyEntryArray) Set(key InputKeyKey, value []InputFunctionKeysEntry) {
	panic(notImplemented)
}

// InputFunctionKeysKeys is function_keys.keys: the list of keys that we
// support and their entry values. It is expected that you'll just use a
// for loop through the entry values, there are never more than a couple
// dozen or so.
//
// Wireframe: left zero (every key has no entries); the implementation
// builds it at start-up exactly as upstream's comptime block does.
var InputFunctionKeysKeys InputFunctionKeysKeyEntryArray

// inputFunctionKeysKpDefault is function_keys.kpDefault: returns the
// default keypad application mode entry.
//
// Zig: fn kpDefault(comptime suffix: []const u8) []const Entry.
func inputFunctionKeysKpDefault(suffix []uint8) []InputFunctionKeysEntry {
	panic(notImplemented)
}

// inputFunctionKeysKpKeys is function_keys.kpKeys: returns the entries for
// a keypad key. The suffix is the final character of the sent sequence,
// such as "r" for kp_2.
//
// Zig: fn kpKeys(comptime suffix: []const u8) []const Entry.
func inputFunctionKeysKpKeys(suffix []uint8) []InputFunctionKeysEntry {
	panic(notImplemented)
}

// inputFunctionKeysCursorKey is function_keys.cursorKey: returns entries
// that are dependent on cursor key settings.
//
// Zig: fn cursorKey(comptime normal: []const u8, comptime application:
// []const u8) []const Entry.
func inputFunctionKeysCursorKey(normal []uint8, application []uint8) []InputFunctionKeysEntry {
	panic(notImplemented)
}

// inputFunctionKeysPcStyle is function_keys.pcStyle: constructs a set of
// pcStyle function keys using the given format. The format should have
// exactly one "hole" for the mods code.
// Example: "\x1b[11;{}~" for F1.
//
// Zig: fn pcStyle(comptime fmt: []const u8) []Entry. One entry per
// `modifiers` element, sequence = std.fmt.comptimePrint(fmt, .{code}) with
// code = index + 2. See the file header.
func inputFunctionKeysPcStyle(fmt []uint8) []InputFunctionKeysEntry {
	panic(notImplemented)
}
