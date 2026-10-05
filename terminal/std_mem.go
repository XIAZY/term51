// Ported from Zig 0.15.2's lib/std/mem.zig: only the functions the port
// reaches (Screen.zig's selectLine, selectAll and selectWord call
// std.mem.indexOfAny). Go's standard library has no function that takes a
// set of values (slices.Index takes one), so the two functions are ported.
// std's mem.zig has no test block for either; there is no matching test
// file.

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
