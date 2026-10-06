// Ported from Zig 0.15.2's lib/std/mem.zig: only the functions the port
// reaches (Screen.zig's selectLine, selectAll and selectWord call
// std.mem.indexOfAny). Go's standard library has no function that takes a
// set of values (slices.Index takes one), so the two functions are ported.
// std's mem.zig has no test block for either.
//
// `reverse` and `rotate` (circ_buf.zig's rotateToZero, sliding_window.zig,
// screen.zig) are ported too, so that the rotation is std's three reversals
// (slices.Reverse would do for `reverse`, but `rotate` has no Go
// counterpart). `reverse`'s SIMD block (`use_vectors`) is not ported: it
// swaps the same elements as the scalar loop (docs/mapping.md, governing
// rule 8). Their std test blocks are in std_mem_test.go.

package terminal

// stdMemIndexOfAny is std.mem.indexOfAny (no doc comment upstream).
//
// Zig: pub fn indexOfAny(comptime T: type, slice: []const T,
// values: []const T) ?usize.
func stdMemIndexOfAny[T comparable](slice []T, values []T) Optional[uint] {
	return stdMemIndexOfAnyPos(slice, 0, values)
}

// stdMemIndexOfAnyPos is std.mem.indexOfAnyPos (no doc comment upstream).
//
// Zig: pub fn indexOfAnyPos(comptime T: type, slice: []const T,
// start_index: usize, values: []const T) ?usize.
func stdMemIndexOfAnyPos[T comparable](slice []T, start_index uint, values []T) Optional[uint] {
	if start_index >= uint(len(slice)) {
		return None[uint]()
	}
	for i := start_index; i < uint(len(slice)); i++ {
		c := slice[i]
		for _, value := range values {
			if c == value {
				return Some(i)
			}
		}
	}
	return None[uint]()
}

// stdMemReverse is std.mem.reverse.
//
// # In-place order reversal of a slice
//
// Zig: pub fn reverse(comptime T: type, items: []T) void. The SIMD block
// (`use_vectors`) is not ported; see the file header.
func stdMemReverse[T any](items []T) {
	var i uint = 0
	end := uint(len(items)) / 2
	for i < end {
		// Zig: swap(T, &items[i], &items[items.len - i - 1])
		items[i], items[uint(len(items))-i-1] = items[uint(len(items))-i-1], items[i]
		i += 1
	}
}

// stdMemRotate is std.mem.rotate.
//
// In-place rotation of the values in an array ([0 1 2 3] becomes [1 2 3 0] if we rotate by 1)
// Assumes 0 <= amount <= items.len
//
// Zig: pub fn rotate(comptime T: type, items: []T, amount: usize) void.
func stdMemRotate[T any](items []T, amount uint) {
	stdMemReverse(items[0:amount])
	stdMemReverse(items[amount:])
	stdMemReverse(items)
}
