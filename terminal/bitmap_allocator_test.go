// Tests translated one for one from src/terminal/bitmap_allocator.zig
// (Ghostty v1.3.1 plus the term51 additional tests). Each Go test carries the
// exact Zig title in a "// Zig: test" comment.
//
// PROVISIONAL: option A (see mapping doc); the tests stand if the choice is reversed only for the parts that do not read byte offsets
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualSlices keep Zig's names and Zig's (expected,
//     actual) argument order. `try testing.expectError(error.OutOfMemory,
//     call)` is bmExpectError(t, ErrOutOfMemory, <the error of the call>).
//     `x.?` is zigUnwrap. A `?usize` compared with expectEqual is an
//     Optional[uint] (bmSome / bmNone).
//   - The Zig type function `BitmapAllocator(chunk_size)` is the generic Go
//     type BitmapAllocator[T] plus the chunk size passed to
//     BitmapAllocatorInit, BitmapAllocatorLayoutOf and
//     BitmapAllocatorBytesRequired. The element type is fixed per allocator:
//     where one Zig allocator hands out several element types (u8, u21, u64,
//     a 12-byte struct) the Go test uses one allocator of the smallest
//     element type involved and allocates the same number of BYTES
//     (n * @sizeOf(T) / @sizeOf(element)); chunk indexes and usedBytes are
//     unchanged, only slice lengths are scaled. Each such test says so.
//     u21 is uint32 (@sizeOf 4).
//   - Zig's test helper `term51WFixture` is bmFixture[T]: layout, allocator,
//     and the helpers alloc / free / usedBytes / chunkIndex / allocChunks /
//     expectUsed / expectAllFree / bitmaps. `fx.alloc(n)` is `try
//     fx.alloc(T, n)` (it stops the test on an error); `fx.allocErr(n)`
//     returns the error for the tests that inspect it. `term51WMax` is bmMax,
//     `term51WBitFree` is bmBitFree, `term51WModelFind` is bmModelFind,
//     `term51WTrailingFree` is bmTrailingFree, `term51WCheckLayoutSweep` is
//     bmCheckLayoutSweep, `term51WGraphemeAlloc` / `term51WStringAlloc` are
//     the chunk sizes bmGraphemeChunk (16) and bmStringChunk (32).
//   - Go memory: there is no backing buffer. `buf` bookkeeping
//     (`alignedAlloc`, `defer alloc.free(buf)`, `buf` arguments, `base_align`,
//     the `bitmap.offset` / `chunks.offset` fields, `OffsetBuf`) has no Go
//     counterpart and is dropped as "Zig-only". The pointer identities Zig
//     compares with @intFromPtr stay: a Go slice returned by Alloc is a
//     sub-slice of `chunks`, so bmAddr (the address of its first element)
//     compares and differs exactly as Zig's pointers do. `fx.baseOffset(s)`
//     (offset from the start of the buffer, which includes the bitmaps) has
//     no Go counterpart; those lines are Zig-only.
//   - `Free` is given sub-slices of an allocation (s[0:4], s[12:16], and
//     slices cut from `chunks` itself) as the Zig tests do: like Zig it only
//     uses the start address and the length of its argument.
//   - Shapes of Zig integer widths: sizes, offsets and capacities are
//     uint64, counts and chunk indexes uint (see bitmap_allocator.go).
//   - The random test uses Zig's own PRNG (std.Random.Xoshiro256, seeded
//     with SplitMix64) reproduced in bmXoshiro, so that the same inputs are
//     drawn; corpus_bitmap_test.go checks the generator against the recorded
//     op streams.

package terminal

import (
	"errors"
	"math"
	"math/bits"
	"testing"
	"unsafe"
)

// bmMax is term51WMax: maxInt(u64). A variable, so that shifts of it are not
// constant expressions (which Go rejects when they overflow).
var bmMax uint64 = math.MaxUint64

// The two chunk sizes page.zig uses: grapheme_chunk = 4 * @sizeOf(u21) = 16
// bytes and string_chunk = 32 * @sizeOf(u8) = 32 bytes (term51WGraphemeAlloc
// and term51WStringAlloc).
const (
	bmGraphemeChunk uint64 = 16
	bmStringChunk   uint64 = 32
)

// bmExpectError is Zig's `try testing.expectError(expected, actual)`.
func bmExpectError(t *testing.T, want error, got error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

// bmExpectPanic runs f and fails the test unless f panics. Zig's tests cannot
// test an assert or a safety-checked panic: the test would abort. The
// wireframe's "not implemented" panic does not count.
func bmExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		switch r := recover(); {
		case r == nil:
			t.Fatalf("expected a panic, got none")
		case r == notImplemented:
			t.Fatalf("got the wireframe's not-implemented panic, not the expected one")
		}
	}()
	f()
}

// bmResult is a value and error returned together; ok is Zig's `try`: it
// stops the test when the call returned an error. A Go call returning two
// values cannot be passed with a further argument, hence
// bmTry(bm.Alloc(1)).ok(t) for `try bm.alloc(...)`.
type bmResult[T any] struct {
	v   T
	err error
}

func bmTry[T any](v T, err error) bmResult[T] {
	return bmResult[T]{v: v, err: err}
}

func (r bmResult[T]) ok(t *testing.T) T {
	t.Helper()
	if r.err != nil {
		t.Fatalf("try failed: %v", r.err)
	}
	return r.v
}

// bmSome is the Zig literal `@as(?usize, n)`.
func bmSome(n uint) Optional[uint] { return Some(n) }

// bmNone is the Zig literal `@as(?usize, null)`.
func bmNone() Optional[uint] { return None[uint]() }

// bmFind is findFreeChunks.
func bmFind(bitmaps []uint64, n uint) Optional[uint] {
	return bitmapAllocatorFindFreeChunks(bitmaps, n)
}

// bmAddr is @intFromPtr(slice.ptr).
func bmAddr[T any](s []T) uintptr {
	return uintptr(unsafe.Pointer(unsafe.SliceData(s)))
}

// bmSizeOf is @sizeOf(T).
func bmSizeOf[T any]() uint64 {
	var z T
	return uint64(unsafe.Sizeof(z))
}

// bmFill is `@memset(slice, v)`.
func bmFill[T any](s []T, v T) {
	for i := range s {
		s[i] = v
	}
}

// bmRepeat is `&@as([n]T, @splat(v))`.
func bmRepeat[T any](v T, n uint) []T {
	out := make([]T, n)
	bmFill(out, v)
	return out
}

// bmFull is `[_]u64{term51WMax} ** n`.
func bmFull(n uint) []uint64 {
	return bmRepeat(bmMax, n)
}

// bmBitFree is term51WBitFree.
func bmBitFree(bitmaps []uint64, i uint) bool {
	return (bitmaps[i/64]>>(i%64))&1 == 1
}

// bmModelFind is term51WModelFind: reference model of findFreeChunks, the
// lowest run of n free chunks. Runs of up to 64 chunks may not span two
// bitmaps; larger runs span by necessity.
func bmModelFind(bitmaps []uint64, n uint) Optional[uint] {
	var run uint
	for j := uint(0); j < uint(len(bitmaps))*64; j++ {
		if n <= 64 && j%64 == 0 {
			run = 0
		}
		if bmBitFree(bitmaps, j) {
			run++
			if run >= n {
				return Some(j + 1 - n)
			}
		} else {
			run = 0
		}
	}
	return None[uint]()
}

// bmTrailingFree is term51WTrailingFree: length of the run of free chunks
// that ends at the very last chunk.
func bmTrailingFree(bitmaps []uint64) uint {
	var t uint
	j := uint(len(bitmaps)) * 64
	for j > 0 {
		if !bmBitFree(bitmaps, j-1) {
			break
		}
		t++
		j--
	}
	return t
}

// bmFixture is term51WFixture(chunk_size): an initialized allocator and its
// layout (the backing memory is the allocator's own in Go).
type bmFixture[T any] struct {
	t          *testing.T
	chunk_size uint64
	layout     BitmapAllocatorLayout
	bm         BitmapAllocator[T]
}

func bmFixtureInit[T any](t *testing.T, chunk_size uint64, cap_ uint64) *bmFixture[T] {
	t.Helper()
	l := BitmapAllocatorLayoutOf(chunk_size, cap_)
	return &bmFixture[T]{t: t, chunk_size: chunk_size, layout: l, bm: BitmapAllocatorInit[T](chunk_size, l)}
}

func (self *bmFixture[T]) bitmaps() []uint64 {
	return self.bm.bitmap[:self.bm.bitmap_count]
}

// allocErr is `fx.alloc(T, n)` as an error union.
func (self *bmFixture[T]) allocErr(n uint) ([]T, error) {
	return self.bm.Alloc(n)
}

// alloc is `try fx.alloc(T, n)`.
func (self *bmFixture[T]) alloc(n uint) []T {
	self.t.Helper()
	return bmTry(self.bm.Alloc(n)).ok(self.t)
}

func (self *bmFixture[T]) free(slice []T) {
	self.bm.Free(slice)
}

func (self *bmFixture[T]) usedBytes() uint64 {
	return self.bm.UsedBytes()
}

// chunkIndex is the index (in chunks) of a slice within the chunk area.
func (self *bmFixture[T]) chunkIndex(slice []T) uint {
	return uint((bmAddr(slice) - bmAddr(self.bm.chunks)) / uintptr(self.chunk_size))
}

// allocChunks allocates exactly n chunks worth of bytes (Zig: `[]u8`).
func (self *bmFixture[T]) allocChunks(n uint) []T {
	self.t.Helper()
	return self.alloc(uint(uint64(n) * self.chunk_size / bmSizeOf[T]()))
}

// expectUsed expects chunk j to be used iff it lies in one of the given
// [start, end) ranges, and free otherwise, for every chunk.
func (self *bmFixture[T]) expectUsed(ranges [][2]uint) {
	self.t.Helper()
	bms := self.bitmaps()
	for j := uint(0); j < uint(len(bms))*64; j++ {
		used := false
		for _, r := range ranges {
			if j >= r[0] && j < r[1] {
				used = true
			}
		}
		expectEqual(self.t, !used, bmBitFree(bms, j))
	}
}

func (self *bmFixture[T]) expectAllFree() {
	self.t.Helper()
	for _, bm := range self.bitmaps() {
		expectEqual(self.t, bmMax, bm)
	}
}

// bmCheckLayoutSweep is term51WCheckLayoutSweep.
func bmCheckLayoutSweep(t *testing.T, chunk_size uint64, max_cap uint64) {
	t.Helper()
	for cap_ := uint64(0); cap_ <= max_cap; cap_++ {
		l := BitmapAllocatorLayoutOf(chunk_size, cap_)
		aligned_cap := (cap_ + chunk_size - 1) / chunk_size * chunk_size
		bitmap_count := (cap_ + 64*chunk_size - 1) / (64 * chunk_size)
		expectEqual(t, uint64(0), l.bitmap_start)
		expectEqual(t, uint(bitmap_count), l.bitmap_count)
		expectEqual(t, 8*bitmap_count, l.chunks_start)
		// The documented over-reservation: aligned_cap * chunk_size bytes.
		expectEqual(t, l.chunks_start+aligned_cap*chunk_size, l.total_size)
	}
}

// bmXoshiro is std.Random.Xoshiro256 (Xoshiro256++), seeded through
// std.Random.SplitMix64 as `Xoshiro256.init(seed)` does, with the few
// std.Random methods the tests use. Zig's `usize` draws are u64 draws.
type bmXoshiro struct {
	s [4]uint64
}

func bmXoshiroInit(seed uint64) bmXoshiro {
	var x bmXoshiro
	sm := seed
	for i := range x.s {
		sm += 0x9e3779b97f4a7c15
		z := sm
		z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
		z = (z ^ (z >> 27)) * 0x94d049bb133111eb
		x.s[i] = z ^ (z >> 31)
	}
	return x
}

func (self *bmXoshiro) next() uint64 {
	r := bits.RotateLeft64(self.s[0]+self.s[3], 23) + self.s[0]
	t := self.s[1] << 17
	self.s[2] ^= self.s[0]
	self.s[3] ^= self.s[1]
	self.s[1] ^= self.s[2]
	self.s[0] ^= self.s[3]
	self.s[2] ^= t
	self.s[3] = bits.RotateLeft64(self.s[3], 45)
	return r
}

// intU64 is r.int(u64).
func (self *bmXoshiro) intU64() uint64 { return self.next() }

// boolean is r.boolean(): r.int(u1) reads one byte and truncates.
func (self *bmXoshiro) boolean() bool { return self.next()&1 != 0 }

// uintLessThanU8 is r.uintLessThan(u8, less) (Lemire's method, as in Zig).
func (self *bmXoshiro) uintLessThanU8(less uint8) uint8 {
	assert(0 < less)
	x := uint8(self.next())
	m := uint16(x) * uint16(less)
	l := uint8(m)
	if l < less {
		t := -less
		if t >= less {
			t -= less
			if t >= less {
				t %= less
			}
		}
		for l < t {
			x = uint8(self.next())
			m = uint16(x) * uint16(less)
			l = uint8(m)
		}
	}
	return uint8(m >> 8)
}

// uintLessThanU64 is r.uintLessThan(u64, less).
func (self *bmXoshiro) uintLessThanU64(less uint64) uint64 {
	assert(0 < less)
	x := self.next()
	hi, l := bits.Mul64(x, less)
	if l < less {
		t := -less
		if t >= less {
			t -= less
			if t >= less {
				t %= less
			}
		}
		for l < t {
			x = self.next()
			hi, l = bits.Mul64(x, less)
		}
	}
	return hi
}

// intRangeAtMostUsize is r.intRangeAtMost(usize, at_least, at_most) (usize is
// 64 bits in the reference build).
func (self *bmXoshiro) intRangeAtMostUsize(at_least, at_most uint64) uint64 {
	assert(at_least <= at_most)
	m := at_most - at_least
	if m == math.MaxUint64 {
		return at_least + self.intU64()
	}
	return at_least + self.uintLessThanU64(m+1)
}

// Zig: test "findFreeChunks single found"
func TestBitmapAllocator_findFreeChunks_single_found(t *testing.T) {
	bitmaps := []uint64{
		0b10000000_00000000_00000000_00000000_00000000_00000000_00001110_00000000,
	}
	idx := zigUnwrap(t, bmFind(bitmaps, 2))
	expectEqual(t, uint(9), idx)
	expectEqual(t,
		uint64(0b10000000_00000000_00000000_00000000_00000000_00000000_00001000_00000000),
		bitmaps[0],
	)
}

// Zig: test "findFreeChunks single not found"
func TestBitmapAllocator_findFreeChunks_single_not_found(t *testing.T) {
	bitmaps := []uint64{0b10000111_00000000_00000000_00000000_00000000_00000000_00000000_00000000}
	idx := bmFind(bitmaps, 4)
	expect(t, !idx.Valid, "idx == null")
}

// Zig: test "findFreeChunks multiple found"
func TestBitmapAllocator_findFreeChunks_multiple_found(t *testing.T) {
	bitmaps := []uint64{
		0b10000111_00000000_00000000_00000000_00000000_00000000_00000000_01110000,
		0b10000000_00111110_00000000_00000000_00000000_00000000_00111110_00000000,
	}
	idx := zigUnwrap(t, bmFind(bitmaps, 4))
	expectEqual(t, uint(73), idx)
	expectEqual(t,
		uint64(0b10000000_00111110_00000000_00000000_00000000_00000000_00100000_00000000),
		bitmaps[1],
	)
}

// Zig: test "findFreeChunks exactly 64 chunks"
func TestBitmapAllocator_findFreeChunks_exactly_64_chunks(t *testing.T) {
	bitmaps := []uint64{
		0b11111111_11111111_11111111_11111111_11111111_11111111_11111111_11111111,
	}
	idx := zigUnwrap(t, bmFind(bitmaps, 64))
	expectEqual(t,
		uint64(0b00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000),
		bitmaps[0],
	)
	expectEqual(t, uint(0), idx)
}

// Zig: test "findFreeChunks larger than 64 chunks"
func TestBitmapAllocator_findFreeChunks_larger_than_64_chunks(t *testing.T) {
	bitmaps := []uint64{
		0b11111111_11111111_11111111_11111111_11111111_11111111_11111111_11111111,
		0b11111111_11111111_11111111_11111111_11111111_11111111_11111111_11111111,
	}
	idx := zigUnwrap(t, bmFind(bitmaps, 65))
	expectEqual(t,
		uint64(0b00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000),
		bitmaps[0],
	)
	expectEqual(t,
		uint64(0b11111111_11111111_11111111_11111111_11111111_11111111_11111111_11111110),
		bitmaps[1],
	)
	expectEqual(t, uint(0), idx)
}

// Zig: test "findFreeChunks larger than 64 chunks not at beginning"
func TestBitmapAllocator_findFreeChunks_larger_than_64_chunks_not_at_beginning(t *testing.T) {
	bitmaps := []uint64{
		0b11111111_00000000_00000000_00000000_00000000_00000000_00000000_00000000,
		0b11111111_11111111_11111111_11111111_11111111_11111111_11111111_11111111,
		0b11111111_11111111_11111111_11111111_11111111_11111111_11111111_11111111,
	}
	idx := zigUnwrap(t, bmFind(bitmaps, 65))
	expectEqual(t,
		uint64(0b00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000),
		bitmaps[0],
	)
	expectEqual(t,
		uint64(0b11111110_00000000_00000000_00000000_00000000_00000000_00000000_00000000),
		bitmaps[1],
	)
	expectEqual(t,
		uint64(0b11111111_11111111_11111111_11111111_11111111_11111111_11111111_11111111),
		bitmaps[2],
	)
	expectEqual(t, uint(56), idx)
}

// Zig: test "findFreeChunks larger than 64 chunks exact"
func TestBitmapAllocator_findFreeChunks_larger_than_64_chunks_exact(t *testing.T) {
	bitmaps := []uint64{
		0b11111111_11111111_11111111_11111111_11111111_11111111_11111111_11111111,
		0b11111111_11111111_11111111_11111111_11111111_11111111_11111111_11111111,
	}
	idx := zigUnwrap(t, bmFind(bitmaps, 128))
	expectEqual(t,
		uint64(0b00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000),
		bitmaps[0],
	)
	expectEqual(t,
		uint64(0b00000000_00000000_00000000_00000000_00000000_00000000_00000000_00000000),
		bitmaps[1],
	)
	expectEqual(t, uint(0), idx)
}

// Zig: test "BitmapAllocator layout"
func TestBitmapAllocator_BitmapAllocator_layout(t *testing.T) {
	const chunk_size = 4
	const cap_ = 64 * 4

	layout := BitmapAllocatorLayoutOf(chunk_size, cap_)

	// We expect to use one bitmap since the cap is bytes.
	expectEqual(t, uint(1), layout.bitmap_count)
}

// Zig: test "BitmapAllocator alloc sequentially"
func TestBitmapAllocator_BitmapAllocator_alloc_sequentially(t *testing.T) {
	// Zig-only: `const alloc = testing.allocator;`, the `alignedAlloc` of the
	// backing buffer and `defer alloc.free(buf)` (the allocator owns its
	// memory in Go).
	const chunk_size = 4
	const cap_ = 64

	layout := BitmapAllocatorLayoutOf(chunk_size, cap_)

	bm := BitmapAllocatorInit[uint8](chunk_size, layout)
	ptr := bmTry(bm.Alloc(1)).ok(t)
	ptr[0] = 'A'

	ptr2 := bmTry(bm.Alloc(1)).ok(t)
	expect(t, bmAddr(ptr) != bmAddr(ptr2), "@intFromPtr(ptr.ptr) != @intFromPtr(ptr2.ptr)")

	// Should grab the next chunk
	expectEqual(t, bmAddr(ptr)+4, bmAddr(ptr2))

	// Free ptr and next allocation should be back
	bm.Free(ptr)
	ptr3 := bmTry(bm.Alloc(1)).ok(t)
	expectEqual(t, bmAddr(ptr), bmAddr(ptr3))
}

// Zig: test "BitmapAllocator alloc non-byte"
func TestBitmapAllocator_BitmapAllocator_alloc_non_byte(t *testing.T) {
	// Zig-only: testing.allocator and the backing buffer, as above.
	const chunk_size = 4
	const cap_ = 128

	layout := BitmapAllocatorLayoutOf(chunk_size, cap_)

	// u21 is uint32.
	bm := BitmapAllocatorInit[uint32](chunk_size, layout)
	ptr := bmTry(bm.Alloc(1)).ok(t)
	ptr[0] = 'A'

	ptr2 := bmTry(bm.Alloc(1)).ok(t)
	expect(t, bmAddr(ptr) != bmAddr(ptr2), "@intFromPtr(ptr.ptr) != @intFromPtr(ptr2.ptr)")
	expectEqual(t, bmAddr(ptr)+4, bmAddr(ptr2))

	// Free ptr and next allocation should be back
	bm.Free(ptr)
	ptr3 := bmTry(bm.Alloc(1)).ok(t)
	expectEqual(t, bmAddr(ptr), bmAddr(ptr3))
}

// Zig: test "BitmapAllocator alloc non-byte multi-chunk"
func TestBitmapAllocator_BitmapAllocator_alloc_non_byte_multi_chunk(t *testing.T) {
	// Zig-only: testing.allocator and the backing buffer, as above.
	const chunk_size = 4 * 4 // 4 * @sizeOf(u21)
	const cap_ = 128

	layout := BitmapAllocatorLayoutOf(chunk_size, cap_)

	bm := BitmapAllocatorInit[uint32](chunk_size, layout)
	ptr := bmTry(bm.Alloc(6)).ok(t)
	expectEqual(t, 6, len(ptr))
	bmFill(ptr, 'A')

	ptr2 := bmTry(bm.Alloc(1)).ok(t)
	expect(t, bmAddr(ptr) != bmAddr(ptr2), "@intFromPtr(ptr.ptr) != @intFromPtr(ptr2.ptr)")
	expectEqual(t, bmAddr(ptr)+(4*4*2), bmAddr(ptr2))

	// Free ptr and next allocation should be back
	bm.Free(ptr)
	ptr3 := bmTry(bm.Alloc(1)).ok(t)
	expectEqual(t, bmAddr(ptr), bmAddr(ptr3))
}

// Zig: test "BitmapAllocator alloc large"
func TestBitmapAllocator_BitmapAllocator_alloc_large(t *testing.T) {
	// Zig-only: testing.allocator and the backing buffer, as above.
	const chunk_size = 2
	const cap_ = 256

	layout := BitmapAllocatorLayoutOf(chunk_size, cap_)

	bm := BitmapAllocatorInit[uint8](chunk_size, layout)
	ptr := bmTry(bm.Alloc(129)).ok(t)
	ptr[0] = 'A'
	bm.Free(ptr)
}

// Zig: test "BitmapAllocator alloc and free one bitmap"
func TestBitmapAllocator_BitmapAllocator_alloc_and_free_one_bitmap(t *testing.T) {
	// Zig-only: testing.allocator and the backing buffer, as above.
	const chunk_size = 1
	// Capacity such that we'll have 3 bitmaps.
	const cap_ = BitmapAllocatorBitmapBitSize * 3

	layout := BitmapAllocatorLayoutOf(chunk_size, cap_)

	bm := BitmapAllocatorInit[uint8](chunk_size, layout)

	// Allocate exactly one bitmap worth of bytes.
	slice := bmTry(bm.Alloc(BitmapAllocatorBitmapBitSize)).ok(t)
	expectEqual(t, BitmapAllocatorBitmapBitSize, len(slice))

	bmFill(slice, 0x11)
	expectEqualSlices(t,
		bmRepeat[uint8](0x11, BitmapAllocatorBitmapBitSize),
		slice,
	)

	// Free it
	expect(t, bm.isAllocated(slice), "bm.isAllocated(buf, slice)")
	bm.Free(slice)
	expect(t, !bm.isAllocated(slice), "!bm.isAllocated(buf, slice)")

	// All of our bitmaps should be free.
	expectEqualSlices(t,
		bmRepeat(^uint64(0), 3),
		bm.bitmap[0:3],
	)
}

// Zig: test "BitmapAllocator alloc and free half bitmap"
func TestBitmapAllocator_BitmapAllocator_alloc_and_free_half_bitmap(t *testing.T) {
	// Zig-only: testing.allocator and the backing buffer, as above.
	const chunk_size = 1
	// Capacity such that we'll have 3 bitmaps.
	const cap_ = BitmapAllocatorBitmapBitSize * 3

	layout := BitmapAllocatorLayoutOf(chunk_size, cap_)

	bm := BitmapAllocatorInit[uint8](chunk_size, layout)

	// Allocate exactly half a bitmap worth of bytes.
	slice := bmTry(bm.Alloc(BitmapAllocatorBitmapBitSize / 2)).ok(t)
	expectEqual(t, BitmapAllocatorBitmapBitSize/2, len(slice))

	bmFill(slice, 0x11)
	expectEqualSlices(t,
		bmRepeat[uint8](0x11, BitmapAllocatorBitmapBitSize/2),
		slice,
	)

	// Free it
	expect(t, bm.isAllocated(slice), "bm.isAllocated(buf, slice)")
	bm.Free(slice)
	expect(t, !bm.isAllocated(slice), "!bm.isAllocated(buf, slice)")

	// All of our bitmaps should be free.
	expectEqualSlices(t,
		bmRepeat(^uint64(0), 3),
		bm.bitmap[0:3],
	)
}

// Zig: test "BitmapAllocator alloc and free two half bitmaps"
func TestBitmapAllocator_BitmapAllocator_alloc_and_free_two_half_bitmaps(t *testing.T) {
	// Zig-only: testing.allocator and the backing buffer, as above.
	const chunk_size = 1
	// Capacity such that we'll have 3 bitmaps.
	const cap_ = BitmapAllocatorBitmapBitSize * 3

	layout := BitmapAllocatorLayoutOf(chunk_size, cap_)

	bm := BitmapAllocatorInit[uint8](chunk_size, layout)

	// Allocate exactly one bitmap worth of bytes across two allocations.
	slice := bmTry(bm.Alloc(BitmapAllocatorBitmapBitSize / 2)).ok(t)
	expectEqual(t, BitmapAllocatorBitmapBitSize/2, len(slice))

	bmFill(slice, 0x11)
	expectEqualSlices(t,
		bmRepeat[uint8](0x11, BitmapAllocatorBitmapBitSize/2),
		slice,
	)

	slice2 := bmTry(bm.Alloc(BitmapAllocatorBitmapBitSize / 2)).ok(t)
	expectEqual(t, BitmapAllocatorBitmapBitSize/2, len(slice2))

	bmFill(slice2, 0x22)
	expectEqualSlices(t,
		bmRepeat[uint8](0x22, BitmapAllocatorBitmapBitSize/2),
		slice2,
	)
	expectEqualSlices(t,
		bmRepeat[uint8](0x11, BitmapAllocatorBitmapBitSize/2),
		slice,
	)

	// Free them
	expect(t, bm.isAllocated(slice2), "bm.isAllocated(buf, slice2)")
	bm.Free(slice2)
	expect(t, !bm.isAllocated(slice2), "!bm.isAllocated(buf, slice2)")
	expect(t, bm.isAllocated(slice), "bm.isAllocated(buf, slice)")
	bm.Free(slice)
	expect(t, !bm.isAllocated(slice), "!bm.isAllocated(buf, slice)")

	// All of our bitmaps should be free.
	expectEqualSlices(t,
		bmRepeat(^uint64(0), 3),
		bm.bitmap[0:3],
	)
}

// Zig: test "BitmapAllocator alloc and free 1.5 bitmaps"
func TestBitmapAllocator_BitmapAllocator_alloc_and_free_1_5_bitmaps(t *testing.T) {
	// Zig-only: testing.allocator and the backing buffer, as above.
	const chunk_size = 1
	// Capacity such that we'll have 3 bitmaps.
	const cap_ = BitmapAllocatorBitmapBitSize * 3

	layout := BitmapAllocatorLayoutOf(chunk_size, cap_)

	bm := BitmapAllocatorInit[uint8](chunk_size, layout)

	// Allocate exactly 1.5 bitmaps worth of bytes.
	slice := bmTry(bm.Alloc(3 * BitmapAllocatorBitmapBitSize / 2)).ok(t)
	expectEqual(t, 3*BitmapAllocatorBitmapBitSize/2, len(slice))

	bmFill(slice, 0x11)
	expectEqualSlices(t,
		bmRepeat[uint8](0x11, 3*BitmapAllocatorBitmapBitSize/2),
		slice,
	)

	// Free them
	expect(t, bm.isAllocated(slice), "bm.isAllocated(buf, slice)")
	bm.Free(slice)
	expect(t, !bm.isAllocated(slice), "!bm.isAllocated(buf, slice)")

	// All of our bitmaps should be free.
	expectEqualSlices(t,
		bmRepeat(^uint64(0), 3),
		bm.bitmap[0:3],
	)
}

// Zig: test "BitmapAllocator alloc and free two 1.5 bitmaps"
func TestBitmapAllocator_BitmapAllocator_alloc_and_free_two_1_5_bitmaps(t *testing.T) {
	// Zig-only: testing.allocator and the backing buffer, as above.
	const chunk_size = 1
	// Capacity such that we'll have 3 bitmaps.
	const cap_ = BitmapAllocatorBitmapBitSize * 3

	layout := BitmapAllocatorLayoutOf(chunk_size, cap_)

	bm := BitmapAllocatorInit[uint8](chunk_size, layout)

	// Allocate exactly 3 bitmaps worth of bytes across two allocations.
	slice := bmTry(bm.Alloc(3 * BitmapAllocatorBitmapBitSize / 2)).ok(t)
	expectEqual(t, 3*BitmapAllocatorBitmapBitSize/2, len(slice))

	bmFill(slice, 0x11)
	expectEqualSlices(t,
		bmRepeat[uint8](0x11, 3*BitmapAllocatorBitmapBitSize/2),
		slice,
	)

	slice2 := bmTry(bm.Alloc(3 * BitmapAllocatorBitmapBitSize / 2)).ok(t)
	expectEqual(t, 3*BitmapAllocatorBitmapBitSize/2, len(slice2))

	bmFill(slice2, 0x22)
	expectEqualSlices(t,
		bmRepeat[uint8](0x22, 3*BitmapAllocatorBitmapBitSize/2),
		slice2,
	)
	expectEqualSlices(t,
		bmRepeat[uint8](0x11, 3*BitmapAllocatorBitmapBitSize/2),
		slice,
	)

	// Free them
	expect(t, bm.isAllocated(slice2), "bm.isAllocated(buf, slice2)")
	bm.Free(slice2)
	expect(t, !bm.isAllocated(slice2), "!bm.isAllocated(buf, slice2)")
	expect(t, bm.isAllocated(slice), "bm.isAllocated(buf, slice)")
	bm.Free(slice)
	expect(t, !bm.isAllocated(slice), "!bm.isAllocated(buf, slice)")

	// All of our bitmaps should be free.
	expectEqualSlices(t,
		bmRepeat(^uint64(0), 3),
		bm.bitmap[0:3],
	)
}

// Zig: test "BitmapAllocator alloc and free 1.5 bitmaps offset by 0.75"
func TestBitmapAllocator_BitmapAllocator_alloc_and_free_1_5_bitmaps_offset_by_0_75(t *testing.T) {
	// Zig-only: testing.allocator and the backing buffer, as above.
	const chunk_size = 1
	// Capacity such that we'll have 3 bitmaps.
	const cap_ = BitmapAllocatorBitmapBitSize * 3

	layout := BitmapAllocatorLayoutOf(chunk_size, cap_)

	bm := BitmapAllocatorInit[uint8](chunk_size, layout)

	// Allocate three quarters of a bitmap first.
	slice := bmTry(bm.Alloc(3 * BitmapAllocatorBitmapBitSize / 4)).ok(t)
	expectEqual(t, 3*BitmapAllocatorBitmapBitSize/4, len(slice))

	bmFill(slice, 0x11)
	expectEqualSlices(t,
		bmRepeat[uint8](0x11, 3*BitmapAllocatorBitmapBitSize/4),
		slice,
	)

	// Then a 1.5 bitmap sized allocation, so that it spans
	// from 0.75 to 2.25, occupying bits in 3 different bitmaps.
	slice2 := bmTry(bm.Alloc(3 * BitmapAllocatorBitmapBitSize / 2)).ok(t)
	expectEqual(t, 3*BitmapAllocatorBitmapBitSize/2, len(slice2))

	bmFill(slice2, 0x22)
	expectEqualSlices(t,
		bmRepeat[uint8](0x22, 3*BitmapAllocatorBitmapBitSize/2),
		slice2,
	)
	expectEqualSlices(t,
		bmRepeat[uint8](0x11, 3*BitmapAllocatorBitmapBitSize/4),
		slice,
	)

	// Free them
	expect(t, bm.isAllocated(slice2), "bm.isAllocated(buf, slice2)")
	bm.Free(slice2)
	expect(t, !bm.isAllocated(slice2), "!bm.isAllocated(buf, slice2)")
	expect(t, bm.isAllocated(slice), "bm.isAllocated(buf, slice)")
	bm.Free(slice)
	expect(t, !bm.isAllocated(slice), "!bm.isAllocated(buf, slice)")

	// All of our bitmaps should be free.
	expectEqualSlices(t,
		bmRepeat(^uint64(0), 3),
		bm.bitmap[0:3],
	)
}

// Zig: test "BitmapAllocator alloc and free three 0.75 bitmaps"
func TestBitmapAllocator_BitmapAllocator_alloc_and_free_three_0_75_bitmaps(t *testing.T) {
	// Zig-only: testing.allocator and the backing buffer, as above.
	const chunk_size = 1
	// Capacity such that we'll have 3 bitmaps.
	const cap_ = BitmapAllocatorBitmapBitSize * 3

	layout := BitmapAllocatorLayoutOf(chunk_size, cap_)

	bm := BitmapAllocatorInit[uint8](chunk_size, layout)

	// Allocate three quarters of a bitmap three times.
	slice := bmTry(bm.Alloc(3 * BitmapAllocatorBitmapBitSize / 4)).ok(t)
	expectEqual(t, 3*BitmapAllocatorBitmapBitSize/4, len(slice))

	bmFill(slice, 0x11)
	expectEqualSlices(t,
		bmRepeat[uint8](0x11, 3*BitmapAllocatorBitmapBitSize/4),
		slice,
	)

	slice2 := bmTry(bm.Alloc(3 * BitmapAllocatorBitmapBitSize / 4)).ok(t)
	expectEqual(t, 3*BitmapAllocatorBitmapBitSize/4, len(slice2))

	bmFill(slice2, 0x22)
	expectEqualSlices(t,
		bmRepeat[uint8](0x22, 3*BitmapAllocatorBitmapBitSize/4),
		slice2,
	)
	expectEqualSlices(t,
		bmRepeat[uint8](0x11, 3*BitmapAllocatorBitmapBitSize/4),
		slice,
	)

	slice3 := bmTry(bm.Alloc(3 * BitmapAllocatorBitmapBitSize / 4)).ok(t)
	expectEqual(t, 3*BitmapAllocatorBitmapBitSize/4, len(slice3))

	bmFill(slice3, 0x33)
	expectEqualSlices(t,
		bmRepeat[uint8](0x33, 3*BitmapAllocatorBitmapBitSize/4),
		slice3,
	)
	expectEqualSlices(t,
		bmRepeat[uint8](0x22, 3*BitmapAllocatorBitmapBitSize/4),
		slice2,
	)
	expectEqualSlices(t,
		bmRepeat[uint8](0x11, 3*BitmapAllocatorBitmapBitSize/4),
		slice,
	)

	// Free them
	expect(t, bm.isAllocated(slice2), "bm.isAllocated(buf, slice2)")
	bm.Free(slice2)
	expect(t, !bm.isAllocated(slice2), "!bm.isAllocated(buf, slice2)")
	expect(t, bm.isAllocated(slice), "bm.isAllocated(buf, slice)")
	bm.Free(slice)
	expect(t, !bm.isAllocated(slice), "!bm.isAllocated(buf, slice)")
	expect(t, bm.isAllocated(slice3), "bm.isAllocated(buf, slice3)")
	bm.Free(slice3)
	expect(t, !bm.isAllocated(slice3), "!bm.isAllocated(buf, slice3)")

	// All of our bitmaps should be free.
	expectEqualSlices(t,
		bmRepeat(^uint64(0), 3),
		bm.bitmap[0:3],
	)
}

// Zig: test "BitmapAllocator alloc and free two 1.5 bitmaps offset 0.75"
func TestBitmapAllocator_BitmapAllocator_alloc_and_free_two_1_5_bitmaps_offset_0_75(t *testing.T) {
	// Zig-only: testing.allocator and the backing buffer, as above.
	const chunk_size = 1
	// Capacity such that we'll have 4 bitmaps.
	const cap_ = BitmapAllocatorBitmapBitSize * 4

	layout := BitmapAllocatorLayoutOf(chunk_size, cap_)

	bm := BitmapAllocatorInit[uint8](chunk_size, layout)

	// First allocate a 0.75 bitmap
	slice := bmTry(bm.Alloc(3 * BitmapAllocatorBitmapBitSize / 4)).ok(t)
	expectEqual(t, 3*BitmapAllocatorBitmapBitSize/4, len(slice))

	bmFill(slice, 0x11)
	expectEqualSlices(t,
		bmRepeat[uint8](0x11, 3*BitmapAllocatorBitmapBitSize/4),
		slice,
	)

	// Then two 1.5 bitmaps
	slice2 := bmTry(bm.Alloc(3 * BitmapAllocatorBitmapBitSize / 2)).ok(t)
	expectEqual(t, 3*BitmapAllocatorBitmapBitSize/2, len(slice2))

	bmFill(slice2, 0x22)
	expectEqualSlices(t,
		bmRepeat[uint8](0x22, 3*BitmapAllocatorBitmapBitSize/2),
		slice2,
	)
	expectEqualSlices(t,
		bmRepeat[uint8](0x11, 3*BitmapAllocatorBitmapBitSize/4),
		slice,
	)

	slice3 := bmTry(bm.Alloc(3 * BitmapAllocatorBitmapBitSize / 2)).ok(t)
	expectEqual(t, 3*BitmapAllocatorBitmapBitSize/2, len(slice3))

	bmFill(slice3, 0x33)
	expectEqualSlices(t,
		bmRepeat[uint8](0x33, 3*BitmapAllocatorBitmapBitSize/2),
		slice3,
	)
	expectEqualSlices(t,
		bmRepeat[uint8](0x22, 3*BitmapAllocatorBitmapBitSize/2),
		slice2,
	)
	expectEqualSlices(t,
		bmRepeat[uint8](0x11, 3*BitmapAllocatorBitmapBitSize/4),
		slice,
	)

	// Free them
	expect(t, bm.isAllocated(slice2), "bm.isAllocated(buf, slice2)")
	bm.Free(slice2)
	expect(t, !bm.isAllocated(slice2), "!bm.isAllocated(buf, slice2)")
	expect(t, bm.isAllocated(slice), "bm.isAllocated(buf, slice)")
	bm.Free(slice)
	expect(t, !bm.isAllocated(slice), "!bm.isAllocated(buf, slice)")
	expect(t, bm.isAllocated(slice3), "bm.isAllocated(buf, slice3)")
	bm.Free(slice3)
	expect(t, !bm.isAllocated(slice3), "!bm.isAllocated(buf, slice3)")

	// All of our bitmaps should be free.
	expectEqualSlices(t,
		bmRepeat(^uint64(0), 4),
		bm.bitmap[0:4],
	)
}

// Zig: test "BitmapAllocator bytesRequired"
func TestBitmapAllocator_BitmapAllocator_bytesRequired(t *testing.T) {
	// Chunk size of 16 bytes (like grapheme_chunk in page.zig)
	{
		const chunk_size = 16

		// Single byte rounds up to chunk size
		expectEqual(t, uint64(16), BitmapAllocatorBytesRequired[uint8](chunk_size, 1))
		expectEqual(t, uint64(16), BitmapAllocatorBytesRequired[uint8](chunk_size, 16))
		expectEqual(t, uint64(32), BitmapAllocatorBytesRequired[uint8](chunk_size, 17))

		// u21 (4 bytes each)
		expectEqual(t, uint64(16), BitmapAllocatorBytesRequired[uint32](chunk_size, 1)) // 4 bytes -> 16
		expectEqual(t, uint64(16), BitmapAllocatorBytesRequired[uint32](chunk_size, 4)) // 16 bytes -> 16
		expectEqual(t, uint64(32), BitmapAllocatorBytesRequired[uint32](chunk_size, 5)) // 20 bytes -> 32
		expectEqual(t, uint64(32), BitmapAllocatorBytesRequired[uint32](chunk_size, 6)) // 24 bytes -> 32
	}

	// Chunk size of 4 bytes
	{
		const chunk_size = 4

		expectEqual(t, uint64(4), BitmapAllocatorBytesRequired[uint8](chunk_size, 1))
		expectEqual(t, uint64(4), BitmapAllocatorBytesRequired[uint8](chunk_size, 4))
		expectEqual(t, uint64(8), BitmapAllocatorBytesRequired[uint8](chunk_size, 5))

		// u32 (4 bytes each) - exactly one chunk per element
		expectEqual(t, uint64(4), BitmapAllocatorBytesRequired[uint32](chunk_size, 1))
		expectEqual(t, uint64(8), BitmapAllocatorBytesRequired[uint32](chunk_size, 2))
	}

	// Chunk size of 32 bytes (like string_chunk in page.zig)
	{
		const chunk_size = 32

		expectEqual(t, uint64(32), BitmapAllocatorBytesRequired[uint8](chunk_size, 1))
		expectEqual(t, uint64(32), BitmapAllocatorBytesRequired[uint8](chunk_size, 32))
		expectEqual(t, uint64(64), BitmapAllocatorBytesRequired[uint8](chunk_size, 33))
	}
}

// Zig: test "term51: findFreeChunks: single path n=1 takes the lowest free bit"
func TestBitmapAllocator_term51_findFreeChunks_single_path_n_1_takes_the_lowest_free_bit(t *testing.T) {
	all := []uint64{bmMax}
	expectEqual(t, bmSome(0), bmFind(all, 1))
	expectEqual(t, bmMax-1, all[0])
	expectEqual(t, bmSome(1), bmFind(all, 1))
	expectEqual(t, bmMax-3, all[0])

	// Only the top bit is free.
	top := []uint64{uint64(1) << 63}
	expectEqual(t, bmSome(63), bmFind(top, 1))
	expectEqual(t, uint64(0), top[0])
	// Nothing free left.
	expectEqual(t, bmNone(), bmFind(top, 1))
}

// Zig: test "term51: findFreeChunks: single path every length at every position"
func TestBitmapAllocator_term51_findFreeChunks_single_path_every_length_at_every_position(t *testing.T) {
	// A bitmap with exactly one free run [p, p+n) is found at p and
	// fully consumed.
	for n := uint(1); n < 65; n++ {
		run := bmMax >> (64 - n)
		for p := uint(0); p < 64-n+1; p++ {
			bitmaps := []uint64{run << p}
			expectEqual(t, bmSome(p), bmFind(bitmaps, n))
			expectEqual(t, uint64(0), bitmaps[0])
		}
	}

	// On a fully free bitmap the first n chunks are taken.
	for n := uint(1); n < 65; n++ {
		bitmaps := []uint64{bmMax}
		expectEqual(t, bmSome(0), bmFind(bitmaps, n))
		var expected uint64
		if n == 64 {
			expected = 0
		} else {
			expected = bmMax << n
		}
		expectEqual(t, expected, bitmaps[0])
	}
}

// Zig: test "term51: findFreeChunks: single path one past the longest run"
func TestBitmapAllocator_term51_findFreeChunks_single_path_one_past_the_longest_run(t *testing.T) {
	// A single free run of r chunks at every position never satisfies r+1.
	for r := uint(1); r < 64; r++ {
		run := bmMax >> (64 - r)
		for p := uint(0); p < 64-r+1; p++ {
			bitmaps := []uint64{run << p}
			before := bitmaps[0]
			expectEqual(t, bmNone(), bmFind(bitmaps, r+1))
			expectEqual(t, before, bitmaps[0])
		}
	}
}

// Zig: test "term51: findFreeChunks: single path first fit picks the lowest run"
func TestBitmapAllocator_term51_findFreeChunks_single_path_first_fit_picks_the_lowest_run(t *testing.T) {
	// Runs of 3 at bits 5..8 and 20..23 and 61..64, and a run of 4 at 30..34.
	b := (uint64(0b111) << 5) | (uint64(0b111) << 20) |
		(uint64(0b1111) << 30) | (uint64(0b111) << 61)
	{
		bitmaps := []uint64{b}
		expectEqual(t, bmSome(5), bmFind(bitmaps, 3))
		expectEqual(t, b^(uint64(0b111)<<5), bitmaps[0])
	}
	{
		// 4 only fits in the run at 30; the lowest start inside it.
		bitmaps := []uint64{b}
		expectEqual(t, bmSome(30), bmFind(bitmaps, 4))
		expectEqual(t, b^(uint64(0b1111)<<30), bitmaps[0])
	}
	{
		// Runs at the very top and the very bottom.
		bitmaps := []uint64{(uint64(0b1111) << 60) | 0b11}
		expectEqual(t, bmSome(60), bmFind(bitmaps, 4))
		expectEqual(t, uint64(0b11), bitmaps[0])
		expectEqual(t, bmSome(0), bmFind(bitmaps, 2))
		expectEqual(t, uint64(0), bitmaps[0])
	}
	{
		// Longer free run than needed: take its low end, leave the rest.
		bitmaps := []uint64{uint64(0b1111_1110)}
		expectEqual(t, bmSome(1), bmFind(bitmaps, 3))
		expectEqual(t, uint64(0b1111_0000), bitmaps[0])
		expectEqual(t, bmSome(4), bmFind(bitmaps, 4))
		expectEqual(t, uint64(0), bitmaps[0])
	}
}

// Zig: test "term51: findFreeChunks: single path runs never span two bitmaps"
func TestBitmapAllocator_term51_findFreeChunks_single_path_runs_never_span_two_bitmaps(t *testing.T) {
	// 32 free chunks at the top of bitmap 0 and 32 at the bottom of bitmap
	// 1: 64 contiguous chunks overall, but not within one bitmap.
	bitmaps := []uint64{
		uint64(0xFFFF_FFFF) << 32,
		uint64(0xFFFF_FFFF),
	}
	expectEqual(t, bmNone(), bmFind(bitmaps, 33))
	expectEqual(t, bmNone(), bmFind(bitmaps, 40))
	expectEqual(t, bmNone(), bmFind(bitmaps, 64))
	expectEqual(t, uint64(0xFFFF_FFFF)<<32, bitmaps[0])
	expectEqual(t, uint64(0xFFFF_FFFF), bitmaps[1])

	// 32 fits, in bitmap 0 first.
	expectEqual(t, bmSome(32), bmFind(bitmaps, 32))
	expectEqual(t, uint64(0), bitmaps[0])
	expectEqual(t, uint64(0xFFFF_FFFF), bitmaps[1])
	expectEqual(t, bmSome(64), bmFind(bitmaps, 32))
	expectEqual(t, uint64(0), bitmaps[1])
}

// Zig: test "term51: findFreeChunks: single path wraps to later bitmaps"
func TestBitmapAllocator_term51_findFreeChunks_single_path_wraps_to_later_bitmaps(t *testing.T) {
	// Bitmap 0 is full, bitmap 1 is too fragmented, bitmap 2 has room.
	bitmaps := []uint64{
		0,
		0b0101_0101_0101_0101_0101_0101_0101_0101_0101_0101_0101_0101_0101_0101_0101_0101,
		0b1111_0000,
		bmMax,
	}
	expectEqual(t, bmSome(128+4), bmFind(bitmaps, 4))
	expectEqual(t, uint64(0), bitmaps[2])
	// A single chunk still prefers the lowest free bit (in bitmap 1).
	expectEqual(t, bmSome(64), bmFind(bitmaps, 1))
	expectEqual(t, uint64(0x5555_5555_5555_5554), bitmaps[1])
	// Two chunks do not fit in bitmap 1, so they go to bitmap 3.
	expectEqual(t, bmSome(192), bmFind(bitmaps, 2))
	expectEqual(t, bmMax<<2, bitmaps[3])
	expectEqual(t, uint64(0), bitmaps[0])
}

// Zig: test "term51: findFreeChunks: single path n=64 needs a fully free bitmap"
func TestBitmapAllocator_term51_findFreeChunks_single_path_n_64_needs_a_fully_free_bitmap(t *testing.T) {
	bitmaps := []uint64{
		bmMax - 1,  // bit 0 used
		bmMax >> 1, // bit 63 used
		bmMax,
	}
	expectEqual(t, bmSome(128), bmFind(bitmaps, 64))
	expectEqual(t, uint64(0), bitmaps[2])
	expectEqual(t, bmMax-1, bitmaps[0])
	expectEqual(t, bmNone(), bmFind(bitmaps, 64))
	// 63 still fits in either of the other two.
	expectEqual(t, bmSome(1), bmFind(bitmaps, 63))
	expectEqual(t, uint64(0), bitmaps[0])
	expectEqual(t, bmSome(64), bmFind(bitmaps, 63))
	expectEqual(t, uint64(0), bitmaps[1])
}

// Zig: test "term51: findFreeChunks: no bitmaps and all used bitmaps"
func TestBitmapAllocator_term51_findFreeChunks_no_bitmaps_and_all_used_bitmaps(t *testing.T) {
	empty := []uint64{}
	expectEqual(t, bmNone(), bmFind(empty, 1))
	expectEqual(t, bmNone(), bmFind(empty, 64))
	// The large path with no bitmaps at all never enters its loop.
	expectEqual(t, bmNone(), bmFind(empty, 65))
	expectEqual(t, bmNone(), bmFind(empty, 1000))

	used := []uint64{0, 0, 0}
	for _, n := range []uint{1, 2, 63, 64, 65, 100, 128, 129, 1000} {
		expectEqual(t, bmNone(), bmFind(used, n))
		expectEqualSlices(t, []uint64{0, 0, 0}, used)
	}
}

// Zig: test "term51: findFreeChunks: large path skips bitmaps whose top bit is used"
func TestBitmapAllocator_term51_findFreeChunks_large_path_skips_bitmaps_whose_top_bit_is_used(t *testing.T) {
	// Bitmap 0 has no free chunk at its top, so it cannot start a span.
	bitmaps := []uint64{
		bmMax >> 1,
		bmMax,
		bmMax,
	}
	expectEqual(t, bmSome(64), bmFind(bitmaps, 65))
	expectEqual(t, bmMax>>1, bitmaps[0])
	expectEqual(t, uint64(0), bitmaps[1])
	expectEqual(t, bmMax-1, bitmaps[2])
}

// Zig: test "term51: findFreeChunks: large path remaining exactly 64 and exact end"
func TestBitmapAllocator_term51_findFreeChunks_large_path_remaining_exactly_64_and_exact_end(t *testing.T) {
	// One free chunk at the top of bitmap 0, then a full free bitmap: the
	// remainder after the prefix is exactly 64 and the span ends exactly at
	// the last chunk of the last bitmap.
	bitmaps := []uint64{uint64(1) << 63, bmMax}
	expectEqual(t, bmSome(63), bmFind(bitmaps, 65))
	expectEqual(t, uint64(0), bitmaps[0])
	expectEqual(t, uint64(0), bitmaps[1])

	// Same shape followed by an untouched third bitmap.
	three := []uint64{uint64(1) << 63, bmMax, bmMax}
	expectEqual(t, bmSome(63), bmFind(three, 65))
	expectEqual(t, uint64(0), three[0])
	expectEqual(t, uint64(0), three[1])
	expectEqual(t, bmMax, three[2])

	// 129 chunks from bit 63: remainder 128 -> one full bitmap in the loop
	// then a remainder of exactly 64.
	four := []uint64{
		uint64(1) << 63,
		bmMax,
		bmMax,
		bmMax,
	}
	expectEqual(t, bmSome(63), bmFind(four, 129))
	expectEqual(t, uint64(0), four[0])
	expectEqual(t, uint64(0), four[1])
	expectEqual(t, uint64(0), four[2])
	expectEqual(t, bmMax, four[3])
}

// Zig: test "term51: findFreeChunks: large path tail too short retries at the next bitmap"
func TestBitmapAllocator_term51_findFreeChunks_large_path_tail_too_short_retries_at_the_next_bitmap(t *testing.T) {
	// Prefix of 8 in bitmap 0, but bitmap 1 only has 56 free chunks at its
	// bottom (57 needed), and its top bit is used so it cannot start a span.
	// The span then starts at bitmap 2.
	b0 := uint64(0xFF) << 56
	b1 := bmMax >> 8
	bitmaps := []uint64{b0, b1, bmMax, bmMax}
	expectEqual(t, bmSome(128), bmFind(bitmaps, 65))
	expectEqual(t, b0, bitmaps[0])
	expectEqual(t, b1, bitmaps[1])
	expectEqual(t, uint64(0), bitmaps[2])
	expectEqual(t, bmMax-1, bitmaps[3])
}

// Zig: test "term51: findFreeChunks: large path tail too short retries with a prefix"
func TestBitmapAllocator_term51_findFreeChunks_large_path_tail_too_short_retries_with_a_prefix(t *testing.T) {
	// Bitmap 1 has bit 56 used: only 56 free chunks at the bottom, which is
	// too short for the 57 needed after bitmap 0's prefix of 8. But bitmap 1
	// has its own prefix of 7 (bits 57..63), so the span restarts there.
	b0 := uint64(0xFF) << 56
	b1 := ^(uint64(1) << 56)
	bitmaps := []uint64{b0, b1, bmMax}
	expectEqual(t, bmSome(64+57), bmFind(bitmaps, 65))
	expectEqual(t, b0, bitmaps[0])
	// Bits 57..63 are now used; bits 0..55 stay free.
	expectEqual(t, (uint64(1)<<56)-1, bitmaps[1])
	// The remaining 58 chunks come out of the bottom of bitmap 2.
	expectEqual(t, bmMax<<58, bitmaps[2])
}

// Zig: test "term51: findFreeChunks: large path spans several full bitmaps"
func TestBitmapAllocator_term51_findFreeChunks_large_path_spans_several_full_bitmaps(t *testing.T) {
	bitmaps := bmFull(5)
	expectEqual(t, bmSome(0), bmFind(bitmaps, 200))
	expectEqual(t, uint64(0), bitmaps[0])
	expectEqual(t, uint64(0), bitmaps[1])
	expectEqual(t, uint64(0), bitmaps[2])
	expectEqual(t, bmMax<<8, bitmaps[3])
	expectEqual(t, bmMax, bitmaps[4])
}

// Zig: test "term51: findFreeChunks: large path restarts at a partially used middle bitmap"
func TestBitmapAllocator_term51_findFreeChunks_large_path_restarts_at_a_partially_used_middle_bitmap(t *testing.T) {
	// Bitmap 2 has bit 0 used. A span starting at bitmap 0 needs two full
	// bitmaps after the first but hits bitmap 2 (not fully free) while more
	// than 64 chunks remain, so the search restarts at bitmap 2 itself, whose
	// prefix is 63 (bits 1..63).
	bitmaps := []uint64{
		bmMax,
		bmMax,
		bmMax - 1,
		bmMax,
		bmMax,
		bmMax,
	}
	expectEqual(t, bmSome(2*64+1), bmFind(bitmaps, 200))
	expectEqual(t, bmMax, bitmaps[0])
	expectEqual(t, bmMax, bitmaps[1])
	expectEqual(t, uint64(0), bitmaps[2])
	expectEqual(t, uint64(0), bitmaps[3])
	expectEqual(t, uint64(0), bitmaps[4])
	// 137 - 128 = 9 chunks from the bottom of the last bitmap.
	expectEqual(t, bmMax<<9, bitmaps[5])
}

// Zig: test "term51: findFreeChunks: large path runs out of bitmaps in the middle loop"
func TestBitmapAllocator_term51_findFreeChunks_large_path_runs_out_of_bitmaps_in_the_middle_loop(t *testing.T) {
	// More than 64 chunks are still needed when the bitmaps run out: no
	// span, and nothing is marked.
	two := []uint64{bmMax, bmMax}
	expectEqual(t, bmNone(), bmFind(two, 200))
	expectEqualSlices(t, []uint64{bmMax, bmMax}, two)

	three := bmFull(3)
	expectEqual(t, bmNone(), bmFind(three, 300))
	expectEqual(t, bmNone(), bmFind(three, 193+64))
	expectEqualSlices(t, bmFull(3), three)

	// Also from a partial prefix.
	partial := []uint64{uint64(1) << 63, bmMax}
	expectEqual(t, bmNone(), bmFind(partial, 65+65))
	expectEqualSlices(t, []uint64{uint64(1) << 63, bmMax}, partial)
}

// Zig: test "term51: findFreeChunks: large path falls off the end without a span"
func TestBitmapAllocator_term51_findFreeChunks_large_path_falls_off_the_end_without_a_span(t *testing.T) {
	// Tail bitmap is fully used: the span fails, the scan continues past the
	// last bitmap and gives up.
	a := []uint64{bmMax, bmMax, 0}
	expectEqual(t, bmNone(), bmFind(a, 130))
	expectEqualSlices(t, []uint64{bmMax, bmMax, 0}, a)

	// Free chunks everywhere but never 65 in a row.
	b := []uint64{bmMax >> 1, bmMax >> 1, bmMax >> 1}
	expectEqual(t, bmNone(), bmFind(b, 65))
	expectEqual(t, bmNone(), bmFind(b, 70))
	expectEqualSlices(t, []uint64{bmMax >> 1, bmMax >> 1, bmMax >> 1}, b)

	// Everything used.
	c := []uint64{0, 0}
	expectEqual(t, bmNone(), bmFind(c, 65))
}

// Zig: test "term51: findFreeChunks: large path every start offset and length"
func TestBitmapAllocator_term51_findFreeChunks_large_path_every_start_offset_and_length(t *testing.T) {
	// Bitmap 0 has p free chunks at its top (p in 1..64) and four more
	// bitmaps are fully free. The span starts at chunk 64-p and covers n.
	lens := []uint{65, 66, 100, 127, 128, 129, 130, 192, 193, 255, 256}
	for p := uint(1); p < 65; p++ {
		for _, n := range lens {
			start := 64 - p
			if start+n > 5*64 {
				continue
			}
			bitmaps := bmFull(5)
			bitmaps[0] = bmMax << (64 - p)
			expectEqual(t, bmSome(start), bmFind(bitmaps, n))
			for j := uint(0); j < 5*64; j++ {
				was_free := j >= start
				used_now := j >= start && j < start+n
				expectEqual(t, was_free && !used_now, bmBitFree(bitmaps, j))
			}
		}
	}

	// All free, n from 65 to the whole of four bitmaps: always chunk 0.
	for n := uint(65); n < 4*64+1; n++ {
		bitmaps := bmFull(4)
		expectEqual(t, bmSome(0), bmFind(bitmaps, n))
		for j := uint(0); j < 4*64; j++ {
			expectEqual(t, j >= n, bmBitFree(bitmaps, j))
		}
	}
}

// bmRandomCase draws one input of the random test the way Zig's loop body
// does: the bitmaps first (length, then for each bitmap a shift and a
// pattern), then the request size. The draws, and so the inputs, are those of
// Zig's `Xoshiro256.init(0x7e57_0b17)`.
func bmRandomCase(r *bmXoshiro) ([]uint64, uint) {
	length := uint(r.intRangeAtMostUsize(1, 6))
	bitmaps := make([]uint64, length)
	for i := range bitmaps {
		sh := uint(r.uintLessThanU8(64))
		switch r.uintLessThanU8(8) {
		case 0:
			bitmaps[i] = bmMax
		case 1:
			bitmaps[i] = 0
		case 2:
			bitmaps[i] = r.intU64()
		case 3:
			a := r.intU64()
			b := r.intU64()
			c := r.intU64()
			bitmaps[i] = a | b | c
		case 4:
			a := r.intU64()
			b := r.intU64()
			c := r.intU64()
			bitmaps[i] = a & b & c
		case 5:
			bitmaps[i] = bmMax << sh
		case 6:
			bitmaps[i] = bmMax >> sh
		default:
			bitmaps[i] = ^(uint64(1) << sh)
		}
	}

	var n uint
	if r.boolean() {
		n = uint(r.intRangeAtMostUsize(1, 64))
	} else {
		n = uint(r.intRangeAtMostUsize(65, uint64(length)*64+80))
	}
	return bitmaps, n
}

// bmRandomSkipped reports whether the random test skips an input: the pinned defect
// (see term51WDefect): inputs where the search reaches the end of the
// bitmaps with a free run that is too short but less than 64 chunks short.
func bmRandomSkipped(bitmaps []uint64, n uint, expect Optional[uint]) bool {
	if n > 64 && !expect.Valid {
		t := bmTrailingFree(bitmaps)
		if t > 0 && n-t <= 64 {
			return true
		}
	}
	return false
}

// Zig: test "term51: findFreeChunks: random bitmaps agree with a first-fit model"
func TestBitmapAllocator_term51_findFreeChunks_random_bitmaps_agree_with_a_first_fit_model(t *testing.T) {
	prng := bmXoshiroInit(0x7e57_0b17)
	r := &prng

	var checked_large uint
	var skipped uint
	for iter := 0; iter < 20000; iter++ {
		bitmaps, n := bmRandomCase(r)
		length := uint(len(bitmaps))

		expectModel := bmModelFind(bitmaps, n)

		// The pinned out-of-bounds defect (see term51WDefect): skip inputs
		// where the search reaches the end of the bitmaps with a free run
		// that is too short but less than 64 chunks short. (Upstream aborts
		// on them; the Go port does not, see
		// TestBitmapAllocator_findFreeChunks_random_bitmaps_skipped_inputs_no_space.)
		if bmRandomSkipped(bitmaps, n, expectModel) {
			skipped++
			continue
		}

		before := make([]uint64, length)
		copy(before, bitmaps)

		got := bmFind(bitmaps, n)
		expectEqual(t, expectModel, got)
		if n > 64 {
			checked_large++
		}

		for j := uint(0); j < length*64; j++ {
			was_free := bmBitFree(before, j)
			taken := false
			if got.Valid {
				taken = j >= got.Value && j < got.Value+n
			}
			expectEqual(t, was_free && !taken, bmBitFree(bitmaps, j))
		}
	}
	expect(t, checked_large > 1000, "checked_large > 1000")
	expect(t, skipped > 0, "skipped > 0")
}

// Go-only: Zig's test "term51: findFreeChunks: large request that spans past
// the last bitmap" sits inside the unreferenced struct term51WDefect and is
// never compiled: upstream aborts there (an out-of-bounds read of
// bitmaps[i] at bitmap_allocator.zig:281, the loop only bounds-checks while
// more than 64 chunks remain). The Go port does not copy the abort
// (term51/docs/deliberate-fixes.md, the one exception): it returns "no
// space", like the reference patch
// patches-reference/0001-term51-reference-bounds-check-in-findFreeChunks.patch.
func TestBitmapAllocator_findFreeChunks_large_request_past_last_bitmap(t *testing.T) {
	one := []uint64{bmMax}
	expectEqual(t, bmNone(), bmFind(one, 65))

	two := []uint64{uint64(1) << 63, bmMax}
	expectEqual(t, bmNone(), bmFind(two, 129))

	three := bmFull(3)
	expectEqual(t, bmNone(), bmFind(three, 193))

	// Nothing is marked.
	expectEqualSlices(t, []uint64{bmMax}, one)
	expectEqualSlices(t, []uint64{uint64(1) << 63, bmMax}, two)
	expectEqualSlices(t, bmFull(3), three)
}

// Go-only: the inputs the random test above skips because upstream aborts on
// them (bitmap_allocator.zig:281, see the test before) are drawn again, from
// the same PRNG stream, and the Go port returns "no space" for each, leaving
// the bitmaps as they were.
func TestBitmapAllocator_findFreeChunks_random_bitmaps_skipped_inputs_no_space(t *testing.T) {
	prng := bmXoshiroInit(0x7e57_0b17)
	r := &prng

	var checked uint
	for iter := 0; iter < 20000; iter++ {
		bitmaps, n := bmRandomCase(r)
		expectModel := bmModelFind(bitmaps, n)
		if !bmRandomSkipped(bitmaps, n, expectModel) {
			continue
		}

		before := make([]uint64, len(bitmaps))
		copy(before, bitmaps)

		expectEqual(t, bmNone(), bmFind(bitmaps, n))
		expectEqualSlices(t, before, bitmaps)
		checked++
	}
	expect(t, checked > 0, "checked > 0")
}

// Zig: test "term51: BitmapAllocator.init: bitmaps all free and fields set"
func TestBitmapAllocator_term51_BitmapAllocator_init_bitmaps_all_free_and_fields_set(t *testing.T) {
	// The Zig test is an `inline for` over the allocator types
	// term51WGraphemeAlloc, term51WStringAlloc, BitmapAllocator(1) and
	// BitmapAllocator(4): here a loop over their chunk sizes.
	for _, chunk_size := range []uint64{bmGraphemeChunk, bmStringChunk, 1, 4} {
		expectEqual(t, 64, BitmapAllocatorBitmapBitSize)
		// Zig-only: `Alloc.base_align.toByteUnits() == 8` (the alignment of
		// the backing buffer; there is none in Go).

		for _, cap_ := range []uint64{1, 64, 1024, 1025, 4096, 70000} {
			l := BitmapAllocatorLayoutOf(chunk_size, cap_)
			// Zig-only: alignedAlloc of the buffer, `defer free`, and the
			// `@memset(buf, 0xAA)` that checks init writes the bitmaps and
			// nothing else.

			bm := BitmapAllocatorInit[uint8](chunk_size, l)
			expectEqual(t, l.bitmap_count, bm.bitmap_count)
			// Zig-only: `bm.bitmap.offset == l.bitmap_start` and
			// `bm.chunks.offset == l.chunks_start` (Go has no offsets), and
			// the two indexOfNone checks of the 0xFF / 0xAA bytes of buf.
			for _, word := range bm.bitmap[:bm.bitmap_count] {
				expectEqual(t, ^uint64(0), word)
			}
			expectEqual(t, uint64(0), bm.UsedBytes())
		}
	}
}

// Zig: test "term51: BitmapAllocator.init: zero capacity has no bitmaps"
func TestBitmapAllocator_term51_BitmapAllocator_init_zero_capacity_has_no_bitmaps(t *testing.T) {
	const chunk_size = bmGraphemeChunk

	l := BitmapAllocatorLayoutOf(chunk_size, 0)
	// Zig-only: alignedAlloc of the buffer and `defer free`.
	bm := BitmapAllocatorInit[uint8](chunk_size, l)
	expectEqual(t, uint(0), bm.bitmap_count)
	// Zig-only: `bm.bitmap.offset == 0` and `bm.chunks.offset == 0`.
	expectEqual(t, uint64(0), bm.CapacityBytes())
	expectEqual(t, uint64(0), bm.UsedBytes())
	_, err := bm.Alloc(1)
	bmExpectError(t, ErrOutOfMemory, err)
}

// Zig: test "term51: BitmapAllocator.init: overwrites stale bitmap contents"
func TestBitmapAllocator_term51_BitmapAllocator_init_overwrites_stale_bitmap_contents(t *testing.T) {
	const chunk_size = 4

	l := BitmapAllocatorLayoutOf(chunk_size, 512)
	expectEqual(t, uint(2), l.bitmap_count)
	// Zig-only: alignedAlloc of the buffer, `defer free` and
	// `@memset(buf, 0)` (Go's Init makes fresh memory; the stale contents
	// this test starts from do not exist).

	bm := BitmapAllocatorInit[uint8](chunk_size, l)
	expectEqualSlices(t, []uint64{^uint64(0), ^uint64(0)}, bm.bitmap[0:2])
	_ = bmTry(bm.Alloc(100)).ok(t)
	expect(t, bm.UsedBytes() != 0, "bm.usedBytes(buf) != 0")

	// Re-initializing the same memory frees everything again.
	bm = BitmapAllocatorInit[uint8](chunk_size, l)
	expectEqual(t, uint64(0), bm.UsedBytes())
	expectEqualSlices(t, []uint64{^uint64(0), ^uint64(0)}, bm.bitmap[0:2])
}

// Zig: test "term51: BitmapAllocator.init: nonzero OffsetBuf offset is added to the member offsets"
func TestBitmapAllocator_term51_BitmapAllocator_init_nonzero_OffsetBuf_offset_is_added_to_the_member_offsets(t *testing.T) {
	const chunk_size = 16

	l := BitmapAllocatorLayoutOf(chunk_size, 2048)
	expectEqual(t, uint(2), l.bitmap_count)
	expectEqual(t, uint64(16), l.chunks_start)
	// Zig-only: the 24 + total_size byte buffer, its 0xAA fill and
	// `OffsetBuf.initOffset(buf, 24)`: Go has no memory base.

	bm := BitmapAllocatorInit[uint8](chunk_size, l)
	// Zig-only: `bm.bitmap.offset == 24`, `bm.chunks.offset == 24 + 16`, and
	// the three loops that check the bytes before, inside and after the
	// structure (the 24 bytes in front are not ours and are untouched).
	expectEqual(t, uint(2), bm.bitmap_count)

	// Allocations are addressed against the true base.
	s := bmTry(bm.Alloc(1)).ok(t)
	// Zig-only: `@intFromPtr(s.ptr) == @intFromPtr(buf.ptr) + 40`. The Go
	// equivalent: the allocation is the first chunk of `chunks`.
	expectEqual(t, bmAddr(bm.chunks), bmAddr(s))
	s2 := bmTry(bm.Alloc(17)).ok(t)
	// Zig-only: `@intFromPtr(s2.ptr) == @intFromPtr(buf.ptr) + 40 + 16`. The
	// Go equivalent: the second chunk of `chunks`.
	expectEqual(t, bmAddr(bm.chunks)+16, bmAddr(s2))
	bm.Free(s)
	bm.Free(s2)
	expectEqual(t, uint64(0), bm.UsedBytes())
}

// Zig: test "term51: BitmapAllocator.layout: pinned fields for page.zig chunk sizes"
func TestBitmapAllocator_term51_BitmapAllocator_layout_pinned_fields_for_page_zig_chunk_sizes(t *testing.T) {
	type row struct {
		cap          uint64
		bitmap_count uint
		chunks_start uint64
		total_size   uint64
	}

	// Grapheme allocator: 16-byte chunks. Note total_size reserves
	// aligned_cap * chunk_size bytes of chunk memory (16x the capacity).
	g := []row{
		{cap: 0, bitmap_count: 0, chunks_start: 0, total_size: 0},
		{cap: 1, bitmap_count: 1, chunks_start: 8, total_size: 8 + 16*16},
		{cap: 15, bitmap_count: 1, chunks_start: 8, total_size: 264},
		{cap: 16, bitmap_count: 1, chunks_start: 8, total_size: 264},
		{cap: 17, bitmap_count: 1, chunks_start: 8, total_size: 8 + 32*16},
		{cap: 1023, bitmap_count: 1, chunks_start: 8, total_size: 16392},
		{cap: 1024, bitmap_count: 1, chunks_start: 8, total_size: 16392},
		{cap: 1025, bitmap_count: 2, chunks_start: 16, total_size: 16 + 1040*16},
		{cap: 2048, bitmap_count: 2, chunks_start: 16, total_size: 32784},
		{cap: 2049, bitmap_count: 3, chunks_start: 24, total_size: 24 + 2064*16},
		{cap: 70000, bitmap_count: 69, chunks_start: 552, total_size: 1120552},
	}
	for _, row := range g {
		l := BitmapAllocatorLayoutOf(bmGraphemeChunk, row.cap)
		expectEqual(t, uint64(0), l.bitmap_start)
		expectEqual(t, row.bitmap_count, l.bitmap_count)
		expectEqual(t, row.chunks_start, l.chunks_start)
		expectEqual(t, row.total_size, l.total_size)
	}

	// String allocator: 32-byte chunks.
	s := []row{
		{cap: 0, bitmap_count: 0, chunks_start: 0, total_size: 0},
		{cap: 1, bitmap_count: 1, chunks_start: 8, total_size: 8 + 32*32},
		{cap: 32, bitmap_count: 1, chunks_start: 8, total_size: 1032},
		{cap: 33, bitmap_count: 1, chunks_start: 8, total_size: 8 + 64*32},
		{cap: 2047, bitmap_count: 1, chunks_start: 8, total_size: 65544},
		{cap: 2048, bitmap_count: 1, chunks_start: 8, total_size: 65544},
		{cap: 2049, bitmap_count: 2, chunks_start: 16, total_size: 16 + 2080*32},
		{cap: 4096, bitmap_count: 2, chunks_start: 16, total_size: 131088},
		{cap: 70000, bitmap_count: 35, chunks_start: 280, total_size: 2240792},
	}
	for _, row := range s {
		l := BitmapAllocatorLayoutOf(bmStringChunk, row.cap)
		expectEqual(t, uint64(0), l.bitmap_start)
		expectEqual(t, row.bitmap_count, l.bitmap_count)
		expectEqual(t, row.chunks_start, l.chunks_start)
		expectEqual(t, row.total_size, l.total_size)
	}

	// Other chunk sizes used by upstream's tests.
	{
		l := BitmapAllocatorLayoutOf(4, 256)
		expectEqual(t, uint(1), l.bitmap_count)
		expectEqual(t, uint64(8), l.chunks_start)
		expectEqual(t, uint64(8+256*4), l.total_size)
	}
	{
		l := BitmapAllocatorLayoutOf(1, 192)
		expectEqual(t, uint(3), l.bitmap_count)
		expectEqual(t, uint64(24), l.chunks_start)
		expectEqual(t, uint64(24+192), l.total_size)
	}
	{
		l := BitmapAllocatorLayoutOf(2, 256)
		expectEqual(t, uint(2), l.bitmap_count)
		expectEqual(t, uint64(16), l.chunks_start)
		expectEqual(t, uint64(16+256*2), l.total_size)
	}
}

// Zig: test "term51: BitmapAllocator.layout: chunk and bitmap boundaries"
func TestBitmapAllocator_term51_BitmapAllocator_layout_chunk_and_bitmap_boundaries(t *testing.T) {
	// For both page.zig chunk sizes: a bitmap covers 64 chunks, so capacity
	// k*64*chunk needs k bitmaps and one byte more needs k+1. (The Zig test
	// is an `inline for` over the allocator types and the chunk sizes.)
	for _, chunk := range []uint64{bmGraphemeChunk, bmStringChunk} {
		layout := func(cap_ uint64) BitmapAllocatorLayout {
			return BitmapAllocatorLayoutOf(chunk, cap_)
		}
		for k := uint64(1); k < 6; k++ {
			per := 64 * chunk
			expectEqual(t, uint(k), layout(k*per-1).bitmap_count)
			expectEqual(t, uint(k), layout(k*per).bitmap_count)
			expectEqual(t, uint(k+1), layout(k*per+1).bitmap_count)
			expectEqual(t, uint(k+1), layout(k*per+chunk).bitmap_count)
			expectEqual(t, uint(k+1), layout((k+1)*per).bitmap_count)

			// chunks_start is one u64 per bitmap.
			expectEqual(t, 8*k, layout(k*per).chunks_start)
			expectEqual(t, 8*(k+1), layout(k*per+1).chunks_start)
		}

		// Capacity within a chunk rounds up to the chunk and not further.
		expectEqual(t, layout(chunk).total_size, layout(1).total_size)
		expectEqual(t, layout(chunk).total_size, layout(chunk-1).total_size)
		expectEqual(t, layout(2*chunk).total_size, layout(chunk+1).total_size)
		expect(t, layout(chunk+1).total_size != layout(chunk).total_size, "Alloc.layout(chunk + 1).total_size != Alloc.layout(chunk).total_size")

		// total_size steps by chunk*chunk per extra chunk of capacity while
		// the bitmap count stays the same (the documented over-reservation).
		expectEqual(t,
			layout(2*chunk).total_size-layout(chunk).total_size,
			chunk*chunk,
		)
	}
}

// Zig: test "term51: BitmapAllocator.layout: every capacity from 0 to 70000 bytes"
func TestBitmapAllocator_term51_BitmapAllocator_layout_every_capacity_from_0_to_70000_bytes(t *testing.T) {
	// Independent closed forms: bitmaps = ceil(cap / (64 * chunk)),
	// chunks_start = 8 * bitmaps, total = chunks_start + aligned_cap * chunk.
	bmCheckLayoutSweep(t, 16, 70000)
	bmCheckLayoutSweep(t, 32, 70000)
	bmCheckLayoutSweep(t, 1, 70000)
	bmCheckLayoutSweep(t, 2, 70000)
	bmCheckLayoutSweep(t, 4, 70000)
	bmCheckLayoutSweep(t, 8, 70000)
	bmCheckLayoutSweep(t, 64, 70000)
}

// Zig: test "term51: BitmapAllocator.bytesRequired: grapheme and string chunk sizes"
func TestBitmapAllocator_term51_BitmapAllocator_bytesRequired_grapheme_and_string_chunk_sizes(t *testing.T) {
	// Grapheme chunk (16): u21 is 4 bytes (uint32).
	for n := uint(0); n < 300; n++ {
		const g = bmGraphemeChunk
		expectEqual(t, uint64((n+15)/16*16), BitmapAllocatorBytesRequired[uint8](g, n))
		expectEqual(t, uint64((4*n+15)/16*16), BitmapAllocatorBytesRequired[uint32](g, n))
		expectEqual(t, uint64((2*n+15)/16*16), BitmapAllocatorBytesRequired[uint16](g, n))
		expectEqual(t, uint64((3*n+15)/16*16), BitmapAllocatorBytesRequired[[3]uint8](g, n))
		expectEqual(t, uint64((8*n+15)/16*16), BitmapAllocatorBytesRequired[uint64](g, n))

		const s = bmStringChunk
		expectEqual(t, uint64((n+31)/32*32), BitmapAllocatorBytesRequired[uint8](s, n))
		expectEqual(t, uint64((4*n+31)/32*32), BitmapAllocatorBytesRequired[uint32](s, n))
		expectEqual(t, uint64((3*n+31)/32*32), BitmapAllocatorBytesRequired[[3]uint8](s, n))
	}

	// Pinned values.
	expectEqual(t, uint64(0), BitmapAllocatorBytesRequired[uint32](bmGraphemeChunk, 0))
	expectEqual(t, uint64(16), BitmapAllocatorBytesRequired[uint32](bmGraphemeChunk, 1))
	expectEqual(t, uint64(16), BitmapAllocatorBytesRequired[uint32](bmGraphemeChunk, 4))
	expectEqual(t, uint64(32), BitmapAllocatorBytesRequired[uint32](bmGraphemeChunk, 5))
	expectEqual(t, uint64(0), BitmapAllocatorBytesRequired[uint8](bmStringChunk, 0))
	expectEqual(t, uint64(32), BitmapAllocatorBytesRequired[uint8](bmStringChunk, 1))
	expectEqual(t, uint64(64), BitmapAllocatorBytesRequired[uint8](bmStringChunk, 33))

	// Large counts, and a chunk size of 1 never rounds.
	expectEqual(t, uint64(70000), BitmapAllocatorBytesRequired[uint8](bmGraphemeChunk, 70000))
	expectEqual(t, uint64(70016), BitmapAllocatorBytesRequired[uint8](bmStringChunk, 70000))
	expectEqual(t, uint64(1_000_000), BitmapAllocatorBytesRequired[uint8](bmGraphemeChunk, 1_000_000))
	expectEqual(t, uint64(1_000_000*4), BitmapAllocatorBytesRequired[uint32](bmGraphemeChunk, 1_000_000))
	expectEqual(t, uint64(7), BitmapAllocatorBytesRequired[uint8](1, 7))
	expectEqual(t, uint64(21), BitmapAllocatorBytesRequired[[3]uint8](1, 7))
	expectEqual(t, uint64(8), BitmapAllocatorBytesRequired[uint8](8, 7))
	expectEqual(t, uint64(24), BitmapAllocatorBytesRequired[[3]uint8](8, 7))
	expectEqual(t, uint64(24), BitmapAllocatorBytesRequired[[3]uint8](8, 8))
}

// Zig: test "term51: BitmapAllocator.alloc: grapheme chunk sizes round up to 16 bytes"
func TestBitmapAllocator_term51_BitmapAllocator_alloc_grapheme_chunk_sizes_round_up_to_16_bytes(t *testing.T) {
	fx := bmFixtureInit[uint8](t, 16, 2048)
	// Zig-only: `defer fx.deinit()` (frees the buffer).
	expectEqual(t, uint64(16), fx.layout.chunks_start)

	sizes := []uint{1, 16, 17, 32, 33}
	counts := []uint{1, 1, 2, 2, 3}
	var idx uint
	var used uint64
	for i := range sizes {
		nbytes, count := sizes[i], counts[i]
		s := fx.alloc(nbytes)
		expectEqual(t, int(nbytes), len(s))
		expectEqual(t, idx, fx.chunkIndex(s))
		// Zig-only: `fx.baseOffset(s) == 16 + idx * 16` (offset from the
		// start of the buffer, which has no Go counterpart).
		idx += count
		used += uint64(count) * 16
		expectEqual(t, used, fx.usedBytes())
	}
	expectEqual(t, uint(9), idx)
}

// Zig: test "term51: BitmapAllocator.alloc: string chunk sizes round up to 32 bytes"
func TestBitmapAllocator_term51_BitmapAllocator_alloc_string_chunk_sizes_round_up_to_32_bytes(t *testing.T) {
	fx := bmFixtureInit[uint8](t, 32, 4096)
	// Zig-only: `defer fx.deinit()`.
	expectEqual(t, uint64(16), fx.layout.chunks_start)

	sizes := []uint{1, 32, 33, 64, 65, 100, 128, 129}
	counts := []uint{1, 1, 2, 2, 3, 4, 4, 5}
	var idx uint
	for i := range sizes {
		nbytes, count := sizes[i], counts[i]
		s := fx.alloc(nbytes)
		expectEqual(t, int(nbytes), len(s))
		expectEqual(t, idx, fx.chunkIndex(s))
		// Zig-only: `fx.baseOffset(s) == 16 + idx * 32`.
		idx += count
		expectEqual(t, uint64(idx)*32, fx.usedBytes())
	}
}

// Zig: test "term51: BitmapAllocator.alloc: every chunk count from 1 to 64"
func TestBitmapAllocator_term51_BitmapAllocator_alloc_every_chunk_count_from_1_to_64(t *testing.T) {
	// Two bitmaps (128 chunks of 4 bytes).
	for n := uint(1); n < 65; n++ {
		fx := bmFixtureInit[uint8](t, 4, 512)
		expectEqual(t, uint(2), fx.layout.bitmap_count)

		s := fx.allocChunks(n)
		expectEqual(t, int(n*4), len(s))
		expectEqual(t, uint(0), fx.chunkIndex(s))
		// Zig-only: `fx.baseOffset(s) == 8 * 2`.
		expectEqual(t, uint64(n*4), fx.usedBytes())
		fx.expectUsed([][2]uint{{0, n}})

		// The next chunk is right after (n=64: first chunk of bitmap 1).
		s2 := fx.alloc(1)
		expectEqual(t, n, fx.chunkIndex(s2))
		expectEqual(t, uint64((n+1)*4), fx.usedBytes())

		fx.free(s)
		fx.free(s2)
		fx.expectAllFree()
	}

	// Any byte count within the last chunk uses the same number of chunks.
	for n := uint(1); n < 65; n++ {
		for _, bytes := range []uint{4*n - 3, 4*n - 2, 4*n - 1, 4 * n} {
			fx := bmFixtureInit[uint8](t, 4, 512)
			s := fx.alloc(bytes)
			expectEqual(t, int(bytes), len(s))
			expectEqual(t, uint64(n*4), fx.usedBytes())
			fx.expectUsed([][2]uint{{0, n}})
		}
	}
}

// Zig: test "term51: BitmapAllocator.alloc: chunk counts above 64 and exact fit"
func TestBitmapAllocator_term51_BitmapAllocator_alloc_chunk_counts_above_64_and_exact_fit(t *testing.T) {
	// Three bitmaps (192 chunks).
	fx := bmFixtureInit[uint8](t, 4, 768)
	expectEqual(t, uint(3), fx.layout.bitmap_count)
	expectEqual(t, uint64(768), fx.bm.CapacityBytes())

	for n := uint(65); n < 193; n++ {
		s := fx.allocChunks(n)
		expectEqual(t, uint(0), fx.chunkIndex(s))
		expectEqual(t, uint64(n*4), fx.usedBytes())
		fx.expectUsed([][2]uint{{0, n}})

		if n < 192 {
			// Exactly the rest of the capacity fits.
			s2 := fx.allocChunks(192 - n)
			expectEqual(t, n, fx.chunkIndex(s2))
			expectEqual(t, uint64(768), fx.usedBytes())
			fx.expectUsed([][2]uint{{0, 192}})
			// And now nothing fits, not even one chunk.
			_, err := fx.allocErr(1)
			bmExpectError(t, ErrOutOfMemory, err)
			fx.free(s2)
		}
		fx.free(s)
		fx.expectAllFree()
		expectEqual(t, uint64(0), fx.usedBytes())
	}
}

// Zig: test "term51: BitmapAllocator.alloc: element types and chunk rounding"
func TestBitmapAllocator_term51_BitmapAllocator_alloc_element_types_and_chunk_rounding(t *testing.T) {
	// Go: one allocator of u21 (uint32, 4 bytes) for all three element types,
	// and the u64 and 12-byte-struct allocations are made as the same number
	// of bytes in uint32 units (u64: 2 units each, the struct: 3 units each).
	// Chunk indexes and usedBytes are exactly Zig's; only slice lengths are
	// scaled.
	fx := bmFixtureInit[uint32](t, 16, 4096)

	// u21 is 4 bytes: 4 per chunk.
	counts_u21 := []uint{1, 4, 5, 8, 9, 13}
	chunks_u21 := []uint{1, 1, 2, 2, 3, 4}
	var idx uint
	for i := range counts_u21 {
		n, c := counts_u21[i], chunks_u21[i]
		s := fx.alloc(n)
		expectEqual(t, int(n), len(s))
		expectEqual(t, idx, fx.chunkIndex(s))
		expectEqual(t, uintptr(0), bmAddr(s)%unsafe.Alignof(uint32(0)))
		idx += c
	}
	expectEqual(t, uint64(idx)*16, fx.usedBytes())

	// u64 (8-byte aligned), 2 per chunk.
	s64 := fx.alloc(3 * 2)
	expectEqual(t, 3*2, len(s64)) // Go: 3 elements of 2 units
	expectEqual(t, idx, fx.chunkIndex(s64))
	// Zig-only: `@intFromPtr(s64.ptr) % 8 == 0` (a []uint32 is 4-aligned).
	idx += 2

	// A 12-byte struct: 1 element is 1 chunk, 2 are 24 bytes = 2 chunks,
	// 4 are 48 bytes = 3 chunks, 5 are 60 bytes = 4 chunks.
	counts_12 := []uint{1, 2, 4, 5}
	chunks_12 := []uint{1, 2, 3, 4}
	for i := range counts_12 {
		n, c := counts_12[i], chunks_12[i]
		s := fx.alloc(n * 3)
		expectEqual(t, int(n*3), len(s)) // Go: n elements of 3 units
		expectEqual(t, idx, fx.chunkIndex(s))
		idx += c
	}
	expectEqual(t, uint64(idx)*16, fx.usedBytes())
}

// Zig: test "term51: BitmapAllocator.alloc: first fit with holes and wrapping to later bitmaps"
func TestBitmapAllocator_term51_BitmapAllocator_alloc_first_fit_with_holes_and_wrapping_to_later_bitmaps(t *testing.T) {
	fx := bmFixtureInit[uint8](t, 1, 192)
	expectEqual(t, uint(3), fx.layout.bitmap_count)

	a := fx.alloc(60)
	expectEqual(t, uint(0), fx.chunkIndex(a))
	// 10 chunks do not fit in the 4 left in bitmap 0: wrap to bitmap 1.
	b := fx.alloc(10)
	expectEqual(t, uint(64), fx.chunkIndex(b))
	// The 4-chunk hole in bitmap 0 is used by a 4-chunk request.
	c := fx.alloc(4)
	expectEqual(t, uint(60), fx.chunkIndex(c))
	// 64 chunks need a bitmap of their own: bitmap 0 is full, bitmap 1 is
	// partly used, so bitmap 2 is taken.
	d := fx.alloc(64)
	expectEqual(t, uint(128), fx.chunkIndex(d))
	// The rest of bitmap 1 (54 chunks) fits exactly.
	e := fx.alloc(54)
	expectEqual(t, uint(74), fx.chunkIndex(e))
	expectEqual(t, uint64(192), fx.usedBytes())
	_, err := fx.allocErr(1)
	bmExpectError(t, ErrOutOfMemory, err)

	// Free the middle one: a request that is too big for the hole fails,
	// one that fits takes the hole's lowest position.
	fx.free(b)
	expectEqual(t, uint64(182), fx.usedBytes())
	_, err = fx.allocErr(11)
	bmExpectError(t, ErrOutOfMemory, err)
	f := fx.alloc(3)
	expectEqual(t, uint(64), fx.chunkIndex(f))
	g := fx.alloc(7)
	expectEqual(t, uint(67), fx.chunkIndex(g))
	_, err = fx.allocErr(1)
	bmExpectError(t, ErrOutOfMemory, err)

	// Free the first allocation and the 4-chunk block: bitmap 0 is entirely
	// free again, so a request for 64 chunks gets it.
	fx.free(a)
	fx.free(c)
	h := fx.alloc(64)
	expectEqual(t, uint(0), fx.chunkIndex(h))
}

// Zig: test "term51: BitmapAllocator.alloc: fragmentation across a bitmap boundary"
func TestBitmapAllocator_term51_BitmapAllocator_alloc_fragmentation_across_a_bitmap_boundary(t *testing.T) {
	fx := bmFixtureInit[uint8](t, 1, 128)

	a := fx.alloc(32)
	b := fx.alloc(32)
	c := fx.alloc(32)
	d := fx.alloc(32)
	expectEqual(t, uint(32), fx.chunkIndex(b))
	expectEqual(t, uint(64), fx.chunkIndex(c))
	expectEqual(t, uint(96), fx.chunkIndex(d))
	_ = a

	// 64 contiguous free chunks (32..96) straddling the bitmap boundary.
	fx.free(b)
	fx.free(c)
	expectEqual(t, uint64(64), fx.usedBytes())
	fx.expectUsed([][2]uint{{0, 32}, {96, 128}})

	// Requests up to 64 cannot span the boundary; 65 and over is the large
	// path and finds no run of that size.
	for _, n := range []uint{33, 40, 64, 65, 100, 128} {
		_, err := fx.allocErr(n)
		bmExpectError(t, ErrOutOfMemory, err)
		fx.expectUsed([][2]uint{{0, 32}, {96, 128}})
	}

	// 32 chunks fit, at the first hole.
	e := fx.alloc(32)
	expectEqual(t, uint(32), fx.chunkIndex(e))
	f := fx.alloc(32)
	expectEqual(t, uint(64), fx.chunkIndex(f))
}

// Zig: test "term51: BitmapAllocator.alloc: large request exact fit in a hole and one past"
func TestBitmapAllocator_term51_BitmapAllocator_alloc_large_request_exact_fit_in_a_hole_and_one_past(t *testing.T) {
	fx := bmFixtureInit[uint8](t, 1, 256)
	expectEqual(t, uint(4), fx.layout.bitmap_count)

	a := fx.alloc(100)
	b := fx.alloc(100)
	c := fx.alloc(56)
	expectEqual(t, uint(0), fx.chunkIndex(a))
	expectEqual(t, uint(100), fx.chunkIndex(b))
	expectEqual(t, uint(200), fx.chunkIndex(c))
	expectEqual(t, uint64(256), fx.usedBytes())

	// A 100-chunk hole in the middle, spanning three bitmaps.
	fx.free(b)
	fx.expectUsed([][2]uint{{0, 100}, {200, 256}})
	_, err := fx.allocErr(101)
	bmExpectError(t, ErrOutOfMemory, err)
	fx.expectUsed([][2]uint{{0, 100}, {200, 256}})
	_, err = fx.allocErr(150)
	bmExpectError(t, ErrOutOfMemory, err)
	b2 := fx.alloc(100)
	expectEqual(t, uint(100), fx.chunkIndex(b2))
	fx.expectUsed([][2]uint{{0, 256}})
	_, err = fx.allocErr(1)
	bmExpectError(t, ErrOutOfMemory, err)
}

// Zig: test "term51: BitmapAllocator.alloc: single path capacity exhausted and hole reuse"
func TestBitmapAllocator_term51_BitmapAllocator_alloc_single_path_capacity_exhausted_and_hole_reuse(t *testing.T) {
	fx := bmFixtureInit[uint8](t, 16, 1024)
	expectEqual(t, uint(1), fx.layout.bitmap_count)

	var slices [64][]uint8
	for i := range slices {
		slices[i] = fx.alloc(16)
		expectEqual(t, uint(i), fx.chunkIndex(slices[i]))
	}
	expectEqual(t, uint64(1024), fx.usedBytes())
	expectEqual(t, fx.bm.CapacityBytes(), fx.usedBytes())
	expectEqual(t, uint64(0), fx.bitmaps()[0])

	// One past the capacity.
	_, err := fx.allocErr(1)
	bmExpectError(t, ErrOutOfMemory, err)
	_, err = fx.allocErr(16)
	bmExpectError(t, ErrOutOfMemory, err)

	// A single freed chunk is reused, but not for two.
	fx.free(slices[37])
	expectEqual(t, uint64(1)<<37, fx.bitmaps()[0])
	_, err = fx.allocErr(17)
	bmExpectError(t, ErrOutOfMemory, err)
	again := fx.alloc(1)
	expectEqual(t, uint(37), fx.chunkIndex(again))
	_, err = fx.allocErr(1)
	bmExpectError(t, ErrOutOfMemory, err)

	// Adjacent freed chunks form a larger hole.
	fx.free(slices[10])
	fx.free(slices[11])
	fx.free(slices[12])
	_, err = fx.allocErr(49)
	bmExpectError(t, ErrOutOfMemory, err)
	three := fx.alloc(48)
	expectEqual(t, uint(10), fx.chunkIndex(three))
	expectEqual(t, uint64(0), fx.bitmaps()[0])
}

// Zig: test "term51: BitmapAllocator.alloc: zero capacity and huge counts are out of memory"
func TestBitmapAllocator_term51_BitmapAllocator_alloc_zero_capacity_and_huge_counts_are_out_of_memory(t *testing.T) {
	{
		fx := bmFixtureInit[uint8](t, 16, 0)
		_, err := fx.allocErr(1)
		bmExpectError(t, ErrOutOfMemory, err)
		_, err = fx.allocErr(64 * 16)
		bmExpectError(t, ErrOutOfMemory, err)
		_, err = fx.allocErr(64*16 + 1)
		bmExpectError(t, ErrOutOfMemory, err)
		_, err = fx.allocErr(100000)
		bmExpectError(t, ErrOutOfMemory, err)
	}

	// Go: the Zig fixture hands out every element type; here one allocator
	// per element type (u8, u21 = uint32, [3]u8) over the same layout.
	fx := bmFixtureInit[uint8](t, 16, 1024)
	fx32 := bmFixtureInit[uint32](t, 16, 1024)
	fx3 := bmFixtureInit[[3]uint8](t, 16, 1024)
	// The byte count overflows usize.
	_, err := fx32.allocErr(math.MaxUint)
	bmExpectError(t, ErrOutOfMemory, err)
	_, err = fx32.allocErr(math.MaxUint/4 + 1)
	bmExpectError(t, ErrOutOfMemory, err)
	_, err = fx3.allocErr(math.MaxUint)
	bmExpectError(t, ErrOutOfMemory, err)
	// The byte count does not overflow but vastly exceeds the bitmaps.
	_, err = fx.allocErr(math.MaxUint)
	bmExpectError(t, ErrOutOfMemory, err)
	_, err = fx32.allocErr(math.MaxUint / 4)
	bmExpectError(t, ErrOutOfMemory, err)
	// 1 << 40 (Zig's usize is 64 bits; on a 32-bit target it saturates).
	_, err = fx.allocErr(uint(min(uint64(1)<<40, uint64(math.MaxUint))))
	bmExpectError(t, ErrOutOfMemory, err)
	_, err = fx.allocErr(1024 + 64*16 + 1)
	bmExpectError(t, ErrOutOfMemory, err)
	expectEqual(t, uint64(0), fx.usedBytes())
	fx.expectAllFree()
	expectEqual(t, uint64(0), fx32.usedBytes())
	fx32.expectAllFree()
	expectEqual(t, uint64(0), fx3.usedBytes())
	fx3.expectAllFree()
}

// Zig: test "term51: BitmapAllocator.free: every position and length at every order"
func TestBitmapAllocator_term51_BitmapAllocator_free_every_position_and_length_at_every_order(t *testing.T) {
	// Three bitmaps (192 chunks). Every chunk is marked used by hand (alloc
	// never places a run of up to 64 chunks across a bitmap boundary, but
	// free has no such restriction). For every [p, p+len) take a "before"
	// block, the target and an "after" block that exactly fill the allocator,
	// then free them in a rotating order and check the bitmap after each step.
	fx := bmFixtureInit[uint8](t, 4, 768)
	expectEqual(t, uint(3), fx.layout.bitmap_count)
	chunks := fx.bm.chunks

	for p := uint(0); p < 193; p++ {
		for length := uint(1); length < 193-p; length++ {
			bmFill(fx.bitmaps(), 0)
			bounds := [3][2]uint{{0, p}, {p, p + length}, {p + length, 192}}
			orders := [3][3]uint{{1, 2, 0}, {0, 1, 2}, {2, 0, 1}}
			order := orders[(p+length)%3]
			live := [3]bool{true, true, true}
			for _, which := range order {
				bnd := bounds[which]
				if bnd[1] > bnd[0] {
					fx.free(chunks[bnd[0]*4 : bnd[1]*4])
				}
				live[which] = false
				var ranges [][2]uint
				for i, other := range bounds {
					if live[i] && other[1] > other[0] {
						ranges = append(ranges, other)
					}
				}
				fx.expectUsed(ranges)
			}
			fx.expectAllFree()
			expectEqual(t, uint64(0), fx.usedBytes())
		}
	}
}

// Zig: test "term51: BitmapAllocator.free: partial slices free the chunks they cover"
func TestBitmapAllocator_term51_BitmapAllocator_free_partial_slices_free_the_chunks_they_cover(t *testing.T) {
	fx := bmFixtureInit[uint8](t, 4, 256)

	// 16 bytes = chunks 0..4.
	s := fx.alloc(16)
	expectEqual(t, uint(0), fx.chunkIndex(s))
	fx.expectUsed([][2]uint{{0, 4}})

	// A 4-byte prefix is exactly chunk 0.
	fx.free(s[0:4])
	fx.expectUsed([][2]uint{{1, 4}})
	expect(t, !fx.bm.isAllocated(s[0:4]), "!fx.bm.isAllocated(fx.buf, s[0..4])")
	expect(t, fx.bm.isAllocated(s[4:16]), "fx.bm.isAllocated(fx.buf, s[4..16])")
	// A 4-byte suffix is exactly chunk 3.
	fx.free(s[12:16])
	fx.expectUsed([][2]uint{{1, 3}})
	// A 1-byte slice still frees a whole chunk (chunk 1).
	fx.free(s[4:5])
	fx.expectUsed([][2]uint{{2, 3}})
	fx.free(s[8:9])
	fx.expectAllFree()

	// A 5-byte slice is rounded up to 2 chunks.
	tt := fx.alloc(8)
	fx.expectUsed([][2]uint{{0, 2}})
	fx.free(tt[0:5])
	fx.expectAllFree()

	// A shorter slice at the start frees only what it covers, leaving the
	// chunk beyond: 4 bytes of an 8-byte allocation.
	u := fx.alloc(8)
	fx.free(u[0:4])
	fx.expectUsed([][2]uint{{1, 2}})
	fx.free(u[4:8])
	fx.expectAllFree()
}

// Zig: test "term51: BitmapAllocator.free: slices of wider element types"
func TestBitmapAllocator_term51_BitmapAllocator_free_slices_of_wider_element_types(t *testing.T) {
	// Go: one allocator of u21 (uint32); the u64 allocation is made as the
	// same number of bytes in uint32 units (2 per element).
	fx := bmFixtureInit[uint32](t, 16, 1024)

	// 5 u21 = 20 bytes = 2 chunks; free the two halves separately.
	s := fx.alloc(5)
	fx.expectUsed([][2]uint{{0, 2}})
	fx.free(s[0:4])
	fx.expectUsed([][2]uint{{1, 2}})
	fx.free(s[4:5])
	fx.expectAllFree()

	// 6 u21 = 24 bytes = 2 chunks, freed whole.
	tt := fx.alloc(6)
	fx.expectUsed([][2]uint{{0, 2}})
	fx.free(tt)
	fx.expectAllFree()

	// u64: 5 elements = 40 bytes = 3 chunks.
	u := fx.alloc(5 * 2)
	fx.expectUsed([][2]uint{{0, 3}})
	fx.free(u)
	fx.expectAllFree()
}

// Zig: test "term51: BitmapAllocator.free: bitmap edge positions in a one chunk allocator"
func TestBitmapAllocator_term51_BitmapAllocator_free_bitmap_edge_positions_in_a_one_chunk_allocator(t *testing.T) {
	fx := bmFixtureInit[uint8](t, 1, 192)

	all := fx.alloc(192)
	expectEqual(t, uint(0), fx.chunkIndex(all))
	expectEqualSlices(t, []uint64{0, 0, 0}, fx.bitmaps())

	// Free the last chunk of each bitmap and the first of the next.
	fx.free(all[63:65])
	expectEqualSlices(t, []uint64{uint64(1) << 63, 1, 0}, fx.bitmaps())
	fx.free(all[127:129])
	expectEqualSlices(t, []uint64{uint64(1) << 63, (uint64(1) << 63) | 1, 1}, fx.bitmaps())
	// The very last chunk (frees through the end of the last bitmap).
	fx.free(all[191:192])
	expectEqualSlices(t, []uint64{uint64(1) << 63, (uint64(1) << 63) | 1, (uint64(1) << 63) | 1}, fx.bitmaps())
	// Exactly one whole middle bitmap.
	fx.free(all[64:128])
	expectEqualSlices(t, []uint64{uint64(1) << 63, bmMax, (uint64(1) << 63) | 1}, fx.bitmaps())
}

// Zig: test "term51: BitmapAllocator.usedBytes: counts used chunks across all bitmaps"
func TestBitmapAllocator_term51_BitmapAllocator_usedBytes_counts_used_chunks_across_all_bitmaps(t *testing.T) {
	// Go: one allocator of u8; the u21 allocation is made as 4 bytes per
	// element.
	fx := bmFixtureInit[uint8](t, 16, 4096)
	expectEqual(t, uint(4), fx.layout.bitmap_count)
	expectEqual(t, uint64(0), fx.usedBytes())

	a := fx.alloc(1)
	expectEqual(t, uint64(16), fx.usedBytes())
	b := fx.alloc(100) // 7 chunks
	expectEqual(t, uint64(8*16), fx.usedBytes())
	c := fx.alloc(4 * 100) // 100 u21: 400 bytes: 25 chunks
	expectEqual(t, uint64(33*16), fx.usedBytes())
	d := fx.alloc(70 * 16) // large path, 70 chunks
	expectEqual(t, uint64(103*16), fx.usedBytes())
	fx.free(b)
	expectEqual(t, uint64(96*16), fx.usedBytes())
	fx.free(d)
	expectEqual(t, uint64(26*16), fx.usedBytes())
	fx.free(a)
	fx.free(c)
	expectEqual(t, uint64(0), fx.usedBytes())

	// Fill everything: used equals capacity.
	all := fx.alloc(4096)
	expectEqual(t, uint64(4096), fx.usedBytes())
	expectEqual(t, fx.bm.CapacityBytes(), fx.usedBytes())
	fx.free(all)
	expectEqual(t, uint64(0), fx.usedBytes())

	// Hand-built bitmaps: only the popcount of the first bitmap_count words
	// matters. free = 0 + 64 + 4 + 1 = 69 of 256; used = 187 chunks.
	bms := fx.bitmaps()
	bms[0] = 0
	bms[1] = bmMax
	bms[2] = 0xF
	bms[3] = 1
	expectEqual(t, uint64(187*16), fx.usedBytes())
	bms[0] = bmMax
	bms[1] = 0
	bms[2] = 0
	bms[3] = 0
	expectEqual(t, uint64(192*16), fx.usedBytes())
}

// Zig: test "term51: BitmapAllocator.usedBytes: string and grapheme allocators at page sizes"
func TestBitmapAllocator_term51_BitmapAllocator_usedBytes_string_and_grapheme_allocators_at_page_sizes(t *testing.T) {
	// page.zig's defaults: 64 chunks each.
	{
		fx := bmFixtureInit[uint32](t, 16, 64*16)
		expectEqual(t, uint64(0), fx.usedBytes())
		s := fx.alloc(6)
		expectEqual(t, uint64(32), fx.usedBytes())
		fx.free(s)
		expectEqual(t, uint64(0), fx.usedBytes())
	}
	{
		fx := bmFixtureInit[uint8](t, 32, 64*32)
		s := fx.alloc(19)
		expectEqual(t, uint64(32), fx.usedBytes())
		tt := fx.alloc(40)
		expectEqual(t, uint64(96), fx.usedBytes())
		fx.free(s)
		expectEqual(t, uint64(64), fx.usedBytes())
		fx.free(tt)
		expectEqual(t, uint64(0), fx.usedBytes())
	}
}

// Zig: test "term51: BitmapAllocator.capacityBytes: bitmaps times 64 times the chunk size"
func TestBitmapAllocator_term51_BitmapAllocator_capacityBytes_bitmaps_times_64_times_the_chunk_size(t *testing.T) {
	// Directly from the bitmap count.
	for _, count := range []uint{0, 1, 2, 7, 69, 1000} {
		bm := BitmapAllocator[uint8]{chunk_size: bmGraphemeChunk, bitmap_count: count}
		expectEqual(t, uint64(count)*64*16, bm.CapacityBytes())
	}
	for _, count := range []uint{0, 1, 2, 7, 35, 1000} {
		bm := BitmapAllocator[uint8]{chunk_size: bmStringChunk, bitmap_count: count}
		expectEqual(t, uint64(count)*64*32, bm.CapacityBytes())
	}

	// From capacities through layout and init: rounds up to a whole bitmap.
	caps := []uint64{0, 1, 16, 1023, 1024, 1025, 2048, 2049, 4096, 70000}
	for _, cap_ := range caps {
		{
			fx := bmFixtureInit[uint8](t, 16, cap_)
			expectEqual(t, (cap_+1023)/1024*1024, fx.bm.CapacityBytes())
		}
		{
			fx := bmFixtureInit[uint8](t, 32, cap_)
			expectEqual(t, (cap_+2047)/2048*2048, fx.bm.CapacityBytes())
		}
	}

	// Pinned examples.
	{
		fx := bmFixtureInit[uint8](t, 16, 1025)
		expectEqual(t, uint64(2048), fx.bm.CapacityBytes())
	}
	{
		fx := bmFixtureInit[uint8](t, 32, 2048)
		expectEqual(t, uint64(2048), fx.bm.CapacityBytes())
	}
}

// Zig: test "term51: BitmapAllocator.isAllocated: true only when every chunk is used"
func TestBitmapAllocator_term51_BitmapAllocator_isAllocated_true_only_when_every_chunk_is_used(t *testing.T) {
	fx := bmFixtureInit[uint8](t, 4, 512)

	s := fx.alloc(12)
	expect(t, fx.bm.isAllocated(s), "fx.bm.isAllocated(fx.buf, s)")
	expect(t, fx.bm.isAllocated(s[0:4]), "fx.bm.isAllocated(fx.buf, s[0..4])")
	expect(t, fx.bm.isAllocated(s[8:12]), "fx.bm.isAllocated(fx.buf, s[8..12])")
	// A slice just past the allocation covers a free chunk. (Zig:
	// `s.ptr[12..16]`, a many-pointer slice past s; s starts at chunk 0, so
	// in Go it is that range of `chunks`.)
	expect(t, !fx.bm.isAllocated(fx.bm.chunks[12:16]), "!fx.bm.isAllocated(fx.buf, s.ptr[12..16])")

	// Freeing the middle chunk makes the whole slice not allocated while
	// its ends still are.
	fx.free(s[4:8])
	expect(t, !fx.bm.isAllocated(s), "!fx.bm.isAllocated(fx.buf, s)")
	expect(t, fx.bm.isAllocated(s[0:4]), "fx.bm.isAllocated(fx.buf, s[0..4])")
	expect(t, !fx.bm.isAllocated(s[4:8]), "!fx.bm.isAllocated(fx.buf, s[4..8])")
	expect(t, fx.bm.isAllocated(s[8:12]), "fx.bm.isAllocated(fx.buf, s[8..12])")
	// A slice crossing a bitmap boundary (chunk 63 and 64).
	fx.free(s[0:4])
	fx.free(s[8:12])
	big := fx.alloc(66 * 4)
	expect(t, fx.bm.isAllocated(big), "fx.bm.isAllocated(fx.buf, big)")
	fx.free(big[0:4])
	expect(t, !fx.bm.isAllocated(big), "!fx.bm.isAllocated(fx.buf, big)")
	expect(t, fx.bm.isAllocated(big[4:]), "fx.bm.isAllocated(fx.buf, big[4..])")
}

// Zig: test "term51: BitmapAllocator.dumpBitmaps: logs without changing the bitmaps"
func TestBitmapAllocator_term51_BitmapAllocator_dumpBitmaps_logs_without_changing_the_bitmaps(t *testing.T) {
	fx := bmFixtureInit[uint8](t, 4, 512)
	_ = fx.alloc(4 * 5)
	before := []uint64{fx.bitmaps()[0], fx.bitmaps()[1]}
	expectEqual(t, bmMax<<5, before[0])

	fx.bm.dumpBitmaps()
	expectEqualSlices(t, before, fx.bitmaps())
	expectEqual(t, uint64(20), fx.usedBytes())
}

// Go-only: panic test: BitmapAllocator.alloc: n is zero: assert(n > 0)
func TestBitmapAllocator_panic_alloc_n_is_zero(t *testing.T) {
	bm := BitmapAllocatorInit[uint8](4, BitmapAllocatorLayoutOf(4, 64))
	bmExpectPanic(t, func() { _, _ = bm.Alloc(0) })
}

// Go-only: panic test: BitmapAllocator.alloc: chunk size is not a multiple of the element alignment: assert(chunk_size % @alignOf(T) == 0)
func TestBitmapAllocator_panic_alloc_chunk_size_not_a_multiple_of_the_element_alignment(t *testing.T) {
	// A uint32 element has alignment 4 on every target; a chunk of 2 bytes
	// is not a multiple of it.
	bm := BitmapAllocatorInit[uint32](2, BitmapAllocatorLayoutOf(2, 128))
	bmExpectPanic(t, func() { _, _ = bm.Alloc(1) })
}

// Go-only: panic test: BitmapAllocator.free: empty slice: `~@as(u64, 0) >> @intCast(64 - bits)` with bits == 0 casts 64 to a u6
func TestBitmapAllocator_panic_free_empty_slice(t *testing.T) {
	bm := BitmapAllocatorInit[uint8](4, BitmapAllocatorLayoutOf(4, 64))
	s := bmTry(bm.Alloc(4)).ok(t)
	bmExpectPanic(t, func() { bm.Free(s[0:0]) })
}

// Go-only: panic test: BitmapAllocator.free: slice does not start at a chunk boundary: @divExact(@intFromPtr(slice.ptr) - @intFromPtr(chunks), chunk_size)
func TestBitmapAllocator_panic_free_slice_not_at_a_chunk_boundary(t *testing.T) {
	bm := BitmapAllocatorInit[uint8](4, BitmapAllocatorLayoutOf(4, 64))
	s := bmTry(bm.Alloc(8)).ok(t)
	bmExpectPanic(t, func() { bm.Free(s[1:5]) })
}

// Go-only: panic test: findFreeChunks: n is zero: `for (1..n)` with n == 0 (start larger than end) and `@intCast(64 - n)` casting 64 to a u6
func TestBitmapAllocator_panic_findFreeChunks_n_is_zero(t *testing.T) {
	bitmaps := []uint64{bmMax}
	bmExpectPanic(t, func() { _ = bmFind(bitmaps, 0) })
}

// Go-only: panic test: BitmapAllocator: chunk size is not a power of two: comptime assert(std.math.isPowerOfTwo(chunk_size)), a compile error in Zig, a run-time assertion in BitmapAllocatorInit in Go
func TestBitmapAllocator_panic_init_chunk_size_not_a_power_of_two(t *testing.T) {
	bmExpectPanic(t, func() { _ = BitmapAllocatorInit[uint8](3, BitmapAllocatorLayoutOf(4, 64)) })
}
