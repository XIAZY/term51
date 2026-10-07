// Tests translated one for one from src/terminal/search.zig (Ghostty v1.3.1
// plus the term51 additional tests), and the shared helpers for the tests of
// src/terminal/search/active.zig, viewport.zig and pagelist.zig
// (search_active_test.go, search_viewport_test.go, search_pagelist_test.go).
// Each Go test carries the exact Zig title in a "// Zig: test" comment.
// Helpers are prefixed sa.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualStrings keep Zig's names and Zig's
//     (expected, actual) argument order. `try` on a call that returns an
//     error is zigTry(t, err); `x.?` is zigUnwrap or a nil check.
//   - The three files define the same term51 helpers under the name
//     term51AL...; Go has one package, so they are shared here under sa:
//     term51ALChunk is saChunk (its `serial: ?u64 = null`, used by
//     pagelist.zig only, is an Optional that the others leave unset),
//     term51ALExpectHighlight is saExpectHighlight, term51ALTerminal is
//     saTerminal, term51ALMark is saMark, term51ALFill is saFill,
//     term51ALWrite is saWrite, term51ALNewlines is saNewlines,
//     term51ALCap is saCap, term51ALScreen is saScreen,
//     term51ALWrapLast is saWrapLast, term51ALMetaNode is saMetaNode,
//     term51ALHasPin is saHasPin.
//   - `Terminal` values are the variable `term` (the testing parameter must
//     be `t`, as scripts/check-test-mapping.py requires); a terminal is a
//     *Terminal returned by saTerminal.
//   - `std.MultiArrayList(Chunk)` is `[]HighlightFlattenedChunk`:
//     `h.chunks.items(.node)[i]` is `h.chunks[i].node`.
//   - A raw-pointer comparison of two buffers (`a.ptr != b.ptr`) is the
//     comparison of the addresses of their first elements, `&a[0] != &b[0]`.
//   - Zig's `alloc` arguments are dropped; the FailingAllocator tests and the
//     allocator-leak checks are not translated (listed in
//     scripts/not-translated.txt, or `// Zig-only:` lines).
//   - Every test calls testStdCapacity first (Zig's unit tests run with
//     grapheme_bytes 512); the page capacity decides rows per page.
//   - Go-only panic tests use saExpectPanic.

package terminal

import (
	"reflect"
	"testing"
)

// saExpectPanic runs f, which must panic. A panic with the wireframe's
// "not implemented" value does not count: it only means the function under
// test does not exist yet. The setup is done outside f.
func saExpectPanic(t *testing.T, f func()) {
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

// saChunk is term51ALChunk: the expected chunk of a flattened highlight.
// `serial` is Zig's `serial: ?u64 = null` (pagelist.zig); unset means the
// node's own serial.
type saChunk struct {
	node   *PageListNode
	start  SizeCellCountInt
	end    SizeCellCountInt
	serial Optional[uint64]
}

// saExpectHighlight is term51ALExpectHighlight. Asserts every field of a
// flattened highlight: the x bounds, the number of chunks and for each chunk
// its node, serial, start row and end row (exclusive).
func saExpectHighlight(
	t *testing.T,
	h HighlightFlattened,
	top_x SizeCellCountInt,
	bot_x SizeCellCountInt,
	want []saChunk,
) {
	t.Helper()
	expectEqual(t, top_x, h.TopX)
	expectEqual(t, bot_x, h.BotX)
	expectEqual(t, uint(len(want)), uint(len(h.Chunks)))
	for i, c := range want {
		expectEqual(t, c.node, h.Chunks[i].Node)
		wantSerial := c.node.serial
		if c.serial.Valid {
			wantSerial = c.serial.Value
		}
		expectEqual(t, wantSerial, h.Chunks[i].Serial)
		expectEqual(t, c.start, h.Chunks[i].Start)
		expectEqual(t, c.end, h.Chunks[i].End)
	}
}

// saTerminal is term51ALTerminal: a terminal with a 100 MiB scrollback
// limit. The terminal is returned by pointer (the methods take a pointer
// receiver); the caller defers Deinit.
func saTerminal(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) *Terminal {
	t.Helper()
	opts := terminalOptionsDefault(cols, rows)
	opts.MaxScrollback = 100 * 1024 * 1024
	term, err := TerminalInit(opts)
	zigTry(t, err)
	return &term
}

// saMark is term51ALMark.
type saMark struct {
	y    uint
	text string
}

// saFill is term51ALFill. Writes `total` lines (screen rows 0..total-1)
// separated by CRLF and ends with the cursor on the last line. Each mark's
// text is written at the start of line `y`. A text longer than the 10
// columns wraps onto the following lines, which are then not separated
// again.
func saFill(t *testing.T, term *Terminal, total uint, marks []saMark) {
	t.Helper()
	s := term.VtStream()
	defer s.Deinit()
	var y uint = 0
	for y < total {
		cur := y
		for _, m := range marks {
			if m.y != y {
				continue
			}
			zigTry(t, s.NextSlice([]uint8(m.text)))
			cur = max(cur, y+(uint(len(m.text))-1)/10)
		}
		if cur+1 < total {
			zigTry(t, s.NextSlice([]uint8("\r\n")))
		}
		y = cur + 1
	}
}

// saWrite is term51ALWrite.
func saWrite(t *testing.T, term *Terminal, data string) {
	t.Helper()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8(data)))
}

// saNewlines is term51ALNewlines.
func saNewlines(t *testing.T, term *Terminal, n uint) {
	t.Helper()
	s := term.VtStream()
	defer s.Deinit()
	for i := uint(0); i < n; i++ {
		zigTry(t, s.NextSlice([]uint8("\r\n")))
	}
}

// saCap is term51ALCap: the row capacity of the first (standard) page.
func saCap(term *Terminal) uint {
	return uint(term.Screens.Active.Pages.pages.first.data.capacity.rows)
}

// saScreen is term51ALScreen: the screen coordinate of a pin.
func saScreen(t *testing.T, term *Terminal, p PageListPin) PointCoordinate {
	t.Helper()
	return zigUnwrap(t, term.Screens.Active.Pages.PointFromPin(PointTagScreen, p)).Screen
}

// saActivePoint is `point.Point{ .active = .{ .x = x, .y = y } }`.
func saActivePoint(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagActive, Active: PointCoordinate{X: x, Y: y}}
}

// saScreenPoint is `point.Point{ .screen = .{ .x = x, .y = y } }`.
func saScreenPoint(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagScreen, Screen: PointCoordinate{X: x, Y: y}}
}

// saWrapLast is term51ALWrapLast: sets the soft-wrap flag of the last row of
// a node.
func saWrapLast(node *PageListNode) {
	node.data.GetRow(uint(node.data.size.rows - 1)).wrap = true
}

// saMetaNode is term51ALMetaNode: the node of the i-th meta of a search
// window.
func saMetaNode(w *SearchSlidingWindow, i uint) *PageListNode {
	it := w.meta.Iterator(DatastructCircBufIteratorDirectionForward)
	var n uint = 0
	for {
		m := it.Next()
		if m == nil {
			break
		}
		if n == i {
			return m.node
		}
		n++
	}
	panic("unreachable")
}

// saHasPin is term51ALHasPin: whether p is one of the tracked pins of list.
func saHasPin(list *PageList, p *PageListPin) bool {
	for _, q := range list.TrackedPins() {
		if q == p {
			return true
		}
	}
	return false
}

// saHasMethod reports whether the type of v has the exported method name
// (Zig's @hasDecl for a `pub fn`).
func saHasMethod(v any, name string) bool {
	_, ok := reflect.TypeOf(v).MethodByName(name)
	return ok
}

// Zig: test "term51: search: exports alias the search implementation types"
func TestSaSearch_term51_search_exports_alias_the_search_implementation_types(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: expect(Active == @import("search/active.zig").ActiveSearch): search.Active has no separate Go name, SearchActive is the type
	expect(t, reflect.TypeOf(SearchPageList{}) == reflect.TypeOf(SearchPagelist{}), "PageList == @import(\"search/pagelist.zig\").PageListSearch")
	expectEqualStrings(t, "SearchPagelist", reflect.TypeOf(SearchPageList{}).Name())
	// Zig-only: expect(Screen == @import("search/screen.zig").ScreenSearch): search.Screen has no separate Go name, SearchScreen is the type
	// Zig-only: expect(Viewport == @import("search/viewport.zig").ViewportSearch): search.Viewport has no separate Go name, SearchViewport is the type
	// Zig-only: expect(options == @import("terminal_options")): `options` is not ported (build options)

	// The same declarations are reachable through the terminal package.
	// Zig-only: expect(terminal.search.Active == Active) and the same for PageList, Screen, Viewport: Go has one package and no `terminal.search` namespace
	// Zig-only: expect(terminal.search.Thread == Thread): `Thread` is not ported
}

// Zig: test "term51: search: public declarations are exactly options Active PageList Screen Viewport Thread"
func TestSaSearch_term51_search_public_declarations_are_exactly_options_Active_PageList_Screen_Viewport_Thread(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: decls.len == 6 (@typeInfo of the namespace): Go has no declaration reflection for a package
	// Zig-only: @hasDecl(@This(), "options") and @hasDecl(@This(), "Thread"): `options` and `Thread` are not ported
	// The four exported types exist under their search.zig names.
	expectEqualStrings(t, "SearchActive", reflect.TypeOf(SearchActive{}).Name())
	expectEqualStrings(t, "SearchPagelist", reflect.TypeOf(SearchPageList{}).Name())
	expectEqualStrings(t, "SearchScreen", reflect.TypeOf(SearchScreen{}).Name())
	expectEqualStrings(t, "SearchViewport", reflect.TypeOf(SearchViewport{}).Name())
	// Zig-only: expect(!@hasDecl(@This(), "SlidingWindow")) and "sliding_window": Go exports the sliding window as SearchSlidingWindow (see the wireframe header of search_sliding_window.go)
}

// Zig: test "term51: search: exported search types expose their public functions"
func TestSaSearch_term51_search_exported_search_types_expose_their_public_functions(t *testing.T) {
	testStdCapacity(t)
	// `init` is a package-level function in Go: referencing it checks it exists.
	_ = SearchActiveInit
	_ = SearchViewportInit
	for _, name := range []string{"Deinit", "Update", "Next"} {
		expect(t, saHasMethod(&SearchActive{}, name), "@hasDecl(Active, name)")
		expect(t, saHasMethod(&SearchViewport{}, name), "@hasDecl(Viewport, name)")
	}
	for _, name := range []string{"Reset", "Needle"} {
		expect(t, saHasMethod(&SearchViewport{}, name), "@hasDecl(Viewport, name)")
		expect(t, !saHasMethod(&SearchActive{}, name), "!@hasDecl(Active, name)")
	}
	_ = SearchPagelistInit
	for _, name := range []string{"Deinit", "Next", "Feed"} {
		expect(t, saHasMethod(&SearchPageList{}, name), "@hasDecl(PageList, name)")
	}
	expect(t, !saHasMethod(&SearchPageList{}, "Update"), "!@hasDecl(PageList, \"update\")")
	_ = SearchScreenInit
	for _, name := range []string{"Deinit", "Needle", "MatchesLen", "Matches", "SearchAll", "Tick", "Feed", "ReloadActive", "SelectedMatch", "Select"} {
		expect(t, saHasMethod(&SearchScreen{}, name), "@hasDecl(Screen, name)")
	}
}

// Zig: test "term51: search: exported Active and Viewport searches work through the package exports"
func TestSaSearch_term51_search_exported_Active_and_Viewport_searches_work_through_the_package_exports(t *testing.T) {
	testStdCapacity(t)
	term, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer term.Deinit()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("Fizz")))

	active, err := SearchActiveInit([]uint8("fizz"))
	zigTry(t, err)
	defer active.Deinit()
	_, err = active.Update(&term.Screens.Active.Pages)
	zigTry(t, err)
	expect(t, active.Next().Valid, "active.next() != null")
	expect(t, !active.Next().Valid, "active.next() == null")

	viewport, err := SearchViewportInit([]uint8("fizz"))
	zigTry(t, err)
	defer viewport.Deinit()
	changed, err := viewport.Update(&term.Screens.Active.Pages)
	zigTry(t, err)
	expect(t, changed, "try viewport.update(&t.screens.active.pages)")
	expect(t, viewport.Next().Valid, "viewport.next() != null")
	expect(t, !viewport.Next().Valid, "viewport.next() == null")

	pagelist, err := SearchPagelistInit([]uint8("fizz"), &term.Screens.Active.Pages, term.Screens.Active.Pages.pages.last)
	zigTry(t, err)
	defer pagelist.Deinit()
	expect(t, pagelist.Next().Valid, "pagelist.next() != null")
	expect(t, !pagelist.Next().Valid, "pagelist.next() == null")
	fed, err := pagelist.Feed()
	zigTry(t, err)
	expect(t, !fed, "!try pagelist.feed()")
}
