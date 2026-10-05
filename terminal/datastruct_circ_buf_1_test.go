// Tests translated one for one from src/datastruct/circ_buf.zig (Ghostty
// v1.3.1 plus the term51 additional tests), first half: the unnamed test and
// the tests from line 325 through 1566 of the Zig file. The second half is
// datastruct_circ_buf_2_test.go (which also holds the Go-only panic tests).
//
// Translation notes:
//   - `const alloc = std.testing.allocator;` and the allocator argument of
//     init / deinit / resize / ensureUnusedCapacity are dropped everywhere
//     (allocator parameters do not exist in Go; docs/mapping.md).
//   - `defer buf.deinit(alloc)` is `defer buf.Deinit()`. Zig's defer inside a
//     loop body runs at the end of each iteration, Go's runs when the test
//     function returns.
//   - `Buf.init(alloc, n)` is `cbInit(t, default, n)`: DatastructCircBufInit
//     with the Zig default first, then `try`.
//   - `term51ANShift`, `term51ANExpect`, `term51ANExpectIter`,
//     `term51ANTypeCycle` and `term51ANElem` are cbShift, cbExpect,
//     cbExpectIter, cbTypeCycle and cbElem; they are used by both files.
//   - `x.?.*` is `*cbSome(t, x)`; Zig's `it.next().?.*` is `cbNext(t, &it)`.
//   - Pointer comparisons of storage elements stay pointer comparisons of
//     `&buf.storage[i]`; comparisons of a slice's raw `.ptr` are slice-index
//     comparisons (`&s[0] == &buf.storage[i]`).
//   - Zig's `usize` is `uint`, `isize` is `int`; `std.math.maxInt(usize)` and
//     friends use math.MaxUint / math.MaxInt so that the tests hold for a
//     32-bit build too.

package terminal

import (
	"errors"
	"math"
	"testing"
)

// cbInit is `try Buf.init(alloc, size)` for CircBuf(u8, default).
func cbInit(t *testing.T, default_ uint8, size uint) DatastructCircBuf[uint8] {
	t.Helper()
	buf, err := DatastructCircBufInit[uint8](default_, size)
	zigTry(t, err)
	return buf
}

// cbExpectOOM is `try testing.expectError(error.OutOfMemory, err)`.
func cbExpectOOM(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, ErrOutOfMemory) {
		t.Fatalf("got error %v, want OutOfMemory", err)
	}
}

// cbSome is Zig's `p.?` for an optional pointer: it fails if p is null.
func cbSome[T any](t *testing.T, p *T) *T {
	t.Helper()
	if p == nil {
		t.Fatalf("unwrapped a null optional")
	}
	return p
}

// cbNext is `it.next().?.*`.
func cbNext[T any](t *testing.T, it *DatastructCircBufIterator[T]) T {
	t.Helper()
	return *cbSome(t, it.Next())
}

// cbAllEqual is `std.mem.allEqual(u8, s, v)`.
func cbAllEqual(s []uint8, v uint8) bool {
	for _, e := range s {
		if e != v {
			return false
		}
	}
	return true
}

// cbExpectPanic runs f and fails the test unless f panics. Zig's tests cannot
// test an assert or a safety-checked panic: the test would abort.
func cbExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if recover() == nil {
			t.Fatalf("expected a panic, got none")
		}
	}()
	f()
}

// cbShift is term51ANShift: moves head and tail of an empty buffer to
// position p (p < capacity) by appending p filler bytes and deleting them
// again.
func cbShift(buf *DatastructCircBuf[uint8], p uint) {
	for i := uint(0); i < p; i++ {
		buf.AppendAssumeCapacity(0xEE)
	}
	buf.DeleteOldest(p)
}

// cbExpect is term51ANExpect: asserts the logical contents (oldest first)
// through len and both iterator directions.
func cbExpect(t *testing.T, buf *DatastructCircBuf[uint8], expected []uint8) {
	t.Helper()
	expectEqual(t, uint(len(expected)), buf.Len())
	it := buf.Iterator(DatastructCircBufIteratorDirectionForward)
	for _, e := range expected {
		expectEqual(t, e, cbNext(t, &it))
	}
	expect(t, it.Next() == nil, "it.next() == null")
	rit := buf.Iterator(DatastructCircBufIteratorDirectionReverse)
	i := len(expected)
	for i > 0 {
		i--
		expectEqual(t, expected[i], cbNext(t, &rit))
	}
	expect(t, rit.Next() == nil, "rit.next() == null")
}

// cbExpectIter is term51ANExpectIter: forward iteration only, for states
// where len() is not the logical length.
func cbExpectIter(t *testing.T, buf *DatastructCircBuf[uint8], expected []uint8) {
	t.Helper()
	it := buf.Iterator(DatastructCircBufIteratorDirectionForward)
	for _, e := range expected {
		expectEqual(t, e, cbNext(t, &it))
	}
	expect(t, it.Next() == nil, "it.next() == null")
}

// cbElem is term51ANElem.
type cbElem struct {
	a uint8
	b uint32
	c uint16
}

// cbU128 stands for Zig's u128 (Go has no 128-bit integer): lo is the low
// and hi the high 64 bits.
type cbU128 struct {
	lo uint64
	hi uint64
}

// cbTypeCycle is term51ANTypeCycle: exercises append / deleteOldest / wrap /
// resize for one element type.
func cbTypeCycle[T comparable](t *testing.T, default_ T, a T, b T, c T) {
	t.Helper()
	buf, err := DatastructCircBufInit[T](default_, 3)
	zigTry(t, err)
	defer buf.Deinit()
	for _, v := range buf.storage {
		expectEqual(t, default_, v)
	}

	zigTry(t, buf.Append(a))
	zigTry(t, buf.Append(b))
	zigTry(t, buf.Append(c))
	cbExpectOOM(t, buf.Append(a))
	expectEqual(t, 3, buf.Len())

	// Delete the oldest: its slot goes back to the default.
	buf.DeleteOldest(1)
	expectEqual(t, default_, buf.storage[0])
	expectEqual(t, 2, buf.Len())

	// Wrap around.
	zigTry(t, buf.Append(a))
	expectEqual(t, a, buf.storage[0])
	it := buf.Iterator(DatastructCircBufIteratorDirectionForward)
	expectEqual(t, b, cbNext(t, &it))
	expectEqual(t, c, cbNext(t, &it))
	expectEqual(t, a, cbNext(t, &it))
	expect(t, it.Next() == nil, "it.next() == null")
	expectEqual(t, b, *cbSome(t, buf.First()))
	expectEqual(t, a, *cbSome(t, buf.Last()))

	// Grow keeps order, new slots are the default.
	zigTry(t, buf.Resize(5))
	expectEqual(t, 5, buf.Capacity())
	expectEqual(t, b, buf.storage[0])
	expectEqual(t, c, buf.storage[1])
	expectEqual(t, a, buf.storage[2])
	expectEqual(t, default_, buf.storage[3])
	expectEqual(t, default_, buf.storage[4])
	zigTry(t, buf.Append(c))
	expectEqual(t, c, *cbSome(t, buf.Last()))
	expectEqual(t, 4, buf.Len())
}

// Zig: test (unnamed, circ_buf.zig:313)
func TestCb1_unnamed(t *testing.T) {
	buf := cbInit(t, 0, 12)
	defer buf.Deinit()

	expect(t, buf.Empty(), "buf.empty()")
	expectEqual(t, 0, buf.Len())
}

// Zig: test "CircBuf append"
func TestCb1_CircBuf_append(t *testing.T) {
	buf := cbInit(t, 0, 3)
	defer buf.Deinit()

	zigTry(t, buf.Append(1))
	zigTry(t, buf.Append(2))
	zigTry(t, buf.Append(3))
	cbExpectOOM(t, buf.Append(4))
	buf.DeleteOldest(1)
	zigTry(t, buf.Append(4))
	cbExpectOOM(t, buf.Append(5))
}

// Zig: test "CircBuf forward iterator"
func TestCb1_CircBuf_forward_iterator(t *testing.T) {
	buf := cbInit(t, 0, 3)
	defer buf.Deinit()

	// Empty
	{
		it := buf.Iterator(DatastructCircBufIteratorDirectionForward)
		expect(t, it.Next() == nil, "it.next() == null")
	}

	// Partially full
	zigTry(t, buf.Append(1))
	zigTry(t, buf.Append(2))
	{
		it := buf.Iterator(DatastructCircBufIteratorDirectionForward)
		expect(t, *cbSome(t, it.Next()) == 1, "it.next().?.* == 1")
		expect(t, *cbSome(t, it.Next()) == 2, "it.next().?.* == 2")
		expect(t, it.Next() == nil, "it.next() == null")
	}

	// Full
	zigTry(t, buf.Append(3))
	{
		it := buf.Iterator(DatastructCircBufIteratorDirectionForward)
		expect(t, *cbSome(t, it.Next()) == 1, "it.next().?.* == 1")
		expect(t, *cbSome(t, it.Next()) == 2, "it.next().?.* == 2")
		expect(t, *cbSome(t, it.Next()) == 3, "it.next().?.* == 3")
		expect(t, it.Next() == nil, "it.next() == null")
	}

	// Delete and add
	buf.DeleteOldest(1)
	zigTry(t, buf.Append(4))
	{
		it := buf.Iterator(DatastructCircBufIteratorDirectionForward)
		expect(t, *cbSome(t, it.Next()) == 2, "it.next().?.* == 2")
		expect(t, *cbSome(t, it.Next()) == 3, "it.next().?.* == 3")
		expect(t, *cbSome(t, it.Next()) == 4, "it.next().?.* == 4")
		expect(t, it.Next() == nil, "it.next() == null")
	}
}

// Zig: test "CircBuf reverse iterator"
func TestCb1_CircBuf_reverse_iterator(t *testing.T) {
	buf := cbInit(t, 0, 3)
	defer buf.Deinit()

	// Empty
	{
		it := buf.Iterator(DatastructCircBufIteratorDirectionReverse)
		expect(t, it.Next() == nil, "it.next() == null")
	}

	// Partially full
	zigTry(t, buf.Append(1))
	zigTry(t, buf.Append(2))
	{
		it := buf.Iterator(DatastructCircBufIteratorDirectionReverse)
		expect(t, *cbSome(t, it.Next()) == 2, "it.next().?.* == 2")
		expect(t, *cbSome(t, it.Next()) == 1, "it.next().?.* == 1")
		expect(t, it.Next() == nil, "it.next() == null")
	}

	// Full
	zigTry(t, buf.Append(3))
	{
		it := buf.Iterator(DatastructCircBufIteratorDirectionReverse)
		expect(t, *cbSome(t, it.Next()) == 3, "it.next().?.* == 3")
		expect(t, *cbSome(t, it.Next()) == 2, "it.next().?.* == 2")
		expect(t, *cbSome(t, it.Next()) == 1, "it.next().?.* == 1")
		expect(t, it.Next() == nil, "it.next() == null")
	}

	// Delete and add
	buf.DeleteOldest(1)
	zigTry(t, buf.Append(4))
	{
		it := buf.Iterator(DatastructCircBufIteratorDirectionReverse)
		expect(t, *cbSome(t, it.Next()) == 4, "it.next().?.* == 4")
		expect(t, *cbSome(t, it.Next()) == 3, "it.next().?.* == 3")
		expect(t, *cbSome(t, it.Next()) == 2, "it.next().?.* == 2")
		expect(t, it.Next() == nil, "it.next() == null")
	}
}

// Zig: test "CircBuf first/last"
func TestCb1_CircBuf_first_last(t *testing.T) {
	buf := cbInit(t, 0, 3)
	defer buf.Deinit()

	zigTry(t, buf.Append(1))
	zigTry(t, buf.Append(2))
	zigTry(t, buf.Append(3))
	expectEqual(t, 3, *cbSome(t, buf.Last()))
	expectEqual(t, 1, *cbSome(t, buf.First()))
}

// Zig: test "CircBuf first/last empty"
func TestCb1_CircBuf_first_last_empty(t *testing.T) {
	buf := cbInit(t, 0, 0)
	defer buf.Deinit()

	expect(t, buf.First() == nil, "buf.first() == null")
	expect(t, buf.Last() == nil, "buf.last() == null")
}

// Zig: test "CircBuf first/last empty with cap"
func TestCb1_CircBuf_first_last_empty_with_cap(t *testing.T) {
	buf := cbInit(t, 0, 3)
	defer buf.Deinit()

	expect(t, buf.First() == nil, "buf.first() == null")
	expect(t, buf.Last() == nil, "buf.last() == null")
}

// Zig: test "CircBuf append slice"
func TestCb1_CircBuf_append_slice(t *testing.T) {
	buf := cbInit(t, 0, 5)
	defer buf.Deinit()

	buf.AppendSliceAssumeCapacity([]uint8("hello"))
	{
		it := buf.Iterator(DatastructCircBufIteratorDirectionForward)
		expect(t, *cbSome(t, it.Next()) == 'h', "it.next().?.* == 'h'")
		expect(t, *cbSome(t, it.Next()) == 'e', "it.next().?.* == 'e'")
		expect(t, *cbSome(t, it.Next()) == 'l', "it.next().?.* == 'l'")
		expect(t, *cbSome(t, it.Next()) == 'l', "it.next().?.* == 'l'")
		expect(t, *cbSome(t, it.Next()) == 'o', "it.next().?.* == 'o'")
		expect(t, it.Next() == nil, "it.next() == null")
	}
}

// Zig: test "CircBuf append slice with wrap"
func TestCb1_CircBuf_append_slice_with_wrap(t *testing.T) {
	buf := cbInit(t, 0, 4)
	defer buf.Deinit()

	// Fill the buffer
	_ = buf.GetPtrSlice(0, buf.Capacity())
	expect(t, buf.full, "buf.full")
	expectEqual(t, 4, buf.Len())

	// Delete
	buf.DeleteOldest(2)
	expect(t, !buf.full, "!buf.full")
	expectEqual(t, 2, buf.Len())

	buf.AppendSliceAssumeCapacity([]uint8("AB"))
	{
		it := buf.Iterator(DatastructCircBufIteratorDirectionForward)
		expect(t, *cbSome(t, it.Next()) == 0, "it.next().?.* == 0")
		expect(t, *cbSome(t, it.Next()) == 0, "it.next().?.* == 0")
		expect(t, *cbSome(t, it.Next()) == 'A', "it.next().?.* == 'A'")
		expect(t, *cbSome(t, it.Next()) == 'B', "it.next().?.* == 'B'")
		expect(t, it.Next() == nil, "it.next() == null")
	}
}

// Zig: test "CircBuf getPtrSlice fits"
func TestCb1_CircBuf_getPtrSlice_fits(t *testing.T) {
	buf := cbInit(t, 0, 12)
	defer buf.Deinit()

	slices := buf.GetPtrSlice(0, 11)
	expectEqual(t, 11, len(slices[0]))
	expectEqual(t, 0, len(slices[1]))
	expectEqual(t, 11, buf.Len())
}

// Zig: test "CircBuf getPtrSlice wraps"
func TestCb1_CircBuf_getPtrSlice_wraps(t *testing.T) {
	buf := cbInit(t, 0, 4)
	defer buf.Deinit()

	// Fill the buffer
	_ = buf.GetPtrSlice(0, buf.Capacity())
	expect(t, buf.full, "buf.full")
	expectEqual(t, 4, buf.Len())

	// Delete
	buf.DeleteOldest(2)
	expect(t, !buf.full, "!buf.full")
	expectEqual(t, 2, buf.Len())

	// Get a slice that doesn't grow
	{
		slices := buf.GetPtrSlice(0, 2)
		expectEqual(t, 2, len(slices[0]))
		expectEqual(t, 0, len(slices[1]))
		expectEqual(t, 2, buf.Len())
		slices[0][0] = 1
		slices[0][1] = 2
	}

	// Get a slice that does grow, and forces wrap
	{
		slices := buf.GetPtrSlice(2, 2)
		expectEqual(t, 2, len(slices[0]))
		expectEqual(t, 0, len(slices[1]))
		expectEqual(t, 4, buf.Len())

		// should be empty
		expectEqual(t, 0, slices[0][0])
		expectEqual(t, 0, slices[0][1])
		slices[0][0] = 3
		slices[0][1] = 4
	}

	// Get a slice across boundaries
	{
		slices := buf.GetPtrSlice(0, 4)
		expectEqual(t, 2, len(slices[0]))
		expectEqual(t, 2, len(slices[1]))
		expectEqual(t, 4, buf.Len())

		expectEqual(t, 1, slices[0][0])
		expectEqual(t, 2, slices[0][1])
		expectEqual(t, 3, slices[1][0])
		expectEqual(t, 4, slices[1][1])
	}
}

// Zig: test "CircBuf rotateToZero"
func TestCb1_CircBuf_rotateToZero(t *testing.T) {
	buf := cbInit(t, 0, 12)
	defer buf.Deinit()

	_ = buf.GetPtrSlice(0, 11)
	zigTry(t, buf.rotateToZero())
}

// Zig: test "CircBuf rotateToZero offset"
func TestCb1_CircBuf_rotateToZero_offset(t *testing.T) {
	buf := cbInit(t, 0, 4)
	defer buf.Deinit()

	// Fill the buffer
	_ = buf.GetPtrSlice(0, 3)
	expectEqual(t, 3, buf.Len())

	// Delete
	buf.DeleteOldest(2)
	expect(t, !buf.full, "!buf.full")
	expectEqual(t, 1, buf.Len())
	expect(t, buf.tail > 0 && buf.head >= buf.tail, "buf.tail > 0 and buf.head >= buf.tail")

	// Rotate to zero
	zigTry(t, buf.rotateToZero())
	expectEqual(t, 0, buf.tail)
	expectEqual(t, 1, buf.head)
}

// Zig: test "CircBuf rotateToZero wraps"
func TestCb1_CircBuf_rotateToZero_wraps(t *testing.T) {
	buf := cbInit(t, 0, 4)
	defer buf.Deinit()

	// Fill the buffer
	_ = buf.GetPtrSlice(0, 3)
	expectEqual(t, 3, buf.Len())
	expect(t, buf.tail == 0 && buf.head == 3, "buf.tail == 0 and buf.head == 3")

	// Delete all
	buf.DeleteOldest(3)
	expectEqual(t, 0, buf.Len())
	expect(t, buf.tail == 3 && buf.head == 3, "buf.tail == 3 and buf.head == 3")

	// Refill to force a wrap
	{
		slices := buf.GetPtrSlice(0, 3)
		slices[0][0] = 1
		slices[1][0] = 2
		slices[1][1] = 3
		expectEqual(t, 3, buf.Len())
		expect(t, buf.tail == 3 && buf.head == 2, "buf.tail == 3 and buf.head == 2")
	}

	// Rotate to zero
	zigTry(t, buf.rotateToZero())
	expectEqual(t, 0, buf.tail)
	expectEqual(t, 3, buf.head)
	{
		slices := buf.GetPtrSlice(0, 3)
		expectEqual(t, 1, slices[0][0])
		expectEqual(t, 2, slices[0][1])
		expectEqual(t, 3, slices[0][2])
	}
}

// Zig: test "CircBuf rotateToZero full no wrap"
func TestCb1_CircBuf_rotateToZero_full_no_wrap(t *testing.T) {
	buf := cbInit(t, 0, 4)
	defer buf.Deinit()

	// Fill the buffer
	_ = buf.GetPtrSlice(0, 3)

	// Delete all
	buf.DeleteOldest(3)

	// Refill to force a wrap
	{
		slices := buf.GetPtrSlice(0, 4)
		expect(t, buf.full, "buf.full")
		slices[0][0] = 1
		slices[1][0] = 2
		slices[1][1] = 3
		slices[1][2] = 4
	}

	// Rotate to zero
	zigTry(t, buf.rotateToZero())
	expect(t, buf.full, "buf.full")
	expectEqual(t, 0, buf.tail)
	expectEqual(t, 0, buf.head)
	{
		slices := buf.GetPtrSlice(0, 4)
		expectEqual(t, 1, slices[0][0])
		expectEqual(t, 2, slices[0][1])
		expectEqual(t, 3, slices[0][2])
		expectEqual(t, 4, slices[0][3])
	}
}

// Zig: test "CircBuf resize grow from zero"
func TestCb1_CircBuf_resize_grow_from_zero(t *testing.T) {
	buf := cbInit(t, 0, 0)
	defer buf.Deinit()
	expect(t, buf.full, "buf.full")

	// Resize
	zigTry(t, buf.Resize(2))
	expect(t, !buf.full, "!buf.full")
	expectEqual(t, 0, buf.Len())
	expectEqual(t, 2, buf.Capacity())

	zigTry(t, buf.Append(1))
	zigTry(t, buf.Append(2))

	{
		slices := buf.GetPtrSlice(0, 2)
		expectEqual(t, 1, slices[0][0])
		expectEqual(t, 2, slices[0][1])
	}
}

// Zig: test "CircBuf resize grow"
func TestCb1_CircBuf_resize_grow(t *testing.T) {
	buf := cbInit(t, 0, 4)
	defer buf.Deinit()

	// Fill and write
	{
		slices := buf.GetPtrSlice(0, 4)
		expect(t, buf.full, "buf.full")
		slices[0][0] = 1
		slices[0][1] = 2
		slices[0][2] = 3
		slices[0][3] = 4
	}

	// Resize
	zigTry(t, buf.Resize(6))
	expect(t, !buf.full, "!buf.full")
	expectEqual(t, 4, buf.Len())
	expectEqual(t, 6, buf.Capacity())

	{
		slices := buf.GetPtrSlice(0, 4)
		expectEqual(t, 1, slices[0][0])
		expectEqual(t, 2, slices[0][1])
		expectEqual(t, 3, slices[0][2])
		expectEqual(t, 4, slices[0][3])
	}
}

// Zig: test "CircBuf resize shrink"
func TestCb1_CircBuf_resize_shrink(t *testing.T) {
	buf := cbInit(t, 0, 4)
	defer buf.Deinit()

	// Fill and write
	{
		slices := buf.GetPtrSlice(0, 4)
		expect(t, buf.full, "buf.full")
		slices[0][0] = 1
		slices[0][1] = 2
		slices[0][2] = 3
		slices[0][3] = 4
	}

	// Resize
	zigTry(t, buf.Resize(3))
	expect(t, buf.full, "buf.full")
	expectEqual(t, 3, buf.Len())
	expectEqual(t, 3, buf.Capacity())

	{
		slices := buf.GetPtrSlice(0, 3)
		expectEqual(t, 1, slices[0][0])
		expectEqual(t, 2, slices[0][1])
		expectEqual(t, 3, slices[0][2])
	}
}

// Zig: test "CircBuf append empty slice"
func TestCb1_CircBuf_append_empty_slice(t *testing.T) {
	buf := cbInit(t, 0, 5)
	defer buf.Deinit()

	// Appending an empty slice to empty buffer should be a no-op
	buf.AppendSliceAssumeCapacity([]uint8(""))
	expectEqual(t, 0, buf.Len())
	expect(t, !buf.full, "!buf.full")

	// Buffer should still work normally after appending empty slice
	buf.AppendSliceAssumeCapacity([]uint8("hi"))
	expectEqual(t, 2, buf.Len())

	// Appending an empty slice to non-empty buffer should also be a no-op
	buf.AppendSliceAssumeCapacity([]uint8(""))
	expectEqual(t, 2, buf.Len())
}

// Zig: test "CircBuf getPtrSlice zero length"
func TestCb1_CircBuf_getPtrSlice_zero_length(t *testing.T) {
	buf := cbInit(t, 0, 5)
	defer buf.Deinit()

	// getPtrSlice with zero length on empty buffer should return empty slices
	slices := buf.GetPtrSlice(0, 0)
	expectEqual(t, 0, len(slices[0]))
	expectEqual(t, 0, len(slices[1]))
	expectEqual(t, 0, buf.Len())

	// Fill buffer partially
	buf.AppendSliceAssumeCapacity([]uint8("abc"))
	expectEqual(t, 3, buf.Len())

	// getPtrSlice with zero length on non-empty buffer should also work
	slices2 := buf.GetPtrSlice(0, 0)
	expectEqual(t, 0, len(slices2[0]))
	expectEqual(t, 0, len(slices2[1]))
	expectEqual(t, 3, buf.Len())
}

// Zig: test "CircBuf deleteOldest zero"
func TestCb1_CircBuf_deleteOldest_zero(t *testing.T) {
	buf := cbInit(t, 0, 5)
	defer buf.Deinit()

	// deleteOldest(0) on empty buffer should be a no-op
	buf.DeleteOldest(0)
	expectEqual(t, 0, buf.Len())

	// Fill buffer
	buf.AppendSliceAssumeCapacity([]uint8("hello"))
	expectEqual(t, 5, buf.Len())

	// deleteOldest(0) on non-empty buffer should be a no-op
	buf.DeleteOldest(0)
	expectEqual(t, 5, buf.Len())

	// Verify data is unchanged
	it := buf.Iterator(DatastructCircBufIteratorDirectionForward)
	expectEqual(t, 'h', cbNext(t, &it))
}

// ---------------------------------------------------------------------------
// term51: additional tests (AN). Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

// Zig: test "term51: CircBuf.init: fields and defaults"
func TestCb1_term51_CircBuf_init_fields_and_defaults(t *testing.T) {
	buf := cbInit(t, 0xAA, 5)
	defer buf.Deinit()

	expectEqual(t, 5, len(buf.storage))
	expectEqual(t, 0, buf.head)
	expectEqual(t, 0, buf.tail)
	expect(t, !buf.full, "!buf.full")
	for _, v := range buf.storage {
		expectEqual(t, 0xAA, v)
	}
	expectEqual(t, 5, buf.Capacity())
	expectEqual(t, 0, buf.Len())
	expect(t, buf.Empty(), "buf.empty()")
	expect(t, buf.First() == nil, "buf.first() == null")
	expect(t, buf.Last() == nil, "buf.last() == null")
}

// Zig: test "term51: CircBuf.init: capacity one"
func TestCb1_term51_CircBuf_init_capacity_one(t *testing.T) {
	buf := cbInit(t, 0xAA, 1)
	defer buf.Deinit()

	expectEqual(t, 1, buf.Capacity())
	expect(t, !buf.full, "!buf.full")
	expect(t, buf.Empty(), "buf.empty()")
	expectEqual(t, 0xAA, buf.storage[0])
}

// Zig: test "term51: CircBuf.init: zero capacity is full but has no elements"
func TestCb1_term51_CircBuf_init_zero_capacity_is_full_but_has_no_elements(t *testing.T) {
	buf := cbInit(t, 0, 0)
	defer buf.Deinit()

	expectEqual(t, 0, len(buf.storage))
	expect(t, buf.full, "buf.full")
	expectEqual(t, 0, buf.Capacity())
	expectEqual(t, 0, buf.Len())
	// Pinned: a zero-capacity buffer is "full" so it is not empty().
	expect(t, !buf.Empty(), "!buf.empty()")
	expect(t, buf.First() == nil, "buf.first() == null")
	expect(t, buf.Last() == nil, "buf.last() == null")
	it := buf.Iterator(DatastructCircBufIteratorDirectionForward)
	expect(t, it.Next() == nil, "it.next() == null")
	rit := buf.Iterator(DatastructCircBufIteratorDirectionReverse)
	expect(t, rit.Next() == nil, "rit.next() == null")
}

// "term51: CircBuf.init: allocation failure" is not translated: it uses
// std.testing.FailingAllocator (see scripts/not-translated.txt).

// Zig: test "term51: CircBuf.init: large capacity"
func TestCb1_term51_CircBuf_init_large_capacity(t *testing.T) {
	const n uint = 1 << 20
	buf := cbInit(t, 0xAA, n)
	defer buf.Deinit()

	expectEqual(t, n, buf.Capacity())
	expectEqual(t, 0, buf.Len())
	expect(t, cbAllEqual(buf.storage, 0xAA), "std.mem.allEqual(u8, buf.storage, 0xAA)")
}

// Zig: test "term51: CircBuf.init: element types of different sizes"
func TestCb1_term51_CircBuf_init_element_types_of_different_sizes(t *testing.T) {
	cbTypeCycle[uint8](t, 0x11, 1, 2, 3)
	cbTypeCycle[uint16](t, 0xBEEF, 1, 2, 3)
	cbTypeCycle[uint32](t, 0xDEADBEEF, 1, 2, 3)
	cbTypeCycle[uint64](t, 0xFEEDFACECAFEBEEF, 1, 2, 3)
	// u128: 1 << 100 is bit 36 of the high word.
	cbTypeCycle(t, cbU128{hi: 1 << 36}, cbU128{lo: 1}, cbU128{lo: 2}, cbU128{lo: 3})
	cbTypeCycle[int16](t, -7, 1, -2, 3)
	cbTypeCycle(t, true, false, false, false)
	cbTypeCycle(t, None[uint16](), Some(uint16(1)), None[uint16](), Some(uint16(3)))
	cbTypeCycle(t, [3]uint8{9, 9, 9}, [3]uint8{1, 2, 3}, [3]uint8{4, 5, 6}, [3]uint8{7, 8, 9})
	cbTypeCycle(
		t,
		cbElem{a: 7, b: 0xDEADBEEF, c: 0x1234},
		cbElem{a: 1, b: 2, c: 3},
		cbElem{a: 4, b: 5, c: 6},
		cbElem{a: 7, b: 8, c: 9},
	)
}

// Zig: test "term51: CircBuf.deinit: frees storage"
func TestCb1_term51_CircBuf_deinit_frees_storage(t *testing.T) {
	// Zig-only: leak detection of the testing allocator verifies the free.
	a := cbInit(t, 0, 16)
	a.Deinit()

	// Zero capacity is allocatable and freeable too.
	b := cbInit(t, 0, 0)
	b.Deinit()

	// After a resize the (new) storage is freed.
	c := cbInit(t, 0, 2)
	zigTry(t, c.Resize(64))
	c.Deinit()
}

// Zig: test "term51: CircBuf.append: fills then reports OutOfMemory without overwriting"
func TestCb1_term51_CircBuf_append_fills_then_reports_OutOfMemory_without_overwriting(t *testing.T) {
	buf := cbInit(t, 0, 3)
	defer buf.Deinit()

	zigTry(t, buf.Append(1))
	expectEqual(t, 1, buf.head)
	expect(t, !buf.full, "!buf.full")
	zigTry(t, buf.Append(2))
	expect(t, !buf.full, "!buf.full")
	zigTry(t, buf.Append(3))
	// Head wrapped to the tail position and the buffer is flagged full.
	expectEqual(t, 0, buf.head)
	expectEqual(t, 0, buf.tail)
	expect(t, buf.full, "buf.full")
	expect(t, !buf.Empty(), "!buf.empty()")

	// Full: error, nothing changes (not an overwrite).
	cbExpectOOM(t, buf.Append(4))
	cbExpectOOM(t, buf.Append(5))
	cbExpect(t, &buf, []uint8{1, 2, 3})
	expectEqualSlices(t, []uint8{1, 2, 3}, buf.storage)
	expect(t, buf.full, "buf.full")
}

// Zig: test "term51: CircBuf.append: capacity one"
func TestCb1_term51_CircBuf_append_capacity_one(t *testing.T) {
	buf := cbInit(t, 0, 1)
	defer buf.Deinit()

	zigTry(t, buf.Append(9))
	expect(t, buf.full, "buf.full")
	expectEqual(t, 0, buf.head)
	cbExpectOOM(t, buf.Append(8))
	cbExpect(t, &buf, []uint8{9})
	buf.DeleteOldest(1)
	expect(t, buf.Empty(), "buf.empty()")
	zigTry(t, buf.Append(7))
	cbExpect(t, &buf, []uint8{7})
}

// Zig: test "term51: CircBuf.append: zero capacity always fails"
func TestCb1_term51_CircBuf_append_zero_capacity_always_fails(t *testing.T) {
	buf := cbInit(t, 0, 0)
	defer buf.Deinit()

	cbExpectOOM(t, buf.Append(1))
	expectEqual(t, 0, buf.Len())
}

// Zig: test "term51: CircBuf.append: wrap-around at every head position"
func TestCb1_term51_CircBuf_append_wrap_around_at_every_head_position(t *testing.T) {
	const capacity uint = 5
	for p := uint(0); p < capacity; p++ {
		buf := cbInit(t, 0, capacity)
		defer buf.Deinit()
		cbShift(&buf, p)
		expectEqual(t, p, buf.head)
		expectEqual(t, p, buf.tail)

		for i := uint8(0); uint(i) < capacity; i++ {
			zigTry(t, buf.Append(i+1))
			expectEqual(t, (p+uint(i)+1)%capacity, buf.head)
			expectEqual(t, uint(i)+1, buf.Len())
			expectEqual(t, uint(i)+1 == capacity, buf.full)
			expectEqual(t, i+1, *cbSome(t, buf.Last()))
			expectEqual(t, 1, *cbSome(t, buf.First()))
		}
		expectEqual(t, p, buf.tail)
		cbExpectOOM(t, buf.Append(99))
		cbExpect(t, &buf, []uint8{1, 2, 3, 4, 5})
	}
}

// Zig: test "term51: CircBuf.append: reuses slots freed by deleteOldest"
func TestCb1_term51_CircBuf_append_reuses_slots_freed_by_deleteOldest(t *testing.T) {
	buf := cbInit(t, 0, 3)
	defer buf.Deinit()

	// Stream 20 values through the buffer, keeping at most 3.
	var next uint8 = 1
	zigTry(t, buf.Append(next))
	next += 1
	zigTry(t, buf.Append(next))
	next += 1
	zigTry(t, buf.Append(next))
	next += 1
	for ; next <= 20; next++ {
		cbExpectOOM(t, buf.Append(next))
		buf.DeleteOldest(1)
		zigTry(t, buf.Append(next))
		cbExpect(t, &buf, []uint8{next - 2, next - 1, next})
	}
}

// Zig: test "term51: CircBuf.appendAssumeCapacity: fills and wraps at every head position"
func TestCb1_term51_CircBuf_appendAssumeCapacity_fills_and_wraps_at_every_head_position(t *testing.T) {
	const capacity uint = 5
	for p := uint(0); p < capacity; p++ {
		buf := cbInit(t, 0, capacity)
		defer buf.Deinit()
		cbShift(&buf, p)

		for i := uint8(0); uint(i) < capacity; i++ {
			buf.AppendAssumeCapacity(i + 1)
			expectEqual(t, (p+uint(i)+1)%capacity, buf.head)
			expectEqual(t, uint(i)+1, buf.Len())
			expectEqual(t, uint(i)+1 == capacity, buf.full)
		}
		expectEqual(t, p, buf.tail)
		cbExpect(t, &buf, []uint8{1, 2, 3, 4, 5})
	}
}

// Zig: test "term51: CircBuf.appendAssumeCapacity: capacity one"
func TestCb1_term51_CircBuf_appendAssumeCapacity_capacity_one(t *testing.T) {
	buf := cbInit(t, 0, 1)
	defer buf.Deinit()

	buf.AppendAssumeCapacity(4)
	expect(t, buf.full, "buf.full")
	expectEqual(t, 0, buf.head)
	cbExpect(t, &buf, []uint8{4})
	buf.DeleteOldest(1)
	buf.AppendAssumeCapacity(5)
	cbExpect(t, &buf, []uint8{5})
}

// Zig: test "term51: CircBuf.appendSliceAssumeCapacity: every head position and length"
func TestCb1_term51_CircBuf_appendSliceAssumeCapacity_every_head_position_and_length(t *testing.T) {
	const capacity uint = 7
	data := []uint8{1, 2, 3, 4, 5, 6, 7}
	for p := uint(0); p < capacity; p++ {
		for k := uint(0); k <= capacity; k++ {
			buf := cbInit(t, 0, capacity)
			defer buf.Deinit()
			cbShift(&buf, p)

			buf.AppendSliceAssumeCapacity(data[0:k])
			cbExpect(t, &buf, data[0:k])
			expectEqual(t, k == capacity, buf.full)
			expectEqual(t, (p+k)%capacity, buf.head)
			expectEqual(t, p, buf.tail)
		}
	}
}

// Zig: test "term51: CircBuf.appendSliceAssumeCapacity: two consecutive slices"
func TestCb1_term51_CircBuf_appendSliceAssumeCapacity_two_consecutive_slices(t *testing.T) {
	const capacity uint = 7
	data := []uint8{1, 2, 3, 4, 5, 6, 7}
	for p := uint(0); p < capacity; p++ {
		for a := uint(0); a <= capacity; a++ {
			for b := uint(0); a+b <= capacity; b++ {
				buf := cbInit(t, 0, capacity)
				defer buf.Deinit()
				cbShift(&buf, p)

				buf.AppendSliceAssumeCapacity(data[0:a])
				buf.AppendSliceAssumeCapacity(data[a : a+b])
				cbExpect(t, &buf, data[0:a+b])
				expectEqual(t, a+b == capacity, buf.full)
			}
		}
	}
}

// Zig: test "term51: CircBuf.appendSliceAssumeCapacity: empty slice is a no-op on a full or zero-capacity buffer"
func TestCb1_term51_CircBuf_appendSliceAssumeCapacity_empty_slice_is_a_no_op_on_a_full_or_zero_capacity_buffer(t *testing.T) {
	full := cbInit(t, 0, 3)
	defer full.Deinit()
	full.AppendSliceAssumeCapacity([]uint8{1, 2, 3})
	expect(t, full.full, "full.full")
	full.AppendSliceAssumeCapacity([]uint8{})
	expect(t, full.full, "full.full")
	cbExpect(t, &full, []uint8{1, 2, 3})

	zero := cbInit(t, 0, 0)
	defer zero.Deinit()
	zero.AppendSliceAssumeCapacity([]uint8{})
	expectEqual(t, 0, zero.Len())
	expect(t, zero.full, "zero.full")
}

// Zig: test "term51: CircBuf.appendSliceAssumeCapacity: wider element type wrapping"
func TestCb1_term51_CircBuf_appendSliceAssumeCapacity_wider_element_type_wrapping(t *testing.T) {
	buf, err := DatastructCircBufInit[uint32](0, 4)
	zigTry(t, err)
	defer buf.Deinit()

	buf.AppendSliceAssumeCapacity([]uint32{10, 20, 30})
	buf.DeleteOldest(2)
	// Head is at 3 and one element (30) remains: three more wrap around.
	buf.AppendSliceAssumeCapacity([]uint32{0xAAAA0001, 0xAAAA0002, 0xAAAA0003})
	expect(t, buf.full, "buf.full")
	expectEqualSlices(t, []uint32{0xAAAA0002, 0xAAAA0003, 30, 0xAAAA0001}, buf.storage)
	it := buf.Iterator(DatastructCircBufIteratorDirectionForward)
	expectEqual(t, uint32(30), cbNext(t, &it))
	expectEqual(t, uint32(0xAAAA0001), cbNext(t, &it))
	expectEqual(t, uint32(0xAAAA0002), cbNext(t, &it))
	expectEqual(t, uint32(0xAAAA0003), cbNext(t, &it))
	expect(t, it.Next() == nil, "it.next() == null")
}

// Zig: test "term51: CircBuf.appendSliceAssumeCapacity: large capacity with wrap"
func TestCb1_term51_CircBuf_appendSliceAssumeCapacity_large_capacity_with_wrap(t *testing.T) {
	const n uint = 1 << 20
	const half = n / 2
	buf := cbInit(t, 0, n)
	defer buf.Deinit()

	// Zig-only: const data = try alloc.alloc(u8, n); defer alloc.free(data);
	data := make([]uint8, n)
	for i := range data {
		data[i] = uint8(i * 7)
	}

	buf.AppendSliceAssumeCapacity(data)
	expect(t, buf.full, "buf.full")
	expectEqual(t, n, buf.Len())
	expectEqualSlices(t, data, buf.storage)

	buf.DeleteOldest(half)
	expectEqual(t, half, buf.Len())
	buf.AppendSliceAssumeCapacity(data[0:half])
	expect(t, buf.full, "buf.full")
	expectEqual(t, half, buf.tail)
	expectEqual(t, half, buf.head)

	slices := buf.GetPtrSlice(0, n)
	expectEqual(t, half, uint(len(slices[0])))
	expectEqual(t, half, uint(len(slices[1])))
	expectEqualSlices(t, data[half:], slices[0])
	expectEqualSlices(t, data[0:half], slices[1])
}

// Zig: test "term51: CircBuf.clear: resets indices but not the stored values"
func TestCb1_term51_CircBuf_clear_resets_indices_but_not_the_stored_values(t *testing.T) {
	buf := cbInit(t, 0, 4)
	defer buf.Deinit()

	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3, 4})
	buf.DeleteOldest(1)
	zigTry(t, buf.Append(5))
	expect(t, buf.full, "buf.full")
	expectEqual(t, 1, buf.tail)

	buf.Clear()
	expectEqual(t, 0, buf.head)
	expectEqual(t, 0, buf.tail)
	expect(t, !buf.full, "!buf.full")
	expect(t, buf.Empty(), "buf.empty()")
	expectEqual(t, 0, buf.Len())
	expectEqual(t, 4, buf.Capacity())
	expect(t, buf.First() == nil, "buf.first() == null")
	expect(t, buf.Last() == nil, "buf.last() == null")

	// Pinned: clear does NOT write the default back into the storage.
	expectEqualSlices(t, []uint8{5, 2, 3, 4}, buf.storage)

	// So stale values are exposed when the buffer is re-extended.
	slices := buf.GetPtrSlice(0, 2)
	expectEqualSlices(t, []uint8{5, 2}, slices[0])
	expectEqual(t, 2, buf.Len())
}

// Zig: test "term51: CircBuf.clear: buffer is reusable and clear on empty is a no-op"
func TestCb1_term51_CircBuf_clear_buffer_is_reusable_and_clear_on_empty_is_a_no_op(t *testing.T) {
	buf := cbInit(t, 0, 3)
	defer buf.Deinit()

	buf.Clear()
	expect(t, buf.Empty(), "buf.empty()")
	expectEqual(t, 0, buf.Len())

	buf.AppendSliceAssumeCapacity([]uint8{1, 2})
	buf.Clear()
	zigTry(t, buf.Append(7))
	zigTry(t, buf.Append(8))
	zigTry(t, buf.Append(9))
	cbExpectOOM(t, buf.Append(10))
	cbExpect(t, &buf, []uint8{7, 8, 9})
}

// Zig: test "term51: CircBuf.clear: zero capacity buffer stops being full"
func TestCb1_term51_CircBuf_clear_zero_capacity_buffer_stops_being_full(t *testing.T) {
	buf := cbInit(t, 0, 0)
	defer buf.Deinit()
	expect(t, buf.full, "buf.full")

	buf.Clear()
	// Pinned: clear drops the full flag even though the capacity is zero,
	// so the buffer now reports empty() while having no room at all
	// (an append would index out of bounds).
	expect(t, !buf.full, "!buf.full")
	expect(t, buf.Empty(), "buf.empty()")
	expectEqual(t, 0, buf.Len())
	expectEqual(t, 0, buf.Capacity())
}

// Zig: test "term51: CircBuf.iterator: starts at index zero with the requested direction"
func TestCb1_term51_CircBuf_iterator_starts_at_index_zero_with_the_requested_direction(t *testing.T) {
	buf := cbInit(t, 0, 4)
	defer buf.Deinit()

	fwd := buf.Iterator(DatastructCircBufIteratorDirectionForward)
	expectEqual(t, 0, fwd.idx)
	expectEqual(t, DatastructCircBufIteratorDirectionForward, fwd.direction)
	rev := buf.Iterator(DatastructCircBufIteratorDirectionReverse)
	expectEqual(t, 0, rev.idx)
	expectEqual(t, DatastructCircBufIteratorDirectionReverse, rev.direction)
}

// Zig: test "term51: CircBuf.iterator: snapshot of head/tail but shared storage"
func TestCb1_term51_CircBuf_iterator_snapshot_of_head_tail_but_shared_storage(t *testing.T) {
	buf := cbInit(t, 0, 4)
	defer buf.Deinit()

	buf.AppendSliceAssumeCapacity([]uint8{1, 2})
	it := buf.Iterator(DatastructCircBufIteratorDirectionForward)

	// Appending after creating the iterator is not seen (len is a copy).
	zigTry(t, buf.Append(3))
	expectEqual(t, 1, cbNext(t, &it))
	expectEqual(t, 2, cbNext(t, &it))
	expect(t, it.Next() == nil, "it.next() == null")

	// But writes through the iterator's pointers hit the real storage.
	it2 := buf.Iterator(DatastructCircBufIteratorDirectionForward)
	*cbSome(t, it2.Next()) = 42
	expectEqual(t, 42, buf.storage[0])
	expectEqual(t, 42, *cbSome(t, buf.First()))
}

// Zig: test "term51: CircBuf.Iterator.next: every tail position and length in both directions"
func TestCb1_term51_CircBuf_Iterator_next_every_tail_position_and_length_in_both_directions(t *testing.T) {
	const capacity uint = 5
	data := []uint8{1, 2, 3, 4, 5}
	for p := uint(0); p < capacity; p++ {
		for l := uint(0); l <= capacity; l++ {
			buf := cbInit(t, 0, capacity)
			defer buf.Deinit()
			cbShift(&buf, p)
			buf.AppendSliceAssumeCapacity(data[0:l])
			cbExpect(t, &buf, data[0:l])
		}
	}
}

// Zig: test "term51: CircBuf.Iterator.next: pointers address the storage slots"
func TestCb1_term51_CircBuf_Iterator_next_pointers_address_the_storage_slots(t *testing.T) {
	buf := cbInit(t, 0, 4)
	defer buf.Deinit()
	cbShift(&buf, 3)
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3})
	// Logical order 1,2,3 sits in storage slots 3,0,1.

	it := buf.Iterator(DatastructCircBufIteratorDirectionForward)
	expect(t, it.Next() == &buf.storage[3], "it.next().? == &buf.storage[3]")
	expect(t, it.Next() == &buf.storage[0], "it.next().? == &buf.storage[0]")
	expect(t, it.Next() == &buf.storage[1], "it.next().? == &buf.storage[1]")
	expect(t, it.Next() == nil, "it.next() == null")

	rit := buf.Iterator(DatastructCircBufIteratorDirectionReverse)
	expect(t, rit.Next() == &buf.storage[1], "rit.next().? == &buf.storage[1]")
	expect(t, rit.Next() == &buf.storage[0], "rit.next().? == &buf.storage[0]")
	expect(t, rit.Next() == &buf.storage[3], "rit.next().? == &buf.storage[3]")
	expect(t, rit.Next() == nil, "rit.next() == null")
}

// Zig: test "term51: CircBuf.Iterator.next: stays exhausted and does not advance past the end"
func TestCb1_term51_CircBuf_Iterator_next_stays_exhausted_and_does_not_advance_past_the_end(t *testing.T) {
	buf := cbInit(t, 0, 3)
	defer buf.Deinit()
	buf.AppendSliceAssumeCapacity([]uint8{1, 2})

	it := buf.Iterator(DatastructCircBufIteratorDirectionForward)
	_ = it.Next()
	_ = it.Next()
	expectEqual(t, 2, it.idx)
	expect(t, it.Next() == nil, "it.next() == null")
	expect(t, it.Next() == nil, "it.next() == null")
	// idx is only incremented when a value is returned.
	expectEqual(t, 2, it.idx)
}

// Zig: test "term51: CircBuf.Iterator.next: full buffer exposes capacity elements"
func TestCb1_term51_CircBuf_Iterator_next_full_buffer_exposes_capacity_elements(t *testing.T) {
	buf := cbInit(t, 0, 3)
	defer buf.Deinit()
	cbShift(&buf, 2)
	buf.AppendSliceAssumeCapacity([]uint8{7, 8, 9})
	expect(t, buf.full, "buf.full")
	cbExpect(t, &buf, []uint8{7, 8, 9})
}

// Zig: test "term51: CircBuf.Iterator.seekBy: forward direction"
func TestCb1_term51_CircBuf_Iterator_seekBy_forward_direction(t *testing.T) {
	buf := cbInit(t, 0, 5)
	defer buf.Deinit()
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3, 4, 5})

	it := buf.Iterator(DatastructCircBufIteratorDirectionForward)
	it.SeekBy(0)
	expectEqual(t, 0, it.idx)
	it.SeekBy(2)
	expectEqual(t, 2, it.idx)
	expectEqual(t, 3, cbNext(t, &it))
	// Backwards by 2 from idx 3 lands on idx 1.
	it.SeekBy(-2)
	expectEqual(t, 1, it.idx)
	expectEqual(t, 2, cbNext(t, &it))
	// Seeking to the last element, then to exactly the end.
	it.SeekBy(2)
	expectEqual(t, 5, cbNext(t, &it))
	expect(t, it.Next() == nil, "it.next() == null")
	it.SeekBy(-1)
	expectEqual(t, 5, cbNext(t, &it))
}

// Zig: test "term51: CircBuf.Iterator.seekBy: reverse direction counts from the newest"
func TestCb1_term51_CircBuf_Iterator_seekBy_reverse_direction_counts_from_the_newest(t *testing.T) {
	buf := cbInit(t, 0, 5)
	defer buf.Deinit()
	cbShift(&buf, 3)
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3, 4, 5})

	it := buf.Iterator(DatastructCircBufIteratorDirectionReverse)
	it.SeekBy(2)
	expectEqual(t, 3, cbNext(t, &it))
	it.SeekBy(-2)
	expectEqual(t, 4, cbNext(t, &it))
	it.SeekBy(1)
	expectEqual(t, 2, cbNext(t, &it))
	expectEqual(t, 1, cbNext(t, &it))
	expect(t, it.Next() == nil, "it.next() == null")
}

// Zig: test "term51: CircBuf.Iterator.seekBy: clamps at zero and saturates at the maximum"
func TestCb1_term51_CircBuf_Iterator_seekBy_clamps_at_zero_and_saturates_at_the_maximum(t *testing.T) {
	buf := cbInit(t, 0, 3)
	defer buf.Deinit()
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3})

	// isize's magnitude of minInt is maxInt(isize) + 1 (2^63 in the Zig
	// 64-bit build); it is written platform-independently.
	const isizeMinMagnitude = uint(math.MaxInt) + 1

	it := buf.Iterator(DatastructCircBufIteratorDirectionForward)
	it.SeekBy(-1)
	expectEqual(t, 0, it.idx)
	it.SeekBy(math.MinInt)
	expectEqual(t, 0, it.idx)
	expectEqual(t, 1, cbNext(t, &it))

	it.SeekBy(math.MaxInt)
	expectEqual(t, uint(math.MaxInt)+1, it.idx)
	expect(t, it.Next() == nil, "it.next() == null")
	it.SeekBy(math.MaxInt)
	it.SeekBy(math.MaxInt)
	// 2^63 + 2*(2^63 - 1) overflows usize: saturate.
	expectEqual(t, uint(math.MaxUint), it.idx)
	expect(t, it.Next() == nil, "it.next() == null")
	it.SeekBy(math.MinInt)
	expectEqual(t, uint(math.MaxUint)-isizeMinMagnitude, it.idx)
}

// Zig: test "term51: CircBuf.Iterator.seekBy: does not clamp to the buffer length"
func TestCb1_term51_CircBuf_Iterator_seekBy_does_not_clamp_to_the_buffer_length(t *testing.T) {
	buf := cbInit(t, 0, 3)
	defer buf.Deinit()
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3})

	it := buf.Iterator(DatastructCircBufIteratorDirectionForward)
	// Pinned: despite the doc comment, seeking past the end is kept as is
	// (only the bounds of usize are clamped), so coming back requires
	// seeking the full overshoot.
	it.SeekBy(10)
	expectEqual(t, 10, it.idx)
	expect(t, it.Next() == nil, "it.next() == null")
	it.SeekBy(-3)
	expectEqual(t, 7, it.idx)
	expect(t, it.Next() == nil, "it.next() == null")
	it.SeekBy(-5)
	expectEqual(t, 2, it.idx)
	expectEqual(t, 3, cbNext(t, &it))
}

// Zig: test "term51: CircBuf.Iterator.seekBy: empty and zero-capacity buffers"
func TestCb1_term51_CircBuf_Iterator_seekBy_empty_and_zero_capacity_buffers(t *testing.T) {
	buf := cbInit(t, 0, 0)
	defer buf.Deinit()

	it := buf.Iterator(DatastructCircBufIteratorDirectionForward)
	it.SeekBy(3)
	expect(t, it.Next() == nil, "it.next() == null")
	it.SeekBy(-100)
	expectEqual(t, 0, it.idx)
	expect(t, it.Next() == nil, "it.next() == null")
}
