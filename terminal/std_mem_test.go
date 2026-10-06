// Tests translated from Zig 0.15.2's lib/std/mem.zig (the toolchain that
// built the recorded corpus), one Go test per Zig test of a ported
// function, each with the exact Zig title in a "// Zig: test" comment.
// scripts/check-test-mapping.py does not check this file: it maps Ghostty
// source files, not std files.
//
// Only `reverse` and `rotate` have test blocks among the ported functions
// (std_mem.go). In `test reverse` the u0 block has no Go element type of
// zero bits: it is translated with an empty slice of struct{}, which is the
// same check (reversing nothing leaves nothing). The union block uses a Go
// struct with a tag, as docs/mapping.md maps a tagged union.

package terminal

import "testing"

// Zig: test reverse (lib/std/mem.zig)
func TestStdMem_reverse(t *testing.T) {
	{
		arr := []int32{5, 3, 1, 2, 4}
		stdMemReverse(arr[0:])
		expectEqualSlices(t, arr, []int32{4, 2, 1, 3, 5})
	}
	{
		arr := []struct{}{}
		stdMemReverse(arr[0:])
		expectEqualSlices(t, arr, []struct{}{})
	}
	{
		arr := []int64{19, 17, 15, 13, 11, 9, 7, 5, 3, 1, 2, 4, 6, 8, 10, 12, 14, 16, 18}
		stdMemReverse(arr[0:])
		expectEqualSlices(t, arr, []int64{18, 16, 14, 12, 10, 8, 6, 4, 2, 1, 3, 5, 7, 9, 11, 13, 15, 17, 19})
	}
	{
		arr := []string{"a", "b", "c", "d"}
		stdMemReverse(arr[0:])
		expectEqualSlices(t, arr, []string{"d", "c", "b", "a"})
	}
	{
		// Zig: union(enum) { a: [3]u8, b: u24, c }.
		type myType struct {
			tag uint8
			a   [3]uint8
			b   uint32
		}
		arr := []myType{{tag: 0, a: [3]uint8{0, 0, 0}}, {tag: 1, b: 0}, {tag: 2}}
		stdMemReverse(arr[0:])
		expectEqualSlices(t, arr, []myType{{tag: 2}, {tag: 1, b: 0}, {tag: 0, a: [3]uint8{0, 0, 0}}})
	}
}

// Zig: test rotate (lib/std/mem.zig)
func TestStdMem_rotate(t *testing.T) {
	arr := []int32{5, 3, 1, 2, 4}
	stdMemRotate(arr[0:], 2)

	expectEqualSlices(t, []int32{1, 2, 4, 5, 3}, arr)
}
