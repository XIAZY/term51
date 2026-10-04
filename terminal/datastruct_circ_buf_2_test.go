// Tests translated one for one from src/datastruct/circ_buf.zig (Ghostty
// v1.3.1 plus the term51 additional tests), second half: the tests from line
// 1567 of the Zig file to the end, followed by the Go-only panic tests. The
// helpers (cbInit, cbShift, cbExpect, ...) and the translation notes are in
// datastruct_circ_buf_1_test.go.

package terminal

import (
	"testing"
)

// Zig: test "term51: CircBuf.Iterator.reset: returns to the first value"
func TestCb2_term51_CircBuf_Iterator_reset_returns_to_the_first_value(t *testing.T) {
	buf := cbInit(t, 0, 4)
	defer buf.Deinit()
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3})

	it := buf.Iterator(DatastructCircBufIteratorDirectionForward)
	_ = it.Next()
	_ = it.Next()
	it.Reset()
	expectEqual(t, 0, it.idx)
	expectEqual(t, 1, cbNext(t, &it))

	// Reset after exhaustion.
	_ = it.Next()
	_ = it.Next()
	expect(t, it.Next() == nil, "it.next() == null")
	it.Reset()
	expectEqual(t, 1, cbNext(t, &it))

	// Reset after an oversized seek.
	it.SeekBy(100)
	it.Reset()
	expectEqual(t, 1, cbNext(t, &it))

	// Reverse keeps its direction.
	rit := buf.Iterator(DatastructCircBufIteratorDirectionReverse)
	_ = rit.Next()
	rit.Reset()
	expectEqual(t, 3, cbNext(t, &rit))

	// Reset on an iterator that never moved.
	fresh := buf.Iterator(DatastructCircBufIteratorDirectionForward)
	fresh.Reset()
	expectEqual(t, 0, fresh.idx)
}

// Zig: test "term51: CircBuf.first: oldest value at every tail position"
func TestCb2_term51_CircBuf_first_oldest_value_at_every_tail_position(t *testing.T) {
	const capacity uint = 4
	for p := uint(0); p < capacity; p++ {
		buf := cbInit(t, 0, capacity)
		defer buf.Deinit()
		cbShift(&buf, p)
		expect(t, buf.First() == nil, "buf.first() == null")

		buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3, 4})
		expect(t, buf.First() == &buf.storage[p], "buf.first().? == &buf.storage[p]")
		expectEqual(t, 1, *cbSome(t, buf.First()))

		buf.DeleteOldest(1)
		expectEqual(t, 2, *cbSome(t, buf.First()))
		buf.DeleteOldest(2)
		expectEqual(t, 4, *cbSome(t, buf.First()))
		buf.DeleteOldest(1)
		expect(t, buf.First() == nil, "buf.first() == null")
	}
}

// Zig: test "term51: CircBuf.first: pointer allows modifying the value"
func TestCb2_term51_CircBuf_first_pointer_allows_modifying_the_value(t *testing.T) {
	buf := cbInit(t, 0, 3)
	defer buf.Deinit()
	buf.AppendSliceAssumeCapacity([]uint8{1, 2})
	*cbSome(t, buf.First()) = 9
	cbExpect(t, &buf, []uint8{9, 2})
}

// Zig: test "term51: CircBuf.last: newest value at every tail position"
func TestCb2_term51_CircBuf_last_newest_value_at_every_tail_position(t *testing.T) {
	const capacity uint = 4
	for p := uint(0); p < capacity; p++ {
		buf := cbInit(t, 0, capacity)
		defer buf.Deinit()
		cbShift(&buf, p)
		expect(t, buf.Last() == nil, "buf.last() == null")

		for i := uint8(0); uint(i) < capacity; i++ {
			zigTry(t, buf.Append(i+1))
			expectEqual(t, i+1, *cbSome(t, buf.Last()))
			expect(t, buf.Last() == &buf.storage[(p+uint(i))%capacity], "buf.last().? == &buf.storage[(p + i) % cap]")
		}
		// Deleting the oldest never changes the newest.
		buf.DeleteOldest(3)
		expectEqual(t, 4, *cbSome(t, buf.Last()))
		// A single element is both first and last.
		expect(t, buf.First() == buf.Last(), "buf.first().? == buf.last().?")
		buf.DeleteOldest(1)
		expect(t, buf.Last() == nil, "buf.last() == null")
	}
}

// Zig: test "term51: CircBuf.last: pointer allows modifying the value"
func TestCb2_term51_CircBuf_last_pointer_allows_modifying_the_value(t *testing.T) {
	buf := cbInit(t, 0, 3)
	defer buf.Deinit()
	buf.AppendSliceAssumeCapacity([]uint8{1, 2})
	*cbSome(t, buf.Last()) = 9
	cbExpect(t, &buf, []uint8{1, 9})
}

// Zig: test "term51: CircBuf.empty: truth table over head, tail and full"
func TestCb2_term51_CircBuf_empty_truth_table_over_head_tail_and_full(t *testing.T) {
	buf := cbInit(t, 0, 5)
	defer buf.Deinit()

	for h := uint(0); h < 5; h++ {
		for tl := uint(0); tl < 5; tl++ {
			buf.head = h
			buf.tail = tl
			buf.full = false
			expectEqual(t, h == tl, buf.Empty())
			buf.full = true
			expect(t, !buf.Empty(), "!buf.empty()")
		}
	}
}

// Zig: test "term51: CircBuf.empty: follows the buffer through its life"
func TestCb2_term51_CircBuf_empty_follows_the_buffer_through_its_life(t *testing.T) {
	buf := cbInit(t, 0, 2)
	defer buf.Deinit()

	expect(t, buf.Empty(), "buf.empty()")
	zigTry(t, buf.Append(1))
	expect(t, !buf.Empty(), "!buf.empty()")
	zigTry(t, buf.Append(2))
	expect(t, !buf.Empty(), "!buf.empty()")
	buf.DeleteOldest(1)
	expect(t, !buf.Empty(), "!buf.empty()")
	buf.DeleteOldest(1)
	expect(t, buf.Empty(), "buf.empty()")
	zigTry(t, buf.Append(3))
	buf.Clear()
	expect(t, buf.Empty(), "buf.empty()")
}

// Zig: test "term51: CircBuf.capacity: storage length independent of contents"
func TestCb2_term51_CircBuf_capacity_storage_length_independent_of_contents(t *testing.T) {
	buf := cbInit(t, 0, 6)
	defer buf.Deinit()

	expectEqual(t, 6, buf.Capacity())
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3, 4, 5, 6})
	expectEqual(t, 6, buf.Capacity())
	buf.DeleteOldest(6)
	expectEqual(t, 6, buf.Capacity())
	buf.Clear()
	expectEqual(t, 6, buf.Capacity())
	zigTry(t, buf.Resize(9))
	expectEqual(t, 9, buf.Capacity())
	zigTry(t, buf.Resize(0))
	expectEqual(t, 0, buf.Capacity())
}

// Zig: test "term51: CircBuf.len: every head/tail combination"
func TestCb2_term51_CircBuf_len_every_head_tail_combination(t *testing.T) {
	buf := cbInit(t, 0, 5)
	defer buf.Deinit()

	for h := uint(0); h < 5; h++ {
		for tl := uint(0); tl < 5; tl++ {
			buf.head = h
			buf.tail = tl
			buf.full = false
			// head == tail and not full is empty; head < tail wraps.
			expectEqual(t, (h+5-tl)%5, buf.Len())
			buf.full = true
			expectEqual(t, 5, buf.Len())
		}
	}
}

// Zig: test "term51: CircBuf.len: tracks appends, deletes and full state"
func TestCb2_term51_CircBuf_len_tracks_appends_deletes_and_full_state(t *testing.T) {
	buf := cbInit(t, 0, 3)
	defer buf.Deinit()

	expectEqual(t, 0, buf.Len())
	zigTry(t, buf.Append(1))
	expectEqual(t, 1, buf.Len())
	zigTry(t, buf.Append(2))
	zigTry(t, buf.Append(3))
	expectEqual(t, 3, buf.Len())
	buf.DeleteOldest(1)
	expectEqual(t, 2, buf.Len())
	zigTry(t, buf.Append(4))
	expectEqual(t, 3, buf.Len())
	buf.DeleteOldest(3)
	expectEqual(t, 0, buf.Len())

	zero := cbInit(t, 0, 0)
	defer zero.Deinit()
	expectEqual(t, 0, zero.Len())
}

// Zig: test "term51: CircBuf.ensureUnusedCapacity: enough room is a no-op"
func TestCb2_term51_CircBuf_ensureUnusedCapacity_enough_room_is_a_no_op(t *testing.T) {
	buf := cbInit(t, 0, 8)
	defer buf.Deinit()
	cbShift(&buf, 5)
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3})
	// Wrapped: head 0, tail 5.
	expectEqual(t, 0, buf.head)
	expectEqual(t, 5, buf.tail)

	// Amount zero and amounts up to capacity - len do not touch anything.
	zigTry(t, buf.EnsureUnusedCapacity(0))
	zigTry(t, buf.EnsureUnusedCapacity(5))
	expectEqual(t, 8, buf.Capacity())
	// No rotation happened either.
	expectEqual(t, 5, buf.tail)
	cbExpect(t, &buf, []uint8{1, 2, 3})
}

// Zig: test "term51: CircBuf.ensureUnusedCapacity: one past the unused capacity grows to len + amount"
func TestCb2_term51_CircBuf_ensureUnusedCapacity_one_past_the_unused_capacity_grows_to_len_amount(t *testing.T) {
	buf := cbInit(t, 0xAA, 8)
	defer buf.Deinit()
	cbShift(&buf, 5)
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3})

	// Needs 3 + 6 = 9 > 8: exactly 9, no growth factor.
	zigTry(t, buf.EnsureUnusedCapacity(6))
	expectEqual(t, 9, buf.Capacity())
	expectEqual(t, 3, buf.Len())
	expect(t, !buf.full, "!buf.full")
	// The data was rotated to the start and the new slot is a default.
	expectEqual(t, 0, buf.tail)
	expectEqual(t, 3, buf.head)
	expectEqualSlices(t, []uint8{1, 2, 3}, buf.storage[0:3])
	expectEqual(t, 0xAA, buf.storage[8])
	cbExpect(t, &buf, []uint8{1, 2, 3})

	// Now six more can be appended.
	for i := uint8(0); i < 6; i++ {
		zigTry(t, buf.Append(10+i))
	}
	expect(t, buf.full, "buf.full")
	cbExpect(t, &buf, []uint8{1, 2, 3, 10, 11, 12, 13, 14, 15})
}

// Zig: test "term51: CircBuf.ensureUnusedCapacity: full buffer, zero capacity and failure"
func TestCb2_term51_CircBuf_ensureUnusedCapacity_full_buffer_zero_capacity_and_failure(t *testing.T) {
	full := cbInit(t, 0, 2)
	defer full.Deinit()
	full.AppendSliceAssumeCapacity([]uint8{1, 2})
	// Amount 0 on a full buffer: len + 0 <= capacity, nothing happens.
	zigTry(t, full.EnsureUnusedCapacity(0))
	expectEqual(t, 2, full.Capacity())
	expect(t, full.full, "full.full")
	zigTry(t, full.EnsureUnusedCapacity(1))
	expectEqual(t, 3, full.Capacity())
	expect(t, !full.full, "!full.full")
	cbExpect(t, &full, []uint8{1, 2})

	zero := cbInit(t, 0, 0)
	defer zero.Deinit()
	zigTry(t, zero.EnsureUnusedCapacity(0))
	expectEqual(t, 0, zero.Capacity())
	zigTry(t, zero.EnsureUnusedCapacity(4))
	expectEqual(t, 4, zero.Capacity())
	expect(t, !zero.full, "!zero.full")
	expect(t, zero.Empty(), "zero.empty()")

	// Zig-only: "Failure: the error propagates and the capacity is unchanged."
	// Zig-only: var failing = std.testing.FailingAllocator.init(alloc, .{ .fail_index = 1, .resize_fail_index = 0 });
	// Zig-only: var small = try term51ANBuf.init(failing.allocator(), 2); defer small.deinit(failing.allocator());
	// Zig-only: small.appendSliceAssumeCapacity(&.{ 1, 2 });
	// Zig-only: try std.testing.expectError(error.OutOfMemory, small.ensureUnusedCapacity(failing.allocator(), 1));
	// Zig-only: try std.testing.expectEqual(@as(usize, 2), small.capacity());
	// Zig-only: try term51ANExpect(&small, &.{ 1, 2 });
}

// Zig: test "term51: CircBuf.resize: grow rotates to zero and fills new slots with the default"
func TestCb2_term51_CircBuf_resize_grow_rotates_to_zero_and_fills_new_slots_with_the_default(t *testing.T) {
	buf := cbInit(t, 0xAA, 5)
	defer buf.Deinit()
	cbShift(&buf, 3)
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3, 4})
	// Not full, wrapped: head 2, tail 3.
	expectEqual(t, 2, buf.head)
	expectEqual(t, 3, buf.tail)

	zigTry(t, buf.Resize(8))
	expectEqual(t, 8, buf.Capacity())
	expectEqual(t, 0, buf.tail)
	expectEqual(t, 4, buf.head)
	expect(t, !buf.full, "!buf.full")
	expectEqualSlices(t, []uint8{1, 2, 3, 4, 0xAA, 0xAA, 0xAA, 0xAA}, buf.storage)
	cbExpect(t, &buf, []uint8{1, 2, 3, 4})
}

// Zig: test "term51: CircBuf.resize: grow a wrapped full buffer"
func TestCb2_term51_CircBuf_resize_grow_a_wrapped_full_buffer(t *testing.T) {
	buf := cbInit(t, 0xAA, 5)
	defer buf.Deinit()
	cbShift(&buf, 3)
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3, 4, 5})
	expect(t, buf.full, "buf.full")
	expectEqual(t, 3, buf.head)
	expectEqual(t, 3, buf.tail)

	zigTry(t, buf.Resize(7))
	expect(t, !buf.full, "!buf.full")
	expectEqual(t, 0, buf.tail)
	// Head points just past the old data.
	expectEqual(t, 5, buf.head)
	expectEqualSlices(t, []uint8{1, 2, 3, 4, 5, 0xAA, 0xAA}, buf.storage)
	cbExpect(t, &buf, []uint8{1, 2, 3, 4, 5})

	// And the new room is usable, including wrap detection at the new end.
	zigTry(t, buf.Append(6))
	zigTry(t, buf.Append(7))
	expect(t, buf.full, "buf.full")
	expectEqual(t, 0, buf.head)
	cbExpectOOM(t, buf.Append(8))
	cbExpect(t, &buf, []uint8{1, 2, 3, 4, 5, 6, 7})
}

// Zig: test "term51: CircBuf.resize: grow by one from every tail position when full"
func TestCb2_term51_CircBuf_resize_grow_by_one_from_every_tail_position_when_full(t *testing.T) {
	const capacity uint = 4
	for p := uint(0); p < capacity; p++ {
		buf := cbInit(t, 0xAA, capacity)
		defer buf.Deinit()
		cbShift(&buf, p)
		buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3, 4})

		zigTry(t, buf.Resize(capacity+1))
		expect(t, !buf.full, "!buf.full")
		expectEqual(t, 0, buf.tail)
		expectEqual(t, capacity, buf.head)
		expectEqual(t, 0xAA, buf.storage[capacity])
		cbExpect(t, &buf, []uint8{1, 2, 3, 4})
	}
}

// Zig: test "term51: CircBuf.resize: grow an empty buffer with moved indices"
func TestCb2_term51_CircBuf_resize_grow_an_empty_buffer_with_moved_indices(t *testing.T) {
	buf := cbInit(t, 0, 4)
	defer buf.Deinit()
	cbShift(&buf, 3)
	expectEqual(t, 3, buf.tail)

	zigTry(t, buf.Resize(6))
	expectEqual(t, 0, buf.tail)
	expectEqual(t, 0, buf.head)
	expect(t, buf.Empty(), "buf.empty()")
	expectEqual(t, 6, buf.Capacity())
}

// Zig: test "term51: CircBuf.resize: grow only resets slots beyond the previous capacity"
func TestCb2_term51_CircBuf_resize_grow_only_resets_slots_beyond_the_previous_capacity(t *testing.T) {
	buf := cbInit(t, 0xAA, 4)
	defer buf.Deinit()
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3, 4})
	buf.Clear()
	buf.AppendSliceAssumeCapacity([]uint8{9})

	zigTry(t, buf.Resize(6))
	// Pinned: stale values left behind by clear() survive inside the old
	// capacity, only the newly allocated slots are set to the default.
	expectEqualSlices(t, []uint8{9, 2, 3, 4, 0xAA, 0xAA}, buf.storage)
	cbExpect(t, &buf, []uint8{9})
}

// Zig: test "term51: CircBuf.resize: same size still rotates and keeps full"
func TestCb2_term51_CircBuf_resize_same_size_still_rotates_and_keeps_full(t *testing.T) {
	full := cbInit(t, 0, 4)
	defer full.Deinit()
	cbShift(&full, 2)
	full.AppendSliceAssumeCapacity([]uint8{1, 2, 3, 4})
	zigTry(t, full.Resize(4))
	expect(t, full.full, "full.full")
	expectEqual(t, 0, full.tail)
	expectEqual(t, 0, full.head)
	expectEqualSlices(t, []uint8{1, 2, 3, 4}, full.storage)

	part := cbInit(t, 0, 4)
	defer part.Deinit()
	cbShift(&part, 3)
	part.AppendSliceAssumeCapacity([]uint8{1, 2})
	zigTry(t, part.Resize(4))
	expect(t, !part.full, "!part.full")
	expectEqual(t, 0, part.tail)
	expectEqual(t, 2, part.head)
	cbExpect(t, &part, []uint8{1, 2})
}

// Zig: test "term51: CircBuf.resize: shrink a full buffer keeps the oldest elements"
func TestCb2_term51_CircBuf_resize_shrink_a_full_buffer_keeps_the_oldest_elements(t *testing.T) {
	buf := cbInit(t, 0, 5)
	defer buf.Deinit()
	cbShift(&buf, 3)
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3, 4, 5})

	zigTry(t, buf.Resize(3))
	expect(t, buf.full, "buf.full")
	expectEqual(t, 3, buf.Capacity())
	expectEqual(t, 0, buf.tail)
	expectEqual(t, 0, buf.head)
	cbExpect(t, &buf, []uint8{1, 2, 3})
}

// Zig: test "term51: CircBuf.resize: shrink a full buffer to zero"
func TestCb2_term51_CircBuf_resize_shrink_a_full_buffer_to_zero(t *testing.T) {
	buf := cbInit(t, 0, 3)
	defer buf.Deinit()
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3})

	zigTry(t, buf.Resize(0))
	expectEqual(t, 0, buf.Capacity())
	expect(t, buf.full, "buf.full")
	expectEqual(t, 0, buf.Len())
	expect(t, buf.First() == nil, "buf.first() == null")
	cbExpectOOM(t, buf.Append(1))

	// And grows back out of it.
	zigTry(t, buf.Resize(2))
	expect(t, !buf.full, "!buf.full")
	zigTry(t, buf.Append(7))
	cbExpect(t, &buf, []uint8{7})
}

// Zig: test "term51: CircBuf.resize: shrink a partial buffer to a larger or equal-to-len size"
func TestCb2_term51_CircBuf_resize_shrink_a_partial_buffer_to_a_larger_or_equal_to_len_size(t *testing.T) {
	// New size larger than len: well formed.
	a := cbInit(t, 0, 6)
	defer a.Deinit()
	cbShift(&a, 4)
	a.AppendSliceAssumeCapacity([]uint8{1, 2})
	zigTry(t, a.Resize(3))
	expectEqual(t, 3, a.Capacity())
	expectEqual(t, 0, a.tail)
	expectEqual(t, 2, a.head)
	expect(t, !a.full, "!a.full")
	cbExpect(t, &a, []uint8{1, 2})
	zigTry(t, a.Append(3))
	expect(t, a.full, "a.full")

	// New size exactly equal to len.
	b := cbInit(t, 0, 6)
	defer b.Deinit()
	b.AppendSliceAssumeCapacity([]uint8{1, 2})
	zigTry(t, b.Resize(2))
	expectEqual(t, 2, b.Capacity())
	expectEqual(t, 2, b.Len())
	// Pinned: the buffer is NOT flagged full and head equals the capacity
	// (one past the last slot); an append would index out of bounds.
	expect(t, !b.full, "!b.full")
	expectEqual(t, 2, b.head)
	expectEqual(t, 0, b.tail)
	cbExpect(t, &b, []uint8{1, 2})
}

// Zig: test "term51: CircBuf.resize: shrink a partial buffer below its length is corrupt"
func TestCb2_term51_CircBuf_resize_shrink_a_partial_buffer_below_its_length_is_corrupt(t *testing.T) {
	buf := cbInit(t, 0, 6)
	defer buf.Deinit()
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3, 4})

	zigTry(t, buf.Resize(2))
	expectEqual(t, 2, buf.Capacity())
	// Pinned: head (4) is left beyond the new capacity (2) and the buffer
	// is not full, so len() reports more elements than fit.
	expectEqual(t, 4, buf.head)
	expectEqual(t, 0, buf.tail)
	expect(t, !buf.full, "!buf.full")
	expectEqual(t, 4, buf.Len())
	// The iterator wraps modulo the capacity and so repeats the data.
	cbExpectIter(t, &buf, []uint8{1, 2, 1, 2})
}

// Zig: test "term51: CircBuf.resize: shrink an empty buffer to zero"
func TestCb2_term51_CircBuf_resize_shrink_an_empty_buffer_to_zero(t *testing.T) {
	buf := cbInit(t, 0, 3)
	defer buf.Deinit()

	zigTry(t, buf.Resize(0))
	expectEqual(t, 0, buf.Capacity())
	expectEqual(t, 0, buf.Len())
	// Pinned: zero capacity but not flagged full (init would flag it).
	expect(t, !buf.full, "!buf.full")
	expect(t, buf.Empty(), "buf.empty()")
}

// "term51: CircBuf.resize: allocation failure leaves capacity and data but
// rotates" is not translated: it uses std.testing.FailingAllocator (see
// scripts/not-translated-cb.txt).

// Zig: test "term51: CircBuf.rotateToZero: tail already zero changes nothing"
func TestCb2_term51_CircBuf_rotateToZero_tail_already_zero_changes_nothing(t *testing.T) {
	buf := cbInit(t, 0, 5)
	defer buf.Deinit()
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3})

	zigTry(t, buf.rotateToZero())
	expectEqual(t, 0, buf.tail)
	expectEqual(t, 3, buf.head)
	expectEqualSlices(t, []uint8{1, 2, 3, 0, 0}, buf.storage)

	// Empty with tail zero and full with tail zero.
	buf.Clear()
	zigTry(t, buf.rotateToZero())
	expect(t, buf.Empty(), "buf.empty()")
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3, 4, 5})
	zigTry(t, buf.rotateToZero())
	expect(t, buf.full, "buf.full")
	expectEqualSlices(t, []uint8{1, 2, 3, 4, 5}, buf.storage)
}

// Zig: test "term51: CircBuf.rotateToZero: every tail position and length"
func TestCb2_term51_CircBuf_rotateToZero_every_tail_position_and_length(t *testing.T) {
	const capacity uint = 5
	data := []uint8{1, 2, 3, 4, 5}
	for p := uint(0); p < capacity; p++ {
		for l := uint(0); l <= capacity; l++ {
			buf := cbInit(t, 0, capacity)
			defer buf.Deinit()
			cbShift(&buf, p)
			buf.AppendSliceAssumeCapacity(data[0:l])

			zigTry(t, buf.rotateToZero())
			expectEqual(t, 0, buf.tail)
			expectEqual(t, l%capacity, buf.head)
			expectEqual(t, l == capacity, buf.full)
			expectEqualSlices(t, data[0:l], buf.storage[0:l])
			expect(t, cbAllEqual(buf.storage[l:], 0), "std.mem.allEqual(u8, buf.storage[l..], 0)")
			cbExpect(t, &buf, data[0:l])
		}
	}
}

// Zig: test "term51: CircBuf.rotateToZero: empty buffer with moved indices resets them"
func TestCb2_term51_CircBuf_rotateToZero_empty_buffer_with_moved_indices_resets_them(t *testing.T) {
	buf := cbInit(t, 0, 4)
	defer buf.Deinit()
	cbShift(&buf, 2)
	expectEqual(t, 2, buf.head)

	zigTry(t, buf.rotateToZero())
	expectEqual(t, 0, buf.head)
	expectEqual(t, 0, buf.tail)
	expect(t, buf.Empty(), "buf.empty()")
}

// Zig: test "term51: CircBuf.rotateToZero: zero capacity"
func TestCb2_term51_CircBuf_rotateToZero_zero_capacity(t *testing.T) {
	buf := cbInit(t, 0, 0)
	defer buf.Deinit()
	zigTry(t, buf.rotateToZero())
	expect(t, buf.full, "buf.full")
	expectEqual(t, 0, buf.tail)
}

// Zig: test "term51: CircBuf.deleteOldest: every tail position, length and count"
func TestCb2_term51_CircBuf_deleteOldest_every_tail_position_length_and_count(t *testing.T) {
	const capacity uint = 5
	data := []uint8{1, 2, 3, 4, 5}
	for p := uint(0); p < capacity; p++ {
		for l := uint(0); l <= capacity; l++ {
			for n := uint(0); n <= capacity; n++ {
				buf := cbInit(t, 0xAA, capacity)
				defer buf.Deinit()
				cbShift(&buf, p)
				buf.AppendSliceAssumeCapacity(data[0:l])

				buf.DeleteOldest(n)

				kept := uint(0)
				if n < l {
					kept = l - n
				}
				cbExpect(t, &buf, data[min(n, l):l])
				// Only a zero count leaves a full buffer full.
				expectEqual(t, n == 0 && l == capacity, buf.full)
				expectEqual(t, kept == 0, buf.Empty())
				// The tail advances by n (also when n exceeds the length).
				expectEqual(t, (p+n)%capacity, buf.tail)
				var wantHead uint
				if n > l {
					wantHead = (p + n) % capacity
				} else {
					wantHead = (p + l) % capacity
				}
				expectEqual(t, wantHead, buf.head)
				// Deleted slots are reset to the default.
				for i := uint(0); i < n; i++ {
					expectEqual(t, 0xAA, buf.storage[(p+i)%capacity])
				}
			}
		}
	}
}

// Zig: test "term51: CircBuf.deleteOldest: zero is a no-op even on a full buffer"
func TestCb2_term51_CircBuf_deleteOldest_zero_is_a_no_op_even_on_a_full_buffer(t *testing.T) {
	buf := cbInit(t, 0, 3)
	defer buf.Deinit()
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3})
	buf.DeleteOldest(0)
	expect(t, buf.full, "buf.full")
	cbExpect(t, &buf, []uint8{1, 2, 3})
	expectEqualSlices(t, []uint8{1, 2, 3}, buf.storage)
}

// Zig: test "term51: CircBuf.deleteOldest: count larger than the length deletes everything"
func TestCb2_term51_CircBuf_deleteOldest_count_larger_than_the_length_deletes_everything(t *testing.T) {
	buf := cbInit(t, 0xAA, 6)
	defer buf.Deinit()
	cbShift(&buf, 4)
	buf.AppendSliceAssumeCapacity([]uint8{1, 2})

	// 5 > len 2 (and head wraps from 0 through 3).
	buf.DeleteOldest(5)
	expect(t, buf.Empty(), "buf.empty()")
	expectEqual(t, 0, buf.Len())
	expectEqual(t, 3, buf.tail)
	expectEqual(t, 3, buf.head)
	expect(t, cbAllEqual(buf.storage[0:3], 0xAA), "std.mem.allEqual(u8, buf.storage[0..3], 0xAA)")
	// Slots outside the deleted range are not touched but were default.
	expect(t, cbAllEqual(buf.storage, 0xAA), "std.mem.allEqual(u8, buf.storage, 0xAA)")
}

// Zig: test "term51: CircBuf.deleteOldest: whole capacity on a full buffer"
func TestCb2_term51_CircBuf_deleteOldest_whole_capacity_on_a_full_buffer(t *testing.T) {
	buf := cbInit(t, 0xAA, 4)
	defer buf.Deinit()
	cbShift(&buf, 1)
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3, 4})
	expect(t, buf.full, "buf.full")

	buf.DeleteOldest(4)
	expect(t, !buf.full, "!buf.full")
	expect(t, buf.Empty(), "buf.empty()")
	expectEqual(t, 1, buf.tail)
	expectEqual(t, 1, buf.head)
	expect(t, cbAllEqual(buf.storage, 0xAA), "std.mem.allEqual(u8, buf.storage, 0xAA)")
}

// Zig: test "term51: CircBuf.deleteOldest: oversized count on an empty buffer leaves it empty"
func TestCb2_term51_CircBuf_deleteOldest_oversized_count_on_an_empty_buffer_leaves_it_empty(t *testing.T) {
	buf := cbInit(t, 0, 4)
	defer buf.Deinit()

	buf.DeleteOldest(4)
	expect(t, buf.Empty(), "buf.empty()")
	expectEqual(t, 0, buf.Len())
	// Pinned: it still moves the tail (and head) by n.
	expectEqual(t, 0, buf.tail)
	buf.DeleteOldest(3)
	expectEqual(t, 3, buf.tail)
	expectEqual(t, 3, buf.head)
	expect(t, buf.Empty(), "buf.empty()")
}

// Zig: test "term51: CircBuf.deleteOldest: zero capacity with count zero"
func TestCb2_term51_CircBuf_deleteOldest_zero_capacity_with_count_zero(t *testing.T) {
	buf := cbInit(t, 0, 0)
	defer buf.Deinit()
	buf.DeleteOldest(0)
	expect(t, buf.full, "buf.full")
	expectEqual(t, 0, buf.Len())
}

// Zig: test "term51: CircBuf.deleteOldest: deleted slots are available for appending again"
func TestCb2_term51_CircBuf_deleteOldest_deleted_slots_are_available_for_appending_again(t *testing.T) {
	buf := cbInit(t, 0, 3)
	defer buf.Deinit()
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3})
	buf.DeleteOldest(2)
	zigTry(t, buf.Append(4))
	zigTry(t, buf.Append(5))
	cbExpectOOM(t, buf.Append(6))
	cbExpect(t, &buf, []uint8{3, 4, 5})
}

// Zig: test "term51: CircBuf.getPtrSlice: zero length returns two empty slices for any offset"
func TestCb2_term51_CircBuf_getPtrSlice_zero_length_returns_two_empty_slices_for_any_offset(t *testing.T) {
	buf := cbInit(t, 0, 4)
	defer buf.Deinit()
	buf.AppendSliceAssumeCapacity([]uint8{1, 2})

	offsets := []uint{0, 1, 2, 3, 4, 5, 1000}
	for _, off := range offsets {
		s := buf.GetPtrSlice(off, 0)
		expectEqual(t, 0, len(s[0]))
		expectEqual(t, 0, len(s[1]))
		// Never extends the buffer.
		expectEqual(t, 2, buf.Len())
	}

	// Also on a zero-capacity buffer.
	zero := cbInit(t, 0, 0)
	defer zero.Deinit()
	z := zero.GetPtrSlice(0, 0)
	expectEqual(t, 0, len(z[0]))
	expectEqual(t, 0, len(z[1]))
}

// Zig: test "term51: CircBuf.getPtrSlice: every tail, length, offset and slice length"
func TestCb2_term51_CircBuf_getPtrSlice_every_tail_length_offset_and_slice_length(t *testing.T) {
	const capacity uint = 6
	for p := uint(0); p < capacity; p++ {
		for l := uint(0); l <= capacity; l++ {
			for o := uint(0); o <= capacity; o++ {
				for n := uint(1); o+n <= capacity; n++ {
					buf := cbInit(t, 0, capacity)
					defer buf.Deinit()
					cbShift(&buf, p)
					for i := uint8(0); uint(i) < l; i++ {
						buf.AppendAssumeCapacity(10 + i)
					}

					s := buf.GetPtrSlice(o, n)

					// Slice geometry.
					start := (p + o) % capacity
					firstLen := min(n, capacity-start)
					expectEqual(t, firstLen, uint(len(s[0])))
					expectEqual(t, n-firstLen, uint(len(s[1])))
					// s[0].ptr == buf.storage.ptr + start (s[0] is never empty here).
					expect(t, &s[0][0] == &buf.storage[start], "s[0].ptr == buf.storage.ptr + start")
					// s[1].ptr == buf.storage.ptr
					if len(s[1]) > 0 {
						expect(t, &s[1][0] == &buf.storage[0], "s[1].ptr == buf.storage.ptr")
					}
					// Zig-only: s[1].ptr == buf.storage.ptr when s[1] is empty (the pointer of an empty slice is not observable in Go)

					// Length grows to cover the slice, never shrinks.
					newLen := max(l, o+n)
					expectEqual(t, newLen, buf.Len())
					expectEqual(t, newLen == capacity, buf.full)
					expectEqual(t, p, buf.tail)
					expectEqual(t, (p+newLen)%capacity, buf.head)

					// Writing through the slices lands at the logical offsets.
					var v uint8 = 100
					for _, sl := range s {
						for j := range sl {
							sl[j] = v
							v++
						}
					}
					it := buf.Iterator(DatastructCircBufIteratorDirectionForward)
					it.SeekBy(int(o))
					for k := uint8(0); uint(k) < n; k++ {
						expectEqual(t, 100+k, cbNext(t, &it))
					}
				}
			}
		}
	}
}

// Zig: test "term51: CircBuf.getPtrSlice: covering the whole buffer from every tail position"
func TestCb2_term51_CircBuf_getPtrSlice_covering_the_whole_buffer_from_every_tail_position(t *testing.T) {
	const capacity uint = 4
	for p := uint(0); p < capacity; p++ {
		buf := cbInit(t, 0, capacity)
		defer buf.Deinit()
		cbShift(&buf, p)

		s := buf.GetPtrSlice(0, capacity)
		expect(t, buf.full, "buf.full")
		expectEqual(t, capacity-p, uint(len(s[0])))
		expectEqual(t, p, uint(len(s[1])))
		expectEqual(t, capacity, buf.Len())
	}
}

// Zig: test "term51: CircBuf.getPtrSlice: a slice ending exactly at the storage end is not split"
func TestCb2_term51_CircBuf_getPtrSlice_a_slice_ending_exactly_at_the_storage_end_is_not_split(t *testing.T) {
	buf := cbInit(t, 0, 5)
	defer buf.Deinit()
	cbShift(&buf, 2)

	s := buf.GetPtrSlice(0, 3)
	expectEqual(t, 3, len(s[0]))
	expectEqual(t, 0, len(s[1]))
	expect(t, &s[0][0] == &buf.storage[2], "s[0].ptr == buf.storage.ptr + 2")
	expectEqual(t, 3, buf.Len())
	// head wrapped to 0 but the buffer is not full (tail is 2).
	expectEqual(t, 0, buf.head)
	expect(t, !buf.full, "!buf.full")
}

// Zig: test "term51: CircBuf.getPtrSlice: an offset beyond the length exposes the gap"
func TestCb2_term51_CircBuf_getPtrSlice_an_offset_beyond_the_length_exposes_the_gap(t *testing.T) {
	buf := cbInit(t, 0xAA, 6)
	defer buf.Deinit()
	buf.AppendSliceAssumeCapacity([]uint8{1, 2})

	// Pinned: a "get" beyond len() extends the buffer, covering the gap
	// (with whatever the storage held: the default here).
	s := buf.GetPtrSlice(4, 2)
	expectEqual(t, 2, len(s[0]))
	expectEqual(t, 6, buf.Len())
	expect(t, buf.full, "buf.full")
	cbExpect(t, &buf, []uint8{1, 2, 0xAA, 0xAA, 0xAA, 0xAA})
}

// Zig: test "term51: CircBuf.getPtrSlice: slice within the length does not modify the buffer"
func TestCb2_term51_CircBuf_getPtrSlice_slice_within_the_length_does_not_modify_the_buffer(t *testing.T) {
	buf := cbInit(t, 0, 5)
	defer buf.Deinit()
	cbShift(&buf, 3)
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3, 4})
	head := buf.head
	tail := buf.tail

	// Logical 0..2 is storage 3..5: one slice.
	s := buf.GetPtrSlice(0, 2)
	expectEqualSlices(t, []uint8{1, 2}, s[0])
	expectEqual(t, 0, len(s[1]))

	// Logical 1..3 is storage 4 and 0: split at the wrap point.
	m := buf.GetPtrSlice(1, 2)
	expectEqualSlices(t, []uint8{2}, m[0])
	expectEqualSlices(t, []uint8{3}, m[1])

	// The whole content.
	w := buf.GetPtrSlice(0, 4)
	expectEqualSlices(t, []uint8{1, 2}, w[0])
	expectEqualSlices(t, []uint8{3, 4}, w[1])

	expectEqual(t, head, buf.head)
	expectEqual(t, tail, buf.tail)
	expectEqual(t, 4, buf.Len())
}

// Zig: test "term51: CircBuf.getPtrSlice: capacity one"
func TestCb2_term51_CircBuf_getPtrSlice_capacity_one(t *testing.T) {
	buf := cbInit(t, 0, 1)
	defer buf.Deinit()

	s := buf.GetPtrSlice(0, 1)
	expectEqual(t, 1, len(s[0]))
	expectEqual(t, 0, len(s[1]))
	expect(t, buf.full, "buf.full")
	s[0][0] = 5
	cbExpect(t, &buf, []uint8{5})
}

// Zig: test "term51: CircBuf.advance: moves the head and sets full when it reaches the tail"
func TestCb2_term51_CircBuf_advance_moves_the_head_and_sets_full_when_it_reaches_the_tail(t *testing.T) {
	buf := cbInit(t, 0, 5)
	defer buf.Deinit()

	buf.advance(2)
	expectEqual(t, 2, buf.head)
	expectEqual(t, 0, buf.tail)
	expect(t, !buf.full, "!buf.full")
	expectEqual(t, 2, buf.Len())

	buf.advance(2)
	expectEqual(t, 4, buf.head)
	expect(t, !buf.full, "!buf.full")

	// One past: head reaches capacity and wraps onto the tail -> full.
	buf.advance(1)
	expectEqual(t, 0, buf.head)
	expectEqual(t, 0, buf.tail)
	expect(t, buf.full, "buf.full")
	expectEqual(t, 5, buf.Len())
}

// Zig: test "term51: CircBuf.advance: wraps around the end of the storage"
func TestCb2_term51_CircBuf_advance_wraps_around_the_end_of_the_storage(t *testing.T) {
	buf := cbInit(t, 0, 5)
	defer buf.Deinit()
	cbShift(&buf, 3)

	buf.advance(1)
	expectEqual(t, 4, buf.head)
	expect(t, !buf.full, "!buf.full")
	// 4 + 3 = 7 wraps to 2 which is before the tail (3): not full.
	buf.advance(3)
	expectEqual(t, 2, buf.head)
	expectEqual(t, 3, buf.tail)
	expect(t, !buf.full, "!buf.full")
	expectEqual(t, 4, buf.Len())
	// The last free slot fills it exactly (head == tail == 3).
	buf.advance(1)
	expectEqual(t, 3, buf.head)
	expect(t, buf.full, "buf.full")
	expectEqual(t, 5, buf.Len())
}

// Zig: test "term51: CircBuf.advance: the whole capacity from an empty buffer"
func TestCb2_term51_CircBuf_advance_the_whole_capacity_from_an_empty_buffer(t *testing.T) {
	const capacity uint = 4
	for p := uint(0); p < capacity; p++ {
		buf := cbInit(t, 0, capacity)
		defer buf.Deinit()
		cbShift(&buf, p)

		buf.advance(capacity)
		expectEqual(t, p, buf.head)
		expectEqual(t, p, buf.tail)
		expect(t, buf.full, "buf.full")
		expectEqual(t, capacity, buf.Len())
	}
}

// Zig: test "term51: CircBuf.advance: zero amount"
func TestCb2_term51_CircBuf_advance_zero_amount(t *testing.T) {
	buf := cbInit(t, 0, 4)
	defer buf.Deinit()

	// Pinned: advancing by zero an EMPTY buffer flags it full because
	// head == tail. getPtrSlice never calls it with zero.
	buf.advance(0)
	expect(t, buf.full, "buf.full")
	expectEqual(t, 4, buf.Len())

	// A non-empty, non-full buffer is unchanged.
	b2 := cbInit(t, 0, 4)
	defer b2.Deinit()
	b2.AppendSliceAssumeCapacity([]uint8{1, 2})
	b2.advance(0)
	expect(t, !b2.full, "!b2.full")
	expectEqual(t, 2, b2.head)
	expectEqual(t, 2, b2.Len())

	// Full buffer: stays full and keeps tail lined up with head.
	b3 := cbInit(t, 0, 4)
	defer b3.Deinit()
	cbShift(&b3, 1)
	b3.AppendSliceAssumeCapacity([]uint8{1, 2, 3, 4})
	b3.advance(0)
	expect(t, b3.full, "b3.full")
	expectEqual(t, 1, b3.head)
	expectEqual(t, 1, b3.tail)
}

// Zig: test "term51: CircBuf.advance: zero capacity with zero amount"
func TestCb2_term51_CircBuf_advance_zero_capacity_with_zero_amount(t *testing.T) {
	buf := cbInit(t, 0, 0)
	defer buf.Deinit()
	buf.advance(0)
	expect(t, buf.full, "buf.full")
	expectEqual(t, 0, buf.head)
	expectEqual(t, 0, buf.tail)
}

// Zig: test "term51: CircBuf.storageOffset: every tail and offset"
func TestCb2_term51_CircBuf_storageOffset_every_tail_and_offset(t *testing.T) {
	buf := cbInit(t, 0, 5)
	defer buf.Deinit()

	for tl := uint(0); tl < 5; tl++ {
		buf.tail = tl
		for o := uint(0); o < 5; o++ {
			expectEqual(t, (tl+o)%5, buf.storageOffset(o))
		}
	}
}

// Zig: test "term51: CircBuf.storageOffset: capacity one"
func TestCb2_term51_CircBuf_storageOffset_capacity_one(t *testing.T) {
	buf := cbInit(t, 0, 1)
	defer buf.Deinit()
	expectEqual(t, 0, buf.storageOffset(0))
}

// Zig: test "term51: CircBuf.storageOffset: large capacity"
func TestCb2_term51_CircBuf_storageOffset_large_capacity(t *testing.T) {
	const n uint = 1 << 20
	buf := cbInit(t, 0, n)
	defer buf.Deinit()
	buf.tail = n - 1
	expectEqual(t, n-1, buf.storageOffset(0))
	expectEqual(t, 0, buf.storageOffset(1))
	expectEqual(t, n-2, buf.storageOffset(n-1))
}

// Zig: test "term51: CircBuf.resize: large capacity grow after wrapping"
func TestCb2_term51_CircBuf_resize_large_capacity_grow_after_wrapping(t *testing.T) {
	const n uint = 1 << 16
	buf := cbInit(t, 0, n)
	defer buf.Deinit()

	// Zig-only: const data = try alloc.alloc(u8, n); defer alloc.free(data);
	data := make([]uint8, n)
	for i := range data {
		data[i] = uint8(i*13 + 1)
	}

	buf.AppendSliceAssumeCapacity(data)
	buf.DeleteOldest(1000)
	buf.AppendSliceAssumeCapacity(data[0:1000])
	expectEqual(t, 1000, buf.tail)

	zigTry(t, buf.Resize(3*n))
	expectEqual(t, n, buf.Len())
	expectEqual(t, 3*n, buf.Capacity())
	expectEqualSlices(t, data[1000:], buf.storage[0:n-1000])
	expectEqualSlices(t, data[0:1000], buf.storage[n-1000:n])
	expect(t, cbAllEqual(buf.storage[n:], 0), "std.mem.allEqual(u8, buf.storage[n..], 0)")
}

// ---------------------------------------------------------------------------
// Go-only panic tests. Zig's tests cannot test an assert (inlineAssert) or a
// safety-checked panic: the test would abort. Each test below drives one of
// those paths through the Go API.
// ---------------------------------------------------------------------------

// Go-only: panic test: appendAssumeCapacity: assert(!self.full), buffer is full
func TestCb2_panic_appendAssumeCapacity_full(t *testing.T) {
	buf := cbInit(t, 0, 3)
	defer buf.Deinit()
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3})
	expect(t, buf.full, "buf.full")
	cbExpectPanic(t, func() { buf.AppendAssumeCapacity(4) })
}

// Go-only: panic test: appendAssumeCapacity: assert(!self.full), zero-capacity buffer is full
func TestCb2_panic_appendAssumeCapacity_zero_capacity(t *testing.T) {
	buf := cbInit(t, 0, 0)
	defer buf.Deinit()
	expect(t, buf.full, "buf.full")
	cbExpectPanic(t, func() { buf.AppendAssumeCapacity(1) })
}

// Go-only: panic test: appendSliceAssumeCapacity: slice longer than the free space (getPtrSlice assert offset + slice_len <= capacity)
func TestCb2_panic_appendSliceAssumeCapacity_beyond_capacity(t *testing.T) {
	buf := cbInit(t, 0, 4)
	defer buf.Deinit()
	buf.AppendSliceAssumeCapacity([]uint8{1, 2})
	cbExpectPanic(t, func() { buf.AppendSliceAssumeCapacity([]uint8{3, 4, 5}) })

	full := cbInit(t, 0, 2)
	defer full.Deinit()
	full.AppendSliceAssumeCapacity([]uint8{1, 2})
	cbExpectPanic(t, func() { full.AppendSliceAssumeCapacity([]uint8{3}) })

	empty := cbInit(t, 0, 2)
	defer empty.Deinit()
	cbExpectPanic(t, func() { empty.AppendSliceAssumeCapacity([]uint8{1, 2, 3}) })
}

// Go-only: panic test: getPtrSlice: assert(offset + slice_len <= capacity), slice ends past the capacity
func TestCb2_panic_getPtrSlice_past_capacity(t *testing.T) {
	buf := cbInit(t, 0, 4)
	defer buf.Deinit()
	cbExpectPanic(t, func() { buf.GetPtrSlice(0, 5) })
	cbExpectPanic(t, func() { buf.GetPtrSlice(3, 2) })
	cbExpectPanic(t, func() { buf.GetPtrSlice(4, 1) })
	cbExpectPanic(t, func() { buf.GetPtrSlice(1000, 1) })
}

// Go-only: panic test: getPtrSlice: assert(offset + slice_len <= capacity), non-zero length on a zero-capacity buffer
func TestCb2_panic_getPtrSlice_zero_capacity(t *testing.T) {
	buf := cbInit(t, 0, 0)
	defer buf.Deinit()
	cbExpectPanic(t, func() { buf.GetPtrSlice(0, 1) })
}

// Go-only: panic test: deleteOldest: assert(n <= storage.len), n greater than the capacity
func TestCb2_panic_deleteOldest_more_than_capacity(t *testing.T) {
	buf := cbInit(t, 0, 3)
	defer buf.Deinit()
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3})
	cbExpectPanic(t, func() { buf.DeleteOldest(4) })

	empty := cbInit(t, 0, 3)
	defer empty.Deinit()
	cbExpectPanic(t, func() { empty.DeleteOldest(4) })

	zero := cbInit(t, 0, 0)
	defer zero.Deinit()
	cbExpectPanic(t, func() { zero.DeleteOldest(1) })
}

// Go-only: panic test: storageOffset: assert(offset < storage.len), offset at or after the capacity
func TestCb2_panic_storageOffset_at_or_after_capacity(t *testing.T) {
	buf := cbInit(t, 0, 5)
	defer buf.Deinit()
	cbExpectPanic(t, func() { buf.storageOffset(5) })
	cbExpectPanic(t, func() { buf.storageOffset(6) })
	cbExpectPanic(t, func() { buf.storageOffset(1000) })

	zero := cbInit(t, 0, 0)
	defer zero.Deinit()
	cbExpectPanic(t, func() { zero.storageOffset(0) })
}

// Go-only: panic test: advance: assert(amount <= storage.len - len), amount beyond the free space
func TestCb2_panic_advance_beyond_free_space(t *testing.T) {
	buf := cbInit(t, 0, 4)
	defer buf.Deinit()
	cbExpectPanic(t, func() { buf.advance(5) })

	part := cbInit(t, 0, 4)
	defer part.Deinit()
	part.AppendSliceAssumeCapacity([]uint8{1, 2, 3})
	cbExpectPanic(t, func() { part.advance(2) })

	full := cbInit(t, 0, 4)
	defer full.Deinit()
	full.AppendSliceAssumeCapacity([]uint8{1, 2, 3, 4})
	cbExpectPanic(t, func() { full.advance(1) })

	zero := cbInit(t, 0, 0)
	defer zero.Deinit()
	cbExpectPanic(t, func() { zero.advance(1) })
}

// Go-only: panic test: append: index out of bounds, head beyond the capacity after a shrink below the length
func TestCb2_panic_append_after_shrink_below_length(t *testing.T) {
	buf := cbInit(t, 0, 6)
	defer buf.Deinit()
	buf.AppendSliceAssumeCapacity([]uint8{1, 2, 3, 4})
	zigTry(t, buf.Resize(2))
	expectEqual(t, 4, buf.head)
	expect(t, !buf.full, "!buf.full")
	cbExpectPanic(t, func() { _ = buf.Append(9) })
}

// Go-only: panic test: append: index out of bounds, head equal to the capacity after a shrink to exactly the length
func TestCb2_panic_append_after_shrink_to_length(t *testing.T) {
	buf := cbInit(t, 0, 6)
	defer buf.Deinit()
	buf.AppendSliceAssumeCapacity([]uint8{1, 2})
	zigTry(t, buf.Resize(2))
	expectEqual(t, 2, buf.head)
	expect(t, !buf.full, "!buf.full")
	cbExpectPanic(t, func() { _ = buf.Append(9) })
}

// Go-only: panic test: appendAssumeCapacity: index out of bounds, head equal to the capacity after a shrink to exactly the length
func TestCb2_panic_appendAssumeCapacity_after_shrink_to_length(t *testing.T) {
	buf := cbInit(t, 0, 6)
	defer buf.Deinit()
	buf.AppendSliceAssumeCapacity([]uint8{1, 2})
	zigTry(t, buf.Resize(2))
	expectEqual(t, 2, buf.head)
	expect(t, !buf.full, "!buf.full")
	cbExpectPanic(t, func() { buf.AppendAssumeCapacity(9) })
}

// Go-only: panic test: append: index out of bounds, zero-capacity buffer after clear (no longer full) and after a shrink of an empty buffer to zero
func TestCb2_panic_append_zero_capacity_not_full(t *testing.T) {
	cleared := cbInit(t, 0, 0)
	defer cleared.Deinit()
	cleared.Clear()
	expect(t, !cleared.full, "!cleared.full")
	cbExpectPanic(t, func() { _ = cleared.Append(1) })

	shrunk := cbInit(t, 0, 3)
	defer shrunk.Deinit()
	zigTry(t, shrunk.Resize(0))
	expect(t, !shrunk.full, "!shrunk.full")
	cbExpectPanic(t, func() { _ = shrunk.Append(1) })
}

// Go-only: panic test: Iterator.next: division by zero (idx % capacity), zero capacity with a non-zero len after a shrink to zero
func TestCb2_panic_iterator_next_zero_capacity_with_len(t *testing.T) {
	buf := cbInit(t, 0, 6)
	defer buf.Deinit()
	buf.AppendSliceAssumeCapacity([]uint8{1, 2})
	zigTry(t, buf.Resize(0))
	expectEqual(t, 0, buf.Capacity())
	expectEqual(t, 2, buf.Len())

	fwd := buf.Iterator(DatastructCircBufIteratorDirectionForward)
	cbExpectPanic(t, func() { fwd.Next() })
	rev := buf.Iterator(DatastructCircBufIteratorDirectionReverse)
	cbExpectPanic(t, func() { rev.Next() })
	cbExpectPanic(t, func() { buf.First() })
	cbExpectPanic(t, func() { buf.Last() })
}
