// Port of src/terminal/point.zig (Ghostty v1.3.1).
//
// File naming: point.zig is a namespace file, so its declarations take the
// prefix `Point`: `point.Tag` is `PointTag`, `point.Point` is `Point`
// (as `modes.Mode` is `ModesMode`), `point.Coordinate` is `PointCoordinate`.
//
// Declarations appear in the same order as in point.zig. The file has no C
// ABI declarations and nothing is omitted.

package terminal

// PointTag is point.Tag: the possible reference locations for a point. When
// someone says "(42, 80)" in the context of a terminal, that could mean
// multiple things: it is in the current visible viewport? the current active
// area of the screen where the cursor is? the entire scrollback history? etc.
//
// This tag is used to differentiate those cases.
//
// Zig: enum, values numbered from zero in the order listed.
type PointTag uint8

const (
	// Top-left is part of the active area where a running program can
	// jump the cursor and make changes. The active area is the "editable"
	// part of the screen.
	//
	// The bottom-right of the active tag differs from all other tags
	// because it includes the full height (rows) of the screen, including
	// rows that may not be written yet. This is required because the active
	// area is fully "addressable" by the running program (see below) whereas
	// the other tags are used primarily for reading/modifying past-written
	// data so they can't address unwritten rows.
	//
	// Note for those less familiar with terminal functionality: there
	// are escape sequences to move the cursor to any position on
	// the screen, but it is limited to the size of the viewport and
	// the bottommost part of the screen. Terminal programs can't --
	// with sequences at the time of writing this comment -- modify
	// anything in the scrollback, visible viewport (if it differs
	// from the active area), etc.
	PointTagActive PointTag = iota

	// Top-left is the visible viewport. This means that if the user
	// has scrolled in any direction, top-left changes. The bottom-right
	// is the last written row from the top-left.
	PointTagViewport

	// Top-left is the furthest back in the scrollback history
	// supported by the screen and the bottom-right is the bottom-right
	// of the last written row. Note this last point is important: the
	// bottom right is NOT necessarily the same as "active" because
	// "active" always allows referencing the full rows tall of the
	// screen whereas "screen" only contains written rows.
	PointTagScreen

	// The top-left is the same as "screen" but the bottom-right is
	// the line just before the top of "active". This contains only
	// the scrollback history.
	PointTagHistory

	// pointTagCount is the number of tags: Zig's
	// @typeInfo(Tag).@"enum".fields.len.
	pointTagCount = iota
)

var pointTagNames = [pointTagCount]string{
	"active",
	"viewport",
	"screen",
	"history",
}

// String returns the Zig tag name (@tagName). A value outside the enum
// panics, as @tagName of an invalid value does.
func (self PointTag) String() string {
	return pointTagNames[self]
}

// Point is point.Point: an x/y point in the terminal for some
// definition of location (tag).
//
// Zig: union(Tag). Only the field named by Tag is meaningful. Every variant
// carries a Coordinate.
type Point struct {
	Tag PointTag

	Active   PointCoordinate
	Viewport PointCoordinate
	Screen   PointCoordinate
	History  PointCoordinate
}

// Coord is point.Point.coord: the coordinate of whichever variant is active.
//
// Zig: `pub inline fn`; one switch prong for all four tags captures the
// payload. A tag outside the enum is unreachable in Zig.
func (self Point) Coord() PointCoordinate {
	switch self.Tag {
	case PointTagActive:
		return self.Active
	case PointTagViewport:
		return self.Viewport
	case PointTagScreen:
		return self.Screen
	case PointTagHistory:
		return self.History
	default:
		panic("unreachable")
	}
}

// PointCoordinate is point.Coordinate.
type PointCoordinate struct {
	// x can use size.CellCountInt because the number of columns
	// can't ever be more than a valid number of columns in a Page.
	X SizeCellCountInt

	// y does not use size.CellCountInt because certain coordinate
	// usage such as screen/history can have more rows than are possible
	// in a single page.
	Y uint32
}

// Eql is point.Coordinate.eql.
func (self PointCoordinate) Eql(other PointCoordinate) bool {
	return self.X == other.X && self.Y == other.Y
}
