// Ported from src/terminal/ref_counted_set.zig (Ghostty v1.3.1).
//
// File naming: ref_counted_set.zig holds one declaration, the type function
// `RefCountedSet(T, IdT, RefCountInt, ContextT)`. Everything nested in the
// type it returns takes the prefix `RefCountedSet`.
//
// Declarations appear in the same order as in ref_counted_set.zig.
//
// # Generic form
//
// Zig's four comptime type parameters are Go type parameters:
//
//	RefCountedSet[T any, Id RefCountedSetInt, Ref RefCountedSetInt, C RefCountedSetContext[T]]
//
// `Id` is Zig's `IdT`, `Ref` is `RefCountInt`, `C` is `ContextT`. Go cannot
// attach a nested type or a static function to a generic type, so:
//   - `Item` is `RefCountedSetItem[T, Id, Ref]`, `Item.Metadata` is
//     `RefCountedSetItemMetadata[Id, Ref]`, `Layout` is
//     `RefCountedSetLayout[Id]`; each takes only the parameters it uses.
//   - `Layout.init(cap)` is the generic function
//     `RefCountedSetLayoutInit[Id](cap, sizes)`; `sizes` is explained under
//     "Byte arithmetic" below.
//   - `init(base, l, context)` is the generic function
//     `RefCountedSetInit[T, Id, Ref, C](l, context)`. Go cannot infer `T`
//     and `Ref` from the arguments, so a caller writes all four.
//   - `capacityForCount` and `load_factor` use none of the type parameters;
//     they are the plain function `RefCountedSetCapacityForCount` and the
//     plain constant `RefCountedSetLoadFactor`.
//   - The re-exports `Id` and `Context` are the type parameters themselves.
//
// # Context
//
// Zig's `Context` is any type with `hash` and `eql`, and optionally
// `deleted`, found by name at compile time. In Go it is the interface
// RefCountedSetContext[T], and the optional `deleted` is the separate
// interface RefCountedSetContextDeleted[T], which a context may also
// implement (as StreamHandlerVtRaw in stream.go). See the comments on both
// for the receiver.
//
// `@hasDecl(Context, "deleted")` is decided at compile time in Zig. The port
// decides it once, in RefCountedSetInit, and keeps the answer in the added
// field `has_deleted` (as stream.go keeps `vt_raw`). Where Zig calls
// `ctx.deleted(v)` behind `if (comptime @hasDecl(Context, "deleted"))`, Go
// tests `self.has_deleted` and calls Deleted through the interface
// conversion `any(ctx).(RefCountedSetContextDeleted[T])`, which Go needs to
// reach a method that is not in the constraint.
//
// # Go-memory rule, option A
//
// (docs/mapping.md, governing rule 3, as refined with the page wireframe:
// ordinary Go memory, but upstream's byte arithmetic is kept.) Zig stores
// the hash table and the items at byte offsets from a `base` pointer that
// every method takes. The Go set owns two Go slices instead:
//   - field `table: Offset(Id)` is `table []Id`;
//   - field `items: Offset(Item)` is `items []RefCountedSetItem[T, Id, Ref]`;
//   - EVERY method drops its `base` parameter and otherwise keeps its Zig
//     name, order, parameters and results;
//   - `init` drops its `base: OffsetBuf` parameter.
//
// Omitted under the Go-memory rule:
//   - `base_align` (alignment of the base pointer).
//
// # Byte arithmetic
//
// `Layout` keeps every Zig field. `cap`, `table_cap` and `table_mask` are
// counts that drive behaviour. `table_start`, `items_start` and `total_size`
// are byte positions: no Go memory is laid out by them, but Page.layout adds
// `total_size` into the page's byte size, against which the scrollback limit
// is counted, so the numbers must equal Zig's.
//
// `Layout.init` reads three numbers from types at compile time:
// @sizeOf(Id), @alignOf(Item) and @sizeOf(Item). @sizeOf(Id) follows from
// the type argument `Id` (1, 2 or 4 bytes; Go's width equals Zig's).
// @sizeOf(Item) and @alignOf(Item) are those of the ZIG item, which Go
// cannot derive; they are passed in as refCountedSetItemSizes, the one
// declaration in this file that is not a Zig declaration. The values for the
// two instantiations the page uses are in page.go (styleSetItemSizes,
// hyperlinkSetItemSizes).
//
// Widths: every byte size and byte offset is a `uint64`, whatever the
// platform word size, because the reference is the recorded 64-bit data and
// the target is 32-bit. Counts, indexes and lengths stay `uint` or their
// fixed Zig width.
//
// # Zero capacity
//
// A set laid out for capacity 0 has no table and no items. Zig's lookup
// still reads table[0] (`hash & 0`) from whatever memory follows; in the
// port a zero-capacity set is empty (docs/mapping.md): lookupContext finds
// nothing, and every add returns ErrOutOfMemory by upstream's own id check.
//
// # Run-time checks
//
// Zig traps on integer overflow in safe builds. Where an addition or
// subtraction on a count could overflow because of what a caller does (a
// ref count incremented past the maximum of its type, `living` going below
// zero), the port asserts before the operation, so that it stops where
// upstream stops instead of wrapping. Index checks are Go's own.
//
// The file has no C ABI declarations.

package terminal

import (
	"errors"
	"math"
	"math/bits"
)

// RefCountedSetInt is the set of integer types Zig instantiates
// RefCountedSet with as `IdT` and `RefCountInt` (u8, u16, u32).
type RefCountedSetInt interface {
	~uint8 | ~uint16 | ~uint32
}

// RefCountedSetContext is the `Context` parameter of RefCountedSet: a type
// containing methods to define behaviors.
//
//   - `fn hash(*Context, T) u64`    - Return a hash for an item.
//
//   - `fn eql(*Context, T, T) bool` - Check two items for equality.
//     The first of the two items passed in is guaranteed to be from
//     a value passed in to an `add` or `lookup` function, the second
//     is guaranteed to be a value already resident in the set.
//
// Receiver: the set holds the context by value (field `context: Context`)
// and the `*Context` functions take it by value too (`ctx: Context`); the
// calls are `ctx.hash(value)` and `ctx.eql(value, item.value)`, for which
// Zig passes the address of that copy. Every context in the Zig sources
// declares the receiver `*const`, so the context itself is never modified.
// In Go the methods are therefore declared with a VALUE receiver on the
// context type, and `C` is that type itself, not a pointer to it. A context
// that records something does so through a pointer field, as in Zig.
type RefCountedSetContext[T any] interface {
	Hash(value T) uint64
	Eql(a T, b T) bool
}

// RefCountedSetContextDeleted is the optional context function `deleted`,
// which Zig detects with `@hasDecl(Context, "deleted")`.
//
//   - `fn deleted(*Context, T) void` - [OPTIONAL] Deletion callback.
//     If present, called whenever an item is finally deleted.
//     Useful if the item has memory that needs to be freed.
//
// Zig calls it as `ctx.deleted(value)` in addContext, addWithIdContext,
// deleteItem, upsert and insert. Receiver: as for RefCountedSetContext.
type RefCountedSetContextDeleted[T any] interface {
	Deleted(value T)
}

// RefCountedSetLoadFactor is RefCountedSet.load_factor.
//
// This is the max load until the set returns OutOfMemory and
// requires more capacity.
//
// Experimentally, this load factor works quite well.
const RefCountedSetLoadFactor = 0.8125

// RefCountedSetCapacityForCount is RefCountedSet.capacityForCount: returns
// the minimum capacity needed to store `n` items, accounting for the load
// factor and the reserved ID 0.
//
// It uses none of the type parameters, so it is not generic.
func RefCountedSetCapacityForCount(n uint) uint {
	if n == 0 {
		return 0
	}
	// +1 because ID 0 is reserved, so we need at least n+1 slots.
	//
	// Zig: `n + 1` traps on overflow and @intFromFloat traps when the
	// result does not fit a usize. 0.8125 is 13/16 exactly, and an f64
	// division is correctly rounded in both languages.
	assert(n < math.MaxUint)
	c := math.Ceil(float64(n+1) / RefCountedSetLoadFactor)
	assert(c < 1<<bits.UintSize)
	return uint(c)
}

// RefCountedSetItem is RefCountedSet.Item: set item.
//
// Zig: `Item = .{}` has `value` undefined and `meta` at the defaults given
// on RefCountedSetItemMetadata. The Go zero value differs in `meta.bucket`.
type RefCountedSetItem[T any, Id RefCountedSetInt, Ref RefCountedSetInt] struct {
	// The value this item represents.
	//
	// Zig default: undefined.
	value T

	// Metadata for this item.
	//
	// Zig default: .{}.
	meta RefCountedSetItemMetadata[Id, Ref]
}

// RefCountedSetItemMetadata is RefCountedSet.Item.Metadata.
type RefCountedSetItemMetadata[Id RefCountedSetInt, Ref RefCountedSetInt] struct {
	// The bucket in the hash table where this item
	// is referenced.
	//
	// Zig default: std.math.maxInt(Id). The Go zero value is 0, so the
	// implementation must set it wherever Zig writes `.{}`.
	bucket Id

	// The length of the probe sequence between this
	// item's starting bucket and the bucket it's in,
	// used for Robin Hood hashing.
	//
	// Zig default: 0.
	psl Id

	// The reference count for this item.
	//
	// Zig default: 0.
	ref Ref
}

// RefCountedSetItemInit stands for Zig's `Item{}`: an item with every field
// at its declared default (meta.bucket = maxInt(Id), meta.psl = 0,
// meta.ref = 0; value undefined, zero in Go). Zig has no named constructor;
// Go needs one because its zero value has bucket 0.
func RefCountedSetItemInit[T any, Id RefCountedSetInt, Ref RefCountedSetInt]() RefCountedSetItem[T, Id, Ref] {
	return RefCountedSetItem[T, Id, Ref]{
		meta: RefCountedSetItemMetadata[Id, Ref]{
			bucket: ^Id(0), // std.math.maxInt(Id)
			psl:    0,
			ref:    0,
		},
	}
}

// RefCountedSet is the type returned by
// ref_counted_set.RefCountedSet(T, IdT, RefCountInt, ContextT): a reference
// counted set.
//
// This set is created with some capacity in mind. You can determine
// the exact memory requirement of a given capacity by calling `layout`
// and checking the total size.
//
// When the set exceeds capacity, an `OutOfMemory` or `NeedsRehash` error
// is returned from any memory-using methods. The caller is responsible
// for determining a path forward.
//
// This set is reference counted. Each item in the set has an associated
// reference count. The caller is responsible for calling release for an
// item when it is no longer being used. Items with 0 references will be
// kept until another item is written to their bucket. This allows items
// to be resurrected if they are re-added before they get overwritten.
//
// The backing data structure of this set is an open addressed hash table
// with linear probing and Robin Hood hashing, and a flat array of items.
//
// The table maps values to item IDs, which are indices in the item array
// which contain the item's value and its reference count. Item IDs can be
// used to efficiently access an item and update its reference count after
// it has been added to the table, to avoid having to use the hash map to
// look the value back up.
//
// ID 0 is reserved and will never be assigned.
//
// The Go zero value is not an initialised set (`next_id` would be 0 and the
// slices nil); use RefCountedSetInit.
type RefCountedSet[T any, Id RefCountedSetInt, Ref RefCountedSetInt, C RefCountedSetContext[T]] struct {
	// A hash table of item indices
	//
	// Zig: Offset(Id), `layout.table_cap` entries. See the file header.
	table []Id

	// By keeping track of the max probe sequence length
	// we can bail out early when looking up values that
	// aren't present.
	//
	// Zig default: 0.
	max_psl Id

	// We keep track of how many items have a PSL of any
	// given length, so that we can shrink max_psl when
	// we delete items.
	//
	// A probe sequence of length 32 or more is astronomically
	// unlikely. Roughly a (1/table_cap)^32 -- with any normal
	// table capacity that is so unlikely that it's not worth
	// handling.
	//
	// However, that assumes a uniform hash function, which
	// is not guaranteed and can be subverted with a crafted
	// input. We handle this gracefully by returning an error
	// anywhere where we're about to insert if there's any
	// item with a PSL in the last slot of the stats array.
	//
	// Zig default: all 0.
	psl_stats [32]Id

	// The backing store of items
	//
	// Zig: Offset(Item), `layout.cap` entries. See the file header.
	items []RefCountedSetItem[T, Id, Ref]

	// The number of living items currently stored in the set.
	//
	// Zig default: 0.
	living uint

	// The next index to store an item at.
	// Id 0 is reserved for unused items.
	//
	// Zig default: 1. The Go zero value is 0, so RefCountedSetInit must
	// set it.
	next_id Id

	layout RefCountedSetLayout[Id]

	// An instance of the context structure.
	context C

	// Zig: no field. `@hasDecl(Context, "deleted")` is decided at compile
	// time; the port decides it once, in RefCountedSetInit (mapping:
	// compile-time work is done once). True when C implements
	// RefCountedSetContextDeleted[T].
	has_deleted bool
}

// refCountedSetItemSizes is NOT a Zig declaration. It carries what Zig reads
// from the item type at compile time in Layout.init: @sizeOf(Item) and
// @alignOf(Item), in bytes. See the file header.
type refCountedSetItemSizes struct {
	item_size  uint64
	item_align uint64
}

// RefCountedSetLayout is RefCountedSet.Layout.
//
// `cap` and `table_cap` are counts (Zig: usize). `table_start`,
// `items_start` and `total_size` are byte positions (Zig: usize) and are
// uint64; see the file header.
type RefCountedSetLayout[Id RefCountedSetInt] struct {
	cap         uint
	table_cap   uint
	table_mask  Id
	table_start uint64
	items_start uint64
	total_size  uint64
}

// RefCountedSetLayoutInit is RefCountedSet.Layout.init.
//
// Returns the memory layout for the given base offset and
// desired capacity. The layout can be used by the caller to
// determine how much memory to allocate, and the layout must
// be used to initialize the set so that the set knows all
// the offsets for the various buffers.
//
// The capacity passed for cap will be used for the hash table,
// which has a load factor of `0.8125` (13/16), so the number of
// items which can actually be stored in the set will be smaller.
//
// The laid out capacity will be at least `cap`, but may be higher,
// since it is rounded up to the next power of 2 for efficiency.
//
// The returned layout `cap` property will be 1 more than the number
// of items that the set can actually store, since ID 0 is reserved.
//
// Zig: `Layout.init(cap: usize) Layout`, a function of the set type. Of the
// set's type parameters it uses `Id` (the assertion
// `cap <= maxInt(Id) + 1`, the type of `table_mask` and @sizeOf(Id)) and the
// size and alignment of `Item`, which `sizes` stands for; see the file
// header. So it is a generic function over `Id` alone, called as
// `RefCountedSetLayoutInit[StyleId](16, styleSetItemSizes)`.
func RefCountedSetLayoutInit[Id RefCountedSetInt](cap uint, sizes refCountedSetItemSizes) RefCountedSetLayout[Id] {
	assert(uint64(cap) <= uint64(^Id(0))+1)

	// Zero-cap set is valid, return special case
	if cap == 0 {
		return RefCountedSetLayout[Id]{
			cap:         0,
			table_cap:   0,
			table_mask:  0,
			table_start: 0,
			items_start: 0,
			total_size:  0,
		}
	}

	// std.math.ceilPowerOfTwoAssert(usize, cap), which asserts that the
	// result fits a usize (only reachable with a 32-bit uint and a u32 Id).
	assert(bits.Len(cap-1) < bits.UintSize)
	table_cap := uint(1) << bits.Len(cap-1)
	// Zig: @intFromFloat(load_factor * @as(f64, @floatFromInt(table_cap))).
	// One product, truncated toward zero; the explicit float64() keeps Go
	// from fusing it with anything.
	items_cap := uint(float64(RefCountedSetLoadFactor * float64(table_cap)))

	// Zig: @intCast((@as(usize, 1) << std.math.log2_int(usize, table_cap)) - 1);
	// log2_int is the index of the highest set bit.
	mask := (uint(1) << (bits.Len(table_cap) - 1)) - 1
	assert(uint64(mask) <= uint64(^Id(0))) // @intCast
	table_mask := Id(mask)

	// @sizeOf(Id): Go's width of Id equals Zig's (1, 2 or 4 bytes).
	id_size := uint64(bits.Len64(uint64(^Id(0))) / 8)

	const table_start = 0
	table_end := table_start + uint64(table_cap)*id_size

	// std.mem.alignForward(usize, table_end, @alignOf(Item)), which asserts
	// that the alignment is a power of two.
	item_align := sizes.item_align
	assert(item_align != 0 && item_align&(item_align-1) == 0)
	items_start := (table_end + (item_align - 1)) &^ (item_align - 1)
	items_end := items_start + uint64(items_cap)*sizes.item_size

	total_size := items_end

	return RefCountedSetLayout[Id]{
		cap:         items_cap,
		table_cap:   table_cap,
		table_mask:  table_mask,
		table_start: table_start,
		items_start: items_start,
		total_size:  total_size,
	}
}

// RefCountedSetInit is RefCountedSet.init.
//
// Zig: fn init(base: OffsetBuf, l: Layout, context: Context) Self. The
// buffer argument is dropped: the set makes its own `table` (`l.table_cap`
// entries, all 0) and `items` (`l.cap` entries, all at the Zig default
// `.{}`).
//
// Go cannot infer `T` and `Ref`, so all four type arguments are written:
// RefCountedSetInit[Style, StyleId, SizeCellCountInt, StyleSetContext](l, ctx).
func RefCountedSetInit[T any, Id RefCountedSetInt, Ref RefCountedSetInt, C RefCountedSetContext[T]](
	l RefCountedSetLayout[Id],
	context C,
) RefCountedSet[T, Id, Ref, C] {
	// @memset(table.ptr(base)[0..l.table_cap], 0);
	table := make([]Id, l.table_cap)
	// @memset(items.ptr(base)[0..l.cap], .{});
	items := make([]RefCountedSetItem[T, Id, Ref], l.cap)
	for i := range items {
		items[i] = RefCountedSetItemInit[T, Id, Ref]()
	}

	// Zig: `@hasDecl(Context, "deleted")`, decided once here.
	_, has_deleted := any(context).(RefCountedSetContextDeleted[T])

	return RefCountedSet[T, Id, Ref, C]{
		table:       table,
		items:       items,
		layout:      l,
		context:     context,
		next_id:     1, // Zig default
		has_deleted: has_deleted,
	}
}

// RefCountedSet.AddError: possible errors for `add` and `addWithId`. An
// error set is not a type in Go; its members are the sentinels
// ErrOutOfMemory (dcs.go) and ErrNeedsRehash.
//
//   - OutOfMemory: there is not enough memory to add a new item.
//     Remove items or grow and reinitialize.
//   - NeedsRehash: the set needs to be rehashed, as there are many dead
//     items with lower IDs which are inaccessible for reuse.

// ErrNeedsRehash is Zig's error.NeedsRehash (RefCountedSet.AddError).
var ErrNeedsRehash = errors.New("NeedsRehash")

// Add is RefCountedSet.add: add an item to the set if not present and
// increment its ref count.
//
// Returns the item's ID.
//
// If the set has no more room, then an OutOfMemory error is returned.
func (self *RefCountedSet[T, Id, Ref, C]) Add(value T) (Id, error) {
	return self.AddContext(value, self.context)
}

// AddContext is RefCountedSet.addContext.
func (self *RefCountedSet[T, Id, Ref, C]) AddContext(value T, ctx C) (Id, error) {
	items := self.items

	// Trim dead items from the end of the list.
	for self.next_id > 1 && items[self.next_id-1].meta.ref == 0 {
		self.next_id -= 1
		self.deleteItem(self.next_id, ctx)
	}

	// If the item already exists, return it.
	if id := self.LookupContext(value, ctx); id.Valid {
		// Notify the context that the value is "deleted" because
		// we're reusing the existing value in the set. This allows
		// callers to clean up any resources associated with the value.
		if self.has_deleted {
			any(ctx).(RefCountedSetContextDeleted[T]).Deleted(value)
		}

		assert(items[id.Value].meta.ref != ^Ref(0)) // Zig: += traps on overflow
		items[id.Value].meta.ref += 1
		return id.Value, nil
	}

	// While it should be statistically impossible to exceed the
	// bounds of `psl_stats`, the hash function is not perfect and
	// in such a case we want to remain stable. If we're about to
	// insert an item and there's something with a PSL of `len - 1`,
	// we may end up with a PSL of `len` which would exceed the bounds.
	// In such a case, we claim to be out of memory.
	if self.psl_stats[len(self.psl_stats)-1] > 0 {
		// Zig: @branchHint(.cold)
		return 0, ErrOutOfMemory
	}

	// If the item doesn't exist, we need an available ID.
	if uint(self.next_id) >= self.layout.cap {
		// Arbitrarily chosen, threshold for rehashing.
		// If less than 90% of currently allocated IDs
		// correspond to living items, we should rehash.
		// Otherwise, claim we're out of memory because
		// we assume that we'll end up running out of
		// memory or rehashing again very soon if we
		// rehash with only a few IDs left.
		const rehash_threshold = 0.9
		// Zig: @as(Id, @intFromFloat(...)): truncated toward zero, and
		// a trap if it does not fit Id.
		threshold := float64(float64(self.layout.cap) * rehash_threshold)
		assert(threshold < float64(uint64(^Id(0)))+1)
		if self.living < uint(Id(threshold)) {
			return 0, ErrNeedsRehash
		}

		// If we don't have at least 10% dead items then
		// we claim we're out of memory.
		return 0, ErrOutOfMemory
	}

	id := self.insert(value, self.next_id, ctx)
	assert(items[id].meta.ref != ^Ref(0)) // Zig: += traps on overflow
	items[id].meta.ref += 1
	assert(items[id].meta.ref == 1)
	self.living += 1

	// Its possible insert returns a different ID by reusing a
	// dead item so we only need to update next id if we used it.
	if id == self.next_id {
		self.next_id += 1
	}

	return id, nil
}

// AddWithId is RefCountedSet.addWithId: add an item to the set if not
// present and increment its ref count. If possible, use the provided ID.
//
// Returns the item's ID, or null if the provided ID was used.
//
// If the set has no more room, then an OutOfMemory error is returned.
//
// Zig: AddError!?Id.
func (self *RefCountedSet[T, Id, Ref, C]) AddWithId(value T, id Id) (Optional[Id], error) {
	return self.AddWithIdContext(value, id, self.context)
}

// AddWithIdContext is RefCountedSet.addWithIdContext.
func (self *RefCountedSet[T, Id, Ref, C]) AddWithIdContext(value T, id Id, ctx C) (Optional[Id], error) {
	items := self.items

	assert(id > 0)

	if id < self.next_id {
		if items[id].meta.ref == 0 {
			// See comment in `addContext` for details.
			if self.psl_stats[len(self.psl_stats)-1] > 0 {
				// Zig: @branchHint(.cold)
				return Optional[Id]{}, ErrOutOfMemory
			}

			self.deleteItem(id, ctx)

			added_id := self.upsert(value, id, ctx)

			assert(items[added_id].meta.ref != ^Ref(0)) // Zig: += traps on overflow
			items[added_id].meta.ref += 1

			self.living += 1

			if added_id == id {
				return None[Id](), nil
			}
			return Some(added_id), nil
		} else if ctx.Eql(value, items[id].value) {
			// Notify the context that the value is "deleted" because
			// we're reusing the existing value in the set. This allows
			// callers to clean up any resources associated with the value.
			if self.has_deleted {
				any(ctx).(RefCountedSetContextDeleted[T]).Deleted(value)
			}

			assert(items[id].meta.ref != ^Ref(0)) // Zig: += traps on overflow
			items[id].meta.ref += 1

			return None[Id](), nil
		}
	}

	added, err := self.AddContext(value, ctx)
	if err != nil {
		return Optional[Id]{}, err
	}
	return Some(added), nil
}

// Use is RefCountedSet.use: increment an item's reference count by 1.
//
// Asserts that the item's reference count is greater than 0.
//
// Zig: self is *const Self; the count it changes lives in the items memory.
func (self *RefCountedSet[T, Id, Ref, C]) Use(id Id) {
	assert(id > 0)
	assert(uint(id) < self.layout.cap)

	items := self.items
	item := &items[id]

	// If `use` is being called on an item with 0 references, then
	// either someone forgot to call it before, released too early
	// or lied about releasing. In any case something is wrong and
	// shouldn't be allowed.
	assert(item.meta.ref > 0)

	assert(item.meta.ref != ^Ref(0)) // Zig: += traps on overflow
	item.meta.ref += 1
}

// UseMultiple is RefCountedSet.useMultiple: increment an item's reference
// count by a specified number.
//
// Asserts that the item's reference count is greater than 0.
//
// Zig: self is *const Self.
func (self *RefCountedSet[T, Id, Ref, C]) UseMultiple(id Id, n Ref) {
	assert(id > 0)
	assert(uint(id) < self.layout.cap)

	items := self.items
	item := &items[id]

	// If `use` is being called on an item with 0 references, then
	// either someone forgot to call it before, released too early
	// or lied about releasing. In any case something is wrong and
	// shouldn't be allowed.
	assert(item.meta.ref > 0)

	assert(item.meta.ref <= ^Ref(0)-n) // Zig: += traps on overflow
	item.meta.ref += n
}

// Get is RefCountedSet.get: get an item by its ID without incrementing its
// reference count.
//
// Asserts that the item's reference count is greater than 0.
//
// Zig: self is *const Self; returns *T, a pointer to the value stored in
// the item.
func (self *RefCountedSet[T, Id, Ref, C]) Get(id Id) *T {
	assert(id > 0)
	assert(uint(id) < self.layout.cap)

	items := self.items
	item := &items[id]

	assert(item.meta.ref > 0)

	return &item.value
}

// Release is RefCountedSet.release: releases a reference to an item by its
// ID.
//
// Asserts that the item's reference count is greater than 0.
func (self *RefCountedSet[T, Id, Ref, C]) Release(id Id) {
	assert(id > 0)
	assert(uint(id) < self.layout.cap)

	items := self.items
	item := &items[id]

	assert(item.meta.ref > 0)
	item.meta.ref -= 1
	if item.meta.ref == 0 {
		assert(self.living > 0) // Zig: -= traps on overflow
		self.living -= 1
	}
}

// ReleaseMultiple is RefCountedSet.releaseMultiple: release a specified
// number of references to an item by its ID.
//
// Asserts that the item's reference count is at least `n`.
//
// Zig: `n: Id` (the id type, not the ref count type).
func (self *RefCountedSet[T, Id, Ref, C]) ReleaseMultiple(id Id, n Id) {
	assert(id > 0)
	assert(uint(id) < self.layout.cap)

	items := self.items
	item := &items[id]

	assert(uint64(item.meta.ref) >= uint64(n))
	item.meta.ref -= Ref(n)

	if item.meta.ref == 0 {
		assert(self.living > 0) // Zig: -= traps on overflow
		self.living -= 1
	}
}

// RefCount is RefCountedSet.refCount: get the ref count for an item by its
// ID.
//
// Zig: self is *const Self.
func (self *RefCountedSet[T, Id, Ref, C]) RefCount(id Id) Ref {
	assert(id > 0)
	assert(uint(id) < self.layout.cap)

	items := self.items
	item := &items[id]
	return item.meta.ref
}

// Count is RefCountedSet.count: get the current number of non-dead items in
// the set.
//
// Zig: self is *const Self.
func (self *RefCountedSet[T, Id, Ref, C]) Count() uint {
	return self.living
}

// deleteItem is RefCountedSet.deleteItem: delete an item, removing any
// references from the table, and freeing its ID to be reused.
func (self *RefCountedSet[T, Id, Ref, C]) deleteItem(id Id, ctx C) {
	table := self.table
	items := self.items

	item := items[id]

	if uint(item.meta.bucket) > self.layout.table_cap {
		return
	}

	assert(table[item.meta.bucket] == id)

	if self.has_deleted {
		// Inform the context struct that we're
		// deleting the dead item's value for good.
		any(ctx).(RefCountedSetContextDeleted[T]).Deleted(item.value)
	}

	self.psl_stats[item.meta.psl] -= 1
	table[item.meta.bucket] = 0
	items[id] = RefCountedSetItemInit[T, Id, Ref]()

	var p Id = item.meta.bucket
	var n Id = (p + 1) & self.layout.table_mask // Zig: +%, wrapping

	for table[n] != 0 && items[table[n]].meta.psl > 0 {
		items[table[n]].meta.bucket = p
		self.psl_stats[items[table[n]].meta.psl] -= 1
		items[table[n]].meta.psl -= 1
		self.psl_stats[items[table[n]].meta.psl] += 1
		table[p] = table[n]
		p = n
		n = (p + 1) & self.layout.table_mask // Zig: +%, wrapping
	}

	for self.max_psl > 0 && self.psl_stats[self.max_psl] == 0 {
		self.max_psl -= 1
	}

	table[p] = 0

	self.assertIntegrity(ctx)
}

// Lookup is RefCountedSet.lookup: find an item in the table and return its
// ID. If the item does not exist in the table, null is returned.
//
// Zig: self is *const Self; returns ?Id.
func (self *RefCountedSet[T, Id, Ref, C]) Lookup(value T) Optional[Id] {
	return self.LookupContext(value, self.context)
}

// LookupContext is RefCountedSet.lookupContext.
//
// Zig: self is *const Self; returns ?Id.
func (self *RefCountedSet[T, Id, Ref, C]) LookupContext(value T, ctx C) Optional[Id] {
	table := self.table
	items := self.items

	// Go only: a zero-capacity set has no table; Zig reads table[0] from
	// the memory that follows. The port's zero-capacity set is empty (see
	// the file header).
	if self.layout.table_cap == 0 {
		return None[Id]()
	}

	var hash uint64 = ctx.Hash(value)

	for i := uint(0); i < uint(self.max_psl)+1; i++ {
		p := uint((hash + uint64(i)) & uint64(self.layout.table_mask)) // Zig: +%, wrapping
		id := table[p]

		// Empty bucket, our item cannot have probed to
		// any point after this, meaning it's not present.
		if id == 0 {
			return None[Id]()
		}

		item := items[id]

		// An item with a shorter probe sequence length would never
		// end up in the middle of another sequence, since it would
		// be swapped out if inserted before the new sequence, and
		// would not be swapped in if inserted afterwards.
		//
		// As such, our item cannot be present.
		if uint(item.meta.psl) < i {
			return None[Id]()
		}

		// If the item is a part of the same probe sequence,
		// we make sure it's not dead and then check to see
		// if it matches the value we're looking for.
		if uint(item.meta.psl) == i &&
			item.meta.ref > 0 &&
			ctx.Eql(value, item.value) {
			return Some(id)
		}
	}

	return None[Id]()
}

// upsert is RefCountedSet.upsert: find the provided value in the hash
// table, or add a new item for it if not present. If a new item is added,
// `new_id` will be used as the ID. If an existing item is found, the
// `new_id` is ignored and the existing item's ID is returned.
func (self *RefCountedSet[T, Id, Ref, C]) upsert(value T, new_id Id, ctx C) Id {
	// If the item already exists, return it.
	if id := self.LookupContext(value, ctx); id.Valid {
		// Notify the context that the value is "deleted" because
		// we're reusing the existing value in the set. This allows
		// callers to clean up any resources associated with the value.
		if self.has_deleted {
			any(ctx).(RefCountedSetContextDeleted[T]).Deleted(value)
		}

		return id.Value
	}

	return self.insert(value, new_id, ctx)
}

// insert is RefCountedSet.insert: insert the given value into the hash
// table with the given ID.
//
// If runtime safety is enabled, asserts that
// the value is not already present in the table.
func (self *RefCountedSet[T, Id, Ref, C]) insert(value T, new_id Id, ctx C) Id {
	// Zig: `if (comptime std.debug.runtime_safety)`; the reference is a
	// safe build.
	assert(!self.LookupContext(value, ctx).Valid)

	table := self.table
	items := self.items

	// The new item that we'll put in to the table.
	new_item := RefCountedSetItem[T, Id, Ref]{
		value: value,
		meta: RefCountedSetItemMetadata[Id, Ref]{
			bucket: ^Id(0), // Zig default, std.math.maxInt(Id)
			psl:    0,
			ref:    0,
		},
	}

	var hash uint64 = ctx.Hash(value)

	var held_id Id = new_id
	var held_item *RefCountedSetItem[T, Id, Ref] = &new_item

	var chosen_id Id = new_id

	for i := uint(0); i < self.layout.table_cap-1; i++ {
		p := Id((hash + uint64(i)) & uint64(self.layout.table_mask)) // Zig: +%, wrapping
		id := table[p]

		// Empty bucket, put our held item in to it and break.
		if id == 0 {
			table[p] = held_id
			held_item.meta.bucket = p
			self.psl_stats[held_item.meta.psl] += 1
			self.max_psl = max(self.max_psl, held_item.meta.psl)
			break
		}

		item := &items[id]

		// If there's a dead item then we resurrect it
		// for our value so that we can reuse its ID,
		// unless its ID is greater than the one we're
		// given (i.e. prefer smaller IDs).
		if item.meta.ref == 0 {
			// Dead items aren't super common relative
			// to other places to insert/swap the held
			// item in to the set.
			// Zig: @branchHint(.unlikely)

			if self.has_deleted {
				// Inform the context struct that we're
				// deleting the dead item's value for good.
				any(ctx).(RefCountedSetContextDeleted[T]).Deleted(item.value)
			}

			// Reap the dead item.
			self.psl_stats[item.meta.psl] -= 1
			*item = RefCountedSetItemInit[T, Id, Ref]()

			// Only resurrect this item if it has a
			// smaller id than the one we were given.
			if id < new_id {
				chosen_id = id
			}

			// Put the currently held item in to the
			// bucket of the item that we just reaped.
			table[p] = held_id
			held_item.meta.bucket = p
			self.psl_stats[held_item.meta.psl] += 1
			self.max_psl = max(self.max_psl, held_item.meta.psl)

			break
		}

		// If this item has a lower PSL, or has equal PSL and lower ref
		// count, then we swap it out with our held item. By doing this,
		// items with high reference counts are prioritized for earlier
		// placement. The assumption is that an item which has a higher
		// reference count will be accessed more frequently, so we want
		// to minimize the time it takes to find it.
		if item.meta.psl < held_item.meta.psl ||
			item.meta.psl == held_item.meta.psl &&
				item.meta.ref < held_item.meta.ref {
			// Put our held item in the bucket.
			table[p] = held_id
			held_item.meta.bucket = p
			self.psl_stats[held_item.meta.psl] += 1
			self.max_psl = max(self.max_psl, held_item.meta.psl)

			// Pick up the item that has a lower PSL.
			held_id = id
			held_item = item
			self.psl_stats[item.meta.psl] -= 1
		}

		// Advance to the next probe position for our held item.
		assert(held_item.meta.psl != ^Id(0)) // Zig: += traps on overflow
		held_item.meta.psl += 1
	}

	// Our chosen ID may have changed if we decided
	// to reuse a dead item's ID, so we make sure
	// the chosen bucket contains the correct ID.
	table[new_item.meta.bucket] = chosen_id

	// Finally place our new item in to our array.
	items[chosen_id] = new_item

	self.assertIntegrity(ctx)

	return chosen_id
}

// assertIntegrity is RefCountedSet.assertIntegrity.
//
// Zig: self is *const Self. Upstream's body is disabled
// (`if (false and std.debug.runtime_safety)`), so it does nothing.
func (self *RefCountedSet[T, Id, Ref, C]) assertIntegrity(ctx C) {
	// Disabled because this is excessively slow, only enable
	// if debugging a RefCountedSet issue or modifying its logic.
	//
	// Zig: `if (false and std.debug.runtime_safety) { ... }`; the body
	// is never compiled into a build, so it is not ported.
	_ = ctx
}
