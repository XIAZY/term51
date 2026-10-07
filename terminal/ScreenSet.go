// Port of src/terminal/ScreenSet.zig (Ghostty v1.3.1).
//
// A ScreenSet holds multiple terminal screens. This is initially created
// to handle simple primary vs alternate screens, but could be extended
// in the future to handle N screens.
//
// One of the goals of this is to allow lazy initialization of screens
// as needed. The primary screen is always initialized, but the alternate
// screen may not be until first used.
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
//   - the `String` method of `ScreenSetKey` and its name table
//     `screenSetKeyNames`.
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
func (self ScreenSetKey) String() string { return screenSetKeyNames[self] }

// screenSetKeyNames are the tag names of ScreenSetKey, in order.
var screenSetKeyNames = [...]string{
	"primary",
	"alternate",
}

// screenSetEnumMap is NOT a declaration of ScreenSet.zig: it is
// std.EnumMap(Key, *Screen), the type of `ScreenSet.all`. The methods below
// are std's, limited to the ones ScreenSet.zig and its tests call. The zero
// value is the empty map.
type screenSetEnumMap struct {
	// One entry per ScreenSetKey, indexed by the key; nil is "not present".
	values [2]*Screen
}

// Get is std's EnumMap.get: the value for key, or null (nil).
func (self *screenSetEnumMap) Get(key ScreenSetKey) *Screen {
	// std: `if (self.bits.isSet(index)) self.values[index] else null`; the
	// presence bit is "not nil". A key outside the enum panics on the index.
	return self.values[key]
}

// Put is std's EnumMap.put: set the value for key.
func (self *screenSetEnumMap) Put(key ScreenSetKey, value *Screen) {
	// A `*Screen` is never null in Zig; nil would read as "not present".
	assert(value != nil)
	self.values[key] = value
}

// FetchRemove is std's EnumMap.fetchRemove: remove key and return the value
// it had, or null (nil).
func (self *screenSetEnumMap) FetchRemove(key ScreenSetKey) *Screen {
	// std: `defer self.bits.unset(index); return if (self.bits.isSet(index))
	// self.values[index] else null;`
	value := self.values[key]
	self.values[key] = nil
	return value
}

// Count is std's EnumMap.count: the number of keys present.
//
// Zig: returns usize.
func (self *screenSetEnumMap) Count() uint {
	// std: `self.bits.count()`.
	var n uint
	for _, v := range self.values {
		if v != nil {
			n++
		}
	}
	return n
}

// ScreenSet is the file-level struct of ScreenSet.zig.
//
// The Go zero value is not an initialised set; use ScreenSetInit.
type ScreenSet struct {
	// The key value of the currently active screen. Useful for simple
	// comparisons, e.g. "is this screen the primary screen".
	ActiveKey ScreenSetKey

	// The active screen pointer.
	Active *Screen

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
func ScreenSetInit(opts ScreenOptions) (ScreenSet, error) {
	// We need to initialize our initial primary screen
	screen := new(Screen)
	// Zig: errdefer alloc.destroy(screen); Go memory.
	s, err := ScreenInit(opts)
	if err != nil {
		return ScreenSet{}, err
	}
	*screen = s
	var all screenSetEnumMap
	all.Put(ScreenSetKeyPrimary, screen) // Zig: .init(.{ .primary = screen })
	return ScreenSet{
		ActiveKey: ScreenSetKeyPrimary,
		Active:    screen,
		all:       all,
	}, nil
}

// Deinit is ScreenSet.deinit.
//
// Zig: pub fn deinit(self: *ScreenSet, alloc: Allocator) void. The
// allocator parameter is dropped. Kept: it deinitialises all initialized
// screens.
func (self *ScreenSet) Deinit() {
	// Destroy all initialized screens
	//
	// Zig: `var it = self.all.iterator(); while (it.next()) |entry|`, which
	// yields the present keys in key order.
	for _, screen := range self.all.values {
		if screen == nil {
			continue
		}
		screen.Deinit()
		// Zig: alloc.destroy(entry.value.*); Go memory.
	}
}

// Get is ScreenSet.get.
//
// Get the screen for the given key, if it is initialized.
//
// Zig: pub fn get(self: *const ScreenSet, key: Key) ?*Screen. Null is nil.
func (self *ScreenSet) Get(key ScreenSetKey) *Screen {
	return self.all.Get(key)
}

// GetInit is ScreenSet.getInit.
//
// Get the screen for the given key, initializing it if necessary.
//
// Zig: pub fn getInit(self: *ScreenSet, alloc: Allocator, key: Key,
// opts: Screen.Options) Allocator.Error!*Screen. The allocator parameter is
// dropped. The error is ErrOutOfMemory (dcs.go).
func (self *ScreenSet) GetInit(key ScreenSetKey, opts ScreenOptions) (*Screen, error) {
	if screen := self.Get(key); screen != nil {
		return screen, nil
	}
	screen := new(Screen)
	// Zig: errdefer alloc.destroy(screen); Go memory.
	s, err := ScreenInit(opts)
	if err != nil {
		return nil, err
	}
	*screen = s
	self.all.Put(key, screen)
	return screen, nil
}

// Remove is ScreenSet.remove.
//
// Remove a key from the set. The primary screen cannot be removed (asserted).
//
// Zig: pub fn remove(self: *ScreenSet, alloc: Allocator, key: Key) void.
// The allocator parameter is dropped.
func (self *ScreenSet) Remove(key ScreenSetKey) {
	assert(key != ScreenSetKeyPrimary)
	if screen := self.all.FetchRemove(key); screen != nil {
		screen.Deinit()
		// Zig: alloc.destroy(screen); Go memory.
	}
}

// SwitchTo is ScreenSet.switchTo.
//
// Switch the active screen to the given key. Requires that the
// screen is initialized.
func (self *ScreenSet) SwitchTo(key ScreenSetKey) {
	self.ActiveKey = key
	active := self.all.Get(key)
	assert(active != nil) // .?
	self.Active = active
}
