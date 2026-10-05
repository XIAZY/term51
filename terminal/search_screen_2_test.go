// Tests translated one for one from src/terminal/search/screen.zig (Ghostty
// v1.3.1 plus the term51 additional tests), lines 2799 to the end of the
// patched file: from `term51: ScreenSearch.reloadActive: selected match gone
// and no matches left clears selection` to `term51: ScreenSearch.selectPrev:
// allocation failure while tracking` (tests 66 to 128). Each Go test carries
// the exact Zig title in a "// Zig: test" comment. Helpers are prefixed ss2.
//
// Translation notes:
//   - `try x` on a call that returns an error is zigTry(t, err); `x.?` is
//     zigUnwrap or, where the payload is modified in place, ss2Selected and
//     ss2History (the Optional's Value, failing on null). `try
//     testing.expect(c)` is expect(t, c, "<the Zig expression>"), and
//     expectEqual keeps Zig's (expected, actual) order.
//   - `term51AMNew` and friends are ss2New, ss2NewSmall, ss2Cap, ss2Nodes,
//     ss2RowBase, ss2GrowTo, ss2ThreePages, ss2ExpectAt, ss2ExpectSel,
//     ss2ExpectRows, ss2PrunePrelude and ss2PruneFirst. The allocator
//     parameter is dropped. `max_scrollback = maxInt(usize)` is
//     math.MaxUint64 (bytes, uint64).
//   - `search.active_results.items` is `search.active_results` (a slice);
//     `m.chunks.items(.node)` is `m.chunks[i].node` (highlight.go).
//     `x.chunks.bytes == y.chunks.bytes` (raw pointer) is a comparison of the
//     addresses of the first chunk (the stored result and the entry of
//     matches() share their backing array).
//   - testing.allocator leak checks and `defer t.deinit(alloc)` allocator
//     arguments are dropped. The allocation failure tests (FailingAllocator,
//     the reloadActive tripwire, the tracked pin set filling that needs a
//     failing pool allocator) are not translated: see
//     scripts/not-translated.txt.
//   - Go-only panic tests are at the end of the file.

package terminal

import (
	"math"
	"strings"
	"testing"
)

// ss2Cols is term51AMCols: multi page tests use the widest standard page
// (215 columns), the one with the fewest rows per page.
const ss2Cols = 215

// ss2Xs is term51AMXs: a row of "x" up to the last two cells.
var ss2Xs = strings.Repeat("x", ss2Cols-2)

// ss2Full is term51AMFull: a full row of "x".
var ss2Full = strings.Repeat("x", ss2Cols)

// ss2Text is term51AMText: four rows of text, two of them "Fizz".
const ss2Text = "Fizz\r\nBuzz\r\nFizz\r\nBang"

// ss2ExpectPanic runs f and fails the test unless it panics with something
// other than the not-implemented placeholder.
func ss2ExpectPanic(t *testing.T, f func()) {
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

// ss2TerminalInit is `try Terminal.init(alloc, opts)`.
func ss2TerminalInit(t *testing.T, opts TerminalOptions) *Terminal {
	t.Helper()
	term, err := TerminalInit(opts)
	zigTry(t, err)
	return &term
}

// ss2New is term51AMNew: max_scrollback = maxInt(usize).
func ss2New(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) *Terminal {
	t.Helper()
	opts := terminalOptionsDefault(cols, rows)
	opts.max_scrollback = math.MaxUint64
	return ss2TerminalInit(t, opts)
}

// ss2NewSmall is term51AMNewSmall: the smallest scrollback limit keeps two
// pages.
func ss2NewSmall(t *testing.T) *Terminal {
	t.Helper()
	opts := terminalOptionsDefault(ss2Cols, 2)
	opts.max_scrollback = 1
	return ss2TerminalInit(t, opts)
}

// ss2Init is `try ScreenSearch.init(alloc, screen, needle)`.
func ss2Init(t *testing.T, screen *Screen, needle string) *SearchScreen {
	t.Helper()
	search, err := SearchScreenInit(screen, []uint8(needle))
	zigTry(t, err)
	return &search
}

// ss2Write is `try s.nextSlice(str)`.
func ss2Write(t *testing.T, s *StreamReadonlyStream, str string) {
	t.Helper()
	zigTry(t, s.NextSlice([]uint8(str)))
}

// ss2First is `list.pages.first.?`.
func ss2First(t *testing.T, list *PageList) *PageListNode {
	t.Helper()
	if list.pages.first == nil {
		t.Fatalf("unwrapped a null optional")
	}
	return list.pages.first
}

// ss2Last is `list.pages.last.?`.
func ss2Last(t *testing.T, list *PageList) *PageListNode {
	t.Helper()
	if list.pages.last == nil {
		t.Fatalf("unwrapped a null optional")
	}
	return list.pages.last
}

// ss2Selected is `search.selected.?`, as a pointer to the payload.
func ss2Selected(t *testing.T, search *SearchScreen) *SearchScreenSelectedMatch {
	t.Helper()
	if !search.selected.Valid {
		t.Fatalf("unwrapped a null optional")
	}
	return &search.selected.Value
}

// ss2History is `search.history.?`, as a pointer to the payload.
func ss2History(t *testing.T, search *SearchScreen) *searchScreenHistorySearch {
	t.Helper()
	if !search.history.Valid {
		t.Fatalf("unwrapped a null optional")
	}
	return &search.history.Value
}

// ss2Select is `try search.select(to)`.
func ss2Select(t *testing.T, search *SearchScreen, to SearchScreenSelect) bool {
	t.Helper()
	ok, err := search.Select(to)
	zigTry(t, err)
	return ok
}

// ss2SelectNext is `try search.selectNext()`.
func ss2SelectNext(t *testing.T, search *SearchScreen) bool {
	t.Helper()
	ok, err := search.selectNext()
	zigTry(t, err)
	return ok
}

// ss2SelectPrev is `try search.selectPrev()`.
func ss2SelectPrev(t *testing.T, search *SearchScreen) bool {
	t.Helper()
	ok, err := search.selectPrev()
	zigTry(t, err)
	return ok
}

// ss2Cap is term51AMCap: the row capacity of the first page.
func ss2Cap(t *testing.T, list *PageList) uint {
	t.Helper()
	return uint(ss2First(t, list).data.capacity.rows)
}

// ss2Nodes is term51AMNodes.
func ss2Nodes(list *PageList) uint {
	return list.TotalPages()
}

// ss2RowBase is term51AMRowBase: the number of screen rows that come before
// the node with the given index.
func ss2RowBase(list *PageList, index uint) uint {
	var rows uint
	node := list.pages.first
	var i uint
	for node != nil {
		if i == index {
			return rows
		}
		rows += uint(node.data.size.rows)
		i++
		node = node.next
	}
	panic("unreachable")
}

// ss2GrowTo is term51AMGrowTo: write newlines until the list has n nodes. The
// cursor is then on the first row of the last node.
func ss2GrowTo(t *testing.T, s *StreamReadonlyStream, list *PageList, n uint) {
	t.Helper()
	for ss2Nodes(list) < n {
		ss2Write(t, s, "\r\n")
	}
}

// ss2ThreePages is term51AMThreePages: builds three pages. Each text is
// written at the first row of its page, the last text may contain further
// lines.
func ss2ThreePages(t *testing.T, s *StreamReadonlyStream, list *PageList, p1 string, p2 string, p3 string) {
	t.Helper()
	ss2Write(t, s, p1)
	ss2GrowTo(t, s, list, 2)
	ss2Write(t, s, p2)
	ss2GrowTo(t, s, list, 3)
	ss2Write(t, s, p3)
}

// ss2ExpectAt is term51AMExpectAt: expect the match to span exactly the given
// screen coordinates.
func ss2ExpectAt(t *testing.T, list *PageList, hl HighlightFlattened, sx uint, sy uint, ex uint, ey uint) {
	t.Helper()
	u := hl.Untracked()
	expectEqualPoint(t, Point{Tag: PointTagScreen, Screen: PointCoordinate{
		x: SizeCellCountInt(sx),
		y: uint32(sy),
	}}, zigUnwrap(t, list.PointFromPin(PointTagScreen, u.start)))
	expectEqualPoint(t, Point{Tag: PointTagScreen, Screen: PointCoordinate{
		x: SizeCellCountInt(ex),
		y: uint32(ey),
	}}, zigUnwrap(t, list.PointFromPin(PointTagScreen, u.end)))
}

// ss2ExpectSel is term51AMExpectSel: expect the selected match to be at the
// given screen coordinates and index.
func ss2ExpectSel(t *testing.T, search *SearchScreen, idx uint, sx uint, sy uint, ex uint, ey uint) {
	t.Helper()
	expectEqual(t, idx, ss2Selected(t, search).idx)
	ss2ExpectAt(t, &search.screen.pages, zigUnwrap(t, search.SelectedMatch()), sx, sy, ex, ey)
}

// ss2ExpectRows is term51AMExpectRows: expect the match list to have exactly
// the given single-row matches (start x, y, end x), in order (newest first).
func ss2ExpectRows(t *testing.T, search *SearchScreen, expected [][3]uint) {
	t.Helper()
	ms, err := search.Matches()
	zigTry(t, err)
	expectEqual(t, uint(len(expected)), uint(len(ms)))
	for i, e := range expected {
		ss2ExpectAt(t, &search.screen.pages, ms[i], e[0], e[1], e[2], e[1])
	}
}

// ss2PrunePrelude is term51AMPrunePrelude: page 1 holds Fizz on its first
// row, page 2 holds the whole active area with Fizz on its second row.
func ss2PrunePrelude(t *testing.T, s *StreamReadonlyStream, list *PageList) {
	t.Helper()
	ss2Write(t, s, "Fizz")
	ss2GrowTo(t, s, list, 2)
	ss2Write(t, s, "\r\nFizz")
}

// ss2PruneFirst is term51AMPruneFirst: keep writing until the first page is
// recycled as a new last page, then write two rows there so the new page holds
// the whole active area.
func ss2PruneFirst(t *testing.T, s *StreamReadonlyStream, list *PageList) {
	t.Helper()
	for list.page_serial_min == 0 {
		ss2Write(t, s, "\r\n")
	}
	ss2Write(t, s, "Fizz\r\nbuzz")
}

// Zig: test "term51: ScreenSearch.reloadActive: selected match gone and no matches left clears selection"
func TestSs2_ScreenSearch_reloadActive_selected_match_gone_and_no_matches_left_clears_selection(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	term := ss2New(t, 10, 5)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2Write(t, &s, "Fizz\r\nBuzz\r\nFizz")

	base := list.CountTrackedPins()
	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	ss2Select(t, search, SearchScreenSelectNext)
	expectEqual(t, base+4, list.CountTrackedPins())

	ss2Write(t, &s, "\x1b[2J\x1b[H")
	zigTry(t, search.ReloadActive())
	expect(t, !search.selected.Valid, "search.selected == null")
	expect(t, !search.SelectedMatch().Valid, "search.selectedMatch() == null")
	expectEqual(t, uint(0), search.MatchesLen())
	// The selection pins were released.
	expectEqual(t, base+2, list.CountTrackedPins())
}

// Zig: test "term51: ScreenSearch.reloadActive: selection in history keeps its match when active changes"
func TestSs2_ScreenSearch_reloadActive_selection_in_history_keeps_its_match_when_active_changes(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2ThreePages(t, &s, list, "Fizz", "fIzZ", "FIZZ\r\nbuzz")

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	// No prior selection: prev selects the oldest, the last index.
	ss2Select(t, search, SearchScreenSelectPrev)
	ss2ExpectSel(t, search, 2, 0, 0, 3, 0)

	// Another active match: the active list grows from 1 to 2.
	ss2Write(t, &s, "\r\nFizz")
	zigTry(t, search.ReloadActive())
	expectEqual(t, 2, len(search.active_results))
	expectEqual(t, 2, len(search.history_results))
	ss2ExpectSel(t, search, 3, 0, 0, 3, 0)

	// Clearing the active area removes the last row's match; the first row
	// of the page is outside the active area but on the active page, so it
	// is still an active result.
	ss2Write(t, &s, "\x1b[2J\x1b[H")
	zigTry(t, search.ReloadActive())
	expectEqual(t, 1, len(search.active_results))
	expectEqual(t, 2, len(search.history_results))
	ss2ExpectSel(t, search, 2, 0, 0, 3, 0)
}

// Zig: test "term51: ScreenSearch.reloadActive: new history node adds matches and shifts the selection"
func TestSs2_ScreenSearch_reloadActive_new_history_node_adds_matches_and_shifts_the_selection(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2ThreePages(t, &s, list, "Fizz", "fIzZ", "FIZZ\r\nbuzz")
	n3 := ss2Last(t, list)

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	ss2Select(t, search, SearchScreenSelectNext)
	ss2Select(t, search, SearchScreenSelectNext)
	b2 := ss2RowBase(list, 1)
	ss2ExpectSel(t, search, 1, 0, b2, 3, b2)
	expect(t, ss2History(t, search).start_pin.node == n3, "search.history.?.start_pin.node == n3")

	// Move on to a fourth page, the third becomes history.
	ss2GrowTo(t, &s, list, 4)
	ss2Write(t, &s, "FiZz\r\nxx")
	n4 := ss2Last(t, list)
	zigTry(t, search.ReloadActive())
	expect(t, ss2History(t, search).start_pin.node == n4, "search.history.?.start_pin.node == n4")
	expectEqual(t, 1, len(search.active_results))
	expectEqual(t, 3, len(search.history_results))
	expectEqual(t, uint(4), search.MatchesLen())

	// Same match is still selected, index moved by the added result.
	ss2ExpectSel(t, search, 2, 0, b2, 3, b2)
	b3 := ss2RowBase(list, 2)
	b4 := ss2RowBase(list, 3)
	ss2ExpectRows(t, search, [][3]uint{
		{0, b4, 3},
		{0, b3, 3},
		{0, b2, 3},
		{0, 0, 3},
	})
}

// Zig: test "term51: ScreenSearch.reloadActive: selection in the active area is reset when it becomes history"
func TestSs2_ScreenSearch_reloadActive_selection_in_the_active_area_is_reset_when_it_becomes_history(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2ThreePages(t, &s, list, "Fizz", "fIzZ", "FIZZ\r\nbuzz")

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	ss2Select(t, search, SearchScreenSelectNext)
	b3 := ss2RowBase(list, 2)
	ss2ExpectSel(t, search, 0, 0, b3, 3, b3)

	ss2GrowTo(t, &s, list, 4)
	ss2Write(t, &s, "FiZz\r\nxx")
	zigTry(t, search.ReloadActive())
	expectEqual(t, uint(4), search.MatchesLen())
	// The selected match is no longer in the active area so we are back
	// at the newest match, which is now in the fourth page.
	b4 := ss2RowBase(list, 3)
	ss2ExpectSel(t, search, 0, 0, b4, 3, b4)
}

// Zig: test "term51: ScreenSearch.reloadActive: new history node without matches keeps the history"
func TestSs2_ScreenSearch_reloadActive_new_history_node_without_matches_keeps_the_history(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2ThreePages(t, &s, list, "Fizz", "fIzZ", "ab\r\ncd")

	// Zig-only: upstream defect measured with an arena behind a counting
	// allocator (var arena: std.heap.ArenaAllocator = .init(alloc);
	// defer arena.deinit(); var fa = std.testing.FailingAllocator.init(
	// arena.allocator(), .{});): the list preallocated for the new results
	// is never freed. Go has no allocator; the rest of the test is kept.
	search := ss2Init(t, term.screens.active, "Fizz")
	zigTry(t, search.SearchAll())
	expectEqual(t, 0, len(search.active_results))
	ss2Select(t, search, SearchScreenSelectPrev)
	ss2ExpectSel(t, search, 1, 0, 0, 3, 0)

	ss2GrowTo(t, &s, list, 4)
	ss2Write(t, &s, "FiZz\r\nxx")
	zigTry(t, search.ReloadActive())
	expect(t, ss2History(t, search).start_pin.node == ss2Last(t, list), "search.history.?.start_pin.node == list.pages.last.?")
	expectEqual(t, 1, len(search.active_results))
	expectEqual(t, 2, len(search.history_results))
	ss2ExpectSel(t, search, 2, 0, 0, 3, 0)

	search.Deinit()
	// Zig-only: try std.testing.expectEqual(2 * @sizeOf(FlattenedHighlight),
	// fa.allocated_bytes - fa.freed_bytes); (two highlights worth of memory
	// were never freed)
}

// Zig: test "term51: ScreenSearch.reloadActive: history added by a move is ordered newest first"
func TestSs2_ScreenSearch_reloadActive_history_added_by_a_move_is_ordered_newest_first(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2ThreePages(t, &s, list, "Fizz", "fIzZ", "Fizz fizz\r\nbuzz")

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expectEqual(t, 2, len(search.active_results))
	expectEqual(t, 2, len(search.history_results))

	ss2GrowTo(t, &s, list, 4)
	ss2Write(t, &s, "xx\r\nyy")
	zigTry(t, search.ReloadActive())
	expectEqual(t, 0, len(search.active_results))
	expectEqual(t, 4, len(search.history_results))
	b2 := ss2RowBase(list, 1)
	b3 := ss2RowBase(list, 2)
	ss2ExpectRows(t, search, [][3]uint{
		{5, b3, 8},
		{0, b3, 3},
		{0, b2, 3},
		{0, 0, 3},
	})
}

// Zig: test "term51: ScreenSearch.reloadActive: new history node adds a match spanning into it"
func TestSs2_ScreenSearch_reloadActive_new_history_node_adds_a_match_spanning_into_it(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()

	// Two pages, the second one holds the whole active area.
	ss2Write(t, &s, "abcdefg")
	ss2GrowTo(t, &s, list, 2)
	ss2Write(t, &s, "\r\n")
	search := ss2Init(t, term.screens.active, "Test")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expectEqual(t, uint(0), search.MatchesLen())

	// Fill the second page, the match wraps from its last row into page 3.
	n2 := ss2Last(t, list)
	for n2.data.size.rows < n2.data.capacity.rows-1 {
		ss2Write(t, &s, "\r\n")
	}
	ss2Write(t, &s, "\r\n"+ss2Xs+"Test\r\nzz")
	expectEqual(t, uint(3), ss2Nodes(list))
	n3 := ss2Last(t, list)
	expect(t, ss2History(t, search).start_pin.node == n2, "search.history.?.start_pin.node == n2")

	zigTry(t, search.ReloadActive())
	expect(t, ss2History(t, search).start_pin.node == n3, "search.history.?.start_pin.node == n3")
	expectEqual(t, 0, len(search.active_results))
	expectEqual(t, 1, len(search.history_results))
	m := search.history_results[0]
	expectEqual(t, 2, len(m.chunks))
	expect(t, m.chunks[0].node == n2, "m.chunks.items(.node)[0] == n2")
	expect(t, m.chunks[1].node == n3, "m.chunks.items(.node)[1] == n3")
	// Forward search: serials are in page order.
	expectEqual(t, n2.serial, m.chunks[0].serial)
	expectEqual(t, n3.serial, m.chunks[1].serial)
	last_row := n2.data.size.rows - 1
	expectEqual(t, SizeCellCountInt(last_row), m.chunks[0].start)
	expectEqual(t, SizeCellCountInt(last_row+1), m.chunks[0].end)
	expectEqual(t, SizeCellCountInt(0), m.chunks[1].start)
	expectEqual(t, SizeCellCountInt(1), m.chunks[1].end)
	expectEqual(t, SizeCellCountInt(ss2Cols-2), m.top_x)
	expectEqual(t, SizeCellCountInt(1), m.bot_x)
	b2 := ss2RowBase(list, 1)
	ss2ExpectAt(t, list, m, ss2Cols-2, b2+uint(last_row), 1, ss2RowBase(list, 2))
}

// Zig: test "term51: ScreenSearch.reloadActive: full reset drops history and reselects the oldest"
func TestSs2_ScreenSearch_reloadActive_full_reset_drops_history_and_reselects_the_oldest(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2ThreePages(t, &s, list, "Fizz", "fIzZ", "FIZZ\r\nbuzz")

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	ss2Select(t, search, SearchScreenSelectPrev)
	expectEqual(t, uint(2), ss2Selected(t, search).idx)
	expectEqual(t, searchScreenStateComplete, search.state)

	term.FullReset()
	ss2Write(t, &s, "xFizz")
	zigTry(t, search.ReloadActive())

	// History and its results are gone, a fresh searcher exists, the
	// selection was garbage so it was reselected: the oldest match is the
	// one match in the new screen.
	expectEqual(t, 0, len(search.history_results))
	expect(t, search.history.Valid, "search.history != null")
	expect(t, !ss2History(t, search).start_pin.garbage, "!search.history.?.start_pin.garbage")
	expect(t, !ss2History(t, search).searcher.pin.garbage, "!search.history.?.searcher.pin.garbage")
	expect(t, ss2History(t, search).start_pin.node == ss2First(t, list), "search.history.?.start_pin.node == list.pages.first.?")
	expectEqual(t, uint(1), search.MatchesLen())
	ss2ExpectSel(t, search, 0, 1, 0, 4, 0)
	// The state is not reset by a reload.
	expectEqual(t, searchScreenStateComplete, search.state)
}

// Zig: test "term51: ScreenSearch.reloadActive: garbage selection pins reselect the oldest match"
func TestSs2_ScreenSearch_reloadActive_garbage_selection_pins_reselect_the_oldest_match(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, 10, 5)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	ss2Write(t, &s, "Fizz\r\nBuzz\r\nFizz")

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	ss2Select(t, search, SearchScreenSelectNext)
	ss2ExpectSel(t, search, 0, 0, 2, 3, 2)

	// Only one of the two pins being garbage is enough.
	ss2Selected(t, search).highlight.end.garbage = true
	zigTry(t, search.ReloadActive())
	ss2ExpectSel(t, search, 1, 0, 0, 3, 0)
	expect(t, !ss2Selected(t, search).highlight.end.garbage, "!search.selected.?.highlight.end.garbage")

	ss2Selected(t, search).highlight.start.garbage = true
	zigTry(t, search.ReloadActive())
	ss2ExpectSel(t, search, 1, 0, 0, 3, 0)
	expect(t, !ss2Selected(t, search).highlight.start.garbage, "!search.selected.?.highlight.start.garbage")

	// Non garbage pins are left alone: still the oldest.
	zigTry(t, search.ReloadActive())
	ss2ExpectSel(t, search, 1, 0, 0, 3, 0)
}

// Zig: test "term51: ScreenSearch.reloadActive: garbage selection with no matches left selects nothing"
func TestSs2_ScreenSearch_reloadActive_garbage_selection_with_no_matches_left_selects_nothing(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, 10, 5)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2Write(t, &s, "Fizz")

	base := list.CountTrackedPins()
	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	ss2Select(t, search, SearchScreenSelectNext)
	ss2Write(t, &s, "\x1b[2J\x1b[H")
	ss2Selected(t, search).highlight.start.garbage = true
	zigTry(t, search.ReloadActive())
	expect(t, !search.selected.Valid, "search.selected == null")
	expectEqual(t, base+2, list.CountTrackedPins())
}

// Zig: test "term51: ScreenSearch.reloadActive: no scrollback prunes matches above the active area"
func TestSs2_ScreenSearch_reloadActive_no_scrollback_prunes_matches_above_the_active_area(t *testing.T) {
	testStdCapacity(t)
	opts := terminalOptionsDefault(10, 2)
	opts.max_scrollback = 0
	term := ss2TerminalInit(t, opts)
	defer term.Deinit()
	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()

	// CSI 22 J pushes the screen into the page even without scrollback.
	ss2Write(t, &s, "Fizz\r\nFizz")
	ss2Write(t, &s, "\x1b[22J")
	ss2Write(t, &s, "x")
	expectEqual(t, uint(4), list.total_rows)

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	// Both matches are above the active area: all pruned.
	expectEqual(t, 0, len(search.active_results))
	expect(t, !search.history.Valid, "search.history == null")

	// One match in the first active row survives, the other is pruned.
	ss2Write(t, &s, "\r\n")
	ss2Write(t, &s, "\x1b[22J")
	ss2Write(t, &s, "Fizz")
	zigTry(t, search.ReloadActive())
	expectEqual(t, 1, len(search.active_results))
	tl := uint(zigUnwrap(t, list.PointFromPin(PointTagScreen, list.GetTopLeft(PointTagActive))).Screen.y)
	ss2ExpectAt(t, list, search.active_results[0], 0, tl, 3, tl)

	// Both in the active area: nothing is pruned.
	ss2Write(t, &s, "\r\nFizz")
	zigTry(t, search.ReloadActive())
	expectEqual(t, 2, len(search.active_results))
	ss2ExpectAt(t, list, search.active_results[0], 0, tl, 3, tl)
	ss2ExpectAt(t, list, search.active_results[1], 0, tl+1, 3, tl+1)
}

// Zig: test "term51: ScreenSearch.reloadActive: no scrollback with all matches in the active area"
func TestSs2_ScreenSearch_reloadActive_no_scrollback_with_all_matches_in_the_active_area(t *testing.T) {
	testStdCapacity(t)
	opts := terminalOptionsDefault(10, 3)
	opts.max_scrollback = 0
	term := ss2TerminalInit(t, opts)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	ss2Write(t, &s, "Fizz\r\nBuzz\r\nFizz")

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	ss2ExpectRows(t, search, [][3]uint{{0, 2, 3}, {0, 0, 3}})
	// Scrolling shifts the rows in place, the first Fizz is gone.
	ss2Write(t, &s, "\r\nBang")
	zigTry(t, search.ReloadActive())
	ss2ExpectRows(t, search, [][3]uint{{0, 1, 3}})
}

// Zig: test "term51: ScreenSearch.reloadActive: no scrollback drops a single cell match at the active top left"
func TestSs2_ScreenSearch_reloadActive_no_scrollback_drops_a_single_cell_match_at_the_active_top_left(t *testing.T) {
	testStdCapacity(t)
	opts := terminalOptionsDefault(10, 2)
	opts.max_scrollback = 0
	term := ss2TerminalInit(t, opts)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	ss2Write(t, &s, "Fizz\r\nxF")

	// The pruning keeps a result only if the top left of the active area
	// is strictly before its end, so a match ending exactly on the top
	// left cell (a one character needle at x=0 of the first row) is lost
	// even though it is visible.
	func() {
		search := ss2Init(t, term.screens.active, "F")
		defer search.Deinit()
		ss2ExpectRows(t, search, [][3]uint{{1, 1, 1}})
	}()
	// Two characters ending on x=1 are kept.
	func() {
		search := ss2Init(t, term.screens.active, "Fi")
		defer search.Deinit()
		ss2ExpectRows(t, search, [][3]uint{{0, 0, 1}})
	}()
	func() {
		search := ss2Init(t, term.screens.active, "i")
		defer search.Deinit()
		ss2ExpectRows(t, search, [][3]uint{{1, 0, 1}})
	}()
}

// Zig: test "term51: ScreenSearch.selectedMatch: null without a selection"
func TestSs2_ScreenSearch_selectedMatch_null_without_a_selection(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, 10, 2)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	ss2Write(t, &s, ss2Text)
	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	expect(t, !search.SelectedMatch().Valid, "search.selectedMatch() == null")
	zigTry(t, search.SearchAll())
	expect(t, !search.SelectedMatch().Valid, "search.selectedMatch() == null")
}

// Zig: test "term51: ScreenSearch.selectedMatch: maps the index from the newest match"
func TestSs2_ScreenSearch_selectedMatch_maps_the_index_from_the_newest_match(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2ThreePages(t, &s, list, "Fizz", "fIzZ", "FIZZ\r\nbuzz\r\nfizz")
	b2 := ss2RowBase(list, 1)
	b3 := ss2RowBase(list, 2)

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expectEqual(t, uint(4), search.MatchesLen())
	ss2Select(t, search, SearchScreenSelectNext)

	// Reads the stored results only: set the index by hand.
	type Expect struct{ idx, x, y uint }
	cases := []Expect{
		{idx: 0, x: 0, y: b3 + 2}, // newest active
		{idx: 1, x: 0, y: b3},     // oldest active
		{idx: 2, x: 0, y: b2},     // first history result
		{idx: 3, x: 0, y: 0},      // last history result
	}
	for _, c := range cases {
		ss2Selected(t, search).idx = c.idx
		m := zigUnwrap(t, search.SelectedMatch())
		ss2ExpectAt(t, list, m, c.x, c.y, c.x+3, c.y)
		// It is the same entry the matches list has at that index.
		ms, err := search.Matches()
		zigTry(t, err)
		// Raw pointer comparison of the chunk storage: the address of the
		// first chunk.
		expect(t, &m.chunks[0] == &ms[c.idx].chunks[0], "m.chunks.bytes == ms[c.idx].chunks.bytes")
	}
}

// Zig: test "term51: ScreenSearch.selectedMatch: index past the results is null"
func TestSs2_ScreenSearch_selectedMatch_index_past_the_results_is_null(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2ThreePages(t, &s, list, "Fizz", "fIzZ", "FIZZ\r\nbuzz")

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	ss2Select(t, search, SearchScreenSelectNext)
	ss2Selected(t, search).idx = 2
	expect(t, search.SelectedMatch().Valid, "search.selectedMatch() != null")
	ss2Selected(t, search).idx = 3
	expect(t, !search.SelectedMatch().Valid, "search.selectedMatch() == null")
	ss2Selected(t, search).idx = math.MaxUint - 1
	expect(t, !search.SelectedMatch().Valid, "search.selectedMatch() == null")
}

// Zig: test "term51: ScreenSearch.selectedMatch: history only and active only"
func TestSs2_ScreenSearch_selectedMatch_history_only_and_active_only(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2ThreePages(t, &s, list, "Fizz", "fIzZ", "ab\r\ncd")
	b2 := ss2RowBase(list, 1)

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expectEqual(t, 0, len(search.active_results))
	ss2Select(t, search, SearchScreenSelectNext)
	ss2ExpectSel(t, search, 0, 0, b2, 3, b2)
	ss2Selected(t, search).idx = 1
	ss2ExpectAt(t, list, zigUnwrap(t, search.SelectedMatch()), 0, 0, 3, 0)
	ss2Selected(t, search).idx = 2
	expect(t, !search.SelectedMatch().Valid, "search.selectedMatch() == null")
}

// Zig: test "term51: ScreenSearch.select: next walks newest to oldest without wrapping"
func TestSs2_ScreenSearch_select_next_walks_newest_to_oldest_without_wrapping(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2ThreePages(t, &s, list, "Fizz", "fIzZ", "FIZZ\r\nbuzz")
	b2 := ss2RowBase(list, 1)
	b3 := ss2RowBase(list, 2)

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())

	expect(t, ss2Select(t, search, SearchScreenSelectNext), "try search.select(.next)")
	ss2ExpectSel(t, search, 0, 0, b3, 3, b3)
	expect(t, ss2Select(t, search, SearchScreenSelectNext), "try search.select(.next)")
	ss2ExpectSel(t, search, 1, 0, b2, 3, b2)
	expect(t, ss2Select(t, search, SearchScreenSelectNext), "try search.select(.next)")
	ss2ExpectSel(t, search, 2, 0, 0, 3, 0)

	// At the oldest: false and nothing changes, again and again.
	expect(t, !ss2Select(t, search, SearchScreenSelectNext), "!try search.select(.next)")
	ss2ExpectSel(t, search, 2, 0, 0, 3, 0)
	expect(t, !ss2Select(t, search, SearchScreenSelectNext), "!try search.select(.next)")
	ss2ExpectSel(t, search, 2, 0, 0, 3, 0)
}

// Zig: test "term51: ScreenSearch.select: prev walks oldest to newest without wrapping"
func TestSs2_ScreenSearch_select_prev_walks_oldest_to_newest_without_wrapping(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2ThreePages(t, &s, list, "Fizz", "fIzZ", "FIZZ\r\nbuzz")
	b2 := ss2RowBase(list, 1)
	b3 := ss2RowBase(list, 2)

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())

	expect(t, ss2Select(t, search, SearchScreenSelectPrev), "try search.select(.prev)")
	ss2ExpectSel(t, search, 2, 0, 0, 3, 0)
	expect(t, ss2Select(t, search, SearchScreenSelectPrev), "try search.select(.prev)")
	ss2ExpectSel(t, search, 1, 0, b2, 3, b2)
	expect(t, ss2Select(t, search, SearchScreenSelectPrev), "try search.select(.prev)")
	ss2ExpectSel(t, search, 0, 0, b3, 3, b3)

	// At the newest: false and nothing changes.
	expect(t, !ss2Select(t, search, SearchScreenSelectPrev), "!try search.select(.prev)")
	ss2ExpectSel(t, search, 0, 0, b3, 3, b3)
	expect(t, !ss2Select(t, search, SearchScreenSelectPrev), "!try search.select(.prev)")
	ss2ExpectSel(t, search, 0, 0, b3, 3, b3)
}

// Zig: test "term51: ScreenSearch.select: next and prev alternate"
func TestSs2_ScreenSearch_select_next_and_prev_alternate(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2ThreePages(t, &s, list, "Fizz", "fIzZ", "FIZZ\r\nbuzz")
	b2 := ss2RowBase(list, 1)
	b3 := ss2RowBase(list, 2)

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expect(t, ss2Select(t, search, SearchScreenSelectNext), "try search.select(.next)")
	expect(t, ss2Select(t, search, SearchScreenSelectNext), "try search.select(.next)")
	ss2ExpectSel(t, search, 1, 0, b2, 3, b2)
	expect(t, ss2Select(t, search, SearchScreenSelectPrev), "try search.select(.prev)")
	ss2ExpectSel(t, search, 0, 0, b3, 3, b3)
	expect(t, ss2Select(t, search, SearchScreenSelectNext), "try search.select(.next)")
	expect(t, ss2Select(t, search, SearchScreenSelectNext), "try search.select(.next)")
	ss2ExpectSel(t, search, 2, 0, 0, 3, 0)
	expect(t, ss2Select(t, search, SearchScreenSelectPrev), "try search.select(.prev)")
	ss2ExpectSel(t, search, 1, 0, b2, 3, b2)
}

// Zig: test "term51: ScreenSearch.select: no matches selects nothing"
func TestSs2_ScreenSearch_select_no_matches_selects_nothing(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2ThreePages(t, &s, list, "Fizz", "fIzZ", "FIZZ\r\nbuzz")

	search := ss2Init(t, term.screens.active, "nomatch")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expect(t, !ss2Select(t, search, SearchScreenSelectNext), "!try search.select(.next)")
	expect(t, !search.selected.Valid, "search.selected == null")
	expect(t, !ss2Select(t, search, SearchScreenSelectPrev), "!try search.select(.prev)")
	expect(t, !search.selected.Valid, "search.selected == null")
	expect(t, !search.SelectedMatch().Valid, "search.selectedMatch() == null")
}

// Zig: test "term51: ScreenSearch.select: one match"
func TestSs2_ScreenSearch_select_one_match(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, 10, 2)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	ss2Write(t, &s, ss2Text)

	search := ss2Init(t, term.screens.active, "Bang")
	defer search.Deinit()
	zigTry(t, search.SearchAll())

	// prev first: selects the only match, then neither direction moves.
	expect(t, ss2Select(t, search, SearchScreenSelectPrev), "try search.select(.prev)")
	ss2ExpectSel(t, search, 0, 0, 3, 3, 3)
	expect(t, !ss2Select(t, search, SearchScreenSelectPrev), "!try search.select(.prev)")
	expect(t, !ss2Select(t, search, SearchScreenSelectNext), "!try search.select(.next)")
	ss2ExpectSel(t, search, 0, 0, 3, 3, 3)

	ss2Selected(t, search).Deinit(term.screens.active)
	search.selected = Optional[SearchScreenSelectedMatch]{}
	// next first.
	expect(t, ss2Select(t, search, SearchScreenSelectNext), "try search.select(.next)")
	ss2ExpectSel(t, search, 0, 0, 3, 3, 3)
	expect(t, !ss2Select(t, search, SearchScreenSelectNext), "!try search.select(.next)")
	expect(t, !ss2Select(t, search, SearchScreenSelectPrev), "!try search.select(.prev)")
	ss2ExpectSel(t, search, 0, 0, 3, 3, 3)
}

// Zig: test "term51: ScreenSearch.select: before history is searched only known matches are used"
func TestSs2_ScreenSearch_select_before_history_is_searched_only_known_matches_are_used(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2ThreePages(t, &s, list, "Fizz", "fIzZ", "FIZZ\r\nbuzz\r\nFizz")
	b2 := ss2RowBase(list, 1)
	b3 := ss2RowBase(list, 2)

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	expectEqual(t, 0, len(search.history_results))

	// The oldest known match is the first active one.
	expect(t, ss2Select(t, search, SearchScreenSelectPrev), "try search.select(.prev)")
	ss2ExpectSel(t, search, 1, 0, b3, 3, b3)
	// select does not search history, it only reloads the active area, so
	// there is nothing older to go to yet.
	expect(t, !ss2Select(t, search, SearchScreenSelectNext), "!try search.select(.next)")
	expectEqual(t, 0, len(search.history_results))
	ss2ExpectSel(t, search, 1, 0, b3, 3, b3)
	expect(t, ss2Select(t, search, SearchScreenSelectPrev), "try search.select(.prev)")
	ss2ExpectSel(t, search, 0, 0, b3+2, 3, b3+2)
	expect(t, ss2Select(t, search, SearchScreenSelectNext), "try search.select(.next)")

	// History results are appended to the end, the index of the selection
	// (counted from the newest) is unaffected.
	zigTry(t, search.Tick())
	zigTry(t, search.Feed())
	zigTry(t, search.Tick())
	expectEqual(t, 1, len(search.history_results))
	ss2ExpectSel(t, search, 1, 0, b3, 3, b3)
	expect(t, ss2Select(t, search, SearchScreenSelectNext), "try search.select(.next)")
	ss2ExpectSel(t, search, 2, 0, b2, 3, b2)
	expect(t, !ss2Select(t, search, SearchScreenSelectNext), "!try search.select(.next)")
	zigTry(t, search.Feed())
	zigTry(t, search.Tick())
	expect(t, ss2Select(t, search, SearchScreenSelectNext), "try search.select(.next)")
	ss2ExpectSel(t, search, 3, 0, 0, 3, 0)
}

// Zig: test "term51: ScreenSearch.select: sees changes to the active area without a reload call"
func TestSs2_ScreenSearch_select_sees_changes_to_the_active_area_without_a_reload_call(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, 10, 5)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	ss2Write(t, &s, "Fizz\r\nBuzz")

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	expect(t, ss2Select(t, search, SearchScreenSelectNext), "try search.select(.next)")
	ss2ExpectSel(t, search, 0, 0, 0, 3, 0)

	ss2Write(t, &s, "\r\nFizz")
	// Moving on reloads first, so the new match is known: the selection
	// moved from index 0 to 1 and next goes to index 2, out of range.
	expect(t, !ss2Select(t, search, SearchScreenSelectNext), "!try search.select(.next)")
	ss2ExpectSel(t, search, 1, 0, 0, 3, 0)
	expectEqual(t, uint(2), search.MatchesLen())
	expect(t, ss2Select(t, search, SearchScreenSelectPrev), "try search.select(.prev)")
	ss2ExpectSel(t, search, 0, 0, 2, 3, 2)
}

// Zig: test "term51: ScreenSearch.select: viewport scrolling does not change matches or selection"
func TestSs2_ScreenSearch_select_viewport_scrolling_does_not_change_matches_or_selection(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2ThreePages(t, &s, list, "Fizz", "fIzZ", "FIZZ\r\nbuzz")
	b2 := ss2RowBase(list, 1)
	b3 := ss2RowBase(list, 2)

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expect(t, ss2Select(t, search, SearchScreenSelectNext), "try search.select(.next)")
	ss2ExpectSel(t, search, 0, 0, b3, 3, b3)

	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	expect(t, ss2Select(t, search, SearchScreenSelectNext), "try search.select(.next)")
	ss2ExpectSel(t, search, 1, 0, b2, 3, b2)
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportDelta, Delta: 1})
	expect(t, ss2Select(t, search, SearchScreenSelectNext), "try search.select(.next)")
	ss2ExpectSel(t, search, 2, 0, 0, 3, 0)
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportBottom})
	expect(t, ss2Select(t, search, SearchScreenSelectPrev), "try search.select(.prev)")
	ss2ExpectSel(t, search, 1, 0, b2, 3, b2)
	expectEqual(t, uint(3), search.MatchesLen())
}

// Zig: test "term51: ScreenSearch.selectNext: first selection is the newest match"
func TestSs2_ScreenSearch_selectNext_first_selection_is_the_newest_match(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2ThreePages(t, &s, list, "Fizz", "fIzZ", "FIZZ\r\nbuzz")
	b2 := ss2RowBase(list, 1)
	b3 := ss2RowBase(list, 2)
	base := list.CountTrackedPins()

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())

	// Active results exist: the newest active result is selected.
	expect(t, ss2SelectNext(t, search), "try search.selectNext()")
	ss2ExpectSel(t, search, 0, 0, b3, 3, b3)
	expectEqual(t, base+4, list.CountTrackedPins())
	// Moving replaces the selection pins.
	expect(t, ss2SelectNext(t, search), "try search.selectNext()")
	ss2ExpectSel(t, search, 1, 0, b2, 3, b2)
	expectEqual(t, base+4, list.CountTrackedPins())
	expect(t, ss2SelectNext(t, search), "try search.selectNext()")
	ss2ExpectSel(t, search, 2, 0, 0, 3, 0)
	// The end: false, nothing changes, no pins leak.
	expect(t, !ss2SelectNext(t, search), "!try search.selectNext()")
	ss2ExpectSel(t, search, 2, 0, 0, 3, 0)
	expectEqual(t, base+4, list.CountTrackedPins())
}

// Zig: test "term51: ScreenSearch.selectNext: history only and nothing"
func TestSs2_ScreenSearch_selectNext_history_only_and_nothing(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2ThreePages(t, &s, list, "Fizz", "fIzZ", "ab\r\ncd")
	b2 := ss2RowBase(list, 1)

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	// Nothing known yet.
	expect(t, !ss2SelectNext(t, search), "!try search.selectNext()")
	expect(t, !search.selected.Valid, "search.selected == null")

	zigTry(t, search.SearchAll())
	expect(t, ss2SelectNext(t, search), "try search.selectNext()")
	ss2ExpectSel(t, search, 0, 0, b2, 3, b2)
	expect(t, ss2SelectNext(t, search), "try search.selectNext()")
	ss2ExpectSel(t, search, 1, 0, 0, 3, 0)
	expect(t, !ss2SelectNext(t, search), "!try search.selectNext()")
}

// Zig: test "term51: ScreenSearch.selectNext: does not reload the active area"
func TestSs2_ScreenSearch_selectNext_does_not_reload_the_active_area(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, 10, 5)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	ss2Write(t, &s, "Fizz\r\nBuzz\r\nFizz")

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	// The screen is cleared but the stored results are used as they are.
	ss2Write(t, &s, "\x1b[2J\x1b[H")
	expect(t, ss2SelectNext(t, search), "try search.selectNext()")
	ss2ExpectSel(t, search, 0, 0, 2, 3, 2)
	expect(t, ss2SelectNext(t, search), "try search.selectNext()")
	ss2ExpectSel(t, search, 1, 0, 0, 3, 0)
	expect(t, !ss2SelectNext(t, search), "!try search.selectNext()")
}

// Zig: test "term51: ScreenSearch.selectPrev: first selection is the oldest match"
func TestSs2_ScreenSearch_selectPrev_first_selection_is_the_oldest_match(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2ThreePages(t, &s, list, "Fizz", "fIzZ", "FIZZ\r\nbuzz")
	b2 := ss2RowBase(list, 1)
	b3 := ss2RowBase(list, 2)
	base := list.CountTrackedPins()

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())

	expect(t, ss2SelectPrev(t, search), "try search.selectPrev()")
	ss2ExpectSel(t, search, 2, 0, 0, 3, 0)
	expectEqual(t, base+4, list.CountTrackedPins())
	expect(t, ss2SelectPrev(t, search), "try search.selectPrev()")
	ss2ExpectSel(t, search, 1, 0, b2, 3, b2)
	expect(t, ss2SelectPrev(t, search), "try search.selectPrev()")
	ss2ExpectSel(t, search, 0, 0, b3, 3, b3)
	expect(t, !ss2SelectPrev(t, search), "!try search.selectPrev()")
	ss2ExpectSel(t, search, 0, 0, b3, 3, b3)
	expectEqual(t, base+4, list.CountTrackedPins())
}

// Zig: test "term51: ScreenSearch.selectPrev: active only and nothing"
func TestSs2_ScreenSearch_selectPrev_active_only_and_nothing(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, 10, 5)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	ss2Write(t, &s, "Buzz")

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	expect(t, !ss2SelectPrev(t, search), "!try search.selectPrev()")
	expect(t, !search.selected.Valid, "search.selected == null")

	ss2Write(t, &s, "\r\nFizz\r\nFizz")
	zigTry(t, search.ReloadActive())
	// Oldest is the first of the active (forward ordered) results.
	expect(t, ss2SelectPrev(t, search), "try search.selectPrev()")
	ss2ExpectSel(t, search, 1, 0, 1, 3, 1)
	expect(t, ss2SelectPrev(t, search), "try search.selectPrev()")
	ss2ExpectSel(t, search, 0, 0, 2, 3, 2)
	expect(t, !ss2SelectPrev(t, search), "!try search.selectPrev()")
}

// Zig: test "term51: ScreenSearch.selectPrev: does not reload the active area"
func TestSs2_ScreenSearch_selectPrev_does_not_reload_the_active_area(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, 10, 5)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	ss2Write(t, &s, "Fizz\r\nBuzz\r\nFizz")

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	ss2Write(t, &s, "\x1b[2J\x1b[H")
	expect(t, ss2SelectPrev(t, search), "try search.selectPrev()")
	ss2ExpectSel(t, search, 1, 0, 0, 3, 0)
	expect(t, ss2SelectPrev(t, search), "try search.selectPrev()")
	ss2ExpectSel(t, search, 0, 0, 2, 3, 2)
	expect(t, !ss2SelectPrev(t, search), "!try search.selectPrev()")
}

// Zig: test "term51: ScreenSearch.pruneHistory: removes results from the first invalid serial on"
func TestSs2_ScreenSearch_pruneHistory_removes_results_from_the_first_invalid_serial_on(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2ThreePages(t, &s, list, "Fizz", "fIzZ", "ab\r\ncd")
	n1 := ss2First(t, list)
	if n1.next == nil {
		t.Fatalf("unwrapped a null optional")
	}
	n2 := n1.next
	expect(t, n1.serial == 0 && n2.serial == 1, "n1.serial == 0 and n2.serial == 1")

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expectEqual(t, 2, len(search.history_results))

	// The serials are only compared with the page list's minimum. Restore
	// it before anything else looks at the page list.
	defer func() { list.page_serial_min = 0 }()

	// Nothing is below the minimum.
	list.page_serial_min = 0
	search.pruneHistory()
	expectEqual(t, 2, len(search.history_results))

	// Serial 0 (the oldest, last in the list) is invalid: only it is dropped.
	list.page_serial_min = 1
	search.pruneHistory()
	expectEqual(t, 1, len(search.history_results))
	ss2ExpectAt(t, list, search.history_results[0], 0, ss2RowBase(list, 1), 3, ss2RowBase(list, 1))

	// A minimum equal to a serial keeps it, one more drops it.
	search.pruneHistory()
	expectEqual(t, 1, len(search.history_results))
	list.page_serial_min = 2
	search.pruneHistory()
	expectEqual(t, 0, len(search.history_results))

	// Nothing to prune is fine.
	search.pruneHistory()
	expectEqual(t, 0, len(search.history_results))
}

// Zig: test "term51: ScreenSearch.pruneHistory: invalid result drops everything older even if valid"
func TestSs2_ScreenSearch_pruneHistory_invalid_result_drops_everything_older_even_if_valid(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2ThreePages(t, &s, list, "Fizz", "fIzZ", "ab\r\ncd")

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	defer func() { list.page_serial_min = 0 }()

	// Swap the two results so the first one is the invalid one: the loop
	// trusts the order and drops the (valid) second one as well.
	search.history_results[0], search.history_results[1] = search.history_results[1], search.history_results[0]
	list.page_serial_min = 1
	search.pruneHistory()
	expectEqual(t, 0, len(search.history_results))
}

// Zig: test "term51: ScreenSearch.pruneHistory: no results is a no-op"
func TestSs2_ScreenSearch_pruneHistory_no_results_is_a_no_op(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, 10, 2)
	defer term.Deinit()
	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	search.pruneHistory()
	expectEqual(t, 0, len(search.history_results))
}

// Zig: test "term51: ScreenSearch.pruneHistory: only the first chunk serial is checked"
func TestSs2_ScreenSearch_pruneHistory_only_the_first_chunk_serial_is_checked(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	cap_ := ss2Cap(t, list)
	for i := uint(0); i < cap_-1; i++ {
		ss2Write(t, &s, "\r\n")
	}
	ss2Write(t, &s, ss2Xs+"Test")
	ss2GrowTo(t, &s, list, 3)
	ss2Write(t, &s, "ab\r\ncd")

	search := ss2Init(t, term.screens.active, "Test")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expectEqual(t, 1, len(search.history_results))
	m := search.history_results[0]
	// The match spans node 1 (serial 0) and node 2 (serial 1), but the
	// reverse search stored the serials newest first: [1, 0].
	expectEqual(t, uint64(1), m.chunks[0].serial)
	expectEqual(t, uint64(0), m.chunks[1].serial)

	defer func() { list.page_serial_min = 0 }()
	// Page 1 (serial 0) is gone but the match is kept because the first
	// chunk serial is the one of page 2.
	list.page_serial_min = 1
	search.pruneHistory()
	expectEqual(t, 1, len(search.history_results))
	list.page_serial_min = 2
	search.pruneHistory()
	expectEqual(t, 0, len(search.history_results))
}

// Zig: test "term51: ScreenSearch.reloadActive: pruned scrollback page reselects the oldest remaining match"
func TestSs2_ScreenSearch_reloadActive_pruned_scrollback_page_reselects_the_oldest_remaining_match(t *testing.T) {
	testStdCapacity(t)
	term := ss2NewSmall(t)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2PrunePrelude(t, &s, list)

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expectEqual(t, 1, len(search.active_results))
	expectEqual(t, 1, len(search.history_results))
	ss2Select(t, search, SearchScreenSelectPrev)
	ss2ExpectSel(t, search, 1, 0, 0, 3, 0)
	n2 := ss2Last(t, list)

	ss2PruneFirst(t, &s, list)
	expectEqual(t, uint64(1), list.page_serial_min)
	expectEqual(t, uint(2), ss2Nodes(list))
	expect(t, ss2First(t, list) == n2, "list.pages.first.? == n2")
	expect(t, ss2Selected(t, search).highlight.start.garbage, "search.selected.?.highlight.start.garbage")
	expect(t, ss2History(t, search).searcher.pin.garbage, "search.history.?.searcher.pin.garbage")
	expect(t, !ss2History(t, search).start_pin.garbage, "!search.history.?.start_pin.garbage")

	zigTry(t, search.ReloadActive())
	// Page 2 became history and its match was added, the stale result
	// from the pruned page was dropped, and the oldest was reselected.
	expectEqual(t, 1, len(search.active_results))
	expectEqual(t, 1, len(search.history_results))
	expect(t, ss2History(t, search).start_pin.node == ss2Last(t, list), "search.history.?.start_pin.node == list.pages.last.?")
	b := ss2RowBase(list, 1)
	ss2ExpectSel(t, search, 1, 0, 1, 3, 1)
	ss2ExpectRows(t, search, [][3]uint{{0, b, 3}, {0, 1, 3}})
}

// Zig: test "term51: ScreenSearch.select: garbage selection after pruning moves one past the oldest"
func TestSs2_ScreenSearch_select_garbage_selection_after_pruning_moves_one_past_the_oldest(t *testing.T) {
	testStdCapacity(t)
	term := ss2NewSmall(t)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2PrunePrelude(t, &s, list)

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	ss2Select(t, search, SearchScreenSelectPrev)
	ss2ExpectSel(t, search, 1, 0, 0, 3, 0)
	ss2PruneFirst(t, &s, list)

	// The reload inside select reselects the oldest (index 1) and then
	// the requested prev moves it one further towards the newest.
	expect(t, ss2Select(t, search, SearchScreenSelectPrev), "try search.select(.prev)")
	expectEqual(t, uint(2), search.MatchesLen())
	b := ss2RowBase(list, 1)
	ss2ExpectSel(t, search, 0, 0, b, 3, b)
}

// Zig: test "term51: ScreenSearch.select: garbage selection after pruning and next stays on the oldest"
func TestSs2_ScreenSearch_select_garbage_selection_after_pruning_and_next_stays_on_the_oldest(t *testing.T) {
	testStdCapacity(t)
	term := ss2NewSmall(t)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2PrunePrelude(t, &s, list)

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	ss2Select(t, search, SearchScreenSelectPrev)
	ss2PruneFirst(t, &s, list)

	// Reselected the oldest (index 1), next has nowhere to go.
	expect(t, !ss2Select(t, search, SearchScreenSelectNext), "!try search.select(.next)")
	ss2ExpectSel(t, search, 1, 0, 1, 3, 1)
}

// Zig: test "term51: ScreenSearch.feed: completing prunes stale history results"
func TestSs2_ScreenSearch_feed_completing_prunes_stale_history_results(t *testing.T) {
	testStdCapacity(t)
	term := ss2NewSmall(t)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2PrunePrelude(t, &s, list)

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expectEqual(t, searchScreenStateComplete, search.state)
	expectEqual(t, 1, len(search.history_results))

	ss2PruneFirst(t, &s, list)
	// The history searcher's pin sat on the pruned page.
	expect(t, ss2History(t, search).searcher.pin.garbage, "search.history.?.searcher.pin.garbage")
	expectEqual(t, 1, len(search.history_results))

	// A complete search still needs feeds: it cleans up the stale result.
	expect(t, search.state.NeedsFeed(), "search.state.needsFeed()")
	zigTry(t, search.Feed())
	expectEqual(t, searchScreenStateComplete, search.state)
	expectEqual(t, 0, len(search.history_results))
	// The active results are untouched (not reloaded).
	expectEqual(t, 1, len(search.active_results))
	ss2ExpectRows(t, search, [][3]uint{{0, 1, 3}})
}

// Zig: test "term51: ScreenSearch.init: match across the first active page and the next is missed"
func TestSs2_ScreenSearch_init_match_across_the_first_active_page_and_the_next_is_missed(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	cap_ := ss2Cap(t, list)
	for i := uint(0); i < cap_-1; i++ {
		ss2Write(t, &s, "\r\n")
	}
	// "Te" ends page 1, "st" wraps onto the first row of page 2. The
	// active area (two rows) is the last row of page 1 and page 2's row.
	ss2Write(t, &s, ss2Xs+"Test")
	expectEqual(t, uint(2), ss2Nodes(list))

	search := ss2Init(t, term.screens.active, "Test")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	// The active search appends the last page first, so the text is not in
	// order; the history search never looks at pages after its start node.
	expectEqual(t, 0, len(search.active_results))
	expectEqual(t, 0, len(search.history_results))
	expectEqual(t, uint(0), search.MatchesLen())
}

// Zig: test "term51: ScreenSearch.init: match across pages is found when the active area is one page"
func TestSs2_ScreenSearch_init_match_across_pages_is_found_when_the_active_area_is_one_page(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	cap_ := ss2Cap(t, list)
	for i := uint(0); i < cap_-1; i++ {
		ss2Write(t, &s, "\r\n")
	}
	ss2Write(t, &s, ss2Xs+"Test\r\n")
	expectEqual(t, uint(2), ss2Nodes(list))

	search := ss2Init(t, term.screens.active, "Test")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	// Found by the history search, which starts at page 2.
	expectEqual(t, 0, len(search.active_results))
	expectEqual(t, 1, len(search.history_results))
	ss2ExpectAt(t, list, search.history_results[0], ss2Cols-2, cap_-1, 1, cap_)
}

// Zig: test "term51: ScreenSearch.init: active results across two pages are not ordered newest first"
func TestSs2_ScreenSearch_init_active_results_across_two_pages_are_not_ordered_newest_first(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	cap_ := ss2Cap(t, list)
	for i := uint(0); i < cap_-1; i++ {
		ss2Write(t, &s, "\r\n")
	}
	// Last row of page 1 and first row of page 2: both are active.
	ss2Write(t, &s, "Fizz\r\nFizz")
	expectEqual(t, uint(2), ss2Nodes(list))

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expectEqual(t, 2, len(search.active_results))
	expectEqual(t, 0, len(search.history_results))

	// The active window gets the last page first so the stored results are
	// newest first and matches() (which reverses them) is oldest first.
	ss2ExpectAt(t, list, search.active_results[0], 0, cap_, 3, cap_)
	ss2ExpectAt(t, list, search.active_results[1], 0, cap_-1, 3, cap_-1)
	ss2ExpectRows(t, search, [][3]uint{{0, cap_ - 1, 3}, {0, cap_, 3}})

	// So "next" (documented as newest first) selects the older match first.
	expect(t, ss2Select(t, search, SearchScreenSelectNext), "try search.select(.next)")
	ss2ExpectSel(t, search, 0, 0, cap_-1, 3, cap_-1)
	expect(t, ss2Select(t, search, SearchScreenSelectNext), "try search.select(.next)")
	ss2ExpectSel(t, search, 1, 0, cap_, 3, cap_)
}

// Zig: test "term51: ScreenSearch.init: match in the page before the active area is found twice"
func TestSs2_ScreenSearch_init_match_in_the_page_before_the_active_area_is_found_twice(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	cap_ := ss2Cap(t, list)
	ss2Write(t, &s, "Fizz")
	for i := uint(0); i < cap_-2; i++ {
		ss2Write(t, &s, "\r\n")
	}
	// The last row of page 1 is soft wrapped into page 2, which holds the
	// whole active area. The active search adds page 1 as overlap.
	ss2Write(t, &s, "\r\n"+ss2Full+"y\r\nzz")
	expectEqual(t, uint(2), ss2Nodes(list))

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	// Found by the active search (the whole overlap page is searched) and
	// again by the history search.
	expectEqual(t, 1, len(search.active_results))
	expectEqual(t, 1, len(search.history_results))
	expectEqual(t, uint(2), search.MatchesLen())
	ss2ExpectRows(t, search, [][3]uint{{0, 0, 3}, {0, 0, 3}})
}

// Zig: test "term51: ScreenSearch.reloadActive: resize without feed keeps stale dimensions and tracks the selection"
func TestSs2_ScreenSearch_reloadActive_resize_without_feed_keeps_stale_dimensions_and_tracks_the_selection(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, 10, 3)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2Write(t, &s, "Fizz")

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	ss2Select(t, search, SearchScreenSelectNext)
	ss2ExpectSel(t, search, 0, 0, 0, 3, 0)

	// "Fiz" / "z": the selection pins move with the text.
	zigTry(t, term.Resize(3, 3))
	zigTry(t, search.ReloadActive())
	expectEqual(t, SizeCellCountInt(10), search.cols)
	expectEqual(t, SizeCellCountInt(3), search.rows)
	expectEqual(t, uint(1), search.MatchesLen())
	ss2ExpectSel(t, search, 0, 0, 0, 0, 1)
	expectEqual(t, SizeCellCountInt(2), search.active_results[0].chunks[0].end)

	// A feed notices the size and starts over: the selection is dropped.
	zigTry(t, search.Feed())
	expectEqual(t, SizeCellCountInt(3), search.cols)
	expect(t, !search.selected.Valid, "search.selected == null")
	expectEqual(t, uint(1), search.MatchesLen())
	ss2ExpectAt(t, list, search.active_results[0], 0, 0, 0, 1)
}

// Zig: test "term51: ScreenSearch.init: matching is ASCII case insensitive over UTF-8 bytes"
func TestSs2_ScreenSearch_init_matching_is_ASCII_case_insensitive_over_UTF_8_bytes(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, 10, 2)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	ss2Write(t, &s, "caf\xc3\xa9 CAF")

	func() {
		search := ss2Init(t, term.screens.active, "caf\xc3\xa9")
		defer search.Deinit()
		ss2ExpectRows(t, search, [][3]uint{{0, 0, 3}})
	}()
	func() {
		// Only ASCII letters fold: a different case of the e acute is a
		// different byte sequence.
		search := ss2Init(t, term.screens.active, "caf\xc3\x89")
		defer search.Deinit()
		expectEqual(t, uint(0), search.MatchesLen())
	}()
	func() {
		// One cell for a two byte character.
		search := ss2Init(t, term.screens.active, "\xc3\xa9")
		defer search.Deinit()
		ss2ExpectRows(t, search, [][3]uint{{3, 0, 3}})
	}()
	func() {
		search := ss2Init(t, term.screens.active, "caf")
		defer search.Deinit()
		ss2ExpectRows(t, search, [][3]uint{{5, 0, 7}, {0, 0, 2}})
	}()
}

// Zig: test "term51: ScreenSearch.init: a needle with a newline matches across hard line breaks"
func TestSs2_ScreenSearch_init_a_needle_with_a_newline_matches_across_hard_line_breaks(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, 10, 3)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2Write(t, &s, "Fizz\r\nBuzz")

	search := ss2Init(t, term.screens.active, "Fizz\nBuzz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expectEqual(t, uint(1), search.MatchesLen())
	m := search.active_results[0]
	ss2ExpectAt(t, list, m, 0, 0, 3, 1)
	expectEqual(t, 1, len(m.chunks))
	expectEqual(t, SizeCellCountInt(0), m.chunks[0].start)
	expectEqual(t, SizeCellCountInt(2), m.chunks[0].end)
	expectEqual(t, SizeCellCountInt(0), m.top_x)
	expectEqual(t, SizeCellCountInt(3), m.bot_x)
}

// Go-only: panic test: SearchScreen.Feed: complete state while the history searcher still has data to feed
func TestSs2_panic_Feed_complete_state_with_history_to_feed(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2ThreePages(t, &s, list, "Fizz", "fIzZ", "FIZZ\r\nbuzz")

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	// Zig: `.complete => unreachable` (the history feed returned true, which
	// a complete search never does).
	search.state = searchScreenStateComplete
	ss2ExpectPanic(t, func() { _ = search.Feed() })
}

// Go-only: panic test: SearchScreen.ReloadActive: a selection in the history whose highlight no longer matches its result
func TestSs2_panic_ReloadActive_history_selection_does_not_match_its_result(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, ss2Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss2ThreePages(t, &s, list, "Fizz", "fIzZ", "FIZZ\r\nbuzz")

	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	ss2Select(t, search, SearchScreenSelectNext)
	ss2Select(t, search, SearchScreenSelectNext)
	b2 := ss2RowBase(list, 1)
	ss2ExpectSel(t, search, 1, 0, b2, 3, b2)

	// Move on to a fourth page, the third becomes history: the selection
	// index is shifted and upstream asserts that the shifted index still
	// names the selected match (assert(m.highlight.start.eql(hl.startPin()))).
	ss2GrowTo(t, &s, list, 4)
	ss2Write(t, &s, "FiZz\r\nxx")
	ss2Selected(t, search).highlight.start.x++
	ss2ExpectPanic(t, func() { _ = search.ReloadActive() })
}

// Go-only: panic test: SearchScreen.pruneHistory: a history result without chunks
func TestSs2_panic_pruneHistory_result_without_chunks(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, 10, 2)
	defer term.Deinit()
	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	// `serials[0]` of an empty chunk list.
	search.history_results = []HighlightFlattened{HighlightFlattenedEmpty}
	ss2ExpectPanic(t, func() { search.pruneHistory() })
	search.history_results = nil
}

// Go-only: panic test: SearchScreen.selectNext: the selection index is the maximum usize (idx + 1 overflows)
func TestSs2_panic_selectNext_index_overflow(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, 10, 2)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	ss2Write(t, &s, ss2Text)
	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	ss2Select(t, search, SearchScreenSelectNext)
	ss2Selected(t, search).idx = math.MaxUint
	// `const next_idx = prev.idx + 1;` traps on overflow in a safe build.
	ss2ExpectPanic(t, func() { _, _ = search.selectNext() })
}

// Go-only: panic test: SearchScreen.Needle: the active window is not a forward search
func TestSs2_panic_Needle_direction_is_not_forward(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, 10, 2)
	defer term.Deinit()
	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	search.active.window.direction = searchSlidingWindowDirectionReverse
	ss2ExpectPanic(t, func() { _ = search.Needle() })
	search.active.window.direction = searchSlidingWindowDirectionForward
}

// Go-only: panic test: SearchScreen.Matches: the active window is not a forward search
func TestSs2_panic_Matches_direction_is_not_forward(t *testing.T) {
	testStdCapacity(t)
	term := ss2New(t, 10, 2)
	defer term.Deinit()
	search := ss2Init(t, term.screens.active, "Fizz")
	defer search.Deinit()
	search.active.window.direction = searchSlidingWindowDirectionReverse
	ss2ExpectPanic(t, func() { _, _ = search.Matches() })
	search.active.window.direction = searchSlidingWindowDirectionForward
}
