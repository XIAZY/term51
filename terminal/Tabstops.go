// WIREFRAME for src/terminal/Tabstops.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// Keep track of the location of tabstops.
//
// This is implemented as a bit set. There is a preallocation segment that
// is used for almost all screen sizes. Then there is a dynamically allocated
// segment if the screen is larger than the preallocation amount.
//
// In reality, tabstops don't need to be the most performant in any metric.
// This implementation tries to balance denser memory usage (by using a bitset)
// and minimizing unnecessary allocations.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented) and every table is left at its zero value. The
// implementation is added only after the project owner and the project lead
// both approve it. Do not fill in bodies before then. See the header of
// color.go for the rules.
//
// File naming: Tabstops.zig is a file-level struct, so the type is
// `Tabstops` and its constructor is `TabstopsInit`. The file-level constants
// keep their Zig names (docs/mapping.md). The private type `Unit` is
// `tabstopsUnit` and the private free functions `entry` and `index` are
// `tabstopsEntry` and `tabstopsIndex`.
//
// Declarations appear in the same order as in Tabstops.zig.
//
// Go-memory rule (docs/mapping.md, governing rule 3): both storage fields
// are kept with their Zig names because behaviour depends on them
// (`capacity`, and which columns `set`/`get` may address). `dynamic_stops`
// is a Go slice; the allocator parameters of `init`, `deinit` and `resize`
// are dropped.
//
// Kept although the rules would drop it:
//   - `deinit` only frees `dynamic_stops` and marks the value undefined, so
//     under docs/mapping.md it has no Go counterpart. Every upstream and
//     project test calls it, so it is declared as the stub `Deinit`.
//
// Omitted:
//   - `resize_tw` (`tripwire.module(enum { dynamic_alloc }, resize)`) and
//     the `tw.check(.dynamic_alloc)` call inside `resize`: test-only fault
//     injection for the allocation in `resize`. There is no allocator in the
//     Go port to fail.
//
// The file has no C ABI declarations.

package terminal

// tabstopsUnit is Tabstops.Unit: the type we use per tabstop unit (see file
// docs).
type tabstopsUnit = uint8

// tabstopsUnitBits is @bitSizeOf(Unit).
const tabstopsUnitBits = 8

// The number of columns we preallocate for. This is kind of high which
// costs us some memory, but this is more columns than my 6k monitor at
// 12-point font size, so this should prevent allocation in almost all
// real world scenarios for the price of wasting at most
// (columns / sizeOf(Unit)) bytes.
const tabstopsPreallocColumns = 512

// The number of entries we need for our preallocation.
const tabstopsPreallocCount = tabstopsPreallocColumns / tabstopsUnitBits

// We precompute all the possible tabstopsMasks since we never use a huge bit size.
//
// Zig computes it at compile time: tabstopsMasks[i] is 1 << i. Wireframe: left
// zero; the implementation initialises it once at start-up.
var tabstopsMasks [tabstopsUnitBits]tabstopsUnit

// Tabstops is the file-level struct of Tabstops.zig.
type Tabstops struct {
	// The number of columns this tabstop is set to manage. Use resize()
	// to change this number.
	cols uint

	// Preallocated tab stops.
	prealloc_stops [tabstopsPreallocCount]tabstopsUnit

	// Dynamically expanded stops above prealloc stops.
	//
	// Zig: []Unit, default the empty slice. A nil Go slice has the same
	// length, 0.
	dynamic_stops []tabstopsUnit
}

// tabstopsEntry is Tabstops.entry: returns the entry in the stops array
// that would contain this column.
func tabstopsEntry(col uint) uint { panic(notImplemented) }

// tabstopsIndex is Tabstops.index.
func tabstopsIndex(col uint) uint { panic(notImplemented) }

// TabstopsInit is Tabstops.init.
//
// Zig: fn init(alloc: Allocator, cols: usize, interval: usize)
// Allocator.Error!Tabstops. The error is error.OutOfMemory (ErrOutOfMemory).
func TabstopsInit(cols uint, interval uint) (Tabstops, error) { panic(notImplemented) }

// Deinit is Tabstops.deinit. Kept only because the tests call it; see the
// file header.
func (self *Tabstops) Deinit() { panic(notImplemented) }

// Set is Tabstops.set: set the tabstop at a certain column. The columns are
// 0-indexed.
func (self *Tabstops) Set(col uint) { panic(notImplemented) }

// Unset is Tabstops.unset: unset the tabstop at a certain column. The
// columns are 0-indexed.
func (self *Tabstops) Unset(col uint) { panic(notImplemented) }

// Get is Tabstops.get: get the value of a tabstop at a specific column. The
// columns are 0-indexed.
//
// Zig takes self by value.
func (self Tabstops) Get(col uint) bool { panic(notImplemented) }

// Resize is Tabstops.resize: resize this to support up to cols columns.
// TODO: needs interval to set new tabstops
//
// Zig: fn resize(self: *Tabstops, alloc: Allocator, cols: usize)
// Allocator.Error!void. The error is error.OutOfMemory (ErrOutOfMemory).
func (self *Tabstops) Resize(cols uint) error { panic(notImplemented) }

// Capacity is Tabstops.capacity: return the maximum number of columns this
// can support currently.
//
// Zig takes self by value.
func (self Tabstops) Capacity() uint { panic(notImplemented) }

// Reset is Tabstops.reset: unset all tabstops and then reset the initial
// tabstops to the given interval. An interval of 0 sets no tabstops.
func (self *Tabstops) Reset(interval uint) { panic(notImplemented) }
