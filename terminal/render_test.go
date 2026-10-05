// Tests translated one for one from src/terminal/render.zig (Ghostty v1.3.1
// plus the term51 additional tests): the RenderState tests. Each Go test
// carries the exact Zig title in a "// Zig: test" comment. Helpers are
// prefixed rn.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualStrings / expectEqualSlices keep Zig's names
//     and Zig's (expected, actual) argument order. `try` on a call that
//     returns an error is zigTry(t, err); `x.?` is zigUnwrap. The testing
//     parameter is `t`, so the Terminal that Zig's tests call `t` is called
//     `term`.
//   - `var t = try Terminal.init(alloc, .{ .cols = c, .rows = r })` is
//     `term := rnInit(t, c, r)`, which starts from terminalOptionsDefault
//     (max_scrollback 10_000, the default colors and modes); other options
//     go through rnInitOpts. `var state: RenderState = .empty` is
//     `state := RenderStateEmpty`.
//   - `state.row_data.items(.x)[y]` is `state.row_data[y].x`,
//     `row_data.slice()` is `state.row_data`, `cells.get(x)` is
//     `cells[x]`, `row_data.capacity` is `cap(state.row_data)`.
//   - `term51ZRsWrite` is rnWrite, `term51ZRsClean` is rnClean and so on.
//     `term51ZRsFlat` (which builds a highlight.Flattened by hand because
//     Flattened.init does not compile upstream) is rnFlat, which builds
//     HighlightFlattened values by hand.
//   - Zig's `alloc` arguments are dropped; `defer ...deinit(alloc)` leak
//     checks of values that are plain Go memory are dropped with
//     `// Zig-only:` comments. The arena (`Row.arena`) is dropped.
//   - Every test calls testStdCapacity first (Zig's unit tests run with
//     grapheme_bytes 512).
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt.

package terminal

import (
	"errors"
	"testing"
)

// rnExpectPanic runs f, which must panic. A panic with the wireframe's
// "not implemented" value does not count: it only means the function under
// test does not exist yet. The setup is done outside f.
func rnExpectPanic(t *testing.T, f func()) {
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

// rnInt is the integer types the point helpers take, so that untyped
// constants, loop variables and size values all work as in Zig.
type rnInt interface {
	~int | ~uint | ~uint16 | ~uint32
}

// rnActive, rnViewport and rnScreen are the Zig literals
// `.{ .active = .{ .x = x, .y = y } }` and so on.
func rnActive[X, Y rnInt](x X, y Y) Point {
	return Point{Tag: PointTagActive, Active: PointCoordinate{x: SizeCellCountInt(x), y: uint32(y)}}
}

func rnViewport[X, Y rnInt](x X, y Y) Point {
	return Point{Tag: PointTagViewport, Viewport: PointCoordinate{x: SizeCellCountInt(x), y: uint32(y)}}
}

func rnScreen[X, Y rnInt](x X, y Y) Point {
	return Point{Tag: PointTagScreen, Screen: PointCoordinate{x: SizeCellCountInt(x), y: uint32(y)}}
}

// rnCoord is the Zig literal `.{ .x = x, .y = y }` of a point.Coordinate.
func rnCoord[X, Y rnInt](x X, y Y) PointCoordinate {
	return PointCoordinate{x: SizeCellCountInt(x), y: uint32(y)}
}

// rnInitOpts is `try Terminal.init(alloc, opts)`; the terminal is returned by
// pointer (the methods take a pointer receiver).
func rnInitOpts(tt *testing.T, opts TerminalOptions) *Terminal {
	tt.Helper()
	term, err := TerminalInit(opts)
	zigTry(tt, err)
	return &term
}

// rnInit is `try Terminal.init(alloc, .{ .cols = cols, .rows = rows })`.
func rnInit(tt *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) *Terminal {
	tt.Helper()
	return rnInitOpts(tt, terminalOptionsDefault(cols, rows))
}

// rnInitScrollback is `try Terminal.init(alloc, .{ .cols = cols, .rows =
// rows, .max_scrollback = max_scrollback })`.
func rnInitScrollback(tt *testing.T, cols SizeCellCountInt, rows SizeCellCountInt, max_scrollback uint64) *Terminal {
	tt.Helper()
	opts := terminalOptionsDefault(cols, rows)
	opts.max_scrollback = max_scrollback
	return rnInitOpts(tt, opts)
}

// rnPtrNode is the Zig `x.?` on an optional node pointer.
func rnPtrNode(tt *testing.T, node *PageListNode, zig string) *PageListNode {
	tt.Helper()
	expect(tt, node != nil, zig)
	return node
}

// rnWrite is term51ZRsWrite: feed data to the terminal through a fresh VT
// stream.
func rnWrite(tt *testing.T, term *Terminal, data string) {
	tt.Helper()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(tt, s.NextSlice([]uint8(data)))
}

// rnMemsetDirty is `@memset(state.row_data.items(.dirty), false)`.
func rnMemsetDirty(state *RenderState) {
	for y := range state.row_data {
		state.row_data[y].dirty = false
	}
}

// rnClean is term51ZRsClean: pretend the renderer consumed everything:
// dirty false, all rows clean.
func rnClean(state *RenderState) {
	state.dirty = RenderStateDirtyFalse
	rnMemsetDirty(state)
}

// rnExpectRowDirty is term51ZRsExpectRowDirty.
func rnExpectRowDirty(tt *testing.T, state *RenderState, expected []bool) {
	tt.Helper()
	got := make([]bool, len(state.row_data))
	for y := range state.row_data {
		got[y] = state.row_data[y].dirty
	}
	expectEqualSlices(tt, expected, got)
}

// rnAllRows is term51ZRsAllRows.
func rnAllRows(tt *testing.T, state *RenderState, value bool) {
	tt.Helper()
	for _, row := range state.row_data {
		expectEqual(tt, value, row.dirty)
	}
}

// rnCell is term51ZRsCell.
func rnCell(state *RenderState, x uint, y uint) RenderStateCell {
	return state.row_data[y].cells[x]
}

// rnFillTwoPages is term51ZRsFillTwoPages: fill a terminal so that the
// viewport (10 rows) spans two pages: the last 5 rows of the first page and
// the first 5 rows of the second, with an "X" on the first row of the second
// page.
func rnFillTwoPages(tt *testing.T, term *Terminal) {
	tt.Helper()
	s := term.VtStream()
	defer s.Deinit()
	pages := &term.screens.active.pages
	first_page_cap := rnPtrNode(tt, pages.pages.first, "pages.pages.first.?").data.capacity.rows
	for range int(first_page_cap) - 1 {
		zigTry(tt, s.NextSlice([]uint8("\r\n")))
	}
	zigTry(tt, s.NextSlice([]uint8("\r\n")))
	zigTry(tt, s.NextSlice([]uint8("X")))
	for range 4 {
		zigTry(tt, s.NextSlice([]uint8("\r\n")))
	}
	expect(tt, pages.pages.first != pages.pages.last, "pages.pages.first != pages.pages.last")
}

// rnSelBounds is `?[2]size.CellCountInt`, the type of Row.selection.
type rnSelBounds = Optional[[2]SizeCellCountInt]

// rnNil is the Zig `null` of a selection range.
var rnNil rnSelBounds

// rnSel is the Zig literal `.{ a, b }` of a selection range.
func rnSel(a SizeCellCountInt, b SizeCellCountInt) rnSelBounds {
	return Some([2]SizeCellCountInt{a, b})
}

// rnExpectSels is term51ZRsExpectSels. A null element is compared by its
// Valid flag only, as Zig compares optionals.
func rnExpectSels(tt *testing.T, state *RenderState, expected []rnSelBounds) {
	tt.Helper()
	if len(state.row_data) != len(expected) {
		tt.Fatalf("got %d rows, want %d", len(state.row_data), len(expected))
	}
	for y := range expected {
		got := state.row_data[y].selection
		want := expected[y]
		if got.Valid != want.Valid || (want.Valid && got.Value != want.Value) {
			tt.Fatalf("selection of row %d: got %v, want %v", y, got, want)
		}
	}
}

// rnPin is term51ZRsPin: the pin of an active point.
func rnPin(tt *testing.T, term *Terminal, x SizeCellCountInt, y SizeCellCountInt) PageListPin {
	tt.Helper()
	return zigUnwrap(tt, term.screens.active.pages.Pin(rnActive(x, y)))
}

// rnFlat is term51ZRsFlat: a highlight.Flattened built by hand.
func rnFlat(top_x SizeCellCountInt, bot_x SizeCellCountInt, chunks []HighlightFlattenedChunk) HighlightFlattened {
	var hl HighlightFlattened
	// Zig-only: errdefer hl.deinit(alloc);
	hl.chunks = append(hl.chunks, chunks...)
	hl.top_x = top_x
	hl.bot_x = bot_x
	return hl
}

// rnHighlights is term51ZRsHighlights.
func rnHighlights(state *RenderState, y uint) []RenderStateHighlight {
	return state.row_data[y].highlights
}

// rnRange is `&hl.range`: the range of a highlight as a slice.
func rnRange(h RenderStateHighlight) []SizeCellCountInt {
	return h.range_[:]
}

// rnMapParam is the Zig `.{ .alloc = alloc, .map = &map }`.
func rnMapParam(m *RenderStateStringMap) Optional[RenderStateStringMapParam] {
	return Some(RenderStateStringMapParam{map_: m})
}

// rnExpectError is `try testing.expectError(want, got)`.
func rnExpectError(tt *testing.T, want error, got error) {
	tt.Helper()
	if !errors.Is(got, want) {
		tt.Fatalf("got error %v, want %v", got, want)
	}
}

// Zig: test "styled"
func TestRender_styled(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 80, 24)
	defer term.Deinit()

	// This fills the screen up
	zigTry(t, term.Decaln())

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))
}

// Zig: test "basic text"
func TestRender_basic_text(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 3)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("ABCD")))

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))

	// Verify we have the right number of rows
	row_data := state.row_data
	expectEqual(t, 3, len(row_data))

	// All rows should have cols cells
	expectEqual(t, 10, len(row_data[0].cells))
	expectEqual(t, 10, len(row_data[1].cells))
	expectEqual(t, 10, len(row_data[2].cells))

	// Row zero should contain our text
	cells := row_data[0].cells
	expectEqual(t, uint32('A'), cells[0].raw.Codepoint())
	expectEqual(t, uint32('B'), cells[1].raw.Codepoint())
	expectEqual(t, uint32('C'), cells[2].raw.Codepoint())
	expectEqual(t, uint32('D'), cells[3].raw.Codepoint())
	expectEqual(t, uint32(0), cells[4].raw.Codepoint())
}

// Zig: test "styled text"
func TestRender_styled_text(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 3)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("\x1b[1mA")))   // Bold
	zigTry(t, s.NextSlice([]uint8("\x1b[0;3mB"))) // Italic
	zigTry(t, s.NextSlice([]uint8("\x1b[0;4mC"))) // Underline

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))

	// Verify we have the right number of rows
	row_data := state.row_data
	expectEqual(t, 3, len(row_data))

	// All rows should have cols cells
	expectEqual(t, 10, len(row_data[0].cells))
	expectEqual(t, 10, len(row_data[1].cells))
	expectEqual(t, 10, len(row_data[2].cells))

	// Row zero should contain our text
	cells := row_data[0].cells
	{
		cell := cells[0]
		expectEqual(t, uint32('A'), cell.raw.Codepoint())
		expect(t, cell.style.flags.bold, "cell.style.flags.bold")
	}
	{
		cell := cells[1]
		expectEqual(t, uint32('B'), cell.raw.Codepoint())
		expect(t, !cell.style.flags.bold, "!cell.style.flags.bold")
		expect(t, cell.style.flags.italic, "cell.style.flags.italic")
	}
	expectEqual(t, uint32('C'), cells[2].raw.Codepoint())
	expectEqual(t, uint32(0), cells[3].raw.Codepoint())
}

// Zig: test "grapheme"
func TestRender_grapheme(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 3)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("A")))
	zigTry(t, s.NextSlice([]uint8("👨‍"))) // this has a ZWJ

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))

	// Verify we have the right number of rows
	row_data := state.row_data
	expectEqual(t, 3, len(row_data))

	// All rows should have cols cells
	expectEqual(t, 10, len(row_data[0].cells))
	expectEqual(t, 10, len(row_data[1].cells))
	expectEqual(t, 10, len(row_data[2].cells))

	// Row zero should contain our text
	cells := row_data[0].cells
	{
		cell := cells[0]
		expectEqual(t, uint32('A'), cell.raw.Codepoint())
	}
	{
		cell := cells[1]
		expectEqual(t, uint32(0x1F468), cell.raw.Codepoint())
		expectEqual(t, PageCellWideWide, cell.raw.wide)
		expectEqualSlices(t, []uint32{0x200D}, cell.grapheme)
	}
	{
		cell := cells[2]
		expectEqual(t, uint32(0), cell.raw.Codepoint())
		expectEqual(t, PageCellWideSpacerTail, cell.raw.wide)
	}
}

// Zig: test "cursor state in viewport"
func TestRender_cursor_state_in_viewport(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 5)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("A\x1b[H")))

	state := RenderStateEmpty
	defer state.Deinit()

	// Initial update
	zigTry(t, state.Update(term))
	expectEqual(t, 0, state.cursor.active.x)
	expectEqual(t, 0, state.cursor.active.y)
	expectEqual(t, 0, zigUnwrap(t, state.cursor.viewport).x)
	expectEqual(t, 0, zigUnwrap(t, state.cursor.viewport).y)
	expectEqual(t, uint32('A'), state.cursor.cell.Codepoint())
	expect(t, state.cursor.style.Default(), "state.cursor.style.default()")

	// Set a style on the cursor
	zigTry(t, s.NextSlice([]uint8("\x1b[1m"))) // Bold
	zigTry(t, state.Update(term))
	expect(t, !state.cursor.style.Default(), "!state.cursor.style.default()")
	expect(t, state.cursor.style.flags.bold, "state.cursor.style.flags.bold")
	zigTry(t, s.NextSlice([]uint8("\x1b[0m"))) // Reset style

	// Move cursor to 2,1
	zigTry(t, s.NextSlice([]uint8("\x1b[2;3H")))
	zigTry(t, state.Update(term))
	expectEqual(t, 2, state.cursor.active.x)
	expectEqual(t, 1, state.cursor.active.y)
	expectEqual(t, 2, zigUnwrap(t, state.cursor.viewport).x)
	expectEqual(t, 1, zigUnwrap(t, state.cursor.viewport).y)
}

// Zig: test "cursor state out of viewport"
func TestRender_cursor_state_out_of_viewport(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 2)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("A\r\nB\r\nC\r\nD\r\n")))

	state := RenderStateEmpty
	defer state.Deinit()

	// Initial update
	zigTry(t, state.Update(term))
	expectEqual(t, 0, state.cursor.active.x)
	expectEqual(t, 1, state.cursor.active.y)
	expectEqual(t, 0, zigUnwrap(t, state.cursor.viewport).x)
	expectEqual(t, 1, zigUnwrap(t, state.cursor.viewport).y)

	// Scroll the viewport
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	zigTry(t, state.Update(term))

	// Set a style on the cursor
	expectEqual(t, 0, state.cursor.active.x)
	expectEqual(t, 1, state.cursor.active.y)
	expect(t, !state.cursor.viewport.Valid, "state.cursor.viewport == null")
}

// Zig: test "dirty state"
func TestRender_dirty_state(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 5)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	state := RenderStateEmpty
	defer state.Deinit()

	// First update should trigger redraw due to resize
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyFull, state.dirty)

	// Reset dirty flag and dirty rows
	state.dirty = RenderStateDirtyFalse
	{
		rnMemsetDirty(&state)
	}

	// Second update with no changes - no dirty rows
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyFalse, state.dirty)
	{
		for _, row := range state.row_data {
			expect(t, !row.dirty, "!d")
		}
	}

	// Write to first line
	zigTry(t, s.NextSlice([]uint8("A")))
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyPartial, state.dirty)
	{
		dirty := state.row_data
		expect(t, dirty[0].dirty, "dirty[0]")   // First row dirty
		expect(t, !dirty[1].dirty, "!dirty[1]") // Second row clean
	}
}

// Zig: test "colors"
func TestRender_colors(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 5)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	state := RenderStateEmpty
	defer state.Deinit()

	// Default colors
	zigTry(t, state.Update(term))

	// Change cursor color
	zigTry(t, s.NextSlice([]uint8("\x1b]12;#FF0000\x07")))
	zigTry(t, state.Update(term))

	c := zigUnwrap(t, state.colors.cursor)
	expectEqual(t, 0xFF, c.r)
	expectEqual(t, 0, c.g)
	expectEqual(t, 0, c.b)

	// Change palette color 0 to White
	zigTry(t, s.NextSlice([]uint8("\x1b]4;0;#FFFFFF\x07")))
	zigTry(t, state.Update(term))
	p0 := state.colors.palette[0]
	expectEqual(t, 0xFF, p0.r)
	expectEqual(t, 0xFF, p0.g)
	expectEqual(t, 0xFF, p0.b)
}

// Zig: test "selection single line"
func TestRender_selection_single_line(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 3)
	defer term.Deinit()

	screen := term.screens.active
	zigTry(t, screen.Select(Some(SelectionInit(
		zigUnwrap(t, screen.pages.Pin(rnActive(0, 1))),
		zigUnwrap(t, screen.pages.Pin(rnActive(2, 1))),
		false,
	))))

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))

	sels := state.row_data
	expect(t, !sels[0].selection.Valid, "sels[0] == null")
	{
		sel1 := zigUnwrap(t, sels[1].selection)
		expectEqualSlices(t, []SizeCellCountInt{0, 2}, sel1[:])
	}
	expect(t, !sels[2].selection.Valid, "sels[2] == null")

	// Clear the selection
	zigTry(t, screen.Select(Optional[Selection]{}))
	zigTry(t, state.Update(term))
	expect(t, !sels[0].selection.Valid, "sels[0] == null")
	expect(t, !sels[1].selection.Valid, "sels[1] == null")
	expect(t, !sels[2].selection.Valid, "sels[2] == null")
}

// Zig: test "selection multiple lines"
func TestRender_selection_multiple_lines(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 3)
	defer term.Deinit()

	screen := term.screens.active
	zigTry(t, screen.Select(Some(SelectionInit(
		zigUnwrap(t, screen.pages.Pin(rnActive(0, 1))),
		zigUnwrap(t, screen.pages.Pin(rnActive(2, 2))),
		false,
	))))

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))

	sels := state.row_data
	expect(t, !sels[0].selection.Valid, "sels[0] == null")
	{
		sel1 := zigUnwrap(t, sels[1].selection)
		expectEqualSlices(t, []SizeCellCountInt{0, screen.pages.cols - 1}, sel1[:])
	}
	{
		sel2 := zigUnwrap(t, sels[2].selection)
		expectEqualSlices(t, []SizeCellCountInt{0, 2}, sel2[:])
	}
}

// Zig: test "linkCells"
func TestRender_linkCells(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 5)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	state := RenderStateEmpty
	defer state.Deinit()

	// Create a hyperlink
	zigTry(t, s.NextSlice([]uint8("\x1b]8;;http://example.com\x1b\\LINK\x1b]8;;\x1b\\")))
	zigTry(t, state.Update(term))

	// Query link at 0,0
	cells, err := state.LinkCells(rnCoord(0, 0))
	zigTry(t, err)
	defer cells.Deinit()

	expectEqual(t, 4, cells.Count())
	expect(t, cells.Contains(rnCoord(0, 0)), "cells.contains(.{ .x = 0, .y = 0 })")
	expect(t, cells.Contains(rnCoord(1, 0)), "cells.contains(.{ .x = 1, .y = 0 })")
	expect(t, cells.Contains(rnCoord(2, 0)), "cells.contains(.{ .x = 2, .y = 0 })")
	expect(t, cells.Contains(rnCoord(3, 0)), "cells.contains(.{ .x = 3, .y = 0 })")

	// Query no link
	cells2, err := state.LinkCells(rnCoord(4, 0))
	zigTry(t, err)
	defer cells2.Deinit()
	expectEqual(t, 0, cells2.Count())
}

// Zig: test "string"
func TestRender_string(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 5, 2)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("AB")))

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))

	w := oscWriterAllocatingInit()
	defer w.deinit()

	zigTry(t, state.String(&w.writer, Optional[RenderStateStringMapParam]{}))

	result, err := w.toOwnedSlice()
	zigTry(t, err)
	// Zig-only: defer alloc.free(result);

	expected := "AB\x00\x00\x00\n\x00\x00\x00\x00\x00\n"
	expectEqualStrings(t, expected, string(result))
}

// Zig: test "linkCells with scrollback spanning pages"
func TestRender_linkCells_with_scrollback_spanning_pages(t *testing.T) {
	testStdCapacity(t)
	const viewport_rows SizeCellCountInt = 10
	const tail_rows SizeCellCountInt = 5

	opts := terminalOptionsDefault(PageStdCapacity.cols, viewport_rows)
	opts.max_scrollback = 10_000
	term := rnInitOpts(t, opts)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	pages := &term.screens.active.pages
	first_page_cap := rnPtrNode(t, pages.pages.first, "pages.pages.first.?").data.capacity.rows

	// Fill first page
	for range int(first_page_cap) - 1 {
		zigTry(t, s.NextSlice([]uint8("\r\n")))
	}

	// Create second page with hyperlink
	zigTry(t, s.NextSlice([]uint8("\r\n")))
	zigTry(t, s.NextSlice([]uint8("\x1b]8;;http://example.com\x1b\\LINK\x1b]8;;\x1b\\")))
	for range int(tail_rows) - 1 {
		zigTry(t, s.NextSlice([]uint8("\r\n")))
	}

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))

	expected_viewport_y := uint(viewport_rows - tail_rows)
	// BUG: This crashes without the fix
	cells, err := state.LinkCells(rnCoord(0, expected_viewport_y))
	zigTry(t, err)
	defer cells.Deinit()
	expectEqual(t, uint(4), cells.Count())
}

// Zig: test "linkCells with invalid viewport point"
func TestRender_linkCells_with_invalid_viewport_point(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 5)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))

	// Row out of bound
	{
		cells, err := state.LinkCells(rnCoord(0, term.rows+10))
		zigTry(t, err)
		defer cells.Deinit()
		expectEqual(t, 0, cells.Count())
	}

	// Col out of bound
	{
		cells, err := state.LinkCells(rnCoord(term.cols+10, 0))
		zigTry(t, err)
		defer cells.Deinit()
		expectEqual(t, 0, cells.Count())
	}
}

// Zig: test "dirty row resets highlights"
func TestRender_dirty_row_resets_highlights(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 3)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()
	zigTry(t, s.NextSlice([]uint8("ABC")))

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))

	// Reset dirty state
	state.dirty = RenderStateDirtyFalse
	{
		rnMemsetDirty(&state)
	}

	// Manually add a highlight to row 0
	{
		row_data := state.row_data
		// Zig-only: var arena = row_arenas[0].promote(alloc);
		// Zig-only: defer row_arenas[0] = arena.state;
		row_data[0].highlights = append(row_data[0].highlights, RenderStateHighlight{
			tag:    1,
			range_: [2]SizeCellCountInt{0, 2},
		})
	}

	// Verify we have a highlight
	{
		row_data := state.row_data
		expectEqual(t, 1, len(row_data[0].highlights))
	}

	// Write to row 0 to make it dirty
	zigTry(t, s.NextSlice([]uint8("\x1b[H"))) // Move to home
	zigTry(t, s.NextSlice([]uint8("X")))
	zigTry(t, state.Update(term))

	// Verify the highlight was reset on the dirty row
	{
		row_data := state.row_data
		expectEqual(t, 0, len(row_data[0].highlights))
	}
}

// Zig: test "term51: RenderState.update: initial state and first update"
func TestRender_term51_RenderState_update_initial_state_and_first_update(t *testing.T) {
	testStdCapacity(t)
	state := RenderStateEmpty
	defer state.Deinit()

	// The empty state.
	expectEqual(t, SizeCellCountInt(0), state.rows)
	expectEqual(t, SizeCellCountInt(0), state.cols)
	expectEqual(t, RenderStateDirtyFalse, state.dirty)
	expectEqual(t, ScreenSetKeyPrimary, state.screen)
	expect(t, !state.viewport_pin.Valid, "state.viewport_pin == null")
	expect(t, !state.selection_cache.Valid, "state.selection_cache == null")
	expectEqual(t, 0, len(state.row_data))
	expect(t, !state.cursor.viewport.Valid, "state.cursor.viewport == null")
	expect(t, state.cursor.cell.IsZero(), "state.cursor.cell.isZero()")
	expectEqual(t, CursorStyleBlock, state.cursor.visual_style)
	expect(t, !state.cursor.password_input, "!state.cursor.password_input")
	expect(t, state.cursor.visible, "state.cursor.visible")
	expect(t, !state.cursor.blinking, "!state.cursor.blinking")
	expect(t, !state.colors.cursor.Valid, "state.colors.cursor == null")
	expectEqualSlices(t, ColorDefault[:], state.colors.palette[:])

	// The Dirty enum order.
	expectEqual(t, uint8(0), uint8(RenderStateDirtyFalse))
	expectEqual(t, uint8(1), uint8(RenderStateDirtyPartial))
	expectEqual(t, uint8(2), uint8(RenderStateDirtyFull))

	term := rnInit(t, 10, 4)
	defer term.Deinit()
	// A soft wrapped row is not needed: use hard lines and a wrap marker.
	rnWrite(t, term, "ab\r\n\r\ncd")
	term.flags.dirty.palette = true
	term.screens.active.dirty.selection = true

	zigTry(t, state.Update(term))
	expectEqual(t, SizeCellCountInt(4), state.rows)
	expectEqual(t, SizeCellCountInt(10), state.cols)
	expectEqual(t, RenderStateDirtyFull, state.dirty)
	expectEqual(t, ScreenSetKeyPrimary, state.screen)
	expect(t, state.viewport_pin.Valid, "state.viewport_pin != null")
	expect(t, zigUnwrap(t, state.viewport_pin).Eql(term.screens.active.pages.GetTopLeft(PointTagViewport)), "state.viewport_pin.?.eql(t.screens.active.pages.getTopLeft(.viewport))")
	expectEqual(t, 4, len(state.row_data))
	rnAllRows(t, &state, true)

	// The terminal and screen dirty flags were consumed.
	expect(t, !term.flags.dirty.palette, "!t.flags.dirty.palette")
	expect(t, !term.screens.active.dirty.selection, "!t.screens.active.dirty.selection")

	// Each row has `cols` cells, a pin matching the viewport and a raw row.
	row_data := state.row_data
	for y := range 4 {
		expectEqual(t, 10, len(row_data[y].cells))
		expected_pin := zigUnwrap(t, term.screens.active.pages.Pin(rnViewport(0, y)))
		expect(t, row_data[y].pin.Eql(expected_pin), "row_data.items(.pin)[y].eql(expected_pin)")
		expect(t, !row_data[y].selection.Valid, "row_data.items(.selection)[y] == null")
		expectEqual(t, 0, len(row_data[y].highlights))
	}
	expectEqual(t, uint32('a'), rnCell(&state, 0, 0).raw.Codepoint())
	expectEqual(t, uint32('b'), rnCell(&state, 1, 0).raw.Codepoint())
	expectEqual(t, uint32(0), rnCell(&state, 2, 0).raw.Codepoint())
	expectEqual(t, uint32('c'), rnCell(&state, 0, 2).raw.Codepoint())
	expectEqual(t, uint32('d'), rnCell(&state, 1, 2).raw.Codepoint())
}

// Zig: test "term51: RenderState.update: dirty state persists until the caller resets it"
func TestRender_term51_RenderState_update_dirty_state_persists_until_the_caller_resets_it(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 4)
	defer term.Deinit()

	state := RenderStateEmpty
	defer state.Deinit()

	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyFull, state.dirty)

	// Without a reset, an unchanged update keeps `full` (and the rows).
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyFull, state.dirty)
	rnAllRows(t, &state, true)

	// Reset: unchanged update stays false and touches no row.
	rnClean(&state)
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyFalse, state.dirty)
	rnExpectRowDirty(t, &state, []bool{false, false, false, false})

	// Writes to rows 1 and 3 give partial. Moving the cursor away from a
	// row also marks that row dirty, so the row the cursor started on (0)
	// is dirty as well, while row 2 was never touched.
	rnWrite(t, term, "\x1b[2;1Ha\x1b[4;1Hb")
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyPartial, state.dirty)
	rnExpectRowDirty(t, &state, []bool{true, true, false, true})
	expectEqual(t, uint32('a'), rnCell(&state, 0, 1).raw.Codepoint())
	expectEqual(t, uint32('b'), rnCell(&state, 0, 3).raw.Codepoint())

	// The row dirty flags of the terminal were consumed by the update.
	for y := range 4 {
		pin := zigUnwrap(t, term.screens.active.pages.Pin(rnViewport(0, y)))
		expect(t, !pin.RowAndCell().row.dirty, "!pin.rowAndCell().row.dirty")
	}

	// Another partial update while already partial stays partial and
	// does not clear the rows set earlier.
	rnExpectRowDirty(t, &state, []bool{true, true, false, true})
	rnMemsetDirty(&state)
	rnWrite(t, term, "z")
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyPartial, state.dirty)
	// The cursor is on row 3 (after "b"), so only that row changed.
	rnExpectRowDirty(t, &state, []bool{false, false, false, true})
	rnMemsetDirty(&state)
	rnWrite(t, term, "\x1b[1;1Hz")
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyPartial, state.dirty)
	rnExpectRowDirty(t, &state, []bool{true, false, false, true})

	// A partial update while `full` leaves it `full`.
	state.dirty = RenderStateDirtyFull
	rnWrite(t, term, "\x1b[3;1Hq")
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyFull, state.dirty)
	rnExpectRowDirty(t, &state, []bool{true, false, true, true})
}

// Zig: test "term51: RenderState.update: every redraw trigger gives full dirty"
func TestRender_term51_RenderState_update_every_redraw_trigger_gives_full_dirty(t *testing.T) {
	testStdCapacity(t)
	term := rnInitScrollback(t, 10, 4, 100)
	defer term.Deinit()
	rnWrite(t, term, "a\r\nb\r\nc\r\nd\r\ne\r\nf")

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))
	rnClean(&state)

	// Each terminal dirty flag alone.
	for _, c := range []struct {
		name string
		set  func(bool)
		get  func() bool
	}{
		{"palette", func(v bool) { term.flags.dirty.palette = v }, func() bool { return term.flags.dirty.palette }},
		{"reverse_colors", func(v bool) { term.flags.dirty.reverse_colors = v }, func() bool { return term.flags.dirty.reverse_colors }},
		{"clear", func(v bool) { term.flags.dirty.clear = v }, func() bool { return term.flags.dirty.clear }},
		{"preedit", func(v bool) { term.flags.dirty.preedit = v }, func() bool { return term.flags.dirty.preedit }},
	} {
		c.set(true)
		zigTry(t, state.Update(term))
		expectEqual(t, RenderStateDirtyFull, state.dirty)
		rnAllRows(t, &state, true)
		// Consumed.
		expect(t, !c.get(), "!@field(t.flags.dirty, name)")
		rnClean(&state)
	}

	// Each screen dirty flag alone.
	for _, c := range []struct {
		name string
		set  func(bool)
		get  func() bool
	}{
		{"selection", func(v bool) { term.screens.active.dirty.selection = v }, func() bool { return term.screens.active.dirty.selection }},
		{"hyperlink_hover", func(v bool) { term.screens.active.dirty.hyperlink_hover = v }, func() bool { return term.screens.active.dirty.hyperlink_hover }},
	} {
		c.set(true)
		zigTry(t, state.Update(term))
		expectEqual(t, RenderStateDirtyFull, state.dirty)
		rnAllRows(t, &state, true)
		expect(t, !c.get(), "!@field(t.screens.active.dirty, name)")
		rnClean(&state)
	}

	// Nothing dirty: no redraw.
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyFalse, state.dirty)

	// Setting the selection through the screen marks it dirty.
	{
		screen := term.screens.active
		zigTry(t, screen.Select(Some(SelectionInit(
			zigUnwrap(t, screen.pages.Pin(rnActive(0, 0))),
			zigUnwrap(t, screen.pages.Pin(rnActive(1, 0))),
			false,
		))))
		zigTry(t, state.Update(term))
		expectEqual(t, RenderStateDirtyFull, state.dirty)
		rnClean(&state)

		zigTry(t, screen.Select(Optional[Selection]{}))
		zigTry(t, state.Update(term))
		expectEqual(t, RenderStateDirtyFull, state.dirty)
		rnClean(&state)
	}

	// Palette change through OSC 4, reverse colors mode, clear screen.
	rnWrite(t, term, "\x1b]4;1;#ff0000\x07")
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyFull, state.dirty)
	rnClean(&state)

	// The read-only stream does not flag the terminal dirty for the
	// reverse colors mode, so toggling it is not a redraw on its own.
	rnWrite(t, term, "\x1b[?5h")
	expect(t, term.modes.Get(ModesModeReverseColors), "t.modes.get(.reverse_colors)")
	expect(t, !term.flags.dirty.reverse_colors, "!t.flags.dirty.reverse_colors")
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyFalse, state.dirty)
	rnClean(&state)

	rnWrite(t, term, "\x1b[2J")
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyFull, state.dirty)
	rnClean(&state)

	// Viewport scroll.
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyFull, state.dirty)
	expect(t, zigUnwrap(t, state.viewport_pin).Eql(term.screens.active.pages.GetTopLeft(PointTagViewport)), "state.viewport_pin.?.eql(t.screens.active.pages.getTopLeft(.viewport))")
	rnClean(&state)

	// Same viewport again: not a redraw.
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyFalse, state.dirty)

	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportBottom})
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyFull, state.dirty)
	rnClean(&state)

	// Cursor visibility/blink mode changes do not dirty the state.
	rnWrite(t, term, "\x1b[?25l\x1b[?12h")
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyFalse, state.dirty)
	expect(t, !state.cursor.visible, "!state.cursor.visible")
	expect(t, state.cursor.blinking, "state.cursor.blinking")
}

// Zig: test "term51: RenderState.update: screen switch and resize redraw"
func TestRender_term51_RenderState_update_screen_switch_and_resize_redraw(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 4)
	defer term.Deinit()
	rnWrite(t, term, "primary")

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))
	expectEqual(t, ScreenSetKeyPrimary, state.screen)
	rnClean(&state)

	// Switch to the alternate screen.
	rnWrite(t, term, "\x1b[?1049h")
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyFull, state.dirty)
	expectEqual(t, ScreenSetKeyAlternate, state.screen)
	rnAllRows(t, &state, true)
	expectEqual(t, uint32(0), rnCell(&state, 0, 0).raw.Codepoint())
	rnClean(&state)

	// The cursor column (7) is kept when entering the alternate screen.
	rnWrite(t, term, "alt")
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyPartial, state.dirty)
	expectEqual(t, uint32('a'), rnCell(&state, 7, 0).raw.Codepoint())
	expectEqual(t, uint32('t'), rnCell(&state, 9, 0).raw.Codepoint())
	expectEqual(t, SizeCellCountInt(9), state.cursor.active.x)
	expect(t, term.screens.active.cursor.pending_wrap, "t.screens.active.cursor.pending_wrap")
	rnClean(&state)

	// And back.
	rnWrite(t, term, "\x1b[?1049l")
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyFull, state.dirty)
	expectEqual(t, ScreenSetKeyPrimary, state.screen)
	expectEqual(t, uint32('p'), rnCell(&state, 0, 0).raw.Codepoint())
	rnClean(&state)

	// Resize: columns only, rows only and both.
	zigTry(t, term.Resize(12, 4))
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyFull, state.dirty)
	expectEqual(t, SizeCellCountInt(12), state.cols)
	rnAllRows(t, &state, true)
	rnClean(&state)

	zigTry(t, term.Resize(12, 6))
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyFull, state.dirty)
	expectEqual(t, SizeCellCountInt(6), state.rows)
	rnClean(&state)

	// The state's size matches the terminal when updated again unchanged.
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyFalse, state.dirty)
}

// Zig: test "term51: RenderState.update: rows and cells are reused across resizes"
func TestRender_term51_RenderState_update_rows_and_cells_are_reused_across_resizes(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 4)
	defer term.Deinit()
	rnWrite(t, term, "ab\r\ncd\r\nef\r\ngh")

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))
	expectEqual(t, 4, len(state.row_data))
	capacity := cap(state.row_data)
	expect(t, capacity >= 4, "cap >= 4")

	// Shrink: rows are dropped but the list keeps its capacity.
	zigTry(t, term.Resize(10, 2))
	zigTry(t, state.Update(term))
	expectEqual(t, 2, len(state.row_data))
	expectEqual(t, capacity, cap(state.row_data))
	expectEqual(t, 10, len(state.row_data[0].cells))
	expectEqual(t, 10, len(state.row_data[1].cells))
	rnAllRows(t, &state, true)

	// Narrower and grow rows: every row has the new column count.
	zigTry(t, term.Resize(6, 5))
	zigTry(t, state.Update(term))
	expectEqual(t, 5, len(state.row_data))
	for _, row := range state.row_data {
		expectEqual(t, 6, len(row.cells))
	}
	rnAllRows(t, &state, true)

	// Wider again.
	zigTry(t, term.Resize(20, 5))
	zigTry(t, state.Update(term))
	for _, row := range state.row_data {
		expectEqual(t, 20, len(row.cells))
	}
	expectEqual(t, SizeCellCountInt(20), state.cols)

	// New rows start fully dirty and clean of selection and highlights.
	rnClean(&state)
	zigTry(t, term.Resize(20, 8))
	zigTry(t, state.Update(term))
	expectEqual(t, 8, len(state.row_data))
	for _, row := range state.row_data {
		expect(t, !row.selection.Valid, "s == null")
	}
	for _, row := range state.row_data {
		expectEqual(t, 0, len(row.highlights))
	}
}

// Zig: test "term51: RenderState.update: page dirty flag marks every row of the page"
func TestRender_term51_RenderState_update_page_dirty_flag_marks_every_row_of_the_page(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 4)
	defer term.Deinit()

	state := RenderStateEmpty
	defer state.Deinit()

	p := &rnPtrNode(t, term.screens.active.pages.pages.first, "t.screens.active.pages.pages.first.?").data
	p.dirty = true

	// A redraw does not consume the page flag.
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyFull, state.dirty)
	expect(t, p.dirty, "p.dirty")
	rnClean(&state)

	// A non-redraw update sees the page as dirty, marks every row and
	// clears the page flag.
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyPartial, state.dirty)
	rnAllRows(t, &state, true)
	expect(t, !p.dirty, "!p.dirty")

	// Now clean: nothing.
	rnClean(&state)
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyFalse, state.dirty)
	rnAllRows(t, &state, false)
}

// Zig: test "term51: RenderState.update: dirty flags of each page in the viewport"
func TestRender_term51_RenderState_update_dirty_flags_of_each_page_in_the_viewport(t *testing.T) {
	testStdCapacity(t)
	opts := terminalOptionsDefault(PageStdCapacity.cols, 10)
	opts.max_scrollback = 10_000
	term := rnInitOpts(t, opts)
	defer term.Deinit()
	rnFillTwoPages(t, term)

	first := &rnPtrNode(t, term.screens.active.pages.pages.first, "t.screens.active.pages.pages.first.?").data
	second := &rnPtrNode(t, term.screens.active.pages.pages.last, "t.screens.active.pages.pages.last.?").data

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))
	expectEqual(t, 10, len(state.row_data))
	// The first five rows come from the first page and the next five from
	// the second.
	pins := state.row_data
	for y := range 5 {
		expect(t, pins[y].pin.node == term.screens.active.pages.pages.first, "pins[y].node == t.screens.active.pages.pages.first.?")
	}
	for y := 5; y < 10; y++ {
		expect(t, pins[y].pin.node == term.screens.active.pages.pages.last, "pins[y].node == t.screens.active.pages.pages.last.?")
	}
	expectEqual(t, uint32('X'), rnCell(&state, 0, 5).raw.Codepoint())
	first.dirty = false
	second.dirty = false
	rnClean(&state)

	// Only the first page dirty.
	first.dirty = true
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyPartial, state.dirty)
	rnExpectRowDirty(t, &state, []bool{
		true, true, true, true, true,
		false, false, false, false, false,
	})
	expect(t, !first.dirty, "!first.dirty")
	expect(t, !second.dirty, "!second.dirty")
	rnClean(&state)

	// Only the second page dirty.
	second.dirty = true
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyPartial, state.dirty)
	rnExpectRowDirty(t, &state, []bool{
		false, false, false, false, false,
		true, true, true, true, true,
	})
	expect(t, !first.dirty, "!first.dirty")
	expect(t, !second.dirty, "!second.dirty")
	rnClean(&state)

	// Both pages dirty: every row and both flags cleared (the flag of
	// the earlier page is cleared when the later one is found).
	first.dirty = true
	second.dirty = true
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyPartial, state.dirty)
	rnAllRows(t, &state, true)
	expect(t, !first.dirty, "!first.dirty")
	expect(t, !second.dirty, "!second.dirty")
	rnClean(&state)

	// A single dirty row on the second page.
	y_pin := zigUnwrap(t, term.screens.active.pages.Pin(rnViewport(0, 7)))
	y_pin.RowAndCell().row.dirty = true
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyPartial, state.dirty)
	rnExpectRowDirty(t, &state, []bool{
		false, false, false, false, false,
		false, false, true, false, false,
	})
	expect(t, !y_pin.RowAndCell().row.dirty, "!y_pin.rowAndCell().row.dirty")
}

// Zig: test "term51: RenderState.update: cursor fields"
func TestRender_term51_RenderState_update_cursor_fields(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 4)
	defer term.Deinit()

	state := RenderStateEmpty
	defer state.Deinit()

	// Defaults of a fresh terminal.
	zigTry(t, state.Update(term))
	expectEqual(t, SizeCellCountInt(0), state.cursor.active.x)
	expectEqual(t, uint32(0), state.cursor.active.y)
	expectEqual(t, SizeCellCountInt(0), zigUnwrap(t, state.cursor.viewport).x)
	expectEqual(t, SizeCellCountInt(0), zigUnwrap(t, state.cursor.viewport).y)
	expect(t, !zigUnwrap(t, state.cursor.viewport).wide_tail, "!state.cursor.viewport.?.wide_tail")
	expect(t, state.cursor.cell.IsZero(), "state.cursor.cell.isZero()")
	expect(t, state.cursor.style.Default(), "state.cursor.style.default()")
	expectEqual(t, CursorStyleBlock, state.cursor.visual_style)
	expect(t, !state.cursor.password_input, "!state.cursor.password_input")
	expect(t, state.cursor.visible, "state.cursor.visible")
	// DECSCUSR default is not blinking until set; the mode default is off.
	expect(t, !state.cursor.blinking, "!state.cursor.blinking")

	// Visual styles through DECSCUSR (and the blink mode they toggle).
	styles := []struct {
		seq   string
		style CursorStyle
		blink bool
	}{
		{seq: "\x1b[1 q", style: CursorStyleBlock, blink: true},
		{seq: "\x1b[2 q", style: CursorStyleBlock, blink: false},
		{seq: "\x1b[3 q", style: CursorStyleUnderline, blink: true},
		{seq: "\x1b[4 q", style: CursorStyleUnderline, blink: false},
		{seq: "\x1b[5 q", style: CursorStyleBar, blink: true},
		{seq: "\x1b[6 q", style: CursorStyleBar, blink: false},
	}
	for _, c := range styles {
		rnWrite(t, term, c.seq)
		zigTry(t, state.Update(term))
		expectEqual(t, c.style, state.cursor.visual_style)
		expectEqual(t, c.blink, state.cursor.blinking)
	}

	// Visibility mode.
	rnWrite(t, term, "\x1b[?25l")
	zigTry(t, state.Update(term))
	expect(t, !state.cursor.visible, "!state.cursor.visible")
	rnWrite(t, term, "\x1b[?25h")
	zigTry(t, state.Update(term))
	expect(t, state.cursor.visible, "state.cursor.visible")

	// Blink mode directly.
	rnWrite(t, term, "\x1b[?12h")
	zigTry(t, state.Update(term))
	expect(t, state.cursor.blinking, "state.cursor.blinking")
	rnWrite(t, term, "\x1b[?12l")
	zigTry(t, state.Update(term))
	expect(t, !state.cursor.blinking, "!state.cursor.blinking")

	// Password input flag.
	term.flags.password_input = true
	zigTry(t, state.Update(term))
	expect(t, state.cursor.password_input, "state.cursor.password_input")
	term.flags.password_input = false
	zigTry(t, state.Update(term))
	expect(t, !state.cursor.password_input, "!state.cursor.password_input")

	// Style and cell under the cursor.
	rnWrite(t, term, "\x1b[H\x1b[1;38;5;3mQ\x1b[D")
	zigTry(t, state.Update(term))
	expectEqual(t, uint32('Q'), state.cursor.cell.Codepoint())
	expect(t, state.cursor.cell.HasStyling(), "state.cursor.cell.hasStyling()")
	expect(t, state.cursor.style.flags.bold, "state.cursor.style.flags.bold")
	expect(t, state.cursor.style.fg_color.Eql(StyleColor{Tag: StyleColorPalette, Palette: 3}), "state.cursor.style.fg_color.eql(.{ .palette = 3 })")
	expectEqual(t, SizeCellCountInt(0), state.cursor.active.x)

	// Positions.
	rnWrite(t, term, "\x1b[3;7H")
	zigTry(t, state.Update(term))
	expectEqual(t, SizeCellCountInt(6), state.cursor.active.x)
	expectEqual(t, uint32(2), state.cursor.active.y)
	expectEqual(t, SizeCellCountInt(6), zigUnwrap(t, state.cursor.viewport).x)
	expectEqual(t, SizeCellCountInt(2), zigUnwrap(t, state.cursor.viewport).y)
	expect(t, state.cursor.cell.IsZero(), "state.cursor.cell.isZero()")

	// Last column / row.
	rnWrite(t, term, "\x1b[4;10H")
	zigTry(t, state.Update(term))
	expectEqual(t, SizeCellCountInt(9), zigUnwrap(t, state.cursor.viewport).x)
	expectEqual(t, SizeCellCountInt(3), zigUnwrap(t, state.cursor.viewport).y)

	// The viewport position is recomputed on each update: scrolled out and
	// back in.
	rnWrite(t, term, "\r\n\r\n\r\n\r\n")
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	zigTry(t, state.Update(term))
	expect(t, !state.cursor.viewport.Valid, "state.cursor.viewport == null")
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportBottom})
	zigTry(t, state.Update(term))
	expect(t, state.cursor.viewport.Valid, "state.cursor.viewport != null")
}

// Zig: test "term51: RenderState.update: cursor on the tail of a wide character"
func TestRender_term51_RenderState_update_cursor_on_the_tail_of_a_wide_character(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 3)
	defer term.Deinit()
	// x0 'a', x1 wide, x2 tail, x3 'b'.
	rnWrite(t, term, "a\xe4\xb8\xadb")

	state := RenderStateEmpty
	defer state.Deinit()

	// On the wide head: not a tail (cell to the left is narrow).
	rnWrite(t, term, "\x1b[2G")
	zigTry(t, state.Update(term))
	expectEqual(t, SizeCellCountInt(1), zigUnwrap(t, state.cursor.viewport).x)
	expect(t, !zigUnwrap(t, state.cursor.viewport).wide_tail, "!state.cursor.viewport.?.wide_tail")
	expectEqual(t, PageCellWideWide, state.cursor.cell.wide)

	// On the tail: the cell to the left is wide.
	rnWrite(t, term, "\x1b[3G")
	zigTry(t, state.Update(term))
	expectEqual(t, SizeCellCountInt(2), zigUnwrap(t, state.cursor.viewport).x)
	expect(t, zigUnwrap(t, state.cursor.viewport).wide_tail, "state.cursor.viewport.?.wide_tail")
	expectEqual(t, PageCellWideSpacerTail, state.cursor.cell.wide)

	// After the tail: the cell to the left is the tail itself.
	rnWrite(t, term, "\x1b[4G")
	zigTry(t, state.Update(term))
	expect(t, !zigUnwrap(t, state.cursor.viewport).wide_tail, "!state.cursor.viewport.?.wide_tail")

	// Column 0 never looks left.
	rnWrite(t, term, "\x1b[1G")
	zigTry(t, state.Update(term))
	expect(t, !zigUnwrap(t, state.cursor.viewport).wide_tail, "!state.cursor.viewport.?.wide_tail")
}

// Zig: test "term51: RenderState.update: colors"
func TestRender_term51_RenderState_update_colors(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 3)
	defer term.Deinit()

	state := RenderStateEmpty
	defer state.Deinit()

	// Nothing set: palette copied, cursor color absent, background and
	// foreground untouched (still the empty state's zero value).
	zigTry(t, state.Update(term))
	expect(t, !state.colors.cursor.Valid, "state.colors.cursor == null")
	expectEqualSlices(t, term.colors.palette.current[:], state.colors.palette[:])
	expectEqual(t, ColorRGB{r: 0, g: 0, b: 0}, state.colors.background)
	expectEqual(t, ColorRGB{r: 0, g: 0, b: 0}, state.colors.foreground)

	// Only the background set: neither is copied.
	rnWrite(t, term, "\x1b]11;#445566\x07")
	zigTry(t, state.Update(term))
	expectEqual(t, ColorRGB{r: 0, g: 0, b: 0}, state.colors.background)
	expectEqual(t, ColorRGB{r: 0, g: 0, b: 0}, state.colors.foreground)

	// Both set.
	rnWrite(t, term, "\x1b]10;#112233\x07")
	zigTry(t, state.Update(term))
	expectEqual(t, ColorRGB{r: 0x44, g: 0x55, b: 0x66}, state.colors.background)
	expectEqual(t, ColorRGB{r: 0x11, g: 0x22, b: 0x33}, state.colors.foreground)

	// Reverse colors swap them.
	rnWrite(t, term, "\x1b[?5h")
	zigTry(t, state.Update(term))
	expectEqual(t, ColorRGB{r: 0x11, g: 0x22, b: 0x33}, state.colors.background)
	expectEqual(t, ColorRGB{r: 0x44, g: 0x55, b: 0x66}, state.colors.foreground)
	rnWrite(t, term, "\x1b[?5l")
	zigTry(t, state.Update(term))
	expectEqual(t, ColorRGB{r: 0x44, g: 0x55, b: 0x66}, state.colors.background)

	// Cursor color set and reset.
	rnWrite(t, term, "\x1b]12;#aabbcc\x07")
	zigTry(t, state.Update(term))
	expectEqual(t, ColorRGB{r: 0xaa, g: 0xbb, b: 0xcc}, zigUnwrap(t, state.colors.cursor))
	rnWrite(t, term, "\x1b]112\x07")
	zigTry(t, state.Update(term))
	expect(t, !state.colors.cursor.Valid, "state.colors.cursor == null")

	// The whole palette is copied: first, last and a middle entry.
	rnWrite(t, term, "\x1b]4;0;#010203\x07\x1b]4;128;#040506\x07\x1b]4;255;#070809\x07")
	zigTry(t, state.Update(term))
	expectEqual(t, ColorRGB{r: 1, g: 2, b: 3}, state.colors.palette[0])
	expectEqual(t, ColorRGB{r: 4, g: 5, b: 6}, state.colors.palette[128])
	expectEqual(t, ColorRGB{r: 7, g: 8, b: 9}, state.colors.palette[255])
	expectEqualSlices(t, term.colors.palette.current[:], state.colors.palette[:])

	// Defaults set through the terminal's colors (as a host would).
	term.colors.background.Set(ColorRGB{r: 9, g: 9, b: 9})
	term.colors.foreground.Set(ColorRGB{r: 8, g: 8, b: 8})
	term.colors.cursor.Set(ColorRGB{r: 7, g: 7, b: 7})
	zigTry(t, state.Update(term))
	expectEqual(t, ColorRGB{r: 9, g: 9, b: 9}, state.colors.background)
	expectEqual(t, ColorRGB{r: 8, g: 8, b: 8}, state.colors.foreground)
	expectEqual(t, ColorRGB{r: 7, g: 7, b: 7}, zigUnwrap(t, state.colors.cursor))
}

// Zig: test "term51: RenderState.update: copies cells styles graphemes and raw rows"
func TestRender_term51_RenderState_update_copies_cells_styles_graphemes_and_raw_rows(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 6, 4)
	defer term.Deinit()
	// Row 0: bold 'A', plain 'b', grapheme e+acute, wide char (x3, tail x4).
	rnWrite(t, term, "\x1b[1mA\x1b[0mb\x1b[?2027he\xcc\x81\xe4\xb8\xad")
	// Row 1: styled 'S' then red bg fill to the end of the row.
	rnWrite(t, term, "\r\n\x1b[3mS\x1b[0m\x1b[41m\x1b[K\x1b[0m")
	// Row 2: bg-only cells with rgb (no styled cell).
	rnWrite(t, term, "\r\n\x1b[48;2;1;2;3m\x1b[K\x1b[0m")
	// Row 3: plain text that wraps into... (end of screen) just text.
	rnWrite(t, term, "\r\nplain!")

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))

	// Row 0.
	{
		c := rnCell(&state, 0, 0)
		expectEqual(t, uint32('A'), c.raw.Codepoint())
		expect(t, c.raw.HasStyling(), "c.raw.hasStyling()")
		expect(t, c.style.flags.bold, "c.style.flags.bold")
		b := rnCell(&state, 1, 0)
		expectEqual(t, uint32('b'), b.raw.Codepoint())
		expect(t, !b.raw.HasStyling(), "!b.raw.hasStyling()")
		g := rnCell(&state, 2, 0)
		expectEqual(t, PageCellContentTagCodepointGrapheme, g.raw.content_tag)
		expectEqualSlices(t, []uint32{0x301}, g.grapheme)
		w := rnCell(&state, 3, 0)
		expectEqual(t, uint32(0x4e2d), w.raw.Codepoint())
		expectEqual(t, PageCellWideWide, w.raw.wide)
		tail := rnCell(&state, 4, 0)
		expectEqual(t, PageCellWideSpacerTail, tail.raw.wide)
		expectEqual(t, uint32(0), tail.raw.Codepoint())
		expectEqual(t, uint32(0), rnCell(&state, 5, 0).raw.Codepoint())
	}

	// Row 1: the styled cell, then background palette cells. The row has
	// managed memory (a styled cell) so the cells' style is filled in.
	{
		s := rnCell(&state, 0, 1)
		expect(t, s.style.flags.italic, "s.style.flags.italic")
		for x := uint(1); x < 6; x++ {
			c := rnCell(&state, x, 1)
			expectEqual(t, PageCellContentTagBgColorPalette, c.raw.content_tag)
			expectEqual(t, uint8(1), c.raw.content.color_palette)
			expect(t, c.style.Eql(Style{bg_color: StyleColor{Tag: StyleColorPalette, Palette: 1}}), "c.style.eql(.{ .bg_color = .{ .palette = 1 } })")
		}
	}

	// Row 2: only background cells and no styled cell: the row has no
	// managed memory, so only the raw cell is copied.
	{
		row := state.row_data[2].raw
		expect(t, !row.ManagedMemory(), "!row.managedMemory()")
		for x := uint(0); x < 6; x++ {
			c := rnCell(&state, x, 2)
			expectEqual(t, PageCellContentTagBgColorRgb, c.raw.content_tag)
			expectEqual(t, uint8(1), c.raw.content.color_rgb.r)
			expectEqual(t, uint8(2), c.raw.content.color_rgb.g)
			expectEqual(t, uint8(3), c.raw.content.color_rgb.b)
		}
	}

	// Row 3.
	for x, ch := range []uint8("plain!") {
		expectEqual(t, uint32(ch), rnCell(&state, uint(x), 3).raw.Codepoint())
	}

	// Raw rows are copies of the page rows, with the wrap flags.
	raws := state.row_data
	expect(t, raws[0].raw.grapheme, "raws[0].grapheme")
	expect(t, raws[0].raw.styled, "raws[0].styled")
	expect(t, !raws[0].raw.wrap, "!raws[0].wrap")
	expect(t, !raws[3].raw.styled, "!raws[3].styled")
	// "plain!" fills the 6 columns: a pending wrap, but not a wrapped row.
	expect(t, !raws[3].raw.wrap, "!raws[3].wrap")
}

// Zig: test "term51: RenderState.update: raw row wrap flags"
func TestRender_term51_RenderState_update_raw_row_wrap_flags(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 5, 3)
	defer term.Deinit()
	rnWrite(t, term, "hello world")

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))

	raws := state.row_data
	expect(t, raws[0].raw.wrap, "raws[0].wrap")
	expect(t, !raws[0].raw.wrap_continuation, "!raws[0].wrap_continuation")
	expect(t, raws[1].raw.wrap, "raws[1].wrap")
	expect(t, raws[1].raw.wrap_continuation, "raws[1].wrap_continuation")
	expect(t, !raws[2].raw.wrap, "!raws[2].wrap")
	expect(t, raws[2].raw.wrap_continuation, "raws[2].wrap_continuation")
}

// Zig: test "term51: RenderState.update: dirty rows are rebuilt and clean rows kept"
func TestRender_term51_RenderState_update_dirty_rows_are_rebuilt_and_clean_rows_kept(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 8, 3)
	defer term.Deinit()
	rnWrite(t, term, "\x1b[2;1Hkeep\x1b[He\xcc\x81\x1b[1mb\x1b[0m")

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))
	expectEqual(t, 1, len(state.row_data[0].cells[0].grapheme))
	expect(t, state.row_data[0].cells[1].style.flags.bold, "state.row_data.items(.cells)[0].get(1).style.flags.bold")
	rnClean(&state)

	// Mutate the clean row 1 in the render state: it must be untouched by
	// an update that only changes row 0.
	state.row_data[1].cells[0].raw = PageCellInit('Z')

	// Overwrite row 0 with plain text: the grapheme and style are gone.
	rnWrite(t, term, "\x1b[Hxy")
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyPartial, state.dirty)
	rnExpectRowDirty(t, &state, []bool{true, false, false})
	expectEqual(t, uint32('x'), rnCell(&state, 0, 0).raw.Codepoint())
	expectEqual(t, PageCellContentTagCodepoint, rnCell(&state, 0, 0).raw.content_tag)
	expectEqual(t, uint32('y'), rnCell(&state, 1, 0).raw.Codepoint())
	expect(t, !rnCell(&state, 1, 0).raw.HasStyling(), "!term51ZRsCell(&state, 1, 0).raw.hasStyling()")
	// Row 1 keeps the sentinel and not 'k'.
	expectEqual(t, uint32('Z'), rnCell(&state, 0, 1).raw.Codepoint())

	// A full redraw restores row 1 from the terminal.
	term.flags.dirty.clear = true
	zigTry(t, state.Update(term))
	expectEqual(t, uint32('k'), rnCell(&state, 0, 1).raw.Codepoint())
}

// Zig: test "term51: RenderState.update: selection rows for every selection shape"
func TestRender_term51_RenderState_update_selection_rows_for_every_selection_shape(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 4)
	defer term.Deinit()
	screen := term.screens.active

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))
	rnExpectSels(t, &state, []rnSelBounds{rnNil, rnNil, rnNil, rnNil})

	// Single line.
	zigTry(t, screen.Select(Some(SelectionInit(rnPin(t, term, 2, 1), rnPin(t, term, 5, 1), false))))
	zigTry(t, state.Update(term))
	rnExpectSels(t, &state, []rnSelBounds{rnNil, rnSel(2, 5), rnNil, rnNil})

	// Single cell.
	zigTry(t, screen.Select(Some(SelectionInit(rnPin(t, term, 4, 2), rnPin(t, term, 4, 2), false))))
	zigTry(t, state.Update(term))
	rnExpectSels(t, &state, []rnSelBounds{rnNil, rnNil, rnSel(4, 4), rnNil})

	// Multi-line: first row from x, middle rows full, last row to x.
	zigTry(t, screen.Select(Some(SelectionInit(rnPin(t, term, 3, 0), rnPin(t, term, 6, 2), false))))
	zigTry(t, state.Update(term))
	rnExpectSels(t, &state, []rnSelBounds{rnSel(3, 9), rnSel(0, 9), rnSel(0, 6), rnNil})

	// The same selection made in reverse.
	zigTry(t, screen.Select(Some(SelectionInit(rnPin(t, term, 6, 2), rnPin(t, term, 3, 0), false))))
	zigTry(t, state.Update(term))
	rnExpectSels(t, &state, []rnSelBounds{rnSel(3, 9), rnSel(0, 9), rnSel(0, 6), rnNil})

	// The whole screen.
	zigTry(t, screen.Select(Some(SelectionInit(rnPin(t, term, 0, 0), rnPin(t, term, 9, 3), false))))
	zigTry(t, state.Update(term))
	rnExpectSels(t, &state, []rnSelBounds{rnSel(0, 9), rnSel(0, 9), rnSel(0, 9), rnSel(0, 9)})

	// Rectangles: the same x range on every row, whichever the direction.
	zigTry(t, screen.Select(Some(SelectionInit(rnPin(t, term, 2, 0), rnPin(t, term, 4, 2), true))))
	zigTry(t, state.Update(term))
	rnExpectSels(t, &state, []rnSelBounds{rnSel(2, 4), rnSel(2, 4), rnSel(2, 4), rnNil})

	zigTry(t, screen.Select(Some(SelectionInit(rnPin(t, term, 4, 0), rnPin(t, term, 2, 2), true))))
	zigTry(t, state.Update(term))
	rnExpectSels(t, &state, []rnSelBounds{rnSel(2, 4), rnSel(2, 4), rnSel(2, 4), rnNil})

	zigTry(t, screen.Select(Some(SelectionInit(rnPin(t, term, 4, 3), rnPin(t, term, 2, 1), true))))
	zigTry(t, state.Update(term))
	rnExpectSels(t, &state, []rnSelBounds{rnNil, rnSel(2, 4), rnSel(2, 4), rnSel(2, 4)})

	// Cleared again.
	zigTry(t, screen.Select(Optional[Selection]{}))
	zigTry(t, state.Update(term))
	rnExpectSels(t, &state, []rnSelBounds{rnNil, rnNil, rnNil, rnNil})
}

// Zig: test "term51: RenderState.update: selection in a scrolled viewport"
func TestRender_term51_RenderState_update_selection_in_a_scrolled_viewport(t *testing.T) {
	testStdCapacity(t)
	term := rnInitScrollback(t, 10, 3, 100)
	defer term.Deinit()
	// Screen rows: a=0, b=1, c=2 (active top), d=3, e=4.
	rnWrite(t, term, "a\r\nb\r\nc\r\nd\r\ne")
	screen := term.screens.active

	state := RenderStateEmpty
	defer state.Deinit()

	// Selection from screen (2,1) to (3,3).
	zigTry(t, screen.Select(Some(SelectionInit(
		zigUnwrap(t, screen.pages.Pin(rnScreen(2, 1))),
		zigUnwrap(t, screen.pages.Pin(rnScreen(3, 3))),
		false,
	))))

	// Viewport at the bottom shows screen rows 2..4.
	zigTry(t, state.Update(term))
	rnExpectSels(t, &state, []rnSelBounds{rnSel(0, 9), rnSel(0, 3), rnNil})

	// Viewport at the top shows screen rows 0..2.
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	zigTry(t, state.Update(term))
	rnExpectSels(t, &state, []rnSelBounds{rnNil, rnSel(2, 9), rnSel(0, 9)})

	// A selection entirely above the viewport.
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportBottom})
	zigTry(t, screen.Select(Some(SelectionInit(
		zigUnwrap(t, screen.pages.Pin(rnScreen(0, 0))),
		zigUnwrap(t, screen.pages.Pin(rnScreen(5, 1))),
		false,
	))))
	zigTry(t, state.Update(term))
	rnExpectSels(t, &state, []rnSelBounds{rnNil, rnNil, rnNil})

	// A selection entirely below the viewport.
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	zigTry(t, screen.Select(Some(SelectionInit(
		zigUnwrap(t, screen.pages.Pin(rnScreen(0, 3))),
		zigUnwrap(t, screen.pages.Pin(rnScreen(5, 4))),
		false,
	))))
	zigTry(t, state.Update(term))
	rnExpectSels(t, &state, []rnSelBounds{rnNil, rnNil, rnNil})
}

// Zig: test "term51: RenderState.update: selection cache"
func TestRender_term51_RenderState_update_selection_cache(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 4)
	defer term.Deinit()
	screen := term.screens.active

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))
	expect(t, !state.selection_cache.Valid, "state.selection_cache == null")

	// A forward selection populates the cache with ordered pins.
	zigTry(t, screen.Select(Some(SelectionInit(rnPin(t, term, 0, 1), rnPin(t, term, 2, 1), false))))
	zigTry(t, state.Update(term))
	expect(t, state.selection_cache.Valid, "state.selection_cache != null")
	expect(t, state.selection_cache.Value.tl_pin.Eql(rnPin(t, term, 0, 1)), "state.selection_cache.?.tl_pin.eql(term51ZRsPin(&t, 0, 1))")
	expect(t, state.selection_cache.Value.br_pin.Eql(rnPin(t, term, 2, 1)), "state.selection_cache.?.br_pin.eql(term51ZRsPin(&t, 2, 1))")
	expect(t, !state.selection_cache.Value.selection.rectangle, "!state.selection_cache.?.selection.rectangle")
	rnClean(&state)

	// Same selection, nothing dirty: the cached result is kept and the
	// rows are not recomputed (a sentinel survives).
	state.row_data[1].selection = rnSel(7, 8)
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyFalse, state.dirty)
	rnExpectSels(t, &state, []rnSelBounds{rnNil, rnSel(7, 8), rnNil, rnNil})

	// Same selection but a dirty row elsewhere: the cache is used to
	// recompute the selected rows (the sentinel is overwritten).
	rnWrite(t, term, "\x1b[4;1Hx")
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyPartial, state.dirty)
	rnExpectSels(t, &state, []rnSelBounds{rnNil, rnSel(0, 2), rnNil, rnNil})
	rnClean(&state)

	// A dirty selected row is reset and recomputed.
	rnWrite(t, term, "\x1b[2;1Hy")
	zigTry(t, state.Update(term))
	rnExpectSels(t, &state, []rnSelBounds{rnNil, rnSel(0, 2), rnNil, rnNil})
	rnClean(&state)

	// A reversed selection never equals the cached (ordered) selection, so
	// it is recomputed every update even when nothing is dirty.
	zigTry(t, screen.Select(Some(SelectionInit(rnPin(t, term, 2, 1), rnPin(t, term, 0, 1), false))))
	zigTry(t, state.Update(term))
	rnClean(&state)
	state.row_data[1].selection = rnSel(7, 8)
	zigTry(t, state.Update(term))
	expectEqual(t, RenderStateDirtyFalse, state.dirty)
	rnExpectSels(t, &state, []rnSelBounds{rnNil, rnSel(0, 2), rnNil, rnNil})

	// Changing the selection builds a new cache.
	zigTry(t, screen.Select(Some(SelectionInit(rnPin(t, term, 1, 2), rnPin(t, term, 3, 2), true))))
	zigTry(t, state.Update(term))
	expect(t, state.selection_cache.Value.tl_pin.Eql(rnPin(t, term, 1, 2)), "state.selection_cache.?.tl_pin.eql(term51ZRsPin(&t, 1, 2))")
	expect(t, state.selection_cache.Value.br_pin.Eql(rnPin(t, term, 3, 2)), "state.selection_cache.?.br_pin.eql(term51ZRsPin(&t, 3, 2))")
	expect(t, state.selection_cache.Value.selection.rectangle, "state.selection_cache.?.selection.rectangle")
	rnExpectSels(t, &state, []rnSelBounds{rnNil, rnNil, rnSel(1, 3), rnNil})

	// Clearing the selection leaves the (stale) cache in place.
	zigTry(t, screen.Select(Optional[Selection]{}))
	zigTry(t, state.Update(term))
	rnExpectSels(t, &state, []rnSelBounds{rnNil, rnNil, rnNil, rnNil})
	expect(t, state.selection_cache.Valid, "state.selection_cache != null")
}

// Zig: test "term51: RenderState.updateHighlightsFlattened: empty input is a no-op"
func TestRender_term51_RenderState_updateHighlightsFlattened_empty_input_is_a_no_op(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 4)
	defer term.Deinit()

	state := RenderStateEmpty
	defer state.Deinit()

	// Before any update there are no rows at all: nothing to do.
	{
		hl := rnFlat(0, 0, nil)
		// Zig-only: defer hl.deinit(alloc);
		zigTry(t, state.UpdateHighlightsFlattened(1, []HighlightFlattened{hl}))
		expectEqual(t, RenderStateDirtyFalse, state.dirty)
	}

	zigTry(t, state.Update(term))
	rnClean(&state)

	// No highlights.
	zigTry(t, state.UpdateHighlightsFlattened(1, []HighlightFlattened{}))
	expectEqual(t, RenderStateDirtyFalse, state.dirty)
	rnAllRows(t, &state, false)

	// A highlight with no chunks.
	{
		hl := rnFlat(0, 0, nil)
		// Zig-only: defer hl.deinit(alloc);
		zigTry(t, state.UpdateHighlightsFlattened(1, []HighlightFlattened{hl}))
		expectEqual(t, RenderStateDirtyFalse, state.dirty)
		rnAllRows(t, &state, false)
	}

	// A chunk of a node that is not in the viewport.
	{
		other := rnInit(t, 10, 4)
		defer other.Deinit()
		other_node := rnPtrNode(t, other.screens.active.pages.pages.first, "other.screens.active.pages.pages.first.?")
		hl := rnFlat(1, 2, []HighlightFlattenedChunk{
			{node: other_node, serial: 0, start: 0, end: 4},
		})
		// Zig-only: defer hl.deinit(alloc);
		zigTry(t, state.UpdateHighlightsFlattened(1, []HighlightFlattened{hl}))
		expectEqual(t, RenderStateDirtyFalse, state.dirty)
		rnAllRows(t, &state, false)
		for y := range 4 {
			expectEqual(t, 0, len(rnHighlights(&state, uint(y))))
		}
	}

	// A chunk with an empty row range (start == end) matches nothing.
	{
		node := state.row_data[0].pin.node
		hl := rnFlat(1, 2, []HighlightFlattenedChunk{
			{node: node, serial: 0, start: 2, end: 2},
		})
		// Zig-only: defer hl.deinit(alloc);
		zigTry(t, state.UpdateHighlightsFlattened(1, []HighlightFlattened{hl}))
		expectEqual(t, RenderStateDirtyFalse, state.dirty)
	}
}

// Zig: test "term51: RenderState.updateHighlightsFlattened: ranges per row and dirty marking"
func TestRender_term51_RenderState_updateHighlightsFlattened_ranges_per_row_and_dirty_marking(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 5)
	defer term.Deinit()

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))
	rnClean(&state)
	node := state.row_data[0].pin.node

	// Rows 1 and 2 (end is exclusive): first row from top_x to the end,
	// last row from the start to bot_x.
	{
		hl := rnFlat(2, 4, []HighlightFlattenedChunk{
			{node: node, serial: node.serial, start: 1, end: 3},
		})
		// Zig-only: defer hl.deinit(alloc);
		zigTry(t, state.UpdateHighlightsFlattened(7, []HighlightFlattened{hl}))
	}
	expectEqual(t, RenderStateDirtyPartial, state.dirty)
	rnExpectRowDirty(t, &state, []bool{false, true, true, false, false})
	expectEqual(t, 0, len(rnHighlights(&state, 0)))
	expectEqual(t, 1, len(rnHighlights(&state, 1)))
	expectEqual(t, uint8(7), rnHighlights(&state, 1)[0].tag)
	expectEqualSlices(t, []SizeCellCountInt{2, 9}, rnRange(rnHighlights(&state, 1)[0]))
	expectEqual(t, 1, len(rnHighlights(&state, 2)))
	expectEqual(t, uint8(7), rnHighlights(&state, 2)[0].tag)
	expectEqualSlices(t, []SizeCellCountInt{0, 4}, rnRange(rnHighlights(&state, 2)[0]))
	expectEqual(t, 0, len(rnHighlights(&state, 3)))

	// Three rows: the middle one is the full width.
	rnClean(&state)
	{
		hl := rnFlat(5, 6, []HighlightFlattenedChunk{
			{node: node, serial: node.serial, start: 2, end: 5},
		})
		// Zig-only: defer hl.deinit(alloc);
		zigTry(t, state.UpdateHighlightsFlattened(9, []HighlightFlattened{hl}))
	}
	expectEqual(t, RenderStateDirtyPartial, state.dirty)
	rnExpectRowDirty(t, &state, []bool{false, false, true, true, true})
	// Highlights accumulate: row 2 has the earlier one and the new one.
	expectEqual(t, 2, len(rnHighlights(&state, 2)))
	expectEqual(t, uint8(9), rnHighlights(&state, 2)[1].tag)
	expectEqualSlices(t, []SizeCellCountInt{5, 9}, rnRange(rnHighlights(&state, 2)[1]))
	expectEqualSlices(t, []SizeCellCountInt{0, 9}, rnRange(rnHighlights(&state, 3)[0]))
	expectEqualSlices(t, []SizeCellCountInt{0, 6}, rnRange(rnHighlights(&state, 4)[0]))

	// A single row: both bounds apply, even when bot_x < top_x.
	rnClean(&state)
	{
		hl := rnFlat(7, 3, []HighlightFlattenedChunk{
			{node: node, serial: node.serial, start: 0, end: 1},
		})
		// Zig-only: defer hl.deinit(alloc);
		zigTry(t, state.UpdateHighlightsFlattened(1, []HighlightFlattened{hl}))
	}
	rnExpectRowDirty(t, &state, []bool{true, false, false, false, false})
	expectEqualSlices(t, []SizeCellCountInt{7, 3}, rnRange(rnHighlights(&state, 0)[0]))

	// The last row of the viewport (end == rows).
	rnClean(&state)
	{
		hl := rnFlat(1, 8, []HighlightFlattenedChunk{
			{node: node, serial: node.serial, start: 4, end: 5},
		})
		// Zig-only: defer hl.deinit(alloc);
		zigTry(t, state.UpdateHighlightsFlattened(1, []HighlightFlattened{hl}))
	}
	rnExpectRowDirty(t, &state, []bool{false, false, false, false, true})
	expectEqual(t, 2, len(rnHighlights(&state, 4)))
	expectEqualSlices(t, []SizeCellCountInt{1, 8}, rnRange(rnHighlights(&state, 4)[1]))

	// Several highlights in one call and the order of the entries on a
	// row follow the order of the slice.
	rnClean(&state)
	{
		a := rnFlat(0, 1, []HighlightFlattenedChunk{
			{node: node, serial: node.serial, start: 3, end: 4},
		})
		// Zig-only: defer a.deinit(alloc);
		b := rnFlat(8, 9, []HighlightFlattenedChunk{
			{node: node, serial: node.serial, start: 3, end: 4},
		})
		// Zig-only: defer b.deinit(alloc);
		zigTry(t, state.UpdateHighlightsFlattened(4, []HighlightFlattened{a, b}))
	}
	// Row 3 also still has the full-width entry of the earlier call.
	expectEqual(t, 3, len(rnHighlights(&state, 3)))
	expectEqualSlices(t, []SizeCellCountInt{0, 9}, rnRange(rnHighlights(&state, 3)[0]))
	expectEqualSlices(t, []SizeCellCountInt{0, 1}, rnRange(rnHighlights(&state, 3)[1]))
	expectEqualSlices(t, []SizeCellCountInt{8, 9}, rnRange(rnHighlights(&state, 3)[2]))
	expectEqual(t, uint8(4), rnHighlights(&state, 3)[1].tag)
	expectEqual(t, uint8(4), rnHighlights(&state, 3)[2].tag)

	// When the state is already full, it stays full.
	rnClean(&state)
	state.dirty = RenderStateDirtyFull
	{
		hl := rnFlat(0, 0, []HighlightFlattenedChunk{
			{node: node, serial: node.serial, start: 0, end: 1},
		})
		// Zig-only: defer hl.deinit(alloc);
		zigTry(t, state.UpdateHighlightsFlattened(1, []HighlightFlattened{hl}))
	}
	expectEqual(t, RenderStateDirtyFull, state.dirty)
}

// Zig: test "term51: RenderState.updateHighlightsFlattened: highlight spanning two pages"
func TestRender_term51_RenderState_updateHighlightsFlattened_highlight_spanning_two_pages(t *testing.T) {
	testStdCapacity(t)
	opts := terminalOptionsDefault(PageStdCapacity.cols, 10)
	opts.max_scrollback = 10_000
	term := rnInitOpts(t, opts)
	defer term.Deinit()
	rnFillTwoPages(t, term)

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))
	rnClean(&state)

	pins := state.row_data
	node1 := pins[0].pin.node
	node2 := pins[9].pin.node
	expect(t, node1 != node2, "node1 != node2")
	capRows := node1.data.size.rows
	last := state.cols - 1

	// Viewport rows 2..4 are the last three rows of page one and rows 5..6
	// are the first two rows of page two.
	hl := rnFlat(3, 6, []HighlightFlattenedChunk{
		{node: node1, serial: node1.serial, start: capRows - 3, end: capRows},
		{node: node2, serial: node2.serial, start: 0, end: 2},
	})
	// Zig-only: defer hl.deinit(alloc);
	zigTry(t, state.UpdateHighlightsFlattened(2, []HighlightFlattened{hl}))

	expectEqual(t, RenderStateDirtyPartial, state.dirty)
	rnExpectRowDirty(t, &state, []bool{
		false, false, true, true, true,
		true, true, false, false, false,
	})
	expectEqualSlices(t, []SizeCellCountInt{3, last}, rnRange(rnHighlights(&state, 2)[0]))
	expectEqualSlices(t, []SizeCellCountInt{0, last}, rnRange(rnHighlights(&state, 3)[0]))
	expectEqualSlices(t, []SizeCellCountInt{0, last}, rnRange(rnHighlights(&state, 4)[0]))
	expectEqualSlices(t, []SizeCellCountInt{0, last}, rnRange(rnHighlights(&state, 5)[0]))
	expectEqualSlices(t, []SizeCellCountInt{0, 6}, rnRange(rnHighlights(&state, 6)[0]))
	expectEqual(t, 0, len(rnHighlights(&state, 7)))
}

// Zig: test "term51: RenderState.updateHighlightsFlattened: highlights survive clean updates and reset on dirty rows"
func TestRender_term51_RenderState_updateHighlightsFlattened_highlights_survive_clean_updates_and_reset_on_dirty_rows(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 3)
	defer term.Deinit()
	rnWrite(t, term, "abc\r\nx")

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))
	rnClean(&state)
	node := state.row_data[0].pin.node

	hl := rnFlat(1, 2, []HighlightFlattenedChunk{
		{node: node, serial: node.serial, start: 0, end: 2},
	})
	// Zig-only: defer hl.deinit(alloc);
	zigTry(t, state.UpdateHighlightsFlattened(3, []HighlightFlattened{hl}))
	expectEqual(t, 1, len(rnHighlights(&state, 0)))
	expectEqual(t, 1, len(rnHighlights(&state, 1)))

	// An update with nothing changed keeps them (and keeps the rows dirty
	// flags the highlight update set, since update never clears them).
	zigTry(t, state.Update(term))
	expectEqual(t, 1, len(rnHighlights(&state, 0)))
	expectEqual(t, 1, len(rnHighlights(&state, 1)))
	rnExpectRowDirty(t, &state, []bool{true, true, false})

	// A write to row 1 (where the cursor is) resets only row 1's highlights.
	rnWrite(t, term, "z")
	zigTry(t, state.Update(term))
	expectEqual(t, 1, len(rnHighlights(&state, 0)))
	expectEqual(t, 0, len(rnHighlights(&state, 1)))

	// A full redraw resets all of them.
	term.flags.dirty.clear = true
	zigTry(t, state.Update(term))
	for y := range 3 {
		expectEqual(t, 0, len(rnHighlights(&state, uint(y))))
	}
}

// Zig: test "term51: RenderState.deinit: frees every kind of row memory"
func TestRender_term51_RenderState_deinit_frees_every_kind_of_row_memory(t *testing.T) {
	testStdCapacity(t)

	// Deinit of an empty state.
	{
		state := RenderStateEmpty
		state.Deinit()
	}

	// After an update with graphemes, styles, selection and highlights.
	{
		term := rnInit(t, 10, 4)
		defer term.Deinit()
		rnWrite(t, term, "\x1b[1me\xcc\x81\x1b[0mx\r\n\x1b]8;;u\x1b\\link\x1b]8;;\x1b\\")
		zigTry(t, term.screens.active.Select(Some(SelectionInit(rnPin(t, term, 0, 0), rnPin(t, term, 3, 1), false))))

		state := RenderStateEmpty
		zigTry(t, state.Update(term))
		node := state.row_data[0].pin.node
		hl := rnFlat(0, 3, []HighlightFlattenedChunk{
			{node: node, serial: node.serial, start: 0, end: 2},
		})
		// Zig-only: defer hl.deinit(alloc);
		zigTry(t, state.UpdateHighlightsFlattened(1, []HighlightFlattened{hl}))
		state.Deinit()
	}

	// After the row count shrank (rows with arenas dropped by update).
	{
		term := rnInit(t, 10, 4)
		defer term.Deinit()
		rnWrite(t, term, "e\xcc\x81\r\ne\xcc\x81\r\ne\xcc\x81\r\ne\xcc\x81")

		state := RenderStateEmpty
		defer state.Deinit()
		zigTry(t, state.Update(term))
		zigTry(t, term.Resize(10, 2))
		zigTry(t, state.Update(term))
		expectEqual(t, 2, len(state.row_data))
	}
}

// Zig: test "term51: RenderState.string: wrapped rows, wide characters and graphemes with a map"
func TestRender_term51_RenderState_string_wrapped_rows_wide_characters_and_graphemes_with_a_map(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 5, 3)
	defer term.Deinit()
	// Row 0 "a" wide(b) c: a, wide, tail, c, empty. Row 1: "e" + acute, 'f'.
	rnWrite(t, term, "a\xe4\xb8\xadc\r\ne\xcc\x81f")

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))

	w := oscWriterAllocatingInit()
	defer w.deinit()
	var map_ RenderStateStringMap
	// Zig-only: defer map.deinit(alloc);

	zigTry(t, state.String(&w.writer, rnMapParam(&map_)))
	expectEqualStrings(t,
		"a\xe4\xb8\xad\x00c\x00\n"+
			"e\xcc\x81f\x00\x00\x00\n"+
			"\x00\x00\x00\x00\x00\n",
		string(w.writer.buffered()),
	)
	expectEqual(t, len(w.writer.buffered()), len(map_))
	expectEqualSlices(t, []PointCoordinate{
		rnCoord(0, 0),                               // a
		rnCoord(1, 0), rnCoord(1, 0), rnCoord(1, 0), // wide
		rnCoord(2, 0),                               // tail (NUL)
		rnCoord(3, 0),                               // c
		rnCoord(4, 0),                               // empty (NUL)
		rnCoord(5, 0),                               // newline at x = cols
		rnCoord(0, 1), rnCoord(0, 1), rnCoord(0, 1), // e + acute
		rnCoord(1, 1),                // f
		rnCoord(2, 1), rnCoord(3, 1), // empties
		rnCoord(4, 1),
		rnCoord(5, 1), // newline
		rnCoord(0, 2),
		rnCoord(1, 2),
		rnCoord(2, 2),
		rnCoord(3, 2),
		rnCoord(4, 2),
		rnCoord(5, 2),
	}, map_)
}

// Zig: test "term51: RenderState.string: soft wrapped rows have no newline"
func TestRender_term51_RenderState_string_soft_wrapped_rows_have_no_newline(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 5, 3)
	defer term.Deinit()
	rnWrite(t, term, "hello world")

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))

	w := oscWriterAllocatingInit()
	defer w.deinit()
	zigTry(t, state.String(&w.writer, Optional[RenderStateStringMapParam]{}))
	expectEqualStrings(t, "hello worl"+"d\x00\x00\x00\x00\n", string(w.writer.buffered()))

	// With a map, there is one entry per byte and none for the missing
	// newlines.
	w2 := oscWriterAllocatingInit()
	defer w2.deinit()
	var map_ RenderStateStringMap
	// Zig-only: defer map.deinit(alloc);
	zigTry(t, state.String(&w2.writer, rnMapParam(&map_)))
	expectEqual(t, len(w2.writer.buffered()), len(map_))
	expectEqual(t, PointCoordinate{x: 4, y: 0}, map_[4])
	expectEqual(t, PointCoordinate{x: 0, y: 1}, map_[5])
	expectEqual(t, PointCoordinate{x: 4, y: 1}, map_[9])
	expectEqual(t, PointCoordinate{x: 0, y: 2}, map_[10])
	expectEqual(t, PointCoordinate{x: 5, y: 2}, map_[15])
}

// Zig: test "term51: RenderState.string: background cells, empty state and writer errors"
func TestRender_term51_RenderState_string_background_cells_empty_state_and_writer_errors(t *testing.T) {
	testStdCapacity(t)

	// An empty state writes nothing.
	{
		state := RenderStateEmpty
		defer state.Deinit()
		w := oscWriterAllocatingInit()
		defer w.deinit()
		var map_ RenderStateStringMap
		// Zig-only: defer map.deinit(alloc);
		zigTry(t, state.String(&w.writer, rnMapParam(&map_)))
		expectEqualStrings(t, "", string(w.writer.buffered()))
		expectEqual(t, 0, len(map_))
	}

	term := rnInit(t, 4, 1)
	defer term.Deinit()
	rnWrite(t, term, "a\x1b[44m\x1b[K")

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))
	expectEqual(t, PageCellContentTagBgColorPalette, rnCell(&state, 1, 0).raw.content_tag)

	// Background cells have no codepoint: NUL like empty cells.
	{
		w := oscWriterAllocatingInit()
		defer w.deinit()
		zigTry(t, state.String(&w.writer, Optional[RenderStateStringMapParam]{}))
		expectEqualStrings(t, "a\x00\x00\x00\n", string(w.writer.buffered()))
	}

	// A writer that is too small fails with WriteFailed.
	{
		var buf [3]uint8
		w := oscWriterFixed(buf[:])
		rnExpectError(t, ErrWriteFailed, state.String(&w, Optional[RenderStateStringMapParam]{}))
	}
}

// Zig: test "term51: RenderState.string: unencodable codepoints"
func TestRender_term51_RenderState_string_unencodable_codepoints(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 4, 1)
	defer term.Deinit()
	rnWrite(t, term, "abc")

	p := &rnPtrNode(t, term.screens.active.pages.pages.first, "t.screens.active.pages.pages.first.?").data

	// A surrogate half passes the length check and is written as U+FFFD.
	p.GetRowAndCell(1, 0).cell.content = PageCellContent{codepoint: 0xd800}
	{
		state := RenderStateEmpty
		defer state.Deinit()
		zigTry(t, state.Update(term))
		w := oscWriterAllocatingInit()
		defer w.deinit()
		var map_ RenderStateStringMap
		// Zig-only: defer map.deinit(alloc);
		zigTry(t, state.String(&w.writer, rnMapParam(&map_)))
		expectEqualStrings(t, "a\xef\xbf\xbdc\x00\n", string(w.writer.buffered()))
		expectEqual(t, len(w.writer.buffered()), len(map_))
	}

	// A value above U+10FFFF cannot be sized: WriteFailed.
	p.GetRowAndCell(1, 0).cell.content = PageCellContent{codepoint: 0x110000}
	{
		state := RenderStateEmpty
		defer state.Deinit()
		zigTry(t, state.Update(term))
		w := oscWriterAllocatingInit()
		defer w.deinit()
		rnExpectError(t, ErrWriteFailed, state.String(&w.writer, Optional[RenderStateStringMapParam]{}))
		// Everything before the bad codepoint was written.
		expectEqualStrings(t, "a", string(w.writer.buffered()))
	}

	// Restore, then put a bad codepoint in a grapheme.
	p.GetRowAndCell(1, 0).cell.content = PageCellContent{codepoint: 'b'}
	{
		rac := p.GetRowAndCell(2, 0)
		zigTry(t, p.AppendGrapheme(rac.row, rac.cell, 0x110000))
		state := RenderStateEmpty
		defer state.Deinit()
		zigTry(t, state.Update(term))
		expectEqualSlices(t, []uint32{0x110000}, rnCell(&state, 2, 0).grapheme)
		w := oscWriterAllocatingInit()
		defer w.deinit()
		rnExpectError(t, ErrWriteFailed, state.String(&w.writer, Optional[RenderStateStringMapParam]{}))
		// The base codepoint of that cell was written before the failure.
		expectEqualStrings(t, "abc", string(w.writer.buffered()))
	}
}

// Zig: test "term51: RenderState.linkCells: separate links, shared ids and rows"
func TestRender_term51_RenderState_linkCells_separate_links_shared_ids_and_rows(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 5)
	defer term.Deinit()
	// Row 0: link a "AB", plain "C", link b "DE".
	rnWrite(t, term, "\x1b]8;;a\x1b\\AB\x1b]8;;\x1b\\C\x1b]8;;b\x1b\\DE\x1b]8;;\x1b\\")
	// Row 1: two implicit links with the same URI back to back.
	rnWrite(t, term, "\r\n\x1b]8;;s\x1b\\F\x1b]8;;s\x1b\\G\x1b]8;;\x1b\\")
	// Row 2: the same explicit id and URI in two places, and the same id
	// with another URI.
	rnWrite(t, term, "\r\n\x1b]8;id=k;u\x1b\\H\x1b]8;;\x1b\\ \x1b]8;id=k;u\x1b\\I\x1b]8;;\x1b\\"+
		"\x1b]8;id=k;v\x1b\\J\x1b]8;;\x1b\\")
	// Rows 3-4: one link over two rows.
	rnWrite(t, term, "\r\n\x1b]8;;m\x1b\\KL\r\nMN\x1b]8;;\x1b\\")

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))

	// Link a.
	{
		cells, err := state.LinkCells(rnCoord(1, 0))
		zigTry(t, err)
		defer cells.Deinit()
		expectEqual(t, uint(2), cells.Count())
		expect(t, cells.Contains(rnCoord(0, 0)), "cells.contains(.{ .x = 0, .y = 0 })")
		expect(t, cells.Contains(rnCoord(1, 0)), "cells.contains(.{ .x = 1, .y = 0 })")
	}
	// Link b.
	{
		cells, err := state.LinkCells(rnCoord(4, 0))
		zigTry(t, err)
		defer cells.Deinit()
		expectEqual(t, uint(2), cells.Count())
		expect(t, cells.Contains(rnCoord(3, 0)), "cells.contains(.{ .x = 3, .y = 0 })")
		expect(t, cells.Contains(rnCoord(4, 0)), "cells.contains(.{ .x = 4, .y = 0 })")
	}
	// A plain cell, an empty cell, and the first empty cell after.
	for _, p := range []PointCoordinate{rnCoord(2, 0), rnCoord(5, 0), rnCoord(9, 4)} {
		cells, err := state.LinkCells(p)
		zigTry(t, err)
		defer cells.Deinit()
		expectEqual(t, uint(0), cells.Count())
	}
	// Same URI but different implicit ids: separate links.
	{
		cells, err := state.LinkCells(rnCoord(0, 1))
		zigTry(t, err)
		defer cells.Deinit()
		expectEqual(t, uint(1), cells.Count())
		expect(t, cells.Contains(rnCoord(0, 1)), "cells.contains(.{ .x = 0, .y = 1 })")
	}
	{
		cells, err := state.LinkCells(rnCoord(1, 1))
		zigTry(t, err)
		defer cells.Deinit()
		expectEqual(t, uint(1), cells.Count())
		expect(t, cells.Contains(rnCoord(1, 1)), "cells.contains(.{ .x = 1, .y = 1 })")
	}
	// Same explicit id and URI in two places: one link.
	{
		cells, err := state.LinkCells(rnCoord(0, 2))
		zigTry(t, err)
		defer cells.Deinit()
		expectEqual(t, uint(2), cells.Count())
		expect(t, cells.Contains(rnCoord(0, 2)), "cells.contains(.{ .x = 0, .y = 2 })")
		expect(t, cells.Contains(rnCoord(2, 2)), "cells.contains(.{ .x = 2, .y = 2 })")
	}
	// Same id but a different URI: not the same link.
	{
		cells, err := state.LinkCells(rnCoord(3, 2))
		zigTry(t, err)
		defer cells.Deinit()
		expectEqual(t, uint(1), cells.Count())
		expect(t, cells.Contains(rnCoord(3, 2)), "cells.contains(.{ .x = 3, .y = 2 })")
	}
	// A link over two rows, queried from the second.
	{
		cells, err := state.LinkCells(rnCoord(1, 4))
		zigTry(t, err)
		defer cells.Deinit()
		expectEqual(t, uint(4), cells.Count())
		expect(t, cells.Contains(rnCoord(0, 3)), "cells.contains(.{ .x = 0, .y = 3 })")
		expect(t, cells.Contains(rnCoord(1, 3)), "cells.contains(.{ .x = 1, .y = 3 })")
		expect(t, cells.Contains(rnCoord(0, 4)), "cells.contains(.{ .x = 0, .y = 4 })")
		expect(t, cells.Contains(rnCoord(1, 4)), "cells.contains(.{ .x = 1, .y = 4 })")
	}
}

// Zig: test "term51: RenderState.linkCells: bounds of the viewport point"
func TestRender_term51_RenderState_linkCells_bounds_of_the_viewport_point(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 4, 2)
	defer term.Deinit()
	// A link on the last cell of the last row and another on the first.
	rnWrite(t, term, "\x1b]8;;a\x1b\\Z\x1b]8;;\x1b\\\x1b[2;4H\x1b]8;;b\x1b\\Y\x1b]8;;\x1b\\")

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))

	// Last valid x and y.
	{
		cells, err := state.LinkCells(rnCoord(3, 1))
		zigTry(t, err)
		defer cells.Deinit()
		expectEqual(t, uint(1), cells.Count())
		expect(t, cells.Contains(rnCoord(3, 1)), "cells.contains(.{ .x = 3, .y = 1 })")
	}
	// One past the last x and y: empty (the cell exists in the next
	// row/column in the data, but is not queried).
	{
		cells, err := state.LinkCells(rnCoord(4, 1))
		zigTry(t, err)
		defer cells.Deinit()
		expectEqual(t, uint(0), cells.Count())
	}
	{
		cells, err := state.LinkCells(rnCoord(3, 2))
		zigTry(t, err)
		defer cells.Deinit()
		expectEqual(t, uint(0), cells.Count())
	}
	{
		cells, err := state.LinkCells(rnCoord(0, 0))
		zigTry(t, err)
		defer cells.Deinit()
		expectEqual(t, uint(1), cells.Count())
	}

	// An empty render state has no rows to query.
	empty := RenderStateEmpty
	defer empty.Deinit()
	cells, err := empty.LinkCells(rnCoord(0, 0))
	zigTry(t, err)
	defer cells.Deinit()
	expectEqual(t, uint(0), cells.Count())
}

// Go-only: panic test: RenderState.update: selection pins that are not in the screen's page list (Selection.order unwraps a null pointFromPin)
func TestRender_panic_update_selection_pins_not_in_the_page_list(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 3)
	defer term.Deinit()
	other := rnInit(t, 10, 3)
	defer other.Deinit()

	// The screen's selection is set directly (Screen.select would track the
	// pins): its pins belong to the page list of another terminal.
	screen := term.screens.active
	screen.selection = Some(SelectionInit(
		rnPin(t, other, 0, 0),
		rnPin(t, other, 2, 1),
		false,
	))
	defer func() { screen.selection = Optional[Selection]{} }()

	state := RenderStateEmpty
	defer state.Deinit()
	rnExpectPanic(t, func() {
		_ = state.Update(term)
	})
}

// Go-only: panic test: RenderState.linkCells: row pin outside the rows of its page (getRowAndCell assert)
func TestRender_panic_linkCells_row_pin_outside_the_page(t *testing.T) {
	testStdCapacity(t)
	term := rnInit(t, 10, 3)
	defer term.Deinit()
	rnWrite(t, term, "\x1b]8;;http://example.com\x1b\\LINK\x1b]8;;\x1b\\")

	state := RenderStateEmpty
	defer state.Deinit()
	zigTry(t, state.Update(term))

	// A stale state: the pin of row 0 no longer names a row of its page.
	state.row_data[0].pin.y = 60000
	rnExpectPanic(t, func() {
		_, _ = state.LinkCells(rnCoord(0, 0))
	})
}
