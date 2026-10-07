// Tests translated one for one from src/terminal/Terminal.zig (Ghostty
// v1.3.1 plus the term51 additional tests), lines 24498 to 26263: the term51
// tests of eraseChars, eraseLine and eraseDisplay (the first half of the
// eraseDisplay protection tests). Each Go test carries the exact Zig title in
// a "// Zig: test" comment.
//
// Translation notes:
//   - The terminal variable is `term` (Zig's `t`), because `t` is the
//     *testing.T. `var t = try init(alloc, .{ .cols = 8, .rows = 2 })` is
//     tm8Init(t, 8, 2) (the Zig defaults of Terminal.Options come from
//     terminalOptionsDefault). `testing.allocator` and `defer t.deinit(alloc)`
//     become `defer term.Deinit()`.
//   - Every test calls testStdCapacity first (Zig's unit tests run with
//     grapheme_bytes 512).
//   - The Zig helpers term51U* are translated as tm8* helpers that take the
//     *testing.T: they fail the test where Zig's `try` would return the error.
//   - `page.styles.refCount(page.memory, id)` is page.styles.RefCount(id):
//     the memory argument is dropped.

package terminal

import (
	"testing"
)

// tm8Init is `init(alloc, .{ .cols = cols, .rows = rows })`.
func tm8Init(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) *Terminal {
	t.Helper()
	term, err := TerminalInit(terminalOptionsDefault(cols, rows))
	zigTry(t, err)
	return &term
}

// tm8Alpha8 is term51UAlpha8.
var tm8Alpha8 = []string{"abcdefgh", "ijklmnop"}

// tm8Alpha4 is term51UAlpha4.
var tm8Alpha4 = []string{"abcdefgh", "ijklmnop", "qrstuvwx", "yzABCDEF"}

// tm8Fill is term51UFill: print every non-'.' byte of every row at its own
// position. Leaves the cursor at the top-left with no pending wrap.
func tm8Fill(t *testing.T, term *Terminal, rows []string) {
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

// tm8CellAt is term51UCellAt.
func tm8CellAt(t *testing.T, term *Terminal, x uint, y uint) *PageCell {
	t.Helper()
	return zigUnwrap(t, term.Screens.Active.Pages.GetCell(Point{
		Tag:    PointTagActive,
		Active: PointCoordinate{X: SizeCellCountInt(x), Y: uint32(y)},
	})).Cell
}

// tm8RowAt is term51URowAt.
func tm8RowAt(t *testing.T, term *Terminal, y uint) *PageRow {
	t.Helper()
	return zigUnwrap(t, term.Screens.Active.Pages.GetCell(Point{
		Tag:    PointTagActive,
		Active: PointCoordinate{X: 0, Y: uint32(y)},
	})).Row
}

// tm8CellChar is term51UCellChar: one character per cell: '.' empty, the
// ASCII character for narrow text, '*' narrow grapheme cluster, '#' wide
// head, '>' spacer tail, '<' spacer head, 'p' palette background cell, 'r'
// RGB background cell.
func tm8CellChar(cell PageCell) uint8 {
	switch cell.wide {
	case PageCellWideWide:
		return '#'
	case PageCellWideSpacerTail:
		return '>'
	case PageCellWideSpacerHead:
		return '<'
	case PageCellWideNarrow:
		switch cell.content_tag {
		case PageCellContentTagCodepoint:
			if cell.content.codepoint == 0 {
				return '.'
			} else if cell.content.codepoint < 128 {
				return uint8(cell.content.codepoint)
			} else {
				return '?'
			}
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

// tm8ExpectGrid is term51UExpectGrid: compare the whole active area against
// the expected rows.
func tm8ExpectGrid(t *testing.T, term *Terminal, expected []string) {
	t.Helper()
	expectEqual(t, uint(term.Rows), uint(len(expected)))
	buf := make([]uint8, 64)
	for y, exp := range expected {
		for x := uint(0); x < uint(term.Cols); x++ {
			buf[x] = tm8CellChar(*tm8CellAt(t, term, x, uint(y)))
		}
		expectEqualStrings(t, exp, string(buf[:term.Cols]))
	}
}

// tm8ExpectDirty is term51UExpectDirty: compare the dirty flag of every
// active row: 'D' dirty, '.' clean.
func tm8ExpectDirty(t *testing.T, term *Terminal, expected string) {
	t.Helper()
	expectEqual(t, uint(term.Rows), uint(len(expected)))
	buf := make([]uint8, 64)
	for y := uint(0); y < uint(term.Rows); y++ {
		if tm8RowAt(t, term, y).dirty {
			buf[y] = 'D'
		} else {
			buf[y] = '.'
		}
	}
	expectEqualStrings(t, expected, string(buf[:term.Rows]))
}

// tm8ExpectWrap is term51UExpectWrap: compare the wrap (w),
// wrap_continuation (c), both (b) or neither (.) of every active row.
func tm8ExpectWrap(t *testing.T, term *Terminal, expected string) {
	t.Helper()
	expectEqual(t, uint(term.Rows), uint(len(expected)))
	buf := make([]uint8, 64)
	for y := uint(0); y < uint(term.Rows); y++ {
		row := tm8RowAt(t, term, y)
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
	expectEqualStrings(t, expected, string(buf[:term.Rows]))
}

// tm8SetWrap is term51USetWrap: set every row to the "wrap" state given by
// the characters of flags.
func tm8SetWrap(t *testing.T, term *Terminal, flags string) {
	t.Helper()
	for y := 0; y < len(flags); y++ {
		f := flags[y]
		row := tm8RowAt(t, term, uint(y))
		row.wrap = f == 'w' || f == 'b'
		row.wrap_continuation = f == 'c' || f == 'b'
	}
}

// tm8ExpectCursor is term51UExpectCursor.
func tm8ExpectCursor(t *testing.T, term *Terminal, x uint, y uint, pending_wrap bool) {
	t.Helper()
	expectEqual(t, x, uint(term.Screens.Active.Cursor.X))
	expectEqual(t, y, uint(term.Screens.Active.Cursor.Y))
	expectEqual(t, pending_wrap, term.Screens.Active.Cursor.PendingWrap)
}

// tm8Page is term51UPage: the page that holds the cursor; fine for terminals
// that fit one page.
func tm8Page(term *Terminal) *Page {
	return &term.Screens.Active.Cursor.PagePin.Node.data
}

// tm8PutStr is term51UPutStr: print str starting at the 0-indexed position
// (x, y).
func tm8PutStr(t *testing.T, term *Terminal, x uint, y uint, str string) {
	t.Helper()
	term.SetCursorPos(y+1, x+1)
	zigTry(t, term.PrintString([]uint8(str)))
}

// tm8PutWide is term51UPutWide: print one wide character (two cells) with its
// head at (x, y).
func tm8PutWide(t *testing.T, term *Terminal, x uint, y uint) {
	t.Helper()
	term.SetCursorPos(y+1, x+1)
	zigTry(t, term.Print(0x4E2D))
}

// tm8PutWideGrapheme is term51UPutWideGrapheme: print a family emoji ZWJ
// sequence, which is one wide cell with a grapheme.
func tm8PutWideGrapheme(t *testing.T, term *Terminal, x uint, y uint) {
	t.Helper()
	term.Modes.Set(ModesModeGraphemeCluster, true)
	term.SetCursorPos(y+1, x+1)
	zigTry(t, term.Print(0x1F468))
	zigTry(t, term.Print(0x200D))
	zigTry(t, term.Print(0x1F469))
}

// tm8PutGrapheme is term51UPutGrapheme: print an 'e' followed by a combining
// acute accent at (x, y): one narrow cell with a grapheme.
func tm8PutGrapheme(t *testing.T, term *Terminal, x uint, y uint) {
	t.Helper()
	term.SetCursorPos(y+1, x+1)
	zigTry(t, term.Print('e'))
	// The grapheme is attached directly: a zero-width character printed
	// after a character on the last column would attach to the cell before.
	zigTry(t, tm8Page(term).SetGraphemes(
		tm8RowAt(t, term, y),
		tm8CellAt(t, term, x, y),
		[]uint32{0x0301},
	))
}

// tm8EightAt is term51UEightAt: a two row, eight column terminal holding
// tm8Alpha8, with the cursor at (x, y), dirty flags cleared.
func tm8EightAt(t *testing.T, x uint, y uint) *Terminal {
	t.Helper()
	term := tm8Init(t, 8, 2)
	tm8Fill(t, term, tm8Alpha8)
	term.SetCursorPos(y+1, x+1)
	term.clearDirty()
	return term
}

// tm8WideRow is term51UWideRow: row "ab#>cdef" with the wide character at
// columns 2 and 3.
func tm8WideRow(t *testing.T, term *Terminal) {
	t.Helper()
	tm8PutStr(t, term, 0, 0, "ab")
	tm8PutWide(t, term, 2, 0)
	tm8PutStr(t, term, 4, 0, "cdef")
	tm8ExpectGrid(t, term, []string{"ab#>cdef", "........"})
	term.SetCursorPos(1, 1)
}

// tm8Protect is term51UProtect: reprint the letters of columns [from, to) of
// the first row of tm8Alpha8 with protection enabled in mode, then switch the
// protection off again (the screen remembers the mode).
func tm8Protect(t *testing.T, term *Terminal, mode ANSIProtectedMode, from uint, to uint) {
	t.Helper()
	term.SetProtectedMode(mode)
	for x := from; x < to; x++ {
		term.SetCursorPos(1, x+1)
		zigTry(t, term.Print(uint32(tm8Alpha8[0][x])))
	}
	term.SetProtectedMode(ANSIProtectedModeOff)
}

// tm8FourAt is term51UFourAt: an 8 column, 4 row terminal holding tm8Alpha4
// with the cursor at (x, y) and dirty flags cleared.
func tm8FourAt(t *testing.T, x uint, y uint) *Terminal {
	t.Helper()
	term := tm8Init(t, 8, 4)
	tm8Fill(t, term, tm8Alpha4)
	term.SetCursorPos(y+1, x+1)
	term.clearDirty()
	term.Flags.Dirty.Clear = false
	return term
}

// tm8HistoryRows is term51UHistoryRows: the number of rows of scrollback.
func tm8HistoryRows(term *Terminal) uint {
	return term.Screens.Active.Pages.total_rows - uint(term.Rows)
}

// tm8ExpectHistoryRow is term51UExpectHistoryRow: the text of a history row,
// one tm8CellChar per column.
func tm8ExpectHistoryRow(t *testing.T, term *Terminal, y uint, expected string) {
	t.Helper()
	buf := make([]uint8, 64)
	for x := uint(0); x < uint(term.Cols); x++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(Point{
			Tag:     PointTagHistory,
			History: PointCoordinate{X: SizeCellCountInt(x), Y: uint32(y)},
		}))
		buf[x] = tm8CellChar(*list_cell.Cell)
	}
	expectEqualStrings(t, expected, string(buf[:term.Cols]))
}

// tm8ScrollbackTerm is term51UScrollbackTerm: fill a three row terminal so
// that its history holds the first three of six numbered lines.
func tm8ScrollbackTerm(t *testing.T) *Terminal {
	t.Helper()
	term := tm8Init(t, 8, 3)
	lines := []string{"line0", "line1", "line2", "line3", "line4", "line5"}
	for i, line := range lines {
		if i > 0 {
			term.CarriageReturn()
			zigTry(t, term.Linefeed())
		}
		zigTry(t, term.PrintString([]uint8(line)))
	}
	return term
}

// tm8ProtectAt is term51UProtectAt: print the letter of tm8Alpha4 at (x, y)
// with protection mode, then switch protection off.
func tm8ProtectAt(t *testing.T, term *Terminal, mode ANSIProtectedMode, x uint, y uint) {
	t.Helper()
	term.SetProtectedMode(mode)
	term.SetCursorPos(y+1, x+1)
	zigTry(t, term.Print(uint32(tm8Alpha4[y][x])))
	term.SetProtectedMode(ANSIProtectedModeOff)
}

// tm8EraseLineProtected is term51UEraseLineProtected: erase line with
// protected cells at columns 1, 2, 4 and 5 of the first row (cursor at column
// 3). The cells are printed with first protection; if then is set it becomes
// the most recent mode afterwards.
func tm8EraseLineProtected(
	t *testing.T,
	mode CSIEraseLine,
	first ANSIProtectedMode,
	then Optional[ANSIProtectedMode],
	req bool,
	expected string,
) {
	t.Helper()
	term := tm8EightAt(t, 3, 0)
	defer term.Deinit()

	tm8Protect(t, term, first, 1, 3)
	tm8Protect(t, term, first, 4, 6)
	if then.Valid {
		m := then.Value
		term.SetProtectedMode(m)
		term.SetProtectedMode(ANSIProtectedModeOff)
	}
	term.SetCursorPos(1, 4)
	term.EraseLine(mode, req)

	tm8ExpectGrid(t, term, []string{expected, "ijklmnop"})
	tm8ExpectCursor(t, term, 3, 0, false)
}

// tm8EraseDisplayProtected is term51UEraseDisplayProtected: erase display
// with protected cells at (1,0), (2,1), (5,1), (3,2) and (6,3) of tm8Alpha4,
// cursor at (3,1). The cells are protected with first; if then is set it
// becomes the most recent mode afterwards.
func tm8EraseDisplayProtected(
	t *testing.T,
	mode CSIEraseDisplay,
	first ANSIProtectedMode,
	then Optional[ANSIProtectedMode],
	req bool,
	expected []string,
) {
	t.Helper()
	term := tm8FourAt(t, 3, 1)
	defer term.Deinit()

	tm8ProtectAt(t, term, first, 1, 0)
	tm8ProtectAt(t, term, first, 2, 1)
	tm8ProtectAt(t, term, first, 5, 1)
	tm8ProtectAt(t, term, first, 3, 2)
	tm8ProtectAt(t, term, first, 6, 3)
	if then.Valid {
		m := then.Value
		term.SetProtectedMode(m)
		term.SetProtectedMode(ANSIProtectedModeOff)
	}
	term.SetCursorPos(2, 4)
	term.EraseDisplay(mode, req)

	tm8ExpectGrid(t, term, expected)
	tm8ExpectCursor(t, term, 3, 1, false)
}

var (
	tm8BelowRespected    = []string{"abcdefgh", "ijk..n..", "...t....", "......E."}
	tm8BelowIgnored      = []string{"abcdefgh", "ijk.....", "........", "........"}
	tm8AboveRespected    = []string{".b......", "..k.mnop", "qrstuvwx", "yzABCDEF"}
	tm8AboveIgnored      = []string{"........", "....mnop", "qrstuvwx", "yzABCDEF"}
	tm8CompleteRespected = []string{".b......", "..k..n..", "...t....", "......E."}
	tm8CompleteIgnored   = []string{"........", "........", "........", "........"}
)

// ---- eraseChars ----

// tm8DirectBg is `.{ .direct_color_bg = .{ .r = r, .g = g, .b = b } }`.
func tm8DirectBg(r uint8, g uint8, b uint8) SGRAttribute {
	return SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: r, G: g, B: b}}
}

// tm8Bg256 is `.{ .@"256_bg" = n }`.
func tm8Bg256(n uint8) SGRAttribute {
	return SGRAttribute{Tag: SGRAttributeTag256Bg, X256Bg: n}
}

// tm8Bold is `.{ .bold = {} }`.
func tm8Bold() SGRAttribute { return SGRAttribute{Tag: SGRAttributeTagBold} }

// tm8Unset is `.{ .unset = {} }`.
func tm8Unset() SGRAttribute { return SGRAttribute{Tag: SGRAttributeTagUnset} }

// tm8StartLink is `startHyperlink("http://a.test", null)`.
func tm8StartLink(t *testing.T, term *Terminal) {
	t.Helper()
	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://a.test"), None[[]uint8]()))
}

// Zig: test "term51: eraseChars: graphemes are released"
func TestTm8_eraseChars_graphemes_are_released(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 0, 0)
	defer term.Deinit()

	tm8PutGrapheme(t, term, 2, 0)
	tm8PutGrapheme(t, term, 6, 0)
	page := tm8Page(term)
	expectEqual(t, uint(2), page.GraphemeCount())

	term.SetCursorPos(1, 2)
	term.EraseChars(3)
	tm8ExpectGrid(t, term, []string{"a...ef*h", "ijklmnop"})
	expectEqual(t, uint(1), page.GraphemeCount())
	expect(t, tm8RowAt(t, term, 0).grapheme, "term51URowAt(&t, 0).grapheme")

	term.SetCursorPos(1, 5)
	term.EraseChars(100)
	tm8ExpectGrid(t, term, []string{"a.......", "ijklmnop"})
	expectEqual(t, uint(0), page.GraphemeCount())
	expect(t, !tm8RowAt(t, term, 0).grapheme, "!term51URowAt(&t, 0).grapheme")
}

// Zig: test "term51: eraseChars: hyperlinks are released"
func TestTm8_eraseChars_hyperlinks_are_released(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 0, 0)
	defer term.Deinit()

	tm8StartLink(t, term)
	tm8PutStr(t, term, 2, 0, "cdef")
	term.Screens.Active.EndHyperlink()
	page := tm8Page(term)
	expectEqual(t, uint(4), page.HyperlinkCount())

	term.SetCursorPos(1, 4)
	term.EraseChars(2)
	tm8ExpectGrid(t, term, []string{"abc..fgh", "ijklmnop"})
	expectEqual(t, uint(2), page.HyperlinkCount())
	for x := uint(0); x < 8; x++ {
		expectEqual(t, x == 2 || x == 5, tm8CellAt(t, term, x, 0).hyperlink)
	}
	expect(t, tm8RowAt(t, term, 0).hyperlink, "term51URowAt(&t, 0).hyperlink")

	term.SetCursorPos(1, 1)
	term.EraseChars(8)
	expectEqual(t, uint(0), page.HyperlinkCount())
	expectEqual(t, uint(0), page.hyperlink_set.Count())
	expect(t, !tm8RowAt(t, term, 0).hyperlink, "!term51URowAt(&t, 0).hyperlink")
}

// Zig: test "term51: eraseChars: the soft wrap of the row is reset"
func TestTm8_eraseChars_the_soft_wrap_of_the_row_is_reset(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 3, 0)
	defer term.Deinit()

	tm8SetWrap(t, term, "wc")
	term.EraseChars(1)

	tm8ExpectGrid(t, term, []string{"abc.efgh", "ijklmnop"})
	tm8ExpectWrap(t, term, "..")
}

// Zig: test "term51: eraseChars: only the wrap of the cursor row is reset"
func TestTm8_eraseChars_only_the_wrap_of_the_cursor_row_is_reset(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 3, 1)
	defer term.Deinit()

	tm8SetWrap(t, term, "wc")
	term.EraseChars(1)

	tm8ExpectWrap(t, term, "wc")
}

// Zig: test "term51: eraseChars: erasing on a wrapped row clears the spacer head at its end"
func TestTm8_eraseChars_erasing_on_a_wrapped_row_clears_the_spacer_head_at_its_end(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 2)
	defer term.Deinit()

	tm8PutStr(t, term, 0, 0, "abcdefg")
	zigTry(t, term.Print(0x4E2D))
	tm8ExpectGrid(t, term, []string{"abcdefg<", "#>......"})

	term.SetCursorPos(1, 5)
	term.EraseChars(2)

	tm8ExpectGrid(t, term, []string{"abcd..g.", "#>......"})
	tm8ExpectWrap(t, term, "..")
}

// Zig: test "term51: eraseChars: erasing the end of a wrapped row clears the spacer head"
func TestTm8_eraseChars_erasing_the_end_of_a_wrapped_row_clears_the_spacer_head(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 2)
	defer term.Deinit()

	tm8PutStr(t, term, 0, 0, "abcdefg")
	zigTry(t, term.Print(0x4E2D))

	term.SetCursorPos(1, 8)
	term.EraseChars(1)

	tm8ExpectGrid(t, term, []string{"abcdefg.", "#>......"})
	tm8ExpectWrap(t, term, "..")
}

// Zig: test "term51: eraseChars: cursor on a spacer tail erases the wide character"
func TestTm8_eraseChars_cursor_on_a_spacer_tail_erases_the_wide_character(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 2)
	defer term.Deinit()

	tm8WideRow(t, term)
	term.SetCursorPos(1, 4)
	term.EraseChars(1)

	tm8ExpectGrid(t, term, []string{"ab..cdef", "........"})
}

// Zig: test "term51: eraseChars: cursor on a wide head erases the whole character"
func TestTm8_eraseChars_cursor_on_a_wide_head_erases_the_whole_character(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 2)
	defer term.Deinit()

	tm8WideRow(t, term)
	term.SetCursorPos(1, 3)
	term.EraseChars(1)

	tm8ExpectGrid(t, term, []string{"ab..cdef", "........"})
}

// Zig: test "term51: eraseChars: a count that ends on a wide head includes its tail"
func TestTm8_eraseChars_a_count_that_ends_on_a_wide_head_includes_its_tail(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 2)
	defer term.Deinit()

	tm8PutStr(t, term, 0, 0, "abcdef")
	tm8PutWide(t, term, 6, 0)
	tm8ExpectGrid(t, term, []string{"abcdef#>", "........"})

	term.SetCursorPos(1, 5)
	term.EraseChars(3)
	tm8ExpectGrid(t, term, []string{"abcd....", "........"})

	// Also when the wide character is not at the end of the row.
	tm8Fill(t, term, []string{"........", "........"})
	tm8WideRow(t, term)
	term.SetCursorPos(1, 2)
	term.EraseChars(2)
	tm8ExpectGrid(t, term, []string{"a...cdef", "........"})
}

// Zig: test "term51: eraseChars: a count that ends on the tail of a wide character erases it whole"
func TestTm8_eraseChars_a_count_that_ends_on_the_tail_of_a_wide_character_erases_it_whole(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 2)
	defer term.Deinit()

	tm8WideRow(t, term)
	term.SetCursorPos(1, 2)
	term.EraseChars(3)

	tm8ExpectGrid(t, term, []string{"a...cdef", "........"})
}

// Zig: test "term51: eraseChars: a count that stops before a wide character leaves it"
func TestTm8_eraseChars_a_count_that_stops_before_a_wide_character_leaves_it(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 2)
	defer term.Deinit()

	tm8PutStr(t, term, 0, 0, "abcd")
	tm8PutWide(t, term, 4, 0)
	tm8PutStr(t, term, 6, 0, "gh")

	term.SetCursorPos(1, 3)
	term.EraseChars(2)

	tm8ExpectGrid(t, term, []string{"ab..#>gh", "........"})
}

// Zig: test "term51: eraseChars: a wide character with a grapheme that is split releases the grapheme"
func TestTm8_eraseChars_a_wide_character_with_a_grapheme_that_is_split_releases_the_grapheme(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 2)
	defer term.Deinit()

	tm8PutStr(t, term, 0, 0, "ab")
	tm8PutWideGrapheme(t, term, 2, 0)
	tm8PutStr(t, term, 4, 0, "cdef")
	page := tm8Page(term)
	expectEqual(t, uint(1), page.GraphemeCount())

	term.SetCursorPos(1, 4)
	term.EraseChars(1)

	tm8ExpectGrid(t, term, []string{"ab..cdef", "........"})
	expectEqual(t, uint(0), page.GraphemeCount())
	expect(t, !tm8RowAt(t, term, 0).grapheme, "!term51URowAt(&t, 0).grapheme")
}

// Zig: test "term51: eraseChars: a wide character at the start of a continuation row clears the spacer head above"
func TestTm8_eraseChars_a_wide_character_at_the_start_of_a_continuation_row_clears_the_spacer_head_above(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 2)
	defer term.Deinit()

	tm8PutStr(t, term, 0, 0, "abcdefg")
	zigTry(t, term.Print(0x4E2D))
	term.SetCursorPos(2, 1)
	term.clearDirty()

	term.EraseChars(1)

	tm8ExpectGrid(t, term, []string{"abcdefg.", "........"})
	tm8ExpectDirty(t, term, ".D")
	tm8ExpectWrap(t, term, "wc")
}

// Zig: test "term51: eraseChars: ISO protection keeps protected cells"
func TestTm8_eraseChars_ISO_protection_keeps_protected_cells(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 1, 0)
	defer term.Deinit()

	tm8Protect(t, term, ANSIProtectedModeIso, 3, 5)
	term.SetCursorPos(1, 2)
	term.EraseChars(5)

	tm8ExpectGrid(t, term, []string{"a..de.gh", "ijklmnop"})
	expect(t, tm8CellAt(t, term, 3, 0).protected, "term51UCellAt(&t, 3, 0).protected")
	expect(t, !tm8CellAt(t, term, 1, 0).protected, "!term51UCellAt(&t, 1, 0).protected")
}

// Zig: test "term51: eraseChars: protection ends are handled at both ends of the range"
func TestTm8_eraseChars_protection_ends_are_handled_at_both_ends_of_the_range(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 1, 0)
	defer term.Deinit()

	tm8Protect(t, term, ANSIProtectedModeIso, 1, 2)
	tm8Protect(t, term, ANSIProtectedModeIso, 4, 5)
	term.SetCursorPos(1, 2)
	term.EraseChars(4)
	tm8ExpectGrid(t, term, []string{"ab..efgh", "ijklmnop"})
	expect(t, tm8CellAt(t, term, 1, 0).protected, "term51UCellAt(&t, 1, 0).protected")
	expect(t, tm8CellAt(t, term, 4, 0).protected, "term51UCellAt(&t, 4, 0).protected")
	expect(t, !tm8CellAt(t, term, 2, 0).protected, "!term51UCellAt(&t, 2, 0).protected")
}

// Zig: test "term51: eraseChars: a fully protected range under ISO protection is untouched"
func TestTm8_eraseChars_a_fully_protected_range_under_ISO_protection_is_untouched(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 1, 0)
	defer term.Deinit()

	tm8Protect(t, term, ANSIProtectedModeIso, 1, 4)
	term.SetCursorPos(1, 2)
	term.EraseChars(3)

	tm8ExpectGrid(t, term, []string{"abcdefgh", "ijklmnop"})
	tm8ExpectCursor(t, term, 1, 0, false)
	tm8ExpectDirty(t, term, "D.")
}

// Zig: test "term51: eraseChars: ISO protection stays in force after it is switched off"
func TestTm8_eraseChars_ISO_protection_stays_in_force_after_it_is_switched_off(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 0, 0)
	defer term.Deinit()

	tm8Protect(t, term, ANSIProtectedModeIso, 3, 5)
	expect(t, !term.Screens.Active.Cursor.Protected, "!t.screens.active.cursor.protected")
	expectEqual(t, ANSIProtectedModeIso, term.Screens.Active.ProtectedMode)
	term.SetCursorPos(1, 1)
	term.EraseChars(8)

	tm8ExpectGrid(t, term, []string{"...de...", "ijklmnop"})
}

// Zig: test "term51: eraseChars: DEC protection is ignored"
func TestTm8_eraseChars_DEC_protection_is_ignored(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 0, 0)
	defer term.Deinit()

	tm8Protect(t, term, ANSIProtectedModeDec, 3, 5)
	term.SetCursorPos(1, 1)
	term.EraseChars(8)

	tm8ExpectGrid(t, term, []string{"........", "ijklmnop"})
	for x := uint(0); x < 8; x++ {
		expect(t, !tm8CellAt(t, term, x, 0).protected, "!term51UCellAt(&t, x, 0).protected")
	}
}

// Zig: test "term51: eraseChars: DEC protection is ignored while it is switched on"
func TestTm8_eraseChars_DEC_protection_is_ignored_while_it_is_switched_on(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 0, 0)
	defer term.Deinit()

	tm8Protect(t, term, ANSIProtectedModeDec, 3, 5)
	term.SetProtectedMode(ANSIProtectedModeDec)
	term.SetCursorPos(1, 1)
	term.EraseChars(8)

	tm8ExpectGrid(t, term, []string{"........", "ijklmnop"})
}

// Zig: test "term51: eraseChars: the most recent protection mode decides"
func TestTm8_eraseChars_the_most_recent_protection_mode_decides(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 0, 0)
	defer term.Deinit()

	// ISO first, then DEC most recently: protection is ignored.
	tm8Protect(t, term, ANSIProtectedModeIso, 3, 5)
	term.SetProtectedMode(ANSIProtectedModeDec)
	term.SetProtectedMode(ANSIProtectedModeOff)
	term.SetCursorPos(1, 1)
	term.EraseChars(8)
	tm8ExpectGrid(t, term, []string{"........", "ijklmnop"})

	// DEC first, then ISO most recently: protection is respected.
	tm8Fill(t, term, tm8Alpha8)
	tm8Protect(t, term, ANSIProtectedModeDec, 3, 5)
	term.SetProtectedMode(ANSIProtectedModeIso)
	term.SetProtectedMode(ANSIProtectedModeOff)
	term.SetCursorPos(1, 1)
	term.EraseChars(8)
	tm8ExpectGrid(t, term, []string{"...de...", "ijklmnop"})
}

// Zig: test "term51: eraseChars: a protected wide character is split by the boundary handling regardless of protection"
func TestTm8_eraseChars_a_protected_wide_character_is_split_by_the_boundary_handling_regardless_of_protection(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 2)
	defer term.Deinit()

	tm8PutStr(t, term, 0, 0, "ab")
	term.SetProtectedMode(ANSIProtectedModeIso)
	tm8PutWide(t, term, 2, 0)
	term.SetProtectedMode(ANSIProtectedModeOff)
	tm8PutStr(t, term, 4, 0, "cdef")
	expect(t, tm8CellAt(t, term, 2, 0).protected, "term51UCellAt(&t, 2, 0).protected")
	expect(t, tm8CellAt(t, term, 3, 0).protected, "term51UCellAt(&t, 3, 0).protected")

	term.SetCursorPos(1, 4)
	term.EraseChars(1)

	// The protected character is cleared although ISO protection is on.
	tm8ExpectGrid(t, term, []string{"ab..cdef", "........"})
	expect(t, !tm8CellAt(t, term, 2, 0).protected, "!term51UCellAt(&t, 2, 0).protected")
}

// ---- eraseLine ----

// Zig: test "term51: eraseLine: right from the middle"
func TestTm8_eraseLine_right_from_the_middle(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 3, 0)
	defer term.Deinit()

	term.EraseLine(CSIEraseLineRight, false)

	tm8ExpectGrid(t, term, []string{"abc.....", "ijklmnop"})
	tm8ExpectCursor(t, term, 3, 0, false)
	tm8ExpectDirty(t, term, "D.")
}

// Zig: test "term51: eraseLine: right from the first column erases the whole row"
func TestTm8_eraseLine_right_from_the_first_column_erases_the_whole_row(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 0, 1)
	defer term.Deinit()

	term.EraseLine(CSIEraseLineRight, false)

	tm8ExpectGrid(t, term, []string{"abcdefgh", "........"})
	tm8ExpectDirty(t, term, ".D")
}

// Zig: test "term51: eraseLine: right on the last column erases only that cell"
func TestTm8_eraseLine_right_on_the_last_column_erases_only_that_cell(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 7, 0)
	defer term.Deinit()

	term.EraseLine(CSIEraseLineRight, false)

	tm8ExpectGrid(t, term, []string{"abcdefg.", "ijklmnop"})
}

// Zig: test "term51: eraseLine: right clears pending wrap"
func TestTm8_eraseLine_right_clears_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 7, 0)
	defer term.Deinit()

	term.Screens.Active.Cursor.PendingWrap = true
	term.EraseLine(CSIEraseLineRight, false)

	tm8ExpectGrid(t, term, []string{"abcdefg.", "ijklmnop"})
	tm8ExpectCursor(t, term, 7, 0, false)
}

// Zig: test "term51: eraseLine: right resets the soft wrap of the row"
func TestTm8_eraseLine_right_resets_the_soft_wrap_of_the_row(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 3, 0)
	defer term.Deinit()

	tm8SetWrap(t, term, "wc")
	term.EraseLine(CSIEraseLineRight, false)

	tm8ExpectWrap(t, term, "..")
}

// Zig: test "term51: eraseLine: right clears a spacer head at the end of a wrapped row"
func TestTm8_eraseLine_right_clears_a_spacer_head_at_the_end_of_a_wrapped_row(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 2)
	defer term.Deinit()

	tm8PutStr(t, term, 0, 0, "abcdefg")
	zigTry(t, term.Print(0x4E2D))
	term.SetCursorPos(1, 3)
	term.EraseLine(CSIEraseLineRight, false)

	tm8ExpectGrid(t, term, []string{"ab......", "#>......"})
	tm8ExpectWrap(t, term, "..")
}

// Zig: test "term51: eraseLine: right erases the head of a wide character when on its tail"
func TestTm8_eraseLine_right_erases_the_head_of_a_wide_character_when_on_its_tail(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 2)
	defer term.Deinit()

	tm8WideRow(t, term)
	term.SetCursorPos(1, 4)
	term.EraseLine(CSIEraseLineRight, false)

	tm8ExpectGrid(t, term, []string{"ab......", "........"})
}

// Zig: test "term51: eraseLine: right erases the whole wide character when on its head"
func TestTm8_eraseLine_right_erases_the_whole_wide_character_when_on_its_head(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 2)
	defer term.Deinit()

	tm8WideRow(t, term)
	term.SetCursorPos(1, 3)
	term.EraseLine(CSIEraseLineRight, false)

	tm8ExpectGrid(t, term, []string{"ab......", "........"})
}

// Zig: test "term51: eraseLine: right takes the background color of the cursor style"
func TestTm8_eraseLine_right_takes_the_background_color_of_the_cursor_style(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 5, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(tm8DirectBg(4, 5, 6)))
	term.EraseLine(CSIEraseLineRight, false)

	tm8ExpectGrid(t, term, []string{"abcderrr", "ijklmnop"})
	for x := uint(5); x < 8; x++ {
		cell := *tm8CellAt(t, term, x, 0)
		expectEqual(t, PageCellContentTagBgColorRgb, cell.content_tag)
		expectEqual(t, PageCellRGB{R: 4, G: 5, B: 6}, cell.content.color_rgb)
	}
}

// Zig: test "term51: eraseLine: right takes a palette background color"
func TestTm8_eraseLine_right_takes_a_palette_background_color(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 6, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(tm8Bg256(9)))
	term.EraseLine(CSIEraseLineRight, false)

	tm8ExpectGrid(t, term, []string{"abcdefpp", "ijklmnop"})
	expectEqual(t, uint8(9), tm8CellAt(t, term, 6, 0).content.color_palette)
}

// Zig: test "term51: eraseLine: right with a cursor style without a background leaves blanks empty"
func TestTm8_eraseLine_right_with_a_cursor_style_without_a_background_leaves_blanks_empty(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 6, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(tm8Bold()))
	term.EraseLine(CSIEraseLineRight, false)

	tm8ExpectGrid(t, term, []string{"abcdef..", "ijklmnop"})
	expectEqual(t, StyleDefaultID, tm8CellAt(t, term, 6, 0).style_id)
}

// Zig: test "term51: eraseLine: right releases styles, graphemes and hyperlinks"
func TestTm8_eraseLine_right_releases_styles_graphemes_and_hyperlinks(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 0, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(tm8Bold()))
	tm8StartLink(t, term)
	tm8PutStr(t, term, 4, 0, "efgh")
	zigTry(t, term.SetAttribute(tm8Unset()))
	term.Screens.Active.EndHyperlink()
	tm8PutGrapheme(t, term, 5, 0)
	page := tm8Page(term)
	// Printing the grapheme over column 5 replaced its styled, linked cell.
	bold := tm8CellAt(t, term, 4, 0).style_id
	expectEqual(t, uint(3), uint(page.styles.RefCount(bold)))
	expectEqual(t, uint(3), page.HyperlinkCount())
	expectEqual(t, uint(1), page.GraphemeCount())

	term.SetCursorPos(1, 7)
	term.EraseLine(CSIEraseLineRight, false)
	tm8ExpectGrid(t, term, []string{"abcde*..", "ijklmnop"})
	expectEqual(t, uint(1), uint(page.styles.RefCount(bold)))
	expectEqual(t, uint(1), page.HyperlinkCount())
	expectEqual(t, uint(1), page.GraphemeCount())
	// The row flags are still needed by the remaining cells.
	expect(t, tm8RowAt(t, term, 0).styled, "term51URowAt(&t, 0).styled")
	expect(t, tm8RowAt(t, term, 0).hyperlink, "term51URowAt(&t, 0).hyperlink")
	expect(t, tm8RowAt(t, term, 0).grapheme, "term51URowAt(&t, 0).grapheme")

	// Erasing from the first column clears everything and the row flags.
	term.SetCursorPos(1, 1)
	term.EraseLine(CSIEraseLineRight, false)
	tm8ExpectGrid(t, term, []string{"........", "ijklmnop"})
	expectEqual(t, uint(0), page.styles.Count())
	expectEqual(t, uint(0), page.HyperlinkCount())
	expectEqual(t, uint(0), page.hyperlink_set.Count())
	expectEqual(t, uint(0), page.GraphemeCount())
	expect(t, !tm8RowAt(t, term, 0).styled, "!term51URowAt(&t, 0).styled")
	expect(t, !tm8RowAt(t, term, 0).hyperlink, "!term51URowAt(&t, 0).hyperlink")
	expect(t, !tm8RowAt(t, term, 0).grapheme, "!term51URowAt(&t, 0).grapheme")
}

// Zig: test "term51: eraseLine: left from the middle includes the cursor cell"
func TestTm8_eraseLine_left_from_the_middle_includes_the_cursor_cell(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 3, 0)
	defer term.Deinit()

	term.Screens.Active.Cursor.PendingWrap = true
	term.EraseLine(CSIEraseLineLeft, false)

	tm8ExpectGrid(t, term, []string{"....efgh", "ijklmnop"})
	tm8ExpectCursor(t, term, 3, 0, false)
	tm8ExpectDirty(t, term, "D.")
}

// Zig: test "term51: eraseLine: left on the first column erases only that cell"
func TestTm8_eraseLine_left_on_the_first_column_erases_only_that_cell(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 0, 1)
	defer term.Deinit()

	term.EraseLine(CSIEraseLineLeft, false)

	tm8ExpectGrid(t, term, []string{"abcdefgh", ".jklmnop"})
	tm8ExpectDirty(t, term, ".D")
}

// Zig: test "term51: eraseLine: left on the last column erases the whole row"
func TestTm8_eraseLine_left_on_the_last_column_erases_the_whole_row(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 7, 0)
	defer term.Deinit()

	term.Screens.Active.Cursor.PendingWrap = true
	term.EraseLine(CSIEraseLineLeft, false)

	tm8ExpectGrid(t, term, []string{"........", "ijklmnop"})
	tm8ExpectCursor(t, term, 7, 0, false)
}

// Zig: test "term51: eraseLine: left keeps the soft wrap of the row and its spacer head"
func TestTm8_eraseLine_left_keeps_the_soft_wrap_of_the_row_and_its_spacer_head(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 2)
	defer term.Deinit()

	tm8PutStr(t, term, 0, 0, "abcdefg")
	zigTry(t, term.Print(0x4E2D))
	term.SetCursorPos(1, 3)
	term.EraseLine(CSIEraseLineLeft, false)

	tm8ExpectGrid(t, term, []string{"...defg<", "#>......"})
	tm8ExpectWrap(t, term, "wc")
}

// Zig: test "term51: eraseLine: left erases the tail of a wide character when on its head"
func TestTm8_eraseLine_left_erases_the_tail_of_a_wide_character_when_on_its_head(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 2)
	defer term.Deinit()

	tm8WideRow(t, term)
	term.SetCursorPos(1, 3)
	term.EraseLine(CSIEraseLineLeft, false)

	tm8ExpectGrid(t, term, []string{"....cdef", "........"})
}

// Zig: test "term51: eraseLine: left includes the head of a wide character when on its tail"
func TestTm8_eraseLine_left_includes_the_head_of_a_wide_character_when_on_its_tail(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 2)
	defer term.Deinit()

	tm8WideRow(t, term)
	term.SetCursorPos(1, 4)
	term.EraseLine(CSIEraseLineLeft, false)

	tm8ExpectGrid(t, term, []string{"....cdef", "........"})
}

// Zig: test "term51: eraseLine: left leaves a spacer head on the row above when it erases the wide character below"
func TestTm8_eraseLine_left_leaves_a_spacer_head_on_the_row_above_when_it_erases_the_wide_character_below(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 2)
	defer term.Deinit()

	tm8PutStr(t, term, 0, 0, "abcdefg")
	zigTry(t, term.Print(0x4E2D))
	term.SetCursorPos(2, 2)
	term.EraseLine(CSIEraseLineLeft, false)

	tm8ExpectGrid(t, term, []string{"abcdefg<", "........"})
	tm8ExpectWrap(t, term, "wc")
}

// Zig: test "term51: eraseLine: left takes the background color of the cursor style"
func TestTm8_eraseLine_left_takes_the_background_color_of_the_cursor_style(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 2, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(tm8DirectBg(4, 5, 6)))
	term.EraseLine(CSIEraseLineLeft, false)

	tm8ExpectGrid(t, term, []string{"rrrdefgh", "ijklmnop"})
	cell := *tm8CellAt(t, term, 0, 0)
	expectEqual(t, PageCellContentTagBgColorRgb, cell.content_tag)
	expectEqual(t, PageCellRGB{R: 4, G: 5, B: 6}, cell.content.color_rgb)
}

// Zig: test "term51: eraseLine: left takes a palette background color"
func TestTm8_eraseLine_left_takes_a_palette_background_color(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 1, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(tm8Bg256(9)))
	term.EraseLine(CSIEraseLineLeft, false)

	tm8ExpectGrid(t, term, []string{"ppcdefgh", "ijklmnop"})
	expectEqual(t, uint8(9), tm8CellAt(t, term, 1, 0).content.color_palette)
}

// Zig: test "term51: eraseLine: left releases styles, graphemes and hyperlinks"
func TestTm8_eraseLine_left_releases_styles_graphemes_and_hyperlinks(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 0, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(tm8Bold()))
	tm8StartLink(t, term)
	tm8PutStr(t, term, 0, 0, "abcd")
	zigTry(t, term.SetAttribute(tm8Unset()))
	term.Screens.Active.EndHyperlink()
	tm8PutGrapheme(t, term, 1, 0)
	page := tm8Page(term)
	bold := tm8CellAt(t, term, 0, 0).style_id
	expectEqual(t, uint(3), uint(page.styles.RefCount(bold)))
	expectEqual(t, uint(3), page.HyperlinkCount())

	term.SetCursorPos(1, 3)
	term.EraseLine(CSIEraseLineLeft, false)
	tm8ExpectGrid(t, term, []string{"...defgh", "ijklmnop"})
	expectEqual(t, uint(1), uint(page.styles.RefCount(bold)))
	expectEqual(t, uint(1), page.HyperlinkCount())
	expectEqual(t, uint(0), page.GraphemeCount())
	expect(t, !tm8RowAt(t, term, 0).grapheme, "!term51URowAt(&t, 0).grapheme")
	expect(t, tm8RowAt(t, term, 0).styled, "term51URowAt(&t, 0).styled")
	expect(t, tm8RowAt(t, term, 0).hyperlink, "term51URowAt(&t, 0).hyperlink")

	term.SetCursorPos(1, 4)
	term.EraseLine(CSIEraseLineLeft, false)
	expectEqual(t, uint(0), page.styles.Count())
	expectEqual(t, uint(0), page.HyperlinkCount())
	expectEqual(t, uint(0), page.hyperlink_set.Count())
}

// Zig: test "term51: eraseLine: complete erases the whole row and keeps the cursor"
func TestTm8_eraseLine_complete_erases_the_whole_row_and_keeps_the_cursor(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 3, 1)
	defer term.Deinit()

	term.Screens.Active.Cursor.PendingWrap = true
	term.EraseLine(CSIEraseLineComplete, false)

	tm8ExpectGrid(t, term, []string{"abcdefgh", "........"})
	tm8ExpectCursor(t, term, 3, 1, false)
	tm8ExpectDirty(t, term, ".D")
}

// Zig: test "term51: eraseLine: complete keeps the soft wrap of the row and its spacer head is cleared"
func TestTm8_eraseLine_complete_keeps_the_soft_wrap_of_the_row_and_its_spacer_head_is_cleared(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 2)
	defer term.Deinit()

	tm8PutStr(t, term, 0, 0, "abcdefg")
	zigTry(t, term.Print(0x4E2D))
	term.SetCursorPos(1, 3)
	term.EraseLine(CSIEraseLineComplete, false)

	// The wrap flags stay although the spacer head is gone.
	tm8ExpectGrid(t, term, []string{"........", "#>......"})
	tm8ExpectWrap(t, term, "wc")
}

// Zig: test "term51: eraseLine: complete erases a wide character"
func TestTm8_eraseLine_complete_erases_a_wide_character(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 2)
	defer term.Deinit()

	tm8WideRow(t, term)
	term.SetCursorPos(1, 4)
	term.EraseLine(CSIEraseLineComplete, false)

	tm8ExpectGrid(t, term, []string{"........", "........"})
}

// Zig: test "term51: eraseLine: complete takes the background color of the cursor style"
func TestTm8_eraseLine_complete_takes_the_background_color_of_the_cursor_style(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 2, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(tm8DirectBg(4, 5, 6)))
	term.EraseLine(CSIEraseLineComplete, false)

	tm8ExpectGrid(t, term, []string{"rrrrrrrr", "ijklmnop"})
	for x := uint(0); x < 8; x++ {
		cell := *tm8CellAt(t, term, x, 0)
		expectEqual(t, PageCellContentTagBgColorRgb, cell.content_tag)
		expectEqual(t, PageCellRGB{R: 4, G: 5, B: 6}, cell.content.color_rgb)
	}
}

// Zig: test "term51: eraseLine: complete takes a palette background color"
func TestTm8_eraseLine_complete_takes_a_palette_background_color(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 2, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(tm8Bg256(9)))
	term.EraseLine(CSIEraseLineComplete, false)

	tm8ExpectGrid(t, term, []string{"pppppppp", "ijklmnop"})
}

// Zig: test "term51: eraseLine: complete releases styles, graphemes and hyperlinks and the row flags"
func TestTm8_eraseLine_complete_releases_styles_graphemes_and_hyperlinks_and_the_row_flags(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 0, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(tm8Bold()))
	tm8StartLink(t, term)
	tm8PutStr(t, term, 0, 0, "abcd")
	zigTry(t, term.SetAttribute(tm8Unset()))
	term.Screens.Active.EndHyperlink()
	tm8PutGrapheme(t, term, 6, 0)
	page := tm8Page(term)
	expectEqual(t, uint(1), page.styles.Count())
	expectEqual(t, uint(4), page.HyperlinkCount())
	expectEqual(t, uint(1), page.GraphemeCount())

	term.SetCursorPos(1, 5)
	term.EraseLine(CSIEraseLineComplete, false)

	tm8ExpectGrid(t, term, []string{"........", "ijklmnop"})
	expectEqual(t, uint(0), page.styles.Count())
	expectEqual(t, uint(0), page.HyperlinkCount())
	expectEqual(t, uint(0), page.hyperlink_set.Count())
	expectEqual(t, uint(0), page.GraphemeCount())
	expect(t, !tm8RowAt(t, term, 0).styled, "!term51URowAt(&t, 0).styled")
	expect(t, !tm8RowAt(t, term, 0).hyperlink, "!term51URowAt(&t, 0).hyperlink")
	expect(t, !tm8RowAt(t, term, 0).grapheme, "!term51URowAt(&t, 0).grapheme")
}

// Zig: test "term51: eraseLine: right respects ISO protection"
func TestTm8_eraseLine_right_respects_ISO_protection(t *testing.T) {
	testStdCapacity(t)
	tm8EraseLineProtected(t, CSIEraseLineRight, ANSIProtectedModeIso, None[ANSIProtectedMode](), false, "abc.ef..")
}

// Zig: test "term51: eraseLine: right ignores DEC protection"
func TestTm8_eraseLine_right_ignores_DEC_protection(t *testing.T) {
	testStdCapacity(t)
	tm8EraseLineProtected(t, CSIEraseLineRight, ANSIProtectedModeDec, None[ANSIProtectedMode](), false, "abc.....")
}

// Zig: test "term51: eraseLine: right respects DEC protection when requested"
func TestTm8_eraseLine_right_respects_DEC_protection_when_requested(t *testing.T) {
	testStdCapacity(t)
	tm8EraseLineProtected(t, CSIEraseLineRight, ANSIProtectedModeDec, None[ANSIProtectedMode](), true, "abc.ef..")
}

// Zig: test "term51: eraseLine: right ignores ISO protection when DEC was set most recently"
func TestTm8_eraseLine_right_ignores_ISO_protection_when_DEC_was_set_most_recently(t *testing.T) {
	testStdCapacity(t)
	tm8EraseLineProtected(t, CSIEraseLineRight, ANSIProtectedModeIso, Some(ANSIProtectedModeDec), false, "abc.....")
}

// Zig: test "term51: eraseLine: right respects DEC cells when ISO was set most recently"
func TestTm8_eraseLine_right_respects_DEC_cells_when_ISO_was_set_most_recently(t *testing.T) {
	testStdCapacity(t)
	tm8EraseLineProtected(t, CSIEraseLineRight, ANSIProtectedModeDec, Some(ANSIProtectedModeIso), false, "abc.ef..")
}

// Zig: test "term51: eraseLine: left respects ISO protection"
func TestTm8_eraseLine_left_respects_ISO_protection(t *testing.T) {
	testStdCapacity(t)
	tm8EraseLineProtected(t, CSIEraseLineLeft, ANSIProtectedModeIso, None[ANSIProtectedMode](), false, ".bc.efgh")
}

// Zig: test "term51: eraseLine: left ignores DEC protection"
func TestTm8_eraseLine_left_ignores_DEC_protection(t *testing.T) {
	testStdCapacity(t)
	tm8EraseLineProtected(t, CSIEraseLineLeft, ANSIProtectedModeDec, None[ANSIProtectedMode](), false, "....efgh")
}

// Zig: test "term51: eraseLine: left respects DEC protection when requested"
func TestTm8_eraseLine_left_respects_DEC_protection_when_requested(t *testing.T) {
	testStdCapacity(t)
	tm8EraseLineProtected(t, CSIEraseLineLeft, ANSIProtectedModeDec, None[ANSIProtectedMode](), true, ".bc.efgh")
}

// Zig: test "term51: eraseLine: left ignores ISO protection when DEC was set most recently"
func TestTm8_eraseLine_left_ignores_ISO_protection_when_DEC_was_set_most_recently(t *testing.T) {
	testStdCapacity(t)
	tm8EraseLineProtected(t, CSIEraseLineLeft, ANSIProtectedModeIso, Some(ANSIProtectedModeDec), false, "....efgh")
}

// Zig: test "term51: eraseLine: left respects DEC cells when ISO was set most recently"
func TestTm8_eraseLine_left_respects_DEC_cells_when_ISO_was_set_most_recently(t *testing.T) {
	testStdCapacity(t)
	tm8EraseLineProtected(t, CSIEraseLineLeft, ANSIProtectedModeDec, Some(ANSIProtectedModeIso), false, ".bc.efgh")
}

// Zig: test "term51: eraseLine: complete respects ISO protection"
func TestTm8_eraseLine_complete_respects_ISO_protection(t *testing.T) {
	testStdCapacity(t)
	tm8EraseLineProtected(t, CSIEraseLineComplete, ANSIProtectedModeIso, None[ANSIProtectedMode](), false, ".bc.ef..")
}

// Zig: test "term51: eraseLine: complete ignores DEC protection"
func TestTm8_eraseLine_complete_ignores_DEC_protection(t *testing.T) {
	testStdCapacity(t)
	tm8EraseLineProtected(t, CSIEraseLineComplete, ANSIProtectedModeDec, None[ANSIProtectedMode](), false, "........")
}

// Zig: test "term51: eraseLine: complete respects DEC protection when requested"
func TestTm8_eraseLine_complete_respects_DEC_protection_when_requested(t *testing.T) {
	testStdCapacity(t)
	tm8EraseLineProtected(t, CSIEraseLineComplete, ANSIProtectedModeDec, None[ANSIProtectedMode](), true, ".bc.ef..")
}

// Zig: test "term51: eraseLine: complete ignores ISO protection when DEC was set most recently"
func TestTm8_eraseLine_complete_ignores_ISO_protection_when_DEC_was_set_most_recently(t *testing.T) {
	testStdCapacity(t)
	tm8EraseLineProtected(t, CSIEraseLineComplete, ANSIProtectedModeIso, Some(ANSIProtectedModeDec), false, "........")
}

// Zig: test "term51: eraseLine: complete respects DEC cells when ISO was set most recently"
func TestTm8_eraseLine_complete_respects_DEC_cells_when_ISO_was_set_most_recently(t *testing.T) {
	testStdCapacity(t)
	tm8EraseLineProtected(t, CSIEraseLineComplete, ANSIProtectedModeDec, Some(ANSIProtectedModeIso), false, ".bc.ef..")
}

// Zig: test "term51: eraseLine: requested protection with no protection mode ever set clears everything"
func TestTm8_eraseLine_requested_protection_with_no_protection_mode_ever_set_clears_everything(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 3, 0)
	defer term.Deinit()

	term.EraseLine(CSIEraseLineComplete, true)
	tm8ExpectGrid(t, term, []string{"........", "ijklmnop"})
}

// Zig: test "term51: eraseLine: protection that is still switched on respects protected cells"
func TestTm8_eraseLine_protection_that_is_still_switched_on_respects_protected_cells(t *testing.T) {
	testStdCapacity(t)
	term := tm8EightAt(t, 0, 0)
	defer term.Deinit()

	term.SetProtectedMode(ANSIProtectedModeIso)
	term.SetCursorPos(1, 3)
	zigTry(t, term.Print('c'))
	term.SetCursorPos(1, 1)
	term.EraseLine(CSIEraseLineComplete, false)

	tm8ExpectGrid(t, term, []string{"..c.....", "ijklmnop"})
	expect(t, term.Screens.Active.Cursor.Protected, "t.screens.active.cursor.protected")
}

// ---- eraseDisplay ----

// Zig: test "term51: eraseDisplay: below from the middle of a row"
func TestTm8_eraseDisplay_below_from_the_middle_of_a_row(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 3, 1)
	defer term.Deinit()

	term.Screens.Active.Cursor.PendingWrap = true
	term.EraseDisplay(CSIEraseDisplayBelow, false)

	tm8ExpectGrid(t, term, []string{"abcdefgh", "ijk.....", "........", "........"})
	tm8ExpectCursor(t, term, 3, 1, false)
	tm8ExpectDirty(t, term, ".DDD")
	expect(t, !term.Flags.Dirty.Clear, "!t.flags.dirty.clear")
}

// Zig: test "term51: eraseDisplay: below from the top left erases everything"
func TestTm8_eraseDisplay_below_from_the_top_left_erases_everything(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 0, 0)
	defer term.Deinit()

	term.EraseDisplay(CSIEraseDisplayBelow, false)

	tm8ExpectGrid(t, term, []string{"........", "........", "........", "........"})
	tm8ExpectDirty(t, term, "DDDD")
}

// Zig: test "term51: eraseDisplay: below on the last row only erases the rest of the row"
func TestTm8_eraseDisplay_below_on_the_last_row_only_erases_the_rest_of_the_row(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 5, 3)
	defer term.Deinit()

	term.Screens.Active.Cursor.PendingWrap = true
	term.EraseDisplay(CSIEraseDisplayBelow, false)

	tm8ExpectGrid(t, term, []string{"abcdefgh", "ijklmnop", "qrstuvwx", "yzABC..."})
	tm8ExpectCursor(t, term, 5, 3, false)
	tm8ExpectDirty(t, term, "...D")
}

// Zig: test "term51: eraseDisplay: below on the last column of the last row"
func TestTm8_eraseDisplay_below_on_the_last_column_of_the_last_row(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 7, 3)
	defer term.Deinit()

	term.EraseDisplay(CSIEraseDisplayBelow, false)

	tm8ExpectGrid(t, term, []string{"abcdefgh", "ijklmnop", "qrstuvwx", "yzABCDE."})
}

// Zig: test "term51: eraseDisplay: below resets the rows below completely and the soft wrap of the cursor row"
func TestTm8_eraseDisplay_below_resets_the_rows_below_completely_and_the_soft_wrap_of_the_cursor_row(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 3, 1)
	defer term.Deinit()

	tm8SetWrap(t, term, "wwbb")
	tm8RowAt(t, term, 1).semantic_prompt = PageRowSemanticPromptPrompt
	tm8RowAt(t, term, 2).semantic_prompt = PageRowSemanticPromptPrompt
	tm8RowAt(t, term, 3).semantic_prompt = PageRowSemanticPromptPromptContinuation
	term.EraseDisplay(CSIEraseDisplayBelow, false)

	tm8ExpectWrap(t, term, "w...")
	// The cursor row keeps its prompt kind, the rows below lose theirs.
	expectEqual(t, PageRowSemanticPromptPrompt, tm8RowAt(t, term, 1).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptNone, tm8RowAt(t, term, 2).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptNone, tm8RowAt(t, term, 3).semantic_prompt)
}

// Zig: test "term51: eraseDisplay: below clears a spacer head at the end of a wrapped cursor row"
func TestTm8_eraseDisplay_below_clears_a_spacer_head_at_the_end_of_a_wrapped_cursor_row(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 3)
	defer term.Deinit()

	tm8PutStr(t, term, 0, 0, "abcdefg")
	zigTry(t, term.Print(0x4E2D))
	term.SetCursorPos(1, 3)
	term.EraseDisplay(CSIEraseDisplayBelow, false)

	tm8ExpectGrid(t, term, []string{"ab......", "........", "........"})
	tm8ExpectWrap(t, term, "...")
}

// Zig: test "term51: eraseDisplay: below erases the head of a wide character when on its tail"
func TestTm8_eraseDisplay_below_erases_the_head_of_a_wide_character_when_on_its_tail(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 2)
	defer term.Deinit()

	tm8WideRow(t, term)
	tm8PutStr(t, term, 0, 1, "xy")
	term.SetCursorPos(1, 4)
	term.EraseDisplay(CSIEraseDisplayBelow, false)

	tm8ExpectGrid(t, term, []string{"ab......", "........"})
}

// Zig: test "term51: eraseDisplay: below takes the background color of the cursor style"
func TestTm8_eraseDisplay_below_takes_the_background_color_of_the_cursor_style(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 6, 2)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(tm8DirectBg(4, 5, 6)))
	term.EraseDisplay(CSIEraseDisplayBelow, false)

	tm8ExpectGrid(t, term, []string{"abcdefgh", "ijklmnop", "qrstuvrr", "rrrrrrrr"})
	cell := *tm8CellAt(t, term, 3, 3)
	expectEqual(t, PageCellContentTagBgColorRgb, cell.content_tag)
	expectEqual(t, PageCellRGB{R: 4, G: 5, B: 6}, cell.content.color_rgb)
}

// Zig: test "term51: eraseDisplay: below takes a palette background color"
func TestTm8_eraseDisplay_below_takes_a_palette_background_color(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 6, 2)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(tm8Bg256(9)))
	term.EraseDisplay(CSIEraseDisplayBelow, false)

	tm8ExpectGrid(t, term, []string{"abcdefgh", "ijklmnop", "qrstuvpp", "pppppppp"})
}

// Zig: test "term51: eraseDisplay: below releases styles, graphemes and hyperlinks of the erased rows"
func TestTm8_eraseDisplay_below_releases_styles_graphemes_and_hyperlinks_of_the_erased_rows(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 0, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(tm8Bold()))
	tm8StartLink(t, term)
	tm8PutStr(t, term, 0, 2, "qrstuvwx")
	zigTry(t, term.SetAttribute(tm8Unset()))
	term.Screens.Active.EndHyperlink()
	tm8PutGrapheme(t, term, 1, 3)
	page := tm8Page(term)
	expectEqual(t, uint(1), page.styles.Count())
	expectEqual(t, uint(8), page.HyperlinkCount())
	expectEqual(t, uint(1), page.GraphemeCount())

	term.SetCursorPos(2, 1)
	term.EraseDisplay(CSIEraseDisplayBelow, false)

	expectEqual(t, uint(0), page.styles.Count())
	expectEqual(t, uint(0), page.HyperlinkCount())
	expectEqual(t, uint(0), page.hyperlink_set.Count())
	expectEqual(t, uint(0), page.GraphemeCount())
	for y := uint(1); y < 4; y++ {
		expect(t, !tm8RowAt(t, term, y).styled, "!term51URowAt(&t, y).styled")
		expect(t, !tm8RowAt(t, term, y).hyperlink, "!term51URowAt(&t, y).hyperlink")
		expect(t, !tm8RowAt(t, term, y).grapheme, "!term51URowAt(&t, y).grapheme")
	}
}

// Zig: test "term51: eraseDisplay: below ignores the margins"
func TestTm8_eraseDisplay_below_ignores_the_margins(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 3, 1)
	defer term.Deinit()

	term.ScrollingRegion = TerminalScrollingRegion{Top: 1, Bottom: 2, Left: 2, Right: 5}
	term.EraseDisplay(CSIEraseDisplayBelow, false)

	tm8ExpectGrid(t, term, []string{"abcdefgh", "ijk.....", "........", "........"})
}

// Zig: test "term51: eraseDisplay: above from the middle of a row"
func TestTm8_eraseDisplay_above_from_the_middle_of_a_row(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 3, 2)
	defer term.Deinit()

	term.Screens.Active.Cursor.PendingWrap = true
	term.EraseDisplay(CSIEraseDisplayAbove, false)

	tm8ExpectGrid(t, term, []string{"........", "........", "....uvwx", "yzABCDEF"})
	tm8ExpectCursor(t, term, 3, 2, false)
	tm8ExpectDirty(t, term, "DDD.")
	expect(t, !term.Flags.Dirty.Clear, "!t.flags.dirty.clear")
}

// Zig: test "term51: eraseDisplay: above on the first row only erases the start of the row"
func TestTm8_eraseDisplay_above_on_the_first_row_only_erases_the_start_of_the_row(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 2, 0)
	defer term.Deinit()

	term.Screens.Active.Cursor.PendingWrap = true
	term.EraseDisplay(CSIEraseDisplayAbove, false)

	tm8ExpectGrid(t, term, []string{"...defgh", "ijklmnop", "qrstuvwx", "yzABCDEF"})
	tm8ExpectCursor(t, term, 2, 0, false)
	tm8ExpectDirty(t, term, "D...")
}

// Zig: test "term51: eraseDisplay: above on the last cell erases everything"
func TestTm8_eraseDisplay_above_on_the_last_cell_erases_everything(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 7, 3)
	defer term.Deinit()

	term.EraseDisplay(CSIEraseDisplayAbove, false)

	tm8ExpectGrid(t, term, []string{"........", "........", "........", "........"})
}

// Zig: test "term51: eraseDisplay: above on the first cell erases only that cell"
func TestTm8_eraseDisplay_above_on_the_first_cell_erases_only_that_cell(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 0, 0)
	defer term.Deinit()

	term.EraseDisplay(CSIEraseDisplayAbove, false)

	tm8ExpectGrid(t, term, []string{".bcdefgh", "ijklmnop", "qrstuvwx", "yzABCDEF"})
}

// Zig: test "term51: eraseDisplay: above resets the rows above completely and keeps the soft wrap of the cursor row"
func TestTm8_eraseDisplay_above_resets_the_rows_above_completely_and_keeps_the_soft_wrap_of_the_cursor_row(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 3, 2)
	defer term.Deinit()

	tm8SetWrap(t, term, "bbwb")
	tm8RowAt(t, term, 0).semantic_prompt = PageRowSemanticPromptPrompt
	tm8RowAt(t, term, 1).semantic_prompt = PageRowSemanticPromptPromptContinuation
	tm8RowAt(t, term, 2).semantic_prompt = PageRowSemanticPromptPrompt
	term.EraseDisplay(CSIEraseDisplayAbove, false)

	// Erasing to the left does not reset the wrap of the cursor row.
	tm8ExpectWrap(t, term, "..wb")
	expectEqual(t, PageRowSemanticPromptNone, tm8RowAt(t, term, 0).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptNone, tm8RowAt(t, term, 1).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptPrompt, tm8RowAt(t, term, 2).semantic_prompt)
}

// Zig: test "term51: eraseDisplay: above includes the tail of a wide character at the cursor"
func TestTm8_eraseDisplay_above_includes_the_tail_of_a_wide_character_at_the_cursor(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 2)
	defer term.Deinit()

	tm8PutStr(t, term, 0, 0, "xy")
	tm8PutStr(t, term, 0, 1, "ab")
	tm8PutWide(t, term, 2, 1)
	tm8PutStr(t, term, 4, 1, "cdef")
	term.SetCursorPos(2, 3)
	term.EraseDisplay(CSIEraseDisplayAbove, false)

	tm8ExpectGrid(t, term, []string{"........", "....cdef"})
}

// Zig: test "term51: eraseDisplay: above takes the background color of the cursor style"
func TestTm8_eraseDisplay_above_takes_the_background_color_of_the_cursor_style(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 1, 2)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(tm8DirectBg(4, 5, 6)))
	term.EraseDisplay(CSIEraseDisplayAbove, false)

	tm8ExpectGrid(t, term, []string{"rrrrrrrr", "rrrrrrrr", "rrstuvwx", "yzABCDEF"})
	expectEqual(t, PageCellContentTagBgColorRgb, tm8CellAt(t, term, 0, 0).content_tag)
}

// Zig: test "term51: eraseDisplay: above takes a palette background color"
func TestTm8_eraseDisplay_above_takes_a_palette_background_color(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 1, 1)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(tm8Bg256(9)))
	term.EraseDisplay(CSIEraseDisplayAbove, false)

	tm8ExpectGrid(t, term, []string{"pppppppp", "ppklmnop", "qrstuvwx", "yzABCDEF"})
}

// Zig: test "term51: eraseDisplay: above releases styles, graphemes and hyperlinks of the erased rows"
func TestTm8_eraseDisplay_above_releases_styles_graphemes_and_hyperlinks_of_the_erased_rows(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 0, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(tm8Bold()))
	tm8StartLink(t, term)
	tm8PutStr(t, term, 0, 0, "abcdefgh")
	zigTry(t, term.SetAttribute(tm8Unset()))
	term.Screens.Active.EndHyperlink()
	tm8PutGrapheme(t, term, 1, 1)
	page := tm8Page(term)

	term.SetCursorPos(3, 1)
	term.EraseDisplay(CSIEraseDisplayAbove, false)

	expectEqual(t, uint(0), page.styles.Count())
	expectEqual(t, uint(0), page.HyperlinkCount())
	expectEqual(t, uint(0), page.GraphemeCount())
	tm8ExpectGrid(t, term, []string{"........", "........", ".rstuvwx", "yzABCDEF"})
}

// Zig: test "term51: eraseDisplay: above ignores the margins"
func TestTm8_eraseDisplay_above_ignores_the_margins(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 3, 2)
	defer term.Deinit()

	term.ScrollingRegion = TerminalScrollingRegion{Top: 1, Bottom: 2, Left: 2, Right: 5}
	term.EraseDisplay(CSIEraseDisplayAbove, false)

	tm8ExpectGrid(t, term, []string{"........", "........", "....uvwx", "yzABCDEF"})
}

// Zig: test "term51: eraseDisplay: complete erases everything and keeps the cursor"
func TestTm8_eraseDisplay_complete_erases_everything_and_keeps_the_cursor(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 3, 2)
	defer term.Deinit()

	term.Screens.Active.Cursor.PendingWrap = true
	term.EraseDisplay(CSIEraseDisplayComplete, false)

	tm8ExpectGrid(t, term, []string{"........", "........", "........", "........"})
	tm8ExpectCursor(t, term, 3, 2, false)
	tm8ExpectDirty(t, term, "DDDD")
	expect(t, term.Flags.Dirty.Clear, "t.flags.dirty.clear")
	expectEqual(t, uint(0), tm8HistoryRows(term))
}

// Zig: test "term51: eraseDisplay: complete resets the rows completely"
func TestTm8_eraseDisplay_complete_resets_the_rows_completely(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 3, 2)
	defer term.Deinit()

	tm8SetWrap(t, term, "wbcw")
	tm8RowAt(t, term, 0).semantic_prompt = PageRowSemanticPromptPrompt
	tm8RowAt(t, term, 1).semantic_prompt = PageRowSemanticPromptPromptContinuation
	term.EraseDisplay(CSIEraseDisplayComplete, false)

	tm8ExpectWrap(t, term, "....")
	for y := uint(0); y < 4; y++ {
		expectEqual(t, PageRowSemanticPromptNone, tm8RowAt(t, term, y).semantic_prompt)
	}
}

// Zig: test "term51: eraseDisplay: complete clears spacer heads and wide characters"
func TestTm8_eraseDisplay_complete_clears_spacer_heads_and_wide_characters(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 3)
	defer term.Deinit()

	tm8PutStr(t, term, 0, 0, "abcdefg")
	zigTry(t, term.Print(0x4E2D))
	tm8PutWide(t, term, 4, 2)
	term.EraseDisplay(CSIEraseDisplayComplete, false)

	tm8ExpectGrid(t, term, []string{"........", "........", "........"})
	tm8ExpectWrap(t, term, "...")
}

// Zig: test "term51: eraseDisplay: complete takes the background color of the cursor style"
func TestTm8_eraseDisplay_complete_takes_the_background_color_of_the_cursor_style(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 0, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(tm8DirectBg(4, 5, 6)))
	term.EraseDisplay(CSIEraseDisplayComplete, false)

	tm8ExpectGrid(t, term, []string{"rrrrrrrr", "rrrrrrrr", "rrrrrrrr", "rrrrrrrr"})
	cell := *tm8CellAt(t, term, 7, 3)
	expectEqual(t, PageCellContentTagBgColorRgb, cell.content_tag)
	expectEqual(t, PageCellRGB{R: 4, G: 5, B: 6}, cell.content.color_rgb)
	expectEqual(t, StyleDefaultID, cell.style_id)
}

// Zig: test "term51: eraseDisplay: complete takes a palette background color"
func TestTm8_eraseDisplay_complete_takes_a_palette_background_color(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 0, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(tm8Bg256(9)))
	term.EraseDisplay(CSIEraseDisplayComplete, false)

	tm8ExpectGrid(t, term, []string{"pppppppp", "pppppppp", "pppppppp", "pppppppp"})
}

// Zig: test "term51: eraseDisplay: complete releases styles, graphemes and hyperlinks"
func TestTm8_eraseDisplay_complete_releases_styles_graphemes_and_hyperlinks(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 0, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(tm8Bold()))
	tm8StartLink(t, term)
	tm8PutStr(t, term, 0, 0, "abcdefgh")
	zigTry(t, term.SetAttribute(tm8Unset()))
	term.Screens.Active.EndHyperlink()
	tm8PutGrapheme(t, term, 1, 3)
	page := tm8Page(term)

	term.EraseDisplay(CSIEraseDisplayComplete, false)

	expectEqual(t, uint(0), page.styles.Count())
	expectEqual(t, uint(0), page.HyperlinkCount())
	expectEqual(t, uint(0), page.hyperlink_set.Count())
	expectEqual(t, uint(0), page.GraphemeCount())
}

// Zig: test "term51: eraseDisplay: complete keeps the cursor style reference"
func TestTm8_eraseDisplay_complete_keeps_the_cursor_style_reference(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 0, 0)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(tm8Bold()))
	zigTry(t, term.SetAttribute(tm8DirectBg(1, 1, 1)))
	tm8PutStr(t, term, 0, 0, "abcdefgh")
	page := tm8Page(term)
	cursor_style := term.Screens.Active.Cursor.StyleID
	expectEqual(t, uint(9), uint(page.styles.RefCount(cursor_style)))

	term.EraseDisplay(CSIEraseDisplayComplete, false)

	expectEqual(t, cursor_style, term.Screens.Active.Cursor.StyleID)
	expectEqual(t, uint(1), uint(page.styles.RefCount(cursor_style)))
}

// Zig: test "term51: eraseDisplay: complete ignores the margins"
func TestTm8_eraseDisplay_complete_ignores_the_margins(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 3, 2)
	defer term.Deinit()

	term.ScrollingRegion = TerminalScrollingRegion{Top: 1, Bottom: 2, Left: 2, Right: 5}
	term.EraseDisplay(CSIEraseDisplayComplete, false)

	tm8ExpectGrid(t, term, []string{"........", "........", "........", "........"})
}

// Zig: test "term51: eraseDisplay: complete on the alternate screen erases without scrolling"
func TestTm8_eraseDisplay_complete_on_the_alternate_screen_erases_without_scrolling(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 3, 2)
	defer term.Deinit()

	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1049, true))
	expectEqual(t, ScreenSetKeyAlternate, term.Screens.ActiveKey)
	tm8Fill(t, term, tm8Alpha4)
	tm8RowAt(t, term, 3).semantic_prompt = PageRowSemanticPromptPrompt
	term.EraseDisplay(CSIEraseDisplayComplete, false)

	tm8ExpectGrid(t, term, []string{"........", "........", "........", "........"})
	expectEqual(t, uint(0), tm8HistoryRows(term))
	expect(t, term.Flags.Dirty.Clear, "t.flags.dirty.clear")
}

// Zig: test "term51: eraseDisplay: complete at a prompt on the last row scrolls the screen into the history first"
func TestTm8_eraseDisplay_complete_at_a_prompt_on_the_last_row_scrolls_the_screen_into_the_history_first(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 3, 2)
	defer term.Deinit()

	tm8RowAt(t, term, 3).semantic_prompt = PageRowSemanticPromptPrompt
	term.Screens.Active.Cursor.PendingWrap = true
	term.EraseDisplay(CSIEraseDisplayComplete, false)

	tm8ExpectGrid(t, term, []string{"........", "........", "........", "........"})
	expectEqual(t, uint(4), tm8HistoryRows(term))
	tm8ExpectHistoryRow(t, term, 0, "abcdefgh")
	tm8ExpectHistoryRow(t, term, 1, "ijklmnop")
	tm8ExpectHistoryRow(t, term, 2, "qrstuvwx")
	tm8ExpectHistoryRow(t, term, 3, "yzABCDEF")
	// The cursor sticks to its row, which is now in the history, so it is
	// reset to the top left of the active area.
	tm8ExpectCursor(t, term, 0, 0, false)
	expect(t, term.Flags.Dirty.Clear, "t.flags.dirty.clear")
}

// Zig: test "term51: eraseDisplay: complete at a prompt moves the cursor up with its row"
func TestTm8_eraseDisplay_complete_at_a_prompt_moves_the_cursor_up_with_its_row(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 4)
	defer term.Deinit()

	tm8Fill(t, term, []string{"abcdefgh", "ijklmnop", "........", "........"})
	tm8RowAt(t, term, 3).semantic_prompt = PageRowSemanticPromptPrompt
	term.SetCursorPos(4, 4)
	term.Screens.Active.Cursor.PendingWrap = true
	term.EraseDisplay(CSIEraseDisplayComplete, false)

	tm8ExpectGrid(t, term, []string{"........", "........", "........", "........"})
	expectEqual(t, uint(2), tm8HistoryRows(term))
	tm8ExpectCursor(t, term, 3, 1, false)
	expectEqual(t, PageRowSemanticPromptNone, tm8RowAt(t, term, 1).semantic_prompt)
}

// Zig: test "term51: eraseDisplay: complete at a prompt continuation on the last row scrolls first"
func TestTm8_eraseDisplay_complete_at_a_prompt_continuation_on_the_last_row_scrolls_first(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 0, 0)
	defer term.Deinit()

	tm8RowAt(t, term, 3).semantic_prompt = PageRowSemanticPromptPromptContinuation
	term.EraseDisplay(CSIEraseDisplayComplete, false)

	tm8ExpectGrid(t, term, []string{"........", "........", "........", "........"})
	expectEqual(t, uint(4), tm8HistoryRows(term))
}

// Zig: test "term51: eraseDisplay: complete only scrolls the rows up to the last non-empty row"
func TestTm8_eraseDisplay_complete_only_scrolls_the_rows_up_to_the_last_non_empty_row(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 4)
	defer term.Deinit()

	tm8Fill(t, term, []string{"abcdefgh", "ijklmnop", "........", "........"})
	tm8RowAt(t, term, 3).semantic_prompt = PageRowSemanticPromptPrompt
	term.EraseDisplay(CSIEraseDisplayComplete, false)

	tm8ExpectGrid(t, term, []string{"........", "........", "........", "........"})
	expectEqual(t, uint(2), tm8HistoryRows(term))
	tm8ExpectHistoryRow(t, term, 0, "abcdefgh")
	tm8ExpectHistoryRow(t, term, 1, "ijklmnop")
}

// Zig: test "term51: eraseDisplay: complete at a prompt on an empty screen does not scroll"
func TestTm8_eraseDisplay_complete_at_a_prompt_on_an_empty_screen_does_not_scroll(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 4)
	defer term.Deinit()

	tm8RowAt(t, term, 3).semantic_prompt = PageRowSemanticPromptPrompt
	term.EraseDisplay(CSIEraseDisplayComplete, false)

	expectEqual(t, uint(0), tm8HistoryRows(term))
}

// Zig: test "term51: eraseDisplay: complete only looks at the last row for a prompt"
func TestTm8_eraseDisplay_complete_only_looks_at_the_last_row_for_a_prompt(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 0, 0)
	defer term.Deinit()

	// A prompt above a row of output does not count as being at a prompt.
	tm8RowAt(t, term, 1).semantic_prompt = PageRowSemanticPromptPrompt
	tm8RowAt(t, term, 2).semantic_prompt = PageRowSemanticPromptPromptContinuation
	term.EraseDisplay(CSIEraseDisplayComplete, false)

	tm8ExpectGrid(t, term, []string{"........", "........", "........", "........"})
	expectEqual(t, uint(0), tm8HistoryRows(term))
}

// Zig: test "term51: eraseDisplay: complete with a blank last row after a prompt does not scroll"
func TestTm8_eraseDisplay_complete_with_a_blank_last_row_after_a_prompt_does_not_scroll(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 4)
	defer term.Deinit()

	tm8Fill(t, term, []string{"abcdefgh", "ijklmnop", "........", "........"})
	tm8RowAt(t, term, 1).semantic_prompt = PageRowSemanticPromptPrompt
	term.EraseDisplay(CSIEraseDisplayComplete, false)

	tm8ExpectGrid(t, term, []string{"........", "........", "........", "........"})
	expectEqual(t, uint(0), tm8HistoryRows(term))
}

// Zig: test "term51: eraseDisplay: complete at a prompt with protection respects protected cells after scrolling"
func TestTm8_eraseDisplay_complete_at_a_prompt_with_protection_respects_protected_cells_after_scrolling(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 0, 0)
	defer term.Deinit()

	tm8Protect(t, term, ANSIProtectedModeIso, 1, 3)
	tm8RowAt(t, term, 3).semantic_prompt = PageRowSemanticPromptPrompt
	term.EraseDisplay(CSIEraseDisplayComplete, false)

	// The content went to the history; the new active area is empty anyway.
	tm8ExpectGrid(t, term, []string{"........", "........", "........", "........"})
	expectEqual(t, uint(4), tm8HistoryRows(term))
	tm8ExpectHistoryRow(t, term, 0, "abcdefgh")
}

// Zig: test "term51: eraseDisplay: scroll_complete moves the screen into the history"
func TestTm8_eraseDisplay_scroll_complete_moves_the_screen_into_the_history(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 3, 2)
	defer term.Deinit()

	term.Screens.Active.Cursor.PendingWrap = true
	term.EraseDisplay(CSIEraseDisplayScrollComplete, false)

	tm8ExpectGrid(t, term, []string{"........", "........", "........", "........"})
	expectEqual(t, uint(4), tm8HistoryRows(term))
	tm8ExpectHistoryRow(t, term, 0, "abcdefgh")
	tm8ExpectHistoryRow(t, term, 3, "yzABCDEF")
	// The cursor sticks to its row, which is now in the history, so it is
	// reset to the top left of the active area.
	tm8ExpectCursor(t, term, 0, 0, false)
	// Unlike a complete erase it does not flag the screen as cleared.
	expect(t, !term.Flags.Dirty.Clear, "!t.flags.dirty.clear")
}

// Zig: test "term51: eraseDisplay: scroll_complete moves the cursor up with its row"
func TestTm8_eraseDisplay_scroll_complete_moves_the_cursor_up_with_its_row(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 4)
	defer term.Deinit()

	tm8Fill(t, term, []string{"abcdefgh", "ijklmnop", "........", "........"})
	term.SetCursorPos(4, 4)
	term.Screens.Active.Cursor.PendingWrap = true
	term.EraseDisplay(CSIEraseDisplayScrollComplete, false)

	expectEqual(t, uint(2), tm8HistoryRows(term))
	tm8ExpectCursor(t, term, 3, 1, false)
}

// Zig: test "term51: eraseDisplay: scroll_complete only scrolls up to the last non-empty row"
func TestTm8_eraseDisplay_scroll_complete_only_scrolls_up_to_the_last_non_empty_row(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 4)
	defer term.Deinit()

	tm8Fill(t, term, []string{"abcdefgh", "........", "qrstuvwx", "........"})
	term.EraseDisplay(CSIEraseDisplayScrollComplete, false)

	expectEqual(t, uint(3), tm8HistoryRows(term))
	tm8ExpectHistoryRow(t, term, 0, "abcdefgh")
	tm8ExpectHistoryRow(t, term, 1, "........")
	tm8ExpectHistoryRow(t, term, 2, "qrstuvwx")
	tm8ExpectGrid(t, term, []string{"........", "........", "........", "........"})
}

// Zig: test "term51: eraseDisplay: scroll_complete on an empty screen does nothing"
func TestTm8_eraseDisplay_scroll_complete_on_an_empty_screen_does_nothing(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 4)
	defer term.Deinit()

	term.Screens.Active.Cursor.PendingWrap = true
	term.EraseDisplay(CSIEraseDisplayScrollComplete, false)

	expectEqual(t, uint(0), tm8HistoryRows(term))
	tm8ExpectCursor(t, term, 0, 0, false)
}

// Zig: test "term51: eraseDisplay: scroll_complete counts a background colored blank as content"
func TestTm8_eraseDisplay_scroll_complete_counts_a_background_colored_blank_as_content(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 4)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(tm8DirectBg(1, 2, 3)))
	term.SetCursorPos(2, 1)
	term.EraseLine(CSIEraseLineComplete, false)
	zigTry(t, term.SetAttribute(tm8Unset()))
	tm8ExpectGrid(t, term, []string{"........", "rrrrrrrr", "........", "........"})
	term.EraseDisplay(CSIEraseDisplayScrollComplete, false)

	expectEqual(t, uint(2), tm8HistoryRows(term))
	tm8ExpectHistoryRow(t, term, 1, "rrrrrrrr")
}

// Zig: test "term51: eraseDisplay: scroll_complete counts a printed space as content"
func TestTm8_eraseDisplay_scroll_complete_counts_a_printed_space_as_content(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 4)
	defer term.Deinit()

	term.SetCursorPos(3, 1)
	zigTry(t, term.Print(' '))
	term.EraseDisplay(CSIEraseDisplayScrollComplete, false)

	expectEqual(t, uint(3), tm8HistoryRows(term))
}

// Zig: test "term51: eraseDisplay: scroll_complete does not count a styled blank as content"
func TestTm8_eraseDisplay_scroll_complete_does_not_count_a_styled_blank_as_content(t *testing.T) {
	testStdCapacity(t)
	term := tm8Init(t, 8, 4)
	defer term.Deinit()

	// A bold erase leaves blank cells without a style or background.
	zigTry(t, term.SetAttribute(tm8Bold()))
	term.SetCursorPos(3, 1)
	term.EraseLine(CSIEraseLineComplete, false)
	term.EraseDisplay(CSIEraseDisplayScrollComplete, false)

	expectEqual(t, uint(0), tm8HistoryRows(term))
}

// Zig: test "term51: eraseDisplay: scroll_complete ignores protection"
func TestTm8_eraseDisplay_scroll_complete_ignores_protection(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 0, 0)
	defer term.Deinit()

	tm8Protect(t, term, ANSIProtectedModeIso, 1, 3)
	term.EraseDisplay(CSIEraseDisplayScrollComplete, true)

	tm8ExpectGrid(t, term, []string{"........", "........", "........", "........"})
	expectEqual(t, uint(4), tm8HistoryRows(term))
	tm8ExpectHistoryRow(t, term, 0, "abcdefgh")
}

// Zig: test "term51: eraseDisplay: scroll_complete on the alternate screen leaves it empty"
func TestTm8_eraseDisplay_scroll_complete_on_the_alternate_screen_leaves_it_empty(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 0, 0)
	defer term.Deinit()

	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1049, true))
	tm8Fill(t, term, tm8Alpha4)
	term.EraseDisplay(CSIEraseDisplayScrollComplete, false)

	tm8ExpectGrid(t, term, []string{"........", "........", "........", "........"})
}

// Zig: test "term51: eraseDisplay: scrollback erases only the history"
func TestTm8_eraseDisplay_scrollback_erases_only_the_history(t *testing.T) {
	testStdCapacity(t)
	term := tm8ScrollbackTerm(t)
	defer term.Deinit()

	expectEqual(t, uint(3), tm8HistoryRows(term))
	tm8ExpectHistoryRow(t, term, 0, "line0...")
	tm8ExpectGrid(t, term, []string{"line3...", "line4...", "line5..."})
	tm8ExpectCursor(t, term, 5, 2, false)

	term.EraseDisplay(CSIEraseDisplayScrollback, false)

	expectEqual(t, uint(0), tm8HistoryRows(term))
	tm8ExpectGrid(t, term, []string{"line3...", "line4...", "line5..."})
	tm8ExpectCursor(t, term, 5, 2, false)
}

// Zig: test "term51: eraseDisplay: scrollback keeps pending wrap"
func TestTm8_eraseDisplay_scrollback_keeps_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	term := tm8ScrollbackTerm(t)
	defer term.Deinit()

	term.Screens.Active.Cursor.PendingWrap = true
	term.EraseDisplay(CSIEraseDisplayScrollback, false)

	expectEqual(t, uint(0), tm8HistoryRows(term))
	tm8ExpectCursor(t, term, 5, 2, true)
}

// Zig: test "term51: eraseDisplay: scrollback ignores protection"
func TestTm8_eraseDisplay_scrollback_ignores_protection(t *testing.T) {
	testStdCapacity(t)
	term := tm8ScrollbackTerm(t)
	defer term.Deinit()

	term.SetProtectedMode(ANSIProtectedModeIso)
	term.EraseDisplay(CSIEraseDisplayScrollback, true)

	expectEqual(t, uint(0), tm8HistoryRows(term))
	tm8ExpectGrid(t, term, []string{"line3...", "line4...", "line5..."})
}

// Zig: test "term51: eraseDisplay: scrollback with no history changes nothing"
func TestTm8_eraseDisplay_scrollback_with_no_history_changes_nothing(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 3, 2)
	defer term.Deinit()

	term.EraseDisplay(CSIEraseDisplayScrollback, false)

	tm8ExpectGrid(t, term, tm8Alpha4)
	expectEqual(t, uint(0), tm8HistoryRows(term))
	tm8ExpectCursor(t, term, 3, 2, false)
}

// Zig: test "term51: eraseDisplay: scrollback releases the history of a screen scrolled with scroll_complete"
func TestTm8_eraseDisplay_scrollback_releases_the_history_of_a_screen_scrolled_with_scroll_complete(t *testing.T) {
	testStdCapacity(t)
	term := tm8FourAt(t, 3, 2)
	defer term.Deinit()

	term.EraseDisplay(CSIEraseDisplayScrollComplete, false)
	expectEqual(t, uint(4), tm8HistoryRows(term))
	term.EraseDisplay(CSIEraseDisplayScrollback, false)

	expectEqual(t, uint(0), tm8HistoryRows(term))
	tm8ExpectGrid(t, term, []string{"........", "........", "........", "........"})
}

// Zig: test "term51: eraseDisplay: below respects ISO protection"
func TestTm8_eraseDisplay_below_respects_ISO_protection(t *testing.T) {
	testStdCapacity(t)
	tm8EraseDisplayProtected(t, CSIEraseDisplayBelow, ANSIProtectedModeIso, None[ANSIProtectedMode](), false, tm8BelowRespected)
}

// Zig: test "term51: eraseDisplay: below ignores DEC protection"
func TestTm8_eraseDisplay_below_ignores_DEC_protection(t *testing.T) {
	testStdCapacity(t)
	tm8EraseDisplayProtected(t, CSIEraseDisplayBelow, ANSIProtectedModeDec, None[ANSIProtectedMode](), false, tm8BelowIgnored)
}

// Zig: test "term51: eraseDisplay: below respects DEC protection when requested"
func TestTm8_eraseDisplay_below_respects_DEC_protection_when_requested(t *testing.T) {
	testStdCapacity(t)
	tm8EraseDisplayProtected(t, CSIEraseDisplayBelow, ANSIProtectedModeDec, None[ANSIProtectedMode](), true, tm8BelowRespected)
}

// Zig: test "term51: eraseDisplay: below ignores ISO protection when DEC was set most recently"
func TestTm8_eraseDisplay_below_ignores_ISO_protection_when_DEC_was_set_most_recently(t *testing.T) {
	testStdCapacity(t)
	tm8EraseDisplayProtected(t, CSIEraseDisplayBelow, ANSIProtectedModeIso, Some(ANSIProtectedModeDec), false, tm8BelowIgnored)
}

// Zig: test "term51: eraseDisplay: below respects DEC cells when ISO was set most recently"
func TestTm8_eraseDisplay_below_respects_DEC_cells_when_ISO_was_set_most_recently(t *testing.T) {
	testStdCapacity(t)
	tm8EraseDisplayProtected(t, CSIEraseDisplayBelow, ANSIProtectedModeDec, Some(ANSIProtectedModeIso), false, tm8BelowRespected)
}

// Zig: test "term51: eraseDisplay: above respects ISO protection"
func TestTm8_eraseDisplay_above_respects_ISO_protection(t *testing.T) {
	testStdCapacity(t)
	tm8EraseDisplayProtected(t, CSIEraseDisplayAbove, ANSIProtectedModeIso, None[ANSIProtectedMode](), false, tm8AboveRespected)
}

// Zig: test "term51: eraseDisplay: above ignores DEC protection"
func TestTm8_eraseDisplay_above_ignores_DEC_protection(t *testing.T) {
	testStdCapacity(t)
	tm8EraseDisplayProtected(t, CSIEraseDisplayAbove, ANSIProtectedModeDec, None[ANSIProtectedMode](), false, tm8AboveIgnored)
}

// Zig: test "term51: eraseDisplay: above respects DEC protection when requested"
func TestTm8_eraseDisplay_above_respects_DEC_protection_when_requested(t *testing.T) {
	testStdCapacity(t)
	tm8EraseDisplayProtected(t, CSIEraseDisplayAbove, ANSIProtectedModeDec, None[ANSIProtectedMode](), true, tm8AboveRespected)
}

// Zig: test "term51: eraseDisplay: above ignores ISO protection when DEC was set most recently"
func TestTm8_eraseDisplay_above_ignores_ISO_protection_when_DEC_was_set_most_recently(t *testing.T) {
	testStdCapacity(t)
	tm8EraseDisplayProtected(t, CSIEraseDisplayAbove, ANSIProtectedModeIso, Some(ANSIProtectedModeDec), false, tm8AboveIgnored)
}

// Zig: test "term51: eraseDisplay: above respects DEC cells when ISO was set most recently"
func TestTm8_eraseDisplay_above_respects_DEC_cells_when_ISO_was_set_most_recently(t *testing.T) {
	testStdCapacity(t)
	tm8EraseDisplayProtected(t, CSIEraseDisplayAbove, ANSIProtectedModeDec, Some(ANSIProtectedModeIso), false, tm8AboveRespected)
}

// Zig: test "term51: eraseDisplay: complete respects ISO protection"
func TestTm8_eraseDisplay_complete_respects_ISO_protection(t *testing.T) {
	testStdCapacity(t)
	tm8EraseDisplayProtected(t, CSIEraseDisplayComplete, ANSIProtectedModeIso, None[ANSIProtectedMode](), false, tm8CompleteRespected)
}

// Zig: test "term51: eraseDisplay: complete ignores DEC protection"
func TestTm8_eraseDisplay_complete_ignores_DEC_protection(t *testing.T) {
	testStdCapacity(t)
	tm8EraseDisplayProtected(t, CSIEraseDisplayComplete, ANSIProtectedModeDec, None[ANSIProtectedMode](), false, tm8CompleteIgnored)
}

// Zig: test "term51: eraseDisplay: complete respects DEC protection when requested"
func TestTm8_eraseDisplay_complete_respects_DEC_protection_when_requested(t *testing.T) {
	testStdCapacity(t)
	tm8EraseDisplayProtected(t, CSIEraseDisplayComplete, ANSIProtectedModeDec, None[ANSIProtectedMode](), true, tm8CompleteRespected)
}

// Zig: test "term51: eraseDisplay: complete ignores ISO protection when DEC was set most recently"
func TestTm8_eraseDisplay_complete_ignores_ISO_protection_when_DEC_was_set_most_recently(t *testing.T) {
	testStdCapacity(t)
	tm8EraseDisplayProtected(t, CSIEraseDisplayComplete, ANSIProtectedModeIso, Some(ANSIProtectedModeDec), false, tm8CompleteIgnored)
}

// Zig: test "term51: eraseDisplay: complete respects DEC cells when ISO was set most recently"
func TestTm8_eraseDisplay_complete_respects_DEC_cells_when_ISO_was_set_most_recently(t *testing.T) {
	testStdCapacity(t)
	tm8EraseDisplayProtected(t, CSIEraseDisplayComplete, ANSIProtectedModeDec, Some(ANSIProtectedModeIso), false, tm8CompleteRespected)
}
