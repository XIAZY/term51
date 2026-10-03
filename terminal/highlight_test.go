// Tests translated one for one from src/terminal/highlight.zig (Ghostty
// v1.3.1 plus the term51 additional tests). Each Go test carries the exact
// Zig title in a "// Zig: test" comment. Helpers are prefixed ss (the
// shared ones, ssYPin, ssYExpectPin, ssYPages, ssYFirstRows, ssScreenInit,
// ssExpectPanic, are in ScreenSet_test.go).
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualStrings keep Zig's names and Zig's
//     (expected, actual) argument order. `try` on a call that returns an
//     error is zigTry(t, err); `x.?` is zigUnwrap or a nil check.
//   - `std.MultiArrayList(Chunk)` is `[]HighlightFlattenedChunk`:
//     `sl.items(.node)[i]` is `chunks[i].node`; two lists have separate
//     storage when the addresses of their first elements differ.
//   - `@typeInfo(Flattened.Chunk)` is checked with package reflect.
//   - Tracked pins and `Tracked` highlights compare by pointer.
//   - Pin accessors that Zig reads as `s.pages.pages.first.?` are the Go
//     pointer `s.pages.pages.first`, checked for nil first.
//   - The three FailingAllocator tests are not translated; they are listed
//     in scripts/not-translated-ss.txt.
//   - Go-only panic tests: Flattened.startPin, endPin and untracked on a
//     highlight without chunks (index 0 out of bounds in Zig).

package terminal

import (
	"reflect"
	"testing"
)

// ssHlFlatten is term51YFlatten: what Flattened.init is meant to do, but
// written so that it compiles (upstream's init cannot be instantiated).
func ssHlFlatten(t *testing.T, start PageListPin, end PageListPin) HighlightFlattened {
	t.Helper()
	var result []HighlightFlattenedChunk
	it := start.PageIterator(PageListDirectionRightDown, Some(end))
	for {
		o := it.Next()
		if !o.Valid {
			break
		}
		chunk := o.Value
		result = append(result, HighlightFlattenedChunk{
			node:   chunk.node,
			serial: chunk.node.serial,
			start:  chunk.start,
			end:    chunk.end,
		})
	}
	return HighlightFlattened{
		chunks: result,
		top_x:  start.x,
		bot_x:  end.x,
	}
}

// Zig: test "term51: Untracked.eql: compares start and end pins"
func TestHighlight_term51_Untracked_eql_compares_start_and_end_pins(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	a := ssYPin(t, s, 1, 1)
	b := ssYPin(t, s, 5, 5)
	hl := HighlightUntracked{start: a, end: b}

	expect(t, hl.Eql(hl), "hl.eql(hl)")
	expect(t, hl.Eql(HighlightUntracked{start: a, end: b}), "hl.eql(.{ .start = a, .end = b })")
	// Each coordinate of each pin matters.
	expect(t, !hl.Eql(HighlightUntracked{start: ssYPin(t, s, 2, 1), end: b}), "!hl.eql(.{ .start = term51YPin(&s, 2, 1), .end = b })")
	expect(t, !hl.Eql(HighlightUntracked{start: ssYPin(t, s, 1, 2), end: b}), "!hl.eql(.{ .start = term51YPin(&s, 1, 2), .end = b })")
	expect(t, !hl.Eql(HighlightUntracked{start: a, end: ssYPin(t, s, 6, 5)}), "!hl.eql(.{ .start = a, .end = term51YPin(&s, 6, 5) })")
	expect(t, !hl.Eql(HighlightUntracked{start: a, end: ssYPin(t, s, 5, 6)}), "!hl.eql(.{ .start = a, .end = term51YPin(&s, 5, 6) })")
	// Swapped pins are a different highlight.
	expect(t, !hl.Eql(HighlightUntracked{start: b, end: a}), "!hl.eql(.{ .start = b, .end = a })")
	// A one-cell highlight.
	one := HighlightUntracked{start: a, end: a}
	expect(t, one.Eql(HighlightUntracked{start: a, end: a}), "one.eql(.{ .start = a, .end = a })")
	expect(t, !one.Eql(hl), "!one.eql(hl)")
	expect(t, !hl.Eql(one), "!hl.eql(one)")

	// The garbage flag is not part of equality.
	g := a
	g.garbage = true
	expect(t, hl.Eql(HighlightUntracked{start: g, end: b}), "hl.eql(.{ .start = g, .end = b })")
}

// Zig: test "term51: Untracked.eql: pins on different pages are different"
func TestHighlight_term51_Untracked_eql_pins_on_different_pages_are_different(t *testing.T) {
	testStdCapacity(t)
	s := ssYPages(t, 2)
	defer s.Deinit()
	p1 := ssYPin(t, s, 1, 1)
	p2 := p1
	expect(t, s.pages.pages.last != nil, "s.pages.pages.last.?")
	p2.node = s.pages.pages.last

	hl := HighlightUntracked{start: p1, end: p1}
	expect(t, hl.Eql(HighlightUntracked{start: p1, end: p1}), "hl.eql(.{ .start = p1, .end = p1 })")
	expect(t, !hl.Eql(HighlightUntracked{start: p2, end: p1}), "!hl.eql(.{ .start = p2, .end = p1 })")
	expect(t, !hl.Eql(HighlightUntracked{start: p1, end: p2}), "!hl.eql(.{ .start = p1, .end = p2 })")
	expect(t, (HighlightUntracked{start: p2, end: p2}).Eql(HighlightUntracked{start: p2, end: p2}), "(Untracked{ .start = p2, .end = p2 }).eql(.{ .start = p2, .end = p2 })")
}

// Zig: test "term51: Untracked.track: tracked copy with the same pins"
func TestHighlight_term51_Untracked_track_tracked_copy_with_the_same_pins(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	base := s.pages.CountTrackedPins()

	hl := HighlightUntracked{start: ssYPin(t, s, 2, 1), end: ssYPin(t, s, 7, 4)}
	tr, err := hl.Track(s)
	zigTry(t, err)
	expectEqual(t, base+2, s.pages.CountTrackedPins())
	expect(t, tr.start != tr.end, "t.start != t.end")
	expect(t, tr.start.Eql(hl.start), "t.start.eql(hl.start)")
	expect(t, tr.end.Eql(hl.end), "t.end.eql(hl.end)")
	ssYExpectPin(t, s, *tr.start, 2, 1)
	ssYExpectPin(t, s, *tr.end, 7, 4)

	// The pins really are tracked by the page list.
	found_start := false
	found_end := false
	for _, p := range s.pages.TrackedPins() {
		if p == tr.start {
			found_start = true
		}
		if p == tr.end {
			found_end = true
		}
	}
	expect(t, found_start && found_end, "found_start and found_end")

	// Moving the tracked pins does not touch the untracked highlight.
	tr.start.x = 0
	ssYExpectPin(t, s, hl.start, 2, 1)

	tr.Deinit(s)
	expectEqual(t, base, s.pages.CountTrackedPins())
}

// Zig: test "term51: Untracked.track: tracked pins follow erased rows"
func TestHighlight_term51_Untracked_track_tracked_pins_follow_erased_rows(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 3, 1000)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("L0\nL1\nL2\nL3\nL4\nL5\nL6\nL7\nL8\nL9\nL10\nL11")))
	expectEqual(t, uint(12), s.pages.total_rows)

	hl := HighlightUntracked{start: ssYPin(t, s, 1, 3), end: ssYPin(t, s, 2, 6)}
	tr, err := hl.Track(s)
	zigTry(t, err)
	defer tr.Deinit(s)

	// Rows above the highlight are removed: both pins move up by two.
	s.pages.EraseRows(ssScreen(0, 0), Some(ssScreen(0, 1)))
	ssYExpectPin(t, s, *tr.start, 1, 1)
	ssYExpectPin(t, s, *tr.end, 2, 4)
	expect(t, !tr.start.garbage, "!t.start.garbage")
	expect(t, !tr.end.garbage, "!t.end.garbage")

	// The start's row is removed: that pin goes to the top-left of the page.
	s.pages.EraseRows(ssScreen(0, 0), Some(ssScreen(0, 1)))
	ssYExpectPin(t, s, *tr.start, 0, 0)
	ssYExpectPin(t, s, *tr.end, 2, 2)
}

// Zig: test "term51: Untracked.track: tracked pins follow an erased page"
func TestHighlight_term51_Untracked_track_tracked_pins_follow_an_erased_page(t *testing.T) {
	testStdCapacity(t)
	s := ssYPages(t, 2)
	defer s.Deinit()
	n := ssYFirstRows(s)

	// A highlight spanning the page boundary.
	hl := HighlightUntracked{start: ssYPin(t, s, 3, n-1), end: ssYPin(t, s, 4, n+1)}
	tr, err := hl.Track(s)
	zigTry(t, err)
	defer tr.Deinit(s)
	expect(t, tr.start.node != tr.end.node, "t.start.node != t.end.node")

	s.pages.EraseRows(ssScreen(0, 0), Some(ssScreen(0, n-1)))
	expectEqual(t, uint(1), s.pages.TotalPages())
	// The pin on the erased page moves to the top-left of the next page; the
	// other keeps its place relative to its own rows.
	ssYExpectPin(t, s, *tr.start, 0, 0)
	ssYExpectPin(t, s, *tr.end, 4, 1)
	expect(t, tr.start.node == tr.end.node, "t.start.node == t.end.node")
	expect(t, !tr.start.garbage, "!t.start.garbage")
}

// Zig: test "term51: Untracked.track: pins on a pruned page are marked garbage"
func TestHighlight_term51_Untracked_track_pins_on_a_pruned_page_are_marked_garbage(t *testing.T) {
	testStdCapacity(t)
	// With a minimal scrollback limit only two pages are kept.
	s := ssScreenInit(t, 200, 5, 1)
	defer s.Deinit()
	for s.pages.TotalPages() < 2 {
		_, err := s.pages.Grow()
		zigTry(t, err)
	}
	n := ssYFirstRows(s)
	old_last := s.pages.pages.last
	expect(t, old_last != nil, "s.pages.pages.last.?")

	hl := HighlightUntracked{start: ssYPin(t, s, 3, 2), end: ssYPin(t, s, 4, n)}
	tr, err := hl.Track(s)
	zigTry(t, err)
	defer tr.Deinit(s)
	expect(t, !tr.start.garbage, "!t.start.garbage")

	for grown := uint32(0); !tr.start.garbage; grown++ {
		expect(t, grown < 4*n, "grown < 4 * n")
		_, err := s.pages.Grow()
		zigTry(t, err)
	}
	// The pruned page was the first one: the start pin is at the top-left of
	// the new first page and marked garbage; the end pin was on the second
	// page which is now the first.
	expect(t, s.pages.pages.first != nil, "s.pages.pages.first.?")
	expect(t, tr.start.node == s.pages.pages.first, "t.start.node == s.pages.pages.first.?")
	expect(t, tr.start.node == old_last, "t.start.node == old_last")
	ssYExpectPin(t, s, *tr.start, 0, 0)
	expect(t, !tr.end.garbage, "!t.end.garbage")
	expect(t, tr.end.node == old_last, "t.end.node == old_last")
	ssYExpectPin(t, s, *tr.end, 4, 0)
}

// Zig: test "term51: Tracked.init: tracks both pins"
func TestHighlight_term51_Tracked_init_tracks_both_pins(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	base := s.pages.CountTrackedPins()

	a := ssYPin(t, s, 2, 1)
	b := ssYPin(t, s, 7, 4)
	tr, err := HighlightTrackedInit(s, a, b)
	zigTry(t, err)
	expectEqual(t, base+2, s.pages.CountTrackedPins())
	expect(t, tr.start != tr.end, "t.start != t.end")
	expect(t, tr.start.Eql(a), "t.start.eql(a)")
	expect(t, tr.end.Eql(b), "t.end.eql(b)")

	// A tracked highlight does not impose an order: start after end is kept.
	rev, err := HighlightTrackedInit(s, b, a)
	zigTry(t, err)
	expectEqual(t, base+4, s.pages.CountTrackedPins())
	ssYExpectPin(t, s, *rev.start, 7, 4)
	ssYExpectPin(t, s, *rev.end, 2, 1)

	// A one-cell highlight still has two separate pins.
	one, err := HighlightTrackedInit(s, a, a)
	zigTry(t, err)
	expectEqual(t, base+6, s.pages.CountTrackedPins())
	expect(t, one.start != one.end, "one.start != one.end")
	expect(t, one.start.Eql(*one.end), "one.start.eql(one.end.*)")

	one.Deinit(s)
	rev.Deinit(s)
	tr.Deinit(s)
	expectEqual(t, base, s.pages.CountTrackedPins())
}

// Zig: test "term51: Tracked.init: first and last cells of the screen"
func TestHighlight_term51_Tracked_init_first_and_last_cells_of_the_screen(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	tr, err := HighlightTrackedInit(s, ssYPin(t, s, 0, 0), ssYPin(t, s, 9, 9))
	zigTry(t, err)
	defer tr.Deinit(s)
	ssYExpectPin(t, s, *tr.start, 0, 0)
	ssYExpectPin(t, s, *tr.end, 9, 9)
}

// Zig: test "term51: Tracked.init: pins follow erased rows"
func TestHighlight_term51_Tracked_init_pins_follow_erased_rows(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 3, 1000)
	defer s.Deinit()
	zigTry(t, s.TestWriteString([]uint8("L0\nL1\nL2\nL3\nL4\nL5\nL6\nL7")))
	expectEqual(t, uint(8), s.pages.total_rows)

	tr, err := HighlightTrackedInit(s, ssYPin(t, s, 4, 2), ssYPin(t, s, 6, 5))
	zigTry(t, err)
	defer tr.Deinit(s)
	// Erase history rows 0..1.
	s.pages.EraseRows(ssScreen(0, 0), Some(ssScreen(0, 1)))
	ssYExpectPin(t, s, *tr.start, 4, 0)
	ssYExpectPin(t, s, *tr.end, 6, 3)
	// Erase history rows 0..2: the start is inside, the end moves.
	s.pages.EraseRows(ssScreen(0, 0), Some(ssScreen(0, 2)))
	ssYExpectPin(t, s, *tr.start, 0, 0)
	ssYExpectPin(t, s, *tr.end, 6, 0)
}

// Zig: test "term51: Tracked.initAssume: wraps the given pointers without tracking"
func TestHighlight_term51_Tracked_initAssume_wraps_the_given_pointers_without_tracking(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	base := s.pages.CountTrackedPins()

	// The caller is responsible for the pins: here we track them ourselves.
	start, err := s.pages.TrackPin(ssYPin(t, s, 2, 1))
	zigTry(t, err)
	defer s.pages.UntrackPin(start)
	end, err := s.pages.TrackPin(ssYPin(t, s, 7, 4))
	zigTry(t, err)
	defer s.pages.UntrackPin(end)
	expectEqual(t, base+2, s.pages.CountTrackedPins())

	tr := HighlightTrackedInitAssume(start, end)
	expect(t, tr.start == start, "t.start == start")
	expect(t, tr.end == end, "t.end == end")
	// Nothing was tracked or untracked by initAssume.
	expectEqual(t, base+2, s.pages.CountTrackedPins())

	// Writes through the highlight reach the original pins.
	tr.end.x = 9
	ssYExpectPin(t, s, *end, 9, 4)
	ssYExpectPin(t, s, *start, 2, 1)
}

// Zig: test "term51: Tracked.initAssume: the same pointer may be used twice"
func TestHighlight_term51_Tracked_initAssume_the_same_pointer_may_be_used_twice(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	p, err := s.pages.TrackPin(ssYPin(t, s, 3, 3))
	zigTry(t, err)
	defer s.pages.UntrackPin(p)
	tr := HighlightTrackedInitAssume(p, p)
	expect(t, tr.start == tr.end, "t.start == t.end")
	tr.start.y = 6
	ssYExpectPin(t, s, *tr.end, 3, 6)
}

// Zig: test "term51: Tracked.deinit: untracks both pins"
func TestHighlight_term51_Tracked_deinit_untracks_both_pins(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	base := s.pages.CountTrackedPins()

	tr, err := HighlightTrackedInit(s, ssYPin(t, s, 2, 1), ssYPin(t, s, 7, 4))
	zigTry(t, err)
	expectEqual(t, base+2, s.pages.CountTrackedPins())
	tr.Deinit(s)
	expectEqual(t, base, s.pages.CountTrackedPins())
	for _, p := range s.pages.TrackedPins() {
		expect(t, p != tr.start && p != tr.end, "p != t.start and p != t.end")
	}

	// Also for highlights made by Untracked.track.
	hl := HighlightUntracked{start: ssYPin(t, s, 0, 0), end: ssYPin(t, s, 9, 9)}
	t2, err := hl.Track(s)
	zigTry(t, err)
	expectEqual(t, base+2, s.pages.CountTrackedPins())
	t2.Deinit(s)
	expectEqual(t, base, s.pages.CountTrackedPins())
}

// Zig: test "term51: Tracked.deinit: pins of a one-cell highlight on the last page"
func TestHighlight_term51_Tracked_deinit_pins_of_a_one_cell_highlight_on_the_last_page(t *testing.T) {
	testStdCapacity(t)
	s := ssYPages(t, 2)
	defer s.Deinit()
	base := s.pages.CountTrackedPins()
	p := ssYPin(t, s, 199, uint32(s.pages.total_rows-1))
	tr, err := HighlightTrackedInit(s, p, p)
	zigTry(t, err)
	expectEqual(t, base+2, s.pages.CountTrackedPins())
	tr.Deinit(s)
	expectEqual(t, base, s.pages.CountTrackedPins())
}

// Zig: test "term51: Flattened.empty: no chunks and zero bounds"
func TestHighlight_term51_Flattened_empty_no_chunks_and_zero_bounds(t *testing.T) {
	testStdCapacity(t)
	f := HighlightFlattenedEmpty
	expectEqual(t, 0, len(f.chunks))
	expectEqual(t, SizeCellCountInt(0), f.top_x)
	expectEqual(t, SizeCellCountInt(0), f.bot_x)
	// Nothing was allocated, so deinit and clone are leak-free.
	c, err := f.Clone()
	zigTry(t, err)
	expectEqual(t, 0, len(c.chunks))
	c.Deinit()
	f.Deinit()
}

// Zig: test "term51: Flattened.Chunk: fields"
func TestHighlight_term51_Flattened_Chunk_fields(t *testing.T) {
	testStdCapacity(t)
	info := reflect.TypeOf(HighlightFlattenedChunk{})
	expectEqual(t, 4, info.NumField())
	expectEqualStrings(t, "node", info.Field(0).Name)
	expectEqualStrings(t, "serial", info.Field(1).Name)
	expectEqualStrings(t, "start", info.Field(2).Name)
	expectEqualStrings(t, "end", info.Field(3).Name)
	expect(t, info.Field(1).Type == reflect.TypeOf(uint64(0)), "info.fields[1].type == u64")
	expect(t, info.Field(2).Type == reflect.TypeOf(SizeCellCountInt(0)), "info.fields[2].type == size.CellCountInt")
	expect(t, info.Field(3).Type == reflect.TypeOf(SizeCellCountInt(0)), "info.fields[3].type == size.CellCountInt")
}

// Zig: test "term51: Flattened.init: chunks per page (via an equivalent helper, init does not compile)"
func TestHighlight_term51_Flattened_init_chunks_per_page_via_an_equivalent_helper_init_does_not_compile(t *testing.T) {
	testStdCapacity(t)
	// Flattened.init itself cannot be instantiated: it initialises a field
	// `end_x` that does not exist (the field is `bot_x`) and appends a
	// `serial` to a MultiArrayList(PageChunk). ssHlFlatten does what it
	// intends, so the chunk layout can still be pinned down.
	s := ssYPages(t, 3)
	defer s.Deinit()
	n := ssYFirstRows(s)
	first := s.pages.pages.first
	expect(t, first != nil, "s.pages.pages.first.?")
	mid := first.next
	expect(t, mid != nil, "first.next.?")
	last := s.pages.pages.last
	expect(t, last != nil, "s.pages.pages.last.?")
	expect(t, mid != last, "mid != last")
	m := uint32(mid.data.size.rows)

	// One page, several rows.
	{
		f := ssHlFlatten(t, ssYPin(t, s, 3, 2), ssYPin(t, s, 1, 4))
		defer f.Deinit()
		expectEqual(t, 1, len(f.chunks))
		sl := f.chunks
		expect(t, sl[0].node == first, "sl.items(.node)[0] == first")
		expectEqual(t, first.serial, sl[0].serial)
		expectEqual(t, SizeCellCountInt(2), sl[0].start)
		expectEqual(t, SizeCellCountInt(5), sl[0].end)
		expectEqual(t, SizeCellCountInt(3), f.top_x)
		expectEqual(t, SizeCellCountInt(1), f.bot_x)
	}

	// One cell: one chunk of one row.
	{
		f := ssHlFlatten(t, ssYPin(t, s, 5, 7), ssYPin(t, s, 5, 7))
		defer f.Deinit()
		expectEqual(t, 1, len(f.chunks))
		sl := f.chunks
		expectEqual(t, SizeCellCountInt(7), sl[0].start)
		expectEqual(t, SizeCellCountInt(8), sl[0].end)
	}

	// Three pages: partial first, full middle, partial last.
	{
		f := ssHlFlatten(t, ssYPin(t, s, 4, n-1), ssYPin(t, s, 2, n+m+2))
		defer f.Deinit()
		expectEqual(t, 3, len(f.chunks))
		sl := f.chunks
		expect(t, sl[0].node == first, "sl.items(.node)[0] == first")
		expect(t, sl[1].node == mid, "sl.items(.node)[1] == mid")
		expect(t, sl[2].node == last, "sl.items(.node)[2] == last")
		expectEqual(t, first.serial, sl[0].serial)
		expectEqual(t, mid.serial, sl[1].serial)
		expectEqual(t, last.serial, sl[2].serial)
		expect(t, first.serial != mid.serial && mid.serial != last.serial, "first.serial != mid.serial and mid.serial != last.serial")
		expectEqual(t, SizeCellCountInt(n-1), sl[0].start)
		expectEqual(t, SizeCellCountInt(n), sl[0].end)
		expectEqual(t, SizeCellCountInt(0), sl[1].start)
		expectEqual(t, mid.data.size.rows, sl[1].end)
		expectEqual(t, SizeCellCountInt(0), sl[2].start)
		expectEqual(t, SizeCellCountInt(3), sl[2].end)
		expectEqual(t, SizeCellCountInt(4), f.top_x)
		expectEqual(t, SizeCellCountInt(2), f.bot_x)
	}
}

// Zig: test "term51: Flattened.deinit: releases the chunk list"
func TestHighlight_term51_Flattened_deinit_releases_the_chunk_list(t *testing.T) {
	testStdCapacity(t)
	s := ssYPages(t, 2)
	defer s.Deinit()
	n := ssYFirstRows(s)
	f := ssHlFlatten(t, ssYPin(t, s, 0, 0), ssYPin(t, s, 5, n+2))
	expectEqual(t, 2, len(f.chunks))
	// The testing allocator fails the test if this does not free the list.
	f.Deinit()
}

// Zig: test "term51: Flattened.clone: independent copy with the same chunks and bounds"
func TestHighlight_term51_Flattened_clone_independent_copy_with_the_same_chunks_and_bounds(t *testing.T) {
	testStdCapacity(t)
	s := ssYPages(t, 2)
	defer s.Deinit()
	n := ssYFirstRows(s)
	f := ssHlFlatten(t, ssYPin(t, s, 3, n-2), ssYPin(t, s, 1, n+1))
	defer f.Deinit()

	c, err := f.Clone()
	zigTry(t, err)
	defer c.Deinit()
	expectEqual(t, len(f.chunks), len(c.chunks))
	expectEqual(t, f.top_x, c.top_x)
	expectEqual(t, f.bot_x, c.bot_x)
	for i := 0; i < len(f.chunks); i++ {
		expect(t, f.chunks[i].node == c.chunks[i].node, "f.chunks.items(.node)[i] == c.chunks.items(.node)[i]")
		expectEqual(t, f.chunks[i].serial, c.chunks[i].serial)
		expectEqual(t, f.chunks[i].start, c.chunks[i].start)
		expectEqual(t, f.chunks[i].end, c.chunks[i].end)
	}
	expect(t, c.Untracked().Eql(f.Untracked()), "c.untracked().eql(f.untracked())")

	// Separate storage: changing the clone leaves the original alone.
	expect(t, &f.chunks[0] != &c.chunks[0], "f.chunks.items(.start).ptr != c.chunks.items(.start).ptr")
	c.chunks[0].start = 0
	c.top_x = 9
	c.bot_x = 8
	expectEqual(t, SizeCellCountInt(n-2), f.chunks[0].start)
	expectEqual(t, SizeCellCountInt(3), f.top_x)
	expectEqual(t, SizeCellCountInt(1), f.bot_x)
}

// Zig: test "term51: Flattened.startPin: first row of the first chunk at top_x"
func TestHighlight_term51_Flattened_startPin_first_row_of_the_first_chunk_at_top_x(t *testing.T) {
	testStdCapacity(t)
	s := ssYPages(t, 2)
	defer s.Deinit()
	n := ssYFirstRows(s)

	// Within one page.
	one := ssHlFlatten(t, ssYPin(t, s, 3, 2), ssYPin(t, s, 1, 4))
	defer one.Deinit()
	p := one.StartPin()
	ssYExpectPin(t, s, p, 3, 2)
	expect(t, p.node == s.pages.pages.first, "p.node == s.pages.pages.first.?")
	expect(t, !p.garbage, "!p.garbage")
	expect(t, p.Eql(ssYPin(t, s, 3, 2)), "p.eql(term51YPin(&s, 3, 2))")

	// Across pages, starting on the last row of the first page.
	two := ssHlFlatten(t, ssYPin(t, s, 199, n-1), ssYPin(t, s, 0, n))
	defer two.Deinit()
	q := two.StartPin()
	ssYExpectPin(t, s, q, 199, n-1)
	expect(t, q.node == s.pages.pages.first, "q.node == s.pages.pages.first.?")

	// First cell of the screen.
	origin := ssHlFlatten(t, ssYPin(t, s, 0, 0), ssYPin(t, s, 0, 0))
	defer origin.Deinit()
	ssYExpectPin(t, s, origin.StartPin(), 0, 0)
}

// Zig: test "term51: Flattened.endPin: last row of the last chunk at bot_x"
func TestHighlight_term51_Flattened_endPin_last_row_of_the_last_chunk_at_bot_x(t *testing.T) {
	testStdCapacity(t)
	s := ssYPages(t, 2)
	defer s.Deinit()
	n := ssYFirstRows(s)

	one := ssHlFlatten(t, ssYPin(t, s, 3, 2), ssYPin(t, s, 1, 4))
	defer one.Deinit()
	p := one.EndPin()
	// end is exclusive in the chunk, endPin converts it back to the last row.
	ssYExpectPin(t, s, p, 1, 4)
	expect(t, p.node == s.pages.pages.first, "p.node == s.pages.pages.first.?")
	expect(t, p.Eql(ssYPin(t, s, 1, 4)), "p.eql(term51YPin(&s, 1, 4))")

	// Ends on the first row of the second page.
	two := ssHlFlatten(t, ssYPin(t, s, 3, n-1), ssYPin(t, s, 6, n))
	defer two.Deinit()
	q := two.EndPin()
	ssYExpectPin(t, s, q, 6, n)
	expect(t, q.node == s.pages.pages.last, "q.node == s.pages.pages.last.?")
	expectEqual(t, SizeCellCountInt(0), q.y)

	// Ends on the last row of the first page.
	three := ssHlFlatten(t, ssYPin(t, s, 3, n-3), ssYPin(t, s, 9, n-1))
	defer three.Deinit()
	r := three.EndPin()
	ssYExpectPin(t, s, r, 9, n-1)
	expect(t, r.node == s.pages.pages.first, "r.node == s.pages.pages.first.?")

	// Last cell of the screen.
	last_y := uint32(s.pages.total_rows - 1)
	tail := ssHlFlatten(t, ssYPin(t, s, 199, last_y), ssYPin(t, s, 199, last_y))
	defer tail.Deinit()
	ssYExpectPin(t, s, tail.EndPin(), 199, last_y)
}

// Zig: test "term51: Flattened.endPin: bot_x may be left of top_x"
func TestHighlight_term51_Flattened_endPin_bot_x_may_be_left_of_top_x(t *testing.T) {
	testStdCapacity(t)
	s := ssScreenInit(t, 10, 10, 0)
	defer s.Deinit()
	// Start right of the end on a higher row: allowed for a flattened highlight.
	f := ssHlFlatten(t, ssYPin(t, s, 8, 1), ssYPin(t, s, 2, 5))
	defer f.Deinit()
	expect(t, f.bot_x < f.top_x, "f.bot_x < f.top_x")
	ssYExpectPin(t, s, f.StartPin(), 8, 1)
	ssYExpectPin(t, s, f.EndPin(), 2, 5)
	// The pins are still ordered: start is before end.
	expect(t, f.StartPin().Before(f.EndPin()), "f.startPin().before(f.endPin())")
}

// Zig: test "term51: Flattened.untracked: start and end pins of the highlight"
func TestHighlight_term51_Flattened_untracked_start_and_end_pins_of_the_highlight(t *testing.T) {
	testStdCapacity(t)
	s := ssYPages(t, 3)
	defer s.Deinit()
	n := ssYFirstRows(s)
	expect(t, s.pages.pages.first.next != nil, "s.pages.pages.first.?.next.?")
	m := uint32(s.pages.pages.first.next.data.size.rows)

	// One page.
	{
		a := ssYPin(t, s, 3, 2)
		b := ssYPin(t, s, 1, 4)
		f := ssHlFlatten(t, a, b)
		defer f.Deinit()
		u := f.Untracked()
		expect(t, u.start.Eql(a), "u.start.eql(a)")
		expect(t, u.end.Eql(b), "u.end.eql(b)")
		expect(t, u.Eql(HighlightUntracked{start: a, end: b}), "u.eql(.{ .start = a, .end = b })")
		expect(t, !u.start.garbage && !u.end.garbage, "!u.start.garbage and !u.end.garbage")
		// Same as the individual accessors.
		expect(t, u.start.Eql(f.StartPin()), "u.start.eql(f.startPin())")
		expect(t, u.end.Eql(f.EndPin()), "u.end.eql(f.endPin())")
	}

	// Three pages.
	{
		a := ssYPin(t, s, 4, n-1)
		b := ssYPin(t, s, 2, n+m+2)
		f := ssHlFlatten(t, a, b)
		defer f.Deinit()
		u := f.Untracked()
		expect(t, u.start.Eql(a), "u.start.eql(a)")
		expect(t, u.end.Eql(b), "u.end.eql(b)")
		expect(t, u.start.node == s.pages.pages.first, "u.start.node == s.pages.pages.first.?")
		expect(t, u.end.node == s.pages.pages.last, "u.end.node == s.pages.pages.last.?")
		expect(t, u.start.Before(u.end), "u.start.before(u.end)")
		expect(t, u.start.node != u.end.node, "u.start.node != u.end.node")
	}

	// One cell.
	{
		a := ssYPin(t, s, 7, 9)
		f := ssHlFlatten(t, a, a)
		defer f.Deinit()
		u := f.Untracked()
		expect(t, u.start.Eql(a), "u.start.eql(a)")
		expect(t, u.end.Eql(a), "u.end.eql(a)")
	}
}

// Zig: test "term51: Flattened.untracked: can be tracked again"
func TestHighlight_term51_Flattened_untracked_can_be_tracked_again(t *testing.T) {
	testStdCapacity(t)
	s := ssYPages(t, 2)
	defer s.Deinit()
	n := ssYFirstRows(s)
	base := s.pages.CountTrackedPins()

	f := ssHlFlatten(t, ssYPin(t, s, 3, n-2), ssYPin(t, s, 1, n+1))
	defer f.Deinit()
	u := f.Untracked()
	tr, err := u.Track(s)
	zigTry(t, err)
	expectEqual(t, base+2, s.pages.CountTrackedPins())
	ssYExpectPin(t, s, *tr.start, 3, n-2)
	ssYExpectPin(t, s, *tr.end, 1, n+1)
	tr.Deinit(s)
	expectEqual(t, base, s.pages.CountTrackedPins())
}

// Go-only: panic test: Flattened.startPin: no chunks (`slice.items(.node)[0]` is out of bounds)
func TestHighlight_panic_Flattened_startPin_no_chunks(t *testing.T) {
	testStdCapacity(t)
	f := HighlightFlattenedEmpty
	ssExpectPanic(t, func() { _ = f.StartPin() })
}

// Go-only: panic test: Flattened.endPin: no chunks (`slice.len - 1` underflows)
func TestHighlight_panic_Flattened_endPin_no_chunks(t *testing.T) {
	testStdCapacity(t)
	f := HighlightFlattenedEmpty
	ssExpectPanic(t, func() { _ = f.EndPin() })
}

// Go-only: panic test: Flattened.untracked: no chunks (`nodes[0]` is out of bounds)
func TestHighlight_panic_Flattened_untracked_no_chunks(t *testing.T) {
	testStdCapacity(t)
	f := HighlightFlattenedEmpty
	ssExpectPanic(t, func() { _ = f.Untracked() })
}
