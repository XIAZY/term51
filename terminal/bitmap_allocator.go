// Ported from src/terminal/bitmap_allocator.zig (Ghostty v1.3.1).
//
// File naming: bitmap_allocator.zig holds the type function
// `BitmapAllocator(chunk_size)` and the private free function
// `findFreeChunks`. Everything nested in the type takes the prefix
// `BitmapAllocator`; the private free function takes the lowerCamel file
// prefix: `bitmapAllocatorFindFreeChunks`.
//
// Declarations appear in the same order as in bitmap_allocator.zig.
//
// # Generic form
//
// Zig: `BitmapAllocator(comptime chunk_size: comptime_int) type`. The
// allocator itself is untyped: it hands out `[]T` for any `T` given to
// `alloc`, out of one byte region. Go differs in two ways:
//
//   - Go generics cannot take a value parameter, so `chunk_size` is the
//     field `chunk_size`, set by BitmapAllocatorInit. The functions that
//     Zig attaches to the type and that have no receiver (`bytesRequired`,
//     `layout`) take `chunk_size` as their first parameter.
//   - A Go allocator owns a slice of one element type, so the element type
//     is the type parameter: `BitmapAllocator[T any]`. The page uses
//     `BitmapAllocator[uint8]` with chunk size 32 (strings) and
//     `BitmapAllocator[uint32]` with chunk size 16 (grapheme codepoints,
//     Zig `u21`, @sizeOf 4).
//   - `Layout` uses no type parameter and is the plain struct
//     `BitmapAllocatorLayout`. The function `layout(cap)` would have the
//     same Go name as the type `Layout`, so the function is
//     `BitmapAllocatorLayoutOf`.
//
// The chunk size and every size the allocator reports stay in BYTES, as in
// Zig. The implementation therefore needs @sizeOf(T): 1 for uint8, 4 for
// uint32 (u21). For both instantiations Go's element size equals Zig's.
// One chunk is `chunk_size / @sizeOf(T)` elements.
//
// # Go-memory rule, option A
//
// (docs/mapping.md, governing rule 3, as refined for this batch: ordinary Go
// memory, but upstream's bookkeeping and byte arithmetic are kept exactly,
// because fill points and byte sizes are visible behaviour.)
//
//   - field `bitmap: Offset(u64)` is `bitmap []uint64`, `bitmap_count`
//     entries;
//   - field `chunks: Offset(u8)` is `chunks []T`, the backing slice that
//     every allocation is a sub-slice of. It holds what the bitmaps can hand
//     out: `bitmap_count * 64` chunks. (Zig's layout reserves
//     `aligned_cap * chunk_size` bytes, `chunk_size` times more than that;
//     the number is kept in `Layout.total_size`, the memory is not.)
//   - every `base` parameter is dropped; `init` drops `buf: OffsetBuf`;
//   - `alloc(T, base, n)` is `Alloc(n)`, `free(base, slice)` is
//     `Free(slice)`;
//   - `Layout` keeps every field as a number.
//
// Omitted under the Go-memory rule:
//   - `base_align` (alignment of the base pointer).
//
// Widths: every byte size and byte offset is a `uint64`, whatever the
// platform word size, because the reference is the recorded 64-bit data and
// the target is 32-bit. Counts, indexes and lengths stay `uint` or their
// fixed Zig width.
// Here: `chunk_size`, the `cap` given to `layout`, the results of
// `bytesRequired`, `capacityBytes` and `usedBytes`, and `Layout.total_size`,
// `bitmap_start` and `chunks_start` are uint64; `bitmap_count`, `n` and
// chunk indexes are uint.
//
// # Implementation notes
//
//   - @sizeOf(T) and @alignOf(T) are comptime values in Zig. Go reads them
//     once, in BitmapAllocatorInit, from reflect.TypeFor[T]() into the
//     added fields `elem_size` and `elem_align` (not Zig fields; the
//     `has_deleted` precedent in ref_counted_set.go).
//     BitmapAllocatorBytesRequired has no receiver and reads @sizeOf(T) per
//     call through bitmapAllocatorSizeOf. Go's sizes equal Zig's for every
//     element type used (u8, u16, u21 = uint32, u64, [3]u8); Go's alignment
//     equals Zig's except for 8-byte scalars on 32-bit ARM (Go 4, Zig 8),
//     which only the `chunk_size % @alignOf(T)` assertion of `alloc` reads,
//     and only with a chunk size below 8 (the page uses 16 and 32).
//   - Zig's `@intFromPtr(slice.ptr) - @intFromPtr(chunks)` (free,
//     isAllocated) is the difference of the two slices' first-element
//     addresses as given by reflect.Value.Pointer (bitmapAllocatorAddr).
//     The addresses are only subtracted, never turned back into pointers;
//     no package unsafe is used.
//   - Zig traps on integer overflow (`+`, `*`) in a safe build; the byte
//     arithmetic of `layout` and `bytesRequired` asserts where it could
//     overflow a u64 (bitmapAllocatorAlignForward, bitmapAllocatorMul).
//
// The file has no C ABI declarations.

package terminal

import (
	"math"
	"math/bits"
	"reflect"
)

// BitmapAllocatorBitmapBitSize is BitmapAllocator.bitmap_bit_size:
// @bitSizeOf(u64).
const BitmapAllocatorBitmapBitSize = 64

// BitmapAllocator is the type returned by
// bitmap_allocator.BitmapAllocator(chunk_size).
//
// A relatively naive bitmap allocator that uses memory offsets against
// a fixed backing buffer so that the backing buffer can be easily moved
// without having to update pointers.
//
// The chunk size determines the size of each chunk in bytes. This is the
// minimum distributed unit of memory. For example, if you request a
// 1-byte allocation, you'll use a chunk of chunk_size bytes. Likewise,
// if your chunk size is 4, and you request a 5-byte allocation, you'll
// use 2 chunks.
//
// The allocator is susceptible to fragmentation. If you allocate and free
// memory in a way that leaves small holes in the memory, you may not be
// able to allocate large chunks of memory even if there is enough free
// memory in aggregate. To avoid fragmentation, use a chunk size that is
// large enough to cover most of your allocations.
//
// Notes for contributors: this is highly contributor friendly part of
// the code. If you can improve this, add tests, show benchmarks, then
// please do so!
//
// Zig asserts at compile time that chunk_size is a power of two.
//
// The Go zero value is not an initialised allocator; use
// BitmapAllocatorInit.
type BitmapAllocator[T any] struct {
	// Zig: the comptime parameter `chunk_size`, in bytes. See the file
	// header.
	chunk_size uint64

	// The bitmap of available chunks. Each bit represents a chunk. A
	// 1 means the chunk is free and a 0 means it's used. We use 1
	// for free since it makes it very slightly faster to find free
	// chunks.
	//
	// Zig: Offset(u64).
	bitmap       []uint64
	bitmap_count uint

	// The contiguous buffer of chunks.
	//
	// Zig: Offset(u8). See the file header.
	chunks []T

	// Not Zig fields: @sizeOf(T) and @alignOf(T), comptime in Zig, read
	// once by BitmapAllocatorInit. See the file header.
	elem_size  uint64
	elem_align uint64
}

// BitmapAllocatorInit is BitmapAllocator.init: initialize the allocator map
// with a given buf and memory layout.
//
// Zig: pub fn init(buf: OffsetBuf, l: Layout) Self. The buffer argument is
// dropped: the allocator makes its own `bitmap` (`l.bitmap_count` entries,
// all bits set: every chunk free) and `chunks`. `chunk_size` is the comptime
// parameter of the Zig type.
func BitmapAllocatorInit[T any](chunk_size uint64, l BitmapAllocatorLayout) BitmapAllocator[T] {
	// Zig: comptime { assert(std.math.isPowerOfTwo(chunk_size)); } in the
	// type function; a run-time assertion here.
	assert(chunk_size != 0 && chunk_size&(chunk_size-1) == 0)

	// Zig: assert(base_align.check(@intFromPtr(buf.start()))). No buffer
	// in Go.

	// Initialize our bitmaps to all 1s to note that all chunks are free.
	bitmap := make([]uint64, l.bitmap_count)
	for i := range bitmap {
		bitmap[i] = math.MaxUint64
	}

	// Zig: .chunks = buf.member(u8, l.chunks_start). Go: the chunks the
	// bitmaps can hand out, bitmap_count * 64 * chunk_size bytes, as
	// elements of T. See the file header.
	t := reflect.TypeFor[T]()
	elem_size := uint64(t.Size())
	chunks_bytes := bitmapAllocatorMul(uint64(l.bitmap_count)*BitmapAllocatorBitmapBitSize, chunk_size)
	var chunks_len uint64
	if elem_size != 0 {
		chunks_len = chunks_bytes / elem_size
	}
	assert(chunks_len <= math.MaxInt) // Go: a slice length is an int

	return BitmapAllocator[T]{
		chunk_size:   chunk_size,
		bitmap:       bitmap,
		bitmap_count: l.bitmap_count,
		chunks:       make([]T, chunks_len),
		elem_size:    elem_size,
		elem_align:   uint64(t.Align()),
	}
}

// BitmapAllocatorBytesRequired is BitmapAllocator.bytesRequired: returns the
// number of bytes required to allocate n elements of type T. This accounts
// for the chunk size alignment used by the bitmap allocator.
//
// Zig: pub fn bytesRequired(comptime T: type, n: usize) usize, a function of
// the type; `chunk_size` is the comptime parameter of the Zig type. Called as
// BitmapAllocatorBytesRequired[uint32](16, n).
func BitmapAllocatorBytesRequired[T any](chunk_size uint64, n uint) uint64 {
	byte_count := bitmapAllocatorMul(bitmapAllocatorSizeOf[T](), uint64(n))
	return bitmapAllocatorAlignForward(byte_count, chunk_size)
}

// Alloc is BitmapAllocator.alloc: allocate n elements of type T. This will
// return error.OutOfMemory if there isn't enough space in the backing
// buffer.
//
// Zig: pub fn alloc(self: *Self, comptime T: type, base: anytype, n: usize)
// Allocator.Error![]T. The result is a sub-slice of `chunks` starting at a
// chunk boundary. Zig asserts n > 0. The error is ErrOutOfMemory (dcs.go).
//
// REQUIREMENT on the implementation: the returned slice has length n and a
// capacity that reaches the end of its last chunk (n rounded up to whole
// chunks), no further. Page.appendGrapheme writes into the unused rest of
// the last chunk (`cps[slice.len] = cp; slice.len += 1`) without
// allocating, which in Go is a reslice within that capacity.
func (self *BitmapAllocator[T]) Alloc(n uint) ([]T, error) {
	// note: we don't handle alignment yet, we just require that all
	// types are properly aligned. This is a limitation that should be
	// fixed but we haven't needed it. Contributor friendly: add tests
	// and fix this.
	assert(self.chunk_size%self.elem_align == 0)
	assert(n > 0)

	// std.math.mul(usize, @sizeOf(T), n) catch return error.OutOfMemory
	hi, byte_count := bits.Mul64(self.elem_size, uint64(n))
	if hi != 0 {
		return nil, ErrOutOfMemory
	}
	// std.math.divCeil(usize, byte_count, chunk_size) catch
	// return error.OutOfMemory: divCeil fails only on a zero denominator,
	// which a power of two is not.
	var chunk_count uint64
	if byte_count > 0 {
		chunk_count = (byte_count-1)/self.chunk_size + 1
	} else {
		chunk_count = byte_count / self.chunk_size
	}

	// Go: findFreeChunks takes the count as a uint. A count that does not
	// fit one (only possible on a 32-bit target, where Zig's reference
	// usize is wider) exceeds every chunk the bitmaps hold, so
	// findFreeChunks would return null: the same error.
	if chunk_count > math.MaxUint {
		return nil, ErrOutOfMemory
	}

	// Find the index of the free chunk. This also marks it as used.
	bitmaps := self.bitmap
	idx := bitmapAllocatorFindFreeChunks(bitmaps[0:self.bitmap_count], uint(chunk_count))
	if !idx.Valid {
		return nil, ErrOutOfMemory
	}

	// const ptr: [*]T = @ptrCast(@alignCast(&chunks[idx * chunk_size]));
	// return ptr[0..n];
	//
	// Go: the byte offset idx * chunk_size as an element index of `chunks`,
	// and a capacity up to the end of the last chunk (see REQUIREMENT
	// above). A chunk that does not start on a whole element (an element
	// size that does not divide the chunk size) cannot be expressed as a Go
	// slice of T: asserted.
	start_byte := uint64(idx.Value) * self.chunk_size
	end_byte := start_byte + chunk_count*self.chunk_size
	assert(start_byte%self.elem_size == 0)
	start := start_byte / self.elem_size
	end := end_byte / self.elem_size
	return self.chunks[start : start+uint64(n) : end], nil
}

// Free is BitmapAllocator.free.
//
// Zig: pub fn free(self: *Self, base: anytype, slice: anytype) void. The
// slice must be one returned by Alloc (same start, same length): Zig finds
// the chunk index from the slice's address.
//
// REQUIREMENT on the implementation: Free must use only the slice's start
// position (its element offset within `chunks`) and its length, as Zig uses
// only `slice.ptr` and `slice.len`; it must not use the slice's capacity.
// Tests free sub-slices obtained by re-slicing an allocation, whose
// capacity differs from what Alloc returned.
//
// BitmapAllocator[T] hands out one element type T. Upstream tests that
// allocate values of mixed types from one allocator (for example u8 and
// u21) are scaled to a single element type in Go.
func (self *BitmapAllocator[T]) Free(slice []T) {
	// Convert the slice of whatever type to a slice of bytes. We
	// can then use the byte len and chunk size to determine the
	// number of chunks that were allocated.
	bytes_len := uint64(len(slice)) * self.elem_size // std.mem.sliceAsBytes(slice).len
	aligned_len := bitmapAllocatorAlignForward(bytes_len, self.chunk_size)
	chunk_count := aligned_len / self.chunk_size // @divExact: exact by construction

	// From the pointer, we can calculate the exact index.
	chunk_idx := self.chunkIndexOf(slice)

	bitmaps := self.bitmap

	// Current bitmap index.
	i := chunk_idx / 64
	// Number of chunks we still have to mark as free.
	rem := chunk_count

	// Mark any bits in the starting bitmap that need to be marked.
	{
		// Bit index.
		bit := chunk_idx % 64
		// Number of bits we need to mark in this bitmap.
		bits := min(rem, 64-bit)

		// Zig: @intCast(64 - bits) to a u6 traps when bits is 0 (an empty
		// slice).
		assert(bits > 0)
		bitmaps[i] |= ^uint64(0) >> (64 - bits) << bit
		rem -= bits
	}

	// Mark any full bitmaps worth of bits that need to be marked.
	i += 1
	for ; rem > 64; i += 1 {
		bitmaps[i] = math.MaxUint64
		rem -= 64
	}

	// Mark any bits at the start of this last bitmap if it needs it.
	if rem > 0 {
		bitmaps[i] |= ^uint64(0) >> (64 - rem)
	}
}

// CapacityBytes is BitmapAllocator.capacityBytes: returns the total
// capacity in bytes.
func (self BitmapAllocator[T]) CapacityBytes() uint64 {
	return uint64(self.bitmap_count) * BitmapAllocatorBitmapBitSize * self.chunk_size
}

// UsedBytes is BitmapAllocator.usedBytes: returns the number of bytes
// currently in use.
func (self BitmapAllocator[T]) UsedBytes() uint64 {
	bitmaps := self.bitmap
	var free_chunks uint64
	for _, bitmap := range bitmaps[0:self.bitmap_count] {
		free_chunks += uint64(bits.OnesCount64(bitmap))
	}
	total_chunks := uint64(self.bitmap_count) * BitmapAllocatorBitmapBitSize
	return (total_chunks - free_chunks) * self.chunk_size
}

// isAllocated is BitmapAllocator.isAllocated: for testing only.
func (self *BitmapAllocator[T]) isAllocated(slice []T) bool {
	// Zig: comptime assert(@import("builtin").is_test). Not checked in Go.

	bytes_len := uint64(len(slice)) * self.elem_size // std.mem.sliceAsBytes(slice).len
	aligned_len := bitmapAllocatorAlignForward(bytes_len, self.chunk_size)
	chunk_count := aligned_len / self.chunk_size // @divExact: exact by construction

	chunk_idx := self.chunkIndexOf(slice)

	bitmaps := self.bitmap

	for i := chunk_idx; i < chunk_idx+chunk_count; i++ {
		bitmap := i / BitmapAllocatorBitmapBitSize
		bit := i % BitmapAllocatorBitmapBitSize
		if bitmaps[bitmap]&(uint64(1)<<bit) != 0 {
			return false
		}
	}

	return true
}

// dumpBitmaps is BitmapAllocator.dumpBitmaps: for debugging.
func (self *BitmapAllocator[T]) dumpBitmaps() {
	bitmaps := self.bitmap
	for idx, bitmap := range bitmaps[0:self.bitmap_count] {
		// std.log.warn("bm={b} idx={}", .{ bitmap, idx }): the default
		// scope.
		warn("default", "bm=%b idx=%d", bitmap, idx)
	}
}

// chunkIndexOf is NOT a Zig declaration. It is the expression free and
// isAllocated share:
//
//	@divExact(@intFromPtr(slice.ptr) - @intFromPtr(chunks), chunk_size)
//
// with Zig's traps kept: the subtraction underflows when the slice starts
// before `chunks`, and @divExact fails when the byte distance is not a
// whole number of chunks. Go-only guard: a slice that starts past the end
// of `chunks` (Zig would go on to write bitmap words outside the bitmaps,
// which it does not bounds-check) also panics.
func (self *BitmapAllocator[T]) chunkIndexOf(slice []T) uint64 {
	ptr := uint64(bitmapAllocatorAddr(slice))
	chunks := uint64(bitmapAllocatorAddr(self.chunks))
	assert(ptr >= chunks) // Zig: `-` traps on overflow
	diff := ptr - chunks
	assert(diff <= uint64(len(self.chunks))*self.elem_size) // Go-only guard
	assert(diff%self.chunk_size == 0)                       // Zig: @divExact
	return diff / self.chunk_size
}

// BitmapAllocatorLayout is BitmapAllocator.Layout. Every field is kept:
// `bitmap_count` is a count, the other three are numbers of bytes.
type BitmapAllocatorLayout struct {
	total_size   uint64
	bitmap_count uint
	bitmap_start uint64
	chunks_start uint64
}

// BitmapAllocatorLayoutOf is BitmapAllocator.layout: get the layout for the
// given capacity. The capacity is in number of bytes, not chunks. The
// capacity will likely be rounded up to the nearest chunk size and bitmap
// size so everything is perfectly divisible.
//
// Zig: pub fn layout(cap: usize) Layout, a function of the type;
// `chunk_size` is the comptime parameter of the Zig type. It uses no element
// type, so it is not generic. Named `...LayoutOf` because `...Layout` is the
// type.
func BitmapAllocatorLayoutOf(chunk_size uint64, cap uint64) BitmapAllocatorLayout {
	// Align the cap forward to our chunk size so we always have
	// a full chunk at the end.
	aligned_cap := bitmapAllocatorAlignForward(cap, chunk_size)

	// Calculate the number of bitmaps. We need 1 bitmap per 64 chunks.
	// We align the chunk count forward so our bitmaps are full so we
	// don't have to handle the case where we have a partial bitmap.
	chunk_count := aligned_cap / chunk_size // @divExact: exact by construction
	aligned_chunk_count := bitmapAllocatorAlignForward(chunk_count, 64)
	bitmap_count := aligned_chunk_count / 64 // @divExact: exact by construction
	assert(bitmap_count <= math.MaxUint)     // Go: the count is a uint

	const bitmap_start = 0
	bitmap_end := 8 * bitmap_count                             // @sizeOf(u64) * bitmap_count
	chunks_start := bitmapAllocatorAlignForward(bitmap_end, 1) // @alignOf(u8)
	chunks_end := chunks_start + bitmapAllocatorMul(aligned_cap, chunk_size)
	assert(chunks_end >= chunks_start) // Zig: `+` traps on overflow
	total_size := chunks_end

	return BitmapAllocatorLayout{
		total_size:   total_size,
		bitmap_count: uint(bitmap_count),
		bitmap_start: bitmap_start,
		chunks_start: chunks_start,
	}
}

// bitmapAllocatorFindFreeChunks is bitmap_allocator.findFreeChunks: find `n`
// sequential free chunks in the given bitmaps and return the index of the
// first chunk. If no chunks are found, return `null`. This also updates the
// bitmap to mark the chunks as used.
func bitmapAllocatorFindFreeChunks(bitmaps []uint64, n uint) Optional[uint] {
	// NOTE: This is a naive implementation that just iterates through the
	// bitmaps. There is very likely a more efficient way to do this but
	// I'm not a bit twiddling expert. Perhaps even SIMD could be used here
	// but unsure. Contributor friendly: let's benchmark and improve this!

	// Large chunks require special handling.
	if n > 64 { // @bitSizeOf(u64)
		var i uint = 0
	search:
		for i < uint(len(bitmaps)) {
			// Number of chunks available at the end of this bitmap.
			prefix := uint(bits.LeadingZeros64(^bitmaps[i]))

			// If there are no chunks available at the end of this bitmap
			// then we can't start in it, so we'll try the next one.
			if prefix == 0 {
				i += 1
				continue
			}

			// Starting position if we manage to find the span we need here.
			start_bitmap := i
			start_bit := 64 - prefix

			// The remaining number of sequential free chunks we need to find.
			rem := n - prefix

			i += 1
			for ; rem > 64; i += 1 {
				// We ran out of bitmaps, there's no sufficiently large gap.
				if i >= uint(len(bitmaps)) {
					return None[uint]()
				}

				// There's more than 64 remaining chunks and this bitmap has
				// content in it, so we try starting again with this bitmap.
				if bitmaps[i] != math.MaxUint64 {
					continue search
				}

				// This bitmap is completely free, we can subtract 64 from
				// our remaining number.
				rem -= 64
			}

			// term51 reference fix (patches-reference/0001, the expected
			// output where the pinned upstream aborts with an out-of-bounds
			// read at bitmap_allocator.zig:281): the loop above only checks
			// the bounds while more than 64 chunks remain. If it ends
			// exactly at the end of the bitmaps there is no bitmap left to
			// hold the tail.
			if i >= uint(len(bitmaps)) {
				return None[uint]()
			}

			// If the number of available chunks at the start of this bitmap
			// is less than the remaining required, we have to try again.
			if uint(bits.TrailingZeros64(^bitmaps[i])) < rem {
				continue
			}

			suffix := (n - prefix) % 64

			// Found! Mark everything between our start and end as full.
			bitmaps[start_bitmap] ^= ^uint64(0) >> start_bit << start_bit
			full_bitmaps := (n - prefix - suffix) / 64
			full := bitmaps[start_bitmap+1:][0:full_bitmaps]
			for j := range full {
				full[j] = 0
			}
			if suffix > 0 {
				bitmaps[i] ^= ^uint64(0) >> (64 - suffix)
			}

			return Some(start_bitmap*64 + start_bit)
		}

		return None[uint]()
	}

	assert(n <= 64) // @bitSizeOf(u64)
	for idx := range bitmaps {
		bitmap := &bitmaps[idx]
		// Shift the bitmap to find `n` sequential free chunks.
		// EXAMPLE:
		// n = 4
		// shifted = 001111001011110010
		//         & 000111100101111001
		//         & 000011110010111100
		//         & 000001111001011110
		//         = 000001000000010000
		//                ^       ^
		// In this example there are 2 places with at least 4 sequential 1s.
		shifted := *bitmap
		// Zig: `for (1..n)` traps when n is 0 (start larger than end).
		assert(n >= 1)
		for i := uint(1); i < n; i++ {
			shifted &= *bitmap >> i
		}

		// If we have zero then we have no matches
		if shifted == 0 {
			continue
		}

		// Trailing zeroes gets us the index of the first bit index with at
		// least `n` sequential 1s. In the example above, that would be `4`.
		bit := uint(bits.TrailingZeros64(shifted))

		// Calculate the mask so we can mark it as used
		mask := (uint64(math.MaxUint64) >> (64 - n)) << bit
		*bitmap ^= mask

		return Some(uint(idx)*64 + bit)
	}

	return None[uint]()
}

// bitmapAllocatorSizeOf is NOT a Zig declaration: @sizeOf(T), for
// BitmapAllocatorBytesRequired, which has no allocator to read it from. See
// the file header.
func bitmapAllocatorSizeOf[T any]() uint64 {
	return uint64(reflect.TypeFor[T]().Size())
}

// bitmapAllocatorAddr is NOT a Zig declaration: @intFromPtr(slice.ptr), the
// address of the slice's first element (also for an empty slice; 0 for a
// nil slice). See the file header.
func bitmapAllocatorAddr[T any](slice []T) uintptr {
	return reflect.ValueOf(slice).Pointer()
}

// bitmapAllocatorAlignForward is NOT a Zig declaration: std.mem.alignForward
// on usize (64 bits in the reference), which asserts that the alignment is a
// power of two and whose `addr + (alignment - 1)` traps on overflow.
func bitmapAllocatorAlignForward(addr uint64, alignment uint64) uint64 {
	assert(alignment != 0 && alignment&(alignment-1) == 0)
	assert(addr <= math.MaxUint64-(alignment-1)) // Zig: `+` traps on overflow
	return (addr + (alignment - 1)) &^ (alignment - 1)
}

// bitmapAllocatorMul is NOT a Zig declaration: Zig's `a * b` on usize (64
// bits in the reference), which traps on overflow.
func bitmapAllocatorMul(a, b uint64) uint64 {
	hi, lo := bits.Mul64(a, b)
	assert(hi == 0) // Zig: `*` traps on overflow
	return lo
}
