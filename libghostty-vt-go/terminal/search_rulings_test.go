// Go-only tests for two project-lead rulings about the search code (no Zig
// test exists for either). Names are TestSearchRuling_..., helpers are
// prefixed sr.
//
// (A) Empty needle. Upstream crashes (ReleaseSafe: integer overflow in
// SlidingWindow.highlight or at active.zig:94 / viewport.zig:165; ReleaseFast:
// SIGSEGV in highlight) in ScreenSearch.init and everything it drives and in
// every next/drain of ActiveSearch, ViewportSearch, PageListSearch and
// SlidingWindow. The Go port treats an empty needle as a search with NO
// MATCHES: init succeeds, next returns none, the match counts are zero,
// Matches() is empty, selection finds nothing, SearchAll completes, Tick and
// Feed progress to complete without error. Where upstream's release build
// works (ActiveSearch.init/update, ViewportSearch.update,
// PageListSearch.init/feed, SlidingWindow.init/append) Go matches the
// recorded results (cases/search/empty_*.case, golden/search/empty_*.search.gz,
// docs/search-corpus.md); the corpus replay covers them and the tests here
// assert the same recorded outcomes again, with no panic.
//
// The tests replay the nine terminal states of the empty_* suites (blank,
// text, text_full, history, wrapped, nosb, pages2, pages3_wrap, pages_blank)
// and the op sequences of the 27 suites, one test per suite (the order of the
// ops is the order of harness/src/search.zig).
//
// (B) ScreenSearch.feed on a completed search. Upstream's
// `.complete => unreachable` arm (reached when the history searcher still has
// a page to feed while state is .complete: after a history prune
// reloadActive builds a new history searcher, finding 20 of
// docs/upstream-findings.md; the 48 aborting cases of
// cases/search/changes_prune.case). Go returns ErrSearchComplete, exactly what
// Tick returns in that state, and changes NOTHING: state, matches, history
// searcher and selection are untouched. The tests translate the situation of
// those 48 cases.
//
// Page capacity. Every test calls testStdCapacity first (as the other search
// tests do). The recorded numbers of both rulings come from the production
// page capacity (grapheme_bytes 8192: 46 rows per page at 1000 columns, 23 at
// 2000), so the scenario helpers then call srProductionCapacity, which sets
// grapheme_bytes back to 8192 for the rest of the test; its cleanup runs
// before testStdCapacity's. The page layout of each state is checked
// against the recorded layout lines.

package terminal

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// srProductionCapacity sets the standard page capacity back to production's
// grapheme_bytes 8192 (see the file header). Call it after testStdCapacity.
func srProductionCapacity(t *testing.T) {
	t.Helper()
	old := PageStdCapacity.grapheme_bytes
	PageStdCapacity.grapheme_bytes = 8192
	t.Cleanup(func() { PageStdCapacity.grapheme_bytes = old })
}

// srRecover turns a panic (also the wireframe's "not implemented") into a
// failure of the one test instead of aborting the whole test binary: the
// rulings say nothing here may panic. Use as `defer srRecover(t)`.
func srRecover(t *testing.T) {
	t.Helper()
	if r := recover(); r != nil {
		t.Fatalf("panic (the ruling is: no panic): %v", r)
	}
}

// srWrite is `repeat n data` of the case files (`write` is n = 1).
type srWrite struct {
	n    uint
	data string
}

// srWin is the recorded `win` line of a sliding window: bytes in data and the
// cell_map length of every meta (first to last). data_offset is 0 in every
// recorded state.
type srWin struct {
	data  uint
	metas []uint
}

// srState is one of the nine terminal states of the empty_* suites with the
// outcomes recorded in the golden files.
type srState struct {
	name     string
	cols     SizeCellCountInt
	rows     SizeCellCountInt
	sb       uint64
	writes   []srWrite
	pageRows []uint // recorded layout: rows in use of every page

	activeNode uint     // empty_active_update: `ret node <i>`
	active     srWin    // window after ActiveSearch.update
	viewport   srWin    // window after ViewportSearch.update (ret bool true)
	pagelist   srWin    // window after PageListSearch.init (start=last)
	feeds      [2]srWin // windows after the first and the second feed (ret bool false)
	appendAll  srWin    // forward window after append of every page first..last; data = `ret bytes`
}

// srOne is a single-page state: every recorded window is the same one meta
// of `bytes` cells.
func srOne(name string, sb uint64, writes []srWrite, pageRows uint, bytes uint) srState {
	w := srWin{data: bytes, metas: []uint{bytes}}
	return srState{
		name:       name,
		cols:       10,
		rows:       5,
		sb:         sb,
		writes:     writes,
		pageRows:   []uint{pageRows},
		activeNode: 0,
		active:     w,
		viewport:   w,
		pagelist:   w,
		feeds:      [2]srWin{w, w},
		appendAll:  w,
	}
}

// srHistoryText is the haystack "line 0" .. "line 11", "end".
func srHistoryText() string {
	var b strings.Builder
	for i := 0; i < 12; i++ {
		fmt.Fprintf(&b, "line %d\r\n", i)
	}
	b.WriteString("end")
	return b.String()
}

var srStates = []srState{
	srOne("blank", 10000, nil, 5, 1),
	srOne("text", 10000, []srWrite{{1, "Fizz\r\nBuzz"}}, 5, 10),
	srOne("text_full", 10000, []srWrite{{1, "Fizz\r\nBuzz\r\nFizz\r\nBang\r\nbaz"}}, 5, 24),
	srOne("history", 10000, []srWrite{{1, srHistoryText()}}, 13, 90),
	srOne("wrapped", 10000, []srWrite{{1, strings.Repeat("a", 37)}}, 5, 38),
	srOne("nosb", 0, []srWrite{{1, srHistoryText()}}, 5, 34),
	{
		name: "pages2", cols: 1000, rows: 4, sb: 10000000,
		writes:     []srWrite{{3, "\r\n"}, {1, "ab"}, {47, "\r\n"}, {1, "cd"}},
		pageRows:   []uint{46, 5},
		activeNode: 1,
		active:     srWin{7, []uint{7}},
		viewport:   srWin{7, []uint{7}},
		pagelist:   srWin{7, []uint{7}},
		feeds:      [2]srWin{{13, []uint{7, 6}}, {13, []uint{7, 6}}},
		appendAll:  srWin{13, []uint{6, 7}},
	},
	{
		// A soft-wrapped last row (10 rows of 100 "x" plus one) at the end of
		// page 0, then a small page: the case that wraps `needle.len - 1`
		// in upstream's ActiveSearch/ViewportSearch.update.
		name: "pages3_wrap", cols: 1000, rows: 4, sb: 10000000,
		writes:     []srWrite{{45, "\r\n"}, {10, strings.Repeat("x", 100)}, {1, "x"}, {46, "\r\n"}, {1, "y"}},
		pageRows:   []uint{46, 46, 1},
		activeNode: 1,
		active:     srWin{1049, []uint{2, 2, 1045}},
		viewport:   srWin{1049, []uint{1045, 2, 2}},
		pagelist:   srWin{2, []uint{2}},
		feeds:      [2]srWin{{4, []uint{2, 2}}, {1049, []uint{2, 2, 1045}}},
		appendAll:  srWin{1049, []uint{1045, 2, 2}},
	},
	{
		name: "pages_blank", cols: 1000, rows: 4, sb: 10000000,
		writes:     []srWrite{{100, "\r\n"}},
		pageRows:   []uint{46, 46, 9},
		activeNode: 2,
		active:     srWin{1, []uint{1}},
		viewport:   srWin{1, []uint{1}},
		pagelist:   srWin{1, []uint{1}},
		feeds:      [2]srWin{{2, []uint{1, 1}}, {3, []uint{1, 1, 1}}},
		appendAll:  srWin{3, []uint{1, 1, 1}},
	},
}

// srBuild is the terminal of a state: init, then the writes through the
// terminal's stream, then a check of the recorded page layout.
func srBuild(t *testing.T, st *srState) *Terminal {
	t.Helper()
	term := ss1InitScrollback(t, st.cols, st.rows, st.sb)
	s := term.VtStream()
	defer s.Deinit()
	for _, w := range st.writes {
		for i := uint(0); i < w.n; i++ {
			zigTry(t, s.NextSlice([]uint8(w.data)))
		}
	}
	var got []uint
	for n := term.Screens.Active.Pages.pages.first; n != nil; n = n.next {
		got = append(got, uint(n.data.size.rows))
	}
	expectEqualSlices(t, st.pageRows, got)
	return term
}

// srEachState runs f once per state, each in a subtest with its own
// terminal. A panic fails the subtest only.
func srEachState(t *testing.T, f func(t *testing.T, st *srState, term *Terminal)) {
	t.Helper()
	for i := range srStates {
		st := &srStates[i]
		t.Run(st.name, func(t *testing.T) {
			testStdCapacity(t)
			srProductionCapacity(t)
			defer srRecover(t)
			term := srBuild(t, st)
			defer term.Deinit()
			f(t, st, term)
		})
	}
}

// srList is the page list of the terminal's active screen.
func srList(term *Terminal) *PageList {
	return &term.Screens.Active.Pages
}

// srNodeIndex is the harness's pageIndex: the index of node in the list, -1
// if it is not in it.
func srNodeIndex(list *PageList, node *PageListNode) int {
	i := 0
	for n := list.pages.first; n != nil; n = n.next {
		if n == node {
			return i
		}
		i++
	}
	return -1
}

// srMetaNodes are the nodes of the metas of a window, first to last.
func srMetaNodes(w *SearchSlidingWindow) []*PageListNode {
	var out []*PageListNode
	it := w.meta.Iterator(DatastructCircBufIteratorDirectionForward)
	for {
		m := it.Next()
		if m == nil {
			return out
		}
		out = append(out, m.node)
	}
}

// srExpectWin asserts the recorded `win` line of a window: data_len,
// data_offset 0 and the cell_map length of every meta.
func srExpectWin(t *testing.T, w *SearchSlidingWindow, want srWin) {
	t.Helper()
	expectEqual(t, want.data, w.data.Len())
	expectEqual(t, uint(0), w.data_offset)
	expectEqual(t, uint(len(want.metas)), w.meta.Len())
	var got []uint
	it := w.meta.Iterator(DatastructCircBufIteratorDirectionForward)
	for {
		m := it.Next()
		if m == nil {
			break
		}
		got = append(got, uint(len(m.cell_map)))
	}
	expectEqualSlices(t, want.metas, got)
}

// srDrain is the harness's `drain`: next() until null, at most limit; it
// returns the number of hits.
func srDrain(next func() Optional[HighlightFlattened], limit uint) uint {
	var count uint
	for count < limit {
		if !next().Valid {
			break
		}
		count++
	}
	return count
}

// srForward and srReverse are the nodes of a list first..last and last..first.
func srForward(list *PageList) []*PageListNode {
	var out []*PageListNode
	for n := list.pages.first; n != nil; n = n.next {
		out = append(out, n)
	}
	return out
}

func srReverse(list *PageList) []*PageListNode {
	var out []*PageListNode
	for n := list.pages.last; n != nil; n = n.prev {
		out = append(out, n)
	}
	return out
}

// srOpenScreen is `open screen -`: ScreenSearch.init with the empty needle.
func srOpenScreen(t *testing.T, term *Terminal) *SearchScreen {
	t.Helper()
	search, err := SearchScreenInit(term.Screens.Active, []uint8(""))
	zigTry(t, err)
	return &search
}

// srExpectNoMatches asserts the empty-needle ruling for a ScreenSearch: no
// matches, nothing selected.
func srExpectNoMatches(t *testing.T, search *SearchScreen) {
	t.Helper()
	expectEqual(t, uint(0), search.MatchesLen())
	expectEqual(t, uint(0), uint(len(search.active_results)))
	expectEqual(t, uint(0), uint(len(search.history_results)))
	ms, err := search.Matches()
	zigTry(t, err)
	expectEqual(t, uint(0), uint(len(ms)))
	expect(t, !search.SelectedMatch().Valid, "search.selectedMatch() == null")
}

// srTickToComplete is the tick/feed loop of searchAll written out: the first
// tick (state .active) and every step returns no error other than
// FeedRequired (then feed) and, at the end, SearchComplete.
func srTickToComplete(t *testing.T, search *SearchScreen) {
	t.Helper()
	zigTry(t, search.Tick())
	for i := 0; ; i++ {
		expect(t, i < 10000, "the tick/feed loop reaches complete")
		err := search.Tick()
		if err == nil {
			continue
		}
		if errors.Is(err, ErrFeedRequired) {
			zigTry(t, search.Feed())
			continue
		}
		ss1ExpectError(t, ErrSearchComplete, err)
		break
	}
	expectEqual(t, searchScreenStateComplete, search.state)
}

// ---------------------------------------------------------------------------
// (A) empty needle, entry points that work upstream: recorded outcomes
// ---------------------------------------------------------------------------

// Go-only: ruling (empty needle: recorded outcome of upstream's release build, empty_active_init; init accepts it)
func TestSearchRuling_empty_needle_active_init(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		search, err := SearchActiveInit([]uint8(""))
		zigTry(t, err)
		defer search.Deinit()
		expectEqual(t, uint(0), uint(len(search.window.needle)))
		expect(t, search.window.direction == searchSlidingWindowDirectionForward, "window.direction == .forward")
		srExpectWin(t, &search.window, srWin{})
	})
}

// Go-only: ruling (empty needle: recorded outcome of upstream's release build, empty_active_update; pages3_wrap is the unchecked_only case where needle.len - 1 wraps)
func TestSearchRuling_empty_needle_active_update(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		search, err := SearchActiveInit([]uint8(""))
		zigTry(t, err)
		defer search.Deinit()
		srExpectWin(t, &search.window, srWin{})
		node, err := search.Update(srList(term))
		zigTry(t, err)
		expectEqual(t, int(st.activeNode), srNodeIndex(srList(term), node))
		srExpectWin(t, &search.window, st.active)
	})
}

// Go-only: ruling (empty needle: recorded outcome of upstream's release build, empty_viewport_update; pages3_wrap is the unchecked_only case where needle.len - 1 wraps)
func TestSearchRuling_empty_needle_viewport_update(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		search, err := SearchViewportInit([]uint8(""))
		zigTry(t, err)
		defer search.Deinit()
		srExpectWin(t, &search.window, srWin{})
		changed, err := search.Update(srList(term))
		zigTry(t, err)
		expect(t, changed, "try viewport.update(list)")
		srExpectWin(t, &search.window, st.viewport)
	})
}

// Go-only: ruling (empty needle: recorded outcome of upstream's release build, empty_pagelist_init; init appends the start page)
func TestSearchRuling_empty_needle_pagelist_init(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		list := srList(term)
		search, err := SearchPagelistInit([]uint8(""), list, list.pages.last)
		zigTry(t, err)
		defer search.Deinit()
		expect(t, search.window.direction == searchSlidingWindowDirectionReverse, "window.direction == .reverse")
		srExpectWin(t, &search.window, st.pagelist)
	})
}

// Go-only: ruling (empty needle: recorded outcome of upstream's release build, empty_pagelist_feed; feed appends one page per call and returns false)
func TestSearchRuling_empty_needle_pagelist_feed(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		list := srList(term)
		search, err := SearchPagelistInit([]uint8(""), list, list.pages.last)
		zigTry(t, err)
		defer search.Deinit()
		for i := range st.feeds {
			fed, err := search.Feed()
			zigTry(t, err)
			expect(t, !fed, "try pagelist.feed() == false")
			srExpectWin(t, &search.window, st.feeds[i])
		}
	})
}

// Go-only: ruling (empty needle: recorded outcome of upstream's release build, empty_window_init; init accepts it)
func TestSearchRuling_empty_needle_window_init(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		w, err := SearchSlidingWindowInit(searchSlidingWindowDirectionForward, []uint8(""))
		zigTry(t, err)
		defer w.Deinit()
		expectEqual(t, uint(0), uint(len(w.needle)))
		srExpectWin(t, &w, srWin{})
	})
}

// Go-only: ruling (empty needle: recorded outcome of upstream's release build, empty_window_append; append accepts it and returns the bytes of the page)
func TestSearchRuling_empty_needle_window_append(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		w, err := SearchSlidingWindowInit(searchSlidingWindowDirectionForward, []uint8(""))
		zigTry(t, err)
		defer w.Deinit()
		var total uint
		for _, node := range srForward(srList(term)) {
			n, err := w.Append(node)
			zigTry(t, err)
			total += n
		}
		expectEqual(t, st.appendAll.data, total)
		srExpectWin(t, &w, st.appendAll)
	})
}

// ---------------------------------------------------------------------------
// (A) empty needle, entry points that crash upstream: no matches
// ---------------------------------------------------------------------------

// Go-only: ruling (empty needle: searching for nothing finds nothing; upstream overflows/segfaults here, docs/upstream-crashes.md)
func TestSearchRuling_empty_needle_active_next(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		search, err := SearchActiveInit([]uint8(""))
		zigTry(t, err)
		defer search.Deinit()
		_, err = search.Update(srList(term))
		zigTry(t, err)
		expect(t, !search.Next().Valid, "active.next() == null")
		expect(t, !search.Next().Valid, "active.next() == null (again)")
	})
}

// Go-only: ruling (empty needle: searching for nothing finds nothing; upstream overflows/segfaults here, docs/upstream-crashes.md)
func TestSearchRuling_empty_needle_active_drain(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		search, err := SearchActiveInit([]uint8(""))
		zigTry(t, err)
		defer search.Deinit()
		_, err = search.Update(srList(term))
		zigTry(t, err)
		expectEqual(t, uint(0), srDrain(search.Next, 50))
	})
}

// Go-only: ruling (empty needle: searching for nothing finds nothing; upstream overflows/segfaults here, docs/upstream-crashes.md)
func TestSearchRuling_empty_needle_viewport_next(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		search, err := SearchViewportInit([]uint8(""))
		zigTry(t, err)
		defer search.Deinit()
		_, err = search.Update(srList(term))
		zigTry(t, err)
		expect(t, !search.Next().Valid, "viewport.next() == null")
		expect(t, !search.Next().Valid, "viewport.next() == null (again)")
	})
}

// Go-only: ruling (empty needle: searching for nothing finds nothing; upstream overflows/segfaults here, docs/upstream-crashes.md)
func TestSearchRuling_empty_needle_viewport_drain(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		search, err := SearchViewportInit([]uint8(""))
		zigTry(t, err)
		defer search.Deinit()
		_, err = search.Update(srList(term))
		zigTry(t, err)
		expectEqual(t, uint(0), srDrain(search.Next, 50))
	})
}

// Go-only: ruling (empty needle: searching for nothing finds nothing; upstream overflows/segfaults here, docs/upstream-crashes.md)
func TestSearchRuling_empty_needle_pagelist_next(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		list := srList(term)
		search, err := SearchPagelistInit([]uint8(""), list, list.pages.last)
		zigTry(t, err)
		defer search.Deinit()
		expect(t, !search.Next().Valid, "pagelist.next() == null")
		expect(t, !search.Next().Valid, "pagelist.next() == null (again)")
	})
}

// Go-only: ruling (empty needle: searching for nothing finds nothing; upstream overflows/segfaults here, docs/upstream-crashes.md)
func TestSearchRuling_empty_needle_pagelist_drain(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		list := srList(term)
		search, err := SearchPagelistInit([]uint8(""), list, list.pages.last)
		zigTry(t, err)
		defer search.Deinit()
		expectEqual(t, uint(0), srDrain(search.Next, 50))
	})
}

// Go-only: ruling (empty needle: searching for nothing finds nothing; upstream overflows/segfaults here, docs/upstream-crashes.md)
func TestSearchRuling_empty_needle_pagelist_feed_next(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		list := srList(term)
		search, err := SearchPagelistInit([]uint8(""), list, list.pages.last)
		zigTry(t, err)
		defer search.Deinit()
		fed, err := search.Feed()
		zigTry(t, err)
		expect(t, !fed, "try pagelist.feed() == false")
		expect(t, !search.Next().Valid, "pagelist.next() == null")
	})
}

// Go-only: ruling (empty needle: searching for nothing finds nothing; upstream overflows/segfaults here, docs/upstream-crashes.md)
func TestSearchRuling_empty_needle_window_next(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		w, err := SearchSlidingWindowInit(searchSlidingWindowDirectionForward, []uint8(""))
		zigTry(t, err)
		defer w.Deinit()
		for _, node := range srForward(srList(term)) {
			_, err := w.Append(node)
			zigTry(t, err)
		}
		expect(t, !w.Next().Valid, "window.next() == null")
		expect(t, !w.Next().Valid, "window.next() == null (again)")
	})
}

// Go-only: ruling (empty needle: searching for nothing finds nothing; upstream overflows/segfaults here, docs/upstream-crashes.md)
func TestSearchRuling_empty_needle_window_next_empty(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		// No page was appended: the window has no data and no meta (upstream
		// takes the `unreachable` at sliding_window.zig:332 here).
		w, err := SearchSlidingWindowInit(searchSlidingWindowDirectionForward, []uint8(""))
		zigTry(t, err)
		defer w.Deinit()
		expect(t, !w.Next().Valid, "window.next() == null")
		expect(t, !w.Next().Valid, "window.next() == null (again)")
		srExpectWin(t, &w, srWin{})
	})
}

// Go-only: ruling (empty needle: searching for nothing finds nothing; upstream overflows/segfaults here, docs/upstream-crashes.md)
func TestSearchRuling_empty_needle_window_drain(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		w, err := SearchSlidingWindowInit(searchSlidingWindowDirectionForward, []uint8(""))
		zigTry(t, err)
		defer w.Deinit()
		for _, node := range srForward(srList(term)) {
			_, err := w.Append(node)
			zigTry(t, err)
		}
		expectEqual(t, uint(0), srDrain(w.Next, 50))
	})
}

// Go-only: ruling (empty needle: searching for nothing finds nothing; upstream overflows/segfaults here, docs/upstream-crashes.md)
func TestSearchRuling_empty_needle_window_rev_next(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		w, err := SearchSlidingWindowInit(searchSlidingWindowDirectionReverse, []uint8(""))
		zigTry(t, err)
		defer w.Deinit()
		for _, node := range srReverse(srList(term)) {
			_, err := w.Append(node)
			zigTry(t, err)
		}
		expect(t, !w.Next().Valid, "window.next() == null")
		expect(t, !w.Next().Valid, "window.next() == null (again)")
	})
}

// Go-only: ruling (empty needle: searching for nothing finds nothing; upstream overflows/segfaults here, docs/upstream-crashes.md)
func TestSearchRuling_empty_needle_window_rev_drain(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		w, err := SearchSlidingWindowInit(searchSlidingWindowDirectionReverse, []uint8(""))
		zigTry(t, err)
		defer w.Deinit()
		for _, node := range srReverse(srList(term)) {
			_, err := w.Append(node)
			zigTry(t, err)
		}
		expectEqual(t, uint(0), srDrain(w.Next, 50))
	})
}

// Go-only: ruling (empty needle: searching for nothing finds nothing; upstream overflows/segfaults here, docs/upstream-crashes.md)
func TestSearchRuling_empty_needle_screen_init(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		search := srOpenScreen(t, term)
		defer search.Deinit()
		expectEqual(t, uint(0), uint(len(search.Needle())))
		expectEqual(t, st.rows, search.rows)
		expectEqual(t, st.cols, search.cols)
		srExpectNoMatches(t, search)
	})
}

// Go-only: ruling (empty needle: searching for nothing finds nothing; upstream overflows/segfaults here, docs/upstream-crashes.md)
func TestSearchRuling_empty_needle_screen_search_all(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		search := srOpenScreen(t, term)
		defer search.Deinit()
		zigTry(t, search.SearchAll())
		// searchAll returns only on SearchComplete.
		expectEqual(t, searchScreenStateComplete, search.state)
		srExpectNoMatches(t, search)
		// Complete: another searchAll returns at once, still nothing.
		zigTry(t, search.SearchAll())
		srExpectNoMatches(t, search)
	})
}

// Go-only: ruling (empty needle: searching for nothing finds nothing; upstream overflows/segfaults here, docs/upstream-crashes.md)
func TestSearchRuling_empty_needle_screen_tick(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		search := srOpenScreen(t, term)
		defer search.Deinit()
		srTickToComplete(t, search)
		srExpectNoMatches(t, search)
		// Complete: tick keeps returning SearchComplete.
		ss1ExpectError(t, ErrSearchComplete, search.Tick())
	})
}

// Go-only: ruling (empty needle: searching for nothing finds nothing; upstream overflows/segfaults here, docs/upstream-crashes.md)
func TestSearchRuling_empty_needle_screen_feed(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		search := srOpenScreen(t, term)
		defer search.Deinit()
		// Feed (on a search that is not complete) makes progress to complete
		// without an error. It is not called again once complete.
		for i := 0; !search.state.IsComplete(); i++ {
			expect(t, i < 10000, "feed reaches complete")
			zigTry(t, search.Feed())
		}
		srExpectNoMatches(t, search)
	})
}

// Go-only: ruling (empty needle: searching for nothing finds nothing; upstream overflows/segfaults here, docs/upstream-crashes.md)
func TestSearchRuling_empty_needle_screen_select_next(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		search := srOpenScreen(t, term)
		defer search.Deinit()
		moved, err := search.Select(SearchScreenSelectNext)
		zigTry(t, err)
		expect(t, !moved, "try search.select(.next) == false")
		srExpectNoMatches(t, search)
		// Also after the whole screen was searched.
		zigTry(t, search.SearchAll())
		moved, err = search.Select(SearchScreenSelectNext)
		zigTry(t, err)
		expect(t, !moved, "try search.select(.next) == false (after searchAll)")
		srExpectNoMatches(t, search)
	})
}

// Go-only: ruling (empty needle: searching for nothing finds nothing; upstream overflows/segfaults here, docs/upstream-crashes.md)
func TestSearchRuling_empty_needle_screen_select_prev(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		search := srOpenScreen(t, term)
		defer search.Deinit()
		moved, err := search.Select(SearchScreenSelectPrev)
		zigTry(t, err)
		expect(t, !moved, "try search.select(.prev) == false")
		srExpectNoMatches(t, search)
		// Also after the whole screen was searched.
		zigTry(t, search.SearchAll())
		moved, err = search.Select(SearchScreenSelectPrev)
		zigTry(t, err)
		expect(t, !moved, "try search.select(.prev) == false (after searchAll)")
		srExpectNoMatches(t, search)
	})
}

// Go-only: ruling (empty needle: searching for nothing finds nothing; upstream overflows/segfaults here, docs/upstream-crashes.md)
func TestSearchRuling_empty_needle_screen_matches(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		search := srOpenScreen(t, term)
		defer search.Deinit()
		ms, err := search.Matches()
		zigTry(t, err)
		expectEqual(t, uint(0), uint(len(ms)))
		expectEqual(t, uint(0), search.MatchesLen())
		expect(t, !search.SelectedMatch().Valid, "search.selectedMatch() == null")
	})
}

// Go-only: ruling (empty needle: searching for nothing finds nothing; upstream overflows/segfaults here, docs/upstream-crashes.md)
func TestSearchRuling_empty_needle_screen_reload(t *testing.T) {
	testStdCapacity(t)
	srEachState(t, func(t *testing.T, st *srState, term *Terminal) {
		search := srOpenScreen(t, term)
		defer search.Deinit()
		zigTry(t, search.ReloadActive())
		srExpectNoMatches(t, search)
		zigTry(t, search.ReloadActive())
		srExpectNoMatches(t, search)
	})
}

// ---------------------------------------------------------------------------
// (B) ScreenSearch.feed on a completed search
// ---------------------------------------------------------------------------

// srHL is a flattened highlight copied out (the stored highlights own their
// chunk slices; a copy keeps the snapshot independent).
type srHL struct {
	top_x  SizeCellCountInt
	bot_x  SizeCellCountInt
	chunks []HighlightFlattenedChunk
}

func srHLOf(h HighlightFlattened) srHL {
	return srHL{top_x: h.TopX, bot_x: h.BotX, chunks: append([]HighlightFlattenedChunk(nil), h.Chunks...)}
}

func srHLs(hs []HighlightFlattened) []srHL {
	out := make([]srHL, len(hs))
	for i, h := range hs {
		out[i] = srHLOf(h)
	}
	return out
}

// srSel is a selected match: its index, its tracked pins (identity and
// current value).
type srSel struct {
	idx      uint
	start    *PageListPin
	end      *PageListPin
	startVal PageListPin
	endVal   PageListPin
}

// srHist is the history searcher: its window, its tracked pin and the
// start pin.
type srHist struct {
	data       uint
	offset     uint
	metas      uint
	metaNodes  []*PageListNode
	pin        *PageListPin
	pinVal     PageListPin
	startPin   *PageListPin
	startPinVl PageListPin
	needleLen  uint
}

// srSnap is everything a ScreenSearch holds that feed on a completed search
// must leave alone.
type srSnap struct {
	state         searchScreenState
	rows          SizeCellCountInt
	cols          SizeCellCountInt
	total         uint
	matches       []srHL
	activeResults []srHL
	histResults   []srHL
	selected      Optional[srSel]
	selectedMatch Optional[srHL]
	history       Optional[srHist]
	actData       uint
	actOffset     uint
	actMetas      uint
}

func srSnapshot(t *testing.T, search *SearchScreen) srSnap {
	t.Helper()
	ms, err := search.Matches()
	zigTry(t, err)
	snap := srSnap{
		state:         search.state,
		rows:          search.rows,
		cols:          search.cols,
		total:         search.MatchesLen(),
		matches:       srHLs(ms),
		activeResults: srHLs(search.active_results),
		histResults:   srHLs(search.history_results),
		actData:       search.active.window.data.Len(),
		actOffset:     search.active.window.data_offset,
		actMetas:      search.active.window.meta.Len(),
	}
	if search.selected.Valid {
		m := &search.selected.Value
		snap.selected = Some(srSel{
			idx:      m.Idx,
			start:    m.Highlight.Start,
			end:      m.Highlight.End,
			startVal: *m.Highlight.Start,
			endVal:   *m.Highlight.End,
		})
	}
	if sm := search.SelectedMatch(); sm.Valid {
		snap.selectedMatch = Some(srHLOf(sm.Value))
	}
	if search.history.Valid {
		h := &search.history.Value
		w := &h.searcher.window
		snap.history = Some(srHist{
			data:       w.data.Len(),
			offset:     w.data_offset,
			metas:      w.meta.Len(),
			metaNodes:  srMetaNodes(w),
			pin:        h.searcher.pin,
			pinVal:     *h.searcher.pin,
			startPin:   h.start_pin,
			startPinVl: *h.start_pin,
			needleLen:  uint(len(w.needle)),
		})
	}
	return snap
}

// srExpectSame fails if the search changed since the snapshot.
func srExpectSame(t *testing.T, what string, want srSnap, search *SearchScreen) {
	t.Helper()
	got := srSnapshot(t, search)
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("%s changed the search:\nbefore %+v\nafter  %+v", what, want, got)
	}
}

// srPrune is a situation of cases/search/changes_prune.case (the 48 cases that
// abort upstream in feed: select, enough new lines to prune pages,
// update_active, then feed).
type srPrune struct {
	cols   SizeCellCountInt
	sb     uint64 // max_scrollback bytes
	nl     uint   // newlines between the three "fizz page N" lines (rows per page - 1 at this width)
	needle string
	sel    SearchScreenSelect
	more   uint // lines of "fizz tail" written after the selection
}

// srFeedComplete runs the situation: three pages of text, search all, select,
// write `more` lines (pruning the oldest pages), reloadActive, matches. The
// search is then complete with a fresh history searcher that still has a
// page to feed. Feed must return ErrSearchComplete and change nothing.
func srFeedComplete(t *testing.T, c srPrune) {
	t.Helper()
	srProductionCapacity(t)
	term := ss1InitScrollback(t, c.cols, 4, c.sb)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	write := func(n uint, data string) {
		for i := uint(0); i < n; i++ {
			zigTry(t, s.NextSlice([]uint8(data)))
		}
	}
	write(2, "\r\n")
	write(1, "fizz page 0")
	write(c.nl, "\r\n")
	write(1, "fizz page 1")
	write(c.nl, "\r\n")
	write(1, "fizz page 2")

	search, err := SearchScreenInit(term.Screens.Active, []uint8(c.needle))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	_, err = search.Select(c.sel)
	zigTry(t, err)
	write(c.more, "fizz tail\r\n")
	zigTry(t, search.ReloadActive())
	_, err = search.Matches()
	zigTry(t, err)

	// The situation: complete, with a history searcher (built by reloadActive
	// after the prune) that can feed one more page.
	expectEqual(t, searchScreenStateComplete, search.state)
	expect(t, search.history.Valid, "search.history != null")
	h := &search.history.Value
	expect(t, !h.searcher.pin.Garbage, "!history.searcher.pin.garbage")
	expect(t, h.searcher.pin.Node.prev != nil, "history.searcher.pin.node.prev != null (feed has a page to add)")

	before := srSnapshot(t, &search)

	ss1ExpectError(t, ErrSearchComplete, search.Feed())
	srExpectSame(t, "feed", before, &search)

	// Nothing was consumed: the same call returns the same error.
	ss1ExpectError(t, ErrSearchComplete, search.Feed())
	srExpectSame(t, "second feed", before, &search)

	// What the recorded cases do next: searchAll (returns without searching:
	// the state is complete) and matches change nothing, and tick says
	// complete.
	zigTry(t, search.SearchAll())
	srExpectSame(t, "searchAll", before, &search)
	ss1ExpectError(t, ErrSearchComplete, search.Tick())
	srExpectSame(t, "tick", before, &search)
}

// srPruneRun is one Go test of (B): the situation in its own capacity scope.
func srPruneRun(t *testing.T, c srPrune) {
	t.Helper()
	testStdCapacity(t)
	defer srRecover(t)
	srFeedComplete(t, c)
}

// The five tests below pick the distinct situations of the 48 cases (the
// case name is c<cols>_sb<max_scrollback>_<needle>_select_<dir>_more<n>):
//   - sb1 is the smallest scrollback (min_max_size, two pages), sb1500000 a
//     larger limit: both prune whole pages;
//   - 1000 columns have 46 rows per page, 2000 columns 23: `more` is the
//     smaller and the larger count that aborts upstream (49/97 and 26/51);
//   - the needle matches in every page ("fizz"), in one page only ("page 1")
//     or as a single letter ("z");
//   - select(.next) selects the newest match, select(.prev) the oldest.

// Go-only: ruling (feed on a completed search returns ErrSearchComplete; upstream's unreachable arm), case c1000_sb1_fizz_select_next_more49
func TestSearchRuling_feed_complete_c1000_sb1_fizz_select_next_more49(t *testing.T) {
	srPruneRun(t, srPrune{cols: 1000, sb: 1, nl: 46, needle: "fizz", sel: SearchScreenSelectNext, more: 49})
}

// Go-only: ruling (feed on a completed search returns ErrSearchComplete; upstream's unreachable arm), case c1000_sb1_page_201_select_prev_more97
func TestSearchRuling_feed_complete_c1000_sb1_page_201_select_prev_more97(t *testing.T) {
	srPruneRun(t, srPrune{cols: 1000, sb: 1, nl: 46, needle: "page 1", sel: SearchScreenSelectPrev, more: 97})
}

// Go-only: ruling (feed on a completed search returns ErrSearchComplete; upstream's unreachable arm), case c1000_sb1500000_z_select_next_more97
func TestSearchRuling_feed_complete_c1000_sb1500000_z_select_next_more97(t *testing.T) {
	srPruneRun(t, srPrune{cols: 1000, sb: 1500000, nl: 46, needle: "z", sel: SearchScreenSelectNext, more: 97})
}

// Go-only: ruling (feed on a completed search returns ErrSearchComplete; upstream's unreachable arm), case c2000_sb1_fizz_select_prev_more26
func TestSearchRuling_feed_complete_c2000_sb1_fizz_select_prev_more26(t *testing.T) {
	srPruneRun(t, srPrune{cols: 2000, sb: 1, nl: 23, needle: "fizz", sel: SearchScreenSelectPrev, more: 26})
}

// Go-only: ruling (feed on a completed search returns ErrSearchComplete; upstream's unreachable arm), case c2000_sb1500000_page_201_select_next_more51
func TestSearchRuling_feed_complete_c2000_sb1500000_page_201_select_next_more51(t *testing.T) {
	srPruneRun(t, srPrune{cols: 2000, sb: 1500000, nl: 23, needle: "page 1", sel: SearchScreenSelectNext, more: 51})
}

// All 48 aborting cases of changes_prune.case (the other 48 cases of the
// suite, `more23`/`more184` and `more11`/`more92`, do not reach the arm: feed
// finds no page to add and prunes; they are recorded and replayed with the
// corpus).
//
// Go-only: ruling (feed on a completed search returns ErrSearchComplete; upstream's unreachable arm), all 48 changes_prune cases
func TestSearchRuling_feed_complete_all_48_changes_prune_cases(t *testing.T) {
	testStdCapacity(t)
	type width struct {
		name string
		cols SizeCellCountInt
		nl   uint
		more [2]uint
	}
	widths := []width{{"c1000", 1000, 46, [2]uint{49, 97}}, {"c2000", 2000, 23, [2]uint{26, 51}}}
	scrollbacks := []struct {
		name string
		sb   uint64
	}{{"sb1", 1}, {"sb1500000", 1500000}}
	needles := []struct{ name, needle string }{{"fizz", "fizz"}, {"page_201", "page 1"}, {"z", "z"}}
	dirs := []struct {
		name string
		sel  SearchScreenSelect
	}{{"next", SearchScreenSelectNext}, {"prev", SearchScreenSelectPrev}}
	var count int
	for _, w := range widths {
		for _, sb := range scrollbacks {
			for _, n := range needles {
				for _, d := range dirs {
					for _, more := range w.more {
						count++
						c := srPrune{cols: w.cols, sb: sb.sb, nl: w.nl, needle: n.needle, sel: d.sel, more: more}
						name := fmt.Sprintf("%s_%s_%s_select_%s_more%d", w.name, sb.name, n.name, d.name, more)
						t.Run(name, func(t *testing.T) {
							srPruneRun(t, c)
						})
					}
				}
			}
		}
	}
	expectEqual(t, 48, count)
}
