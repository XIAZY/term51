// Port of src/terminal/Selection.zig (Ghostty v1.3.1).
//
// Represents a single selection within the terminal (i.e. a highlight
// region).
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
//   - `SelectionBoundsTag`, the `String` methods and their name tables
//     (`selectionBoundsTagNames`, `selectionOrderNames`,
//     `selectionAdjustmentNames`).
//
// # Unwraps
//
// `s.pages.pointFromPin(.screen, pin).?.screen` is
// `PointFromPin(PointTagScreen, pin)`, `assert(x.Valid)` (Zig's `.?` traps
// on null; Go-only tests), then the `Screen` field. `pointFromPin` returns
// a point with the tag it was given, so the `.screen` access cannot fail
// and is not asserted. `s.pages.pin(...).?` is asserted the same way.
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
func (self SelectionBoundsTag) String() string { return selectionBoundsTagNames[self] }

// selectionBoundsTagNames are the tag names of SelectionBoundsTag, in order.
var selectionBoundsTagNames = [...]string{
	"untracked",
	"tracked",
}

// SelectionInit is Selection.init.
//
// Initialize a new selection with the given start and end pins on
// the screen. The screen will be used for pin tracking.
//
// Zig: pub fn init(start_pin: Pin, end_pin: Pin, rect: bool) Selection.
// The result is an untracked selection.
func SelectionInit(start_pin PageListPin, end_pin PageListPin, rect bool) Selection {
	return Selection{
		bounds: SelectionBounds{Tag: SelectionBoundsTagUntracked, Untracked: SelectionBoundsUntracked{
			start: start_pin,
			end:   end_pin,
		}},
		rectangle: rect,
	}
}

// Deinit is Selection.deinit.
//
// Zig: pub fn deinit(self: Selection, s: *Screen) void. Kept: it untracks
// the pins of a tracked selection in the screen's page list; it does
// nothing for an untracked one.
func (self Selection) Deinit(s *Screen) {
	switch self.bounds.Tag {
	case SelectionBoundsTagTracked:
		v := self.bounds.Tracked
		s.pages.UntrackPin(v.start)
		s.pages.UntrackPin(v.end)

	case SelectionBoundsTagUntracked:

	default:
		panic("unreachable")
	}
}

// Eql is Selection.eql.
//
// Returns true if this selection is equal to another selection.
func (self Selection) Eql(other Selection) bool {
	return self.Start().Eql(other.Start()) &&
		self.End().Eql(other.End()) &&
		self.rectangle == other.rectangle
}

// StartPtr is Selection.startPtr.
//
// The starting pin of the selection. This is NOT ordered.
//
// Zig: pub fn startPtr(self: *Selection) *Pin.
func (self *Selection) StartPtr() *PageListPin {
	switch self.bounds.Tag {
	case SelectionBoundsTagUntracked:
		v := &self.bounds.Untracked
		return &v.start
	case SelectionBoundsTagTracked:
		v := self.bounds.Tracked
		return v.start
	default:
		panic("unreachable")
	}
}

// EndPtr is Selection.endPtr.
//
// The ending pin of the selection. This is NOT ordered.
//
// Zig: pub fn endPtr(self: *Selection) *Pin.
func (self *Selection) EndPtr() *PageListPin {
	switch self.bounds.Tag {
	case SelectionBoundsTagUntracked:
		v := &self.bounds.Untracked
		return &v.end
	case SelectionBoundsTagTracked:
		v := self.bounds.Tracked
		return v.end
	default:
		panic("unreachable")
	}
}

// Start is Selection.start.
func (self Selection) Start() PageListPin {
	switch self.bounds.Tag {
	case SelectionBoundsTagUntracked:
		v := self.bounds.Untracked
		return v.start
	case SelectionBoundsTagTracked:
		v := self.bounds.Tracked
		return *v.start
	default:
		panic("unreachable")
	}
}

// End is Selection.end.
func (self Selection) End() PageListPin {
	switch self.bounds.Tag {
	case SelectionBoundsTagUntracked:
		v := self.bounds.Untracked
		return v.end
	case SelectionBoundsTagTracked:
		v := self.bounds.Tracked
		return *v.end
	default:
		panic("unreachable")
	}
}

// Tracked is Selection.tracked.
//
// Returns true if this is a tracked selection.
//
// Zig: self is *const Selection.
func (self *Selection) Tracked() bool {
	switch self.bounds.Tag {
	case SelectionBoundsTagUntracked:
		return false
	case SelectionBoundsTagTracked:
		return true
	default:
		panic("unreachable")
	}
}

// Track is Selection.track.
//
// Convert this selection a tracked selection. It is asserted this is
// an untracked selection. The tracked selection is returned.
//
// Zig: pub fn track(self: *const Selection, s: *Screen)
// Allocator.Error!Selection. The error is ErrOutOfMemory (dcs.go).
func (self *Selection) Track(s *Screen) (_ Selection, err error) {
	assert(!self.Tracked())

	// Track our pins
	start_pin := self.bounds.Untracked.start
	end_pin := self.bounds.Untracked.end
	tracked_start, err := s.pages.TrackPin(start_pin)
	if err != nil {
		return Selection{}, err
	}
	defer func() {
		// errdefer
		if err != nil {
			s.pages.UntrackPin(tracked_start)
		}
	}()
	tracked_end, err := s.pages.TrackPin(end_pin)
	if err != nil {
		return Selection{}, err
	}
	// Zig: errdefer s.pages.untrackPin(tracked_end); nothing after it can
	// fail, so it never runs.

	return Selection{
		bounds: SelectionBounds{Tag: SelectionBoundsTagTracked, Tracked: SelectionBoundsTracked{
			start: tracked_start,
			end:   tracked_end,
		}},
		rectangle: self.rectangle,
	}, nil
}

// TopLeft is Selection.topLeft.
//
// Returns the top left point of the selection.
//
// Zig: s is *const Screen.
func (self Selection) TopLeft(s *Screen) PageListPin {
	switch self.Order(s) {
	case SelectionOrderForward:
		return self.Start()
	case SelectionOrderReverse:
		return self.End()
	case SelectionOrderMirroredForward:
		// Zig: pin: { ... break :pin p; }
		p := self.Start()
		p.x = self.End().x
		return p
	case SelectionOrderMirroredReverse:
		// Zig: pin: { ... break :pin p; }
		p := self.End()
		p.x = self.Start().x
		return p
	default:
		panic("unreachable")
	}
}

// BottomRight is Selection.bottomRight.
//
// Returns the bottom right point of the selection.
//
// Zig: s is *const Screen.
func (self Selection) BottomRight(s *Screen) PageListPin {
	switch self.Order(s) {
	case SelectionOrderForward:
		return self.End()
	case SelectionOrderReverse:
		return self.Start()
	case SelectionOrderMirroredForward:
		// Zig: pin: { ... break :pin p; }
		p := self.End()
		p.x = self.Start().x
		return p
	case SelectionOrderMirroredReverse:
		// Zig: pin: { ... break :pin p; }
		p := self.Start()
		p.x = self.End().x
		return p
	default:
		panic("unreachable")
	}
}

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
func (self SelectionOrder) String() string { return selectionOrderNames[self] }

// selectionOrderNames are the tag names of SelectionOrder, in order.
var selectionOrderNames = [...]string{
	"forward",
	"reverse",
	"mirrored_forward",
	"mirrored_reverse",
}

// Order is Selection.order.
//
// Zig: pub fn order(self: Selection, s: *const Screen) Order.
func (self Selection) Order(s *Screen) SelectionOrder {
	start_pt_ := s.pages.PointFromPin(PointTagScreen, self.Start())
	assert(start_pt_.Valid) // .?
	start_pt := start_pt_.Value.Screen
	end_pt_ := s.pages.PointFromPin(PointTagScreen, self.End())
	assert(end_pt_.Valid) // .?
	end_pt := end_pt_.Value.Screen

	if self.rectangle {
		// Reverse (also handles single-column)
		if start_pt.y > end_pt.y && start_pt.x >= end_pt.x {
			return SelectionOrderReverse
		}
		if start_pt.y >= end_pt.y && start_pt.x > end_pt.x {
			return SelectionOrderReverse
		}

		// Mirror, bottom-left to top-right
		if start_pt.y > end_pt.y && start_pt.x < end_pt.x {
			return SelectionOrderMirroredReverse
		}

		// Mirror, top-right to bottom-left
		if start_pt.y < end_pt.y && start_pt.x > end_pt.x {
			return SelectionOrderMirroredForward
		}

		// Forward
		return SelectionOrderForward
	}

	if start_pt.y < end_pt.y {
		return SelectionOrderForward
	}
	if start_pt.y > end_pt.y {
		return SelectionOrderReverse
	}
	if start_pt.x <= end_pt.x {
		return SelectionOrderForward
	}
	return SelectionOrderReverse
}

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
	if self.Order(s) == desired {
		return SelectionInit(
			self.Start(),
			self.End(),
			self.rectangle,
		)
	}

	tl := self.TopLeft(s)
	br := self.BottomRight(s)
	switch desired {
	case SelectionOrderForward:
		return SelectionInit(tl, br, self.rectangle)
	case SelectionOrderReverse:
		return SelectionInit(br, tl, self.rectangle)
	default:
		return SelectionInit(tl, br, self.rectangle)
	}
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
func (self Selection) Contains(s *Screen, pin PageListPin) bool {
	tl_pin := self.TopLeft(s)
	br_pin := self.BottomRight(s)

	// This is definitely not very efficient. Low-hanging fruit to
	// improve this.
	tl_ := s.pages.PointFromPin(PointTagScreen, tl_pin)
	assert(tl_.Valid) // .?
	tl := tl_.Value.Screen
	br_ := s.pages.PointFromPin(PointTagScreen, br_pin)
	assert(br_.Valid) // .?
	br := br_.Value.Screen
	p_ := s.pages.PointFromPin(PointTagScreen, pin)
	assert(p_.Valid) // .?
	p := p_.Value.Screen

	// If we're in rectangle select, we can short-circuit with an easy check
	// here
	if self.rectangle {
		return p.y >= tl.y && p.y <= br.y && p.x >= tl.x && p.x <= br.x
	}

	// If tl/br are same line
	if tl.y == br.y {
		return p.y == tl.y &&
			p.x >= tl.x &&
			p.x <= br.x
	}

	// If on top line, just has to be left of X
	if p.y == tl.y {
		return p.x >= tl.x
	}

	// If on bottom line, just has to be right of X
	if p.y == br.y {
		return p.x <= br.x
	}

	// If between the top/bottom, always good.
	return p.y > tl.y && p.y < br.y
}

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
	tl_pin := self.TopLeft(s)
	br_pin := self.BottomRight(s)

	// This is definitely not very efficient. Low-hanging fruit to
	// improve this. Callers should prefer containedRowCached if they
	// can swing it.
	tl_ := s.pages.PointFromPin(PointTagScreen, tl_pin)
	assert(tl_.Valid) // .?
	tl := tl_.Value.Screen
	br_ := s.pages.PointFromPin(PointTagScreen, br_pin)
	assert(br_.Valid) // .?
	br := br_.Value.Screen
	p_ := s.pages.PointFromPin(PointTagScreen, pin)
	assert(p_.Valid) // .?
	p := p_.Value.Screen

	return self.ContainedRowCached(
		s,
		tl_pin,
		br_pin,
		pin,
		tl,
		br,
		p,
	)
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
	if p.y < tl.y || p.y > br.y {
		return None[Selection]()
	}

	// Rectangle case: we can return early as the x range will always be the
	// same. We've already validated that the row is in the selection.
	if self.rectangle {
		// Zig: start: { var copy: Pin = pin; copy.x = tl.x; break :start copy; }
		start := pin
		start.x = tl.x
		// Zig: end: { var copy: Pin = pin; copy.x = br.x; break :end copy; }
		end := pin
		end.x = br.x
		return Some(SelectionInit(
			start,
			end,
			true,
		))
	}

	if p.y == tl.y {
		// If the selection is JUST this line, return it as-is.
		if p.y == br.y {
			return Some(SelectionInit(tl_pin, br_pin, false))
		}

		// Selection top-left line matches only.
		// Zig: end: { var copy: Pin = pin; copy.x = s.pages.cols - 1; break :end copy; }
		end := pin
		assert(s.pages.cols > 0) // Zig: `cols - 1` traps on u16 underflow.
		end.x = s.pages.cols - 1
		return Some(SelectionInit(
			tl_pin,
			end,
			false,
		))
	}

	// Row is our bottom selection, so we return the selection from the
	// beginning of the line to the br. We know our selection is more than
	// one line (due to conditionals above)
	if p.y == br.y {
		assert(p.y != tl.y)
		// Zig: start: { var copy: Pin = pin; copy.x = 0; break :start copy; }
		start := pin
		start.x = 0
		return Some(SelectionInit(
			start,
			br_pin,
			false,
		))
	}

	// Row is somewhere between our selection lines so we return the full line.
	// Zig: start: { var copy: Pin = pin; copy.x = 0; break :start copy; }
	start := pin
	start.x = 0
	// Zig: end: { var copy: Pin = pin; copy.x = s.pages.cols - 1; break :end copy; }
	end := pin
	assert(s.pages.cols > 0) // Zig: `cols - 1` traps on u16 underflow.
	end.x = s.pages.cols - 1
	return Some(SelectionInit(
		start,
		end,
		false,
	))
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
func (self SelectionAdjustment) String() string { return selectionAdjustmentNames[self] }

// selectionAdjustmentNames are the tag names of SelectionAdjustment, in
// order.
var selectionAdjustmentNames = [...]string{
	"left",
	"right",
	"up",
	"down",
	"home",
	"end",
	"page_up",
	"page_down",
	"beginning_of_line",
	"end_of_line",
}

// Adjust is Selection.adjust.
//
// Adjust the selection by some given adjustment. An adjustment allows
// a selection to be expanded slightly left, right, up, down, etc.
//
// Zig: pub fn adjust(self: *Selection, s: *const Screen,
// adjustment: Adjustment) void.
func (self *Selection) Adjust(s *Screen, adjustment SelectionAdjustment) {
	// Note that we always adjust "end" because end always represents
	// the last point of the selection by mouse, not necessarily the
	// top/bottom visually. So this results in the correct behavior
	// whether the user drags up or down.
	end_pin := self.EndPtr()
	switch adjustment {
	case SelectionAdjustmentUp:
		if new_end := end_pin.Up(1); new_end.Valid {
			*end_pin = new_end.Value
		} else {
			self.Adjust(s, SelectionAdjustmentBeginningOfLine)
		}

	case SelectionAdjustmentDown:
		// Find the next non-blank row
		current := *end_pin
		// Zig: while (current.down(1)) |next| : (current = next) { ... } else { ... }
		for {
			next_ := current.Down(1)
			if !next_.Valid {
				// If we're at the bottom, just go to the end of the line
				self.Adjust(s, SelectionAdjustmentEndOfLine)
				break
			}
			next := next_.Value
			rac := next.RowAndCell()
			cells := next.node.data.GetCells(rac.row)
			if PageCellHasTextAny(cells) {
				*end_pin = next
				break
			}
			current = next
		}

	case SelectionAdjustmentLeft:
		it := end_pin.CellIterator(PageListDirectionLeftUp, None[PageListPin]())
		_ = it.Next()
		for {
			next_ := it.Next()
			if !next_.Valid {
				break
			}
			next := next_.Value
			rac := next.RowAndCell()
			if rac.cell.HasText() {
				*end_pin = next
				break
			}
		}

	case SelectionAdjustmentRight:
		// Step right, wrapping to the next row down at the start of each new line,
		// until we find a non-empty cell.
		it := end_pin.CellIterator(PageListDirectionRightDown, None[PageListPin]())
		_ = it.Next()
		for {
			next_ := it.Next()
			if !next_.Valid {
				break
			}
			next := next_.Value
			rac := next.RowAndCell()
			if rac.cell.HasText() {
				*end_pin = next
				break
			}
		}

	case SelectionAdjustmentPageUp:
		if new_end := end_pin.Up(uint(s.pages.rows)); new_end.Valid {
			*end_pin = new_end.Value
		} else {
			self.Adjust(s, SelectionAdjustmentHome)
		}

	case SelectionAdjustmentPageDown:
		if new_end := end_pin.Down(uint(s.pages.rows)); new_end.Valid {
			*end_pin = new_end.Value
		} else {
			self.Adjust(s, SelectionAdjustmentEnd)
		}

	case SelectionAdjustmentHome:
		home := s.pages.Pin(Point{Tag: PointTagScreen, Screen: PointCoordinate{
			x: 0,
			y: 0,
		}})
		assert(home.Valid) // .?
		*end_pin = home.Value

	case SelectionAdjustmentEnd:
		it := s.pages.RowIterator(
			PageListDirectionLeftUp,
			Point{Tag: PointTagScreen, Screen: PointCoordinate{}},
			None[Point](),
		)
		for {
			next_ := it.Next()
			if !next_.Valid {
				break
			}
			next := next_.Value
			rac := next.RowAndCell()
			cells := next.node.data.GetCells(rac.row)
			if PageCellHasTextAny(cells) {
				*end_pin = next
				// Zig: @intCast(cells.len - 1); cells is not empty here
				// (hasTextAny found a cell with text).
				end_pin.x = SizeCellCountInt(len(cells) - 1)
				break
			}
		}

	case SelectionAdjustmentBeginningOfLine:
		end_pin.x = 0

	case SelectionAdjustmentEndOfLine:
		assert(end_pin.node.data.size.cols > 0) // Zig: `cols - 1` traps on u16 underflow.
		end_pin.x = end_pin.node.data.size.cols - 1

	default:
		panic("unreachable")
	}
}
