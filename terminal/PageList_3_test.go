// Tests translated one for one from src/terminal/PageList.zig (Ghostty
// v1.3.1 plus the term51 additional tests), the tests from
// "term51: reset: multiple pages of history collapse to the page count of
// the active area" up to and including "term51: ReflowCursor.reflowRow:
// unwrapping drops the spacer head". Each Go test carries the exact Zig
// title in a "// Zig: test" comment. Helpers are prefixed pl3.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualSlices / expectEqualStrings keep Zig's names
//     and Zig's (expected, actual) argument order. `try` on a call that
//     returns an error is zigTry(t, err); `x.?` is zigUnwrap.
//   - Every test starts with testStdCapacity(t): Zig's unit tests run with a
//     standard page capacity of grapheme_bytes 512.
//   - `init(alloc, cols, rows, max)` is pl3Init; the allocator parameters
//     are gone, `defer s.deinit()` is `defer s.Deinit()`, and
//     `std.testing.allocator` leak checking has no counterpart (nothing is
//     written for it).
//   - `std_size` is pageListStdSize(), `PagePool.item_size` is
//     pageListPagePoolItemSize(), `node.data.memory.len` is
//     pageListMemoryLen(&node.data), `std_capacity` is PageStdCapacity.
//   - `s.pin(pt).?` is pl3Pin; `getCell(pt).?.cell.content.codepoint` is
//     pl3Cp; a point literal `.{ .screen = .{ .x = x, .y = y } }` is
//     pl3Screen(x, y) (pl3Active, pl3History, pl3Viewport likewise).
//   - Raw-pointer, pool and memory-offset assertions are dropped with a
//     `// Zig-only:` comment.
//   - Zig's default option values are written out in every literal
//     (Resize.reflow = true, the page iterator direction, ...).
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt.

package terminal

import (
	"errors"
	"math"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers (term51J, K).
// ---------------------------------------------------------------------------

// pl3ExpectPanic runs f and fails the test if f does not panic. A panic with
// the wireframe's "not implemented" value is not the expected panic.
func pl3ExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("expected a panic, got none")
		}
		if s, ok := r.(string); ok && s == notImplemented {
			t.Fatalf("got the wireframe panic %q, not the assertion", s)
		}
	}()
	f()
}

// pl3Screen is the Zig literal `.{ .screen = .{ .x = x, .y = y } }`.
func pl3Screen(x int, y int) Point {
	return Point{Tag: PointTagScreen, Screen: PointCoordinate{X: SizeCellCountInt(x), Y: uint32(y)}}
}

// pl3Active is the Zig literal `.{ .active = .{ .x = x, .y = y } }`.
func pl3Active(x int, y int) Point {
	return Point{Tag: PointTagActive, Active: PointCoordinate{X: SizeCellCountInt(x), Y: uint32(y)}}
}

// pl3History is the Zig literal `.{ .history = .{ .x = x, .y = y } }`.
func pl3History(x int, y int) Point {
	return Point{Tag: PointTagHistory, History: PointCoordinate{X: SizeCellCountInt(x), Y: uint32(y)}}
}

// pl3Viewport is the Zig literal `.{ .viewport = .{ .x = x, .y = y } }`.
func pl3Viewport(x int, y int) Point {
	return Point{Tag: PointTagViewport, Viewport: PointCoordinate{X: SizeCellCountInt(x), Y: uint32(y)}}
}

// pl3Init is `try init(alloc, cols, rows, max_size)`. The list is returned by
// pointer so the test can use it as `&s`.
func pl3Init(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt, max_size Optional[uint64]) *PageList {
	t.Helper()
	s, err := PageListInit(cols, rows, max_size)
	zigTry(t, err)
	return &s
}

// pl3Pin is `s.pin(pt).?`.
func pl3Pin(t *testing.T, s *PageList, pt Point) PageListPin {
	t.Helper()
	return zigUnwrap(t, s.Pin(pt))
}

// pl3Track is `try s.trackPin(p)`.
func pl3Track(t *testing.T, s *PageList, p PageListPin) *PageListPin {
	t.Helper()
	tracked, err := s.TrackPin(p)
	zigTry(t, err)
	return tracked
}

// pl3GrowRows is `try s.growRows(n)`.
func pl3GrowRows(t *testing.T, s *PageList, n uint) {
	t.Helper()
	zigTry(t, s.growRows(n))
}

// pl3Verify is `try s.verifyIntegrity()`.
func pl3Verify(t *testing.T, s *PageList) {
	t.Helper()
	zigTry(t, s.verifyIntegrity())
}

// pl3Clone is `try s.clone(alloc, opts)`.
func pl3Clone(t *testing.T, s *PageList, opts PageListClone) *PageList {
	t.Helper()
	c, err := s.Clone(opts)
	zigTry(t, err)
	return &c
}

// pl3Codepoint is the Zig literal `.{ .content_tag = .codepoint, .content =
// .{ .codepoint = cp } }`.
func pl3Codepoint(cp uint32) PageCell {
	return PageCell{
		content_tag: PageCellContentTagCodepoint,
		content:     PageCellContent{codepoint: cp},
	}
}

// pl3Bold is the style `.{ .flags = .{ .bold = true } }`.
func pl3Bold() Style {
	return Style{Flags: StyleFlags{Bold: true}}
}

// pl3SetCell is term51JSetCell: writes a codepoint into the cell the pin
// points at.
func pl3SetCell(p PageListPin, cp uint32) {
	rac := p.RowAndCell()
	*rac.cell = pl3Codepoint(cp)
}

// pl3Cp is term51JCp: reads the codepoint at the point (0 for an empty cell).
func pl3Cp(t *testing.T, s *PageList, pt Point) uint32 {
	t.Helper()
	return zigUnwrap(t, s.GetCell(pt)).Cell.content.codepoint
}

// pl3MarkRows is term51JMarkRows: marks every screen row of the list with a
// distinct codepoint in x=0: row y holds 0x100 + y.
func pl3MarkRows(t *testing.T, s *PageList) {
	t.Helper()
	total := s.totalRows()
	for y := 0; y < int(total); y++ {
		p := pl3Pin(t, s, pl3Screen(0, y))
		pl3SetCell(p, uint32(0x100+y))
	}
}

// pl3ExpectPages is term51JExpectPages: asserts the list has exactly the
// given per-page row counts, that the intrusive links are consistent, and
// that the cached accounting (total_rows, page_size) matches the pages.
func pl3ExpectPages(t *testing.T, s *PageList, expected []uint) {
	t.Helper()
	var count uint
	var rows uint
	var bytes uint64
	var prev *PageListNode
	for node := s.pages.first; node != nil; node = node.next {
		expect(t, count < uint(len(expected)), "count < expected.len")
		expectEqual(t, expected[count], uint(node.data.size.rows))
		expect(t, node.prev == prev, "node.prev == prev")
		rows += uint(node.data.size.rows)
		bytes += pageListMemoryLen(&node.data)
		prev = node
		count += 1
	}
	expectEqual(t, uint(len(expected)), count)
	expect(t, s.pages.last == prev, "s.pages.last == prev")
	expectEqual(t, rows, s.total_rows)
	expectEqual(t, bytes, s.page_size)
	expectEqual(t, uint(len(expected)), s.TotalPages())
}

// pl3Column is term51JColumn: returns the codepoints of x=0 for every screen
// row of the list.
func pl3Column(t *testing.T, s *PageList) []uint32 {
	t.Helper()
	total := s.totalRows()
	out := make([]uint32, total)
	for y := range out {
		out[y] = pl3Cp(t, s, pl3Screen(0, y))
	}
	return out
}

// pl3Viewport tag values.
var (
	pl3ViewportActive = PageListViewport{Tag: PageListViewportActive}
	pl3ViewportTop    = PageListViewport{Tag: PageListViewportTop}
	pl3ViewportPin    = PageListViewport{Tag: PageListViewportPin}
)

// ---------------------------------------------------------------------------
// reset
// ---------------------------------------------------------------------------

// Zig: test "term51: reset: multiple pages of history collapse to the page count of the active area"
func TestPl3_term51_reset_multiple_pages_of_history_collapse_to_the_page_count_of_the_active_area(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 500, 24, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 250)
	pl3ExpectPages(t, s, []uint{93, 93, 88})
	expectEqual(t, uint64(3), s.page_serial)

	s.Reset()
	pl3ExpectPages(t, s, []uint{24})
	expectEqual(t, uint64(4), s.page_serial)
	expectEqual(t, uint64(3), s.pages.first.serial)
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	expect(t, s.pages.first.prev == nil, "s.pages.first.?.prev == null")
	expect(t, s.pages.first.next == nil, "s.pages.first.?.next == null")
	pl3Verify(t, s)
}

// Zig: test "term51: reset: active area spanning several pages"
func TestPl3_term51_reset_active_area_spanning_several_pages(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 500, 200, None[uint64]())
	defer s.Deinit()
	pl3ExpectPages(t, s, []uint{93, 93, 14})
	pl3GrowRows(t, s, 400)
	expect(t, s.TotalPages() > 3, "s.totalPages() > 3")

	s.Reset()
	pl3ExpectPages(t, s, []uint{93, 93, 14})
	expectEqual(t, uint(200), s.total_rows)
	expectEqual(t, 3*pageListStdSize(), s.page_size)
	pl3Verify(t, s)

	// Pages have fresh serial numbers in order.
	serial := s.pages.first.serial
	for node := s.pages.first.next; node != nil; node = node.next {
		expectEqual(t, serial+1, node.serial)
		serial = node.serial
	}
	expectEqual(t, uint64(s.page_serial), serial+1)
}

// Zig: test "term51: reset: one row per page"
func TestPl3_term51_reset_one_row_per_page(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 46950, 3, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 5)
	pl3ExpectPages(t, s, []uint{1, 1, 1, 1, 1, 1, 1, 1})

	s.Reset()
	pl3ExpectPages(t, s, []uint{1, 1, 1})
	expectEqual(t, 3*pageListStdSize(), s.page_size)
	expectEqual(t, uint64(11), s.page_serial)
	pl3Verify(t, s)
}

// Zig: test "term51: reset: non-standard pages are freed and rebuilt"
func TestPl3_term51_reset_non_standard_pages_are_freed_and_rebuilt(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, zigUnwrap(t, PageStdCapacity.MaxCols())+1, 216, None[uint64]())
	defer s.Deinit()
	pl3ExpectPages(t, s, []uint{215, 1})
	pl3SetCell(pl3Pin(t, s, pl3Screen(46950, 215)), 'n')

	s.Reset()
	// Zig-only: the old heap pages are freed (leak checked by the testing allocator)
	// They are replaced by two new non-standard pages.
	pl3ExpectPages(t, s, []uint{215, 1})
	expectEqual(t, uint64(2*80838656), s.page_size)
	expectEqual(t, uint64(80838656), pageListMemoryLen(&s.pages.first.data))
	expectEqual(t, uint64(4), s.page_serial)
	expectEqual(t, uint32(0), pl3Cp(t, s, pl3Screen(46950, 215)))
	pl3Verify(t, s)
}

// Zig: test "term51: reset: non-standard history shrinks to the active area"
func TestPl3_term51_reset_non_standard_history_shrinks_to_the_active_area(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, zigUnwrap(t, PageStdCapacity.MaxCols())+1, 3, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 3)
	pl3ExpectPages(t, s, []uint{6})
	s.Reset()
	pl3ExpectPages(t, s, []uint{3})
	expectEqual(t, uint64(80838656), s.page_size)
	pl3Verify(t, s)
}

// Zig: test "term51: reset: tracked pins move to the top-left and become garbage"
func TestPl3_term51_reset_tracked_pins_move_to_the_top_left_and_become_garbage(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 500, 24, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 250)

	a := pl3Track(t, s, pl3Pin(t, s, pl3Screen(7, 3)))
	b := pl3Track(t, s, pl3Pin(t, s, pl3Screen(499, 273)))
	c := pl3Track(t, s, pl3Pin(t, s, pl3Screen(0, 100)))
	c.Garbage = true
	defer s.UntrackPin(a)
	defer s.UntrackPin(b)
	defer s.UntrackPin(c)
	expectEqual(t, uint(4), s.CountTrackedPins())

	// Move the viewport to a pin so it is not at the top-left.
	s.viewport = pl3ViewportPin
	*s.viewport_pin = pl3Pin(t, s, pl3Screen(4, 120))
	s.viewport_pin_row_offset = Some[uint](120)

	s.Reset()

	first := s.pages.first
	for _, p := range []*PageListPin{a, b, c} {
		expect(t, p.Node == first, "p.node == first")
		expectEqual(t, SizeCellCountInt(0), p.X)
		expectEqual(t, SizeCellCountInt(0), p.Y)
		expect(t, p.Garbage, "p.garbage")
	}

	// Still tracked, count unchanged.
	expectEqual(t, uint(4), s.CountTrackedPins())

	// The viewport pin is moved too but is never garbage, and the
	// viewport returns to the active area.
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	expect(t, s.viewport_pin.Node == first, "s.viewport_pin.node == first")
	expectEqual(t, SizeCellCountInt(0), s.viewport_pin.X)
	expectEqual(t, SizeCellCountInt(0), s.viewport_pin.Y)
	expect(t, !s.viewport_pin.Garbage, "!s.viewport_pin.garbage")

	// Pin objects stay at the same address and valid for the new list.
	pl3Verify(t, s)
}

// Zig: test "term51: reset: viewport variants all return to active"
func TestPl3_term51_reset_viewport_variants_all_return_to_active(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 60)

	s.viewport = pl3ViewportTop
	s.Reset()
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	pl3GrowRows(t, s, 60)
	s.viewport = pl3ViewportPin
	*s.viewport_pin = pl3Pin(t, s, pl3Screen(0, 10))
	s.viewport_pin_row_offset = Some[uint](10)
	s.Reset()
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	// The cached row offset is not cleared by reset (only meaningful for
	// a pin viewport).
	expectEqual(t, Some[uint](10), s.viewport_pin_row_offset)

	s.Reset()
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	pl3Verify(t, s)
}

// Zig: test "term51: reset: page contents and dirty state are cleared"
func TestPl3_term51_reset_page_contents_and_dirty_state_are_cleared(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Write text with a style, a wrapped row, and mark things dirty.
	page := &s.pages.first.data
	style_id, err := page.styles.Add(pl3Bold())
	zigTry(t, err)
	for y := 0; y < 24; y++ {
		p := pl3Pin(t, s, pl3Active(0, y))
		rac := p.RowAndCell()
		rac.row.wrap = true
		rac.row.styled = true
		*rac.cell = PageCell{
			content_tag: PageCellContentTagCodepoint,
			content:     PageCellContent{codepoint: 'Z'},
			style_id:    style_id,
		}
		page.styles.Use(style_id)
	}
	page.styles.Release(style_id)
	expectEqual(t, uint(1), page.styles.Count())
	page.dirty = true
	s.markDirty(pl3Active(0, 3))

	s.Reset()

	new_page := &s.pages.first.data
	expectEqual(t, uint(0), new_page.styles.Count())
	expect(t, !new_page.dirty, "!new_page.dirty")
	for y := 0; y < 24; y++ {
		rac := new_page.GetRowAndCell(0, uint(y))
		expectEqual(t, uint32(0), rac.cell.content.codepoint)
		expectEqual(t, uint16(0), rac.cell.style_id)
		expect(t, !rac.row.wrap, "!rac.row.wrap")
		expect(t, !rac.row.styled, "!rac.row.styled")
		expect(t, !rac.row.dirty, "!rac.row.dirty")
	}
	expect(t, !s.IsDirty(pl3Active(0, 3)), "!s.isDirty(.{ .active = .{ .x = 0, .y = 3 } })")
}

// Zig: test "term51: reset: repeated resets keep counting serials"
func TestPl3_term51_reset_repeated_resets_keep_counting_serials(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 500, 100, None[uint64]())
	defer s.Deinit()
	pl3ExpectPages(t, s, []uint{93, 7})
	expectEqual(t, uint64(2), s.page_serial)

	s.Reset()
	pl3ExpectPages(t, s, []uint{93, 7})
	expectEqual(t, uint64(4), s.page_serial)
	expectEqual(t, uint64(2), s.pages.first.serial)

	s.Reset()
	pl3ExpectPages(t, s, []uint{93, 7})
	expectEqual(t, uint64(6), s.page_serial)
	expectEqual(t, uint64(4), s.pages.first.serial)
	expectEqual(t, uint64(5), s.pages.last.serial)
	expectEqual(t, uint64(0), s.page_serial_min)
}

// Zig: test "term51: reset: preserves paused integrity state"
func TestPl3_term51_reset_preserves_paused_integrity_state(t *testing.T) {
	testStdCapacity(t)
	// Zig: if (comptime !build_options.slow_runtime_safety) return error.SkipZigTest;
	if !pageSlowRuntimeSafety {
		t.Skip("needs slow_runtime_safety")
	}
	s := pl3Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	s.PauseIntegrityChecks(true)
	s.Reset()
	expectEqual(t, uint64(1), s.pause_integrity_checks)
	s.PauseIntegrityChecks(false)
	pl3Verify(t, s)
}

// Zig: test "term51: reset: smallest list"
func TestPl3_term51_reset_smallest_list(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 1, 1, None[uint64]())
	defer s.Deinit()
	pl3SetCell(pl3Pin(t, s, pl3Active(0, 0)), 'q')
	pl3GrowRows(t, s, 5)
	s.Reset()
	pl3ExpectPages(t, s, []uint{1})
	expectEqual(t, uint32(0), pl3Cp(t, s, pl3Screen(0, 0)))
}

// ---------------------------------------------------------------------------
// clone
// ---------------------------------------------------------------------------

// Zig: test "term51: clone: full clone copies fields"
func TestPl3_term51_clone_full_clone_copies_fields(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 80, 24, Some[uint64](987654321))
	defer s.Deinit()
	pl3GrowRows(t, s, 30)
	pl3MarkRows(t, s)

	c := pl3Clone(t, s, PageListClone{Top: pl3Screen(0, 0)})
	defer c.Deinit()

	pl3ExpectPages(t, c, []uint{54})
	expectEqual(t, SizeCellCountInt(80), c.cols)
	expectEqual(t, SizeCellCountInt(24), c.rows)
	expectEqual(t, uint(54), c.total_rows)
	expectEqual(t, uint64(987654321), c.explicit_max_size)
	expectEqual(t, s.min_max_size, c.min_max_size)
	expectEqual(t, uint64(1), c.page_serial)
	expectEqual(t, uint64(0), c.page_serial_min)
	expectEqual(t, uint64(0), c.pages.first.serial)
	expectEqual(t, pageListStdSize(), c.page_size)
	expect(t, c.viewport.Tag == PageListViewportActive, "c.viewport == .active")
	expectEqual(t, None[uint](), c.viewport_pin_row_offset)
	// Zig-only: try std.testing.expect(c.pool.alloc.ptr == alloc.ptr);

	// Page capacity is the source page's capacity, independent memory.
	sp := s.pages.first
	cp := c.pages.first
	expect(t, sp != cp, "sp != cp")
	// Zig-only: try std.testing.expect(sp.data.memory.ptr != cp.data.memory.ptr);
	expectEqual(t, sp.data.capacity, cp.data.capacity)
	expectEqual(t, sp.data.size.cols, cp.data.size.cols)

	// Only the viewport pin is tracked, at the top-left of the clone.
	expectEqual(t, uint(1), c.CountTrackedPins())
	expect(t, c.TrackedPins()[0] == c.viewport_pin, "c.trackedPins()[0] == c.viewport_pin")
	expect(t, c.viewport_pin != s.viewport_pin, "c.viewport_pin != s.viewport_pin")
	expect(t, c.viewport_pin.Node == cp, "c.viewport_pin.node == cp")
	expectEqual(t, SizeCellCountInt(0), c.viewport_pin.Y)
	expectEqual(t, SizeCellCountInt(0), c.viewport_pin.X)
	expect(t, !c.viewport_pin.Garbage, "!c.viewport_pin.garbage")

	// All contents copied.
	for y := 0; y < 54; y++ {
		expectEqual(t, uint32(0x100+y), pl3Cp(t, c, pl3Screen(0, y)))
	}

	// The clone is independent of the source.
	pl3SetCell(pl3Pin(t, c, pl3Screen(0, 0)), 'X')
	expectEqual(t, uint32(0x100), pl3Cp(t, s, pl3Screen(0, 0)))
	pl3Verify(t, s)
}

// Zig: test "term51: clone: does not copy the pause counter or pins of the source"
func TestPl3_term51_clone_does_not_copy_the_pause_counter_or_pins_of_the_source(t *testing.T) {
	testStdCapacity(t)
	// Zig: if (comptime !build_options.slow_runtime_safety) return error.SkipZigTest;
	if !pageSlowRuntimeSafety {
		t.Skip("needs slow_runtime_safety")
	}
	s := pl3Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	p := pl3Track(t, s, pl3Pin(t, s, pl3Active(2, 2)))
	defer s.UntrackPin(p)
	s.PauseIntegrityChecks(true)
	defer s.PauseIntegrityChecks(false)

	c := pl3Clone(t, s, PageListClone{Top: pl3Screen(0, 0)})
	defer c.Deinit()
	expectEqual(t, uint64(0), c.pause_integrity_checks)
	expectEqual(t, uint(1), c.CountTrackedPins())
	expectEqual(t, uint(2), s.CountTrackedPins())
}

// Zig: test "term51: clone: viewport of the source is ignored"
func TestPl3_term51_clone_viewport_of_the_source_is_ignored(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 30)

	s.viewport = pl3ViewportTop
	{
		c := pl3Clone(t, s, PageListClone{Top: pl3Screen(0, 0)})
		defer c.Deinit()
		expect(t, c.viewport.Tag == PageListViewportActive, "c.viewport == .active")
		expectEqual(t, None[uint](), c.viewport_pin_row_offset)
	}

	s.viewport = pl3ViewportPin
	*s.viewport_pin = pl3Pin(t, s, pl3Screen(0, 5))
	s.viewport_pin_row_offset = Some[uint](5)
	{
		c := pl3Clone(t, s, PageListClone{Top: pl3Screen(0, 0)})
		defer c.Deinit()
		expect(t, c.viewport.Tag == PageListViewportActive, "c.viewport == .active")
		expectEqual(t, None[uint](), c.viewport_pin_row_offset)
		expect(t, c.viewport_pin.Node == c.pages.first, "c.viewport_pin.node == c.pages.first.?")
		expectEqual(t, SizeCellCountInt(0), c.viewport_pin.Y)
	}

	// The source is unchanged.
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, Some[uint](5), s.viewport_pin_row_offset)
	s.viewport = pl3ViewportActive
}

// Zig: test "term51: clone: top and bottom bounds are inclusive rows"
func TestPl3_term51_clone_top_and_bottom_bounds_are_inclusive_rows(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 30)
	pl3MarkRows(t, s)

	type Case struct {
		top  SizeCellCountInt
		bot  Optional[SizeCellCountInt]
		rows uint
	}
	for _, case_ := range []Case{
		{top: 0, bot: None[SizeCellCountInt](), rows: 54},
		{top: 0, bot: Some[SizeCellCountInt](53), rows: 54},
		{top: 0, bot: Some[SizeCellCountInt](52), rows: 53},
		{top: 0, bot: Some[SizeCellCountInt](23), rows: 24},
		{top: 10, bot: None[SizeCellCountInt](), rows: 44},
		{top: 53, bot: None[SizeCellCountInt](), rows: 24}, // 1 row, padded to 24
		{top: 30, bot: Some[SizeCellCountInt](53), rows: 24},
		{top: 31, bot: Some[SizeCellCountInt](53), rows: 24}, // 23 rows, padded
		{top: 10, bot: Some[SizeCellCountInt](35), rows: 26},
		{top: 7, bot: Some[SizeCellCountInt](7), rows: 24}, // 1 row, padded
		{top: 20, bot: Some[SizeCellCountInt](43), rows: 24},
		{top: 20, bot: Some[SizeCellCountInt](42), rows: 24}, // 23 rows, padded
	} {
		bot := None[Point]()
		if case_.bot.Valid {
			bot = Some(pl3Screen(0, int(case_.bot.Value)))
		}
		c := pl3Clone(t, s, PageListClone{
			Top: pl3Screen(0, int(case_.top)),
			Bot: bot,
		})
		defer c.Deinit()
		expectEqual(t, case_.rows, c.total_rows)
		expectEqual(t, case_.rows, c.totalRows())
		pl3Verify(t, c)

		// Cloned rows have the source's contents, padding rows are blank.
		bot_row := uint(53)
		if case_.bot.Valid {
			bot_row = uint(case_.bot.Value)
		}
		copied := bot_row - uint(case_.top) + 1
		for y := uint(0); y < c.total_rows; y++ {
			got := pl3Cp(t, c, pl3Screen(0, int(y)))
			if y < copied {
				expectEqual(t, uint32(0x100+uint(case_.top)+y), got)
			} else {
				expectEqual(t, uint32(0), got)
			}
		}
	}
}

// Zig: test "term51: clone: x coordinates of the bounds are ignored"
func TestPl3_term51_clone_x_coordinates_of_the_bounds_are_ignored(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 30)
	pl3MarkRows(t, s)

	a := pl3Clone(t, s, PageListClone{
		Top: pl3Screen(0, 10),
		Bot: Some(pl3Screen(0, 40)),
	})
	defer a.Deinit()
	b := pl3Clone(t, s, PageListClone{
		Top: pl3Screen(79, 10),
		Bot: Some(pl3Screen(5, 40)),
	})
	defer b.Deinit()

	ca := pl3Column(t, a)
	cb := pl3Column(t, b)
	expectEqual(t, uint(31), a.total_rows)
	expectEqualSlices(t, ca, cb)
}

// Zig: test "term51: clone: top point tags"
func TestPl3_term51_clone_top_point_tags(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 30) // 54 rows, active area is screen rows 30..53
	pl3MarkRows(t, s)

	// active: y offsets into the active area, to the end of the screen.
	{
		c := pl3Clone(t, s, PageListClone{Top: pl3Active(0, 0)})
		defer c.Deinit()
		expectEqual(t, uint(24), c.total_rows)
		expectEqual(t, uint32(0x100+30), pl3Cp(t, c, pl3Screen(0, 0)))
	}
	{
		c := pl3Clone(t, s, PageListClone{Top: pl3Active(0, 5)})
		defer c.Deinit()
		// 19 rows are copied, padded to the 24 row active area.
		expectEqual(t, uint(24), c.total_rows)
		expectEqual(t, uint32(0x100+35), pl3Cp(t, c, pl3Screen(0, 0)))
		expectEqual(t, uint32(0x100+53), pl3Cp(t, c, pl3Screen(0, 18)))
		expectEqual(t, uint32(0), pl3Cp(t, c, pl3Screen(0, 19)))
		expectEqual(t, uint32(0), pl3Cp(t, c, pl3Screen(0, 23)))
	}
	{
		c := pl3Clone(t, s, PageListClone{Top: pl3Active(0, 23)})
		defer c.Deinit()
		expectEqual(t, uint(24), c.total_rows)
		expectEqual(t, uint32(0x100+53), pl3Cp(t, c, pl3Screen(0, 0)))
		expectEqual(t, uint32(0), pl3Cp(t, c, pl3Screen(0, 1)))
	}

	// active with an explicit bottom.
	{
		c := pl3Clone(t, s, PageListClone{
			Top: pl3Active(0, 2),
			Bot: Some(pl3Active(0, 22)),
		})
		defer c.Deinit()
		expectEqual(t, uint(24), c.total_rows)
		expectEqual(t, uint32(0x100+32), pl3Cp(t, c, pl3Screen(0, 0)))
		expectEqual(t, uint32(0x100+52), pl3Cp(t, c, pl3Screen(0, 20)))
		expectEqual(t, uint32(0), pl3Cp(t, c, pl3Screen(0, 21)))
	}

	// history: everything above the active area, 30 rows.
	{
		c := pl3Clone(t, s, PageListClone{Top: pl3History(0, 0)})
		defer c.Deinit()
		expectEqual(t, uint(30), c.total_rows)
		expectEqual(t, uint32(0x100), pl3Cp(t, c, pl3Screen(0, 0)))
		expectEqual(t, uint32(0x100+29), pl3Cp(t, c, pl3Screen(0, 29)))
	}
	// history with a bottom smaller than the active area gets padded.
	{
		c := pl3Clone(t, s, PageListClone{
			Top: pl3History(0, 1),
			Bot: Some(pl3History(0, 4)),
		})
		defer c.Deinit()
		expectEqual(t, uint(24), c.total_rows)
		expectEqual(t, uint32(0x100+1), pl3Cp(t, c, pl3Screen(0, 0)))
		expectEqual(t, uint32(0x100+4), pl3Cp(t, c, pl3Screen(0, 3)))
		expectEqual(t, uint32(0), pl3Cp(t, c, pl3Screen(0, 4)))
	}
}

// Zig: test "term51: clone: viewport tags follow the viewport"
func TestPl3_term51_clone_viewport_tags_follow_the_viewport(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 30)
	pl3MarkRows(t, s)

	// Active viewport: the active area (to the bottom of the viewport).
	{
		c := pl3Clone(t, s, PageListClone{Top: pl3Viewport(0, 0)})
		defer c.Deinit()
		expectEqual(t, uint(24), c.total_rows)
		expectEqual(t, uint32(0x100+30), pl3Cp(t, c, pl3Screen(0, 0)))
	}

	// Top viewport.
	s.viewport = pl3ViewportTop
	{
		c := pl3Clone(t, s, PageListClone{
			Top: pl3Viewport(0, 0),
			Bot: Some(pl3Viewport(0, 23)),
		})
		defer c.Deinit()
		expectEqual(t, uint(24), c.total_rows)
		expectEqual(t, uint32(0x100), pl3Cp(t, c, pl3Screen(0, 0)))
		expectEqual(t, uint32(0x100+23), pl3Cp(t, c, pl3Screen(0, 23)))
	}

	// Top viewport without an explicit bottom: the viewport's 24 rows.
	{
		c := pl3Clone(t, s, PageListClone{Top: pl3Viewport(0, 0)})
		defer c.Deinit()
		expectEqual(t, uint(24), c.total_rows)
		expectEqual(t, uint32(0x100+23), pl3Cp(t, c, pl3Screen(0, 23)))
	}

	// Pin viewport.
	s.viewport = pl3ViewportPin
	*s.viewport_pin = pl3Pin(t, s, pl3Screen(0, 7))
	s.viewport_pin_row_offset = Some[uint](7)
	{
		// The default bottom is the bottom of the viewport (rows 7..30).
		c := pl3Clone(t, s, PageListClone{Top: pl3Viewport(0, 0)})
		defer c.Deinit()
		expectEqual(t, uint(24), c.total_rows)
		expectEqual(t, uint32(0x100+7), pl3Cp(t, c, pl3Screen(0, 0)))
		expectEqual(t, uint32(0x100+30), pl3Cp(t, c, pl3Screen(0, 23)))
	}
	{
		// 23 rows (8..30) padded to 24.
		c := pl3Clone(t, s, PageListClone{
			Top: pl3Viewport(0, 1),
			Bot: Some(pl3Viewport(0, 23)),
		})
		defer c.Deinit()
		expectEqual(t, uint(24), c.total_rows)
		expectEqual(t, uint32(0x100+8), pl3Cp(t, c, pl3Screen(0, 0)))
		expectEqual(t, uint32(0x100+30), pl3Cp(t, c, pl3Screen(0, 22)))
		expectEqual(t, uint32(0), pl3Cp(t, c, pl3Screen(0, 23)))
	}
	s.viewport = pl3ViewportActive
}

// Zig: test "term51: clone: multiple pages with trimmed ends"
func TestPl3_term51_clone_multiple_pages_with_trimmed_ends(t *testing.T) {
	testStdCapacity(t)
	// cols=500: 93 rows per page. 50 rows initially, growing to 200 rows.
	s := pl3Init(t, 500, 50, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 150)
	pl3ExpectPages(t, s, []uint{93, 93, 14})
	pl3MarkRows(t, s)

	// Full clone keeps the page structure and serials 0..2.
	{
		c := pl3Clone(t, s, PageListClone{Top: pl3Screen(0, 0)})
		defer c.Deinit()
		pl3ExpectPages(t, c, []uint{93, 93, 14})
		expectEqual(t, uint64(3), c.page_serial)
		var serial uint64 = 0
		for node := c.pages.first; node != nil; node = node.next {
			expectEqual(t, serial, node.serial)
			expectEqual(t, SizeCellCountInt(500), node.data.size.cols)
			expectEqual(t, SizeCellCountInt(93), node.data.capacity.rows)
			serial += 1
		}
		expectEqual(t, 3*pageListStdSize(), c.page_size)
	}

	// Trimmed on both ends across a page boundary: rows 60..130.
	{
		c := pl3Clone(t, s, PageListClone{
			Top: pl3Screen(0, 60),
			Bot: Some(pl3Screen(0, 130)),
		})
		defer c.Deinit()
		pl3ExpectPages(t, c, []uint{33, 38})
		expectEqual(t, uint(71), c.total_rows)
		expectEqual(t, uint64(2), c.page_serial)
		expectEqual(t, 2*pageListStdSize(), c.page_size)
		col := pl3Column(t, c)
		for y, cp := range col {
			expectEqual(t, uint32(0x100+60+y), cp)
		}
	}

	// Exactly one full page in the middle.
	{
		c := pl3Clone(t, s, PageListClone{
			Top: pl3Screen(0, 93),
			Bot: Some(pl3Screen(0, 185)),
		})
		defer c.Deinit()
		pl3ExpectPages(t, c, []uint{93})
		expect(t, c.pages.first.data.size.rows == 93, "c.pages.first.?.data.size.rows == 93")
	}

	// Last row of a page and first row of the next.
	{
		c := pl3Clone(t, s, PageListClone{
			Top: pl3Screen(0, 92),
			Bot: Some(pl3Screen(0, 93)),
		})
		defer c.Deinit()
		// 2 rows, padded to 50. The two rows come from 2 pages, so
		// chunks of 1 + 1 rows then padded by growing the last.
		expectEqual(t, uint(50), c.total_rows)
		expectEqual(t, uint32(0x100+92), pl3Cp(t, c, pl3Screen(0, 0)))
		expectEqual(t, uint32(0x100+93), pl3Cp(t, c, pl3Screen(0, 1)))
		expectEqual(t, uint32(0), pl3Cp(t, c, pl3Screen(0, 2)))
		pl3ExpectPages(t, c, []uint{1, 49})
	}

	// Source unchanged.
	pl3ExpectPages(t, s, []uint{93, 93, 14})
	pl3Verify(t, s)
}

// Zig: test "term51: clone: padding below the active area size"
func TestPl3_term51_clone_padding_below_the_active_area_size(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 30)
	pl3MarkRows(t, s)

	// 23 rows copied + 1 padding row; 24 rows copied + 0.
	{
		c := pl3Clone(t, s, PageListClone{
			Top: pl3Screen(0, 0),
			Bot: Some(pl3Screen(0, 22)),
		})
		defer c.Deinit()
		expectEqual(t, uint(24), c.total_rows)
		pl3ExpectPages(t, c, []uint{24})
		expectEqual(t, uint32(0x100+22), pl3Cp(t, c, pl3Screen(0, 22)))
		expectEqual(t, uint32(0), pl3Cp(t, c, pl3Screen(0, 23)))
	}
	{
		c := pl3Clone(t, s, PageListClone{
			Top: pl3Screen(0, 0),
			Bot: Some(pl3Screen(0, 23)),
		})
		defer c.Deinit()
		pl3ExpectPages(t, c, []uint{24})
		expectEqual(t, uint32(0x100+23), pl3Cp(t, c, pl3Screen(0, 23)))
	}

	// A single row is padded with 23 blank rows.
	{
		c := pl3Clone(t, s, PageListClone{
			Top: pl3Screen(0, 40),
			Bot: Some(pl3Screen(0, 40)),
		})
		defer c.Deinit()
		pl3ExpectPages(t, c, []uint{24})
		expectEqual(t, uint(24), c.total_rows)
		expectEqual(t, uint32(0x100+40), pl3Cp(t, c, pl3Screen(0, 0)))
		for y := 1; y < 24; y++ {
			expectEqual(t, uint32(0), pl3Cp(t, c, pl3Screen(0, y)))
		}
		// The active area of the clone is the whole clone.
		expectEqual(t, PageListPin{Node: c.pages.first, Y: 0, X: 0}, c.GetTopLeft(PointTagActive))
		// Serial reflects only the one cloned page.
		expectEqual(t, uint64(1), c.page_serial)
	}
}

// Zig: test "term51: clone: padded rows are blank"
func TestPl3_term51_clone_padded_rows_are_blank(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Fill the rows beyond the first 3 in the source page with data and
	// styles, then clone only 3 rows.
	page := &s.pages.first.data
	style_id, err := page.styles.Add(pl3Bold())
	zigTry(t, err)
	for y := 3; y < 24; y++ {
		rac := page.GetRowAndCell(5, uint(y))
		rac.row.styled = true
		rac.row.wrap = true
		*rac.cell = PageCell{
			content_tag: PageCellContentTagCodepoint,
			content:     PageCellContent{codepoint: 'W'},
			style_id:    style_id,
		}
		page.styles.Use(style_id)
	}
	page.styles.Release(style_id)

	c := pl3Clone(t, s, PageListClone{
		Top: pl3Screen(0, 0),
		Bot: Some(pl3Screen(0, 2)),
	})
	defer c.Deinit()
	expectEqual(t, uint(24), c.total_rows)
	for y := 3; y < 24; y++ {
		rac := c.pages.first.data.GetRowAndCell(5, uint(y))
		expectEqual(t, uint32(0), rac.cell.content.codepoint)
		expectEqual(t, uint16(0), rac.cell.style_id)
		expect(t, !rac.row.styled, "!rac.row.styled")
		expect(t, !rac.row.wrap, "!rac.row.wrap")
	}
	expectEqual(t, uint(0), c.pages.first.data.styles.Count())
}

// Zig: test "term51: clone: padding allocates a page when the last page is full"
func TestPl3_term51_clone_padding_allocates_a_page_when_the_last_page_is_full(t *testing.T) {
	testStdCapacity(t)
	// cols=500 has 93 rows per page; the active area is 100 rows.
	s := pl3Init(t, 500, 100, None[uint64]())
	defer s.Deinit()
	pl3ExpectPages(t, s, []uint{93, 7})
	pl3MarkRows(t, s)

	// A full page of 93 rows: 7 rows of padding require a new page.
	{
		c := pl3Clone(t, s, PageListClone{
			Top: pl3Screen(0, 0),
			Bot: Some(pl3Screen(0, 92)),
		})
		defer c.Deinit()
		pl3ExpectPages(t, c, []uint{93, 7})
		expectEqual(t, uint(100), c.total_rows)
		// The cloned page took serial 0, the grown page serial 1.
		expectEqual(t, uint64(0), c.pages.first.serial)
		expectEqual(t, uint64(1), c.pages.last.serial)
		expectEqual(t, uint64(2), c.page_serial)
		expectEqual(t, 2*pageListStdSize(), c.page_size)
		expectEqual(t, uint32(0x100+92), pl3Cp(t, c, pl3Screen(0, 92)))
		expectEqual(t, uint32(0), pl3Cp(t, c, pl3Screen(0, 93)))
	}

	// A partial last page absorbs the padding and then spills.
	{
		c := pl3Clone(t, s, PageListClone{
			Top: pl3Screen(0, 90),
			Bot: Some(pl3Screen(0, 95)),
		})
		defer c.Deinit()
		pl3ExpectPages(t, c, []uint{3, 93, 4})
		expectEqual(t, uint(100), c.total_rows)
		expectEqual(t, uint64(3), c.page_serial)
		expectEqual(t, uint32(0x100+90), pl3Cp(t, c, pl3Screen(0, 0)))
		expectEqual(t, uint32(0x100+95), pl3Cp(t, c, pl3Screen(0, 5)))
		expectEqual(t, uint32(0), pl3Cp(t, c, pl3Screen(0, 6)))
	}

	// Only the last page (7 rows) padded to 100 rows.
	{
		c := pl3Clone(t, s, PageListClone{Top: pl3Screen(0, 93)})
		defer c.Deinit()
		pl3ExpectPages(t, c, []uint{93, 7})
		expectEqual(t, uint32(0x100+93), pl3Cp(t, c, pl3Screen(0, 0)))
		expectEqual(t, uint32(0x100+99), pl3Cp(t, c, pl3Screen(0, 6)))
		expectEqual(t, uint32(0), pl3Cp(t, c, pl3Screen(0, 7)))
	}
}

// Zig: test "term51: clone: one row per page pads with new pages"
func TestPl3_term51_clone_one_row_per_page_pads_with_new_pages(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 46950, 24, None[uint64]())
	defer s.Deinit()
	expectEqual(t, uint(24), s.TotalPages())
	pl3MarkRows(t, s)

	c := pl3Clone(t, s, PageListClone{
		Top: pl3Screen(0, 5),
		Bot: Some(pl3Screen(0, 10)),
	})
	defer c.Deinit()
	expectEqual(t, uint(24), c.TotalPages())
	expectEqual(t, uint(24), c.total_rows)
	expectEqual(t, uint64(24), c.page_serial)
	expectEqual(t, 24*pageListStdSize(), c.page_size)
	for y := 0; y < 6; y++ {
		expectEqual(t, uint32(0x100+5+y), pl3Cp(t, c, pl3Screen(0, y)))
	}
	for y := 6; y < 24; y++ {
		expectEqual(t, uint32(0), pl3Cp(t, c, pl3Screen(0, y)))
	}
}

// Zig: test "term51: clone: smallest list"
func TestPl3_term51_clone_smallest_list(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 1, 1, None[uint64]())
	defer s.Deinit()
	pl3SetCell(pl3Pin(t, s, pl3Active(0, 0)), 'k')

	c := pl3Clone(t, s, PageListClone{Top: pl3Active(0, 0), Bot: Some(pl3Active(0, 0))})
	defer c.Deinit()
	pl3ExpectPages(t, c, []uint{1})
	expectEqual(t, uint32('k'), pl3Cp(t, c, pl3Screen(0, 0)))
	expectEqual(t, SizeCellCountInt(1), c.cols)
	expectEqual(t, SizeCellCountInt(1), c.rows)
}

// Zig: test "term51: clone: many rows in a single page"
func TestPl3_term51_clone_many_rows_in_a_single_page(t *testing.T) {
	testStdCapacity(t)
	// cols=1 holds 23475 rows per page, so 2000 rows are one page.
	s := pl3Init(t, 1, 2000, None[uint64]())
	defer s.Deinit()
	pl3ExpectPages(t, s, []uint{2000})
	pl3MarkRows(t, s)

	// Full clone.
	{
		c := pl3Clone(t, s, PageListClone{Top: pl3Screen(0, 0)})
		defer c.Deinit()
		pl3ExpectPages(t, c, []uint{2000})
		expectEqual(t, uint32(0x100+1999), pl3Cp(t, c, pl3Screen(0, 1999)))
		expectEqual(t, SizeCellCountInt(23475), c.pages.first.data.capacity.rows)
	}

	// Half is cloned and the rest padded back to the active height.
	{
		c := pl3Clone(t, s, PageListClone{Top: pl3Screen(0, 1000)})
		defer c.Deinit()
		pl3ExpectPages(t, c, []uint{2000})
		expectEqual(t, uint32(0x100+1000), pl3Cp(t, c, pl3Screen(0, 0)))
		expectEqual(t, uint32(0x100+1999), pl3Cp(t, c, pl3Screen(0, 999)))
		expectEqual(t, uint32(0), pl3Cp(t, c, pl3Screen(0, 1000)))
		expectEqual(t, uint32(0), pl3Cp(t, c, pl3Screen(0, 1999)))
	}
}

// Zig: test "term51: clone: page level dirty flag is copied"
func TestPl3_term51_clone_page_level_dirty_flag_is_copied(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 500, 100, None[uint64]())
	defer s.Deinit()
	pl3ExpectPages(t, s, []uint{93, 7})
	s.ClearDirty()
	s.pages.first.data.dirty = true

	c := pl3Clone(t, s, PageListClone{Top: pl3Screen(0, 0)})
	defer c.Deinit()
	expect(t, c.pages.first.data.dirty, "c.pages.first.?.data.dirty")
	expect(t, !c.pages.last.data.dirty, "!c.pages.last.?.data.dirty")
	expect(t, c.IsDirty(pl3Screen(0, 50)), "c.isDirty(.{ .screen = .{ .y = 50 } })")
	expect(t, !c.IsDirty(pl3Screen(0, 95)), "!c.isDirty(.{ .screen = .{ .y = 95 } })")
	// Dirtiness is a property of the source only, not shared.
	c.ClearDirty()
	expect(t, s.pages.first.data.dirty, "s.pages.first.?.data.dirty")

	// Row-level dirty flags copied individually.
	s.ClearDirty()
	s.markDirty(pl3Screen(0, 97))
	c2 := pl3Clone(t, s, PageListClone{Top: pl3Screen(0, 0)})
	defer c2.Deinit()
	expect(t, !c2.IsDirty(pl3Screen(0, 96)), "!c2.isDirty(.{ .screen = .{ .y = 96 } })")
	expect(t, c2.IsDirty(pl3Screen(0, 97)), "c2.isDirty(.{ .screen = .{ .y = 97 } })")
	expect(t, !c2.IsDirty(pl3Screen(0, 98)), "!c2.isDirty(.{ .screen = .{ .y = 98 } })")
}

// Zig: test "term51: clone: page size columns follow the source page"
func TestPl3_term51_clone_page_size_columns_follow_the_source_page(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// A page whose size.cols is below the list's cols (as left by a lazy
	// column resize).
	s.pages.first.data.size.cols = 40
	defer func() { s.pages.first.data.size.cols = 80 }()

	c := pl3Clone(t, s, PageListClone{Top: pl3Screen(0, 0)})
	defer c.Deinit()
	expectEqual(t, SizeCellCountInt(80), c.cols)
	expectEqual(t, SizeCellCountInt(40), c.pages.first.data.size.cols)
	expectEqual(t, SizeCellCountInt(80), c.pages.first.data.capacity.cols)
	pl3Verify(t, c)
}

// Zig: test "term51: clone: non-standard pages"
func TestPl3_term51_clone_non_standard_pages(t *testing.T) {
	testStdCapacity(t)
	cols := zigUnwrap(t, PageStdCapacity.MaxCols()) + 1
	s := pl3Init(t, cols, 3, None[uint64]())
	defer s.Deinit()
	pl3SetCell(pl3Pin(t, s, pl3Screen(46950, 2)), 'w')
	pl3SetCell(pl3Pin(t, s, pl3Screen(0, 1)), 'v')

	// Full clone.
	{
		c := pl3Clone(t, s, PageListClone{Top: pl3Screen(0, 0)})
		defer c.Deinit()
		pl3ExpectPages(t, c, []uint{3})
		expectEqual(t, uint64(80838656), c.page_size)
		expectEqual(t, uint64(80838656), pageListMemoryLen(&c.pages.first.data))
		expectEqual(t, cols, c.cols)
		expectEqual(t, s.pages.first.data.capacity, c.pages.first.data.capacity)
		expectEqual(t, uint32('w'), pl3Cp(t, c, pl3Screen(46950, 2)))
	}

	// Partial clone padded within the capacity of the non-standard page.
	{
		c := pl3Clone(t, s, PageListClone{
			Top: pl3Screen(0, 1),
			Bot: Some(pl3Screen(0, 1)),
		})
		defer c.Deinit()
		pl3ExpectPages(t, c, []uint{3})
		expectEqual(t, uint32('v'), pl3Cp(t, c, pl3Screen(0, 0)))
		expectEqual(t, uint32(0), pl3Cp(t, c, pl3Screen(46950, 2)))
		expectEqual(t, uint64(80838656), c.page_size)
	}
}

// Zig: test "term51: clone: tracked pins are only kept when a remap is given"
func TestPl3_term51_clone_tracked_pins_are_only_kept_when_a_remap_is_given(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	p := pl3Track(t, s, pl3Pin(t, s, pl3Active(4, 6)))
	defer s.UntrackPin(p)

	// The default options have no remap and no bottom.
	opts := PageListClone{Top: pl3Screen(0, 0), Bot: None[Point](), TrackedPins: nil}
	expect(t, !opts.Bot.Valid, "opts.bot == null")
	expect(t, opts.TrackedPins == nil, "opts.tracked_pins == null")

	c := pl3Clone(t, s, opts)
	defer c.Deinit()
	expectEqual(t, uint(1), c.CountTrackedPins())
	expectEqual(t, uint(2), s.CountTrackedPins())
}

// Zig: test "term51: clone: remaps tracked pins into the cloned area"
func TestPl3_term51_clone_remaps_tracked_pins_into_the_cloned_area(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 30)

	a := pl3Track(t, s, pl3Pin(t, s, pl3Screen(4, 10)))
	b := pl3Track(t, s, pl3Pin(t, s, pl3Screen(79, 20)))
	dup := pl3Track(t, s, pl3Pin(t, s, pl3Screen(4, 10)))
	garbage := pl3Track(t, s, pl3Pin(t, s, pl3Screen(1, 12)))
	garbage.Garbage = true
	defer s.UntrackPin(a)
	defer s.UntrackPin(b)
	defer s.UntrackPin(dup)
	defer s.UntrackPin(garbage)

	remap := PageListCloneTrackedPinsRemap{}
	c := pl3Clone(t, s, PageListClone{
		Top:         pl3Screen(0, 10),
		Bot:         None[Point](),
		TrackedPins: &remap,
	})
	defer c.Deinit()

	// Four pins remapped plus nothing else (the source viewport pin is at
	// screen y=0, above the cloned area).
	expectEqual(t, 4, len(remap))
	expectEqual(t, uint(5), c.CountTrackedPins())
	expect(t, remap[s.viewport_pin] == nil, "remap.get(s.viewport_pin) == null")

	na := remap[a]
	nb := remap[b]
	ndup := remap[dup]
	ng := remap[garbage]
	expect(t, na != nil, "remap.get(a) != null")
	expect(t, nb != nil, "remap.get(b) != null")
	expect(t, ndup != nil, "remap.get(dup) != null")
	expect(t, ng != nil, "remap.get(garbage) != null")

	// Each source pin gets its own new pin, owned by the clone.
	expect(t, na != a, "na != a")
	expect(t, na != ndup, "na != ndup")
	expect(t, na != c.viewport_pin, "na != c.viewport_pin")
	for _, np := range []*PageListPin{na, nb, ndup, ng} {
		expect(t, np.Node == c.pages.first, "np.node == c.pages.first.?")
		found := false
		for _, tp := range c.TrackedPins() {
			if tp == np {
				found = true
			}
		}
		expect(t, found, "found")
		expect(t, c.PinIsValid(*np), "c.pinIsValid(np.*)")
	}

	// y is offset by the chunk start, x and garbage are copied.
	expectEqual(t, SizeCellCountInt(0), na.Y)
	expectEqual(t, SizeCellCountInt(4), na.X)
	expectEqual(t, SizeCellCountInt(0), ndup.Y)
	expectEqual(t, SizeCellCountInt(10), nb.Y)
	expectEqual(t, SizeCellCountInt(79), nb.X)
	expectEqual(t, SizeCellCountInt(2), ng.Y)
	expect(t, !na.Garbage, "!na.garbage")
	expect(t, ng.Garbage, "ng.garbage")
	expectEqualPoint(t, pl3Screen(79, 10), zigUnwrap(t, c.PointFromPin(PointTagScreen, *nb)))

	// The clone's own viewport pin stays at the top-left.
	expect(t, c.viewport_pin.Node == c.pages.first, "c.viewport_pin.node == c.pages.first.?")
	expectEqual(t, SizeCellCountInt(0), c.viewport_pin.Y)
	expect(t, !c.viewport_pin.Garbage, "!c.viewport_pin.garbage")

	// The source's pins are unchanged.
	expectEqual(t, SizeCellCountInt(10), a.Y)
	expectEqual(t, SizeCellCountInt(20), b.Y)
	expect(t, garbage.Garbage, "garbage.garbage")
}

// Zig: test "term51: clone: remap includes the source viewport pin when in range"
func TestPl3_term51_clone_remap_includes_the_source_viewport_pin_when_in_range(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 30)

	// viewport pin at screen row 7, within the cloned rows 7..
	s.viewport = pl3ViewportPin
	*s.viewport_pin = pl3Pin(t, s, pl3Screen(3, 7))
	s.viewport_pin_row_offset = Some[uint](7)
	defer func() { s.viewport = pl3ViewportActive }()

	remap := PageListCloneTrackedPinsRemap{}
	c := pl3Clone(t, s, PageListClone{
		Top:         pl3Screen(0, 7),
		Bot:         None[Point](),
		TrackedPins: &remap,
	})
	defer c.Deinit()

	expectEqual(t, 1, len(remap))
	np := remap[s.viewport_pin]
	expect(t, np != nil, "remap.get(s.viewport_pin) != null")
	expect(t, np != c.viewport_pin, "np != c.viewport_pin")
	expectEqual(t, SizeCellCountInt(0), np.Y)
	expectEqual(t, SizeCellCountInt(3), np.X)
	// The clone has the new pin plus its own viewport pin.
	expectEqual(t, uint(2), c.CountTrackedPins())
	expect(t, c.viewport.Tag == PageListViewportActive, "c.viewport == .active")
}

// Zig: test "term51: clone: remap boundaries and multiple pages"
func TestPl3_term51_clone_remap_boundaries_and_multiple_pages(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 500, 50, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 150)
	pl3ExpectPages(t, s, []uint{93, 93, 14})

	// Cloned rows are screen rows 60..130 inclusive.
	ys := []int{59, 60, 92, 93, 130, 131}
	pins := make([]*PageListPin, len(ys))
	for i, y := range ys {
		pins[i] = pl3Track(t, s, pl3Pin(t, s, pl3Screen(9, y)))
	}
	defer func() {
		for _, p := range pins {
			s.UntrackPin(p)
		}
	}()

	remap := PageListCloneTrackedPinsRemap{}
	c := pl3Clone(t, s, PageListClone{
		Top:         pl3Screen(100, 60),
		Bot:         Some(pl3Screen(100, 130)),
		TrackedPins: &remap,
	})
	defer c.Deinit()
	pl3ExpectPages(t, c, []uint{33, 38})

	// Excluded: just above and just below the range.
	expect(t, remap[pins[0]] == nil, "remap.get(pins[0]) == null")
	expect(t, remap[pins[5]] == nil, "remap.get(pins[5]) == null")
	expectEqual(t, 4, len(remap))
	expectEqual(t, uint(5), c.CountTrackedPins())

	first := c.pages.first
	second := c.pages.last
	type Expect struct {
		idx  int
		node *PageListNode
		y    SizeCellCountInt
	}
	for _, e := range []Expect{
		{idx: 1, node: first, y: 0},   // first row of first chunk
		{idx: 2, node: first, y: 32},  // last row of first chunk
		{idx: 3, node: second, y: 0},  // first row of second chunk
		{idx: 4, node: second, y: 37}, // last row of second chunk
	} {
		np := remap[pins[e.idx]]
		expect(t, np != nil, "remap.get(pins[e.idx]) != null")
		expect(t, np.Node == e.node, "np.node == e.node")
		expectEqual(t, e.y, np.Y)
		expectEqual(t, SizeCellCountInt(9), np.X)
		expect(t, c.PinIsValid(*np), "c.pinIsValid(np.*)")
	}
	pl3Verify(t, c)
}

// Zig: test "term51: clone: remap with padded clone keeps pins valid"
func TestPl3_term51_clone_remap_with_padded_clone_keeps_pins_valid(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 30)

	p := pl3Track(t, s, pl3Pin(t, s, pl3Screen(0, 50)))
	defer s.UntrackPin(p)

	remap := PageListCloneTrackedPinsRemap{}
	c := pl3Clone(t, s, PageListClone{
		Top:         pl3Screen(0, 49),
		Bot:         Some(pl3Screen(0, 52)),
		TrackedPins: &remap,
	})
	defer c.Deinit()
	expectEqual(t, uint(24), c.total_rows)
	np := remap[p]
	expect(t, np != nil, "remap.get(p) != null")
	expectEqual(t, SizeCellCountInt(1), np.Y)
	expect(t, c.PinIsValid(*np), "c.pinIsValid(np.*)")
	pl3Verify(t, c)
}

// Zig: test "term51: clone: remap with no pins in range is empty"
func TestPl3_term51_clone_remap_with_no_pins_in_range_is_empty(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 30)

	remap := PageListCloneTrackedPinsRemap{}
	c := pl3Clone(t, s, PageListClone{
		Top:         pl3Screen(0, 20),
		Bot:         None[Point](),
		TrackedPins: &remap,
	})
	defer c.Deinit()
	// The viewport pin (screen y=0) is outside; only the clone's own pin.
	expectEqual(t, 0, len(remap))
	expectEqual(t, uint(1), c.CountTrackedPins())
}

// Zig: test "term51: clone: clone of a clone"
func TestPl3_term51_clone_clone_of_a_clone(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 500, 50, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 150)
	pl3MarkRows(t, s)

	c1 := pl3Clone(t, s, PageListClone{
		Top: pl3Screen(0, 40),
		Bot: Some(pl3Screen(0, 150)),
	})
	defer c1.Deinit()
	c2 := pl3Clone(t, c1, PageListClone{
		Top: pl3Screen(0, 10),
		Bot: Some(pl3Screen(0, 100)),
	})
	defer c2.Deinit()

	expectEqual(t, uint(91), c2.total_rows)
	expectEqual(t, uint32(0x100+50), pl3Cp(t, c2, pl3Screen(0, 0)))
	expectEqual(t, uint32(0x100+140), pl3Cp(t, c2, pl3Screen(0, 90)))
	expectEqual(t, s.min_max_size, c2.min_max_size)
	pl3Verify(t, c2)
}

// The Zig test "term51: clone: allocator failure sweeps" is not translated
// (FailingAllocator fault injection); see scripts/not-translated.txt.

// ---------------------------------------------------------------------------
// term51: additional tests (K): helpers.
// ---------------------------------------------------------------------------

// pl3ExpectError is Zig's `try testing.expectError(expected, actual)`.
func pl3ExpectError(t *testing.T, want error, got error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

// pl3Cap is the Zig literal `.{ .cols = cols, .rows = rows }`: every other
// field of Capacity at its declared default.
func pl3Cap(cols SizeCellCountInt, rows SizeCellCountInt) PageCapacity {
	return PageCapacityInit(cols, rows)
}

// pl3Dim is `@as(?size.CellCountInt, n)`, a Resize.cols / Resize.rows value.
func pl3Dim(n SizeCellCountInt) Optional[SizeCellCountInt] {
	return Some(n)
}

// pl3NoDim is a null Resize.cols / Resize.rows.
func pl3NoDim() Optional[SizeCellCountInt] {
	return None[SizeCellCountInt]()
}

// pl3NoCursor is a null Resize.Cursor.
func pl3NoCursor() Optional[PageListResizeCursor] {
	return None[PageListResizeCursor]()
}

// pl3CursorAt is the Zig literal `.{ .x = x, .y = y }` of Resize.Cursor.
func pl3CursorAt(x SizeCellCountInt, y SizeCellCountInt) Optional[PageListResizeCursor] {
	return Some(PageListResizeCursor{x: x, y: y})
}

// pl3Resize is `try s.resize(opts)`.
func pl3Resize(t *testing.T, s *PageList, opts PageListResize) {
	t.Helper()
	zigTry(t, s.Resize(opts))
}

// pl3ResizeCols is `try s.resizeCols(cols, cursor)`.
func pl3ResizeCols(t *testing.T, s *PageList, cols SizeCellCountInt, cursor Optional[PageListResizeCursor]) {
	t.Helper()
	zigTry(t, s.resizeCols(cols, cursor))
}

// pl3KPin is term51KPin.
func pl3KPin(t *testing.T, s *PageList, x int, y int) PageListPin {
	t.Helper()
	return pl3Pin(t, s, pl3Screen(x, y))
}

// pl3KLoad is term51KLoad: fill the existing screen rows of `s` from a text
// picture. Each line is one row: a leading `<` (wrap continuation) or `|`,
// one character per cell (`.` empty, `W` wide, `_` spacer tail, `^` spacer
// head, anything else is that codepoint), then `>` (wrapped) or `|`, then an
// optional `P` (prompt) or `C` (prompt continuation). This is the inverse of
// pl3KDump.
func pl3KLoad(t *testing.T, s *PageList, text string) {
	t.Helper()
	for y, line := range strings.Split(text, "\n") {
		end := len(line)
		sp := PageRowSemanticPromptNone
		switch line[end-1] {
		case 'P':
			sp = PageRowSemanticPromptPrompt
			end -= 1
		case 'C':
			sp = PageRowSemanticPromptPromptContinuation
			end -= 1
		}
		wrap := line[end-1] == '>'
		cont := line[0] == '<'
		body := line[1 : end-1]
		if len(body) != int(s.cols) {
			t.Fatalf("TestUnexpectedResult: line %d has %d cells, want %d", y, len(body), s.cols)
		}

		p := pl3KPin(t, s, 0, y)
		row := p.RowAndCell().row
		row.wrap = wrap
		row.wrap_continuation = cont
		row.semantic_prompt = sp
		cells := p.Node.data.GetCells(row)
		for x := 0; x < len(body); x++ {
			ch := body[x]
			switch ch {
			case '.':
				cells[x] = PageCell{}
			case 'W':
				cells[x] = PageCell{
					content_tag: PageCellContentTagCodepoint,
					content:     PageCellContent{codepoint: 0x1F600},
					wide:        PageCellWideWide,
				}
			case '_':
				cells[x] = PageCell{
					content_tag: PageCellContentTagCodepoint,
					content:     PageCellContent{codepoint: 0},
					wide:        PageCellWideSpacerTail,
				}
			case '^':
				cells[x] = PageCell{
					content_tag: PageCellContentTagCodepoint,
					content:     PageCellContent{codepoint: 0},
					wide:        PageCellWideSpacerHead,
				}
			default:
				cells[x] = PageCell{
					content_tag: PageCellContentTagCodepoint,
					content:     PageCellContent{codepoint: uint32(ch)},
				}
			}
		}
	}
}

// pl3KCh is term51KCh.
func pl3KCh(c PageCell) uint8 {
	switch c.content_tag {
	case PageCellContentTagBgColorPalette, PageCellContentTagBgColorRgb:
		return '#'
	case PageCellContentTagCodepoint, PageCellContentTagCodepointGrapheme:
	}
	switch c.wide {
	case PageCellWideWide:
		return 'W'
	case PageCellWideSpacerTail:
		return '_'
	case PageCellWideSpacerHead:
		return '^'
	default:
		if c.content.codepoint == 0 {
			return '.'
		} else if c.content.codepoint < 128 {
			return uint8(c.content.codepoint)
		}
		return '?'
	}
}

// pl3KDump is term51KDump: render every screen row of `s` in the format
// pl3KLoad reads.
func pl3KDump(t *testing.T, s *PageList) string {
	t.Helper()
	var buf []uint8
	it := s.RowIterator(PageListDirectionRightDown, pl3Screen(0, 0), None[Point]())
	first := true
	for {
		next := it.Next()
		if !next.Valid {
			break
		}
		p := next.Value
		if !first {
			buf = append(buf, '\n')
		}
		first = false
		row := p.RowAndCell().row
		cells := p.Node.data.GetCells(row)
		if len(cells) != int(s.cols) {
			t.Fatalf("TestUnexpectedResult: row has %d cells, want %d", len(cells), s.cols)
		}
		if row.wrap_continuation {
			buf = append(buf, '<')
		} else {
			buf = append(buf, '|')
		}
		for _, c := range cells {
			buf = append(buf, pl3KCh(c))
		}
		if row.wrap {
			buf = append(buf, '>')
		} else {
			buf = append(buf, '|')
		}
		switch row.semantic_prompt {
		case PageRowSemanticPromptNone:
		case PageRowSemanticPromptPrompt:
			buf = append(buf, 'P')
		case PageRowSemanticPromptPromptContinuation:
			buf = append(buf, 'C')
		}
	}
	return string(buf)
}

// pl3KExpect is term51KExpect: assert the integrity of the list, the number
// of rows, and that the whole screen renders as `expected`.
func pl3KExpect(t *testing.T, s *PageList, expected string) {
	t.Helper()
	s.AssertIntegrity()
	got := pl3KDump(t, s)
	expectEqualStrings(t, expected, got)
	lines := uint(strings.Count(expected, "\n") + 1)
	expectEqual(t, lines, s.totalRows())
	expectEqual(t, lines, s.total_rows)
	for node := s.pages.first; node != nil; node = node.next {
		expectEqual(t, s.cols, node.data.size.cols)
		expect(t, node.data.size.rows > 0, "node.data.size.rows > 0")
	}
}

// pl3KExpectPin is term51KExpectPin: assert that a pin is at the given
// screen coordinate.
func pl3KExpectPin(t *testing.T, s *PageList, p PageListPin, x int, y int) {
	t.Helper()
	pt := zigUnwrap(t, s.PointFromPin(PointTagScreen, p))
	expectEqual(t, uint(x), uint(pt.Screen.X))
	expectEqual(t, uint(y), uint(pt.Screen.Y))
}

// pl3KCell is term51KCell.
func pl3KCell(t *testing.T, s *PageList, x int, y int) PageCell {
	t.Helper()
	return *pl3KPin(t, s, x, y).RowAndCell().cell
}

// pl3KRow is term51KRow.
func pl3KRow(t *testing.T, s *PageList, y int) *PageRow {
	t.Helper()
	return pl3KPin(t, s, 0, y).RowAndCell().row
}

// pl3KRowsPerPage is term51KRowsPerPage: the number of rows of every page.
func pl3KRowsPerPage(s *PageList) []uint {
	out := []uint{}
	for node := s.pages.first; node != nil; node = node.next {
		out = append(out, uint(node.data.size.rows))
	}
	return out
}

// pl3KAddNode is term51KAddNode: add a node (it IS in the list, after the
// last page) with `rows` used rows and keep the list's row cache consistent.
func pl3KAddNode(t *testing.T, s *PageList, cap PageCapacity, rows SizeCellCountInt) *PageListNode {
	t.Helper()
	node, err := s.createPage(cap)
	zigTry(t, err)
	node.data.size.rows = rows
	s.pages.Append(node)
	s.total_rows += uint(rows)
	return node
}

// pl3KReflow is term51KReflow: drives a ReflowCursor over every row of the
// list exactly as resizeCols does, but one row at a time, and without the
// post processing.
type pl3KReflow struct {
	list *PageList
	rc   pageListReflowCursor
	it   PageListRowIterator
}

// pl3KReflowInit is term51KReflow.init.
func pl3KReflowInit(t *testing.T, list *PageList, cols SizeCellCountInt, dst_cap Optional[PageCapacity]) *pl3KReflow {
	t.Helper()
	list.cols = cols
	var node *PageListNode
	{
		page := &list.pages.first.data
		var cap PageCapacity
		if dst_cap.Valid {
			cap = dst_cap.Value
		} else {
			adjusted, err := page.capacity.Adjust(PageCapacityAdjustment{cols: Some(cols)})
			if err != nil {
				c := page.capacity
				c.cols = cols
				c.rows = min(page.size.rows, c.rows)
				adjusted = c
			}
			cap = adjusted
		}
		n, err := list.createPage(cap)
		zigTry(t, err)
		n.data.size.rows = 1
		node = n
	}
	it := list.RowIterator(PageListDirectionRightDown, pl3Screen(0, 0), None[Point]())
	list.pages.first = node
	list.pages.last = node
	return &pl3KReflow{list: list, rc: pageListReflowCursorInit(node), it: it}
}

// step reflows the next source row, returns false when there are no more.
func (self *pl3KReflow) step(t *testing.T) bool {
	t.Helper()
	next := self.it.Next()
	if !next.Valid {
		return false
	}
	row := next.Value
	zigTry(t, self.rc.reflowRow(self.list, row))
	if row.Y == row.Node.data.size.rows-1 {
		self.list.destroyNode(row.Node)
	}
	return true
}

// run is term51KReflow.run.
func (self *pl3KReflow) run(t *testing.T) {
	t.Helper()
	for self.step(t) {
	}
}

// finish is term51KReflow.finish.
func (self *pl3KReflow) finish() {
	self.list.total_rows = self.rc.total_rows
}

// pl3KSetCell is term51KSetCell.
func pl3KSetCell(rc *pageListReflowCursor, x int, cell PageCell) {
	rc.page.GetCells(rc.page_row)[x] = cell
}

// pl3KWide is term51KWide.
func pl3KWide() PageCell {
	return PageCell{
		content_tag: PageCellContentTagCodepoint,
		content:     PageCellContent{codepoint: 0x1F600},
		wide:        PageCellWideWide,
	}
}

// pl3KDstCell is term51KDstCell.
func pl3KDstCell(rc *pageListReflowCursor, x int) PageCell {
	return rc.page.GetCells(rc.page_row)[x]
}

// pl3KSpacerTail, pl3KSpacerHead are the Zig literals
// `.{ .content_tag = .codepoint, .content = .{ .codepoint = 0 }, .wide = ... }`.
func pl3KSpacerTail() PageCell {
	return PageCell{
		content_tag: PageCellContentTagCodepoint,
		content:     PageCellContent{codepoint: 0},
		wide:        PageCellWideSpacerTail,
	}
}

func pl3KSpacerHead() PageCell {
	return PageCell{
		content_tag: PageCellContentTagCodepoint,
		content:     PageCellContent{codepoint: 0},
		wide:        PageCellWideSpacerHead,
	}
}

// pl3KStyleCell is the Zig literal `.{ .content_tag = .codepoint, .content =
// .{ .codepoint = cp }, .style_id = id }`.
func pl3KStyleCell(cp uint32, id StyleID) PageCell {
	c := pl3Codepoint(cp)
	c.style_id = id
	return c
}

// pl3KRowCell is `&page.rows[y].cells.ptr(page.memory)[x]`.
func pl3KRowCell(page *Page, y int, x int) *PageCell {
	return &page.GetCells(&page.rows[y])[x]
}

// pl3KFillStyles is term51KFillStyles: fill every cell of row 0 of the page
// with a distinct style until the style set is full. Returns how many
// styles were added.
func pl3KFillStyles(page *Page) uint {
	var n uint = 0
	for x := 0; x < int(page.size.cols); x++ {
		id, err := page.styles.Add(Style{BgColor: StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{
			R: uint8(x),
			G: uint8(x >> 8),
			B: 1,
		}}})
		if err != nil {
			break
		}
		rac := page.GetRowAndCell(uint(x), 0)
		rac.row.styled = true
		*rac.cell = PageCell{
			content_tag: PageCellContentTagCodepoint,
			content:     PageCellContent{codepoint: 'X'},
			style_id:    id,
		}
		n += 1
	}
	return n
}

// pl3KFillLinks is term51KFillLinks: put a distinct hyperlink on every cell
// of row 0 until the set is full.
func pl3KFillLinks(t *testing.T, page *Page) uint {
	t.Helper()
	var n uint = 0
	for x := 0; x < int(page.size.cols); x++ {
		id, err := page.InsertHyperlink(Hyperlink{
			ID:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: SizeOffsetInt(x)},
			URI: []uint8("u"),
		})
		if err != nil {
			break
		}
		rac := page.GetRowAndCell(uint(x), 0)
		*rac.cell = PageCellInit('L')
		zigTry(t, page.SetHyperlink(rac.row, rac.cell, id))
		n += 1
	}
	return n
}

// pl3Link is the Zig literal `.{ .id = .{ .implicit = n }, .uri = uri }`.
func pl3Link(n SizeOffsetInt, uri string) Hyperlink {
	return Hyperlink{
		ID:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: n},
		URI: []uint8(uri),
	}
}

// pl3KAA is `("|aa>\n<aa|\n" ** 5) ++ "|aa>\n<aa|"`.
var pl3KAA = strings.Repeat("|aa>\n<aa|\n", 5) + "|aa>\n<aa|"

// pl3Scroll is the Zig literal `.{ .pin = p }` of Scroll.
func pl3ScrollPin(p PageListPin) PageListScroll {
	return PageListScroll{Tag: PageListScrollPin, Pin: p}
}

// ---------------------------------------------------------------------------
// resize
// ---------------------------------------------------------------------------

// Zig: test "term51: resize: reflow false truncates without reflow and resets the offset cache"
func TestPl3_term51_resize_reflow_false_truncates_without_reflow_and_resets_the_offset_cache(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 3, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|abcd>\n<ef..|\n|g...|")
	s.viewport_pin_row_offset = Some[uint](7)

	pl3Resize(t, s, PageListResize{Cols: pl3Dim(2), Reflow: false})
	expectEqual(t, SizeCellCountInt(2), s.cols)
	expectEqual(t, SizeCellCountInt(3), s.rows)
	expectEqual(t, None[uint](), s.viewport_pin_row_offset)
	// Truncated, wrap flags kept as they were, nothing reflowed.
	pl3KExpect(t, s, "|ab>\n<ef|\n|g.|")
}

// Zig: test "term51: resize: equal cols with reflow only changes rows"
func TestPl3_term51_resize_equal_cols_with_reflow_only_changes_rows(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 4, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|abcd>\n<ef..|\n|....|\n|....|")

	// Shrinking rows trims the two trailing blank rows instead of making history.
	pl3Resize(t, s, PageListResize{Rows: pl3Dim(2), Reflow: true})
	expectEqual(t, SizeCellCountInt(4), s.cols)
	expectEqual(t, SizeCellCountInt(2), s.rows)
	pl3KExpect(t, s, "|abcd>\n<ef..|")

	// Same columns but explicit: no reflow happens.
	pl3Resize(t, s, PageListResize{Cols: pl3Dim(4), Rows: pl3Dim(3), Reflow: true})
	pl3KExpect(t, s, "|abcd>\n<ef..|\n|....|")
}

// Zig: test "term51: resize: layout min_max_size follows the requested dimensions"
func TestPl3_term51_resize_layout_min_max_size_follows_the_requested_dimensions(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 20, 3, Some[uint64](0))
	defer s.Deinit()

	// Without reflow (resizeWithoutReflow computes it).
	pl3Resize(t, s, PageListResize{Cols: pl3Dim(3000), Rows: pl3Dim(40), Reflow: false})
	expectEqual(t, pageListMinMaxSize(3000, 40), s.min_max_size)
	// With reflow and the same number of columns (rows only).
	pl3Resize(t, s, PageListResize{Rows: pl3Dim(5), Reflow: true})
	expectEqual(t, pageListMinMaxSize(3000, 5), s.min_max_size)
	// With reflow and other columns.
	pl3Resize(t, s, PageListResize{Cols: pl3Dim(20), Reflow: true})
	expectEqual(t, pageListMinMaxSize(20, 5), s.min_max_size)
	// No change at all.
	pl3Resize(t, s, PageListResize{Reflow: true})
	expectEqual(t, pageListMinMaxSize(20, 5), s.min_max_size)
}

// Zig: test "term51: resize: layout reflow recomputes min_max_size from the new dimensions"
func TestPl3_term51_resize_layout_reflow_recomputes_min_max_size_from_the_new_dimensions(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 20, 3, Some[uint64](0))
	defer s.Deinit()
	old := s.min_max_size
	*s.pages.first.data.GetRowAndCell(0, 0).cell = pl3Codepoint('a')

	pl3Resize(t, s, PageListResize{Cols: pl3Dim(3000), Rows: pl3Dim(40), Reflow: true})
	expectEqual(t, SizeCellCountInt(3000), s.cols)
	expectEqual(t, SizeCellCountInt(40), s.rows)
	expectEqual(t, uint(40), s.totalRows())
	expectEqual(t, pageListMinMaxSize(3000, 40), s.min_max_size)
	expect(t, s.min_max_size > old, "s.min_max_size > old")
	expectEqual(t, uint32('a'), pl3KCell(t, s, 0, 0).content.codepoint)
}

// Zig: test "term51: resize: growing cols reflows before growing rows"
func TestPl3_term51_resize_growing_cols_reflows_before_growing_rows(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 4, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|abcd>\n<ef..|\n|....|\n|....|")

	// Cols first: the reflow leaves 1 row which grow() pads back to 4 rows
	// (the old row count) and then rows shrink to 2 trimming the blanks.
	pl3Resize(t, s, PageListResize{Cols: pl3Dim(8), Rows: pl3Dim(2), Reflow: true})
	expectEqual(t, SizeCellCountInt(8), s.cols)
	expectEqual(t, SizeCellCountInt(2), s.rows)
	pl3KExpect(t, s, "|abcdef..|\n|........|")
}

// Zig: test "term51: resize: shrinking cols changes rows before reflowing"
func TestPl3_term51_resize_shrinking_cols_changes_rows_before_reflowing(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 4, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|abcd>\n<ef..|\n|....|\n|....|")

	// Rows first (trimming the 2 blank rows), then the reflow to 2 cols makes
	// 3 rows of which the first one is history.
	pl3Resize(t, s, PageListResize{Cols: pl3Dim(2), Rows: pl3Dim(2), Reflow: true})
	expectEqual(t, SizeCellCountInt(2), s.cols)
	expectEqual(t, SizeCellCountInt(2), s.rows)
	pl3KExpect(t, s, "|ab>\n<cd>\n<ef|")
	pl3KExpectPin(t, s, s.GetTopLeft(PointTagActive), 0, 1)
}

// Zig: test "term51: resize: single column and single row"
func TestPl3_term51_resize_single_column_and_single_row(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 3, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|abcd|\n|e...|\n|....|")

	pl3Resize(t, s, PageListResize{Cols: pl3Dim(1), Rows: pl3Dim(1), Reflow: true})
	expectEqual(t, SizeCellCountInt(1), s.cols)
	expectEqual(t, SizeCellCountInt(1), s.rows)
	pl3KExpect(t, s, "|a>\n<b>\n<c>\n<d|\n|e|")
	pl3KExpectPin(t, s, s.GetTopLeft(PointTagActive), 0, 4)
}

// Zig: test "term51: resize: cursor option is passed to resizeCols"
func TestPl3_term51_resize_cursor_option_is_passed_to_resizeCols(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 3, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 2)
	pl3KLoad(t, s, "|ab..|\n|cd..|\n|....|\n|....|\n|....|")

	pl3Resize(t, s, PageListResize{Cols: pl3Dim(2), Reflow: true, Cursor: pl3CursorAt(0, 0)})
	// The active area does not pull the history down: 2 rows were added.
	pl3KExpect(t, s, "|ab|\n|cd|\n|..|\n|..|\n|..|")
	pl3KExpectPin(t, s, s.GetTopLeft(PointTagActive), 0, 2)
}

// Zig: test "term51: resize: pinned viewport that becomes active after reflow is made active"
func TestPl3_term51_resize_pinned_viewport_that_becomes_active_after_reflow_is_made_active(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 2, 4, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 8)
	pl3KLoad(t, s, pl3KAA)

	s.Scroll(pl3ScrollPin(pl3Pin(t, s, pl3Screen(0, 6))))
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")

	// 12 rows unwrap to 6, the active area starts at 2 and the pin that was
	// on row 6 is now on row 3.
	pl3Resize(t, s, PageListResize{Cols: pl3Dim(4), Reflow: true})
	expectEqual(t, uint(6), s.totalRows())
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	pl3KExpect(t, s, "|aaaa|\n|aaaa|\n|aaaa|\n|aaaa|\n|aaaa|\n|aaaa|")
}

// Zig: test "term51: resize: pinned viewport stays pinned when still in history after reflow"
func TestPl3_term51_resize_pinned_viewport_stays_pinned_when_still_in_history_after_reflow(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 2, 4, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 8)
	pl3KLoad(t, s, pl3KAA)

	s.Scroll(pl3ScrollPin(pl3Pin(t, s, pl3Screen(0, 2))))
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")

	pl3Resize(t, s, PageListResize{Cols: pl3Dim(4), Reflow: true})
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	pl3KExpectPin(t, s, *s.viewport_pin, 0, 1)
	expectEqual(t, uint(6), s.totalRows())
}

// Zig: test "term51: resize: top and active viewports are unchanged by a reflow"
func TestPl3_term51_resize_top_and_active_viewports_are_unchanged_by_a_reflow(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 2, 4, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 8)
	pl3KLoad(t, s, pl3KAA)

	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
	pl3Resize(t, s, PageListResize{Cols: pl3Dim(4), Reflow: true})
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")

	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	pl3Resize(t, s, PageListResize{Cols: pl3Dim(2), Reflow: true})
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	expectEqual(t, uint(12), s.totalRows())
}

// Zig: test "term51: resize: growing rows makes a pinned viewport active when it no longer fits"
func TestPl3_term51_resize_growing_rows_makes_a_pinned_viewport_active_when_it_no_longer_fits(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 2, 4, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 8)

	s.Scroll(pl3ScrollPin(pl3Pin(t, s, pl3Screen(0, 7))))
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")

	// Pin row 7 plus 6 rows is beyond the 12 total rows.
	pl3Resize(t, s, PageListResize{Rows: pl3Dim(6), Reflow: true})
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	expectEqual(t, uint(12), s.totalRows())
}

// Zig: test "term51: resize: pinned viewport survives a cursor guided row growth without reflow"
func TestPl3_term51_resize_pinned_viewport_survives_a_cursor_guided_row_growth_without_reflow(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 2, 4, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 8)

	s.Scroll(pl3ScrollPin(pl3Pin(t, s, pl3Screen(0, 1))))
	pl3Resize(t, s, PageListResize{Rows: pl3Dim(6), Reflow: false, Cursor: pl3CursorAt(0, 0)})
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	pl3KExpectPin(t, s, *s.viewport_pin, 0, 1)
	expectEqual(t, uint(14), s.totalRows())
	expectEqual(t, None[uint](), s.viewport_pin_row_offset)
}

// Zig: test "term51: resize: invalidates the cached viewport row offset"
func TestPl3_term51_resize_invalidates_the_cached_viewport_row_offset(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 2, 4, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 8)

	s.Scroll(pl3ScrollPin(pl3Pin(t, s, pl3Screen(0, 2))))
	_ = s.Scrollbar()
	expectEqual(t, Some[uint](2), s.viewport_pin_row_offset)

	pl3Resize(t, s, PageListResize{Rows: pl3Dim(2), Reflow: true})
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, None[uint](), s.viewport_pin_row_offset)
	// The two trailing blank rows were trimmed.
	expectEqual(t, uint(10), s.totalRows())
}

// ---------------------------------------------------------------------------
// resizeCols
// ---------------------------------------------------------------------------

// Zig: test "term51: resizeCols: shrinking wraps rows that no longer fit"
func TestPl3_term51_resizeCols_shrinking_wraps_rows_that_no_longer_fit(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 6, 3, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|abcdef|\n|gh....|\n|......|")

	pl3ResizeCols(t, s, 4, pl3NoCursor())
	expectEqual(t, SizeCellCountInt(4), s.cols)
	expectEqual(t, SizeCellCountInt(3), s.rows)
	pl3KExpect(t, s, "|abcd>\n<ef..|\n|gh..|")
}

// Zig: test "term51: resizeCols: growing unwraps rows and pads the active area"
func TestPl3_term51_resizeCols_growing_unwraps_rows_and_pads_the_active_area(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 4, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|abcd>\n<ef..|\n|gh..|\n|....|")

	pl3ResizeCols(t, s, 8, pl3NoCursor())
	expectEqual(t, SizeCellCountInt(8), s.cols)
	expectEqual(t, SizeCellCountInt(4), s.rows)
	// 2 rows after the reflow, grow() added the 2 missing active rows.
	pl3KExpect(t, s, "|abcdef..|\n|gh......|\n|........|\n|........|")
}

// Zig: test "term51: resizeCols: blank rows between text are kept and trailing ones padded"
func TestPl3_term51_resizeCols_blank_rows_between_text_are_kept_and_trailing_ones_padded(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 5, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|a...|\n|....|\n|b...|\n|....|\n|....|")

	pl3ResizeCols(t, s, 2, pl3NoCursor())
	pl3KExpect(t, s, "|a.|\n|..|\n|b.|\n|..|\n|..|")
}

// Zig: test "term51: resizeCols: wrapped source rows keep their blank cells"
func TestPl3_term51_resizeCols_wrapped_source_rows_keep_their_blank_cells(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 2, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|abc.>\n<d...|")

	pl3ResizeCols(t, s, 8, pl3NoCursor())
	pl3KExpect(t, s, "|abc.d...|\n|........|")
}

// Zig: test "term51: resizeCols: cursor keeps the active area from pulling history down"
func TestPl3_term51_resizeCols_cursor_keeps_the_active_area_from_pulling_history_down(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 3, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 2)
	pl3KLoad(t, s, "|ab..|\n|cd..|\n|....|\n|....|\n|....|")

	pl3ResizeCols(t, s, 2, pl3CursorAt(0, 0))
	// The blank cursor row is written and 2 more rows are added so that the
	// cursor is back on the first row of the active area.
	pl3KExpect(t, s, "|ab|\n|cd|\n|..|\n|..|\n|..|")
	pl3KExpectPin(t, s, s.GetTopLeft(PointTagActive), 0, 2)
	// The temporary tracked pin is gone.
	expectEqual(t, uint(1), s.CountTrackedPins())
}

// Zig: test "term51: resizeCols: without a cursor the active area pulls history down"
func TestPl3_term51_resizeCols_without_a_cursor_the_active_area_pulls_history_down(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 3, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 2)
	pl3KLoad(t, s, "|ab..|\n|cd..|\n|....|\n|....|\n|....|")

	pl3ResizeCols(t, s, 2, pl3NoCursor())
	// Trailing blank rows are never written so only 2 rows come out of the
	// reflow, grow() pads to the 3 active rows.
	pl3KExpect(t, s, "|ab|\n|cd|\n|..|")
	pl3KExpectPin(t, s, s.GetTopLeft(PointTagActive), 0, 0)
}

// Zig: test "term51: resizeCols: cursor outside the active area is ignored"
func TestPl3_term51_resizeCols_cursor_outside_the_active_area_is_ignored(t *testing.T) {
	testStdCapacity(t)
	{
		// y is beyond the last row.
		s := pl3Init(t, 4, 3, None[uint64]())
		defer s.Deinit()
		pl3KLoad(t, s, "|ab..|\n|cd..|\n|....|")
		pl3ResizeCols(t, s, 2, pl3CursorAt(0, 10))
		pl3KExpect(t, s, "|ab|\n|cd|\n|..|")
		expectEqual(t, uint(1), s.CountTrackedPins())
	}
	{
		// x is beyond the last column.
		s := pl3Init(t, 4, 3, None[uint64]())
		defer s.Deinit()
		pl3KLoad(t, s, "|ab..|\n|cd..|\n|....|")
		pl3ResizeCols(t, s, 2, pl3CursorAt(4, 0))
		pl3KExpect(t, s, "|ab|\n|cd|\n|..|")
		expectEqual(t, uint(1), s.CountTrackedPins())
	}
}

// Zig: test "term51: resizeCols: rows wrapped by the reflow reduce the rows added for the cursor"
func TestPl3_term51_resizeCols_rows_wrapped_by_the_reflow_reduce_the_rows_added_for_the_cursor(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 3, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 2)
	pl3KLoad(t, s, "|abcd|\n|efgh|\n|....|\n|....|\n|....|")

	pl3ResizeCols(t, s, 2, pl3CursorAt(0, 0))
	// After the reflow there are 5 rows (ab cd ef gh + cursor row). gh is a
	// wrapped row inside the active area that did not exist before, so only 1
	// row (not 2) is added.
	pl3KExpect(t, s, "|ab>\n<cd|\n|ef>\n<gh|\n|..|\n|..|")
	pl3KExpectPin(t, s, s.GetTopLeft(PointTagActive), 0, 3)
}

// Zig: test "term51: resizeCols: cursor in a wrapped row counts the original wraps"
func TestPl3_term51_resizeCols_cursor_in_a_wrapped_row_counts_the_original_wraps(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 5, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|abcd>\n<ef..|\n|....|\n|....|\n|....|")

	// The cursor is on the continuation row; unwrapping removes the wrap.
	pl3ResizeCols(t, s, 8, pl3CursorAt(0, 1))
	pl3KExpect(t, s, "|abcdef..|\n|........|\n|........|\n|........|\n|........|")
	pl3KExpectPin(t, s, s.GetTopLeft(PointTagActive), 0, 0)
}

// Zig: test "term51: resizeCols: layout capacity falls back when the columns do not fit a standard page"
func TestPl3_term51_resizeCols_layout_capacity_falls_back_when_the_columns_do_not_fit_a_standard_page(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 10, 1, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|abc.......|")

	cols := zigUnwrap(t, PageStdCapacity.MaxCols()) + 1
	_, err := PageStdCapacity.Adjust(PageCapacityAdjustment{cols: Some(cols)})
	pl3ExpectError(t, ErrOutOfMemory, err)
	pl3ResizeCols(t, s, cols, pl3NoCursor())

	expectEqual(t, uint(1), s.TotalPages())
	page := &s.pages.first.data
	expectEqual(t, cols, page.capacity.cols)
	expectEqual(t, cols, page.size.cols)
	// rows = min(rows of the old page, capacity rows).
	expectEqual(t, SizeCellCountInt(1), page.capacity.rows)
	expectEqual(t, uint(1), s.totalRows())
	expectEqual(t, uint32('a'), pl3KCell(t, s, 0, 0).content.codepoint)
	expectEqual(t, uint32('c'), pl3KCell(t, s, 2, 0).content.codepoint)
	expectEqual(t, uint32(0), pl3KCell(t, s, 3, 0).content.codepoint)
	s.AssertIntegrity()
}

// Zig: test "term51: resizeCols: layout the maximum fitting columns still use a standard capacity"
func TestPl3_term51_resizeCols_layout_the_maximum_fitting_columns_still_use_a_standard_capacity(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 10, 1, Some[uint64](0))
	defer s.Deinit()

	cols := zigUnwrap(t, PageStdCapacity.MaxCols())
	pl3ResizeCols(t, s, cols, pl3NoCursor())
	page := &s.pages.first.data
	expectEqual(t, cols, page.capacity.cols)
	expectEqual(t, SizeCellCountInt(1), page.capacity.rows)
	expect(t, pageListMemoryLen(page) <= pageListStdSize(), "page.memory.len <= std_size")
	s.AssertIntegrity()
}

// Zig: test "term51: resizeCols: layout source pages are consumed and replaced across pages"
func TestPl3_term51_resizeCols_layout_source_pages_are_consumed_and_replaced_across_pages(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 1000, 3, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 60)
	expectEqual(t, uint(2), s.TotalPages())
	expectEqual(t, uint(63), s.totalRows())

	// Every row has text in the first and last column.
	for y := 0; y < 63; y++ {
		first := pl3KPin(t, s, 0, y)
		*first.RowAndCell().cell = pl3Codepoint('a' + uint32(y%26))
		last := pl3KPin(t, s, 999, y)
		*last.RowAndCell().cell = pl3Codepoint('Z')
	}

	cap, err := s.pages.first.data.capacity.Adjust(PageCapacityAdjustment{cols: Some[SizeCellCountInt](500)})
	zigTry(t, err)
	pl3ResizeCols(t, s, 500, pl3NoCursor())
	expectEqual(t, uint(126), s.totalRows())
	expectEqual(t, uint(126), s.total_rows)
	// Pages hold as many rows as fit with 500 columns.
	expectEqualSlices(t, []uint{uint(cap.rows), 126 - uint(cap.rows)}, pl3KRowsPerPage(s))

	for y := 0; y < 63; y++ {
		c := pl3KCell(t, s, 0, y*2)
		expectEqual(t, 'a'+uint32(y%26), c.content.codepoint)
		expect(t, pl3KRow(t, s, y*2).wrap, "term51KRow(&s, y * 2).wrap")
		expect(t, pl3KRow(t, s, y*2+1).wrap_continuation, "term51KRow(&s, y * 2 + 1).wrap_continuation")
		expectEqual(t, uint32('Z'), pl3KCell(t, s, 499, y*2+1).content.codepoint)
	}
	s.AssertIntegrity()
}

// Zig: test "term51: resizeCols: layout pruning while padding for the cursor drops the oldest page"
func TestPl3_term51_resizeCols_layout_pruning_while_padding_for_the_cursor_drops_the_oldest_page(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 2000, 4, None[uint64]())
	defer s.Deinit()
	// 2000 columns: 23 rows per page. 93 rows span 5 pages.
	pl3GrowRows(t, s, 89)
	expectEqual(t, uint(93), s.totalRows())
	s.explicit_max_size = 0
	for y := 0; y < 90; y++ {
		p := pl3KPin(t, s, 0, y)
		*p.RowAndCell().cell = pl3Codepoint('a' + uint32(y%26))
	}
	// A user pin on the very first row.
	first_pin := pl3Track(t, s, pl3KPin(t, s, 0, 0))
	defer s.UntrackPin(first_pin)

	s.min_max_size = pageListMinMaxSize(1000, 4)
	pl3ResizeCols(t, s, 1000, pl3CursorAt(0, 0))

	// 1000 columns: 46 rows per page. The reflow leaves 90 rows (the cursor
	// row is the last one) in pages of 46 and 44 rows. Padding the 3 rows
	// beneath the cursor adds 2 rows and then needs a third page, which
	// prunes the first page.
	expectEqual(t, uint(47), s.totalRows())
	expectEqual(t, uint(47), s.total_rows)
	expectEqual(t, 2, len(pl3KRowsPerPage(s)))
	expectEqualSlices(t, []uint{46, 1}, pl3KRowsPerPage(s))
	expect(t, first_pin.Garbage, "first_pin.garbage")
	pl3KExpectPin(t, s, *first_pin, 0, 0)
	expectEqual(t, uint32('a'+(46%26)), pl3KCell(t, s, 0, 0).content.codepoint)
	pl3KExpectPin(t, s, s.GetTopLeft(PointTagActive), 0, 43)
	s.AssertIntegrity()
}

// Zig: test "term51: resizeCols: a cursor that ends up in history does not add rows"
func TestPl3_term51_resizeCols_a_cursor_that_ends_up_in_history_does_not_add_rows(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 3, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|abcd|\n|efgh|\n|ijkl|")

	// The rows below the cursor wrap and push its row out of the active area.
	pl3ResizeCols(t, s, 2, pl3CursorAt(0, 0))
	pl3KExpect(t, s, "|ab>\n<cd|\n|ef>\n<gh|\n|ij>\n<kl|")
	pl3KExpectPin(t, s, s.GetTopLeft(PointTagActive), 0, 3)
	expectEqual(t, uint(1), s.CountTrackedPins())
}

// Zig: test "term51: resizeCols: layout cursor in the second page"
func TestPl3_term51_resizeCols_layout_cursor_in_the_second_page(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 1000, 3, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 60)
	expectEqualSlices(t, []uint{46, 17}, pl3KRowsPerPage(s))
	// Text in the history only, one char per row.
	for y := 0; y < 60; y++ {
		*pl3KPin(t, s, 0, y).RowAndCell().cell = PageCellInit('a' + uint32(y%26))
	}

	// The cursor is on the first active row (screen row 60, in the second page).
	pl3ResizeCols(t, s, 500, pl3CursorAt(0, 0))
	// The cursor row is written, so there are 61 rows after the reflow and 2
	// are added below the cursor.
	expectEqual(t, uint(63), s.totalRows())
	expectEqual(t, uint(63), s.total_rows)
	pl3KExpectPin(t, s, s.GetTopLeft(PointTagActive), 0, 60)
	expectEqual(t, uint32('a'+(59%26)), pl3KCell(t, s, 0, 59).content.codepoint)
	expectEqual(t, uint32(0), pl3KCell(t, s, 0, 60).content.codepoint)
	s.AssertIntegrity()
}

// Zig: test "term51: resizeCols: layout cursor in the second page that wraps out of the active area"
func TestPl3_term51_resizeCols_layout_cursor_in_the_second_page_that_wraps_out_of_the_active_area(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 1000, 3, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 60)
	for y := 0; y < 63; y++ {
		*pl3KPin(t, s, 0, y).RowAndCell().cell = PageCellInit('a')
		*pl3KPin(t, s, 999, y).RowAndCell().cell = PageCellInit('Z')
	}

	// Every row has 2 rows afterwards, the cursor (row 61 of the original) is
	// now row 122 and the active area starts at row 123.
	pl3ResizeCols(t, s, 500, pl3CursorAt(0, 1))
	expectEqual(t, uint(126), s.totalRows())
	pl3KExpectPin(t, s, s.GetTopLeft(PointTagActive), 0, 123)
	s.AssertIntegrity()
}

// ---------------------------------------------------------------------------
// ReflowCursor
// ---------------------------------------------------------------------------

// Zig: test "term51: ReflowCursor.init: positions the cursor on the first cell of the node"
func TestPl3_term51_ReflowCursor_init_positions_the_cursor_on_the_first_cell_of_the_node(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 3, Some[uint64](0))
	defer s.Deinit()

	node := s.pages.first
	rc := pageListReflowCursorInit(node)
	rows := node.data.rows
	expectEqual(t, SizeCellCountInt(0), rc.x)
	expectEqual(t, SizeCellCountInt(0), rc.y)
	expect(t, !rc.pending_wrap, "!rc.pending_wrap")
	expect(t, node == rc.node, "node == rc.node")
	expect(t, &node.data == rc.page, "&node.data == rc.page")
	expect(t, &rows[0] == rc.page_row, "&rows[0] == rc.page_row")
	expect(t, pl3KRowCell(&node.data, 0, 0) == rc.page_cell, "&rows[0].cells.ptr(node.data.memory)[0] == rc.page_cell")
	expectEqual(t, uint(0), rc.new_rows)
	// The total rows is the size of the node, not its capacity.
	expectEqual(t, uint(3), rc.total_rows)
}

// Zig: test "term51: ReflowCursor.init: total rows follows the node size"
func TestPl3_term51_ReflowCursor_init_total_rows_follows_the_node_size(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 3, Some[uint64](0))
	defer s.Deinit()
	node := pl3KAddNode(t, s, pl3Cap(6, 8), 1)

	rc := pageListReflowCursorInit(node)
	expectEqual(t, uint(1), rc.total_rows)
	expectEqual(t, SizeCellCountInt(6), rc.page.size.cols)
	expectEqual(t, SizeCellCountInt(1), rc.page.size.rows)
}

// Zig: test "term51: ReflowCursor.cursorForward: moves right and sets pending wrap at the last column"
func TestPl3_term51_ReflowCursor_cursorForward_moves_right_and_sets_pending_wrap_at_the_last_column(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 3, 1, Some[uint64](0))
	defer s.Deinit()

	rc := pageListReflowCursorInit(s.pages.first)
	cells := rc.page.GetCells(rc.page_row)

	rc.cursorForward()
	expectEqual(t, SizeCellCountInt(1), rc.x)
	expect(t, &cells[1] == rc.page_cell, "&cells[1] == rc.page_cell")
	expect(t, !rc.pending_wrap, "!rc.pending_wrap")

	rc.cursorForward()
	expectEqual(t, SizeCellCountInt(2), rc.x)
	expect(t, &cells[2] == rc.page_cell, "&cells[2] == rc.page_cell")
	expect(t, !rc.pending_wrap, "!rc.pending_wrap")

	// At the last column the cursor stays and a wrap is pending.
	rc.cursorForward()
	expectEqual(t, SizeCellCountInt(2), rc.x)
	expect(t, &cells[2] == rc.page_cell, "&cells[2] == rc.page_cell")
	expect(t, rc.pending_wrap, "rc.pending_wrap")

	// And stays pending.
	rc.cursorForward()
	expectEqual(t, SizeCellCountInt(2), rc.x)
	expect(t, rc.pending_wrap, "rc.pending_wrap")
}

// Zig: test "term51: ReflowCursor.cursorForward: single column page is immediately pending"
func TestPl3_term51_ReflowCursor_cursorForward_single_column_page_is_immediately_pending(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 1, 1, Some[uint64](0))
	defer s.Deinit()

	rc := pageListReflowCursorInit(s.pages.first)
	rc.cursorForward()
	expectEqual(t, SizeCellCountInt(0), rc.x)
	expect(t, rc.pending_wrap, "rc.pending_wrap")
}

// Zig: test "term51: ReflowCursor.cursorScroll: adds a row and moves to its start"
func TestPl3_term51_ReflowCursor_cursorScroll_adds_a_row_and_moves_to_its_start(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	node := pl3KAddNode(t, s, pl3Cap(4, 3), 1)

	rc := pageListReflowCursorInit(node)
	rows := node.data.rows
	rc.cursorAbsolute(2, 0)
	rc.pending_wrap = true

	rc.cursorScroll()
	expectEqual(t, SizeCellCountInt(2), node.data.size.rows)
	expectEqual(t, SizeCellCountInt(0), rc.x)
	expectEqual(t, SizeCellCountInt(1), rc.y)
	expect(t, !rc.pending_wrap, "!rc.pending_wrap")
	expect(t, &rows[1] == rc.page_row, "&rows[1] == rc.page_row")
	expect(t, pl3KRowCell(&node.data, 1, 0) == rc.page_cell, "&rows[1].cells.ptr(node.data.memory)[0] == rc.page_cell")
	// cursorScroll alone does not touch the row count of the cursor.
	expectEqual(t, uint(1), rc.total_rows)

	// Fill the page up to its capacity.
	rc.cursorScroll()
	expectEqual(t, SizeCellCountInt(3), node.data.size.rows)
	expectEqual(t, SizeCellCountInt(2), rc.y)
	expect(t, &rows[2] == rc.page_row, "&rows[2] == rc.page_row")
	expect(t, rc.bottom(), "rc.bottom()")
	s.total_rows += 2
}

// Zig: test "term51: ReflowCursor.bottom: compares against the capacity not the size"
func TestPl3_term51_ReflowCursor_bottom_compares_against_the_capacity_not_the_size(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	node := pl3KAddNode(t, s, pl3Cap(4, 3), 3)

	rc := pageListReflowCursorInit(node)
	expect(t, !rc.bottom(), "!rc.bottom()")
	rc.cursorAbsolute(0, 1)
	expect(t, !rc.bottom(), "!rc.bottom()")
	rc.cursorAbsolute(0, 2)
	expect(t, rc.bottom(), "rc.bottom()")

	// A page with a single row of capacity is always at the bottom.
	small := pl3KAddNode(t, s, pl3Cap(4, 1), 1)
	rc2 := pageListReflowCursorInit(small)
	expect(t, rc2.bottom(), "rc2.bottom()")

	// A page whose size is below its capacity is not at the bottom on its
	// last row.
	big := pl3KAddNode(t, s, pl3Cap(4, 5), 2)
	rc3 := pageListReflowCursorInit(big)
	rc3.cursorAbsolute(0, 1)
	expect(t, !rc3.bottom(), "!rc3.bottom()")
}

// Zig: test "term51: ReflowCursor.cursorNewPage: starts a new node after the current one"
func TestPl3_term51_ReflowCursor_cursorNewPage_starts_a_new_node_after_the_current_one(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	node := pl3KAddNode(t, s, pl3Cap(4, 3), 3)

	rc := pageListReflowCursorInit(node)
	rc.cursorAbsolute(3, 2)
	rc.cursorForward()
	expect(t, rc.pending_wrap, "rc.pending_wrap")
	rc.new_rows = 5
	rc.total_rows = 9

	zigTry(t, rc.cursorNewPage(s, pl3Cap(4, 2)))
	s.total_rows += 1
	expect(t, rc.node != node, "rc.node != node")
	expect(t, node.next == rc.node, "node.next == rc.node")
	expect(t, rc.node == s.pages.last, "rc.node == s.pages.last.?")
	expect(t, &rc.node.data == rc.page, "&rc.node.data == rc.page")
	expectEqual(t, SizeCellCountInt(0), rc.x)
	expectEqual(t, SizeCellCountInt(0), rc.y)
	expect(t, !rc.pending_wrap, "!rc.pending_wrap")
	expectEqual(t, SizeCellCountInt(1), rc.page.size.rows)
	expectEqual(t, SizeCellCountInt(2), rc.page.capacity.rows)
	// The deferred rows are preserved but the total row count is reset to the
	// new node's size: callers restore it.
	expectEqual(t, uint(5), rc.new_rows)
	expectEqual(t, uint(1), rc.total_rows)
	// The old page is untouched.
	expectEqual(t, SizeCellCountInt(3), node.data.size.rows)
}

// Zig: test "term51: ReflowCursor.cursorNewPage: the new page may have other dimensions"
func TestPl3_term51_ReflowCursor_cursorNewPage_the_new_page_may_have_other_dimensions(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	node := pl3KAddNode(t, s, pl3Cap(4, 2), 1)

	rc := pageListReflowCursorInit(node)
	cap := pl3Cap(7, 4)
	cap.styles = 32
	zigTry(t, rc.cursorNewPage(s, cap))
	s.total_rows += 1
	expectEqual(t, SizeCellCountInt(7), rc.page.size.cols)
	expectEqual(t, SizeCellCountInt(4), rc.page.capacity.rows)
	expectEqual(t, SizeStyleCountInt(32), rc.page.capacity.styles)
	// Serial numbers keep increasing.
	expect(t, rc.node.serial > node.serial, "rc.node.serial > node.serial")
}

// Zig: test "term51: ReflowCursor.cursorScrollOrNewPage: scrolls when there is room"
func TestPl3_term51_ReflowCursor_cursorScrollOrNewPage_scrolls_when_there_is_room(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	node := pl3KAddNode(t, s, pl3Cap(4, 3), 1)

	rc := pageListReflowCursorInit(node)
	rc.total_rows = 10
	rc.new_rows = 3
	zigTry(t, rc.cursorScrollOrNewPage(s, pl3Cap(4, 3)))
	s.total_rows += 1
	expect(t, node == rc.node, "node == rc.node")
	expectEqual(t, SizeCellCountInt(1), rc.y)
	expectEqual(t, SizeCellCountInt(2), node.data.size.rows)
	expectEqual(t, uint(11), rc.total_rows)
	expectEqual(t, uint(3), rc.new_rows)
	expectEqual(t, uint(2), s.TotalPages())
}

// Zig: test "term51: ReflowCursor.cursorScrollOrNewPage: creates a page at the bottom of the capacity"
func TestPl3_term51_ReflowCursor_cursorScrollOrNewPage_creates_a_page_at_the_bottom_of_the_capacity(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	node := pl3KAddNode(t, s, pl3Cap(4, 2), 2)

	rc := pageListReflowCursorInit(node)
	rc.cursorAbsolute(0, 1)
	rc.total_rows = 10
	rc.new_rows = 3
	zigTry(t, rc.cursorScrollOrNewPage(s, pl3Cap(4, 2)))
	s.total_rows += 1
	expect(t, rc.node != node, "rc.node != node")
	expectEqual(t, SizeCellCountInt(0), rc.y)
	expectEqual(t, uint(11), rc.total_rows)
	expectEqual(t, uint(3), rc.new_rows)
	expectEqual(t, SizeCellCountInt(2), node.data.size.rows)
	expectEqual(t, uint(3), s.TotalPages())
}

// Zig: test "term51: ReflowCursor.cursorAbsolute: moves in every direction and clears the pending wrap"
func TestPl3_term51_ReflowCursor_cursorAbsolute_moves_in_every_direction_and_clears_the_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	node := pl3KAddNode(t, s, pl3Cap(4, 4), 3)

	rc := pageListReflowCursorInit(node)
	rows := node.data.rows
	rc.pending_wrap = true

	// Down and right.
	rc.cursorAbsolute(3, 2)
	expectEqual(t, SizeCellCountInt(3), rc.x)
	expectEqual(t, SizeCellCountInt(2), rc.y)
	expect(t, !rc.pending_wrap, "!rc.pending_wrap")
	expect(t, &rows[2] == rc.page_row, "&rows[2] == rc.page_row")
	expect(t, pl3KRowCell(&node.data, 2, 3) == rc.page_cell, "&rows[2].cells.ptr(node.data.memory)[3] == rc.page_cell")

	// Up and left.
	rc.pending_wrap = true
	rc.cursorAbsolute(1, 0)
	expectEqual(t, SizeCellCountInt(1), rc.x)
	expectEqual(t, SizeCellCountInt(0), rc.y)
	expect(t, !rc.pending_wrap, "!rc.pending_wrap")
	expect(t, &rows[0] == rc.page_row, "&rows[0] == rc.page_row")
	expect(t, pl3KRowCell(&node.data, 0, 1) == rc.page_cell, "&rows[0].cells.ptr(node.data.memory)[1] == rc.page_cell")

	// Same row.
	rc.pending_wrap = true
	rc.cursorAbsolute(2, 0)
	expectEqual(t, SizeCellCountInt(2), rc.x)
	expectEqual(t, SizeCellCountInt(0), rc.y)
	expect(t, !rc.pending_wrap, "!rc.pending_wrap")
	expect(t, &rows[0] == rc.page_row, "&rows[0] == rc.page_row")
	expect(t, pl3KRowCell(&node.data, 0, 2) == rc.page_cell, "&rows[0].cells.ptr(node.data.memory)[2] == rc.page_cell")

	// One row only, last valid position.
	rc.cursorAbsolute(3, 2)
	expect(t, pl3KRowCell(&node.data, 2, 3) == rc.page_cell, "&rows[2].cells.ptr(node.data.memory)[3] == rc.page_cell")
	// Back to the origin.
	rc.cursorAbsolute(0, 0)
	expect(t, pl3KRowCell(&node.data, 0, 0) == rc.page_cell, "&rows[0].cells.ptr(node.data.memory)[0] == rc.page_cell")
}

// Zig: test "term51: ReflowCursor.countTrailingEmptyCells: counts the empty cells at the end of the row"
func TestPl3_term51_ReflowCursor_countTrailingEmptyCells_counts_the_empty_cells_at_the_end_of_the_row(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()

	rc := pageListReflowCursorInit(s.pages.first)
	// Empty row without prompt: everything from the cursor on.
	expectEqual(t, uint(4), rc.countTrailingEmptyCells())
	rc.cursorAbsolute(1, 0)
	expectEqual(t, uint(3), rc.countTrailingEmptyCells())
	rc.cursorAbsolute(3, 0)
	expectEqual(t, uint(1), rc.countTrailingEmptyCells())

	// Text at the end: none.
	rc.cursorAbsolute(0, 0)
	pl3KSetCell(&rc, 3, PageCellInit('x'))
	expectEqual(t, uint(0), rc.countTrailingEmptyCells())

	// Text in the middle: only the cells after it count.
	pl3KSetCell(&rc, 3, PageCell{})
	pl3KSetCell(&rc, 1, PageCellInit('x'))
	expectEqual(t, uint(2), rc.countTrailingEmptyCells())

	// Text before the cursor is not looked at.
	rc.cursorAbsolute(2, 0)
	expectEqual(t, uint(2), rc.countTrailingEmptyCells())
	// Text on the cursor cell is.
	pl3KSetCell(&rc, 2, PageCellInit('y'))
	expectEqual(t, uint(1), rc.countTrailingEmptyCells())
}

// Zig: test "term51: ReflowCursor.countTrailingEmptyCells: wrapped rows have no trailing empty cells"
func TestPl3_term51_ReflowCursor_countTrailingEmptyCells_wrapped_rows_have_no_trailing_empty_cells(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()

	rc := pageListReflowCursorInit(s.pages.first)
	rc.page_row.wrap = true
	expectEqual(t, uint(0), rc.countTrailingEmptyCells())
	rc.page_row.wrap = false
	expectEqual(t, uint(4), rc.countTrailingEmptyCells())
}

// Zig: test "term51: ReflowCursor.countTrailingEmptyCells: semantic prompt rows keep one cell"
func TestPl3_term51_ReflowCursor_countTrailingEmptyCells_semantic_prompt_rows_keep_one_cell(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()

	rc := pageListReflowCursorInit(s.pages.first)
	rc.page_row.semantic_prompt = PageRowSemanticPromptPrompt
	expectEqual(t, uint(3), rc.countTrailingEmptyCells())
	rc.page_row.semantic_prompt = PageRowSemanticPromptPromptContinuation
	expectEqual(t, uint(3), rc.countTrailingEmptyCells())
	rc.cursorAbsolute(3, 0)
	expectEqual(t, uint(0), rc.countTrailingEmptyCells())

	// With text the prompt does not matter.
	rc.cursorAbsolute(0, 0)
	pl3KSetCell(&rc, 1, PageCellInit('x'))
	expectEqual(t, uint(2), rc.countTrailingEmptyCells())
	rc.page_row.semantic_prompt = PageRowSemanticPromptNone
	expectEqual(t, uint(2), rc.countTrailingEmptyCells())
}

// Zig: test "term51: ReflowCursor.countTrailingEmptyCells: spacers and colors are not empty"
func TestPl3_term51_ReflowCursor_countTrailingEmptyCells_spacers_and_colors_are_not_empty(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()

	rc := pageListReflowCursorInit(s.pages.first)
	pl3KSetCell(&rc, 3, pl3KSpacerHead())
	expectEqual(t, uint(0), rc.countTrailingEmptyCells())
	pl3KSetCell(&rc, 3, pl3KSpacerTail())
	expectEqual(t, uint(0), rc.countTrailingEmptyCells())
	pl3KSetCell(&rc, 3, PageCell{content_tag: PageCellContentTagBgColorPalette, content: PageCellContent{color_palette: 3}})
	expectEqual(t, uint(0), rc.countTrailingEmptyCells())
	pl3KSetCell(&rc, 3, PageCell{content_tag: PageCellContentTagBgColorRgb, content: PageCellContent{color_rgb: PageCellRGB{R: 1, G: 2, B: 3}}})
	expectEqual(t, uint(0), rc.countTrailingEmptyCells())
	// A blank cell that carries a style id but no text is empty.
	pl3KSetCell(&rc, 3, PageCell{style_id: 5})
	expectEqual(t, uint(4), rc.countTrailingEmptyCells())
	pl3KSetCell(&rc, 3, PageCell{})
}

// Zig: test "term51: ReflowCursor.copyRowMetadata: copies only the semantic prompt"
func TestPl3_term51_ReflowCursor_copyRowMetadata_copies_only_the_semantic_prompt(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 2, Some[uint64](0))
	defer s.Deinit()

	rc := pageListReflowCursorInit(s.pages.first)
	rows := rc.page.rows
	src := rows[1]
	src.wrap = true
	src.wrap_continuation = true
	src.styled = true
	src.semantic_prompt = PageRowSemanticPromptPromptContinuation

	rc.copyRowMetadata(&src)
	expectEqual(t, PageRowSemanticPromptPromptContinuation, rc.page_row.semantic_prompt)
	expect(t, !rc.page_row.wrap, "!rc.page_row.wrap")
	expect(t, !rc.page_row.wrap_continuation, "!rc.page_row.wrap_continuation")
	expect(t, !rc.page_row.styled, "!rc.page_row.styled")
	// The cells pointer of the destination is untouched.
	expectEqual(t, rows[0].cells, rc.page_row.cells)

	src.semantic_prompt = PageRowSemanticPromptPrompt
	rc.copyRowMetadata(&src)
	expectEqual(t, PageRowSemanticPromptPrompt, rc.page_row.semantic_prompt)

	// Copying none clears the destination prompt and keeps its own flags.
	rc.page_row.wrap = true
	src.semantic_prompt = PageRowSemanticPromptNone
	rc.copyRowMetadata(&src)
	expectEqual(t, PageRowSemanticPromptNone, rc.page_row.semantic_prompt)
	expect(t, rc.page_row.wrap, "rc.page_row.wrap")
}

// Zig: test "term51: ReflowCursor.moveLastRowToNewPage: moves the row and its pins to a new page"
func TestPl3_term51_ReflowCursor_moveLastRowToNewPage_moves_the_row_and_its_pins_to_a_new_page(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	node := pl3KAddNode(t, s, pl3Cap(4, 3), 2)
	page := &node.data

	// Row 0: "ab", row 1: "cd" with a grapheme on c and style on d, wrapped,
	// with a prompt.
	*page.GetRowAndCell(0, 0).cell = PageCellInit('a')
	*page.GetRowAndCell(1, 0).cell = PageCellInit('b')
	{
		c := page.GetRowAndCell(0, 1)
		*c.cell = PageCellInit('c')
		zigTry(t, page.AppendGrapheme(c.row, c.cell, 0x301))
		c.row.wrap = true
		c.row.semantic_prompt = PageRowSemanticPromptPrompt
		d := page.GetRowAndCell(1, 1)
		*d.cell = PageCellInit('d')
		id, err := page.styles.Add(pl3Bold())
		zigTry(t, err)
		d.cell.style_id = id
		d.row.styled = true
	}

	pin_row1 := pl3Track(t, s, PageListPin{Node: node, X: 1, Y: 1})
	defer s.UntrackPin(pin_row1)
	pin_row0 := pl3Track(t, s, PageListPin{Node: node, X: 1, Y: 0})
	defer s.UntrackPin(pin_row0)

	rc := pageListReflowCursorInit(node)
	rc.cursorAbsolute(2, 1)
	rc.total_rows = 7
	rc.new_rows = 4
	zigTry(t, rc.moveLastRowToNewPage(s, pl3Cap(4, 3)))

	// Cursor on the same column of the first row of the new page.
	expect(t, rc.node != node, "rc.node != node")
	expect(t, node.next == rc.node, "node.next == rc.node")
	expectEqual(t, SizeCellCountInt(0), rc.y)
	expectEqual(t, SizeCellCountInt(2), rc.x)
	expect(t, !rc.pending_wrap, "!rc.pending_wrap")
	expectEqual(t, uint(7), rc.total_rows)
	expectEqual(t, uint(4), rc.new_rows)
	expectEqual(t, SizeCellCountInt(1), rc.page.size.rows)
	expectEqual(t, SizeCellCountInt(1), node.data.size.rows)
	expectEqual(t, uint(3), s.totalRows())

	// The new row has all the data.
	new_page := &rc.node.data
	row := new_page.GetRowAndCell(0, 0).row
	expect(t, row.wrap, "row.wrap")
	expect(t, row.grapheme, "row.grapheme")
	expect(t, row.styled, "row.styled")
	expectEqual(t, PageRowSemanticPromptPrompt, row.semantic_prompt)
	{
		c := new_page.GetRowAndCell(0, 0).cell
		expectEqual(t, uint32('c'), c.content.codepoint)
		expectEqualSlices(t, []uint32{0x301}, zigUnwrap(t, new_page.LookupGrapheme(c)))
		d := new_page.GetRowAndCell(1, 0).cell
		expectEqual(t, uint32('d'), d.content.codepoint)
		expect(t, new_page.styles.Get(d.style_id).Flags.Bold, "new_page.styles.get(new_page.memory, d.style_id).flags.bold")
	}

	// The old row is cleared, including its managed memory.
	{
		// (Beyond the page size, so not reachable with getRowAndCell.)
		old_row := &page.rows[1]
		expect(t, !old_row.grapheme, "!old_row.grapheme")
		expect(t, !old_row.styled, "!old_row.styled")
		for _, c := range page.GetCells(old_row)[0:4] {
			expectEqual(t, uint64(0), c.bitCast())
		}
		expectEqual(t, uint(0), page.GraphemeCount())
		expectEqual(t, uint(0), page.styles.living)
		expectEqual(t, uint(1), new_page.styles.living)
	}
	// The row above is untouched.
	expectEqual(t, uint32('a'), page.GetRowAndCell(0, 0).cell.content.codepoint)

	// The pin in the moved row follows, the other one stays.
	expect(t, rc.node == pin_row1.Node, "rc.node == pin_row1.node")
	expectEqual(t, SizeCellCountInt(0), pin_row1.Y)
	expectEqual(t, SizeCellCountInt(1), pin_row1.X)
	expect(t, node == pin_row0.Node, "node == pin_row0.node")
	expectEqual(t, SizeCellCountInt(0), pin_row0.Y)
	s.AssertIntegrity()
}

// Zig: test "term51: ReflowCursor.moveLastRowToNewPage: a page with a single row is removed"
func TestPl3_term51_ReflowCursor_moveLastRowToNewPage_a_page_with_a_single_row_is_removed(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	node := pl3KAddNode(t, s, pl3Cap(4, 3), 1)
	*node.data.GetRowAndCell(1, 0).cell = PageCellInit('x')
	tpin := pl3Track(t, s, PageListPin{Node: node, X: 1, Y: 0})
	defer s.UntrackPin(tpin)
	expectEqual(t, uint(2), s.TotalPages())

	rc := pageListReflowCursorInit(node)
	rc.cursorAbsolute(1, 0)
	zigTry(t, rc.moveLastRowToNewPage(s, pl3Cap(4, 3)))

	// The old node is gone and replaced by the new one, in the same position.
	expectEqual(t, uint(2), s.TotalPages())
	expect(t, rc.node == s.pages.last, "rc.node == s.pages.last.?")
	expectEqual(t, uint(2), s.totalRows())
	expectEqual(t, SizeCellCountInt(0), rc.y)
	expectEqual(t, SizeCellCountInt(1), rc.x)
	expect(t, rc.node == tpin.Node, "rc.node == tpin.node")
	expectEqual(t, uint32('x'), rc.page.GetRowAndCell(1, 0).cell.content.codepoint)
	s.AssertIntegrity()
}

// Zig: test "term51: ReflowCursor.moveLastRowToNewPage: layout page byte accounting"
func TestPl3_term51_ReflowCursor_moveLastRowToNewPage_layout_page_byte_accounting(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	node := pl3KAddNode(t, s, pl3Cap(4, 3), 1)
	old_len := pageListMemoryLen(&node.data)
	bytes_before := s.page_size

	rc := pageListReflowCursorInit(node)
	zigTry(t, rc.moveLastRowToNewPage(s, pl3Cap(4, 3)))
	// A pooled page adds the whole pool buffer to the accounting but removing
	// a page subtracts only the layout size of that page, so the total grows
	// by the difference every time a smaller than standard page is replaced.
	expect(t, old_len < pageListStdSize(), "old_len < std_size")
	expectEqual(t, bytes_before+pageListStdSize()-old_len, s.page_size)
	s.AssertIntegrity()
}

// Zig: test "term51: ReflowCursor.increaseCapacity: replaces the node keeping position and counts"
func TestPl3_term51_ReflowCursor_increaseCapacity_replaces_the_node_keeping_position_and_counts(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	node := pl3KAddNode(t, s, pl3Cap(4, 3), 2)
	*node.data.GetRowAndCell(0, 0).cell = PageCellInit('a')
	*node.data.GetRowAndCell(2, 1).cell = PageCellInit('b')
	tpin := pl3Track(t, s, PageListPin{Node: node, X: 2, Y: 1})
	defer s.UntrackPin(tpin)

	rc := pageListReflowCursorInit(node)
	rc.cursorAbsolute(2, 1)
	rc.total_rows = 6
	rc.new_rows = 4
	rc.pending_wrap = true
	old_cap := node.data.capacity

	zigTry(t, rc.increaseCapacity(s, Some(PageListIncreaseCapacityStyles)))
	expect(t, rc.node != node, "rc.node != node")
	expectEqual(t, old_cap.styles*2, rc.page.capacity.styles)
	expectEqual(t, SizeCellCountInt(2), rc.x)
	expectEqual(t, SizeCellCountInt(1), rc.y)
	expectEqual(t, uint(6), rc.total_rows)
	// The cursor is reinitialized: deferred rows and a pending wrap are lost.
	expectEqual(t, uint(0), rc.new_rows)
	expect(t, !rc.pending_wrap, "!rc.pending_wrap")
	expectEqual(t, uint32('b'), rc.page_cell.content.codepoint)
	expect(t, rc.node == tpin.Node, "rc.node == tpin.node")
	expectEqual(t, uint(2), s.TotalPages())
	expect(t, rc.node == s.pages.last, "rc.node == s.pages.last.?")
	if pageSlowRuntimeSafety {
		expectEqual(t, uint64(0), s.pause_integrity_checks)
	}

	zigTry(t, rc.increaseCapacity(s, Some(PageListIncreaseCapacityGraphemeBytes)))
	expectEqual(t, old_cap.grapheme_bytes*2, rc.page.capacity.grapheme_bytes)
	zigTry(t, rc.increaseCapacity(s, Some(PageListIncreaseCapacityHyperlinkBytes)))
	expectEqual(t, old_cap.hyperlink_bytes*2, rc.page.capacity.hyperlink_bytes)
	zigTry(t, rc.increaseCapacity(s, Some(PageListIncreaseCapacityStringBytes)))
	expectEqual(t, old_cap.string_bytes*2, rc.page.capacity.string_bytes)

	// A null adjustment recreates the page with the same capacity.
	before := rc.node
	cap := rc.page.capacity
	zigTry(t, rc.increaseCapacity(s, None[PageListIncreaseCapacity]()))
	expect(t, rc.node != before, "rc.node != before")
	expectEqual(t, cap, rc.page.capacity)
	expectEqual(t, uint32('a'), rc.page.GetRowAndCell(0, 0).cell.content.codepoint)
	expectEqual(t, SizeCellCountInt(2), rc.x)
	s.AssertIntegrity()
}

// Zig: test "term51: ReflowCursor.increaseCapacity: reports out of space at the maximum capacity"
func TestPl3_term51_ReflowCursor_increaseCapacity_reports_out_of_space_at_the_maximum_capacity(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	cap := pl3Cap(4, 2)
	cap.styles = math.MaxUint16
	node := pl3KAddNode(t, s, cap, 1)

	rc := pageListReflowCursorInit(node)
	rc.total_rows = 3
	pl3ExpectError(t, ErrOutOfSpace, rc.increaseCapacity(s, Some(PageListIncreaseCapacityStyles)))
	// Nothing changed.
	expect(t, node == rc.node, "node == rc.node")
	expectEqual(t, uint(3), rc.total_rows)
	if pageSlowRuntimeSafety {
		expectEqual(t, uint64(0), s.pause_integrity_checks)
	}
	s.AssertIntegrity()
}

// ---------------------------------------------------------------------------
// ReflowCursor.writeCell
// ---------------------------------------------------------------------------

// Zig: test "term51: ReflowCursor.writeCell: narrow cells are copied whole and the cursor advances"
func TestPl3_term51_ReflowCursor_writeCell_narrow_cells_are_copied_whole_and_the_cursor_advances(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	node := pl3KAddNode(t, s, pl3Cap(4, 2), 1)
	rc := pageListReflowCursorInit(node)
	src_page := &s.pages.first.data

	cell := PageCell{
		content_tag:      PageCellContentTagCodepoint,
		content:          PageCellContent{codepoint: 'x'},
		protected:        true,
		semantic_content: PageCellSemanticContentInput,
	}
	r, err := rc.writeCell(s, &cell, src_page)
	zigTry(t, err)
	expect(t, r == pageListReflowCursorWriteCellResultSuccess, "r == .success")
	expectEqual(t, cell.bitCast(), pl3KDstCell(&rc, 0).bitCast())
	expectEqual(t, SizeCellCountInt(1), rc.x)
	expect(t, !rc.pending_wrap, "!rc.pending_wrap")

	// An empty cell is copied as well.
	r2, err := rc.writeCell(s, &PageCell{}, src_page)
	zigTry(t, err)
	expect(t, r2 == pageListReflowCursorWriteCellResultSuccess, "r2 == .success")
	expectEqual(t, uint64(0), pl3KDstCell(&rc, 1).bitCast())
	expectEqual(t, SizeCellCountInt(2), rc.x)

	// Writing the last column leaves a pending wrap on that column.
	_, err = rc.writeCell(s, &cell, src_page)
	zigTry(t, err)
	_, err = rc.writeCell(s, &cell, src_page)
	zigTry(t, err)
	expectEqual(t, SizeCellCountInt(3), rc.x)
	expect(t, rc.pending_wrap, "rc.pending_wrap")
}

// Zig: test "term51: ReflowCursor.writeCell: wide cell in the last column becomes a spacer head and repeats"
func TestPl3_term51_ReflowCursor_writeCell_wide_cell_in_the_last_column_becomes_a_spacer_head_and_repeats(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	node := pl3KAddNode(t, s, pl3Cap(3, 2), 1)
	rc := pageListReflowCursorInit(node)
	src_page := &s.pages.first.data

	// Not the last column: copied as is.
	wide := pl3KWide()
	r, err := rc.writeCell(s, &wide, src_page)
	zigTry(t, err)
	expect(t, r == pageListReflowCursorWriteCellResultSuccess, "r == .success")
	expectEqual(t, wide.bitCast(), pl3KDstCell(&rc, 0).bitCast())
	expectEqual(t, SizeCellCountInt(1), rc.x)

	// The spacer tail follows.
	tail := pl3KSpacerTail()
	r, err = rc.writeCell(s, &tail, src_page)
	zigTry(t, err)
	expect(t, r == pageListReflowCursorWriteCellResultSuccess, "r == .success")
	expectEqual(t, tail.bitCast(), pl3KDstCell(&rc, 1).bitCast())
	expectEqual(t, SizeCellCountInt(2), rc.x)

	// Now on the last column.
	r, err = rc.writeCell(s, &wide, src_page)
	zigTry(t, err)
	expect(t, r == pageListReflowCursorWriteCellResultRepeat, "r == .repeat")
	head := pl3KDstCell(&rc, 2)
	expectEqual(t, PageCellWideSpacerHead, head.wide)
	expectEqual(t, uint32(0), head.content.codepoint)
	expectEqual(t, SizeCellCountInt(2), rc.x)
	expect(t, rc.pending_wrap, "rc.pending_wrap")
}

// Zig: test "term51: ReflowCursor.writeCell: wide cell in a one column page is destroyed"
func TestPl3_term51_ReflowCursor_writeCell_wide_cell_in_a_one_column_page_is_destroyed(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	node := pl3KAddNode(t, s, pl3Cap(1, 2), 1)
	rc := pageListReflowCursorInit(node)
	src_page := &s.pages.first.data

	// Pre-existing garbage in the destination: only the codepoint and wide
	// flags are reset.
	*rc.page_cell = PageCell{content_tag: PageCellContentTagCodepoint, content: PageCellContent{codepoint: 'q'}, wide: PageCellWideWide}
	wide := pl3KWide()
	r, err := rc.writeCell(s, &wide, src_page)
	zigTry(t, err)
	expect(t, r == pageListReflowCursorWriteCellResultSkipNext, "r == .skip_next")
	dst := pl3KDstCell(&rc, 0)
	expectEqual(t, uint32(0), dst.content.codepoint)
	expectEqual(t, PageCellWideNarrow, dst.wide)
	expect(t, rc.pending_wrap, "rc.pending_wrap")
	expectEqual(t, SizeCellCountInt(0), rc.x)
}

// Zig: test "term51: ReflowCursor.writeCell: spacer tails are dropped in a one column page"
func TestPl3_term51_ReflowCursor_writeCell_spacer_tails_are_dropped_in_a_one_column_page(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	node := pl3KAddNode(t, s, pl3Cap(1, 2), 1)
	rc := pageListReflowCursorInit(node)
	src_page := &s.pages.first.data

	tail := pl3KSpacerTail()
	r, err := rc.writeCell(s, &tail, src_page)
	zigTry(t, err)
	expect(t, r == pageListReflowCursorWriteCellResultSuccess, "r == .success")
	expectEqual(t, uint64(0), pl3KDstCell(&rc, 0).bitCast())
	// Nothing was written and the cursor did not move.
	expect(t, !rc.pending_wrap, "!rc.pending_wrap")
	expectEqual(t, SizeCellCountInt(0), rc.x)
}

// Zig: test "term51: ReflowCursor.writeCell: spacer heads are ignored"
func TestPl3_term51_ReflowCursor_writeCell_spacer_heads_are_ignored(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	src_page := &s.pages.first.data
	head := pl3KSpacerHead()

	for _, cols := range []SizeCellCountInt{1, 3} {
		node := pl3KAddNode(t, s, pl3Cap(cols, 2), 1)
		rc := pageListReflowCursorInit(node)
		r, err := rc.writeCell(s, &head, src_page)
		zigTry(t, err)
		expect(t, r == pageListReflowCursorWriteCellResultSuccess, "r == .success")
		expectEqual(t, uint64(0), pl3KDstCell(&rc, 0).bitCast())
		expectEqual(t, SizeCellCountInt(0), rc.x)
		expect(t, !rc.pending_wrap, "!rc.pending_wrap")
	}
}

// Zig: test "term51: ReflowCursor.writeCell: background color cells are copied without styles"
func TestPl3_term51_ReflowCursor_writeCell_background_color_cells_are_copied_without_styles(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	node := pl3KAddNode(t, s, pl3Cap(2, 2), 1)
	rc := pageListReflowCursorInit(node)
	src_page := &s.pages.first.data

	pal := PageCell{
		content_tag:      PageCellContentTagBgColorPalette,
		content:          PageCellContent{color_palette: 9},
		protected:        true,
		semantic_content: PageCellSemanticContentPrompt,
	}
	r, err := rc.writeCell(s, &pal, src_page)
	zigTry(t, err)
	expect(t, r == pageListReflowCursorWriteCellResultSuccess, "r == .success")
	expectEqual(t, pal.bitCast(), pl3KDstCell(&rc, 0).bitCast())
	expectEqual(t, SizeCellCountInt(1), rc.x)

	rgb := PageCell{
		content_tag: PageCellContentTagBgColorRgb,
		content:     PageCellContent{color_rgb: PageCellRGB{R: 1, G: 2, B: 3}},
	}
	r, err = rc.writeCell(s, &rgb, src_page)
	zigTry(t, err)
	expect(t, r == pageListReflowCursorWriteCellResultSuccess, "r == .success")
	expectEqual(t, rgb.bitCast(), pl3KDstCell(&rc, 1).bitCast())
	expect(t, rc.pending_wrap, "rc.pending_wrap")
	expect(t, !rc.page_row.styled, "!rc.page_row.styled")
}

// Zig: test "term51: ReflowCursor.writeCell: graphemes are copied"
func TestPl3_term51_ReflowCursor_writeCell_graphemes_are_copied(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	node := pl3KAddNode(t, s, pl3Cap(4, 2), 1)
	rc := pageListReflowCursorInit(node)
	src_page := &s.pages.first.data

	src := src_page.GetRowAndCell(0, 0)
	*src.cell = PageCellInit('e')
	zigTry(t, src_page.AppendGrapheme(src.row, src.cell, 0x301))
	zigTry(t, src_page.AppendGrapheme(src.row, src.cell, 0x302))

	r, err := rc.writeCell(s, src.cell, src_page)
	zigTry(t, err)
	expect(t, r == pageListReflowCursorWriteCellResultSuccess, "r == .success")
	dst := rc.page.GetRowAndCell(0, 0)
	expectEqual(t, PageCellContentTagCodepointGrapheme, dst.cell.content_tag)
	expectEqual(t, uint32('e'), dst.cell.content.codepoint)
	expect(t, dst.row.grapheme, "dst.row.grapheme")
	expectEqualSlices(t, []uint32{0x301, 0x302}, zigUnwrap(t, rc.page.LookupGrapheme(dst.cell)))
	expectEqual(t, uint(1), rc.page.GraphemeCount())
	// The page is the same one: no capacity increase was needed.
	expect(t, node == rc.node, "node == rc.node")
	expectEqual(t, SizeCellCountInt(1), rc.x)
}

// Zig: test "term51: ReflowCursor.writeCell: a full grapheme map increases the capacity"
func TestPl3_term51_ReflowCursor_writeCell_a_full_grapheme_map_increases_the_capacity(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	src_page := &s.pages.first.data
	src := src_page.GetRowAndCell(0, 0)
	*src.cell = PageCellInit('e')
	zigTry(t, src_page.AppendGrapheme(src.row, src.cell, 0x301))

	// Fill the grapheme map of the destination page entirely.
	node := pl3KAddNode(t, s, pl3Cap(64, 2), 2)
	page := &node.data
	expectEqual(t, uint(64), page.GraphemeCapacity())
	for x := 0; x < 64; x++ {
		c := page.GetRowAndCell(uint(x), 0)
		*c.cell = PageCellInit('a')
		zigTry(t, page.AppendGrapheme(c.row, c.cell, 0x300))
	}
	expectEqual(t, uint(64), page.GraphemeCount())

	rc := pageListReflowCursorInit(node)
	rc.cursorAbsolute(5, 1)
	rc.total_rows = 2
	old_bytes := page.capacity.grapheme_bytes
	r, err := rc.writeCell(s, src.cell, src_page)
	zigTry(t, err)
	expect(t, r == pageListReflowCursorWriteCellResultSuccess, "r == .success")
	expect(t, rc.node != node, "rc.node != node")
	expectEqual(t, old_bytes*2, rc.page.capacity.grapheme_bytes)
	expectEqual(t, uint(65), rc.page.GraphemeCount())
	expectEqual(t, SizeCellCountInt(6), rc.x)
	expectEqual(t, SizeCellCountInt(1), rc.y)
	expectEqual(t, uint(2), rc.total_rows)
	dst := rc.page.GetRowAndCell(5, 1)
	expectEqualSlices(t, []uint32{0x301}, zigUnwrap(t, rc.page.LookupGrapheme(dst.cell)))
	// The old content survived the move.
	old := rc.page.GetRowAndCell(63, 0)
	expectEqualSlices(t, []uint32{0x300}, zigUnwrap(t, rc.page.LookupGrapheme(old.cell)))
	expect(t, rc.node == s.pages.last, "rc.node == s.pages.last.?")
	s.AssertIntegrity()
}

// Zig: test "term51: ReflowCursor.writeCell: exhausted grapheme bytes increase the capacity until they fit"
func TestPl3_term51_ReflowCursor_writeCell_exhausted_grapheme_bytes_increase_the_capacity_until_they_fit(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	src_page := &s.pages.first.data
	src := src_page.GetRowAndCell(0, 0)
	*src.cell = PageCellInit('e')
	zigTry(t, src_page.AppendGrapheme(src.row, src.cell, 0x301))

	// One grapheme that takes all the bytes of the destination page.
	node := pl3KAddNode(t, s, pl3Cap(4, 2), 1)
	page := &node.data
	{
		c := page.GetRowAndCell(3, 0)
		*c.cell = PageCellInit('X')
		cps := make([]uint32, PageGraphemeBytesDefault/4) // @sizeOf(u21) is 4
		for i := range cps {
			cps[i] = 'a'
		}
		zigTry(t, page.SetGraphemes(c.row, c.cell, cps))
		_, err := page.grapheme_alloc.Alloc(1)
		pl3ExpectError(t, ErrOutOfMemory, err)
	}
	expect(t, page.GraphemeCount() < page.GraphemeCapacity(), "page.graphemeCount() < page.graphemeCapacity()")

	rc := pageListReflowCursorInit(node)
	rc.cursorAbsolute(1, 0)
	old_bytes := page.capacity.grapheme_bytes
	r, err := rc.writeCell(s, src.cell, src_page)
	zigTry(t, err)
	expect(t, r == pageListReflowCursorWriteCellResultSuccess, "r == .success")
	expect(t, rc.node != node, "rc.node != node")
	expect(t, rc.page.capacity.grapheme_bytes > old_bytes, "rc.page.capacity.grapheme_bytes > old_bytes")
	expectEqual(t, uint(2), rc.page.GraphemeCount())
	expectEqual(t, SizeCellCountInt(2), rc.x)
	s.AssertIntegrity()
}

// Zig: test "term51: ReflowCursor.writeCell: hyperlinks are copied with their strings"
func TestPl3_term51_ReflowCursor_writeCell_hyperlinks_are_copied_with_their_strings(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	src_page := &s.pages.first.data

	// An implicit and an explicit link.
	id1, err := src_page.InsertHyperlink(pl3Link(7, "https://a.example"))
	zigTry(t, err)
	id2, err := src_page.InsertHyperlink(Hyperlink{
		ID:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdExplicit, Explicit: []uint8("my-id")},
		URI: []uint8("https://b.example"),
	})
	zigTry(t, err)
	c1 := src_page.GetRowAndCell(0, 0)
	*c1.cell = PageCellInit('x')
	zigTry(t, src_page.SetHyperlink(c1.row, c1.cell, id1))
	c2 := src_page.GetRowAndCell(1, 0)
	*c2.cell = PageCellInit('y')
	zigTry(t, src_page.SetHyperlink(c2.row, c2.cell, id2))
	c3 := src_page.GetRowAndCell(2, 0)
	*c3.cell = PageCellInit('z')
	src_page.hyperlink_set.Use(id1)
	zigTry(t, src_page.SetHyperlink(c3.row, c3.cell, id1))

	node := pl3KAddNode(t, s, pl3Cap(4, 2), 1)
	rc := pageListReflowCursorInit(node)
	for _, c := range []*PageCell{c1.cell, c2.cell, c3.cell} {
		r, err := rc.writeCell(s, c, src_page)
		zigTry(t, err)
		expect(t, r == pageListReflowCursorWriteCellResultSuccess, "r == .success")
	}
	expect(t, node == rc.node, "node == rc.node")
	page := rc.page
	row := page.GetRowAndCell(0, 0).row
	expect(t, row.hyperlink, "row.hyperlink")
	expectEqual(t, uint(3), page.HyperlinkCount())

	d1 := page.GetRowAndCell(0, 0).cell
	d2 := page.GetRowAndCell(1, 0).cell
	d3 := page.GetRowAndCell(2, 0).cell
	expect(t, d1.hyperlink && d2.hyperlink && d3.hyperlink, "d1.hyperlink and d2.hyperlink and d3.hyperlink")
	l1 := zigUnwrap(t, page.LookupHyperlink(d1))
	l2 := zigUnwrap(t, page.LookupHyperlink(d2))
	l3 := zigUnwrap(t, page.LookupHyperlink(d3))
	// The same source link is the same link (and ID) in the destination.
	expectEqual(t, l1, l3)
	expect(t, l1 != l2, "l1 != l2")
	expectEqual(t, uint16(2), page.hyperlink_set.RefCount(l1))
	expectEqual(t, uint16(1), page.hyperlink_set.RefCount(l2))
	e1 := page.hyperlink_set.Get(l1)
	expectEqualStrings(t, "https://a.example", string(e1.uri))
	expectEqual(t, SizeOffsetInt(7), e1.id.Implicit)
	e2 := page.hyperlink_set.Get(l2)
	expectEqualStrings(t, "https://b.example", string(e2.uri))
	expectEqualStrings(t, "my-id", string(e2.id.Explicit))
	page.AssertIntegrity()
}

// Zig: test "term51: ReflowCursor.writeCell: a full hyperlink map increases the capacity"
func TestPl3_term51_ReflowCursor_writeCell_a_full_hyperlink_map_increases_the_capacity(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	src_page := &s.pages.first.data
	sid, err := src_page.InsertHyperlink(pl3Link(1, "https://src.example"))
	zigTry(t, err)
	src := src_page.GetRowAndCell(0, 0)
	*src.cell = PageCellInit('s')
	zigTry(t, src_page.SetHyperlink(src.row, src.cell, sid))

	node := pl3KAddNode(t, s, pl3Cap(64, 2), 2)
	page := &node.data
	expectEqual(t, uint(64), page.HyperlinkCapacity())
	did, err := page.InsertHyperlink(pl3Link(0, "d"))
	zigTry(t, err)
	for x := 0; x < 64; x++ {
		c := page.GetRowAndCell(uint(x), 0)
		*c.cell = PageCellInit('a')
		if x > 0 {
			page.hyperlink_set.Use(did)
		}
		zigTry(t, page.SetHyperlink(c.row, c.cell, did))
	}
	expectEqual(t, uint(64), page.HyperlinkCount())

	rc := pageListReflowCursorInit(node)
	rc.cursorAbsolute(2, 1)
	old_bytes := page.capacity.hyperlink_bytes
	r, err := rc.writeCell(s, src.cell, src_page)
	zigTry(t, err)
	expect(t, r == pageListReflowCursorWriteCellResultSuccess, "r == .success")
	expect(t, rc.node != node, "rc.node != node")
	expectEqual(t, old_bytes*2, rc.page.capacity.hyperlink_bytes)
	expectEqual(t, uint(65), rc.page.HyperlinkCount())
	expectEqual(t, SizeCellCountInt(3), rc.x)
	dst := rc.page.GetRowAndCell(2, 1).cell
	expect(t, dst.hyperlink, "dst.hyperlink")
	e := rc.page.hyperlink_set.Get(zigUnwrap(t, rc.page.LookupHyperlink(dst)))
	expectEqualStrings(t, "https://src.example", string(e.uri))
	s.AssertIntegrity()
}

// Zig: test "term51: ReflowCursor.writeCell: exhausted string bytes increase the capacity"
func TestPl3_term51_ReflowCursor_writeCell_exhausted_string_bytes_increase_the_capacity(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	src_page := &s.pages.first.data
	sid, err := src_page.InsertHyperlink(pl3Link(1, "https://src.example"))
	zigTry(t, err)
	src := src_page.GetRowAndCell(0, 0)
	*src.cell = PageCellInit('s')
	zigTry(t, src_page.SetHyperlink(src.row, src.cell, sid))

	node := pl3KAddNode(t, s, pl3Cap(4, 2), 1)
	page := &node.data
	{
		did, err := page.InsertHyperlink(pl3Link(0, strings.Repeat("a", PageStringBytesDefault-1)))
		zigTry(t, err)
		c := page.GetRowAndCell(3, 0)
		*c.cell = PageCellInit('d')
		zigTry(t, page.SetHyperlink(c.row, c.cell, did))
		_, err = page.InsertHyperlink(pl3Link(5, "https://src.example"))
		pl3ExpectError(t, ErrStringsOutOfMemory, err)
	}

	rc := pageListReflowCursorInit(node)
	old_bytes := page.capacity.string_bytes
	r, err := rc.writeCell(s, src.cell, src_page)
	zigTry(t, err)
	expect(t, r == pageListReflowCursorWriteCellResultSuccess, "r == .success")
	expect(t, rc.node != node, "rc.node != node")
	expect(t, rc.page.capacity.string_bytes > old_bytes, "rc.page.capacity.string_bytes > old_bytes")
	expectEqual(t, uint(2), rc.page.HyperlinkCount())
	dst := rc.page.GetRowAndCell(0, 0).cell
	e := rc.page.hyperlink_set.Get(zigUnwrap(t, rc.page.LookupHyperlink(dst)))
	expectEqualStrings(t, "https://src.example", string(e.uri))
	s.AssertIntegrity()
}

// Zig: test "term51: ReflowCursor.writeCell: styles are copied and shared"
func TestPl3_term51_ReflowCursor_writeCell_styles_are_copied_and_shared(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	src_page := &s.pages.first.data
	bold, err := src_page.styles.Add(pl3Bold())
	zigTry(t, err)
	ital, err := src_page.styles.Add(Style{Flags: StyleFlags{Italic: true}})
	zigTry(t, err)
	sc := [3]PageCell{
		pl3KStyleCell('a', bold),
		pl3KStyleCell('b', ital),
		pl3KStyleCell('c', bold),
	}
	src_page.styles.Use(bold)

	// The destination already has an unrelated style with the id of the
	// first source style.
	node := pl3KAddNode(t, s, pl3Cap(4, 2), 1)
	page := &node.data
	other, err := page.styles.Add(Style{Flags: StyleFlags{Underline: SGRAttributeUnderlineSingle}})
	zigTry(t, err)
	expectEqual(t, bold, other)
	*page.GetRowAndCell(3, 0).cell = pl3KStyleCell('o', other)
	page.GetRowAndCell(3, 0).row.styled = true

	rc := pageListReflowCursorInit(node)
	for i := range sc {
		r, err := rc.writeCell(s, &sc[i], src_page)
		zigTry(t, err)
		expect(t, r == pageListReflowCursorWriteCellResultSuccess, "r == .success")
	}
	expect(t, rc.page_row.styled, "rc.page_row.styled")
	d0 := pl3KDstCell(&rc, 0)
	d1 := pl3KDstCell(&rc, 1)
	d2 := pl3KDstCell(&rc, 2)
	// The wanted id was taken so the style got another id.
	expect(t, d0.style_id != other, "d0.style_id != other")
	expect(t, d0.style_id != 0, "d0.style_id != 0")
	expect(t, page.styles.Get(d0.style_id).Flags.Bold, "page.styles.get(page.memory, d0.style_id).flags.bold")
	expect(t, page.styles.Get(d1.style_id).Flags.Italic, "page.styles.get(page.memory, d1.style_id).flags.italic")
	expectEqual(t, d0.style_id, d2.style_id)
	expectEqual(t, uint16(2), page.styles.RefCount(d0.style_id))
	expectEqual(t, uint16(1), page.styles.RefCount(d1.style_id))
	page.AssertIntegrity()
}

// Zig: test "term51: ReflowCursor.writeCell: styles get fresh ids in an empty destination"
func TestPl3_term51_ReflowCursor_writeCell_styles_get_fresh_ids_in_an_empty_destination(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	src_page := &s.pages.first.data
	bold, err := src_page.styles.Add(pl3Bold())
	zigTry(t, err)
	ital, err := src_page.styles.Add(Style{Flags: StyleFlags{Italic: true}})
	zigTry(t, err)
	cell := pl3KStyleCell('b', ital)
	expect(t, bold != ital, "bold != ital")

	node := pl3KAddNode(t, s, pl3Cap(4, 2), 1)
	rc := pageListReflowCursorInit(node)
	r, err := rc.writeCell(s, &cell, src_page)
	zigTry(t, err)
	expect(t, r == pageListReflowCursorWriteCellResultSuccess, "r == .success")
	// The wanted id (2) is beyond the ids in use in the destination, so
	// the first free one is used.
	expectEqual(t, SizeStyleCountInt(2), ital)
	expectEqual(t, SizeStyleCountInt(1), pl3KDstCell(&rc, 0).style_id)
	expect(t, rc.page_row.styled, "rc.page_row.styled")
	expectEqual(t, uint16(1), rc.page.styles.RefCount(1))
	rc.page.AssertIntegrity()
}

// Zig: test "term51: ReflowCursor.writeCell: styles reuse their id when its slot is dead"
func TestPl3_term51_ReflowCursor_writeCell_styles_reuse_their_id_when_its_slot_is_dead(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	src_page := &s.pages.first.data
	bold, err := src_page.styles.Add(pl3Bold())
	zigTry(t, err)
	ital, err := src_page.styles.Add(Style{Flags: StyleFlags{Italic: true}})
	zigTry(t, err)
	cell := pl3KStyleCell('b', ital)

	// The destination has 2 styles of which the first one is dead and the
	// last is alive (so nothing is trimmed from the end).
	node := pl3KAddNode(t, s, pl3Cap(4, 2), 1)
	page := &node.data
	dead, err := page.styles.Add(Style{Flags: StyleFlags{Underline: SGRAttributeUnderlineSingle}})
	zigTry(t, err)
	alive, err := page.styles.Add(Style{Flags: StyleFlags{Faint: true}})
	zigTry(t, err)
	page.styles.Release(dead)
	*page.GetRowAndCell(3, 0).cell = pl3KStyleCell('o', alive)
	page.GetRowAndCell(3, 0).row.styled = true
	expectEqual(t, SizeStyleCountInt(1), dead)
	expectEqual(t, SizeStyleCountInt(2), alive)
	expectEqual(t, SizeStyleCountInt(2), ital)
	expectEqual(t, SizeStyleCountInt(1), bold)

	// Ids below the next free id are taken over when their slot is dead: bold
	// (1) replaces the dead style.
	rc := pageListReflowCursorInit(node)
	bcell := pl3KStyleCell('a', bold)
	_, err = rc.writeCell(s, &bcell, src_page)
	zigTry(t, err)
	expectEqual(t, bold, pl3KDstCell(&rc, 0).style_id)
	expect(t, page.styles.Get(bold).Flags.Bold, "page.styles.get(page.memory, bold).flags.bold")
	// ital (2) is alive with another style, so it gets the next id (3).
	r, err := rc.writeCell(s, &cell, src_page)
	zigTry(t, err)
	expect(t, r == pageListReflowCursorWriteCellResultSuccess, "r == .success")
	expectEqual(t, SizeStyleCountInt(3), pl3KDstCell(&rc, 1).style_id)
	page.AssertIntegrity()
}

// The Zig test "term51: ReflowCursor.writeCell: kitty placeholders mark the
// destination row" is not translated (Kitty graphics is compiled out of the
// library build, the test is skipped upstream); see
// scripts/not-translated.txt.

// Zig: test "term51: ReflowCursor.writeCell: a full style set increases the style capacity"
func TestPl3_term51_ReflowCursor_writeCell_a_full_style_set_increases_the_style_capacity(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	src_page := &s.pages.first.data
	sid, err := src_page.styles.Add(pl3Bold())
	zigTry(t, err)
	cell := pl3KStyleCell('s', sid)

	cap := pl3Cap(200, 2)
	cap.styles = 128
	node := pl3KAddNode(t, s, cap, 2)
	page := &node.data
	n := pl3KFillStyles(page)
	expect(t, n > 0 && n < 200, "n > 0 and n < 200")
	_, err = page.styles.Add(Style{Flags: StyleFlags{Italic: true}})
	pl3ExpectError(t, ErrOutOfMemory, err)

	rc := pageListReflowCursorInit(node)
	rc.cursorAbsolute(3, 1)
	r, err := rc.writeCell(s, &cell, src_page)
	zigTry(t, err)
	expect(t, r == pageListReflowCursorWriteCellResultSuccess, "r == .success")
	expect(t, rc.node != node, "rc.node != node")
	expectEqual(t, SizeStyleCountInt(256), rc.page.capacity.styles)
	expectEqual(t, SizeCellCountInt(4), rc.x)
	dst := rc.page.GetRowAndCell(3, 1)
	expect(t, dst.row.styled, "dst.row.styled")
	expect(t, rc.page.styles.Get(dst.cell.style_id).Flags.Bold, "rc.page.styles.get(rc.page.memory, dst.cell.style_id).flags.bold")
	expectEqual(t, n+1, rc.page.styles.living)
	s.AssertIntegrity()
}

// Zig: test "term51: ReflowCursor.writeCell: a style set that needs a rehash is recreated with the same capacity"
func TestPl3_term51_ReflowCursor_writeCell_a_style_set_that_needs_a_rehash_is_recreated_with_the_same_capacity(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	src_page := &s.pages.first.data
	sid, err := src_page.styles.Add(pl3Bold())
	zigTry(t, err)
	cell := pl3KStyleCell('s', sid)

	cap := pl3Cap(200, 2)
	cap.styles = 128
	node := pl3KAddNode(t, s, cap, 2)
	page := &node.data
	n := pl3KFillStyles(page)
	// Kill all the styles but the first and last: they stay as dead items in
	// the middle of the set so they are not trimmed.
	for x := 1; x < int(n)-1; x++ {
		rac := page.GetRowAndCell(uint(x), 0)
		page.styles.Release(rac.cell.style_id)
		rac.cell.style_id = 0
	}
	_, err = page.styles.Add(Style{Flags: StyleFlags{Italic: true}})
	pl3ExpectError(t, ErrNeedsRehash, err)

	rc := pageListReflowCursorInit(node)
	rc.cursorAbsolute(3, 1)
	r, err := rc.writeCell(s, &cell, src_page)
	zigTry(t, err)
	expect(t, r == pageListReflowCursorWriteCellResultSuccess, "r == .success")
	expect(t, rc.node != node, "rc.node != node")
	// Same capacity: the page was only rebuilt.
	expectEqual(t, SizeStyleCountInt(128), rc.page.capacity.styles)
	expectEqual(t, uint(3), rc.page.styles.living)
	dst := rc.page.GetRowAndCell(3, 1)
	expect(t, rc.page.styles.Get(dst.cell.style_id).Flags.Bold, "rc.page.styles.get(rc.page.memory, dst.cell.style_id).flags.bold")
	s.AssertIntegrity()
}

// Zig: test "term51: ReflowCursor.writeCell: a full hyperlink set increases the hyperlink capacity"
func TestPl3_term51_ReflowCursor_writeCell_a_full_hyperlink_set_increases_the_hyperlink_capacity(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	src_page := &s.pages.first.data
	sid, err := src_page.InsertHyperlink(pl3Link(99, "https://src.example"))
	zigTry(t, err)
	src := src_page.GetRowAndCell(0, 0)
	*src.cell = PageCellInit('s')
	zigTry(t, src_page.SetHyperlink(src.row, src.cell, sid))

	cap := pl3Cap(32, 2)
	cap.string_bytes = 4096
	node := pl3KAddNode(t, s, cap, 2)
	page := &node.data
	n := pl3KFillLinks(t, page)
	expect(t, n > 0 && n < 32, "n > 0 and n < 32")
	_, err = page.InsertHyperlink(pl3Link(98, "v"))
	pl3ExpectError(t, ErrSetOutOfMemory, err)
	expect(t, page.HyperlinkCount() < page.HyperlinkCapacity(), "page.hyperlinkCount() < page.hyperlinkCapacity()")

	rc := pageListReflowCursorInit(node)
	rc.cursorAbsolute(1, 1)
	old_bytes := page.capacity.hyperlink_bytes
	r, err := rc.writeCell(s, src.cell, src_page)
	zigTry(t, err)
	expect(t, r == pageListReflowCursorWriteCellResultSuccess, "r == .success")
	expect(t, rc.node != node, "rc.node != node")
	expectEqual(t, old_bytes*2, rc.page.capacity.hyperlink_bytes)
	expectEqual(t, n+1, rc.page.HyperlinkCount())
	dst := rc.page.GetRowAndCell(1, 1).cell
	e := rc.page.hyperlink_set.Get(zigUnwrap(t, rc.page.LookupHyperlink(dst)))
	expectEqualStrings(t, "https://src.example", string(e.uri))
	expectEqual(t, SizeOffsetInt(99), e.id.Implicit)
	s.AssertIntegrity()
}

// Zig: test "term51: ReflowCursor.writeCell: a hyperlink set that needs a rehash is recreated with the same capacity"
func TestPl3_term51_ReflowCursor_writeCell_a_hyperlink_set_that_needs_a_rehash_is_recreated_with_the_same_capacity(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	src_page := &s.pages.first.data
	sid, err := src_page.InsertHyperlink(pl3Link(99, "https://src.example"))
	zigTry(t, err)
	src := src_page.GetRowAndCell(0, 0)
	*src.cell = PageCellInit('s')
	zigTry(t, src_page.SetHyperlink(src.row, src.cell, sid))

	base := PageCapacityInit(1, 1)
	cap := pl3Cap(64, 2)
	cap.string_bytes = 4096
	cap.hyperlink_bytes = base.hyperlink_bytes * 8
	node := pl3KAddNode(t, s, cap, 2)
	page := &node.data
	n := pl3KFillLinks(t, page)
	// Remove all links but the first and last: dead items in the middle.
	expect(t, n >= 3, "n >= 3")
	for x := 1; x < int(n)-1; x++ {
		rac := page.GetRowAndCell(uint(x), 0)
		page.ClearHyperlink(rac.cell)
	}
	_, err = page.InsertHyperlink(pl3Link(98, "v"))
	pl3ExpectError(t, ErrSetNeedsRehash, err)

	rc := pageListReflowCursorInit(node)
	rc.cursorAbsolute(1, 1)
	old_bytes := page.capacity.hyperlink_bytes
	r, err := rc.writeCell(s, src.cell, src_page)
	zigTry(t, err)
	expect(t, r == pageListReflowCursorWriteCellResultSuccess, "r == .success")
	expect(t, rc.node != node, "rc.node != node")
	expectEqual(t, old_bytes, rc.page.capacity.hyperlink_bytes)
	expectEqual(t, uint(3), rc.page.HyperlinkCount())
	dst := rc.page.GetRowAndCell(1, 1).cell
	e := rc.page.hyperlink_set.Get(zigUnwrap(t, rc.page.LookupHyperlink(dst)))
	expectEqualStrings(t, "https://src.example", string(e.uri))
	s.AssertIntegrity()
}

// Zig: test "term51: ReflowCursor.writeCell: out of space is reported when the capacity cannot grow"
func TestPl3_term51_ReflowCursor_writeCell_out_of_space_is_reported_when_the_capacity_cannot_grow(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	src_page := &s.pages.first.data
	sid, err := src_page.styles.Add(pl3Bold())
	zigTry(t, err)
	cell := pl3KStyleCell('s', sid)

	// A page with the maximum number of styles that is full.
	const max = math.MaxUint16
	cap := pl3Cap(max, 2)
	cap.styles = max
	node := pl3KAddNode(t, s, cap, 2)
	page := &node.data
	var n uint = 0
	for x := 0; x < int(page.size.cols); x++ {
		id, err := page.styles.Add(Style{BgColor: StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{
			R: uint8(x),
			G: uint8(x >> 8),
			B: 1,
		}}})
		if err != nil {
			break
		}
		rac := page.GetRowAndCell(uint(x), 0)
		rac.row.styled = true
		*rac.cell = PageCell{
			content_tag: PageCellContentTagCodepoint,
			content:     PageCellContent{codepoint: 'X'},
			style_id:    id,
		}
		n += 1
	}
	expect(t, n < max, "n < max")

	rc := pageListReflowCursorInit(node)
	rc.cursorAbsolute(3, 1)
	_, err = rc.writeCell(s, &cell, src_page)
	pl3ExpectError(t, ErrOutOfSpace, err)
	// The cell is valid but has lost its style.
	expect(t, node == rc.node, "node == rc.node")
	dst := pl3KDstCell(&rc, 3)
	expectEqual(t, uint32('s'), dst.content.codepoint)
	expectEqual(t, SizeStyleCountInt(0), dst.style_id)
	expectEqual(t, SizeCellCountInt(3), rc.x)
	s.AssertIntegrity()
}

// ---------------------------------------------------------------------------
// ReflowCursor.reflowRow
// ---------------------------------------------------------------------------

// Zig: test "term51: ReflowCursor.reflowRow: trailing blanks of an unwrapped row are not written"
func TestPl3_term51_ReflowCursor_reflowRow_trailing_blanks_of_an_unwrapped_row_are_not_written(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 6, 2, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|ab....|\n|cd....|")

	r := pl3KReflowInit(t, s, 4, None[PageCapacity]())
	expect(t, r.step(t), "try r.step()")
	expectEqual(t, SizeCellCountInt(2), r.rc.x)
	expectEqual(t, SizeCellCountInt(0), r.rc.y)
	expect(t, !r.rc.pending_wrap, "!r.rc.pending_wrap")
	// The row ended, so a new row is owed.
	expectEqual(t, uint(1), r.rc.new_rows)
	expectEqual(t, uint(1), r.rc.total_rows)

	expect(t, r.step(t), "try r.step()")
	expectEqual(t, SizeCellCountInt(2), r.rc.x)
	expectEqual(t, SizeCellCountInt(1), r.rc.y)
	expectEqual(t, uint(1), r.rc.new_rows)
	expectEqual(t, uint(2), r.rc.total_rows)

	expect(t, !r.step(t), "!try r.step()")
	r.finish()
	pl3KExpect(t, s, "|ab..|\n|cd..|")
}

// Zig: test "term51: ReflowCursor.reflowRow: wrapped rows keep every cell and owe no new row"
func TestPl3_term51_ReflowCursor_reflowRow_wrapped_rows_keep_every_cell_and_owe_no_new_row(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 2, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|ab..>\n<c...|")

	r := pl3KReflowInit(t, s, 6, None[PageCapacity]())
	expect(t, r.step(t), "try r.step()")
	expectEqual(t, SizeCellCountInt(4), r.rc.x)
	expectEqual(t, uint(0), r.rc.new_rows)
	expect(t, r.step(t), "try r.step()")
	expectEqual(t, SizeCellCountInt(5), r.rc.x)
	expectEqual(t, uint(1), r.rc.new_rows)
	expectEqual(t, uint(1), r.rc.total_rows)
	r.finish()
	pl3KExpect(t, s, "|ab..c.|")
}

// Zig: test "term51: ReflowCursor.reflowRow: blank rows are deferred until text follows"
func TestPl3_term51_ReflowCursor_reflowRow_blank_rows_are_deferred_until_text_follows(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 4, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|a...|\n|....|\n|....|\n|b...|")

	r := pl3KReflowInit(t, s, 4, None[PageCapacity]())
	expect(t, r.step(t), "try r.step()")
	expectEqual(t, uint(1), r.rc.new_rows)
	expect(t, r.step(t), "try r.step()")
	expectEqual(t, uint(2), r.rc.new_rows)
	expectEqual(t, uint(1), r.rc.total_rows)
	expect(t, r.step(t), "try r.step()")
	expectEqual(t, uint(3), r.rc.new_rows)
	expectEqual(t, uint(1), r.rc.total_rows)
	expectEqual(t, SizeCellCountInt(0), r.rc.y)
	// The text flushes the 3 owed rows (one per source row so far).
	expect(t, r.step(t), "try r.step()")
	expectEqual(t, uint(1), r.rc.new_rows)
	expectEqual(t, uint(4), r.rc.total_rows)
	expectEqual(t, SizeCellCountInt(3), r.rc.y)
	r.finish()
	pl3KExpect(t, s, "|a...|\n|....|\n|....|\n|b...|")
}

// Zig: test "term51: ReflowCursor.reflowRow: trailing blank rows are never written"
func TestPl3_term51_ReflowCursor_reflowRow_trailing_blank_rows_are_never_written(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 3, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|a...|\n|....|\n|....|")

	r := pl3KReflowInit(t, s, 2, None[PageCapacity]())
	r.run(t)
	expectEqual(t, uint(3), r.rc.new_rows)
	expectEqual(t, uint(1), r.rc.total_rows)
	r.finish()
	pl3KExpect(t, s, "|a.|")
}

// Zig: test "term51: ReflowCursor.reflowRow: a blank wrap continuation row is dropped"
func TestPl3_term51_ReflowCursor_reflowRow_a_blank_wrap_continuation_row_is_dropped(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 3, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|a...>\n<....|\n|b...|")

	r := pl3KReflowInit(t, s, 4, None[PageCapacity]())
	expect(t, r.step(t), "try r.step()")
	expectEqual(t, uint(0), r.rc.new_rows)
	// Neither a new row nor a cell: the row simply disappears.
	expect(t, r.step(t), "try r.step()")
	expectEqual(t, uint(0), r.rc.new_rows)
	expectEqual(t, uint(1), r.rc.total_rows)
	r.run(t)
	r.finish()
	// The following row continues the line instead.
	pl3KExpect(t, s, "|a...>\n<b...|")
}

// Zig: test "term51: ReflowCursor.reflowRow: blank rows with a semantic prompt are written"
func TestPl3_term51_ReflowCursor_reflowRow_blank_rows_with_a_semantic_prompt_are_written(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 4, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|....|P\n|....|\n|....|C\n|....|")

	r := pl3KReflowInit(t, s, 2, None[PageCapacity]())
	r.run(t)
	r.finish()
	// One empty cell is written for each prompt row, the plain blank row
	// between is kept and the trailing one is dropped.
	pl3KExpect(t, s, "|..|P\n|..|\n|..|C")
}

// Zig: test "term51: ReflowCursor.reflowRow: a blank continuation row with a prompt is written as a continuation"
func TestPl3_term51_ReflowCursor_reflowRow_a_blank_continuation_row_with_a_prompt_is_written_as_a_continuation(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 2, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|a...>\n<....|P")

	r := pl3KReflowInit(t, s, 4, None[PageCapacity]())
	r.run(t)
	r.finish()
	// The prompt of the continuation row is also copied onto the row it
	// continues (copyRowMetadata runs before the wrap is processed).
	pl3KExpect(t, s, "|a...>P\n<....|P")
}

// Zig: test "term51: ReflowCursor.reflowRow: every wrapped line of a row gets its semantic prompt"
func TestPl3_term51_ReflowCursor_reflowRow_every_wrapped_line_of_a_row_gets_its_semantic_prompt(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 6, 3, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|abcdef|P\n|g.....|C\n|h.....|")

	r := pl3KReflowInit(t, s, 4, None[PageCapacity]())
	r.run(t)
	r.finish()
	pl3KExpect(t, s, "|abcd>P\n<ef..|P\n|g...|C\n|h...|")
}

// Zig: test "term51: ReflowCursor.reflowRow: pins on text move with it"
func TestPl3_term51_ReflowCursor_reflowRow_pins_on_text_move_with_it(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 6, 2, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|abcdef|\n|gh....|")
	p_b := pl3Track(t, s, pl3KPin(t, s, 1, 0))
	p_e := pl3Track(t, s, pl3KPin(t, s, 4, 0))
	p_f := pl3Track(t, s, pl3KPin(t, s, 5, 0))
	p_h := pl3Track(t, s, pl3KPin(t, s, 1, 1))
	defer s.UntrackPin(p_b)
	defer s.UntrackPin(p_e)
	defer s.UntrackPin(p_f)
	defer s.UntrackPin(p_h)

	r := pl3KReflowInit(t, s, 4, None[PageCapacity]())
	r.run(t)
	r.finish()
	pl3KExpect(t, s, "|abcd>\n<ef..|\n|gh..|")
	pl3KExpectPin(t, s, *p_b, 1, 0)
	pl3KExpectPin(t, s, *p_e, 0, 1)
	pl3KExpectPin(t, s, *p_f, 1, 1)
	pl3KExpectPin(t, s, *p_h, 1, 2)
	// The viewport pin was at the top left.
	pl3KExpectPin(t, s, *s.viewport_pin, 0, 0)
}

// Zig: test "term51: ReflowCursor.reflowRow: pins in the trailing blanks are clamped to the new width"
func TestPl3_term51_ReflowCursor_reflowRow_pins_in_the_trailing_blanks_are_clamped_to_the_new_width(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 6, 1, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|ab....|")
	// Beyond the dst width, inside it, and exactly after the text.
	p5 := pl3Track(t, s, pl3KPin(t, s, 5, 0))
	p3 := pl3Track(t, s, pl3KPin(t, s, 3, 0))
	p2 := pl3Track(t, s, pl3KPin(t, s, 2, 0))
	defer s.UntrackPin(p5)
	defer s.UntrackPin(p3)
	defer s.UntrackPin(p2)

	r := pl3KReflowInit(t, s, 4, None[PageCapacity]())
	r.run(t)
	r.finish()
	// The row is extended up to the furthest pin and does not wrap.
	pl3KExpect(t, s, "|ab..|")
	pl3KExpectPin(t, s, *p5, 3, 0)
	pl3KExpectPin(t, s, *p3, 3, 0)
	pl3KExpectPin(t, s, *p2, 2, 0)
}

// Zig: test "term51: ReflowCursor.reflowRow: a pin in the blanks is clamped to a narrower width"
func TestPl3_term51_ReflowCursor_reflowRow_a_pin_in_the_blanks_is_clamped_to_a_narrower_width(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 6, 1, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|ab....|")
	p5 := pl3Track(t, s, pl3KPin(t, s, 5, 0))
	defer s.UntrackPin(p5)

	r := pl3KReflowInit(t, s, 2, None[PageCapacity]())
	r.run(t)
	r.finish()
	pl3KExpect(t, s, "|ab|")
	pl3KExpectPin(t, s, *p5, 1, 0)
}

// Zig: test "term51: ReflowCursor.reflowRow: a pin on a blank row keeps that row alive"
func TestPl3_term51_ReflowCursor_reflowRow_a_pin_on_a_blank_row_keeps_that_row_alive(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 3, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|a...|\n|....|\n|....|")
	p := pl3Track(t, s, pl3KPin(t, s, 2, 1))
	defer s.UntrackPin(p)

	r := pl3KReflowInit(t, s, 4, None[PageCapacity]())
	r.run(t)
	r.finish()
	pl3KExpect(t, s, "|a...|\n|....|")
	pl3KExpectPin(t, s, *p, 2, 1)
}

// Zig: test "term51: ReflowCursor.reflowRow: a pin in a blank row is clamped using the previous row's cursor column"
func TestPl3_term51_ReflowCursor_reflowRow_a_pin_in_a_blank_row_is_clamped_using_the_previous_row_s_cursor_column(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 3, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|a...|\n|....|\n|....|")
	// x=3 is a valid column of the 4 column destination but the clamp uses
	// the x of the cursor left by the previous row (1): 4 - 1 - 1 = 2.
	p := pl3Track(t, s, pl3KPin(t, s, 3, 1))
	defer s.UntrackPin(p)

	r := pl3KReflowInit(t, s, 4, None[PageCapacity]())
	r.run(t)
	r.finish()
	pl3KExpect(t, s, "|a...|\n|....|")
	pl3KExpectPin(t, s, *p, 2, 1)
}

// Zig: test "term51: ReflowCursor.reflowRow: a pin on a wide cell moved to the next row stays on the spacer head"
func TestPl3_term51_ReflowCursor_reflowRow_a_pin_on_a_wide_cell_moved_to_the_next_row_stays_on_the_spacer_head(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|abW_|")
	p_w := pl3Track(t, s, pl3KPin(t, s, 2, 0))
	p_t := pl3Track(t, s, pl3KPin(t, s, 3, 0))
	defer s.UntrackPin(p_w)
	defer s.UntrackPin(p_t)

	r := pl3KReflowInit(t, s, 3, None[PageCapacity]())
	r.run(t)
	r.finish()
	pl3KExpect(t, s, "|ab^>\n<W_.|")
	// The pin is remapped when the wide cell is first attempted (at the spacer
	// head position) and the remap never matches again after it left the
	// source page: it stays on the spacer head.
	pl3KExpectPin(t, s, *p_w, 2, 0)
	pl3KExpectPin(t, s, *p_t, 1, 1)
}

// Zig: test "term51: ReflowCursor.reflowRow: pins on a destroyed wide cell and its tail share the empty cell"
func TestPl3_term51_ReflowCursor_reflowRow_pins_on_a_destroyed_wide_cell_and_its_tail_share_the_empty_cell(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|aW_b|")
	p_a := pl3Track(t, s, pl3KPin(t, s, 0, 0))
	p_w := pl3Track(t, s, pl3KPin(t, s, 1, 0))
	p_t := pl3Track(t, s, pl3KPin(t, s, 2, 0))
	p_b := pl3Track(t, s, pl3KPin(t, s, 3, 0))
	defer s.UntrackPin(p_a)
	defer s.UntrackPin(p_w)
	defer s.UntrackPin(p_t)
	defer s.UntrackPin(p_b)

	r := pl3KReflowInit(t, s, 1, None[PageCapacity]())
	r.run(t)
	r.finish()
	pl3KExpect(t, s, "|a>\n<.>\n<b|")
	pl3KExpectPin(t, s, *p_a, 0, 0)
	pl3KExpectPin(t, s, *p_w, 0, 1)
	pl3KExpectPin(t, s, *p_t, 0, 1)
	pl3KExpectPin(t, s, *p_b, 0, 2)
}

// Zig: test "term51: ReflowCursor.reflowRow: a pin on an ignored spacer head moves to the next written cell"
func TestPl3_term51_ReflowCursor_reflowRow_a_pin_on_an_ignored_spacer_head_moves_to_the_next_written_cell(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 3, 2, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|ab^>\n<W_.|")
	p_h := pl3Track(t, s, pl3KPin(t, s, 2, 0))
	p_w := pl3Track(t, s, pl3KPin(t, s, 0, 1))
	defer s.UntrackPin(p_h)
	defer s.UntrackPin(p_w)

	r := pl3KReflowInit(t, s, 4, None[PageCapacity]())
	r.run(t)
	r.finish()
	pl3KExpect(t, s, "|abW_|")
	pl3KExpectPin(t, s, *p_h, 2, 0)
	pl3KExpectPin(t, s, *p_w, 2, 0)
}

// Zig: test "term51: ReflowCursor.reflowRow: wide cell at the new last column wraps with a spacer head"
func TestPl3_term51_ReflowCursor_reflowRow_wide_cell_at_the_new_last_column_wraps_with_a_spacer_head(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 5, 2, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|abW_c|\n|dW_..|")

	r := pl3KReflowInit(t, s, 3, None[PageCapacity]())
	r.run(t)
	r.finish()
	// The wide cell of the first row does not fit in column 2 and wraps; the
	// second row has its wide cell in columns 1-2 and fits.
	pl3KExpect(t, s, "|ab^>\n<W_c|\n|dW_|")
}

// Zig: test "term51: ReflowCursor.reflowRow: wide cells are destroyed when reflowing to one column"
func TestPl3_term51_ReflowCursor_reflowRow_wide_cells_are_destroyed_when_reflowing_to_one_column(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|aW_b|")

	r := pl3KReflowInit(t, s, 1, None[PageCapacity]())
	r.run(t)
	r.finish()
	pl3KExpect(t, s, "|a>\n<.>\n<b|")
}

// Zig: test "term51: ReflowCursor.reflowRow: a stray spacer tail is dropped when reflowing to one column"
func TestPl3_term51_ReflowCursor_reflowRow_a_stray_spacer_tail_is_dropped_when_reflowing_to_one_column(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 3, 1, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|a_b|")

	r := pl3KReflowInit(t, s, 1, None[PageCapacity]())
	r.run(t)
	r.finish()
	pl3KExpect(t, s, "|a>\n<b|")
}

// Zig: test "term51: ReflowCursor.reflowRow: unwrapping drops the spacer head"
func TestPl3_term51_ReflowCursor_reflowRow_unwrapping_drops_the_spacer_head(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 3, 2, Some[uint64](0))
	defer s.Deinit()
	pl3KLoad(t, s, "|ab^>\n<W_c|")

	r := pl3KReflowInit(t, s, 6, None[PageCapacity]())
	r.run(t)
	r.finish()
	pl3KExpect(t, s, "|abW_c.|")
}

// ---------------------------------------------------------------------------
// Go-only: panic tests. Zig's tests cannot exercise an assert, an
// `unreachable` or an unwrap of null (the test runner would abort); each of
// these checks one such path of a function under test in this file.
// ---------------------------------------------------------------------------

// Go-only: panic test: resize: cols is zero (assert(v > 0))
func TestPl3_panic_resize_cols_is_zero(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 3, None[uint64]())
	pl3ExpectPanic(t, func() {
		_ = s.Resize(PageListResize{Cols: pl3Dim(0), Reflow: true})
	})
}

// Go-only: panic test: resize: rows is zero (assert(v > 0))
func TestPl3_panic_resize_rows_is_zero(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 3, None[uint64]())
	pl3ExpectPanic(t, func() {
		_ = s.Resize(PageListResize{Rows: pl3Dim(0), Reflow: true})
	})
}

// Go-only: panic test: resizeCols: the new cols equal the current cols (assert(cols != self.cols))
func TestPl3_panic_resizeCols_cols_equal_to_the_current_cols(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 3, None[uint64]())
	pl3ExpectPanic(t, func() {
		_ = s.resizeCols(4, pl3NoCursor())
	})
}

// Go-only: panic test: ReflowCursor.cursorScroll: the cursor is not on the last row of the page (assert(self.y == self.page.size.rows - 1))
func TestPl3_panic_ReflowCursor_cursorScroll_cursor_not_on_the_last_row(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, None[uint64]())
	node := pl3KAddNode(t, s, pl3Cap(4, 3), 2)
	rc := pageListReflowCursorInit(node)
	pl3ExpectPanic(t, func() {
		rc.cursorScroll()
	})
}

// Go-only: panic test: ReflowCursor.cursorScroll: the page has no capacity for another row (assert(self.page.size.rows < self.page.capacity.rows))
func TestPl3_panic_ReflowCursor_cursorScroll_page_is_at_capacity(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, None[uint64]())
	node := pl3KAddNode(t, s, pl3Cap(4, 3), 3)
	rc := pageListReflowCursorInit(node)
	rc.cursorAbsolute(0, 2)
	pl3ExpectPanic(t, func() {
		rc.cursorScroll()
	})
}

// Go-only: panic test: ReflowCursor.cursorAbsolute: x is not inside the page (assert(x < self.page.size.cols))
func TestPl3_panic_ReflowCursor_cursorAbsolute_x_out_of_range(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, None[uint64]())
	node := pl3KAddNode(t, s, pl3Cap(4, 3), 2)
	rc := pageListReflowCursorInit(node)
	pl3ExpectPanic(t, func() {
		rc.cursorAbsolute(4, 0)
	})
}

// Go-only: panic test: ReflowCursor.cursorAbsolute: y is not inside the page (assert(y < self.page.size.rows))
func TestPl3_panic_ReflowCursor_cursorAbsolute_y_out_of_range(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, None[uint64]())
	node := pl3KAddNode(t, s, pl3Cap(4, 3), 2)
	rc := pageListReflowCursorInit(node)
	pl3ExpectPanic(t, func() {
		rc.cursorAbsolute(0, 2)
	})
}

// Go-only: panic test: ReflowCursor.moveLastRowToNewPage: the cursor is not on the final row of the page (assert(self.y == self.page.size.rows - 1))
func TestPl3_panic_ReflowCursor_moveLastRowToNewPage_cursor_not_on_the_final_row(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, None[uint64]())
	node := pl3KAddNode(t, s, pl3Cap(4, 3), 2)
	rc := pageListReflowCursorInit(node)
	pl3ExpectPanic(t, func() {
		_ = rc.moveLastRowToNewPage(s, pl3Cap(4, 3))
	})
}

// Go-only: panic test: ReflowCursor.moveLastRowToNewPage: the cursor has a pending wrap (assert(!self.pending_wrap))
func TestPl3_panic_ReflowCursor_moveLastRowToNewPage_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, None[uint64]())
	node := pl3KAddNode(t, s, pl3Cap(4, 3), 1)
	rc := pageListReflowCursorInit(node)
	rc.pending_wrap = true
	pl3ExpectPanic(t, func() {
		_ = rc.moveLastRowToNewPage(s, pl3Cap(4, 3))
	})
}

// Go-only: panic test: ReflowCursor.writeCell: a grapheme cell without grapheme data in the source page (src_page.lookupGrapheme(cell).? unwraps null)
func TestPl3_panic_ReflowCursor_writeCell_grapheme_cell_without_grapheme_data(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, None[uint64]())
	node := pl3KAddNode(t, s, pl3Cap(4, 2), 1)
	rc := pageListReflowCursorInit(node)
	src_page := &s.pages.first.data
	cell := PageCell{
		content_tag: PageCellContentTagCodepointGrapheme,
		content:     PageCellContent{codepoint: 'e'},
	}
	pl3ExpectPanic(t, func() {
		_, _ = rc.writeCell(s, &cell, src_page)
	})
}

// Go-only: panic test: ReflowCursor.writeCell: a hyperlink cell without hyperlink data in the source page (src_page.lookupHyperlink(cell).? unwraps null)
func TestPl3_panic_ReflowCursor_writeCell_hyperlink_cell_without_hyperlink_data(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 4, 1, None[uint64]())
	node := pl3KAddNode(t, s, pl3Cap(4, 2), 1)
	rc := pageListReflowCursorInit(node)
	src_page := &s.pages.first.data
	cell := PageCell{
		content_tag: PageCellContentTagCodepoint,
		content:     PageCellContent{codepoint: 'x'},
		hyperlink:   true,
	}
	pl3ExpectPanic(t, func() {
		_, _ = rc.writeCell(s, &cell, src_page)
	})
}

// Go-only: panic test: clone: the top point is outside the list (pageIterator unwraps self.pin(tl_pt).?)
func TestPl3_panic_clone_top_point_outside_the_list(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl3ExpectPanic(t, func() {
		_, _ = s.Clone(PageListClone{Top: pl3Screen(0, 1000)})
	})
}

// Go-only: panic test: clone: the bottom point is outside the list (pageIterator unwraps self.pin(pt).?)
func TestPl3_panic_clone_bottom_point_outside_the_list(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl3ExpectPanic(t, func() {
		_, _ = s.Clone(PageListClone{Top: pl3Screen(0, 0), Bot: Some(pl3Screen(0, 1000))})
	})
}

// Go-only: panic test: clone: the top point is below the bottom point (assert(tl_pin.eql(bl_pin) or tl_pin.before(bl_pin)))
func TestPl3_panic_clone_top_point_below_the_bottom_point(t *testing.T) {
	testStdCapacity(t)
	s := pl3Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl3GrowRows(t, s, 30)
	pl3ExpectPanic(t, func() {
		_, _ = s.Clone(PageListClone{Top: pl3Screen(0, 10), Bot: Some(pl3Screen(0, 5))})
	})
}
