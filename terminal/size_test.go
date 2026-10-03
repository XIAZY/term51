// Tests translated from src/terminal/size.zig (Ghostty v1.3.1 plus the
// term51 additional tests). Each Go test carries the exact Zig title in a
// "// Zig: test" comment.
//
// Only the tests of the integer types and of max_page_size are translated.
// Every test of Offset, OffsetBuf, getOffset and intFromBase is not
// translated (the Go wireframe has none of them; see
// scripts/not-translated-sm1.txt).
//
// Translation notes:
//   - `try testing.expect(A == u32)` on a type alias is a reflect.Type
//     comparison (the Go types are aliases, so the types compare equal).
//   - expect and expectEqual are the helpers of Parser_test.go.

package terminal

import (
	"reflect"
	"testing"
)

// Zig: test "Offset"
func TestSize_Offset(t *testing.T) {
	// This test is here so that if Offset changes, we can be very aware
	// of this effect and think about the implications of it.
	expect(t, reflect.TypeOf(SizeOffsetInt(0)) == reflect.TypeOf(uint32(0)), "OffsetInt == u32")
}

// Zig: test "term51: Offset: layout of the offset int and the count types"
func TestSize_term51_Offset_layout_of_the_offset_int_and_the_count_types(t *testing.T) {
	expectEqual(t, uint64(4294967295), uint64(SizeMaxPageSize))
	expect(t, reflect.TypeOf(SizeOffsetInt(0)) == reflect.TypeOf(uint32(0)), "OffsetInt == u32")
	expect(t, reflect.TypeOf(SizeCellCountInt(0)) == reflect.TypeOf(uint16(0)), "CellCountInt == u16")
	expect(t, reflect.TypeOf(SizeStyleCountInt(0)) == reflect.TypeOf(uint16(0)), "StyleCountInt == u16")
	expect(t, reflect.TypeOf(SizeHyperlinkCountInt(0)) == reflect.TypeOf(uint16(0)), "HyperlinkCountInt == u16")
	expect(t, reflect.TypeOf(SizeGraphemeBytesInt(0)) == reflect.TypeOf(uint32(0)), "GraphemeBytesInt == u32")
	expect(t, reflect.TypeOf(SizeStringBytesInt(0)) == reflect.TypeOf(uint32(0)), "StringBytesInt == u32")

	// Zig-only: Offset(T) is a packed struct over OffsetInt regardless of T:
	// the @sizeOf, @bitSizeOf, @alignOf and @bitCast lines on Offset(u8),
	// Offset(u64), Offset([4]u64), Offset(u16) and `Offset(u8) != Offset(u16)`.
}
