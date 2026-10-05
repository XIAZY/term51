// Tests translated one for one from src/terminal/search/sliding_window.zig
// (Ghostty v1.3.1 plus the term51 additional tests), from the first test
// (`test "SlidingWindow empty on init"`) to `test "term51: SlidingWindow.append:
// reverse reverses the data and the cell map"`. Each Go test carries the exact
// Zig title in a "// Zig: test" comment. Helpers are prefixed sw1.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualSlices / expectEqualStrings keep Zig's names and
//     Zig's (expected, actual) argument order. `try` on a call that returns an
//     error is zigTry(t, err); `x.?` is zigUnwrap (or sw1Unwrap for a pointer).
//   - `SlidingWindow.init(alloc, dir, needle)` is sw1Init (always the Init
//     function, never a literal); the window is a *SearchSlidingWindow so that
//     Zig's `&w` is `w`. `Screen.init(alloc, .{...})` is sw1Screen (the Zig
//     defaults written out). `s.testWriteString(...)` is sw1Write.
//   - The repeated block `const h = w.next().?; const sel = h.untracked();
//     expectEqual(point.Point{ .active = ... }, s.pages.pointFromPin(.active,
//     sel.start)); expectEqual(... sel.end)` is the helper sw1ExpectNext (same
//     assertions, same order).
//   - Every test starts with testStdCapacity(t): Zig's unit tests run with
//     grapheme_bytes 512, and the page-boundary tests depend on the rows per
//     page.
//   - `const alloc = testing.allocator;`, the allocator arguments and
//     `defer alloc.free(...)` are allocator bookkeeping with no Go counterpart;
//     they are dropped without a per-test note. Other allocator-only lines
//     (leak checks) are dropped with a `// Zig-only:` comment.
//   - Zig's `std.MultiArrayList` chunk_buf is a Go slice: `.len` is len(),
//     `.capacity` is cap().
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt.
//   - Go-only tests at the end of the file (title `// Go-only:`) test the
//     assert / panic paths of the functions under test that Zig's tests
//     cannot reach, and the release-build behaviour of `next` when pruning
//     leaves exactly needle.len - 1 bytes.

package terminal

import (
	"bytes"
	"testing"
)

// sw1ExpectPanic runs f and fails unless it panics. The wireframe's
// not-implemented stub does not count: a test that only reaches the stub must
// fail until the implementation panics for the right reason. Zig's tests
// cannot test an assert or `unreachable`: the test would abort.
func sw1ExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("expected a panic, got none")
		}
		if s, ok := r.(string); ok && s == notImplemented {
			t.Fatalf("panicked with the not-implemented stub, not with the expected panic")
		}
	}()
	f()
}

// sw1Unwrap is Zig's `p.?` on an optional pointer.
func sw1Unwrap[T any](t *testing.T, p *T) *T {
	t.Helper()
	if p == nil {
		t.Fatalf("unwrapped a null pointer")
	}
	return p
}

// sw1Init is `try SlidingWindow.init(alloc, direction, needle)`.
func sw1Init(t *testing.T, direction searchSlidingWindowDirection, needle []uint8) *SearchSlidingWindow {
	t.Helper()
	w, err := SearchSlidingWindowInit(direction, needle)
	zigTry(t, err)
	return &w
}

// sw1Screen is `try Screen.init(alloc, .{ .cols = cols, .rows = rows,
// .max_scrollback = max_scrollback })`: kitty_image_storage_limit is the Zig
// default, 320 * 1000 * 1000.
func sw1Screen(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt, max_scrollback uint64) *Screen {
	t.Helper()
	s, err := ScreenInit(ScreenOptions{
		cols:                      cols,
		rows:                      rows,
		max_scrollback:            max_scrollback,
		kitty_image_storage_limit: 320 * 1000 * 1000,
	})
	zigTry(t, err)
	return &s
}

// sw1Write is `try s.testWriteString(text)`.
func sw1Write(t *testing.T, s *Screen, text string) {
	t.Helper()
	zigTry(t, s.TestWriteString([]uint8(text)))
}

// sw1Append is `try w.append(node)`.
func sw1Append(t *testing.T, w *SearchSlidingWindow, node *PageListNode) uint {
	t.Helper()
	n, err := w.Append(node)
	zigTry(t, err)
	return n
}

// sw1Active is the Zig literal `point.Point{ .active = .{ .x = x, .y = y } }`.
func sw1Active(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagActive, Active: PointCoordinate{x: x, y: y}}
}

// sw1ActivePt is `pages.pointFromPin(.active, pin).?`. (Where Zig compares
// the optional itself with expectEqual, a null result fails the test just as
// the unwrap does.)
func sw1ActivePt(t *testing.T, pages *PageList, pin PageListPin) Point {
	t.Helper()
	return zigUnwrap(t, pages.PointFromPin(PointTagActive, pin))
}

// sw1ExpectNext is the block repeated throughout the tests:
//
//	const h = w.next().?;
//	const sel = h.untracked();
//	try testing.expectEqual(point.Point{ .active = .{ .x = sx, .y = sy } },
//	    pages.pointFromPin(.active, sel.start));
//	try testing.expectEqual(point.Point{ .active = .{ .x = ex, .y = ey } },
//	    pages.pointFromPin(.active, sel.end));
func sw1ExpectNext(
	t *testing.T,
	pages *PageList,
	w *SearchSlidingWindow,
	sx SizeCellCountInt, sy uint32,
	ex SizeCellCountInt, ey uint32,
) {
	t.Helper()
	h := zigUnwrap(t, w.Next())
	sel := h.Untracked()
	expectEqual(t, sw1Active(sx, sy), sw1ActivePt(t, pages, sel.start))
	expectEqual(t, sw1Active(ex, ey), sw1ActivePt(t, pages, sel.end))
}

// sw1Dirs is `.{ SlidingWindow.Direction.forward, SlidingWindow.Direction.reverse }`
// of an `inline for`.
var sw1Dirs = []searchSlidingWindowDirection{
	searchSlidingWindowDirectionForward,
	searchSlidingWindowDirectionReverse,
}

// ---------------------------------------------------------------------------
// term51: additional tests (AK) helpers.
// ---------------------------------------------------------------------------

// sw1AKCols is term51AKCols: column count of every page list built by
// sw1AKList.
const sw1AKCols SizeCellCountInt = 200

// sw1AKChunk is term51AKChunk: the expected y bounds of one flattened
// highlight chunk.
type sw1AKChunk struct {
	node  *PageListNode
	start SizeCellCountInt
	end   SizeCellCountInt
}

// sw1AKList is term51AKList: a page list with exactly `pages` pages (the
// earlier ones are full) and `extra` additional rows grown on the last page
// after it was created.
func sw1AKList(t *testing.T, pages uint, extra uint) *PageList {
	t.Helper()
	pl, err := PageListInit(sw1AKCols, 5, None[uint64]())
	zigTry(t, err)
	p := &pl
	p.PauseIntegrityChecks(true)
	for sw1AKNodeCount(p) < pages {
		_, err := p.Grow()
		zigTry(t, err)
	}
	for range extra {
		_, err := p.Grow()
		zigTry(t, err)
	}
	p.PauseIntegrityChecks(false)
	return p
}

// sw1AKNodeCount is term51AKNodeCount.
func sw1AKNodeCount(pl *PageList) uint {
	var count uint
	it := pl.pages.first
	for it != nil {
		count++
		it = it.next
	}
	return count
}

// sw1AKNodeAt is term51AKNodeAt.
func sw1AKNodeAt(t *testing.T, pl *PageList, i uint) *PageListNode {
	t.Helper()
	n := sw1Unwrap(t, pl.pages.first)
	for range i {
		n = sw1Unwrap(t, n.next)
	}
	return n
}

// sw1AKPut is term51AKPut: write narrow ASCII cells into row `y` starting at
// column `x`.
func sw1AKPut(node *PageListNode, y uint, x uint, text string) {
	row := node.data.GetRow(y)
	cells := node.data.GetCells(row)
	for i, c := range []uint8(text) {
		cells[x+uint(i)] = PageCell{
			content_tag: PageCellContentTagCodepoint,
			content:     PageCellContent{codepoint: uint32(c)},
		}
	}
}

// sw1AKSetWrap is term51AKSetWrap: set the soft-wrap flag of the last row of
// the page.
func sw1AKSetWrap(node *PageListNode, wrap bool) {
	node.data.GetRow(uint(node.data.size.rows - 1)).wrap = wrap
}

// sw1AKXY is term51AKXY.
func sw1AKXY(x SizeCellCountInt, y uint32) PointCoordinate {
	return PointCoordinate{x: x, y: y}
}

// sw1AKExpectData is term51AKExpectData.
func sw1AKExpectData(t *testing.T, w *SearchSlidingWindow, expected string) {
	t.Helper()
	expectEqual(t, uint(len(expected)), w.data.Len())
	s := w.data.GetPtrSlice(0, w.data.Len())
	expectEqualStrings(t, expected[0:len(s[0])], string(s[0]))
	expectEqualStrings(t, expected[len(s[0]):], string(s[1]))
}

// sw1AKMeta is term51AKMeta.
func sw1AKMeta(t *testing.T, w *SearchSlidingWindow, i uint) *searchSlidingWindowMeta {
	t.Helper()
	it := w.meta.Iterator(DatastructCircBufIteratorDirectionForward)
	it.SeekBy(int(i))
	return sw1Unwrap(t, it.Next())
}

// sw1AKExpectMap is term51AKExpectMap.
func sw1AKExpectMap(t *testing.T, w *SearchSlidingWindow, i uint, expected []PointCoordinate) {
	t.Helper()
	expectEqualSlices(t, expected, sw1AKMeta(t, w, i).cell_map)
}

// sw1AKExpectHl is term51AKExpectHl.
func sw1AKExpectHl(
	t *testing.T,
	h HighlightFlattened,
	top_x SizeCellCountInt,
	bot_x SizeCellCountInt,
	want []sw1AKChunk,
) {
	t.Helper()
	expectEqual(t, len(want), len(h.chunks))
	for i, c := range want {
		expectEqual(t, c.node, h.chunks[i].node)
		expectEqual(t, c.start, h.chunks[i].start)
		expectEqual(t, c.end, h.chunks[i].end)
	}
	expectEqual(t, top_x, h.top_x)
	expectEqual(t, bot_x, h.bot_x)
}

// sw1AKMany is term51AKMany: a page list with one page per entry of `texts`.
// Each text is written at row 0 of its page, and the last row of every page
// gets the matching soft-wrap flag, so a wrapped page encodes to exactly its
// text and a non-wrapped page to its text plus a newline. All pages except
// the last are full (their last row is far below row 0), the last has one
// row.
func sw1AKMany(t *testing.T, texts []string, wraps []bool) *PageList {
	t.Helper()
	pl := sw1AKList(t, uint(len(texts)), 0)
	for i := range texts {
		node := sw1AKNodeAt(t, pl, uint(i))
		sw1AKPut(node, 0, 0, texts[i])
		sw1AKSetWrap(node, wraps[i])
	}
	return pl
}

// sw1AKThree is term51AKThree.
func sw1AKThree(t *testing.T, texts [3]string, wraps [3]bool) *PageList {
	t.Helper()
	return sw1AKMany(t, texts[:], wraps[:])
}

// sw1AKTwo is term51AKTwo.
func sw1AKTwo(t *testing.T, texts [2]string, wraps [2]bool) *PageList {
	t.Helper()
	return sw1AKMany(t, texts[:], wraps[:])
}

// ---------------------------------------------------------------------------
// Upstream tests.
// ---------------------------------------------------------------------------

// Zig: test "SlidingWindow empty on init"
func TestSw1_SlidingWindow_empty_on_init(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("boo!"))
	defer w.Deinit()
	expectEqual(t, 0, w.data.Len())
	expectEqual(t, 0, w.meta.Len())
}

// Zig: test "SlidingWindow single append"
func TestSw1_SlidingWindow_single_append(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("boo!"))
	defer w.Deinit()

	s := sw1Screen(t, 80, 24, 0)
	defer s.Deinit()
	sw1Write(t, s, "hello. boo! hello. boo!")

	// We want to test single-page cases.
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	node := sw1Unwrap(t, s.pages.pages.first)
	_ = sw1Append(t, w, node)

	// We should be able to find two matches.
	sw1ExpectNext(t, &s.pages, w, 7, 0, 10, 0)
	sw1ExpectNext(t, &s.pages, w, 19, 0, 22, 0)
	expect(t, !w.Next().Valid, "w.next() == null")
	expect(t, !w.Next().Valid, "w.next() == null")
}

// Zig: test "SlidingWindow single append case insensitive ASCII"
func TestSw1_SlidingWindow_single_append_case_insensitive_ASCII(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("Boo!"))
	defer w.Deinit()

	s := sw1Screen(t, 80, 24, 0)
	defer s.Deinit()
	sw1Write(t, s, "hello. boo! hello. boo!")

	// We want to test single-page cases.
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	node := sw1Unwrap(t, s.pages.pages.first)
	_ = sw1Append(t, w, node)

	// We should be able to find two matches.
	sw1ExpectNext(t, &s.pages, w, 7, 0, 10, 0)
	sw1ExpectNext(t, &s.pages, w, 19, 0, 22, 0)
	expect(t, !w.Next().Valid, "w.next() == null")
	expect(t, !w.Next().Valid, "w.next() == null")
}

// Zig: test "SlidingWindow single append single char"
func TestSw1_SlidingWindow_single_append_single_char(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("b"))
	defer w.Deinit()

	s := sw1Screen(t, 80, 24, 0)
	defer s.Deinit()
	sw1Write(t, s, "hello. boo! hello. boo!")

	// We want to test single-page cases.
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	node := sw1Unwrap(t, s.pages.pages.first)
	_ = sw1Append(t, w, node)

	// We should be able to find two matches.
	sw1ExpectNext(t, &s.pages, w, 7, 0, 7, 0)
	sw1ExpectNext(t, &s.pages, w, 19, 0, 19, 0)
	expect(t, !w.Next().Valid, "w.next() == null")
	expect(t, !w.Next().Valid, "w.next() == null")
}

// Zig: test "SlidingWindow single append no match"
func TestSw1_SlidingWindow_single_append_no_match(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("nope!"))
	defer w.Deinit()

	s := sw1Screen(t, 80, 24, 0)
	defer s.Deinit()
	sw1Write(t, s, "hello. boo! hello. boo!")

	// We want to test single-page cases.
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	node := sw1Unwrap(t, s.pages.pages.first)
	_ = sw1Append(t, w, node)

	// No matches
	expect(t, !w.Next().Valid, "w.next() == null")
	expect(t, !w.Next().Valid, "w.next() == null")

	// Should still keep the page
	expectEqual(t, 1, w.meta.Len())
}

// Zig: test "SlidingWindow two pages"
func TestSw1_SlidingWindow_two_pages(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("boo!"))
	defer w.Deinit()

	s := sw1Screen(t, 80, 24, 1000)
	defer s.Deinit()

	// Fill up the first page. The final bytes in the first page
	// are "boo!"
	first_page_rows := sw1Unwrap(t, s.pages.pages.first).data.capacity.rows
	for range first_page_rows - 1 {
		sw1Write(t, s, "\n")
	}
	for range s.pages.cols - 4 {
		sw1Write(t, s, "x")
	}
	sw1Write(t, s, "boo!")
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	sw1Write(t, s, "\n")
	expect(t, s.pages.pages.first != s.pages.pages.last, "s.pages.pages.first != s.pages.pages.last")
	sw1Write(t, s, "hello. boo!")

	// Add both pages
	node := sw1Unwrap(t, s.pages.pages.first)
	_ = sw1Append(t, w, node)
	_ = sw1Append(t, w, sw1Unwrap(t, node.next))

	// Search should find two matches
	sw1ExpectNext(t, &s.pages, w, 76, 22, 79, 22)
	sw1ExpectNext(t, &s.pages, w, 7, 23, 10, 23)
	expect(t, !w.Next().Valid, "w.next() == null")
	expect(t, !w.Next().Valid, "w.next() == null")
}

// Zig: test "SlidingWindow two pages single char"
func TestSw1_SlidingWindow_two_pages_single_char(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("b"))
	defer w.Deinit()

	s := sw1Screen(t, 80, 24, 1000)
	defer s.Deinit()

	// Fill up the first page. The final bytes in the first page
	// are "boo!"
	first_page_rows := sw1Unwrap(t, s.pages.pages.first).data.capacity.rows
	for range first_page_rows - 1 {
		sw1Write(t, s, "\n")
	}
	for range s.pages.cols - 4 {
		sw1Write(t, s, "x")
	}
	sw1Write(t, s, "boo!")
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	sw1Write(t, s, "\n")
	expect(t, s.pages.pages.first != s.pages.pages.last, "s.pages.pages.first != s.pages.pages.last")
	sw1Write(t, s, "hello. boo!")

	// Add both pages
	node := sw1Unwrap(t, s.pages.pages.first)
	_ = sw1Append(t, w, node)
	_ = sw1Append(t, w, sw1Unwrap(t, node.next))

	// Search should find two matches
	sw1ExpectNext(t, &s.pages, w, 76, 22, 76, 22)
	sw1ExpectNext(t, &s.pages, w, 7, 23, 7, 23)
	expect(t, !w.Next().Valid, "w.next() == null")
	expect(t, !w.Next().Valid, "w.next() == null")
}

// Zig: test "SlidingWindow two pages match across boundary"
func TestSw1_SlidingWindow_two_pages_match_across_boundary(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("hello, world"))
	defer w.Deinit()

	s := sw1Screen(t, 80, 24, 1000)
	defer s.Deinit()

	// Fill up the first page. The final bytes in the first page
	// are "boo!"
	first_page_rows := sw1Unwrap(t, s.pages.pages.first).data.capacity.rows
	for range first_page_rows - 1 {
		sw1Write(t, s, "\n")
	}
	for range s.pages.cols - 4 {
		sw1Write(t, s, "x")
	}
	sw1Write(t, s, "hell")
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	sw1Write(t, s, "o, world!")
	expect(t, s.pages.pages.first != s.pages.pages.last, "s.pages.pages.first != s.pages.pages.last")

	// Add both pages
	node := sw1Unwrap(t, s.pages.pages.first)
	_ = sw1Append(t, w, node)
	_ = sw1Append(t, w, sw1Unwrap(t, node.next))

	// Search should find a match
	sw1ExpectNext(t, &s.pages, w, 76, 22, 7, 23)
	expect(t, !w.Next().Valid, "w.next() == null")
	expect(t, !w.Next().Valid, "w.next() == null")

	// We shouldn't prune because we don't have enough space
	expectEqual(t, 2, w.meta.Len())
}

// Zig: test "SlidingWindow two pages no match across boundary with newline"
func TestSw1_SlidingWindow_two_pages_no_match_across_boundary_with_newline(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("hello, world"))
	defer w.Deinit()

	s := sw1Screen(t, 80, 24, 1000)
	defer s.Deinit()

	// Fill up the first page. The final bytes in the first page
	// are "boo!"
	first_page_rows := sw1Unwrap(t, s.pages.pages.first).data.capacity.rows
	for range first_page_rows - 1 {
		sw1Write(t, s, "\n")
	}
	for range s.pages.cols - 4 {
		sw1Write(t, s, "x")
	}
	sw1Write(t, s, "hell")
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	sw1Write(t, s, "\no, world!")
	expect(t, s.pages.pages.first != s.pages.pages.last, "s.pages.pages.first != s.pages.pages.last")

	// Add both pages
	node := sw1Unwrap(t, s.pages.pages.first)
	_ = sw1Append(t, w, node)
	_ = sw1Append(t, w, sw1Unwrap(t, node.next))

	// Search should NOT find a match
	expect(t, !w.Next().Valid, "w.next() == null")
	expect(t, !w.Next().Valid, "w.next() == null")

	// We shouldn't prune because we don't have enough space
	expectEqual(t, 2, w.meta.Len())
}

// Zig: test "SlidingWindow two pages no match across boundary with newline reverse"
func TestSw1_SlidingWindow_two_pages_no_match_across_boundary_with_newline_reverse(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionReverse, []uint8("hello, world"))
	defer w.Deinit()

	s := sw1Screen(t, 80, 24, 1000)
	defer s.Deinit()

	// Fill up the first page. The final bytes in the first page
	// are "boo!"
	first_page_rows := sw1Unwrap(t, s.pages.pages.first).data.capacity.rows
	for range first_page_rows - 1 {
		sw1Write(t, s, "\n")
	}
	for range s.pages.cols - 4 {
		sw1Write(t, s, "x")
	}
	sw1Write(t, s, "hell")
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	sw1Write(t, s, "\no, world!")
	expect(t, s.pages.pages.first != s.pages.pages.last, "s.pages.pages.first != s.pages.pages.last")

	// Add both pages in reverse order
	node := sw1Unwrap(t, s.pages.pages.first)
	_ = sw1Append(t, w, sw1Unwrap(t, node.next))
	_ = sw1Append(t, w, node)

	// Search should NOT find a match
	expect(t, !w.Next().Valid, "w.next() == null")
	expect(t, !w.Next().Valid, "w.next() == null")
}

// Zig: test "SlidingWindow two pages no match prunes first page"
func TestSw1_SlidingWindow_two_pages_no_match_prunes_first_page(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("nope!"))
	defer w.Deinit()

	s := sw1Screen(t, 80, 24, 1000)
	defer s.Deinit()

	// Fill up the first page. The final bytes in the first page
	// are "boo!"
	first_page_rows := sw1Unwrap(t, s.pages.pages.first).data.capacity.rows
	for range first_page_rows - 1 {
		sw1Write(t, s, "\n")
	}
	for range s.pages.cols - 4 {
		sw1Write(t, s, "x")
	}
	sw1Write(t, s, "boo!")
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	sw1Write(t, s, "\n")
	expect(t, s.pages.pages.first != s.pages.pages.last, "s.pages.pages.first != s.pages.pages.last")
	sw1Write(t, s, "hello. boo!")

	// Add both pages
	node := sw1Unwrap(t, s.pages.pages.first)
	_ = sw1Append(t, w, node)
	_ = sw1Append(t, w, sw1Unwrap(t, node.next))

	// Search should find nothing
	expect(t, !w.Next().Valid, "w.next() == null")
	expect(t, !w.Next().Valid, "w.next() == null")

	// We should've pruned our page because the second page
	// has enough text to contain our needle.
	expectEqual(t, 1, w.meta.Len())
}

// Zig: test "SlidingWindow two pages no match keeps both pages"
func TestSw1_SlidingWindow_two_pages_no_match_keeps_both_pages(t *testing.T) {
	testStdCapacity(t)

	s := sw1Screen(t, 80, 24, 1000)
	defer s.Deinit()

	// Fill up the first page. The final bytes in the first page
	// are "boo!"
	first_page_rows := sw1Unwrap(t, s.pages.pages.first).data.capacity.rows
	for range first_page_rows - 1 {
		sw1Write(t, s, "\n")
	}
	for range s.pages.cols - 4 {
		sw1Write(t, s, "x")
	}
	sw1Write(t, s, "boo!")
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	sw1Write(t, s, "\n")
	expect(t, s.pages.pages.first != s.pages.pages.last, "s.pages.pages.first != s.pages.pages.last")
	sw1Write(t, s, "hello. boo!")

	// Imaginary needle for search. Doesn't match!
	needle := bytes.Repeat([]uint8{'x'}, int(uint(first_page_rows)*uint(s.pages.cols)))

	w := sw1Init(t, searchSlidingWindowDirectionForward, needle)
	defer w.Deinit()

	// Add both pages
	node := sw1Unwrap(t, s.pages.pages.first)
	_ = sw1Append(t, w, node)
	_ = sw1Append(t, w, sw1Unwrap(t, node.next))

	// Search should find nothing
	expect(t, !w.Next().Valid, "w.next() == null")
	expect(t, !w.Next().Valid, "w.next() == null")

	// No pruning because both pages are needed to fit needle.
	expectEqual(t, 2, w.meta.Len())
}

// Zig: test "SlidingWindow single append across circular buffer boundary"
func TestSw1_SlidingWindow_single_append_across_circular_buffer_boundary(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("abc"))
	defer w.Deinit()

	s := sw1Screen(t, 80, 24, 0)
	defer s.Deinit()
	sw1Write(t, s, "XXXXXXXXXXXXXXXXXXXboo!XXXXX")

	// We are trying to break a circular buffer boundary so the way we
	// do this is to duplicate the data then do a failing search. This
	// will cause the first page to be pruned. The next time we append we'll
	// put it in the middle of the circ buffer. We assert this so that if
	// our implementation changes our test will fail.
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	node := sw1Unwrap(t, s.pages.pages.first)
	_ = sw1Append(t, w, node)
	_ = sw1Append(t, w, node)
	{
		// No wrap around yet
		slices := w.data.GetPtrSlice(0, w.data.Len())
		expect(t, len(slices[0]) > 0, "slices[0].len > 0")
		expect(t, len(slices[1]) == 0, "slices[1].len == 0")
	}

	// Search non-match, prunes page
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 1, w.meta.Len())

	// Change the needle, just needs to be the same length (not a real API)
	w.testChangeNeedle([]uint8("boo"))

	// Add new page, now wraps
	_ = sw1Append(t, w, node)
	{
		slices := w.data.GetPtrSlice(0, w.data.Len())
		expect(t, len(slices[0]) > 0, "slices[0].len > 0")
		expect(t, len(slices[1]) > 0, "slices[1].len > 0")
	}
	sw1ExpectNext(t, &s.pages, w, 19, 0, 21, 0)
	expect(t, !w.Next().Valid, "w.next() == null")
}

// Zig: test "SlidingWindow single append match on boundary"
func TestSw1_SlidingWindow_single_append_match_on_boundary(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("abcd"))
	defer w.Deinit()

	s := sw1Screen(t, 80, 24, 0)
	defer s.Deinit()
	sw1Write(t, s, "o!XXXXXXXXXXXXXXXXXXXbo")

	// We need to surgically modify the last row to be soft-wrapped
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	node := sw1Unwrap(t, s.pages.pages.first)
	node.data.GetRow(uint(node.data.size.rows - 1)).wrap = true

	// We are trying to break a circular buffer boundary so the way we
	// do this is to duplicate the data then do a failing search. This
	// will cause the first page to be pruned. The next time we append we'll
	// put it in the middle of the circ buffer. We assert this so that if
	// our implementation changes our test will fail.
	_ = sw1Append(t, w, node)
	_ = sw1Append(t, w, node)
	{
		// No wrap around yet
		slices := w.data.GetPtrSlice(0, w.data.Len())
		expect(t, len(slices[0]) > 0, "slices[0].len > 0")
		expect(t, len(slices[1]) == 0, "slices[1].len == 0")
	}

	// Search non-match, prunes page
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 1, w.meta.Len())

	// Change the needle, just needs to be the same length (not a real API)
	w.testChangeNeedle([]uint8("boo!"))

	// Add new page, now wraps
	_ = sw1Append(t, w, node)
	{
		slices := w.data.GetPtrSlice(0, w.data.Len())
		expect(t, len(slices[0]) > 0, "slices[0].len > 0")
		expect(t, len(slices[1]) > 0, "slices[1].len > 0")
	}
	sw1ExpectNext(t, &s.pages, w, 21, 0, 1, 0)
	expect(t, !w.Next().Valid, "w.next() == null")
}

// Zig: test "SlidingWindow single append reversed"
func TestSw1_SlidingWindow_single_append_reversed(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionReverse, []uint8("boo!"))
	defer w.Deinit()

	s := sw1Screen(t, 80, 24, 0)
	defer s.Deinit()
	sw1Write(t, s, "hello. boo! hello. boo!")

	// We want to test single-page cases.
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	node := sw1Unwrap(t, s.pages.pages.first)
	_ = sw1Append(t, w, node)

	// We should be able to find two matches.
	sw1ExpectNext(t, &s.pages, w, 19, 0, 22, 0)
	sw1ExpectNext(t, &s.pages, w, 7, 0, 10, 0)
	expect(t, !w.Next().Valid, "w.next() == null")
	expect(t, !w.Next().Valid, "w.next() == null")
}

// Zig: test "SlidingWindow single append no match reversed"
func TestSw1_SlidingWindow_single_append_no_match_reversed(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionReverse, []uint8("nope!"))
	defer w.Deinit()

	s := sw1Screen(t, 80, 24, 0)
	defer s.Deinit()
	sw1Write(t, s, "hello. boo! hello. boo!")

	// We want to test single-page cases.
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	node := sw1Unwrap(t, s.pages.pages.first)
	_ = sw1Append(t, w, node)

	// No matches
	expect(t, !w.Next().Valid, "w.next() == null")
	expect(t, !w.Next().Valid, "w.next() == null")

	// Should still keep the page
	expectEqual(t, 1, w.meta.Len())
}

// Zig: test "SlidingWindow two pages reversed"
func TestSw1_SlidingWindow_two_pages_reversed(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionReverse, []uint8("boo!"))
	defer w.Deinit()

	s := sw1Screen(t, 80, 24, 1000)
	defer s.Deinit()

	// Fill up the first page. The final bytes in the first page
	// are "boo!"
	first_page_rows := sw1Unwrap(t, s.pages.pages.first).data.capacity.rows
	for range first_page_rows - 1 {
		sw1Write(t, s, "\n")
	}
	for range s.pages.cols - 4 {
		sw1Write(t, s, "x")
	}
	sw1Write(t, s, "boo!")
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	sw1Write(t, s, "\n")
	expect(t, s.pages.pages.first != s.pages.pages.last, "s.pages.pages.first != s.pages.pages.last")
	sw1Write(t, s, "hello. boo!")

	// Add both pages in reverse order
	node := sw1Unwrap(t, s.pages.pages.first)
	_ = sw1Append(t, w, sw1Unwrap(t, node.next))
	_ = sw1Append(t, w, node)

	// Search should find two matches (in reverse order)
	sw1ExpectNext(t, &s.pages, w, 7, 23, 10, 23)
	sw1ExpectNext(t, &s.pages, w, 76, 22, 79, 22)
	expect(t, !w.Next().Valid, "w.next() == null")
	expect(t, !w.Next().Valid, "w.next() == null")
}

// Zig: test "SlidingWindow two pages match across boundary reversed"
func TestSw1_SlidingWindow_two_pages_match_across_boundary_reversed(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionReverse, []uint8("hello, world"))
	defer w.Deinit()

	s := sw1Screen(t, 80, 24, 1000)
	defer s.Deinit()

	// Fill up the first page. The final bytes in the first page
	// are "hell"
	first_page_rows := sw1Unwrap(t, s.pages.pages.first).data.capacity.rows
	for range first_page_rows - 1 {
		sw1Write(t, s, "\n")
	}
	for range s.pages.cols - 4 {
		sw1Write(t, s, "x")
	}
	sw1Write(t, s, "hell")
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	sw1Write(t, s, "o, world!")
	expect(t, s.pages.pages.first != s.pages.pages.last, "s.pages.pages.first != s.pages.pages.last")

	// Add both pages in reverse order
	node := sw1Unwrap(t, s.pages.pages.first)
	_ = sw1Append(t, w, sw1Unwrap(t, node.next))
	_ = sw1Append(t, w, node)

	// Search should find a match
	sw1ExpectNext(t, &s.pages, w, 76, 22, 7, 23)
	expect(t, !w.Next().Valid, "w.next() == null")
	expect(t, !w.Next().Valid, "w.next() == null")

	// In reverse mode, the last appended meta (first original page) is large
	// enough to contain needle.len - 1 bytes, so pruning occurs
	expectEqual(t, 1, w.meta.Len())
}

// Zig: test "SlidingWindow two pages no match prunes first page reversed"
func TestSw1_SlidingWindow_two_pages_no_match_prunes_first_page_reversed(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionReverse, []uint8("nope!"))
	defer w.Deinit()

	s := sw1Screen(t, 80, 24, 1000)
	defer s.Deinit()

	// Fill up the first page. The final bytes in the first page
	// are "boo!"
	first_page_rows := sw1Unwrap(t, s.pages.pages.first).data.capacity.rows
	for range first_page_rows - 1 {
		sw1Write(t, s, "\n")
	}
	for range s.pages.cols - 4 {
		sw1Write(t, s, "x")
	}
	sw1Write(t, s, "boo!")
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	sw1Write(t, s, "\n")
	expect(t, s.pages.pages.first != s.pages.pages.last, "s.pages.pages.first != s.pages.pages.last")
	sw1Write(t, s, "hello. boo!")

	// Add both pages in reverse order
	node := sw1Unwrap(t, s.pages.pages.first)
	_ = sw1Append(t, w, sw1Unwrap(t, node.next))
	_ = sw1Append(t, w, node)

	// Search should find nothing
	expect(t, !w.Next().Valid, "w.next() == null")
	expect(t, !w.Next().Valid, "w.next() == null")

	// We should've pruned our page because the second page
	// has enough text to contain our needle.
	expectEqual(t, 1, w.meta.Len())
}

// Zig: test "SlidingWindow two pages no match keeps both pages reversed"
func TestSw1_SlidingWindow_two_pages_no_match_keeps_both_pages_reversed(t *testing.T) {
	testStdCapacity(t)

	s := sw1Screen(t, 80, 24, 1000)
	defer s.Deinit()

	// Fill up the first page. The final bytes in the first page
	// are "boo!"
	first_page_rows := sw1Unwrap(t, s.pages.pages.first).data.capacity.rows
	for range first_page_rows - 1 {
		sw1Write(t, s, "\n")
	}
	for range s.pages.cols - 4 {
		sw1Write(t, s, "x")
	}
	sw1Write(t, s, "boo!")
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	sw1Write(t, s, "\n")
	expect(t, s.pages.pages.first != s.pages.pages.last, "s.pages.pages.first != s.pages.pages.last")
	sw1Write(t, s, "hello. boo!")

	// Imaginary needle for search. Doesn't match!
	needle := bytes.Repeat([]uint8{'x'}, int(uint(first_page_rows)*uint(s.pages.cols)))

	w := sw1Init(t, searchSlidingWindowDirectionReverse, needle)
	defer w.Deinit()

	// Add both pages in reverse order
	node := sw1Unwrap(t, s.pages.pages.first)
	_ = sw1Append(t, w, sw1Unwrap(t, node.next))
	_ = sw1Append(t, w, node)

	// Search should find nothing
	expect(t, !w.Next().Valid, "w.next() == null")
	expect(t, !w.Next().Valid, "w.next() == null")

	// No pruning because both pages are needed to fit needle.
	expectEqual(t, 2, w.meta.Len())
}

// Zig: test "SlidingWindow single append across circular buffer boundary reversed"
func TestSw1_SlidingWindow_single_append_across_circular_buffer_boundary_reversed(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionReverse, []uint8("abc"))
	defer w.Deinit()

	s := sw1Screen(t, 80, 24, 0)
	defer s.Deinit()
	sw1Write(t, s, "XXXXXXXXXXXXXXXXXXXboo!XXXXX")

	// We are trying to break a circular buffer boundary so the way we
	// do this is to duplicate the data then do a failing search. This
	// will cause the first page to be pruned. The next time we append we'll
	// put it in the middle of the circ buffer. We assert this so that if
	// our implementation changes our test will fail.
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	node := sw1Unwrap(t, s.pages.pages.first)
	_ = sw1Append(t, w, node)
	_ = sw1Append(t, w, node)
	{
		// No wrap around yet
		slices := w.data.GetPtrSlice(0, w.data.Len())
		expect(t, len(slices[0]) > 0, "slices[0].len > 0")
		expect(t, len(slices[1]) == 0, "slices[1].len == 0")
	}

	// Search non-match, prunes page
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 1, w.meta.Len())

	// Change the needle, just needs to be the same length (not a real API)
	// testChangeNeedle doesn't reverse, so pass reversed needle for reverse mode
	w.testChangeNeedle([]uint8("oob"))

	// Add new page, now wraps
	_ = sw1Append(t, w, node)
	{
		slices := w.data.GetPtrSlice(0, w.data.Len())
		expect(t, len(slices[0]) > 0, "slices[0].len > 0")
		expect(t, len(slices[1]) > 0, "slices[1].len > 0")
	}
	sw1ExpectNext(t, &s.pages, w, 19, 0, 21, 0)
	expect(t, !w.Next().Valid, "w.next() == null")
}

// Zig: test "SlidingWindow single append match on boundary reversed"
func TestSw1_SlidingWindow_single_append_match_on_boundary_reversed(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionReverse, []uint8("abcd"))
	defer w.Deinit()

	s := sw1Screen(t, 80, 24, 0)
	defer s.Deinit()
	sw1Write(t, s, "o!XXXXXXXXXXXXXXXXXXXbo")

	// We need to surgically modify the last row to be soft-wrapped
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	node := sw1Unwrap(t, s.pages.pages.first)
	node.data.GetRow(uint(node.data.size.rows - 1)).wrap = true

	// We are trying to break a circular buffer boundary so the way we
	// do this is to duplicate the data then do a failing search. This
	// will cause the first page to be pruned. The next time we append we'll
	// put it in the middle of the circ buffer. We assert this so that if
	// our implementation changes our test will fail.
	_ = sw1Append(t, w, node)
	_ = sw1Append(t, w, node)
	{
		// No wrap around yet
		slices := w.data.GetPtrSlice(0, w.data.Len())
		expect(t, len(slices[0]) > 0, "slices[0].len > 0")
		expect(t, len(slices[1]) == 0, "slices[1].len == 0")
	}

	// Search non-match, prunes page
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 1, w.meta.Len())

	// Change the needle, just needs to be the same length (not a real API)
	// testChangeNeedle doesn't reverse, so pass reversed needle for reverse mode
	w.testChangeNeedle([]uint8("!oob"))

	// Add new page, now wraps
	_ = sw1Append(t, w, node)
	{
		slices := w.data.GetPtrSlice(0, w.data.Len())
		expect(t, len(slices[0]) > 0, "slices[0].len > 0")
		expect(t, len(slices[1]) > 0, "slices[1].len > 0")
	}
	sw1ExpectNext(t, &s.pages, w, 21, 0, 1, 0)
	expect(t, !w.Next().Valid, "w.next() == null")
}

// Zig: test "SlidingWindow single append soft wrapped"
func TestSw1_SlidingWindow_single_append_soft_wrapped(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("boo!"))
	defer w.Deinit()

	term, err := TerminalInit(terminalOptionsDefault(4, 5))
	zigTry(t, err)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("A\r\nxxboo!\r\nC")))

	// We want to test single-page cases.
	screen := term.screens.active
	expect(t, screen.pages.pages.first == screen.pages.pages.last, "screen.pages.pages.first == screen.pages.pages.last")
	node := sw1Unwrap(t, screen.pages.pages.first)
	_ = sw1Append(t, w, node)

	// We should be able to find two matches.
	sw1ExpectNext(t, &screen.pages, w, 2, 1, 1, 2)
	expect(t, !w.Next().Valid, "w.next() == null")
	expect(t, !w.Next().Valid, "w.next() == null")
}

// Zig: test "SlidingWindow single append reversed soft wrapped"
func TestSw1_SlidingWindow_single_append_reversed_soft_wrapped(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionReverse, []uint8("boo!"))
	defer w.Deinit()

	term, err := TerminalInit(terminalOptionsDefault(4, 5))
	zigTry(t, err)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("A\r\nxxboo!\r\nC")))

	// We want to test single-page cases.
	screen := term.screens.active
	expect(t, screen.pages.pages.first == screen.pages.pages.last, "screen.pages.pages.first == screen.pages.pages.last")
	node := sw1Unwrap(t, screen.pages.pages.first)
	_ = sw1Append(t, w, node)

	// We should be able to find two matches.
	sw1ExpectNext(t, &screen.pages, w, 2, 1, 1, 2)
	expect(t, !w.Next().Valid, "w.next() == null")
	expect(t, !w.Next().Valid, "w.next() == null")
}

// This tests a real bug that occurred where a whitespace-only page
// that encodes to zero bytes would crash.
//
// Zig: test "SlidingWindow append whitespace only node"
func TestSw1_SlidingWindow_append_whitespace_only_node(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("x"))
	defer w.Deinit()

	s := sw1Screen(t, 80, 24, 0)
	defer s.Deinit()

	// By setting the empty page to wrap we get a zero-byte page.
	// This is invasive but its otherwise hard to reproduce naturally
	// without creating a slow test.
	node := sw1Unwrap(t, s.pages.pages.first)
	last_row := node.data.GetRow(uint(node.data.size.rows - 1))
	last_row.wrap = true

	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	_ = sw1Append(t, w, node)

	// No matches expected
	expect(t, !w.Next().Valid, "w.next() == null")
}

// ---------------------------------------------------------------------------
// term51: additional tests (AK). Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

// Zig: test "term51: SlidingWindow.init: forward copies the needle and sizes the buffers"
func TestSw1_term51_SlidingWindow_init_forward_copies_the_needle_and_sizes_the_buffers(t *testing.T) {
	testStdCapacity(t)

	needle := []uint8("Boo!")
	w := sw1Init(t, searchSlidingWindowDirectionForward, needle)
	defer w.Deinit()

	expectEqualStrings(t, "Boo!", string(w.needle))
	expect(t, &w.needle[0] != &needle[0], "w.needle.ptr != needle.ptr")
	expectEqual(t, searchSlidingWindowDirectionForward, w.direction)
	expectEqual(t, 8, len(w.overlap_buf))
	expectEqual(t, 0, w.data_offset)

	// Every buffer starts with zero capacity. A zero-capacity CircBuf
	// reports itself as full.
	expectEqual(t, 0, w.data.Len())
	expectEqual(t, 0, w.data.Capacity())
	expect(t, w.data.full, "w.data.full")
	expectEqual(t, 0, w.meta.Len())
	expectEqual(t, 0, w.meta.Capacity())
	expectEqual(t, 0, len(w.chunk_buf))
	expectEqual(t, 0, cap(w.chunk_buf))
}

// Zig: test "term51: SlidingWindow.init: reverse stores the needle byte-reversed"
func TestSw1_term51_SlidingWindow_init_reverse_stores_the_needle_byte_reversed(t *testing.T) {
	testStdCapacity(t)

	{
		w := sw1Init(t, searchSlidingWindowDirectionReverse, []uint8("Boo!"))
		defer w.Deinit()
		expectEqualStrings(t, "!ooB", string(w.needle))
		expectEqual(t, searchSlidingWindowDirectionReverse, w.direction)
		expectEqual(t, 8, len(w.overlap_buf))
	}

	// The reversal is bytewise, so a multi-byte character is mangled.
	{
		w := sw1Init(t, searchSlidingWindowDirectionReverse, []uint8("aé"))
		defer w.Deinit()
		expectEqualSlices(t, []uint8{0xa9, 0xc3, 'a'}, w.needle)
		expectEqual(t, 6, len(w.overlap_buf))
	}

	// A single byte needle is unchanged.
	{
		w := sw1Init(t, searchSlidingWindowDirectionReverse, []uint8("x"))
		defer w.Deinit()
		expectEqualStrings(t, "x", string(w.needle))
		expectEqual(t, 2, len(w.overlap_buf))
	}
}

// Zig: test "term51: SlidingWindow.init: empty needle is accepted"
func TestSw1_term51_SlidingWindow_init_empty_needle_is_accepted(t *testing.T) {
	testStdCapacity(t)

	// An empty needle is not rejected by init. (next() on such a window
	// would underflow `needle.len - 1` / `start + len - 1`, which panics
	// in safe builds, so it can't be exercised.)
	for _, dir := range sw1Dirs {
		func() {
			w := sw1Init(t, dir, []uint8(""))
			defer w.Deinit()
			expectEqual(t, 0, len(w.needle))
			expectEqual(t, 0, len(w.overlap_buf))
			expectEqual(t, 0, w.data.Len())
			expectEqual(t, 0, w.meta.Len())
		}()
	}

	// Appending still works with an empty needle.
	pl := sw1AKTwo(t, [2]string{"ab", "cd"}, [2]bool{true, false})
	defer pl.Deinit()
	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8(""))
	defer w.Deinit()
	expectEqual(t, 2, sw1Append(t, w, sw1AKNodeAt(t, pl, 0)))
	expectEqual(t, 3, sw1Append(t, w, sw1AKNodeAt(t, pl, 1)))
	sw1AKExpectData(t, w, "abcd\n")
}

// Zig-only: the test "term51: SlidingWindow.init: out of memory unwinds every
// allocation" (FailingAllocator fault injection) is not translated; see
// scripts/not-translated.txt.

// Zig: test "term51: SlidingWindow.deinit: frees metas and buffers after pruning and wrapping"
func TestSw1_term51_SlidingWindow_deinit_frees_metas_and_buffers_after_pruning_and_wrapping(t *testing.T) {
	testStdCapacity(t)

	pl := sw1AKThree(t, [3]string{"pppppp", "xxab", "cdq"}, [3]bool{true, true, true})
	defer pl.Deinit()

	// Zig-only: The testing allocator reports any leak of the cell maps,
	// buffers, needle or overlap buffer.
	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("abcd"))
	defer w.Deinit()
	_ = sw1Append(t, w, sw1AKNodeAt(t, pl, 0))
	_ = sw1Append(t, w, sw1AKNodeAt(t, pl, 1))
	expect(t, !w.Next().Valid, "w.next() == null")
	_ = sw1Append(t, w, sw1AKNodeAt(t, pl, 2))
	expectEqual(t, 2, w.meta.Len())
}

// Zig: test "term51: SlidingWindow.deinit: freshly initialized window"
func TestSw1_term51_SlidingWindow_deinit_freshly_initialized_window(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionReverse, []uint8("needle"))
	w.Deinit()
}

// Zig: test "term51: SlidingWindow.Meta.deinit: frees the cell map"
func TestSw1_term51_SlidingWindow_Meta_deinit_frees_the_cell_map(t *testing.T) {
	testStdCapacity(t)

	pl := sw1AKList(t, 1, 0)
	defer pl.Deinit()

	meta := searchSlidingWindowMeta{
		node:     sw1Unwrap(t, pl.pages.first),
		serial:   7,
		cell_map: nil,
	}
	meta.cell_map = append(meta.cell_map, sw1AKXY(1, 2))
	meta.cell_map = append(meta.cell_map, sw1AKXY(3, 4))
	expectEqual(t, 2, len(meta.cell_map))
	expectEqual(t, 7, meta.serial)

	// Zig-only: Leak checking verifies the memory is released.
	meta.Deinit()

	// An empty cell map is fine too.
	empty := searchSlidingWindowMeta{
		node:     sw1Unwrap(t, pl.pages.first),
		serial:   0,
		cell_map: nil,
	}
	empty.Deinit()
}

// Zig: test "term51: SlidingWindow.clearAndRetainCapacity: resets state but keeps capacity"
func TestSw1_term51_SlidingWindow_clearAndRetainCapacity_resets_state_but_keeps_capacity(t *testing.T) {
	testStdCapacity(t)

	pl := sw1AKTwo(t, [2]string{"xxab", "cdyy"}, [2]bool{true, true})
	defer pl.Deinit()
	a := sw1AKNodeAt(t, pl, 0)
	b := sw1AKNodeAt(t, pl, 1)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("abcd"))
	defer w.Deinit()
	_ = sw1Append(t, w, a)
	_ = sw1Append(t, w, b)

	// Consume the match so that data_offset is non-zero.
	expect(t, w.Next().Valid, "w.next() != null")
	expectEqual(t, 3, w.data_offset)

	data_cap := w.data.Capacity()
	meta_cap := w.meta.Capacity()
	chunk_cap := cap(w.chunk_buf)
	expectEqual(t, 8, data_cap)
	expectEqual(t, 2, meta_cap)
	expect(t, chunk_cap >= 2, "chunk_cap >= 2")

	w.ClearAndRetainCapacity()
	expectEqual(t, 0, w.data_offset)
	expectEqual(t, 0, w.data.Len())
	expectEqual(t, 0, w.meta.Len())
	expectEqual(t, data_cap, w.data.Capacity())
	expectEqual(t, meta_cap, w.meta.Capacity())
	expectEqual(t, chunk_cap, cap(w.chunk_buf))
	expectEqual(t, 0, w.data.head)
	expectEqual(t, 0, w.data.tail)
	expect(t, !w.data.full, "!w.data.full")
	expect(t, !w.meta.full, "!w.meta.full")
	expectEqualStrings(t, "abcd", string(w.needle))

	// Nothing to find in an empty window.
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 0, w.data_offset)

	// Re-use: the very same pages are found again, with no growth.
	_ = sw1Append(t, w, a)
	_ = sw1Append(t, w, b)
	expectEqual(t, data_cap, w.data.Capacity())
	expectEqual(t, meta_cap, w.meta.Capacity())
	h := zigUnwrap(t, w.Next())
	sw1AKExpectHl(t, h, 2, 1, []sw1AKChunk{
		{node: a, start: 0, end: a.data.size.rows},
		{node: b, start: 0, end: 1},
	})
	expect(t, !w.Next().Valid, "w.next() == null")
}

// Zig: test "term51: SlidingWindow.clearAndRetainCapacity: empty window and repeated clears"
func TestSw1_term51_SlidingWindow_clearAndRetainCapacity_empty_window_and_repeated_clears(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("abc"))
	defer w.Deinit()

	w.ClearAndRetainCapacity()
	w.ClearAndRetainCapacity()
	expectEqual(t, 0, w.data_offset)
	expectEqual(t, 0, w.data.Len())
	expectEqual(t, 0, w.meta.Len())
	expectEqual(t, 0, w.data.Capacity())
	// Clearing a zero capacity buffer un-fulls it (head == tail == 0).
	expect(t, !w.data.full, "!w.data.full")
	expect(t, w.data.Empty(), "w.data.empty()")

	pl := sw1AKTwo(t, [2]string{"ab", "cd"}, [2]bool{true, true})
	defer pl.Deinit()
	_ = sw1Append(t, w, sw1AKNodeAt(t, pl, 0))
	w.ClearAndRetainCapacity()
	w.ClearAndRetainCapacity()
	expectEqual(t, 0, w.meta.Len())
	expectEqual(t, 2, w.data.Capacity())
}

// Zig: test "term51: SlidingWindow.clearAndRetainCapacity: clears a wrapped circular buffer"
func TestSw1_term51_SlidingWindow_clearAndRetainCapacity_clears_a_wrapped_circular_buffer(t *testing.T) {
	testStdCapacity(t)

	pl := sw1AKThree(t, [3]string{"pppppp", "xxab", "cdq"}, [3]bool{true, true, true})
	defer pl.Deinit()

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("abcd"))
	defer w.Deinit()
	_ = sw1Append(t, w, sw1AKNodeAt(t, pl, 0))
	_ = sw1Append(t, w, sw1AKNodeAt(t, pl, 1))
	expect(t, !w.Next().Valid, "w.next() == null")
	_ = sw1Append(t, w, sw1AKNodeAt(t, pl, 2))
	expect(t, len(w.data.GetPtrSlice(0, w.data.Len())[1]) > 0, "w.data.getPtrSlice(0, w.data.len())[1].len > 0")

	w.ClearAndRetainCapacity()
	expectEqual(t, 0, w.data.head)
	expectEqual(t, 0, w.data.tail)
	expectEqual(t, 10, w.data.Capacity())

	// After clearing there is no wrap: the data is contiguous again.
	_ = sw1Append(t, w, sw1AKNodeAt(t, pl, 1))
	_ = sw1Append(t, w, sw1AKNodeAt(t, pl, 2))
	slices := w.data.GetPtrSlice(0, w.data.Len())
	expectEqual(t, 7, len(slices[0]))
	expectEqual(t, 0, len(slices[1]))
	expectEqualStrings(t, "xxabcdq", string(slices[0]))
	expect(t, w.Next().Valid, "w.next() != null")
}

// Zig: test "term51: SlidingWindow.append: encodes a page and records cell coordinates"
func TestSw1_term51_SlidingWindow_append_encodes_a_page_and_records_cell_coordinates(t *testing.T) {
	testStdCapacity(t)

	pl := sw1AKList(t, 1, 0)
	defer pl.Deinit()
	node := sw1Unwrap(t, pl.pages.first)
	sw1AKPut(node, 0, 0, "hello")

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("lo"))
	defer w.Deinit()

	// The last row isn't soft-wrapped so a trailing newline is added.
	// That newline maps to the last cell of the text.
	expectEqual(t, 6, sw1Append(t, w, node))
	sw1AKExpectData(t, w, "hello\n")
	expectEqual(t, 1, w.meta.Len())
	expectEqual(t, 6, w.data.Len())
	expectEqual(t, node, sw1AKMeta(t, w, 0).node)
	expectEqual(t, node.serial, sw1AKMeta(t, w, 0).serial)
	sw1AKExpectMap(t, w, 0, []PointCoordinate{
		sw1AKXY(0, 0), sw1AKXY(1, 0), sw1AKXY(2, 0),
		sw1AKXY(3, 0), sw1AKXY(4, 0), sw1AKXY(4, 0),
	})
	expectEqual(t, 0, w.data_offset)
	expectEqual(t, 0, len(w.chunk_buf))
}

// Zig: test "term51: SlidingWindow.append: soft-wrapped last row adds no newline"
func TestSw1_term51_SlidingWindow_append_soft_wrapped_last_row_adds_no_newline(t *testing.T) {
	testStdCapacity(t)

	pl := sw1AKList(t, 1, 0)
	defer pl.Deinit()
	node := sw1Unwrap(t, pl.pages.first)
	sw1AKPut(node, 0, 0, "hello")
	sw1AKSetWrap(node, true)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("lo"))
	defer w.Deinit()
	expectEqual(t, 5, sw1Append(t, w, node))
	sw1AKExpectData(t, w, "hello")
	sw1AKExpectMap(t, w, 0, []PointCoordinate{
		sw1AKXY(0, 0), sw1AKXY(1, 0), sw1AKXY(2, 0),
		sw1AKXY(3, 0), sw1AKXY(4, 0),
	})
}

// Zig: test "term51: SlidingWindow.append: blank page is a single newline mapped to the origin"
func TestSw1_term51_SlidingWindow_append_blank_page_is_a_single_newline_mapped_to_the_origin(t *testing.T) {
	testStdCapacity(t)

	pl := sw1AKList(t, 1, 0)
	defer pl.Deinit()
	node := sw1Unwrap(t, pl.pages.first)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("x"))
	defer w.Deinit()
	expectEqual(t, 1, sw1Append(t, w, node))
	sw1AKExpectData(t, w, "\n")
	expectEqual(t, 1, w.meta.Len())
	sw1AKExpectMap(t, w, 0, []PointCoordinate{sw1AKXY(0, 0)})
}

// Zig: test "term51: SlidingWindow.append: wrapped blank page adds nothing"
func TestSw1_term51_SlidingWindow_append_wrapped_blank_page_adds_nothing(t *testing.T) {
	testStdCapacity(t)

	pl := sw1AKList(t, 1, 0)
	defer pl.Deinit()
	node := sw1Unwrap(t, pl.pages.first)
	sw1AKSetWrap(node, true)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("x"))
	defer w.Deinit()
	expectEqual(t, 0, sw1Append(t, w, node))
	expectEqual(t, 0, w.data.Len())
	expectEqual(t, 0, w.meta.Len())

	// Nothing was allocated for the window buffers.
	expectEqual(t, 0, w.data.Capacity())
	expectEqual(t, 0, w.meta.Capacity())
	expectEqual(t, 0, cap(w.chunk_buf))
	expect(t, !w.Next().Valid, "w.next() == null")

	// Same in reverse.
	r := sw1Init(t, searchSlidingWindowDirectionReverse, []uint8("x"))
	defer r.Deinit()
	expectEqual(t, 0, sw1Append(t, r, node))
	expectEqual(t, 0, r.meta.Len())
}

// Zig: test "term51: SlidingWindow.append: blank rows become newlines with row coordinates"
func TestSw1_term51_SlidingWindow_append_blank_rows_become_newlines_with_row_coordinates(t *testing.T) {
	testStdCapacity(t)

	// Leading blank rows are emitted as newlines mapped to rows 0, 1, ...
	{
		pl := sw1AKList(t, 1, 0)
		defer pl.Deinit()
		node := sw1Unwrap(t, pl.pages.first)
		sw1AKPut(node, 2, 0, "hi")
		sw1AKSetWrap(node, true)

		w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("hi"))
		defer w.Deinit()
		expectEqual(t, 4, sw1Append(t, w, node))
		sw1AKExpectData(t, w, "\n\nhi")
		sw1AKExpectMap(t, w, 0, []PointCoordinate{
			sw1AKXY(0, 0), sw1AKXY(0, 1),
			sw1AKXY(0, 2), sw1AKXY(1, 2),
		})
	}

	// Blank rows between two lines: the first newline belongs to the last
	// cell of the previous text, the others to their own blank row.
	{
		pl := sw1AKList(t, 1, 0)
		defer pl.Deinit()
		node := sw1Unwrap(t, pl.pages.first)
		sw1AKPut(node, 0, 0, "ab")
		sw1AKPut(node, 3, 0, "cd")
		sw1AKSetWrap(node, true)

		w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("hi"))
		defer w.Deinit()
		expectEqual(t, 7, sw1Append(t, w, node))
		sw1AKExpectData(t, w, "ab\n\n\ncd")
		sw1AKExpectMap(t, w, 0, []PointCoordinate{
			sw1AKXY(0, 0), sw1AKXY(1, 0), sw1AKXY(1, 0),
			sw1AKXY(0, 1), sw1AKXY(0, 2), sw1AKXY(0, 3),
			sw1AKXY(1, 3),
		})
	}
}

// Zig: test "term51: SlidingWindow.append: blank cells and trailing blanks"
func TestSw1_term51_SlidingWindow_append_blank_cells_and_trailing_blanks(t *testing.T) {
	testStdCapacity(t)

	pl := sw1AKList(t, 1, 0)
	defer pl.Deinit()
	node := sw1Unwrap(t, pl.pages.first)

	// "a" at x=0, "b" at x=3 (two empty cells between), a trailing explicit
	// space and then empty cells.
	sw1AKPut(node, 0, 0, "a")
	sw1AKPut(node, 0, 3, "b ")
	sw1AKSetWrap(node, true)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("a"))
	defer w.Deinit()
	expectEqual(t, 4, sw1Append(t, w, node))
	sw1AKExpectData(t, w, "a  b")

	// Upstream quirk: the cells of the interior blanks are mapped in
	// descending x order (x=2 then x=1).
	sw1AKExpectMap(t, w, 0, []PointCoordinate{
		sw1AKXY(0, 0), sw1AKXY(2, 0), sw1AKXY(1, 0), sw1AKXY(3, 0),
	})
}

// Zig: test "term51: SlidingWindow.append: multi-byte characters map every byte to its cell"
func TestSw1_term51_SlidingWindow_append_multi_byte_characters_map_every_byte_to_its_cell(t *testing.T) {
	testStdCapacity(t)

	s := sw1Screen(t, 10, 5, 0)
	defer s.Deinit()
	sw1Write(t, s, "aé世\U0001F600b")
	node := sw1Unwrap(t, s.pages.pages.first)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("b"))
	defer w.Deinit()

	// The count is bytes: 1 + 2 + 3 + 4 + 1 plus the trailing newline.
	expectEqual(t, 12, sw1Append(t, w, node))
	sw1AKExpectData(t, w, "aé世\U0001F600b\n")
	// a=x0, e-acute=x1, CJK wide at x2 (tail x3), emoji wide at x4 (tail x5),
	// b at x6. The spacer tails are skipped.
	sw1AKExpectMap(t, w, 0, []PointCoordinate{
		sw1AKXY(0, 0),
		sw1AKXY(1, 0),
		sw1AKXY(1, 0),
		sw1AKXY(2, 0),
		sw1AKXY(2, 0),
		sw1AKXY(2, 0),
		sw1AKXY(4, 0),
		sw1AKXY(4, 0),
		sw1AKXY(4, 0),
		sw1AKXY(4, 0),
		sw1AKXY(6, 0),
		sw1AKXY(6, 0),
	})
}

// Zig: test "term51: SlidingWindow.append: grapheme clusters map all codepoints to one cell"
func TestSw1_term51_SlidingWindow_append_grapheme_clusters_map_all_codepoints_to_one_cell(t *testing.T) {
	testStdCapacity(t)

	s := sw1Screen(t, 10, 5, 0)
	defer s.Deinit()
	sw1Write(t, s, "aéb")
	node := sw1Unwrap(t, s.pages.pages.first)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("b"))
	defer w.Deinit()
	expectEqual(t, 6, sw1Append(t, w, node))
	sw1AKExpectData(t, w, "aéb\n")
	sw1AKExpectMap(t, w, 0, []PointCoordinate{
		sw1AKXY(0, 0),
		sw1AKXY(1, 0),
		sw1AKXY(1, 0),
		sw1AKXY(1, 0),
		sw1AKXY(2, 0),
		sw1AKXY(2, 0),
	})
}

// Zig: test "term51: SlidingWindow.append: wide character split by a spacer head"
func TestSw1_term51_SlidingWindow_append_wide_character_split_by_a_spacer_head(t *testing.T) {
	testStdCapacity(t)

	s := sw1Screen(t, 4, 5, 0)
	defer s.Deinit()
	// The wide character doesn't fit in the last column: a spacer head
	// is placed there and the row is soft-wrapped.
	sw1Write(t, s, "abc世d")
	node := sw1Unwrap(t, s.pages.pages.first)
	expect(t, node.data.GetRow(0).wrap, "node.data.getRow(0).wrap")

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("b"))
	defer w.Deinit()
	expectEqual(t, 8, sw1Append(t, w, node))
	sw1AKExpectData(t, w, "abc世d\n")
	// The spacer head (x=3, y=0) contributes no byte at all.
	sw1AKExpectMap(t, w, 0, []PointCoordinate{
		sw1AKXY(0, 0), sw1AKXY(1, 0), sw1AKXY(2, 0),
		sw1AKXY(0, 1), sw1AKXY(0, 1), sw1AKXY(0, 1),
		sw1AKXY(2, 1), sw1AKXY(2, 1),
	})
}

// Zig: test "term51: SlidingWindow.append: grows buffers to exactly the needed size"
func TestSw1_term51_SlidingWindow_append_grows_buffers_to_exactly_the_needed_size(t *testing.T) {
	testStdCapacity(t)

	pl := sw1AKThree(t, [3]string{"hello", "wide", "!"}, [3]bool{true, true, false})
	defer pl.Deinit()

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("x"))
	defer w.Deinit()

	expectEqual(t, 5, sw1Append(t, w, sw1AKNodeAt(t, pl, 0)))
	expectEqual(t, 5, w.data.Capacity())
	expect(t, w.data.full, "w.data.full")
	expectEqual(t, 1, w.meta.Capacity())
	expect(t, w.meta.full, "w.meta.full")
	expect(t, uint(cap(w.chunk_buf)) >= w.meta.Capacity(), "w.chunk_buf.capacity >= w.meta.capacity()")

	expectEqual(t, 4, sw1Append(t, w, sw1AKNodeAt(t, pl, 1)))
	expectEqual(t, 9, w.data.Capacity())
	expectEqual(t, 2, w.meta.Capacity())
	expect(t, uint(cap(w.chunk_buf)) >= w.meta.Capacity(), "w.chunk_buf.capacity >= w.meta.capacity()")

	expectEqual(t, 2, sw1Append(t, w, sw1AKNodeAt(t, pl, 2)))
	expectEqual(t, 11, w.data.Capacity())
	expectEqual(t, 3, w.meta.Capacity())
	expect(t, uint(cap(w.chunk_buf)) >= w.meta.Capacity(), "w.chunk_buf.capacity >= w.meta.capacity()")
	expectEqual(t, 3, w.meta.Len())
	sw1AKExpectData(t, w, "hellowide!\n")

	// Appending never prunes: the data offset is untouched.
	expectEqual(t, 0, w.data_offset)
}

// Zig: test "term51: SlidingWindow.append: reverse reverses the data and the cell map"
func TestSw1_term51_SlidingWindow_append_reverse_reverses_the_data_and_the_cell_map(t *testing.T) {
	testStdCapacity(t)

	pl := sw1AKTwo(t, [2]string{"ab", "cd"}, [2]bool{true, false})
	defer pl.Deinit()

	w := sw1Init(t, searchSlidingWindowDirectionReverse, []uint8("x"))
	defer w.Deinit()

	// Pages are appended last page first for a reverse search.
	expectEqual(t, 3, sw1Append(t, w, sw1AKNodeAt(t, pl, 1)))
	expectEqual(t, 2, sw1Append(t, w, sw1AKNodeAt(t, pl, 0)))
	sw1AKExpectData(t, w, "\ndcba")
	sw1AKExpectMap(t, w, 0, []PointCoordinate{
		sw1AKXY(1, 0), sw1AKXY(1, 0), sw1AKXY(0, 0),
	})
	sw1AKExpectMap(t, w, 1, []PointCoordinate{
		sw1AKXY(1, 0), sw1AKXY(0, 0),
	})
	expectEqual(t, sw1AKNodeAt(t, pl, 1), sw1AKMeta(t, w, 0).node)
	expectEqual(t, sw1AKNodeAt(t, pl, 0), sw1AKMeta(t, w, 1).node)
}

// ---------------------------------------------------------------------------
// Go-only tests.
// ---------------------------------------------------------------------------

// Go-only: next: pruning leaves exactly needle.len - 1 bytes. In Zig the last
// statement of `next`, `self.data_offset = self.data.len() - self.needle.len +
// 1`, underflows when the data holds needle.len - 1 bytes (a panic in a safe
// build, untestable in Zig). The Go port computes len + 1 - needle.len, which
// is what a release build produces (the wrapped subtraction plus one): 0. The
// search then simply continues from offset 0 of the remaining data.
func TestSw1_next_prune_leaves_exactly_needle_len_minus_1_bytes(t *testing.T) {
	testStdCapacity(t)

	// "pppp" (4 bytes), "xab" (3 bytes) and "cdq" (3 bytes), all soft
	// wrapped so each page encodes to exactly its text. The needle is
	// "abcd" (4 bytes): the first two pages hold "ppppxab", which has no
	// match.
	pl := sw1AKThree(t, [3]string{"pppp", "xab", "cdq"}, [3]bool{true, true, true})
	defer pl.Deinit()
	a := sw1AKNodeAt(t, pl, 0)
	b := sw1AKNodeAt(t, pl, 1)
	c := sw1AKNodeAt(t, pl, 2)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("abcd"))
	defer w.Deinit()
	_ = sw1Append(t, w, a)
	_ = sw1Append(t, w, b)
	expect(t, !w.Next().Valid, "w.next() == null")

	// The pruning keeps needle.len - 1 = 3 bytes, exactly the second page:
	// the first page is dropped and the offset is 0, not an underflow.
	expectEqual(t, 1, w.meta.Len())
	expectEqual(t, 3, w.data.Len())
	expectEqual(t, 0, w.data_offset)
	sw1AKExpectData(t, w, "xab")

	// Fewer bytes than the needle: no match possible.
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 0, w.data_offset)

	// The next page continues the search: "xab" + "cdq" holds "abcd" at
	// index 1, across the page boundary.
	_ = sw1Append(t, w, c)
	h := zigUnwrap(t, w.Next())
	sw1AKExpectHl(t, h, 1, 1, []sw1AKChunk{
		{node: b, start: 0, end: b.data.size.rows},
		{node: c, start: 0, end: 1},
	})
	expectEqual(t, 2, w.data_offset)
	expect(t, !w.Next().Valid, "w.next() == null")
}

// Go-only: panic test: testChangeNeedle: the new needle has a different length
// (`assert(new.len == self.needle.len)`)
func TestSw1_panic_testChangeNeedle_new_needle_has_a_different_length(t *testing.T) {
	testStdCapacity(t)

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("abc"))
	defer w.Deinit()
	sw1ExpectPanic(t, func() { w.testChangeNeedle([]uint8("ab")) })
}

// Go-only: panic test: highlight: the start index is not within the data
// (`assert(start < self.data.len())`)
func TestSw1_panic_highlight_start_beyond_the_data(t *testing.T) {
	testStdCapacity(t)

	pl := sw1AKList(t, 1, 0)
	defer pl.Deinit()
	node := sw1Unwrap(t, pl.pages.first)
	sw1AKPut(node, 0, 0, "hello")

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("lo"))
	defer w.Deinit()

	// The window is empty: no index is within the data.
	sw1ExpectPanic(t, func() { _ = w.highlight(0, 2) })

	// "hello\n" is 6 bytes: index 6 is one past the end.
	expectEqual(t, 6, sw1Append(t, w, node))
	sw1ExpectPanic(t, func() { _ = w.highlight(6, 1) })
}

// Go-only: panic test: highlight: the match extends beyond the data
// (`assert(start + len <= self.data.len())`)
func TestSw1_panic_highlight_match_extends_beyond_the_data(t *testing.T) {
	testStdCapacity(t)

	pl := sw1AKList(t, 1, 0)
	defer pl.Deinit()
	node := sw1Unwrap(t, pl.pages.first)
	sw1AKPut(node, 0, 0, "hello")

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("lo"))
	defer w.Deinit()
	expectEqual(t, 6, sw1Append(t, w, node))

	// Start 4 is within the 6 bytes, but 4 + 4 is not.
	sw1ExpectPanic(t, func() { _ = w.highlight(4, 4) })
}

// Go-only: panic test: assertIntegrity: the data length differs from the sum
// of the cell map lengths (`assert(data_len == self.data.len())`)
func TestSw1_panic_assertIntegrity_data_length_differs_from_the_cell_maps(t *testing.T) {
	testStdCapacity(t)

	pl := sw1AKList(t, 1, 0)
	defer pl.Deinit()
	node := sw1Unwrap(t, pl.pages.first)
	sw1AKPut(node, 0, 0, "hello")

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("lo"))
	defer w.Deinit()
	expectEqual(t, 6, sw1Append(t, w, node))

	// A consistent window passes.
	w.assertIntegrity()

	// Drop one byte of data but keep its cell map entry.
	w.data.DeleteOldest(1)
	sw1ExpectPanic(t, func() { w.assertIntegrity() })
}

// Go-only: panic test: assertIntegrity: the data offset is out of bounds
// (`assert(self.data.len() == 0 or self.data_offset < self.data.len())`)
func TestSw1_panic_assertIntegrity_data_offset_out_of_bounds(t *testing.T) {
	testStdCapacity(t)

	pl := sw1AKList(t, 1, 0)
	defer pl.Deinit()
	node := sw1Unwrap(t, pl.pages.first)
	sw1AKPut(node, 0, 0, "hello")

	w := sw1Init(t, searchSlidingWindowDirectionForward, []uint8("lo"))
	defer w.Deinit()
	expectEqual(t, 6, sw1Append(t, w, node))

	// The last byte is the highest valid offset; an offset at (or past)
	// the end of the non-empty data is not accepted.
	w.data_offset = w.data.Len() - 1
	w.assertIntegrity()
	w.data_offset = w.data.Len()
	sw1ExpectPanic(t, func() { w.assertIntegrity() })
}
