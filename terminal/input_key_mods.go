// WIREFRAME for src/input/key_mods.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented) and every table is left at its zero value. The
// implementation is added only after the project owner and the project lead
// both approve it. Do not fill in bodies before then. See the header of
// color.go for the rules.
//
// File naming: key_mods.zig lives in src/input/, so the file is
// `input_key_mods.go` and every declaration takes the prefix `InputKeyMods`
// (`inputKeyMods` for private ones): `key_mods.Mods` is `InputKeyModsMods`,
// `key_mods.Mod` is `InputKeyModsMod`, `key_mods.Mod.Side` is
// `InputKeyModsModSide` (the left/right enum), `key_mods.Mods.Side` is
// `InputKeyModsModsSide` (the packed struct of four of those),
// `key_mods.Mods.Keys` is `InputKeyModsModsKeys`, `key_mods.alias` is
// `InputKeyModsAlias`, `key_mods.RemapSet` is `InputKeyModsRemapSet`,
// `RemapSet.Mask` is `InputKeyModsRemapSetMask`. Constants inside types
// take the type's name: `Mods.side_mask` is `InputKeyModsModsSideMask`,
// `RemapSet.empty` is `InputKeyModsRemapSetEmpty`, the `Backing` types are
// `InputKeyModsModsBacking`, `InputKeyModsModsKeysBacking`,
// `InputKeyModsModsSideBacking`. `pub fn` methods are exported (`Int`,
// `Empty`, `Binding`, `Parse`, `Apply`, `Update`, `Match`); the functions
// of RemapSet that have no `self` are free functions with the type's name
// as prefix: the private `parseMod`, `initMods`, `formatMod` are
// `inputKeyModsRemapSetParseMod`, `inputKeyModsRemapSetInitMods`,
// `inputKeyModsRemapSetFormatMod`.
//
// Declarations appear in the same order as in key_mods.zig.
//
// # Scope
//
// libghostty-vt uses `Mods` (through key.zig and the encoders). `RemapSet`
// exists for the app's `key-remap` config and is not used by any encoder;
// it is declared in full because the project's tests (patch 0017, 64 tests
// "term51: RemapSet...") cover every function of it.
//
// # Packed structs
//
// `Mods` (packed struct(u16)), `Mods.Keys` (u4), `Mods.Side` (u4),
// `RemapSet.Mask` (u12) are ordinary Go structs with the same fields in the
// same order, comparable with `==` as `expectEqual` compares them. The
// packed-integer conversions that key_mods.zig writes as `@bitCast` or
// `.int()` are stubs: `Int` on each (Zig's own `int` for Mods and
// Mods.Keys; a stand-in for `@bitCast` for Mods.Side and Mask) and
// `...FromInt` for Mods, Mods.Keys and Mods.Side (used by `unset`, `apply`
// and `Mask.update`). The bit layout is Zig's: first field in bit 0.
// `Mods._padding` (u6, bits 10 to 15) is kept as a field: Mods.int and
// therefore `empty` and `equal` see it.
//
// # Go-memory rule
//
//   - Every allocator parameter is dropped (`parseCLI`, `parse`, `deinit`,
//     `clone`). `Deinit` is kept as a stub because the tests call it.
//   - `RemapSet.map` (field `map_`: `map` is a Go keyword) is
//     `std.AutoArrayHashMapUnmanaged(Mods, Mods)`, an insertion-ordered
//     map: `inputKeyModsRemapSetMap`, two parallel slices
//     with stubs for the operations key_mods.zig and the tests use, under
//     their std names (as `pageListPinSet` in PageList.go): `Put`, `Get`,
//     `SwapRemove`, `ClearRetainingCapacity`, `Keys`, `Values`, `Count`,
//     `Clone`, `Sort`. Not declared: `deinit` (only frees) and `iterator`
//     (the Go code ranges over `Keys()` and `Values()`).
//   - `finalize` sorts with a context whose `lessThan` ignores `b_index`.
//     That is not a strict ordering, so the result depends on the exact
//     algorithm: Zig's ArrayHashMap.sort is MultiArrayList.sort, i.e.
//     `std.sort.insertionContext` (stable insertion sort). The Go `Sort`
//     must perform the same comparisons and swaps in the same order, not
//     `sort.SliceStable`.
//
// # anytype parameters
//
//   - `formatEntry(formatter: anytype)`: the formatter is anything with
//     `formatEntry(comptime T: type, value: T) !void`, called with
//     `(void, {})` and `([]const u8, bytes)`. Go: the interface
//     `inputKeyModsRemapSetEntryFormatter` with
//     `formatEntry(value Optional[[]uint8]) error`; a null value is the
//     `void` call. The test's recording and failing formatters implement
//     it; `config/formatter.zig`'s `entryFormatter` is app code with no Go
//     counterpart.
//   - `formatMod(writer: anytype, ...)` is only ever given a
//     `std.io.fixedBufferStream` writer (by formatEntry and by the tests).
//     Go: `*inputKeyModsFixedBufferStream`, a stand-in for
//     std.io.FixedBufferStream([]u8) declared here (its first user), with
//     `inputKeyModsFixedBufferStreamInit` (std.io.fixedBufferStream) and
//     `getWritten`. A write that does not fit fails with ErrNoSpaceLeft
//     (color.go), as in Zig. The implementation adds `writeAll` and
//     `writeByte` with Zig's names.
//
// # Errors
//
// `RemapSet.ParseError` is `Allocator.Error || error{ MissingAssignment,
// InvalidMod }`; Go has no error-set types, so it is the sentinels
// ErrOutOfMemory (dcs.go), ErrMissingAssignment and ErrInvalidMod (declared
// here). `parseCLI` maps the last two to ErrInvalidValue
// (osc_parsers_kitty_text_sizing.go).
//
// # Darwin
//
// `Mods.translation` and `Mods.ctrlOrSuper` have a comptime branch on
// `builtin.target.os.tag.isDarwin()`. The port runs on Linux and QNX and so
// follows the non-Darwin branch. The check is `inputKeyModsIsDarwin()`, a
// stub standing for the comptime test; Darwin behaviour would be selected
// by making it return true (runtime.GOOS "darwin" or "ios"). With it false,
// `translation` returns the mods unchanged and `ctrlOrSuper` returns
// `ctrl`.
//
// # Declarations that are not Zig declarations
//
//   - `InputKeyModsAliasEntry`: the anonymous tuple struct
//     `struct { []const u8, Mod }` of `alias`; tuple fields "0" and "1"
//     begin with a digit and so take the prefix `x` (as `x132_column` in
//     modes.go): `x0`, `x1`;
//   - `inputKeyModsModCount`, `inputKeyModsModSideCount` and the `String`
//     methods;
//   - the `...FromInt` functions and the `Int` methods of Mods.Side and
//     Mask (stand-ins for `@bitCast`);
//   - `inputKeyModsRemapSetMap`, `inputKeyModsRemapSetEntryFormatter`,
//     `inputKeyModsFixedBufferStream` (std stand-ins, above);
//   - `inputKeyModsIsDarwin` (above);
//   - ErrMissingAssignment, ErrInvalidMod.
//
// # Omitted
//
//   - the nested tests' comptime layout assertions (none outside tests);
//   - `finalize`'s local `Context` struct: the Go `Sort` takes the
//     `lessThan` function directly;
//   - aliases and imports with nothing to declare: `std`, `assert`,
//     `Allocator`, `builtin`, `OptionAsAlt` (use InputConfigOptionAsAlt).
// No C ABI declaration exists in this file ("IMPORTANT: Any changes here
// update include/ghostty.h" concerns the C header only).
//
// # Zig defaults a Go literal does not give
//
// None. Every field default in this file is the Go zero value: Mods (all
// false, `sides` all `.left` = 0, `_padding` 0), Mods.Keys (all false),
// Mods.Side (all `.left` = 0), RemapSet.Mask (all zero). RemapSet has no
// field defaults; `RemapSet.empty` is the zero value.

package terminal

import "errors"

// ErrMissingAssignment is Zig's error.MissingAssignment
// (key_mods.RemapSet.ParseError).
var ErrMissingAssignment = errors.New("MissingAssignment")

// ErrInvalidMod is Zig's error.InvalidMod (key_mods.RemapSet.ParseError,
// RemapSet.parseMod).
var ErrInvalidMod = errors.New("InvalidMod")

// InputKeyModsAliasEntry is the element type of key_mods.alias:
// `struct { []const u8, Mod }`. Not a Zig declaration (an anonymous tuple).
type InputKeyModsAliasEntry struct {
	x0 []uint8
	x1 InputKeyModsMod
}

// InputKeyModsAlias is key_mods.alias: aliases for modifier names.
//
// Wireframe: left nil; the implementation fills it with upstream's five
// entries in upstream's order: "cmd" super, "command" super, "opt" alt,
// "option" alt, "control" ctrl.
var InputKeyModsAlias []InputKeyModsAliasEntry

// InputKeyModsMod is key_mods.Mod: single modifier.
//
// Zig: enum, values numbered from zero in the order listed.
type InputKeyModsMod uint8

const (
	InputKeyModsModShift InputKeyModsMod = iota
	InputKeyModsModCtrl
	InputKeyModsModAlt
	InputKeyModsModSuper

	inputKeyModsModCount = iota
)

// String returns the Zig tag name (@tagName); also the name
// std.meta.stringToEnum matches in RemapSet.parseMod.
func (self InputKeyModsMod) String() string { panic(notImplemented) }

// InputKeyModsModSide is key_mods.Mod.Side.
//
// Zig: enum(u1) { left, right }.
type InputKeyModsModSide uint8

const (
	InputKeyModsModSideLeft InputKeyModsModSide = iota
	InputKeyModsModSideRight

	inputKeyModsModSideCount = iota
)

// String returns the Zig tag name (@tagName); also the name
// std.meta.stringToEnum matches in RemapSet.parseMod.
func (self InputKeyModsModSide) String() string { panic(notImplemented) }

// InputKeyModsMods is key_mods.Mods: a bitmask for all key modifiers.
//
// IMPORTANT: Any changes here update include/ghostty.h
//
// Zig: packed struct(u16), shift in bit 0 through num_lock in bit 5, sides
// in bits 6 to 9, _padding in bits 10 to 15. Every field defaults to the Go
// zero value. See the file header for the conversions.
type InputKeyModsMods struct {
	shift     bool
	ctrl      bool
	alt       bool
	super     bool
	caps_lock bool
	num_lock  bool
	sides     InputKeyModsModsSide
	// Zig: u6, default 0.
	_padding uint8
}

// InputKeyModsModsBacking is key_mods.Mods.Backing.
type InputKeyModsModsBacking = uint16

// InputKeyModsModsKeys is key_mods.Mods.Keys: the standard modifier keys
// only. Does not include the lock keys, only standard bindable keys.
//
// Zig: packed struct(u4), every field default false.
type InputKeyModsModsKeys struct {
	shift bool
	ctrl  bool
	alt   bool
	super bool
}

// InputKeyModsModsKeysBacking is key_mods.Mods.Keys.Backing (u4).
type InputKeyModsModsKeysBacking = uint8

// Int is key_mods.Mods.Keys.int: the packed integer.
//
// Zig: pub inline fn int(self: Keys) Keys.Backing.
func (self InputKeyModsModsKeys) Int() InputKeyModsModsKeysBacking { panic(notImplemented) }

// InputKeyModsModsKeysFromInt stands for Zig's `@bitCast` from Keys.Backing
// to Keys (Mods.keys, RemapSet.Mask.update). Only the low four bits are
// meaningful.
func InputKeyModsModsKeysFromInt(v InputKeyModsModsKeysBacking) InputKeyModsModsKeys {
	panic(notImplemented)
}

// InputKeyModsModsSide is key_mods.Mods.Side: tracks the side that is
// active for any given modifier. Note that this doesn't confirm a modifier
// is pressed; you must check the bool for that in addition to this.
//
// Not all platforms support this, check apprt for more info.
//
// Zig: packed struct(u4), every field default .left (the Go zero value).
type InputKeyModsModsSide struct {
	shift InputKeyModsModSide
	ctrl  InputKeyModsModSide
	alt   InputKeyModsModSide
	super InputKeyModsModSide
}

// InputKeyModsModsSideBacking is key_mods.Mods.Side.Backing (u4).
type InputKeyModsModsSideBacking = uint8

// Int stands for Zig's `@bitCast` from Side to Side.Backing (RemapSet.apply,
// RemapSet.Mask.update and match): shift in bit 0 through super in bit 3,
// .right is 1. Zig has no named function for it.
func (self InputKeyModsModsSide) Int() InputKeyModsModsSideBacking { panic(notImplemented) }

// InputKeyModsModsSideFromInt stands for Zig's `@bitCast` from Side.Backing
// to Side (RemapSet.Mask.update), the inverse of Int.
func InputKeyModsModsSideFromInt(v InputKeyModsModsSideBacking) InputKeyModsModsSide {
	panic(notImplemented)
}

// InputKeyModsModsSideMask is key_mods.Mods.side_mask: the mask that has all
// the side bits set.
//
// A constant value, so it is given here; it is not logic.
var InputKeyModsModsSideMask = InputKeyModsMods{
	sides: InputKeyModsModsSide{
		shift: InputKeyModsModSideRight,
		ctrl:  InputKeyModsModSideRight,
		alt:   InputKeyModsModSideRight,
		super: InputKeyModsModSideRight,
	},
}

// Int is key_mods.Mods.int: integer value of this struct.
func (self InputKeyModsMods) Int() InputKeyModsModsBacking { panic(notImplemented) }

// InputKeyModsModsFromInt stands for Zig's `@bitCast` from Mods.Backing to
// Mods (Mods.unset, RemapSet.apply), the inverse of Int. Every bit is kept,
// including _padding.
func InputKeyModsModsFromInt(v InputKeyModsModsBacking) InputKeyModsMods {
	panic(notImplemented)
}

// Empty is key_mods.Mods.empty: returns true if no modifiers are set.
func (self InputKeyModsMods) Empty() bool { panic(notImplemented) }

// Equal is key_mods.Mods.equal: returns true if two mods are equal.
func (self InputKeyModsMods) Equal(other InputKeyModsMods) bool { panic(notImplemented) }

// Keys is key_mods.Mods.keys: returns only the keys.
//
// In the future I want to remove `binding` for this. I didn't want
// to do that all in one PR where I added this because its a bigger
// change.
func (self InputKeyModsMods) Keys() InputKeyModsModsKeys { panic(notImplemented) }

// Binding is key_mods.Mods.binding: return mods that are only relevant for
// bindings.
func (self InputKeyModsMods) Binding() InputKeyModsMods { panic(notImplemented) }

// Unset is key_mods.Mods.unset: perform `self &~ other` to remove the other
// mods from self.
func (self InputKeyModsMods) Unset(other InputKeyModsMods) InputKeyModsMods {
	panic(notImplemented)
}

// WithoutLocks is key_mods.Mods.withoutLocks: returns the mods without
// locks set.
func (self InputKeyModsMods) WithoutLocks() InputKeyModsMods { panic(notImplemented) }

// Translation is key_mods.Mods.translation: return the mods to use for key
// translation. This handles settings like macos-option-as-alt. The
// translation mods should be used for translation but never sent back in
// for the key callback.
//
// Darwin: the alt handling is inside `if (comptime isDarwin())`; see the
// file header. Off Darwin the result is self.
func (self InputKeyModsMods) Translation(option_as_alt InputConfigOptionAsAlt) InputKeyModsMods {
	panic(notImplemented)
}

// CtrlOrSuper is key_mods.Mods.ctrlOrSuper: checks to see if super is on
// (MacOS) or ctrl.
//
// Darwin: see the file header. Off Darwin the result is self.ctrl.
func (self InputKeyModsMods) CtrlOrSuper() bool { panic(notImplemented) }

// inputKeyModsIsDarwin stands for the comptime check
// `builtin.target.os.tag.isDarwin()` in Mods.translation and
// Mods.ctrlOrSuper. The port follows the non-Darwin branch (Linux, QNX);
// see the file header.
func inputKeyModsIsDarwin() bool { panic(notImplemented) }

// InputKeyModsRemapSet is key_mods.RemapSet: modifier remapping. See
// `key-remap` in Config.zig for detailed docs.
type InputKeyModsRemapSet struct {
	// Available mappings.
	//
	// Zig: std.AutoArrayHashMapUnmanaged(Mods, Mods). See the file header.
	map_ inputKeyModsRemapSetMap

	// The mask of remapped modifiers that can be used to quickly
	// check if some input mods need remapping.
	mask InputKeyModsRemapSetMask
}

// inputKeyModsRemapSetMap stands for std.AutoArrayHashMapUnmanaged(Mods,
// Mods): an insertion-ordered map. Not a Zig declaration. See the file
// header.
type inputKeyModsRemapSetMap struct {
	// The keys in order; what `keys()` returns.
	keys []InputKeyModsMods
	// The values, parallel to keys; what `values()` returns.
	values []InputKeyModsMods
}

// Put is std's put: insert or replace the value for key. A new key goes at
// the end; an existing key keeps its position.
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go).
func (self *inputKeyModsRemapSetMap) Put(key InputKeyModsMods, value InputKeyModsMods) error {
	panic(notImplemented)
}

// Get is std's get: the value for key, or null.
func (self *inputKeyModsRemapSetMap) Get(key InputKeyModsMods) Optional[InputKeyModsMods] {
	panic(notImplemented)
}

// SwapRemove is std's swapRemove: if key is present, remove it by moving the
// last entry into its place and return true; otherwise return false.
func (self *inputKeyModsRemapSetMap) SwapRemove(key InputKeyModsMods) bool {
	panic(notImplemented)
}

// ClearRetainingCapacity is std's clearRetainingCapacity: remove every
// entry.
func (self *inputKeyModsRemapSetMap) ClearRetainingCapacity() { panic(notImplemented) }

// Keys is std's keys: the keys in order. The slice is the map's own and is
// only valid until the map is modified.
func (self *inputKeyModsRemapSetMap) Keys() []InputKeyModsMods { panic(notImplemented) }

// Values is std's values: the values, parallel to Keys. The slice is the
// map's own and is only valid until the map is modified.
func (self *inputKeyModsRemapSetMap) Values() []InputKeyModsMods { panic(notImplemented) }

// Count is std's count: the number of entries.
func (self *inputKeyModsRemapSetMap) Count() uint { panic(notImplemented) }

// Clone is std's clone: an independent copy with the same entries in the
// same order. The allocator parameter is dropped.
//
// Zig: Allocator.Error!Self. The error is ErrOutOfMemory (dcs.go).
func (self *inputKeyModsRemapSetMap) Clone() (inputKeyModsRemapSetMap, error) {
	panic(notImplemented)
}

// Sort is std's sort (ArrayHashMap.sort, stable): reorder the entries with
// `lessThan(a_index, b_index)`, the `lessThan` of Zig's sort context. Must
// be std.sort.insertionContext step for step; see the file header.
func (self *inputKeyModsRemapSetMap) Sort(lessThan func(a_index uint, b_index uint) bool) {
	panic(notImplemented)
}

// InputKeyModsRemapSetEmpty is key_mods.RemapSet.empty. The zero value is
// that state.
var InputKeyModsRemapSetEmpty = InputKeyModsRemapSet{}

// ParseCLI is key_mods.RemapSet.parseCLI: parse from CLI input. Required by
// Config.
//
// Zig: pub fn parseCLI(self: *RemapSet, alloc: Allocator, input: ?[]const
// u8) !void. The allocator is dropped. The inferred error set is
// ErrOutOfMemory or ErrInvalidValue.
func (self *InputKeyModsRemapSet) ParseCLI(input Optional[[]uint8]) error {
	panic(notImplemented)
}

// Parse is key_mods.RemapSet.parse: parse a modifier remap and add it to
// the set.
//
// Zig: ParseError!void. The allocator is dropped. The errors are
// ErrOutOfMemory, ErrMissingAssignment and ErrInvalidMod.
func (self *InputKeyModsRemapSet) Parse(input []uint8) error { panic(notImplemented) }

// Deinit is key_mods.RemapSet.deinit. It only frees in Zig; the stub is
// kept because every test calls it.
func (self *InputKeyModsRemapSet) Deinit() { panic(notImplemented) }

// Finalize is key_mods.RemapSet.finalize: must be called prior to any
// remappings so that the mapping is sorted properly. Otherwise, you will
// get invalid results.
func (self *InputKeyModsRemapSet) Finalize() { panic(notImplemented) }

// Clone is key_mods.RemapSet.clone: deep copy of the struct. Required by
// Config.
//
// Zig: Allocator.Error!RemapSet. The allocator is dropped; the error is
// ErrOutOfMemory (dcs.go).
func (self *InputKeyModsRemapSet) Clone() (InputKeyModsRemapSet, error) { panic(notImplemented) }

// Equal is key_mods.RemapSet.equal: compare if two RemapSets are equal.
// Required by Config.
func (self InputKeyModsRemapSet) Equal(other InputKeyModsRemapSet) bool { panic(notImplemented) }

// inputKeyModsRemapSetEntryFormatter stands for the `formatter: anytype`
// of RemapSet.formatEntry: anything with Zig's
// `formatEntry(comptime T: type, value: T) !void`. Not a Zig declaration.
// formatEntry calls it with `(void, {})`, which is a null value here, and
// with `([]const u8, bytes)`, which is a valid value.
type inputKeyModsRemapSetEntryFormatter interface {
	formatEntry(value Optional[[]uint8]) error
}

// FormatEntry is key_mods.RemapSet.formatEntry: used by Formatter. Required
// by Config.
//
// Zig: pub fn formatEntry(self: RemapSet, formatter: anytype) !void. The
// inferred error set is the formatter's errors plus ErrOutOfMemory (a
// formatMod or writeByte failure into the 64-byte buffer is reported as
// OutOfMemory).
func (self InputKeyModsRemapSet) FormatEntry(formatter inputKeyModsRemapSetEntryFormatter) error {
	panic(notImplemented)
}

// inputKeyModsFixedBufferStream stands for std.io.FixedBufferStream([]u8):
// a buffer and a write position. Not a Zig declaration. Zig's
// `fbs.writer()` is the stream itself here; a write that does not fit
// fails with ErrNoSpaceLeft (color.go). Wireframe: fields, Init and
// getWritten; the implementation adds writeAll and writeByte.
type inputKeyModsFixedBufferStream struct {
	buffer []uint8
	// Zig: pos: usize.
	pos uint
}

// inputKeyModsFixedBufferStreamInit is std.io.fixedBufferStream(buf).
func inputKeyModsFixedBufferStreamInit(buf []uint8) inputKeyModsFixedBufferStream {
	panic(notImplemented)
}

// getWritten is std.io.FixedBufferStream.getWritten: buffer[:pos].
func (self *inputKeyModsFixedBufferStream) getWritten() []uint8 { panic(notImplemented) }

// inputKeyModsRemapSetFormatMod is key_mods.RemapSet.formatMod.
//
// Zig: fn formatMod(writer: anytype, mods: Mods) !void. The writer is
// always a fixedBufferStream writer; see the file header. The error is
// ErrNoSpaceLeft.
func inputKeyModsRemapSetFormatMod(writer *inputKeyModsFixedBufferStream, mods InputKeyModsMods) error {
	panic(notImplemented)
}

// inputKeyModsRemapSetParseMod is key_mods.RemapSet.parseMod: parses a
// single mode in a single remapping string. E.g. `ctrl` or `left_shift`.
//
// Zig: error{InvalidMod}!struct { Mod, ?Mod.Side }. The error is
// ErrInvalidMod.
func inputKeyModsRemapSetParseMod(input []uint8) (InputKeyModsMod, Optional[InputKeyModsModSide], error) {
	panic(notImplemented)
}

// inputKeyModsRemapSetInitMods is key_mods.RemapSet.initMods.
func inputKeyModsRemapSetInitMods(mod InputKeyModsMod, side InputKeyModsModSide) InputKeyModsMods {
	panic(notImplemented)
}

// IsRemapped is key_mods.RemapSet.isRemapped: returns true if the given
// mods need remapping.
func (self *InputKeyModsRemapSet) IsRemapped(mods InputKeyModsMods) bool { panic(notImplemented) }

// Apply is key_mods.RemapSet.apply: apply a remap to the given mods.
//
// Zig ends with `unreachable` when the mask matched but no entry did; the
// port panics there.
func (self *InputKeyModsRemapSet) Apply(mods InputKeyModsMods) InputKeyModsMods {
	panic(notImplemented)
}

// InputKeyModsRemapSetMask is key_mods.RemapSet.Mask: tracks which modifier
// keys and sides have remappings registered. Used as a fast pre-check
// before doing expensive map lookups.
//
// The mask uses separate tracking for left and right sides because
// remappings can be side-specific (e.g., only remap left_ctrl).
//
// Note: `left_sides` uses inverted logic where 1 means "left is remapped"
// even though `Mod.Side.left = 0`. This allows efficient bitwise matching
// since we can AND directly with the side bits.
//
// Zig: packed struct(u12), keys in bits 0 to 3, left_sides 4 to 7,
// right_sides 8 to 11. Every field default is the Go zero value.
type InputKeyModsRemapSetMask struct {
	// Which modifier keys (shift/ctrl/alt/super) have any remapping.
	keys InputKeyModsModsKeys
	// Which modifiers have left-side remappings (inverted: 1 = left remapped).
	left_sides InputKeyModsModsSide
	// Which modifiers have right-side remappings (1 = right remapped).
	right_sides InputKeyModsModsSide
}

// Int stands for Zig's `@bitCast` from Mask to u12. Zig has no named
// function for it and key_mods.zig does not cast a Mask; the tests do.
func (self InputKeyModsRemapSetMask) Int() uint16 { panic(notImplemented) }

// Update is key_mods.RemapSet.Mask.update: adds a modifier to the mask,
// marking it as having a remapping.
func (self *InputKeyModsRemapSetMask) Update(mods InputKeyModsMods) { panic(notImplemented) }

// Match is key_mods.RemapSet.Mask.match: returns true if the given mods
// match any remapping in this mask. This is a fast check to avoid expensive
// map lookups when no remapping could possibly apply.
//
// Checks both that the modifier key is remapped AND that the
// specific side (left/right) being pressed has a remapping.
func (self *InputKeyModsRemapSetMask) Match(mods InputKeyModsMods) bool { panic(notImplemented) }
