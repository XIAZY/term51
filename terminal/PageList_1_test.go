// Tests translated one for one from src/terminal/PageList.zig (Ghostty v1.3.1),
// first part: the tests from "PageList" (line 5491) up to and including
// "PageList resize (no reflow) less rows and cols" (line 10751 is the last
// line of scope; the next test starts at line 10752). Each Go test carries
// the exact Zig title in a "// Zig: test" comment. Helpers are prefixed pl1.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"),
//     expectEqual / expectEqualStrings keep Zig's names and Zig's (expected,
//     actual) argument order. `try` on a call that returns an error is
//     zigTry(t, err); `x.?` is zigUnwrap(t, x) for an Optional and a plain
//     (nil-checked by the next line) pointer otherwise.
//   - Every test starts with testStdCapacity(t): Zig's unit tests run with a
//     standard page capacity of grapheme_bytes 512, so `std_capacity` is
//     PageStdCapacity and `std_size` is pageListStdSize().
//   - `try init(alloc, cols, rows, max)` is pl1Init(t, cols, rows, max) with
//     max an Optional[uint64] (`null` is None[uint64](), `0` is Some[uint64](0)).
//     `defer s.deinit()` is `defer s.Deinit()`. `try s.grow()` is pl1Grow, which
//     returns the new node or nil (`?*List.Node`). `try s.growRows(n)` is
//     pl1GrowRows. `try s.clone(alloc, opts)` is pl1Clone.
//   - Zig's `usize` row/column counts compare against uint, `size.CellCountInt`
//     values against SizeCellCountInt; the conversions are written out.
//   - Zig struct literals with defaulted fields (`.{ .screen = .{} }`,
//     `Resize{ .rows = 5, .reflow = false }`, `Cell{ .content_tag = ...}`)
//     are written out with every field in Go; pl1Active, pl1Screen and
//     pl1History build the three Point literals, pl1Codepoint and
//     pl1CellSemantic the two Cell literals the tests use.
//   - The `testing.allocator` / `alloc` arguments and the leak checks are
//     dropped. `Clone.TrackedPinsRemap.init(alloc)` is a Go map literal.
//   - `page.rows.ptr(page.memory)` is `page.rows`; `page.memory` and the
//     style set's `page.memory` argument are gone (Go memory).
//   - `s.pages.first.?` is `s.pages.first` (a *PageListNode; the tests would
//     dereference a nil pointer where Zig's `.?` would trap).
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt. The Go-only panic tests at the end
//     (`TestPl1_panic_...`) cover assert / unreachable / null-unwrap paths
//     that Zig's tests cannot reach.

package terminal

import (
	"errors"
	"math"
	"testing"
)

// pl1ExpectError is Zig's `try testing.expectError(expected, actual)`.
func pl1ExpectError(t *testing.T, want error, got error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

// pl1ExpectPanic runs f and fails the test if f does not panic.
func pl1ExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if recover() == nil {
			t.Fatalf("expected a panic, got none")
		}
	}()
	f()
}

// pl1Init is `try init(alloc, cols, rows, max_size)`.
func pl1Init(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt, max_size Optional[uint64]) PageList {
	t.Helper()
	s, err := PageListInit(cols, rows, max_size)
	zigTry(t, err)
	return s
}

// pl1Grow is `try s.grow()`: the new node, or nil when the last page had
// capacity (Zig's null).
func pl1Grow(t *testing.T, s *PageList) *PageListNode {
	t.Helper()
	node, err := s.Grow()
	zigTry(t, err)
	return node
}

// pl1GrowRows is `try s.growRows(n)`.
func pl1GrowRows(t *testing.T, s *PageList, n uint) {
	t.Helper()
	zigTry(t, s.growRows(n))
}

// pl1Clone is `try s.clone(alloc, opts)`.
func pl1Clone(t *testing.T, s *PageList, opts PageListClone) PageList {
	t.Helper()
	c, err := s.Clone(opts)
	zigTry(t, err)
	return c
}

// pl1Active is the Zig literal `.{ .active = .{ .x = x, .y = y } }`.
func pl1Active(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagActive, Active: PointCoordinate{x: x, y: y}}
}

// pl1Screen is the Zig literal `.{ .screen = .{ .x = x, .y = y } }`.
func pl1Screen(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagScreen, Screen: PointCoordinate{x: x, y: y}}
}

// pl1History is the Zig literal `.{ .history = .{ .x = x, .y = y } }`.
func pl1History(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagHistory, History: PointCoordinate{x: x, y: y}}
}

// pl1Codepoint is the Zig literal `.{ .content_tag = .codepoint, .content =
// .{ .codepoint = cp } }`.
func pl1Codepoint(cp uint32) PageCell {
	return PageCell{
		content_tag: PageCellContentTagCodepoint,
		content:     PageCellContent{codepoint: cp},
	}
}

// pl1CellSemantic is the Zig literal `.{ .content_tag = .codepoint, .content =
// .{ .codepoint = cp }, .semantic_content = sc }`.
func pl1CellSemantic(cp uint32, sc PageCellSemanticContent) PageCell {
	return PageCell{
		content_tag:      PageCellContentTagCodepoint,
		content:          PageCellContent{codepoint: cp},
		semantic_content: sc,
	}
}

// ---------------------------------------------------------------------------
// Tests.
// ---------------------------------------------------------------------------

// Zig: test "PageList"
func TestPl1_PageList(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	expect(t, s.pages.first != nil, "s.pages.first != null")
	expectEqual(t, uint(s.rows), s.totalRows())

	// Initial total rows should be our row count
	expectEqual(t, uint(s.rows), s.total_rows)

	// Our viewport pin must be defined. It isn't used until the
	// viewport is a pin but it prevents undefined access on clone.
	expect(t, s.viewport_pin.node == s.pages.first, "s.viewport_pin.node == s.pages.first.?")

	// Active area should be the top
	expectEqual(t, PageListPin{
		node: s.pages.first,
		y:    0,
		x:    0,
	}, s.GetTopLeft(PointTagActive))

	// Scrollbar should be where we expect it
	expectEqual(t, PageListScrollbar{
		total:  uint(s.rows),
		offset: 0,
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList init rows across two pages"
func TestPl1_PageList_init_rows_across_two_pages(t *testing.T) {
	testStdCapacity(t)

	// Find a cap that makes it so that rows don't fit on one page.
	const rows = 100
	cap := func() PageCapacity {
		cap, err := PageStdCapacity.Adjust(PageCapacityAdjustment{cols: Some[SizeCellCountInt](50)})
		zigTry(t, err)
		for cap.rows >= rows {
			cap, err = PageStdCapacity.Adjust(PageCapacityAdjustment{
				cols: Some(cap.cols + 50),
			})
			zigTry(t, err)
		}

		return cap
	}()

	// Init
	s := pl1Init(t, cap.cols, rows, None[uint64]())
	defer s.Deinit()
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	expect(t, s.pages.first != nil, "s.pages.first != null")
	expectEqual(t, uint(s.rows), s.totalRows())

	// Initial total rows should be our row count
	expectEqual(t, uint(s.rows), s.total_rows)

	// Scrollbar should be where we expect it
	expectEqual(t, PageListScrollbar{
		total:  uint(s.rows),
		offset: 0,
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList init more than max cols"
func TestPl1_PageList_init_more_than_max_cols(t *testing.T) {
	testStdCapacity(t)

	// Initialize with more columns than we can fit in our standard
	// capacity. This is going to force us to go to a non-standard page
	// immediately.
	s := pl1Init(
		t,
		zigUnwrap(t, PageStdCapacity.MaxCols())+1,
		80,
		None[uint64](),
	)
	defer s.Deinit()
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	expectEqual(t, uint(s.rows), s.totalRows())

	// We expect a single, non-standard page
	expect(t, s.pages.first != nil, "s.pages.first != null")
	expect(t, pageListMemoryLen(&s.pages.first.data) > pageListStdSize(),
		"s.pages.first.?.data.memory.len > std_size")

	// Initial total rows should be our row count
	expectEqual(t, uint(s.rows), s.total_rows)

	// Scrollbar should be where we expect it
	expectEqual(t, PageListScrollbar{
		total:  uint(s.rows),
		offset: 0,
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList pointFromPin active no history"
func TestPl1_PageList_pointFromPin_active_no_history(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	{
		expectEqualPoint(t, pl1Active(0, 0), zigUnwrap(t, s.PointFromPin(PointTagActive, PageListPin{
			node: s.pages.first,
			y:    0,
			x:    0,
		})))
	}
	{
		expectEqualPoint(t, pl1Active(4, 2), zigUnwrap(t, s.PointFromPin(PointTagActive, PageListPin{
			node: s.pages.first,
			y:    2,
			x:    4,
		})))
	}
}

// Zig: test "PageList pointFromPin active with history"
func TestPl1_PageList_pointFromPin_active_with_history(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl1GrowRows(t, &s, 30)

	{
		expectEqualPoint(t, pl1Active(2, 0), zigUnwrap(t, s.PointFromPin(PointTagActive, PageListPin{
			node: s.pages.first,
			y:    30,
			x:    2,
		})))
	}

	// In history, invalid
	{
		expect(t, !s.PointFromPin(PointTagActive, PageListPin{
			node: s.pages.first,
			y:    21,
			x:    2,
		}).Valid, "s.pointFromPin(.active, .{ ... }) == null")
	}
}

// Zig: test "PageList pointFromPin active from prior page"
func TestPl1_PageList_pointFromPin_active_from_prior_page(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	// Grow so we take up at least 5 pages.
	page := &s.pages.last.data
	cur_page := s.pages.last
	cur_page.data.PauseIntegrityChecks(true)
	for range uint(page.capacity.rows) * 5 {
		if new_page := pl1Grow(t, &s); new_page != nil {
			cur_page.data.PauseIntegrityChecks(false)
			cur_page = new_page
			cur_page.data.PauseIntegrityChecks(true)
		}
	}
	cur_page.data.PauseIntegrityChecks(false)

	{
		expectEqualPoint(t, pl1Active(2, 0), zigUnwrap(t, s.PointFromPin(PointTagActive, PageListPin{
			node: s.pages.last,
			y:    0,
			x:    2,
		})))
	}

	// Prior page
	{
		expect(t, !s.PointFromPin(PointTagActive, PageListPin{
			node: s.pages.first,
			y:    0,
			x:    0,
		}).Valid, "s.pointFromPin(.active, .{ ... }) == null")
	}
}

// Zig: test "PageList pointFromPin traverse pages"
func TestPl1_PageList_pointFromPin_traverse_pages(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Grow so we take up at least 2 pages.
	page := &s.pages.last.data
	cur_page := s.pages.last
	cur_page.data.PauseIntegrityChecks(true)
	for range uint(page.capacity.rows) * 2 {
		if new_page := pl1Grow(t, &s); new_page != nil {
			cur_page.data.PauseIntegrityChecks(false)
			cur_page = new_page
			cur_page.data.PauseIntegrityChecks(true)
		}
	}
	cur_page.data.PauseIntegrityChecks(false)

	{
		pages := s.TotalPages()
		page_cap := uint(page.capacity.rows)
		expected_y := page_cap*(pages-2) + 5

		expectEqualPoint(t, pl1Screen(2, uint32(expected_y)), zigUnwrap(t, s.PointFromPin(PointTagScreen, PageListPin{
			node: s.pages.last.prev,
			y:    5,
			x:    2,
		})))
	}

	// Prior page
	{
		expect(t, !s.PointFromPin(PointTagActive, PageListPin{
			node: s.pages.first,
			y:    0,
			x:    0,
		}).Valid, "s.pointFromPin(.active, .{ ... }) == null")
	}
}

// Zig: test "PageList active after grow"
func TestPl1_PageList_active_after_grow(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	expectEqual(t, uint(s.rows), s.totalRows())

	pl1GrowRows(t, &s, 10)
	expectEqual(t, uint(s.rows)+10, s.totalRows())

	// Make sure all points make sense
	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 10), pt)
	}
	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagScreen})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 0), pt)
	}
	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagActive})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 10), pt)
	}

	// Scrollbar should be in the active area
	expectEqual(t, PageListScrollbar{
		total:  s.totalRows(),
		offset: 10,
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList grow allows exceeding max size for active area"
func TestPl1_PageList_grow_allows_exceeding_max_size_for_active_area(t *testing.T) {
	testStdCapacity(t)

	// Setup our initial page so that we fully take up one page.
	cap, err := PageStdCapacity.Adjust(PageCapacityAdjustment{cols: Some[SizeCellCountInt](5)})
	zigTry(t, err)
	s := pl1Init(t, 5, cap.rows, Some[uint64](0))
	defer s.Deinit()
	expectEqual(t, uint(s.rows), s.totalRows())

	// Grow once because we guarantee at least two pages of
	// capacity so we want to get to that.
	pl1Grow(t, &s)
	start_pages := s.TotalPages()
	expect(t, start_pages >= 2, "start_pages >= 2")

	// Surgically modify our pages so that they have a smaller size.
	{
		it := s.pages.first
		for it != nil {
			page := it
			page.data.size.rows = 1
			page.data.capacity.rows = 1
			it = page.next
		}

		// Avoid integrity check failures
		s.total_rows = s.totalRows()
	}

	// Grow our row and ensure we don't prune pages because we need
	// enough for the active area.
	pl1Grow(t, &s)
	expectEqual(t, start_pages+1, s.TotalPages())
}

// Zig: test "PageList grow prune required with a single page"
func TestPl1_PageList_grow_prune_required_with_a_single_page(t *testing.T) {
	testStdCapacity(t)

	// Need scrollback > 0 to have a scrollbar to test
	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// This block is all test setup. There is nothing required about this
	// behavior during a refactor. This is setting up a scenario that is
	// possible to trigger a bug (#2280).
	{
		// Increase our capacity until our page is larger than the standard size.
		// This is important because it triggers a scenario where our calculated
		// minSize() which is supposed to accommodate 2 pages is no longer true.
		for {
			layout := PageLayoutOf(s.pages.first.data.capacity)
			if layout.total_size > pageListStdSize() {
				break
			}
			_, err := s.IncreaseCapacity(s.pages.first, Some(PageListIncreaseCapacityGraphemeBytes))
			zigTry(t, err)
		}
		expect(t, s.pages.first != nil, "s.pages.first != null")
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	}

	// Figure out the remaining number of rows. This is the amount that
	// can be added to the current page before we need to allocate a new
	// page.
	rem := func() uint {
		page := s.pages.first
		return uint(page.data.capacity.rows) - uint(page.data.size.rows)
	}()
	for range rem {
		expect(t, pl1Grow(t, &s) == nil, "try s.grow() == null")
	}

	// The next one we add will trigger a new page.
	new := pl1Grow(t, &s)
	expect(t, new != nil, "new != null")
	expect(t, new != s.pages.first, "new != s.pages.first")

	// Scrollbar should be in the active area
	expectEqual(t, PageListScrollbar{
		total:  s.totalRows(),
		offset: s.total_rows - uint(s.rows),
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList scrollbar with max_size 0 after grow"
func TestPl1_PageList_scrollbar_with_max_size_0_after_grow(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, Some[uint64](0))
	defer s.Deinit()

	// Grow some rows (simulates normal terminal output)
	pl1GrowRows(t, &s, 10)

	sb := s.Scrollbar()

	// With no scrollback (max_size = 0), total should equal rows
	expectEqual(t, uint(s.rows), sb.total)

	// With no scrollback, offset should be 0 (nowhere to scroll back to)
	expectEqual(t, uint(0), sb.offset)
}

// Zig: test "PageList scroll with max_size 0 no history"
func TestPl1_PageList_scroll_with_max_size_0_no_history(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, Some[uint64](0))
	defer s.Deinit()

	pl1GrowRows(t, &s, 10)

	// Remember initial viewport position
	pt_before := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()

	// Try to scroll backwards into "history" - should be no-op
	s.Scroll(PageListScroll{Tag: PageListScrollDeltaRow, DeltaRow: -5})
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	// Scroll to top - should also be no-op with no scrollback
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	pt_after := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
	expectEqualPoint(t, pt_before, pt_after)
}

// Zig: test "PageList scroll top"
func TestPl1_PageList_scroll_top(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl1GrowRows(t, &s, 10)

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 10), pt)
	}

	s.Scroll(PageListScroll{Tag: PageListScrollTop})

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 0), pt)
	}

	expectEqual(t, PageListScrollbar{
		total:  s.totalRows(),
		offset: 0,
		len:    uint(s.rows),
	}, s.Scrollbar())

	pl1GrowRows(t, &s, 10)
	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 0), pt)
	}

	expectEqual(t, PageListScrollbar{
		total:  s.totalRows(),
		offset: 0,
		len:    uint(s.rows),
	}, s.Scrollbar())

	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 20), pt)
	}

	expectEqual(t, PageListScrollbar{
		total:  s.totalRows(),
		offset: s.total_rows - uint(s.rows),
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList scroll delta row back"
func TestPl1_PageList_scroll_delta_row_back(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl1GrowRows(t, &s, 10)

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 10), pt)
	}

	s.Scroll(PageListScroll{Tag: PageListScrollDeltaRow, DeltaRow: -1})

	expectEqual(t, PageListScrollbar{
		total:  s.totalRows(),
		offset: s.total_rows - uint(s.rows) - 1,
		len:    uint(s.rows),
	}, s.Scrollbar())

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 9), pt)
	}

	pl1GrowRows(t, &s, 10)
	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 9), pt)
	}

	expectEqual(t, PageListScrollbar{
		total:  s.totalRows(),
		offset: s.total_rows - uint(s.rows) - 11,
		len:    uint(s.rows),
	}, s.Scrollbar())

	s.Scroll(PageListScroll{Tag: PageListScrollDeltaRow, DeltaRow: -1})

	expectEqual(t, PageListScrollbar{
		total:  s.totalRows(),
		offset: s.total_rows - uint(s.rows) - 12,
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList scroll delta row back overflow"
func TestPl1_PageList_scroll_delta_row_back_overflow(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl1GrowRows(t, &s, 10)

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 10), pt)
	}

	s.Scroll(PageListScroll{Tag: PageListScrollDeltaRow, DeltaRow: -100})

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 0), pt)
	}

	expectEqual(t, PageListScrollbar{
		total:  s.totalRows(),
		offset: 0,
		len:    uint(s.rows),
	}, s.Scrollbar())

	pl1GrowRows(t, &s, 10)
	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 0), pt)
	}

	expectEqual(t, PageListScrollbar{
		total:  s.totalRows(),
		offset: 0,
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList scroll delta row forward"
func TestPl1_PageList_scroll_delta_row_forward(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl1GrowRows(t, &s, 10)

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 10), pt)
	}

	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	s.Scroll(PageListScroll{Tag: PageListScrollDeltaRow, DeltaRow: 2})

	expectEqual(t, PageListScrollbar{
		total:  s.totalRows(),
		offset: 2,
		len:    uint(s.rows),
	}, s.Scrollbar())

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 2), pt)
	}

	pl1GrowRows(t, &s, 10)
	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 2), pt)
	}

	expectEqual(t, PageListScrollbar{
		total:  s.totalRows(),
		offset: 2,
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList scroll delta row forward into active"
func TestPl1_PageList_scroll_delta_row_forward_into_active(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	s.Scroll(PageListScroll{Tag: PageListScrollDeltaRow, DeltaRow: 2})

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 0), pt)
	}

	expectEqual(t, PageListScrollbar{
		total:  s.totalRows(),
		offset: s.total_rows - uint(s.rows),
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList scroll delta row back without space preserves active"
func TestPl1_PageList_scroll_delta_row_back_without_space_preserves_active(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	s.Scroll(PageListScroll{Tag: PageListScrollDeltaRow, DeltaRow: -1})

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 0), pt)
	}

	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	expectEqual(t, PageListScrollbar{
		total:  s.totalRows(),
		offset: s.total_rows - uint(s.rows),
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList scroll to pin"
func TestPl1_PageList_scroll_to_pin(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl1GrowRows(t, &s, 10)

	s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: zigUnwrap(t, s.Pin(pl1Screen(2, 4)))})

	expectEqual(t, PageListScrollbar{
		total:  s.totalRows(),
		offset: 4,
		len:    uint(s.rows),
	}, s.Scrollbar())

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 4), pt)
	}

	s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: zigUnwrap(t, s.Pin(pl1Screen(2, 5)))})

	expectEqual(t, PageListScrollbar{
		total:  s.totalRows(),
		offset: 5,
		len:    uint(s.rows),
	}, s.Scrollbar())

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 5), pt)
	}
}

// Zig: test "PageList scroll to pin in active"
func TestPl1_PageList_scroll_to_pin_in_active(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl1GrowRows(t, &s, 10)

	s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: zigUnwrap(t, s.Pin(pl1Screen(2, 30)))})

	expectEqual(t, PageListScrollbar{
		total:  s.totalRows(),
		offset: s.total_rows - uint(s.rows),
		len:    uint(s.rows),
	}, s.Scrollbar())

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 10), pt)
	}
}

// Zig: test "PageList scroll to pin at top"
func TestPl1_PageList_scroll_to_pin_at_top(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl1GrowRows(t, &s, 10)

	s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: zigUnwrap(t, s.Pin(pl1Screen(2, 0)))})

	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")

	expectEqual(t, PageListScrollbar{
		total:  s.totalRows(),
		offset: 0,
		len:    uint(s.rows),
	}, s.Scrollbar())

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 0), pt)
	}
}

// Zig: test "PageList scroll to row 0"
func TestPl1_PageList_scroll_to_row_0(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl1GrowRows(t, &s, 10)

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 10), pt)
	}

	s.Scroll(PageListScroll{Tag: PageListScrollRow, Row: 0})
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 0), pt)
	}

	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: 0,
		len:    uint(s.rows),
	}, s.Scrollbar())

	pl1GrowRows(t, &s, 10)
	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 0), pt)
	}

	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: 0,
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList scroll to row in scrollback"
func TestPl1_PageList_scroll_to_row_in_scrollback(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl1GrowRows(t, &s, 20)

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 20), pt)
	}

	s.Scroll(PageListScroll{Tag: PageListScrollRow, Row: 5})
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: 5,
		len:    uint(s.rows),
	}, s.Scrollbar())

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 5), pt)
	}

	pl1GrowRows(t, &s, 10)
	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 5), pt)
	}

	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: 5,
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList scroll to row in middle"
func TestPl1_PageList_scroll_to_row_in_middle(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl1GrowRows(t, &s, 50)

	total := s.total_rows
	midpoint := total / 2
	s.Scroll(PageListScroll{Tag: PageListScrollRow, Row: midpoint})

	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: midpoint,
		len:    uint(s.rows),
	}, s.Scrollbar())

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, uint32(SizeCellCountInt(midpoint))), pt)
	}

	pl1GrowRows(t, &s, 10)
	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, uint32(SizeCellCountInt(midpoint))), pt)
	}

	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: midpoint,
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList scroll to row at active boundary"
func TestPl1_PageList_scroll_to_row_at_active_boundary(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl1GrowRows(t, &s, 20)

	active_start := s.total_rows - uint(s.rows)

	s.Scroll(PageListScroll{Tag: PageListScrollRow, Row: active_start})

	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, uint32(SizeCellCountInt(active_start))), pt)
	}

	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: s.total_rows - uint(s.rows),
		len:    uint(s.rows),
	}, s.Scrollbar())

	pl1GrowRows(t, &s, 10)

	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: s.total_rows - uint(s.rows),
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList scroll to row beyond active"
func TestPl1_PageList_scroll_to_row_beyond_active(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl1GrowRows(t, &s, 10)

	s.Scroll(PageListScroll{Tag: PageListScrollRow, Row: 1000})

	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 10), pt)
	}

	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: s.total_rows - uint(s.rows),
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList scroll to row without scrollback"
func TestPl1_PageList_scroll_to_row_without_scrollback(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	s.Scroll(PageListScroll{Tag: PageListScrollRow, Row: 5})

	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 0), pt)
	}

	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: s.total_rows - uint(s.rows),
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList scroll to row then delta"
func TestPl1_PageList_scroll_to_row_then_delta(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl1GrowRows(t, &s, 30)

	s.Scroll(PageListScroll{Tag: PageListScrollRow, Row: 10})

	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 10), pt)
	}

	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: 10,
		len:    uint(s.rows),
	}, s.Scrollbar())

	s.Scroll(PageListScroll{Tag: PageListScrollDeltaRow, DeltaRow: 5})

	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 15), pt)
	}

	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: 15,
		len:    uint(s.rows),
	}, s.Scrollbar())

	s.Scroll(PageListScroll{Tag: PageListScrollDeltaRow, DeltaRow: -3})

	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 12), pt)
	}

	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: 12,
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList scroll to row with cache fast path down"
func TestPl1_PageList_scroll_to_row_with_cache_fast_path_down(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl1GrowRows(t, &s, 50)

	s.Scroll(PageListScroll{Tag: PageListScrollRow, Row: 10})

	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: 10,
		len:    uint(s.rows),
	}, s.Scrollbar())

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 10), pt)
	}

	// Verify cache is populated
	expect(t, s.viewport_pin_row_offset.Valid, "s.viewport_pin_row_offset != null")
	expectEqual(t, uint(10), zigUnwrap(t, s.viewport_pin_row_offset))

	// Now scroll to a different row - this should use the fast path
	s.Scroll(PageListScroll{Tag: PageListScrollRow, Row: 20})

	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: 20,
		len:    uint(s.rows),
	}, s.Scrollbar())

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 20), pt)
	}

	pl1GrowRows(t, &s, 10)
	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 20), pt)
	}

	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: 20,
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList scroll to row with cache fast path up"
func TestPl1_PageList_scroll_to_row_with_cache_fast_path_up(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl1GrowRows(t, &s, 50)

	s.Scroll(PageListScroll{Tag: PageListScrollRow, Row: 30})

	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: 30,
		len:    uint(s.rows),
	}, s.Scrollbar())

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 30), pt)
	}

	// Verify cache is populated
	expect(t, s.viewport_pin_row_offset.Valid, "s.viewport_pin_row_offset != null")
	expectEqual(t, uint(30), zigUnwrap(t, s.viewport_pin_row_offset))

	// Now scroll up to a different row - this should use the fast path
	s.Scroll(PageListScroll{Tag: PageListScrollRow, Row: 15})

	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: 15,
		len:    uint(s.rows),
	}, s.Scrollbar())

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 15), pt)
	}

	pl1GrowRows(t, &s, 10)
	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 15), pt)
	}

	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: 15,
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList scroll clear"
func TestPl1_PageList_scroll_clear(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	{
		cell := zigUnwrap(t, s.GetCell(pl1Active(0, 0)))
		*cell.cell = pl1Codepoint('A')
	}
	{
		cell := zigUnwrap(t, s.GetCell(pl1Active(0, 1)))
		*cell.cell = pl1Codepoint('A')
	}

	zigTry(t, s.ScrollClear())

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 2), pt)
	}
}

// Zig: test "PageList: jump zero prompts"
func TestPl1_PageList_jump_zero_prompts(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 5, 3, None[uint64]())
	defer s.Deinit()
	pl1GrowRows(t, &s, 3)
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	{
		rac := page.GetRowAndCell(0, 1)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}
	{
		rac := page.GetRowAndCell(0, 5)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}

	s.Scroll(PageListScroll{Tag: PageListScrollDeltaPrompt, DeltaPrompt: 0})
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: s.total_rows - uint(s.rows),
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "Screen: jump back one prompt"
func TestPl1_Screen_jump_back_one_prompt(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 5, 3, None[uint64]())
	defer s.Deinit()
	pl1GrowRows(t, &s, 3)
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	{
		rac := page.GetRowAndCell(0, 1)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}
	{
		rac := page.GetRowAndCell(0, 5)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}

	// Jump back
	{
		s.Scroll(PageListScroll{Tag: PageListScrollDeltaPrompt, DeltaPrompt: -1})
		expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
		expectEqualPoint(t, pl1Screen(0, 1), zigUnwrap(t, s.PointFromPin(PointTagScreen, zigUnwrap(t, s.Pin(Point{Tag: PointTagViewport})))))

		expectEqual(t, PageListScrollbar{
			total:  s.total_rows,
			offset: 1,
			len:    uint(s.rows),
		}, s.Scrollbar())
	}
	{
		s.Scroll(PageListScroll{Tag: PageListScrollDeltaPrompt, DeltaPrompt: -1})
		expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
		expectEqualPoint(t, pl1Screen(0, 1), zigUnwrap(t, s.PointFromPin(PointTagScreen, zigUnwrap(t, s.Pin(Point{Tag: PointTagViewport})))))

		expectEqual(t, PageListScrollbar{
			total:  s.total_rows,
			offset: 1,
			len:    uint(s.rows),
		}, s.Scrollbar())
	}

	// Jump forward
	{
		s.Scroll(PageListScroll{Tag: PageListScrollDeltaPrompt, DeltaPrompt: 1})
		expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
		expectEqual(t, PageListScrollbar{
			total:  s.total_rows,
			offset: s.total_rows - uint(s.rows),
			len:    uint(s.rows),
		}, s.Scrollbar())
	}
	{
		s.Scroll(PageListScroll{Tag: PageListScrollDeltaPrompt, DeltaPrompt: 1})
		expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
		expectEqual(t, PageListScrollbar{
			total:  s.total_rows,
			offset: s.total_rows - uint(s.rows),
			len:    uint(s.rows),
		}, s.Scrollbar())
	}
}

// Zig: test "Screen: jump forward prompt skips multiline continuation"
func TestPl1_Screen_jump_forward_prompt_skips_multiline_continuation(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 5, 3, None[uint64]())
	defer s.Deinit()
	pl1GrowRows(t, &s, 7)

	// Multiline prompt on rows 1-3.
	{
		p := zigUnwrap(t, s.Pin(pl1Screen(0, 1)))
		p.RowAndCell().row.semantic_prompt = PageRowSemanticPromptPrompt
	}
	{
		p := zigUnwrap(t, s.Pin(pl1Screen(0, 2)))
		p.RowAndCell().row.semantic_prompt = PageRowSemanticPromptPromptContinuation
	}
	{
		p := zigUnwrap(t, s.Pin(pl1Screen(0, 3)))
		p.RowAndCell().row.semantic_prompt = PageRowSemanticPromptPromptContinuation
	}

	// Next prompt after command output.
	{
		p := zigUnwrap(t, s.Pin(pl1Screen(0, 6)))
		p.RowAndCell().row.semantic_prompt = PageRowSemanticPromptPrompt
	}

	// Starting at the first prompt line should jump to the next prompt,
	// not to continuation lines.
	s.Scroll(PageListScroll{Tag: PageListScrollRow, Row: 1})
	s.Scroll(PageListScroll{Tag: PageListScrollDeltaPrompt, DeltaPrompt: 1})
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqualPoint(t, pl1Screen(0, 6), zigUnwrap(t, s.PointFromPin(PointTagScreen, zigUnwrap(t, s.Pin(Point{Tag: PointTagViewport})))))

	// Starting in the middle of continuation lines should also jump to
	// the next prompt.
	s.Scroll(PageListScroll{Tag: PageListScrollRow, Row: 2})
	s.Scroll(PageListScroll{Tag: PageListScrollDeltaPrompt, DeltaPrompt: 1})
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqualPoint(t, pl1Screen(0, 6), zigUnwrap(t, s.PointFromPin(PointTagScreen, zigUnwrap(t, s.Pin(Point{Tag: PointTagViewport})))))
}

// Zig: test "PageList grow fit in capacity"
func TestPl1_PageList_grow_fit_in_capacity(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// So we know we're using capacity to grow
	last := &s.pages.last.data
	expect(t, last.size.rows < last.capacity.rows, "last.size.rows < last.capacity.rows")

	// Grow
	expect(t, pl1Grow(t, &s) == nil, "try s.grow() == null")
	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagActive})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 1), pt)
	}
}

// Zig: test "PageList grow allocate"
func TestPl1_PageList_grow_allocate(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Grow to capacity
	last_node := s.pages.last
	last := &s.pages.last.data
	for range uint(last.capacity.rows) - uint(last.size.rows) {
		expect(t, pl1Grow(t, &s) == nil, "try s.grow() == null")
	}

	// Grow, should allocate
	new := pl1Grow(t, &s)
	expect(t, new != nil, "(try s.grow()).? != null")
	expect(t, s.pages.last == new, "s.pages.last.? == new")
	expect(t, last_node.next == new, "last_node.next.? == new")
	{
		cell := zigUnwrap(t, s.GetCell(pl1Active(0, uint32(s.rows-1))))
		expect(t, cell.node == new, "cell.node == new")
		expectEqualPoint(t, pl1Screen(0, uint32(last.capacity.rows)), cell.ScreenPoint())
	}
}

// Zig: test "PageList grow prune scrollback"
func TestPl1_PageList_grow_prune_scrollback(t *testing.T) {
	testStdCapacity(t)

	// Use std_size to limit scrollback so pruning is triggered.
	s := pl1Init(t, 80, 24, Some(pageListStdSize()))
	defer s.Deinit()

	// Grow to capacity
	page1_node := s.pages.last
	page1 := page1_node.data
	for range uint(page1.capacity.rows) - uint(page1.size.rows) {
		expect(t, pl1Grow(t, &s) == nil, "try s.grow() == null")
	}

	// Grow and allocate one more page. Then fill that page up.
	page2_node := pl1Grow(t, &s)
	expect(t, page2_node != nil, "(try s.grow()).? != null")
	page2 := page2_node.data
	for range uint(page2.capacity.rows) - uint(page2.size.rows) {
		expect(t, pl1Grow(t, &s) == nil, "try s.grow() == null")
	}

	// Get our page size
	old_page_size := s.page_size

	// Create a tracked pin in the first page
	p, err := s.TrackPin(zigUnwrap(t, s.Pin(Point{Tag: PointTagScreen})))
	zigTry(t, err)
	defer s.UntrackPin(p)
	expect(t, p.node == s.pages.first, "p.node == s.pages.first.?")

	// Scroll back to create a pinned viewport (not active)
	pin_y := page1.capacity.rows / 2
	s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: zigUnwrap(t, s.Pin(pl1Screen(0, uint32(pin_y))))})
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")

	// Get the scrollbar state to populate the cache
	scrollbar_before := s.Scrollbar()
	expectEqual(t, uint(pin_y), scrollbar_before.offset)

	// Next should create a new page, but it should reuse our first
	// page since we're at max size.
	new := pl1Grow(t, &s)
	expect(t, new != nil, "(try s.grow()).? != null")
	expect(t, s.pages.last == new, "s.pages.last.? == new")
	expectEqual(t, s.page_size, old_page_size)

	// Our first should now be page2 and our last should be page1
	expectEqual(t, page2_node, s.pages.first)
	expectEqual(t, page1_node, s.pages.last)

	// Our tracked pin should point to the top-left of the first page
	expect(t, p.node == s.pages.first, "p.node == s.pages.first.?")
	expect(t, p.x == 0, "p.x == 0")
	expect(t, p.y == 0, "p.y == 0")
	expect(t, p.garbage, "p.garbage")

	// Verify the viewport offset cache was invalidated. After pruning,
	// the offset should have changed because we removed rows from
	// the beginning.
	{
		scrollbar_after := s.Scrollbar()
		rows_pruned := page1.capacity.rows
		var expected_offset uint
		if pin_y >= rows_pruned {
			expected_offset = uint(pin_y - rows_pruned)
		} else {
			expected_offset = 0
		}
		expectEqual(t, expected_offset, scrollbar_after.offset)
	}
}

// Zig: test "PageList grow prune scrollback with viewport pin not in pruned page"
func TestPl1_PageList_grow_prune_scrollback_with_viewport_pin_not_in_pruned_page(t *testing.T) {
	testStdCapacity(t)

	// Use std_size to limit scrollback so pruning is triggered.
	s := pl1Init(t, 80, 24, Some(pageListStdSize()))
	defer s.Deinit()

	// Grow to capacity of first page
	page1_node := s.pages.last
	page1 := page1_node.data
	for range uint(page1.capacity.rows) - uint(page1.size.rows) {
		expect(t, pl1Grow(t, &s) == nil, "try s.grow() == null")
	}

	// Grow and allocate second page, then fill it up
	page2_node := pl1Grow(t, &s)
	expect(t, page2_node != nil, "(try s.grow()).? != null")
	page2 := page2_node.data
	for range uint(page2.capacity.rows) - uint(page2.size.rows) {
		expect(t, pl1Grow(t, &s) == nil, "try s.grow() == null")
	}

	// Get our page size
	old_page_size := s.page_size

	// Scroll back to create a pinned viewport in page2 (NOT page1)
	// This is the key difference from the previous test - the viewport
	// pin is NOT in the page that will be pruned.
	pin_y := page1.capacity.rows + 5
	s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: zigUnwrap(t, s.Pin(pl1Screen(0, uint32(pin_y))))})
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expect(t, s.viewport_pin.node == page2_node, "s.viewport_pin.node == page2_node")

	// Get the scrollbar state to populate the cache
	scrollbar_before := s.Scrollbar()
	expectEqual(t, uint(pin_y), scrollbar_before.offset)

	// Next grow will trigger pruning of the first page.
	// The viewport_pin.node is page2, not page1, so it won't be moved
	// by the pin update loop, but the cached offset still needs to be
	// invalidated because rows were removed from the beginning.
	new := pl1Grow(t, &s)
	expect(t, new != nil, "(try s.grow()).? != null")
	expect(t, s.pages.last == new, "s.pages.last.? == new")
	expectEqual(t, s.page_size, old_page_size)

	// Our first should now be page2 (page1 was pruned)
	expectEqual(t, page2_node, s.pages.first)

	// The viewport pin should still be on page2, unchanged
	expect(t, s.viewport_pin.node == page2_node, "s.viewport_pin.node == page2_node")

	// Verify the viewport offset cache was invalidated/updated.
	// After pruning, the offset should have decreased by the number
	// of rows that were pruned.
	scrollbar_after := s.Scrollbar()
	rows_pruned := page1.capacity.rows
	expected_offset := pin_y - rows_pruned
	expectEqual(t, uint(expected_offset), scrollbar_after.offset)
}

// Zig: test "PageList eraseRows invalidates viewport offset cache"
func TestPl1_PageList_eraseRows_invalidates_viewport_offset_cache(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Grow so we take up several pages worth of history
	page := &s.pages.last.data
	{
		cur_page := s.pages.last
		for range uint(page.capacity.rows) * 3 {
			if new_page := pl1Grow(t, &s); new_page != nil {
				cur_page = new_page
			}
		}
		_ = cur_page
	}

	// Scroll back to create a pinned viewport somewhere in the middle
	// of the scrollback
	pin_y := page.capacity.rows
	s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: zigUnwrap(t, s.Pin(pl1Screen(0, uint32(pin_y))))})
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: uint(pin_y),
		len:    uint(s.rows),
	}, s.Scrollbar())

	// Erase some history rows BEFORE the viewport pin.
	// This removes rows from before our pin, which changes its absolute
	// offset from the top, but the cache is not invalidated.
	rows_to_erase := page.capacity.rows / 2
	s.EraseRows(
		Point{Tag: PointTagHistory},
		Some(Point{Tag: PointTagHistory, History: PointCoordinate{y: uint32(rows_to_erase - 1)}}),
	)

	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: uint(pin_y - rows_to_erase),
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList eraseRow invalidates viewport offset cache"
func TestPl1_PageList_eraseRow_invalidates_viewport_offset_cache(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Grow so we take up several pages worth of history
	page := &s.pages.last.data
	{
		cur_page := s.pages.last
		for range uint(page.capacity.rows) * 3 {
			if new_page := pl1Grow(t, &s); new_page != nil {
				cur_page = new_page
			}
		}
		_ = cur_page
	}

	// Scroll back to create a pinned viewport somewhere in the middle
	// of the scrollback
	pin_y := page.capacity.rows
	s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: zigUnwrap(t, s.Pin(pl1Screen(0, uint32(pin_y))))})
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: uint(pin_y),
		len:    uint(s.rows),
	}, s.Scrollbar())

	// Erase a single row from the history BEFORE the viewport pin.
	// This removes one row from before our pin, which changes its absolute
	// offset from the top by 1, but the cache is not invalidated.
	zigTry(t, s.EraseRow(pl1History(0, 0)))

	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: uint(pin_y) - 1,
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList eraseRowBounded invalidates viewport offset cache"
func TestPl1_PageList_eraseRowBounded_invalidates_viewport_offset_cache(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Grow so we take up several pages worth of history
	page := &s.pages.last.data
	{
		cur_page := s.pages.last
		for range uint(page.capacity.rows) * 3 {
			if new_page := pl1Grow(t, &s); new_page != nil {
				cur_page = new_page
			}
		}
		_ = cur_page
	}

	// Scroll back to create a pinned viewport somewhere in the middle
	// of the scrollback
	const pin_y uint16 = 4
	s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: zigUnwrap(t, s.Pin(pl1Screen(0, uint32(pin_y))))})
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: uint(pin_y),
		len:    uint(s.rows),
	}, s.Scrollbar())

	// Erase a row from the history BEFORE the viewport pin with a bounded
	// shift. This removes one row from before our pin, which changes its
	// absolute offset from the top by 1, but the cache is not invalidated.
	zigTry(t, s.EraseRowBounded(pl1History(0, 0), 10))

	// Verify the scrollbar reflects the change (offset decreased by 1)
	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: uint(pin_y) - 1,
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList eraseRowBounded multi-page invalidates viewport offset cache"
func TestPl1_PageList_eraseRowBounded_multi_page_invalidates_viewport_offset_cache(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Grow so we take up several pages worth of history
	page := &s.pages.last.data
	{
		cur_page := s.pages.last
		for range uint(page.capacity.rows) * 3 {
			if new_page := pl1Grow(t, &s); new_page != nil {
				cur_page = new_page
			}
		}
		_ = cur_page
	}

	// Scroll back to create a pinned viewport somewhere in the middle
	// of the scrollback, after the first page
	pin_y := page.capacity.rows + 1
	s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: zigUnwrap(t, s.Pin(pl1Screen(0, uint32(pin_y))))})
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: uint(pin_y),
		len:    uint(s.rows),
	}, s.Scrollbar())

	// Erase a row from the beginning of history with a limit that spans
	// across multiple pages. This ensures we hit the code path where
	// eraseRowBounded finds the limit boundary in a subsequent page.
	limit := uint(page.capacity.rows) + 10
	zigTry(t, s.EraseRowBounded(pl1History(0, 0), limit))

	// Verify the scrollbar reflects the change (offset decreased by 1)
	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: uint(pin_y) - 1,
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList eraseRowBounded full page shift invalidates viewport offset cache"
func TestPl1_PageList_eraseRowBounded_full_page_shift_invalidates_viewport_offset_cache(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Grow so we take up several pages worth of history
	page := &s.pages.last.data
	{
		cur_page := s.pages.last
		for range uint(page.capacity.rows) * 4 {
			if new_page := pl1Grow(t, &s); new_page != nil {
				cur_page = new_page
			}
		}
		_ = cur_page
	}

	// Scroll back to create a pinned viewport somewhere well beyond
	// the first two pages
	const pin_y = 5
	s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: zigUnwrap(t, s.Pin(pl1Screen(0, pin_y)))})
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: pin_y,
		len:    uint(s.rows),
	}, s.Scrollbar())

	// Erase a row from the beginning of history with a limit that is
	// larger than multiple full pages. This ensures we hit the code path
	// where eraseRowBounded continues looping through entire pages,
	// rotating all rows in each page until it reaches the limit or
	// runs out of pages.
	limit := uint(page.capacity.rows)*2 + 10
	zigTry(t, s.EraseRowBounded(pl1History(0, 0), limit))

	// Verify the scrollbar reflects the change (offset decreased by 1)
	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: pin_y - 1,
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList eraseRowBounded exhausts pages invalidates viewport offset cache"
func TestPl1_PageList_eraseRowBounded_exhausts_pages_invalidates_viewport_offset_cache(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Grow so we take up several pages worth of history
	page := &s.pages.last.data
	{
		cur_page := s.pages.last
		for range uint(page.capacity.rows) * 3 {
			if new_page := pl1Grow(t, &s); new_page != nil {
				cur_page = new_page
			}
		}
		_ = cur_page
	}

	// Our total rows should include history
	total_rows_before := s.totalRows()
	expect(t, total_rows_before > uint(s.rows), "total_rows_before > s.rows")

	// Scroll back to create a pinned viewport somewhere in the history,
	// well after the erase will complete
	pin_y := page.capacity.rows*2 + 10
	s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: zigUnwrap(t, s.Pin(pl1Screen(0, uint32(pin_y))))})
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: uint(pin_y),
		len:    uint(s.rows),
	}, s.Scrollbar())

	// Erase a row from the beginning of history with a limit that is
	// LARGER than all remaining pages combined. This ensures we exhaust
	// all pages in the while loop and reach the cleanup code after the loop.
	limit := total_rows_before * 2
	zigTry(t, s.EraseRowBounded(pl1History(0, 0), limit))

	// Verify the scrollbar reflects the change (offset decreased by 1)
	expectEqual(t, PageListScrollbar{
		total:  s.total_rows,
		offset: uint(pin_y) - 1,
		len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList increaseCapacity to increase styles"
func TestPl1_PageList_increaseCapacity_to_increase_styles(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 2, 2, Some[uint64](0))
	defer s.Deinit()

	original_styles_cap := s.pages.first.data.capacity.styles

	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		// Write all our data so we can assert its the same after
		for y := range uint(s.rows) {
			for x := range uint(s.cols) {
				rac := page.GetRowAndCell(x, y)
				*rac.cell = pl1Codepoint(uint32(x))
			}
		}
	}

	// Increase our styles
	_, err := s.IncreaseCapacity(s.pages.first, Some(PageListIncreaseCapacityStyles))
	zigTry(t, err)

	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		// Verify capacity doubled
		expectEqual(t,
			original_styles_cap*2,
			page.capacity.styles,
		)

		// Verify data preserved
		for y := range uint(s.rows) {
			for x := range uint(s.cols) {
				rac := page.GetRowAndCell(x, y)
				expectEqual(t,
					uint32(x),
					rac.cell.content.codepoint,
				)
			}
		}
	}
}

// Zig: test "PageList increaseCapacity to increase graphemes"
func TestPl1_PageList_increaseCapacity_to_increase_graphemes(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 2, 2, Some[uint64](0))
	defer s.Deinit()

	original_cap := s.pages.first.data.capacity.grapheme_bytes

	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		for y := range uint(s.rows) {
			for x := range uint(s.cols) {
				rac := page.GetRowAndCell(x, y)
				*rac.cell = pl1Codepoint(uint32(x))
			}
		}
	}

	_, err := s.IncreaseCapacity(s.pages.first, Some(PageListIncreaseCapacityGraphemeBytes))
	zigTry(t, err)

	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		expectEqual(t, original_cap*2, page.capacity.grapheme_bytes)

		for y := range uint(s.rows) {
			for x := range uint(s.cols) {
				rac := page.GetRowAndCell(x, y)
				expectEqual(t,
					uint32(x),
					rac.cell.content.codepoint,
				)
			}
		}
	}
}

// Zig: test "PageList increaseCapacity to increase hyperlinks"
func TestPl1_PageList_increaseCapacity_to_increase_hyperlinks(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 2, 2, Some[uint64](0))
	defer s.Deinit()

	original_cap := s.pages.first.data.capacity.hyperlink_bytes

	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		for y := range uint(s.rows) {
			for x := range uint(s.cols) {
				rac := page.GetRowAndCell(x, y)
				*rac.cell = pl1Codepoint(uint32(x))
			}
		}
	}

	_, err := s.IncreaseCapacity(s.pages.first, Some(PageListIncreaseCapacityHyperlinkBytes))
	zigTry(t, err)

	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		expectEqual(t, original_cap*2, page.capacity.hyperlink_bytes)

		for y := range uint(s.rows) {
			for x := range uint(s.cols) {
				rac := page.GetRowAndCell(x, y)
				expectEqual(t,
					uint32(x),
					rac.cell.content.codepoint,
				)
			}
		}
	}
}

// Zig: test "PageList increaseCapacity to increase string_bytes"
func TestPl1_PageList_increaseCapacity_to_increase_string_bytes(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 2, 2, Some[uint64](0))
	defer s.Deinit()

	original_cap := s.pages.first.data.capacity.string_bytes

	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		for y := range uint(s.rows) {
			for x := range uint(s.cols) {
				rac := page.GetRowAndCell(x, y)
				*rac.cell = pl1Codepoint(uint32(x))
			}
		}
	}

	_, err := s.IncreaseCapacity(s.pages.first, Some(PageListIncreaseCapacityStringBytes))
	zigTry(t, err)

	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		expectEqual(t, original_cap*2, page.capacity.string_bytes)

		for y := range uint(s.rows) {
			for x := range uint(s.cols) {
				rac := page.GetRowAndCell(x, y)
				expectEqual(t,
					uint32(x),
					rac.cell.content.codepoint,
				)
			}
		}
	}
}

// Zig: test "PageList increaseCapacity tracked pins"
func TestPl1_PageList_increaseCapacity_tracked_pins(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 2, 2, Some[uint64](0))
	defer s.Deinit()

	// Create a tracked pin on the first page
	tracked, err := s.TrackPin(zigUnwrap(t, s.Pin(pl1Active(1, 1))))
	zigTry(t, err)
	defer s.UntrackPin(tracked)

	old_node := s.pages.first
	expectEqual(t, old_node, tracked.node)

	// Increase capacity
	new_node, err := s.IncreaseCapacity(s.pages.first, Some(PageListIncreaseCapacityStyles))
	zigTry(t, err)

	// Pin should now point to the new node
	expectEqual(t, new_node, tracked.node)
	expectEqual(t, SizeCellCountInt(1), tracked.x)
	expectEqual(t, SizeCellCountInt(1), tracked.y)
}

// Zig: test "PageList increaseCapacity returns OutOfSpace at max capacity"
func TestPl1_PageList_increaseCapacity_returns_OutOfSpace_at_max_capacity(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 2, 2, Some[uint64](0))
	defer s.Deinit()

	// Keep increasing styles capacity until we get OutOfSpace
	const max_styles = math.MaxUint16
	for {
		_, err := s.IncreaseCapacity(
			s.pages.first,
			Some(PageListIncreaseCapacityStyles),
		)
		if err != nil {
			// Before OutOfSpace, we should have reached maxInt
			pl1ExpectError(t, ErrOutOfSpace, err)
			expectEqual(t, SizeStyleCountInt(max_styles), s.pages.first.data.capacity.styles)
			break
		}
	}
}

// Zig: test "PageList increaseCapacity after col shrink"
func TestPl1_PageList_increaseCapacity_after_col_shrink(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 2, Some[uint64](0))
	defer s.Deinit()

	// Shrink columns
	zigTry(t, s.Resize(PageListResize{cols: Some[SizeCellCountInt](5), reflow: false}))
	expectEqual(t, SizeCellCountInt(5), s.cols)

	{
		page := &s.pages.first.data
		expectEqual(t, SizeCellCountInt(5), page.size.cols)
		expect(t, page.capacity.cols >= 10, "page.capacity.cols >= 10")
	}

	// Increase capacity
	_, err := s.IncreaseCapacity(s.pages.first, Some(PageListIncreaseCapacityStyles))
	zigTry(t, err)

	{
		page := &s.pages.first.data
		// size.cols should still be 5, not reverted to capacity.cols
		expectEqual(t, SizeCellCountInt(5), page.size.cols)
		expectEqual(t, SizeCellCountInt(5), s.cols)
	}
}

// Zig: test "PageList increaseCapacity multi-page"
func TestPl1_PageList_increaseCapacity_multi_page(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Grow to create a second page
	page1_node := s.pages.last
	page1_node.data.PauseIntegrityChecks(true)
	for range uint(page1_node.data.capacity.rows) - uint(page1_node.data.size.rows) {
		expect(t, pl1Grow(t, &s) == nil, "try s.grow() == null")
	}
	page1_node.data.PauseIntegrityChecks(false)
	expect(t, pl1Grow(t, &s) != nil, "try s.grow() != null")

	// Now we have two pages
	expect(t, s.pages.first != s.pages.last, "s.pages.first != s.pages.last")
	page2_node := s.pages.last

	page1_styles_cap := s.pages.first.data.capacity.styles
	page2_styles_cap := page2_node.data.capacity.styles

	// Increase capacity on the first page only
	_, err := s.IncreaseCapacity(s.pages.first, Some(PageListIncreaseCapacityStyles))
	zigTry(t, err)

	// First page capacity should be doubled
	expectEqual(t,
		page1_styles_cap*2,
		s.pages.first.data.capacity.styles,
	)

	// Second page should be unchanged
	expectEqual(t,
		page2_styles_cap,
		s.pages.last.data.capacity.styles,
	)
}

// Zig: test "PageList increaseCapacity preserves dirty flag"
func TestPl1_PageList_increaseCapacity_preserves_dirty_flag(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 2, 4, Some[uint64](0))
	defer s.Deinit()

	// Set page dirty flag and mark some rows as dirty
	page := &s.pages.first.data
	page.dirty = true

	rows := page.rows
	rows[0].dirty = true
	rows[1].dirty = false
	rows[2].dirty = true
	rows[3].dirty = false

	// Increase capacity
	new_node, err := s.IncreaseCapacity(s.pages.first, Some(PageListIncreaseCapacityStyles))
	zigTry(t, err)

	// The page dirty flag should be preserved
	expect(t, new_node.data.dirty, "new_node.data.dirty")

	// Row dirty flags should be preserved
	new_rows := new_node.data.rows
	expect(t, new_rows[0].dirty, "new_rows[0].dirty")
	expect(t, !new_rows[1].dirty, "!new_rows[1].dirty")
	expect(t, new_rows[2].dirty, "new_rows[2].dirty")
	expect(t, !new_rows[3].dirty, "!new_rows[3].dirty")
}

// Zig: test "PageList pageIterator single page"
func TestPl1_PageList_pageIterator_single_page(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// The viewport should be within a single page
	expect(t, s.pages.first.next == nil, "s.pages.first.?.next == null")

	// Iterate the active area
	it := s.PageIterator(PageListDirectionRightDown, Point{Tag: PointTagActive}, None[Point]())
	{
		chunk := zigUnwrap(t, it.Next())
		expect(t, chunk.node == s.pages.first, "chunk.node == s.pages.first.?")
		expectEqual(t, uint(0), uint(chunk.start))
		expectEqual(t, uint(s.rows), uint(chunk.end))
	}

	// Should only have one chunk
	expect(t, !it.Next().Valid, "it.next() == null")
}

// Zig: test "PageList pageIterator two pages"
func TestPl1_PageList_pageIterator_two_pages(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Grow to capacity
	page1_node := s.pages.last
	page1 := page1_node.data
	page1_node.data.PauseIntegrityChecks(true)
	for range uint(page1.capacity.rows) - uint(page1.size.rows) {
		expect(t, pl1Grow(t, &s) == nil, "try s.grow() == null")
	}
	page1_node.data.PauseIntegrityChecks(false)
	expect(t, pl1Grow(t, &s) != nil, "try s.grow() != null")

	// Iterate the active area
	it := s.PageIterator(PageListDirectionRightDown, Point{Tag: PointTagActive}, None[Point]())
	{
		chunk := zigUnwrap(t, it.Next())
		expect(t, chunk.node == s.pages.first, "chunk.node == s.pages.first.?")
		start := chunk.node.data.size.rows - s.rows + 1
		expectEqual(t, start, chunk.start)
		expectEqual(t, chunk.node.data.size.rows, chunk.end)
	}
	{
		chunk := zigUnwrap(t, it.Next())
		expect(t, chunk.node == s.pages.last, "chunk.node == s.pages.last.?")
		start := uint(0)
		expectEqual(t, start, uint(chunk.start))
		expectEqual(t, start+1, uint(chunk.end))
	}
	expect(t, !it.Next().Valid, "it.next() == null")
}

// Zig: test "PageList pageIterator history two pages"
func TestPl1_PageList_pageIterator_history_two_pages(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Grow to capacity
	page1_node := s.pages.last
	page1 := page1_node.data
	page1_node.data.PauseIntegrityChecks(true)
	for range uint(page1.capacity.rows) - uint(page1.size.rows) {
		expect(t, pl1Grow(t, &s) == nil, "try s.grow() == null")
	}
	page1_node.data.PauseIntegrityChecks(false)
	expect(t, pl1Grow(t, &s) != nil, "try s.grow() != null")

	// Iterate the active area
	it := s.PageIterator(PageListDirectionRightDown, Point{Tag: PointTagHistory}, None[Point]())
	{
		active_tl := s.GetTopLeft(PointTagActive)
		chunk := zigUnwrap(t, it.Next())
		expect(t, chunk.node == s.pages.first, "chunk.node == s.pages.first.?")
		start := uint(0)
		expectEqual(t, start, uint(chunk.start))
		expectEqual(t, active_tl.y, chunk.end)
	}
	expect(t, !it.Next().Valid, "it.next() == null")
}

// Zig: test "PageList pageIterator reverse single page"
func TestPl1_PageList_pageIterator_reverse_single_page(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// The viewport should be within a single page
	expect(t, s.pages.first.next == nil, "s.pages.first.?.next == null")

	// Iterate the active area
	it := s.PageIterator(PageListDirectionLeftUp, Point{Tag: PointTagActive}, None[Point]())
	{
		chunk := zigUnwrap(t, it.Next())
		expect(t, chunk.node == s.pages.first, "chunk.node == s.pages.first.?")
		expectEqual(t, uint(0), uint(chunk.start))
		expectEqual(t, uint(s.rows), uint(chunk.end))
	}

	// Should only have one chunk
	expect(t, !it.Next().Valid, "it.next() == null")
}

// Zig: test "PageList pageIterator reverse two pages"
func TestPl1_PageList_pageIterator_reverse_two_pages(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Grow to capacity
	page1_node := s.pages.last
	page1 := page1_node.data
	page1_node.data.PauseIntegrityChecks(true)
	for range uint(page1.capacity.rows) - uint(page1.size.rows) {
		expect(t, pl1Grow(t, &s) == nil, "try s.grow() == null")
	}
	page1_node.data.PauseIntegrityChecks(false)
	expect(t, pl1Grow(t, &s) != nil, "try s.grow() != null")

	// Iterate the active area
	it := s.PageIterator(PageListDirectionLeftUp, Point{Tag: PointTagActive}, None[Point]())
	count := uint(0)
	{
		chunk := zigUnwrap(t, it.Next())
		expect(t, chunk.node == s.pages.last, "chunk.node == s.pages.last.?")
		start := uint(0)
		expectEqual(t, start, uint(chunk.start))
		expectEqual(t, start+1, uint(chunk.end))
		count += uint(chunk.end - chunk.start)
	}
	{
		chunk := zigUnwrap(t, it.Next())
		expect(t, chunk.node == s.pages.first, "chunk.node == s.pages.first.?")
		start := chunk.node.data.size.rows - s.rows + 1
		expectEqual(t, start, chunk.start)
		expectEqual(t, chunk.node.data.size.rows, chunk.end)
		count += uint(chunk.end - chunk.start)
	}
	expect(t, !it.Next().Valid, "it.next() == null")
	expectEqual(t, uint(s.rows), count)
}

// Zig: test "PageList pageIterator reverse history two pages"
func TestPl1_PageList_pageIterator_reverse_history_two_pages(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Grow to capacity
	page1_node := s.pages.last
	page1 := page1_node.data
	page1_node.data.PauseIntegrityChecks(true)
	for range uint(page1.capacity.rows) - uint(page1.size.rows) {
		expect(t, pl1Grow(t, &s) == nil, "try s.grow() == null")
	}
	page1_node.data.PauseIntegrityChecks(false)
	expect(t, pl1Grow(t, &s) != nil, "try s.grow() != null")

	// Iterate the active area
	it := s.PageIterator(PageListDirectionLeftUp, Point{Tag: PointTagHistory}, None[Point]())
	{
		active_tl := s.GetTopLeft(PointTagActive)
		chunk := zigUnwrap(t, it.Next())
		expect(t, chunk.node == s.pages.first, "chunk.node == s.pages.first.?")
		start := uint(0)
		expectEqual(t, start, uint(chunk.start))
		expectEqual(t, active_tl.y, chunk.end)
	}
	expect(t, !it.Next().Valid, "it.next() == null")
}

// Zig: test "PageList cellIterator"
func TestPl1_PageList_cellIterator(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 2, 2, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	for y := range uint(s.rows) {
		for x := range uint(s.cols) {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pl1Codepoint(uint32(x))
		}
	}

	it := s.CellIterator(PageListDirectionRightDown, Point{Tag: PointTagScreen}, None[Point]())
	{
		p := zigUnwrap(t, it.Next())
		expectEqualPoint(t, pl1Screen(0, 0), zigUnwrap(t, s.PointFromPin(PointTagScreen, p)))
	}
	{
		p := zigUnwrap(t, it.Next())
		expectEqualPoint(t, pl1Screen(1, 0), zigUnwrap(t, s.PointFromPin(PointTagScreen, p)))
	}
	{
		p := zigUnwrap(t, it.Next())
		expectEqualPoint(t, pl1Screen(0, 1), zigUnwrap(t, s.PointFromPin(PointTagScreen, p)))
	}
	{
		p := zigUnwrap(t, it.Next())
		expectEqualPoint(t, pl1Screen(1, 1), zigUnwrap(t, s.PointFromPin(PointTagScreen, p)))
	}
	expect(t, !it.Next().Valid, "it.next() == null")
}

// Zig: test "PageList cellIterator reverse"
func TestPl1_PageList_cellIterator_reverse(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 2, 2, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	for y := range uint(s.rows) {
		for x := range uint(s.cols) {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pl1Codepoint(uint32(x))
		}
	}

	it := s.CellIterator(PageListDirectionLeftUp, Point{Tag: PointTagScreen}, None[Point]())
	{
		p := zigUnwrap(t, it.Next())
		expectEqualPoint(t, pl1Screen(1, 1), zigUnwrap(t, s.PointFromPin(PointTagScreen, p)))
	}
	{
		p := zigUnwrap(t, it.Next())
		expectEqualPoint(t, pl1Screen(0, 1), zigUnwrap(t, s.PointFromPin(PointTagScreen, p)))
	}
	{
		p := zigUnwrap(t, it.Next())
		expectEqualPoint(t, pl1Screen(1, 0), zigUnwrap(t, s.PointFromPin(PointTagScreen, p)))
	}
	{
		p := zigUnwrap(t, it.Next())
		expectEqualPoint(t, pl1Screen(0, 0), zigUnwrap(t, s.PointFromPin(PointTagScreen, p)))
	}
	expect(t, !it.Next().Valid, "it.next() == null")
}

// Zig: test "PageList promptIterator left_up"
func TestPl1_PageList_promptIterator_left_up(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 2, 20, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	// Normal prompt
	{
		rac := page.GetRowAndCell(0, 3)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}
	// Continuation
	{
		rac := page.GetRowAndCell(0, 6)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}
	{
		rac := page.GetRowAndCell(0, 7)
		rac.row.semantic_prompt = PageRowSemanticPromptPromptContinuation
	}
	{
		rac := page.GetRowAndCell(0, 8)
		rac.row.semantic_prompt = PageRowSemanticPromptPromptContinuation
	}
	// Broken continuation that has non-prompts in between
	{
		rac := page.GetRowAndCell(0, 12)
		rac.row.semantic_prompt = PageRowSemanticPromptPromptContinuation
	}

	it := s.PromptIterator(PageListDirectionLeftUp, Point{Tag: PointTagScreen}, None[Point]())
	{
		p := zigUnwrap(t, it.Next())
		expectEqualPoint(t, pl1Screen(0, 12), zigUnwrap(t, s.PointFromPin(PointTagScreen, p)))
	}
	{
		p := zigUnwrap(t, it.Next())
		expectEqualPoint(t, pl1Screen(0, 6), zigUnwrap(t, s.PointFromPin(PointTagScreen, p)))
	}
	{
		p := zigUnwrap(t, it.Next())
		expectEqualPoint(t, pl1Screen(0, 3), zigUnwrap(t, s.PointFromPin(PointTagScreen, p)))
	}
	expect(t, !it.Next().Valid, "it.next() == null")
}

// Zig: test "PageList promptIterator right_down"
func TestPl1_PageList_promptIterator_right_down(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 2, 20, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	// Normal prompt
	{
		rac := page.GetRowAndCell(0, 3)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}
	// Continuation (prompt on row 6, continuation on rows 7-8)
	{
		rac := page.GetRowAndCell(0, 6)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}
	{
		rac := page.GetRowAndCell(0, 7)
		rac.row.semantic_prompt = PageRowSemanticPromptPromptContinuation
	}
	{
		rac := page.GetRowAndCell(0, 8)
		rac.row.semantic_prompt = PageRowSemanticPromptPromptContinuation
	}
	// Broken continuation that has non-prompts in between (orphaned continuation at row 12)
	{
		rac := page.GetRowAndCell(0, 12)
		rac.row.semantic_prompt = PageRowSemanticPromptPromptContinuation
	}

	it := s.PromptIterator(PageListDirectionRightDown, Point{Tag: PointTagScreen}, None[Point]())
	{
		p := zigUnwrap(t, it.Next())
		expectEqualPoint(t, pl1Screen(0, 3), zigUnwrap(t, s.PointFromPin(PointTagScreen, p)))
	}
	{
		p := zigUnwrap(t, it.Next())
		expectEqualPoint(t, pl1Screen(0, 6), zigUnwrap(t, s.PointFromPin(PointTagScreen, p)))
	}
	{
		p := zigUnwrap(t, it.Next())
		expectEqualPoint(t, pl1Screen(0, 12), zigUnwrap(t, s.PointFromPin(PointTagScreen, p)))
	}
	expect(t, !it.Next().Valid, "it.next() == null")
}

// Zig: test "PageList promptIterator right_down continuation at start"
func TestPl1_PageList_promptIterator_right_down_continuation_at_start(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 2, 20, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Prompt continuation at row 0 (no prior rows - simulates trimmed scrollback)
	{
		rac := page.GetRowAndCell(0, 0)
		rac.row.semantic_prompt = PageRowSemanticPromptPromptContinuation
	}
	{
		rac := page.GetRowAndCell(0, 1)
		rac.row.semantic_prompt = PageRowSemanticPromptPromptContinuation
	}
	// Normal prompt later
	{
		rac := page.GetRowAndCell(0, 5)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}

	it := s.PromptIterator(PageListDirectionRightDown, Point{Tag: PointTagScreen}, None[Point]())
	{
		// Should return the first continuation line since there's no prior prompt
		p := zigUnwrap(t, it.Next())
		expectEqualPoint(t, pl1Screen(0, 0), zigUnwrap(t, s.PointFromPin(PointTagScreen, p)))
	}
	{
		p := zigUnwrap(t, it.Next())
		expectEqualPoint(t, pl1Screen(0, 5), zigUnwrap(t, s.PointFromPin(PointTagScreen, p)))
	}
	expect(t, !it.Next().Valid, "it.next() == null")
}

// Zig: test "PageList promptIterator right_down with prompt before continuation"
func TestPl1_PageList_promptIterator_right_down_with_prompt_before_continuation(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 2, 20, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Prompt on row 2, continuation on rows 3-4
	// Starting iteration from row 3 should still find the prompt at row 2
	{
		rac := page.GetRowAndCell(0, 2)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}
	{
		rac := page.GetRowAndCell(0, 3)
		rac.row.semantic_prompt = PageRowSemanticPromptPromptContinuation
	}
	{
		rac := page.GetRowAndCell(0, 4)
		rac.row.semantic_prompt = PageRowSemanticPromptPromptContinuation
	}

	// Start iteration from row 3 (middle of the continuation)
	// Since we start on a continuation line, we treat it as the prompt start
	// (handles case where scrollback pruned the actual prompt)
	it := s.PromptIterator(PageListDirectionRightDown, pl1Screen(0, 3), None[Point]())
	{
		p := zigUnwrap(t, it.Next())
		// Returns row 3 since that's the first prompt-related line we encounter
		expectEqualPoint(t, pl1Screen(0, 3), zigUnwrap(t, s.PointFromPin(PointTagScreen, p)))
	}
	expect(t, !it.Next().Valid, "it.next() == null")
}

// Zig: test "PageList promptIterator right_down limit inclusive"
func TestPl1_PageList_promptIterator_right_down_limit_inclusive(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 2, 20, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Prompt on row 5
	{
		rac := page.GetRowAndCell(0, 5)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}
	// Prompt on row 10
	{
		rac := page.GetRowAndCell(0, 10)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}

	// Iterate with limit at row 5 (the prompt row) - should include it
	it := s.PromptIterator(PageListDirectionRightDown, Point{Tag: PointTagScreen}, Some(pl1Screen(0, 5)))
	{
		p := zigUnwrap(t, it.Next())
		expectEqualPoint(t, pl1Screen(0, 5), zigUnwrap(t, s.PointFromPin(PointTagScreen, p)))
	}
	expect(t, !it.Next().Valid, "it.next() == null")
}

// Zig: test "PageList promptIterator left_up limit inclusive"
func TestPl1_PageList_promptIterator_left_up_limit_inclusive(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 2, 20, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Prompt on row 5
	{
		rac := page.GetRowAndCell(0, 5)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}
	// Prompt on row 10
	{
		rac := page.GetRowAndCell(0, 10)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}

	// Iterate with limit at row 10 (the prompt row) - should include it
	// tl_pt is the limit (upper bound), bl_pt is the start point for left_up
	it := s.PromptIterator(PageListDirectionLeftUp, pl1Screen(0, 10), Some(pl1Screen(0, 15)))
	{
		p := zigUnwrap(t, it.Next())
		expectEqualPoint(t, pl1Screen(0, 10), zigUnwrap(t, s.PointFromPin(PointTagScreen, p)))
	}
	expect(t, !it.Next().Valid, "it.next() == null")
}

// Zig: test "PageList highlightSemanticContent prompt"
func TestPl1_PageList_highlightSemanticContent_prompt(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 20, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Prompt on row 5
	{
		rac := page.GetRowAndCell(0, 5)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt

		// Start the prompt for the first 5 cols
		for x := range uint(5) {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('A', PageCellSemanticContentPrompt)
		}

		// Next 3 let's make input
		for x := uint(5); x < 8; x++ {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('B', PageCellSemanticContentInput)
		}
	}
	// Prompt on row 10
	{
		rac := page.GetRowAndCell(0, 10)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}

	hl := zigUnwrap(t, s.HighlightSemanticContent(
		zigUnwrap(t, s.Pin(pl1Screen(2, 5))),
		PageCellSemanticContentPrompt,
	))
	expectEqualPoint(t, pl1Screen(0, 5), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.start)))
	expectEqualPoint(t, pl1Screen(7, 5), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.end)))
}

// Zig: test "PageList highlightSemanticContent prompt with output"
func TestPl1_PageList_highlightSemanticContent_prompt_with_output(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 20, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Prompt on row 5
	{
		rac := page.GetRowAndCell(0, 5)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt

		// First 3 cols are prompt
		for x := range uint(3) {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('$', PageCellSemanticContentPrompt)
		}

		// Next 4 are input
		for x := uint(3); x < 7; x++ {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('l', PageCellSemanticContentInput)
		}

		// Rest is output (shouldn't be included in prompt highlight)
		for x := uint(7); x < 10; x++ {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('o', PageCellSemanticContentOutput)
		}
	}
	// Prompt on row 10
	{
		rac := page.GetRowAndCell(0, 10)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}

	// Highlighting from prompt should include prompt and input, but stop at output
	hl := zigUnwrap(t, s.HighlightSemanticContent(
		zigUnwrap(t, s.Pin(pl1Screen(0, 5))),
		PageCellSemanticContentPrompt,
	))
	expectEqualPoint(t, pl1Screen(0, 5), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.start)))
	expectEqualPoint(t, pl1Screen(6, 5), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.end)))
}

// Zig: test "PageList highlightSemanticContent prompt multiline"
func TestPl1_PageList_highlightSemanticContent_prompt_multiline(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 20, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Prompt starts on row 5
	{
		rac := page.GetRowAndCell(0, 5)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt

		// First row is all prompt
		for x := range uint(10) {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('$', PageCellSemanticContentPrompt)
		}
	}
	// Row 6 continues with input
	{
		for x := range uint(5) {
			cell := page.GetRowAndCell(x, 6).cell
			*cell = pl1CellSemantic('c', PageCellSemanticContentInput)
		}
	}
	// Prompt on row 10
	{
		rac := page.GetRowAndCell(0, 10)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}

	// Highlighting should span both rows
	hl := zigUnwrap(t, s.HighlightSemanticContent(
		zigUnwrap(t, s.Pin(pl1Screen(2, 5))),
		PageCellSemanticContentPrompt,
	))
	expectEqualPoint(t, pl1Screen(0, 5), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.start)))
	expectEqualPoint(t, pl1Screen(4, 6), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.end)))
}

// Zig: test "PageList highlightSemanticContent prompt only"
func TestPl1_PageList_highlightSemanticContent_prompt_only(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 20, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Prompt on row 5 with only prompt content (no input)
	{
		rac := page.GetRowAndCell(0, 5)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt

		for x := range uint(5) {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('$', PageCellSemanticContentPrompt)
		}
	}
	// Prompt on row 10
	{
		rac := page.GetRowAndCell(0, 10)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}

	// Highlighting should only include the prompt cells
	hl := zigUnwrap(t, s.HighlightSemanticContent(
		zigUnwrap(t, s.Pin(pl1Screen(0, 5))),
		PageCellSemanticContentPrompt,
	))
	expectEqualPoint(t, pl1Screen(0, 5), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.start)))
	expectEqualPoint(t, pl1Screen(4, 5), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.end)))
}

// Zig: test "PageList highlightSemanticContent prompt to end of screen"
func TestPl1_PageList_highlightSemanticContent_prompt_to_end_of_screen(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 20, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Single prompt on row 15, no following prompt
	{
		rac := page.GetRowAndCell(0, 15)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt

		for x := range uint(3) {
			cell := page.GetRowAndCell(x, 15).cell
			*cell = pl1CellSemantic('$', PageCellSemanticContentPrompt)
		}

		for x := uint(3); x < 8; x++ {
			cell := page.GetRowAndCell(x, 15).cell
			*cell = pl1CellSemantic('c', PageCellSemanticContentInput)
		}
	}

	// Highlighting should include prompt and input up to column 7
	hl := zigUnwrap(t, s.HighlightSemanticContent(
		zigUnwrap(t, s.Pin(pl1Screen(0, 15))),
		PageCellSemanticContentPrompt,
	))
	expectEqualPoint(t, pl1Screen(0, 15), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.start)))
	expectEqualPoint(t, pl1Screen(7, 15), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.end)))
}

// Zig: test "PageList highlightSemanticContent input basic"
func TestPl1_PageList_highlightSemanticContent_input_basic(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 20, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Prompt on row 5
	{
		rac := page.GetRowAndCell(0, 5)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt

		// First 3 cols are prompt
		for x := range uint(3) {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('$', PageCellSemanticContentPrompt)
		}

		// Next 5 are input
		for x := uint(3); x < 8; x++ {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('l', PageCellSemanticContentInput)
		}
	}
	// Prompt on row 10
	{
		rac := page.GetRowAndCell(0, 10)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}

	// Highlighting input should only include input cells
	hl := zigUnwrap(t, s.HighlightSemanticContent(
		zigUnwrap(t, s.Pin(pl1Screen(0, 5))),
		PageCellSemanticContentInput,
	))
	expectEqualPoint(t, pl1Screen(3, 5), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.start)))
	expectEqualPoint(t, pl1Screen(7, 5), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.end)))
}

// Zig: test "PageList highlightSemanticContent input with output"
func TestPl1_PageList_highlightSemanticContent_input_with_output(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 20, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Prompt on row 5
	{
		rac := page.GetRowAndCell(0, 5)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt

		// First 2 cols are prompt
		for x := range uint(2) {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('$', PageCellSemanticContentPrompt)
		}

		// Next 3 are input
		for x := uint(2); x < 5; x++ {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('c', PageCellSemanticContentInput)
		}

		// Rest is output
		for x := uint(5); x < 10; x++ {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('o', PageCellSemanticContentOutput)
		}
	}
	// Prompt on row 10
	{
		rac := page.GetRowAndCell(0, 10)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}

	// Highlighting input should stop at output
	hl := zigUnwrap(t, s.HighlightSemanticContent(
		zigUnwrap(t, s.Pin(pl1Screen(0, 5))),
		PageCellSemanticContentInput,
	))
	expectEqualPoint(t, pl1Screen(2, 5), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.start)))
	expectEqualPoint(t, pl1Screen(4, 5), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.end)))
}

// Zig: test "PageList highlightSemanticContent input multiline with continuation"
func TestPl1_PageList_highlightSemanticContent_input_multiline_with_continuation(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 20, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Prompt on row 5
	{
		rac := page.GetRowAndCell(0, 5)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt

		// First 2 cols are prompt
		for x := range uint(2) {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('$', PageCellSemanticContentPrompt)
		}

		// Rest is input
		for x := uint(2); x < 10; x++ {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('c', PageCellSemanticContentInput)
		}
	}
	// Row 6 has continuation prompt then more input
	{
		// Continuation prompt
		for x := range uint(2) {
			cell := page.GetRowAndCell(x, 6).cell
			*cell = pl1CellSemantic('>', PageCellSemanticContentPrompt)
		}

		// More input
		for x := uint(2); x < 6; x++ {
			cell := page.GetRowAndCell(x, 6).cell
			*cell = pl1CellSemantic('d', PageCellSemanticContentInput)
		}
	}
	// Prompt on row 10
	{
		rac := page.GetRowAndCell(0, 10)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}

	// Highlighting input should span both rows, skipping continuation prompts
	hl := zigUnwrap(t, s.HighlightSemanticContent(
		zigUnwrap(t, s.Pin(pl1Screen(0, 5))),
		PageCellSemanticContentInput,
	))
	expectEqualPoint(t, pl1Screen(2, 5), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.start)))
	expectEqualPoint(t, pl1Screen(5, 6), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.end)))
}

// Zig: test "PageList highlightSemanticContent input no input returns null"
func TestPl1_PageList_highlightSemanticContent_input_no_input_returns_null(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 20, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Prompt on row 5 with only prompt, then immediately output
	{
		rac := page.GetRowAndCell(0, 5)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt

		// First 3 cols are prompt
		for x := range uint(3) {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('$', PageCellSemanticContentPrompt)
		}

		// Rest is output (no input!)
		for x := uint(3); x < 10; x++ {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('o', PageCellSemanticContentOutput)
		}
	}
	// Prompt on row 10
	{
		rac := page.GetRowAndCell(0, 10)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}

	// Highlighting input should return null when there's no input
	hl := s.HighlightSemanticContent(
		zigUnwrap(t, s.Pin(pl1Screen(0, 5))),
		PageCellSemanticContentInput,
	)
	expect(t, !hl.Valid, "hl == null")
}

// Zig: test "PageList highlightSemanticContent input to end of screen"
func TestPl1_PageList_highlightSemanticContent_input_to_end_of_screen(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 20, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Single prompt on row 15, no following prompt
	{
		rac := page.GetRowAndCell(0, 15)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt

		for x := range uint(2) {
			cell := page.GetRowAndCell(x, 15).cell
			*cell = pl1CellSemantic('$', PageCellSemanticContentPrompt)
		}

		for x := uint(2); x < 7; x++ {
			cell := page.GetRowAndCell(x, 15).cell
			*cell = pl1CellSemantic('c', PageCellSemanticContentInput)
		}
	}

	// Highlighting input with no following prompt
	hl := zigUnwrap(t, s.HighlightSemanticContent(
		zigUnwrap(t, s.Pin(pl1Screen(0, 15))),
		PageCellSemanticContentInput,
	))
	expectEqualPoint(t, pl1Screen(2, 15), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.start)))
	expectEqualPoint(t, pl1Screen(6, 15), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.end)))
}

// Zig: test "PageList highlightSemanticContent input prompt only returns null"
func TestPl1_PageList_highlightSemanticContent_input_prompt_only_returns_null(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 20, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Prompt on row 5 with only prompt content, no input or output
	{
		rac := page.GetRowAndCell(0, 5)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt

		// All cells are prompt
		for x := range uint(10) {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('$', PageCellSemanticContentPrompt)
		}
	}
	// Mark rows 6-9 as prompt to ensure no input before next prompt
	{
		for y := uint(6); y < 10; y++ {
			for x := range uint(10) {
				cell := page.GetRowAndCell(x, y).cell
				cell.semantic_content = PageCellSemanticContentPrompt
			}
		}
	}
	// Prompt on row 10
	{
		rac := page.GetRowAndCell(0, 10)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}

	// Highlighting input should return null when there's only prompts
	hl := s.HighlightSemanticContent(
		zigUnwrap(t, s.Pin(pl1Screen(0, 5))),
		PageCellSemanticContentInput,
	)
	expect(t, !hl.Valid, "hl == null")
}

// Zig: test "PageList highlightSemanticContent output basic"
func TestPl1_PageList_highlightSemanticContent_output_basic(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 20, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Prompt on row 5
	{
		rac := page.GetRowAndCell(0, 5)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt

		// First 2 cols are prompt
		for x := range uint(2) {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('$', PageCellSemanticContentPrompt)
		}

		// Next 3 are input
		for x := uint(2); x < 5; x++ {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('l', PageCellSemanticContentInput)
		}

		// Cols 5-7 are output
		for x := uint(5); x < 8; x++ {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('o', PageCellSemanticContentOutput)
		}

		// Mark remaining cells as prompt to bound the output
		for x := uint(8); x < 10; x++ {
			cell := page.GetRowAndCell(x, 5).cell
			cell.semantic_content = PageCellSemanticContentPrompt
		}
	}
	// Prompt on row 10
	{
		rac := page.GetRowAndCell(0, 10)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}

	// Highlighting output should only include output cells
	hl := zigUnwrap(t, s.HighlightSemanticContent(
		zigUnwrap(t, s.Pin(pl1Screen(0, 5))),
		PageCellSemanticContentOutput,
	))
	expectEqualPoint(t, pl1Screen(5, 5), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.start)))
	expectEqualPoint(t, pl1Screen(7, 5), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.end)))
}

// Zig: test "PageList highlightSemanticContent output multiline"
func TestPl1_PageList_highlightSemanticContent_output_multiline(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 20, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Prompt on row 5
	{
		rac := page.GetRowAndCell(0, 5)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt

		// First 2 cols are prompt
		for x := range uint(2) {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('$', PageCellSemanticContentPrompt)
		}

		// Next 2 are input
		for x := uint(2); x < 4; x++ {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('l', PageCellSemanticContentInput)
		}

		// Rest of row 5 is output
		for x := uint(4); x < 10; x++ {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('o', PageCellSemanticContentOutput)
		}
	}
	// Row 6 is all output
	{
		for x := range uint(10) {
			cell := page.GetRowAndCell(x, 6).cell
			*cell = pl1CellSemantic('o', PageCellSemanticContentOutput)
		}
	}
	// Row 7 has partial output then input to bound it
	{
		for x := range uint(5) {
			cell := page.GetRowAndCell(x, 7).cell
			*cell = pl1CellSemantic('o', PageCellSemanticContentOutput)
		}
		for x := uint(5); x < 10; x++ {
			cell := page.GetRowAndCell(x, 7).cell
			cell.semantic_content = PageCellSemanticContentInput
		}
	}
	// Prompt on row 10
	{
		rac := page.GetRowAndCell(0, 10)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}

	// Highlighting output should span multiple rows
	hl := zigUnwrap(t, s.HighlightSemanticContent(
		zigUnwrap(t, s.Pin(pl1Screen(0, 5))),
		PageCellSemanticContentOutput,
	))
	expectEqualPoint(t, pl1Screen(4, 5), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.start)))
	expectEqualPoint(t, pl1Screen(4, 7), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.end)))
}

// Zig: test "PageList highlightSemanticContent output stops at next prompt"
func TestPl1_PageList_highlightSemanticContent_output_stops_at_next_prompt(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 20, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Prompt on row 5
	{
		rac := page.GetRowAndCell(0, 5)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt

		// First 2 cols are prompt
		for x := range uint(2) {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('$', PageCellSemanticContentPrompt)
		}

		// Next 2 are input
		for x := uint(2); x < 4; x++ {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('l', PageCellSemanticContentInput)
		}

		// Rest is output
		for x := uint(4); x < 10; x++ {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('o', PageCellSemanticContentOutput)
		}
	}
	// Row 6 has output then prompt starts
	{
		for x := range uint(3) {
			cell := page.GetRowAndCell(x, 6).cell
			*cell = pl1CellSemantic('o', PageCellSemanticContentOutput)
		}
		// Next prompt marker on same row
		for x := uint(3); x < 6; x++ {
			cell := page.GetRowAndCell(x, 6).cell
			*cell = pl1CellSemantic('$', PageCellSemanticContentPrompt)
		}
	}
	// Prompt on row 10
	{
		rac := page.GetRowAndCell(0, 10)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}

	// Highlighting output should stop before prompt/input
	hl := zigUnwrap(t, s.HighlightSemanticContent(
		zigUnwrap(t, s.Pin(pl1Screen(0, 5))),
		PageCellSemanticContentOutput,
	))
	expectEqualPoint(t, pl1Screen(4, 5), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.start)))
	expectEqualPoint(t, pl1Screen(2, 6), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.end)))
}

// Zig: test "PageList highlightSemanticContent output to end of screen"
func TestPl1_PageList_highlightSemanticContent_output_to_end_of_screen(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 20, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Single prompt on row 15, no following prompt
	{
		rac := page.GetRowAndCell(0, 15)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt

		for x := range uint(2) {
			cell := page.GetRowAndCell(x, 15).cell
			*cell = pl1CellSemantic('$', PageCellSemanticContentPrompt)
		}

		for x := uint(2); x < 4; x++ {
			cell := page.GetRowAndCell(x, 15).cell
			*cell = pl1CellSemantic('c', PageCellSemanticContentInput)
		}

		for x := uint(4); x < 10; x++ {
			cell := page.GetRowAndCell(x, 15).cell
			*cell = pl1CellSemantic('o', PageCellSemanticContentOutput)
		}
	}
	// Row 16 has output then prompt to bound it
	{
		for x := range uint(8) {
			cell := page.GetRowAndCell(x, 16).cell
			*cell = pl1CellSemantic('o', PageCellSemanticContentOutput)
		}
		for x := uint(8); x < 10; x++ {
			cell := page.GetRowAndCell(x, 16).cell
			cell.semantic_content = PageCellSemanticContentPrompt
		}
	}

	// Highlighting output with no following prompt
	hl := zigUnwrap(t, s.HighlightSemanticContent(
		zigUnwrap(t, s.Pin(pl1Screen(0, 15))),
		PageCellSemanticContentOutput,
	))
	expectEqualPoint(t, pl1Screen(4, 15), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.start)))
	expectEqualPoint(t, pl1Screen(7, 16), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.end)))
}

// Zig: test "PageList highlightSemanticContent output no output returns null"
func TestPl1_PageList_highlightSemanticContent_output_no_output_returns_null(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 20, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Prompt on row 5 with only prompt and input, no output
	{
		rac := page.GetRowAndCell(0, 5)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt

		// First 3 cols are prompt
		for x := range uint(3) {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('$', PageCellSemanticContentPrompt)
		}

		// Rest is input (must explicitly mark all cells to avoid default .output)
		for x := uint(3); x < 10; x++ {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('c', PageCellSemanticContentInput)
		}
	}
	// Mark rows 6-9 as input to ensure no output between prompts
	{
		for y := uint(6); y < 10; y++ {
			for x := range uint(10) {
				cell := page.GetRowAndCell(x, y).cell
				cell.semantic_content = PageCellSemanticContentInput
			}
		}
	}
	// Prompt on row 10 (no output between prompts)
	{
		rac := page.GetRowAndCell(0, 10)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}

	// Highlighting output should return null when there's no output
	hl := s.HighlightSemanticContent(
		zigUnwrap(t, s.Pin(pl1Screen(0, 5))),
		PageCellSemanticContentOutput,
	)
	expect(t, !hl.Valid, "hl == null")
}

// Zig: test "PageList highlightSemanticContent output skips empty cells"
func TestPl1_PageList_highlightSemanticContent_output_skips_empty_cells(t *testing.T) {
	testStdCapacity(t)

	// Tests that empty cells with default .output semantic content are
	// not selected as output. This can happen when a prompt/input line
	// doesn't fill the entire row - trailing cells have default .output.
	s := pl1Init(t, 10, 20, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Prompt on row 5 - only fills first 3 cells, rest are empty with default .output
	{
		rac := page.GetRowAndCell(0, 5)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt

		// First 3 cols are prompt with text
		for x := range uint(3) {
			cell := page.GetRowAndCell(x, 5).cell
			*cell = pl1CellSemantic('$', PageCellSemanticContentPrompt)
		}
		// Cells 3-9 are empty (codepoint = 0) with default .output semantic content
		// This simulates what happens when a short prompt is written
	}

	// Row 6 has input (short, doesn't fill line)
	{
		for x := range uint(4) {
			cell := page.GetRowAndCell(x, 6).cell
			*cell = pl1CellSemantic('l', PageCellSemanticContentInput)
		}
		// Cells 4-9 are empty with default .output
	}

	// Row 7-8 have actual output with text
	{
		for y := uint(7); y < 9; y++ {
			for x := range uint(5) {
				cell := page.GetRowAndCell(x, y).cell
				*cell = pl1CellSemantic('o', PageCellSemanticContentOutput)
			}
		}
	}

	// Prompt on row 10
	{
		rac := page.GetRowAndCell(0, 10)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}

	// Highlighting output should skip empty cells on rows 5-6 and find
	// the actual output starting at row 7
	hl := zigUnwrap(t, s.HighlightSemanticContent(
		zigUnwrap(t, s.Pin(pl1Screen(0, 5))),
		PageCellSemanticContentOutput,
	))
	// Output should start at row 7, not row 5 (where empty cells have default .output)
	expectEqualPoint(t, pl1Screen(0, 7), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.start)))
	expectEqualPoint(t, pl1Screen(4, 8), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.end)))
}

// Zig: test "PageList erase"
func TestPl1_PageList_erase(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	expectEqual(t, uint(1), s.TotalPages())

	// Grow so we take up at least 5 pages.
	page := &s.pages.last.data
	cur_page := s.pages.last
	cur_page.data.PauseIntegrityChecks(true)
	for range uint(page.capacity.rows) * 5 {
		if new_page := pl1Grow(t, &s); new_page != nil {
			cur_page.data.PauseIntegrityChecks(false)
			cur_page = new_page
			cur_page.data.PauseIntegrityChecks(true)
		}
	}
	cur_page.data.PauseIntegrityChecks(false)
	expectEqual(t, uint(6), s.TotalPages())

	// Our total rows should be large
	expect(t, s.total_rows > uint(s.rows), "s.total_rows > s.rows")

	// Erase the entire history, we should be back to just our active set.
	s.EraseRows(Point{Tag: PointTagHistory}, None[Point]())
	expectEqual(t, uint(s.rows), s.total_rows)

	// We should be back to just one page
	expectEqual(t, uint(1), s.TotalPages())
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
}

// Zig: test "PageList erase reaccounts page size"
func TestPl1_PageList_erase_reaccounts_page_size(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	start_size := s.page_size

	// Grow so we take up at least 5 pages.
	page := &s.pages.last.data
	cur_page := s.pages.last
	cur_page.data.PauseIntegrityChecks(true)
	for range uint(page.capacity.rows) * 5 {
		if new_page := pl1Grow(t, &s); new_page != nil {
			cur_page.data.PauseIntegrityChecks(false)
			cur_page = new_page
			cur_page.data.PauseIntegrityChecks(true)
		}
	}
	cur_page.data.PauseIntegrityChecks(false)
	expect(t, s.page_size > start_size, "s.page_size > start_size")

	// Erase the entire history, we should be back to just our active set.
	s.EraseRows(Point{Tag: PointTagHistory}, None[Point]())
	expectEqual(t, start_size, s.page_size)
}

// Zig: test "PageList erase row with tracked pin resets to top-left"
func TestPl1_PageList_erase_row_with_tracked_pin_resets_to_top_left(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Grow so we take up at least 5 pages.
	page := &s.pages.last.data
	cur_page := s.pages.last
	cur_page.data.PauseIntegrityChecks(true)
	for range uint(page.capacity.rows) * 5 {
		if new_page := pl1Grow(t, &s); new_page != nil {
			cur_page.data.PauseIntegrityChecks(false)
			cur_page = new_page
			cur_page.data.PauseIntegrityChecks(true)
		}
	}
	cur_page.data.PauseIntegrityChecks(false)

	// Our total rows should be large
	expect(t, s.total_rows > uint(s.rows), "s.total_rows > s.rows")

	// Put a tracked pin in the history
	p, err := s.TrackPin(zigUnwrap(t, s.Pin(Point{Tag: PointTagHistory})))
	zigTry(t, err)
	defer s.UntrackPin(p)

	// Erase the entire history, we should be back to just our active set.
	s.EraseRows(Point{Tag: PointTagHistory}, None[Point]())
	expectEqual(t, uint(s.rows), s.total_rows)

	// Our pin should move to the first page
	expectEqual(t, s.pages.first, p.node)
	expectEqual(t, uint(0), uint(p.y))
	expectEqual(t, uint(0), uint(p.x))
}

// Zig: test "PageList erase row with tracked pin shifts"
func TestPl1_PageList_erase_row_with_tracked_pin_shifts(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Put a tracked pin in the history
	p, err := s.TrackPin(zigUnwrap(t, s.Pin(pl1Active(2, 4))))
	zigTry(t, err)
	defer s.UntrackPin(p)

	// Erase only a few rows in our active
	s.EraseRows(Point{Tag: PointTagActive}, Some(pl1Active(0, 3)))
	expectEqual(t, uint(s.rows), s.total_rows)

	// Our pin should move to the first page
	expectEqual(t, s.pages.first, p.node)
	expectEqual(t, uint(0), uint(p.y))
	expectEqual(t, uint(2), uint(p.x))
}

// Zig: test "PageList erase row with tracked pin is erased"
func TestPl1_PageList_erase_row_with_tracked_pin_is_erased(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Put a tracked pin in the history
	p, err := s.TrackPin(zigUnwrap(t, s.Pin(pl1Active(2, 2))))
	zigTry(t, err)
	defer s.UntrackPin(p)

	// Erase the entire history, we should be back to just our active set.
	s.EraseRows(Point{Tag: PointTagActive}, Some(pl1Active(0, 3)))
	expectEqual(t, uint(s.rows), s.total_rows)

	// Our pin should move to the first page
	expectEqual(t, s.pages.first, p.node)
	expectEqual(t, uint(0), uint(p.y))
	expectEqual(t, uint(0), uint(p.x))
}

// Zig: test "PageList erase resets viewport to active if moves within active"
func TestPl1_PageList_erase_resets_viewport_to_active_if_moves_within_active(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Grow so we take up at least 5 pages.
	page := &s.pages.last.data
	cur_page := s.pages.last
	cur_page.data.PauseIntegrityChecks(true)
	for range uint(page.capacity.rows) * 5 {
		if new_page := pl1Grow(t, &s); new_page != nil {
			cur_page.data.PauseIntegrityChecks(false)
			cur_page = new_page
			cur_page.data.PauseIntegrityChecks(true)
		}
	}
	cur_page.data.PauseIntegrityChecks(false)

	// Move our viewport to the top
	s.Scroll(PageListScroll{Tag: PageListScrollDeltaRow, DeltaRow: -int(s.total_rows)})
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")

	// Erase the entire history, we should be back to just our active set.
	s.EraseRows(Point{Tag: PointTagHistory}, None[Point]())
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
}

// Zig: test "PageList erase resets viewport if inside erased page but not active"
func TestPl1_PageList_erase_resets_viewport_if_inside_erased_page_but_not_active(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Grow so we take up at least 5 pages.
	page := &s.pages.last.data
	cur_page := s.pages.last
	cur_page.data.PauseIntegrityChecks(true)
	for range uint(page.capacity.rows) * 5 {
		if new_page := pl1Grow(t, &s); new_page != nil {
			cur_page.data.PauseIntegrityChecks(false)
			cur_page = new_page
			cur_page.data.PauseIntegrityChecks(true)
		}
	}
	cur_page.data.PauseIntegrityChecks(false)

	// Move our viewport to the top
	s.Scroll(PageListScroll{Tag: PageListScrollDeltaRow, DeltaRow: -int(s.total_rows)})
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")

	// Erase the entire history, we should be back to just our active set.
	s.EraseRows(Point{Tag: PointTagHistory}, Some(pl1History(0, 2)))
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
}

// Zig: test "PageList erase resets viewport to active if top is inside active"
func TestPl1_PageList_erase_resets_viewport_to_active_if_top_is_inside_active(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Grow so we take up at least 5 pages.
	page := &s.pages.last.data
	cur_page := s.pages.last
	cur_page.data.PauseIntegrityChecks(true)
	for range uint(page.capacity.rows) * 5 {
		if new_page := pl1Grow(t, &s); new_page != nil {
			cur_page.data.PauseIntegrityChecks(false)
			cur_page = new_page
			cur_page.data.PauseIntegrityChecks(true)
		}
	}
	cur_page.data.PauseIntegrityChecks(false)

	// Move our viewport to the top
	s.Scroll(PageListScroll{Tag: PageListScrollTop})

	// Erase the entire history, we should be back to just our active set.
	s.EraseRows(Point{Tag: PointTagHistory}, None[Point]())
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
}

// Zig: test "PageList erase active regrows automatically"
func TestPl1_PageList_erase_active_regrows_automatically(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	expect(t, s.totalRows() == uint(s.rows), "s.totalRows() == s.rows")
	s.EraseRows(Point{Tag: PointTagActive}, Some(pl1Active(0, 10)))
	expect(t, s.totalRows() == uint(s.rows), "s.totalRows() == s.rows")
}

// Zig: test "PageList erase a one-row active"
func TestPl1_PageList_erase_a_one_row_active(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 1, None[uint64]())
	defer s.Deinit()
	expectEqual(t, uint(1), s.TotalPages())

	// Write our letter
	page := &s.pages.first.data
	for y := range uint(s.rows) {
		rac := page.GetRowAndCell(0, y)
		*rac.cell = pl1Codepoint('A')
	}

	s.EraseRows(Point{Tag: PointTagActive}, Some(Point{Tag: PointTagActive}))
	expectEqual(t, uint(s.rows), s.total_rows)

	// The row should be empty
	{
		get := zigUnwrap(t, s.GetCell(pl1Active(0, 0)))
		expectEqual(t, uint32(0), get.cell.content.codepoint)
	}
}

// Zig: test "PageList eraseRowBounded less than full row"
func TestPl1_PageList_eraseRowBounded_less_than_full_row(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 10, None[uint64]())
	defer s.Deinit()

	// Pins
	p_top, err := s.TrackPin(zigUnwrap(t, s.Pin(pl1Active(0, 5))))
	zigTry(t, err)
	defer s.UntrackPin(p_top)
	p_bot, err := s.TrackPin(zigUnwrap(t, s.Pin(pl1Active(0, 8))))
	zigTry(t, err)
	defer s.UntrackPin(p_bot)
	p_out, err := s.TrackPin(zigUnwrap(t, s.Pin(pl1Active(0, 9))))
	zigTry(t, err)
	defer s.UntrackPin(p_out)

	// Erase only a few rows in our active
	zigTry(t, s.EraseRowBounded(pl1Active(0, 5), 3))
	expectEqual(t, uint(s.rows), s.totalRows())

	// The erased rows should be dirty
	expect(t, s.IsDirty(pl1Active(0, 5)), "s.isDirty(.{ .active = .{ .x = 0, .y = 5 } })")
	expect(t, s.IsDirty(pl1Active(0, 6)), "s.isDirty(.{ .active = .{ .x = 0, .y = 6 } })")
	expect(t, s.IsDirty(pl1Active(0, 7)), "s.isDirty(.{ .active = .{ .x = 0, .y = 7 } })")

	expectEqual(t, s.pages.first, p_top.node)
	expectEqual(t, uint(4), uint(p_top.y))
	expectEqual(t, uint(0), uint(p_top.x))

	expectEqual(t, s.pages.first, p_bot.node)
	expectEqual(t, uint(7), uint(p_bot.y))
	expectEqual(t, uint(0), uint(p_bot.x))

	expectEqual(t, s.pages.first, p_out.node)
	expectEqual(t, uint(9), uint(p_out.y))
	expectEqual(t, uint(0), uint(p_out.x))
}

// Zig: test "PageList eraseRowBounded with pin at top"
func TestPl1_PageList_eraseRowBounded_with_pin_at_top(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 10, None[uint64]())
	defer s.Deinit()

	// Pins
	p_top, err := s.TrackPin(zigUnwrap(t, s.Pin(pl1Active(5, 0))))
	zigTry(t, err)
	defer s.UntrackPin(p_top)

	// Erase only a few rows in our active
	zigTry(t, s.EraseRowBounded(pl1Active(0, 0), 3))
	expectEqual(t, uint(s.rows), s.totalRows())

	// The erased rows should be dirty
	expect(t, s.IsDirty(pl1Active(0, 0)), "s.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, s.IsDirty(pl1Active(0, 1)), "s.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, s.IsDirty(pl1Active(0, 2)), "s.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")

	expectEqual(t, s.pages.first, p_top.node)
	expectEqual(t, uint(0), uint(p_top.y))
	expectEqual(t, uint(0), uint(p_top.x))
}

// Zig: test "PageList eraseRowBounded full rows single page"
func TestPl1_PageList_eraseRowBounded_full_rows_single_page(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 10, None[uint64]())
	defer s.Deinit()

	// Pins
	p_in, err := s.TrackPin(zigUnwrap(t, s.Pin(pl1Active(0, 7))))
	zigTry(t, err)
	defer s.UntrackPin(p_in)
	p_out, err := s.TrackPin(zigUnwrap(t, s.Pin(pl1Active(0, 9))))
	zigTry(t, err)
	defer s.UntrackPin(p_out)

	// Erase only a few rows in our active
	zigTry(t, s.EraseRowBounded(pl1Active(0, 5), 10))
	expectEqual(t, uint(s.rows), s.totalRows())

	// The erased rows should be dirty
	for y := uint(5); y < 10; y++ {
		expect(t, s.IsDirty(pl1Active(0, uint32(y))), "s.isDirty(.{ .active = .{ .x = 0, .y = y } })")
	}

	// Our pin should move to the first page
	expectEqual(t, s.pages.first, p_in.node)
	expectEqual(t, uint(6), uint(p_in.y))
	expectEqual(t, uint(0), uint(p_in.x))

	expectEqual(t, s.pages.first, p_out.node)
	expectEqual(t, uint(8), uint(p_out.y))
	expectEqual(t, uint(0), uint(p_out.x))
}

// Zig: test "PageList eraseRowBounded full rows two pages"
func TestPl1_PageList_eraseRowBounded_full_rows_two_pages(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 10, None[uint64]())
	defer s.Deinit()

	// Grow to two pages so our active area straddles
	{
		page := &s.pages.last.data
		page.PauseIntegrityChecks(true)
		for range uint(page.capacity.rows) - uint(page.size.rows) {
			pl1Grow(t, &s)
		}
		page.PauseIntegrityChecks(false)
		pl1GrowRows(t, &s, 5)
		expectEqual(t, uint(2), s.TotalPages())
		expectEqual(t, uint(5), uint(s.pages.last.data.size.rows))
	}

	// Pins
	p_first, err := s.TrackPin(zigUnwrap(t, s.Pin(pl1Active(0, 4))))
	zigTry(t, err)
	defer s.UntrackPin(p_first)
	p_first_out, err := s.TrackPin(zigUnwrap(t, s.Pin(pl1Active(0, 3))))
	zigTry(t, err)
	defer s.UntrackPin(p_first_out)
	p_in, err := s.TrackPin(zigUnwrap(t, s.Pin(pl1Active(0, 8))))
	zigTry(t, err)
	defer s.UntrackPin(p_in)
	p_out, err := s.TrackPin(zigUnwrap(t, s.Pin(pl1Active(0, 9))))
	zigTry(t, err)
	defer s.UntrackPin(p_out)

	{
		expectEqual(t, s.pages.last.prev, p_first.node)
		expectEqual(t, uint(p_first.node.data.size.rows-1), uint(p_first.y))
		expectEqual(t, uint(0), uint(p_first.x))

		expectEqual(t, s.pages.last.prev, p_first_out.node)
		expectEqual(t, uint(p_first_out.node.data.size.rows-2), uint(p_first_out.y))
		expectEqual(t, uint(0), uint(p_first_out.x))

		expectEqual(t, s.pages.last, p_in.node)
		expectEqual(t, uint(3), uint(p_in.y))
		expectEqual(t, uint(0), uint(p_in.x))

		expectEqual(t, s.pages.last, p_out.node)
		expectEqual(t, uint(4), uint(p_out.y))
		expectEqual(t, uint(0), uint(p_out.x))
	}

	// Erase only a few rows in our active
	zigTry(t, s.EraseRowBounded(pl1Active(0, 4), 4))

	// The erased rows should be dirty
	for y := uint(4); y < 8; y++ {
		expect(t, s.IsDirty(pl1Active(0, uint32(y))), "s.isDirty(.{ .active = .{ .x = 0, .y = y } })")
	}

	// In page in first page is shifted
	expectEqual(t, s.pages.last.prev, p_first.node)
	expectEqual(t, uint(p_first.node.data.size.rows-2), uint(p_first.y))
	expectEqual(t, uint(0), uint(p_first.x))

	// Out page in first page should not be shifted
	expectEqual(t, s.pages.last.prev, p_first_out.node)
	expectEqual(t, uint(p_first_out.node.data.size.rows-2), uint(p_first_out.y))
	expectEqual(t, uint(0), uint(p_first_out.x))

	// In page is shifted
	expectEqual(t, s.pages.last, p_in.node)
	expectEqual(t, uint(2), uint(p_in.y))
	expectEqual(t, uint(0), uint(p_in.x))

	// Out page is not shifted
	expectEqual(t, s.pages.last, p_out.node)
	expectEqual(t, uint(4), uint(p_out.y))
	expectEqual(t, uint(0), uint(p_out.x))
}

// Zig: test "PageList clone"
func TestPl1_PageList_clone(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	expectEqual(t, uint(s.rows), s.totalRows())

	s2 := pl1Clone(t, &s, PageListClone{
		top: pl1Screen(0, 0),
	})
	defer s2.Deinit()
	expectEqual(t, uint(s.rows), s2.totalRows())
}

// Zig: test "PageList clone partial trimmed right"
func TestPl1_PageList_clone_partial_trimmed_right(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 20, None[uint64]())
	defer s.Deinit()
	expectEqual(t, uint(s.rows), s.totalRows())
	pl1GrowRows(t, &s, 30)

	s2 := pl1Clone(t, &s, PageListClone{
		top: pl1Screen(0, 0),
		bot: Some(pl1Screen(0, 39)),
	})
	defer s2.Deinit()
	expectEqual(t, uint(40), s2.totalRows())
}

// Zig: test "PageList clone partial trimmed left"
func TestPl1_PageList_clone_partial_trimmed_left(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 20, None[uint64]())
	defer s.Deinit()
	expectEqual(t, uint(s.rows), s.totalRows())
	pl1GrowRows(t, &s, 30)

	s2 := pl1Clone(t, &s, PageListClone{
		top: pl1Screen(0, 10),
	})
	defer s2.Deinit()
	expectEqual(t, uint(40), s2.totalRows())
}

// Zig: test "PageList clone partial trimmed left reclaims styles"
func TestPl1_PageList_clone_partial_trimmed_left_reclaims_styles(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 20, None[uint64]())
	defer s.Deinit()
	expectEqual(t, uint(s.rows), s.totalRows())
	pl1GrowRows(t, &s, 30)

	// Style the rows we're trimming
	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		style := Style{flags: styleFlags{bold: true}}
		style_id, err := page.styles.Add(style)
		zigTry(t, err)

		it := s.RowIterator(PageListDirectionLeftUp, Point{Tag: PointTagScreen}, Some(pl1Screen(0, 9)))
		for {
			pn := it.Next()
			if !pn.Valid {
				break
			}
			p := pn.Value
			rac := p.RowAndCell()
			rac.row.styled = true
			*rac.cell = PageCell{
				content_tag: PageCellContentTagCodepoint,
				content:     PageCellContent{codepoint: 'A'},
				style_id:    style_id,
			}
			page.styles.Use(style_id)
		}

		// We're over-counted by 1 because `add` implies `use`.
		page.styles.Release(style_id)

		// Expect to have one style
		expectEqual(t, uint(1), page.styles.Count())
	}

	s2 := pl1Clone(t, &s, PageListClone{
		top: pl1Screen(0, 10),
	})
	defer s2.Deinit()
	expectEqual(t, uint(40), s2.totalRows())

	{
		expect(t, s2.pages.first == s2.pages.last, "s2.pages.first == s2.pages.last")
		page := &s2.pages.first.data
		expectEqual(t, uint(0), page.styles.Count())
	}
}

// Zig: test "PageList clone partial trimmed both"
func TestPl1_PageList_clone_partial_trimmed_both(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 20, None[uint64]())
	defer s.Deinit()
	expectEqual(t, uint(s.rows), s.totalRows())
	pl1GrowRows(t, &s, 30)

	s2 := pl1Clone(t, &s, PageListClone{
		top: pl1Screen(0, 10),
		bot: Some(pl1Screen(0, 35)),
	})
	defer s2.Deinit()
	expectEqual(t, uint(26), s2.totalRows())
}

// Zig: test "PageList clone less than active"
func TestPl1_PageList_clone_less_than_active(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	expectEqual(t, uint(s.rows), s.totalRows())

	s2 := pl1Clone(t, &s, PageListClone{
		top: pl1Active(0, 5),
	})
	defer s2.Deinit()
	expectEqual(t, uint(s.rows), s2.totalRows())
}

// Zig: test "PageList clone remap tracked pin"
func TestPl1_PageList_clone_remap_tracked_pin(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	expectEqual(t, uint(s.rows), s.totalRows())

	// Put a tracked pin in the screen
	p, err := s.TrackPin(zigUnwrap(t, s.Pin(pl1Active(0, 6))))
	zigTry(t, err)
	defer s.UntrackPin(p)

	pin_remap := PageListCloneTrackedPinsRemap{}
	s2 := pl1Clone(t, &s, PageListClone{
		top:          pl1Active(0, 5),
		tracked_pins: &pin_remap,
	})
	defer s2.Deinit()

	// We should be able to find our tracked pin
	p2, ok := pin_remap[p]
	expect(t, ok, "pin_remap.get(p) != null")
	expectEqualPoint(t,
		pl1Active(0, 1),
		zigUnwrap(t, s2.PointFromPin(PointTagActive, *p2)),
	)
}

// Zig: test "PageList clone remap tracked pin not in cloned area"
func TestPl1_PageList_clone_remap_tracked_pin_not_in_cloned_area(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	expectEqual(t, uint(s.rows), s.totalRows())

	// Put a tracked pin in the screen
	p, err := s.TrackPin(zigUnwrap(t, s.Pin(pl1Active(0, 3))))
	zigTry(t, err)
	defer s.UntrackPin(p)

	pin_remap := PageListCloneTrackedPinsRemap{}
	s2 := pl1Clone(t, &s, PageListClone{
		top:          pl1Active(0, 5),
		tracked_pins: &pin_remap,
	})
	defer s2.Deinit()

	// We should be able to find our tracked pin
	_, ok := pin_remap[p]
	expect(t, !ok, "pin_remap.get(p) == null")
}

// Zig: test "PageList clone full dirty"
func TestPl1_PageList_clone_full_dirty(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	expectEqual(t, uint(s.rows), s.totalRows())

	// Mark a row as dirty
	s.markDirty(pl1Active(0, 0))
	s.markDirty(pl1Active(0, 12))
	s.markDirty(pl1Active(0, 23))

	s2 := pl1Clone(t, &s, PageListClone{
		top: pl1Screen(0, 0),
	})
	defer s2.Deinit()
	expectEqual(t, uint(s.rows), s2.totalRows())

	// Should still be dirty
	expect(t, s2.IsDirty(pl1Active(0, 0)), "s2.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, !s2.IsDirty(pl1Active(0, 1)), "!s2.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, s2.IsDirty(pl1Active(0, 12)), "s2.isDirty(.{ .active = .{ .x = 0, .y = 12 } })")
	expect(t, !s2.IsDirty(pl1Active(0, 14)), "!s2.isDirty(.{ .active = .{ .x = 0, .y = 14 } })")
	expect(t, s2.IsDirty(pl1Active(0, 23)), "s2.isDirty(.{ .active = .{ .x = 0, .y = 23 } })")
}

// Zig: test "PageList resize (no reflow) more rows"
func TestPl1_PageList_resize_no_reflow_more_rows(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 3, Some[uint64](0))
	defer s.Deinit()
	expectEqual(t, uint(3), s.totalRows())

	// Put a tracked pin in the history
	p, err := s.TrackPin(zigUnwrap(t, s.Pin(pl1Active(0, 2))))
	zigTry(t, err)
	defer s.UntrackPin(p)

	// Resize
	zigTry(t, s.Resize(PageListResize{rows: Some[SizeCellCountInt](10), reflow: false}))
	expectEqual(t, uint(10), uint(s.rows))
	expectEqual(t, uint(10), s.totalRows())

	// Our cursor should not move because we have no scrollback so
	// we just grew.
	expectEqualPoint(t, pl1Active(0, 2), zigUnwrap(t, s.PointFromPin(PointTagActive, *p)))

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagActive})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 0), pt)
	}
}

// Zig: test "PageList resize (no reflow) more rows with history"
func TestPl1_PageList_resize_no_reflow_more_rows_with_history(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 3, None[uint64]())
	defer s.Deinit()
	pl1GrowRows(t, &s, 50)
	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagActive})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 50), pt)
	}

	// Put a tracked pin in the history
	p, err := s.TrackPin(zigUnwrap(t, s.Pin(pl1Active(0, 2))))
	zigTry(t, err)
	defer s.UntrackPin(p)

	// Resize
	zigTry(t, s.Resize(PageListResize{rows: Some[SizeCellCountInt](5), reflow: false}))
	expectEqual(t, uint(5), uint(s.rows))
	expectEqual(t, uint(53), s.totalRows())

	// Our cursor should move since it's in the scrollback
	expectEqualPoint(t, pl1Active(0, 4), zigUnwrap(t, s.PointFromPin(PointTagActive, *p)))

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagActive})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 48), pt)
	}
}

// Zig: test "PageList resize (no reflow) less rows"
func TestPl1_PageList_resize_no_reflow_less_rows(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 10, Some[uint64](0))
	defer s.Deinit()
	expectEqual(t, uint(10), s.totalRows())

	// This is required for our writing below to work
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Write into all rows so we don't get trim behavior
	for y := range uint(s.rows) {
		rac := page.GetRowAndCell(0, y)
		*rac.cell = pl1Codepoint('A')
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{rows: Some[SizeCellCountInt](5), reflow: false}))
	expectEqual(t, uint(5), uint(s.rows))
	expectEqual(t, uint(10), s.totalRows())
	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagActive})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 5), pt)
	}
}

// Zig: test "PageList resize (no reflow) one rows"
func TestPl1_PageList_resize_no_reflow_one_rows(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 10, Some[uint64](0))
	defer s.Deinit()
	expectEqual(t, uint(10), s.totalRows())

	// This is required for our writing below to work
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Write into all rows so we don't get trim behavior
	for y := range uint(s.rows) {
		rac := page.GetRowAndCell(0, y)
		*rac.cell = pl1Codepoint('A')
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{rows: Some[SizeCellCountInt](1), reflow: false}))
	expectEqual(t, uint(1), uint(s.rows))
	expectEqual(t, uint(10), s.totalRows())
	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagActive})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 9), pt)
	}
}

// Zig: test "PageList resize (no reflow) less rows cursor on bottom"
func TestPl1_PageList_resize_no_reflow_less_rows_cursor_on_bottom(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 10, Some[uint64](0))
	defer s.Deinit()
	expectEqual(t, uint(10), s.totalRows())

	// This is required for our writing below to work
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Write into all rows so we don't get trim behavior
	for y := range uint(s.rows) {
		rac := page.GetRowAndCell(0, y)
		*rac.cell = pl1Codepoint(uint32(y))
	}

	// Put a tracked pin in the history
	p, err := s.TrackPin(zigUnwrap(t, s.Pin(pl1Active(0, 9))))
	zigTry(t, err)
	defer s.UntrackPin(p)
	{
		cursor := zigUnwrap(t, s.PointFromPin(PointTagActive, *p)).Active
		get := zigUnwrap(t, s.GetCell(pl1Active(cursor.x, cursor.y)))
		expectEqual(t, uint32(9), get.cell.content.codepoint)
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{rows: Some[SizeCellCountInt](5), reflow: false}))
	expectEqual(t, uint(5), uint(s.rows))
	expectEqual(t, uint(10), s.totalRows())

	// Our cursor should move since it's in the scrollback
	expectEqualPoint(t, pl1Active(0, 4), zigUnwrap(t, s.PointFromPin(PointTagActive, *p)))

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagActive})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 5), pt)
	}
}

// Zig: test "PageList resize (no reflow) less rows cursor in scrollback"
func TestPl1_PageList_resize_no_reflow_less_rows_cursor_in_scrollback(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 10, Some[uint64](0))
	defer s.Deinit()
	expectEqual(t, uint(10), s.totalRows())

	// This is required for our writing below to work
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Write into all rows so we don't get trim behavior
	for y := range uint(s.rows) {
		rac := page.GetRowAndCell(0, y)
		*rac.cell = pl1Codepoint(uint32(y))
	}

	// Put a tracked pin in the history
	p, err := s.TrackPin(zigUnwrap(t, s.Pin(pl1Active(0, 2))))
	zigTry(t, err)
	defer s.UntrackPin(p)
	{
		cursor := zigUnwrap(t, s.PointFromPin(PointTagActive, *p)).Active
		get := zigUnwrap(t, s.GetCell(pl1Active(cursor.x, cursor.y)))
		expectEqual(t, uint32(2), get.cell.content.codepoint)
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{rows: Some[SizeCellCountInt](5), reflow: false}))
	expectEqual(t, uint(5), uint(s.rows))
	expectEqual(t, uint(10), s.totalRows())

	// Our cursor should move since it's in the scrollback
	expect(t, !s.PointFromPin(PointTagActive, *p).Valid, "s.pointFromPin(.active, p.*) == null")
	expectEqualPoint(t, pl1Screen(0, 2), zigUnwrap(t, s.PointFromPin(PointTagScreen, *p)))

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagActive})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 5), pt)
	}
}

// Zig: test "PageList resize (no reflow) less rows trims blank lines"
func TestPl1_PageList_resize_no_reflow_less_rows_trims_blank_lines(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 5, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Write codepoint into first line
	{
		rac := page.GetRowAndCell(0, 0)
		*rac.cell = pl1Codepoint('A')
	}

	// Fill remaining lines with a background color
	for y := uint(1); y < uint(s.rows); y++ {
		rac := page.GetRowAndCell(0, y)
		*rac.cell = PageCell{
			content_tag: PageCellContentTagBgColorRgb,
			content:     PageCellContent{color_rgb: PageCellRGB{r: 0xFF, g: 0, b: 0}},
		}
	}

	// Put a tracked pin in the history
	p, err := s.TrackPin(zigUnwrap(t, s.Pin(pl1Active(0, 0))))
	zigTry(t, err)
	defer s.UntrackPin(p)
	{
		cursor := zigUnwrap(t, s.PointFromPin(PointTagActive, *p)).Active
		get := zigUnwrap(t, s.GetCell(pl1Active(cursor.x, cursor.y)))
		expectEqual(t, uint32('A'), get.cell.content.codepoint)
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{rows: Some[SizeCellCountInt](2), reflow: false}))
	expectEqual(t, uint(2), uint(s.rows))
	expectEqual(t, uint(2), s.totalRows())

	// Our cursor should not move since we trimmed
	expectEqualPoint(t, pl1Active(0, 0), zigUnwrap(t, s.PointFromPin(PointTagActive, *p)))

	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagActive})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 0), pt)
	}
}

// Zig: test "PageList resize (no reflow) less rows trims blank lines cursor in blank line"
func TestPl1_PageList_resize_no_reflow_less_rows_trims_blank_lines_cursor_in_blank_line(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 5, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Write codepoint into first line
	{
		rac := page.GetRowAndCell(0, 0)
		*rac.cell = pl1Codepoint('A')
	}

	// Fill remaining lines with a background color
	for y := uint(1); y < uint(s.rows); y++ {
		rac := page.GetRowAndCell(0, y)
		*rac.cell = PageCell{
			content_tag: PageCellContentTagBgColorRgb,
			content:     PageCellContent{color_rgb: PageCellRGB{r: 0xFF, g: 0, b: 0}},
		}
	}

	// Put a tracked pin in a blank line
	p, err := s.TrackPin(zigUnwrap(t, s.Pin(pl1Active(0, 3))))
	zigTry(t, err)
	defer s.UntrackPin(p)

	// Resize
	zigTry(t, s.Resize(PageListResize{rows: Some[SizeCellCountInt](2), reflow: false}))
	expectEqual(t, uint(2), uint(s.rows))
	expectEqual(t, uint(4), s.totalRows())

	// Our cursor should not move since we trimmed
	expectEqualPoint(t, pl1Active(0, 1), zigUnwrap(t, s.PointFromPin(PointTagActive, *p)))
}

// Zig: test "PageList resize (no reflow) less rows trims blank lines erases pages"
func TestPl1_PageList_resize_no_reflow_less_rows_trims_blank_lines_erases_pages(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 100, 5, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Resize to take up two pages
	{
		rows := page.capacity.rows + 10
		zigTry(t, s.Resize(PageListResize{rows: Some(rows), reflow: false}))
		expectEqual(t, uint(2), s.TotalPages())
	}

	// Write codepoint into first line
	{
		rac := page.GetRowAndCell(0, 0)
		*rac.cell = pl1Codepoint('A')
	}

	// Resize down. Every row except the first is blank so we
	// should erase the second page.
	zigTry(t, s.Resize(PageListResize{rows: Some[SizeCellCountInt](5), reflow: false}))
	expectEqual(t, uint(5), uint(s.rows))
	expectEqual(t, uint(5), s.totalRows())
	expectEqual(t, uint(1), s.TotalPages())
}

// Zig: test "PageList resize (no reflow) more rows extends blank lines"
func TestPl1_PageList_resize_no_reflow_more_rows_extends_blank_lines(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 3, Some[uint64](0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data

	// Write codepoint into first line
	{
		rac := page.GetRowAndCell(0, 0)
		*rac.cell = pl1Codepoint('A')
	}

	// Fill remaining lines with a background color
	for y := uint(1); y < uint(s.rows); y++ {
		rac := page.GetRowAndCell(0, y)
		*rac.cell = PageCell{
			content_tag: PageCellContentTagBgColorRgb,
			content:     PageCellContent{color_rgb: PageCellRGB{r: 0xFF, g: 0, b: 0}},
		}
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{rows: Some[SizeCellCountInt](7), reflow: false}))
	expectEqual(t, uint(7), uint(s.rows))
	expectEqual(t, uint(7), s.totalRows())
	{
		pt := zigUnwrap(t, s.GetCell(Point{Tag: PointTagActive})).ScreenPoint()
		expectEqualPoint(t, pl1Screen(0, 0), pt)
	}
}

// Zig: test "PageList resize (no reflow) more rows contains viewport"
func TestPl1_PageList_resize_no_reflow_more_rows_contains_viewport(t *testing.T) {
	testStdCapacity(t)

	// When the rows are increased we need to make sure that the viewport
	// doesn't end up below the active area if it's currently in pin mode.

	s := pl1Init(t, 5, 5, Some[uint64](1))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")

	// Make it so we have scrollback
	pl1Grow(t, &s)

	expectEqual(t, uint(5), uint(s.rows))
	expectEqual(t, uint(6), s.totalRows())

	// Set viewport above active by scrolling up one.
	s.Scroll(PageListScroll{Tag: PageListScrollDeltaRow, DeltaRow: -1})
	// The viewport should be a pin now.
	expectEqual(t, PageListViewport{Tag: PageListViewportTop}, s.viewport)

	// Resize
	zigTry(t, s.Resize(PageListResize{rows: Some[SizeCellCountInt](7), reflow: false}))
	expectEqual(t, uint(7), uint(s.rows))
	expectEqual(t, uint(7), s.totalRows())

	// Question: maybe the viewport should actually be in the active
	// here and not pinned to the top.
	expectEqual(t, PageListViewport{Tag: PageListViewportTop}, s.viewport)
}

// Zig: test "PageList resize (no reflow) less cols"
func TestPl1_PageList_resize_no_reflow_less_cols(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 10, Some[uint64](0))
	defer s.Deinit()

	// Resize
	zigTry(t, s.Resize(PageListResize{cols: Some[SizeCellCountInt](5), reflow: false}))
	expectEqual(t, uint(5), uint(s.cols))
	expectEqual(t, uint(10), s.totalRows())

	it := s.RowIterator(PageListDirectionRightDown, Point{Tag: PointTagScreen}, None[Point]())
	for {
		o := it.Next()
		if !o.Valid {
			break
		}
		offset := o.Value
		rac := offset.RowAndCell()
		cells := offset.node.data.GetCells(rac.row)
		expectEqual(t, uint(5), uint(len(cells)))
	}
}

// Zig: test "PageList resize (no reflow) less cols pin in trimmed cols"
func TestPl1_PageList_resize_no_reflow_less_cols_pin_in_trimmed_cols(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 10, Some[uint64](0))
	defer s.Deinit()

	// Put a tracked pin in the history
	p, err := s.TrackPin(zigUnwrap(t, s.Pin(pl1Active(8, 2))))
	zigTry(t, err)
	defer s.UntrackPin(p)

	// Resize
	zigTry(t, s.Resize(PageListResize{cols: Some[SizeCellCountInt](5), reflow: false}))
	expectEqual(t, uint(5), uint(s.cols))
	expectEqual(t, uint(10), s.totalRows())

	it := s.RowIterator(PageListDirectionRightDown, Point{Tag: PointTagScreen}, None[Point]())
	for {
		o := it.Next()
		if !o.Valid {
			break
		}
		offset := o.Value
		rac := offset.RowAndCell()
		cells := offset.node.data.GetCells(rac.row)
		expectEqual(t, uint(5), uint(len(cells)))
	}

	expectEqualPoint(t, pl1Active(4, 2), zigUnwrap(t, s.PointFromPin(PointTagActive, *p)))
}

// Zig: test "PageList resize (no reflow) less cols clears graphemes"
func TestPl1_PageList_resize_no_reflow_less_cols_clears_graphemes(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 10, Some[uint64](0))
	defer s.Deinit()

	// Add a grapheme.
	page := &s.pages.first.data
	{
		rac := page.GetRowAndCell(9, 0)
		*rac.cell = pl1Codepoint('A')
		zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 'A'))
	}
	expectEqual(t, uint(1), page.GraphemeCount())

	// Resize
	zigTry(t, s.Resize(PageListResize{cols: Some[SizeCellCountInt](5), reflow: false}))
	expectEqual(t, uint(5), uint(s.cols))
	expectEqual(t, uint(10), s.totalRows())

	it := s.PageIterator(PageListDirectionRightDown, Point{Tag: PointTagScreen}, None[Point]())
	for {
		c := it.Next()
		if !c.Valid {
			break
		}
		chunk := c.Value
		expectEqual(t, uint(0), chunk.node.data.GraphemeCount())
	}
}

// Zig: test "PageList resize (no reflow) more cols"
func TestPl1_PageList_resize_no_reflow_more_cols(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 5, 3, Some[uint64](0))
	defer s.Deinit()

	// Resize
	zigTry(t, s.Resize(PageListResize{cols: Some[SizeCellCountInt](10), reflow: false}))
	expectEqual(t, uint(10), uint(s.cols))
	expectEqual(t, uint(3), s.totalRows())

	it := s.RowIterator(PageListDirectionRightDown, Point{Tag: PointTagScreen}, None[Point]())
	for {
		o := it.Next()
		if !o.Valid {
			break
		}
		offset := o.Value
		rac := offset.RowAndCell()
		cells := offset.node.data.GetCells(rac.row)
		expectEqual(t, uint(10), uint(len(cells)))
	}
}

// Zig: test "PageList resize (no reflow) more cols with spacer head"
func TestPl1_PageList_resize_no_reflow_more_cols_with_spacer_head(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 2, 3, Some[uint64](0))
	defer s.Deinit()
	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		{
			rac := page.GetRowAndCell(0, 0)
			rac.row.wrap = true
			*rac.cell = pl1Codepoint('x')
		}
		{
			rac := page.GetRowAndCell(1, 0)
			*rac.cell = PageCell{
				content_tag: PageCellContentTagCodepoint,
				content:     PageCellContent{codepoint: 0},
				wide:        PageCellWideSpacerHead,
			}
		}
		{
			rac := page.GetRowAndCell(0, 1)
			*rac.cell = PageCell{
				content_tag: PageCellContentTagCodepoint,
				content:     PageCellContent{codepoint: '😀'},
				wide:        PageCellWideWide,
			}
		}
		{
			rac := page.GetRowAndCell(1, 1)
			*rac.cell = PageCell{
				content_tag: PageCellContentTagCodepoint,
				content:     PageCellContent{codepoint: 0},
				wide:        PageCellWideSpacerTail,
			}
		}
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{cols: Some[SizeCellCountInt](3), reflow: false}))
	expectEqual(t, uint(3), uint(s.cols))
	expectEqual(t, uint(3), s.totalRows())

	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		{
			rac := page.GetRowAndCell(0, 0)
			expectEqual(t, uint32('x'), rac.cell.content.codepoint)
			expectEqual(t, PageCellWideNarrow, rac.cell.wide)
			// try testing.expect(!rac.row.wrap);
		}
		{
			rac := page.GetRowAndCell(1, 0)
			expectEqual(t, uint32(0), rac.cell.content.codepoint)
			expectEqual(t, PageCellWideNarrow, rac.cell.wide)
		}
		{
			rac := page.GetRowAndCell(2, 0)
			expectEqual(t, uint32(0), rac.cell.content.codepoint)
			expectEqual(t, PageCellWideNarrow, rac.cell.wide)
		}
	}
}

// Zig: test "PageList resize (no reflow) grow cols fast path with spacer head"
func TestPl1_PageList_resize_no_reflow_grow_cols_fast_path_with_spacer_head(t *testing.T) {
	testStdCapacity(t)

	// Regression test for fuzz crash. When we shrink cols and then
	// grow back, the page retains capacity from the original size so the grow
	// takes the fast path (just bumps page.size.cols). If any row has a
	// spacer_head at the old last column, that cell is no longer at the end
	// of the wider row, violating page integrity.
	s := pl1Init(t, 10, 3, Some[uint64](0))
	defer s.Deinit()

	// Shrink to 5 cols. The page keeps capacity for 10 cols.
	zigTry(t, s.Resize(PageListResize{cols: Some[SizeCellCountInt](5), reflow: false}))
	expectEqual(t, uint(5), uint(s.cols))

	// Place a spacer_head at the last column (col 4) on two rows
	// to simulate a wide character that didn't fit at the right edge.
	{
		page := &s.pages.first.data

		// Row 0: 'x' at col 0..3, spacer_head at col 4, wrap = true
		{
			rac := page.GetRowAndCell(0, 0)
			*rac.cell = pl1Codepoint('x')
		}
		{
			rac := page.GetRowAndCell(4, 0)
			*rac.cell = PageCell{
				content_tag: PageCellContentTagCodepoint,
				content:     PageCellContent{codepoint: 0},
				wide:        PageCellWideSpacerHead,
			}
			rac.row.wrap = true
		}

		// Row 1: spacer_head at col 4, wrap = true
		{
			rac := page.GetRowAndCell(4, 1)
			*rac.cell = PageCell{
				content_tag: PageCellContentTagCodepoint,
				content:     PageCellContent{codepoint: 0},
				wide:        PageCellWideSpacerHead,
			}
			rac.row.wrap = true
		}
	}

	// Grow back to 10 cols. This must not leave stale spacer_head
	// cells at col 4 (which is no longer the last column).
	zigTry(t, s.Resize(PageListResize{cols: Some[SizeCellCountInt](10), reflow: false}))
	expectEqual(t, uint(10), uint(s.cols))

	// Verify the old spacer_head positions are now narrow.
	{
		page := &s.pages.first.data
		{
			rac := page.GetRowAndCell(4, 0)
			expectEqual(t, PageCellWideNarrow, rac.cell.wide)
			expect(t, !rac.row.wrap, "!rac.row.wrap")
		}
		{
			rac := page.GetRowAndCell(4, 1)
			expectEqual(t, PageCellWideNarrow, rac.cell.wide)
			expect(t, !rac.row.wrap, "!rac.row.wrap")
		}
	}
}

// Zig: test "PageList resize (no reflow) more cols forces less rows per page"
func TestPl1_PageList_resize_no_reflow_more_cols_forces_less_rows_per_page(t *testing.T) {
	testStdCapacity(t)

	// This test is a bit convoluted so I want to explain: what we are trying
	// to verify here is that when we increase cols such that our rows per page
	// shrinks, we don't fragment our rows across many pages because this ends
	// up wasting a lot of memory.
	//
	// This is particularly important for alternate screen buffers where we
	// don't have scrollback so our max size is very small. If we don't do this,
	// we end up pruning our pages and that causes resizes to fail!

	// This test requires initially that our rows fit into one page.
	const cols SizeCellCountInt = 5
	const rows SizeCellCountInt = 150
	{
		cap, err := PageStdCapacity.Adjust(PageCapacityAdjustment{cols: Some(cols)})
		zigTry(t, err)
		expect(t, cap.rows >= rows, "(try std_capacity.adjust(.{ .cols = cols })).rows >= rows")
	}
	s := pl1Init(t, cols, rows, Some[uint64](0))
	defer s.Deinit()

	// Then we need to resize our cols so that our rows per page shrinks.
	// This will force our resize to split our rows across two pages.
	{
		new_cols := func() SizeCellCountInt {
			var new_cols SizeCellCountInt = 50
			cap, err := PageStdCapacity.Adjust(PageCapacityAdjustment{cols: Some(new_cols)})
			zigTry(t, err)
			for cap.rows >= rows {
				new_cols += 50
				cap, err = PageStdCapacity.Adjust(PageCapacityAdjustment{cols: Some(new_cols)})
				zigTry(t, err)
			}

			return new_cols
		}()
		zigTry(t, s.Resize(PageListResize{cols: Some(new_cols), reflow: false}))
		expectEqual(t, uint(new_cols), uint(s.cols))
		expectEqual(t, uint(rows), s.totalRows())
	}

	// Every page except the last should be full
	{
		it := s.pages.first
		for it != nil {
			page := it
			if page == s.pages.last {
				break
			}
			expectEqual(t, page.data.capacity.rows, page.data.size.rows)
			it = page.next
		}
	}

	// Now we need to resize again to a col size that further shrinks
	// our last capacity.
	{
		page := &s.pages.first.data
		expect(t, page.size.rows == page.capacity.rows, "page.size.rows == page.capacity.rows")
		new_cols := func() SizeCellCountInt {
			new_cols := page.size.cols + 50
			cap, err := PageStdCapacity.Adjust(PageCapacityAdjustment{cols: Some(new_cols)})
			zigTry(t, err)
			for cap.rows >= page.size.rows {
				new_cols += 50
				cap, err = PageStdCapacity.Adjust(PageCapacityAdjustment{cols: Some(new_cols)})
				zigTry(t, err)
			}

			return new_cols
		}()

		zigTry(t, s.Resize(PageListResize{cols: Some(new_cols), reflow: false}))
		expectEqual(t, uint(new_cols), uint(s.cols))
		expectEqual(t, uint(rows), s.totalRows())
	}

	// Every page except the last should be full
	{
		it := s.pages.first
		for it != nil {
			page := it
			if page == s.pages.last {
				break
			}
			expectEqual(t, page.data.capacity.rows, page.data.size.rows)
			it = page.next
		}
	}
}

// Zig: test "PageList resize (no reflow) less cols then more cols"
func TestPl1_PageList_resize_no_reflow_less_cols_then_more_cols(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 5, 3, Some[uint64](0))
	defer s.Deinit()

	// Resize less
	zigTry(t, s.Resize(PageListResize{cols: Some[SizeCellCountInt](2), reflow: false}))
	expectEqual(t, uint(2), uint(s.cols))

	// Resize
	zigTry(t, s.Resize(PageListResize{cols: Some[SizeCellCountInt](5), reflow: false}))
	expectEqual(t, uint(5), uint(s.cols))
	expectEqual(t, uint(3), s.totalRows())

	it := s.RowIterator(PageListDirectionRightDown, Point{Tag: PointTagScreen}, None[Point]())
	for {
		o := it.Next()
		if !o.Valid {
			break
		}
		offset := o.Value
		rac := offset.RowAndCell()
		cells := offset.node.data.GetCells(rac.row)
		expectEqual(t, uint(5), uint(len(cells)))
	}
}

// Zig: test "PageList resize (no reflow) less rows and cols"
func TestPl1_PageList_resize_no_reflow_less_rows_and_cols(t *testing.T) {
	testStdCapacity(t)

	s := pl1Init(t, 10, 10, Some[uint64](0))
	defer s.Deinit()

	// Resize less
	zigTry(t, s.Resize(PageListResize{
		cols:   Some[SizeCellCountInt](5),
		rows:   Some[SizeCellCountInt](7),
		reflow: false,
	}))
	expectEqual(t, uint(5), uint(s.cols))
	expectEqual(t, uint(7), uint(s.rows))

	it := s.RowIterator(PageListDirectionRightDown, Point{Tag: PointTagScreen}, None[Point]())
	for {
		o := it.Next()
		if !o.Valid {
			break
		}
		offset := o.Value
		rac := offset.RowAndCell()
		cells := offset.node.data.GetCells(rac.row)
		expectEqual(t, uint(5), uint(len(cells)))
	}
}

// ---------------------------------------------------------------------------
// Go-only panic tests. Zig's tests cannot test an assert, `unreachable` or a
// null unwrap (`.?`) because the test binary would abort; these check that
// the Go port stops where upstream would. The tests do not call Deinit: it
// asserts integrity again and the state after a panic is not under test.
// ---------------------------------------------------------------------------

// Go-only: panic test: PageListInit: rows == 0 (initPages: assert(page_list.first != null))
func TestPl1_panic_PageListInit_zero_rows(t *testing.T) {
	testStdCapacity(t)
	pl1ExpectPanic(t, func() {
		_, _ = PageListInit(80, 0, None[uint64]())
	})
}

// Go-only: panic test: PageList.AssertIntegrity: total_rows does not match the pages (@panic("PageList integrity check failed"))
func TestPl1_panic_AssertIntegrity_total_rows_mismatch(t *testing.T) {
	testStdCapacity(t)
	s := pl1Init(t, 80, 24, None[uint64]())
	s.total_rows += 1
	pl1ExpectPanic(t, func() { s.AssertIntegrity() })
}

// Go-only: panic test: PageList.PauseIntegrityChecks: resume without a pause (pause_integrity_checks -= 1 underflows usize)
func TestPl1_panic_PauseIntegrityChecks_resume_without_pause(t *testing.T) {
	testStdCapacity(t)
	s := pl1Init(t, 80, 24, None[uint64]())
	expectEqual(t, uint(0), s.pause_integrity_checks)
	pl1ExpectPanic(t, func() { s.PauseIntegrityChecks(false) })
}

// Go-only: panic test: PageList.Resize: cols == 0 (assert(v > 0))
func TestPl1_panic_Resize_zero_cols(t *testing.T) {
	testStdCapacity(t)
	s := pl1Init(t, 80, 24, None[uint64]())
	pl1ExpectPanic(t, func() {
		_ = s.Resize(PageListResize{cols: Some[SizeCellCountInt](0), reflow: false})
	})
}

// Go-only: panic test: PageList.Resize: rows == 0 (assert(v > 0))
func TestPl1_panic_Resize_zero_rows(t *testing.T) {
	testStdCapacity(t)
	s := pl1Init(t, 80, 24, None[uint64]())
	pl1ExpectPanic(t, func() {
		_ = s.Resize(PageListResize{rows: Some[SizeCellCountInt](0), reflow: false})
	})
}

// Go-only: panic test: PageList.TrackPin: pin row beyond its page (assert(self.pinIsValid(p)))
func TestPl1_panic_TrackPin_row_out_of_page(t *testing.T) {
	testStdCapacity(t)
	s := pl1Init(t, 80, 24, None[uint64]())
	p := PageListPin{node: s.pages.first, y: s.pages.first.data.size.rows, x: 0}
	pl1ExpectPanic(t, func() { _, _ = s.TrackPin(p) })
}

// Go-only: panic test: PageList.TrackPin: pin column beyond its page (assert(self.pinIsValid(p)))
func TestPl1_panic_TrackPin_col_out_of_page(t *testing.T) {
	testStdCapacity(t)
	s := pl1Init(t, 80, 24, None[uint64]())
	p := PageListPin{node: s.pages.first, y: 0, x: s.pages.first.data.size.cols}
	pl1ExpectPanic(t, func() { _, _ = s.TrackPin(p) })
}

// Go-only: panic test: PageList.TrackPin: pin on a node that is not in the list (assert(self.pinIsValid(p)))
func TestPl1_panic_TrackPin_node_not_in_list(t *testing.T) {
	testStdCapacity(t)
	s := pl1Init(t, 80, 24, None[uint64]())
	other := pl1Init(t, 80, 24, None[uint64]())
	p := PageListPin{node: other.pages.first, y: 0, x: 0}
	pl1ExpectPanic(t, func() { _, _ = s.TrackPin(p) })
}

// Go-only: panic test: PageList.UntrackPin: the viewport pin (assert(p != self.viewport_pin))
func TestPl1_panic_UntrackPin_viewport_pin(t *testing.T) {
	testStdCapacity(t)
	s := pl1Init(t, 80, 24, None[uint64]())
	pl1ExpectPanic(t, func() { s.UntrackPin(s.viewport_pin) })
}

// Go-only: panic test: PageList.PageIterator: bottom point before the top point (assert(tl_pin.eql(bl_pin) or tl_pin.before(bl_pin)))
func TestPl1_panic_PageIterator_bl_before_tl(t *testing.T) {
	testStdCapacity(t)
	s := pl1Init(t, 80, 24, None[uint64]())
	pl1GrowRows(t, &s, 10)
	pl1ExpectPanic(t, func() {
		s.PageIterator(PageListDirectionRightDown, pl1Screen(0, 5), Some(pl1Screen(0, 2)))
	})
}

// Go-only: panic test: PageList.PageIterator: top point outside the screen (self.pin(tl_pt).?)
func TestPl1_panic_PageIterator_tl_out_of_range(t *testing.T) {
	testStdCapacity(t)
	s := pl1Init(t, 80, 24, None[uint64]())
	pl1ExpectPanic(t, func() {
		s.PageIterator(PageListDirectionRightDown, pl1Screen(0, 1000), None[Point]())
	})
}

// Go-only: panic test: PageList.PageIterator: bottom point outside the screen (self.pin(pt).?)
func TestPl1_panic_PageIterator_bl_out_of_range(t *testing.T) {
	testStdCapacity(t)
	s := pl1Init(t, 80, 24, None[uint64]())
	pl1ExpectPanic(t, func() {
		s.PageIterator(PageListDirectionRightDown, pl1Screen(0, 0), Some(pl1Screen(0, 1000)))
	})
}

// Go-only: panic test: PageList.RowIterator: top point outside the screen (self.pin(tl_pt).?)
func TestPl1_panic_RowIterator_tl_out_of_range(t *testing.T) {
	testStdCapacity(t)
	s := pl1Init(t, 80, 24, None[uint64]())
	pl1ExpectPanic(t, func() {
		s.RowIterator(PageListDirectionRightDown, pl1Screen(0, 1000), None[Point]())
	})
}

// Go-only: panic test: PageList.CellIterator: top point outside the screen (self.pin(tl_pt).?)
func TestPl1_panic_CellIterator_tl_out_of_range(t *testing.T) {
	testStdCapacity(t)
	s := pl1Init(t, 80, 24, None[uint64]())
	pl1ExpectPanic(t, func() {
		s.CellIterator(PageListDirectionRightDown, pl1Screen(0, 1000), None[Point]())
	})
}

// Go-only: panic test: PageList.PromptIterator: top point outside the screen (self.pin(tl_pt).?)
func TestPl1_panic_PromptIterator_tl_out_of_range(t *testing.T) {
	testStdCapacity(t)
	s := pl1Init(t, 80, 24, None[uint64]())
	pl1ExpectPanic(t, func() {
		s.PromptIterator(PageListDirectionRightDown, pl1Screen(0, 1000), None[Point]())
	})
}

// Go-only: panic test: PageList.EraseRow: point outside the screen (self.pin(pt).?)
func TestPl1_panic_EraseRow_out_of_range(t *testing.T) {
	testStdCapacity(t)
	s := pl1Init(t, 80, 24, None[uint64]())
	pl1ExpectPanic(t, func() { _ = s.EraseRow(pl1Screen(0, 1000)) })
}

// Go-only: panic test: PageList.EraseRowBounded: point outside the screen (self.pin(pt).?)
func TestPl1_panic_EraseRowBounded_out_of_range(t *testing.T) {
	testStdCapacity(t)
	s := pl1Init(t, 80, 24, None[uint64]())
	pl1ExpectPanic(t, func() { _ = s.EraseRowBounded(pl1Screen(0, 1000), 3) })
}

// Go-only: panic test: PageList.EraseRows: top point outside the screen (pageIterator: self.pin(tl_pt).?)
func TestPl1_panic_EraseRows_tl_out_of_range(t *testing.T) {
	testStdCapacity(t)
	s := pl1Init(t, 80, 24, None[uint64]())
	pl1ExpectPanic(t, func() { s.EraseRows(pl1Screen(0, 1000), None[Point]()) })
}

// Go-only: panic test: PageList.Clone: top point outside the screen (pageIterator: self.pin(tl_pt).?)
func TestPl1_panic_Clone_top_out_of_range(t *testing.T) {
	testStdCapacity(t)
	s := pl1Init(t, 80, 24, None[uint64]())
	pl1ExpectPanic(t, func() { _, _ = s.Clone(PageListClone{top: pl1Screen(0, 1000)}) })
}

// Go-only: panic test: PageList.IsDirty: point outside the screen (self.getCell(pt).?)
func TestPl1_panic_IsDirty_out_of_range(t *testing.T) {
	testStdCapacity(t)
	s := pl1Init(t, 80, 24, None[uint64]())
	pl1ExpectPanic(t, func() { _ = s.IsDirty(pl1Screen(0, 1000)) })
}

// Go-only: panic test: PageList.markDirty: point outside the screen (self.pin(pt).?)
func TestPl1_panic_markDirty_out_of_range(t *testing.T) {
	testStdCapacity(t)
	s := pl1Init(t, 80, 24, None[uint64]())
	pl1ExpectPanic(t, func() { s.markDirty(pl1Screen(0, 1000)) })
}
