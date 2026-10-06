// Port of src/input/key_mods.zig (Ghostty v1.3.1).
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
// it is declared in full because the project's tests (patch 0017, 60 tests
// "term51: RemapSet...") cover every function of it.
//
// # Packed structs
//
// `Mods` (packed struct(u16)), `Mods.Keys` (u4), `Mods.Side` (u4),
// `RemapSet.Mask` (u12) are ordinary Go structs with the same fields in the
// same order, comparable with `==` as `expectEqual` compares them. The
// packed-integer conversions that key_mods.zig writes as `@bitCast` or
// `.int()` are `Int` on each (Zig's own `int` for Mods and Mods.Keys; a
// stand-in for `@bitCast` for Mods.Side and Mask) and `...FromInt` for
// Mods, Mods.Keys and Mods.Side (used by `keys`, `unset`, `apply` and
// `Mask.update`). The bit layout is Zig's: first field in bit 0.
// `Mods._padding` (u6, bits 10 to 15) is kept as a field: Mods.int and
// therefore `empty` and `equal` see it. A Go field wider than its Zig field
// (a Mod.Side above 1, a _padding above 0x3F) holds a value that cannot
// exist in Zig; the `Int` conversions assert it fits rather than drop bits.
// `@truncate` to a u4 is an explicit `& 0xF`; `~` on a u4 is `^x & 0xF`.
//
// `@field(mods, name)` and `@field(mods.sides, name)` with a comptime
// modifier name (formatMod, initMods) are the accessors
// inputKeyModsModsKeyField and inputKeyModsModsSideField, a switch on the
// Mod value returning a pointer to the field named after its tag (no string
// work per call). formatMod's `inline for (.{ "shift", "ctrl", "alt",
// "super" })` is a loop over inputKeyModsRemapSetFormatModFields, the Mod
// values with those tag names, in that order.
//
// # Go-memory rule
//
//   - Every allocator parameter is dropped (`parseCLI`, `parse`, `deinit`,
//     `clone`). `Deinit` is kept because the tests call it; it resets the
//     map.
//   - `RemapSet.map` (field `map_`: `map` is a Go keyword) is
//     `std.AutoArrayHashMapUnmanaged(Mods, Mods)`, an insertion-ordered
//     map: `inputKeyModsRemapSetMap`, two parallel slices
//     with the operations key_mods.zig and the tests use, under
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
//     (color.go), as in Zig. The implementation adds `write`
//     (FixedBufferStream.write), `writeAll` and `writeByte` (the
//     GenericWriter / DeprecatedWriter methods) with Zig 0.15.2's code: a
//     writeAll copies what fits, then fails on the next write with
//     ErrNoSpaceLeft. Terminal.go's printAttributes has its own stand-in;
//     this one is kept separate (could be shared later).
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
// `builtin.target.os.tag.isDarwin()`. The library is platform-neutral: both
// branches are ported and the choice is made by `inputKeyModsIsDarwin()`,
// which returns platformIsDarwin() (platform.go). Off Darwin, `translation`
// returns the mods unchanged and `ctrlOrSuper` returns `ctrl`.
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
//   - `inputKeyModsStringToEnumMod`, `inputKeyModsStringToEnumModSide`:
//     `std.meta.stringToEnum(Mod, s)` and `std.meta.stringToEnum(Mod.Side,
//     s)`. Zig 0.15.2 looks the name up in a std.StaticStringMap of the tag
//     names; the Go functions are a loop over the tag names comparing the
//     whole string, which gives the StaticStringMap's exact-match result
//     (the value whose name is exactly s, else null);
//   - `inputKeyModsRemapSetFinalizeSideMask`, finalize's
//     `comptime Mods.side_mask.int()`, computed once at start-up;
//   - `inputKeyModsRemapSetFormatModFields` and
//     `inputKeyModsRemapSetFormatModNames`, formatMod's comptime tuple of
//     names (above);
//   - `inputKeyModsModsKeyField` and `inputKeyModsModsSideField`, the
//     `@field` accessors (above);
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

import (
	"bytes"
	"errors"
)

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
// Upstream's five entries in upstream's order (generated from the Zig
// source by a throwaway script).
var InputKeyModsAlias = []InputKeyModsAliasEntry{
	{[]uint8("cmd"), InputKeyModsModSuper},
	{[]uint8("command"), InputKeyModsModSuper},
	{[]uint8("opt"), InputKeyModsModAlt},
	{[]uint8("option"), InputKeyModsModAlt},
	{[]uint8("control"), InputKeyModsModCtrl},
}

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
// std.meta.stringToEnum matches in RemapSet.parseMod. A value outside the
// enum panics (index out of range).
func (self InputKeyModsMod) String() string { return inputKeyModsModNames[self] }

// inputKeyModsModNames holds the Zig tag names, indexed by value.
var inputKeyModsModNames = [inputKeyModsModCount]string{"shift", "ctrl", "alt", "super"}

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
// std.meta.stringToEnum matches in RemapSet.parseMod. A value outside the
// enum panics (index out of range).
func (self InputKeyModsModSide) String() string { return inputKeyModsModSideNames[self] }

// inputKeyModsModSideNames holds the Zig tag names, indexed by value.
var inputKeyModsModSideNames = [inputKeyModsModSideCount]string{"left", "right"}

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
func (self InputKeyModsModsKeys) Int() InputKeyModsModsKeysBacking {
	var v InputKeyModsModsKeysBacking
	if self.shift {
		v |= 1 << 0
	}
	if self.ctrl {
		v |= 1 << 1
	}
	if self.alt {
		v |= 1 << 2
	}
	if self.super {
		v |= 1 << 3
	}
	return v
}

// InputKeyModsModsKeysFromInt stands for Zig's `@bitCast` from Keys.Backing
// to Keys (Mods.keys, RemapSet.Mask.update). Only the low four bits are
// meaningful.
//
// A value above 0xF cannot be a u4 in Zig: asserted.
func InputKeyModsModsKeysFromInt(v InputKeyModsModsKeysBacking) InputKeyModsModsKeys {
	assert(v <= 0xF)
	return InputKeyModsModsKeys{
		shift: v&(1<<0) != 0,
		ctrl:  v&(1<<1) != 0,
		alt:   v&(1<<2) != 0,
		super: v&(1<<3) != 0,
	}
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
//
// A field above 1 cannot be a Mod.Side (enum(u1)) in Zig: asserted.
func (self InputKeyModsModsSide) Int() InputKeyModsModsSideBacking {
	assert(self.shift <= 1 && self.ctrl <= 1 && self.alt <= 1 && self.super <= 1)
	return InputKeyModsModsSideBacking(self.shift) |
		InputKeyModsModsSideBacking(self.ctrl)<<1 |
		InputKeyModsModsSideBacking(self.alt)<<2 |
		InputKeyModsModsSideBacking(self.super)<<3
}

// InputKeyModsModsSideFromInt stands for Zig's `@bitCast` from Side.Backing
// to Side (RemapSet.Mask.update), the inverse of Int.
//
// A value above 0xF cannot be a u4 in Zig: asserted.
func InputKeyModsModsSideFromInt(v InputKeyModsModsSideBacking) InputKeyModsModsSide {
	assert(v <= 0xF)
	return InputKeyModsModsSide{
		shift: InputKeyModsModSide(v >> 0 & 1),
		ctrl:  InputKeyModsModSide(v >> 1 & 1),
		alt:   InputKeyModsModSide(v >> 2 & 1),
		super: InputKeyModsModSide(v >> 3 & 1),
	}
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
//
// Zig: `@bitCast(self)`. A _padding above 0x3F cannot be a u6 in Zig:
// asserted (the sides are checked by Side.Int).
func (self InputKeyModsMods) Int() InputKeyModsModsBacking {
	assert(self._padding <= 0x3F)
	var v InputKeyModsModsBacking
	if self.shift {
		v |= 1 << 0
	}
	if self.ctrl {
		v |= 1 << 1
	}
	if self.alt {
		v |= 1 << 2
	}
	if self.super {
		v |= 1 << 3
	}
	if self.caps_lock {
		v |= 1 << 4
	}
	if self.num_lock {
		v |= 1 << 5
	}
	v |= InputKeyModsModsBacking(self.sides.Int()) << 6
	v |= InputKeyModsModsBacking(self._padding) << 10
	return v
}

// InputKeyModsModsFromInt stands for Zig's `@bitCast` from Mods.Backing to
// Mods (Mods.unset, RemapSet.apply), the inverse of Int. Every bit is kept,
// including _padding.
func InputKeyModsModsFromInt(v InputKeyModsModsBacking) InputKeyModsMods {
	return InputKeyModsMods{
		shift:     v&(1<<0) != 0,
		ctrl:      v&(1<<1) != 0,
		alt:       v&(1<<2) != 0,
		super:     v&(1<<3) != 0,
		caps_lock: v&(1<<4) != 0,
		num_lock:  v&(1<<5) != 0,
		sides:     InputKeyModsModsSideFromInt(InputKeyModsModsSideBacking(v >> 6 & 0xF)),
		_padding:  uint8(v >> 10 & 0x3F),
	}
}

// Empty is key_mods.Mods.empty: returns true if no modifiers are set.
func (self InputKeyModsMods) Empty() bool {
	return self.Int() == 0
}

// Equal is key_mods.Mods.equal: returns true if two mods are equal.
func (self InputKeyModsMods) Equal(other InputKeyModsMods) bool {
	return self.Int() == other.Int()
}

// Keys is key_mods.Mods.keys: returns only the keys.
//
// In the future I want to remove `binding` for this. I didn't want
// to do that all in one PR where I added this because its a bigger
// change.
func (self InputKeyModsMods) Keys() InputKeyModsModsKeys {
	backing := InputKeyModsModsKeysBacking(self.Int() & 0xF) // @truncate
	return InputKeyModsModsKeysFromInt(backing)
}

// Binding is key_mods.Mods.binding: return mods that are only relevant for
// bindings.
func (self InputKeyModsMods) Binding() InputKeyModsMods {
	return InputKeyModsMods{
		shift: self.shift,
		ctrl:  self.ctrl,
		alt:   self.alt,
		super: self.super,
	}
}

// Unset is key_mods.Mods.unset: perform `self &~ other` to remove the other
// mods from self.
func (self InputKeyModsMods) Unset(other InputKeyModsMods) InputKeyModsMods {
	return InputKeyModsModsFromInt(self.Int() & ^other.Int())
}

// WithoutLocks is key_mods.Mods.withoutLocks: returns the mods without
// locks set.
func (self InputKeyModsMods) WithoutLocks() InputKeyModsMods {
	copy := self
	copy.caps_lock = false
	copy.num_lock = false
	return copy
}

// Translation is key_mods.Mods.translation: return the mods to use for key
// translation. This handles settings like macos-option-as-alt. The
// translation mods should be used for translation but never sent back in
// for the key callback.
//
// Darwin: the alt handling is inside `if (comptime isDarwin())`; see the
// file header. Off Darwin the result is self.
func (self InputKeyModsMods) Translation(option_as_alt InputConfigOptionAsAlt) InputKeyModsMods {
	result := self

	// macos-option-as-alt for darwin
	//
	// Zig: labeled block `alt:`; `break :alt` is `goto alt`.
	if inputKeyModsIsDarwin() {
		// Alt has to be set only on the correct side
		switch option_as_alt {
		case InputConfigOptionAsAltFalse:
			goto alt
		case InputConfigOptionAsAltTrue:
		case InputConfigOptionAsAltLeft:
			if self.sides.alt == InputKeyModsModSideRight {
				goto alt
			}
		case InputConfigOptionAsAltRight:
			if self.sides.alt == InputKeyModsModSideLeft {
				goto alt
			}
		default:
			panic("unreachable")
		}

		// Unset alt
		result.alt = false
	}
alt:

	return result
}

// CtrlOrSuper is key_mods.Mods.ctrlOrSuper: checks to see if super is on
// (MacOS) or ctrl.
//
// Darwin: see the file header. Off Darwin the result is self.ctrl.
func (self InputKeyModsMods) CtrlOrSuper() bool {
	if inputKeyModsIsDarwin() {
		return self.super
	}
	return self.ctrl
}

// inputKeyModsIsDarwin stands for the comptime check
// `builtin.target.os.tag.isDarwin()` in Mods.translation and
// Mods.ctrlOrSuper. It returns platformIsDarwin() (platform.go).
func inputKeyModsIsDarwin() bool { return platformIsDarwin() }

// inputKeyModsModsKeyField stands for `@field(mods, @tagName(mod))`: a
// pointer to the bool of mods named after mod. Not a Zig declaration. A
// value outside the enum panics (Zig: a compile error).
func inputKeyModsModsKeyField(mods *InputKeyModsMods, mod InputKeyModsMod) *bool {
	switch mod {
	case InputKeyModsModShift:
		return &mods.shift
	case InputKeyModsModCtrl:
		return &mods.ctrl
	case InputKeyModsModAlt:
		return &mods.alt
	case InputKeyModsModSuper:
		return &mods.super
	default:
		panic("unreachable")
	}
}

// inputKeyModsModsSideField stands for `@field(mods.sides, @tagName(mod))`:
// a pointer to the side of sides named after mod. Not a Zig declaration. A
// value outside the enum panics (Zig: a compile error).
func inputKeyModsModsSideField(sides *InputKeyModsModsSide, mod InputKeyModsMod) *InputKeyModsModSide {
	switch mod {
	case InputKeyModsModShift:
		return &sides.shift
	case InputKeyModsModCtrl:
		return &sides.ctrl
	case InputKeyModsModAlt:
		return &sides.alt
	case InputKeyModsModSuper:
		return &sides.super
	default:
		panic("unreachable")
	}
}

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
//
// Keys compare with Go's ==, every field: the bit equality of std's
// AutoContext on a packed struct (the Int() of every stored key is valid).
func (self *inputKeyModsRemapSetMap) Put(key InputKeyModsMods, value InputKeyModsMods) error {
	for i := range self.keys {
		if self.keys[i] == key {
			self.values[i] = value
			return nil
		}
	}
	self.keys = append(self.keys, key)
	self.values = append(self.values, value)
	return nil
}

// Get is std's get: the value for key, or null.
func (self *inputKeyModsRemapSetMap) Get(key InputKeyModsMods) Optional[InputKeyModsMods] {
	for i := range self.keys {
		if self.keys[i] == key {
			return Some(self.values[i])
		}
	}
	return None[InputKeyModsMods]()
}

// SwapRemove is std's swapRemove: if key is present, remove it by moving the
// last entry into its place and return true; otherwise return false.
func (self *inputKeyModsRemapSetMap) SwapRemove(key InputKeyModsMods) bool {
	for i := range self.keys {
		if self.keys[i] == key {
			last := len(self.keys) - 1
			self.keys[i] = self.keys[last]
			self.values[i] = self.values[last]
			self.keys = self.keys[:last]
			self.values = self.values[:last]
			return true
		}
	}
	return false
}

// ClearRetainingCapacity is std's clearRetainingCapacity: remove every
// entry.
func (self *inputKeyModsRemapSetMap) ClearRetainingCapacity() {
	self.keys = self.keys[:0]
	self.values = self.values[:0]
}

// Keys is std's keys: the keys in order. The slice is the map's own and is
// only valid until the map is modified.
func (self *inputKeyModsRemapSetMap) Keys() []InputKeyModsMods { return self.keys }

// Values is std's values: the values, parallel to Keys. The slice is the
// map's own and is only valid until the map is modified.
func (self *inputKeyModsRemapSetMap) Values() []InputKeyModsMods { return self.values }

// Count is std's count: the number of entries.
func (self *inputKeyModsRemapSetMap) Count() uint { return uint(len(self.keys)) }

// Clone is std's clone: an independent copy with the same entries in the
// same order. The allocator parameter is dropped.
//
// Zig: Allocator.Error!Self. The error is ErrOutOfMemory (dcs.go).
func (self *inputKeyModsRemapSetMap) Clone() (inputKeyModsRemapSetMap, error) {
	return inputKeyModsRemapSetMap{
		keys:   append([]InputKeyModsMods(nil), self.keys...),
		values: append([]InputKeyModsMods(nil), self.values...),
	}, nil
}

// Sort is std's sort (ArrayHashMap.sort, stable): reorder the entries with
// `lessThan(a_index, b_index)`, the `lessThan` of Zig's sort context. Must
// be std.sort.insertionContext step for step; see the file header.
//
// ArrayHashMap.sort -> MultiArrayList.sort -> std.mem.sortContext(0, len,
// ...) -> std.sort.insertionContext, whose swap exchanges the key and the
// value at the two indexes. lessThan sees the entries as they are at the
// moment of the call (the context's `keys` is the live keys slice).
func (self *inputKeyModsRemapSetMap) Sort(lessThan func(a_index uint, b_index uint) bool) {
	// std.sort.insertionContext(a = 0, b = len, context)
	a := uint(0)
	b := uint(len(self.keys))
	assert(a <= b)

	i := a + 1
	for ; i < b; i += 1 {
		j := i
		for ; j > a && lessThan(j, j-1); j -= 1 {
			// context.swap(j, j - 1)
			self.keys[j], self.keys[j-1] = self.keys[j-1], self.keys[j]
			self.values[j], self.values[j-1] = self.values[j-1], self.values[j]
		}
	}
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
	value := []uint8("")
	if input.Valid {
		value = input.Value
	}

	// Empty value resets the set
	if len(value) == 0 {
		self.map_.ClearRetainingCapacity()
		self.mask = InputKeyModsRemapSetMask{}
		return nil
	}

	if err := self.Parse(value); err != nil {
		switch err {
		case ErrOutOfMemory:
			return ErrOutOfMemory
		case ErrMissingAssignment, ErrInvalidMod:
			return ErrInvalidValue
		default:
			panic("unreachable")
		}
	}
	return nil
}

// Parse is key_mods.RemapSet.parse: parse a modifier remap and add it to
// the set.
//
// Zig: ParseError!void. The allocator is dropped. The errors are
// ErrOutOfMemory, ErrMissingAssignment and ErrInvalidMod.
func (self *InputKeyModsRemapSet) Parse(input []uint8) error {
	// Find the assignment point ('=')
	eql_idx := bytes.IndexByte(input, '=')
	if eql_idx < 0 {
		return ErrMissingAssignment
	}

	// The to side defaults to "left" if no explicit side is given.
	// This is because this is the default unsided value provided by
	// the apprts in the current Mods layout.
	var to InputKeyModsMods
	{
		// Zig: labeled block `to:`.
		raw0, raw1, err := inputKeyModsRemapSetParseMod(input[eql_idx+1:])
		if err != nil {
			return err
		}
		side := InputKeyModsModSideLeft
		if raw1.Valid {
			side = raw1.Value
		}
		to = inputKeyModsRemapSetInitMods(raw0, side)
	}

	// The from side, if sided, is easy and we put it directly into
	// the map.
	from_raw0, from_raw1, err := inputKeyModsRemapSetParseMod(input[0:eql_idx])
	if err != nil {
		return err
	}
	if from_raw1.Valid {
		from_side := from_raw1.Value
		from := inputKeyModsRemapSetInitMods(from_raw0, from_side)
		if err := self.map_.Put(from, to); err != nil {
			return err
		}
		// Zig: errdefer comptime unreachable;
		self.mask.Update(from)
		return nil
	}

	// We need to do some combinatorial explosion here for unsided
	// from in order to assign all possible sides.
	from_left := inputKeyModsRemapSetInitMods(from_raw0, InputKeyModsModSideLeft)
	from_right := inputKeyModsRemapSetInitMods(from_raw0, InputKeyModsModSideRight)
	if err := self.map_.Put(from_left, to); err != nil {
		return err
	}
	// Zig: errdefer _ = self.map.swapRemove(from_left);
	if err := self.map_.Put(from_right, to); err != nil {
		_ = self.map_.SwapRemove(from_left)
		return err
	}
	// Zig: errdefer _ = self.map.swapRemove(from_right); (nothing after
	// this point can fail)

	// Zig: errdefer comptime unreachable;
	self.mask.Update(from_left)
	self.mask.Update(from_right)
	return nil
}

// Deinit is key_mods.RemapSet.deinit. It only frees in Zig; it is kept
// because every test calls it.
//
// Zig: self.map.deinit(alloc), which frees and leaves the map undefined;
// Go resets the map to its zero value (the GC frees). The mask is left as
// it is, as in Zig.
func (self *InputKeyModsRemapSet) Deinit() {
	self.map_ = inputKeyModsRemapSetMap{}
}

// Finalize is key_mods.RemapSet.finalize: must be called prior to any
// remappings so that the mapping is sorted properly. Otherwise, you will
// get invalid results.
//
// Zig's local `Context` struct holds `keys: []const Mods`
// (self.map.keys(), the live slice the sort swaps in place); its lessThan
// is the closure below.
func (self *InputKeyModsRemapSet) Finalize() {
	keys := self.map_.Keys()
	lessThan := func(a_index uint, b_index uint) bool {
		_ = b_index

		// Mods with any right sides prioritize
		side_mask := inputKeyModsRemapSetFinalizeSideMask
		a := keys[a_index]
		return a.Int()&side_mask != 0
	}

	self.map_.Sort(lessThan)
}

// inputKeyModsRemapSetFinalizeSideMask is finalize's
// `comptime Mods.side_mask.int()`, computed once at start-up.
var inputKeyModsRemapSetFinalizeSideMask = InputKeyModsModsSideMask.Int()

// Clone is key_mods.RemapSet.clone: deep copy of the struct. Required by
// Config.
//
// Zig: Allocator.Error!RemapSet. The allocator is dropped; the error is
// ErrOutOfMemory (dcs.go).
func (self *InputKeyModsRemapSet) Clone() (InputKeyModsRemapSet, error) {
	map_, err := self.map_.Clone()
	if err != nil {
		return InputKeyModsRemapSet{}, err
	}
	return InputKeyModsRemapSet{
		map_: map_,
		mask: self.mask,
	}, nil
}

// Equal is key_mods.RemapSet.equal: compare if two RemapSets are equal.
// Required by Config.
func (self InputKeyModsRemapSet) Equal(other InputKeyModsRemapSet) bool {
	if self.map_.Count() != other.map_.Count() {
		return false
	}

	keys := self.map_.Keys()
	values := self.map_.Values()
	for i := range keys {
		other_value := other.map_.Get(keys[i])
		if !other_value.Valid {
			return false
		}
		if !values[i].Equal(other_value.Value) {
			return false
		}
	}

	return true
}

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
	if self.map_.Count() == 0 {
		// Zig: try formatter.formatEntry(void, {});
		if err := formatter.formatEntry(None[[]uint8]()); err != nil {
			return err
		}
		return nil
	}

	keys := self.map_.Keys()
	values := self.map_.Values()
	for i := range keys {
		from := keys[i]
		to := values[i]

		var buf [64]uint8
		fbs := inputKeyModsFixedBufferStreamInit(buf[:])
		writer := &fbs
		if inputKeyModsRemapSetFormatMod(writer, from) != nil {
			return ErrOutOfMemory
		}
		if writer.writeByte('=') != nil {
			return ErrOutOfMemory
		}
		if inputKeyModsRemapSetFormatMod(writer, to) != nil {
			return ErrOutOfMemory
		}
		if err := formatter.formatEntry(Some(fbs.getWritten())); err != nil {
			return err
		}
	}
	return nil
}

// inputKeyModsFixedBufferStream stands for std.io.FixedBufferStream([]u8):
// a buffer and a write position. Not a Zig declaration. Zig's
// `fbs.writer()` is the stream itself here; a write that does not fit
// fails with ErrNoSpaceLeft (color.go). Declared: the fields, Init,
// getWritten, write (FixedBufferStream.write), and writeAll and writeByte
// (the writer's methods).
type inputKeyModsFixedBufferStream struct {
	buffer []uint8
	// Zig: pos: usize.
	pos uint
}

// inputKeyModsFixedBufferStreamInit is std.io.fixedBufferStream(buf).
func inputKeyModsFixedBufferStreamInit(buf []uint8) inputKeyModsFixedBufferStream {
	return inputKeyModsFixedBufferStream{buffer: buf, pos: 0}
}

// getWritten is std.io.FixedBufferStream.getWritten: buffer[:pos].
func (self *inputKeyModsFixedBufferStream) getWritten() []uint8 {
	return self.buffer[0:self.pos]
}

// write is std.io.FixedBufferStream.write.
//
// If the returned number of bytes written is less than requested, the
// buffer is full. Returns `error.NoSpaceLeft` when no bytes would be written.
// Note: `error.NoSpaceLeft` matches the corresponding error from
// `std.fs.File.WriteError`.
func (self *inputKeyModsFixedBufferStream) write(bytes []uint8) (uint, error) {
	if len(bytes) == 0 {
		return 0, nil
	}
	if self.pos >= uint(len(self.buffer)) {
		return 0, ErrNoSpaceLeft
	}

	n := min(uint(len(self.buffer))-self.pos, uint(len(bytes)))
	copy(self.buffer[self.pos:][0:n], bytes[0:n])
	self.pos += n

	if n == 0 {
		return 0, ErrNoSpaceLeft
	}

	return n, nil
}

// writeAll is the fixed-buffer writer's writeAll (std.io.GenericWriter ->
// DeprecatedWriter.writeAll): write until everything is written, failing
// with the first error. What fit before the error stays in the buffer.
func (self *inputKeyModsFixedBufferStream) writeAll(bytes []uint8) error {
	var index uint = 0
	for index != uint(len(bytes)) {
		n, err := self.write(bytes[index:])
		if err != nil {
			return err
		}
		index += n
	}
	return nil
}

// writeByte is the fixed-buffer writer's writeByte (DeprecatedWriter):
// writeAll of a one-byte array.
func (self *inputKeyModsFixedBufferStream) writeByte(byte_ uint8) error {
	array := [1]uint8{byte_}
	return self.writeAll(array[:])
}

// inputKeyModsRemapSetFormatMod is key_mods.RemapSet.formatMod.
//
// Zig: fn formatMod(writer: anytype, mods: Mods) !void. The writer is
// always a fixedBufferStream writer; see the file header. The error is
// ErrNoSpaceLeft.
func inputKeyModsRemapSetFormatMod(writer *inputKeyModsFixedBufferStream, mods InputKeyModsMods) error {
	// Check which mod is set and format it with optional side prefix
	for _, name := range inputKeyModsRemapSetFormatModFields {
		if *inputKeyModsModsKeyField(&mods, name) {
			side := *inputKeyModsModsSideField(&mods.sides, name)
			if side == InputKeyModsModSideRight {
				if err := writer.writeAll([]uint8("right_")); err != nil {
					return err
				}
			} else {
				// Only write left_ if we need to distinguish
				// For now, always write left_ if it's a sided mapping
				if err := writer.writeAll([]uint8("left_")); err != nil {
					return err
				}
			}
			if err := writer.writeAll(inputKeyModsRemapSetFormatModNames[name]); err != nil {
				return err
			}
			return nil
		}
	}
	return nil
}

// inputKeyModsRemapSetFormatModFields is formatMod's comptime tuple
// `.{ "shift", "ctrl", "alt", "super" }`, as the Mod values whose tag names
// those are, in that order.
var inputKeyModsRemapSetFormatModFields = [...]InputKeyModsMod{
	InputKeyModsModShift,
	InputKeyModsModCtrl,
	InputKeyModsModAlt,
	InputKeyModsModSuper,
}

// inputKeyModsRemapSetFormatModNames holds the names formatMod writes,
// indexed by Mod: the strings of the tuple, as bytes, built once.
var inputKeyModsRemapSetFormatModNames = [inputKeyModsModCount][]uint8{
	InputKeyModsModShift: []uint8("shift"),
	InputKeyModsModCtrl:  []uint8("ctrl"),
	InputKeyModsModAlt:   []uint8("alt"),
	InputKeyModsModSuper: []uint8("super"),
}

// inputKeyModsRemapSetParseMod is key_mods.RemapSet.parseMod: parses a
// single mode in a single remapping string. E.g. `ctrl` or `left_shift`.
//
// Zig: error{InvalidMod}!struct { Mod, ?Mod.Side }. The error is
// ErrInvalidMod.
func inputKeyModsRemapSetParseMod(input []uint8) (InputKeyModsMod, Optional[InputKeyModsModSide], error) {
	var side_str, mod_str []uint8
	if idx := bytes.IndexByte(input, '_'); idx >= 0 {
		side_str, mod_str = input[0:idx], input[idx+1:]
	} else {
		side_str, mod_str = []uint8(""), input
	}

	// Zig: `if (std.meta.stringToEnum(Mod, mod_str)) |mod| mod else mod: {
	// ... }`; the labeled block is the alias loop.
	var mod InputKeyModsMod
	if m := inputKeyModsStringToEnumMod(mod_str); m.Valid {
		mod = m.Value
	} else {
		found := false
		for _, pair := range InputKeyModsAlias {
			if bytes.Equal(mod_str, pair.x0) {
				mod = pair.x1
				found = true
				break
			}
		}

		if !found {
			return 0, Optional[InputKeyModsModSide]{}, ErrInvalidMod
		}
	}

	var side Optional[InputKeyModsModSide]
	if len(side_str) > 0 {
		side = inputKeyModsStringToEnumModSide(side_str)
		if !side.Valid {
			return 0, Optional[InputKeyModsModSide]{}, ErrInvalidMod
		}
	}
	return mod, side, nil
}

// inputKeyModsStringToEnumMod is std.meta.stringToEnum(Mod, s): the value
// whose tag name is exactly s. Not a Zig declaration.
func inputKeyModsStringToEnumMod(s []uint8) Optional[InputKeyModsMod] {
	for v := InputKeyModsMod(0); v < inputKeyModsModCount; v++ {
		if string(s) == v.String() {
			return Some(v)
		}
	}
	return None[InputKeyModsMod]()
}

// inputKeyModsStringToEnumModSide is std.meta.stringToEnum(Mod.Side, s):
// the value whose tag name is exactly s. Not a Zig declaration.
func inputKeyModsStringToEnumModSide(s []uint8) Optional[InputKeyModsModSide] {
	for v := InputKeyModsModSide(0); v < inputKeyModsModSideCount; v++ {
		if string(s) == v.String() {
			return Some(v)
		}
	}
	return None[InputKeyModsModSide]()
}

// inputKeyModsRemapSetInitMods is key_mods.RemapSet.initMods.
func inputKeyModsRemapSetInitMods(mod InputKeyModsMod, side InputKeyModsModSide) InputKeyModsMods {
	// Zig: switch (mod) { inline else => |tag| { ... } }.
	mods := InputKeyModsMods{}
	*inputKeyModsModsKeyField(&mods, mod) = true
	*inputKeyModsModsSideField(&mods.sides, mod) = side
	return mods
}

// IsRemapped is key_mods.RemapSet.isRemapped: returns true if the given
// mods need remapping.
func (self *InputKeyModsRemapSet) IsRemapped(mods InputKeyModsMods) bool {
	return self.mask.Match(mods)
}

// Apply is key_mods.RemapSet.apply: apply a remap to the given mods.
//
// Zig ends with `unreachable` when the mask matched but no entry did; the
// port panics there.
func (self *InputKeyModsRemapSet) Apply(mods InputKeyModsMods) InputKeyModsMods {
	if !self.IsRemapped(mods) {
		return mods
	}

	mods_binding := InputKeyModsModsKeysBacking(mods.Int() & 0xF) // @truncate
	mods_sides := mods.sides.Int()

	keys := self.map_.Keys()
	values := self.map_.Values()
	for i := range keys {
		from := keys[i]
		from_binding := InputKeyModsModsKeysBacking(from.Int() & 0xF) // @truncate
		if mods_binding&from_binding != from_binding {
			continue
		}
		from_sides := from.sides.Int()
		if (mods_sides^from_sides)&from_binding != 0 {
			continue
		}

		mods_int := mods.Int()
		mods_int &= ^from.Int()
		mods_int |= values[i].Int()
		return InputKeyModsModsFromInt(mods_int)
	}

	panic("unreachable")
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
func (self InputKeyModsRemapSetMask) Int() uint16 {
	return uint16(self.keys.Int()) |
		uint16(self.left_sides.Int())<<4 |
		uint16(self.right_sides.Int())<<8
}

// Update is key_mods.RemapSet.Mask.update: adds a modifier to the mask,
// marking it as having a remapping.
func (self *InputKeyModsRemapSetMask) Update(mods InputKeyModsMods) {
	keys_int := mods.Keys().Int()

	// OR the new keys into our existing keys mask.
	// Example: keys=0b0000, new ctrl → keys=0b0010
	self.keys = InputKeyModsModsKeysFromInt(self.keys.Int() | keys_int)

	// Both Keys and Side are u4 with matching bit positions.
	// This lets us use keys_int to select which side bits to update.
	sides := mods.sides.Int()
	left_int := self.left_sides.Int()
	right_int := self.right_sides.Int()

	// Update left_sides: set bit if this key is active AND side is left.
	// Since Side.left=0, we invert sides (~sides) so left becomes 1.
	// keys_int masks to only affect the modifier being added.
	// Example: left_ctrl → keys_int=0b0010, ~sides=0b1111 (left=0 inverted)
	//          result: left_int | (0b0010 & 0b1111) = left_int | 0b0010
	self.left_sides = InputKeyModsModsSideFromInt(left_int | (keys_int & (^sides & 0xF)))

	// Update right_sides: set bit if this key is active AND side is right.
	// Since Side.right=1, we use sides directly.
	// Example: right_ctrl → keys_int=0b0010, sides=0b0010 (right=1)
	//          result: right_int | (0b0010 & 0b0010) = right_int | 0b0010
	self.right_sides = InputKeyModsModsSideFromInt(right_int | (keys_int & sides))
}

// Match is key_mods.RemapSet.Mask.match: returns true if the given mods
// match any remapping in this mask. This is a fast check to avoid expensive
// map lookups when no remapping could possibly apply.
//
// Checks both that the modifier key is remapped AND that the
// specific side (left/right) being pressed has a remapping.
func (self *InputKeyModsRemapSetMask) Match(mods InputKeyModsMods) bool {
	// Find which pressed keys have remappings registered.
	// Example: pressed={ctrl,alt}, mask={ctrl} → active=0b0010 (just ctrl)
	active := mods.Keys().Int() & self.keys.Int()
	if active == 0 {
		return false
	}

	// Check if the pressed side matches a remapped side.
	// For left (sides bit = 0): check against left_int (where 1 = left remapped)
	//   ~sides inverts so left becomes 1, then AND with left_int
	// For right (sides bit = 1): check against right_int directly
	//
	// Example: pressing left_ctrl (sides.ctrl=0, left_int.ctrl=1)
	//   ~sides = 0b1111, left_int = 0b0010
	//   (~sides & left_int) = 0b0010 ✓ matches
	//
	// Example: pressing right_ctrl but only left_ctrl is remapped
	//   sides = 0b0010, left_int = 0b0010, right_int = 0b0000
	//   (~0b0010 & 0b0010) | (0b0010 & 0b0000) = 0b0000 ✗ no match
	sides := mods.sides.Int()
	left_int := self.left_sides.Int()
	right_int := self.right_sides.Int()
	side_match := ((^sides & 0xF) & left_int) | (sides & right_int)

	// Final check: is any active (pressed + remapped) key also side-matched?
	return (active & side_match) != 0
}
