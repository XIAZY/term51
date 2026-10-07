// Tests translated one for one from src/terminal/Selection.zig (Ghostty
// v1.3.1 plus the term51 additional tests). Each Go test carries the exact
// Zig title in a "// Zig: test" comment. Helpers are prefixed ss (the
// shared ones, ssScreenInit, ssPin, ssActive, ssScreen, ssHistory,
// ssYPin, ssYCoord, ssYExpectPin, ssYPages, ssYFirstRows, ssExpectPanic,
// are in ScreenSet_test.go).
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualStrings keep Zig's names and Zig's
//     (expected, actual) argument order. `try` on a call that returns an
//     error is zigTry(t, err); `x.?` is zigUnwrap or a nil check.
//   - `s.pages.pin(.{ .screen = .{ .x = x, .y = y } }).?` is ssYPin(t, s, x,
//     y) and `Selection.init(pin, pin, rect)` of two such pins is ssSelAt.
//     `expectEqual(point.Point{ .screen = .{ .x, .y } },
//     s.pages.pointFromPin(.screen, p).?)` is ssExpectScreenPoint.
//   - `expectEqual(a, b)` of two Selections compares the structs with ==
//     (as Zig compares the union tag and payload: both untracked pins by
//     node, x, y, garbage; tracked pins by address).
//   - `defer sel.deinit(&s)` is `defer sel.Deinit(s)`; the Go defer runs at
//     the end of the test function, not of the block. Where the order of
//     deinit matters (tracked selections) the block is a closure.
//   - Zig `Selection.init(...)` on a `const sel` that is later adjusted is
//     `var sel`; Go's pointer-receiver methods work on the variable.
//   - The allocation-failure tests (FailingAllocator) are not translated:
//     scripts/not-translated.txt.
//   - `@typeInfo` checks of unions/enums use package reflect and the Go
//     constants.

package terminal

import (
	"reflect"
	"strconv"
	"testing"
)

// ssSelAt is `Selection.init(s.pages.pin(.{ .screen = .{ .x = sx, .y = sy } }).?,
// s.pages.pin(.{ .screen = .{ .x = ex, .y = ey } }).?, rect)`.
func ssSelAt(t *testing.T, s *Screen, sx SizeCellCountInt, sy uint32, ex SizeCellCountInt, ey uint32, rect bool) Selection {
	t.Helper()
	return SelectionInit(ssYPin(t, s, sx, sy), ssYPin(t, s, ex, ey), rect)
}

// ssExpectScreenPoint is `try testing.expectEqual(point.Point{ .screen = .{
// .x = x, .y = y } }, s.pages.pointFromPin(.screen, p).?)`.
func ssExpectScreenPoint(t *testing.T, s *Screen, p PageListPin, x SizeCellCountInt, y uint32) {
	t.Helper()
	expectEqualPoint(t, ssScreen(x, y), zigUnwrap(t, s.Pages.PointFromPin(PointTagScreen, p)))
}

// Zig: test "Selection: adjust right"
func TestSelection_Selection_adjust_right(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("A1234\nB5678\nC1234\nD5678")))

	// Simple movement right
	{
		sel := ssSelAt(t, s, 5, 1, 3, 3, false)
		defer sel.Deinit(s)
		sel.Adjust(s, SelectionAdjustmentRight)

		ssExpectScreenPoint(t, s, sel.Start(), 5, 1)
		ssExpectScreenPoint(t, s, sel.End(), 4, 3)
	}

	// Already at end of the line.
	{
		sel := ssSelAt(t, s, 4, 1, 4, 2, false)
		defer sel.Deinit(s)
		sel.Adjust(s, SelectionAdjustmentRight)

		ssExpectScreenPoint(t, s, sel.Start(), 4, 1)
		ssExpectScreenPoint(t, s, sel.End(), 0, 3)
	}

	// Already at end of the screen
	{
		sel := ssSelAt(t, s, 5, 1, 4, 3, false)
		defer sel.Deinit(s)
		sel.Adjust(s, SelectionAdjustmentRight)

		ssExpectScreenPoint(t, s, sel.Start(), 5, 1)
		ssExpectScreenPoint(t, s, sel.End(), 4, 3)
	}
}

// Zig: test "Selection: adjust left"
func TestSelection_Selection_adjust_left(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("A1234\nB5678\nC1234\nD5678")))

	// Simple movement left
	{
		sel := ssSelAt(t, s, 5, 1, 3, 3, false)
		defer sel.Deinit(s)
		sel.Adjust(s, SelectionAdjustmentLeft)

		// Start line
		ssExpectScreenPoint(t, s, sel.Start(), 5, 1)
		ssExpectScreenPoint(t, s, sel.End(), 2, 3)
	}

	// Already at beginning of the line.
	{
		sel := ssSelAt(t, s, 5, 1, 0, 3, false)
		defer sel.Deinit(s)
		sel.Adjust(s, SelectionAdjustmentLeft)

		// Start line
		ssExpectScreenPoint(t, s, sel.Start(), 5, 1)
		ssExpectScreenPoint(t, s, sel.End(), 4, 2)
	}
}

// Zig: test "Selection: adjust left skips blanks"
func TestSelection_Selection_adjust_left_skips_blanks(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("A1234\nB5678\nC12\nD56")))

	// Same line
	{
		sel := ssSelAt(t, s, 5, 1, 4, 3, false)
		defer sel.Deinit(s)
		sel.Adjust(s, SelectionAdjustmentLeft)

		// Start line
		ssExpectScreenPoint(t, s, sel.Start(), 5, 1)
		ssExpectScreenPoint(t, s, sel.End(), 2, 3)
	}

	// Edge
	{
		sel := ssSelAt(t, s, 5, 1, 0, 3, false)
		defer sel.Deinit(s)
		sel.Adjust(s, SelectionAdjustmentLeft)

		// Start line
		ssExpectScreenPoint(t, s, sel.Start(), 5, 1)
		ssExpectScreenPoint(t, s, sel.End(), 2, 2)
	}
}

// Zig: test "Selection: adjust up"
func TestSelection_Selection_adjust_up(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("A\nB\nC\nD\nE")))

	// Not on the first line
	{
		sel := ssSelAt(t, s, 5, 1, 3, 3, false)
		defer sel.Deinit(s)
		sel.Adjust(s, SelectionAdjustmentUp)

		ssExpectScreenPoint(t, s, sel.Start(), 5, 1)
		ssExpectScreenPoint(t, s, sel.End(), 3, 2)
	}

	// On the first line
	{
		sel := ssSelAt(t, s, 5, 1, 3, 0, false)
		defer sel.Deinit(s)
		sel.Adjust(s, SelectionAdjustmentUp)

		ssExpectScreenPoint(t, s, sel.Start(), 5, 1)
		ssExpectScreenPoint(t, s, sel.End(), 0, 0)
	}
}

// Zig: test "Selection: adjust down"
func TestSelection_Selection_adjust_down(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("A\nB\nC\nD\nE")))

	// Not on the first line
	{
		sel := ssSelAt(t, s, 5, 1, 3, 3, false)
		defer sel.Deinit(s)
		sel.Adjust(s, SelectionAdjustmentDown)

		ssExpectScreenPoint(t, s, sel.Start(), 5, 1)
		ssExpectScreenPoint(t, s, sel.End(), 3, 4)
	}

	// On the last line
	{
		sel := ssSelAt(t, s, 4, 1, 3, 4, false)
		defer sel.Deinit(s)
		sel.Adjust(s, SelectionAdjustmentDown)

		ssExpectScreenPoint(t, s, sel.Start(), 4, 1)
		ssExpectScreenPoint(t, s, sel.End(), 9, 4)
	}
}

// Zig: test "Selection: adjust down with not full screen"
func TestSelection_Selection_adjust_down_with_not_full_screen(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("A\nB\nC")))

	// On the last line
	{
		sel := ssSelAt(t, s, 4, 1, 3, 2, false)
		defer sel.Deinit(s)
		sel.Adjust(s, SelectionAdjustmentDown)

		// Start line
		ssExpectScreenPoint(t, s, sel.Start(), 4, 1)
		ssExpectScreenPoint(t, s, sel.End(), 9, 2)
	}
}

// Zig: test "Selection: adjust home"
func TestSelection_Selection_adjust_home(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("A\nB\nC")))

	// On the last line
	{
		sel := ssSelAt(t, s, 4, 1, 1, 2, false)
		defer sel.Deinit(s)
		sel.Adjust(s, SelectionAdjustmentHome)

		// Start line
		ssExpectScreenPoint(t, s, sel.Start(), 4, 1)
		ssExpectScreenPoint(t, s, sel.End(), 0, 0)
	}
}

// Zig: test "Selection: adjust end with not full screen"
func TestSelection_Selection_adjust_end_with_not_full_screen(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("A\nB\nC")))

	// On the last line
	{
		sel := ssSelAt(t, s, 4, 0, 1, 1, false)
		defer sel.Deinit(s)
		sel.Adjust(s, SelectionAdjustmentEnd)

		// Start line
		ssExpectScreenPoint(t, s, sel.Start(), 4, 0)
		ssExpectScreenPoint(t, s, sel.End(), 9, 2)
	}
}

// Zig: test "Selection: adjust beginning of line"
func TestSelection_Selection_adjust_beginning_of_line(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 8, 10, 0)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("A12 B34\nC12 D34")))

	// Not at beginning of the line
	{
		sel := ssSelAt(t, s, 5, 1, 5, 1, false)
		defer sel.Deinit(s)
		sel.Adjust(s, SelectionAdjustmentBeginningOfLine)

		// Start line
		ssExpectScreenPoint(t, s, sel.Start(), 5, 1)
		ssExpectScreenPoint(t, s, sel.End(), 0, 1)
	}

	// Already at beginning of the line
	{
		sel := ssSelAt(t, s, 5, 1, 0, 1, false)
		defer sel.Deinit(s)
		sel.Adjust(s, SelectionAdjustmentBeginningOfLine)

		// Start line
		ssExpectScreenPoint(t, s, sel.Start(), 5, 1)
		ssExpectScreenPoint(t, s, sel.End(), 0, 1)
	}

	// End pin moves to start pin
	{
		sel := ssSelAt(t, s, 0, 1, 5, 1, false)
		defer sel.Deinit(s)
		sel.Adjust(s, SelectionAdjustmentBeginningOfLine)

		// Start line
		ssExpectScreenPoint(t, s, sel.Start(), 0, 1)
		ssExpectScreenPoint(t, s, sel.End(), 0, 1)
	}
}

// Zig: test "Selection: adjust end of line"
func TestSelection_Selection_adjust_end_of_line(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 8, 10, 0)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("A12 B34\nC12 D34")))

	// Not at end of the line
	{
		sel := ssSelAt(t, s, 1, 0, 1, 0, false)
		defer sel.Deinit(s)
		sel.Adjust(s, SelectionAdjustmentEndOfLine)

		ssExpectScreenPoint(t, s, sel.Start(), 1, 0)
		ssExpectScreenPoint(t, s, sel.End(), 7, 0)
	}

	// Already at end of the line
	{
		sel := ssSelAt(t, s, 1, 0, 7, 0, false)
		defer sel.Deinit(s)
		sel.Adjust(s, SelectionAdjustmentEndOfLine)

		ssExpectScreenPoint(t, s, sel.Start(), 1, 0)
		ssExpectScreenPoint(t, s, sel.End(), 7, 0)
	}

	// End pin moves to start pin
	{
		sel := ssSelAt(t, s, 7, 0, 1, 0, false)
		defer sel.Deinit(s)
		sel.Adjust(s, SelectionAdjustmentEndOfLine)

		// Start line
		ssExpectScreenPoint(t, s, sel.Start(), 7, 0)
		ssExpectScreenPoint(t, s, sel.End(), 7, 0)
	}
}

// Zig: test "Selection: order, standard"
func TestSelection_Selection_order_standard(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;

	s := ssScreenInit(t, 100, 100, 1)
	defer s.Deinit()

	{
		// forward, multi-line
		sel := ssSelAt(t, s, 2, 1, 2, 2, false)
		defer sel.Deinit(s)

		expect(t, sel.Order(s) == SelectionOrderForward, "sel.order(&s) == .forward")
	}
	{
		// reverse, multi-line
		sel := ssSelAt(t, s, 2, 2, 2, 1, false)
		defer sel.Deinit(s)

		expect(t, sel.Order(s) == SelectionOrderReverse, "sel.order(&s) == .reverse")
	}
	{
		// forward, same-line
		sel := ssSelAt(t, s, 2, 1, 3, 1, false)
		defer sel.Deinit(s)

		expect(t, sel.Order(s) == SelectionOrderForward, "sel.order(&s) == .forward")
	}
	{
		// forward, single char
		sel := ssSelAt(t, s, 2, 1, 2, 1, false)
		defer sel.Deinit(s)

		expect(t, sel.Order(s) == SelectionOrderForward, "sel.order(&s) == .forward")
	}
	{
		// reverse, single line
		sel := ssSelAt(t, s, 2, 1, 1, 1, false)
		defer sel.Deinit(s)

		expect(t, sel.Order(s) == SelectionOrderReverse, "sel.order(&s) == .reverse")
	}
}

// Zig: test "Selection: order, rectangle"
func TestSelection_Selection_order_rectangle(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;

	s := ssScreenInit(t, 100, 100, 1)
	defer s.Deinit()

	// Conventions:
	// TL - top left
	// BL - bottom left
	// TR - top right
	// BR - bottom right
	{
		// forward (TL -> BR)
		sel := ssSelAt(t, s, 1, 1, 2, 2, true)
		defer sel.Deinit(s)

		expect(t, sel.Order(s) == SelectionOrderForward, "sel.order(&s) == .forward")
	}
	{
		// reverse (BR -> TL)
		sel := ssSelAt(t, s, 2, 2, 1, 1, true)
		defer sel.Deinit(s)

		expect(t, sel.Order(s) == SelectionOrderReverse, "sel.order(&s) == .reverse")
	}
	{
		// mirrored_forward (TR -> BL)
		sel := ssSelAt(t, s, 3, 1, 1, 3, true)
		defer sel.Deinit(s)

		expect(t, sel.Order(s) == SelectionOrderMirroredForward, "sel.order(&s) == .mirrored_forward")
	}
	{
		// mirrored_reverse (BL -> TR)
		sel := ssSelAt(t, s, 1, 3, 3, 1, true)
		defer sel.Deinit(s)

		expect(t, sel.Order(s) == SelectionOrderMirroredReverse, "sel.order(&s) == .mirrored_reverse")
	}
	{
		// forward, single line (left -> right )
		sel := ssSelAt(t, s, 1, 1, 3, 1, true)
		defer sel.Deinit(s)

		expect(t, sel.Order(s) == SelectionOrderForward, "sel.order(&s) == .forward")
	}
	{
		// reverse, single line (right -> left)
		sel := ssSelAt(t, s, 3, 1, 1, 1, true)
		defer sel.Deinit(s)

		expect(t, sel.Order(s) == SelectionOrderReverse, "sel.order(&s) == .reverse")
	}
	{
		// forward, single column (top -> bottom)
		sel := ssSelAt(t, s, 2, 1, 2, 3, true)
		defer sel.Deinit(s)

		expect(t, sel.Order(s) == SelectionOrderForward, "sel.order(&s) == .forward")
	}
	{
		// reverse, single column (bottom -> top)
		sel := ssSelAt(t, s, 2, 3, 2, 1, true)
		defer sel.Deinit(s)

		expect(t, sel.Order(s) == SelectionOrderReverse, "sel.order(&s) == .reverse")
	}
	{
		// forward, single cell
		sel := ssSelAt(t, s, 1, 1, 1, 1, true)
		defer sel.Deinit(s)

		expect(t, sel.Order(s) == SelectionOrderForward, "sel.order(&s) == .forward")
	}
}

// Zig: test "topLeft"
func TestSelection_topLeft(t *testing.T) {
	testStdCapacity(t)

	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	{
		// forward
		sel := ssSelAt(t, s, 1, 1, 3, 1, true)
		defer sel.Deinit(s)
		tl := sel.TopLeft(s)
		expectEqualOptionalPoint(t, Some(ssScreen(1, 1)), s.Pages.PointFromPin(PointTagScreen, tl))
	}
	{
		// reverse
		sel := ssSelAt(t, s, 3, 1, 1, 1, true)
		defer sel.Deinit(s)
		tl := sel.TopLeft(s)
		expectEqualOptionalPoint(t, Some(ssScreen(1, 1)), s.Pages.PointFromPin(PointTagScreen, tl))
	}
	{
		// mirrored_forward
		sel := ssSelAt(t, s, 3, 1, 1, 3, true)
		defer sel.Deinit(s)
		tl := sel.TopLeft(s)
		expectEqualOptionalPoint(t, Some(ssScreen(1, 1)), s.Pages.PointFromPin(PointTagScreen, tl))
	}
	{
		// mirrored_reverse
		sel := ssSelAt(t, s, 1, 3, 3, 1, true)
		defer sel.Deinit(s)
		tl := sel.TopLeft(s)
		expectEqualOptionalPoint(t, Some(ssScreen(1, 1)), s.Pages.PointFromPin(PointTagScreen, tl))
	}
}

// Zig: test "bottomRight"
func TestSelection_bottomRight(t *testing.T) {
	testStdCapacity(t)

	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	{
		// forward
		sel := ssSelAt(t, s, 1, 1, 3, 1, false)
		defer sel.Deinit(s)
		br := sel.BottomRight(s)
		expectEqualOptionalPoint(t, Some(ssScreen(3, 1)), s.Pages.PointFromPin(PointTagScreen, br))
	}
	{
		// reverse
		sel := ssSelAt(t, s, 3, 1, 1, 1, false)
		defer sel.Deinit(s)
		br := sel.BottomRight(s)
		expectEqualOptionalPoint(t, Some(ssScreen(3, 1)), s.Pages.PointFromPin(PointTagScreen, br))
	}
	{
		// mirrored_forward
		sel := ssSelAt(t, s, 3, 1, 1, 3, true)
		defer sel.Deinit(s)
		br := sel.BottomRight(s)
		expectEqualOptionalPoint(t, Some(ssScreen(3, 3)), s.Pages.PointFromPin(PointTagScreen, br))
	}
	{
		// mirrored_reverse
		sel := ssSelAt(t, s, 1, 3, 3, 1, true)
		defer sel.Deinit(s)
		br := sel.BottomRight(s)
		expectEqualOptionalPoint(t, Some(ssScreen(3, 3)), s.Pages.PointFromPin(PointTagScreen, br))
	}
}

// Zig: test "ordered"
func TestSelection_ordered(t *testing.T) {
	testStdCapacity(t)

	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	{
		// forward
		sel := ssSelAt(t, s, 1, 1, 3, 1, false)
		sel_reverse := ssSelAt(t, s, 3, 1, 1, 1, false)
		expect(t, sel.Ordered(s, SelectionOrderForward).Eql(sel), "sel.ordered(&s, .forward).eql(sel)")
		expect(t, sel.Ordered(s, SelectionOrderReverse).Eql(sel_reverse), "sel.ordered(&s, .reverse).eql(sel_reverse)")
		expect(t, sel.Ordered(s, SelectionOrderMirroredForward).Eql(sel), "sel.ordered(&s, .mirrored_forward).eql(sel)")
	}
	{
		// reverse
		sel := ssSelAt(t, s, 3, 1, 1, 1, false)
		sel_forward := ssSelAt(t, s, 1, 1, 3, 1, false)
		expect(t, sel.Ordered(s, SelectionOrderForward).Eql(sel_forward), "sel.ordered(&s, .forward).eql(sel_forward)")
		expect(t, sel.Ordered(s, SelectionOrderReverse).Eql(sel), "sel.ordered(&s, .reverse).eql(sel)")
		expect(t, sel.Ordered(s, SelectionOrderMirroredForward).Eql(sel_forward), "sel.ordered(&s, .mirrored_forward).eql(sel_forward)")
	}
	{
		// mirrored_forward
		sel := ssSelAt(t, s, 3, 1, 1, 3, true)
		sel_forward := ssSelAt(t, s, 1, 1, 3, 3, true)
		sel_reverse := ssSelAt(t, s, 3, 3, 1, 1, true)
		expect(t, sel.Ordered(s, SelectionOrderForward).Eql(sel_forward), "sel.ordered(&s, .forward).eql(sel_forward)")
		expect(t, sel.Ordered(s, SelectionOrderReverse).Eql(sel_reverse), "sel.ordered(&s, .reverse).eql(sel_reverse)")
		expect(t, sel.Ordered(s, SelectionOrderMirroredReverse).Eql(sel_forward), "sel.ordered(&s, .mirrored_reverse).eql(sel_forward)")
	}
	{
		// mirrored_reverse
		sel := ssSelAt(t, s, 1, 3, 3, 1, true)
		sel_forward := ssSelAt(t, s, 1, 1, 3, 3, true)
		sel_reverse := ssSelAt(t, s, 3, 3, 1, 1, true)
		expect(t, sel.Ordered(s, SelectionOrderForward).Eql(sel_forward), "sel.ordered(&s, .forward).eql(sel_forward)")
		expect(t, sel.Ordered(s, SelectionOrderReverse).Eql(sel_reverse), "sel.ordered(&s, .reverse).eql(sel_reverse)")
		expect(t, sel.Ordered(s, SelectionOrderMirroredForward).Eql(sel_forward), "sel.ordered(&s, .mirrored_forward).eql(sel_forward)")
	}
}

// Zig: test "Selection: contains"
func TestSelection_Selection_contains(t *testing.T) {
	testStdCapacity(t)

	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	{
		sel := ssSelAt(t, s, 5, 1, 3, 2, false)

		expect(t, sel.Contains(s, ssYPin(t, s, 6, 1)), "sel.contains(&s, (6, 1))")
		expect(t, sel.Contains(s, ssYPin(t, s, 1, 2)), "sel.contains(&s, (1, 2))")
		expect(t, !sel.Contains(s, ssYPin(t, s, 1, 1)), "!sel.contains(&s, (1, 1))")
		expect(t, !sel.Contains(s, ssYPin(t, s, 5, 2)), "!sel.contains(&s, (5, 2))")
	}

	// Reverse
	{
		sel := ssSelAt(t, s, 3, 2, 5, 1, false)

		expect(t, sel.Contains(s, ssYPin(t, s, 6, 1)), "sel.contains(&s, (6, 1))")
		expect(t, sel.Contains(s, ssYPin(t, s, 1, 2)), "sel.contains(&s, (1, 2))")
		expect(t, !sel.Contains(s, ssYPin(t, s, 1, 1)), "!sel.contains(&s, (1, 1))")
		expect(t, !sel.Contains(s, ssYPin(t, s, 5, 2)), "!sel.contains(&s, (5, 2))")
	}

	// Single line
	{
		sel := ssSelAt(t, s, 5, 1, 8, 1, false)

		expect(t, sel.Contains(s, ssYPin(t, s, 6, 1)), "sel.contains(&s, (6, 1))")
		expect(t, !sel.Contains(s, ssYPin(t, s, 2, 1)), "!sel.contains(&s, (2, 1))")
		expect(t, !sel.Contains(s, ssYPin(t, s, 9, 1)), "!sel.contains(&s, (9, 1))")
	}
}

// Zig: test "Selection: contains, rectangle"
func TestSelection_Selection_contains_rectangle(t *testing.T) {
	testStdCapacity(t)

	s := ssScreenInit(t, 15, 15, 0)
	defer s.Deinit()
	{
		sel := ssSelAt(t, s, 3, 3, 7, 9, true)

		expect(t, sel.Contains(s, ssYPin(t, s, 5, 6)), "sel.contains(&s, (5, 6))") // Center
		expect(t, sel.Contains(s, ssYPin(t, s, 3, 6)), "sel.contains(&s, (3, 6))") // Left border
		expect(t, sel.Contains(s, ssYPin(t, s, 7, 6)), "sel.contains(&s, (7, 6))") // Right border
		expect(t, sel.Contains(s, ssYPin(t, s, 5, 3)), "sel.contains(&s, (5, 3))") // Top border
		expect(t, sel.Contains(s, ssYPin(t, s, 5, 9)), "sel.contains(&s, (5, 9))") // Bottom border

		expect(t, !sel.Contains(s, ssYPin(t, s, 5, 2)), "!sel.contains(&s, (5, 2))")   // Above center
		expect(t, !sel.Contains(s, ssYPin(t, s, 5, 10)), "!sel.contains(&s, (5, 10))") // Below center
		expect(t, !sel.Contains(s, ssYPin(t, s, 2, 6)), "!sel.contains(&s, (2, 6))")   // Left center
		expect(t, !sel.Contains(s, ssYPin(t, s, 8, 6)), "!sel.contains(&s, (8, 6))")   // Right center
		expect(t, !sel.Contains(s, ssYPin(t, s, 8, 3)), "!sel.contains(&s, (8, 3))")   // Just right of top right
		expect(t, !sel.Contains(s, ssYPin(t, s, 2, 9)), "!sel.contains(&s, (2, 9))")   // Just left of bottom left
	}

	// Reverse
	{
		sel := ssSelAt(t, s, 7, 9, 3, 3, true)

		expect(t, sel.Contains(s, ssYPin(t, s, 5, 6)), "sel.contains(&s, (5, 6))") // Center
		expect(t, sel.Contains(s, ssYPin(t, s, 3, 6)), "sel.contains(&s, (3, 6))") // Left border
		expect(t, sel.Contains(s, ssYPin(t, s, 7, 6)), "sel.contains(&s, (7, 6))") // Right border
		expect(t, sel.Contains(s, ssYPin(t, s, 5, 3)), "sel.contains(&s, (5, 3))") // Top border
		expect(t, sel.Contains(s, ssYPin(t, s, 5, 9)), "sel.contains(&s, (5, 9))") // Bottom border

		expect(t, !sel.Contains(s, ssYPin(t, s, 5, 2)), "!sel.contains(&s, (5, 2))")   // Above center
		expect(t, !sel.Contains(s, ssYPin(t, s, 5, 10)), "!sel.contains(&s, (5, 10))") // Below center
		expect(t, !sel.Contains(s, ssYPin(t, s, 2, 6)), "!sel.contains(&s, (2, 6))")   // Left center
		expect(t, !sel.Contains(s, ssYPin(t, s, 8, 6)), "!sel.contains(&s, (8, 6))")   // Right center
		expect(t, !sel.Contains(s, ssYPin(t, s, 8, 3)), "!sel.contains(&s, (8, 3))")   // Just right of top right
		expect(t, !sel.Contains(s, ssYPin(t, s, 2, 9)), "!sel.contains(&s, (2, 9))")   // Just left of bottom left
	}

	// Single line
	// NOTE: This is the same as normal selection but we just do it for brevity
	{
		sel := ssSelAt(t, s, 5, 1, 10, 1, true)

		expect(t, sel.Contains(s, ssYPin(t, s, 6, 1)), "sel.contains(&s, (6, 1))")
		expect(t, !sel.Contains(s, ssYPin(t, s, 2, 1)), "!sel.contains(&s, (2, 1))")
		expect(t, !sel.Contains(s, ssYPin(t, s, 12, 1)), "!sel.contains(&s, (12, 1))")
	}
}

// Zig: test "Selection: containedRow"
func TestSelection_Selection_containedRow(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 5, 0)
	defer s.Deinit()

	{
		sel := ssSelAt(t, s, 5, 1, 3, 3, false)

		// Not contained
		expect(t, !sel.ContainedRow(s, ssYPin(t, s, 1, 4)).Valid, "sel.containedRow(&s, (1, 4)) == null")

		// Start line
		expectEqual(t, SelectionInit(
			sel.Start(),
			ssYPin(t, s, s.Pages.cols-1, 1),
			false,
		), zigUnwrap(t, sel.ContainedRow(s, ssYPin(t, s, 1, 1))))

		// End line
		expectEqual(t, SelectionInit(
			ssYPin(t, s, 0, 3),
			sel.End(),
			false,
		), zigUnwrap(t, sel.ContainedRow(s, ssYPin(t, s, 2, 3))))

		// Middle line
		expectEqual(t, SelectionInit(
			ssYPin(t, s, 0, 2),
			ssYPin(t, s, s.Pages.cols-1, 2),
			false,
		), zigUnwrap(t, sel.ContainedRow(s, ssYPin(t, s, 2, 2))))
	}

	// Rectangle
	{
		sel := ssSelAt(t, s, 3, 1, 6, 3, true)

		// Not contained
		expect(t, !sel.ContainedRow(s, ssYPin(t, s, 1, 4)).Valid, "sel.containedRow(&s, (1, 4)) == null")

		// Start line
		expectEqual(t, SelectionInit(
			ssYPin(t, s, 3, 1),
			ssYPin(t, s, 6, 1),
			true,
		), zigUnwrap(t, sel.ContainedRow(s, ssYPin(t, s, 1, 1))))

		// End line
		expectEqual(t, SelectionInit(
			ssYPin(t, s, 3, 3),
			ssYPin(t, s, 6, 3),
			true,
		), zigUnwrap(t, sel.ContainedRow(s, ssYPin(t, s, 2, 3))))

		// Middle line
		expectEqual(t, SelectionInit(
			ssYPin(t, s, 3, 2),
			ssYPin(t, s, 6, 2),
			true,
		), zigUnwrap(t, sel.ContainedRow(s, ssYPin(t, s, 2, 2))))
	}

	// Single-line selection
	{
		sel := ssSelAt(t, s, 2, 1, 6, 1, false)

		// Not contained
		expect(t, !sel.ContainedRow(s, ssYPin(t, s, 1, 0)).Valid, "sel.containedRow(&s, (1, 0)) == null")
		expect(t, !sel.ContainedRow(s, ssYPin(t, s, 1, 2)).Valid, "sel.containedRow(&s, (1, 2)) == null")

		// Contained
		expectEqual(t, sel, zigUnwrap(t, sel.ContainedRow(s, ssYPin(t, s, 1, 1))))
	}
}

// ---------------------------------------------------------------------------
// term51 tests (helpers prefixed ssY).
// ---------------------------------------------------------------------------

// ssYExpectSel is term51YExpectSel: asserts an untracked selection with the
// given start/end/rectangle.
func ssYExpectSel(t *testing.T, s *Screen, sel Selection, sx SizeCellCountInt, sy uint32, ex SizeCellCountInt, ey uint32, rect bool) {
	t.Helper()
	expect(t, !sel.Tracked(), "!sel.tracked()")
	ssYExpectPin(t, s, sel.Start(), sx, sy)
	ssYExpectPin(t, s, sel.End(), ex, ey)
	expectEqual(t, rect, sel.Rectangle)
}

// ssYSetText is term51YSetText: writes narrow ASCII cells directly into the
// screen.
func ssYSetText(t *testing.T, s *Screen, x SizeCellCountInt, y uint32, text string) {
	t.Helper()
	p := ssYPin(t, s, x, y)
	for i := 0; i < len(text); i++ {
		c := text[i]
		*p.RowAndCell().cell = PageCell{
			content_tag: PageCellContentTagCodepoint,
			content:     PageCellContent{codepoint: uint32(c)},
		}
		p.X += 1
	}
}

// ssYSetWide is term51YSetWide: writes a wide character (head at x, spacer
// tail at x + 1).
func ssYSetWide(t *testing.T, s *Screen, x SizeCellCountInt, y uint32, cp uint32) {
	t.Helper()
	p := ssYPin(t, s, x, y)
	*p.RowAndCell().cell = PageCell{
		content_tag: PageCellContentTagCodepoint,
		content:     PageCellContent{codepoint: cp},
		wide:        PageCellWideWide,
	}
	p.X += 1
	*p.RowAndCell().cell = PageCell{
		content_tag: PageCellContentTagCodepoint,
		content:     PageCellContent{codepoint: 0},
		wide:        PageCellWideSpacerTail,
	}
}

// ssYSetBg is term51YSetBg: a cell that is only a background color: it has
// no text.
func ssYSetBg(t *testing.T, s *Screen, x SizeCellCountInt, y uint32) {
	t.Helper()
	*ssYPin(t, s, x, y).RowAndCell().cell = PageCell{
		content_tag: PageCellContentTagBgColorRgb,
		content:     PageCellContent{color_rgb: PageCellRGB{R: 1, G: 2, B: 3}},
	}
}

// ssYHistory is term51YHistory: a screen whose history holds `lines` lines
// "L0", "L1", ... (one per row).
func ssYHistory(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt, lines int) *Screen {
	t.Helper()
	s := ssScreenInit(t, cols, rows, 1000)
	for i := 0; i < lines; i++ {
		if i > 0 {
			zigTry(t, s.TestWriteString([]uint8("\n")))
		}
		zigTry(t, s.TestWriteString([]uint8("L"+strconv.Itoa(i))))
	}
	return s
}

// ssYPt is term51YPt.
type ssYPt struct {
	x    SizeCellCountInt
	y    uint32
	want bool
}

// ssYCheckContains is term51YCheckContains.
func ssYCheckContains(t *testing.T, s *Screen, sel Selection, pts []ssYPt) {
	t.Helper()
	for _, p := range pts {
		got := sel.Contains(s, ssYPin(t, s, p.x, p.y))
		if got != p.want {
			t.Logf("contains case x=%d y=%d want=%v", p.x, p.y, p.want)
		}
		expectEqual(t, p.want, got)
	}
}

// ssYEndCase is term51YEndCase.
type ssYEndCase struct {
	ex SizeCellCountInt
	ey uint32
	nx SizeCellCountInt
	ny uint32
}

// ssYCheckAdjust is term51YCheckAdjust: for each case, selection (7,0) ->
// (ex,ey), adjust, expect the end at (nx,ny) and the start untouched.
func ssYCheckAdjust(t *testing.T, s *Screen, adj SelectionAdjustment, cases []ssYEndCase) {
	t.Helper()
	for _, c := range cases {
		func() {
			// errdefer std.debug.print("adjust {s} case end=({d},{d})\n", ...)
			defer func() {
				if t.Failed() {
					t.Logf("adjust %v case end=(%d,%d)", adj, c.ex, c.ey)
				}
			}()
			sel := SelectionInit(ssYPin(t, s, 7, 0), ssYPin(t, s, c.ex, c.ey), false)
			sel.Adjust(s, adj)
			ssYExpectSel(t, s, sel, 7, 0, c.nx, c.ny, false)
		}()
	}
}

// Zig: test "term51: Selection.Bounds: union tags and payloads"
func TestSelection_term51_Selection_Bounds_union_tags_and_payloads(t *testing.T) {
	testStdCapacity(t)
	// The Go union is the struct { Tag; Untracked; Tracked }: the payload
	// fields are the fields after Tag.
	info := reflect.TypeOf(SelectionBounds{})
	expectEqual(t, 2, info.NumField()-1)
	expectEqualStrings(t, "Untracked", info.Field(1).Name)
	expectEqualStrings(t, "Tracked", info.Field(2).Name)
	expect(t, info.Field(2).Type != info.Field(1).Type, "info.fields[1].type != info.fields[0].type")
	expectEqual(t, SelectionBoundsTag(0), SelectionBoundsTagUntracked)
	expectEqual(t, SelectionBoundsTag(1), SelectionBoundsTagTracked)

	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	sel := SelectionInit(ssYPin(t, s, 1, 1), ssYPin(t, s, 2, 2), false)
	expect(t, sel.Bounds.Tag == SelectionBoundsTagUntracked, "std.meta.activeTag(sel.bounds) == .untracked")
	var err error
	sel, err = sel.Track(s)
	zigTry(t, err)
	defer sel.Deinit(s)
	expect(t, sel.Bounds.Tag == SelectionBoundsTagTracked, "std.meta.activeTag(sel.bounds) == .tracked")
}

// Zig: test "term51: Selection.Order: variants and order"
func TestSelection_term51_Selection_Order_variants_and_order(t *testing.T) {
	testStdCapacity(t)
	// Four variants, numbered from zero in this order.
	expectEqual(t, 4, int(SelectionOrderMirroredReverse)+1)
	expectEqual(t, SelectionOrder(0), SelectionOrderForward)
	expectEqual(t, SelectionOrder(1), SelectionOrderReverse)
	expectEqual(t, SelectionOrder(2), SelectionOrderMirroredForward)
	expectEqual(t, SelectionOrder(3), SelectionOrderMirroredReverse)
	expectEqualStrings(t, "forward", SelectionOrder(0).String())
	expectEqualStrings(t, "reverse", SelectionOrder(1).String())
	expectEqualStrings(t, "mirrored_forward", SelectionOrder(2).String())
	expectEqualStrings(t, "mirrored_reverse", SelectionOrder(3).String())
}

// Zig: test "term51: Selection.Adjustment: variants and order"
func TestSelection_term51_Selection_Adjustment_variants_and_order(t *testing.T) {
	testStdCapacity(t)
	names := []struct {
		name string
		adj  SelectionAdjustment
	}{
		{"left", SelectionAdjustmentLeft}, {"right", SelectionAdjustmentRight},
		{"up", SelectionAdjustmentUp}, {"down", SelectionAdjustmentDown},
		{"home", SelectionAdjustmentHome}, {"end", SelectionAdjustmentEnd},
		{"page_up", SelectionAdjustmentPageUp}, {"page_down", SelectionAdjustmentPageDown},
		{"beginning_of_line", SelectionAdjustmentBeginningOfLine}, {"end_of_line", SelectionAdjustmentEndOfLine},
	}
	// Ten variants.
	expectEqual(t, len(names), int(SelectionAdjustmentEndOfLine)+1)
	for i, n := range names {
		expectEqualStrings(t, n.name, SelectionAdjustment(i).String())
		expectEqual(t, i, int(n.adj))
	}
}

// Zig: test "term51: init: stores untracked pins and the rectangle flag"
func TestSelection_term51_init_stores_untracked_pins_and_the_rectangle_flag(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	base := s.Pages.CountTrackedPins()

	a := ssYPin(t, s, 2, 1)
	b := ssYPin(t, s, 7, 4)

	linear := SelectionInit(a, b, false)
	expect(t, !linear.Tracked(), "!linear.tracked()")
	expect(t, linear.Bounds.Tag == SelectionBoundsTagUntracked, "std.meta.activeTag(linear.bounds) == .untracked")
	expect(t, linear.Start().Eql(a), "linear.start().eql(a)")
	expect(t, linear.End().Eql(b), "linear.end().eql(b)")
	expect(t, !linear.Rectangle, "!linear.rectangle")

	rect := SelectionInit(a, b, true)
	expect(t, rect.Rectangle, "rect.rectangle")
	expect(t, rect.Start().Eql(a), "rect.start().eql(a)")
	expect(t, rect.End().Eql(b), "rect.end().eql(b)")

	// Pins are stored as given: no reordering, even for a backwards range.
	back := SelectionInit(b, a, false)
	expect(t, back.Start().Eql(b), "back.start().eql(b)")
	expect(t, back.End().Eql(a), "back.end().eql(a)")

	// One-cell selection.
	one := SelectionInit(a, a, false)
	expect(t, one.Start().Eql(one.End()), "one.start().eql(one.end())")

	// Nothing was tracked.
	expectEqual(t, base, s.Pages.CountTrackedPins())
}

// Zig: test "term51: deinit: untracked does nothing, tracked releases both pins"
func TestSelection_term51_deinit_untracked_does_nothing_tracked_releases_both_pins(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	base := s.Pages.CountTrackedPins()

	sel := SelectionInit(ssYPin(t, s, 2, 1), ssYPin(t, s, 7, 4), false)
	sel.Deinit(s)
	sel.Deinit(s)
	expectEqual(t, base, s.Pages.CountTrackedPins())

	tsel, err := sel.Track(s)
	zigTry(t, err)
	expectEqual(t, base+2, s.Pages.CountTrackedPins())
	tsel.Deinit(s)
	expectEqual(t, base, s.Pages.CountTrackedPins())

	// A one-cell tracked selection still owns two separate pins.
	one := SelectionInit(ssYPin(t, s, 3, 3), ssYPin(t, s, 3, 3), true)
	tone, err := one.Track(s)
	zigTry(t, err)
	expectEqual(t, base+2, s.Pages.CountTrackedPins())
	tone.Deinit(s)
	expectEqual(t, base, s.Pages.CountTrackedPins())
}

// Zig: test "term51: tracked: false for untracked, true after track"
func TestSelection_term51_tracked_false_for_untracked_true_after_track(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	sel := SelectionInit(ssYPin(t, s, 2, 1), ssYPin(t, s, 7, 4), false)
	expect(t, !sel.Tracked(), "!sel.tracked()")
	tsel, err := sel.Track(s)
	zigTry(t, err)
	defer tsel.Deinit(s)
	expect(t, tsel.Tracked(), "tsel.tracked()")
	// The original value is unchanged.
	expect(t, !sel.Tracked(), "!sel.tracked()")
}

// Zig: test "term51: track: tracked copy keeps pins and flag, original stays untracked"
func TestSelection_term51_track_tracked_copy_keeps_pins_and_flag_original_stays_untracked(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	base := s.Pages.CountTrackedPins()

	for _, rect := range []bool{false, true} {
		sel := SelectionInit(ssYPin(t, s, 2, 1), ssYPin(t, s, 7, 4), rect)
		tsel, err := sel.Track(s)
		zigTry(t, err)
		expect(t, tsel.Tracked(), "tsel.tracked()")
		expectEqual(t, rect, tsel.Rectangle)
		ssYExpectPin(t, s, tsel.Start(), 2, 1)
		ssYExpectPin(t, s, tsel.End(), 7, 4)
		expect(t, tsel.Eql(sel), "tsel.eql(sel)")
		expectEqual(t, base+2, s.Pages.CountTrackedPins())

		// Distinct heap pins, not pointers into the untracked selection.
		expect(t, tsel.Bounds.Tracked.Start != tsel.Bounds.Tracked.End, "tsel.bounds.tracked.start != tsel.bounds.tracked.end")
		expect(t, !sel.Tracked(), "!sel.tracked()")
		expect(t, sel.Bounds.Untracked.Start.Eql(ssYPin(t, s, 2, 1)), "sel.bounds.untracked.start.eql(term51YPin(&s, 2, 1))")

		// Moving the tracked pin does not move the original.
		tsel.Bounds.Tracked.End.X = 9
		ssYExpectPin(t, s, sel.End(), 7, 4)
		ssYExpectPin(t, s, tsel.End(), 9, 4)

		tsel.Deinit(s)
		expectEqual(t, base, s.Pages.CountTrackedPins())
	}
}

// Zig: test "term51: track: tracked pins follow erased rows"
func TestSelection_term51_track_tracked_pins_follow_erased_rows(t *testing.T) {
	testStdCapacity(t)
	s := ssYHistory(t, 10, 3, 12)
	defer s.Deinit()
	expectEqual(t, uint(12), s.Pages.total_rows)

	sel := SelectionInit(ssYPin(t, s, 1, 3), ssYPin(t, s, 2, 6), false)
	tsel, err := sel.Track(s)
	zigTry(t, err)
	defer tsel.Deinit(s)

	// Erase history rows 0 and 1: both pins shift up by two rows.
	s.Pages.EraseRows(ssScreen(0, 0), Some(ssScreen(0, 1)))
	expectEqual(t, uint(10), s.Pages.total_rows)
	ssYExpectPin(t, s, tsel.Start(), 1, 1)
	ssYExpectPin(t, s, tsel.End(), 2, 4)
	// The untracked original is not updated (it is stale by design).
	expectEqual(t, sel.Start().Y, SizeCellCountInt(3))

	// Erase rows 0 and 1 again: the start pin is inside the erased range and
	// goes to the top-left of the page, the end shifts.
	s.Pages.EraseRows(ssScreen(0, 0), Some(ssScreen(0, 1)))
	ssYExpectPin(t, s, tsel.Start(), 0, 0)
	ssYExpectPin(t, s, tsel.End(), 2, 2)
	expectEqual(t, SelectionOrderForward, tsel.Order(s))
}

// Zig: test "term51: track: tracked pins follow an erased first page"
func TestSelection_term51_track_tracked_pins_follow_an_erased_first_page(t *testing.T) {
	testStdCapacity(t)
	s := ssYPages(t, 2)
	defer s.Deinit()
	n := ssYFirstRows(s)

	sel := SelectionInit(ssYPin(t, s, 3, 2), ssYPin(t, s, 4, n+1), false)
	tsel, err := sel.Track(s)
	zigTry(t, err)
	defer tsel.Deinit(s)

	s.Pages.EraseRows(ssScreen(0, 0), Some(ssScreen(0, n-1)))
	expectEqual(t, uint(1), s.Pages.TotalPages())
	// Start was inside the erased page: moved to the top-left of the next page.
	ssYExpectPin(t, s, tsel.Start(), 0, 0)
	ssYExpectPin(t, s, tsel.End(), 4, 1)
	expect(t, s.Pages.pages.first != nil, "s.pages.pages.first.?")
	expect(t, tsel.Start().Node == s.Pages.pages.first, "tsel.start().node == s.pages.pages.first.?")
	expect(t, tsel.End().Node == s.Pages.pages.first, "tsel.end().node == s.pages.pages.first.?")
}

// Zig: test "term51: start: reads the current value of tracked pins"
func TestSelection_term51_start_reads_the_current_value_of_tracked_pins(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	sel := SelectionInit(ssYPin(t, s, 2, 1), ssYPin(t, s, 7, 4), false)
	ssYExpectPin(t, s, sel.Start(), 2, 1)
	tsel, err := sel.Track(s)
	zigTry(t, err)
	defer tsel.Deinit(s)
	tsel.Bounds.Tracked.Start.X = 5
	tsel.Bounds.Tracked.Start.Y = 3
	ssYExpectPin(t, s, tsel.Start(), 5, 3)
	// end() is independent of start().
	ssYExpectPin(t, s, tsel.End(), 7, 4)
}

// Zig: test "term51: end: reads the current value of tracked pins"
func TestSelection_term51_end_reads_the_current_value_of_tracked_pins(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	sel := SelectionInit(ssYPin(t, s, 2, 1), ssYPin(t, s, 7, 4), false)
	ssYExpectPin(t, s, sel.End(), 7, 4)
	tsel, err := sel.Track(s)
	zigTry(t, err)
	defer tsel.Deinit(s)
	tsel.Bounds.Tracked.End.X = 0
	tsel.Bounds.Tracked.End.Y = 9
	ssYExpectPin(t, s, tsel.End(), 0, 9)
	ssYExpectPin(t, s, tsel.Start(), 2, 1)
}

// Zig: test "term51: startPtr: points at the stored start pin"
func TestSelection_term51_startPtr_points_at_the_stored_start_pin(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()

	// Untracked: a pointer into the selection itself.
	sel := SelectionInit(ssYPin(t, s, 2, 1), ssYPin(t, s, 7, 4), false)
	expect(t, sel.StartPtr() == &sel.Bounds.Untracked.Start, "sel.startPtr() == &sel.bounds.untracked.start")
	expect(t, sel.StartPtr() != sel.EndPtr(), "sel.startPtr() != sel.endPtr()")
	sel.StartPtr().X = 4
	ssYExpectPin(t, s, sel.Start(), 4, 1)
	ssYExpectPin(t, s, sel.End(), 7, 4)

	// Tracked: the tracked pin itself.
	tsel, err := sel.Track(s)
	zigTry(t, err)
	defer tsel.Deinit(s)
	expect(t, tsel.StartPtr() == tsel.Bounds.Tracked.Start, "tsel.startPtr() == tsel.bounds.tracked.start")
	expect(t, tsel.StartPtr() != sel.StartPtr(), "tsel.startPtr() != sel.startPtr()")
	tsel.StartPtr().Y = 6
	ssYExpectPin(t, s, tsel.Start(), 4, 6)
	ssYExpectPin(t, s, tsel.End(), 7, 4)
	ssYExpectPin(t, s, sel.Start(), 4, 1)
}

// Zig: test "term51: endPtr: points at the stored end pin"
func TestSelection_term51_endPtr_points_at_the_stored_end_pin(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()

	sel := SelectionInit(ssYPin(t, s, 2, 1), ssYPin(t, s, 7, 4), false)
	expect(t, sel.EndPtr() == &sel.Bounds.Untracked.End, "sel.endPtr() == &sel.bounds.untracked.end")
	sel.EndPtr().Y = 8
	ssYExpectPin(t, s, sel.End(), 7, 8)
	ssYExpectPin(t, s, sel.Start(), 2, 1)

	tsel, err := sel.Track(s)
	zigTry(t, err)
	defer tsel.Deinit(s)
	expect(t, tsel.EndPtr() == tsel.Bounds.Tracked.End, "tsel.endPtr() == tsel.bounds.tracked.end")
	expect(t, tsel.EndPtr() != tsel.StartPtr(), "tsel.endPtr() != tsel.startPtr()")
	tsel.EndPtr().X = 1
	ssYExpectPin(t, s, tsel.End(), 1, 8)
	ssYExpectPin(t, s, tsel.Start(), 2, 1)
	ssYExpectPin(t, s, sel.End(), 7, 8)
}

// Zig: test "term51: eql: compares start, end and rectangle"
func TestSelection_term51_eql_compares_start_end_and_rectangle(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	a := ssYPin(t, s, 2, 1)
	b := ssYPin(t, s, 7, 4)

	sel := SelectionInit(a, b, false)
	expect(t, sel.Eql(sel), "sel.eql(sel)")
	expect(t, sel.Eql(SelectionInit(a, b, false)), "sel.eql(Selection.init(a, b, false))")

	// Rectangle flag matters.
	expect(t, !sel.Eql(SelectionInit(a, b, true)), "!sel.eql(Selection.init(a, b, true))")
	// Each coordinate of each pin matters.
	expect(t, !sel.Eql(SelectionInit(ssYPin(t, s, 3, 1), b, false)), "!sel.eql(Selection.init(term51YPin(&s, 3, 1), b, false))")
	expect(t, !sel.Eql(SelectionInit(ssYPin(t, s, 2, 2), b, false)), "!sel.eql(Selection.init(term51YPin(&s, 2, 2), b, false))")
	expect(t, !sel.Eql(SelectionInit(a, ssYPin(t, s, 8, 4), false)), "!sel.eql(Selection.init(a, term51YPin(&s, 8, 4), false))")
	expect(t, !sel.Eql(SelectionInit(a, ssYPin(t, s, 7, 5), false)), "!sel.eql(Selection.init(a, term51YPin(&s, 7, 5), false))")
	// Direction matters: the same cells swapped are not equal.
	expect(t, !sel.Eql(SelectionInit(b, a, false)), "!sel.eql(Selection.init(b, a, false))")

	// One-cell selections.
	expect(t, SelectionInit(a, a, false).Eql(SelectionInit(a, a, false)), "Selection.init(a, a, false).eql(Selection.init(a, a, false))")
	expect(t, !SelectionInit(a, a, false).Eql(SelectionInit(a, b, false)), "!Selection.init(a, a, false).eql(Selection.init(a, b, false))")

	// Tracked and untracked selections with the same pin values are equal.
	tsel, err := sel.Track(s)
	zigTry(t, err)
	defer tsel.Deinit(s)
	expect(t, sel.Eql(tsel), "sel.eql(tsel)")
	expect(t, tsel.Eql(sel), "tsel.eql(sel)")
	expect(t, tsel.Eql(tsel), "tsel.eql(tsel)")
	tsel.Bounds.Tracked.End.X = 8
	expect(t, !sel.Eql(tsel), "!sel.eql(tsel)")

	// Pins on different pages with the same x/y are different.
	s2 := ssYPages(t, 2)
	defer s2.Deinit()
	p1 := ssYPin(t, s2, 1, 1)
	p2 := p1
	expect(t, s2.Pages.pages.last != nil, "s2.pages.pages.last.?")
	p2.Node = s2.Pages.pages.last
	expect(t, !SelectionInit(p1, p1, false).Eql(SelectionInit(p1, p2, false)), "!Selection.init(p1, p1, false).eql(Selection.init(p1, p2, false))")
	expect(t, !SelectionInit(p1, p1, false).Eql(SelectionInit(p2, p1, false)), "!Selection.init(p1, p1, false).eql(Selection.init(p2, p1, false))")
}

// Zig: test "term51: topLeft: linear selections return start or end unchanged"
func TestSelection_term51_topLeft_linear_selections_return_start_or_end_unchanged(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()

	// Forward: the start pin, x not mirrored even though end.x < start.x.
	fwd := SelectionInit(ssYPin(t, s, 5, 1), ssYPin(t, s, 3, 4), false)
	ssYExpectPin(t, s, fwd.TopLeft(s), 5, 1)
	// Reverse: the end pin.
	rev := SelectionInit(ssYPin(t, s, 5, 4), ssYPin(t, s, 3, 1), false)
	ssYExpectPin(t, s, rev.TopLeft(s), 3, 1)
	// Same row, backwards.
	row := SelectionInit(ssYPin(t, s, 7, 2), ssYPin(t, s, 3, 2), false)
	ssYExpectPin(t, s, row.TopLeft(s), 3, 2)
	// One cell.
	one := SelectionInit(ssYPin(t, s, 9, 9), ssYPin(t, s, 9, 9), false)
	ssYExpectPin(t, s, one.TopLeft(s), 9, 9)
	// First cell of the screen.
	first := SelectionInit(ssYPin(t, s, 0, 0), ssYPin(t, s, 9, 9), false)
	ssYExpectPin(t, s, first.TopLeft(s), 0, 0)
}

// Zig: test "term51: topLeft: rectangle selections for every order"
func TestSelection_term51_topLeft_rectangle_selections_for_every_order(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()

	// forward
	fwd := SelectionInit(ssYPin(t, s, 2, 1), ssYPin(t, s, 7, 4), true)
	expectEqual(t, SelectionOrderForward, fwd.Order(s))
	ssYExpectPin(t, s, fwd.TopLeft(s), 2, 1)
	// reverse
	rev := SelectionInit(ssYPin(t, s, 7, 4), ssYPin(t, s, 2, 1), true)
	expectEqual(t, SelectionOrderReverse, rev.Order(s))
	ssYExpectPin(t, s, rev.TopLeft(s), 2, 1)
	// mirrored_forward: top-right start, bottom-left end
	mf := SelectionInit(ssYPin(t, s, 7, 1), ssYPin(t, s, 2, 4), true)
	expectEqual(t, SelectionOrderMirroredForward, mf.Order(s))
	ssYExpectPin(t, s, mf.TopLeft(s), 2, 1)
	// mirrored_reverse: bottom-left start, top-right end
	mr := SelectionInit(ssYPin(t, s, 2, 4), ssYPin(t, s, 7, 1), true)
	expectEqual(t, SelectionOrderMirroredReverse, mr.Order(s))
	ssYExpectPin(t, s, mr.TopLeft(s), 2, 1)
	// Single row, backwards: x are ordered.
	row := SelectionInit(ssYPin(t, s, 7, 2), ssYPin(t, s, 3, 2), true)
	ssYExpectPin(t, s, row.TopLeft(s), 3, 2)
	// Single column, bottom to top.
	col := SelectionInit(ssYPin(t, s, 4, 6), ssYPin(t, s, 4, 2), true)
	ssYExpectPin(t, s, col.TopLeft(s), 4, 2)
	// One cell.
	one := SelectionInit(ssYPin(t, s, 4, 4), ssYPin(t, s, 4, 4), true)
	ssYExpectPin(t, s, one.TopLeft(s), 4, 4)
}

// Zig: test "term51: topLeft: tracked selection across pages"
func TestSelection_term51_topLeft_tracked_selection_across_pages(t *testing.T) {
	testStdCapacity(t)
	s := ssYPages(t, 2)
	defer s.Deinit()
	n := ssYFirstRows(s)

	// Mirrored rectangle: start is on the second page, end on the first.
	sel := SelectionInit(ssYPin(t, s, 3, n+1), ssYPin(t, s, 90, n-2), true)
	tsel, err := sel.Track(s)
	zigTry(t, err)
	defer tsel.Deinit(s)
	expectEqual(t, SelectionOrderMirroredReverse, tsel.Order(s))
	tl := tsel.TopLeft(s)
	ssYExpectPin(t, s, tl, 3, n-2)
	expect(t, s.Pages.pages.first != nil, "s.pages.pages.first.?")
	expect(t, tl.Node == s.Pages.pages.first, "tl.node == s.pages.pages.first.?")
}

// Zig: test "term51: bottomRight: linear selections return end or start unchanged"
func TestSelection_term51_bottomRight_linear_selections_return_end_or_start_unchanged(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()

	fwd := SelectionInit(ssYPin(t, s, 5, 1), ssYPin(t, s, 3, 4), false)
	ssYExpectPin(t, s, fwd.BottomRight(s), 3, 4)
	rev := SelectionInit(ssYPin(t, s, 5, 4), ssYPin(t, s, 3, 1), false)
	ssYExpectPin(t, s, rev.BottomRight(s), 5, 4)
	row := SelectionInit(ssYPin(t, s, 7, 2), ssYPin(t, s, 3, 2), false)
	ssYExpectPin(t, s, row.BottomRight(s), 7, 2)
	one := SelectionInit(ssYPin(t, s, 0, 0), ssYPin(t, s, 0, 0), false)
	ssYExpectPin(t, s, one.BottomRight(s), 0, 0)
	last := SelectionInit(ssYPin(t, s, 0, 0), ssYPin(t, s, 9, 9), false)
	ssYExpectPin(t, s, last.BottomRight(s), 9, 9)
}

// Zig: test "term51: bottomRight: rectangle selections for every order"
func TestSelection_term51_bottomRight_rectangle_selections_for_every_order(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()

	fwd := SelectionInit(ssYPin(t, s, 2, 1), ssYPin(t, s, 7, 4), true)
	ssYExpectPin(t, s, fwd.BottomRight(s), 7, 4)
	rev := SelectionInit(ssYPin(t, s, 7, 4), ssYPin(t, s, 2, 1), true)
	ssYExpectPin(t, s, rev.BottomRight(s), 7, 4)
	mf := SelectionInit(ssYPin(t, s, 7, 1), ssYPin(t, s, 2, 4), true)
	ssYExpectPin(t, s, mf.BottomRight(s), 7, 4)
	mr := SelectionInit(ssYPin(t, s, 2, 4), ssYPin(t, s, 7, 1), true)
	ssYExpectPin(t, s, mr.BottomRight(s), 7, 4)
	row := SelectionInit(ssYPin(t, s, 7, 2), ssYPin(t, s, 3, 2), true)
	ssYExpectPin(t, s, row.BottomRight(s), 7, 2)
	col := SelectionInit(ssYPin(t, s, 4, 6), ssYPin(t, s, 4, 2), true)
	ssYExpectPin(t, s, col.BottomRight(s), 4, 6)
	one := SelectionInit(ssYPin(t, s, 9, 9), ssYPin(t, s, 9, 9), true)
	ssYExpectPin(t, s, one.BottomRight(s), 9, 9)
}

// Zig: test "term51: bottomRight: tracked selection across pages"
func TestSelection_term51_bottomRight_tracked_selection_across_pages(t *testing.T) {
	testStdCapacity(t)
	s := ssYPages(t, 2)
	defer s.Deinit()
	n := ssYFirstRows(s)

	// Forward rectangle: the end pin.
	fwd := SelectionInit(ssYPin(t, s, 3, n-2), ssYPin(t, s, 90, n+1), true)
	tfwd, err := fwd.Track(s)
	zigTry(t, err)
	defer tfwd.Deinit(s)
	expectEqual(t, SelectionOrderForward, tfwd.Order(s))
	br := tfwd.BottomRight(s)
	ssYExpectPin(t, s, br, 90, n+1)
	expect(t, s.Pages.pages.last != nil, "s.pages.pages.last.?")
	expect(t, br.Node == s.Pages.pages.last, "br.node == s.pages.pages.last.?")

	// Mirrored forward: bottom-right takes the x of the start pin.
	mir := SelectionInit(ssYPin(t, s, 90, n-2), ssYPin(t, s, 3, n+1), true)
	tmir, err := mir.Track(s)
	zigTry(t, err)
	defer tmir.Deinit(s)
	expectEqual(t, SelectionOrderMirroredForward, tmir.Order(s))
	br2 := tmir.BottomRight(s)
	ssYExpectPin(t, s, br2, 90, n+1)
	expect(t, br2.Node == s.Pages.pages.last, "br2.node == s.pages.pages.last.?")
}

// Zig: test "term51: order: every relative position of start to end"
func TestSelection_term51_order_every_relative_position_of_start_to_end(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 20, 20, 0)
	defer s.Deinit()

	type Case struct {
		dx     int32
		dy     int32
		linear SelectionOrder
		rect   SelectionOrder
	}
	cases := []Case{
		{dx: -1, dy: -1, linear: SelectionOrderForward, rect: SelectionOrderForward},
		{dx: 0, dy: -1, linear: SelectionOrderForward, rect: SelectionOrderForward},
		{dx: 1, dy: -1, linear: SelectionOrderForward, rect: SelectionOrderMirroredForward},
		{dx: -1, dy: 0, linear: SelectionOrderForward, rect: SelectionOrderForward},
		{dx: 0, dy: 0, linear: SelectionOrderForward, rect: SelectionOrderForward},
		{dx: 1, dy: 0, linear: SelectionOrderReverse, rect: SelectionOrderReverse},
		{dx: -1, dy: 1, linear: SelectionOrderReverse, rect: SelectionOrderMirroredReverse},
		{dx: 0, dy: 1, linear: SelectionOrderReverse, rect: SelectionOrderReverse},
		{dx: 1, dy: 1, linear: SelectionOrderReverse, rect: SelectionOrderReverse},
	}
	ep := ssYPin(t, s, 10, 10)
	for _, c := range cases {
		func() {
			// errdefer std.debug.print("order case dx={d} dy={d}\n", ...)
			defer func() {
				if t.Failed() {
					t.Logf("order case dx=%d dy=%d", c.dx, c.dy)
				}
			}()
			sp := ssYPin(t, s, SizeCellCountInt(10+c.dx), uint32(10+c.dy))
			expectEqual(t, c.linear, SelectionInit(sp, ep, false).Order(s))
			expectEqual(t, c.rect, SelectionInit(sp, ep, true).Order(s))
		}()
	}
}

// Zig: test "term51: order: boundary cells of the screen"
func TestSelection_term51_order_boundary_cells_of_the_screen(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	tl := ssYPin(t, s, 0, 0)
	tr := ssYPin(t, s, 9, 0)
	bl := ssYPin(t, s, 0, 9)
	br := ssYPin(t, s, 9, 9)

	expectEqual(t, SelectionOrderForward, SelectionInit(tl, br, false).Order(s))
	expectEqual(t, SelectionOrderReverse, SelectionInit(br, tl, false).Order(s))
	expectEqual(t, SelectionOrderForward, SelectionInit(tr, bl, false).Order(s))
	expectEqual(t, SelectionOrderReverse, SelectionInit(bl, tr, false).Order(s))
	expectEqual(t, SelectionOrderForward, SelectionInit(tl, tr, false).Order(s))
	expectEqual(t, SelectionOrderReverse, SelectionInit(tr, tl, false).Order(s))
	expectEqual(t, SelectionOrderForward, SelectionInit(tl, tl, false).Order(s))
	expectEqual(t, SelectionOrderForward, SelectionInit(br, br, false).Order(s))

	expectEqual(t, SelectionOrderForward, SelectionInit(tl, br, true).Order(s))
	expectEqual(t, SelectionOrderReverse, SelectionInit(br, tl, true).Order(s))
	expectEqual(t, SelectionOrderMirroredForward, SelectionInit(tr, bl, true).Order(s))
	expectEqual(t, SelectionOrderMirroredReverse, SelectionInit(bl, tr, true).Order(s))
	expectEqual(t, SelectionOrderForward, SelectionInit(tl, tr, true).Order(s))
	expectEqual(t, SelectionOrderReverse, SelectionInit(tr, tl, true).Order(s))
	expectEqual(t, SelectionOrderForward, SelectionInit(tl, bl, true).Order(s))
	expectEqual(t, SelectionOrderReverse, SelectionInit(bl, tl, true).Order(s))
	expectEqual(t, SelectionOrderForward, SelectionInit(br, br, true).Order(s))
}

// Zig: test "term51: order: tracked selections, history and page boundaries"
func TestSelection_term51_order_tracked_selections_history_and_page_boundaries(t *testing.T) {
	testStdCapacity(t)
	s := ssYHistory(t, 10, 3, 12)
	defer s.Deinit()

	// History rows participate in ordering (screen coordinates).
	hist := SelectionInit(ssYPin(t, s, 5, 1), ssYPin(t, s, 2, 10), false)
	expectEqual(t, SelectionOrderForward, hist.Order(s))
	tr, err := hist.Track(s)
	zigTry(t, err)
	defer tr.Deinit(s)
	expectEqual(t, SelectionOrderForward, tr.Order(s))
	tr.Bounds.Tracked.Start.Y = 11
	expectEqual(t, SelectionOrderReverse, tr.Order(s))

	s2 := ssYPages(t, 2)
	defer s2.Deinit()
	n := ssYFirstRows(s2)
	// Last row of page one vs first row of page two.
	a := ssYPin(t, s2, 199, n-1)
	b := ssYPin(t, s2, 0, n)
	expectEqual(t, SelectionOrderForward, SelectionInit(a, b, false).Order(s2))
	expectEqual(t, SelectionOrderReverse, SelectionInit(b, a, false).Order(s2))
	expectEqual(t, SelectionOrderMirroredForward, SelectionInit(a, b, true).Order(s2))
	expectEqual(t, SelectionOrderMirroredReverse, SelectionInit(b, a, true).Order(s2))
}

// Zig: test "term51: ordered: returns an untracked copy in the requested direction"
func TestSelection_term51_ordered_returns_an_untracked_copy_in_the_requested_direction(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	base := s.Pages.CountTrackedPins()

	// Same order requested: the pins are copied as they are.
	fwd := SelectionInit(ssYPin(t, s, 5, 1), ssYPin(t, s, 3, 4), false)
	ssYExpectSel(t, s, fwd.Ordered(s, SelectionOrderForward), 5, 1, 3, 4, false)
	ssYExpectSel(t, s, fwd.Ordered(s, SelectionOrderReverse), 3, 4, 5, 1, false)
	// Mirrored orders act as forward.
	ssYExpectSel(t, s, fwd.Ordered(s, SelectionOrderMirroredForward), 5, 1, 3, 4, false)
	ssYExpectSel(t, s, fwd.Ordered(s, SelectionOrderMirroredReverse), 5, 1, 3, 4, false)

	rev := SelectionInit(ssYPin(t, s, 3, 4), ssYPin(t, s, 5, 1), false)
	ssYExpectSel(t, s, rev.Ordered(s, SelectionOrderReverse), 3, 4, 5, 1, false)
	ssYExpectSel(t, s, rev.Ordered(s, SelectionOrderForward), 5, 1, 3, 4, false)
	ssYExpectSel(t, s, rev.Ordered(s, SelectionOrderMirroredForward), 5, 1, 3, 4, false)
	ssYExpectSel(t, s, rev.Ordered(s, SelectionOrderMirroredReverse), 5, 1, 3, 4, false)

	// One cell is forward, and reversing it changes nothing observable.
	one := SelectionInit(ssYPin(t, s, 4, 4), ssYPin(t, s, 4, 4), false)
	ssYExpectSel(t, s, one.Ordered(s, SelectionOrderForward), 4, 4, 4, 4, false)
	ssYExpectSel(t, s, one.Ordered(s, SelectionOrderReverse), 4, 4, 4, 4, false)

	// Nothing was tracked by any of this.
	expectEqual(t, base, s.Pages.CountTrackedPins())
}

// Zig: test "term51: ordered: rectangle selections for every order"
func TestSelection_term51_ordered_rectangle_selections_for_every_order(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()

	// Rectangle (2,1)-(7,4) written in each of the four orders.
	fwd := SelectionInit(ssYPin(t, s, 2, 1), ssYPin(t, s, 7, 4), true)
	rev := SelectionInit(ssYPin(t, s, 7, 4), ssYPin(t, s, 2, 1), true)
	mf := SelectionInit(ssYPin(t, s, 7, 1), ssYPin(t, s, 2, 4), true)
	mr := SelectionInit(ssYPin(t, s, 2, 4), ssYPin(t, s, 7, 1), true)

	all := []Selection{fwd, rev, mf, mr}
	for _, sel := range all {
		ssYExpectSel(t, s, sel.Ordered(s, SelectionOrderForward), 2, 1, 7, 4, true)
		ssYExpectSel(t, s, sel.Ordered(s, SelectionOrderReverse), 7, 4, 2, 1, true)
		// Mirrored desired orders act as forward unless already in that order.
	}
	ssYExpectSel(t, s, fwd.Ordered(s, SelectionOrderMirroredForward), 2, 1, 7, 4, true)
	ssYExpectSel(t, s, fwd.Ordered(s, SelectionOrderMirroredReverse), 2, 1, 7, 4, true)
	ssYExpectSel(t, s, rev.Ordered(s, SelectionOrderMirroredForward), 2, 1, 7, 4, true)
	ssYExpectSel(t, s, rev.Ordered(s, SelectionOrderMirroredReverse), 2, 1, 7, 4, true)
	// Already mirrored_forward: unchanged pins.
	ssYExpectSel(t, s, mf.Ordered(s, SelectionOrderMirroredForward), 7, 1, 2, 4, true)
	ssYExpectSel(t, s, mf.Ordered(s, SelectionOrderMirroredReverse), 2, 1, 7, 4, true)
	// Already mirrored_reverse: unchanged pins.
	ssYExpectSel(t, s, mr.Ordered(s, SelectionOrderMirroredReverse), 2, 4, 7, 1, true)
	ssYExpectSel(t, s, mr.Ordered(s, SelectionOrderMirroredForward), 2, 1, 7, 4, true)

	// Single column and single row, reversed.
	col := SelectionInit(ssYPin(t, s, 4, 6), ssYPin(t, s, 4, 2), true)
	ssYExpectSel(t, s, col.Ordered(s, SelectionOrderForward), 4, 2, 4, 6, true)
	row := SelectionInit(ssYPin(t, s, 8, 3), ssYPin(t, s, 1, 3), true)
	ssYExpectSel(t, s, row.Ordered(s, SelectionOrderForward), 1, 3, 8, 3, true)
	ssYExpectSel(t, s, row.Ordered(s, SelectionOrderReverse), 8, 3, 1, 3, true)
}

// Zig: test "term51: ordered: tracked selection yields an independent untracked one"
func TestSelection_term51_ordered_tracked_selection_yields_an_independent_untracked_one(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	sel := SelectionInit(ssYPin(t, s, 5, 4), ssYPin(t, s, 3, 1), false)
	tsel, err := sel.Track(s)
	zigTry(t, err)
	defer tsel.Deinit(s)
	base := s.Pages.CountTrackedPins()

	same := tsel.Ordered(s, SelectionOrderReverse)
	ssYExpectSel(t, s, same, 5, 4, 3, 1, false)
	flipped := tsel.Ordered(s, SelectionOrderForward)
	ssYExpectSel(t, s, flipped, 3, 1, 5, 4, false)
	expectEqual(t, base, s.Pages.CountTrackedPins())

	// Moving the tracked pins afterwards does not affect the copies.
	tsel.Bounds.Tracked.End.X = 9
	ssYExpectSel(t, s, flipped, 3, 1, 5, 4, false)
	ssYExpectSel(t, s, same, 5, 4, 3, 1, false)
}

// Zig: test "term51: contains: linear selection boundaries"
func TestSelection_term51_contains_linear_selection_boundaries(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()

	pts := []ssYPt{
		// Row above the top line.
		{x: 9, y: 0, want: false},
		// Top line: x >= 5.
		{x: 4, y: 1, want: false},
		{x: 0, y: 1, want: false},
		{x: 5, y: 1, want: true},
		{x: 9, y: 1, want: true},
		// Middle lines: everything, including both screen edges.
		{x: 0, y: 2, want: true},
		{x: 9, y: 2, want: true},
		{x: 0, y: 3, want: true},
		{x: 9, y: 3, want: true},
		// Bottom line: x <= 3.
		{x: 0, y: 4, want: true},
		{x: 3, y: 4, want: true},
		{x: 4, y: 4, want: false},
		{x: 9, y: 4, want: false},
		// Row below the bottom line.
		{x: 0, y: 5, want: false},
	}
	// Forward and reverse give identical answers.
	fwd := SelectionInit(ssYPin(t, s, 5, 1), ssYPin(t, s, 3, 4), false)
	rev := SelectionInit(ssYPin(t, s, 3, 4), ssYPin(t, s, 5, 1), false)
	ssYCheckContains(t, s, fwd, pts)
	ssYCheckContains(t, s, rev, pts)
}

// Zig: test "term51: contains: one cell, one row and adjacent rows"
func TestSelection_term51_contains_one_cell_one_row_and_adjacent_rows(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()

	// One cell.
	one := SelectionInit(ssYPin(t, s, 4, 4), ssYPin(t, s, 4, 4), false)
	ssYCheckContains(t, s, one, []ssYPt{
		{x: 4, y: 4, want: true},
		{x: 3, y: 4, want: false},
		{x: 5, y: 4, want: false},
		{x: 4, y: 3, want: false},
		{x: 4, y: 5, want: false},
	})

	// One row, in both directions.
	row_pts := []ssYPt{
		{x: 2, y: 2, want: false},
		{x: 3, y: 2, want: true},
		{x: 6, y: 2, want: true},
		{x: 7, y: 2, want: false},
		{x: 4, y: 1, want: false},
		{x: 4, y: 3, want: false},
	}
	ssYCheckContains(t, s, SelectionInit(ssYPin(t, s, 3, 2), ssYPin(t, s, 6, 2), false), row_pts)
	ssYCheckContains(t, s, SelectionInit(ssYPin(t, s, 6, 2), ssYPin(t, s, 3, 2), false), row_pts)

	// Two adjacent rows: the tail of the first and the head of the second.
	two := SelectionInit(ssYPin(t, s, 8, 3), ssYPin(t, s, 1, 4), false)
	ssYCheckContains(t, s, two, []ssYPt{
		{x: 7, y: 3, want: false},
		{x: 8, y: 3, want: true},
		{x: 9, y: 3, want: true},
		{x: 0, y: 4, want: true},
		{x: 1, y: 4, want: true},
		{x: 2, y: 4, want: false},
	})
}

// Zig: test "term51: contains: whole screen and screen corners"
func TestSelection_term51_contains_whole_screen_and_screen_corners(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()

	all := SelectionInit(ssYPin(t, s, 0, 0), ssYPin(t, s, 9, 9), false)
	ssYCheckContains(t, s, all, []ssYPt{
		{x: 0, y: 0, want: true},
		{x: 9, y: 0, want: true},
		{x: 0, y: 9, want: true},
		{x: 9, y: 9, want: true},
		{x: 5, y: 5, want: true},
	})

	// Last cell of the first row to first cell of the last row.
	edge := SelectionInit(ssYPin(t, s, 9, 0), ssYPin(t, s, 0, 9), false)
	ssYCheckContains(t, s, edge, []ssYPt{
		{x: 8, y: 0, want: false},
		{x: 9, y: 0, want: true},
		{x: 0, y: 1, want: true},
		{x: 9, y: 8, want: true},
		{x: 0, y: 9, want: true},
		{x: 1, y: 9, want: false},
	})
}

// Zig: test "term51: contains: rectangle in every order"
func TestSelection_term51_contains_rectangle_in_every_order(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 15, 15, 0)
	defer s.Deinit()

	pts := []ssYPt{
		{x: 3, y: 3, want: true}, // top-left corner
		{x: 7, y: 3, want: true}, // top-right corner
		{x: 3, y: 9, want: true}, // bottom-left corner
		{x: 7, y: 9, want: true}, // bottom-right corner
		{x: 5, y: 6, want: true},
		{x: 2, y: 3, want: false},
		{x: 8, y: 3, want: false},
		{x: 2, y: 9, want: false},
		{x: 8, y: 9, want: false},
		{x: 3, y: 2, want: false},
		{x: 7, y: 2, want: false},
		{x: 3, y: 10, want: false},
		{x: 7, y: 10, want: false},
		{x: 0, y: 6, want: false},
		{x: 14, y: 6, want: false},
	}
	fwd := SelectionInit(ssYPin(t, s, 3, 3), ssYPin(t, s, 7, 9), true)
	rev := SelectionInit(ssYPin(t, s, 7, 9), ssYPin(t, s, 3, 3), true)
	mf := SelectionInit(ssYPin(t, s, 7, 3), ssYPin(t, s, 3, 9), true)
	mr := SelectionInit(ssYPin(t, s, 3, 9), ssYPin(t, s, 7, 3), true)
	expectEqual(t, SelectionOrderMirroredForward, mf.Order(s))
	expectEqual(t, SelectionOrderMirroredReverse, mr.Order(s))
	ssYCheckContains(t, s, fwd, pts)
	ssYCheckContains(t, s, rev, pts)
	ssYCheckContains(t, s, mf, pts)
	ssYCheckContains(t, s, mr, pts)
}

// Zig: test "term51: contains: rectangle one cell, one row, one column"
func TestSelection_term51_contains_rectangle_one_cell_one_row_one_column(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()

	one := SelectionInit(ssYPin(t, s, 4, 4), ssYPin(t, s, 4, 4), true)
	ssYCheckContains(t, s, one, []ssYPt{
		{x: 4, y: 4, want: true},
		{x: 3, y: 4, want: false},
		{x: 5, y: 4, want: false},
		{x: 4, y: 3, want: false},
		{x: 4, y: 5, want: false},
	})

	row := SelectionInit(ssYPin(t, s, 6, 2), ssYPin(t, s, 3, 2), true)
	ssYCheckContains(t, s, row, []ssYPt{
		{x: 2, y: 2, want: false},
		{x: 3, y: 2, want: true},
		{x: 6, y: 2, want: true},
		{x: 7, y: 2, want: false},
		{x: 4, y: 1, want: false},
		{x: 4, y: 3, want: false},
	})

	col := SelectionInit(ssYPin(t, s, 5, 7), ssYPin(t, s, 5, 3), true)
	ssYCheckContains(t, s, col, []ssYPt{
		{x: 5, y: 2, want: false},
		{x: 5, y: 3, want: true},
		{x: 5, y: 7, want: true},
		{x: 5, y: 8, want: false},
		{x: 4, y: 5, want: false},
		{x: 6, y: 5, want: false},
	})
}

// Zig: test "term51: contains: tracked selection and wide characters"
func TestSelection_term51_contains_tracked_selection_and_wide_characters(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("あい")))

	// Starting on a spacer tail: no snapping to the wide cell, containment is
	// purely by coordinate.
	sel := SelectionInit(ssYPin(t, s, 1, 0), ssYPin(t, s, 2, 0), false)
	tsel, err := sel.Track(s)
	zigTry(t, err)
	defer tsel.Deinit(s)
	pts := []ssYPt{
		{x: 0, y: 0, want: false},
		{x: 1, y: 0, want: true},
		{x: 2, y: 0, want: true},
		{x: 3, y: 0, want: false},
	}
	ssYCheckContains(t, s, sel, pts)
	ssYCheckContains(t, s, tsel, pts)
}

// Zig: test "term51: contains: history rows and page boundaries"
func TestSelection_term51_contains_history_rows_and_page_boundaries(t *testing.T) {
	testStdCapacity(t)
	s := ssYHistory(t, 10, 3, 12)
	defer s.Deinit()
	// Selection entirely in history.
	sel := SelectionInit(ssYPin(t, s, 4, 1), ssYPin(t, s, 2, 3), false)
	ssYCheckContains(t, s, sel, []ssYPt{
		{x: 3, y: 1, want: false},
		{x: 4, y: 1, want: true},
		{x: 0, y: 2, want: true},
		{x: 9, y: 2, want: true},
		{x: 2, y: 3, want: true},
		{x: 3, y: 3, want: false},
		{x: 0, y: 0, want: false},
		{x: 0, y: 4, want: false},
		{x: 0, y: 11, want: false},
	})

	s2 := ssYPages(t, 2)
	defer s2.Deinit()
	n := ssYFirstRows(s2)
	lsel := SelectionInit(ssYPin(t, s2, 2, n-1), ssYPin(t, s2, 5, n+1), false)
	ssYCheckContains(t, s2, lsel, []ssYPt{
		{x: 1, y: n - 1, want: false},
		{x: 2, y: n - 1, want: true},
		{x: 199, y: n - 1, want: true},
		{x: 0, y: n, want: true},
		{x: 199, y: n, want: true},
		{x: 0, y: n + 1, want: true},
		{x: 5, y: n + 1, want: true},
		{x: 6, y: n + 1, want: false},
		{x: 0, y: n + 2, want: false},
		{x: 199, y: n - 2, want: false},
	})
	rsel := SelectionInit(ssYPin(t, s2, 150, n-1), ssYPin(t, s2, 3, n+1), true)
	expectEqual(t, SelectionOrderMirroredForward, rsel.Order(s2))
	ssYCheckContains(t, s2, rsel, []ssYPt{
		{x: 3, y: n - 1, want: true},
		{x: 150, y: n, want: true},
		{x: 151, y: n, want: false},
		{x: 2, y: n + 1, want: false},
		{x: 100, y: n + 1, want: true},
		{x: 100, y: n + 2, want: false},
		{x: 100, y: n - 2, want: false},
	})
}

// Zig: test "term51: containedRow: linear selection rows, forward and reverse"
func TestSelection_term51_containedRow_linear_selection_rows_forward_and_reverse(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()

	fwd := SelectionInit(ssYPin(t, s, 5, 1), ssYPin(t, s, 3, 4), false)
	rev := SelectionInit(ssYPin(t, s, 3, 4), ssYPin(t, s, 5, 1), false)
	for _, sel := range []Selection{fwd, rev} {
		// Rows outside the selection.
		expect(t, !sel.ContainedRow(s, ssYPin(t, s, 0, 0)).Valid, "sel.containedRow(&s, (0, 0)) == null")
		expect(t, !sel.ContainedRow(s, ssYPin(t, s, 9, 0)).Valid, "sel.containedRow(&s, (9, 0)) == null")
		expect(t, !sel.ContainedRow(s, ssYPin(t, s, 4, 5)).Valid, "sel.containedRow(&s, (4, 5)) == null")
		expect(t, !sel.ContainedRow(s, ssYPin(t, s, 4, 9)).Valid, "sel.containedRow(&s, (4, 9)) == null")
		// Top row: from the top-left pin to the end of the row. The x of the
		// probe pin is irrelevant.
		ssYExpectSel(t, s, zigUnwrap(t, sel.ContainedRow(s, ssYPin(t, s, 0, 1))), 5, 1, 9, 1, false)
		ssYExpectSel(t, s, zigUnwrap(t, sel.ContainedRow(s, ssYPin(t, s, 9, 1))), 5, 1, 9, 1, false)
		// Middle rows: the whole row.
		ssYExpectSel(t, s, zigUnwrap(t, sel.ContainedRow(s, ssYPin(t, s, 4, 2))), 0, 2, 9, 2, false)
		ssYExpectSel(t, s, zigUnwrap(t, sel.ContainedRow(s, ssYPin(t, s, 0, 3))), 0, 3, 9, 3, false)
		// Bottom row: from the start of the row to the bottom-right pin.
		ssYExpectSel(t, s, zigUnwrap(t, sel.ContainedRow(s, ssYPin(t, s, 9, 4))), 0, 4, 3, 4, false)
		ssYExpectSel(t, s, zigUnwrap(t, sel.ContainedRow(s, ssYPin(t, s, 0, 4))), 0, 4, 3, 4, false)
	}
}

// Zig: test "term51: containedRow: one row, one cell, first and last screen rows"
func TestSelection_term51_containedRow_one_row_one_cell_first_and_last_screen_rows(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()

	// One row: returned in forward order whichever direction it was made.
	fwd := SelectionInit(ssYPin(t, s, 2, 3), ssYPin(t, s, 6, 3), false)
	rev := SelectionInit(ssYPin(t, s, 6, 3), ssYPin(t, s, 2, 3), false)
	for _, sel := range []Selection{fwd, rev} {
		expect(t, !sel.ContainedRow(s, ssYPin(t, s, 4, 2)).Valid, "sel.containedRow(&s, (4, 2)) == null")
		expect(t, !sel.ContainedRow(s, ssYPin(t, s, 4, 4)).Valid, "sel.containedRow(&s, (4, 4)) == null")
		ssYExpectSel(t, s, zigUnwrap(t, sel.ContainedRow(s, ssYPin(t, s, 9, 3))), 2, 3, 6, 3, false)
	}

	// One cell at the last row/column and at the first row/column.
	last := SelectionInit(ssYPin(t, s, 9, 9), ssYPin(t, s, 9, 9), false)
	ssYExpectSel(t, s, zigUnwrap(t, last.ContainedRow(s, ssYPin(t, s, 0, 9))), 9, 9, 9, 9, false)
	expect(t, !last.ContainedRow(s, ssYPin(t, s, 9, 8)).Valid, "last.containedRow(&s, (9, 8)) == null")
	first := SelectionInit(ssYPin(t, s, 0, 0), ssYPin(t, s, 0, 0), false)
	ssYExpectSel(t, s, zigUnwrap(t, first.ContainedRow(s, ssYPin(t, s, 5, 0))), 0, 0, 0, 0, false)
	expect(t, !first.ContainedRow(s, ssYPin(t, s, 0, 1)).Valid, "first.containedRow(&s, (0, 1)) == null")

	// Whole screen: every row is covered; first and last are partial only
	// by the selection's own pins.
	all := SelectionInit(ssYPin(t, s, 0, 0), ssYPin(t, s, 9, 9), false)
	ssYExpectSel(t, s, zigUnwrap(t, all.ContainedRow(s, ssYPin(t, s, 3, 0))), 0, 0, 9, 0, false)
	ssYExpectSel(t, s, zigUnwrap(t, all.ContainedRow(s, ssYPin(t, s, 3, 5))), 0, 5, 9, 5, false)
	ssYExpectSel(t, s, zigUnwrap(t, all.ContainedRow(s, ssYPin(t, s, 3, 9))), 0, 9, 9, 9, false)

	// Last column of the first row to first column of the last row.
	edge := SelectionInit(ssYPin(t, s, 9, 0), ssYPin(t, s, 0, 9), false)
	ssYExpectSel(t, s, zigUnwrap(t, edge.ContainedRow(s, ssYPin(t, s, 3, 0))), 9, 0, 9, 0, false)
	ssYExpectSel(t, s, zigUnwrap(t, edge.ContainedRow(s, ssYPin(t, s, 3, 9))), 0, 9, 0, 9, false)
}

// Zig: test "term51: containedRow: rectangle selections in every order"
func TestSelection_term51_containedRow_rectangle_selections_in_every_order(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()

	fwd := SelectionInit(ssYPin(t, s, 2, 1), ssYPin(t, s, 7, 4), true)
	rev := SelectionInit(ssYPin(t, s, 7, 4), ssYPin(t, s, 2, 1), true)
	mf := SelectionInit(ssYPin(t, s, 7, 1), ssYPin(t, s, 2, 4), true)
	mr := SelectionInit(ssYPin(t, s, 2, 4), ssYPin(t, s, 7, 1), true)
	for _, sel := range []Selection{fwd, rev, mf, mr} {
		expect(t, !sel.ContainedRow(s, ssYPin(t, s, 4, 0)).Valid, "sel.containedRow(&s, (4, 0)) == null")
		expect(t, !sel.ContainedRow(s, ssYPin(t, s, 4, 5)).Valid, "sel.containedRow(&s, (4, 5)) == null")
		// The probe pin's x is irrelevant; the rectangle's x range is used.
		for y := uint32(1); y <= 4; y++ {
			ssYExpectSel(t, s, zigUnwrap(t, sel.ContainedRow(s, ssYPin(t, s, 9, y))), 2, y, 7, y, true)
			ssYExpectSel(t, s, zigUnwrap(t, sel.ContainedRow(s, ssYPin(t, s, 0, y))), 2, y, 7, y, true)
		}
	}

	// A one-row rectangle stays a rectangle (and is ordered left to right).
	row := SelectionInit(ssYPin(t, s, 6, 2), ssYPin(t, s, 2, 2), true)
	ssYExpectSel(t, s, zigUnwrap(t, row.ContainedRow(s, ssYPin(t, s, 0, 2))), 2, 2, 6, 2, true)
	expect(t, !row.ContainedRow(s, ssYPin(t, s, 0, 3)).Valid, "row.containedRow(&s, (0, 3)) == null")
	// One cell.
	one := SelectionInit(ssYPin(t, s, 4, 4), ssYPin(t, s, 4, 4), true)
	ssYExpectSel(t, s, zigUnwrap(t, one.ContainedRow(s, ssYPin(t, s, 0, 4))), 4, 4, 4, 4, true)
	// One column, bottom to top.
	col := SelectionInit(ssYPin(t, s, 4, 6), ssYPin(t, s, 4, 2), true)
	ssYExpectSel(t, s, zigUnwrap(t, col.ContainedRow(s, ssYPin(t, s, 9, 5))), 4, 5, 4, 5, true)
	expect(t, !col.ContainedRow(s, ssYPin(t, s, 4, 7)).Valid, "col.containedRow(&s, (4, 7)) == null")
	// Full width rectangle.
	wide := SelectionInit(ssYPin(t, s, 0, 2), ssYPin(t, s, 9, 3), true)
	ssYExpectSel(t, s, zigUnwrap(t, wide.ContainedRow(s, ssYPin(t, s, 5, 3))), 0, 3, 9, 3, true)
}

// Zig: test "term51: containedRow: tracked selection yields untracked rows"
func TestSelection_term51_containedRow_tracked_selection_yields_untracked_rows(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	sel := SelectionInit(ssYPin(t, s, 5, 1), ssYPin(t, s, 3, 4), false)
	tsel, err := sel.Track(s)
	zigTry(t, err)
	defer tsel.Deinit(s)
	base := s.Pages.CountTrackedPins()

	row := zigUnwrap(t, tsel.ContainedRow(s, ssYPin(t, s, 0, 1)))
	ssYExpectSel(t, s, row, 5, 1, 9, 1, false)
	expectEqual(t, base, s.Pages.CountTrackedPins())
	// Moving the tracked pin afterwards does not affect the returned row.
	tsel.Bounds.Tracked.Start.X = 8
	ssYExpectSel(t, s, row, 5, 1, 9, 1, false)
	ssYExpectSel(t, s, zigUnwrap(t, tsel.ContainedRow(s, ssYPin(t, s, 0, 1))), 8, 1, 9, 1, false)
}

// Zig: test "term51: containedRow: history rows and page boundaries"
func TestSelection_term51_containedRow_history_rows_and_page_boundaries(t *testing.T) {
	testStdCapacity(t)
	s := ssYHistory(t, 10, 3, 12)
	defer s.Deinit()
	sel := SelectionInit(ssYPin(t, s, 4, 1), ssYPin(t, s, 2, 10), false)
	expect(t, !sel.ContainedRow(s, ssYPin(t, s, 0, 0)).Valid, "sel.containedRow(&s, (0, 0)) == null")
	ssYExpectSel(t, s, zigUnwrap(t, sel.ContainedRow(s, ssYPin(t, s, 0, 1))), 4, 1, 9, 1, false)
	ssYExpectSel(t, s, zigUnwrap(t, sel.ContainedRow(s, ssYPin(t, s, 0, 9))), 0, 9, 9, 9, false)
	ssYExpectSel(t, s, zigUnwrap(t, sel.ContainedRow(s, ssYPin(t, s, 0, 10))), 0, 10, 2, 10, false)
	expect(t, !sel.ContainedRow(s, ssYPin(t, s, 0, 11)).Valid, "sel.containedRow(&s, (0, 11)) == null")

	s2 := ssYPages(t, 2)
	defer s2.Deinit()
	n := ssYFirstRows(s2)
	expect(t, s2.Pages.pages.first != nil, "s2.pages.pages.first.?")
	first := s2.Pages.pages.first
	expect(t, s2.Pages.pages.last != nil, "s2.pages.pages.last.?")
	last := s2.Pages.pages.last
	psel := SelectionInit(ssYPin(t, s2, 2, n-1), ssYPin(t, s2, 5, n+1), false)
	expect(t, !psel.ContainedRow(s2, ssYPin(t, s2, 0, n-2)).Valid, "psel.containedRow(&s2, (0, n - 2)) == null")
	r1 := zigUnwrap(t, psel.ContainedRow(s2, ssYPin(t, s2, 0, n-1)))
	ssYExpectSel(t, s2, r1, 2, n-1, 199, n-1, false)
	expect(t, r1.Start().Node == first && r1.End().Node == first, "r1.start().node == first and r1.end().node == first")
	r2 := zigUnwrap(t, psel.ContainedRow(s2, ssYPin(t, s2, 0, n)))
	ssYExpectSel(t, s2, r2, 0, n, 199, n, false)
	expect(t, r2.Start().Node == last && r2.End().Node == last, "r2.start().node == last and r2.end().node == last")
	r3 := zigUnwrap(t, psel.ContainedRow(s2, ssYPin(t, s2, 0, n+1)))
	ssYExpectSel(t, s2, r3, 0, n+1, 5, n+1, false)
	expect(t, !psel.ContainedRow(s2, ssYPin(t, s2, 0, n+2)).Valid, "psel.containedRow(&s2, (0, n + 2)) == null")

	// Rectangle across the boundary, mirrored.
	rsel := SelectionInit(ssYPin(t, s2, 150, n-1), ssYPin(t, s2, 3, n+1), true)
	q := zigUnwrap(t, rsel.ContainedRow(s2, ssYPin(t, s2, 0, n)))
	ssYExpectSel(t, s2, q, 3, n, 150, n, true)
	expect(t, q.Start().Node == last, "q.start().node == last")
	expect(t, !rsel.ContainedRow(s2, ssYPin(t, s2, 0, n+2)).Valid, "rsel.containedRow(&s2, (0, n + 2)) == null")
}

// Zig: test "term51: containedRow: wide characters are not snapped"
func TestSelection_term51_containedRow_wide_characters_are_not_snapped(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("あい")))
	sel := SelectionInit(ssYPin(t, s, 1, 0), ssYPin(t, s, 2, 0), false)
	ssYExpectSel(t, s, zigUnwrap(t, sel.ContainedRow(s, ssYPin(t, s, 0, 0))), 1, 0, 2, 0, false)
}

// Zig: test "term51: containedRowCached: row outside the coordinate range is null"
func TestSelection_term51_containedRowCached_row_outside_the_coordinate_range_is_null(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	tl_pin := ssYPin(t, s, 2, 2)
	br_pin := ssYPin(t, s, 6, 5)
	tl := PointCoordinate{X: 2, Y: 2}
	br := PointCoordinate{X: 6, Y: 5}

	for _, rect := range []bool{false, true} {
		sel := SelectionInit(tl_pin, br_pin, rect)
		// One row above the top and one row below the bottom.
		expect(t, !sel.ContainedRowCached(s, tl_pin, br_pin, ssYPin(t, s, 4, 1), tl, br, PointCoordinate{X: 4, Y: 1}).Valid, "sel.containedRowCached(&s, tl_pin, br_pin, (4, 1), tl, br, .{ .x = 4, .y = 1 }) == null")
		expect(t, !sel.ContainedRowCached(s, tl_pin, br_pin, ssYPin(t, s, 4, 6), tl, br, PointCoordinate{X: 4, Y: 6}).Valid, "sel.containedRowCached(&s, tl_pin, br_pin, (4, 6), tl, br, .{ .x = 4, .y = 6 }) == null")
		// The first and last rows are in range.
		expect(t, sel.ContainedRowCached(s, tl_pin, br_pin, ssYPin(t, s, 4, 2), tl, br, PointCoordinate{X: 4, Y: 2}).Valid, "sel.containedRowCached(&s, tl_pin, br_pin, (4, 2), tl, br, .{ .x = 4, .y = 2 }) != null")
		expect(t, sel.ContainedRowCached(s, tl_pin, br_pin, ssYPin(t, s, 4, 5), tl, br, PointCoordinate{X: 4, Y: 5}).Valid, "sel.containedRowCached(&s, tl_pin, br_pin, (4, 5), tl, br, .{ .x = 4, .y = 5 }) != null")
	}
}

// Zig: test "term51: containedRowCached: only the coordinates decide the branch, pins supply the result"
func TestSelection_term51_containedRowCached_only_the_coordinates_decide_the_branch_pins_supply_the_result(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	tl_pin := ssYPin(t, s, 2, 2)
	br_pin := ssYPin(t, s, 6, 5)
	sel := SelectionInit(tl_pin, br_pin, false)
	tl := PointCoordinate{X: 2, Y: 2}
	br := PointCoordinate{X: 6, Y: 5}

	// Top row branch: start is tl_pin, end is the probe pin with x = cols - 1.
	// The probe pin is deliberately on another row than the coordinates say.
	top := zigUnwrap(t, sel.ContainedRowCached(s, tl_pin, br_pin, ssYPin(t, s, 3, 7), tl, br, PointCoordinate{X: 0, Y: 2}))
	ssYExpectSel(t, s, top, 2, 2, 9, 7, false)

	// Bottom row branch: start is the probe pin with x = 0, end is br_pin.
	bot := zigUnwrap(t, sel.ContainedRowCached(s, tl_pin, br_pin, ssYPin(t, s, 3, 7), tl, br, PointCoordinate{X: 0, Y: 5}))
	ssYExpectSel(t, s, bot, 0, 7, 6, 5, false)

	// Middle branch: the full probe row.
	mid := zigUnwrap(t, sel.ContainedRowCached(s, tl_pin, br_pin, ssYPin(t, s, 3, 7), tl, br, PointCoordinate{X: 0, Y: 3}))
	ssYExpectSel(t, s, mid, 0, 7, 9, 7, false)
	mid2 := zigUnwrap(t, sel.ContainedRowCached(s, tl_pin, br_pin, ssYPin(t, s, 3, 7), tl, br, PointCoordinate{X: 9, Y: 4}))
	ssYExpectSel(t, s, mid2, 0, 7, 9, 7, false)

	// Single-row branch (tl.y == br.y == p.y): the pins are returned as given,
	// without reordering, and the rectangle flag is false.
	one_tl := PointCoordinate{X: 4, Y: 3}
	one_br := PointCoordinate{X: 8, Y: 3}
	single := zigUnwrap(t, sel.ContainedRowCached(s, tl_pin, br_pin, ssYPin(t, s, 3, 7), one_tl, one_br, PointCoordinate{X: 0, Y: 3}))
	ssYExpectSel(t, s, single, 2, 2, 6, 5, false)
	swapped := zigUnwrap(t, sel.ContainedRowCached(s, br_pin, tl_pin, ssYPin(t, s, 3, 7), one_tl, one_br, PointCoordinate{X: 0, Y: 3}))
	ssYExpectSel(t, s, swapped, 6, 5, 2, 2, false)

	// Rectangle: the x range comes from the coordinates, y from the probe pin,
	// and the result is a rectangle regardless of the coordinates' rows.
	rect := SelectionInit(tl_pin, br_pin, true)
	rtl := PointCoordinate{X: 1, Y: 2}
	rbr := PointCoordinate{X: 8, Y: 5}
	r1 := zigUnwrap(t, rect.ContainedRowCached(s, tl_pin, br_pin, ssYPin(t, s, 9, 7), rtl, rbr, PointCoordinate{X: 9, Y: 3}))
	ssYExpectSel(t, s, r1, 1, 7, 8, 7, true)
	// Rectangle also applies to a single-row range.
	rs := PointCoordinate{X: 1, Y: 4}
	re := PointCoordinate{X: 8, Y: 4}
	r2 := zigUnwrap(t, rect.ContainedRowCached(s, tl_pin, br_pin, ssYPin(t, s, 9, 4), rs, re, PointCoordinate{X: 0, Y: 4}))
	ssYExpectSel(t, s, r2, 1, 4, 8, 4, true)
}

// Zig: test "term51: containedRowCached: one-row range with unequal tl and br rows"
func TestSelection_term51_containedRowCached_one_row_range_with_unequal_tl_and_br_rows(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	tl_pin := ssYPin(t, s, 2, 2)
	br_pin := ssYPin(t, s, 6, 3)
	sel := SelectionInit(tl_pin, br_pin, false)
	tl := PointCoordinate{X: 2, Y: 2}
	br := PointCoordinate{X: 6, Y: 3}
	// Top: end at cols - 1.
	top := zigUnwrap(t, sel.ContainedRowCached(s, tl_pin, br_pin, ssYPin(t, s, 0, 2), tl, br, tl))
	ssYExpectSel(t, s, top, 2, 2, 9, 2, false)
	// Bottom: start at x = 0.
	bot := zigUnwrap(t, sel.ContainedRowCached(s, tl_pin, br_pin, ssYPin(t, s, 0, 3), tl, br, br))
	ssYExpectSel(t, s, bot, 0, 3, 6, 3, false)
}

// Zig: test "term51: adjust: up"
func TestSelection_term51_adjust_up(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("A1234\nB5678\nC1234\nD5678")))

	ssYCheckAdjust(t, s, SelectionAdjustmentUp, []ssYEndCase{
		{ex: 3, ey: 3, nx: 3, ny: 2},
		{ex: 3, ey: 1, nx: 3, ny: 0},
		{ex: 9, ey: 9, nx: 9, ny: 8},
		{ex: 0, ey: 1, nx: 0, ny: 0},
		// First row: moves to the beginning of the line instead.
		{ex: 6, ey: 0, nx: 0, ny: 0},
		{ex: 9, ey: 0, nx: 0, ny: 0},
		{ex: 0, ey: 0, nx: 0, ny: 0},
	})

	// Reverse selection (end above start): the end pin is still the one moved.
	rev := SelectionInit(ssYPin(t, s, 4, 5), ssYPin(t, s, 2, 2), false)
	rev.Adjust(s, SelectionAdjustmentUp)
	ssYExpectSel(t, s, rev, 4, 5, 2, 1, false)

	// Rectangle selections adjust the same way and stay rectangles.
	rect := SelectionInit(ssYPin(t, s, 4, 5), ssYPin(t, s, 2, 2), true)
	rect.Adjust(s, SelectionAdjustmentUp)
	ssYExpectSel(t, s, rect, 4, 5, 2, 1, true)
}

// Zig: test "term51: adjust: down"
func TestSelection_term51_adjust_down(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("A1234\nB5678\nC1234\nD5678")))

	ssYCheckAdjust(t, s, SelectionAdjustmentDown, []ssYEndCase{
		{ex: 3, ey: 0, nx: 3, ny: 1},
		{ex: 9, ey: 0, nx: 9, ny: 1},
		{ex: 0, ey: 1, nx: 0, ny: 2},
		{ex: 3, ey: 2, nx: 3, ny: 3},
		// No text below: move to the end of the current line.
		{ex: 3, ey: 3, nx: 9, ny: 3},
		{ex: 9, ey: 3, nx: 9, ny: 3},
		{ex: 3, ey: 4, nx: 9, ny: 4},
		{ex: 0, ey: 8, nx: 9, ny: 8},
		// Last row of the screen: down() is null.
		{ex: 3, ey: 9, nx: 9, ny: 9},
		{ex: 9, ey: 9, nx: 9, ny: 9},
	})

	rect := SelectionInit(ssYPin(t, s, 4, 5), ssYPin(t, s, 2, 0), true)
	rect.Adjust(s, SelectionAdjustmentDown)
	ssYExpectSel(t, s, rect, 4, 5, 2, 1, true)
}

// Zig: test "term51: adjust: down skips rows without text"
func TestSelection_term51_adjust_down_skips_rows_without_text(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	// Rows: 0 "A", 1 blank, 2 blank, 3 "D", 4 blank, 5 "F", 6 spaces, 7 bg only,
	// 8 "H", 9 blank.
	zigTry(t, s.TestWriteString([]uint8("A\n\n\nD\n\nF\n   \n\nH")))
	ssYSetBg(t, s, 3, 7)

	ssYCheckAdjust(t, s, SelectionAdjustmentDown, []ssYEndCase{
		{ex: 0, ey: 0, nx: 0, ny: 3},
		{ex: 4, ey: 1, nx: 4, ny: 3},
		{ex: 4, ey: 3, nx: 4, ny: 5},
		// A row of spaces has text (codepoint 32).
		{ex: 2, ey: 5, nx: 2, ny: 6},
		// A row with only a background colour cell has no text: skipped.
		{ex: 2, ey: 6, nx: 2, ny: 8},
		{ex: 2, ey: 7, nx: 2, ny: 8},
		{ex: 2, ey: 8, nx: 9, ny: 8},
		{ex: 2, ey: 9, nx: 9, ny: 9},
	})
}

// Zig: test "term51: adjust: left"
func TestSelection_term51_adjust_left(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("A1234\nB5678\nC1234\nD5678")))

	ssYCheckAdjust(t, s, SelectionAdjustmentLeft, []ssYEndCase{
		{ex: 3, ey: 3, nx: 2, ny: 3},
		{ex: 4, ey: 0, nx: 3, ny: 0},
		{ex: 1, ey: 0, nx: 0, ny: 0},
		// Start of a row: the last cell with text of the previous row.
		{ex: 0, ey: 3, nx: 4, ny: 2},
		{ex: 0, ey: 1, nx: 4, ny: 0},
		// Blank cells are skipped.
		{ex: 9, ey: 1, nx: 4, ny: 1},
		{ex: 5, ey: 0, nx: 4, ny: 0},
		// Blank rows are skipped too.
		{ex: 7, ey: 6, nx: 4, ny: 3},
		{ex: 0, ey: 6, nx: 4, ny: 3},
		// Nothing before: unchanged.
		{ex: 0, ey: 0, nx: 0, ny: 0},
	})

	// Nothing with text before the end pin at all.
	blank := ssScreenInit(t, 10, 10, 0)
	defer blank.Deinit()
	ssYCheckAdjust(t, blank, SelectionAdjustmentLeft, []ssYEndCase{
		{ex: 5, ey: 5, nx: 5, ny: 5},
		{ex: 0, ey: 9, nx: 0, ny: 9},
	})
}

// Zig: test "term51: adjust: right"
func TestSelection_term51_adjust_right(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("A1234\nB5678\nC1234\nD5678")))

	ssYCheckAdjust(t, s, SelectionAdjustmentRight, []ssYEndCase{
		{ex: 3, ey: 0, nx: 4, ny: 0},
		{ex: 0, ey: 3, nx: 1, ny: 3},
		{ex: 3, ey: 3, nx: 4, ny: 3},
		// End of the text on a row: next row with text.
		{ex: 4, ey: 0, nx: 0, ny: 1},
		{ex: 9, ey: 0, nx: 0, ny: 1},
		{ex: 5, ey: 2, nx: 0, ny: 3},
		// No text after: unchanged.
		{ex: 4, ey: 3, nx: 4, ny: 3},
		{ex: 5, ey: 3, nx: 5, ny: 3},
		{ex: 9, ey: 3, nx: 9, ny: 3},
		{ex: 0, ey: 5, nx: 0, ny: 5},
		{ex: 9, ey: 9, nx: 9, ny: 9},
	})
}

// Zig: test "term51: adjust: left and right with wide characters"
func TestSelection_term51_adjust_left_and_right_with_wide_characters(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	// Row 0: a, wide(1,2), b.
	zigTry(t, s.TestWriteString([]uint8("aあb")))
	expectEqual(t, PageCellWideWide, ssYPin(t, s, 1, 0).RowAndCell().cell.wide)
	expectEqual(t, PageCellWideSpacerTail, ssYPin(t, s, 2, 0).RowAndCell().cell.wide)

	// The spacer tail has no text, so it is skipped in both directions.
	ssYCheckAdjust(t, s, SelectionAdjustmentRight, []ssYEndCase{
		{ex: 0, ey: 0, nx: 1, ny: 0},
		{ex: 1, ey: 0, nx: 3, ny: 0},
		{ex: 2, ey: 0, nx: 3, ny: 0},
		{ex: 3, ey: 0, nx: 3, ny: 0},
	})
	ssYCheckAdjust(t, s, SelectionAdjustmentLeft, []ssYEndCase{
		{ex: 3, ey: 0, nx: 1, ny: 0},
		{ex: 2, ey: 0, nx: 1, ny: 0},
		{ex: 1, ey: 0, nx: 0, ny: 0},
		{ex: 0, ey: 0, nx: 0, ny: 0},
	})
}

// Zig: test "term51: adjust: left and right across a wide character wrap"
func TestSelection_term51_adjust_left_and_right_across_a_wide_character_wrap(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	// Nine narrow characters then a wide one that does not fit: a spacer head
	// at (9,0), the wide pair at (0,1)-(1,1), then another wide pair.
	zigTry(t, s.TestWriteString([]uint8("abcdefghiあい")))
	expectEqual(t, PageCellWideSpacerHead, ssYPin(t, s, 9, 0).RowAndCell().cell.wide)
	expect(t, ssYPin(t, s, 0, 0).RowAndCell().row.wrap, "term51YPin(&s, 0, 0).rowAndCell().row.wrap")
	expectEqual(t, PageCellWideWide, ssYPin(t, s, 0, 1).RowAndCell().cell.wide)
	expectEqual(t, PageCellWideWide, ssYPin(t, s, 2, 1).RowAndCell().cell.wide)

	ssYCheckAdjust(t, s, SelectionAdjustmentRight, []ssYEndCase{
		{ex: 8, ey: 0, nx: 0, ny: 1},
		{ex: 9, ey: 0, nx: 0, ny: 1},
		{ex: 0, ey: 1, nx: 2, ny: 1},
		{ex: 2, ey: 1, nx: 2, ny: 1},
	})
	ssYCheckAdjust(t, s, SelectionAdjustmentLeft, []ssYEndCase{
		{ex: 0, ey: 1, nx: 8, ny: 0},
		{ex: 1, ey: 1, nx: 0, ny: 1},
		{ex: 2, ey: 1, nx: 0, ny: 1},
		{ex: 3, ey: 1, nx: 2, ny: 1},
	})
}

// Zig: test "term51: adjust: left and right across a soft wrap"
func TestSelection_term51_adjust_left_and_right_across_a_soft_wrap(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("0123456789ABC")))
	expect(t, ssYPin(t, s, 0, 0).RowAndCell().row.wrap, "term51YPin(&s, 0, 0).rowAndCell().row.wrap")
	expect(t, ssYPin(t, s, 0, 1).RowAndCell().row.wrap_continuation, "term51YPin(&s, 0, 1).rowAndCell().row.wrap_continuation")

	ssYCheckAdjust(t, s, SelectionAdjustmentRight, []ssYEndCase{
		{ex: 8, ey: 0, nx: 9, ny: 0},
		{ex: 9, ey: 0, nx: 0, ny: 1},
		{ex: 1, ey: 1, nx: 2, ny: 1},
		{ex: 2, ey: 1, nx: 2, ny: 1},
	})
	ssYCheckAdjust(t, s, SelectionAdjustmentLeft, []ssYEndCase{
		{ex: 1, ey: 1, nx: 0, ny: 1},
		{ex: 0, ey: 1, nx: 9, ny: 0},
		{ex: 9, ey: 0, nx: 8, ny: 0},
		{ex: 0, ey: 0, nx: 0, ny: 0},
	})
}

// Zig: test "term51: adjust: page_up"
func TestSelection_term51_adjust_page_up(t *testing.T) {
	testStdCapacity(t)
	s := ssYHistory(t, 10, 3, 12)
	defer s.Deinit()
	expectEqual(t, SizeCellCountInt(3), s.Pages.rows)

	ssYCheckAdjust(t, s, SelectionAdjustmentPageUp, []ssYEndCase{
		{ex: 4, ey: 7, nx: 4, ny: 4},
		{ex: 9, ey: 11, nx: 9, ny: 8},
		{ex: 4, ey: 5, nx: 4, ny: 2},
		// y == rows: lands exactly on the first row.
		{ex: 4, ey: 3, nx: 4, ny: 0},
		// y < rows: goes home (x reset as well).
		{ex: 4, ey: 2, nx: 0, ny: 0},
		{ex: 4, ey: 1, nx: 0, ny: 0},
		{ex: 9, ey: 0, nx: 0, ny: 0},
		{ex: 0, ey: 0, nx: 0, ny: 0},
	})

	// A screen with as many total rows as active rows: always home.
	full := ssScreenInit(t, 10, 10, 0)
	defer full.Deinit()
	ssYCheckAdjust(t, full, SelectionAdjustmentPageUp, []ssYEndCase{
		{ex: 2, ey: 9, nx: 0, ny: 0},
		{ex: 2, ey: 0, nx: 0, ny: 0},
	})
}

// Zig: test "term51: adjust: page_down"
func TestSelection_term51_adjust_page_down(t *testing.T) {
	testStdCapacity(t)
	s := ssYHistory(t, 10, 3, 12)
	defer s.Deinit()

	ssYCheckAdjust(t, s, SelectionAdjustmentPageDown, []ssYEndCase{
		{ex: 4, ey: 0, nx: 4, ny: 3},
		{ex: 4, ey: 5, nx: 4, ny: 8},
		// y + rows == last row: still a plain move.
		{ex: 4, ey: 8, nx: 4, ny: 11},
		// One more: overflows, goes to the end of the last row with text.
		{ex: 4, ey: 9, nx: 9, ny: 11},
		{ex: 4, ey: 10, nx: 9, ny: 11},
		{ex: 4, ey: 11, nx: 9, ny: 11},
	})

	// Text only in the first rows: page_down overflows to `.end`, which can
	// move the end pin UP to the last row with text.
	tt := ssScreenInit(t, 10, 10, 0)
	defer tt.Deinit()
	zigTry(t, tt.TestWriteString([]uint8("A1234\nB5678\nC1234\nD5678")))
	ssYCheckAdjust(t, tt, SelectionAdjustmentPageDown, []ssYEndCase{
		{ex: 3, ey: 0, nx: 9, ny: 3},
		{ex: 2, ey: 6, nx: 9, ny: 3},
		{ex: 2, ey: 9, nx: 9, ny: 3},
	})
}

// Zig: test "term51: adjust: home"
func TestSelection_term51_adjust_home(t *testing.T) {
	testStdCapacity(t)
	s := ssYHistory(t, 10, 3, 12)
	defer s.Deinit()
	// Home is the top-left of the whole screen, including history.
	ssYCheckAdjust(t, s, SelectionAdjustmentHome, []ssYEndCase{
		{ex: 4, ey: 7, nx: 0, ny: 0},
		{ex: 9, ey: 11, nx: 0, ny: 0},
		{ex: 0, ey: 0, nx: 0, ny: 0},
		{ex: 9, ey: 0, nx: 0, ny: 0},
	})
	sel := SelectionInit(ssYPin(t, s, 7, 0), ssYPin(t, s, 4, 5), false)
	sel.Adjust(s, SelectionAdjustmentHome)
	expect(t, s.Pages.pages.first != nil, "s.pages.pages.first.?")
	expect(t, sel.End().Node == s.Pages.pages.first, "sel.end().node == s.pages.pages.first.?")
}

// Zig: test "term51: adjust: end"
func TestSelection_term51_adjust_end(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("A1234\nB5678\nC1234\nD5678")))

	// The end of the last row with text, regardless of where the pin was.
	ssYCheckAdjust(t, s, SelectionAdjustmentEnd, []ssYEndCase{
		{ex: 0, ey: 0, nx: 9, ny: 3},
		{ex: 3, ey: 3, nx: 9, ny: 3},
		{ex: 2, ey: 8, nx: 9, ny: 3},
		{ex: 9, ey: 9, nx: 9, ny: 3},
	})

	// A background-colour-only row below does not count; a row of spaces does.
	ssYSetBg(t, s, 0, 5)
	ssYCheckAdjust(t, s, SelectionAdjustmentEnd, []ssYEndCase{{ex: 0, ey: 0, nx: 9, ny: 3}})
	ssYSetText(t, s, 0, 6, " ")
	ssYCheckAdjust(t, s, SelectionAdjustmentEnd, []ssYEndCase{{ex: 0, ey: 0, nx: 9, ny: 6}})

	// Text on the very last row.
	ssYSetText(t, s, 3, 9, "z")
	ssYCheckAdjust(t, s, SelectionAdjustmentEnd, []ssYEndCase{{ex: 0, ey: 0, nx: 9, ny: 9}})

	// No text anywhere: unchanged.
	blank := ssScreenInit(t, 10, 10, 0)
	defer blank.Deinit()
	ssYCheckAdjust(t, blank, SelectionAdjustmentEnd, []ssYEndCase{
		{ex: 3, ey: 3, nx: 3, ny: 3},
		{ex: 0, ey: 9, nx: 0, ny: 9},
	})
}

// Zig: test "term51: adjust: beginning_of_line"
func TestSelection_term51_adjust_beginning_of_line(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	ssYCheckAdjust(t, s, SelectionAdjustmentBeginningOfLine, []ssYEndCase{
		{ex: 5, ey: 1, nx: 0, ny: 1},
		{ex: 0, ey: 1, nx: 0, ny: 1},
		{ex: 9, ey: 9, nx: 0, ny: 9},
		{ex: 9, ey: 0, nx: 0, ny: 0},
	})
}

// Zig: test "term51: adjust: end_of_line"
func TestSelection_term51_adjust_end_of_line(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	ssYCheckAdjust(t, s, SelectionAdjustmentEndOfLine, []ssYEndCase{
		{ex: 5, ey: 1, nx: 9, ny: 1},
		{ex: 9, ey: 1, nx: 9, ny: 1},
		{ex: 0, ey: 9, nx: 9, ny: 9},
		{ex: 0, ey: 0, nx: 9, ny: 0},
	})
}

// Zig: test "term51: adjust: tracked selection moves the tracked end pin in place"
func TestSelection_term51_adjust_tracked_selection_moves_the_tracked_end_pin_in_place(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("A1234\nB5678\nC1234\nD5678")))

	untracked := SelectionInit(ssYPin(t, s, 5, 1), ssYPin(t, s, 3, 3), false)
	sel, err := untracked.Track(s)
	zigTry(t, err)
	defer sel.Deinit(s)
	end_ptr := sel.Bounds.Tracked.End
	start_ptr := sel.Bounds.Tracked.Start
	base := s.Pages.CountTrackedPins()

	all := []SelectionAdjustment{
		SelectionAdjustmentLeft, SelectionAdjustmentRight, SelectionAdjustmentUp, SelectionAdjustmentDown,
		SelectionAdjustmentPageUp, SelectionAdjustmentPageDown, SelectionAdjustmentHome, SelectionAdjustmentEnd,
		SelectionAdjustmentBeginningOfLine, SelectionAdjustmentEndOfLine,
	}
	for _, adj := range all {
		sel.Adjust(s, adj)
		// The same pins, never replaced, and the start never moves.
		expect(t, sel.Bounds.Tracked.End == end_ptr, "sel.bounds.tracked.end == end_ptr")
		expect(t, sel.Bounds.Tracked.Start == start_ptr, "sel.bounds.tracked.start == start_ptr")
		ssYExpectPin(t, s, sel.Start(), 5, 1)
		expect(t, sel.Tracked(), "sel.tracked()")
		expectEqual(t, base, s.Pages.CountTrackedPins())
	}

	// And the move itself is visible through the tracked pin.
	sel.Adjust(s, SelectionAdjustmentHome)
	ssYExpectPin(t, s, *end_ptr, 0, 0)
	sel.Adjust(s, SelectionAdjustmentDown)
	ssYExpectPin(t, s, *end_ptr, 0, 1)
}

// Zig: test "term51: adjust: page boundaries"
func TestSelection_term51_adjust_page_boundaries(t *testing.T) {
	testStdCapacity(t)
	s := ssYPages(t, 2)
	defer s.Deinit()
	n := ssYFirstRows(s)
	expect(t, s.Pages.pages.first != nil, "s.pages.pages.first.?")
	first := s.Pages.pages.first
	expect(t, s.Pages.pages.last != nil, "s.pages.pages.last.?")
	last := s.Pages.pages.last
	// "ab" ends page one, "cd" starts page two, "ef" is three rows in.
	ssYSetText(t, s, 0, n-1, "ab")
	ssYSetText(t, s, 0, n, "cd")
	ssYSetText(t, s, 3, n+3, "ef")

	// up: from the first row of page two to the last row of page one.
	sel := SelectionInit(ssYPin(t, s, 7, 0), ssYPin(t, s, 5, n), false)
	sel.Adjust(s, SelectionAdjustmentUp)
	ssYExpectPin(t, s, sel.End(), 5, n-1)
	expect(t, sel.End().Node == first, "sel.end().node == first")
	sel.Adjust(s, SelectionAdjustmentUp)
	ssYExpectPin(t, s, sel.End(), 5, n-2)

	// down: skipping blank rows, across the boundary, then to end of line.
	sel = SelectionInit(ssYPin(t, s, 7, 0), ssYPin(t, s, 5, n-4), false)
	sel.Adjust(s, SelectionAdjustmentDown)
	ssYExpectPin(t, s, sel.End(), 5, n-1)
	expect(t, sel.End().Node == first, "sel.end().node == first")
	sel.Adjust(s, SelectionAdjustmentDown)
	ssYExpectPin(t, s, sel.End(), 5, n)
	expect(t, sel.End().Node == last, "sel.end().node == last")
	sel.Adjust(s, SelectionAdjustmentDown)
	ssYExpectPin(t, s, sel.End(), 5, n+3)
	sel.Adjust(s, SelectionAdjustmentDown)
	ssYExpectPin(t, s, sel.End(), 199, n+3)

	// left: from the start of page two to the last text of page one.
	sel = SelectionInit(ssYPin(t, s, 7, 0), ssYPin(t, s, 0, n), false)
	sel.Adjust(s, SelectionAdjustmentLeft)
	ssYExpectPin(t, s, sel.End(), 1, n-1)
	expect(t, sel.End().Node == first, "sel.end().node == first")
	// left across blank rows within page two.
	sel = SelectionInit(ssYPin(t, s, 7, 0), ssYPin(t, s, 3, n+3), false)
	sel.Adjust(s, SelectionAdjustmentLeft)
	ssYExpectPin(t, s, sel.End(), 1, n)

	// right: across the boundary and over blank rows.
	sel = SelectionInit(ssYPin(t, s, 7, 0), ssYPin(t, s, 1, n-1), false)
	sel.Adjust(s, SelectionAdjustmentRight)
	ssYExpectPin(t, s, sel.End(), 0, n)
	expect(t, sel.End().Node == last, "sel.end().node == last")
	sel.Adjust(s, SelectionAdjustmentRight)
	ssYExpectPin(t, s, sel.End(), 1, n)
	sel.Adjust(s, SelectionAdjustmentRight)
	ssYExpectPin(t, s, sel.End(), 3, n+3)
	sel.Adjust(s, SelectionAdjustmentRight)
	ssYExpectPin(t, s, sel.End(), 4, n+3)
	sel.Adjust(s, SelectionAdjustmentRight)
	ssYExpectPin(t, s, sel.End(), 4, n+3)

	// page_up across the boundary (rows = 5).
	sel = SelectionInit(ssYPin(t, s, 7, 0), ssYPin(t, s, 5, n+2), false)
	sel.Adjust(s, SelectionAdjustmentPageUp)
	ssYExpectPin(t, s, sel.End(), 5, n-3)
	expect(t, sel.End().Node == first, "sel.end().node == first")
	sel = SelectionInit(ssYPin(t, s, 7, 0), ssYPin(t, s, 5, n+4), false)
	sel.Adjust(s, SelectionAdjustmentPageUp)
	ssYExpectPin(t, s, sel.End(), 5, n-1)

	// page_down across the boundary, and the overflow into `.end`.
	sel = SelectionInit(ssYPin(t, s, 7, 0), ssYPin(t, s, 5, n-3), false)
	sel.Adjust(s, SelectionAdjustmentPageDown)
	ssYExpectPin(t, s, sel.End(), 5, n+2)
	expect(t, sel.End().Node == last, "sel.end().node == last")
	sel = SelectionInit(ssYPin(t, s, 7, 0), ssYPin(t, s, 5, n+1), false)
	sel.Adjust(s, SelectionAdjustmentPageDown)
	ssYExpectPin(t, s, sel.End(), 199, n+3)

	// home / end / beginning_of_line / end_of_line.
	sel = SelectionInit(ssYPin(t, s, 7, 0), ssYPin(t, s, 5, n+1), false)
	sel.Adjust(s, SelectionAdjustmentHome)
	ssYExpectPin(t, s, sel.End(), 0, 0)
	expect(t, sel.End().Node == first, "sel.end().node == first")
	sel.Adjust(s, SelectionAdjustmentEnd)
	ssYExpectPin(t, s, sel.End(), 199, n+3)
	expect(t, sel.End().Node == last, "sel.end().node == last")
	sel.Adjust(s, SelectionAdjustmentBeginningOfLine)
	ssYExpectPin(t, s, sel.End(), 0, n+3)
	sel.Adjust(s, SelectionAdjustmentEndOfLine)
	ssYExpectPin(t, s, sel.End(), 199, n+3)
}

// Zig: test "term51: adjust: end finds text on an earlier page"
func TestSelection_term51_adjust_end_finds_text_on_an_earlier_page(t *testing.T) {
	testStdCapacity(t)
	s := ssYPages(t, 2)
	defer s.Deinit()
	n := ssYFirstRows(s)
	ssYSetText(t, s, 0, n-2, "x")

	// Page two is blank: `.end` goes to the last column of the last text row,
	// which is on page one.
	sel := SelectionInit(ssYPin(t, s, 7, 0), ssYPin(t, s, 5, n+2), false)
	sel.Adjust(s, SelectionAdjustmentEnd)
	ssYExpectPin(t, s, sel.End(), 199, n-2)
	expect(t, s.Pages.pages.first != nil, "s.pages.pages.first.?")
	expect(t, sel.End().Node == s.Pages.pages.first, "sel.end().node == s.pages.pages.first.?")
}

// Go-only: panic test: Selection.track: the selection is already tracked (assert(!self.tracked()))
func TestSelection_panic_track_already_tracked(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	sel := SelectionInit(ssYPin(t, s, 2, 1), ssYPin(t, s, 7, 4), false)
	tsel, err := sel.Track(s)
	zigTry(t, err)
	defer tsel.Deinit(s)

	ssExpectPanic(t, func() { _, _ = tsel.Track(s) })
}

// Go-only: panic test: Selection.order: a pin is not in the screen (`pointFromPin(...).?` on null)
func TestSelection_panic_order_pin_not_in_screen(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	foreign := PageListPin{Node: new(PageListNode)}
	sel := SelectionInit(ssYPin(t, s, 2, 1), foreign, false)

	ssExpectPanic(t, func() { _ = sel.Order(s) })
}

// Go-only: panic test: Selection.contains: the pin is not in the screen (`pointFromPin(...).?` on null)
func TestSelection_panic_contains_pin_not_in_screen(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	sel := SelectionInit(ssYPin(t, s, 2, 1), ssYPin(t, s, 7, 4), false)
	foreign := PageListPin{Node: new(PageListNode)}

	ssExpectPanic(t, func() { _ = sel.Contains(s, foreign) })
}

// Go-only: panic test: Selection.containedRow: the pin is not in the screen (`pointFromPin(...).?` on null)
func TestSelection_panic_containedRow_pin_not_in_screen(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	sel := SelectionInit(ssYPin(t, s, 2, 1), ssYPin(t, s, 7, 4), false)
	foreign := PageListPin{Node: new(PageListNode)}

	ssExpectPanic(t, func() { _ = sel.ContainedRow(s, foreign) })
}
