// Ported from src/terminal/hash_map.zig (Ghostty v1.3.1).
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
// # Implementation notes
//
//   - Zig's run-time safety traps are kept as assertions: `self.metadata.?`
//     on a null map (in header and initMetadatas), `capacity() - 1` on a
//     zero capacity (putAssumeCapacityNoClobberContext, getIndex,
//     getOrPutAssumeCapacityAdapted), `capacity() - size` in growIfNeeded,
//     `count() + additional_size` in ensureUnusedCapacity, and the u32
//     `size += 1` / `size -= 1`.
//   - `x.* = undefined` is the Go zero value.
//   - `self.metadata.? + idx` (a many-pointer, unchecked) is the index
//     `idx` into the metadata slice; the loops index it directly.
//   - The `...Context`-less functions are a compile error in Zig unless
//     @sizeOf(Context) == 0 and pass `undefined`; Go passes the zero value
//     of C.
//   - Metadata's `@bitCast` to u8 is the helper bitCast (not a Zig
//     declaration): fingerprint in bits 0..6, used in bit 7.
//   - AutoContext.hash: see Hash; it switches on the key's type once per
//     call, the only way Go has to pick the key's Zig width inside a
//     method of an empty generic struct.
//   - keyIndex uses package unsafe, as the wireframe allows, and nothing
//     else in this file does.
//
// The file has no C ABI declarations.

package terminal

import (
	"encoding/binary"
	"math"
	"unsafe"
)

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
func (self hashMapAutoContext[K]) Hash(key K) uint64 {
	// getAutoHashFn: std.meta.hasUniqueRepresentation(K) holds for every
	// integer key, so the hash is Wyhash.hash(0, std.mem.asBytes(&key)):
	// the key's bytes, little-endian, at the Zig width. Go cannot
	// specialise a method body per type argument; the type switch is the
	// one run-time step that stands for the comptime choice of width.
	var buf [8]uint8
	var n int
	switch k := any(key).(type) {
	case uint8:
		buf[0], n = k, 1
	case int8:
		buf[0], n = uint8(k), 1
	case uint16:
		binary.LittleEndian.PutUint16(buf[:], k)
		n = 2
	case int16:
		binary.LittleEndian.PutUint16(buf[:], uint16(k))
		n = 2
	case uint32:
		binary.LittleEndian.PutUint32(buf[:], k)
		n = 4
	case int32:
		binary.LittleEndian.PutUint32(buf[:], uint32(k))
		n = 4
	case uint64:
		binary.LittleEndian.PutUint64(buf[:], k)
		n = 8
	case int64:
		binary.LittleEndian.PutUint64(buf[:], uint64(k))
		n = 8
	case uint:
		// usize: 8 bytes, the reference width, on every platform.
		binary.LittleEndian.PutUint64(buf[:], uint64(k))
		n = 8
	case int:
		// isize: 8 bytes (sign-extended), the reference width, on every
		// platform.
		binary.LittleEndian.PutUint64(buf[:], uint64(int64(k)))
		n = 8
	default:
		panic("not ported: hash_map.AutoContext.hash of this key type")
	}
	return stdHashWyhashHash(0, buf[:n])
}

// Eql is AutoContext.eql.
func (self hashMapAutoContext[K]) Eql(a K, b K) bool {
	// getAutoEqlFn: std.meta.eql(a, b), `==` for the key types used.
	return a == b
}

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
	return hashMapLayoutForCapacity(cap, sizes)
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
	// Zig: assert(base_align.check(@intFromPtr(buf.start()))). No buffer
	// in Go.

	m := hashMapUnmanagedInit[K, V, C](l)
	// Zig: .metadata = getOffset(Unmanaged.Metadata, buf, @ptrCast(m.metadata.?))
	return HashMapOffsetHashMap[K, V, C]{metadata: m.metadata, hdr: m.hdr}
}

// Map is OffsetHashMap.map: returns the pointer-based map from a base
// pointer.
//
// Zig: pub fn map(self: Self, base: anytype) Unmanaged.
func (self HashMapOffsetHashMap[K, V, C]) Map() hashMapUnmanaged[K, V, C] {
	// Zig: .{ .metadata = self.metadata.ptr(base) }
	return hashMapUnmanaged[K, V, C]{metadata: self.metadata, hdr: self.hdr}
}

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
func (self hashMapMetadata) IsUsed() bool {
	return self.used == 1
}

// IsTombstone is Metadata.isTombstone.
func (self hashMapMetadata) IsTombstone() bool {
	return self.bitCast() == hashMapMetadataSlotTombstone
}

// IsFree is Metadata.isFree.
func (self hashMapMetadata) IsFree() bool {
	return self.bitCast() == hashMapMetadataSlotFree
}

// hashMapMetadataTakeFingerprint is Metadata.takeFingerprint: the 7 highest
// bits of the hash.
func hashMapMetadataTakeFingerprint(hash HashMapHash) hashMapMetadataFingerPrint {
	const hash_bits = 64 // @typeInfo(Hash).int.bits
	const fp_bits = 7    // @typeInfo(FingerPrint).int.bits
	// @truncate to a u7: the shift leaves 7 bits.
	return hashMapMetadataFingerPrint(hash >> (hash_bits - fp_bits))
}

// Fill is Metadata.fill.
func (self *hashMapMetadata) Fill(fp hashMapMetadataFingerPrint) {
	self.used = 1
	self.fingerprint = fp
}

// Remove is Metadata.remove.
func (self *hashMapMetadata) Remove() {
	self.used = 0
	self.fingerprint = hashMapMetadataTombstone
}

// bitCast is NOT a Zig declaration: `@as(u8, @bitCast(self))` of the packed
// struct, fingerprint in bits 0..6 and used in bit 7.
func (self hashMapMetadata) bitCast() uint8 {
	return self.fingerprint&0x7f | (self.used&1)<<7
}

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
func (it *HashMapIterator[K, V, C]) Next() Optional[HashMapEntry[K, V]] {
	assert(it.index <= it.hm.Capacity())
	if it.hm.header().size == 0 {
		return None[HashMapEntry[K, V]]()
	}

	cap := it.hm.Capacity()
	// const end = it.hm.metadata.? + cap;
	// var metadata = it.hm.metadata.? + it.index;
	// Go: the metadata from it.index up to end.
	metadata := it.hm.metadata[it.index:cap]

	for ; len(metadata) != 0; metadata, it.index = metadata[1:], it.index+1 {
		if metadata[0].IsUsed() {
			key := &it.hm.keys()[it.index]
			value := &it.hm.values()[it.index]
			it.index += 1
			return Some(HashMapEntry[K, V]{key_ptr: key, value_ptr: value})
		}
	}

	return None[HashMapEntry[K, V]]()
}

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
func (self *hashMapFieldIterator[T]) Next() *T {
	for self.len > 0 {
		self.len -= 1
		used := self.metadata[0].IsUsed()
		item := &self.items[0]
		self.metadata = self.metadata[1:]
		self.items = self.items[1:]
		if used {
			return item
		}
	}
	return nil
}

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
	// Zig: assert(base_align.check(@intFromPtr(buf.start()))). No buffer
	// in Go.

	// Get all our main pointers
	//
	// Zig: the metadata starts @sizeOf(Header) into buf. Go: its own
	// slice, `layout.capacity` long (non-nil also for capacity 0), and the
	// Header the map points at.
	metadata_ptr := make([]hashMapMetadata, layout.capacity)

	// Build our map
	m := hashMapUnmanaged[K, V, C]{metadata: metadata_ptr, hdr: &hashMapHeader[K, V]{}}
	hdr := m.header()
	hdr.capacity = layout.capacity
	hdr.size = 0
	// if (@sizeOf([*]K) != 0) hdr.keys = metadata_buf.member(K, layout.keys_start);
	hdr.keys = make([]K, layout.capacity)
	// if (@sizeOf([*]V) != 0) hdr.values = metadata_buf.member(V, layout.vals_start);
	hdr.values = make([]V, layout.capacity)
	m.initMetadatas()

	return m
}

// EnsureTotalCapacity is HashMapUnmanaged.ensureTotalCapacity.
//
// The error is ErrOutOfMemory (dcs.go), here and in every function of this
// file that returns an error.
func (self hashMapUnmanaged[K, V, C]) EnsureTotalCapacity(new_size HashMapSize) error {
	if new_size > self.header().size {
		if err := self.growIfNeeded(new_size - self.header().size); err != nil {
			return err
		}
	}
	return nil
}

// EnsureUnusedCapacity is HashMapUnmanaged.ensureUnusedCapacity.
func (self hashMapUnmanaged[K, V, C]) EnsureUnusedCapacity(additional_size HashMapSize) error {
	count := self.Count()
	assert(additional_size <= math.MaxUint32-count) // Zig: `+` traps on overflow
	return self.EnsureTotalCapacity(count + additional_size)
}

// ClearRetainingCapacity is HashMapUnmanaged.clearRetainingCapacity.
func (self hashMapUnmanaged[K, V, C]) ClearRetainingCapacity() {
	if self.metadata != nil {
		self.initMetadatas()
		self.header().size = 0
	}
}

// Count is HashMapUnmanaged.count.
func (self hashMapUnmanaged[K, V, C]) Count() HashMapSize {
	return self.header().size
}

// header is HashMapUnmanaged.header.
func (self hashMapUnmanaged[K, V, C]) header() *hashMapHeader[K, V] {
	// Zig: self.metadata.? (null is a safety-checked panic), then the Header
	// just before it. Go: the header the map points at.
	assert(self.metadata != nil)
	return self.hdr
}

// keys is HashMapUnmanaged.keys.
//
// Zig: fn keys(self: *const Self) [*]K.
func (self hashMapUnmanaged[K, V, C]) keys() []K {
	return self.header().keys
}

// values is HashMapUnmanaged.values.
//
// Zig: fn values(self: *const Self) [*]V.
func (self hashMapUnmanaged[K, V, C]) values() []V {
	return self.header().values
}

// Capacity is HashMapUnmanaged.capacity.
func (self hashMapUnmanaged[K, V, C]) Capacity() HashMapSize {
	if self.metadata == nil {
		return 0
	}

	return self.header().capacity
}

// Iterator is HashMapUnmanaged.iterator.
//
// Zig: self is `*const Self`. The Go receiver is a value, like every
// method of hashMapUnmanaged: the map type is a handle onto shared state
// (metadata slice and header pointer), so the iterator sees the same map.
func (self hashMapUnmanaged[K, V, C]) Iterator() HashMapIterator[K, V, C] {
	return HashMapIterator[K, V, C]{hm: &self}
}

// KeyIterator is HashMapUnmanaged.keyIterator.
func (self hashMapUnmanaged[K, V, C]) KeyIterator() HashMapKeyIterator[K] {
	if metadata := self.metadata; metadata != nil {
		return HashMapKeyIterator[K]{
			len:      uint(self.Capacity()),
			metadata: metadata,
			items:    self.keys(),
		}
	} else {
		// Zig: metadata and items undefined.
		return HashMapKeyIterator[K]{
			len: 0,
		}
	}
}

// ValueIterator is HashMapUnmanaged.valueIterator.
func (self hashMapUnmanaged[K, V, C]) ValueIterator() HashMapValueIterator[V] {
	if metadata := self.metadata; metadata != nil {
		return HashMapValueIterator[V]{
			len:      uint(self.Capacity()),
			metadata: metadata,
			items:    self.values(),
		}
	} else {
		// Zig: metadata and items undefined.
		return HashMapValueIterator[V]{
			len: 0,
		}
	}
}

// PutNoClobber is HashMapUnmanaged.putNoClobber: insert an entry in the map.
// Assumes it is not already present.
func (self hashMapUnmanaged[K, V, C]) PutNoClobber(key K, value V) error {
	// Zig: a compile error unless @sizeOf(Context) == 0; the context is
	// `undefined`, in Go the zero value of C.
	var ctx C
	return self.PutNoClobberContext(key, value, ctx)
}

// PutNoClobberContext is HashMapUnmanaged.putNoClobberContext.
func (self hashMapUnmanaged[K, V, C]) PutNoClobberContext(key K, value V, ctx C) error {
	assert(!self.ContainsContext(key, ctx))
	if err := self.growIfNeeded(1); err != nil {
		return err
	}

	self.PutAssumeCapacityNoClobberContext(key, value, ctx)
	return nil
}

// PutAssumeCapacity is HashMapUnmanaged.putAssumeCapacity.
//
// Asserts there is enough capacity to store the new key-value pair.
// Clobbers any existing data. To detect if a put would clobber
// existing data, see `getOrPutAssumeCapacity`.
func (self hashMapUnmanaged[K, V, C]) PutAssumeCapacity(key K, value V) {
	// Zig: a compile error unless @sizeOf(Context) == 0; the context is
	// `undefined`, in Go the zero value of C.
	var ctx C
	self.PutAssumeCapacityContext(key, value, ctx)
}

// PutAssumeCapacityContext is HashMapUnmanaged.putAssumeCapacityContext.
func (self hashMapUnmanaged[K, V, C]) PutAssumeCapacityContext(key K, value V, ctx C) {
	gop := self.GetOrPutAssumeCapacityContext(key, ctx)
	*gop.value_ptr = value
}

// PutAssumeCapacityNoClobber is HashMapUnmanaged.putAssumeCapacityNoClobber:
// insert an entry in the map. Assumes it is not already present,
// and that no allocation is needed.
func (self hashMapUnmanaged[K, V, C]) PutAssumeCapacityNoClobber(key K, value V) {
	// Zig: a compile error unless @sizeOf(Context) == 0; the context is
	// `undefined`, in Go the zero value of C.
	var ctx C
	self.PutAssumeCapacityNoClobberContext(key, value, ctx)
}

// PutAssumeCapacityNoClobberContext is
// HashMapUnmanaged.putAssumeCapacityNoClobberContext.
func (self hashMapUnmanaged[K, V, C]) PutAssumeCapacityNoClobberContext(key K, value V, ctx C) {
	assert(!self.ContainsContext(key, ctx))

	hash := ctx.Hash(key)
	capacity := self.Capacity()
	assert(capacity >= 1) // Zig: `capacity() - 1` traps on overflow
	mask := capacity - 1
	idx := uint(hash & uint64(mask)) // @truncate to usize

	// var metadata = self.metadata.? + idx;
	for self.metadata[idx].IsUsed() {
		idx = (idx + 1) & uint(mask)
	}

	fingerprint := hashMapMetadataTakeFingerprint(hash)
	self.metadata[idx].Fill(fingerprint)
	self.keys()[idx] = key
	self.values()[idx] = value
	hdr := self.header()
	assert(hdr.size < math.MaxUint32) // Zig: `+=` traps on overflow
	hdr.size += 1
}

// FetchPut is HashMapUnmanaged.fetchPut: inserts a new `Entry` into the hash
// map, returning the previous one, if any.
//
// Zig: Allocator.Error!?KV.
func (self hashMapUnmanaged[K, V, C]) FetchPut(key K, value V) (Optional[HashMapKV[K, V]], error) {
	// Zig: a compile error unless @sizeOf(Context) == 0; the context is
	// `undefined`, in Go the zero value of C.
	var ctx C
	return self.FetchPutContext(key, value, ctx)
}

// FetchPutContext is HashMapUnmanaged.fetchPutContext.
func (self hashMapUnmanaged[K, V, C]) FetchPutContext(key K, value V, ctx C) (Optional[HashMapKV[K, V]], error) {
	gop, err := self.GetOrPutContext(key, ctx)
	if err != nil {
		return None[HashMapKV[K, V]](), err
	}
	result := None[HashMapKV[K, V]]()
	if gop.found_existing {
		result = Some(HashMapKV[K, V]{
			key:   *gop.key_ptr,
			value: *gop.value_ptr,
		})
	}
	*gop.value_ptr = value
	return result, nil
}

// FetchPutAssumeCapacity is HashMapUnmanaged.fetchPutAssumeCapacity: inserts
// a new `Entry` into the hash map, returning the previous one, if any.
// If insertion happens, asserts there is enough capacity without allocating.
func (self hashMapUnmanaged[K, V, C]) FetchPutAssumeCapacity(key K, value V) Optional[HashMapKV[K, V]] {
	// Zig: a compile error unless @sizeOf(Context) == 0; the context is
	// `undefined`, in Go the zero value of C.
	var ctx C
	return self.FetchPutAssumeCapacityContext(key, value, ctx)
}

// FetchPutAssumeCapacityContext is
// HashMapUnmanaged.fetchPutAssumeCapacityContext.
func (self hashMapUnmanaged[K, V, C]) FetchPutAssumeCapacityContext(key K, value V, ctx C) Optional[HashMapKV[K, V]] {
	gop := self.GetOrPutAssumeCapacityContext(key, ctx)
	result := None[HashMapKV[K, V]]()
	if gop.found_existing {
		result = Some(HashMapKV[K, V]{
			key:   *gop.key_ptr,
			value: *gop.value_ptr,
		})
	}
	*gop.value_ptr = value
	return result
}

// FetchRemove is HashMapUnmanaged.fetchRemove: if there is an `Entry` with a
// matching key, it is deleted from the hash map, and then returned from this
// function.
func (self hashMapUnmanaged[K, V, C]) FetchRemove(key K) Optional[HashMapKV[K, V]] {
	// Zig: a compile error unless @sizeOf(Context) == 0; the context is
	// `undefined`, in Go the zero value of C.
	var ctx C
	return self.FetchRemoveContext(key, ctx)
}

// FetchRemoveContext is HashMapUnmanaged.fetchRemoveContext.
func (self hashMapUnmanaged[K, V, C]) FetchRemoveContext(key K, ctx C) Optional[HashMapKV[K, V]] {
	return self.FetchRemoveAdapted(key, ctx)
}

// FetchRemoveAdapted is HashMapUnmanaged.fetchRemoveAdapted.
//
// Zig: key: anytype, ctx: anytype. See the file header.
func (self hashMapUnmanaged[K, V, C]) FetchRemoveAdapted(key K, ctx C) Optional[HashMapKV[K, V]] {
	if idx := self.getIndex(key, ctx); idx.Valid {
		old_key := &self.keys()[idx.Value]
		old_val := &self.values()[idx.Value]
		result := HashMapKV[K, V]{
			key:   *old_key,
			value: *old_val,
		}
		self.metadata[idx.Value].Remove()
		// old_key.* = undefined; old_val.* = undefined;
		var zero_key K
		var zero_val V
		*old_key = zero_key
		*old_val = zero_val
		hdr := self.header()
		assert(hdr.size > 0) // Zig: `-=` traps on overflow
		hdr.size -= 1
		return Some(result)
	}

	return None[HashMapKV[K, V]]()
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
	if self.header().size == 0 {
		return None[uint]()
	}

	// If you get a compile error on this line, it means that your generic hash
	// function is invalid for these parameters.
	hash := ctx.Hash(key)
	capacity := self.Capacity()
	assert(capacity >= 1) // Zig: `capacity() - 1` traps on overflow
	mask := capacity - 1
	fingerprint := hashMapMetadataTakeFingerprint(hash)
	// Don't loop indefinitely when there are no empty slots.
	limit := self.Capacity()
	idx := uint(hash & uint64(mask)) // @truncate to usize

	// var metadata = self.metadata.? + idx;
	for !self.metadata[idx].IsFree() && limit != 0 {
		if self.metadata[idx].IsUsed() && self.metadata[idx].fingerprint == fingerprint {
			test_key := &self.keys()[idx]
			// If you get a compile error on this line, it means that your generic eql
			// function is invalid for these parameters.
			eql := ctx.Eql(key, *test_key)
			if eql {
				return Some(idx)
			}
		}

		limit -= 1
		idx = (idx + 1) & uint(mask)
	}

	return None[uint]()
}

// GetEntry is HashMapUnmanaged.getEntry.
func (self hashMapUnmanaged[K, V, C]) GetEntry(key K) Optional[HashMapEntry[K, V]] {
	// Zig: a compile error unless @sizeOf(Context) == 0; the context is
	// `undefined`, in Go the zero value of C.
	var ctx C
	return self.GetEntryContext(key, ctx)
}

// GetEntryContext is HashMapUnmanaged.getEntryContext.
func (self hashMapUnmanaged[K, V, C]) GetEntryContext(key K, ctx C) Optional[HashMapEntry[K, V]] {
	return self.GetEntryAdapted(key, ctx)
}

// GetEntryAdapted is HashMapUnmanaged.getEntryAdapted.
//
// Zig: key: anytype, ctx: anytype.
func (self hashMapUnmanaged[K, V, C]) GetEntryAdapted(key K, ctx C) Optional[HashMapEntry[K, V]] {
	if idx := self.getIndex(key, ctx); idx.Valid {
		return Some(HashMapEntry[K, V]{
			key_ptr:   &self.keys()[idx.Value],
			value_ptr: &self.values()[idx.Value],
		})
	}
	return None[HashMapEntry[K, V]]()
}

// Put is HashMapUnmanaged.put: insert an entry if the associated key is not
// already present, otherwise update preexisting value.
func (self hashMapUnmanaged[K, V, C]) Put(key K, value V) error {
	// Zig: a compile error unless @sizeOf(Context) == 0; the context is
	// `undefined`, in Go the zero value of C.
	var ctx C
	return self.PutContext(key, value, ctx)
}

// PutContext is HashMapUnmanaged.putContext.
func (self hashMapUnmanaged[K, V, C]) PutContext(key K, value V, ctx C) error {
	result, err := self.GetOrPutContext(key, ctx)
	if err != nil {
		return err
	}
	*result.value_ptr = value
	return nil
}

// GetKeyPtr is HashMapUnmanaged.getKeyPtr: get an optional pointer to the
// actual key associated with adapted key, if present.
//
// Zig: ?*K.
func (self hashMapUnmanaged[K, V, C]) GetKeyPtr(key K) *K {
	// Zig: a compile error unless @sizeOf(Context) == 0; the context is
	// `undefined`, in Go the zero value of C.
	var ctx C
	return self.GetKeyPtrContext(key, ctx)
}

// GetKeyPtrContext is HashMapUnmanaged.getKeyPtrContext.
func (self hashMapUnmanaged[K, V, C]) GetKeyPtrContext(key K, ctx C) *K {
	return self.GetKeyPtrAdapted(key, ctx)
}

// GetKeyPtrAdapted is HashMapUnmanaged.getKeyPtrAdapted.
//
// Zig: key: anytype, ctx: anytype.
func (self hashMapUnmanaged[K, V, C]) GetKeyPtrAdapted(key K, ctx C) *K {
	if idx := self.getIndex(key, ctx); idx.Valid {
		return &self.keys()[idx.Value]
	}
	return nil
}

// GetKey is HashMapUnmanaged.getKey: get a copy of the actual key associated
// with adapted key, if present.
//
// Zig: ?K.
func (self hashMapUnmanaged[K, V, C]) GetKey(key K) Optional[K] {
	// Zig: a compile error unless @sizeOf(Context) == 0; the context is
	// `undefined`, in Go the zero value of C.
	var ctx C
	return self.GetKeyContext(key, ctx)
}

// GetKeyContext is HashMapUnmanaged.getKeyContext.
func (self hashMapUnmanaged[K, V, C]) GetKeyContext(key K, ctx C) Optional[K] {
	return self.GetKeyAdapted(key, ctx)
}

// GetKeyAdapted is HashMapUnmanaged.getKeyAdapted.
//
// Zig: key: anytype, ctx: anytype.
func (self hashMapUnmanaged[K, V, C]) GetKeyAdapted(key K, ctx C) Optional[K] {
	if idx := self.getIndex(key, ctx); idx.Valid {
		return Some(self.keys()[idx.Value])
	}
	return None[K]()
}

// GetPtr is HashMapUnmanaged.getPtr: get an optional pointer to the value
// associated with key, if present.
//
// Zig: ?*V.
func (self hashMapUnmanaged[K, V, C]) GetPtr(key K) *V {
	// Zig: a compile error unless @sizeOf(Context) == 0; the context is
	// `undefined`, in Go the zero value of C.
	var ctx C
	return self.GetPtrContext(key, ctx)
}

// GetPtrContext is HashMapUnmanaged.getPtrContext.
func (self hashMapUnmanaged[K, V, C]) GetPtrContext(key K, ctx C) *V {
	return self.GetPtrAdapted(key, ctx)
}

// GetPtrAdapted is HashMapUnmanaged.getPtrAdapted.
//
// Zig: key: anytype, ctx: anytype.
func (self hashMapUnmanaged[K, V, C]) GetPtrAdapted(key K, ctx C) *V {
	if idx := self.getIndex(key, ctx); idx.Valid {
		return &self.values()[idx.Value]
	}
	return nil
}

// Get is HashMapUnmanaged.get: get a copy of the value associated with key,
// if present.
//
// Zig: ?V.
func (self hashMapUnmanaged[K, V, C]) Get(key K) Optional[V] {
	// Zig: a compile error unless @sizeOf(Context) == 0; the context is
	// `undefined`, in Go the zero value of C.
	var ctx C
	return self.GetContext(key, ctx)
}

// GetContext is HashMapUnmanaged.getContext.
func (self hashMapUnmanaged[K, V, C]) GetContext(key K, ctx C) Optional[V] {
	return self.GetAdapted(key, ctx)
}

// GetAdapted is HashMapUnmanaged.getAdapted.
//
// Zig: key: anytype, ctx: anytype.
func (self hashMapUnmanaged[K, V, C]) GetAdapted(key K, ctx C) Optional[V] {
	if idx := self.getIndex(key, ctx); idx.Valid {
		return Some(self.values()[idx.Value])
	}
	return None[V]()
}

// GetOrPut is HashMapUnmanaged.getOrPut.
func (self hashMapUnmanaged[K, V, C]) GetOrPut(key K) (HashMapGetOrPutResult[K, V], error) {
	// Zig: a compile error unless @sizeOf(Context) == 0; the context is
	// `undefined`, in Go the zero value of C.
	var ctx C
	return self.GetOrPutContext(key, ctx)
}

// GetOrPutContext is HashMapUnmanaged.getOrPutContext.
func (self hashMapUnmanaged[K, V, C]) GetOrPutContext(key K, ctx C) (HashMapGetOrPutResult[K, V], error) {
	gop, err := self.GetOrPutContextAdapted(key, ctx)
	if err != nil {
		return HashMapGetOrPutResult[K, V]{}, err
	}
	if !gop.found_existing {
		*gop.key_ptr = key
	}
	return gop, nil
}

// GetOrPutAdapted is HashMapUnmanaged.getOrPutAdapted.
//
// Zig: key: anytype, key_ctx: anytype.
func (self hashMapUnmanaged[K, V, C]) GetOrPutAdapted(key K, key_ctx C) (HashMapGetOrPutResult[K, V], error) {
	// Zig: a compile error unless @sizeOf(Context) == 0.
	return self.GetOrPutContextAdapted(key, key_ctx)
}

// GetOrPutContextAdapted is HashMapUnmanaged.getOrPutContextAdapted.
//
// Zig: key: anytype, key_ctx: anytype.
func (self hashMapUnmanaged[K, V, C]) GetOrPutContextAdapted(key K, key_ctx C) (HashMapGetOrPutResult[K, V], error) {
	if err := self.growIfNeeded(1); err != nil {
		// If allocation fails, try to do the lookup anyway.
		// If we find an existing item, we can return it.
		// Otherwise return the error, we could not add another.
		index := self.getIndex(key, key_ctx)
		if !index.Valid {
			return HashMapGetOrPutResult[K, V]{}, err
		}
		return HashMapGetOrPutResult[K, V]{
			key_ptr:        &self.keys()[index.Value],
			value_ptr:      &self.values()[index.Value],
			found_existing: true,
		}, nil
	}
	return self.GetOrPutAssumeCapacityAdapted(key, key_ctx), nil
}

// GetOrPutAssumeCapacity is HashMapUnmanaged.getOrPutAssumeCapacity.
func (self hashMapUnmanaged[K, V, C]) GetOrPutAssumeCapacity(key K) HashMapGetOrPutResult[K, V] {
	// Zig: a compile error unless @sizeOf(Context) == 0; the context is
	// `undefined`, in Go the zero value of C.
	var ctx C
	return self.GetOrPutAssumeCapacityContext(key, ctx)
}

// GetOrPutAssumeCapacityContext is
// HashMapUnmanaged.getOrPutAssumeCapacityContext.
func (self hashMapUnmanaged[K, V, C]) GetOrPutAssumeCapacityContext(key K, ctx C) HashMapGetOrPutResult[K, V] {
	result := self.GetOrPutAssumeCapacityAdapted(key, ctx)
	if !result.found_existing {
		*result.key_ptr = key
	}
	return result
}

// GetOrPutAssumeCapacityAdapted is
// HashMapUnmanaged.getOrPutAssumeCapacityAdapted.
//
// Zig: key: anytype, ctx: anytype.
func (self hashMapUnmanaged[K, V, C]) GetOrPutAssumeCapacityAdapted(key K, ctx C) HashMapGetOrPutResult[K, V] {
	// If you get a compile error on this line, it means that your generic hash
	// function is invalid for these parameters.
	hash := ctx.Hash(key)
	capacity := self.Capacity()
	assert(capacity >= 1) // Zig: `capacity() - 1` traps on overflow
	mask := capacity - 1
	fingerprint := hashMapMetadataTakeFingerprint(hash)
	limit := self.Capacity()
	idx := uint(hash & uint64(mask)) // @truncate to usize

	first_tombstone_idx := uint(self.Capacity()) // invalid index
	// var metadata = self.metadata.? + idx;
	for !self.metadata[idx].IsFree() && limit != 0 {
		if self.metadata[idx].IsUsed() && self.metadata[idx].fingerprint == fingerprint {
			test_key := &self.keys()[idx]
			// If you get a compile error on this line, it means that your generic eql
			// function is invalid for these parameters.
			eql := ctx.Eql(key, *test_key)
			if eql {
				return HashMapGetOrPutResult[K, V]{
					key_ptr:        test_key,
					value_ptr:      &self.values()[idx],
					found_existing: true,
				}
			}
		} else if first_tombstone_idx == uint(self.Capacity()) && self.metadata[idx].IsTombstone() {
			first_tombstone_idx = idx
		}

		limit -= 1
		idx = (idx + 1) & uint(mask)
	}

	if first_tombstone_idx < uint(self.Capacity()) {
		// Cheap try to lower probing lengths after deletions. Recycle a tombstone.
		idx = first_tombstone_idx
	}

	self.metadata[idx].Fill(fingerprint)
	new_key := &self.keys()[idx]
	new_value := &self.values()[idx]
	// new_key.* = undefined; new_value.* = undefined;
	var zero_key K
	var zero_val V
	*new_key = zero_key
	*new_value = zero_val
	hdr := self.header()
	assert(hdr.size < math.MaxUint32) // Zig: `+=` traps on overflow
	hdr.size += 1

	return HashMapGetOrPutResult[K, V]{
		key_ptr:        new_key,
		value_ptr:      new_value,
		found_existing: false,
	}
}

// GetOrPutValue is HashMapUnmanaged.getOrPutValue.
func (self hashMapUnmanaged[K, V, C]) GetOrPutValue(key K, value V) (HashMapEntry[K, V], error) {
	// Zig: a compile error unless @sizeOf(Context) == 0; the context is
	// `undefined`, in Go the zero value of C.
	var ctx C
	return self.GetOrPutValueContext(key, value, ctx)
}

// GetOrPutValueContext is HashMapUnmanaged.getOrPutValueContext.
func (self hashMapUnmanaged[K, V, C]) GetOrPutValueContext(key K, value V, ctx C) (HashMapEntry[K, V], error) {
	res, err := self.GetOrPutAdapted(key, ctx)
	if err != nil {
		return HashMapEntry[K, V]{}, err
	}
	if !res.found_existing {
		*res.key_ptr = key
		*res.value_ptr = value
	}
	return HashMapEntry[K, V]{key_ptr: res.key_ptr, value_ptr: res.value_ptr}, nil
}

// Contains is HashMapUnmanaged.contains: return true if there is a value
// associated with key in the map.
func (self hashMapUnmanaged[K, V, C]) Contains(key K) bool {
	// Zig: a compile error unless @sizeOf(Context) == 0; the context is
	// `undefined`, in Go the zero value of C.
	var ctx C
	return self.ContainsContext(key, ctx)
}

// ContainsContext is HashMapUnmanaged.containsContext.
func (self hashMapUnmanaged[K, V, C]) ContainsContext(key K, ctx C) bool {
	return self.ContainsAdapted(key, ctx)
}

// ContainsAdapted is HashMapUnmanaged.containsAdapted.
//
// Zig: key: anytype, ctx: anytype.
func (self hashMapUnmanaged[K, V, C]) ContainsAdapted(key K, ctx C) bool {
	return self.getIndex(key, ctx).Valid
}

// removeByIndex is HashMapUnmanaged.removeByIndex.
func (self hashMapUnmanaged[K, V, C]) removeByIndex(idx uint) {
	self.metadata[idx].Remove() // Zig: self.metadata.? (a nil slice panics on the index)
	// self.keys()[idx] = undefined; self.values()[idx] = undefined;
	var zero_key K
	var zero_val V
	self.keys()[idx] = zero_key
	self.values()[idx] = zero_val
	hdr := self.header()
	assert(hdr.size > 0) // Zig: `-=` traps on overflow
	hdr.size -= 1
}

// Remove is HashMapUnmanaged.remove.
//
// If there is an `Entry` with a matching key, it is deleted from
// the hash map, and this function returns true.  Otherwise this
// function returns false.
func (self hashMapUnmanaged[K, V, C]) Remove(key K) bool {
	// Zig: a compile error unless @sizeOf(Context) == 0; the context is
	// `undefined`, in Go the zero value of C.
	var ctx C
	return self.RemoveContext(key, ctx)
}

// RemoveContext is HashMapUnmanaged.removeContext.
func (self hashMapUnmanaged[K, V, C]) RemoveContext(key K, ctx C) bool {
	return self.RemoveAdapted(key, ctx)
}

// RemoveAdapted is HashMapUnmanaged.removeAdapted.
//
// Zig: key: anytype, ctx: anytype.
func (self hashMapUnmanaged[K, V, C]) RemoveAdapted(key K, ctx C) bool {
	if idx := self.getIndex(key, ctx); idx.Valid {
		self.removeByIndex(idx.Value)
		return true
	}

	return false
}

// RemoveByPtr is HashMapUnmanaged.removeByPtr.
//
// Delete the entry with key pointed to by key_ptr from the hash map.
// key_ptr is assumed to be a valid pointer to a key that is present
// in the hash map.
//
// Zig finds the index by pointer subtraction; see keyIndex.
func (self hashMapUnmanaged[K, V, C]) RemoveByPtr(key_ptr *K) {
	// TODO: replace with pointer subtraction once supported by zig
	// if @sizeOf(K) == 0 then there is at most one item in the hash
	// map, which is assumed to exist as key_ptr must be valid.  This
	// item must be at index 0.
	//
	// Go: keyIndex is the whole `if (@sizeOf(K) > 0) ... else 0`.
	idx := self.keyIndex(key_ptr)

	self.removeByIndex(idx)
}

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
func (self hashMapUnmanaged[K, V, C]) keyIndex(key_ptr *K) uint {
	// const idx = if (@sizeOf(K) > 0)
	//     (@intFromPtr(key_ptr) - @intFromPtr(self.keys())) / @sizeOf(K)
	// else
	//     0;
	size := unsafe.Sizeof(*key_ptr)
	if size == 0 {
		return 0
	}
	keys := self.keys()
	ptr := uintptr(unsafe.Pointer(key_ptr))
	base := uintptr(unsafe.Pointer(unsafe.SliceData(keys)))
	// Go-only (wireframe REQUIREMENT): the pointer must be an element of
	// keys. Zig's `-` traps when it lies before keys; the rest is checked
	// by comparing the element at the computed index.
	assert(ptr >= base)
	idx := uint((ptr - base) / size)
	assert(idx < uint(len(keys)) && &keys[idx] == key_ptr)
	return idx
}

// initMetadatas is HashMapUnmanaged.initMetadatas.
func (self hashMapUnmanaged[K, V, C]) initMetadatas() {
	// @memset(@as([*]u8, @ptrCast(self.metadata.?))[0 .. @sizeOf(Metadata) * self.capacity()], 0);
	assert(self.metadata != nil) // Zig: self.metadata.?
	clear(self.metadata[0:self.Capacity()])
}

// growIfNeeded is HashMapUnmanaged.growIfNeeded.
func (self hashMapUnmanaged[K, V, C]) growIfNeeded(new_count HashMapSize) error {
	capacity := self.Capacity()
	size := self.header().size
	assert(capacity >= size) // Zig: `-` traps on overflow
	available := capacity - size
	if new_count > available {
		return ErrOutOfMemory
	}
	return nil
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
	assert(new_capacity == 0 || new_capacity&(new_capacity-1) == 0) // std.math.isPowerOfTwo

	// Cast to usize to prevent overflow in size calculations.
	// See: https://github.com/ziglang/zig/pull/19048
	cap := uint64(new_capacity)

	// The comptime declarations header_align, key_align, val_align and
	// base_align (see the file header).
	const header_size = 16 // @sizeOf(Header)
	const header_align = 4 // @alignOf(Header)
	const metadata_size = 1
	key_align := sizes.key_align
	if sizes.key_size == 0 {
		key_align = 1
	}
	val_align := sizes.val_align
	if sizes.val_size == 0 {
		val_align = 1
	}
	base_align := max(header_align, key_align, val_align)

	// std.mem.alignForward(usize, addr, alignment), which asserts that the
	// alignment is a power of two. Nothing here can overflow a u64: cap is
	// at most 2^31.
	alignForward := func(addr uint64, alignment uint64) uint64 {
		assert(alignment != 0 && alignment&(alignment-1) == 0)
		return (addr + (alignment - 1)) &^ (alignment - 1)
	}

	// Pack our metadata, keys, and values.
	const meta_start = header_size
	meta_end := header_size + cap*metadata_size
	keys_start := alignForward(meta_end, key_align)
	keys_end := keys_start + cap*sizes.key_size
	vals_start := alignForward(keys_end, val_align)
	vals_end := vals_start + cap*sizes.val_size

	// Our total memory size required is the end of our values
	// aligned to the base required alignment.
	total_size := alignForward(
		vals_end,
		base_align,
	)

	// The offsets we actually store in the map are from the
	// metadata pointer so that we can use self.metadata as
	// the base.
	keys_offset := keys_start - meta_start
	vals_offset := vals_start - meta_start

	return hashMapLayout{
		total_size: total_size,
		keys_start: keys_offset,
		vals_start: vals_offset,
		capacity:   new_capacity,
	}
}
