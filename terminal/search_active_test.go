// Tests translated one for one from src/terminal/search/active.zig (Ghostty
// v1.3.1 plus the term51 additional tests). Each Go test carries the exact
// Zig title in a "// Zig: test" comment. The helpers (saTerminal, saFill,
// saWrite, saNewlines, saCap, saScreen, saWrapLast, saMetaNode,
// saExpectHighlight, ...) are shared with the other search tests and are in
// search_test.go; see the translation notes there.
//
// Not translated (scripts/not-translated-sa.txt): the two out-of-memory
// tests (FailingAllocator).
//
// Go-only panic tests: ActiveSearch.update on a previous page with zero rows
// (`node.data.size.rows - 1` overflows in Zig).

package terminal

import "testing"

// Zig: test "simple search"
func TestSaActive_simple_search(t *testing.T) {
	testStdCapacity(t)
	term, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz\r\nBuzz\r\nFizz\r\nBang")))

	search, err := SearchActiveInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	_, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)

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
func TestSaActive_clear_screen_and_search(t *testing.T) {
	testStdCapacity(t)
	term, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz\r\nBuzz\r\nFizz\r\nBang")))

	search, err := SearchActiveInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	_, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)

	zigTry(t, s.NextSlice([]uint8("\x1b[2J"))) // Clear screen
	zigTry(t, s.NextSlice([]uint8("\x1b[H")))  // Move cursor home
	zigTry(t, s.NextSlice([]uint8("Buzz\r\nFizz\r\nBuzz")))
	_, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)

	{
		h := zigUnwrap(t, search.Next())
		sel := h.Untracked()
		expectEqual(t, saActivePoint(0, 1), zigUnwrap(t, term.screens.active.pages.PointFromPin(PointTagActive, sel.start)))
		expectEqual(t, saActivePoint(3, 1), zigUnwrap(t, term.screens.active.pages.PointFromPin(PointTagActive, sel.end)))
	}
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ActiveSearch.init: window is empty, forward and owns a copy of the needle"
func TestSaActive_term51_ActiveSearch_init_window_is_empty_forward_and_owns_a_copy_of_the_needle(t *testing.T) {
	testStdCapacity(t)
	buf := []uint8("Fizz")
	search, err := SearchActiveInit(buf)
	zigTry(t, err)
	defer search.Deinit()

	expectEqual(t, true, search.window.direction == searchSlidingWindowDirectionForward)
	expectEqualStrings(t, "Fizz", string(search.window.needle))
	expect(t, &search.window.needle[0] != &buf[0], "search.window.needle.ptr != @as([*]const u8, &buf)")
	expectEqual(t, uint(0), search.window.data.Len())
	expectEqual(t, uint(0), search.window.meta.Len())
	expectEqual(t, uint(0), search.window.data_offset)
	expectEqual(t, uint(8), uint(len(search.window.overlap_buf)))

	// Mutating the caller's buffer does not change the needle.
	buf[0] = 'X'
	expectEqualStrings(t, "Fizz", string(search.window.needle))
}

// Zig: test "term51: ActiveSearch.init: empty needle is accepted"
func TestSaActive_term51_ActiveSearch_init_empty_needle_is_accepted(t *testing.T) {
	testStdCapacity(t)
	search, err := SearchActiveInit([]uint8(""))
	zigTry(t, err)
	defer search.Deinit()
	expectEqual(t, uint(0), uint(len(search.window.needle)))
	expectEqual(t, uint(0), uint(len(search.window.overlap_buf)))

	// Updating a single page list with an empty needle works (the
	// overlap loop is never reached). Calling next() would panic on an
	// integer underflow in SlidingWindow.highlight, so it is not called.
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	saWrite(t, term, "Fizz")
	node, err := search.Update(&term.screens.active.pages)
	zigTry(t, err)
	expect(t, node == term.screens.active.pages.pages.first, "node == t.screens.active.pages.pages.first.?")
	expectEqual(t, uint(1), search.window.meta.Len())
}

// Zig: test "term51: ActiveSearch.deinit: frees a populated and an unpopulated search"
func TestSaActive_term51_ActiveSearch_deinit_frees_a_populated_and_an_unpopulated_search(t *testing.T) {
	testStdCapacity(t)
	{
		search, err := SearchActiveInit([]uint8("Fizz"))
		zigTry(t, err)
		search.Deinit()
	}

	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	saWrite(t, term, "Fizz\r\nBuzz\r\nFizz")
	search, err := SearchActiveInit([]uint8("Fizz"))
	zigTry(t, err)
	_, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	// Leave one match unconsumed.
	// Zig-only: The leak checker verifies the frees.
	_ = zigUnwrap(t, search.Next())
	search.Deinit()
}

// Zig: test "term51: ActiveSearch.update: single page returns that page and finds exact chunks"
func TestSaActive_term51_ActiveSearch_update_single_page_returns_that_page_and_finds_exact_chunks(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	saWrite(t, term, "Fizz\r\nBuzz\r\nFizz\r\nBang")
	p := term.screens.active.pages.pages.first

	search, err := SearchActiveInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	node, err := search.Update(&term.screens.active.pages)
	zigTry(t, err)
	expect(t, node == p, "node == p")
	expectEqual(t, uint(1), search.window.meta.Len())

	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p, start: 0, end: 1}})
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p, start: 2, end: 3}})
	expect(t, !search.Next().Valid, "search.next() == null")
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ActiveSearch.update: returns a non-null node even when the active area is the whole list"
func TestSaActive_term51_ActiveSearch_update_returns_a_non_null_node_even_when_the_active_area_is_the_whole_list(t *testing.T) {
	testStdCapacity(t)
	// The doc comment says null means "the active area covers the entire
	// PageList", but the loop always records the last node it appended,
	// so a non-empty list never returns null.
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	search, err := SearchActiveInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	node, err := search.Update(&term.screens.active.pages)
	zigTry(t, err)
	expect(t, node != nil, "node != null")
	expect(t, node == term.screens.active.pages.pages.first, "node.? == t.screens.active.pages.pages.first.?")
}

// Zig: test "term51: ActiveSearch.update: empty terminal has a window of one page and no matches"
func TestSaActive_term51_ActiveSearch_update_empty_terminal_has_a_window_of_one_page_and_no_matches(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	search, err := SearchActiveInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	_, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	expectEqual(t, uint(1), search.window.meta.Len())
	expectEqual(t, uint(1), search.window.data.Len())
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ActiveSearch.update: needle in the first cell and in the last cell"
func TestSaActive_term51_ActiveSearch_update_needle_in_the_first_cell_and_in_the_last_cell(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	p := term.screens.active.pages.pages.first
	saWrite(t, term, "Ab")
	saWrite(t, term, "\x1b[10;9HAb")

	search, err := SearchActiveInit([]uint8("ab"))
	zigTry(t, err)
	defer search.Deinit()
	_, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 1, []saChunk{{node: p, start: 0, end: 1}})
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 8, 9, []saChunk{{node: p, start: 9, end: 10}})
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ActiveSearch.update: one-character needle at the first and last cell clears the window when exhausted"
func TestSaActive_term51_ActiveSearch_update_one_character_needle_at_the_first_and_last_cell_clears_the_window_when_exhausted(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	p := term.screens.active.pages.pages.first
	saWrite(t, term, "x")
	saWrite(t, term, "\x1b[10;10Hx")

	search, err := SearchActiveInit([]uint8("X"))
	zigTry(t, err)
	defer search.Deinit()
	_, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 0, []saChunk{{node: p, start: 0, end: 1}})
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 9, 9, []saChunk{{node: p, start: 9, end: 10}})
	expectEqual(t, uint(1), search.window.meta.Len())
	expect(t, !search.Next().Valid, "search.next() == null")

	// The one-character special case dropped all buffered data.
	expectEqual(t, uint(0), search.window.meta.Len())
	expectEqual(t, uint(0), search.window.data.Len())
	expect(t, !search.Next().Valid, "search.next() == null")

	// Updating again refills the window.
	_, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	expectEqual(t, uint(1), search.window.meta.Len())
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 0, []saChunk{{node: p, start: 0, end: 1}})
}

// Zig: test "term51: ActiveSearch.update: matching is ASCII case-insensitive and overlapping matches are all reported"
func TestSaActive_term51_ActiveSearch_update_matching_is_ASCII_case_insensitive_and_overlapping_matches_are_all_reported(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	p := term.screens.active.pages.pages.first
	saWrite(t, term, "aAaA\r\nfIZZ")

	search, err := SearchActiveInit([]uint8("aa"))
	zigTry(t, err)
	defer search.Deinit()
	_, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 1, []saChunk{{node: p, start: 0, end: 1}})
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 1, 2, []saChunk{{node: p, start: 0, end: 1}})
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 2, 3, []saChunk{{node: p, start: 0, end: 1}})
	expect(t, !search.Next().Valid, "search.next() == null")

	search2, err := SearchActiveInit([]uint8("FIzz"))
	zigTry(t, err)
	defer search2.Deinit()
	_, err = search2.Update(&term.screens.active.pages)
	zigTry(t, err)
	saExpectHighlight(t, zigUnwrap(t, search2.Next()), 0, 3, []saChunk{{node: p, start: 1, end: 2}})
	expect(t, !search2.Next().Valid, "search2.next() == null")
}

// Zig: test "term51: ActiveSearch.update: needle longer than the screen text and absent needle find nothing"
func TestSaActive_term51_ActiveSearch_update_needle_longer_than_the_screen_text_and_absent_needle_find_nothing(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	saWrite(t, term, "Fizz")

	longer, err := SearchActiveInit([]uint8("FizzBuzzBangFizzBuzz"))
	zigTry(t, err)
	defer longer.Deinit()
	_, err = longer.Update(&term.screens.active.pages)
	zigTry(t, err)
	expect(t, !longer.Next().Valid, "longer.next() == null")

	absent, err := SearchActiveInit([]uint8("Buzz"))
	zigTry(t, err)
	defer absent.Deinit()
	_, err = absent.Update(&term.screens.active.pages)
	zigTry(t, err)
	expect(t, !absent.Next().Valid, "absent.next() == null")
}

// Zig: test "term51: ActiveSearch.update: soft-wrapped match is one chunk spanning two rows, hard newline breaks it"
func TestSaActive_term51_ActiveSearch_update_soft_wrapped_match_is_one_chunk_spanning_two_rows_hard_newline_breaks_it(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	p := term.screens.active.pages.pages.first
	// "abcdefgFiz" wraps to "z" on row 1; the second "Fiz" + "z" is split by a newline.
	saWrite(t, term, "abcdefgFizz\r\nabcdefgFiz\r\nz")

	search, err := SearchActiveInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	_, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 7, 0, []saChunk{{node: p, start: 0, end: 2}})
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ActiveSearch.update: byte offsets map to cells for wide characters"
func TestSaActive_term51_ActiveSearch_update_byte_offsets_map_to_cells_for_wide_characters(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	p := term.screens.active.pages.pages.first
	saWrite(t, term, "\xe6\x97\xa5Fizz")

	search, err := SearchActiveInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	_, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	// The wide character occupies x=0 and x=1.
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 2, 5, []saChunk{{node: p, start: 0, end: 1}})
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ActiveSearch.update: clears the previous window instead of accumulating"
func TestSaActive_term51_ActiveSearch_update_clears_the_previous_window_instead_of_accumulating(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	p := term.screens.active.pages.pages.first
	saWrite(t, term, "Fizz")

	search, err := SearchActiveInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	_, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	length := search.window.data.Len()
	_, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	_, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	expectEqual(t, uint(1), search.window.meta.Len())
	expectEqual(t, length, search.window.data.Len())
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p, start: 0, end: 1}})
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ActiveSearch.update: update after partial or full consumption restarts from the first match"
func TestSaActive_term51_ActiveSearch_update_update_after_partial_or_full_consumption_restarts_from_the_first_match(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	p := term.screens.active.pages.pages.first
	saWrite(t, term, "Fizz\r\nBuzz\r\nFizz")

	search, err := SearchActiveInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	_, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p, start: 0, end: 1}})
	expect(t, search.window.data_offset != 0, "search.window.data_offset != 0")

	// Partially consumed: update restarts.
	_, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	expectEqual(t, uint(0), search.window.data_offset)
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p, start: 0, end: 1}})
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p, start: 2, end: 3}})
	expect(t, !search.Next().Valid, "search.next() == null")

	// Fully consumed: update restarts.
	_, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p, start: 0, end: 1}})
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p, start: 2, end: 3}})
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ActiveSearch.update: results are a snapshot until the next update"
func TestSaActive_term51_ActiveSearch_update_results_are_a_snapshot_until_the_next_update(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	p := term.screens.active.pages.pages.first
	saWrite(t, term, "Fizz\r\nBuzz")

	search, err := SearchActiveInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	_, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)

	// Change the terminal without updating: the window still has the old copy.
	saWrite(t, term, "\x1b[2J\x1b[HNothing\r\nFIZZ")
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p, start: 0, end: 1}})
	expect(t, !search.Next().Valid, "search.next() == null")

	// After update the new contents are searched.
	_, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p, start: 1, end: 2}})
	expect(t, !search.Next().Valid, "search.next() == null")

	// Clearing and re-searching finds nothing.
	saWrite(t, term, "\x1b[2J\x1b[H")
	_, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ActiveSearch.update: history rows in the same page are searched as well"
func TestSaActive_term51_ActiveSearch_update_history_rows_in_the_same_page_are_searched_as_well(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	saWrite(t, term, "Fizz")
	saNewlines(t, term, 14)
	saWrite(t, term, "Fizz")
	expect(t, pages.pages.first == pages.pages.last, "pages.pages.first == pages.pages.last")
	expectEqual(t, uint(15), pages.total_rows)
	p := pages.pages.first

	search, err := SearchActiveInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	node, err := search.Update(pages)
	zigTry(t, err)
	expect(t, node == p, "node == p")

	// The first match is in history, i.e. outside of the active area.
	h1 := zigUnwrap(t, search.Next())
	saExpectHighlight(t, h1, 0, 3, []saChunk{{node: p, start: 0, end: 1}})
	expect(t, !pages.PointFromPin(PointTagActive, h1.Untracked().start).Valid, "pages.pointFromPin(.active, h1.untracked().start) == null")
	expectEqual(t, PointCoordinate{x: 0, y: 0}, saScreen(t, term, h1.Untracked().start))

	h2 := zigUnwrap(t, search.Next())
	saExpectHighlight(t, h2, 0, 3, []saChunk{{node: p, start: 14, end: 15}})
	expectEqual(t, saActivePoint(0, 9), zigUnwrap(t, pages.PointFromPin(PointTagActive, h2.Untracked().start)))
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ActiveSearch.update: active area across two pages appends pages last to first"
func TestSaActive_term51_ActiveSearch_update_active_area_across_two_pages_appends_pages_last_to_first(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)

	// Line 0 (history), line cap-9 (first active row, page one), line cap
	// (first row of page two).
	saWrite(t, term, "Fizz")
	saNewlines(t, term, cap_-9)
	saWrite(t, term, "Fizz")
	saNewlines(t, term, 9)
	saWrite(t, term, "Fizz")
	expect(t, pages.pages.first != pages.pages.last, "pages.pages.first != pages.pages.last")
	p1 := pages.pages.first
	p2 := pages.pages.last
	expect(t, p1.next == p2, "p1.next == p2")
	expectEqual(t, cap_, uint(p1.data.size.rows))
	expectEqual(t, uint(1), uint(p2.data.size.rows))

	search, err := SearchActiveInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	// The returned node is the earliest page that was appended.
	node, err := search.Update(pages)
	zigTry(t, err)
	expect(t, node == p1, "node == p1")
	expectEqual(t, uint(2), search.window.meta.Len())
	expect(t, saMetaNode(&search.window, 0) == p2, "term51ALMetaNode(&search.window, 0) == p2")
	expect(t, saMetaNode(&search.window, 1) == p1, "term51ALMetaNode(&search.window, 1) == p1")

	// The forward window holds [page two, page one], so the later page's
	// match comes first. Page one is searched entirely, including history
	// rows (line 0), in row order.
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p2, start: 0, end: 1}})
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p1, start: 0, end: 1}})
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p1, start: SizeCellCountInt(cap_ - 9), end: SizeCellCountInt(cap_ - 8)}})
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ActiveSearch.update: a match split across two pages inside the active area is not found"
func TestSaActive_term51_ActiveSearch_update_a_match_split_across_two_pages_inside_the_active_area_is_not_found(t *testing.T) {
	testStdCapacity(t)
	// The pages are appended in reverse order into a forward window, so a
	// match that wraps from the end of page one onto the start of page
	// two is never contiguous in the window.
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saNewlines(t, term, cap_-1)
	saWrite(t, term, "xxxxxxxxTest")
	expect(t, pages.pages.first != pages.pages.last, "pages.pages.first != pages.pages.last")
	p1 := pages.pages.first
	p2 := pages.pages.last
	expect(t, p1.data.GetRow(uint(p1.data.size.rows-1)).wrap, "p1.data.getRow(p1.data.size.rows - 1).wrap")

	// The same text is found by a forward window with the pages in order.
	ordered, err := SearchSlidingWindowInit(searchSlidingWindowDirectionForward, []uint8("Test"))
	zigTry(t, err)
	defer ordered.Deinit()
	_, err = ordered.Append(p1)
	zigTry(t, err)
	_, err = ordered.Append(p2)
	zigTry(t, err)
	saExpectHighlight(t, zigUnwrap(t, ordered.Next()), 8, 1, []saChunk{
		{node: p1, start: SizeCellCountInt(cap_ - 1), end: SizeCellCountInt(cap_)},
		{node: p2, start: 0, end: 1},
	})

	search, err := SearchActiveInit([]uint8("Test"))
	zigTry(t, err)
	defer search.Deinit()
	node, err := search.Update(pages)
	zigTry(t, err)
	expect(t, node == p1, "node == p1")
	expectEqual(t, uint(2), search.window.meta.Len())
	expect(t, !search.Next().Valid, "search.next() == null")

	// Pieces of the match are found separately.
	piece, err := SearchActiveInit([]uint8("st"))
	zigTry(t, err)
	defer piece.Deinit()
	_, err = piece.Update(pages)
	zigTry(t, err)
	saExpectHighlight(t, zigUnwrap(t, piece.Next()), 0, 1, []saChunk{{node: p2, start: 0, end: 1}})
	expect(t, !piece.Next().Valid, "piece.next() == null")
}

// Zig: test "term51: ActiveSearch.update: active area inside the last page does not add an unwrapped previous page"
func TestSaActive_term51_ActiveSearch_update_active_area_inside_the_last_page_does_not_add_an_unwrapped_previous_page(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saWrite(t, term, "Fizz")
	saNewlines(t, term, cap_+9)
	saWrite(t, term, "Fizz")
	expect(t, pages.pages.first != pages.pages.last, "pages.pages.first != pages.pages.last")
	p1 := pages.pages.first
	p2 := pages.pages.last
	expectEqual(t, uint(10), uint(p2.data.size.rows))
	expect(t, !p1.data.GetRow(uint(p1.data.size.rows-1)).wrap, "!p1.data.getRow(p1.data.size.rows - 1).wrap")

	search, err := SearchActiveInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	node, err := search.Update(pages)
	zigTry(t, err)
	expect(t, node == p2, "node == p2")
	expectEqual(t, uint(1), search.window.meta.Len())
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p2, start: 9, end: 10}})
	// The match in the previous page ("Fizz" on line 0) is not searched.
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ActiveSearch.update: active area inside the last page adds a soft-wrapped previous page after it"
func TestSaActive_term51_ActiveSearch_update_active_area_inside_the_last_page_adds_a_soft_wrapped_previous_page_after_it(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saWrite(t, term, "Fizz")
	saNewlines(t, term, cap_-1)
	saWrite(t, term, "xxxxxxxxTest")
	saNewlines(t, term, 9)
	p1 := pages.pages.first
	p2 := pages.pages.last
	expect(t, p1 != p2, "p1 != p2")
	expectEqual(t, uint(10), uint(p2.data.size.rows))
	expect(t, p1.data.GetRow(uint(p1.data.size.rows-1)).wrap, "p1.data.getRow(p1.data.size.rows - 1).wrap")

	search, err := SearchActiveInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	// The returned node is the last page that holds active rows, not the
	// overlap page.
	node, err := search.Update(pages)
	zigTry(t, err)
	expect(t, node == p2, "node == p2")
	expectEqual(t, uint(2), search.window.meta.Len())
	expect(t, saMetaNode(&search.window, 0) == p2, "term51ALMetaNode(&search.window, 0) == p2")
	expect(t, saMetaNode(&search.window, 1) == p1, "term51ALMetaNode(&search.window, 1) == p1")

	// The overlap page is searched entirely, including its history rows.
	h := zigUnwrap(t, search.Next())
	saExpectHighlight(t, h, 0, 3, []saChunk{{node: p1, start: 0, end: 1}})
	expect(t, !pages.PointFromPin(PointTagActive, h.Untracked().start).Valid, "pages.pointFromPin(.active, h.untracked().start) == null")
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ActiveSearch.update: overlap loop skips empty wrapped pages and stops at the first unwrapped page"
func TestSaActive_term51_ActiveSearch_update_overlap_loop_skips_empty_wrapped_pages_and_stops_at_the_first_unwrapped_page(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saWrite(t, term, "Fizz")
	saNewlines(t, term, 2*cap_+9)
	p1 := pages.pages.first
	p2 := p1.next
	p3 := pages.pages.last
	expect(t, p2.next == p3, "p2.next == p3")
	expectEqual(t, uint(10), uint(p3.data.size.rows))

	search, err := SearchActiveInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()

	// Nothing wrapped: only the last page.
	_, err = search.Update(pages)
	zigTry(t, err)
	expectEqual(t, uint(1), search.window.meta.Len())
	expect(t, !search.Next().Valid, "search.next() == null")

	// Page two wrapped but empty: it adds zero bytes so the loop goes on to
	// page one, which is not wrapped, so the loop stops there.
	saWrapLast(p2)
	node, err := search.Update(pages)
	zigTry(t, err)
	expect(t, node == p3, "node == p3")
	expectEqual(t, uint(1), search.window.meta.Len())
	expect(t, !search.Next().Valid, "search.next() == null")

	// Page one wrapped as well: its bytes are added (page two adds none and
	// has no meta of its own).
	saWrapLast(p1)
	_, err = search.Update(pages)
	zigTry(t, err)
	expectEqual(t, uint(2), search.window.meta.Len())
	expect(t, saMetaNode(&search.window, 0) == p3, "term51ALMetaNode(&search.window, 0) == p3")
	expect(t, saMetaNode(&search.window, 1) == p1, "term51ALMetaNode(&search.window, 1) == p1")
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: p1, start: 0, end: 1}})
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ActiveSearch.update: overlap loop compares the bytes of each single page, not the sum"
func TestSaActive_term51_ActiveSearch_update_overlap_loop_compares_the_bytes_of_each_single_page_not_the_sum(t *testing.T) {
	testStdCapacity(t)
	// Four pages. The first three are wrapped and hold 1, 1 and 2 bytes, each
	// fewer than needle-1 = 3, so the loop keeps going through all of them
	// although together they hold more than 3 bytes.
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
	expectEqual(t, uint(10), uint(p4.data.size.rows))
	saWrapLast(p1)
	saWrapLast(p2)
	saWrapLast(p3)

	search, err := SearchActiveInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	node, err := search.Update(pages)
	zigTry(t, err)
	expect(t, node == p4, "node == p4")
	expectEqual(t, uint(4), search.window.meta.Len())
	expect(t, saMetaNode(&search.window, 0) == p4, "term51ALMetaNode(&search.window, 0) == p4")
	expect(t, saMetaNode(&search.window, 1) == p3, "term51ALMetaNode(&search.window, 1) == p3")
	expect(t, saMetaNode(&search.window, 2) == p2, "term51ALMetaNode(&search.window, 2) == p2")
	expect(t, saMetaNode(&search.window, 3) == p1, "term51ALMetaNode(&search.window, 3) == p1")

	// A page with exactly needle-1 bytes stops the loop; with a one-byte
	// needle (needle-1 = 0) the first wrapped page stops it.
	one, err := SearchActiveInit([]uint8("z"))
	zigTry(t, err)
	defer one.Deinit()
	_, err = one.Update(pages)
	zigTry(t, err)
	expectEqual(t, uint(2), one.window.meta.Len())
	expect(t, saMetaNode(&one.window, 1) == p3, "term51ALMetaNode(&one.window, 1) == p3")

	two, err := SearchActiveInit([]uint8("zzz"))
	zigTry(t, err)
	defer two.Deinit()
	// needle-1 = 2: page three has exactly two bytes.
	_, err = two.Update(pages)
	zigTry(t, err)
	expectEqual(t, uint(2), two.window.meta.Len())
	expect(t, saMetaNode(&two.window, 1) == p3, "term51ALMetaNode(&two.window, 1) == p3")
}

// Zig: test "term51: ActiveSearch.update: active area across three pages appends all of them last to first"
func TestSaActive_term51_ActiveSearch_update_active_area_across_three_pages_appends_all_of_them_last_to_first(t *testing.T) {
	testStdCapacity(t)
	// Find a column count whose page capacity is small enough that 300 rows
	// need at least three pages.
	var cols SizeCellCountInt = 100
	for {
		probe := saTerminal(t, cols, 300)
		enough := probe.screens.active.pages.TotalPages() >= 3
		probe.Deinit()
		if enough {
			break
		}
		expect(t, cols < 8000, "cols < 8000")
		cols *= 2
	}
	term := saTerminal(t, cols, 300)
	defer term.Deinit()
	pages := &term.screens.active.pages
	expect(t, pages.TotalPages() >= 3, "pages.totalPages() >= 3")
	expectEqual(t, uint(300), pages.total_rows)
	saWrite(t, term, "\x1b[1;1HFizz\x1b[300;1HFizz")

	first := pages.pages.first
	last := pages.pages.last
	search, err := SearchActiveInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	// The whole list is the active area; the earliest page is returned.
	node, err := search.Update(pages)
	zigTry(t, err)
	expect(t, node == first, "node == first")
	expectEqual(t, pages.TotalPages(), search.window.meta.Len())
	it := pages.pages.last
	var i uint = 0
	for it != nil {
		expect(t, saMetaNode(&search.window, i) == it, "term51ALMetaNode(&search.window, i) == n")
		i += 1
		it = it.prev
	}

	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: last, start: last.data.size.rows - 1, end: last.data.size.rows}})
	saExpectHighlight(t, zigUnwrap(t, search.Next()), 0, 3, []saChunk{{node: first, start: 0, end: 1}})
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ActiveSearch.next: null before any update and after the matches are exhausted"
func TestSaActive_term51_ActiveSearch_next_null_before_any_update_and_after_the_matches_are_exhausted(t *testing.T) {
	testStdCapacity(t)
	search, err := SearchActiveInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	expect(t, !search.Next().Valid, "search.next() == null")
	expect(t, !search.Next().Valid, "search.next() == null")

	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	p := term.screens.active.pages.pages.first
	saWrite(t, term, "Fizz")
	_, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	h := zigUnwrap(t, search.Next())
	saExpectHighlight(t, h, 0, 3, []saChunk{{node: p, start: 0, end: 1}})
	// The same result is not repeated.
	expect(t, !search.Next().Valid, "search.next() == null")
	expect(t, !search.Next().Valid, "search.next() == null")
}

// Zig: test "term51: ActiveSearch.next: highlight can be cloned and survives the next call"
func TestSaActive_term51_ActiveSearch_next_highlight_can_be_cloned_and_survives_the_next_call(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	p := term.screens.active.pages.pages.first
	saWrite(t, term, "Fizz\r\nFizz")
	search, err := SearchActiveInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	_, err = search.Update(&term.screens.active.pages)
	zigTry(t, err)
	h1 := zigUnwrap(t, search.Next())
	first, err := h1.Clone()
	zigTry(t, err)
	defer first.Deinit()
	second := zigUnwrap(t, search.Next())
	saExpectHighlight(t, first, 0, 3, []saChunk{{node: p, start: 0, end: 1}})
	saExpectHighlight(t, second, 0, 3, []saChunk{{node: p, start: 1, end: 2}})
}

// Go-only: panic test: ActiveSearch.update: a wrapped-overlap candidate page with zero rows (`node.data.size.rows - 1` overflows)
func TestSaActive_panic_ActiveSearch_update_a_wrapped_overlap_candidate_page_with_zero_rows_node_data_size_rows_1_overflows(t *testing.T) {
	testStdCapacity(t)
	term := saTerminal(t, 10, 10)
	defer term.Deinit()
	pages := &term.screens.active.pages
	cap_ := saCap(term)
	saWrite(t, term, "Fizz")
	saNewlines(t, term, cap_-1)
	saWrite(t, term, "xxxxxxxxTest")
	saNewlines(t, term, 9)
	p1 := pages.pages.first
	p2 := pages.pages.last
	expect(t, p1 != p2, "p1 != p2")
	expectEqual(t, uint(10), uint(p2.data.size.rows))

	search, err := SearchActiveInit([]uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()

	// The previous page is the overlap candidate: its last row is read with
	// `size.rows - 1`, which is an integer overflow when it has no rows.
	rows := p1.data.size.rows
	p1.data.size.rows = 0
	defer func() { p1.data.size.rows = rows }()
	saExpectPanic(t, func() {
		_, _ = search.Update(pages)
	})
}
