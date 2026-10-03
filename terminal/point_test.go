// Tests translated one for one from src/terminal/point.zig (Ghostty v1.3.1
// plus the term51 additional tests). Each Go test carries the exact Zig title
// in a "// Zig: test" comment.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualStrings keep Zig's names and Zig's
//     (expected, actual) argument order (helpers in Parser_test.go).
//   - `@tagName` is PointTag.String(), `@intFromEnum` is a conversion to int.
//   - `@TypeOf(c.x) == u16` is a reflect check on the field's kind.

package terminal

import (
	"reflect"
	"testing"
)

// Zig: test "term51: Tag: values and order"
func TestPoint_term51_Tag_values_and_order(t *testing.T) {
	// Zig: fields.len is 4. Go has no enum field list; the number of tags is
	// the value of the last tag plus one (the values are consecutive from 0).
	expectEqual(t, 4, int(PointTagHistory)+1)
	expectEqualStrings(t, "active", PointTag(0).String())
	expectEqualStrings(t, "viewport", PointTag(1).String())
	expectEqualStrings(t, "screen", PointTag(2).String())
	expectEqualStrings(t, "history", PointTag(3).String())
	expectEqual(t, 0, int(PointTagActive))
	expectEqual(t, 1, int(PointTagViewport))
	expectEqual(t, 2, int(PointTagScreen))
	expectEqual(t, 3, int(PointTagHistory))
}

// Zig: test "term51: Point.coord: returns the coordinate for every tag"
func TestPoint_term51_Point_coord_returns_the_coordinate_for_every_tag(t *testing.T) {
	c := PointCoordinate{x: 12, y: 345}

	active := Point{Tag: PointTagActive, Active: c}
	viewport := Point{Tag: PointTagViewport, Viewport: c}
	screen := Point{Tag: PointTagScreen, Screen: c}
	history := Point{Tag: PointTagHistory, History: c}

	expectEqual(t, PointTagActive, active.Tag)
	expectEqual(t, PointTagViewport, viewport.Tag)
	expectEqual(t, PointTagScreen, screen.Tag)
	expectEqual(t, PointTagHistory, history.Tag)

	// The tag is dropped; only the coordinate comes back.
	for _, p := range []Point{active, viewport, screen, history} {
		expectEqual(t, SizeCellCountInt(12), p.Coord().x)
		expectEqual(t, uint32(345), p.Coord().y)
		expect(t, p.Coord().Eql(c), "p.coord().eql(c)")
	}
}

// Zig: test "term51: Point.coord: default and extreme coordinates"
func TestPoint_term51_Point_coord_default_and_extreme_coordinates(t *testing.T) {
	zero := Point{Tag: PointTagScreen, Screen: PointCoordinate{}}
	expectEqual(t, SizeCellCountInt(0), zero.Coord().x)
	expectEqual(t, uint32(0), zero.Coord().y)

	max := Point{Tag: PointTagHistory, History: PointCoordinate{
		x: 1<<16 - 1, // std.math.maxInt(size.CellCountInt)
		y: 1<<32 - 1, // std.math.maxInt(u32)
	}}
	expectEqual(t, SizeCellCountInt(65535), max.Coord().x)
	expectEqual(t, uint32(4294967295), max.Coord().y)

	// Only x/y are returned; the result is a copy.
	p := Point{Tag: PointTagActive, Active: PointCoordinate{x: 1, y: 2}}
	got := p.Coord()
	got.x = 9
	expectEqual(t, SizeCellCountInt(1), p.Active.x)
	p.Active.y = 7
	expectEqual(t, uint32(7), p.Coord().y)
}

// Zig: test "term51: Coordinate.eql: defaults and field types"
func TestPoint_term51_Coordinate_eql_defaults_and_field_types(t *testing.T) {
	c := PointCoordinate{}
	expectEqual(t, SizeCellCountInt(0), c.x)
	expectEqual(t, uint32(0), c.y)
	expect(t, reflect.TypeOf(c.x).Kind() == reflect.Uint16, "@TypeOf(c.x) == u16")
	expect(t, reflect.TypeOf(c.y).Kind() == reflect.Uint32, "@TypeOf(c.y) == u32")
	expect(t, c.Eql(PointCoordinate{}), "c.eql(.{})")
}

// Zig: test "term51: Coordinate.eql: compares both fields"
func TestPoint_term51_Coordinate_eql_compares_both_fields(t *testing.T) {
	a := PointCoordinate{x: 3, y: 4}

	expect(t, a.Eql(PointCoordinate{x: 3, y: 4}), "a.eql(.{ .x = 3, .y = 4 })")
	expect(t, !a.Eql(PointCoordinate{x: 4, y: 4}), "!a.eql(.{ .x = 4, .y = 4 })")
	expect(t, !a.Eql(PointCoordinate{x: 3, y: 5}), "!a.eql(.{ .x = 3, .y = 5 })")
	expect(t, !a.Eql(PointCoordinate{x: 4, y: 5}), "!a.eql(.{ .x = 4, .y = 5 })")
	// Swapped fields are not equal.
	expect(t, !a.Eql(PointCoordinate{x: 4, y: 3}), "!a.eql(.{ .x = 4, .y = 3 })")
	// Zero in one field only.
	expect(t, !a.Eql(PointCoordinate{x: 0, y: 4}), "!a.eql(.{ .x = 0, .y = 4 })")
	expect(t, !a.Eql(PointCoordinate{x: 3, y: 0}), "!a.eql(.{ .x = 3, .y = 0 })")
	expect(t, !a.Eql(PointCoordinate{}), "!a.eql(.{})")

	// Symmetric.
	b := PointCoordinate{x: 4, y: 3}
	expectEqual(t, a.Eql(b), b.Eql(a))
	expect(t, a.Eql(a), "a.eql(a)")
}

// Zig: test "term51: Coordinate.eql: maximum values and y beyond a u16"
func TestPoint_term51_Coordinate_eql_maximum_values_and_y_beyond_a_u16(t *testing.T) {
	max := PointCoordinate{x: 65535, y: 4294967295}
	expect(t, max.Eql(PointCoordinate{x: 65535, y: 4294967295}), "max.eql(.{ .x = 65535, .y = 4294967295 })")
	expect(t, !max.Eql(PointCoordinate{x: 65534, y: 4294967295}), "!max.eql(.{ .x = 65534, .y = 4294967295 })")
	expect(t, !max.Eql(PointCoordinate{x: 65535, y: 4294967294}), "!max.eql(.{ .x = 65535, .y = 4294967294 })")

	// y is a full u32: 65536 is not the same as 0.
	y0 := PointCoordinate{x: 1, y: 0}
	expect(t, !y0.Eql(PointCoordinate{x: 1, y: 65536}), "!y0.eql(.{ .x = 1, .y = 65536 })")
	y1 := PointCoordinate{x: 1, y: 65536}
	expect(t, y1.Eql(PointCoordinate{x: 1, y: 65536}), "y1.eql(.{ .x = 1, .y = 65536 })")
	expect(t, !y1.Eql(PointCoordinate{x: 1, y: 1}), "!y1.eql(.{ .x = 1, .y = 1 })")
}
