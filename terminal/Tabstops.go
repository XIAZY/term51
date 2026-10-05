// Port of src/terminal/Tabstops.zig (Ghostty v1.3.1).
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
//     project test calls it, so it is kept as `Deinit`, which resets the
//     value to the zero Tabstops (Zig: `self.* = undefined`).
//
// Traps. Where Zig's safe build stops, the port panics: the `assert`s of
// `set`, `unset` and `get` (a column beyond the dynamic storage), and in
// `reset` the usize subtraction `self.cols - 1` with cols == 0 (and an
// interval above 0) and the addition `i += interval` if it overflows.
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

// We precompute all the possible masks since we never use a huge bit size.
//
// Zig computes it at compile time (`masks`); Go once at start-up.
var tabstopsMasks = func() [tabstopsUnitBits]tabstopsUnit {
	var res [tabstopsUnitBits]tabstopsUnit
	for i := range res {
		// Zig: @shlExact(@as(Unit, 1), @as(u3, @intCast(i))); i < 8, so
		// no bit is shifted out.
		res[i] = tabstopsUnit(1) << uint(i)
	}

	return res
}()

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
func tabstopsEntry(col uint) uint {
	return col / tabstopsUnitBits
}

// tabstopsIndex is Tabstops.index.
func tabstopsIndex(col uint) uint {
	// Zig: @mod on an unsigned value is %.
	return col % tabstopsUnitBits
}

// TabstopsInit is Tabstops.init.
//
// Zig: fn init(alloc: Allocator, cols: usize, interval: usize)
// Allocator.Error!Tabstops. The error is error.OutOfMemory (ErrOutOfMemory).
func TabstopsInit(cols uint, interval uint) (Tabstops, error) {
	res := Tabstops{}
	if err := res.Resize(cols); err != nil {
		return Tabstops{}, err
	}
	res.Reset(interval)
	return res, nil
}

// Deinit is Tabstops.deinit. Kept only because the tests call it; see the
// file header.
func (self *Tabstops) Deinit() {
	// Zig: if (self.dynamic_stops.len > 0) alloc.free(self.dynamic_stops);
	// the garbage collector frees.
	*self = Tabstops{} // Zig: self.* = undefined
}

// Set is Tabstops.set: set the tabstop at a certain column. The columns are
// 0-indexed.
func (self *Tabstops) Set(col uint) {
	i := tabstopsEntry(col)
	idx := tabstopsIndex(col)
	if i < tabstopsPreallocCount {
		self.prealloc_stops[i] |= tabstopsMasks[idx]
		return
	}

	dynamic_i := i - tabstopsPreallocCount
	assert(dynamic_i < uint(len(self.dynamic_stops)))
	self.dynamic_stops[dynamic_i] |= tabstopsMasks[idx]
}

// Unset is Tabstops.unset: unset the tabstop at a certain column. The
// columns are 0-indexed.
//
// Zig toggles the bit (`^=`), so unsetting a column without a stop sets
// one; the port keeps that.
func (self *Tabstops) Unset(col uint) {
	i := tabstopsEntry(col)
	idx := tabstopsIndex(col)
	if i < tabstopsPreallocCount {
		self.prealloc_stops[i] ^= tabstopsMasks[idx]
		return
	}

	dynamic_i := i - tabstopsPreallocCount
	assert(dynamic_i < uint(len(self.dynamic_stops)))
	self.dynamic_stops[dynamic_i] ^= tabstopsMasks[idx]
}

// Get is Tabstops.get: get the value of a tabstop at a specific column. The
// columns are 0-indexed.
//
// Zig takes self by value.
func (self Tabstops) Get(col uint) bool {
	i := tabstopsEntry(col)
	idx := tabstopsIndex(col)
	mask := tabstopsMasks[idx]
	var unit tabstopsUnit
	if i < tabstopsPreallocCount {
		unit = self.prealloc_stops[i]
	} else {
		// Zig: labeled block `unit:`.
		dynamic_i := i - tabstopsPreallocCount
		assert(dynamic_i < uint(len(self.dynamic_stops)))
		unit = self.dynamic_stops[dynamic_i]
	}

	return unit&mask == mask
}

// Resize is Tabstops.resize: resize this to support up to cols columns.
// TODO: needs interval to set new tabstops
//
// Zig: fn resize(self: *Tabstops, alloc: Allocator, cols: usize)
// Allocator.Error!void. The error is error.OutOfMemory (ErrOutOfMemory).
// Go's allocation cannot fail, so the result is always nil.
func (self *Tabstops) Resize(cols uint) error {
	// Zig: const tw = resize_tw; omitted (see the file header).

	if cols <= tabstopsPreallocColumns {
		self.cols = cols
		return nil
	}

	size := cols - tabstopsPreallocColumns
	if size < uint(len(self.dynamic_stops)) {
		self.cols = cols
		return nil
	}

	// Zig: try tw.check(.dynamic_alloc); omitted (see the file header).
	new := make([]tabstopsUnit, size) // Zig: try alloc.alloc(Unit, size)
	// Zig: errdefer comptime unreachable; nothing below can fail.
	// Zig: @memset(new, 0); make returns zeroed memory.
	if len(self.dynamic_stops) > 0 {
		// Zig: fastmem.copy(Unit, new, self.dynamic_stops), which copies
		// into new[0..dynamic_stops.len]; alloc.free(self.dynamic_stops) is
		// left to the garbage collector.
		copy(new[0:len(self.dynamic_stops)], self.dynamic_stops)
	}

	self.dynamic_stops = new
	self.cols = cols
	return nil
}

// Capacity is Tabstops.capacity: return the maximum number of columns this
// can support currently.
//
// Zig takes self by value.
func (self Tabstops) Capacity() uint {
	return (tabstopsPreallocCount + uint(len(self.dynamic_stops))) * tabstopsUnitBits
}

// Reset is Tabstops.reset: unset all tabstops and then reset the initial
// tabstops to the given interval. An interval of 0 sets no tabstops.
func (self *Tabstops) Reset(interval uint) {
	clear(self.prealloc_stops[:]) // Zig: @memset(&self.prealloc_stops, 0)
	clear(self.dynamic_stops)     // Zig: @memset(self.dynamic_stops, 0)

	if interval > 0 {
		// Zig: `self.cols - 1` is a usize subtraction and traps when cols
		// is 0.
		if self.cols == 0 {
			panic("integer overflow")
		}
		var i uint = interval
		for i < self.cols-1 {
			self.Set(i)
			// Zig: `i += interval` traps on overflow.
			if i > ^uint(0)-interval {
				panic("integer overflow")
			}
			i += interval
		}
	}
}
