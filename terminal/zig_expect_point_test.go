package terminal

import "testing"

// Points are compared the way Zig compares them, never with Go's ==.
//
// point.Point is a Zig union(Tag): std.testing.expectEqual on it compares the
// active tags and then only the active payload. The Go Point is a struct with
// all four payload fields, so Go's == would also compare the three inactive
// ones (docs/mapping.md, "Points are compared by tag and active coordinate").

// expectEqualPoint is Zig's `try testing.expectEqual(expected, actual)` with
// both of type point.Point.
func expectEqualPoint(t *testing.T, want Point, got Point) {
	t.Helper()
	if got.Tag != want.Tag || !got.Coord().Eql(want.Coord()) {
		t.Fatalf("got %v %+v, want %v %+v", got.Tag, got.Coord(), want.Tag, want.Coord())
	}
}

// expectEqualOptionalPoint is Zig's `try testing.expectEqual(expected,
// actual)` with both of type ?point.Point: both null, or both non-null and
// equal as expectEqualPoint.
func expectEqualOptionalPoint(t *testing.T, want Optional[Point], got Optional[Point]) {
	t.Helper()
	if got.Valid != want.Valid {
		t.Fatalf("got valid=%v, want valid=%v (want %v)", got.Valid, want.Valid, want)
	}
	if want.Valid {
		expectEqualPoint(t, want.Value, got.Value)
	}
}
