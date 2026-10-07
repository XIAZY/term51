// Tests translated one for one from src/terminal/Terminal.zig (Ghostty
// v1.3.1 plus the term51 additional tests), from the test "term51: eraseDisplay:
// complete keeps the row attributes of rows with protected cells" to the end
// of the file. Each Go test carries the exact Zig title in a "// Zig: test"
// comment. Helpers are prefixed tm9.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualStrings keep Zig's names and Zig's
//     (expected, actual) argument order. `try` on a call that returns an
//     error is zigTry(t, err); `x.?` is zigUnwrap or a nil check.
//   - The Zig terminal `t` is `term` here, because `t` is the *testing.T.
//   - `init(alloc, .{ .cols = c, .rows = r })` is tm9Init(t, c, r), which
//     writes out the Zig defaults (terminalOptionsDefault); other options
//     modify terminalOptionsDefault(c, r) (tm9InitOpts).
//   - Zig's `alloc` arguments are dropped (Terminal.deccolm, resize);
//     `testing.allocator` leak checks are dropped with `// Zig-only:`.
//   - `defer t.deinit(alloc)` inside a Zig loop body runs at the end of the
//     iteration: the Go loop body is a func literal with the defer.
//   - Pointer comparisons of byte buffers (`.ptr`) are comparisons of the
//     address of the first element.
//   - A `usize` compared with a CellCountInt field is converted to uint on
//     both sides.
//   - The Zig test helpers term51U* (defined earlier in the Zig file) that
//     these tests use are translated as tm9* helpers.
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt.

package terminal

import (
	"errors"
	"testing"
)

// tm9ExpectPanic runs f, which must panic. A panic with the wireframe's
// "not implemented" value does not count: it only means the function under
// test does not exist yet. The setup is done outside f.
func tm9ExpectPanic(t *testing.T, f func()) {
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

// tm9ExpectError is Zig's `try testing.expectError(want, got)`.
func tm9ExpectError(t *testing.T, want error, got error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("expected error %v, got %v", want, got)
	}
}

// tm9InitOpts is `try init(alloc, opts)`; the terminal is returned by
// pointer (the methods take a pointer receiver).
func tm9InitOpts(t *testing.T, opts TerminalOptions) *Terminal {
	t.Helper()
	term, err := TerminalInit(opts)
	zigTry(t, err)
	return &term
}

// tm9Init is `try init(alloc, .{ .cols = cols, .rows = rows })`.
func tm9Init(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) *Terminal {
	t.Helper()
	return tm9InitOpts(t, terminalOptionsDefault(cols, rows))
}

// tm9PrintString is `try t.printString(str)`.
func tm9PrintString(t *testing.T, term *Terminal, str string) {
	t.Helper()
	zigTry(t, term.PrintString([]uint8(str)))
}

// tm9SetAttribute is `try t.setAttribute(attr)`.
func tm9SetAttribute(t *testing.T, term *Terminal, attr SGRAttribute) {
	t.Helper()
	zigTry(t, term.SetAttribute(attr))
}

// tm9Attr builds an SGRAttribute that has no payload.
func tm9Attr(tag SGRAttributeTag) SGRAttribute {
	return SGRAttribute{Tag: tag}
}

// tm9Underline is `.{ .underline = u }`.
func tm9Underline(u SGRAttributeUnderline) SGRAttribute {
	return SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: u}
}

// tm9Active is the Zig literal `.{ .active = .{ .x = x, .y = y } }`.
func tm9Active(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagActive, Active: PointCoordinate{X: x, Y: y}}
}

// tm9Screen is `.{ .screen = .{ .x = x, .y = y } }`.
func tm9Screen(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagScreen, Screen: PointCoordinate{X: x, Y: y}}
}

// tm9Viewport is `.{ .viewport = .{ .x = x, .y = y } }`.
func tm9Viewport(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagViewport, Viewport: PointCoordinate{X: x, Y: y}}
}

// tm9History is `.{ .history = .{ .x = x, .y = y } }`.
func tm9History(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagHistory, History: PointCoordinate{X: x, Y: y}}
}

// tm9PalColor is `.{ .palette = idx }` of style.Style.Color.
func tm9PalColor(idx uint8) StyleColor {
	return StyleColor{Tag: StyleColorPalette, Palette: idx}
}

// tm9RgbColor is `.{ .rgb = .{ .r = r, .g = g, .b = b } }` of style.Style.Color.
func tm9RgbColor(r uint8, g uint8, b uint8) StyleColor {
	return StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{R: r, G: g, B: b}}
}

// ---------------------------------------------------------------------------
// term51U helpers (defined before this range in Terminal.zig).
// ---------------------------------------------------------------------------

// tm9Alpha4 is term51UAlpha4.
var tm9Alpha4 = []string{"abcdefgh", "ijklmnop", "qrstuvwx", "yzABCDEF"}

// tm9Fill is term51UFill.
func tm9Fill(t *testing.T, term *Terminal, rows []string) {
	t.Helper()
	for y, row := range rows {
		for x := 0; x < len(row); x++ {
			c := row[x]
			if c == '.' {
				continue
			}
			term.SetCursorPos(uint(y+1), uint(x+1))
			zigTry(t, term.Print(uint32(c)))
		}
	}
	term.SetCursorPos(1, 1)
}

// tm9CellAt is term51UCellAt.
func tm9CellAt(t *testing.T, term *Terminal, x uint, y uint) *PageCell {
	t.Helper()
	return zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm9Active(SizeCellCountInt(x), uint32(y)))).Cell
}

// tm9RowAt is term51URowAt.
func tm9RowAt(t *testing.T, term *Terminal, y uint) *PageRow {
	t.Helper()
	return zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm9Active(0, uint32(y)))).Row
}

// tm9CellChar is term51UCellChar: one character per cell: '.' empty, the
// ASCII character for narrow text, '*' narrow grapheme cluster, '#' wide
// head, '>' spacer tail, '<' spacer head, 'p' palette background cell, 'r'
// RGB background cell.
func tm9CellChar(cell PageCell) uint8 {
	switch cell.wide {
	case PageCellWideWide:
		return '#'
	case PageCellWideSpacerTail:
		return '>'
	case PageCellWideSpacerHead:
		return '<'
	default: // narrow
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
	}
	panic("unreachable")
}

// tm9ExpectGrid is term51UExpectGrid: compare the whole active area against
// the expected rows.
func tm9ExpectGrid(t *testing.T, term *Terminal, expected []string) {
	t.Helper()
	expectEqual(t, uint(term.Rows), uint(len(expected)))
	buf := make([]uint8, term.Cols)
	for y, exp := range expected {
		for x := uint(0); x < uint(term.Cols); x++ {
			buf[x] = tm9CellChar(*tm9CellAt(t, term, x, uint(y)))
		}
		expectEqualStrings(t, exp, string(buf[0:term.Cols]))
	}
}

// tm9ExpectDirty is term51UExpectDirty: compare the dirty flag of every
// active row: 'D' dirty, '.' clean.
func tm9ExpectDirty(t *testing.T, term *Terminal, expected string) {
	t.Helper()
	expectEqual(t, uint(term.Rows), uint(len(expected)))
	buf := make([]uint8, term.Rows)
	for y := uint(0); y < uint(term.Rows); y++ {
		if tm9RowAt(t, term, y).dirty {
			buf[y] = 'D'
		} else {
			buf[y] = '.'
		}
	}
	expectEqualStrings(t, expected, string(buf[0:term.Rows]))
}

// tm9ExpectWrap is term51UExpectWrap: compare the wrap (w),
// wrap_continuation (c), both (b) or neither (.) of every active row.
func tm9ExpectWrap(t *testing.T, term *Terminal, expected string) {
	t.Helper()
	expectEqual(t, uint(term.Rows), uint(len(expected)))
	buf := make([]uint8, term.Rows)
	for y := uint(0); y < uint(term.Rows); y++ {
		row := tm9RowAt(t, term, y)
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

// tm9SetWrap is term51USetWrap: set every row to the "wrap" state given by
// the characters of flags.
func tm9SetWrap(t *testing.T, term *Terminal, flags string) {
	t.Helper()
	for y := 0; y < len(flags); y++ {
		f := flags[y]
		row := tm9RowAt(t, term, uint(y))
		row.wrap = f == 'w' || f == 'b'
		row.wrap_continuation = f == 'c' || f == 'b'
	}
}

// tm9ExpectCursor is term51UExpectCursor.
func tm9ExpectCursor(t *testing.T, term *Terminal, x uint, y uint, pending_wrap bool) {
	t.Helper()
	expectEqual(t, x, uint(term.Screens.Active.Cursor.X))
	expectEqual(t, y, uint(term.Screens.Active.Cursor.Y))
	expectEqual(t, pending_wrap, term.Screens.Active.Cursor.PendingWrap)
}

// tm9Page is term51UPage: the page that holds the cursor; fine for terminals
// that fit one page.
func tm9Page(term *Terminal) *Page {
	return &term.Screens.Active.Cursor.PagePin.Node.data
}

// tm9PutStr is term51UPutStr.
func tm9PutStr(t *testing.T, term *Terminal, x uint, y uint, str string) {
	t.Helper()
	term.SetCursorPos(y+1, x+1)
	tm9PrintString(t, term, str)
}

// tm9PutWide is term51UPutWide: print one wide character (two cells) with
// its head at (x, y).
func tm9PutWide(t *testing.T, term *Terminal, x uint, y uint) {
	t.Helper()
	term.SetCursorPos(y+1, x+1)
	zigTry(t, term.Print(0x4E2D))
}

// tm9PutWideGrapheme is term51UPutWideGrapheme: print a family emoji ZWJ
// sequence, which is one wide cell with a grapheme.
func tm9PutWideGrapheme(t *testing.T, term *Terminal, x uint, y uint) {
	t.Helper()
	term.Modes.Set(ModesModeGraphemeCluster, true)
	term.SetCursorPos(y+1, x+1)
	zigTry(t, term.Print(0x1F468))
	zigTry(t, term.Print(0x200D))
	zigTry(t, term.Print(0x1F469))
}

// tm9PutGrapheme is term51UPutGrapheme: print an 'e' followed by a combining
// acute accent at (x, y): one narrow cell with a grapheme.
func tm9PutGrapheme(t *testing.T, term *Terminal, x uint, y uint) {
	t.Helper()
	term.SetCursorPos(y+1, x+1)
	zigTry(t, term.Print('e'))
	// The grapheme is attached directly: a zero-width character printed
	// after a character on the last column would attach to the cell before.
	zigTry(t, tm9Page(term).SetGraphemes(
		tm9RowAt(t, term, y),
		tm9CellAt(t, term, x, y),
		[]uint32{0x0301},
	))
}

// tm9WideRow is term51UWideRow: row "ab#>cdef" with the wide character at
// columns 2 and 3.
func tm9WideRow(t *testing.T, term *Terminal) {
	t.Helper()
	tm9PutStr(t, term, 0, 0, "ab")
	tm9PutWide(t, term, 2, 0)
	tm9PutStr(t, term, 4, 0, "cdef")
	tm9ExpectGrid(t, term, []string{"ab#>cdef", "........"})
	term.SetCursorPos(1, 1)
}

// tm9FourAt is term51UFourAt: an 8 column, 4 row terminal holding tm9Alpha4
// with the cursor at (x, y) and dirty flags cleared.
func tm9FourAt(t *testing.T, x uint, y uint) *Terminal {
	t.Helper()
	// Zig: errdefer t.deinit(alloc) (a failing step stops the Go test).
	term := tm9Init(t, 8, 4)
	tm9Fill(t, term, tm9Alpha4)
	term.SetCursorPos(y+1, x+1)
	term.clearDirty()
	term.Flags.Dirty.Clear = false
	return term
}

// tm9HistoryRows is term51UHistoryRows: the number of rows of scrollback.
func tm9HistoryRows(term *Terminal) uint {
	return term.Screens.Active.Pages.total_rows - uint(term.Rows)
}

// tm9ExpectHistoryRow is term51UExpectHistoryRow: the text of a history row,
// one tm9CellChar per column.
func tm9ExpectHistoryRow(t *testing.T, term *Terminal, y uint, expected string) {
	t.Helper()
	buf := make([]uint8, term.Cols)
	for x := uint(0); x < uint(term.Cols); x++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm9History(SizeCellCountInt(x), uint32(y))))
		buf[x] = tm9CellChar(*list_cell.Cell)
	}
	expectEqualStrings(t, expected, string(buf[0:term.Cols]))
}

// tm9ScrollbackTerm is term51UScrollbackTerm.
func tm9ScrollbackTerm(t *testing.T) *Terminal {
	t.Helper()
	// Zig: errdefer t.deinit(alloc) (a failing step stops the Go test).
	term := tm9Init(t, 8, 3)
	lines := []string{"line0", "line1", "line2", "line3", "line4", "line5"}
	for i, line := range lines {
		if i > 0 {
			term.CarriageReturn()
			zigTry(t, term.Linefeed())
		}
		tm9PrintString(t, term, line)
	}
	return term
}

// tm9ProtectAt is term51UProtectAt: print the letter of tm9Alpha4 at (x, y)
// with protection mode, then switch protection off.
func tm9ProtectAt(t *testing.T, term *Terminal, mode ANSIProtectedMode, x uint, y uint) {
	t.Helper()
	term.SetProtectedMode(mode)
	term.SetCursorPos(y+1, x+1)
	zigTry(t, term.Print(uint32(tm9Alpha4[y][x])))
	term.SetProtectedMode(ANSIProtectedModeOff)
}

// ---------------------------------------------------------------------------
// term51: additional tests (V) helpers.
// ---------------------------------------------------------------------------

// tm9ExpectPlain is term51VExpectPlain.
func tm9ExpectPlain(t *testing.T, term *Terminal, expected string) {
	t.Helper()
	str, err := term.PlainString()
	zigTry(t, err)
	expectEqualStrings(t, expected, string(str))
}

// tm9ExpectPlainUnwrapped is term51VExpectPlainUnwrapped.
func tm9ExpectPlainUnwrapped(t *testing.T, term *Terminal, expected string) {
	t.Helper()
	str, err := term.PlainStringUnwrapped()
	zigTry(t, err)
	expectEqualStrings(t, expected, string(str))
}

// tm9AttrCase is term51VAttrCase.
type tm9AttrCase struct {
	attr   SGRAttribute
	expect Style
}

// tm9Apply is term51VApply: apply one attribute to the default style of a
// fresh terminal and return the resulting cursor style.
func tm9Apply(t *testing.T, attr SGRAttribute) Style {
	t.Helper()
	term := tm9Init(t, 5, 5)
	defer term.Deinit()
	tm9SetAttribute(t, term, attr)
	return term.Screens.Active.Cursor.Style
}

// tm9FullStyle is term51VFullStyle: set every flag and the three colors so
// that resets can be observed.
func tm9FullStyle(t *testing.T, term *Terminal) {
	t.Helper()
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagBold))
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagFaint))
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagItalic))
	tm9SetAttribute(t, term, tm9Underline(SGRAttributeUnderlineCurly))
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagBlink))
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagInverse))
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagInvisible))
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagStrikethrough))
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagOverline))
	tm9SetAttribute(t, term, SGRAttribute{Tag: SGRAttributeTagUnderlineColor, UnderlineColor: ColorRGB{R: 9, G: 8, B: 7}})
	tm9SetAttribute(t, term, SGRAttribute{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: ColorRGB{R: 1, G: 2, B: 3}})
	tm9SetAttribute(t, term, SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: 4, G: 5, B: 6}})
}

// tm9FullStyleValue is term51VFullStyleValue.
func tm9FullStyleValue() Style {
	return Style{
		FgColor:         tm9RgbColor(1, 2, 3),
		BgColor:         tm9RgbColor(4, 5, 6),
		underline_color: tm9RgbColor(9, 8, 7),
		Flags: StyleFlags{
			Bold:          true,
			Faint:         true,
			Italic:        true,
			Underline:     SGRAttributeUnderlineCurly,
			Blink:         true,
			Inverse:       true,
			Invisible:     true,
			Strikethrough: true,
			Overline:      true,
		},
	}
}

// tm9ScreenPlain is term51VScreenPlain.
func tm9ScreenPlain(t *testing.T, s *Screen, expected string) {
	t.Helper()
	str, err := s.DumpStringAlloc(Point{Tag: PointTagViewport})
	zigTry(t, err)
	expectEqualStrings(t, expected, string(str))
}

// tm9Select is term51VSelect: select (x1,y1)-(x2,y2) of the active area of a
// screen.
func tm9Select(t *testing.T, s *Screen, x1 SizeCellCountInt, y1 SizeCellCountInt, x2 SizeCellCountInt, y2 SizeCellCountInt) {
	t.Helper()
	tl := zigUnwrap(t, s.Pages.Pin(tm9Active(x1, uint32(y1))))
	br := zigUnwrap(t, s.Pages.Pin(tm9Active(x2, uint32(y2))))
	zigTry(t, s.Select(Some(SelectionInit(tl, br, false))))
}

// tm9ExpectRegion is term51VExpectRegion.
func tm9ExpectRegion(t *testing.T, term *Terminal, top SizeCellCountInt, bottom SizeCellCountInt, left SizeCellCountInt, right SizeCellCountInt) {
	t.Helper()
	expectEqual(t, top, term.ScrollingRegion.Top)
	expectEqual(t, bottom, term.ScrollingRegion.Bottom)
	expectEqual(t, left, term.ScrollingRegion.Left)
	expectEqual(t, right, term.ScrollingRegion.Right)
}

// tm9PrimarySetup is term51VPrimarySetup.
func tm9PrimarySetup(t *testing.T, term *Terminal) {
	t.Helper()
	tm9PrintString(t, term, "AB")
	term.SetCursorPos(3, 4)
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagBold))
	term.SetProtectedMode(ANSIProtectedModeIso)
	term.Screens.Active.Cursor.PendingWrap = true
	term.Screens.Active.Cursor.CursorStyle = ScreenCursorStyleBar
}

// tm9StartHyperlink is `try s.startHyperlink(uri, null)`.
func tm9StartHyperlink(t *testing.T, s *Screen, uri string) {
	t.Helper()
	zigTry(t, s.StartHyperlink([]uint8(uri), None[[]uint8]()))
}

// tm9ModesMap applies f to every mode of a ModePacked (Zig does this with
// `~@as(Int, @bitCast(p))`, a bit-wise not of the packed integer).
func tm9ModesMap(p ModesModePacked, f func(bool) bool) ModesModePacked {
	for _, name := range sm1_modesFieldNames {
		sm1_modesFieldSet(&p, name, f(sm1_modesField(&p, name)))
	}
	return p
}

// tm9ModesNot is `~@as(Int, @bitCast(p))`.
func tm9ModesNot(p ModesModePacked) ModesModePacked {
	return tm9ModesMap(p, func(b bool) bool { return !b })
}

// Zig: test "term51: eraseDisplay: complete keeps the row attributes of rows with protected cells"
func TestTm9_eraseDisplay_complete_keeps_the_row_attributes_of_rows_with_protected_cells(t *testing.T) {
	testStdCapacity(t)
	term := tm9FourAt(t, 3, 1)
	defer term.Deinit()

	tm9ProtectAt(t, term, ANSIProtectedModeIso, 3, 2)
	tm9SetWrap(t, term, "bwcb")
	tm9RowAt(t, term, 2).semantic_prompt = PageRowSemanticPromptPrompt
	term.EraseDisplay(CSIEraseDisplayComplete, false)

	tm9ExpectGrid(t, term, []string{"........", "........", "...t....", "........"})
	// The protected path only clears cells and so keeps the row flags, even
	// those of rows without protected cells.
	tm9ExpectWrap(t, term, "bwcb")
	expectEqual(t, PageRowSemanticPromptPrompt, tm9RowAt(t, term, 2).semantic_prompt)
	tm9ExpectDirty(t, term, "DDDD")
}

// Zig: test "term51: eraseDisplay: complete without protection resets the row attributes"
func TestTm9_eraseDisplay_complete_without_protection_resets_the_row_attributes(t *testing.T) {
	testStdCapacity(t)
	term := tm9FourAt(t, 3, 1)
	defer term.Deinit()

	tm9ProtectAt(t, term, ANSIProtectedModeDec, 3, 2)
	tm9SetWrap(t, term, "bwcb")
	tm9RowAt(t, term, 2).semantic_prompt = PageRowSemanticPromptPrompt
	term.EraseDisplay(CSIEraseDisplayComplete, false)

	tm9ExpectGrid(t, term, []string{"........", "........", "........", "........"})
	tm9ExpectWrap(t, term, "....")
	expectEqual(t, PageRowSemanticPromptNone, tm9RowAt(t, term, 2).semantic_prompt)
}

// Zig: test "term51: eraseDisplay: protected cells that remain keep their protection"
func TestTm9_eraseDisplay_protected_cells_that_remain_keep_their_protection(t *testing.T) {
	testStdCapacity(t)
	term := tm9FourAt(t, 3, 1)
	defer term.Deinit()

	tm9ProtectAt(t, term, ANSIProtectedModeIso, 3, 2)
	term.EraseDisplay(CSIEraseDisplayComplete, false)

	expect(t, tm9CellAt(t, term, 3, 2).protected, "term51UCellAt(&t, 3, 2).protected")
	expect(t, !tm9CellAt(t, term, 4, 2).protected, "!term51UCellAt(&t, 4, 2).protected")
}

// ---- decaln ----

// Zig: test "term51: decaln: fills the screen with E and homes the cursor"
func TestTm9_decaln_fills_the_screen_with_E_and_homes_the_cursor(t *testing.T) {
	testStdCapacity(t)
	term := tm9FourAt(t, 3, 2)
	defer term.Deinit()

	term.Screens.Active.Cursor.PendingWrap = true
	zigTry(t, term.Decaln())

	tm9ExpectGrid(t, term, []string{"EEEEEEEE", "EEEEEEEE", "EEEEEEEE", "EEEEEEEE"})
	tm9ExpectCursor(t, term, 0, 0, false)
	tm9ExpectDirty(t, term, "DDDD")
	// Unlike erase display it does not flag the screen as cleared.
	expect(t, !term.Flags.Dirty.Clear, "!t.flags.dirty.clear")
}

// Zig: test "term51: decaln: works on a single row terminal"
func TestTm9_decaln_works_on_a_single_row_terminal(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 3, 1)
	defer term.Deinit()

	tm9PrintString(t, term, "abc")
	zigTry(t, term.Decaln())

	tm9ExpectGrid(t, term, []string{"EEE"})
	tm9ExpectCursor(t, term, 0, 0, false)
}

// Zig: test "term51: decaln: resets all margins"
func TestTm9_decaln_resets_all_margins(t *testing.T) {
	testStdCapacity(t)
	term := tm9FourAt(t, 3, 2)
	defer term.Deinit()

	term.ScrollingRegion = TerminalScrollingRegion{Top: 1, Bottom: 2, Left: 2, Right: 5}
	zigTry(t, term.Decaln())

	expectEqual(t, uint(0), uint(term.ScrollingRegion.Top))
	expectEqual(t, uint(3), uint(term.ScrollingRegion.Bottom))
	expectEqual(t, uint(0), uint(term.ScrollingRegion.Left))
	expectEqual(t, uint(7), uint(term.ScrollingRegion.Right))
	tm9ExpectGrid(t, term, []string{"EEEEEEEE", "EEEEEEEE", "EEEEEEEE", "EEEEEEEE"})
}

// Zig: test "term51: decaln: leaves the left and right margin mode set"
func TestTm9_decaln_leaves_the_left_and_right_margin_mode_set(t *testing.T) {
	testStdCapacity(t)
	term := tm9FourAt(t, 3, 2)
	defer term.Deinit()

	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(3, 5)
	zigTry(t, term.Decaln())

	expect(t, term.Modes.Get(ModesModeEnableLeftAndRightMargin), "t.modes.get(.enable_left_and_right_margin)")
	expectEqual(t, uint(0), uint(term.ScrollingRegion.Left))
	expectEqual(t, uint(7), uint(term.ScrollingRegion.Right))
}

// Zig: test "term51: decaln: turns origin mode off"
func TestTm9_decaln_turns_origin_mode_off(t *testing.T) {
	testStdCapacity(t)
	term := tm9FourAt(t, 3, 2)
	defer term.Deinit()

	term.Modes.Set(ModesModeOrigin, true)
	term.SetTopAndBottomMargin(2, 3)
	zigTry(t, term.Decaln())

	expect(t, !term.Modes.Get(ModesModeOrigin), "!t.modes.get(.origin)")
	tm9ExpectCursor(t, term, 0, 0, false)
	// With origin mode off the cursor addresses the whole screen.
	term.SetCursorPos(4, 8)
	tm9ExpectCursor(t, term, 7, 3, false)
}

// Zig: test "term51: decaln: replaces wide characters, graphemes and hyperlinks"
func TestTm9_decaln_replaces_wide_characters_graphemes_and_hyperlinks(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 8, 3)
	defer term.Deinit()

	tm9StartHyperlink(t, term.Screens.Active, "http://a.test")
	tm9PutStr(t, term, 0, 0, "abcdefg")
	zigTry(t, term.Print(0x4E2D))
	tm9PutGrapheme(t, term, 3, 1)
	tm9PutWideGrapheme(t, term, 4, 2)
	term.Screens.Active.EndHyperlink()
	page := tm9Page(term)
	expect(t, page.HyperlinkCount() > 0, "page.hyperlinkCount() > 0")
	expect(t, page.GraphemeCount() > 0, "page.graphemeCount() > 0")

	zigTry(t, term.Decaln())

	tm9ExpectGrid(t, term, []string{"EEEEEEEE", "EEEEEEEE", "EEEEEEEE"})
	expectEqual(t, uint(0), page.HyperlinkCount())
	expectEqual(t, uint(0), page.hyperlink_set.Count())
	expectEqual(t, uint(0), page.GraphemeCount())
	tm9ExpectWrap(t, term, "...")
	for y := uint(0); y < 3; y++ {
		expect(t, !tm9RowAt(t, term, y).hyperlink, "!term51URowAt(&t, y).hyperlink")
		expect(t, !tm9RowAt(t, term, y).grapheme, "!term51URowAt(&t, y).grapheme")
	}
}

// Zig: test "term51: decaln: does not carry the cursor hyperlink into the cells"
func TestTm9_decaln_does_not_carry_the_cursor_hyperlink_into_the_cells(t *testing.T) {
	testStdCapacity(t)
	term := tm9FourAt(t, 0, 0)
	defer term.Deinit()

	tm9StartHyperlink(t, term.Screens.Active, "http://a.test")
	zigTry(t, term.Decaln())

	expect(t, term.Screens.Active.Cursor.HyperlinkID != 0, "t.screens.active.cursor.hyperlink_id != 0")
	page := tm9Page(term)
	expectEqual(t, uint(0), page.HyperlinkCount())
	expect(t, !tm9CellAt(t, term, 0, 0).hyperlink, "!term51UCellAt(&t, 0, 0).hyperlink")
}

// Zig: test "term51: decaln: clears the style attributes but keeps the colors"
func TestTm9_decaln_clears_the_style_attributes_but_keeps_the_colors(t *testing.T) {
	testStdCapacity(t)
	term := tm9FourAt(t, 3, 2)
	defer term.Deinit()

	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagBold))
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagItalic))
	tm9SetAttribute(t, term, tm9Underline(SGRAttributeUnderlineSingle))
	tm9SetAttribute(t, term, SGRAttribute{Tag: SGRAttributeTag256Fg, X256Fg: 3})
	tm9SetAttribute(t, term, SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: 1, G: 2, B: 3}})
	zigTry(t, term.Decaln())

	cursor_style := term.Screens.Active.Cursor.Style
	expect(t, !cursor_style.Flags.Bold, "!cursor_style.flags.bold")
	expect(t, !cursor_style.Flags.Italic, "!cursor_style.flags.italic")
	expectEqual(t, SGRAttributeUnderlineNone, cursor_style.Flags.Underline)
	expectEqual(t, tm9PalColor(3), cursor_style.FgColor)
	expectEqual(t, tm9RgbColor(1, 2, 3), cursor_style.BgColor)

	// Every cell holds the style and the cursor holds one more reference.
	page := tm9Page(term)
	id := term.Screens.Active.Cursor.StyleID
	expect(t, id != StyleDefaultID, "id != style.default_id")
	expectEqual(t, uint(1), page.styles.Count())
	expectEqual(t, SizeCellCountInt(8*4+1), page.styles.RefCount(id))
	for y := uint(0); y < 4; y++ {
		expect(t, tm9RowAt(t, term, y).styled, "term51URowAt(&t, y).styled")
		for x := uint(0); x < 8; x++ {
			cell := *tm9CellAt(t, term, x, y)
			expectEqual(t, id, cell.style_id)
			expectEqual(t, PageCellContentTagCodepoint, cell.content_tag)
			expectEqual(t, uint32('E'), cell.content.codepoint)
		}
	}
}

// Zig: test "term51: decaln: a cursor style with only attributes becomes the default style"
func TestTm9_decaln_a_cursor_style_with_only_attributes_becomes_the_default_style(t *testing.T) {
	testStdCapacity(t)
	term := tm9FourAt(t, 3, 2)
	defer term.Deinit()

	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagBold))
	zigTry(t, term.Decaln())

	expectEqual(t, StyleDefaultID, term.Screens.Active.Cursor.StyleID)
	page := tm9Page(term)
	expectEqual(t, uint(0), page.styles.Count())
	for y := uint(0); y < 4; y++ {
		expect(t, !tm9RowAt(t, term, y).styled, "!term51URowAt(&t, y).styled")
	}
	tm9ExpectGrid(t, term, []string{"EEEEEEEE", "EEEEEEEE", "EEEEEEEE", "EEEEEEEE"})
}

// Zig: test "term51: decaln: releases the styles of the previous content"
func TestTm9_decaln_releases_the_styles_of_the_previous_content(t *testing.T) {
	testStdCapacity(t)
	term := tm9FourAt(t, 0, 0)
	defer term.Deinit()

	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagBold))
	tm9PutStr(t, term, 0, 1, "abcdefgh")
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagUnset))
	page := tm9Page(term)
	expectEqual(t, uint(1), page.styles.Count())

	zigTry(t, term.Decaln())

	expectEqual(t, uint(0), page.styles.Count())
}

// Zig: test "term51: decaln: does not respect protection and keeps the cursor protected"
func TestTm9_decaln_does_not_respect_protection_and_keeps_the_cursor_protected(t *testing.T) {
	testStdCapacity(t)
	term := tm9FourAt(t, 0, 0)
	defer term.Deinit()

	tm9ProtectAt(t, term, ANSIProtectedModeIso, 3, 2)
	term.SetProtectedMode(ANSIProtectedModeIso)
	zigTry(t, term.Decaln())

	tm9ExpectGrid(t, term, []string{"EEEEEEEE", "EEEEEEEE", "EEEEEEEE", "EEEEEEEE"})
	for y := uint(0); y < 4; y++ {
		for x := uint(0); x < 8; x++ {
			expect(t, !tm9CellAt(t, term, x, y).protected, "!term51UCellAt(&t, x, y).protected")
		}
	}
	expect(t, term.Screens.Active.Cursor.Protected, "t.screens.active.cursor.protected")
	expectEqual(t, ANSIProtectedModeIso, term.Screens.Active.ProtectedMode)
}

// Zig: test "term51: decaln: resets the row attributes"
func TestTm9_decaln_resets_the_row_attributes(t *testing.T) {
	testStdCapacity(t)
	term := tm9FourAt(t, 0, 0)
	defer term.Deinit()

	tm9SetWrap(t, term, "bwcb")
	tm9RowAt(t, term, 1).semantic_prompt = PageRowSemanticPromptPrompt
	zigTry(t, term.Decaln())

	tm9ExpectWrap(t, term, "....")
	expectEqual(t, PageRowSemanticPromptNone, tm9RowAt(t, term, 1).semantic_prompt)
}

// Zig: test "term51: decaln: leaves the scrollback alone"
func TestTm9_decaln_leaves_the_scrollback_alone(t *testing.T) {
	testStdCapacity(t)
	term := tm9ScrollbackTerm(t)
	defer term.Deinit()

	zigTry(t, term.Decaln())

	tm9ExpectGrid(t, term, []string{"EEEEEEEE", "EEEEEEEE", "EEEEEEEE"})
	expectEqual(t, uint(3), tm9HistoryRows(term))
	tm9ExpectHistoryRow(t, term, 0, "line0...")
	tm9ExpectHistoryRow(t, term, 2, "line2...")
}

// Zig: test "term51: decaln: moves the cursor out of the way of the fill and back"
func TestTm9_decaln_moves_the_cursor_out_of_the_way_of_the_fill_and_back(t *testing.T) {
	testStdCapacity(t)
	term := tm9FourAt(t, 7, 3)
	defer term.Deinit()

	term.Screens.Active.Cursor.PendingWrap = true
	zigTry(t, term.Decaln())

	tm9ExpectCursor(t, term, 0, 0, false)
	// The cursor is usable: printing continues from the top left.
	zigTry(t, term.Print('x'))
	tm9ExpectGrid(t, term, []string{"xEEEEEEE", "EEEEEEEE", "EEEEEEEE", "EEEEEEEE"})
}

// Zig: test "term51: deleteChars: cursor on a spacer tail at the left margin clears the wide character outside it"
func TestTm9_deleteChars_cursor_on_a_spacer_tail_at_the_left_margin_clears_the_wide_character_outside_it(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 8, 2)
	defer term.Deinit()

	tm9WideRow(t, term)
	term.ScrollingRegion.Left = 3
	term.SetCursorPos(1, 4)
	term.DeleteChars(1)

	tm9ExpectGrid(t, term, []string{"ab.cdef.", "........"})
}

// The test "term51: kittyGraphics: compiled out in the library build" is not
// translated; see scripts/not-translated.txt.

// Zig: test "term51: setAttribute: every setting attribute from the default style"
func TestTm9_setAttribute_every_setting_attribute_from_the_default_style(t *testing.T) {
	testStdCapacity(t)
	cases := []tm9AttrCase{
		{tm9Attr(SGRAttributeTagBold), Style{Flags: StyleFlags{Bold: true}}},
		{tm9Attr(SGRAttributeTagItalic), Style{Flags: StyleFlags{Italic: true}}},
		{tm9Attr(SGRAttributeTagFaint), Style{Flags: StyleFlags{Faint: true}}},
		{tm9Attr(SGRAttributeTagBlink), Style{Flags: StyleFlags{Blink: true}}},
		{tm9Attr(SGRAttributeTagInverse), Style{Flags: StyleFlags{Inverse: true}}},
		{tm9Attr(SGRAttributeTagInvisible), Style{Flags: StyleFlags{Invisible: true}}},
		{tm9Attr(SGRAttributeTagStrikethrough), Style{Flags: StyleFlags{Strikethrough: true}}},
		{tm9Attr(SGRAttributeTagOverline), Style{Flags: StyleFlags{Overline: true}}},
		{tm9Underline(SGRAttributeUnderlineNone), Style{}},
		{tm9Underline(SGRAttributeUnderlineSingle), Style{Flags: StyleFlags{Underline: SGRAttributeUnderlineSingle}}},
		{tm9Underline(SGRAttributeUnderlineDouble), Style{Flags: StyleFlags{Underline: SGRAttributeUnderlineDouble}}},
		{tm9Underline(SGRAttributeUnderlineCurly), Style{Flags: StyleFlags{Underline: SGRAttributeUnderlineCurly}}},
		{tm9Underline(SGRAttributeUnderlineDotted), Style{Flags: StyleFlags{Underline: SGRAttributeUnderlineDotted}}},
		{tm9Underline(SGRAttributeUnderlineDashed), Style{Flags: StyleFlags{Underline: SGRAttributeUnderlineDashed}}},
		{
			SGRAttribute{Tag: SGRAttributeTagUnderlineColor, UnderlineColor: ColorRGB{R: 1, G: 2, B: 3}},
			Style{underline_color: tm9RgbColor(1, 2, 3)},
		},
		{
			SGRAttribute{Tag: SGRAttributeTagUnderlineColor, UnderlineColor: ColorRGB{R: 0, G: 0, B: 0}},
			Style{underline_color: tm9RgbColor(0, 0, 0)},
		},
		{
			SGRAttribute{Tag: SGRAttributeTag256UnderlineColor, X256UnderlineColor: 0},
			Style{underline_color: tm9PalColor(0)},
		},
		{
			SGRAttribute{Tag: SGRAttributeTag256UnderlineColor, X256UnderlineColor: 255},
			Style{underline_color: tm9PalColor(255)},
		},
		{
			SGRAttribute{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: ColorRGB{R: 255, G: 0, B: 127}},
			Style{FgColor: tm9RgbColor(255, 0, 127)},
		},
		{
			SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: 255, G: 0, B: 127}},
			Style{BgColor: tm9RgbColor(255, 0, 127)},
		},
		{SGRAttribute{Tag: SGRAttributeTag256Fg, X256Fg: 0}, Style{FgColor: tm9PalColor(0)}},
		{SGRAttribute{Tag: SGRAttributeTag256Fg, X256Fg: 16}, Style{FgColor: tm9PalColor(16)}},
		{SGRAttribute{Tag: SGRAttributeTag256Fg, X256Fg: 255}, Style{FgColor: tm9PalColor(255)}},
		{SGRAttribute{Tag: SGRAttributeTag256Bg, X256Bg: 0}, Style{BgColor: tm9PalColor(0)}},
		{SGRAttribute{Tag: SGRAttributeTag256Bg, X256Bg: 16}, Style{BgColor: tm9PalColor(16)}},
		{SGRAttribute{Tag: SGRAttributeTag256Bg, X256Bg: 255}, Style{BgColor: tm9PalColor(255)}},
	}

	for _, c := range cases {
		got := tm9Apply(t, c.attr)
		expect(t, got.Eql(c.expect), "got.eql(c.expect)")
	}
}

// Zig: test "term51: setAttribute: named colors map to their palette index"
func TestTm9_setAttribute_named_colors_map_to_their_palette_index(t *testing.T) {
	testStdCapacity(t)
	// Every color.Name value maps to the palette index equal to its integer
	// value, for the 8_* and the 8_bright_* attributes alike (the bright
	// attributes do NOT add 8: the caller passes bright_* names).
	names := []ColorName{
		ColorNameBlack, ColorNameRed, ColorNameGreen, ColorNameYellow,
		ColorNameBlue, ColorNameMagenta, ColorNameCyan, ColorNameWhite,
		ColorNameBrightBlack, ColorNameBrightRed, ColorNameBrightGreen, ColorNameBrightYellow,
		ColorNameBrightBlue, ColorNameBrightMagenta, ColorNameBrightCyan, ColorNameBrightWhite,
	}
	for i, name := range names {
		idx := uint8(i)
		expect(t, tm9Apply(t, SGRAttribute{Tag: SGRAttributeTag8Fg, X8Fg: name}).Eql(Style{FgColor: tm9PalColor(idx)}), "(try term51VApply(.{ .@\"8_fg\" = name })).eql(.{ .fg_color = .{ .palette = idx } })")
		expect(t, tm9Apply(t, SGRAttribute{Tag: SGRAttributeTag8Bg, X8Bg: name}).Eql(Style{BgColor: tm9PalColor(idx)}), "(try term51VApply(.{ .@\"8_bg\" = name })).eql(.{ .bg_color = .{ .palette = idx } })")
		expect(t, tm9Apply(t, SGRAttribute{Tag: SGRAttributeTag8BrightFg, X8BrightFg: name}).Eql(Style{FgColor: tm9PalColor(idx)}), "(try term51VApply(.{ .@\"8_bright_fg\" = name })).eql(.{ .fg_color = .{ .palette = idx } })")
		expect(t, tm9Apply(t, SGRAttribute{Tag: SGRAttributeTag8BrightBg, X8BrightBg: name}).Eql(Style{BgColor: tm9PalColor(idx)}), "(try term51VApply(.{ .@\"8_bright_bg\" = name })).eql(.{ .bg_color = .{ .palette = idx } })")
	}

	// color.Name is non-exhaustive: other values are plain palette indexes.
	high := ColorName(200)
	expect(t, tm9Apply(t, SGRAttribute{Tag: SGRAttributeTag8Fg, X8Fg: high}).Eql(Style{FgColor: tm9PalColor(200)}), "(try term51VApply(.{ .@\"8_fg\" = high })).eql(.{ .fg_color = .{ .palette = 200 } })")
	expect(t, tm9Apply(t, SGRAttribute{Tag: SGRAttributeTag8Bg, X8Bg: high}).Eql(Style{BgColor: tm9PalColor(200)}), "(try term51VApply(.{ .@\"8_bg\" = high })).eql(.{ .bg_color = .{ .palette = 200 } })")
	expect(t, tm9Apply(t, SGRAttribute{Tag: SGRAttributeTag8BrightFg, X8BrightFg: high}).Eql(Style{FgColor: tm9PalColor(200)}), "(try term51VApply(.{ .@\"8_bright_fg\" = high })).eql(.{ .fg_color = .{ .palette = 200 } })")
	expect(t, tm9Apply(t, SGRAttribute{Tag: SGRAttributeTag8BrightBg, X8BrightBg: high}).Eql(Style{BgColor: tm9PalColor(200)}), "(try term51VApply(.{ .@\"8_bright_bg\" = high })).eql(.{ .bg_color = .{ .palette = 200 } })")
}

// Zig: test "term51: setAttribute: resetting attributes change only their own field"
func TestTm9_setAttribute_resetting_attributes_change_only_their_own_field(t *testing.T) {
	testStdCapacity(t)
	type Case struct {
		attr SGRAttribute
		// Applied to the full style to get the expected result.
		mutate func(s *Style)
	}
	cases := []Case{
		{
			attr: tm9Attr(SGRAttributeTagResetBold),
			mutate: func(s *Style) {
				// Bold and faint share a reset.
				s.Flags.Bold = false
				s.Flags.Faint = false
			},
		},
		{attr: tm9Attr(SGRAttributeTagResetItalic), mutate: func(s *Style) {
			s.Flags.Italic = false
		}},
		{attr: tm9Underline(SGRAttributeUnderlineNone), mutate: func(s *Style) {
			s.Flags.Underline = SGRAttributeUnderlineNone
		}},
		{attr: tm9Attr(SGRAttributeTagResetUnderlineColor), mutate: func(s *Style) {
			s.underline_color = StyleColor{Tag: StyleColorNone}
		}},
		{attr: tm9Attr(SGRAttributeTagResetOverline), mutate: func(s *Style) {
			s.Flags.Overline = false
		}},
		{attr: tm9Attr(SGRAttributeTagResetBlink), mutate: func(s *Style) {
			s.Flags.Blink = false
		}},
		{attr: tm9Attr(SGRAttributeTagResetInverse), mutate: func(s *Style) {
			s.Flags.Inverse = false
		}},
		{attr: tm9Attr(SGRAttributeTagResetInvisible), mutate: func(s *Style) {
			s.Flags.Invisible = false
		}},
		{attr: tm9Attr(SGRAttributeTagResetStrikethrough), mutate: func(s *Style) {
			s.Flags.Strikethrough = false
		}},
		{attr: tm9Attr(SGRAttributeTagResetFg), mutate: func(s *Style) {
			s.FgColor = StyleColor{Tag: StyleColorNone}
		}},
		{attr: tm9Attr(SGRAttributeTagResetBg), mutate: func(s *Style) {
			s.BgColor = StyleColor{Tag: StyleColorNone}
		}},
	}

	for _, c := range cases {
		func() {
			term := tm9Init(t, 5, 5)
			defer term.Deinit()
			tm9FullStyle(t, term)
			expect(t, term.Screens.Active.Cursor.Style.Eql(tm9FullStyleValue()), "t.screens.active.cursor.style.eql(term51VFullStyleValue())")

			expected := tm9FullStyleValue()
			c.mutate(&expected)
			tm9SetAttribute(t, term, c.attr)
			expect(t, term.Screens.Active.Cursor.Style.Eql(expected), "t.screens.active.cursor.style.eql(expected)")
			expect(t, term.Screens.Active.Cursor.StyleID != StyleDefaultID, "t.screens.active.cursor.style_id != style.default_id")
		}()
	}
}

// Zig: test "term51: setAttribute: resets on an already default style keep it default"
func TestTm9_setAttribute_resets_on_an_already_default_style_keep_it_default(t *testing.T) {
	testStdCapacity(t)
	attrs := []SGRAttribute{
		tm9Attr(SGRAttributeTagResetBold),
		tm9Attr(SGRAttributeTagResetItalic),
		tm9Underline(SGRAttributeUnderlineNone),
		tm9Attr(SGRAttributeTagResetUnderlineColor),
		tm9Attr(SGRAttributeTagResetOverline),
		tm9Attr(SGRAttributeTagResetBlink),
		tm9Attr(SGRAttributeTagResetInverse),
		tm9Attr(SGRAttributeTagResetInvisible),
		tm9Attr(SGRAttributeTagResetStrikethrough),
		tm9Attr(SGRAttributeTagResetFg),
		tm9Attr(SGRAttributeTagResetBg),
		tm9Attr(SGRAttributeTagUnset),
	}
	term := tm9Init(t, 5, 5)
	defer term.Deinit()
	for _, attr := range attrs {
		tm9SetAttribute(t, term, attr)
		expect(t, term.Screens.Active.Cursor.Style.Default(), "t.screens.active.cursor.style.default()")
		expectEqual(t, StyleDefaultID, term.Screens.Active.Cursor.StyleID)
		page := &term.Screens.Active.Cursor.PagePin.Node.data
		expectEqual(t, uint(0), page.styles.Count())
	}
}

// Zig: test "term51: setAttribute: reset_bold also clears faint and nothing clears faint alone"
func TestTm9_setAttribute_reset_bold_also_clears_faint_and_nothing_clears_faint_alone(t *testing.T) {
	testStdCapacity(t)
	// reset_bold clears faint as well, but there is no separate reset for
	// faint: the only way to clear faint alone is unset.
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagFaint))
	expect(t, term.Screens.Active.Cursor.Style.Flags.Faint, "t.screens.active.cursor.style.flags.faint")
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagBold))
	expect(t, term.Screens.Active.Cursor.Style.Flags.Bold, "t.screens.active.cursor.style.flags.bold")
	expect(t, term.Screens.Active.Cursor.Style.Flags.Faint, "t.screens.active.cursor.style.flags.faint")
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagResetBold))
	expect(t, !term.Screens.Active.Cursor.Style.Flags.Bold, "!t.screens.active.cursor.style.flags.bold")
	expect(t, !term.Screens.Active.Cursor.Style.Flags.Faint, "!t.screens.active.cursor.style.flags.faint")

	// reset_bold with only faint set also clears faint.
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagFaint))
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagResetBold))
	expect(t, term.Screens.Active.Cursor.Style.Default(), "t.screens.active.cursor.style.default()")
}

// Zig: test "term51: setAttribute: unset clears every attribute and releases the style"
func TestTm9_setAttribute_unset_clears_every_attribute_and_releases_the_style(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	tm9FullStyle(t, term)
	{
		page := &term.Screens.Active.Cursor.PagePin.Node.data
		expectEqual(t, uint(1), page.styles.Count())
		expect(t, term.Screens.Active.Cursor.StyleID != StyleDefaultID, "t.screens.active.cursor.style_id != style.default_id")
	}

	// Protected mode is cursor state, not style: unset leaves it alone.
	term.SetProtectedMode(ANSIProtectedModeIso)
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagUnset))
	expect(t, term.Screens.Active.Cursor.Style.Default(), "t.screens.active.cursor.style.default()")
	expectEqual(t, StyleDefaultID, term.Screens.Active.Cursor.StyleID)
	expect(t, term.Screens.Active.Cursor.Protected, "t.screens.active.cursor.protected")
	{
		page := &term.Screens.Active.Cursor.PagePin.Node.data
		expectEqual(t, uint(0), page.styles.Count())
	}
}

// Zig: test "term51: setAttribute: unknown attribute changes nothing"
func TestTm9_setAttribute_unknown_attribute_changes_nothing(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	// On the default style.
	tm9SetAttribute(t, term, SGRAttribute{Tag: SGRAttributeTagUnknown, Unknown: SGRAttributeUnknown{Full: []uint16{38, 9}, Partial: []uint16{9}}})
	expect(t, term.Screens.Active.Cursor.Style.Default(), "t.screens.active.cursor.style.default()")
	expectEqual(t, StyleDefaultID, term.Screens.Active.Cursor.StyleID)

	// On a non-default style the style and its id are untouched.
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagBold))
	id := term.Screens.Active.Cursor.StyleID
	expect(t, id != StyleDefaultID, "id != style.default_id")
	tm9SetAttribute(t, term, SGRAttribute{Tag: SGRAttributeTagUnknown, Unknown: SGRAttributeUnknown{Full: []uint16{}, Partial: []uint16{}}})
	expectEqual(t, id, term.Screens.Active.Cursor.StyleID)
	expect(t, term.Screens.Active.Cursor.Style.Eql(Style{Flags: StyleFlags{Bold: true}}), "t.screens.active.cursor.style.eql(.{ .flags = .{ .bold = true } })")
}

// Zig: test "term51: setAttribute: style ids are reference counted in the page"
func TestTm9_setAttribute_style_ids_are_reference_counted_in_the_page(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()
	page := &term.Screens.Active.Cursor.PagePin.Node.data

	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagBold))
	bold_id := term.Screens.Active.Cursor.StyleID
	expect(t, bold_id != StyleDefaultID, "bold_id != style.default_id")
	expectEqual(t, uint(1), page.styles.Count())

	// Same attribute again: same style, same id.
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagBold))
	expectEqual(t, bold_id, term.Screens.Active.Cursor.StyleID)
	expectEqual(t, uint(1), page.styles.Count())

	// A different style replaces the old one (the old one is released).
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagItalic))
	expect(t, term.Screens.Active.Cursor.Style.Eql(Style{Flags: StyleFlags{Bold: true, Italic: true}}), "t.screens.active.cursor.style.eql(.{ .flags = .{ .bold = true, .italic = true } })")
	expectEqual(t, uint(1), page.styles.Count())

	// A style used by a cell stays alive after the cursor moves on.
	zigTry(t, term.Print('X'))
	cell_id := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm9Active(0, 0))).Cell.style_id
	expectEqual(t, term.Screens.Active.Cursor.StyleID, cell_id)
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagUnset))
	expectEqual(t, StyleDefaultID, term.Screens.Active.Cursor.StyleID)
	expectEqual(t, uint(1), page.styles.Count())
	cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm9Active(0, 0)))
	expect(t, cell.Style().Eql(Style{Flags: StyleFlags{Bold: true, Italic: true}}), "cell.style().eql(.{ .flags = .{ .bold = true, .italic = true } })")
}

// Zig: test "term51: setAttribute: applies to the active screen only"
func TestTm9_setAttribute_applies_to_the_active_screen_only(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagBold))
	primary := term.Screens.Active
	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	expect(t, term.Screens.Active != primary, "t.screens.active != primary")
	// switchScreen does not copy the cursor, so the alt style is default.
	expect(t, term.Screens.Active.Cursor.Style.Default(), "t.screens.active.cursor.style.default()")

	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagItalic))
	expect(t, term.Screens.Active.Cursor.Style.Eql(Style{Flags: StyleFlags{Italic: true}}), "t.screens.active.cursor.style.eql(.{ .flags = .{ .italic = true } })")
	expect(t, primary.Cursor.Style.Eql(Style{Flags: StyleFlags{Bold: true}}), "primary.cursor.style.eql(.{ .flags = .{ .bold = true } })")
}

// Zig: test "term51: printAttributes: no attributes prints a lone zero"
func TestTm9_printAttributes_no_attributes_prints_a_lone_zero(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()
	storage := make([]uint8, 64)
	buf, err := term.PrintAttributes(storage)
	zigTry(t, err)
	expectEqualStrings(t, "0", string(buf))
	// The result is a slice of the caller's buffer.
	expect(t, &storage[0] == &buf[0], "@intFromPtr(&storage[0]) == @intFromPtr(buf.ptr)")
}

// tm9PrintAttributes is `try t.printAttributes(&storage)`.
func tm9PrintAttributes(t *testing.T, term *Terminal, storage []uint8) string {
	t.Helper()
	buf, err := term.PrintAttributes(storage)
	zigTry(t, err)
	return string(buf)
}

// Zig: test "term51: printAttributes: each flag alone"
func TestTm9_printAttributes_each_flag_alone(t *testing.T) {
	testStdCapacity(t)
	type Case struct {
		attr   SGRAttribute
		expect string
	}
	cases := []Case{
		{tm9Attr(SGRAttributeTagBold), "0;1"},
		{tm9Attr(SGRAttributeTagFaint), "0;2"},
		{tm9Attr(SGRAttributeTagItalic), "0;3"},
		{tm9Underline(SGRAttributeUnderlineSingle), "0;4"},
		{tm9Underline(SGRAttributeUnderlineDouble), "0;4"},
		{tm9Underline(SGRAttributeUnderlineCurly), "0;4"},
		{tm9Underline(SGRAttributeUnderlineDotted), "0;4"},
		{tm9Underline(SGRAttributeUnderlineDashed), "0;4"},
		{tm9Attr(SGRAttributeTagBlink), "0;5"},
		{tm9Attr(SGRAttributeTagInverse), "0;7"},
		{tm9Attr(SGRAttributeTagInvisible), "0;8"},
		{tm9Attr(SGRAttributeTagStrikethrough), "0;9"},
	}
	for _, c := range cases {
		func() {
			term := tm9Init(t, 5, 5)
			defer term.Deinit()
			tm9SetAttribute(t, term, c.attr)
			storage := make([]uint8, 64)
			expectEqualStrings(t, c.expect, tm9PrintAttributes(t, term, storage))
		}()
	}
}

// Zig: test "term51: printAttributes: overline and underline color are not reported"
func TestTm9_printAttributes_overline_and_underline_color_are_not_reported(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()
	storage := make([]uint8, 64)

	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagOverline))
	expectEqualStrings(t, "0", tm9PrintAttributes(t, term, storage))
	tm9SetAttribute(t, term, SGRAttribute{Tag: SGRAttributeTagUnderlineColor, UnderlineColor: ColorRGB{R: 1, G: 2, B: 3}})
	expectEqualStrings(t, "0", tm9PrintAttributes(t, term, storage))
	tm9SetAttribute(t, term, SGRAttribute{Tag: SGRAttributeTag256UnderlineColor, X256UnderlineColor: 5})
	expectEqualStrings(t, "0", tm9PrintAttributes(t, term, storage))

	// They do not disturb the attributes that are reported.
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagBold))
	expectEqualStrings(t, "0;1", tm9PrintAttributes(t, term, storage))
}

// Zig: test "term51: printAttributes: flags print in fixed order not setting order"
func TestTm9_printAttributes_flags_print_in_fixed_order_not_setting_order(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()
	storage := make([]uint8, 64)

	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagStrikethrough))
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagInvisible))
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagInverse))
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagBlink))
	tm9SetAttribute(t, term, tm9Underline(SGRAttributeUnderlineDashed))
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagItalic))
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagFaint))
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagBold))
	expectEqualStrings(t, "0;1;2;3;4;5;7;8;9", tm9PrintAttributes(t, term, storage))

	// A subset keeps the order and skips the gaps.
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagUnset))
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagStrikethrough))
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagItalic))
	expectEqualStrings(t, "0;3;9", tm9PrintAttributes(t, term, storage))
}

// Zig: test "term51: printAttributes: foreground palette boundaries"
func TestTm9_printAttributes_foreground_palette_boundaries(t *testing.T) {
	testStdCapacity(t)
	type Case struct {
		idx    uint8
		expect string
	}
	cases := []Case{
		{0, "0;30"},
		{1, "0;31"},
		{7, "0;37"},
		{8, "0;90"},
		{9, "0;91"},
		{15, "0;97"},
		{16, "0;38:5:16"},
		{17, "0;38:5:17"},
		{255, "0;38:5:255"},
	}
	for _, c := range cases {
		func() {
			term := tm9Init(t, 5, 5)
			defer term.Deinit()
			tm9SetAttribute(t, term, SGRAttribute{Tag: SGRAttributeTag256Fg, X256Fg: c.idx})
			storage := make([]uint8, 64)
			expectEqualStrings(t, c.expect, tm9PrintAttributes(t, term, storage))
		}()
	}
}

// Zig: test "term51: printAttributes: background palette boundaries"
func TestTm9_printAttributes_background_palette_boundaries(t *testing.T) {
	testStdCapacity(t)
	type Case struct {
		idx    uint8
		expect string
	}
	cases := []Case{
		{0, "0;40"},
		{1, "0;41"},
		{7, "0;47"},
		{8, "0;100"},
		{9, "0;101"},
		{15, "0;107"},
		{16, "0;48:5:16"},
		{17, "0;48:5:17"},
		{255, "0;48:5:255"},
	}
	for _, c := range cases {
		func() {
			term := tm9Init(t, 5, 5)
			defer term.Deinit()
			tm9SetAttribute(t, term, SGRAttribute{Tag: SGRAttributeTag256Bg, X256Bg: c.idx})
			storage := make([]uint8, 64)
			expectEqualStrings(t, c.expect, tm9PrintAttributes(t, term, storage))
		}()
	}
}

// Zig: test "term51: printAttributes: rgb colors and the full combination"
func TestTm9_printAttributes_rgb_colors_and_the_full_combination(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()
	storage := make([]uint8, 128)

	tm9SetAttribute(t, term, SGRAttribute{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: ColorRGB{R: 0, G: 0, B: 0}})
	expectEqualStrings(t, "0;38:2::0:0:0", tm9PrintAttributes(t, term, storage))
	tm9SetAttribute(t, term, SGRAttribute{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: ColorRGB{R: 255, G: 255, B: 255}})
	expectEqualStrings(t, "0;38:2::255:255:255", tm9PrintAttributes(t, term, storage))
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagResetFg))

	tm9SetAttribute(t, term, SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: 0, G: 0, B: 0}})
	expectEqualStrings(t, "0;48:2::0:0:0", tm9PrintAttributes(t, term, storage))
	tm9SetAttribute(t, term, SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: 255, G: 1, B: 20}})
	expectEqualStrings(t, "0;48:2::255:1:20", tm9PrintAttributes(t, term, storage))

	// Foreground comes before background, both after the flags.
	tm9SetAttribute(t, term, SGRAttribute{Tag: SGRAttributeTag256Fg, X256Fg: 9})
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagBold))
	expectEqualStrings(t, "0;1;91;48:2::255:1:20", tm9PrintAttributes(t, term, storage))
	tm9SetAttribute(t, term, SGRAttribute{Tag: SGRAttributeTag256Bg, X256Bg: 200})
	expectEqualStrings(t, "0;1;91;48:5:200", tm9PrintAttributes(t, term, storage))
	tm9SetAttribute(t, term, SGRAttribute{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: ColorRGB{R: 7, G: 8, B: 9}})
	tm9SetAttribute(t, term, SGRAttribute{Tag: SGRAttributeTag256Bg, X256Bg: 3})
	expectEqualStrings(t, "0;1;38:2::7:8:9;43", tm9PrintAttributes(t, term, storage))
}

// Zig: test "term51: printAttributes: buffer capacity boundary"
func TestTm9_printAttributes_buffer_capacity_boundary(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	// Zero-length buffer: even the leading "0" does not fit.
	empty := make([]uint8, 0)
	_, err := term.PrintAttributes(empty)
	tm9ExpectError(t, ErrNoSpaceLeft, err)

	// "0" fits exactly in one byte.
	one := make([]uint8, 1)
	expectEqualStrings(t, "0", tm9PrintAttributes(t, term, one))

	// "0;1" is three bytes: two bytes fail, three succeed.
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagBold))
	two := make([]uint8, 2)
	_, err = term.PrintAttributes(two)
	tm9ExpectError(t, ErrNoSpaceLeft, err)
	three := make([]uint8, 3)
	expectEqualStrings(t, "0;1", tm9PrintAttributes(t, term, three))

	// "0;1;38:5:200" is 12 bytes: 11 fail, 12 succeed.
	tm9SetAttribute(t, term, SGRAttribute{Tag: SGRAttributeTag256Fg, X256Fg: 200})
	eleven := make([]uint8, 11)
	_, err = term.PrintAttributes(eleven)
	tm9ExpectError(t, ErrNoSpaceLeft, err)
	twelve := make([]uint8, 12)
	expectEqualStrings(t, "0;1;38:5:200", tm9PrintAttributes(t, term, twelve))
}

// Zig: test "term51: printAttributes: reads the active screen's cursor"
func TestTm9_printAttributes_reads_the_active_screen_s_cursor(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()
	storage := make([]uint8, 64)

	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagBold))
	expectEqualStrings(t, "0;1", tm9PrintAttributes(t, term, storage))
	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	expectEqualStrings(t, "0", tm9PrintAttributes(t, term, storage))
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagItalic))
	expectEqualStrings(t, "0;3", tm9PrintAttributes(t, term, storage))
	_, err = term.SwitchScreen(ScreenSetKeyPrimary)
	zigTry(t, err)
	expectEqualStrings(t, "0;1", tm9PrintAttributes(t, term, storage))
}

// Zig: test "term51: setPwd: stores a copy and getPwd returns it"
func TestTm9_setPwd_stores_a_copy_and_getPwd_returns_it(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	expect(t, !term.GetPwd().Valid, "t.getPwd() == null")

	src := []uint8("file://host/tmp")
	zigTry(t, term.SetPwd(src))
	expectEqualStrings(t, "file://host/tmp", string(zigUnwrap(t, term.GetPwd())))
	// It is a copy owned by the terminal.
	src[0] = 'X'
	expectEqualStrings(t, "file://host/tmp", string(zigUnwrap(t, term.GetPwd())))
	expect(t, &zigUnwrap(t, term.GetPwd())[0] != &src[0], "t.getPwd().?.ptr != @as([]const u8, &src).ptr")
}

// Zig: test "term51: setPwd: replaces the previous value"
func TestTm9_setPwd_replaces_the_previous_value(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.SetPwd([]uint8("/a/long/directory/name")))
	expectEqualStrings(t, "/a/long/directory/name", string(zigUnwrap(t, term.GetPwd())))
	zigTry(t, term.SetPwd([]uint8("/b")))
	expectEqualStrings(t, "/b", string(zigUnwrap(t, term.GetPwd())))
	zigTry(t, term.SetPwd([]uint8("/a/much/longer/directory/name/than/before/to/force/growth")))
	expectEqualStrings(t, "/a/much/longer/directory/name/than/before/to/force/growth", string(zigUnwrap(t, term.GetPwd())))
}

// Zig: test "term51: setPwd: empty pwd clears it"
func TestTm9_setPwd_empty_pwd_clears_it(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.SetPwd([]uint8("/tmp")))
	expect(t, term.GetPwd().Valid, "t.getPwd() != null")
	zigTry(t, term.SetPwd([]uint8("")))
	expect(t, !term.GetPwd().Valid, "t.getPwd() == null")
	expectEqual(t, uint(0), uint(len(term.Pwd)))

	// Setting empty on an unset pwd is fine.
	zigTry(t, term.SetPwd([]uint8("")))
	expect(t, !term.GetPwd().Valid, "t.getPwd() == null")
}

// Zig: test "term51: setPwd: bytes are stored verbatim"
func TestTm9_setPwd_bytes_are_stored_verbatim(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	// No validation: NUL, invalid UTF-8 and control bytes are kept.
	raw := []uint8("a\x00b\xff\xfe\n\x1b")
	zigTry(t, term.SetPwd(raw))
	expectEqualSlices(t, raw, zigUnwrap(t, term.GetPwd()))

	// A long value (larger than any fixed buffer in sight).
	long := make([]uint8, 10000)
	for i := range long {
		long[i] = 'x'
	}
	zigTry(t, term.SetPwd(long))
	expectEqual(t, uint(10000), uint(len(zigUnwrap(t, term.GetPwd()))))
	expectEqualStrings(t, string(long), string(zigUnwrap(t, term.GetPwd())))
}

// Zig: test "term51: setPwd: layout clearing retains capacity"
func TestTm9_setPwd_layout_clearing_retains_capacity(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	hundred := make([]uint8, 100)
	for i := range hundred {
		hundred[i] = 'x'
	}
	zigTry(t, term.SetPwd(hundred))
	cap_ := uint(cap(term.Pwd))
	expect(t, cap_ >= 100, "cap >= 100")
	zigTry(t, term.SetPwd([]uint8("y")))
	expectEqual(t, cap_, uint(cap(term.Pwd)))
	zigTry(t, term.SetPwd([]uint8("")))
	expectEqual(t, cap_, uint(cap(term.Pwd)))
}

// The test "term51: setPwd: allocation failure keeps the pwd empty" is not
// translated; see scripts/not-translated.txt.

// Zig: test "term51: getPwd: pointer aliases the terminal's storage"
func TestTm9_getPwd_pointer_aliases_the_terminal_s_storage(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()
	zigTry(t, term.SetPwd([]uint8("/home")))
	a := zigUnwrap(t, term.GetPwd())
	expect(t, &term.Pwd[0] == &a[0], "@intFromPtr(t.pwd.items.ptr) == @intFromPtr(a.ptr)")
	expectEqual(t, uint(5), uint(len(a)))
	// getPwd works through a const pointer.
	ct := term
	expectEqualStrings(t, "/home", string(zigUnwrap(t, ct.GetPwd())))
}

// Zig: test "term51: isDirty: tracks the row of a point"
func TestTm9_isDirty_tracks_the_row_of_a_point(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()
	term.clearDirty()
	for y := SizeCellCountInt(0); y < 5; y++ {
		for x := SizeCellCountInt(0); x < 5; x++ {
			expect(t, !term.isDirty(tm9Active(x, uint32(y))), "!t.isDirty(.{ .active = .{ .x = @intCast(x), .y = @intCast(y) } })")
		}
	}

	// Printing dirties the whole row, no other row.
	zigTry(t, term.Print('A'))
	for x := SizeCellCountInt(0); x < 5; x++ {
		expect(t, term.isDirty(tm9Active(x, 0)), "t.isDirty(.{ .active = .{ .x = @intCast(x), .y = 0 } })")
		expect(t, !term.isDirty(tm9Active(x, 1)), "!t.isDirty(.{ .active = .{ .x = @intCast(x), .y = 1 } })")
		expect(t, !term.isDirty(tm9Active(x, 4)), "!t.isDirty(.{ .active = .{ .x = @intCast(x), .y = 4 } })")
	}
}

// Zig: test "term51: isDirty: page level dirty flag marks every cell"
func TestTm9_isDirty_page_level_dirty_flag_marks_every_cell(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()
	term.clearDirty()
	page := &term.Screens.Active.Pages.pages.first.data
	expect(t, !page.dirty, "!page.dirty")

	page.dirty = true
	for y := uint32(0); y < 5; y++ {
		expect(t, term.isDirty(tm9Active(0, y)), "t.isDirty(.{ .active = .{ .x = 0, .y = @intCast(y) } })")
		expect(t, term.isDirty(tm9Active(4, y)), "t.isDirty(.{ .active = .{ .x = 4, .y = @intCast(y) } })")
	}
	expect(t, term.isDirty(tm9Screen(2, 2)), "t.isDirty(.{ .screen = .{ .x = 2, .y = 2 } })")
	expect(t, term.isDirty(tm9Viewport(2, 2)), "t.isDirty(.{ .viewport = .{ .x = 2, .y = 2 } })")
}

// Zig: test "term51: isDirty: point tags screen viewport active history"
func TestTm9_isDirty_point_tags_screen_viewport_active_history(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 3)
	defer term.Deinit()

	// Five lines in three rows: two rows of history.
	tm9PrintString(t, term, "1\n2\n3\n4\n5")
	term.clearDirty()

	// Dirty exactly one row through a pin and look at it via every tag.
	zigUnwrap(t, term.Screens.Active.Pages.Pin(tm9Active(0, 1))).MarkDirty()
	expect(t, !term.isDirty(tm9Active(0, 0)), "!t.isDirty(.{ .active = .{ .y = 0 } })")
	expect(t, term.isDirty(tm9Active(0, 1)), "t.isDirty(.{ .active = .{ .y = 1 } })")
	expect(t, !term.isDirty(tm9Active(0, 2)), "!t.isDirty(.{ .active = .{ .y = 2 } })")
	// Viewport is at the active area, screen has two history rows first.
	expect(t, term.isDirty(tm9Viewport(0, 1)), "t.isDirty(.{ .viewport = .{ .y = 1 } })")
	expect(t, !term.isDirty(tm9Viewport(0, 0)), "!t.isDirty(.{ .viewport = .{ .y = 0 } })")
	expect(t, term.isDirty(tm9Screen(0, 3)), "t.isDirty(.{ .screen = .{ .y = 3 } })")
	expect(t, !term.isDirty(tm9Screen(0, 2)), "!t.isDirty(.{ .screen = .{ .y = 2 } })")
	expect(t, !term.isDirty(tm9History(0, 0)), "!t.isDirty(.{ .history = .{ .y = 0 } })")
	expect(t, !term.isDirty(tm9History(0, 1)), "!t.isDirty(.{ .history = .{ .y = 1 } })")

	zigUnwrap(t, term.Screens.Active.Pages.Pin(tm9History(0, 1))).MarkDirty()
	expect(t, term.isDirty(tm9History(0, 1)), "t.isDirty(.{ .history = .{ .y = 1 } })")
	expect(t, !term.isDirty(tm9History(0, 0)), "!t.isDirty(.{ .history = .{ .y = 0 } })")
	expect(t, term.isDirty(tm9Screen(0, 1)), "t.isDirty(.{ .screen = .{ .y = 1 } })")
}

// Zig: test "term51: isDirty: asks the active screen"
func TestTm9_isDirty_asks_the_active_screen(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.Print('A'))
	expect(t, term.isDirty(tm9Active(0, 0)), "t.isDirty(.{ .active = .{} })")

	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	term.clearDirty()
	expect(t, !term.isDirty(tm9Active(0, 0)), "!t.isDirty(.{ .active = .{} })")
	zigTry(t, term.Print('B'))
	expect(t, term.isDirty(tm9Active(0, 0)), "t.isDirty(.{ .active = .{} })")

	// Back on the primary screen its row is still dirty.
	_, err = term.SwitchScreen(ScreenSetKeyPrimary)
	zigTry(t, err)
	expect(t, term.isDirty(tm9Active(0, 0)), "t.isDirty(.{ .active = .{} })")
}

// Zig: test "term51: clearDirty: clears page and row bits including history"
func TestTm9_clearDirty_clears_page_and_row_bits_including_history(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 3)
	defer term.Deinit()
	tm9PrintString(t, term, "1\n2\n3\n4\n5")

	// Everything printed is dirty, including history rows.
	expect(t, term.isDirty(tm9Active(0, 2)), "t.isDirty(.{ .active = .{ .y = 2 } })")
	zigUnwrap(t, term.Screens.Active.Pages.Pin(tm9History(0, 0))).MarkDirty()
	page := &term.Screens.Active.Pages.pages.first.data
	page.dirty = true
	expect(t, term.isDirty(tm9History(0, 0)), "t.isDirty(.{ .history = .{} })")

	term.clearDirty()
	expect(t, !page.dirty, "!page.dirty")
	expect(t, !page.IsDirty(), "!page.isDirty()")
	for y := uint32(0); y < 5; y++ {
		expect(t, !term.isDirty(tm9Screen(0, y)), "!t.isDirty(.{ .screen = .{ .y = @intCast(y) } })")
	}

	// Clearing again is harmless.
	term.clearDirty()
	expect(t, !page.IsDirty(), "!page.isDirty()")
}

// Zig: test "term51: clearDirty: leaves terminal and screen flags and other screens alone"
func TestTm9_clearDirty_leaves_terminal_and_screen_flags_and_other_screens_alone(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.Print('A'))
	term.Flags.Dirty = TerminalDirty{Palette: true, ReverseColors: true, Clear: true, Preedit: true}
	term.Screens.Active.Dirty = ScreenDirty{Selection: true, HyperlinkHover: true}
	term.clearDirty()
	expect(t, !term.isDirty(tm9Active(0, 0)), "!t.isDirty(.{ .active = .{} })")
	expect(t, term.Flags.Dirty.Palette, "t.flags.dirty.palette")
	expect(t, term.Flags.Dirty.ReverseColors, "t.flags.dirty.reverse_colors")
	expect(t, term.Flags.Dirty.Clear, "t.flags.dirty.clear")
	expect(t, term.Flags.Dirty.Preedit, "t.flags.dirty.preedit")
	expect(t, term.Screens.Active.Dirty.Selection, "t.screens.active.dirty.selection")
	expect(t, term.Screens.Active.Dirty.HyperlinkHover, "t.screens.active.dirty.hyperlink_hover")

	// Only the active screen is cleared.
	zigTry(t, term.Print('B'))
	primary := term.Screens.Active
	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	zigTry(t, term.Print('C'))
	term.clearDirty()
	expect(t, !term.isDirty(tm9Active(0, 0)), "!t.isDirty(.{ .active = .{} })")
	expect(t, primary.Pages.IsDirty(tm9Active(0, 0)), "primary.pages.isDirty(.{ .active = .{} })")
}

// Zig: test "term51: plainString: empty and simple content"
func TestTm9_plainString_empty_and_simple_content(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()
	tm9ExpectPlain(t, term, "")

	tm9PrintString(t, term, "ABC")
	tm9ExpectPlain(t, term, "ABC")
	tm9PrintString(t, term, "\nD")
	tm9ExpectPlain(t, term, "ABC\nD")
}

// Zig: test "term51: plainString: blank rows and gaps"
func TestTm9_plainString_blank_rows_and_gaps(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	// Blank row in the middle is kept, trailing blank rows are not.
	tm9PrintString(t, term, "A\n\nC")
	tm9ExpectPlain(t, term, "A\n\nC")

	// Leading blank rows and leading blank cells are kept.
	term.FullReset()
	term.SetCursorPos(3, 3)
	zigTry(t, term.Print('X'))
	tm9ExpectPlain(t, term, "\n\n  X")
}

// Zig: test "term51: plainString: only the viewport is returned"
func TestTm9_plainString_only_the_viewport_is_returned(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 3)
	defer term.Deinit()
	tm9PrintString(t, term, "1\n2\n3\n4\n5")
	tm9ExpectPlain(t, term, "3\n4\n5")

	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	tm9ExpectPlain(t, term, "1\n2\n3")
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportDelta, Delta: 1})
	tm9ExpectPlain(t, term, "2\n3\n4")
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportBottom})
	tm9ExpectPlain(t, term, "3\n4\n5")
}

// Zig: test "term51: plainString: returns the active screen"
func TestTm9_plainString_returns_the_active_screen(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 3)
	defer term.Deinit()
	tm9PrintString(t, term, "pri")

	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	tm9ExpectPlain(t, term, "")
	tm9PrintString(t, term, "alt")
	tm9ExpectPlain(t, term, "alt")

	_, err = term.SwitchScreen(ScreenSetKeyPrimary)
	zigTry(t, err)
	tm9ExpectPlain(t, term, "pri")
}

// Zig: test "term51: plainString: wrapped rows are separated, wide chars and graphemes"
func TestTm9_plainString_wrapped_rows_are_separated_wide_chars_and_graphemes(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 4)
	defer term.Deinit()

	// Soft-wrapped text appears on separate lines in plainString.
	tm9PrintString(t, term, "ABCDEFGHIJKL")
	tm9ExpectPlain(t, term, "ABCDE\nFGHIJ\nKL")

	term.FullReset()
	zigTry(t, term.Print(0x4E2D))
	zigTry(t, term.Print('x'))
	tm9ExpectPlain(t, term, "中x")
}

// The test "term51: plainString: allocation failure is reported" is not
// translated; see scripts/not-translated.txt.

// Zig: test "term51: plainStringUnwrapped: joins soft-wrapped rows"
func TestTm9_plainStringUnwrapped_joins_soft_wrapped_rows(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 4)
	defer term.Deinit()
	tm9ExpectPlainUnwrapped(t, term, "")

	tm9PrintString(t, term, "ABCDEFGHIJKL")
	tm9ExpectPlainUnwrapped(t, term, "ABCDEFGHIJKL")

	// Hard line breaks are kept.
	tm9PrintString(t, term, "\nxy")
	tm9ExpectPlainUnwrapped(t, term, "ABCDEFGHIJKL\nxy")
	tm9ExpectPlain(t, term, "ABCDE\nFGHIJ\nKL\nxy")
}

// Zig: test "term51: plainStringUnwrapped: viewport and screen selection"
func TestTm9_plainStringUnwrapped_viewport_and_screen_selection(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 3)
	defer term.Deinit()
	tm9PrintString(t, term, "1\n2\n3\n4\n5")
	tm9ExpectPlainUnwrapped(t, term, "3\n4\n5")
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	tm9ExpectPlainUnwrapped(t, term, "1\n2\n3")

	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportBottom})
	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	tm9PrintString(t, term, "alt")
	tm9ExpectPlainUnwrapped(t, term, "alt")
}

// The test "term51: plainStringUnwrapped: allocation failure is reported" is
// not translated; see scripts/not-translated.txt.

// Zig: test "term51: deccolm: modes enum values"
func TestTm9_deccolm_modes_enum_values(t *testing.T) {
	expectEqual(t, uint8(0), uint8(TerminalDeccolmMode80Cols))
	expectEqual(t, uint8(1), uint8(TerminalDeccolmMode132Cols))
}

// Zig: test "term51: deccolm: ignored without DEC mode 40"
func TestTm9_deccolm_ignored_without_DEC_mode_40(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	tm9PrintString(t, term, "AB")
	term.SetCursorPos(3, 2)
	term.Modes.Set(ModesMode132Column, true)
	term.Flags.Dirty = TerminalDirty{}

	zigTry(t, term.Deccolm(TerminalDeccolmMode80Cols))
	expect(t, !term.Modes.Get(ModesMode132Column), "!t.modes.get(.@\"132_column\")")
	expectEqual(t, SizeCellCountInt(5), term.Cols)
	expectEqual(t, SizeCellCountInt(5), term.Rows)

	// Asking for 132 columns also leaves the mode cleared.
	term.Modes.Set(ModesMode132Column, true)
	zigTry(t, term.Deccolm(TerminalDeccolmMode132Cols))
	expect(t, !term.Modes.Get(ModesMode132Column), "!t.modes.get(.@\"132_column\")")
	expectEqual(t, SizeCellCountInt(5), term.Cols)

	// Nothing was erased or moved and nothing was marked dirty.
	tm9ExpectPlain(t, term, "AB")
	expectEqual(t, SizeCellCountInt(1), term.Screens.Active.Cursor.X)
	expectEqual(t, SizeCellCountInt(2), term.Screens.Active.Cursor.Y)
	expect(t, !term.Flags.Dirty.Clear, "!t.flags.dirty.clear")
	expect(t, !term.Modes.Get(ModesModeEnableMode3), "!t.modes.get(.enable_mode_3)")
}

// Zig: test "term51: deccolm: 132 columns resizes erases and homes the cursor"
func TestTm9_deccolm_132_columns_resizes_erases_and_homes_the_cursor(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 7)
	defer term.Deinit()
	term.Modes.Set(ModesModeEnableMode3, true)

	tm9PrintString(t, term, "AB")
	term.SetCursorPos(4, 3)
	term.Screens.Active.Cursor.PendingWrap = true
	term.Flags.Dirty = TerminalDirty{}

	zigTry(t, term.Deccolm(TerminalDeccolmMode132Cols))
	expectEqual(t, SizeCellCountInt(132), term.Cols)
	expectEqual(t, SizeCellCountInt(7), term.Rows)
	expect(t, term.Modes.Get(ModesMode132Column), "t.modes.get(.@\"132_column\")")
	expect(t, term.Modes.Get(ModesModeEnableMode3), "t.modes.get(.enable_mode_3)")
	tm9ExpectPlain(t, term, "")
	expectEqual(t, SizeCellCountInt(0), term.Screens.Active.Cursor.X)
	expectEqual(t, SizeCellCountInt(0), term.Screens.Active.Cursor.Y)
	expect(t, !term.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
	expect(t, term.Flags.Dirty.Clear, "t.flags.dirty.clear")
	expectEqual(t, SizeCellCountInt(132), term.Screens.Active.Pages.cols)
	expectEqual(t, SizeCellCountInt(7), term.Screens.Active.Pages.rows)
	tm9ExpectRegion(t, term, 0, 6, 0, 131)

	// Tabstops are rebuilt for the new width: every 8 columns up to cols-2.
	expect(t, term.Tabstops.Get(8), "t.tabstops.get(8)")
	expect(t, term.Tabstops.Get(128), "t.tabstops.get(128)")
	expect(t, !term.Tabstops.Get(129), "!t.tabstops.get(129)")
	expect(t, !term.Tabstops.Get(130), "!t.tabstops.get(130)")
	expect(t, !term.Tabstops.Get(131), "!t.tabstops.get(131)")
}

// Zig: test "term51: deccolm: 80 columns from 132 and back"
func TestTm9_deccolm_80_columns_from_132_and_back(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 20, 4)
	defer term.Deinit()
	term.Modes.Set(ModesModeEnableMode3, true)

	zigTry(t, term.Deccolm(TerminalDeccolmMode132Cols))
	expectEqual(t, SizeCellCountInt(132), term.Cols)
	expect(t, term.Modes.Get(ModesMode132Column), "t.modes.get(.@\"132_column\")")

	tm9PrintString(t, term, "hello")
	zigTry(t, term.Deccolm(TerminalDeccolmMode80Cols))
	expectEqual(t, SizeCellCountInt(80), term.Cols)
	expectEqual(t, SizeCellCountInt(4), term.Rows)
	expect(t, !term.Modes.Get(ModesMode132Column), "!t.modes.get(.@\"132_column\")")
	tm9ExpectPlain(t, term, "")
	tm9ExpectRegion(t, term, 0, 3, 0, 79)
	expect(t, term.Tabstops.Get(72), "t.tabstops.get(72)")
	expect(t, !term.Tabstops.Get(78), "!t.tabstops.get(78)")
	expect(t, !term.Tabstops.Get(88), "!t.tabstops.get(88)")
}

// Zig: test "term51: deccolm: same width still erases and homes but keeps the scroll region"
func TestTm9_deccolm_same_width_still_erases_and_homes_but_keeps_the_scroll_region(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 80, 5)
	defer term.Deinit()
	term.Modes.Set(ModesModeEnableMode3, true)

	// 80 columns requested on an 80 column terminal: resize is a no-op so
	// the scrolling region (and left/right margin mode) is kept.
	term.SetTopAndBottomMargin(2, 4)
	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(10, 20)
	tm9PrintString(t, term, "AB")
	term.SetCursorPos(3, 3)
	term.Flags.Dirty = TerminalDirty{}

	zigTry(t, term.Deccolm(TerminalDeccolmMode80Cols))
	expectEqual(t, SizeCellCountInt(80), term.Cols)
	expect(t, !term.Modes.Get(ModesMode132Column), "!t.modes.get(.@\"132_column\")")
	tm9ExpectRegion(t, term, 1, 3, 9, 19)
	expect(t, term.Modes.Get(ModesModeEnableLeftAndRightMargin), "t.modes.get(.enable_left_and_right_margin)")
	tm9ExpectPlain(t, term, "")
	expectEqual(t, SizeCellCountInt(0), term.Screens.Active.Cursor.X)
	expectEqual(t, SizeCellCountInt(0), term.Screens.Active.Cursor.Y)
	expect(t, term.Flags.Dirty.Clear, "t.flags.dirty.clear")
}

// Zig: test "term51: deccolm: cursor home honours origin mode and the region"
func TestTm9_deccolm_cursor_home_honours_origin_mode_and_the_region(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 80, 10)
	defer term.Deinit()
	term.Modes.Set(ModesModeEnableMode3, true)
	term.SetTopAndBottomMargin(3, 8)
	term.Modes.Set(ModesModeOrigin, true)

	// No resize: the region survives, so home is the top of the region.
	zigTry(t, term.Deccolm(TerminalDeccolmMode80Cols))
	expectEqual(t, SizeCellCountInt(2), term.Screens.Active.Cursor.Y)
	expectEqual(t, SizeCellCountInt(0), term.Screens.Active.Cursor.X)
	tm9ExpectRegion(t, term, 2, 7, 0, 79)

	// With a resize the region is reset first, so home is row 0.
	zigTry(t, term.Deccolm(TerminalDeccolmMode132Cols))
	expectEqual(t, SizeCellCountInt(0), term.Screens.Active.Cursor.Y)
	expectEqual(t, SizeCellCountInt(0), term.Screens.Active.Cursor.X)
	tm9ExpectRegion(t, term, 0, 9, 0, 131)
	expect(t, term.Modes.Get(ModesModeOrigin), "t.modes.get(.origin)")
}

// Zig: test "term51: deccolm: protected cells survive the erase"
func TestTm9_deccolm_protected_cells_survive_the_erase(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 80, 5)
	defer term.Deinit()
	term.Modes.Set(ModesModeEnableMode3, true)

	// The erase is a non-selective ED 2 that still respects ISO protection
	// because the screen's most recent protected mode is ISO.
	term.SetProtectedMode(ANSIProtectedModeIso)
	tm9PrintString(t, term, "AB")
	term.SetProtectedMode(ANSIProtectedModeOff)
	tm9PrintString(t, term, "CD")
	tm9ExpectPlain(t, term, "ABCD")

	zigTry(t, term.Deccolm(TerminalDeccolmMode80Cols))
	tm9ExpectPlain(t, term, "AB")
	expectEqual(t, SizeCellCountInt(0), term.Screens.Active.Cursor.X)

	zigTry(t, term.Deccolm(TerminalDeccolmMode132Cols))
	tm9ExpectPlain(t, term, "AB")
	expectEqual(t, SizeCellCountInt(132), term.Cols)
}

// Zig: test "term51: deccolm: keeps the current row count"
func TestTm9_deccolm_keeps_the_current_row_count(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()
	term.Modes.Set(ModesModeEnableMode3, true)
	zigTry(t, term.Resize(40, 12))
	zigTry(t, term.Deccolm(TerminalDeccolmMode132Cols))
	expectEqual(t, SizeCellCountInt(132), term.Cols)
	expectEqual(t, SizeCellCountInt(12), term.Rows)
	zigTry(t, term.Deccolm(TerminalDeccolmMode80Cols))
	expectEqual(t, SizeCellCountInt(80), term.Cols)
	expectEqual(t, SizeCellCountInt(12), term.Rows)
}

// Zig: test "term51: resize: same size is a complete no-op"
func TestTm9_resize_same_size_is_a_complete_no_op(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 5)
	defer term.Deinit()

	tm9PrintString(t, term, "hello")
	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetTopAndBottomMargin(2, 4)
	term.SetLeftAndRightMargin(3, 8)
	term.Tabstops.Set(3)
	term.Flags.Dirty = TerminalDirty{}

	zigTry(t, term.Resize(10, 5))
	tm9ExpectRegion(t, term, 1, 3, 2, 7)
	expect(t, term.Tabstops.Get(3), "t.tabstops.get(3)")
	expect(t, term.Tabstops.Get(8), "t.tabstops.get(8)")
	expect(t, !term.Flags.Dirty.Clear, "!t.flags.dirty.clear")
	expectEqual(t, SizeCellCountInt(10), term.Cols)
	expectEqual(t, SizeCellCountInt(5), term.Rows)
	tm9ExpectPlain(t, term, "hello")
}

// Zig: test "term51: resize: width change rebuilds tabstops and resets the region"
func TestTm9_resize_width_change_rebuilds_tabstops_and_resets_the_region(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 5)
	defer term.Deinit()

	term.Tabstops.Set(3)
	term.Tabstops.Unset(8)
	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetTopAndBottomMargin(2, 4)
	term.SetLeftAndRightMargin(3, 8)
	term.Flags.Dirty = TerminalDirty{}

	zigTry(t, term.Resize(20, 5))
	expectEqual(t, SizeCellCountInt(20), term.Cols)
	expectEqual(t, SizeCellCountInt(5), term.Rows)
	expect(t, !term.Tabstops.Get(3), "!t.tabstops.get(3)")
	expect(t, term.Tabstops.Get(8), "t.tabstops.get(8)")
	expect(t, term.Tabstops.Get(16), "t.tabstops.get(16)")
	expect(t, !term.Tabstops.Get(17), "!t.tabstops.get(17)")
	expect(t, !term.Tabstops.Get(19), "!t.tabstops.get(19)")
	expectEqual(t, uint(20), term.Tabstops.cols)
	tm9ExpectRegion(t, term, 0, 4, 0, 19)
	expect(t, term.Flags.Dirty.Clear, "t.flags.dirty.clear")
	// The margin mode itself is not touched.
	expect(t, term.Modes.Get(ModesModeEnableLeftAndRightMargin), "t.modes.get(.enable_left_and_right_margin)")
	expectEqual(t, SizeCellCountInt(20), term.Screens.Active.Pages.cols)
}

// Zig: test "term51: resize: height change keeps tabstops and resets the region"
func TestTm9_resize_height_change_keeps_tabstops_and_resets_the_region(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 5)
	defer term.Deinit()

	term.Tabstops.Set(3)
	term.Tabstops.Unset(8)
	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetTopAndBottomMargin(2, 4)
	term.SetLeftAndRightMargin(3, 8)
	term.Flags.Dirty = TerminalDirty{}

	zigTry(t, term.Resize(10, 8))
	expectEqual(t, SizeCellCountInt(10), term.Cols)
	expectEqual(t, SizeCellCountInt(8), term.Rows)
	// Same width: the tabstop array is untouched.
	expect(t, term.Tabstops.Get(3), "t.tabstops.get(3)")
	expect(t, !term.Tabstops.Get(8), "!t.tabstops.get(8)")
	tm9ExpectRegion(t, term, 0, 7, 0, 9)
	expect(t, term.Flags.Dirty.Clear, "t.flags.dirty.clear")
	expectEqual(t, SizeCellCountInt(8), term.Screens.Active.Pages.rows)

	// Shrinking works the same way.
	term.SetTopAndBottomMargin(2, 4)
	zigTry(t, term.Resize(10, 3))
	expectEqual(t, SizeCellCountInt(3), term.Rows)
	tm9ExpectRegion(t, term, 0, 2, 0, 9)
}

// Zig: test "term51: resize: both dimensions and other dirty flags are preserved"
func TestTm9_resize_both_dimensions_and_other_dirty_flags_are_preserved(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 5)
	defer term.Deinit()
	term.WidthPx = 640
	term.HeightPx = 480
	term.Modes.Set(ModesMode132Column, true)
	term.Flags.Dirty = TerminalDirty{Palette: true, ReverseColors: true, Clear: false, Preedit: true}

	zigTry(t, term.Resize(7, 9))
	expectEqual(t, SizeCellCountInt(7), term.Cols)
	expectEqual(t, SizeCellCountInt(9), term.Rows)
	tm9ExpectRegion(t, term, 0, 8, 0, 6)
	expect(t, term.Flags.Dirty.Clear, "t.flags.dirty.clear")
	expect(t, term.Flags.Dirty.Palette, "t.flags.dirty.palette")
	expect(t, term.Flags.Dirty.ReverseColors, "t.flags.dirty.reverse_colors")
	expect(t, term.Flags.Dirty.Preedit, "t.flags.dirty.preedit")
	expectEqual(t, uint32(640), term.WidthPx)
	expectEqual(t, uint32(480), term.HeightPx)
	expect(t, term.Modes.Get(ModesMode132Column), "t.modes.get(.@\"132_column\")")
}

// Zig: test "term51: resize: reflows the primary screen when wraparound is on"
func TestTm9_resize_reflows_the_primary_screen_when_wraparound_is_on(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	tm9PrintString(t, term, "ABCDEFG")
	tm9ExpectPlain(t, term, "ABCDE\nFG")
	zigTry(t, term.Resize(10, 5))
	tm9ExpectPlain(t, term, "ABCDEFG")
	zigTry(t, term.Resize(4, 5))
	tm9ExpectPlain(t, term, "ABCD\nEFG")
	tm9ExpectPlainUnwrapped(t, term, "ABCDEFG")
}

// Zig: test "term51: resize: does not reflow when wraparound is off"
func TestTm9_resize_does_not_reflow_when_wraparound_is_off(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 3)
	defer term.Deinit()
	term.Modes.Set(ModesModeWraparound, false)

	tm9PrintString(t, term, "ABCDEFGH")
	zigTry(t, term.Resize(5, 3))
	tm9ExpectPlain(t, term, "ABCDE")
	// Content past the new width is gone for good.
	zigTry(t, term.Resize(10, 3))
	tm9ExpectPlain(t, term, "ABCDE")
}

// Zig: test "term51: resize: alternate screen is resized without reflow and never created"
func TestTm9_resize_alternate_screen_is_resized_without_reflow_and_never_created(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	// No alternate screen yet: resize does not create it.
	zigTry(t, term.Resize(6, 5))
	expect(t, term.Screens.Get(ScreenSetKeyAlternate) == nil, "t.screens.get(.alternate) == null")

	// Same long line on both screens.
	tm9PrintString(t, term, "ABCDEFGHI")
	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	tm9PrintString(t, term, "ABCDEFGHI")
	tm9ExpectPlain(t, term, "ABCDEF\nGHI")
	_, err = term.SwitchScreen(ScreenSetKeyPrimary)
	zigTry(t, err)
	tm9ExpectPlain(t, term, "ABCDEF\nGHI")

	zigTry(t, term.Resize(12, 5))
	expectEqual(t, SizeCellCountInt(12), term.Screens.Get(ScreenSetKeyPrimary).Pages.cols)
	expectEqual(t, SizeCellCountInt(12), term.Screens.Get(ScreenSetKeyAlternate).Pages.cols)
	// Primary reflowed, alternate did not.
	tm9ExpectPlain(t, term, "ABCDEFGHI")
	_, err = term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	tm9ExpectPlain(t, term, "ABCDEF\nGHI")

	// Rows too.
	zigTry(t, term.Resize(12, 8))
	expectEqual(t, SizeCellCountInt(8), term.Screens.Get(ScreenSetKeyPrimary).Pages.rows)
	expectEqual(t, SizeCellCountInt(8), term.Screens.Get(ScreenSetKeyAlternate).Pages.rows)
}

// Zig: test "term51: resize: prompt redraw flag clears the prompt on a width change"
func TestTm9_resize_prompt_redraw_flag_clears_the_prompt_on_a_width_change(t *testing.T) {
	testStdCapacity(t)
	type Case struct {
		redraw OSCParsersSemanticPromptRedraw
		expect string
	}
	cases := []Case{
		{OSCParsersSemanticPromptRedrawTrue, "ABCDE"},
		{OSCParsersSemanticPromptRedrawLast, "ABCDE"},
		{OSCParsersSemanticPromptRedrawFalse, "ABCDE\n> echo"},
	}
	for _, c := range cases {
		func() {
			term := tm9Init(t, 10, 3)
			defer term.Deinit()
			term.Flags.ShellRedrawsPrompt = c.redraw

			tm9PrintString(t, term, "ABCDE\n")
			term.Screens.Active.CursorSetSemanticContent(ScreenCursorSetSemanticContent{Tag: ScreenCursorSetSemanticContentTagPrompt, Prompt: OSCParsersSemanticPromptPromptKindInitial})
			tm9PrintString(t, term, "> ")
			term.Screens.Active.CursorSetSemanticContent(ScreenCursorSetSemanticContent{Tag: ScreenCursorSetSemanticContentTagInput, Input: ScreenCursorSetSemanticContentInputClearEol})
			tm9PrintString(t, term, "echo")
			tm9ExpectPlain(t, term, "ABCDE\n> echo")

			zigTry(t, term.Resize(20, 3))
			tm9ExpectPlain(t, term, c.expect)
			// The cursor does not move.
			expectEqual(t, SizeCellCountInt(6), term.Screens.Active.Cursor.X)
			expectEqual(t, SizeCellCountInt(1), term.Screens.Active.Cursor.Y)
		}()
	}
}

// Zig: test "term51: resize: prompt redraw does not apply on the alternate screen"
func TestTm9_resize_prompt_redraw_does_not_apply_on_the_alternate_screen(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 3)
	defer term.Deinit()

	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	tm9PrintString(t, term, "ABCDE\n")
	term.Screens.Active.CursorSetSemanticContent(ScreenCursorSetSemanticContent{Tag: ScreenCursorSetSemanticContentTagPrompt, Prompt: OSCParsersSemanticPromptPromptKindInitial})
	tm9PrintString(t, term, "> ")
	term.Screens.Active.CursorSetSemanticContent(ScreenCursorSetSemanticContent{Tag: ScreenCursorSetSemanticContentTagInput, Input: ScreenCursorSetSemanticContentInputClearEol})
	tm9PrintString(t, term, "echo")

	// The alternate screen is resized with the default prompt_redraw
	// (.false), regardless of the terminal flag.
	expect(t, term.Flags.ShellRedrawsPrompt == OSCParsersSemanticPromptRedrawTrue, "t.flags.shell_redraws_prompt == .true")
	zigTry(t, term.Resize(20, 3))
	tm9ExpectPlain(t, term, "ABCDE\n> echo")
}

// Zig: test "term51: resize: cursor is kept inside a smaller screen"
func TestTm9_resize_cursor_is_kept_inside_a_smaller_screen(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 8, 8)
	defer term.Deinit()

	term.SetCursorPos(8, 8)
	zigTry(t, term.Resize(3, 4))
	expect(t, term.Screens.Active.Cursor.X < 3, "t.screens.active.cursor.x < 3")
	expect(t, term.Screens.Active.Cursor.Y < 4, "t.screens.active.cursor.y < 4")
	expectEqual(t, SizeCellCountInt(3), term.Screens.Active.Pages.cols)
	expectEqual(t, SizeCellCountInt(4), term.Screens.Active.Pages.rows)
}

// Zig: test "term51: resize: shrinking rows pushes lines to history and growing pulls them back"
func TestTm9_resize_shrinking_rows_pushes_lines_to_history_and_growing_pulls_them_back(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	tm9PrintString(t, term, "1\n2\n3\n4\n5")
	zigTry(t, term.Resize(5, 3))
	tm9ExpectPlain(t, term, "3\n4\n5")
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	tm9ExpectPlain(t, term, "1\n2\n3")
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportBottom})

	zigTry(t, term.Resize(5, 5))
	tm9ExpectPlain(t, term, "1\n2\n3\n4\n5")
}

// Zig: test "term51: resize: minimal size one by one"
func TestTm9_resize_minimal_size_one_by_one(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.Resize(1, 1))
	expectEqual(t, SizeCellCountInt(1), term.Cols)
	expectEqual(t, SizeCellCountInt(1), term.Rows)
	tm9ExpectRegion(t, term, 0, 0, 0, 0)
	expectEqual(t, uint(1), term.Tabstops.cols)
	expect(t, !term.Tabstops.Get(0), "!t.tabstops.get(0)")
	expectEqual(t, SizeCellCountInt(0), term.Screens.Active.Cursor.X)
	expectEqual(t, SizeCellCountInt(0), term.Screens.Active.Cursor.Y)

	// Still usable.
	zigTry(t, term.Print('X'))
	tm9ExpectPlain(t, term, "X")
}

// Zig: test "term51: resize: wide terminal uses dynamic tabstop storage"
func TestTm9_resize_wide_terminal_uses_dynamic_tabstop_storage(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 2)
	defer term.Deinit()

	// 512 columns are preallocated; 600 needs the dynamic segment.
	zigTry(t, term.Resize(600, 2))
	expectEqual(t, SizeCellCountInt(600), term.Cols)
	expectEqual(t, uint(600), term.Tabstops.cols)
	expect(t, term.Tabstops.Capacity() >= 600, "t.tabstops.capacity() >= 600")
	expect(t, term.Tabstops.Get(8), "t.tabstops.get(8)")
	expect(t, term.Tabstops.Get(512), "t.tabstops.get(512)")
	expect(t, term.Tabstops.Get(592), "t.tabstops.get(592)")
	expect(t, !term.Tabstops.Get(593), "!t.tabstops.get(593)")
	expect(t, !term.Tabstops.Get(598), "!t.tabstops.get(598)")
	tm9ExpectRegion(t, term, 0, 1, 0, 599)

	// Going back down rebuilds the tabstops from scratch.
	term.Tabstops.Set(3)
	zigTry(t, term.Resize(80, 2))
	expectEqual(t, uint(80), term.Tabstops.cols)
	expect(t, !term.Tabstops.Get(3), "!t.tabstops.get(3)")
	expect(t, term.Tabstops.Get(72), "t.tabstops.get(72)")
	expect(t, !term.Tabstops.Get(78), "!t.tabstops.get(78)")
}

// The test "term51: resize: tabstop allocation failure is reported before
// any state changes" is not translated; see scripts/not-translated.txt.

// Zig: test "term51: switchScreen: already on the requested screen does nothing"
func TestTm9_switchScreen_already_on_the_requested_screen_does_nothing(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	tm9PrintString(t, term, "AB")
	tm9StartHyperlink(t, term.Screens.Active, "http://example.com")
	tm9Select(t, term.Screens.Active, 0, 0, 1, 0)
	term.Flags.Dirty = TerminalDirty{}

	old, err := term.SwitchScreen(ScreenSetKeyPrimary)
	zigTry(t, err)
	expect(t, old == nil, "try t.switchScreen(.primary) == null")
	expectEqual(t, ScreenSetKeyPrimary, term.Screens.ActiveKey)
	// The hyperlink is still active, the selection kept, nothing dirtied
	// and no alternate screen was created.
	expect(t, term.Screens.Active.Cursor.HyperlinkID != 0, "t.screens.active.cursor.hyperlink_id != 0")
	expect(t, term.Screens.Active.Cursor.Hyperlink != nil, "t.screens.active.cursor.hyperlink != null")
	expect(t, term.Screens.Active.Selection.Valid, "t.screens.active.selection != null")
	expect(t, !term.Flags.Dirty.Clear, "!t.flags.dirty.clear")
	expect(t, term.Screens.Get(ScreenSetKeyAlternate) == nil, "t.screens.get(.alternate) == null")

	// Same on the alternate screen.
	term.Screens.Active.EndHyperlink()
	term.Screens.Active.ClearSelection()
	_, err = term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	tm9StartHyperlink(t, term.Screens.Active, "http://example.com")
	tm9Select(t, term.Screens.Active, 0, 0, 1, 0)
	term.Flags.Dirty = TerminalDirty{}
	old, err = term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	expect(t, old == nil, "try t.switchScreen(.alternate) == null")
	expect(t, term.Screens.Active.Cursor.HyperlinkID != 0, "t.screens.active.cursor.hyperlink_id != 0")
	expect(t, term.Screens.Active.Selection.Valid, "t.screens.active.selection != null")
	expect(t, !term.Flags.Dirty.Clear, "!t.flags.dirty.clear")
}

// Zig: test "term51: switchScreen: to alternate creates it lazily and returns the old screen"
func TestTm9_switchScreen_to_alternate_creates_it_lazily_and_returns_the_old_screen(t *testing.T) {
	testStdCapacity(t)
	opts := terminalOptionsDefault(7, 4)
	opts.MaxScrollback = 1000
	term := tm9InitOpts(t, opts)
	defer term.Deinit()

	tm9PrintString(t, term, "AB")
	term.SetCursorPos(3, 4)
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagBold))
	primary := term.Screens.Active
	expect(t, term.Screens.Get(ScreenSetKeyAlternate) == nil, "t.screens.get(.alternate) == null")
	term.Flags.Dirty = TerminalDirty{}

	old, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	expect(t, old != nil, "(try t.switchScreen(.alternate)).? != null")
	expect(t, old == primary, "old == primary")
	expectEqual(t, ScreenSetKeyAlternate, term.Screens.ActiveKey)
	expect(t, term.Screens.Active == term.Screens.Get(ScreenSetKeyAlternate), "t.screens.active == t.screens.get(.alternate).?")
	expect(t, term.Screens.Active != primary, "t.screens.active != primary")
	expect(t, term.Screens.Get(ScreenSetKeyPrimary) == primary, "t.screens.get(.primary).? == primary")

	// The new screen has the terminal's size, no scrollback and a fresh
	// cursor: switchScreen itself does not copy the cursor.
	alt := term.Screens.Active
	expectEqual(t, SizeCellCountInt(7), alt.Pages.cols)
	expectEqual(t, SizeCellCountInt(4), alt.Pages.rows)
	expect(t, alt.NoScrollback, "alt.no_scrollback")
	expect(t, !primary.NoScrollback, "!primary.no_scrollback")
	expectEqual(t, uint64(0), alt.Pages.explicit_max_size)
	expectEqual(t, SizeCellCountInt(0), alt.Cursor.X)
	expectEqual(t, SizeCellCountInt(0), alt.Cursor.Y)
	expect(t, alt.Cursor.Style.Default(), "alt.cursor.style.default()")
	expect(t, !alt.SavedCursor.Valid, "alt.saved_cursor == null")
	tm9ScreenPlain(t, alt, "")

	// The primary screen is untouched.
	expectEqual(t, SizeCellCountInt(3), primary.Cursor.X)
	expectEqual(t, SizeCellCountInt(2), primary.Cursor.Y)
	expect(t, primary.Cursor.Style.Flags.Bold, "primary.cursor.style.flags.bold")
	tm9ScreenPlain(t, primary, "AB")
	expect(t, term.Flags.Dirty.Clear, "t.flags.dirty.clear")
}

// Zig: test "term51: switchScreen: back to primary and again to alternate reuse the screens"
func TestTm9_switchScreen_back_to_primary_and_again_to_alternate_reuse_the_screens(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 6, 4)
	defer term.Deinit()

	tm9PrintString(t, term, "pri")
	term.SaveCursor()
	primary := term.Screens.Active
	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	alt_screen := term.Screens.Active
	tm9PrintString(t, term, "alt")
	term.SetCursorPos(2, 2)

	term.Flags.Dirty = TerminalDirty{}
	back, err := term.SwitchScreen(ScreenSetKeyPrimary)
	zigTry(t, err)
	expect(t, back == alt_screen, "back == alt_screen")
	expect(t, term.Screens.Active == primary, "t.screens.active == primary")
	expectEqual(t, ScreenSetKeyPrimary, term.Screens.ActiveKey)
	expect(t, term.Flags.Dirty.Clear, "t.flags.dirty.clear")
	// Primary state is exactly as it was left, including its saved cursor
	// and cursor position (no copy from the alternate screen).
	tm9ScreenPlain(t, primary, "pri")
	expectEqual(t, SizeCellCountInt(3), primary.Cursor.X)
	expectEqual(t, SizeCellCountInt(0), primary.Cursor.Y)
	expect(t, primary.SavedCursor.Valid, "primary.saved_cursor != null")
	expectEqual(t, SizeCellCountInt(3), zigUnwrap(t, primary.SavedCursor).X)

	// The alternate screen is the same object with its content and cursor.
	again, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	expect(t, again == primary, "again == primary")
	expect(t, term.Screens.Active == alt_screen, "t.screens.active == alt_screen")
	tm9ScreenPlain(t, alt_screen, "alt")
	expectEqual(t, SizeCellCountInt(1), alt_screen.Cursor.X)
	expectEqual(t, SizeCellCountInt(1), alt_screen.Cursor.Y)
}

// Zig: test "term51: switchScreen: new alternate screen uses the current terminal size"
func TestTm9_switchScreen_new_alternate_screen_uses_the_current_terminal_size(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.Resize(9, 3))
	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	expectEqual(t, SizeCellCountInt(9), term.Screens.Active.Pages.cols)
	expectEqual(t, SizeCellCountInt(3), term.Screens.Active.Pages.rows)
}

// Zig: test "term51: switchScreen: alternate screen has no scrollback"
func TestTm9_switchScreen_alternate_screen_has_no_scrollback(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 3)
	defer term.Deinit()

	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	tm9PrintString(t, term, "1\n2\n3\n4\n5\n6")
	tm9ExpectPlain(t, term, "4\n5\n6")
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	tm9ExpectPlain(t, term, "4\n5\n6")
	expectEqual(t, uint(3), term.Screens.Active.Pages.total_rows)

	// The primary screen does keep scrollback.
	_, err = term.SwitchScreen(ScreenSetKeyPrimary)
	zigTry(t, err)
	tm9PrintString(t, term, "1\n2\n3\n4\n5\n6")
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	tm9ExpectPlain(t, term, "1\n2\n3")
}

// Zig: test "term51: switchScreen: ends the hyperlink on the screen being left"
func TestTm9_switchScreen_ends_the_hyperlink_on_the_screen_being_left(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	tm9StartHyperlink(t, term.Screens.Active, "http://one.example")
	primary := term.Screens.Active
	expect(t, primary.Cursor.HyperlinkID != 0, "primary.cursor.hyperlink_id != 0")

	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	expectEqual(t, HyperlinkID(0), primary.Cursor.HyperlinkID)
	expect(t, primary.Cursor.Hyperlink == nil, "primary.cursor.hyperlink == null")
	expectEqual(t, HyperlinkID(0), term.Screens.Active.Cursor.HyperlinkID)

	alt := term.Screens.Active
	zigTry(t, alt.StartHyperlink([]uint8("http://two.example"), Some([]uint8("id"))))
	expect(t, alt.Cursor.HyperlinkID != 0, "alt.cursor.hyperlink_id != 0")
	_, err = term.SwitchScreen(ScreenSetKeyPrimary)
	zigTry(t, err)
	expectEqual(t, HyperlinkID(0), alt.Cursor.HyperlinkID)
	expect(t, alt.Cursor.Hyperlink == nil, "alt.cursor.hyperlink == null")
	expectEqual(t, HyperlinkID(0), term.Screens.Active.Cursor.HyperlinkID)
}

// Zig: test "term51: switchScreen: charset state is carried to the new screen"
func TestTm9_switchScreen_charset_state_is_carried_to_the_new_screen(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	term.ConfigureCharset(CharsetsSlotsG1, CharsetsCharsetDecSpecial)
	term.ConfigureCharset(CharsetsSlotsG3, CharsetsCharsetBritish)
	term.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG1, false)
	term.InvokeCharset(CharsetsActiveSlotGR, CharsetsSlotsG3, false)
	term.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG2, true)

	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	{
		cs := term.Screens.Active.Charset
		expectEqual(t, CharsetsCharsetUtf8, cs.charsets.Get(CharsetsSlotsG0))
		expectEqual(t, CharsetsCharsetDecSpecial, cs.charsets.Get(CharsetsSlotsG1))
		expectEqual(t, CharsetsCharsetUtf8, cs.charsets.Get(CharsetsSlotsG2))
		expectEqual(t, CharsetsCharsetBritish, cs.charsets.Get(CharsetsSlotsG3))
		expectEqual(t, CharsetsSlotsG1, cs.Gl)
		expectEqual(t, CharsetsSlotsG3, cs.Gr)
		expectEqual(t, Some(CharsetsSlotsG2), cs.SingleShift)
	}

	// Changes made on the alternate screen come back, overwriting the
	// primary screen's own state.
	term.ConfigureCharset(CharsetsSlotsG0, CharsetsCharsetAscii)
	term.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG0, false)
	term.Screens.Active.Charset.SingleShift = None[CharsetsSlots]()
	_, err = term.SwitchScreen(ScreenSetKeyPrimary)
	zigTry(t, err)
	{
		cs := term.Screens.Active.Charset
		expectEqual(t, CharsetsCharsetAscii, cs.charsets.Get(CharsetsSlotsG0))
		expectEqual(t, CharsetsSlotsG0, cs.Gl)
		expect(t, !cs.SingleShift.Valid, "cs.single_shift == null")
	}
}

// Zig: test "term51: switchScreen: clears the selection of the destination screen only"
func TestTm9_switchScreen_clears_the_selection_of_the_destination_screen_only(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	tm9PrintString(t, term, "ABC")
	primary := term.Screens.Active
	tm9Select(t, primary, 0, 0, 2, 0)
	primary.Dirty = ScreenDirty{}

	// Entering a screen without a selection: the old selection stays and
	// the new screen is not marked dirty by the clear.
	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	alt := term.Screens.Active
	expect(t, primary.Selection.Valid, "primary.selection != null")
	expect(t, !alt.Selection.Valid, "alt.selection == null")
	expect(t, !alt.Dirty.Selection, "!alt.dirty.selection")
	expect(t, !primary.Dirty.Selection, "!primary.dirty.selection")

	// A selection on the alternate screen survives leaving it ...
	tm9Select(t, alt, 0, 0, 1, 0)
	alt.Dirty = ScreenDirty{}
	_, err = term.SwitchScreen(ScreenSetKeyPrimary)
	zigTry(t, err)
	expect(t, alt.Selection.Valid, "alt.selection != null")
	// ... but the primary screen's selection is cleared on re-entry.
	expect(t, !primary.Selection.Valid, "primary.selection == null")
	expect(t, primary.Dirty.Selection, "primary.dirty.selection")
	expect(t, !alt.Dirty.Selection, "!alt.dirty.selection")

	// Re-entering the alternate screen clears its selection and marks it.
	_, err = term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	expect(t, !alt.Selection.Valid, "alt.selection == null")
	expect(t, alt.Dirty.Selection, "alt.dirty.selection")
}

// Zig: test "term51: switchScreen: viewport of the screen is not changed"
func TestTm9_switchScreen_viewport_of_the_screen_is_not_changed(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 3)
	defer term.Deinit()

	tm9PrintString(t, term, "1\n2\n3\n4\n5")
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	primary := term.Screens.Active
	expectEqual(t, PageListViewportTop, primary.Pages.viewport.Tag)

	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	expectEqual(t, PageListViewportActive, term.Screens.Active.Pages.viewport.Tag)
	expectEqual(t, PageListViewportTop, primary.Pages.viewport.Tag)
	_, err = term.SwitchScreen(ScreenSetKeyPrimary)
	zigTry(t, err)
	expectEqual(t, PageListViewportTop, primary.Pages.viewport.Tag)
	tm9ExpectPlain(t, term, "1\n2\n3")
}

// Zig: test "term51: switchScreen: marks the terminal dirty only when switching"
func TestTm9_switchScreen_marks_the_terminal_dirty_only_when_switching(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	term.Flags.Dirty = TerminalDirty{}
	_, err := term.SwitchScreen(ScreenSetKeyPrimary)
	zigTry(t, err)
	expect(t, !term.Flags.Dirty.Clear, "!t.flags.dirty.clear")
	_, err = term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	expect(t, term.Flags.Dirty.Clear, "t.flags.dirty.clear")
	// Only the clear flag is touched.
	expect(t, !term.Flags.Dirty.Palette, "!t.flags.dirty.palette")
	expect(t, !term.Flags.Dirty.ReverseColors, "!t.flags.dirty.reverse_colors")
	expect(t, !term.Flags.Dirty.Preedit, "!t.flags.dirty.preedit")
}

// The test "term51: switchScreen: allocation failure leaves the primary
// screen active" is not translated; see scripts/not-translated.txt.

// Zig: test "term51: switchScreenMode: 47 enable copies the cursor without saving or erasing"
func TestTm9_switchScreenMode_47_enable_copies_the_cursor_without_saving_or_erasing(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 5)
	defer term.Deinit()
	tm9PrimarySetup(t, term)
	primary := term.Screens.Active
	term.Flags.Dirty = TerminalDirty{}

	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode47, true))
	alt := term.Screens.Active
	expectEqual(t, ScreenSetKeyAlternate, term.Screens.ActiveKey)
	expect(t, alt != primary, "alt != primary")
	expectEqual(t, SizeCellCountInt(3), alt.Cursor.X)
	expectEqual(t, SizeCellCountInt(2), alt.Cursor.Y)
	expect(t, alt.Cursor.PendingWrap, "alt.cursor.pending_wrap")
	expect(t, alt.Cursor.Protected, "alt.cursor.protected")
	expectEqual(t, ScreenCursorStyleBar, alt.Cursor.CursorStyle)
	expect(t, alt.Cursor.Style.Eql(Style{Flags: StyleFlags{Bold: true}}), "alt.cursor.style.eql(.{ .flags = .{ .bold = true } })")
	expect(t, alt.Cursor.StyleID != StyleDefaultID, "alt.cursor.style_id != style.default_id")
	expectEqual(t, uint(1), alt.Cursor.PagePin.Node.data.styles.Count())
	// Only the cursor is copied, not the screen-level protected mode.
	expectEqual(t, ANSIProtectedModeOff, alt.ProtectedMode)

	// No cursor was saved on either screen, nothing was erased.
	expect(t, !primary.SavedCursor.Valid, "primary.saved_cursor == null")
	expect(t, !alt.SavedCursor.Valid, "alt.saved_cursor == null")
	tm9ScreenPlain(t, primary, "AB")
	tm9ScreenPlain(t, alt, "")
	expect(t, term.Flags.Dirty.Clear, "t.flags.dirty.clear")
	// The mode flags themselves are the stream handler's business.
	expect(t, !term.Modes.Get(ModesModeAltScreenLegacy), "!t.modes.get(.alt_screen_legacy)")
}

// Zig: test "term51: switchScreenMode: 47 disable copies the alternate cursor to primary"
func TestTm9_switchScreenMode_47_disable_copies_the_alternate_cursor_to_primary(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 5)
	defer term.Deinit()
	tm9PrintString(t, term, "AB")
	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode47, true))

	// Move and restyle the cursor on the alternate screen.
	tm9PrintString(t, term, "xyz")
	term.SetCursorPos(5, 9)
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagItalic))
	term.Screens.Active.Cursor.PendingWrap = true
	alt := term.Screens.Active
	primary := term.Screens.Get(ScreenSetKeyPrimary)
	term.Flags.Dirty = TerminalDirty{}

	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode47, false))
	expectEqual(t, ScreenSetKeyPrimary, term.Screens.ActiveKey)
	expect(t, term.Screens.Active == primary, "t.screens.active == primary")
	expectEqual(t, SizeCellCountInt(8), primary.Cursor.X)
	expectEqual(t, SizeCellCountInt(4), primary.Cursor.Y)
	expect(t, primary.Cursor.PendingWrap, "primary.cursor.pending_wrap")
	expect(t, primary.Cursor.Style.Eql(Style{Flags: StyleFlags{Italic: true}}), "primary.cursor.style.eql(.{ .flags = .{ .italic = true } })")
	expect(t, term.Flags.Dirty.Clear, "t.flags.dirty.clear")

	// Neither screen was erased.
	tm9ScreenPlain(t, primary, "AB")
	tm9ScreenPlain(t, alt, "  xyz")
	expect(t, !primary.SavedCursor.Valid, "primary.saved_cursor == null")
}

// Zig: test "term51: switchScreenMode: 47 when already in the target screen is a no-op"
func TestTm9_switchScreenMode_47_when_already_in_the_target_screen_is_a_no_op(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 5)
	defer term.Deinit()
	tm9PrintString(t, term, "AB")

	// Disable on primary: no switch, no alternate screen is created.
	term.Flags.Dirty = TerminalDirty{}
	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode47, false))
	expectEqual(t, ScreenSetKeyPrimary, term.Screens.ActiveKey)
	expect(t, term.Screens.Get(ScreenSetKeyAlternate) == nil, "t.screens.get(.alternate) == null")
	expectEqual(t, SizeCellCountInt(2), term.Screens.Active.Cursor.X)
	expect(t, !term.Flags.Dirty.Clear, "!t.flags.dirty.clear")

	// Enable on alternate: the cursor is NOT copied again.
	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode47, true))
	tm9PrintString(t, term, "xyz")
	term.Screens.Active.Cursor.PendingWrap = false
	term.Flags.Dirty = TerminalDirty{}
	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode47, true))
	expectEqual(t, ScreenSetKeyAlternate, term.Screens.ActiveKey)
	expectEqual(t, SizeCellCountInt(5), term.Screens.Active.Cursor.X)
	tm9ExpectPlain(t, term, "  xyz")
	expect(t, !term.Flags.Dirty.Clear, "!t.flags.dirty.clear")
	expectEqual(t, SizeCellCountInt(2), term.Screens.Get(ScreenSetKeyPrimary).Cursor.X)
}

// Zig: test "term51: switchScreenMode: 1047 enable does not erase the alternate screen"
func TestTm9_switchScreenMode_1047_enable_does_not_erase_the_alternate_screen(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 5)
	defer term.Deinit()
	tm9PrimarySetup(t, term)
	primary := term.Screens.Active

	// Leave stale content on the alternate screen (switchScreen never erases).
	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	tm9PrintString(t, term, "stale")
	_, err = term.SwitchScreen(ScreenSetKeyPrimary)
	zigTry(t, err)
	term.Flags.Dirty = TerminalDirty{}

	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1047, true))
	alt := term.Screens.Active
	expectEqual(t, ScreenSetKeyAlternate, term.Screens.ActiveKey)
	tm9ScreenPlain(t, alt, "stale")
	// The cursor is copied from the primary screen, after the switch.
	expectEqual(t, SizeCellCountInt(3), alt.Cursor.X)
	expectEqual(t, SizeCellCountInt(2), alt.Cursor.Y)
	expect(t, alt.Cursor.PendingWrap, "alt.cursor.pending_wrap")
	expect(t, alt.Cursor.Style.Flags.Bold, "alt.cursor.style.flags.bold")
	expect(t, !primary.SavedCursor.Valid, "primary.saved_cursor == null")
	expect(t, !alt.SavedCursor.Valid, "alt.saved_cursor == null")
	expect(t, term.Flags.Dirty.Clear, "t.flags.dirty.clear")
}

// Zig: test "term51: switchScreenMode: 1047 disable erases the alternate screen then copies the cursor"
func TestTm9_switchScreenMode_1047_disable_erases_the_alternate_screen_then_copies_the_cursor(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 5)
	defer term.Deinit()
	tm9PrintString(t, term, "AB")
	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1047, true))
	tm9PrintString(t, term, "xyz")
	term.SetCursorPos(4, 7)
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagItalic))
	term.Screens.Active.Cursor.PendingWrap = true
	alt := term.Screens.Active
	primary := term.Screens.Get(ScreenSetKeyPrimary)
	term.Flags.Dirty = TerminalDirty{}

	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1047, false))
	expectEqual(t, ScreenSetKeyPrimary, term.Screens.ActiveKey)
	expect(t, term.Flags.Dirty.Clear, "t.flags.dirty.clear")
	tm9ScreenPlain(t, alt, "")
	tm9ScreenPlain(t, primary, "AB")
	expectEqual(t, SizeCellCountInt(6), primary.Cursor.X)
	expectEqual(t, SizeCellCountInt(3), primary.Cursor.Y)
	expect(t, primary.Cursor.Style.Eql(Style{Flags: StyleFlags{Italic: true}}), "primary.cursor.style.eql(.{ .flags = .{ .italic = true } })")
	// The erase happens before the copy and clears pending wrap, so the
	// pending wrap state of the alternate cursor is not carried over.
	expect(t, !alt.Cursor.PendingWrap, "!alt.cursor.pending_wrap")
	expect(t, !primary.Cursor.PendingWrap, "!primary.cursor.pending_wrap")
	// Re-entering shows an empty screen.
	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1047, true))
	tm9ExpectPlain(t, term, "")
}

// Zig: test "term51: switchScreenMode: 1047 disable on primary neither erases nor copies"
func TestTm9_switchScreenMode_1047_disable_on_primary_neither_erases_nor_copies(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 5)
	defer term.Deinit()
	tm9PrintString(t, term, "AB")

	// An alternate screen with content exists but is not active.
	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	tm9PrintString(t, term, "alt")
	_, err = term.SwitchScreen(ScreenSetKeyPrimary)
	zigTry(t, err)
	term.SetCursorPos(2, 2)
	term.Flags.Dirty = TerminalDirty{}

	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1047, false))
	expectEqual(t, ScreenSetKeyPrimary, term.Screens.ActiveKey)
	tm9ExpectPlain(t, term, "AB")
	tm9ScreenPlain(t, term.Screens.Get(ScreenSetKeyAlternate), "alt")
	expectEqual(t, SizeCellCountInt(1), term.Screens.Active.Cursor.X)
	expectEqual(t, SizeCellCountInt(1), term.Screens.Active.Cursor.Y)
	expect(t, !term.Flags.Dirty.Clear, "!t.flags.dirty.clear")
}

// Zig: test "term51: switchScreenMode: 1047 enable when already on the alternate screen is a no-op"
func TestTm9_switchScreenMode_1047_enable_when_already_on_the_alternate_screen_is_a_no_op(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 5)
	defer term.Deinit()
	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1047, true))
	tm9PrintString(t, term, "xyz")
	term.Flags.Dirty = TerminalDirty{}

	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1047, true))
	expectEqual(t, ScreenSetKeyAlternate, term.Screens.ActiveKey)
	tm9ExpectPlain(t, term, "xyz")
	expectEqual(t, SizeCellCountInt(3), term.Screens.Active.Cursor.X)
	expect(t, !term.Flags.Dirty.Clear, "!t.flags.dirty.clear")
}

// Zig: test "term51: switchScreenMode: 1047 disable on alternate honours ISO protection"
func TestTm9_switchScreenMode_1047_disable_on_alternate_honours_ISO_protection(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 5)
	defer term.Deinit()
	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1047, true))

	term.SetProtectedMode(ANSIProtectedModeIso)
	tm9PrintString(t, term, "P")
	term.SetProtectedMode(ANSIProtectedModeOff)
	tm9PrintString(t, term, "u")
	tm9ExpectPlain(t, term, "Pu")

	// The erase on exit is a plain ED 2 which skips protected cells.
	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1047, false))
	tm9ScreenPlain(t, term.Screens.Get(ScreenSetKeyAlternate), "P")
}

// Zig: test "term51: switchScreenMode: 1049 enable saves the cursor, erases and copies"
func TestTm9_switchScreenMode_1049_enable_saves_the_cursor_erases_and_copies(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 5)
	defer term.Deinit()
	tm9PrimarySetup(t, term)
	term.Modes.Set(ModesModeOrigin, true)
	term.ConfigureCharset(CharsetsSlotsG1, CharsetsCharsetDecSpecial)
	term.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG1, false)
	primary := term.Screens.Active

	// Stale content on the alternate screen is erased on entry.
	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	tm9PrintString(t, term, "stale")
	_, err = term.SwitchScreen(ScreenSetKeyPrimary)
	zigTry(t, err)
	term.Flags.Dirty = TerminalDirty{}

	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1049, true))
	alt := term.Screens.Active
	expectEqual(t, ScreenSetKeyAlternate, term.Screens.ActiveKey)
	tm9ScreenPlain(t, alt, "")
	tm9ScreenPlain(t, primary, "AB")
	expect(t, term.Flags.Dirty.Clear, "t.flags.dirty.clear")

	// The primary screen got a saved cursor with everything in it.
	sc := zigUnwrap(t, primary.SavedCursor)
	expectEqual(t, SizeCellCountInt(3), sc.X)
	expectEqual(t, SizeCellCountInt(2), sc.Y)
	expect(t, sc.Style.Eql(Style{Flags: StyleFlags{Bold: true}}), "sc.style.eql(.{ .flags = .{ .bold = true } })")
	expect(t, sc.Protected, "sc.protected")
	expect(t, sc.PendingWrap, "sc.pending_wrap")
	expect(t, sc.Origin, "sc.origin")
	expectEqual(t, CharsetsSlotsG1, sc.Charset.Gl)
	expectEqual(t, CharsetsCharsetDecSpecial, sc.Charset.charsets.Get(CharsetsSlotsG1))
	expect(t, !alt.SavedCursor.Valid, "alt.saved_cursor == null")

	// The cursor was copied from the primary screen, after the erase, so
	// pending wrap (which the erase clears) is the primary's.
	expectEqual(t, SizeCellCountInt(3), alt.Cursor.X)
	expectEqual(t, SizeCellCountInt(2), alt.Cursor.Y)
	expect(t, alt.Cursor.PendingWrap, "alt.cursor.pending_wrap")
	expect(t, alt.Cursor.Protected, "alt.cursor.protected")
	expect(t, alt.Cursor.Style.Flags.Bold, "alt.cursor.style.flags.bold")
	expectEqual(t, ScreenCursorStyleBar, alt.Cursor.CursorStyle)
}

// Zig: test "term51: switchScreenMode: 1049 enable on the alternate screen saves there and erases"
func TestTm9_switchScreenMode_1049_enable_on_the_alternate_screen_saves_there_and_erases(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 5)
	defer term.Deinit()
	tm9PrintString(t, term, "AB")
	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1049, true))
	alt := term.Screens.Active
	primary := term.Screens.Get(ScreenSetKeyPrimary)
	primary_saved := zigUnwrap(t, primary.SavedCursor)

	tm9PrintString(t, term, "xyz")
	term.SetCursorPos(2, 3)
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagItalic))
	term.Flags.Dirty = TerminalDirty{}

	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1049, true))
	expectEqual(t, ScreenSetKeyAlternate, term.Screens.ActiveKey)
	expect(t, term.Screens.Active == alt, "t.screens.active == alt")
	// Saved on the alternate screen at its current cursor.
	sc := zigUnwrap(t, alt.SavedCursor)
	expectEqual(t, SizeCellCountInt(2), sc.X)
	expectEqual(t, SizeCellCountInt(1), sc.Y)
	expect(t, sc.Style.Flags.Italic, "sc.style.flags.italic")
	// The primary screen's saved cursor is untouched.
	expectEqual(t, primary_saved.X, zigUnwrap(t, primary.SavedCursor).X)
	expect(t, !zigUnwrap(t, primary.SavedCursor).Style.Flags.Italic, "!primary.saved_cursor.?.style.flags.italic")
	// The alternate screen is erased but the cursor stays put (no copy).
	tm9ExpectPlain(t, term, "")
	expectEqual(t, SizeCellCountInt(2), alt.Cursor.X)
	expectEqual(t, SizeCellCountInt(1), alt.Cursor.Y)
	expect(t, alt.Cursor.Style.Flags.Italic, "alt.cursor.style.flags.italic")
	expect(t, term.Flags.Dirty.Clear, "t.flags.dirty.clear")
}

// Zig: test "term51: switchScreenMode: 1049 disable restores the saved cursor on primary"
func TestTm9_switchScreenMode_1049_disable_restores_the_saved_cursor_on_primary(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 5)
	defer term.Deinit()
	tm9PrimarySetup(t, term)
	term.ConfigureCharset(CharsetsSlotsG1, CharsetsCharsetDecSpecial)
	term.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG1, false)
	primary := term.Screens.Active
	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1049, true))
	alt := term.Screens.Active

	// Change everything the restore is supposed to bring back. The charset
	// came along with the switch, so the text is drawn with line graphics.
	term.Screens.Active.Cursor.PendingWrap = false
	tm9PrintString(t, term, "xyz")
	term.SetCursorPos(5, 9)
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagUnset))
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagItalic))
	term.SetProtectedMode(ANSIProtectedModeOff)
	term.Modes.Set(ModesModeOrigin, true)
	term.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG0, false)
	term.ConfigureCharset(CharsetsSlotsG0, CharsetsCharsetBritish)
	term.Flags.Dirty = TerminalDirty{}

	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1049, false))
	expectEqual(t, ScreenSetKeyPrimary, term.Screens.ActiveKey)
	expect(t, term.Screens.Active == primary, "t.screens.active == primary")
	expect(t, term.Flags.Dirty.Clear, "t.flags.dirty.clear")
	expectEqual(t, SizeCellCountInt(3), primary.Cursor.X)
	expectEqual(t, SizeCellCountInt(2), primary.Cursor.Y)
	expect(t, primary.Cursor.Style.Eql(Style{Flags: StyleFlags{Bold: true}}), "primary.cursor.style.eql(.{ .flags = .{ .bold = true } })")
	expect(t, primary.Cursor.Protected, "primary.cursor.protected")
	expect(t, primary.Cursor.PendingWrap, "primary.cursor.pending_wrap")
	expect(t, !term.Modes.Get(ModesModeOrigin), "!t.modes.get(.origin)")
	expectEqual(t, CharsetsSlotsG1, primary.Charset.Gl)
	expectEqual(t, CharsetsCharsetUtf8, primary.Charset.charsets.Get(CharsetsSlotsG0))
	expectEqual(t, CharsetsCharsetDecSpecial, primary.Charset.charsets.Get(CharsetsSlotsG1))
	// The saved cursor stays saved, the alternate screen is not erased and
	// the primary content is intact.
	expect(t, primary.SavedCursor.Valid, "primary.saved_cursor != null")
	tm9ScreenPlain(t, alt, "\n\n   │≤≥")
	tm9ScreenPlain(t, primary, "AB")
}

// Zig: test "term51: switchScreenMode: 1049 disable without a saved cursor restores defaults"
func TestTm9_switchScreenMode_1049_disable_without_a_saved_cursor_restores_defaults(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 5)
	defer term.Deinit()
	tm9PrimarySetup(t, term)
	term.Modes.Set(ModesModeOrigin, true)
	term.ConfigureCharset(CharsetsSlotsG2, CharsetsCharsetBritish)
	primary := term.Screens.Active
	expect(t, !primary.SavedCursor.Valid, "primary.saved_cursor == null")

	// Switch without 1049 so nothing is saved.
	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	term.Flags.Dirty = TerminalDirty{}
	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1049, false))
	expectEqual(t, ScreenSetKeyPrimary, term.Screens.ActiveKey)
	expectEqual(t, SizeCellCountInt(0), primary.Cursor.X)
	expectEqual(t, SizeCellCountInt(0), primary.Cursor.Y)
	expect(t, primary.Cursor.Style.Default(), "primary.cursor.style.default()")
	expectEqual(t, StyleDefaultID, primary.Cursor.StyleID)
	expect(t, !primary.Cursor.PendingWrap, "!primary.cursor.pending_wrap")
	expect(t, !primary.Cursor.Protected, "!primary.cursor.protected")
	expect(t, !term.Modes.Get(ModesModeOrigin), "!t.modes.get(.origin)")
	expectEqual(t, CharsetsCharsetUtf8, primary.Charset.charsets.Get(CharsetsSlotsG2))
	expectEqual(t, CharsetsSlotsG0, primary.Charset.Gl)
	expect(t, term.Flags.Dirty.Clear, "t.flags.dirty.clear")
	// Cursor shape is not part of the saved state.
	expectEqual(t, ScreenCursorStyleBar, primary.Cursor.CursorStyle)
}

// Zig: test "term51: switchScreenMode: 1049 disable on primary still restores the cursor"
func TestTm9_switchScreenMode_1049_disable_on_primary_still_restores_the_cursor(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 5)
	defer term.Deinit()
	tm9PrintString(t, term, "AB")
	term.SetCursorPos(2, 3)
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagBold))
	term.SaveCursor()

	term.SetCursorPos(5, 9)
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagUnset))
	term.Flags.Dirty = TerminalDirty{}

	// Already on primary: no switch (no dirty), but restoreCursor still runs.
	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1049, false))
	expectEqual(t, ScreenSetKeyPrimary, term.Screens.ActiveKey)
	expect(t, term.Screens.Get(ScreenSetKeyAlternate) == nil, "t.screens.get(.alternate) == null")
	expect(t, !term.Flags.Dirty.Clear, "!t.flags.dirty.clear")
	expectEqual(t, SizeCellCountInt(2), term.Screens.Active.Cursor.X)
	expectEqual(t, SizeCellCountInt(1), term.Screens.Active.Cursor.Y)
	expect(t, term.Screens.Active.Cursor.Style.Flags.Bold, "t.screens.active.cursor.style.flags.bold")
}

// Zig: test "term51: switchScreenMode: 1049 full cycle"
func TestTm9_switchScreenMode_1049_full_cycle(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 5)
	defer term.Deinit()
	tm9PrintString(t, term, "1A")

	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1049, true))
	tm9PrintString(t, term, "full screen app")
	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1049, false))
	// The primary screen is exactly as it was, cursor included.
	tm9ExpectPlain(t, term, "1A")
	tm9PrintString(t, term, "C")
	tm9ExpectPlain(t, term, "1AC")

	// The alternate screen kept its content while inactive ...
	tm9ScreenPlain(t, term.Screens.Get(ScreenSetKeyAlternate), "  full scr\neen app")
	// ... and is erased again on the next entry.
	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1049, true))
	tm9ExpectPlain(t, term, "")
	expectEqual(t, SizeCellCountInt(3), term.Screens.Active.Cursor.X)
}

// Zig: test "term51: switchScreenMode: 1049 enable with alternate protected mode ISO keeps protected cells"
func TestTm9_switchScreenMode_1049_enable_with_alternate_protected_mode_ISO_keeps_protected_cells(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 5)
	defer term.Deinit()
	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode47, true))
	term.SetProtectedMode(ANSIProtectedModeIso)
	tm9PrintString(t, term, "P")
	term.SetProtectedMode(ANSIProtectedModeOff)
	tm9PrintString(t, term, "u")
	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode47, false))

	// protected_mode is per screen and sticky: the erase on entry is a
	// plain ED 2 that preserves protected cells of that screen.
	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1049, true))
	tm9ExpectPlain(t, term, "P")
}

// tm9SwitchScreenModes is `[_]SwitchScreenMode{ .@"47", .@"1047", .@"1049" }`.
var tm9SwitchScreenModes = []TerminalSwitchScreenMode{
	TerminalSwitchScreenMode47,
	TerminalSwitchScreenMode1047,
	TerminalSwitchScreenMode1049,
}

// Zig: test "term51: switchScreenMode: selection of the destination screen is cleared"
func TestTm9_switchScreenMode_selection_of_the_destination_screen_is_cleared(t *testing.T) {
	testStdCapacity(t)
	for _, mode := range tm9SwitchScreenModes {
		func() {
			term := tm9Init(t, 10, 5)
			defer term.Deinit()
			tm9PrintString(t, term, "ABC")
			primary := term.Screens.Active
			tm9Select(t, primary, 0, 0, 2, 0)
			primary.Dirty = ScreenDirty{}

			zigTry(t, term.SwitchScreenMode(mode, true))
			// Selection stays on the screen that was left.
			expect(t, primary.Selection.Valid, "primary.selection != null")
			expect(t, !term.Screens.Active.Selection.Valid, "t.screens.active.selection == null")

			zigTry(t, term.SwitchScreenMode(mode, false))
			// Returning clears the primary screen's selection.
			expect(t, !primary.Selection.Valid, "primary.selection == null")
			expect(t, primary.Dirty.Selection, "primary.dirty.selection")
		}()
	}
}

// Zig: test "term51: switchScreenMode: primary viewport is left alone"
func TestTm9_switchScreenMode_primary_viewport_is_left_alone(t *testing.T) {
	testStdCapacity(t)
	for _, mode := range tm9SwitchScreenModes {
		func() {
			term := tm9Init(t, 5, 3)
			defer term.Deinit()
			tm9PrintString(t, term, "1\n2\n3\n4\n5")
			term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
			primary := term.Screens.Active

			zigTry(t, term.SwitchScreenMode(mode, true))
			expectEqual(t, PageListViewportActive, term.Screens.Active.Pages.viewport.Tag)
			expectEqual(t, PageListViewportTop, primary.Pages.viewport.Tag)
			zigTry(t, term.SwitchScreenMode(mode, false))
			expectEqual(t, PageListViewportTop, primary.Pages.viewport.Tag)
			tm9ExpectPlain(t, term, "1\n2\n3")
		}()
	}
}

// Zig: test "term51: switchScreenMode: hyperlink is never copied and ends on the old screen"
func TestTm9_switchScreenMode_hyperlink_is_never_copied_and_ends_on_the_old_screen(t *testing.T) {
	testStdCapacity(t)
	for _, mode := range tm9SwitchScreenModes {
		func() {
			term := tm9Init(t, 10, 5)
			defer term.Deinit()
			primary := term.Screens.Active
			tm9StartHyperlink(t, primary, "http://example.com")

			zigTry(t, term.SwitchScreenMode(mode, true))
			alt := term.Screens.Active
			expectEqual(t, HyperlinkID(0), primary.Cursor.HyperlinkID)
			expectEqual(t, HyperlinkID(0), alt.Cursor.HyperlinkID)
			expect(t, alt.Cursor.Hyperlink == nil, "alt.cursor.hyperlink == null")

			tm9StartHyperlink(t, alt, "http://example.org")
			zigTry(t, term.SwitchScreenMode(mode, false))
			expectEqual(t, HyperlinkID(0), alt.Cursor.HyperlinkID)
			expectEqual(t, HyperlinkID(0), primary.Cursor.HyperlinkID)
			expect(t, primary.Cursor.Hyperlink == nil, "primary.cursor.hyperlink == null")
		}()
	}
}

// Zig: test "term51: switchScreenMode: charset is carried by every mode"
func TestTm9_switchScreenMode_charset_is_carried_by_every_mode(t *testing.T) {
	testStdCapacity(t)
	for _, mode := range tm9SwitchScreenModes {
		func() {
			term := tm9Init(t, 10, 5)
			defer term.Deinit()
			term.ConfigureCharset(CharsetsSlotsG1, CharsetsCharsetDecSpecial)
			term.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG1, false)

			zigTry(t, term.SwitchScreenMode(mode, true))
			expectEqual(t, CharsetsSlotsG1, term.Screens.Active.Charset.Gl)
			expectEqual(t, CharsetsCharsetDecSpecial, term.Screens.Active.Charset.charsets.Get(CharsetsSlotsG1))
		}()
	}
}

// The test "term51: switchScreenMode: allocation failure when creating the
// alternate screen" is not translated; see scripts/not-translated.txt.

// Zig: test "term51: switchScreenMode: the three modes"
func TestTm9_switchScreenMode_the_three_modes(t *testing.T) {
	// Zig: @typeInfo(SwitchScreenMode).@"enum".fields, written out.
	fields := []TerminalSwitchScreenMode{
		TerminalSwitchScreenMode47,
		TerminalSwitchScreenMode1047,
		TerminalSwitchScreenMode1049,
	}
	expectEqual(t, uint(3), uint(len(fields)))
	expectEqualStrings(t, "47", fields[0].String())
	expectEqualStrings(t, "1047", fields[1].String())
	expectEqualStrings(t, "1049", fields[2].String())
}

// Zig: test "term51: fullReset: returns to the primary screen and drops the alternate"
func TestTm9_fullReset_returns_to_the_primary_screen_and_drops_the_alternate(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 5)
	defer term.Deinit()

	// Nothing to drop on a fresh terminal.
	term.FullReset()
	expectEqual(t, ScreenSetKeyPrimary, term.Screens.ActiveKey)
	expect(t, term.Screens.Get(ScreenSetKeyAlternate) == nil, "t.screens.get(.alternate) == null")

	primary := term.Screens.Active
	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1049, true))
	tm9PrintString(t, term, "alt")
	tm9StartHyperlink(t, term.Screens.Active, "http://example.com")
	tm9Select(t, term.Screens.Active, 0, 0, 2, 0)
	expect(t, term.Screens.Get(ScreenSetKeyAlternate) != nil, "t.screens.get(.alternate) != null")

	term.FullReset()
	expectEqual(t, ScreenSetKeyPrimary, term.Screens.ActiveKey)
	expect(t, term.Screens.Active == primary, "t.screens.active == primary")
	expect(t, term.Screens.Get(ScreenSetKeyAlternate) == nil, "t.screens.get(.alternate) == null")

	// A new alternate screen is created on demand again.
	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	tm9ExpectPlain(t, term, "")
}

// Zig: test "term51: fullReset: primary screen contents cursor and per-screen state"
func TestTm9_fullReset_primary_screen_contents_cursor_and_per_screen_state(t *testing.T) {
	testStdCapacity(t)
	opts := terminalOptionsDefault(10, 3)
	opts.MaxScrollback = 5000
	term := tm9InitOpts(t, opts)
	defer term.Deinit()

	tm9PrintString(t, term, "1\n2\n3\n4\n5")
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	s := term.Screens.Active
	s.Cursor.CursorStyle = ScreenCursorStyleBar
	tm9SetAttribute(t, term, tm9Attr(SGRAttributeTagBold))
	term.SetProtectedMode(ANSIProtectedModeIso)
	tm9StartHyperlink(t, s, "http://example.com")
	s.Cursor.SemanticContent = PageCellSemanticContentInput
	s.Cursor.SemanticContentClearEol = true
	s.Cursor.PendingWrap = true
	s.SemanticPrompt = ScreenSemanticPrompt{Seen: true, Click: ScreenSemanticPromptSemanticClick{Tag: ScreenSemanticPromptSemanticClickClickEvents}}
	s.KittyKeyboard.Push(KittyKeyFlags{Disambiguate: true, ReportAll: true})
	term.ConfigureCharset(CharsetsSlotsG1, CharsetsCharsetDecSpecial)
	term.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG1, true)
	term.SaveCursor()
	tm9Select(t, s, 0, 0, 2, 0)
	s.Dirty = ScreenDirty{}

	term.FullReset()

	// Same screen object, emptied: no history and the viewport is back on
	// the active area.
	expect(t, term.Screens.Active == s, "t.screens.active == s")
	tm9ExpectPlain(t, term, "")
	expectEqual(t, uint(3), s.Pages.total_rows)
	expectEqual(t, PageListViewportActive, s.Pages.viewport.Tag)
	expectEqual(t, uint64(5000), s.Pages.explicit_max_size)
	expect(t, !s.NoScrollback, "!s.no_scrollback")

	// The cursor is entirely default, including its shape.
	expectEqual(t, SizeCellCountInt(0), s.Cursor.X)
	expectEqual(t, SizeCellCountInt(0), s.Cursor.Y)
	expectEqual(t, ScreenCursorStyleBlock, s.Cursor.CursorStyle)
	expect(t, !s.Cursor.PendingWrap, "!s.cursor.pending_wrap")
	expect(t, !s.Cursor.Protected, "!s.cursor.protected")
	expect(t, s.Cursor.Style.Default(), "s.cursor.style.default()")
	expectEqual(t, StyleDefaultID, s.Cursor.StyleID)
	expectEqual(t, HyperlinkID(0), s.Cursor.HyperlinkID)
	expectEqual(t, SizeOffsetInt(0), s.Cursor.HyperlinkImplicitID)
	expect(t, s.Cursor.Hyperlink == nil, "s.cursor.hyperlink == null")
	expectEqual(t, PageCellSemanticContentOutput, s.Cursor.SemanticContent)
	expect(t, !s.Cursor.SemanticContentClearEol, "!s.cursor.semantic_content_clear_eol")
	expect(t, s.Cursor.PagePin.Node == s.Pages.pages.first, "s.cursor.page_pin.node == s.pages.pages.first.?")
	expectEqual(t, SizeCellCountInt(0), s.Cursor.PagePin.X)
	expectEqual(t, SizeCellCountInt(0), s.Cursor.PagePin.Y)

	// Everything else on the screen.
	expect(t, !s.SavedCursor.Valid, "s.saved_cursor == null")
	expect(t, !s.Selection.Valid, "s.selection == null")
	expect(t, s.Dirty.Selection, "s.dirty.selection")
	expectEqual(t, ANSIProtectedModeOff, s.ProtectedMode)
	expect(t, !s.SemanticPrompt.Seen, "!s.semantic_prompt.seen")
	expect(t, s.SemanticPrompt.Click.Tag == ScreenSemanticPromptSemanticClickNone, "s.semantic_prompt.click == .none")
	expectEqual(t, uint8(0), s.KittyKeyboard.idx)
	expectEqual(t, uint8(0), s.KittyKeyboard.Current().Int())
	expectEqual(t, CharsetsSlotsG0, s.Charset.Gl)
	expectEqual(t, CharsetsSlotsG2, s.Charset.Gr)
	expect(t, !s.Charset.SingleShift.Valid, "s.charset.single_shift == null")
	expectEqual(t, CharsetsCharsetUtf8, s.Charset.charsets.Get(CharsetsSlotsG1))
}

// Zig: test "term51: fullReset: selection dirty flag only set when there was a selection"
func TestTm9_fullReset_selection_dirty_flag_only_set_when_there_was_a_selection(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 3)
	defer term.Deinit()
	term.Screens.Active.Dirty = ScreenDirty{}
	term.FullReset()
	expect(t, !term.Screens.Active.Dirty.Selection, "!t.screens.active.dirty.selection")
	expect(t, !term.Screens.Active.Dirty.HyperlinkHover, "!t.screens.active.dirty.hyperlink_hover")
}

// Zig: test "term51: fullReset: terminal flags are reset and dirty clear is set"
func TestTm9_fullReset_terminal_flags_are_reset_and_dirty_clear_is_set(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 5)
	defer term.Deinit()

	term.Flags = TerminalFlags{
		ShellRedrawsPrompt:  OSCParsersSemanticPromptRedrawFalse,
		ModifyOtherKeys2:    true,
		MouseEvent:          TerminalMouseEventsAny,
		MouseFormat:         TerminalMouseFormatSgrPixels,
		MouseShiftCapture:   TerminalFlagsMouseShiftCaptureTrue,
		Focused:             false,
		PasswordInput:       true,
		SelectionScroll:     true,
		SearchViewportDirty: true,
		Dirty:               TerminalDirty{Palette: true, ReverseColors: true, Clear: false, Preedit: true},
	}
	term.FullReset()

	expect(t, term.Flags.ShellRedrawsPrompt == OSCParsersSemanticPromptRedrawTrue, "t.flags.shell_redraws_prompt == .true")
	expect(t, !term.Flags.ModifyOtherKeys2, "!t.flags.modify_other_keys_2")
	expectEqual(t, TerminalMouseEventsNone, term.Flags.MouseEvent)
	expectEqual(t, TerminalMouseFormatX10, term.Flags.MouseFormat)
	expect(t, term.Flags.MouseShiftCapture == TerminalFlagsMouseShiftCaptureNull, "t.flags.mouse_shift_capture == .null")
	// Focus is reset to focused even if the window is not.
	expect(t, term.Flags.Focused, "t.flags.focused")
	expect(t, !term.Flags.PasswordInput, "!t.flags.password_input")
	expect(t, !term.Flags.SelectionScroll, "!t.flags.selection_scroll")
	expect(t, !term.Flags.SearchViewportDirty, "!t.flags.search_viewport_dirty")
	// Dirty flags are all cleared except `clear`.
	expect(t, term.Flags.Dirty.Clear, "t.flags.dirty.clear")
	expect(t, !term.Flags.Dirty.Palette, "!t.flags.dirty.palette")
	expect(t, !term.Flags.Dirty.ReverseColors, "!t.flags.dirty.reverse_colors")
	expect(t, !term.Flags.Dirty.Preedit, "!t.flags.dirty.preedit")
}

// Zig: test "term51: fullReset: modes return to the defaults and saved modes are cleared"
func TestTm9_fullReset_modes_return_to_the_defaults_and_saved_modes_are_cleared(t *testing.T) {
	testStdCapacity(t)

	// Default defaults.
	{
		term := tm9Init(t, 10, 5)
		defer term.Deinit()
		defaults := term.Modes.Default
		// Flip every mode, and mark every mode as saved.
		term.Modes.Values = tm9ModesNot(defaults)
		term.Modes.Saved = tm9ModesNot(ModesModePacked{})
		term.FullReset()
		expectEqual(t, defaults, term.Modes.Values)
		// Saved modes are reset to the struct's own field defaults, which
		// are the same as the terminal defaults only for the default
		// `default_modes` (see the custom defaults case below).
		expectEqual(t, ModesModePackedInit(), term.Modes.Saved)
		expectEqual(t, defaults, term.Modes.Default)
		expect(t, term.Modes.Get(ModesModeWraparound), "t.modes.get(.wraparound)")
		expect(t, term.Modes.Get(ModesModeCursorVisible), "t.modes.get(.cursor_visible)")
		expect(t, !term.Modes.Get(ModesModeOrigin), "!t.modes.get(.origin)")
	}

	// Custom defaults are what the reset returns to.
	{
		opts := terminalOptionsDefault(10, 5)
		opts.DefaultModes = ModesModePackedInit()
		sm1_modesFieldSet(&opts.DefaultModes, "wraparound", false)
		sm1_modesFieldSet(&opts.DefaultModes, "grapheme_cluster", true)
		sm1_modesFieldSet(&opts.DefaultModes, "insert", true)
		term := tm9InitOpts(t, opts)
		defer term.Deinit()
		expect(t, !term.Modes.Get(ModesModeWraparound), "!t.modes.get(.wraparound)")
		term.Modes.Set(ModesModeWraparound, true)
		term.Modes.Set(ModesModeGraphemeCluster, false)
		term.Modes.Set(ModesModeOrigin, true)
		term.Modes.Save(ModesModeOrigin)
		term.FullReset()
		expect(t, !term.Modes.Get(ModesModeWraparound), "!t.modes.get(.wraparound)")
		expect(t, term.Modes.Get(ModesModeGraphemeCluster), "t.modes.get(.grapheme_cluster)")
		expect(t, term.Modes.Get(ModesModeInsert), "t.modes.get(.insert)")
		expect(t, !term.Modes.Get(ModesModeOrigin), "!t.modes.get(.origin)")
		// The saved value is gone: restoring brings back false, not true.
		term.Modes.Set(ModesModeOrigin, true)
		expect(t, !term.Modes.Restore(ModesModeOrigin), "!t.modes.restore(.origin)")
		// Upstream quirk: the saved set is cleared to ModePacked{} and not
		// to the terminal's defaults, so restoring a mode whose default
		// differs from the struct default (wraparound) yields the struct
		// default (true), not the terminal default (false).
		term.Modes.Set(ModesModeWraparound, false)
		expect(t, term.Modes.Restore(ModesModeWraparound), "t.modes.restore(.wraparound)")
		expect(t, !sm1_modesField(&term.Modes.Default, "wraparound"), "!t.modes.default.wraparound")
	}
}

// Zig: test "term51: fullReset: tabstops scroll region previous char pwd and status display"
func TestTm9_fullReset_tabstops_scroll_region_previous_char_pwd_and_status_display(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 20, 6)
	defer term.Deinit()

	term.TabClear(CSITabClearAll)
	term.Tabstops.Set(5)
	expect(t, !term.Tabstops.Get(8), "!t.tabstops.get(8)")
	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetTopAndBottomMargin(2, 4)
	term.SetLeftAndRightMargin(3, 8)
	zigTry(t, term.Print('A'))
	expectEqual(t, Some(uint32('A')), term.PreviousChar)
	zigTry(t, term.SetPwd([]uint8("file://host/very/long/path/that/has/capacity")))
	pwd_cap := uint(cap(term.Pwd))
	term.StatusDisplay = ANSIStatusDisplayStatusLine

	term.FullReset()
	expect(t, !term.Tabstops.Get(5), "!t.tabstops.get(5)")
	expect(t, term.Tabstops.Get(8), "t.tabstops.get(8)")
	expect(t, term.Tabstops.Get(16), "t.tabstops.get(16)")
	expect(t, !term.Tabstops.Get(17), "!t.tabstops.get(17)")
	expect(t, !term.Tabstops.Get(19), "!t.tabstops.get(19)")
	expectEqual(t, uint(20), term.Tabstops.cols)
	tm9ExpectRegion(t, term, 0, 5, 0, 19)
	expect(t, !term.PreviousChar.Valid, "t.previous_char == null")
	expect(t, !term.GetPwd().Valid, "t.getPwd() == null")
	expectEqual(t, pwd_cap, uint(cap(term.Pwd)))
	expectEqual(t, ANSIStatusDisplayMain, term.StatusDisplay)
	// The margin mode was reset with the other modes.
	expect(t, !term.Modes.Get(ModesModeEnableLeftAndRightMargin), "!t.modes.get(.enable_left_and_right_margin)")
}

// Zig: test "term51: fullReset: region follows the current size after a resize"
func TestTm9_fullReset_region_follows_the_current_size_after_a_resize(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 10, 5)
	defer term.Deinit()
	zigTry(t, term.Resize(7, 4))
	term.SetTopAndBottomMargin(2, 3)
	term.FullReset()
	tm9ExpectRegion(t, term, 0, 3, 0, 6)
	expectEqual(t, SizeCellCountInt(7), term.Cols)
	expectEqual(t, SizeCellCountInt(4), term.Rows)
	expectEqual(t, uint(7), term.Tabstops.cols)
	expect(t, term.Tabstops.Get(0) == false, "t.tabstops.get(0) == false")
}

// Zig: test "term51: fullReset: size colors mouse shape and default modes are kept"
func TestTm9_fullReset_size_colors_mouse_shape_and_default_modes_are_kept(t *testing.T) {
	testStdCapacity(t)
	opts := terminalOptionsDefault(10, 5)
	opts.Colors = TerminalColors{
		Background: ColorDynamicRGBInit(ColorRGB{R: 1, G: 2, B: 3}),
		Foreground: ColorDynamicRGBInit(ColorRGB{R: 4, G: 5, B: 6}),
		Cursor:     ColorDynamicRGBUnset,
		Palette:    ColorDynamicPaletteDefault,
	}
	opts.DefaultModes = ModesModePackedInit()
	sm1_modesFieldSet(&opts.DefaultModes, "grapheme_cluster", true)
	term := tm9InitOpts(t, opts)
	defer term.Deinit()

	term.WidthPx = 800
	term.HeightPx = 600
	term.MouseShape = MouseShapePointer
	term.Colors.Background.Set(ColorRGB{R: 9, G: 9, B: 9})
	term.Colors.Cursor.Set(ColorRGB{R: 7, G: 7, B: 7})
	term.Colors.Palette.Set(5, ColorRGB{R: 10, G: 20, B: 30})

	term.FullReset()

	expectEqual(t, SizeCellCountInt(10), term.Cols)
	expectEqual(t, SizeCellCountInt(5), term.Rows)
	expectEqual(t, uint32(800), term.WidthPx)
	expectEqual(t, uint32(600), term.HeightPx)
	expectEqual(t, MouseShapePointer, term.MouseShape)
	// Dynamic colors, including overrides, survive a reset.
	expectEqual(t, ColorRGB{R: 9, G: 9, B: 9}, zigUnwrap(t, term.Colors.Background.Get()))
	expectEqual(t, ColorRGB{R: 4, G: 5, B: 6}, zigUnwrap(t, term.Colors.Foreground.Get()))
	expectEqual(t, ColorRGB{R: 7, G: 7, B: 7}, zigUnwrap(t, term.Colors.Cursor.Get()))
	expectEqual(t, ColorRGB{R: 10, G: 20, B: 30}, term.Colors.Palette.Current[5])
	expect(t, term.Colors.Palette.Mask.isSet(5), "t.colors.palette.mask.isSet(5)")
	// The palette flag is not raised by a reset.
	expect(t, !term.Flags.Dirty.Palette, "!t.flags.dirty.palette")
	expect(t, term.Modes.Get(ModesModeGraphemeCluster), "t.modes.get(.grapheme_cluster)")
}

// Zig: test "term51: fullReset: terminal is usable afterwards"
func TestTm9_fullReset_terminal_is_usable_afterwards(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 3)
	defer term.Deinit()

	tm9PrintString(t, term, "1\n2\n3\n4\n5\n6\n7")
	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode1049, true))
	tm9PrintString(t, term, "alt")
	term.FullReset()

	tm9PrintString(t, term, "new")
	tm9ExpectPlain(t, term, "new")
	expectEqual(t, SizeCellCountInt(3), term.Screens.Active.Cursor.X)
	term.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	tm9ExpectPlain(t, term, "new")

	// Reset again, with the tracked cursor pin staying valid.
	term.FullReset()
	expect(t, term.Screens.Active.Pages.PinIsValid(*term.Screens.Active.Cursor.PagePin), "t.screens.active.pages.pinIsValid(t.screens.active.cursor.page_pin.*)")
	tm9ExpectPlain(t, term, "")
}

// Zig: test "term51: fullReset: raises the clear flag without marking grid rows dirty"
func TestTm9_fullReset_raises_the_clear_flag_without_marking_grid_rows_dirty(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 3)
	defer term.Deinit()
	tm9PrintString(t, term, "abc")
	expect(t, term.isDirty(tm9Active(0, 0)), "t.isDirty(.{ .active = .{} })")
	term.Flags.Dirty = TerminalDirty{}
	term.FullReset()
	expect(t, term.Flags.Dirty.Clear, "t.flags.dirty.clear")
	// The screen was rebuilt from fresh pages: the renderer is told to clear
	// through the flag, the new rows themselves are not dirty.
	for y := uint32(0); y < 3; y++ {
		expect(t, !term.isDirty(tm9Active(0, y)), "!t.isDirty(.{ .active = .{ .y = @intCast(y) } })")
	}
}

// Zig: test "term51: fullReset: cursor pin is flagged garbage by the page reset"
func TestTm9_fullReset_cursor_pin_is_flagged_garbage_by_the_page_reset(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 3)
	defer term.Deinit()
	tm9PrintString(t, term, "1\n2\n3\n4\n5")
	expect(t, !term.Screens.Active.Cursor.PagePin.Garbage, "!t.screens.active.cursor.page_pin.garbage")

	// PageList.reset flags every tracked pin, including the cursor's, as
	// garbage even though the cursor pin is repositioned to the top-left.
	term.FullReset()
	expect(t, term.Screens.Active.Cursor.PagePin.Garbage, "t.screens.active.cursor.page_pin.garbage")
	expectEqual(t, SizeCellCountInt(0), term.Screens.Active.Cursor.PagePin.X)
	expectEqual(t, SizeCellCountInt(0), term.Screens.Active.Cursor.PagePin.Y)
	// The viewport pin is exempt.
	expect(t, !term.Screens.Active.Pages.viewport_pin.Garbage, "!t.screens.active.pages.viewport_pin.garbage")
}

// ---------------------------------------------------------------------------
// Go-only panic tests: assert, unreachable and panic paths that the Zig tests
// cannot exercise (an assert aborts the test).
// ---------------------------------------------------------------------------

// Go-only: panic test: Terminal.isDirty: point outside the screen (getCell returns null, `.?` unwraps null)
func TestTm9_panic_isDirty_point_outside_the_screen(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()
	tm9ExpectPanic(t, func() { term.isDirty(tm9Active(5, 0)) })
	tm9ExpectPanic(t, func() { term.isDirty(tm9Active(0, 5)) })
}

// Go-only: panic test: Terminal.switchScreen: assert(new.cursor.hyperlink_id == 0) when the destination screen has an active hyperlink
func TestTm9_panic_switchScreen_destination_screen_has_a_hyperlink(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	defer term.Deinit()

	// Create the alternate screen, come back, and start a hyperlink on the
	// inactive alternate screen directly: switchScreen only ends the
	// hyperlink of the screen being left.
	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	_, err = term.SwitchScreen(ScreenSetKeyPrimary)
	zigTry(t, err)
	tm9StartHyperlink(t, term.Screens.Get(ScreenSetKeyAlternate), "http://example.com")

	tm9ExpectPanic(t, func() { _, _ = term.SwitchScreen(ScreenSetKeyAlternate) })
}

// Go-only: panic test: Terminal.resize: zero rows (PageList.resize asserts opts.rows > 0)
func TestTm9_panic_resize_zero_rows(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	// Go: an error since the zero-size ruling; the test keeps its name.
	if err := error(term.Resize(5, 0)); err != ErrZeroRowCount {
		t.Fatalf("got %v, want %v", err, ErrZeroRowCount)
	}
}

// Go-only: panic test: Terminal.resize: zero columns (PageList.resize asserts opts.cols > 0)
func TestTm9_panic_resize_zero_columns(t *testing.T) {
	testStdCapacity(t)
	term := tm9Init(t, 5, 5)
	// Go: an error since the zero-size ruling; the test keeps its name.
	if err := error(term.Resize(0, 5)); err != ErrZeroColCount {
		t.Fatalf("got %v, want %v", err, ErrZeroColCount)
	}
}
