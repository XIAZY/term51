// Tests translated one for one from src/terminal/Terminal.zig (Ghostty
// v1.3.1 plus the term51 additional tests), the tests between
// `test "term51: deleteLines: a wide character split by the right margin is
// cleared"` and `test "term51: eraseChars: style reference counts follow the
// erased cells"` (Terminal.zig lines 22152 to 24497). Each Go test carries the
// exact Zig title in a "// Zig: test" comment. Helpers are prefixed tm7.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualStrings / expectEqualSlices keep Zig's names
//     and Zig's (expected, actual) argument order. `try` on a call that
//     returns an error is zigTry(t, err); `x.?` is zigUnwrap or a nil check
//     (tm7Node for a page list node).
//   - The Zig terminal `t` is `term` (the Go `t` is the *testing.T). The
//     Zig helpers term51U* are tm7*; they take (t, term, ...). Positions
//     are uint, as Zig's usize.
//   - `.{ .cols = c, .rows = r }` is terminalOptionsDefault(c, r) (Zig
//     defaults written out) and `.max_scrollback = n` is set on the result.
//   - `@as(usize, n)` is the Go type of the other side: uint for counts,
//     SizeCellCountInt for style reference counts.
//   - `std.math.maxInt(usize)` is math.MaxUint.
//   - `testing.allocator` is dropped (`// Zig-only:`); every test starts with
//     testStdCapacity because several depend on the page capacity.
//   - Zig's "terminal returned by value" helpers (term51UEightAt,
//     term51UCrossTerm) return *Terminal.
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt (none).

package terminal

import (
	"errors"
	"math"
	"strings"
	"testing"
)

// tm7Init is `try init(alloc, opts)`; the terminal is returned by pointer
// (the methods take a pointer receiver and the helpers keep it).
func tm7Init(t *testing.T, opts TerminalOptions) *Terminal {
	t.Helper()
	term, err := TerminalInit(opts)
	zigTry(t, err)
	return &term
}

// tm7Node is `n.?` for a page list node.
func tm7Node(t *testing.T, n *PageListNode) *PageListNode {
	t.Helper()
	if n == nil {
		t.Fatalf("unwrapped a null node")
	}
	return n
}

// tm7Alpha6 is term51UAlpha6: six rows of six distinct letters, used as the
// standard fixture.
var tm7Alpha6 = [6]string{
	"abcdef",
	"ghijkl",
	"mnopqr",
	"stuvwx",
	"yzABCD",
	"EFGHIJ",
}

// tm7Fill is term51UFill: print every non-'.' byte of every row at its own
// position. Leaves the cursor at the top-left with no pending wrap.
func tm7Fill(t *testing.T, term *Terminal, rows []string) {
	t.Helper()
	for y, row := range rows {
		for x := 0; x < len(row); x++ {
			c := row[x]
			if c == '.' {
				continue
			}
			term.SetCursorPos(uint(y)+1, uint(x)+1)
			zigTry(t, term.Print(uint32(c)))
		}
	}
	term.SetCursorPos(1, 1)
}

// tm7CellAt is term51UCellAt.
func tm7CellAt(t *testing.T, term *Terminal, x uint, y uint) *PageCell {
	t.Helper()
	return zigUnwrap(t, term.Screens.Active.Pages.GetCell(Point{
		Tag:    PointTagActive,
		Active: PointCoordinate{X: SizeCellCountInt(x), Y: uint32(y)},
	})).Cell
}

// tm7RowAt is term51URowAt.
func tm7RowAt(t *testing.T, term *Terminal, y uint) *PageRow {
	t.Helper()
	return zigUnwrap(t, term.Screens.Active.Pages.GetCell(Point{
		Tag:    PointTagActive,
		Active: PointCoordinate{X: 0, Y: uint32(y)},
	})).Row
}

// tm7CellChar is term51UCellChar. One character per cell: '.' empty, the
// ASCII character for narrow text, '*' narrow grapheme cluster, '#' wide
// head, '>' spacer tail, '<' spacer head, 'p' palette background cell, 'r'
// RGB background cell.
func tm7CellChar(cell PageCell) uint8 {
	switch cell.Wide() {
	case PageCellWideWide:
		return '#'
	case PageCellWideSpacerTail:
		return '>'
	case PageCellWideSpacerHead:
		return '<'
	case PageCellWideNarrow:
		switch cell.ContentTag() {
		case PageCellContentTagCodepoint:
			if cell.content().codepoint() == 0 {
				return '.'
			} else if cell.content().codepoint() < 128 {
				return uint8(cell.content().codepoint())
			}
			return '?'
		case PageCellContentTagCodepointGrapheme:
			return '*'
		case PageCellContentTagBgColorPalette:
			return 'p'
		case PageCellContentTagBgColorRgb:
			return 'r'
		}
	}
	panic("unreachable")
}

// tm7ExpectGrid is term51UExpectGrid: compare the whole active area against
// the expected rows.
func tm7ExpectGrid(t *testing.T, term *Terminal, expected []string) {
	t.Helper()
	expectEqual(t, uint(term.Rows), uint(len(expected)))
	var buf [64]uint8
	for y, exp := range expected {
		for x := uint(0); x < uint(term.Cols); x++ {
			buf[x] = tm7CellChar(*tm7CellAt(t, term, x, uint(y)))
		}
		expectEqualStrings(t, exp, string(buf[0:term.Cols]))
	}
}

// tm7ExpectDirty is term51UExpectDirty: compare the dirty flag of every
// active row: 'D' dirty, '.' clean.
func tm7ExpectDirty(t *testing.T, term *Terminal, expected string) {
	t.Helper()
	expectEqual(t, uint(term.Rows), uint(len(expected)))
	var buf [64]uint8
	for y := uint(0); y < uint(term.Rows); y++ {
		if tm7RowAt(t, term, y).dirty {
			buf[y] = 'D'
		} else {
			buf[y] = '.'
		}
	}
	expectEqualStrings(t, expected, string(buf[0:term.Rows]))
}

// tm7ExpectWrap is term51UExpectWrap: compare the wrap (w),
// wrap_continuation (c), both (b) or neither (.) of every active row.
func tm7ExpectWrap(t *testing.T, term *Terminal, expected string) {
	t.Helper()
	expectEqual(t, uint(term.Rows), uint(len(expected)))
	var buf [64]uint8
	for y := uint(0); y < uint(term.Rows); y++ {
		row := tm7RowAt(t, term, y)
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
	expectEqualStrings(t, expected, string(buf[0:term.Rows]))
}

// tm7SetWrap is term51USetWrap: set every row to the "wrap" state given by
// the characters of `flags`.
func tm7SetWrap(t *testing.T, term *Terminal, flags string) {
	t.Helper()
	for y := 0; y < len(flags); y++ {
		f := flags[y]
		row := tm7RowAt(t, term, uint(y))
		row.wrap = f == 'w' || f == 'b'
		row.wrap_continuation = f == 'c' || f == 'b'
	}
}

// tm7ExpectCursor is term51UExpectCursor.
func tm7ExpectCursor(t *testing.T, term *Terminal, x uint, y uint, pending_wrap bool) {
	t.Helper()
	expectEqual(t, x, uint(term.Screens.Active.Cursor.X))
	expectEqual(t, y, uint(term.Screens.Active.Cursor.Y))
	expectEqual(t, pending_wrap, term.Screens.Active.Cursor.PendingWrap)
}

// tm7Page is term51UPage: the page that holds the cursor; fine for terminals
// that fit one page.
func tm7Page(term *Terminal) *Page {
	return &term.Screens.Active.Cursor.PagePin.Node.data
}

// tm7PutStr is term51UPutStr: print `str` starting at the 0-indexed position
// (x, y).
func tm7PutStr(t *testing.T, term *Terminal, x uint, y uint, str string) {
	t.Helper()
	term.SetCursorPos(y+1, x+1)
	zigTry(t, term.PrintString([]uint8(str)))
}

// tm7PutWide is term51UPutWide: print one wide character (two cells) with
// its head at (x, y).
func tm7PutWide(t *testing.T, term *Terminal, x uint, y uint) {
	t.Helper()
	term.SetCursorPos(y+1, x+1)
	zigTry(t, term.Print(0x4E2D))
}

// tm7PutWideGrapheme is term51UPutWideGrapheme: print a family emoji ZWJ
// sequence, which is one wide cell with a grapheme.
func tm7PutWideGrapheme(t *testing.T, term *Terminal, x uint, y uint) {
	t.Helper()
	term.Modes.Set(ModesModeGraphemeCluster, true)
	term.SetCursorPos(y+1, x+1)
	zigTry(t, term.Print(0x1F468))
	zigTry(t, term.Print(0x200D))
	zigTry(t, term.Print(0x1F469))
}

// tm7PutGrapheme is term51UPutGrapheme: print an 'e' followed by a combining
// acute accent at (x, y): one narrow cell with a grapheme.
func tm7PutGrapheme(t *testing.T, term *Terminal, x uint, y uint) {
	t.Helper()
	term.SetCursorPos(y+1, x+1)
	zigTry(t, term.Print('e'))
	// The grapheme is attached directly: a zero-width character printed
	// after a character on the last column would attach to the cell before.
	zigTry(t, tm7Page(term).SetGraphemes(
		tm7RowAt(t, term, y),
		tm7CellAt(t, term, x, y),
		[]uint32{0x0301},
	))
}

// tm7CrossPage is term51UCrossPage: linefeed so that the active area is
// split over two pages, with `tail_rows` active rows on the second page. The
// cursor ends up at the top-left of the screen. The terminal must have 5
// rows.
func tm7CrossPage(t *testing.T, term *Terminal, tail_rows uint) {
	t.Helper()
	first := tm7Node(t, term.Screens.Active.Pages.pages.first)
	n := uint(first.data.capacity.rows)
	for i := uint(0); i < n+tail_rows-1; i++ {
		zigTry(t, term.Linefeed())
	}
	term.SetCursorPos(1, 1)
	expect(t, first.next != nil, "first.next != null")
	for y := uint(0); y < uint(term.Rows); y++ {
		node := zigUnwrap(t, term.Screens.Active.Pages.Pin(Point{
			Tag:    PointTagActive,
			Active: PointCoordinate{Y: uint32(y)},
		})).Node
		if y+tail_rows < uint(term.Rows) {
			expect(t, node == first, "node == first")
		} else {
			expect(t, node == first.next, "node == first.next.?")
		}
	}
}

// tm7ExpectRowChars is term51UExpectRowChars: assert that cells x in
// [from, to) of row y all show the character `ch`.
func tm7ExpectRowChars(t *testing.T, term *Terminal, y uint, from uint, to uint, ch uint8) {
	t.Helper()
	for x := from; x < to; x++ {
		expectEqual(t, ch, tm7CellChar(*tm7CellAt(t, term, x, y)))
	}
}

// tm7PrintDistinctStyles is term51UPrintDistinctStyles: print `count` single
// cells, each with its own distinct foreground color (so each takes one entry
// of the page style set), at row-major positions starting at (0, y_first)
// with the given width, then restore the default style.
func tm7PrintDistinctStyles(t *testing.T, term *Terminal, y_first uint, cols uint, count uint) {
	t.Helper()
	for i := uint(0); i < count; i++ {
		term.SetCursorPos(y_first+i/cols+1, i%cols+1)
		zigTry(t, term.SetAttribute(SGRAttribute{
			Tag: SGRAttributeTagDirectColorFg,
			DirectColorFg: ColorRGB{
				R: uint8(i),
				G: uint8(i >> 8),
				B: 0,
			},
		}))
		zigTry(t, term.Print('S'))
	}
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))
}

// tm7Cross is term51UCross: the geometry of the page-crossing scenarios. For
// insertLines the active rows 1..4 are on the second page and the row that
// crosses is moved from row 0 to row 1. For deleteLines rows 0..3 are on the
// first page and the row that crosses is moved from row 4 to row 3. In both
// cases rows `fill_y` and `fill_y + 1` are used to fill up the destination
// page.
type tm7Cross struct {
	is_insert bool
}

func (self tm7Cross) tail() uint {
	if self.is_insert {
		return 4
	}
	return 1
}

func (self tm7Cross) srcY() uint {
	if self.is_insert {
		return 0
	}
	return 4
}

func (self tm7Cross) dstY() uint {
	if self.is_insert {
		return 1
	}
	return 3
}

func (self tm7Cross) fillY() uint {
	if self.is_insert {
		return 2
	}
	return 1
}

func (self tm7Cross) dstPage(term *Terminal) *Page {
	var node *PageListNode
	if self.is_insert {
		node = term.Screens.Active.Pages.pages.last
	} else {
		node = term.Screens.Active.Pages.pages.first
	}
	return &node.data
}

func (self tm7Cross) run(term *Terminal) {
	term.SetCursorPos(1, 1)
	if self.is_insert {
		term.InsertLines(1)
	} else {
		term.DeleteLines(1)
	}
}

// tm7CrossTerm is term51UCrossTerm; the terminal is returned by pointer.
func tm7CrossTerm(t *testing.T, cross tm7Cross) *Terminal {
	t.Helper()
	opts := terminalOptionsDefault(60, 5)
	opts.MaxScrollback = 1024
	term := tm7Init(t, opts)
	// Zig-only: errdefer t.deinit(alloc);
	tm7CrossPage(t, term, cross.tail())
	return term
}

// tm7CrossGrapheme is term51UCrossGrapheme: the destination page runs out of
// grapheme map entries (`full_map`) or of grapheme bytes (otherwise); the
// crossing row has one grapheme.
func tm7CrossGrapheme(t *testing.T, cross tm7Cross, full_map bool) {
	t.Helper()
	// Zig-only: const alloc = testing.allocator;
	term := tm7CrossTerm(t, cross)
	defer term.Deinit()

	fill_y := cross.fillY()
	old := cross.dstPage(term).capacity.grapheme_bytes
	var placed uint = 0
	if full_map {
		dst := cross.dstPage(term)
		broke := false
	outer:
		for dy := uint(0); dy < 2; dy++ {
			for x := uint(0); x < 60; x++ {
				tm7PutStr(t, term, x, fill_y+dy, "e")
				row := tm7RowAt(t, term, fill_y+dy)
				cell := tm7CellAt(t, term, x, fill_y+dy)
				err := dst.SetGraphemes(row, cell, []uint32{0x301})
				if err != nil {
					expect(t, errors.Is(err, ErrGraphemeMapOutOfMemory), "err == error.GraphemeMapOutOfMemory")
					broke = true
					break outer
				}
				placed += 1
			}
		}
		// Zig: the `else` of the `for` returns error.TestUnexpectedResult.
		if !broke {
			t.Fatalf("TestUnexpectedResult: the grapheme map never filled up")
		}
	} else {
		dst := cross.dstPage(term)
		// The allocator rounds 512 bytes up to a whole bitmap: 64 chunks of
		// four codepoints. One grapheme with 62 chunks and two with one chunk
		// use all of them.
		expectEqual(t, uint64(64*16), dst.grapheme_alloc.CapacityBytes())
		big := make([]uint32, 62*4)
		for i := range big {
			big[i] = 0x301
		}
		tm7PutStr(t, term, 0, fill_y, "eee")
		row := tm7RowAt(t, term, fill_y)
		zigTry(t, dst.SetGraphemes(row, tm7CellAt(t, term, 0, fill_y), big))
		zigTry(t, dst.SetGraphemes(row, tm7CellAt(t, term, 1, fill_y), []uint32{0x301}))
		zigTry(t, dst.SetGraphemes(row, tm7CellAt(t, term, 2, fill_y), []uint32{0x301}))
		tm7PutStr(t, term, 3, fill_y, "e")
		err := dst.SetGraphemes(row, tm7CellAt(t, term, 3, fill_y), []uint32{0x301})
		expect(t, errors.Is(err, ErrGraphemeAllocOutOfMemory), "dst.setGraphemes(...) == error.GraphemeAllocOutOfMemory")
		placed = 3
	}

	tm7PutGrapheme(t, term, 0, cross.srcY())
	cross.run(term)

	dst := cross.dstPage(term)
	// Doubling 512 bytes still rounds to a single 64 chunk bitmap, so the
	// retry fails again and the capacity is doubled a second time.
	var factor uint = 4
	if full_map {
		factor = 2
	}
	expectEqual(t, uint(old)*factor, uint(dst.capacity.grapheme_bytes))
	expectEqual(t, placed+1, dst.GraphemeCount())
	cell := tm7CellAt(t, term, 0, cross.dstY())
	expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
	expectEqualSlices(t, []uint32{0x301}, zigUnwrap(t, dst.LookupGrapheme(cell)))
	expect(t, tm7RowAt(t, term, cross.dstY()).grapheme, "term51URowAt(&t, cross.dstY()).grapheme")
	tm7ExpectCursor(t, term, 0, 0, false)
}

// tm7HyperlinkFull is term51UHyperlinkFull.
type tm7HyperlinkFull uint8

const (
	tm7HyperlinkFullMap tm7HyperlinkFull = iota
	tm7HyperlinkFullString
	tm7HyperlinkFullSet
	tm7HyperlinkFullRehash
)

// tm7CrossHyperlink is term51UCrossHyperlink: the destination page is full
// in the given way; the crossing row has a hyperlink that is not in the
// destination page.
func tm7CrossHyperlink(t *testing.T, cross tm7Cross, full tm7HyperlinkFull) {
	t.Helper()
	// Zig-only: const alloc = testing.allocator;
	term := tm7CrossTerm(t, cross)
	defer term.Deinit()

	fill_y := cross.fillY()
	dst := cross.dstPage(term)
	old := dst.capacity
	tm7PutStr(t, term, 0, fill_y, strings.Repeat("L", 60))
	tm7PutStr(t, term, 0, fill_y+1, strings.Repeat("L", 60))
	switch full {
	case tm7HyperlinkFullMap:
		// One link on every cell until the map has no more room.
		id, err := dst.InsertHyperlink(Hyperlink{
			ID:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: 1},
			URI: []uint8("http://d.test"),
		})
		zigTry(t, err)
		var n uint = 0
		for dst.HyperlinkCount() < dst.HyperlinkCapacity() {
			expect(t, n < 120, "n < 120")
			y := fill_y + n/60
			if n > 0 {
				dst.hyperlink_set.Use(id)
			}
			zigTry(t, dst.SetHyperlink(tm7RowAt(t, term, y), tm7CellAt(t, term, n%60, y), id))
			n += 1
		}
		expect(t, n > 0, "n > 0")
	case tm7HyperlinkFullString:
		// One link that owns all 64 string chunks: 63 for the URI and one
		// for the explicit id.
		uri := strings.Repeat("u", 63*32)
		id, err := dst.InsertHyperlink(Hyperlink{
			ID:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdExplicit, Explicit: []uint8("i")},
			URI: []uint8(uri),
		})
		zigTry(t, err)
		zigTry(t, dst.SetHyperlink(tm7RowAt(t, term, fill_y), tm7CellAt(t, term, 0, fill_y), id))
	case tm7HyperlinkFullSet, tm7HyperlinkFullRehash:
		id1, err := dst.InsertHyperlink(Hyperlink{
			ID:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: 1},
			URI: []uint8("http://d1.test"),
		})
		zigTry(t, err)
		zigTry(t, dst.SetHyperlink(tm7RowAt(t, term, fill_y), tm7CellAt(t, term, 0, fill_y), id1))
		id2, err := dst.InsertHyperlink(Hyperlink{
			ID:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: 2},
			URI: []uint8("http://d2.test"),
		})
		zigTry(t, err)
		zigTry(t, dst.SetHyperlink(tm7RowAt(t, term, fill_y+1), tm7CellAt(t, term, 0, fill_y+1), id2))
		expectEqual(t, uint(2), dst.hyperlink_set.Count())
		if full == tm7HyperlinkFullRehash {
			// The first link dies, but its id stays allocated.
			dst.ClearHyperlink(tm7CellAt(t, term, 0, fill_y))
			dst.UpdateRowHyperlinkFlag(tm7RowAt(t, term, fill_y))
			expectEqual(t, uint(1), dst.hyperlink_set.Count())
		}
	}
	placed := dst.HyperlinkCount()

	// The source row has a different hyperlink.
	term.SetCursorPos(cross.srcY()+1, 1)
	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://s.test"), None[[]uint8]()))
	zigTry(t, term.Print('X'))
	term.Screens.Active.EndHyperlink()

	cross.run(term)

	after := cross.dstPage(term)
	switch full {
	case tm7HyperlinkFullMap:
		expectEqual(t, uint(old.hyperlink_bytes)*2, uint(after.capacity.hyperlink_bytes))
	case tm7HyperlinkFullString:
		expectEqual(t, uint(old.string_bytes)*2, uint(after.capacity.string_bytes))
	case tm7HyperlinkFullSet:
		expectEqual(t, uint(old.hyperlink_bytes)*2, uint(after.capacity.hyperlink_bytes))
	case tm7HyperlinkFullRehash:
		// A rehash keeps the capacity.
		expectEqual(t, uint(old.hyperlink_bytes), uint(after.capacity.hyperlink_bytes))
		expectEqual(t, uint(old.string_bytes), uint(after.capacity.string_bytes))
	}
	expectEqual(t, placed+1, after.HyperlinkCount())
	cell := tm7CellAt(t, term, 0, cross.dstY())
	expect(t, cell.Hyperlink(), "cell.hyperlink")
	expect(t, tm7RowAt(t, term, cross.dstY()).hyperlink, "term51URowAt(&t, cross.dstY()).hyperlink")
	id := zigUnwrap(t, after.LookupHyperlink(cell))
	expectEqualStrings(t, "http://s.test", string(after.hyperlink_set.Get(id).uri))
	tm7ExpectCursor(t, term, 0, 0, false)
}

// tm7Alpha8 is term51UAlpha8.
var tm7Alpha8 = [2]string{"abcdefgh", "ijklmnop"}

// tm7EightAt is term51UEightAt: a two row, eight column terminal holding
// tm7Alpha8, with the cursor at (x, y), dirty flags cleared. Returned by
// pointer.
func tm7EightAt(t *testing.T, x uint, y uint) *Terminal {
	t.Helper()
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	// Zig-only: errdefer t.deinit(alloc);
	tm7Fill(t, term, tm7Alpha8[:])
	term.SetCursorPos(y+1, x+1)
	term.clearDirty()
	return term
}

// tm7WideRow is term51UWideRow: row "ab#>cdef" with the wide character at
// columns 2 and 3.
func tm7WideRow(t *testing.T, term *Terminal) {
	t.Helper()
	tm7PutStr(t, term, 0, 0, "ab")
	tm7PutWide(t, term, 2, 0)
	tm7PutStr(t, term, 4, 0, "cdef")
	tm7ExpectGrid(t, term, []string{"ab#>cdef", "........"})
	term.SetCursorPos(1, 1)
}

// tm7Protect is term51UProtect: reprint the letters of columns [from, to) of
// the first row of tm7Alpha8 with protection enabled in `mode`, then switch
// the protection off again (the screen remembers the mode).
func tm7Protect(t *testing.T, term *Terminal, mode ANSIProtectedMode, from uint, to uint) {
	t.Helper()
	term.SetProtectedMode(mode)
	for x := from; x < to; x++ {
		term.SetCursorPos(1, x+1)
		zigTry(t, term.Print(uint32(tm7Alpha8[0][x])))
	}
	term.SetProtectedMode(ANSIProtectedModeOff)
}

// Zig: test "term51: deleteLines: a wide character split by the right margin is cleared"
func TestTm7_term51_deleteLines_a_wide_character_split_by_the_right_margin_is_cleared(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 4))
	defer term.Deinit()

	tm7PutStr(t, term, 0, 1, "abcde")
	tm7PutWide(t, term, 5, 1)
	tm7PutStr(t, term, 7, 1, "h")

	term.ScrollingRegion.Right = 5
	term.SetCursorPos(1, 1)
	term.DeleteLines(1)

	tm7ExpectGrid(t, term, []string{"abcde...", ".......h", "........", "........"})
}

// Zig: test "term51: deleteLines: a wide character on the right margin is cleared when all rows are cleared"
func TestTm7_term51_deleteLines_a_wide_character_on_the_right_margin_is_cleared_when_all_rows_are_cleared(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 4))
	defer term.Deinit()

	tm7PutStr(t, term, 0, 1, "abcde")
	tm7PutWide(t, term, 5, 1)
	tm7PutStr(t, term, 7, 1, "h")

	term.ScrollingRegion.Right = 5
	term.SetCursorPos(2, 1)
	term.DeleteLines(math.MaxUint)

	tm7ExpectGrid(t, term, []string{"........", ".......h", "........", "........"})
}

// Zig: test "term51: deleteLines: a wide character with a grapheme split by the left margin releases the grapheme"
func TestTm7_term51_deleteLines_a_wide_character_with_a_grapheme_split_by_the_left_margin_releases_the_grapheme(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 4))
	defer term.Deinit()

	tm7PutStr(t, term, 0, 1, "ab")
	tm7PutWideGrapheme(t, term, 2, 1)
	tm7PutStr(t, term, 4, 1, "efgh")
	expectEqual(t, uint(1), tm7Page(term).GraphemeCount())

	term.ScrollingRegion.Left = 3
	term.SetCursorPos(1, 4)
	term.DeleteLines(1)

	tm7ExpectGrid(t, term, []string{"....efgh", "ab......", "........", "........"})
	expectEqual(t, uint(0), tm7Page(term).GraphemeCount())
	expect(t, !tm7RowAt(t, term, 0).grapheme, "!term51URowAt(&t, 0).grapheme")
	expect(t, !tm7RowAt(t, term, 1).grapheme, "!term51URowAt(&t, 1).grapheme")
}

// Zig: test "term51: deleteLines: a wide character with a grapheme split by the right margin releases the grapheme"
func TestTm7_term51_deleteLines_a_wide_character_with_a_grapheme_split_by_the_right_margin_releases_the_grapheme(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 4))
	defer term.Deinit()

	tm7PutStr(t, term, 0, 1, "abcde")
	tm7PutWideGrapheme(t, term, 5, 1)
	tm7PutStr(t, term, 7, 1, "h")
	expectEqual(t, uint(1), tm7Page(term).GraphemeCount())

	term.ScrollingRegion.Right = 5
	term.SetCursorPos(1, 1)
	term.DeleteLines(1)

	tm7ExpectGrid(t, term, []string{"abcde...", ".......h", "........", "........"})
	expectEqual(t, uint(0), tm7Page(term).GraphemeCount())
}

// Zig: test "term51: deleteLines: wide characters inside the margins move whole"
func TestTm7_term51_deleteLines_wide_characters_inside_the_margins_move_whole(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 4))
	defer term.Deinit()

	tm7PutWide(t, term, 2, 1)
	tm7PutWide(t, term, 4, 1)
	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 5
	term.SetCursorPos(1, 3)
	term.DeleteLines(1)

	tm7ExpectGrid(t, term, []string{"..#>#>..", "........", "........", "........"})
}

// Zig: test "term51: deleteLines: wide characters on a full width region move whole"
func TestTm7_term51_deleteLines_wide_characters_on_a_full_width_region_move_whole(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 4))
	defer term.Deinit()

	tm7PutWide(t, term, 0, 2)
	tm7PutWide(t, term, 6, 2)
	term.SetCursorPos(1, 1)
	term.DeleteLines(2)

	tm7ExpectGrid(t, term, []string{"#>....#>", "........", "........", "........"})
}

// Zig: test "term51: deleteLines: style reference counts follow the shifted and discarded rows"
func TestTm7_term51_deleteLines_style_reference_counts_follow_the_shifted_and_discarded_rows(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(6, 4))
	defer term.Deinit()

	tm7PutStr(t, term, 0, 0, "abcdef")
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	tm7PutStr(t, term, 0, 1, "XXXXXX")
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagItalic}))
	tm7PutStr(t, term, 0, 2, "YYYYYY")
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))
	page := tm7Page(term)
	bold := tm7CellAt(t, term, 0, 1).StyleID()
	italic := tm7CellAt(t, term, 0, 2).StyleID()
	expectEqual(t, uint(2), page.styles.Count())

	// Deleting the plain top row moves both styled rows up.
	term.SetCursorPos(1, 1)
	term.DeleteLines(1)
	tm7ExpectGrid(t, term, []string{"XXXXXX", "YYYYYY", "......", "......"})
	expectEqual(t, uint(2), page.styles.Count())
	expectEqual(t, SizeCellCountInt(6), page.styles.RefCount(bold))
	expectEqual(t, SizeCellCountInt(6), page.styles.RefCount(italic))
	expect(t, tm7RowAt(t, term, 0).styled, "term51URowAt(&t, 0).styled")
	expect(t, tm7RowAt(t, term, 1).styled, "term51URowAt(&t, 1).styled")
	expect(t, !tm7RowAt(t, term, 2).styled, "!term51URowAt(&t, 2).styled")
	expect(t, !tm7RowAt(t, term, 3).styled, "!term51URowAt(&t, 3).styled")

	// Deleting the bold row releases its references.
	term.DeleteLines(1)
	tm7ExpectGrid(t, term, []string{"YYYYYY", "......", "......", "......"})
	expectEqual(t, uint(1), page.styles.Count())
	expectEqual(t, SizeCellCountInt(6), page.styles.RefCount(italic))
	expect(t, tm7RowAt(t, term, 0).styled, "term51URowAt(&t, 0).styled")
	expect(t, !tm7RowAt(t, term, 3).styled, "!term51URowAt(&t, 3).styled")

	term.DeleteLines(math.MaxUint)
	tm7ExpectGrid(t, term, []string{"......", "......", "......", "......"})
	expectEqual(t, uint(0), page.styles.Count())
	expect(t, !tm7RowAt(t, term, 0).styled, "!term51URowAt(&t, 0).styled")
}

// Zig: test "term51: deleteLines: style reference counts with left and right margins"
func TestTm7_term51_deleteLines_style_reference_counts_with_left_and_right_margins(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(6, 4))
	defer term.Deinit()

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	tm7PutStr(t, term, 0, 1, "XXXXXX")
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagItalic}))
	tm7PutStr(t, term, 0, 2, "YYYYYY")
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))
	page := tm7Page(term)
	bold := tm7CellAt(t, term, 0, 1).StyleID()
	italic := tm7CellAt(t, term, 0, 2).StyleID()

	term.ScrollingRegion.Left = 1
	term.ScrollingRegion.Right = 3
	term.SetCursorPos(2, 2)
	term.DeleteLines(1)

	tm7ExpectGrid(t, term, []string{"......", "XYYYXX", "Y...YY", "......"})
	expectEqual(t, uint(2), page.styles.Count())
	// The three margin cells of the bold row were deleted.
	expectEqual(t, SizeCellCountInt(3), page.styles.RefCount(bold))
	expectEqual(t, SizeCellCountInt(6), page.styles.RefCount(italic))
	expect(t, tm7RowAt(t, term, 1).styled, "term51URowAt(&t, 1).styled")
	expect(t, tm7RowAt(t, term, 2).styled, "term51URowAt(&t, 2).styled")
	expect(t, !tm7RowAt(t, term, 0).styled, "!term51URowAt(&t, 0).styled")
}

// Zig: test "term51: deleteLines: hyperlinks move with their rows and are released when discarded"
func TestTm7_term51_deleteLines_hyperlinks_move_with_their_rows_and_are_released_when_discarded(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(6, 4))
	defer term.Deinit()

	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://a.test"), None[[]uint8]()))
	tm7PutStr(t, term, 0, 1, "XXXXXX")
	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://b.test"), None[[]uint8]()))
	tm7PutStr(t, term, 0, 2, "YYYYYY")
	term.Screens.Active.EndHyperlink()
	page := tm7Page(term)
	expectEqual(t, uint(12), page.HyperlinkCount())

	term.SetCursorPos(1, 1)
	term.DeleteLines(1)
	tm7ExpectGrid(t, term, []string{"XXXXXX", "YYYYYY", "......", "......"})
	expectEqual(t, uint(12), page.HyperlinkCount())
	expectEqual(t, uint(2), page.hyperlink_set.Count())
	expect(t, tm7RowAt(t, term, 0).hyperlink, "term51URowAt(&t, 0).hyperlink")
	expect(t, tm7RowAt(t, term, 1).hyperlink, "term51URowAt(&t, 1).hyperlink")
	expect(t, !tm7RowAt(t, term, 2).hyperlink, "!term51URowAt(&t, 2).hyperlink")
	expect(t, !tm7RowAt(t, term, 3).hyperlink, "!term51URowAt(&t, 3).hyperlink")
	id_a := zigUnwrap(t, page.LookupHyperlink(tm7CellAt(t, term, 0, 0)))
	expectEqualStrings(t, "http://a.test", string(page.hyperlink_set.Get(id_a).uri))

	term.DeleteLines(1)
	tm7ExpectGrid(t, term, []string{"YYYYYY", "......", "......", "......"})
	expectEqual(t, uint(6), page.HyperlinkCount())
	expectEqual(t, uint(1), page.hyperlink_set.Count())

	term.DeleteLines(math.MaxUint)
	expectEqual(t, uint(0), page.HyperlinkCount())
	expectEqual(t, uint(0), page.hyperlink_set.Count())
	expect(t, !tm7RowAt(t, term, 0).hyperlink, "!term51URowAt(&t, 0).hyperlink")
}

// Zig: test "term51: deleteLines: hyperlinks with left and right margins"
func TestTm7_term51_deleteLines_hyperlinks_with_left_and_right_margins(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(6, 4))
	defer term.Deinit()

	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://a.test"), None[[]uint8]()))
	tm7PutStr(t, term, 0, 1, "XXXXXX")
	term.Screens.Active.EndHyperlink()
	page := tm7Page(term)

	term.ScrollingRegion.Left = 1
	term.ScrollingRegion.Right = 3
	term.SetCursorPos(1, 2)
	term.DeleteLines(1)

	tm7ExpectGrid(t, term, []string{".XXX..", "X...XX", "......", "......"})
	expectEqual(t, uint(6), page.HyperlinkCount())
	expectEqual(t, uint(1), page.hyperlink_set.Count())
	expect(t, tm7RowAt(t, term, 0).hyperlink, "term51URowAt(&t, 0).hyperlink")
	expect(t, tm7RowAt(t, term, 1).hyperlink, "term51URowAt(&t, 1).hyperlink")
	expect(t, !tm7CellAt(t, term, 0, 0).Hyperlink(), "!term51UCellAt(&t, 0, 0).hyperlink")
	expect(t, tm7CellAt(t, term, 1, 0).Hyperlink(), "term51UCellAt(&t, 1, 0).hyperlink")
	expect(t, tm7CellAt(t, term, 0, 1).Hyperlink(), "term51UCellAt(&t, 0, 1).hyperlink")
	expect(t, !tm7CellAt(t, term, 1, 1).Hyperlink(), "!term51UCellAt(&t, 1, 1).hyperlink")

	term.SetCursorPos(1, 2)
	term.DeleteLines(math.MaxUint)
	tm7ExpectGrid(t, term, []string{"......", "X...XX", "......", "......"})
	expectEqual(t, uint(3), page.HyperlinkCount())
	expect(t, !tm7RowAt(t, term, 0).hyperlink, "!term51URowAt(&t, 0).hyperlink")
	expect(t, tm7RowAt(t, term, 1).hyperlink, "term51URowAt(&t, 1).hyperlink")
}

// Zig: test "term51: deleteLines: graphemes move with their rows and are released when discarded"
func TestTm7_term51_deleteLines_graphemes_move_with_their_rows_and_are_released_when_discarded(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(6, 4))
	defer term.Deinit()

	tm7PutGrapheme(t, term, 1, 2)
	tm7PutStr(t, term, 3, 2, "x")
	page := tm7Page(term)
	expectEqual(t, uint(1), page.GraphemeCount())

	term.SetCursorPos(1, 1)
	term.DeleteLines(1)
	tm7ExpectGrid(t, term, []string{"......", ".*.x..", "......", "......"})
	expectEqual(t, uint(1), page.GraphemeCount())
	expect(t, !tm7RowAt(t, term, 0).grapheme, "!term51URowAt(&t, 0).grapheme")
	expect(t, tm7RowAt(t, term, 1).grapheme, "term51URowAt(&t, 1).grapheme")
	expectEqualSlices(t, []uint32{0x0301}, zigUnwrap(t, page.LookupGrapheme(tm7CellAt(t, term, 1, 1))))

	term.DeleteLines(1)
	tm7ExpectGrid(t, term, []string{".*.x..", "......", "......", "......"})
	expectEqual(t, uint(1), page.GraphemeCount())

	term.DeleteLines(1)
	tm7ExpectGrid(t, term, []string{"......", "......", "......", "......"})
	expectEqual(t, uint(0), page.GraphemeCount())
	expect(t, !tm7RowAt(t, term, 0).grapheme, "!term51URowAt(&t, 0).grapheme")
}

// Zig: test "term51: deleteLines: graphemes with left and right margins"
func TestTm7_term51_deleteLines_graphemes_with_left_and_right_margins(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(6, 4))
	defer term.Deinit()

	tm7PutGrapheme(t, term, 0, 1)
	tm7PutGrapheme(t, term, 2, 1)
	page := tm7Page(term)

	term.ScrollingRegion.Left = 1
	term.ScrollingRegion.Right = 3
	term.SetCursorPos(1, 2)
	term.DeleteLines(1)
	tm7ExpectGrid(t, term, []string{"..*...", "*.....", "......", "......"})
	expectEqual(t, uint(2), page.GraphemeCount())
	expect(t, tm7RowAt(t, term, 0).grapheme, "term51URowAt(&t, 0).grapheme")
	expect(t, tm7RowAt(t, term, 1).grapheme, "term51URowAt(&t, 1).grapheme")

	term.SetCursorPos(1, 2)
	term.DeleteLines(math.MaxUint)
	tm7ExpectGrid(t, term, []string{"......", "*.....", "......", "......"})
	expectEqual(t, uint(1), page.GraphemeCount())
	expect(t, !tm7RowAt(t, term, 0).grapheme, "!term51URowAt(&t, 0).grapheme")
	expect(t, tm7RowAt(t, term, 1).grapheme, "term51URowAt(&t, 1).grapheme")
}

// Zig: test "term51: deleteLines: multi-codepoint graphemes survive the shift"
func TestTm7_term51_deleteLines_multi_codepoint_graphemes_survive_the_shift(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(6, 4))
	defer term.Deinit()

	tm7PutWideGrapheme(t, term, 0, 3)
	page := tm7Page(term)

	term.SetCursorPos(1, 1)
	term.DeleteLines(2)
	tm7ExpectGrid(t, term, []string{"......", "#>....", "......", "......"})
	expectEqual(t, uint(1), page.GraphemeCount())
	expectEqualSlices(t, []uint32{0x200D, 0x1F469}, zigUnwrap(t, page.LookupGrapheme(tm7CellAt(t, term, 0, 1))))
}

// Zig: test "term51: deleteLines: rows crossing a page boundary keep styles, hyperlinks and graphemes"
func TestTm7_term51_deleteLines_rows_crossing_a_page_boundary_keep_styles_hyperlinks_and_graphemes(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	opts := terminalOptionsDefault(10, 5)
	opts.MaxScrollback = 1024
	term := tm7Init(t, opts)
	defer term.Deinit()

	tm7CrossPage(t, term, 2)
	first := tm7Node(t, term.Screens.Active.Pages.pages.first)
	second := tm7Node(t, first.next)

	tm7Fill(t, term, []string{"AAAAAAAAAA", "BBBBBBBBBB", "CCCCCCCCCC", "..........", "EEEEEEEEEE"})
	// Row 3 is the first row of the second page and has all three kinds of
	// managed memory.
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://x.test"), None[[]uint8]()))
	tm7PutGrapheme(t, term, 0, 3)
	tm7PutStr(t, term, 1, 3, "XXXXXXXXX")
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))
	term.Screens.Active.EndHyperlink()
	expectEqual(t, uint(1), second.data.GraphemeCount())
	expectEqual(t, uint(10), second.data.HyperlinkCount())

	term.SetCursorPos(3, 1)
	term.clearDirty()
	term.DeleteLines(1)

	// Row 3 moved up into the first page: the clone crosses the boundary.
	tm7ExpectGrid(t, term, []string{"AAAAAAAAAA", "BBBBBBBBBB", "*XXXXXXXXX", "EEEEEEEEEE", ".........."})
	tm7ExpectDirty(t, term, "..DDD")
	expect(t, term.Screens.Active.Pages.pages.first == first, "t.screens.active.pages.pages.first.? == first")
	expect(t, first.next == second, "first.next.? == second")
	expectEqual(t, uint(1), first.data.GraphemeCount())
	expectEqual(t, uint(10), first.data.HyperlinkCount())
	expectEqual(t, uint(1), first.data.styles.Count())
	expectEqual(t, uint(1), first.data.hyperlink_set.Count())
	expectEqual(t, uint(0), second.data.GraphemeCount())
	expectEqual(t, uint(0), second.data.HyperlinkCount())
	expectEqual(t, uint(0), second.data.styles.Count())
	expectEqual(t, uint(0), second.data.hyperlink_set.Count())
	row := tm7RowAt(t, term, 2)
	expect(t, row.grapheme && row.hyperlink && row.styled, "row.grapheme and row.hyperlink and row.styled")
	id := zigUnwrap(t, first.data.LookupHyperlink(tm7CellAt(t, term, 5, 2)))
	expectEqualStrings(t, "http://x.test", string(first.data.hyperlink_set.Get(id).uri))
	expectEqualSlices(t, []uint32{0x0301}, zigUnwrap(t, first.data.LookupGrapheme(tm7CellAt(t, term, 0, 2))))
}

// Zig: test "term51: deleteLines: left and right margins across a page boundary copy only the margin columns"
func TestTm7_term51_deleteLines_left_and_right_margins_across_a_page_boundary_copy_only_the_margin_columns(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	opts := terminalOptionsDefault(10, 5)
	opts.MaxScrollback = 1024
	term := tm7Init(t, opts)
	defer term.Deinit()

	tm7CrossPage(t, term, 2)
	first := tm7Node(t, term.Screens.Active.Pages.pages.first)
	second := tm7Node(t, first.next)

	tm7Fill(t, term, []string{"AAAAAAAAAA", "BBBBBBBBBB", "CCCCCCCCCC", "..........", "EEEEEEEEEE"})
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://x.test"), None[[]uint8]()))
	tm7PutGrapheme(t, term, 0, 3)
	tm7PutStr(t, term, 1, 3, "XXXXXXXXX")
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))
	term.Screens.Active.EndHyperlink()

	term.ScrollingRegion.Left = 1
	term.ScrollingRegion.Right = 8
	term.SetCursorPos(3, 2)
	term.DeleteLines(1)

	tm7ExpectGrid(t, term, []string{"AAAAAAAAAA", "BBBBBBBBBB", "CXXXXXXXXC", "*EEEEEEEEX", "E........E"})
	// The cells outside the margins stay behind in the second page.
	expectEqual(t, uint(1), second.data.GraphemeCount())
	expectEqual(t, uint(2), second.data.HyperlinkCount())
	expectEqual(t, uint(1), second.data.styles.Count())
	expectEqual(t, SizeCellCountInt(2), second.data.styles.RefCount(tm7CellAt(t, term, 0, 3).StyleID()))
	// The margin columns were copied into the first page.
	expectEqual(t, uint(0), first.data.GraphemeCount())
	expectEqual(t, uint(8), first.data.HyperlinkCount())
	expectEqual(t, uint(1), first.data.styles.Count())
	expectEqual(t, SizeCellCountInt(8), first.data.styles.RefCount(tm7CellAt(t, term, 1, 2).StyleID()))
	expect(t, tm7RowAt(t, term, 2).hyperlink, "term51URowAt(&t, 2).hyperlink")
	expect(t, tm7RowAt(t, term, 2).styled, "term51URowAt(&t, 2).styled")
	expect(t, !tm7RowAt(t, term, 2).grapheme, "!term51URowAt(&t, 2).grapheme")
}

// Zig: test "term51: insertLines: a full style set in the destination page is grown when a row crosses pages"
func TestTm7_term51_insertLines_a_full_style_set_in_the_destination_page_is_grown_when_a_row_crosses_pages(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	opts := terminalOptionsDefault(60, 5)
	opts.MaxScrollback = 1024
	term := tm7Init(t, opts)
	defer term.Deinit()

	// Row 0 is the last row of the first page, rows 1-4 are on the second.
	tm7CrossPage(t, term, 4)
	first := tm7Node(t, term.Screens.Active.Pages.pages.first)
	second := tm7Node(t, first.next)
	old_capacity := second.data.capacity.styles
	need := second.data.styles.layout.cap - 1
	expect(t, need > 60 && need <= 120, "need > 60 and need <= 120")

	tm7PutStr(t, term, 0, 1, strings.Repeat("1", 60))
	tm7PutStr(t, term, 0, 4, strings.Repeat("4", 60))
	tm7PrintDistinctStyles(t, term, 2, 60, need)
	expectEqual(t, need, second.data.styles.Count())

	// The bold style of row 0 is not in the second page, which is full.
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	tm7PutStr(t, term, 0, 0, strings.Repeat("Z", 60))
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))
	expectEqual(t, uint(1), first.data.styles.Count())

	term.SetCursorPos(1, 1)
	term.InsertLines(1)

	tm7ExpectRowChars(t, term, 0, 0, 60, '.')
	tm7ExpectRowChars(t, term, 1, 0, 60, 'Z')
	tm7ExpectRowChars(t, term, 2, 0, 60, '1')
	tm7ExpectRowChars(t, term, 3, 0, 60, 'S')
	tm7ExpectRowChars(t, term, 4, 0, need-60, 'S')
	tm7ExpectRowChars(t, term, 4, need-60, 60, '.')
	tm7ExpectCursor(t, term, 0, 0, false)

	// Only the destination page grew; the first page lost its bold style.
	expect(t, term.Screens.Active.Pages.pages.first == first, "t.screens.active.pages.pages.first.? == first")
	grown := tm7Node(t, term.Screens.Active.Pages.pages.last)
	expect(t, first.next == grown, "first.next.? == grown")
	expectEqual(t, uint(old_capacity)*2, uint(grown.data.capacity.styles))
	expectEqual(t, uint(0), first.data.styles.Count())
	expectEqual(t, need+1, grown.data.styles.Count())
	bold := tm7CellAt(t, term, 0, 1).StyleID()
	expectEqual(t, SizeCellCountInt(60), grown.data.styles.RefCount(bold))
	expect(t, tm7RowAt(t, term, 1).styled, "term51URowAt(&t, 1).styled")
}

// Zig: test "term51: insertLines: a style set that needs a rehash in the destination page is rehashed when a row crosses pages"
func TestTm7_term51_insertLines_a_style_set_that_needs_a_rehash_in_the_destination_page_is_rehashed_when_a_row_crosses_pages(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	opts := terminalOptionsDefault(60, 5)
	opts.MaxScrollback = 1024
	term := tm7Init(t, opts)
	defer term.Deinit()

	tm7CrossPage(t, term, 4)
	first := tm7Node(t, term.Screens.Active.Pages.pages.first)
	second := tm7Node(t, first.next)
	old_capacity := second.data.capacity.styles
	need := second.data.styles.layout.cap - 1
	expect(t, need > 60 && need <= 120, "need > 60 and need <= 120")

	tm7PrintDistinctStyles(t, term, 2, 60, need)
	// Overwrite the first 50 styled cells with plain text: those styles die
	// but their ids stay allocated, so the set is full of dead items.
	tm7PutStr(t, term, 0, 2, strings.Repeat("p", 50))
	expectEqual(t, need-50, second.data.styles.Count())
	expectEqual(t, second.data.styles.layout.cap, uint(second.data.styles.next_id))

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	tm7PutStr(t, term, 0, 0, strings.Repeat("Z", 60))
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))

	term.SetCursorPos(1, 1)
	term.InsertLines(1)

	tm7ExpectRowChars(t, term, 0, 0, 60, '.')
	tm7ExpectRowChars(t, term, 1, 0, 60, 'Z')
	tm7ExpectRowChars(t, term, 2, 0, 60, '.')
	tm7ExpectRowChars(t, term, 3, 0, 50, 'p')
	tm7ExpectRowChars(t, term, 3, 50, 60, 'S')

	// The capacity stayed the same: the page was only rehashed.
	rehashed := tm7Node(t, term.Screens.Active.Pages.pages.last)
	expectEqual(t, uint(old_capacity), uint(rehashed.data.capacity.styles))
	expectEqual(t, need-50+1, rehashed.data.styles.Count())
	expect(t, uint(rehashed.data.styles.next_id) <= need-50+2, "rehashed.data.styles.next_id <= need - 50 + 2")
	expectEqual(t, uint(0), first.data.styles.Count())
}

// Zig: test "term51: deleteLines: a full style set in the destination page is grown when a row crosses pages"
func TestTm7_term51_deleteLines_a_full_style_set_in_the_destination_page_is_grown_when_a_row_crosses_pages(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	opts := terminalOptionsDefault(60, 5)
	opts.MaxScrollback = 1024
	term := tm7Init(t, opts)
	defer term.Deinit()

	// Rows 0-3 are on the first page, row 4 is the only one on the second.
	tm7CrossPage(t, term, 1)
	first := tm7Node(t, term.Screens.Active.Pages.pages.first)
	second := tm7Node(t, first.next)
	old_capacity := first.data.capacity.styles
	need := first.data.styles.layout.cap - 1
	expect(t, need > 60 && need <= 120, "need > 60 and need <= 120")

	tm7PutStr(t, term, 0, 0, strings.Repeat("0", 60))
	tm7PutStr(t, term, 0, 3, strings.Repeat("3", 60))
	tm7PrintDistinctStyles(t, term, 1, 60, need)
	expectEqual(t, need, first.data.styles.Count())

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	tm7PutStr(t, term, 0, 4, strings.Repeat("Z", 60))
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))
	expectEqual(t, uint(1), second.data.styles.Count())

	term.SetCursorPos(1, 1)
	term.DeleteLines(1)

	tm7ExpectRowChars(t, term, 0, 0, 60, 'S')
	tm7ExpectRowChars(t, term, 1, 0, need-60, 'S')
	tm7ExpectRowChars(t, term, 1, need-60, 60, '.')
	tm7ExpectRowChars(t, term, 2, 0, 60, '3')
	tm7ExpectRowChars(t, term, 3, 0, 60, 'Z')
	tm7ExpectRowChars(t, term, 4, 0, 60, '.')
	tm7ExpectCursor(t, term, 0, 0, false)

	grown := tm7Node(t, term.Screens.Active.Pages.pages.first)
	expectEqual(t, uint(old_capacity)*2, uint(grown.data.capacity.styles))
	expectEqual(t, need+1, grown.data.styles.Count())
	expect(t, grown.next == term.Screens.Active.Pages.pages.last, "grown.next.? == t.screens.active.pages.pages.last.?")
	expectEqual(t, uint(0), tm7Node(t, grown.next).data.styles.Count())
	bold := tm7CellAt(t, term, 0, 3).StyleID()
	expectEqual(t, SizeCellCountInt(60), grown.data.styles.RefCount(bold))
}

// Zig: test "term51: deleteLines: a style set that needs a rehash in the destination page is rehashed when a row crosses pages"
func TestTm7_term51_deleteLines_a_style_set_that_needs_a_rehash_in_the_destination_page_is_rehashed_when_a_row_crosses_pages(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	opts := terminalOptionsDefault(60, 5)
	opts.MaxScrollback = 1024
	term := tm7Init(t, opts)
	defer term.Deinit()

	tm7CrossPage(t, term, 1)
	first := tm7Node(t, term.Screens.Active.Pages.pages.first)
	old_capacity := first.data.capacity.styles
	need := first.data.styles.layout.cap - 1
	expect(t, need > 60 && need <= 120, "need > 60 and need <= 120")

	tm7PrintDistinctStyles(t, term, 1, 60, need)
	tm7PutStr(t, term, 0, 1, strings.Repeat("p", 50))
	expectEqual(t, need-50, first.data.styles.Count())
	expectEqual(t, first.data.styles.layout.cap, uint(first.data.styles.next_id))

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	tm7PutStr(t, term, 0, 4, strings.Repeat("Z", 60))
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))

	term.SetCursorPos(1, 1)
	term.DeleteLines(1)

	tm7ExpectRowChars(t, term, 0, 0, 50, 'p')
	tm7ExpectRowChars(t, term, 0, 50, 60, 'S')
	tm7ExpectRowChars(t, term, 3, 0, 60, 'Z')
	tm7ExpectRowChars(t, term, 4, 0, 60, '.')

	rehashed := tm7Node(t, term.Screens.Active.Pages.pages.first)
	expectEqual(t, uint(old_capacity), uint(rehashed.data.capacity.styles))
	expectEqual(t, need-50+1, rehashed.data.styles.Count())
	expect(t, uint(rehashed.data.styles.next_id) <= need-50+2, "rehashed.data.styles.next_id <= need - 50 + 2")
}

// Zig: test "term51: insertLines: a full grapheme map in the destination page is grown when a row crosses pages"
func TestTm7_term51_insertLines_a_full_grapheme_map_in_the_destination_page_is_grown_when_a_row_crosses_pages(t *testing.T) {
	testStdCapacity(t)
	tm7CrossGrapheme(t, tm7Cross{is_insert: true}, true)
}

// Zig: test "term51: insertLines: full grapheme storage in the destination page is grown when a row crosses pages"
func TestTm7_term51_insertLines_full_grapheme_storage_in_the_destination_page_is_grown_when_a_row_crosses_pages(t *testing.T) {
	testStdCapacity(t)
	tm7CrossGrapheme(t, tm7Cross{is_insert: true}, false)
}

// Zig: test "term51: insertLines: a full hyperlink map in the destination page is grown when a row crosses pages"
func TestTm7_term51_insertLines_a_full_hyperlink_map_in_the_destination_page_is_grown_when_a_row_crosses_pages(t *testing.T) {
	testStdCapacity(t)
	tm7CrossHyperlink(t, tm7Cross{is_insert: true}, tm7HyperlinkFullMap)
}

// Zig: test "term51: insertLines: full string storage in the destination page is grown when a row crosses pages"
func TestTm7_term51_insertLines_full_string_storage_in_the_destination_page_is_grown_when_a_row_crosses_pages(t *testing.T) {
	testStdCapacity(t)
	tm7CrossHyperlink(t, tm7Cross{is_insert: true}, tm7HyperlinkFullString)
}

// Zig: test "term51: insertLines: a full hyperlink set in the destination page is grown when a row crosses pages"
func TestTm7_term51_insertLines_a_full_hyperlink_set_in_the_destination_page_is_grown_when_a_row_crosses_pages(t *testing.T) {
	testStdCapacity(t)
	tm7CrossHyperlink(t, tm7Cross{is_insert: true}, tm7HyperlinkFullSet)
}

// Zig: test "term51: insertLines: a hyperlink set that needs a rehash in the destination page is rehashed when a row crosses pages"
func TestTm7_term51_insertLines_a_hyperlink_set_that_needs_a_rehash_in_the_destination_page_is_rehashed_when_a_row_crosses_pages(t *testing.T) {
	testStdCapacity(t)
	tm7CrossHyperlink(t, tm7Cross{is_insert: true}, tm7HyperlinkFullRehash)
}

// Zig: test "term51: deleteLines: a full grapheme map in the destination page is grown when a row crosses pages"
func TestTm7_term51_deleteLines_a_full_grapheme_map_in_the_destination_page_is_grown_when_a_row_crosses_pages(t *testing.T) {
	testStdCapacity(t)
	tm7CrossGrapheme(t, tm7Cross{is_insert: false}, true)
}

// Zig: test "term51: deleteLines: full grapheme storage in the destination page is grown when a row crosses pages"
func TestTm7_term51_deleteLines_full_grapheme_storage_in_the_destination_page_is_grown_when_a_row_crosses_pages(t *testing.T) {
	testStdCapacity(t)
	tm7CrossGrapheme(t, tm7Cross{is_insert: false}, false)
}

// Zig: test "term51: deleteLines: a full hyperlink map in the destination page is grown when a row crosses pages"
func TestTm7_term51_deleteLines_a_full_hyperlink_map_in_the_destination_page_is_grown_when_a_row_crosses_pages(t *testing.T) {
	testStdCapacity(t)
	tm7CrossHyperlink(t, tm7Cross{is_insert: false}, tm7HyperlinkFullMap)
}

// Zig: test "term51: deleteLines: full string storage in the destination page is grown when a row crosses pages"
func TestTm7_term51_deleteLines_full_string_storage_in_the_destination_page_is_grown_when_a_row_crosses_pages(t *testing.T) {
	testStdCapacity(t)
	tm7CrossHyperlink(t, tm7Cross{is_insert: false}, tm7HyperlinkFullString)
}

// Zig: test "term51: deleteLines: a full hyperlink set in the destination page is grown when a row crosses pages"
func TestTm7_term51_deleteLines_a_full_hyperlink_set_in_the_destination_page_is_grown_when_a_row_crosses_pages(t *testing.T) {
	testStdCapacity(t)
	tm7CrossHyperlink(t, tm7Cross{is_insert: false}, tm7HyperlinkFullSet)
}

// Zig: test "term51: deleteLines: a hyperlink set that needs a rehash in the destination page is rehashed when a row crosses pages"
func TestTm7_term51_deleteLines_a_hyperlink_set_that_needs_a_rehash_in_the_destination_page_is_rehashed_when_a_row_crosses_pages(t *testing.T) {
	testStdCapacity(t)
	tm7CrossHyperlink(t, tm7Cross{is_insert: false}, tm7HyperlinkFullRehash)
}

// Zig: test "term51: insertBlanks: zero count clears pending wrap and changes nothing"
func TestTm7_term51_insertBlanks_zero_count_clears_pending_wrap_and_changes_nothing(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 3, 0)
	defer term.Deinit()

	term.Screens.Active.Cursor.PendingWrap = true
	term.InsertBlanks(0)

	tm7ExpectGrid(t, term, []string{"abcdefgh", "ijklmnop"})
	tm7ExpectCursor(t, term, 3, 0, false)
	tm7ExpectDirty(t, term, "..")
}

// Zig: test "term51: insertBlanks: cursor left of the left margin clears pending wrap and does nothing"
func TestTm7_term51_insertBlanks_cursor_left_of_the_left_margin_clears_pending_wrap_and_does_nothing(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 1, 0)
	defer term.Deinit()

	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 5
	term.Screens.Active.Cursor.PendingWrap = true
	term.InsertBlanks(1)

	tm7ExpectGrid(t, term, []string{"abcdefgh", "ijklmnop"})
	tm7ExpectCursor(t, term, 1, 0, false)
	tm7ExpectDirty(t, term, "..")
}

// Zig: test "term51: insertBlanks: cursor right of the right margin clears pending wrap and does nothing"
func TestTm7_term51_insertBlanks_cursor_right_of_the_right_margin_clears_pending_wrap_and_does_nothing(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 6, 0)
	defer term.Deinit()

	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 5
	term.Screens.Active.Cursor.PendingWrap = true
	term.InsertBlanks(1)

	tm7ExpectGrid(t, term, []string{"abcdefgh", "ijklmnop"})
	tm7ExpectCursor(t, term, 6, 0, false)
	tm7ExpectDirty(t, term, "..")
}

// Zig: test "term51: insertBlanks: one blank in the middle shifts the rest right"
func TestTm7_term51_insertBlanks_one_blank_in_the_middle_shifts_the_rest_right(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 2, 0)
	defer term.Deinit()

	term.Screens.Active.Cursor.PendingWrap = true
	term.InsertBlanks(1)

	tm7ExpectGrid(t, term, []string{"ab.cdefg", "ijklmnop"})
	tm7ExpectCursor(t, term, 2, 0, false)
	tm7ExpectDirty(t, term, "D.")
}

// Zig: test "term51: insertBlanks: count one less than the remaining columns keeps one cell"
func TestTm7_term51_insertBlanks_count_one_less_than_the_remaining_columns_keeps_one_cell(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 2, 0)
	defer term.Deinit()

	term.InsertBlanks(5)

	tm7ExpectGrid(t, term, []string{"ab.....c", "ijklmnop"})
}

// Zig: test "term51: insertBlanks: count equal to the remaining columns clears them"
func TestTm7_term51_insertBlanks_count_equal_to_the_remaining_columns_clears_them(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 2, 0)
	defer term.Deinit()

	term.InsertBlanks(6)

	tm7ExpectGrid(t, term, []string{"ab......", "ijklmnop"})
}

// Zig: test "term51: insertBlanks: count one more than the remaining columns is clamped"
func TestTm7_term51_insertBlanks_count_one_more_than_the_remaining_columns_is_clamped(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 2, 0)
	defer term.Deinit()

	term.InsertBlanks(7)

	tm7ExpectGrid(t, term, []string{"ab......", "ijklmnop"})
}

// Zig: test "term51: insertBlanks: maximum count is clamped to the remaining columns"
func TestTm7_term51_insertBlanks_maximum_count_is_clamped_to_the_remaining_columns(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 2, 0)
	defer term.Deinit()

	term.InsertBlanks(math.MaxUint)

	tm7ExpectGrid(t, term, []string{"ab......", "ijklmnop"})
	tm7ExpectCursor(t, term, 2, 0, false)
}

// Zig: test "term51: insertBlanks: on the last column only that cell is cleared"
func TestTm7_term51_insertBlanks_on_the_last_column_only_that_cell_is_cleared(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 7, 0)
	defer term.Deinit()

	term.InsertBlanks(1)
	tm7ExpectGrid(t, term, []string{"abcdefg.", "ijklmnop"})

	tm7Fill(t, term, tm7Alpha8[:])
	term.SetCursorPos(1, 8)
	term.InsertBlanks(math.MaxUint)
	tm7ExpectGrid(t, term, []string{"abcdefg.", "ijklmnop"})
}

// Zig: test "term51: insertBlanks: at the first column with the full width"
func TestTm7_term51_insertBlanks_at_the_first_column_with_the_full_width(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 0)
	defer term.Deinit()

	term.InsertBlanks(8)

	tm7ExpectGrid(t, term, []string{"........", "ijklmnop"})
}

// Zig: test "term51: insertBlanks: only the cursor row is touched"
func TestTm7_term51_insertBlanks_only_the_cursor_row_is_touched(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 1)
	defer term.Deinit()

	term.InsertBlanks(2)

	tm7ExpectGrid(t, term, []string{"abcdefgh", "..ijklmn"})
	tm7ExpectDirty(t, term, ".D")
}

// Zig: test "term51: insertBlanks: left and right margins confine the shift"
func TestTm7_term51_insertBlanks_left_and_right_margins_confine_the_shift(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 3, 0)
	defer term.Deinit()

	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 5
	term.InsertBlanks(1)

	tm7ExpectGrid(t, term, []string{"abc.degh", "ijklmnop"})
	tm7ExpectCursor(t, term, 3, 0, false)
}

// Zig: test "term51: insertBlanks: count larger than the margin region clears only the margin"
func TestTm7_term51_insertBlanks_count_larger_than_the_margin_region_clears_only_the_margin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 3, 0)
	defer term.Deinit()

	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 5
	term.InsertBlanks(100)

	tm7ExpectGrid(t, term, []string{"abc...gh", "ijklmnop"})
}

// Zig: test "term51: insertBlanks: cursor on the left margin column"
func TestTm7_term51_insertBlanks_cursor_on_the_left_margin_column(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 2, 0)
	defer term.Deinit()

	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 5
	term.InsertBlanks(1)

	tm7ExpectGrid(t, term, []string{"ab.cdegh", "ijklmnop"})
}

// Zig: test "term51: insertBlanks: cursor on the right margin column"
func TestTm7_term51_insertBlanks_cursor_on_the_right_margin_column(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 5, 0)
	defer term.Deinit()

	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 5
	term.InsertBlanks(1)

	tm7ExpectGrid(t, term, []string{"abcde.gh", "ijklmnop"})
}

// Zig: test "term51: insertBlanks: only a right margin"
func TestTm7_term51_insertBlanks_only_a_right_margin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 1, 0)
	defer term.Deinit()

	term.ScrollingRegion.Right = 4
	term.InsertBlanks(2)

	tm7ExpectGrid(t, term, []string{"a..bcfgh", "ijklmnop"})
}

// Zig: test "term51: insertBlanks: only a left margin"
func TestTm7_term51_insertBlanks_only_a_left_margin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 4, 0)
	defer term.Deinit()

	term.ScrollingRegion.Left = 3
	term.InsertBlanks(2)

	tm7ExpectGrid(t, term, []string{"abcd..ef", "ijklmnop"})
}

// Zig: test "term51: insertBlanks: top and bottom margins do not matter"
func TestTm7_term51_insertBlanks_top_and_bottom_margins_do_not_matter(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 2, 0)
	defer term.Deinit()

	// The cursor is above the scroll region but still inside its columns.
	term.ScrollingRegion.Top = 1
	term.ScrollingRegion.Bottom = 1
	term.InsertBlanks(1)

	tm7ExpectGrid(t, term, []string{"ab.cdefg", "ijklmnop"})
}

// Zig: test "term51: insertBlanks: blanks take the background color of the cursor style"
func TestTm7_term51_insertBlanks_blanks_take_the_background_color_of_the_cursor_style(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 1, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: 4, G: 5, B: 6}}))
	term.InsertBlanks(2)

	tm7ExpectGrid(t, term, []string{"arrbcdef", "ijklmnop"})
	for x := uint(1); x < 3; x++ {
		cell := *tm7CellAt(t, term, x, 0)
		expectEqual(t, PageCellContentTagBgColorRgb, cell.ContentTag())
		expectEqual(t, PageCellRGB{R: 4, G: 5, B: 6}, cell.content().colorRGB())
		expectEqual(t, StyleDefaultID, cell.StyleID())
	}
}

// Zig: test "term51: insertBlanks: blanks take a palette background color"
func TestTm7_term51_insertBlanks_blanks_take_a_palette_background_color(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTag256Bg, X256Bg: 9}))
	term.InsertBlanks(3)

	tm7ExpectGrid(t, term, []string{"pppabcde", "ijklmnop"})
	for x := uint(0); x < 3; x++ {
		cell := *tm7CellAt(t, term, x, 0)
		expectEqual(t, PageCellContentTagBgColorPalette, cell.ContentTag())
		expectEqual(t, uint8(9), cell.content().colorPalette())
	}
}

// Zig: test "term51: insertBlanks: only the margin columns are colored"
func TestTm7_term51_insertBlanks_only_the_margin_columns_are_colored(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 3, 0)
	defer term.Deinit()

	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 5
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: 4, G: 5, B: 6}}))
	term.InsertBlanks(1)

	tm7ExpectGrid(t, term, []string{"abcrdegh", "ijklmnop"})
}

// Zig: test "term51: insertBlanks: a cursor style without a background leaves blanks empty"
func TestTm7_term51_insertBlanks_a_cursor_style_without_a_background_leaves_blanks_empty(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 1, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	term.InsertBlanks(2)

	tm7ExpectGrid(t, term, []string{"a..bcdef", "ijklmnop"})
	expectEqual(t, StyleDefaultID, tm7CellAt(t, term, 1, 0).StyleID())
}

// Zig: test "term51: insertBlanks: protected cells move with their text and blanks are unprotected"
func TestTm7_term51_insertBlanks_protected_cells_move_with_their_text_and_blanks_are_unprotected(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 0)
	defer term.Deinit()

	term.SetCursorPos(1, 3)
	term.SetProtectedMode(ANSIProtectedModeDec)
	zigTry(t, term.Print('C'))
	zigTry(t, term.Print('D'))
	term.SetProtectedMode(ANSIProtectedModeOff)
	term.SetCursorPos(1, 1)
	term.InsertBlanks(2)

	tm7ExpectGrid(t, term, []string{"..abCDef", "ijklmnop"})
	for x := uint(0); x < 8; x++ {
		expectEqual(t, x == 4 || x == 5, tm7CellAt(t, term, x, 0).Protected())
	}
}

// Zig: test "term51: insertBlanks: ISO protection does not keep cells from being pushed out"
func TestTm7_term51_insertBlanks_ISO_protection_does_not_keep_cells_from_being_pushed_out(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 0)
	defer term.Deinit()

	term.SetCursorPos(1, 7)
	term.SetProtectedMode(ANSIProtectedModeIso)
	zigTry(t, term.Print('G'))
	zigTry(t, term.Print('H'))
	term.SetCursorPos(1, 1)
	term.InsertBlanks(2)

	tm7ExpectGrid(t, term, []string{"..abcdef", "ijklmnop"})
	for x := uint(0); x < 8; x++ {
		expect(t, !tm7CellAt(t, term, x, 0).Protected(), "!term51UCellAt(&t, x, 0).protected")
	}
}

// Zig: test "term51: insertBlanks: style reference counts follow the shifted and pushed out cells"
func TestTm7_term51_insertBlanks_style_reference_counts_follow_the_shifted_and_pushed_out_cells(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	tm7PutStr(t, term, 0, 0, "abcdefgh")
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))
	page := tm7Page(term)
	bold := tm7CellAt(t, term, 0, 0).StyleID()
	expectEqual(t, SizeCellCountInt(8), page.styles.RefCount(bold))

	term.SetCursorPos(1, 2)
	term.InsertBlanks(2)
	tm7ExpectGrid(t, term, []string{"a..bcdef", "ijklmnop"})
	expectEqual(t, SizeCellCountInt(6), page.styles.RefCount(bold))
	expectEqual(t, StyleDefaultID, tm7CellAt(t, term, 1, 0).StyleID())
	expectEqual(t, StyleDefaultID, tm7CellAt(t, term, 2, 0).StyleID())
	expectEqual(t, bold, tm7CellAt(t, term, 3, 0).StyleID())
	expectEqual(t, bold, tm7CellAt(t, term, 7, 0).StyleID())
	expect(t, tm7RowAt(t, term, 0).styled, "term51URowAt(&t, 0).styled")

	// Clearing the whole row releases the rest and clears the row flag.
	term.SetCursorPos(1, 1)
	term.InsertBlanks(8)
	tm7ExpectGrid(t, term, []string{"........", "ijklmnop"})
	expectEqual(t, uint(0), page.styles.Count())
	expect(t, !tm7RowAt(t, term, 0).styled, "!term51URowAt(&t, 0).styled")
}

// Zig: test "term51: insertBlanks: the styled flag of a partially cleared row stays set"
func TestTm7_term51_insertBlanks_the_styled_flag_of_a_partially_cleared_row_stays_set(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	tm7PutStr(t, term, 6, 0, "gh")
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))
	page := tm7Page(term)
	expectEqual(t, uint(1), page.styles.Count())

	term.SetCursorPos(1, 1)
	term.InsertBlanks(2)

	tm7ExpectGrid(t, term, []string{"..abcdef", "ijklmnop"})
	expectEqual(t, uint(0), page.styles.Count())
	// Quirk: the flag is refreshed before the cells are blanked, so it stays.
	expect(t, tm7RowAt(t, term, 0).styled, "term51URowAt(&t, 0).styled")
	for x := uint(0); x < 8; x++ {
		expectEqual(t, StyleDefaultID, tm7CellAt(t, term, x, 0).StyleID())
	}
}

// Zig: test "term51: insertBlanks: graphemes move right and are released when pushed out"
func TestTm7_term51_insertBlanks_graphemes_move_right_and_are_released_when_pushed_out(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 0)
	defer term.Deinit()

	tm7PutGrapheme(t, term, 2, 0)
	page := tm7Page(term)
	expectEqual(t, uint(1), page.GraphemeCount())

	term.SetCursorPos(1, 1)
	term.InsertBlanks(1)
	tm7ExpectGrid(t, term, []string{".ab*defg", "ijklmnop"})
	expectEqual(t, uint(1), page.GraphemeCount())
	expectEqualSlices(t, []uint32{0x0301}, zigUnwrap(t, page.LookupGrapheme(tm7CellAt(t, term, 3, 0))))
	expect(t, tm7RowAt(t, term, 0).grapheme, "term51URowAt(&t, 0).grapheme")

	term.SetCursorPos(1, 4)
	term.InsertBlanks(5)
	tm7ExpectGrid(t, term, []string{".ab.....", "ijklmnop"})
	expectEqual(t, uint(0), page.GraphemeCount())
	expect(t, !tm7RowAt(t, term, 0).grapheme, "!term51URowAt(&t, 0).grapheme")
}

// Zig: test "term51: insertBlanks: a grapheme pushed off the end of a partially cleared row is released"
func TestTm7_term51_insertBlanks_a_grapheme_pushed_off_the_end_of_a_partially_cleared_row_is_released(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 0)
	defer term.Deinit()

	tm7PutGrapheme(t, term, 7, 0)
	page := tm7Page(term)

	term.SetCursorPos(1, 4)
	term.InsertBlanks(1)
	tm7ExpectGrid(t, term, []string{"abc.defg", "ijklmnop"})
	expectEqual(t, uint(0), page.GraphemeCount())
	expect(t, !tm7RowAt(t, term, 0).grapheme, "!term51URowAt(&t, 0).grapheme")
}

// Zig: test "term51: insertBlanks: hyperlinks move right and are released when pushed out"
func TestTm7_term51_insertBlanks_hyperlinks_move_right_and_are_released_when_pushed_out(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 0)
	defer term.Deinit()

	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://a.test"), None[[]uint8]()))
	tm7PutStr(t, term, 4, 0, "efgh")
	term.Screens.Active.EndHyperlink()
	page := tm7Page(term)
	expectEqual(t, uint(4), page.HyperlinkCount())

	term.SetCursorPos(1, 1)
	term.InsertBlanks(2)
	tm7ExpectGrid(t, term, []string{"..abcdef", "ijklmnop"})
	expectEqual(t, uint(2), page.HyperlinkCount())
	for x := uint(0); x < 8; x++ {
		expectEqual(t, x >= 6, tm7CellAt(t, term, x, 0).Hyperlink())
	}
	expect(t, tm7RowAt(t, term, 0).hyperlink, "term51URowAt(&t, 0).hyperlink")

	term.InsertBlanks(8)
	expectEqual(t, uint(0), page.HyperlinkCount())
	expectEqual(t, uint(0), page.hyperlink_set.Count())
	expect(t, !tm7RowAt(t, term, 0).hyperlink, "!term51URowAt(&t, 0).hyperlink")
}

// Zig: test "term51: insertBlanks: a hyperlink pushed off the end of a partially cleared row clears the row flag"
func TestTm7_term51_insertBlanks_a_hyperlink_pushed_off_the_end_of_a_partially_cleared_row_clears_the_row_flag(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 0)
	defer term.Deinit()

	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://a.test"), None[[]uint8]()))
	tm7PutStr(t, term, 7, 0, "h")
	term.Screens.Active.EndHyperlink()
	page := tm7Page(term)
	expectEqual(t, uint(1), page.HyperlinkCount())

	term.SetCursorPos(1, 2)
	term.InsertBlanks(1)
	tm7ExpectGrid(t, term, []string{"a.bcdefg", "ijklmnop"})
	expectEqual(t, uint(0), page.HyperlinkCount())
	expect(t, !tm7RowAt(t, term, 0).hyperlink, "!term51URowAt(&t, 0).hyperlink")
}

// Zig: test "term51: insertBlanks: the soft wrap flags of the row are kept"
func TestTm7_term51_insertBlanks_the_soft_wrap_flags_of_the_row_are_kept(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 0)
	defer term.Deinit()

	tm7SetWrap(t, term, "wc")
	term.InsertBlanks(1)

	tm7ExpectGrid(t, term, []string{".abcdefg", "ijklmnop"})
	tm7ExpectWrap(t, term, "wc")
}

// Zig: test "term51: insertBlanks: cursor on a spacer tail clears the wide character first"
func TestTm7_term51_insertBlanks_cursor_on_a_spacer_tail_clears_the_wide_character_first(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7PutStr(t, term, 0, 0, "ab")
	tm7PutWide(t, term, 2, 0)
	tm7PutStr(t, term, 4, 0, "cdef")
	tm7ExpectGrid(t, term, []string{"ab#>cdef", "........"})

	term.SetCursorPos(1, 4)
	term.InsertBlanks(1)

	tm7ExpectGrid(t, term, []string{"ab...cde", "........"})
	tm7ExpectCursor(t, term, 3, 0, false)
}

// Zig: test "term51: insertBlanks: cursor on a spacer tail with a large count clears through the margin"
func TestTm7_term51_insertBlanks_cursor_on_a_spacer_tail_with_a_large_count_clears_through_the_margin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7PutStr(t, term, 0, 0, "ab")
	tm7PutWide(t, term, 2, 0)
	tm7PutStr(t, term, 4, 0, "cdef")

	term.SetCursorPos(1, 4)
	term.InsertBlanks(math.MaxUint)

	tm7ExpectGrid(t, term, []string{"ab......", "........"})
}

// Zig: test "term51: insertBlanks: cursor on a spacer tail at the left margin clears the wide character outside it"
func TestTm7_term51_insertBlanks_cursor_on_a_spacer_tail_at_the_left_margin_clears_the_wide_character_outside_it(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7PutStr(t, term, 0, 0, "ab")
	tm7PutWide(t, term, 2, 0)
	tm7PutStr(t, term, 4, 0, "cdef")

	term.ScrollingRegion.Left = 3
	term.SetCursorPos(1, 4)
	term.InsertBlanks(1)

	tm7ExpectGrid(t, term, []string{"ab...cde", "........"})
}

// Zig: test "term51: insertBlanks: cursor on a wide head moves the whole character"
func TestTm7_term51_insertBlanks_cursor_on_a_wide_head_moves_the_whole_character(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7PutStr(t, term, 0, 0, "ab")
	tm7PutWide(t, term, 2, 0)
	tm7PutStr(t, term, 4, 0, "cdef")

	term.SetCursorPos(1, 3)
	term.InsertBlanks(1)
	tm7ExpectGrid(t, term, []string{"ab.#>cde", "........"})

	tm7Fill(t, term, []string{"ab..cdef", "........"})
	tm7PutWide(t, term, 2, 0)
	term.SetCursorPos(1, 3)
	term.InsertBlanks(2)
	tm7ExpectGrid(t, term, []string{"ab..#>cd", "........"})
}

// Zig: test "term51: insertBlanks: a shift that ends just after a wide character keeps it whole"
func TestTm7_term51_insertBlanks_a_shift_that_ends_just_after_a_wide_character_keeps_it_whole(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	// Count 3: the last shifted cell is the narrow 'c'; count 4: it is the
	// spacer tail.
	tm7WideRow(t, term)
	term.InsertBlanks(3)
	tm7ExpectGrid(t, term, []string{"...ab#>c", "........"})

	tm7Fill(t, term, []string{"........", "........"})
	tm7WideRow(t, term)
	term.InsertBlanks(4)
	tm7ExpectGrid(t, term, []string{"....ab#>", "........"})
}

// Zig: test "term51: insertBlanks: a shift whose last cell is a wide head drops the wide character"
func TestTm7_term51_insertBlanks_a_shift_whose_last_cell_is_a_wide_head_drops_the_wide_character(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7WideRow(t, term)
	term.InsertBlanks(5)

	tm7ExpectGrid(t, term, []string{".....ab.", "........"})
}

// Zig: test "term51: insertBlanks: a shift that stops before the wide character moves only narrow cells"
func TestTm7_term51_insertBlanks_a_shift_that_stops_before_the_wide_character_moves_only_narrow_cells(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7WideRow(t, term)
	term.InsertBlanks(6)
	tm7ExpectGrid(t, term, []string{"......ab", "........"})

	tm7Fill(t, term, []string{"........", "........"})
	tm7WideRow(t, term)
	term.InsertBlanks(7)
	tm7ExpectGrid(t, term, []string{".......a", "........"})
}

// Zig: test "term51: insertBlanks: a wide character at the end of the row is dropped whole"
func TestTm7_term51_insertBlanks_a_wide_character_at_the_end_of_the_row_is_dropped_whole(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7PutStr(t, term, 0, 0, "abcdef")
	tm7PutWide(t, term, 6, 0)
	tm7ExpectGrid(t, term, []string{"abcdef#>", "........"})

	term.SetCursorPos(1, 1)
	term.InsertBlanks(1)

	tm7ExpectGrid(t, term, []string{".abcdef.", "........"})
}

// Zig: test "term51: insertBlanks: a wide character on the right margin is cleared with its tail beyond the margin"
func TestTm7_term51_insertBlanks_a_wide_character_on_the_right_margin_is_cleared_with_its_tail_beyond_the_margin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7PutStr(t, term, 0, 0, "abcd")
	tm7PutWide(t, term, 4, 0)
	tm7PutStr(t, term, 6, 0, "gh")
	tm7ExpectGrid(t, term, []string{"abcd#>gh", "........"})

	term.ScrollingRegion.Right = 4
	term.SetCursorPos(1, 1)
	term.InsertBlanks(1)
	tm7ExpectGrid(t, term, []string{".abcd.gh", "........"})

	tm7Fill(t, term, []string{"abcd..gh", "........"})
	term.ScrollingRegion.Right = 7
	tm7PutWide(t, term, 4, 0)
	term.ScrollingRegion.Right = 4
	term.SetCursorPos(1, 1)
	term.InsertBlanks(math.MaxUint)
	tm7ExpectGrid(t, term, []string{"......gh", "........"})
}

// Zig: test "term51: insertBlanks: a wide character ending on the right margin is dropped when shifted"
func TestTm7_term51_insertBlanks_a_wide_character_ending_on_the_right_margin_is_dropped_when_shifted(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7PutStr(t, term, 0, 0, "abc")
	tm7PutWide(t, term, 3, 0)
	tm7PutStr(t, term, 5, 0, "fgh")
	tm7ExpectGrid(t, term, []string{"abc#>fgh", "........"})

	term.ScrollingRegion.Right = 4
	term.SetCursorPos(1, 1)
	term.InsertBlanks(1)

	tm7ExpectGrid(t, term, []string{".abc.fgh", "........"})
}

// Zig: test "term51: insertBlanks: a spacer head at the end of the row is shifted out"
func TestTm7_term51_insertBlanks_a_spacer_head_at_the_end_of_the_row_is_shifted_out(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7PutStr(t, term, 0, 0, "abcdefg")
	zigTry(t, term.Print(0x4E2D))
	tm7ExpectGrid(t, term, []string{"abcdefg<", "#>......"})

	term.SetCursorPos(1, 1)
	term.InsertBlanks(1)

	tm7ExpectGrid(t, term, []string{".abcdefg", "#>......"})
	// The wrap flags are not touched.
	tm7ExpectWrap(t, term, "wc")
}

// Zig: test "term51: insertBlanks: a spacer head beyond the right margin is kept"
func TestTm7_term51_insertBlanks_a_spacer_head_beyond_the_right_margin_is_kept(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7PutStr(t, term, 0, 0, "abcdefg")
	zigTry(t, term.Print(0x4E2D))

	term.ScrollingRegion.Right = 5
	term.SetCursorPos(1, 1)
	term.InsertBlanks(1)

	tm7ExpectGrid(t, term, []string{".abcdeg<", "#>......"})
}

// Zig: test "term51: insertBlanks: a wide character with a grapheme that is dropped releases the grapheme"
func TestTm7_term51_insertBlanks_a_wide_character_with_a_grapheme_that_is_dropped_releases_the_grapheme(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7PutStr(t, term, 0, 0, "abcdef")
	tm7PutWideGrapheme(t, term, 6, 0)
	page := tm7Page(term)
	expectEqual(t, uint(1), page.GraphemeCount())

	term.SetCursorPos(1, 1)
	term.InsertBlanks(1)

	tm7ExpectGrid(t, term, []string{".abcdef.", "........"})
	expectEqual(t, uint(0), page.GraphemeCount())
	expect(t, !tm7RowAt(t, term, 0).grapheme, "!term51URowAt(&t, 0).grapheme")
}

// Zig: test "term51: insertBlanks: a styled wide character that is dropped releases both cells"
func TestTm7_term51_insertBlanks_a_styled_wide_character_that_is_dropped_releases_both_cells(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7PutStr(t, term, 0, 0, "abcdef")
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	tm7PutWide(t, term, 6, 0)
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))
	page := tm7Page(term)
	expectEqual(t, uint(1), page.styles.Count())
	expectEqual(t, SizeCellCountInt(2), page.styles.RefCount(tm7CellAt(t, term, 6, 0).StyleID()))

	term.SetCursorPos(1, 1)
	term.InsertBlanks(1)

	tm7ExpectGrid(t, term, []string{".abcdef.", "........"})
	expectEqual(t, uint(0), page.styles.Count())
}

// Zig: test "term51: deleteChars: zero count keeps pending wrap and changes nothing"
func TestTm7_term51_deleteChars_zero_count_keeps_pending_wrap_and_changes_nothing(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 3, 0)
	defer term.Deinit()

	term.Screens.Active.Cursor.PendingWrap = true
	term.DeleteChars(0)

	tm7ExpectGrid(t, term, []string{"abcdefgh", "ijklmnop"})
	tm7ExpectCursor(t, term, 3, 0, true)
	tm7ExpectDirty(t, term, "..")
}

// Zig: test "term51: deleteChars: cursor left of the left margin keeps pending wrap and does nothing"
func TestTm7_term51_deleteChars_cursor_left_of_the_left_margin_keeps_pending_wrap_and_does_nothing(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 1, 0)
	defer term.Deinit()

	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 5
	term.Screens.Active.Cursor.PendingWrap = true
	term.DeleteChars(1)

	tm7ExpectGrid(t, term, []string{"abcdefgh", "ijklmnop"})
	tm7ExpectCursor(t, term, 1, 0, true)
	tm7ExpectDirty(t, term, "..")
}

// Zig: test "term51: deleteChars: cursor right of the right margin keeps pending wrap and does nothing"
func TestTm7_term51_deleteChars_cursor_right_of_the_right_margin_keeps_pending_wrap_and_does_nothing(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 6, 0)
	defer term.Deinit()

	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 5
	term.Screens.Active.Cursor.PendingWrap = true
	term.DeleteChars(1)

	tm7ExpectGrid(t, term, []string{"abcdefgh", "ijklmnop"})
	tm7ExpectCursor(t, term, 6, 0, true)
	tm7ExpectDirty(t, term, "..")
}

// Zig: test "term51: deleteChars: one character in the middle shifts the rest left"
func TestTm7_term51_deleteChars_one_character_in_the_middle_shifts_the_rest_left(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 2, 0)
	defer term.Deinit()

	term.Screens.Active.Cursor.PendingWrap = true
	term.DeleteChars(1)

	tm7ExpectGrid(t, term, []string{"abdefgh.", "ijklmnop"})
	tm7ExpectCursor(t, term, 2, 0, false)
	tm7ExpectDirty(t, term, "D.")
}

// Zig: test "term51: deleteChars: count one less than the remaining columns keeps one cell"
func TestTm7_term51_deleteChars_count_one_less_than_the_remaining_columns_keeps_one_cell(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 2, 0)
	defer term.Deinit()

	term.DeleteChars(5)

	tm7ExpectGrid(t, term, []string{"abh.....", "ijklmnop"})
}

// Zig: test "term51: deleteChars: count equal to the remaining columns clears them"
func TestTm7_term51_deleteChars_count_equal_to_the_remaining_columns_clears_them(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 2, 0)
	defer term.Deinit()

	term.DeleteChars(6)

	tm7ExpectGrid(t, term, []string{"ab......", "ijklmnop"})
}

// Zig: test "term51: deleteChars: count one more than the remaining columns is clamped"
func TestTm7_term51_deleteChars_count_one_more_than_the_remaining_columns_is_clamped(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 2, 0)
	defer term.Deinit()

	term.DeleteChars(7)

	tm7ExpectGrid(t, term, []string{"ab......", "ijklmnop"})
}

// Zig: test "term51: deleteChars: maximum count is clamped to the remaining columns"
func TestTm7_term51_deleteChars_maximum_count_is_clamped_to_the_remaining_columns(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 2, 0)
	defer term.Deinit()

	term.DeleteChars(math.MaxUint)

	tm7ExpectGrid(t, term, []string{"ab......", "ijklmnop"})
	tm7ExpectCursor(t, term, 2, 0, false)
}

// Zig: test "term51: deleteChars: on the last column only that cell is cleared"
func TestTm7_term51_deleteChars_on_the_last_column_only_that_cell_is_cleared(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 7, 0)
	defer term.Deinit()

	term.DeleteChars(1)
	tm7ExpectGrid(t, term, []string{"abcdefg.", "ijklmnop"})

	tm7Fill(t, term, tm7Alpha8[:])
	term.SetCursorPos(1, 8)
	term.DeleteChars(math.MaxUint)
	tm7ExpectGrid(t, term, []string{"abcdefg.", "ijklmnop"})
}

// Zig: test "term51: deleteChars: at the first column with the full width"
func TestTm7_term51_deleteChars_at_the_first_column_with_the_full_width(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 0)
	defer term.Deinit()

	term.DeleteChars(8)

	tm7ExpectGrid(t, term, []string{"........", "ijklmnop"})
}

// Zig: test "term51: deleteChars: only the cursor row is touched"
func TestTm7_term51_deleteChars_only_the_cursor_row_is_touched(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 1)
	defer term.Deinit()

	term.DeleteChars(2)

	tm7ExpectGrid(t, term, []string{"abcdefgh", "klmnop.."})
	tm7ExpectDirty(t, term, ".D")
}

// Zig: test "term51: deleteChars: left and right margins confine the shift"
func TestTm7_term51_deleteChars_left_and_right_margins_confine_the_shift(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 3, 0)
	defer term.Deinit()

	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 5
	term.DeleteChars(1)

	tm7ExpectGrid(t, term, []string{"abcef.gh", "ijklmnop"})
	tm7ExpectCursor(t, term, 3, 0, false)
}

// Zig: test "term51: deleteChars: count larger than the margin region clears only the margin"
func TestTm7_term51_deleteChars_count_larger_than_the_margin_region_clears_only_the_margin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 3, 0)
	defer term.Deinit()

	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 5
	term.DeleteChars(100)

	tm7ExpectGrid(t, term, []string{"abc...gh", "ijklmnop"})
}

// Zig: test "term51: deleteChars: cursor on the left margin column"
func TestTm7_term51_deleteChars_cursor_on_the_left_margin_column(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 2, 0)
	defer term.Deinit()

	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 5
	term.DeleteChars(1)

	tm7ExpectGrid(t, term, []string{"abdef.gh", "ijklmnop"})
}

// Zig: test "term51: deleteChars: cursor on the right margin column"
func TestTm7_term51_deleteChars_cursor_on_the_right_margin_column(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 5, 0)
	defer term.Deinit()

	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 5
	term.DeleteChars(1)

	tm7ExpectGrid(t, term, []string{"abcde.gh", "ijklmnop"})
}

// Zig: test "term51: deleteChars: only a right margin"
func TestTm7_term51_deleteChars_only_a_right_margin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 1, 0)
	defer term.Deinit()

	term.ScrollingRegion.Right = 4
	term.DeleteChars(2)

	tm7ExpectGrid(t, term, []string{"ade..fgh", "ijklmnop"})
}

// Zig: test "term51: deleteChars: only a left margin"
func TestTm7_term51_deleteChars_only_a_left_margin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 4, 0)
	defer term.Deinit()

	term.ScrollingRegion.Left = 3
	term.DeleteChars(2)

	tm7ExpectGrid(t, term, []string{"abcdgh..", "ijklmnop"})
}

// Zig: test "term51: deleteChars: top and bottom margins do not matter"
func TestTm7_term51_deleteChars_top_and_bottom_margins_do_not_matter(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 2, 0)
	defer term.Deinit()

	term.ScrollingRegion.Top = 1
	term.ScrollingRegion.Bottom = 1
	term.DeleteChars(1)

	tm7ExpectGrid(t, term, []string{"abdefgh.", "ijklmnop"})
}

// Zig: test "term51: deleteChars: blanks take the background color of the cursor style"
func TestTm7_term51_deleteChars_blanks_take_the_background_color_of_the_cursor_style(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 1, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: 4, G: 5, B: 6}}))
	term.DeleteChars(2)

	tm7ExpectGrid(t, term, []string{"adefghrr", "ijklmnop"})
	for x := uint(6); x < 8; x++ {
		cell := *tm7CellAt(t, term, x, 0)
		expectEqual(t, PageCellContentTagBgColorRgb, cell.ContentTag())
		expectEqual(t, PageCellRGB{R: 4, G: 5, B: 6}, cell.content().colorRGB())
	}
}

// Zig: test "term51: deleteChars: blanks take a palette background color"
func TestTm7_term51_deleteChars_blanks_take_a_palette_background_color(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTag256Bg, X256Bg: 9}))
	term.DeleteChars(3)

	tm7ExpectGrid(t, term, []string{"defghppp", "ijklmnop"})
	for x := uint(5); x < 8; x++ {
		cell := *tm7CellAt(t, term, x, 0)
		expectEqual(t, PageCellContentTagBgColorPalette, cell.ContentTag())
		expectEqual(t, uint8(9), cell.content().colorPalette())
	}
}

// Zig: test "term51: deleteChars: only the margin columns are colored"
func TestTm7_term51_deleteChars_only_the_margin_columns_are_colored(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 3, 0)
	defer term.Deinit()

	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 5
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: 4, G: 5, B: 6}}))
	term.DeleteChars(1)

	tm7ExpectGrid(t, term, []string{"abcefrgh", "ijklmnop"})
}

// Zig: test "term51: deleteChars: a cursor style without a background leaves blanks empty"
func TestTm7_term51_deleteChars_a_cursor_style_without_a_background_leaves_blanks_empty(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 1, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	term.DeleteChars(2)

	tm7ExpectGrid(t, term, []string{"adefgh..", "ijklmnop"})
	expectEqual(t, StyleDefaultID, tm7CellAt(t, term, 7, 0).StyleID())
}

// Zig: test "term51: deleteChars: protected cells are deleted and move with their text"
func TestTm7_term51_deleteChars_protected_cells_are_deleted_and_move_with_their_text(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 0)
	defer term.Deinit()

	term.SetCursorPos(1, 2)
	term.SetProtectedMode(ANSIProtectedModeDec)
	zigTry(t, term.Print('B'))
	term.SetCursorPos(1, 6)
	zigTry(t, term.Print('F'))
	term.SetProtectedMode(ANSIProtectedModeOff)
	term.SetCursorPos(1, 1)
	term.DeleteChars(1)

	// Protection does not stop the deletion, and the protected cells shift.
	tm7ExpectGrid(t, term, []string{"BcdeFgh.", "ijklmnop"})
	for x := uint(0); x < 8; x++ {
		expectEqual(t, x == 0 || x == 4, tm7CellAt(t, term, x, 0).Protected())
	}
}

// Zig: test "term51: deleteChars: ISO protection does not keep cells from being deleted"
func TestTm7_term51_deleteChars_ISO_protection_does_not_keep_cells_from_being_deleted(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 0)
	defer term.Deinit()

	term.SetCursorPos(1, 1)
	term.SetProtectedMode(ANSIProtectedModeIso)
	zigTry(t, term.Print('A'))
	zigTry(t, term.Print('B'))
	term.SetCursorPos(1, 1)
	term.DeleteChars(2)

	tm7ExpectGrid(t, term, []string{"cdefgh..", "ijklmnop"})
	for x := uint(0); x < 8; x++ {
		expect(t, !tm7CellAt(t, term, x, 0).Protected(), "!term51UCellAt(&t, x, 0).protected")
	}
}

// Zig: test "term51: deleteChars: style reference counts follow the shifted and deleted cells"
func TestTm7_term51_deleteChars_style_reference_counts_follow_the_shifted_and_deleted_cells(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	tm7PutStr(t, term, 0, 0, "abcdefgh")
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))
	page := tm7Page(term)
	bold := tm7CellAt(t, term, 0, 0).StyleID()

	term.SetCursorPos(1, 2)
	term.DeleteChars(2)
	tm7ExpectGrid(t, term, []string{"adefgh..", "ijklmnop"})
	expectEqual(t, SizeCellCountInt(6), page.styles.RefCount(bold))
	expectEqual(t, bold, tm7CellAt(t, term, 5, 0).StyleID())
	expectEqual(t, StyleDefaultID, tm7CellAt(t, term, 6, 0).StyleID())
	expect(t, tm7RowAt(t, term, 0).styled, "term51URowAt(&t, 0).styled")

	term.SetCursorPos(1, 1)
	term.DeleteChars(8)
	tm7ExpectGrid(t, term, []string{"........", "ijklmnop"})
	expectEqual(t, uint(0), page.styles.Count())
	expect(t, !tm7RowAt(t, term, 0).styled, "!term51URowAt(&t, 0).styled")
}

// Zig: test "term51: deleteChars: graphemes move left and are released when deleted"
func TestTm7_term51_deleteChars_graphemes_move_left_and_are_released_when_deleted(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 0)
	defer term.Deinit()

	tm7PutGrapheme(t, term, 4, 0)
	page := tm7Page(term)

	term.SetCursorPos(1, 1)
	term.DeleteChars(1)
	tm7ExpectGrid(t, term, []string{"bcd*fgh.", "ijklmnop"})
	expectEqual(t, uint(1), page.GraphemeCount())
	expectEqualSlices(t, []uint32{0x0301}, zigUnwrap(t, page.LookupGrapheme(tm7CellAt(t, term, 3, 0))))
	expect(t, tm7RowAt(t, term, 0).grapheme, "term51URowAt(&t, 0).grapheme")

	term.SetCursorPos(1, 4)
	term.DeleteChars(1)
	tm7ExpectGrid(t, term, []string{"bcdfgh..", "ijklmnop"})
	expectEqual(t, uint(0), page.GraphemeCount())
	expect(t, !tm7RowAt(t, term, 0).grapheme, "!term51URowAt(&t, 0).grapheme")
}

// Zig: test "term51: deleteChars: hyperlinks move left and are released when deleted"
func TestTm7_term51_deleteChars_hyperlinks_move_left_and_are_released_when_deleted(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 0)
	defer term.Deinit()

	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://a.test"), None[[]uint8]()))
	tm7PutStr(t, term, 4, 0, "efgh")
	term.Screens.Active.EndHyperlink()
	page := tm7Page(term)

	term.SetCursorPos(1, 1)
	term.DeleteChars(2)
	tm7ExpectGrid(t, term, []string{"cdefgh..", "ijklmnop"})
	expectEqual(t, uint(4), page.HyperlinkCount())
	for x := uint(0); x < 8; x++ {
		expectEqual(t, x >= 2 && x < 6, tm7CellAt(t, term, x, 0).Hyperlink())
	}
	expect(t, tm7RowAt(t, term, 0).hyperlink, "term51URowAt(&t, 0).hyperlink")

	term.SetCursorPos(1, 3)
	term.DeleteChars(100)
	expectEqual(t, uint(0), page.HyperlinkCount())
	expectEqual(t, uint(0), page.hyperlink_set.Count())
	expect(t, !tm7RowAt(t, term, 0).hyperlink, "!term51URowAt(&t, 0).hyperlink")
}

// Zig: test "term51: deleteChars: the soft wrap of the row is reset"
func TestTm7_term51_deleteChars_the_soft_wrap_of_the_row_is_reset(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 0)
	defer term.Deinit()

	tm7SetWrap(t, term, "wc")
	term.DeleteChars(1)

	tm7ExpectGrid(t, term, []string{"bcdefgh.", "ijklmnop"})
	// The row is no longer wrapped, and the next row no longer continues it.
	tm7ExpectWrap(t, term, "..")
}

// Zig: test "term51: deleteChars: a continuation flag on the cursor row is kept"
func TestTm7_term51_deleteChars_a_continuation_flag_on_the_cursor_row_is_kept(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 1)
	defer term.Deinit()

	tm7SetWrap(t, term, "wc")
	term.DeleteChars(1)

	// Only the wrap of the cursor row is reset.
	tm7ExpectWrap(t, term, "wc")
}

// Zig: test "term51: deleteChars: a spacer head at the end of the wrapped row is cleared"
func TestTm7_term51_deleteChars_a_spacer_head_at_the_end_of_the_wrapped_row_is_cleared(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7PutStr(t, term, 0, 0, "abcdefg")
	zigTry(t, term.Print(0x4E2D))
	tm7ExpectGrid(t, term, []string{"abcdefg<", "#>......"})

	term.SetCursorPos(1, 1)
	term.DeleteChars(1)

	tm7ExpectGrid(t, term, []string{"bcdefg..", "#>......"})
	tm7ExpectWrap(t, term, "..")
}

// Zig: test "term51: deleteChars: a spacer head beyond the right margin is cleared with the wrap reset"
func TestTm7_term51_deleteChars_a_spacer_head_beyond_the_right_margin_is_cleared_with_the_wrap_reset(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7PutStr(t, term, 0, 0, "abcdefg")
	zigTry(t, term.Print(0x4E2D))

	term.ScrollingRegion.Right = 5
	term.SetCursorPos(1, 1)
	term.DeleteChars(1)

	// The wrap reset is not aware of the margins and clears the head.
	tm7ExpectGrid(t, term, []string{"bcdef.g.", "#>......"})
	tm7ExpectWrap(t, term, "..")
}

// Zig: test "term51: deleteChars: cursor on a spacer tail clears the wide character"
func TestTm7_term51_deleteChars_cursor_on_a_spacer_tail_clears_the_wide_character(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7WideRow(t, term)
	term.SetCursorPos(1, 4)
	term.DeleteChars(1)

	tm7ExpectGrid(t, term, []string{"ab.cdef.", "........"})
}

// Zig: test "term51: deleteChars: cursor on a wide head deleting one cell leaves one blank"
func TestTm7_term51_deleteChars_cursor_on_a_wide_head_deleting_one_cell_leaves_one_blank(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7WideRow(t, term)
	term.SetCursorPos(1, 3)
	term.DeleteChars(1)

	tm7ExpectGrid(t, term, []string{"ab.cdef.", "........"})
}

// Zig: test "term51: deleteChars: deleting a whole wide character"
func TestTm7_term51_deleteChars_deleting_a_whole_wide_character(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7WideRow(t, term)
	term.SetCursorPos(1, 3)
	term.DeleteChars(2)

	tm7ExpectGrid(t, term, []string{"abcdef..", "........"})
}

// Zig: test "term51: deleteChars: a count that ends inside a wide character clears it"
func TestTm7_term51_deleteChars_a_count_that_ends_inside_a_wide_character_clears_it(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7PutStr(t, term, 0, 0, "abc")
	tm7PutWide(t, term, 3, 0)
	tm7PutStr(t, term, 5, 0, "fgh")
	tm7ExpectGrid(t, term, []string{"abc#>fgh", "........"})

	term.SetCursorPos(1, 2)
	term.DeleteChars(3)

	tm7ExpectGrid(t, term, []string{"a.fgh...", "........"})
}

// Zig: test "term51: deleteChars: a wide character moves whole when it is not split"
func TestTm7_term51_deleteChars_a_wide_character_moves_whole_when_it_is_not_split(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7WideRow(t, term)
	term.DeleteChars(1)

	tm7ExpectGrid(t, term, []string{"b#>cdef.", "........"})
}

// Zig: test "term51: deleteChars: a wide character on the right margin is cleared with its tail beyond the margin"
func TestTm7_term51_deleteChars_a_wide_character_on_the_right_margin_is_cleared_with_its_tail_beyond_the_margin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7PutStr(t, term, 0, 0, "abcd")
	tm7PutWide(t, term, 4, 0)
	tm7PutStr(t, term, 6, 0, "gh")
	tm7ExpectGrid(t, term, []string{"abcd#>gh", "........"})

	term.ScrollingRegion.Right = 4
	term.SetCursorPos(1, 1)
	term.DeleteChars(1)

	tm7ExpectGrid(t, term, []string{"bcd...gh", "........"})
}

// Zig: test "term51: deleteChars: a wide character ending on the right margin moves whole"
func TestTm7_term51_deleteChars_a_wide_character_ending_on_the_right_margin_moves_whole(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7PutStr(t, term, 0, 0, "abc")
	tm7PutWide(t, term, 3, 0)
	tm7PutStr(t, term, 5, 0, "fgh")

	term.ScrollingRegion.Right = 4
	term.SetCursorPos(1, 1)
	term.DeleteChars(1)

	tm7ExpectGrid(t, term, []string{"bc#>.fgh", "........"})
}

// Zig: test "term51: deleteChars: a wide character with a grapheme split by the count releases the grapheme"
func TestTm7_term51_deleteChars_a_wide_character_with_a_grapheme_split_by_the_count_releases_the_grapheme(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7PutStr(t, term, 0, 0, "abc")
	tm7PutWideGrapheme(t, term, 3, 0)
	tm7PutStr(t, term, 5, 0, "fgh")
	page := tm7Page(term)
	expectEqual(t, uint(1), page.GraphemeCount())

	term.SetCursorPos(1, 2)
	term.DeleteChars(3)

	tm7ExpectGrid(t, term, []string{"a.fgh...", "........"})
	expectEqual(t, uint(0), page.GraphemeCount())
	expect(t, !tm7RowAt(t, term, 0).grapheme, "!term51URowAt(&t, 0).grapheme")
}

// Zig: test "term51: deleteChars: a wide character at the start of a continuation row clears the spacer head above"
func TestTm7_term51_deleteChars_a_wide_character_at_the_start_of_a_continuation_row_clears_the_spacer_head_above(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	tm7PutStr(t, term, 0, 0, "abcdefg")
	zigTry(t, term.Print(0x4E2D))
	tm7ExpectGrid(t, term, []string{"abcdefg<", "#>......"})
	term.SetCursorPos(2, 1)
	term.clearDirty()

	term.DeleteChars(1)

	tm7ExpectGrid(t, term, []string{"abcdefg.", "........"})
	// The previous row is edited in place and not marked dirty.
	tm7ExpectDirty(t, term, ".D")
	// The wrap flag of the previous row is not touched.
	tm7ExpectWrap(t, term, "wc")
}

// Zig: test "term51: eraseChars: one character"
func TestTm7_term51_eraseChars_one_character(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 2, 0)
	defer term.Deinit()

	term.EraseChars(1)

	tm7ExpectGrid(t, term, []string{"ab.defgh", "ijklmnop"})
	tm7ExpectCursor(t, term, 2, 0, false)
	tm7ExpectDirty(t, term, "D.")
}

// Zig: test "term51: eraseChars: zero count erases one character"
func TestTm7_term51_eraseChars_zero_count_erases_one_character(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 2, 0)
	defer term.Deinit()

	term.EraseChars(0)

	tm7ExpectGrid(t, term, []string{"ab.defgh", "ijklmnop"})
}

// Zig: test "term51: eraseChars: several characters do not shift the rest"
func TestTm7_term51_eraseChars_several_characters_do_not_shift_the_rest(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 2, 0)
	defer term.Deinit()

	term.EraseChars(3)

	tm7ExpectGrid(t, term, []string{"ab...fgh", "ijklmnop"})
}

// Zig: test "term51: eraseChars: count one less than the remaining columns"
func TestTm7_term51_eraseChars_count_one_less_than_the_remaining_columns(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 2, 0)
	defer term.Deinit()

	term.EraseChars(5)

	tm7ExpectGrid(t, term, []string{"ab.....h", "ijklmnop"})
}

// Zig: test "term51: eraseChars: count equal to the remaining columns"
func TestTm7_term51_eraseChars_count_equal_to_the_remaining_columns(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 2, 0)
	defer term.Deinit()

	term.EraseChars(6)

	tm7ExpectGrid(t, term, []string{"ab......", "ijklmnop"})
}

// Zig: test "term51: eraseChars: count one more than the remaining columns is clamped"
func TestTm7_term51_eraseChars_count_one_more_than_the_remaining_columns_is_clamped(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 2, 0)
	defer term.Deinit()

	term.EraseChars(7)

	tm7ExpectGrid(t, term, []string{"ab......", "ijklmnop"})
}

// Zig: test "term51: eraseChars: maximum count is clamped to the remaining columns"
func TestTm7_term51_eraseChars_maximum_count_is_clamped_to_the_remaining_columns(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 2, 0)
	defer term.Deinit()

	term.EraseChars(math.MaxUint)

	tm7ExpectGrid(t, term, []string{"ab......", "ijklmnop"})
	tm7ExpectCursor(t, term, 2, 0, false)
}

// Zig: test "term51: eraseChars: on the last column only that cell is erased"
func TestTm7_term51_eraseChars_on_the_last_column_only_that_cell_is_erased(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 7, 0)
	defer term.Deinit()

	term.EraseChars(1)
	tm7ExpectGrid(t, term, []string{"abcdefg.", "ijklmnop"})

	tm7Fill(t, term, tm7Alpha8[:])
	term.SetCursorPos(1, 8)
	term.EraseChars(math.MaxUint)
	tm7ExpectGrid(t, term, []string{"abcdefg.", "ijklmnop"})
}

// Zig: test "term51: eraseChars: the full width from the first column"
func TestTm7_term51_eraseChars_the_full_width_from_the_first_column(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 0)
	defer term.Deinit()

	term.EraseChars(8)

	tm7ExpectGrid(t, term, []string{"........", "ijklmnop"})
}

// Zig: test "term51: eraseChars: only the cursor row is touched"
func TestTm7_term51_eraseChars_only_the_cursor_row_is_touched(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 1, 1)
	defer term.Deinit()

	term.EraseChars(2)

	tm7ExpectGrid(t, term, []string{"abcdefgh", "i..lmnop"})
	tm7ExpectDirty(t, term, ".D")
}

// Zig: test "term51: eraseChars: pending wrap is cleared"
func TestTm7_term51_eraseChars_pending_wrap_is_cleared(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 7, 0)
	defer term.Deinit()

	term.Screens.Active.Cursor.PendingWrap = true
	term.EraseChars(1)

	tm7ExpectGrid(t, term, []string{"abcdefg.", "ijklmnop"})
	tm7ExpectCursor(t, term, 7, 0, false)
}

// Zig: test "term51: eraseChars: margins are ignored"
func TestTm7_term51_eraseChars_margins_are_ignored(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 3, 0)
	defer term.Deinit()

	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 5
	term.EraseChars(100)

	// The erase runs through the right edge of the screen, not the margin.
	tm7ExpectGrid(t, term, []string{"abc.....", "ijklmnop"})
}

// Zig: test "term51: eraseChars: works outside the margins"
func TestTm7_term51_eraseChars_works_outside_the_margins(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 1, 0)
	defer term.Deinit()

	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 5
	term.EraseChars(1)
	tm7ExpectGrid(t, term, []string{"a.cdefgh", "ijklmnop"})

	term.SetCursorPos(1, 7)
	term.EraseChars(2)
	tm7ExpectGrid(t, term, []string{"a.cdef..", "ijklmnop"})
}

// Zig: test "term51: eraseChars: blanks take the background color of the cursor style"
func TestTm7_term51_eraseChars_blanks_take_the_background_color_of_the_cursor_style(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 1, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: 4, G: 5, B: 6}}))
	term.EraseChars(2)

	tm7ExpectGrid(t, term, []string{"arrdefgh", "ijklmnop"})
	for x := uint(1); x < 3; x++ {
		cell := *tm7CellAt(t, term, x, 0)
		expectEqual(t, PageCellContentTagBgColorRgb, cell.ContentTag())
		expectEqual(t, PageCellRGB{R: 4, G: 5, B: 6}, cell.content().colorRGB())
		expectEqual(t, StyleDefaultID, cell.StyleID())
	}
}

// Zig: test "term51: eraseChars: blanks take a palette background color"
func TestTm7_term51_eraseChars_blanks_take_a_palette_background_color(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 6, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTag256Bg, X256Bg: 9}))
	term.EraseChars(5)

	tm7ExpectGrid(t, term, []string{"abcdefpp", "ijklmnop"})
	for x := uint(6); x < 8; x++ {
		cell := *tm7CellAt(t, term, x, 0)
		expectEqual(t, PageCellContentTagBgColorPalette, cell.ContentTag())
		expectEqual(t, uint8(9), cell.content().colorPalette())
	}
}

// Zig: test "term51: eraseChars: a cursor style without a background leaves blanks empty"
func TestTm7_term51_eraseChars_a_cursor_style_without_a_background_leaves_blanks_empty(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 1, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	term.EraseChars(2)

	tm7ExpectGrid(t, term, []string{"a..defgh", "ijklmnop"})
	expectEqual(t, StyleDefaultID, tm7CellAt(t, term, 1, 0).StyleID())
}

// Zig: test "term51: eraseChars: style reference counts follow the erased cells"
func TestTm7_term51_eraseChars_style_reference_counts_follow_the_erased_cells(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm7EightAt(t, 0, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	tm7PutStr(t, term, 0, 0, "abcdefgh")
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))
	page := tm7Page(term)
	bold := tm7CellAt(t, term, 0, 0).StyleID()

	term.SetCursorPos(1, 3)
	term.EraseChars(3)
	tm7ExpectGrid(t, term, []string{"ab...fgh", "ijklmnop"})
	expectEqual(t, SizeCellCountInt(5), page.styles.RefCount(bold))
	expect(t, tm7RowAt(t, term, 0).styled, "term51URowAt(&t, 0).styled")

	term.SetCursorPos(1, 1)
	term.EraseChars(8)
	tm7ExpectGrid(t, term, []string{"........", "ijklmnop"})
	expectEqual(t, uint(0), page.styles.Count())
	expect(t, !tm7RowAt(t, term, 0).styled, "!term51URowAt(&t, 0).styled")
}

// tm7ExpectPanic runs f, which must panic. A panic with the wireframe's
// "not implemented" value does not count: it only means the function under
// test does not exist yet. The setup is done outside f.
func tm7ExpectPanic(t *testing.T, f func()) {
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

// Go-only: panic test: insertBlanks: assert(self.screens.active.cursor.x > 0) when the cursor cell is a spacer tail in column 0
func TestTm7_panic_insertBlanks_spacer_tail_in_column_zero(t *testing.T) {
	testStdCapacity(t)
	term := tm7EightAt(t, 0, 0)
	defer term.Deinit()

	// A spacer tail can never be in column 0; force one.
	term.Screens.Active.Cursor.PageCell.setWide(PageCellWideSpacerTail)
	tm7ExpectPanic(t, func() { term.InsertBlanks(1) })
}

// Go-only: panic test: insertBlanks: assert(end_multi[1].wide == .spacer_tail) when the last shifted cell is a wide head without a spacer tail
func TestTm7_panic_insertBlanks_wide_head_without_spacer_tail(t *testing.T) {
	testStdCapacity(t)
	term := tm7EightAt(t, 0, 0)
	defer term.Deinit()

	// Cell 2 is a wide head but cell 3 is a narrow cell: with a count of 5
	// the last cell shifted is cell 2.
	tm7CellAt(t, term, 2, 0).setWide(PageCellWideWide)
	tm7ExpectPanic(t, func() { term.InsertBlanks(5) })
}
