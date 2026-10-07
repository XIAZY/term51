// Tests translated one for one from src/terminal/Terminal.zig (Ghostty
// v1.3.1 plus the term51 additional tests), lines 16108 through 19357 of the
// patched source. Each Go test carries the exact Zig title in a
// "// Zig: test" comment. Helpers are prefixed tm5 (the Zig helpers
// term51S* and term51T* are tm5S* and tm5T*).
//
// Translation notes:
//   - The testing.T is `t`; the Terminal, `t` in Zig, is `tm`.
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualStrings keep Zig's names and Zig's
//     (expected, actual) argument order. `try` on a call that returns an
//     error is zigTry(t, err); `x.?` is zigUnwrap or a nil check.
//   - `try init(alloc, .{ .cols = c, .rows = r })` is tm5Init(t, c, r), which
//     starts from terminalOptionsDefault; the terminal is returned by
//     pointer and `defer t.deinit(alloc)` is `defer tm.Deinit()`.
//   - A Zig block `{ var t = ...; defer t.deinit(...); ... }` and a loop body
//     with a defer are Go function literals called at once, so that the
//     deferred Deinit runs at the end of the block.
//   - `errdefer std.debug.print(...)` is a deferred t.Logf that only prints
//     when the test failed.
//   - Zig's `std.math.maxInt(usize)` is math.MaxUint.
//   - `.{ .kitty_graphics }` blocks (`if (comptime
//     build_options.kitty_graphics)`) are compiled out of the library build
//     and carry `// Zig-only:` comments.
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt.

package terminal

import (
	"math"
	"math/bits"
	"testing"
)

// tm5ExpectPanic runs f, which must panic. A panic with the wireframe's
// "not implemented" value does not count: it only means the function under
// test does not exist yet. The setup is done outside f.
func tm5ExpectPanic(t *testing.T, f func()) {
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

// tm5Init is `try init(alloc, .{ .cols = cols, .rows = rows })`: the Zig
// defaults max_scrollback 10_000, colors .default and default_modes .{}.
func tm5Init(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) *Terminal {
	t.Helper()
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(cols, rows))
	zigTry(t, err)
	return &tm
}

// tm5SwitchScreen is `_ = try t.switchScreen(key)`.
func tm5SwitchScreen(t *testing.T, tm *Terminal, key ScreenSetKey) {
	t.Helper()
	_, err := tm.SwitchScreen(key)
	zigTry(t, err)
}

// tm5Attr is `try t.screens.active.setAttribute(attr)`.
func tm5Attr(t *testing.T, tm *Terminal, attr SGRAttribute) {
	t.Helper()
	zigTry(t, tm.Screens.Active.SetAttribute(attr))
}

// tm5ActivePoint is `.{ .active = .{ .x = x, .y = y } }`.
func tm5ActivePoint(x uint, y uint) Point {
	return Point{Tag: PointTagActive, Active: PointCoordinate{X: SizeCellCountInt(x), Y: uint32(y)}}
}

// tm5SAt is term51SAt: the page list cell at the given active-area
// position.
func tm5SAt(t *testing.T, tm *Terminal, x uint, y uint) PageListCell {
	t.Helper()
	return zigUnwrap(t, tm.Screens.Active.Pages.GetCell(tm5ActivePoint(x, y)))
}

// tm5SExpectCell is term51SExpectCell: expect the cell at (x, y) to hold
// exactly the codepoint and wide state.
func tm5SExpectCell(t *testing.T, tm *Terminal, x uint, y uint, cp uint32, wide PageCellWide) {
	t.Helper()
	lc := tm5SAt(t, tm, x, y)
	expectEqual(t, cp, lc.Cell.Codepoint())
	expectEqual(t, wide, lc.Cell.wide)
}

// tm5SExpectGrapheme is term51SExpectGrapheme: expect the cell at (x, y) to
// carry exactly this grapheme tail.
func tm5SExpectGrapheme(t *testing.T, tm *Terminal, x uint, y uint, expected []uint32) {
	t.Helper()
	lc := tm5SAt(t, tm, x, y)
	expect(t, lc.Cell.HasGrapheme(), "lc.cell.hasGrapheme()")
	expect(t, lc.Row.grapheme, "lc.row.grapheme")
	expectEqualSlices(t, expected, zigUnwrap(t, lc.Node.data.LookupGrapheme(lc.Cell)))
}

// tm5SExpectNoGrapheme is term51SExpectNoGrapheme: expect the cell at (x, y)
// to carry no grapheme data.
func tm5SExpectNoGrapheme(t *testing.T, tm *Terminal, x uint, y uint) {
	t.Helper()
	lc := tm5SAt(t, tm, x, y)
	expect(t, !lc.Cell.HasGrapheme(), "!lc.cell.hasGrapheme()")
	expect(t, !lc.Node.data.LookupGrapheme(lc.Cell).Valid, "lc.node.data.lookupGrapheme(lc.cell) == null")
}

// tm5SExpectCursor is term51SExpectCursor: expect the cursor position and
// pending wrap state.
func tm5SExpectCursor(t *testing.T, tm *Terminal, x uint, y uint, pending_wrap bool) {
	t.Helper()
	expectEqual(t, x, uint(tm.Screens.Active.Cursor.X))
	expectEqual(t, y, uint(tm.Screens.Active.Cursor.Y))
	expectEqual(t, pending_wrap, tm.Screens.Active.Cursor.PendingWrap)
}

// tm5TLabel is term51TLabel: the label for the cell at (x, y) of a grid
// `cols` wide: 'A'-'Z' then 'a'-'z', then wrapping. Unique for up to 52
// cells.
func tm5TLabel(cols uint, x uint, y uint) uint32 {
	n := (y*cols + x) % 52
	if n < 26 {
		return uint32('A' + n)
	}
	return uint32('a' + (n - 26))
}

// tm5TFill is term51TFill: fill the whole active area with unique labels
// (see tm5TLabel) and leave the cursor at the home position with no pending
// wrap. The terminal must not be in origin mode.
func tm5TFill(t *testing.T, tm *Terminal) {
	t.Helper()
	for y := uint(0); y < uint(tm.Rows); y++ {
		for x := uint(0); x < uint(tm.Cols); x++ {
			tm.SetCursorPos(y+1, x+1)
			zigTry(t, tm.Print(tm5TLabel(uint(tm.Cols), x, y)))
		}
	}
	tm.SetCursorPos(1, 1)
}

// tm5TRegion is term51TRegion: set the scrolling region directly (all
// values 0-indexed, inclusive).
func tm5TRegion(tm *Terminal, top uint, bottom uint, left uint, right uint) {
	tm.ScrollingRegion = TerminalScrollingRegion{
		Top:    SizeCellCountInt(top),
		Bottom: SizeCellCountInt(bottom),
		Left:   SizeCellCountInt(left),
		Right:  SizeCellCountInt(right),
	}
}

// tm5TCellAt is term51TCellAt.
func tm5TCellAt(t *testing.T, tm *Terminal, x uint, y uint) PageCell {
	t.Helper()
	return *zigUnwrap(t, tm.Screens.Active.Pages.GetCell(tm5ActivePoint(x, y))).Cell
}

// tm5TRowAt is term51TRowAt.
func tm5TRowAt(t *testing.T, tm *Terminal, y uint) *PageRow {
	t.Helper()
	return zigUnwrap(t, tm.Screens.Active.Pages.GetCell(tm5ActivePoint(0, y))).Row
}

// tm5TGridRow is term51TGridRow: one character per cell: '.' for an empty
// narrow cell, '#' for a cell that only carries a background colour, the
// ASCII character for a narrow cell with a codepoint below 128 ('?' above),
// 'W' for a wide cell, 'w' for a wide cell with no codepoint, '_' for a
// spacer tail, '>' for a spacer head.
func tm5TGridRow(t *testing.T, tm *Terminal, y uint) []uint8 {
	t.Helper()
	buf := make([]uint8, tm.Cols)
	for x := uint(0); x < uint(tm.Cols); x++ {
		c := tm5TCellAt(t, tm, x, y)
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
	return buf
}

// tm5TExpectGrid is term51TExpectGrid: compare the whole active area
// against `expected`, one string per row.
func tm5TExpectGrid(t *testing.T, tm *Terminal, expected []string) {
	t.Helper()
	expectEqual(t, uint(tm.Rows), uint(len(expected)))
	for y, row := range expected {
		expectEqualStrings(t, row, string(tm5TGridRow(t, tm, uint(y))))
	}
}

// tm5TExpectCursor is term51TExpectCursor.
func tm5TExpectCursor(t *testing.T, tm *Terminal, x uint, y uint) {
	t.Helper()
	expectEqual(t, x, uint(tm.Screens.Active.Cursor.X))
	expectEqual(t, y, uint(tm.Screens.Active.Cursor.Y))
}

// tm5TExpectRegion is term51TExpectRegion.
func tm5TExpectRegion(t *testing.T, tm *Terminal, top uint, bottom uint, left uint, right uint) {
	t.Helper()
	expectEqual(t, top, uint(tm.ScrollingRegion.Top))
	expectEqual(t, bottom, uint(tm.ScrollingRegion.Bottom))
	expectEqual(t, left, uint(tm.ScrollingRegion.Left))
	expectEqual(t, right, uint(tm.ScrollingRegion.Right))
}

// tm5TScrollbackRows is term51TScrollbackRows: the number of rows above the
// active area.
func tm5TScrollbackRows(tm *Terminal) uint {
	return tm.Screens.Active.Pages.total_rows - uint(tm.Rows)
}

// tm5TTabString is term51TTabString: the set tabstops of the terminal as a
// string of '^' (set) and '.'.
func tm5TTabString(tm *Terminal) string {
	buf := make([]uint8, tm.Cols)
	for x := uint(0); x < uint(tm.Cols); x++ {
		if tm.Tabstops.Get(x) {
			buf[x] = '^'
		} else {
			buf[x] = '.'
		}
	}
	return string(buf)
}

// tm5TPrompt is term51TPrompt.
func tm5TPrompt(action OSCParsersSemanticPromptCommandAction, opts string) OSCCommandSemanticPrompt {
	return OSCCommandSemanticPrompt{Action: action, OptionsUnvalidated: []uint8(opts)}
}

// tm5ClickNone, tm5ClickEvents and tm5ClickCl build Screen.SemanticPrompt.
// SemanticClick values: `.none`, `.click_events` and `.{ .cl = v }`.
func tm5ClickNone() ScreenSemanticPromptSemanticClick {
	return ScreenSemanticPromptSemanticClick{Tag: ScreenSemanticPromptSemanticClickNone}
}

func tm5ClickEvents() ScreenSemanticPromptSemanticClick {
	return ScreenSemanticPromptSemanticClick{Tag: ScreenSemanticPromptSemanticClickClickEvents}
}

func tm5ClickCl(v OSCParsersSemanticPromptClick) ScreenSemanticPromptSemanticClick {
	return ScreenSemanticPromptSemanticClick{Tag: ScreenSemanticPromptSemanticClickCl, Cl: v}
}

// tm5ExpectClick is `try testing.expectEqual(want, got)` on a SemanticClick:
// a tagged union compares its tag and the payload the tag names only.
func tm5ExpectClick(t *testing.T, want ScreenSemanticPromptSemanticClick, got ScreenSemanticPromptSemanticClick) {
	t.Helper()
	if want.Tag != got.Tag {
		t.Fatalf("got click tag %v, want %v", got.Tag, want.Tag)
	}
	if want.Tag == ScreenSemanticPromptSemanticClickCl && want.Cl != got.Cl {
		t.Fatalf("got click cl %v, want %v", got.Cl, want.Cl)
	}
}

// tm5ExpectOptNull is `try testing.expectEqual(@as(?T, null), actual)`.
func tm5ExpectOptNull[T any](t *testing.T, got Optional[T]) {
	t.Helper()
	if got.Valid {
		t.Fatalf("got %v, want null", got.Value)
	}
}

// tm5ExpectOptSome is `try testing.expectEqual(@as(?T, want), actual)`.
func tm5ExpectOptSome[T comparable](t *testing.T, want T, got Optional[T]) {
	t.Helper()
	if !got.Valid || got.Value != want {
		t.Fatalf("got %v (valid %v), want %v", got.Value, got.Valid, want)
	}
}

// Zig: test "term51: cursorRight: right of the right margin goes to the last column"
func TestTm5_term51_cursorRight_right_of_the_right_margin_goes_to_the_last_column(t *testing.T) {
	tm := tm5Init(t, 10, 3)
	defer tm.Deinit()
	tm.ScrollingRegion.Left = 2
	tm.ScrollingRegion.Right = 6

	tm.Screens.Active.CursorAbsolute(7, 1)
	tm.CursorRight(1)
	tm5SExpectCursor(t, tm, 8, 1, false)
	tm.CursorRight(100)
	tm5SExpectCursor(t, tm, 9, 1, false)
	tm.CursorRight(1)
	tm5SExpectCursor(t, tm, 9, 1, false)
}

// Zig: test "term51: cursorRight: always clears pending wrap and ignores origin mode"
func TestTm5_term51_cursorRight_always_clears_pending_wrap_and_ignores_origin_mode(t *testing.T) {
	tm := tm5Init(t, 5, 3)
	defer tm.Deinit()

	zigTry(t, tm.PrintString([]uint8("abcde")))
	tm5SExpectCursor(t, tm, 4, 0, true)
	tm.CursorRight(1)
	tm5SExpectCursor(t, tm, 4, 0, false)

	tm.ScrollingRegion.Left = 1
	tm.ScrollingRegion.Right = 3
	tm.Modes.Set(ModesModeOrigin, true)
	tm.Screens.Active.CursorAbsolute(1, 0)
	tm.Screens.Active.Cursor.PendingWrap = true
	tm.CursorRight(1)
	tm5SExpectCursor(t, tm, 2, 0, false)
}

// Zig: test "term51: cursorLeft: without wraparound moves left and stops at column zero"
func TestTm5_term51_cursorLeft_without_wraparound_moves_left_and_stops_at_column_zero(t *testing.T) {
	tm := tm5Init(t, 10, 3)
	defer tm.Deinit()
	tm.Modes.Set(ModesModeWraparound, false)

	tm.Screens.Active.CursorAbsolute(8, 1)
	tm.CursorLeft(2)
	tm5SExpectCursor(t, tm, 6, 1, false)
	tm.CursorLeft(0)
	tm5SExpectCursor(t, tm, 5, 1, false)
	tm.CursorLeft(1)
	tm5SExpectCursor(t, tm, 4, 1, false)
	tm.CursorLeft(100)
	tm5SExpectCursor(t, tm, 0, 1, false)
	tm.CursorLeft(1)
	tm5SExpectCursor(t, tm, 0, 1, false)
	tm.Screens.Active.CursorAbsolute(9, 1)
	tm.CursorLeft(math.MaxUint)
	tm5SExpectCursor(t, tm, 0, 1, false)
	tm.Screens.Active.CursorAbsolute(9, 1)
	tm.CursorLeft(9)
	tm5SExpectCursor(t, tm, 0, 1, false)
	tm.Screens.Active.CursorAbsolute(9, 1)
	tm.CursorLeft(10)
	tm5SExpectCursor(t, tm, 0, 1, false)
}

// Zig: test "term51: cursorLeft: reverse wrap flags are ignored without wraparound"
func TestTm5_term51_cursorLeft_reverse_wrap_flags_are_ignored_without_wraparound(t *testing.T) {
	tm := tm5Init(t, 5, 3)
	defer tm.Deinit()
	tm.Modes.Set(ModesModeWraparound, false)
	tm.Modes.Set(ModesModeReverseWrap, true)
	tm.Modes.Set(ModesModeReverseWrapExtended, true)

	tm.Screens.Active.CursorAbsolute(1, 1)
	tm.CursorLeft(5)
	tm5SExpectCursor(t, tm, 0, 1, false)
}

// Zig: test "term51: cursorLeft: with wraparound but no reverse wrap behaves as plain left"
func TestTm5_term51_cursorLeft_with_wraparound_but_no_reverse_wrap_behaves_as_plain_left(t *testing.T) {
	tm := tm5Init(t, 5, 3)
	defer tm.Deinit()

	zigTry(t, tm.PrintString([]uint8("abcde")))
	tm5SExpectCursor(t, tm, 4, 0, true)
	// The pending wrap does not reduce the distance in this mode.
	tm.CursorLeft(1)
	tm5SExpectCursor(t, tm, 3, 0, false)

	// No wrap to the previous line.
	tm.Screens.Active.CursorAbsolute(1, 1)
	tm.CursorLeft(5)
	tm5SExpectCursor(t, tm, 0, 1, false)
}

// Zig: test "term51: cursorLeft: plain left ignores the left margin"
func TestTm5_term51_cursorLeft_plain_left_ignores_the_left_margin(t *testing.T) {
	tm := tm5Init(t, 10, 3)
	defer tm.Deinit()
	tm.ScrollingRegion.Left = 4
	tm.ScrollingRegion.Right = 8

	tm.Screens.Active.CursorAbsolute(6, 1)
	tm.CursorLeft(5)
	tm5SExpectCursor(t, tm, 1, 1, false)
}

// Zig: test "term51: cursorLeft: reverse wrap within a line and pending wrap"
func TestTm5_term51_cursorLeft_reverse_wrap_within_a_line_and_pending_wrap(t *testing.T) {
	tm := tm5Init(t, 5, 3)
	defer tm.Deinit()
	tm.Modes.Set(ModesModeReverseWrap, true)

	tm.Screens.Active.CursorAbsolute(4, 0)
	tm.CursorLeft(2)
	tm5SExpectCursor(t, tm, 2, 0, false)
	tm.CursorLeft(0)
	tm5SExpectCursor(t, tm, 1, 0, false)

	// A pending wrap uses up one unit of the count.
	tm.Screens.Active.CursorAbsolute(4, 1)
	tm.Screens.Active.Cursor.PendingWrap = true
	tm.CursorLeft(1)
	tm5SExpectCursor(t, tm, 4, 1, false)
	tm.Screens.Active.Cursor.PendingWrap = true
	tm.CursorLeft(3)
	tm5SExpectCursor(t, tm, 2, 1, false)
}

// Zig: test "term51: cursorLeft: reverse wrap goes up to the end of a soft-wrapped line"
func TestTm5_term51_cursorLeft_reverse_wrap_goes_up_to_the_end_of_a_soft_wrapped_line(t *testing.T) {
	tm := tm5Init(t, 5, 4)
	defer tm.Deinit()
	tm.Modes.Set(ModesModeReverseWrap, true)

	zigTry(t, tm.PrintString([]uint8("abcdefgh")))
	tm5SExpectCursor(t, tm, 3, 1, false)
	// 3 to reach column 0, 1 for the wrap itself.
	tm.CursorLeft(4)
	tm5SExpectCursor(t, tm, 4, 0, false)

	// 3 to column 0, 1 for the wrap, 2 more along the previous line.
	tm.Screens.Active.CursorAbsolute(3, 1)
	tm.CursorLeft(6)
	tm5SExpectCursor(t, tm, 2, 0, false)
}

// Zig: test "term51: cursorLeft: reverse wrap stops at column zero when the line above is not wrapped"
func TestTm5_term51_cursorLeft_reverse_wrap_stops_at_column_zero_when_the_line_above_is_not_wrapped(t *testing.T) {
	tm := tm5Init(t, 5, 4)
	defer tm.Deinit()
	tm.Modes.Set(ModesModeReverseWrap, true)

	zigTry(t, tm.PrintString([]uint8("abc\ndef")))
	tm5SExpectCursor(t, tm, 3, 1, false)
	tm.CursorLeft(100)
	tm5SExpectCursor(t, tm, 0, 1, false)
}

// Zig: test "term51: cursorLeft: reverse wrap at the top-left goes nowhere"
func TestTm5_term51_cursorLeft_reverse_wrap_at_the_top_left_goes_nowhere(t *testing.T) {
	tm := tm5Init(t, 5, 4)
	defer tm.Deinit()
	tm.Modes.Set(ModesModeReverseWrap, true)

	tm.Screens.Active.CursorAbsolute(0, 0)
	tm.CursorLeft(3)
	tm5SExpectCursor(t, tm, 0, 0, false)
	tm.Screens.Active.CursorAbsolute(2, 0)
	tm.CursorLeft(10)
	tm5SExpectCursor(t, tm, 0, 0, false)
}

// Zig: test "term51: cursorLeft: reverse wrap respects left and right margins"
func TestTm5_term51_cursorLeft_reverse_wrap_respects_left_and_right_margins(t *testing.T) {
	tm := tm5Init(t, 10, 4)
	defer tm.Deinit()
	tm.Modes.Set(ModesModeReverseWrap, true)
	tm.ScrollingRegion.Left = 2
	tm.ScrollingRegion.Right = 7

	// Stop at the left margin when the line above is not wrapped.
	tm.Screens.Active.CursorAbsolute(4, 2)
	tm.CursorLeft(10)
	tm5SExpectCursor(t, tm, 2, 2, false)

	// With the line above soft wrapped we continue from the RIGHT margin.
	tm5SAt(t, tm, 0, 1).Row.wrap = true
	tm.Screens.Active.CursorAbsolute(3, 2)
	tm.CursorLeft(3)
	// 1 to the margin, 1 for the wrap, 1 more on the line above.
	tm5SExpectCursor(t, tm, 6, 1, false)

	// Cursor left of the left margin uses column zero as its edge.
	tm5SAt(t, tm, 0, 1).Row.wrap = false
	tm.Screens.Active.CursorAbsolute(1, 2)
	tm.CursorLeft(10)
	tm5SExpectCursor(t, tm, 0, 2, false)
}

// Zig: test "term51: cursorLeft: reverse wrap at the left margin above the top margin moves to the top-left"
func TestTm5_term51_cursorLeft_reverse_wrap_at_the_left_margin_above_the_top_margin_moves_to_the_top_left(t *testing.T) {
	// At the left margin with y <= top the cursor is moved to the top
	// margin row even if that means moving DOWN, regardless of the count.
	tm := tm5Init(t, 10, 6)
	defer tm.Deinit()
	tm.Modes.Set(ModesModeReverseWrap, true)
	tm.ScrollingRegion.Top = 2
	tm.ScrollingRegion.Bottom = 4

	tm.Screens.Active.CursorAbsolute(0, 1)
	tm.CursorLeft(1)
	tm5SExpectCursor(t, tm, 0, 2, false)

	// At the top margin row already.
	tm.Screens.Active.CursorAbsolute(0, 2)
	tm.CursorLeft(3)
	tm5SExpectCursor(t, tm, 0, 2, false)

	// With a left margin the target column is the left margin.
	tm.ScrollingRegion.Left = 3
	tm.ScrollingRegion.Right = 8
	tm.Screens.Active.CursorAbsolute(3, 0)
	tm.CursorLeft(1)
	tm5SExpectCursor(t, tm, 3, 2, false)
}

// Zig: test "term51: cursorLeft: reverse wrap with pending wrap in a one column terminal"
func TestTm5_term51_cursorLeft_reverse_wrap_with_pending_wrap_in_a_one_column_terminal(t *testing.T) {
	// Pending wrap consumes one unit of the count; with a single column the
	// cursor is always at the left margin.
	tm := tm5Init(t, 1, 3)
	defer tm.Deinit()
	tm.Modes.Set(ModesModeReverseWrap, true)

	tm.Screens.Active.CursorAbsolute(0, 2)
	tm.Screens.Active.Cursor.PendingWrap = true
	tm.CursorLeft(1)
	tm5SExpectCursor(t, tm, 0, 2, false)

	// Count 2: one for the pending wrap, one left but the line above is not
	// wrapped so the cursor stays.
	tm.Screens.Active.Cursor.PendingWrap = true
	tm.CursorLeft(2)
	tm5SExpectCursor(t, tm, 0, 2, false)

	// With a wrapped line above the second unit is the wrap itself.
	tm5SAt(t, tm, 0, 1).Row.wrap = true
	tm.Screens.Active.Cursor.PendingWrap = true
	tm.CursorLeft(2)
	tm5SExpectCursor(t, tm, 0, 1, false)

	// At the top-left it is the shortcut: y <= top.
	tm.Screens.Active.CursorAbsolute(0, 0)
	tm.Screens.Active.Cursor.PendingWrap = true
	tm.CursorLeft(1)
	tm5SExpectCursor(t, tm, 0, 0, false)
}

// Zig: test "term51: cursorLeft: reverse wrap does not wrap above the top margin when inside the region"
func TestTm5_term51_cursorLeft_reverse_wrap_does_not_wrap_above_the_top_margin_when_inside_the_region(t *testing.T) {
	tm := tm5Init(t, 10, 6)
	defer tm.Deinit()
	tm.Modes.Set(ModesModeReverseWrap, true)
	tm.ScrollingRegion.Top = 2
	tm.ScrollingRegion.Bottom = 4

	// Not at the left margin: moves to it and stops at the top margin row.
	tm.Screens.Active.CursorAbsolute(3, 2)
	tm.CursorLeft(100)
	tm5SExpectCursor(t, tm, 0, 2, false)
}

// Zig: test "term51: cursorLeft: extended reverse wrap crosses lines without a soft wrap"
func TestTm5_term51_cursorLeft_extended_reverse_wrap_crosses_lines_without_a_soft_wrap(t *testing.T) {
	tm := tm5Init(t, 5, 4)
	defer tm.Deinit()
	tm.Modes.Set(ModesModeReverseWrapExtended, true)

	tm.Screens.Active.CursorAbsolute(1, 2)
	// 1 to column 0, 1 for the wrap, 1 more on the previous line.
	tm.CursorLeft(3)
	tm5SExpectCursor(t, tm, 3, 1, false)
	// The line above is not wrapped (no wrap flag) yet we crossed it.
	expect(t, !tm5SAt(t, tm, 0, 1).Row.wrap, "!term51SAt(&t, 0, 1).row.wrap")
}

// Zig: test "term51: cursorLeft: extended reverse wrap from the top wraps to the bottom right"
func TestTm5_term51_cursorLeft_extended_reverse_wrap_from_the_top_wraps_to_the_bottom_right(t *testing.T) {
	tm := tm5Init(t, 5, 4)
	defer tm.Deinit()
	tm.Modes.Set(ModesModeReverseWrapExtended, true)

	tm.Screens.Active.CursorAbsolute(0, 0)
	tm.CursorLeft(1)
	tm5SExpectCursor(t, tm, 4, 3, false)

	tm.Screens.Active.CursorAbsolute(1, 0)
	tm.CursorLeft(3)
	// 1 to column 0, 1 to wrap, 1 more leftwards on the bottom row.
	tm5SExpectCursor(t, tm, 3, 3, false)

	// A large count walks up through the rows, wrapping from the top to the
	// bottom once: 2 to column 0, then each wrap costs 1 and each full row
	// 4 more: (4,0) at 24 left, (0,0) 4 later... then (4,3), (4,2), (4,1),
	// (4,0) again and the final 4 steps end at (0,0).
	tm.Screens.Active.CursorAbsolute(2, 1)
	tm.CursorLeft(27)
	tm5SExpectCursor(t, tm, 0, 0, false)
}

// Zig: test "term51: cursorLeft: extended reverse wrap uses the margins"
func TestTm5_term51_cursorLeft_extended_reverse_wrap_uses_the_margins(t *testing.T) {
	tm := tm5Init(t, 10, 6)
	defer tm.Deinit()
	tm.Modes.Set(ModesModeReverseWrapExtended, true)
	tm.ScrollingRegion.Top = 1
	tm.ScrollingRegion.Bottom = 3
	tm.ScrollingRegion.Left = 2
	tm.ScrollingRegion.Right = 6

	// From the top-left corner of the region to the bottom-right.
	tm.Screens.Active.CursorAbsolute(2, 1)
	tm.CursorLeft(1)
	tm5SExpectCursor(t, tm, 6, 3, false)

	// From the left margin of a middle row to the right margin above.
	tm.Screens.Active.CursorAbsolute(2, 3)
	tm.CursorLeft(1)
	tm5SExpectCursor(t, tm, 6, 2, false)
}

// Zig: test "term51: cursorLeft: extended reverse wrap above the region stops at the origin"
func TestTm5_term51_cursorLeft_extended_reverse_wrap_above_the_region_stops_at_the_origin(t *testing.T) {
	tm := tm5Init(t, 5, 6)
	defer tm.Deinit()
	tm.Modes.Set(ModesModeReverseWrapExtended, true)
	tm.ScrollingRegion.Top = 3
	tm.ScrollingRegion.Bottom = 4

	// Row 0 is above the region: the cursor walks left and stops at (0, 0).
	tm.Screens.Active.CursorAbsolute(3, 0)
	tm.CursorLeft(100)
	tm5SExpectCursor(t, tm, 0, 0, false)

	// From row 1, it wraps up to row 0 and keeps going.
	tm.Screens.Active.CursorAbsolute(1, 1)
	tm.CursorLeft(100)
	tm5SExpectCursor(t, tm, 0, 0, false)
}

// Zig: test "term51: cursorLeft: extended reverse wrap takes precedence and consumes pending wrap"
func TestTm5_term51_cursorLeft_extended_reverse_wrap_takes_precedence_and_consumes_pending_wrap(t *testing.T) {
	tm := tm5Init(t, 5, 4)
	defer tm.Deinit()
	tm.Modes.Set(ModesModeReverseWrap, true)
	tm.Modes.Set(ModesModeReverseWrapExtended, true)

	tm.Screens.Active.CursorAbsolute(4, 1)
	tm.Screens.Active.Cursor.PendingWrap = true
	tm.CursorLeft(1)
	tm5SExpectCursor(t, tm, 4, 1, false)

	tm.Screens.Active.Cursor.PendingWrap = true
	tm.CursorLeft(2)
	tm5SExpectCursor(t, tm, 3, 1, false)

	// No wrap flag needed on the line above.
	tm.Screens.Active.CursorAbsolute(0, 2)
	tm.CursorLeft(1)
	tm5SExpectCursor(t, tm, 4, 1, false)
}

// Zig: test "term51: cursorLeft: cursor right of the right margin moves left normally"
func TestTm5_term51_cursorLeft_cursor_right_of_the_right_margin_moves_left_normally(t *testing.T) {
	tm := tm5Init(t, 10, 3)
	defer tm.Deinit()
	tm.Modes.Set(ModesModeReverseWrapExtended, true)
	tm.ScrollingRegion.Left = 2
	tm.ScrollingRegion.Right = 6

	tm.Screens.Active.CursorAbsolute(9, 1)
	tm.CursorLeft(3)
	tm5SExpectCursor(t, tm, 6, 1, false)
	// Continuing over the left margin edge wraps to the right margin above.
	tm.CursorLeft(5)
	// 4 to the left margin, 1 to wrap.
	tm5SExpectCursor(t, tm, 6, 0, false)
}

// Zig: test "term51: saveCursor: stores position, pen state, modes and charset"
func TestTm5_term51_saveCursor_stores_position_pen_state_modes_and_charset(t *testing.T) {
	tm := tm5Init(t, 10, 5)
	defer tm.Deinit()

	expect(t, !tm.Screens.Active.SavedCursor.Valid, "t.screens.active.saved_cursor == null")

	tm.Screens.Active.CursorAbsolute(9, 3)
	tm.Screens.Active.Cursor.PendingWrap = true
	tm.Screens.Active.Cursor.Protected = true
	tm5Attr(t, tm, SGRAttribute{Tag: SGRAttributeTagBold})
	tm.Modes.Set(ModesModeOrigin, true)
	tm.ConfigureCharset(CharsetsSlotsG1, CharsetsCharsetDecSpecial)
	tm.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG1, false)
	tm.InvokeCharset(CharsetsActiveSlotGR, CharsetsSlotsG3, false)
	tm.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG2, true)
	tm.SaveCursor()

	saved := zigUnwrap(t, tm.Screens.Active.SavedCursor)
	expectEqual(t, SizeCellCountInt(9), saved.X)
	expectEqual(t, SizeCellCountInt(3), saved.Y)
	expect(t, saved.PendingWrap, "saved.pending_wrap")
	expect(t, saved.Protected, "saved.protected")
	expect(t, saved.Origin, "saved.origin")
	expect(t, saved.Style.Flags.Bold, "saved.style.flags.bold")
	expectEqual(t, CharsetsCharsetDecSpecial, saved.Charset.charsets.Get(CharsetsSlotsG1))
	expectEqual(t, CharsetsSlotsG1, saved.Charset.Gl)
	expectEqual(t, CharsetsSlotsG3, saved.Charset.Gr)
	tm5ExpectOptSome(t, CharsetsSlotsG2, saved.Charset.SingleShift)
}

// Zig: test "term51: saveCursor: overwrites a previous save"
func TestTm5_term51_saveCursor_overwrites_a_previous_save(t *testing.T) {
	tm := tm5Init(t, 10, 5)
	defer tm.Deinit()

	tm.Screens.Active.CursorAbsolute(1, 1)
	tm.SaveCursor()
	tm.Screens.Active.CursorAbsolute(5, 2)
	tm.SaveCursor()
	saved := zigUnwrap(t, tm.Screens.Active.SavedCursor)
	expectEqual(t, SizeCellCountInt(5), saved.X)
	expectEqual(t, SizeCellCountInt(2), saved.Y)
	expect(t, !saved.PendingWrap, "!saved.pending_wrap")
	expect(t, !saved.Protected, "!saved.protected")
	expect(t, !saved.Origin, "!saved.origin")
	expect(t, saved.Style.Default(), "saved.style.default()")
}

// Zig: test "term51: restoreCursor: restores everything that was saved"
func TestTm5_term51_restoreCursor_restores_everything_that_was_saved(t *testing.T) {
	tm := tm5Init(t, 10, 5)
	defer tm.Deinit()

	tm.Screens.Active.CursorAbsolute(9, 3)
	tm.Screens.Active.Cursor.PendingWrap = true
	tm.Screens.Active.Cursor.Protected = true
	tm5Attr(t, tm, SGRAttribute{Tag: SGRAttributeTagBold})
	tm.Modes.Set(ModesModeOrigin, true)
	tm.ConfigureCharset(CharsetsSlotsG1, CharsetsCharsetDecSpecial)
	tm.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG1, false)
	tm.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG2, true)
	tm.SaveCursor()

	// Change everything.
	tm.Screens.Active.CursorAbsolute(2, 0)
	tm.Screens.Active.Cursor.PendingWrap = false
	tm.Screens.Active.Cursor.Protected = false
	tm5Attr(t, tm, SGRAttribute{Tag: SGRAttributeTagUnset})
	tm5Attr(t, tm, SGRAttribute{Tag: SGRAttributeTagItalic})
	tm.Modes.Set(ModesModeOrigin, false)
	tm.ConfigureCharset(CharsetsSlotsG1, CharsetsCharsetBritish)
	tm.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG0, false)
	tm.Screens.Active.Charset.SingleShift = Optional[CharsetsSlots]{}

	tm.RestoreCursor()
	tm5SExpectCursor(t, tm, 9, 3, true)
	expect(t, tm.Screens.Active.Cursor.Protected, "t.screens.active.cursor.protected")
	expect(t, tm.Modes.Get(ModesModeOrigin), "t.modes.get(.origin)")
	expect(t, tm.Screens.Active.Cursor.Style.Flags.Bold, "t.screens.active.cursor.style.flags.bold")
	expect(t, !tm.Screens.Active.Cursor.Style.Flags.Italic, "!t.screens.active.cursor.style.flags.italic")
	expect(t, tm.Screens.Active.Cursor.StyleID != StyleDefaultID, "t.screens.active.cursor.style_id != style.default_id")
	expectEqual(t, CharsetsCharsetDecSpecial, tm.Screens.Active.Charset.charsets.Get(CharsetsSlotsG1))
	expectEqual(t, CharsetsSlotsG1, tm.Screens.Active.Charset.Gl)
	tm5ExpectOptSome(t, CharsetsSlotsG2, tm.Screens.Active.Charset.SingleShift)

	// The restored style is real: printing uses it.
	tm.Screens.Active.Cursor.PendingWrap = false
	tm.Screens.Active.CursorAbsolute(0, 0)
	zigTry(t, tm.Print('a'))
	lc := tm5SAt(t, tm, 0, 0)
	expect(t, lc.Node.data.styles.Get(lc.Cell.style_id).Flags.Bold, "lc.node.data.styles.get(lc.node.data.memory, lc.cell.style_id).flags.bold")

	// The save is not consumed: restoring again gives the same state.
	tm.Screens.Active.CursorAbsolute(0, 0)
	tm.RestoreCursor()
	tm5SExpectCursor(t, tm, 9, 3, true)
}

// Zig: test "term51: restoreCursor: without a previous save resets to defaults"
func TestTm5_term51_restoreCursor_without_a_previous_save_resets_to_defaults(t *testing.T) {
	tm := tm5Init(t, 10, 5)
	defer tm.Deinit()

	tm.Screens.Active.CursorAbsolute(5, 3)
	tm.Screens.Active.Cursor.PendingWrap = true
	tm.Screens.Active.Cursor.Protected = true
	tm5Attr(t, tm, SGRAttribute{Tag: SGRAttributeTagBold})
	tm.Modes.Set(ModesModeOrigin, true)
	tm.ConfigureCharset(CharsetsSlotsG0, CharsetsCharsetDecSpecial)
	tm.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG3, false)
	tm.InvokeCharset(CharsetsActiveSlotGR, CharsetsSlotsG1, false)
	tm.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG1, true)

	tm.RestoreCursor()
	tm5SExpectCursor(t, tm, 0, 0, false)
	expect(t, !tm.Screens.Active.Cursor.Protected, "!t.screens.active.cursor.protected")
	expect(t, !tm.Modes.Get(ModesModeOrigin), "!t.modes.get(.origin)")
	expect(t, tm.Screens.Active.Cursor.Style.Default(), "t.screens.active.cursor.style.default()")
	expectEqual(t, StyleDefaultID, tm.Screens.Active.Cursor.StyleID)
	expectEqual(t, CharsetsCharsetUtf8, tm.Screens.Active.Charset.charsets.Get(CharsetsSlotsG0))
	expectEqual(t, CharsetsSlotsG0, tm.Screens.Active.Charset.Gl)
	expectEqual(t, CharsetsSlotsG2, tm.Screens.Active.Charset.Gr)
	tm5ExpectOptNull(t, tm.Screens.Active.Charset.SingleShift)
	// Restoring without a save does not create a save.
	expect(t, !tm.Screens.Active.SavedCursor.Valid, "t.screens.active.saved_cursor == null")
}

// Zig: test "term51: restoreCursor: releases the style it replaces"
func TestTm5_term51_restoreCursor_releases_the_style_it_replaces(t *testing.T) {
	tm := tm5Init(t, 10, 5)
	defer tm.Deinit()

	tm5Attr(t, tm, SGRAttribute{Tag: SGRAttributeTagBold})
	tm.SaveCursor()
	tm5Attr(t, tm, SGRAttribute{Tag: SGRAttributeTagUnset})
	tm5Attr(t, tm, SGRAttribute{Tag: SGRAttributeTagItalic})
	page := &tm.Screens.Active.Cursor.PagePin.Node.data
	// Only the italic style is live (the cursor holds it).
	expectEqual(t, uint(1), page.styles.Count())
	expect(t, page.styles.Get(tm.Screens.Active.Cursor.StyleID).Flags.Italic, "page.styles.get(page.memory, t.screens.active.cursor.style_id).flags.italic")

	tm.RestoreCursor()
	// The italic style was released and bold took its place.
	expectEqual(t, uint(1), page.styles.Count())
	expect(t, tm.Screens.Active.Cursor.Style.Flags.Bold, "t.screens.active.cursor.style.flags.bold")
	expect(t, !tm.Screens.Active.Cursor.Style.Flags.Italic, "!t.screens.active.cursor.style.flags.italic")
	bold_id := tm.Screens.Active.Cursor.StyleID
	expect(t, page.styles.Get(bold_id).Flags.Bold, "page.styles.get(page.memory, bold_id).flags.bold")
	expectEqual(t, SizeCellCountInt(1), page.styles.RefCount(bold_id))

	// Restoring a saved default style while styled releases the style.
	tm.Screens.Active.SavedCursor.Value.Style = Style{}
	tm.RestoreCursor()
	expectEqual(t, StyleDefaultID, tm.Screens.Active.Cursor.StyleID)
	expectEqual(t, uint(0), page.styles.Count())
}

// Zig: test "term51: restoreCursor: clamps a saved position that no longer fits"
func TestTm5_term51_restoreCursor_clamps_a_saved_position_that_no_longer_fits(t *testing.T) {
	tm := tm5Init(t, 10, 5)
	defer tm.Deinit()

	tm.SaveCursor()
	zigUnwrap(t, tm.Screens.Active.SavedCursor)
	saved := &tm.Screens.Active.SavedCursor.Value

	// Exactly the last cell: unchanged.
	saved.X = 9
	saved.Y = 4
	tm.RestoreCursor()
	tm5SExpectCursor(t, tm, 9, 4, false)

	// One beyond each edge, and far beyond.
	saved.X = 10
	saved.Y = 5
	tm.RestoreCursor()
	tm5SExpectCursor(t, tm, 9, 4, false)
	saved.X = 1000
	saved.Y = 0
	tm.RestoreCursor()
	tm5SExpectCursor(t, tm, 9, 0, false)
	saved.X = 0
	saved.Y = 1000
	tm.RestoreCursor()
	tm5SExpectCursor(t, tm, 0, 4, false)
}

// Zig: test "term51: restoreCursor: does not change the protected mode or hyperlink state"
func TestTm5_term51_restoreCursor_does_not_change_the_protected_mode_or_hyperlink_state(t *testing.T) {
	tm := tm5Init(t, 10, 5)
	defer tm.Deinit()

	tm.SetProtectedMode(ANSIProtectedModeDec)
	zigTry(t, tm.Screens.Active.StartHyperlink([]uint8("http://example.com"), Optional[[]uint8]{}))
	tm.SaveCursor()
	tm.SetProtectedMode(ANSIProtectedModeIso)
	tm.Screens.Active.EndHyperlink()
	tm.RestoreCursor()
	// The pen protection flag comes from the save (set by DEC mode), the
	// screen's most recent protected mode is untouched.
	expect(t, tm.Screens.Active.Cursor.Protected, "t.screens.active.cursor.protected")
	expectEqual(t, ANSIProtectedModeIso, tm.Screens.Active.ProtectedMode)
	expectEqual(t, HyperlinkID(0), tm.Screens.Active.Cursor.HyperlinkID)
}

// Zig: test "term51: restoreCursor: origin mode from the save affects cursor positioning"
func TestTm5_term51_restoreCursor_origin_mode_from_the_save_affects_cursor_positioning(t *testing.T) {
	tm := tm5Init(t, 10, 6)
	defer tm.Deinit()

	tm.ScrollingRegion.Top = 2
	tm.ScrollingRegion.Bottom = 4
	tm.Modes.Set(ModesModeOrigin, true)
	tm.SaveCursor()
	tm.Modes.Set(ModesModeOrigin, false)
	tm.RestoreCursor()
	tm.SetCursorPos(1, 1)
	// Relative to the top margin again.
	tm5SExpectCursor(t, tm, 0, 2, false)
}

// Zig: test "term51: saveCursor: each screen keeps its own save"
func TestTm5_term51_saveCursor_each_screen_keeps_its_own_save(t *testing.T) {
	tm := tm5Init(t, 10, 5)
	defer tm.Deinit()

	tm.Screens.Active.CursorAbsolute(4, 2)
	tm5Attr(t, tm, SGRAttribute{Tag: SGRAttributeTagBold})
	tm.SaveCursor()

	tm5SwitchScreen(t, tm, ScreenSetKeyAlternate)
	// The alternate screen has no save: restore gives defaults.
	expect(t, !tm.Screens.Active.SavedCursor.Valid, "t.screens.active.saved_cursor == null")
	tm.Screens.Active.CursorAbsolute(7, 1)
	tm.RestoreCursor()
	tm5SExpectCursor(t, tm, 0, 0, false)
	expect(t, tm.Screens.Active.Cursor.Style.Default(), "t.screens.active.cursor.style.default()")

	// Save on the alternate screen, which does not disturb the primary.
	tm.Screens.Active.CursorAbsolute(2, 2)
	tm.SaveCursor()
	tm5SwitchScreen(t, tm, ScreenSetKeyPrimary)
	primary_saved := zigUnwrap(t, tm.Screens.Active.SavedCursor)
	expectEqual(t, SizeCellCountInt(4), primary_saved.X)
	expectEqual(t, SizeCellCountInt(2), primary_saved.Y)
	expect(t, primary_saved.Style.Flags.Bold, "primary_saved.style.flags.bold")
	tm.Screens.Active.CursorAbsolute(0, 0)
	tm.RestoreCursor()
	tm5SExpectCursor(t, tm, 4, 2, false)
	expect(t, tm.Screens.Active.Cursor.Style.Flags.Bold, "t.screens.active.cursor.style.flags.bold")

	tm5SwitchScreen(t, tm, ScreenSetKeyAlternate)
	tm.RestoreCursor()
	tm5SExpectCursor(t, tm, 2, 2, false)
}

// Zig: test "term51: print: clustering a width-contributing character after an empty cell prints normally"
func TestTm5_term51_print_clustering_a_width_contributing_character_after_an_empty_cell_prints_normally(t *testing.T) {
	tm := tm5Init(t, 8, 2)
	defer tm.Deinit()
	tm.Modes.Set(ModesModeGraphemeCluster, true)

	tm.SetCursorPos(1, 4)
	zigTry(t, tm.Print(0x100))
	zigTry(t, tm.Print(0x1F600))
	tm5SExpectCell(t, tm, 3, 0, 0x100, PageCellWideNarrow)
	tm5SExpectCell(t, tm, 4, 0, 0x1F600, PageCellWideWide)
	tm5SExpectCell(t, tm, 5, 0, 0, PageCellWideSpacerTail)
	tm5SExpectCursor(t, tm, 6, 0, false)
}

// Zig: test "term51: print: clustering VS16 with grapheme data at a right margin moves the data"
func TestTm5_term51_print_clustering_VS16_with_grapheme_data_at_a_right_margin_moves_the_data(t *testing.T) {
	tm := tm5Init(t, 10, 3)
	defer tm.Deinit()
	tm.Modes.Set(ModesModeGraphemeCluster, true)
	tm.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	tm.SetLeftAndRightMargin(3, 5)

	tm.SetCursorPos(1, 5)
	zigTry(t, tm.Print('#'))
	zigTry(t, tm.Print(0x301))
	tm5SExpectGrapheme(t, tm, 4, 0, []uint32{0x301})
	zigTry(t, tm.Print(0xFE0F))

	tm5SExpectCell(t, tm, 4, 0, 0, PageCellWideNarrow)
	tm5SExpectNoGrapheme(t, tm, 4, 0)
	expect(t, !tm5SAt(t, tm, 0, 0).Row.grapheme, "!term51SAt(&t, 0, 0).row.grapheme")
	expect(t, !tm5SAt(t, tm, 0, 0).Row.wrap, "!term51SAt(&t, 0, 0).row.wrap")
	tm5SExpectCell(t, tm, 2, 1, '#', PageCellWideWide)
	tm5SExpectGrapheme(t, tm, 2, 1, []uint32{0x301, 0xFE0F})
	tm5SExpectCell(t, tm, 3, 1, 0, PageCellWideSpacerTail)
	tm5SExpectCursor(t, tm, 4, 1, false)
}

// Zig: test "term51: setProtectedMode: initial state is unprotected with mode off"
func TestTm5_term51_setProtectedMode_initial_state_is_unprotected_with_mode_off(t *testing.T) {
	tm := tm5Init(t, 3, 3)
	defer tm.Deinit()

	expect(t, !tm.Screens.Active.Cursor.Protected, "!t.screens.active.cursor.protected")
	expectEqual(t, ANSIProtectedModeOff, tm.Screens.Active.ProtectedMode)
}

// Zig: test "term51: setProtectedMode: every transition between the three modes"
func TestTm5_term51_setProtectedMode_every_transition_between_the_three_modes(t *testing.T) {
	tm := tm5Init(t, 3, 3)
	defer tm.Deinit()
	s := tm.Screens.Active

	// off from the initial state changes nothing
	tm.SetProtectedMode(ANSIProtectedModeOff)
	expect(t, !s.Cursor.Protected, "!s.cursor.protected")
	expectEqual(t, ANSIProtectedModeOff, s.ProtectedMode)

	// iso
	tm.SetProtectedMode(ANSIProtectedModeIso)
	expect(t, s.Cursor.Protected, "s.cursor.protected")
	expectEqual(t, ANSIProtectedModeIso, s.ProtectedMode)

	// iso -> iso is idempotent
	tm.SetProtectedMode(ANSIProtectedModeIso)
	expect(t, s.Cursor.Protected, "s.cursor.protected")
	expectEqual(t, ANSIProtectedModeIso, s.ProtectedMode)

	// off clears the cursor flag but keeps the remembered mode
	tm.SetProtectedMode(ANSIProtectedModeOff)
	expect(t, !s.Cursor.Protected, "!s.cursor.protected")
	expectEqual(t, ANSIProtectedModeIso, s.ProtectedMode)

	// dec replaces the remembered mode
	tm.SetProtectedMode(ANSIProtectedModeDec)
	expect(t, s.Cursor.Protected, "s.cursor.protected")
	expectEqual(t, ANSIProtectedModeDec, s.ProtectedMode)

	// iso directly from dec (cursor stays protected)
	tm.SetProtectedMode(ANSIProtectedModeIso)
	expect(t, s.Cursor.Protected, "s.cursor.protected")
	expectEqual(t, ANSIProtectedModeIso, s.ProtectedMode)

	// dec, then off: the mode stays dec
	tm.SetProtectedMode(ANSIProtectedModeDec)
	tm.SetProtectedMode(ANSIProtectedModeOff)
	expect(t, !s.Cursor.Protected, "!s.cursor.protected")
	expectEqual(t, ANSIProtectedModeDec, s.ProtectedMode)

	// off twice is still off
	tm.SetProtectedMode(ANSIProtectedModeOff)
	expect(t, !s.Cursor.Protected, "!s.cursor.protected")
	expectEqual(t, ANSIProtectedModeDec, s.ProtectedMode)
}

// Zig: test "term51: setProtectedMode: printed cells carry the protected flag of the moment"
func TestTm5_term51_setProtectedMode_printed_cells_carry_the_protected_flag_of_the_moment(t *testing.T) {
	tm := tm5Init(t, 6, 2)
	defer tm.Deinit()

	zigTry(t, tm.Print('a'))
	tm.SetProtectedMode(ANSIProtectedModeIso)
	zigTry(t, tm.Print('b'))
	tm.SetProtectedMode(ANSIProtectedModeOff)
	zigTry(t, tm.Print('c'))
	tm.SetProtectedMode(ANSIProtectedModeDec)
	zigTry(t, tm.Print('d'))

	expect(t, !tm5TCellAt(t, tm, 0, 0).protected, "!term51TCellAt(&t, 0, 0).protected")
	expect(t, tm5TCellAt(t, tm, 1, 0).protected, "term51TCellAt(&t, 1, 0).protected")
	expect(t, !tm5TCellAt(t, tm, 2, 0).protected, "!term51TCellAt(&t, 2, 0).protected")
	expect(t, tm5TCellAt(t, tm, 3, 0).protected, "term51TCellAt(&t, 3, 0).protected")
}

// Zig: test "term51: setProtectedMode: applies to the active screen only"
func TestTm5_term51_setProtectedMode_applies_to_the_active_screen_only(t *testing.T) {
	tm := tm5Init(t, 3, 3)
	defer tm.Deinit()

	tm.SetProtectedMode(ANSIProtectedModeDec)
	tm5SwitchScreen(t, tm, ScreenSetKeyAlternate)
	expect(t, !tm.Screens.Active.Cursor.Protected, "!t.screens.active.cursor.protected")
	expectEqual(t, ANSIProtectedModeOff, tm.Screens.Active.ProtectedMode)

	tm.SetProtectedMode(ANSIProtectedModeIso)
	expect(t, tm.Screens.Active.Cursor.Protected, "t.screens.active.cursor.protected")
	expectEqual(t, ANSIProtectedModeIso, tm.Screens.Active.ProtectedMode)

	// The primary screen kept its own state.
	primary := tm.Screens.Get(ScreenSetKeyPrimary)
	expect(t, primary != nil, "t.screens.get(.primary) != null")
	expect(t, primary.Cursor.Protected, "primary.cursor.protected")
	expectEqual(t, ANSIProtectedModeDec, primary.ProtectedMode)
}

// Zig: test "term51: tabSet: sets the stop at the cursor column and is idempotent"
func TestTm5_term51_tabSet_sets_the_stop_at_the_cursor_column_and_is_idempotent(t *testing.T) {
	tm := tm5Init(t, 20, 3)
	defer tm.Deinit()

	expectEqualStrings(t, "........^.......^...", tm5TTabString(tm))

	tm.SetCursorPos(1, 4)
	tm.TabSet()
	expectEqualStrings(t, "...^....^.......^...", tm5TTabString(tm))

	// Setting again does not toggle.
	tm.TabSet()
	expectEqualStrings(t, "...^....^.......^...", tm5TTabString(tm))

	// Setting on an existing default stop keeps it set.
	tm.SetCursorPos(1, 9)
	tm.TabSet()
	expectEqualStrings(t, "...^....^.......^...", tm5TTabString(tm))

	// Column 0 and the last column are valid.
	tm.SetCursorPos(1, 1)
	tm.TabSet()
	tm.SetCursorPos(1, 20)
	tm.TabSet()
	expectEqualStrings(t, "^..^....^.......^..^", tm5TTabString(tm))

	// The cursor does not move.
	tm5TExpectCursor(t, tm, 19, 0)
}

// Zig: test "term51: tabSet: stop in the dynamically allocated range"
func TestTm5_term51_tabSet_stop_in_the_dynamically_allocated_range(t *testing.T) {
	tm := tm5Init(t, 600, 2)
	defer tm.Deinit()

	expect(t, !tm.Tabstops.Get(515), "!t.tabstops.get(515)")
	tm.SetCursorPos(1, 516)
	tm.TabSet()
	expect(t, tm.Tabstops.Get(515), "t.tabstops.get(515)")
	expect(t, !tm.Tabstops.Get(514), "!t.tabstops.get(514)")
	expect(t, !tm.Tabstops.Get(516), "!t.tabstops.get(516)")

	// And it is honoured by horizontalTab.
	tm.SetCursorPos(1, 513)
	tm.HorizontalTab()
	tm5TExpectCursor(t, tm, 515, 0)
}

// Zig: test "term51: tabSet: independent of margins and origin mode"
func TestTm5_term51_tabSet_independent_of_margins_and_origin_mode(t *testing.T) {
	tm := tm5Init(t, 20, 5)
	defer tm.Deinit()

	tm.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	tm.SetLeftAndRightMargin(5, 10)
	tm.SetCursorPos(3, 2)
	tm.TabSet()
	expect(t, tm.Tabstops.Get(1), "t.tabstops.get(1)")
	tm.Modes.Set(ModesModeOrigin, true)
	tm.SetCursorPos(1, 1)
	// origin mode: column 1 is the left margin, absolute column 4
	tm5TExpectCursor(t, tm, 4, 0)
	tm.TabSet()
	expect(t, tm.Tabstops.Get(4), "t.tabstops.get(4)")
}

// Zig: test "term51: tabReset: restores the default stops every 8 columns"
func TestTm5_term51_tabReset_restores_the_default_stops_every_8_columns(t *testing.T) {
	tm := tm5Init(t, 30, 3)
	defer tm.Deinit()
	defaults := "........^.......^.......^....."

	expectEqualStrings(t, defaults, tm5TTabString(tm))

	tm.TabClear(CSITabClearAll)
	tm.SetCursorPos(2, 4)
	tm.TabSet()
	tm.SetCursorPos(2, 30)
	tm.TabSet()
	expectEqualStrings(t, "...^.........................^", tm5TTabString(tm))

	tm.TabReset()
	expectEqualStrings(t, defaults, tm5TTabString(tm))

	// The cursor did not move.
	tm5TExpectCursor(t, tm, 29, 1)

	// Resetting twice changes nothing.
	tm.TabReset()
	expectEqualStrings(t, defaults, tm5TTabString(tm))
}

// Zig: test "term51: tabReset: the last column never gets a stop and tiny widths get none"
func TestTm5_term51_tabReset_the_last_column_never_gets_a_stop_and_tiny_widths_get_none(t *testing.T) {
	// 9 columns: first candidate (8) is the last column, so no stops.
	func() {
		tm := tm5Init(t, 9, 2)
		defer tm.Deinit()
		expectEqualStrings(t, ".........", tm5TTabString(tm))
		tm.SetCursorPos(1, 4)
		tm.TabSet()
		tm.TabReset()
		expectEqualStrings(t, ".........", tm5TTabString(tm))
	}()
	// 10 columns: stop at 8.
	func() {
		tm := tm5Init(t, 10, 2)
		defer tm.Deinit()
		tm.TabClear(CSITabClearAll)
		tm.TabReset()
		expectEqualStrings(t, "........^.", tm5TTabString(tm))
	}()
	// 17 columns: stops at 8 and 16 is the last column so not set.
	func() {
		tm := tm5Init(t, 17, 2)
		defer tm.Deinit()
		tm.TabClear(CSITabClearAll)
		tm.TabReset()
		expectEqualStrings(t, "........^........", tm5TTabString(tm))
	}()
	// 18 columns: 16 is set.
	func() {
		tm := tm5Init(t, 18, 2)
		defer tm.Deinit()
		tm.TabClear(CSITabClearAll)
		tm.TabReset()
		expectEqualStrings(t, "........^.......^.", tm5TTabString(tm))
	}()
	// 1 column.
	func() {
		tm := tm5Init(t, 1, 1)
		defer tm.Deinit()
		tm.TabReset()
		expectEqualStrings(t, ".", tm5TTabString(tm))
	}()
}

// Zig: test "term51: tabReset: covers the dynamically allocated range"
func TestTm5_term51_tabReset_covers_the_dynamically_allocated_range(t *testing.T) {
	tm := tm5Init(t, 600, 2)
	defer tm.Deinit()

	tm.TabClear(CSITabClearAll)
	for x := uint(0); x < 600; x++ {
		expect(t, !tm.Tabstops.Get(x), "!t.tabstops.get(x)")
	}
	tm.SetCursorPos(1, 601)
	tm.TabSet()
	tm.SetCursorPos(1, 531)
	tm.TabSet()
	expect(t, tm.Tabstops.Get(599), "t.tabstops.get(599)")
	expect(t, tm.Tabstops.Get(530), "t.tabstops.get(530)")

	tm.TabReset()
	for x := uint(0); x < 600; x++ {
		expected := x%8 == 0 && x != 0 && x < 599
		expectEqual(t, expected, tm.Tabstops.Get(x))
	}
	// 592 is the last multiple of 8 below 599.
	expect(t, tm.Tabstops.Get(592), "t.tabstops.get(592)")
	expect(t, !tm.Tabstops.Get(600-1), "!t.tabstops.get(600 - 1)")
}

// Zig: test "term51: tabClear: current clears a set stop and leaves the others"
func TestTm5_term51_tabClear_current_clears_a_set_stop_and_leaves_the_others(t *testing.T) {
	tm := tm5Init(t, 30, 3)
	defer tm.Deinit()

	tm.SetCursorPos(2, 9)
	tm.TabClear(CSITabClearCurrent)
	expectEqualStrings(t, "................^.......^.....", tm5TTabString(tm))
	tm5TExpectCursor(t, tm, 8, 1)

	tm.SetCursorPos(2, 17)
	tm.TabClear(CSITabClearCurrent)
	expectEqualStrings(t, "........................^.....", tm5TTabString(tm))
}

// Zig: test "term51: tabClear: current on a column without a stop sets one (xor toggle)"
func TestTm5_term51_tabClear_current_on_a_column_without_a_stop_sets_one_xor_toggle(t *testing.T) {
	// Upstream pins: Tabstops.unset is an xor, so clearing a column that has
	// no stop turns one on.
	tm := tm5Init(t, 30, 3)
	defer tm.Deinit()

	expect(t, !tm.Tabstops.Get(3), "!t.tabstops.get(3)")
	tm.SetCursorPos(1, 4)
	tm.TabClear(CSITabClearCurrent)
	expect(t, tm.Tabstops.Get(3), "t.tabstops.get(3)")
	tm.TabClear(CSITabClearCurrent)
	expect(t, !tm.Tabstops.Get(3), "!t.tabstops.get(3)")

	// Same at column 0 and at the last column.
	tm.TabClear(CSITabClearCurrent)
	tm.SetCursorPos(1, 1)
	tm.TabClear(CSITabClearCurrent)
	expect(t, tm.Tabstops.Get(0), "t.tabstops.get(0)")
	tm.SetCursorPos(1, 30)
	tm.TabClear(CSITabClearCurrent)
	expect(t, tm.Tabstops.Get(29), "t.tabstops.get(29)")
}

// Zig: test "term51: tabClear: current in the dynamically allocated range"
func TestTm5_term51_tabClear_current_in_the_dynamically_allocated_range(t *testing.T) {
	tm := tm5Init(t, 600, 2)
	defer tm.Deinit()

	// 520 = 65 * 8 is a default stop in the dynamic range.
	expect(t, tm.Tabstops.Get(520), "t.tabstops.get(520)")
	tm.SetCursorPos(1, 521)
	tm.TabClear(CSITabClearCurrent)
	expect(t, !tm.Tabstops.Get(520), "!t.tabstops.get(520)")
	expect(t, tm.Tabstops.Get(512), "t.tabstops.get(512)")
	expect(t, tm.Tabstops.Get(528), "t.tabstops.get(528)")
}

// Zig: test "term51: tabClear: all clears every stop including the dynamic range"
func TestTm5_term51_tabClear_all_clears_every_stop_including_the_dynamic_range(t *testing.T) {
	tm := tm5Init(t, 600, 2)
	defer tm.Deinit()

	tm.SetCursorPos(2, 300)
	tm.TabClear(CSITabClearAll)
	for x := uint(0); x < 600; x++ {
		expect(t, !tm.Tabstops.Get(x), "!t.tabstops.get(x)")
	}
	tm5TExpectCursor(t, tm, 299, 1)

	// Tab goes straight to the right margin.
	tm.SetCursorPos(1, 1)
	tm.HorizontalTab()
	tm5TExpectCursor(t, tm, 599, 0)

	// Clearing again changes nothing.
	tm.TabClear(CSITabClearAll)
	for x := uint(0); x < 600; x++ {
		expect(t, !tm.Tabstops.Get(x), "!t.tabstops.get(x)")
	}
}

// Zig: test "term51: tabClear: unknown values change nothing"
func TestTm5_term51_tabClear_unknown_values_change_nothing(t *testing.T) {
	tm := tm5Init(t, 30, 3)
	defer tm.Deinit()
	defaults := "........^.......^.......^....."

	tm.SetCursorPos(1, 9)
	for _, v := range []uint8{1, 2, 4, 5, 127, 255} {
		tm.TabClear(CSITabClear(v))
		expectEqualStrings(t, defaults, tm5TTabString(tm))
	}
	tm5TExpectCursor(t, tm, 8, 0)
}

// Zig: test "term51: horizontalTab: from every column with default stops"
func TestTm5_term51_horizontalTab_from_every_column_with_default_stops(t *testing.T) {
	tm := tm5Init(t, 80, 2)
	defer tm.Deinit()

	for x := uint(0); x < 80; x++ {
		tm.Screens.Active.CursorAbsolute(SizeCellCountInt(x), 0)
		tm.HorizontalTab()
		// stops at 8, 16, ..., 72; past 72 the right edge (79) is the target
		var expected uint
		if x >= 72 {
			expected = 79
		} else {
			expected = (x/8 + 1) * 8
		}
		tm5TExpectCursor(t, tm, expected, 0)
	}
}

// Zig: test "term51: horizontalTab: right margin limits the move"
func TestTm5_term51_horizontalTab_right_margin_limits_the_move(t *testing.T) {
	tm := tm5Init(t, 80, 2)
	defer tm.Deinit()
	tm.ScrollingRegion.Right = 20

	for x := uint(0); x < 80; x++ {
		tm.Screens.Active.CursorAbsolute(SizeCellCountInt(x), 0)
		tm.HorizontalTab()
		var next_stop uint
		if x >= 72 {
			next_stop = 79
		} else {
			next_stop = (x/8 + 1) * 8
		}
		var expected uint
		if x >= 20 {
			expected = x
		} else {
			expected = min(next_stop, 20)
		}
		tm5TExpectCursor(t, tm, expected, 0)
	}
}

// Zig: test "term51: horizontalTab: left margin is ignored"
func TestTm5_term51_horizontalTab_left_margin_is_ignored(t *testing.T) {
	tm := tm5Init(t, 80, 2)
	defer tm.Deinit()
	tm.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	tm.SetLeftAndRightMargin(21, 60)
	tm5TExpectRegion(t, tm, 0, 1, 20, 59)

	tm.Screens.Active.CursorAbsolute(2, 0)
	tm.HorizontalTab()
	tm5TExpectCursor(t, tm, 8, 0)
	tm.HorizontalTab()
	tm5TExpectCursor(t, tm, 16, 0)
	tm.HorizontalTab()
	tm5TExpectCursor(t, tm, 24, 0)

	// Past the right margin nothing moves.
	tm.Screens.Active.CursorAbsolute(60, 0)
	tm.HorizontalTab()
	tm5TExpectCursor(t, tm, 60, 0)
	tm.Screens.Active.CursorAbsolute(59, 0)
	tm.HorizontalTab()
	tm5TExpectCursor(t, tm, 59, 0)
}

// Zig: test "term51: horizontalTab: custom stops, a stop under the cursor does not count"
func TestTm5_term51_horizontalTab_custom_stops_a_stop_under_the_cursor_does_not_count(t *testing.T) {
	tm := tm5Init(t, 30, 2)
	defer tm.Deinit()

	tm.TabClear(CSITabClearAll)
	for _, x := range []uint{3, 10, 11} {
		tm.SetCursorPos(1, x+1)
		tm.TabSet()
	}
	tm.SetCursorPos(1, 1)

	expected := []uint{3, 10, 11, 29, 29}
	for _, e := range expected {
		tm.HorizontalTab()
		tm5TExpectCursor(t, tm, e, 0)
	}
}

// Zig: test "term51: horizontalTab: does not erase or write cells"
func TestTm5_term51_horizontalTab_does_not_erase_or_write_cells(t *testing.T) {
	tm := tm5Init(t, 12, 2)
	defer tm.Deinit()

	zigTry(t, tm.PrintString([]uint8("ABCDEFGHIJKL")))
	tm.SetCursorPos(1, 2)
	tm.HorizontalTab()
	tm5TExpectCursor(t, tm, 8, 0)
	tm5TExpectGrid(t, tm, []string{"ABCDEFGHIJKL", "............"})
}

// Zig: test "term51: horizontalTab: does not reset pending wrap"
func TestTm5_term51_horizontalTab_does_not_reset_pending_wrap(t *testing.T) {
	tm := tm5Init(t, 20, 2)
	defer tm.Deinit()

	tm.Screens.Active.Cursor.PendingWrap = true
	tm.HorizontalTab()
	tm5TExpectCursor(t, tm, 8, 0)
	expect(t, tm.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")

	// At the right edge nothing moves and pending wrap is kept.
	zigTry(t, tm.PrintString([]uint8("12345678901")))
	tm.Screens.Active.CursorAbsolute(19, 0)
	tm.Screens.Active.Cursor.PendingWrap = true
	tm.HorizontalTab()
	tm5TExpectCursor(t, tm, 19, 0)
	expect(t, tm.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
}

// Zig: test "term51: horizontalTab: narrow screens"
func TestTm5_term51_horizontalTab_narrow_screens(t *testing.T) {
	// 1 column: nowhere to go.
	func() {
		tm := tm5Init(t, 1, 1)
		defer tm.Deinit()
		tm.HorizontalTab()
		tm5TExpectCursor(t, tm, 0, 0)
	}()
	// 2 columns: no stops, goes to the last column.
	func() {
		tm := tm5Init(t, 2, 1)
		defer tm.Deinit()
		tm.HorizontalTab()
		tm5TExpectCursor(t, tm, 1, 0)
		tm.HorizontalTab()
		tm5TExpectCursor(t, tm, 1, 0)
	}()
	// 9 columns: stops would be at 8 which is the last column anyway.
	func() {
		tm := tm5Init(t, 9, 1)
		defer tm.Deinit()
		tm.HorizontalTab()
		tm5TExpectCursor(t, tm, 8, 0)
	}()
	// 10 columns: stop at 8, then the edge at 9.
	func() {
		tm := tm5Init(t, 10, 1)
		defer tm.Deinit()
		tm.HorizontalTab()
		tm5TExpectCursor(t, tm, 8, 0)
		tm.HorizontalTab()
		tm5TExpectCursor(t, tm, 9, 0)
	}()
}

// Zig: test "term51: horizontalTab: wide screen walks every stop across the dynamic range"
func TestTm5_term51_horizontalTab_wide_screen_walks_every_stop_across_the_dynamic_range(t *testing.T) {
	tm := tm5Init(t, 600, 1)
	defer tm.Deinit()

	expected := uint(8)
	for expected < 599 {
		tm.HorizontalTab()
		tm5TExpectCursor(t, tm, expected, 0)
		expected += 8
	}
	// 592 was the last stop; next goes to the right edge.
	tm.HorizontalTab()
	tm5TExpectCursor(t, tm, 599, 0)
}

// Zig: test "term51: horizontalTabBack: from every column with default stops"
func TestTm5_term51_horizontalTabBack_from_every_column_with_default_stops(t *testing.T) {
	tm := tm5Init(t, 80, 2)
	defer tm.Deinit()

	for x := uint(0); x < 80; x++ {
		tm.Screens.Active.CursorAbsolute(SizeCellCountInt(x), 0)
		tm.HorizontalTabBack()
		var expected uint
		if x == 0 {
			expected = 0
		} else {
			expected = ((x - 1) / 8) * 8
		}
		tm5TExpectCursor(t, tm, expected, 0)
	}
}

// Zig: test "term51: horizontalTabBack: origin mode stops at the left margin"
func TestTm5_term51_horizontalTabBack_origin_mode_stops_at_the_left_margin(t *testing.T) {
	tm := tm5Init(t, 80, 2)
	defer tm.Deinit()
	tm.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	tm.SetLeftAndRightMargin(11, 60)
	tm.Modes.Set(ModesModeOrigin, true)
	tm5TExpectRegion(t, tm, 0, 1, 10, 59)

	for x := uint(0); x < 80; x++ {
		tm.Screens.Active.CursorAbsolute(SizeCellCountInt(x), 0)
		tm.HorizontalTabBack()
		var prev_stop uint
		if x == 0 {
			prev_stop = 0
		} else {
			prev_stop = ((x - 1) / 8) * 8
		}
		var expected uint
		if x <= 10 {
			expected = x
		} else {
			expected = max(prev_stop, 10)
		}
		tm5TExpectCursor(t, tm, expected, 0)
	}
}

// Zig: test "term51: horizontalTabBack: without origin mode the left margin is ignored"
func TestTm5_term51_horizontalTabBack_without_origin_mode_the_left_margin_is_ignored(t *testing.T) {
	tm := tm5Init(t, 80, 2)
	defer tm.Deinit()
	tm.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	tm.SetLeftAndRightMargin(11, 60)

	tm.Screens.Active.CursorAbsolute(12, 0)
	tm.HorizontalTabBack()
	tm5TExpectCursor(t, tm, 8, 0)
	tm.HorizontalTabBack()
	tm5TExpectCursor(t, tm, 0, 0)
	tm.HorizontalTabBack()
	tm5TExpectCursor(t, tm, 0, 0)

	// A cursor left of the margin moves normally too.
	tm.Screens.Active.CursorAbsolute(5, 0)
	tm.HorizontalTabBack()
	tm5TExpectCursor(t, tm, 0, 0)
}

// Zig: test "term51: horizontalTabBack: stop at the left limit itself and at column 0"
func TestTm5_term51_horizontalTabBack_stop_at_the_left_limit_itself_and_at_column_0(t *testing.T) {
	tm := tm5Init(t, 30, 2)
	defer tm.Deinit()

	// A stop at column 0 is honoured (loop returns anyway).
	tm.SetCursorPos(1, 1)
	tm.TabSet()
	tm.Screens.Active.CursorAbsolute(5, 0)
	tm.HorizontalTabBack()
	tm5TExpectCursor(t, tm, 0, 0)

	// Left margin exactly on a stop: origin mode, from beyond it.
	tm.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	tm.SetLeftAndRightMargin(9, 30)
	tm.Modes.Set(ModesModeOrigin, true)
	tm.Screens.Active.CursorAbsolute(20, 0)
	tm.HorizontalTabBack()
	tm5TExpectCursor(t, tm, 16, 0)
	tm.HorizontalTabBack()
	tm5TExpectCursor(t, tm, 8, 0)
	tm.HorizontalTabBack()
	tm5TExpectCursor(t, tm, 8, 0)
}

// Zig: test "term51: horizontalTabBack: custom and cleared stops"
func TestTm5_term51_horizontalTabBack_custom_and_cleared_stops(t *testing.T) {
	tm := tm5Init(t, 30, 2)
	defer tm.Deinit()

	tm.TabClear(CSITabClearAll)
	tm.SetCursorPos(1, 6)
	tm.TabSet()
	tm.SetCursorPos(1, 21)
	tm.TabSet()

	tm.Screens.Active.CursorAbsolute(29, 0)
	tm.HorizontalTabBack()
	tm5TExpectCursor(t, tm, 20, 0)
	tm.HorizontalTabBack()
	tm5TExpectCursor(t, tm, 5, 0)
	tm.HorizontalTabBack()
	tm5TExpectCursor(t, tm, 0, 0)

	// The stop under the cursor does not count.
	tm.Screens.Active.CursorAbsolute(5, 0)
	tm.HorizontalTabBack()
	tm5TExpectCursor(t, tm, 0, 0)
}

// Zig: test "term51: horizontalTabBack: does not reset pending wrap and does not touch cells"
func TestTm5_term51_horizontalTabBack_does_not_reset_pending_wrap_and_does_not_touch_cells(t *testing.T) {
	tm := tm5Init(t, 20, 2)
	defer tm.Deinit()

	zigTry(t, tm.PrintString([]uint8("12345678901234567890")))
	expect(t, tm.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	tm.HorizontalTabBack()
	tm5TExpectCursor(t, tm, 16, 0)
	expect(t, tm.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	tm5TExpectGrid(t, tm, []string{"12345678901234567890", "...................."})

	// At column 0 nothing moves and pending wrap stays.
	tm.Screens.Active.CursorAbsolute(0, 0)
	tm.HorizontalTabBack()
	tm5TExpectCursor(t, tm, 0, 0)
	expect(t, tm.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
}

// Zig: test "term51: horizontalTabBack: dynamic range and narrow screens"
func TestTm5_term51_horizontalTabBack_dynamic_range_and_narrow_screens(t *testing.T) {
	func() {
		tm := tm5Init(t, 600, 1)
		defer tm.Deinit()
		tm.Screens.Active.CursorAbsolute(599, 0)
		tm.HorizontalTabBack()
		tm5TExpectCursor(t, tm, 592, 0)
		tm.HorizontalTabBack()
		tm5TExpectCursor(t, tm, 584, 0)
		tm.Screens.Active.CursorAbsolute(513, 0)
		tm.HorizontalTabBack()
		tm5TExpectCursor(t, tm, 512, 0)
		tm.HorizontalTabBack()
		tm5TExpectCursor(t, tm, 504, 0)
	}()
	func() {
		tm := tm5Init(t, 1, 1)
		defer tm.Deinit()
		tm.HorizontalTabBack()
		tm5TExpectCursor(t, tm, 0, 0)
	}()
	func() {
		tm := tm5Init(t, 10, 1)
		defer tm.Deinit()
		tm.Screens.Active.CursorAbsolute(9, 0)
		tm.HorizontalTabBack()
		tm5TExpectCursor(t, tm, 8, 0)
		tm.HorizontalTabBack()
		tm5TExpectCursor(t, tm, 0, 0)
	}()
}

// Zig: test "term51: cursorIsAtPrompt: fresh terminal is not at a prompt"
func TestTm5_term51_cursorIsAtPrompt_fresh_terminal_is_not_at_a_prompt(t *testing.T) {
	tm := tm5Init(t, 5, 3)
	defer tm.Deinit()
	expect(t, !tm.CursorIsAtPrompt(), "!t.cursorIsAtPrompt()")
}

// Zig: test "term51: cursorIsAtPrompt: every row mark and every cursor content state"
func TestTm5_term51_cursorIsAtPrompt_every_row_mark_and_every_cursor_content_state(t *testing.T) {
	tm := tm5Init(t, 5, 3)
	defer tm.Deinit()
	s := tm.Screens.Active

	contents := []PageCellSemanticContent{
		PageCellSemanticContentOutput,
		PageCellSemanticContentInput,
		PageCellSemanticContentPrompt,
	}
	marks := []PageRowSemanticPrompt{
		PageRowSemanticPromptNone,
		PageRowSemanticPromptPrompt,
		PageRowSemanticPromptPromptContinuation,
	}
	for _, mark := range marks {
		for _, content := range contents {
			s.Cursor.PageRow.semantic_prompt = mark
			s.Cursor.SemanticContent = content
			expected := mark != PageRowSemanticPromptNone || content != PageCellSemanticContentOutput
			expectEqual(t, expected, tm.CursorIsAtPrompt())
		}
	}
}

// Zig: test "term51: cursorIsAtPrompt: only the cursor row mark counts"
func TestTm5_term51_cursorIsAtPrompt_only_the_cursor_row_mark_counts(t *testing.T) {
	tm := tm5Init(t, 5, 3)
	defer tm.Deinit()

	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionPromptStart, "")))
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionEndCommand, "")))
	// Row 0 is marked as a prompt, content is output.
	expect(t, tm.CursorIsAtPrompt(), "t.cursorIsAtPrompt()")
	tm.SetCursorPos(2, 1)
	expect(t, !tm.CursorIsAtPrompt(), "!t.cursorIsAtPrompt()")
	tm.SetCursorPos(1, 1)
	expect(t, tm.CursorIsAtPrompt(), "t.cursorIsAtPrompt()")
}

// Zig: test "term51: cursorIsAtPrompt: alternate screen is never at a prompt"
func TestTm5_term51_cursorIsAtPrompt_alternate_screen_is_never_at_a_prompt(t *testing.T) {
	tm := tm5Init(t, 5, 3)
	defer tm.Deinit()

	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionPromptStart, "")))
	expect(t, tm.CursorIsAtPrompt(), "t.cursorIsAtPrompt()")

	tm5SwitchScreen(t, tm, ScreenSetKeyAlternate)
	// Mark the alternate screen the same way, it still reports false.
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionPromptStart, "")))
	expectEqual(t, PageRowSemanticPromptPrompt, tm.Screens.Active.Cursor.PageRow.semantic_prompt)
	expect(t, !tm.CursorIsAtPrompt(), "!t.cursorIsAtPrompt()")
	tm.Screens.Active.Cursor.SemanticContent = PageCellSemanticContentInput
	expect(t, !tm.CursorIsAtPrompt(), "!t.cursorIsAtPrompt()")

	tm5SwitchScreen(t, tm, ScreenSetKeyPrimary)
	expect(t, tm.CursorIsAtPrompt(), "t.cursorIsAtPrompt()")
}

// Zig: test "term51: semanticPrompt: fresh_line at column 0 does nothing"
func TestTm5_term51_semanticPrompt_fresh_line_at_column_0_does_nothing(t *testing.T) {
	tm := tm5Init(t, 5, 4)
	defer tm.Deinit()

	tm.SetCursorPos(2, 1)
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionFreshLine, "")))
	tm5TExpectCursor(t, tm, 0, 1)
	expectEqual(t, PageRowSemanticPromptNone, tm5TRowAt(t, tm, 1).semantic_prompt)
	expectEqual(t, PageCellSemanticContentOutput, tm.Screens.Active.Cursor.SemanticContent)
	expect(t, !tm.Screens.Active.SemanticPrompt.Seen, "!t.screens.active.semantic_prompt.seen")
}

// Zig: test "term51: semanticPrompt: fresh_line away from column 0 moves to the next line start"
func TestTm5_term51_semanticPrompt_fresh_line_away_from_column_0_moves_to_the_next_line_start(t *testing.T) {
	tm := tm5Init(t, 5, 4)
	defer tm.Deinit()

	zigTry(t, tm.PrintString([]uint8("ab")))
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionFreshLine, "k=r;redraw=0;cl=m")))
	tm5TExpectCursor(t, tm, 0, 1)
	tm5TExpectGrid(t, tm, []string{"ab...", ".....", ".....", "....."})
	// Options are ignored for L.
	expectEqual(t, OSCParsersSemanticPromptRedrawTrue, tm.Flags.ShellRedrawsPrompt)
	tm5ExpectClick(t, tm5ClickNone(), tm.Screens.Active.SemanticPrompt.Click)
	expect(t, !tm.Screens.Active.SemanticPrompt.Seen, "!t.screens.active.semantic_prompt.seen")
}

// Zig: test "term51: semanticPrompt: fresh_line on the last row scrolls"
func TestTm5_term51_semanticPrompt_fresh_line_on_the_last_row_scrolls(t *testing.T) {
	tm := tm5Init(t, 3, 2)
	defer tm.Deinit()

	zigTry(t, tm.PrintString([]uint8("abc")))
	tm.CarriageReturn()
	zigTry(t, tm.Linefeed())
	zigTry(t, tm.PrintString([]uint8("de")))
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionFreshLine, "")))
	tm5TExpectCursor(t, tm, 0, 1)
	tm5TExpectGrid(t, tm, []string{"de.", "..."})
	expectEqual(t, uint(1), tm5TScrollbackRows(tm))
}

// Zig: test "term51: semanticPrompt: fresh_line_new_prompt marks the row and content"
func TestTm5_term51_semanticPrompt_fresh_line_new_prompt_marks_the_row_and_content(t *testing.T) {
	tm := tm5Init(t, 5, 4)
	defer tm.Deinit()
	s := tm.Screens.Active

	expect(t, !s.SemanticPrompt.Seen, "!s.semantic_prompt.seen")
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "")))
	tm5TExpectCursor(t, tm, 0, 0)
	expectEqual(t, PageRowSemanticPromptPrompt, tm5TRowAt(t, tm, 0).semantic_prompt)
	expectEqual(t, PageCellSemanticContentPrompt, s.Cursor.SemanticContent)
	expect(t, !s.Cursor.SemanticContentClearEol, "!s.cursor.semantic_content_clear_eol")
	expect(t, s.SemanticPrompt.Seen, "s.semantic_prompt.seen")
	// Defaults of the options remain.
	expectEqual(t, OSCParsersSemanticPromptRedrawTrue, tm.Flags.ShellRedrawsPrompt)
	tm5ExpectClick(t, tm5ClickNone(), s.SemanticPrompt.Click)
}

// Zig: test "term51: semanticPrompt: fresh_line_new_prompt does a fresh line first"
func TestTm5_term51_semanticPrompt_fresh_line_new_prompt_does_a_fresh_line_first(t *testing.T) {
	tm := tm5Init(t, 5, 4)
	defer tm.Deinit()

	zigTry(t, tm.PrintString([]uint8("ab")))
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "")))
	tm5TExpectCursor(t, tm, 0, 1)
	expectEqual(t, PageRowSemanticPromptNone, tm5TRowAt(t, tm, 0).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptPrompt, tm5TRowAt(t, tm, 1).semantic_prompt)
}

// Zig: test "term51: semanticPrompt: fresh_line_new_prompt overrides the continuation mark index would add"
func TestTm5_term51_semanticPrompt_fresh_line_new_prompt_overrides_the_continuation_mark_index_would_add(t *testing.T) {
	tm := tm5Init(t, 5, 4)
	defer tm.Deinit()
	s := tm.Screens.Active

	// In input state at column > 0 the fresh line's index marks the new
	// row as a continuation, then the new prompt marks it as a prompt.
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionPromptStart, "")))
	zigTry(t, tm.Print('$'))
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionEndPromptStartInput, "")))
	zigTry(t, tm.Print('x'))
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "")))
	tm5TExpectCursor(t, tm, 0, 1)
	expectEqual(t, PageRowSemanticPromptPrompt, tm5TRowAt(t, tm, 0).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptPrompt, tm5TRowAt(t, tm, 1).semantic_prompt)
	expectEqual(t, PageCellSemanticContentPrompt, s.Cursor.SemanticContent)

	// With the eol flag set, index drops back to output without marking and
	// the prompt then sets prompt state with the flag clear.
	zigTry(t, tm.Print('$'))
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionEndPromptStartInputTerminateEol, "")))
	expect(t, s.Cursor.SemanticContentClearEol, "s.cursor.semantic_content_clear_eol")
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "")))
	tm5TExpectCursor(t, tm, 0, 2)
	expectEqual(t, PageRowSemanticPromptPrompt, tm5TRowAt(t, tm, 1).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptPrompt, tm5TRowAt(t, tm, 2).semantic_prompt)
	expectEqual(t, PageCellSemanticContentPrompt, s.Cursor.SemanticContent)
	expect(t, !s.Cursor.SemanticContentClearEol, "!s.cursor.semantic_content_clear_eol")
}

// Zig: test "term51: semanticPrompt: fresh_line_new_prompt at column 0 re-marks the current row"
func TestTm5_term51_semanticPrompt_fresh_line_new_prompt_at_column_0_re_marks_the_current_row(t *testing.T) {
	tm := tm5Init(t, 5, 4)
	defer tm.Deinit()

	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "k=c")))
	expectEqual(t, PageRowSemanticPromptPromptContinuation, tm5TRowAt(t, tm, 0).semantic_prompt)
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "")))
	tm5TExpectCursor(t, tm, 0, 0)
	expectEqual(t, PageRowSemanticPromptPrompt, tm5TRowAt(t, tm, 0).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptNone, tm5TRowAt(t, tm, 1).semantic_prompt)
}

// Zig: test "term51: semanticPrompt: prompt kind option selects the row mark"
func TestTm5_term51_semanticPrompt_prompt_kind_option_selects_the_row_mark(t *testing.T) {
	type Case struct {
		opts     string
		expected PageRowSemanticPrompt
	}
	cases := []Case{
		{opts: "", expected: PageRowSemanticPromptPrompt},
		{opts: "k=i", expected: PageRowSemanticPromptPrompt},
		{opts: "k=r", expected: PageRowSemanticPromptPrompt},
		{opts: "k=c", expected: PageRowSemanticPromptPromptContinuation},
		{opts: "k=s", expected: PageRowSemanticPromptPromptContinuation},
		{opts: "k=x", expected: PageRowSemanticPromptPrompt},
		{opts: "k=", expected: PageRowSemanticPromptPrompt},
		{opts: "k=ic", expected: PageRowSemanticPromptPrompt},
		{opts: "k=C", expected: PageRowSemanticPromptPrompt},
		{opts: "aid=7;k=s", expected: PageRowSemanticPromptPromptContinuation},
		{opts: "k=s;k=i", expected: PageRowSemanticPromptPromptContinuation},
		{opts: "kk=s", expected: PageRowSemanticPromptPrompt},
	}
	actions := []OSCParsersSemanticPromptCommandAction{
		OSCParsersSemanticPromptCommandActionFreshLineNewPrompt,
		OSCParsersSemanticPromptCommandActionNewCommand,
		OSCParsersSemanticPromptCommandActionPromptStart,
	}
	for _, action := range actions {
		for _, c := range cases {
			func() {
				tm := tm5Init(t, 5, 3)
				defer tm.Deinit()
				zigTry(t, tm.SemanticPrompt(tm5TPrompt(action, c.opts)))
				expectEqual(t, c.expected, tm5TRowAt(t, tm, 0).semantic_prompt)
				expectEqual(t, PageCellSemanticContentPrompt, tm.Screens.Active.Cursor.SemanticContent)
				expect(t, tm.Screens.Active.SemanticPrompt.Seen, "t.screens.active.semantic_prompt.seen")
			}()
		}
	}
}

// Zig: test "term51: semanticPrompt: redraw option"
func TestTm5_term51_semanticPrompt_redraw_option(t *testing.T) {
	type Case struct {
		opts     string
		expected OSCParsersSemanticPromptRedraw
	}
	// Every case starts from the opposite value so that "unchanged" is visible.
	cases := []Case{
		{opts: "redraw=0", expected: OSCParsersSemanticPromptRedrawFalse},
		{opts: "redraw=1", expected: OSCParsersSemanticPromptRedrawTrue},
		{opts: "redraw=last", expected: OSCParsersSemanticPromptRedrawLast},
		{opts: "redraw=2", expected: OSCParsersSemanticPromptRedrawLast},
		{opts: "redraw=", expected: OSCParsersSemanticPromptRedrawLast},
		{opts: "redraw=LAST", expected: OSCParsersSemanticPromptRedrawLast},
		{opts: "redraw=00", expected: OSCParsersSemanticPromptRedrawLast},
		{opts: "redrawx=0", expected: OSCParsersSemanticPromptRedrawLast},
		{opts: "", expected: OSCParsersSemanticPromptRedrawLast},
		{opts: "k=r", expected: OSCParsersSemanticPromptRedrawLast},
		{opts: "k=r;redraw=0", expected: OSCParsersSemanticPromptRedrawFalse},
		{opts: "redraw", expected: OSCParsersSemanticPromptRedrawFalse},
	}
	tm := tm5Init(t, 5, 3)
	defer tm.Deinit()
	tm.Flags.ShellRedrawsPrompt = OSCParsersSemanticPromptRedrawLast
	for _, c := range cases {
		// Re-seed with a value different from the expectation of invalid inputs.
		if c.opts == "redraw=2" {
			tm.Flags.ShellRedrawsPrompt = OSCParsersSemanticPromptRedrawLast
		}
		if c.opts == "redraw" {
			tm.Flags.ShellRedrawsPrompt = OSCParsersSemanticPromptRedrawFalse
		}
		zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, c.opts)))
		expectEqual(t, c.expected, tm.Flags.ShellRedrawsPrompt)
	}
}

// Zig: test "term51: semanticPrompt: redraw option is not applied by prompt_start"
func TestTm5_term51_semanticPrompt_redraw_option_is_not_applied_by_prompt_start(t *testing.T) {
	tm := tm5Init(t, 5, 3)
	defer tm.Deinit()

	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionPromptStart, "redraw=0;click_events=1;cl=line")))
	expectEqual(t, OSCParsersSemanticPromptRedrawTrue, tm.Flags.ShellRedrawsPrompt)
	tm5ExpectClick(t, tm5ClickNone(), tm.Screens.Active.SemanticPrompt.Click)
}

// Zig: test "term51: semanticPrompt: click_events and cl options"
func TestTm5_term51_semanticPrompt_click_events_and_cl_options(t *testing.T) {
	type Case struct {
		opts     string
		expected ScreenSemanticPromptSemanticClick
	}
	none := tm5ClickNone()
	cases := []Case{
		// From none: cl alone
		{opts: "cl=line", expected: tm5ClickCl(OSCParsersSemanticPromptClickLine)},
		{opts: "cl=m", expected: tm5ClickCl(OSCParsersSemanticPromptClickMultiple)},
		{opts: "cl=v", expected: tm5ClickCl(OSCParsersSemanticPromptClickConservativeVertical)},
		{opts: "cl=w", expected: tm5ClickCl(OSCParsersSemanticPromptClickSmartVertical)},
		// click_events wins over cl when true
		{opts: "click_events=1", expected: tm5ClickEvents()},
		{opts: "click_events=1;cl=line", expected: tm5ClickEvents()},
		{opts: "cl=line;click_events=1", expected: tm5ClickEvents()},
		// click_events false or invalid falls back to cl
		{opts: "click_events=0;cl=m", expected: tm5ClickCl(OSCParsersSemanticPromptClickMultiple)},
		{opts: "click_events=2;cl=v", expected: tm5ClickCl(OSCParsersSemanticPromptClickConservativeVertical)},
		{opts: "click_events=;cl=w", expected: tm5ClickCl(OSCParsersSemanticPromptClickSmartVertical)},
		{opts: "click_events=10;cl=line", expected: tm5ClickCl(OSCParsersSemanticPromptClickLine)},
	}
	for _, c := range cases {
		func() {
			tm := tm5Init(t, 5, 3)
			defer tm.Deinit()
			tm5ExpectClick(t, none, tm.Screens.Active.SemanticPrompt.Click)
			zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, c.opts)))
			tm5ExpectClick(t, c.expected, tm.Screens.Active.SemanticPrompt.Click)
		}()
	}

	// Options that do not change the value leave the earlier value in place.
	keep := []string{
		"",
		"click_events=0",
		"click_events=2",
		"cl=x",
		"cl=",
		"cl=lin",
		"cl=linee",
		"cl=M",
		"k=r",
		"click_events=0;cl=zz",
	}
	for _, opts := range keep {
		func() {
			tm := tm5Init(t, 5, 3)
			defer tm.Deinit()
			zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "cl=w")))
			tm5ExpectClick(t, tm5ClickCl(OSCParsersSemanticPromptClickSmartVertical), tm.Screens.Active.SemanticPrompt.Click)
			zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, opts)))
			tm5ExpectClick(t, tm5ClickCl(OSCParsersSemanticPromptClickSmartVertical), tm.Screens.Active.SemanticPrompt.Click)
		}()
	}

	// click_events state is sticky and then replaced by a later cl.
	tm := tm5Init(t, 5, 3)
	defer tm.Deinit()
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "click_events=1")))
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "")))
	tm5ExpectClick(t, tm5ClickEvents(), tm.Screens.Active.SemanticPrompt.Click)
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "click_events=0")))
	tm5ExpectClick(t, tm5ClickEvents(), tm.Screens.Active.SemanticPrompt.Click)
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "cl=m")))
	tm5ExpectClick(t, tm5ClickCl(OSCParsersSemanticPromptClickMultiple), tm.Screens.Active.SemanticPrompt.Click)
}

// Zig: test "term51: semanticPrompt: new_command behaves as fresh_line_new_prompt"
func TestTm5_term51_semanticPrompt_new_command_behaves_as_fresh_line_new_prompt(t *testing.T) {
	opts_list := []string{
		"",
		"k=r",
		"k=s;redraw=last",
		"redraw=0;click_events=1;cl=line",
		"aid=5;cl=v",
	}
	for _, opts := range opts_list {
		func() {
			a := tm5Init(t, 5, 3)
			defer a.Deinit()
			b := tm5Init(t, 5, 3)
			defer b.Deinit()

			zigTry(t, a.PrintString([]uint8("xy")))
			zigTry(t, b.PrintString([]uint8("xy")))
			zigTry(t, a.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, opts)))
			zigTry(t, b.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionNewCommand, opts)))

			tm5TExpectCursor(t, b, 0, 1)
			expectEqual(t, a.Screens.Active.Cursor.X, b.Screens.Active.Cursor.X)
			expectEqual(t, a.Screens.Active.Cursor.Y, b.Screens.Active.Cursor.Y)
			expectEqual(t, a.Screens.Active.Cursor.SemanticContent, b.Screens.Active.Cursor.SemanticContent)
			expectEqual(t, tm5TRowAt(t, a, 1).semantic_prompt, tm5TRowAt(t, b, 1).semantic_prompt)
			expectEqual(t, a.Flags.ShellRedrawsPrompt, b.Flags.ShellRedrawsPrompt)
			tm5ExpectClick(t, a.Screens.Active.SemanticPrompt.Click, b.Screens.Active.SemanticPrompt.Click)
			expectEqual(t, a.Screens.Active.SemanticPrompt.Seen, b.Screens.Active.SemanticPrompt.Seen)
		}()
	}
}

// Zig: test "term51: semanticPrompt: prompt_start does not move the cursor"
func TestTm5_term51_semanticPrompt_prompt_start_does_not_move_the_cursor(t *testing.T) {
	tm := tm5Init(t, 5, 3)
	defer tm.Deinit()

	zigTry(t, tm.PrintString([]uint8("ab")))
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionPromptStart, "")))
	tm5TExpectCursor(t, tm, 2, 0)
	expectEqual(t, PageRowSemanticPromptPrompt, tm5TRowAt(t, tm, 0).semantic_prompt)
	expectEqual(t, PageCellSemanticContentPrompt, tm.Screens.Active.Cursor.SemanticContent)
	expect(t, tm.Screens.Active.SemanticPrompt.Seen, "t.screens.active.semantic_prompt.seen")
}

// Zig: test "term51: semanticPrompt: end_prompt_start_input switches to explicit input"
func TestTm5_term51_semanticPrompt_end_prompt_start_input_switches_to_explicit_input(t *testing.T) {
	tm := tm5Init(t, 5, 3)
	defer tm.Deinit()
	s := tm.Screens.Active

	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionPromptStart, "")))
	zigTry(t, tm.Print('$'))
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionEndPromptStartInput, "k=c;aid=1")))
	expectEqual(t, PageCellSemanticContentInput, s.Cursor.SemanticContent)
	expect(t, !s.Cursor.SemanticContentClearEol, "!s.cursor.semantic_content_clear_eol")
	// Row mark and cursor untouched.
	expectEqual(t, PageRowSemanticPromptPrompt, tm5TRowAt(t, tm, 0).semantic_prompt)
	tm5TExpectCursor(t, tm, 1, 0)

	// From the eol flavour the flag is cleared again.
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionEndPromptStartInputTerminateEol, "")))
	expect(t, s.Cursor.SemanticContentClearEol, "s.cursor.semantic_content_clear_eol")
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionEndPromptStartInput, "")))
	expectEqual(t, PageCellSemanticContentInput, s.Cursor.SemanticContent)
	expect(t, !s.Cursor.SemanticContentClearEol, "!s.cursor.semantic_content_clear_eol")
}

// Zig: test "term51: semanticPrompt: end_prompt_start_input_terminate_eol sets the eol flag"
func TestTm5_term51_semanticPrompt_end_prompt_start_input_terminate_eol_sets_the_eol_flag(t *testing.T) {
	tm := tm5Init(t, 5, 4)
	defer tm.Deinit()
	s := tm.Screens.Active

	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionPromptStart, "")))
	zigTry(t, tm.Print('$'))
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionEndPromptStartInputTerminateEol, "")))
	expectEqual(t, PageCellSemanticContentInput, s.Cursor.SemanticContent)
	expect(t, s.Cursor.SemanticContentClearEol, "s.cursor.semantic_content_clear_eol")
	expectEqual(t, PageRowSemanticPromptPrompt, tm5TRowAt(t, tm, 0).semantic_prompt)
	tm5TExpectCursor(t, tm, 1, 0)

	// The next index ends the input state without marking the new row.
	tm.CarriageReturn()
	zigTry(t, tm.Linefeed())
	expectEqual(t, PageCellSemanticContentOutput, s.Cursor.SemanticContent)
	expect(t, !s.Cursor.SemanticContentClearEol, "!s.cursor.semantic_content_clear_eol")
	expectEqual(t, PageRowSemanticPromptNone, tm5TRowAt(t, tm, 1).semantic_prompt)
}

// Zig: test "term51: semanticPrompt: end_input_start_output clears a prompt row at column 0"
func TestTm5_term51_semanticPrompt_end_input_start_output_clears_a_prompt_row_at_column_0(t *testing.T) {
	marks := []PageRowSemanticPrompt{
		PageRowSemanticPromptNone,
		PageRowSemanticPromptPrompt,
		PageRowSemanticPromptPromptContinuation,
	}
	for _, mark := range marks {
		for _, x := range []uint{0, 1} {
			func() {
				tm := tm5Init(t, 5, 3)
				defer tm.Deinit()
				s := tm.Screens.Active

				s.Cursor.SemanticContent = PageCellSemanticContentInput
				s.Cursor.SemanticContentClearEol = true
				s.Cursor.PageRow.semantic_prompt = mark
				tm.Screens.Active.CursorAbsolute(SizeCellCountInt(x), 0)
				zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionEndInputStartOutput, "")))

				expectEqual(t, PageCellSemanticContentOutput, s.Cursor.SemanticContent)
				expect(t, !s.Cursor.SemanticContentClearEol, "!s.cursor.semantic_content_clear_eol")
				// The mark is dropped only when we are at column 0.
				expected := mark
				if x == 0 {
					expected = PageRowSemanticPromptNone
				}
				expectEqual(t, expected, tm5TRowAt(t, tm, 0).semantic_prompt)
				tm5TExpectCursor(t, tm, x, 0)
			}()
		}
	}
}

// Zig: test "term51: semanticPrompt: end_input_start_output only touches the cursor row"
func TestTm5_term51_semanticPrompt_end_input_start_output_only_touches_the_cursor_row(t *testing.T) {
	tm := tm5Init(t, 5, 3)
	defer tm.Deinit()

	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionPromptStart, "")))
	tm.SetCursorPos(2, 1)
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionPromptStart, "k=c")))
	tm.SetCursorPos(3, 1)
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionEndInputStartOutput, "")))
	expectEqual(t, PageRowSemanticPromptPrompt, tm5TRowAt(t, tm, 0).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptPromptContinuation, tm5TRowAt(t, tm, 1).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptNone, tm5TRowAt(t, tm, 2).semantic_prompt)
}

// Zig: test "term51: semanticPrompt: end_command resets to output and leaves row marks"
func TestTm5_term51_semanticPrompt_end_command_resets_to_output_and_leaves_row_marks(t *testing.T) {
	tm := tm5Init(t, 5, 3)
	defer tm.Deinit()
	s := tm.Screens.Active

	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionPromptStart, "")))
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionEndPromptStartInputTerminateEol, "")))
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionEndCommand, "0")))
	expectEqual(t, PageCellSemanticContentOutput, s.Cursor.SemanticContent)
	expect(t, !s.Cursor.SemanticContentClearEol, "!s.cursor.semantic_content_clear_eol")
	// Column 0 on a prompt row: unlike C, D keeps the mark.
	tm5TExpectCursor(t, tm, 0, 0)
	expectEqual(t, PageRowSemanticPromptPrompt, tm5TRowAt(t, tm, 0).semantic_prompt)

	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionEndCommand, "err=1;137")))
	expectEqual(t, PageCellSemanticContentOutput, s.Cursor.SemanticContent)
	expectEqual(t, PageRowSemanticPromptPrompt, tm5TRowAt(t, tm, 0).semantic_prompt)
	tm5TExpectCursor(t, tm, 0, 0)
}

// Zig: test "term51: semanticPrompt: complete prompt cycle"
func TestTm5_term51_semanticPrompt_complete_prompt_cycle(t *testing.T) {
	tm := tm5Init(t, 10, 5)
	defer tm.Deinit()
	s := tm.Screens.Active

	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "")))
	zigTry(t, tm.PrintString([]uint8("$ ")))
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionEndPromptStartInput, "")))
	zigTry(t, tm.PrintString([]uint8("ls")))
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionEndInputStartOutput, "")))
	expectEqual(t, PageCellSemanticContentOutput, s.Cursor.SemanticContent)
	// Row 0 keeps its mark since the cursor is not at column 0.
	expectEqual(t, PageRowSemanticPromptPrompt, tm5TRowAt(t, tm, 0).semantic_prompt)
	tm.CarriageReturn()
	zigTry(t, tm.Linefeed())
	zigTry(t, tm.PrintString([]uint8("out")))
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionEndCommand, "0")))
	expect(t, !tm.CursorIsAtPrompt(), "!t.cursorIsAtPrompt()")
	// The next prompt starts on a fresh line.
	zigTry(t, tm.SemanticPrompt(tm5TPrompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "")))
	tm5TExpectCursor(t, tm, 0, 2)
	expect(t, tm.CursorIsAtPrompt(), "t.cursorIsAtPrompt()")
	expectEqual(t, PageRowSemanticPromptNone, tm5TRowAt(t, tm, 1).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptPrompt, tm5TRowAt(t, tm, 2).semantic_prompt)
}

// Zig: test "term51: semanticPromptFreshLine: column 0 and the left margin are no-ops"
func TestTm5_term51_semanticPromptFreshLine_column_0_and_the_left_margin_are_no_ops(t *testing.T) {
	tm := tm5Init(t, 10, 5)
	defer tm.Deinit()

	// No margins: only column 0.
	tm.Screens.Active.CursorAbsolute(0, 2)
	zigTry(t, tm.semanticPromptFreshLine())
	tm5TExpectCursor(t, tm, 0, 2)

	// Left margin 3: the margin column itself and column 0 do nothing.
	tm5TRegion(tm, 0, 4, 3, 7)
	tm.Screens.Active.CursorAbsolute(3, 2)
	zigTry(t, tm.semanticPromptFreshLine())
	tm5TExpectCursor(t, tm, 3, 2)
	tm.Screens.Active.CursorAbsolute(0, 2)
	zigTry(t, tm.semanticPromptFreshLine())
	tm5TExpectCursor(t, tm, 0, 2)
}

// Zig: test "term51: semanticPromptFreshLine: moves to the next line start without margins"
func TestTm5_term51_semanticPromptFreshLine_moves_to_the_next_line_start_without_margins(t *testing.T) {
	tm := tm5Init(t, 10, 5)
	defer tm.Deinit()

	tm.Screens.Active.CursorAbsolute(4, 2)
	zigTry(t, tm.semanticPromptFreshLine())
	tm5TExpectCursor(t, tm, 0, 3)
	tm.Screens.Active.CursorAbsolute(9, 3)
	zigTry(t, tm.semanticPromptFreshLine())
	tm5TExpectCursor(t, tm, 0, 4)
	// On the last row it scrolls instead.
	tm.Screens.Active.CursorAbsolute(1, 4)
	zigTry(t, tm.PrintString([]uint8("")))
	zigTry(t, tm.semanticPromptFreshLine())
	tm5TExpectCursor(t, tm, 0, 4)
	expectEqual(t, uint(1), tm5TScrollbackRows(tm))
}

// Zig: test "term51: semanticPromptFreshLine: cursor left of, at, inside, and right of the left margin"
func TestTm5_term51_semanticPromptFreshLine_cursor_left_of_at_inside_and_right_of_the_left_margin(t *testing.T) {
	type Case struct{ x, expect_x, expect_y uint }
	// left margin 3, right margin 7
	cases := []Case{
		{x: 0, expect_x: 0, expect_y: 1}, // no-op
		{x: 1, expect_x: 0, expect_y: 2}, // left of the margin: column 0 is "initial", moves, CR goes to 0
		{x: 2, expect_x: 0, expect_y: 2},
		{x: 3, expect_x: 3, expect_y: 1}, // no-op
		{x: 4, expect_x: 3, expect_y: 2},
		{x: 7, expect_x: 3, expect_y: 2},
		{x: 8, expect_x: 3, expect_y: 2}, // right of the right margin: CR goes to the left margin
		{x: 9, expect_x: 3, expect_y: 2},
	}
	for _, c := range cases {
		func() {
			tm := tm5Init(t, 10, 5)
			defer tm.Deinit()
			tm5TRegion(tm, 0, 4, 3, 7)
			tm.Screens.Active.CursorAbsolute(SizeCellCountInt(c.x), 1)
			zigTry(t, tm.semanticPromptFreshLine())
			tm5TExpectCursor(t, tm, c.expect_x, c.expect_y)
		}()
	}
}

// Zig: test "term51: semanticPromptFreshLine: origin mode sends the carriage return to the left margin"
func TestTm5_term51_semanticPromptFreshLine_origin_mode_sends_the_carriage_return_to_the_left_margin(t *testing.T) {
	tm := tm5Init(t, 10, 5)
	defer tm.Deinit()

	tm5TRegion(tm, 0, 4, 3, 7)
	tm.Modes.Set(ModesModeOrigin, true)
	tm.Screens.Active.CursorAbsolute(1, 1)
	zigTry(t, tm.semanticPromptFreshLine())
	tm5TExpectCursor(t, tm, 3, 2)
}

// Zig: test "term51: semanticPromptFreshLine: scrolls the region at the bottom margin"
func TestTm5_term51_semanticPromptFreshLine_scrolls_the_region_at_the_bottom_margin(t *testing.T) {
	tm := tm5Init(t, 5, 5)
	defer tm.Deinit()
	tm5TFill(t, tm)

	tm5TRegion(tm, 1, 3, 1, 3)
	tm.Screens.Active.CursorAbsolute(2, 3)
	zigTry(t, tm.semanticPromptFreshLine())
	tm5TExpectCursor(t, tm, 1, 3)
	tm5TExpectGrid(t, tm, []string{
		"ABCDE",
		"FLMNJ",
		"KQRSO",
		"P...T",
		"UVWXY",
	})
}

// Zig: test "term51: semanticPromptFreshLine: pending wrap kept by the no-op, cleared by a move"
func TestTm5_term51_semanticPromptFreshLine_pending_wrap_kept_by_the_no_op_cleared_by_a_move(t *testing.T) {
	tm := tm5Init(t, 10, 5)
	defer tm.Deinit()

	tm.Screens.Active.CursorAbsolute(0, 1)
	tm.Screens.Active.Cursor.PendingWrap = true
	zigTry(t, tm.semanticPromptFreshLine())
	tm5TExpectCursor(t, tm, 0, 1)
	expect(t, tm.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")

	tm.Screens.Active.CursorAbsolute(5, 1)
	tm.Screens.Active.Cursor.PendingWrap = true
	zigTry(t, tm.semanticPromptFreshLine())
	tm5TExpectCursor(t, tm, 0, 2)
	expect(t, !tm.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
}

// tm5TGrid is term51TGrid.
type tm5TGrid [5][5]uint8

// tm5TLabels is term51TLabels: the labels tm5TFill writes into a 5x5
// terminal: "ABCDE" ... "UVWXY".
func tm5TLabels() tm5TGrid {
	var g tm5TGrid
	for y := uint(0); y < 5; y++ {
		for x := uint(0); x < 5; x++ {
			g[y][x] = uint8(tm5TLabel(5, x, y))
		}
	}
	return g
}

// tm5TExpectModel is term51TExpectModel.
func tm5TExpectModel(t *testing.T, tm *Terminal, g *tm5TGrid) {
	t.Helper()
	for y := uint(0); y < 5; y++ {
		expectEqualStrings(t, string(g[y][:]), string(tm5TGridRow(t, tm, y)))
	}
}

// tm5TModelScrollUp is term51TModelScrollUp: the reference model of scrolling
// the rectangle up by count (blank = fill).
func tm5TModelScrollUp(g *tm5TGrid, top uint, bottom uint, left uint, right uint, count uint, blank uint8) {
	n := min(count, bottom-top+1)
	for y := top; y < bottom+1; y++ {
		for x := left; x < right+1; x++ {
			if y+n <= bottom {
				g[y][x] = g[y+n][x]
			} else {
				g[y][x] = blank
			}
		}
	}
}

// tm5TModelScrollDown is term51TModelScrollDown: the reference model of
// scrolling the rectangle down by count (blank = fill).
func tm5TModelScrollDown(g *tm5TGrid, top uint, bottom uint, left uint, right uint, count uint, blank uint8) {
	n := min(count, bottom-top+1)
	y := bottom + 1
	for y > top {
		y -= 1
		for x := left; x < right+1; x++ {
			if y >= top+n {
				g[y][x] = g[y-n][x]
			} else {
				g[y][x] = blank
			}
		}
	}
}

// tm5TExpectCursorConsistent is term51TExpectCursorConsistent: the cursor's
// cached row and cell pointers and its pin match its x/y.
func tm5TExpectCursorConsistent(t *testing.T, tm *Terminal) {
	t.Helper()
	s := tm.Screens.Active
	lc := zigUnwrap(t, s.Pages.GetCell(tm5ActivePoint(uint(s.Cursor.X), uint(s.Cursor.Y))))
	expect(t, s.Cursor.PageCell == lc.Cell, "s.cursor.page_cell == lc.cell")
	expect(t, s.Cursor.PageRow == lc.Row, "s.cursor.page_row == lc.row")
	expectEqual(t, s.Cursor.X, s.Cursor.PagePin.X)
}

// tm5TBg is term51TBg.
var tm5TBg = SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: 1, G: 2, B: 3}}

// tm5TPairs is term51TPairs: (top, bottom) / (left, right) pairs used by the
// sweeps: full, two rows at the top, top three, bottom four, two rows near
// the top, middle two, bottom two.
var tm5TPairs = [][2]uint{
	{0, 4}, {0, 1}, {0, 3}, {1, 4}, {1, 2}, {2, 3}, {3, 4},
}

// tm5TBlank is `if (bg) '#' else '.'`.
func tm5TBlank(bg bool) uint8 {
	if bg {
		return '#'
	}
	return '.'
}

// tm5TIndexCase is term51TIndexCase.
func tm5TIndexCase(t *testing.T, top uint, bottom uint, left uint, right uint, cx uint, cy uint, bg bool) {
	t.Helper()
	defer func() {
		if t.Failed() {
			t.Logf("index case region=(%d,%d,%d,%d) cursor=(%d,%d) bg=%v", top, bottom, left, right, cx, cy, bg)
		}
	}()
	tm := tm5Init(t, 5, 5)
	defer tm.Deinit()
	tm5TFill(t, tm)
	g := tm5TLabels()
	tm.SetCursorPos(cy+1, cx+1)
	tm5TRegion(tm, top, bottom, left, right)
	if bg {
		tm5Attr(t, tm, tm5TBg)
	}
	tm.Screens.Active.Cursor.PendingWrap = true
	// Zig-only: if (comptime build_options.kitty_graphics) t.screens.active.kitty_images.dirty = false;

	zigTry(t, tm.Index())

	ey := cy
	scrolled := false
	if cy < top || cy > bottom {
		if cy < 4 {
			ey = cy + 1
		}
	} else if cy == bottom && cx >= left && cx <= right {
		tm5TModelScrollUp(&g, top, bottom, left, right, 1, tm5TBlank(bg))
		scrolled = true
	} else if cy < bottom {
		ey = cy + 1
	}

	tm5TExpectModel(t, tm, &g)
	tm5TExpectCursor(t, tm, cx, ey)
	tm5TExpectCursorConsistent(t, tm)
	expect(t, !tm.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
	full := top == 0 && left == 0 && right == 4
	var want uint
	if scrolled && full {
		want = 1
	}
	expectEqual(t, want, tm5TScrollbackRows(tm))
	// Zig-only: if (comptime build_options.kitty_graphics) { try term51TTesting.expectEqual(scrolled, t.screens.active.kitty_images.dirty); }
	tm5TExpectRegion(t, tm, top, bottom, left, right)
}

// Zig: test "term51: index: sweep of every region pair and cursor position against a model"
func TestTm5_term51_index_sweep_of_every_region_pair_and_cursor_position_against_a_model(t *testing.T) {
	for _, tb := range tm5TPairs {
		for _, lr := range tm5TPairs {
			for cy := uint(0); cy < 5; cy++ {
				for cx := uint(0); cx < 5; cx++ {
					tm5TIndexCase(t, tb[0], tb[1], lr[0], lr[1], cx, cy, false)
					tm5TIndexCase(t, tb[0], tb[1], lr[0], lr[1], cx, cy, true)
				}
			}
		}
	}
}

// Zig: test "term51: index: outside the region moves down one row and never scrolls"
func TestTm5_term51_index_outside_the_region_moves_down_one_row_and_never_scrolls(t *testing.T) {
	tm := tm5Init(t, 5, 5)
	defer tm.Deinit()
	tm5TFill(t, tm)
	original := []string{"ABCDE", "FGHIJ", "KLMNO", "PQRST", "UVWXY"}

	tm.SetTopAndBottomMargin(2, 3) // rows 1..2
	tm.SetCursorPos(1, 3)
	tm.Screens.Active.Cursor.PendingWrap = true
	zigTry(t, tm.Index())
	tm5TExpectCursor(t, tm, 2, 1)
	expect(t, !tm.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")

	tm.Screens.Active.CursorAbsolute(4, 3)
	tm.Screens.Active.Cursor.PendingWrap = true
	zigTry(t, tm.Index())
	tm5TExpectCursor(t, tm, 4, 4)
	expect(t, !tm.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")

	// On the last row of the screen it stays and still resets pending wrap.
	tm.Screens.Active.Cursor.PendingWrap = true
	zigTry(t, tm.Index())
	tm5TExpectCursor(t, tm, 4, 4)
	expect(t, !tm.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")

	tm5TExpectGrid(t, tm, original)
	expectEqual(t, uint(0), tm5TScrollbackRows(tm))
}

// Zig: test "term51: index: inside the region above the bottom margin moves down"
func TestTm5_term51_index_inside_the_region_above_the_bottom_margin_moves_down(t *testing.T) {
	tm := tm5Init(t, 5, 5)
	defer tm.Deinit()
	tm5TFill(t, tm)

	tm5TRegion(tm, 1, 3, 1, 3)
	// Even when the cursor column is outside the left/right margins.
	tm.Screens.Active.CursorAbsolute(0, 1)
	zigTry(t, tm.Index())
	tm5TExpectCursor(t, tm, 0, 2)
	tm.Screens.Active.CursorAbsolute(4, 2)
	zigTry(t, tm.Index())
	tm5TExpectCursor(t, tm, 4, 3)
	tm.Screens.Active.CursorAbsolute(2, 1)
	zigTry(t, tm.Index())
	tm5TExpectCursor(t, tm, 2, 2)
	tm5TExpectGrid(t, tm, []string{"ABCDE", "FGHIJ", "KLMNO", "PQRST", "UVWXY"})
}

// Zig: test "term51: index: on the bottom margin outside the column margins does nothing"
func TestTm5_term51_index_on_the_bottom_margin_outside_the_column_margins_does_nothing(t *testing.T) {
	tm := tm5Init(t, 5, 5)
	defer tm.Deinit()
	tm5TFill(t, tm)

	tm5TRegion(tm, 1, 3, 1, 3)
	for _, x := range []uint{0, 4} {
		tm.Screens.Active.CursorAbsolute(SizeCellCountInt(x), 3)
		tm.Screens.Active.Cursor.PendingWrap = true
		// Zig-only: if (comptime build_options.kitty_graphics) t.screens.active.kitty_images.dirty = false;
		zigTry(t, tm.Index())
		tm5TExpectCursor(t, tm, x, 3)
		expect(t, !tm.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
		tm5TExpectGrid(t, tm, []string{"ABCDE", "FGHIJ", "KLMNO", "PQRST", "UVWXY"})
		// Zig-only: if (comptime build_options.kitty_graphics) { try term51TTesting.expect(!t.screens.active.kitty_images.dirty); }
	}
}

// Zig: test "term51: index: full screen scrolls into scrollback"
func TestTm5_term51_index_full_screen_scrolls_into_scrollback(t *testing.T) {
	tm := tm5Init(t, 5, 5)
	defer tm.Deinit()
	tm5TFill(t, tm)

	tm.Screens.Active.CursorAbsolute(2, 4)
	zigTry(t, tm.Index())
	tm5TExpectCursor(t, tm, 2, 4)
	tm5TExpectGrid(t, tm, []string{"FGHIJ", "KLMNO", "PQRST", "UVWXY", "....."})
	expectEqual(t, uint(1), tm5TScrollbackRows(tm))

	tm.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	{
		str, err := tm.PlainString()
		zigTry(t, err)
		// Zig-only: defer term51TTesting.allocator.free(str)
		expectEqualStrings(t, "ABCDE\nFGHIJ\nKLMNO\nPQRST\nUVWXY", string(str))
	}
}

// Zig: test "term51: index: region at the top with a bottom margin keeps rows below it"
func TestTm5_term51_index_region_at_the_top_with_a_bottom_margin_keeps_rows_below_it(t *testing.T) {
	tm := tm5Init(t, 5, 5)
	defer tm.Deinit()
	tm5TFill(t, tm)

	tm.SetTopAndBottomMargin(1, 3)
	tm.Screens.Active.CursorAbsolute(3, 2)
	zigTry(t, tm.Index())
	tm5TExpectCursor(t, tm, 3, 2)
	tm5TExpectGrid(t, tm, []string{"FGHIJ", "KLMNO", ".....", "PQRST", "UVWXY"})
	expectEqual(t, uint(1), tm5TScrollbackRows(tm))

	tm.ScrollViewport(TerminalScrollViewport{Tag: TerminalScrollViewportTop})
	{
		str, err := tm.PlainString()
		zigTry(t, err)
		// Zig-only: defer term51TTesting.allocator.free(str)
		expectEqualStrings(t, "ABCDE\nFGHIJ\nKLMNO\n\nPQRST", string(str))
	}
}

// Zig: test "term51: index: fast path for a region below the top row"
func TestTm5_term51_index_fast_path_for_a_region_below_the_top_row(t *testing.T) {
	tm := tm5Init(t, 5, 5)
	defer tm.Deinit()
	tm5TFill(t, tm)

	tm.SetTopAndBottomMargin(2, 4)
	tm.Screens.Active.CursorAbsolute(2, 3)
	tm.clearDirty()
	zigTry(t, tm.Index())
	tm5TExpectCursor(t, tm, 2, 3)
	tm5TExpectCursorConsistent(t, tm)
	tm5TExpectGrid(t, tm, []string{"ABCDE", "KLMNO", "PQRST", ".....", "UVWXY"})
	expectEqual(t, uint(0), tm5TScrollbackRows(tm))
	for y := uint(1); y < 4; y++ {
		expect(t, tm.isDirty(tm5ActivePoint(0, y)), "t.isDirty(.{ .active = .{ .x = 0, .y = y } })")
	}

	// Region reaching the bottom of the screen.
	tm.SetTopAndBottomMargin(3, 5)
	tm.Screens.Active.CursorAbsolute(0, 4)
	zigTry(t, tm.Index())
	tm5TExpectCursor(t, tm, 0, 4)
	tm5TExpectGrid(t, tm, []string{"ABCDE", "KLMNO", ".....", "UVWXY", "....."})
	expectEqual(t, uint(0), tm5TScrollbackRows(tm))

	// Two-row region.
	tm.SetTopAndBottomMargin(1, 2)
	tm.Screens.Active.CursorAbsolute(4, 1)
	zigTry(t, tm.Index())
	tm5TExpectCursor(t, tm, 4, 1)
	tm5TExpectGrid(t, tm, []string{"KLMNO", ".....", ".....", "UVWXY", "....."})
}

// Zig: test "term51: index: fast path keeps a non-background cursor style"
func TestTm5_term51_index_fast_path_keeps_a_non_background_cursor_style(t *testing.T) {
	tm := tm5Init(t, 5, 5)
	defer tm.Deinit()
	tm5TFill(t, tm)

	tm.SetTopAndBottomMargin(2, 4)
	tm5Attr(t, tm, SGRAttribute{Tag: SGRAttributeTagBold})
	tm5Attr(t, tm, SGRAttribute{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: ColorRGB{R: 9, G: 9, B: 9}})
	tm5Attr(t, tm, SGRAttribute{Tag: SGRAttributeTagInverse})
	tm5Attr(t, tm, SGRAttribute{Tag: SGRAttributeTagUnderlineColor, UnderlineColor: ColorRGB{R: 1, G: 1, B: 1}})
	tm.Screens.Active.CursorAbsolute(2, 3)
	style_id := tm.Screens.Active.Cursor.StyleID
	expect(t, style_id != StyleDefaultID, "style_id != style.default_id")

	zigTry(t, tm.Index())
	tm5TExpectGrid(t, tm, []string{"ABCDE", "KLMNO", "PQRST", ".....", "UVWXY"})
	expect(t, tm.Screens.Active.Cursor.StyleID != StyleDefaultID, "t.screens.active.cursor.style_id != style.default_id")
	expect(t, tm.Screens.Active.Cursor.Style.Flags.Bold, "t.screens.active.cursor.style.flags.bold")
	expect(t, tm.Screens.Active.Cursor.Style.Flags.Inverse, "t.screens.active.cursor.style.flags.inverse")

	// Printing afterwards uses the style.
	zigTry(t, tm.Print('z'))
	c := tm5TCellAt(t, tm, 2, 3)
	expectEqual(t, uint32('z'), c.content.codepoint)
	expectEqual(t, tm.Screens.Active.Cursor.StyleID, c.style_id)
}

// Zig: test "term51: index: background colour forces the slow path and fills the blank row"
func TestTm5_term51_index_background_colour_forces_the_slow_path_and_fills_the_blank_row(t *testing.T) {
	type Case struct {
		top, bottom, left, right uint
		attr                     SGRAttribute
	}
	rgb0 := SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: 0, G: 0, B: 0}}
	pal0 := SGRAttribute{Tag: SGRAttributeTag256Bg, X256Bg: 0}
	pal200 := SGRAttribute{Tag: SGRAttributeTag256Bg, X256Bg: 200}
	cases := []Case{
		{top: 1, bottom: 3, left: 0, right: 4, attr: tm5TBg},
		{top: 1, bottom: 3, left: 0, right: 4, attr: rgb0},
		{top: 1, bottom: 3, left: 0, right: 4, attr: pal0},
		{top: 1, bottom: 3, left: 0, right: 4, attr: pal200},
		{top: 1, bottom: 3, left: 0, right: 4, attr: SGRAttribute{Tag: SGRAttributeTag8Bg, X8Bg: ColorNameRed}},
		{top: 1, bottom: 3, left: 1, right: 3, attr: tm5TBg},
		{top: 0, bottom: 3, left: 0, right: 4, attr: tm5TBg},
		{top: 0, bottom: 4, left: 0, right: 4, attr: pal0},
	}
	for _, c := range cases {
		func() {
			tm := tm5Init(t, 5, 5)
			defer tm.Deinit()
			tm5TFill(t, tm)
			tm5TRegion(tm, c.top, c.bottom, c.left, c.right)
			tm5Attr(t, tm, c.attr)
			tm.Screens.Active.CursorAbsolute(SizeCellCountInt(c.left), SizeCellCountInt(c.bottom))
			zigTry(t, tm.Index())

			g := tm5TLabels()
			tm5TModelScrollUp(&g, c.top, c.bottom, c.left, c.right, 1, '#')
			tm5TExpectModel(t, tm, &g)
			tm5TExpectCursor(t, tm, c.left, c.bottom)

			// The blank cells carry the colour.
			for x := c.left; x < c.right+1; x++ {
				cell := tm5TCellAt(t, tm, x, c.bottom)
				switch c.attr.Tag {
				case SGRAttributeTagDirectColorBg:
					v := c.attr.DirectColorBg
					expectEqual(t, PageCellContentTagBgColorRgb, cell.content_tag)
					expectEqual(t, PageCellRGB{R: v.R, G: v.G, B: v.B}, cell.content.color_rgb)
				case SGRAttributeTag256Bg:
					v := c.attr.X256Bg
					expectEqual(t, PageCellContentTagBgColorPalette, cell.content_tag)
					expectEqual(t, v, cell.content.color_palette)
				case SGRAttributeTag8Bg:
					v := c.attr.X8Bg
					expectEqual(t, PageCellContentTagBgColorPalette, cell.content_tag)
					expectEqual(t, uint8(v), cell.content.color_palette)
				default:
					panic("unreachable")
				}
			}
		}()
	}
}

// Zig: test "term51: index: reset background returns to plain blank cells"
func TestTm5_term51_index_reset_background_returns_to_plain_blank_cells(t *testing.T) {
	tm := tm5Init(t, 5, 5)
	defer tm.Deinit()
	tm5TFill(t, tm)

	tm.SetTopAndBottomMargin(2, 4)
	tm5Attr(t, tm, tm5TBg)
	tm5Attr(t, tm, SGRAttribute{Tag: SGRAttributeTagResetBg})
	tm.Screens.Active.CursorAbsolute(0, 3)
	zigTry(t, tm.Index())
	tm5TExpectGrid(t, tm, []string{"ABCDE", "KLMNO", "PQRST", ".....", "UVWXY"})
	expectEqual(t, PageCellContentTagCodepoint, tm5TCellAt(t, tm, 0, 3).content_tag)
}

// Zig: test "term51: index: attributes without a background colour leave blank cells plain"
func TestTm5_term51_index_attributes_without_a_background_colour_leave_blank_cells_plain(t *testing.T) {
	attrs := []SGRAttribute{
		{Tag: SGRAttributeTagBold},
		{Tag: SGRAttributeTagInverse},
		{Tag: SGRAttributeTagItalic},
		{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: ColorRGB{R: 1, G: 2, B: 3}},
		{Tag: SGRAttributeTag256Fg, X256Fg: 7},
		{Tag: SGRAttributeTagUnderlineColor, UnderlineColor: ColorRGB{R: 1, G: 1, B: 1}},
		{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineSingle},
	}
	for _, attr := range attrs {
		func() {
			tm := tm5Init(t, 5, 5)
			defer tm.Deinit()
			tm5TFill(t, tm)
			tm.SetTopAndBottomMargin(2, 4)
			tm5Attr(t, tm, attr)
			tm.Screens.Active.CursorAbsolute(1, 3)
			zigTry(t, tm.Index())
			tm5TExpectGrid(t, tm, []string{"ABCDE", "KLMNO", "PQRST", ".....", "UVWXY"})
			cell := tm5TCellAt(t, tm, 1, 3)
			expectEqual(t, PageCellContentTagCodepoint, cell.content_tag)
			expectEqual(t, uint32(0), cell.content.codepoint)
		}()
	}
}

// Zig: test "term51: index: full width region with a background fills the whole new row"
func TestTm5_term51_index_full_width_region_with_a_background_fills_the_whole_new_row(t *testing.T) {
	tm := tm5Init(t, 5, 5)
	defer tm.Deinit()
	tm5TFill(t, tm)

	tm5Attr(t, tm, tm5TBg)
	tm.Screens.Active.CursorAbsolute(2, 4)
	zigTry(t, tm.Index())
	tm5TExpectGrid(t, tm, []string{"FGHIJ", "KLMNO", "PQRST", "UVWXY", "#####"})
	expectEqual(t, uint(1), tm5TScrollbackRows(tm))
}

// Zig: test "term51: index: fast path and slow path treat row wrap flags differently"
func TestTm5_term51_index_fast_path_and_slow_path_treat_row_wrap_flags_differently(t *testing.T) {
	// Upstream pins: the fast path (eraseRowBounded) rotates whole rows so
	// wrap flags travel with them and the blank row gets the flags of the
	// erased row; the slow path (a background colour) clears the wrap and
	// wrap_continuation flags of every shifted row.
	var results [2][5][2]bool
	for i, bg := range []bool{false, true} {
		func() {
			tm := tm5Init(t, 5, 5)
			defer tm.Deinit()
			tm5TFill(t, tm)
			tm.SetTopAndBottomMargin(2, 4)
			tm5TRowAt(t, tm, 1).wrap = true
			tm5TRowAt(t, tm, 2).wrap = true
			tm5TRowAt(t, tm, 2).wrap_continuation = true
			tm5TRowAt(t, tm, 3).wrap_continuation = true
			if bg {
				tm5Attr(t, tm, tm5TBg)
			}
			tm.Screens.Active.CursorAbsolute(0, 3)
			zigTry(t, tm.Index())
			for y := uint(0); y < 5; y++ {
				results[i][y] = [2]bool{tm5TRowAt(t, tm, y).wrap, tm5TRowAt(t, tm, y).wrap_continuation}
			}
		}()
	}
	// Fast path: rows 2, 3 -> 1, 2 with their flags; erased row 1 -> 3.
	expectEqual(t, [2]bool{false, false}, results[0][0])
	expectEqual(t, [2]bool{true, true}, results[0][1])
	expectEqual(t, [2]bool{false, true}, results[0][2])
	expectEqual(t, [2]bool{true, false}, results[0][3])
	expectEqual(t, [2]bool{false, false}, results[0][4])
	// Slow path.
	expectEqual(t, [2]bool{false, false}, results[1][0])
	expectEqual(t, [2]bool{false, false}, results[1][1])
	expectEqual(t, [2]bool{false, false}, results[1][2])
	expectEqual(t, [2]bool{false, false}, results[1][3])
	expectEqual(t, [2]bool{false, false}, results[1][4])
}

// tm5TSemanticScenario is term51TSemanticScenario.
type tm5TSemanticScenario struct {
	top    uint
	bottom uint
	left   uint
	right  uint
	cx     uint
	cy     uint
	bg     bool
}

// tm5TSemanticScenarios is term51TSemanticScenarios.
var tm5TSemanticScenarios = []tm5TSemanticScenario{
	{top: 0, bottom: 4, left: 0, right: 4, cx: 1, cy: 4},           // full screen scroll
	{top: 0, bottom: 2, left: 0, right: 4, cx: 1, cy: 2},           // top region scroll
	{top: 1, bottom: 3, left: 0, right: 4, cx: 1, cy: 3},           // fast path
	{top: 1, bottom: 3, left: 1, right: 3, cx: 2, cy: 3},           // slow path, margins
	{top: 1, bottom: 3, left: 0, right: 4, cx: 1, cy: 3, bg: true}, // slow path, bg
	{top: 1, bottom: 3, left: 0, right: 4, cx: 1, cy: 1},           // move inside
	{top: 1, bottom: 3, left: 0, right: 4, cx: 1, cy: 0},           // move above region
	{top: 1, bottom: 3, left: 0, right: 4, cx: 1, cy: 4},           // last row below region
	{top: 1, bottom: 3, left: 1, right: 3, cx: 0, cy: 3},           // bottom margin, outside columns
}

// tm5TSemanticSetup is term51TSemanticSetup.
func tm5TSemanticSetup(t *testing.T, sc tm5TSemanticScenario) *Terminal {
	t.Helper()
	tm := tm5Init(t, 5, 5)
	// Zig-only: errdefer t.deinit(term51TTesting.allocator)
	tm5TFill(t, tm)
	tm5TRegion(tm, sc.top, sc.bottom, sc.left, sc.right)
	if sc.bg {
		tm5Attr(t, tm, tm5TBg)
	}
	tm.Screens.Active.CursorAbsolute(SizeCellCountInt(sc.cx), SizeCellCountInt(sc.cy))
	return tm
}

// Zig: test "term51: index: prompt and input content mark the cursor row as a continuation"
func TestTm5_term51_index_prompt_and_input_content_mark_the_cursor_row_as_a_continuation(t *testing.T) {
	contents := []PageCellSemanticContent{PageCellSemanticContentPrompt, PageCellSemanticContentInput}
	for _, sc := range tm5TSemanticScenarios {
		for _, content := range contents {
			func() {
				defer func() {
					if t.Failed() {
						t.Logf("scenario %+v %v", sc, uint8(content))
					}
				}()
				tm := tm5TSemanticSetup(t, sc)
				defer tm.Deinit()
				tm.Screens.Active.Cursor.SemanticContent = content
				zigTry(t, tm.Index())

				// Exactly one row is marked and it is the cursor row.
				marked := uint(0)
				for y := uint(0); y < 5; y++ {
					if tm5TRowAt(t, tm, y).semantic_prompt != PageRowSemanticPromptNone {
						marked += 1
						expectEqual(t, uint(tm.Screens.Active.Cursor.Y), y)
						expectEqual(t, PageRowSemanticPromptPromptContinuation, tm5TRowAt(t, tm, y).semantic_prompt)
					}
				}
				expectEqual(t, uint(1), marked)
				expectEqual(t, content, tm.Screens.Active.Cursor.SemanticContent)
				expect(t, !tm.Screens.Active.Cursor.SemanticContentClearEol, "!t.screens.active.cursor.semantic_content_clear_eol")
			}()
		}
	}
}

// Zig: test "term51: index: input with the end-of-line flag returns to output without marking"
func TestTm5_term51_index_input_with_the_end_of_line_flag_returns_to_output_without_marking(t *testing.T) {
	for _, sc := range tm5TSemanticScenarios {
		func() {
			defer func() {
				if t.Failed() {
					t.Logf("scenario %+v", sc)
				}
			}()
			tm := tm5TSemanticSetup(t, sc)
			defer tm.Deinit()
			tm.Screens.Active.Cursor.SemanticContent = PageCellSemanticContentInput
			tm.Screens.Active.Cursor.SemanticContentClearEol = true
			zigTry(t, tm.Index())

			for y := uint(0); y < 5; y++ {
				expectEqual(t, PageRowSemanticPromptNone, tm5TRowAt(t, tm, y).semantic_prompt)
			}
			expectEqual(t, PageCellSemanticContentOutput, tm.Screens.Active.Cursor.SemanticContent)
			expect(t, !tm.Screens.Active.Cursor.SemanticContentClearEol, "!t.screens.active.cursor.semantic_content_clear_eol")
		}()
	}
}

// Zig: test "term51: index: output content never marks rows"
func TestTm5_term51_index_output_content_never_marks_rows(t *testing.T) {
	for _, sc := range tm5TSemanticScenarios {
		func() {
			defer func() {
				if t.Failed() {
					t.Logf("scenario %+v", sc)
				}
			}()
			tm := tm5TSemanticSetup(t, sc)
			defer tm.Deinit()
			zigTry(t, tm.Index())
			for y := uint(0); y < 5; y++ {
				expectEqual(t, PageRowSemanticPromptNone, tm5TRowAt(t, tm, y).semantic_prompt)
			}
			expectEqual(t, PageCellSemanticContentOutput, tm.Screens.Active.Cursor.SemanticContent)
		}()
	}
}

// tm5TReverseIndexCase is term51TReverseIndexCase.
func tm5TReverseIndexCase(t *testing.T, top uint, bottom uint, left uint, right uint, cx uint, cy uint, bg bool) {
	t.Helper()
	defer func() {
		if t.Failed() {
			t.Logf("reverseIndex case region=(%d,%d,%d,%d) cursor=(%d,%d) bg=%v", top, bottom, left, right, cx, cy, bg)
		}
	}()
	tm := tm5Init(t, 5, 5)
	defer tm.Deinit()
	tm5TFill(t, tm)
	g := tm5TLabels()
	tm.SetCursorPos(cy+1, cx+1)
	tm5TRegion(tm, top, bottom, left, right)
	if bg {
		tm5Attr(t, tm, tm5TBg)
	}
	tm.Screens.Active.Cursor.PendingWrap = true
	// Zig-only: if (comptime build_options.kitty_graphics) t.screens.active.kitty_images.dirty = false;

	tm.ReverseIndex()

	ey := cy
	scrolled := false
	if cy != top || cx < left || cx > right {
		// moves up by one but never above the top margin when at or below it
		var max_ uint
		if cy >= top {
			max_ = cy - top
		} else {
			max_ = cy
		}
		if max_ >= 1 {
			ey = cy - 1
		}
	} else {
		tm5TModelScrollDown(&g, top, bottom, left, right, 1, tm5TBlank(bg))
		scrolled = true
	}

	tm5TExpectModel(t, tm, &g)
	tm5TExpectCursor(t, tm, cx, ey)
	tm5TExpectCursorConsistent(t, tm)
	// Scrolling restores the pending wrap state, moving clears it.
	expectEqual(t, scrolled, tm.Screens.Active.Cursor.PendingWrap)
	expectEqual(t, uint(0), tm5TScrollbackRows(tm))
	// Zig-only: if (comptime build_options.kitty_graphics) { try term51TTesting.expectEqual(scrolled, t.screens.active.kitty_images.dirty); }
	tm5TExpectRegion(t, tm, top, bottom, left, right)
}

// Zig: test "term51: reverseIndex: sweep of every region pair and cursor position against a model"
func TestTm5_term51_reverseIndex_sweep_of_every_region_pair_and_cursor_position_against_a_model(t *testing.T) {
	for _, tb := range tm5TPairs {
		for _, lr := range tm5TPairs {
			for cy := uint(0); cy < 5; cy++ {
				for cx := uint(0); cx < 5; cx++ {
					tm5TReverseIndexCase(t, tb[0], tb[1], lr[0], lr[1], cx, cy, false)
					tm5TReverseIndexCase(t, tb[0], tb[1], lr[0], lr[1], cx, cy, true)
				}
			}
		}
	}
}

// Zig: test "term51: reverseIndex: top of the region scrolls down"
func TestTm5_term51_reverseIndex_top_of_the_region_scrolls_down(t *testing.T) {
	tm := tm5Init(t, 5, 5)
	defer tm.Deinit()
	tm5TFill(t, tm)

	tm.SetTopAndBottomMargin(2, 4)
	tm.Screens.Active.CursorAbsolute(2, 1)
	tm.ReverseIndex()
	tm5TExpectCursor(t, tm, 2, 1)
	tm5TExpectGrid(t, tm, []string{"ABCDE", ".....", "FGHIJ", "KLMNO", "UVWXY"})
	expectEqual(t, uint(0), tm5TScrollbackRows(tm))
}

// Zig: test "term51: reverseIndex: top of the screen scrolls down without scrollback"
func TestTm5_term51_reverseIndex_top_of_the_screen_scrolls_down_without_scrollback(t *testing.T) {
	tm := tm5Init(t, 5, 5)
	defer tm.Deinit()
	tm5TFill(t, tm)

	tm.ReverseIndex()
	tm5TExpectCursor(t, tm, 0, 0)
	tm5TExpectGrid(t, tm, []string{".....", "ABCDE", "FGHIJ", "KLMNO", "PQRST"})
	expectEqual(t, uint(0), tm5TScrollbackRows(tm))
}

// Zig: test "term51: reverseIndex: moves up by one in every other position"
func TestTm5_term51_reverseIndex_moves_up_by_one_in_every_other_position(t *testing.T) {
	tm := tm5Init(t, 5, 5)
	defer tm.Deinit()
	tm5TFill(t, tm)
	original := []string{"ABCDE", "FGHIJ", "KLMNO", "PQRST", "UVWXY"}

	tm.SetTopAndBottomMargin(2, 4)

	// Below the top margin inside the region.
	tm.Screens.Active.CursorAbsolute(3, 3)
	tm.ReverseIndex()
	tm5TExpectCursor(t, tm, 3, 2)
	tm.ReverseIndex()
	tm5TExpectCursor(t, tm, 3, 1)

	// Below the bottom margin.
	tm.Screens.Active.CursorAbsolute(3, 4)
	tm.ReverseIndex()
	tm5TExpectCursor(t, tm, 3, 3)

	// Above the top margin: moves up, but not off the screen.
	tm.Screens.Active.CursorAbsolute(3, 1)
	tm.Screens.Active.CursorAbsolute(3, 0)
	tm.ReverseIndex()
	tm5TExpectCursor(t, tm, 3, 0)

	tm5TExpectGrid(t, tm, original)
}

// Zig: test "term51: reverseIndex: above the top margin moves up"
func TestTm5_term51_reverseIndex_above_the_top_margin_moves_up(t *testing.T) {
	tm := tm5Init(t, 5, 5)
	defer tm.Deinit()
	tm5TFill(t, tm)

	tm5TRegion(tm, 3, 4, 0, 4)
	tm.Screens.Active.CursorAbsolute(1, 2)
	tm.Screens.Active.Cursor.PendingWrap = true
	tm.ReverseIndex()
	tm5TExpectCursor(t, tm, 1, 1)
	expect(t, !tm.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
	tm.ReverseIndex()
	tm5TExpectCursor(t, tm, 1, 0)
	tm.ReverseIndex()
	tm5TExpectCursor(t, tm, 1, 0)
	tm5TExpectGrid(t, tm, []string{"ABCDE", "FGHIJ", "KLMNO", "PQRST", "UVWXY"})
}

// Zig: test "term51: reverseIndex: on the top margin outside the column margins does nothing"
func TestTm5_term51_reverseIndex_on_the_top_margin_outside_the_column_margins_does_nothing(t *testing.T) {
	tm := tm5Init(t, 5, 5)
	defer tm.Deinit()
	tm5TFill(t, tm)

	tm5TRegion(tm, 1, 3, 1, 3)
	for _, x := range []uint{0, 4} {
		tm.Screens.Active.CursorAbsolute(SizeCellCountInt(x), 1)
		tm.Screens.Active.Cursor.PendingWrap = true
		tm.ReverseIndex()
		tm5TExpectCursor(t, tm, x, 1)
		expect(t, !tm.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
		tm5TExpectGrid(t, tm, []string{"ABCDE", "FGHIJ", "KLMNO", "PQRST", "UVWXY"})
	}
}

// Zig: test "term51: reverseIndex: left and right margins scroll only their columns"
func TestTm5_term51_reverseIndex_left_and_right_margins_scroll_only_their_columns(t *testing.T) {
	tm := tm5Init(t, 5, 5)
	defer tm.Deinit()
	tm5TFill(t, tm)

	tm.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	tm.SetTopAndBottomMargin(2, 4)
	tm.SetLeftAndRightMargin(2, 4)
	tm.Screens.Active.CursorAbsolute(1, 1)
	tm.ReverseIndex()
	tm5TExpectCursor(t, tm, 1, 1)
	tm5TExpectGrid(t, tm, []string{"ABCDE", "F...J", "KGHIO", "PLMNT", "UVWXY"})
}

// Zig: test "term51: reverseIndex: scrolling keeps pending wrap, background fills the new row"
func TestTm5_term51_reverseIndex_scrolling_keeps_pending_wrap_background_fills_the_new_row(t *testing.T) {
	tm := tm5Init(t, 5, 5)
	defer tm.Deinit()
	tm5TFill(t, tm)

	tm.SetTopAndBottomMargin(2, 4)
	tm5Attr(t, tm, tm5TBg)
	tm.Screens.Active.CursorAbsolute(4, 1)
	tm.Screens.Active.Cursor.PendingWrap = true
	tm.ReverseIndex()
	expect(t, tm.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	tm5TExpectGrid(t, tm, []string{"ABCDE", "#####", "FGHIJ", "KLMNO", "UVWXY"})
	cell := tm5TCellAt(t, tm, 0, 1)
	expectEqual(t, PageCellContentTagBgColorRgb, cell.content_tag)
	expectEqual(t, PageCellRGB{R: 1, G: 2, B: 3}, cell.content.color_rgb)
}

// Zig: test "term51: setCursorPos: zero means one and the limits clamp"
func TestTm5_term51_setCursorPos_zero_means_one_and_the_limits_clamp(t *testing.T) {
	tm := tm5Init(t, 6, 4)
	defer tm.Deinit()

	type Case struct{ row, col, x, y uint }
	cases := []Case{
		{row: 0, col: 0, x: 0, y: 0},
		{row: 1, col: 1, x: 0, y: 0},
		{row: 0, col: 5, x: 4, y: 0},
		{row: 3, col: 0, x: 0, y: 2},
		{row: 2, col: 2, x: 1, y: 1},
		{row: 4, col: 6, x: 5, y: 3}, // exactly the size
		{row: 5, col: 7, x: 5, y: 3}, // one past
		{row: 3, col: 7, x: 5, y: 2},
		{row: 5, col: 3, x: 2, y: 3},
		{row: 1_000_000, col: 1_000_000, x: 5, y: 3},
		{row: math.MaxUint, col: math.MaxUint, x: 5, y: 3},
		{row: 1, col: math.MaxUint, x: 5, y: 0},
		{row: math.MaxUint, col: 1, x: 0, y: 3},
	}
	for _, c := range cases {
		tm.SetCursorPos(c.row, c.col)
		tm5TExpectCursor(t, tm, c.x, c.y)
		tm5TExpectCursorConsistent(t, tm)
	}
}

// Zig: test "term51: setCursorPos: resets pending wrap on every path"
func TestTm5_term51_setCursorPos_resets_pending_wrap_on_every_path(t *testing.T) {
	tm := tm5Init(t, 6, 4)
	defer tm.Deinit()

	// same position
	tm.SetCursorPos(2, 3)
	tm.Screens.Active.Cursor.PendingWrap = true
	tm.SetCursorPos(2, 3)
	expect(t, !tm.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
	// same row, moves right
	tm.Screens.Active.Cursor.PendingWrap = true
	tm.SetCursorPos(2, 6)
	expect(t, !tm.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
	// same row, moves left
	tm.Screens.Active.Cursor.PendingWrap = true
	tm.SetCursorPos(2, 1)
	expect(t, !tm.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
	// different row
	tm.Screens.Active.Cursor.PendingWrap = true
	tm.SetCursorPos(4, 1)
	expect(t, !tm.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
	// printing in the last cell sets it, a move clears it
	tm.SetCursorPos(4, 6)
	zigTry(t, tm.Print('x'))
	expect(t, tm.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	tm.SetCursorPos(4, 6)
	expect(t, !tm.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
	tm5TExpectCursor(t, tm, 5, 3)
}

// Zig: test "term51: setCursorPos: printing after a move writes at the new cell"
func TestTm5_term51_setCursorPos_printing_after_a_move_writes_at_the_new_cell(t *testing.T) {
	tm := tm5Init(t, 4, 3)
	defer tm.Deinit()

	tm.SetCursorPos(2, 3)
	zigTry(t, tm.Print('a'))
	tm.SetCursorPos(2, 1) // same row, left
	zigTry(t, tm.Print('b'))
	tm.SetCursorPos(2, 4) // same row, right
	zigTry(t, tm.Print('c'))
	tm.SetCursorPos(3, 2) // other row
	zigTry(t, tm.Print('d'))
	tm.SetCursorPos(1, 1)
	zigTry(t, tm.Print('e'))
	tm5TExpectGrid(t, tm, []string{"e...", "b.ac", ".d.."})
}

// Zig: test "term51: setCursorPos: origin mode is relative to the margins and clamps inside them"
func TestTm5_term51_setCursorPos_origin_mode_is_relative_to_the_margins_and_clamps_inside_them(t *testing.T) {
	tm := tm5Init(t, 8, 6)
	defer tm.Deinit()

	tm.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	tm.SetTopAndBottomMargin(2, 5) // rows 1..4
	tm.SetLeftAndRightMargin(3, 6) // columns 2..5
	tm.Modes.Set(ModesModeOrigin, true)

	type Case struct{ row, col, x, y uint }
	cases := []Case{
		{row: 0, col: 0, x: 2, y: 1},
		{row: 1, col: 1, x: 2, y: 1},
		{row: 2, col: 3, x: 4, y: 2},
		{row: 4, col: 4, x: 5, y: 4}, // last row/column of the region
		{row: 5, col: 5, x: 5, y: 4}, // one past the region
		{row: 100, col: 100, x: 5, y: 4},
		{row: 1, col: 100, x: 5, y: 1},
		{row: 100, col: 1, x: 2, y: 4},
		{row: 0, col: 4, x: 5, y: 1},
		{row: 3, col: 0, x: 2, y: 3},
	}
	for _, c := range cases {
		tm.SetCursorPos(c.row, c.col)
		tm5TExpectCursor(t, tm, c.x, c.y)
		tm5TExpectCursorConsistent(t, tm)
	}

	// Origin mode off: absolute again, margins ignored.
	tm.Modes.Set(ModesModeOrigin, false)
	tm.SetCursorPos(1, 1)
	tm5TExpectCursor(t, tm, 0, 0)
	tm.SetCursorPos(6, 8)
	tm5TExpectCursor(t, tm, 7, 5)
}

// Zig: test "term51: setCursorPos: origin mode with only top and bottom margins"
func TestTm5_term51_setCursorPos_origin_mode_with_only_top_and_bottom_margins(t *testing.T) {
	tm := tm5Init(t, 4, 5)
	defer tm.Deinit()

	tm.SetTopAndBottomMargin(3, 4) // rows 2..3
	tm.Modes.Set(ModesModeOrigin, true)
	tm.SetCursorPos(1, 4)
	tm5TExpectCursor(t, tm, 3, 2)
	tm.SetCursorPos(2, 9)
	tm5TExpectCursor(t, tm, 3, 3)
	tm.SetCursorPos(3, 1)
	tm5TExpectCursor(t, tm, 0, 3)
}

// Zig: test "term51: setCursorPos: minimal two-cell regions and regions touching the edges"
func TestTm5_term51_setCursorPos_minimal_two_cell_regions_and_regions_touching_the_edges(t *testing.T) {
	tm := tm5Init(t, 5, 5)
	defer tm.Deinit()
	tm.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	tm.Modes.Set(ModesModeOrigin, true)

	// Region in the bottom-right corner, 2x2.
	tm.SetTopAndBottomMargin(4, 5)
	tm.SetLeftAndRightMargin(4, 5)
	tm.SetCursorPos(1, 1)
	tm5TExpectCursor(t, tm, 3, 3)
	tm.SetCursorPos(2, 2)
	tm5TExpectCursor(t, tm, 4, 4)
	tm.SetCursorPos(3, 3)
	tm5TExpectCursor(t, tm, 4, 4)

	// Region in the top-left corner, 2x2.
	tm.SetTopAndBottomMargin(1, 2)
	tm.SetLeftAndRightMargin(1, 2)
	tm.SetCursorPos(1, 1)
	tm5TExpectCursor(t, tm, 0, 0)
	tm.SetCursorPos(9, 9)
	tm5TExpectCursor(t, tm, 1, 1)
}

// Zig: test "term51: setCursorPos: sweep of requests, starts, origin mode and margins against a model"
func TestTm5_term51_setCursorPos_sweep_of_requests_starts_origin_mode_and_margins_against_a_model(t *testing.T) {
	reqs := []uint{0, 1, 2, 3, 4, 5, 6, 1000}
	regions := [][4]uint{
		{0, 4, 0, 4}, {1, 3, 1, 3}, {0, 1, 3, 4},
		{3, 4, 0, 1}, {2, 3, 2, 3}, {1, 4, 0, 2},
	}
	tm := tm5Init(t, 5, 5)
	defer tm.Deinit()

	for _, reg := range regions {
		tm5TRegion(tm, reg[0], reg[1], reg[2], reg[3])
		for _, origin := range []bool{false, true} {
			tm.Modes.Set(ModesModeOrigin, origin)
			var x_off, y_off uint
			x_max, y_max := uint(5), uint(5)
			if origin {
				x_off = reg[2]
				y_off = reg[0]
				x_max = reg[3] + 1
				y_max = reg[1] + 1
			}
			for sy := uint(0); sy < 5; sy++ {
				for sx := uint(0); sx < 5; sx++ {
					for _, row := range reqs {
						for _, col := range reqs {
							tm.Screens.Active.CursorAbsolute(SizeCellCountInt(sx), SizeCellCountInt(sy))
							tm.Screens.Active.Cursor.PendingWrap = true
							tm.SetCursorPos(row, col)
							ex := min(x_max, max(col, 1)+x_off) - 1
							ey := min(y_max, max(row, 1)+y_off) - 1
							func() {
								defer func() {
									if t.Failed() {
										t.Logf("setCursorPos region=%v origin=%v start=(%d,%d) req=(%d,%d)", reg, origin, sx, sy, row, col)
									}
								}()
								tm5TExpectCursor(t, tm, ex, ey)
								tm5TExpectCursorConsistent(t, tm)
								expect(t, !tm.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
							}()
						}
					}
				}
			}
		}
	}
}

// tm5Pair is `.{ a, b }` for an optional [2]usize.
func tm5Pair(a uint, b uint) *[2]uint {
	return &[2]uint{a, b}
}

// Zig: test "term51: setTopAndBottomMargin: table of requests on a 5-row screen"
func TestTm5_term51_setTopAndBottomMargin_table_of_requests_on_a_5_row_screen(t *testing.T) {
	type Case struct {
		top, bottom uint
		expect      *[2]uint
	}
	cases := []Case{
		{top: 0, bottom: 0, expect: tm5Pair(0, 4)},
		{top: 1, bottom: 0, expect: tm5Pair(0, 4)},
		{top: 0, bottom: 5, expect: tm5Pair(0, 4)},
		{top: 1, bottom: 5, expect: tm5Pair(0, 4)},
		{top: 0, bottom: 6, expect: tm5Pair(0, 4)},
		{top: 1, bottom: 1000, expect: tm5Pair(0, 4)},
		{top: 0, bottom: math.MaxUint, expect: tm5Pair(0, 4)},
		{top: 2, bottom: 4, expect: tm5Pair(1, 3)},
		{top: 2, bottom: 0, expect: tm5Pair(1, 4)},
		{top: 4, bottom: 5, expect: tm5Pair(3, 4)},
		{top: 4, bottom: 0, expect: tm5Pair(3, 4)},
		{top: 4, bottom: 6, expect: tm5Pair(3, 4)},
		{top: 1, bottom: 2, expect: tm5Pair(0, 1)},
		{top: 0, bottom: 2, expect: tm5Pair(0, 1)},
		{top: 3, bottom: 4, expect: tm5Pair(2, 3)},
		// top >= bottom: nothing changes
		{top: 1, bottom: 1, expect: nil},
		{top: 0, bottom: 1, expect: nil},
		{top: 2, bottom: 2, expect: nil},
		{top: 3, bottom: 2, expect: nil},
		{top: 5, bottom: 5, expect: nil},
		{top: 5, bottom: 0, expect: nil},
		{top: 5, bottom: 6, expect: nil},
		{top: 6, bottom: 0, expect: nil},
		{top: 6, bottom: 7, expect: nil},
		{top: 1000, bottom: 1000, expect: nil},
		{top: math.MaxUint, bottom: 0, expect: nil},
		{top: math.MaxUint, bottom: math.MaxUint, expect: nil},
	}
	for _, c := range cases {
		func() {
			tm := tm5Init(t, 4, 5)
			defer tm.Deinit()
			// A distinctive earlier region and cursor to see what is left alone.
			tm5TRegion(tm, 1, 3, 1, 2)
			tm.Screens.Active.CursorAbsolute(3, 4)
			tm.Screens.Active.Cursor.PendingWrap = true

			tm.SetTopAndBottomMargin(c.top, c.bottom)
			defer func() {
				if t.Failed() {
					t.Logf("setTopAndBottomMargin(%d, %d)", c.top, c.bottom)
				}
			}()
			if c.expect != nil {
				e := *c.expect
				tm5TExpectRegion(t, tm, e[0], e[1], 1, 2)
				tm5TExpectCursor(t, tm, 0, 0)
				expect(t, !tm.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
			} else {
				tm5TExpectRegion(t, tm, 1, 3, 1, 2)
				tm5TExpectCursor(t, tm, 3, 4)
				expect(t, tm.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
			}
		}()
	}
}

// Zig: test "term51: setTopAndBottomMargin: very small screens"
func TestTm5_term51_setTopAndBottomMargin_very_small_screens(t *testing.T) {
	// One row: nothing is ever valid.
	func() {
		tm := tm5Init(t, 3, 1)
		defer tm.Deinit()
		for _, c := range [][2]uint{{0, 0}, {1, 1}, {1, 2}, {0, 5}} {
			tm.SetTopAndBottomMargin(c[0], c[1])
			tm5TExpectRegion(t, tm, 0, 0, 0, 2)
		}
	}()
	// Two rows: only the whole screen is valid.
	func() {
		tm := tm5Init(t, 3, 2)
		defer tm.Deinit()
		tm.SetTopAndBottomMargin(2, 2)
		tm5TExpectRegion(t, tm, 0, 1, 0, 2)
		tm.SetTopAndBottomMargin(1, 2)
		tm5TExpectRegion(t, tm, 0, 1, 0, 2)
		tm.SetTopAndBottomMargin(2, 0)
		tm5TExpectRegion(t, tm, 0, 1, 0, 2)
		tm.SetTopAndBottomMargin(0, 0)
		tm5TExpectRegion(t, tm, 0, 1, 0, 2)
	}()
	// Three rows: smallest regions at each end and in the middle.
	func() {
		tm := tm5Init(t, 3, 3)
		defer tm.Deinit()
		tm.SetTopAndBottomMargin(1, 2)
		tm5TExpectRegion(t, tm, 0, 1, 0, 2)
		tm.SetTopAndBottomMargin(2, 3)
		tm5TExpectRegion(t, tm, 1, 2, 0, 2)
		tm.SetTopAndBottomMargin(2, 2)
		tm5TExpectRegion(t, tm, 1, 2, 0, 2)
	}()
}

// Zig: test "term51: setTopAndBottomMargin: cursor homes relative to the region in origin mode"
func TestTm5_term51_setTopAndBottomMargin_cursor_homes_relative_to_the_region_in_origin_mode(t *testing.T) {
	tm := tm5Init(t, 6, 5)
	defer tm.Deinit()

	tm.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	tm.SetLeftAndRightMargin(3, 5)
	tm.Screens.Active.CursorAbsolute(5, 4)

	// Not in origin mode: home is absolute (0, 0).
	tm.SetTopAndBottomMargin(2, 4)
	tm5TExpectRegion(t, tm, 1, 3, 2, 4)
	tm5TExpectCursor(t, tm, 0, 0)

	// In origin mode: top-left of the region.
	tm.Modes.Set(ModesModeOrigin, true)
	tm.Screens.Active.CursorAbsolute(5, 4)
	tm.SetTopAndBottomMargin(3, 5)
	tm5TExpectRegion(t, tm, 2, 4, 2, 4)
	tm5TExpectCursor(t, tm, 2, 2)

	// An invalid request in origin mode leaves cursor and region alone.
	tm.Screens.Active.CursorAbsolute(5, 4)
	tm.SetTopAndBottomMargin(4, 3)
	tm5TExpectRegion(t, tm, 2, 4, 2, 4)
	tm5TExpectCursor(t, tm, 5, 4)
}

// Zig: test "term51: setTopAndBottomMargin: does not touch the cell contents"
func TestTm5_term51_setTopAndBottomMargin_does_not_touch_the_cell_contents(t *testing.T) {
	tm := tm5Init(t, 5, 5)
	defer tm.Deinit()
	tm5TFill(t, tm)
	tm.SetTopAndBottomMargin(2, 4)
	tm.SetTopAndBottomMargin(0, 0)
	tm5TExpectGrid(t, tm, []string{"ABCDE", "FGHIJ", "KLMNO", "PQRST", "UVWXY"})
}

// Zig: test "term51: setLeftAndRightMargin: ignored while mode 69 is off"
func TestTm5_term51_setLeftAndRightMargin_ignored_while_mode_69_is_off(t *testing.T) {
	tm := tm5Init(t, 5, 4)
	defer tm.Deinit()

	tm.Screens.Active.CursorAbsolute(3, 2)
	tm.Screens.Active.Cursor.PendingWrap = true
	for _, c := range [][2]uint{{2, 4}, {0, 0}, {1, 5}, {3, 0}} {
		tm.SetLeftAndRightMargin(c[0], c[1])
		tm5TExpectRegion(t, tm, 0, 3, 0, 4)
		tm5TExpectCursor(t, tm, 3, 2)
		expect(t, tm.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	}
}

// Zig: test "term51: setLeftAndRightMargin: table of requests on a 5-column screen"
func TestTm5_term51_setLeftAndRightMargin_table_of_requests_on_a_5_column_screen(t *testing.T) {
	type Case struct {
		left, right uint
		expect      *[2]uint
	}
	cases := []Case{
		{left: 0, right: 0, expect: tm5Pair(0, 4)},
		{left: 1, right: 0, expect: tm5Pair(0, 4)},
		{left: 0, right: 5, expect: tm5Pair(0, 4)},
		{left: 1, right: 5, expect: tm5Pair(0, 4)},
		{left: 0, right: 6, expect: tm5Pair(0, 4)},
		{left: 1, right: 1000, expect: tm5Pair(0, 4)},
		{left: 0, right: math.MaxUint, expect: tm5Pair(0, 4)},
		{left: 2, right: 4, expect: tm5Pair(1, 3)},
		{left: 2, right: 0, expect: tm5Pair(1, 4)},
		{left: 4, right: 5, expect: tm5Pair(3, 4)},
		{left: 4, right: 0, expect: tm5Pair(3, 4)},
		{left: 4, right: 6, expect: tm5Pair(3, 4)},
		{left: 1, right: 2, expect: tm5Pair(0, 1)},
		{left: 0, right: 2, expect: tm5Pair(0, 1)},
		{left: 3, right: 4, expect: tm5Pair(2, 3)},
		// left >= right: nothing changes
		{left: 1, right: 1, expect: nil},
		{left: 0, right: 1, expect: nil},
		{left: 2, right: 2, expect: nil},
		{left: 3, right: 2, expect: nil},
		{left: 5, right: 5, expect: nil},
		{left: 5, right: 0, expect: nil},
		{left: 5, right: 6, expect: nil},
		{left: 6, right: 0, expect: nil},
		{left: 1000, right: 1000, expect: nil},
		{left: math.MaxUint, right: 0, expect: nil},
		{left: math.MaxUint, right: math.MaxUint, expect: nil},
	}
	for _, c := range cases {
		func() {
			tm := tm5Init(t, 5, 4)
			defer tm.Deinit()
			tm.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
			tm5TRegion(tm, 1, 2, 1, 3)
			tm.Screens.Active.CursorAbsolute(4, 3)
			tm.Screens.Active.Cursor.PendingWrap = true

			tm.SetLeftAndRightMargin(c.left, c.right)
			defer func() {
				if t.Failed() {
					t.Logf("setLeftAndRightMargin(%d, %d)", c.left, c.right)
				}
			}()
			if c.expect != nil {
				e := *c.expect
				tm5TExpectRegion(t, tm, 1, 2, e[0], e[1])
				tm5TExpectCursor(t, tm, 0, 0)
				expect(t, !tm.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
			} else {
				tm5TExpectRegion(t, tm, 1, 2, 1, 3)
				tm5TExpectCursor(t, tm, 4, 3)
				expect(t, tm.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
			}
		}()
	}
}

// Zig: test "term51: setLeftAndRightMargin: very small screens"
func TestTm5_term51_setLeftAndRightMargin_very_small_screens(t *testing.T) {
	// One column: nothing is ever valid.
	func() {
		tm := tm5Init(t, 1, 3)
		defer tm.Deinit()
		tm.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
		for _, c := range [][2]uint{{0, 0}, {1, 1}, {1, 2}, {0, 5}} {
			tm.SetLeftAndRightMargin(c[0], c[1])
			tm5TExpectRegion(t, tm, 0, 2, 0, 0)
		}
	}()
	// Two columns: only the whole width is valid.
	func() {
		tm := tm5Init(t, 2, 3)
		defer tm.Deinit()
		tm.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
		tm.SetLeftAndRightMargin(2, 2)
		tm5TExpectRegion(t, tm, 0, 2, 0, 1)
		tm.SetLeftAndRightMargin(1, 2)
		tm5TExpectRegion(t, tm, 0, 2, 0, 1)
		tm.SetLeftAndRightMargin(2, 0)
		tm5TExpectRegion(t, tm, 0, 2, 0, 1)
	}()
	// Three columns.
	func() {
		tm := tm5Init(t, 3, 3)
		defer tm.Deinit()
		tm.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
		tm.SetLeftAndRightMargin(1, 2)
		tm5TExpectRegion(t, tm, 0, 2, 0, 1)
		tm.SetLeftAndRightMargin(2, 3)
		tm5TExpectRegion(t, tm, 0, 2, 1, 2)
		tm.SetLeftAndRightMargin(2, 2)
		tm5TExpectRegion(t, tm, 0, 2, 1, 2)
	}()
}

// Zig: test "term51: setLeftAndRightMargin: cursor homes relative to the region in origin mode"
func TestTm5_term51_setLeftAndRightMargin_cursor_homes_relative_to_the_region_in_origin_mode(t *testing.T) {
	tm := tm5Init(t, 6, 5)
	defer tm.Deinit()

	tm.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	tm.SetTopAndBottomMargin(2, 4)
	tm.Screens.Active.CursorAbsolute(5, 4)

	tm.SetLeftAndRightMargin(2, 5)
	tm5TExpectRegion(t, tm, 1, 3, 1, 4)
	tm5TExpectCursor(t, tm, 0, 0)

	tm.Modes.Set(ModesModeOrigin, true)
	tm.Screens.Active.CursorAbsolute(5, 4)
	tm.SetLeftAndRightMargin(3, 6)
	tm5TExpectRegion(t, tm, 1, 3, 2, 5)
	tm5TExpectCursor(t, tm, 2, 1)

	// Invalid request in origin mode: nothing changes.
	tm.Screens.Active.CursorAbsolute(5, 4)
	tm.SetLeftAndRightMargin(5, 4)
	tm5TExpectRegion(t, tm, 1, 3, 2, 5)
	tm5TExpectCursor(t, tm, 5, 4)
}

// Zig: test "term51: setLeftAndRightMargin: sweep against a model"
func TestTm5_term51_setLeftAndRightMargin_sweep_against_a_model(t *testing.T) {
	reqs := []uint{0, 1, 2, 3, 4, 5, 6, 7, 1000, math.MaxUint}
	for _, cols := range []uint{1, 2, 3, 5, 6} {
		for _, mode := range []bool{false, true} {
			for _, origin := range []bool{false, true} {
				func() {
					tm := tm5Init(t, SizeCellCountInt(cols), 4)
					defer tm.Deinit()
					tm.Modes.Set(ModesModeEnableLeftAndRightMargin, mode)
					tm.Modes.Set(ModesModeOrigin, origin)
					for _, l := range reqs {
						for _, r := range reqs {
							var prev_left uint
							if cols >= 3 {
								prev_left = 1
							}
							tm5TRegion(tm, 1, 2, prev_left, cols-1)
							tm.Screens.Active.CursorAbsolute(SizeCellCountInt(cols-1), 3)
							tm.Screens.Active.Cursor.PendingWrap = true
							tm.SetLeftAndRightMargin(l, r)

							left := max(1, l)
							var right_req uint
							if r == 0 {
								right_req = cols
							} else {
								right_req = r
							}
							right := min(cols, right_req)
							func() {
								defer func() {
									if t.Failed() {
										t.Logf("setLeftAndRightMargin cols=%d mode=%v origin=%v (%d, %d)", cols, mode, origin, l, r)
									}
								}()
								if mode && left < right {
									tm5TExpectRegion(t, tm, 1, 2, left-1, right-1)
									// origin: x = left margin, y = top margin
									var ex, ey uint
									if origin {
										ex = left - 1
										ey = 1
									}
									tm5TExpectCursor(t, tm, ex, ey)
									expect(t, !tm.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
								} else {
									tm5TExpectRegion(t, tm, 1, 2, prev_left, cols-1)
									tm5TExpectCursor(t, tm, cols-1, 3)
									expect(t, tm.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
								}
							}()
						}
					}
				}()
			}
		}
	}
}

// Zig: test "term51: setTopAndBottomMargin: sweep against a model"
func TestTm5_term51_setTopAndBottomMargin_sweep_against_a_model(t *testing.T) {
	reqs := []uint{0, 1, 2, 3, 4, 5, 6, 7, 1000, math.MaxUint}
	for _, rows := range []uint{1, 2, 3, 5, 6} {
		for _, origin := range []bool{false, true} {
			func() {
				tm := tm5Init(t, 6, SizeCellCountInt(rows))
				defer tm.Deinit()
				tm.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
				tm.SetLeftAndRightMargin(2, 5)
				tm.Modes.Set(ModesModeOrigin, origin)
				for _, tp := range reqs {
					for _, bt := range reqs {
						var prev_top uint
						if rows >= 3 {
							prev_top = 1
						}
						tm5TRegion(tm, prev_top, rows-1, 1, 4)
						tm.Screens.Active.CursorAbsolute(5, SizeCellCountInt(rows-1))
						tm.Screens.Active.Cursor.PendingWrap = true
						tm.SetTopAndBottomMargin(tp, bt)

						top := max(1, tp)
						var bottom_req uint
						if bt == 0 {
							bottom_req = rows
						} else {
							bottom_req = bt
						}
						bottom := min(rows, bottom_req)
						func() {
							defer func() {
								if t.Failed() {
									t.Logf("setTopAndBottomMargin rows=%d origin=%v (%d, %d)", rows, origin, tp, bt)
								}
							}()
							if top < bottom {
								tm5TExpectRegion(t, tm, top-1, bottom-1, 1, 4)
								var ex, ey uint
								if origin {
									ex = 1
									ey = top - 1
								}
								tm5TExpectCursor(t, tm, ex, ey)
								expect(t, !tm.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
							} else {
								tm5TExpectRegion(t, tm, prev_top, rows-1, 1, 4)
								tm5TExpectCursor(t, tm, 5, rows-1)
								expect(t, tm.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
							}
						}()
					}
				}
			}()
		}
	}
}

// tm5TScrollCase is term51TScrollCase.
func tm5TScrollCase(t *testing.T, up bool, top uint, bottom uint, left uint, right uint, count uint, cx uint, cy uint, bg bool) {
	t.Helper()
	defer func() {
		if t.Failed() {
			t.Logf("scroll case up=%v region=(%d,%d,%d,%d) count=%d cursor=(%d,%d) bg=%v", up, top, bottom, left, right, count, cx, cy, bg)
		}
	}()
	tm := tm5Init(t, 5, 5)
	defer tm.Deinit()
	tm5TFill(t, tm)
	g := tm5TLabels()
	tm.SetCursorPos(cy+1, cx+1)
	tm5TRegion(tm, top, bottom, left, right)
	if bg {
		tm5Attr(t, tm, tm5TBg)
	}
	tm.Screens.Active.Cursor.PendingWrap = true
	// Zig-only: if (comptime build_options.kitty_graphics) t.screens.active.kitty_images.dirty = false;

	blank := tm5TBlank(bg)
	full := top == 0 && left == 0 && right == 4
	if up {
		zigTry(t, tm.ScrollUp(count))
		tm5TModelScrollUp(&g, top, bottom, left, right, count, blank)
		var want uint
		if full {
			want = min(count, bottom+1)
		}
		expectEqual(t, want, tm5TScrollbackRows(tm))
	} else {
		tm.ScrollDown(count)
		tm5TModelScrollDown(&g, top, bottom, left, right, count, blank)
		expectEqual(t, uint(0), tm5TScrollbackRows(tm))
	}

	tm5TExpectModel(t, tm, &g)
	tm5TExpectCursor(t, tm, cx, cy)
	tm5TExpectCursorConsistent(t, tm)
	expect(t, tm.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	tm5TExpectRegion(t, tm, top, bottom, left, right)
	// Zig-only: if (comptime build_options.kitty_graphics) { // scrollUp on a full-width region at the top marks images dirty even for a count of zero; every other path only does when it scrolls. const expected = count > 0 or (up and full); try term51TTesting.expectEqual(expected, t.screens.active.kitty_images.dirty); }
}

// Zig: test "term51: scrollUp: sweep of every region pair, count, cursor and background against a model"
func TestTm5_term51_scrollUp_sweep_of_every_region_pair_count_cursor_and_background_against_a_model(t *testing.T) {
	counts := []uint{0, 1, 2, 3, 4, 5, 6, 100}
	cursors := [][2]uint{{0, 0}, {4, 4}, {2, 2}, {1, 3}}
	for _, tb := range tm5TPairs {
		for _, lr := range tm5TPairs {
			for _, count := range counts {
				for _, cur := range cursors {
					tm5TScrollCase(t, true, tb[0], tb[1], lr[0], lr[1], count, cur[0], cur[1], false)
					tm5TScrollCase(t, true, tb[0], tb[1], lr[0], lr[1], count, cur[0], cur[1], true)
				}
			}
		}
	}
}

// Zig: test "term51: scrollDown: sweep of every region pair, count, cursor and background against a model"
func TestTm5_term51_scrollDown_sweep_of_every_region_pair_count_cursor_and_background_against_a_model(t *testing.T) {
	counts := []uint{0, 1, 2, 3, 4, 5, 6, 100}
	cursors := [][2]uint{{0, 0}, {4, 4}, {2, 2}, {1, 3}}
	for _, tb := range tm5TPairs {
		for _, lr := range tm5TPairs {
			for _, count := range counts {
				for _, cur := range cursors {
					tm5TScrollCase(t, false, tb[0], tb[1], lr[0], lr[1], count, cur[0], cur[1], false)
					tm5TScrollCase(t, false, tb[0], tb[1], lr[0], lr[1], count, cur[0], cur[1], true)
				}
			}
		}
	}
}

// Go-only: panic test: index: semantic_content is output while semantic_content_clear_eol is set
func TestTm5_panic_index_output_content_with_clear_eol_set(t *testing.T) {
	// Terminal.index: `assert(!screen.cursor.semantic_content_clear_eol)` in
	// the output branch of the deferred semantic content handling.
	tm := tm5Init(t, 5, 3)
	defer tm.Deinit()
	tm.Screens.Active.Cursor.SemanticContent = PageCellSemanticContentOutput
	tm.Screens.Active.Cursor.SemanticContentClearEol = true
	tm5ExpectPanic(t, func() { _ = tm.Index() })
}

// Go-only: panic test: setCursorPos: col_req + x_offset overflows usize in origin mode
func TestTm5_panic_setCursorPos_column_overflow_in_origin_mode(t *testing.T) {
	// Terminal.setCursorPos: `col + params.x_offset` is a checked usize
	// addition; the Zig tests only use maxInt(usize) without an offset.
	tm := tm5Init(t, 10, 5)
	defer tm.Deinit()
	tm.ScrollingRegion.Left = 1
	tm.Modes.Set(ModesModeOrigin, true)
	// The addition is in 64-bit usize (the reference) on every platform. A
	// Go uint is 32 bits on 32-bit ARM, where maxInt(uint) + 1 fits a
	// usize: no trap, the column clamps to the margin as on the reference.
	if bits.UintSize == 32 {
		tm.SetCursorPos(1, math.MaxUint)
		expectEqual(t, SizeCellCountInt(tm.ScrollingRegion.Right), tm.Screens.Active.Cursor.X)
		return
	}
	tm5ExpectPanic(t, func() { tm.SetCursorPos(1, math.MaxUint) })
}

// Go-only: panic test: setCursorPos: row_req + y_offset overflows usize in origin mode
func TestTm5_panic_setCursorPos_row_overflow_in_origin_mode(t *testing.T) {
	// Terminal.setCursorPos: `row + params.y_offset` is a checked usize
	// addition; the Zig tests only use maxInt(usize) without an offset.
	tm := tm5Init(t, 10, 5)
	defer tm.Deinit()
	tm.ScrollingRegion.Top = 1
	tm.Modes.Set(ModesModeOrigin, true)
	// As above: no trap where a Go uint is 32 bits; the row clamps.
	if bits.UintSize == 32 {
		tm.SetCursorPos(math.MaxUint, 1)
		expectEqual(t, SizeCellCountInt(tm.ScrollingRegion.Bottom), tm.Screens.Active.Cursor.Y)
		return
	}
	tm5ExpectPanic(t, func() { tm.SetCursorPos(math.MaxUint, 1) })
}
