// WIREFRAME for src/terminal/Selection.zig (Ghostty v1.3.1). NOT YET PORTED.
//
// Represents a single selection within the terminal (i.e. a highlight
// region).
//
// This file declares types, constants and function signatures only, so that
// the translated Zig tests compile. Every function body is
// panic(notImplemented). The implementation is added only after the project
// owner and the project lead both approve it. Do not fill in bodies before
// then. See the header of color.go for the rules.
//
// File naming: Selection.zig is a file-level struct, so the type is
// `Selection` and its constructor is `SelectionInit`. What is nested in it
// takes the prefix `Selection`: `Selection.Bounds` is `SelectionBounds`,
// `Selection.Order` is `SelectionOrder`, `Selection.Adjustment` is
// `SelectionAdjustment`. `pub fn` methods are exported (`Start`, `End`,
// `TopLeft`, `Adjust`); fields keep their Zig names (`bounds`, `rectangle`).
//
// Declarations appear in the same order as in Selection.zig.
//
// NOTE(mitchellh): I'm not very happy with how this is implemented, because
// the ordering operations which are used frequently require using
// pointFromPin which -- at the time of writing this -- is slow. The overall
// style of this struct is due to porting it from the previous implementation
// which had an efficient ordering operation.
//
// While reimplementing this, there were too many callers that already
// depended on this behavior so I kept it despite the inefficiency. In the
// future, we should take a look at this again!
//
// # Go-memory rule
//
// (docs/mapping.md, governing rule 3.) Ordinary Go memory. A pin is a
// `PageListPin`; a tracked pin is a `*PageListPin` owned by the screen's
// page list (`PageList.TrackPin`). There is no allocator parameter in this
// file. `deinit` is kept: it untracks the pins of a tracked selection.
//
// # Declarations that are not Zig declarations
//
//   - `SelectionBoundsUntracked` and `SelectionBoundsTracked`: the two
//     anonymous structs that are the payloads of `Bounds`;
//   - `SelectionBoundsTag` and the `String` methods.
//
// # Receivers
//
// `self: Selection` is a value receiver, `self: *Selection` a pointer
// receiver. `self: *const Selection` (`tracked`, `track`) is a pointer
// receiver too, as in PageList.go. `StartPtr` and `EndPtr` on an untracked
// selection return the address of the pin stored inside the selection
// itself (`&v.start`), so they must be called on an addressable Selection;
// on a tracked selection they return the tracked pin.
//
// # Zig defaults a Go literal does not give
//
// None. `rectangle` defaults to false and the Go zero value of
// `SelectionBounds` is an untracked pair of zero pins; Zig gives `bounds`
// no default.
//
// # Omitted
//
// Aliases and imports with nothing to declare: `page`, `point`, `PageList`,
// `Screen`, `Pin` (use `PageListPin`), `Allocator`, `assert`.

package terminal

// Selection is the file-level struct of Selection.zig.
type Selection struct {
	// The bounds of the selection.
	bounds SelectionBounds

	// Whether or not this selection refers to a rectangle, rather than whole
	// lines of a buffer. In this mode, start and end refer to the top left and
	// bottom right of the rectangle, or vice versa if the selection is backwards.
	//
	// Zig default: false.
	rectangle bool
}

// SelectionBounds is Selection.Bounds.
//
// The bounds of the selection. A selection bounds can be either tracked
// or untracked. Untracked bounds are unsafe beyond the point the terminal
// screen may be modified, since they may point to invalid memory. Tracked
// bounds are always valid and will be updated as the screen changes, but
// are more expensive to exist.
//
// In all cases, start and end can be in any order. There is no guarantee that
// start is before end or vice versa. If a user selects backwards,
// start will be after end, and vice versa. Use the struct functions
// to not have to worry about this.
//
// Zig: union(enum) { untracked: struct { start: Pin, end: Pin },
// tracked: struct { start: *Pin, end: *Pin } }. Only the field named by Tag
// is meaningful. The Go zero value is `.untracked`.
type SelectionBounds struct {
	Tag SelectionBoundsTag

	Untracked SelectionBoundsUntracked
	Tracked   SelectionBoundsTracked
}

// SelectionBoundsUntracked is NOT a named Zig declaration: it is the
// anonymous struct that is the payload of Selection.Bounds.untracked.
type SelectionBoundsUntracked struct {
	start PageListPin
	end   PageListPin
}

// SelectionBoundsTracked is NOT a named Zig declaration: it is the
// anonymous struct that is the payload of Selection.Bounds.tracked.
type SelectionBoundsTracked struct {
	start *PageListPin
	end   *PageListPin
}

// SelectionBoundsTag is the tag of Selection.Bounds, values numbered from
// zero in the order listed. The constants are `SelectionBoundsTag<Variant>`
// because `SelectionBounds<Variant>` is the name of the payload type.
type SelectionBoundsTag uint8

const (
	SelectionBoundsTagUntracked SelectionBoundsTag = iota
	SelectionBoundsTagTracked
)

// String returns the Zig tag name (@tagName).
func (self SelectionBoundsTag) String() string { panic(notImplemented) }

// SelectionInit is Selection.init.
//
// Initialize a new selection with the given start and end pins on
// the screen. The screen will be used for pin tracking.
//
// Zig: pub fn init(start_pin: Pin, end_pin: Pin, rect: bool) Selection.
// The result is an untracked selection.
func SelectionInit(start_pin PageListPin, end_pin PageListPin, rect bool) Selection {
	panic(notImplemented)
}

// Deinit is Selection.deinit.
//
// Zig: pub fn deinit(self: Selection, s: *Screen) void. Kept: it untracks
// the pins of a tracked selection in the screen's page list; it does
// nothing for an untracked one.
func (self Selection) Deinit(s *Screen) { panic(notImplemented) }

// Eql is Selection.eql.
//
// Returns true if this selection is equal to another selection.
func (self Selection) Eql(other Selection) bool { panic(notImplemented) }

// StartPtr is Selection.startPtr.
//
// The starting pin of the selection. This is NOT ordered.
//
// Zig: pub fn startPtr(self: *Selection) *Pin.
func (self *Selection) StartPtr() *PageListPin { panic(notImplemented) }

// EndPtr is Selection.endPtr.
//
// The ending pin of the selection. This is NOT ordered.
//
// Zig: pub fn endPtr(self: *Selection) *Pin.
func (self *Selection) EndPtr() *PageListPin { panic(notImplemented) }

// Start is Selection.start.
func (self Selection) Start() PageListPin { panic(notImplemented) }

// End is Selection.end.
func (self Selection) End() PageListPin { panic(notImplemented) }

// Tracked is Selection.tracked.
//
// Returns true if this is a tracked selection.
//
// Zig: self is *const Selection.
func (self *Selection) Tracked() bool { panic(notImplemented) }

// Track is Selection.track.
//
// Convert this selection a tracked selection. It is asserted this is
// an untracked selection. The tracked selection is returned.
//
// Zig: pub fn track(self: *const Selection, s: *Screen)
// Allocator.Error!Selection. The error is ErrOutOfMemory (dcs.go).
func (self *Selection) Track(s *Screen) (Selection, error) { panic(notImplemented) }

// TopLeft is Selection.topLeft.
//
// Returns the top left point of the selection.
//
// Zig: s is *const Screen.
func (self Selection) TopLeft(s *Screen) PageListPin { panic(notImplemented) }

// BottomRight is Selection.bottomRight.
//
// Returns the bottom right point of the selection.
//
// Zig: s is *const Screen.
func (self Selection) BottomRight(s *Screen) PageListPin { panic(notImplemented) }

// SelectionOrder is Selection.Order.
//
// The order of the selection:
//
//   - forward: start(x, y) is before end(x, y) (top-left to bottom-right).
//   - reverse: end(x, y) is before start(x, y) (bottom-right to top-left).
//   - mirrored_[forward|reverse]: special, rectangle selections only (see below).
//
// For regular selections, the above also holds for top-right to bottom-left
// (forward) and bottom-left to top-right (reverse). However, for rectangle
// selections, both of these selections are *mirrored* as orientation
// operations only flip the x or y axis, not both. Depending on the y axis
// direction, this is either mirrored_forward or mirrored_reverse.
//
// Zig: enum { forward, reverse, mirrored_forward, mirrored_reverse },
// values numbered from zero in the order listed.
type SelectionOrder uint8

const (
	SelectionOrderForward SelectionOrder = iota
	SelectionOrderReverse
	SelectionOrderMirroredForward
	SelectionOrderMirroredReverse
)

// String returns the Zig tag name (@tagName).
func (self SelectionOrder) String() string { panic(notImplemented) }

// Order is Selection.order.
//
// Zig: pub fn order(self: Selection, s: *const Screen) Order.
func (self Selection) Order(s *Screen) SelectionOrder { panic(notImplemented) }

// Ordered is Selection.ordered.
//
// Returns the selection in the given order.
//
// The returned selection is always a new untracked selection.
//
// Note that only forward and reverse are useful desired orders for this
// function. All other orders act as if forward order was desired.
//
// Zig: s is *const Screen.
func (self Selection) Ordered(s *Screen, desired SelectionOrder) Selection {
	panic(notImplemented)
}

// Contains is Selection.contains.
//
// Returns true if the selection contains the given point.
//
// This recalculates top left and bottom right each call. If you have
// many points to check, it is cheaper to do the containment logic
// yourself and cache the topleft/bottomright.
//
// Zig: s is *const Screen.
func (self Selection) Contains(s *Screen, pin PageListPin) bool { panic(notImplemented) }

// ContainedRow is Selection.containedRow.
//
// Get a selection for a single row in the screen. This will return null
// if the row is not included in the selection.
//
// This is a very expensive operation. It has to traverse the linked list
// of pages for the top-left, bottom-right, and the given pin to find
// the coordinates. If you are calling this repeatedly, prefer
// `containedRowCached`.
//
// Zig: s is *const Screen; returns ?Selection.
func (self Selection) ContainedRow(s *Screen, pin PageListPin) Optional[Selection] {
	panic(notImplemented)
}

// ContainedRowCached is Selection.containedRowCached.
//
// Same as containedRow but useful if you're calling it repeatedly
// so that the pins can be cached across calls. Advanced.
//
// Zig: s is *const Screen; tl, br and p are point.Coordinate; returns
// ?Selection.
func (self Selection) ContainedRowCached(
	s *Screen,
	tl_pin PageListPin,
	br_pin PageListPin,
	pin PageListPin,
	tl PointCoordinate,
	br PointCoordinate,
	p PointCoordinate,
) Optional[Selection] {
	panic(notImplemented)
}

// SelectionAdjustment is Selection.Adjustment: possible adjustments to the
// selection.
//
// Zig: enum, values numbered from zero in the order listed.
type SelectionAdjustment uint8

const (
	SelectionAdjustmentLeft SelectionAdjustment = iota
	SelectionAdjustmentRight
	SelectionAdjustmentUp
	SelectionAdjustmentDown
	SelectionAdjustmentHome
	SelectionAdjustmentEnd
	SelectionAdjustmentPageUp
	SelectionAdjustmentPageDown
	SelectionAdjustmentBeginningOfLine
	SelectionAdjustmentEndOfLine
)

// String returns the Zig tag name (@tagName).
func (self SelectionAdjustment) String() string { panic(notImplemented) }

// Adjust is Selection.adjust.
//
// Adjust the selection by some given adjustment. An adjustment allows
// a selection to be expanded slightly left, right, up, down, etc.
//
// Zig: pub fn adjust(self: *Selection, s: *const Screen,
// adjustment: Adjustment) void.
func (self *Selection) Adjust(s *Screen, adjustment SelectionAdjustment) {
	panic(notImplemented)
}
