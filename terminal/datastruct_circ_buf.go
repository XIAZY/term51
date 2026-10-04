// WIREFRAME for src/datastruct/circ_buf.zig (Ghostty v1.3.1). NOT YET
// PORTED.
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
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
// (`usize`) are counts and so `uint`; `seekBy`'s `isize` is `int`.
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
//   - the `String` method of DatastructCircBufIteratorDirection.
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

// String returns the Zig tag name (@tagName).
func (self DatastructCircBufIteratorDirection) String() string { panic(notImplemented) }

// Next is CircBuf.Iterator.next.
//
// Zig: pub fn next(self: *Iterator) ?*T. nil is null.
func (self *DatastructCircBufIterator[T]) Next() *T { panic(notImplemented) }

// SeekBy is CircBuf.Iterator.seekBy.
//
// Seek the iterator by a given amount. This will clamp
// the values to the bounds of the buffer so overflows are
// not possible.
//
// Zig: amount is isize; idx is moved with saturating `+|=` and `-|=`.
func (self *DatastructCircBufIterator[T]) SeekBy(amount int) { panic(notImplemented) }

// Reset is CircBuf.Iterator.reset.
//
// Reset the iterator back to the first value.
func (self *DatastructCircBufIterator[T]) Reset() { panic(notImplemented) }

// DatastructCircBufInit is CircBuf.init.
//
// Initialize a new circular buffer that can store size elements.
//
// Zig: pub fn init(alloc: Allocator, size: usize) Allocator.Error!Self. The
// allocator parameter is dropped; `default_` is CircBuf's comptime
// `default` (see the file header). Every storage element is set to it, and
// `full` is `size == 0`. The error is ErrOutOfMemory (dcs.go).
func DatastructCircBufInit[T any](default_ T, size uint) (DatastructCircBuf[T], error) {
	panic(notImplemented)
}

// Deinit is CircBuf.deinit.
//
// Zig: pub fn deinit(self: *Self, alloc: Allocator) void; frees storage and
// sets `self.* = undefined`. The allocator parameter is dropped. Kept as a
// stub because every test calls it.
func (self *DatastructCircBuf[T]) Deinit() { panic(notImplemented) }

// Append is CircBuf.append.
//
// Append a single value to the buffer. If the buffer is full,
// an error will be returned.
//
// Zig: Allocator.Error!void. The error is ErrOutOfMemory (dcs.go), returned
// when `full` is set.
func (self *DatastructCircBuf[T]) Append(v T) error { panic(notImplemented) }

// AppendAssumeCapacity is CircBuf.appendAssumeCapacity.
//
// Append a single value to the buffer, assuming there is capacity.
func (self *DatastructCircBuf[T]) AppendAssumeCapacity(v T) { panic(notImplemented) }

// AppendSliceAssumeCapacity is CircBuf.appendSliceAssumeCapacity.
//
// Append a slice to the buffer.
//
// Zig: slice is []const T.
func (self *DatastructCircBuf[T]) AppendSliceAssumeCapacity(slice []T) { panic(notImplemented) }

// Clear is CircBuf.clear.
//
// Clear the buffer.
func (self *DatastructCircBuf[T]) Clear() { panic(notImplemented) }

// Iterator is CircBuf.iterator.
//
// Iterate over the circular buffer.
//
// Zig: self is Self (by value); the iterator holds a copy of the buffer
// header. See the file header.
func (self DatastructCircBuf[T]) Iterator(direction DatastructCircBufIteratorDirection) DatastructCircBufIterator[T] {
	panic(notImplemented)
}

// First is CircBuf.first.
//
// Get the first (oldest) value in the buffer.
//
// Zig: ?*T. nil is null.
func (self DatastructCircBuf[T]) First() *T { panic(notImplemented) }

// Last is CircBuf.last.
//
// Get the last (newest) value in the buffer.
//
// Zig: ?*T. nil is null.
func (self DatastructCircBuf[T]) Last() *T { panic(notImplemented) }

// EnsureUnusedCapacity is CircBuf.ensureUnusedCapacity.
//
// Ensures that there is enough capacity to store amount more
// items via append.
//
// Zig: pub fn ensureUnusedCapacity(self: *Self, alloc: Allocator, amount:
// usize) Allocator.Error!void. The allocator parameter is dropped. The
// error is ErrOutOfMemory (dcs.go).
func (self *DatastructCircBuf[T]) EnsureUnusedCapacity(amount uint) error { panic(notImplemented) }

// Resize is CircBuf.resize.
//
// Resize the buffer to the given size (larger or smaller).
// If larger, new values will be set to the default value.
//
// Zig: pub fn resize(self: *Self, alloc: Allocator, size: usize)
// Allocator.Error!void. The allocator parameter is dropped. The error is
// ErrOutOfMemory (dcs.go).
func (self *DatastructCircBuf[T]) Resize(size uint) error { panic(notImplemented) }

// rotateToZero is CircBuf.rotateToZero.
//
// Rotate the data so that it is zero-aligned.
//
// Zig: fn rotateToZero(self: *Self) Allocator.Error!void. The error is
// ErrOutOfMemory (dcs.go); Zig's body cannot actually fail.
func (self *DatastructCircBuf[T]) rotateToZero() error { panic(notImplemented) }

// Empty is CircBuf.empty.
//
// Returns if the buffer is currently empty. To check if its
// full, just check the "full" attribute.
func (self DatastructCircBuf[T]) Empty() bool { panic(notImplemented) }

// Capacity is CircBuf.capacity.
//
// Returns the total capacity allocated for this buffer.
func (self DatastructCircBuf[T]) Capacity() uint { panic(notImplemented) }

// Len is CircBuf.len.
//
// Returns the length in elements that are used.
func (self DatastructCircBuf[T]) Len() uint { panic(notImplemented) }

// DeleteOldest is CircBuf.deleteOldest.
//
// Delete the oldest n values from the buffer. If there are less
// than n values in the buffer, it'll delete everything.
//
// The deleted cells are set back to `default_`.
func (self *DatastructCircBuf[T]) DeleteOldest(n uint) { panic(notImplemented) }

// GetPtrSlice is CircBuf.getPtrSlice.
//
// Returns a pointer to the value at offset with the given length,
// and considers this full amount of data "written" if it is beyond
// the end of our buffer. This never "rotates" the buffer because
// the offset can only be within the size of the buffer.
//
// Zig: [2][]T, two slices of storage; see the file header.
func (self *DatastructCircBuf[T]) GetPtrSlice(offset uint, slice_len uint) [2][]T {
	panic(notImplemented)
}

// advance is CircBuf.advance.
//
// Advances the head/tail so that we can store amount.
func (self *DatastructCircBuf[T]) advance(amount uint) { panic(notImplemented) }

// storageOffset is CircBuf.storageOffset.
//
// For a given offset from zero, this returns the offset in the
// storage buffer where this data can be found.
func (self DatastructCircBuf[T]) storageOffset(offset uint) uint { panic(notImplemented) }
