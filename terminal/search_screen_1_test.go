// Tests translated one for one from src/terminal/search/screen.zig (Ghostty
// v1.3.1 plus the term51 additional tests), lines 831 to 2798: the upstream
// ScreenSearch tests and the term51 tests from the State tests up to
// "ScreenSearch.reloadActive: selected match gone selects the newest". Each
// Go test carries the exact Zig title in a "// Zig: test" comment. Helpers
// are prefixed ss1.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualStrings keep Zig's names and Zig's (expected,
//     actual) argument order. `try` on a call that returns an error is
//     zigTry(t, err); `x.?` is zigUnwrap. `testing.expectError` is
//     ss1ExpectError.
//   - `var t: Terminal = try .init(alloc, .{ .cols = c, .rows = r })` is
//     `term := ss1Init(t, c, r)` (terminalOptionsDefault: max_scrollback
//     10_000); the tests' term51AMNew (max_scrollback maxInt(usize)) is
//     ss1New; with another max_scrollback it is ss1InitScrollback. The Go
//     testing parameter is `t`, as scripts/check-test-mapping.py requires,
//     so Zig's Terminal `t` is called `term`.
//   - `.{ .screen = .{ .x = x, .y = y } }` and the `.active` form are
//     ss1Screen and ss1Active.
//   - `search.history_results.items.len` is len(search.history_results),
//     `chunks.items(.start)[0]` is chunks[0].start, `chunks.len` is
//     len(chunks), `x.?` on an Optional field is zigUnwrap(t, x), and
//     `x == null` is !x.Valid.
//   - Zig's `alloc` arguments are dropped; `defer alloc.free(...)` leak
//     checks are dropped with `// Zig-only:` comments.
//   - Raw-pointer comparisons: node pointers are compared as Go pointers;
//     `chunks.bytes == other.chunks.bytes` (shared chunk memory) is a
//     comparison of the address of the first chunk; `needle().ptr != &buf`
//     is a comparison of the address of the first needle byte.
//   - Every test calls testStdCapacity first (Zig's unit tests run with
//     grapheme_bytes 512).
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt.

package terminal

import (
	"errors"
	"strings"
	"testing"
)

// ss1ExpectPanic runs f, which must panic. A panic with the wireframe's
// "not implemented" value does not count: it only means the function under
// test does not exist yet. The setup is done outside f.
func ss1ExpectPanic(t *testing.T, f func()) {
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

// ss1ExpectError is `try testing.expectError(want, got)`.
func ss1ExpectError(t *testing.T, want error, got error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("expected error %v, found %v", want, got)
	}
}

// ss1Init is `try Terminal.init(alloc, .{ .cols = cols, .rows = rows })`;
// the terminal is returned by pointer (the methods take a pointer receiver).
func ss1Init(tt *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) *Terminal {
	tt.Helper()
	term, err := TerminalInit(terminalOptionsDefault(cols, rows))
	zigTry(tt, err)
	return &term
}

// ss1InitScrollback is `try Terminal.init(alloc, .{ .cols = cols, .rows =
// rows, .max_scrollback = max_scrollback })`.
func ss1InitScrollback(tt *testing.T, cols SizeCellCountInt, rows SizeCellCountInt, max_scrollback uint64) *Terminal {
	tt.Helper()
	opts := terminalOptionsDefault(cols, rows)
	opts.max_scrollback = max_scrollback
	term, err := TerminalInit(opts)
	zigTry(tt, err)
	return &term
}

// ss1MaxScrollback is std.math.maxInt(usize) of the recorded 64-bit build.
const ss1MaxScrollback = ^uint64(0)

// ss1Screen and ss1Active are the Zig literals
// `point.Point{ .screen = .{ .x = x, .y = y } }` and the `.active` form.
func ss1Screen(x uint, y uint) Point {
	return Point{Tag: PointTagScreen, Screen: PointCoordinate{x: SizeCellCountInt(x), y: uint32(y)}}
}

func ss1Active(x uint, y uint) Point {
	return Point{Tag: PointTagActive, Active: PointCoordinate{x: SizeCellCountInt(x), y: uint32(y)}}
}

// ss1ExpectPoint is `try testing.expectEqual(want,
// list.pointFromPin(tag, pin).?)`.
func ss1ExpectPoint(tt *testing.T, list *PageList, tag PointTag, pin PageListPin, want Point) {
	tt.Helper()
	expectEqual(tt, want, zigUnwrap(tt, list.PointFromPin(tag, pin)))
}

// ss1NonNil is `node.?`.
func ss1NonNil(tt *testing.T, node *PageListNode, zig string) *PageListNode {
	tt.Helper()
	expect(tt, node != nil, zig)
	return node
}

// ss1Text is term51AMText: four rows of text, two of them "Fizz". On a 10x2
// terminal the first two rows scroll into the history (but stay on the same
// page).
const ss1Text = "Fizz\r\nBuzz\r\nFizz\r\nBang"

// ss1Cols is term51AMCols: multi page tests use the widest standard page
// (215 columns) because it has the fewest rows per page (215), which keeps
// filling pages cheap.
const ss1Cols = 215

// ss1Xs and ss1Full are term51AMXs and term51AMFull: a row of "x" up to the
// last two cells, and a full row.
var (
	ss1Xs   = strings.Repeat("x", ss1Cols-2)
	ss1Full = strings.Repeat("x", ss1Cols)
)

// ss1New is term51AMNew.
func ss1New(tt *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) *Terminal {
	tt.Helper()
	return ss1InitScrollback(tt, cols, rows, ss1MaxScrollback)
}

// ss1Cap is term51AMCap: the row capacity of the first page.
func ss1Cap(list *PageList) uint {
	return uint(list.pages.first.data.capacity.rows)
}

// ss1Nodes is term51AMNodes.
func ss1Nodes(list *PageList) uint {
	return list.TotalPages()
}

// ss1RowBase is term51AMRowBase: the number of screen rows that come before
// the node with the given index.
func ss1RowBase(list *PageList, index uint) uint {
	var rows uint
	node := list.pages.first
	var i uint
	for node != nil {
		if i == index {
			return rows
		}
		rows += uint(node.data.size.rows)
		i += 1
		node = node.next
	}
	panic("unreachable")
}

// ss1GrowTo is term51AMGrowTo: write newlines until the list has `n` nodes.
// The cursor is then on the first row of the last node.
func ss1GrowTo(tt *testing.T, s *StreamReadonlyStream, list *PageList, n uint) {
	tt.Helper()
	for ss1Nodes(list) < n {
		zigTry(tt, s.NextSlice([]uint8("\r\n")))
	}
}

// ss1ThreePages is term51AMThreePages: builds three pages. Each text is
// written at the first row of its page, the last text may contain further
// lines. The first page is left as soon as the second text starts, so only
// `p3` may contain newlines.
func ss1ThreePages(tt *testing.T, s *StreamReadonlyStream, list *PageList, p1 string, p2 string, p3 string) {
	tt.Helper()
	zigTry(tt, s.NextSlice([]uint8(p1)))
	ss1GrowTo(tt, s, list, 2)
	zigTry(tt, s.NextSlice([]uint8(p2)))
	ss1GrowTo(tt, s, list, 3)
	zigTry(tt, s.NextSlice([]uint8(p3)))
}

// ss1ExpectAt is term51AMExpectAt: expect the match to span exactly the
// given screen coordinates.
func ss1ExpectAt(tt *testing.T, list *PageList, hl HighlightFlattened, sx uint, sy uint, ex uint, ey uint) {
	tt.Helper()
	u := hl.Untracked()
	expectEqual(tt, ss1Screen(sx, sy), zigUnwrap(tt, list.PointFromPin(PointTagScreen, u.start)))
	expectEqual(tt, ss1Screen(ex, ey), zigUnwrap(tt, list.PointFromPin(PointTagScreen, u.end)))
}

// ss1ExpectSel is term51AMExpectSel: expect the selected match to be at the
// given screen coordinates and index.
func ss1ExpectSel(tt *testing.T, search *SearchScreen, idx uint, sx uint, sy uint, ex uint, ey uint) {
	tt.Helper()
	expectEqual(tt, idx, zigUnwrap(tt, search.selected).idx)
	ss1ExpectAt(tt, &search.screen.pages, zigUnwrap(tt, search.SelectedMatch()), sx, sy, ex, ey)
}

// ss1ExpectRows is term51AMExpectRows: expect the match list to have exactly
// the given single-row matches (start x, y, end x), in order (newest first).
func ss1ExpectRows(tt *testing.T, search *SearchScreen, expected [][3]uint) {
	tt.Helper()
	ms, err := search.Matches()
	zigTry(tt, err)
	// Zig-only: defer alloc.free(ms);
	expectEqual(tt, len(expected), len(ms))
	for i, e := range expected {
		ss1ExpectAt(tt, &search.screen.pages, ms[i], e[0], e[1], e[2], e[1])
	}
}

// Zig: test "simple search"
func TestSs1_simple_search(t *testing.T) {
	testStdCapacity(t)
	term := ss1Init(t, 10, 2)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz\r\nBuzz\r\nFizz\r\nBang")))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expectEqual(t, 2, len(search.active_results))
	// We don't test history results since there is overlap

	// Get all matches
	matches, err := search.Matches()
	zigTry(t, err)
	// Zig-only: defer alloc.free(matches);
	expectEqual(t, 2, len(matches))

	{
		sel := matches[0].Untracked()
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.start, ss1Screen(0, 2))
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.end, ss1Screen(3, 2))
	}
	{
		sel := matches[1].Untracked()
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.start, ss1Screen(0, 0))
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.end, ss1Screen(3, 0))
	}
}

// Zig: test "simple search with history"
func TestSs1_simple_search_with_history(t *testing.T) {
	testStdCapacity(t)
	term := ss1InitScrollback(t, 10, 2, ss1MaxScrollback)
	defer term.Deinit()
	list := &term.screens.active.pages

	s := term.VtStream()
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("Fizz\r\n")))
	for list.TotalPages() < 3 {
		zigTry(t, s.NextSlice([]uint8("\r\n")))
	}
	for range list.rows {
		zigTry(t, s.NextSlice([]uint8("\r\n")))
	}
	zigTry(t, s.NextSlice([]uint8("hello.")))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expectEqual(t, 0, len(search.active_results))

	// Get all matches
	matches, err := search.Matches()
	zigTry(t, err)
	// Zig-only: defer alloc.free(matches);
	expectEqual(t, 1, len(matches))

	{
		sel := matches[0].Untracked()
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.start, ss1Screen(0, 0))
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.end, ss1Screen(3, 0))
	}
}

// Zig: test "reload active with history change"
func TestSs1_reload_active_with_history_change(t *testing.T) {
	testStdCapacity(t)
	term := ss1InitScrollback(t, 10, 2, ss1MaxScrollback)
	defer term.Deinit()
	list := &term.screens.active.pages

	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz\r\n")))

	// Start up our search which will populate our initial active area.
	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	{
		matches, err := search.Matches()
		zigTry(t, err)
		// Zig-only: defer alloc.free(matches);
		expectEqual(t, 1, len(matches))
	}

	// Grow into two pages so our history pin will move.
	for list.TotalPages() < 2 {
		zigTry(t, s.NextSlice([]uint8("\r\n")))
	}
	for range list.rows {
		zigTry(t, s.NextSlice([]uint8("\r\n")))
	}
	zigTry(t, s.NextSlice([]uint8("2Fizz")))

	// Active area changed so reload
	zigTry(t, search.ReloadActive())
	zigTry(t, search.SearchAll())

	// Get all matches
	{
		matches, err := search.Matches()
		zigTry(t, err)
		// Zig-only: defer alloc.free(matches);
		expectEqual(t, 2, len(matches))
		{
			sel := matches[1].Untracked()
			ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.start, ss1Screen(0, 0))
			ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.end, ss1Screen(3, 0))
		}
		{
			sel := matches[0].Untracked()
			ss1ExpectPoint(t, &term.screens.active.pages, PointTagActive, sel.start, ss1Active(1, 1))
			ss1ExpectPoint(t, &term.screens.active.pages, PointTagActive, sel.end, ss1Active(4, 1))
		}
	}

	// Reset the screen which will make our pin garbage.
	term.FullReset()
	zigTry(t, s.NextSlice([]uint8("WeFizzing")))
	zigTry(t, search.ReloadActive())
	zigTry(t, search.SearchAll())

	{
		matches, err := search.Matches()
		zigTry(t, err)
		// Zig-only: defer alloc.free(matches);
		expectEqual(t, 1, len(matches))
		{
			sel := matches[0].Untracked()
			ss1ExpectPoint(t, &term.screens.active.pages, PointTagActive, sel.start, ss1Active(2, 0))
			ss1ExpectPoint(t, &term.screens.active.pages, PointTagActive, sel.end, ss1Active(5, 0))
		}
	}
}

// Zig: test "active change contents"
func TestSs1_active_change_contents(t *testing.T) {
	testStdCapacity(t)
	term := ss1Init(t, 10, 5)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fuzz\r\nBuzz\r\nFizz\r\nBang")))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expectEqual(t, 1, len(search.active_results))

	// Erase the screen, move our cursor to the top, and change contents.
	zigTry(t, s.NextSlice([]uint8("\x1b[2J\x1b[H"))) // Clear screen and move home
	zigTry(t, s.NextSlice([]uint8("Bang\r\nFizz\r\nHello!")))

	zigTry(t, search.ReloadActive())
	zigTry(t, search.SearchAll())
	expectEqual(t, 1, len(search.active_results))

	// Get all matches
	matches, err := search.Matches()
	zigTry(t, err)
	// Zig-only: defer alloc.free(matches);
	expectEqual(t, 1, len(matches))

	{
		sel := matches[0].Untracked()
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.start, ss1Screen(0, 1))
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.end, ss1Screen(3, 1))
	}
}

// Zig: test "select next"
func TestSs1_select_next(t *testing.T) {
	testStdCapacity(t)
	term := ss1Init(t, 10, 2)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz\r\nBuzz\r\nFizz\r\nBang")))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()

	// Initially no selection
	expect(t, !search.SelectedMatch().Valid, "search.selectedMatch() == null")

	// Select our next match (first)
	zigTry(t, search.SearchAll())
	_, err = search.Select(SearchScreenSelectNext)
	zigTry(t, err)
	{
		sel := zigUnwrap(t, search.SelectedMatch()).Untracked()
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.start, ss1Screen(0, 2))
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.end, ss1Screen(3, 2))
	}

	// Next match
	_, err = search.Select(SearchScreenSelectNext)
	zigTry(t, err)
	{
		sel := zigUnwrap(t, search.SelectedMatch()).Untracked()
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.start, ss1Screen(0, 0))
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.end, ss1Screen(3, 0))
	}

	// Next match (no wrap)
	_, err = search.Select(SearchScreenSelectNext)
	zigTry(t, err)
	{
		sel := zigUnwrap(t, search.SelectedMatch()).Untracked()
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.start, ss1Screen(0, 0))
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.end, ss1Screen(3, 0))
	}
}

// Zig: test "select in active changes contents completely"
func TestSs1_select_in_active_changes_contents_completely(t *testing.T) {
	testStdCapacity(t)
	term := ss1Init(t, 10, 5)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz\r\nBuzz\r\nFizz\r\nBang")))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	_, err = search.Select(SearchScreenSelectNext)
	zigTry(t, err)
	_, err = search.Select(SearchScreenSelectNext)
	zigTry(t, err)
	{
		// Initial selection is the first fizz
		sel := zigUnwrap(t, search.SelectedMatch()).Untracked()
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.start, ss1Screen(0, 0))
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.end, ss1Screen(3, 0))
	}

	// Erase the screen, move our cursor to the top, and change contents.
	zigTry(t, s.NextSlice([]uint8("\x1b[2J\x1b[H"))) // Clear screen and move home
	zigTry(t, s.NextSlice([]uint8("Fuzz\r\nFizz\r\nHello!")))

	zigTry(t, search.ReloadActive())
	{
		// Our selection should move to the first
		sel := zigUnwrap(t, search.SelectedMatch()).Untracked()
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.start, ss1Screen(0, 1))
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.end, ss1Screen(3, 1))
	}

	// Erase the screen, redraw with same contents.
	zigTry(t, s.NextSlice([]uint8("\x1b[2J\x1b[H"))) // Clear screen and move home
	zigTry(t, s.NextSlice([]uint8("Fuzz\r\nFizz\r\nFizz")))

	zigTry(t, search.ReloadActive())
	{
		// Our selection should not move to the first
		sel := zigUnwrap(t, search.SelectedMatch()).Untracked()
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.start, ss1Screen(0, 1))
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.end, ss1Screen(3, 1))
	}
}

// Zig: test "select into history"
func TestSs1_select_into_history(t *testing.T) {
	testStdCapacity(t)
	term := ss1InitScrollback(t, 10, 2, ss1MaxScrollback)
	defer term.Deinit()
	list := &term.screens.active.pages

	s := term.VtStream()
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("Fizz\r\n")))
	for list.TotalPages() < 3 {
		zigTry(t, s.NextSlice([]uint8("\r\n")))
	}
	for range list.rows {
		zigTry(t, s.NextSlice([]uint8("\r\n")))
	}
	zigTry(t, s.NextSlice([]uint8("hello.")))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())

	// Get all matches
	_, err = search.Select(SearchScreenSelectNext)
	zigTry(t, err)
	{
		sel := zigUnwrap(t, search.SelectedMatch()).Untracked()
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.start, ss1Screen(0, 0))
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.end, ss1Screen(3, 0))
	}

	// Erase the screen, redraw with same contents.
	zigTry(t, s.NextSlice([]uint8("\x1b[2J\x1b[H"))) // Clear screen and move home
	zigTry(t, s.NextSlice([]uint8("yo yo")))

	zigTry(t, search.ReloadActive())
	{
		// Our selection should not move since the history is still active.
		sel := zigUnwrap(t, search.SelectedMatch()).Untracked()
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.start, ss1Screen(0, 0))
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.end, ss1Screen(3, 0))
	}

	// Create some new history by adding more lines.
	zigTry(t, s.NextSlice([]uint8("\r\nfizz\r\nfizz\r\nfizz"))) // Clear screen and move home
	zigTry(t, search.ReloadActive())
	{
		// Our selection should not move since the history is still not
		// pruned.
		sel := zigUnwrap(t, search.SelectedMatch()).Untracked()
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.start, ss1Screen(0, 0))
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.end, ss1Screen(3, 0))
	}
}

// Zig: test "select prev"
func TestSs1_select_prev(t *testing.T) {
	testStdCapacity(t)
	term := ss1Init(t, 10, 2)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz\r\nBuzz\r\nFizz\r\nBang")))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()

	// Initially no selection
	expect(t, !search.SelectedMatch().Valid, "search.selectedMatch() == null")

	// Select prev (oldest first)
	zigTry(t, search.SearchAll())
	_, err = search.Select(SearchScreenSelectPrev)
	zigTry(t, err)
	{
		sel := zigUnwrap(t, search.SelectedMatch()).Untracked()
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.start, ss1Screen(0, 0))
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.end, ss1Screen(3, 0))
	}

	// Prev match (towards newest)
	_, err = search.Select(SearchScreenSelectPrev)
	zigTry(t, err)
	{
		sel := zigUnwrap(t, search.SelectedMatch()).Untracked()
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.start, ss1Screen(0, 2))
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.end, ss1Screen(3, 2))
	}

	// Prev match (no wrap, stays at newest)
	_, err = search.Select(SearchScreenSelectPrev)
	zigTry(t, err)
	{
		sel := zigUnwrap(t, search.SelectedMatch()).Untracked()
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.start, ss1Screen(0, 2))
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.end, ss1Screen(3, 2))
	}
}

// Zig: test "select prev then next"
func TestSs1_select_prev_then_next(t *testing.T) {
	testStdCapacity(t)
	term := ss1Init(t, 10, 2)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz\r\nBuzz\r\nFizz\r\nBang")))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())

	// Select next (newest first)
	_, err = search.Select(SearchScreenSelectNext)
	zigTry(t, err)
	{
		sel := zigUnwrap(t, search.SelectedMatch()).Untracked()
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.start, ss1Screen(0, 2))
	}

	// Select next (older)
	_, err = search.Select(SearchScreenSelectNext)
	zigTry(t, err)
	{
		sel := zigUnwrap(t, search.SelectedMatch()).Untracked()
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.start, ss1Screen(0, 0))
	}

	// Select prev (back to newer)
	_, err = search.Select(SearchScreenSelectPrev)
	zigTry(t, err)
	{
		sel := zigUnwrap(t, search.SelectedMatch()).Untracked()
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.start, ss1Screen(0, 2))
	}
}

// Zig: test "select prev with history"
func TestSs1_select_prev_with_history(t *testing.T) {
	testStdCapacity(t)
	term := ss1InitScrollback(t, 10, 2, ss1MaxScrollback)
	defer term.Deinit()
	list := &term.screens.active.pages

	s := term.VtStream()
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("Fizz\r\n")))
	for list.TotalPages() < 3 {
		zigTry(t, s.NextSlice([]uint8("\r\n")))
	}
	for range list.rows {
		zigTry(t, s.NextSlice([]uint8("\r\n")))
	}
	zigTry(t, s.NextSlice([]uint8("Fizz.")))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())

	// Select prev (oldest first, should be in history)
	_, err = search.Select(SearchScreenSelectPrev)
	zigTry(t, err)
	{
		sel := zigUnwrap(t, search.SelectedMatch()).Untracked()
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.start, ss1Screen(0, 0))
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagScreen, sel.end, ss1Screen(3, 0))
	}

	// Select prev (towards newer, should move to active area)
	_, err = search.Select(SearchScreenSelectPrev)
	zigTry(t, err)
	{
		sel := zigUnwrap(t, search.SelectedMatch()).Untracked()
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagActive, sel.start, ss1Active(0, 1))
		ss1ExpectPoint(t, &term.screens.active.pages, PointTagActive, sel.end, ss1Active(3, 1))
	}
}

// Zig: test "screen search no scrollback has no history"
func TestSs1_screen_search_no_scrollback_has_no_history(t *testing.T) {
	testStdCapacity(t)
	term := ss1InitScrollback(t, 10, 2, 0)
	defer term.Deinit()

	// Alt screen has no scrollback
	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)

	s := term.VtStream()
	defer s.Deinit()

	// This will probably stop working at some point and we'll have
	// no way to test it using public APIs, but at the time of writing
	// this test, CSI 22 J (scroll complete) pushes into scrollback
	// with alt screen.
	zigTry(t, s.NextSlice([]uint8("Fizz\r\n")))
	zigTry(t, s.NextSlice([]uint8("\x1b[22J")))
	zigTry(t, s.NextSlice([]uint8("hello.")))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expectEqual(t, 0, len(search.active_results))

	// Get all matches
	matches, err := search.Matches()
	zigTry(t, err)
	// Zig-only: defer alloc.free(matches);
	expectEqual(t, 0, len(matches))
}

// Zig: test "term51: ScreenSearch.State.isComplete: only complete is complete"
func TestSs1_term51_ScreenSearch_State_isComplete_only_complete_is_complete(t *testing.T) {
	testStdCapacity(t)
	expect(t, !searchScreenStateActive.IsComplete(), "!State.isComplete(.active)")
	expect(t, !searchScreenStateHistory.IsComplete(), "!State.isComplete(.history)")
	expect(t, !searchScreenStateHistoryFeed.IsComplete(), "!State.isComplete(.history_feed)")
	expect(t, searchScreenStateComplete.IsComplete(), "State.isComplete(.complete)")
}

// Zig: test "term51: ScreenSearch.State.needsFeed: history_feed and complete need a feed"
func TestSs1_term51_ScreenSearch_State_needsFeed_history_feed_and_complete_need_a_feed(t *testing.T) {
	testStdCapacity(t)
	expect(t, !searchScreenStateActive.NeedsFeed(), "!State.needsFeed(.active)")
	expect(t, !searchScreenStateHistory.NeedsFeed(), "!State.needsFeed(.history)")
	expect(t, searchScreenStateHistoryFeed.NeedsFeed(), "State.needsFeed(.history_feed)")
	expect(t, searchScreenStateComplete.NeedsFeed(), "State.needsFeed(.complete)")
}

// Zig: test "term51: ScreenSearch.init: fresh search with scrollback"
func TestSs1_term51_ScreenSearch_init_fresh_search_with_scrollback(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8(ss1Text)))

	base := list.CountTrackedPins()
	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()

	// The active area is searched immediately (the whole page, including
	// scrollback rows that share the page), history is not searched yet.
	expect(t, search.screen == term.screens.active, "search.screen == t.screens.active")
	expectEqual(t, searchScreenStateHistory, search.state)
	expectEqual(t, SizeCellCountInt(2), search.rows)
	expectEqual(t, SizeCellCountInt(10), search.cols)
	expect(t, !search.selected.Valid, "search.selected == null")
	expectEqual(t, 2, len(search.active_results))
	expectEqual(t, 0, len(search.history_results))
	expectEqual(t, uint(2), search.MatchesLen())
	expectEqualStrings(t, "Fizz", string(search.Needle()))

	// With scrollback there is always a history searcher starting at the
	// first node of the active area, with two tracked pins.
	expect(t, search.history.Valid, "search.history != null")
	expect(t, search.history.Value.start_pin.node == ss1NonNil(t, list.pages.first, "list.pages.first.?"), "search.history.?.start_pin.node == list.pages.first.?")
	expect(t, search.history.Value.start_pin != search.history.Value.searcher.pin, "search.history.?.start_pin != search.history.?.searcher.pin")
	expectEqual(t, base+2, list.CountTrackedPins())

	ss1ExpectRows(t, &search, [][3]uint{{0, 2, 3}, {0, 0, 3}})
}

// Zig: test "term51: ScreenSearch.init: no scrollback has no history searcher"
func TestSs1_term51_ScreenSearch_init_no_scrollback_has_no_history_searcher(t *testing.T) {
	testStdCapacity(t)
	term := ss1InitScrollback(t, 10, 2, 0)
	defer term.Deinit()
	list := &term.screens.active.pages
	expect(t, term.screens.active.no_scrollback, "t.screens.active.no_scrollback")
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8(ss1Text)))

	base := list.CountTrackedPins()
	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()

	expect(t, !search.history.Valid, "search.history == null")
	expectEqual(t, searchScreenStateHistory, search.state)
	expectEqual(t, base, list.CountTrackedPins())

	// The rows shifted instead of scrolling: the screen is Fizz / Bang.
	expectEqual(t, uint(1), search.MatchesLen())
	ss1ExpectRows(t, &search, [][3]uint{{0, 0, 3}})
}

// Zig: test "term51: ScreenSearch.init: needle not found"
func TestSs1_term51_ScreenSearch_init_needle_not_found(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8(ss1Text)))

	search, err := SearchScreenInit(term.screens.active, []uint8("nope"))
	zigTry(t, err)
	defer search.Deinit()
	expectEqual(t, searchScreenStateHistory, search.state)
	expectEqual(t, uint(0), search.MatchesLen())
	zigTry(t, search.SearchAll())
	expectEqual(t, searchScreenStateComplete, search.state)
	expectEqual(t, uint(0), search.MatchesLen())
	expect(t, !search.SelectedMatch().Valid, "search.selectedMatch() == null")
}

// Zig: test "term51: ScreenSearch.init: needle longer than the content is not found"
func TestSs1_term51_ScreenSearch_init_needle_longer_than_the_content_is_not_found(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz")))

	search, err := SearchScreenInit(term.screens.active, []uint8(strings.Repeat("Fizz", 40)))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expectEqual(t, uint(0), search.MatchesLen())
}

// Zig: test "term51: ScreenSearch.init: one match"
func TestSs1_term51_ScreenSearch_init_one_match(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8(ss1Text)))

	search, err := SearchScreenInit(term.screens.active, []uint8("Bang"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expectEqual(t, uint(1), search.MatchesLen())
	ss1ExpectRows(t, &search, [][3]uint{{0, 3, 3}})
}

// Zig: test "term51: ScreenSearch.init: search is case insensitive and keeps the needle case"
func TestSs1_term51_ScreenSearch_init_search_is_case_insensitive_and_keeps_the_needle_case(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz\r\nfIZZ\r\nBuzz\r\nFIZZ")))

	search, err := SearchScreenInit(term.screens.active, []uint8("fIzZ"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expectEqualStrings(t, "fIzZ", string(search.Needle()))
	ss1ExpectRows(t, &search, [][3]uint{{0, 3, 3}, {0, 1, 3}, {0, 0, 3}})
}

// Zig: test "term51: ScreenSearch.init: single character needle finds every cell"
func TestSs1_term51_ScreenSearch_init_single_character_needle_finds_every_cell(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8(ss1Text)))

	search, err := SearchScreenInit(term.screens.active, []uint8("z"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	// Newest first: each match is a single cell.
	ss1ExpectRows(t, &search, [][3]uint{
		{3, 2, 3}, {2, 2, 2},
		{3, 1, 3}, {2, 1, 2},
		{3, 0, 3}, {2, 0, 2},
	})
}

// Zig: test "term51: ScreenSearch.init: soft wrapped match is one chunk across two rows"
func TestSs1_term51_ScreenSearch_init_soft_wrapped_match_is_one_chunk_across_two_rows(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	// Row 0: "FizzBuzzFi", row 1 (wrapped): "zz".
	zigTry(t, s.NextSlice([]uint8("FizzBuzzFizz")))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	list := &term.screens.active.pages
	ms, err := search.Matches()
	zigTry(t, err)
	// Zig-only: defer alloc.free(ms);
	expectEqual(t, 2, len(ms))

	// Newest first: the wrapped match.
	ss1ExpectAt(t, list, ms[0], 8, 0, 1, 1)
	expectEqual(t, 1, len(ms[0].chunks))
	expectEqual(t, SizeCellCountInt(8), ms[0].top_x)
	expectEqual(t, SizeCellCountInt(1), ms[0].bot_x)
	expectEqual(t, SizeCellCountInt(0), ms[0].chunks[0].start)
	expectEqual(t, SizeCellCountInt(2), ms[0].chunks[0].end)

	ss1ExpectAt(t, list, ms[1], 0, 0, 3, 0)
	expectEqual(t, 1, len(ms[1].chunks))
	expectEqual(t, SizeCellCountInt(0), ms[1].top_x)
	expectEqual(t, SizeCellCountInt(3), ms[1].bot_x)
	expectEqual(t, SizeCellCountInt(0), ms[1].chunks[0].start)
	expectEqual(t, SizeCellCountInt(1), ms[1].chunks[0].end)
}

// Zig: test "term51: ScreenSearch.init: records node serial in chunks"
func TestSs1_term51_ScreenSearch_init_records_node_serial_in_chunks(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz")))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	hl := search.active_results[0]
	first := ss1NonNil(t, term.screens.active.pages.pages.first, "t.screens.active.pages.pages.first.?")
	expect(t, hl.chunks[0].node == first, "hl.chunks.items(.node)[0] == t.screens.active.pages.pages.first.?")
	expectEqual(t, first.serial, hl.chunks[0].serial)
}

// Zig: test "term51: ScreenSearch.deinit: releases tracked pins and results after full search"
func TestSs1_term51_ScreenSearch_deinit_releases_tracked_pins_and_results_after_full_search(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, ss1Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss1ThreePages(t, &s, list, "Fizz", "fizz", "FIZZ\r\nbuzz")

	base := list.CountTrackedPins()
	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	zigTry(t, search.SearchAll())
	_, err = search.Select(SearchScreenSelectNext)
	zigTry(t, err)
	expectEqual(t, uint(3), search.MatchesLen())
	// history searcher (2) + selection (2)
	expectEqual(t, base+4, list.CountTrackedPins())

	search.Deinit()
	expectEqual(t, base, list.CountTrackedPins())
}

// Zig: test "term51: ScreenSearch.deinit: after a partial feed"
func TestSs1_term51_ScreenSearch_deinit_after_a_partial_feed(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, ss1Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss1ThreePages(t, &s, list, "Fizz", "fizz", "FIZZ\r\nbuzz")

	base := list.CountTrackedPins()
	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	// Make some progress: history window holds node 3 and node 2 but the
	// match in node 2 was not consumed yet.
	zigTry(t, search.Tick())
	zigTry(t, search.Feed())
	expectEqual(t, searchScreenStateHistory, search.state)
	expectEqual(t, 0, len(search.history_results))
	search.Deinit()
	expectEqual(t, base, list.CountTrackedPins())
}

// Zig: test "term51: ScreenSearch.deinit: right after init and with a selection"
func TestSs1_term51_ScreenSearch_deinit_right_after_init_and_with_a_selection(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8(ss1Text)))

	base := list.CountTrackedPins()
	{
		search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
		zigTry(t, err)
		search.Deinit()
		expectEqual(t, base, list.CountTrackedPins())
	}
	{
		search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
		zigTry(t, err)
		_, err = search.Select(SearchScreenSelectNext)
		zigTry(t, err)
		expectEqual(t, base+4, list.CountTrackedPins())
		search.Deinit()
		expectEqual(t, base, list.CountTrackedPins())
	}
}

// Zig: test "term51: ScreenSearch.SelectedMatch.deinit: untracks both pins"
func TestSs1_term51_ScreenSearch_SelectedMatch_deinit_untracks_both_pins(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8(ss1Text)))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())

	base := list.CountTrackedPins()
	untracked := search.active_results[0].Untracked()
	tracked, err := untracked.Track(term.screens.active)
	zigTry(t, err)
	m := SearchScreenSelectedMatch{
		idx:       7,
		highlight: tracked,
	}
	expectEqual(t, base+2, list.CountTrackedPins())
	expect(t, m.highlight.start != m.highlight.end, "m.highlight.start != m.highlight.end")
	m.Deinit(term.screens.active)
	expectEqual(t, base, list.CountTrackedPins())
}

// Zig: test "term51: ScreenSearch.HistorySearch.deinit: untracks the searcher pin and the start pin"
func TestSs1_term51_ScreenSearch_HistorySearch_deinit_untracks_the_searcher_pin_and_the_start_pin(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8(ss1Text)))

	base := list.CountTrackedPins()
	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	expectEqual(t, base+2, list.CountTrackedPins())

	h := zigUnwrap(t, search.history)
	search.history = None[searchScreenHistorySearch]()
	h.Deinit(term.screens.active)
	expectEqual(t, base, list.CountTrackedPins())
}

// Zig: test "term51: ScreenSearch.needle: is an owned copy with the original case"
func TestSs1_term51_ScreenSearch_needle_is_an_owned_copy_with_the_original_case(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()

	buf := [8]uint8{'F', 'i', 'Z', 'z', 0, 0, 0, 0}
	search, err := SearchScreenInit(term.screens.active, buf[0:4])
	zigTry(t, err)
	defer search.Deinit()
	expectEqualStrings(t, "FiZz", string(search.Needle()))
	expect(t, &search.Needle()[0] != &buf[0], "search.needle().ptr != @as([*]const u8, &buf)")

	// Changing the original does not change the search.
	buf[0] = 'x'
	expectEqualStrings(t, "FiZz", string(search.Needle()))
}

// Zig: test "term51: ScreenSearch.needle: is not reversed and keeps whitespace and non letters"
func TestSs1_term51_ScreenSearch_needle_is_not_reversed_and_keeps_whitespace_and_non_letters(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("a b-c\r\na  b")))

	search, err := SearchScreenInit(term.screens.active, []uint8("a b-"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expectEqualStrings(t, "a b-", string(search.Needle()))
	expectEqual(t, uint(1), search.MatchesLen())
	ss1ExpectRows(t, &search, [][3]uint{{0, 0, 3}})
}

// Zig: test "term51: ScreenSearch.matchesLen: counts active and history results"
func TestSs1_term51_ScreenSearch_matchesLen_counts_active_and_history_results(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, ss1Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss1ThreePages(t, &s, list, "Fizz", "fizz", "FIZZ\r\nbuzz")

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	// Only the active area was searched by init.
	expectEqual(t, uint(1), search.MatchesLen())
	expectEqual(t, 1, len(search.active_results))
	expectEqual(t, 0, len(search.history_results))

	zigTry(t, search.SearchAll())
	expectEqual(t, uint(3), search.MatchesLen())
	expectEqual(t, 1, len(search.active_results))
	expectEqual(t, 2, len(search.history_results))
}

// Zig: test "term51: ScreenSearch.matchesLen: zero when nothing matches"
func TestSs1_term51_ScreenSearch_matchesLen_zero_when_nothing_matches(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expectEqual(t, uint(0), search.MatchesLen())
}

// Zig: test "term51: ScreenSearch.matches: empty list is an empty slice"
func TestSs1_term51_ScreenSearch_matches_empty_list_is_an_empty_slice(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	ms, err := search.Matches()
	zigTry(t, err)
	// Zig-only: defer alloc.free(ms);
	expectEqual(t, 0, len(ms))
}

// Zig: test "term51: ScreenSearch.matches: active reversed then history newest to oldest"
func TestSs1_term51_ScreenSearch_matches_active_reversed_then_history_newest_to_oldest(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, ss1Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	// Page 3 holds two matches (rows 0 and 2) with the active area on its
	// last two rows; page 2 holds two matches on one row; page 1 one match.
	ss1ThreePages(t, &s, list, "Fizz", "Fizz fizz", "FIZZ\r\nbuzz\r\nfiZZ")
	expectEqual(t, uint(3), ss1Nodes(list))

	search, err := SearchScreenInit(term.screens.active, []uint8("fizz"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expectEqual(t, 2, len(search.active_results))
	expectEqual(t, 3, len(search.history_results))

	b2 := ss1RowBase(list, 1)
	b3 := ss1RowBase(list, 2)
	ss1ExpectRows(t, &search, [][3]uint{
		{0, b3 + 2, 3}, // active, newest
		{0, b3, 3},
		{5, b2, 8}, // history: page 2 right to left
		{0, b2, 3},
		{0, 0, 3}, // page 1
	})
}

// Zig: test "term51: ScreenSearch.matches: shares chunk memory with the stored results"
func TestSs1_term51_ScreenSearch_matches_shares_chunk_memory_with_the_stored_results(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, ss1Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss1ThreePages(t, &s, list, "Fizz", "fizz", "FIZZ\r\nbuzz")

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	ms, err := search.Matches()
	zigTry(t, err)
	// Zig-only: defer alloc.free(ms);
	expectEqual(t, 3, len(ms))
	// The returned slice is owned by the caller but the highlights are
	// shallow copies: the caller must not deinit them.
	expect(t, &ms[0].chunks[0] == &search.active_results[0].chunks[0], "ms[0].chunks.bytes == search.active_results.items[0].chunks.bytes")
	expect(t, &ms[1].chunks[0] == &search.history_results[0].chunks[0], "ms[1].chunks.bytes == search.history_results.items[0].chunks.bytes")
	expect(t, &ms[2].chunks[0] == &search.history_results[1].chunks[0], "ms[2].chunks.bytes == search.history_results.items[1].chunks.bytes")
}

// Zig: test "term51: ScreenSearch.matches: spanning match flattened across two pages"
func TestSs1_term51_ScreenSearch_matches_spanning_match_flattened_across_two_pages(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, ss1Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()

	// Last row of page 1 ends with "Te", it wraps to "st" in page 2.
	pageCap := ss1Cap(list)
	for range pageCap - 1 {
		zigTry(t, s.NextSlice([]uint8("\r\n")))
	}
	zigTry(t, s.NextSlice([]uint8(ss1Xs+"Test")))
	ss1GrowTo(t, &s, list, 3)
	zigTry(t, s.NextSlice([]uint8("ab\r\ncd")))
	expectEqual(t, uint(3), ss1Nodes(list))

	search, err := SearchScreenInit(term.screens.active, []uint8("Test"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expectEqual(t, 0, len(search.active_results))
	expectEqual(t, 1, len(search.history_results))

	ms, err := search.Matches()
	zigTry(t, err)
	// Zig-only: defer alloc.free(ms);
	expectEqual(t, 1, len(ms))
	m := ms[0]
	n1 := ss1NonNil(t, list.pages.first, "list.pages.first.?")
	n2 := ss1NonNil(t, n1.next, "n1.next.?")
	expectEqual(t, 2, len(m.chunks))
	expect(t, m.chunks[0].node == n1, "m.chunks.items(.node)[0] == n1")
	expect(t, m.chunks[1].node == n2, "m.chunks.items(.node)[1] == n2")
	// Surprising: the history search is a reverse search and reverses
	// nodes, starts and ends but not the serials, so the serials are in
	// the order the pages were searched (newest page first) and do not
	// line up with the nodes.
	expect(t, n1.serial != n2.serial, "n1.serial != n2.serial")
	expectEqual(t, n2.serial, m.chunks[0].serial)
	expectEqual(t, n1.serial, m.chunks[1].serial)
	// First chunk: the last row of page 1; second: the first row of page 2.
	expectEqual(t, SizeCellCountInt(pageCap-1), m.chunks[0].start)
	expectEqual(t, SizeCellCountInt(pageCap), m.chunks[0].end)
	expectEqual(t, SizeCellCountInt(0), m.chunks[1].start)
	expectEqual(t, SizeCellCountInt(1), m.chunks[1].end)
	expectEqual(t, SizeCellCountInt(ss1Cols-2), m.top_x)
	expectEqual(t, SizeCellCountInt(1), m.bot_x)
	ss1ExpectAt(t, list, m, ss1Cols-2, pageCap-1, 1, pageCap)
}

// Zig: test "term51: ScreenSearch.tick: single page goes history, feed required, complete"
func TestSs1_term51_ScreenSearch_tick_single_page_goes_history_feed_required_complete(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8(ss1Text)))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	expectEqual(t, searchScreenStateHistory, search.state)

	// History: the only node is the start node so its matches are skipped.
	zigTry(t, search.Tick())
	expectEqual(t, searchScreenStateHistoryFeed, search.state)
	expectEqual(t, 0, len(search.history_results))
	expectEqual(t, 2, len(search.active_results))

	// Feed required, and asking again changes nothing.
	ss1ExpectError(t, ErrFeedRequired, search.Tick())
	ss1ExpectError(t, ErrFeedRequired, search.Tick())
	expectEqual(t, searchScreenStateHistoryFeed, search.state)

	// Nothing to feed on a single page: complete.
	zigTry(t, search.Feed())
	expectEqual(t, searchScreenStateComplete, search.state)
	ss1ExpectError(t, ErrSearchComplete, search.Tick())
	ss1ExpectError(t, ErrSearchComplete, search.Tick())
	expectEqual(t, searchScreenStateComplete, search.state)
	expectEqual(t, uint(2), search.MatchesLen())
}

// Zig: test "term51: ScreenSearch.tick: no scrollback completes without a feed"
func TestSs1_term51_ScreenSearch_tick_no_scrollback_completes_without_a_feed(t *testing.T) {
	testStdCapacity(t)
	term := ss1InitScrollback(t, 10, 2, 0)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz")))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	expectEqual(t, searchScreenStateHistory, search.state)
	// No history searcher: tick goes straight to complete, no error.
	zigTry(t, search.Tick())
	expectEqual(t, searchScreenStateComplete, search.state)
	ss1ExpectError(t, ErrSearchComplete, search.Tick())
	expectEqual(t, uint(1), search.MatchesLen())
}

// Zig: test "term51: ScreenSearch.tick: active state moves to history without new results"
func TestSs1_term51_ScreenSearch_tick_active_state_moves_to_history_without_new_results(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8(ss1Text)))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	expectEqual(t, 2, len(search.active_results))

	// tickActive never clears but the window was already consumed by init,
	// so running it again finds nothing and only changes the state.
	search.state = searchScreenStateActive
	zigTry(t, search.Tick())
	expectEqual(t, searchScreenStateHistory, search.state)
	expectEqual(t, 2, len(search.active_results))
	expectEqual(t, uint(2), search.MatchesLen())
}

// Zig: test "term51: ScreenSearch.tick: history_feed and complete do not touch the results"
func TestSs1_term51_ScreenSearch_tick_history_feed_and_complete_do_not_touch_the_results(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8(ss1Text)))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()

	search.state = searchScreenStateHistoryFeed
	ss1ExpectError(t, ErrFeedRequired, search.Tick())
	expectEqual(t, searchScreenStateHistoryFeed, search.state)
	search.state = searchScreenStateComplete
	ss1ExpectError(t, ErrSearchComplete, search.Tick())
	expectEqual(t, searchScreenStateComplete, search.state)
	expectEqual(t, uint(2), search.MatchesLen())
}

// Zig: test "term51: ScreenSearch.tickActive: appends all active matches in forward order"
func TestSs1_term51_ScreenSearch_tickActive_appends_all_active_matches_in_forward_order(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8(ss1Text)))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	// The window was consumed by init so a second run finds nothing new
	// (the window is only refilled by a reload); results are in forward order.
	ss1ExpectAt(t, list, search.active_results[0], 0, 0, 3, 0)
	ss1ExpectAt(t, list, search.active_results[1], 0, 2, 3, 2)
	zigTry(t, search.tickActive())
	expectEqual(t, 2, len(search.active_results))
	expectEqual(t, searchScreenStateHistory, search.state)
}

// Zig: test "term51: ScreenSearch.tickActive: always sets the state to history"
func TestSs1_term51_ScreenSearch_tickActive_always_sets_the_state_to_history(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8(ss1Text)))

	search, err := SearchScreenInit(term.screens.active, []uint8("nomatch"))
	zigTry(t, err)
	defer search.Deinit()
	for _, st := range []searchScreenState{
		searchScreenStateActive,
		searchScreenStateHistory,
		searchScreenStateHistoryFeed,
		searchScreenStateComplete,
	} {
		search.state = st
		zigTry(t, search.tickActive())
		expectEqual(t, searchScreenStateHistory, search.state)
	}
	expectEqual(t, uint(0), search.MatchesLen())
}

// Zig: test "term51: ScreenSearch.tickHistory: no history searcher completes"
func TestSs1_term51_ScreenSearch_tickHistory_no_history_searcher_completes(t *testing.T) {
	testStdCapacity(t)
	term := ss1InitScrollback(t, 10, 2, 0)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz")))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	expect(t, !search.history.Valid, "search.history == null")
	zigTry(t, search.tickHistory())
	expectEqual(t, searchScreenStateComplete, search.state)
	expectEqual(t, 0, len(search.history_results))
}

// Zig: test "term51: ScreenSearch.tickHistory: skips matches in the start node and asks for a feed"
func TestSs1_term51_ScreenSearch_tickHistory_skips_matches_in_the_start_node_and_asks_for_a_feed(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8(ss1Text)))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.tickHistory())
	expectEqual(t, searchScreenStateHistoryFeed, search.state)
	expectEqual(t, 0, len(search.history_results))

	// Again: nothing more in the window.
	zigTry(t, search.tickHistory())
	expectEqual(t, searchScreenStateHistoryFeed, search.state)
	expectEqual(t, 0, len(search.history_results))
}

// Zig: test "term51: ScreenSearch.tickHistory: appends fed matches newest first and sets history_feed"
func TestSs1_term51_ScreenSearch_tickHistory_appends_fed_matches_newest_first_and_sets_history_feed(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, ss1Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss1ThreePages(t, &s, list, "Fizz", "Fizz fizz", "FIZZ\r\nbuzz")

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.tickHistory())
	expectEqual(t, 0, len(search.history_results))
	zigTry(t, search.Feed())
	// Feeding itself does not search.
	expectEqual(t, 0, len(search.history_results))

	// Force a state other than history to show tickHistory sets it.
	search.state = searchScreenStateComplete
	zigTry(t, search.tickHistory())
	expectEqual(t, searchScreenStateHistoryFeed, search.state)
	b2 := ss1RowBase(list, 1)
	expectEqual(t, 2, len(search.history_results))
	ss1ExpectAt(t, list, search.history_results[0], 5, b2, 8, b2)
	ss1ExpectAt(t, list, search.history_results[1], 0, b2, 3, b2)
}

// Zig: test "term51: ScreenSearch.feed: single page completes"
func TestSs1_term51_ScreenSearch_feed_single_page_completes(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8(ss1Text)))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	// Feeding is allowed in any state: nothing to feed means complete.
	expectEqual(t, searchScreenStateHistory, search.state)
	zigTry(t, search.Feed())
	expectEqual(t, searchScreenStateComplete, search.state)
	// And stays complete.
	zigTry(t, search.Feed())
	expectEqual(t, searchScreenStateComplete, search.state)
	expectEqual(t, uint(2), search.MatchesLen())
}

// Zig: test "term51: ScreenSearch.feed: no scrollback completes"
func TestSs1_term51_ScreenSearch_feed_no_scrollback_completes(t *testing.T) {
	testStdCapacity(t)
	term := ss1InitScrollback(t, 10, 2, 0)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz")))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	search.state = searchScreenStateHistoryFeed
	zigTry(t, search.Feed())
	expectEqual(t, searchScreenStateComplete, search.state)
}

// Zig: test "term51: ScreenSearch.feed: history progress through three pages"
func TestSs1_term51_ScreenSearch_feed_history_progress_through_three_pages(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, ss1Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss1ThreePages(t, &s, list, "Fizz", "fIzZ", "FIZZ\r\nbuzz")
	n3 := ss1NonNil(t, list.pages.last, "list.pages.last.?")
	n2 := ss1NonNil(t, n3.prev, "n3.prev.?")
	n1 := ss1NonNil(t, n2.prev, "n2.prev.?")
	b2 := ss1RowBase(list, 1)
	b3 := ss1RowBase(list, 2)

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	expectEqual(t, searchScreenStateHistory, search.state)
	expectEqual(t, 1, len(search.active_results))
	expect(t, zigUnwrap(t, search.history).start_pin.node == n3, "search.history.?.start_pin.node == n3")
	expect(t, zigUnwrap(t, search.history).searcher.pin.node == n3, "search.history.?.searcher.pin.node == n3")

	// Window holds only the start node, which is skipped.
	zigTry(t, search.Tick())
	expectEqual(t, searchScreenStateHistoryFeed, search.state)
	expectEqual(t, 0, len(search.history_results))
	ss1ExpectError(t, ErrFeedRequired, search.Tick())

	// Feed the second page: back to history state.
	zigTry(t, search.Feed())
	expectEqual(t, searchScreenStateHistory, search.state)
	expect(t, zigUnwrap(t, search.history).searcher.pin.node == n2, "search.history.?.searcher.pin.node == n2")
	expect(t, zigUnwrap(t, search.history).start_pin.node == n3, "search.history.?.start_pin.node == n3")
	expectEqual(t, 0, len(search.history_results))
	zigTry(t, search.Tick())
	expectEqual(t, searchScreenStateHistoryFeed, search.state)
	expectEqual(t, 1, len(search.history_results))
	ss1ExpectAt(t, list, search.history_results[0], 0, b2, 3, b2)

	// Feed the first page.
	zigTry(t, search.Feed())
	expectEqual(t, searchScreenStateHistory, search.state)
	expect(t, zigUnwrap(t, search.history).searcher.pin.node == n1, "search.history.?.searcher.pin.node == n1")
	zigTry(t, search.Tick())
	expectEqual(t, searchScreenStateHistoryFeed, search.state)
	expectEqual(t, 2, len(search.history_results))
	ss1ExpectAt(t, list, search.history_results[1], 0, 0, 3, 0)

	// Nothing left: complete.
	ss1ExpectError(t, ErrFeedRequired, search.Tick())
	zigTry(t, search.Feed())
	expectEqual(t, searchScreenStateComplete, search.state)
	ss1ExpectError(t, ErrSearchComplete, search.Tick())

	ss1ExpectRows(t, &search, [][3]uint{
		{0, b3, 3},
		{0, b2, 3},
		{0, 0, 3},
	})
}

// Zig: test "term51: ScreenSearch.feed: one feed adds several pages until the needle length is covered"
func TestSs1_term51_ScreenSearch_feed_one_feed_adds_several_pages_until_the_needle_length_is_covered(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, ss1Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	// Page 2 is empty so it contributes less than the needle length.
	ss1ThreePages(t, &s, list, "Fizz", "", "FIZZ\r\nbuzz")
	n1 := ss1NonNil(t, list.pages.first, "list.pages.first.?")

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.Tick())
	expectEqual(t, searchScreenStateHistoryFeed, search.state)
	zigTry(t, search.Feed())
	expectEqual(t, searchScreenStateHistory, search.state)
	// Both earlier pages were fed by one call.
	expect(t, zigUnwrap(t, search.history).searcher.pin.node == n1, "search.history.?.searcher.pin.node == n1")
	zigTry(t, search.Tick())
	expectEqual(t, 1, len(search.history_results))
	ss1ExpectAt(t, list, search.history_results[0], 0, 0, 3, 0)
	zigTry(t, search.Feed())
	expectEqual(t, searchScreenStateComplete, search.state)
}

// Zig: test "term51: ScreenSearch.feed: feeding in active or history state keeps the state"
func TestSs1_term51_ScreenSearch_feed_feeding_in_active_or_history_state_keeps_the_state(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, ss1Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss1ThreePages(t, &s, list, "Fizz", "fIzZ", "FIZZ\r\nbuzz")

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	search.state = searchScreenStateActive
	zigTry(t, search.Feed())
	expectEqual(t, searchScreenStateActive, search.state)
	expect(t, zigUnwrap(t, search.history).searcher.pin.node == ss1NonNil(t, list.pages.first.next, "list.pages.first.?.next.?"), "search.history.?.searcher.pin.node == list.pages.first.?.next.?")

	search.state = searchScreenStateHistory
	zigTry(t, search.Feed())
	expectEqual(t, searchScreenStateHistory, search.state)
	expect(t, zigUnwrap(t, search.history).searcher.pin.node == ss1NonNil(t, list.pages.first, "list.pages.first.?"), "search.history.?.searcher.pin.node == list.pages.first.?")

	// Out of data while not complete: goes complete.
	zigTry(t, search.Feed())
	expectEqual(t, searchScreenStateComplete, search.state)
}

// Zig: test "term51: ScreenSearch.feed: unchanged size keeps results and selection"
func TestSs1_term51_ScreenSearch_feed_unchanged_size_keeps_results_and_selection(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8(ss1Text)))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	_, err = search.Select(SearchScreenSelectNext)
	zigTry(t, err)
	// Resizing to the same size is a no-op in the terminal.
	zigTry(t, term.Resize(10, 2))
	zigTry(t, search.Feed())
	expectEqual(t, uint(0), zigUnwrap(t, search.selected).idx)
	expectEqual(t, uint(2), search.MatchesLen())
	expectEqual(t, searchScreenStateComplete, search.state)
}

// Zig: test "term51: ScreenSearch.feed: wider screen restarts the whole search"
func TestSs1_term51_ScreenSearch_feed_wider_screen_restarts_the_whole_search(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 3)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz\r\nBuzz\r\nFizz")))

	base := list.CountTrackedPins()
	search, err := SearchScreenInit(term.screens.active, []uint8("FiZz"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	_, err = search.Select(SearchScreenSelectNext)
	zigTry(t, err)
	expectEqual(t, uint(0), zigUnwrap(t, search.selected).idx)
	expectEqual(t, base+4, list.CountTrackedPins())

	zigTry(t, term.Resize(20, 3))
	// Nothing notices before the feed.
	expectEqual(t, SizeCellCountInt(10), search.cols)
	expectEqual(t, SizeCellCountInt(3), search.rows)

	zigTry(t, search.Feed())
	expectEqual(t, SizeCellCountInt(20), search.cols)
	expectEqual(t, SizeCellCountInt(3), search.rows)
	// Selection is dropped, results are recomputed, same needle case.
	expect(t, !search.selected.Valid, "search.selected == null")
	expectEqualStrings(t, "FiZz", string(search.Needle()))
	expectEqual(t, uint(2), search.MatchesLen())
	// The restart ran init (active searched) and then the feed finished it.
	expectEqual(t, searchScreenStateComplete, search.state)
	expectEqual(t, base+2, list.CountTrackedPins())
	ss1ExpectRows(t, &search, [][3]uint{{0, 2, 3}, {0, 0, 3}})
}

// Zig: test "term51: ScreenSearch.feed: row count change restarts the search"
func TestSs1_term51_ScreenSearch_feed_row_count_change_restarts_the_search(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 3)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz\r\nBuzz\r\nFizz")))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	_, err = search.Select(SearchScreenSelectPrev)
	zigTry(t, err)
	expectEqual(t, uint(1), zigUnwrap(t, search.selected).idx)

	zigTry(t, term.Resize(10, 5))
	zigTry(t, search.Feed())
	expectEqual(t, SizeCellCountInt(10), search.cols)
	expectEqual(t, SizeCellCountInt(5), search.rows)
	expect(t, !search.selected.Valid, "search.selected == null")
	expectEqual(t, uint(2), search.MatchesLen())
	expectEqual(t, searchScreenStateComplete, search.state)
}

// Zig: test "term51: ScreenSearch.feed: narrower screen reflows and finds the wrapped match"
func TestSs1_term51_ScreenSearch_feed_narrower_screen_reflows_and_finds_the_wrapped_match(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 3)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz")))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	expectEqual(t, uint(1), search.MatchesLen())

	// "Fiz" / "z"
	zigTry(t, term.Resize(3, 3))
	zigTry(t, search.Feed())
	expectEqual(t, SizeCellCountInt(3), search.cols)
	expectEqual(t, uint(1), search.MatchesLen())
	m := search.active_results[0]
	ss1ExpectAt(t, list, m, 0, 0, 0, 1)
	expectEqual(t, 1, len(m.chunks))
	expectEqual(t, SizeCellCountInt(0), m.chunks[0].start)
	expectEqual(t, SizeCellCountInt(2), m.chunks[0].end)
	expectEqual(t, SizeCellCountInt(0), m.top_x)
	expectEqual(t, SizeCellCountInt(0), m.bot_x)
}

// Zig: test "term51: ScreenSearch.searchAll: runs to complete and is repeatable"
func TestSs1_term51_ScreenSearch_searchAll_runs_to_complete_and_is_repeatable(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, ss1Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss1ThreePages(t, &s, list, "Fizz", "fIzZ", "FIZZ\r\nbuzz")

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expectEqual(t, searchScreenStateComplete, search.state)
	expectEqual(t, uint(3), search.MatchesLen())
	zigTry(t, search.SearchAll())
	expectEqual(t, searchScreenStateComplete, search.state)
	expectEqual(t, uint(3), search.MatchesLen())
}

// Zig: test "term51: ScreenSearch.searchAll: no scrollback and no matches"
func TestSs1_term51_ScreenSearch_searchAll_no_scrollback_and_no_matches(t *testing.T) {
	testStdCapacity(t)
	term := ss1InitScrollback(t, 10, 2, 0)
	defer term.Deinit()
	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	expectEqual(t, searchScreenStateComplete, search.state)
	expectEqual(t, uint(0), search.MatchesLen())
}

// Zig: test "term51: ScreenSearch.searchAll: continues from a partially searched state"
func TestSs1_term51_ScreenSearch_searchAll_continues_from_a_partially_searched_state(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, ss1Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss1ThreePages(t, &s, list, "Fizz", "fIzZ", "FIZZ\r\nbuzz")

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.Tick())
	zigTry(t, search.Feed())
	expectEqual(t, searchScreenStateHistory, search.state)
	zigTry(t, search.SearchAll())
	expectEqual(t, searchScreenStateComplete, search.state)
	expectEqual(t, uint(3), search.MatchesLen())
}

// Zig: test "term51: ScreenSearch.searchAll: starting from the active state completes without duplicates"
func TestSs1_term51_ScreenSearch_searchAll_starting_from_the_active_state_completes_without_duplicates(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8(ss1Text)))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	search.state = searchScreenStateActive
	zigTry(t, search.SearchAll())
	expectEqual(t, searchScreenStateComplete, search.state)
	expectEqual(t, 2, len(search.active_results))
}

// Zig: test "term51: ScreenSearch.searchAll: complete search does not see later screen changes"
func TestSs1_term51_ScreenSearch_searchAll_complete_search_does_not_see_later_screen_changes(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 2)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8(ss1Text)))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	zigTry(t, search.SearchAll())
	zigTry(t, s.NextSlice([]uint8("\r\nFizz")))
	// Complete: searchAll returns without touching the screen.
	zigTry(t, search.SearchAll())
	expectEqual(t, uint(2), search.MatchesLen())
	// Until the caller reloads the active area.
	zigTry(t, search.ReloadActive())
	expectEqual(t, uint(3), search.MatchesLen())
	expectEqual(t, searchScreenStateComplete, search.state)
}

// Zig: test "term51: ScreenSearch.reloadActive: active area is searched again"
func TestSs1_term51_ScreenSearch_reloadActive_active_area_is_searched_again(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 5)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz\r\nBuzz\r\nFizz")))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	expectEqual(t, 2, len(search.active_results))

	zigTry(t, s.NextSlice([]uint8("\r\nFizz")))
	zigTry(t, search.ReloadActive())
	expectEqual(t, 3, len(search.active_results))
	ss1ExpectRows(t, &search, [][3]uint{{0, 3, 3}, {0, 2, 3}, {0, 0, 3}})

	// Removing content removes the results, reloading twice is stable.
	zigTry(t, s.NextSlice([]uint8("\x1b[2J\x1b[H")))
	zigTry(t, search.ReloadActive())
	zigTry(t, search.ReloadActive())
	expectEqual(t, uint(0), search.MatchesLen())
	expectEqual(t, 0, len(search.history_results))
}

// Zig: test "term51: ScreenSearch.reloadActive: state is kept except active becomes history"
func TestSs1_term51_ScreenSearch_reloadActive_state_is_kept_except_active_becomes_history(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 5)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz\r\nBuzz\r\nFizz")))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()

	states := []searchScreenState{
		searchScreenStateActive,
		searchScreenStateHistory,
		searchScreenStateHistoryFeed,
		searchScreenStateComplete,
	}
	expected := []searchScreenState{
		searchScreenStateHistory,
		searchScreenStateHistory,
		searchScreenStateHistoryFeed,
		searchScreenStateComplete,
	}
	for i, st := range states {
		exp := expected[i]
		search.state = st
		zigTry(t, search.ReloadActive())
		expectEqual(t, exp, search.state)
		// Results are reset each time, never duplicated.
		expectEqual(t, 2, len(search.active_results))
	}
}

// Zig: test "term51: ScreenSearch.reloadActive: selection in the active area follows its match"
func TestSs1_term51_ScreenSearch_reloadActive_selection_in_the_active_area_follows_its_match(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 6)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz\r\nBuzz\r\nFizz")))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	_, err = search.Select(SearchScreenSelectNext)
	zigTry(t, err)
	ss1ExpectSel(t, &search, 0, 0, 2, 3, 2)

	// New newest matches push the index up but keep the same match.
	zigTry(t, s.NextSlice([]uint8("\r\nFizz")))
	zigTry(t, search.ReloadActive())
	expectEqual(t, uint(3), search.MatchesLen())
	ss1ExpectSel(t, &search, 1, 0, 2, 3, 2)

	zigTry(t, s.NextSlice([]uint8("\r\nFizz")))
	zigTry(t, search.ReloadActive())
	expectEqual(t, uint(4), search.MatchesLen())
	ss1ExpectSel(t, &search, 2, 0, 2, 3, 2)

	// Selecting the oldest then adding another keeps the oldest.
	_, err = search.Select(SearchScreenSelectNext)
	zigTry(t, err)
	ss1ExpectSel(t, &search, 3, 0, 0, 3, 0)
	zigTry(t, s.NextSlice([]uint8("\r\nFizz")))
	zigTry(t, search.ReloadActive())
	ss1ExpectSel(t, &search, 4, 0, 0, 3, 0)
}

// Zig: test "term51: ScreenSearch.reloadActive: selected match gone selects the newest"
func TestSs1_term51_ScreenSearch_reloadActive_selected_match_gone_selects_the_newest(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, 10, 5)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz\r\nBuzz\r\nFizz")))

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	defer search.Deinit()
	_, err = search.Select(SearchScreenSelectNext)
	zigTry(t, err)
	_, err = search.Select(SearchScreenSelectNext)
	zigTry(t, err)
	ss1ExpectSel(t, &search, 1, 0, 0, 3, 0)

	// Only one match left, on another row.
	zigTry(t, s.NextSlice([]uint8("\x1b[2J\x1b[H\r\nFizz")))
	zigTry(t, search.ReloadActive())
	expectEqual(t, uint(1), search.MatchesLen())
	ss1ExpectSel(t, &search, 0, 0, 1, 3, 1)
}

// Go-only: panic test: ScreenSearch.feed: a complete state while the history searcher can still be fed reaches unreachable
func TestSs1_panic_feed_complete_state_with_more_history_to_feed(t *testing.T) {
	testStdCapacity(t)
	term := ss1New(t, ss1Cols, 2)
	defer term.Deinit()
	list := &term.screens.active.pages
	s := term.VtStream()
	defer s.Deinit()
	ss1ThreePages(t, &s, list, "Fizz", "fIzZ", "FIZZ\r\nbuzz")

	search, err := SearchScreenInit(term.screens.active, []uint8("Fizz"))
	zigTry(t, err)
	// No search.Deinit(): the panic leaves the search half way through feed.

	// The history searcher still has two pages to feed, so history.feed
	// returns true, and in the complete state the switch reaches
	// `.complete => unreachable` ("the feed call above should always return
	// false and we can't reach this").
	search.state = searchScreenStateComplete
	ss1ExpectPanic(t, func() {
		_ = search.Feed()
	})
}
