// WIREFRAME for src/terminal/bitmap_allocator.zig (Ghostty v1.3.1). NOT YET
// PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
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
// The file has no C ABI declarations.

package terminal

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
}

// BitmapAllocatorInit is BitmapAllocator.init: initialize the allocator map
// with a given buf and memory layout.
//
// Zig: pub fn init(buf: OffsetBuf, l: Layout) Self. The buffer argument is
// dropped: the allocator makes its own `bitmap` (`l.bitmap_count` entries,
// all bits set: every chunk free) and `chunks`. `chunk_size` is the comptime
// parameter of the Zig type.
func BitmapAllocatorInit[T any](chunk_size uint64, l BitmapAllocatorLayout) BitmapAllocator[T] {
	panic(notImplemented)
}

// BitmapAllocatorBytesRequired is BitmapAllocator.bytesRequired: returns the
// number of bytes required to allocate n elements of type T. This accounts
// for the chunk size alignment used by the bitmap allocator.
//
// Zig: pub fn bytesRequired(comptime T: type, n: usize) usize, a function of
// the type; `chunk_size` is the comptime parameter of the Zig type. Called as
// BitmapAllocatorBytesRequired[uint32](16, n).
func BitmapAllocatorBytesRequired[T any](chunk_size uint64, n uint) uint64 {
	panic(notImplemented)
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
func (self *BitmapAllocator[T]) Alloc(n uint) ([]T, error) { panic(notImplemented) }

// Free is BitmapAllocator.free.
//
// Zig: pub fn free(self: *Self, base: anytype, slice: anytype) void. The
// slice must be one returned by Alloc (same start, same length): Zig finds
// the chunk index from the slice's address.
func (self *BitmapAllocator[T]) Free(slice []T) { panic(notImplemented) }

// CapacityBytes is BitmapAllocator.capacityBytes: returns the total
// capacity in bytes.
func (self BitmapAllocator[T]) CapacityBytes() uint64 { panic(notImplemented) }

// UsedBytes is BitmapAllocator.usedBytes: returns the number of bytes
// currently in use.
func (self BitmapAllocator[T]) UsedBytes() uint64 { panic(notImplemented) }

// isAllocated is BitmapAllocator.isAllocated: for testing only.
func (self *BitmapAllocator[T]) isAllocated(slice []T) bool { panic(notImplemented) }

// dumpBitmaps is BitmapAllocator.dumpBitmaps: for debugging.
func (self *BitmapAllocator[T]) dumpBitmaps() { panic(notImplemented) }

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
	panic(notImplemented)
}

// bitmapAllocatorFindFreeChunks is bitmap_allocator.findFreeChunks: find `n`
// sequential free chunks in the given bitmaps and return the index of the
// first chunk. If no chunks are found, return `null`. This also updates the
// bitmap to mark the chunks as used.
func bitmapAllocatorFindFreeChunks(bitmaps []uint64, n uint) Optional[uint] {
	panic(notImplemented)
}
