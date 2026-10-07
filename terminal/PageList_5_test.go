// Tests translated one for one from src/terminal/PageList.zig (Ghostty v1.3.1
// plus the term51 additional tests), fifth part: the tests from
// "term51: viewportRowOffset: top and active" up to and including
// "term51: Pin.isDirty: page dirty row dirty or clean". Each Go test carries
// the exact Zig title in a "// Zig: test" comment. Helpers are prefixed pl5.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualSlices / expectEqualStrings keep Zig's names
//     and Zig's (expected, actual) argument order. `try` on a call that
//     returns an error is zigTry(t, err); `x.?` is zigUnwrap. Where Zig's
//     expectEqual compares values of different widths (`@as(usize, ..)`) the
//     Go call converts both sides to the same type.
//   - `try testing.expectError(error.X, call)` is pl5ExpectError(t, ErrX,
//     <the error of the call>).
//   - Every test starts with testStdCapacity(t): Zig's unit tests run with a
//     standard page capacity of grapheme_bytes 512 (see zig_defaults_test.go).
//   - `std_size` is pageListStdSize(), `PagePool.item_size` is
//     pageListPagePoolItemSize(), `page.memory.len` is pageListMemoryLen(&page)
//     and `initialCapacity` is pageListInitialCapacity. `?usize` max sizes are
//     Optional[uint64]; `std.math.maxInt(usize)` is math.MaxUint64.
//   - A Zig `{ ... defer s.deinit(); ... }` block is a Go closure called at
//     once, so the deferred Deinit runs at the end of the block as in Zig.
//   - Zig's `@field(cap, @tagName(kind))` over the IncreaseCapacity tags is a
//     switch on the kind (pl5CapField / pl5SetCapField).
//   - `.{ .screen = .{ .x = x, .y = y } }` is pl5Screen(x, y) (and pl5Active,
//     pl5History, pl5Viewport): a Point with only the field named by its Tag.
//   - Raw pointer comparisons of nodes, rows and cells are pointer comparisons;
//     `slice.ptr == other.ptr + n` is `&slice[0] == &other[n]`. Lines that
//     compare the addresses of page memory buffers are dropped with a
//     `// Zig-only:` comment.
//   - `term51LRawCell` reads a cell of a row past size.rows; it is
//     GetCells(&page.rows[y])[x].
//   - Tests that were not translated (FailingAllocator, memory-pool reuse,
//     the diagram signature test) are listed in
//     scripts/not-translated.txt. Panic tests (Go-only) follow the
//     translated ones at the end of the file.

package terminal

import (
	"errors"
	"math"
	"testing"
)

// pl5ExpectError is Zig's `try testing.expectError(expected, actual)`.
func pl5ExpectError(t *testing.T, want error, got error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

// pl5ExpectPanic runs f and fails the test if f does not panic.
func pl5ExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if recover() == nil {
			t.Fatalf("expected a panic, got none")
		}
	}()
	f()
}

// pl5Init is `try init(alloc, cols, rows, max_size)`.
func pl5Init(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt, max_size Optional[uint64]) *PageList {
	t.Helper()
	s, err := PageListInit(cols, rows, max_size)
	zigTry(t, err)
	return &s
}

// pl5Screen, pl5Active, pl5History and pl5Viewport are the Zig literals
// `.{ .screen = .{ .x = x, .y = y } }` and so on.
func pl5Screen(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagScreen, Screen: PointCoordinate{X: x, Y: y}}
}

func pl5Active(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagActive, Active: PointCoordinate{X: x, Y: y}}
}

func pl5History(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagHistory, History: PointCoordinate{X: x, Y: y}}
}

func pl5Viewport(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagViewport, Viewport: PointCoordinate{X: x, Y: y}}
}

// pl5PinOf is `s.pin(pt).?`.
func pl5PinOf(t *testing.T, s *PageList, pt Point) PageListPin {
	t.Helper()
	return zigUnwrap(t, s.Pin(pt))
}

// pl5Track is `try s.trackPin(p)`.
func pl5Track(t *testing.T, s *PageList, p PageListPin) *PageListPin {
	t.Helper()
	tracked, err := s.TrackPin(p)
	zigTry(t, err)
	return tracked
}

// pl5Grow is `try s.grow()`: the new node or nil.
func pl5Grow(t *testing.T, s *PageList) *PageListNode {
	t.Helper()
	n, err := s.Grow()
	zigTry(t, err)
	return n
}

// pl5GrowNode is `(try s.grow()).?`.
func pl5GrowNode(t *testing.T, s *PageList) *PageListNode {
	t.Helper()
	n := pl5Grow(t, s)
	if n == nil {
		t.Fatalf("unwrapped a null optional")
	}
	return n
}

// pl5Increase is `try s.increaseCapacity(node, adjustment)`.
func pl5Increase(t *testing.T, s *PageList, node *PageListNode, adjustment Optional[PageListIncreaseCapacity]) *PageListNode {
	t.Helper()
	n, err := s.IncreaseCapacity(node, adjustment)
	zigTry(t, err)
	return n
}

// pl5CreatePage is `try s.createPage(cap)`.
func pl5CreatePage(t *testing.T, s *PageList, cap PageCapacity) *PageListNode {
	t.Helper()
	n, err := s.createPage(cap)
	zigTry(t, err)
	return n
}

// pl5Compact is `try s.compact(node)`.
func pl5Compact(t *testing.T, s *PageList, node *PageListNode) *PageListNode {
	t.Helper()
	n, err := s.Compact(node)
	zigTry(t, err)
	return n
}

// pl5Kinds is std.meta.tags(IncreaseCapacity).
var pl5Kinds = []PageListIncreaseCapacity{
	PageListIncreaseCapacityStyles,
	PageListIncreaseCapacityGraphemeBytes,
	PageListIncreaseCapacityHyperlinkBytes,
	PageListIncreaseCapacityStringBytes,
}

// pl5CapField is `@field(cap, @tagName(kind))`, widened to uint64.
func pl5CapField(cap PageCapacity, kind PageListIncreaseCapacity) uint64 {
	switch kind {
	case PageListIncreaseCapacityStyles:
		return uint64(cap.styles)
	case PageListIncreaseCapacityGraphemeBytes:
		return uint64(cap.grapheme_bytes)
	case PageListIncreaseCapacityHyperlinkBytes:
		return uint64(cap.hyperlink_bytes)
	case PageListIncreaseCapacityStringBytes:
		return uint64(cap.string_bytes)
	}
	panic("unreachable")
}

// pl5SetCapField is `@field(cap, @tagName(kind)) = v` (v fits the field).
func pl5SetCapField(cap *PageCapacity, kind PageListIncreaseCapacity, v uint64) {
	switch kind {
	case PageListIncreaseCapacityStyles:
		cap.styles = SizeStyleCountInt(v)
	case PageListIncreaseCapacityGraphemeBytes:
		cap.grapheme_bytes = SizeGraphemeBytesInt(v)
	case PageListIncreaseCapacityHyperlinkBytes:
		cap.hyperlink_bytes = SizeHyperlinkCountInt(v)
	case PageListIncreaseCapacityStringBytes:
		cap.string_bytes = SizeStringBytesInt(v)
	}
}

// ---------------------------------------------------------------------------
// term51: additional tests (L).
// ---------------------------------------------------------------------------

// pl5LGrowRows is term51LGrowRows: grow by n rows with the (expensive)
// per-page integrity check paused on the page that receives the row so that
// large grows stay fast.
func pl5LGrowRows(t *testing.T, s *PageList, n uint) {
	t.Helper()
	for i := uint(0); i < n; i++ {
		last := s.pages.last
		last.data.PauseIntegrityChecks(true)
		func() {
			defer last.data.PauseIntegrityChecks(false)
			_, err := s.Grow()
			zigTry(t, err)
		}()
	}
}

// pl5LFillLast is term51LFillLast: grow until the last page has size.rows ==
// capacity.rows.
func pl5LFillLast(t *testing.T, s *PageList) {
	t.Helper()
	last := s.pages.last
	pl5LGrowRows(t, s, uint(last.data.capacity.rows)-uint(last.data.size.rows))
}

// pl5LInitPages is term51LInitPages: three pages of 80 columns, 24 active
// rows: the first two are full (capacity.rows rows each) and the last has
// last_rows rows.
func pl5LInitPages(t *testing.T, last_rows uint) *PageList {
	t.Helper()
	s := pl5Init(t, 80, 24, None[uint64]())
	pl5LFillLast(t, s)
	pl5GrowNode(t, s)
	pl5LFillLast(t, s)
	pl5GrowNode(t, s)
	pl5LGrowRows(t, s, last_rows-1)
	return s
}

// pl5LText is term51LText: set a plain text cell at the given pin.
func pl5LText(p PageListPin, cp uint32) {
	*p.RowAndCell().cell = PageCell{
		content_tag: PageCellContentTagCodepoint,
		content:     PageCellContent{codepoint: cp},
	}
}

// pl5LMakeOversized is term51LMakeOversized: make the page of the given node
// larger than a standard page by doubling its style capacity. Returns the
// replacement node.
func pl5LMakeOversized(t *testing.T, s *PageList, node *PageListNode) *PageListNode {
	t.Helper()
	n := node
	for pageListMemoryLen(&n.data) <= pageListStdSize() {
		n = pl5Increase(t, s, n, Some(PageListIncreaseCapacityStyles))
	}
	return n
}

// pl5LTwoFull is term51LTwoFull: two full pages (capacity.rows rows each) of
// 80 columns and 24 active rows.
func pl5LTwoFull(t *testing.T) *PageList {
	t.Helper()
	s := pl5Init(t, 80, 24, None[uint64]())
	pl5LFillLast(t, s)
	pl5GrowNode(t, s)
	pl5LFillLast(t, s)
	return s
}

// pl5LRawCell is term51LRawCell: the cell at x, y of the page's raw storage
// even if y is not below size.rows.
func pl5LRawCell(page *Page, x uint, y uint) PageCell {
	return page.GetCells(&page.rows[y])[x]
}

// pl5LForcePrune is term51LForcePrune: make the next grow on a list of full
// pages take the prune path. The explicit max size is 1, not 0, because zero
// disables scrolling.
func pl5LForcePrune(s *PageList) {
	s.explicit_max_size = 1
	s.min_max_size = 0
}

// pl5LCreatedBytes is term51LCreatedBytes: bytes that createPage accounts for
// a page: the whole pool item for pooled pages, the exact length otherwise.
func pl5LCreatedBytes(page *Page) uint64 {
	if pageListMemoryLen(page) <= pageListStdSize() {
		return pageListStdSize()
	}
	return pageListMemoryLen(page)
}

// pl5LFillPage is term51LFillPage: put one of everything in a page: text,
// shared styles, a grapheme cluster, a hyperlink and row flags.
func pl5LFillPage(t *testing.T, page *Page) {
	t.Helper()
	*page.GetRowAndCell(0, 0).cell = PageCell{content_tag: PageCellContentTagCodepoint, content: PageCellContent{codepoint: 'A'}}
	*page.GetRowAndCell(19, 4).cell = PageCell{content_tag: PageCellContentTagCodepoint, content: PageCellContent{codepoint: 'Z'}}

	// Two cells share bold; one cell has italic.
	bold, err := page.styles.Add(Style{Flags: StyleFlags{Bold: true}})
	zigTry(t, err)
	page.styles.Use(bold)
	italic, err := page.styles.Add(Style{Flags: StyleFlags{Italic: true}})
	zigTry(t, err)
	for _, e := range []struct {
		x  uint
		y  uint
		id StyleID
	}{
		{1, 0, bold},
		{2, 0, bold},
		{0, 1, italic},
	} {
		rac := page.GetRowAndCell(e.x, e.y)
		*rac.cell = PageCell{
			content_tag: PageCellContentTagCodepoint,
			content:     PageCellContent{codepoint: 'S'},
			style_id:    e.id,
		}
		rac.row.styled = true
	}

	{
		rac := page.GetRowAndCell(3, 0)
		*rac.cell = PageCell{
			content_tag: PageCellContentTagCodepoint,
			content:     PageCellContent{codepoint: 0x1F468},
		}
		zigTry(t, page.SetGraphemes(rac.row, rac.cell, []uint32{0x200D, 0x1F469}))
	}

	link, err := page.InsertHyperlink(Hyperlink{
		ID:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: 0},
		URI: []uint8("https://example.com"),
	})
	zigTry(t, err)
	{
		rac := page.GetRowAndCell(4, 2)
		*rac.cell = PageCell{
			content_tag: PageCellContentTagCodepoint,
			content:     PageCellContent{codepoint: 'L'},
		}
		zigTry(t, page.SetHyperlink(rac.row, rac.cell, link))
	}

	{
		rac := page.GetRowAndCell(0, 3)
		rac.row.wrap = true
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
		page.GetRowAndCell(0, 4).row.wrap_continuation = true
	}
}

// pl5LCheckPage is term51LCheckPage: verify what pl5LFillPage put in the page.
func pl5LCheckPage(t *testing.T, page *Page) {
	t.Helper()
	expectEqual(t, uint32('A'), page.GetRowAndCell(0, 0).cell.content.codepoint)
	expectEqual(t, uint32('Z'), page.GetRowAndCell(19, 4).cell.content.codepoint)

	expectEqual(t, uint(2), page.styles.Count())
	{
		a := page.GetRowAndCell(1, 0)
		b := page.GetRowAndCell(2, 0)
		expect(t, a.row.styled, "a.row.styled")
		expectEqual(t, a.cell.style_id, b.cell.style_id)
		expect(t, page.styles.Get(a.cell.style_id).Flags.Bold, "page.styles.get(page.memory, a.cell.style_id).flags.bold")
		c := page.GetRowAndCell(0, 1)
		expect(t, page.styles.Get(c.cell.style_id).Flags.Italic, "page.styles.get(page.memory, c.cell.style_id).flags.italic")
	}

	expectEqual(t, uint(1), page.GraphemeCount())
	{
		rac := page.GetRowAndCell(3, 0)
		expectEqual(t, uint32(0x1F468), rac.cell.content.codepoint)
		cps := zigUnwrap(t, page.LookupGrapheme(rac.cell))
		expectEqual(t, 2, len(cps))
		expectEqual(t, uint32(0x200D), cps[0])
		expectEqual(t, uint32(0x1F469), cps[1])
	}

	expectEqual(t, uint(1), page.hyperlink_set.Count())
	{
		rac := page.GetRowAndCell(4, 2)
		expect(t, rac.cell.hyperlink, "rac.cell.hyperlink")
		id := zigUnwrap(t, page.LookupHyperlink(rac.cell))
		entry := page.hyperlink_set.Get(id)
		expectEqualStrings(t, "https://example.com", string(entry.uri))
	}

	{
		rac := page.GetRowAndCell(0, 3)
		expect(t, rac.row.wrap, "rac.row.wrap")
		expectEqual(t, PageRowSemanticPromptPrompt, rac.row.semantic_prompt)
		expect(t, page.GetRowAndCell(0, 4).row.wrap_continuation, "page.getRowAndCell(0, 4).row.wrap_continuation")
		expect(t, !page.GetRowAndCell(0, 2).row.wrap, "!page.getRowAndCell(0, 2).row.wrap")
	}
}

// Zig: test "term51: viewportRowOffset: top and active"
func TestPl5_term51_viewportRowOffset_top_and_active(t *testing.T) {
	testStdCapacity(t)
	s := pl5LInitPages(t, 100)
	defer s.Deinit()

	expectEqual(t, s.total_rows-24, s.viewportRowOffset())
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	expectEqual(t, uint(0), s.viewportRowOffset())
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	expectEqual(t, s.total_rows-24, s.viewportRowOffset())

	// The cache is not involved.
	expectEqual(t, None[uint](), s.viewport_pin_row_offset)
}

// Zig: test "term51: viewportRowOffset: pin computes and caches"
func TestPl5_term51_viewportRowOffset_pin_computes_and_caches(t *testing.T) {
	testStdCapacity(t)
	s := pl5LInitPages(t, 100)
	defer s.Deinit()
	c := uint(s.pages.first.data.capacity.rows)
	first := s.pages.first
	second := first.next
	third := s.pages.last

	cases := []struct {
		node *PageListNode
		y    SizeCellCountInt
		row  uint
	}{
		{first, 0, 0},
		{first, 5, 5},
		{first, SizeCellCountInt(c - 1), c - 1},
		{second, 0, c},
		{second, 100, c + 100},
		{second, SizeCellCountInt(c - 1), 2*c - 1},
		{third, 0, 2 * c},
		{third, 75, 2*c + 75},
		// The first active row is a valid pin viewport: 24 rows remain.
		{third, 76, 2*c + 76},
	}
	for _, cs := range cases {
		s.viewport = PageListViewport{Tag: PageListViewportPin}
		*s.viewport_pin = PageListPin{Node: cs.node, Y: cs.y}
		s.viewport_pin_row_offset = None[uint]()
		expectEqual(t, cs.row, s.viewportRowOffset())
		expectEqual(t, Some(cs.row), s.viewport_pin_row_offset)

		// Second call is served from the cache and gives the same answer.
		expectEqual(t, cs.row, s.viewportRowOffset())
		expectEqual(t, Some(cs.row), s.viewport_pin_row_offset)
	}
	s.viewport = PageListViewport{Tag: PageListViewportActive}
	s.viewport_pin_row_offset = None[uint]()
}

// Zig: test "term51: viewportRowOffset: pin follows growth and a cached value is returned"
func TestPl5_term51_viewportRowOffset_pin_follows_growth_and_a_cached_value_is_returned(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl5LGrowRows(t, s, 100)
	s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: pl5PinOf(t, s, pl5Screen(0, 30))})
	expectEqual(t, uint(30), s.viewportRowOffset())

	// More rows below do not change the offset from the top.
	pl5LGrowRows(t, s, 50)
	expectEqual(t, uint(30), s.viewportRowOffset())
	expectEqual(t, Some[uint](30), s.viewport_pin_row_offset)
	expectEqual(t, uint(30), s.Scrollbar().Offset)
	expectEqual(t, uint(174), s.Scrollbar().Total)
}

// Zig: test "term51: fixupViewport: active is untouched"
func TestPl5_term51_fixupViewport_active_is_untouched(t *testing.T) {
	testStdCapacity(t)
	s := pl5LInitPages(t, 100)
	defer s.Deinit()

	s.fixupViewport(0)
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	s.fixupViewport(100000)
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	expectEqual(t, None[uint](), s.viewport_pin_row_offset)
}

// Zig: test "term51: fixupViewport: pin inside the active area becomes active"
func TestPl5_term51_fixupViewport_pin_inside_the_active_area_becomes_active(t *testing.T) {
	testStdCapacity(t)
	s := pl5LInitPages(t, 100)
	defer s.Deinit()
	third := s.pages.last

	// The first active row, with and without a cached offset.
	s.viewport = PageListViewport{Tag: PageListViewportPin}
	*s.viewport_pin = PageListPin{Node: third, Y: 76}
	s.viewport_pin_row_offset = None[uint]()
	s.fixupViewport(5)
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	// With a cache: the cache is not adjusted.
	s.viewport = PageListViewport{Tag: PageListViewportPin}
	*s.viewport_pin = PageListPin{Node: third, Y: 99}
	s.viewport_pin_row_offset = Some[uint](42)
	s.fixupViewport(5)
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	expectEqual(t, Some[uint](42), s.viewport_pin_row_offset)
	s.viewport_pin_row_offset = None[uint]()
}

// Zig: test "term51: fixupViewport: pin outside active adjusts the cached offset"
func TestPl5_term51_fixupViewport_pin_outside_active_adjusts_the_cached_offset(t *testing.T) {
	testStdCapacity(t)
	s := pl5LInitPages(t, 100)
	defer s.Deinit()
	third := s.pages.last
	s.viewport = PageListViewport{Tag: PageListViewportPin}
	*s.viewport_pin = PageListPin{Node: third, Y: 75}

	// No cache: nothing to adjust.
	s.viewport_pin_row_offset = None[uint]()
	s.fixupViewport(10)
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, None[uint](), s.viewport_pin_row_offset)

	// Zero removed rows.
	s.viewport_pin_row_offset = Some[uint](100)
	s.fixupViewport(0)
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, Some[uint](100), s.viewport_pin_row_offset)

	// Fewer removed than the offset.
	s.fixupViewport(99)
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, Some[uint](1), s.viewport_pin_row_offset)

	// Exactly the offset: it becomes 0 but the viewport stays a pin.
	s.viewport_pin_row_offset = Some[uint](100)
	s.fixupViewport(100)
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, Some[uint](0), s.viewport_pin_row_offset)

	// More than the offset: top, and the offset is left alone.
	s.viewport_pin_row_offset = Some[uint](100)
	s.fixupViewport(101)
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
	expectEqual(t, Some[uint](100), s.viewport_pin_row_offset)

	s.viewport = PageListViewport{Tag: PageListViewportActive}
	s.viewport_pin_row_offset = None[uint]()
}

// Zig: test "term51: fixupViewport: top becomes active only if active starts at the first row"
func TestPl5_term51_fixupViewport_top_becomes_active_only_if_active_starts_at_the_first_row(t *testing.T) {
	testStdCapacity(t)

	// History exists: stays top.
	func() {
		s := pl5LInitPages(t, 100)
		defer s.Deinit()
		s.viewport = PageListViewport{Tag: PageListViewportTop}
		s.fixupViewport(0)
		expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
		s.fixupViewport(50)
		expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
	}()

	// Same page with a one row history: still top.
	func() {
		s := pl5Init(t, 80, 24, None[uint64]())
		defer s.Deinit()
		pl5LGrowRows(t, s, 1)
		s.viewport = PageListViewport{Tag: PageListViewportTop}
		s.fixupViewport(0)
		expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
	}()

	// No history: the top is the active area.
	func() {
		s := pl5Init(t, 80, 24, None[uint64]())
		defer s.Deinit()
		s.viewport = PageListViewport{Tag: PageListViewportTop}
		s.fixupViewport(0)
		expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	}()
}

// Zig: test "term51: maxSize: explicit value above the minimum"
func TestPl5_term51_maxSize_explicit_value_above_the_minimum(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, Some[uint64](10*pageListStdSize()))
	defer s.Deinit()
	expectEqual(t, 2*pageListPagePoolItemSize(), s.min_max_size)
	expectEqual(t, 10*pageListStdSize(), s.MaxSize())
}

// Zig: test "term51: maxSize: explicit value below the minimum is raised"
func TestPl5_term51_maxSize_explicit_value_below_the_minimum_is_raised(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, Some[uint64](0))
	defer s.Deinit()
	expectEqual(t, s.min_max_size, s.MaxSize())

	s.explicit_max_size = 1
	expectEqual(t, s.min_max_size, s.MaxSize())
	s.explicit_max_size = s.min_max_size - 1
	expectEqual(t, s.min_max_size, s.MaxSize())
}

// Zig: test "term51: maxSize: equal to the minimum and just above"
func TestPl5_term51_maxSize_equal_to_the_minimum_and_just_above(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, Some[uint64](0))
	defer s.Deinit()
	s.explicit_max_size = s.min_max_size
	expectEqual(t, s.min_max_size, s.MaxSize())
	s.explicit_max_size = s.min_max_size + 1
	expectEqual(t, s.min_max_size+1, s.MaxSize())
}

// Zig: test "term51: maxSize: unlimited"
func TestPl5_term51_maxSize_unlimited(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	expectEqual(t, uint64(math.MaxUint64), s.MaxSize())
}

// Zig: test "term51: maxSize: minimum depends on rows needing several pages"
func TestPl5_term51_maxSize_minimum_depends_on_rows_needing_several_pages(t *testing.T) {
	testStdCapacity(t)
	// 80 columns fit 579 rows per page: 1000 rows need 2 pages plus one.
	s := pl5Init(t, 80, 1000, Some[uint64](0))
	defer s.Deinit()
	cap_rows := uint64(s.pages.first.data.capacity.rows)
	pages := (uint64(1000) + cap_rows - 1) / cap_rows // std.math.divCeil
	expectEqual(t, (pages+1)*pageListPagePoolItemSize(), s.MaxSize())
}

// Zig: test "term51: grow: fits in the last page without allocating"
func TestPl5_term51_grow_fits_in_the_last_page_without_allocating(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	last := s.pages.last
	c := uint(last.data.capacity.rows)
	page_size := s.page_size
	serial := s.page_serial

	expect(t, pl5Grow(t, s) == nil, "try s.grow() == null")
	expectEqual(t, uint(25), uint(last.data.size.rows))
	expectEqual(t, uint(25), s.total_rows)
	expectEqual(t, page_size, s.page_size)
	expectEqual(t, serial, s.page_serial)
	expectEqual(t, uint(1), s.TotalPages())

	// Up to and including the grow that fills the page.
	pl5LGrowRows(t, s, c-25-1)
	expectEqual(t, c-1, uint(last.data.size.rows))
	expect(t, pl5Grow(t, s) == nil, "try s.grow() == null")
	expectEqual(t, c, uint(last.data.size.rows))
	expectEqual(t, c, s.total_rows)
	expectEqual(t, uint(1), s.TotalPages())
	expectEqual(t, page_size, s.page_size)
	expectEqual(t, serial, s.page_serial)
}

// Zig: test "term51: grow: fast path ignores the max size"
func TestPl5_term51_grow_fast_path_ignores_the_max_size(t *testing.T) {
	testStdCapacity(t)
	s := pl5LTwoFull(t)
	defer s.Deinit()
	// Add one row to a third page, then forbid growth: the third page has
	// space, so no pruning happens.
	pl5GrowNode(t, s)
	pl5LForcePrune(s)
	total := s.total_rows
	page_size := s.page_size

	expect(t, pl5Grow(t, s) == nil, "try s.grow() == null")
	expectEqual(t, total+1, s.total_rows)
	expectEqual(t, page_size, s.page_size)
	expectEqual(t, uint(3), s.TotalPages())
}

// Zig: test "term51: grow: allocates a new page when the last one is full"
func TestPl5_term51_grow_allocates_a_new_page_when_the_last_one_is_full(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl5LFillLast(t, s)
	old_last := s.pages.last
	c := uint(old_last.data.capacity.rows)
	serial := s.page_serial
	page_size := s.page_size
	total := s.total_rows

	new := pl5GrowNode(t, s)

	expectEqual(t, new, s.pages.last)
	expectEqual(t, old_last, new.prev)
	expectEqual(t, new, old_last.next)
	expect(t, new.next == nil, "new.next == null")
	expectEqual(t, old_last, s.pages.first)
	expectEqual(t, uint(1), uint(new.data.size.rows))
	expectEqual(t, pageListInitialCapacity(80), new.data.capacity)
	expectEqual(t, c, uint(old_last.data.size.rows))
	expectEqual(t, total+1, s.total_rows)
	expectEqual(t, serial, new.serial)
	expectEqual(t, serial+1, s.page_serial)
	expectEqual(t, uint64(0), s.page_serial_min)
	expectEqual(t, page_size+pageListStdSize(), s.page_size)
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	// The active area now spans the two pages.
	expectEqual(t, old_last, s.GetTopLeft(PointTagActive).Node)
	expectEqual(t, SizeCellCountInt(c-23), s.GetTopLeft(PointTagActive).Y)
}

// Zig: test "term51: grow: a single page is never pruned even over the max size"
func TestPl5_term51_grow_a_single_page_is_never_pruned_even_over_the_max_size(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl5LFillLast(t, s)
	pl5LForcePrune(s)
	first := s.pages.first
	total := s.total_rows
	expectEqual(t, uint(1), s.TotalPages())
	expect(t, s.page_size+pageListPagePoolItemSize() > s.MaxSize(), "s.page_size + PagePool.item_size > s.maxSize()")

	new := pl5GrowNode(t, s)
	expectEqual(t, uint(2), s.TotalPages())
	expectEqual(t, first, s.pages.first)
	expectEqual(t, new, s.pages.last)
	expectEqual(t, total+1, s.total_rows)
}

// Zig: test "term51: grow: no pruning when the new page exactly fits the max size"
func TestPl5_term51_grow_no_pruning_when_the_new_page_exactly_fits_the_max_size(t *testing.T) {
	testStdCapacity(t)
	s := pl5LTwoFull(t)
	defer s.Deinit()
	first := s.pages.first
	total := s.total_rows
	expectEqual(t, 2*pageListPagePoolItemSize(), s.page_size)

	// page_size + item_size == maxSize(): not above it, so allocate.
	s.explicit_max_size = 3 * pageListPagePoolItemSize()
	new := pl5GrowNode(t, s)
	expectEqual(t, uint(3), s.TotalPages())
	expectEqual(t, first, s.pages.first)
	expectEqual(t, new, s.pages.last)
	expect(t, new != first, "new != first")
	expectEqual(t, total+1, s.total_rows)
	expectEqual(t, 3*pageListPagePoolItemSize(), s.page_size)
}

// Zig: test "term51: grow: prunes and reuses the first page one byte over the max size"
func TestPl5_term51_grow_prunes_and_reuses_the_first_page_one_byte_over_the_max_size(t *testing.T) {
	testStdCapacity(t)
	s := pl5LTwoFull(t)
	defer s.Deinit()
	c := uint(s.pages.first.data.capacity.rows)
	first := s.pages.first
	second := first.next
	first_serial := first.serial
	second_serial := second.serial
	pl5LText(PageListPin{Node: first, Y: 5, X: 2}, 'F')
	pl5LText(PageListPin{Node: second, Y: 5, X: 2}, 'S')
	serial := s.page_serial
	page_size := s.page_size

	s.explicit_max_size = 3*pageListPagePoolItemSize() - 1
	new := pl5GrowNode(t, s)

	// The first node object and its memory were reused as the last page.
	expectEqual(t, first, new)
	expectEqual(t, second, s.pages.first)
	expectEqual(t, new, s.pages.last)
	expectEqual(t, new, second.next)
	expectEqual(t, second, new.prev)
	expect(t, new.next == nil, "new.next == null")
	expect(t, second.prev == nil, "second.prev == null")
	expectEqual(t, uint(2), s.TotalPages())

	// Rows: the pruned page's rows are gone, one row added.
	expectEqual(t, c+1, s.total_rows)
	expectEqual(t, uint(1), uint(new.data.size.rows))
	expectEqual(t, pageListInitialCapacity(80), new.data.capacity)
	expectEqual(t, page_size, s.page_size)

	// Serial numbers.
	expectEqual(t, serial, new.serial)
	expectEqual(t, serial+1, s.page_serial)
	expectEqual(t, first_serial+1, s.page_serial_min)
	expect(t, second_serial >= s.page_serial_min, "second_serial >= s.page_serial_min")

	// Memory was cleared, the other page is intact.
	expect(t, pl5LRawCell(&new.data, 2, 5).IsEmpty(), "term51LRawCell(&new.data, 2, 5).isEmpty()")
	expect(t, new.data.GetRowAndCell(2, 0).cell.IsEmpty(), "new.data.getRowAndCell(2, 0).cell.isEmpty()")
	expectEqual(t, uint32('S'), second.data.GetRowAndCell(2, 5).cell.content.codepoint)
}

// Zig: test "term51: grow: pruning needs the remaining rows to cover the active area"
func TestPl5_term51_grow_pruning_needs_the_remaining_rows_to_cover_the_active_area(t *testing.T) {
	testStdCapacity(t)

	// Remaining rows (c) + the new row must be at least rows: with rows
	// c + 2 the prune is undone and a new page is allocated instead.
	func() {
		s := pl5LTwoFull(t)
		defer s.Deinit()
		c := uint(s.pages.first.data.capacity.rows)
		first := s.pages.first
		second := first.next
		pl5LForcePrune(s)
		s.rows = SizeCellCountInt(c + 2)

		// Viewport state is untouched by the undone prune.
		s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: PageListPin{Node: first, Y: 10}})
		expectEqual(t, uint(10), s.Scrollbar().Offset)
		tracked := pl5Track(t, s, PageListPin{Node: first, Y: 3, X: 4})
		page_size := s.page_size
		serial_min := s.page_serial_min

		new := pl5GrowNode(t, s)
		expectEqual(t, uint(3), s.TotalPages())
		expectEqual(t, first, s.pages.first)
		expectEqual(t, second, first.next)
		expectEqual(t, new, second.next)
		expectEqual(t, new, s.pages.last)
		expect(t, new != first, "new != first")
		expectEqual(t, 2*c+1, s.total_rows)
		expectEqual(t, page_size+pageListStdSize(), s.page_size)
		expectEqual(t, serial_min, s.page_serial_min)
		expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
		expectEqual(t, Some[uint](10), s.viewport_pin_row_offset)
		expectEqual(t, first, s.viewport_pin.Node)
		expectEqual(t, SizeCellCountInt(10), s.viewport_pin.Y)
		expect(t, !s.viewport_pin.Garbage, "!s.viewport_pin.garbage")
		expectEqual(t, first, tracked.Node)
		expectEqual(t, SizeCellCountInt(3), tracked.Y)
		expectEqual(t, SizeCellCountInt(4), tracked.X)
		expect(t, !tracked.Garbage, "!tracked.garbage")
		s.rows = 24
	}()

	// Remaining rows + 1 == rows: pruning is allowed.
	func() {
		s := pl5LTwoFull(t)
		defer s.Deinit()
		c := uint(s.pages.first.data.capacity.rows)
		first := s.pages.first
		pl5LForcePrune(s)
		s.rows = SizeCellCountInt(c + 1)

		new := pl5GrowNode(t, s)
		expectEqual(t, first, new)
		expectEqual(t, uint(2), s.TotalPages())
		expectEqual(t, c+1, s.total_rows)
		s.rows = 24
	}()
}

// Zig: test "term51: grow: pruning moves tracked pins on the pruned page to the new top left"
func TestPl5_term51_grow_pruning_moves_tracked_pins_on_the_pruned_page_to_the_new_top_left(t *testing.T) {
	testStdCapacity(t)
	s := pl5LTwoFull(t)
	defer s.Deinit()
	c := uint(s.pages.first.data.capacity.rows)
	first := s.pages.first
	second := first.next

	pa := pl5Track(t, s, PageListPin{Node: first, X: 5, Y: 100})
	pb := pl5Track(t, s, PageListPin{Node: first, X: 7, Y: SizeCellCountInt(c - 1)})
	pc := pl5Track(t, s, PageListPin{Node: second, X: 3, Y: 4})
	pd := pl5Track(t, s, PageListPin{Node: second, X: 0, Y: 0})
	pd.Garbage = true
	// The viewport pin is on the first page (viewport is active). Its
	// garbage flag is cleared after pruning even if it was set.
	expectEqual(t, first, s.viewport_pin.Node)
	s.viewport_pin.Garbage = true

	pl5LForcePrune(s)
	new := pl5GrowNode(t, s)
	expectEqual(t, first, new)

	for _, p := range []*PageListPin{pa, pb} {
		expectEqual(t, second, p.Node)
		expectEqual(t, SizeCellCountInt(0), p.X)
		expectEqual(t, SizeCellCountInt(0), p.Y)
		expect(t, p.Garbage, "p.garbage")
	}
	expectEqual(t, second, pc.Node)
	expectEqual(t, SizeCellCountInt(3), pc.X)
	expectEqual(t, SizeCellCountInt(4), pc.Y)
	expect(t, !pc.Garbage, "!pc.garbage")
	expectEqual(t, second, pd.Node)
	expect(t, pd.Garbage, "pd.garbage")

	expectEqual(t, second, s.viewport_pin.Node)
	expectEqual(t, SizeCellCountInt(0), s.viewport_pin.X)
	expectEqual(t, SizeCellCountInt(0), s.viewport_pin.Y)
	expect(t, !s.viewport_pin.Garbage, "!s.viewport_pin.garbage")
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
}

// Zig: test "term51: grow: pruning with active and top viewports"
func TestPl5_term51_grow_pruning_with_active_and_top_viewports(t *testing.T) {
	testStdCapacity(t)

	func() {
		s := pl5LTwoFull(t)
		defer s.Deinit()
		pl5LForcePrune(s)
		pl5GrowNode(t, s)
		expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	}()
	func() {
		s := pl5LTwoFull(t)
		defer s.Deinit()
		s.Scroll(PageListScroll{Tag: PageListScrollTop})
		pl5LForcePrune(s)
		pl5GrowNode(t, s)
		expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
		expectEqual(t, uint(0), s.viewportRowOffset())
	}()
}

// Zig: test "term51: grow: pruning updates a cached pin viewport offset"
func TestPl5_term51_grow_pruning_updates_a_cached_pin_viewport_offset(t *testing.T) {
	testStdCapacity(t)

	// Pin in the page that is pruned: the cached offset is below the
	// pruned rows so the viewport becomes top. Check both ends of the page.
	for _, row_in_first := range []uint{0, 1, 100} {
		func() {
			s := pl5LTwoFull(t)
			defer s.Deinit()
			first := s.pages.first
			second := first.next
			s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: PageListPin{Node: first, Y: SizeCellCountInt(row_in_first)}})
			if row_in_first == 0 {
				expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
				s.viewport = PageListViewport{Tag: PageListViewportPin}
				*s.viewport_pin = PageListPin{Node: first, Y: 0}
			}
			expectEqual(t, row_in_first, s.Scrollbar().Offset)
			pl5LForcePrune(s)
			pl5GrowNode(t, s)
			expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
			expectEqual(t, second, s.viewport_pin.Node)
			expectEqual(t, SizeCellCountInt(0), s.viewport_pin.Y)
			expect(t, !s.viewport_pin.Garbage, "!s.viewport_pin.garbage")
		}()
	}

	// The last row of the pruned page.
	func() {
		s := pl5LTwoFull(t)
		defer s.Deinit()
		c := uint(s.pages.first.data.capacity.rows)
		s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: PageListPin{Node: s.pages.first, Y: SizeCellCountInt(c - 1)}})
		expectEqual(t, c-1, s.Scrollbar().Offset)
		pl5LForcePrune(s)
		pl5GrowNode(t, s)
		expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
	}()

	// First row of the next page: offset is exactly the pruned row count,
	// so it stays a pin with offset 0.
	func() {
		s := pl5LTwoFull(t)
		defer s.Deinit()
		c := uint(s.pages.first.data.capacity.rows)
		second := s.pages.first.next
		s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: PageListPin{Node: second, Y: 0}})
		expectEqual(t, c, s.Scrollbar().Offset)
		pl5LForcePrune(s)
		pl5GrowNode(t, s)
		expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
		expectEqual(t, Some[uint](0), s.viewport_pin_row_offset)
		expectEqual(t, second, s.viewport_pin.Node)
		expectEqual(t, uint(0), s.Scrollbar().Offset)
	}()

	// Further in: the offset is reduced.
	func() {
		s := pl5LTwoFull(t)
		defer s.Deinit()
		c := uint(s.pages.first.data.capacity.rows)
		second := s.pages.first.next
		s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: PageListPin{Node: second, Y: 50}})
		expectEqual(t, c+50, s.Scrollbar().Offset)
		pl5LForcePrune(s)
		pl5GrowNode(t, s)
		expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
		expectEqual(t, Some[uint](50), s.viewport_pin_row_offset)
		expectEqual(t, uint(50), s.Scrollbar().Offset)
	}()
}

// Zig: test "term51: grow: pruning a pin viewport without a cached offset"
func TestPl5_term51_grow_pruning_a_pin_viewport_without_a_cached_offset(t *testing.T) {
	testStdCapacity(t)

	// Pin in the pruned page and no cache: it is moved to the new top
	// left but the viewport is still a pin (not converted to top).
	func() {
		s := pl5LTwoFull(t)
		defer s.Deinit()
		c := uint(s.pages.first.data.capacity.rows)
		second := s.pages.first.next
		s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: PageListPin{Node: s.pages.first, Y: SizeCellCountInt(c - 1)}})
		expectEqual(t, None[uint](), s.viewport_pin_row_offset)
		pl5LForcePrune(s)
		pl5GrowNode(t, s)
		expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
		expectEqual(t, second, s.viewport_pin.Node)
		expectEqual(t, SizeCellCountInt(0), s.viewport_pin.Y)
		expect(t, !s.viewport_pin.Garbage, "!s.viewport_pin.garbage")
		expectEqual(t, uint(0), s.Scrollbar().Offset)
	}()

	// Pin on the surviving page and no cache.
	func() {
		s := pl5LTwoFull(t)
		defer s.Deinit()
		second := s.pages.first.next
		s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: PageListPin{Node: second, Y: 70}})
		expectEqual(t, None[uint](), s.viewport_pin_row_offset)
		pl5LForcePrune(s)
		pl5GrowNode(t, s)
		expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
		expectEqual(t, None[uint](), s.viewport_pin_row_offset)
		expectEqual(t, uint(70), s.Scrollbar().Offset)
	}()
}

// Zig: test "term51: grow: pruning counts the rows of the pruned page not its capacity"
func TestPl5_term51_grow_pruning_counts_the_rows_of_the_pruned_page_not_its_capacity(t *testing.T) {
	testStdCapacity(t)
	s := pl5LTwoFull(t)
	defer s.Deinit()
	c := uint(s.pages.first.data.capacity.rows)
	first := s.pages.first

	// Make the first page a short one: 100 rows.
	zigTry(t, s.Split(PageListPin{Node: first, Y: 100}))
	expectEqual(t, uint(3), s.TotalPages())
	expectEqual(t, 2*c, s.total_rows)

	pl5LForcePrune(s)
	new := pl5GrowNode(t, s)
	expectEqual(t, first, new)
	expectEqual(t, uint(3), s.TotalPages())
	expectEqual(t, 2*c-100+1, s.total_rows)
	expectEqual(t, uint(1), uint(new.data.size.rows))
}

// Zig: test "term51: grow: a non-standard first page is destroyed and a new page allocated"
func TestPl5_term51_grow_a_non_standard_first_page_is_destroyed_and_a_new_page_allocated(t *testing.T) {
	testStdCapacity(t)
	s := pl5LTwoFull(t)
	defer s.Deinit()
	c := uint(s.pages.first.data.capacity.rows)
	first := pl5LMakeOversized(t, s, s.pages.first)
	second := first.next
	first_len := pageListMemoryLen(&first.data)
	expect(t, first_len > pageListStdSize(), "first_len > std_size")
	expectEqual(t, first, s.pages.first)

	tracked := pl5Track(t, s, PageListPin{Node: first, X: 6, Y: 20})
	serial := s.page_serial
	serial_min := s.page_serial_min
	page_size := s.page_size
	pl5LForcePrune(s)

	new := pl5GrowNode(t, s)
	expectEqual(t, second, s.pages.first)
	expectEqual(t, new, s.pages.last)
	expectEqual(t, uint(2), s.TotalPages())
	expectEqual(t, c+1, s.total_rows)
	expectEqual(t, uint(1), uint(new.data.size.rows))

	// Accounting: the oversized page's length is subtracted and a pooled
	// page added.
	expectEqual(t, page_size-first_len+pageListStdSize(), s.page_size)

	// New page serial, but no minimum bump since nothing was reused.
	expectEqual(t, serial, new.serial)
	expectEqual(t, serial+1, s.page_serial)
	expectEqual(t, serial_min, s.page_serial_min)

	// Pins on the destroyed page moved to the new top left.
	expectEqual(t, second, tracked.Node)
	expectEqual(t, SizeCellCountInt(0), tracked.X)
	expectEqual(t, SizeCellCountInt(0), tracked.Y)
	expect(t, tracked.Garbage, "tracked.garbage")
}

// Zig: test "term51: grow: a non-standard first page is kept when pruning would drop the active area"
func TestPl5_term51_grow_a_non_standard_first_page_is_kept_when_pruning_would_drop_the_active_area(t *testing.T) {
	testStdCapacity(t)
	s := pl5LTwoFull(t)
	defer s.Deinit()
	c := uint(s.pages.first.data.capacity.rows)
	first := pl5LMakeOversized(t, s, s.pages.first)
	first_len := pageListMemoryLen(&first.data)
	page_size := s.page_size
	pl5LForcePrune(s)
	s.rows = SizeCellCountInt(c + 2)

	new := pl5GrowNode(t, s)
	expectEqual(t, first, s.pages.first)
	expectEqual(t, first_len, pageListMemoryLen(&first.data))
	expectEqual(t, new, s.pages.last)
	expectEqual(t, uint(3), s.TotalPages())
	expectEqual(t, 2*c+1, s.total_rows)
	expectEqual(t, page_size+pageListStdSize(), s.page_size)
	s.rows = 24
}

// Zig: test "term51: grow: new and reused pages use the current column count"
func TestPl5_term51_grow_new_and_reused_pages_use_the_current_column_count(t *testing.T) {
	testStdCapacity(t)

	// New page.
	func() {
		s := pl5Init(t, 80, 24, None[uint64]())
		defer s.Deinit()
		pl5LFillLast(t, s)
		s.cols = 40
		new := pl5GrowNode(t, s)
		expectEqual(t, pageListInitialCapacity(40), new.data.capacity)
		expectEqual(t, SizeCellCountInt(40), new.data.size.cols)
		expectEqual(t, SizeCellCountInt(1), new.data.size.rows)
		s.cols = 80
	}()

	// Reused page.
	func() {
		s := pl5LTwoFull(t)
		defer s.Deinit()
		s.cols = 40
		pl5LForcePrune(s)
		new := pl5GrowNode(t, s)
		expectEqual(t, pageListInitialCapacity(40), new.data.capacity)
		expectEqual(t, SizeCellCountInt(40), new.data.size.cols)
		expectEqual(t, SizeCellCountInt(1), new.data.size.rows)
		s.cols = 80
	}()
}

// Zig: test "term51: grow: layout reused page memory length follows the new column count"
func TestPl5_term51_grow_layout_reused_page_memory_length_follows_the_new_column_count(t *testing.T) {
	testStdCapacity(t)
	s := pl5LTwoFull(t)
	defer s.Deinit()
	page_size := s.page_size
	s.cols = 40
	pl5LForcePrune(s)
	new := pl5GrowNode(t, s)

	// The reused page's slice is the new layout's size, which can be
	// smaller than the pool item. page_size is not updated for reuse.
	expectEqual(t, PageLayoutOf(pageListInitialCapacity(40)).total_size, pageListMemoryLen(&new.data))
	expect(t, pageListMemoryLen(&new.data) <= pageListStdSize(), "new.data.memory.len <= std_size")
	expectEqual(t, page_size, s.page_size)
	s.cols = 80
}

// Zig: test "term51: grow: steady state at the minimum max size"
func TestPl5_term51_grow_steady_state_at_the_minimum_max_size(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, Some[uint64](0))
	defer s.Deinit()
	c := uint(s.pages.first.data.capacity.rows)

	// Grow through many pages' worth of rows. At the minimum max size
	// (two pages) the list settles at two pages of std_size.
	pl5LGrowRows(t, s, 6*c)
	expectEqual(t, uint(2), s.TotalPages())
	expectEqual(t, 2*pageListStdSize(), s.page_size)
	expect(t, s.total_rows >= c+1, "s.total_rows >= c + 1")
	expect(t, s.total_rows <= 2*c, "s.total_rows <= 2 * c")
	expectEqual(t, s.totalRows(), s.total_rows)

	// 3 more pages than started with have been allocated or reused.
	expectEqual(t, uint64(2+5), s.page_serial)
	expect(t, s.page_serial_min > 0, "s.page_serial_min > 0")
}

// Zig: test "term51: increaseCapacity: each kind doubles only its own dimension"
func TestPl5_term51_increaseCapacity_each_kind_doubles_only_its_own_dimension(t *testing.T) {
	testStdCapacity(t)
	for _, kind := range pl5Kinds {
		func() {
			s := pl5Init(t, 20, 5, None[uint64]())
			defer s.Deinit()
			node := s.pages.first
			before := node.data.capacity
			old_len := pageListMemoryLen(&node.data)
			page_size := s.page_size
			serial := s.page_serial
			total := s.total_rows

			new := pl5Increase(t, s, node, Some(kind))

			want := before
			pl5SetCapField(&want, kind, pl5CapField(before, kind)*2)
			expectEqual(t, want, new.data.capacity)
			expect(t, new != node, "new != node")

			// Same rows/cols/serial bookkeeping.
			expectEqual(t, SizeCellCountInt(5), new.data.size.rows)
			expectEqual(t, SizeCellCountInt(20), new.data.size.cols)
			expectEqual(t, new, s.pages.first)
			expectEqual(t, new, s.pages.last)
			expect(t, new.prev == nil, "new.prev == null")
			expect(t, new.next == nil, "new.next == null")
			expectEqual(t, total, s.total_rows)
			expectEqual(t, serial, new.serial)
			expectEqual(t, serial+1, s.page_serial)
			expectEqual(t, page_size+pl5LCreatedBytes(&new.data)-old_len, s.page_size)
		}()
	}
}

// Zig: test "term51: increaseCapacity: null adjustment recreates the page with the same capacity"
func TestPl5_term51_increaseCapacity_null_adjustment_recreates_the_page_with_the_same_capacity(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 20, 5, None[uint64]())
	defer s.Deinit()
	pl5LFillPage(t, &s.pages.first.data)
	node := s.pages.first
	before := node.data.capacity
	page_size := s.page_size
	serial := s.page_serial

	new := pl5Increase(t, s, node, None[PageListIncreaseCapacity]())
	expect(t, new != node, "new != node")
	expectEqual(t, before, new.data.capacity)
	expectEqual(t, new, s.pages.first)
	expectEqual(t, serial, new.serial)
	expectEqual(t, serial+1, s.page_serial)
	expectEqual(t, page_size, s.page_size)
	pl5LCheckPage(t, &new.data)
}

// Zig: test "term51: increaseCapacity: contents survive every kind"
func TestPl5_term51_increaseCapacity_contents_survive_every_kind(t *testing.T) {
	testStdCapacity(t)
	for _, kind := range pl5Kinds {
		func() {
			s := pl5Init(t, 20, 5, None[uint64]())
			defer s.Deinit()
			pl5LFillPage(t, &s.pages.first.data)
			new := pl5Increase(t, s, s.pages.first, Some(kind))
			pl5LCheckPage(t, &new.data)

			// And again from the grown page.
			again := pl5Increase(t, s, new, Some(kind))
			pl5LCheckPage(t, &again.data)
		}()
	}
}

// Zig: test "term51: increaseCapacity: tracked pins and viewport follow the page"
func TestPl5_term51_increaseCapacity_tracked_pins_and_viewport_follow_the_page(t *testing.T) {
	testStdCapacity(t)
	s := pl5LInitPages(t, 100)
	defer s.Deinit()
	c := uint(s.pages.first.data.capacity.rows)
	first := s.pages.first
	second := first.next
	third := s.pages.last

	on_second_a := pl5Track(t, s, PageListPin{Node: second, X: 5, Y: 10})
	on_second_b := pl5Track(t, s, PageListPin{Node: second, X: 79, Y: SizeCellCountInt(c - 1)})
	on_first := pl5Track(t, s, PageListPin{Node: first, X: 2, Y: 3})
	on_third := pl5Track(t, s, PageListPin{Node: third, X: 1, Y: 0})
	on_second_b.Garbage = true
	s.Scroll(PageListScroll{Tag: PageListScrollPin, Pin: PageListPin{Node: second, Y: 40}})
	expectEqual(t, c+40, s.Scrollbar().Offset)

	new := pl5Increase(t, s, second, Some(PageListIncreaseCapacityStyles))

	// List position.
	expectEqual(t, first, s.pages.first)
	expectEqual(t, third, s.pages.last)
	expectEqual(t, new, first.next)
	expectEqual(t, first, new.prev)
	expectEqual(t, third, new.next)
	expectEqual(t, new, third.prev)
	expectEqual(t, uint(3), s.TotalPages())

	// Pins.
	expectEqual(t, new, on_second_a.Node)
	expectEqual(t, SizeCellCountInt(5), on_second_a.X)
	expectEqual(t, SizeCellCountInt(10), on_second_a.Y)
	expectEqual(t, new, on_second_b.Node)
	expectEqual(t, SizeCellCountInt(79), on_second_b.X)
	expectEqual(t, SizeCellCountInt(c-1), on_second_b.Y)
	expect(t, on_second_b.Garbage, "on_second_b.garbage")
	expectEqual(t, first, on_first.Node)
	expectEqual(t, third, on_third.Node)

	// Viewport pin and its cached offset.
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, new, s.viewport_pin.Node)
	expectEqual(t, SizeCellCountInt(40), s.viewport_pin.Y)
	expectEqual(t, Some(c+40), s.viewport_pin_row_offset)
	expectEqual(t, c+40, s.Scrollbar().Offset)

	// First and last pages too.
	new_first := pl5Increase(t, s, first, None[PageListIncreaseCapacity]())
	expectEqual(t, new_first, s.pages.first)
	expectEqual(t, new_first, on_first.Node)
	expectEqual(t, new, new_first.next)
	expect(t, new_first.prev == nil, "new_first.prev == null")
	new_third := pl5Increase(t, s, third, None[PageListIncreaseCapacity]())
	expectEqual(t, new_third, s.pages.last)
	expectEqual(t, new_third, on_third.Node)
	expectEqual(t, new, new_third.prev)
	expect(t, new_third.next == nil, "new_third.next == null")
}

// Zig: test "term51: increaseCapacity: keeps the column count a page was shrunk to"
func TestPl5_term51_increaseCapacity_keeps_the_column_count_a_page_was_shrunk_to(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 20, 3, Some[uint64](0))
	defer s.Deinit()
	zigTry(t, s.Resize(PageListResize{Cols: Some[SizeCellCountInt](10), Reflow: false}))
	expectEqual(t, SizeCellCountInt(10), s.pages.first.data.size.cols)
	expect(t, s.pages.first.data.capacity.cols >= 20, "s.pages.first.?.data.capacity.cols >= 20")

	new := pl5Increase(t, s, s.pages.first, Some(PageListIncreaseCapacityStyles))
	expectEqual(t, SizeCellCountInt(10), new.data.size.cols)
	expectEqual(t, SizeCellCountInt(3), new.data.size.rows)
	expectEqual(t, SizeCellCountInt(10), s.cols)
}

// Zig: test "term51: increaseCapacity: preserves the page dirty flag and row dirty flags"
func TestPl5_term51_increaseCapacity_preserves_the_page_dirty_flag_and_row_dirty_flags(t *testing.T) {
	testStdCapacity(t)

	for _, dirty := range []bool{false, true} {
		func() {
			s := pl5Init(t, 20, 3, None[uint64]())
			defer s.Deinit()
			node := s.pages.first
			s.ClearDirty()
			node.data.dirty = dirty
			node.data.GetRowAndCell(0, 1).row.dirty = true

			new := pl5Increase(t, s, node, Some(PageListIncreaseCapacityStyles))
			expectEqual(t, dirty, new.data.dirty)
			expect(t, !new.data.GetRowAndCell(0, 0).row.dirty, "!new.data.getRowAndCell(0, 0).row.dirty")
			expect(t, new.data.GetRowAndCell(0, 1).row.dirty, "new.data.getRowAndCell(0, 1).row.dirty")
			expect(t, !new.data.GetRowAndCell(0, 2).row.dirty, "!new.data.getRowAndCell(0, 2).row.dirty")
		}()
	}
}

// Zig: test "term51: increaseCapacity: styles double until the 16 bit limit then run out of space"
func TestPl5_term51_increaseCapacity_styles_double_until_the_16_bit_limit_then_run_out_of_space(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 2, 2, Some[uint64](0))
	defer s.Deinit()

	node := s.pages.first
	steps := uint(0)
	for {
		old := node.data.capacity.styles
		new, err := s.IncreaseCapacity(node, Some(PageListIncreaseCapacityStyles))
		if err != nil {
			pl5ExpectError(t, ErrOutOfSpace, err)
			break
		}
		var want uint
		if old > math.MaxUint16/2 {
			want = math.MaxUint16
		} else {
			want = uint(old) * 2
		}
		expectEqual(t, want, uint(new.data.capacity.styles))
		node = new
		steps++
	}

	// 128 -> 256 -> ... -> 32768 -> 65535.
	expectEqual(t, uint(9), steps)
	expectEqual(t, SizeStyleCountInt(math.MaxUint16), node.data.capacity.styles)

	// The failed call changed nothing.
	page_size := s.page_size
	serial := s.page_serial
	_, err := s.IncreaseCapacity(node, Some(PageListIncreaseCapacityStyles))
	pl5ExpectError(t, ErrOutOfSpace, err)
	expectEqual(t, node, s.pages.first)
	expectEqual(t, page_size, s.page_size)
	expectEqual(t, serial, s.page_serial)
	expectEqual(t, uint(1), s.TotalPages())

	// Other dimensions of the maxed page can still grow.
	grown := pl5Increase(t, s, node, Some(PageListIncreaseCapacityGraphemeBytes))
	expectEqual(t, SizeStyleCountInt(math.MaxUint16), grown.data.capacity.styles)
}

// Zig: test "term51: increaseCapacity: hyperlink bytes double until the 16 bit limit then run out of space"
func TestPl5_term51_increaseCapacity_hyperlink_bytes_double_until_the_16_bit_limit_then_run_out_of_space(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 2, 2, Some[uint64](0))
	defer s.Deinit()

	node := s.pages.first
	steps := uint(0)
	for {
		old := node.data.capacity.hyperlink_bytes
		new, err := s.IncreaseCapacity(node, Some(PageListIncreaseCapacityHyperlinkBytes))
		if err != nil {
			pl5ExpectError(t, ErrOutOfSpace, err)
			break
		}
		var want uint
		if old > math.MaxUint16/2 {
			want = math.MaxUint16
		} else {
			want = uint(old) * 2
		}
		expectEqual(t, want, uint(new.data.capacity.hyperlink_bytes))
		node = new
		steps++
	}
	expect(t, steps > 0, "steps > 0")
	expectEqual(t, SizeHyperlinkCountInt(math.MaxUint16), node.data.capacity.hyperlink_bytes)
	expectEqual(t, node, s.pages.first)
}

// Zig: test "term51: increaseCapacity: 32 bit dimensions run out of space before the page size limit"
func TestPl5_term51_increaseCapacity_32_bit_dimensions_run_out_of_space_before_the_page_size_limit(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 2, 2, Some[uint64](0))
	defer s.Deinit()
	node := s.pages.first
	page_size := s.page_size
	serial := s.page_serial
	original := node.data.capacity

	// The capacity fields are changed without reallocating: the checks
	// all happen before any allocation, so nothing is ever allocated.
	for _, kind := range []PageListIncreaseCapacity{PageListIncreaseCapacityGraphemeBytes, PageListIncreaseCapacityStringBytes} {
		values := []uint32{
			// Already the maximum: doubling overflows with no room left.
			math.MaxUint32,
			// Doubling overflows: clamped to the maximum, whose page is
			// larger than the maximum page size.
			0x8000_0000,
			// Doubling does not overflow but the page is too large.
			0x7FFF_FFFF,
		}
		for _, v := range values {
			pl5SetCapField(&node.data.capacity, kind, uint64(v))
			_, err := s.IncreaseCapacity(node, Some(kind))
			pl5ExpectError(t, ErrOutOfSpace, err)
			expectEqual(t, node, s.pages.first)
			expectEqual(t, page_size, s.page_size)
			expectEqual(t, serial, s.page_serial)
		}
		node.data.capacity = original
	}
}

// Zig: test "term51: increaseCapacity: a zero capacity stays zero"
func TestPl5_term51_increaseCapacity_a_zero_capacity_stays_zero(t *testing.T) {
	testStdCapacity(t)
	// Doubling zero is zero: the "at least one unit" promise of the doc
	// comment does not hold for an empty dimension. The capacity field is
	// changed without reallocating to get such a page.
	for _, kind := range []PageListIncreaseCapacity{
		PageListIncreaseCapacityHyperlinkBytes,
		PageListIncreaseCapacityStringBytes,
		PageListIncreaseCapacityGraphemeBytes,
	} {
		func() {
			s := pl5Init(t, 2, 2, Some[uint64](0))
			defer s.Deinit()
			node := s.pages.first
			pl5SetCapField(&node.data.capacity, kind, 0)
			new := pl5Increase(t, s, node, Some(kind))
			expectEqual(t, uint64(0), pl5CapField(new.data.capacity, kind))
			expectEqual(t, new, s.pages.first)
		}()
	}
}

// Zig: test "term51: increaseCapacity: layout new page is non-pooled once it exceeds the standard size"
func TestPl5_term51_increaseCapacity_layout_new_page_is_non_pooled_once_it_exceeds_the_standard_size(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	expectEqual(t, pageListStdSize(), pageListMemoryLen(&s.pages.first.data))
	new := pl5Increase(t, s, s.pages.first, Some(PageListIncreaseCapacityStyles))
	expectEqual(t, PageLayoutOf(new.data.capacity).total_size, pageListMemoryLen(&new.data))
	expect(t, pageListMemoryLen(&new.data) > pageListStdSize(), "new.data.memory.len > std_size")
	expectEqual(t, pageListMemoryLen(&new.data), s.page_size)

	// Back to a pooled page via a compaction.
	compacted := pl5Compact(t, s, new)
	expect(t, compacted != nil, "(try s.compact(new)).? != null")
	expect(t, pageListMemoryLen(&compacted.data) <= pageListStdSize(), "compacted.data.memory.len <= std_size")
}

// Zig: test "term51: createPage: creates an unlinked page and accounts for it"
func TestPl5_term51_createPage_creates_an_unlinked_page_and_accounts_for_it(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	page_size := s.page_size
	serial := s.page_serial
	total := s.total_rows
	cap := pageListInitialCapacity(80)

	node := pl5CreatePage(t, s, cap)
	expect(t, node.prev == nil, "node.prev == null")
	expect(t, node.next == nil, "node.next == null")
	expectEqual(t, uint(1), s.TotalPages())
	expectEqual(t, total, s.total_rows)
	expectEqual(t, cap, node.data.capacity)
	expectEqual(t, SizeCellCountInt(0), node.data.size.rows)
	expectEqual(t, cap.cols, node.data.size.cols)
	expectEqual(t, serial, node.serial)
	expectEqual(t, serial+1, s.page_serial)
	expectEqual(t, page_size+pageListStdSize(), s.page_size)

	// Memory starts out blank.
	expect(t, pl5LRawCell(&node.data, 0, 0).IsEmpty(), "term51LRawCell(&node.data, 0, 0).isEmpty()")
	expect(t, pl5LRawCell(&node.data, 79, uint(cap.rows)-1).IsEmpty(), "term51LRawCell(&node.data, 79, cap.rows - 1).isEmpty()")
	expectEqual(t, uint(0), node.data.styles.Count())

	// A second page takes the next serial.
	node2 := pl5CreatePage(t, s, cap)
	expect(t, node2 != node, "node2 != node")
	expectEqual(t, serial+1, node2.serial)
	expectEqual(t, serial+2, s.page_serial)
	expectEqual(t, page_size+2*pageListStdSize(), s.page_size)

	s.destroyNode(node2)
	s.destroyNode(node)
	expectEqual(t, page_size, s.page_size)
	// Serials are never given back.
	expectEqual(t, serial+2, s.page_serial)
}

// Zig: test "term51: createPage: non-standard capacity gets its own allocation"
func TestPl5_term51_createPage_non_standard_capacity_gets_its_own_allocation(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	page_size := s.page_size

	cap := pageListInitialCapacity(80)
	cap.styles = 1024
	layout := PageLayoutOf(cap)
	expect(t, layout.total_size > pageListStdSize(), "layout.total_size > std_size")

	node := pl5CreatePage(t, s, cap)
	expectEqual(t, layout.total_size, pageListMemoryLen(&node.data))
	expectEqual(t, cap, node.data.capacity)
	expectEqual(t, page_size+layout.total_size, s.page_size)
	expect(t, pl5LRawCell(&node.data, 0, 0).IsEmpty(), "term51LRawCell(&node.data, 0, 0).isEmpty()")

	s.destroyNode(node)
	expectEqual(t, page_size, s.page_size)
}

// Zig: test "term51: createPage: layout a small pooled page accounts a whole pool item"
func TestPl5_term51_createPage_layout_a_small_pooled_page_accounts_a_whole_pool_item(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	page_size := s.page_size

	// A capacity far below the standard size still takes a pool item.
	cap := PageCapacityInit(10, 10)
	node := pl5CreatePage(t, s, cap)
	expect(t, pageListMemoryLen(&node.data) < pageListStdSize(), "node.data.memory.len < std_size")
	expectEqual(t, PageLayoutOf(cap).total_size, pageListMemoryLen(&node.data))
	expectEqual(t, page_size+pageListStdSize(), s.page_size)

	// Destroying subtracts the (smaller) memory length, so the accounting
	// drifts by the difference.
	len_ := pageListMemoryLen(&node.data)
	s.destroyNode(node)
	expectEqual(t, page_size+pageListStdSize()-len_, s.page_size)
	s.page_size = page_size
}

// Zig: test "term51: createPageExt: with and without size accounting"
func TestPl5_term51_createPageExt_with_and_without_size_accounting(t *testing.T) {
	testStdCapacity(t)
	pool, err := PageListMemoryPoolInit(2)
	zigTry(t, err)
	defer pool.Deinit()
	cap := pageListInitialCapacity(80)

	// With a counter.
	serial := uint64(7)
	total := uint64(100)
	a, err := pageListCreatePageExt(&pool, cap, &serial, &total)
	zigTry(t, err)
	expectEqual(t, uint64(7), a.serial)
	expectEqual(t, uint64(8), serial)
	expectEqual(t, 100+pageListStdSize(), total)
	expectEqual(t, SizeCellCountInt(0), a.data.size.rows)
	expectEqual(t, cap, a.data.capacity)
	expect(t, a.prev == nil, "a.prev == null")
	expect(t, a.next == nil, "a.next == null")

	// Without one.
	b, err := pageListCreatePageExt(&pool, cap, &serial, nil)
	zigTry(t, err)
	expectEqual(t, uint64(8), b.serial)
	expectEqual(t, uint64(9), serial)
	expectEqual(t, 100+pageListStdSize(), total)
	expect(t, a != b, "a != b")
	// Zig-only: try testing.expect(a.data.memory.ptr != b.data.memory.ptr);

	pageListDestroyNodeExt(&pool, b, nil)
	pageListDestroyNodeExt(&pool, a, &total)
	expectEqual(t, uint64(100), total)
}

// Zig: test "term51: createPageExt: non-pooled pages come from the page allocator"
func TestPl5_term51_createPageExt_non_pooled_pages_come_from_the_page_allocator(t *testing.T) {
	testStdCapacity(t)
	pool, err := PageListMemoryPoolInit(1)
	zigTry(t, err)
	defer pool.Deinit()
	cap := pageListInitialCapacity(80)
	cap.styles = 1024
	layout := PageLayoutOf(cap)
	expect(t, layout.total_size > pageListStdSize(), "layout.total_size > std_size")

	serial := uint64(0)
	total := uint64(0)
	node, err := pageListCreatePageExt(&pool, cap, &serial, &total)
	zigTry(t, err)
	expectEqual(t, layout.total_size, pageListMemoryLen(&node.data))
	expectEqual(t, layout.total_size, total)
	expectEqual(t, uint64(1), serial)
	pageListDestroyNodeExt(&pool, node, &total)
	expectEqual(t, uint64(0), total)
}

// Zig: test "term51: destroyNode: subtracts the page size and leaves rows alone"
func TestPl5_term51_destroyNode_subtracts_the_page_size_and_leaves_rows_alone(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	page_size := s.page_size
	total := s.total_rows

	// A pooled page that was given rows.
	node := pl5CreatePage(t, s, pageListInitialCapacity(80))
	node.data.size.rows = 10
	expectEqual(t, page_size+pageListStdSize(), s.page_size)
	s.destroyNode(node)
	expectEqual(t, page_size, s.page_size)
	expectEqual(t, total, s.total_rows)

	// A non-pooled one.
	cap := pageListInitialCapacity(80)
	cap.styles = 1024
	big := pl5CreatePage(t, s, cap)
	len_ := pageListMemoryLen(&big.data)
	expectEqual(t, page_size+len_, s.page_size)
	s.destroyNode(big)
	expectEqual(t, page_size, s.page_size)
	expectEqual(t, uint(1), s.TotalPages())
}

// Zig: test "term51: destroyNode: removing a page and its rows is up to the caller"
func TestPl5_term51_destroyNode_removing_a_page_and_its_rows_is_up_to_the_caller(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	defer s.Deinit()
	pl5LFillLast(t, s)
	extra := pl5GrowNode(t, s)
	total := s.total_rows
	page_size := s.page_size

	// Remove the last page from the list by hand, then destroy it.
	s.pages.Remove(extra)
	s.destroyNode(extra)
	expectEqual(t, page_size-pageListStdSize(), s.page_size)
	// total_rows still counts the destroyed page's row until fixed up.
	expectEqual(t, total, s.total_rows)
	s.total_rows -= 1
}

// Zig: test "term51: destroyNodeExt: optional size accounting"
func TestPl5_term51_destroyNodeExt_optional_size_accounting(t *testing.T) {
	testStdCapacity(t)
	pool, err := PageListMemoryPoolInit(2)
	zigTry(t, err)
	defer pool.Deinit()
	cap := pageListInitialCapacity(80)
	big := cap
	big.styles = 1024
	serial := uint64(0)

	// Pooled, without and with a counter.
	{
		node, err := pageListCreatePageExt(&pool, cap, &serial, nil)
		zigTry(t, err)
		pageListDestroyNodeExt(&pool, node, nil)
	}
	{
		total := uint64(0)
		node, err := pageListCreatePageExt(&pool, cap, &serial, &total)
		zigTry(t, err)
		expectEqual(t, pageListStdSize(), total)
		pageListDestroyNodeExt(&pool, node, &total)
		expectEqual(t, uint64(0), total)
	}

	// Non-pooled: the memory is returned to the allocator (a leak would
	// fail the test).
	{
		node, err := pageListCreatePageExt(&pool, big, &serial, nil)
		zigTry(t, err)
		pageListDestroyNodeExt(&pool, node, nil)
	}
	{
		total := uint64(0)
		node, err := pageListCreatePageExt(&pool, big, &serial, &total)
		zigTry(t, err)
		expect(t, total > pageListStdSize(), "total > std_size")
		pageListDestroyNodeExt(&pool, node, &total)
		expectEqual(t, uint64(0), total)
	}
}

// ---------------------------------------------------------------------------
// term51: additional tests (M).
// ---------------------------------------------------------------------------

// pl5MPos is term51MPos: location of a pin as (page index in the list, row in
// page, column).
type pl5MPos struct {
	page uint
	y    uint
	x    uint
}

func pl5MpageIndex(s *PageList, node *PageListNode) uint {
	i := uint(0)
	for it := s.pages.first; it != nil; it = it.next {
		if it == node {
			return i
		}
		i++
	}
	panic("term51M: node not in list")
}

func pl5MnodeAt(s *PageList, idx uint) *PageListNode {
	i := uint(0)
	for it := s.pages.first; it != nil; it = it.next {
		if i == idx {
			return it
		}
		i++
	}
	panic("term51M: page index out of range")
}

func pl5MpinPos(s *PageList, p *PageListPin) pl5MPos {
	return pl5MPos{
		page: pl5MpageIndex(s, p.Node),
		y:    uint(p.Y),
		x:    uint(p.X),
	}
}

func pl5MexpectPinAt(t *testing.T, s *PageList, p *PageListPin, page uint, y uint, x uint) {
	t.Helper()
	expectEqual(t, pl5MPos{page: page, y: y, x: x}, pl5MpinPos(s, p))
}

// pl5MlocOf is term51MlocOf: location (page, row in page) of a global row
// index given page sizes.
func pl5MlocOf(sizes []uint, g uint) (page uint, y uint) {
	rem := g
	for i, n := range sizes {
		if rem < n {
			return uint(i), rem
		}
		rem -= n
	}
	panic("term51M: global row out of range")
}

// pl5MglobalOf is term51MglobalOf: global row index of a pin.
func pl5MglobalOf(s *PageList, p *PageListPin) uint {
	g := uint(0)
	for it := s.pages.first; it != nil; it = it.next {
		if it == p.Node {
			return g + uint(p.Y)
		}
		g += uint(it.data.size.rows)
	}
	panic("term51M: node not in list")
}

func pl5MexpectPinGlobal(t *testing.T, s *PageList, sizes []uint, p *PageListPin, g uint, x uint) {
	t.Helper()
	page, y := pl5MlocOf(sizes, g)
	expectEqual(t, pl5MPos{page: page, y: y, x: x}, pl5MpinPos(s, p))
}

// pl5Mbuild is term51Mbuild: build a PageList whose pages have exactly the
// given row counts (the pages are created by splitting one page, so all share
// a capacity). The active area is `rows` tall. Every cell of global row i
// holds the codepoint i + 1. All dirty bits are cleared.
func pl5Mbuild(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt, sizes []uint) *PageList {
	t.Helper()
	total := uint(0)
	for _, n := range sizes {
		total += n
	}
	s := pl5Init(t, cols, rows, None[uint64]())
	zigTry(t, s.growRows(total-uint(rows)))
	expectEqual(t, uint(1), s.TotalPages())

	cum := uint(0)
	for _, n := range sizes[:len(sizes)-1] {
		cum += n
		zigTry(t, s.Split(pl5PinOf(t, s, pl5Screen(0, uint32(cum)))))
	}

	label := uint32(1)
	for node := s.pages.first; node != nil; node = node.next {
		for y := uint(0); y < uint(node.data.size.rows); y++ {
			row := node.data.GetRow(y)
			cells := node.data.GetCells(row)
			for i := range cells {
				cells[i] = PageCell{
					content_tag: PageCellContentTagCodepoint,
					content:     PageCellContent{codepoint: label},
				}
			}
			label++
		}
	}
	s.ClearDirty()
	return s
}

// pl5Mlabels is term51Mlabels: dump the label (codepoint of column 0) of
// every row, checking that all cells of a row agree. len(out) must equal the
// number of rows.
func pl5Mlabels(t *testing.T, s *PageList, out []uint32) {
	t.Helper()
	i := uint(0)
	for node := s.pages.first; node != nil; node = node.next {
		for y := uint(0); y < uint(node.data.size.rows); y++ {
			row := node.data.GetRow(y)
			cells := node.data.GetCells(row)
			for _, c := range cells {
				expectEqual(t, cells[0].Codepoint(), c.Codepoint())
			}
			if i < uint(len(out)) {
				out[i] = cells[0].Codepoint()
			}
			i++
		}
	}
	expectEqual(t, uint(len(out)), i)
}

func pl5MexpectLabels(t *testing.T, s *PageList, expected []uint32) {
	t.Helper()
	buf := make([]uint32, len(expected))
	pl5Mlabels(t, s, buf)
	expectEqualSlices(t, expected, buf)
}

func pl5MexpectSizes(t *testing.T, s *PageList, expected []uint) {
	t.Helper()
	var buf []uint
	for node := s.pages.first; node != nil; node = node.next {
		buf = append(buf, uint(node.data.size.rows))
	}
	expectEqualSlices(t, expected, buf)
}

// pl5MexpectPageDirty is term51MexpectPageDirty: expect page-level dirty
// flags (not row flags) per page.
func pl5MexpectPageDirty(t *testing.T, s *PageList, expected []bool) {
	t.Helper()
	var buf []bool
	for node := s.pages.first; node != nil; node = node.next {
		buf = append(buf, node.data.dirty)
	}
	expectEqualSlices(t, expected, buf)
}

func pl5MtrackAt(t *testing.T, s *PageList, g uint, x uint) *PageListPin {
	t.Helper()
	return pl5Track(t, s, pl5PinOf(t, s, pl5Screen(SizeCellCountInt(x), uint32(g))))
}

// pl5MeraseRowCheck is term51MeraseRowCheck: exhaustive check of eraseRow
// against a simple model. Pins sit on every row at x = 3; optionally the
// viewport is a pin at row `vq` (x = 2).
func pl5MeraseRowCheck(t *testing.T, sizes []uint, g uint, vq Optional[uint], cache bool) {
	t.Helper()
	const active_rows SizeCellCountInt = 4
	s := pl5Mbuild(t, 6, active_rows, sizes)
	defer s.Deinit()
	total := s.total_rows
	active_start := total - uint(active_rows)

	old := make([]uint32, total)
	pl5Mlabels(t, s, old)

	pins := make([]*PageListPin, total)
	for q := uint(0); q < total; q++ {
		pins[q] = pl5MtrackAt(t, s, q, 3)
	}
	if vq.Valid {
		q := vq.Value
		*s.viewport_pin = pl5PinOf(t, s, pl5Screen(2, uint32(q)))
		s.viewport = PageListViewport{Tag: PageListViewportPin}
		if cache {
			s.viewport_pin_row_offset = Some(q)
		} else {
			s.viewport_pin_row_offset = None[uint]()
		}
	}

	zigTry(t, s.EraseRow(pl5Screen(0, uint32(g))))

	// Sizes and totals never change.
	pl5MexpectSizes(t, s, sizes)
	expectEqual(t, total, s.total_rows)

	// Rows move up, the last row is blank.
	expected := make([]uint32, total)
	for i := uint(0); i < total; i++ {
		if i < g {
			expected[i] = old[i]
		} else if i+1 < total {
			expected[i] = old[i+1]
		} else {
			expected[i] = 0
		}
	}
	pl5MexpectLabels(t, s, expected)

	// Pins after the erased row move up one global row; the rest stay.
	for q := uint(0); q < total; q++ {
		q2 := q
		if q > g {
			q2 = q - 1
		}
		pl5MexpectPinGlobal(t, s, sizes, pins[q], q2, 3)
		expect(t, !pins[q].Garbage, "!pins[q].garbage")
	}

	if vq.Valid {
		q := vq.Value
		q2 := q
		if q > g {
			q2 = q - 1
		}
		pl5MexpectPinGlobal(t, s, sizes, s.viewport_pin, q2, 2)

		// fixupViewport runs after only the FIRST page's pins were shifted,
		// so a viewport pin on a later page is judged at its old row.
		qpage, _ := pl5MlocOf(sizes, q)
		gpage, _ := pl5MlocOf(sizes, g)
		same_page := qpage == gpage
		q_judged := q
		if same_page {
			q_judged = q2
		}
		if q_judged >= active_start {
			expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
		} else {
			expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
			if cache {
				expectEqual(t, Some(q2), s.viewport_pin_row_offset)
			}
			if !cache {
				expectEqual(t, None[uint](), s.viewport_pin_row_offset)
			}
		}
	}

	// Every page from the erased row's page to the end is dirty.
	first, _ := pl5MlocOf(sizes, g)
	i := uint(0)
	for n := s.pages.first; n != nil; n = n.next {
		expectEqual(t, i >= first, n.data.dirty)
		i++
	}
}

var pl5MLayouts = [][]uint{
	{9},
	{4, 2, 3},
	{1, 1, 1, 1, 1, 1, 1, 1, 1},
	{2, 5, 1, 1},
}

// Zig: test "term51: eraseRow: oracle over page layouts, every row, pins and viewport pin"
func TestPl5_term51_eraseRow_oracle_over_page_layouts_every_row_pins_and_viewport_pin(t *testing.T) {
	testStdCapacity(t)
	for _, sizes := range pl5MLayouts {
		for g := uint(0); g < 9; g++ {
			pl5MeraseRowCheck(t, sizes, g, None[uint](), false)
			for vq := uint(0); vq < 6; vq++ {
				// Cache not computed: any viewport pin position is fine.
				pl5MeraseRowCheck(t, sizes, g, Some(vq), false)
				// Cache computed: only valid when the viewport pin is below
				// the erased row (see the stale-cache test below).
				if vq > g {
					pl5MeraseRowCheck(t, sizes, g, Some(vq), true)
				}
			}
		}
	}
}

// Zig: test "term51: eraseRow: single page middle row, first row, last row"
func TestPl5_term51_eraseRow_single_page_middle_row_first_row_last_row(t *testing.T) {
	testStdCapacity(t)

	// Middle row.
	func() {
		s := pl5Mbuild(t, 5, 3, []uint{6})
		defer s.Deinit()
		zigTry(t, s.EraseRow(pl5Screen(0, 2)))
		pl5MexpectLabels(t, s, []uint32{1, 2, 4, 5, 6, 0})
		pl5MexpectSizes(t, s, []uint{6})
		expectEqual(t, uint(6), s.total_rows)
		pl5MexpectPageDirty(t, s, []bool{true})
	}()

	// First row.
	func() {
		s := pl5Mbuild(t, 5, 3, []uint{6})
		defer s.Deinit()
		zigTry(t, s.EraseRow(pl5Screen(0, 0)))
		pl5MexpectLabels(t, s, []uint32{2, 3, 4, 5, 6, 0})
	}()

	// Last row: only clears it.
	func() {
		s := pl5Mbuild(t, 5, 3, []uint{6})
		defer s.Deinit()
		zigTry(t, s.EraseRow(pl5Screen(0, 5)))
		pl5MexpectLabels(t, s, []uint32{1, 2, 3, 4, 5, 0})
		pl5MexpectPageDirty(t, s, []bool{true})
	}()

	// One-row list.
	func() {
		s := pl5Mbuild(t, 5, 1, []uint{1})
		defer s.Deinit()
		pl5MexpectLabels(t, s, []uint32{1})
		zigTry(t, s.EraseRow(pl5Screen(0, 0)))
		pl5MexpectLabels(t, s, []uint32{0})
		expectEqual(t, uint(1), s.total_rows)
	}()
}

// Zig: test "term51: eraseRow: three pages hand-computed rows, pins, dirty flags"
func TestPl5_term51_eraseRow_three_pages_hand_computed_rows_pins_dirty_flags(t *testing.T) {
	testStdCapacity(t)
	sizes := []uint{4, 3, 5}

	s := pl5Mbuild(t, 6, 5, sizes)
	defer s.Deinit()

	// Pins: p0 above the erased row, pe on it, pa after it (same page),
	// then pins at the top and bottom of the following pages.
	p0 := pl5MtrackAt(t, s, 0, 1)
	pe := pl5MtrackAt(t, s, 1, 2)
	pa := pl5MtrackAt(t, s, 3, 4)
	p1top := pl5MtrackAt(t, s, 4, 5)
	p1mid := pl5MtrackAt(t, s, 5, 0)
	p2top := pl5MtrackAt(t, s, 7, 3)
	p2end := pl5MtrackAt(t, s, 11, 1)

	zigTry(t, s.EraseRow(pl5Screen(0, 1)))

	pl5MexpectLabels(t, s, []uint32{1, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 0})
	pl5MexpectSizes(t, s, sizes)

	pl5MexpectPinAt(t, s, p0, 0, 0, 1)
	pl5MexpectPinAt(t, s, pe, 0, 1, 2) // stays on the (now different) row
	pl5MexpectPinAt(t, s, pa, 0, 2, 4)
	pl5MexpectPinAt(t, s, p1top, 0, 3, 5) // crosses to previous page
	pl5MexpectPinAt(t, s, p1mid, 1, 0, 0)
	pl5MexpectPinAt(t, s, p2top, 1, 2, 3) // crosses to previous page
	pl5MexpectPinAt(t, s, p2end, 2, 3, 1)
	pl5MexpectPageDirty(t, s, []bool{true, true, true})

	// Erasing in the middle page leaves page 0 untouched.
	s.ClearDirty()
	zigTry(t, s.EraseRow(pl5Screen(0, 4)))
	pl5MexpectLabels(t, s, []uint32{1, 3, 4, 5, 7, 8, 9, 10, 11, 12, 0, 0})
	pl5MexpectPageDirty(t, s, []bool{false, true, true})
	pl5MexpectPinAt(t, s, pa, 0, 2, 4)
	pl5MexpectPinAt(t, s, p1top, 0, 3, 5) // page 0 untouched
	pl5MexpectPinAt(t, s, p1mid, 1, 0, 0) // on the erased row: stays
	pl5MexpectPinAt(t, s, p2top, 1, 1, 3) // below it in the same page
	pl5MexpectPinAt(t, s, p2end, 2, 2, 1)
}

// Zig: test "term51: eraseRow: every point tag selects the row"
func TestPl5_term51_eraseRow_every_point_tag_selects_the_row(t *testing.T) {
	testStdCapacity(t)
	sizes := []uint{4, 3, 5}

	// history y=2 is global row 2
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		zigTry(t, s.EraseRow(pl5History(0, 2)))
		pl5MexpectLabels(t, s, []uint32{1, 2, 4, 5, 6, 7, 8, 9, 10, 11, 12, 0})
	}()
	// active y=1 is global row 8 (the active area starts at row 7)
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		zigTry(t, s.EraseRow(pl5Active(0, 1)))
		pl5MexpectLabels(t, s, []uint32{1, 2, 3, 4, 5, 6, 7, 8, 10, 11, 12, 0})
	}()
	// viewport with an active viewport: same as active
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		zigTry(t, s.EraseRow(pl5Viewport(0, 0)))
		pl5MexpectLabels(t, s, []uint32{1, 2, 3, 4, 5, 6, 7, 9, 10, 11, 12, 0})
	}()
	// viewport at top: same as screen
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		s.viewport = PageListViewport{Tag: PageListViewportTop}
		zigTry(t, s.EraseRow(pl5Viewport(0, 5)))
		pl5MexpectLabels(t, s, []uint32{1, 2, 3, 4, 5, 7, 8, 9, 10, 11, 12, 0})
		expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
	}()
	// viewport pinned at row 2: viewport y=2 is global row 4
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		*s.viewport_pin = pl5PinOf(t, s, pl5Screen(0, 2))
		s.viewport = PageListViewport{Tag: PageListViewportPin}
		zigTry(t, s.EraseRow(pl5Viewport(0, 2)))
		pl5MexpectLabels(t, s, []uint32{1, 2, 3, 4, 6, 7, 8, 9, 10, 11, 12, 0})
	}()
	// screen y=11 (last row)
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		zigTry(t, s.EraseRow(pl5Screen(0, 11)))
		pl5MexpectLabels(t, s, []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 0})
	}()
}

// Zig: test "term51: eraseRow: viewport top becomes active when the whole list is active"
func TestPl5_term51_eraseRow_viewport_top_becomes_active_when_the_whole_list_is_active(t *testing.T) {
	testStdCapacity(t)

	// No history: the first page is in the active area.
	func() {
		s := pl5Mbuild(t, 6, 5, []uint{5})
		defer s.Deinit()
		s.viewport = PageListViewport{Tag: PageListViewportTop}
		zigTry(t, s.EraseRow(pl5Screen(0, 0)))
		expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	}()
	// With history the top viewport is unchanged.
	func() {
		s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
		defer s.Deinit()
		s.viewport = PageListViewport{Tag: PageListViewportTop}
		zigTry(t, s.EraseRow(pl5Screen(0, 0)))
		expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
	}()
	// Active viewport stays active.
	func() {
		s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
		defer s.Deinit()
		zigTry(t, s.EraseRow(pl5Screen(0, 0)))
		expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	}()
}

// Zig: test "term51: eraseRow: viewport pin cache is decremented even when the erased row is below the pin"
func TestPl5_term51_eraseRow_viewport_pin_cache_is_decremented_even_when_the_erased_row_is_below_the_pin(t *testing.T) {
	testStdCapacity(t)

	// The cached row offset is decremented unconditionally by
	// fixupViewport(1), even if the erased row is after the viewport pin.
	// That leaves the cache stale; upstream's integrity check would panic,
	// so pause it to observe the state.
	func() {
		s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
		defer s.Deinit()
		*s.viewport_pin = pl5PinOf(t, s, pl5Screen(0, 2))
		s.viewport = PageListViewport{Tag: PageListViewportPin}
		s.viewport_pin_row_offset = Some[uint](2)

		s.PauseIntegrityChecks(true)
		zigTry(t, s.EraseRow(pl5Screen(0, 3)))
		s.PauseIntegrityChecks(false)

		expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
		expectEqual(t, uint(2), pl5MglobalOf(s, s.viewport_pin))
		expectEqual(t, Some[uint](1), s.viewport_pin_row_offset)

		// deinit verifies integrity: drop the stale cache.
		s.viewport_pin_row_offset = None[uint]()
	}()

	// A pin at row 0 with a cached offset of 0: the offset would go below
	// zero so the viewport becomes `top` instead.
	func() {
		s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
		defer s.Deinit()
		*s.viewport_pin = pl5PinOf(t, s, pl5Screen(0, 0))
		s.viewport = PageListViewport{Tag: PageListViewportPin}
		s.viewport_pin_row_offset = Some[uint](0)
		zigTry(t, s.EraseRow(pl5Screen(0, 2)))
		expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
	}()

	// Without a cached offset nothing is touched.
	func() {
		s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
		defer s.Deinit()
		*s.viewport_pin = pl5PinOf(t, s, pl5Screen(0, 2))
		s.viewport = PageListViewport{Tag: PageListViewportPin}
		zigTry(t, s.EraseRow(pl5Screen(0, 3)))
		expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
		expectEqual(t, None[uint](), s.viewport_pin_row_offset)
	}()
}

// Zig: test "term51: eraseRow: row metadata moves with the row and managed memory is freed"
func TestPl5_term51_eraseRow_row_metadata_moves_with_the_row_and_managed_memory_is_freed(t *testing.T) {
	testStdCapacity(t)

	s := pl5Mbuild(t, 5, 2, []uint{2, 2})
	defer s.Deinit()
	p0 := &s.pages.first.data
	p1 := &s.pages.first.next.data

	// Row 0 (to be erased) has a style and a grapheme.
	{
		rac := p0.GetRowAndCell(0, 0)
		id, err := p0.styles.Add(Style{Flags: StyleFlags{Bold: true}})
		zigTry(t, err)
		rac.row.styled = true
		rac.cell.style_id = id
		rac2 := p0.GetRowAndCell(1, 0)
		zigTry(t, p0.AppendGrapheme(rac2.row, rac2.cell, 0x301))
	}
	expectEqual(t, uint(1), p0.styles.Count())
	expectEqual(t, uint(1), p0.GraphemeCount())

	// Row metadata: row 1 (page 0) wraps, row 2 (page 1) is a wrap
	// continuation.
	p0.GetRow(1).wrap = true
	p1.GetRow(0).wrap_continuation = true
	// The row at the bottom of the list has its own flags.
	p1.GetRow(1).wrap = true

	zigTry(t, s.EraseRow(pl5Screen(0, 0)))

	// The styled/grapheme row was erased; its memory was released.
	expectEqual(t, uint(0), p0.styles.Count())
	expectEqual(t, uint(0), p0.GraphemeCount())
	pl5MexpectLabels(t, s, []uint32{2, 3, 4, 0})

	// Flags moved with their rows. Old row 1 (wrap) is now row 0; old row 2
	// (wrap_continuation) is now row 1; old row 3 (wrap) is now row 2.
	expect(t, p0.GetRow(0).wrap, "p0.getRow(0).wrap")
	expect(t, !p0.GetRow(0).wrap_continuation, "!p0.getRow(0).wrap_continuation")
	expect(t, !p0.GetRow(1).wrap, "!p0.getRow(1).wrap")
	expect(t, p0.GetRow(1).wrap_continuation, "p0.getRow(1).wrap_continuation")
	expect(t, p1.GetRow(0).wrap, "p1.getRow(0).wrap")
	expect(t, !p1.GetRow(0).wrap_continuation, "!p1.getRow(0).wrap_continuation")
	// The final row's cells were cleared but its row flags are NOT reset:
	// it is the row struct that was cloned from (old row 2), so it keeps
	// wrap_continuation.
	expect(t, !p1.GetRow(1).wrap, "!p1.getRow(1).wrap")
	expect(t, p1.GetRow(1).wrap_continuation, "p1.getRow(1).wrap_continuation")
	expect(t, !p1.GetRow(1).styled, "!p1.getRow(1).styled")
	expect(t, !p1.GetRow(1).grapheme, "!p1.getRow(1).grapheme")
}

// pl5MfillStyles is the loop shared by the two clone failure tests: fill
// every style slot of page 0 on row 0, one cell per style.
func pl5MfillStyles(t *testing.T, p0 *Page) uint {
	t.Helper()
	n := uint(0)
	for ; ; n++ {
		id, err := p0.styles.Add(Style{FgColor: StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{
			R: uint8(n & 0xFF),
			G: uint8(n >> 8),
			B: 7,
		}}})
		if err != nil {
			break
		}
		rac := p0.GetRowAndCell(n, 0)
		rac.row.styled = true
		rac.cell.style_id = id
	}
	return n
}

// Zig: test "term51: eraseRow: clone failure across a page boundary returns the page's error"
func TestPl5_term51_eraseRow_clone_failure_across_a_page_boundary_returns_the_page_s_error(t *testing.T) {
	testStdCapacity(t)

	// Default 215 col pages have 128 style slots.
	s := pl5Mbuild(t, 215, 2, []uint{2, 2})
	defer s.Deinit()
	p0 := &s.pages.first.data
	p1 := &s.pages.first.next.data

	// Fill every style slot of page 0 on row 0, one cell per style.
	n := pl5MfillStyles(t, p0)
	expect(t, n > 0, "n > 0")

	// Page 1 row 0 uses a new style.
	{
		id, err := p1.styles.Add(Style{Flags: StyleFlags{Bold: true}})
		zigTry(t, err)
		rac := p1.GetRowAndCell(0, 0)
		rac.row.styled = true
		rac.cell.style_id = id
	}

	// Erasing the last row of page 0 must pull page 1's first row into it,
	// which needs a new style in a full page.
	pl5ExpectError(t, ErrStyleSetOutOfMemory, s.EraseRow(pl5Screen(0, 1)))
}

// pl5MboundedMove is term51MboundedMove: where a pin at global row `q` (page
// row `qy`) ends up after eraseRowBounded(g, limit): rows g..g+limit move up
// by one; a pin on the erased row of a page's row 0 stays but loses its x.
func pl5MboundedMove(q uint, qy uint, g uint, limit uint) (uint, bool) {
	if q < g || q > g+limit {
		return q, false
	}
	if q == g && qy == 0 {
		return q, true
	}
	return q - 1, false
}

// pl5MeraseRowBoundedCheck is term51MeraseRowBoundedCheck: exhaustive check
// of eraseRowBounded against a model. Pins sit on every row at x = 3;
// optionally the viewport is a pin at row `vq` (x = 2) with a computed row
// offset cache.
func pl5MeraseRowBoundedCheck(t *testing.T, sizes []uint, g uint, limit uint, vq Optional[uint]) {
	t.Helper()
	const active_rows SizeCellCountInt = 4
	s := pl5Mbuild(t, 6, active_rows, sizes)
	defer s.Deinit()
	total := s.total_rows

	old := make([]uint32, total)
	pl5Mlabels(t, s, old)

	pins := make([]*PageListPin, total)
	for q := uint(0); q < total; q++ {
		pins[q] = pl5MtrackAt(t, s, q, 3)
	}
	if vq.Valid {
		q := vq.Value
		*s.viewport_pin = pl5PinOf(t, s, pl5Screen(2, uint32(q)))
		s.viewport = PageListViewport{Tag: PageListViewportPin}
		s.viewport_pin_row_offset = Some(q)
	}

	zigTry(t, s.EraseRowBounded(pl5Screen(0, uint32(g)), limit))

	pl5MexpectSizes(t, s, sizes)
	expectEqual(t, total, s.total_rows)

	// Rows g..end shift up, row `end` becomes blank.
	end := g + limit
	if total-1 < end {
		end = total - 1
	}
	expected := make([]uint32, total)
	for i := uint(0); i < total; i++ {
		if i < g || i > end {
			expected[i] = old[i]
		} else if i < end {
			expected[i] = old[i+1]
		} else {
			expected[i] = 0
		}
	}
	pl5MexpectLabels(t, s, expected)

	for q := uint(0); q < total; q++ {
		_, qy := pl5MlocOf(sizes, q)
		mg, x0 := pl5MboundedMove(q, qy, g, limit)
		x := uint(3)
		if x0 {
			x = 0
		}
		pl5MexpectPinGlobal(t, s, sizes, pins[q], mg, x)
	}

	if vq.Valid {
		q := vq.Value
		_, qy := pl5MlocOf(sizes, q)
		mg, x0 := pl5MboundedMove(q, qy, g, limit)
		x := uint(2)
		if x0 {
			x = 0
		}
		pl5MexpectPinGlobal(t, s, sizes, s.viewport_pin, mg, x)
		expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
		expectEqual(t, Some(mg), s.viewport_pin_row_offset)
	}

	// Pages from the erased row's page to the page holding `end` are dirty.
	first, _ := pl5MlocOf(sizes, g)
	last, _ := pl5MlocOf(sizes, end)
	i := uint(0)
	for n := s.pages.first; n != nil; n = n.next {
		expectEqual(t, i >= first && i <= last, n.data.dirty)
		i++
	}
}

// Zig: test "term51: eraseRowBounded: oracle over page layouts, rows, limits, pins and viewport pin"
func TestPl5_term51_eraseRowBounded_oracle_over_page_layouts_rows_limits_pins_and_viewport_pin(t *testing.T) {
	testStdCapacity(t)
	for _, sizes := range pl5MLayouts {
		for g := uint(0); g < 9; g++ {
			for limit := uint(0); limit < 13; limit++ {
				pl5MeraseRowBoundedCheck(t, sizes, g, limit, None[uint]())
				for vq := uint(0); vq < 6; vq++ {
					pl5MeraseRowBoundedCheck(t, sizes, g, limit, Some(vq))
				}
			}
		}
	}
}

// Zig: test "term51: eraseRowBounded: limit within the first page (hand computed)"
func TestPl5_term51_eraseRowBounded_limit_within_the_first_page_hand_computed(t *testing.T) {
	testStdCapacity(t)
	sizes := []uint{4, 3, 5}

	// Rows 1.. of a 4 row page: limit 0, 1, 2 stay in the page.
	cases := []struct {
		limit    uint
		expected [12]uint32
	}{
		{0, [12]uint32{1, 0, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}},
		{1, [12]uint32{1, 3, 0, 4, 5, 6, 7, 8, 9, 10, 11, 12}},
		{2, [12]uint32{1, 3, 4, 0, 5, 6, 7, 8, 9, 10, 11, 12}},
		// limit 3 is exactly the rest of the page plus nothing: it takes the
		// slow path and also blanks the first row of the next page.
		{3, [12]uint32{1, 3, 4, 5, 0, 6, 7, 8, 9, 10, 11, 12}},
		{4, [12]uint32{1, 3, 4, 5, 6, 0, 7, 8, 9, 10, 11, 12}},
		{5, [12]uint32{1, 3, 4, 5, 6, 7, 0, 8, 9, 10, 11, 12}},
		// limit 6: the second page is shifted completely.
		{6, [12]uint32{1, 3, 4, 5, 6, 7, 8, 0, 9, 10, 11, 12}},
		{7, [12]uint32{1, 3, 4, 5, 6, 7, 8, 9, 0, 10, 11, 12}},
		{10, [12]uint32{1, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 0}},
		// Past the end of the list.
		{11, [12]uint32{1, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 0}},
		{1000, [12]uint32{1, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 0}},
	}
	for _, c := range cases {
		func() {
			s := pl5Mbuild(t, 6, 5, sizes)
			defer s.Deinit()
			zigTry(t, s.EraseRowBounded(pl5Screen(0, 1), c.limit))
			pl5MexpectLabels(t, s, c.expected[:])
			pl5MexpectSizes(t, s, sizes)
			expectEqual(t, uint(12), s.total_rows)
		}()
	}
}

// Zig: test "term51: eraseRowBounded: pins in the erased range move up, the erased row's pin included"
func TestPl5_term51_eraseRowBounded_pins_in_the_erased_range_move_up_the_erased_row_s_pin_included(t *testing.T) {
	testStdCapacity(t)

	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	defer s.Deinit()

	above := pl5MtrackAt(t, s, 0, 1)
	on_row := pl5MtrackAt(t, s, 1, 2)
	inside := pl5MtrackAt(t, s, 2, 3)
	last_in := pl5MtrackAt(t, s, 3, 4)
	below := pl5MtrackAt(t, s, 4, 5)

	// limit 2 covers rows 1..3 (the erased row and two following rows).
	zigTry(t, s.EraseRowBounded(pl5Screen(0, 1), 2))

	pl5MexpectPinAt(t, s, above, 0, 0, 1)
	// Upstream: the pin ON the erased row moves up to the previous row
	// rather than staying on the row that took its place.
	pl5MexpectPinAt(t, s, on_row, 0, 0, 2)
	pl5MexpectPinAt(t, s, inside, 0, 1, 3)
	pl5MexpectPinAt(t, s, last_in, 0, 2, 4)
	pl5MexpectPinAt(t, s, below, 1, 0, 5)
}

// Zig: test "term51: eraseRowBounded: pin on page row 0 of the erased row stays and loses x"
func TestPl5_term51_eraseRowBounded_pin_on_page_row_0_of_the_erased_row_stays_and_loses_x(t *testing.T) {
	testStdCapacity(t)

	// Erasing the top row of the SECOND page: a pin there does not move
	// to the previous page; its x is reset instead.
	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	defer s.Deinit()

	on_row := pl5MtrackAt(t, s, 4, 5)
	below := pl5MtrackAt(t, s, 5, 4)
	prev_page := pl5MtrackAt(t, s, 3, 2)

	// Branch 1 (limit smaller than the rest of the page).
	zigTry(t, s.EraseRowBounded(pl5Screen(0, 4), 1))
	pl5MexpectPinAt(t, s, on_row, 1, 0, 0)
	pl5MexpectPinAt(t, s, below, 1, 0, 4)
	pl5MexpectPinAt(t, s, prev_page, 0, 3, 2)

	// Branch 2 (limit reaches the end of the page and beyond).
	on_row.X = 5
	zigTry(t, s.EraseRowBounded(pl5Screen(0, 4), 100))
	pl5MexpectPinAt(t, s, on_row, 1, 0, 0)
	pl5MexpectPinAt(t, s, prev_page, 0, 3, 2)
}

// Zig: test "term51: eraseRowBounded: pins in following pages cross the boundary"
func TestPl5_term51_eraseRowBounded_pins_in_following_pages_cross_the_boundary(t *testing.T) {
	testStdCapacity(t)

	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	defer s.Deinit()

	// global rows: page1 = 4,5,6; page2 = 7..11
	p4 := pl5MtrackAt(t, s, 4, 1)
	p5 := pl5MtrackAt(t, s, 5, 2)
	p6 := pl5MtrackAt(t, s, 6, 3)
	p7 := pl5MtrackAt(t, s, 7, 4)
	p8 := pl5MtrackAt(t, s, 8, 5)
	p9 := pl5MtrackAt(t, s, 9, 0)

	// Erase row 2 with limit 7 -> rows 2..9 are affected (page 2 partially).
	zigTry(t, s.EraseRowBounded(pl5Screen(0, 2), 7))

	pl5MexpectPinAt(t, s, p4, 0, 3, 1)
	pl5MexpectPinAt(t, s, p5, 1, 0, 2)
	pl5MexpectPinAt(t, s, p6, 1, 1, 3)
	pl5MexpectPinAt(t, s, p7, 1, 2, 4)
	pl5MexpectPinAt(t, s, p8, 2, 0, 5)
	pl5MexpectPinAt(t, s, p9, 2, 1, 0)
	pl5MexpectLabels(t, s, []uint32{1, 2, 4, 5, 6, 7, 8, 9, 10, 0, 11, 12})
	pl5MexpectPageDirty(t, s, []bool{true, true, true})
}

// Zig: test "term51: eraseRowBounded: dirty flags only cover the touched pages"
func TestPl5_term51_eraseRowBounded_dirty_flags_only_cover_the_touched_pages(t *testing.T) {
	testStdCapacity(t)
	sizes := []uint{4, 3, 5}

	// limit inside page 1 only
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		zigTry(t, s.EraseRowBounded(pl5Screen(0, 4), 1))
		pl5MexpectPageDirty(t, s, []bool{false, true, false})
	}()
	// reaching into page 2
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		zigTry(t, s.EraseRowBounded(pl5Screen(0, 4), 3))
		pl5MexpectPageDirty(t, s, []bool{false, true, true})
	}()
	// starting in the last page
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		zigTry(t, s.EraseRowBounded(pl5Screen(0, 8), 50))
		pl5MexpectPageDirty(t, s, []bool{false, false, true})
	}()
}

// Zig: test "term51: eraseRowBounded: every point tag selects the row"
func TestPl5_term51_eraseRowBounded_every_point_tag_selects_the_row(t *testing.T) {
	testStdCapacity(t)
	sizes := []uint{4, 3, 5}

	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		zigTry(t, s.EraseRowBounded(pl5History(0, 2), 1))
		pl5MexpectLabels(t, s, []uint32{1, 2, 4, 0, 5, 6, 7, 8, 9, 10, 11, 12})
	}()
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		zigTry(t, s.EraseRowBounded(pl5Active(0, 1), 1))
		pl5MexpectLabels(t, s, []uint32{1, 2, 3, 4, 5, 6, 7, 8, 10, 0, 11, 12})
	}()
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		zigTry(t, s.EraseRowBounded(pl5Viewport(0, 0), 1))
		pl5MexpectLabels(t, s, []uint32{1, 2, 3, 4, 5, 6, 7, 9, 0, 10, 11, 12})
	}()
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		s.viewport = PageListViewport{Tag: PageListViewportTop}
		zigTry(t, s.EraseRowBounded(pl5Viewport(0, 5), 1))
		pl5MexpectLabels(t, s, []uint32{1, 2, 3, 4, 5, 7, 0, 8, 9, 10, 11, 12})
	}()
}

// Zig: test "term51: eraseRowBounded: viewport pin cache is adjusted per branch"
func TestPl5_term51_eraseRowBounded_viewport_pin_cache_is_adjusted_per_branch(t *testing.T) {
	testStdCapacity(t)
	sizes := []uint{4, 3, 5}

	// Viewport pin (with a cache) below the erased range: unchanged.
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		*s.viewport_pin = pl5PinOf(t, s, pl5Screen(0, 6))
		s.viewport = PageListViewport{Tag: PageListViewportPin}
		s.viewport_pin_row_offset = Some[uint](6)
		zigTry(t, s.EraseRowBounded(pl5Screen(0, 1), 2))
		expectEqual(t, Some[uint](6), s.viewport_pin_row_offset)
		pl5MexpectPinAt(t, s, s.viewport_pin, 1, 2, 0)
	}()
	// Inside the range, first-page branch.
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		*s.viewport_pin = pl5PinOf(t, s, pl5Screen(0, 2))
		s.viewport = PageListViewport{Tag: PageListViewportPin}
		s.viewport_pin_row_offset = Some[uint](2)
		zigTry(t, s.EraseRowBounded(pl5Screen(0, 1), 2))
		expectEqual(t, Some[uint](1), s.viewport_pin_row_offset)
		pl5MexpectPinAt(t, s, s.viewport_pin, 0, 1, 0)
	}()
	// At page row 0 of the erased row: the pin stays, the cache too.
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		*s.viewport_pin = pl5PinOf(t, s, pl5Screen(3, 4))
		s.viewport = PageListViewport{Tag: PageListViewportPin}
		s.viewport_pin_row_offset = Some[uint](4)
		zigTry(t, s.EraseRowBounded(pl5Screen(0, 4), 1))
		expectEqual(t, Some[uint](4), s.viewport_pin_row_offset)
		pl5MexpectPinAt(t, s, s.viewport_pin, 1, 0, 0)
	}()
	// Without a cache nothing is created.
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		*s.viewport_pin = pl5PinOf(t, s, pl5Screen(0, 2))
		s.viewport = PageListViewport{Tag: PageListViewportPin}
		zigTry(t, s.EraseRowBounded(pl5Screen(0, 1), 2))
		expectEqual(t, None[uint](), s.viewport_pin_row_offset)
		expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	}()
	// Active viewport is never changed by eraseRowBounded.
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		zigTry(t, s.EraseRowBounded(pl5Screen(0, 0), 100))
		expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	}()
}

// Zig: test "term51: eraseRowBounded: managed memory of the vacated row is released"
func TestPl5_term51_eraseRowBounded_managed_memory_of_the_vacated_row_is_released(t *testing.T) {
	testStdCapacity(t)

	s := pl5Mbuild(t, 5, 2, []uint{2, 2})
	defer s.Deinit()
	p0 := &s.pages.first.data

	rac := p0.GetRowAndCell(0, 0)
	id, err := p0.styles.Add(Style{Flags: StyleFlags{Bold: true}})
	zigTry(t, err)
	rac.row.styled = true
	rac.cell.style_id = id
	rac2 := p0.GetRowAndCell(1, 0)
	zigTry(t, p0.AppendGrapheme(rac2.row, rac2.cell, 0x301))

	// Branch 1 clears the erased row before rotating.
	zigTry(t, s.EraseRowBounded(pl5Screen(0, 0), 1))
	expectEqual(t, uint(0), p0.styles.Count())
	expectEqual(t, uint(0), p0.GraphemeCount())
	pl5MexpectLabels(t, s, []uint32{2, 0, 3, 4})
}

// Zig: test "term51: eraseRowBounded: clone failure across a page boundary returns the page's error"
func TestPl5_term51_eraseRowBounded_clone_failure_across_a_page_boundary_returns_the_page_s_error(t *testing.T) {
	testStdCapacity(t)

	s := pl5Mbuild(t, 215, 2, []uint{2, 2})
	defer s.Deinit()
	p0 := &s.pages.first.data
	p1 := &s.pages.first.next.data

	pl5MfillStyles(t, p0)
	{
		id, err := p1.styles.Add(Style{Flags: StyleFlags{Bold: true}})
		zigTry(t, err)
		rac := p1.GetRowAndCell(0, 0)
		rac.row.styled = true
		rac.cell.style_id = id
	}

	// limit 2 from the last row of page 0 reaches into page 1.
	pl5ExpectError(t, ErrStyleSetOutOfMemory, s.EraseRowBounded(pl5Screen(0, 1), 2))
}

// pl5MmemoryTotal is term51MmemoryTotal.
func pl5MmemoryTotal(s *PageList) uint64 {
	n := uint64(0)
	for node := s.pages.first; node != nil; node = node.next {
		n += pageListMemoryLen(&node.data)
	}
	return n
}

// pl5MeraseRowsCheck is term51MeraseRowsCheck: check an eraseRows call
// against a model. `k` is the page whose row 0 is the top-left, `e` the
// global index of the last erased row, `regrow` is true for an `active`
// top-left. Pins sit on every row at x = 3.
func pl5MeraseRowsCheck(
	t *testing.T,
	sizes []uint,
	active_rows SizeCellCountInt,
	tl Point,
	bl Optional[Point],
	k uint,
	e uint,
	regrow bool,
) {
	t.Helper()
	s := pl5Mbuild(t, 6, active_rows, sizes)
	defer s.Deinit()
	total := s.total_rows

	old := make([]uint32, total)
	pl5Mlabels(t, s, old)

	pins := make([]*PageListPin, total)
	for q := uint(0); q < total; q++ {
		pins[q] = pl5MtrackAt(t, s, q, 3)
	}

	base_k := uint(0)
	for _, n := range sizes[:k] {
		base_k += n
	}
	erased := e - base_k + 1
	ek, ey := pl5MlocOf(sizes, e)
	full_last := ey == sizes[ek]-1

	// New page sizes.
	var new_sizes []uint
	new_sizes = append(new_sizes, sizes[:k]...)
	if !full_last {
		new_sizes = append(new_sizes, sizes[ek]-(ey+1))
	}
	new_sizes = append(new_sizes, sizes[ek+1:]...)
	if regrow {
		new_sizes[len(new_sizes)-1] += erased
	}

	s.EraseRows(tl, bl)

	// Rows
	var expected []uint32
	for i := uint(0); i < total; i++ {
		if i >= base_k && i <= e {
			continue
		}
		expected = append(expected, old[i])
	}
	if regrow {
		for i := uint(0); i < erased; i++ {
			expected = append(expected, 0)
		}
	}
	pl5MexpectLabels(t, s, expected)
	pl5MexpectSizes(t, s, new_sizes)
	expectEqual(t, uint(len(expected)), s.total_rows)

	// Pins
	for q := uint(0); q < total; q++ {
		if q < base_k {
			pl5MexpectPinGlobal(t, s, new_sizes, pins[q], q, 3)
		} else if q > e {
			pl5MexpectPinGlobal(t, s, new_sizes, pins[q], q-erased, 3)
		} else {
			j, _ := pl5MlocOf(sizes, q)
			fully_erased := j < ek || full_last
			var dest uint
			if fully_erased {
				if k > 0 {
					dest = k - 1
				} else {
					dest = 0
				}
			} else {
				dest = k
			}
			pl5MexpectPinAt(t, s, pins[q], dest, 0, 0)
		}
		expect(t, !pins[q].Garbage, "!pins[q].garbage")
	}
	// The viewport pin started at (page 0, row 0, x 0): it is either
	// untouched or, when on an erased row, reset to (0, 0) of the new page.
	pl5MexpectPinAt(t, s, s.viewport_pin, 0, 0, 0)

	// The partially erased page: moved rows are dirty, the cells beyond
	// the new size were cleared, and the page itself is not marked dirty.
	if !full_last {
		node := pl5MnodeAt(s, k)
		new_rows := sizes[ek] - (ey + 1)
		expect(t, !node.data.dirty, "!node.data.dirty")
		for y := uint(0); y < new_rows; y++ {
			expect(t, node.data.GetRow(y).dirty, "node.data.getRow(y).dirty")
		}
		for y := new_rows; y < sizes[ek]; y++ {
			row := &node.data.rows[y]
			for _, c := range node.data.GetCells(row) {
				expectEqual(t, uint32(0), c.Codepoint())
			}
		}
	}
}

// Zig: test "term51: eraseRows: oracle with screen top-left over layouts, pages and end rows"
func TestPl5_term51_eraseRows_oracle_with_screen_top_left_over_layouts_pages_and_end_rows(t *testing.T) {
	testStdCapacity(t)
	for _, sizes := range pl5MLayouts {
		base := uint(0)
		for ki, n := range sizes {
			k := uint(ki)
			for e := base; e < 9; e++ {
				// Erasing every row would leave the list without rows.
				if k == 0 && e == 8 {
					continue
				}
				pl5MeraseRowsCheck(
					t,
					sizes,
					4,
					pl5Screen(0, uint32(base)),
					Some(pl5Screen(0, uint32(e))),
					k,
					e,
					false,
				)
			}
			// bl = null erases to the last row
			if k > 0 {
				pl5MeraseRowsCheck(
					t,
					sizes,
					4,
					pl5Screen(0, uint32(base)),
					None[Point](),
					k,
					8,
					false,
				)
			}
			base += n
		}
	}
}

// Zig: test "term51: eraseRows: oracle with history top-left over layouts, pages and end rows"
func TestPl5_term51_eraseRows_oracle_with_history_top_left_over_layouts_pages_and_end_rows(t *testing.T) {
	testStdCapacity(t)
	// With 4 active rows and 9 total, history is rows 0..4.
	for _, sizes := range pl5MLayouts {
		base := uint(0)
		for ki, n := range sizes {
			k := uint(ki)
			if base > 4 {
				break
			}
			for e := base; e < 5; e++ {
				pl5MeraseRowsCheck(
					t,
					sizes,
					4,
					pl5History(0, uint32(base)),
					Some(pl5History(0, uint32(e))),
					k,
					e,
					false,
				)
			}
			pl5MeraseRowsCheck(
				t,
				sizes,
				4,
				pl5History(0, uint32(base)),
				None[Point](),
				k,
				4,
				false,
			)
			base += n
		}
	}
}

// Zig: test "term51: eraseRows: oracle with active top-left regrows the active area"
func TestPl5_term51_eraseRows_oracle_with_active_top_left_regrows_the_active_area(t *testing.T) {
	testStdCapacity(t)
	// The active area (rows 5..8) must start at a page boundary.
	layouts := [][]uint{
		{5, 4},
		{3, 2, 2, 2},
		{1, 1, 1, 1, 1, 1, 1, 1, 1},
		{5, 1, 3},
	}
	for _, sizes := range layouts {
		k := uint(0)
		base := uint(0)
		for base < 5 {
			base += sizes[k]
			k++
		}
		expectEqual(t, uint(5), base)

		for e := uint(5); e < 9; e++ {
			pl5MeraseRowsCheck(
				t,
				sizes,
				4,
				pl5Active(0, 0),
				Some(pl5Active(0, uint32(e-5))),
				k,
				e,
				true,
			)
		}
		pl5MeraseRowsCheck(t, sizes, 4, pl5Active(0, 0), None[Point](), k, 8, true)
	}
}

// Zig: test "term51: eraseRows: removes a whole middle page (hand computed)"
func TestPl5_term51_eraseRows_removes_a_whole_middle_page_hand_computed(t *testing.T) {
	testStdCapacity(t)
	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	defer s.Deinit()

	in_p0 := pl5MtrackAt(t, s, 3, 1)
	in_p1_a := pl5MtrackAt(t, s, 4, 2)
	in_p1_b := pl5MtrackAt(t, s, 6, 5)
	in_p2 := pl5MtrackAt(t, s, 9, 4)
	before_pages := s.TotalPages()

	s.EraseRows(pl5Screen(0, 4), Some(pl5Screen(0, 6)))

	expectEqual(t, before_pages-1, s.TotalPages())
	pl5MexpectSizes(t, s, []uint{4, 5})
	pl5MexpectLabels(t, s, []uint32{1, 2, 3, 4, 8, 9, 10, 11, 12})
	expectEqual(t, uint(9), s.total_rows)

	pl5MexpectPinAt(t, s, in_p0, 0, 3, 1)
	// Pins in an erased page go to the TOP of the previous page.
	pl5MexpectPinAt(t, s, in_p1_a, 0, 0, 0)
	pl5MexpectPinAt(t, s, in_p1_b, 0, 0, 0)
	pl5MexpectPinAt(t, s, in_p2, 1, 2, 4)
}

// Zig: test "term51: eraseRows: partial top page (hand computed)"
func TestPl5_term51_eraseRows_partial_top_page_hand_computed(t *testing.T) {
	testStdCapacity(t)
	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	defer s.Deinit()

	p0 := pl5MtrackAt(t, s, 0, 2)
	p1 := pl5MtrackAt(t, s, 1, 3)
	p2 := pl5MtrackAt(t, s, 2, 4)
	p3 := pl5MtrackAt(t, s, 3, 5)
	other := pl5MtrackAt(t, s, 5, 1)

	// Erase rows 0 and 1.
	s.EraseRows(pl5Screen(0, 0), Some(pl5Screen(0, 1)))

	pl5MexpectSizes(t, s, []uint{2, 3, 5})
	pl5MexpectLabels(t, s, []uint32{3, 4, 5, 6, 7, 8, 9, 10, 11, 12})
	expectEqual(t, uint(10), s.total_rows)
	pl5MexpectPinAt(t, s, p0, 0, 0, 0)
	pl5MexpectPinAt(t, s, p1, 0, 0, 0)
	pl5MexpectPinAt(t, s, p2, 0, 0, 4)
	pl5MexpectPinAt(t, s, p3, 0, 1, 5)
	pl5MexpectPinAt(t, s, other, 1, 1, 1)
	pl5MexpectPinAt(t, s, s.viewport_pin, 0, 0, 0)

	// The moved rows are dirty; the page flag is not set.
	expect(t, s.pages.first.data.GetRow(0).dirty, "s.pages.first.?.data.getRow(0).dirty")
	expect(t, s.pages.first.data.GetRow(1).dirty, "s.pages.first.?.data.getRow(1).dirty")
	expect(t, !s.pages.first.data.dirty, "!s.pages.first.?.data.dirty")
	expect(t, !s.pages.first.next.data.dirty, "!s.pages.first.?.next.?.data.dirty")
}

// Zig: test "term51: eraseRows: spans pages, ending inside a later page (hand computed)"
func TestPl5_term51_eraseRows_spans_pages_ending_inside_a_later_page_hand_computed(t *testing.T) {
	testStdCapacity(t)
	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	defer s.Deinit()

	p1 := pl5MtrackAt(t, s, 5, 2)   // in erased page 1
	p2a := pl5MtrackAt(t, s, 7, 3)  // erased row of page 2
	p2b := pl5MtrackAt(t, s, 8, 4)  // erased row of page 2
	p2c := pl5MtrackAt(t, s, 9, 5)  // survives, y 2 -> 0
	p2d := pl5MtrackAt(t, s, 11, 0) // survives, y 4 -> 2

	// Rows 4..8: page 1 fully, page 2's first two rows.
	s.EraseRows(pl5Screen(0, 4), Some(pl5Screen(0, 8)))

	pl5MexpectSizes(t, s, []uint{4, 3})
	pl5MexpectLabels(t, s, []uint32{1, 2, 3, 4, 10, 11, 12})
	expectEqual(t, uint(7), s.total_rows)
	pl5MexpectPinAt(t, s, p1, 0, 0, 0)
	pl5MexpectPinAt(t, s, p2a, 1, 0, 0)
	pl5MexpectPinAt(t, s, p2b, 1, 0, 0)
	pl5MexpectPinAt(t, s, p2c, 1, 0, 5)
	pl5MexpectPinAt(t, s, p2d, 1, 2, 0)
}

// Zig: test "term51: eraseRows: first page erased moves pins to the next page"
func TestPl5_term51_eraseRows_first_page_erased_moves_pins_to_the_next_page(t *testing.T) {
	testStdCapacity(t)
	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	defer s.Deinit()

	a := pl5MtrackAt(t, s, 2, 3)
	b := pl5MtrackAt(t, s, 6, 3)

	s.EraseRows(pl5Screen(0, 0), Some(pl5Screen(0, 3)))

	pl5MexpectSizes(t, s, []uint{3, 5})
	pl5MexpectPinAt(t, s, a, 0, 0, 0)
	pl5MexpectPinAt(t, s, b, 0, 2, 3)
	// The viewport pin (it was on the erased first page too).
	pl5MexpectPinAt(t, s, s.viewport_pin, 0, 0, 0)
}

// Zig: test "term51: eraseRows: bl_pt null with every top-left tag"
func TestPl5_term51_eraseRows_bl_pt_null_with_every_top_left_tag(t *testing.T) {
	testStdCapacity(t)

	// history: ends just before the active area (rows 0..6 here)
	func() {
		s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
		defer s.Deinit()
		s.EraseRows(pl5History(0, 0), None[Point]())
		pl5MexpectSizes(t, s, []uint{5})
		pl5MexpectLabels(t, s, []uint32{8, 9, 10, 11, 12})
		expectEqual(t, uint(5), s.total_rows)
	}()
	// history from a page boundary
	func() {
		s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
		defer s.Deinit()
		s.EraseRows(pl5History(0, 4), None[Point]())
		pl5MexpectSizes(t, s, []uint{4, 5})
		pl5MexpectLabels(t, s, []uint32{1, 2, 3, 4, 8, 9, 10, 11, 12})
	}()
	// screen from the second page to the end
	func() {
		s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
		defer s.Deinit()
		s.EraseRows(pl5Screen(0, 4), None[Point]())
		pl5MexpectSizes(t, s, []uint{4})
		pl5MexpectLabels(t, s, []uint32{1, 2, 3, 4})
		expectEqual(t, uint(4), s.total_rows)
	}()
	// active: the whole active area, regrown blank
	func() {
		s := pl5Mbuild(t, 6, 5, []uint{2, 5})
		defer s.Deinit()
		s.EraseRows(pl5Active(0, 0), None[Point]())
		pl5MexpectSizes(t, s, []uint{7})
		pl5MexpectLabels(t, s, []uint32{1, 2, 0, 0, 0, 0, 0})
		expectEqual(t, uint(7), s.total_rows)
	}()
}

// Zig: test "term51: eraseRows: erasing a single-page list entirely keeps pins as they are"
func TestPl5_term51_eraseRows_erasing_a_single_page_list_entirely_keeps_pins_as_they_are(t *testing.T) {
	testStdCapacity(t)

	s := pl5Mbuild(t, 6, 5, []uint{5})
	defer s.Deinit()
	a := pl5MtrackAt(t, s, 3, 2)

	// While regrowing one row at a time, the pin at row 3 is beyond the
	// page's size, which upstream's integrity check in grow() rejects
	// (TrackedPinInvalid). Pause the checks around the call.
	s.PauseIntegrityChecks(true)
	s.EraseRows(pl5Active(0, 0), None[Point]())
	s.PauseIntegrityChecks(false)

	// Upstream special case: the page is reinit'd in place and the active
	// area regrown; pins on that page are not touched.
	pl5MexpectSizes(t, s, []uint{5})
	pl5MexpectLabels(t, s, []uint32{0, 0, 0, 0, 0})
	expectEqual(t, uint(5), s.total_rows)
	pl5MexpectPinAt(t, s, a, 0, 3, 2)
}

// Zig: test "term51: eraseRows: erasing every page of an active list keeps pins on the last page"
func TestPl5_term51_eraseRows_erasing_every_page_of_an_active_list_keeps_pins_on_the_last_page(t *testing.T) {
	testStdCapacity(t)

	// 9 active rows over 4 pages.
	s := pl5Mbuild(t, 6, 9, []uint{3, 2, 2, 2})
	defer s.Deinit()
	on_p0 := pl5MtrackAt(t, s, 1, 2)
	on_p1 := pl5MtrackAt(t, s, 4, 3)
	on_p2 := pl5MtrackAt(t, s, 6, 4)
	on_p3 := pl5MtrackAt(t, s, 8, 5)

	// Same integrity pause as above (pins at rows > 0 of the emptied page).
	s.PauseIntegrityChecks(true)
	s.EraseRows(pl5Active(0, 0), None[Point]())
	s.PauseIntegrityChecks(false)

	pl5MexpectSizes(t, s, []uint{9})
	pl5MexpectLabels(t, s, []uint32{0, 0, 0, 0, 0, 0, 0, 0, 0})
	expectEqual(t, uint(9), s.total_rows)
	// Pins of erased pages hop forward page by page to (row 0, x 0) of the
	// surviving page; the pin on the surviving page itself is untouched.
	pl5MexpectPinAt(t, s, on_p0, 0, 0, 0)
	pl5MexpectPinAt(t, s, on_p1, 0, 0, 0)
	pl5MexpectPinAt(t, s, on_p2, 0, 0, 0)
	pl5MexpectPinAt(t, s, on_p3, 0, 1, 5)
	pl5MexpectPinAt(t, s, s.viewport_pin, 0, 0, 0)
}

// Zig: test "term51: eraseRows: erasing every screen row leaves an empty page (integrity paused)"
func TestPl5_term51_eraseRows_erasing_every_screen_row_leaves_an_empty_page_integrity_paused(t *testing.T) {
	testStdCapacity(t)

	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	defer s.Deinit()
	// The empty list is not a valid state (every pin becomes invalid), so
	// upstream's integrity check would panic; pause it for the whole test.
	s.PauseIntegrityChecks(true)

	s.EraseRows(pl5Screen(0, 0), None[Point]())

	pl5MexpectSizes(t, s, []uint{0})
	expectEqual(t, uint(0), s.total_rows)
	expectEqual(t, uint(1), s.TotalPages())
}

// Zig: test "term51: eraseRows: viewport transitions"
func TestPl5_term51_eraseRows_viewport_transitions(t *testing.T) {
	testStdCapacity(t)
	sizes := []uint{4, 3, 5}

	// top -> active when the history is erased entirely
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		s.viewport = PageListViewport{Tag: PageListViewportTop}
		s.EraseRows(pl5History(0, 0), None[Point]())
		expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	}()
	// top stays top when only part of the history is erased
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		s.viewport = PageListViewport{Tag: PageListViewportTop}
		s.EraseRows(pl5History(0, 0), Some(pl5History(0, 1)))
		expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
	}()
	// pin with cached offset: offset reduced by the number of erased rows
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		*s.viewport_pin = pl5PinOf(t, s, pl5Screen(0, 5))
		s.viewport = PageListViewport{Tag: PageListViewportPin}
		s.viewport_pin_row_offset = Some[uint](5)
		s.EraseRows(pl5Screen(0, 0), Some(pl5Screen(0, 3)))
		expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
		expectEqual(t, Some[uint](1), s.viewport_pin_row_offset)
		pl5MexpectPinAt(t, s, s.viewport_pin, 0, 1, 0)
	}()
	// pin without a cached offset: left alone
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		*s.viewport_pin = pl5PinOf(t, s, pl5Screen(0, 5))
		s.viewport = PageListViewport{Tag: PageListViewportPin}
		s.EraseRows(pl5Screen(0, 0), Some(pl5Screen(0, 3)))
		expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
		expectEqual(t, None[uint](), s.viewport_pin_row_offset)
	}()
	// pin offset smaller than the erased rows -> top (pin was erased)
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		*s.viewport_pin = pl5PinOf(t, s, pl5Screen(0, 2))
		s.viewport = PageListViewport{Tag: PageListViewportPin}
		s.viewport_pin_row_offset = Some[uint](2)
		s.EraseRows(pl5Screen(0, 0), Some(pl5Screen(0, 3)))
		expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
	}()
	// offset exactly equal to the erased rows: not smaller, stays a pin at 0
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		*s.viewport_pin = pl5PinOf(t, s, pl5Screen(0, 4))
		s.viewport = PageListViewport{Tag: PageListViewportPin}
		s.viewport_pin_row_offset = Some[uint](4)
		s.EraseRows(pl5Screen(0, 0), Some(pl5Screen(0, 3)))
		expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
		expectEqual(t, Some[uint](0), s.viewport_pin_row_offset)
	}()
	// pin becomes active
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		*s.viewport_pin = pl5PinOf(t, s, pl5Screen(0, 5))
		s.viewport = PageListViewport{Tag: PageListViewportPin}
		s.viewport_pin_row_offset = Some[uint](5)
		s.EraseRows(pl5Screen(0, 0), Some(pl5Screen(0, 6)))
		expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	}()
	// active stays active
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		s.EraseRows(pl5Screen(0, 0), Some(pl5Screen(0, 3)))
		expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	}()
}

// Zig: test "term51: eraseRows: viewport cache is reduced even if the erased rows are below the pin"
func TestPl5_term51_eraseRows_viewport_cache_is_reduced_even_if_the_erased_rows_are_below_the_pin(t *testing.T) {
	testStdCapacity(t)

	// Pin at row 3 (cache 3); erase page 1 (3 rows, below the pin). The
	// cache is reduced by 3 anyway, leaving it stale (integrity paused).
	func() {
		s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
		defer s.Deinit()
		*s.viewport_pin = pl5PinOf(t, s, pl5Screen(0, 3))
		s.viewport = PageListViewport{Tag: PageListViewportPin}
		s.viewport_pin_row_offset = Some[uint](3)
		s.PauseIntegrityChecks(true)
		s.EraseRows(pl5Screen(0, 4), Some(pl5Screen(0, 6)))
		s.PauseIntegrityChecks(false)
		expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
		expectEqual(t, uint(3), pl5MglobalOf(s, s.viewport_pin))
		expectEqual(t, Some[uint](0), s.viewport_pin_row_offset)
		s.viewport_pin_row_offset = None[uint]()
	}()
	// Pin at row 2 (cache 2): 2 < 3 erased, so the viewport jumps to top
	// although its pin did not move.
	func() {
		s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
		defer s.Deinit()
		*s.viewport_pin = pl5PinOf(t, s, pl5Screen(0, 2))
		s.viewport = PageListViewport{Tag: PageListViewportPin}
		s.viewport_pin_row_offset = Some[uint](2)
		s.EraseRows(pl5Screen(0, 4), Some(pl5Screen(0, 6)))
		expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
		pl5MexpectPinAt(t, s, s.viewport_pin, 0, 2, 0)
	}()
}

// Zig: test "term51: eraseRows: top-left tags viewport (active and pinned)"
func TestPl5_term51_eraseRows_top_left_tags_viewport_active_and_pinned(t *testing.T) {
	testStdCapacity(t)

	// Pinned viewport at a page boundary: viewport y0..y2 is page 1.
	func() {
		s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
		defer s.Deinit()
		*s.viewport_pin = pl5PinOf(t, s, pl5Screen(0, 4))
		s.viewport = PageListViewport{Tag: PageListViewportPin}
		s.EraseRows(pl5Viewport(0, 0), Some(pl5Viewport(0, 2)))
		pl5MexpectSizes(t, s, []uint{4, 5})
		pl5MexpectLabels(t, s, []uint32{1, 2, 3, 4, 8, 9, 10, 11, 12})
		// The viewport pin was in the erased page: it moved to page 0.
		pl5MexpectPinAt(t, s, s.viewport_pin, 0, 0, 0)
	}()
	// Top viewport behaves like screen.
	func() {
		s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
		defer s.Deinit()
		s.viewport = PageListViewport{Tag: PageListViewportTop}
		s.EraseRows(pl5Viewport(0, 0), Some(pl5Viewport(0, 3)))
		pl5MexpectSizes(t, s, []uint{3, 5})
	}()
}

// Zig: test "term51: eraseRows: styles and graphemes of erased rows are released"
func TestPl5_term51_eraseRows_styles_and_graphemes_of_erased_rows_are_released(t *testing.T) {
	testStdCapacity(t)
	s := pl5Mbuild(t, 5, 2, []uint{4, 2})
	defer s.Deinit()
	p0 := &s.pages.first.data

	id, err := p0.styles.Add(Style{Flags: StyleFlags{Bold: true}})
	zigTry(t, err)
	rac := p0.GetRowAndCell(0, 1)
	rac.row.styled = true
	rac.cell.style_id = id
	rac2 := p0.GetRowAndCell(1, 2)
	zigTry(t, p0.AppendGrapheme(rac2.row, rac2.cell, 0x301))
	expectEqual(t, uint(1), p0.styles.Count())
	expectEqual(t, uint(1), p0.GraphemeCount())

	// Rows 0..2 are erased; row 3 remains.
	s.EraseRows(pl5Screen(0, 0), Some(pl5Screen(0, 2)))
	expectEqual(t, uint(0), p0.styles.Count())
	expectEqual(t, uint(0), p0.GraphemeCount())
	pl5MexpectLabels(t, s, []uint32{4, 5, 6})
}

// Zig: test "term51: erasePage: middle page"
func TestPl5_term51_erasePage_middle_page(t *testing.T) {
	testStdCapacity(t)
	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	defer s.Deinit()

	a := pl5MtrackAt(t, s, 1, 2)
	mid_a := pl5MtrackAt(t, s, 4, 3)
	mid_b := pl5MtrackAt(t, s, 6, 4)
	c := pl5MtrackAt(t, s, 9, 5)
	node := pl5MnodeAt(s, 1)

	s.erasePage(node)
	// erasePage does not account for the removed rows.
	expectEqual(t, uint(12), s.total_rows)
	s.total_rows -= 3

	expectEqual(t, uint(2), s.TotalPages())
	pl5MexpectSizes(t, s, []uint{4, 5})
	pl5MexpectLabels(t, s, []uint32{1, 2, 3, 4, 8, 9, 10, 11, 12})
	pl5MexpectPinAt(t, s, a, 0, 1, 2)
	pl5MexpectPinAt(t, s, mid_a, 0, 0, 0)
	pl5MexpectPinAt(t, s, mid_b, 0, 0, 0)
	pl5MexpectPinAt(t, s, c, 1, 2, 5)
	expect(t, !mid_a.Garbage, "!mid_a.garbage")
	expect(t, !mid_b.Garbage, "!mid_b.garbage")
	expect(t, s.pages.first.next == s.pages.last, "s.pages.first.?.next == s.pages.last")
}

// Zig: test "term51: erasePage: first page moves pins to the next page, last page to the previous"
func TestPl5_term51_erasePage_first_page_moves_pins_to_the_next_page_last_page_to_the_previous(t *testing.T) {
	testStdCapacity(t)

	// first page
	func() {
		s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
		defer s.Deinit()
		a := pl5MtrackAt(t, s, 2, 3)
		b := pl5MtrackAt(t, s, 5, 2)
		s.erasePage(s.pages.first)
		s.total_rows -= 4
		pl5MexpectSizes(t, s, []uint{3, 5})
		pl5MexpectPinAt(t, s, a, 0, 0, 0)
		pl5MexpectPinAt(t, s, b, 0, 1, 2)
		// viewport pin was at (0,0): follows the same rule
		pl5MexpectPinAt(t, s, s.viewport_pin, 0, 0, 0)
		expect(t, s.pages.first.prev == nil, "s.pages.first.?.prev == null")
		pl5MexpectLabels(t, s, []uint32{5, 6, 7, 8, 9, 10, 11, 12})
	}()
	// last page
	func() {
		s := pl5Mbuild(t, 6, 3, []uint{4, 3, 5})
		defer s.Deinit()
		a := pl5MtrackAt(t, s, 10, 3)
		b := pl5MtrackAt(t, s, 5, 2)
		s.erasePage(s.pages.last)
		s.total_rows -= 5
		pl5MexpectSizes(t, s, []uint{4, 3})
		pl5MexpectPinAt(t, s, a, 1, 0, 0)
		pl5MexpectPinAt(t, s, b, 1, 1, 2)
		expect(t, s.pages.last.next == nil, "s.pages.last.?.next == null")
	}()
	// two pages: erase the first, the second is the only one left
	func() {
		s := pl5Mbuild(t, 6, 3, []uint{2, 4})
		defer s.Deinit()
		a := pl5MtrackAt(t, s, 1, 3)
		s.erasePage(s.pages.first)
		s.total_rows -= 2
		expectEqual(t, uint(1), s.TotalPages())
		expect(t, s.pages.first == s.pages.last, "s.pages.first == s.pages.last")
		pl5MexpectPinAt(t, s, a, 0, 0, 0)
	}()
}

// Zig: test "term51: eraseRows: layout page_size byte accounting follows erased pages"
func TestPl5_term51_eraseRows_layout_page_size_byte_accounting_follows_erased_pages(t *testing.T) {
	testStdCapacity(t)
	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	defer s.Deinit()
	expectEqual(t, pl5MmemoryTotal(s), s.page_size)
	before := s.page_size
	bytes := pageListMemoryLen(&pl5MnodeAt(s, 1).data)

	// Erasing a page releases its bytes; a partially erased page keeps its
	// memory.
	s.EraseRows(pl5Screen(0, 4), Some(pl5Screen(0, 7)))
	pl5MexpectSizes(t, s, []uint{4, 4})
	expectEqual(t, before-bytes, s.page_size)
	expectEqual(t, pl5MmemoryTotal(s), s.page_size)

	// Erasing everything of a single page keeps it (reinit in place).
	one := pl5Mbuild(t, 6, 5, []uint{5})
	defer one.Deinit()
	one_bytes := one.page_size
	one.EraseRows(pl5Active(0, 0), None[Point]())
	expectEqual(t, one_bytes, one.page_size)
}

// Zig: test "term51: erasePage: layout page_size byte accounting drops the page's bytes"
func TestPl5_term51_erasePage_layout_page_size_byte_accounting_drops_the_page_s_bytes(t *testing.T) {
	testStdCapacity(t)
	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	defer s.Deinit()
	node := pl5MnodeAt(s, 1)
	bytes := pageListMemoryLen(&node.data)
	before := s.page_size
	s.erasePage(node)
	s.total_rows -= 3
	expectEqual(t, before-bytes, s.page_size)
	expectEqual(t, pl5MmemoryTotal(s), s.page_size)
}

// pl5MexpectPinIs is term51MexpectPinIs.
func pl5MexpectPinIs(t *testing.T, s *PageList, sizes []uint, got Optional[PageListPin], g uint, x uint) {
	t.Helper()
	if !got.Valid {
		t.Fatalf("TestExpectedNotNull")
	}
	p := got.Value
	page, y := pl5MlocOf(sizes, g)
	expectEqual(t, pl5MnodeAt(s, page), p.Node)
	expectEqual(t, y, uint(p.Y))
	expectEqual(t, x, uint(p.X))
	expect(t, !p.Garbage, "!p.garbage")
}

// Zig: test "term51: pin: screen tag maps every row and rejects out-of-range points"
func TestPl5_term51_pin_screen_tag_maps_every_row_and_rejects_out_of_range_points(t *testing.T) {
	testStdCapacity(t)
	sizes := []uint{4, 3, 5}
	s := pl5Mbuild(t, 6, 5, sizes)
	defer s.Deinit()

	for y := uint(0); y < 12; y++ {
		for _, x := range []uint{0, 3, 5} {
			pl5MexpectPinIs(t, s, sizes, s.Pin(pl5Screen(SizeCellCountInt(x), uint32(y))), y, x)
		}
	}
	// One past the last row, far past it, and the largest u32.
	expect(t, !s.Pin(pl5Screen(0, 12)).Valid, "s.pin(.{ .screen = .{ .y = 12 } }) == null")
	expect(t, !s.Pin(pl5Screen(0, 13)).Valid, "s.pin(.{ .screen = .{ .y = 13 } }) == null")
	expect(t, !s.Pin(pl5Screen(0, 1000)).Valid, "s.pin(.{ .screen = .{ .y = 1000 } }) == null")
	expect(t, !s.Pin(pl5Screen(0, math.MaxUint32)).Valid, "s.pin(.{ .screen = .{ .y = std.math.maxInt(u32) } }) == null")
	// One past the last column (cols = 6) and the largest column.
	expect(t, !s.Pin(pl5Screen(6, 0)).Valid, "s.pin(.{ .screen = .{ .x = 6, .y = 0 } }) == null")
	expect(t, !s.Pin(pl5Screen(6, 11)).Valid, "s.pin(.{ .screen = .{ .x = 6, .y = 11 } }) == null")
	expect(t, !s.Pin(pl5Screen(math.MaxUint16, 0)).Valid, "s.pin(.{ .screen = .{ .x = std.math.maxInt(size.CellCountInt), .y = 0 } }) == null")
	// Column is checked before row: both out of range is still null.
	expect(t, !s.Pin(pl5Screen(6, 12)).Valid, "s.pin(.{ .screen = .{ .x = 6, .y = 12 } }) == null")
	// The returned pin is not tracked.
	expectEqual(t, uint(1), s.CountTrackedPins())
}

// Zig: test "term51: pin: active tag starts at the active area and stops at its last row"
func TestPl5_term51_pin_active_tag_starts_at_the_active_area_and_stops_at_its_last_row(t *testing.T) {
	testStdCapacity(t)
	sizes := []uint{4, 3, 5}
	s := pl5Mbuild(t, 6, 5, sizes)
	defer s.Deinit()

	// The active area is global rows 7..11.
	for y := uint(0); y < 5; y++ {
		pl5MexpectPinIs(t, s, sizes, s.Pin(pl5Active(5, uint32(y))), 7+y, 5)
	}
	expect(t, !s.Pin(pl5Active(0, 5)).Valid, "s.pin(.{ .active = .{ .y = 5 } }) == null")
	expect(t, !s.Pin(pl5Active(0, math.MaxUint32)).Valid, "s.pin(.{ .active = .{ .y = std.math.maxInt(u32) } }) == null")
	expect(t, !s.Pin(pl5Active(6, 0)).Valid, "s.pin(.{ .active = .{ .x = 6, .y = 0 } }) == null")

	// Active area straddling two pages.
	s2 := pl5Mbuild(t, 6, 6, sizes)
	defer s2.Deinit()
	// Starts at global row 6 (page 1, row 2).
	pl5MexpectPinIs(t, s2, sizes, s2.Pin(pl5Active(0, 0)), 6, 0)
	pl5MexpectPinIs(t, s2, sizes, s2.Pin(pl5Active(0, 5)), 11, 0)
	expect(t, !s2.Pin(pl5Active(0, 6)).Valid, "s2.pin(.{ .active = .{ .y = 6 } }) == null")
}

// Zig: test "term51: pin: history tag counts from the first row and is not limited to the history"
func TestPl5_term51_pin_history_tag_counts_from_the_first_row_and_is_not_limited_to_the_history(t *testing.T) {
	testStdCapacity(t)
	sizes := []uint{4, 3, 5}
	s := pl5Mbuild(t, 6, 5, sizes)
	defer s.Deinit()

	// The history is rows 0..6, but pin() only bounds-checks the list.
	for y := uint(0); y < 12; y++ {
		pl5MexpectPinIs(t, s, sizes, s.Pin(pl5History(1, uint32(y))), y, 1)
	}
	expect(t, !s.Pin(pl5History(0, 12)).Valid, "s.pin(.{ .history = .{ .y = 12 } }) == null")
	expect(t, !s.Pin(pl5History(6, 0)).Valid, "s.pin(.{ .history = .{ .x = 6 } }) == null")

	// With no history at all, history y=0 is still the first row.
	s2 := pl5Mbuild(t, 6, 5, []uint{5})
	defer s2.Deinit()
	pl5MexpectPinIs(t, s2, []uint{5}, s2.Pin(pl5History(0, 0)), 0, 0)
}

// Zig: test "term51: pin: viewport tag follows the viewport kind"
func TestPl5_term51_pin_viewport_tag_follows_the_viewport_kind(t *testing.T) {
	testStdCapacity(t)
	sizes := []uint{4, 3, 5}

	// active viewport = active area
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		pl5MexpectPinIs(t, s, sizes, s.Pin(pl5Viewport(0, 0)), 7, 0)
		pl5MexpectPinIs(t, s, sizes, s.Pin(pl5Viewport(2, 4)), 11, 2)
		expect(t, !s.Pin(pl5Viewport(0, 5)).Valid, "s.pin(.{ .viewport = .{ .y = 5 } }) == null")
	}()
	// top viewport = screen
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		s.viewport = PageListViewport{Tag: PageListViewportTop}
		pl5MexpectPinIs(t, s, sizes, s.Pin(pl5Viewport(0, 0)), 0, 0)
		pl5MexpectPinIs(t, s, sizes, s.Pin(pl5Viewport(0, 11)), 11, 0)
		expect(t, !s.Pin(pl5Viewport(0, 12)).Valid, "s.pin(.{ .viewport = .{ .y = 12 } }) == null")
	}()
	// pinned viewport: starts at the viewport pin. Rows beyond the
	// viewport height (5) are still returned while they exist.
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		*s.viewport_pin = pl5PinOf(t, s, pl5Screen(4, 5))
		s.viewport = PageListViewport{Tag: PageListViewportPin}
		// x always comes from the point, not from the viewport pin
		pl5MexpectPinIs(t, s, sizes, s.Pin(pl5Viewport(0, 0)), 5, 0)
		pl5MexpectPinIs(t, s, sizes, s.Pin(pl5Viewport(3, 6)), 11, 3)
		expect(t, !s.Pin(pl5Viewport(0, 7)).Valid, "s.pin(.{ .viewport = .{ .y = 7 } }) == null")
	}()
}

// Zig: test "term51: pin: single row and single column lists"
func TestPl5_term51_pin_single_row_and_single_column_lists(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 1, 1, None[uint64]())
	defer s.Deinit()
	p := pl5PinOf(t, s, pl5Active(0, 0))
	expectEqual(t, s.pages.first, p.Node)
	expectEqual(t, uint(0), uint(p.Y))
	expectEqual(t, uint(0), uint(p.X))
	expect(t, !s.Pin(pl5Active(1, 0)).Valid, "s.pin(.{ .active = .{ .x = 1 } }) == null")
	expect(t, !s.Pin(pl5Active(0, 1)).Valid, "s.pin(.{ .active = .{ .y = 1 } }) == null")
	expect(t, !s.Pin(pl5Screen(0, 1)).Valid, "s.pin(.{ .screen = .{ .y = 1 } }) == null")
}

// Zig: test "term51: trackedPins: a fresh list tracks only the viewport pin"
func TestPl5_term51_trackedPins_a_fresh_list_tracks_only_the_viewport_pin(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 6, 5, None[uint64]())
	defer s.Deinit()
	expectEqual(t, uint(1), s.CountTrackedPins())
	expectEqual(t, 1, len(s.TrackedPins()))
	expect(t, s.TrackedPins()[0] == s.viewport_pin, "s.trackedPins()[0] == s.viewport_pin")
}

// Zig: test "term51: countTrackedPins: counts the viewport pin plus tracked pins"
func TestPl5_term51_countTrackedPins_counts_the_viewport_pin_plus_tracked_pins(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 6, 5, None[uint64]())
	defer s.Deinit()

	var pins [20]*PageListPin
	for i := uint(0); i < 20; i++ {
		expectEqual(t, 1+i, s.CountTrackedPins())
		pins[i] = pl5Track(t, s, pl5PinOf(t, s, pl5Active(0, 0)))
		expectEqual(t, 2+i, s.CountTrackedPins())
	}
	for i := uint(0); i < 20; i++ {
		s.UntrackPin(pins[i])
		expectEqual(t, 20-i, s.CountTrackedPins())
	}
	expectEqual(t, uint(1), s.CountTrackedPins())
}

// Zig: test "term51: trackPin: creates an independent pin with the same location"
func TestPl5_term51_trackPin_creates_an_independent_pin_with_the_same_location(t *testing.T) {
	testStdCapacity(t)
	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	defer s.Deinit()

	orig := pl5PinOf(t, s, pl5Screen(3, 5))
	tr := pl5Track(t, s, orig)
	expect(t, tr != &orig, "t != &orig")
	expect(t, tr != s.viewport_pin, "t != s.viewport_pin")
	expectEqual(t, orig.Node, tr.Node)
	expectEqual(t, orig.Y, tr.Y)
	expectEqual(t, orig.X, tr.X)
	expect(t, !tr.Garbage, "!t.garbage")

	// Changing the original does not change the tracked pin.
	orig.X = 0
	orig.Y = 0
	pl5MexpectPinAt(t, s, tr, 1, 1, 3)

	// The tracked pin is in the list, after the viewport pin.
	expectEqual(t, uint(2), s.CountTrackedPins())
	expect(t, s.TrackedPins()[0] == s.viewport_pin, "s.trackedPins()[0] == s.viewport_pin")
	expect(t, s.TrackedPins()[1] == tr, "s.trackedPins()[1] == t")

	// Two pins at the same place are two pins.
	t2 := pl5Track(t, s, *tr)
	expect(t, t2 != tr, "t2 != t")
	expectEqual(t, uint(3), s.CountTrackedPins())
	expect(t, s.TrackedPins()[2] == t2, "s.trackedPins()[2] == t2")

	// Pins in the first and last cells.
	first := pl5Track(t, s, pl5PinOf(t, s, pl5Screen(0, 0)))
	pl5MexpectPinAt(t, s, first, 0, 0, 0)
	last := pl5Track(t, s, pl5PinOf(t, s, pl5Screen(5, 11)))
	pl5MexpectPinAt(t, s, last, 2, 4, 5)
}

// Zig: test "term51: trackPin: keeps a pin's garbage flag"
func TestPl5_term51_trackPin_keeps_a_pin_s_garbage_flag(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 6, 5, None[uint64]())
	defer s.Deinit()
	p := pl5PinOf(t, s, pl5Active(0, 0))
	p.Garbage = true
	tr := pl5Track(t, s, p)
	expect(t, tr.Garbage, "t.garbage")
}

// Zig: test "term51: untrackPin: removes the pin and keeps the others (swap remove order)"
func TestPl5_term51_untrackPin_removes_the_pin_and_keeps_the_others_swap_remove_order(t *testing.T) {
	testStdCapacity(t)
	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	defer s.Deinit()

	a := pl5MtrackAt(t, s, 0, 0)
	b := pl5MtrackAt(t, s, 1, 0)
	c := pl5MtrackAt(t, s, 2, 0)
	d := pl5MtrackAt(t, s, 3, 0)
	expectEqual(t, uint(5), s.CountTrackedPins())

	// The last key replaces the removed one.
	s.UntrackPin(b)
	expectEqual(t, uint(4), s.CountTrackedPins())
	{
		keys := s.TrackedPins()
		expect(t, keys[0] == s.viewport_pin, "keys[0] == s.viewport_pin")
		expect(t, keys[1] == a, "keys[1] == a")
		expect(t, keys[2] == d, "keys[2] == d")
		expect(t, keys[3] == c, "keys[3] == c")
	}

	// Removing the last one.
	s.UntrackPin(c)
	expectEqual(t, uint(3), s.CountTrackedPins())
	expect(t, s.TrackedPins()[2] == d, "s.trackedPins()[2] == d")

	// Removing the first tracked one.
	s.UntrackPin(a)
	expectEqual(t, uint(2), s.CountTrackedPins())
	expect(t, s.TrackedPins()[1] == d, "s.trackedPins()[1] == d")

	s.UntrackPin(d)
	expectEqual(t, uint(1), s.CountTrackedPins())
	expect(t, s.TrackedPins()[0] == s.viewport_pin, "s.trackedPins()[0] == s.viewport_pin")
}

// Zig: test "term51: untrackPin: a pin that is not tracked is ignored"
func TestPl5_term51_untrackPin_a_pin_that_is_not_tracked_is_ignored(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 6, 5, None[uint64]())
	defer s.Deinit()

	stack_pin := pl5PinOf(t, s, pl5Active(0, 0))
	tr := pl5Track(t, s, stack_pin)

	s.UntrackPin(&stack_pin)
	expectEqual(t, uint(2), s.CountTrackedPins())

	// Untracking twice is harmless: the second call finds nothing.
	s.UntrackPin(tr)
	expectEqual(t, uint(1), s.CountTrackedPins())
	s.UntrackPin(tr)
	expectEqual(t, uint(1), s.CountTrackedPins())
}

// Zig: test "term51: untrackPin: an untracked pin stops following the list"
func TestPl5_term51_untrackPin_an_untracked_pin_stops_following_the_list(t *testing.T) {
	testStdCapacity(t)
	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	defer s.Deinit()

	kept := pl5MtrackAt(t, s, 6, 1)
	dropped := pl5MtrackAt(t, s, 6, 2)
	s.UntrackPin(dropped)
	zigTry(t, s.EraseRow(pl5Screen(0, 0)))
	pl5MexpectPinAt(t, s, kept, 1, 1, 1)
	expectEqual(t, uint(2), s.CountTrackedPins())
}

// Zig: test "term51: trackedPins: returns the live keys in insertion order"
func TestPl5_term51_trackedPins_returns_the_live_keys_in_insertion_order(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 6, 5, None[uint64]())
	defer s.Deinit()
	a := pl5Track(t, s, pl5PinOf(t, s, pl5Active(1, 0)))
	b := pl5Track(t, s, pl5PinOf(t, s, pl5Active(2, 0)))
	keys := s.TrackedPins()
	expectEqual(t, 3, len(keys))
	expect(t, keys[0] == s.viewport_pin, "keys[0] == s.viewport_pin")
	expect(t, keys[1] == a, "keys[1] == a")
	expect(t, keys[2] == b, "keys[2] == b")
	// The slice aliases the pins themselves (writes show through).
	a.X = 4
	expectEqual(t, uint(4), uint(keys[1].X))
}

// Zig: test "term51: pinIsValid: bounds against the page's size, not its capacity"
func TestPl5_term51_pinIsValid_bounds_against_the_page_s_size_not_its_capacity(t *testing.T) {
	testStdCapacity(t)
	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	defer s.Deinit()
	n0 := pl5MnodeAt(s, 0)
	n1 := pl5MnodeAt(s, 1)
	n2 := pl5MnodeAt(s, 2)

	// valid corners
	expect(t, s.PinIsValid(PageListPin{Node: n0}), "s.pinIsValid(.{ .node = n0 })")
	expect(t, s.PinIsValid(PageListPin{Node: n0, Y: 3, X: 5}), "s.pinIsValid(.{ .node = n0, .y = 3, .x = 5 })")
	expect(t, s.PinIsValid(PageListPin{Node: n1, Y: 2, X: 5}), "s.pinIsValid(.{ .node = n1, .y = 2, .x = 5 })")
	expect(t, s.PinIsValid(PageListPin{Node: n2, Y: 4, X: 5}), "s.pinIsValid(.{ .node = n2, .y = 4, .x = 5 })")
	// one past the rows / columns of each page
	expect(t, !s.PinIsValid(PageListPin{Node: n0, Y: 4}), "!s.pinIsValid(.{ .node = n0, .y = 4 })")
	expect(t, !s.PinIsValid(PageListPin{Node: n1, Y: 3}), "!s.pinIsValid(.{ .node = n1, .y = 3 })")
	expect(t, !s.PinIsValid(PageListPin{Node: n2, Y: 5}), "!s.pinIsValid(.{ .node = n2, .y = 5 })")
	expect(t, !s.PinIsValid(PageListPin{Node: n0, X: 6}), "!s.pinIsValid(.{ .node = n0, .x = 6 })")
	expect(t, !s.PinIsValid(PageListPin{Node: n2, Y: 4, X: 6}), "!s.pinIsValid(.{ .node = n2, .y = 4, .x = 6 })")
	// far out of range
	expect(t, !s.PinIsValid(PageListPin{Node: n0, Y: 1000}), "!s.pinIsValid(.{ .node = n0, .y = 1000 })")
	expect(t, !s.PinIsValid(PageListPin{Node: n0, X: 1000}), "!s.pinIsValid(.{ .node = n0, .x = 1000 })")
	// The page capacity is larger than its size: row 4 of page 0 exists in
	// memory but is not valid.
	expect(t, n0.data.capacity.rows > 4, "n0.data.capacity.rows > 4")

	// garbage does not matter
	expect(t, s.PinIsValid(PageListPin{Node: n1, Garbage: true}), "s.pinIsValid(.{ .node = n1, .garbage = true })")

	// A node of another list is not valid for this list.
	other := pl5Init(t, 6, 5, None[uint64]())
	defer other.Deinit()
	expect(t, !s.PinIsValid(PageListPin{Node: other.pages.first}), "!s.pinIsValid(.{ .node = other.pages.first.? })")
	expect(t, other.PinIsValid(PageListPin{Node: other.pages.first}), "other.pinIsValid(.{ .node = other.pages.first.? })")
}

// Zig: test "term51: pinIsActive: boundary of the active area"
func TestPl5_term51_pinIsActive_boundary_of_the_active_area(t *testing.T) {
	testStdCapacity(t)
	sizes := []uint{4, 3, 5}

	// 5 active rows: active starts at the top of page 2.
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		for y := 0; y < 4; y++ {
			expect(t, !s.pinIsActive(PageListPin{Node: pl5MnodeAt(s, 0), Y: SizeCellCountInt(y)}), "!s.pinIsActive(.{ .node = nodeAt(0), .y = y })")
		}
		for y := 0; y < 3; y++ {
			expect(t, !s.pinIsActive(PageListPin{Node: pl5MnodeAt(s, 1), Y: SizeCellCountInt(y)}), "!s.pinIsActive(.{ .node = nodeAt(1), .y = y })")
		}
		for y := 0; y < 5; y++ {
			expect(t, s.pinIsActive(PageListPin{Node: pl5MnodeAt(s, 2), Y: SizeCellCountInt(y)}), "s.pinIsActive(.{ .node = nodeAt(2), .y = y })")
		}
	}()
	// 6 active rows: active starts at row 2 of page 1 (straddles pages).
	func() {
		s := pl5Mbuild(t, 6, 6, sizes)
		defer s.Deinit()
		for y := 0; y < 4; y++ {
			expect(t, !s.pinIsActive(PageListPin{Node: pl5MnodeAt(s, 0), Y: SizeCellCountInt(y)}), "!s.pinIsActive(.{ .node = nodeAt(0), .y = y })")
		}
		expect(t, !s.pinIsActive(PageListPin{Node: pl5MnodeAt(s, 1), Y: 0}), "!s.pinIsActive(.{ .node = nodeAt(1), .y = 0 })")
		expect(t, !s.pinIsActive(PageListPin{Node: pl5MnodeAt(s, 1), Y: 1}), "!s.pinIsActive(.{ .node = nodeAt(1), .y = 1 })")
		expect(t, s.pinIsActive(PageListPin{Node: pl5MnodeAt(s, 1), Y: 2}), "s.pinIsActive(.{ .node = nodeAt(1), .y = 2 })")
		for y := 0; y < 5; y++ {
			expect(t, s.pinIsActive(PageListPin{Node: pl5MnodeAt(s, 2), Y: SizeCellCountInt(y)}), "s.pinIsActive(.{ .node = nodeAt(2), .y = y })")
		}
		// x is ignored
		expect(t, s.pinIsActive(PageListPin{Node: pl5MnodeAt(s, 1), Y: 2, X: 5}), "s.pinIsActive(.{ .node = nodeAt(1), .y = 2, .x = 5 })")
		expect(t, !s.pinIsActive(PageListPin{Node: pl5MnodeAt(s, 1), Y: 1, X: 5}), "!s.pinIsActive(.{ .node = nodeAt(1), .y = 1, .x = 5 })")
	}()
	// Active area inside a single page with history above it.
	func() {
		s := pl5Mbuild(t, 6, 4, []uint{9})
		defer s.Deinit()
		expect(t, !s.pinIsActive(PageListPin{Node: s.pages.first, Y: 4}), "!s.pinIsActive(.{ .node = s.pages.first.?, .y = 4 })")
		expect(t, s.pinIsActive(PageListPin{Node: s.pages.first, Y: 5}), "s.pinIsActive(.{ .node = s.pages.first.?, .y = 5 })")
		expect(t, s.pinIsActive(PageListPin{Node: s.pages.first, Y: 8}), "s.pinIsActive(.{ .node = s.pages.first.?, .y = 8 })")
		expect(t, !s.pinIsActive(PageListPin{Node: s.pages.first, Y: 0}), "!s.pinIsActive(.{ .node = s.pages.first.?, .y = 0 })")
	}()
	// Whole list active.
	func() {
		s := pl5Mbuild(t, 6, 9, []uint{4, 5})
		defer s.Deinit()
		expect(t, s.pinIsActive(PageListPin{Node: s.pages.first}), "s.pinIsActive(.{ .node = s.pages.first.? })")
		expect(t, s.pinIsActive(PageListPin{Node: s.pages.last}), "s.pinIsActive(.{ .node = s.pages.last.? })")
	}()
	// A node of another list is not active.
	func() {
		s := pl5Mbuild(t, 6, 5, sizes)
		defer s.Deinit()
		other := pl5Init(t, 6, 5, None[uint64]())
		defer other.Deinit()
		expect(t, !s.pinIsActive(PageListPin{Node: other.pages.first}), "!s.pinIsActive(.{ .node = other.pages.first.? })")
	}()
}

// Zig: test "term51: pinIsTop: only row 0 of the first page"
func TestPl5_term51_pinIsTop_only_row_0_of_the_first_page(t *testing.T) {
	testStdCapacity(t)
	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	defer s.Deinit()
	n0 := pl5MnodeAt(s, 0)
	n1 := pl5MnodeAt(s, 1)
	n2 := pl5MnodeAt(s, 2)

	expect(t, s.pinIsTop(PageListPin{Node: n0}), "s.pinIsTop(.{ .node = n0 })")
	// x is ignored
	expect(t, s.pinIsTop(PageListPin{Node: n0, X: 5}), "s.pinIsTop(.{ .node = n0, .x = 5 })")
	expect(t, !s.pinIsTop(PageListPin{Node: n0, Y: 1}), "!s.pinIsTop(.{ .node = n0, .y = 1 })")
	expect(t, !s.pinIsTop(PageListPin{Node: n0, Y: 3}), "!s.pinIsTop(.{ .node = n0, .y = 3 })")
	expect(t, !s.pinIsTop(PageListPin{Node: n1}), "!s.pinIsTop(.{ .node = n1 })")
	expect(t, !s.pinIsTop(PageListPin{Node: n2}), "!s.pinIsTop(.{ .node = n2 })")
	expect(t, !s.pinIsTop(PageListPin{Node: n1, Y: 1}), "!s.pinIsTop(.{ .node = n1, .y = 1 })")

	other := pl5Init(t, 6, 5, None[uint64]())
	defer other.Deinit()
	expect(t, !s.pinIsTop(PageListPin{Node: other.pages.first}), "!s.pinIsTop(.{ .node = other.pages.first.? })")

	// A single page list.
	one := pl5Init(t, 6, 5, None[uint64]())
	defer one.Deinit()
	expect(t, one.pinIsTop(PageListPin{Node: one.pages.first}), "one.pinIsTop(.{ .node = one.pages.first.? })")
	expect(t, !one.pinIsTop(PageListPin{Node: one.pages.first, Y: 4}), "!one.pinIsTop(.{ .node = one.pages.first.?, .y = 4 })")
}

// Zig: test "term51: pointFromPin: every tag over every row of a multi-page list"
func TestPl5_term51_pointFromPin_every_tag_over_every_row_of_a_multi_page_list(t *testing.T) {
	testStdCapacity(t)
	sizes := []uint{4, 3, 5}

	// viewport = active (default); active area = global rows 7..11
	s := pl5Mbuild(t, 6, 5, sizes)
	defer s.Deinit()
	for g := uint(0); g < 12; g++ {
		page, y := pl5MlocOf(sizes, g)
		p := PageListPin{Node: pl5MnodeAt(s, page), Y: SizeCellCountInt(y), X: 2}

		expectEqualPoint(t, pl5Screen(2, uint32(g)), zigUnwrap(t, s.PointFromPin(PointTagScreen, p)))
		// history counts from the first row and has no upper bound
		expectEqualPoint(t, pl5History(2, uint32(g)), zigUnwrap(t, s.PointFromPin(PointTagHistory, p)))
		if g >= 7 {
			expectEqualPoint(t, pl5Active(2, uint32(g-7)), zigUnwrap(t, s.PointFromPin(PointTagActive, p)))
			expectEqualPoint(t, pl5Viewport(2, uint32(g-7)), zigUnwrap(t, s.PointFromPin(PointTagViewport, p)))
		} else {
			expect(t, !s.PointFromPin(PointTagActive, p).Valid, "s.pointFromPin(.active, p) == null")
			expect(t, !s.PointFromPin(PointTagViewport, p).Valid, "s.pointFromPin(.viewport, p) == null")
		}
	}

	// viewport at top: like screen
	s.viewport = PageListViewport{Tag: PageListViewportTop}
	for g := uint(0); g < 12; g++ {
		page, y := pl5MlocOf(sizes, g)
		p := PageListPin{Node: pl5MnodeAt(s, page), Y: SizeCellCountInt(y), X: 0}
		expectEqualPoint(t, pl5Viewport(0, uint32(g)), zigUnwrap(t, s.PointFromPin(PointTagViewport, p)))
	}

	// pinned viewport at global row 5 (page 1, row 1)
	*s.viewport_pin = pl5PinOf(t, s, pl5Screen(0, 5))
	s.viewport = PageListViewport{Tag: PageListViewportPin}
	for g := uint(0); g < 12; g++ {
		page, y := pl5MlocOf(sizes, g)
		p := PageListPin{Node: pl5MnodeAt(s, page), Y: SizeCellCountInt(y), X: 1}
		if g >= 5 {
			expectEqualPoint(t, pl5Viewport(1, uint32(g-5)), zigUnwrap(t, s.PointFromPin(PointTagViewport, p)))
		} else {
			expect(t, !s.PointFromPin(PointTagViewport, p).Valid, "s.pointFromPin(.viewport, p) == null")
		}
	}
}

// Zig: test "term51: pointFromPin: active area straddling two pages"
func TestPl5_term51_pointFromPin_active_area_straddling_two_pages(t *testing.T) {
	testStdCapacity(t)
	sizes := []uint{4, 3, 5}
	// 6 active rows: the area starts at page 1, row 2 (global row 6).
	s := pl5Mbuild(t, 6, 6, sizes)
	defer s.Deinit()
	n0 := pl5MnodeAt(s, 0)
	n1 := pl5MnodeAt(s, 1)
	n2 := pl5MnodeAt(s, 2)

	expect(t, !s.PointFromPin(PointTagActive, PageListPin{Node: n0}).Valid, "s.pointFromPin(.active, .{ .node = n0 }) == null")
	expect(t, !s.PointFromPin(PointTagActive, PageListPin{Node: n0, Y: 3}).Valid, "s.pointFromPin(.active, .{ .node = n0, .y = 3 }) == null")
	expect(t, !s.PointFromPin(PointTagActive, PageListPin{Node: n1, Y: 0}).Valid, "s.pointFromPin(.active, .{ .node = n1, .y = 0 }) == null")
	// tl.y > p.y is outside; tl.y == p.y is the first row
	expect(t, !s.PointFromPin(PointTagActive, PageListPin{Node: n1, Y: 1}).Valid, "s.pointFromPin(.active, .{ .node = n1, .y = 1 }) == null")
	expectEqualPoint(t, pl5Active(3, 0), zigUnwrap(t, s.PointFromPin(PointTagActive, PageListPin{Node: n1, Y: 2, X: 3})))
	// Later pages add the rows remaining in the first page of the area (1).
	expectEqualPoint(t, pl5Active(0, 1), zigUnwrap(t, s.PointFromPin(PointTagActive, PageListPin{Node: n2, Y: 0})))
	expectEqualPoint(t, pl5Active(5, 5), zigUnwrap(t, s.PointFromPin(PointTagActive, PageListPin{Node: n2, Y: 4, X: 5})))
}

// Zig: test "term51: pointFromPin: pins are not bounds checked and foreign nodes are rejected"
func TestPl5_term51_pointFromPin_pins_are_not_bounds_checked_and_foreign_nodes_are_rejected(t *testing.T) {
	testStdCapacity(t)
	sizes := []uint{4, 3, 5}
	s := pl5Mbuild(t, 6, 5, sizes)
	defer s.Deinit()
	n0 := pl5MnodeAt(s, 0)
	n1 := pl5MnodeAt(s, 1)

	// y and x beyond the page are passed through (screen: page 1 starts at 4)
	expectEqualPoint(t, pl5Screen(100, 13), zigUnwrap(t, s.PointFromPin(PointTagScreen, PageListPin{Node: n1, Y: 9, X: 100})))
	expectEqualPoint(t, pl5Screen(0, 9), zigUnwrap(t, s.PointFromPin(PointTagScreen, PageListPin{Node: n0, Y: 9})))
	// garbage is ignored
	expectEqualPoint(t, pl5Screen(0, 4), zigUnwrap(t, s.PointFromPin(PointTagScreen, PageListPin{Node: n1, Garbage: true})))

	// A node of another list is never found.
	other := pl5Init(t, 6, 5, None[uint64]())
	defer other.Deinit()
	foreign := PageListPin{Node: other.pages.first}
	expect(t, !s.PointFromPin(PointTagScreen, foreign).Valid, "s.pointFromPin(.screen, foreign) == null")
	expect(t, !s.PointFromPin(PointTagHistory, foreign).Valid, "s.pointFromPin(.history, foreign) == null")
	expect(t, !s.PointFromPin(PointTagActive, foreign).Valid, "s.pointFromPin(.active, foreign) == null")
	expect(t, !s.PointFromPin(PointTagViewport, foreign).Valid, "s.pointFromPin(.viewport, foreign) == null")
}

// Zig: test "term51: pointFromPin: single page lists and history without scrollback"
func TestPl5_term51_pointFromPin_single_page_lists_and_history_without_scrollback(t *testing.T) {
	testStdCapacity(t)

	s := pl5Mbuild(t, 6, 5, []uint{5})
	defer s.Deinit()
	n := s.pages.first
	for y := uint(0); y < 5; y++ {
		p := PageListPin{Node: n, Y: SizeCellCountInt(y), X: 5}
		expectEqualPoint(t, pl5Active(5, uint32(y)), zigUnwrap(t, s.PointFromPin(PointTagActive, p)))
		expectEqualPoint(t, pl5Screen(5, uint32(y)), zigUnwrap(t, s.PointFromPin(PointTagScreen, p)))
		expectEqualPoint(t, pl5Viewport(5, uint32(y)), zigUnwrap(t, s.PointFromPin(PointTagViewport, p)))
		expectEqualPoint(t, pl5History(5, uint32(y)), zigUnwrap(t, s.PointFromPin(PointTagHistory, p)))
	}

	// History inside one page above the active area: rows 0..3 of 9.
	s2 := pl5Mbuild(t, 6, 5, []uint{9})
	defer s2.Deinit()
	n2 := s2.pages.first
	expect(t, !s2.PointFromPin(PointTagActive, PageListPin{Node: n2, Y: 3}).Valid, "s2.pointFromPin(.active, .{ .node = n2, .y = 3 }) == null")
	expectEqualPoint(t, pl5Active(0, 0), zigUnwrap(t, s2.PointFromPin(PointTagActive, PageListPin{Node: n2, Y: 4})))
}

// Zig: test "term51: getCell: every row and column in every tag"
func TestPl5_term51_getCell_every_row_and_column_in_every_tag(t *testing.T) {
	testStdCapacity(t)
	sizes := []uint{4, 3, 5}
	s := pl5Mbuild(t, 6, 5, sizes)
	defer s.Deinit()

	for g := uint(0); g < 12; g++ {
		page, ly := pl5MlocOf(sizes, g)
		node := pl5MnodeAt(s, page)
		for x := uint(0); x < 6; x++ {
			c := zigUnwrap(t, s.GetCell(pl5Screen(SizeCellCountInt(x), uint32(g))))
			expectEqual(t, node, c.Node)
			expectEqual(t, ly, uint(c.RowIdx))
			expectEqual(t, x, uint(c.ColIdx))
			rac := node.data.GetRowAndCell(x, ly)
			expect(t, c.Row == rac.row, "c.row == rac.row")
			expect(t, c.Cell == rac.cell, "c.cell == rac.cell")
			expectEqual(t, uint32(g+1), c.Cell.Codepoint())

			// history addresses the same rows from the first row
			h := zigUnwrap(t, s.GetCell(pl5History(SizeCellCountInt(x), uint32(g))))
			expect(t, h.Cell == c.Cell, "h.cell == c.cell")
			expect(t, h.Row == c.Row, "h.row == c.row")
		}
	}

	// active y=0 is global row 7
	{
		c := zigUnwrap(t, s.GetCell(pl5Active(1, 0)))
		expectEqual(t, pl5MnodeAt(s, 2), c.Node)
		expectEqual(t, uint(0), uint(c.RowIdx))
		expectEqual(t, uint(1), uint(c.ColIdx))
		expectEqual(t, uint32(8), c.Cell.Codepoint())
	}
	// viewport (active) y=4 is global row 11
	{
		c := zigUnwrap(t, s.GetCell(pl5Viewport(5, 4)))
		expectEqual(t, uint(4), uint(c.RowIdx))
		expectEqual(t, uint32(12), c.Cell.Codepoint())
	}
	// pinned viewport at row 5
	*s.viewport_pin = pl5PinOf(t, s, pl5Screen(0, 5))
	s.viewport = PageListViewport{Tag: PageListViewportPin}
	{
		c := zigUnwrap(t, s.GetCell(pl5Viewport(0, 0)))
		expectEqual(t, pl5MnodeAt(s, 1), c.Node)
		expectEqual(t, uint(1), uint(c.RowIdx))
		expectEqual(t, uint32(6), c.Cell.Codepoint())
	}
}

// Zig: test "term51: getCell: out of range points are null"
func TestPl5_term51_getCell_out_of_range_points_are_null(t *testing.T) {
	testStdCapacity(t)
	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	defer s.Deinit()

	expect(t, !s.GetCell(pl5Screen(6, 0)).Valid, "s.getCell(.{ .screen = .{ .x = 6, .y = 0 } }) == null")
	expect(t, !s.GetCell(pl5Screen(0, 12)).Valid, "s.getCell(.{ .screen = .{ .x = 0, .y = 12 } }) == null")
	expect(t, !s.GetCell(pl5Active(0, 5)).Valid, "s.getCell(.{ .active = .{ .x = 0, .y = 5 } }) == null")
	expect(t, !s.GetCell(pl5Active(6, 0)).Valid, "s.getCell(.{ .active = .{ .x = 6, .y = 0 } }) == null")
	expect(t, !s.GetCell(pl5History(0, 12)).Valid, "s.getCell(.{ .history = .{ .x = 0, .y = 12 } }) == null")
	expect(t, !s.GetCell(pl5Viewport(0, 5)).Valid, "s.getCell(.{ .viewport = .{ .x = 0, .y = 5 } }) == null")
	expect(t, !s.GetCell(pl5Screen(math.MaxUint16, 0)).Valid, "s.getCell(.{ .screen = .{ .x = std.math.maxInt(size.CellCountInt), .y = 0 } }) == null")
	expect(t, !s.GetCell(pl5Screen(0, math.MaxUint32)).Valid, "s.getCell(.{ .screen = .{ .y = std.math.maxInt(u32) } }) == null")
}

// Zig: test "term51: getCell: returned pointers address the live cell and row"
func TestPl5_term51_getCell_returned_pointers_address_the_live_cell_and_row(t *testing.T) {
	testStdCapacity(t)
	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	defer s.Deinit()

	c := zigUnwrap(t, s.GetCell(pl5Screen(2, 5)))
	*c.Cell = PageCell{content_tag: PageCellContentTagCodepoint, content: PageCellContent{codepoint: 'Z'}}
	c.Row.wrap = true
	page := &pl5MnodeAt(s, 1).data
	expectEqual(t, uint32('Z'), page.GetRowAndCell(2, 1).cell.Codepoint())
	expect(t, page.GetRow(1).wrap, "page.getRow(1).wrap")

	// Dirty state is read through the returned handle.
	s.ClearDirty()
	c2 := zigUnwrap(t, s.GetCell(pl5Screen(0, 5)))
	expect(t, !c2.IsDirty(), "!c2.isDirty()")
	c2.Row.dirty = true
	expect(t, c2.IsDirty(), "c2.isDirty()")
	expect(t, zigUnwrap(t, s.GetCell(pl5Screen(3, 5))).IsDirty(), "s.getCell(.{ .screen = .{ .x = 3, .y = 5 } }).?.isDirty()")
	expect(t, !zigUnwrap(t, s.GetCell(pl5Screen(3, 6))).IsDirty(), "!s.getCell(.{ .screen = .{ .x = 3, .y = 6 } }).?.isDirty()")
}

// ---------------------------------------------------------------------------
// term51: additional tests (N).
// ---------------------------------------------------------------------------

// pl5NPages is term51NPages: build a PageList (no max size, so nothing is ever
// pruned) in which the first `pages` pages are all completely full
// (size.rows == capacity.rows).
func pl5NPages(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt, pages uint) *PageList {
	t.Helper()
	s := pl5Init(t, cols, rows, None[uint64]())
	n := uint(1)
	for {
		node := s.pages.last
		node.data.PauseIntegrityChecks(true)
		for node.data.size.rows < node.data.capacity.rows {
			expect(t, pl5Grow(t, s) == nil, "try s.grow() == null")
		}
		node.data.PauseIntegrityChecks(false)
		if n == pages {
			break
		}
		expect(t, pl5Grow(t, s) != nil, "try s.grow() != null")
		n++
	}
	return s
}

// Zig: test "term51: Pin.rowAndCell: returns the row and cell at the pin position"
func TestPl5_term51_Pin_rowAndCell_returns_the_row_and_cell_at_the_pin_position(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 5, 4, None[uint64]())
	defer s.Deinit()
	node := s.pages.first
	page := &node.data

	p := PageListPin{Node: node, Y: 2, X: 3}
	rac := p.RowAndCell()
	want := page.GetRowAndCell(3, 2)
	expect(t, rac.row == want.row, "rac.row == want.row")
	expect(t, rac.cell == want.cell, "rac.cell == want.cell")

	// Different x or y give different pointers.
	px := PageListPin{Node: node, Y: 2, X: 2}
	expect(t, px.RowAndCell().row == rac.row, "px.rowAndCell().row == rac.row")
	expect(t, px.RowAndCell().cell != rac.cell, "px.rowAndCell().cell != rac.cell")
	py := PageListPin{Node: node, Y: 1, X: 3}
	expect(t, py.RowAndCell().row != rac.row, "py.rowAndCell().row != rac.row")
	expect(t, py.RowAndCell().cell != rac.cell, "py.rowAndCell().cell != rac.cell")

	// Writes through the pointers are visible in the page.
	*rac.cell = PageCellInit('Z')
	rac.row.wrap = true
	expectEqual(t, uint32('Z'), page.GetRowAndCell(3, 2).cell.content.codepoint)
	expect(t, page.GetRowAndCell(0, 2).row.wrap, "page.getRowAndCell(0, 2).row.wrap")
	expect(t, !page.GetRowAndCell(0, 1).row.wrap, "!page.getRowAndCell(0, 1).row.wrap")
	expect(t, !page.GetRowAndCell(0, 3).row.wrap, "!page.getRowAndCell(0, 3).row.wrap")

	// Corners.
	tl := PageListPin{Node: node}
	expect(t, tl.RowAndCell().cell == page.GetRowAndCell(0, 0).cell, "tl.rowAndCell().cell == page.getRowAndCell(0, 0).cell")
	br := PageListPin{Node: node, Y: 3, X: 4}
	expect(t, br.RowAndCell().cell == page.GetRowAndCell(4, 3).cell, "br.rowAndCell().cell == page.getRowAndCell(4, 3).cell")
	expect(t, br.RowAndCell().row == page.GetRowAndCell(4, 3).row, "br.rowAndCell().row == page.getRowAndCell(4, 3).row")
}

// Zig: test "term51: Pin.rowAndCell: uses the page of the pin node"
func TestPl5_term51_Pin_rowAndCell_uses_the_page_of_the_pin_node(t *testing.T) {
	testStdCapacity(t)
	s := pl5NPages(t, 10, 3, 2)
	defer s.Deinit()
	first := s.pages.first
	last := s.pages.last
	expect(t, first != last, "first != last")

	*first.data.GetRowAndCell(1, 0).cell = PageCellInit('A')
	*last.data.GetRowAndCell(1, 0).cell = PageCellInit('B')

	pf := PageListPin{Node: first, X: 1}
	pl := PageListPin{Node: last, X: 1}
	expectEqual(t, uint32('A'), pf.RowAndCell().cell.content.codepoint)
	expectEqual(t, uint32('B'), pl.RowAndCell().cell.content.codepoint)
	expect(t, pf.RowAndCell().cell == first.data.GetRowAndCell(1, 0).cell, "pf.rowAndCell().cell == first.data.getRowAndCell(1, 0).cell")
	expect(t, pl.RowAndCell().cell == last.data.GetRowAndCell(1, 0).cell, "pl.rowAndCell().cell == last.data.getRowAndCell(1, 0).cell")

	// Last row of the (full) last page.
	bottom := PageListPin{Node: last, Y: last.data.size.rows - 1, X: 9}
	expect(t, bottom.RowAndCell().row == last.data.GetRowAndCell(9, uint(last.data.size.rows)-1).row, "bottom.rowAndCell().row == last.data.getRowAndCell(9, last.data.size.rows - 1).row")
}

// Zig: test "term51: Pin.cells: all left and right subsets"
func TestPl5_term51_Pin_cells_all_left_and_right_subsets(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 6, 3, None[uint64]())
	defer s.Deinit()
	node := s.pages.first
	page := &node.data
	for x := uint(0); x < 6; x++ {
		*page.GetRowAndCell(x, 1).cell = PageCellInit(uint32('a' + x))
	}
	// A different row must never leak in.
	for x := uint(0); x < 6; x++ {
		*page.GetRowAndCell(x, 0).cell = PageCellInit('Z')
	}

	base := page.GetRowAndCell(0, 1).cell
	p := PageListPin{Node: node, Y: 1, X: 2}

	all := p.Cells(PageListPinCellSubsetAll)
	expectEqual(t, 6, len(all))
	expect(t, &all[0] == base, "all.ptr == @as([*]pagepkg.Cell, @ptrCast(base))")
	expectEqual(t, uint32('a'), all[0].content.codepoint)
	expectEqual(t, uint32('f'), all[5].content.codepoint)

	// Left is inclusive of x: 0..=x
	l := p.Cells(PageListPinCellSubsetLeft)
	expectEqual(t, 3, len(l))
	expect(t, &l[0] == &all[0], "l.ptr == all.ptr")
	expectEqual(t, uint32('c'), l[2].content.codepoint)

	// Right is inclusive of x: x..cols
	r := p.Cells(PageListPinCellSubsetRight)
	expectEqual(t, 4, len(r))
	expect(t, &r[0] == &all[2], "r.ptr == all.ptr + 2")
	expectEqual(t, uint32('c'), r[0].content.codepoint)
	expectEqual(t, uint32('f'), r[3].content.codepoint)
}

// Zig: test "term51: Pin.cells: first and last column boundaries"
func TestPl5_term51_Pin_cells_first_and_last_column_boundaries(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 6, 3, None[uint64]())
	defer s.Deinit()
	node := s.pages.first

	first := PageListPin{Node: node, Y: 2, X: 0}
	expectEqual(t, 6, len(first.Cells(PageListPinCellSubsetAll)))
	expectEqual(t, 1, len(first.Cells(PageListPinCellSubsetLeft)))
	expectEqual(t, 6, len(first.Cells(PageListPinCellSubsetRight)))

	last := PageListPin{Node: node, Y: 2, X: 5}
	expectEqual(t, 6, len(last.Cells(PageListPinCellSubsetAll)))
	expectEqual(t, 6, len(last.Cells(PageListPinCellSubsetLeft)))
	expectEqual(t, 1, len(last.Cells(PageListPinCellSubsetRight)))
	expect(t, &last.Cells(PageListPinCellSubsetRight)[0] == &last.Cells(PageListPinCellSubsetAll)[5], "last.cells(.right).ptr == last.cells(.all).ptr + 5")
}

// Zig: test "term51: Pin.grapheme: null without graphemes and extra codepoints with"
func TestPl5_term51_Pin_grapheme_null_without_graphemes_and_extra_codepoints_with(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 5, 3, None[uint64]())
	defer s.Deinit()
	node := s.pages.first
	page := &node.data

	// Plain cell: no graphemes.
	plain := page.GetRowAndCell(0, 0)
	*plain.cell = PageCellInit('A')
	p0 := PageListPin{Node: node}
	expect(t, !p0.Grapheme(plain.cell).Valid, "p0.grapheme(plain.cell) == null")

	// Empty cell: no graphemes.
	empty := page.GetRowAndCell(1, 0)
	expect(t, !p0.Grapheme(empty.cell).Valid, "p0.grapheme(empty.cell) == null")

	// One extra codepoint.
	one := page.GetRowAndCell(2, 1)
	*one.cell = PageCellInit('e')
	zigTry(t, page.AppendGrapheme(one.row, one.cell, 0x0301))
	p1 := PageListPin{Node: node, Y: 1, X: 2}
	got1 := zigUnwrap(t, p1.Grapheme(one.cell))
	expectEqual(t, 1, len(got1))
	expectEqual(t, uint32(0x0301), got1[0])

	// Two extra codepoints, in order, and the first codepoint is not included.
	two := page.GetRowAndCell(3, 1)
	*two.cell = PageCellInit('o')
	zigTry(t, page.AppendGrapheme(two.row, two.cell, 0x0308))
	zigTry(t, page.AppendGrapheme(two.row, two.cell, 0x0301))
	got2 := zigUnwrap(t, p1.Grapheme(two.cell))
	expectEqual(t, 2, len(got2))
	expectEqual(t, uint32(0x0308), got2[0])
	expectEqual(t, uint32(0x0301), got2[1])

	// The cell with one codepoint is unchanged and a neighbour still has none.
	expectEqual(t, 1, len(zigUnwrap(t, p1.Grapheme(one.cell))))
	expect(t, !p1.Grapheme(page.GetRowAndCell(4, 1).cell).Valid, "p1.grapheme(page.getRowAndCell(4, 1).cell) == null")
}

// Zig: test "term51: Pin.grapheme: looks up in the page of the pin node"
func TestPl5_term51_Pin_grapheme_looks_up_in_the_page_of_the_pin_node(t *testing.T) {
	testStdCapacity(t)
	s := pl5NPages(t, 10, 3, 2)
	defer s.Deinit()
	last := s.pages.last
	rac := last.data.GetRowAndCell(4, 7)
	*rac.cell = PageCellInit('x')
	zigTry(t, last.data.AppendGrapheme(rac.row, rac.cell, 0x0300))

	p := PageListPin{Node: last, Y: 7, X: 4}
	got := zigUnwrap(t, p.Grapheme(rac.cell))
	expectEqual(t, 1, len(got))
	expectEqual(t, uint32(0x0300), got[0])
}

// Zig: test "term51: Pin.style: default style id gives the default style"
func TestPl5_term51_Pin_style_default_style_id_gives_the_default_style(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 5, 3, None[uint64]())
	defer s.Deinit()
	node := s.pages.first
	p := PageListPin{Node: node}
	cell := node.data.GetRowAndCell(0, 0).cell
	expectEqual(t, StyleDefaultID, cell.style_id)
	st := p.Style(cell)
	expect(t, st.Default(), "st.default()")
	expect(t, st.Eql(Style{}), "st.eql(.{})")
}

// Zig: test "term51: Pin.style: non-default style id returns the page style"
func TestPl5_term51_Pin_style_non_default_style_id_returns_the_page_style(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 5, 3, None[uint64]())
	defer s.Deinit()
	node := s.pages.first
	page := &node.data

	bold := Style{Flags: StyleFlags{Bold: true}}
	red := Style{FgColor: StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{R: 255, G: 1, B: 2}}}
	bold_id, err := page.styles.Add(bold)
	zigTry(t, err)
	red_id, err := page.styles.Add(red)
	zigTry(t, err)
	expect(t, bold_id != StyleDefaultID, "bold_id != stylepkg.default_id")
	expect(t, red_id != bold_id, "red_id != bold_id")

	a := page.GetRowAndCell(1, 1)
	a.row.styled = true
	*a.cell = PageCell{
		content_tag: PageCellContentTagCodepoint,
		content:     PageCellContent{codepoint: 'A'},
		style_id:    bold_id,
	}
	b := page.GetRowAndCell(2, 1)
	*b.cell = PageCell{
		content_tag: PageCellContentTagCodepoint,
		content:     PageCellContent{codepoint: 'B'},
		style_id:    red_id,
	}

	p := PageListPin{Node: node, Y: 1, X: 1}
	sa := p.Style(a.cell)
	expect(t, sa.Eql(bold), "sa.eql(bold)")
	expect(t, sa.Flags.Bold, "sa.flags.bold")
	expect(t, !sa.Default(), "!sa.default()")
	sb := p.Style(b.cell)
	expect(t, sb.Eql(red), "sb.eql(red)")
	expect(t, !sb.Flags.Bold, "!sb.flags.bold")

	// The lookup is by the cell, not the pin position.
	expect(t, p.Style(page.GetRowAndCell(0, 0).cell).Default(), "p.style(page.getRowAndCell(0, 0).cell).default()")
}

// Zig: test "term51: Pin.isDirty: page dirty row dirty or clean"
func TestPl5_term51_Pin_isDirty_page_dirty_row_dirty_or_clean(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 5, 4, None[uint64]())
	defer s.Deinit()
	node := s.pages.first
	page := &node.data
	s.ClearDirty()

	p0 := PageListPin{Node: node, Y: 0, X: 0}
	p2 := PageListPin{Node: node, Y: 2, X: 4}
	expect(t, !p0.IsDirty(), "!p0.isDirty()")
	expect(t, !p2.IsDirty(), "!p2.isDirty()")

	// A dirty row only affects pins on that row, at any x.
	page.GetRowAndCell(0, 2).row.dirty = true
	expect(t, !p0.IsDirty(), "!p0.isDirty()")
	expect(t, p2.IsDirty(), "p2.isDirty()")
	expect(t, (PageListPin{Node: node, Y: 2, X: 0}).IsDirty(), "(Pin{ .node = node, .y = 2, .x = 0 }).isDirty()")
	expect(t, !(PageListPin{Node: node, Y: 1, X: 4}).IsDirty(), "!(Pin{ .node = node, .y = 1, .x = 4 }).isDirty()")
	expect(t, !(PageListPin{Node: node, Y: 3, X: 4}).IsDirty(), "!(Pin{ .node = node, .y = 3, .x = 4 }).isDirty()")

	// A dirty page affects every pin.
	page.GetRowAndCell(0, 2).row.dirty = false
	page.dirty = true
	expect(t, p0.IsDirty(), "p0.isDirty()")
	expect(t, p2.IsDirty(), "p2.isDirty()")
	expect(t, (PageListPin{Node: node, Y: 3, X: 1}).IsDirty(), "(Pin{ .node = node, .y = 3, .x = 1 }).isDirty()")
}

// ---------------------------------------------------------------------------
// Go-only panic tests: assert / unreachable / overflow paths of the functions
// above that Zig's tests avoid (a Zig test cannot test a panic). The lists are
// left in a broken state, so these tests never call Deinit.
// ---------------------------------------------------------------------------

// Go-only: panic test: viewportRowOffset: viewport pin on a node that is not in the list: unreachable
func TestPl5_panic_viewportRowOffset_pin_node_not_in_list(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	other := pl5Init(t, 80, 24, None[uint64]())
	s.viewport = PageListViewport{Tag: PageListViewportPin}
	*s.viewport_pin = PageListPin{Node: other.pages.first}
	s.viewport_pin_row_offset = None[uint]()
	pl5ExpectPanic(t, func() { s.viewportRowOffset() })
}

// Go-only: panic test: viewportRowOffset: viewport pin row is not below the page's size.rows: assert(n.data.size.rows > self.viewport_pin.y)
func TestPl5_panic_viewportRowOffset_pin_row_beyond_the_page(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	s.viewport = PageListViewport{Tag: PageListViewportPin}
	*s.viewport_pin = PageListPin{Node: s.pages.first, Y: 24}
	s.viewport_pin_row_offset = None[uint]()
	pl5ExpectPanic(t, func() { s.viewportRowOffset() })
}

// Go-only: panic test: viewportRowOffset: a stale cached offset fails the integrity assertion
func TestPl5_panic_viewportRowOffset_stale_cached_offset(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	pl5LGrowRows(t, s, 10)
	*s.viewport_pin = pl5PinOf(t, s, pl5Screen(0, 5))
	s.viewport = PageListViewport{Tag: PageListViewportPin}
	s.viewport_pin_row_offset = Some[uint](3)
	pl5ExpectPanic(t, func() { s.viewportRowOffset() })
}

// Go-only: panic test: fixupViewport: top viewport with more active rows than total rows: getTopLeft(.active) unreachable
func TestPl5_panic_fixupViewport_top_with_too_few_rows(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	s.viewport = PageListViewport{Tag: PageListViewportTop}
	s.rows = 100
	pl5ExpectPanic(t, func() { s.fixupViewport(0) })
}

// Go-only: panic test: grow: total_rows that does not match the pages fails the integrity assertion
func TestPl5_panic_grow_total_rows_mismatch(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	s.total_rows++
	pl5ExpectPanic(t, func() { s.Grow() })
}

// Go-only: panic test: increaseCapacity: total_rows that does not match the pages fails the integrity assertion
func TestPl5_panic_increaseCapacity_total_rows_mismatch(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	s.total_rows++
	pl5ExpectPanic(t, func() {
		s.IncreaseCapacity(s.pages.first, Some(PageListIncreaseCapacityStyles))
	})
}

// Go-only: panic test: createPage: layout total_size above size.max_page_size: assert(layout.total_size <= size.max_page_size)
func TestPl5_panic_createPage_capacity_above_the_max_page_size(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	cap := PageCapacityInit(math.MaxUint16, math.MaxUint16)
	expect(t, PageLayoutOf(cap).total_size > SizeMaxPageSize, "Page.layout(cap).total_size > size.max_page_size")
	pl5ExpectPanic(t, func() { s.createPage(cap) })
}

// Go-only: panic test: destroyNodeExt: total_size smaller than the page's memory length: integer overflow in `v.* -= page.memory.len`
func TestPl5_panic_destroyNodeExt_total_size_underflow(t *testing.T) {
	testStdCapacity(t)
	pool, err := PageListMemoryPoolInit(1)
	zigTry(t, err)
	serial := uint64(0)
	node, err := pageListCreatePageExt(&pool, pageListInitialCapacity(80), &serial, nil)
	zigTry(t, err)
	total := uint64(0)
	pl5ExpectPanic(t, func() { pageListDestroyNodeExt(&pool, node, &total) })
}

// Go-only: panic test: eraseRow: point out of range: `self.pin(pt).?` of null
func TestPl5_panic_eraseRow_point_out_of_range(t *testing.T) {
	testStdCapacity(t)
	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	pl5ExpectPanic(t, func() { s.EraseRow(pl5Screen(0, 12)) })
}

// Go-only: panic test: eraseRow: total_rows that does not match the pages fails the integrity assertion
func TestPl5_panic_eraseRow_total_rows_mismatch(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	testStdCapacity(t)
	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	s.total_rows++
	pl5ExpectPanic(t, func() { s.EraseRow(pl5Screen(0, 0)) })
}

// Go-only: panic test: eraseRowBounded: point out of range: `self.pin(pt).?` of null
func TestPl5_panic_eraseRowBounded_point_out_of_range(t *testing.T) {
	testStdCapacity(t)
	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	pl5ExpectPanic(t, func() { s.EraseRowBounded(pl5Screen(0, 12), 1) })
}

// Go-only: panic test: eraseRowBounded: cached viewport offset of 0 with the viewport pin inside the erased range at a row above 0: integer overflow in `v.* -= 1`
func TestPl5_panic_eraseRowBounded_cached_offset_underflow(t *testing.T) {
	testStdCapacity(t)
	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	*s.viewport_pin = pl5PinOf(t, s, pl5Screen(0, 2))
	s.viewport = PageListViewport{Tag: PageListViewportPin}
	s.viewport_pin_row_offset = Some[uint](0)
	pl5ExpectPanic(t, func() { s.EraseRowBounded(pl5Screen(0, 1), 2) })
}

// Go-only: panic test: eraseRowBounded: total_rows that does not match the pages fails the integrity assertion
func TestPl5_panic_eraseRowBounded_total_rows_mismatch(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	testStdCapacity(t)
	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	s.total_rows++
	pl5ExpectPanic(t, func() { s.EraseRowBounded(pl5Screen(0, 0), 1) })
}

// Go-only: panic test: eraseRows: a partial chunk that does not start at row 0: assert(chunk.start == 0)
func TestPl5_panic_eraseRows_chunk_start_not_zero(t *testing.T) {
	testStdCapacity(t)
	s := pl5Mbuild(t, 6, 5, []uint{5})
	pl5ExpectPanic(t, func() { s.EraseRows(pl5Screen(0, 1), Some(pl5Screen(0, 2))) })
}

// Go-only: panic test: eraseRows: total_rows that does not match the pages fails the integrity assertion
func TestPl5_panic_eraseRows_total_rows_mismatch(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	testStdCapacity(t)
	s := pl5Mbuild(t, 6, 5, []uint{4, 3, 5})
	s.total_rows++
	pl5ExpectPanic(t, func() { s.EraseRows(pl5Screen(0, 0), Some(pl5Screen(0, 0))) })
}

// Go-only: panic test: erasePage: the only page of the list: assert(node.next != null or node.prev != null)
func TestPl5_panic_erasePage_only_page(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	pl5ExpectPanic(t, func() { s.erasePage(s.pages.first) })
}

// Go-only: panic test: trackPin: pin on a node of another list: assert(self.pinIsValid(p))
func TestPl5_panic_trackPin_pin_of_another_list(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	other := pl5Init(t, 80, 24, None[uint64]())
	pl5ExpectPanic(t, func() { s.TrackPin(PageListPin{Node: other.pages.first}) })
}

// Go-only: panic test: trackPin: pin row beyond the page's size: assert(self.pinIsValid(p))
func TestPl5_panic_trackPin_pin_beyond_the_page(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	pl5ExpectPanic(t, func() { s.TrackPin(PageListPin{Node: s.pages.first, Y: 24}) })
}

// Go-only: panic test: untrackPin: the viewport pin: assert(p != self.viewport_pin)
func TestPl5_panic_untrackPin_viewport_pin(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	pl5ExpectPanic(t, func() { s.UntrackPin(s.viewport_pin) })
}

// Go-only: panic test: pin: active point with more active rows than total rows: getTopLeft(.active) unreachable
func TestPl5_panic_pin_active_with_too_few_rows(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	s.rows = 100
	pl5ExpectPanic(t, func() { s.Pin(pl5Active(0, 0)) })
}

// Go-only: panic test: getCell: column beyond the page's columns: assert(x < self.size.cols) in getRowAndCell
func TestPl5_panic_getCell_column_beyond_the_page(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	s.cols = 100
	pl5ExpectPanic(t, func() { s.GetCell(pl5Screen(90, 0)) })
}

// Go-only: panic test: Pin.rowAndCell: row beyond the page's size: assert(y < self.size.rows)
func TestPl5_panic_Pin_rowAndCell_row_beyond_the_page(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	p := PageListPin{Node: s.pages.first, Y: 24}
	pl5ExpectPanic(t, func() { p.RowAndCell() })
}

// Go-only: panic test: Pin.rowAndCell: column beyond the page's columns: assert(x < self.size.cols)
func TestPl5_panic_Pin_rowAndCell_column_beyond_the_page(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	p := PageListPin{Node: s.pages.first, X: 80}
	pl5ExpectPanic(t, func() { p.RowAndCell() })
}

// Go-only: panic test: Pin.cells: row beyond the page's size: assert(y < self.size.rows) in rowAndCell
func TestPl5_panic_Pin_cells_row_beyond_the_page(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	p := PageListPin{Node: s.pages.first, Y: 24}
	pl5ExpectPanic(t, func() { p.Cells(PageListPinCellSubsetAll) })
}

// Go-only: panic test: Pin.isDirty: row beyond the page's size: assert(y < self.size.rows) in rowAndCell
func TestPl5_panic_Pin_isDirty_row_beyond_the_page(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	s.ClearDirty()
	p := PageListPin{Node: s.pages.first, Y: 24}
	pl5ExpectPanic(t, func() { p.IsDirty() })
}

// Go-only: panic test: Pin.markDirty: row beyond the page's size: assert(y < self.size.rows) in rowAndCell
func TestPl5_panic_Pin_markDirty_row_beyond_the_page(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 80, 24, None[uint64]())
	p := PageListPin{Node: s.pages.first, Y: 24}
	pl5ExpectPanic(t, func() { p.MarkDirty() })
}

// Go-only: panic test: Pin.style: style id with no reference in the page's style set: assert(item.meta.ref > 0)
func TestPl5_panic_Pin_style_unused_style_id(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 5, 3, None[uint64]())
	node := s.pages.first
	cell := node.data.GetRowAndCell(0, 0).cell
	cell.style_id = 1
	p := PageListPin{Node: node}
	pl5ExpectPanic(t, func() { p.Style(cell) })
}

// Go-only: panic test: Pin.style: style id beyond the page's style set: assert(id < self.layout.cap)
func TestPl5_panic_Pin_style_style_id_beyond_the_set(t *testing.T) {
	testStdCapacity(t)
	s := pl5Init(t, 5, 3, None[uint64]())
	node := s.pages.first
	cell := node.data.GetRowAndCell(0, 0).cell
	cell.style_id = math.MaxUint16
	p := PageListPin{Node: node}
	pl5ExpectPanic(t, func() { p.Style(cell) })
}
