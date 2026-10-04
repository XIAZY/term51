// Tests translated one for one from src/terminal/search/viewport.zig (Ghostty
// v1.3.1 plus the term51 additional tests). Each Go test carries the exact
// Zig title in a "// Zig: test" comment. The helpers (saTerminal, saFill,
// saWrite, saCap, saWrapLast, saMetaNode, saExpectHighlight, ...) are shared
// with the other search tests and are in search_test.go; see the translation
// notes there. Helpers of this file: saFpNodes (term51ALFpNodes) and
// saPageCountOfViewport (term51ALPageCountOfViewport).
//
// Translation notes:
//   - `search.active_dirty = false` is `search.active_dirty = Some(false)`;
//     `search.active_dirty.?` is zigUnwrap(t, search.active_dirty);
//     `search.fingerprint == null` is `!search.fingerprint.Valid`.
//   - The raw pointer of the fingerprint's node list (`.ptr`) is the address
//     of its first element.
//   - A block with a `defer` inside (Fingerprint tests) is a closure called
//     in place, so the defer runs at the end of the block.
//
// Not translated (scripts/not-translated-sa.txt): the allocation failure
// tests (FailingAllocator).
//
// Go-only panic tests: ViewportSearch.needle on a reverse window (assert),
// ViewportSearch.update on a previous page with zero rows (`size.rows - 1`
// overflows in Zig).

package terminal

import "testing"

// saFpNodes is term51ALFpNodes: `search.fingerprint.?.nodes`.
func saFpNodes(t *testing.T, search *SearchViewport) []*PageListNode {
	t.Helper()
	return zigUnwrap(t, search.fingerprint).nodes
}

// saPageCountOfViewport is term51ALPageCountOfViewport.
func saPageCountOfViewport(t *testing.T, pages *PageList) uint {
	t.Helper()
	fp, err := searchViewportFingerprintInit(pages)
	zigTry(t, err)
	defer fp.Deinit()
	return uint(len(fp.nodes))
}

// saScrollRow is `pages.scroll(.{ .row = row })`.
func saScrollRow(pages *PageList, row uint) {
	pages.Scroll(PageListScroll{Tag: PageListScrollRow, Row: row})
}

// Zig: test "simple search"
func TestSaViewport_simple_search(t *testing.T) {
	testStdCapacity(t)
	term, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz\r\nBuzz\r\nFizz\r\nBang")))

	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	changed, err := search.Update(&term.screens.active.pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(&t.screens.active.pages)")

	// Viewport contains active so update should always re-search.
	changed, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(&t.screens.active.pages)")

	{
		h := zigUnwrap(t, search.Next())
		sel := h.Untracked()
		expectEqual(t, saActivePoint(0, 0), zigUnwrap(t, term.screens.active.pages.PointFromPin(PointTagActive, sel.start)))
		expectEqual(t, saActivePoint(3, 0), zigUnwrap(t, term.screens.active.pages.PointFromPin(PointTagActive, sel.end)))
	}
	{
		h := zigUnwrap(t, search.Next())
		sel := h.Untracked()
		expectEqual(t, saActivePoint(0, 2), zigUnwrap(t, term.screens.active.pages.PointFromPin(PointTagActive, sel.start)))
		expectEqual(t, saActivePoint(3, 2), zigUnwrap(t, term.screens.active.pages.PointFromPin(PointTagActive, sel.end)))
	}
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "clear screen and search"
func TestSaViewport_clear_screen_and_search(t *testing.T) {
	testStdCapacity(t)
	term, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz\r\nBuzz\r\nFizz\r\nBang")))

	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	changed, err := search.Update(&term.screens.active.pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(&t.screens.active.pages)")

	zigTry(t, s.NextSlice([]uint8("\x1b[2J"))) // Clear screen
	zigTry(t, s.NextSlice([]uint8("\x1b[H")))  // Move cursor home
	zigTry(t, s.NextSlice([]uint8("Buzz\r\nFizz\r\nBuzz")))
	changed, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(&t.screens.active.pages)")

	{
		h := zigUnwrap(t, search.Next())
		sel := h.Untracked()
		expectEqual(t, saActivePoint(0, 1), zigUnwrap(t, term.screens.active.pages.PointFromPin(PointTagActive, sel.start)))
		expectEqual(t, saActivePoint(3, 1), zigUnwrap(t, term.screens.active.pages.PointFromPin(PointTagActive, sel.end)))
	}
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "clear screen and search dirty tracking"
func TestSaViewport_clear_screen_and_search_dirty_tracking(t *testing.T) {
	testStdCapacity(t)
	term, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz\r\nBuzz\r\nFizz\r\nBang")))

	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()

	// Turn on dirty tracking
	search.active_dirty = Some(false)

	// Should update since we've never searched before
	changed, err := search.Update(&term.screens.active.pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(&t.screens.active.pages)")

	// Should not update since nothing changed
	changed, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	expect(t, !changed, "!try search.update(&t.screens.active.pages)")

	zigTry(t, s.NextSlice([]uint8("\x1b[2J"))) // Clear screen
	zigTry(t, s.NextSlice([]uint8("\x1b[H")))  // Move cursor home
	zigTry(t, s.NextSlice([]uint8("Buzz\r\nFizz\r\nBuzz")))

	// Should still not update since active area isn't dirty
	changed, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	expect(t, !changed, "!try search.update(&t.screens.active.pages)")

	// Mark
	search.active_dirty = Some(true)
	changed, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(&t.screens.active.pages)")

	{
		h := zigUnwrap(t, search.Next())
		sel := h.Untracked()
		expectEqual(t, saActivePoint(0, 1), zigUnwrap(t, term.screens.active.pages.PointFromPin(PointTagActive, sel.start)))
		expectEqual(t, saActivePoint(3, 1), zigUnwrap(t, term.screens.active.pages.PointFromPin(PointTagActive, sel.end)))
	}
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "history search, no active area"
func TestSaViewport_history_search_no_active_area(t *testing.T) {
	testStdCapacity(t)
	term, err := TerminalInit(terminalOptionsDefault(10, 2))
	zigTry(t, err)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Fill up first page
	first_page_rows := uint(term.screens.active.pages.pages.first.data.capacity.rows)
	zigTry(t, s.NextSlice([]uint8("Fizz\r\n")))
	for i := uint(1); i < first_page_rows-1; i++ {
		zigTry(t, s.NextSlice([]uint8("\r\n")))
	}
	expect(t, term.screens.active.pages.pages.first == term.screens.active.pages.pages.last, "t.screens.active.pages.pages.first == t.screens.active.pages.pages.last")

	// Create second page
	zigTry(t, s.NextSlice([]uint8("\r\n")))
	expect(t, term.screens.active.pages.pages.first != term.screens.active.pages.pages.last, "t.screens.active.pages.pages.first != t.screens.active.pages.pages.last")
	zigTry(t, s.NextSlice([]uint8("Buzz\r\nFizz")))

	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})

	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	changed, err := search.Update(&term.screens.active.pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(&t.screens.active.pages)")

	{
		h := zigUnwrap(t, search.Next())
		sel := h.Untracked()
		expectEqual(t, saScreenPoint(0, 0), zigUnwrap(t, term.screens.active.pages.PointFromPin(PointTagScreen, sel.start)))
		expectEqual(t, saScreenPoint(3, 0), zigUnwrap(t, term.screens.active.pages.PointFromPin(PointTagScreen, sel.end)))
	}
	expect(t, !search.Next().Valid, "search.next() == null")

	// Viewport doesn't contain active
	changed, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	expect(t, !changed, "!try search.update(&t.screens.active.pages)")
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ViewportSearch.init: no fingerprint, no dirty tracking, empty forward window"
func TestSaViewport_term51_ViewportSearch_init_no_fingerprint_no_dirty_tracking_empty_forward_window(t *testing.T) {
	testStdCapacity(t)
	buf := []uint8("Fizz")
	search, err := SearchViewportInit(buf)
	zigTry(t, err)
	defer search.Deinit()
	expect(t, !search.fingerprint.Valid, "search.fingerprint == null")
	expect(t, !search.active_dirty.Valid, "search.active_dirty == null")
	expectEqual(t, true, search.window.direction == searchSlidingWindowDirectionForward)
	expectEqualStrings(t, "Fizz", string(search.window.needle))
	expect(t, &search.window.needle[0] != &buf[0], "search.window.needle.ptr != @as([*]const u8, &buf)")
	expectEqual(t, uint(0), search.window.meta.Len())
	expectEqual(t, uint(0), search.window.data.Len())
	buf[0] = 'X'
	expectEqualStrings(t, "Fizz", string(search.window.needle))
}

// Zig: test "term51: ViewportSearch.init: empty needle is accepted"
func TestSaViewport_term51_ViewportSearch_init_empty_needle_is_accepted(t *testing.T) {
	testStdCapacity(t)
	search, err := SearchViewportInit([]uint8(""))
	zigTry(t, err)
	defer search.Deinit()
	expectEqual(t, uint(0), uint(len(search.window.needle)))

	// An update on a single page works (no overlap loop is reached). next()
	// would panic on an integer underflow in SlidingWindow.highlight.
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	changed, err := search.Update(&term.screens.active.pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(&t.screens.active.pages)")
	expectEqual(t, uint(1), search.window.meta.Len())
}

// Zig: test "term51: ViewportSearch.deinit: frees fingerprint and window"
func TestSaViewport_term51_ViewportSearch_deinit_frees_fingerprint_and_window(t *testing.T) {
	testStdCapacity(t)
	{
		search, err := SearchViewportInit([]uint8("Fizz"))
		zigTry(t, err)
		search.Deinit()
	}
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	saWrite(t, term, "Fizz")
	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	changed, err := search.Update(&term.screens.active.pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(&t.screens.active.pages)")
	expect(t, search.fingerprint.Valid, "search.fingerprint != null")
	search.Deinit()
}

// Zig: test "term51: ViewportSearch.reset: drops fingerprint and results so the next update re-searches"
func TestSaViewport_term51_ViewportSearch_reset_drops_fingerprint_and_results_so_the_next_update_re_searches(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, cap_+10, []saMark{{y: 0, text: "Fizz"}})
	p1 := pages.pages.first
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})

	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	search.active_dirty = Some(false)
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, !changed, "!try search.update(pages)")
	expect(t, search.fingerprint.Valid, "search.fingerprint != null")
	_ = zigUnwrap(t, search.Next())
	expect(t, search.window.data_offset != 0, "search.window.data_offset != 0")

	search.Reset()
	expect(t, !search.fingerprint.Valid, "search.fingerprint == null")
	expectEqual(t, uint(0), search.window.meta.Len())
	expectEqual(t, uint(0), search.window.data.Len())
	expectEqual(t, uint(0), search.window.data_offset)
	expect(t, !search.Next().Valid, "search.next() == null")
	// reset does not touch the dirty tracking mode.
	expect(t, search.active_dirty.Valid, "search.active_dirty != null")
	expectEqual(t, false, zigUnwrap(t, search.active_dirty))

	// The viewport did not change and the active area is not visible, but
	// the next update re-searches and finds the match again.
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p1, start: 0, end: 1}})

	// Resetting twice and resetting a fresh search is fine.
	search.Reset()
	search.Reset()
	expect(t, !search.fingerprint.Valid, "search.fingerprint == null")
	fresh, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer fresh.Deinit()
	fresh.Reset()
	expect(t, !fresh.fingerprint.Valid, "fresh.fingerprint == null")
	expect(t, !fresh.active_dirty.Valid, "fresh.active_dirty == null")
}

// Zig: test "term51: ViewportSearch.needle: returns the owned needle"
func TestSaViewport_term51_ViewportSearch_needle_returns_the_owned_needle(t *testing.T) {
	testStdCapacity(t)
	buf := []uint8("Fizz")
	search, err := SearchViewportInit(buf)
	zigTry(t, err)
	defer search.Deinit()
	expectEqualStrings(t, "Fizz", string(search.Needle()))
	expect(t, &search.Needle()[0] == &search.window.needle[0], "search.needle().ptr == search.window.needle.ptr")
	expect(t, &search.Needle()[0] != &buf[0], "search.needle().ptr != @as([*]const u8, &buf)")

	empty, err := SearchViewportInit([]uint8(""))
	zigTry(t, err)
	defer empty.Deinit()
	expectEqual(t, uint(0), uint(len(empty.Needle())))

	// The case of the needle is preserved (only matching ignores case).
	mixed, err := SearchViewportInit([]uint8("fIzZ"))
	zigTry(t, err)
	defer mixed.Deinit()
	expectEqualStrings(t, "fIzZ", string(mixed.Needle()))
}

// Zig: test "term51: ViewportSearch.update: first update fingerprints the viewport and fills the window"
func TestSaViewport_term51_ViewportSearch_update_first_update_fingerprints_the_viewport_and_fills_the_window(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	saWrite(t, term, "Fizz\r\nBuzz\r\nFizz\r\nBang")
	p := pages.pages.first

	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	expectEqual(t, uint(1), uint(len(saFpNodes(t, &search))))
	expect(t, saFpNodes(t, &search)[0] == p, "term51ALFpNodes(&search)[0] == p")
	expectEqual(t, uint(1), search.window.meta.Len())
	// Dirty tracking stays off.
	expect(t, !search.active_dirty.Valid, "search.active_dirty == null")

	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p, start: 0, end: 1}})
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p, start: 2, end: 3}})
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ViewportSearch.update: viewport at the bottom re-searches every time without dirty tracking"
func TestSaViewport_term51_ViewportSearch_update_viewport_at_the_bottom_re_searches_every_time_without_dirty_tracking(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	saWrite(t, term, "Fizz")
	p := pages.pages.first

	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	for i := 0; i < 3; i++ {
		changed, err := search.Update(pages)
		zigTry(t, err)
		expect(t, changed, "try search.update(pages)")
		expectEqual(t, uint(1), search.window.meta.Len())
		saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p, start: 0, end: 1}})
		expect(t, !search.Next().Valid, "search.next() == null")
	}
}

// Zig: test "term51: ViewportSearch.update: terminal changes are found by re-searching and the window is rebuilt"
func TestSaViewport_term51_ViewportSearch_update_terminal_changes_are_found_by_re_searching_and_the_window_is_rebuilt(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	saWrite(t, term, "Fizz\r\nBuzz")
	p := pages.pages.first

	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p, start: 0, end: 1}})

	// Stale until update.
	saWrite(t, term, "\x1b[2J\x1b[HBuzz\r\nbuzz\r\nFIZZ")
	expect(t, !search.Next().Valid, "search.next() == null")

	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p, start: 2, end: 3}})
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ViewportSearch.update: unchanged viewport in history returns false and keeps unconsumed results"
func TestSaViewport_term51_ViewportSearch_update_unchanged_viewport_in_history_returns_false_and_keeps_unconsumed_results(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, cap_+10, []saMark{
		{y: 0, text: "Fizz"},
		{y: 2, text: "fizz"},
	})
	p1 := pages.pages.first
	expect(t, p1 != pages.pages.last, "p1 != pages.pages.last.?")
	expectEqual(t, uint(10), uint(pages.pages.last.data.size.rows))
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})

	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	fp_ptr := &saFpNodes(t, &search)[0]
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p1, start: 0, end: 1}})

	// Not re-searched: the second match is still pending in the window.
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, !changed, "!try search.update(pages)")
	expect(t, &saFpNodes(t, &search)[0] == fp_ptr, "term51ALFpNodes(&search).ptr == fp_ptr")
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p1, start: 2, end: 3}})
	expect(t, !search.Next().Valid, "search.next() == null")
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, !changed, "!try search.update(pages)")
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ViewportSearch.update: history viewport reports matches outside the visible rows"
func TestSaViewport_term51_ViewportSearch_update_history_viewport_reports_matches_outside_the_visible_rows(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, cap_+10, []saMark{
		{y: 0, text: "Fizz"},
		{y: 20, text: "Fizz"},
		{y: cap_ - 1, text: "Fizz"},
	})
	p1 := pages.pages.first

	// The viewport shows lines 20..29: only the middle match is visible.
	saScrollRow(pages, 20)
	expect(t, pages.viewport.Tag == PageListViewportPin, "pages.viewport == .pin")

	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	expectEqual(t, uint(1), uint(len(saFpNodes(t, &search))))
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p1, start: 0, end: 1}})
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p1, start: 20, end: 21}})
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p1, start: SizeCellCountInt(cap_ - 1), end: SizeCellCountInt(cap_)}})
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ViewportSearch.update: moving within the same page is not a viewport change"
func TestSaViewport_term51_ViewportSearch_update_moving_within_the_same_page_is_not_a_viewport_change(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, cap_+10, nil)
	p1 := pages.pages.first

	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	// The fingerprint is only the list of page nodes.
	saScrollRow(pages, 5)
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, !changed, "!try search.update(pages)")
	saScrollRow(pages, cap_-10)
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, !changed, "!try search.update(pages)")
	expect(t, saFpNodes(t, &search)[0] == p1, "term51ALFpNodes(&search)[0] == p1")
	expectEqual(t, uint(1), uint(len(saFpNodes(t, &search))))
}

// Zig: test "term51: ViewportSearch.update: viewport moving to other pages re-searches"
func TestSaViewport_term51_ViewportSearch_update_viewport_moving_to_other_pages_re_searches(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, cap_+10, []saMark{
		{y: 0, text: "Fizz"},
		{y: cap_ - 1, text: "Fizz"},
		{y: cap_, text: "Fizz"},
		{y: cap_ + 9, text: "Fizz"},
	})
	p1 := pages.pages.first
	p2 := pages.pages.last
	expect(t, p1.next == p2, "p1.next == p2")
	expectEqual(t, uint(10), uint(p2.data.size.rows))

	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()

	// Bottom: only the last page.
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	expectEqual(t, uint(1), uint(len(saFpNodes(t, &search))))
	expect(t, saFpNodes(t, &search)[0] == p2, "term51ALFpNodes(&search)[0] == p2")
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p2, start: 0, end: 1}})
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p2, start: 9, end: 10}})
	expect(t, !search.Next().Valid, "search.next() == null")

	// Top: only the first page.
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	expectEqual(t, uint(1), uint(len(saFpNodes(t, &search))))
	expect(t, saFpNodes(t, &search)[0] == p1, "term51ALFpNodes(&search)[0] == p1")
	expectEqual(t, uint(1), search.window.meta.Len())
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p1, start: 0, end: 1}})
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p1, start: SizeCellCountInt(cap_ - 1), end: SizeCellCountInt(cap_)}})
	expect(t, !search.Next().Valid, "search.next() == null")

	// Straddling the boundary: both pages, in page order.
	saScrollRow(pages, cap_-3)
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	expectEqual(t, uint(2), uint(len(saFpNodes(t, &search))))
	expect(t, saFpNodes(t, &search)[0] == p1, "term51ALFpNodes(&search)[0] == p1")
	expect(t, saFpNodes(t, &search)[1] == p2, "term51ALFpNodes(&search)[1] == p2")
	expectEqual(t, uint(2), search.window.meta.Len())
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p1, start: 0, end: 1}})
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p1, start: SizeCellCountInt(cap_ - 1), end: SizeCellCountInt(cap_)}})
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p2, start: 0, end: 1}})
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p2, start: 9, end: 10}})
	expect(t, !search.Next().Valid, "search.next() == null")

	// Back at the top and unchanged afterwards.
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, !changed, "!try search.update(pages)")
}

// Zig: test "term51: ViewportSearch.update: a viewport that overlaps the active area always re-searches"
func TestSaViewport_term51_ViewportSearch_update_a_viewport_that_overlaps_the_active_area_always_re_searches(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, cap_+10, nil)

	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()

	// Viewport three rows above the active area: pages [p1, p2], p2 holds the active area.
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportDelta, Delta: -3})
	expectEqual(t, uint(2), saPageCountOfViewport(t, pages))
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")

	// Ten rows above: p1 only, which does not hold the active area.
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportDelta, Delta: -7})
	expectEqual(t, uint(1), saPageCountOfViewport(t, pages))
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, !changed, "!try search.update(pages)")
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, !changed, "!try search.update(pages)")
}

// Zig: test "term51: ViewportSearch.update: overlap with the active area is decided per page"
func TestSaViewport_term51_ViewportSearch_update_overlap_with_the_active_area_is_decided_per_page(t *testing.T) {
	testStdCapacity(t)
	// The active area starts in page one (page two has a single row) so a
	// viewport that shows only history rows of page one still counts as
	// overlapping.
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, cap_+1, nil)
	p1 := pages.pages.first
	p2 := pages.pages.last
	expect(t, p1 != p2, "p1 != p2")
	expect(t, pages.GetTopLeft(PointTagActive).node == p1, "pages.getTopLeft(.active).node == p1")
	expect(t, zigUnwrap(t, pages.GetBottomRight(PointTagActive)).node == p2, "pages.getBottomRight(.active).?.node == p2")

	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	expectEqual(t, uint(1), saPageCountOfViewport(t, pages))
	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")

	// With dirty tracking it is only re-searched when dirty.
	search.active_dirty = Some(false)
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, !changed, "!try search.update(pages)")
	search.active_dirty = Some(true)
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	expectEqual(t, false, zigUnwrap(t, search.active_dirty))
}

// Zig: test "term51: ViewportSearch.update: dirty tracking only re-searches the active area when marked dirty"
func TestSaViewport_term51_ViewportSearch_update_dirty_tracking_only_re_searches_the_active_area_when_marked_dirty(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	saWrite(t, term, "Fizz")
	p := pages.pages.first

	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	search.active_dirty = Some(true)

	// The first search clears the flag.
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	expectEqual(t, false, zigUnwrap(t, search.active_dirty))
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p, start: 0, end: 1}})

	// Clean: nothing is redone and the window keeps its state.
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, !changed, "!try search.update(pages)")
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, !changed, "!try search.update(pages)")
	expect(t, !search.Next().Valid, "search.next() == null")

	// Dirty: re-search, flag cleared again.
	search.active_dirty = Some(true)
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	expectEqual(t, false, zigUnwrap(t, search.active_dirty))
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p, start: 0, end: 1}})
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, !changed, "!try search.update(pages)")
}

// Zig: test "term51: ViewportSearch.update: the dirty flag is consumed even when the viewport does not touch the active area"
func TestSaViewport_term51_ViewportSearch_update_the_dirty_flag_is_consumed_even_when_the_viewport_does_not_touch_the_active_area(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, cap_+10, nil)
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})

	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	search.active_dirty = Some(false)
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	search.active_dirty = Some(true)
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, !changed, "!try search.update(pages)")
	expectEqual(t, false, zigUnwrap(t, search.active_dirty))
}

// Zig: test "term51: ViewportSearch.update: a changed viewport re-searches and clears the dirty flag, null stays null"
func TestSaViewport_term51_ViewportSearch_update_a_changed_viewport_re_searches_and_clears_the_dirty_flag_null_stays_null(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, cap_+10, nil)

	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	search.active_dirty = Some(false)
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	search.active_dirty = Some(true)
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportBottom})
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	expectEqual(t, false, zigUnwrap(t, search.active_dirty))

	plain, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer plain.Deinit()
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	changed, err = plain.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try plain.update(pages)")
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportBottom})
	changed, err = plain.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try plain.update(pages)")
	expect(t, !plain.active_dirty.Valid, "plain.active_dirty == null")
}

// Zig: test "term51: ViewportSearch.update: leading overlap adds a soft-wrapped previous page before the viewport"
func TestSaViewport_term51_ViewportSearch_update_leading_overlap_adds_a_soft_wrapped_previous_page_before_the_viewport(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, cap_+10, []saMark{{y: cap_ - 1, text: "xxxxxxxxTestZ"}})
	p1 := pages.pages.first
	p2 := pages.pages.last
	expect(t, p1.next == p2, "p1.next == p2")
	expectEqual(t, uint(10), uint(p2.data.size.rows))
	expect(t, p1.data.GetRow(uint(p1.data.size.rows-1)).wrap, "p1.data.getRow(p1.data.size.rows - 1).wrap")

	search, err := SearchViewportInit([]uint8("Test"))
	zigTry(t, err)
	defer search.Deinit()
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	// The fingerprint only lists the viewport pages, the window has more.
	expectEqual(t, uint(1), uint(len(saFpNodes(t, &search))))
	expect(t, saFpNodes(t, &search)[0] == p2, "term51ALFpNodes(&search)[0] == p2")
	expectEqual(t, uint(2), search.window.meta.Len())
	expect(t, saMetaNode(&search.window, 0) == p1, "term51ALMetaNode(&search.window, 0) == p1")
	expect(t, saMetaNode(&search.window, 1) == p2, "term51ALMetaNode(&search.window, 1) == p2")
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 8, 1, []saChunk{
		{node: p1, start: SizeCellCountInt(cap_ - 1), end: SizeCellCountInt(cap_)},
		{node: p2, start: 0, end: 1},
	})
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ViewportSearch.update: previous page that is not soft-wrapped is not added"
func TestSaViewport_term51_ViewportSearch_update_previous_page_that_is_not_soft_wrapped_is_not_added(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, cap_+10, []saMark{
		{y: cap_ - 1, text: "xxxxxxxxTe"},
		{y: cap_, text: "st"},
	})
	p1 := pages.pages.first
	p2 := pages.pages.last
	expect(t, !p1.data.GetRow(uint(p1.data.size.rows-1)).wrap, "!p1.data.getRow(p1.data.size.rows - 1).wrap")

	search, err := SearchViewportInit([]uint8("Test"))
	zigTry(t, err)
	defer search.Deinit()
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	expectEqual(t, uint(1), search.window.meta.Len())
	expect(t, saMetaNode(&search.window, 0) == p2, "term51ALMetaNode(&search.window, 0) == p2")
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ViewportSearch.update: leading overlap skips empty wrapped pages and stops at an unwrapped one"
func TestSaViewport_term51_ViewportSearch_update_leading_overlap_skips_empty_wrapped_pages_and_stops_at_an_unwrapped_one(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, 2*cap_+10, []saMark{{y: 0, text: "Fizz"}})
	p1 := pages.pages.first
	p2 := p1.next
	p3 := pages.pages.last
	expect(t, p2.next == p3, "p2.next == p3")

	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()

	// Nothing wrapped.
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	expectEqual(t, uint(1), search.window.meta.Len())

	// Page two wrapped and empty (adds zero bytes), page one not wrapped.
	saWrapLast(p2)
	search.Reset()
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	expectEqual(t, uint(1), search.window.meta.Len())
	expect(t, saMetaNode(&search.window, 0) == p3, "term51ALMetaNode(&search.window, 0) == p3")
	expect(t, !search.Next().Valid, "search.next() == null")

	// Both wrapped: page two adds nothing and has no meta, page one has
	// four bytes which is enough.
	saWrapLast(p1)
	search.Reset()
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	expectEqual(t, uint(2), search.window.meta.Len())
	expect(t, saMetaNode(&search.window, 0) == p1, "term51ALMetaNode(&search.window, 0) == p1")
	expect(t, saMetaNode(&search.window, 1) == p3, "term51ALMetaNode(&search.window, 1) == p3")
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p1, start: 0, end: 1}})
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ViewportSearch.update: leading overlap pages are appended nearest first so the window is out of order"
func TestSaViewport_term51_ViewportSearch_update_leading_overlap_pages_are_appended_nearest_first_so_the_window_is_out_of_order(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, 2*cap_+10, []saMark{
		{y: 0, text: "Fizz"},
		{y: cap_, text: "a"},
	})
	p1 := pages.pages.first
	p2 := p1.next
	p3 := pages.pages.last
	saWrapLast(p1)
	saWrapLast(p2)

	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	// Page two (one byte, fewer than needle-1) then page one (enough).
	expectEqual(t, uint(3), search.window.meta.Len())
	expect(t, saMetaNode(&search.window, 0) == p2, "term51ALMetaNode(&search.window, 0) == p2")
	expect(t, saMetaNode(&search.window, 1) == p1, "term51ALMetaNode(&search.window, 1) == p1")
	expect(t, saMetaNode(&search.window, 2) == p3, "term51ALMetaNode(&search.window, 2) == p3")
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p1, start: 0, end: 1}})
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ViewportSearch.update: leading overlap sums the bytes of the pages and stops once needle-1 are reached"
func TestSaViewport_term51_ViewportSearch_update_leading_overlap_sums_the_bytes_of_the_pages_and_stops_once_needle_1_are_reached(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, 3*cap_+10, []saMark{
		{y: 0, text: "z"},
		{y: cap_, text: "c"},
		{y: 2 * cap_, text: "ab"},
	})
	p1 := pages.pages.first
	p2 := p1.next
	p3 := p2.next
	p4 := pages.pages.last
	expect(t, p3.next == p4, "p3.next == p4")
	saWrapLast(p1)
	saWrapLast(p2)
	saWrapLast(p3)

	// Needle-1 = 3: page three adds two bytes, page two one more: stop.
	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	expectEqual(t, uint(3), search.window.meta.Len())
	expect(t, saMetaNode(&search.window, 0) == p3, "term51ALMetaNode(&search.window, 0) == p3")
	expect(t, saMetaNode(&search.window, 1) == p2, "term51ALMetaNode(&search.window, 1) == p2")
	expect(t, saMetaNode(&search.window, 2) == p4, "term51ALMetaNode(&search.window, 2) == p4")

	// Needle-1 = 4: all three pages are needed (2 + 1 + 1 = 4).
	five, err := SearchViewportInit([]uint8("Fizzy"))
	zigTry(t, err)
	defer five.Deinit()
	changed, err = five.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try five.update(pages)")
	expectEqual(t, uint(4), five.window.meta.Len())
	expect(t, saMetaNode(&five.window, 2) == p1, "term51ALMetaNode(&five.window, 2) == p1")

	// Needle-1 = 0: stops after the first wrapped page, even one with no bytes.
	one, err := SearchViewportInit([]uint8("z"))
	zigTry(t, err)
	defer one.Deinit()
	changed, err = one.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try one.update(pages)")
	expectEqual(t, uint(2), one.window.meta.Len())
	expect(t, saMetaNode(&one.window, 0) == p3, "term51ALMetaNode(&one.window, 0) == p3")
}

// Zig: test "term51: ViewportSearch.update: one-character needle stops leading overlap at an empty wrapped page"
func TestSaViewport_term51_ViewportSearch_update_one_character_needle_stops_leading_overlap_at_an_empty_wrapped_page(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, 2*cap_+10, []saMark{{y: 0, text: "z"}})
	p1 := pages.pages.first
	p2 := p1.next
	saWrapLast(p1)
	saWrapLast(p2)

	search, err := SearchViewportInit([]uint8("z"))
	zigTry(t, err)
	defer search.Deinit()
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	// Page two adds zero bytes and 0 >= needle-1 stops the loop, so page
	// one (which holds the match) is not searched.
	expectEqual(t, uint(1), search.window.meta.Len())
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ViewportSearch.update: trailing overlap adds a soft-wrapped last viewport page's successor"
func TestSaViewport_term51_ViewportSearch_update_trailing_overlap_adds_a_soft_wrapped_last_viewport_page_s_successor(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, cap_+10, []saMark{{y: cap_ - 1, text: "xxxxxxxxTestZ"}})
	p1 := pages.pages.first
	p2 := pages.pages.last
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})

	search, err := SearchViewportInit([]uint8("Test"))
	zigTry(t, err)
	defer search.Deinit()
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	expectEqual(t, uint(1), uint(len(saFpNodes(t, &search))))
	expect(t, saFpNodes(t, &search)[0] == p1, "term51ALFpNodes(&search)[0] == p1")
	expectEqual(t, uint(2), search.window.meta.Len())
	expect(t, saMetaNode(&search.window, 0) == p1, "term51ALMetaNode(&search.window, 0) == p1")
	expect(t, saMetaNode(&search.window, 1) == p2, "term51ALMetaNode(&search.window, 1) == p2")
	// The match is not visible (the viewport shows lines 0..9).
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 8, 1, []saChunk{
		{node: p1, start: SizeCellCountInt(cap_ - 1), end: SizeCellCountInt(cap_)},
		{node: p2, start: 0, end: 1},
	})
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ViewportSearch.update: no trailing overlap when the last viewport page is not soft-wrapped"
func TestSaViewport_term51_ViewportSearch_update_no_trailing_overlap_when_the_last_viewport_page_is_not_soft_wrapped(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, cap_+10, []saMark{
		{y: cap_ - 1, text: "xxxxxxxxTe"},
		{y: cap_, text: "st"},
	})
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})

	search, err := SearchViewportInit([]uint8("Test"))
	zigTry(t, err)
	defer search.Deinit()
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	expectEqual(t, uint(1), search.window.meta.Len())
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ViewportSearch.update: trailing overlap stops at a short unwrapped page, continues over empty wrapped pages"
func TestSaViewport_term51_ViewportSearch_update_trailing_overlap_stops_at_a_short_unwrapped_page_continues_over_empty_wrapped_pages(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, 2*cap_+10, []saMark{{y: 0, text: "Q"}})
	p1 := pages.pages.first
	p2 := p1.next
	p3 := pages.pages.last
	expect(t, p2.next == p3, "p2.next == p3")
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})

	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()

	// Page one not wrapped: nothing trailing.
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	expectEqual(t, uint(1), search.window.meta.Len())

	// Page one wrapped (it holds one byte, "Q"). Page two is blank and not wrapped: it adds its
	// newline (one byte, fewer than 3) and then the loop stops.
	saWrapLast(p1)
	search.Reset()
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	expectEqual(t, uint(2), search.window.meta.Len())
	expect(t, saMetaNode(&search.window, 0) == p1, "term51ALMetaNode(&search.window, 0) == p1")
	expect(t, saMetaNode(&search.window, 1) == p2, "term51ALMetaNode(&search.window, 1) == p2")

	// Page two also wrapped: zero bytes, no meta, the loop goes on to
	// page three which adds one byte and stops.
	saWrapLast(p2)
	search.Reset()
	changed, err = search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	expectEqual(t, uint(2), search.window.meta.Len())
	expect(t, saMetaNode(&search.window, 0) == p1, "term51ALMetaNode(&search.window, 0) == p1")
	expect(t, saMetaNode(&search.window, 1) == p3, "term51ALMetaNode(&search.window, 1) == p3")
}

// Zig: test "term51: ViewportSearch.update: trailing overlap stops when a page reaches needle-1 bytes even if it is wrapped"
func TestSaViewport_term51_ViewportSearch_update_trailing_overlap_stops_when_a_page_reaches_needle_1_bytes_even_if_it_is_wrapped(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, 2*cap_+10, []saMark{{y: 0, text: "Q"}, {y: cap_, text: "abc"}})
	p1 := pages.pages.first
	p2 := p1.next
	saWrapLast(p1)
	saWrapLast(p2)
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})

	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	expectEqual(t, uint(2), search.window.meta.Len())
	expect(t, saMetaNode(&search.window, 1) == p2, "term51ALMetaNode(&search.window, 1) == p2")
}

// Zig: test "term51: ViewportSearch.update: trailing overlap sums the bytes of the pages"
func TestSaViewport_term51_ViewportSearch_update_trailing_overlap_sums_the_bytes_of_the_pages(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, 3*cap_+10, []saMark{
		{y: 0, text: "Q"},
		{y: cap_, text: "ab"},
		{y: 2 * cap_, text: "c"},
	})
	p1 := pages.pages.first
	p2 := p1.next
	p3 := p2.next
	p4 := pages.pages.last
	expect(t, p3.next == p4, "p3.next == p4")
	saWrapLast(p1)
	saWrapLast(p2)
	saWrapLast(p3)
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})

	// Needle-1 = 3: page two adds two, page three one more: stop.
	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	expectEqual(t, uint(3), search.window.meta.Len())
	expect(t, saMetaNode(&search.window, 0) == p1, "term51ALMetaNode(&search.window, 0) == p1")
	expect(t, saMetaNode(&search.window, 1) == p2, "term51ALMetaNode(&search.window, 1) == p2")
	expect(t, saMetaNode(&search.window, 2) == p3, "term51ALMetaNode(&search.window, 2) == p3")

	// Needle-1 = 4: page four is needed as well.
	five, err := SearchViewportInit([]uint8("Fizzy"))
	zigTry(t, err)
	defer five.Deinit()
	changed, err = five.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try five.update(pages)")
	expectEqual(t, uint(4), five.window.meta.Len())
	expect(t, saMetaNode(&five.window, 3) == p4, "term51ALMetaNode(&five.window, 3) == p4")
}

// Zig: test "term51: ViewportSearch.update: one-character needle stops trailing overlap after the first appended page"
func TestSaViewport_term51_ViewportSearch_update_one_character_needle_stops_trailing_overlap_after_the_first_appended_page(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, 2*cap_+10, []saMark{{y: 0, text: "Q"}})
	p1 := pages.pages.first
	p2 := p1.next
	p3 := pages.pages.last
	saWrapLast(p1)
	saWrapLast(p2)
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})

	search, err := SearchViewportInit([]uint8("z"))
	zigTry(t, err)
	defer search.Deinit()
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	// Page two adds no bytes; 0 >= needle-1 = 0 ends the loop.
	expectEqual(t, uint(1), search.window.meta.Len())
	_ = p3
}

// Zig: test "term51: ViewportSearch.update: a soft-wrapped last page has no successor to add"
func TestSaViewport_term51_ViewportSearch_update_a_soft_wrapped_last_page_has_no_successor_to_add(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	saWrite(t, term, "Fizz")
	saWrapLast(pages.pages.last)

	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	changed, err := search.Update(pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(pages)")
	expectEqual(t, uint(1), search.window.meta.Len())
	expect(t, search.Next().Valid, "search.next() != null")
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ViewportSearch.next: null before any update and after exhaustion"
func TestSaViewport_term51_ViewportSearch_next_null_before_any_update_and_after_exhaustion(t *testing.T) {
	testStdCapacity(t)
	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	expect(t, !search.Next().Valid, "search.next() == null")

	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	p := term.screens.active.pages.pages.first
	saWrite(t, term, "Fizz\r\nxfizz")
	changed, err := search.Update(&term.screens.active.pages)
	zigTry(t, err)
	expect(t, changed, "try search.update(&t.screens.active.pages)")
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p, start: 0, end: 1}})
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 1, 4, []saChunk{{node: p, start: 1, end: 2}})
	expect(t, !search.Next().Valid, "search.next() == null")
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ViewportSearch.Fingerprint.init: lists the pages of the viewport top to bottom"
func TestSaViewport_term51_ViewportSearch_Fingerprint_init_lists_the_pages_of_the_viewport_top_to_bottom(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages

	// One page.
	func() {
		fp, err := searchViewportFingerprintInit(pages)
		zigTry(t, err)
		defer fp.Deinit()
		expectEqual(t, uint(1), uint(len(fp.nodes)))
		expect(t, fp.nodes[0] == pages.pages.first, "fp.nodes[0] == pages.pages.first.?")
	}()

	cap_ := saCap(term)
	saFill(t, term, cap_+10, nil)
	p1 := pages.pages.first
	p2 := pages.pages.last

	// Bottom: the last page only.
	func() {
		fp, err := searchViewportFingerprintInit(pages)
		zigTry(t, err)
		defer fp.Deinit()
		expectEqual(t, uint(1), uint(len(fp.nodes)))
		expect(t, fp.nodes[0] == p2, "fp.nodes[0] == p2")
	}()
	// Top: the first page only.
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	func() {
		fp, err := searchViewportFingerprintInit(pages)
		zigTry(t, err)
		defer fp.Deinit()
		expectEqual(t, uint(1), uint(len(fp.nodes)))
		expect(t, fp.nodes[0] == p1, "fp.nodes[0] == p1")
	}()
	// Straddling.
	saScrollRow(pages, cap_-1)
	func() {
		fp, err := searchViewportFingerprintInit(pages)
		zigTry(t, err)
		defer fp.Deinit()
		expectEqual(t, uint(2), uint(len(fp.nodes)))
		expect(t, fp.nodes[0] == p1, "fp.nodes[0] == p1")
		expect(t, fp.nodes[1] == p2, "fp.nodes[1] == p2")
	}()
	// Last row of the first page at the bottom of the viewport.
	saScrollRow(pages, cap_-10)
	func() {
		fp, err := searchViewportFingerprintInit(pages)
		zigTry(t, err)
		defer fp.Deinit()
		expectEqual(t, uint(1), uint(len(fp.nodes)))
		expect(t, fp.nodes[0] == p1, "fp.nodes[0] == p1")
	}()
}

// Zig: test "term51: ViewportSearch.Fingerprint.deinit: frees the node list"
func TestSaViewport_term51_ViewportSearch_Fingerprint_deinit_frees_the_node_list(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	fp, err := searchViewportFingerprintInit(&term.screens.active.pages)
	zigTry(t, err)
	fp.Deinit()
}

// Zig: test "term51: ViewportSearch.Fingerprint.eql: compares length and node pointers in order"
func TestSaViewport_term51_ViewportSearch_Fingerprint_eql_compares_length_and_node_pointers_in_order(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, cap_+10, nil)
	p1 := pages.pages.first
	p2 := pages.pages.last

	a, err := searchViewportFingerprintInit(pages)
	zigTry(t, err)
	defer a.Deinit()
	b, err := searchViewportFingerprintInit(pages)
	zigTry(t, err)
	defer b.Deinit()
	// Equal contents in separate allocations.
	expect(t, &a.nodes[0] != &b.nodes[0], "a.nodes.ptr != b.nodes.ptr")
	expect(t, a.Eql(b), "a.eql(b)")
	expect(t, b.Eql(a), "b.eql(a)")
	expect(t, a.Eql(a), "a.eql(a)")

	one_first := searchViewportFingerprint{nodes: []*PageListNode{p1}}
	one_second := searchViewportFingerprint{nodes: []*PageListNode{p2}}
	two := searchViewportFingerprint{nodes: []*PageListNode{p1, p2}}
	two_swapped := searchViewportFingerprint{nodes: []*PageListNode{p2, p1}}
	empty := searchViewportFingerprint{nodes: []*PageListNode{}}
	expect(t, a.Eql(one_second), "a.eql(one_second)")
	expect(t, !a.Eql(one_first), "!a.eql(one_first)")
	expect(t, !one_first.Eql(one_second), "!one_first.eql(one_second)")
	expect(t, !one_first.Eql(two), "!one_first.eql(two)")
	expect(t, !two.Eql(one_first), "!two.eql(one_first)")
	expect(t, two.Eql(two), "two.eql(two)")
	expect(t, !two.Eql(two_swapped), "!two.eql(two_swapped)")
	expect(t, !two_swapped.Eql(two), "!two_swapped.eql(two)")
	expect(t, empty.Eql(empty), "empty.eql(empty)")
	expect(t, !empty.Eql(one_first), "!empty.eql(one_first)")
	expect(t, !one_first.Eql(empty), "!one_first.eql(empty)")
}

// Go-only: panic test: ViewportSearch.needle: the window is not a forward window (assert)
func TestSaViewport_panic_ViewportSearch_needle_the_window_is_not_a_forward_window_assert(t *testing.T) {
	testStdCapacity(t)
	search, err := SearchViewportInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	search.window.direction = searchSlidingWindowDirectionReverse
	saExpectPanic(t, func() {
		_ = search.Needle()
	})
}

// Go-only: panic test: ViewportSearch.update: a soft-wrap overlap candidate page before the viewport with zero rows (`node.data.size.rows - 1` overflows)
func TestSaViewport_panic_ViewportSearch_update_a_soft_wrap_overlap_candidate_page_before_the_viewport_with_zero_rows_node_data_size_rows_1_overflows(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saFill(t, term, cap_+10, []saMark{{y: cap_ - 1, text: "xxxxxxxxTestZ"}})
	p1 := pages.pages.first
	p2 := pages.pages.last
	expect(t, p1.next == p2, "p1.next == p2")
	expectEqual(t, uint(10), uint(p2.data.size.rows))

	search, err := SearchViewportInit([]uint8("Test"))
	zigTry(t, err)
	defer search.Deinit()

	// The previous page of the viewport is the leading overlap candidate: its
	// last row is read with `size.rows - 1`, which is an integer overflow
	// when it has no rows.
	rows := p1.data.size.rows
	p1.data.size.rows = 0
	defer func() { p1.data.size.rows = rows }()
	saExpectPanic(t, func() {
		_, _ = search.Update(pages)
	})
}
