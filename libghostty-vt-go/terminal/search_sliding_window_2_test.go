// Tests translated one for one from src/terminal/search/sliding_window.zig
// (Ghostty v1.3.1 plus the term51 additional tests), second part: the tests
// from "term51: SlidingWindow.append: forward order concatenates pages"
// (line 2424) to the end of the file (tests 47 to 90). Each Go test carries
// the exact Zig title in a "// Zig: test" comment. Helpers are prefixed sw2.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualStrings keep Zig's names and Zig's (expected,
//     actual) argument order. `try` on a call that returns an error is
//     zigTry(t, err) (inside the helpers sw2Init, sw2Append, sw2List, ...),
//     `x.?` is zigUnwrap(t, x) for an Optional and a plain pointer
//     otherwise.
//   - The `testing.allocator` / `alloc` arguments are dropped. Every window
//     is made with SearchSlidingWindowInit through sw2Init (a literal would
//     be a zero-size ring buffer that is not full), and every page list with
//     PageListInit through sw2List (term51AKList), sw2Many, sw2Two and
//     sw2Three (term51AKMany, term51AKTwo, term51AKThree).
//   - The helpers term51AKXY, term51AKExpectMap and term51AKAppendOom are
//     not used by the tests of this part and are not translated here.
//   - `inline for (.{ forward, reverse }) |dir|` is a range loop over the
//     two directions. Zig's `defer` inside a loop or block body runs at the
//     end of the iteration or block; Go's defer runs when the test function
//     returns.
//   - A flattened highlight's `chunks` (a MultiArrayList in Zig) is a slice
//     of HighlightFlattenedChunk: `h.chunks.items(.serial)[i]` is
//     `h.chunks[i].serial`, `h.chunks.len` is `len(h.chunks)`. The raw
//     pointer comparison `h1.chunks.bytes == w.chunk_buf.bytes` is a
//     comparison of the address of the first element of the two slices.
//   - The two "out of memory leaves the window empty" tests are not
//     translated (std.testing.FailingAllocator): see
//     scripts/not-translated.txt.
//   - The Go-only tests at the end cover the intentional difference in the
//     underflow `data.len() - needle.len + 1` of `next` (the Go port
//     computes `len + 1 - needle.len`) and the assert / unreachable paths
//     that Zig's tests cannot reach (`TestSw2_panic_...`).

package terminal

import (
	"testing"
)

// sw2Cols is term51AKCols: the column count of every page list built by
// sw2List.
const sw2Cols = 200

// sw2Chunk is term51AKChunk: the expected y bounds of one flattened
// highlight chunk.
type sw2Chunk struct {
	node  *PageListNode
	start SizeCellCountInt
	end   SizeCellCountInt
}

// sw2Directions is `.{ SlidingWindow.Direction.forward,
// SlidingWindow.Direction.reverse }`, the tuple of the `inline for` loops.
var sw2Directions = []searchSlidingWindowDirection{
	searchSlidingWindowDirectionForward,
	searchSlidingWindowDirectionReverse,
}

// sw2ExpectPanic runs f and fails the test if f does not panic.
func sw2ExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if recover() == nil {
			t.Fatalf("expected a panic, got none")
		}
	}()
	f()
}

// sw2Init is `try SlidingWindow.init(alloc, dir, needle)`.
func sw2Init(t *testing.T, dir searchSlidingWindowDirection, needle string) *SearchSlidingWindow {
	t.Helper()
	w, err := SearchSlidingWindowInit(dir, []uint8(needle))
	zigTry(t, err)
	return &w
}

// sw2Append is `try w.append(node)`.
func sw2Append(t *testing.T, w *SearchSlidingWindow, node *PageListNode) uint {
	t.Helper()
	n, err := w.Append(node)
	zigTry(t, err)
	return n
}

// sw2Next is `w.next().?`.
func sw2Next(t *testing.T, w *SearchSlidingWindow) HighlightFlattened {
	t.Helper()
	return zigUnwrap(t, w.Next())
}

// sw2Screen is `try Screen.init(alloc, .{ .cols = cols, .rows = rows,
// .max_scrollback = 0 })` (kitty_image_storage_limit has its Zig default).
func sw2Screen(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) *Screen {
	t.Helper()
	s, err := ScreenInit(ScreenOptions{
		Cols:                   cols,
		Rows:                   rows,
		MaxScrollback:          0,
		KittyImageStorageLimit: 320 * 1000 * 1000,
	})
	zigTry(t, err)
	return &s
}

// sw2Grow is `_ = try pl.grow()`.
func sw2Grow(t *testing.T, pl *PageList) {
	t.Helper()
	_, err := pl.Grow()
	zigTry(t, err)
}

// sw2List is term51AKList: a page list with exactly `pages` pages (the
// earlier ones are full) and `extra` additional rows grown on the last page
// after it was created.
func sw2List(t *testing.T, pages uint, extra uint) *PageList {
	t.Helper()
	pl, err := PageListInit(sw2Cols, 5, None[uint64]())
	zigTry(t, err)
	pl.PauseIntegrityChecks(true)
	for sw2NodeCount(&pl) < pages {
		sw2Grow(t, &pl)
	}
	for i := uint(0); i < extra; i++ {
		sw2Grow(t, &pl)
	}
	pl.PauseIntegrityChecks(false)
	return &pl
}

// sw2NodeCount is term51AKNodeCount.
func sw2NodeCount(pl *PageList) uint {
	var count uint
	for it := pl.pages.first; it != nil; it = it.next {
		count++
	}
	return count
}

// sw2NodeAt is term51AKNodeAt.
func sw2NodeAt(pl *PageList, i uint) *PageListNode {
	n := pl.pages.first
	for j := uint(0); j < i; j++ {
		n = n.next
	}
	return n
}

// sw2Put is term51AKPut: write narrow ASCII cells into row `y` starting at
// column `x`.
func sw2Put(node *PageListNode, y uint, x uint, text string) {
	row := node.data.GetRow(y)
	cells := node.data.GetCells(row)
	for i := 0; i < len(text); i++ {
		cells[x+uint(i)] = pageCellFields{
			content_tag: PageCellContentTagCodepoint,
			content:     pageCellContentFields{codepoint: uint32(text[i])}.pack(),
		}.pack()
	}
}

// sw2SetWrap is term51AKSetWrap: set the soft-wrap flag of the last row of
// the page.
func sw2SetWrap(node *PageListNode, wrap bool) {
	node.data.GetRow(uint(node.data.size.rows - 1)).wrap = wrap
}

// sw2ExpectData is term51AKExpectData.
func sw2ExpectData(t *testing.T, w *SearchSlidingWindow, expected string) {
	t.Helper()
	expectEqual(t, uint(len(expected)), w.data.Len())
	s := w.data.GetPtrSlice(0, w.data.Len())
	expectEqualStrings(t, expected[0:len(s[0])], string(s[0]))
	expectEqualStrings(t, expected[len(s[0]):], string(s[1]))
}

// sw2Meta is term51AKMeta.
func sw2Meta(t *testing.T, w *SearchSlidingWindow, i uint) *searchSlidingWindowMeta {
	t.Helper()
	it := w.meta.Iterator(DatastructCircBufIteratorDirectionForward)
	it.SeekBy(int64(i))
	m := it.Next()
	if m == nil {
		t.Fatalf("unwrapped a null optional")
	}
	return m
}

// sw2ExpectHl is term51AKExpectHl.
func sw2ExpectHl(
	t *testing.T,
	h HighlightFlattened,
	top_x SizeCellCountInt,
	bot_x SizeCellCountInt,
	want []sw2Chunk,
) {
	t.Helper()
	expectEqual(t, len(want), len(h.Chunks))
	for i, c := range want {
		expectEqual(t, c.node, h.Chunks[i].Node)
		expectEqual(t, c.start, h.Chunks[i].Start)
		expectEqual(t, c.end, h.Chunks[i].End)
	}
	expectEqual(t, top_x, h.TopX)
	expectEqual(t, bot_x, h.BotX)
}

// sw2Many is term51AKMany: a page list with one page per entry of `texts`.
// Each text is written at row 0 of its page, and the last row of every page
// gets the matching soft-wrap flag, so a wrapped page encodes to exactly its
// text and a non-wrapped page to its text plus a newline. All pages except
// the last are full (their last row is far below row 0), the last has one
// row.
func sw2Many(t *testing.T, texts []string, wraps []bool) *PageList {
	t.Helper()
	pl := sw2List(t, uint(len(texts)), 0)
	for i := range texts {
		node := sw2NodeAt(pl, uint(i))
		sw2Put(node, 0, 0, texts[i])
		sw2SetWrap(node, wraps[i])
	}
	return pl
}

// sw2Three is term51AKThree.
func sw2Three(t *testing.T, texts [3]string, wraps [3]bool) *PageList {
	t.Helper()
	return sw2Many(t, texts[:], wraps[:])
}

// sw2Two is term51AKTwo.
func sw2Two(t *testing.T, texts [2]string, wraps [2]bool) *PageList {
	t.Helper()
	return sw2Many(t, texts[:], wraps[:])
}

// sw2Rows is term51AKRows.
func sw2Rows(node *PageListNode) SizeCellCountInt {
	return node.data.size.rows
}

// Zig: test "term51: SlidingWindow.append: forward order concatenates pages"
func TestSw2_SlidingWindow_append_forward_order_concatenates_pages(t *testing.T) {
	testStdCapacity(t)

	pl := sw2Two(t, [2]string{"ab", "cd"}, [2]bool{true, false})
	defer pl.Deinit()

	w := sw2Init(t, searchSlidingWindowDirectionForward, "x")
	defer w.Deinit()
	sw2Append(t, w, sw2NodeAt(pl, 0))
	sw2Append(t, w, sw2NodeAt(pl, 1))
	sw2ExpectData(t, w, "abcd\n")
	expectEqual(t, sw2NodeAt(pl, 0), sw2Meta(t, w, 0).node)
	expectEqual(t, sw2NodeAt(pl, 1), sw2Meta(t, w, 1).node)
	expectEqual(t, sw2NodeAt(pl, 0).serial, sw2Meta(t, w, 0).serial)
	expectEqual(t, sw2NodeAt(pl, 1).serial, sw2Meta(t, w, 1).serial)
	expect(t, sw2Meta(t, w, 0).serial != sw2Meta(t, w, 1).serial, "term51AKMeta(&w, 0).serial != term51AKMeta(&w, 1).serial")
}

// Zig: test "term51: SlidingWindow.append: trailing spaces of a wrapped last row are lost across pages"
func TestSw2_SlidingWindow_append_trailing_spaces_of_a_wrapped_last_row_are_lost_across_pages(t *testing.T) {
	testStdCapacity(t)

	// Page A ends with "ab" followed by an explicit space cell, soft
	// wrapped into page B which starts with "cd". Within a single page
	// the space would be kept ("ab cd"), but every page is encoded with
	// a fresh formatter so the pending blank is dropped at the boundary.
	pl := sw2Two(t, [2]string{"ab ", "cd"}, [2]bool{true, true})
	defer pl.Deinit()
	sw2NodeAt(pl, 1).data.GetRow(0).wrap_continuation = true

	w := sw2Init(t, searchSlidingWindowDirectionForward, "b cd")
	defer w.Deinit()
	expectEqual(t, 2, sw2Append(t, w, sw2NodeAt(pl, 0)))
	expectEqual(t, 2, sw2Append(t, w, sw2NodeAt(pl, 1)))
	sw2ExpectData(t, w, "abcd")
	expect(t, !w.Next().Valid, "w.next() == null")

	// Without the space the match is found, across the page boundary.
	w2 := sw2Init(t, searchSlidingWindowDirectionForward, "bcd")
	defer w2.Deinit()
	sw2Append(t, w2, sw2NodeAt(pl, 0))
	sw2Append(t, w2, sw2NodeAt(pl, 1))
	sw2ExpectHl(t, sw2Next(t, w2), 1, 1, []sw2Chunk{
		{node: sw2NodeAt(pl, 0), start: 0, end: sw2NodeAt(pl, 0).data.size.rows},
		{node: sw2NodeAt(pl, 1), start: 0, end: 1},
	})
}

// Zig: test "term51: SlidingWindow.next: empty window returns null without changing state"
func TestSw2_SlidingWindow_next_empty_window_returns_null_without_changing_state(t *testing.T) {
	testStdCapacity(t)

	for _, dir := range sw2Directions {
		w := sw2Init(t, dir, "abc")
		defer w.Deinit()
		expect(t, !w.Next().Valid, "w.next() == null")
		expect(t, !w.Next().Valid, "w.next() == null")
		expectEqual(t, 0, w.data_offset)
		expectEqual(t, 0, w.data.Len())
		expectEqual(t, 0, w.meta.Len())
		expectEqual(t, 0, w.data.Capacity())
	}
}

// Zig: test "term51: SlidingWindow.next: data shorter than the needle is not searched or pruned"
func TestSw2_SlidingWindow_next_data_shorter_than_the_needle_is_not_searched_or_pruned(t *testing.T) {
	testStdCapacity(t)

	pl := sw2List(t, 1, 0)
	defer pl.Deinit()
	node := pl.pages.first
	sw2Put(node, 0, 0, "ab")

	// Data is "ab\n" (3 bytes).
	w := sw2Init(t, searchSlidingWindowDirectionForward, "abcd")
	defer w.Deinit()
	sw2Append(t, w, node)
	expect(t, !w.Next().Valid, "w.next() == null")
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 3, w.data.Len())
	expectEqual(t, 1, w.meta.Len())
	expectEqual(t, 0, w.data_offset)

	// One byte longer than the data, even though it is a prefix of it.
	w2 := sw2Init(t, searchSlidingWindowDirectionForward, "ab\nx")
	defer w2.Deinit()
	sw2Append(t, w2, node)
	expect(t, !w2.Next().Valid, "w2.next() == null")
	expectEqual(t, 0, w2.data_offset)
	expectEqual(t, 1, w2.meta.Len())
}

// Zig: test "term51: SlidingWindow.next: needle equal to the whole data matches once"
func TestSw2_SlidingWindow_next_needle_equal_to_the_whole_data_matches_once(t *testing.T) {
	testStdCapacity(t)

	pl := sw2List(t, 1, 0)
	defer pl.Deinit()
	node := pl.pages.first
	sw2Put(node, 0, 0, "ab")

	// The newline is part of the haystack, and maps to the last cell.
	w := sw2Init(t, searchSlidingWindowDirectionForward, "ab\n")
	defer w.Deinit()
	sw2Append(t, w, node)
	sw2ExpectHl(t, sw2Next(t, w), 0, 1, []sw2Chunk{{node: node, start: 0, end: 1}})
	expectEqual(t, 1, w.data_offset)
	expect(t, !w.Next().Valid, "w.next() == null")
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 1, w.meta.Len())
	expectEqual(t, 3, w.data.Len())
	expectEqual(t, 1, w.data_offset)

	// A needle that is only the last two bytes.
	w2 := sw2Init(t, searchSlidingWindowDirectionForward, "b\n")
	defer w2.Deinit()
	sw2Append(t, w2, node)
	sw2ExpectHl(t, sw2Next(t, w2), 1, 1, []sw2Chunk{{node: node, start: 0, end: 1}})
	expect(t, !w2.Next().Valid, "w2.next() == null")
}

// Zig: test "term51: SlidingWindow.next: single byte needle at the first and last cell clears the window"
func TestSw2_SlidingWindow_next_single_byte_needle_at_the_first_and_last_cell_clears_the_window(t *testing.T) {
	testStdCapacity(t)

	pl := sw2Three(t, [3]string{"Zbc", "def", "ghZ"}, [3]bool{true, true, true})
	defer pl.Deinit()
	a := sw2NodeAt(pl, 0)
	c := sw2NodeAt(pl, 2)

	w := sw2Init(t, searchSlidingWindowDirectionForward, "Z")
	defer w.Deinit()
	sw2Append(t, w, a)
	sw2Append(t, w, sw2NodeAt(pl, 1))
	sw2Append(t, w, c)

	// First cell of the first page.
	sw2ExpectHl(t, sw2Next(t, w), 0, 0, []sw2Chunk{{node: a, start: 0, end: 1}})
	expectEqual(t, 1, w.data_offset)
	expectEqual(t, 3, w.meta.Len())

	// Last cell of the last page; the earlier pages are pruned.
	sw2ExpectHl(t, sw2Next(t, w), 2, 2, []sw2Chunk{{node: c, start: 0, end: 1}})
	expectEqual(t, 1, w.meta.Len())
	expectEqual(t, 3, w.data.Len())
	// Upstream quirk: the offset is one past the match, which here is
	// one past the end of the data.
	expectEqual(t, 3, w.data_offset)

	// A single byte needle that finds nothing discards the whole window,
	// including the capacity-retaining state.
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 0, w.meta.Len())
	expectEqual(t, 0, w.data.Len())
	expectEqual(t, 0, w.data_offset)
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 0, w.meta.Len())
}

// Zig: test "term51: SlidingWindow.next: single byte needle clears even a window of non-matching pages"
func TestSw2_SlidingWindow_next_single_byte_needle_clears_even_a_window_of_non_matching_pages(t *testing.T) {
	testStdCapacity(t)

	pl := sw2Two(t, [2]string{"abc", "def"}, [2]bool{true, true})
	defer pl.Deinit()

	w := sw2Init(t, searchSlidingWindowDirectionForward, "Z")
	defer w.Deinit()
	sw2Append(t, w, sw2NodeAt(pl, 0))
	sw2Append(t, w, sw2NodeAt(pl, 1))
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 0, w.meta.Len())
	expectEqual(t, 0, w.data.Len())
	// Capacity is retained.
	expectEqual(t, 6, w.data.Capacity())
	expectEqual(t, 2, w.meta.Capacity())

	// The window is reusable.
	sw2Append(t, w, sw2NodeAt(pl, 0))
	expectEqual(t, 1, w.meta.Len())
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 0, w.meta.Len())
}

// Zig: test "term51: SlidingWindow.next: single byte needle reverse first and last"
func TestSw2_SlidingWindow_next_single_byte_needle_reverse_first_and_last(t *testing.T) {
	testStdCapacity(t)

	pl := sw2Three(t, [3]string{"Zbc", "def", "ghZ"}, [3]bool{true, true, true})
	defer pl.Deinit()
	a := sw2NodeAt(pl, 0)
	c := sw2NodeAt(pl, 2)

	w := sw2Init(t, searchSlidingWindowDirectionReverse, "z")
	defer w.Deinit()
	sw2Append(t, w, c)
	sw2Append(t, w, sw2NodeAt(pl, 1))
	sw2Append(t, w, a)

	// The reverse search sees the last cell of the last page first.
	sw2ExpectHl(t, sw2Next(t, w), 2, 2, []sw2Chunk{{node: c, start: 0, end: 1}})
	sw2ExpectHl(t, sw2Next(t, w), 0, 0, []sw2Chunk{{node: a, start: 0, end: 1}})
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 0, w.meta.Len())
}

// Zig: test "term51: SlidingWindow.next: overlapping and adjacent matches"
func TestSw2_SlidingWindow_next_overlapping_and_adjacent_matches(t *testing.T) {
	testStdCapacity(t)

	pl := sw2List(t, 1, 0)
	defer pl.Deinit()
	node := pl.pages.first
	sw2SetWrap(node, true)
	sw2Put(node, 0, 0, "aaaa")

	// "aa" in "aaaa": every start offset is reported, so three matches.
	{
		w := sw2Init(t, searchSlidingWindowDirectionForward, "aa")
		defer w.Deinit()
		sw2Append(t, w, node)
		sw2ExpectHl(t, sw2Next(t, w), 0, 1, []sw2Chunk{{node: node, start: 0, end: 1}})
		expectEqual(t, 1, w.data_offset)
		sw2ExpectHl(t, sw2Next(t, w), 1, 2, []sw2Chunk{{node: node, start: 0, end: 1}})
		expectEqual(t, 2, w.data_offset)
		sw2ExpectHl(t, sw2Next(t, w), 2, 3, []sw2Chunk{{node: node, start: 0, end: 1}})
		expectEqual(t, 3, w.data_offset)
		expect(t, !w.Next().Valid, "w.next() == null")
		expect(t, !w.Next().Valid, "w.next() == null")
		expectEqual(t, 3, w.data_offset)
		expectEqual(t, 1, w.meta.Len())
	}

	// Reverse: same matches from the right, and top/bottom x are swapped
	// back so that top_x is the left end.
	{
		w := sw2Init(t, searchSlidingWindowDirectionReverse, "aa")
		defer w.Deinit()
		sw2Append(t, w, node)
		sw2ExpectHl(t, sw2Next(t, w), 2, 3, []sw2Chunk{{node: node, start: 0, end: 1}})
		sw2ExpectHl(t, sw2Next(t, w), 1, 2, []sw2Chunk{{node: node, start: 0, end: 1}})
		sw2ExpectHl(t, sw2Next(t, w), 0, 1, []sw2Chunk{{node: node, start: 0, end: 1}})
		expect(t, !w.Next().Valid, "w.next() == null")
	}

	// Three-byte needle in five bytes.
	{
		pl2 := sw2List(t, 1, 0)
		defer pl2.Deinit()
		n2 := pl2.pages.first
		sw2SetWrap(n2, true)
		sw2Put(n2, 0, 0, "aaaaa")
		w := sw2Init(t, searchSlidingWindowDirectionForward, "aaa")
		defer w.Deinit()
		sw2Append(t, w, n2)
		sw2ExpectHl(t, sw2Next(t, w), 0, 2, []sw2Chunk{{node: n2, start: 0, end: 1}})
		sw2ExpectHl(t, sw2Next(t, w), 1, 3, []sw2Chunk{{node: n2, start: 0, end: 1}})
		sw2ExpectHl(t, sw2Next(t, w), 2, 4, []sw2Chunk{{node: n2, start: 0, end: 1}})
		expect(t, !w.Next().Valid, "w.next() == null")
	}

	// Adjacent, non-overlapping matches.
	{
		pl2 := sw2List(t, 1, 0)
		defer pl2.Deinit()
		n2 := pl2.pages.first
		sw2SetWrap(n2, true)
		sw2Put(n2, 0, 0, "abab")
		w := sw2Init(t, searchSlidingWindowDirectionForward, "ab")
		defer w.Deinit()
		sw2Append(t, w, n2)
		sw2ExpectHl(t, sw2Next(t, w), 0, 1, []sw2Chunk{{node: n2, start: 0, end: 1}})
		sw2ExpectHl(t, sw2Next(t, w), 2, 3, []sw2Chunk{{node: n2, start: 0, end: 1}})
		expect(t, !w.Next().Valid, "w.next() == null")
	}
}

// Zig: test "term51: SlidingWindow.next: ASCII case folding only"
func TestSw2_SlidingWindow_next_ASCII_case_folding_only(t *testing.T) {
	testStdCapacity(t)

	pl := sw2List(t, 1, 0)
	defer pl.Deinit()
	node := pl.pages.first
	sw2SetWrap(node, true)
	sw2Put(node, 0, 0, "boo Boo bOO")

	for _, dir := range sw2Directions {
		w := sw2Init(t, dir, "BoO")
		defer w.Deinit()
		sw2Append(t, w, node)
		xs := [3]SizeCellCountInt{8, 4, 0}
		if dir == searchSlidingWindowDirectionForward {
			xs = [3]SizeCellCountInt{0, 4, 8}
		}
		for _, x := range xs {
			sw2ExpectHl(t, sw2Next(t, w), x, x+2, []sw2Chunk{{node: node, start: 0, end: 1}})
		}
		expect(t, !w.Next().Valid, "w.next() == null")
	}

	// Non-ASCII bytes are compared exactly: U+00E9 (c3 a9) never matches
	// U+00C9 (c3 89), but the ASCII letters around it still fold.
	s := sw2Screen(t, 20, 5)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("xAéy")))
	n2 := s.Pages.pages.first
	{
		w := sw2Init(t, searchSlidingWindowDirectionForward, "aÉ")
		defer w.Deinit()
		sw2Append(t, w, n2)
		expect(t, !w.Next().Valid, "w.next() == null")
	}
	{
		w := sw2Init(t, searchSlidingWindowDirectionForward, "aéY")
		defer w.Deinit()
		sw2Append(t, w, n2)
		sw2ExpectHl(t, sw2Next(t, w), 1, 3, []sw2Chunk{{node: n2, start: 0, end: 1}})
		expect(t, !w.Next().Valid, "w.next() == null")
	}
}

// Zig: test "term51: SlidingWindow.next: hard line ends enter the haystack as newlines"
func TestSw2_SlidingWindow_next_hard_line_ends_enter_the_haystack_as_newlines(t *testing.T) {
	testStdCapacity(t)

	pl := sw2List(t, 1, 0)
	defer pl.Deinit()
	node := pl.pages.first
	sw2Put(node, 0, 0, "ab")
	sw2Put(node, 1, 0, "cd")
	// Data: "ab\ncd\n"

	// A needle with a newline spans both rows.
	{
		w := sw2Init(t, searchSlidingWindowDirectionForward, "b\nc")
		defer w.Deinit()
		sw2Append(t, w, node)
		sw2ExpectHl(t, sw2Next(t, w), 1, 0, []sw2Chunk{{node: node, start: 0, end: 2}})
		expect(t, !w.Next().Valid, "w.next() == null")
	}

	// Without the newline there is no match across the rows.
	{
		w := sw2Init(t, searchSlidingWindowDirectionForward, "bc")
		defer w.Deinit()
		sw2Append(t, w, node)
		expect(t, !w.Next().Valid, "w.next() == null")
	}

	// A needle of just a newline maps to the last cell of the row it
	// terminates, including the page's own trailing newline.
	{
		w := sw2Init(t, searchSlidingWindowDirectionForward, "\n")
		defer w.Deinit()
		sw2Append(t, w, node)
		sw2ExpectHl(t, sw2Next(t, w), 1, 1, []sw2Chunk{{node: node, start: 0, end: 1}})
		sw2ExpectHl(t, sw2Next(t, w), 1, 1, []sw2Chunk{{node: node, start: 1, end: 2}})
		expect(t, !w.Next().Valid, "w.next() == null")
	}

	// Same, reversed.
	{
		w := sw2Init(t, searchSlidingWindowDirectionReverse, "b\nc")
		defer w.Deinit()
		sw2Append(t, w, node)
		sw2ExpectHl(t, sw2Next(t, w), 1, 0, []sw2Chunk{{node: node, start: 0, end: 2}})
		expect(t, !w.Next().Valid, "w.next() == null")
	}
}

// Zig: test "term51: SlidingWindow.next: several blank rows are several newlines"
func TestSw2_SlidingWindow_next_several_blank_rows_are_several_newlines(t *testing.T) {
	testStdCapacity(t)

	pl := sw2List(t, 1, 0)
	defer pl.Deinit()
	node := pl.pages.first
	sw2Put(node, 0, 0, "ab")
	sw2Put(node, 3, 0, "cd")
	sw2SetWrap(node, true)
	// Data: "ab\n\n\ncd"

	w := sw2Init(t, searchSlidingWindowDirectionForward, "\n\n\n")
	defer w.Deinit()
	sw2Append(t, w, node)
	sw2ExpectHl(t, sw2Next(t, w), 1, 0, []sw2Chunk{{node: node, start: 0, end: 3}})
	expect(t, !w.Next().Valid, "w.next() == null")

	w2 := sw2Init(t, searchSlidingWindowDirectionForward, "b\n\n\nc")
	defer w2.Deinit()
	sw2Append(t, w2, node)
	sw2ExpectHl(t, sw2Next(t, w2), 1, 0, []sw2Chunk{{node: node, start: 0, end: 4}})

	// Four newlines don't exist.
	w3 := sw2Init(t, searchSlidingWindowDirectionForward, "\n\n\n\n")
	defer w3.Deinit()
	sw2Append(t, w3, node)
	expect(t, !w3.Next().Valid, "w3.next() == null")
}

// Zig: test "term51: SlidingWindow.next: soft wrapped rows join, hard line ends do not"
func TestSw2_SlidingWindow_next_soft_wrapped_rows_join_hard_line_ends_do_not(t *testing.T) {
	testStdCapacity(t)

	// Soft wrap: "abcdefgh" in four columns.
	{
		s := sw2Screen(t, 4, 5)
		defer s.Deinit()
		zigTry(t, s.TestWriteString([]uint8("abcdefgh")))
		node := s.Pages.pages.first

		w := sw2Init(t, searchSlidingWindowDirectionForward, "cdef")
		defer w.Deinit()
		sw2Append(t, w, node)
		sw2ExpectHl(t, sw2Next(t, w), 2, 1, []sw2Chunk{{node: node, start: 0, end: 2}})
		expect(t, !w.Next().Valid, "w.next() == null")

		// The reverse search of the same text.
		r := sw2Init(t, searchSlidingWindowDirectionReverse, "cdef")
		defer r.Deinit()
		sw2Append(t, r, node)
		sw2ExpectHl(t, sw2Next(t, r), 2, 1, []sw2Chunk{{node: node, start: 0, end: 2}})
		expect(t, !r.Next().Valid, "r.next() == null")
	}

	// Hard line end after "abcd": no wrap flag, so the newline is part of
	// the haystack ("abcd\nefgh\n").
	{
		s := sw2Screen(t, 4, 5)
		defer s.Deinit()
		zigTry(t, s.TestWriteString([]uint8("abcd\nefgh")))
		node := s.Pages.pages.first
		expect(t, !node.data.GetRow(0).wrap, "!node.data.getRow(0).wrap")

		w := sw2Init(t, searchSlidingWindowDirectionForward, "cdef")
		defer w.Deinit()
		sw2Append(t, w, node)
		expect(t, !w.Next().Valid, "w.next() == null")

		w2 := sw2Init(t, searchSlidingWindowDirectionForward, "d\ne")
		defer w2.Deinit()
		sw2Append(t, w2, node)
		sw2ExpectHl(t, sw2Next(t, w2), 3, 0, []sw2Chunk{{node: node, start: 0, end: 2}})
		expect(t, !w2.Next().Valid, "w2.next() == null")

		w3 := sw2Init(t, searchSlidingWindowDirectionForward, "abcd\nefgh")
		defer w3.Deinit()
		sw2Append(t, w3, node)
		sw2ExpectHl(t, sw2Next(t, w3), 0, 3, []sw2Chunk{{node: node, start: 0, end: 2}})
	}
}

// Zig: test "term51: SlidingWindow.next: wide characters split across rows by a spacer head"
func TestSw2_SlidingWindow_next_wide_characters_split_across_rows_by_a_spacer_head(t *testing.T) {
	testStdCapacity(t)

	s := sw2Screen(t, 4, 5)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("abc世d")))
	node := s.Pages.pages.first

	for _, dir := range sw2Directions {
		// The spacer head produces no byte, so "c" and the wide character
		// are adjacent in the haystack.
		{
			w := sw2Init(t, dir, "c世")
			defer w.Deinit()
			sw2Append(t, w, node)
			sw2ExpectHl(t, sw2Next(t, w), 2, 0, []sw2Chunk{{node: node, start: 0, end: 2}})
			expect(t, !w.Next().Valid, "w.next() == null")
		}
		// The wide character highlights its head cell (x=0) not the tail.
		{
			w := sw2Init(t, dir, "世")
			defer w.Deinit()
			sw2Append(t, w, node)
			sw2ExpectHl(t, sw2Next(t, w), 0, 0, []sw2Chunk{{node: node, start: 1, end: 2}})
			expect(t, !w.Next().Valid, "w.next() == null")
		}
		{
			w := sw2Init(t, dir, "世d")
			defer w.Deinit()
			sw2Append(t, w, node)
			sw2ExpectHl(t, sw2Next(t, w), 0, 2, []sw2Chunk{{node: node, start: 1, end: 2}})
			expect(t, !w.Next().Valid, "w.next() == null")
		}
	}
}

// Zig: test "term51: SlidingWindow.next: wide characters on one row skip the spacer tail"
func TestSw2_SlidingWindow_next_wide_characters_on_one_row_skip_the_spacer_tail(t *testing.T) {
	testStdCapacity(t)

	s := sw2Screen(t, 10, 5)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("a世b")))
	node := s.Pages.pages.first

	w := sw2Init(t, searchSlidingWindowDirectionForward, "世b")
	defer w.Deinit()
	sw2Append(t, w, node)
	// a=x0, wide=x1 (tail x2), b=x3
	sw2ExpectHl(t, sw2Next(t, w), 1, 3, []sw2Chunk{{node: node, start: 0, end: 1}})
	expect(t, !w.Next().Valid, "w.next() == null")

	w2 := sw2Init(t, searchSlidingWindowDirectionForward, "a世")
	defer w2.Deinit()
	sw2Append(t, w2, node)
	sw2ExpectHl(t, sw2Next(t, w2), 0, 1, []sw2Chunk{{node: node, start: 0, end: 1}})
}

// Zig: test "term51: SlidingWindow.next: grapheme clusters are searched bytewise"
func TestSw2_SlidingWindow_next_grapheme_clusters_are_searched_bytewise(t *testing.T) {
	testStdCapacity(t)

	s := sw2Screen(t, 10, 5)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("aéb")))
	node := s.Pages.pages.first

	// The whole cluster plus a neighbour.
	{
		w := sw2Init(t, searchSlidingWindowDirectionForward, "éb")
		defer w.Deinit()
		sw2Append(t, w, node)
		sw2ExpectHl(t, sw2Next(t, w), 1, 2, []sw2Chunk{{node: node, start: 0, end: 1}})
		expect(t, !w.Next().Valid, "w.next() == null")
	}
	// The combining mark alone matches, on its base cell.
	{
		w := sw2Init(t, searchSlidingWindowDirectionForward, "́")
		defer w.Deinit()
		sw2Append(t, w, node)
		sw2ExpectHl(t, sw2Next(t, w), 1, 1, []sw2Chunk{{node: node, start: 0, end: 1}})
		expect(t, !w.Next().Valid, "w.next() == null")
	}
	// The base letter alone also matches although it is part of a
	// cluster (the search is not grapheme aware).
	{
		w := sw2Init(t, searchSlidingWindowDirectionForward, "e")
		defer w.Deinit()
		sw2Append(t, w, node)
		sw2ExpectHl(t, sw2Next(t, w), 1, 1, []sw2Chunk{{node: node, start: 0, end: 1}})
		expect(t, !w.Next().Valid, "w.next() == null")
	}
	// Precomposed U+00E9 is a different byte sequence, no match.
	{
		w := sw2Init(t, searchSlidingWindowDirectionForward, "é")
		defer w.Deinit()
		sw2Append(t, w, node)
		expect(t, !w.Next().Valid, "w.next() == null")
	}
	// Reverse of the full cluster.
	{
		w := sw2Init(t, searchSlidingWindowDirectionReverse, "éb")
		defer w.Deinit()
		sw2Append(t, w, node)
		sw2ExpectHl(t, sw2Next(t, w), 1, 2, []sw2Chunk{{node: node, start: 0, end: 1}})
		expect(t, !w.Next().Valid, "w.next() == null")
	}
}

// Zig: test "term51: SlidingWindow.next: match across two pages forward"
func TestSw2_SlidingWindow_next_match_across_two_pages_forward(t *testing.T) {
	testStdCapacity(t)

	pl := sw2Two(t, [2]string{"xxhel", "loyy"}, [2]bool{true, false})
	defer pl.Deinit()
	a := sw2NodeAt(pl, 0)
	b := sw2NodeAt(pl, 1)

	w := sw2Init(t, searchSlidingWindowDirectionForward, "hello")
	defer w.Deinit()
	sw2Append(t, w, a)
	sw2Append(t, w, b)

	// Data: "xxhel" + "loyy\n"
	h := sw2Next(t, w)
	sw2ExpectHl(t, h, 2, 1, []sw2Chunk{
		{node: a, start: 0, end: sw2Rows(a)},
		{node: b, start: 0, end: 1},
	})
	expectEqual(t, a.serial, h.Chunks[0].Serial)
	expectEqual(t, b.serial, h.Chunks[1].Serial)
	// The match starts in page A so nothing is pruned.
	expectEqual(t, 2, w.meta.Len())
	expectEqual(t, 10, w.data.Len())
	expectEqual(t, 3, w.data_offset)

	// No more matches; page A can now be pruned since page B alone holds
	// needle.len - 1 bytes.
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 1, w.meta.Len())
	expectEqual(t, 5, w.data.Len())
	expectEqual(t, 1, w.data_offset)
	expect(t, !w.Next().Valid, "w.next() == null")
}

// Zig: test "term51: SlidingWindow.next: match across two pages reverse"
func TestSw2_SlidingWindow_next_match_across_two_pages_reverse(t *testing.T) {
	testStdCapacity(t)

	pl := sw2Two(t, [2]string{"xxhel", "loyy"}, [2]bool{true, false})
	defer pl.Deinit()
	a := sw2NodeAt(pl, 0)
	b := sw2NodeAt(pl, 1)

	w := sw2Init(t, searchSlidingWindowDirectionReverse, "hello")
	defer w.Deinit()
	sw2Append(t, w, b)
	sw2Append(t, w, a)

	// Data: "\nyyol" + "lehxx". The chunks come back in screen order
	// with the same shape as the forward result.
	h := sw2Next(t, w)
	sw2ExpectHl(t, h, 2, 1, []sw2Chunk{
		{node: a, start: 0, end: sw2Rows(a)},
		{node: b, start: 0, end: 1},
	})
	// Upstream defect: the reverse fix-up reverses node/start/end but not
	// serial, so the serials stay in data order (B, A) while the nodes
	// are in screen order (A, B).
	expectEqual(t, b.serial, h.Chunks[0].Serial)
	expectEqual(t, a.serial, h.Chunks[1].Serial)
	expect(t, a.serial != b.serial, "a.serial != b.serial")
	expectEqual(t, 2, w.meta.Len())
	expectEqual(t, 4, w.data_offset)
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 1, w.meta.Len())
	expectEqual(t, 5, w.data.Len())
	expectEqual(t, 1, w.data_offset)
}

// Zig: test "term51: SlidingWindow.next: hard newline between pages is part of the haystack"
func TestSw2_SlidingWindow_next_hard_newline_between_pages_is_part_of_the_haystack(t *testing.T) {
	testStdCapacity(t)

	// Page A's last row isn't soft-wrapped: "xxhel\n" + "lo\n".
	pl := sw2Two(t, [2]string{"xxhel", "lo"}, [2]bool{false, false})
	defer pl.Deinit()
	a := sw2NodeAt(pl, 0)
	b := sw2NodeAt(pl, 1)

	for _, dir := range sw2Directions {
		pages := [2]*PageListNode{b, a}
		if dir == searchSlidingWindowDirectionForward {
			pages = [2]*PageListNode{a, b}
		}
		{
			w := sw2Init(t, dir, "hello")
			defer w.Deinit()
			for _, p := range pages {
				sw2Append(t, w, p)
			}
			expect(t, !w.Next().Valid, "w.next() == null")
		}
		{
			w := sw2Init(t, dir, "hel\nlo")
			defer w.Deinit()
			for _, p := range pages {
				sw2Append(t, w, p)
			}
			sw2ExpectHl(t, sw2Next(t, w), 2, 1, []sw2Chunk{
				{node: a, start: 0, end: sw2Rows(a)},
				{node: b, start: 0, end: 1},
			})
			expect(t, !w.Next().Valid, "w.next() == null")
		}
		// The newline itself maps to the last cell of page A's text.
		{
			w := sw2Init(t, dir, "l\nl")
			defer w.Deinit()
			for _, p := range pages {
				sw2Append(t, w, p)
			}
			sw2ExpectHl(t, sw2Next(t, w), 4, 0, []sw2Chunk{
				{node: a, start: 0, end: sw2Rows(a)},
				{node: b, start: 0, end: 1},
			})
		}
	}
}

// Zig: test "term51: SlidingWindow.next: match across three pages"
func TestSw2_SlidingWindow_next_match_across_three_pages(t *testing.T) {
	testStdCapacity(t)

	pl := sw2Three(t, [3]string{"xxxhe", "ll", "oyyzz"}, [3]bool{true, true, false})
	defer pl.Deinit()
	a := sw2NodeAt(pl, 0)
	b := sw2NodeAt(pl, 1)
	c := sw2NodeAt(pl, 2)

	// Forward: data "xxxhe" + "ll" + "oyyzz\n".
	{
		w := sw2Init(t, searchSlidingWindowDirectionForward, "hello")
		defer w.Deinit()
		sw2Append(t, w, a)
		sw2Append(t, w, b)
		sw2Append(t, w, c)
		h := sw2Next(t, w)
		sw2ExpectHl(t, h, 3, 0, []sw2Chunk{
			{node: a, start: 0, end: sw2Rows(a)},
			{node: b, start: 0, end: sw2Rows(b)},
			{node: c, start: 0, end: 1},
		})
		expectEqual(t, a.serial, h.Chunks[0].Serial)
		expectEqual(t, b.serial, h.Chunks[1].Serial)
		expectEqual(t, c.serial, h.Chunks[2].Serial)
		expectEqual(t, 3, w.meta.Len())
		expectEqual(t, 4, w.data_offset)

		// Exhausting: page C holds the needle.len - 1 bytes to keep, so
		// A and B are pruned.
		expect(t, !w.Next().Valid, "w.next() == null")
		expectEqual(t, 1, w.meta.Len())
		expectEqual(t, 6, w.data.Len())
		expectEqual(t, 2, w.data_offset)
		expect(t, !w.Next().Valid, "w.next() == null")
	}

	// Reverse.
	{
		w := sw2Init(t, searchSlidingWindowDirectionReverse, "hello")
		defer w.Deinit()
		sw2Append(t, w, c)
		sw2Append(t, w, b)
		sw2Append(t, w, a)
		h := sw2Next(t, w)
		sw2ExpectHl(t, h, 3, 0, []sw2Chunk{
			{node: a, start: 0, end: sw2Rows(a)},
			{node: b, start: 0, end: sw2Rows(b)},
			{node: c, start: 0, end: 1},
		})
		// Upstream defect (see the two page test): serials are in data
		// order (C, B, A) while the nodes are in screen order (A, B, C).
		expectEqual(t, c.serial, h.Chunks[0].Serial)
		expectEqual(t, b.serial, h.Chunks[1].Serial)
		expectEqual(t, a.serial, h.Chunks[2].Serial)
		expectEqual(t, 3, w.meta.Len())
		expectEqual(t, 6, w.data_offset)
		expect(t, !w.Next().Valid, "w.next() == null")
		expectEqual(t, 1, w.meta.Len())
		expectEqual(t, 5, w.data.Len())
		expectEqual(t, 1, w.data_offset)
	}
}

// Zig: test "term51: SlidingWindow.next: matches at the very first and last cell of the page list"
func TestSw2_SlidingWindow_next_matches_at_the_very_first_and_last_cell_of_the_page_list(t *testing.T) {
	testStdCapacity(t)

	pl := sw2Three(t, [3]string{"abXX", "mid", "XXyz"}, [3]bool{true, true, true})
	defer pl.Deinit()
	a := sw2NodeAt(pl, 0)
	b := sw2NodeAt(pl, 1)
	c := sw2NodeAt(pl, 2)

	for _, dir := range sw2Directions {
		pages := [3]*PageListNode{c, b, a}
		if dir == searchSlidingWindowDirectionForward {
			pages = [3]*PageListNode{a, b, c}
		}

		{
			w := sw2Init(t, dir, "ab")
			defer w.Deinit()
			for _, p := range pages {
				sw2Append(t, w, p)
			}
			sw2ExpectHl(t, sw2Next(t, w), 0, 1, []sw2Chunk{{node: a, start: 0, end: 1}})
			expect(t, !w.Next().Valid, "w.next() == null")
		}
		{
			w := sw2Init(t, dir, "yz")
			defer w.Deinit()
			for _, p := range pages {
				sw2Append(t, w, p)
			}
			sw2ExpectHl(t, sw2Next(t, w), 2, 3, []sw2Chunk{{node: c, start: 0, end: 1}})
			expect(t, !w.Next().Valid, "w.next() == null")
		}
		// The whole page list at once.
		{
			w := sw2Init(t, dir, "abXXmidXXyz")
			defer w.Deinit()
			for _, p := range pages {
				sw2Append(t, w, p)
			}
			sw2ExpectHl(t, sw2Next(t, w), 0, 3, []sw2Chunk{
				{node: a, start: 0, end: sw2Rows(a)},
				{node: b, start: 0, end: sw2Rows(b)},
				{node: c, start: 0, end: 1},
			})
			expect(t, !w.Next().Valid, "w.next() == null")
		}
	}
}

// Zig: test "term51: SlidingWindow.next: match straddling the last row of a full page and the first row of the next"
func TestSw2_SlidingWindow_next_match_straddling_the_last_row_of_a_full_page_and_the_first_row_of_the_next(t *testing.T) {
	testStdCapacity(t)

	pl := sw2List(t, 2, 0)
	defer pl.Deinit()
	a := sw2NodeAt(pl, 0)
	b := sw2NodeAt(pl, 1)
	last := a.data.size.rows - 1
	sw2Put(a, uint(last), 3, "xxhel")
	sw2SetWrap(a, true)
	sw2Put(b, 0, 0, "loyy")

	w := sw2Init(t, searchSlidingWindowDirectionForward, "hello")
	defer w.Deinit()
	sw2Append(t, w, a)
	// The empty rows above the text are newlines, one per row, and the
	// three empty cells before the text are spaces.
	expectEqual(t, uint(last)+3+5, w.data.Len())
	sw2Append(t, w, b)
	sw2ExpectHl(t, sw2Next(t, w), 5, 1, []sw2Chunk{
		{node: a, start: last, end: last + 1},
		{node: b, start: 0, end: 1},
	})
	expectEqual(t, a.data.size.rows, last+1)
	expect(t, !w.Next().Valid, "w.next() == null")
}

// Zig: test "term51: SlidingWindow.next: needle longer than the data matches after more pages are appended"
func TestSw2_SlidingWindow_next_needle_longer_than_the_data_matches_after_more_pages_are_appended(t *testing.T) {
	testStdCapacity(t)

	pl := sw2Two(t, [2]string{"abc", "def"}, [2]bool{true, true})
	defer pl.Deinit()
	a := sw2NodeAt(pl, 0)
	b := sw2NodeAt(pl, 1)

	w := sw2Init(t, searchSlidingWindowDirectionForward, "abcdef")
	defer w.Deinit()
	sw2Append(t, w, a)
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 1, w.meta.Len())
	expectEqual(t, 0, w.data_offset)
	sw2Append(t, w, b)
	sw2ExpectHl(t, sw2Next(t, w), 0, 2, []sw2Chunk{
		{node: a, start: 0, end: sw2Rows(a)},
		{node: b, start: 0, end: 1},
	})
	expect(t, !w.Next().Valid, "w.next() == null")
}

// Zig: test "term51: SlidingWindow.next: exhausted window continues with newly appended pages"
func TestSw2_SlidingWindow_next_exhausted_window_continues_with_newly_appended_pages(t *testing.T) {
	testStdCapacity(t)

	pl := sw2Two(t, [2]string{"xxa", "bxx"}, [2]bool{true, true})
	defer pl.Deinit()
	a := sw2NodeAt(pl, 0)
	b := sw2NodeAt(pl, 1)

	w := sw2Init(t, searchSlidingWindowDirectionForward, "ab")
	defer w.Deinit()
	sw2Append(t, w, a)
	expect(t, !w.Next().Valid, "w.next() == null")
	// Only needle.len - 1 = 1 trailing byte need to be searched again.
	expectEqual(t, 2, w.data_offset)
	expectEqual(t, 1, w.meta.Len())

	sw2Append(t, w, b)
	sw2ExpectHl(t, sw2Next(t, w), 2, 0, []sw2Chunk{
		{node: a, start: 0, end: sw2Rows(a)},
		{node: b, start: 0, end: 1},
	})
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 1, w.meta.Len())
	expectEqual(t, 3, w.data.Len())
	expectEqual(t, 2, w.data_offset)
}

// Zig: test "term51: SlidingWindow.next: prune keeps needle.len - 1 bytes (forward)"
func TestSw2_SlidingWindow_next_prune_keeps_needle_len_minus_1_bytes_forward(t *testing.T) {
	testStdCapacity(t)

	pl := sw2Three(t, [3]string{"aaaaa", "bb", "cc"}, [3]bool{true, true, true})
	defer pl.Deinit()

	// metas, data length, data_offset after a failed search. needle.len of
	// 3 and 5 would leave exactly needle.len - 1 bytes, which underflows
	// in upstream (see findings), so they aren't included. (The Go-only
	// test TestSw2_goonly_prune_leaves_exactly_needle_len_minus_1_bytes
	// covers them.)
	cases := []struct {
		n      int
		metas  uint
		data   uint
		offset uint
	}{
		{n: 1, metas: 0, data: 0, offset: 0},
		{n: 2, metas: 1, data: 2, offset: 1},
		{n: 4, metas: 2, data: 4, offset: 1},
		{n: 6, metas: 3, data: 9, offset: 4},
		{n: 7, metas: 3, data: 9, offset: 3},
		{n: 8, metas: 3, data: 9, offset: 2},
		{n: 9, metas: 3, data: 9, offset: 1},
		// Longer than the data: no search, no change.
		{n: 10, metas: 3, data: 9, offset: 0},
	}
	for _, case_ := range cases {
		w := sw2Init(t, searchSlidingWindowDirectionForward, "zzzzzzzzzz"[0:case_.n])
		defer w.Deinit()
		for i := uint(0); i < 3; i++ {
			sw2Append(t, w, sw2NodeAt(pl, i))
		}
		expect(t, !w.Next().Valid, "w.next() == null")
		expectEqual(t, case_.metas, w.meta.Len())
		expectEqual(t, case_.data, w.data.Len())
		expectEqual(t, case_.offset, w.data_offset)

		// A second search over the same state changes nothing more.
		expect(t, !w.Next().Valid, "w.next() == null")
		expectEqual(t, case_.metas, w.meta.Len())
		expectEqual(t, case_.data, w.data.Len())
		expectEqual(t, case_.offset, w.data_offset)
	}
}

// Zig: test "term51: SlidingWindow.next: prune keeps needle.len - 1 bytes (reverse)"
func TestSw2_SlidingWindow_next_prune_keeps_needle_len_minus_1_bytes_reverse(t *testing.T) {
	testStdCapacity(t)

	pl := sw2Three(t, [3]string{"aaaaa", "bb", "cc"}, [3]bool{true, true, true})
	defer pl.Deinit()

	// Appended last page first, so the newest meta is the 5 byte page.
	cases := []struct {
		n      int
		metas  uint
		data   uint
		offset uint
	}{
		{n: 1, metas: 0, data: 0, offset: 0},
		{n: 2, metas: 1, data: 5, offset: 4},
		{n: 3, metas: 1, data: 5, offset: 3},
		{n: 4, metas: 1, data: 5, offset: 2},
		{n: 5, metas: 1, data: 5, offset: 1},
		{n: 7, metas: 2, data: 7, offset: 1},
		{n: 9, metas: 3, data: 9, offset: 1},
		{n: 10, metas: 3, data: 9, offset: 0},
	}
	for _, case_ := range cases {
		w := sw2Init(t, searchSlidingWindowDirectionReverse, "zzzzzzzzzz"[0:case_.n])
		defer w.Deinit()
		for i := uint(3); i > 0; i-- {
			sw2Append(t, w, sw2NodeAt(pl, i-1))
		}
		expect(t, !w.Next().Valid, "w.next() == null")
		expectEqual(t, case_.metas, w.meta.Len())
		expectEqual(t, case_.data, w.data.Len())
		expectEqual(t, case_.offset, w.data_offset)
	}
}

// Zig: test "term51: SlidingWindow.next: pruning never shrinks the buffers"
func TestSw2_SlidingWindow_next_pruning_never_shrinks_the_buffers(t *testing.T) {
	testStdCapacity(t)

	pl := sw2Three(t, [3]string{"aaaaa", "bb", "cc"}, [3]bool{true, true, true})
	defer pl.Deinit()

	w := sw2Init(t, searchSlidingWindowDirectionForward, "zz")
	defer w.Deinit()
	for i := uint(0); i < 3; i++ {
		sw2Append(t, w, sw2NodeAt(pl, i))
	}
	expectEqual(t, 9, w.data.Capacity())
	expectEqual(t, 3, w.meta.Capacity())
	expect(t, w.data.full, "w.data.full")
	expect(t, w.meta.full, "w.meta.full")

	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 1, w.meta.Len())
	expectEqual(t, 2, w.data.Len())
	expectEqual(t, 9, w.data.Capacity())
	expectEqual(t, 3, w.meta.Capacity())
	expect(t, !w.data.full, "!w.data.full")
	expect(t, !w.meta.full, "!w.meta.full")
	// The oldest data was removed from the front.
	expectEqual(t, 7, w.data.tail)
	expectEqual(t, 2, w.meta.tail)
	sw2ExpectData(t, w, "cc")
	expectEqual(t, sw2NodeAt(pl, 2), sw2Meta(t, w, 0).node)
}

// Zig: test "term51: SlidingWindow.next: match in the overlap of a wrapped circular buffer"
func TestSw2_SlidingWindow_next_match_in_the_overlap_of_a_wrapped_circular_buffer(t *testing.T) {
	testStdCapacity(t)

	pl := sw2Three(t, [3]string{"pppppp", "xxab", "cdqq"}, [3]bool{true, true, true})
	defer pl.Deinit()
	p2 := sw2NodeAt(pl, 1)
	p3 := sw2NodeAt(pl, 2)

	w := sw2Init(t, searchSlidingWindowDirectionForward, "abcd")
	defer w.Deinit()
	sw2Append(t, w, sw2NodeAt(pl, 0))
	sw2Append(t, w, p2)
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 1, w.meta.Len())
	expectEqual(t, 4, w.data.Len())
	expectEqual(t, 1, w.data_offset)
	expectEqual(t, 10, w.data.Capacity())
	expectEqual(t, 6, w.data.tail)

	// The new page doesn't need a resize: it wraps around the end of
	// the storage: "xxab" at the end, "cdqq" at the start.
	sw2Append(t, w, p3)
	expectEqual(t, 10, w.data.Capacity())
	{
		slices := w.data.GetPtrSlice(0, w.data.Len())
		expectEqualStrings(t, "xxab", string(slices[0]))
		expectEqualStrings(t, "cdqq", string(slices[1]))
	}

	// slices[0] (after the offset) is "xab", slices[1] is "cdqq"; the
	// match "abcd" only exists across the two, found through the
	// overlap buffer.
	sw2ExpectHl(t, sw2Next(t, w), 2, 1, []sw2Chunk{
		{node: p2, start: 0, end: sw2Rows(p2)},
		{node: p3, start: 0, end: 1},
	})
	expectEqual(t, 2, w.meta.Len())
	expectEqual(t, 3, w.data_offset)

	// Now slices[0] is "b" and slices[1] "cdqq": the overlap buffer
	// is "bcdq" and nothing matches. The oldest page is pruned.
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 1, w.meta.Len())
	expectEqual(t, 4, w.data.Len())
	expectEqual(t, 1, w.data_offset)
	sw2ExpectData(t, w, "cdqq")
}

// Zig: test "term51: SlidingWindow.next: match in the second slice of a wrapped circular buffer"
func TestSw2_SlidingWindow_next_match_in_the_second_slice_of_a_wrapped_circular_buffer(t *testing.T) {
	testStdCapacity(t)

	pl := sw2Three(t, [3]string{"pppppp", "xxab", "zabcd"}, [3]bool{true, true, true})
	defer pl.Deinit()
	p3 := sw2NodeAt(pl, 2)

	w := sw2Init(t, searchSlidingWindowDirectionForward, "abcd")
	defer w.Deinit()
	sw2Append(t, w, sw2NodeAt(pl, 0))
	sw2Append(t, w, sw2NodeAt(pl, 1))
	expect(t, !w.Next().Valid, "w.next() == null")
	sw2Append(t, w, p3)
	expectEqual(t, 10, w.data.Capacity())
	{
		slices := w.data.GetPtrSlice(0, w.data.Len())
		expectEqualStrings(t, "xxab", string(slices[0]))
		expectEqualStrings(t, "zabcd", string(slices[1]))
	}

	// The overlap buffer would be "xabzab": no match. The second slice
	// holds the match.
	sw2ExpectHl(t, sw2Next(t, w), 1, 4, []sw2Chunk{{node: p3, start: 0, end: 1}})
	// The match started in the newest page, so the older one is pruned.
	expectEqual(t, 1, w.meta.Len())
	expectEqual(t, 5, w.data.Len())
	expectEqual(t, 2, w.data_offset)
	sw2ExpectData(t, w, "zabcd")
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 1, w.meta.Len())
	expectEqual(t, 2, w.data_offset)
}

// Zig: test "term51: SlidingWindow.next: match in the first slice of a wrapped circular buffer"
func TestSw2_SlidingWindow_next_match_in_the_first_slice_of_a_wrapped_circular_buffer(t *testing.T) {
	testStdCapacity(t)

	pl := sw2Three(t, [3]string{"pppppp", "abcdxxabcd", "zz"}, [3]bool{true, true, true})
	defer pl.Deinit()
	p2 := sw2NodeAt(pl, 1)

	w := sw2Init(t, searchSlidingWindowDirectionForward, "abcd")
	defer w.Deinit()
	sw2Append(t, w, sw2NodeAt(pl, 0))
	sw2Append(t, w, p2)
	sw2ExpectHl(t, sw2Next(t, w), 0, 3, []sw2Chunk{{node: p2, start: 0, end: 1}})
	expectEqual(t, 1, w.meta.Len())
	expectEqual(t, 10, w.data.Len())
	expectEqual(t, 1, w.data_offset)

	sw2Append(t, w, sw2NodeAt(pl, 2))
	expectEqual(t, 16, w.data.Capacity())
	expectEqual(t, 2, w.meta.Len())
	expectEqual(t, 2, len(w.data.GetPtrSlice(0, w.data.Len())[1]))

	sw2ExpectHl(t, sw2Next(t, w), 6, 9, []sw2Chunk{{node: p2, start: 0, end: 1}})
	expectEqual(t, 7, w.data_offset)
	expectEqual(t, 2, w.meta.Len())

	// No match left: slices are "bcd" and "zz".
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 2, w.meta.Len())
	expectEqual(t, 12, w.data.Len())
	expectEqual(t, 9, w.data_offset)
}

// Zig: test "term51: SlidingWindow.next: reverse match in the overlap of a wrapped circular buffer"
func TestSw2_SlidingWindow_next_reverse_match_in_the_overlap_of_a_wrapped_circular_buffer(t *testing.T) {
	testStdCapacity(t)

	// Screen order: "qqab" "cdxx" "pppppp"; appended last page first.
	pl := sw2Three(t, [3]string{"qqab", "cdxx", "pppppp"}, [3]bool{true, true, true})
	defer pl.Deinit()
	r3 := sw2NodeAt(pl, 0)
	r2 := sw2NodeAt(pl, 1)

	w := sw2Init(t, searchSlidingWindowDirectionReverse, "abcd")
	defer w.Deinit()
	sw2Append(t, w, sw2NodeAt(pl, 2))
	sw2Append(t, w, r2)
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 1, w.meta.Len())
	expectEqual(t, 1, w.data_offset)

	sw2Append(t, w, r3)
	{
		slices := w.data.GetPtrSlice(0, w.data.Len())
		expectEqualStrings(t, "xxdc", string(slices[0]))
		expectEqualStrings(t, "baqq", string(slices[1]))
	}
	h := sw2Next(t, w)
	sw2ExpectHl(t, h, 2, 1, []sw2Chunk{
		{node: r3, start: 0, end: sw2Rows(r3)},
		{node: r2, start: 0, end: 1},
	})
	expectEqual(t, 2, w.meta.Len())
	expectEqual(t, 3, w.data_offset)
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 1, w.meta.Len())
	expectEqual(t, 4, w.data.Len())
	expectEqual(t, 1, w.data_offset)
}

// Zig: test "term51: SlidingWindow.next: single byte needle with a wrapped circular buffer"
func TestSw2_SlidingWindow_next_single_byte_needle_with_a_wrapped_circular_buffer(t *testing.T) {
	testStdCapacity(t)

	pl := sw2Three(t, [3]string{"pppppp", "xqab", "zq"}, [3]bool{true, true, true})
	defer pl.Deinit()
	p2 := sw2NodeAt(pl, 1)
	p3 := sw2NodeAt(pl, 2)

	w := sw2Init(t, searchSlidingWindowDirectionForward, "q")
	defer w.Deinit()
	sw2Append(t, w, sw2NodeAt(pl, 0))
	sw2Append(t, w, p2)
	sw2ExpectHl(t, sw2Next(t, w), 1, 1, []sw2Chunk{{node: p2, start: 0, end: 1}})
	expectEqual(t, 1, w.meta.Len())
	expectEqual(t, 2, w.data_offset)

	sw2Append(t, w, p3)
	{
		slices := w.data.GetPtrSlice(0, w.data.Len())
		expectEqualStrings(t, "xqab", string(slices[0]))
		expectEqualStrings(t, "zq", string(slices[1]))
	}
	// With needle.len - 1 == 0 the overlap buffer is empty; the match
	// is found in the second slice.
	sw2ExpectHl(t, sw2Next(t, w), 1, 1, []sw2Chunk{{node: p3, start: 0, end: 1}})
	expectEqual(t, 1, w.meta.Len())
	expectEqual(t, 2, w.data.Len())
	expectEqual(t, 2, w.data_offset)

	// And the single byte special case clears the window.
	expect(t, !w.Next().Valid, "w.next() == null")
	expectEqual(t, 0, w.meta.Len())
	expectEqual(t, 0, w.data.Len())
}

// Zig: test "term51: SlidingWindow.next: growing a wrapped buffer rotates it and keeps the search state"
func TestSw2_SlidingWindow_next_growing_a_wrapped_buffer_rotates_it_and_keeps_the_search_state(t *testing.T) {
	testStdCapacity(t)

	pl := sw2Many(
		t,
		[]string{"pppppp", "xxab", "cdqq", "wxyz!!!!"},
		[]bool{true, true, true, true},
	)
	defer pl.Deinit()
	p2 := sw2NodeAt(pl, 1)
	p3 := sw2NodeAt(pl, 2)

	w := sw2Init(t, searchSlidingWindowDirectionForward, "abcd")
	defer w.Deinit()
	sw2Append(t, w, sw2NodeAt(pl, 0))
	sw2Append(t, w, p2)
	expect(t, !w.Next().Valid, "w.next() == null")
	sw2Append(t, w, p3)
	expect(t, len(w.data.GetPtrSlice(0, w.data.Len())[1]) > 0, "w.data.getPtrSlice(0, w.data.len())[1].len > 0")
	expectEqual(t, 2, w.meta.Capacity())

	// 8 bytes + 8 more don't fit in the 10 byte storage: it is grown to
	// exactly 16, rotated so the oldest byte is at storage[0].
	sw2Append(t, w, sw2NodeAt(pl, 3))
	expectEqual(t, 16, w.data.Capacity())
	expectEqual(t, 0, w.data.tail)
	expectEqual(t, 3, w.meta.Capacity())
	expectEqual(t, 0, w.meta.tail)
	expectEqual(t, 0, len(w.data.GetPtrSlice(0, w.data.Len())[1]))
	sw2ExpectData(t, w, "xxabcdqqwxyz!!!!")
	expectEqual(t, p2, sw2Meta(t, w, 0).node)
	expectEqual(t, p3, sw2Meta(t, w, 1).node)
	expectEqual(t, sw2NodeAt(pl, 3), sw2Meta(t, w, 2).node)
	expectEqual(t, 1, w.data_offset)

	sw2ExpectHl(t, sw2Next(t, w), 2, 1, []sw2Chunk{
		{node: p2, start: 0, end: sw2Rows(p2)},
		{node: p3, start: 0, end: 1},
	})
	expectEqual(t, 3, w.meta.Len())
	expect(t, !w.Next().Valid, "w.next() == null")
}

// Zig: test "term51: SlidingWindow.next: window refilled from a recycled page sees the blank page and its new serial"
func TestSw2_SlidingWindow_next_window_refilled_from_a_recycled_page_sees_the_blank_page_and_its_new_serial(t *testing.T) {
	testStdCapacity(t)

	// Room for exactly two pages: growing past that recycles the first.
	pl, err := PageListInit(sw2Cols, 5, Some[uint64](0))
	zigTry(t, err)
	defer pl.Deinit()
	pl.PauseIntegrityChecks(true)
	for sw2NodeCount(&pl) < 2 {
		sw2Grow(t, &pl)
	}
	a := sw2NodeAt(&pl, 0)
	b := sw2NodeAt(&pl, 1)
	sw2Put(a, 3, 0, "xxabc")
	sw2SetWrap(a, true)
	sw2Put(b, 0, 0, "def")
	a_rows := a.data.size.rows
	a_serial := a.serial
	b_serial := b.serial
	expect(t, a_rows > 4, "a_rows > 4")

	w := sw2Init(t, searchSlidingWindowDirectionForward, "abcdef")
	defer w.Deinit()
	sw2Append(t, w, a)
	sw2Append(t, w, b)

	// Not recycled: the highlight is as expected.
	{
		h := sw2Next(t, w)
		sw2ExpectHl(t, h, 2, 2, []sw2Chunk{
			{node: a, start: 3, end: a_rows},
			{node: b, start: 0, end: 1},
		})
		expectEqual(t, a_serial, h.Chunks[0].Serial)
		expectEqual(t, b_serial, h.Chunks[1].Serial)
	}

	// Recycle page A: it becomes the last page again with a new serial
	// and one blank row, while the window still points at the node.
	for pl.pages.last != a {
		sw2Grow(t, &pl)
	}
	pl.PauseIntegrityChecks(false)
	expect(t, a.serial != a_serial, "a.serial != a_serial")
	expectEqual(t, 1, a.data.size.rows)
	expectEqual(t, b, pl.pages.first)

	// The recycled node is now blank: a window refilled from it sees one
	// newline for it, and records its new serial.
	w.ClearAndRetainCapacity()
	sw2Append(t, w, b)
	expectEqual(t, 1, w.meta.Len())
	expectEqual(t, b_serial, sw2Meta(t, w, 0).serial)
	sw2Append(t, w, a)
	expectEqual(t, 2, w.meta.Len())
	sw2ExpectData(t, w, "def\n\n")
	expectEqual(t, a.serial, sw2Meta(t, w, 1).serial)
	expect(t, sw2Meta(t, w, 1).serial != a_serial, "term51AKMeta(&w, 1).serial != a_serial")
}

// Zig: test "term51: SlidingWindow.next: stale pages keep their append-time serial and live row count"
func TestSw2_SlidingWindow_next_stale_pages_keep_their_append_time_serial_and_live_row_count(t *testing.T) {
	testStdCapacity(t)

	pl, err := PageListInit(sw2Cols, 5, Some[uint64](0))
	zigTry(t, err)
	defer pl.Deinit()
	pl.PauseIntegrityChecks(true)
	for sw2NodeCount(&pl) < 2 {
		sw2Grow(t, &pl)
	}
	a := sw2NodeAt(&pl, 0)
	b := sw2NodeAt(&pl, 1)
	sw2Put(a, 3, 0, "xxabc")
	sw2SetWrap(a, true)
	sw2Put(b, 0, 0, "def")
	a_serial := a.serial

	w := sw2Init(t, searchSlidingWindowDirectionForward, "abcdef")
	defer w.Deinit()
	sw2Append(t, w, a)
	sw2Append(t, w, b)

	for pl.pages.last != a {
		sw2Grow(t, &pl)
	}
	pl.PauseIntegrityChecks(false)
	expect(t, a.serial != a_serial, "a.serial != a_serial")
	expectEqual(t, 1, a.data.size.rows)

	h := sw2Next(t, w)
	sw2ExpectHl(t, h, 2, 2, []sw2Chunk{
		// start is the row from append time (3), end is the recycled
		// node's current size (1).
		{node: a, start: 3, end: 1},
		{node: b, start: 0, end: 1},
	})
	expectEqual(t, a_serial, h.Chunks[0].Serial)
	expect(t, h.Chunks[0].Serial != a.serial, "h.chunks.items(.serial)[0] != a.serial")
	expectEqual(t, b.serial, h.Chunks[1].Serial)
}

// Zig: test "term51: SlidingWindow.next: results alias the internal chunk buffer"
func TestSw2_SlidingWindow_next_results_alias_the_internal_chunk_buffer(t *testing.T) {
	testStdCapacity(t)

	pl := sw2List(t, 1, 0)
	defer pl.Deinit()
	node := pl.pages.first
	sw2Put(node, 0, 0, "ab")
	sw2Put(node, 2, 0, "ab")
	sw2SetWrap(node, true)
	// Data: "ab\n\nab"

	w := sw2Init(t, searchSlidingWindowDirectionForward, "ab")
	defer w.Deinit()
	sw2Append(t, w, node)

	h1 := sw2Next(t, w)
	expectEqual(t, 1, len(w.chunk_buf))
	expect(t, &h1.Chunks[0] == &w.chunk_buf[0], "h1.chunks.bytes == w.chunk_buf.bytes")
	expectEqual(t, 0, h1.Chunks[0].Start)
	expectEqual(t, 1, h1.Chunks[0].End)
	kept, err := h1.Clone()
	zigTry(t, err)
	defer kept.Deinit()

	h2 := sw2Next(t, w)
	expect(t, &h2.Chunks[0] == &h1.Chunks[0], "h2.chunks.bytes == h1.chunks.bytes")
	expectEqual(t, 2, h2.Chunks[0].Start)
	expectEqual(t, 3, h2.Chunks[0].End)
	// The first result's chunk memory now shows the second match, but a
	// clone is unaffected.
	expectEqual(t, 2, h1.Chunks[0].Start)
	expectEqual(t, 0, kept.Chunks[0].Start)
	expectEqual(t, 1, kept.Chunks[0].End)
	expectEqual(t, 0, kept.TopX)
	expectEqual(t, 1, kept.BotX)
}

// Zig: test "term51: SlidingWindow.highlight: start offset is relative to data_offset"
func TestSw2_SlidingWindow_highlight_start_offset_is_relative_to_data_offset(t *testing.T) {
	testStdCapacity(t)

	pl := sw2List(t, 1, 0)
	defer pl.Deinit()
	node := pl.pages.first
	sw2Put(node, 0, 0, "abcdefgh")
	sw2SetWrap(node, true)

	w := sw2Init(t, searchSlidingWindowDirectionForward, "zz")
	defer w.Deinit()
	sw2Append(t, w, node)

	// The length is independent of the needle length.
	w.data_offset = 2
	sw2ExpectHl(t, w.highlight(1, 2), 3, 4, []sw2Chunk{{node: node, start: 0, end: 1}})
	expectEqual(t, 4, w.data_offset)
	expectEqual(t, 1, w.meta.Len())

	w.data_offset = 0
	sw2ExpectHl(t, w.highlight(1, 6), 1, 6, []sw2Chunk{{node: node, start: 0, end: 1}})
	expectEqual(t, 2, w.data_offset)

	// The last byte, and a single byte.
	sw2ExpectHl(t, w.highlight(5, 1), 7, 7, []sw2Chunk{{node: node, start: 0, end: 1}})
	expectEqual(t, 8, w.data_offset)
	w.data_offset = 0
	sw2ExpectHl(t, w.highlight(0, 8), 0, 7, []sw2Chunk{{node: node, start: 0, end: 1}})
	expectEqual(t, 1, w.data_offset)
}

// Zig: test "term51: SlidingWindow.highlight: prunes metas before the one holding the match start"
func TestSw2_SlidingWindow_highlight_prunes_metas_before_the_one_holding_the_match_start(t *testing.T) {
	testStdCapacity(t)

	pl := sw2Three(t, [3]string{"aaaa", "bbbb", "cccc"}, [3]bool{true, true, true})
	defer pl.Deinit()
	a := sw2NodeAt(pl, 0)
	b := sw2NodeAt(pl, 1)
	c := sw2NodeAt(pl, 2)

	// Match inside the middle page only: A is pruned.
	{
		w := sw2Init(t, searchSlidingWindowDirectionForward, "zz")
		defer w.Deinit()
		for i := uint(0); i < 3; i++ {
			sw2Append(t, w, sw2NodeAt(pl, i))
		}
		sw2ExpectHl(t, w.highlight(5, 2), 1, 2, []sw2Chunk{{node: b, start: 0, end: 1}})
		expectEqual(t, 2, w.meta.Len())
		expectEqual(t, 8, w.data.Len())
		expectEqual(t, 2, w.data_offset)
		expectEqual(t, b, sw2Meta(t, w, 0).node)
		expectEqual(t, c, sw2Meta(t, w, 1).node)
		sw2ExpectData(t, w, "bbbbcccc")
	}

	// Match from the middle page into the last: A is pruned, B is kept.
	{
		w := sw2Init(t, searchSlidingWindowDirectionForward, "zz")
		defer w.Deinit()
		for i := uint(0); i < 3; i++ {
			sw2Append(t, w, sw2NodeAt(pl, i))
		}
		sw2ExpectHl(t, w.highlight(6, 4), 2, 1, []sw2Chunk{
			{node: b, start: 0, end: sw2Rows(b)},
			{node: c, start: 0, end: 1},
		})
		expectEqual(t, 2, w.meta.Len())
		expectEqual(t, 8, w.data.Len())
		expectEqual(t, 3, w.data_offset)
	}

	// Match across all three pages: nothing is pruned.
	{
		w := sw2Init(t, searchSlidingWindowDirectionForward, "zz")
		defer w.Deinit()
		for i := uint(0); i < 3; i++ {
			sw2Append(t, w, sw2NodeAt(pl, i))
		}
		sw2ExpectHl(t, w.highlight(2, 10), 2, 3, []sw2Chunk{
			{node: a, start: 0, end: sw2Rows(a)},
			{node: b, start: 0, end: sw2Rows(b)},
			{node: c, start: 0, end: 1},
		})
		expectEqual(t, 3, w.meta.Len())
		expectEqual(t, 12, w.data.Len())
		expectEqual(t, 3, w.data_offset)
	}

	// Match in the last page only: both earlier pages are pruned.
	{
		w := sw2Init(t, searchSlidingWindowDirectionForward, "zz")
		defer w.Deinit()
		for i := uint(0); i < 3; i++ {
			sw2Append(t, w, sw2NodeAt(pl, i))
		}
		sw2ExpectHl(t, w.highlight(10, 2), 2, 3, []sw2Chunk{{node: c, start: 0, end: 1}})
		expectEqual(t, 1, w.meta.Len())
		expectEqual(t, 4, w.data.Len())
		expectEqual(t, 3, w.data_offset)
	}
}

// Zig: test "term51: SlidingWindow.highlight: reverse flips chunks, y bounds and x bounds"
func TestSw2_SlidingWindow_highlight_reverse_flips_chunks_y_bounds_and_x_bounds(t *testing.T) {
	testStdCapacity(t)

	// Single chunk over several rows: "abcdefgh" soft-wrapped in four columns.
	{
		s := sw2Screen(t, 4, 5)
		defer s.Deinit()
		zigTry(t, s.TestWriteString([]uint8("abcdefgh")))
		node := s.Pages.pages.first

		// Reversed data: "\n" "hgfe" "dcba", map y: 1 for \nhgfe, 0 for dcba.
		w := sw2Init(t, searchSlidingWindowDirectionReverse, "zz")
		defer w.Deinit()
		sw2Append(t, w, node)
		// idx 3..6 are "fedc": from (1,1) to (2,0) in the reversed order.
		sw2ExpectHl(t, w.highlight(3, 4), 2, 1, []sw2Chunk{{node: node, start: 0, end: 2}})
		expectEqual(t, 4, w.data_offset)

		// A one-row match keeps a one-row chunk (idx 5..6 = "dc").
		w.data_offset = 0
		sw2ExpectHl(t, w.highlight(5, 2), 2, 3, []sw2Chunk{{node: node, start: 0, end: 1}})
	}

	// The upstream-verified two and three chunk shapes are checked in
	// the `next` tests. Here: a match that starts in the first reverse
	// page and ends mid-way in the second one, with the newest page
	// (A) being the screen-earlier one.
	{
		pl := sw2Three(t, [3]string{"aaaa", "bbbb", "cccc"}, [3]bool{true, true, true})
		defer pl.Deinit()
		a := sw2NodeAt(pl, 0)
		b := sw2NodeAt(pl, 1)
		c := sw2NodeAt(pl, 2)
		w := sw2Init(t, searchSlidingWindowDirectionReverse, "zz")
		defer w.Deinit()
		sw2Append(t, w, c)
		sw2Append(t, w, b)
		sw2Append(t, w, a)
		// Reversed data "cccc" "bbbb" "aaaa": idx 2..9 covers the last two
		// reversed cells of C (x=1,0), all of B and the first two reversed
		// cells of A (x=3,2).
		sw2ExpectHl(t, w.highlight(2, 8), 2, 1, []sw2Chunk{
			{node: a, start: 0, end: sw2Rows(a)},
			{node: b, start: 0, end: sw2Rows(b)},
			{node: c, start: 0, end: 1},
		})
	}
}

// Zig: test "term51: SlidingWindow.highlight: interior blank cells are mapped in descending x order"
func TestSw2_SlidingWindow_highlight_interior_blank_cells_are_mapped_in_descending_x_order(t *testing.T) {
	testStdCapacity(t)

	pl := sw2List(t, 1, 0)
	defer pl.Deinit()
	node := pl.pages.first
	sw2Put(node, 0, 0, "a")
	sw2Put(node, 0, 3, "b")
	sw2SetWrap(node, true)
	// Data "a  b", map x: 0, 2, 1, 3.

	for _, dir := range sw2Directions {
		w := sw2Init(t, dir, "  ")
		defer w.Deinit()
		sw2Append(t, w, node)
		// Upstream quirk: the highlight of the two blank cells starts at
		// x=2 and ends at x=1, in either direction.
		sw2ExpectHl(t, sw2Next(t, w), 2, 1, []sw2Chunk{{node: node, start: 0, end: 1}})
		expect(t, !w.Next().Valid, "w.next() == null")
	}

	// The match across the blanks and the letters is still sensible.
	w := sw2Init(t, searchSlidingWindowDirectionForward, "a  b")
	defer w.Deinit()
	sw2Append(t, w, node)
	sw2ExpectHl(t, sw2Next(t, w), 0, 3, []sw2Chunk{{node: node, start: 0, end: 1}})
}

// Zig: test "term51: SlidingWindow.testChangeNeedle: replaces the needle as given"
func TestSw2_SlidingWindow_testChangeNeedle_replaces_the_needle_as_given(t *testing.T) {
	testStdCapacity(t)

	w := sw2Init(t, searchSlidingWindowDirectionForward, "abc")
	defer w.Deinit()
	w.testChangeNeedle([]uint8("xyz"))
	expectEqualStrings(t, "xyz", string(w.needle))
	expectEqual(t, 6, len(w.overlap_buf))

	// It does not reverse the needle for a reverse window.
	r := sw2Init(t, searchSlidingWindowDirectionReverse, "abc")
	defer r.Deinit()
	expectEqualStrings(t, "cba", string(r.needle))
	r.testChangeNeedle([]uint8("xyz"))
	expectEqualStrings(t, "xyz", string(r.needle))

	// The new needle is the one searched for.
	pl := sw2List(t, 1, 0)
	defer pl.Deinit()
	node := pl.pages.first
	sw2Put(node, 0, 0, "..xyz..")
	sw2SetWrap(node, true)
	sw2Append(t, w, node)
	sw2ExpectHl(t, sw2Next(t, w), 2, 4, []sw2Chunk{{node: node, start: 0, end: 1}})
}

// Zig: test "term51: SlidingWindow.assertIntegrity: holds for every reachable state"
func TestSw2_SlidingWindow_assertIntegrity_holds_for_every_reachable_state(t *testing.T) {
	testStdCapacity(t)

	pl := sw2Two(t, [2]string{"xxab", "cdyy"}, [2]bool{true, true})
	defer pl.Deinit()

	w := sw2Init(t, searchSlidingWindowDirectionForward, "abcd")
	defer w.Deinit()

	// Empty window (data_offset == 0 == data.len()).
	w.assertIntegrity()

	// Filled window.
	sw2Append(t, w, sw2NodeAt(pl, 0))
	w.assertIntegrity()
	sw2Append(t, w, sw2NodeAt(pl, 1))
	w.assertIntegrity()

	// After a match (offset > 0, still inside the data).
	expect(t, w.Next().Valid, "w.next() != null")
	expectEqual(t, 3, w.data_offset)
	w.assertIntegrity()

	// The largest valid offset is data.len() - 1.
	w.data_offset = w.data.Len() - 1
	w.assertIntegrity()
	w.data_offset = 0
	w.assertIntegrity()

	// After clearing.
	w.ClearAndRetainCapacity()
	w.assertIntegrity()

	// A violated invariant (data_offset == data.len() with data, or meta
	// lengths not matching the data length) panics, which can't be
	// asserted in a Zig test. Verify the sums that it checks instead.
	// (The Go-only panic tests below do assert the panics.)
	sw2Append(t, w, sw2NodeAt(pl, 0))
	var total uint
	it := w.meta.Iterator(DatastructCircBufIteratorDirectionForward)
	for m := it.Next(); m != nil; m = it.Next() {
		total += uint(len(m.cell_map))
	}
	expectEqual(t, w.data.Len(), total)
}

// Go-only: next: pruning leaves exactly needle.len - 1 bytes (forward). Upstream
// computes `self.data.len() - self.needle.len + 1` as a usize and underflows
// (a panic in a safe build, wrapping to the intended 0 in a release build);
// the Go port computes `len + 1 - needle.len`. With a data_offset of 0 after
// the prune the next search starts at the beginning of the remaining data,
// which is shorter than the needle, so `next` returns null at once and
// changes nothing. Expected values derived from the Zig source by hand, not
// from a release build run.
func TestSw2_goonly_prune_leaves_exactly_needle_len_minus_1_bytes_forward(t *testing.T) {
	testStdCapacity(t)

	pl := sw2Three(t, [3]string{"aaaaa", "bb", "cc"}, [3]bool{true, true, true})
	defer pl.Deinit()

	cases := []struct {
		n      int
		metas  uint
		data   uint
		offset uint
	}{
		// needed 2: "cc" holds exactly 2 bytes, the older pages are pruned.
		{n: 3, metas: 1, data: 2, offset: 0},
		// needed 4: "cc" and "bb" hold exactly 4 bytes, "aaaaa" is pruned.
		{n: 5, metas: 2, data: 4, offset: 0},
	}
	for _, case_ := range cases {
		w := sw2Init(t, searchSlidingWindowDirectionForward, "zzzzzzzzzz"[0:case_.n])
		defer w.Deinit()
		for i := uint(0); i < 3; i++ {
			sw2Append(t, w, sw2NodeAt(pl, i))
		}
		expect(t, !w.Next().Valid, "w.next() == null")
		expectEqual(t, case_.metas, w.meta.Len())
		expectEqual(t, case_.data, w.data.Len())
		expectEqual(t, case_.offset, w.data_offset)

		// The window now holds fewer bytes than the needle: nothing more.
		expect(t, !w.Next().Valid, "w.next() == null")
		expectEqual(t, case_.metas, w.meta.Len())
		expectEqual(t, case_.data, w.data.Len())
		expectEqual(t, case_.offset, w.data_offset)
	}
}

// Go-only: next: pruning leaves exactly needle.len - 1 bytes (reverse). See
// the forward test above.
func TestSw2_goonly_prune_leaves_exactly_needle_len_minus_1_bytes_reverse(t *testing.T) {
	testStdCapacity(t)

	pl := sw2Three(t, [3]string{"aaaaa", "bb", "cc"}, [3]bool{true, true, true})
	defer pl.Deinit()

	// Appended last page first, so the newest meta is the 5 byte page.
	cases := []struct {
		n      int
		metas  uint
		data   uint
		offset uint
	}{
		// needed 5: the 5 byte page holds exactly 5 bytes.
		{n: 6, metas: 1, data: 5, offset: 0},
		// needed 7: the 5 byte page and the "bb" page hold exactly 7 bytes.
		{n: 8, metas: 2, data: 7, offset: 0},
	}
	for _, case_ := range cases {
		w := sw2Init(t, searchSlidingWindowDirectionReverse, "zzzzzzzzzz"[0:case_.n])
		defer w.Deinit()
		for i := uint(3); i > 0; i-- {
			sw2Append(t, w, sw2NodeAt(pl, i-1))
		}
		expect(t, !w.Next().Valid, "w.next() == null")
		expectEqual(t, case_.metas, w.meta.Len())
		expectEqual(t, case_.data, w.data.Len())
		expectEqual(t, case_.offset, w.data_offset)

		expect(t, !w.Next().Valid, "w.next() == null")
		expectEqual(t, case_.metas, w.meta.Len())
		expectEqual(t, case_.data, w.data.Len())
		expectEqual(t, case_.offset, w.data_offset)
	}
}

// sw2WrappedWindow is a forward window with the one wrapped page "abcd"
// appended (data "abcd", one meta).
func sw2WrappedWindow(t *testing.T) (*PageList, *SearchSlidingWindow) {
	t.Helper()
	pl := sw2List(t, 1, 0)
	node := pl.pages.first
	sw2Put(node, 0, 0, "abcd")
	sw2SetWrap(node, true)
	w := sw2Init(t, searchSlidingWindowDirectionForward, "zz")
	sw2Append(t, w, node)
	return pl, w
}

// Go-only: panic test: testChangeNeedle: the new needle has a different length
func TestSw2_panic_testChangeNeedle_new_needle_has_a_different_length(t *testing.T) {
	testStdCapacity(t)

	w := sw2Init(t, searchSlidingWindowDirectionForward, "abc")
	defer w.Deinit()
	sw2ExpectPanic(t, func() { w.testChangeNeedle([]uint8("ab")) })
	sw2ExpectPanic(t, func() { w.testChangeNeedle([]uint8("abcd")) })
}

// Go-only: panic test: assertIntegrity: data_offset equals data.len() while there is data
func TestSw2_panic_assertIntegrity_data_offset_equals_data_len(t *testing.T) {
	testStdCapacity(t)

	pl, w := sw2WrappedWindow(t)
	defer pl.Deinit()
	defer w.Deinit()
	w.data_offset = w.data.Len()
	sw2ExpectPanic(t, func() { w.assertIntegrity() })
	w.data_offset = w.data.Len() + 1
	sw2ExpectPanic(t, func() { w.assertIntegrity() })
}

// Go-only: panic test: assertIntegrity: the cell maps do not add up to data.len()
func TestSw2_panic_assertIntegrity_cell_maps_do_not_match_data_len(t *testing.T) {
	testStdCapacity(t)

	pl, w := sw2WrappedWindow(t)
	defer pl.Deinit()
	defer w.Deinit()
	m := sw2Meta(t, w, 0)
	m.cell_map = append(m.cell_map, PointCoordinate{})
	sw2ExpectPanic(t, func() { w.assertIntegrity() })
	m.cell_map = m.cell_map[:len(m.cell_map)-2]
	sw2ExpectPanic(t, func() { w.assertIntegrity() })
}

// Go-only: panic test: highlight: start is not inside the data
func TestSw2_panic_highlight_start_is_not_inside_the_data(t *testing.T) {
	testStdCapacity(t)

	pl, w := sw2WrappedWindow(t)
	defer pl.Deinit()
	defer w.Deinit()
	sw2ExpectPanic(t, func() { w.highlight(4, 1) })

	// The start is relative to data_offset.
	w.data_offset = 2
	sw2ExpectPanic(t, func() { w.highlight(2, 1) })
}

// Go-only: panic test: highlight: start + len goes past the end of the data
func TestSw2_panic_highlight_end_is_not_inside_the_data(t *testing.T) {
	testStdCapacity(t)

	pl, w := sw2WrappedWindow(t)
	defer pl.Deinit()
	defer w.Deinit()
	sw2ExpectPanic(t, func() { w.highlight(3, 2) })
	sw2ExpectPanic(t, func() { w.highlight(0, 5) })
}

// Go-only: panic test: next: data_offset is past the end of the data (data.len() - data_offset underflows)
func TestSw2_panic_next_data_offset_past_the_end_of_the_data(t *testing.T) {
	testStdCapacity(t)

	pl, w := sw2WrappedWindow(t)
	defer pl.Deinit()
	defer w.Deinit()
	w.data_offset = w.data.Len() + 1
	sw2ExpectPanic(t, func() { w.Next() })
}
