// WIREFRAME for src/terminal/ref_counted_set.zig (Ghostty v1.3.1). NOT YET
// PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
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
//     `RefCountedSetLayoutInit[Id](cap)`.
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
// # Go-memory rule
//
// (docs/mapping.md, governing rule 3.) Zig stores the hash table and the
// items at byte offsets from a `base` pointer that every method takes. The
// Go set owns two Go slices instead:
//   - field `table: Offset(Id)` is `table []Id`;
//   - field `items: Offset(Item)` is `items []RefCountedSetItem[T, Id, Ref]`;
//   - EVERY method drops its `base` parameter and otherwise keeps its Zig
//     name, order, parameters and results;
//   - `init` drops its `base: OffsetBuf` parameter.
//
// Omitted under the Go-memory rule, because they exist only to compute a
// byte layout:
//   - `base_align`;
//   - `Layout.table_start`, `Layout.items_start`, `Layout.total_size`.
//
// `Layout.cap`, `Layout.table_cap` and `Layout.table_mask` are counts that
// drive behaviour and are kept.
//
// The file has no C ABI declarations.

package terminal

import "errors"

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
func RefCountedSetCapacityForCount(n uint) uint { panic(notImplemented) }

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
	panic(notImplemented)
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
}

// RefCountedSetLayout is RefCountedSet.Layout.
//
// Zig's `table_start`, `items_start` and `total_size` are byte positions and
// are omitted; see the file header.
type RefCountedSetLayout[Id RefCountedSetInt] struct {
	cap        uint
	table_cap  uint
	table_mask Id
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
// set's type parameters it uses only `Id` (the assertion
// `cap <= maxInt(Id) + 1` and the type of `table_mask`), so it is a generic
// function over `Id` alone, called as `RefCountedSetLayoutInit[uint16](16)`.
func RefCountedSetLayoutInit[Id RefCountedSetInt](cap uint) RefCountedSetLayout[Id] {
	panic(notImplemented)
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
	panic(notImplemented)
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
func (self *RefCountedSet[T, Id, Ref, C]) Add(value T) (Id, error) { panic(notImplemented) }

// AddContext is RefCountedSet.addContext.
func (self *RefCountedSet[T, Id, Ref, C]) AddContext(value T, ctx C) (Id, error) {
	panic(notImplemented)
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
	panic(notImplemented)
}

// AddWithIdContext is RefCountedSet.addWithIdContext.
func (self *RefCountedSet[T, Id, Ref, C]) AddWithIdContext(value T, id Id, ctx C) (Optional[Id], error) {
	panic(notImplemented)
}

// Use is RefCountedSet.use: increment an item's reference count by 1.
//
// Asserts that the item's reference count is greater than 0.
//
// Zig: self is *const Self; the count it changes lives in the items memory.
func (self *RefCountedSet[T, Id, Ref, C]) Use(id Id) { panic(notImplemented) }

// UseMultiple is RefCountedSet.useMultiple: increment an item's reference
// count by a specified number.
//
// Asserts that the item's reference count is greater than 0.
//
// Zig: self is *const Self.
func (self *RefCountedSet[T, Id, Ref, C]) UseMultiple(id Id, n Ref) { panic(notImplemented) }

// Get is RefCountedSet.get: get an item by its ID without incrementing its
// reference count.
//
// Asserts that the item's reference count is greater than 0.
//
// Zig: self is *const Self; returns *T, a pointer to the value stored in
// the item.
func (self *RefCountedSet[T, Id, Ref, C]) Get(id Id) *T { panic(notImplemented) }

// Release is RefCountedSet.release: releases a reference to an item by its
// ID.
//
// Asserts that the item's reference count is greater than 0.
func (self *RefCountedSet[T, Id, Ref, C]) Release(id Id) { panic(notImplemented) }

// ReleaseMultiple is RefCountedSet.releaseMultiple: release a specified
// number of references to an item by its ID.
//
// Asserts that the item's reference count is at least `n`.
//
// Zig: `n: Id` (the id type, not the ref count type).
func (self *RefCountedSet[T, Id, Ref, C]) ReleaseMultiple(id Id, n Id) { panic(notImplemented) }

// RefCount is RefCountedSet.refCount: get the ref count for an item by its
// ID.
//
// Zig: self is *const Self.
func (self *RefCountedSet[T, Id, Ref, C]) RefCount(id Id) Ref { panic(notImplemented) }

// Count is RefCountedSet.count: get the current number of non-dead items in
// the set.
//
// Zig: self is *const Self.
func (self *RefCountedSet[T, Id, Ref, C]) Count() uint { panic(notImplemented) }

// deleteItem is RefCountedSet.deleteItem: delete an item, removing any
// references from the table, and freeing its ID to be reused.
func (self *RefCountedSet[T, Id, Ref, C]) deleteItem(id Id, ctx C) { panic(notImplemented) }

// Lookup is RefCountedSet.lookup: find an item in the table and return its
// ID. If the item does not exist in the table, null is returned.
//
// Zig: self is *const Self; returns ?Id.
func (self *RefCountedSet[T, Id, Ref, C]) Lookup(value T) Optional[Id] { panic(notImplemented) }

// LookupContext is RefCountedSet.lookupContext.
//
// Zig: self is *const Self; returns ?Id.
func (self *RefCountedSet[T, Id, Ref, C]) LookupContext(value T, ctx C) Optional[Id] {
	panic(notImplemented)
}

// upsert is RefCountedSet.upsert: find the provided value in the hash
// table, or add a new item for it if not present. If a new item is added,
// `new_id` will be used as the ID. If an existing item is found, the
// `new_id` is ignored and the existing item's ID is returned.
func (self *RefCountedSet[T, Id, Ref, C]) upsert(value T, new_id Id, ctx C) Id {
	panic(notImplemented)
}

// insert is RefCountedSet.insert: insert the given value into the hash
// table with the given ID.
//
// If runtime safety is enabled, asserts that
// the value is not already present in the table.
func (self *RefCountedSet[T, Id, Ref, C]) insert(value T, new_id Id, ctx C) Id {
	panic(notImplemented)
}

// assertIntegrity is RefCountedSet.assertIntegrity.
//
// Zig: self is *const Self. Upstream's body is disabled
// (`if (false and std.debug.runtime_safety)`), so it does nothing.
func (self *RefCountedSet[T, Id, Ref, C]) assertIntegrity(ctx C) { panic(notImplemented) }
