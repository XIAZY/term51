// Port of src/input/function_keys.zig (Ghostty v1.3.1).
//
// This is the list of "PC style function keys" that xterm supports for
// the legacy keyboard protocols. These always take priority since at the
// time of writing this, even the most modern keyboard protocols still
// are backwards compatible with regards to these sequences.
//
// This is based on a variety of sources cross-referenced but mostly
// based on foot's keymap.h: https://codeberg.org/dnkl/foot/src/branch/master/keymap.h
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
// project test uses "ab{}cd{{}}"). `std.fmt.comptimePrint(fmt, .{code})` is
// inputFunctionKeysComptimePrint, a port of the format-string scan of Zig
// 0.15.2's std.Io.Writer.print for the one argument: literal runs, "{{" and
// "}}" unescaped, and the empty placeholder "{}" printing the usize code in
// decimal. Every format error that is a compile error in Zig (a lone "}",
// a missing "}", a placeholder that is not "{}" or a hole count other than
// one) panics. Zig's `++` on the comptime entry arrays is
// inputFunctionKeysConcat (a new slice holding the parts in order); on the
// strings of kpDefault it is Go's append into a new slice.
//
// The `keys` comptime block is inputFunctionKeysKeysInit, called once to
// initialise InputFunctionKeysKeys. Its `result.set(...)` lines and the
// literal entry lists of backspace, tab, enter and escape, as well as
// `modifiers`, were generated mechanically from function_keys.zig by a
// throwaway script (every Entry literal gets `mods_empty_is_any: true`, its
// Zig default).
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
func (self InputFunctionKeysCursorMode) String() string {
	return inputFunctionKeysCursorModeNames[self]
}

// inputFunctionKeysCursorModeNames holds the Zig tag names, indexed by value.
var inputFunctionKeysCursorModeNames = [inputFunctionKeysCursorModeCount]string{"any", "normal", "application"}

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
func (self InputFunctionKeysKeypadMode) String() string {
	return inputFunctionKeysKeypadModeNames[self]
}

// inputFunctionKeysKeypadModeNames holds the Zig tag names, indexed by value.
var inputFunctionKeysKeypadModeNames = [inputFunctionKeysKeypadModeCount]string{"any", "normal", "application"}

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
func (self InputFunctionKeysModifyKeys) String() string {
	return inputFunctionKeysModifyKeysNames[self]
}

// inputFunctionKeysModifyKeysNames holds the Zig tag names, indexed by value.
var inputFunctionKeysModifyKeysNames = [inputFunctionKeysModifyKeysCount]string{"any", "set", "set_other"}

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
// Upstream's 15 entries in upstream's order (generated from the Zig source).
var InputFunctionKeysModifiers = []InputKeyMods{
	{shift: true},
	{alt: true},
	{shift: true, alt: true},
	{ctrl: true},
	{shift: true, ctrl: true},
	{alt: true, ctrl: true},
	{shift: true, alt: true, ctrl: true},
	{super: true},
	{shift: true, super: true},
	{alt: true, super: true},
	{shift: true, alt: true, super: true},
	{ctrl: true, super: true},
	{shift: true, ctrl: true, super: true},
	{alt: true, ctrl: true, super: true},
	{shift: true, alt: true, ctrl: true, super: true},
}

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
	var self InputFunctionKeysKeyEntryArray
	for i := range self.values {
		self.values[i] = value
	}
	return self
}

// Get is std.EnumArray.get: the entries of key.
func (self *InputFunctionKeysKeyEntryArray) Get(key InputKeyKey) []InputFunctionKeysEntry {
	return self.values[key]
}

// Set is std.EnumArray.set: replace the entries of key.
func (self *InputFunctionKeysKeyEntryArray) Set(key InputKeyKey, value []InputFunctionKeysEntry) {
	self.values[key] = value
}

// InputFunctionKeysKeys is function_keys.keys: the list of keys that we
// support and their entry values. It is expected that you'll just use a
// for loop through the entry values, there are never more than a couple
// dozen or so.
//
// Built once at start-up by inputFunctionKeysKeysInit, upstream's comptime
// block.
var InputFunctionKeysKeys = inputFunctionKeysKeysInit()

// inputFunctionKeysKeysInit is the comptime block `keys: { ... }` that
// initialises function_keys.keys.
func inputFunctionKeysKeysInit() InputFunctionKeysKeyEntryArray {
	result := InputFunctionKeysKeyEntryArrayInitFill([]InputFunctionKeysEntry{})

	result.Set(InputKeyKeyArrowUp, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[1;{}A")), inputFunctionKeysCursorKey([]uint8("\x1b[A"), []uint8("\x1bOA"))))
	result.Set(InputKeyKeyArrowDown, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[1;{}B")), inputFunctionKeysCursorKey([]uint8("\x1b[B"), []uint8("\x1bOB"))))
	result.Set(InputKeyKeyArrowRight, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[1;{}C")), inputFunctionKeysCursorKey([]uint8("\x1b[C"), []uint8("\x1bOC"))))
	result.Set(InputKeyKeyArrowLeft, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[1;{}D")), inputFunctionKeysCursorKey([]uint8("\x1b[D"), []uint8("\x1bOD"))))
	result.Set(InputKeyKeyHome, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[1;{}H")), inputFunctionKeysCursorKey([]uint8("\x1b[H"), []uint8("\x1bOH"))))
	result.Set(InputKeyKeyEnd, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[1;{}F")), inputFunctionKeysCursorKey([]uint8("\x1b[F"), []uint8("\x1bOF"))))
	result.Set(InputKeyKeyInsert, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[2;{}~")), []InputFunctionKeysEntry{{mods_empty_is_any: true, sequence: []uint8("\x1b[2~")}}))
	result.Set(InputKeyKeyDelete, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[3;{}~")), []InputFunctionKeysEntry{{mods_empty_is_any: true, sequence: []uint8("\x1b[3~")}}))
	result.Set(InputKeyKeyPageUp, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[5;{}~")), []InputFunctionKeysEntry{{mods_empty_is_any: true, sequence: []uint8("\x1b[5~")}}))
	result.Set(InputKeyKeyPageDown, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[6;{}~")), []InputFunctionKeysEntry{{mods_empty_is_any: true, sequence: []uint8("\x1b[6~")}}))

	// Function Keys. todo: f13-f35 but we need to add to input.Key
	result.Set(InputKeyKeyF1, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[1;{}P")), []InputFunctionKeysEntry{{mods_empty_is_any: true, sequence: []uint8("\x1bOP")}}))
	result.Set(InputKeyKeyF2, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[1;{}Q")), []InputFunctionKeysEntry{{mods_empty_is_any: true, sequence: []uint8("\x1bOQ")}}))
	result.Set(InputKeyKeyF3, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[13;{}~")), []InputFunctionKeysEntry{{mods_empty_is_any: true, sequence: []uint8("\x1bOR")}}))
	result.Set(InputKeyKeyF4, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[1;{}S")), []InputFunctionKeysEntry{{mods_empty_is_any: true, sequence: []uint8("\x1bOS")}}))
	result.Set(InputKeyKeyF5, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[15;{}~")), []InputFunctionKeysEntry{{mods_empty_is_any: true, sequence: []uint8("\x1b[15~")}}))
	result.Set(InputKeyKeyF6, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[17;{}~")), []InputFunctionKeysEntry{{mods_empty_is_any: true, sequence: []uint8("\x1b[17~")}}))
	result.Set(InputKeyKeyF7, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[18;{}~")), []InputFunctionKeysEntry{{mods_empty_is_any: true, sequence: []uint8("\x1b[18~")}}))
	result.Set(InputKeyKeyF8, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[19;{}~")), []InputFunctionKeysEntry{{mods_empty_is_any: true, sequence: []uint8("\x1b[19~")}}))
	result.Set(InputKeyKeyF9, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[20;{}~")), []InputFunctionKeysEntry{{mods_empty_is_any: true, sequence: []uint8("\x1b[20~")}}))
	result.Set(InputKeyKeyF10, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[21;{}~")), []InputFunctionKeysEntry{{mods_empty_is_any: true, sequence: []uint8("\x1b[21~")}}))
	result.Set(InputKeyKeyF11, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[23;{}~")), []InputFunctionKeysEntry{{mods_empty_is_any: true, sequence: []uint8("\x1b[23~")}}))
	result.Set(InputKeyKeyF12, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[24;{}~")), []InputFunctionKeysEntry{{mods_empty_is_any: true, sequence: []uint8("\x1b[24~")}}))

	// Keypad keys
	result.Set(InputKeyKeyNumpad0, inputFunctionKeysKpKeys([]uint8("p")))
	result.Set(InputKeyKeyNumpad1, inputFunctionKeysKpKeys([]uint8("q")))
	result.Set(InputKeyKeyNumpad2, inputFunctionKeysKpKeys([]uint8("r")))
	result.Set(InputKeyKeyNumpad3, inputFunctionKeysKpKeys([]uint8("s")))
	result.Set(InputKeyKeyNumpad4, inputFunctionKeysKpKeys([]uint8("t")))
	result.Set(InputKeyKeyNumpad5, inputFunctionKeysKpKeys([]uint8("u")))
	result.Set(InputKeyKeyNumpad6, inputFunctionKeysKpKeys([]uint8("v")))
	result.Set(InputKeyKeyNumpad7, inputFunctionKeysKpKeys([]uint8("w")))
	result.Set(InputKeyKeyNumpad8, inputFunctionKeysKpKeys([]uint8("x")))
	result.Set(InputKeyKeyNumpad9, inputFunctionKeysKpKeys([]uint8("y")))
	result.Set(InputKeyKeyNumpadDecimal, inputFunctionKeysKpKeys([]uint8("n")))
	result.Set(InputKeyKeyNumpadDivide, inputFunctionKeysKpKeys([]uint8("o")))
	result.Set(InputKeyKeyNumpadMultiply, inputFunctionKeysKpKeys([]uint8("j")))
	result.Set(InputKeyKeyNumpadSubtract, inputFunctionKeysKpKeys([]uint8("m")))
	result.Set(InputKeyKeyNumpadAdd, inputFunctionKeysKpKeys([]uint8("k")))
	result.Set(InputKeyKeyNumpadEnter, inputFunctionKeysConcat(inputFunctionKeysKpKeys([]uint8("M")), []InputFunctionKeysEntry{{mods_empty_is_any: true, sequence: []uint8("\r")}}))
	result.Set(InputKeyKeyNumpadUp, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[1;{}A")), inputFunctionKeysCursorKey([]uint8("\x1b[A"), []uint8("\x1bOA"))))
	result.Set(InputKeyKeyNumpadDown, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[1;{}B")), inputFunctionKeysCursorKey([]uint8("\x1b[B"), []uint8("\x1bOB"))))
	result.Set(InputKeyKeyNumpadRight, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[1;{}C")), inputFunctionKeysCursorKey([]uint8("\x1b[C"), []uint8("\x1bOC"))))
	result.Set(InputKeyKeyNumpadLeft, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[1;{}D")), inputFunctionKeysCursorKey([]uint8("\x1b[D"), []uint8("\x1bOD"))))
	result.Set(InputKeyKeyNumpadBegin, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[1;{}E")), inputFunctionKeysCursorKey([]uint8("\x1b[E"), []uint8("\x1bOE"))))
	result.Set(InputKeyKeyNumpadHome, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[1;{}H")), inputFunctionKeysCursorKey([]uint8("\x1b[H"), []uint8("\x1bOH"))))
	result.Set(InputKeyKeyNumpadEnd, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[1;{}F")), inputFunctionKeysCursorKey([]uint8("\x1b[F"), []uint8("\x1bOF"))))
	result.Set(InputKeyKeyNumpadInsert, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[2;{}~")), []InputFunctionKeysEntry{{mods_empty_is_any: true, sequence: []uint8("\x1b[2~")}}))
	result.Set(InputKeyKeyNumpadDelete, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[3;{}~")), []InputFunctionKeysEntry{{mods_empty_is_any: true, sequence: []uint8("\x1b[3~")}}))
	result.Set(InputKeyKeyNumpadPageUp, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[5;{}~")), []InputFunctionKeysEntry{{mods_empty_is_any: true, sequence: []uint8("\x1b[5~")}}))
	result.Set(InputKeyKeyNumpadPageDown, inputFunctionKeysConcat(inputFunctionKeysPcStyle([]uint8("\x1b[6;{}~")), []InputFunctionKeysEntry{{mods_empty_is_any: true, sequence: []uint8("\x1b[6~")}}))

	result.Set(InputKeyKeyBackspace, []InputFunctionKeysEntry{
		// Modify Keys Normal
		{mods: InputKeyMods{shift: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSet, sequence: []uint8("\x7f")},
		{mods: InputKeyMods{alt: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSet, sequence: []uint8("\x1b\x7f")},
		{mods: InputKeyMods{alt: true, shift: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSet, sequence: []uint8("\x1b\x7f")},
		{mods: InputKeyMods{ctrl: true, shift: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSet, sequence: []uint8("\x08")},
		{mods: InputKeyMods{alt: true, ctrl: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSet, sequence: []uint8("\x1b\x08")},
		{mods: InputKeyMods{super: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSet, sequence: []uint8("\x7f")},
		{mods: InputKeyMods{super: true, shift: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSet, sequence: []uint8("\x7f")},
		{mods: InputKeyMods{alt: true, super: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSet, sequence: []uint8("\x1b\x7f")},
		{mods: InputKeyMods{alt: true, super: true, shift: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSet, sequence: []uint8("\x1b\x7f")},
		{mods: InputKeyMods{super: true, ctrl: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSet, sequence: []uint8("\x08")},
		{mods: InputKeyMods{super: true, ctrl: true, shift: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSet, sequence: []uint8("\x08")},
		{mods: InputKeyMods{alt: true, super: true, ctrl: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSet, sequence: []uint8("\x1b\x08")},
		{mods: InputKeyMods{alt: true, super: true, ctrl: true, shift: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSet, sequence: []uint8("\x1b\x08")},

		// Modify Keys Other
		{mods: InputKeyMods{shift: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSetOther, sequence: []uint8("\x1b[27;2;127~")},
		{mods: InputKeyMods{alt: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSetOther, sequence: []uint8("\x1b[27;3;127~")},
		{mods: InputKeyMods{alt: true, shift: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSetOther, sequence: []uint8("\x1b[27;4;127~")},
		{mods: InputKeyMods{ctrl: true, shift: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSetOther, sequence: []uint8("\x1b[27;6;127~")},
		{mods: InputKeyMods{alt: true, ctrl: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSetOther, sequence: []uint8("\x1b[27;7;127~")},
		{mods: InputKeyMods{alt: true, shift: true, ctrl: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSetOther, sequence: []uint8("\x1b[27;8;127~")},
		{mods: InputKeyMods{super: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSetOther, sequence: []uint8("\x1b[27;9;127~")},
		{mods: InputKeyMods{super: true, shift: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSetOther, sequence: []uint8("\x1b[27;10;127~")},
		{mods: InputKeyMods{alt: true, super: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSetOther, sequence: []uint8("\x1b[27;11;127~")},
		{mods: InputKeyMods{alt: true, super: true, shift: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSetOther, sequence: []uint8("\x1b[27;12;127~")},
		{mods: InputKeyMods{super: true, ctrl: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSetOther, sequence: []uint8("\x1b[27;13;127~")},
		{mods: InputKeyMods{super: true, ctrl: true, shift: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSetOther, sequence: []uint8("\x1b[27;14;127~")},
		{mods: InputKeyMods{alt: true, super: true, ctrl: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSetOther, sequence: []uint8("\x1b[27;15;127~")},
		{mods: InputKeyMods{alt: true, super: true, ctrl: true, shift: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSetOther, sequence: []uint8("\x1b[27;16;127~")},

		{mods: InputKeyMods{ctrl: true}, mods_empty_is_any: true, sequence: []uint8("\x08")},
		{mods_empty_is_any: true, sequence: []uint8("\x7f")},
	})

	result.Set(InputKeyKeyTab, []InputFunctionKeysEntry{
		// Modify Keys Normal
		{mods: InputKeyMods{shift: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSet, sequence: []uint8("\x1b[Z")},
		{mods: InputKeyMods{alt: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSet, sequence: []uint8("\x1b\t")},

		// Modify Keys Other
		{mods: InputKeyMods{shift: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSetOther, sequence: []uint8("\x1b[27;2;9~")},
		{mods: InputKeyMods{alt: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSetOther, sequence: []uint8("\x1b[27;3;9~")},

		// Everything else
		{mods: InputKeyMods{alt: true, shift: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;4;9~")},
		{mods: InputKeyMods{ctrl: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;5;9~")},
		{mods: InputKeyMods{ctrl: true, shift: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;6;9~")},
		{mods: InputKeyMods{alt: true, ctrl: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;7;9~")},
		{mods: InputKeyMods{alt: true, ctrl: true, shift: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;8;9~")},
		{mods: InputKeyMods{super: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;9;9~")},
		{mods: InputKeyMods{super: true, shift: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;10;9~")},
		{mods: InputKeyMods{alt: true, super: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;11;9~")},
		{mods: InputKeyMods{alt: true, super: true, shift: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;12;9~")},
		{mods: InputKeyMods{super: true, ctrl: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;13;9~")},
		{mods: InputKeyMods{super: true, ctrl: true, shift: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;14;9~")},
		{mods: InputKeyMods{alt: true, super: true, ctrl: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;15;9~")},
		{mods: InputKeyMods{alt: true, super: true, ctrl: true, shift: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;16;9~")},

		{mods_empty_is_any: true, sequence: []uint8("\t")},
	})

	result.Set(InputKeyKeyEnter, []InputFunctionKeysEntry{
		{mods: InputKeyMods{shift: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;2;13~")},

		// Modify Keys Normal
		{mods: InputKeyMods{alt: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSet, sequence: []uint8("\x1b\r")},

		// Modify Keys Other
		{mods: InputKeyMods{alt: true}, mods_empty_is_any: true, modify_other_keys: InputFunctionKeysModifyKeysSetOther, sequence: []uint8("\x1b[27;3;13~")},

		// Everything else
		{mods: InputKeyMods{alt: true, shift: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;4;13~")},
		{mods: InputKeyMods{ctrl: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;5;13~")},
		{mods: InputKeyMods{ctrl: true, shift: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;6;13~")},
		{mods: InputKeyMods{alt: true, ctrl: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;7;13~")},
		{mods: InputKeyMods{alt: true, ctrl: true, shift: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;8;13~")},
		{mods: InputKeyMods{super: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;9;13~")},
		{mods: InputKeyMods{super: true, shift: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;10;13~")},
		{mods: InputKeyMods{alt: true, super: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;11;13~")},
		{mods: InputKeyMods{alt: true, super: true, shift: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;12;13~")},
		{mods: InputKeyMods{super: true, ctrl: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;13;13~")},
		{mods: InputKeyMods{super: true, ctrl: true, shift: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;14;13~")},
		{mods: InputKeyMods{alt: true, super: true, ctrl: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;15;13~")},
		{mods: InputKeyMods{alt: true, super: true, ctrl: true, shift: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;16;13~")},

		{mods_empty_is_any: true, sequence: []uint8("\r")},
	})

	result.Set(InputKeyKeyEscape, []InputFunctionKeysEntry{
		{mods: InputKeyMods{shift: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;2;27~")},
		{mods: InputKeyMods{alt: true}, mods_empty_is_any: true, sequence: []uint8("\x1b\x1b")},
		{mods: InputKeyMods{alt: true, shift: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;4;27~")},
		{mods: InputKeyMods{ctrl: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;5;27~")},
		{mods: InputKeyMods{ctrl: true, shift: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;6;27~")},
		{mods: InputKeyMods{alt: true, ctrl: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;7;27~")},
		{mods: InputKeyMods{alt: true, ctrl: true, shift: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;8;27~")},
		{mods: InputKeyMods{super: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;9;27~")},
		{mods: InputKeyMods{super: true, shift: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;10;27~")},
		{mods: InputKeyMods{alt: true, super: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;11;27~")},
		{mods: InputKeyMods{alt: true, super: true, shift: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;12;27~")},
		{mods: InputKeyMods{super: true, ctrl: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;13;27~")},
		{mods: InputKeyMods{super: true, ctrl: true, shift: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;14;27~")},
		{mods: InputKeyMods{alt: true, super: true, ctrl: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;15;27~")},
		{mods: InputKeyMods{alt: true, super: true, ctrl: true, shift: true}, mods_empty_is_any: true, sequence: []uint8("\x1b[27;16;27~")},
		{mods_empty_is_any: true, sequence: []uint8("\x1b")},
	})

	return result
}

// inputFunctionKeysKpDefault is function_keys.kpDefault: returns the
// default keypad application mode entry.
//
// Zig: fn kpDefault(comptime suffix: []const u8) []const Entry.
func inputFunctionKeysKpDefault(suffix []uint8) []InputFunctionKeysEntry {
	return []InputFunctionKeysEntry{
		{
			mods_empty_is_any: false,
			keypad:            InputFunctionKeysKeypadModeApplication,
			// Zig: "\x1bO" ++ suffix.
			sequence: append([]uint8("\x1bO"), suffix...),
		},
	}
}

// inputFunctionKeysKpKeys is function_keys.kpKeys: returns the entries for
// a keypad key. The suffix is the final character of the sent sequence,
// such as "r" for kp_2.
//
// Zig: fn kpKeys(comptime suffix: []const u8) []const Entry.
func inputFunctionKeysKpKeys(suffix []uint8) []InputFunctionKeysEntry {
	// Zig: pcStyle("\x1bO{}" ++ suffix).
	pc := inputFunctionKeysPcStyle(append([]uint8("\x1bO{}"), suffix...))
	for i := range pc {
		pc[i].keypad = InputFunctionKeysKeypadModeApplication
	}
	return inputFunctionKeysConcat(inputFunctionKeysKpDefault(suffix), pc)
}

// inputFunctionKeysCursorKey is function_keys.cursorKey: returns entries
// that are dependent on cursor key settings.
//
// Zig: fn cursorKey(comptime normal: []const u8, comptime application:
// []const u8) []const Entry.
func inputFunctionKeysCursorKey(normal []uint8, application []uint8) []InputFunctionKeysEntry {
	return []InputFunctionKeysEntry{
		{mods_empty_is_any: true, cursor: InputFunctionKeysCursorModeNormal, sequence: normal},
		{mods_empty_is_any: true, cursor: InputFunctionKeysCursorModeApplication, sequence: application},
	}
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
	// The comptime {} wrapper is superfluous but it prevents us from
	// accidentally running this function at runtime.
	entries := make([]InputFunctionKeysEntry, len(InputFunctionKeysModifiers))
	for i, mods := range InputFunctionKeysModifiers {
		code := uint(2 + i)
		entries[i] = InputFunctionKeysEntry{
			mods:              mods,
			mods_empty_is_any: true,
			sequence:          inputFunctionKeysComptimePrint(fmt, code),
		}
	}

	return entries
}

// inputFunctionKeysConcat is Zig's `++` on the comptime entry arrays of
// the `keys` block and of kpKeys: a new slice with the parts in order. Not
// a Zig declaration.
func inputFunctionKeysConcat(parts ...[]InputFunctionKeysEntry) []InputFunctionKeysEntry {
	var result []InputFunctionKeysEntry
	for _, part := range parts {
		result = append(result, part...)
	}
	return result
}

// inputFunctionKeysComptimePrint is std.fmt.comptimePrint(fmt, .{code})
// with one usize argument, as pcStyle calls it. Not a Ghostty declaration:
// it follows the format-string scan of Zig 0.15.2's std.Io.Writer.print.
// Literal runs are copied, "{{" and "}}" are single braces, and the empty
// placeholder "{}" prints code in decimal. Zig rejects every other format
// at compile time ("missing opening {", "missing closing }", too few
// arguments, unused argument); those panic here, as does any placeholder
// with a specifier, which pcStyle's formats never have.
func inputFunctionKeysComptimePrint(fmt []uint8, code uint) []uint8 {
	var out []uint8
	used := false
	i := 0
	for {
		start_index := i

		for i < len(fmt) {
			if fmt[i] == '{' || fmt[i] == '}' {
				break
			}
			i += 1
		}

		end_index := i
		unescape_brace := false

		// Handle {{ and }}, those are un-escaped as single braces
		if i+1 < len(fmt) && fmt[i+1] == fmt[i] {
			unescape_brace = true
			// Make the first brace part of the literal...
			end_index += 1
			// ...and skip both
			i += 2
		}

		out = append(out, fmt[start_index:end_index]...)

		// We've already skipped the other brace, restart the loop
		if unescape_brace {
			continue
		}

		if i >= len(fmt) {
			break
		}

		if fmt[i] == '}' {
			panic("comptimePrint: missing opening {")
		}

		// Get past the {
		i += 1

		fmt_begin := i
		// Find the closing brace
		for i < len(fmt) && fmt[i] != '}' {
			i += 1
		}
		fmt_end := i

		if i >= len(fmt) {
			panic("comptimePrint: missing closing }")
		}

		// Get past the }
		i += 1

		if fmt_end != fmt_begin {
			panic("not ported: comptimePrint placeholder other than {}")
		}
		if used {
			panic("comptimePrint: too few arguments")
		}
		used = true
		out = append(out, itoa(uint64(code))...)
	}
	if !used {
		panic("comptimePrint: unused argument")
	}
	return out
}
