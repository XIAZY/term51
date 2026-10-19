// Tests translated one for one from src/terminal/PageList.zig (Ghostty
// v1.3.1 plus the term51 additional tests), second part: the tests from
// "PageList resize (no reflow) more rows and less cols" up to and including
// "term51: reset: clears history and rebuilds a single page" (Zig lines 10752
// to 16084). Each Go test carries the exact Zig title in a "// Zig: test"
// comment. Helpers are prefixed pl2.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualSlices / expectEqualStrings keep Zig's names
//     and Zig's (expected, actual) argument order. `try` on a call that
//     returns an error is zigTry(t, err); `x.?` is zigUnwrap.
//   - `init(alloc, cols, rows, max_size)` is pl2Init, which calls
//     PageListInit and returns the list by pointer; the integer max_size of
//     the Zig tests is Some(uint64), `null` is None[uint64]().
//   - Every test starts with testStdCapacity(t): Zig's unit tests run with a
//     standard page capacity of grapheme_bytes 512.
//   - Zig's `.{ .x = x, .y = y }` points are pl2Active, pl2Screen, pl2History
//     and pl2Viewport; Zig defaults a Go literal does not give (Resize.reflow,
//     PageIterator.direction) are written out.
//   - `defer s.deinit()` is `defer s.Deinit()`; `testing.allocator` and the
//     `alloc` arguments are dropped.
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt.

package terminal

import (
	"bytes"
	"errors"
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers (prefix pl2).
// ---------------------------------------------------------------------------

// pl2ExpectPanic runs f, which must panic. A panic with the wireframe's
// "not implemented" value does not count: it only means the function under
// test does not exist yet. The setup of the list is done outside f.
func pl2ExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("expected a panic, got none")
		}
		if s, ok := r.(string); ok && s == notImplemented {
			t.Fatalf("expected a panic from the assertion, got: %v", r)
		}
	}()
	f()
}

// pl2ExpectError is Zig's `try testing.expectError(expected, actual)`.
func pl2ExpectError(t *testing.T, want error, got error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

// pl2Init is `try init(alloc, cols, rows, max_size)`: PageListInit, with the
// list returned by pointer (the methods take a pointer receiver).
func pl2Init(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt, max_size Optional[uint64]) *PageList {
	t.Helper()
	s, err := PageListInit(cols, rows, max_size)
	zigTry(t, err)
	return &s
}

// pl2Max is the Zig integer literal given as max_size (`0`, `1000`, ...).
func pl2Max(n uint64) Optional[uint64] { return Some(n) }

// pl2C is a Zig `?size.CellCountInt` option with a value.
func pl2C(n SizeCellCountInt) Optional[SizeCellCountInt] { return Some(n) }

// pl2Active, pl2Screen, pl2History and pl2Viewport are the Zig literals
// `.{ .active = .{ .x = x, .y = y } }` and so on.
func pl2Active(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagActive, Active: PointCoordinate{X: x, Y: y}}
}

func pl2Screen(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagScreen, Screen: PointCoordinate{X: x, Y: y}}
}

func pl2History(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagHistory, History: PointCoordinate{X: x, Y: y}}
}

func pl2Viewport(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagViewport, Viewport: PointCoordinate{X: x, Y: y}}
}

// pl2Pin is `s.pin(pt).?`.
func pl2Pin(t *testing.T, s *PageList, pt Point) PageListPin {
	t.Helper()
	return zigUnwrap(t, s.Pin(pt))
}

// pl2Track is `try s.trackPin(p)`.
func pl2Track(t *testing.T, s *PageList, p PageListPin) *PageListPin {
	t.Helper()
	tp, err := s.TrackPin(p)
	zigTry(t, err)
	return tp
}

// pl2Grow is `_ = try s.grow()`.
func pl2Grow(t *testing.T, s *PageList) *PageListNode {
	t.Helper()
	n, err := s.Grow()
	zigTry(t, err)
	return n
}

// pl2CpCell is the Zig literal `.{ .content_tag = .codepoint, .content =
// .{ .codepoint = cp } }`.
func pl2CpCell(cp uint32) PageCell {
	return pageCellFields{
		content_tag: PageCellContentTagCodepoint,
		content:     pageCellContentFields{codepoint: cp}.pack(),
	}.pack()
}

// pl2RowIterAll walks `s.rowIterator(.right_down, .{ .screen = .{} }, null)`
// and calls f for every pin.
func pl2RowIterAll(s *PageList, f func(offset PageListPin)) {
	it := s.RowIterator(PageListDirectionRightDown, pl2Screen(0, 0), None[Point]())
	for {
		o := it.Next()
		if !o.Valid {
			break
		}
		f(o.Value)
	}
}

// pl2JSetCell is term51JSetCell: writes a codepoint into the cell the pin
// points at.
func pl2JSetCell(p PageListPin, cp uint32) {
	rac := p.RowAndCell()
	*rac.cell = pl2CpCell(cp)
}

// pl2JCp is term51JCp: reads the codepoint at the point (0 for an empty
// cell).
func pl2JCp(t *testing.T, s *PageList, pt Point) uint32 {
	t.Helper()
	return zigUnwrap(t, s.GetCell(pt)).Cell.content().codepoint()
}

// pl2JMarkRows is term51JMarkRows: marks every screen row of the list with
// a distinct codepoint in x=0: row y holds 0x100 + y.
func pl2JMarkRows(t *testing.T, s *PageList) {
	t.Helper()
	total := s.totalRows()
	for y := uint(0); y < total; y++ {
		p := pl2Pin(t, s, pl2Screen(0, uint32(y)))
		pl2JSetCell(p, uint32(0x100+y))
	}
}

// pl2JExpectPages is term51JExpectPages: asserts the list has exactly the
// given per-page row counts, that the intrusive links are consistent, and
// that the cached accounting (total_rows, page_size) matches the pages.
func pl2JExpectPages(t *testing.T, s *PageList, expected []uint) {
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
		count++
	}
	expectEqual(t, uint(len(expected)), count)
	expect(t, s.pages.last == prev, "s.pages.last == prev")
	expectEqual(t, rows, s.total_rows)
	expectEqual(t, bytes, s.page_size)
	expectEqual(t, uint(len(expected)), s.TotalPages())
}

// pl2Implicit is the Zig literal `.{ .implicit = n }` of Hyperlink.Id.
func pl2Implicit(n SizeOffsetInt) HyperlinkHyperlinkId {
	return HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: n}
}

// pl2WideCell is the Zig literal `.{ .content_tag = .codepoint, .content =
// .{ .codepoint = cp }, .wide = wide }`.
func pl2WideCell(cp uint32, wide PageCellWide) PageCell {
	c := pl2CpCell(cp)
	c.setWide(wide)
	return c
}

// pl2Splat is Zig's `@as([n]u21, @splat(cp))`.
func pl2Splat(cp uint32, n int) []uint32 {
	r := make([]uint32, n)
	for i := range r {
		r[i] = cp
	}
	return r
}

// pl2RowIterActive walks `s.rowIterator(.right_down, .{ .active = .{} },
// null)` and calls f for every pin.
func pl2RowIterActive(s *PageList, f func(offset PageListPin)) {
	it := s.RowIterator(PageListDirectionRightDown, pl2Active(0, 0), None[Point]())
	for {
		o := it.Next()
		if !o.Valid {
			break
		}
		f(o.Value)
	}
}

// pl2IncreaseCapacity is `try s.increaseCapacity(node, adjustment)` with a
// non-null adjustment.
func pl2IncreaseCapacity(t *testing.T, s *PageList, node *PageListNode, adjustment PageListIncreaseCapacity) *PageListNode {
	t.Helper()
	n, err := s.IncreaseCapacity(node, Some(adjustment))
	zigTry(t, err)
	return n
}

// pl2StdMaxCols is `std_capacity.maxCols().?`.
func pl2StdMaxCols(t *testing.T) SizeCellCountInt {
	t.Helper()
	return zigUnwrap(t, PageStdCapacity.MaxCols())
}

// ---------------------------------------------------------------------------
// Tests.
// ---------------------------------------------------------------------------

// Zig: test "PageList resize (no reflow) more rows and less cols"
func TestPl2_PageList_resize_no_reflow_more_rows_and_less_cols(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 10, 10, pl2Max(0))
	defer s.Deinit()

	// Resize less
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(5), Rows: pl2C(20), Reflow: false}))
	expectEqual(t, 5, s.cols)
	expectEqual(t, 20, s.rows)
	expectEqual(t, 20, s.totalRows())

	pl2RowIterAll(s, func(offset PageListPin) {
		rac := offset.RowAndCell()
		cells := offset.Node.data.GetCells(rac.row)
		expectEqual(t, 5, len(cells))
	})
}

// Zig: test "PageList resize more rows and cols doesn't fit in single std page"
func TestPl2_PageList_resize_more_rows_and_cols_doesn_t_fit_in_single_std_page(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 10, 10, pl2Max(0))
	defer s.Deinit()

	// Resize to a size that requires more than one page to fit our rows.
	const new_cols = 600
	const new_rows = 600
	cap, err := PageStdCapacity.Adjust(PageCapacityAdjustment{cols: pl2C(new_cols)})
	zigTry(t, err)
	expect(t, cap.rows < new_rows, "cap.rows < new_rows")

	zigTry(t, s.Resize(PageListResize{Cols: pl2C(new_cols), Rows: pl2C(new_rows), Reflow: true}))
	expectEqual(t, new_cols, s.cols)
	expectEqual(t, new_rows, s.rows)
	expectEqual(t, new_rows, s.totalRows())
}

// Zig: test "PageList resize (no reflow) empty screen"
func TestPl2_PageList_resize_no_reflow_empty_screen(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 5, 5, pl2Max(0))
	defer s.Deinit()

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(10), Rows: pl2C(10), Reflow: false}))
	expectEqual(t, 10, s.cols)
	expectEqual(t, 10, s.rows)
	expectEqual(t, 10, s.totalRows())

	pl2RowIterAll(s, func(offset PageListPin) {
		rac := offset.RowAndCell()
		cells := offset.Node.data.GetCells(rac.row)
		expectEqual(t, 10, len(cells))
	})
}

// Zig: test "PageList resize (no reflow) more cols forces smaller cap"
func TestPl2_PageList_resize_no_reflow_more_cols_forces_smaller_cap(t *testing.T) {
	testStdCapacity(t)

	// We want a cap that forces us to have less rows
	cap, err := PageStdCapacity.Adjust(PageCapacityAdjustment{cols: pl2C(100)})
	zigTry(t, err)
	cap2, err := PageStdCapacity.Adjust(PageCapacityAdjustment{cols: pl2C(500)})
	zigTry(t, err)
	expectEqual[SizeCellCountInt](t, 500, cap2.cols)
	expect(t, cap2.rows < cap.rows, "cap2.rows < cap.rows")

	// Create initial cap, fits in one page
	s := pl2Init(t, cap.cols, cap.rows, None[uint64]())
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	for y := uint(0); y < uint(s.rows); y++ {
		for x := uint(0); x < uint(s.cols); x++ {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pl2CpCell('A')
		}
	}

	// Resize to our large cap
	rows := s.totalRows()
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(cap2.cols), Reflow: false}))

	// Our total rows should be the same, and contents should be the same.
	expectEqual(t, rows, s.totalRows())
	pl2RowIterAll(s, func(offset PageListPin) {
		rac := offset.RowAndCell()
		cells := offset.Node.data.GetCells(rac.row)
		expectEqual(t, uint(cap2.cols), uint(len(cells)))
		expectEqual[uint32](t, 'A', cells[0].content().codepoint())
	})
}

// Zig: test "PageList resize (no reflow) more rows adds blank rows if cursor at bottom"
func TestPl2_PageList_resize_no_reflow_more_rows_adds_blank_rows_if_cursor_at_bottom(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 5, 3, None[uint64]())
	defer s.Deinit()

	// Grow to 5 total rows, simulating 3 active + 2 scrollback
	zigTry(t, s.growRows(2))
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	for y := uint(0); y < s.totalRows(); y++ {
		rac := page.GetRowAndCell(0, y)
		*rac.cell = pl2CpCell(uint32(y))
	}

	// Active should be on row 3
	{
		pt := zigUnwrap(t, s.GetCell(pl2Active(0, 0))).ScreenPoint()
		expectEqualPoint(t, pl2Screen(0, 2), pt)
	}

	// Put a tracked pin in the history
	p := pl2Track(t, s, pl2Pin(t, s, pl2Active(0, uint32(s.rows-2))))
	defer s.UntrackPin(p)
	original_cursor := zigUnwrap(t, s.PointFromPin(PointTagActive, *p)).Active
	{
		get := zigUnwrap(t, s.GetCell(pl2Active(original_cursor.X, original_cursor.Y)))
		expectEqual[uint32](t, 3, get.Cell.content().codepoint())
	}

	// Resize
	zigTry(t, s.resizeWithoutReflow(PageListResize{
		Rows:   pl2C(10),
		Reflow: false,
		Cursor: Some(PageListResizeCursor{x: 0, y: s.rows - 2}),
	}))
	expectEqual(t, 5, s.cols)
	expectEqual(t, 10, s.rows)

	// Our cursor should not change
	expectEqual(t, original_cursor, zigUnwrap(t, s.PointFromPin(PointTagActive, *p)).Active)

	// 12 because we have our 10 rows in the active + 2 in the scrollback
	// because we're preserving the cursor.
	expectEqual(t, 12, s.totalRows())

	// Active should be at the same place it was.
	{
		pt := zigUnwrap(t, s.GetCell(pl2Active(0, 0))).ScreenPoint()
		expectEqualPoint(t, pl2Screen(0, 2), pt)
	}

	// Go through our active, we should get only 3,4,5
	for y := uint(0); y < 3; y++ {
		get := zigUnwrap(t, s.GetCell(pl2Active(0, uint32(y))))
		expected := uint32(y + 2)
		expectEqual(t, expected, get.Cell.content().codepoint())
	}
}

// Zig: test "PageList resize reflow more cols no wrapped rows"
func TestPl2_PageList_resize_reflow_more_cols_no_wrapped_rows(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 5, 3, pl2Max(0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	for y := uint(0); y < uint(s.rows); y++ {
		for x := uint(0); x < uint(s.cols); x++ {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pl2CpCell('A')
		}
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(10), Reflow: true}))
	expectEqual(t, 10, s.cols)
	expectEqual(t, 3, s.totalRows())

	pl2RowIterAll(s, func(offset PageListPin) {
		rac := offset.RowAndCell()
		cells := offset.Node.data.GetCells(rac.row)
		expectEqual(t, 10, len(cells))
		expectEqual[uint32](t, 'A', cells[0].content().codepoint())
	})
}

// Zig: test "PageList resize reflow more cols wrapped rows"
func TestPl2_PageList_resize_reflow_more_cols_wrapped_rows(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 2, 4, pl2Max(0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	for y := uint(0); y < uint(s.rows); y++ {
		if y%2 == 0 {
			rac := page.GetRowAndCell(0, y)
			rac.row.wrap = true
		} else {
			rac := page.GetRowAndCell(0, y)
			rac.row.wrap_continuation = true
		}

		for x := uint(0); x < uint(s.cols); x++ {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pl2CpCell('A')
		}
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(4), Reflow: true}))
	expectEqual(t, 4, s.cols)
	expectEqual(t, 4, s.totalRows())

	// Active should still be on top
	{
		pt := zigUnwrap(t, s.GetCell(pl2Active(0, 0))).ScreenPoint()
		expectEqualPoint(t, pl2Screen(0, 0), pt)
	}

	it := s.RowIterator(PageListDirectionRightDown, pl2Screen(0, 0), None[Point]())
	{
		// First row should be unwrapped
		offset := zigUnwrap(t, it.Next())
		rac := offset.RowAndCell()
		cells := offset.Node.data.GetCells(rac.row)
		expect(t, !rac.row.wrap, "!rac.row.wrap")
		expectEqual(t, 4, len(cells))
		expectEqual[uint32](t, 'A', cells[0].content().codepoint())
		expectEqual[uint32](t, 'A', cells[2].content().codepoint())
	}
}

// Zig: test "PageList resize reflow invalidates viewport offset cache"
func TestPl2_PageList_resize_reflow_invalidates_viewport_offset_cache(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 2, 4, None[uint64]())
	defer s.Deinit()
	zigTry(t, s.growRows(20))

	page := &s.pages.last.data
	for y := uint(0); y < uint(s.rows); y++ {
		if y%2 == 0 {
			rac := page.GetRowAndCell(0, y)
			rac.row.wrap = true
		} else {
			rac := page.GetRowAndCell(0, y)
			rac.row.wrap_continuation = true
		}

		for x := uint(0); x < uint(s.cols); x++ {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pl2CpCell('A')
		}
	}

	// Scroll to a pinned viewport in history
	const pin_y = 10
	s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: pl2Pin(t, s, pl2Screen(0, pin_y))})
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, PageListScrollbar{
		Total:  s.total_rows,
		Offset: pin_y,
		Len:    uint(s.rows),
	}, s.Scrollbar())

	// Resize with reflow - unwrapping rows changes total_rows
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(4), Reflow: true}))
	expectEqual(t, 4, s.cols)

	// Verify scrollbar cache was invalidated during reflow
	expectEqual(t, PageListScrollbar{
		Total:  s.total_rows,
		Offset: 5,
		Len:    uint(s.rows),
	}, s.Scrollbar())
}

// Zig: test "PageList resize reflow more cols creates multiple pages"
func TestPl2_PageList_resize_reflow_more_cols_creates_multiple_pages(t *testing.T) {
	testStdCapacity(t)

	// We want a wide viewport so our row limit is rather small. This will
	// force the reflow below to create multiple pages, which we assert.
	cap := func() PageCapacity {
		var current SizeCellCountInt = 100
		for {
			cap, err := PageStdCapacity.Adjust(PageCapacityAdjustment{cols: pl2C(current)})
			zigTry(t, err)
			if cap.rows < 100 {
				return cap
			}
			current += 100
		}
	}()

	s := pl2Init(t, cap.cols, cap.rows, None[uint64]())
	defer s.Deinit()

	// Wrap every other row so every line is wrapped for reflow
	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data
		for y := uint(0); y < uint(s.rows); y++ {
			if y%2 == 0 {
				rac := page.GetRowAndCell(0, y)
				rac.row.wrap = true
			} else {
				rac := page.GetRowAndCell(0, y)
				rac.row.wrap_continuation = true
			}

			rac := page.GetRowAndCell(0, y)
			*rac.cell = pl2CpCell('A')
		}
	}

	// Resize
	newcap, err := cap.Adjust(PageCapacityAdjustment{cols: pl2C(cap.cols + 100)})
	zigTry(t, err)
	expect(t, newcap.rows < cap.rows, "newcap.rows < cap.rows")
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(newcap.cols), Reflow: true}))
	expectEqual(t, uint(newcap.cols), uint(s.cols))
	expectEqual(t, uint(cap.rows), s.totalRows())

	{
		var count uint
		for page := s.pages.first; page != nil; page = page.next {
			count += 1

			// All pages should have the new capacity
			expectEqual(t, newcap.cols, page.data.capacity.cols)
			expectEqual(t, newcap.rows, page.data.capacity.rows)
		}

		// We should have more than one page, meaning we created at least
		// one page. This is the critical aspect of this test so if this
		// ever goes false we need to adjust this test.
		expect(t, count > 1, "count > 1")
	}
}

// Zig: test "PageList resize reflow more cols wrap across page boundary"
func TestPl2_PageList_resize_reflow_more_cols_wrap_across_page_boundary(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 2, 10, pl2Max(0))
	defer s.Deinit()
	expectEqual(t, 1, s.TotalPages())

	// Grow to the capacity of the first page.
	{
		page := &s.pages.first.data
		page.PauseIntegrityChecks(true)
		for i := page.size.rows; i < page.capacity.rows; i++ {
			pl2Grow(t, s)
		}
		page.PauseIntegrityChecks(false)
		expectEqual(t, 1, s.TotalPages())
		zigTry(t, s.growRows(1))
		expectEqual(t, 2, s.TotalPages())
	}

	// At this point, we have some rows on the first page, and some on the second.
	// We can now wrap across the boundary condition.
	{
		page := &s.pages.first.data
		y := uint(page.size.rows - 1)
		{
			rac := page.GetRowAndCell(0, y)
			rac.row.wrap = true
		}
		for x := uint(0); x < uint(s.cols); x++ {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}
	{
		page2 := &s.pages.last.data
		const y = 0
		{
			rac := page2.GetRowAndCell(0, y)
			rac.row.wrap_continuation = true
		}
		for x := uint(0); x < uint(s.cols); x++ {
			rac := page2.GetRowAndCell(x, y)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}

	// PageList.diagram ->
	//
	//       +--+ = PAGE 0
	//   ... :  :
	//      +-----+ ACTIVE
	// 15744 |  | | 0
	// 15745 |  | | 1
	// 15746 |  | | 2
	// 15747 |  | | 3
	// 15748 |  | | 4
	// 15749 |  | | 5
	// 15750 |  | | 6
	// 15751 |  | | 7
	// 15752 |01… | 8
	//       +--+ :
	//       +--+ : = PAGE 1
	//     0 …01| | 9
	//       +--+ :
	//      +-----+

	// We expect one fewer rows since we unwrapped a row.
	end_rows := s.totalRows() - 1

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(4), Reflow: true}))
	expectEqual(t, 4, s.cols)
	expectEqual(t, end_rows, s.totalRows())

	// PageList.diagram ->
	//
	//      +----+ = PAGE 0
	//  ... :    :
	//      +----+
	//      +----+ = PAGE 1
	//  ... :    :
	//     +-------+ ACTIVE
	// 6272 |    | | 0
	// 6273 |    | | 1
	// 6274 |    | | 2
	// 6275 |    | | 3
	// 6276 |    | | 4
	// 6277 |    | | 5
	// 6278 |    | | 6
	// 6279 |    | | 7
	// 6280 |    | | 8
	// 6281 |0101| | 9
	//      +----+ :
	//     +-------+

	{
		// PAGE 1 ROW 6280, ACTIVE 8
		p := pl2Pin(t, s, pl2Active(0, 8))
		row := p.RowAndCell().row
		expect(t, !row.wrap, "!row.wrap")
		expect(t, !row.wrap_continuation, "!row.wrap_continuation")

		cells := p.Cells(PageListPinCellSubsetAll)
		expect(t, !cells[0].HasText(), "!cells[0].hasText()")
		expect(t, !cells[1].HasText(), "!cells[1].hasText()")
		expect(t, !cells[2].HasText(), "!cells[2].hasText()")
		expect(t, !cells[3].HasText(), "!cells[3].hasText()")
	}
	{
		// PAGE 1 ROW 6281, ACTIVE 9
		p := pl2Pin(t, s, pl2Active(0, 9))
		row := p.RowAndCell().row
		expect(t, !row.wrap, "!row.wrap")
		expect(t, !row.wrap_continuation, "!row.wrap_continuation")

		cells := p.Cells(PageListPinCellSubsetAll)
		expectEqual[uint32](t, 0, cells[0].content().codepoint())
		expectEqual[uint32](t, 1, cells[1].content().codepoint())
		expectEqual[uint32](t, 0, cells[2].content().codepoint())
		expectEqual[uint32](t, 1, cells[3].content().codepoint())
	}
}

// Zig: test "PageList resize reflow more cols wrap across page boundary cursor in second page"
func TestPl2_PageList_resize_reflow_more_cols_wrap_across_page_boundary_cursor_in_second_page(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 2, 10, pl2Max(0))
	defer s.Deinit()
	expectEqual(t, 1, s.TotalPages())

	// Grow to the capacity of the first page.
	{
		page := &s.pages.first.data
		page.PauseIntegrityChecks(true)
		for i := page.size.rows; i < page.capacity.rows; i++ {
			pl2Grow(t, s)
		}
		page.PauseIntegrityChecks(false)
		expectEqual(t, 1, s.TotalPages())
		zigTry(t, s.growRows(1))
		expectEqual(t, 2, s.TotalPages())
	}

	// At this point, we have some rows on the first page, and some on the second.
	// We can now wrap across the boundary condition.
	{
		page := &s.pages.first.data
		y := uint(page.size.rows - 1)
		{
			rac := page.GetRowAndCell(0, y)
			rac.row.wrap = true
		}
		for x := uint(0); x < uint(s.cols); x++ {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}
	{
		page2 := &s.pages.last.data
		const y = 0
		{
			rac := page2.GetRowAndCell(0, y)
			rac.row.wrap_continuation = true
		}
		for x := uint(0); x < uint(s.cols); x++ {
			rac := page2.GetRowAndCell(x, y)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}

	// Put a tracked pin in wrapped row on the last page
	p := pl2Track(t, s, pl2Pin(t, s, pl2Active(1, 9)))
	defer s.UntrackPin(p)
	expect(t, p.Node == s.pages.last, "p.node == s.pages.last")

	// We expect one fewer rows since we unwrapped a row.
	end_rows := s.totalRows() - 1

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(4), Reflow: true}))
	expectEqual(t, 4, s.cols)
	expectEqual(t, end_rows, s.totalRows())

	// Our cursor should move to the first row
	expectEqualPoint(t, pl2Active(3, 9), zigUnwrap(t, s.PointFromPin(PointTagActive, *p)))

	{
		p2 := pl2Pin(t, s, pl2Active(0, 9))
		row := p2.RowAndCell().row
		expect(t, !row.wrap, "!row.wrap")

		cells := p2.Cells(PageListPinCellSubsetAll)
		expectEqual[uint32](t, 0, cells[0].content().codepoint())
		expectEqual[uint32](t, 1, cells[1].content().codepoint())
		expectEqual[uint32](t, 0, cells[2].content().codepoint())
		expectEqual[uint32](t, 1, cells[3].content().codepoint())
	}
}

// Zig: test "PageList resize reflow less cols wrap across page boundary cursor in second page"
func TestPl2_PageList_resize_reflow_less_cols_wrap_across_page_boundary_cursor_in_second_page(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 5, 10, None[uint64]())
	defer s.Deinit()
	expectEqual(t, 1, s.TotalPages())

	// Grow to the capacity of the first page.
	{
		page := &s.pages.first.data
		page.PauseIntegrityChecks(true)
		for i := page.size.rows; i < page.capacity.rows; i++ {
			pl2Grow(t, s)
		}
		page.PauseIntegrityChecks(false)
		expectEqual(t, 1, s.TotalPages())
		zigTry(t, s.growRows(5))
		expectEqual(t, 2, s.TotalPages())
	}

	// At this point, we have some rows on the first page, and some on the second.
	// We can now wrap across the boundary condition.
	{
		page := &s.pages.first.data
		y := uint(page.size.rows - 1)
		{
			rac := page.GetRowAndCell(0, y)
			rac.row.wrap = true
		}
		for x := uint(0); x < uint(s.cols); x++ {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}
	{
		page2 := &s.pages.last.data
		const y = 0
		{
			rac := page2.GetRowAndCell(0, y)
			rac.row.wrap_continuation = true
		}
		for x := uint(0); x < uint(s.cols); x++ {
			rac := page2.GetRowAndCell(x, y)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}

	// Put a tracked pin in wrapped row on the last page
	p := pl2Track(t, s, pl2Pin(t, s, pl2Active(2, 5)))
	defer s.UntrackPin(p)
	expect(t, p.Node == s.pages.last, "p.node == s.pages.last")
	expect(t, p.Y == 0, "p.y == 0")

	// PageList.diagram ->
	//
	//      +-----+ = PAGE 0
	//  ... :     :
	//     +--------+ ACTIVE
	// 7892 |     | | 0
	// 7893 |     | | 1
	// 7894 |     | | 2
	// 7895 |     | | 3
	// 7896 |01234… | 4
	//      +-----+ :
	//      +-----+ : = PAGE 1
	//    0 …01234| | 5
	//      :  ^  : : = PIN 0
	//    1 |     | | 6
	//    2 |     | | 7
	//    3 |     | | 8
	//    4 |     | | 9
	//      +-----+ :
	//     +--------+

	// Resize
	zigTry(t, s.Resize(PageListResize{
		Cols:   pl2C(4),
		Reflow: true,
		Cursor: Some(PageListResizeCursor{x: 2, y: 5}),
	}))
	expectEqual(t, 4, s.cols)

	// PageList.diagram ->
	//
	//      +----+ = PAGE 0
	//  ... :    :
	//     +-------+ ACTIVE
	// 7892 |    | | 0
	// 7893 |    | | 1
	// 7894 |    | | 2
	// 7895 |    | | 3
	// 7896 |0123… | 4
	// 7897 …4012… | 5
	//      :   ^: : = PIN 0
	// 7898 …3400| | 6
	// 7899 |    | | 7
	// 7900 |    | | 8
	// 7901 |    | | 9
	//      +----+ :
	//     +-------+

	// Our cursor should remain on the same cell
	expectEqualPoint(t, pl2Active(3, 5), zigUnwrap(t, s.PointFromPin(PointTagActive, *p)))

	{
		// PAGE 0 ROW 7895, ACTIVE 3
		p2 := pl2Pin(t, s, pl2Active(0, 3))
		row := p2.RowAndCell().row
		expect(t, !row.wrap, "!row.wrap")
		expect(t, !row.wrap_continuation, "!row.wrap_continuation")

		cells := p2.Cells(PageListPinCellSubsetAll)
		expect(t, !cells[0].HasText(), "!cells[0].hasText()")
		expect(t, !cells[1].HasText(), "!cells[1].hasText()")
		expect(t, !cells[2].HasText(), "!cells[2].hasText()")
		expect(t, !cells[3].HasText(), "!cells[3].hasText()")
	}
	{
		// PAGE 0 ROW 7896, ACTIVE 4
		p2 := pl2Pin(t, s, pl2Active(0, 4))
		row := p2.RowAndCell().row
		expect(t, row.wrap, "row.wrap")
		expect(t, !row.wrap_continuation, "!row.wrap_continuation")

		cells := p2.Cells(PageListPinCellSubsetAll)
		expectEqual[uint32](t, 0, cells[0].content().codepoint())
		expectEqual[uint32](t, 1, cells[1].content().codepoint())
		expectEqual[uint32](t, 2, cells[2].content().codepoint())
		expectEqual[uint32](t, 3, cells[3].content().codepoint())
	}
	{
		// PAGE 0 ROW 7897, ACTIVE 5
		p2 := pl2Pin(t, s, pl2Active(0, 5))
		row := p2.RowAndCell().row
		expect(t, row.wrap, "row.wrap")
		expect(t, row.wrap_continuation, "row.wrap_continuation")

		cells := p2.Cells(PageListPinCellSubsetAll)
		expectEqual[uint32](t, 4, cells[0].content().codepoint())
		expectEqual[uint32](t, 0, cells[1].content().codepoint())
		expectEqual[uint32](t, 1, cells[2].content().codepoint())
		expectEqual[uint32](t, 2, cells[3].content().codepoint())
	}
	{
		// PAGE 0 ROW 7898, ACTIVE 6
		p2 := pl2Pin(t, s, pl2Active(0, 6))
		row := p2.RowAndCell().row
		expect(t, !row.wrap, "!row.wrap")
		expect(t, row.wrap_continuation, "row.wrap_continuation")

		cells := p2.Cells(PageListPinCellSubsetAll)
		expectEqual[uint32](t, 3, cells[0].content().codepoint())
		expectEqual[uint32](t, 4, cells[1].content().codepoint())
	}
	{
		// PAGE 0 ROW 7899, ACTIVE 7
		p2 := pl2Pin(t, s, pl2Active(0, 7))
		row := p2.RowAndCell().row
		expect(t, !row.wrap, "!row.wrap")
		expect(t, !row.wrap_continuation, "!row.wrap_continuation")

		cells := p2.Cells(PageListPinCellSubsetAll)
		expect(t, !cells[0].HasText(), "!cells[0].hasText()")
		expect(t, !cells[1].HasText(), "!cells[1].hasText()")
		expect(t, !cells[2].HasText(), "!cells[2].hasText()")
		expect(t, !cells[3].HasText(), "!cells[3].hasText()")
	}
}

// Zig: test "PageList resize reflow more cols cursor in wrapped row"
func TestPl2_PageList_resize_reflow_more_cols_cursor_in_wrapped_row(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 2, 4, pl2Max(0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	{
		{
			rac := page.GetRowAndCell(0, 0)
			rac.row.wrap = true
		}
		for x := uint(0); x < uint(s.cols); x++ {
			rac := page.GetRowAndCell(x, 0)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}
	{
		{
			rac := page.GetRowAndCell(0, 1)
			rac.row.wrap_continuation = true
		}
		for x := uint(0); x < uint(s.cols); x++ {
			rac := page.GetRowAndCell(x, 1)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}

	// Put a tracked pin in the history
	p := pl2Track(t, s, pl2Pin(t, s, pl2Active(1, 1)))
	defer s.UntrackPin(p)

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(4), Reflow: true}))
	expectEqual(t, 4, s.cols)
	expectEqual(t, 4, s.totalRows())

	// Our cursor should move to the first row
	expectEqualPoint(t, pl2Active(3, 0), zigUnwrap(t, s.PointFromPin(PointTagActive, *p)))
}

// Zig: test "PageList resize reflow more cols cursor in not wrapped row"
func TestPl2_PageList_resize_reflow_more_cols_cursor_in_not_wrapped_row(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 2, 4, pl2Max(0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	{
		{
			rac := page.GetRowAndCell(0, 0)
			rac.row.wrap = true
		}
		for x := uint(0); x < uint(s.cols); x++ {
			rac := page.GetRowAndCell(x, 0)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}
	{
		{
			rac := page.GetRowAndCell(0, 1)
			rac.row.wrap_continuation = true
		}
		for x := uint(0); x < uint(s.cols); x++ {
			rac := page.GetRowAndCell(x, 1)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}

	// Put a tracked pin in the history
	p := pl2Track(t, s, pl2Pin(t, s, pl2Active(1, 0)))
	defer s.UntrackPin(p)

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(4), Reflow: true}))
	expectEqual(t, 4, s.cols)
	expectEqual(t, 4, s.totalRows())

	// Our cursor should move to the first row
	expectEqualPoint(t, pl2Active(1, 0), zigUnwrap(t, s.PointFromPin(PointTagActive, *p)))
}

// Zig: test "PageList resize reflow more cols cursor in wrapped row that isn't unwrapped"
func TestPl2_PageList_resize_reflow_more_cols_cursor_in_wrapped_row_that_isn_t_unwrapped(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 2, 4, pl2Max(0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	{
		{
			rac := page.GetRowAndCell(0, 0)
			rac.row.wrap = true
		}
		for x := uint(0); x < uint(s.cols); x++ {
			rac := page.GetRowAndCell(x, 0)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}
	{
		{
			rac := page.GetRowAndCell(0, 1)
			rac.row.wrap = true
			rac.row.wrap_continuation = true
		}
		for x := uint(0); x < uint(s.cols); x++ {
			rac := page.GetRowAndCell(x, 1)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}
	{
		{
			rac := page.GetRowAndCell(0, 2)
			rac.row.wrap_continuation = true
		}
		for x := uint(0); x < uint(s.cols); x++ {
			rac := page.GetRowAndCell(x, 2)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}

	// Put a tracked pin in the history
	p := pl2Track(t, s, pl2Pin(t, s, pl2Active(1, 2)))
	defer s.UntrackPin(p)

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(4), Reflow: true}))
	expectEqual(t, 4, s.cols)
	expectEqual(t, 4, s.totalRows())

	// Our cursor should move to the first row
	expectEqualPoint(t, pl2Active(1, 1), zigUnwrap(t, s.PointFromPin(PointTagActive, *p)))
}

// Zig: test "PageList resize reflow more cols no reflow preserves semantic prompt"
func TestPl2_PageList_resize_reflow_more_cols_no_reflow_preserves_semantic_prompt(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 2, 4, pl2Max(0))
	defer s.Deinit()
	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data
		rac := page.GetRowAndCell(0, 1)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(4), Reflow: true}))
	expectEqual(t, 4, s.cols)
	expectEqual(t, 4, s.totalRows())

	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data
		rac := page.GetRowAndCell(0, 1)
		expect(t, rac.row.semantic_prompt == PageRowSemanticPromptPrompt, "rac.row.semantic_prompt == .prompt")
	}
}

// Zig: test "PageList resize reflow exceeds hyperlink memory forcing capacity increase"
func TestPl2_PageList_resize_reflow_exceeds_hyperlink_memory_forcing_capacity_increase(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 2, 10, pl2Max(0))
	defer s.Deinit()
	expectEqual(t, 1, s.TotalPages())

	// Grow to the capacity of the first page and add
	// one more row so that we have two pages total.
	{
		page := &s.pages.first.data
		page.PauseIntegrityChecks(true)
		for i := page.size.rows; i < page.capacity.rows; i++ {
			pl2Grow(t, s)
		}
		page.PauseIntegrityChecks(false)
		expectEqual(t, 1, s.TotalPages())
		zigTry(t, s.growRows(1))
		expectEqual(t, 2, s.TotalPages())

		// We now have two pages.
		expect(t, s.pages.first != s.pages.last, "s.pages.first.? != s.pages.last.?")
		expectEqual(t, s.pages.last, s.pages.first.next)
	}

	// We use almost all string alloc capacity with a hyperlink in the final
	// row of the first page, and do the same on the first row of the second
	// page. We also mark the row as wrapped so that when we resize with more
	// cols the row unwraps and we have a single row that requires almost two
	// times the base string alloc capacity.
	//
	// This forces the reflow to increase capacity.
	//
	//  +--+ = PAGE 0
	//  :  :
	//  | X… <- where X is hyperlinked with almost all string cap.
	//  +--+
	//  +--+ = PAGE 1
	//  …X | <- X here also almost hits string cap with a hyperlink.
	//  +--+

	// Almost hit string alloc cap in bottom right of first page.
	// Mark the final row as wrapped.
	{
		page := &s.pages.first.data
		id, err := page.InsertHyperlink(Hyperlink{
			ID:  pl2Implicit(0),
			URI: bytes.Repeat([]byte("a"), PageStringBytesDefault-1),
		})
		zigTry(t, err)
		rac := page.GetRowAndCell(uint(page.size.cols-1), uint(page.size.rows-1))
		rac.row.wrap = true
		*rac.cell = pl2CpCell('X')
		zigTry(t, page.SetHyperlink(rac.row, rac.cell, id))
		_, err = page.InsertHyperlink(Hyperlink{
			ID:  pl2Implicit(1),
			URI: []byte("AAAAAAAAAAAAAAAAAAAAAAAAAA"),
		})
		pl2ExpectError(t, ErrStringsOutOfMemory, err)
	}

	// Almost hit string alloc cap in top left of second page.
	// Mark the first row as a wrap continuation.
	{
		page := &s.pages.last.data
		id, err := page.InsertHyperlink(Hyperlink{
			ID:  pl2Implicit(1),
			URI: bytes.Repeat([]byte("a"), PageStringBytesDefault-1),
		})
		zigTry(t, err)
		rac := page.GetRowAndCell(0, 0)
		rac.row.wrap_continuation = true
		*rac.cell = pl2CpCell('X')
		zigTry(t, page.SetHyperlink(rac.row, rac.cell, id))
		_, err = page.InsertHyperlink(Hyperlink{
			ID:  pl2Implicit(2),
			URI: []byte("AAAAAAAAAAAAAAAAAAAAAAAAAA"),
		})
		pl2ExpectError(t, ErrStringsOutOfMemory, err)
	}

	// Resize to 1 column wider, unwrapping the row.
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(s.cols + 1), Reflow: true}))
}

// Zig: test "PageList resize reflow exceeds grapheme memory forcing capacity increase"
func TestPl2_PageList_resize_reflow_exceeds_grapheme_memory_forcing_capacity_increase(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 2, 10, pl2Max(0))
	defer s.Deinit()
	expectEqual(t, 1, s.TotalPages())

	// Grow to the capacity of the first page and add
	// one more row so that we have two pages total.
	{
		page := &s.pages.first.data
		page.PauseIntegrityChecks(true)
		for i := page.size.rows; i < page.capacity.rows; i++ {
			pl2Grow(t, s)
		}
		page.PauseIntegrityChecks(false)
		expectEqual(t, 1, s.TotalPages())
		zigTry(t, s.growRows(1))
		expectEqual(t, 2, s.TotalPages())

		// We now have two pages.
		expect(t, s.pages.first != s.pages.last, "s.pages.first.? != s.pages.last.?")
		expectEqual(t, s.pages.last, s.pages.first.next)
	}

	// We use almost all grapheme alloc capacity with a grapheme in the final
	// row of the first page, and do the same on the first row of the second
	// page. We also mark the row as wrapped so that when we resize with more
	// cols the row unwraps and we have a single row that requires almost two
	// times the base grapheme alloc capacity.
	//
	// This forces the reflow to increase capacity.
	//
	//  +--+ = PAGE 0
	//  :  :
	//  | X… <- where X is a grapheme which uses almost all the capacity.
	//  +--+
	//  +--+ = PAGE 1
	//  …X | <- X here also almost hits grapheme cap.
	//  +--+

	// Almost hit grapheme alloc cap in bottom right of first page.
	// Mark the final row as wrapped.
	{
		page := &s.pages.first.data
		rac := page.GetRowAndCell(uint(page.size.cols-1), uint(page.size.rows-1))
		rac.row.wrap = true
		*rac.cell = pl2CpCell('X')
		zigTry(t, page.SetGraphemes(rac.row, rac.cell, pl2Splat('a', (PageGraphemeBytesDefault-1)/4)))
		_, err := page.grapheme_alloc.Alloc(16)
		pl2ExpectError(t, ErrOutOfMemory, err)
	}

	// Almost hit grapheme alloc cap in top left of second page.
	// Mark the first row as a wrap continuation.
	{
		page := &s.pages.last.data
		rac := page.GetRowAndCell(0, 0)
		rac.row.wrap = true
		*rac.cell = pl2CpCell('X')
		zigTry(t, page.SetGraphemes(rac.row, rac.cell, pl2Splat('a', (PageGraphemeBytesDefault-1)/4)))
		_, err := page.grapheme_alloc.Alloc(16)
		pl2ExpectError(t, ErrOutOfMemory, err)
	}

	// Resize to 1 column wider, unwrapping the row.
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(s.cols + 1), Reflow: true}))
}

// Zig: test "PageList resize reflow exceeds style memory forcing capacity increase"
func TestPl2_PageList_resize_reflow_exceeds_style_memory_forcing_capacity_increase(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, PageStdCapacity.styles-1, 10, pl2Max(0))
	defer s.Deinit()
	expectEqual(t, 1, s.TotalPages())

	// Grow to the capacity of the first page and add
	// one more row so that we have two pages total.
	{
		page := &s.pages.first.data
		page.PauseIntegrityChecks(true)
		for i := page.size.rows; i < page.capacity.rows; i++ {
			pl2Grow(t, s)
		}
		page.PauseIntegrityChecks(false)
		expectEqual(t, 1, s.TotalPages())
		zigTry(t, s.growRows(1))
		expectEqual(t, 2, s.TotalPages())

		// We now have two pages.
		expect(t, s.pages.first != s.pages.last, "s.pages.first.? != s.pages.last.?")
		expectEqual(t, s.pages.last, s.pages.first.next)
	}

	// Give each cell in the final row of the first page a unique style.
	// Mark the final row as wrapped.
	{
		page := &s.pages.first.data
		for x := uint(0); x < uint(s.cols); x++ {
			id, err := page.styles.Add(Style{
				BgColor: StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{
					R: uint8(x),
					G: uint8(x >> 8),
					B: uint8(x >> 16),
				}},
			})
			if err != nil {
				break
			}

			rac := page.GetRowAndCell(x, uint(page.size.rows-1))
			rac.row.wrap = true
			rac.row.styled = true
			*rac.cell = pageCellFields{
				content_tag: PageCellContentTagCodepoint,
				content:     pageCellContentFields{codepoint: 'X'}.pack(),
				style_id:    id,
			}.pack()
		}
	}

	// Do the same for the first row of the second page.
	// Mark the first row as a wrap continuation.
	{
		page := &s.pages.last.data
		for x := uint(0); x < uint(s.cols); x++ {
			id, err := page.styles.Add(Style{
				FgColor: StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{
					R: uint8(x),
					G: uint8(x >> 8),
					B: uint8(x >> 16),
				}},
			})
			if err != nil {
				break
			}

			rac := page.GetRowAndCell(x, 0)
			rac.row.wrap_continuation = true
			rac.row.styled = true
			*rac.cell = pageCellFields{
				content_tag: PageCellContentTagCodepoint,
				content:     pageCellContentFields{codepoint: 'X'}.pack(),
				style_id:    id,
			}.pack()
		}
	}

	// Resize to twice as wide, fully unwrapping the row.
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(s.cols * 2), Reflow: true}))
}

// Zig: test "PageList resize reflow more cols unwrap wide spacer head"
func TestPl2_PageList_resize_reflow_more_cols_unwrap_wide_spacer_head(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 2, 2, pl2Max(0))
	defer s.Deinit()
	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		{
			rac := page.GetRowAndCell(0, 0)
			rac.row.wrap = true
			*rac.cell = pl2CpCell('x')
		}
		{
			rac := page.GetRowAndCell(1, 0)
			*rac.cell = pl2WideCell(0, PageCellWideSpacerHead)
		}
		{
			rac := page.GetRowAndCell(0, 1)
			rac.row.wrap_continuation = true
			*rac.cell = pl2WideCell('😀', PageCellWideWide)
		}
		{
			rac := page.GetRowAndCell(1, 1)
			*rac.cell = pl2WideCell(0, PageCellWideSpacerTail)
		}
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(4), Reflow: true}))
	expectEqual(t, 4, s.cols)
	expectEqual(t, 2, s.totalRows())

	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		{
			rac := page.GetRowAndCell(0, 0)
			expectEqual[uint32](t, 'x', rac.cell.content().codepoint())
			expectEqual(t, PageCellWideNarrow, rac.cell.Wide())
			expect(t, !rac.row.wrap, "!rac.row.wrap")
		}
		{
			rac := page.GetRowAndCell(1, 0)
			expectEqual[uint32](t, '😀', rac.cell.content().codepoint())
			expectEqual(t, PageCellWideWide, rac.cell.Wide())
		}
		{
			rac := page.GetRowAndCell(2, 0)
			expectEqual[uint32](t, 0, rac.cell.content().codepoint())
			expectEqual(t, PageCellWideSpacerTail, rac.cell.Wide())
		}
	}
}

// Zig: test "PageList resize reflow more cols unwrap wide spacer head across two rows"
func TestPl2_PageList_resize_reflow_more_cols_unwrap_wide_spacer_head_across_two_rows(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 2, 3, pl2Max(0))
	defer s.Deinit()
	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		{
			rac := page.GetRowAndCell(0, 0)
			rac.row.wrap = true
			*rac.cell = pl2CpCell('x')
		}
		{
			rac := page.GetRowAndCell(1, 0)
			*rac.cell = pl2CpCell('x')
		}
		{
			rac := page.GetRowAndCell(0, 1)
			rac.row.wrap_continuation = true
			rac.row.wrap = true
			*rac.cell = pl2CpCell('x')
		}
		{
			rac := page.GetRowAndCell(1, 1)
			*rac.cell = pl2WideCell(0, PageCellWideSpacerHead)
		}
		{
			rac := page.GetRowAndCell(0, 2)
			rac.row.wrap_continuation = true
			*rac.cell = pl2WideCell('😀', PageCellWideWide)
		}
		{
			rac := page.GetRowAndCell(1, 2)
			*rac.cell = pl2WideCell(0, PageCellWideSpacerTail)
		}
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(4), Reflow: true}))
	expectEqual(t, 4, s.cols)
	expectEqual(t, 3, s.totalRows())

	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		{
			rac := page.GetRowAndCell(0, 0)
			expectEqual[uint32](t, 'x', rac.cell.content().codepoint())
			expectEqual(t, PageCellWideNarrow, rac.cell.Wide())
			expect(t, rac.row.wrap, "rac.row.wrap")
		}
		{
			rac := page.GetRowAndCell(1, 0)
			expectEqual[uint32](t, 'x', rac.cell.content().codepoint())
			expectEqual(t, PageCellWideNarrow, rac.cell.Wide())
		}
		{
			rac := page.GetRowAndCell(2, 0)
			expectEqual[uint32](t, 'x', rac.cell.content().codepoint())
			expectEqual(t, PageCellWideNarrow, rac.cell.Wide())
		}
		{
			rac := page.GetRowAndCell(3, 0)
			expectEqual[uint32](t, 0, rac.cell.content().codepoint())
			expectEqual(t, PageCellWideSpacerHead, rac.cell.Wide())
		}
		{
			rac := page.GetRowAndCell(0, 1)
			expectEqual[uint32](t, '😀', rac.cell.content().codepoint())
			expectEqual(t, PageCellWideWide, rac.cell.Wide())
		}
		{
			rac := page.GetRowAndCell(1, 1)
			expectEqual[uint32](t, 0, rac.cell.content().codepoint())
			expectEqual(t, PageCellWideSpacerTail, rac.cell.Wide())
		}
	}
}

// Zig: test "PageList resize reflow more cols unwrap still requires wide spacer head"
func TestPl2_PageList_resize_reflow_more_cols_unwrap_still_requires_wide_spacer_head(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 2, 2, pl2Max(0))
	defer s.Deinit()
	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		{
			rac := page.GetRowAndCell(0, 0)
			rac.row.wrap = true
			*rac.cell = pl2CpCell('x')
		}
		{
			rac := page.GetRowAndCell(1, 0)
			*rac.cell = pl2CpCell('x')
		}
		{
			rac := page.GetRowAndCell(0, 1)
			rac.row.wrap_continuation = true
			*rac.cell = pl2WideCell('😀', PageCellWideWide)
		}
		{
			rac := page.GetRowAndCell(1, 1)
			*rac.cell = pl2WideCell(0, PageCellWideSpacerTail)
		}
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(3), Reflow: true}))
	expectEqual(t, 3, s.cols)
	expectEqual(t, 2, s.totalRows())

	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		{
			rac := page.GetRowAndCell(0, 0)
			expectEqual[uint32](t, 'x', rac.cell.content().codepoint())
			expectEqual(t, PageCellWideNarrow, rac.cell.Wide())
			expect(t, rac.row.wrap, "rac.row.wrap")
		}
		{
			rac := page.GetRowAndCell(1, 0)
			expectEqual[uint32](t, 'x', rac.cell.content().codepoint())
			expectEqual(t, PageCellWideNarrow, rac.cell.Wide())
		}
		{
			rac := page.GetRowAndCell(2, 0)
			expectEqual[uint32](t, 0, rac.cell.content().codepoint())
			expectEqual(t, PageCellWideSpacerHead, rac.cell.Wide())
		}
		{
			rac := page.GetRowAndCell(0, 1)
			expectEqual[uint32](t, '😀', rac.cell.content().codepoint())
			expectEqual(t, PageCellWideWide, rac.cell.Wide())
		}
		{
			rac := page.GetRowAndCell(1, 1)
			expectEqual[uint32](t, 0, rac.cell.content().codepoint())
			expectEqual(t, PageCellWideSpacerTail, rac.cell.Wide())
		}
	}
}

// Zig: test "PageList resize reflow less cols no reflow preserves semantic prompt"
func TestPl2_PageList_resize_reflow_less_cols_no_reflow_preserves_semantic_prompt(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 4, 4, pl2Max(0))
	defer s.Deinit()
	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data
		{
			rac := page.GetRowAndCell(0, 1)
			rac.row.semantic_prompt = PageRowSemanticPromptPrompt
		}
		for x := uint(0); x < uint(s.cols); x++ {
			rac := page.GetRowAndCell(x, 1)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(2), Reflow: true}))
	expectEqual(t, 2, s.cols)
	expectEqual(t, 4, s.totalRows())

	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		{
			p := pl2Pin(t, s, pl2Active(0, 1))
			rac := p.RowAndCell()
			expect(t, rac.row.wrap, "rac.row.wrap")
			expect(t, rac.row.semantic_prompt == PageRowSemanticPromptPrompt, "rac.row.semantic_prompt == .prompt")
		}
		{
			p := pl2Pin(t, s, pl2Active(0, 2))
			rac := p.RowAndCell()
			expect(t, rac.row.semantic_prompt == PageRowSemanticPromptPrompt, "rac.row.semantic_prompt == .prompt")
		}
	}
}

// Zig: test "PageList resize reflow less cols no reflow preserves semantic prompt on first line"
func TestPl2_PageList_resize_reflow_less_cols_no_reflow_preserves_semantic_prompt_on_first_line(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 4, 4, pl2Max(0))
	defer s.Deinit()
	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data
		rac := page.GetRowAndCell(0, 0)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(2), Reflow: true}))
	expectEqual(t, 2, s.cols)
	expectEqual(t, 4, s.totalRows())

	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data
		rac := page.GetRowAndCell(0, 0)
		expect(t, rac.row.semantic_prompt == PageRowSemanticPromptPrompt, "rac.row.semantic_prompt == .prompt")
	}
}

// Zig: test "PageList resize reflow less cols wrap preserves semantic prompt"
func TestPl2_PageList_resize_reflow_less_cols_wrap_preserves_semantic_prompt(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 4, 4, pl2Max(0))
	defer s.Deinit()
	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data
		rac := page.GetRowAndCell(0, 0)
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(2), Reflow: true}))
	expectEqual(t, 2, s.cols)
	expectEqual(t, 4, s.totalRows())

	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data
		rac := page.GetRowAndCell(0, 0)
		expect(t, rac.row.semantic_prompt == PageRowSemanticPromptPrompt, "rac.row.semantic_prompt == .prompt")
	}
}

// Zig: test "PageList resize reflow less cols no wrapped rows"
func TestPl2_PageList_resize_reflow_less_cols_no_wrapped_rows(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 10, 3, pl2Max(0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	for y := uint(0); y < uint(s.rows); y++ {
		const end = 4
		assert(end < s.cols)
		for x := uint(0); x < 4; x++ {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(5), Reflow: true}))
	expectEqual(t, 5, s.cols)
	expectEqual(t, 3, s.totalRows())

	pl2RowIterAll(s, func(offset PageListPin) {
		for x := uint(0); x < 4; x++ {
			offset_copy := offset
			offset_copy.X = SizeCellCountInt(x)
			rac := offset_copy.RowAndCell()
			cells := offset.Node.data.GetCells(rac.row)
			expectEqual(t, 5, len(cells))
			expectEqual(t, uint32(x), cells[x].content().codepoint())
		}
	})
}

// Zig: test "PageList resize reflow less cols wrapped rows"
func TestPl2_PageList_resize_reflow_less_cols_wrapped_rows(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 4, 2, None[uint64]())
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	for y := uint(0); y < uint(s.rows); y++ {
		for x := uint(0); x < uint(s.cols); x++ {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(2), Reflow: true}))
	expectEqual(t, 2, s.cols)
	expectEqual(t, 4, s.totalRows())

	// Active moves due to scrollback
	{
		pt := zigUnwrap(t, s.GetCell(pl2Active(0, 0))).ScreenPoint()
		expectEqualPoint(t, pl2Screen(0, 2), pt)
	}

	it := s.RowIterator(PageListDirectionRightDown, pl2Screen(0, 0), None[Point]())
	{
		// First row should be wrapped
		offset := zigUnwrap(t, it.Next())
		rac := offset.RowAndCell()
		cells := offset.Node.data.GetCells(rac.row)
		expect(t, rac.row.wrap, "rac.row.wrap")
		expectEqual(t, 2, len(cells))
		expectEqual[uint32](t, 0, cells[0].content().codepoint())
	}
	{
		offset := zigUnwrap(t, it.Next())
		rac := offset.RowAndCell()
		cells := offset.Node.data.GetCells(rac.row)
		expect(t, !rac.row.wrap, "!rac.row.wrap")
		expectEqual(t, 2, len(cells))
		expectEqual[uint32](t, 2, cells[0].content().codepoint())
	}
	{
		// First row should be wrapped
		offset := zigUnwrap(t, it.Next())
		rac := offset.RowAndCell()
		cells := offset.Node.data.GetCells(rac.row)
		expect(t, rac.row.wrap, "rac.row.wrap")
		expectEqual(t, 2, len(cells))
		expectEqual[uint32](t, 0, cells[0].content().codepoint())
	}
	{
		offset := zigUnwrap(t, it.Next())
		rac := offset.RowAndCell()
		cells := offset.Node.data.GetCells(rac.row)
		expect(t, !rac.row.wrap, "!rac.row.wrap")
		expectEqual(t, 2, len(cells))
		expectEqual[uint32](t, 2, cells[0].content().codepoint())
	}
}

// Zig: test "PageList resize reflow less cols wrapped rows with graphemes"
func TestPl2_PageList_resize_reflow_less_cols_wrapped_rows_with_graphemes(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 4, 2, None[uint64]())
	defer s.Deinit()
	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data
		for y := uint(0); y < uint(s.rows); y++ {
			for x := uint(0); x < uint(s.cols); x++ {
				rac := page.GetRowAndCell(x, y)
				*rac.cell = pl2CpCell(uint32(x))
			}

			rac := page.GetRowAndCell(2, y)
			zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 'A'))
		}
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(2), Reflow: true}))
	expectEqual(t, 2, s.cols)
	expectEqual(t, 4, s.totalRows())

	// Active moves due to scrollback
	{
		pt := zigUnwrap(t, s.GetCell(pl2Active(0, 0))).ScreenPoint()
		expectEqualPoint(t, pl2Screen(0, 2), pt)
	}

	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	it := s.RowIterator(PageListDirectionRightDown, pl2Screen(0, 0), None[Point]())
	{
		// First row should be wrapped
		offset := zigUnwrap(t, it.Next())
		rac := offset.RowAndCell()
		cells := offset.Node.data.GetCells(rac.row)
		expect(t, rac.row.wrap, "rac.row.wrap")
		expectEqual(t, 2, len(cells))
		expectEqual[uint32](t, 0, cells[0].content().codepoint())
	}
	{
		offset := zigUnwrap(t, it.Next())
		rac := offset.RowAndCell()
		cells := offset.Node.data.GetCells(rac.row)
		expect(t, !rac.row.wrap, "!rac.row.wrap")
		expect(t, rac.row.grapheme, "rac.row.grapheme")
		expectEqual(t, 2, len(cells))
		expectEqual[uint32](t, 2, cells[0].content().codepoint())

		cps := zigUnwrap(t, page.LookupGrapheme(rac.cell))
		expectEqual(t, 1, len(cps))
		expectEqual[uint32](t, 'A', cps[0])
	}
	{
		// First row should be wrapped
		offset := zigUnwrap(t, it.Next())
		rac := offset.RowAndCell()
		cells := offset.Node.data.GetCells(rac.row)
		expect(t, rac.row.wrap, "rac.row.wrap")
		expectEqual(t, 2, len(cells))
		expectEqual[uint32](t, 0, cells[0].content().codepoint())
	}
	{
		offset := zigUnwrap(t, it.Next())
		rac := offset.RowAndCell()
		cells := offset.Node.data.GetCells(rac.row)
		expect(t, !rac.row.wrap, "!rac.row.wrap")
		expect(t, rac.row.grapheme, "rac.row.grapheme")
		expectEqual(t, 2, len(cells))
		expectEqual[uint32](t, 2, cells[0].content().codepoint())

		cps := zigUnwrap(t, page.LookupGrapheme(rac.cell))
		expectEqual(t, 1, len(cps))
		expectEqual[uint32](t, 'A', cps[0])
	}
}

// Zig: test "PageList resize reflow less cols cursor in wrapped row"
func TestPl2_PageList_resize_reflow_less_cols_cursor_in_wrapped_row(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 4, 2, None[uint64]())
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	for y := uint(0); y < uint(s.rows); y++ {
		for x := uint(0); x < uint(s.cols); x++ {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}

	// Put a tracked pin in the history
	p := pl2Track(t, s, pl2Pin(t, s, pl2Active(2, 1)))
	defer s.UntrackPin(p)

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(2), Reflow: true}))
	expectEqual(t, 2, s.cols)
	expectEqual(t, 4, s.totalRows())

	// Our cursor should move to the first row
	expectEqualPoint(t, pl2Active(0, 1), zigUnwrap(t, s.PointFromPin(PointTagActive, *p)))
}

// Zig: test "PageList resize reflow less cols wraps spacer head"
func TestPl2_PageList_resize_reflow_less_cols_wraps_spacer_head(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 4, 3, pl2Max(0))
	defer s.Deinit()
	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		{
			rac := page.GetRowAndCell(0, 0)
			rac.row.wrap = true
			*rac.cell = pl2CpCell('x')
		}
		{
			rac := page.GetRowAndCell(1, 0)
			*rac.cell = pl2CpCell('x')
		}
		{
			rac := page.GetRowAndCell(2, 0)
			*rac.cell = pl2CpCell('x')
		}
		{
			rac := page.GetRowAndCell(3, 0)
			*rac.cell = pl2WideCell(0, PageCellWideSpacerHead)
		}
		{
			rac := page.GetRowAndCell(0, 1)
			rac.row.wrap_continuation = true
			*rac.cell = pl2WideCell('😀', PageCellWideWide)
		}
		{
			rac := page.GetRowAndCell(1, 1)
			*rac.cell = pl2WideCell(0, PageCellWideSpacerTail)
		}
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(3), Reflow: true}))
	expectEqual(t, 3, s.cols)
	expectEqual(t, 3, s.totalRows())

	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		{
			rac := page.GetRowAndCell(0, 0)
			expectEqual[uint32](t, 'x', rac.cell.content().codepoint())
			expectEqual(t, PageCellWideNarrow, rac.cell.Wide())
			expect(t, rac.row.wrap, "rac.row.wrap")
		}
		{
			rac := page.GetRowAndCell(1, 0)
			expectEqual[uint32](t, 'x', rac.cell.content().codepoint())
			expectEqual(t, PageCellWideNarrow, rac.cell.Wide())
		}
		{
			rac := page.GetRowAndCell(2, 0)
			expectEqual[uint32](t, 'x', rac.cell.content().codepoint())
			expectEqual(t, PageCellWideNarrow, rac.cell.Wide())
		}
		{
			rac := page.GetRowAndCell(0, 1)
			expectEqual[uint32](t, '😀', rac.cell.content().codepoint())
			expectEqual(t, PageCellWideWide, rac.cell.Wide())
		}
		{
			rac := page.GetRowAndCell(1, 1)
			expectEqual[uint32](t, 0, rac.cell.content().codepoint())
			expectEqual(t, PageCellWideSpacerTail, rac.cell.Wide())
		}
	}
}

// Zig: test "PageList resize reflow less cols cursor goes to scrollback"
func TestPl2_PageList_resize_reflow_less_cols_cursor_goes_to_scrollback(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 4, 2, None[uint64]())
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	for y := uint(0); y < uint(s.rows); y++ {
		for x := uint(0); x < uint(s.cols); x++ {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}

	// Put a tracked pin in the history
	p := pl2Track(t, s, pl2Pin(t, s, pl2Active(2, 0)))
	defer s.UntrackPin(p)

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(2), Reflow: true}))
	expectEqual(t, 2, s.cols)
	expectEqual(t, 4, s.totalRows())

	// Our cursor should move to the first row
	expect(t, !s.PointFromPin(PointTagActive, *p).Valid, "s.pointFromPin(.active, p.*) == null")
}

// Zig: test "PageList resize reflow less cols cursor in unchanged row"
func TestPl2_PageList_resize_reflow_less_cols_cursor_in_unchanged_row(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 4, 2, None[uint64]())
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	for y := uint(0); y < uint(s.rows); y++ {
		for x := uint(0); x < 2; x++ {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}

	// Put a tracked pin in the history
	p := pl2Track(t, s, pl2Pin(t, s, pl2Active(1, 0)))
	defer s.UntrackPin(p)

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(2), Reflow: true}))
	expectEqual(t, 2, s.cols)
	expectEqual(t, 2, s.totalRows())

	// Our cursor should move to the first row
	expectEqualPoint(t, pl2Active(1, 0), zigUnwrap(t, s.PointFromPin(PointTagActive, *p)))
}

// Zig: test "PageList resize reflow less cols cursor in blank cell"
func TestPl2_PageList_resize_reflow_less_cols_cursor_in_blank_cell(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 6, 2, None[uint64]())
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	for y := uint(0); y < uint(s.rows); y++ {
		for x := uint(0); x < 2; x++ {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}

	// Put a tracked pin in the history
	p := pl2Track(t, s, pl2Pin(t, s, pl2Active(2, 0)))
	defer s.UntrackPin(p)

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(4), Reflow: true}))
	expectEqual(t, 4, s.cols)
	expectEqual(t, 2, s.totalRows())

	// Our cursor should not move
	expectEqualPoint(t, pl2Active(2, 0), zigUnwrap(t, s.PointFromPin(PointTagActive, *p)))
}

// Zig: test "PageList resize reflow less cols cursor in final blank cell"
func TestPl2_PageList_resize_reflow_less_cols_cursor_in_final_blank_cell(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 6, 2, None[uint64]())
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	for y := uint(0); y < uint(s.rows); y++ {
		for x := uint(0); x < 2; x++ {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}

	// Put a tracked pin in the history
	p := pl2Track(t, s, pl2Pin(t, s, pl2Active(3, 0)))
	defer s.UntrackPin(p)

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(4), Reflow: true}))
	expectEqual(t, 4, s.cols)
	expectEqual(t, 2, s.totalRows())

	// Our cursor should move to the first row
	expectEqualPoint(t, pl2Active(3, 0), zigUnwrap(t, s.PointFromPin(PointTagActive, *p)))
}

// Zig: test "PageList resize reflow less cols cursor in wrapped blank cell"
func TestPl2_PageList_resize_reflow_less_cols_cursor_in_wrapped_blank_cell(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 6, 2, None[uint64]())
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	for y := uint(0); y < uint(s.rows); y++ {
		for x := uint(0); x < 2; x++ {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}

	// Put a tracked pin in the history
	p := pl2Track(t, s, pl2Pin(t, s, pl2Active(5, 0)))
	defer s.UntrackPin(p)

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(4), Reflow: true}))
	expectEqual(t, 4, s.cols)
	expectEqual(t, 2, s.totalRows())

	// Our cursor should move to the first row
	expectEqualPoint(t, pl2Active(3, 0), zigUnwrap(t, s.PointFromPin(PointTagActive, *p)))
}

// Zig: test "PageList resize reflow less cols blank lines"
func TestPl2_PageList_resize_reflow_less_cols_blank_lines(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 4, 3, pl2Max(0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	for y := uint(0); y < 1; y++ {
		for x := uint(0); x < 4; x++ {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(2), Reflow: true}))
	expectEqual(t, 2, s.cols)
	expectEqual(t, 3, s.totalRows())

	it := s.RowIterator(PageListDirectionRightDown, pl2Active(0, 0), None[Point]())
	{
		// First row should be wrapped
		offset := zigUnwrap(t, it.Next())
		rac := offset.RowAndCell()
		cells := offset.Node.data.GetCells(rac.row)
		expect(t, rac.row.wrap, "rac.row.wrap")
		expectEqual(t, 2, len(cells))
		expectEqual[uint32](t, 0, cells[0].content().codepoint())
	}
	{
		offset := zigUnwrap(t, it.Next())
		rac := offset.RowAndCell()
		cells := offset.Node.data.GetCells(rac.row)
		expect(t, !rac.row.wrap, "!rac.row.wrap")
		expectEqual(t, 2, len(cells))
		expectEqual[uint32](t, 2, cells[0].content().codepoint())
	}
}

// Zig: test "PageList resize reflow less cols blank lines between"
func TestPl2_PageList_resize_reflow_less_cols_blank_lines_between(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 4, 3, pl2Max(0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	{
		for x := uint(0); x < 4; x++ {
			rac := page.GetRowAndCell(x, 0)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}
	{
		for x := uint(0); x < 4; x++ {
			rac := page.GetRowAndCell(x, 2)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(2), Reflow: true}))
	expectEqual(t, 2, s.cols)
	expectEqual(t, 5, s.totalRows())

	it := s.RowIterator(PageListDirectionRightDown, pl2Active(0, 0), None[Point]())
	{
		offset := zigUnwrap(t, it.Next())
		rac := offset.RowAndCell()
		expect(t, !rac.row.wrap, "!rac.row.wrap")
	}
	{
		offset := zigUnwrap(t, it.Next())
		rac := offset.RowAndCell()
		cells := offset.Node.data.GetCells(rac.row)
		expect(t, rac.row.wrap, "rac.row.wrap")
		expectEqual(t, 2, len(cells))
		expectEqual[uint32](t, 0, cells[0].content().codepoint())
	}
	{
		offset := zigUnwrap(t, it.Next())
		rac := offset.RowAndCell()
		cells := offset.Node.data.GetCells(rac.row)
		expect(t, !rac.row.wrap, "!rac.row.wrap")
		expectEqual(t, 2, len(cells))
		expectEqual[uint32](t, 2, cells[0].content().codepoint())
	}
}

// Zig: test "PageList resize reflow less cols blank lines between no scrollback"
func TestPl2_PageList_resize_reflow_less_cols_blank_lines_between_no_scrollback(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 5, 3, pl2Max(0))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	{
		rac := page.GetRowAndCell(0, 0)
		*rac.cell = pl2CpCell('A')
	}
	{
		rac := page.GetRowAndCell(0, 2)
		*rac.cell = pl2CpCell('C')
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(2), Reflow: true}))
	expectEqual(t, 2, s.cols)
	expectEqual(t, 3, s.totalRows())

	it := s.RowIterator(PageListDirectionRightDown, pl2Active(0, 0), None[Point]())
	{
		offset := zigUnwrap(t, it.Next())
		rac := offset.RowAndCell()
		cells := offset.Node.data.GetCells(rac.row)
		expect(t, !rac.row.wrap, "!rac.row.wrap")
		expectEqual(t, 2, len(cells))
		expectEqual[uint32](t, 'A', cells[0].content().codepoint())
	}
	{
		offset := zigUnwrap(t, it.Next())
		rac := offset.RowAndCell()
		cells := offset.Node.data.GetCells(rac.row)
		expectEqual[uint32](t, 0, cells[0].content().codepoint())
	}
	{
		offset := zigUnwrap(t, it.Next())
		rac := offset.RowAndCell()
		cells := offset.Node.data.GetCells(rac.row)
		expect(t, !rac.row.wrap, "!rac.row.wrap")
		expectEqual(t, 2, len(cells))
		expectEqual[uint32](t, 'C', cells[0].content().codepoint())
	}
}

// Zig: test "PageList resize reflow less cols cursor not on last line preserves location"
func TestPl2_PageList_resize_reflow_less_cols_cursor_not_on_last_line_preserves_location(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 5, 5, pl2Max(1))
	defer s.Deinit()
	expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
	page := &s.pages.first.data
	for y := uint(0); y < uint(s.rows); y++ {
		for x := uint(0); x < 2; x++ {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pl2CpCell(uint32(x))
		}
	}

	// Grow blank rows to push our rows back into scrollback
	zigTry(t, s.growRows(5))
	expectEqual(t, 10, s.totalRows())

	// Put a tracked pin in the history
	p := pl2Track(t, s, pl2Pin(t, s, pl2Active(0, 0)))
	defer s.UntrackPin(p)

	// Resize
	zigTry(t, s.Resize(PageListResize{
		Cols:   pl2C(4),
		Reflow: true,

		// Important: not on last row
		Cursor: Some(PageListResizeCursor{x: 1, y: 1}),
	}))
	expectEqual(t, 4, s.cols)
	expectEqual(t, 10, s.totalRows())

	// Our cursor should move to the first row
	expectEqualPoint(t, pl2Active(0, 0), zigUnwrap(t, s.PointFromPin(PointTagActive, *p)))
}

// Zig: test "PageList resize reflow less cols copy style"
func TestPl2_PageList_resize_reflow_less_cols_copy_style(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 4, 2, pl2Max(0))
	defer s.Deinit()
	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		// Create a style
		style := Style{Flags: StyleFlags{Bold: true}}
		style_id, err := page.styles.Add(style)
		zigTry(t, err)

		for x := uint(0); x < uint(s.cols-1); x++ {
			rac := page.GetRowAndCell(x, 0)
			*rac.cell = pageCellFields{
				content_tag: PageCellContentTagCodepoint,
				content:     pageCellContentFields{codepoint: uint32(x)}.pack(),
				style_id:    style_id,
			}.pack()
			page.styles.Use(style_id)
		}

		// We're over-counted by 1 because `add` implies `use`.
		page.styles.Release(style_id)
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(2), Reflow: true}))
	expectEqual(t, 2, s.cols)
	expectEqual(t, 2, s.totalRows())

	pl2RowIterActive(s, func(offset PageListPin) {
		for x := uint(0); x < uint(s.cols-1); x++ {
			offset_copy := offset
			offset_copy.X = SizeCellCountInt(x)
			rac := offset_copy.RowAndCell()
			style_id := rac.cell.StyleID()
			expect(t, style_id != 0, "style_id != 0")

			style := offset.Node.data.styles.Get(style_id)
			expect(t, style.Flags.Bold, "style.flags.bold")

			row := rac.row
			expect(t, row.styled, "row.styled")
		}
	})
}

// Zig: test "PageList resize reflow less cols to eliminate a wide char"
func TestPl2_PageList_resize_reflow_less_cols_to_eliminate_a_wide_char(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 2, 1, pl2Max(0))
	defer s.Deinit()
	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		{
			rac := page.GetRowAndCell(0, 0)
			*rac.cell = pl2WideCell('😀', PageCellWideWide)
		}
		{
			rac := page.GetRowAndCell(1, 0)
			*rac.cell = pl2WideCell(0, PageCellWideSpacerTail)
		}
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(1), Reflow: true}))
	expectEqual(t, 1, s.cols)
	expectEqual(t, 1, s.totalRows())

	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		{
			rac := page.GetRowAndCell(0, 0)
			expectEqual[uint32](t, 0, rac.cell.content().codepoint())
			expectEqual(t, PageCellWideNarrow, rac.cell.Wide())
		}
	}
}

// Zig: test "PageList resize reflow less cols to wrap a wide char"
func TestPl2_PageList_resize_reflow_less_cols_to_wrap_a_wide_char(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 3, 1, pl2Max(0))
	defer s.Deinit()
	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		{
			rac := page.GetRowAndCell(0, 0)
			*rac.cell = pl2CpCell('x')
		}
		{
			rac := page.GetRowAndCell(1, 0)
			*rac.cell = pl2WideCell('😀', PageCellWideWide)
		}
		{
			rac := page.GetRowAndCell(2, 0)
			*rac.cell = pl2WideCell(0, PageCellWideSpacerTail)
		}
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(2), Reflow: true}))
	expectEqual(t, 2, s.cols)
	expectEqual(t, 2, s.totalRows())

	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		{
			rac := page.GetRowAndCell(0, 0)
			expectEqual[uint32](t, 'x', rac.cell.content().codepoint())
			expectEqual(t, PageCellWideNarrow, rac.cell.Wide())
			expect(t, rac.row.wrap, "rac.row.wrap")
		}
		{
			rac := page.GetRowAndCell(1, 0)
			expectEqual[uint32](t, 0, rac.cell.content().codepoint())
			expectEqual(t, PageCellWideSpacerHead, rac.cell.Wide())
		}
		{
			rac := page.GetRowAndCell(0, 1)
			expectEqual[uint32](t, '😀', rac.cell.content().codepoint())
			expectEqual(t, PageCellWideWide, rac.cell.Wide())
		}
		{
			rac := page.GetRowAndCell(1, 1)
			expectEqual[uint32](t, 0, rac.cell.content().codepoint())
			expectEqual(t, PageCellWideSpacerTail, rac.cell.Wide())
		}
	}
}

// Zig: test "PageList resize reflow less cols to wrap a multi-codepoint grapheme with a spacer head"
func TestPl2_PageList_resize_reflow_less_cols_to_wrap_a_multi_codepoint_grapheme_with_a_spacer_head(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 4, 2, pl2Max(0))
	defer s.Deinit()
	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		// We want to make the screen look like this:
		//
		// 👨‍👨‍👦‍👦👨‍👨‍👦‍👦

		// First family emoji at (0, 0)
		{
			rac := page.GetRowAndCell(0, 0)
			*rac.cell = pl2WideCell(0x1F468, PageCellWideWide) // First codepoint of the grapheme
			zigTry(t, page.SetGraphemes(rac.row, rac.cell, []uint32{
				0x200D, 0x1F468,
				0x200D, 0x1F466,
				0x200D, 0x1F466,
			}))
		}
		{
			rac := page.GetRowAndCell(1, 0)
			*rac.cell = pl2WideCell(0, PageCellWideSpacerTail)
		}
		// Second family emoji at (2, 0)
		{
			rac := page.GetRowAndCell(2, 0)
			*rac.cell = pl2WideCell(0x1F468, PageCellWideWide) // First codepoint of the grapheme
			zigTry(t, page.SetGraphemes(rac.row, rac.cell, []uint32{
				0x200D, 0x1F468,
				0x200D, 0x1F466,
				0x200D, 0x1F466,
			}))
		}
		{
			rac := page.GetRowAndCell(3, 0)
			*rac.cell = pl2WideCell(0, PageCellWideSpacerTail)
		}
	}

	// Resize
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(3), Reflow: true}))
	expectEqual(t, 3, s.cols)
	expectEqual(t, 2, s.totalRows())

	{
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		page := &s.pages.first.data

		{
			rac := page.GetRowAndCell(0, 0)
			expectEqual[uint32](t, 0x1F468, rac.cell.content().codepoint())
			expectEqual(t, PageCellWideWide, rac.cell.Wide())

			cps := zigUnwrap(t, page.LookupGrapheme(rac.cell))
			expectEqual(t, 6, len(cps))
			expectEqual[uint32](t, 0x200D, cps[0])
			expectEqual[uint32](t, 0x1F468, cps[1])
			expectEqual[uint32](t, 0x200D, cps[2])
			expectEqual[uint32](t, 0x1F466, cps[3])
			expectEqual[uint32](t, 0x200D, cps[4])
			expectEqual[uint32](t, 0x1F466, cps[5])

			// Row should be wrapped
			expect(t, rac.row.wrap, "rac.row.wrap")
		}
		{
			rac := page.GetRowAndCell(1, 0)
			expectEqual[uint32](t, 0, rac.cell.content().codepoint())
			expectEqual(t, PageCellWideSpacerTail, rac.cell.Wide())
		}
		{
			rac := page.GetRowAndCell(2, 0)
			expectEqual[uint32](t, 0, rac.cell.content().codepoint())
			expectEqual(t, PageCellWideSpacerHead, rac.cell.Wide())
		}

		{
			rac := page.GetRowAndCell(0, 0)
			expectEqual[uint32](t, 0x1F468, rac.cell.content().codepoint())
			expectEqual(t, PageCellWideWide, rac.cell.Wide())

			cps := zigUnwrap(t, page.LookupGrapheme(rac.cell))
			expectEqual(t, 6, len(cps))
			expectEqual[uint32](t, 0x200D, cps[0])
			expectEqual[uint32](t, 0x1F468, cps[1])
			expectEqual[uint32](t, 0x200D, cps[2])
			expectEqual[uint32](t, 0x1F466, cps[3])
			expectEqual[uint32](t, 0x200D, cps[4])
			expectEqual[uint32](t, 0x1F466, cps[5])
		}
		{
			rac := page.GetRowAndCell(1, 1)
			expectEqual[uint32](t, 0, rac.cell.content().codepoint())
			expectEqual(t, PageCellWideSpacerTail, rac.cell.Wide())
		}
	}
}

// Zig: test "PageList reset"
func TestPl2_PageList_reset(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	s.Reset()
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	expect(t, s.pages.first != nil, "s.pages.first != null")
	expectEqual(t, uint(s.rows), s.totalRows())

	// Active area should be the top
	expectEqual(t, PageListPin{
		Node: s.pages.first,
		Y:    0,
		X:    0,
	}, s.GetTopLeft(PointTagActive))
}

// Zig: test "PageList reset across two pages"
func TestPl2_PageList_reset_across_two_pages(t *testing.T) {
	testStdCapacity(t)

	// Find a cap that makes it so that rows don't fit on one page.
	const rows = 100
	cap := func() PageCapacity {
		cap, err := PageStdCapacity.Adjust(PageCapacityAdjustment{cols: pl2C(50)})
		zigTry(t, err)
		for cap.rows >= rows {
			cap, err = PageStdCapacity.Adjust(PageCapacityAdjustment{
				cols: pl2C(cap.cols + 50),
			})
			zigTry(t, err)
		}

		return cap
	}()

	// Init
	s := pl2Init(t, cap.cols, rows, None[uint64]())
	defer s.Deinit()
	s.Reset()
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	expect(t, s.pages.first != nil, "s.pages.first != null")
	expectEqual(t, uint(s.rows), s.totalRows())
}

// Zig: test "PageList reset moves tracked pins and marks them as garbage"
func TestPl2_PageList_reset_moves_tracked_pins_and_marks_them_as_garbage(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Create a tracked pin into the active area
	p := pl2Track(t, s, pl2Pin(t, s, pl2Active(42, 12)))
	defer s.UntrackPin(p)

	s.Reset()

	// Our added pin should now be garbage
	expect(t, p.Garbage, "p.garbage")

	// Viewport pin should not be garbage because it makes sense.
	expect(t, !s.viewport_pin.Garbage, "!s.viewport_pin.garbage")
}

// Zig: test "PageList clears history"
func TestPl2_PageList_clears_history(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	zigTry(t, s.growRows(30))
	s.Reset()
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	expect(t, s.pages.first != nil, "s.pages.first != null")
	expectEqual(t, uint(s.rows), s.totalRows())

	// Active area should be the top
	expectEqual(t, PageListPin{
		Node: s.pages.first,
		Y:    0,
		X:    0,
	}, s.GetTopLeft(PointTagActive))
}

// Zig: test "PageList resize reflow grapheme map capacity exceeded"
func TestPl2_PageList_resize_reflow_grapheme_map_capacity_exceeded(t *testing.T) {
	testStdCapacity(t)

	// This test verifies that when reflowing content with many graphemes,
	// the grapheme map capacity is correctly increased when needed.
	s := pl2Init(t, 4, 10, pl2Max(0))
	defer s.Deinit()
	expectEqual(t, 1, s.TotalPages())

	// Get the grapheme capacity from the page. We need more than this many
	// graphemes in a single destination page to trigger capacity increase
	// during reflow. Since each source page can only hold this many graphemes,
	// we create two source pages with graphemes that will merge into one
	// destination page.
	grapheme_capacity := s.pages.first.data.GraphemeCapacity()
	// Use slightly more than half the capacity per page, so combined they
	// exceed the capacity of a single destination page.
	graphemes_per_page := grapheme_capacity/2 + grapheme_capacity/4

	// Grow to the capacity of the first page and add more rows
	// so that we have two pages total.
	{
		page := &s.pages.first.data
		page.PauseIntegrityChecks(true)
		for i := page.size.rows; i < page.capacity.rows; i++ {
			pl2Grow(t, s)
		}
		page.PauseIntegrityChecks(false)
		expectEqual(t, 1, s.TotalPages())
		zigTry(t, s.growRows(graphemes_per_page))
		expectEqual(t, 2, s.TotalPages())

		// We now have two pages.
		expect(t, s.pages.first != s.pages.last, "s.pages.first.? != s.pages.last.?")
		expectEqual(t, s.pages.last, s.pages.first.next)
	}

	// Add graphemes to both pages. We add graphemes to rows at the END of the
	// first page, and graphemes to rows at the START of the second page.
	// When reflowing to 2 columns, these rows will wrap and stay together
	// on the same destination page, requiring capacity increase.

	// Add graphemes to the end of the first page (last rows)
	{
		page := &s.pages.first.data
		start_row := uint(page.size.rows) - graphemes_per_page
		for i := uint(0); i < graphemes_per_page; i++ {
			y := start_row + i
			rac := page.GetRowAndCell(0, y)
			*rac.cell = pl2CpCell('A')
			zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 0x0301))
		}
	}

	// Add graphemes to the beginning of the second page
	{
		page := &s.pages.last.data
		count := min(graphemes_per_page, uint(page.size.rows))
		for y := uint(0); y < count; y++ {
			rac := page.GetRowAndCell(0, y)
			*rac.cell = pl2CpCell('B')
			zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 0x0302))
		}
	}

	// Resize to fewer columns to trigger reflow.
	// The graphemes from both pages will be copied to destination pages.
	// They will all end up in a contiguous region of the destination.
	// If the bug exists (hyperlink_bytes increased instead of grapheme_bytes),
	// this will fail with GraphemeMapOutOfMemory when we exceed capacity.
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(2), Reflow: true}))

	// Verify the resize succeeded
	expectEqual(t, 2, s.cols)
}

// Zig: test "PageList resize grow cols with unwrap fixes viewport pin"
func TestPl2_PageList_resize_grow_cols_with_unwrap_fixes_viewport_pin(t *testing.T) {
	testStdCapacity(t)

	// Regression test: after resize/reflow, the viewport pin can end up at a
	// position where pin.y + rows > total_rows, causing getBottomRight to panic.

	// The plan is to pin viewport in history, then grow columns to unwrap rows.
	// The unwrap reduces total_rows, but the tracked pin moves to a position
	// that no longer has enough rows below it for the viewport height.
	s := pl2Init(t, 2, 10, None[uint64]())
	defer s.Deinit()

	// Make sure we have some history, in this case we have 30 rows of history
	zigTry(t, s.growRows(30))
	expectEqual(t, 40, s.totalRows())

	// Fill all rows with wrapped content (pairs that unwrap when cols increase)
	it := s.PageIterator(PageListDirectionRightDown, pl2Screen(0, 0), None[Point]())
	for {
		o := it.Next()
		if !o.Valid {
			break
		}
		chunk := o.Value
		page := &chunk.node.data
		for y := uint(chunk.start); y < uint(chunk.end); y++ {
			rac := page.GetRowAndCell(0, y)
			if y%2 == 0 {
				rac.row.wrap = true
			} else {
				rac.row.wrap_continuation = true
			}
			for x := uint(0); x < uint(s.cols); x++ {
				*page.GetRowAndCell(x, y).cell = pl2CpCell('A')
			}
		}
	}

	// Pin viewport at row 28 (in history, 2 rows before active area at row 30).
	// After unwrap: row 28 -> row 14, total_rows 40 -> 20, active starts at 10.
	// Pin at 14 needs rows 14-23, but only 0-19 exist -> overflow.
	s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: pl2Pin(t, s, pl2Screen(0, 28))})
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expect(t, s.GetBottomRight(PointTagViewport).Valid, "s.getBottomRight(.viewport) != null")

	// Resize with reflow: unwraps rows, reducing total_rows
	zigTry(t, s.Resize(PageListResize{Cols: pl2C(4), Reflow: true}))
	expectEqual(t, 4, s.cols)
	expect(t, s.totalRows() < 40, "s.totalRows() < 40")

	// Used to panic here, so test that we can get the bottom right.
	br_after := s.GetBottomRight(PointTagViewport)
	expect(t, br_after.Valid, "br_after != null")
}

// Zig: test "PageList grow reuses non-standard page without leak"
func TestPl2_PageList_grow_reuses_non_standard_page_without_leak(t *testing.T) {
	testStdCapacity(t)

	// Create a PageList with 3 * std_size max so we can fit multiple pages
	// but will still trigger reuse.
	s := pl2Init(t, 80, 24, Some(3*pageListStdSize()))
	defer s.Deinit()

	// Increase the first page capacity to make it non-standard (larger than std_size).
	for pageListMemoryLen(&s.pages.first.data) <= pageListStdSize() {
		pl2IncreaseCapacity(t, s, s.pages.first, PageListIncreaseCapacityGraphemeBytes)
	}

	// The first page should now have non-standard memory size.
	expect(t, pageListMemoryLen(&s.pages.first.data) > pageListStdSize(), "s.pages.first.?.data.memory.len > std_size")

	// First, fill up the first page's capacity
	first_page := s.pages.first
	for first_page.data.size.rows < first_page.data.capacity.rows {
		pl2Grow(t, s)
	}

	// Now grow to create a second page
	pl2Grow(t, s)
	expect(t, s.pages.first != s.pages.last, "s.pages.first != s.pages.last")

	// Continue growing until we exceed max_size AND the last page is full
	for s.page_size+pageListPagePoolItemSize() <= s.MaxSize() ||
		s.pages.last.data.size.rows < s.pages.last.data.capacity.rows {
		pl2Grow(t, s)
	}

	// The first page should still be non-standard
	expect(t, pageListMemoryLen(&s.pages.first.data) > pageListStdSize(), "s.pages.first.?.data.memory.len > std_size")

	// Verify we have enough rows for active area (so prune path isn't skipped)
	expect(t, s.totalRows() >= uint(s.rows), "s.totalRows() >= s.rows")

	// Verify last page is full (so grow will need to allocate/reuse)
	expect(t, s.pages.last.data.size.rows == s.pages.last.data.capacity.rows, "s.pages.last.?.data.size.rows == s.pages.last.?.data.capacity.rows")

	// Remember the first page memory pointer before the reuse attempt
	first_page_ptr := s.pages.first
	// Zig-only: const first_page_mem_ptr = s.pages.first.?.data.memory.ptr; (raw pointer of the page buffer)

	// Create a tracked pin pointing to the non-standard first page
	tracked_pin := pl2Track(t, s, PageListPin{Node: first_page_ptr, X: 0, Y: 0})
	defer s.UntrackPin(tracked_pin)

	// Now grow one more time to trigger the reuse path. Since the first page
	// is non-standard, it should be destroyed (not reused). The testing
	// allocator will detect a leak if destroyNode doesn't properly free
	// the non-standard memory.
	pl2Grow(t, s)

	// After grow, check if the first page is a different one
	// (meaning the non-standard page was pruned, not reused at the end)
	// The original first page should no longer be the first page
	expect(t, s.pages.first != first_page_ptr, "s.pages.first.? != first_page_ptr")

	// If the non-standard page was properly destroyed and not reused,
	// the last page should not have the same memory pointer
	// Zig-only: try testing.expect(s.pages.last.?.data.memory.ptr != first_page_mem_ptr); (raw pointer of the page buffer)

	// The tracked pin should have been moved to the new first page and marked as garbage
	expectEqual(t, s.pages.first, tracked_pin.Node)
	expectEqual(t, 0, tracked_pin.X)
	expectEqual(t, 0, tracked_pin.Y)
	expect(t, tracked_pin.Garbage, "tracked_pin.garbage")
}

// Zig: test "PageList grow non-standard page prune protection"
func TestPl2_PageList_grow_non_standard_page_prune_protection(t *testing.T) {
	testStdCapacity(t)

	// This test specifically verifies the fix for the bug where pruning a
	// non-standard page would cause totalRows() < self.rows.
	//
	// Bug trigger conditions (all must be true simultaneously):
	// 1. first page is non-standard (memory.len > std_size)
	// 2. page_size + PagePool.item_size > maxSize() (triggers prune consideration)
	// 3. pages.first != pages.last (have multiple pages)
	// 4. total_rows >= self.rows (have enough rows for active area)
	// 5. total_rows - first.size.rows + 1 < self.rows (prune would lose too many)

	// This is kind of magic and likely depends on std_size.
	const rows_count = 600
	s := pl2Init(t, 80, rows_count, Some(pageListStdSize()))
	defer s.Deinit()

	// Make the first page non-standard
	for pageListMemoryLen(&s.pages.first.data) <= pageListStdSize() {
		pl2IncreaseCapacity(t, s, s.pages.first, PageListIncreaseCapacityGraphemeBytes)
	}
	expect(t, pageListMemoryLen(&s.pages.first.data) > pageListStdSize(), "s.pages.first.?.data.memory.len > std_size")

	first_page_node := s.pages.first
	first_page_cap := first_page_node.data.capacity.rows

	// Fill first page to capacity
	for first_page_node.data.size.rows < first_page_cap {
		pl2Grow(t, s)
	}

	// Grow until we have a second page (first page fills up first)
	var second_node *PageListNode
	for s.pages.first == s.pages.last {
		second_node = pl2Grow(t, s)
	}
	_ = second_node
	expect(t, s.pages.first != s.pages.last, "s.pages.first != s.pages.last")

	// Fill the second page to capacity so that the next grow() triggers prune
	last_node := s.pages.last
	second_cap := last_node.data.capacity.rows
	for last_node.data.size.rows < second_cap {
		pl2Grow(t, s)
	}

	// Now the last page is full. The next grow must either:
	// 1. Prune the first page and reuse it, OR
	// 2. Allocate a new page
	total := s.totalRows()
	would_remain := total - uint(first_page_cap) + 1

	// Verify the bug condition is present: pruning first page would leave < rows
	expect(t, would_remain < uint(s.rows), "would_remain < s.rows")

	// Verify prune path conditions are met
	expect(t, s.pages.first != s.pages.last, "s.pages.first != s.pages.last")
	expect(t, s.page_size+pageListPagePoolItemSize() > s.MaxSize(), "s.page_size + PagePool.item_size > s.maxSize()")
	expect(t, s.totalRows() >= uint(s.rows), "s.totalRows() >= s.rows")

	// Verify last page is at capacity (so grow must prune or allocate new)
	expectEqual(t, second_cap, last_node.data.size.rows)

	// The next grow should trigger prune consideration.
	// Without the fix, this would destroy the non-standard first page,
	// leaving only second_cap + 1 rows, which is < self.rows.
	pl2Grow(t, s)

	// Verify the invariant holds - the fix prevents the destructive prune
	expect(t, s.totalRows() >= uint(s.rows), "s.totalRows() >= s.rows")
}

// Zig: test "PageList resize (no reflow) more cols remaps pins in backfill path"
func TestPl2_PageList_resize_no_reflow_more_cols_remaps_pins_in_backfill_path(t *testing.T) {
	testStdCapacity(t)

	// Regression test: when resizeWithoutReflowGrowCols copies rows to a previous
	// page with spare capacity, tracked pins in those rows must be remapped.
	// Without the fix, pins become dangling pointers when the original page is destroyed.
	const cols SizeCellCountInt = 5
	cap, err := PageStdCapacity.Adjust(PageCapacityAdjustment{cols: pl2C(cols)})
	zigTry(t, err)
	s := pl2Init(t, cols, cap.rows, None[uint64]())
	defer s.Deinit()

	// Grow until we have two pages.
	for s.pages.first == s.pages.last {
		pl2Grow(t, s)
	}
	first_page := s.pages.first
	second_page := s.pages.last
	expect(t, first_page != second_page, "first_page != second_page")

	// Trim a history row so the first page has spare capacity.
	// This triggers the backfill path in resizeWithoutReflowGrowCols.
	s.EraseRows(pl2History(0, 0), Some(pl2History(0, 0)))
	expect(t, first_page.data.size.rows < first_page.data.capacity.rows, "first_page.data.size.rows < first_page.data.capacity.rows")

	// Ensure the resize takes the slow path (new capacity > current capacity).
	const new_cols SizeCellCountInt = cols + 1
	adjusted, err := second_page.data.capacity.Adjust(PageCapacityAdjustment{cols: pl2C(new_cols)})
	zigTry(t, err)
	expect(t, second_page.data.capacity.cols < adjusted.cols, "second_page.data.capacity.cols < adjusted.cols")

	// Track a pin in row 0 of the second page. This row will be copied
	// to the first page during backfill and the pin must be remapped.
	tracked := pl2Track(t, s, PageListPin{Node: second_page, X: 0, Y: 0})
	defer s.UntrackPin(tracked)

	// Write a marker character to the tracked cell so we can verify
	// the pin points to the correct cell after resize.
	const marker uint32 = 'X'
	*tracked.RowAndCell().cell = pl2CpCell(marker)

	zigTry(t, s.Resize(PageListResize{Cols: pl2C(new_cols), Reflow: false}))

	// Verify the pin points to a valid node still in the page list.
	found := false
	for node := s.pages.first; node != nil; node = node.next {
		if node == tracked.Node {
			found = true
			break
		}
	}
	expect(t, found, "found")
	expect(t, tracked.Y < tracked.Node.data.size.rows, "tracked.y < tracked.node.data.size.rows")

	// Verify the pin still points to the cell with our marker content.
	cell := tracked.RowAndCell().cell
	expectEqual(t, PageCellContentTagCodepoint, cell.ContentTag())
	expectEqual(t, marker, cell.content().codepoint())
}

// Zig: test "PageList compact std_size page returns null"
func TestPl2_PageList_compact_std_size_page_returns_null(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 80, 24, pl2Max(0))
	defer s.Deinit()

	// A freshly created page should be at std_size
	node := s.pages.first
	expect(t, pageListMemoryLen(&node.data) <= pageListStdSize(), "node.data.memory.len <= std_size")

	// compact should return null since there's nothing to compact
	result, err := s.Compact(node)
	zigTry(t, err)
	expectEqual[*PageListNode](t, nil, result)

	// Page should still be the same
	expectEqual(t, node, s.pages.first)
}

// Zig: test "PageList compact oversized page"
func TestPl2_PageList_compact_oversized_page(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Grow until we have multiple pages
	page1_node := s.pages.first
	page1_node.data.PauseIntegrityChecks(true)
	// Zig: `for (0..n)` evaluates n once, before the loop; grow changes it.
	n := uint(page1_node.data.capacity.rows - page1_node.data.size.rows)
	for i := uint(0); i < n; i++ {
		pl2Grow(t, s)
	}
	page1_node.data.PauseIntegrityChecks(false)
	pl2Grow(t, s)
	expect(t, s.pages.first != s.pages.last, "s.pages.first != s.pages.last")

	node := s.pages.first

	// Write content to verify it's preserved
	{
		page := &node.data
		for y := uint(0); y < uint(page.size.rows); y++ {
			for x := uint(0); x < uint(s.cols); x++ {
				rac := page.GetRowAndCell(x, y)
				*rac.cell = pl2CpCell(uint32(x + y*uint(s.cols)))
			}
		}
	}

	// Create a tracked pin on this page
	tracked := pl2Track(t, s, PageListPin{Node: node, X: 5, Y: 10})
	defer s.UntrackPin(tracked)

	// Make the page oversized
	for pageListMemoryLen(&node.data) <= pageListStdSize() {
		node = pl2IncreaseCapacity(t, s, node, PageListIncreaseCapacityGraphemeBytes)
	}
	expect(t, pageListMemoryLen(&node.data) > pageListStdSize(), "node.data.memory.len > std_size")
	oversized_len := pageListMemoryLen(&node.data)
	original_size := node.data.size
	second_node := node.next

	// Set dirty flag after increaseCapacity
	node.data.dirty = true

	// Compact the page
	new_node, err := s.Compact(node)
	zigTry(t, err)
	expect(t, new_node != nil, "new_node != null")

	// Verify memory is smaller
	expect(t, pageListMemoryLen(&new_node.data) < oversized_len, "new_node.?.data.memory.len < oversized_len")

	// Verify size preserved
	expectEqual(t, original_size.rows, new_node.data.size.rows)
	expectEqual(t, original_size.cols, new_node.data.size.cols)

	// Verify dirty flag preserved
	expect(t, new_node.data.dirty, "new_node.?.data.dirty")

	// Verify linked list integrity
	expectEqual(t, new_node, s.pages.first)
	expectEqual[*PageListNode](t, nil, new_node.prev)
	expectEqual(t, second_node, new_node.next)
	expectEqual(t, new_node, second_node.prev)

	// Verify pin updated correctly
	expectEqual(t, new_node, tracked.Node)
	expectEqual[SizeCellCountInt](t, 5, tracked.X)
	expectEqual[SizeCellCountInt](t, 10, tracked.Y)

	// Verify content preserved
	page := &new_node.data
	for y := uint(0); y < uint(page.size.rows); y++ {
		for x := uint(0); x < uint(s.cols); x++ {
			rac := page.GetRowAndCell(x, y)
			expectEqual(t,
				uint32(x+y*uint(s.cols)),
				rac.cell.content().codepoint(),
			)
		}
	}
}

// Zig: test "PageList compact insufficient savings returns null"
func TestPl2_PageList_compact_insufficient_savings_returns_null(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 80, 24, pl2Max(0))
	defer s.Deinit()

	node := s.pages.first

	// Make the page slightly oversized (just one increase)
	// This might not provide enough savings to justify compaction
	node = pl2IncreaseCapacity(t, s, node, PageListIncreaseCapacityGraphemeBytes)

	// If the page is still at or below std_size, compact returns null
	if pageListMemoryLen(&node.data) <= pageListStdSize() {
		result, err := s.Compact(node)
		zigTry(t, err)
		expectEqual[*PageListNode](t, nil, result)
	} else {
		// If it did grow beyond std_size, verify that compaction
		// works or returns null based on savings calculation
		result, err := s.Compact(node)
		zigTry(t, err)
		// Either it compacted or determined insufficient savings
		if new_node := result; new_node != nil {
			expect(t, pageListMemoryLen(&new_node.data) < pageListMemoryLen(&node.data), "new_node.data.memory.len < node.data.memory.len")
		}
	}
}

// Zig: test "PageList split at middle row"
func TestPl2_PageList_split_at_middle_row(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 10, 10, pl2Max(0))
	defer s.Deinit()

	page := &s.pages.first.data

	// Write content to rows: row 0 gets codepoint 0, row 1 gets 1, etc.
	for y := uint(0); y < uint(page.size.rows); y++ {
		rac := page.GetRowAndCell(0, y)
		*rac.cell = pl2CpCell(uint32(y))
	}

	// Split at row 5 (middle)
	split_pin := PageListPin{Node: s.pages.first, Y: 5, X: 0}
	zigTry(t, s.Split(split_pin))

	// Verify two pages exist
	expect(t, s.pages.first != nil, "s.pages.first != null")
	expect(t, s.pages.first.next != nil, "s.pages.first.?.next != null")

	first_page := &s.pages.first.data
	second_page := &s.pages.first.next.data

	// First page should have rows 0-4 (5 rows)
	expectEqual(t, 5, first_page.size.rows)
	// Second page should have rows 5-9 (5 rows)
	expectEqual(t, 5, second_page.size.rows)

	// Verify content in first page is preserved (rows 0-4 have codepoints 0-4)
	for y := uint(0); y < 5; y++ {
		rac := first_page.GetRowAndCell(0, y)
		expectEqual(t, uint32(y), rac.cell.content().codepoint())
	}

	// Verify content in second page (original rows 5-9, now at y=0-4)
	for y := uint(0); y < 5; y++ {
		rac := second_page.GetRowAndCell(0, y)
		expectEqual(t, uint32(y+5), rac.cell.content().codepoint())
	}
}

// Zig: test "PageList split at row 0 is no-op"
func TestPl2_PageList_split_at_row_0_is_no_op(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 10, 10, pl2Max(0))
	defer s.Deinit()

	page := &s.pages.first.data

	// Write content to all rows
	for y := uint(0); y < uint(page.size.rows); y++ {
		rac := page.GetRowAndCell(0, y)
		*rac.cell = pl2CpCell(uint32(y))
	}

	// Split at row 0 should be a no-op
	split_pin := PageListPin{Node: s.pages.first, Y: 0, X: 0}
	zigTry(t, s.Split(split_pin))

	// Verify only one page exists (no split occurred)
	expect(t, s.pages.first != nil, "s.pages.first != null")
	expect(t, s.pages.first.next == nil, "s.pages.first.?.next == null")

	// Verify all content is still in the original page
	expectEqual(t, 10, page.size.rows)
	for y := uint(0); y < 10; y++ {
		rac := page.GetRowAndCell(0, y)
		expectEqual(t, uint32(y), rac.cell.content().codepoint())
	}
}

// Zig: test "PageList split at last row"
func TestPl2_PageList_split_at_last_row(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 10, 10, pl2Max(0))
	defer s.Deinit()

	page := &s.pages.first.data

	// Write content to all rows
	for y := uint(0); y < uint(page.size.rows); y++ {
		rac := page.GetRowAndCell(0, y)
		*rac.cell = pl2CpCell(uint32(y))
	}

	// Split at last row (row 9)
	split_pin := PageListPin{Node: s.pages.first, Y: 9, X: 0}
	zigTry(t, s.Split(split_pin))

	// Verify two pages exist
	expect(t, s.pages.first != nil, "s.pages.first != null")
	expect(t, s.pages.first.next != nil, "s.pages.first.?.next != null")

	first_page := &s.pages.first.data
	second_page := &s.pages.first.next.data

	// First page should have 9 rows
	expectEqual(t, 9, first_page.size.rows)
	// Second page should have 1 row
	expectEqual(t, 1, second_page.size.rows)

	// Verify content in second page (original row 9, now at y=0)
	rac := second_page.GetRowAndCell(0, 0)
	expectEqual[uint32](t, 9, rac.cell.content().codepoint())
}

// Zig: test "PageList split single row page returns OutOfSpace"
func TestPl2_PageList_split_single_row_page_returns_OutOfSpace(t *testing.T) {
	testStdCapacity(t)

	// Initialize with 1 row
	s := pl2Init(t, 10, 1, pl2Max(0))
	defer s.Deinit()

	split_pin := PageListPin{Node: s.pages.first, Y: 0, X: 0}
	result := s.Split(split_pin)

	pl2ExpectError(t, ErrOutOfSpace, result)
}

// Zig: test "PageList split moves tracked pins"
func TestPl2_PageList_split_moves_tracked_pins(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 10, 10, pl2Max(0))
	defer s.Deinit()

	// Track a pin at row 7
	tracked := pl2Track(t, s, PageListPin{Node: s.pages.first, Y: 7, X: 3})
	defer s.UntrackPin(tracked)

	// Split at row 5
	split_pin := PageListPin{Node: s.pages.first, Y: 5, X: 0}
	zigTry(t, s.Split(split_pin))

	// The tracked pin should now be in the second page
	expect(t, tracked.Node == s.pages.first.next, "tracked.node == s.pages.first.?.next.?")
	// y should be adjusted: was 7, split at 5, so new y = 7 - 5 = 2
	expectEqual(t, 2, tracked.Y)
	// x should remain unchanged
	expectEqual(t, 3, tracked.X)
}

// Zig: test "PageList split tracked pin before split point unchanged"
func TestPl2_PageList_split_tracked_pin_before_split_point_unchanged(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 10, 10, pl2Max(0))
	defer s.Deinit()

	original_node := s.pages.first

	// Track a pin at row 2 (before the split point)
	tracked := pl2Track(t, s, PageListPin{Node: original_node, Y: 2, X: 5})
	defer s.UntrackPin(tracked)

	// Split at row 5
	split_pin := PageListPin{Node: original_node, Y: 5, X: 0}
	zigTry(t, s.Split(split_pin))

	// The tracked pin should remain in the original page
	expect(t, tracked.Node == s.pages.first, "tracked.node == s.pages.first.?")
	// y and x should be unchanged
	expectEqual(t, 2, tracked.Y)
	expectEqual(t, 5, tracked.X)
}

// Zig: test "PageList split tracked pin at split point moves to new page"
func TestPl2_PageList_split_tracked_pin_at_split_point_moves_to_new_page(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 10, 10, pl2Max(0))
	defer s.Deinit()

	original_node := s.pages.first

	// Track a pin at the exact split point (row 5)
	tracked := pl2Track(t, s, PageListPin{Node: original_node, Y: 5, X: 4})
	defer s.UntrackPin(tracked)

	// Split at row 5
	split_pin := PageListPin{Node: original_node, Y: 5, X: 0}
	zigTry(t, s.Split(split_pin))

	// The tracked pin should be in the new page
	expect(t, tracked.Node == s.pages.first.next, "tracked.node == s.pages.first.?.next.?")
	// y should be 0 since it was at the split point: 5 - 5 = 0
	expectEqual(t, 0, tracked.Y)
	// x should remain unchanged
	expectEqual(t, 4, tracked.X)
}

// Zig: test "PageList split multiple tracked pins across regions"
func TestPl2_PageList_split_multiple_tracked_pins_across_regions(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 10, 10, pl2Max(0))
	defer s.Deinit()

	original_node := s.pages.first

	// Track multiple pins in different regions
	pin_before := pl2Track(t, s, PageListPin{Node: original_node, Y: 1, X: 0})
	defer s.UntrackPin(pin_before)
	pin_at_split := pl2Track(t, s, PageListPin{Node: original_node, Y: 5, X: 2})
	defer s.UntrackPin(pin_at_split)
	pin_after1 := pl2Track(t, s, PageListPin{Node: original_node, Y: 7, X: 3})
	defer s.UntrackPin(pin_after1)
	pin_after2 := pl2Track(t, s, PageListPin{Node: original_node, Y: 9, X: 8})
	defer s.UntrackPin(pin_after2)

	// Split at row 5
	split_pin := PageListPin{Node: original_node, Y: 5, X: 0}
	zigTry(t, s.Split(split_pin))

	first_page := s.pages.first
	second_page := first_page.next

	// Pin before split point stays in original page
	expect(t, pin_before.Node == first_page, "pin_before.node == first_page")
	expectEqual(t, 1, pin_before.Y)
	expectEqual(t, 0, pin_before.X)

	// Pin at split point moves to new page with y=0
	expect(t, pin_at_split.Node == second_page, "pin_at_split.node == second_page")
	expectEqual(t, 0, pin_at_split.Y)
	expectEqual(t, 2, pin_at_split.X)

	// Pins after split point move to new page with adjusted y
	expect(t, pin_after1.Node == second_page, "pin_after1.node == second_page")
	expectEqual(t, 2, pin_after1.Y) // 7 - 5 = 2
	expectEqual(t, 3, pin_after1.X)

	expect(t, pin_after2.Node == second_page, "pin_after2.node == second_page")
	expectEqual(t, 4, pin_after2.Y) // 9 - 5 = 4
	expectEqual(t, 8, pin_after2.X)
}

// Zig: test "PageList split tracked viewport_pin in split region moves correctly"
func TestPl2_PageList_split_tracked_viewport_pin_in_split_region_moves_correctly(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 10, 10, pl2Max(0))
	defer s.Deinit()

	original_node := s.pages.first

	// Set viewport_pin to row 7 (after split point)
	s.viewport_pin.Node = original_node
	s.viewport_pin.Y = 7
	s.viewport_pin.X = 6

	// Split at row 5
	split_pin := PageListPin{Node: original_node, Y: 5, X: 0}
	zigTry(t, s.Split(split_pin))

	// viewport_pin should be in the new page
	expect(t, s.viewport_pin.Node == s.pages.first.next, "s.viewport_pin.node == s.pages.first.?.next.?")
	// y should be adjusted: 7 - 5 = 2
	expectEqual(t, 2, s.viewport_pin.Y)
	// x should remain unchanged
	expectEqual(t, 6, s.viewport_pin.X)
}

// Zig: test "PageList split middle page preserves linked list order"
func TestPl2_PageList_split_middle_page_preserves_linked_list_order(t *testing.T) {
	testStdCapacity(t)

	// Create a single page with 12 rows
	s := pl2Init(t, 10, 12, pl2Max(0))
	defer s.Deinit()

	// Split at row 4 to create: page1 (rows 0-3), page2 (rows 4-11)
	first_node := s.pages.first
	split_pin1 := PageListPin{Node: first_node, Y: 4, X: 0}
	zigTry(t, s.Split(split_pin1))

	// Now we have 2 pages
	page1 := s.pages.first
	page2 := s.pages.first.next
	expectEqual(t, 4, page1.data.size.rows)
	expectEqual(t, 8, page2.data.size.rows)

	// Split page2 at row 4 to create: page1 -> page2 (rows 0-3) -> page3 (rows 4-7)
	split_pin2 := PageListPin{Node: page2, Y: 4, X: 0}
	zigTry(t, s.Split(split_pin2))

	// Now we have 3 pages
	first := s.pages.first
	middle := first.next
	last := middle.next

	// Verify linked list order: first -> middle -> last
	expectEqual(t, page1, first)
	expectEqual(t, page2, middle)
	expectEqual(t, s.pages.last, last)

	// Verify prev pointers
	expect(t, first.prev == nil, "first.prev == null")
	expectEqual(t, first, middle.prev)
	expectEqual(t, middle, last.prev)

	// Verify next pointers
	expectEqual(t, middle, first.next)
	expectEqual(t, last, middle.next)
	expect(t, last.next == nil, "last.next == null")

	// Verify row counts
	expectEqual(t, 4, first.data.size.rows)
	expectEqual(t, 4, middle.data.size.rows)
	expectEqual(t, 4, last.data.size.rows)
}

// Zig: test "PageList split last page makes new page the last"
func TestPl2_PageList_split_last_page_makes_new_page_the_last(t *testing.T) {
	testStdCapacity(t)

	// Create a single page with 10 rows
	s := pl2Init(t, 10, 10, pl2Max(0))
	defer s.Deinit()

	// Split to create 2 pages first
	first_node := s.pages.first
	split_pin1 := PageListPin{Node: first_node, Y: 5, X: 0}
	zigTry(t, s.Split(split_pin1))

	// Now split the last page
	last_before_split := s.pages.last
	expectEqual(t, 5, last_before_split.data.size.rows)

	split_pin2 := PageListPin{Node: last_before_split, Y: 2, X: 0}
	zigTry(t, s.Split(split_pin2))

	// The new page should be the new last
	new_last := s.pages.last
	expect(t, new_last != last_before_split, "new_last != last_before_split")
	expectEqual(t, last_before_split, new_last.prev)
	expect(t, new_last.next == nil, "new_last.next == null")

	// Verify row counts: original last has 2 rows, new last has 3 rows
	expectEqual(t, 2, last_before_split.data.size.rows)
	expectEqual(t, 3, new_last.data.size.rows)
}

// Zig: test "PageList split first page keeps original as first"
func TestPl2_PageList_split_first_page_keeps_original_as_first(t *testing.T) {
	testStdCapacity(t)

	// Create 2 pages by splitting
	s := pl2Init(t, 10, 10, pl2Max(0))
	defer s.Deinit()

	original_first := s.pages.first
	split_pin1 := PageListPin{Node: original_first, Y: 5, X: 0}
	zigTry(t, s.Split(split_pin1))

	// Get second page (created by first split)
	second_page := s.pages.first.next

	// Now split the first page again
	split_pin2 := PageListPin{Node: s.pages.first, Y: 2, X: 0}
	zigTry(t, s.Split(split_pin2))

	// Original first should still be first
	expectEqual(t, original_first, s.pages.first)
	expect(t, s.pages.first.prev == nil, "s.pages.first.?.prev == null")

	// New page should be inserted between first and second
	inserted := s.pages.first.next
	expect(t, inserted != second_page, "inserted != second_page")
	expectEqual(t, second_page, inserted.next)

	// Verify row counts: first has 2, inserted has 3, second has 5
	expectEqual(t, 2, s.pages.first.data.size.rows)
	expectEqual(t, 3, inserted.data.size.rows)
	expectEqual(t, 5, second_page.data.size.rows)
}

// Zig: test "PageList split preserves wrap flags"
func TestPl2_PageList_split_preserves_wrap_flags(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 10, 10, pl2Max(0))
	defer s.Deinit()

	page := &s.pages.first.data

	// Set wrap flags on rows that will be in the second page after split
	// Row 5: wrap = true (this is the start of a wrapped line)
	// Row 6: wrap_continuation = true (this continues the wrap)
	// Row 7: wrap = true, wrap_continuation = true (wrapped and continues)
	{
		rac5 := page.GetRowAndCell(0, 5)
		rac5.row.wrap = true

		rac6 := page.GetRowAndCell(0, 6)
		rac6.row.wrap_continuation = true

		rac7 := page.GetRowAndCell(0, 7)
		rac7.row.wrap = true
		rac7.row.wrap_continuation = true
	}

	// Split at row 5
	split_pin := PageListPin{Node: s.pages.first, Y: 5, X: 0}
	zigTry(t, s.Split(split_pin))

	second_page := &s.pages.first.next.data

	// Verify wrap flags are preserved in new page
	// Original row 5 is now row 0 in second page
	{
		rac0 := second_page.GetRowAndCell(0, 0)
		expect(t, rac0.row.wrap, "rac0.row.wrap")
		expect(t, !rac0.row.wrap_continuation, "!rac0.row.wrap_continuation")
	}

	// Original row 6 is now row 1 in second page
	{
		rac1 := second_page.GetRowAndCell(0, 1)
		expect(t, !rac1.row.wrap, "!rac1.row.wrap")
		expect(t, rac1.row.wrap_continuation, "rac1.row.wrap_continuation")
	}

	// Original row 7 is now row 2 in second page
	{
		rac2 := second_page.GetRowAndCell(0, 2)
		expect(t, rac2.row.wrap, "rac2.row.wrap")
		expect(t, rac2.row.wrap_continuation, "rac2.row.wrap_continuation")
	}
}

// Zig: test "PageList split preserves styled cells"
func TestPl2_PageList_split_preserves_styled_cells(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 10, 10, pl2Max(0))
	defer s.Deinit()

	page := &s.pages.first.data

	// Create a style and apply it to cells in rows 5-7 (which will be in the second page)
	style := Style{Flags: StyleFlags{Bold: true}}
	style_id, err := page.styles.Add(style)
	zigTry(t, err)

	for y := uint(5); y < 8; y++ {
		rac := page.GetRowAndCell(0, y)
		*rac.cell = pageCellFields{
			content_tag: PageCellContentTagCodepoint,
			content:     pageCellContentFields{codepoint: 'S'}.pack(),
			style_id:    style_id,
		}.pack()
		rac.row.styled = true
		page.styles.Use(style_id)
	}
	// Release the extra ref from add
	page.styles.Release(style_id)

	// Split at row 5
	split_pin := PageListPin{Node: s.pages.first, Y: 5, X: 0}
	zigTry(t, s.Split(split_pin))

	first_page := &s.pages.first.data
	second_page := &s.pages.first.next.data

	// First page should have no styles (all styled rows moved to second page)
	expectEqual(t, 0, first_page.styles.Count())

	// Second page should have exactly 1 style (the bold style, used by 3 cells)
	expectEqual(t, 1, second_page.styles.Count())

	// Verify styled cells are preserved in new page
	for y := uint(0); y < 3; y++ {
		rac := second_page.GetRowAndCell(0, y)
		expectEqual[uint32](t, 'S', rac.cell.content().codepoint())
		expect(t, rac.cell.StyleID() != 0, "rac.cell.style_id != 0")

		got_style := second_page.styles.Get(rac.cell.StyleID())
		expect(t, got_style.Flags.Bold, "got_style.flags.bold")
		expect(t, rac.row.styled, "rac.row.styled")
	}
}

// Zig: test "PageList split preserves grapheme clusters"
func TestPl2_PageList_split_preserves_grapheme_clusters(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 10, 10, pl2Max(0))
	defer s.Deinit()

	page := &s.pages.first.data

	// Add a grapheme cluster to row 6 (will be row 1 in second page after split at 5)
	{
		rac := page.GetRowAndCell(0, 6)
		*rac.cell = pl2CpCell(0x1F468) // Man emoji
		zigTry(t, page.SetGraphemes(rac.row, rac.cell, []uint32{
			0x200D,  // ZWJ
			0x1F469, // Woman emoji
		}))
	}

	// Split at row 5
	split_pin := PageListPin{Node: s.pages.first, Y: 5, X: 0}
	zigTry(t, s.Split(split_pin))

	first_page := &s.pages.first.data
	second_page := &s.pages.first.next.data

	// First page should have no graphemes (the grapheme row moved to second page)
	expectEqual(t, 0, first_page.GraphemeCount())

	// Second page should have exactly 1 grapheme
	expectEqual(t, 1, second_page.GraphemeCount())

	// Verify grapheme is preserved in new page (original row 6 is now row 1)
	{
		rac := second_page.GetRowAndCell(0, 1)
		expectEqual[uint32](t, 0x1F468, rac.cell.content().codepoint())
		expect(t, rac.row.grapheme, "rac.row.grapheme")

		cps := zigUnwrap(t, second_page.LookupGrapheme(rac.cell))
		expectEqual(t, 2, len(cps))
		expectEqual[uint32](t, 0x200D, cps[0])
		expectEqual[uint32](t, 0x1F469, cps[1])
	}
}

// Zig: test "PageList split preserves hyperlinks"
func TestPl2_PageList_split_preserves_hyperlinks(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 10, 10, pl2Max(0))
	defer s.Deinit()

	page := &s.pages.first.data

	// Add a hyperlink to row 7 (will be row 2 in second page after split at 5)
	hyperlink_id, err := page.InsertHyperlink(Hyperlink{
		ID:  pl2Implicit(0),
		URI: []uint8("https://example.com"),
	})
	zigTry(t, err)
	{
		rac := page.GetRowAndCell(0, 7)
		*rac.cell = pl2CpCell('L')
		zigTry(t, page.SetHyperlink(rac.row, rac.cell, hyperlink_id))
	}

	// Split at row 5
	split_pin := PageListPin{Node: s.pages.first, Y: 5, X: 0}
	zigTry(t, s.Split(split_pin))

	first_page := &s.pages.first.data
	second_page := &s.pages.first.next.data

	// First page should have no hyperlinks (the hyperlink row moved to second page)
	expectEqual(t, 0, first_page.hyperlink_set.Count())

	// Second page should have exactly 1 hyperlink
	expectEqual(t, 1, second_page.hyperlink_set.Count())

	// Verify hyperlink is preserved in new page (original row 7 is now row 2)
	{
		rac := second_page.GetRowAndCell(0, 2)
		expectEqual[uint32](t, 'L', rac.cell.content().codepoint())
		expect(t, rac.cell.Hyperlink(), "rac.cell.hyperlink")

		link_id := zigUnwrap(t, second_page.LookupHyperlink(rac.cell))
		link := second_page.hyperlink_set.Get(link_id)
		expectEqualStrings(t, "https://example.com", string(link.uri))
	}
}

// Zig: test "term51: initialCapacity: standard range keeps total size"
func TestPl2_term51_initialCapacity_standard_range_keeps_total_size(t *testing.T) {
	testStdCapacity(t)

	// Every column count that fits is adjusted to the same total size so
	// pages can come from the pool.
	for _, cols := range []SizeCellCountInt{0, 1, 2, 80, 215, 216, 500, 1000, pl2StdMaxCols(t)} {
		cap := pageListInitialCapacity(cols)
		expectEqual(t, cols, cap.cols)
		expect(t, cap.rows >= 1, "cap.rows >= 1")
		expect(t, PageLayoutOf(cap).total_size <= pageListStdSize(), "Page.layout(cap).total_size <= std_size")

		// Everything but cols/rows is the standard capacity.
		expectEqual(t, PageStdCapacity.styles, cap.styles)
		expectEqual(t, PageStdCapacity.hyperlink_bytes, cap.hyperlink_bytes)
		expectEqual(t, PageStdCapacity.grapheme_bytes, cap.grapheme_bytes)
		expectEqual(t, PageStdCapacity.string_bytes, cap.string_bytes)

		// It is exactly what adjust returns.
		adj, err := PageStdCapacity.Adjust(PageCapacityAdjustment{cols: pl2C(cols)})
		zigTry(t, err)
		expectEqual(t, adj.rows, cap.rows)
	}
}

// Zig: test "term51: initialCapacity: rows never increase with cols"
func TestPl2_term51_initialCapacity_rows_never_increase_with_cols(t *testing.T) {
	testStdCapacity(t)

	var last SizeCellCountInt = math.MaxUint16
	for cols := uint(0); cols <= uint(pl2StdMaxCols(t)); cols += 997 {
		cap := pageListInitialCapacity(SizeCellCountInt(cols))
		expect(t, cap.rows <= last, "cap.rows <= last")
		last = cap.rows
	}
}

// Zig: test "term51: initialCapacity: boundary at maxCols fits one row"
func TestPl2_term51_initialCapacity_boundary_at_maxCols_fits_one_row(t *testing.T) {
	testStdCapacity(t)

	max := pl2StdMaxCols(t)

	// At the largest column count that fits we get exactly one row and
	// still the standard size.
	at := pageListInitialCapacity(max)
	expectEqual(t, max, at.cols)
	expectEqual[SizeCellCountInt](t, 1, at.rows)
	expect(t, PageLayoutOf(at).total_size <= pageListStdSize(), "Page.layout(at).total_size <= std_size")

	// One column fewer cannot have fewer rows than one.
	expect(t, pageListInitialCapacity(max-1).rows >= 1, "initialCapacity(max - 1).rows >= 1")

	// One past the limit goes non-standard: the standard capacity with
	// only the column count replaced (rows stay at the standard rows).
	past := pageListInitialCapacity(max + 1)
	expectEqual(t, max+1, past.cols)
	expectEqual(t, PageStdCapacity.rows, past.rows)
	expectEqual(t, PageStdCapacity.styles, past.styles)
	expectEqual(t, PageStdCapacity.hyperlink_bytes, past.hyperlink_bytes)
	expectEqual(t, PageStdCapacity.grapheme_bytes, past.grapheme_bytes)
	expectEqual(t, PageStdCapacity.string_bytes, past.string_bytes)
	expect(t, PageLayoutOf(past).total_size > pageListStdSize(), "Page.layout(past).total_size > std_size")
	expect(t, PageLayoutOf(past).total_size <= SizeMaxPageSize, "Page.layout(past).total_size <= size.max_page_size")
}

// Zig: test "term51: initialCapacity: maximum cols still lays out"
func TestPl2_term51_initialCapacity_maximum_cols_still_lays_out(t *testing.T) {
	testStdCapacity(t)

	const max_cols = math.MaxUint16
	cap := pageListInitialCapacity(max_cols)
	expectEqual[SizeCellCountInt](t, max_cols, cap.cols)
	expectEqual(t, PageStdCapacity.rows, cap.rows)
	expect(t, PageLayoutOf(cap).total_size <= SizeMaxPageSize, "Page.layout(cap).total_size <= size.max_page_size")
}

// Zig: test "term51: initialCapacity: layout exact rows per standard page"
func TestPl2_term51_initialCapacity_layout_exact_rows_per_standard_page(t *testing.T) {
	testStdCapacity(t)

	// Exact numbers for the pinned build (these depend on the byte layout
	// of Row/Cell and the standard capacity).
	expectEqual[uint64](t, 454656, pageListStdSize())
	type Case struct{ cols, rows SizeCellCountInt }
	for _, c := range []Case{
		{cols: 0, rows: 46951},
		{cols: 1, rows: 23475},
		{cols: 2, rows: 15650},
		{cols: 80, rows: 579},
		{cols: 215, rows: 217},
		{cols: 216, rows: 216},
		{cols: 500, rows: 93},
		{cols: 1000, rows: 46},
		{cols: 46950, rows: 1},
		{cols: 46951, rows: 215},
		{cols: 65535, rows: 215},
	} {
		expectEqual(t, c.rows, pageListInitialCapacity(c.cols).rows)
	}
	expectEqual[uint64](t, 454656, PageLayoutOf(pageListInitialCapacity(80)).total_size)
	expectEqual[uint64](t, 80838656, PageLayoutOf(pageListInitialCapacity(46951)).total_size)
	expectEqual[uint64](t, 112803840, PageLayoutOf(pageListInitialCapacity(65535)).total_size)
}

// Zig: test "term51: minMaxSize: pages needed is ceil(rows / cap rows) plus one"
func TestPl2_term51_minMaxSize_pages_needed_is_ceil_rows_cap_rows_plus_one(t *testing.T) {
	testStdCapacity(t)

	// The result is always a whole number of pool items.
	type Case struct{ cols, rows SizeCellCountInt }
	for _, c := range []Case{
		{cols: 1, rows: 1},
		{cols: 80, rows: 24},
		{cols: 80, rows: 579},
		{cols: 80, rows: 580},
		{cols: 80, rows: 1158},
		{cols: 80, rows: 1159},
		{cols: 500, rows: 93},
		{cols: 500, rows: 94},
		{cols: 46950, rows: 24},
		{cols: 46951, rows: 215},
		{cols: 46951, rows: 216},
		{cols: 65535, rows: 65535},
		{cols: 1, rows: 65535},
		{cols: 0, rows: 1},
	} {
		cap := pageListInitialCapacity(c.cols)
		var exact uint64
		if cap.rows >= c.rows {
			exact = 1
		} else {
			exact = (uint64(c.rows) + uint64(cap.rows) - 1) / uint64(cap.rows)
		}
		got := pageListMinMaxSize(c.cols, c.rows)
		expectEqual(t, pageListPagePoolItemSize()*(exact+1), got)
		expectEqual[uint64](t, 0, got%pageListPagePoolItemSize())
	}
}

// Zig: test "term51: minMaxSize: smallest result is two pages"
func TestPl2_term51_minMaxSize_smallest_result_is_two_pages(t *testing.T) {
	testStdCapacity(t)

	// Even for a single row the minimum is two pages: one for the active
	// area and one so the active area can straddle two pages.
	expectEqual(t, 2*pageListStdSize(), pageListMinMaxSize(1, 1))
	expectEqual(t, 2*pageListStdSize(), pageListMinMaxSize(80, 24))
	expectEqual(t, 2*pageListStdSize(), pageListMinMaxSize(0, 1))
	// Largest cols: non-standard page, but the item size is still the
	// standard one.
	expectEqual(t, 2*pageListStdSize(), pageListMinMaxSize(65535, 1))
	expectEqual(t, 2*pageListStdSize(), pageListMinMaxSize(46951, 215))
}

// Zig: test "term51: minMaxSize: boundary at rows equal to page rows"
func TestPl2_term51_minMaxSize_boundary_at_rows_equal_to_page_rows(t *testing.T) {
	testStdCapacity(t)

	// cols=80 holds 579 rows per page.
	expectEqual(t, pageListInitialCapacity(80).rows, 579)
	expectEqual(t, 2*pageListStdSize(), pageListMinMaxSize(80, 578))
	expectEqual(t, 2*pageListStdSize(), pageListMinMaxSize(80, 579))
	// One row past a full page needs a second page (+1 extra).
	expectEqual(t, 3*pageListStdSize(), pageListMinMaxSize(80, 580))
	expectEqual(t, 3*pageListStdSize(), pageListMinMaxSize(80, 1158))
	expectEqual(t, 4*pageListStdSize(), pageListMinMaxSize(80, 1159))

	// Non-standard pages hold 215 rows.
	expectEqual(t, 2*pageListStdSize(), pageListMinMaxSize(46951, 215))
	expectEqual(t, 3*pageListStdSize(), pageListMinMaxSize(46951, 216))
}

// Zig: test "term51: minMaxSize: extreme dimensions"
func TestPl2_term51_minMaxSize_extreme_dimensions(t *testing.T) {
	testStdCapacity(t)

	// One row per page: every row is a page, plus one.
	expectEqual(t, 25*pageListStdSize(), pageListMinMaxSize(46950, 24))
	expectEqual(t, 65536*pageListStdSize(), pageListMinMaxSize(46950, 65535))
	// cols=1: 23475 rows per page.
	expectEqual(t, 2*pageListStdSize(), pageListMinMaxSize(1, 23475))
	expectEqual(t, 3*pageListStdSize(), pageListMinMaxSize(1, 23476))
	expectEqual(t, 4*pageListStdSize(), pageListMinMaxSize(1, 65535))
	// Max rows and cols: ceil(65535 / 215) = 305 pages, plus one.
	expectEqual(t, 306*pageListStdSize(), pageListMinMaxSize(65535, 65535))
}

// Zig: test "term51: minMaxSize: matches what init stores"
func TestPl2_term51_minMaxSize_matches_what_init_stores(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 500, 100, None[uint64]())
	defer s.Deinit()
	expectEqual(t, pageListMinMaxSize(500, 100), s.min_max_size)
	expectEqual(t, 3*pageListStdSize(), s.min_max_size)
}

// Zig: test "term51: init: fields of a default list"
func TestPl2_term51_init_fields_of_a_default_list(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	expectEqual[SizeCellCountInt](t, 80, s.cols)
	expectEqual[SizeCellCountInt](t, 24, s.rows)
	expectEqual[uint](t, 24, s.total_rows)
	expectEqual[uint64](t, 1, s.page_serial)
	expectEqual[uint64](t, 0, s.page_serial_min)
	expectEqual(t, pageListStdSize(), s.page_size)
	expectEqual[uint64](t, math.MaxUint64, s.explicit_max_size)
	expectEqual(t, pageListMinMaxSize(80, 24), s.min_max_size)
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	expectEqual(t, None[uint](), s.viewport_pin_row_offset)
	// Zig-only: try std.testing.expect(s.pool.alloc.ptr == alloc.ptr); (the allocator is not kept)

	// One page, serial 0, rows = 24, cols = 80, standard memory.
	pl2JExpectPages(t, s, []uint{24})
	node := s.pages.first
	expect(t, node.prev == nil, "node.prev == null")
	expect(t, node.next == nil, "node.next == null")
	expectEqual[uint64](t, 0, node.serial)
	expectEqual[SizeCellCountInt](t, 80, node.data.size.cols)
	expectEqual[SizeCellCountInt](t, 80, node.data.capacity.cols)
	expectEqual[SizeCellCountInt](t, 579, node.data.capacity.rows)
	expectEqual(t, pageListStdSize(), pageListMemoryLen(&node.data))

	// The only tracked pin is the viewport pin at the top left.
	expectEqual[uint](t, 1, s.CountTrackedPins())
	expect(t, s.TrackedPins()[0] == s.viewport_pin, "s.trackedPins()[0] == s.viewport_pin")
	expect(t, s.viewport_pin.Node == node, "s.viewport_pin.node == node")
	expectEqual[SizeCellCountInt](t, 0, s.viewport_pin.X)
	expectEqual[SizeCellCountInt](t, 0, s.viewport_pin.Y)
	expect(t, !s.viewport_pin.Garbage, "!s.viewport_pin.garbage")

	// The grid is blank.
	for y := uint(0); y < 24; y++ {
		expectEqual[uint32](t, 0, pl2JCp(t, s, pl2Screen(0, uint32(y))))
	}
}

// Zig: test "term51: init: max size handling"
func TestPl2_term51_init_max_size_handling(t *testing.T) {
	testStdCapacity(t)

	// null means unlimited.
	func() {
		s := pl2Init(t, 80, 24, None[uint64]())
		defer s.Deinit()
		expectEqual[uint64](t, math.MaxUint64, s.explicit_max_size)
		expectEqual[uint64](t, math.MaxUint64, s.MaxSize())
	}()

	// Zero is stored as-is but maxSize clamps up to the minimum.
	func() {
		s := pl2Init(t, 80, 24, pl2Max(0))
		defer s.Deinit()
		expectEqual[uint64](t, 0, s.explicit_max_size)
		expectEqual(t, s.min_max_size, s.MaxSize())
		expectEqual(t, 2*pageListStdSize(), s.MaxSize())
		// Still a full active area.
		expectEqual[uint](t, 24, s.total_rows)
		expectEqual(t, pageListStdSize(), s.page_size)
	}()

	// One below the minimum clamps; exactly the minimum and above do not.
	func() {
		s := pl2Init(t, 80, 24, Some(2*pageListStdSize()-1))
		defer s.Deinit()
		expectEqual(t, 2*pageListStdSize()-1, s.explicit_max_size)
		expectEqual(t, 2*pageListStdSize(), s.MaxSize())
	}()
	func() {
		s := pl2Init(t, 80, 24, Some(2*pageListStdSize()))
		defer s.Deinit()
		expectEqual(t, 2*pageListStdSize(), s.MaxSize())
	}()
	func() {
		s := pl2Init(t, 80, 24, Some(10*pageListStdSize()+1))
		defer s.Deinit()
		expectEqual(t, 10*pageListStdSize()+1, s.explicit_max_size)
		expectEqual(t, 10*pageListStdSize()+1, s.MaxSize())
	}()

	// The largest representable explicit size.
	func() {
		s := pl2Init(t, 80, 24, Some(uint64(math.MaxUint64)))
		defer s.Deinit()
		expectEqual[uint64](t, math.MaxUint64, s.explicit_max_size)
	}()
}

// Zig: test "term51: init: smallest dimensions"
func TestPl2_term51_init_smallest_dimensions(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 1, 1, None[uint64]())
	defer s.Deinit()
	pl2JExpectPages(t, s, []uint{1})
	expectEqual[uint](t, 1, s.total_rows)
	expectEqual[SizeCellCountInt](t, 1, s.cols)
	expectEqual[SizeCellCountInt](t, 1, s.rows)
	expectEqual[SizeCellCountInt](t, 23475, s.pages.first.data.capacity.rows)
	expectEqual(t, 2*pageListStdSize(), s.min_max_size)
	expectEqual(t, PageListPin{Node: s.pages.first, Y: 0, X: 0}, s.GetTopLeft(PointTagActive))
	expect(t, !s.Pin(pl2Active(1, 0)).Valid, "s.pin(.{ .active = .{ .x = 1, .y = 0 } }) == null")
	expect(t, !s.Pin(pl2Active(0, 1)).Valid, "s.pin(.{ .active = .{ .x = 0, .y = 1 } }) == null")
}

// Zig: test "term51: init: rows split across pages at the page row capacity"
func TestPl2_term51_init_rows_split_across_pages_at_the_page_row_capacity(t *testing.T) {
	testStdCapacity(t)

	// cols=500 holds 93 rows per page.
	type Case struct {
		rows  SizeCellCountInt
		pages []uint
	}
	for _, c := range []Case{
		{rows: 92, pages: []uint{92}},
		{rows: 93, pages: []uint{93}},
		{rows: 94, pages: []uint{93, 1}},
		{rows: 186, pages: []uint{93, 93}},
		{rows: 187, pages: []uint{93, 93, 1}},
		{rows: 200, pages: []uint{93, 93, 14}},
	} {
		// Zig's `defer s.deinit()` runs at the end of each iteration.
		func() {
			s := pl2Init(t, 500, c.rows, None[uint64]())
			defer s.Deinit()
			pl2JExpectPages(t, s, c.pages)
			expectEqual(t, uint(c.rows), s.total_rows)
			expectEqual(t, uint64(len(c.pages))*pageListStdSize(), s.page_size)
			expectEqual(t, uint(len(c.pages)), uint(s.page_serial))

			// Serials are 0..n-1 in order, every page has the list's columns
			// and the full capacity of the row count per page.
			var serial uint64 = 0
			for node := s.pages.first; node != nil; node = node.next {
				expectEqual(t, serial, node.serial)
				serial += 1
				expectEqual[SizeCellCountInt](t, 500, node.data.size.cols)
				expectEqual[SizeCellCountInt](t, 93, node.data.capacity.rows)
			}

			// Active area is the whole list.
			expect(t, s.GetTopLeft(PointTagActive).Node == s.pages.first, "s.getTopLeft(.active).node == s.pages.first.?")
			expectEqual[SizeCellCountInt](t, 0, s.GetTopLeft(PointTagActive).Y)
		}()
	}
}

// Zig: test "term51: init: one row per page"
func TestPl2_term51_init_one_row_per_page(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 46950, 3, None[uint64]())
	defer s.Deinit()
	pl2JExpectPages(t, s, []uint{1, 1, 1})
	expectEqual[uint64](t, 3, s.page_serial)
	expectEqual(t, 4*pageListStdSize(), s.min_max_size)
	expectEqual(t, 3*pageListStdSize(), s.page_size)
	expectEqual[SizeCellCountInt](t, 1, s.pages.first.data.capacity.rows)
}

// Zig: test "term51: init: maximum rows"
func TestPl2_term51_init_maximum_rows(t *testing.T) {
	testStdCapacity(t)

	// cols=1 holds 23475 rows per page: 65535 = 23475 + 23475 + 18585.
	s := pl2Init(t, 1, 65535, None[uint64]())
	defer s.Deinit()
	pl2JExpectPages(t, s, []uint{23475, 23475, 18585})
	expectEqual[uint](t, 65535, s.total_rows)
	expectEqual[SizeCellCountInt](t, 65535, s.rows)
	expectEqual(t, 4*pageListStdSize(), s.min_max_size)
	expectEqual[uint64](t, 3, s.page_serial)
	expectEqual(t, s.pages.first, s.GetTopLeft(PointTagActive).Node)
	expectEqual[SizeCellCountInt](t, 0, s.GetTopLeft(PointTagActive).Y)

	last := pl2Pin(t, s, pl2Active(0, 65534))
	expect(t, last.Node == s.pages.last, "last.node == s.pages.last.?")
	expectEqual[SizeCellCountInt](t, 18584, last.Y)
	expect(t, !s.Pin(pl2Active(0, 65535)).Valid, "s.pin(.{ .active = .{ .x = 0, .y = 65535 } }) == null")
}

// Zig: test "term51: init: non-standard page for wide columns"
func TestPl2_term51_init_non_standard_page_for_wide_columns(t *testing.T) {
	testStdCapacity(t)

	cols := pl2StdMaxCols(t) + 1
	s := pl2Init(t, cols, 216, None[uint64]())
	defer s.Deinit()

	// Two non-standard pages of 215 + 1 rows, outside the pool.
	pl2JExpectPages(t, s, []uint{215, 1})
	expectEqual[uint64](t, 2, s.page_serial)
	for node := s.pages.first; node != nil; node = node.next {
		expectEqual[uint64](t, 80838656, pageListMemoryLen(&node.data))
		expect(t, pageListMemoryLen(&node.data) > pageListStdSize(), "node.data.memory.len > std_size")
		expectEqual(t, cols, node.data.capacity.cols)
		expectEqual[SizeCellCountInt](t, 215, node.data.capacity.rows)
	}
	expectEqual[uint64](t, 2*80838656, s.page_size)
	// The minimum is still computed with standard pool items.
	expectEqual(t, 3*pageListStdSize(), s.min_max_size)
}

// Zig: test "term51: init: maximum columns"
func TestPl2_term51_init_maximum_columns(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 65535, 1, None[uint64]())
	defer s.Deinit()
	pl2JExpectPages(t, s, []uint{1})
	expectEqual[SizeCellCountInt](t, 65535, s.cols)
	expectEqual[uint64](t, 112803840, s.page_size)
	expectEqual(t, 2*pageListStdSize(), s.min_max_size)
	expect(t, s.Pin(pl2Active(65534, 0)).Valid, "s.pin(.{ .active = .{ .x = 65534, .y = 0 } }) != null")
	expect(t, !s.Pin(pl2Active(65535, 0)).Valid, "s.pin(.{ .active = .{ .x = 65535, .y = 0 } }) == null")
}

// Zig: test "term51: init: cols exactly at the standard limit"
func TestPl2_term51_init_cols_exactly_at_the_standard_limit(t *testing.T) {
	testStdCapacity(t)

	max := pl2StdMaxCols(t)

	// One row per standard page.
	func() {
		s := pl2Init(t, max, 2, None[uint64]())
		defer s.Deinit()
		pl2JExpectPages(t, s, []uint{1, 1})
		expectEqual(t, 2*pageListStdSize(), s.page_size)
	}()
	// One column more is non-standard (a 215 row page).
	func() {
		s := pl2Init(t, max+1, 2, None[uint64]())
		defer s.Deinit()
		pl2JExpectPages(t, s, []uint{2})
		expectEqual[uint64](t, 80838656, s.page_size)
	}()
}

// Zig: test "term51: initPages: single page"
func TestPl2_term51_initPages_single_page(t *testing.T) {
	testStdCapacity(t)

	pool, err := PageListMemoryPoolInit(0)
	zigTry(t, err)
	defer pool.Deinit()

	var serial uint64 = 0
	list, bytes, err := pageListInitPages(&pool, &serial, 80, 24)
	zigTry(t, err)
	expectEqual(t, pageListStdSize(), bytes)
	expectEqual[uint64](t, 1, serial)
	expect(t, list.first != nil, "list.first != null")
	expect(t, list.first == list.last, "list.first == list.last")
	node := list.first
	expect(t, node.prev == nil, "node.prev == null")
	expect(t, node.next == nil, "node.next == null")
	expectEqual[uint64](t, 0, node.serial)
	expectEqual[SizeCellCountInt](t, 24, node.data.size.rows)
	expectEqual[SizeCellCountInt](t, 80, node.data.size.cols)
	expectEqual[SizeCellCountInt](t, 579, node.data.capacity.rows)
	expectEqual[SizeCellCountInt](t, 80, node.data.capacity.cols)
	// Blank page memory.
	rac := node.data.GetRowAndCell(0, 0)
	expectEqual[uint32](t, 0, rac.cell.content().codepoint())
}

// Zig: test "term51: initPages: serial continues from the given value"
func TestPl2_term51_initPages_serial_continues_from_the_given_value(t *testing.T) {
	testStdCapacity(t)

	pool, err := PageListMemoryPoolInit(0)
	zigTry(t, err)
	defer pool.Deinit()

	var serial uint64 = 100
	list, bytes, err := pageListInitPages(&pool, &serial, 500, 200)
	zigTry(t, err)
	expectEqual(t, 3*pageListStdSize(), bytes)
	expectEqual[uint64](t, 103, serial)

	var expected_serial uint64 = 100
	rows := [...]uint{93, 93, 14}
	var i uint = 0
	var prev *PageListNode
	for node := list.first; node != nil; node = node.next {
		expectEqual(t, expected_serial, node.serial)
		expectEqual(t, rows[i], uint(node.data.size.rows))
		expect(t, node.prev == prev, "node.prev == prev")
		prev = node
		expected_serial += 1
		i += 1
	}
	expectEqual[uint](t, 3, i)
	expect(t, list.last == prev, "list.last == prev")
}

// Zig: test "term51: initPages: row and page boundaries"
func TestPl2_term51_initPages_row_and_page_boundaries(t *testing.T) {
	testStdCapacity(t)

	pool, err := PageListMemoryPoolInit(0)
	zigTry(t, err)
	defer pool.Deinit()

	type Case struct {
		cols  SizeCellCountInt
		rows  SizeCellCountInt
		pages []uint
	}
	for _, c := range []Case{
		{cols: 500, rows: 1, pages: []uint{1}},
		{cols: 500, rows: 93, pages: []uint{93}},
		{cols: 500, rows: 94, pages: []uint{93, 1}},
		{cols: 500, rows: 279, pages: []uint{93, 93, 93}},
		{cols: 46950, rows: 4, pages: []uint{1, 1, 1, 1}},
		{cols: 1, rows: 65535, pages: []uint{23475, 23475, 18585}},
	} {
		var serial uint64 = 0
		list, bytes, err := pageListInitPages(&pool, &serial, c.cols, c.rows)
		zigTry(t, err)
		expectEqual(t, uint64(len(c.pages))*pageListStdSize(), bytes)
		expectEqual(t, uint64(len(c.pages)), serial)
		var i uint = 0
		var rows_total uint = 0
		for node := list.first; node != nil; node = node.next {
			expectEqual(t, c.pages[i], uint(node.data.size.rows))
			expectEqual(t, c.cols, node.data.size.cols)
			rows_total += uint(node.data.size.rows)
			i += 1
		}
		expectEqual(t, uint(len(c.pages)), i)
		expectEqual(t, uint(c.rows), rows_total)
	}
}

// Zig: test "term51: initPages: non-standard pages come from the page allocator"
func TestPl2_term51_initPages_non_standard_pages_come_from_the_page_allocator(t *testing.T) {
	testStdCapacity(t)

	pool, err := PageListMemoryPoolInit(0)
	zigTry(t, err)
	defer pool.Deinit()

	var serial uint64 = 0
	list, bytes, err := pageListInitPages(&pool, &serial, pl2StdMaxCols(t)+1, 216)
	zigTry(t, err)
	// Zig-only: defer term51JFreeNonStd(&pool, list); (frees page-allocator memory by hand)
	expectEqual[uint64](t, 2*80838656, bytes)
	expectEqual[uint64](t, 2, serial)
	expectEqual[SizeCellCountInt](t, 215, list.first.data.size.rows)
	expectEqual[SizeCellCountInt](t, 1, list.last.data.size.rows)
	expectEqual[uint64](t, 80838656, pageListMemoryLen(&list.first.data))
	// Zig-only: try std.testing.expect(@intFromPtr(list.first.?.data.memory.ptr) % std.heap.page_size_min == 0); (page alignment of the raw buffer)
}

// Zig: test "term51: assertIntegrity: valid lists do not panic"
func TestPl2_term51_assertIntegrity_valid_lists_do_not_panic(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	s.AssertIntegrity()

	zigTry(t, s.growRows(30))
	s.AssertIntegrity()

	// A tracked pin and a pin viewport are both fine.
	p := pl2Track(t, s, pl2Pin(t, s, pl2Screen(3, 10)))
	defer s.UntrackPin(p)
	s.viewport = PageListViewport{Tag: PageListViewportPin}
	*s.viewport_pin = pl2Pin(t, s, pl2Screen(0, 5))
	s.AssertIntegrity()
	s.viewport = PageListViewport{Tag: PageListViewportTop}
	s.AssertIntegrity()
	s.viewport = PageListViewport{Tag: PageListViewportActive}
	s.AssertIntegrity()
}

// Zig: test "term51: assertIntegrity: does nothing while paused"
func TestPl2_term51_assertIntegrity_does_nothing_while_paused(t *testing.T) {
	testStdCapacity(t)

	// Zig: if (comptime !build_options.slow_runtime_safety) return error.SkipZigTest;
	if !pageSlowRuntimeSafety {
		t.Skip("needs slow_runtime_safety")
	}

	s := pl2Init(t, 80, 24, None[uint64]())
	defer s.Deinit()

	// Break the cached row count; assertIntegrity would panic if not paused.
	s.total_rows += 3
	s.PauseIntegrityChecks(true)
	s.AssertIntegrity()
	s.total_rows -= 3
	s.PauseIntegrityChecks(false)
	s.AssertIntegrity()
}

// Zig: test "term51: pauseIntegrityChecks: counts nested pauses"
func TestPl2_term51_pauseIntegrityChecks_counts_nested_pauses(t *testing.T) {
	testStdCapacity(t)

	// Zig: if (comptime !build_options.slow_runtime_safety) return error.SkipZigTest;
	if !pageSlowRuntimeSafety {
		t.Skip("needs slow_runtime_safety")
	}

	s := pl2Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	expectEqual[uint64](t, 0, s.pause_integrity_checks)

	s.PauseIntegrityChecks(true)
	expectEqual[uint64](t, 1, s.pause_integrity_checks)
	s.PauseIntegrityChecks(true)
	expectEqual[uint64](t, 2, s.pause_integrity_checks)

	// Broken state is tolerated while paused at any depth.
	s.total_rows += 1
	zigTry(t, s.verifyIntegrity())
	s.PauseIntegrityChecks(false)
	expectEqual[uint64](t, 1, s.pause_integrity_checks)
	zigTry(t, s.verifyIntegrity())
	s.PauseIntegrityChecks(false)
	expectEqual[uint64](t, 0, s.pause_integrity_checks)
	// No longer paused: the corruption is detected again.
	pl2ExpectError(t, ErrTotalRowsMismatch, s.verifyIntegrity())
	s.total_rows -= 1
	zigTry(t, s.verifyIntegrity())
}

// Zig: test "term51: verifyIntegrity: valid states"
func TestPl2_term51_verifyIntegrity_valid_states(t *testing.T) {
	testStdCapacity(t)

	// Zig: if (comptime !build_options.slow_runtime_safety) return error.SkipZigTest;
	if !pageSlowRuntimeSafety {
		t.Skip("needs slow_runtime_safety")
	}

	s := pl2Init(t, 500, 24, None[uint64]())
	defer s.Deinit()
	zigTry(t, s.verifyIntegrity())
	zigTry(t, s.growRows(250))
	pl2JExpectPages(t, s, []uint{93, 93, 88})
	zigTry(t, s.verifyIntegrity())

	// Pin viewport with correct and absent cached offsets.
	s.viewport = PageListViewport{Tag: PageListViewportPin}
	*s.viewport_pin = pl2Pin(t, s, pl2Screen(0, 100))
	s.viewport_pin_row_offset = None[uint]()
	zigTry(t, s.verifyIntegrity())
	s.viewport_pin_row_offset = Some[uint](100)
	zigTry(t, s.verifyIntegrity())
	s.viewport = PageListViewport{Tag: PageListViewportActive}
	s.viewport_pin_row_offset = None[uint]()
}

// Zig: test "term51: verifyIntegrity: PageSerialInvalid"
func TestPl2_term51_verifyIntegrity_PageSerialInvalid(t *testing.T) {
	testStdCapacity(t)

	// Zig: if (comptime !build_options.slow_runtime_safety) return error.SkipZigTest;
	if !pageSlowRuntimeSafety {
		t.Skip("needs slow_runtime_safety")
	}

	s := pl2Init(t, 500, 100, None[uint64]())
	defer s.Deinit()
	pl2JExpectPages(t, s, []uint{93, 7})
	first := s.pages.first
	last := s.pages.last
	expectEqual[uint64](t, 0, first.serial)
	expectEqual[uint64](t, 1, last.serial)

	// The minimum is inclusive: serial == min is fine.
	s.page_serial_min = 0
	zigTry(t, s.verifyIntegrity())
	first.serial = 5
	last.serial = 6
	s.page_serial_min = 5
	zigTry(t, s.verifyIntegrity())

	// One past the first node's serial fails.
	s.page_serial_min = 6
	pl2ExpectError(t, ErrPageSerialInvalid, s.verifyIntegrity())
	s.page_serial_min = 7
	pl2ExpectError(t, ErrPageSerialInvalid, s.verifyIntegrity())

	// Only the last node is too low.
	first.serial = 9
	last.serial = 4
	s.page_serial_min = 5
	pl2ExpectError(t, ErrPageSerialInvalid, s.verifyIntegrity())
	s.page_serial_min = 4
	zigTry(t, s.verifyIntegrity())

	// The serial check comes before the row count check.
	s.page_serial_min = 5
	s.total_rows += 1
	pl2ExpectError(t, ErrPageSerialInvalid, s.verifyIntegrity())

	// Paused: nothing is checked.
	s.PauseIntegrityChecks(true)
	zigTry(t, s.verifyIntegrity())
	s.PauseIntegrityChecks(false)

	s.total_rows -= 1
	s.page_serial_min = 0
	first.serial = 0
	last.serial = 1
	zigTry(t, s.verifyIntegrity())
}

// Zig: test "term51: verifyIntegrity: TotalRowsMismatch"
func TestPl2_term51_verifyIntegrity_TotalRowsMismatch(t *testing.T) {
	testStdCapacity(t)

	// Zig: if (comptime !build_options.slow_runtime_safety) return error.SkipZigTest;
	if !pageSlowRuntimeSafety {
		t.Skip("needs slow_runtime_safety")
	}

	s := pl2Init(t, 500, 100, None[uint64]())
	defer s.Deinit()
	expectEqual[uint](t, 100, s.total_rows)
	zigTry(t, s.verifyIntegrity())

	// Off by one in either direction and wildly off.
	s.total_rows = 101
	pl2ExpectError(t, ErrTotalRowsMismatch, s.verifyIntegrity())
	s.total_rows = 99
	pl2ExpectError(t, ErrTotalRowsMismatch, s.verifyIntegrity())
	s.total_rows = 0
	pl2ExpectError(t, ErrTotalRowsMismatch, s.verifyIntegrity())
	s.total_rows = math.MaxUint
	pl2ExpectError(t, ErrTotalRowsMismatch, s.verifyIntegrity())

	// A page's own row count changing without updating the cache.
	s.total_rows = 100
	s.pages.last.data.size.rows += 1
	pl2ExpectError(t, ErrTotalRowsMismatch, s.verifyIntegrity())
	s.pages.last.data.size.rows -= 1

	// It takes precedence over bad tracked pins.
	p := pl2Track(t, s, pl2Pin(t, s, pl2Screen(0, 50)))
	defer s.UntrackPin(p)
	p.Y = 200
	s.total_rows = 99
	pl2ExpectError(t, ErrTotalRowsMismatch, s.verifyIntegrity())
	s.total_rows = 100
	p.Y = 50
	zigTry(t, s.verifyIntegrity())
}

// Zig: test "term51: verifyIntegrity: TrackedPinInvalid"
func TestPl2_term51_verifyIntegrity_TrackedPinInvalid(t *testing.T) {
	testStdCapacity(t)

	// Zig: if (comptime !build_options.slow_runtime_safety) return error.SkipZigTest;
	if !pageSlowRuntimeSafety {
		t.Skip("needs slow_runtime_safety")
	}

	s := pl2Init(t, 500, 100, None[uint64]())
	defer s.Deinit()
	other := pl2Init(t, 500, 10, None[uint64]())
	defer other.Deinit()

	p := pl2Track(t, s, pl2Pin(t, s, pl2Screen(0, 0)))
	defer s.UntrackPin(p)
	zigTry(t, s.verifyIntegrity())

	// y: last row of the page is valid, one past is not.
	p.Node = s.pages.first
	p.Y = 92
	zigTry(t, s.verifyIntegrity())
	p.Y = 93
	pl2ExpectError(t, ErrTrackedPinInvalid, s.verifyIntegrity())

	// The last page has only 7 rows.
	p.Node = s.pages.last
	p.Y = 6
	zigTry(t, s.verifyIntegrity())
	p.Y = 7
	pl2ExpectError(t, ErrTrackedPinInvalid, s.verifyIntegrity())

	// x: last column valid, one past not (checked against the page's
	// size.cols).
	p.Y = 0
	p.X = 499
	zigTry(t, s.verifyIntegrity())
	p.X = 500
	pl2ExpectError(t, ErrTrackedPinInvalid, s.verifyIntegrity())
	p.X = math.MaxUint16
	pl2ExpectError(t, ErrTrackedPinInvalid, s.verifyIntegrity())

	// The page's size.cols (not the list's cols) is the bound.
	p.X = 100
	s.pages.last.data.size.cols = 100
	pl2ExpectError(t, ErrTrackedPinInvalid, s.verifyIntegrity())
	p.X = 99
	zigTry(t, s.verifyIntegrity())
	s.pages.last.data.size.cols = 500
	p.X = 0

	// A node that is not part of this list.
	p.Node = other.pages.first
	pl2ExpectError(t, ErrTrackedPinInvalid, s.verifyIntegrity())

	// The viewport pin is itself tracked so it is checked too.
	p.Node = s.pages.first
	zigTry(t, s.verifyIntegrity())
	s.viewport_pin.Y = 93
	pl2ExpectError(t, ErrTrackedPinInvalid, s.verifyIntegrity())
	s.viewport_pin.Y = 0
	zigTry(t, s.verifyIntegrity())

	// Paused: not reported.
	p.Y = 93
	s.PauseIntegrityChecks(true)
	zigTry(t, s.verifyIntegrity())
	s.PauseIntegrityChecks(false)
	p.Y = 0
}

// Zig: test "term51: verifyIntegrity: viewport pin insufficient rows"
func TestPl2_term51_verifyIntegrity_viewport_pin_insufficient_rows(t *testing.T) {
	testStdCapacity(t)

	// Zig: if (comptime !build_options.slow_runtime_safety) return error.SkipZigTest;
	if !pageSlowRuntimeSafety {
		t.Skip("needs slow_runtime_safety")
	}

	s := pl2Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	zigTry(t, s.growRows(30))
	pl2JExpectPages(t, s, []uint{54})

	s.viewport = PageListViewport{Tag: PageListViewportPin}
	s.viewport_pin_row_offset = None[uint]()

	// 54 total rows, 24 needed: offsets 0..30 are fine.
	for _, y := range []SizeCellCountInt{0, 1, 29, 30} {
		*s.viewport_pin = PageListPin{Node: s.pages.first, Y: y}
		zigTry(t, s.verifyIntegrity())
	}
	// 31 leaves only 23 rows.
	for _, y := range []SizeCellCountInt{31, 40, 53} {
		*s.viewport_pin = PageListPin{Node: s.pages.first, Y: y}
		pl2ExpectError(t, ErrViewportPinInsufficientRows, s.verifyIntegrity())
	}

	// Paused: not reported.
	s.PauseIntegrityChecks(true)
	zigTry(t, s.verifyIntegrity())
	s.PauseIntegrityChecks(false)

	s.viewport = PageListViewport{Tag: PageListViewportActive}
	*s.viewport_pin = PageListPin{Node: s.pages.first}
	zigTry(t, s.verifyIntegrity())
}

// Zig: test "term51: verifyIntegrity: viewport pin across pages"
func TestPl2_term51_verifyIntegrity_viewport_pin_across_pages(t *testing.T) {
	testStdCapacity(t)

	// Zig: if (comptime !build_options.slow_runtime_safety) return error.SkipZigTest;
	if !pageSlowRuntimeSafety {
		t.Skip("needs slow_runtime_safety")
	}

	s := pl2Init(t, 500, 24, None[uint64]())
	defer s.Deinit()
	zigTry(t, s.growRows(250))
	pl2JExpectPages(t, s, []uint{93, 93, 88})
	s.viewport = PageListViewport{Tag: PageListViewportPin}

	// Offsets are summed across the pages above the pin's page.
	type Case struct {
		page   uint
		y      SizeCellCountInt
		offset uint
	}
	for _, c := range []Case{
		{page: 0, y: 0, offset: 0},
		{page: 0, y: 92, offset: 92},
		{page: 1, y: 0, offset: 93},
		{page: 1, y: 10, offset: 103},
		{page: 2, y: 0, offset: 186},
		{page: 2, y: 64, offset: 250},
	} {
		node := s.pages.first
		for i := uint(0); i < c.page; i++ {
			node = node.next
		}
		*s.viewport_pin = PageListPin{Node: node, Y: c.y}

		s.viewport_pin_row_offset = None[uint]()
		zigTry(t, s.verifyIntegrity())
		s.viewport_pin_row_offset = Some(c.offset)
		zigTry(t, s.verifyIntegrity())
		s.viewport_pin_row_offset = Some(c.offset + 1)
		pl2ExpectError(t, ErrViewportPinOffsetMismatch, s.verifyIntegrity())
		if c.offset > 0 {
			s.viewport_pin_row_offset = Some(c.offset - 1)
			pl2ExpectError(t, ErrViewportPinOffsetMismatch, s.verifyIntegrity())
		}
	}

	// Boundary of the insufficient rows check: 274 total rows, 24 needed.
	s.viewport_pin_row_offset = None[uint]()
	*s.viewport_pin = PageListPin{Node: s.pages.last, Y: 64}
	zigTry(t, s.verifyIntegrity())
	*s.viewport_pin = PageListPin{Node: s.pages.last, Y: 65}
	pl2ExpectError(t, ErrViewportPinInsufficientRows, s.verifyIntegrity())
	*s.viewport_pin = PageListPin{Node: s.pages.last, Y: 87}
	pl2ExpectError(t, ErrViewportPinInsufficientRows, s.verifyIntegrity())

	s.viewport = PageListViewport{Tag: PageListViewportActive}
	*s.viewport_pin = PageListPin{Node: s.pages.first}
}

// Zig: test "term51: verifyIntegrity: viewport pin offset mismatch precedes insufficient rows"
func TestPl2_term51_verifyIntegrity_viewport_pin_offset_mismatch_precedes_insufficient_rows(t *testing.T) {
	testStdCapacity(t)

	// Zig: if (comptime !build_options.slow_runtime_safety) return error.SkipZigTest;
	if !pageSlowRuntimeSafety {
		t.Skip("needs slow_runtime_safety")
	}

	s := pl2Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	zigTry(t, s.growRows(30))

	s.viewport = PageListViewport{Tag: PageListViewportPin}
	*s.viewport_pin = PageListPin{Node: s.pages.first, Y: 40}

	// Wrong cache and too few rows: the mismatch is reported first.
	s.viewport_pin_row_offset = Some[uint](1)
	pl2ExpectError(t, ErrViewportPinOffsetMismatch, s.verifyIntegrity())
	// Correct cache: insufficient rows.
	s.viewport_pin_row_offset = Some[uint](40)
	pl2ExpectError(t, ErrViewportPinInsufficientRows, s.verifyIntegrity())

	s.viewport = PageListViewport{Tag: PageListViewportActive}
	*s.viewport_pin = PageListPin{Node: s.pages.first}
	s.viewport_pin_row_offset = None[uint]()
}

// Zig: test "term51: verifyIntegrity: viewport pin not in list"
func TestPl2_term51_verifyIntegrity_viewport_pin_not_in_list(t *testing.T) {
	testStdCapacity(t)

	// Zig: if (comptime !build_options.slow_runtime_safety) return error.SkipZigTest;
	if !pageSlowRuntimeSafety {
		t.Skip("needs slow_runtime_safety")
	}

	s := pl2Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	other := pl2Init(t, 80, 24, None[uint64]())
	defer other.Deinit()

	// With the viewport pin tracked, a foreign node is first reported as
	// an invalid tracked pin.
	s.viewport = PageListViewport{Tag: PageListViewportPin}
	vp_node := s.viewport_pin.Node
	s.viewport_pin.Node = other.pages.first
	pl2ExpectError(t, ErrTrackedPinInvalid, s.verifyIntegrity())

	// Hand construct the otherwise unreachable state: viewport pin not
	// tracked and not in the list.
	expect(t, s.tracked_pins.SwapRemove(s.viewport_pin), "s.tracked_pins.swapRemove(s.viewport_pin)")
	pl2ExpectError(t, ErrViewportPinOffsetMismatch, s.verifyIntegrity())
	// Even with a cached offset set.
	s.viewport_pin_row_offset = Some[uint](0)
	pl2ExpectError(t, ErrViewportPinOffsetMismatch, s.verifyIntegrity())

	// Not a pin viewport: the viewport pin isn't looked at.
	s.viewport = PageListViewport{Tag: PageListViewportActive}
	zigTry(t, s.verifyIntegrity())

	// Restore.
	s.viewport_pin.Node = vp_node
	s.viewport_pin_row_offset = None[uint]()
	zigTry(t, s.tracked_pins.PutNoClobber(s.viewport_pin))
	zigTry(t, s.verifyIntegrity())
}

// Zig: test "term51: verifyIntegrity: viewport state ignored unless pin"
func TestPl2_term51_verifyIntegrity_viewport_state_ignored_unless_pin(t *testing.T) {
	testStdCapacity(t)

	// Zig: if (comptime !build_options.slow_runtime_safety) return error.SkipZigTest;
	if !pageSlowRuntimeSafety {
		t.Skip("needs slow_runtime_safety")
	}

	s := pl2Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	zigTry(t, s.growRows(30))

	// Nonsense cached offset and a pin with too few rows below it do not
	// matter for the active and top viewports.
	s.viewport_pin_row_offset = Some[uint](9999)
	*s.viewport_pin = PageListPin{Node: s.pages.first, Y: 53}
	s.viewport = PageListViewport{Tag: PageListViewportActive}
	zigTry(t, s.verifyIntegrity())
	s.viewport = PageListViewport{Tag: PageListViewportTop}
	zigTry(t, s.verifyIntegrity())
	s.viewport = PageListViewport{Tag: PageListViewportPin}
	pl2ExpectError(t, ErrViewportPinOffsetMismatch, s.verifyIntegrity())

	s.viewport = PageListViewport{Tag: PageListViewportActive}
	*s.viewport_pin = PageListPin{Node: s.pages.first}
	s.viewport_pin_row_offset = None[uint]()
}

// Zig: test "term51: verifyIntegrity: tracked pin check precedes viewport checks"
func TestPl2_term51_verifyIntegrity_tracked_pin_check_precedes_viewport_checks(t *testing.T) {
	testStdCapacity(t)

	// Zig: if (comptime !build_options.slow_runtime_safety) return error.SkipZigTest;
	if !pageSlowRuntimeSafety {
		t.Skip("needs slow_runtime_safety")
	}

	s := pl2Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	zigTry(t, s.growRows(30))

	p := pl2Track(t, s, pl2Pin(t, s, pl2Screen(0, 1)))
	defer s.UntrackPin(p)
	s.viewport = PageListViewport{Tag: PageListViewportPin}
	*s.viewport_pin = PageListPin{Node: s.pages.first, Y: 53}
	s.viewport_pin_row_offset = Some[uint](5)
	p.Y = 54
	pl2ExpectError(t, ErrTrackedPinInvalid, s.verifyIntegrity())
	p.Y = 1
	pl2ExpectError(t, ErrViewportPinOffsetMismatch, s.verifyIntegrity())

	s.viewport = PageListViewport{Tag: PageListViewportActive}
	*s.viewport_pin = PageListPin{Node: s.pages.first}
	s.viewport_pin_row_offset = None[uint]()
}

// Zig: test "term51: deinit: releases a grown multi-page list with extra pins"
func TestPl2_term51_deinit_releases_a_grown_multi_page_list_with_extra_pins(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 500, 24, None[uint64]())
	zigTry(t, s.growRows(500))
	expect(t, s.TotalPages() > 3, "s.totalPages() > 3")

	// Tracked pins that are never untracked are owned by the pool.
	pl2Track(t, s, pl2Pin(t, s, pl2Screen(1, 1)))
	pl2Track(t, s, pl2Pin(t, s, pl2Screen(2, 400)))
	expectEqual[uint](t, 3, s.CountTrackedPins())

	// Zig-only: the testing allocator reports any leak (pages, nodes, pins and the tracked pin map)
	s.Deinit()
}

// Zig: test "term51: deinit: frees non-standard pages"
func TestPl2_term51_deinit_frees_non_standard_pages(t *testing.T) {
	testStdCapacity(t)

	// 216 rows are two non-standard pages of 215 + 1 rows.
	s := pl2Init(t, pl2StdMaxCols(t)+1, 216, None[uint64]())
	pl2JExpectPages(t, s, []uint{215, 1})
	for node := s.pages.first; node != nil; node = node.next {
		expect(t, pageListMemoryLen(&node.data) > pageListStdSize(), "node.data.memory.len > std_size")
	}
	s.Deinit()
}

// Zig: test "term51: deinit: mixed standard and non-standard pages"
func TestPl2_term51_deinit_mixed_standard_and_non_standard_pages(t *testing.T) {
	testStdCapacity(t)

	// Non-standard pages can only come from a different capacity; build
	// the mix by growing a standard list with an increased-capacity page.
	s := pl2Init(t, 80, 24, None[uint64]())
	zigTry(t, s.growRows(20))
	// Grapheme bytes can double until the page exceeds the standard size.
	node := s.pages.first
	for pageListMemoryLen(&node.data) <= pageListStdSize() {
		node = pl2IncreaseCapacity(t, s, node, PageListIncreaseCapacityGraphemeBytes)
	}
	expect(t, pageListMemoryLen(&s.pages.first.data) > pageListStdSize(), "s.pages.first.?.data.memory.len > std_size")
	zigTry(t, s.growRows(1000))
	expect(t, s.TotalPages() >= 2, "s.totalPages() >= 2")
	expect(t, pageListMemoryLen(&s.pages.last.data) == pageListStdSize(), "s.pages.last.?.data.memory.len == std_size")
	s.Deinit()
}

// Zig: test "term51: deinit: paused integrity checks tolerate corruption"
func TestPl2_term51_deinit_paused_integrity_checks_tolerate_corruption(t *testing.T) {
	testStdCapacity(t)

	// Zig: if (comptime !build_options.slow_runtime_safety) return error.SkipZigTest;
	if !pageSlowRuntimeSafety {
		t.Skip("needs slow_runtime_safety")
	}

	s := pl2Init(t, 80, 24, None[uint64]())
	// deinit asserts integrity first; with checks paused this broken
	// state must not panic.
	s.total_rows += 7
	s.PauseIntegrityChecks(true)
	s.Deinit()
}

// Zig: test "term51: deinit: cloned list does not touch the source"
func TestPl2_term51_deinit_cloned_list_does_not_touch_the_source(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 500, 100, None[uint64]())
	defer s.Deinit()
	pl2JMarkRows(t, s)

	c, err := s.Clone(PageListClone{Top: pl2Screen(0, 0)})
	zigTry(t, err)
	c.Deinit()

	// Source still intact.
	zigTry(t, s.verifyIntegrity())
	expectEqual[uint32](t, 0x100+99, pl2JCp(t, s, pl2Screen(0, 99)))
}

// Zig: test "term51: reset: clears history and rebuilds a single page"
func TestPl2_term51_reset_clears_history_and_rebuilds_a_single_page(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 80, 24, pl2Max(12345678))
	defer s.Deinit()
	zigTry(t, s.growRows(100))
	pl2JMarkRows(t, s)
	pl2JExpectPages(t, s, []uint{124})
	expectEqual[uint64](t, 1, s.page_serial)

	s.Reset()

	pl2JExpectPages(t, s, []uint{24})
	expectEqual[uint](t, 24, s.total_rows)
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	expectEqual(t, pageListStdSize(), s.page_size)

	// The serial keeps counting and the minimum is untouched.
	expectEqual[uint64](t, 2, s.page_serial)
	expectEqual[uint64](t, 1, s.pages.first.serial)
	expectEqual[uint64](t, 0, s.page_serial_min)

	// Sizes and limits are preserved.
	expectEqual[SizeCellCountInt](t, 80, s.cols)
	expectEqual[SizeCellCountInt](t, 24, s.rows)
	expectEqual[uint64](t, 12345678, s.explicit_max_size)
	expectEqual(t, pageListMinMaxSize(80, 24), s.min_max_size)

	// All contents are gone.
	for y := uint(0); y < 24; y++ {
		expectEqual[uint32](t, 0, pl2JCp(t, s, pl2Screen(0, uint32(y))))
	}
	expect(t, !s.Pin(pl2Screen(0, 24)).Valid, "s.pin(.{ .screen = .{ .x = 0, .y = 24 } }) == null")
	expectEqual(t,
		PageListPin{Node: s.pages.first, Y: 0, X: 0},
		s.GetTopLeft(PointTagActive),
	)
	zigTry(t, s.verifyIntegrity())

	// It remains usable afterwards.
	zigTry(t, s.growRows(30))
	expectEqual[uint](t, 54, s.total_rows)
	zigTry(t, s.verifyIntegrity())
}

// Go-only: panic test: assertIntegrity: integrity violation panics (TotalRowsMismatch)
func TestPl2_panic_assertIntegrity_integrity_violation(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	testStdCapacity(t)

	s := pl2Init(t, 80, 24, None[uint64]())
	s.total_rows += 1
	pl2ExpectPanic(t, func() { s.AssertIntegrity() })
	s.total_rows -= 1
	s.Deinit()
}

// Go-only: panic test: verifyIntegrity: assert(!self.viewport_pin.garbage)
func TestPl2_panic_verifyIntegrity_viewport_pin_garbage(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	testStdCapacity(t)

	s := pl2Init(t, 80, 24, None[uint64]())
	s.viewport_pin.Garbage = true
	pl2ExpectPanic(t, func() { _ = s.verifyIntegrity() })
	s.viewport_pin.Garbage = false
	s.Deinit()
}

// Go-only: panic test: deinit: assertIntegrity first, integrity violation panics
func TestPl2_panic_deinit_integrity_violation(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	testStdCapacity(t)

	s := pl2Init(t, 80, 24, None[uint64]())
	s.total_rows += 1
	pl2ExpectPanic(t, func() { s.Deinit() })
}

// Go-only: panic test: initPages: assert(page_list.first != null) with zero rows
func TestPl2_panic_initPages_zero_rows(t *testing.T) {
	testStdCapacity(t)

	pool, err := PageListMemoryPoolInit(0)
	zigTry(t, err)
	defer pool.Deinit()

	var serial uint64 = 0
	pl2ExpectPanic(t, func() { _, _, _ = pageListInitPages(&pool, &serial, 80, 0) })
}

// Go-only: panic test: resize: assert(v > 0) for cols
func TestPl2_panic_resize_zero_cols(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 10, 10, None[uint64]())
	defer s.Deinit()
	pl2ExpectPanic(t, func() { _ = s.Resize(PageListResize{Cols: pl2C(0), Reflow: true}) })
}

// Go-only: panic test: resize: assert(v > 0) for rows
func TestPl2_panic_resize_zero_rows(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 10, 10, None[uint64]())
	defer s.Deinit()
	pl2ExpectPanic(t, func() { _ = s.Resize(PageListResize{Rows: pl2C(0), Reflow: true}) })
}

// Go-only: panic test: resizeCols: assert(cols != self.cols)
func TestPl2_panic_resizeCols_same_cols(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 10, 10, None[uint64]())
	defer s.Deinit()
	pl2ExpectPanic(t, func() { _ = s.resizeCols(10, None[PageListResizeCursor]()) })
}

// Go-only: panic test: compact: assert(page.size.rows > 0)
func TestPl2_panic_compact_zero_row_page(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 10, 10, None[uint64]())
	node := s.pages.first
	node.data.size.rows = 0
	pl2ExpectPanic(t, func() { _, _ = s.Compact(node) })
	node.data.size.rows = 10
	s.Deinit()
}

// Go-only: panic test: split: assert(self.pinIsValid(p)) with a node that is not in the list
func TestPl2_panic_split_invalid_pin(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	testStdCapacity(t)

	s := pl2Init(t, 10, 10, None[uint64]())
	defer s.Deinit()
	other := pl2Init(t, 10, 10, None[uint64]())
	defer other.Deinit()
	pl2ExpectPanic(t, func() { _ = s.Split(PageListPin{Node: other.pages.first, Y: 5, X: 0}) })
}

// Go-only: panic test: split: assert(self.pinIsValid(p)) with y beyond the page rows
func TestPl2_panic_split_pin_y_out_of_range(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	testStdCapacity(t)

	s := pl2Init(t, 10, 10, None[uint64]())
	defer s.Deinit()
	pl2ExpectPanic(t, func() { _ = s.Split(PageListPin{Node: s.pages.first, Y: 10, X: 0}) })
}

// Go-only: panic test: trackPin: assert(self.pinIsValid(p)) with a node that is not in the list
func TestPl2_panic_trackPin_invalid_pin(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	testStdCapacity(t)

	s := pl2Init(t, 10, 10, None[uint64]())
	defer s.Deinit()
	other := pl2Init(t, 10, 10, None[uint64]())
	defer other.Deinit()
	pl2ExpectPanic(t, func() { _, _ = s.TrackPin(PageListPin{Node: other.pages.first, Y: 0, X: 0}) })
}

// Go-only: panic test: untrackPin: assert(p != self.viewport_pin)
func TestPl2_panic_untrackPin_viewport_pin(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 10, 10, None[uint64]())
	defer s.Deinit()
	pl2ExpectPanic(t, func() { s.UntrackPin(s.viewport_pin) })
}

// Go-only: panic test: erasePage: assert(node.next != null or node.prev != null) on the only page
func TestPl2_panic_erasePage_only_page(t *testing.T) {
	testStdCapacity(t)

	s := pl2Init(t, 10, 10, None[uint64]())
	defer s.Deinit()
	pl2ExpectPanic(t, func() { s.erasePage(s.pages.first) })
}
