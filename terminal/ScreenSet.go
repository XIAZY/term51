// WIREFRAME for src/terminal/ScreenSet.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// A ScreenSet holds multiple terminal screens. This is initially created
// to handle simple primary vs alternate screens, but could be extended
// in the future to handle N screens.
//
// One of the goals of this is to allow lazy initialization of screens
// as needed. The primary screen is always initialized, but the alternate
// screen may not be until first used.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
//
// File naming: ScreenSet.zig is a file-level struct, so the type is
// `ScreenSet` and its constructor is `ScreenSetInit`. What is nested in it
// takes the prefix `ScreenSet`: `ScreenSet.Key` is `ScreenSetKey`.
//
// Declarations appear in the same order as in ScreenSet.zig.
//
// # Go-memory rule
//
// (docs/mapping.md, governing rule 3.) Ordinary Go memory. A screen is a
// `*Screen` allocated by Go (`alloc.create(Screen)` is `new(Screen)`); the
// garbage collector frees it (`alloc.destroy`). Allocator parameters are
// dropped; `deinit` is kept because it deinitialises every screen.
//
// `all` is a `std.EnumMap(Key, *Screen)`: one optional value per key. Go:
// `screenSetEnumMap`, an array of `*Screen` indexed by the key in which nil
// means "not present", with stubs for the std operations ScreenSet.zig and
// its tests use, under their std names: `Get`, `Put`, `FetchRemove`,
// `Count`. Not declared: `init` (a composite literal) and `iterator` (used
// once, in `deinit`; the Go code ranges over the array in key order, which
// is the order std's iterator yields).
//
// # Declarations that are not Zig declarations
//
//   - `screenSetEnumMap` and its methods: std.EnumMap, see above;
//   - the `String` method of `ScreenSetKey`.
//
// # Omitted
//
// Aliases and imports with nothing to declare: `Screen`, `Allocator`,
// `assert`, `testing`.

package terminal

// ScreenSetKey is ScreenSet.Key: the possible keys for screens in the
// screen set.
//
// Zig: enum(u1), values numbered from zero in the order listed.
type ScreenSetKey uint8

const (
	ScreenSetKeyPrimary ScreenSetKey = iota
	ScreenSetKeyAlternate
)

// String returns the Zig tag name (@tagName).
func (self ScreenSetKey) String() string { panic(notImplemented) }

// screenSetEnumMap is NOT a declaration of ScreenSet.zig: it is
// std.EnumMap(Key, *Screen), the type of `ScreenSet.all`. The methods below
// are std's, limited to the ones ScreenSet.zig and its tests call. The zero
// value is the empty map.
type screenSetEnumMap struct {
	// One entry per ScreenSetKey, indexed by the key; nil is "not present".
	values [2]*Screen
}

// Get is std's EnumMap.get: the value for key, or null (nil).
func (self *screenSetEnumMap) Get(key ScreenSetKey) *Screen { panic(notImplemented) }

// Put is std's EnumMap.put: set the value for key.
func (self *screenSetEnumMap) Put(key ScreenSetKey, value *Screen) { panic(notImplemented) }

// FetchRemove is std's EnumMap.fetchRemove: remove key and return the value
// it had, or null (nil).
func (self *screenSetEnumMap) FetchRemove(key ScreenSetKey) *Screen { panic(notImplemented) }

// Count is std's EnumMap.count: the number of keys present.
//
// Zig: returns usize.
func (self *screenSetEnumMap) Count() uint { panic(notImplemented) }

// ScreenSet is the file-level struct of ScreenSet.zig.
//
// The Go zero value is not an initialised set; use ScreenSetInit.
type ScreenSet struct {
	// The key value of the currently active screen. Useful for simple
	// comparisons, e.g. "is this screen the primary screen".
	active_key ScreenSetKey

	// The active screen pointer.
	active *Screen

	// All screens that are initialized.
	//
	// Zig: std.EnumMap(Key, *Screen). See the file header.
	all screenSetEnumMap
}

// ScreenSetInit is ScreenSet.init.
//
// Zig: pub fn init(alloc: Allocator, opts: Screen.Options)
// Allocator.Error!ScreenSet. The allocator parameter is dropped. The error
// is ErrOutOfMemory (dcs.go).
func ScreenSetInit(opts ScreenOptions) (ScreenSet, error) { panic(notImplemented) }

// Deinit is ScreenSet.deinit.
//
// Zig: pub fn deinit(self: *ScreenSet, alloc: Allocator) void. The
// allocator parameter is dropped. Kept: it deinitialises all initialized
// screens.
func (self *ScreenSet) Deinit() { panic(notImplemented) }

// Get is ScreenSet.get.
//
// Get the screen for the given key, if it is initialized.
//
// Zig: pub fn get(self: *const ScreenSet, key: Key) ?*Screen. Null is nil.
func (self *ScreenSet) Get(key ScreenSetKey) *Screen { panic(notImplemented) }

// GetInit is ScreenSet.getInit.
//
// Get the screen for the given key, initializing it if necessary.
//
// Zig: pub fn getInit(self: *ScreenSet, alloc: Allocator, key: Key,
// opts: Screen.Options) Allocator.Error!*Screen. The allocator parameter is
// dropped. The error is ErrOutOfMemory (dcs.go).
func (self *ScreenSet) GetInit(key ScreenSetKey, opts ScreenOptions) (*Screen, error) {
	panic(notImplemented)
}

// Remove is ScreenSet.remove.
//
// Remove a key from the set. The primary screen cannot be removed (asserted).
//
// Zig: pub fn remove(self: *ScreenSet, alloc: Allocator, key: Key) void.
// The allocator parameter is dropped.
func (self *ScreenSet) Remove(key ScreenSetKey) { panic(notImplemented) }

// SwitchTo is ScreenSet.switchTo.
//
// Switch the active screen to the given key. Requires that the
// screen is initialized.
func (self *ScreenSet) SwitchTo(key ScreenSetKey) { panic(notImplemented) }
