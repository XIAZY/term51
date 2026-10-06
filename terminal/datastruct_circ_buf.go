// Port of src/datastruct/circ_buf.zig (Ghostty v1.3.1).
//
// File naming: circ_buf.zig is in src/datastruct, outside src/terminal; it
// is in package terminal because sliding_window.zig (search) uses it. As for
// a subdirectory of src/terminal, the directory is part of the name: the
// file is `datastruct_circ_buf.go` and its prefix is `DatastructCircBuf`.
// The file holds one declaration, the type function `CircBuf(T, default)`;
// the type it returns is `DatastructCircBuf[T]` (the declaration and the
// file have the same name, so it is not doubled). What is nested in it takes
// that prefix: `CircBuf.Iterator` is `DatastructCircBufIterator[T]`,
// `CircBuf.Iterator.Direction` is `DatastructCircBufIteratorDirection`.
// `init` is `DatastructCircBufInit`. `pub fn` methods are exported
// (`Append`, `GetPtrSlice`, `DeleteOldest`); private methods keep their Zig
// name (`rotateToZero`, `advance`, `storageOffset`).
//
// Declarations appear in the same order as in circ_buf.zig.
//
// # Generic form
//
// Zig: `pub fn CircBuf(comptime T: type, comptime default: T) type`. `T` is
// the Go type parameter. Go has no value type parameters, so `default` is a
// FIELD of the buffer, `default_` (`default` is a Go keyword), which
// `DatastructCircBufInit` sets from its first argument:
//
//	DatastructCircBufInit[T any](default_ T, size uint) (DatastructCircBuf[T], error)
//
// The default argument comes first, as in Zig, where it is given with the
// type (`CircBuf(u8, 0)`) before `init(alloc, size)`. So Zig's
// `CircBuf(u8, 0).init(alloc, 3)` is `DatastructCircBufInit[uint8](0, 3)`
// (or `DatastructCircBufInit(uint8(0), 3)`). The default is behaviour, not
// decoration: `init`, `resize` (growing) and `deleteOldest` write it into
// storage, and getPtrSlice then hands those cells out (test "CircBuf
// getPtrSlice wraps" reads them back as 0). `CircBuf(Meta, undefined)` in
// sliding_window.zig passes the Go zero value of the element type.
//
// Go cannot nest a type in a generic type. `Iterator` uses `T` and is the
// generic `DatastructCircBufIterator[T]`; `Iterator.Direction` does not and
// is the plain enum `DatastructCircBufIteratorDirection`. `Self` is
// `DatastructCircBuf[T]`.
//
// # Go-memory rule
//
// (docs/mapping.md, governing rule 3.) `storage: []T` is a Go slice. Every
// allocator parameter is dropped (`init`, `deinit`, `ensureUnusedCapacity`,
// `resize`); `Deinit` is kept as a stub (every test calls it). Zig's
// `alloc.realloc` in `resize` is a new Go slice holding the old contents.
//
// Receivers: Zig methods taking `self: Self` (iterator, first, last, empty,
// capacity, len, storageOffset) have Go value receivers; those taking
// `self: *Self` have pointer receivers. `Iterator.buf` is `Self` by value in
// Zig, a copy of the buffer header that shares `storage`; a Go struct copy
// shares the slice's backing array the same way, so the `*T` that `Next`,
// `First` and `Last` return points into the buffer's storage, as in Zig.
//
// Results: `?*T` (Iterator.next, first, last) is `*T`, nil for null.
// `getPtrSlice` returns `[2][]T`, two Go sub-slices of `storage`.
// `fastmem.copy` (appendSliceAssumeCapacity) is Go's `copy`;
// `std.mem.rotate` (rotateToZero) is a rotation of the slice.
//
// # Widths
//
// `head`, `tail`, `Iterator.idx` and every size, length, offset and amount
// (`usize`) are counts and so `uint`. `seekBy`'s `isize` is `int64`, the
// reference's isize on every platform (project lead's ruling, the rule of
// the scroll deltas in PageList.go); a positive amount larger than the
// largest `uint` of a 32-bit target saturates (`pageListUsizeFromIsize`),
// which `+|=` and `-|=` would saturate to anyway.
//
// # Errors
//
// `Allocator.Error` is ErrOutOfMemory (dcs.go). `append` returns it when the
// buffer is full, which is behaviour, not an allocation failure. `resize`,
// `rotateToZero` and `ensureUnusedCapacity` keep their error result so that
// call sites keep Zig's shape; in Go they cannot fail except where Zig
// would.
//
// # Declarations that are not Zig declarations
//
//   - the field `default_` (Zig's comptime parameter `default`);
//   - the `String` method of DatastructCircBufIteratorDirection and its
//     name table datastructCircBufIteratorDirectionNames.
//
// # Integer traps made explicit
//
// Zig traps on usize overflow and underflow; Go wraps. Asserted: `len() +
// amount` in ensureUnusedCapacity, `offset + slice_len` in getPtrSlice
// (before upstream's own assert), `storage.len - len()` in advance and in
// len() (a length beyond the capacity after a shrink). Division by a zero
// capacity (`% capacity()` in Iterator.next, `% storage.len` in
// rotateToZero) panics in Go as it traps in Zig. `@memset` and
// `fastmem.copy` are loops and Go's `copy`.
//
// # Omitted
//
// Imports with nothing to declare: `std`, `assert`, `Allocator`, `fastmem`.
// No C ABI declaration exists in this file.
//
// # Zig defaults a Go literal does not give
//
// None: no field of CircBuf or Iterator has a Zig default. But the Go zero
// value `DatastructCircBuf[T]{}` is NOT what `init(alloc, 0)` returns: init
// sets `full = (size == 0)`, so a zero-capacity buffer made by init is FULL
// (test "CircBuf resize grow from zero" checks `buf.full`), and init also
// sets `default_`. Make buffers with DatastructCircBufInit, never with a
// literal.

package terminal

import "math"

// DatastructCircBuf is the type returned by circ_buf.CircBuf(T, default):
// a circular buffer containing type T.
//
// Implementation note: there's a lot of unsafe addition of usize
// here in this implementation that can technically overflow. If someone
// wants to fix this and make it overflow safe (use subtractions for
// checks prior to additions) then I welcome it. In reality, we'd
// have to be a really, really large terminal screen to even worry
// about this so I'm punting it.
type DatastructCircBuf[T any] struct {
	// Zig: the comptime parameter `default` of CircBuf, set by
	// DatastructCircBufInit. Not a Zig field; see the file header.
	default_ T

	storage []T
	head    uint
	tail    uint

	// We could remove this and just use math with head/tail to figure
	// it out, but our usage of circular buffers stores so much data that
	// this minor overhead is not worth optimizing out.
	full bool
}

// DatastructCircBufIterator is CircBuf.Iterator.
type DatastructCircBufIterator[T any] struct {
	// Zig: `buf: Self`, the buffer by value; see the file header.
	buf       DatastructCircBuf[T]
	idx       uint
	direction DatastructCircBufIteratorDirection
}

// DatastructCircBufIteratorDirection is CircBuf.Iterator.Direction.
//
// Zig: enum { forward, reverse }, values numbered from zero in the order
// listed.
type DatastructCircBufIteratorDirection uint8

const (
	DatastructCircBufIteratorDirectionForward DatastructCircBufIteratorDirection = iota
	DatastructCircBufIteratorDirectionReverse
)

// datastructCircBufIteratorDirectionNames are the tag names of
// DatastructCircBufIteratorDirection, in order.
var datastructCircBufIteratorDirectionNames = [...]string{
	"forward",
	"reverse",
}

// String returns the Zig tag name (@tagName).
func (self DatastructCircBufIteratorDirection) String() string {
	return datastructCircBufIteratorDirectionNames[self]
}

// Next is CircBuf.Iterator.next.
//
// Zig: pub fn next(self: *Iterator) ?*T. nil is null.
func (self *DatastructCircBufIterator[T]) Next() *T {
	if self.idx >= self.buf.Len() {
		return nil
	}

	// Get our index from the tail
	var tail_idx uint
	switch self.direction {
	case DatastructCircBufIteratorDirectionForward:
		tail_idx = self.idx
	case DatastructCircBufIteratorDirectionReverse:
		tail_idx = self.buf.Len() - self.idx - 1
	default:
		panic("unreachable")
	}

	// Translate the tail index to a storage index
	storage_idx := (self.buf.tail + tail_idx) % self.buf.Capacity()
	self.idx += 1
	return &self.buf.storage[storage_idx]
}

// SeekBy is CircBuf.Iterator.seekBy.
//
// Seek the iterator by a given amount. This will clamp
// the values to the bounds of the buffer so overflows are
// not possible.
//
// Zig: amount is isize (int64, the reference's isize; see the file
// header); idx is moved with saturating `+|=` and `-|=`.
func (self *DatastructCircBufIterator[T]) SeekBy(amount int64) {
	if amount > 0 {
		// Zig: self.idx +|= @intCast(amount)
		v := pageListUsizeFromIsize(amount)
		if self.idx > math.MaxUint-v {
			self.idx = math.MaxUint
		} else {
			self.idx += v
		}
	} else {
		// Zig: self.idx -|= @intCast(@abs(amount)). @abs of an i64 is a
		// u64, so minInt has the magnitude 2^63.
		abs := uint64(0) - uint64(amount)
		if abs >= uint64(self.idx) {
			self.idx = 0
		} else {
			self.idx -= uint(abs)
		}
	}
}

// Reset is CircBuf.Iterator.reset.
//
// Reset the iterator back to the first value.
func (self *DatastructCircBufIterator[T]) Reset() {
	self.idx = 0
}

// DatastructCircBufInit is CircBuf.init.
//
// Initialize a new circular buffer that can store size elements.
//
// Zig: pub fn init(alloc: Allocator, size: usize) Allocator.Error!Self. The
// allocator parameter is dropped; `default_` is CircBuf's comptime
// `default` (see the file header). Every storage element is set to it, and
// `full` is `size == 0`. The error is ErrOutOfMemory (dcs.go).
func DatastructCircBufInit[T any](default_ T, size uint) (DatastructCircBuf[T], error) {
	buf := make([]T, size)
	for i := range buf {
		buf[i] = default_
	}

	return DatastructCircBuf[T]{
		default_: default_,
		storage:  buf,
		head:     0,
		tail:     0,
		full:     size == 0,
	}, nil
}

// Deinit is CircBuf.deinit.
//
// Zig: pub fn deinit(self: *Self, alloc: Allocator) void; frees storage and
// sets `self.* = undefined`. The allocator parameter is dropped. Kept as a
// stub because every test calls it.
func (self *DatastructCircBuf[T]) Deinit() {
	// Zig: alloc.free(self.storage); self.* = undefined. The garbage
	// collector frees; the buffer is reset to the Go zero value.
	*self = DatastructCircBuf[T]{}
}

// Append is CircBuf.append.
//
// Append a single value to the buffer. If the buffer is full,
// an error will be returned.
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go), returned
// when `full` is set.
func (self *DatastructCircBuf[T]) Append(v T) error {
	if self.full {
		return ErrOutOfMemory
	}
	self.storage[self.head] = v
	self.head += 1
	if self.head >= uint(len(self.storage)) {
		self.head = 0
	}
	self.full = self.head == self.tail
	return nil
}

// AppendAssumeCapacity is CircBuf.appendAssumeCapacity.
//
// Append a single value to the buffer, assuming there is capacity.
func (self *DatastructCircBuf[T]) AppendAssumeCapacity(v T) {
	assert(!self.full)
	self.storage[self.head] = v
	self.head += 1
	if self.head >= uint(len(self.storage)) {
		self.head = 0
	}
	self.full = self.head == self.tail
}

// AppendSliceAssumeCapacity is CircBuf.appendSliceAssumeCapacity.
//
// Append a slice to the buffer.
//
// Zig: slice is []const T.
func (self *DatastructCircBuf[T]) AppendSliceAssumeCapacity(slice []T) {
	storage := self.GetPtrSlice(
		self.Len(),
		uint(len(slice)),
	)
	// Zig: fastmem.copy (a memcpy; the lengths are equal).
	copy(storage[0], slice[0:len(storage[0])])
	copy(storage[1], slice[len(storage[0]):])
}

// Clear is CircBuf.clear.
//
// Clear the buffer.
func (self *DatastructCircBuf[T]) Clear() {
	self.head = 0
	self.tail = 0
	self.full = false
}

// Iterator is CircBuf.iterator.
//
// Iterate over the circular buffer.
//
// Zig: self is Self (by value); the iterator holds a copy of the buffer
// header. See the file header.
func (self DatastructCircBuf[T]) Iterator(direction DatastructCircBufIteratorDirection) DatastructCircBufIterator[T] {
	return DatastructCircBufIterator[T]{
		buf:       self,
		idx:       0,
		direction: direction,
	}
}

// First is CircBuf.first.
//
// Get the first (oldest) value in the buffer.
//
// Zig: ?*T. nil is null.
func (self DatastructCircBuf[T]) First() *T {
	// Note: this can be more efficient by not using the
	// iterator, but this was an easy way to implement it.
	it := self.Iterator(DatastructCircBufIteratorDirectionForward)
	return it.Next()
}

// Last is CircBuf.last.
//
// Get the last (newest) value in the buffer.
//
// Zig: ?*T. nil is null.
func (self DatastructCircBuf[T]) Last() *T {
	// Note: this can be more efficient by not using the
	// iterator, but this was an easy way to implement it.
	it := self.Iterator(DatastructCircBufIteratorDirectionReverse)
	return it.Next()
}

// EnsureUnusedCapacity is CircBuf.ensureUnusedCapacity.
//
// Ensures that there is enough capacity to store amount more
// items via append.
//
// Zig: pub fn ensureUnusedCapacity(self: *Self, alloc: Allocator, amount:
// usize) Allocator.Error!void. The allocator parameter is dropped. The
// error is ErrOutOfMemory (dcs.go).
func (self *DatastructCircBuf[T]) EnsureUnusedCapacity(amount uint) error {
	// Zig: usize addition traps on overflow. usize is 64 bits on the
	// reference on every platform, so the sum is computed in uint64.
	assert(uint64(amount) <= math.MaxUint64-uint64(self.Len()))
	new_cap := uint64(self.Len()) + uint64(amount)
	if new_cap <= uint64(self.Capacity()) {
		return nil
	}
	// A capacity beyond what a Go uint can index (only on 32-bit
	// platforms) cannot be allocated: the reference's allocation fails
	// there too, with OutOfMemory.
	if new_cap > math.MaxUint {
		return ErrOutOfMemory
	}
	return self.Resize(uint(new_cap))
}

// Resize is CircBuf.resize.
//
// Resize the buffer to the given size (larger or smaller).
// If larger, new values will be set to the default value.
//
// Zig: pub fn resize(self: *Self, alloc: Allocator, size: usize)
// Allocator.Error!void. The allocator parameter is dropped. The error is
// ErrOutOfMemory (dcs.go).
func (self *DatastructCircBuf[T]) Resize(size uint) error {
	// Rotate to zero so it is aligned.
	if err := self.rotateToZero(); err != nil {
		return err
	}

	// Reallocate, this adds to the end so we're ready to go.
	prev_len := self.Len()
	prev_cap := uint(len(self.storage))
	// Zig: self.storage = try alloc.realloc(self.storage, size): a new
	// slice of `size` elements holding the old contents (truncated when
	// shrinking).
	storage := make([]T, size)
	copy(storage, self.storage)
	self.storage = storage

	// If we grew, we need to set our new defaults. We can add it
	// at the end since we rotated to start.
	if size > prev_cap {
		for i := prev_cap; i < size; i++ {
			self.storage[i] = self.default_
		}

		// Fix up our head/tail
		if self.full {
			self.head = prev_len
			self.full = false
		}
	}
	return nil
}

// rotateToZero is CircBuf.rotateToZero.
//
// Rotate the data so that it is zero-aligned.
//
// Zig: fn rotateToZero(self: *Self) Allocator.Error!void. The error is
// ErrOutOfMemory (dcs.go); Zig's body cannot actually fail.
func (self *DatastructCircBuf[T]) rotateToZero() error {
	// If we're already at zero then do nothing.
	if self.tail == 0 {
		return nil
	}

	// We use std.mem.rotate to rotate our storage in-place.
	stdMemRotate(self.storage, self.tail)

	// Then fix up our head and tail.
	self.head = self.Len() % uint(len(self.storage))
	self.tail = 0
	return nil
}

// Empty is CircBuf.empty.
//
// Returns if the buffer is currently empty. To check if its
// full, just check the "full" attribute.
func (self DatastructCircBuf[T]) Empty() bool {
	return !self.full && self.head == self.tail
}

// Capacity is CircBuf.capacity.
//
// Returns the total capacity allocated for this buffer.
func (self DatastructCircBuf[T]) Capacity() uint {
	return uint(len(self.storage))
}

// Len is CircBuf.len.
//
// Returns the length in elements that are used.
func (self DatastructCircBuf[T]) Len() uint {
	if self.full {
		return uint(len(self.storage))
	}
	if self.head >= self.tail {
		return self.head - self.tail
	}
	// Zig: usize subtraction traps on underflow (a head beyond the
	// capacity after a shrink).
	assert(self.tail-self.head <= uint(len(self.storage)))
	return uint(len(self.storage)) - (self.tail - self.head)
}

// DeleteOldest is CircBuf.deleteOldest.
//
// Delete the oldest n values from the buffer. If there are less
// than n values in the buffer, it'll delete everything.
//
// The deleted cells are set back to `default_`.
func (self *DatastructCircBuf[T]) DeleteOldest(n uint) {
	assert(n <= uint(len(self.storage)))

	// Special case n == 0 otherwise we will accidentally break
	// our circular buffer.
	if n == 0 {
		return
	}

	// Clear the values back to default
	slices := self.GetPtrSlice(0, n)
	for _, slice := range slices {
		for i := range slice {
			slice[i] = self.default_
		}
	}

	// If we're not full, we can just advance the tail. We know
	// it'll be less than the length because otherwise we'd be full.
	self.tail += min(self.Len(), n)
	if self.tail >= uint(len(self.storage)) {
		self.tail -= uint(len(self.storage))
	}
	self.full = false
}

// GetPtrSlice is CircBuf.getPtrSlice.
//
// Returns a pointer to the value at offset with the given length,
// and considers this full amount of data "written" if it is beyond
// the end of our buffer. This never "rotates" the buffer because
// the offset can only be within the size of the buffer.
//
// Zig: [2][]T, two slices of storage; see the file header.
func (self *DatastructCircBuf[T]) GetPtrSlice(offset uint, slice_len uint) [2][]T {
	// Special case the empty slice fast-path.
	if slice_len == 0 {
		return [2][]T{{}, {}}
	}

	// Note: this assertion is very important, it hints the compiler
	// which generates ~10% faster code than without it.
	//
	// Zig: the usize addition `offset + slice_len` traps on overflow first.
	// usize is 64 bits on the reference on every platform.
	assert(uint64(offset) <= math.MaxUint64-uint64(slice_len))
	assert(uint64(offset)+uint64(slice_len) <= uint64(self.Capacity()))

	// End offset is the last offset (exclusive) for our slice.
	// We use exclusive because it makes the math easier and it
	// matches Zigs slicing parameterization.
	end_offset := offset + slice_len

	// If our slice can't fit it in our length, then we need to advance.
	if end_offset > self.Len() {
		self.advance(end_offset - self.Len())
	}

	// Our start and end indexes into the storage buffer
	start_idx := self.storageOffset(offset)
	end_idx := self.storageOffset(end_offset - 1)
	// std.log.warn("A={} B={}", .{ start_idx, end_idx });

	// Optimistically, our data fits in one slice
	if end_idx >= start_idx {
		return [2][]T{
			self.storage[start_idx : end_idx+1],
			self.storage[0:0], // So there is an empty slice
		}
	}

	return [2][]T{
		self.storage[start_idx:],
		self.storage[0 : end_idx+1],
	}
}

// advance is CircBuf.advance.
//
// Advances the head/tail so that we can store amount.
func (self *DatastructCircBuf[T]) advance(amount uint) {
	// Zig: `self.storage.len - self.len()` traps on underflow (a length
	// beyond the capacity after a shrink) before the assert.
	assert(self.Len() <= uint(len(self.storage)))
	assert(amount <= uint(len(self.storage))-self.Len())

	// Optimistically add our amount
	self.head += amount

	// If we exceeded the length of the buffer, wrap around.
	if self.head >= uint(len(self.storage)) {
		self.head = self.head - uint(len(self.storage))
	}

	// If we're full, we have to keep tail lined up.
	if self.full {
		self.tail = self.head
	}

	// We're full if the head reached the tail. The head can never
	// pass the tail because advance asserts amount is only in
	// available space left
	self.full = self.head == self.tail
}

// storageOffset is CircBuf.storageOffset.
//
// For a given offset from zero, this returns the offset in the
// storage buffer where this data can be found.
func (self DatastructCircBuf[T]) storageOffset(offset uint) uint {
	assert(offset < uint(len(self.storage)))

	// This should be subtraction ideally to avoid overflows but
	// it would take a really, really, huge buffer to overflow.
	fits_offset := self.tail + offset
	if fits_offset < uint(len(self.storage)) {
		return fits_offset
	}
	return fits_offset - uint(len(self.storage))
}
