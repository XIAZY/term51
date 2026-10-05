// Tests translated one for one from src/terminal/search/pagelist.zig (Ghostty
// v1.3.1 plus the term51 additional tests). Each Go test carries the exact
// Zig title in a "// Zig: test" comment. The helpers (saTerminal, saFill,
// saWrite, saCap, saWrapLast, saMetaNode, saHasPin, saExpectHighlight, ...)
// are shared with the other search tests and are in search_test.go; see the
// translation notes there. Helper of this file: saGrowNull
// (`try testing.expect(try p.grow() == null)`).
//
// Translation notes:
//   - `PageList.init(alloc, 80, 24, 0)` is `PageListInit(80, 24, Some(uint64(0)))`.
//   - `try p.grow()` returns the new node or nil (Zig's null).
//   - `.{ .sel = .init(a, b, false) }` for selectionString is
//     screenSelectionStringDefault(SelectionInit(a, b, false)) (trim = true).
//
// Not translated (scripts/not-translated.txt): the allocation failure
// test (FailingAllocator).
//
// Go-only panic tests: PageListSearch.init on a start page with zero rows or
// zero columns (`size.rows - 1` and `size.cols - 1` overflow in Zig).

package terminal

import "testing"

// saGrowNull is `try testing.expect(try p.grow() == null)`.
func saGrowNull(t *testing.T, p *PageList) {
	t.Helper()
	n, err := p.Grow()
	zigTry(t, err)
	expect(t, n == nil, "try p.grow() == null")
}

// Zig: test "simple search"
func TestSaPagelist_simple_search(t *testing.T) {
	testStdCapacity(t)
	term, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz\r\nBuzz\r\nFizz\r\nBang")))

	search, err := SearchPagelistInit(
		[]uint8("Fizz"),
		&term.screens.active.pages,
		term.screens.active.pages.pages.last,
	)
	zigTry(t, err)
	defer search.Deinit()

	{
		h := zigUnwrap(t, search.Next())
		sel := h.Untracked()
		expectEqualPoint(t, saActivePoint(0, 2), zigUnwrap(t, term.screens.active.pages.PointFromPin(PointTagActive, sel.start)))
		expectEqualPoint(t, saActivePoint(3, 2), zigUnwrap(t, term.screens.active.pages.PointFromPin(PointTagActive, sel.end)))
	}
	{
		h := zigUnwrap(t, search.Next())
		sel := h.Untracked()
		expectEqualPoint(t, saActivePoint(0, 0), zigUnwrap(t, term.screens.active.pages.PointFromPin(PointTagActive, sel.start)))
		expectEqualPoint(t, saActivePoint(3, 0), zigUnwrap(t, term.screens.active.pages.PointFromPin(PointTagActive, sel.end)))
	}
	expect(t, !search.Next().Valid, "search.next() == null")

	// We should not be able to feed since we have one page
	fed, err := search.Feed()
	zigTry(t, err)
	expect(t, !fed, "!try search.feed()")
}

// Zig: test "feed multiple pages with matches"
func TestSaPagelist_feed_multiple_pages_with_matches(t *testing.T) {
	testStdCapacity(t)
	term, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Fill up first page
	first_page_rows := uint(term.screens.active.pages.pages.first.data.capacity.rows)
	for i := uint(0); i < first_page_rows-1; i++ {
		zigTry(t, s.NextSlice([]uint8("\r\n")))
	}
	zigTry(t, s.NextSlice([]uint8("Fizz")))
	expect(t, term.screens.active.pages.pages.first == term.screens.active.pages.pages.last, "t.screens.active.pages.pages.first == t.screens.active.pages.pages.last")

	// Create second page
	zigTry(t, s.NextSlice([]uint8("\r\n")))
	expect(t, term.screens.active.pages.pages.first != term.screens.active.pages.pages.last, "t.screens.active.pages.pages.first != t.screens.active.pages.pages.last")
	zigTry(t, s.NextSlice([]uint8("Buzz\r\nFizz")))

	search, err := SearchPagelistInit(
		[]uint8("Fizz"),
		&term.screens.active.pages,
		term.screens.active.pages.pages.last,
	)
	zigTry(t, err)
	defer search.Deinit()

	// First match on the last page
	sel1 := search.Next()
	expect(t, sel1.Valid, "sel1 != null")
	expect(t, !search.Next().Valid, "search.next() == null")

	// Feed should succeed and load the first page
	fed, err := search.Feed()
	zigTry(t, err)
	expect(t, fed, "try search.feed()")

	// Now we should find the match on the first page
	sel2 := search.Next()
	expect(t, sel2.Valid, "sel2 != null")
	expect(t, !search.Next().Valid, "search.next() == null")

	// No more pages to feed
	fed, err = search.Feed()
	zigTry(t, err)
	expect(t, !fed, "!try search.feed()")
}

// Zig: test "feed multiple pages no matches"
func TestSaPagelist_feed_multiple_pages_no_matches(t *testing.T) {
	testStdCapacity(t)
	term, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Fill up first page
	first_page_rows := uint(term.screens.active.pages.pages.first.data.capacity.rows)
	for i := uint(0); i < first_page_rows-1; i++ {
		zigTry(t, s.NextSlice([]uint8("\r\n")))
	}
	zigTry(t, s.NextSlice([]uint8("Hello")))

	// Create second page
	zigTry(t, s.NextSlice([]uint8("\r\n")))
	expect(t, term.screens.active.pages.pages.first != term.screens.active.pages.pages.last, "t.screens.active.pages.pages.first != t.screens.active.pages.pages.last")
	zigTry(t, s.NextSlice([]uint8("World")))

	search, err := SearchPagelistInit(
		[]uint8("Nope"),
		&term.screens.active.pages,
		term.screens.active.pages.pages.last,
	)
	zigTry(t, err)
	defer search.Deinit()

	// No matches on last page
	expect(t, !search.Next().Valid, "search.next() == null")

	// Feed first page
	fed, err := search.Feed()
	zigTry(t, err)
	expect(t, fed, "try search.feed()")

	// Still no matches
	expect(t, !search.Next().Valid, "search.next() == null")

	// No more pages
	fed, err = search.Feed()
	zigTry(t, err)
	expect(t, !fed, "!try search.feed()")
}

// Zig: test "feed iteratively through multiple matches"
func TestSaPagelist_feed_iteratively_through_multiple_matches(t *testing.T) {
	testStdCapacity(t)
	term, err := TerminalInit(terminalOptionsDefault(80, 24))
	zigTry(t, err)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	first_page_rows := uint(term.screens.active.pages.pages.first.data.capacity.rows)

	// Fill first page with a match at the end
	for i := uint(0); i < first_page_rows-1; i++ {
		zigTry(t, s.NextSlice([]uint8("\r\n")))
	}
	zigTry(t, s.NextSlice([]uint8("Page1Test")))
	expect(t, term.screens.active.pages.pages.first == term.screens.active.pages.pages.last, "t.screens.active.pages.pages.first == t.screens.active.pages.pages.last")

	// Create second page with a match
	zigTry(t, s.NextSlice([]uint8("\r\n")))
	expect(t, term.screens.active.pages.pages.first != term.screens.active.pages.pages.last, "t.screens.active.pages.pages.first != t.screens.active.pages.pages.last")
	zigTry(t, s.NextSlice([]uint8("Page2Test")))

	search, err := SearchPagelistInit(
		[]uint8("Test"),
		&term.screens.active.pages,
		term.screens.active.pages.pages.last,
	)
	zigTry(t, err)
	defer search.Deinit()

	// Match on page 2
	expect(t, search.Next().Valid, "search.next() != null")
	expect(t, !search.Next().Valid, "search.next() == null")

	// Feed page 1
	fed, err := search.Feed()
	zigTry(t, err)
	expect(t, fed, "try search.feed()")
	expect(t, search.Next().Valid, "search.next() != null")
	expect(t, !search.Next().Valid, "search.next() == null")

	// No more pages
	fed, err = search.Feed()
	zigTry(t, err)
	expect(t, !fed, "!try search.feed()")
}

// Zig: test "feed with match spanning page boundary"
func TestSaPagelist_feed_with_match_spanning_page_boundary(t *testing.T) {
	testStdCapacity(t)
	term, err := TerminalInit(terminalOptionsDefault(80, 24))
	zigTry(t, err)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	first_page_rows := uint(term.screens.active.pages.pages.first.data.capacity.rows)

	// Fill first page ending with "Te"
	for i := uint(0); i < first_page_rows-1; i++ {
		zigTry(t, s.NextSlice([]uint8("\r\n")))
	}
	for i := uint(0); i < uint(term.screens.active.pages.cols)-2; i++ {
		zigTry(t, s.NextSlice([]uint8("x")))
	}
	zigTry(t, s.NextSlice([]uint8("Te")))
	expect(t, term.screens.active.pages.pages.first == term.screens.active.pages.pages.last, "t.screens.active.pages.pages.first == t.screens.active.pages.pages.last")

	// Second page starts with "st"
	zigTry(t, s.NextSlice([]uint8("st")))
	expect(t, term.screens.active.pages.pages.first != term.screens.active.pages.pages.last, "t.screens.active.pages.pages.first != t.screens.active.pages.pages.last")

	search, err := SearchPagelistInit(
		[]uint8("Test"),
		&term.screens.active.pages,
		term.screens.active.pages.pages.last,
	)
	zigTry(t, err)
	defer search.Deinit()

	// No complete match on last page alone (only has "st")
	expect(t, !search.Next().Valid, "search.next() == null")

	// Feed first page - this should give us enough data to find "Test"
	fed, err := search.Feed()
	zigTry(t, err)
	expect(t, fed, "try search.feed()")

	// Should find the spanning match
	h := zigUnwrap(t, search.Next())
	sel := h.Untracked()
	expect(t, sel.start.node != sel.end.node, "sel.start.node != sel.end.node")
	{
		str, err := term.screens.active.SelectionString(screenSelectionStringDefault(SelectionInit(sel.start, sel.end, false)))
		zigTry(t, err)
		// Zig-only: defer alloc.free(str)
		expectEqualStrings(t, string(str), "Test")
	}

	// No more matches
	expect(t, !search.Next().Valid, "search.next() == null")

	// No more pages
	fed, err = search.Feed()
	zigTry(t, err)
	expect(t, !fed, "!try search.feed()")
}

// Zig: test "feed with match spanning page boundary with newline"
func TestSaPagelist_feed_with_match_spanning_page_boundary_with_newline(t *testing.T) {
	testStdCapacity(t)
	term, err := TerminalInit(terminalOptionsDefault(80, 24))
	zigTry(t, err)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	first_page_rows := uint(term.screens.active.pages.pages.first.data.capacity.rows)

	// Fill first page ending with "Te"
	for i := uint(0); i < first_page_rows-1; i++ {
		zigTry(t, s.NextSlice([]uint8("\r\n")))
	}
	for i := uint(0); i < uint(term.screens.active.pages.cols)-2; i++ {
		zigTry(t, s.NextSlice([]uint8("x")))
	}
	zigTry(t, s.NextSlice([]uint8("Te")))
	expect(t, term.screens.active.pages.pages.first == term.screens.active.pages.pages.last, "t.screens.active.pages.pages.first == t.screens.active.pages.pages.last")

	// Second page starts with "st"
	zigTry(t, s.NextSlice([]uint8("\r\n")))
	expect(t, term.screens.active.pages.pages.first != term.screens.active.pages.pages.last, "t.screens.active.pages.pages.first != t.screens.active.pages.pages.last")
	zigTry(t, s.NextSlice([]uint8("st")))

	search, err := SearchPagelistInit(
		[]uint8("Test"),
		&term.screens.active.pages,
		term.screens.active.pages.pages.last,
	)
	zigTry(t, err)
	defer search.Deinit()

	// Should not find any matches since we broke with an explicit newline.
	expect(t, !search.Next().Valid, "search.next() == null")
	fed, err := search.Feed()
	zigTry(t, err)
	expect(t, fed, "try search.feed()")
	expect(t, !search.Next().Valid, "search.next() == null")
	fed, err = search.Feed()
	zigTry(t, err)
	expect(t, !fed, "!try search.feed()")
}

// Zig: test "feed with pruned page"
func TestSaPagelist_feed_with_pruned_page(t *testing.T) {
	testStdCapacity(t)
	// Zero here forces minimum max size to effectively two pages.
	p, err := PageListInit(80, 24, Some(uint64(0)))
	zigTry(t, err)
	defer p.Deinit()

	// Grow to capacity
	page1_node := p.pages.last
	page1 := page1_node.data
	for i := uint(0); i < uint(page1.capacity.rows-page1.size.rows); i++ {
		saGrowNull(t, &p)
	}

	// Grow and allocate one more page. Then fill that page up.
	page2_node, err := p.Grow()
	zigTry(t, err)
	expect(t, page2_node != nil, "(try p.grow()).?")
	page2 := page2_node.data
	for i := uint(0); i < uint(page2.capacity.rows-page2.size.rows); i++ {
		saGrowNull(t, &p)
	}

	// Setup search and feed until we can't
	search, err := SearchPagelistInit(
		[]uint8("Test"),
		&p,
		p.pages.last,
	)
	zigTry(t, err)
	defer search.Deinit()
	fed, err := search.Feed()
	zigTry(t, err)
	expect(t, fed, "try search.feed()")
	fed, err = search.Feed()
	zigTry(t, err)
	expect(t, !fed, "!try search.feed()")

	// Next should create a new page, but it should reuse our first
	// page since we're at max size.
	new_, err := p.Grow()
	zigTry(t, err)
	expect(t, new_ != nil, "(try p.grow()).?")
	expect(t, p.pages.last == new_, "p.pages.last.? == new")

	// Our first should now be page2 and our last should be page1
	expectEqual(t, page2_node, p.pages.first)
	expectEqual(t, page1_node, p.pages.last)

	// Feed should still do nothing
	fed, err = search.Feed()
	zigTry(t, err)
	expect(t, !fed, "!try search.feed()")
}

// Zig: test "term51: PageListSearch.init: tracks a pin at the last cell of the start node and feeds it"
func TestSaPagelist_term51_PageListSearch_init_tracks_a_pin_at_the_last_cell_of_the_start_node_and_feeds_it(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, cap_+2, nil)
	p1 := pages.pages.first
	p2 := pages.pages.last
	expectEqual(t, uint(2), uint(p2.data.size.rows))

	// Start at the last page (two rows).
	before := pages.CountTrackedPins()
	search, err := SearchPagelistInit([]uint8("Fizz"), pages, p2)
	zigTry(t, err)
	expectEqual(t, before+1, pages.CountTrackedPins())
	expect(t, saHasPin(pages, search.pin), "term51ALHasPin(pages, search.pin)")
	expect(t, search.list == pages, "search.list == pages")
	expect(t, search.pin.node == p2, "search.pin.node == p2")
	expectEqual(t, SizeCellCountInt(1), search.pin.y)
	expectEqual(t, SizeCellCountInt(9), search.pin.x)
	expect(t, !search.pin.garbage, "!search.pin.garbage")
	expectEqual(t, uint(1), search.window.meta.Len())
	expect(t, saMetaNode(&search.window, 0) == p2, "term51ALMetaNode(&search.window, 0) == p2")
	search.Deinit()
	expectEqual(t, before, pages.CountTrackedPins())

	// Start at a page that is not the last one: pin at its last row.
	search2, err := SearchPagelistInit([]uint8("Fizz"), pages, p1)
	zigTry(t, err)
	defer search2.Deinit()
	expect(t, search2.pin.node == p1, "search2.pin.node == p1")
	expectEqual(t, SizeCellCountInt(cap_-1), search2.pin.y)
	expectEqual(t, SizeCellCountInt(9), search2.pin.x)
	expectEqual(t, uint(1), search2.window.meta.Len())
	expect(t, saMetaNode(&search2.window, 0) == p1, "term51ALMetaNode(&search2.window, 0) == p1")
}

// Zig: test "term51: PageListSearch.init: reverse window with a reversed copy of the needle"
func TestSaPagelist_term51_PageListSearch_init_reverse_window_with_a_reversed_copy_of_the_needle(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	buf := []uint8("Fizz")
	search, err := SearchPagelistInit(buf, &term.screens.active.pages, term.screens.active.pages.pages.last)
	zigTry(t, err)
	defer search.Deinit()
	expectEqual(t, true, search.window.direction == searchSlidingWindowDirectionReverse)
	expectEqualStrings(t, "zziF", string(search.window.needle))
	expect(t, &search.window.needle[0] != &buf[0], "search.window.needle.ptr != @as([*]const u8, &buf)")
	expectEqualStrings(t, "Fizz", string(buf))
	// A blank page contributes its trailing newline.
	expectEqual(t, uint(1), search.window.data.Len())
}

// Zig: test "term51: PageListSearch.init: start node that adds no bytes leaves the window empty"
func TestSaPagelist_term51_PageListSearch_init_start_node_that_adds_no_bytes_leaves_the_window_empty(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	p := pages.pages.first
	saWrapLast(p)
	search, err := SearchPagelistInit([]uint8("Fizz"), pages, p)
	zigTry(t, err)
	defer search.Deinit()
	expectEqual(t, uint(0), search.window.meta.Len())
	expectEqual(t, uint(0), search.window.data.Len())
	expect(t, search.pin.node == p, "search.pin.node == p")
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: PageListSearch.init: empty needle is accepted"
func TestSaPagelist_term51_PageListSearch_init_empty_needle_is_accepted(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	search, err := SearchPagelistInit([]uint8(""), &term.screens.active.pages, term.screens.active.pages.pages.last)
	zigTry(t, err)
	defer search.Deinit()
	expectEqual(t, uint(0), uint(len(search.window.needle)))
	expectEqual(t, uint(1), search.window.meta.Len())
}

// Zig: test "term51: PageListSearch.deinit: untracks the pin wherever it moved"
func TestSaPagelist_term51_PageListSearch_deinit_untracks_the_pin_wherever_it_moved(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, cap_+2, []saMark{{y: 0, text: "Fizz"}})
	before := pages.CountTrackedPins()

	fresh, err := SearchPagelistInit([]uint8("Fizz"), pages, pages.pages.last)
	zigTry(t, err)
	expectEqual(t, before+1, pages.CountTrackedPins())
	fresh.Deinit()
	expectEqual(t, before, pages.CountTrackedPins())

	fed_, err := SearchPagelistInit([]uint8("Fizz"), pages, pages.pages.last)
	zigTry(t, err)
	fed, err := fed_.Feed()
	zigTry(t, err)
	expect(t, fed, "try fed.feed()")
	_ = zigUnwrap(t, fed_.Next())
	fed_.Deinit()
	expectEqual(t, before, pages.CountTrackedPins())

	// The viewport pin that PageList always tracks is not affected.
	expect(t, saHasPin(pages, pages.viewport_pin), "term51ALHasPin(pages, pages.viewport_pin)")
}

// Zig: test "term51: PageListSearch.deinit: works after the pin became garbage"
func TestSaPagelist_term51_PageListSearch_deinit_works_after_the_pin_became_garbage(t *testing.T) {
	testStdCapacity(t)
	p, err := PageListInit(80, 24, Some(uint64(0)))
	zigTry(t, err)
	defer p.Deinit()
	first := p.pages.first
	// Zig: for (0..n) evaluates n once.
	n_first := uint(first.data.capacity.rows - first.data.size.rows)
	for i := uint(0); i < n_first; i++ {
		saGrowNull(t, &p)
	}
	second, err := p.Grow()
	zigTry(t, err)
	expect(t, second != nil, "(try p.grow()).?")
	// Zig: for (0..n) evaluates n once.
	n_second := uint(second.data.capacity.rows - second.data.size.rows)
	for i := uint(0); i < n_second; i++ {
		saGrowNull(t, &p)
	}
	before := p.CountTrackedPins()

	search, err := SearchPagelistInit([]uint8("Fizz"), &p, first)
	zigTry(t, err)
	expectEqual(t, before+1, p.CountTrackedPins())
	// Pruning the first page marks the pin garbage.
	pruned, err := p.Grow()
	zigTry(t, err)
	expect(t, pruned != nil, "(try p.grow()).?")
	expect(t, search.pin.garbage, "search.pin.garbage")
	search.Deinit()
	expectEqual(t, before, p.CountTrackedPins())
}

// Zig: test "term51: PageListSearch.next: returns matches bottom to top with exact chunks"
func TestSaPagelist_term51_PageListSearch_next_returns_matches_bottom_to_top_with_exact_chunks(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	saWrite(t, term, "Fizz\r\nBuzz\r\nfizz\r\nBang")
	p := pages.pages.first

	search, err := SearchPagelistInit([]uint8("Fizz"), pages, p)
	zigTry(t, err)
	defer search.Deinit()
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p, start: 2, end: 3}})
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p, start: 0, end: 1}})
	expect(t, !search.Next().Valid, "search.next() == null")
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: PageListSearch.next: needle in the first cell and in the last cell"
func TestSaPagelist_term51_PageListSearch_next_needle_in_the_first_cell_and_in_the_last_cell(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	p := pages.pages.first
	saWrite(t, term, "Ab\x1b[10;9HAb")

	search, err := SearchPagelistInit([]uint8("ab"), pages, p)
	zigTry(t, err)
	defer search.Deinit()
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 8, 9, []saChunk{{node: p, start: 9, end: 10}})
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 1, []saChunk{{node: p, start: 0, end: 1}})
	expect(t, !search.Next().Valid, "search.next() == null")

	one, err := SearchPagelistInit([]uint8("B"), pages, p)
	zigTry(t, err)
	defer one.Deinit()
	saExpectHighlight(t, zigUnwrap(t, one.Next()), 9, 9, []saChunk{{node: p, start: 9, end: 10}})
	saExpectHighlight(t, zigUnwrap(t, one.Next()), 1, 1, []saChunk{{node: p, start: 0, end: 1}})
	expect(t, !one.Next().Valid, "one.next() == null")
	// One-character needle drops the buffered data once nothing matches.
	expectEqual(t, uint(0), one.window.data.Len())
}

// Zig: test "term51: PageListSearch.next: overlapping matches come last to first and soft wraps are one chunk"
func TestSaPagelist_term51_PageListSearch_next_overlapping_matches_come_last_to_first_and_soft_wraps_are_one_chunk(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	p := pages.pages.first
	// Row 0: "aaaa"; rows 1-2: soft wrapped "abcdefgFizz".
	saWrite(t, term, "aaaa\r\nabcdefgFizz")

	aa, err := SearchPagelistInit([]uint8("aa"), pages, p)
	zigTry(t, err)
	defer aa.Deinit()
	saExpectHighlight(t, zigUnwrap(t, aa.Next()), 2, 3, []saChunk{{node: p, start: 0, end: 1}})
	saExpectHighlight(t, zigUnwrap(t, aa.Next()), 1, 2, []saChunk{{node: p, start: 0, end: 1}})
	saExpectHighlight(t, zigUnwrap(t, aa.Next()), 0, 1, []saChunk{{node: p, start: 0, end: 1}})
	expect(t, !aa.Next().Valid, "aa.next() == null")

	wrapped, err := SearchPagelistInit([]uint8("Fizz"), pages, p)
	zigTry(t, err)
	defer wrapped.Deinit()
	saExpectHighlight(t, zigUnwrap(t, wrapped.Next()), 7, 0, []saChunk{{node: p, start: 1, end: 3}})
	expect(t, !wrapped.Next().Valid, "wrapped.next() == null")

	// A hard newline splits the needle.
	split, err := SearchPagelistInit([]uint8("gFizzz"), pages, p)
	zigTry(t, err)
	defer split.Deinit()
	expect(t, !split.Next().Valid, "split.next() == null")
}

// Zig: test "term51: PageListSearch.next: a match across two pages is found after feed"
func TestSaPagelist_term51_PageListSearch_next_a_match_across_two_pages_is_found_after_feed(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, cap_+1, []saMark{{y: cap_ - 1, text: "xxxxxxxxTestZ"}})
	p1 := pages.pages.first
	p2 := pages.pages.last

	search, err := SearchPagelistInit([]uint8("Test"), pages, p2)
	zigTry(t, err)
	defer search.Deinit()
	// Only "st" is loaded.
	expect(t, !search.Next().Valid, "search.next() == null")
	fed, err := search.Feed()
	zigTry(t, err)
	expect(t, fed, "try search.feed()")
	// The window reverses nodes, starts and ends of a multi-chunk match but
	// not the serials: chunk 0 (page one) carries page two's serial and
	// chunk 1 (page two) carries page one's serial.
	expect(t, p1.serial != p2.serial, "p1.serial != p2.serial")
	h := zigUnwrap(t, search.Next())
	saExpectHighlight(t, h, 8, 1, []saChunk{
		{node: p1, start: SizeCellCountInt(cap_ - 1), end: SizeCellCountInt(cap_), serial: Some(p2.serial)},
		{node: p2, start: 0, end: 1, serial: Some(p1.serial)},
	})
	expect(t, h.chunks[0].serial != h.chunks[0].node.serial, "h.chunks.items(.serial)[0] != h.chunks.items(.node)[0].serial")
	expect(t, !search.Next().Valid, "search.next() == null")
	fed, err = search.Feed()
	zigTry(t, err)
	expect(t, !fed, "!try search.feed()")
}

// Zig: test "term51: PageListSearch.next: reverse order across pages, nothing before feed"
func TestSaPagelist_term51_PageListSearch_next_reverse_order_across_pages_nothing_before_feed(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, 2*cap_+1, []saMark{
		{y: 0, text: "Fizz"},
		{y: 5, text: "Fizz"},
		{y: cap_, text: "Fizz"},
		{y: 2 * cap_, text: "Fizz"},
	})
	p1 := pages.pages.first
	p2 := p1.next
	p3 := pages.pages.last

	search, err := SearchPagelistInit([]uint8("Fizz"), pages, p3)
	zigTry(t, err)
	defer search.Deinit()
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p3, start: 0, end: 1}})
	expect(t, !search.Next().Valid, "search.next() == null")

	fed, err := search.Feed()
	zigTry(t, err)
	expect(t, fed, "try search.feed()")
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p2, start: 0, end: 1}})
	expect(t, !search.Next().Valid, "search.next() == null")

	fed, err = search.Feed()
	zigTry(t, err)
	expect(t, fed, "try search.feed()")
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p1, start: 5, end: 6}})
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p1, start: 0, end: 1}})
	expect(t, !search.Next().Valid, "search.next() == null")
	fed, err = search.Feed()
	zigTry(t, err)
	expect(t, !fed, "!try search.feed()")
}

// Zig: test "term51: PageListSearch.next: starting at a middle page never sees later pages"
func TestSaPagelist_term51_PageListSearch_next_starting_at_a_middle_page_never_sees_later_pages(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, 2*cap_+1, []saMark{
		{y: 0, text: "Fizz"},
		{y: cap_, text: "Fizz"},
		{y: 2 * cap_, text: "Fizz"},
	})
	p1 := pages.pages.first
	p2 := p1.next

	search, err := SearchPagelistInit([]uint8("Fizz"), pages, p2)
	zigTry(t, err)
	defer search.Deinit()
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p2, start: 0, end: 1}})
	expect(t, !search.Next().Valid, "search.next() == null")
	fed, err := search.Feed()
	zigTry(t, err)
	expect(t, fed, "try search.feed()")
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p1, start: 0, end: 1}})
	fed, err = search.Feed()
	zigTry(t, err)
	expect(t, !fed, "!try search.feed()")
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: PageListSearch.feed: feeds one page per call when it has enough bytes and moves the pin"
func TestSaPagelist_term51_PageListSearch_feed_feeds_one_page_per_call_when_it_has_enough_bytes_and_moves_the_pin(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, 2*cap_+1, []saMark{
		{y: 0, text: "abcd"},
		{y: cap_, text: "abcd"},
	})
	p1 := pages.pages.first
	p2 := p1.next
	p3 := pages.pages.last

	search, err := SearchPagelistInit([]uint8("Fizz"), pages, p3)
	zigTry(t, err)
	defer search.Deinit()
	expectEqual(t, uint(1), search.window.meta.Len())

	fed, err := search.Feed()
	zigTry(t, err)
	expect(t, fed, "try search.feed()")
	expect(t, search.pin.node == p2, "search.pin.node == p2")
	expectEqual(t, uint(2), search.window.meta.Len())
	// The window is [p3, p2] in reverse search order.
	expect(t, saMetaNode(&search.window, 1) == p2, "term51ALMetaNode(&search.window, 1) == p2")

	fed, err = search.Feed()
	zigTry(t, err)
	expect(t, fed, "try search.feed()")
	expect(t, search.pin.node == p1, "search.pin.node == p1")
	expectEqual(t, uint(3), search.window.meta.Len())
	expect(t, saMetaNode(&search.window, 2) == p1, "term51ALMetaNode(&search.window, 2) == p1")

	// The pin keeps its x/y from init and is still tracked.
	expectEqual(t, SizeCellCountInt(9), search.pin.x)
	expect(t, saHasPin(pages, search.pin), "term51ALHasPin(pages, search.pin)")

	fed, err = search.Feed()
	zigTry(t, err)
	expect(t, !fed, "!try search.feed()")
	expect(t, search.pin.node == p1, "search.pin.node == p1")
	expectEqual(t, uint(3), search.window.meta.Len())
}

// Zig: test "term51: PageListSearch.feed: false without a previous page and nothing changes"
func TestSaPagelist_term51_PageListSearch_feed_false_without_a_previous_page_and_nothing_changes(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	saWrite(t, term, "Fizz")
	p := pages.pages.first
	search, err := SearchPagelistInit([]uint8("Fizz"), pages, p)
	zigTry(t, err)
	defer search.Deinit()
	length := search.window.data.Len()
	for i := 0; i < 3; i++ {
		fed, err := search.Feed()
		zigTry(t, err)
		expect(t, !fed, "!try search.feed()")
		expect(t, search.pin.node == p, "search.pin.node == p")
		expectEqual(t, length, search.window.data.Len())
		expectEqual(t, uint(1), search.window.meta.Len())
	}
}

// Zig: test "term51: PageListSearch.feed: keeps appending pages until needle length bytes were added"
func TestSaPagelist_term51_PageListSearch_feed_keeps_appending_pages_until_needle_length_bytes_were_added(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	// Three blank pages: each adds one byte (its trailing newline).
	saFill(t, term, 2*cap_+1, nil)
	p1 := pages.pages.first
	p2 := p1.next
	p3 := pages.pages.last

	// Needle of 4: p2 and p1 add two bytes, fewer than needed; still true.
	four, err := SearchPagelistInit([]uint8("Fizz"), pages, p3)
	zigTry(t, err)
	defer four.Deinit()
	fed, err := four.Feed()
	zigTry(t, err)
	expect(t, fed, "try four.feed()")
	expect(t, four.pin.node == p1, "four.pin.node == p1")
	expectEqual(t, uint(3), four.window.meta.Len())
	fed, err = four.Feed()
	zigTry(t, err)
	expect(t, !fed, "!try four.feed()")

	// Needle of 2: bytes needed 2, p2 adds 1, p1 adds 1: both in one call.
	two, err := SearchPagelistInit([]uint8("ab"), pages, p3)
	zigTry(t, err)
	defer two.Deinit()
	fed, err = two.Feed()
	zigTry(t, err)
	expect(t, fed, "try two.feed()")
	expect(t, two.pin.node == p1, "two.pin.node == p1")
	expectEqual(t, uint(3), two.window.meta.Len())

	// Needle of 1: exactly one byte per page, so one page per call.
	one, err := SearchPagelistInit([]uint8("a"), pages, p3)
	zigTry(t, err)
	defer one.Deinit()
	fed, err = one.Feed()
	zigTry(t, err)
	expect(t, fed, "try one.feed()")
	expect(t, one.pin.node == p2, "one.pin.node == p2")
	expectEqual(t, uint(2), one.window.meta.Len())
	fed, err = one.Feed()
	zigTry(t, err)
	expect(t, fed, "try one.feed()")
	expect(t, one.pin.node == p1, "one.pin.node == p1")
	fed, err = one.Feed()
	zigTry(t, err)
	expect(t, !fed, "!try one.feed()")
}

// Zig: test "term51: PageListSearch.feed: a page of exactly needle length bytes stops the feed"
func TestSaPagelist_term51_PageListSearch_feed_a_page_of_exactly_needle_length_bytes_stops_the_feed(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	// Page two holds "abc" plus its newline: four bytes.
	saFill(t, term, 2*cap_+1, []saMark{{y: cap_, text: "abc"}})
	p1 := pages.pages.first
	p2 := p1.next
	p3 := pages.pages.last

	search, err := SearchPagelistInit([]uint8("Fizz"), pages, p3)
	zigTry(t, err)
	defer search.Deinit()
	fed, err := search.Feed()
	zigTry(t, err)
	expect(t, fed, "try search.feed()")
	expect(t, search.pin.node == p2, "search.pin.node == p2")
	expectEqual(t, uint(2), search.window.meta.Len())
	fed, err = search.Feed()
	zigTry(t, err)
	expect(t, fed, "try search.feed()")
	expect(t, search.pin.node == p1, "search.pin.node == p1")
}

// Zig: test "term51: PageListSearch.feed: pages that add no bytes are walked but reported as not fed"
func TestSaPagelist_term51_PageListSearch_feed_pages_that_add_no_bytes_are_walked_but_reported_as_not_fed(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, 2*cap_+1, nil)
	p1 := pages.pages.first
	p2 := p1.next
	p3 := pages.pages.last
	saWrapLast(p1)
	saWrapLast(p2)

	search, err := SearchPagelistInit([]uint8("Fizz"), pages, p3)
	zigTry(t, err)
	defer search.Deinit()
	// Both earlier pages are wrapped and blank: nothing is added, the pin
	// still moves all the way to the first page and feed reports false.
	fed, err := search.Feed()
	zigTry(t, err)
	expect(t, !fed, "!try search.feed()")
	expect(t, search.pin.node == p1, "search.pin.node == p1")
	expectEqual(t, uint(1), search.window.meta.Len())
	fed, err = search.Feed()
	zigTry(t, err)
	expect(t, !fed, "!try search.feed()")
	expect(t, search.pin.node == p1, "search.pin.node == p1")
}

// Zig: test "term51: PageListSearch.feed: skips an empty wrapped page and adds the next one"
func TestSaPagelist_term51_PageListSearch_feed_skips_an_empty_wrapped_page_and_adds_the_next_one(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, 2*cap_+1, []saMark{{y: 0, text: "Fizz"}})
	p1 := pages.pages.first
	p2 := p1.next
	p3 := pages.pages.last
	saWrapLast(p2)

	search, err := SearchPagelistInit([]uint8("Fizz"), pages, p3)
	zigTry(t, err)
	defer search.Deinit()
	fed, err := search.Feed()
	zigTry(t, err)
	expect(t, fed, "try search.feed()")
	expect(t, search.pin.node == p1, "search.pin.node == p1")
	// p2 has no meta of its own.
	expectEqual(t, uint(2), search.window.meta.Len())
	expect(t, saMetaNode(&search.window, 0) == p3, "term51ALMetaNode(&search.window, 0) == p3")
	expect(t, saMetaNode(&search.window, 1) == p1, "term51ALMetaNode(&search.window, 1) == p1")
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p1, start: 0, end: 1}})
}

// Zig: test "term51: PageListSearch.feed: empty needle feeds one page but always reports false"
func TestSaPagelist_term51_PageListSearch_feed_empty_needle_feeds_one_page_but_always_reports_false(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, 2*cap_+1, nil)
	p1 := pages.pages.first
	p2 := p1.next
	p3 := pages.pages.last

	search, err := SearchPagelistInit([]uint8(""), pages, p3)
	zigTry(t, err)
	defer search.Deinit()
	fed, err := search.Feed()
	zigTry(t, err)
	expect(t, !fed, "!try search.feed()")
	expect(t, search.pin.node == p2, "search.pin.node == p2")
	expectEqual(t, uint(2), search.window.meta.Len())
	fed, err = search.Feed()
	zigTry(t, err)
	expect(t, !fed, "!try search.feed()")
	expect(t, search.pin.node == p1, "search.pin.node == p1")
	expectEqual(t, uint(3), search.window.meta.Len())
	fed, err = search.Feed()
	zigTry(t, err)
	expect(t, !fed, "!try search.feed()")
}

// Zig: test "term51: PageListSearch.feed: garbage pin ends the search without touching the window"
func TestSaPagelist_term51_PageListSearch_feed_garbage_pin_ends_the_search_without_touching_the_window(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, cap_+1, []saMark{{y: 0, text: "Fizz"}})
	p2 := pages.pages.last

	search, err := SearchPagelistInit([]uint8("Fizz"), pages, p2)
	zigTry(t, err)
	defer search.Deinit()
	search.pin.garbage = true
	for i := 0; i < 2; i++ {
		fed, err := search.Feed()
		zigTry(t, err)
		expect(t, !fed, "!try search.feed()")
		expect(t, search.pin.node == p2, "search.pin.node == p2")
		expectEqual(t, uint(1), search.window.meta.Len())
	}
	// next() still works on what was already loaded.
	expect(t, !search.Next().Valid, "search.next() == null")

	// Without the flag the previous page is fed.
	search.pin.garbage = false
	fed, err := search.Feed()
	zigTry(t, err)
	expect(t, fed, "try search.feed()")
	expect(t, search.Next().Valid, "search.next() != null")
}

// Zig: test "term51: PageListSearch.feed: pin becomes garbage when its page is pruned"
func TestSaPagelist_term51_PageListSearch_feed_pin_becomes_garbage_when_its_page_is_pruned(t *testing.T) {
	testStdCapacity(t)
	// Zero here forces minimum max size to effectively two pages.
	p, err := PageListInit(80, 24, Some(uint64(0)))
	zigTry(t, err)
	defer p.Deinit()
	page1_node := p.pages.last
	// Zig: for (0..n) evaluates n once.
	n1 := uint(page1_node.data.capacity.rows - page1_node.data.size.rows)
	for i := uint(0); i < n1; i++ {
		saGrowNull(t, &p)
	}
	page2_node, err := p.Grow()
	zigTry(t, err)
	expect(t, page2_node != nil, "(try p.grow()).?")
	// Zig: for (0..n) evaluates n once.
	n2 := uint(page2_node.data.capacity.rows - page2_node.data.size.rows)
	for i := uint(0); i < n2; i++ {
		saGrowNull(t, &p)
	}

	// Search from the first page and from the last page.
	from_first, err := SearchPagelistInit([]uint8("Test"), &p, page1_node)
	zigTry(t, err)
	defer from_first.Deinit()
	from_last, err := SearchPagelistInit([]uint8("Test"), &p, page2_node)
	zigTry(t, err)
	defer from_last.Deinit()
	expect(t, !from_first.pin.garbage, "!from_first.pin.garbage")
	expect(t, !from_last.pin.garbage, "!from_last.pin.garbage")

	// The first page is reused as the new last page.
	new_, err := p.Grow()
	zigTry(t, err)
	expect(t, new_ != nil, "(try p.grow()).?")
	expect(t, new_ == page1_node, "new == page1_node")
	expect(t, p.pages.first == page2_node, "p.pages.first.? == page2_node")

	// The pin in the pruned page moved to the top-left of the new first page.
	expect(t, from_first.pin.garbage, "from_first.pin.garbage")
	expect(t, from_first.pin.node == page2_node, "from_first.pin.node == page2_node")
	expectEqual(t, SizeCellCountInt(0), from_first.pin.x)
	expectEqual(t, SizeCellCountInt(0), from_first.pin.y)
	fed, err := from_first.Feed()
	zigTry(t, err)
	expect(t, !fed, "!try from_first.feed()")

	// The other pin is fine, but its previous page is gone.
	expect(t, !from_last.pin.garbage, "!from_last.pin.garbage")
	expect(t, from_last.pin.node == page2_node, "from_last.pin.node == page2_node")
	fed, err = from_last.Feed()
	zigTry(t, err)
	expect(t, !fed, "!try from_last.feed()")
}

// Go-only: panic test: PageListSearch.init: the start node has zero rows (`start.data.size.rows - 1` overflows)
func TestSaPagelist_panic_PageListSearch_init_the_start_node_has_zero_rows_start_data_size_rows_1_overflows(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	p := pages.pages.first

	rows := p.data.size.rows
	p.data.size.rows = 0
	defer func() { p.data.size.rows = rows }()
	saExpectPanic(t, func() {
		_, _ = SearchPagelistInit([]uint8("Fizz"), pages, p)
	})
}

// Go-only: panic test: PageListSearch.init: the start node has zero columns (`start.data.size.cols - 1` overflows)
func TestSaPagelist_panic_PageListSearch_init_the_start_node_has_zero_columns_start_data_size_cols_1_overflows(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	p := pages.pages.first

	cols := p.data.size.cols
	p.data.size.cols = 0
	defer func() { p.data.size.cols = cols }()
	saExpectPanic(t, func() {
		_, _ = SearchPagelistInit([]uint8("Fizz"), pages, p)
	})
}
