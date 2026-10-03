// Tests translated one for one from src/terminal/Terminal.zig (Ghostty
// v1.3.1 plus the term51 additional tests), sixth part: from "term51:
// scrollUp: full screen counts, scrollback and clamping" up to and including
// "term51: deleteLines: a wide character split by the left margin is cleared"
// (Zig lines 19358 to 22151). Each Go test carries the exact Zig title in a
// "// Zig: test" comment. Helpers are prefixed tm6.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualSlices / expectEqualStrings keep Zig's names
//     and Zig's (expected, actual) argument order. `try` on a call that
//     returns an error is zigTry(t, err); `x.?` is zigUnwrap.
//   - The Zig variable `t` (the terminal) is `tm` in Go, because `t` is the
//     *testing.T. `init(alloc, .{ .cols = c, .rows = r })` is tm6Init, which
//     applies the Zig option defaults (terminalOptionsDefault).
//   - `defer t.deinit(alloc)` is `defer tm.Deinit()`; `testing.allocator` and
//     the `alloc` arguments are dropped.
//   - Every test starts with testStdCapacity(t): Zig's unit tests run with a
//     standard page capacity of grapheme_bytes 512.
//   - `errdefer std.debug.print(...)` in a loop is t.Logf before the
//     iteration (shown by go test only when the test fails). The body of a
//     loop iteration that creates a terminal is a function literal, so that
//     `defer tm.Deinit()` runs at the end of the iteration as in Zig.
//   - `build_options.kitty_graphics` is false in the library build: Screen
//     has no kitty_images, so term51UKittyDirty is always false.
//   - Tests that were not translated are listed in
//     scripts/not-translated-tm6.txt.

package terminal

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers (prefix tm6).
// ---------------------------------------------------------------------------

// tm6ExpectPanic runs f, which must panic. A panic with the wireframe's
// "not implemented" value does not count: it only means the function under
// test does not exist yet. The setup is done outside f.
func tm6ExpectPanic(t *testing.T, f func()) {
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

// tm6Init is `try init(alloc, .{ .cols = cols, .rows = rows })` with the Zig
// option defaults; the terminal is returned by pointer (the methods take a
// pointer receiver).
func tm6Init(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) *Terminal {
	t.Helper()
	return tm6InitOpts(t, terminalOptionsDefault(cols, rows))
}

// tm6InitOpts is `try init(alloc, opts)`.
func tm6InitOpts(t *testing.T, opts TerminalOptions) *Terminal {
	t.Helper()
	tm, err := TerminalInit(opts)
	zigTry(t, err)
	return &tm
}

// tm6Active is the Zig literal `.{ .active = .{ .x = x, .y = y } }`.
func tm6Active(x uint, y uint) Point {
	return Point{Tag: PointTagActive, Active: PointCoordinate{x: SizeCellCountInt(x), y: uint32(y)}}
}

// tm6Print is `try t.print(c)`.
func tm6Print(t *testing.T, tm *Terminal, c uint32) {
	t.Helper()
	zigTry(t, tm.Print(c))
}

// tm6PrintString is `try t.printString(str)`.
func tm6PrintString(t *testing.T, tm *Terminal, str string) {
	t.Helper()
	zigTry(t, tm.PrintString([]uint8(str)))
}

// tm6Linefeed is `try t.linefeed()`.
func tm6Linefeed(t *testing.T, tm *Terminal) {
	t.Helper()
	zigTry(t, tm.Linefeed())
}

// tm6ScrollUp is `try t.scrollUp(count)`.
func tm6ScrollUp(t *testing.T, tm *Terminal, count uint) {
	t.Helper()
	zigTry(t, tm.ScrollUp(count))
}

// tm6SetAttribute is `try t.setAttribute(attr)`.
func tm6SetAttribute(t *testing.T, tm *Terminal, attr SGRAttribute) {
	t.Helper()
	zigTry(t, tm.SetAttribute(attr))
}

// The sgr.Attribute literals the tests use.
var (
	tm6AttrBold   = SGRAttribute{Tag: SGRAttributeTagBold}
	tm6AttrItalic = SGRAttribute{Tag: SGRAttributeTagItalic}
	tm6AttrUnset  = SGRAttribute{Tag: SGRAttributeTagUnset}
)

// tm6Attr256Bg is `.{ .@"256_bg" = n }`.
func tm6Attr256Bg(n uint8) SGRAttribute {
	return SGRAttribute{Tag: SGRAttributeTag256Bg, X256Bg: n}
}

// tm6AttrDirectBg is `.{ .direct_color_bg = .{ .r = r, .g = g, .b = b } }`.
func tm6AttrDirectBg(r uint8, g uint8, b uint8) SGRAttribute {
	return SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{r: r, g: g, b: b}}
}

// tm6ScrollViewportDelta is `.{ .delta = n }`.
func tm6ScrollViewportDelta(n int) TerminalScrollViewport {
	return TerminalScrollViewport{Tag: TerminalScrollViewportDelta, Delta: n}
}

// tm6StyleRefCount is `page.styles.refCount(page.memory, id)`.
func tm6StyleRefCount(page *Page, id StyleId) uint {
	return uint(page.styles.RefCount(id))
}

// tm6SwitchScreen is `_ = try t.switchScreen(key)`.
func tm6SwitchScreen(t *testing.T, tm *Terminal, key ScreenSetKey) {
	t.Helper()
	_, err := tm.SwitchScreen(key)
	zigTry(t, err)
}

// ---- term51 T helpers ----

// tm6TLabel is term51TLabel: label for the cell at (x, y) of a grid `cols`
// wide: 'A'-'Z' then 'a'-'z', then wrapping. Unique for up to 52 cells.
func tm6TLabel(cols uint, x uint, y uint) uint32 {
	n := (y*cols + x) % 52
	if n < 26 {
		return uint32('A' + n)
	}
	return uint32('a' + (n - 26))
}

// tm6TFill is term51TFill: fill the whole active area with unique labels and
// leave the cursor at the home position with no pending wrap.
func tm6TFill(t *testing.T, tm *Terminal) {
	t.Helper()
	for y := uint(0); y < uint(tm.rows); y++ {
		for x := uint(0); x < uint(tm.cols); x++ {
			tm.SetCursorPos(y+1, x+1)
			tm6Print(t, tm, tm6TLabel(uint(tm.cols), x, y))
		}
	}
	tm.SetCursorPos(1, 1)
}

// tm6TRegion is term51TRegion: set the scrolling region directly (all values
// 0-indexed, inclusive).
func tm6TRegion(tm *Terminal, top uint, bottom uint, left uint, right uint) {
	tm.scrolling_region = TerminalScrollingRegion{
		top:    SizeCellCountInt(top),
		bottom: SizeCellCountInt(bottom),
		left:   SizeCellCountInt(left),
		right:  SizeCellCountInt(right),
	}
}

// tm6TCellAt is term51TCellAt: a copy of the cell.
func tm6TCellAt(t *testing.T, tm *Terminal, x uint, y uint) PageCell {
	t.Helper()
	return *zigUnwrap(t, tm.screens.active.pages.GetCell(tm6Active(x, y))).cell
}

// tm6TRowAt is term51TRowAt.
func tm6TRowAt(t *testing.T, tm *Terminal, y uint) *PageRow {
	t.Helper()
	return zigUnwrap(t, tm.screens.active.pages.GetCell(tm6Active(0, y))).row
}

// tm6TGridRow is term51TGridRow: one character per cell: '.' for an empty
// narrow cell, '#' for a cell that only carries a background colour, the
// ASCII character for a narrow cell with a codepoint below 128 ('?' above),
// 'W' for a wide cell, 'w' for a wide cell with no codepoint, '_' for a
// spacer tail, '>' for a spacer head.
func tm6TGridRow(t *testing.T, tm *Terminal, y uint) string {
	t.Helper()
	buf := make([]uint8, tm.cols)
	for x := uint(0); x < uint(tm.cols); x++ {
		c := tm6TCellAt(t, tm, x, y)
		switch c.wide {
		case PageCellWideSpacerTail:
			buf[x] = '_'
		case PageCellWideSpacerHead:
			buf[x] = '>'
		case PageCellWideWide:
			if c.content.codepoint == 0 {
				buf[x] = 'w'
			} else {
				buf[x] = 'W'
			}
		case PageCellWideNarrow:
			switch c.content_tag {
			case PageCellContentTagBgColorPalette, PageCellContentTagBgColorRgb:
				buf[x] = '#'
			case PageCellContentTagCodepoint, PageCellContentTagCodepointGrapheme:
				if c.content.codepoint == 0 {
					buf[x] = '.'
				} else if c.content.codepoint < 128 {
					buf[x] = uint8(c.content.codepoint)
				} else {
					buf[x] = '?'
				}
			}
		}
	}
	return string(buf)
}

// tm6TExpectGrid is term51TExpectGrid: compare the whole active area against
// `expected`, one string per row.
func tm6TExpectGrid(t *testing.T, tm *Terminal, expected ...string) {
	t.Helper()
	expectEqual(t, uint(tm.rows), uint(len(expected)))
	for y, row := range expected {
		expectEqualStrings(t, row, tm6TGridRow(t, tm, uint(y)))
	}
}

// tm6TExpectCursor is term51TExpectCursor.
func tm6TExpectCursor(t *testing.T, tm *Terminal, x uint, y uint) {
	t.Helper()
	expectEqual(t, x, uint(tm.screens.active.cursor.x))
	expectEqual(t, y, uint(tm.screens.active.cursor.y))
}

// tm6TScrollbackRows is term51TScrollbackRows: the number of rows above the
// active area.
func tm6TScrollbackRows(tm *Terminal) uint {
	return tm.screens.active.pages.total_rows - uint(tm.rows)
}

// tm6TExpectCursorConsistent is term51TExpectCursorConsistent: the cursor's
// cached row and cell pointers and its pin match its x/y.
func tm6TExpectCursorConsistent(t *testing.T, tm *Terminal) {
	t.Helper()
	s := tm.screens.active
	lc := zigUnwrap(t, s.pages.GetCell(tm6Active(uint(s.cursor.x), uint(s.cursor.y))))
	expect(t, s.cursor.page_cell == lc.cell, "s.cursor.page_cell == lc.cell")
	expect(t, s.cursor.page_row == lc.row, "s.cursor.page_row == lc.row")
	expectEqual(t, s.cursor.x, s.cursor.page_pin.x)
}

// tm6TBg is term51TBg.
var tm6TBg = tm6AttrDirectBg(1, 2, 3)

// tm6TScrollbackTerminal is term51TScrollbackTerminal: fill a terminal with
// the lines "A".."J" (10 lines, 3 visible rows, so 7 rows of scrollback) and
// leave the viewport at the bottom.
func tm6TScrollbackTerminal(t *testing.T) *Terminal {
	t.Helper()
	tm := tm6Init(t, 4, 3)
	for i, c := range []uint8("ABCDEFGHIJ") {
		if i > 0 {
			tm.CarriageReturn()
			tm6Linefeed(t, tm)
		}
		tm6Print(t, tm, uint32(c))
	}
	return tm
}

// tm6TViewportTop is term51TViewportTop: the row (counted from the top of
// the scrollback) shown at the top of the viewport.
func tm6TViewportTop(t *testing.T, tm *Terminal) uint {
	t.Helper()
	pages := &tm.screens.active.pages
	return uint(zigUnwrap(t, pages.PointFromPin(PointTagScreen, pages.GetTopLeft(PointTagViewport))).Screen.y)
}

// tm6TExpectViewport is term51TExpectViewport.
func tm6TExpectViewport(t *testing.T, tm *Terminal, expected string) {
	t.Helper()
	str, err := tm.PlainString()
	zigTry(t, err)
	expectEqualStrings(t, expected, string(str))
}

// tm6TPutRow is term51TPutRow: write `str` at the start of row y.
func tm6TPutRow(t *testing.T, tm *Terminal, y uint, str string) {
	t.Helper()
	tm.SetCursorPos(y+1, 1)
	tm6PrintString(t, tm, str)
}

// tm6TShiftRow is term51TShiftRow: call rowWillBeShifted for row y with the
// region's columns set to left..right (rows are the full screen).
func tm6TShiftRow(t *testing.T, tm *Terminal, y uint, left uint, right uint) {
	t.Helper()
	tm6TRegion(tm, 0, uint(tm.rows)-1, left, right)
	lc := zigUnwrap(t, tm.screens.active.pages.GetCell(tm6Active(0, y)))
	tm.rowWillBeShifted(&lc.node.data, lc.row)
	lc.node.data.AssertIntegrity()
}

// tm6TRawRow is term51TRawRow.
func tm6TRawRow(t *testing.T, tm *Terminal, y uint, out []uint64) []uint64 {
	t.Helper()
	for x := uint(0); x < uint(tm.cols); x++ {
		out[x] = tm6TCellAt(t, tm, x, y).bitCast()
	}
	return out[:tm.cols]
}

// tm6TSpacerHeadTerminal is term51TSpacerHeadTerminal: row 0 is "abcdefghi"
// + a spacer head at column 9, the wide character is on row 1.
func tm6TSpacerHeadTerminal(t *testing.T) *Terminal {
	t.Helper()
	tm := tm6Init(t, 10, 3)
	tm6PrintString(t, tm, "abcdefghi中")
	return tm
}

// ---- term51 U helpers ----

// tm6UAlpha6 is term51UAlpha6: six rows of six distinct letters, used as the
// standard fixture.
var tm6UAlpha6 = []string{
	"abcdef",
	"ghijkl",
	"mnopqr",
	"stuvwx",
	"yzABCD",
	"EFGHIJ",
}

// tm6UFill is term51UFill: print every non-'.' byte of every row at its own
// position. Leaves the cursor at the top-left with no pending wrap.
func tm6UFill(t *testing.T, tm *Terminal, rows []string) {
	t.Helper()
	for y, row := range rows {
		for x, c := range []uint8(row) {
			if c == '.' {
				continue
			}
			tm.SetCursorPos(uint(y)+1, uint(x)+1)
			tm6Print(t, tm, uint32(c))
		}
	}
	tm.SetCursorPos(1, 1)
}

// tm6UCellAt is term51UCellAt.
func tm6UCellAt(t *testing.T, tm *Terminal, x uint, y uint) *PageCell {
	t.Helper()
	return zigUnwrap(t, tm.screens.active.pages.GetCell(tm6Active(x, y))).cell
}

// tm6URowAt is term51URowAt.
func tm6URowAt(t *testing.T, tm *Terminal, y uint) *PageRow {
	t.Helper()
	return zigUnwrap(t, tm.screens.active.pages.GetCell(tm6Active(0, y))).row
}

// tm6UCellChar is term51UCellChar: one character per cell: '.' empty, the
// ASCII character for narrow text, '*' narrow grapheme cluster, '#' wide
// head, '>' spacer tail, '<' spacer head, 'p' palette background cell, 'r'
// RGB background cell.
func tm6UCellChar(cell PageCell) uint8 {
	switch cell.wide {
	case PageCellWideWide:
		return '#'
	case PageCellWideSpacerTail:
		return '>'
	case PageCellWideSpacerHead:
		return '<'
	}
	// narrow
	switch cell.content_tag {
	case PageCellContentTagCodepoint:
		if cell.content.codepoint == 0 {
			return '.'
		} else if cell.content.codepoint < 128 {
			return uint8(cell.content.codepoint)
		}
		return '?'
	case PageCellContentTagCodepointGrapheme:
		return '*'
	case PageCellContentTagBgColorPalette:
		return 'p'
	case PageCellContentTagBgColorRgb:
		return 'r'
	}
	panic("unreachable")
}

// tm6UExpectGrid is term51UExpectGrid: compare the whole active area against
// the expected rows.
func tm6UExpectGrid(t *testing.T, tm *Terminal, expected ...string) {
	t.Helper()
	expectEqual(t, uint(tm.rows), uint(len(expected)))
	buf := make([]uint8, tm.cols)
	for y, exp := range expected {
		for x := uint(0); x < uint(tm.cols); x++ {
			buf[x] = tm6UCellChar(*tm6UCellAt(t, tm, x, uint(y)))
		}
		expectEqualStrings(t, exp, string(buf))
	}
}

// tm6UExpectDirty is term51UExpectDirty: compare the dirty flag of every
// active row: 'D' dirty, '.' clean.
func tm6UExpectDirty(t *testing.T, tm *Terminal, expected string) {
	t.Helper()
	expectEqual(t, uint(tm.rows), uint(len(expected)))
	buf := make([]uint8, tm.rows)
	for y := uint(0); y < uint(tm.rows); y++ {
		if tm6URowAt(t, tm, y).dirty {
			buf[y] = 'D'
		} else {
			buf[y] = '.'
		}
	}
	expectEqualStrings(t, expected, string(buf))
}

// tm6UExpectWrap is term51UExpectWrap: compare the wrap (w),
// wrap_continuation (c), both (b) or neither (.) of every active row.
func tm6UExpectWrap(t *testing.T, tm *Terminal, expected string) {
	t.Helper()
	expectEqual(t, uint(tm.rows), uint(len(expected)))
	buf := make([]uint8, tm.rows)
	for y := uint(0); y < uint(tm.rows); y++ {
		row := tm6URowAt(t, tm, y)
		if row.wrap && row.wrap_continuation {
			buf[y] = 'b'
		} else if row.wrap {
			buf[y] = 'w'
		} else if row.wrap_continuation {
			buf[y] = 'c'
		} else {
			buf[y] = '.'
		}
	}
	expectEqualStrings(t, expected, string(buf))
}

// tm6UKittyDirty is term51UKittyDirty. build_options.kitty_graphics is false
// in the library build, so the function returns false.
func tm6UKittyDirty(tm *Terminal) bool {
	return false
}

// tm6UResetKittyDirty is term51UResetKittyDirty. With kitty graphics compiled
// out it does nothing.
func tm6UResetKittyDirty(tm *Terminal) {
}

// tm6USetWrap is term51USetWrap: set every row to the "wrap" state given by
// the characters of `flags`.
func tm6USetWrap(t *testing.T, tm *Terminal, flags string) {
	t.Helper()
	for y, f := range []uint8(flags) {
		row := tm6URowAt(t, tm, uint(y))
		row.wrap = f == 'w' || f == 'b'
		row.wrap_continuation = f == 'c' || f == 'b'
	}
}

// tm6UExpectCursor is term51UExpectCursor.
func tm6UExpectCursor(t *testing.T, tm *Terminal, x uint, y uint, pending_wrap bool) {
	t.Helper()
	expectEqual(t, x, uint(tm.screens.active.cursor.x))
	expectEqual(t, y, uint(tm.screens.active.cursor.y))
	expectEqual(t, pending_wrap, tm.screens.active.cursor.pending_wrap)
}

// tm6UPage is term51UPage: the page that holds the cursor; fine for
// terminals that fit one page.
func tm6UPage(tm *Terminal) *Page {
	return &tm.screens.active.cursor.page_pin.node.data
}

// tm6UAlphaAt is term51UAlphaAt: prepare the standard fixture with the
// cursor at (x, y) 0-indexed, dirty flags cleared.
func tm6UAlphaAt(t *testing.T, tm *Terminal, x uint, y uint) {
	t.Helper()
	tm6UFill(t, tm, tm6UAlpha6)
	tm.SetCursorPos(y+1, x+1)
	tm.clearDirty()
	tm6UResetKittyDirty(tm)
}

// tm6UPutStr is term51UPutStr: print `str` starting at the 0-indexed
// position (x, y).
func tm6UPutStr(t *testing.T, tm *Terminal, x uint, y uint, str string) {
	t.Helper()
	tm.SetCursorPos(y+1, x+1)
	tm6PrintString(t, tm, str)
}

// tm6UPutWide is term51UPutWide: print one wide character (two cells) with
// its head at (x, y).
func tm6UPutWide(t *testing.T, tm *Terminal, x uint, y uint) {
	t.Helper()
	tm.SetCursorPos(y+1, x+1)
	tm6Print(t, tm, 0x4E2D)
}

// tm6UPutWideGrapheme is term51UPutWideGrapheme: print a family emoji ZWJ
// sequence, which is one wide cell with a grapheme.
func tm6UPutWideGrapheme(t *testing.T, tm *Terminal, x uint, y uint) {
	t.Helper()
	tm.modes.Set(ModesModeGraphemeCluster, true)
	tm.SetCursorPos(y+1, x+1)
	tm6Print(t, tm, 0x1F468)
	tm6Print(t, tm, 0x200D)
	tm6Print(t, tm, 0x1F469)
}

// tm6UPutGrapheme is term51UPutGrapheme: print an 'e' followed by a
// combining acute accent at (x, y): one narrow cell with a grapheme.
func tm6UPutGrapheme(t *testing.T, tm *Terminal, x uint, y uint) {
	t.Helper()
	tm.SetCursorPos(y+1, x+1)
	tm6Print(t, tm, 'e')
	// The grapheme is attached directly: a zero-width character printed
	// after a character on the last column would attach to the cell before.
	zigTry(t, tm6UPage(tm).SetGraphemes(
		tm6URowAt(t, tm, y),
		tm6UCellAt(t, tm, x, y),
		[]uint32{0x0301},
	))
}

// tm6UCrossPage is term51UCrossPage: linefeed so that the active area is
// split over two pages, with `tail_rows` active rows on the second page. The
// cursor ends up at the top-left of the screen. The terminal must have 5
// rows.
func tm6UCrossPage(t *testing.T, tm *Terminal, tail_rows uint) {
	t.Helper()
	first := tm.screens.active.pages.pages.first
	expect(t, first != nil, "pages.pages.first != null")
	n := uint(first.data.capacity.rows)
	for i := uint(0); i < n+tail_rows-1; i++ {
		tm6Linefeed(t, tm)
	}
	tm.SetCursorPos(1, 1)
	expect(t, first.next != nil, "first.next != null")
	for y := uint(0); y < uint(tm.rows); y++ {
		node := zigUnwrap(t, tm.screens.active.pages.Pin(tm6Active(0, y))).node
		if y+tail_rows < uint(tm.rows) {
			expect(t, node == first, "node == first")
		} else {
			expect(t, node == first.next, "node == first.next.?")
		}
	}
}

// tm6UNodeAt is term51UNodeAt.
func tm6UNodeAt(t *testing.T, tm *Terminal, y uint) *Page {
	t.Helper()
	return &zigUnwrap(t, tm.screens.active.pages.Pin(tm6Active(0, y))).node.data
}

// ---------------------------------------------------------------------------
// scrollUp, scrollDown
// ---------------------------------------------------------------------------

// Zig: test "term51: scrollUp: full screen counts, scrollback and clamping"
func TestTm6_term51_scrollUp_full_screen_counts_scrollback_and_clamping(t *testing.T) {
	testStdCapacity(t)
	type Case struct {
		count      uint
		scrollback uint
		rows       [5]string
	}
	cases := []Case{
		{count: 0, scrollback: 0, rows: [5]string{"ABCDE", "FGHIJ", "KLMNO", "PQRST", "UVWXY"}},
		{count: 1, scrollback: 1, rows: [5]string{"FGHIJ", "KLMNO", "PQRST", "UVWXY", "....."}},
		{count: 2, scrollback: 2, rows: [5]string{"KLMNO", "PQRST", "UVWXY", ".....", "....."}},
		{count: 4, scrollback: 4, rows: [5]string{"UVWXY", ".....", ".....", ".....", "....."}},
		{count: 5, scrollback: 5, rows: [5]string{".....", ".....", ".....", ".....", "....."}},
		{count: 6, scrollback: 5, rows: [5]string{".....", ".....", ".....", ".....", "....."}},
		{count: ^uint(0), scrollback: 5, rows: [5]string{".....", ".....", ".....", ".....", "....."}},
	}
	for _, c := range cases {
		t.Logf("scrollUp(%d)", c.count)
		func() {
			tm := tm6Init(t, 5, 5)
			defer tm.Deinit()
			tm6TFill(t, tm)
			tm.SetCursorPos(3, 4)
			tm.screens.active.cursor.pending_wrap = true
			tm6ScrollUp(t, tm, c.count)
			tm6TExpectGrid(t, tm, c.rows[:]...)
			tm6TExpectCursor(t, tm, 3, 2)
			expect(t, tm.screens.active.cursor.pending_wrap, "t.screens.active.cursor.pending_wrap")
			expectEqual(t, c.scrollback, tm6TScrollbackRows(tm))
		}()
	}
}

// Zig: test "term51: scrollUp: scrolled-out rows land in scrollback in order"
func TestTm6_term51_scrollUp_scrolled_out_rows_land_in_scrollback_in_order(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 5, 5)
	defer tm.Deinit()
	tm6TFill(t, tm)

	tm6ScrollUp(t, tm, 3)
	tm.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	str, err := tm.PlainString()
	zigTry(t, err)
	expectEqualStrings(t, "ABCDE\nFGHIJ\nKLMNO\nPQRST\nUVWXY", string(str))
}

// Zig: test "term51: scrollUp: top-anchored region with a bottom margin keeps the rows below"
func TestTm6_term51_scrollUp_top_anchored_region_with_a_bottom_margin_keeps_the_rows_below(t *testing.T) {
	testStdCapacity(t)
	type Case struct {
		count      uint
		scrollback uint
		rows       [5]string
	}
	cases := []Case{
		{count: 1, scrollback: 1, rows: [5]string{"FGHIJ", "KLMNO", ".....", "PQRST", "UVWXY"}},
		{count: 2, scrollback: 2, rows: [5]string{"KLMNO", ".....", ".....", "PQRST", "UVWXY"}},
		{count: 3, scrollback: 3, rows: [5]string{".....", ".....", ".....", "PQRST", "UVWXY"}},
		{count: 4, scrollback: 3, rows: [5]string{".....", ".....", ".....", "PQRST", "UVWXY"}},
	}
	for _, c := range cases {
		func() {
			tm := tm6Init(t, 5, 5)
			defer tm.Deinit()
			tm6TFill(t, tm)
			tm.SetTopAndBottomMargin(1, 3)
			tm.SetCursorPos(5, 5)
			tm6ScrollUp(t, tm, c.count)
			tm6TExpectGrid(t, tm, c.rows[:]...)
			tm6TExpectCursor(t, tm, 4, 4)
			expectEqual(t, c.scrollback, tm6TScrollbackRows(tm))
		}()
	}
}

// Zig: test "term51: scrollUp: region below the top row or with margins creates no scrollback"
func TestTm6_term51_scrollUp_region_below_the_top_row_or_with_margins_creates_no_scrollback(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 5, 5)
	defer tm.Deinit()
	tm6TFill(t, tm)

	tm.SetTopAndBottomMargin(2, 4)
	tm6ScrollUp(t, tm, 1)
	tm6TExpectGrid(t, tm, "ABCDE", "KLMNO", "PQRST", ".....", "UVWXY")
	expectEqual(t, uint(0), tm6TScrollbackRows(tm))

	tm.SetTopAndBottomMargin(0, 0)
	tm.modes.Set(ModesModeEnableLeftAndRightMargin, true)
	tm.SetLeftAndRightMargin(2, 4)
	tm6ScrollUp(t, tm, 1)
	tm6TExpectGrid(t, tm, "ALMNE", "KQRSO", "P...T", ".VWX.", "U...Y")
}

// Zig: test "term51: scrollUp: cursor outside the region is restored"
func TestTm6_term51_scrollUp_cursor_outside_the_region_is_restored(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 5, 5)
	defer tm.Deinit()
	tm6TFill(t, tm)

	tm.SetTopAndBottomMargin(2, 3)
	tm.modes.Set(ModesModeEnableLeftAndRightMargin, true)
	tm.SetLeftAndRightMargin(2, 4)
	cursors := [][2]uint{{4, 4}, {0, 0}, {0, 2}, {4, 2}}
	for _, c := range cursors {
		tm.screens.active.CursorAbsolute(SizeCellCountInt(c[0]), SizeCellCountInt(c[1]))
		tm.screens.active.cursor.pending_wrap = true
		tm6ScrollUp(t, tm, 1)
		tm6TExpectCursor(t, tm, c[0], c[1])
		tm6TExpectCursorConsistent(t, tm)
		expect(t, tm.screens.active.cursor.pending_wrap, "t.screens.active.cursor.pending_wrap")
	}
	// Four scrolls of a two-row region: rows 1..2, columns 1..3 are blank.
	tm6TExpectGrid(t, tm, "ABCDE", "F...J", "K...O", "PQRST", "UVWXY")
}

// Zig: test "term51: scrollUp: background colour fills the new rows"
func TestTm6_term51_scrollUp_background_colour_fills_the_new_rows(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 5, 5)
	defer tm.Deinit()
	tm6TFill(t, tm)

	tm6SetAttribute(t, tm, tm6TBg)
	tm6ScrollUp(t, tm, 2)
	tm6TExpectGrid(t, tm, "KLMNO", "PQRST", "UVWXY", "#####", "#####")
	cell := tm6TCellAt(t, tm, 2, 4)
	expectEqual(t, PageCellContentTagBgColorRgb, cell.content_tag)
	expectEqual(t, PageCellRGB{r: 1, g: 2, b: 3}, cell.content.color_rgb)
}

// Zig: test "term51: scrollUp: alternate screen has no scrollback"
func TestTm6_term51_scrollUp_alternate_screen_has_no_scrollback(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 5, 5)
	defer tm.Deinit()

	tm6SwitchScreen(t, tm, ScreenSetKeyAlternate)
	tm6TFill(t, tm)
	tm6ScrollUp(t, tm, 2)
	tm6TExpectGrid(t, tm, "KLMNO", "PQRST", "UVWXY", ".....", ".....")
	expectEqual(t, uint(0), tm6TScrollbackRows(tm))
}

// Zig: test "term51: scrollUp: no scrollback configured"
func TestTm6_term51_scrollUp_no_scrollback_configured(t *testing.T) {
	testStdCapacity(t)
	opts := terminalOptionsDefault(5, 5)
	opts.max_scrollback = 0
	tm := tm6InitOpts(t, opts)
	defer tm.Deinit()
	tm6TFill(t, tm)

	tm.SetCursorPos(2, 2)
	tm6ScrollUp(t, tm, 3)
	tm6TExpectGrid(t, tm, "PQRST", "UVWXY", ".....", ".....", ".....")
	tm6TExpectCursor(t, tm, 1, 1)
	expectEqual(t, uint(0), tm6TScrollbackRows(tm))
}

// Zig: test "term51: scrollDown: counts and clamping"
func TestTm6_term51_scrollDown_counts_and_clamping(t *testing.T) {
	testStdCapacity(t)
	type Case struct {
		count uint
		rows  [5]string
	}
	cases := []Case{
		{count: 0, rows: [5]string{"ABCDE", "FGHIJ", "KLMNO", "PQRST", "UVWXY"}},
		{count: 1, rows: [5]string{".....", "ABCDE", "FGHIJ", "KLMNO", "PQRST"}},
		{count: 2, rows: [5]string{".....", ".....", "ABCDE", "FGHIJ", "KLMNO"}},
		{count: 4, rows: [5]string{".....", ".....", ".....", ".....", "ABCDE"}},
		{count: 5, rows: [5]string{".....", ".....", ".....", ".....", "....."}},
		{count: 6, rows: [5]string{".....", ".....", ".....", ".....", "....."}},
		{count: ^uint(0), rows: [5]string{".....", ".....", ".....", ".....", "....."}},
	}
	for _, c := range cases {
		t.Logf("scrollDown(%d)", c.count)
		func() {
			tm := tm6Init(t, 5, 5)
			defer tm.Deinit()
			tm6TFill(t, tm)
			tm.SetCursorPos(3, 4)
			tm.screens.active.cursor.pending_wrap = true
			tm.ScrollDown(c.count)
			tm6TExpectGrid(t, tm, c.rows[:]...)
			tm6TExpectCursor(t, tm, 3, 2)
			expect(t, tm.screens.active.cursor.pending_wrap, "t.screens.active.cursor.pending_wrap")
			expectEqual(t, uint(0), tm6TScrollbackRows(tm))
		}()
	}
}

// Zig: test "term51: scrollDown: works with the cursor anywhere and restores it"
func TestTm6_term51_scrollDown_works_with_the_cursor_anywhere_and_restores_it(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 5, 5)
	defer tm.Deinit()
	tm6TFill(t, tm)

	tm.SetTopAndBottomMargin(2, 3)
	tm.modes.Set(ModesModeEnableLeftAndRightMargin, true)
	tm.SetLeftAndRightMargin(2, 4)
	cursors := [][2]uint{{4, 4}, {0, 0}, {0, 2}, {4, 2}, {2, 1}}
	for _, c := range cursors {
		tm.screens.active.CursorAbsolute(SizeCellCountInt(c[0]), SizeCellCountInt(c[1]))
		tm.screens.active.cursor.pending_wrap = true
		tm.ScrollDown(1)
		tm6TExpectCursor(t, tm, c[0], c[1])
		tm6TExpectCursorConsistent(t, tm)
		expect(t, tm.screens.active.cursor.pending_wrap, "t.screens.active.cursor.pending_wrap")
	}
	tm6TExpectGrid(t, tm, "ABCDE", "F...J", "K...O", "PQRST", "UVWXY")
}

// Zig: test "term51: scrollDown: background colour fills the new rows"
func TestTm6_term51_scrollDown_background_colour_fills_the_new_rows(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 5, 5)
	defer tm.Deinit()
	tm6TFill(t, tm)

	tm6SetAttribute(t, tm, tm6Attr256Bg(0))
	tm.ScrollDown(2)
	tm6TExpectGrid(t, tm, "#####", "#####", "ABCDE", "FGHIJ", "KLMNO")
	cell := tm6TCellAt(t, tm, 0, 0)
	expectEqual(t, PageCellContentTagBgColorPalette, cell.content_tag)
	expectEqual(t, uint8(0), cell.content.color_palette)
}

// Zig: test "term51: scrollDown: alternate screen"
func TestTm6_term51_scrollDown_alternate_screen(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 5, 5)
	defer tm.Deinit()

	tm6SwitchScreen(t, tm, ScreenSetKeyAlternate)
	tm6TFill(t, tm)
	tm.ScrollDown(1)
	tm6TExpectGrid(t, tm, ".....", "ABCDE", "FGHIJ", "KLMNO", "PQRST")
}

// ---------------------------------------------------------------------------
// scrollViewport
// ---------------------------------------------------------------------------

// Zig: test "term51: scrollViewport: top and bottom"
func TestTm6_term51_scrollViewport_top_and_bottom(t *testing.T) {
	testStdCapacity(t)
	tm := tm6TScrollbackTerminal(t)
	defer tm.Deinit()

	expectEqual(t, uint(7), tm6TScrollbackRows(tm))
	expect(t, tm.screens.active.ViewportIsBottom(), "t.screens.active.viewportIsBottom()")
	expectEqual(t, uint(7), tm6TViewportTop(t, tm))
	tm6TExpectViewport(t, tm, "H\nI\nJ")

	tm.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	expectEqual(t, PageListViewportTop, tm.screens.active.pages.viewport.Tag)
	expectEqual(t, uint(0), tm6TViewportTop(t, tm))
	expect(t, !tm.screens.active.ViewportIsBottom(), "!t.screens.active.viewportIsBottom()")
	tm6TExpectViewport(t, tm, "A\nB\nC")

	// Top again is a no-op.
	tm.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	expectEqual(t, uint(0), tm6TViewportTop(t, tm))

	tm.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportBottom})
	expectEqual(t, PageListViewportActive, tm.screens.active.pages.viewport.Tag)
	expect(t, tm.screens.active.ViewportIsBottom(), "t.screens.active.viewportIsBottom()")
	tm6TExpectViewport(t, tm, "H\nI\nJ")

	// Bottom again is a no-op.
	tm.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportBottom})
	expect(t, tm.screens.active.ViewportIsBottom(), "t.screens.active.viewportIsBottom()")
}

// Zig: test "term51: scrollViewport: delta from the bottom"
func TestTm6_term51_scrollViewport_delta_from_the_bottom(t *testing.T) {
	testStdCapacity(t)
	tm := tm6TScrollbackTerminal(t)
	defer tm.Deinit()

	// Zero and positive deltas at the bottom stay at the bottom.
	tm.ScrollViewport(tm6ScrollViewportDelta(0))
	expect(t, tm.screens.active.ViewportIsBottom(), "t.screens.active.viewportIsBottom()")
	tm.ScrollViewport(tm6ScrollViewportDelta(3))
	expect(t, tm.screens.active.ViewportIsBottom(), "t.screens.active.viewportIsBottom()")

	tm.ScrollViewport(tm6ScrollViewportDelta(-1))
	expectEqual(t, uint(6), tm6TViewportTop(t, tm))
	tm6TExpectViewport(t, tm, "G\nH\nI")
	expect(t, !tm.screens.active.ViewportIsBottom(), "!t.screens.active.viewportIsBottom()")

	tm.ScrollViewport(tm6ScrollViewportDelta(-2))
	expectEqual(t, uint(4), tm6TViewportTop(t, tm))
	tm6TExpectViewport(t, tm, "E\nF\nG")

	// Forwards by one.
	tm.ScrollViewport(tm6ScrollViewportDelta(1))
	expectEqual(t, uint(5), tm6TViewportTop(t, tm))

	// Zero changes nothing in the middle.
	tm.ScrollViewport(tm6ScrollViewportDelta(0))
	expectEqual(t, uint(5), tm6TViewportTop(t, tm))

	// Forwards to exactly the active area snaps to the bottom.
	tm.ScrollViewport(tm6ScrollViewportDelta(2))
	expect(t, tm.screens.active.ViewportIsBottom(), "t.screens.active.viewportIsBottom()")
	expectEqual(t, PageListViewportActive, tm.screens.active.pages.viewport.Tag)
}

// Zig: test "term51: scrollViewport: delta clamps at both ends"
func TestTm6_term51_scrollViewport_delta_clamps_at_both_ends(t *testing.T) {
	testStdCapacity(t)
	tm := tm6TScrollbackTerminal(t)
	defer tm.Deinit()

	// From the bottom, one past the top and far past it.
	tm.ScrollViewport(tm6ScrollViewportDelta(-8))
	expectEqual(t, uint(0), tm6TViewportTop(t, tm))
	expectEqual(t, PageListViewportTop, tm.screens.active.pages.viewport.Tag)
	tm.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportBottom})
	tm.ScrollViewport(tm6ScrollViewportDelta(-1_000_000_000))
	expectEqual(t, uint(0), tm6TViewportTop(t, tm))
	expectEqual(t, PageListViewportTop, tm.screens.active.pages.viewport.Tag)

	// At the top, backwards stays and forwards moves.
	tm.ScrollViewport(tm6ScrollViewportDelta(-1))
	expectEqual(t, uint(0), tm6TViewportTop(t, tm))
	tm.ScrollViewport(tm6ScrollViewportDelta(0))
	expectEqual(t, uint(0), tm6TViewportTop(t, tm))
	tm.ScrollViewport(tm6ScrollViewportDelta(2))
	expectEqual(t, uint(2), tm6TViewportTop(t, tm))
	tm6TExpectViewport(t, tm, "C\nD\nE")

	// Far past the bottom from the middle.
	tm.ScrollViewport(tm6ScrollViewportDelta(1_000_000_000))
	expectEqual(t, PageListViewportActive, tm.screens.active.pages.viewport.Tag)
	expect(t, tm.screens.active.ViewportIsBottom(), "t.screens.active.viewportIsBottom()")

	// Far past the top from the middle.
	tm.ScrollViewport(tm6ScrollViewportDelta(-3))
	tm.ScrollViewport(tm6ScrollViewportDelta(-1_000_000_000))
	expectEqual(t, uint(0), tm6TViewportTop(t, tm))
	expectEqual(t, PageListViewportTop, tm.screens.active.pages.viewport.Tag)

	// Exactly to the bottom from the top.
	tm.ScrollViewport(tm6ScrollViewportDelta(7))
	expect(t, tm.screens.active.ViewportIsBottom(), "t.screens.active.viewportIsBottom()")
}

// Zig: test "term51: scrollViewport: nothing to scroll back to"
func TestTm6_term51_scrollViewport_nothing_to_scroll_back_to(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 4, 3)
	defer tm.Deinit()
	tm6PrintString(t, tm, "ab")

	tm.ScrollViewport(tm6ScrollViewportDelta(-5))
	expectEqual(t, uint(0), tm6TViewportTop(t, tm))
	expectEqual(t, uint(0), tm6TScrollbackRows(tm))
	tm.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	expectEqual(t, uint(0), tm6TViewportTop(t, tm))
	tm.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportBottom})
	expect(t, tm.screens.active.ViewportIsBottom(), "t.screens.active.viewportIsBottom()")
	tm6TExpectViewport(t, tm, "ab")
}

// Zig: test "term51: scrollViewport: screens without scrollback never scroll"
func TestTm6_term51_scrollViewport_screens_without_scrollback_never_scroll(t *testing.T) {
	testStdCapacity(t)
	// Alternate screen.
	func() {
		tm := tm6Init(t, 4, 3)
		defer tm.Deinit()
		tm6SwitchScreen(t, tm, ScreenSetKeyAlternate)
		tm.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
		expectEqual(t, PageListViewportActive, tm.screens.active.pages.viewport.Tag)
		tm.ScrollViewport(tm6ScrollViewportDelta(-3))
		expectEqual(t, PageListViewportActive, tm.screens.active.pages.viewport.Tag)
	}()
	// max_scrollback = 0.
	func() {
		opts := terminalOptionsDefault(4, 3)
		opts.max_scrollback = 0
		tm := tm6InitOpts(t, opts)
		defer tm.Deinit()
		for i := 0; i < 8; i++ {
			tm6Linefeed(t, tm)
		}
		tm.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
		expectEqual(t, PageListViewportActive, tm.screens.active.pages.viewport.Tag)
		tm.ScrollViewport(tm6ScrollViewportDelta(-3))
		expectEqual(t, PageListViewportActive, tm.screens.active.pages.viewport.Tag)
		tm.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportBottom})
		expectEqual(t, PageListViewportActive, tm.screens.active.pages.viewport.Tag)
	}()
}

// Zig: test "term51: scrollViewport: does not move the cursor or the active area"
func TestTm6_term51_scrollViewport_does_not_move_the_cursor_or_the_active_area(t *testing.T) {
	testStdCapacity(t)
	tm := tm6TScrollbackTerminal(t)
	defer tm.Deinit()

	x := uint(tm.screens.active.cursor.x)
	y := uint(tm.screens.active.cursor.y)
	tm.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	tm.ScrollViewport(tm6ScrollViewportDelta(3))
	tm6TExpectCursor(t, tm, x, y)
	tm6TExpectCursorConsistent(t, tm)
	tm.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportBottom})
	tm6TExpectViewport(t, tm, "H\nI\nJ")
}

// ---------------------------------------------------------------------------
// rowWillBeShifted
// ---------------------------------------------------------------------------

// Zig: test "term51: rowWillBeShifted: spacer head at the last column"
func TestTm6_term51_rowWillBeShifted_spacer_head_at_the_last_column(t *testing.T) {
	testStdCapacity(t)
	// The head is turned into an empty narrow cell when the region reaches
	// the last column or starts in column 0 or 1.
	type Case struct {
		left    uint
		right   uint
		cleared bool
	}
	cases := []Case{
		{left: 0, right: 9, cleared: true},
		{left: 0, right: 5, cleared: true},
		{left: 0, right: 1, cleared: true},
		{left: 1, right: 5, cleared: true},
		{left: 1, right: 8, cleared: true},
		{left: 1, right: 9, cleared: true},
		{left: 2, right: 9, cleared: true},
		{left: 3, right: 9, cleared: true},
		{left: 8, right: 9, cleared: true},
		{left: 2, right: 3, cleared: false},
		{left: 2, right: 5, cleared: false},
		{left: 2, right: 8, cleared: false},
		{left: 3, right: 8, cleared: false},
		{left: 7, right: 8, cleared: false},
	}
	for _, c := range cases {
		t.Logf("region columns %d..%d", c.left, c.right)
		func() {
			tm := tm6TSpacerHeadTerminal(t)
			defer tm.Deinit()
			tm6TExpectGrid(t, tm, "abcdefghi>", "W_........", "..........")
			expect(t, tm6TRowAt(t, tm, 0).wrap, "term51TRowAt(&t, 0).wrap")

			tm6TShiftRow(t, tm, 0, c.left, c.right)

			expected := "abcdefghi>"
			if c.cleared {
				expected = "abcdefghi."
			}
			expectEqualStrings(t, expected, tm6TGridRow(t, tm, 0))
			// The row's wrap flag and the next row are not touched.
			expect(t, tm6TRowAt(t, tm, 0).wrap, "term51TRowAt(&t, 0).wrap")
			expectEqualStrings(t, "W_........", tm6TGridRow(t, tm, 1))
		}()
	}
}

// Zig: test "term51: rowWillBeShifted: other cells at the last column are left alone"
func TestTm6_term51_rowWillBeShifted_other_cells_at_the_last_column_are_left_alone(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 10, 2)
	defer tm.Deinit()

	// A narrow cell with content, and an empty cell.
	tm6TPutRow(t, tm, 0, "abcdefghij")
	var raw_before [10]uint64
	tm6TRawRow(t, tm, 0, raw_before[:])
	tm6TShiftRow(t, tm, 0, 0, 9)
	var raw_after [10]uint64
	expectEqualSlices(t, raw_before[:], tm6TRawRow(t, tm, 0, raw_after[:]))

	// Empty row.
	tm6TRawRow(t, tm, 1, raw_before[:])
	tm6TShiftRow(t, tm, 1, 0, 9)
	expectEqualSlices(t, raw_before[:], tm6TRawRow(t, tm, 1, raw_after[:]))
}

// Zig: test "term51: rowWillBeShifted: wide character split by the left margin"
func TestTm6_term51_rowWillBeShifted_wide_character_split_by_the_left_margin(t *testing.T) {
	testStdCapacity(t)
	// Row: a, wide at columns 1-2, then "bcdefg" at 3..8, h at 9.
	type Case struct {
		left     uint
		right    uint
		expected string
	}
	cases := []Case{
		// left margin on the spacer tail: the pair is cleared
		{left: 2, right: 6, expected: "a..bcdefgh"},
		{left: 2, right: 9, expected: "a..bcdefgh"},
		{left: 2, right: 3, expected: "a..bcdefgh"},
		// left margin on the wide cell itself, or past the tail: nothing is split
		{left: 1, right: 6, expected: "aW_bcdefgh"},
		{left: 0, right: 6, expected: "aW_bcdefgh"},
		{left: 3, right: 6, expected: "aW_bcdefgh"},
		{left: 4, right: 6, expected: "aW_bcdefgh"},
	}
	for _, c := range cases {
		t.Logf("region columns %d..%d", c.left, c.right)
		func() {
			tm := tm6Init(t, 10, 2)
			defer tm.Deinit()
			tm6TPutRow(t, tm, 0, "a中bcdefgh")
			expectEqualStrings(t, "aW_bcdefgh", tm6TGridRow(t, tm, 0))

			tm6TShiftRow(t, tm, 0, c.left, c.right)
			expectEqualStrings(t, c.expected, tm6TGridRow(t, tm, 0))
		}()
	}
}

// Zig: test "term51: rowWillBeShifted: wide character split by the right margin"
func TestTm6_term51_rowWillBeShifted_wide_character_split_by_the_right_margin(t *testing.T) {
	testStdCapacity(t)
	// Row: "abcdefg", wide at columns 7-8, h at 9.
	type Case struct {
		left     uint
		right    uint
		expected string
	}
	cases := []Case{
		// right margin on the wide cell: the pair is cleared
		{left: 0, right: 7, expected: "abcdefg..h"},
		{left: 3, right: 7, expected: "abcdefg..h"},
		{left: 6, right: 7, expected: "abcdefg..h"},
		// right margin on the spacer tail, or before the wide cell: not split
		{left: 0, right: 8, expected: "abcdefgW_h"},
		{left: 3, right: 8, expected: "abcdefgW_h"},
		{left: 0, right: 6, expected: "abcdefgW_h"},
		{left: 2, right: 5, expected: "abcdefgW_h"},
		{left: 0, right: 9, expected: "abcdefgW_h"},
	}
	for _, c := range cases {
		t.Logf("region columns %d..%d", c.left, c.right)
		func() {
			tm := tm6Init(t, 10, 2)
			defer tm.Deinit()
			tm6TPutRow(t, tm, 0, "abcdefg中h")
			expectEqualStrings(t, "abcdefgW_h", tm6TGridRow(t, tm, 0))

			tm6TShiftRow(t, tm, 0, c.left, c.right)
			expectEqualStrings(t, c.expected, tm6TGridRow(t, tm, 0))
		}()
	}
}

// Zig: test "term51: rowWillBeShifted: wide characters split at both margins"
func TestTm6_term51_rowWillBeShifted_wide_characters_split_at_both_margins(t *testing.T) {
	testStdCapacity(t)
	// Row: a, wide 1-2, "bcd" 3..5, wide 6-7, "ef" 8-9.
	type Case struct {
		left     uint
		right    uint
		expected string
	}
	cases := []Case{
		{left: 2, right: 6, expected: "a..bcd..ef"},
		{left: 2, right: 7, expected: "a..bcdW_ef"},
		{left: 1, right: 6, expected: "aW_bcd..ef"},
		{left: 1, right: 7, expected: "aW_bcdW_ef"},
		{left: 3, right: 5, expected: "aW_bcdW_ef"},
		{left: 3, right: 6, expected: "aW_bcd..ef"},
		{left: 2, right: 5, expected: "a..bcdW_ef"},
	}
	for _, c := range cases {
		t.Logf("region columns %d..%d", c.left, c.right)
		func() {
			tm := tm6Init(t, 10, 2)
			defer tm.Deinit()
			tm6TPutRow(t, tm, 0, "a中bcd中ef")
			expectEqualStrings(t, "aW_bcdW_ef", tm6TGridRow(t, tm, 0))

			tm6TShiftRow(t, tm, 0, c.left, c.right)
			expectEqualStrings(t, c.expected, tm6TGridRow(t, tm, 0))
		}()
	}
}

// Zig: test "term51: rowWillBeShifted: cleared wide cells lose their grapheme"
func TestTm6_term51_rowWillBeShifted_cleared_wide_cells_lose_their_grapheme(t *testing.T) {
	testStdCapacity(t)

	// Left margin split, grapheme on the wide cell, only grapheme in the row.
	func() {
		tm := tm6Init(t, 10, 2)
		defer tm.Deinit()
		tm6TPutRow(t, tm, 0, "a中́bcdefgh")
		lc := zigUnwrap(t, tm.screens.active.pages.GetCell(tm6Active(1, 0)))
		expect(t, lc.cell.HasGrapheme(), "lc.cell.hasGrapheme()")
		expectEqual(t, uint(1), lc.node.data.GraphemeCount())
		expect(t, lc.row.grapheme, "lc.row.grapheme")

		tm6TShiftRow(t, tm, 0, 2, 6)
		expectEqualStrings(t, "a..bcdefgh", tm6TGridRow(t, tm, 0))
		cell := tm6TCellAt(t, tm, 1, 0)
		expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
		expectEqual(t, PageCellContentTagCodepoint, cell.content_tag)
		expectEqual(t, uint32(0), cell.content.codepoint)
		expectEqual(t, uint(0), lc.node.data.GraphemeCount())
		expect(t, !tm6TRowAt(t, tm, 0).grapheme, "!term51TRowAt(&t, 0).grapheme")
	}()

	// Right margin split, grapheme on the wide cell.
	func() {
		tm := tm6Init(t, 10, 2)
		defer tm.Deinit()
		tm6TPutRow(t, tm, 0, "abcdefg中́h")
		lc := zigUnwrap(t, tm.screens.active.pages.GetCell(tm6Active(7, 0)))
		expect(t, lc.cell.HasGrapheme(), "lc.cell.hasGrapheme()")

		tm6TShiftRow(t, tm, 0, 0, 7)
		expectEqualStrings(t, "abcdefg..h", tm6TGridRow(t, tm, 0))
		cell := tm6TCellAt(t, tm, 7, 0)
		expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
		expectEqual(t, uint32(0), cell.content.codepoint)
		expectEqual(t, uint(0), lc.node.data.GraphemeCount())
		expect(t, !tm6TRowAt(t, tm, 0).grapheme, "!term51TRowAt(&t, 0).grapheme")
	}()

	// Both margins with graphemes on both wide cells.
	func() {
		tm := tm6Init(t, 10, 2)
		defer tm.Deinit()
		tm6TPutRow(t, tm, 0, "a中́bcd中́ef")
		lc := zigUnwrap(t, tm.screens.active.pages.GetCell(tm6Active(0, 0)))
		expectEqual(t, uint(2), lc.node.data.GraphemeCount())

		tm6TShiftRow(t, tm, 0, 2, 6)
		expectEqualStrings(t, "a..bcd..ef", tm6TGridRow(t, tm, 0))
		expectEqual(t, uint(0), lc.node.data.GraphemeCount())
		expect(t, !tm6TRowAt(t, tm, 0).grapheme, "!term51TRowAt(&t, 0).grapheme")
	}()

	// Another grapheme in the row keeps the row flag and its entry.
	func() {
		tm := tm6Init(t, 10, 2)
		defer tm.Deinit()
		tm6TPutRow(t, tm, 0, "a中́bcd́efgh")
		lc := zigUnwrap(t, tm.screens.active.pages.GetCell(tm6Active(0, 0)))
		expectEqual(t, uint(2), lc.node.data.GraphemeCount())

		tm6TShiftRow(t, tm, 0, 2, 6)
		expectEqualStrings(t, "a..bcdefgh", tm6TGridRow(t, tm, 0))
		expectEqual(t, uint(1), lc.node.data.GraphemeCount())
		expect(t, tm6TRowAt(t, tm, 0).grapheme, "term51TRowAt(&t, 0).grapheme")
		cell := tm6TCellAt(t, tm, 5, 0)
		expect(t, cell.HasGrapheme(), "term51TCellAt(&t, 5, 0).hasGrapheme()")
	}()

	// A narrow grapheme cell on the margins is not a wide cell: untouched.
	func() {
		tm := tm6Init(t, 10, 2)
		defer tm.Deinit()
		tm6TPutRow(t, tm, 0, "abcd́efghi")
		lc := zigUnwrap(t, tm.screens.active.pages.GetCell(tm6Active(0, 0)))
		tm6TShiftRow(t, tm, 0, 3, 3+3)
		cell := tm6TCellAt(t, tm, 3, 0)
		expect(t, cell.HasGrapheme(), "term51TCellAt(&t, 3, 0).hasGrapheme()")
		expectEqual(t, uint(1), lc.node.data.GraphemeCount())
		expect(t, tm6TRowAt(t, tm, 0).grapheme, "term51TRowAt(&t, 0).grapheme")
	}()
}

// Zig: test "term51: rowWillBeShifted: grapheme on a wide cell that is not split is kept"
func TestTm6_term51_rowWillBeShifted_grapheme_on_a_wide_cell_that_is_not_split_is_kept(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 10, 2)
	defer tm.Deinit()
	tm6TPutRow(t, tm, 0, "a中́bcdefgh")
	lc := zigUnwrap(t, tm.screens.active.pages.GetCell(tm6Active(1, 0)))

	// Margins that start on the wide cell and end past the tail.
	tm6TShiftRow(t, tm, 0, 1, 6)
	cell := tm6TCellAt(t, tm, 1, 0)
	expect(t, cell.HasGrapheme(), "term51TCellAt(&t, 1, 0).hasGrapheme()")
	expectEqual(t, uint(1), lc.node.data.GraphemeCount())
	expect(t, tm6TRowAt(t, tm, 0).grapheme, "term51TRowAt(&t, 0).grapheme")
	expectEqualStrings(t, "aW_bcdefgh", tm6TGridRow(t, tm, 0))
}

// Zig: test "term51: rowWillBeShifted: the cleared wide cell keeps its style"
func TestTm6_term51_rowWillBeShifted_the_cleared_wide_cell_keeps_its_style(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 10, 2)
	defer tm.Deinit()

	tm6SetAttribute(t, tm, tm6AttrBold)
	tm6TPutRow(t, tm, 0, "a中bcdefgh")
	before := tm6TCellAt(t, tm, 1, 0)
	expect(t, before.style_id != StyleDefaultId, "before.style_id != style.default_id")

	tm6TShiftRow(t, tm, 0, 2, 6)
	after := tm6TCellAt(t, tm, 1, 0)
	expectEqual(t, before.style_id, after.style_id)
	expectEqual(t, uint32(0), after.content.codepoint)
	expectEqual(t, PageCellWideNarrow, after.wide)
	expectEqual(t, PageCellWideNarrow, tm6TCellAt(t, tm, 2, 0).wide)
}

// Zig: test "term51: rowWillBeShifted: only the split cells and the given row change"
func TestTm6_term51_rowWillBeShifted_only_the_split_cells_and_the_given_row_change(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 10, 3)
	defer tm.Deinit()
	tm6TPutRow(t, tm, 0, "a中bcd中ef")
	tm6TPutRow(t, tm, 1, "a中bcd中ef")

	var before0 [10]uint64
	var before1 [10]uint64
	tm6TRawRow(t, tm, 0, before0[:])
	tm6TRawRow(t, tm, 1, before1[:])

	tm6TShiftRow(t, tm, 0, 2, 6)

	var after0 [10]uint64
	var after1 [10]uint64
	tm6TRawRow(t, tm, 0, after0[:])
	tm6TRawRow(t, tm, 1, after1[:])
	expectEqualSlices(t, before1[:], after1[:])
	for x := 0; x < 10; x++ {
		changed := x == 1 || x == 2 || x == 6 || x == 7
		expectEqual(t, !changed, before0[x] == after0[x])
	}
	// The region columns outside the split pairs are bit-identical.
	expectEqual(t, uint(0), tm6TScrollbackRows(tm))
}

// Zig: test "term51: rowWillBeShifted: wide pair fully inside or outside the margins is untouched"
func TestTm6_term51_rowWillBeShifted_wide_pair_fully_inside_or_outside_the_margins_is_untouched(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 10, 2)
	defer tm.Deinit()
	tm6TPutRow(t, tm, 0, "ab中cdefgh")
	var before [10]uint64
	var after [10]uint64
	tm6TRawRow(t, tm, 0, before[:])

	regions := [][2]uint{{2, 5}, {2, 3}, {0, 4}, {4, 8}, {2, 9}, {0, 9}}
	for _, r := range regions {
		tm6TShiftRow(t, tm, 0, r[0], r[1])
		expectEqualSlices(t, before[:], tm6TRawRow(t, tm, 0, after[:]))
	}
	// A region that starts on the spacer tail does split the pair.
	tm6TShiftRow(t, tm, 0, 3, 8)
	expectEqualStrings(t, "ab..cdefgh", tm6TGridRow(t, tm, 0))
}

// Zig: test "term51: rowWillBeShifted: scrollUp with margins splitting wide characters"
func TestTm6_term51_rowWillBeShifted_scrollUp_with_margins_splitting_wide_characters(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 10, 3)
	defer tm.Deinit()
	tm6TPutRow(t, tm, 0, "a中bcdefgh")
	tm6TPutRow(t, tm, 1, "ij中klmnop")
	tm6TPutRow(t, tm, 2, "qrstuvwxyz")
	tm6TExpectGrid(t, tm, "aW_bcdefgh", "ijW_klmnop", "qrstuvwxyz")

	tm6TRegion(tm, 0, 2, 2, 5)
	tm6ScrollUp(t, tm, 1)
	tm6TExpectGrid(t, tm, "a.W_klefgh", "ijstuvmnop", "qr....wxyz")
}

// Zig: test "term51: rowWillBeShifted: scrollDown with margins splitting wide characters"
func TestTm6_term51_rowWillBeShifted_scrollDown_with_margins_splitting_wide_characters(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 10, 3)
	defer tm.Deinit()
	tm6TPutRow(t, tm, 0, "a中bcdefgh")
	tm6TPutRow(t, tm, 1, "ij中klmnop")
	tm6TPutRow(t, tm, 2, "qrstuvwxyz")

	tm6TRegion(tm, 0, 2, 2, 5)
	tm.ScrollDown(1)
	tm6TExpectGrid(t, tm, "a.....efgh", "ij.bcdmnop", "qrW_klwxyz")
}

// Zig: test "term51: rowWillBeShifted: scrollUp with a wide character on the right margin"
func TestTm6_term51_rowWillBeShifted_scrollUp_with_a_wide_character_on_the_right_margin(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 10, 3)
	defer tm.Deinit()
	tm6TPutRow(t, tm, 0, "abcdef中gh")
	tm6TPutRow(t, tm, 1, "ijklmnopqr")
	tm6TPutRow(t, tm, 2, "stuvwxyzAB")
	tm6TExpectGrid(t, tm, "abcdefW_gh", "ijklmnopqr", "stuvwxyzAB")

	tm6TRegion(tm, 0, 2, 0, 6)
	tm6ScrollUp(t, tm, 1)
	tm6TExpectGrid(t, tm, "ijklmno.gh", "stuvwxypqr", ".......zAB")
}

// Zig: test "term51: rowWillBeShifted: scrollUp turns a spacer head at the last column into an empty cell"
func TestTm6_term51_rowWillBeShifted_scrollUp_turns_a_spacer_head_at_the_last_column_into_an_empty_cell(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 10, 4)
	defer tm.Deinit()
	tm6TPutRow(t, tm, 1, "abcdefghi中")
	tm6TPutRow(t, tm, 3, "klmnopqrst")
	tm6TExpectGrid(t, tm, "..........", "abcdefghi>", "W_........", "klmnopqrst")
	expect(t, tm6TRowAt(t, tm, 1).wrap, "term51TRowAt(&t, 1).wrap")
	expect(t, tm6TRowAt(t, tm, 2).wrap_continuation, "term51TRowAt(&t, 2).wrap_continuation")

	tm.SetTopAndBottomMargin(2, 4)
	tm6ScrollUp(t, tm, 1)
	tm6TExpectGrid(t, tm, "..........", "W_........", "klmnopqrst", "..........")
	for y := uint(0); y < 4; y++ {
		expect(t, !tm6TRowAt(t, tm, y).wrap, "!term51TRowAt(&t, y).wrap")
		expect(t, !tm6TRowAt(t, tm, y).wrap_continuation, "!term51TRowAt(&t, y).wrap_continuation")
	}
}

// ---------------------------------------------------------------------------
// insertLines
// ---------------------------------------------------------------------------

// Zig: test "term51: insertLines: zero count changes nothing and keeps cursor and pending wrap"
func TestTm6_term51_insertLines_zero_count_changes_nothing_and_keeps_cursor_and_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 5, 2)
	tm6Print(t, tm, 'Z')
	tm6UExpectCursor(t, tm, 5, 2, true)
	tm.clearDirty()

	tm.InsertLines(0)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqZ", "stuvwx", "yzABCD", "EFGHIJ")
	tm6UExpectCursor(t, tm, 5, 2, true)
	tm6UExpectDirty(t, tm, "......")
	expect(t, !tm6UKittyDirty(tm), "!term51UKittyDirty(&t)")
}

// Zig: test "term51: insertLines: one line in the middle shifts down and moves cursor to column zero"
func TestTm6_term51_insertLines_one_line_in_the_middle_shifts_down_and_moves_cursor_to_column_zero(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 3, 2)
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "......", "mnopqr", "stuvwx", "yzABCD")
	tm6UExpectCursor(t, tm, 0, 2, false)
	tm6UExpectDirty(t, tm, "..DDDD")
	// Scrolling always dirties the kitty images when kitty graphics exist.
	// build_options.kitty_graphics is false in the library build.
	expectEqual(t, false, tm6UKittyDirty(tm))
}

// Zig: test "term51: insertLines: at the top row shifts the whole screen and drops the last row"
func TestTm6_term51_insertLines_at_the_top_row_shifts_the_whole_screen_and_drops_the_last_row(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 0, 0)
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "......", "abcdef", "ghijkl", "mnopqr", "stuvwx", "yzABCD")
	tm6UExpectDirty(t, tm, "DDDDDD")
}

// Zig: test "term51: insertLines: count one less than the remaining rows keeps one row"
func TestTm6_term51_insertLines_count_one_less_than_the_remaining_rows_keeps_one_row(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	// Cursor on row 3, so three rows remain: count 2 leaves "stuvwx" last.
	tm6UAlphaAt(t, tm, 2, 3)
	tm.InsertLines(2)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "......", "......", "stuvwx")
	tm6UExpectCursor(t, tm, 0, 3, false)
	tm6UExpectDirty(t, tm, "...DDD")
}

// Zig: test "term51: insertLines: count equal to the remaining rows clears them all"
func TestTm6_term51_insertLines_count_equal_to_the_remaining_rows_clears_them_all(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 2, 3)
	tm.InsertLines(3)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "......", "......", "......")
	tm6UExpectCursor(t, tm, 0, 3, false)
	tm6UExpectDirty(t, tm, "...DDD")
}

// Zig: test "term51: insertLines: count one more than the remaining rows is clamped"
func TestTm6_term51_insertLines_count_one_more_than_the_remaining_rows_is_clamped(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 2, 3)
	tm.InsertLines(4)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "......", "......", "......")
	tm6UExpectCursor(t, tm, 0, 3, false)
	tm6UExpectDirty(t, tm, "...DDD")
}

// Zig: test "term51: insertLines: maximum count is clamped to the remaining rows"
func TestTm6_term51_insertLines_maximum_count_is_clamped_to_the_remaining_rows(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 2, 3)
	tm.InsertLines(^uint(0))

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "......", "......", "......")
	tm6UExpectCursor(t, tm, 0, 3, false)
	tm6UExpectDirty(t, tm, "...DDD")
}

// Zig: test "term51: insertLines: on the last row only that row is cleared"
func TestTm6_term51_insertLines_on_the_last_row_only_that_row_is_cleared(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 4, 5)
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "stuvwx", "yzABCD", "......")
	tm6UExpectCursor(t, tm, 0, 5, false)
	tm6UExpectDirty(t, tm, ".....D")
}

// Zig: test "term51: insertLines: maximum count on the last row clears only that row"
func TestTm6_term51_insertLines_maximum_count_on_the_last_row_clears_only_that_row(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 4, 5)
	tm.InsertLines(^uint(0))

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "stuvwx", "yzABCD", "......")
	tm6UExpectDirty(t, tm, ".....D")
}

// Zig: test "term51: insertLines: top and bottom margins confine the shift"
func TestTm6_term51_insertLines_top_and_bottom_margins_confine_the_shift(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UFill(t, tm, tm6UAlpha6)
	tm.SetTopAndBottomMargin(2, 4)
	tm.SetCursorPos(3, 5)
	tm.clearDirty()
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "......", "mnopqr", "yzABCD", "EFGHIJ")
	tm6UExpectCursor(t, tm, 0, 2, false)
	tm6UExpectDirty(t, tm, "..DD..")
}

// Zig: test "term51: insertLines: cursor on the top margin row with margins"
func TestTm6_term51_insertLines_cursor_on_the_top_margin_row_with_margins(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UFill(t, tm, tm6UAlpha6)
	tm.SetTopAndBottomMargin(2, 4)
	tm.SetCursorPos(2, 1)
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "......", "ghijkl", "mnopqr", "yzABCD", "EFGHIJ")
	tm6UExpectCursor(t, tm, 0, 1, false)
}

// Zig: test "term51: insertLines: cursor on the bottom margin row only clears that row"
func TestTm6_term51_insertLines_cursor_on_the_bottom_margin_row_only_clears_that_row(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UFill(t, tm, tm6UAlpha6)
	tm.SetTopAndBottomMargin(2, 4)
	tm.SetCursorPos(4, 3)
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "......", "yzABCD", "EFGHIJ")
	tm6UExpectCursor(t, tm, 0, 3, false)
}

// Zig: test "term51: insertLines: count larger than the margin region clears the region only"
func TestTm6_term51_insertLines_count_larger_than_the_margin_region_clears_the_region_only(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UFill(t, tm, tm6UAlpha6)
	tm.SetTopAndBottomMargin(2, 4)
	tm.SetCursorPos(3, 1)
	tm.InsertLines(100)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "......", "......", "yzABCD", "EFGHIJ")
}

// Zig: test "term51: insertLines: cursor above the top margin does nothing and keeps pending wrap"
func TestTm6_term51_insertLines_cursor_above_the_top_margin_does_nothing_and_keeps_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UFill(t, tm, tm6UAlpha6)
	tm.SetTopAndBottomMargin(3, 5)
	tm.SetCursorPos(2, 6)
	tm6Print(t, tm, 'Z')
	tm6UExpectCursor(t, tm, 5, 1, true)
	tm.clearDirty()
	tm6UResetKittyDirty(tm)

	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkZ", "mnopqr", "stuvwx", "yzABCD", "EFGHIJ")
	tm6UExpectCursor(t, tm, 5, 1, true)
	tm6UExpectDirty(t, tm, "......")
	expect(t, !tm6UKittyDirty(tm), "!term51UKittyDirty(&t)")
}

// Zig: test "term51: insertLines: cursor below the bottom margin does nothing and keeps pending wrap"
func TestTm6_term51_insertLines_cursor_below_the_bottom_margin_does_nothing_and_keeps_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UFill(t, tm, tm6UAlpha6)
	tm.SetTopAndBottomMargin(2, 4)
	tm.SetCursorPos(5, 6)
	tm6Print(t, tm, 'Z')
	tm6UExpectCursor(t, tm, 5, 4, true)
	tm.clearDirty()
	tm6UResetKittyDirty(tm)

	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "stuvwx", "yzABCZ", "EFGHIJ")
	tm6UExpectCursor(t, tm, 5, 4, true)
	tm6UExpectDirty(t, tm, "......")
	expect(t, !tm6UKittyDirty(tm), "!term51UKittyDirty(&t)")
}

// Zig: test "term51: insertLines: cursor left of the left margin does nothing"
func TestTm6_term51_insertLines_cursor_left_of_the_left_margin_does_nothing(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 0, 2)
	tm.scrolling_region.left = 1
	tm.scrolling_region.right = 3
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, tm6UAlpha6...)
	tm6UExpectCursor(t, tm, 0, 2, false)
	tm6UExpectDirty(t, tm, "......")
	expect(t, !tm6UKittyDirty(tm), "!term51UKittyDirty(&t)")
}

// Zig: test "term51: insertLines: cursor right of the right margin does nothing and keeps pending wrap"
func TestTm6_term51_insertLines_cursor_right_of_the_right_margin_does_nothing_and_keeps_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 5, 2)
	tm6Print(t, tm, 'Z')
	tm.scrolling_region.left = 1
	tm.scrolling_region.right = 3
	tm.clearDirty()
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqZ", "stuvwx", "yzABCD", "EFGHIJ")
	tm6UExpectCursor(t, tm, 5, 2, true)
	tm6UExpectDirty(t, tm, "......")
	expect(t, !tm6UKittyDirty(tm), "!term51UKittyDirty(&t)")
}

// Zig: test "term51: insertLines: cursor on the margin columns is inside the region"
func TestTm6_term51_insertLines_cursor_on_the_margin_columns_is_inside_the_region(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	// x == left and x == right are both inside the scroll region.
	tm6UAlphaAt(t, tm, 1, 1)
	tm.scrolling_region.left = 1
	tm.scrolling_region.right = 3
	tm.InsertLines(1)
	tm6UExpectGrid(t, tm, "abcdef", "g...kl", "mhijqr", "snopwx", "ytuvCD", "EzABIJ")
	tm6UExpectCursor(t, tm, 1, 1, false)

	tm6UFill(t, tm, tm6UAlpha6)
	tm.SetCursorPos(2, 4)
	tm.InsertLines(1)
	tm6UExpectGrid(t, tm, "abcdef", "g...kl", "mhijqr", "snopwx", "ytuvCD", "EzABIJ")
	tm6UExpectCursor(t, tm, 1, 1, false)
}

// Zig: test "term51: insertLines: left and right margins shift only the margin columns"
func TestTm6_term51_insertLines_left_and_right_margins_shift_only_the_margin_columns(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 2, 1)
	tm.scrolling_region.left = 1
	tm.scrolling_region.right = 3
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "g...kl", "mhijqr", "snopwx", "ytuvCD", "EzABIJ")
	tm6UExpectCursor(t, tm, 1, 1, false)
	tm6UExpectDirty(t, tm, ".DDDDD")
}

// Zig: test "term51: insertLines: only a left margin shifts the columns from the margin to the edge"
func TestTm6_term51_insertLines_only_a_left_margin_shifts_the_columns_from_the_margin_to_the_edge(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 3, 2)
	tm.scrolling_region.left = 2
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mn....", "stopqr", "yzuvwx", "EFABCD")
	tm6UExpectCursor(t, tm, 2, 2, false)
}

// Zig: test "term51: insertLines: only a right margin shifts the columns from the edge to the margin"
func TestTm6_term51_insertLines_only_a_right_margin_shifts_the_columns_from_the_edge_to_the_margin(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 1, 1)
	tm.scrolling_region.right = 3
	tm.InsertLines(2)

	tm6UExpectGrid(t, tm, "abcdef", "....kl", "....qr", "ghijwx", "mnopCD", "stuvIJ")
	tm6UExpectCursor(t, tm, 0, 1, false)
}

// Zig: test "term51: insertLines: left and right margins with a count past the bottom"
func TestTm6_term51_insertLines_left_and_right_margins_with_a_count_past_the_bottom(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 2, 4)
	tm.scrolling_region.left = 1
	tm.scrolling_region.right = 3
	tm.InsertLines(100)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "stuvwx", "y...CD", "E...IJ")
	tm6UExpectCursor(t, tm, 1, 4, false)
}

// Zig: test "term51: insertLines: margins in both directions with a top and bottom region"
func TestTm6_term51_insertLines_margins_in_both_directions_with_a_top_and_bottom_region(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 2, 2)
	tm.scrolling_region.top = 1
	tm.scrolling_region.bottom = 4
	tm.scrolling_region.left = 2
	tm.scrolling_region.right = 4
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mn...r", "stopqx", "yzuvwD", "EFGHIJ")
	tm6UExpectCursor(t, tm, 2, 2, false)
	tm6UExpectDirty(t, tm, "..DDD.")
}

// Zig: test "term51: insertLines: margins set through the setters behave like direct margins"
func TestTm6_term51_insertLines_margins_set_through_the_setters_behave_like_direct_margins(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UFill(t, tm, tm6UAlpha6)
	tm.modes.Set(ModesModeEnableLeftAndRightMargin, true)
	tm.SetLeftAndRightMargin(2, 4)
	tm.SetTopAndBottomMargin(2, 5)
	// The margin setters home the cursor; origin mode is off.
	tm6UExpectCursor(t, tm, 0, 0, false)
	tm.SetCursorPos(2, 2)
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "g...kl", "mhijqr", "snopwx", "ytuvCD", "EFGHIJ")
	tm6UExpectCursor(t, tm, 1, 1, false)
}

// Zig: test "term51: insertLines: pending wrap is cleared"
func TestTm6_term51_insertLines_pending_wrap_is_cleared(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 5, 1)
	tm6Print(t, tm, 'Z')
	tm6UExpectCursor(t, tm, 5, 1, true)
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "......", "ghijkZ", "mnopqr", "stuvwx", "yzABCD")
	tm6UExpectCursor(t, tm, 0, 1, false)
}

// Zig: test "term51: insertLines: wrap flags of shifted rows are cleared on a full width region"
func TestTm6_term51_insertLines_wrap_flags_of_shifted_rows_are_cleared_on_a_full_width_region(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 0, 1)
	tm6USetWrap(t, tm, "bbbbbb")
	tm.InsertLines(1)

	// Row 0 is above the region and untouched. Rows 1..5 took part in swaps.
	tm6UExpectWrap(t, tm, "b.....")
}

// Zig: test "term51: insertLines: wrap flags are kept when only a left margin is set"
func TestTm6_term51_insertLines_wrap_flags_are_kept_when_only_a_left_margin_is_set(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 2, 1)
	tm.scrolling_region.left = 2
	tm6USetWrap(t, tm, "bbbbbb")
	tm.InsertLines(1)

	tm6UExpectWrap(t, tm, "bbbbbb")
}

// Zig: test "term51: insertLines: wrap flags are kept when only a right margin is set"
func TestTm6_term51_insertLines_wrap_flags_are_kept_when_only_a_right_margin_is_set(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 0, 1)
	tm.scrolling_region.right = 3
	tm6USetWrap(t, tm, "wcwcwc")
	tm.InsertLines(1)

	tm6UExpectWrap(t, tm, "wcwcwc")
}

// Zig: test "term51: insertLines: wrap flags of cleared rows are kept when nothing is shifted"
func TestTm6_term51_insertLines_wrap_flags_of_cleared_rows_are_kept_when_nothing_is_shifted(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	// The count covers every remaining row so no row is swapped; the rows
	// are only cleared and keep their wrap flags.
	tm6UAlphaAt(t, tm, 0, 3)
	tm6USetWrap(t, tm, "wwwbcw")
	tm.InsertLines(3)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "......", "......", "......")
	tm6UExpectWrap(t, tm, "wwwbcw")
}

// Zig: test "term51: insertLines: a single cleared row keeps its wrap flag"
func TestTm6_term51_insertLines_a_single_cleared_row_keeps_its_wrap_flag(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 0, 5)
	tm6USetWrap(t, tm, "bbbbbw")
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "stuvwx", "yzABCD", "......")
	tm6UExpectWrap(t, tm, "bbbbbw")
}

// Zig: test "term51: insertLines: row semantic prompt travels with the row on a full width region"
func TestTm6_term51_insertLines_row_semantic_prompt_travels_with_the_row_on_a_full_width_region(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 0, 0)
	tm6URowAt(t, tm, 2).semantic_prompt = PageRowSemanticPromptPrompt
	tm6URowAt(t, tm, 5).semantic_prompt = PageRowSemanticPromptPromptContinuation
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "......", "abcdef", "ghijkl", "mnopqr", "stuvwx", "yzABCD")
	// The row that moved keeps its prompt; the cleared top row is the old
	// bottom row structure and so still carries the stale prompt kind.
	expectEqual(t, PageRowSemanticPromptPromptContinuation, tm6URowAt(t, tm, 0).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptNone, tm6URowAt(t, tm, 1).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptNone, tm6URowAt(t, tm, 2).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptPrompt, tm6URowAt(t, tm, 3).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptNone, tm6URowAt(t, tm, 4).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptNone, tm6URowAt(t, tm, 5).semantic_prompt)
}

// Zig: test "term51: insertLines: row semantic prompt stays in place with left and right margins"
func TestTm6_term51_insertLines_row_semantic_prompt_stays_in_place_with_left_and_right_margins(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 1, 0)
	tm.scrolling_region.left = 1
	tm.scrolling_region.right = 3
	tm6URowAt(t, tm, 2).semantic_prompt = PageRowSemanticPromptPrompt
	tm6URowAt(t, tm, 5).semantic_prompt = PageRowSemanticPromptPromptContinuation
	tm.InsertLines(1)

	expectEqual(t, PageRowSemanticPromptNone, tm6URowAt(t, tm, 0).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptNone, tm6URowAt(t, tm, 1).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptPrompt, tm6URowAt(t, tm, 2).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptNone, tm6URowAt(t, tm, 3).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptNone, tm6URowAt(t, tm, 4).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptPromptContinuation, tm6URowAt(t, tm, 5).semantic_prompt)
}

// Zig: test "term51: insertLines: protected cells move with their text and blanks are unprotected"
func TestTm6_term51_insertLines_protected_cells_move_with_their_text_and_blanks_are_unprotected(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UFill(t, tm, tm6UAlpha6)
	tm.SetCursorPos(2, 3)
	tm.SetProtectedMode(ANSIProtectedModeDec)
	tm6Print(t, tm, 'X')
	tm.SetProtectedMode(ANSIProtectedModeOff)
	expect(t, tm6UCellAt(t, tm, 2, 1).protected, "term51UCellAt(&t, 2, 1).protected")

	tm.SetCursorPos(2, 1)
	tm.InsertLines(1)

	// Neither mode nor the protected flag prevents insert lines.
	tm6UExpectGrid(t, tm, "abcdef", "......", "ghXjkl", "mnopqr", "stuvwx", "yzABCD")
	expect(t, !tm6UCellAt(t, tm, 2, 1).protected, "!term51UCellAt(&t, 2, 1).protected")
	expect(t, tm6UCellAt(t, tm, 2, 2).protected, "term51UCellAt(&t, 2, 2).protected")
	expect(t, !tm6UCellAt(t, tm, 2, 3).protected, "!term51UCellAt(&t, 2, 3).protected")
}

// Zig: test "term51: insertLines: protected ISO mode does not stop clearing"
func TestTm6_term51_insertLines_protected_ISO_mode_does_not_stop_clearing(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UFill(t, tm, tm6UAlpha6)
	tm.SetCursorPos(6, 1)
	tm.SetProtectedMode(ANSIProtectedModeIso)
	tm6Print(t, tm, 'Y')
	tm.SetCursorPos(6, 1)
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "stuvwx", "yzABCD", "......")
	expect(t, !tm6UCellAt(t, tm, 0, 5).protected, "!term51UCellAt(&t, 0, 5).protected")
}

// Zig: test "term51: insertLines: cleared rows take the background of the cursor style"
func TestTm6_term51_insertLines_cleared_rows_take_the_background_of_the_cursor_style(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 0, 2)
	tm6SetAttribute(t, tm, tm6Attr256Bg(7))
	tm.InsertLines(2)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "pppppp", "pppppp", "mnopqr", "stuvwx")
	for x := uint(0); x < 6; x++ {
		cell := *tm6UCellAt(t, tm, x, 2)
		expectEqual(t, PageCellContentTagBgColorPalette, cell.content_tag)
		expectEqual(t, uint8(7), cell.content.color_palette)
		expectEqual(t, StyleDefaultId, cell.style_id)
	}
}

// Zig: test "term51: insertLines: left and right margins color only the margin columns"
func TestTm6_term51_insertLines_left_and_right_margins_color_only_the_margin_columns(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 1, 1)
	tm.scrolling_region.left = 1
	tm.scrolling_region.right = 3
	tm6SetAttribute(t, tm, tm6AttrDirectBg(1, 2, 3))
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "grrrkl", "mhijqr", "snopwx", "ytuvCD", "EzABIJ")
	cell := *tm6UCellAt(t, tm, 2, 1)
	expectEqual(t, PageCellContentTagBgColorRgb, cell.content_tag)
	expectEqual(t, PageCellRGB{r: 1, g: 2, b: 3}, cell.content.color_rgb)
}

// Zig: test "term51: insertLines: a cursor style without a background leaves blanks empty"
func TestTm6_term51_insertLines_a_cursor_style_without_a_background_leaves_blanks_empty(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 0, 1)
	tm6SetAttribute(t, tm, tm6AttrBold)
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "......", "ghijkl", "mnopqr", "stuvwx", "yzABCD")
	// The blank cells carry no style; only the cursor holds the bold style.
	expectEqual(t, StyleDefaultId, tm6UCellAt(t, tm, 0, 1).style_id)
	expectEqual(t, uint(1), tm6UPage(tm).styles.Count())
	expectEqual(t, uint(1), tm6StyleRefCount(tm6UPage(tm), tm.screens.active.cursor.style_id))
}

// ---- insertLines: wide characters ----

// Zig: test "term51: insertLines: spacer head is turned narrow on a full width region"
func TestTm6_term51_insertLines_spacer_head_is_turned_narrow_on_a_full_width_region(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 8, 4)
	defer tm.Deinit()

	tm6UPutStr(t, tm, 0, 0, "abcdefg")
	tm6Print(t, tm, 0x4E2D)
	tm6UExpectGrid(t, tm, "abcdefg<", "#>......", "........", "........")
	tm6UExpectWrap(t, tm, "wc..")
	tm.SetCursorPos(1, 1)
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "........", "abcdefg.", "#>......", "........")
	expectEqual(t, PageCellWideNarrow, tm6UCellAt(t, tm, 7, 1).wide)
	tm6UExpectWrap(t, tm, "....")
}

// Zig: test "term51: insertLines: spacer head outside the margins is kept when the left margin is two or more"
func TestTm6_term51_insertLines_spacer_head_outside_the_margins_is_kept_when_the_left_margin_is_two_or_more(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 8, 4)
	defer tm.Deinit()

	tm6UPutStr(t, tm, 0, 0, "abcdefg")
	tm6Print(t, tm, 0x4E2D)
	tm.scrolling_region.left = 2
	tm.scrolling_region.right = 5
	tm.SetCursorPos(1, 3)
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "ab....g<", "#>cdef..", "........", "........")
	// Left/right mode keeps the soft wrap bookkeeping of the rows.
	tm6UExpectWrap(t, tm, "wc..")
}

// Zig: test "term51: insertLines: spacer head outside the margins is turned narrow when the left margin is below two"
func TestTm6_term51_insertLines_spacer_head_outside_the_margins_is_turned_narrow_when_the_left_margin_is_below_two(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 8, 4)
	defer tm.Deinit()

	tm6UPutStr(t, tm, 0, 0, "abcdefg")
	tm6Print(t, tm, 0x4E2D)
	tm.scrolling_region.left = 1
	tm.scrolling_region.right = 5
	tm.SetCursorPos(1, 2)
	tm.InsertLines(1)

	// The wide character in column 0/1 straddles the left margin, so it is
	// cleared too.
	tm6UExpectGrid(t, tm, "a.....g.", ".bcdef..", "........", "........")
	expectEqual(t, PageCellWideNarrow, tm6UCellAt(t, tm, 7, 0).wide)
}

// Zig: test "term51: insertLines: left margin zero turns the spacer head narrow and moves a contained wide character"
func TestTm6_term51_insertLines_left_margin_zero_turns_the_spacer_head_narrow_and_moves_a_contained_wide_character(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 8, 4)
	defer tm.Deinit()

	tm6UPutStr(t, tm, 0, 0, "abcdefg")
	tm6Print(t, tm, 0x4E2D)
	tm.scrolling_region.right = 5
	tm.SetCursorPos(1, 1)
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "......g.", "abcdef..", "#>......", "........")
}

// Zig: test "term51: insertLines: a wide character split by the left margin is cleared"
func TestTm6_term51_insertLines_a_wide_character_split_by_the_left_margin_is_cleared(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 8, 4)
	defer tm.Deinit()

	tm6UPutStr(t, tm, 0, 1, "ab")
	tm6UPutWide(t, tm, 2, 1)
	tm6UPutStr(t, tm, 4, 1, "efgh")
	tm6UExpectGrid(t, tm, "........", "ab#>efgh", "........", "........")

	tm.scrolling_region.left = 3
	tm.SetCursorPos(1, 4)
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "........", "ab......", "....efgh", "........")
}

// Zig: test "term51: insertLines: a wide character split by the right margin is cleared"
func TestTm6_term51_insertLines_a_wide_character_split_by_the_right_margin_is_cleared(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 8, 4)
	defer tm.Deinit()

	tm6UPutStr(t, tm, 0, 1, "abcde")
	tm6UPutWide(t, tm, 5, 1)
	tm6UPutStr(t, tm, 7, 1, "h")
	tm6UExpectGrid(t, tm, "........", "abcde#>h", "........", "........")

	tm.scrolling_region.right = 5
	tm.SetCursorPos(1, 1)
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "........", ".......h", "abcde...", "........")
}

// Zig: test "term51: insertLines: a wide character on the right margin is cleared when all rows are cleared"
func TestTm6_term51_insertLines_a_wide_character_on_the_right_margin_is_cleared_when_all_rows_are_cleared(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 8, 4)
	defer tm.Deinit()

	tm6UPutStr(t, tm, 0, 1, "abcde")
	tm6UPutWide(t, tm, 5, 1)
	tm6UPutStr(t, tm, 7, 1, "h")

	tm.scrolling_region.right = 5
	tm.SetCursorPos(2, 1)
	tm.InsertLines(^uint(0))

	tm6UExpectGrid(t, tm, "........", ".......h", "........", "........")
}

// Zig: test "term51: insertLines: a wide character with a grapheme split by the left margin releases the grapheme"
func TestTm6_term51_insertLines_a_wide_character_with_a_grapheme_split_by_the_left_margin_releases_the_grapheme(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 8, 4)
	defer tm.Deinit()

	tm6UPutStr(t, tm, 0, 1, "ab")
	tm6UPutWideGrapheme(t, tm, 2, 1)
	tm6UPutStr(t, tm, 4, 1, "efgh")
	tm6UExpectGrid(t, tm, "........", "ab#>efgh", "........", "........")
	expectEqual(t, uint(1), tm6UPage(tm).GraphemeCount())
	expect(t, tm6URowAt(t, tm, 1).grapheme, "term51URowAt(&t, 1).grapheme")

	tm.scrolling_region.left = 3
	tm.SetCursorPos(1, 4)
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "........", "ab......", "....efgh", "........")
	expectEqual(t, uint(0), tm6UPage(tm).GraphemeCount())
	expect(t, !tm6URowAt(t, tm, 1).grapheme, "!term51URowAt(&t, 1).grapheme")
	expect(t, !tm6URowAt(t, tm, 2).grapheme, "!term51URowAt(&t, 2).grapheme")
}

// Zig: test "term51: insertLines: a wide character with a grapheme split by the right margin releases the grapheme"
func TestTm6_term51_insertLines_a_wide_character_with_a_grapheme_split_by_the_right_margin_releases_the_grapheme(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 8, 4)
	defer tm.Deinit()

	tm6UPutStr(t, tm, 0, 1, "abcde")
	tm6UPutWideGrapheme(t, tm, 5, 1)
	tm6UPutStr(t, tm, 7, 1, "h")
	expectEqual(t, uint(1), tm6UPage(tm).GraphemeCount())

	tm.scrolling_region.right = 5
	tm.SetCursorPos(1, 1)
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "........", ".......h", "abcde...", "........")
	expectEqual(t, uint(0), tm6UPage(tm).GraphemeCount())
}

// Zig: test "term51: insertLines: wide characters inside the margins move whole"
func TestTm6_term51_insertLines_wide_characters_inside_the_margins_move_whole(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 8, 4)
	defer tm.Deinit()

	tm6UPutWide(t, tm, 2, 0)
	tm6UPutWide(t, tm, 4, 0)
	tm.scrolling_region.left = 2
	tm.scrolling_region.right = 5
	tm.SetCursorPos(1, 3)
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "........", "..#>#>..", "........", "........")
}

// Zig: test "term51: insertLines: wide characters on a full width region move whole"
func TestTm6_term51_insertLines_wide_characters_on_a_full_width_region_move_whole(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 8, 4)
	defer tm.Deinit()

	tm6UPutWide(t, tm, 0, 0)
	tm6UPutWide(t, tm, 6, 0)
	tm.SetCursorPos(1, 1)
	tm.InsertLines(2)

	tm6UExpectGrid(t, tm, "........", "........", "#>....#>", "........")
}

// ---- insertLines: styles, hyperlinks, graphemes ----

// Zig: test "term51: insertLines: style reference counts follow the shifted and discarded rows"
func TestTm6_term51_insertLines_style_reference_counts_follow_the_shifted_and_discarded_rows(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 4)
	defer tm.Deinit()

	tm6UPutStr(t, tm, 0, 0, "abcdef")
	tm6SetAttribute(t, tm, tm6AttrBold)
	tm6UPutStr(t, tm, 0, 1, "XXXXXX")
	tm6SetAttribute(t, tm, tm6AttrUnset)
	tm6SetAttribute(t, tm, tm6AttrItalic)
	tm6UPutStr(t, tm, 0, 2, "YYYYYY")
	tm6SetAttribute(t, tm, tm6AttrUnset)
	page := tm6UPage(tm)
	bold := tm6UCellAt(t, tm, 0, 1).style_id
	italic := tm6UCellAt(t, tm, 0, 2).style_id
	expectEqual(t, uint(2), page.styles.Count())
	expectEqual(t, uint(6), tm6StyleRefCount(page, bold))
	expectEqual(t, uint(6), tm6StyleRefCount(page, italic))

	// Nothing falls off the region: both styled rows move down.
	tm.SetCursorPos(2, 1)
	tm.InsertLines(1)
	tm6UExpectGrid(t, tm, "abcdef", "......", "XXXXXX", "YYYYYY")
	expectEqual(t, uint(2), page.styles.Count())
	expectEqual(t, uint(6), tm6StyleRefCount(page, bold))
	expectEqual(t, uint(6), tm6StyleRefCount(page, italic))
	expect(t, !tm6URowAt(t, tm, 1).styled, "!term51URowAt(&t, 1).styled")
	expect(t, tm6URowAt(t, tm, 2).styled, "term51URowAt(&t, 2).styled")
	expect(t, tm6URowAt(t, tm, 3).styled, "term51URowAt(&t, 3).styled")
	expectEqual(t, bold, tm6UCellAt(t, tm, 3, 2).style_id)
	expectEqual(t, italic, tm6UCellAt(t, tm, 3, 3).style_id)

	// The italic row falls off the bottom and its references are released.
	tm.InsertLines(1)
	tm6UExpectGrid(t, tm, "abcdef", "......", "......", "XXXXXX")
	expectEqual(t, uint(1), page.styles.Count())
	expectEqual(t, uint(6), tm6StyleRefCount(page, bold))
	expect(t, !tm6URowAt(t, tm, 1).styled, "!term51URowAt(&t, 1).styled")
	expect(t, !tm6URowAt(t, tm, 2).styled, "!term51URowAt(&t, 2).styled")
	expect(t, tm6URowAt(t, tm, 3).styled, "term51URowAt(&t, 3).styled")

	// Clearing everything left releases the last style.
	tm.InsertLines(^uint(0))
	tm6UExpectGrid(t, tm, "abcdef", "......", "......", "......")
	expectEqual(t, uint(0), page.styles.Count())
	expect(t, !tm6URowAt(t, tm, 3).styled, "!term51URowAt(&t, 3).styled")
}

// Zig: test "term51: insertLines: style reference counts with left and right margins"
func TestTm6_term51_insertLines_style_reference_counts_with_left_and_right_margins(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 4)
	defer tm.Deinit()

	tm6SetAttribute(t, tm, tm6AttrBold)
	tm6UPutStr(t, tm, 0, 1, "XXXXXX")
	tm6SetAttribute(t, tm, tm6AttrUnset)
	tm6SetAttribute(t, tm, tm6AttrItalic)
	tm6UPutStr(t, tm, 0, 2, "YYYYYY")
	tm6SetAttribute(t, tm, tm6AttrUnset)
	page := tm6UPage(tm)
	bold := tm6UCellAt(t, tm, 0, 1).style_id
	italic := tm6UCellAt(t, tm, 0, 2).style_id

	tm.scrolling_region.left = 1
	tm.scrolling_region.right = 3
	tm.SetCursorPos(2, 2)
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "......", "X...XX", "YXXXYY", ".YYY..")
	expectEqual(t, uint(2), page.styles.Count())
	// Three cells of each style stay outside the margins, three moved.
	expectEqual(t, uint(6), tm6StyleRefCount(page, bold))
	expectEqual(t, uint(6), tm6StyleRefCount(page, italic))
	expect(t, tm6URowAt(t, tm, 1).styled, "term51URowAt(&t, 1).styled")
	expect(t, tm6URowAt(t, tm, 2).styled, "term51URowAt(&t, 2).styled")
	expect(t, tm6URowAt(t, tm, 3).styled, "term51URowAt(&t, 3).styled")
	expectEqual(t, StyleDefaultId, tm6UCellAt(t, tm, 1, 1).style_id)
	expectEqual(t, bold, tm6UCellAt(t, tm, 0, 1).style_id)
	expectEqual(t, bold, tm6UCellAt(t, tm, 2, 2).style_id)
	expectEqual(t, italic, tm6UCellAt(t, tm, 0, 2).style_id)
	expectEqual(t, italic, tm6UCellAt(t, tm, 2, 3).style_id)

	// Clearing the region releases the margin cells but not those outside.
	tm.InsertLines(^uint(0))
	tm6UExpectGrid(t, tm, "......", "X...XX", "Y...YY", "......")
	expectEqual(t, uint(2), page.styles.Count())
	expectEqual(t, uint(3), tm6StyleRefCount(page, bold))
	expectEqual(t, uint(3), tm6StyleRefCount(page, italic))
	expect(t, tm6URowAt(t, tm, 1).styled, "term51URowAt(&t, 1).styled")
	expect(t, tm6URowAt(t, tm, 2).styled, "term51URowAt(&t, 2).styled")
	// Quirk: Screen.clearCells refreshes the row's styled flag before it
	// blanks the cells, so a partial-width clear leaves the flag set even
	// though no styled cell remains in the row.
	expect(t, tm6URowAt(t, tm, 3).styled, "term51URowAt(&t, 3).styled")
	expectEqual(t, StyleDefaultId, tm6UCellAt(t, tm, 2, 3).style_id)
}

// Zig: test "term51: insertLines: the cursor style reference is untouched"
func TestTm6_term51_insertLines_the_cursor_style_reference_is_untouched(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 4)
	defer tm.Deinit()

	tm6SetAttribute(t, tm, tm6AttrBold)
	tm6SetAttribute(t, tm, tm6AttrDirectBg(9, 8, 7))
	tm6UPutStr(t, tm, 0, 1, "XXXXXX")
	page := tm6UPage(tm)
	cursor_style := tm.screens.active.cursor.style_id
	expectEqual(t, uint(7), tm6StyleRefCount(page, cursor_style))

	tm.SetCursorPos(2, 1)
	tm.InsertLines(1)
	tm6UExpectGrid(t, tm, "......", "rrrrrr", "XXXXXX", "......")
	expectEqual(t, cursor_style, tm.screens.active.cursor.style_id)
	expectEqual(t, uint(1), page.styles.Count())
	expectEqual(t, uint(7), tm6StyleRefCount(page, cursor_style))

	tm.InsertLines(^uint(0))
	expectEqual(t, cursor_style, tm.screens.active.cursor.style_id)
	expectEqual(t, uint(1), page.styles.Count())
	expectEqual(t, uint(1), tm6StyleRefCount(page, cursor_style))
}

// Zig: test "term51: insertLines: hyperlinks move with their rows and are released when discarded"
func TestTm6_term51_insertLines_hyperlinks_move_with_their_rows_and_are_released_when_discarded(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 4)
	defer tm.Deinit()

	zigTry(t, tm.screens.active.StartHyperlink([]uint8("http://a.test"), None[[]uint8]()))
	tm6UPutStr(t, tm, 0, 1, "XXXXXX")
	zigTry(t, tm.screens.active.StartHyperlink([]uint8("http://b.test"), None[[]uint8]()))
	tm6UPutStr(t, tm, 0, 2, "YYYYYY")
	tm.screens.active.EndHyperlink()
	page := tm6UPage(tm)
	expectEqual(t, uint(12), page.HyperlinkCount())
	expectEqual(t, uint(2), page.hyperlink_set.Count())

	tm.SetCursorPos(2, 1)
	tm.InsertLines(1)
	tm6UExpectGrid(t, tm, "......", "......", "XXXXXX", "YYYYYY")
	expectEqual(t, uint(12), page.HyperlinkCount())
	expectEqual(t, uint(2), page.hyperlink_set.Count())
	expect(t, !tm6URowAt(t, tm, 1).hyperlink, "!term51URowAt(&t, 1).hyperlink")
	expect(t, tm6URowAt(t, tm, 2).hyperlink, "term51URowAt(&t, 2).hyperlink")
	expect(t, tm6URowAt(t, tm, 3).hyperlink, "term51URowAt(&t, 3).hyperlink")
	for x := uint(0); x < 6; x++ {
		expect(t, !tm6UCellAt(t, tm, x, 1).hyperlink, "!term51UCellAt(&t, x, 1).hyperlink")
		expect(t, tm6UCellAt(t, tm, x, 2).hyperlink, "term51UCellAt(&t, x, 2).hyperlink")
		expect(t, tm6UCellAt(t, tm, x, 3).hyperlink, "term51UCellAt(&t, x, 3).hyperlink")
	}
	id_a := zigUnwrap(t, page.LookupHyperlink(tm6UCellAt(t, tm, 0, 2)))
	id_b := zigUnwrap(t, page.LookupHyperlink(tm6UCellAt(t, tm, 5, 3)))
	expect(t, id_a != id_b, "id_a != id_b")
	expectEqualStrings(t, "http://a.test", string(page.hyperlink_set.Get(id_a).uri))
	expectEqualStrings(t, "http://b.test", string(page.hyperlink_set.Get(id_b).uri))
	expectEqual(t, uint(6), uint(page.hyperlink_set.RefCount(id_a)))

	// Push the second link off the bottom.
	tm.InsertLines(1)
	tm6UExpectGrid(t, tm, "......", "......", "......", "XXXXXX")
	expectEqual(t, uint(6), page.HyperlinkCount())
	expectEqual(t, uint(1), page.hyperlink_set.Count())
	expect(t, !tm6URowAt(t, tm, 2).hyperlink, "!term51URowAt(&t, 2).hyperlink")
	expect(t, tm6URowAt(t, tm, 3).hyperlink, "term51URowAt(&t, 3).hyperlink")

	// And clear the last one.
	tm.InsertLines(^uint(0))
	expectEqual(t, uint(0), page.HyperlinkCount())
	expectEqual(t, uint(0), page.hyperlink_set.Count())
	expect(t, !tm6URowAt(t, tm, 3).hyperlink, "!term51URowAt(&t, 3).hyperlink")
}

// Zig: test "term51: insertLines: hyperlinks with left and right margins"
func TestTm6_term51_insertLines_hyperlinks_with_left_and_right_margins(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 4)
	defer tm.Deinit()

	zigTry(t, tm.screens.active.StartHyperlink([]uint8("http://a.test"), None[[]uint8]()))
	tm6UPutStr(t, tm, 0, 1, "XXXXXX")
	tm.screens.active.EndHyperlink()
	page := tm6UPage(tm)

	tm.scrolling_region.left = 1
	tm.scrolling_region.right = 3
	tm.SetCursorPos(2, 2)
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "......", "X...XX", ".XXX..", "......")
	expectEqual(t, uint(6), page.HyperlinkCount())
	expectEqual(t, uint(1), page.hyperlink_set.Count())
	expect(t, tm6URowAt(t, tm, 1).hyperlink, "term51URowAt(&t, 1).hyperlink")
	expect(t, tm6URowAt(t, tm, 2).hyperlink, "term51URowAt(&t, 2).hyperlink")
	expect(t, tm6UCellAt(t, tm, 0, 1).hyperlink, "term51UCellAt(&t, 0, 1).hyperlink")
	expect(t, !tm6UCellAt(t, tm, 1, 1).hyperlink, "!term51UCellAt(&t, 1, 1).hyperlink")
	expect(t, !tm6UCellAt(t, tm, 0, 2).hyperlink, "!term51UCellAt(&t, 0, 2).hyperlink")
	expect(t, tm6UCellAt(t, tm, 1, 2).hyperlink, "term51UCellAt(&t, 1, 2).hyperlink")

	tm.InsertLines(^uint(0))
	tm6UExpectGrid(t, tm, "......", "X...XX", "......", "......")
	expectEqual(t, uint(3), page.HyperlinkCount())
	expect(t, tm6URowAt(t, tm, 1).hyperlink, "term51URowAt(&t, 1).hyperlink")
	expect(t, !tm6URowAt(t, tm, 2).hyperlink, "!term51URowAt(&t, 2).hyperlink")
}

// Zig: test "term51: insertLines: graphemes move with their rows and are released when discarded"
func TestTm6_term51_insertLines_graphemes_move_with_their_rows_and_are_released_when_discarded(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 4)
	defer tm.Deinit()

	tm6UPutGrapheme(t, tm, 1, 1)
	tm6UPutStr(t, tm, 3, 1, "x")
	page := tm6UPage(tm)
	expectEqual(t, uint(1), page.GraphemeCount())
	tm6UExpectGrid(t, tm, "......", ".*.x..", "......", "......")

	tm.SetCursorPos(1, 1)
	tm.InsertLines(1)
	tm6UExpectGrid(t, tm, "......", "......", ".*.x..", "......")
	expectEqual(t, uint(1), page.GraphemeCount())
	expect(t, !tm6URowAt(t, tm, 1).grapheme, "!term51URowAt(&t, 1).grapheme")
	expect(t, tm6URowAt(t, tm, 2).grapheme, "term51URowAt(&t, 2).grapheme")
	expectEqualSlices(t, []uint32{0x0301}, zigUnwrap(t, page.LookupGrapheme(tm6UCellAt(t, tm, 1, 2))))

	tm.InsertLines(1)
	tm6UExpectGrid(t, tm, "......", "......", "......", ".*.x..")
	expectEqual(t, uint(1), page.GraphemeCount())

	tm.InsertLines(1)
	tm6UExpectGrid(t, tm, "......", "......", "......", "......")
	expectEqual(t, uint(0), page.GraphemeCount())
	expect(t, !tm6URowAt(t, tm, 3).grapheme, "!term51URowAt(&t, 3).grapheme")
}

// Zig: test "term51: insertLines: graphemes with left and right margins"
func TestTm6_term51_insertLines_graphemes_with_left_and_right_margins(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 4)
	defer tm.Deinit()

	// One grapheme inside the margins and one outside, on the same row.
	tm6UPutGrapheme(t, tm, 0, 1)
	tm6UPutGrapheme(t, tm, 2, 1)
	page := tm6UPage(tm)
	expectEqual(t, uint(2), page.GraphemeCount())

	tm.scrolling_region.left = 1
	tm.scrolling_region.right = 3
	tm.SetCursorPos(2, 2)
	tm.InsertLines(1)
	tm6UExpectGrid(t, tm, "......", "*.....", "..*...", "......")
	expectEqual(t, uint(2), page.GraphemeCount())
	expect(t, tm6URowAt(t, tm, 1).grapheme, "term51URowAt(&t, 1).grapheme")
	expect(t, tm6URowAt(t, tm, 2).grapheme, "term51URowAt(&t, 2).grapheme")

	// Clearing the margins releases only the grapheme inside them.
	tm.InsertLines(^uint(0))
	tm6UExpectGrid(t, tm, "......", "*.....", "......", "......")
	expectEqual(t, uint(1), page.GraphemeCount())
	expect(t, tm6URowAt(t, tm, 1).grapheme, "term51URowAt(&t, 1).grapheme")
	expect(t, !tm6URowAt(t, tm, 2).grapheme, "!term51URowAt(&t, 2).grapheme")
}

// Zig: test "term51: insertLines: multi-codepoint graphemes survive the shift"
func TestTm6_term51_insertLines_multi_codepoint_graphemes_survive_the_shift(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 4)
	defer tm.Deinit()

	tm6UPutWideGrapheme(t, tm, 0, 1)
	page := tm6UPage(tm)
	expectEqual(t, uint(1), page.GraphemeCount())

	tm.SetCursorPos(1, 1)
	tm.InsertLines(2)
	tm6UExpectGrid(t, tm, "......", "......", "......", "#>....")
	expectEqual(t, uint(1), page.GraphemeCount())
	expectEqualSlices(
		t,
		[]uint32{0x200D, 0x1F469},
		zigUnwrap(t, page.LookupGrapheme(tm6UCellAt(t, tm, 0, 3))),
	)
}

// ---- insertLines: pages ----

// Zig: test "term51: insertLines: rows crossing a page boundary keep styles, hyperlinks and graphemes"
func TestTm6_term51_insertLines_rows_crossing_a_page_boundary_keep_styles_hyperlinks_and_graphemes(t *testing.T) {
	testStdCapacity(t)
	opts := terminalOptionsDefault(10, 5)
	opts.max_scrollback = 1024
	tm := tm6InitOpts(t, opts)
	defer tm.Deinit()

	tm6UCrossPage(t, tm, 2)
	first := tm.screens.active.pages.pages.first
	second := first.next

	tm6UFill(t, tm, []string{"AAAAAAAAAA", "BBBBBBBBBB", "..........", "DDDDDDDDDD", "EEEEEEEEEE"})
	// Row 2 is the last row of the first page and has all three kinds of
	// managed memory.
	tm6SetAttribute(t, tm, tm6AttrBold)
	zigTry(t, tm.screens.active.StartHyperlink([]uint8("http://x.test"), None[[]uint8]()))
	tm6UPutGrapheme(t, tm, 0, 2)
	tm6UPutStr(t, tm, 1, 2, "XXXXXXXXX")
	tm6SetAttribute(t, tm, tm6AttrUnset)
	tm.screens.active.EndHyperlink()
	expectEqual(t, uint(1), first.data.GraphemeCount())
	expectEqual(t, uint(10), first.data.HyperlinkCount())
	expectEqual(t, uint(1), first.data.styles.Count())

	tm.SetCursorPos(1, 1)
	tm.clearDirty()
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "..........", "AAAAAAAAAA", "BBBBBBBBBB", "*XXXXXXXXX", "DDDDDDDDDD")
	tm6UExpectDirty(t, tm, "DDDDD")
	// The pages did not change, and the rows moved over to the second page.
	expect(t, tm.screens.active.pages.pages.first == first, "t.screens.active.pages.pages.first.? == first")
	expect(t, first.next == second, "first.next.? == second")
	expectEqual(t, uint(0), first.data.GraphemeCount())
	expectEqual(t, uint(0), first.data.HyperlinkCount())
	expectEqual(t, uint(0), first.data.styles.Count())
	expectEqual(t, uint(0), first.data.hyperlink_set.Count())
	expect(t, !tm6URowAt(t, tm, 0).grapheme, "!term51URowAt(&t, 0).grapheme")
	expect(t, !tm6URowAt(t, tm, 0).hyperlink, "!term51URowAt(&t, 0).hyperlink")
	expect(t, !tm6URowAt(t, tm, 0).styled, "!term51URowAt(&t, 0).styled")

	page := &second.data
	expectEqual(t, uint(1), page.GraphemeCount())
	expectEqual(t, uint(10), page.HyperlinkCount())
	expectEqual(t, uint(1), page.styles.Count())
	expectEqual(t, uint(1), page.hyperlink_set.Count())
	row := tm6URowAt(t, tm, 3)
	expect(t, row.grapheme && row.hyperlink && row.styled, "row.grapheme and row.hyperlink and row.styled")
	id := zigUnwrap(t, page.LookupHyperlink(tm6UCellAt(t, tm, 5, 3)))
	expectEqualStrings(t, "http://x.test", string(page.hyperlink_set.Get(id).uri))
	expectEqualSlices(t, []uint32{0x0301}, zigUnwrap(t, page.LookupGrapheme(tm6UCellAt(t, tm, 0, 3))))
	expectEqual(t, uint(10), tm6StyleRefCount(page, tm6UCellAt(t, tm, 0, 3).style_id))
}

// Zig: test "term51: insertLines: left and right margins across a page boundary copy only the margin columns"
func TestTm6_term51_insertLines_left_and_right_margins_across_a_page_boundary_copy_only_the_margin_columns(t *testing.T) {
	testStdCapacity(t)
	opts := terminalOptionsDefault(10, 5)
	opts.max_scrollback = 1024
	tm := tm6InitOpts(t, opts)
	defer tm.Deinit()

	tm6UCrossPage(t, tm, 2)
	first := tm.screens.active.pages.pages.first
	second := first.next

	tm6UFill(t, tm, []string{"AAAAAAAAAA", "BBBBBBBBBB", "..........", "DDDDDDDDDD", "EEEEEEEEEE"})
	tm6SetAttribute(t, tm, tm6AttrBold)
	zigTry(t, tm.screens.active.StartHyperlink([]uint8("http://x.test"), None[[]uint8]()))
	tm6UPutGrapheme(t, tm, 0, 2)
	tm6UPutStr(t, tm, 1, 2, "XXXXXXXXX")
	tm6SetAttribute(t, tm, tm6AttrUnset)
	tm.screens.active.EndHyperlink()

	tm.scrolling_region.left = 1
	tm.scrolling_region.right = 8
	tm.SetCursorPos(1, 2)
	tm.InsertLines(1)

	tm6UExpectGrid(t, tm, "A........A", "BAAAAAAAAB", "*BBBBBBBBX", "DXXXXXXXXD", "EDDDDDDDDE")
	// The cells outside the margins stay behind in the first page.
	expectEqual(t, uint(1), first.data.GraphemeCount())
	expectEqual(t, uint(2), first.data.HyperlinkCount())
	expectEqual(t, uint(1), first.data.styles.Count())
	expectEqual(t, uint(2), tm6StyleRefCount(&first.data, tm6UCellAt(t, tm, 0, 2).style_id))
	// The margin columns were copied into the second page.
	expectEqual(t, uint(0), second.data.GraphemeCount())
	expectEqual(t, uint(8), second.data.HyperlinkCount())
	expectEqual(t, uint(1), second.data.styles.Count())
	expectEqual(t, uint(8), tm6StyleRefCount(&second.data, tm6UCellAt(t, tm, 1, 3).style_id))
	expect(t, tm6URowAt(t, tm, 3).hyperlink, "term51URowAt(&t, 3).hyperlink")
	expect(t, tm6URowAt(t, tm, 3).styled, "term51URowAt(&t, 3).styled")
	expect(t, !tm6URowAt(t, tm, 3).grapheme, "!term51URowAt(&t, 3).grapheme")
	expect(t, !tm6UCellAt(t, tm, 0, 3).hyperlink, "!term51UCellAt(&t, 0, 3).hyperlink")
	expect(t, tm6UCellAt(t, tm, 1, 3).hyperlink, "term51UCellAt(&t, 1, 3).hyperlink")
}

// ---------------------------------------------------------------------------
// deleteLines
// ---------------------------------------------------------------------------

// Zig: test "term51: deleteLines: zero count changes nothing and keeps cursor and pending wrap"
func TestTm6_term51_deleteLines_zero_count_changes_nothing_and_keeps_cursor_and_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 5, 2)
	tm6Print(t, tm, 'Z')
	tm6UExpectCursor(t, tm, 5, 2, true)
	tm.clearDirty()

	tm.DeleteLines(0)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqZ", "stuvwx", "yzABCD", "EFGHIJ")
	tm6UExpectCursor(t, tm, 5, 2, true)
	tm6UExpectDirty(t, tm, "......")
	expect(t, !tm6UKittyDirty(tm), "!term51UKittyDirty(&t)")
}

// Zig: test "term51: deleteLines: one line in the middle shifts up and moves cursor to column zero"
func TestTm6_term51_deleteLines_one_line_in_the_middle_shifts_up_and_moves_cursor_to_column_zero(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 3, 2)
	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "stuvwx", "yzABCD", "EFGHIJ", "......")
	tm6UExpectCursor(t, tm, 0, 2, false)
	tm6UExpectDirty(t, tm, "..DDDD")
	// build_options.kitty_graphics is false in the library build.
	expectEqual(t, false, tm6UKittyDirty(tm))
}

// Zig: test "term51: deleteLines: at the top row shifts the whole screen up"
func TestTm6_term51_deleteLines_at_the_top_row_shifts_the_whole_screen_up(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 0, 0)
	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "ghijkl", "mnopqr", "stuvwx", "yzABCD", "EFGHIJ", "......")
	tm6UExpectDirty(t, tm, "DDDDDD")
}

// Zig: test "term51: deleteLines: count one less than the remaining rows keeps one row"
func TestTm6_term51_deleteLines_count_one_less_than_the_remaining_rows_keeps_one_row(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 2, 3)
	tm.DeleteLines(2)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "EFGHIJ", "......", "......")
	tm6UExpectCursor(t, tm, 0, 3, false)
	tm6UExpectDirty(t, tm, "...DDD")
}

// Zig: test "term51: deleteLines: count equal to the remaining rows clears them all"
func TestTm6_term51_deleteLines_count_equal_to_the_remaining_rows_clears_them_all(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 2, 3)
	tm.DeleteLines(3)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "......", "......", "......")
	tm6UExpectCursor(t, tm, 0, 3, false)
	tm6UExpectDirty(t, tm, "...DDD")
}

// Zig: test "term51: deleteLines: count one more than the remaining rows is clamped"
func TestTm6_term51_deleteLines_count_one_more_than_the_remaining_rows_is_clamped(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 2, 3)
	tm.DeleteLines(4)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "......", "......", "......")
	tm6UExpectCursor(t, tm, 0, 3, false)
	tm6UExpectDirty(t, tm, "...DDD")
}

// Zig: test "term51: deleteLines: maximum count is clamped to the remaining rows"
func TestTm6_term51_deleteLines_maximum_count_is_clamped_to_the_remaining_rows(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 2, 3)
	tm.DeleteLines(^uint(0))

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "......", "......", "......")
	tm6UExpectCursor(t, tm, 0, 3, false)
	tm6UExpectDirty(t, tm, "...DDD")
}

// Zig: test "term51: deleteLines: on the last row only that row is cleared"
func TestTm6_term51_deleteLines_on_the_last_row_only_that_row_is_cleared(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 4, 5)
	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "stuvwx", "yzABCD", "......")
	tm6UExpectCursor(t, tm, 0, 5, false)
	tm6UExpectDirty(t, tm, ".....D")
}

// Zig: test "term51: deleteLines: maximum count on the last row clears only that row"
func TestTm6_term51_deleteLines_maximum_count_on_the_last_row_clears_only_that_row(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 4, 5)
	tm.DeleteLines(^uint(0))

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "stuvwx", "yzABCD", "......")
	tm6UExpectDirty(t, tm, ".....D")
}

// Zig: test "term51: deleteLines: top and bottom margins confine the shift"
func TestTm6_term51_deleteLines_top_and_bottom_margins_confine_the_shift(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UFill(t, tm, tm6UAlpha6)
	tm.SetTopAndBottomMargin(2, 4)
	tm.SetCursorPos(3, 5)
	tm.clearDirty()
	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "stuvwx", "......", "yzABCD", "EFGHIJ")
	tm6UExpectCursor(t, tm, 0, 2, false)
	tm6UExpectDirty(t, tm, "..DD..")
}

// Zig: test "term51: deleteLines: cursor on the top margin row with margins"
func TestTm6_term51_deleteLines_cursor_on_the_top_margin_row_with_margins(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UFill(t, tm, tm6UAlpha6)
	tm.SetTopAndBottomMargin(2, 4)
	tm.SetCursorPos(2, 1)
	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "mnopqr", "stuvwx", "......", "yzABCD", "EFGHIJ")
	tm6UExpectCursor(t, tm, 0, 1, false)
}

// Zig: test "term51: deleteLines: cursor on the bottom margin row only clears that row"
func TestTm6_term51_deleteLines_cursor_on_the_bottom_margin_row_only_clears_that_row(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UFill(t, tm, tm6UAlpha6)
	tm.SetTopAndBottomMargin(2, 4)
	tm.SetCursorPos(4, 3)
	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "......", "yzABCD", "EFGHIJ")
	tm6UExpectCursor(t, tm, 0, 3, false)
}

// Zig: test "term51: deleteLines: count larger than the margin region clears the region only"
func TestTm6_term51_deleteLines_count_larger_than_the_margin_region_clears_the_region_only(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UFill(t, tm, tm6UAlpha6)
	tm.SetTopAndBottomMargin(2, 4)
	tm.SetCursorPos(3, 1)
	tm.DeleteLines(100)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "......", "......", "yzABCD", "EFGHIJ")
}

// Zig: test "term51: deleteLines: cursor above the top margin does nothing and keeps pending wrap"
func TestTm6_term51_deleteLines_cursor_above_the_top_margin_does_nothing_and_keeps_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UFill(t, tm, tm6UAlpha6)
	tm.SetTopAndBottomMargin(3, 5)
	tm.SetCursorPos(2, 6)
	tm6Print(t, tm, 'Z')
	tm6UExpectCursor(t, tm, 5, 1, true)
	tm.clearDirty()
	tm6UResetKittyDirty(tm)

	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkZ", "mnopqr", "stuvwx", "yzABCD", "EFGHIJ")
	tm6UExpectCursor(t, tm, 5, 1, true)
	tm6UExpectDirty(t, tm, "......")
	expect(t, !tm6UKittyDirty(tm), "!term51UKittyDirty(&t)")
}

// Zig: test "term51: deleteLines: cursor below the bottom margin does nothing and keeps pending wrap"
func TestTm6_term51_deleteLines_cursor_below_the_bottom_margin_does_nothing_and_keeps_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UFill(t, tm, tm6UAlpha6)
	tm.SetTopAndBottomMargin(2, 4)
	tm.SetCursorPos(5, 6)
	tm6Print(t, tm, 'Z')
	tm6UExpectCursor(t, tm, 5, 4, true)
	tm.clearDirty()
	tm6UResetKittyDirty(tm)

	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "stuvwx", "yzABCZ", "EFGHIJ")
	tm6UExpectCursor(t, tm, 5, 4, true)
	tm6UExpectDirty(t, tm, "......")
	expect(t, !tm6UKittyDirty(tm), "!term51UKittyDirty(&t)")
}

// Zig: test "term51: deleteLines: cursor left of the left margin does nothing"
func TestTm6_term51_deleteLines_cursor_left_of_the_left_margin_does_nothing(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 0, 2)
	tm.scrolling_region.left = 1
	tm.scrolling_region.right = 3
	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, tm6UAlpha6...)
	tm6UExpectCursor(t, tm, 0, 2, false)
	tm6UExpectDirty(t, tm, "......")
	expect(t, !tm6UKittyDirty(tm), "!term51UKittyDirty(&t)")
}

// Zig: test "term51: deleteLines: cursor right of the right margin does nothing and keeps pending wrap"
func TestTm6_term51_deleteLines_cursor_right_of_the_right_margin_does_nothing_and_keeps_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 5, 2)
	tm6Print(t, tm, 'Z')
	tm.scrolling_region.left = 1
	tm.scrolling_region.right = 3
	tm.clearDirty()
	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqZ", "stuvwx", "yzABCD", "EFGHIJ")
	tm6UExpectCursor(t, tm, 5, 2, true)
	tm6UExpectDirty(t, tm, "......")
	expect(t, !tm6UKittyDirty(tm), "!term51UKittyDirty(&t)")
}

// Zig: test "term51: deleteLines: cursor on the margin columns is inside the region"
func TestTm6_term51_deleteLines_cursor_on_the_margin_columns_is_inside_the_region(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 1, 1)
	tm.scrolling_region.left = 1
	tm.scrolling_region.right = 3
	tm.DeleteLines(1)
	tm6UExpectGrid(t, tm, "abcdef", "gnopkl", "mtuvqr", "szABwx", "yFGHCD", "E...IJ")
	tm6UExpectCursor(t, tm, 1, 1, false)

	tm6UFill(t, tm, tm6UAlpha6)
	tm.SetCursorPos(2, 4)
	tm.DeleteLines(1)
	tm6UExpectGrid(t, tm, "abcdef", "gnopkl", "mtuvqr", "szABwx", "yFGHCD", "E...IJ")
	tm6UExpectCursor(t, tm, 1, 1, false)
}

// Zig: test "term51: deleteLines: left and right margins shift only the margin columns"
func TestTm6_term51_deleteLines_left_and_right_margins_shift_only_the_margin_columns(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 2, 1)
	tm.scrolling_region.left = 1
	tm.scrolling_region.right = 3
	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "gnopkl", "mtuvqr", "szABwx", "yFGHCD", "E...IJ")
	tm6UExpectCursor(t, tm, 1, 1, false)
	tm6UExpectDirty(t, tm, ".DDDDD")
}

// Zig: test "term51: deleteLines: only a left margin shifts the columns from the margin to the edge"
func TestTm6_term51_deleteLines_only_a_left_margin_shifts_the_columns_from_the_margin_to_the_edge(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 3, 2)
	tm.scrolling_region.left = 2
	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnuvwx", "stABCD", "yzGHIJ", "EF....")
	tm6UExpectCursor(t, tm, 2, 2, false)
}

// Zig: test "term51: deleteLines: only a right margin shifts the columns from the edge to the margin"
func TestTm6_term51_deleteLines_only_a_right_margin_shifts_the_columns_from_the_edge_to_the_margin(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 1, 1)
	tm.scrolling_region.right = 3
	tm.DeleteLines(2)

	tm6UExpectGrid(t, tm, "abcdef", "stuvkl", "yzABqr", "EFGHwx", "....CD", "....IJ")
	tm6UExpectCursor(t, tm, 0, 1, false)
}

// Zig: test "term51: deleteLines: left and right margins with a count past the bottom"
func TestTm6_term51_deleteLines_left_and_right_margins_with_a_count_past_the_bottom(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 2, 4)
	tm.scrolling_region.left = 1
	tm.scrolling_region.right = 3
	tm.DeleteLines(100)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "stuvwx", "y...CD", "E...IJ")
	tm6UExpectCursor(t, tm, 1, 4, false)
}

// Zig: test "term51: deleteLines: margins in both directions with a top and bottom region"
func TestTm6_term51_deleteLines_margins_in_both_directions_with_a_top_and_bottom_region(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 2, 2)
	tm.scrolling_region.top = 1
	tm.scrolling_region.bottom = 4
	tm.scrolling_region.left = 2
	tm.scrolling_region.right = 4
	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnuvwr", "stABCx", "yz...D", "EFGHIJ")
	tm6UExpectCursor(t, tm, 2, 2, false)
	tm6UExpectDirty(t, tm, "..DDD.")
}

// Zig: test "term51: deleteLines: margins set through the setters behave like direct margins"
func TestTm6_term51_deleteLines_margins_set_through_the_setters_behave_like_direct_margins(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UFill(t, tm, tm6UAlpha6)
	tm.modes.Set(ModesModeEnableLeftAndRightMargin, true)
	tm.SetLeftAndRightMargin(2, 4)
	tm.SetTopAndBottomMargin(2, 5)
	tm6UExpectCursor(t, tm, 0, 0, false)
	tm.SetCursorPos(2, 2)
	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "gnopkl", "mtuvqr", "szABwx", "y...CD", "EFGHIJ")
	tm6UExpectCursor(t, tm, 1, 1, false)
}

// Zig: test "term51: deleteLines: pending wrap is cleared"
func TestTm6_term51_deleteLines_pending_wrap_is_cleared(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 5, 1)
	tm6Print(t, tm, 'Z')
	tm6UExpectCursor(t, tm, 5, 1, true)
	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "mnopqr", "stuvwx", "yzABCD", "EFGHIJ", "......")
	tm6UExpectCursor(t, tm, 0, 1, false)
}

// Zig: test "term51: deleteLines: wrap flags of shifted rows are cleared on a full width region"
func TestTm6_term51_deleteLines_wrap_flags_of_shifted_rows_are_cleared_on_a_full_width_region(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 0, 1)
	tm6USetWrap(t, tm, "bbbbbb")
	tm.DeleteLines(1)

	tm6UExpectWrap(t, tm, "b.....")
}

// Zig: test "term51: deleteLines: wrap flags are kept when only a left margin is set"
func TestTm6_term51_deleteLines_wrap_flags_are_kept_when_only_a_left_margin_is_set(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 2, 1)
	tm.scrolling_region.left = 2
	tm6USetWrap(t, tm, "bbbbbb")
	tm.DeleteLines(1)

	tm6UExpectWrap(t, tm, "bbbbbb")
}

// Zig: test "term51: deleteLines: wrap flags are kept when only a right margin is set"
func TestTm6_term51_deleteLines_wrap_flags_are_kept_when_only_a_right_margin_is_set(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 0, 1)
	tm.scrolling_region.right = 3
	tm6USetWrap(t, tm, "wcwcwc")
	tm.DeleteLines(1)

	tm6UExpectWrap(t, tm, "wcwcwc")
}

// Zig: test "term51: deleteLines: wrap flags of cleared rows are kept when nothing is shifted"
func TestTm6_term51_deleteLines_wrap_flags_of_cleared_rows_are_kept_when_nothing_is_shifted(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 0, 3)
	tm6USetWrap(t, tm, "wwwbcw")
	tm.DeleteLines(3)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "......", "......", "......")
	tm6UExpectWrap(t, tm, "wwwbcw")
}

// Zig: test "term51: deleteLines: a single cleared row keeps its wrap flag"
func TestTm6_term51_deleteLines_a_single_cleared_row_keeps_its_wrap_flag(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 0, 5)
	tm6USetWrap(t, tm, "bbbbbw")
	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "stuvwx", "yzABCD", "......")
	tm6UExpectWrap(t, tm, "bbbbbw")
}

// Zig: test "term51: deleteLines: row semantic prompt travels with the row on a full width region"
func TestTm6_term51_deleteLines_row_semantic_prompt_travels_with_the_row_on_a_full_width_region(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 0, 0)
	tm6URowAt(t, tm, 0).semantic_prompt = PageRowSemanticPromptPromptContinuation
	tm6URowAt(t, tm, 3).semantic_prompt = PageRowSemanticPromptPrompt
	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "ghijkl", "mnopqr", "stuvwx", "yzABCD", "EFGHIJ", "......")
	expectEqual(t, PageRowSemanticPromptNone, tm6URowAt(t, tm, 0).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptNone, tm6URowAt(t, tm, 1).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptPrompt, tm6URowAt(t, tm, 2).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptNone, tm6URowAt(t, tm, 3).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptNone, tm6URowAt(t, tm, 4).semantic_prompt)
	// The cleared bottom row is the old top row structure and keeps its
	// stale prompt kind.
	expectEqual(t, PageRowSemanticPromptPromptContinuation, tm6URowAt(t, tm, 5).semantic_prompt)
}

// Zig: test "term51: deleteLines: row semantic prompt stays in place with left and right margins"
func TestTm6_term51_deleteLines_row_semantic_prompt_stays_in_place_with_left_and_right_margins(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 1, 0)
	tm.scrolling_region.left = 1
	tm.scrolling_region.right = 3
	tm6URowAt(t, tm, 0).semantic_prompt = PageRowSemanticPromptPromptContinuation
	tm6URowAt(t, tm, 3).semantic_prompt = PageRowSemanticPromptPrompt
	tm.DeleteLines(1)

	expectEqual(t, PageRowSemanticPromptPromptContinuation, tm6URowAt(t, tm, 0).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptNone, tm6URowAt(t, tm, 1).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptNone, tm6URowAt(t, tm, 2).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptPrompt, tm6URowAt(t, tm, 3).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptNone, tm6URowAt(t, tm, 4).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptNone, tm6URowAt(t, tm, 5).semantic_prompt)
}

// Zig: test "term51: deleteLines: protected cells move with their text and blanks are unprotected"
func TestTm6_term51_deleteLines_protected_cells_move_with_their_text_and_blanks_are_unprotected(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UFill(t, tm, tm6UAlpha6)
	tm.SetCursorPos(3, 3)
	tm.SetProtectedMode(ANSIProtectedModeDec)
	tm6Print(t, tm, 'X')
	tm.SetProtectedMode(ANSIProtectedModeOff)
	expect(t, tm6UCellAt(t, tm, 2, 2).protected, "term51UCellAt(&t, 2, 2).protected")

	tm.SetCursorPos(2, 1)
	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "mnXpqr", "stuvwx", "yzABCD", "EFGHIJ", "......")
	expect(t, tm6UCellAt(t, tm, 2, 1).protected, "term51UCellAt(&t, 2, 1).protected")
	expect(t, !tm6UCellAt(t, tm, 2, 2).protected, "!term51UCellAt(&t, 2, 2).protected")
	expect(t, !tm6UCellAt(t, tm, 2, 5).protected, "!term51UCellAt(&t, 2, 5).protected")

	// A protected cell on the deleted row is deleted as well.
	tm.SetCursorPos(2, 1)
	tm.DeleteLines(1)
	tm6UExpectGrid(t, tm, "abcdef", "stuvwx", "yzABCD", "EFGHIJ", "......", "......")
	expect(t, !tm6UCellAt(t, tm, 2, 1).protected, "!term51UCellAt(&t, 2, 1).protected")
}

// Zig: test "term51: deleteLines: protected ISO mode does not stop clearing"
func TestTm6_term51_deleteLines_protected_ISO_mode_does_not_stop_clearing(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UFill(t, tm, tm6UAlpha6)
	tm.SetCursorPos(6, 1)
	tm.SetProtectedMode(ANSIProtectedModeIso)
	tm6Print(t, tm, 'Y')
	tm.SetCursorPos(6, 1)
	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "mnopqr", "stuvwx", "yzABCD", "......")
	expect(t, !tm6UCellAt(t, tm, 0, 5).protected, "!term51UCellAt(&t, 0, 5).protected")
}

// Zig: test "term51: deleteLines: cleared rows take the background of the cursor style"
func TestTm6_term51_deleteLines_cleared_rows_take_the_background_of_the_cursor_style(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 0, 2)
	tm6SetAttribute(t, tm, tm6Attr256Bg(7))
	tm.DeleteLines(2)

	tm6UExpectGrid(t, tm, "abcdef", "ghijkl", "yzABCD", "EFGHIJ", "pppppp", "pppppp")
	for x := uint(0); x < 6; x++ {
		cell := *tm6UCellAt(t, tm, x, 4)
		expectEqual(t, PageCellContentTagBgColorPalette, cell.content_tag)
		expectEqual(t, uint8(7), cell.content.color_palette)
		expectEqual(t, StyleDefaultId, cell.style_id)
	}
}

// Zig: test "term51: deleteLines: left and right margins color only the margin columns"
func TestTm6_term51_deleteLines_left_and_right_margins_color_only_the_margin_columns(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 1, 1)
	tm.scrolling_region.left = 1
	tm.scrolling_region.right = 3
	tm6SetAttribute(t, tm, tm6AttrDirectBg(1, 2, 3))
	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "gnopkl", "mtuvqr", "szABwx", "yFGHCD", "ErrrIJ")
	cell := *tm6UCellAt(t, tm, 2, 5)
	expectEqual(t, PageCellContentTagBgColorRgb, cell.content_tag)
	expectEqual(t, PageCellRGB{r: 1, g: 2, b: 3}, cell.content.color_rgb)
}

// Zig: test "term51: deleteLines: a cursor style without a background leaves blanks empty"
func TestTm6_term51_deleteLines_a_cursor_style_without_a_background_leaves_blanks_empty(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 6, 6)
	defer tm.Deinit()

	tm6UAlphaAt(t, tm, 0, 1)
	tm6SetAttribute(t, tm, tm6AttrBold)
	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "abcdef", "mnopqr", "stuvwx", "yzABCD", "EFGHIJ", "......")
	expectEqual(t, StyleDefaultId, tm6UCellAt(t, tm, 0, 5).style_id)
	expectEqual(t, uint(1), tm6UPage(tm).styles.Count())
}

// Zig: test "term51: deleteLines: spacer head is turned narrow on a full width region"
func TestTm6_term51_deleteLines_spacer_head_is_turned_narrow_on_a_full_width_region(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 8, 4)
	defer tm.Deinit()

	// The first row is a wrapped row ending in a spacer head; deleting the
	// row above it moves it up.
	tm6UPutStr(t, tm, 0, 1, "abcdefg")
	tm6Print(t, tm, 0x4E2D)
	tm6UExpectGrid(t, tm, "........", "abcdefg<", "#>......", "........")

	tm.SetCursorPos(1, 1)
	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "abcdefg.", "#>......", "........", "........")
	expectEqual(t, PageCellWideNarrow, tm6UCellAt(t, tm, 7, 0).wide)
	tm6UExpectWrap(t, tm, "....")
}

// Zig: test "term51: deleteLines: spacer head outside the margins is kept when the left margin is two or more"
func TestTm6_term51_deleteLines_spacer_head_outside_the_margins_is_kept_when_the_left_margin_is_two_or_more(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 8, 4)
	defer tm.Deinit()

	tm6UPutStr(t, tm, 0, 1, "abcdefg")
	tm6Print(t, tm, 0x4E2D)
	tm.scrolling_region.left = 2
	tm.scrolling_region.right = 5
	tm.SetCursorPos(1, 3)
	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "..cdef..", "ab....g<", "#>......", "........")
	tm6UExpectWrap(t, tm, ".wc.")
}

// Zig: test "term51: deleteLines: left margin zero turns the spacer head narrow"
func TestTm6_term51_deleteLines_left_margin_zero_turns_the_spacer_head_narrow(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 8, 4)
	defer tm.Deinit()

	tm6UPutStr(t, tm, 0, 1, "abcdefg")
	tm6Print(t, tm, 0x4E2D)
	tm.scrolling_region.right = 5
	tm.SetCursorPos(1, 1)
	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "abcdef..", "#>....g.", "........", "........")
}

// Zig: test "term51: deleteLines: a wide character split by the left margin is cleared"
func TestTm6_term51_deleteLines_a_wide_character_split_by_the_left_margin_is_cleared(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 8, 4)
	defer tm.Deinit()

	tm6UPutStr(t, tm, 0, 1, "ab")
	tm6UPutWide(t, tm, 2, 1)
	tm6UPutStr(t, tm, 4, 1, "efgh")

	tm.scrolling_region.left = 3
	tm.SetCursorPos(1, 4)
	tm.DeleteLines(1)

	tm6UExpectGrid(t, tm, "....efgh", "ab......", "........", "........")
}

// ---------------------------------------------------------------------------
// Go-only panic tests
// ---------------------------------------------------------------------------

// Go-only: panic test: rowWillBeShifted: a spacer tail in the left margin column 0 (left - 1 overflows)
func TestTm6_panic_rowWillBeShifted_spacer_tail_at_left_margin_zero(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 10, 2)
	defer tm.Deinit()

	tm6TPutRow(t, tm, 0, "abcdefghij")
	tm6UCellAt(t, tm, 0, 0).wide = PageCellWideSpacerTail
	tm6TRegion(tm, 0, 1, 0, 9)
	lc := zigUnwrap(t, tm.screens.active.pages.GetCell(tm6Active(0, 0)))
	tm6ExpectPanic(t, func() {
		tm.rowWillBeShifted(&lc.node.data, lc.row)
	})
}

// Go-only: panic test: insertLines: scrolling region bottom beyond the active area (pin.down(rem - 1).? is null)
func TestTm6_panic_insertLines_region_bottom_beyond_the_active_area(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 5, 5)
	defer tm.Deinit()

	tm.scrolling_region.bottom = 9
	tm6ExpectPanic(t, func() {
		tm.InsertLines(1)
	})
}

// Go-only: panic test: deleteLines: scrolling region bottom beyond the active area (cur_p.down(adjusted_count).? is null)
func TestTm6_panic_deleteLines_region_bottom_beyond_the_active_area(t *testing.T) {
	testStdCapacity(t)
	tm := tm6Init(t, 5, 5)
	defer tm.Deinit()

	tm.scrolling_region.bottom = 9
	tm6ExpectPanic(t, func() {
		tm.DeleteLines(1)
	})
}
