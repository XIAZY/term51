// WIREFRAME for src/terminal/hash_map.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// This file contains a fork of the Zig stdlib HashMap implementation tuned
// for use with our terminal page representation.
//
// The main goal we need to achieve that wasn't possible with the stdlib
// HashMap is to utilize offsets rather than full pointers so that we can
// copy around the entire backing memory and keep the hash map working.
//
// Additionally, for serialization/deserialization purposes, we need to be
// able to create a HashMap instance and manually set the offsets up. The
// stdlib HashMap does not export Metadata so this isn't possible.
//
// Also, I want to be able to understand possible capacity for a given K,V
// type and fixed memory amount. The stdlib HashMap doesn't publish its
// internal allocation size calculation.
//
// Finally, I removed many of the APIs that we'll never require for our
// usage just so that this file is smaller, easier to understand, and has
// less opportunity for bugs.
//
// Besides these shortcomings, the stdlib HashMap has some great qualities
// that we want to keep, namely the fact that it is backed by a single large
// allocation rather than pointers to separate allocations. This is important
// because our terminal page representation is backed by a single large
// allocation so we can give the HashMap a slice of memory to operate in.
//
// I haven't carefully benchmarked this implementation against other hash
// map implementations. It's possible using some of the newer variants out
// there would be better. However, I trust the built-in version is pretty good
// and its more important to get the terminal page representation working
// first then we can measure and improve this later if we find it to be a
// bottleneck.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
//
// # File naming
//
// hash_map.zig is a namespace file, prefix `HashMap` (`hashMap` for
// declarations that are private in Zig):
//
//	AutoOffsetHashMap(K, V)            HashMapAutoOffsetHashMap[K, V]
//	AutoHashMapUnmanaged(K, V)         hashMapAutoHashMapUnmanaged[K, V]
//	AutoContext(K)                     hashMapAutoContext[K]
//	OffsetHashMap(K, V, Context)       HashMapOffsetHashMap[K, V, C]
//	HashMapUnmanaged(K, V, Context)    hashMapUnmanaged[K, V, C]
//
// `HashMapUnmanaged` already starts with the file's name, so the prefix is
// not doubled, and what is nested in it takes the prefix `HashMap` /
// `hashMap`: `Size` is `HashMapSize`, `Entry` is `HashMapEntry`, `Metadata`
// is `hashMapMetadata`, `Layout` is `hashMapLayout`. What is nested in
// `OffsetHashMap` takes `HashMapOffsetHashMap`.
//
// Declarations appear in the same order as in hash_map.zig.
//
// # Generic form
//
// Go cannot attach a nested type or a receiver-less function to a generic
// type, so:
//   - nested types take only the type parameters they use
//     (`HashMapEntry[K, V]`, `HashMapIterator[K, V, C]`, `hashMapLayout`);
//   - `init` is the generic function `hashMapUnmanagedInit[K, V, C](layout)`
//     (and `HashMapOffsetHashMapInit[K, V, C](l)`); Go cannot infer the type
//     arguments, so a caller writes all three;
//   - `layoutForCapacity` and `OffsetHashMap.layout` use no Go type
//     parameter and are plain functions; see "Byte arithmetic" below. The
//     function `OffsetHashMap.layout` would have the same Go name as the
//     type `OffsetHashMap.Layout`, so the function is
//     `HashMapOffsetHashMapLayoutOf`.
//   - the `...Adapted` functions take `key: anytype, ctx: anytype`. Every
//     caller in the file passes a `K` and a `Context`, and Go methods cannot
//     have type parameters of their own, so they take `key K, ctx C`.
//
// # Context
//
// Zig's `Context` is any type with `hash(self, K) u64` and
// `eql(self, K, K) bool`. In Go it is the interface HashMapContext[K], like
// RefCountedSetContext. The functions without `Context` in their name are a
// compile error in Zig unless `@sizeOf(Context) == 0`, and pass `undefined`
// as the context; in Go they use the zero value of `C`.
//
// # Go-memory rule, option A
//
// (docs/mapping.md, governing rule 3, as refined for this batch: ordinary Go
// memory, but upstream's bookkeeping and byte arithmetic are kept exactly.)
//
// Zig keeps one buffer: a `Header`, then `capacity` metadata bytes, then the
// keys, then the values; the map is a pointer to the metadata and finds the
// header just before it. In Go:
//   - the metadata is `metadata []hashMapMetadata`, the keys are
//     `keys []K` and the values `values []V` (in hashMapHeader, where Zig
//     has their offsets), each `capacity` long;
//   - `*K` and `*V` (in Entry, GetOrPutResult, getPtr, getKeyPtr and the
//     iterators) are Go pointers to elements of those slices;
//   - the header is reached through the pointer field `hdr`, which is not a
//     Zig field: it stands for "the Header just before the metadata". Copies
//     of a map therefore share one header (and so one `size`), as copies of
//     the Zig pointer do;
//   - a map with `metadata == nil` is Zig's `metadata == null`;
//   - `init` drops `buf: OffsetBuf`; `OffsetHashMap.map(base)` drops `base`.
//     `OffsetHashMap` is kept as its own type because the page stores it and
//     calls `map` for every operation;
//   - hashMapUnmanaged is a handle (two words pointing at shared state), so
//     every method has a value receiver, whether Zig says `Self`, `*Self` or
//     `*const Self`. This lets `m.Map().Count()` compile.
//
// Omitted under the Go-memory rule:
//   - `base_align`, `header_align`, `key_align`, `val_align` as
//     declarations (alignments of the buffer); the numbers they contribute
//     to the layout are in hashMapEntrySizes.
//
// # Byte arithmetic
//
// `layoutForCapacity` is kept and returns the same numbers as Zig, because
// Page.layout adds `total_size` into the page's byte size. It needs
// @sizeOf(K), @alignOf(K), @sizeOf(V), @alignOf(V) of the ZIG types, which
// Go cannot derive (the page's grapheme map stores an `Offset(u21).Slice`,
// 16 bytes, where Go stores a `[]uint32`). They are passed in as
// hashMapEntrySizes, which is the one declaration in this file that is not a
// Zig declaration. @sizeOf(Header) is 16 and @alignOf(Header) is 4 (four
// u32: `values`, `keys`, `capacity`, `size`); @sizeOf(Metadata) is 1.
//
// Widths: every byte size and byte offset is a `uint64`, whatever the
// platform word size, because the reference is the recorded 64-bit data and
// the target is 32-bit. Counts, indexes and lengths stay `uint` or their
// fixed Zig width.
// Here: hashMapEntrySizes and `Layout.total_size`, `keys_start` and
// `vals_start` are uint64; capacities and sizes are HashMapSize (u32), slot
// indexes are uint.
//
// # Pointer to index
//
// `removeByPtr` finds a key's slot by pointer subtraction. Go has none; the
// stub `keyIndex`, which is not a Zig declaration, stands for it.
//
// The file has no C ABI declarations.

package terminal

// HashMapContext is the `Context` parameter of the hash maps: a type with
//
//   - `fn hash(self, K) u64`
//   - `fn eql(self, K, K) bool`
//
// See the file header.
type HashMapContext[K any] interface {
	Hash(key K) uint64
	Eql(a K, b K) bool
}

// hashMapEntrySizes is NOT a Zig declaration. It carries what Zig reads from
// the types at compile time in layoutForCapacity: @sizeOf(K), @alignOf(K),
// @sizeOf(V), @alignOf(V), in bytes. Zig uses an alignment of 1 for a type
// of size 0 (`key_align`, `val_align`).
type hashMapEntrySizes struct {
	key_size  uint64
	key_align uint64
	val_size  uint64
	val_align uint64
}

// HashMapAutoOffsetHashMap is hash_map.AutoOffsetHashMap(K, V):
// OffsetHashMap(K, V, AutoContext(K)).
type HashMapAutoOffsetHashMap[K comparable, V any] = HashMapOffsetHashMap[K, V, hashMapAutoContext[K]]

// hashMapAutoHashMapUnmanaged is hash_map.AutoHashMapUnmanaged(K, V):
// HashMapUnmanaged(K, V, AutoContext(K)).
type hashMapAutoHashMapUnmanaged[K comparable, V any] = hashMapUnmanaged[K, V, hashMapAutoContext[K]]

// hashMapAutoContext is hash_map.AutoContext(K): the context whose `hash`
// is std.hash_map.getAutoHashFn(K, @This()) and whose `eql` is
// std.hash_map.getAutoEqlFn(K, @This()). It has no fields.
//
// For the key types used (integers, and the page's cell offset, a
// packed struct(u32)) getAutoHashFn is Wyhash with seed 0 over the key's
// bytes, and getAutoEqlFn is `==`.
type hashMapAutoContext[K comparable] struct{}

// Hash is AutoContext.hash.
//
// REQUIREMENT on the implementation: for every integer key type used (the
// page's u32 cell offset, and the u32, i32, u64 and usize keys of the
// tests) the result is Zig's std.hash.Wyhash.hash(0, bytes) over the key's
// bytes in little-endian order, at the key's ZIG width (usize is 8 bytes in
// the recorded data). The hash decides slot placement and so when a map
// reports that it is full.
//
// REQUIREMENT on the implementation: a Go `uint` or `int` key is hashed as
// 8 bytes (Zig usize/isize on the 64-bit reference) on every platform,
// including 32-bit targets where the Go type is 4 bytes wide.
func (self hashMapAutoContext[K]) Hash(key K) uint64 { panic(notImplemented) }

// Eql is AutoContext.eql.
func (self hashMapAutoContext[K]) Eql(a K, b K) bool { panic(notImplemented) }

// HashMapOffsetHashMap is the type returned by
// hash_map.OffsetHashMap(K, V, Context): a HashMap type that uses offsets
// rather than pointers, making it possible to efficiently move around the
// backing memory without invalidating the HashMap.
//
// Zig: one field, `metadata: Offset(Unmanaged.Metadata) = .{}`. In Go it
// holds what the pointer-based map holds; see the file header.
// `OffsetHashMap.Unmanaged` is hashMapUnmanaged[K, V, C].
type HashMapOffsetHashMap[K any, V any, C HashMapContext[K]] struct {
	// Zig: Offset(Unmanaged.Metadata), default .{}.
	metadata []hashMapMetadata

	// Not a Zig field; see the file header.
	hdr *hashMapHeader[K, V]
}

// HashMapOffsetHashMapLayout is OffsetHashMap.Layout: Unmanaged.Layout.
type HashMapOffsetHashMapLayout = hashMapLayout

// HashMapOffsetHashMapLayoutOf is OffsetHashMap.layout: returns the total
// size of the backing memory required for a HashMap with the given capacity.
// The base ptr must also be aligned to base_align.
//
// Zig: pub fn layout(cap: Unmanaged.Size) Layout, a function of the type.
// `sizes` stands for the type parameters K and V; see the file header.
// Named `...LayoutOf` because `...Layout` is the type.
func HashMapOffsetHashMapLayoutOf(cap HashMapSize, sizes hashMapEntrySizes) HashMapOffsetHashMapLayout {
	panic(notImplemented)
}

// HashMapOffsetHashMapInit is OffsetHashMap.init: initialize a new HashMap
// with the given capacity and backing memory. The backing memory must be
// aligned to base_align.
//
// Zig: pub fn init(buf: OffsetBuf, l: Layout) Self. The buffer argument is
// dropped. Go cannot infer the type arguments, so a caller writes all
// three:
// HashMapOffsetHashMapInit[uint32, uint32, hashMapAutoContext[uint32]](l).
func HashMapOffsetHashMapInit[K any, V any, C HashMapContext[K]](l HashMapOffsetHashMapLayout) HashMapOffsetHashMap[K, V, C] {
	panic(notImplemented)
}

// Map is OffsetHashMap.map: returns the pointer-based map from a base
// pointer.
//
// Zig: pub fn map(self: Self, base: anytype) Unmanaged.
func (self HashMapOffsetHashMap[K, V, C]) Map() hashMapUnmanaged[K, V, C] { panic(notImplemented) }

// hashMapUnmanaged is the type returned by
// hash_map.HashMapUnmanaged(K, V, Context).
//
// Fork of stdlib.HashMap as of Zig 0.12 modified to to use offsets
// for the key/values pointer. The metadata is still a pointer to limit
// the amount of arithmetic required to access it. See the file comment
// for full details.
type hashMapUnmanaged[K any, V any, C HashMapContext[K]] struct {
	// This is actually a midway pointer to the single buffer containing
	// a `Header` field, the `Metadata`s and `Entry`s.
	// At `-@sizeOf(Header)` is the Header field.
	// At `sizeOf(Metadata) * capacity + offset`, which is pointed to by
	// self.header().entries, is the array of entries.
	// This means that the hashmap only holds one live allocation, to
	// reduce memory fragmentation and struct size.
	//
	// Pointer to the metadata.
	//
	// Zig: ?[*]Metadata, default null. nil is null.
	metadata []hashMapMetadata

	// Not a Zig field: the Header that Zig finds just before the metadata.
	// See the file header.
	hdr *hashMapHeader[K, V]
}

// hashMapMinimalCapacity is HashMapUnmanaged.minimal_capacity.
//
// This is purely empirical and not a /very smart magic constant™/.
// Capacity of the first grow when bootstrapping the hashmap.
const hashMapMinimalCapacity = 8

// HashMapSize is HashMapUnmanaged.Size.
//
// This hashmap is specially designed for sizes that fit in a u32.
type HashMapSize = uint32

// HashMapHash is HashMapUnmanaged.Hash.
//
// u64 hashes guarantee us that the fingerprint bits will never be used
// to compute the index of a slot, maximizing the use of entropy.
type HashMapHash = uint64

// HashMapEntry is HashMapUnmanaged.Entry.
type HashMapEntry[K any, V any] struct {
	key_ptr   *K
	value_ptr *V
}

// HashMapKV is HashMapUnmanaged.KV.
type HashMapKV[K any, V any] struct {
	key   K
	value V
}

// hashMapHeader is HashMapUnmanaged.Header.
//
// Zig: @sizeOf 16, @alignOf 4.
type hashMapHeader[K any, V any] struct {
	// The keys/values offset are relative to the metadata
	//
	// Zig: Offset(V) and Offset(K). In Go the slices themselves, each
	// `capacity` long.
	values   []V
	keys     []K
	capacity HashMapSize
	size     HashMapSize
}

// hashMapMetadata is HashMapUnmanaged.Metadata.
//
// Metadata for a slot. It can be in three states: empty, used or
// tombstone. Tombstones indicate that an entry was previously used,
// they are a simple way to handle removal.
// To this state, we add 7 bits from the slot's key hash. These are
// used as a fast way to disambiguate between entries without
// having to use the equality function. If two fingerprints are
// different, we know that we don't have to compare the keys at all.
// The 7 bits are the highest ones from a 64 bit hash. This way, not
// only we use the `log2(capacity)` lowest bits from the hash to determine
// a slot index, but we use 7 more bits to quickly resolve collisions
// when multiple elements with different hashes end up wanting to be in the same slot.
// Not using the equality function means we don't have to read into
// the entries array, likely avoiding a cache miss and a potentially
// costly function call.
//
// Zig: packed struct (8 bits): fingerprint is bits 0..6, used is bit 7. The
// Go zero value is Zig's default `.{}`: a free slot.
type hashMapMetadata struct {
	// Zig: FingerPrint (u7), default free.
	fingerprint hashMapMetadataFingerPrint

	// Zig: u1, default 0.
	used uint8
}

// hashMapMetadataFingerPrint is HashMapUnmanaged.Metadata.FingerPrint.
//
// Zig: u7.
type hashMapMetadataFingerPrint = uint8

// hashMapMetadataFree and hashMapMetadataTombstone are Metadata.free and
// Metadata.tombstone.
const (
	hashMapMetadataFree      hashMapMetadataFingerPrint = 0
	hashMapMetadataTombstone hashMapMetadataFingerPrint = 1
)

// hashMapMetadataSlotFree and hashMapMetadataSlotTombstone are
// Metadata.slot_free and Metadata.slot_tombstone: the u8 bit patterns of
// Metadata{ .fingerprint = free } and Metadata{ .fingerprint = tombstone }.
const (
	hashMapMetadataSlotFree      uint8 = 0
	hashMapMetadataSlotTombstone uint8 = 1
)

// IsUsed is Metadata.isUsed.
func (self hashMapMetadata) IsUsed() bool { panic(notImplemented) }

// IsTombstone is Metadata.isTombstone.
func (self hashMapMetadata) IsTombstone() bool { panic(notImplemented) }

// IsFree is Metadata.isFree.
func (self hashMapMetadata) IsFree() bool { panic(notImplemented) }

// hashMapMetadataTakeFingerprint is Metadata.takeFingerprint: the 7 highest
// bits of the hash.
func hashMapMetadataTakeFingerprint(hash HashMapHash) hashMapMetadataFingerPrint {
	panic(notImplemented)
}

// Fill is Metadata.fill.
func (self *hashMapMetadata) Fill(fp hashMapMetadataFingerPrint) { panic(notImplemented) }

// Remove is Metadata.remove.
func (self *hashMapMetadata) Remove() { panic(notImplemented) }

// HashMapIterator is HashMapUnmanaged.Iterator.
type HashMapIterator[K any, V any, C HashMapContext[K]] struct {
	// Zig: *const Self.
	hm *hashMapUnmanaged[K, V, C]

	// Zig default: 0.
	index HashMapSize
}

// Next is HashMapUnmanaged.Iterator.next.
//
// Zig: pub fn next(it: *Iterator) ?Entry.
func (it *HashMapIterator[K, V, C]) Next() Optional[HashMapEntry[K, V]] { panic(notImplemented) }

// HashMapKeyIterator is HashMapUnmanaged.KeyIterator: FieldIterator(K).
type HashMapKeyIterator[K any] = hashMapFieldIterator[K]

// HashMapValueIterator is HashMapUnmanaged.ValueIterator: FieldIterator(V).
type HashMapValueIterator[V any] = hashMapFieldIterator[V]

// hashMapFieldIterator is the type returned by
// HashMapUnmanaged.FieldIterator(T).
type hashMapFieldIterator[T any] struct {
	len uint

	// Zig: [*]const Metadata. In Go the remaining metadata.
	metadata []hashMapMetadata

	// Zig: [*]T. In Go the remaining items.
	items []T
}

// Next is FieldIterator.next.
//
// Zig: pub fn next(self: *@This()) ?*T.
func (self *hashMapFieldIterator[T]) Next() *T { panic(notImplemented) }

// HashMapGetOrPutResult is HashMapUnmanaged.GetOrPutResult.
type HashMapGetOrPutResult[K any, V any] struct {
	key_ptr        *K
	value_ptr      *V
	found_existing bool
}

// hashMapUnmanagedInit is HashMapUnmanaged.init: initialize a hash map with
// a given capacity and a buffer. The buffer must fit within the size defined
// by `layoutForCapacity`.
//
// Zig: pub fn init(buf: OffsetBuf, layout: Layout) Self. The buffer argument
// is dropped: the map makes its own metadata, keys and values,
// `layout.capacity` long. Go cannot infer the type arguments, so a caller
// writes all three:
// hashMapUnmanagedInit[uint32, uint32, hashMapAutoContext[uint32]](layout).
//
// REQUIREMENT on the implementation: a map initialised with a capacity of
// zero has non-nil, empty metadata (`metadata != nil`, `len(metadata) ==
// 0`), as Zig's metadata pointer is non-null for any initialised map; nil
// metadata stands only for Zig's `metadata == null`.
func hashMapUnmanagedInit[K any, V any, C HashMapContext[K]](layout hashMapLayout) hashMapUnmanaged[K, V, C] {
	panic(notImplemented)
}

// EnsureTotalCapacity is HashMapUnmanaged.ensureTotalCapacity.
//
// The error is ErrOutOfMemory (dcs.go), here and in every function of this
// file that returns an error.
func (self hashMapUnmanaged[K, V, C]) EnsureTotalCapacity(new_size HashMapSize) error {
	panic(notImplemented)
}

// EnsureUnusedCapacity is HashMapUnmanaged.ensureUnusedCapacity.
func (self hashMapUnmanaged[K, V, C]) EnsureUnusedCapacity(additional_size HashMapSize) error {
	panic(notImplemented)
}

// ClearRetainingCapacity is HashMapUnmanaged.clearRetainingCapacity.
func (self hashMapUnmanaged[K, V, C]) ClearRetainingCapacity() { panic(notImplemented) }

// Count is HashMapUnmanaged.count.
func (self hashMapUnmanaged[K, V, C]) Count() HashMapSize { panic(notImplemented) }

// header is HashMapUnmanaged.header.
func (self hashMapUnmanaged[K, V, C]) header() *hashMapHeader[K, V] { panic(notImplemented) }

// keys is HashMapUnmanaged.keys.
//
// Zig: fn keys(self: *const Self) [*]K.
func (self hashMapUnmanaged[K, V, C]) keys() []K { panic(notImplemented) }

// values is HashMapUnmanaged.values.
//
// Zig: fn values(self: *const Self) [*]V.
func (self hashMapUnmanaged[K, V, C]) values() []V { panic(notImplemented) }

// Capacity is HashMapUnmanaged.capacity.
func (self hashMapUnmanaged[K, V, C]) Capacity() HashMapSize { panic(notImplemented) }

// Iterator is HashMapUnmanaged.iterator.
//
// Zig: self is `*const Self`. The Go receiver is a value, like every
// method of hashMapUnmanaged: the map type is a handle onto shared state
// (metadata slice and header pointer), so the iterator sees the same map.
func (self hashMapUnmanaged[K, V, C]) Iterator() HashMapIterator[K, V, C] { panic(notImplemented) }

// KeyIterator is HashMapUnmanaged.keyIterator.
func (self hashMapUnmanaged[K, V, C]) KeyIterator() HashMapKeyIterator[K] { panic(notImplemented) }

// ValueIterator is HashMapUnmanaged.valueIterator.
func (self hashMapUnmanaged[K, V, C]) ValueIterator() HashMapValueIterator[V] {
	panic(notImplemented)
}

// PutNoClobber is HashMapUnmanaged.putNoClobber: insert an entry in the map.
// Assumes it is not already present.
func (self hashMapUnmanaged[K, V, C]) PutNoClobber(key K, value V) error { panic(notImplemented) }

// PutNoClobberContext is HashMapUnmanaged.putNoClobberContext.
func (self hashMapUnmanaged[K, V, C]) PutNoClobberContext(key K, value V, ctx C) error {
	panic(notImplemented)
}

// PutAssumeCapacity is HashMapUnmanaged.putAssumeCapacity.
//
// Asserts there is enough capacity to store the new key-value pair.
// Clobbers any existing data. To detect if a put would clobber
// existing data, see `getOrPutAssumeCapacity`.
func (self hashMapUnmanaged[K, V, C]) PutAssumeCapacity(key K, value V) { panic(notImplemented) }

// PutAssumeCapacityContext is HashMapUnmanaged.putAssumeCapacityContext.
func (self hashMapUnmanaged[K, V, C]) PutAssumeCapacityContext(key K, value V, ctx C) {
	panic(notImplemented)
}

// PutAssumeCapacityNoClobber is HashMapUnmanaged.putAssumeCapacityNoClobber:
// insert an entry in the map. Assumes it is not already present,
// and that no allocation is needed.
func (self hashMapUnmanaged[K, V, C]) PutAssumeCapacityNoClobber(key K, value V) {
	panic(notImplemented)
}

// PutAssumeCapacityNoClobberContext is
// HashMapUnmanaged.putAssumeCapacityNoClobberContext.
func (self hashMapUnmanaged[K, V, C]) PutAssumeCapacityNoClobberContext(key K, value V, ctx C) {
	panic(notImplemented)
}

// FetchPut is HashMapUnmanaged.fetchPut: inserts a new `Entry` into the hash
// map, returning the previous one, if any.
//
// Zig: Allocator.Error!?KV.
func (self hashMapUnmanaged[K, V, C]) FetchPut(key K, value V) (Optional[HashMapKV[K, V]], error) {
	panic(notImplemented)
}

// FetchPutContext is HashMapUnmanaged.fetchPutContext.
func (self hashMapUnmanaged[K, V, C]) FetchPutContext(key K, value V, ctx C) (Optional[HashMapKV[K, V]], error) {
	panic(notImplemented)
}

// FetchPutAssumeCapacity is HashMapUnmanaged.fetchPutAssumeCapacity: inserts
// a new `Entry` into the hash map, returning the previous one, if any.
// If insertion happens, asserts there is enough capacity without allocating.
func (self hashMapUnmanaged[K, V, C]) FetchPutAssumeCapacity(key K, value V) Optional[HashMapKV[K, V]] {
	panic(notImplemented)
}

// FetchPutAssumeCapacityContext is
// HashMapUnmanaged.fetchPutAssumeCapacityContext.
func (self hashMapUnmanaged[K, V, C]) FetchPutAssumeCapacityContext(key K, value V, ctx C) Optional[HashMapKV[K, V]] {
	panic(notImplemented)
}

// FetchRemove is HashMapUnmanaged.fetchRemove: if there is an `Entry` with a
// matching key, it is deleted from the hash map, and then returned from this
// function.
func (self hashMapUnmanaged[K, V, C]) FetchRemove(key K) Optional[HashMapKV[K, V]] {
	panic(notImplemented)
}

// FetchRemoveContext is HashMapUnmanaged.fetchRemoveContext.
func (self hashMapUnmanaged[K, V, C]) FetchRemoveContext(key K, ctx C) Optional[HashMapKV[K, V]] {
	panic(notImplemented)
}

// FetchRemoveAdapted is HashMapUnmanaged.fetchRemoveAdapted.
//
// Zig: key: anytype, ctx: anytype. See the file header.
func (self hashMapUnmanaged[K, V, C]) FetchRemoveAdapted(key K, ctx C) Optional[HashMapKV[K, V]] {
	panic(notImplemented)
}

// getIndex is HashMapUnmanaged.getIndex.
//
// Find the index containing the data for the given key.
// Whether this function returns null is almost always
// branched on after this function returns, and this function
// returns null/not null from separate code paths.  We
// want the optimizer to remove that branch and instead directly
// fuse the basic blocks after the branch to the basic blocks
// from this function.  To encourage that, this function is
// marked as inline.
//
// Zig: inline fn getIndex(self: Self, key: anytype, ctx: anytype) ?usize.
func (self hashMapUnmanaged[K, V, C]) getIndex(key K, ctx C) Optional[uint] {
	panic(notImplemented)
}

// GetEntry is HashMapUnmanaged.getEntry.
func (self hashMapUnmanaged[K, V, C]) GetEntry(key K) Optional[HashMapEntry[K, V]] {
	panic(notImplemented)
}

// GetEntryContext is HashMapUnmanaged.getEntryContext.
func (self hashMapUnmanaged[K, V, C]) GetEntryContext(key K, ctx C) Optional[HashMapEntry[K, V]] {
	panic(notImplemented)
}

// GetEntryAdapted is HashMapUnmanaged.getEntryAdapted.
//
// Zig: key: anytype, ctx: anytype.
func (self hashMapUnmanaged[K, V, C]) GetEntryAdapted(key K, ctx C) Optional[HashMapEntry[K, V]] {
	panic(notImplemented)
}

// Put is HashMapUnmanaged.put: insert an entry if the associated key is not
// already present, otherwise update preexisting value.
func (self hashMapUnmanaged[K, V, C]) Put(key K, value V) error { panic(notImplemented) }

// PutContext is HashMapUnmanaged.putContext.
func (self hashMapUnmanaged[K, V, C]) PutContext(key K, value V, ctx C) error {
	panic(notImplemented)
}

// GetKeyPtr is HashMapUnmanaged.getKeyPtr: get an optional pointer to the
// actual key associated with adapted key, if present.
//
// Zig: ?*K.
func (self hashMapUnmanaged[K, V, C]) GetKeyPtr(key K) *K { panic(notImplemented) }

// GetKeyPtrContext is HashMapUnmanaged.getKeyPtrContext.
func (self hashMapUnmanaged[K, V, C]) GetKeyPtrContext(key K, ctx C) *K { panic(notImplemented) }

// GetKeyPtrAdapted is HashMapUnmanaged.getKeyPtrAdapted.
//
// Zig: key: anytype, ctx: anytype.
func (self hashMapUnmanaged[K, V, C]) GetKeyPtrAdapted(key K, ctx C) *K { panic(notImplemented) }

// GetKey is HashMapUnmanaged.getKey: get a copy of the actual key associated
// with adapted key, if present.
//
// Zig: ?K.
func (self hashMapUnmanaged[K, V, C]) GetKey(key K) Optional[K] { panic(notImplemented) }

// GetKeyContext is HashMapUnmanaged.getKeyContext.
func (self hashMapUnmanaged[K, V, C]) GetKeyContext(key K, ctx C) Optional[K] {
	panic(notImplemented)
}

// GetKeyAdapted is HashMapUnmanaged.getKeyAdapted.
//
// Zig: key: anytype, ctx: anytype.
func (self hashMapUnmanaged[K, V, C]) GetKeyAdapted(key K, ctx C) Optional[K] {
	panic(notImplemented)
}

// GetPtr is HashMapUnmanaged.getPtr: get an optional pointer to the value
// associated with key, if present.
//
// Zig: ?*V.
func (self hashMapUnmanaged[K, V, C]) GetPtr(key K) *V { panic(notImplemented) }

// GetPtrContext is HashMapUnmanaged.getPtrContext.
func (self hashMapUnmanaged[K, V, C]) GetPtrContext(key K, ctx C) *V { panic(notImplemented) }

// GetPtrAdapted is HashMapUnmanaged.getPtrAdapted.
//
// Zig: key: anytype, ctx: anytype.
func (self hashMapUnmanaged[K, V, C]) GetPtrAdapted(key K, ctx C) *V { panic(notImplemented) }

// Get is HashMapUnmanaged.get: get a copy of the value associated with key,
// if present.
//
// Zig: ?V.
func (self hashMapUnmanaged[K, V, C]) Get(key K) Optional[V] { panic(notImplemented) }

// GetContext is HashMapUnmanaged.getContext.
func (self hashMapUnmanaged[K, V, C]) GetContext(key K, ctx C) Optional[V] {
	panic(notImplemented)
}

// GetAdapted is HashMapUnmanaged.getAdapted.
//
// Zig: key: anytype, ctx: anytype.
func (self hashMapUnmanaged[K, V, C]) GetAdapted(key K, ctx C) Optional[V] {
	panic(notImplemented)
}

// GetOrPut is HashMapUnmanaged.getOrPut.
func (self hashMapUnmanaged[K, V, C]) GetOrPut(key K) (HashMapGetOrPutResult[K, V], error) {
	panic(notImplemented)
}

// GetOrPutContext is HashMapUnmanaged.getOrPutContext.
func (self hashMapUnmanaged[K, V, C]) GetOrPutContext(key K, ctx C) (HashMapGetOrPutResult[K, V], error) {
	panic(notImplemented)
}

// GetOrPutAdapted is HashMapUnmanaged.getOrPutAdapted.
//
// Zig: key: anytype, key_ctx: anytype.
func (self hashMapUnmanaged[K, V, C]) GetOrPutAdapted(key K, key_ctx C) (HashMapGetOrPutResult[K, V], error) {
	panic(notImplemented)
}

// GetOrPutContextAdapted is HashMapUnmanaged.getOrPutContextAdapted.
//
// Zig: key: anytype, key_ctx: anytype.
func (self hashMapUnmanaged[K, V, C]) GetOrPutContextAdapted(key K, key_ctx C) (HashMapGetOrPutResult[K, V], error) {
	panic(notImplemented)
}

// GetOrPutAssumeCapacity is HashMapUnmanaged.getOrPutAssumeCapacity.
func (self hashMapUnmanaged[K, V, C]) GetOrPutAssumeCapacity(key K) HashMapGetOrPutResult[K, V] {
	panic(notImplemented)
}

// GetOrPutAssumeCapacityContext is
// HashMapUnmanaged.getOrPutAssumeCapacityContext.
func (self hashMapUnmanaged[K, V, C]) GetOrPutAssumeCapacityContext(key K, ctx C) HashMapGetOrPutResult[K, V] {
	panic(notImplemented)
}

// GetOrPutAssumeCapacityAdapted is
// HashMapUnmanaged.getOrPutAssumeCapacityAdapted.
//
// Zig: key: anytype, ctx: anytype.
func (self hashMapUnmanaged[K, V, C]) GetOrPutAssumeCapacityAdapted(key K, ctx C) HashMapGetOrPutResult[K, V] {
	panic(notImplemented)
}

// GetOrPutValue is HashMapUnmanaged.getOrPutValue.
func (self hashMapUnmanaged[K, V, C]) GetOrPutValue(key K, value V) (HashMapEntry[K, V], error) {
	panic(notImplemented)
}

// GetOrPutValueContext is HashMapUnmanaged.getOrPutValueContext.
func (self hashMapUnmanaged[K, V, C]) GetOrPutValueContext(key K, value V, ctx C) (HashMapEntry[K, V], error) {
	panic(notImplemented)
}

// Contains is HashMapUnmanaged.contains: return true if there is a value
// associated with key in the map.
func (self hashMapUnmanaged[K, V, C]) Contains(key K) bool { panic(notImplemented) }

// ContainsContext is HashMapUnmanaged.containsContext.
func (self hashMapUnmanaged[K, V, C]) ContainsContext(key K, ctx C) bool { panic(notImplemented) }

// ContainsAdapted is HashMapUnmanaged.containsAdapted.
//
// Zig: key: anytype, ctx: anytype.
func (self hashMapUnmanaged[K, V, C]) ContainsAdapted(key K, ctx C) bool { panic(notImplemented) }

// removeByIndex is HashMapUnmanaged.removeByIndex.
func (self hashMapUnmanaged[K, V, C]) removeByIndex(idx uint) { panic(notImplemented) }

// Remove is HashMapUnmanaged.remove.
//
// If there is an `Entry` with a matching key, it is deleted from
// the hash map, and this function returns true.  Otherwise this
// function returns false.
func (self hashMapUnmanaged[K, V, C]) Remove(key K) bool { panic(notImplemented) }

// RemoveContext is HashMapUnmanaged.removeContext.
func (self hashMapUnmanaged[K, V, C]) RemoveContext(key K, ctx C) bool { panic(notImplemented) }

// RemoveAdapted is HashMapUnmanaged.removeAdapted.
//
// Zig: key: anytype, ctx: anytype.
func (self hashMapUnmanaged[K, V, C]) RemoveAdapted(key K, ctx C) bool { panic(notImplemented) }

// RemoveByPtr is HashMapUnmanaged.removeByPtr.
//
// Delete the entry with key pointed to by key_ptr from the hash map.
// key_ptr is assumed to be a valid pointer to a key that is present
// in the hash map.
//
// Zig finds the index by pointer subtraction; see keyIndex.
func (self hashMapUnmanaged[K, V, C]) RemoveByPtr(key_ptr *K) { panic(notImplemented) }

// keyIndex is NOT a Zig declaration. It stands for the pointer subtraction
// in removeByPtr,
// `(@intFromPtr(key_ptr) - @intFromPtr(self.keys())) / @sizeOf(K)`: the
// index in `keys` of the element key_ptr points at.
//
// The implementation may use package unsafe for this and for nothing else.
//
// REQUIREMENT on the implementation: keyIndex must panic if key_ptr does
// not point at an element of the map's key slice (where Zig's result would
// be meaningless); tests check this.
func (self hashMapUnmanaged[K, V, C]) keyIndex(key_ptr *K) uint { panic(notImplemented) }

// initMetadatas is HashMapUnmanaged.initMetadatas.
func (self hashMapUnmanaged[K, V, C]) initMetadatas() { panic(notImplemented) }

// growIfNeeded is HashMapUnmanaged.growIfNeeded.
func (self hashMapUnmanaged[K, V, C]) growIfNeeded(new_count HashMapSize) error {
	panic(notImplemented)
}

// hashMapLayout is HashMapUnmanaged.Layout: the memory layout for the
// underlying buffer for a given capacity. Every field is a number, as in
// Zig.
type hashMapLayout struct {
	// The total size of the buffer required. The buffer is expected
	// to be aligned to `base_align`.
	total_size uint64

	// The offset to the start of the keys data.
	keys_start uint64

	// The offset to the start of the values data.
	vals_start uint64

	// The capacity that was used to calculate this layout.
	capacity HashMapSize
}

// hashMapLayoutForCapacity is HashMapUnmanaged.layoutForCapacity.
//
// Returns the memory layout for the buffer for a given capacity.
// The actual size may be able to fit more than the given capacity
// because capacity is rounded up to the next power of two. This is
// a design requirement for this hash map implementation.
//
// Zig: pub fn layoutForCapacity(new_capacity: Size) Layout, a function of
// the (private) type. `sizes` stands for the type parameters K and V; see
// the file header. Zig asserts that new_capacity is 0 or a power of two.
func hashMapLayoutForCapacity(new_capacity HashMapSize, sizes hashMapEntrySizes) hashMapLayout {
	panic(notImplemented)
}
