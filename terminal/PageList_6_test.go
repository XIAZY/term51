// Tests translated one for one from src/terminal/PageList.zig (Ghostty
// v1.3.1 plus the term51 additional tests), last part: the tests from
// "term51: Pin.isDirty: dirty state is per page" through the end of the file
// (all of them are term51 additional tests). Each Go test carries the exact
// Zig title in a "// Zig: test" comment. Helpers are prefixed pl6.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualSlices keep Zig's names and Zig's
//     (expected, actual) argument order. `try` on a call that returns an
//     error is zigTry(t, err); `x.?` is zigUnwrap. Every test starts with
//     testStdCapacity(t) (Zig's unit tests use grapheme_bytes 512).
//   - `init(alloc, cols, rows, null)` is pl6Init; `defer s.deinit()` is
//     `defer s.Deinit()`. The allocator argument and `testing.allocator` are
//     dropped (no counterpart in Go).
//   - Zig's `errdefer s.deinit()` in the helpers is dropped (Go memory).
//   - The helpers term51N* (Pages, Pt, ExpectPin, ExpectChunk, Semantic,
//     Cells, DrainPages, ...) and the other helpers these tests use are
//     translated here as pl6*; `?Pin` parameters are Optional[PageListPin],
//     and pl6ExpectPinV / pl6ExpectChunkV take the plain value Zig coerces to
//     the optional.
//   - A Zig union field access (`r.offset`, `r.overflow`) is safety checked;
//     pl6Offset and pl6Overflow check the tag first and then the test reads
//     the matching Go field.
//   - Raw pointer comparisons of nodes, rows and cells are pointer (or
//     element address) comparisons; `rows.ptr == all + 2` is `&rows[0] ==
//     &all[2]`.
//   - Option structs have no defaults in Go: every iterator literal names its
//     direction (Zig default right_down) and every Resize literal says
//     `reflow: true`.
//   - Tests that were not translated are listed in
//     scripts/not-translated-pl6.txt.

package terminal

import (
	"math"
	"testing"
)

// pl6ExpectPanic runs f and fails the test if f does not panic.
func pl6ExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if recover() == nil {
			t.Fatalf("expected a panic, got none")
		}
	}()
	f()
}

// pl6Init is `try init(alloc, cols, rows, null)`.
func pl6Init(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) PageList {
	t.Helper()
	s, err := PageListInit(cols, rows, None[uint64]())
	zigTry(t, err)
	return s
}

// pl6Grow is `try s.grow()`.
func pl6Grow(t *testing.T, s *PageList) *PageListNode {
	t.Helper()
	n, err := s.Grow()
	zigTry(t, err)
	return n
}

// pl6GrowRows is `try s.growRows(n)`.
func pl6GrowRows(t *testing.T, s *PageList, n uint) {
	t.Helper()
	zigTry(t, s.growRows(n))
}

// ---------------------------------------------------------------------------
// term51: additional tests (N). Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

// pl6Pages is term51NPages: build a PageList (no max size, so nothing is ever
// pruned) in which the first `pages` pages are all completely full
// (size.rows == capacity.rows).
func pl6Pages(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt, pages uint) PageList {
	t.Helper()
	s := pl6Init(t, cols, rows)
	var n uint = 1
	for {
		node := s.pages.last
		node.data.PauseIntegrityChecks(true)
		for node.data.size.rows < node.data.capacity.rows {
			expect(t, pl6Grow(t, &s) == nil, "try s.grow() == null")
		}
		node.data.PauseIntegrityChecks(false)
		if n == pages {
			break
		}
		expect(t, pl6Grow(t, &s) != nil, "try s.grow() != null")
		n += 1
	}
	return s
}

// pl6Pt is term51NPt: shortcut for a screen point.
func pl6Pt(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagScreen, Screen: PointCoordinate{x: x, y: y}}
}

// pl6ExpectPin is term51NExpectPin.
func pl6ExpectPin(t *testing.T, p Optional[PageListPin], node *PageListNode, y uint, x uint) {
	t.Helper()
	expect(t, p.Valid, "p != null")
	got := p.Value
	expect(t, got.node == node, "got.node == node")
	expectEqual(t, y, uint(got.y))
	expectEqual(t, x, uint(got.x))
}

// pl6ExpectPinV is term51NExpectPin for a plain Pin, which Zig coerces to the
// optional parameter.
func pl6ExpectPinV(t *testing.T, p PageListPin, node *PageListNode, y uint, x uint) {
	t.Helper()
	pl6ExpectPin(t, Some(p), node, y, x)
}

// pl6ExpectChunk is term51NExpectChunk.
func pl6ExpectChunk(
	t *testing.T,
	chunk Optional[PageListPageIteratorChunk],
	node *PageListNode,
	start uint,
	end uint,
) {
	t.Helper()
	expect(t, chunk.Valid, "chunk != null")
	got := chunk.Value
	expect(t, got.node == node, "got.node == node")
	expectEqual(t, start, uint(got.start))
	expectEqual(t, end, uint(got.end))
}

// pl6ExpectChunkV is term51NExpectChunk for a plain Chunk.
func pl6ExpectChunkV(t *testing.T, chunk PageListPageIteratorChunk, node *PageListNode, start uint, end uint) {
	t.Helper()
	pl6ExpectChunk(t, Some(chunk), node, start, end)
}

// pl6Offset is the safety checked access `r.offset` of a Pin.downOverflow /
// Pin.upOverflow result.
func pl6Offset(t *testing.T, r PageListPinOverflow) PageListPin {
	t.Helper()
	if r.Tag != PageListPinOverflowOffset {
		t.Fatalf("access of union field offset while field %v is active", r.Tag)
	}
	return r.Offset
}

// pl6Overflow is the safety checked access `r.overflow`: it checks the tag and
// returns r, whose Overflow field the test then reads.
func pl6Overflow(t *testing.T, r PageListPinOverflow) PageListPinOverflow {
	t.Helper()
	if r.Tag != PageListPinOverflowOverflow {
		t.Fatalf("access of union field overflow while field %v is active", r.Tag)
	}
	return r
}

// pl6LimitRow is the Zig literal `.{ .row = p }` of a PageIterator.Limit.
func pl6LimitRow(p PageListPin) pageListPageIteratorLimit {
	return pageListPageIteratorLimit{Tag: pageListPageIteratorLimitRow, Row: p}
}

// pl6LimitCount is the Zig literal `.{ .count = n }` of a PageIterator.Limit.
func pl6LimitCount(n uint) pageListPageIteratorLimit {
	return pageListPageIteratorLimit{Tag: pageListPageIteratorLimitCount, Count: n}
}

// pl6DrainPages is term51NDrainPages.
func pl6DrainPages(t *testing.T, it *PageListPageIterator, out []PageListPageIteratorChunk) uint {
	t.Helper()
	var n uint = 0
	for {
		c := it.Next()
		if !c.Valid {
			break
		}
		expect(t, n < uint(len(out)), "n < out.len")
		out[n] = c.Value
		n += 1
	}
	// Exhausted iterators stay exhausted.
	expect(t, !it.row.Valid, "it.row == null")
	expect(t, !it.Next().Valid, "it.next() == null")
	expect(t, !it.Next().Valid, "it.next() == null")
	return n
}

// ---- Pin: rowAndCell, cells, grapheme, style, dirty ----

// Zig: test "term51: Pin.isDirty: dirty state is per page"
func TestPl6_Pin_isDirty_dirty_state_is_per_page(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	s.ClearDirty()
	first := s.pages.first
	last := s.pages.last

	first.data.dirty = true
	expect(t, PageListPin{node: first, y: 5}.IsDirty(), "(Pin{ .node = first, .y = 5 }).isDirty()")
	expect(t, !PageListPin{node: last, y: 5}.IsDirty(), "!(Pin{ .node = last, .y = 5 }).isDirty()")
}

// Zig: test "term51: Pin.markDirty: marks only the row of the pin"
func TestPl6_Pin_markDirty_marks_only_the_row_of_the_pin(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 5, 4)
	defer s.Deinit()
	node := s.pages.first
	page := &node.data
	s.ClearDirty()

	p := PageListPin{node: node, y: 1, x: 3}
	p.MarkDirty()
	expect(t, page.GetRowAndCell(0, 1).row.dirty, "page.getRowAndCell(0, 1).row.dirty")
	expect(t, !page.GetRowAndCell(0, 0).row.dirty, "!page.getRowAndCell(0, 0).row.dirty")
	expect(t, !page.GetRowAndCell(0, 2).row.dirty, "!page.getRowAndCell(0, 2).row.dirty")
	expect(t, !page.GetRowAndCell(0, 3).row.dirty, "!page.getRowAndCell(0, 3).row.dirty")
	// The page flag is not set by marking a single row.
	expect(t, !page.dirty, "!page.dirty")
	expect(t, p.IsDirty(), "p.isDirty()")
	// Idempotent.
	p.MarkDirty()
	expect(t, page.GetRowAndCell(0, 1).row.dirty, "page.getRowAndCell(0, 1).row.dirty")

	// Row 0 and the last row.
	PageListPin{node: node}.MarkDirty()
	PageListPin{node: node, y: 3, x: 4}.MarkDirty()
	expect(t, page.GetRowAndCell(0, 0).row.dirty, "page.getRowAndCell(0, 0).row.dirty")
	expect(t, !page.GetRowAndCell(0, 2).row.dirty, "!page.getRowAndCell(0, 2).row.dirty")
	expect(t, page.GetRowAndCell(0, 3).row.dirty, "page.getRowAndCell(0, 3).row.dirty")
}

// ---- Cell ----

// Zig: test "term51: Cell.isDirty: row dirty page dirty or clean"
func TestPl6_Cell_isDirty_row_dirty_page_dirty_or_clean(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 5, 4)
	defer s.Deinit()
	page := &s.pages.first.data
	s.ClearDirty()

	c1 := zigUnwrap(t, s.GetCell(Point{Tag: PointTagActive, Active: PointCoordinate{x: 2, y: 1}}))
	c2 := zigUnwrap(t, s.GetCell(Point{Tag: PointTagActive, Active: PointCoordinate{x: 2, y: 2}}))
	expect(t, !c1.IsDirty(), "!c1.isDirty()")
	expect(t, !c2.IsDirty(), "!c2.isDirty()")

	page.GetRowAndCell(0, 1).row.dirty = true
	expect(t, c1.IsDirty(), "c1.isDirty()")
	expect(t, !c2.IsDirty(), "!c2.isDirty()")
	// Another cell in the same row is dirty too.
	expect(t, zigUnwrap(t, s.GetCell(Point{Tag: PointTagActive, Active: PointCoordinate{x: 4, y: 1}})).IsDirty(),
		"s.getCell(.{ .active = .{ .x = 4, .y = 1 } }).?.isDirty()")

	page.GetRowAndCell(0, 1).row.dirty = false
	page.dirty = true
	expect(t, c1.IsDirty(), "c1.isDirty()")
	expect(t, c2.IsDirty(), "c2.isDirty()")
}

// Zig: test "term51: Cell.style: default and non-default style"
func TestPl6_Cell_style_default_and_non_default_style(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 5, 3)
	defer s.Deinit()
	page := &s.pages.first.data

	italic := Style{flags: styleFlags{italic: true}}
	id, err := page.styles.Add(italic)
	zigTry(t, err)
	rac := page.GetRowAndCell(3, 2)
	rac.row.styled = true
	*rac.cell = PageCell{
		content_tag: PageCellContentTagCodepoint,
		content:     PageCellContent{codepoint: 'I'},
		style_id:    id,
	}

	styled := zigUnwrap(t, s.GetCell(Point{Tag: PointTagActive, Active: PointCoordinate{x: 3, y: 2}}))
	expect(t, styled.Style().Eql(italic), "styled.style().eql(italic)")
	expect(t, styled.Style().flags.italic, "styled.style().flags.italic")

	plain := zigUnwrap(t, s.GetCell(Point{Tag: PointTagActive, Active: PointCoordinate{x: 0, y: 0}}))
	expect(t, plain.Style().Default(), "plain.style().default()")
}

// Zig: test "term51: Cell.screenPoint: row and column on a single page"
func TestPl6_Cell_screenPoint_row_and_column_on_a_single_page(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 5, 4)
	defer s.Deinit()

	c := zigUnwrap(t, s.GetCell(Point{Tag: PointTagActive, Active: PointCoordinate{x: 3, y: 2}}))
	expectEqual(t, pl6Pt(3, 2), c.ScreenPoint())
	tl := zigUnwrap(t, s.GetCell(Point{Tag: PointTagScreen}))
	expectEqual(t, pl6Pt(0, 0), tl.ScreenPoint())
	br := zigUnwrap(t, s.GetCell(Point{Tag: PointTagScreen, Screen: PointCoordinate{x: 4, y: 3}}))
	expectEqual(t, pl6Pt(4, 3), br.ScreenPoint())
	// The result tag is always screen, whatever tag was used to get the cell.
	v := zigUnwrap(t, s.GetCell(Point{Tag: PointTagViewport, Viewport: PointCoordinate{x: 1, y: 1}}))
	expect(t, v.ScreenPoint().Tag == PointTagScreen, "v.screenPoint() == .screen")
}

// Zig: test "term51: Cell.screenPoint: adds the rows of all previous pages"
func TestPl6_Cell_screenPoint_adds_the_rows_of_all_previous_pages(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	n1 := uint32(s.pages.first.data.size.rows)
	n2 := uint32(s.pages.first.next.data.size.rows)

	// Page 2 (middle)
	{
		c := zigUnwrap(t, s.GetCell(Point{Tag: PointTagScreen, Screen: PointCoordinate{x: 4, y: n1 + 7}}))
		expect(t, c.node == s.pages.first.next, "c.node == s.pages.first.?.next.?")
		expectEqual(t, uint(7), uint(c.row_idx))
		expectEqual(t, uint(4), uint(c.col_idx))
		expectEqual(t, pl6Pt(4, n1+7), c.ScreenPoint())
	}
	// First row of page 3 and last row of page 2
	{
		c := zigUnwrap(t, s.GetCell(Point{Tag: PointTagScreen, Screen: PointCoordinate{x: 0, y: n1 + n2}}))
		expect(t, c.node == s.pages.last, "c.node == s.pages.last.?")
		expectEqual(t, pl6Pt(0, n1+n2), c.ScreenPoint())
		d := zigUnwrap(t, s.GetCell(Point{Tag: PointTagScreen, Screen: PointCoordinate{x: 9, y: n1 + n2 - 1}}))
		expect(t, d.node == s.pages.first.next, "d.node == s.pages.first.?.next.?")
		expectEqual(t, pl6Pt(9, n1+n2-1), d.ScreenPoint())
	}
}

// ---- Pin: comparison ----

// Zig: test "term51: Pin.eql: node y and x all have to match"
func TestPl6_Pin_eql_node_y_and_x_all_have_to_match(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	first := s.pages.first
	last := s.pages.last

	a := PageListPin{node: first, y: 4, x: 5}
	expect(t, a.Eql(a), "a.eql(a)")
	expect(t, a.Eql(PageListPin{node: first, y: 4, x: 5}), "a.eql(.{ .node = first, .y = 4, .x = 5 })")
	expect(t, !a.Eql(PageListPin{node: last, y: 4, x: 5}), "!a.eql(.{ .node = last, .y = 4, .x = 5 })")
	expect(t, !a.Eql(PageListPin{node: first, y: 3, x: 5}), "!a.eql(.{ .node = first, .y = 3, .x = 5 })")
	expect(t, !a.Eql(PageListPin{node: first, y: 5, x: 5}), "!a.eql(.{ .node = first, .y = 5, .x = 5 })")
	expect(t, !a.Eql(PageListPin{node: first, y: 4, x: 4}), "!a.eql(.{ .node = first, .y = 4, .x = 4 })")
	expect(t, !a.Eql(PageListPin{node: first, y: 4, x: 6}), "!a.eql(.{ .node = first, .y = 4, .x = 6 })")
	// The default y and x are zero.
	expect(t, PageListPin{node: first}.Eql(PageListPin{node: first, y: 0, x: 0}),
		"(Pin{ .node = first }).eql(.{ .node = first, .y = 0, .x = 0 })")
	// The garbage flag does not take part in the comparison.
	expect(t, a.Eql(PageListPin{node: first, y: 4, x: 5, garbage: true}),
		"a.eql(.{ .node = first, .y = 4, .x = 5, .garbage = true })")
	expect(t, PageListPin{node: first, y: 4, x: 5, garbage: true}.Eql(a),
		"(Pin{ .node = first, .y = 4, .x = 5, .garbage = true }).eql(a)")
}

// Zig: test "term51: Pin.before: same page orders by y then x"
func TestPl6_Pin_before_same_page_orders_by_y_then_x(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 5)
	defer s.Deinit()
	node := s.pages.first

	a := PageListPin{node: node, y: 2, x: 3}
	// Equal is not before.
	expect(t, !a.Before(a), "!a.before(a)")
	// x only
	expect(t, a.Before(PageListPin{node: node, y: 2, x: 4}), "a.before(.{ .node = node, .y = 2, .x = 4 })")
	expect(t, !a.Before(PageListPin{node: node, y: 2, x: 2}), "!a.before(.{ .node = node, .y = 2, .x = 2 })")
	// y wins over x in both directions
	expect(t, a.Before(PageListPin{node: node, y: 3, x: 0}), "a.before(.{ .node = node, .y = 3, .x = 0 })")
	expect(t, !a.Before(PageListPin{node: node, y: 1, x: 9}), "!a.before(.{ .node = node, .y = 1, .x = 9 })")
	expect(t, PageListPin{node: node, y: 1, x: 9}.Before(a), "(Pin{ .node = node, .y = 1, .x = 9 }).before(a)")
	expect(t, !PageListPin{node: node, y: 3, x: 0}.Before(a), "!(Pin{ .node = node, .y = 3, .x = 0 }).before(a)")
	// Largest gap
	expect(t, PageListPin{node: node}.Before(PageListPin{node: node, y: 4, x: 9}),
		"(Pin{ .node = node }).before(.{ .node = node, .y = 4, .x = 9 })")
	expect(t, !PageListPin{node: node, y: 4, x: 9}.Before(PageListPin{node: node}),
		"!(Pin{ .node = node, .y = 4, .x = 9 }).before(.{ .node = node })")
}

// Zig: test "term51: Pin.before: different pages follow list order"
func TestPl6_Pin_before_different_pages_follow_list_order(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	n3 := n2.next

	// The y and x are irrelevant when the pages differ.
	hi1 := PageListPin{node: n1, y: n1.data.size.rows - 1, x: 9}
	lo2 := PageListPin{node: n2}
	hi2 := PageListPin{node: n2, y: n2.data.size.rows - 1, x: 9}
	lo3 := PageListPin{node: n3}
	expect(t, hi1.Before(lo2), "hi1.before(lo2)")
	expect(t, !lo2.Before(hi1), "!lo2.before(hi1)")
	// Non adjacent pages
	expect(t, hi1.Before(lo3), "hi1.before(lo3)")
	expect(t, !lo3.Before(hi1), "!lo3.before(hi1)")
	expect(t, PageListPin{node: n1}.Before(hi2), "(Pin{ .node = n1 }).before(hi2)")
	expect(t, hi2.Before(lo3), "hi2.before(lo3)")
	expect(t, !lo3.Before(hi2), "!lo3.before(hi2)")
	expect(t, !hi2.Before(lo2), "!hi2.before(lo2)")
	expect(t, lo2.Before(hi2), "lo2.before(hi2)")
}

// ---- Pin: isBetween ----

// Zig: test "term51: Pin.isBetween: single page y and x boundaries"
func TestPl6_Pin_isBetween_single_page_y_and_x_boundaries(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 10)
	defer s.Deinit()
	node := s.pages.first

	top := PageListPin{node: node, y: 3, x: 2}
	bot := PageListPin{node: node, y: 7, x: 5}
	type Case struct {
		y    uint
		x    uint
		want bool
	}
	cases := []Case{
		// Above the top row
		{y: 0, x: 0, want: false},
		{y: 2, x: 9, want: false},
		// On the top row: x has to be at least the top x.
		{y: 3, x: 0, want: false},
		{y: 3, x: 1, want: false},
		{y: 3, x: 2, want: true},
		{y: 3, x: 3, want: true},
		{y: 3, x: 9, want: true},
		// Strictly between
		{y: 4, x: 0, want: true},
		{y: 5, x: 9, want: true},
		{y: 6, x: 4, want: true},
		// On the bottom row. Upstream quirk pin: when the pin is on the
		// top page below the top row and the bottom is on the same page,
		// only y is compared with the bottom, so cells right of the bottom x
		// are reported inside the range.
		{y: 7, x: 0, want: true},
		{y: 7, x: 5, want: true},
		{y: 7, x: 6, want: true},
		{y: 7, x: 9, want: true},
		// Below the bottom row
		{y: 8, x: 0, want: false},
		{y: 9, x: 9, want: false},
	}
	for _, c := range cases {
		p := PageListPin{node: node, y: SizeCellCountInt(c.y), x: SizeCellCountInt(c.x)}
		expectEqual(t, c.want, p.IsBetween(top, bot))
	}
}

// Zig: test "term51: Pin.isBetween: top and bottom on the same row"
func TestPl6_Pin_isBetween_top_and_bottom_on_the_same_row(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 10)
	defer s.Deinit()
	node := s.pages.first

	top := PageListPin{node: node, y: 4, x: 2}
	bot := PageListPin{node: node, y: 4, x: 6}
	for x := uint(0); x < 10; x++ {
		p := PageListPin{node: node, y: 4, x: SizeCellCountInt(x)}
		expectEqual(t, x >= 2 && x <= 6, p.IsBetween(top, bot))
	}
	expect(t, !PageListPin{node: node, y: 3, x: 4}.IsBetween(top, bot), "!(Pin{ .node = node, .y = 3, .x = 4 }).isBetween(top, bot)")
	expect(t, !PageListPin{node: node, y: 5, x: 4}.IsBetween(top, bot), "!(Pin{ .node = node, .y = 5, .x = 4 }).isBetween(top, bot)")

	// A single cell range only contains itself.
	expect(t, top.IsBetween(top, top), "top.isBetween(top, top)")
	expect(t, !PageListPin{node: node, y: 4, x: 3}.IsBetween(top, top), "!(Pin{ .node = node, .y = 4, .x = 3 }).isBetween(top, top)")
	expect(t, !PageListPin{node: node, y: 4, x: 1}.IsBetween(top, top), "!(Pin{ .node = node, .y = 4, .x = 1 }).isBetween(top, top)")
	expect(t, !PageListPin{node: node, y: 5, x: 2}.IsBetween(top, top), "!(Pin{ .node = node, .y = 5, .x = 2 }).isBetween(top, top)")

	// The first and last cell of the page.
	first := PageListPin{node: node}
	last := PageListPin{node: node, y: 9, x: 9}
	expect(t, first.IsBetween(first, last), "first.isBetween(first, last)")
	expect(t, last.IsBetween(first, last), "last.isBetween(first, last)")
}

// Zig: test "term51: Pin.isBetween: range across pages"
func TestPl6_Pin_isBetween_range_across_pages(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 4)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	n3 := n2.next
	n4 := n3.next

	// Range from the middle of page 2 to the middle of page 3.
	top := PageListPin{node: n2, y: 5, x: 4}
	bot := PageListPin{node: n3, y: 6, x: 3}

	// Page 1 and page 4 are outside.
	expect(t, !PageListPin{node: n1, y: 3}.IsBetween(top, bot), "!(Pin{ .node = n1, .y = 3 }).isBetween(top, bot)")
	expect(t, !PageListPin{node: n1, y: n1.data.size.rows - 1, x: 9}.IsBetween(top, bot),
		"!(Pin{ .node = n1, .y = n1.data.size.rows - 1, .x = 9 }).isBetween(top, bot)")
	expect(t, !PageListPin{node: n4, y: 0}.IsBetween(top, bot), "!(Pin{ .node = n4, .y = 0 }).isBetween(top, bot)")

	// Page 2, the top page: only rows after the top row are in.
	expect(t, !PageListPin{node: n2, y: 4, x: 9}.IsBetween(top, bot), "!(Pin{ .node = n2, .y = 4, .x = 9 }).isBetween(top, bot)")
	expect(t, !PageListPin{node: n2, y: 0, x: 0}.IsBetween(top, bot), "!(Pin{ .node = n2, .y = 0, .x = 0 }).isBetween(top, bot)")
	expect(t, PageListPin{node: n2, y: 6, x: 0}.IsBetween(top, bot), "(Pin{ .node = n2, .y = 6, .x = 0 }).isBetween(top, bot)")
	expect(t, PageListPin{node: n2, y: n2.data.size.rows - 1, x: 9}.IsBetween(top, bot),
		"(Pin{ .node = n2, .y = n2.data.size.rows - 1, .x = 9 }).isBetween(top, bot)")
	// The top row itself, x before the top x is out.
	expect(t, !PageListPin{node: n2, y: 5, x: 3}.IsBetween(top, bot), "!(Pin{ .node = n2, .y = 5, .x = 3 }).isBetween(top, bot)")

	// Page 3, the bottom page: only rows up to the bottom row are in.
	expect(t, PageListPin{node: n3, y: 0, x: 9}.IsBetween(top, bot), "(Pin{ .node = n3, .y = 0, .x = 9 }).isBetween(top, bot)")
	expect(t, PageListPin{node: n3, y: 5, x: 9}.IsBetween(top, bot), "(Pin{ .node = n3, .y = 5, .x = 9 }).isBetween(top, bot)")
	expect(t, PageListPin{node: n3, y: 6, x: 3}.IsBetween(top, bot), "(Pin{ .node = n3, .y = 6, .x = 3 }).isBetween(top, bot)")
	expect(t, !PageListPin{node: n3, y: 6, x: 4}.IsBetween(top, bot), "!(Pin{ .node = n3, .y = 6, .x = 4 }).isBetween(top, bot)")
	expect(t, !PageListPin{node: n3, y: 7, x: 0}.IsBetween(top, bot), "!(Pin{ .node = n3, .y = 7, .x = 0 }).isBetween(top, bot)")

	// Whole pages in between: with 4 pages, a range from page 1 to page 4.
	wide_top := PageListPin{node: n1, y: 1, x: 0}
	wide_bot := PageListPin{node: n4, y: 2, x: 0}
	expect(t, PageListPin{node: n2, y: 0, x: 0}.IsBetween(wide_top, wide_bot), "(Pin{ .node = n2, .y = 0, .x = 0 }).isBetween(wide_top, wide_bot)")
	expect(t, PageListPin{node: n3, y: 9, x: 9}.IsBetween(wide_top, wide_bot), "(Pin{ .node = n3, .y = 9, .x = 9 }).isBetween(wide_top, wide_bot)")
	expect(t, PageListPin{node: n4, y: 2, x: 0}.IsBetween(wide_top, wide_bot), "(Pin{ .node = n4, .y = 2, .x = 0 }).isBetween(wide_top, wide_bot)")
	expect(t, !PageListPin{node: n4, y: 2, x: 1}.IsBetween(wide_top, wide_bot), "!(Pin{ .node = n4, .y = 2, .x = 1 }).isBetween(wide_top, wide_bot)")
	expect(t, !PageListPin{node: n1, y: 0, x: 9}.IsBetween(wide_top, wide_bot), "!(Pin{ .node = n1, .y = 0, .x = 9 }).isBetween(wide_top, wide_bot)")
	// The row after the top row on the top page is in.
	expect(t, PageListPin{node: n1, y: 2, x: 0}.IsBetween(wide_top, wide_bot), "(Pin{ .node = n1, .y = 2, .x = 0 }).isBetween(wide_top, wide_bot)")
}

// Zig: test "term51: Pin.isBetween: the top pin itself with the bottom on a later page"
func TestPl6_Pin_isBetween_the_top_pin_itself_with_the_bottom_on_a_later_page(t *testing.T) {
	// Upstream defect pin: when self is on the top page, on the top row, at
	// or right of the top x, and the bottom is on another page, none of the
	// early returns apply and the page walk starts after the top page, so
	// the pin is reported as outside the range. Even the top pin itself is
	// reported as not between.
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	n3 := n2.next

	top := PageListPin{node: n2, y: 5, x: 4}
	bot := PageListPin{node: n3, y: 6, x: 3}
	expectEqual(t, false, top.IsBetween(top, bot))
	expectEqual(t, false, PageListPin{node: n2, y: 5, x: 9}.IsBetween(top, bot))
	// The row below the top row is in, as is the bottom pin itself.
	expectEqual(t, true, PageListPin{node: n2, y: 6, x: 0}.IsBetween(top, bot))
	expectEqual(t, true, bot.IsBetween(top, bot))

	// Adjacent pages, same behaviour.
	bot2 := PageListPin{node: n3}
	top2 := PageListPin{node: n2, y: n2.data.size.rows - 1, x: 0}
	expectEqual(t, false, top2.IsBetween(top2, bot2))
	expectEqual(t, true, bot2.IsBetween(top2, bot2))
}

// ---- Pin: column movement ----

// Zig: test "term51: Pin.left: moves the x left and keeps everything else"
func TestPl6_Pin_left_moves_the_x_left_and_keeps_everything_else(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 3)
	defer s.Deinit()
	node := s.pages.first

	p := PageListPin{node: node, y: 2, x: 7, garbage: true}
	pl6ExpectPinV(t, p.Left(0), node, 2, 7)
	pl6ExpectPinV(t, p.Left(1), node, 2, 6)
	pl6ExpectPinV(t, p.Left(3), node, 2, 4)
	pl6ExpectPinV(t, p.Left(7), node, 2, 0)
	expect(t, p.Left(3).garbage, "p.left(3).garbage")
	expect(t, p.Left(0).Eql(p), "p.left(0).eql(p)")

	// x of zero can only move by zero.
	z := PageListPin{node: node, y: 1}
	pl6ExpectPinV(t, z.Left(0), node, 1, 0)

	// The largest x
	e := PageListPin{node: node, y: 0, x: 9}
	pl6ExpectPinV(t, e.Left(9), node, 0, 0)
	pl6ExpectPinV(t, e.Left(1), node, 0, 8)
}

// Zig: test "term51: Pin.right: moves the x right and keeps everything else"
func TestPl6_Pin_right_moves_the_x_right_and_keeps_everything_else(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 3)
	defer s.Deinit()
	node := s.pages.first

	p := PageListPin{node: node, y: 1, x: 2, garbage: true}
	pl6ExpectPinV(t, p.Right(0), node, 1, 2)
	pl6ExpectPinV(t, p.Right(1), node, 1, 3)
	pl6ExpectPinV(t, p.Right(5), node, 1, 7)
	// To the last column exactly.
	pl6ExpectPinV(t, p.Right(7), node, 1, 9)
	expect(t, p.Right(7).garbage, "p.right(7).garbage")

	z := PageListPin{node: node}
	pl6ExpectPinV(t, z.Right(9), node, 0, 9)
	e := PageListPin{node: node, y: 2, x: 9}
	pl6ExpectPinV(t, e.Right(0), node, 2, 9)
}

// Zig: test "term51: Pin.leftClamp: stops at the first column"
func TestPl6_Pin_leftClamp_stops_at_the_first_column(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 3)
	defer s.Deinit()
	node := s.pages.first

	p := PageListPin{node: node, y: 2, x: 5, garbage: true}
	pl6ExpectPinV(t, p.LeftClamp(0), node, 2, 5)
	pl6ExpectPinV(t, p.LeftClamp(1), node, 2, 4)
	pl6ExpectPinV(t, p.LeftClamp(4), node, 2, 1)
	pl6ExpectPinV(t, p.LeftClamp(5), node, 2, 0)
	pl6ExpectPinV(t, p.LeftClamp(6), node, 2, 0)
	pl6ExpectPinV(t, p.LeftClamp(math.MaxUint16), node, 2, 0)
	expect(t, p.LeftClamp(100).garbage, "p.leftClamp(100).garbage")

	z := PageListPin{node: node, y: 1}
	pl6ExpectPinV(t, z.LeftClamp(0), node, 1, 0)
	pl6ExpectPinV(t, z.LeftClamp(3), node, 1, 0)
}

// Zig: test "term51: Pin.rightClamp: stops at the last column"
func TestPl6_Pin_rightClamp_stops_at_the_last_column(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 3)
	defer s.Deinit()
	node := s.pages.first

	p := PageListPin{node: node, y: 2, x: 5, garbage: true}
	pl6ExpectPinV(t, p.RightClamp(0), node, 2, 5)
	pl6ExpectPinV(t, p.RightClamp(1), node, 2, 6)
	pl6ExpectPinV(t, p.RightClamp(3), node, 2, 8)
	pl6ExpectPinV(t, p.RightClamp(4), node, 2, 9)
	pl6ExpectPinV(t, p.RightClamp(5), node, 2, 9)
	pl6ExpectPinV(t, p.RightClamp(6), node, 2, 9)
	// The addition saturates instead of overflowing.
	pl6ExpectPinV(t, p.RightClamp(math.MaxUint16), node, 2, 9)
	expect(t, p.RightClamp(100).garbage, "p.rightClamp(100).garbage")

	e := PageListPin{node: node, y: 0, x: 9}
	pl6ExpectPinV(t, e.RightClamp(0), node, 0, 9)
	pl6ExpectPinV(t, e.RightClamp(1), node, 0, 9)
	z := PageListPin{node: node}
	pl6ExpectPinV(t, z.RightClamp(9), node, 0, 9)
	pl6ExpectPinV(t, z.RightClamp(8), node, 0, 8)
}

// Zig: test "term51: Pin.leftWrap: within the row and wrapping to previous rows"
func TestPl6_Pin_leftWrap_within_the_row_and_wrapping_to_previous_rows(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 8)
	defer s.Deinit()
	node := s.pages.first

	p := PageListPin{node: node, y: 5, x: 3}
	// Fits in the row: same as left.
	pl6ExpectPin(t, p.LeftWrap(0), node, 5, 3)
	pl6ExpectPin(t, p.LeftWrap(1), node, 5, 2)
	pl6ExpectPin(t, p.LeftWrap(3), node, 5, 0)
	// One past the row start wraps to the last column of the row above.
	pl6ExpectPin(t, p.LeftWrap(4), node, 4, 9)
	pl6ExpectPin(t, p.LeftWrap(5), node, 4, 8)
	pl6ExpectPin(t, p.LeftWrap(12), node, 4, 1)
	pl6ExpectPin(t, p.LeftWrap(14), node, 3, 9)
	pl6ExpectPin(t, p.LeftWrap(22), node, 3, 1)
	pl6ExpectPin(t, p.LeftWrap(24), node, 2, 9)
	// The last count that still fits above the top of the page: 5 rows up.
	pl6ExpectPin(t, p.LeftWrap(52), node, 0, 1)
	// One more row would be above the top.
	expect(t, !p.LeftWrap(53).Valid, "p.leftWrap(53) == null")
	expect(t, !p.LeftWrap(1000).Valid, "p.leftWrap(1000) == null")
}

// Zig: test "term51: Pin.leftWrap: landing exactly on the first column of a row is off by a row"
func TestPl6_Pin_leftWrap_landing_exactly_on_the_first_column_of_a_row_is_off_by_a_row(t *testing.T) {
	// Upstream defect pin: when the remaining count after the current row is
	// a multiple of cols, x is computed as cols (one past the last column)
	// and the pin is one row further up than the cell the offset describes.
	testStdCapacity(t)
	s := pl6Init(t, 10, 8)
	defer s.Deinit()
	node := s.pages.first

	p := PageListPin{node: node, y: 5, x: 3}
	// 3 + 10 cells left would be (0, 4); upstream returns (10, 3).
	pl6ExpectPin(t, p.LeftWrap(13), node, 3, 10)
	pl6ExpectPin(t, p.LeftWrap(23), node, 2, 10)
	pl6ExpectPin(t, p.LeftWrap(33), node, 1, 10)

	// From the first column: cols cells left is (0, y-1); upstream gives (10, y-2).
	z := PageListPin{node: node, y: 5, x: 0}
	pl6ExpectPin(t, z.LeftWrap(10), node, 3, 10)
	// But one cell and cols-1 cells are right.
	pl6ExpectPin(t, z.LeftWrap(1), node, 4, 9)
	pl6ExpectPin(t, z.LeftWrap(9), node, 4, 1)
	pl6ExpectPin(t, z.LeftWrap(11), node, 3, 9)
}

// Zig: test "term51: Pin.leftWrap: top of the screen and page crossing"
func TestPl6_Pin_leftWrap_top_of_the_screen_and_page_crossing(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next

	// First row of the first page.
	top := PageListPin{node: n1, y: 0, x: 3}
	pl6ExpectPin(t, top.LeftWrap(0), n1, 0, 3)
	pl6ExpectPin(t, top.LeftWrap(3), n1, 0, 0)
	expect(t, !top.LeftWrap(4).Valid, "top.leftWrap(4) == null")
	expect(t, !PageListPin{node: n1}.LeftWrap(1).Valid, "(Pin{ .node = n1 }).leftWrap(1) == null")
	pl6ExpectPin(t, PageListPin{node: n1}.LeftWrap(0), n1, 0, 0)

	// Wrapping from the first row of the second page goes to the last row of
	// the first page.
	p := PageListPin{node: n2, y: 0, x: 2}
	pl6ExpectPin(t, p.LeftWrap(2), n2, 0, 0)
	pl6ExpectPin(t, p.LeftWrap(3), n1, uint(n1.data.size.rows)-1, 9)
	pl6ExpectPin(t, p.LeftWrap(5), n1, uint(n1.data.size.rows)-1, 7)
	pl6ExpectPin(t, p.LeftWrap(13), n1, uint(n1.data.size.rows)-2, 9)
	// Wrapping from a later row of the second page that stays on that page.
	q := PageListPin{node: n2, y: 2, x: 2}
	pl6ExpectPin(t, q.LeftWrap(3), n2, 1, 9)
	// Wrapping from a later row that crosses into the first page.
	pl6ExpectPin(t, q.LeftWrap(23), n1, uint(n1.data.size.rows)-1, 9)
}

// Zig: test "term51: Pin.rightWrap: within the row and wrapping to next rows"
func TestPl6_Pin_rightWrap_within_the_row_and_wrapping_to_next_rows(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 8)
	defer s.Deinit()
	node := s.pages.first

	p := PageListPin{node: node, y: 2, x: 3}
	// Fits in the row: same as right.
	pl6ExpectPin(t, p.RightWrap(0), node, 2, 3)
	pl6ExpectPin(t, p.RightWrap(1), node, 2, 4)
	pl6ExpectPin(t, p.RightWrap(6), node, 2, 9)
	// One past the row end wraps to the first column of the row below.
	pl6ExpectPin(t, p.RightWrap(7), node, 3, 0)
	pl6ExpectPin(t, p.RightWrap(8), node, 3, 1)
	pl6ExpectPin(t, p.RightWrap(15), node, 3, 8)
	// Two rows down
	pl6ExpectPin(t, p.RightWrap(17), node, 4, 0)
	pl6ExpectPin(t, p.RightWrap(25), node, 4, 8)
	// From the last column.
	e := PageListPin{node: node, y: 2, x: 9}
	pl6ExpectPin(t, e.RightWrap(0), node, 2, 9)
	pl6ExpectPin(t, e.RightWrap(1), node, 3, 0)
	pl6ExpectPin(t, e.RightWrap(2), node, 3, 1)
	pl6ExpectPin(t, e.RightWrap(9), node, 3, 8)
	pl6ExpectPin(t, e.RightWrap(11), node, 4, 0)
	// From the first column.
	z := PageListPin{node: node, y: 0, x: 0}
	pl6ExpectPin(t, z.RightWrap(9), node, 0, 9)
	pl6ExpectPin(t, z.RightWrap(10), node, 1, 0)
	// The last count that still fits below: 5 rows down from y=2 is y=7.
	pl6ExpectPin(t, p.RightWrap(55), node, 7, 8)
	// One more row would be past the bottom of the page list.
	expect(t, !p.RightWrap(56).Valid, "p.rightWrap(56) == null")
	expect(t, !p.RightWrap(1000).Valid, "p.rightWrap(1000) == null")
}

// Zig: test "term51: Pin.rightWrap: bottom of the screen and page crossing"
func TestPl6_Pin_rightWrap_bottom_of_the_screen_and_page_crossing(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	last_y := n2.data.size.rows - 1

	// Last cell of the page list.
	bottom := PageListPin{node: n2, y: last_y, x: 9}
	pl6ExpectPin(t, bottom.RightWrap(0), n2, uint(last_y), 9)
	expect(t, !bottom.RightWrap(1).Valid, "bottom.rightWrap(1) == null")
	// Overflow is detected before any x computation, so even counts that
	// would land on the last column of a row return null.
	expect(t, !bottom.RightWrap(10).Valid, "bottom.rightWrap(10) == null")
	expect(t, !bottom.RightWrap(11).Valid, "bottom.rightWrap(11) == null")
	expect(t, !bottom.RightWrap(20).Valid, "bottom.rightWrap(20) == null")
	mid := PageListPin{node: n2, y: last_y, x: 3}
	pl6ExpectPin(t, mid.RightWrap(6), n2, uint(last_y), 9)
	expect(t, !mid.RightWrap(7).Valid, "mid.rightWrap(7) == null")
	expect(t, !mid.RightWrap(16).Valid, "mid.rightWrap(16) == null")

	// Wrapping from the last row of the first page goes to the second page.
	p := PageListPin{node: n1, y: n1.data.size.rows - 1, x: 8}
	pl6ExpectPin(t, p.RightWrap(1), n1, uint(n1.data.size.rows)-1, 9)
	pl6ExpectPin(t, p.RightWrap(2), n2, 0, 0)
	pl6ExpectPin(t, p.RightWrap(3), n2, 0, 1)
	pl6ExpectPin(t, p.RightWrap(12), n2, 1, 0)
}

// ---- Pin: row movement ----

// Zig: test "term51: Pin.downOverflow: offsets within and across pages"
func TestPl6_Pin_downOverflow_offsets_within_and_across_pages(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	n3 := n2.next
	r1 := uint(n1.data.size.rows)
	r2 := uint(n2.data.size.rows)
	r3 := uint(n3.data.size.rows)

	p := PageListPin{node: n1, y: 2, x: 4, garbage: true}
	// Rows left on the pin's page below the pin.
	below := r1 - 1 - 2

	// Zero keeps the position, but the garbage flag is not carried.
	{
		r := p.DownOverflow(0)
		pl6ExpectPinV(t, pl6Offset(t, r), n1, 2, 4)
		expect(t, !r.Offset.garbage, "!r.offset.garbage")
	}
	pl6ExpectPinV(t, pl6Offset(t, p.DownOverflow(1)), n1, 3, 4)
	// Last row of the page.
	pl6ExpectPinV(t, pl6Offset(t, p.DownOverflow(below)), n1, r1-1, 4)
	// First row of the next page.
	pl6ExpectPinV(t, pl6Offset(t, p.DownOverflow(below+1)), n2, 0, 4)
	pl6ExpectPinV(t, pl6Offset(t, p.DownOverflow(below+2)), n2, 1, 4)
	// Last row of the next page.
	pl6ExpectPinV(t, pl6Offset(t, p.DownOverflow(below+r2)), n2, r2-1, 4)
	// First row of the page after, skipping the whole middle page.
	pl6ExpectPinV(t, pl6Offset(t, p.DownOverflow(below+r2+1)), n3, 0, 4)
	// Last row of the last page.
	pl6ExpectPinV(t, pl6Offset(t, p.DownOverflow(below+r2+r3)), n3, r3-1, 4)

	// One past the end: overflow by exactly one, ending on the last row.
	{
		r := pl6Overflow(t, p.DownOverflow(below+r2+r3+1)).Overflow
		expect(t, r.end.node == n3, "r.end.node == n3")
		expectEqual(t, r3-1, uint(r.end.y))
		expectEqual(t, uint(4), uint(r.end.x))
		expectEqual(t, uint(1), r.remaining)
	}
	{
		r := pl6Overflow(t, p.DownOverflow(below+r2+r3+100)).Overflow
		expect(t, r.end.node == n3, "r.end.node == n3")
		expectEqual(t, r3-1, uint(r.end.y))
		expectEqual(t, uint(100), r.remaining)
	}
}

// Zig: test "term51: Pin.downOverflow: single page"
func TestPl6_Pin_downOverflow_single_page(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 6)
	defer s.Deinit()
	node := s.pages.first

	p := PageListPin{node: node, y: 4, x: 7}
	pl6ExpectPinV(t, pl6Offset(t, p.DownOverflow(0)), node, 4, 7)
	pl6ExpectPinV(t, pl6Offset(t, p.DownOverflow(1)), node, 5, 7)
	{
		r := pl6Overflow(t, p.DownOverflow(2)).Overflow
		expect(t, r.end.node == node, "r.end.node == node")
		expectEqual(t, uint(5), uint(r.end.y))
		expectEqual(t, uint(7), uint(r.end.x))
		expectEqual(t, uint(1), r.remaining)
	}

	// On the last row: zero is fine, anything else overflows by n.
	last := PageListPin{node: node, y: 5, x: 0}
	pl6ExpectPinV(t, pl6Offset(t, last.DownOverflow(0)), node, 5, 0)
	expectEqual(t, uint(1), pl6Overflow(t, last.DownOverflow(1)).Overflow.remaining)
	expectEqual(t, uint(9), pl6Overflow(t, last.DownOverflow(9)).Overflow.remaining)

	// From the first row to the last
	first := PageListPin{node: node}
	pl6ExpectPinV(t, pl6Offset(t, first.DownOverflow(5)), node, 5, 0)
	expectEqual(t, uint(1), pl6Overflow(t, first.DownOverflow(6)).Overflow.remaining)
}

// Zig: test "term51: Pin.upOverflow: offsets within and across pages"
func TestPl6_Pin_upOverflow_offsets_within_and_across_pages(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	n3 := n2.next
	r1 := uint(n1.data.size.rows)
	r2 := uint(n2.data.size.rows)

	p := PageListPin{node: n3, y: 4, x: 6, garbage: true}

	// Zero keeps the position, but the garbage flag is not carried.
	{
		r := p.UpOverflow(0)
		pl6ExpectPinV(t, pl6Offset(t, r), n3, 4, 6)
		expect(t, !r.Offset.garbage, "!r.offset.garbage")
	}
	pl6ExpectPinV(t, pl6Offset(t, p.UpOverflow(1)), n3, 3, 6)
	// First row of the page.
	pl6ExpectPinV(t, pl6Offset(t, p.UpOverflow(4)), n3, 0, 6)
	// Last row of the previous page.
	pl6ExpectPinV(t, pl6Offset(t, p.UpOverflow(5)), n2, r2-1, 6)
	pl6ExpectPinV(t, pl6Offset(t, p.UpOverflow(6)), n2, r2-2, 6)
	// First row of the previous page.
	pl6ExpectPinV(t, pl6Offset(t, p.UpOverflow(4+r2)), n2, 0, 6)
	// Last row of the page before that, skipping the whole middle page.
	pl6ExpectPinV(t, pl6Offset(t, p.UpOverflow(4+r2+1)), n1, r1-1, 6)
	// First row of the first page.
	pl6ExpectPinV(t, pl6Offset(t, p.UpOverflow(4+r2+r1)), n1, 0, 6)

	// One past the top: overflow by exactly one, ending on row zero.
	{
		r := pl6Overflow(t, p.UpOverflow(4+r2+r1+1)).Overflow
		expect(t, r.end.node == n1, "r.end.node == n1")
		expectEqual(t, uint(0), uint(r.end.y))
		expectEqual(t, uint(6), uint(r.end.x))
		expectEqual(t, uint(1), r.remaining)
	}
	{
		r := pl6Overflow(t, p.UpOverflow(4+r2+r1+100)).Overflow
		expect(t, r.end.node == n1, "r.end.node == n1")
		expectEqual(t, uint(0), uint(r.end.y))
		expectEqual(t, uint(100), r.remaining)
	}
}

// Zig: test "term51: Pin.upOverflow: single page"
func TestPl6_Pin_upOverflow_single_page(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 6)
	defer s.Deinit()
	node := s.pages.first

	p := PageListPin{node: node, y: 1, x: 7}
	pl6ExpectPinV(t, pl6Offset(t, p.UpOverflow(0)), node, 1, 7)
	pl6ExpectPinV(t, pl6Offset(t, p.UpOverflow(1)), node, 0, 7)
	{
		r := pl6Overflow(t, p.UpOverflow(2)).Overflow
		expect(t, r.end.node == node, "r.end.node == node")
		expectEqual(t, uint(0), uint(r.end.y))
		expectEqual(t, uint(7), uint(r.end.x))
		expectEqual(t, uint(1), r.remaining)
	}

	// On the first row: zero is fine, anything else overflows by n.
	first := PageListPin{node: node, y: 0, x: 3}
	pl6ExpectPinV(t, pl6Offset(t, first.UpOverflow(0)), node, 0, 3)
	expectEqual(t, uint(1), pl6Overflow(t, first.UpOverflow(1)).Overflow.remaining)
	expectEqual(t, uint(9), pl6Overflow(t, first.UpOverflow(9)).Overflow.remaining)

	// From the last row to the first
	last := PageListPin{node: node, y: 5}
	pl6ExpectPinV(t, pl6Offset(t, last.UpOverflow(5)), node, 0, 0)
	expectEqual(t, uint(1), pl6Overflow(t, last.UpOverflow(6)).Overflow.remaining)
}

// Zig: test "term51: Pin.down: moves down or returns null past the end"
func TestPl6_Pin_down_moves_down_or_returns_null_past_the_end(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	r1 := uint(n1.data.size.rows)
	r2 := uint(n2.data.size.rows)

	p := PageListPin{node: n1, y: 2, x: 4, garbage: true}
	pl6ExpectPin(t, p.Down(0), n1, 2, 4)
	expect(t, !p.Down(0).Value.garbage, "!p.down(0).?.garbage")
	pl6ExpectPin(t, p.Down(1), n1, 3, 4)
	pl6ExpectPin(t, p.Down(r1-1-2), n1, r1-1, 4)
	pl6ExpectPin(t, p.Down(r1-2), n2, 0, 4)
	// Last row of the page list.
	pl6ExpectPin(t, p.Down(r1-3+r2), n2, r2-1, 4)
	// One past
	expect(t, !p.Down(r1-3+r2+1).Valid, "p.down(r1 - 3 + r2 + 1) == null")
	expect(t, !p.Down(1_000_000).Valid, "p.down(1_000_000) == null")

	// From the very last row
	last := PageListPin{node: n2, y: SizeCellCountInt(r2 - 1), x: 9}
	pl6ExpectPin(t, last.Down(0), n2, r2-1, 9)
	expect(t, !last.Down(1).Valid, "last.down(1) == null")
}

// Zig: test "term51: Pin.up: moves up or returns null past the start"
func TestPl6_Pin_up_moves_up_or_returns_null_past_the_start(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	r1 := uint(n1.data.size.rows)

	p := PageListPin{node: n2, y: 2, x: 4, garbage: true}
	pl6ExpectPin(t, p.Up(0), n2, 2, 4)
	expect(t, !p.Up(0).Value.garbage, "!p.up(0).?.garbage")
	pl6ExpectPin(t, p.Up(1), n2, 1, 4)
	pl6ExpectPin(t, p.Up(2), n2, 0, 4)
	pl6ExpectPin(t, p.Up(3), n1, r1-1, 4)
	// First row of the page list.
	pl6ExpectPin(t, p.Up(2+r1), n1, 0, 4)
	// One past
	expect(t, !p.Up(2+r1+1).Valid, "p.up(2 + r1 + 1) == null")
	expect(t, !p.Up(1_000_000).Valid, "p.up(1_000_000) == null")

	// From the very first row
	first := PageListPin{node: n1, y: 0, x: 9}
	pl6ExpectPin(t, first.Up(0), n1, 0, 9)
	expect(t, !first.Up(1).Valid, "first.up(1) == null")
}

// ---- PageList: getTopLeft, getBottomRight, counts, growRows, dirty ----

// Zig: test "term51: getTopLeft: screen and history are the origin of the first page"
func TestPl6_getTopLeft_screen_and_history_are_the_origin_of_the_first_page(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	first := s.pages.first

	pl6ExpectPinV(t, s.GetTopLeft(PointTagScreen), first, 0, 0)
	pl6ExpectPinV(t, s.GetTopLeft(PointTagHistory), first, 0, 0)

	// Not affected by the viewport.
	s.Scroll(PageListScroll{Tag: PageListScrollRow, Row: uint(first.data.size.rows) + 5})
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	pl6ExpectPinV(t, s.GetTopLeft(PointTagScreen), first, 0, 0)
	pl6ExpectPinV(t, s.GetTopLeft(PointTagHistory), first, 0, 0)
}

// Zig: test "term51: getTopLeft: active on a single page"
func TestPl6_getTopLeft_active_on_a_single_page(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 5)
	defer s.Deinit()
	first := s.pages.first

	pl6ExpectPinV(t, s.GetTopLeft(PointTagActive), first, 0, 0)
	pl6GrowRows(t, &s, 1)
	pl6ExpectPinV(t, s.GetTopLeft(PointTagActive), first, 1, 0)
	pl6GrowRows(t, &s, 6)
	pl6ExpectPinV(t, s.GetTopLeft(PointTagActive), first, 7, 0)
	expectEqual(t, uint(12), uint(first.data.size.rows))
}

// Zig: test "term51: getTopLeft: active crossing page boundaries"
func TestPl6_getTopLeft_active_crossing_page_boundaries(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	r2 := uint(n2.data.size.rows)
	expect(t, n2.next == nil, "n2.next == null")
	// The active area is the last 3 rows of the full second page.
	pl6ExpectPinV(t, s.GetTopLeft(PointTagActive), n2, r2-3, 0)

	// One row on a new page: active covers 2 rows of page 2 and 1 of page 3.
	pl6GrowRows(t, &s, 1)
	n3 := n2.next
	pl6ExpectPinV(t, s.GetTopLeft(PointTagActive), n2, r2-2, 0)
	// Two rows on the new page.
	pl6GrowRows(t, &s, 1)
	pl6ExpectPinV(t, s.GetTopLeft(PointTagActive), n2, r2-1, 0)
	// Exactly the active rows on the new page.
	pl6GrowRows(t, &s, 1)
	expectEqual(t, uint(3), uint(n3.data.size.rows))
	pl6ExpectPinV(t, s.GetTopLeft(PointTagActive), n3, 0, 0)
	// More rows than active on the new page.
	pl6GrowRows(t, &s, 1)
	pl6ExpectPinV(t, s.GetTopLeft(PointTagActive), n3, 1, 0)
}

// Zig: test "term51: getTopLeft: active spanning more than two pages from init"
func TestPl6_getTopLeft_active_spanning_more_than_two_pages_from_init(t *testing.T) {
	testStdCapacity(t)
	// Rows that do not fit in a single page for 80 columns.
	cap_rows := pageListInitialCapacity(80).rows
	s := pl6Init(t, 80, cap_rows+10)
	defer s.Deinit()
	first := s.pages.first
	last := s.pages.last
	expect(t, first != last, "first != last")
	expectEqual(t, uint(cap_rows), uint(first.data.size.rows))
	expectEqual(t, uint(10), uint(last.data.size.rows))

	// The active area is the full first page plus the 10 rows of the last.
	pl6ExpectPinV(t, s.GetTopLeft(PointTagActive), first, 0, 0)
	pl6GrowRows(t, &s, 5)
	expectEqual(t, uint(15), uint(last.data.size.rows))
	pl6ExpectPinV(t, s.GetTopLeft(PointTagActive), first, 5, 0)
}

// Zig: test "term51: getTopLeft: viewport follows active top and pin"
func TestPl6_getTopLeft_viewport_follows_active_top_and_pin(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	r1 := uint(n1.data.size.rows)

	// Active
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	expect(t, s.GetTopLeft(PointTagViewport).Eql(s.GetTopLeft(PointTagActive)), "s.getTopLeft(.viewport).eql(s.getTopLeft(.active))")

	// Top
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
	pl6ExpectPinV(t, s.GetTopLeft(PointTagViewport), n1, 0, 0)
	// Other tags are not affected.
	expect(t, !s.GetTopLeft(PointTagActive).Eql(s.GetTopLeft(PointTagViewport)), "!s.getTopLeft(.active).eql(s.getTopLeft(.viewport))")

	// Pin: the viewport pin itself, including its x.
	s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: PageListPin{node: n2, y: 5, x: 3}})
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	pl6ExpectPinV(t, s.GetTopLeft(PointTagViewport), n2, 5, 3)
	expect(t, s.GetTopLeft(PointTagViewport).Eql(*s.viewport_pin), "s.getTopLeft(.viewport).eql(s.viewport_pin.*)")

	// Pin set by row
	s.Scroll(PageListScroll{Tag: PageListScrollRow, Row: r1 - 2})
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	pl6ExpectPinV(t, s.GetTopLeft(PointTagViewport), n1, r1-2, 0)

	// Back to active
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	expect(t, s.GetTopLeft(PointTagViewport).Eql(s.GetTopLeft(PointTagActive)), "s.getTopLeft(.viewport).eql(s.getTopLeft(.active))")
}

// Zig: test "term51: getBottomRight: screen and active are the last cell of the last page"
func TestPl6_getBottomRight_screen_and_active_are_the_last_cell_of_the_last_page(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 5)
	defer s.Deinit()
	first := s.pages.first

	pl6ExpectPin(t, s.GetBottomRight(PointTagScreen), first, 4, 9)
	pl6ExpectPin(t, s.GetBottomRight(PointTagActive), first, 4, 9)
	pl6GrowRows(t, &s, 3)
	pl6ExpectPin(t, s.GetBottomRight(PointTagScreen), first, 7, 9)
	pl6ExpectPin(t, s.GetBottomRight(PointTagActive), first, 7, 9)
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	pl6ExpectPin(t, s.GetBottomRight(PointTagScreen), first, 7, 9)
	pl6ExpectPin(t, s.GetBottomRight(PointTagActive), first, 7, 9)

	// Multi page, with a single row in the last page.
	m := pl6Pages(t, 10, 3, 2)
	defer m.Deinit()
	pl6GrowRows(t, &m, 1)
	last := m.pages.last
	expectEqual(t, uint(1), uint(last.data.size.rows))
	pl6ExpectPin(t, m.GetBottomRight(PointTagScreen), last, 0, 9)
	pl6ExpectPin(t, m.GetBottomRight(PointTagActive), last, 0, 9)
}

// Zig: test "term51: getBottomRight: viewport is rows below the viewport top left"
func TestPl6_getBottomRight_viewport_is_rows_below_the_viewport_top_left(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	n3 := n2.next
	r1 := uint(n1.data.size.rows)
	r3 := uint(n3.data.size.rows)

	// Active viewport: the last row of the last page.
	pl6ExpectPin(t, s.GetBottomRight(PointTagViewport), n3, r3-1, 9)

	// Top viewport: 3 rows from the first page.
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	pl6ExpectPin(t, s.GetBottomRight(PointTagViewport), n1, 2, 9)

	// Pin viewport inside a page.
	s.Scroll(PageListScroll{Tag: PageListScrollRow, Row: 10})
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	pl6ExpectPin(t, s.GetBottomRight(PointTagViewport), n1, 12, 9)

	// Pin viewport that crosses a page boundary: the x is the last column
	// even though the viewport pin x is not.
	s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: PageListPin{node: n1, y: SizeCellCountInt(r1 - 2), x: 4}})
	pl6ExpectPin(t, s.GetBottomRight(PointTagViewport), n2, 0, 9)
	s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: PageListPin{node: n1, y: SizeCellCountInt(r1 - 1), x: 4}})
	pl6ExpectPin(t, s.GetBottomRight(PointTagViewport), n2, 1, 9)
}

// Zig: test "term51: getBottomRight: history is null until there is history"
func TestPl6_getBottomRight_history_is_null_until_there_is_history(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 5)
	defer s.Deinit()
	first := s.pages.first

	expect(t, !s.GetBottomRight(PointTagHistory).Valid, "s.getBottomRight(.history) == null")
	pl6GrowRows(t, &s, 1)
	pl6ExpectPin(t, s.GetBottomRight(PointTagHistory), first, 0, 9)
	pl6GrowRows(t, &s, 4)
	pl6ExpectPin(t, s.GetBottomRight(PointTagHistory), first, 4, 9)

	// History that ends on the previous page.
	m := pl6Pages(t, 10, 3, 2)
	defer m.Deinit()
	n1 := m.pages.first
	n2 := n1.next
	// Active is the last three rows of page 2, history is everything above.
	pl6ExpectPin(t, m.GetBottomRight(PointTagHistory), n2, uint(n2.data.size.rows)-4, 9)
	pl6GrowRows(t, &m, 3)
	n3 := n2.next
	expectEqual(t, uint(3), uint(n3.data.size.rows))
	pl6ExpectPin(t, m.GetBottomRight(PointTagHistory), n2, uint(n2.data.size.rows)-1, 9)
}

// Zig: test "term51: totalRows: counts the rows of all pages"
func TestPl6_totalRows_counts_the_rows_of_all_pages(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 5)
	defer s.Deinit()
	expectEqual(t, uint(5), s.totalRows())
	pl6GrowRows(t, &s, 3)
	expectEqual(t, uint(8), s.totalRows())
	expectEqual(t, s.total_rows, s.totalRows())

	m := pl6Pages(t, 10, 3, 3)
	defer m.Deinit()
	n1 := m.pages.first
	n2 := n1.next
	n3 := n2.next
	want := uint(n1.data.size.rows) + uint(n2.data.size.rows) + uint(n3.data.size.rows)
	expectEqual(t, want, m.totalRows())
	expectEqual(t, m.total_rows, m.totalRows())
	pl6GrowRows(t, &m, 1)
	expectEqual(t, want+1, m.totalRows())

	// More rows than a single page can hold at init.
	cap_rows := pageListInitialCapacity(80).rows
	big := pl6Init(t, 80, cap_rows+10)
	defer big.Deinit()
	expectEqual(t, uint(cap_rows)+10, big.totalRows())
}

// Zig: test "term51: totalPages: counts the pages"
func TestPl6_totalPages_counts_the_pages(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 5)
	defer s.Deinit()
	expectEqual(t, uint(1), s.TotalPages())
	pl6GrowRows(t, &s, 10)
	expectEqual(t, uint(1), s.TotalPages())

	m := pl6Pages(t, 10, 3, 3)
	defer m.Deinit()
	expectEqual(t, uint(3), m.TotalPages())
	// Still 3: the last page was exactly full, the next row makes it 4.
	pl6GrowRows(t, &m, 1)
	expectEqual(t, uint(4), m.TotalPages())
	pl6GrowRows(t, &m, 5)
	expectEqual(t, uint(4), m.TotalPages())

	// More rows than a single page can hold at init.
	cap_rows := pageListInitialCapacity(80).rows
	big := pl6Init(t, 80, cap_rows+10)
	defer big.Deinit()
	expectEqual(t, uint(2), big.TotalPages())
}

// Zig: test "term51: growRows: adds rows and allocates pages when full"
func TestPl6_growRows_adds_rows_and_allocates_pages_when_full(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 80, 24)
	defer s.Deinit()
	first := s.pages.first
	cap := uint(first.data.capacity.rows)

	// Zero is a no-op.
	pl6GrowRows(t, &s, 0)
	expectEqual(t, uint(24), s.totalRows())
	expectEqual(t, uint(24), s.total_rows)
	expectEqual(t, uint(1), s.TotalPages())

	// Rows are added to the existing page.
	pl6GrowRows(t, &s, 1)
	expectEqual(t, uint(25), s.totalRows())
	expectEqual(t, uint(25), s.total_rows)
	expectEqual(t, uint(25), uint(first.data.size.rows))
	pl6GrowRows(t, &s, 9)
	expectEqual(t, uint(34), s.totalRows())
	expectEqual(t, uint(1), s.TotalPages())

	// Fill the page exactly.
	first.data.PauseIntegrityChecks(true)
	pl6GrowRows(t, &s, cap-34)
	first.data.PauseIntegrityChecks(false)
	expectEqual(t, cap, s.totalRows())
	expectEqual(t, cap, uint(first.data.size.rows))
	expectEqual(t, uint(1), s.TotalPages())

	// One more row goes to a new page.
	pl6GrowRows(t, &s, 1)
	expectEqual(t, cap+1, s.totalRows())
	expectEqual(t, cap+1, s.total_rows)
	expectEqual(t, uint(2), s.TotalPages())
	expectEqual(t, uint(1), uint(s.pages.last.data.size.rows))
	expectEqual(t, cap, uint(first.data.size.rows))

	// The active area is now across the two pages.
	pl6GrowRows(t, &s, 2)
	expectEqual(t, uint(3), uint(s.pages.last.data.size.rows))
	expectEqual(t, uint(2), s.TotalPages())
}

// Zig: test "term51: growRows: rows in a new page start blank"
func TestPl6_growRows_rows_in_a_new_page_start_blank(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	pl6GrowRows(t, &s, 2)
	last := s.pages.last
	expectEqual(t, uint(2), uint(last.data.size.rows))
	for y := uint(0); y < 2; y++ {
		for x := uint(0); x < 10; x++ {
			expect(t, last.data.GetRowAndCell(x, y).cell.IsZero(), "last.data.getRowAndCell(x, y).cell.isZero()")
		}
	}
}

// Zig: test "term51: clearDirty: clears page and row dirty flags on every page"
func TestPl6_clearDirty_clears_page_and_row_dirty_flags_on_every_page(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	n3 := n2.next

	// Set everything dirty.
	n1.data.dirty = true
	n3.data.dirty = true
	for _, node := range []*PageListNode{n1, n2, n3} {
		rows := node.data.rows
		for i := range rows[:node.data.size.rows] {
			rows[i].dirty = true
		}
	}
	expect(t, n1.data.IsDirty(), "n1.data.isDirty()")
	expect(t, n2.data.IsDirty(), "n2.data.isDirty()")
	expect(t, n3.data.IsDirty(), "n3.data.isDirty()")

	s.ClearDirty()
	for _, node := range []*PageListNode{n1, n2, n3} {
		expect(t, !node.data.dirty, "!node.data.dirty")
		expect(t, !node.data.IsDirty(), "!node.data.isDirty()")
		rows := node.data.rows
		for _, row := range rows[:node.data.size.rows] {
			expect(t, !row.dirty, "!row.dirty")
		}
	}

	// Clean list stays clean.
	s.ClearDirty()
	expect(t, !n2.data.IsDirty(), "!n2.data.isDirty()")

	// Only one row dirty on a middle page.
	n2.data.GetRowAndCell(0, 7).row.dirty = true
	expect(t, n2.data.IsDirty(), "n2.data.isDirty()")
	s.ClearDirty()
	expect(t, !n2.data.IsDirty(), "!n2.data.isDirty()")
}

// Zig: test "term51: clearDirty: leaves rows beyond the page size alone and keeps contents"
func TestPl6_clearDirty_leaves_rows_beyond_the_page_size_alone_and_keeps_contents(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 5)
	defer s.Deinit()
	node := s.pages.first
	page := &node.data
	expect(t, page.capacity.rows > page.size.rows, "page.capacity.rows > page.size.rows")

	rows := page.rows
	rows[page.size.rows].dirty = true
	rows[page.size.rows-1].dirty = true
	*page.GetRowAndCell(1, 1).cell = PageCellInit('K')
	page.GetRowAndCell(0, 2).row.wrap = true
	page.dirty = true

	s.ClearDirty()
	expect(t, !page.dirty, "!page.dirty")
	expect(t, !rows[page.size.rows-1].dirty, "!rows[page.size.rows - 1].dirty")
	// Beyond the page size is not touched.
	expect(t, rows[page.size.rows].dirty, "rows[page.size.rows].dirty")
	rows[page.size.rows].dirty = false
	// Other data is not touched.
	expectEqual(t, uint32('K'), page.GetRowAndCell(1, 1).cell.content.codepoint)
	expect(t, page.GetRowAndCell(0, 2).row.wrap, "page.getRowAndCell(0, 2).row.wrap")
}

// Zig: test "term51: markDirty: marks the row of a point in any tag"
func TestPl6_markDirty_marks_the_row_of_a_point_in_any_tag(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 5)
	defer s.Deinit()
	page := &s.pages.first.data
	pl6GrowRows(t, &s, 10)
	s.ClearDirty()

	// Active y=2 is screen y=12.
	s.markDirty(Point{Tag: PointTagActive, Active: PointCoordinate{x: 3, y: 2}})
	expect(t, page.GetRowAndCell(0, 12).row.dirty, "page.getRowAndCell(0, 12).row.dirty")
	expect(t, !page.GetRowAndCell(0, 2).row.dirty, "!page.getRowAndCell(0, 2).row.dirty")
	expect(t, !page.GetRowAndCell(0, 11).row.dirty, "!page.getRowAndCell(0, 11).row.dirty")
	expect(t, !page.GetRowAndCell(0, 13).row.dirty, "!page.getRowAndCell(0, 13).row.dirty")
	expect(t, !page.dirty, "!page.dirty")

	// Screen
	s.markDirty(Point{Tag: PointTagScreen, Screen: PointCoordinate{x: 9, y: 2}})
	expect(t, page.GetRowAndCell(0, 2).row.dirty, "page.getRowAndCell(0, 2).row.dirty")
	expect(t, !page.GetRowAndCell(0, 3).row.dirty, "!page.getRowAndCell(0, 3).row.dirty")

	// History and viewport
	s.markDirty(Point{Tag: PointTagHistory, History: PointCoordinate{x: 0, y: 4}})
	expect(t, page.GetRowAndCell(0, 4).row.dirty, "page.getRowAndCell(0, 4).row.dirty")
	s.markDirty(Point{Tag: PointTagViewport, Viewport: PointCoordinate{x: 0, y: 4}})
	expect(t, page.GetRowAndCell(0, 14).row.dirty, "page.getRowAndCell(0, 14).row.dirty")
	expect(t, !page.GetRowAndCell(0, 13).row.dirty, "!page.getRowAndCell(0, 13).row.dirty")
}

// Zig: test "term51: markDirty: on a later page"
func TestPl6_markDirty_on_a_later_page(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	s.ClearDirty()

	s.markDirty(pl6Pt(5, uint32(n1.data.size.rows)+4))
	expect(t, n2.data.GetRowAndCell(0, 4).row.dirty, "n2.data.getRowAndCell(0, 4).row.dirty")
	expect(t, !n2.data.GetRowAndCell(0, 3).row.dirty, "!n2.data.getRowAndCell(0, 3).row.dirty")
	expect(t, !n1.data.GetRowAndCell(0, 4).row.dirty, "!n1.data.getRowAndCell(0, 4).row.dirty")
	expect(t, !n1.data.dirty, "!n1.data.dirty")
	expect(t, !n2.data.dirty, "!n2.data.dirty")
}

// Zig: test "term51: isDirty: row and page dirty state of a point"
func TestPl6_isDirty_row_and_page_dirty_state_of_a_point(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	r1 := uint32(n1.data.size.rows)
	s.ClearDirty()

	expect(t, !s.IsDirty(pl6Pt(0, 0)), "!s.isDirty(term51NPt(0, 0))")
	expect(t, !s.IsDirty(pl6Pt(9, r1)), "!s.isDirty(term51NPt(9, r1))")
	expect(t, !s.IsDirty(Point{Tag: PointTagActive, Active: PointCoordinate{x: 0, y: 0}}), "!s.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	// A dirty row: every x of that row, no other rows.
	n2.data.GetRowAndCell(0, 0).row.dirty = true
	expect(t, s.IsDirty(pl6Pt(0, r1)), "s.isDirty(term51NPt(0, r1))")
	expect(t, s.IsDirty(pl6Pt(9, r1)), "s.isDirty(term51NPt(9, r1))")
	expect(t, !s.IsDirty(pl6Pt(0, r1+1)), "!s.isDirty(term51NPt(0, r1 + 1))")
	expect(t, !s.IsDirty(pl6Pt(0, r1-1)), "!s.isDirty(term51NPt(0, r1 - 1))")

	// The same row through another tag.
	active_tl_y := uint32(n2.data.size.rows) - 3
	n2.data.GetRowAndCell(0, 0).row.dirty = false
	n2.data.GetRowAndCell(0, uint(active_tl_y)+1).row.dirty = true
	expect(t, s.IsDirty(Point{Tag: PointTagActive, Active: PointCoordinate{x: 4, y: 1}}), "s.isDirty(.{ .active = .{ .x = 4, .y = 1 } })")
	expect(t, !s.IsDirty(Point{Tag: PointTagActive, Active: PointCoordinate{x: 4, y: 0}}), "!s.isDirty(.{ .active = .{ .x = 4, .y = 0 } })")
	expect(t, !s.IsDirty(Point{Tag: PointTagActive, Active: PointCoordinate{x: 4, y: 2}}), "!s.isDirty(.{ .active = .{ .x = 4, .y = 2 } })")

	// A dirty page marks all of its rows dirty and no other page.
	n2.data.GetRowAndCell(0, uint(active_tl_y)+1).row.dirty = false
	n1.data.dirty = true
	expect(t, s.IsDirty(pl6Pt(3, 5)), "s.isDirty(term51NPt(3, 5))")
	expect(t, s.IsDirty(pl6Pt(3, r1-1)), "s.isDirty(term51NPt(3, r1 - 1))")
	expect(t, !s.IsDirty(pl6Pt(3, r1)), "!s.isDirty(term51NPt(3, r1))")
}

// ---- PageIterator.Chunk ----

// Zig: test "term51: PageIterator.Chunk.rows: slice of the rows of the chunk"
func TestPl6_PageIterator_Chunk_rows_slice_of_the_rows_of_the_chunk(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 8)
	defer s.Deinit()
	node := s.pages.first
	all := node.data.rows

	mid := PageListPageIteratorChunk{node: node, start: 2, end: 5}
	rows := mid.Rows()
	expectEqual(t, uint(3), uint(len(rows)))
	expect(t, &rows[0] == &all[2], "rows.ptr == all + 2")
	expect(t, &rows[2] == &all[4], "&rows[2] == &all[4]")
	// They are the rows of the page, not copies.
	rows[1].wrap = true
	expect(t, node.data.GetRowAndCell(0, 3).row.wrap, "node.data.getRowAndCell(0, 3).row.wrap")
	expect(t, !node.data.GetRowAndCell(0, 2).row.wrap, "!node.data.getRowAndCell(0, 2).row.wrap")

	// A single row, the first and the last.
	one := PageListPageIteratorChunk{node: node, start: 7, end: 8}
	expectEqual(t, uint(1), uint(len(one.Rows())))
	expect(t, &one.Rows()[0] == &all[7], "one.rows().ptr == all + 7")
	first := PageListPageIteratorChunk{node: node, start: 0, end: 1}
	expect(t, &first.Rows()[0] == &all[0], "first.rows().ptr == all")

	// The whole page
	full := PageListPageIteratorChunk{node: node, start: 0, end: node.data.size.rows}
	expectEqual(t, uint(8), uint(len(full.Rows())))
	expect(t, &full.Rows()[0] == &all[0], "full.rows().ptr == all")

	// Empty
	empty := PageListPageIteratorChunk{node: node, start: 3, end: 3}
	expectEqual(t, uint(0), uint(len(empty.Rows())))
}

// Zig: test "term51: PageIterator.Chunk.rows: uses the page of the chunk node"
func TestPl6_PageIterator_Chunk_rows_uses_the_page_of_the_chunk_node(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	c1 := PageListPageIteratorChunk{node: n1, start: 1, end: 3}
	c2 := PageListPageIteratorChunk{node: n2, start: 1, end: 3}
	expect(t, &c1.Rows()[0] == &n1.data.rows[1], "c1.rows().ptr == n1.data.rows.ptr(n1.data.memory) + 1")
	expect(t, &c2.Rows()[0] == &n2.data.rows[1], "c2.rows().ptr == n2.data.rows.ptr(n2.data.memory) + 1")
	expect(t, &c1.Rows()[0] != &c2.Rows()[0], "c1.rows().ptr != c2.rows().ptr")
}

// Zig: test "term51: PageIterator.Chunk.fullPage: only when the chunk covers every row"
func TestPl6_PageIterator_Chunk_fullPage_only_when_the_chunk_covers_every_row(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 8)
	defer s.Deinit()
	node := s.pages.first

	expect(t, PageListPageIteratorChunk{node: node, start: 0, end: 8}.FullPage(), "(PageIterator.Chunk{ .node = node, .start = 0, .end = 8 }).fullPage()")
	expect(t, !PageListPageIteratorChunk{node: node, start: 0, end: 7}.FullPage(), "!(PageIterator.Chunk{ .node = node, .start = 0, .end = 7 }).fullPage()")
	expect(t, !PageListPageIteratorChunk{node: node, start: 1, end: 8}.FullPage(), "!(PageIterator.Chunk{ .node = node, .start = 1, .end = 8 }).fullPage()")
	expect(t, !PageListPageIteratorChunk{node: node, start: 1, end: 7}.FullPage(), "!(PageIterator.Chunk{ .node = node, .start = 1, .end = 7 }).fullPage()")
	expect(t, !PageListPageIteratorChunk{node: node, start: 0, end: 1}.FullPage(), "!(PageIterator.Chunk{ .node = node, .start = 0, .end = 1 }).fullPage()")
	expect(t, !PageListPageIteratorChunk{node: node, start: 0, end: 0}.FullPage(), "!(PageIterator.Chunk{ .node = node, .start = 0, .end = 0 }).fullPage()")
	expect(t, !PageListPageIteratorChunk{node: node, start: 8, end: 8}.FullPage(), "!(PageIterator.Chunk{ .node = node, .start = 8, .end = 8 }).fullPage()")

	// The page size is the number of rows in use, not the capacity.
	pl6GrowRows(t, &s, 1)
	expect(t, !PageListPageIteratorChunk{node: node, start: 0, end: 8}.FullPage(), "!(PageIterator.Chunk{ .node = node, .start = 0, .end = 8 }).fullPage()")
	expect(t, PageListPageIteratorChunk{node: node, start: 0, end: 9}.FullPage(), "(PageIterator.Chunk{ .node = node, .start = 0, .end = 9 }).fullPage()")
}

// Zig: test "term51: PageIterator.Chunk.fullPage: single row page"
func TestPl6_PageIterator_Chunk_fullPage_single_row_page(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 1)
	defer s.Deinit()
	node := s.pages.first
	expect(t, PageListPageIteratorChunk{node: node, start: 0, end: 1}.FullPage(), "(PageIterator.Chunk{ .node = node, .start = 0, .end = 1 }).fullPage()")
	expect(t, !PageListPageIteratorChunk{node: node, start: 0, end: 0}.FullPage(), "!(PageIterator.Chunk{ .node = node, .start = 0, .end = 0 }).fullPage()")
	expect(t, !PageListPageIteratorChunk{node: node, start: 1, end: 1}.FullPage(), "!(PageIterator.Chunk{ .node = node, .start = 1, .end = 1 }).fullPage()")
}

// Zig: test "term51: PageIterator.Chunk.overlaps: ranges on the same page"
func TestPl6_PageIterator_Chunk_overlaps_ranges_on_the_same_page(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 12)
	defer s.Deinit()
	node := s.pages.first

	a := PageListPageIteratorChunk{node: node, start: 2, end: 5}
	type Case struct {
		start uint16
		end   uint16
		want  bool
	}
	cases := []Case{
		// Entirely before, touching before (end is exclusive)
		{start: 0, end: 1, want: false},
		{start: 0, end: 2, want: false},
		// Partial overlap at the start
		{start: 0, end: 3, want: true},
		{start: 1, end: 4, want: true},
		// One row overlap at each end
		{start: 4, end: 9, want: true},
		{start: 1, end: 3, want: true},
		// Identical, inside, containing
		{start: 2, end: 5, want: true},
		{start: 3, end: 4, want: true},
		{start: 2, end: 3, want: true},
		{start: 4, end: 5, want: true},
		{start: 0, end: 12, want: true},
		{start: 1, end: 6, want: true},
		// Touching after, entirely after
		{start: 5, end: 8, want: false},
		{start: 6, end: 12, want: false},
		{start: 11, end: 12, want: false},
	}
	for _, c := range cases {
		b := PageListPageIteratorChunk{node: node, start: c.start, end: c.end}
		expectEqual(t, c.want, a.Overlaps(b))
		// Symmetric
		expectEqual(t, c.want, b.Overlaps(a))
	}
}

// Zig: test "term51: PageIterator.Chunk.overlaps: different pages never overlap"
func TestPl6_PageIterator_Chunk_overlaps_different_pages_never_overlap(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next

	a := PageListPageIteratorChunk{node: n1, start: 0, end: n1.data.size.rows}
	b := PageListPageIteratorChunk{node: n2, start: 0, end: n2.data.size.rows}
	expect(t, !a.Overlaps(b), "!a.overlaps(b)")
	expect(t, !b.Overlaps(a), "!b.overlaps(a)")
	// Same row range on both pages.
	c := PageListPageIteratorChunk{node: n1, start: 2, end: 4}
	d := PageListPageIteratorChunk{node: n2, start: 2, end: 4}
	expect(t, !c.Overlaps(d), "!c.overlaps(d)")
	expect(t, c.Overlaps(c), "c.overlaps(c)")
}

// Zig: test "term51: PageIterator.Chunk.overlaps: empty chunks"
func TestPl6_PageIterator_Chunk_overlaps_empty_chunks(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 12)
	defer s.Deinit()
	node := s.pages.first

	// An empty chunk strictly inside another range overlaps it, because
	// only the bounds are compared.
	empty := PageListPageIteratorChunk{node: node, start: 3, end: 3}
	wide := PageListPageIteratorChunk{node: node, start: 0, end: 10}
	expect(t, empty.Overlaps(wide), "empty.overlaps(wide)")
	expect(t, wide.Overlaps(empty), "wide.overlaps(empty)")
	// An empty chunk at the edge of a range does not.
	expect(t, !empty.Overlaps(PageListPageIteratorChunk{node: node, start: 3, end: 5}), "!empty.overlaps(.{ .node = node, .start = 3, .end = 5 })")
	expect(t, !empty.Overlaps(PageListPageIteratorChunk{node: node, start: 0, end: 3}), "!empty.overlaps(.{ .node = node, .start = 0, .end = 3 })")
	// An empty chunk never overlaps itself.
	expect(t, !empty.Overlaps(empty), "!empty.overlaps(empty)")
	// A chunk and an empty chunk right after it
	expect(t, !PageListPageIteratorChunk{node: node, start: 0, end: 3}.Overlaps(empty), "!(PageIterator.Chunk{ .node = node, .start = 0, .end = 3 }).overlaps(empty)")
}

// ---- PageIterator.next / nextDown / nextUp ----

// Zig: test "term51: PageIterator.next: default iterator is empty and dispatches by direction"
func TestPl6_PageIterator_next_default_iterator_is_empty_and_dispatches_by_direction(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	n3 := n2.next

	// The default iterator has no row.
	empty := PageListPageIterator{direction: PageListDirectionRightDown}
	expect(t, empty.direction == PageListDirectionRightDown, "empty.direction == .right_down")
	expect(t, empty.limit.Tag == pageListPageIteratorLimitNone, "empty.limit == .none")
	expect(t, !empty.Next().Valid, "empty.next() == null")
	expect(t, !empty.nextDown().Valid, "empty.nextDown() == null")
	expect(t, !empty.nextUp().Valid, "empty.nextUp() == null")
	empty_up := PageListPageIterator{direction: PageListDirectionLeftUp}
	expect(t, !empty_up.Next().Valid, "empty_up.next() == null")

	// right_down behaves as nextDown.
	a := PageListPageIterator{row: Some(PageListPin{node: n2, y: 4}), direction: PageListDirectionRightDown}
	b := a
	pl6ExpectChunk(t, a.Next(), n2, 4, uint(n2.data.size.rows))
	pl6ExpectChunk(t, b.nextDown(), n2, 4, uint(n2.data.size.rows))
	pl6ExpectChunk(t, a.Next(), n3, 0, uint(n3.data.size.rows))
	pl6ExpectChunk(t, b.nextDown(), n3, 0, uint(n3.data.size.rows))
	expect(t, !a.Next().Valid, "a.next() == null")
	expect(t, !b.nextDown().Valid, "b.nextDown() == null")

	// left_up behaves as nextUp.
	c := PageListPageIterator{row: Some(PageListPin{node: n2, y: 4}), direction: PageListDirectionLeftUp}
	d := c
	pl6ExpectChunk(t, c.Next(), n2, 0, 5)
	pl6ExpectChunk(t, d.nextUp(), n2, 0, 5)
	pl6ExpectChunk(t, c.Next(), n1, 0, uint(n1.data.size.rows))
	pl6ExpectChunk(t, d.nextUp(), n1, 0, uint(n1.data.size.rows))
	expect(t, !c.Next().Valid, "c.next() == null")
	expect(t, !d.nextUp().Valid, "d.nextUp() == null")
}

// Zig: test "term51: PageIterator.nextDown: no limit consumes whole pages from the row"
func TestPl6_PageIterator_nextDown_no_limit_consumes_whole_pages_from_the_row(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	n3 := n2.next

	// From the middle of the first page.
	it := PageListPageIterator{row: Some(PageListPin{node: n1, y: 5, x: 7}), direction: PageListDirectionRightDown}
	var buf [4]PageListPageIteratorChunk
	n := pl6DrainPages(t, &it, buf[:])
	expectEqual(t, uint(3), n)
	pl6ExpectChunkV(t, buf[0], n1, 5, uint(n1.data.size.rows))
	pl6ExpectChunkV(t, buf[1], n2, 0, uint(n2.data.size.rows))
	pl6ExpectChunkV(t, buf[2], n3, 0, uint(n3.data.size.rows))
	expect(t, !buf[0].FullPage(), "!buf[0].fullPage()")
	expect(t, buf[1].FullPage(), "buf[1].fullPage()")

	// The state after one call: the next page, row 0.
	it2 := PageListPageIterator{row: Some(PageListPin{node: n1, y: 5}), direction: PageListDirectionRightDown}
	_ = zigUnwrap(t, it2.nextDown())
	pl6ExpectPin(t, it2.row, n2, 0, 0)

	// Starting on the last row of a page gives a one row chunk.
	it3 := PageListPageIterator{row: Some(PageListPin{node: n2, y: n2.data.size.rows - 1}), direction: PageListDirectionRightDown}
	pl6ExpectChunk(t, it3.nextDown(), n2, uint(n2.data.size.rows)-1, uint(n2.data.size.rows))
	pl6ExpectChunk(t, it3.nextDown(), n3, 0, uint(n3.data.size.rows))
	expect(t, !it3.nextDown().Valid, "it3.nextDown() == null")

	// Starting at the first row of the last page.
	it4 := PageListPageIterator{row: Some(PageListPin{node: n3}), direction: PageListDirectionRightDown}
	pl6ExpectChunk(t, it4.nextDown(), n3, 0, uint(n3.data.size.rows))
	expect(t, !it4.row.Valid, "it4.row == null")
	expect(t, !it4.nextDown().Valid, "it4.nextDown() == null")
}

// Zig: test "term51: PageIterator.nextDown: row limit on the same page"
func TestPl6_PageIterator_nextDown_row_limit_on_the_same_page(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	n1 := s.pages.first

	// The limit row is included, and the iterator ends after one chunk even
	// though more pages follow.
	it := PageListPageIterator{
		row:       Some(PageListPin{node: n1, y: 2}),
		limit:     pl6LimitRow(PageListPin{node: n1, y: 7}),
		direction: PageListDirectionRightDown,
	}
	pl6ExpectChunk(t, it.nextDown(), n1, 2, 8)
	expect(t, !it.row.Valid, "it.row == null")
	expect(t, !it.nextDown().Valid, "it.nextDown() == null")

	// Single row.
	one := PageListPageIterator{
		row:       Some(PageListPin{node: n1, y: 5}),
		limit:     pl6LimitRow(PageListPin{node: n1, y: 5}),
		direction: PageListDirectionRightDown,
	}
	pl6ExpectChunk(t, one.nextDown(), n1, 5, 6)
	expect(t, !one.nextDown().Valid, "one.nextDown() == null")

	// The first row only.
	zero := PageListPageIterator{
		row:       Some(PageListPin{node: n1, y: 0}),
		limit:     pl6LimitRow(PageListPin{node: n1, y: 0}),
		direction: PageListDirectionRightDown,
	}
	pl6ExpectChunk(t, zero.nextDown(), n1, 0, 1)
	expect(t, !zero.nextDown().Valid, "zero.nextDown() == null")

	// The limit is the last row of the page: the whole rest of the page,
	// and nothing from the next page.
	last_y := n1.data.size.rows - 1
	last := PageListPageIterator{
		row:       Some(PageListPin{node: n1, y: 4}),
		limit:     pl6LimitRow(PageListPin{node: n1, y: last_y}),
		direction: PageListDirectionRightDown,
	}
	pl6ExpectChunk(t, last.nextDown(), n1, 4, uint(n1.data.size.rows))
	expect(t, !last.nextDown().Valid, "last.nextDown() == null")

	// The x of the row and the limit is not used.
	xs := PageListPageIterator{
		row:       Some(PageListPin{node: n1, y: 3, x: 9}),
		limit:     pl6LimitRow(PageListPin{node: n1, y: 3, x: 0}),
		direction: PageListDirectionRightDown,
	}
	pl6ExpectChunk(t, xs.nextDown(), n1, 3, 4)
}

// Zig: test "term51: PageIterator.nextDown: row limit on a later page"
func TestPl6_PageIterator_nextDown_row_limit_on_a_later_page(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	n3 := n2.next

	it := PageListPageIterator{
		row:       Some(PageListPin{node: n1, y: 4}),
		limit:     pl6LimitRow(PageListPin{node: n3, y: 6}),
		direction: PageListDirectionRightDown,
	}
	var buf [4]PageListPageIteratorChunk
	n := pl6DrainPages(t, &it, buf[:])
	expectEqual(t, uint(3), n)
	pl6ExpectChunkV(t, buf[0], n1, 4, uint(n1.data.size.rows))
	pl6ExpectChunkV(t, buf[1], n2, 0, uint(n2.data.size.rows))
	pl6ExpectChunkV(t, buf[2], n3, 0, 7)

	// Limit on the first row of the next page.
	adj := PageListPageIterator{
		row:       Some(PageListPin{node: n2, y: n2.data.size.rows - 1}),
		limit:     pl6LimitRow(PageListPin{node: n3, y: 0}),
		direction: PageListDirectionRightDown,
	}
	pl6ExpectChunk(t, adj.nextDown(), n2, uint(n2.data.size.rows)-1, uint(n2.data.size.rows))
	expect(t, adj.row.Valid, "adj.row != null")
	pl6ExpectChunk(t, adj.nextDown(), n3, 0, 1)
	expect(t, !adj.nextDown().Valid, "adj.nextDown() == null")
}

// Zig: test "term51: PageIterator.nextDown: row limit before the row or on an earlier page"
func TestPl6_PageIterator_nextDown_row_limit_before_the_row_or_on_an_earlier_page(t *testing.T) {
	// These need hand built iterators: Pin.pageIterator asserts the order
	// in safe builds.
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	n3 := n2.next

	// Same page, limit above the row: nothing, and the iterator is finished.
	above := PageListPageIterator{
		row:       Some(PageListPin{node: n1, y: 7}),
		limit:     pl6LimitRow(PageListPin{node: n1, y: 2}),
		direction: PageListDirectionRightDown,
	}
	expect(t, !above.nextDown().Valid, "above.nextDown() == null")
	expect(t, !above.row.Valid, "above.row == null")
	expect(t, !above.nextDown().Valid, "above.nextDown() == null")

	// The limit is on an earlier page, so it is never reached and every page
	// to the end of the list is returned.
	earlier := PageListPageIterator{
		row:       Some(PageListPin{node: n2, y: 3}),
		limit:     pl6LimitRow(PageListPin{node: n1, y: 3}),
		direction: PageListDirectionRightDown,
	}
	var buf [4]PageListPageIteratorChunk
	n := pl6DrainPages(t, &earlier, buf[:])
	expectEqual(t, uint(2), n)
	pl6ExpectChunkV(t, buf[0], n2, 3, uint(n2.data.size.rows))
	pl6ExpectChunkV(t, buf[1], n3, 0, uint(n3.data.size.rows))
}

// Zig: test "term51: PageIterator.nextDown: count limit yields a single chunk"
func TestPl6_PageIterator_nextDown_count_limit_yields_a_single_chunk(t *testing.T) {
	// Upstream quirk pin: the count branch computes len = min(rows left in
	// the page, count) and then tests len > count, which is never true, so
	// the iterator always finishes after the first chunk and never moves on
	// to the next page, and the count is never decremented.
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	n1 := s.pages.first
	r1 := n1.data.size.rows

	// Fewer rows than remain in the page.
	it := PageListPageIterator{row: Some(PageListPin{node: n1, y: 2}), limit: pl6LimitCount(5), direction: PageListDirectionRightDown}
	pl6ExpectChunk(t, it.nextDown(), n1, 2, 7)
	expect(t, !it.row.Valid, "it.row == null")
	expectEqual(t, uint(5), it.limit.Count)
	expect(t, !it.nextDown().Valid, "it.nextDown() == null")

	// Exactly the rows that remain in the page.
	exact := PageListPageIterator{row: Some(PageListPin{node: n1, y: r1 - 3}), limit: pl6LimitCount(3), direction: PageListDirectionRightDown}
	pl6ExpectChunk(t, exact.nextDown(), n1, uint(r1)-3, uint(r1))
	expect(t, !exact.nextDown().Valid, "exact.nextDown() == null")

	// More rows than remain: the chunk is cut at the page end and the next
	// page is not visited.
	more := PageListPageIterator{row: Some(PageListPin{node: n1, y: r1 - 3}), limit: pl6LimitCount(10), direction: PageListDirectionRightDown}
	pl6ExpectChunk(t, more.nextDown(), n1, uint(r1)-3, uint(r1))
	expect(t, !more.row.Valid, "more.row == null")
	expect(t, !more.nextDown().Valid, "more.nextDown() == null")

	// A count of one.
	one := PageListPageIterator{row: Some(PageListPin{node: n1, y: 0}), limit: pl6LimitCount(1), direction: PageListDirectionRightDown}
	pl6ExpectChunk(t, one.nextDown(), n1, 0, 1)
	expect(t, !one.nextDown().Valid, "one.nextDown() == null")

	// Through next().
	via := PageListPageIterator{
		row:       Some(PageListPin{node: n1, y: 1}),
		limit:     pl6LimitCount(2),
		direction: PageListDirectionRightDown,
	}
	pl6ExpectChunk(t, via.Next(), n1, 1, 3)
	expect(t, !via.Next().Valid, "via.next() == null")
}

// Zig: test "term51: PageIterator.nextUp: no limit consumes whole pages up to row zero"
func TestPl6_PageIterator_nextUp_no_limit_consumes_whole_pages_up_to_row_zero(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	n3 := n2.next

	it := PageListPageIterator{row: Some(PageListPin{node: n3, y: 5, x: 7}), direction: PageListDirectionLeftUp}
	var buf [4]PageListPageIteratorChunk
	n := pl6DrainPages(t, &it, buf[:])
	expectEqual(t, uint(3), n)
	pl6ExpectChunkV(t, buf[0], n3, 0, 6)
	pl6ExpectChunkV(t, buf[1], n2, 0, uint(n2.data.size.rows))
	pl6ExpectChunkV(t, buf[2], n1, 0, uint(n1.data.size.rows))

	// The state after one call: the last row of the previous page.
	it2 := PageListPageIterator{row: Some(PageListPin{node: n3, y: 5}), direction: PageListDirectionLeftUp}
	_ = zigUnwrap(t, it2.nextUp())
	pl6ExpectPin(t, it2.row, n2, uint(n2.data.size.rows)-1, 0)

	// The first row of a page gives a one row chunk.
	it3 := PageListPageIterator{row: Some(PageListPin{node: n2, y: 0}), direction: PageListDirectionLeftUp}
	pl6ExpectChunk(t, it3.nextUp(), n2, 0, 1)
	pl6ExpectChunk(t, it3.nextUp(), n1, 0, uint(n1.data.size.rows))
	expect(t, !it3.nextUp().Valid, "it3.nextUp() == null")

	// The first page ends the iteration.
	it4 := PageListPageIterator{row: Some(PageListPin{node: n1, y: 0}), direction: PageListDirectionLeftUp}
	pl6ExpectChunk(t, it4.nextUp(), n1, 0, 1)
	expect(t, !it4.row.Valid, "it4.row == null")
	expect(t, !it4.nextUp().Valid, "it4.nextUp() == null")
}

// Zig: test "term51: PageIterator.nextUp: row limit on the same page"
func TestPl6_PageIterator_nextUp_row_limit_on_the_same_page(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next

	// Limit row is included; one chunk even though earlier pages exist.
	it := PageListPageIterator{
		row:       Some(PageListPin{node: n2, y: 7}),
		limit:     pl6LimitRow(PageListPin{node: n2, y: 2}),
		direction: PageListDirectionLeftUp,
	}
	pl6ExpectChunk(t, it.nextUp(), n2, 2, 8)
	expect(t, !it.row.Valid, "it.row == null")
	expect(t, !it.nextUp().Valid, "it.nextUp() == null")

	// Single row.
	one := PageListPageIterator{
		row:       Some(PageListPin{node: n2, y: 5}),
		limit:     pl6LimitRow(PageListPin{node: n2, y: 5}),
		direction: PageListDirectionLeftUp,
	}
	pl6ExpectChunk(t, one.nextUp(), n2, 5, 6)
	expect(t, !one.nextUp().Valid, "one.nextUp() == null")

	// Limit is the first row of the page.
	zero := PageListPageIterator{
		row:       Some(PageListPin{node: n2, y: 4}),
		limit:     pl6LimitRow(PageListPin{node: n2, y: 0}),
		direction: PageListDirectionLeftUp,
	}
	pl6ExpectChunk(t, zero.nextUp(), n2, 0, 5)
	expect(t, !zero.nextUp().Valid, "zero.nextUp() == null")

	// Both are row zero.
	zz := PageListPageIterator{
		row:       Some(PageListPin{node: n1, y: 0}),
		limit:     pl6LimitRow(PageListPin{node: n1, y: 0}),
		direction: PageListDirectionLeftUp,
	}
	pl6ExpectChunk(t, zz.nextUp(), n1, 0, 1)
	expect(t, !zz.nextUp().Valid, "zz.nextUp() == null")

	// The x of the row and the limit is not used.
	xs := PageListPageIterator{
		row:       Some(PageListPin{node: n1, y: 3, x: 0}),
		limit:     pl6LimitRow(PageListPin{node: n1, y: 3, x: 9}),
		direction: PageListDirectionLeftUp,
	}
	pl6ExpectChunk(t, xs.nextUp(), n1, 3, 4)
}

// Zig: test "term51: PageIterator.nextUp: row limit on an earlier page"
func TestPl6_PageIterator_nextUp_row_limit_on_an_earlier_page(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	n3 := n2.next

	it := PageListPageIterator{
		row:       Some(PageListPin{node: n3, y: 4}),
		limit:     pl6LimitRow(PageListPin{node: n1, y: 6}),
		direction: PageListDirectionLeftUp,
	}
	var buf [4]PageListPageIteratorChunk
	n := pl6DrainPages(t, &it, buf[:])
	expectEqual(t, uint(3), n)
	pl6ExpectChunkV(t, buf[0], n3, 0, 5)
	pl6ExpectChunkV(t, buf[1], n2, 0, uint(n2.data.size.rows))
	pl6ExpectChunkV(t, buf[2], n1, 6, uint(n1.data.size.rows))

	// Limit on the last row of the previous page.
	adj := PageListPageIterator{
		row:       Some(PageListPin{node: n2, y: 0}),
		limit:     pl6LimitRow(PageListPin{node: n1, y: n1.data.size.rows - 1}),
		direction: PageListDirectionLeftUp,
	}
	pl6ExpectChunk(t, adj.nextUp(), n2, 0, 1)
	expect(t, adj.row.Valid, "adj.row != null")
	pl6ExpectChunk(t, adj.nextUp(), n1, uint(n1.data.size.rows)-1, uint(n1.data.size.rows))
	expect(t, !adj.nextUp().Valid, "adj.nextUp() == null")
}

// Zig: test "term51: PageIterator.nextUp: row limit below the row or on a later page"
func TestPl6_PageIterator_nextUp_row_limit_below_the_row_or_on_a_later_page(t *testing.T) {
	// Hand built iterators: Pin.pageIterator asserts the order in safe builds.
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next

	// Same page, limit below the row: nothing, and the iterator is finished.
	below := PageListPageIterator{
		row:       Some(PageListPin{node: n2, y: 2}),
		limit:     pl6LimitRow(PageListPin{node: n2, y: 7}),
		direction: PageListDirectionLeftUp,
	}
	expect(t, !below.nextUp().Valid, "below.nextUp() == null")
	expect(t, !below.row.Valid, "below.row == null")

	// The limit is on a later page so it is never reached and every page to
	// the start of the list is returned.
	later := PageListPageIterator{
		row:       Some(PageListPin{node: n2, y: 3}),
		limit:     pl6LimitRow(PageListPin{node: s.pages.last, y: 3}),
		direction: PageListDirectionLeftUp,
	}
	var buf [4]PageListPageIteratorChunk
	n := pl6DrainPages(t, &later, buf[:])
	expectEqual(t, uint(2), n)
	pl6ExpectChunkV(t, buf[0], n2, 0, 4)
	pl6ExpectChunkV(t, buf[1], n1, 0, uint(n1.data.size.rows))
}

// Zig: test "term51: PageIterator.nextUp: count limit yields a single chunk with a short end"
func TestPl6_PageIterator_nextUp_count_limit_yields_a_single_chunk_with_a_short_end(t *testing.T) {
	// Upstream quirk pin: like the down case the iterator always ends after
	// one chunk without touching the count. The chunk end is row.y - 1 where
	// row.y + 1 would cover the starting row, so the chunk is [y - len, y - 1).
	// (With y == 0 the end computation underflows and panics in safe builds,
	// so that case is not exercised.)
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next

	it := PageListPageIterator{
		row:       Some(PageListPin{node: n2, y: 5}),
		limit:     pl6LimitCount(3),
		direction: PageListDirectionLeftUp,
	}
	pl6ExpectChunk(t, it.nextUp(), n2, 2, 4)
	expect(t, !it.row.Valid, "it.row == null")
	expectEqual(t, uint(3), it.limit.Count)
	expect(t, !it.nextUp().Valid, "it.nextUp() == null")

	// Count larger than the rows above: capped at row.y.
	big := PageListPageIterator{
		row:       Some(PageListPin{node: n2, y: 5}),
		limit:     pl6LimitCount(100),
		direction: PageListDirectionLeftUp,
	}
	pl6ExpectChunk(t, big.nextUp(), n2, 0, 4)
	expect(t, !big.nextUp().Valid, "big.nextUp() == null")

	// Count equal to the rows above.
	eq := PageListPageIterator{
		row:       Some(PageListPin{node: n2, y: 5}),
		limit:     pl6LimitCount(5),
		direction: PageListDirectionLeftUp,
	}
	pl6ExpectChunk(t, eq.nextUp(), n2, 0, 4)

	// From row 1 the chunk is empty.
	one := PageListPageIterator{
		row:       Some(PageListPin{node: n2, y: 1}),
		limit:     pl6LimitCount(10),
		direction: PageListDirectionLeftUp,
	}
	pl6ExpectChunk(t, one.nextUp(), n2, 0, 0)
	expect(t, !one.nextUp().Valid, "one.nextUp() == null")

	// Through next(), and the earlier page is not visited.
	via := PageListPageIterator{
		row:       Some(PageListPin{node: n2, y: 4}),
		limit:     pl6LimitCount(2),
		direction: PageListDirectionLeftUp,
	}
	pl6ExpectChunk(t, via.Next(), n2, 2, 3)
	expect(t, !via.Next().Valid, "via.next() == null")
}

// ---- pageIterator / Pin.pageIterator ----

// pl6ExpChunk is term51NExpChunk.
type pl6ExpChunk struct {
	node  *PageListNode
	start uint
	end   uint
}

// pl6CheckPages is term51NCheckPages: check the chunks of an iterator.
// `expected` is in right_down order and is walked backwards for left_up
// iterators.
func pl6CheckPages(
	t *testing.T,
	it *PageListPageIterator,
	dir PageListDirection,
	expected []pl6ExpChunk,
) {
	t.Helper()
	for i := 0; i < len(expected); i++ {
		var e pl6ExpChunk
		switch dir {
		case PageListDirectionRightDown:
			e = expected[i]
		case PageListDirectionLeftUp:
			e = expected[len(expected)-1-i]
		}
		pl6ExpectChunk(t, it.Next(), e.node, e.start, e.end)
	}
	expect(t, !it.Next().Valid, "it.next() == null")
	expect(t, !it.Next().Valid, "it.next() == null")
}

// pl6Dirs is the Zig `[_]Direction{ .right_down, .left_up }` of the
// `inline for` loops.
var pl6Dirs = []PageListDirection{PageListDirectionRightDown, PageListDirectionLeftUp}

// Zig: test "term51: pageIterator: no bottom bound iterates to the end of the tag area"
func TestPl6_pageIterator_no_bottom_bound_iterates_to_the_end_of_the_tag_area(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	pl6GrowRows(t, &s, 1)
	n1 := s.pages.first
	n2 := n1.next
	n3 := n2.next
	n4 := n3.next
	expect(t, n4.next == nil, "n4.next == null")
	r1 := uint(n1.data.size.rows)
	r2 := uint(n2.data.size.rows)
	r3 := uint(n3.data.size.rows)
	expectEqual(t, uint(1), uint(n4.data.size.rows))

	screen := []pl6ExpChunk{
		{node: n1, start: 0, end: r1},
		{node: n2, start: 0, end: r2},
		{node: n3, start: 0, end: r3},
		{node: n4, start: 0, end: 1},
	}
	active := []pl6ExpChunk{
		{node: n3, start: r3 - 2, end: r3},
		{node: n4, start: 0, end: 1},
	}
	history := []pl6ExpChunk{
		{node: n1, start: 0, end: r1},
		{node: n2, start: 0, end: r2},
		{node: n3, start: 0, end: r3 - 2},
	}

	for _, dir := range pl6Dirs {
		it := s.PageIterator(dir, Point{Tag: PointTagScreen}, None[Point]())
		pl6CheckPages(t, &it, dir, screen)
		it = s.PageIterator(dir, Point{Tag: PointTagActive}, None[Point]())
		pl6CheckPages(t, &it, dir, active)
		it = s.PageIterator(dir, Point{Tag: PointTagHistory}, None[Point]())
		pl6CheckPages(t, &it, dir, history)
		// The viewport is the active area.
		it = s.PageIterator(dir, Point{Tag: PointTagViewport}, None[Point]())
		pl6CheckPages(t, &it, dir, active)
	}
}

// Zig: test "term51: pageIterator: viewport tag follows a scrolled viewport"
func TestPl6_pageIterator_viewport_tag_follows_a_scrolled_viewport(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	pl6GrowRows(t, &s, 1)
	n1 := s.pages.first
	n2 := n1.next
	r1 := uint(n1.data.size.rows)

	// Viewport at the top: the first three rows.
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	for _, dir := range pl6Dirs {
		it := s.PageIterator(dir, Point{Tag: PointTagViewport}, None[Point]())
		pl6CheckPages(t, &it, dir, []pl6ExpChunk{
			{node: n1, start: 0, end: 3},
		})
	}

	// Viewport pinned in the middle of the second page.
	s.Scroll(PageListScroll{Tag: PageListScrollRow, Row: r1 + 5})
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	for _, dir := range pl6Dirs {
		it := s.PageIterator(dir, Point{Tag: PointTagViewport}, None[Point]())
		pl6CheckPages(t, &it, dir, []pl6ExpChunk{
			{node: n2, start: 5, end: 8},
		})
	}

	// Viewport pinned so that it crosses a page boundary.
	s.Scroll(PageListScroll{Tag: PageListScrollRow, Row: r1 - 1})
	for _, dir := range pl6Dirs {
		it := s.PageIterator(dir, Point{Tag: PointTagViewport}, None[Point]())
		pl6CheckPages(t, &it, dir, []pl6ExpChunk{
			{node: n1, start: r1 - 1, end: r1},
			{node: n2, start: 0, end: 2},
		})
	}
}

// Zig: test "term51: pageIterator: history is empty when there is no history"
func TestPl6_pageIterator_history_is_empty_when_there_is_no_history(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 5)
	defer s.Deinit()

	for _, dir := range pl6Dirs {
		it := s.PageIterator(dir, Point{Tag: PointTagHistory}, None[Point]())
		expect(t, !it.row.Valid, "it.row == null")
		expect(t, !it.Next().Valid, "it.next() == null")
		expect(t, !it.Next().Valid, "it.next() == null")
	}

	// Once there is one row of history it is the only chunk.
	pl6GrowRows(t, &s, 1)
	node := s.pages.first
	for _, dir := range pl6Dirs {
		it := s.PageIterator(dir, Point{Tag: PointTagHistory}, None[Point]())
		pl6CheckPages(t, &it, dir, []pl6ExpChunk{
			{node: node, start: 0, end: 1},
		})
	}
}

// Zig: test "term51: pageIterator: explicit bounds on one page"
func TestPl6_pageIterator_explicit_bounds_on_one_page(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	r1 := uint32(n1.data.size.rows)

	for _, dir := range pl6Dirs {
		// Several rows.
		it := s.PageIterator(dir, pl6Pt(0, r1+4), Some(pl6Pt(0, r1+9)))
		pl6CheckPages(t, &it, dir, []pl6ExpChunk{
			{node: n2, start: 4, end: 10},
		})
		// Single row; the x of both ends is ignored.
		it = s.PageIterator(dir, pl6Pt(2, r1+4), Some(pl6Pt(7, r1+4)))
		pl6CheckPages(t, &it, dir, []pl6ExpChunk{
			{node: n2, start: 4, end: 5},
		})
		// Single cell.
		it = s.PageIterator(dir, pl6Pt(5, r1+4), Some(pl6Pt(5, r1+4)))
		pl6CheckPages(t, &it, dir, []pl6ExpChunk{
			{node: n2, start: 4, end: 5},
		})
		// First and last rows of the first page.
		it = s.PageIterator(dir, pl6Pt(0, 0), Some(pl6Pt(0, 0)))
		pl6CheckPages(t, &it, dir, []pl6ExpChunk{
			{node: n1, start: 0, end: 1},
		})
		it = s.PageIterator(dir, pl6Pt(0, r1-1), Some(pl6Pt(0, r1-1)))
		pl6CheckPages(t, &it, dir, []pl6ExpChunk{
			{node: n1, start: uint(r1) - 1, end: uint(r1)},
		})
		// Whole first page
		it = s.PageIterator(dir, pl6Pt(0, 0), Some(pl6Pt(0, r1-1)))
		pl6CheckPages(t, &it, dir, []pl6ExpChunk{
			{node: n1, start: 0, end: uint(r1)},
		})
	}
}

// Zig: test "term51: pageIterator: explicit bounds across pages"
func TestPl6_pageIterator_explicit_bounds_across_pages(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	pl6GrowRows(t, &s, 1)
	n1 := s.pages.first
	n2 := n1.next
	n3 := n2.next
	n4 := n3.next
	r1 := uint32(n1.data.size.rows)
	r2 := uint32(n2.data.size.rows)
	r3 := uint32(n3.data.size.rows)

	for _, dir := range pl6Dirs {
		// Two adjacent pages.
		it := s.PageIterator(dir, pl6Pt(0, r1-2), Some(pl6Pt(0, r1+1)))
		pl6CheckPages(t, &it, dir, []pl6ExpChunk{
			{node: n1, start: uint(r1) - 2, end: uint(r1)},
			{node: n2, start: 0, end: 2},
		})
		// Last row of a page to the first row of the next.
		it = s.PageIterator(dir, pl6Pt(0, r1-1), Some(pl6Pt(0, r1)))
		pl6CheckPages(t, &it, dir, []pl6ExpChunk{
			{node: n1, start: uint(r1) - 1, end: uint(r1)},
			{node: n2, start: 0, end: 1},
		})
		// Three pages with a full middle page.
		it = s.PageIterator(dir, pl6Pt(3, 3), Some(pl6Pt(1, r1+r2+5)))
		pl6CheckPages(t, &it, dir, []pl6ExpChunk{
			{node: n1, start: 3, end: uint(r1)},
			{node: n2, start: 0, end: uint(r2)},
			{node: n3, start: 0, end: 6},
		})
		// Every page, ending on the single row of the last page.
		it = s.PageIterator(dir, pl6Pt(0, 0), Some(pl6Pt(9, r1+r2+r3)))
		pl6CheckPages(t, &it, dir, []pl6ExpChunk{
			{node: n1, start: 0, end: uint(r1)},
			{node: n2, start: 0, end: uint(r2)},
			{node: n3, start: 0, end: uint(r3)},
			{node: n4, start: 0, end: 1},
		})
		// Whole middle pages only.
		it = s.PageIterator(dir, pl6Pt(0, r1), Some(pl6Pt(0, r1+r2-1)))
		pl6CheckPages(t, &it, dir, []pl6ExpChunk{
			{node: n2, start: 0, end: uint(r2)},
		})
	}
}

// Zig: test "term51: pageIterator: bounds in different tags"
func TestPl6_pageIterator_bounds_in_different_tags(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	pl6GrowRows(t, &s, 1)
	n3 := s.pages.first.next.next
	n4 := n3.next
	r1 := uint32(s.pages.first.data.size.rows)
	r2 := uint32(s.pages.first.next.data.size.rows)
	r3 := uint32(n3.data.size.rows)

	for _, dir := range pl6Dirs {
		// The whole active area given in active points.
		it := s.PageIterator(dir, Point{Tag: PointTagActive}, Some(Point{Tag: PointTagActive, Active: PointCoordinate{x: 9, y: 2}}))
		pl6CheckPages(t, &it, dir, []pl6ExpChunk{
			{node: n3, start: uint(r3) - 2, end: uint(r3)},
			{node: n4, start: 0, end: 1},
		})
		// The first two active rows
		it = s.PageIterator(dir, Point{Tag: PointTagActive}, Some(Point{Tag: PointTagActive, Active: PointCoordinate{y: 1}}))
		pl6CheckPages(t, &it, dir, []pl6ExpChunk{
			{node: n3, start: uint(r3) - 2, end: uint(r3)},
		})
		// The last active row only, in viewport points.
		it = s.PageIterator(dir, Point{Tag: PointTagViewport, Viewport: PointCoordinate{y: 2}}, Some(Point{Tag: PointTagViewport, Viewport: PointCoordinate{y: 2}}))
		pl6CheckPages(t, &it, dir, []pl6ExpChunk{
			{node: n4, start: 0, end: 1},
		})
		// Last row of history to the first active row.
		it = s.PageIterator(dir, Point{Tag: PointTagHistory, History: PointCoordinate{y: r1 + r2 + r3 - 3}}, Some(Point{Tag: PointTagActive}))
		pl6CheckPages(t, &it, dir, []pl6ExpChunk{
			{node: n3, start: uint(r3) - 3, end: uint(r3) - 1},
		})
	}
}

// Zig: test "term51: Pin.pageIterator: sets row limit and direction"
func TestPl6_Pin_pageIterator_sets_row_limit_and_direction(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	n3 := n2.next

	a := PageListPin{node: n1, y: 3, x: 4}
	b := PageListPin{node: n3, y: 7, x: 5}

	// No limit
	{
		it := a.PageIterator(PageListDirectionRightDown, None[PageListPin]())
		expect(t, it.direction == PageListDirectionRightDown, "it.direction == .right_down")
		expect(t, it.limit.Tag == pageListPageIteratorLimitNone, "it.limit == .none")
		pl6ExpectPin(t, it.row, n1, 3, 4)
	}
	{
		it := b.PageIterator(PageListDirectionLeftUp, None[PageListPin]())
		expect(t, it.direction == PageListDirectionLeftUp, "it.direction == .left_up")
		expect(t, it.limit.Tag == pageListPageIteratorLimitNone, "it.limit == .none")
		pl6ExpectPin(t, it.row, n3, 7, 5)
	}

	// With a limit the iteration starts at self and the limit is the limit.
	{
		it := a.PageIterator(PageListDirectionRightDown, Some(b))
		expect(t, it.direction == PageListDirectionRightDown, "it.direction == .right_down")
		expect(t, it.limit.Tag == pageListPageIteratorLimitRow, "it.limit == .row")
		expect(t, it.limit.Row.Eql(b), "it.limit.row.eql(b)")
		pl6ExpectPin(t, it.row, n1, 3, 4)
		pl6ExpectChunk(t, it.Next(), n1, 3, uint(n1.data.size.rows))
	}
	{
		it := b.PageIterator(PageListDirectionLeftUp, Some(a))
		expect(t, it.direction == PageListDirectionLeftUp, "it.direction == .left_up")
		expect(t, it.limit.Row.Eql(a), "it.limit.row.eql(a)")
		pl6ExpectPin(t, it.row, n3, 7, 5)
		pl6ExpectChunk(t, it.Next(), n3, 0, 8)
	}

	// The limit may equal the pin in both directions.
	{
		it := a.PageIterator(PageListDirectionRightDown, Some(a))
		pl6ExpectChunk(t, it.Next(), n1, 3, 4)
		expect(t, !it.Next().Valid, "it.next() == null")
	}
	{
		it := a.PageIterator(PageListDirectionLeftUp, Some(a))
		pl6ExpectChunk(t, it.Next(), n1, 3, 4)
		expect(t, !it.Next().Valid, "it.next() == null")
	}
}

// ---- RowIterator / rowIterator / Pin.rowIterator ----

// pl6Drained is term51NDrained.
type pl6Drained struct {
	count uint
	first Optional[PageListPin]
	last  Optional[PageListPin]
}

// pl6DrainRows is term51NDrainRows: drain a row iterator, verifying that every
// pin has x == 0 and is exactly one row after (right_down) or before
// (left_up) the previous one, and that the exhausted iterator keeps returning
// null.
func pl6DrainRows(t *testing.T, it *PageListRowIterator, dir PageListDirection) pl6Drained {
	t.Helper()
	result := pl6Drained{count: 0, first: None[PageListPin](), last: None[PageListPin]()}
	for {
		p := it.Next()
		if !p.Valid {
			break
		}
		expectEqual(t, uint(0), uint(p.Value.x))
		if result.last.Valid {
			prev := result.last.Value
			var want Optional[PageListPin]
			switch dir {
			case PageListDirectionRightDown:
				want = prev.Down(1)
			case PageListDirectionLeftUp:
				want = prev.Up(1)
			}
			expect(t, want.Valid, "want != null")
			expect(t, want.Value.Eql(p.Value), "want.?.eql(p)")
		} else {
			result.first = p
		}
		result.last = p
		result.count += 1
	}
	expect(t, !it.chunk.Valid, "it.chunk == null")
	expect(t, !it.Next().Valid, "it.next() == null")
	expect(t, !it.Next().Valid, "it.next() == null")
	return result
}

// Zig: test "term51: RowIterator.next: right_down on one page"
func TestPl6_RowIterator_next_right_down_on_one_page(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 10)
	defer s.Deinit()
	node := s.pages.first

	// Rows 2..=5, in order, then exhausted. The x of the start pin is not used.
	start := PageListPin{node: node, y: 2, x: 7}
	it := start.RowIterator(PageListDirectionRightDown, Some(PageListPin{node: node, y: 5, x: 3}))
	pl6ExpectPin(t, it.Next(), node, 2, 0)
	pl6ExpectPin(t, it.Next(), node, 3, 0)
	pl6ExpectPin(t, it.Next(), node, 4, 0)
	pl6ExpectPin(t, it.Next(), node, 5, 0)
	expect(t, !it.Next().Valid, "it.next() == null")
	expect(t, !it.chunk.Valid, "it.chunk == null")
	expect(t, !it.Next().Valid, "it.next() == null")

	// Single row.
	one := start.RowIterator(PageListDirectionRightDown, Some(start))
	pl6ExpectPin(t, one.Next(), node, 2, 0)
	expect(t, !one.Next().Valid, "one.next() == null")
	expect(t, !one.Next().Valid, "one.next() == null")

	// First row only, and last row only.
	top := PageListPin{node: node}
	first := top.RowIterator(PageListDirectionRightDown, Some(top))
	pl6ExpectPin(t, first.Next(), node, 0, 0)
	expect(t, !first.Next().Valid, "first.next() == null")
	bottom := PageListPin{node: node, y: 9}
	last := bottom.RowIterator(PageListDirectionRightDown, Some(bottom))
	pl6ExpectPin(t, last.Next(), node, 9, 0)
	expect(t, !last.Next().Valid, "last.next() == null")

	// The whole page.
	all := top.RowIterator(PageListDirectionRightDown, Some(bottom))
	d := pl6DrainRows(t, &all, PageListDirectionRightDown)
	expectEqual(t, uint(10), d.count)
}

// Zig: test "term51: RowIterator.next: left_up on one page"
func TestPl6_RowIterator_next_left_up_on_one_page(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 10)
	defer s.Deinit()
	node := s.pages.first

	// From row 5 up to row 2 inclusive; the limit page does not start at row 0.
	start := PageListPin{node: node, y: 5, x: 7}
	it := start.RowIterator(PageListDirectionLeftUp, Some(PageListPin{node: node, y: 2, x: 3}))
	pl6ExpectPin(t, it.Next(), node, 5, 0)
	pl6ExpectPin(t, it.Next(), node, 4, 0)
	pl6ExpectPin(t, it.Next(), node, 3, 0)
	pl6ExpectPin(t, it.Next(), node, 2, 0)
	expect(t, !it.Next().Valid, "it.next() == null")
	expect(t, !it.chunk.Valid, "it.chunk == null")
	expect(t, !it.Next().Valid, "it.next() == null")

	// Single row that is not row 0.
	one := start.RowIterator(PageListDirectionLeftUp, Some(start))
	pl6ExpectPin(t, one.Next(), node, 5, 0)
	expect(t, !one.Next().Valid, "one.next() == null")
	expect(t, !one.Next().Valid, "one.next() == null")

	// Down to row 0.
	z := PageListPin{node: node, y: 2}
	to0 := z.RowIterator(PageListDirectionLeftUp, Some(PageListPin{node: node}))
	pl6ExpectPin(t, to0.Next(), node, 2, 0)
	pl6ExpectPin(t, to0.Next(), node, 1, 0)
	pl6ExpectPin(t, to0.Next(), node, 0, 0)
	expect(t, !to0.Next().Valid, "to0.next() == null")

	// Row 0 alone.
	top := PageListPin{node: node}
	r0 := top.RowIterator(PageListDirectionLeftUp, Some(top))
	pl6ExpectPin(t, r0.Next(), node, 0, 0)
	expect(t, !r0.Next().Valid, "r0.next() == null")

	// The last row alone, and the whole page.
	bottom := PageListPin{node: node, y: 9}
	last := bottom.RowIterator(PageListDirectionLeftUp, Some(bottom))
	pl6ExpectPin(t, last.Next(), node, 9, 0)
	expect(t, !last.Next().Valid, "last.next() == null")
	all := bottom.RowIterator(PageListDirectionLeftUp, Some(top))
	d := pl6DrainRows(t, &all, PageListDirectionLeftUp)
	expectEqual(t, uint(10), d.count)
}

// Zig: test "term51: RowIterator.next: right_down across pages"
func TestPl6_RowIterator_next_right_down_across_pages(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	n3 := n2.next
	r1 := uint(n1.data.size.rows)
	r2 := uint(n2.data.size.rows)

	// Over the boundary of two pages.
	a := PageListPin{node: n1, y: SizeCellCountInt(r1 - 2)}
	it := a.RowIterator(PageListDirectionRightDown, Some(PageListPin{node: n2, y: 1}))
	pl6ExpectPin(t, it.Next(), n1, r1-2, 0)
	pl6ExpectPin(t, it.Next(), n1, r1-1, 0)
	pl6ExpectPin(t, it.Next(), n2, 0, 0)
	pl6ExpectPin(t, it.Next(), n2, 1, 0)
	expect(t, !it.Next().Valid, "it.next() == null")
	expect(t, !it.Next().Valid, "it.next() == null")

	// Across a whole middle page.
	wide := a.RowIterator(PageListDirectionRightDown, Some(PageListPin{node: n3, y: 2}))
	d := pl6DrainRows(t, &wide, PageListDirectionRightDown)
	expectEqual(t, 2+r2+3, d.count)
	expect(t, d.first.Value.node == n1, "d.first.?.node == n1")
	expectEqual(t, r1-2, uint(d.first.Value.y))
	pl6ExpectPin(t, d.last, n3, 2, 0)

	// Last row of a page to the first row of the next one.
	b := PageListPin{node: n2, y: SizeCellCountInt(r2 - 1)}
	edge := b.RowIterator(PageListDirectionRightDown, Some(PageListPin{node: n3}))
	pl6ExpectPin(t, edge.Next(), n2, r2-1, 0)
	pl6ExpectPin(t, edge.Next(), n3, 0, 0)
	expect(t, !edge.Next().Valid, "edge.next() == null")

	// Limit at the end of the middle page does not touch the next one.
	mid := a.RowIterator(PageListDirectionRightDown, Some(b))
	dm := pl6DrainRows(t, &mid, PageListDirectionRightDown)
	expectEqual(t, 2+r2, dm.count)
	pl6ExpectPin(t, dm.last, n2, r2-1, 0)
}

// Zig: test "term51: RowIterator.next: left_up across pages"
func TestPl6_RowIterator_next_left_up_across_pages(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	n3 := n2.next
	r1 := uint(n1.data.size.rows)
	r2 := uint(n2.data.size.rows)

	// Over the boundary of two pages.
	a := PageListPin{node: n2, y: 1}
	it := a.RowIterator(PageListDirectionLeftUp, Some(PageListPin{node: n1, y: SizeCellCountInt(r1 - 2)}))
	pl6ExpectPin(t, it.Next(), n2, 1, 0)
	pl6ExpectPin(t, it.Next(), n2, 0, 0)
	pl6ExpectPin(t, it.Next(), n1, r1-1, 0)
	pl6ExpectPin(t, it.Next(), n1, r1-2, 0)
	expect(t, !it.Next().Valid, "it.next() == null")
	expect(t, !it.Next().Valid, "it.next() == null")

	// Across a whole middle page.
	b := PageListPin{node: n3, y: 2}
	wide := b.RowIterator(PageListDirectionLeftUp, Some(PageListPin{node: n1, y: SizeCellCountInt(r1 - 2)}))
	d := pl6DrainRows(t, &wide, PageListDirectionLeftUp)
	expectEqual(t, 3+r2+2, d.count)
	pl6ExpectPin(t, d.first, n3, 2, 0)
	pl6ExpectPin(t, d.last, n1, r1-2, 0)

	// First row of a page down to the last row of the previous one.
	c := PageListPin{node: n3}
	edge := c.RowIterator(PageListDirectionLeftUp, Some(PageListPin{node: n2, y: SizeCellCountInt(r2 - 1)}))
	pl6ExpectPin(t, edge.Next(), n3, 0, 0)
	pl6ExpectPin(t, edge.Next(), n2, r2-1, 0)
	expect(t, !edge.Next().Valid, "edge.next() == null")

	// All the way to the first row of the first page.
	all := b.RowIterator(PageListDirectionLeftUp, Some(PageListPin{node: n1}))
	da := pl6DrainRows(t, &all, PageListDirectionLeftUp)
	expectEqual(t, 3+r2+r1, da.count)
	pl6ExpectPin(t, da.last, n1, 0, 0)
}

// Zig: test "term51: Pin.rowIterator: initial chunk and offset"
func TestPl6_Pin_rowIterator_initial_chunk_and_offset(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	r1 := uint(n1.data.size.rows)

	a := PageListPin{node: n1, y: 4, x: 2}
	b := PageListPin{node: n2, y: 6, x: 5}

	// right_down starts at the first row of the first chunk.
	{
		it := a.RowIterator(PageListDirectionRightDown, Some(b))
		pl6ExpectChunk(t, it.chunk, n1, 4, r1)
		expectEqual(t, uint(4), uint(it.offset))
		expect(t, it.page_it.direction == PageListDirectionRightDown, "it.page_it.direction == .right_down")
		// The first chunk was consumed from the page iterator.
		pl6ExpectPin(t, it.page_it.row, n2, 0, 0)
	}
	// left_up starts at the last row of the first chunk.
	{
		it := b.RowIterator(PageListDirectionLeftUp, Some(a))
		pl6ExpectChunk(t, it.chunk, n2, 0, 7)
		expectEqual(t, uint(6), uint(it.offset))
		expect(t, it.page_it.direction == PageListDirectionLeftUp, "it.page_it.direction == .left_up")
		pl6ExpectPin(t, it.page_it.row, n1, r1-1, 0)
	}
	// No limit
	{
		it := a.RowIterator(PageListDirectionRightDown, None[PageListPin]())
		pl6ExpectChunk(t, it.chunk, n1, 4, r1)
		expectEqual(t, uint(4), uint(it.offset))
		expect(t, it.page_it.limit.Tag == pageListPageIteratorLimitNone, "it.page_it.limit == .none")
	}
	{
		it := b.RowIterator(PageListDirectionLeftUp, None[PageListPin]())
		pl6ExpectChunk(t, it.chunk, n2, 0, 7)
		expectEqual(t, uint(6), uint(it.offset))
	}
	// Single row: the chunk is that row in either direction.
	{
		it := a.RowIterator(PageListDirectionRightDown, Some(a))
		pl6ExpectChunk(t, it.chunk, n1, 4, 5)
		expectEqual(t, uint(4), uint(it.offset))
		it2 := a.RowIterator(PageListDirectionLeftUp, Some(a))
		pl6ExpectChunk(t, it2.chunk, n1, 4, 5)
		expectEqual(t, uint(4), uint(it2.offset))
	}
}

// Zig: test "term51: Pin.rowIterator: no limit iterates to the edge of the list"
func TestPl6_Pin_rowIterator_no_limit_iterates_to_the_edge_of_the_list(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	r1 := uint(n1.data.size.rows)
	r2 := uint(n2.data.size.rows)

	a := PageListPin{node: n1, y: SizeCellCountInt(r1 - 3), x: 6}
	down := a.RowIterator(PageListDirectionRightDown, None[PageListPin]())
	dd := pl6DrainRows(t, &down, PageListDirectionRightDown)
	expectEqual(t, 3+r2, dd.count)
	pl6ExpectPin(t, dd.last, n2, r2-1, 0)

	b := PageListPin{node: n2, y: 2, x: 6}
	up := b.RowIterator(PageListDirectionLeftUp, None[PageListPin]())
	du := pl6DrainRows(t, &up, PageListDirectionLeftUp)
	expectEqual(t, 3+r1, du.count)
	pl6ExpectPin(t, du.last, n1, 0, 0)
}

// Zig: test "term51: rowIterator: bounds directions and tags"
func TestPl6_rowIterator_bounds_directions_and_tags(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 3)
	defer s.Deinit()
	pl6GrowRows(t, &s, 1)
	n1 := s.pages.first
	n2 := n1.next
	n3 := n2.next
	n4 := n3.next
	r1 := uint32(n1.data.size.rows)
	r2 := uint32(n2.data.size.rows)
	r3 := uint32(n3.data.size.rows)
	total := uint(r1 + r2 + r3 + 1)

	// Explicit bounds in both directions: same rows in opposite order.
	{
		down := s.RowIterator(PageListDirectionRightDown, pl6Pt(1, r1-2), Some(pl6Pt(1, r1+r2+1)))
		dd := pl6DrainRows(t, &down, PageListDirectionRightDown)
		expectEqual(t, uint(2+r2+2), dd.count)
		pl6ExpectPin(t, dd.first, n1, uint(r1)-2, 0)
		pl6ExpectPin(t, dd.last, n3, 1, 0)

		up := s.RowIterator(PageListDirectionLeftUp, pl6Pt(1, r1-2), Some(pl6Pt(1, r1+r2+1)))
		du := pl6DrainRows(t, &up, PageListDirectionLeftUp)
		expectEqual(t, dd.count, du.count)
		expect(t, du.first.Value.Eql(dd.last.Value), "du.first.?.eql(dd.last.?)")
		expect(t, du.last.Value.Eql(dd.first.Value), "du.last.?.eql(dd.first.?)")
	}

	// Single row
	{
		down := s.RowIterator(PageListDirectionRightDown, pl6Pt(3, r1), Some(pl6Pt(3, r1)))
		pl6ExpectPin(t, down.Next(), n2, 0, 0)
		expect(t, !down.Next().Valid, "down.next() == null")
		up := s.RowIterator(PageListDirectionLeftUp, pl6Pt(3, r1), Some(pl6Pt(3, r1)))
		pl6ExpectPin(t, up.Next(), n2, 0, 0)
		expect(t, !up.Next().Valid, "up.next() == null")
	}

	// No bottom bound: the end of the area of the tag.
	for _, dir := range pl6Dirs {
		screen := s.RowIterator(dir, Point{Tag: PointTagScreen}, None[Point]())
		ds := pl6DrainRows(t, &screen, dir)
		expectEqual(t, total, ds.count)

		active := s.RowIterator(dir, Point{Tag: PointTagActive}, None[Point]())
		da := pl6DrainRows(t, &active, dir)
		expectEqual(t, uint(3), da.count)

		history := s.RowIterator(dir, Point{Tag: PointTagHistory}, None[Point]())
		dh := pl6DrainRows(t, &history, dir)
		expectEqual(t, total-3, dh.count)

		viewport := s.RowIterator(dir, Point{Tag: PointTagViewport}, None[Point]())
		dv := pl6DrainRows(t, &viewport, dir)
		expectEqual(t, uint(3), dv.count)

		// The first and last rows of each.
		lo := pl6If(dir == PageListDirectionRightDown, ds.first, ds.last)
		hi := pl6If(dir == PageListDirectionRightDown, ds.last, ds.first)
		pl6ExpectPin(t, lo, n1, 0, 0)
		pl6ExpectPin(t, hi, n4, 0, 0)
		alo := pl6If(dir == PageListDirectionRightDown, da.first, da.last)
		ahi := pl6If(dir == PageListDirectionRightDown, da.last, da.first)
		pl6ExpectPin(t, alo, n3, uint(r3)-2, 0)
		pl6ExpectPin(t, ahi, n4, 0, 0)
		hlo := pl6If(dir == PageListDirectionRightDown, dh.first, dh.last)
		hhi := pl6If(dir == PageListDirectionRightDown, dh.last, dh.first)
		pl6ExpectPin(t, hlo, n1, 0, 0)
		pl6ExpectPin(t, hhi, n3, uint(r3)-3, 0)
	}

	// Scrolled viewport
	s.Scroll(PageListScroll{Tag: PageListScrollRow, Row: uint(r1) + 5})
	{
		viewport := s.RowIterator(PageListDirectionRightDown, Point{Tag: PointTagViewport}, None[Point]())
		dv := pl6DrainRows(t, &viewport, PageListDirectionRightDown)
		expectEqual(t, uint(3), dv.count)
		pl6ExpectPin(t, dv.first, n2, 5, 0)
		pl6ExpectPin(t, dv.last, n2, 7, 0)
	}
}

// Zig: test "term51: rowIterator: history without history is empty"
func TestPl6_rowIterator_history_without_history_is_empty(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 5)
	defer s.Deinit()

	for _, dir := range pl6Dirs {
		it := s.RowIterator(dir, Point{Tag: PointTagHistory}, None[Point]())
		expect(t, !it.chunk.Valid, "it.chunk == null")
		expect(t, !it.Next().Valid, "it.next() == null")
		expect(t, !it.Next().Valid, "it.next() == null")
	}

	// With one row of history there is one row.
	pl6GrowRows(t, &s, 1)
	for _, dir := range pl6Dirs {
		it := s.RowIterator(dir, Point{Tag: PointTagHistory}, None[Point]())
		d := pl6DrainRows(t, &it, dir)
		expectEqual(t, uint(1), d.count)
		pl6ExpectPin(t, d.first, s.pages.first, 0, 0)
	}
}

// ---- CellIterator / cellIterator / Pin.cellIterator ----

// pl6ExpPin is term51NExpPin.
type pl6ExpPin struct {
	node *PageListNode
	y    uint
	x    uint
}

// pl6CheckCells is term51NCheckCells: drain a cell iterator into `out`
// (exhaustion is verified) and check that it produced exactly the expected
// pins in order.
func pl6CheckCells(t *testing.T, it *PageListCellIterator, expected []pl6ExpPin) {
	t.Helper()
	for _, e := range expected {
		pl6ExpectPin(t, it.Next(), e.node, e.y, e.x)
	}
	expect(t, !it.cell.Valid, "it.cell == null")
	expect(t, !it.Next().Valid, "it.next() == null")
	expect(t, !it.Next().Valid, "it.next() == null")
}

// pl6CellDrain is term51NCellDrain.
type pl6CellDrain struct {
	count uint
	first Optional[PageListPin]
	last  Optional[PageListPin]
}

// pl6DrainCells is term51NDrainCells: drain a cell iterator checking that
// consecutive pins are adjacent cells (in row-major screen order, forwards or
// backwards) and that the iterator stays exhausted.
func pl6DrainCells(t *testing.T, s *PageList, it *PageListCellIterator, dir PageListDirection) pl6CellDrain {
	t.Helper()
	result := pl6CellDrain{count: 0, first: None[PageListPin](), last: None[PageListPin]()}
	prev := None[int64]()
	for {
		p := it.Next()
		if !p.Valid {
			break
		}
		pt := zigUnwrap(t, s.PointFromPin(PointTagScreen, p.Value))
		lin := int64(pt.Screen.y)*int64(s.cols) + int64(pt.Screen.x)
		if prev.Valid {
			q := prev.Value
			var want int64
			switch dir {
			case PageListDirectionRightDown:
				want = q + 1
			case PageListDirectionLeftUp:
				want = q - 1
			}
			expectEqual(t, want, lin)
		} else {
			result.first = p
		}
		prev = Some(lin)
		result.last = p
		result.count += 1
	}
	expect(t, !it.cell.Valid, "it.cell == null")
	expect(t, !it.Next().Valid, "it.next() == null")
	expect(t, !it.Next().Valid, "it.next() == null")
	return result
}

// pl6If is Zig's `if (c) a else b` expression.
func pl6If[T any](c bool, a T, b T) T {
	if c {
		return a
	}
	return b
}

// Zig: test "term51: CellIterator.next: right_down walks rows left to right"
func TestPl6_CellIterator_next_right_down_walks_rows_left_to_right(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 5)
	defer s.Deinit()
	node := s.pages.first

	// The x of the start applies to the first row only and the x of the
	// limit is ignored: the last row is walked to its last column.
	start := PageListPin{node: node, y: 1, x: 1}
	it := start.CellIterator(PageListDirectionRightDown, Some(PageListPin{node: node, y: 2, x: 2}))
	pl6CheckCells(t, &it, []pl6ExpPin{
		{node: node, y: 1, x: 1},
		{node: node, y: 1, x: 2},
		{node: node, y: 1, x: 3},
		{node: node, y: 2, x: 0},
		{node: node, y: 2, x: 1},
		{node: node, y: 2, x: 2},
		{node: node, y: 2, x: 3},
	})

	// Starting on the last column of the first row.
	last_col := PageListPin{node: node, y: 3, x: 3}
	it2 := last_col.CellIterator(PageListDirectionRightDown, Some(PageListPin{node: node, y: 4}))
	pl6CheckCells(t, &it2, []pl6ExpPin{
		{node: node, y: 3, x: 3},
		{node: node, y: 4, x: 0},
		{node: node, y: 4, x: 1},
		{node: node, y: 4, x: 2},
		{node: node, y: 4, x: 3},
	})

	// The state before the first call is the first cell.
	it3 := start.CellIterator(PageListDirectionRightDown, Some(start))
	pl6ExpectPin(t, it3.cell, node, 1, 1)
	_ = zigUnwrap(t, it3.Next())
	pl6ExpectPin(t, it3.cell, node, 1, 2)
}

// Zig: test "term51: CellIterator.next: left_up walks rows right to left"
func TestPl6_CellIterator_next_left_up_walks_rows_right_to_left(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 5)
	defer s.Deinit()
	node := s.pages.first

	// The x of the start applies to the first row only; the x of the limit
	// is ignored: the limit row is walked to its first column. Rows above
	// the first are entered at their last column.
	start := PageListPin{node: node, y: 2, x: 2}
	it := start.CellIterator(PageListDirectionLeftUp, Some(PageListPin{node: node, y: 1, x: 1}))
	pl6CheckCells(t, &it, []pl6ExpPin{
		{node: node, y: 2, x: 2},
		{node: node, y: 2, x: 1},
		{node: node, y: 2, x: 0},
		{node: node, y: 1, x: 3},
		{node: node, y: 1, x: 2},
		{node: node, y: 1, x: 1},
		{node: node, y: 1, x: 0},
	})

	// Starting on the first column.
	first_col := PageListPin{node: node, y: 4, x: 0}
	it2 := first_col.CellIterator(PageListDirectionLeftUp, Some(PageListPin{node: node, y: 3, x: 3}))
	pl6CheckCells(t, &it2, []pl6ExpPin{
		{node: node, y: 4, x: 0},
		{node: node, y: 3, x: 3},
		{node: node, y: 3, x: 2},
		{node: node, y: 3, x: 1},
		{node: node, y: 3, x: 0},
	})

	// The state before the first call is the first cell.
	it3 := start.CellIterator(PageListDirectionLeftUp, Some(start))
	pl6ExpectPin(t, it3.cell, node, 2, 2)
	_ = zigUnwrap(t, it3.Next())
	pl6ExpectPin(t, it3.cell, node, 2, 1)
}

// Zig: test "term51: CellIterator.next: single cell ranges cover the rest of the row"
func TestPl6_CellIterator_next_single_cell_ranges_cover_the_rest_of_the_row(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 5)
	defer s.Deinit()
	node := s.pages.first

	// right_down from a single cell goes to the end of its row.
	down := s.CellIterator(PageListDirectionRightDown, pl6Pt(2, 1), Some(pl6Pt(2, 1)))
	pl6CheckCells(t, &down, []pl6ExpPin{
		{node: node, y: 1, x: 2},
		{node: node, y: 1, x: 3},
	})
	// left_up from a single cell goes to the start of its row.
	up := s.CellIterator(PageListDirectionLeftUp, pl6Pt(2, 1), Some(pl6Pt(2, 1)))
	pl6CheckCells(t, &up, []pl6ExpPin{
		{node: node, y: 1, x: 2},
		{node: node, y: 1, x: 1},
		{node: node, y: 1, x: 0},
	})

	// The very first and the very last cell.
	d0 := s.CellIterator(PageListDirectionRightDown, pl6Pt(0, 0), Some(pl6Pt(0, 0)))
	pl6CheckCells(t, &d0, []pl6ExpPin{
		{node: node, y: 0, x: 0},
		{node: node, y: 0, x: 1},
		{node: node, y: 0, x: 2},
		{node: node, y: 0, x: 3},
	})
	cell_up0 := s.CellIterator(PageListDirectionLeftUp, pl6Pt(0, 0), Some(pl6Pt(0, 0)))
	pl6CheckCells(t, &cell_up0, []pl6ExpPin{{node: node, y: 0, x: 0}})
	d1 := s.CellIterator(PageListDirectionRightDown, pl6Pt(3, 4), Some(pl6Pt(3, 4)))
	pl6CheckCells(t, &d1, []pl6ExpPin{{node: node, y: 4, x: 3}})
	cell_up1 := s.CellIterator(PageListDirectionLeftUp, pl6Pt(3, 4), Some(pl6Pt(3, 4)))
	pl6CheckCells(t, &cell_up1, []pl6ExpPin{
		{node: node, y: 4, x: 3},
		{node: node, y: 4, x: 2},
		{node: node, y: 4, x: 1},
		{node: node, y: 4, x: 0},
	})
}

// Zig: test "term51: CellIterator.next: whole screen in both directions"
func TestPl6_CellIterator_next_whole_screen_in_both_directions(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 5)
	defer s.Deinit()

	down := s.CellIterator(PageListDirectionRightDown, Point{Tag: PointTagScreen}, None[Point]())
	dd := pl6DrainCells(t, &s, &down, PageListDirectionRightDown)
	expectEqual(t, uint(20), dd.count)
	expectEqual(t, pl6Pt(0, 0), zigUnwrap(t, s.PointFromPin(PointTagScreen, zigUnwrap(t, dd.first))))
	expectEqual(t, pl6Pt(3, 4), zigUnwrap(t, s.PointFromPin(PointTagScreen, zigUnwrap(t, dd.last))))

	up := s.CellIterator(PageListDirectionLeftUp, Point{Tag: PointTagScreen}, None[Point]())
	du := pl6DrainCells(t, &s, &up, PageListDirectionLeftUp)
	expectEqual(t, uint(20), du.count)
	expectEqual(t, pl6Pt(3, 4), zigUnwrap(t, s.PointFromPin(PointTagScreen, zigUnwrap(t, du.first))))
	expectEqual(t, pl6Pt(0, 0), zigUnwrap(t, s.PointFromPin(PointTagScreen, zigUnwrap(t, du.last))))
}

// Zig: test "term51: CellIterator.next: one column pages move to the next row on every call"
func TestPl6_CellIterator_next_one_column_pages_move_to_the_next_row_on_every_call(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 1, 4)
	defer s.Deinit()
	node := s.pages.first

	down := s.CellIterator(PageListDirectionRightDown, pl6Pt(0, 1), Some(pl6Pt(0, 2)))
	pl6CheckCells(t, &down, []pl6ExpPin{
		{node: node, y: 1, x: 0},
		{node: node, y: 2, x: 0},
	})
	up := s.CellIterator(PageListDirectionLeftUp, pl6Pt(0, 1), Some(pl6Pt(0, 2)))
	pl6CheckCells(t, &up, []pl6ExpPin{
		{node: node, y: 2, x: 0},
		{node: node, y: 1, x: 0},
	})
	one := s.CellIterator(PageListDirectionRightDown, pl6Pt(0, 3), Some(pl6Pt(0, 3)))
	pl6CheckCells(t, &one, []pl6ExpPin{{node: node, y: 3, x: 0}})
	one_up := s.CellIterator(PageListDirectionLeftUp, pl6Pt(0, 0), Some(pl6Pt(0, 0)))
	pl6CheckCells(t, &one_up, []pl6ExpPin{{node: node, y: 0, x: 0}})
}

// Zig: test "term51: CellIterator.next: across a page boundary"
func TestPl6_CellIterator_next_across_a_page_boundary(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 3, 3, 2)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	last := n1.data.size.rows - 1

	// right_down: the end of the first page then the start of the second.
	a := PageListPin{node: n1, y: last, x: 1}
	down := a.CellIterator(PageListDirectionRightDown, Some(PageListPin{node: n2, y: 1}))
	pl6CheckCells(t, &down, []pl6ExpPin{
		{node: n1, y: uint(last), x: 1},
		{node: n1, y: uint(last), x: 2},
		{node: n2, y: 0, x: 0},
		{node: n2, y: 0, x: 1},
		{node: n2, y: 0, x: 2},
		{node: n2, y: 1, x: 0},
		{node: n2, y: 1, x: 1},
		{node: n2, y: 1, x: 2},
	})

	// left_up: the start of the second page then the end of the first.
	b := PageListPin{node: n2, y: 0, x: 1}
	up := b.CellIterator(PageListDirectionLeftUp, Some(PageListPin{node: n1, y: last}))
	pl6CheckCells(t, &up, []pl6ExpPin{
		{node: n2, y: 0, x: 1},
		{node: n2, y: 0, x: 0},
		{node: n1, y: uint(last), x: 2},
		{node: n1, y: uint(last), x: 1},
		{node: n1, y: uint(last), x: 0},
	})

	// Exactly one row on each side of the boundary through points.
	r1 := uint32(n1.data.size.rows)
	pts := s.CellIterator(PageListDirectionRightDown, pl6Pt(2, r1-1), Some(pl6Pt(0, r1)))
	pl6CheckCells(t, &pts, []pl6ExpPin{
		{node: n1, y: uint(last), x: 2},
		{node: n2, y: 0, x: 0},
		{node: n2, y: 0, x: 1},
		{node: n2, y: 0, x: 2},
	})
	pts_up := s.CellIterator(PageListDirectionLeftUp, pl6Pt(2, r1-1), Some(pl6Pt(0, r1)))
	pl6CheckCells(t, &pts_up, []pl6ExpPin{
		{node: n2, y: 0, x: 0},
		{node: n1, y: uint(last), x: 2},
		{node: n1, y: uint(last), x: 1},
		{node: n1, y: uint(last), x: 0},
	})
}

// Zig: test "term51: cellIterator: no bottom bound iterates to the end of the tag area"
func TestPl6_cellIterator_no_bottom_bound_iterates_to_the_end_of_the_tag_area(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 3)
	defer s.Deinit()
	pl6GrowRows(t, &s, 2)
	// 5 rows: 2 of history and 3 active.

	for _, dir := range pl6Dirs {
		fwd := dir == PageListDirectionRightDown
		screen := s.CellIterator(dir, Point{Tag: PointTagScreen}, None[Point]())
		ds := pl6DrainCells(t, &s, &screen, dir)
		expectEqual(t, uint(20), ds.count)
		expectEqual(t,
			pl6If(fwd, pl6Pt(0, 0), pl6Pt(3, 4)),
			zigUnwrap(t, s.PointFromPin(PointTagScreen, zigUnwrap(t, ds.first))),
		)
		expectEqual(t,
			pl6If(fwd, pl6Pt(3, 4), pl6Pt(0, 0)),
			zigUnwrap(t, s.PointFromPin(PointTagScreen, zigUnwrap(t, ds.last))),
		)

		active := s.CellIterator(dir, Point{Tag: PointTagActive}, None[Point]())
		da := pl6DrainCells(t, &s, &active, dir)
		expectEqual(t, uint(12), da.count)
		expectEqual(t,
			pl6If(fwd, pl6Pt(0, 2), pl6Pt(3, 4)),
			zigUnwrap(t, s.PointFromPin(PointTagScreen, zigUnwrap(t, da.first))),
		)
		expectEqual(t,
			pl6If(fwd, pl6Pt(3, 4), pl6Pt(0, 2)),
			zigUnwrap(t, s.PointFromPin(PointTagScreen, zigUnwrap(t, da.last))),
		)

		history := s.CellIterator(dir, Point{Tag: PointTagHistory}, None[Point]())
		dh := pl6DrainCells(t, &s, &history, dir)
		expectEqual(t, uint(8), dh.count)
		expectEqual(t,
			pl6If(fwd, pl6Pt(0, 0), pl6Pt(3, 1)),
			zigUnwrap(t, s.PointFromPin(PointTagScreen, zigUnwrap(t, dh.first))),
		)
		expectEqual(t,
			pl6If(fwd, pl6Pt(3, 1), pl6Pt(0, 0)),
			zigUnwrap(t, s.PointFromPin(PointTagScreen, zigUnwrap(t, dh.last))),
		)

		viewport := s.CellIterator(dir, Point{Tag: PointTagViewport}, None[Point]())
		dv := pl6DrainCells(t, &s, &viewport, dir)
		expectEqual(t, uint(12), dv.count)
	}

	// A viewport scrolled to the top covers the first three rows.
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	top := s.CellIterator(PageListDirectionRightDown, Point{Tag: PointTagViewport}, None[Point]())
	dt := pl6DrainCells(t, &s, &top, PageListDirectionRightDown)
	expectEqual(t, uint(12), dt.count)
	expectEqual(t, pl6Pt(0, 0), zigUnwrap(t, s.PointFromPin(PointTagScreen, zigUnwrap(t, dt.first))))
	expectEqual(t, pl6Pt(3, 2), zigUnwrap(t, s.PointFromPin(PointTagScreen, zigUnwrap(t, dt.last))))
}

// Zig: test "term51: cellIterator: explicit bounds in both directions"
func TestPl6_cellIterator_explicit_bounds_in_both_directions(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 5)
	defer s.Deinit()

	down := s.CellIterator(PageListDirectionRightDown, pl6Pt(2, 1), Some(pl6Pt(1, 3)))
	dd := pl6DrainCells(t, &s, &down, PageListDirectionRightDown)
	// (2,1) to the end of row 1: 2, rows 2: 4, row 3: 4
	expectEqual(t, uint(10), dd.count)
	expectEqual(t, pl6Pt(2, 1), zigUnwrap(t, s.PointFromPin(PointTagScreen, zigUnwrap(t, dd.first))))
	expectEqual(t, pl6Pt(3, 3), zigUnwrap(t, s.PointFromPin(PointTagScreen, zigUnwrap(t, dd.last))))

	up := s.CellIterator(PageListDirectionLeftUp, pl6Pt(2, 1), Some(pl6Pt(1, 3)))
	du := pl6DrainCells(t, &s, &up, PageListDirectionLeftUp)
	// (1,3) to the start of row 3: 2, row 2: 4, row 1: 4
	expectEqual(t, uint(10), du.count)
	expectEqual(t, pl6Pt(1, 3), zigUnwrap(t, s.PointFromPin(PointTagScreen, zigUnwrap(t, du.first))))
	expectEqual(t, pl6Pt(0, 1), zigUnwrap(t, s.PointFromPin(PointTagScreen, zigUnwrap(t, du.last))))

	// Bounds in the active tag.
	act := s.CellIterator(PageListDirectionRightDown,
		Point{Tag: PointTagActive, Active: PointCoordinate{x: 3, y: 0}},
		Some(Point{Tag: PointTagActive, Active: PointCoordinate{x: 0, y: 1}}))
	da := pl6DrainCells(t, &s, &act, PageListDirectionRightDown)
	expectEqual(t, uint(5), da.count)
}

// Zig: test "term51: cellIterator: history without history is empty"
func TestPl6_cellIterator_history_without_history_is_empty(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 3)
	defer s.Deinit()

	for _, dir := range pl6Dirs {
		it := s.CellIterator(dir, Point{Tag: PointTagHistory}, None[Point]())
		expect(t, !it.cell.Valid, "it.cell == null")
		expect(t, !it.Next().Valid, "it.next() == null")
		expect(t, !it.Next().Valid, "it.next() == null")
	}

	// With history there are cells.
	pl6GrowRows(t, &s, 1)
	it := s.CellIterator(PageListDirectionRightDown, Point{Tag: PointTagHistory}, None[Point]())
	d := pl6DrainCells(t, &s, &it, PageListDirectionRightDown)
	expectEqual(t, uint(4), d.count)
}

// Zig: test "term51: Pin.cellIterator: first cell uses the x of the pin"
func TestPl6_Pin_cellIterator_first_cell_uses_the_x_of_the_pin(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next

	a := PageListPin{node: n1, y: 3, x: 6}
	b := PageListPin{node: n2, y: 4, x: 2}

	// right_down: the first cell is `a`, row_it holds the rest.
	{
		it := a.CellIterator(PageListDirectionRightDown, Some(b))
		pl6ExpectPin(t, it.cell, n1, 3, 6)
		expect(t, it.row_it.page_it.direction == PageListDirectionRightDown, "it.row_it.page_it.direction == .right_down")
		// The first row was consumed.
		expectEqual(t, uint(4), uint(it.row_it.offset))
	}
	// left_up: the first cell is `b`.
	{
		it := b.CellIterator(PageListDirectionLeftUp, Some(a))
		pl6ExpectPin(t, it.cell, n2, 4, 2)
		expect(t, it.row_it.page_it.direction == PageListDirectionLeftUp, "it.row_it.page_it.direction == .left_up")
		expectEqual(t, uint(3), uint(it.row_it.offset))
	}
	// No limit.
	{
		it := a.CellIterator(PageListDirectionRightDown, None[PageListPin]())
		pl6ExpectPin(t, it.cell, n1, 3, 6)
		pl6ExpectPin(t, it.Next(), n1, 3, 6)
		pl6ExpectPin(t, it.Next(), n1, 3, 7)
	}
	{
		it := b.CellIterator(PageListDirectionLeftUp, None[PageListPin]())
		pl6ExpectPin(t, it.cell, n2, 4, 2)
		pl6ExpectPin(t, it.Next(), n2, 4, 2)
		pl6ExpectPin(t, it.Next(), n2, 4, 1)
	}
}

// ---- PromptIterator ----

// pl6DrainPrompts is term51NDrainPrompts: drain a prompt iterator into `out`,
// verifying that every pin is on column 0 and that the exhausted iterator
// stays exhausted. Returns the number of pins; `out` receives their rows.
func pl6DrainPrompts(t *testing.T, it *PageListPromptIterator, out []uint) uint {
	t.Helper()
	var n uint = 0
	for {
		p := it.Next()
		if !p.Valid {
			break
		}
		expectEqual(t, uint(0), uint(p.Value.x))
		expect(t, n < uint(len(out)), "n < out.len")
		out[n] = uint(p.Value.y)
		n += 1
	}
	expect(t, !it.current.Valid, "it.current == null")
	expect(t, !it.Next().Valid, "it.next() == null")
	expect(t, !it.Next().Valid, "it.next() == null")
	return n
}

// pl6ExpectRows is term51NExpectRows.
func pl6ExpectRows(t *testing.T, it *PageListPromptIterator, expected []uint) {
	t.Helper()
	var buf [32]uint
	n := pl6DrainPrompts(t, it, buf[:])
	expectEqualSlices(t, expected, buf[0:n])
}

// pl6Semantic is term51NSemantic: set the semantic_prompt of rows of a page
// from a pattern, one character per row starting at y=0: 'P' prompt, 'c'
// prompt continuation, anything else none. Every row of the page is written
// (rows past the pattern are none).
func pl6Semantic(page *Page, pattern string) {
	// Rows past the pattern are reset to none.
	for y := uint(0); y < uint(page.size.rows); y++ {
		page.GetRowAndCell(0, y).row.semantic_prompt = PageRowSemanticPromptNone
	}
	for y := 0; y < len(pattern); y++ {
		c := pattern[y]
		row := page.GetRowAndCell(0, uint(y)).row
		switch c {
		case 'P':
			row.semantic_prompt = PageRowSemanticPromptPrompt
		case 'c':
			row.semantic_prompt = PageRowSemanticPromptPromptContinuation
		default:
			row.semantic_prompt = PageRowSemanticPromptNone
		}
	}
}

// pl6Cells is term51NCells: write the cells of row y from a pattern, one
// character per column starting at x=0: 'p' prompt with text, 'i' input with
// text, 'o' output with text, '.' empty output cell, 'q' prompt without text,
// 'n' input without text.
func pl6Cells(page *Page, y uint, pattern string) {
	for x := uint(0); x < uint(page.size.cols); x++ {
		var c byte
		if x < uint(len(pattern)) {
			c = pattern[x]
		} else {
			c = '.'
		}
		cell := page.GetRowAndCell(x, y).cell
		switch c {
		case 'p':
			*cell = PageCell{
				content_tag:      PageCellContentTagCodepoint,
				content:          PageCellContent{codepoint: '$'},
				semantic_content: PageCellSemanticContentPrompt,
			}
		case 'i':
			*cell = PageCell{
				content_tag:      PageCellContentTagCodepoint,
				content:          PageCellContent{codepoint: 'i'},
				semantic_content: PageCellSemanticContentInput,
			}
		case 'o':
			*cell = PageCell{
				content_tag:      PageCellContentTagCodepoint,
				content:          PageCellContent{codepoint: 'o'},
				semantic_content: PageCellSemanticContentOutput,
			}
		case 'q':
			*cell = PageCell{semantic_content: PageCellSemanticContentPrompt}
		case 'n':
			*cell = PageCell{semantic_content: PageCellSemanticContentInput}
		default:
			*cell = PageCell{}
		}
	}
}

// Zig: test "term51: PromptIterator.next: empty iterator and direction dispatch"
func TestPl6_PromptIterator_next_empty_iterator_and_direction_dispatch(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 20)
	defer s.Deinit()
	node := s.pages.first
	pl6Semantic(&node.data, "..Pcc.PPc.ccP")

	// The constant empty iterator.
	empty := PageListPromptIteratorEmpty
	expect(t, !empty.current.Valid, "empty.current == null")
	expect(t, !empty.limit.Valid, "empty.limit == null")
	expect(t, empty.direction == PageListDirectionLeftUp, "empty.direction == .left_up")
	expect(t, !empty.Next().Valid, "empty.next() == null")
	expect(t, !empty.NextRightDown().Valid, "empty.nextRightDown() == null")
	expect(t, !empty.NextLeftUp().Valid, "empty.nextLeftUp() == null")
	empty_down := PageListPromptIterator{current: None[PageListPin](), limit: None[PageListPin](), direction: PageListDirectionRightDown}
	expect(t, !empty_down.Next().Valid, "empty_down.next() == null")

	// next() is nextRightDown for right_down and nextLeftUp for left_up.
	top := PageListPin{node: node}
	bottom := PageListPin{node: node, y: 19}
	a := top.PromptIterator(PageListDirectionRightDown, None[PageListPin]())
	b := a
	c := bottom.PromptIterator(PageListDirectionLeftUp, None[PageListPin]())
	d := c
	for i := 0; i < 6; i++ {
		pa := a.Next()
		pb := b.NextRightDown()
		expectEqual(t, !pa.Valid, !pb.Valid)
		if pa.Valid {
			expect(t, pa.Value.Eql(pb.Value), "p.eql(pb.?)")
		}
		expectEqual(t, !a.current.Valid, !b.current.Valid)

		pc := c.Next()
		pd := d.NextLeftUp()
		expectEqual(t, !pc.Valid, !pd.Valid)
		if pc.Valid {
			expect(t, pc.Value.Eql(pd.Value), "p.eql(pd.?)")
		}
		expectEqual(t, !c.current.Valid, !d.current.Valid)
	}
}

// Zig: test "term51: PromptIterator.next: both directions find the same prompts"
func TestPl6_PromptIterator_next_both_directions_find_the_same_prompts(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 20)
	defer s.Deinit()
	node := s.pages.first
	pl6Semantic(&node.data, "..Pcc.PPc.ccP")

	top := PageListPin{node: node}
	bottom := PageListPin{node: node, y: 19}
	down := top.PromptIterator(PageListDirectionRightDown, None[PageListPin]())
	pl6ExpectRows(t, &down, []uint{2, 6, 7, 10, 12})
	up := bottom.PromptIterator(PageListDirectionLeftUp, None[PageListPin]())
	pl6ExpectRows(t, &up, []uint{12, 10, 7, 6, 2})
}

// Zig: test "term51: PromptIterator.nextRightDown: no prompts and exhausted iterators"
func TestPl6_PromptIterator_nextRightDown_no_prompts_and_exhausted_iterators(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 10)
	defer s.Deinit()
	node := s.pages.first

	// No prompt rows at all: the whole list is scanned.
	top := PageListPin{node: node}
	it := top.PromptIterator(PageListDirectionRightDown, None[PageListPin]())
	expect(t, !it.NextRightDown().Valid, "it.nextRightDown() == null")
	expect(t, !it.current.Valid, "it.current == null")
	expect(t, !it.NextRightDown().Valid, "it.nextRightDown() == null")

	// A limit that is never hit.
	lim := top.PromptIterator(PageListDirectionRightDown, Some(PageListPin{node: node, y: 5}))
	expect(t, !lim.NextRightDown().Valid, "lim.nextRightDown() == null")
	expect(t, !lim.current.Valid, "lim.current == null")

	// Started on the last row.
	bottom := PageListPin{node: node, y: 9}
	last := bottom.PromptIterator(PageListDirectionRightDown, None[PageListPin]())
	expect(t, !last.NextRightDown().Valid, "last.nextRightDown() == null")
	expect(t, !last.current.Valid, "last.current == null")
}

// Zig: test "term51: PromptIterator.nextRightDown: prompts and continuations in order"
func TestPl6_PromptIterator_nextRightDown_prompts_and_continuations_in_order(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 20)
	defer s.Deinit()
	node := s.pages.first
	pl6Semantic(&node.data, "..Pcc.PPc.ccP")

	top := PageListPin{node: node}
	it := top.PromptIterator(PageListDirectionRightDown, None[PageListPin]())

	// Prompt at 2 with continuations 3 and 4, resumes at the next row.
	pl6ExpectPin(t, it.NextRightDown(), node, 2, 0)
	pl6ExpectPin(t, it.current, node, 5, 0)
	// Prompt at 6 directly followed by another prompt: resume on it.
	pl6ExpectPin(t, it.NextRightDown(), node, 6, 0)
	pl6ExpectPin(t, it.current, node, 7, 0)
	// Prompt at 7 with a continuation at 8.
	pl6ExpectPin(t, it.NextRightDown(), node, 7, 0)
	pl6ExpectPin(t, it.current, node, 9, 0)
	// Continuations 10 and 11 without a prompt, the first is returned.
	pl6ExpectPin(t, it.NextRightDown(), node, 10, 0)
	pl6ExpectPin(t, it.current, node, 12, 0)
	// Prompt at 12 followed by a non prompt row.
	pl6ExpectPin(t, it.NextRightDown(), node, 12, 0)
	pl6ExpectPin(t, it.current, node, 13, 0)
	// The rest has no prompts.
	expect(t, !it.NextRightDown().Valid, "it.nextRightDown() == null")
	expect(t, !it.current.Valid, "it.current == null")
	expect(t, !it.NextRightDown().Valid, "it.nextRightDown() == null")
}

// Zig: test "term51: PromptIterator.nextRightDown: returned pins are on column zero"
func TestPl6_PromptIterator_nextRightDown_returned_pins_are_on_column_zero(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 20)
	defer s.Deinit()
	node := s.pages.first
	pl6Semantic(&node.data, "..Pcc.P")

	// The start x is carried in the resume pin but not in returned pins.
	start := PageListPin{node: node, y: 0, x: 3}
	it := start.PromptIterator(PageListDirectionRightDown, None[PageListPin]())
	pl6ExpectPin(t, it.NextRightDown(), node, 2, 0)
	pl6ExpectPin(t, it.current, node, 5, 3)
	pl6ExpectPin(t, it.NextRightDown(), node, 6, 0)
	expect(t, !it.NextRightDown().Valid, "it.nextRightDown() == null")

	// Starting on a prompt row at x > 0 returns column 0 of the row.
	mid := PageListPin{node: node, y: 2, x: 2}
	it2 := mid.PromptIterator(PageListDirectionRightDown, None[PageListPin]())
	pl6ExpectPin(t, it2.NextRightDown(), node, 2, 0)
}

// Zig: test "term51: PromptIterator.nextRightDown: limit on a non prompt row"
func TestPl6_PromptIterator_nextRightDown_limit_on_a_non_prompt_row(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 20)
	defer s.Deinit()
	node := s.pages.first
	pl6Semantic(&node.data, "..P..")

	// The limit stops the scan before the prompt below it.
	top := PageListPin{node: node}
	it := top.PromptIterator(PageListDirectionRightDown, Some(PageListPin{node: node, y: 1}))
	expect(t, !it.NextRightDown().Valid, "it.nextRightDown() == null")
	expect(t, !it.current.Valid, "it.current == null")

	// A limit equal to the start on a non prompt row.
	one := PageListPin{node: node, y: 1}
	it2 := one.PromptIterator(PageListDirectionRightDown, Some(one))
	expect(t, !it2.NextRightDown().Valid, "it2.nextRightDown() == null")
	expect(t, !it2.current.Valid, "it2.current == null")

	// A limit after a prompt on a non prompt row: the prompt is found, then
	// the following scan stops on the limit.
	it3 := top.PromptIterator(PageListDirectionRightDown, Some(PageListPin{node: node, y: 4}))
	pl6ExpectPin(t, it3.NextRightDown(), node, 2, 0)
	pl6ExpectPin(t, it3.current, node, 3, 0)
	expect(t, !it3.NextRightDown().Valid, "it3.nextRightDown() == null")
	expect(t, !it3.current.Valid, "it3.current == null")
}

// Zig: test "term51: PromptIterator.nextRightDown: limit on a prompt or continuation row"
func TestPl6_PromptIterator_nextRightDown_limit_on_a_prompt_or_continuation_row(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 20)
	defer s.Deinit()
	node := s.pages.first
	top := PageListPin{node: node}

	// Prompt at the limit is returned, and the iterator is finished even
	// though continuation rows follow.
	pl6Semantic(&node.data, "..Pcc..")
	a := top.PromptIterator(PageListDirectionRightDown, Some(PageListPin{node: node, y: 2}))
	pl6ExpectPin(t, a.NextRightDown(), node, 2, 0)
	expect(t, !a.current.Valid, "a.current == null")
	expect(t, !a.NextRightDown().Valid, "a.nextRightDown() == null")

	// A limit in the middle of the continuation rows ends the scan.
	pl6Semantic(&node.data, "..Pcccc..")
	b := top.PromptIterator(PageListDirectionRightDown, Some(PageListPin{node: node, y: 4}))
	pl6ExpectPin(t, b.NextRightDown(), node, 2, 0)
	expect(t, !b.current.Valid, "b.current == null")
	expect(t, !b.NextRightDown().Valid, "b.nextRightDown() == null")

	// On the first continuation row.
	b2 := top.PromptIterator(PageListDirectionRightDown, Some(PageListPin{node: node, y: 3}))
	pl6ExpectPin(t, b2.NextRightDown(), node, 2, 0)
	expect(t, !b2.current.Valid, "b2.current == null")

	// On the last continuation row.
	b3 := top.PromptIterator(PageListDirectionRightDown, Some(PageListPin{node: node, y: 6}))
	pl6ExpectPin(t, b3.NextRightDown(), node, 2, 0)
	expect(t, !b3.current.Valid, "b3.current == null")

	// On the row right after the continuation rows: the prompt is returned and
	// the next call stops at the limit.
	b4 := top.PromptIterator(PageListDirectionRightDown, Some(PageListPin{node: node, y: 7}))
	pl6ExpectPin(t, b4.NextRightDown(), node, 2, 0)
	pl6ExpectPin(t, b4.current, node, 7, 0)
	expect(t, !b4.NextRightDown().Valid, "b4.nextRightDown() == null")
	expect(t, !b4.current.Valid, "b4.current == null")

	// A limit on a following prompt is returned.
	pl6Semantic(&node.data, "..P.P")
	c := top.PromptIterator(PageListDirectionRightDown, Some(PageListPin{node: node, y: 4}))
	pl6ExpectPin(t, c.NextRightDown(), node, 2, 0)
	pl6ExpectPin(t, c.NextRightDown(), node, 4, 0)
	expect(t, !c.current.Valid, "c.current == null")
	expect(t, !c.NextRightDown().Valid, "c.nextRightDown() == null")

	// A continuation at the limit is returned.
	pl6Semantic(&node.data, "..cc..")
	at2 := PageListPin{node: node, y: 2}
	d := at2.PromptIterator(PageListDirectionRightDown, Some(at2))
	pl6ExpectPin(t, d.NextRightDown(), node, 2, 0)
	expect(t, !d.current.Valid, "d.current == null")
	// ... and a limit on the second continuation row ends the scan.
	d2 := top.PromptIterator(PageListDirectionRightDown, Some(PageListPin{node: node, y: 3}))
	pl6ExpectPin(t, d2.NextRightDown(), node, 2, 0)
	expect(t, !d2.current.Valid, "d2.current == null")

	// A limit above the start is never reached.
	pl6Semantic(&node.data, "..P.P")
	e := PageListPin{node: node, y: 2}.PromptIterator(PageListDirectionRightDown, Some(top))
	pl6ExpectRows(t, &e, []uint{2, 4})
}

// Zig: test "term51: PromptIterator.nextRightDown: prompts at the end of the page list"
func TestPl6_PromptIterator_nextRightDown_prompts_at_the_end_of_the_page_list(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 6)
	defer s.Deinit()
	node := s.pages.first
	top := PageListPin{node: node}

	// Continuations to the last row.
	pl6Semantic(&node.data, "...Pcc")
	a := top.PromptIterator(PageListDirectionRightDown, None[PageListPin]())
	pl6ExpectPin(t, a.NextRightDown(), node, 3, 0)
	expect(t, !a.current.Valid, "a.current == null")
	expect(t, !a.NextRightDown().Valid, "a.nextRightDown() == null")

	// A prompt on the last row.
	pl6Semantic(&node.data, ".....P")
	b := top.PromptIterator(PageListDirectionRightDown, None[PageListPin]())
	pl6ExpectPin(t, b.NextRightDown(), node, 5, 0)
	expect(t, !b.current.Valid, "b.current == null")
	expect(t, !b.NextRightDown().Valid, "b.nextRightDown() == null")

	// A continuation on the last row only.
	pl6Semantic(&node.data, ".....c")
	c := top.PromptIterator(PageListDirectionRightDown, None[PageListPin]())
	pl6ExpectPin(t, c.NextRightDown(), node, 5, 0)
	expect(t, !c.current.Valid, "c.current == null")

	// Continuations at the very start (trimmed scrollback): the first one is
	// returned and the rest of the run is skipped.
	pl6Semantic(&node.data, "cc.P")
	d := top.PromptIterator(PageListDirectionRightDown, None[PageListPin]())
	pl6ExpectPin(t, d.NextRightDown(), node, 0, 0)
	pl6ExpectPin(t, d.current, node, 2, 0)
	pl6ExpectPin(t, d.NextRightDown(), node, 3, 0)
	expect(t, !d.NextRightDown().Valid, "d.nextRightDown() == null")

	// Whole page is prompt rows.
	pl6Semantic(&node.data, "PPPPPP")
	e := top.PromptIterator(PageListDirectionRightDown, None[PageListPin]())
	pl6ExpectRows(t, &e, []uint{0, 1, 2, 3, 4, 5})
	pl6Semantic(&node.data, "Pccccc")
	f := top.PromptIterator(PageListDirectionRightDown, None[PageListPin]())
	pl6ExpectRows(t, &f, []uint{0})
}

// Zig: test "term51: PromptIterator.nextRightDown: across pages"
func TestPl6_PromptIterator_nextRightDown_across_pages(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 4, 3, 3)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	n3 := n2.next
	l1 := n1.data.size.rows - 1
	l3 := n3.data.size.rows - 1

	// Prompt on the last row of page 1 with continuations that run into
	// page 2, a prompt in the middle of page 2, a prompt at the start of
	// page 3 and a lone continuation on the last row of page 3.
	n1.data.GetRowAndCell(0, uint(l1)).row.semantic_prompt = PageRowSemanticPromptPrompt
	n2.data.GetRowAndCell(0, 0).row.semantic_prompt = PageRowSemanticPromptPromptContinuation
	n2.data.GetRowAndCell(0, 1).row.semantic_prompt = PageRowSemanticPromptPromptContinuation
	n2.data.GetRowAndCell(0, 5).row.semantic_prompt = PageRowSemanticPromptPrompt
	n3.data.GetRowAndCell(0, 0).row.semantic_prompt = PageRowSemanticPromptPrompt
	n3.data.GetRowAndCell(0, uint(l3)).row.semantic_prompt = PageRowSemanticPromptPromptContinuation

	it := PageListPin{node: n1}.PromptIterator(PageListDirectionRightDown, None[PageListPin]())
	pl6ExpectPin(t, it.Next(), n1, uint(l1), 0)
	pl6ExpectPin(t, it.current, n2, 2, 0)
	pl6ExpectPin(t, it.Next(), n2, 5, 0)
	pl6ExpectPin(t, it.current, n2, 6, 0)
	pl6ExpectPin(t, it.Next(), n3, 0, 0)
	pl6ExpectPin(t, it.current, n3, 1, 0)
	pl6ExpectPin(t, it.Next(), n3, uint(l3), 0)
	expect(t, !it.current.Valid, "it.current == null")
	expect(t, !it.Next().Valid, "it.next() == null")

	// A limit inside the continuation run in the second page.
	lim := PageListPin{node: n1}.PromptIterator(PageListDirectionRightDown, Some(PageListPin{node: n2, y: 1}))
	pl6ExpectPin(t, lim.Next(), n1, uint(l1), 0)
	expect(t, !lim.current.Valid, "lim.current == null")
	expect(t, !lim.Next().Valid, "lim.next() == null")

	// A limit on the second page's prompt.
	lim2 := PageListPin{node: n1}.PromptIterator(PageListDirectionRightDown, Some(PageListPin{node: n2, y: 5}))
	pl6ExpectPin(t, lim2.Next(), n1, uint(l1), 0)
	pl6ExpectPin(t, lim2.Next(), n2, 5, 0)
	expect(t, !lim2.Next().Valid, "lim2.next() == null")

	// Starting in the middle of the list.
	mid := PageListPin{node: n2, y: 3}.PromptIterator(PageListDirectionRightDown, None[PageListPin]())
	pl6ExpectPin(t, mid.Next(), n2, 5, 0)
	pl6ExpectPin(t, mid.Next(), n3, 0, 0)
}

// Zig: test "term51: PromptIterator.nextLeftUp: no prompts and exhausted iterators"
func TestPl6_PromptIterator_nextLeftUp_no_prompts_and_exhausted_iterators(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 10)
	defer s.Deinit()
	node := s.pages.first

	bottom := PageListPin{node: node, y: 9}
	it := bottom.PromptIterator(PageListDirectionLeftUp, None[PageListPin]())
	expect(t, !it.NextLeftUp().Valid, "it.nextLeftUp() == null")
	expect(t, !it.current.Valid, "it.current == null")
	expect(t, !it.NextLeftUp().Valid, "it.nextLeftUp() == null")

	// A limit that is never hit.
	lim := bottom.PromptIterator(PageListDirectionLeftUp, Some(PageListPin{node: node, y: 5}))
	expect(t, !lim.NextLeftUp().Valid, "lim.nextLeftUp() == null")
	expect(t, !lim.current.Valid, "lim.current == null")

	// Started on the first row.
	top := PageListPin{node: node}
	first := top.PromptIterator(PageListDirectionLeftUp, None[PageListPin]())
	expect(t, !first.NextLeftUp().Valid, "first.nextLeftUp() == null")
	expect(t, !first.current.Valid, "first.current == null")
}

// Zig: test "term51: PromptIterator.nextLeftUp: prompts and continuations in order"
func TestPl6_PromptIterator_nextLeftUp_prompts_and_continuations_in_order(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 20)
	defer s.Deinit()
	node := s.pages.first
	pl6Semantic(&node.data, "..Pcc.PPc.ccP")

	bottom := PageListPin{node: node, y: 19}
	it := bottom.PromptIterator(PageListDirectionLeftUp, None[PageListPin]())

	// Prompt on 12: resume right above it.
	pl6ExpectPin(t, it.NextLeftUp(), node, 12, 0)
	pl6ExpectPin(t, it.current, node, 11, 0)
	// Continuations 11 and 10 with a non prompt row above: the top-most
	// continuation is returned and the scan resumes on the non prompt row.
	pl6ExpectPin(t, it.NextLeftUp(), node, 10, 0)
	pl6ExpectPin(t, it.current, node, 9, 0)
	// Continuation 8 with its prompt at 7: the prompt is returned, resume above it.
	pl6ExpectPin(t, it.NextLeftUp(), node, 7, 0)
	pl6ExpectPin(t, it.current, node, 6, 0)
	// The prompt at 6.
	pl6ExpectPin(t, it.NextLeftUp(), node, 6, 0)
	pl6ExpectPin(t, it.current, node, 5, 0)
	// Continuations 4 and 3 with the prompt at 2.
	pl6ExpectPin(t, it.NextLeftUp(), node, 2, 0)
	pl6ExpectPin(t, it.current, node, 1, 0)
	// Nothing above.
	expect(t, !it.NextLeftUp().Valid, "it.nextLeftUp() == null")
	expect(t, !it.current.Valid, "it.current == null")
	expect(t, !it.NextLeftUp().Valid, "it.nextLeftUp() == null")
}

// Zig: test "term51: PromptIterator.nextLeftUp: returned pins are on column zero"
func TestPl6_PromptIterator_nextLeftUp_returned_pins_are_on_column_zero(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 20)
	defer s.Deinit()
	node := s.pages.first
	pl6Semantic(&node.data, ".Pcc.cc.P")

	// The start x is carried in the resume pin but not in returned pins:
	// for a prompt row, a continuation run ended by a non prompt row, and a
	// continuation run ended by a prompt.
	start := PageListPin{node: node, y: 12, x: 3}
	it := start.PromptIterator(PageListDirectionLeftUp, None[PageListPin]())
	pl6ExpectPin(t, it.NextLeftUp(), node, 8, 0)
	pl6ExpectPin(t, it.current, node, 7, 3)
	pl6ExpectPin(t, it.NextLeftUp(), node, 5, 0)
	pl6ExpectPin(t, it.current, node, 4, 3)
	pl6ExpectPin(t, it.NextLeftUp(), node, 1, 0)
	pl6ExpectPin(t, it.current, node, 0, 3)
	expect(t, !it.NextLeftUp().Valid, "it.nextLeftUp() == null")

	// Starting on a prompt row at x > 0.
	mid := PageListPin{node: node, y: 1, x: 2}
	it2 := mid.PromptIterator(PageListDirectionLeftUp, None[PageListPin]())
	pl6ExpectPin(t, it2.NextLeftUp(), node, 1, 0)
	// Starting on a continuation row at x > 0 whose prompt is above.
	cont := PageListPin{node: node, y: 3, x: 2}
	it3 := cont.PromptIterator(PageListDirectionLeftUp, None[PageListPin]())
	pl6ExpectPin(t, it3.NextLeftUp(), node, 1, 0)
}

// Zig: test "term51: PromptIterator.nextLeftUp: limit on a non prompt row"
func TestPl6_PromptIterator_nextLeftUp_limit_on_a_non_prompt_row(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 20)
	defer s.Deinit()
	node := s.pages.first
	pl6Semantic(&node.data, "..P...")

	// The limit stops the scan before the prompt above it.
	bottom := PageListPin{node: node, y: 5}
	it := bottom.PromptIterator(PageListDirectionLeftUp, Some(PageListPin{node: node, y: 4}))
	expect(t, !it.NextLeftUp().Valid, "it.nextLeftUp() == null")
	expect(t, !it.current.Valid, "it.current == null")

	// A limit equal to the start on a non prompt row.
	it2 := bottom.PromptIterator(PageListDirectionLeftUp, Some(bottom))
	expect(t, !it2.NextLeftUp().Valid, "it2.nextLeftUp() == null")
	expect(t, !it2.current.Valid, "it2.current == null")

	// A limit before the prompt (above it): the prompt is found first.
	it3 := bottom.PromptIterator(PageListDirectionLeftUp, Some(PageListPin{node: node, y: 1}))
	pl6ExpectPin(t, it3.NextLeftUp(), node, 2, 0)
	pl6ExpectPin(t, it3.current, node, 1, 0)
	// ... and then the limit row stops the scan.
	expect(t, !it3.NextLeftUp().Valid, "it3.nextLeftUp() == null")
	expect(t, !it3.current.Valid, "it3.current == null")
}

// Zig: test "term51: PromptIterator.nextLeftUp: limit on a prompt row"
func TestPl6_PromptIterator_nextLeftUp_limit_on_a_prompt_row(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 20)
	defer s.Deinit()
	node := s.pages.first

	// A prompt at the limit is returned and finishes the iterator.
	pl6Semantic(&node.data, "..P..")
	bottom := PageListPin{node: node, y: 4}
	a := bottom.PromptIterator(PageListDirectionLeftUp, Some(PageListPin{node: node, y: 2}))
	pl6ExpectPin(t, a.NextLeftUp(), node, 2, 0)
	expect(t, !a.current.Valid, "a.current == null")
	expect(t, !a.NextLeftUp().Valid, "a.nextLeftUp() == null")

	// Without a limit the resume pin is the row above the prompt.
	b := bottom.PromptIterator(PageListDirectionLeftUp, None[PageListPin]())
	pl6ExpectPin(t, b.NextLeftUp(), node, 2, 0)
	pl6ExpectPin(t, b.current, node, 1, 0)
	expect(t, !b.NextLeftUp().Valid, "b.nextLeftUp() == null")

	// A prompt on the first row has nothing above it to resume on.
	pl6Semantic(&node.data, "P...")
	row3 := PageListPin{node: node, y: 3}
	c := row3.PromptIterator(PageListDirectionLeftUp, None[PageListPin]())
	pl6ExpectPin(t, c.NextLeftUp(), node, 0, 0)
	expect(t, !c.current.Valid, "c.current == null")
	expect(t, !c.NextLeftUp().Valid, "c.nextLeftUp() == null")

	// A limit below the start is never reached.
	pl6Semantic(&node.data, "..P.P")
	at4 := PageListPin{node: node, y: 4}
	d := at4.PromptIterator(PageListDirectionLeftUp, Some(PageListPin{node: node, y: 8}))
	pl6ExpectRows(t, &d, []uint{4, 2})
}

// Zig: test "term51: PromptIterator.nextLeftUp: limit on a continuation row"
func TestPl6_PromptIterator_nextLeftUp_limit_on_a_continuation_row(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 20)
	defer s.Deinit()
	node := s.pages.first
	pl6Semantic(&node.data, "..cc..")

	// A continuation at the limit is returned and finishes the iterator.
	at3 := PageListPin{node: node, y: 3}
	a := at3.PromptIterator(PageListDirectionLeftUp, Some(at3))
	pl6ExpectPin(t, a.NextLeftUp(), node, 3, 0)
	expect(t, !a.current.Valid, "a.current == null")
	expect(t, !a.NextLeftUp().Valid, "a.nextLeftUp() == null")
}

// Zig: test "term51: PromptIterator.nextLeftUp: continuation runs"
func TestPl6_PromptIterator_nextLeftUp_continuation_runs(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 20)
	defer s.Deinit()
	node := s.pages.first

	// A run of continuations ended by a non prompt row: the top-most
	// continuation is returned and the scan resumes on the non prompt row.
	pl6Semantic(&node.data, ".cc")
	a := PageListPin{node: node, y: 2}.PromptIterator(PageListDirectionLeftUp, None[PageListPin]())
	pl6ExpectPin(t, a.NextLeftUp(), node, 1, 0)
	pl6ExpectPin(t, a.current, node, 0, 0)
	expect(t, !a.NextLeftUp().Valid, "a.nextLeftUp() == null")

	// A run ended by a non prompt row with a prompt further above.
	pl6Semantic(&node.data, "P.cc")
	b := PageListPin{node: node, y: 3}.PromptIterator(PageListDirectionLeftUp, None[PageListPin]())
	pl6ExpectPin(t, b.NextLeftUp(), node, 2, 0)
	pl6ExpectPin(t, b.current, node, 1, 0)
	pl6ExpectPin(t, b.NextLeftUp(), node, 0, 0)
	expect(t, !b.current.Valid, "b.current == null")
	expect(t, !b.NextLeftUp().Valid, "b.nextLeftUp() == null")

	// A single continuation with a non prompt row above.
	pl6Semantic(&node.data, ".c")
	c := PageListPin{node: node, y: 1}.PromptIterator(PageListDirectionLeftUp, None[PageListPin]())
	pl6ExpectPin(t, c.NextLeftUp(), node, 1, 0)
	pl6ExpectPin(t, c.current, node, 0, 0)

	// A run ended by its prompt on the first row.
	pl6Semantic(&node.data, "Pcc")
	d := PageListPin{node: node, y: 2}.PromptIterator(PageListDirectionLeftUp, None[PageListPin]())
	pl6ExpectPin(t, d.NextLeftUp(), node, 0, 0)
	expect(t, !d.current.Valid, "d.current == null")
	expect(t, !d.NextLeftUp().Valid, "d.nextLeftUp() == null")

	// A single continuation right below its prompt.
	pl6Semantic(&node.data, ".Pc")
	e := PageListPin{node: node, y: 2}.PromptIterator(PageListDirectionLeftUp, None[PageListPin]())
	pl6ExpectPin(t, e.NextLeftUp(), node, 1, 0)
	pl6ExpectPin(t, e.current, node, 0, 0)
}

// Zig: test "term51: PromptIterator.nextLeftUp: continuations running to the first row return the starting row"
func TestPl6_PromptIterator_nextLeftUp_continuations_running_to_the_first_row_return_the_starting_row(t *testing.T) {
	// Upstream quirk pin: when the scan reaches the first row of the list
	// without finding a prompt or a non prompt row, the row the scan STARTED
	// on (the bottom-most continuation) is returned, not the top-most.
	// The right_down direction returns the first (top-most) one.
	testStdCapacity(t)
	s := pl6Init(t, 4, 20)
	defer s.Deinit()
	node := s.pages.first
	pl6Semantic(&node.data, "ccc")

	up := PageListPin{node: node, y: 2}.PromptIterator(PageListDirectionLeftUp, None[PageListPin]())
	pl6ExpectPin(t, up.NextLeftUp(), node, 2, 0)
	expect(t, !up.current.Valid, "up.current == null")
	expect(t, !up.NextLeftUp().Valid, "up.nextLeftUp() == null")

	down := PageListPin{node: node}.PromptIterator(PageListDirectionRightDown, None[PageListPin]())
	pl6ExpectPin(t, down.NextRightDown(), node, 0, 0)

	// The same for a single continuation on the first row and from x > 0.
	pl6Semantic(&node.data, "c")
	one := PageListPin{node: node, y: 0, x: 3}.PromptIterator(PageListDirectionLeftUp, None[PageListPin]())
	pl6ExpectPin(t, one.NextLeftUp(), node, 0, 0)
	expect(t, !one.current.Valid, "one.current == null")
}

// Zig: test "term51: PromptIterator.nextLeftUp: limit reached while scanning continuations"
func TestPl6_PromptIterator_nextLeftUp_limit_reached_while_scanning_continuations(t *testing.T) {
	// Upstream quirk pin: when the limit is met while scanning up through a
	// continuation run, the scan stops and the row it started on is returned
	// (the bottom-most continuation), even when the limit row is the prompt
	// itself. A prompt that is the very first row examined at the limit
	// IS returned (see the prompt at limit test).
	testStdCapacity(t)
	s := pl6Init(t, 4, 20)
	defer s.Deinit()
	node := s.pages.first
	pl6Semantic(&node.data, "..Pcc")
	start := PageListPin{node: node, y: 4}

	// Limit on the first continuation row.
	a := start.PromptIterator(PageListDirectionLeftUp, Some(PageListPin{node: node, y: 3}))
	pl6ExpectPin(t, a.NextLeftUp(), node, 4, 0)
	expect(t, !a.current.Valid, "a.current == null")
	expect(t, !a.NextLeftUp().Valid, "a.nextLeftUp() == null")

	// Limit on the prompt row of the run.
	b := start.PromptIterator(PageListDirectionLeftUp, Some(PageListPin{node: node, y: 2}))
	pl6ExpectPin(t, b.NextLeftUp(), node, 4, 0)
	expect(t, !b.current.Valid, "b.current == null")
	expect(t, !b.NextLeftUp().Valid, "b.nextLeftUp() == null")

	// Limit on the starting row itself.
	c := start.PromptIterator(PageListDirectionLeftUp, Some(start))
	pl6ExpectPin(t, c.NextLeftUp(), node, 4, 0)
	expect(t, !c.current.Valid, "c.current == null")

	// A limit above the prompt does not interfere.
	d := start.PromptIterator(PageListDirectionLeftUp, Some(PageListPin{node: node, y: 1}))
	pl6ExpectPin(t, d.NextLeftUp(), node, 2, 0)
	pl6ExpectPin(t, d.current, node, 1, 0)
}

// Zig: test "term51: PromptIterator.nextLeftUp: across pages"
func TestPl6_PromptIterator_nextLeftUp_across_pages(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 4, 3, 3)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	n3 := n2.next
	l1 := n1.data.size.rows - 1
	l3 := n3.data.size.rows - 1

	n1.data.GetRowAndCell(0, uint(l1)).row.semantic_prompt = PageRowSemanticPromptPrompt
	n2.data.GetRowAndCell(0, 0).row.semantic_prompt = PageRowSemanticPromptPromptContinuation
	n2.data.GetRowAndCell(0, 1).row.semantic_prompt = PageRowSemanticPromptPromptContinuation
	n2.data.GetRowAndCell(0, 5).row.semantic_prompt = PageRowSemanticPromptPrompt
	n3.data.GetRowAndCell(0, 0).row.semantic_prompt = PageRowSemanticPromptPrompt
	n3.data.GetRowAndCell(0, uint(l3)).row.semantic_prompt = PageRowSemanticPromptPromptContinuation

	it := PageListPin{node: n3, y: l3}.PromptIterator(PageListDirectionLeftUp, None[PageListPin]())
	// Continuation on the last row with a non prompt row above.
	pl6ExpectPin(t, it.Next(), n3, uint(l3), 0)
	pl6ExpectPin(t, it.current, n3, uint(l3)-1, 0)
	// The prompt at the start of page 3; resume on the last row of page 2.
	pl6ExpectPin(t, it.Next(), n3, 0, 0)
	pl6ExpectPin(t, it.current, n2, uint(n2.data.size.rows)-1, 0)
	// The prompt in the middle of page 2.
	pl6ExpectPin(t, it.Next(), n2, 5, 0)
	pl6ExpectPin(t, it.current, n2, 4, 0)
	// The continuation run at the start of page 2 belongs to the prompt at
	// the end of page 1.
	pl6ExpectPin(t, it.Next(), n1, uint(l1), 0)
	pl6ExpectPin(t, it.current, n1, uint(l1)-1, 0)
	expect(t, !it.Next().Valid, "it.next() == null")
	expect(t, !it.current.Valid, "it.current == null")

	// A limit on the second page's prompt.
	lim := PageListPin{node: n3, y: l3}.PromptIterator(PageListDirectionLeftUp, Some(PageListPin{node: n2, y: 5}))
	pl6ExpectPin(t, lim.Next(), n3, uint(l3), 0)
	pl6ExpectPin(t, lim.Next(), n3, 0, 0)
	pl6ExpectPin(t, lim.Next(), n2, 5, 0)
	expect(t, !lim.current.Valid, "lim.current == null")
	expect(t, !lim.Next().Valid, "lim.next() == null")

	// Starting on the page boundary.
	mid := PageListPin{node: n2}.PromptIterator(PageListDirectionLeftUp, None[PageListPin]())
	pl6ExpectPin(t, mid.Next(), n1, uint(l1), 0)
	expect(t, !mid.Next().Valid, "mid.next() == null")
}

// Zig: test "term51: Pin.promptIterator: stores the pin limit and direction"
func TestPl6_Pin_promptIterator_stores_the_pin_limit_and_direction(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 10)
	defer s.Deinit()
	node := s.pages.first

	a := PageListPin{node: node, y: 2, x: 3, garbage: true}
	b := PageListPin{node: node, y: 7, x: 1}

	it := a.PromptIterator(PageListDirectionRightDown, Some(b))
	expect(t, it.direction == PageListDirectionRightDown, "it.direction == .right_down")
	expect(t, zigUnwrap(t, it.current).Eql(a), "it.current.?.eql(a)")
	expect(t, zigUnwrap(t, it.current).garbage, "it.current.?.garbage")
	expect(t, zigUnwrap(t, it.limit).Eql(b), "it.limit.?.eql(b)")

	it2 := b.PromptIterator(PageListDirectionLeftUp, Some(a))
	expect(t, it2.direction == PageListDirectionLeftUp, "it2.direction == .left_up")
	expect(t, zigUnwrap(t, it2.current).Eql(b), "it2.current.?.eql(b)")
	expect(t, zigUnwrap(t, it2.limit).Eql(a), "it2.limit.?.eql(a)")

	it3 := a.PromptIterator(PageListDirectionLeftUp, None[PageListPin]())
	expect(t, !it3.limit.Valid, "it3.limit == null")
	expect(t, zigUnwrap(t, it3.current).Eql(a), "it3.current.?.eql(a)")
}

// Zig: test "term51: promptIterator: bounds and directions"
func TestPl6_promptIterator_bounds_and_directions(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 20)
	defer s.Deinit()
	node := s.pages.first
	pl6Semantic(&node.data, "..Pcc.PPc.ccP")

	// The pins of the iterator for explicit bounds.
	{
		it := s.PromptIterator(PageListDirectionRightDown, pl6Pt(2, 3), Some(pl6Pt(1, 9)))
		expect(t, it.direction == PageListDirectionRightDown, "it.direction == .right_down")
		pl6ExpectPin(t, it.current, node, 3, 2)
		pl6ExpectPin(t, it.limit, node, 9, 1)
	}
	{
		it := s.PromptIterator(PageListDirectionLeftUp, pl6Pt(2, 3), Some(pl6Pt(1, 9)))
		expect(t, it.direction == PageListDirectionLeftUp, "it.direction == .left_up")
		pl6ExpectPin(t, it.current, node, 9, 1)
		pl6ExpectPin(t, it.limit, node, 3, 2)
	}

	// Results for explicit bounds, limit rows inclusive.
	down := s.PromptIterator(PageListDirectionRightDown, pl6Pt(0, 0), Some(pl6Pt(0, 7)))
	pl6ExpectRows(t, &down, []uint{2, 6, 7})
	up := s.PromptIterator(PageListDirectionLeftUp, pl6Pt(0, 0), Some(pl6Pt(0, 7)))
	pl6ExpectRows(t, &up, []uint{7, 6, 2})
	// Starting in the middle of the first prompt, from its continuation.
	down2 := s.PromptIterator(PageListDirectionRightDown, pl6Pt(0, 3), Some(pl6Pt(0, 12)))
	pl6ExpectRows(t, &down2, []uint{3, 6, 7, 10, 12})
	// (Walking up, the continuation run 3-4 ends on the limit row 3 so the
	// row the scan started on, 4, is returned: see the limit reached while
	// scanning continuations test.)
	up2 := s.PromptIterator(PageListDirectionLeftUp, pl6Pt(0, 3), Some(pl6Pt(0, 12)))
	pl6ExpectRows(t, &up2, []uint{12, 10, 7, 6, 4})

	// A range with no prompts.
	none := s.PromptIterator(PageListDirectionRightDown, pl6Pt(0, 13), Some(pl6Pt(0, 19)))
	pl6ExpectRows(t, &none, []uint{})
	none_up := s.PromptIterator(PageListDirectionLeftUp, pl6Pt(0, 13), Some(pl6Pt(0, 19)))
	pl6ExpectRows(t, &none_up, []uint{})

	// A single row range with a prompt and without.
	one := s.PromptIterator(PageListDirectionRightDown, pl6Pt(0, 6), Some(pl6Pt(0, 6)))
	pl6ExpectRows(t, &one, []uint{6})
	one_up := s.PromptIterator(PageListDirectionLeftUp, pl6Pt(0, 6), Some(pl6Pt(0, 6)))
	pl6ExpectRows(t, &one_up, []uint{6})
	blank := s.PromptIterator(PageListDirectionRightDown, pl6Pt(0, 5), Some(pl6Pt(0, 5)))
	pl6ExpectRows(t, &blank, []uint{})
}

// Zig: test "term51: promptIterator: no bottom bound uses the area of the tag"
func TestPl6_promptIterator_no_bottom_bound_uses_the_area_of_the_tag(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 10)
	defer s.Deinit()
	pl6GrowRows(t, &s, 6)
	// 16 rows: 6 history, 10 active.
	node := s.pages.first
	pl6Semantic(&node.data, "..P..P...P.....P")
	// prompts on 2, 5, 9, 15; active starts at row 6.

	screen_d := s.PromptIterator(PageListDirectionRightDown, Point{Tag: PointTagScreen}, None[Point]())
	pl6ExpectRows(t, &screen_d, []uint{2, 5, 9, 15})
	screen_u := s.PromptIterator(PageListDirectionLeftUp, Point{Tag: PointTagScreen}, None[Point]())
	pl6ExpectRows(t, &screen_u, []uint{15, 9, 5, 2})

	active_d := s.PromptIterator(PageListDirectionRightDown, Point{Tag: PointTagActive}, None[Point]())
	pl6ExpectRows(t, &active_d, []uint{9, 15})

	history_u := s.PromptIterator(PageListDirectionLeftUp, Point{Tag: PointTagHistory}, None[Point]())
	pl6ExpectRows(t, &history_u, []uint{5, 2})

	view_d := s.PromptIterator(PageListDirectionRightDown, Point{Tag: PointTagViewport}, None[Point]())
	pl6ExpectRows(t, &view_d, []uint{9, 15})

	// Scrolled to the top: the viewport is the first 10 rows.
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	top_u := s.PromptIterator(PageListDirectionLeftUp, Point{Tag: PointTagViewport}, None[Point]())
	pl6ExpectRows(t, &top_u, []uint{9, 5, 2})

	// The starting point at a later x keeps the x in the resume pin.
	xs := s.PromptIterator(PageListDirectionRightDown, Point{Tag: PointTagScreen, Screen: PointCoordinate{x: 2, y: 3}}, None[Point]())
	pl6ExpectPin(t, xs.current, node, 3, 2)
	pl6ExpectRows(t, &xs, []uint{5, 9, 15})
}

// Zig: test "term51: promptIterator: the limit only matches rows scanned with the same x"
func TestPl6_promptIterator_the_limit_only_matches_rows_scanned_with_the_same_x(t *testing.T) {
	// Upstream quirk pin: the limit is compared with Pin.eql, which includes
	// x, while the pins that are scanned keep the x of the starting pin. A
	// limit with a different x than the start is therefore never hit and the
	// scan runs to the end of the page list. With no bottom bound the limit
	// is the bottom right (x = cols - 1) or the top left (x = 0) of the area,
	// so it only has an effect when the start pin has that x.
	testStdCapacity(t)
	s := pl6Init(t, 4, 10)
	defer s.Deinit()
	pl6GrowRows(t, &s, 6)
	// 16 rows: 6 history, 10 active.
	node := s.pages.first
	pl6Semantic(&node.data, "..P..P...P.....P")
	// prompts on 2, 5, 9, 15; active starts at row 6.

	// left_up from the bottom right: the limit (top left of the active
	// area, x = 0) is never seen, so history prompts are included too.
	active_u := s.PromptIterator(PageListDirectionLeftUp, Point{Tag: PointTagActive}, None[Point]())
	pl6ExpectRows(t, &active_u, []uint{15, 9, 5, 2})

	// right_down from the top left of history: the limit is the bottom right
	// of history (x = 3), never seen, so active prompts are included too.
	history_d := s.PromptIterator(PageListDirectionRightDown, Point{Tag: PointTagHistory}, None[Point]())
	pl6ExpectRows(t, &history_d, []uint{2, 5, 9, 15})

	// Same for a viewport scrolled to the top.
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	top_d := s.PromptIterator(PageListDirectionRightDown, Point{Tag: PointTagViewport}, None[Point]())
	pl6ExpectRows(t, &top_d, []uint{2, 5, 9, 15})

	// Explicit bounds: limit x differs from the start x, so it is ignored.
	mism := s.PromptIterator(PageListDirectionRightDown, pl6Pt(0, 0), Some(pl6Pt(1, 5)))
	pl6ExpectRows(t, &mism, []uint{2, 5, 9, 15})
	mism_up := s.PromptIterator(PageListDirectionLeftUp, pl6Pt(1, 6), Some(pl6Pt(0, 15)))
	pl6ExpectRows(t, &mism_up, []uint{15, 9, 5, 2})

	// Same x on both: the limit works.
	same := s.PromptIterator(PageListDirectionRightDown, pl6Pt(1, 0), Some(pl6Pt(1, 5)))
	pl6ExpectRows(t, &same, []uint{2, 5})
	same_up := s.PromptIterator(PageListDirectionLeftUp, pl6Pt(3, 6), Some(pl6Pt(3, 15)))
	pl6ExpectRows(t, &same_up, []uint{15, 9})
}

// Zig: test "term51: promptIterator: history without history is the empty iterator"
func TestPl6_promptIterator_history_without_history_is_the_empty_iterator(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 4, 10)
	defer s.Deinit()
	pl6Semantic(&s.pages.first.data, "P.P.P")

	// The empty iterator is returned for either direction, and it is a
	// left_up iterator.
	for _, dir := range pl6Dirs {
		it := s.PromptIterator(dir, Point{Tag: PointTagHistory}, None[Point]())
		expect(t, !it.current.Valid, "it.current == null")
		expect(t, !it.limit.Valid, "it.limit == null")
		expect(t, it.direction == PageListDirectionLeftUp, "it.direction == .left_up")
		expect(t, !it.Next().Valid, "it.next() == null")
	}

	// With an explicit bottom bound the area is fine.
	it := s.PromptIterator(PageListDirectionRightDown, Point{Tag: PointTagHistory}, Some(Point{Tag: PointTagHistory, History: PointCoordinate{y: 3}}))
	pl6ExpectRows(t, &it, []uint{0, 2})
}

// ---- highlightSemanticContent ----

// pl6ExpectHl is term51NExpectHl.
func pl6ExpectHl(
	t *testing.T,
	s *PageList,
	hl Optional[HighlightUntracked],
	sx SizeCellCountInt,
	sy uint32,
	ex SizeCellCountInt,
	ey uint32,
) {
	t.Helper()
	expect(t, hl.Valid, "hl != null")
	expectEqual(t, pl6Pt(sx, sy), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.Value.start)))
	expectEqual(t, pl6Pt(ex, ey), zigUnwrap(t, s.PointFromPin(PointTagScreen, hl.Value.end)))
}

// pl6At is `s.pin(term51NPt(x, y)).?`.
func pl6At(t *testing.T, s *PageList, x SizeCellCountInt, y uint32) PageListPin {
	t.Helper()
	return zigUnwrap(t, s.Pin(pl6Pt(x, y)))
}

// Zig: test "term51: highlightSemanticContent: prompt scans from the pin x and starts at column zero"
func TestPl6_highlightSemanticContent_prompt_scans_from_the_pin_x_and_starts_at_column_zero(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 20)
	defer s.Deinit()
	page := &s.pages.first.data
	pl6Semantic(page, ".....P....P")
	pl6Cells(page, 5, "pppiiioooo")
	pl6Cells(page, 10, "pppp")

	// Starting at the first cell: prompt and input, up to the output.
	hl := s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentPrompt)
	pl6ExpectHl(t, &s, hl, 0, 5, 5, 5)
	// Starting inside the prompt text: the start is still column zero.
	hl = s.HighlightSemanticContent(pl6At(t, &s, 2, 5), PageCellSemanticContentPrompt)
	pl6ExpectHl(t, &s, hl, 0, 5, 5, 5)
	// Starting inside the input: cells before the pin are not examined.
	hl = s.HighlightSemanticContent(pl6At(t, &s, 4, 5), PageCellSemanticContentPrompt)
	pl6ExpectHl(t, &s, hl, 0, 5, 5, 5)
	// Starting on the last input cell
	hl = s.HighlightSemanticContent(pl6At(t, &s, 5, 5), PageCellSemanticContentPrompt)
	pl6ExpectHl(t, &s, hl, 0, 5, 5, 5)
	// Starting inside the output: the first cell examined ends the scan, the
	// end is the pin itself.
	hl = s.HighlightSemanticContent(pl6At(t, &s, 8, 5), PageCellSemanticContentPrompt)
	pl6ExpectHl(t, &s, hl, 0, 5, 8, 5)
	hl = s.HighlightSemanticContent(pl6At(t, &s, 6, 5), PageCellSemanticContentPrompt)
	pl6ExpectHl(t, &s, hl, 0, 5, 6, 5)
}

// Zig: test "term51: highlightSemanticContent: prompt stops at default output cells"
func TestPl6_highlightSemanticContent_prompt_stops_at_default_output_cells(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 20)
	defer s.Deinit()
	page := &s.pages.first.data
	pl6Semantic(page, ".....P")

	// A whole row of prompt cells, the next row is default (output) cells.
	pl6Cells(page, 5, "pppppppppp")
	hl := s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentPrompt)
	pl6ExpectHl(t, &s, hl, 0, 5, 9, 5)

	// Prompt cells without text count as prompt.
	pl6Cells(page, 5, "qqqqqnnnn.")
	hl = s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentPrompt)
	pl6ExpectHl(t, &s, hl, 0, 5, 8, 5)

	// The very first cell is output: start is the pin and so is the end.
	pl6Cells(page, 5, "..........")
	hl = s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentPrompt)
	pl6ExpectHl(t, &s, hl, 0, 5, 0, 5)
}

// Zig: test "term51: highlightSemanticContent: prompt is bounded by the next prompt row"
func TestPl6_highlightSemanticContent_prompt_is_bounded_by_the_next_prompt_row(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 20)
	defer s.Deinit()
	page := &s.pages.first.data
	pl6Semantic(page, ".....P.P")
	pl6Cells(page, 5, "ppiiiiiiii")
	pl6Cells(page, 6, "iiiiiiiiii")
	pl6Cells(page, 7, "pppp")

	// Everything on rows 5 and 6 is prompt or input, the scan ends with row 6.
	hl := s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentPrompt)
	pl6ExpectHl(t, &s, hl, 0, 5, 9, 6)

	// Next prompt right on the following row.
	pl6Semantic(page, ".....PP")
	hl = s.HighlightSemanticContent(pl6At(t, &s, 3, 5), PageCellSemanticContentPrompt)
	pl6ExpectHl(t, &s, hl, 0, 5, 9, 5)
}

// Zig: test "term51: highlightSemanticContent: prompt across pages"
func TestPl6_highlightSemanticContent_prompt_across_pages(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	l1 := n1.data.size.rows - 1

	// Prompt on the last row of page 1, input rows at the start of page 2.
	n1.data.GetRowAndCell(0, uint(l1)).row.semantic_prompt = PageRowSemanticPromptPrompt
	pl6Cells(&n1.data, uint(l1), "ppppiiiiii")
	pl6Cells(&n2.data, 0, "iiiiiiiiii")
	pl6Cells(&n2.data, 1, "iiioooooo")
	at := PageListPin{node: n1, y: l1, x: 1}

	// No further prompt: the scan crosses the page, and stops at the output.
	hl := s.HighlightSemanticContent(at, PageCellSemanticContentPrompt)
	expect(t, hl.Valid, "hl != null")
	pl6ExpectPinV(t, hl.Value.start, n1, uint(l1), 0)
	pl6ExpectPinV(t, hl.Value.end, n2, 1, 2)

	// A prompt at the start of the next page bounds the result to page 1.
	n2.data.GetRowAndCell(0, 0).row.semantic_prompt = PageRowSemanticPromptPrompt
	hl = s.HighlightSemanticContent(at, PageCellSemanticContentPrompt)
	expect(t, hl.Valid, "hl != null")
	pl6ExpectPinV(t, hl.Value.start, n1, uint(l1), 0)
	pl6ExpectPinV(t, hl.Value.end, n1, uint(l1), 9)
}

// Zig: test "term51: highlightSemanticContent: input between prompt and output"
func TestPl6_highlightSemanticContent_input_between_prompt_and_output(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 20)
	defer s.Deinit()
	page := &s.pages.first.data
	pl6Semantic(page, ".....P....P")
	pl6Cells(page, 10, "pppp")

	// Input cells 2 and 3.
	pl6Cells(page, 5, "ppiioooooo")
	hl := s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentInput)
	pl6ExpectHl(t, &s, hl, 2, 5, 3, 5)
	hl = s.HighlightSemanticContent(pl6At(t, &s, 7, 5), PageCellSemanticContentPrompt)
	pl6ExpectHl(t, &s, hl, 0, 5, 7, 5)

	// A single input cell: start and end are the same.
	pl6Cells(page, 5, "ppioooooo.")
	hl = s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentInput)
	pl6ExpectHl(t, &s, hl, 2, 5, 2, 5)

	// Input cells without text count.
	pl6Cells(page, 5, "ppnnoooooo")
	hl = s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentInput)
	pl6ExpectHl(t, &s, hl, 2, 5, 3, 5)

	// Input starting at the very first cell.
	pl6Cells(page, 5, "iiioooooo.")
	hl = s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentInput)
	pl6ExpectHl(t, &s, hl, 0, 5, 2, 5)
}

// Zig: test "term51: highlightSemanticContent: input skips nested prompt cells"
func TestPl6_highlightSemanticContent_input_skips_nested_prompt_cells(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 20)
	defer s.Deinit()
	page := &s.pages.first.data
	pl6Semantic(page, ".....P....P")
	pl6Cells(page, 10, "pppp")

	// Prompt cells before the first input and between inputs are skipped, the
	// end is the last input before the output.
	pl6Cells(page, 5, "piippioooo")
	hl := s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentInput)
	pl6ExpectHl(t, &s, hl, 1, 5, 5, 5)

	// Prompt cells after the last input do not extend it.
	pl6Cells(page, 5, "ppiippoooo")
	hl = s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentInput)
	pl6ExpectHl(t, &s, hl, 2, 5, 3, 5)
}

// Zig: test "term51: highlightSemanticContent: input without input is null"
func TestPl6_highlightSemanticContent_input_without_input_is_null(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 20)
	defer s.Deinit()
	page := &s.pages.first.data
	pl6Semantic(page, ".....P....P")
	pl6Cells(page, 10, "pppp")

	// Output text right after the prompt.
	pl6Cells(page, 5, "ppoooooooo")
	expect(t, !s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentInput).Valid, "s.highlightSemanticContent(s.pin(term51NPt(0, 5)).?, .input) == null")

	// Default (empty output) cells right after the prompt.
	pl6Cells(page, 5, "pp")
	expect(t, !s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentInput).Valid, "s.highlightSemanticContent(s.pin(term51NPt(0, 5)).?, .input) == null")

	// Output as the very first cell.
	pl6Cells(page, 5, "oppiiiiiii")
	expect(t, !s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentInput).Valid, "s.highlightSemanticContent(s.pin(term51NPt(0, 5)).?, .input) == null")

	// Prompt cells all the way to the next prompt row: the scan runs out.
	pl6Semantic(page, ".....P.P")
	pl6Cells(page, 5, "pppppppppp")
	pl6Cells(page, 6, "pppppppppp")
	expect(t, !s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentInput).Valid, "s.highlightSemanticContent(s.pin(term51NPt(0, 5)).?, .input) == null")
}

// Zig: test "term51: highlightSemanticContent: input starts scanning at the pin x"
func TestPl6_highlightSemanticContent_input_starts_scanning_at_the_pin_x(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 20)
	defer s.Deinit()
	page := &s.pages.first.data
	pl6Semantic(page, ".....P....P")
	pl6Cells(page, 10, "pppp")
	pl6Cells(page, 5, "ppiioooooo")

	hl := s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentInput)
	pl6ExpectHl(t, &s, hl, 2, 5, 3, 5)
	hl = s.HighlightSemanticContent(pl6At(t, &s, 3, 5), PageCellSemanticContentInput)
	pl6ExpectHl(t, &s, hl, 3, 5, 3, 5)
	// Cells before the pin are not looked at, output at the pin gives null.
	expect(t, !s.HighlightSemanticContent(pl6At(t, &s, 4, 5), PageCellSemanticContentInput).Valid, "s.highlightSemanticContent(s.pin(term51NPt(4, 5)).?, .input) == null")
}

// Zig: test "term51: highlightSemanticContent: input on several rows"
func TestPl6_highlightSemanticContent_input_on_several_rows(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 20)
	defer s.Deinit()
	page := &s.pages.first.data

	// Input continues on the next row and ends in output on the row after.
	pl6Semantic(page, ".....Pc..P")
	pl6Cells(page, 5, "ppiiiiiiii")
	pl6Cells(page, 6, "ppiiiiiiii")
	pl6Cells(page, 7, "oooooooooo")
	pl6Cells(page, 9, "pppp")
	hl := s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentInput)
	pl6ExpectHl(t, &s, hl, 2, 5, 9, 6)

	// The next prompt row bounds the scan even without output cells.
	pl6Semantic(page, ".....P.P")
	pl6Cells(page, 5, "ppiiiiiiii")
	pl6Cells(page, 6, "iiiiiiiiii")
	pl6Cells(page, 7, "pppp")
	hl = s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentInput)
	pl6ExpectHl(t, &s, hl, 2, 5, 9, 6)

	// Input starting on a later row than the prompt cells.
	pl6Semantic(page, ".....Pc..P")
	pl6Cells(page, 5, "pppppppppp")
	pl6Cells(page, 6, "ppiioooooo")
	pl6Cells(page, 7, "oooooooooo")
	hl = s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentInput)
	pl6ExpectHl(t, &s, hl, 2, 6, 3, 6)
}

// Zig: test "term51: highlightSemanticContent: output skips cells without text"
func TestPl6_highlightSemanticContent_output_skips_cells_without_text(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 20)
	defer s.Deinit()
	page := &s.pages.first.data
	pl6Semantic(page, ".....P....P")
	pl6Cells(page, 10, "pppp")

	// Empty output cells before and after the text are not part of it.
	pl6Cells(page, 5, "ppii..oo..")
	hl := s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentOutput)
	pl6ExpectHl(t, &s, hl, 6, 5, 7, 5)

	// Text with an empty gap inside: the end is the last text cell.
	pl6Cells(page, 5, "ppiio..o..")
	hl = s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentOutput)
	pl6ExpectHl(t, &s, hl, 4, 5, 7, 5)

	// A single text cell.
	pl6Cells(page, 5, "ppiio.....")
	hl = s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentOutput)
	pl6ExpectHl(t, &s, hl, 4, 5, 4, 5)

	// A background-only cell is output without text, and skipped.
	pl6Cells(page, 5, "ppii")
	*page.GetRowAndCell(4, 5).cell = PageCell{
		content_tag: PageCellContentTagBgColorRgb,
		content:     PageCellContent{color_rgb: PageCellRGB{r: 1, g: 2, b: 3}},
	}
	*page.GetRowAndCell(5, 5).cell = PageCellInit('x')
	hl = s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentOutput)
	pl6ExpectHl(t, &s, hl, 5, 5, 5, 5)
}

// Zig: test "term51: highlightSemanticContent: output ends at prompt or input"
func TestPl6_highlightSemanticContent_output_ends_at_prompt_or_input(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 20)
	defer s.Deinit()
	page := &s.pages.first.data
	pl6Semantic(page, ".....P....P")
	pl6Cells(page, 10, "pppp")

	// Input after the output ends it, later output is not included.
	pl6Cells(page, 5, "pp")
	pl6Cells(page, 6, "oooooooooo")
	pl6Cells(page, 7, "iiiiiiiiii")
	pl6Cells(page, 8, "oooooooooo")
	hl := s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentOutput)
	pl6ExpectHl(t, &s, hl, 0, 6, 9, 6)

	// A prompt cell after the output ends it too: the text before it on the
	// same row is still included.
	pl6Cells(page, 7, "ooopppoooo")
	hl = s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentOutput)
	pl6ExpectHl(t, &s, hl, 0, 6, 2, 7)
	pl6Cells(page, 6, "ppoooooooo")
	hl = s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentOutput)
	pl6ExpectHl(t, &s, hl, 2, 6, 2, 7)
	// ... in the middle of the same row
	pl6Cells(page, 6, "ppooopoooo")
	hl = s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentOutput)
	pl6ExpectHl(t, &s, hl, 2, 6, 4, 6)
}

// Zig: test "term51: highlightSemanticContent: output on several rows up to the next prompt"
func TestPl6_highlightSemanticContent_output_on_several_rows_up_to_the_next_prompt(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 20)
	defer s.Deinit()
	page := &s.pages.first.data
	pl6Semantic(page, ".....P...P")
	pl6Cells(page, 9, "pppp")
	pl6Cells(page, 5, "ppiiiiiiii")
	pl6Cells(page, 6, "oooooooooo")
	pl6Cells(page, 7, "oooo......")

	// Row 8 has only empty cells; the end is the last cell with text.
	hl := s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentOutput)
	pl6ExpectHl(t, &s, hl, 0, 6, 3, 7)

	// Output text up to the end of the row before the next prompt.
	pl6Cells(page, 7, "oooooooooo")
	pl6Cells(page, 8, "oooooooooo")
	hl = s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentOutput)
	pl6ExpectHl(t, &s, hl, 0, 6, 9, 8)

	// The pin x does not matter for the rows after the first.
	hl = s.HighlightSemanticContent(pl6At(t, &s, 9, 5), PageCellSemanticContentOutput)
	pl6ExpectHl(t, &s, hl, 0, 6, 9, 8)
}

// Zig: test "term51: highlightSemanticContent: output starts scanning at the pin x"
func TestPl6_highlightSemanticContent_output_starts_scanning_at_the_pin_x(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 20)
	defer s.Deinit()
	page := &s.pages.first.data
	pl6Semantic(page, ".....P....P")
	pl6Cells(page, 10, "pppp")
	pl6Cells(page, 5, "ppoooooooo")

	hl := s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentOutput)
	pl6ExpectHl(t, &s, hl, 2, 5, 9, 5)
	hl = s.HighlightSemanticContent(pl6At(t, &s, 7, 5), PageCellSemanticContentOutput)
	pl6ExpectHl(t, &s, hl, 7, 5, 9, 5)
}

// Zig: test "term51: highlightSemanticContent: output without text is null"
func TestPl6_highlightSemanticContent_output_without_text_is_null(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 20)
	defer s.Deinit()
	page := &s.pages.first.data
	pl6Semantic(page, ".....P....P")
	pl6Cells(page, 10, "pppp")

	// Only prompt and input and then empty cells until the next prompt.
	pl6Cells(page, 5, "ppii")
	expect(t, !s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentOutput).Valid, "s.highlightSemanticContent(s.pin(term51NPt(0, 5)).?, .output) == null")

	// Only background cells in the output.
	*page.GetRowAndCell(0, 6).cell = PageCell{
		content_tag: PageCellContentTagBgColorPalette,
		content:     PageCellContent{color_palette: 2},
	}
	expect(t, !s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentOutput).Valid, "s.highlightSemanticContent(s.pin(term51NPt(0, 5)).?, .output) == null")

	// Text beyond the next prompt row is not output of this prompt.
	pl6Cells(page, 12, "oooo")
	expect(t, !s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentOutput).Valid, "s.highlightSemanticContent(s.pin(term51NPt(0, 5)).?, .output) == null")

	// Only prompt and input cells all the way down.
	pl6Cells(page, 5, "pppppppppp")
	pl6Cells(page, 6, "iiiiiiiiii")
	pl6Cells(page, 7, "iiiiiiiiii")
	pl6Cells(page, 8, "iiiiiiiiii")
	pl6Cells(page, 9, "iiiiiiiiii")
	expect(t, !s.HighlightSemanticContent(pl6At(t, &s, 0, 5), PageCellSemanticContentOutput).Valid, "s.highlightSemanticContent(s.pin(term51NPt(0, 5)).?, .output) == null")
}

// Zig: test "term51: highlightSemanticContent: output to the end of the screen without a next prompt"
func TestPl6_highlightSemanticContent_output_to_the_end_of_the_screen_without_a_next_prompt(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	l1 := n1.data.size.rows - 1

	n1.data.GetRowAndCell(0, uint(l1)).row.semantic_prompt = PageRowSemanticPromptPrompt
	pl6Cells(&n1.data, uint(l1), "ppii")
	pl6Cells(&n2.data, 0, "oooo")
	at := PageListPin{node: n1, y: l1}

	// The output is on the next page; the scan ends at the end of the list.
	hl := s.HighlightSemanticContent(at, PageCellSemanticContentOutput)
	expect(t, hl.Valid, "hl != null")
	pl6ExpectPinV(t, hl.Value.start, n2, 0, 0)
	pl6ExpectPinV(t, hl.Value.end, n2, 0, 3)

	// Output on the last row of the last page is found too.
	last := n2.data.size.rows - 1
	pl6Cells(&n2.data, uint(last), ".........o")
	hl = s.HighlightSemanticContent(at, PageCellSemanticContentOutput)
	expect(t, hl.Valid, "hl != null")
	pl6ExpectPinV(t, hl.Value.start, n2, 0, 0)
	pl6ExpectPinV(t, hl.Value.end, n2, uint(last), 9)

	// Input at the end of the screen with no output.
	prompt_at_end := PageListPin{node: n2, y: last}
	n2.data.GetRowAndCell(0, uint(last)).row.semantic_prompt = PageRowSemanticPromptPrompt
	pl6Cells(&n2.data, uint(last), "ppiiiiiiii")
	input := s.HighlightSemanticContent(prompt_at_end, PageCellSemanticContentInput)
	expect(t, input.Valid, "input != null")
	pl6ExpectPinV(t, input.Value.start, n2, uint(last), 2)
	pl6ExpectPinV(t, input.Value.end, n2, uint(last), 9)
	prompt := s.HighlightSemanticContent(prompt_at_end, PageCellSemanticContentPrompt)
	expect(t, prompt.Valid, "prompt != null")
	pl6ExpectPinV(t, prompt.Value.start, n2, uint(last), 0)
	pl6ExpectPinV(t, prompt.Value.end, n2, uint(last), 9)
	expect(t, !s.HighlightSemanticContent(prompt_at_end, PageCellSemanticContentOutput).Valid, "s.highlightSemanticContent(prompt_at_end, .output) == null")
}

// ---------------------------------------------------------------------------
// Go-only panic tests. Zig's unit tests cannot test an assert, an `unreachable`,
// a null unwrap or an integer overflow trap (the test would abort), so the
// tests above avoid these paths. Each test here drives one of them and
// expects the panic.
// ---------------------------------------------------------------------------

// Go-only: panic test: Pin.left: n greater than x (assert(n <= self.x))
func TestPl6_panic_Pin_left_n_greater_than_x(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 3)
	defer s.Deinit()
	p := PageListPin{node: s.pages.first, y: 1, x: 4}
	pl6ExpectPanic(t, func() { p.Left(5) })
}

// Go-only: panic test: Pin.right: x + n beyond the last column (assert(self.x + n < cols))
func TestPl6_panic_Pin_right_beyond_the_last_column(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 3)
	defer s.Deinit()
	p := PageListPin{node: s.pages.first, y: 1, x: 4}
	pl6ExpectPanic(t, func() { p.Right(6) })
	z := PageListPin{node: s.pages.first}
	pl6ExpectPanic(t, func() { z.Right(10) })
}

// Go-only: panic test: Pin.rightWrap: count that lands on a multiple of cols (usize underflow of `extra % cols - 1`)
func TestPl6_panic_Pin_rightWrap_extra_multiple_of_cols(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 8)
	defer s.Deinit()
	// 6 cells remain in the row, 16 - 6 = 10 is a multiple of cols.
	p := PageListPin{node: s.pages.first, y: 2, x: 3}
	pl6ExpectPanic(t, func() { p.RightWrap(16) })
	pl6ExpectPanic(t, func() { p.RightWrap(26) })
}

// Go-only: panic test: Pin.downOverflow: pin y beyond the rows of the page (usize underflow of `rows - (y + 1)`)
func TestPl6_panic_Pin_downOverflow_y_beyond_rows(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 6)
	defer s.Deinit()
	p := PageListPin{node: s.pages.first, y: 6}
	pl6ExpectPanic(t, func() { p.DownOverflow(1) })
	pl6ExpectPanic(t, func() { p.Down(0) })
}

// Go-only: panic test: Pin.isBetween: top below bottom on the same page (assert(top.y <= bottom.y))
func TestPl6_panic_Pin_isBetween_top_below_bottom(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 10)
	defer s.Deinit()
	node := s.pages.first
	top := PageListPin{node: node, y: 7, x: 2}
	bot := PageListPin{node: node, y: 3, x: 5}
	p := PageListPin{node: node, y: 5}
	pl6ExpectPanic(t, func() { p.IsBetween(top, bot) })
}

// Go-only: panic test: Pin.isBetween: top right of bottom on the same row (assert(top.x <= bottom.x))
func TestPl6_panic_Pin_isBetween_top_right_of_bottom(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 10)
	defer s.Deinit()
	node := s.pages.first
	top := PageListPin{node: node, y: 4, x: 6}
	bot := PageListPin{node: node, y: 4, x: 2}
	p := PageListPin{node: node, y: 4, x: 3}
	pl6ExpectPanic(t, func() { p.IsBetween(top, bot) })
}

// Go-only: panic test: Pin.isBetween: bottom page before the top page (assert(false) after the page walk)
func TestPl6_panic_Pin_isBetween_bottom_page_before_top_page(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	top := PageListPin{node: n2}
	bot := PageListPin{node: n1}
	p := PageListPin{node: n1, y: 1}
	pl6ExpectPanic(t, func() { p.IsBetween(top, bot) })
}

// Go-only: panic test: Pin.pageIterator: right_down limit before the pin (assert(self.eql(l) or self.before(l)))
func TestPl6_panic_Pin_pageIterator_right_down_limit_before_pin(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	a := PageListPin{node: n1, y: 3}
	pl6ExpectPanic(t, func() {
		a.PageIterator(PageListDirectionRightDown, Some(PageListPin{node: n1, y: 2}))
	})
	pl6ExpectPanic(t, func() {
		PageListPin{node: n2}.PageIterator(PageListDirectionRightDown, Some(PageListPin{node: n1, y: 9}))
	})
}

// Go-only: panic test: Pin.pageIterator: left_up limit after the pin (assert(self.eql(l) or l.before(self)))
func TestPl6_panic_Pin_pageIterator_left_up_limit_after_pin(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	n1 := s.pages.first
	n2 := n1.next
	a := PageListPin{node: n1, y: 3}
	pl6ExpectPanic(t, func() {
		a.PageIterator(PageListDirectionLeftUp, Some(PageListPin{node: n1, y: 4}))
	})
	pl6ExpectPanic(t, func() {
		a.PageIterator(PageListDirectionLeftUp, Some(PageListPin{node: n2}))
	})
}

// Go-only: panic test: Pin.rowIterator: limit before the pin (assert in Pin.pageIterator)
func TestPl6_panic_Pin_rowIterator_limit_before_pin(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 10)
	defer s.Deinit()
	node := s.pages.first
	a := PageListPin{node: node, y: 5}
	pl6ExpectPanic(t, func() {
		a.RowIterator(PageListDirectionRightDown, Some(PageListPin{node: node, y: 4}))
	})
	pl6ExpectPanic(t, func() {
		a.RowIterator(PageListDirectionLeftUp, Some(PageListPin{node: node, y: 6}))
	})
}

// Go-only: panic test: Pin.cellIterator: limit before the pin (assert in Pin.pageIterator)
func TestPl6_panic_Pin_cellIterator_limit_before_pin(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 10)
	defer s.Deinit()
	node := s.pages.first
	a := PageListPin{node: node, y: 5}
	pl6ExpectPanic(t, func() {
		a.CellIterator(PageListDirectionRightDown, Some(PageListPin{node: node, y: 4}))
	})
	pl6ExpectPanic(t, func() {
		a.CellIterator(PageListDirectionLeftUp, Some(PageListPin{node: node, y: 6}))
	})
}

// Go-only: panic test: PageIterator.nextDown: count limit of zero (assert(limit.* > 0))
func TestPl6_panic_PageIterator_nextDown_count_zero(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 5)
	defer s.Deinit()
	it := PageListPageIterator{
		row:       Some(PageListPin{node: s.pages.first, y: 1}),
		limit:     pl6LimitCount(0),
		direction: PageListDirectionRightDown,
	}
	pl6ExpectPanic(t, func() { it.nextDown() })
	it2 := PageListPageIterator{
		row:       Some(PageListPin{node: s.pages.first, y: 1}),
		limit:     pl6LimitCount(0),
		direction: PageListDirectionRightDown,
	}
	pl6ExpectPanic(t, func() { it2.Next() })
}

// Go-only: panic test: PageIterator.nextUp: count limit of zero (assert(limit.* > 0))
func TestPl6_panic_PageIterator_nextUp_count_zero(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 5)
	defer s.Deinit()
	it := PageListPageIterator{
		row:       Some(PageListPin{node: s.pages.first, y: 3}),
		limit:     pl6LimitCount(0),
		direction: PageListDirectionLeftUp,
	}
	pl6ExpectPanic(t, func() { it.nextUp() })
	it2 := PageListPageIterator{
		row:       Some(PageListPin{node: s.pages.first, y: 3}),
		limit:     pl6LimitCount(0),
		direction: PageListDirectionLeftUp,
	}
	pl6ExpectPanic(t, func() { it2.Next() })
}

// Go-only: panic test: PageIterator.nextUp: count limit from row zero (underflow of `row.y - 1`)
func TestPl6_panic_PageIterator_nextUp_count_from_row_zero(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 5)
	defer s.Deinit()
	it := PageListPageIterator{
		row:       Some(PageListPin{node: s.pages.first, y: 0}),
		limit:     pl6LimitCount(3),
		direction: PageListDirectionLeftUp,
	}
	pl6ExpectPanic(t, func() { it.nextUp() })
}

// Go-only: panic test: PageIterator.Chunk.rows: start after end (slice bounds)
func TestPl6_panic_PageIterator_Chunk_rows_start_after_end(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 8)
	defer s.Deinit()
	c := PageListPageIteratorChunk{node: s.pages.first, start: 5, end: 3}
	pl6ExpectPanic(t, func() { c.Rows() })
}

// Go-only: panic test: pageIterator: top-left point outside the list (null unwrap of self.pin(tl_pt))
func TestPl6_panic_pageIterator_top_left_point_outside(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 5)
	defer s.Deinit()
	pl6ExpectPanic(t, func() {
		s.PageIterator(PageListDirectionRightDown, pl6Pt(0, 1000), None[Point]())
	})
	pl6ExpectPanic(t, func() {
		s.PageIterator(PageListDirectionLeftUp, pl6Pt(10, 0), Some(pl6Pt(0, 3)))
	})
}

// Go-only: panic test: pageIterator: bottom-left point outside the list (null unwrap of self.pin(pt))
func TestPl6_panic_pageIterator_bottom_left_point_outside(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 5)
	defer s.Deinit()
	pl6ExpectPanic(t, func() {
		s.PageIterator(PageListDirectionRightDown, pl6Pt(0, 0), Some(pl6Pt(0, 1000)))
	})
}

// Go-only: panic test: pageIterator: top-left point after the bottom-left point (assert(tl_pin.eql(bl_pin) or tl_pin.before(bl_pin)))
func TestPl6_panic_pageIterator_top_left_after_bottom_left(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 5)
	defer s.Deinit()
	pl6ExpectPanic(t, func() {
		s.PageIterator(PageListDirectionRightDown, pl6Pt(0, 3), Some(pl6Pt(0, 1)))
	})
	pl6ExpectPanic(t, func() {
		s.PageIterator(PageListDirectionLeftUp, pl6Pt(5, 2), Some(pl6Pt(4, 2)))
	})
}

// Go-only: panic test: rowIterator: point outside the list (null unwrap of self.pin(pt))
func TestPl6_panic_rowIterator_point_outside(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 5)
	defer s.Deinit()
	pl6ExpectPanic(t, func() {
		s.RowIterator(PageListDirectionRightDown, pl6Pt(0, 1000), None[Point]())
	})
	pl6ExpectPanic(t, func() {
		s.RowIterator(PageListDirectionRightDown, pl6Pt(0, 0), Some(pl6Pt(0, 1000)))
	})
}

// Go-only: panic test: cellIterator: point outside the list (null unwrap of self.pin(pt))
func TestPl6_panic_cellIterator_point_outside(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 5)
	defer s.Deinit()
	pl6ExpectPanic(t, func() {
		s.CellIterator(PageListDirectionRightDown, pl6Pt(0, 1000), None[Point]())
	})
	pl6ExpectPanic(t, func() {
		s.CellIterator(PageListDirectionLeftUp, pl6Pt(0, 0), Some(pl6Pt(10, 1)))
	})
}

// Go-only: panic test: promptIterator: point outside the list (null unwrap of self.pin(pt))
func TestPl6_panic_promptIterator_point_outside(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 5)
	defer s.Deinit()
	pl6ExpectPanic(t, func() {
		s.PromptIterator(PageListDirectionRightDown, pl6Pt(0, 1000), None[Point]())
	})
	pl6ExpectPanic(t, func() {
		s.PromptIterator(PageListDirectionLeftUp, pl6Pt(0, 0), Some(pl6Pt(0, 1000)))
	})
}

// Go-only: panic test: getTopLeft: active with more rows than the list has (unreachable)
func TestPl6_panic_getTopLeft_active_rows_exceed_list(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 5)
	defer s.Deinit()
	s.rows = 100
	pl6ExpectPanic(t, func() { s.GetTopLeft(PointTagActive) })
	s.rows = 5
}

// Go-only: panic test: getBottomRight: viewport pin with fewer rows below it than the viewport (null unwrap of br.down(self.rows - 1))
func TestPl6_panic_getBottomRight_viewport_pin_near_the_end(t *testing.T) {
	testStdCapacity(t)
	s := pl6Pages(t, 10, 3, 2)
	defer s.Deinit()
	last := s.pages.last
	// Scroll never produces this (a pin in the active area becomes the active
	// viewport), so the viewport is set by hand.
	s.viewport = PageListViewport{Tag: PageListViewportPin}
	*s.viewport_pin = PageListPin{node: last, y: last.data.size.rows - 1}
	pl6ExpectPanic(t, func() { s.GetBottomRight(PointTagViewport) })
	s.viewport = PageListViewport{Tag: PageListViewportActive}
}

// Go-only: panic test: isDirty: point outside the list (null unwrap of self.getCell(pt))
func TestPl6_panic_isDirty_point_outside(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 5)
	defer s.Deinit()
	pl6ExpectPanic(t, func() { s.IsDirty(pl6Pt(0, 1000)) })
	pl6ExpectPanic(t, func() { s.IsDirty(pl6Pt(10, 0)) })
}

// Go-only: panic test: markDirty: point outside the list (null unwrap of self.pin(pt))
func TestPl6_panic_markDirty_point_outside(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 5)
	defer s.Deinit()
	pl6ExpectPanic(t, func() { s.markDirty(pl6Pt(0, 1000)) })
	pl6ExpectPanic(t, func() { s.markDirty(pl6Pt(10, 0)) })
}

// Go-only: panic test: highlightSemanticContent: no prompt at or after the pin (null unwrap of it.next())
func TestPl6_panic_highlightSemanticContent_no_prompt(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 20)
	defer s.Deinit()
	at := pl6At(t, &s, 0, 5)
	for _, content := range []PageCellSemanticContent{
		PageCellSemanticContentPrompt,
		PageCellSemanticContentInput,
		PageCellSemanticContentOutput,
	} {
		pl6ExpectPanic(t, func() { s.HighlightSemanticContent(at, content) })
	}
}

// Go-only: panic test: highlightSemanticContent: pin not on the first row of a prompt (assert(it.next().?.y == at.y))
func TestPl6_panic_highlightSemanticContent_pin_not_on_prompt_row(t *testing.T) {
	testStdCapacity(t)
	s := pl6Init(t, 10, 20)
	defer s.Deinit()
	pl6Semantic(&s.pages.first.data, "........P")
	// The first prompt at or after the pin is on row 8, not on the pin row.
	at := pl6At(t, &s, 0, 5)
	for _, content := range []PageCellSemanticContent{
		PageCellSemanticContentPrompt,
		PageCellSemanticContentInput,
		PageCellSemanticContentOutput,
	} {
		pl6ExpectPanic(t, func() { s.HighlightSemanticContent(at, content) })
	}
}
