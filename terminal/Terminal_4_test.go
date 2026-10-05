// Tests translated one for one from src/terminal/Terminal.zig (Ghostty
// v1.3.1 plus the term51 additional tests), lines 13635 to 16106 (the tests
// from `term51: printRepeat: huge count ...` to `term51: cursorRight:
// stops at the right margin when at or left of it`). Each Go test carries
// the exact Zig title in a "// Zig: test" comment. Helpers are prefixed tm4.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual keeps Zig's (expected, actual) argument order. `try` on a
//     call that returns an error is zigTry(t, err); `x.?` is zigUnwrap.
//   - The Zig terminal variable `t` is `term` in Go (`t` is the *testing.T);
//     `var t = try init(alloc, .{ .cols = c, .rows = r })` is
//     `term := tm4Init(t, terminalOptionsDefault(c, r))`, which writes out
//     the Zig defaults (max_scrollback 10_000, colors, default_modes).
//   - Every test starts with testStdCapacity(t) (grapheme_bytes 512 as in
//     Zig's unit tests).
//   - `@as(?u21, x)` / `@as(?charsets.Slots, x)` comparisons are
//     tm4ExpectOptional on Optional values.
//   - Zig's `alloc` arguments are dropped; `testing.allocator` leak checks
//     (`defer t.deinit(std.testing.allocator)`) become `defer term.Deinit()`.
//   - `page.styles.get(page.memory, id)`, `.refCount(page.memory, id)` and
//     `hyperlink_set.get(memory, id)` lose the memory argument.
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt.

package terminal

import (
	"math"
	"testing"
)

// tm4ExpectPanic runs f, which must panic. A panic with the wireframe's
// "not implemented" value does not count: it only means the function under
// test does not exist yet. The setup is done outside f.
func tm4ExpectPanic(t *testing.T, f func()) {
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

// tm4Init is `try Terminal.init(alloc, opts)`; the terminal is returned by
// pointer (the methods take a pointer receiver).
func tm4Init(t *testing.T, opts TerminalOptions) *Terminal {
	t.Helper()
	term, err := TerminalInit(opts)
	zigTry(t, err)
	return &term
}

// tm4ExpectOptional is `try testing.expectEqual(@as(?T, want), got)`.
func tm4ExpectOptional[T comparable](t *testing.T, want Optional[T], got Optional[T]) {
	t.Helper()
	if !want.Valid {
		if got.Valid {
			t.Fatalf("got %v, want null", got.Value)
		}
		return
	}
	if !got.Valid {
		t.Fatalf("got null, want %v", want.Value)
	}
	if got.Value != want.Value {
		t.Fatalf("got %v, want %v", got.Value, want.Value)
	}
}

// tm4Primary is `t.screens.get(.primary).?`: Get returns nil for null.
func tm4Primary(t *testing.T, term *Terminal) *Screen {
	t.Helper()
	s := term.screens.Get(ScreenSetKeyPrimary)
	if s == nil {
		t.Fatalf("unwrapped a null optional")
	}
	return s
}

// tm4SAt is term51SAt: the page list cell at the given active-area position.
func tm4SAt(t *testing.T, term *Terminal, x uint, y uint) PageListCell {
	t.Helper()
	return zigUnwrap(t, term.screens.active.pages.GetCell(Point{
		Tag: PointTagActive,
		Active: PointCoordinate{
			x: SizeCellCountInt(x),
			y: uint32(y),
		},
	}))
}

// tm4SExpectCell is term51SExpectCell: expect the cell at (x, y) to hold
// exactly the codepoint and wide state.
func tm4SExpectCell(t *testing.T, term *Terminal, x uint, y uint, cp uint32, wide PageCellWide) {
	t.Helper()
	lc := tm4SAt(t, term, x, y)
	expectEqual(t, cp, lc.cell.Codepoint())
	expectEqual(t, wide, lc.cell.wide)
}

// tm4SExpectGrapheme is term51SExpectGrapheme: expect the cell at (x, y) to
// carry exactly this grapheme tail.
func tm4SExpectGrapheme(t *testing.T, term *Terminal, x uint, y uint, expected []uint32) {
	t.Helper()
	lc := tm4SAt(t, term, x, y)
	expect(t, lc.cell.HasGrapheme(), "lc.cell.hasGrapheme()")
	expect(t, lc.row.grapheme, "lc.row.grapheme")
	expectEqualSlices(t, expected, zigUnwrap(t, lc.node.data.LookupGrapheme(lc.cell)))
}

// tm4SExpectNoGrapheme is term51SExpectNoGrapheme: expect the cell at
// (x, y) to carry no grapheme data.
func tm4SExpectNoGrapheme(t *testing.T, term *Terminal, x uint, y uint) {
	t.Helper()
	lc := tm4SAt(t, term, x, y)
	expect(t, !lc.cell.HasGrapheme(), "!lc.cell.hasGrapheme()")
	expect(t, !lc.node.data.LookupGrapheme(lc.cell).Valid, "lc.node.data.lookupGrapheme(lc.cell) == null")
}

// tm4SExpectCursor is term51SExpectCursor: expect the cursor position and
// pending wrap state.
func tm4SExpectCursor(t *testing.T, term *Terminal, x uint, y uint, pending_wrap bool) {
	t.Helper()
	expectEqual(t, x, uint(term.screens.active.cursor.x))
	expectEqual(t, y, uint(term.screens.active.cursor.y))
	expectEqual(t, pending_wrap, term.screens.active.cursor.pending_wrap)
}

// tm4SExpectPlain is term51SExpectPlain: expect the plain string of the
// viewport.
func tm4SExpectPlain(t *testing.T, term *Terminal, expected string) {
	t.Helper()
	str, err := term.PlainString()
	zigTry(t, err)
	expectEqualStrings(t, expected, string(str))
}

// tm4SPrintAll is term51SPrintAll: print every codepoint.
func tm4SPrintAll(t *testing.T, term *Terminal, cps []uint32) {
	t.Helper()
	for _, cp := range cps {
		zigTry(t, term.Print(cp))
	}
}

// tm4SExpectRowDirty is term51SExpectRowDirty: expect the row at y to be
// marked dirty (or not).
func tm4SExpectRowDirty(t *testing.T, term *Terminal, y uint, dirty bool) {
	t.Helper()
	expectEqual(t, dirty, term.isDirty(Point{
		Tag:    PointTagActive,
		Active: PointCoordinate{x: 0, y: uint32(y)},
	}))
}

// Zig: test "term51: printRepeat: huge count is bounded by the caller, repeats print()"
func TestTm4_printRepeat_huge_count_is_bounded_by_the_caller_repeats_print(t *testing.T) {
	testStdCapacity(t)
	opts := terminalOptionsDefault(3, 2)
	opts.max_scrollback = 0
	term := tm4Init(t, opts)
	defer term.Deinit()

	zigTry(t, term.Print('x'))
	zigTry(t, term.PrintRepeat(1000))
	// 1001 chars over 3 columns: the last row holds the remainder of two.
	tm4SExpectPlain(t, term, "xxx\nxx")
	tm4SExpectCursor(t, term, 2, 1, false)
}

// Zig: test "term51: print: ignored while the status line display is selected"
func TestTm4_print_ignored_while_the_status_line_display_is_selected(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 2))
	defer term.Deinit()
	term.clearDirty()

	term.status_display = ANSIStatusDisplayStatusLine
	zigTry(t, term.Print('a'))
	zigTry(t, term.Print(0x4E2D))
	zigTry(t, term.Print(0x301))
	zigTry(t, term.Print(0xFE0F))
	tm4SExpectCursor(t, term, 0, 0, false)
	tm4ExpectOptional(t, None[uint32](), term.previous_char)
	tm4SExpectPlain(t, term, "")
	tm4SExpectRowDirty(t, term, 0, false)

	term.status_display = ANSIStatusDisplayMain
	zigTry(t, term.Print('b'))
	tm4SExpectPlain(t, term, "b")
	tm4SExpectRowDirty(t, term, 0, true)
}

// Zig: test "term51: print: ascii fast path writes narrow cells and advances"
func TestTm4_print_ascii_fast_path_writes_narrow_cells_and_advances(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(100, 2))
	defer term.Deinit()
	term.clearDirty()

	for c := uint32(0x20); c < 0x7F; c++ {
		zigTry(t, term.Print(c))
	}
	tm4SExpectCursor(t, term, 95, 0, false)
	for i := uint(0); i < 95; i++ {
		tm4SExpectCell(t, term, i, 0, uint32(0x20+i), PageCellWideNarrow)
	}
	tm4ExpectOptional(t, Some[uint32](0x7E), term.previous_char)
	tm4SExpectRowDirty(t, term, 0, true)
	tm4SExpectRowDirty(t, term, 1, false)

	// Cells carry the default style and no extras.
	lc := tm4SAt(t, term, 3, 0)
	expectEqual(t, StyleDefaultId, lc.cell.style_id)
	expect(t, !lc.cell.protected, "!lc.cell.protected")
	expect(t, !lc.cell.hyperlink, "!lc.cell.hyperlink")
	expect(t, !lc.row.wrap, "!lc.row.wrap")
	expect(t, !lc.row.styled, "!lc.row.styled")
}

// Zig: test "term51: print: codepoints up to 0xFF take the one-cell path"
func TestTm4_print_codepoints_up_to_0xFF_take_the_one_cell_path(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(20, 2))
	defer term.Deinit()

	// NUL, a C0 control, DEL, C1 controls, soft hyphen, 0xFF and the first
	// codepoint beyond the byte range are all width 1.
	cps := [...]uint32{0, 0x1F, 0x7F, 0x80, 0x9F, 0xAD, 0xFF, 0x100}
	for _, cp := range cps {
		zigTry(t, term.Print(cp))
	}
	tm4SExpectCursor(t, term, uint(len(cps)), 0, false)
	for i, cp := range cps {
		tm4SExpectCell(t, term, uint(i), 0, cp, PageCellWideNarrow)
	}
	tm4ExpectOptional(t, Some[uint32](0x100), term.previous_char)

	// NUL consumed a cell but has no text.
	expect(t, !tm4SAt(t, term, 0, 0).cell.HasText(), "!term51SAt(&t, 0, 0).cell.hasText()")
	expect(t, tm4SAt(t, term, 0, 0).cell.IsEmpty(), "term51SAt(&t, 0, 0).cell.isEmpty()")

	// A NUL is remembered as the previous char and repeats as a NUL.
	term2 := tm4Init(t, terminalOptionsDefault(4, 2))
	defer term2.Deinit()
	zigTry(t, term2.Print(0))
	tm4ExpectOptional(t, Some[uint32](0), term2.previous_char)
	zigTry(t, term2.PrintRepeat(2))
	tm4SExpectCursor(t, term2, 3, 0, false)
}

// Zig: test "term51: print: narrow codepoints beyond the byte range use the width table"
func TestTm4_print_narrow_codepoints_beyond_the_byte_range_use_the_width_table(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(10, 2))
	defer term.Deinit()

	// U+0100 (Latin), U+2500 (box drawing), U+2764 (heart, text default)
	tm4SPrintAll(t, term, []uint32{0x100, 0x2500, 0x2764})
	tm4SExpectCursor(t, term, 3, 0, false)
	tm4SExpectCell(t, term, 0, 0, 0x100, PageCellWideNarrow)
	tm4SExpectCell(t, term, 1, 0, 0x2500, PageCellWideNarrow)
	tm4SExpectCell(t, term, 2, 0, 0x2764, PageCellWideNarrow)
}

// Zig: test "term51: print: wide character occupies head and spacer tail"
func TestTm4_print_wide_character_occupies_head_and_spacer_tail(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()
	term.clearDirty()

	zigTry(t, term.Print(0x4E2D))
	tm4SExpectCell(t, term, 0, 0, 0x4E2D, PageCellWideWide)
	tm4SExpectCell(t, term, 1, 0, 0, PageCellWideSpacerTail)
	tm4SExpectCursor(t, term, 2, 0, false)
	tm4ExpectOptional(t, Some[uint32](0x4E2D), term.previous_char)
	tm4SExpectRowDirty(t, term, 0, true)

	zigTry(t, term.Print('a'))
	tm4SExpectCell(t, term, 2, 0, 'a', PageCellWideNarrow)
	zigTry(t, term.Print(0x1F600))
	tm4SExpectCell(t, term, 3, 0, 0x1F600, PageCellWideWide)
	tm4SExpectCell(t, term, 4, 0, 0, PageCellWideSpacerTail)
	tm4SExpectCursor(t, term, 5, 0, false)
}

// Zig: test "term51: print: wide character exactly fitting the last two columns"
func TestTm4_print_wide_character_exactly_fitting_the_last_two_columns(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 2))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("abc")))
	zigTry(t, term.Print(0x4E2D))
	tm4SExpectCell(t, term, 3, 0, 0x4E2D, PageCellWideWide)
	tm4SExpectCell(t, term, 4, 0, 0, PageCellWideSpacerTail)
	tm4SExpectCursor(t, term, 4, 0, true)
	// No wrap has happened yet.
	expect(t, !tm4SAt(t, term, 0, 0).row.wrap, "!term51SAt(&t, 0, 0).row.wrap")
	expect(t, !tm4SAt(t, term, 0, 1).row.wrap_continuation, "!term51SAt(&t, 0, 1).row.wrap_continuation")
}

// Zig: test "term51: print: wide character at the last column wraps with a spacer head"
func TestTm4_print_wide_character_at_the_last_column_wraps_with_a_spacer_head(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 3))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("abcd")))
	tm4SExpectCursor(t, term, 4, 0, false)
	zigTry(t, term.Print(0x4E2D))

	tm4SExpectCell(t, term, 3, 0, 'd', PageCellWideNarrow)
	tm4SExpectCell(t, term, 4, 0, 0, PageCellWideSpacerHead)
	expect(t, tm4SAt(t, term, 0, 0).row.wrap, "term51SAt(&t, 0, 0).row.wrap")
	expect(t, tm4SAt(t, term, 0, 1).row.wrap_continuation, "term51SAt(&t, 0, 1).row.wrap_continuation")
	tm4SExpectCell(t, term, 0, 1, 0x4E2D, PageCellWideWide)
	tm4SExpectCell(t, term, 1, 1, 0, PageCellWideSpacerTail)
	tm4SExpectCursor(t, term, 2, 1, false)
	tm4ExpectOptional(t, Some[uint32](0x4E2D), term.previous_char)
}

// Zig: test "term51: print: wide character at the last column replaces existing text with the spacer head"
func TestTm4_print_wide_character_at_the_last_column_replaces_existing_text_with_the_spacer_head(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 3))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("abcde")))
	term.SetCursorPos(1, 5)
	zigTry(t, term.Print(0x4E2D))
	// The old 'e' was overwritten by the spacer head.
	tm4SExpectCell(t, term, 4, 0, 0, PageCellWideSpacerHead)
	tm4SExpectCell(t, term, 0, 1, 0x4E2D, PageCellWideWide)
}

// Zig: test "term51: print: wide character with pending wrap wraps first"
func TestTm4_print_wide_character_with_pending_wrap_wraps_first(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 3))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("abcde")))
	tm4SExpectCursor(t, term, 4, 0, true)
	zigTry(t, term.Print(0x4E2D))
	// The pending wrap was honoured, so no spacer head: 'e' stays.
	tm4SExpectCell(t, term, 4, 0, 'e', PageCellWideNarrow)
	expect(t, tm4SAt(t, term, 0, 0).row.wrap, "term51SAt(&t, 0, 0).row.wrap")
	expect(t, tm4SAt(t, term, 0, 1).row.wrap_continuation, "term51SAt(&t, 0, 1).row.wrap_continuation")
	tm4SExpectCell(t, term, 0, 1, 0x4E2D, PageCellWideWide)
	tm4SExpectCell(t, term, 1, 1, 0, PageCellWideSpacerTail)
	tm4SExpectCursor(t, term, 2, 1, false)
}

// Zig: test "term51: print: wide character at the last column without wraparound is dropped"
func TestTm4_print_wide_character_at_the_last_column_without_wraparound_is_dropped(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 3))
	defer term.Deinit()
	term.modes.Set(ModesModeWraparound, false)

	zigTry(t, term.PrintString([]uint8("abcd")))
	tm4SExpectCursor(t, term, 4, 0, false)
	zigTry(t, term.Print(0x4E2D))
	// Not printed, cursor not moved, nothing scrolled. The previous char is
	// still recorded because it is set before the width check.
	tm4SExpectCell(t, term, 4, 0, 0, PageCellWideNarrow)
	tm4SExpectCursor(t, term, 4, 0, false)
	tm4ExpectOptional(t, Some[uint32](0x4E2D), term.previous_char)
	expect(t, !tm4SAt(t, term, 0, 0).row.wrap, "!term51SAt(&t, 0, 0).row.wrap")

	// A narrow character is still printed there, which also pends a wrap.
	zigTry(t, term.Print('e'))
	tm4SExpectCell(t, term, 4, 0, 'e', PageCellWideNarrow)
	tm4SExpectCursor(t, term, 4, 0, true)

	// With the pending wrap set but wraparound off, a wide char is still
	// dropped and the state is unchanged.
	zigTry(t, term.Print(0x1F600))
	tm4SExpectCell(t, term, 4, 0, 'e', PageCellWideNarrow)
	tm4SExpectCursor(t, term, 4, 0, true)
	tm4SExpectPlain(t, term, "abcde")
}

// Zig: test "term51: print: wide character in a one column terminal prints a blank"
func TestTm4_print_wide_character_in_a_one_column_terminal_prints_a_blank(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(1, 3))
	defer term.Deinit()

	zigTry(t, term.Print(0x4E2D))
	tm4SExpectCell(t, term, 0, 0, 0, PageCellWideNarrow)
	tm4SExpectCursor(t, term, 0, 0, true)
	tm4ExpectOptional(t, Some[uint32](0x4E2D), term.previous_char)

	// Next one wraps first and again prints a blank.
	zigTry(t, term.Print(0x4E2D))
	tm4SExpectCursor(t, term, 0, 1, true)
	expect(t, tm4SAt(t, term, 0, 0).row.wrap, "term51SAt(&t, 0, 0).row.wrap")
	expect(t, tm4SAt(t, term, 0, 1).row.wrap_continuation, "term51SAt(&t, 0, 1).row.wrap_continuation")

	// A narrow char wraps first (pending) and prints on the next row.
	zigTry(t, term.Print('x'))
	tm4SExpectCell(t, term, 0, 1, 0, PageCellWideNarrow)
	tm4SExpectCell(t, term, 0, 2, 'x', PageCellWideNarrow)
	tm4SExpectCursor(t, term, 0, 2, true)
}

// Zig: test "term51: print: wide character overwriting parts of other wide characters"
func TestTm4_print_wide_character_overwriting_parts_of_other_wide_characters(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	// Two wide chars: [中][ ][日][ ]
	tm4SPrintAll(t, term, []uint32{0x4E2D, 0x65E5})

	// Overwrite the narrow position x=1 (the tail of the first) with wide:
	// the first head is cleared and the wide char takes [1][2], which
	// clears the head at 2 and leaves its old tail at 3 orphaned-cleared.
	term.SetCursorPos(1, 2)
	zigTry(t, term.Print(0x1F600))
	tm4SExpectCell(t, term, 0, 0, 0, PageCellWideNarrow)
	tm4SExpectCell(t, term, 1, 0, 0x1F600, PageCellWideWide)
	tm4SExpectCell(t, term, 2, 0, 0, PageCellWideSpacerTail)
	tm4SExpectCell(t, term, 3, 0, 0, PageCellWideNarrow)
	tm4SExpectCursor(t, term, 3, 0, false)
}

// Zig: test "term51: print: narrow character overwriting wide head and tail"
func TestTm4_print_narrow_character_overwriting_wide_head_and_tail(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	zigTry(t, term.Print(0x4E2D))
	term.SetCursorPos(1, 1)
	zigTry(t, term.Print('a'))
	tm4SExpectCell(t, term, 0, 0, 'a', PageCellWideNarrow)
	tm4SExpectCell(t, term, 1, 0, 0, PageCellWideNarrow)

	term.SetCursorPos(1, 4)
	zigTry(t, term.Print(0x4E2D))
	term.SetCursorPos(1, 5)
	zigTry(t, term.Print('b'))
	tm4SExpectCell(t, term, 3, 0, 0, PageCellWideNarrow)
	tm4SExpectCell(t, term, 4, 0, 'b', PageCellWideNarrow)
}

// Zig: test "term51: print: wraparound writes a soft wrap and continuation"
func TestTm4_print_wraparound_writes_a_soft_wrap_and_continuation(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(3, 3))
	defer term.Deinit()

	zigTry(t, term.Print('a'))
	zigTry(t, term.Print('b'))
	tm4SExpectCursor(t, term, 2, 0, false)
	zigTry(t, term.Print('c'))
	// The last column stays and a wrap is pending.
	tm4SExpectCursor(t, term, 2, 0, true)
	expect(t, !tm4SAt(t, term, 0, 0).row.wrap, "!term51SAt(&t, 0, 0).row.wrap")
	zigTry(t, term.Print('d'))
	tm4SExpectCursor(t, term, 1, 1, false)
	expect(t, tm4SAt(t, term, 0, 0).row.wrap, "term51SAt(&t, 0, 0).row.wrap")
	expect(t, !tm4SAt(t, term, 0, 0).row.wrap_continuation, "!term51SAt(&t, 0, 0).row.wrap_continuation")
	expect(t, tm4SAt(t, term, 0, 1).row.wrap_continuation, "term51SAt(&t, 0, 1).row.wrap_continuation")
	expect(t, !tm4SAt(t, term, 0, 1).row.wrap, "!term51SAt(&t, 0, 1).row.wrap")
	tm4SExpectPlain(t, term, "abc\nd")
}

// Zig: test "term51: print: without wraparound the last cell is overwritten"
func TestTm4_print_without_wraparound_the_last_cell_is_overwritten(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(3, 3))
	defer term.Deinit()
	term.modes.Set(ModesModeWraparound, false)

	zigTry(t, term.PrintString([]uint8("abcdef")))
	tm4SExpectPlain(t, term, "abf")
	// The pending wrap flag is still raised even though it does nothing.
	tm4SExpectCursor(t, term, 2, 0, true)
	expect(t, !tm4SAt(t, term, 0, 0).row.wrap, "!term51SAt(&t, 0, 0).row.wrap")
	tm4ExpectOptional(t, Some[uint32]('f'), term.previous_char)
}

// Zig: test "term51: print: pending wrap at the bottom scrolls and creates scrollback"
func TestTm4_print_pending_wrap_at_the_bottom_scrolls_and_creates_scrollback(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(2, 2))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("abcde")))
	tm4SExpectPlain(t, term, "cd\ne")
	tm4SExpectCursor(t, term, 1, 1, false)

	// "ab" is in scrollback, soft wrapped into "cd" which wraps into "e".
	func() {
		lc := zigUnwrap(t, term.screens.active.pages.GetCell(Point{Tag: PointTagScreen, Screen: PointCoordinate{x: 0, y: 0}}))
		expectEqual(t, uint32('a'), lc.cell.Codepoint())
		expect(t, lc.row.wrap, "lc.row.wrap")
	}()
	expect(t, tm4SAt(t, term, 0, 0).row.wrap, "term51SAt(&t, 0, 0).row.wrap")
	expect(t, tm4SAt(t, term, 0, 0).row.wrap_continuation, "term51SAt(&t, 0, 0).row.wrap_continuation")
	expect(t, tm4SAt(t, term, 0, 1).row.wrap_continuation, "term51SAt(&t, 0, 1).row.wrap_continuation")
}

// Zig: test "term51: print: insert mode shifts the rest of the line right"
func TestTm4_print_insert_mode_shifts_the_rest_of_the_line_right(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("abcd")))
	term.modes.Set(ModesModeInsert, true)
	term.SetCursorPos(1, 2)
	zigTry(t, term.Print('X'))
	tm4SExpectPlain(t, term, "aXbcd")
	tm4SExpectCursor(t, term, 2, 0, false)
	zigTry(t, term.Print('Y'))
	tm4SExpectPlain(t, term, "aXYbcd")
	// The line is full; the next insert pushes the last char off.
	zigTry(t, term.Print('Z'))
	tm4SExpectPlain(t, term, "aXYZbc")
	tm4SExpectCursor(t, term, 4, 0, false)
}

// Zig: test "term51: print: insert mode boundary one column before the end shifts"
func TestTm4_print_insert_mode_boundary_one_column_before_the_end_shifts(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("abcdef")))
	term.modes.Set(ModesModeInsert, true)
	// x + width < cols: 4 + 1 < 6 so we insert.
	term.SetCursorPos(1, 5)
	zigTry(t, term.Print('X'))
	tm4SExpectPlain(t, term, "abcdXe")
	tm4SExpectCursor(t, term, 5, 0, false)
}

// Zig: test "term51: print: insert mode boundary at the last column overwrites"
func TestTm4_print_insert_mode_boundary_at_the_last_column_overwrites(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("abcdef")))
	term.modes.Set(ModesModeInsert, true)
	// x + width < cols: 5 + 1 < 6 is false so no insert, just overwrite.
	term.SetCursorPos(1, 6)
	zigTry(t, term.Print('X'))
	tm4SExpectPlain(t, term, "abcdeX")
	tm4SExpectCursor(t, term, 5, 0, true)
}

// Zig: test "term51: print: insert mode with a wide character"
func TestTm4_print_insert_mode_with_a_wide_character(t *testing.T) {
	testStdCapacity(t)
	// Wide at cols-2: x + 2 < cols false, no insert, overwrites the last two.
	func() {
		term := tm4Init(t, terminalOptionsDefault(6, 2))
		defer term.Deinit()
		zigTry(t, term.PrintString([]uint8("abcdef")))
		term.modes.Set(ModesModeInsert, true)
		term.SetCursorPos(1, 5)
		zigTry(t, term.Print(0x4E2D))
		tm4SExpectCell(t, term, 3, 0, 'd', PageCellWideNarrow)
		tm4SExpectCell(t, term, 4, 0, 0x4E2D, PageCellWideWide)
		tm4SExpectCell(t, term, 5, 0, 0, PageCellWideSpacerTail)
		tm4SExpectCursor(t, term, 5, 0, true)
	}()

	// Wide at cols-3: 3 + 2 < 6 so two blanks are inserted first.
	func() {
		term := tm4Init(t, terminalOptionsDefault(6, 2))
		defer term.Deinit()
		zigTry(t, term.PrintString([]uint8("abcdef")))
		term.modes.Set(ModesModeInsert, true)
		term.SetCursorPos(1, 4)
		zigTry(t, term.Print(0x4E2D))
		tm4SExpectCell(t, term, 2, 0, 'c', PageCellWideNarrow)
		tm4SExpectCell(t, term, 3, 0, 0x4E2D, PageCellWideWide)
		tm4SExpectCell(t, term, 4, 0, 0, PageCellWideSpacerTail)
		tm4SExpectCell(t, term, 5, 0, 'd', PageCellWideNarrow)
		tm4SExpectCursor(t, term, 5, 0, false)
	}()
}

// Zig: test "term51: print: insert mode with pending wrap inserts on the next line"
func TestTm4_print_insert_mode_with_pending_wrap_inserts_on_the_next_line(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(3, 3))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("abc")))
	term.modes.Set(ModesModeInsert, true)
	tm4SExpectCursor(t, term, 2, 0, true)
	zigTry(t, term.Print('X'))
	tm4SExpectPlain(t, term, "abc\nX")
	tm4SExpectCursor(t, term, 1, 1, false)
}

// Zig: test "term51: print: insert mode respects the right margin"
func TestTm4_print_insert_mode_respects_the_right_margin(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(10, 2))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("0123456789")))
	term.modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(3, 6)
	term.modes.Set(ModesModeInsert, true)
	term.SetCursorPos(1, 4)
	zigTry(t, term.Print('X'))
	// Only columns 3..5 shifted; column 6.. untouched.
	tm4SExpectPlain(t, term, "012X346789")
	tm4SExpectCursor(t, term, 4, 0, false)

	// At the right margin cell the single blank replaces it and the print
	// pends a wrap at the margin.
	term.SetCursorPos(1, 6)
	zigTry(t, term.Print('Y'))
	tm4SExpectPlain(t, term, "012X3Y6789")
	tm4SExpectCursor(t, term, 5, 0, true)
}

// Zig: test "term51: print: right margin wraps to the left margin without marking a soft wrap"
func TestTm4_print_right_margin_wraps_to_the_left_margin_without_marking_a_soft_wrap(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(10, 3))
	defer term.Deinit()

	term.modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(3, 5)
	term.SetCursorPos(1, 3)
	zigTry(t, term.PrintString([]uint8("abcd")))
	tm4SExpectCell(t, term, 2, 0, 'a', PageCellWideNarrow)
	tm4SExpectCell(t, term, 3, 0, 'b', PageCellWideNarrow)
	tm4SExpectCell(t, term, 4, 0, 'c', PageCellWideNarrow)
	tm4SExpectCell(t, term, 2, 1, 'd', PageCellWideNarrow)
	tm4SExpectCursor(t, term, 3, 1, false)
	expect(t, !tm4SAt(t, term, 0, 0).row.wrap, "!term51SAt(&t, 0, 0).row.wrap")
	expect(t, !tm4SAt(t, term, 0, 1).row.wrap_continuation, "!term51SAt(&t, 0, 1).row.wrap_continuation")
}

// Zig: test "term51: print: cursor right of the right margin uses the screen edge and wraps to the left margin"
func TestTm4_print_cursor_right_of_the_right_margin_uses_the_screen_edge_and_wraps_to_the_left_margin(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(10, 3))
	defer term.Deinit()

	term.modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(3, 5)
	term.SetCursorPos(1, 9)
	zigTry(t, term.PrintString([]uint8("abc")))
	tm4SExpectCell(t, term, 8, 0, 'a', PageCellWideNarrow)
	tm4SExpectCell(t, term, 9, 0, 'b', PageCellWideNarrow)
	// 'c' wrapped to the LEFT MARGIN (col 2), with a real soft wrap since
	// the old column was the last of the screen.
	tm4SExpectCell(t, term, 2, 1, 'c', PageCellWideNarrow)
	tm4SExpectCursor(t, term, 3, 1, false)
	expect(t, tm4SAt(t, term, 0, 0).row.wrap, "term51SAt(&t, 0, 0).row.wrap")
	expect(t, tm4SAt(t, term, 0, 1).row.wrap_continuation, "term51SAt(&t, 0, 1).row.wrap_continuation")
}

// Zig: test "term51: print: cursor left of the left margin prints until the right margin"
func TestTm4_print_cursor_left_of_the_left_margin_prints_until_the_right_margin(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(10, 3))
	defer term.Deinit()

	term.modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(3, 5)
	term.SetCursorPos(1, 1)
	zigTry(t, term.PrintString([]uint8("abcde")))
	tm4SExpectPlain(t, term, "abcde")
	// Pending wrap raised at the right margin column (index 4).
	tm4SExpectCursor(t, term, 4, 0, true)
	zigTry(t, term.Print('f'))
	tm4SExpectCell(t, term, 2, 1, 'f', PageCellWideNarrow)
	tm4SExpectCursor(t, term, 3, 1, false)
}

// Zig: test "term51: print: wide character at the right margin wraps with a narrow blank"
func TestTm4_print_wide_character_at_the_right_margin_wraps_with_a_narrow_blank(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(10, 3))
	defer term.Deinit()

	term.modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(3, 6)
	term.SetCursorPos(1, 6)
	zigTry(t, term.Print('X'))
	term.SetCursorPos(1, 6)
	zigTry(t, term.Print(0x4E2D))
	tm4SExpectCell(t, term, 5, 0, 0, PageCellWideNarrow)
	expect(t, !tm4SAt(t, term, 0, 0).row.wrap, "!term51SAt(&t, 0, 0).row.wrap")
	tm4SExpectCell(t, term, 2, 1, 0x4E2D, PageCellWideWide)
	tm4SExpectCell(t, term, 3, 1, 0, PageCellWideSpacerTail)
	tm4SExpectCursor(t, term, 4, 1, false)
	expect(t, !tm4SAt(t, term, 0, 1).row.wrap_continuation, "!term51SAt(&t, 0, 1).row.wrap_continuation")
}

// Zig: test "term51: print: wide character ending exactly on the right margin pends a wrap"
func TestTm4_print_wide_character_ending_exactly_on_the_right_margin_pends_a_wrap(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(10, 3))
	defer term.Deinit()

	term.modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(3, 6)
	term.SetCursorPos(1, 5)
	zigTry(t, term.Print(0x4E2D))
	tm4SExpectCell(t, term, 4, 0, 0x4E2D, PageCellWideWide)
	tm4SExpectCell(t, term, 5, 0, 0, PageCellWideSpacerTail)
	tm4SExpectCursor(t, term, 5, 0, true)
}

// Zig: test "term51: print: protected mode marks printed cells"
func TestTm4_print_protected_mode_marks_printed_cells(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()

	zigTry(t, term.Print('a'))
	term.SetProtectedMode(ANSIProtectedModeIso)
	zigTry(t, term.Print('b'))
	zigTry(t, term.Print(0x4E2D))
	term.SetProtectedMode(ANSIProtectedModeOff)
	zigTry(t, term.Print('c'))
	expect(t, !tm4SAt(t, term, 0, 0).cell.protected, "!term51SAt(&t, 0, 0).cell.protected")
	expect(t, tm4SAt(t, term, 1, 0).cell.protected, "term51SAt(&t, 1, 0).cell.protected")
	// Both halves of a wide character are protected.
	expect(t, tm4SAt(t, term, 2, 0).cell.protected, "term51SAt(&t, 2, 0).cell.protected")
	expect(t, tm4SAt(t, term, 3, 0).cell.protected, "term51SAt(&t, 3, 0).cell.protected")
	expect(t, !tm4SAt(t, term, 4, 0).cell.protected, "!term51SAt(&t, 4, 0).cell.protected")

	// Printing again unprotected over protected cells clears the flag.
	term.SetCursorPos(1, 2)
	zigTry(t, term.Print('z'))
	expect(t, !tm4SAt(t, term, 1, 0).cell.protected, "!term51SAt(&t, 1, 0).cell.protected")
}

// Zig: test "term51: print: cells take the cursor style and track style reference counts"
func TestTm4_print_cells_take_the_cursor_style_and_track_style_reference_counts(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()

	zigTry(t, term.screens.active.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	zigTry(t, term.Print('a'))
	zigTry(t, term.Print('b'))
	func() {
		lc := tm4SAt(t, term, 0, 0)
		page := &lc.node.data
		expect(t, lc.cell.style_id != StyleDefaultId, "lc.cell.style_id != style.default_id")
		expect(t, lc.row.styled, "lc.row.styled")
		expect(t, page.styles.Get(lc.cell.style_id).flags.bold, "page.styles.get(page.memory, lc.cell.style_id).flags.bold")
		expectEqual(t, lc.cell.style_id, tm4SAt(t, term, 1, 0).cell.style_id)
		// One ref for the cursor plus the two cells.
		expectEqual(t, SizeCellCountInt(3), page.styles.RefCount(lc.cell.style_id))
	}()

	// Overwriting with the same style does not change the reference count.
	term.SetCursorPos(1, 1)
	zigTry(t, term.Print('c'))
	func() {
		lc := tm4SAt(t, term, 0, 0)
		page := &lc.node.data
		expectEqual(t, SizeCellCountInt(3), page.styles.RefCount(lc.cell.style_id))
	}()

	// Overwriting with the default style releases a reference.
	bold_id := tm4SAt(t, term, 0, 0).cell.style_id
	zigTry(t, term.screens.active.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))
	term.SetCursorPos(1, 1)
	zigTry(t, term.Print('d'))
	func() {
		lc := tm4SAt(t, term, 0, 0)
		page := &lc.node.data
		expectEqual(t, StyleDefaultId, lc.cell.style_id)
		expectEqual(t, SizeCellCountInt(1), page.styles.RefCount(bold_id))
	}()

	// Overwriting a styled cell with another style swaps the references.
	zigTry(t, term.screens.active.SetAttribute(SGRAttribute{Tag: SGRAttributeTagItalic}))
	zigTry(t, term.Print('e'))
	func() {
		lc := tm4SAt(t, term, 1, 0)
		page := &lc.node.data
		expect(t, page.styles.Get(lc.cell.style_id).flags.italic, "page.styles.get(page.memory, lc.cell.style_id).flags.italic")
		// cursor + this cell
		expectEqual(t, SizeCellCountInt(2), page.styles.RefCount(lc.cell.style_id))
	}()
}

// Zig: test "term51: print: wide character with a style styles both cells"
func TestTm4_print_wide_character_with_a_style_styles_both_cells(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()

	zigTry(t, term.screens.active.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineSingle}))
	zigTry(t, term.Print(0x4E2D))
	head := tm4SAt(t, term, 0, 0)
	tail := tm4SAt(t, term, 1, 0)
	expect(t, head.cell.style_id != StyleDefaultId, "head.cell.style_id != style.default_id")
	expectEqual(t, head.cell.style_id, tail.cell.style_id)
	expect(t, head.row.styled, "head.row.styled")
	page := &head.node.data
	// cursor + head + tail
	expectEqual(t, SizeCellCountInt(3), page.styles.RefCount(head.cell.style_id))
}

// Zig: test "term51: print: hyperlink applies to both halves of a wide character"
func TestTm4_print_hyperlink_applies_to_both_halves_of_a_wide_character(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()

	zigTry(t, term.screens.active.StartHyperlink([]uint8("http://example.com"), None[[]uint8]()))
	zigTry(t, term.Print(0x4E2D))
	zigTry(t, term.Print('a'))
	for x := uint(0); x < 3; x++ {
		lc := tm4SAt(t, term, x, 0)
		expect(t, lc.cell.hyperlink, "lc.cell.hyperlink")
		expectEqual(t, HyperlinkId(1), zigUnwrap(t, lc.node.data.LookupHyperlink(lc.cell)))
	}
	expect(t, tm4SAt(t, term, 0, 0).row.hyperlink, "term51SAt(&t, 0, 0).row.hyperlink")
	expectEqual(t, uint(1), tm4SAt(t, term, 0, 0).node.data.hyperlink_set.Count())
}

// Zig: test "term51: print: overwriting hyperlink cells keeps the row flag while any link remains"
func TestTm4_print_overwriting_hyperlink_cells_keeps_the_row_flag_while_any_link_remains(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()

	zigTry(t, term.screens.active.StartHyperlink([]uint8("http://example.com"), None[[]uint8]()))
	zigTry(t, term.PrintString([]uint8("ab")))
	term.screens.active.EndHyperlink()

	// Overwrite the first cell with a plain char: the second still has a link.
	term.SetCursorPos(1, 1)
	zigTry(t, term.Print('x'))
	expect(t, !tm4SAt(t, term, 0, 0).cell.hyperlink, "!term51SAt(&t, 0, 0).cell.hyperlink")
	expect(t, tm4SAt(t, term, 1, 0).cell.hyperlink, "term51SAt(&t, 1, 0).cell.hyperlink")
	expect(t, tm4SAt(t, term, 0, 0).row.hyperlink, "term51SAt(&t, 0, 0).row.hyperlink")

	// Overwrite the second too: the row flag clears and the page set empties.
	zigTry(t, term.Print('y'))
	expect(t, !tm4SAt(t, term, 1, 0).cell.hyperlink, "!term51SAt(&t, 1, 0).cell.hyperlink")
	expect(t, !tm4SAt(t, term, 0, 0).row.hyperlink, "!term51SAt(&t, 0, 0).row.hyperlink")
	expectEqual(t, uint(0), tm4SAt(t, term, 0, 0).node.data.hyperlink_set.Count())
}

// Zig: test "term51: print: overwriting a hyperlink with a different one swaps ids"
func TestTm4_print_overwriting_a_hyperlink_with_a_different_one_swaps_ids(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()

	zigTry(t, term.screens.active.StartHyperlink([]uint8("http://one.example.com"), None[[]uint8]()))
	zigTry(t, term.Print('a'))
	zigTry(t, term.screens.active.StartHyperlink([]uint8("http://two.example.com"), None[[]uint8]()))
	term.SetCursorPos(1, 1)
	zigTry(t, term.Print('b'))
	lc := tm4SAt(t, term, 0, 0)
	expect(t, lc.cell.hyperlink, "lc.cell.hyperlink")
	id := zigUnwrap(t, lc.node.data.LookupHyperlink(lc.cell))
	entry := lc.node.data.hyperlink_set.Get(id)
	expectEqualStrings(t, "http://two.example.com", string(entry.uri))
	// The first link is no longer referenced by any cell.
	expectEqual(t, uint(1), lc.node.data.hyperlink_set.Count())
}

// Zig: test "term51: print: cells take the cursor semantic content"
func TestTm4_print_cells_take_the_cursor_semantic_content(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()

	zigTry(t, term.Print('a'))
	term.screens.active.cursor.semantic_content = PageCellSemanticContentPrompt
	zigTry(t, term.Print('b'))
	term.screens.active.cursor.semantic_content = PageCellSemanticContentInput
	zigTry(t, term.Print(0x4E2D))
	term.screens.active.cursor.semantic_content = PageCellSemanticContentOutput
	zigTry(t, term.Print('c'))
	expectEqual(t, PageCellSemanticContentOutput, tm4SAt(t, term, 0, 0).cell.semantic_content)
	expectEqual(t, PageCellSemanticContentPrompt, tm4SAt(t, term, 1, 0).cell.semantic_content)
	expectEqual(t, PageCellSemanticContentInput, tm4SAt(t, term, 2, 0).cell.semantic_content)
	expectEqual(t, PageCellSemanticContentInput, tm4SAt(t, term, 3, 0).cell.semantic_content)
	expectEqual(t, PageCellSemanticContentOutput, tm4SAt(t, term, 4, 0).cell.semantic_content)
}

// Zig: test "term51: print: charset maps ascii through the active GL table"
func TestTm4_print_charset_maps_ascii_through_the_active_GL_table(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(20, 2))
	defer term.Deinit()

	term.ConfigureCharset(CharsetsSlotsG0, CharsetsCharsetDecSpecial)
	zigTry(t, term.PrintString([]uint8("q`x~A#")))
	tm4SExpectCell(t, term, 0, 0, 0x2500, PageCellWideNarrow)
	tm4SExpectCell(t, term, 1, 0, 0x25C6, PageCellWideNarrow)
	tm4SExpectCell(t, term, 2, 0, 0x2502, PageCellWideNarrow)
	tm4SExpectCell(t, term, 3, 0, 0x00B7, PageCellWideNarrow)
	// Characters the table leaves alone.
	tm4SExpectCell(t, term, 4, 0, 'A', PageCellWideNarrow)
	tm4SExpectCell(t, term, 5, 0, '#', PageCellWideNarrow)

	term.ConfigureCharset(CharsetsSlotsG0, CharsetsCharsetBritish)
	zigTry(t, term.PrintString([]uint8("#a")))
	tm4SExpectCell(t, term, 6, 0, 0xA3, PageCellWideNarrow)
	tm4SExpectCell(t, term, 7, 0, 'a', PageCellWideNarrow)

	term.ConfigureCharset(CharsetsSlotsG0, CharsetsCharsetAscii)
	zigTry(t, term.PrintString([]uint8("#q")))
	tm4SExpectCell(t, term, 8, 0, '#', PageCellWideNarrow)
	tm4SExpectCell(t, term, 9, 0, 'q', PageCellWideNarrow)

	term.ConfigureCharset(CharsetsSlotsG0, CharsetsCharsetUtf8)
	zigTry(t, term.PrintString([]uint8("#q")))
	tm4SExpectCell(t, term, 10, 0, '#', PageCellWideNarrow)
	tm4SExpectCell(t, term, 11, 0, 'q', PageCellWideNarrow)
}

// Zig: test "term51: print: previous char is the unmapped character"
func TestTm4_print_previous_char_is_the_unmapped_character(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()

	term.ConfigureCharset(CharsetsSlotsG0, CharsetsCharsetDecSpecial)
	zigTry(t, term.Print('q'))
	tm4ExpectOptional(t, Some[uint32]('q'), term.previous_char)
	zigTry(t, term.PrintRepeat(1))
	tm4SExpectCell(t, term, 1, 0, 0x2500, PageCellWideNarrow)
}

// Zig: test "term51: print: charset table covers the whole byte range, beyond it becomes space"
func TestTm4_print_charset_table_covers_the_whole_byte_range_beyond_it_becomes_space(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(12, 2))
	defer term.Deinit()

	term.ConfigureCharset(CharsetsSlotsG0, CharsetsCharsetDecSpecial)
	// 0x7F..0xFF are identity in the table, 0x100 and above are spaces.
	tm4SPrintAll(t, term, []uint32{0x7F, 0xE9, 0xFF, 0x100, 0x2500})
	tm4SExpectCell(t, term, 0, 0, 0x7F, PageCellWideNarrow)
	tm4SExpectCell(t, term, 1, 0, 0xE9, PageCellWideNarrow)
	tm4SExpectCell(t, term, 2, 0, 0xFF, PageCellWideNarrow)
	tm4SExpectCell(t, term, 3, 0, ' ', PageCellWideNarrow)
	tm4SExpectCell(t, term, 4, 0, ' ', PageCellWideNarrow)

	// Also true for the british table.
	term.ConfigureCharset(CharsetsSlotsG0, CharsetsCharsetBritish)
	tm4SPrintAll(t, term, []uint32{0xFF, 0x100})
	tm4SExpectCell(t, term, 5, 0, 0xFF, PageCellWideNarrow)
	tm4SExpectCell(t, term, 6, 0, ' ', PageCellWideNarrow)
}

// Zig: test "term51: print: wide character with a mapping charset becomes a wide space"
func TestTm4_print_wide_character_with_a_mapping_charset_becomes_a_wide_space(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()

	term.ConfigureCharset(CharsetsSlotsG0, CharsetsCharsetDecSpecial)
	zigTry(t, term.Print(0x4E2D))
	tm4SExpectCell(t, term, 0, 0, ' ', PageCellWideWide)
	tm4SExpectCell(t, term, 1, 0, 0, PageCellWideSpacerTail)
	tm4SExpectCursor(t, term, 2, 0, false)
	tm4ExpectOptional(t, Some[uint32](0x4E2D), term.previous_char)

	// The spacer tail's zero goes through the table identically.
	term.ConfigureCharset(CharsetsSlotsG0, CharsetsCharsetBritish)
	zigTry(t, term.Print(0x4E2D))
	tm4SExpectCell(t, term, 2, 0, ' ', PageCellWideWide)
	tm4SExpectCell(t, term, 3, 0, 0, PageCellWideSpacerTail)
}

// Zig: test "term51: print: single shift is consumed by exactly one printed character"
func TestTm4_print_single_shift_is_consumed_by_exactly_one_printed_character(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	term.ConfigureCharset(CharsetsSlotsG2, CharsetsCharsetDecSpecial)
	term.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG2, true)
	tm4ExpectOptional(t, Some[CharsetsSlots](CharsetsSlotsG2), term.screens.active.charset.single_shift)
	zigTry(t, term.PrintString([]uint8("qqq")))
	tm4SExpectCell(t, term, 0, 0, 0x2500, PageCellWideNarrow)
	tm4SExpectCell(t, term, 1, 0, 'q', PageCellWideNarrow)
	tm4SExpectCell(t, term, 2, 0, 'q', PageCellWideNarrow)
	tm4ExpectOptional(t, None[CharsetsSlots](), term.screens.active.charset.single_shift)
}

// Zig: test "term51: print: single shift takes priority over GL and keeps GL"
func TestTm4_print_single_shift_takes_priority_over_GL_and_keeps_GL(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	term.ConfigureCharset(CharsetsSlotsG0, CharsetsCharsetBritish)
	term.ConfigureCharset(CharsetsSlotsG3, CharsetsCharsetDecSpecial)
	term.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG3, true)
	zigTry(t, term.PrintString([]uint8("##")))
	// First char: G3 (dec special leaves '#'), second: G0 british.
	tm4SExpectCell(t, term, 0, 0, '#', PageCellWideNarrow)
	tm4SExpectCell(t, term, 1, 0, 0xA3, PageCellWideNarrow)
	expectEqual(t, CharsetsSlotsG0, term.screens.active.charset.gl)
}

// Zig: test "term51: print: single shift on a wide character is spent on the head"
func TestTm4_print_single_shift_on_a_wide_character_is_spent_on_the_head(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	term.ConfigureCharset(CharsetsSlotsG1, CharsetsCharsetDecSpecial)
	term.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG1, true)
	term.ConfigureCharset(CharsetsSlotsG0, CharsetsCharsetBritish)
	zigTry(t, term.Print(0x4E2D))
	tm4SExpectCell(t, term, 0, 0, ' ', PageCellWideWide)
	tm4SExpectCell(t, term, 1, 0, 0, PageCellWideSpacerTail)
	tm4ExpectOptional(t, None[CharsetsSlots](), term.screens.active.charset.single_shift)
}

// Zig: test "term51: print: zero-width attach does not consume a single shift"
func TestTm4_print_zero_width_attach_does_not_consume_a_single_shift(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	zigTry(t, term.Print('e'))
	term.ConfigureCharset(CharsetsSlotsG1, CharsetsCharsetDecSpecial)
	term.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG1, true)
	zigTry(t, term.Print(0x301))
	tm4ExpectOptional(t, Some[CharsetsSlots](CharsetsSlotsG1), term.screens.active.charset.single_shift)
	zigTry(t, term.Print('q'))
	tm4SExpectCell(t, term, 1, 0, 0x2500, PageCellWideNarrow)
}

// Zig: test "term51: print: GR is never used when printing"
func TestTm4_print_GR_is_never_used_when_printing(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	term.ConfigureCharset(CharsetsSlotsG1, CharsetsCharsetDecSpecial)
	term.InvokeCharset(CharsetsActiveSlotGR, CharsetsSlotsG1, false)
	zigTry(t, term.Print('q'))
	zigTry(t, term.Print(0xE9))
	tm4SExpectCell(t, term, 0, 0, 'q', PageCellWideNarrow)
	tm4SExpectCell(t, term, 1, 0, 0xE9, PageCellWideNarrow)

	// Switching GL to the mapped slot does apply.
	term.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG1, false)
	zigTry(t, term.Print('q'))
	tm4SExpectCell(t, term, 2, 0, 0x2500, PageCellWideNarrow)
}

// Zig: test "term51: print: overwriting grapheme cells clears grapheme data and the row flag"
func TestTm4_print_overwriting_grapheme_cells_clears_grapheme_data_and_the_row_flag(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()

	// Two cells with graphemes (2027 off, so combining marks attach).
	tm4SPrintAll(t, term, []uint32{'a', 0x301, 'b', 0x302})
	tm4SExpectGrapheme(t, term, 0, 0, []uint32{0x301})
	tm4SExpectGrapheme(t, term, 1, 0, []uint32{0x302})

	term.SetCursorPos(1, 1)
	zigTry(t, term.Print('x'))
	tm4SExpectNoGrapheme(t, term, 0, 0)
	// The other cell still has one, so the row flag remains.
	expect(t, tm4SAt(t, term, 0, 0).row.grapheme, "term51SAt(&t, 0, 0).row.grapheme")
	tm4SExpectGrapheme(t, term, 1, 0, []uint32{0x302})

	zigTry(t, term.Print('y'))
	tm4SExpectNoGrapheme(t, term, 1, 0)
	expect(t, !tm4SAt(t, term, 0, 0).row.grapheme, "!term51SAt(&t, 0, 0).row.grapheme")
	expectEqual(t, uint(0), tm4SAt(t, term, 0, 0).node.data.GraphemeCount())
}

// Zig: test "term51: print: combining mark without clustering attaches to the previous cell"
func TestTm4_print_combining_mark_without_clustering_attaches_to_the_previous_cell(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 2))
	defer term.Deinit()

	zigTry(t, term.Print('e'))
	term.clearDirty()
	zigTry(t, term.Print(0x301))
	// This path does not mark the row dirty (unlike the clustering path).
	tm4SExpectRowDirty(t, term, 0, false)
	tm4SExpectCell(t, term, 0, 0, 'e', PageCellWideNarrow)
	tm4SExpectGrapheme(t, term, 0, 0, []uint32{0x301})
	// The cursor does not move and the zero-width char is not the previous
	// char for repeat.
	tm4SExpectCursor(t, term, 1, 0, false)
	tm4ExpectOptional(t, Some[uint32]('e'), term.previous_char)

	// A second mark is appended to the same cell.
	zigTry(t, term.Print(0x302))
	tm4SExpectGrapheme(t, term, 0, 0, []uint32{0x301, 0x302})
	tm4SExpectCursor(t, term, 1, 0, false)
	tm4SExpectPlain(t, term, "e\u0301\u0302")
}

// Zig: test "term51: print: zero-width character at column zero is ignored"
func TestTm4_print_zero_width_character_at_column_zero_is_ignored(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 2))
	defer term.Deinit()

	zigTry(t, term.Print(0x301))
	zigTry(t, term.Print(0x200D))
	zigTry(t, term.Print(0xFE0F))
	tm4SExpectCursor(t, term, 0, 0, false)
	tm4SExpectPlain(t, term, "")
	tm4ExpectOptional(t, None[uint32](), term.previous_char)
	tm4SExpectNoGrapheme(t, term, 0, 0)
}

// Zig: test "term51: print: zero-width character after an empty cell is ignored"
func TestTm4_print_zero_width_character_after_an_empty_cell_is_ignored(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 2))
	defer term.Deinit()

	term.SetCursorPos(1, 3)
	zigTry(t, term.Print(0x301))
	tm4SExpectCursor(t, term, 2, 0, false)
	tm4SExpectNoGrapheme(t, term, 1, 0)
	tm4SExpectNoGrapheme(t, term, 2, 0)
	tm4SExpectPlain(t, term, "")
}

// Zig: test "term51: print: zero-width character after a wide character attaches to the head"
func TestTm4_print_zero_width_character_after_a_wide_character_attaches_to_the_head(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()

	zigTry(t, term.Print(0x4E2D))
	zigTry(t, term.Print(0x301))
	tm4SExpectGrapheme(t, term, 0, 0, []uint32{0x301})
	tm4SExpectNoGrapheme(t, term, 1, 0)
	tm4SExpectCell(t, term, 1, 0, 0, PageCellWideSpacerTail)
	tm4SExpectCursor(t, term, 2, 0, false)
}

// Zig: test "term51: print: zero-width after text at the last column pends and attaches to the cell before"
func TestTm4_print_zero_width_after_text_at_the_last_column_pends_and_attaches_to_the_cell_before(t *testing.T) {
	testStdCapacity(t)
	// Without clustering the previous cell is always the one left of the
	// cursor, even when a pending wrap leaves the cursor ON the last glyph.
	term := tm4Init(t, terminalOptionsDefault(5, 2))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("abcde")))
	tm4SExpectCursor(t, term, 4, 0, true)
	zigTry(t, term.Print(0x301))
	tm4SExpectNoGrapheme(t, term, 4, 0)
	tm4SExpectGrapheme(t, term, 3, 0, []uint32{0x301})
	tm4SExpectCursor(t, term, 4, 0, true)
}

// Zig: test "term51: print: variation selectors without clustering need an extended pictographic base"
func TestTm4_print_variation_selectors_without_clustering_need_an_extended_pictographic_base(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(12, 2))
	defer term.Deinit()

	// '1' is a valid variation sequence base but not extended pictographic.
	zigTry(t, term.Print('1'))
	zigTry(t, term.Print(0xFE0F))
	tm4SExpectNoGrapheme(t, term, 0, 0)
	// 'a' neither.
	zigTry(t, term.Print('a'))
	zigTry(t, term.Print(0xFE0E))
	tm4SExpectNoGrapheme(t, term, 1, 0)

	// The heart is extended pictographic: the selector is attached, but the
	// width is NOT changed in this mode.
	zigTry(t, term.Print(0x2764))
	zigTry(t, term.Print(0xFE0F))
	tm4SExpectCell(t, term, 2, 0, 0x2764, PageCellWideNarrow)
	tm4SExpectGrapheme(t, term, 2, 0, []uint32{0xFE0F})
	tm4SExpectCursor(t, term, 3, 0, false)

	// A wide emoji accepts VS15 and stays wide.
	zigTry(t, term.Print(0x1F600))
	zigTry(t, term.Print(0xFE0E))
	tm4SExpectCell(t, term, 3, 0, 0x1F600, PageCellWideWide)
	tm4SExpectGrapheme(t, term, 3, 0, []uint32{0xFE0E})
	tm4SExpectCursor(t, term, 5, 0, false)
}

// Zig: test "term51: print: zero width joiner sequence without clustering uses separate cells"
func TestTm4_print_zero_width_joiner_sequence_without_clustering_uses_separate_cells(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(12, 2))
	defer term.Deinit()

	tm4SPrintAll(t, term, []uint32{0x1F468, 0x200D, 0x1F469})
	tm4SExpectCell(t, term, 0, 0, 0x1F468, PageCellWideWide)
	tm4SExpectGrapheme(t, term, 0, 0, []uint32{0x200D})
	tm4SExpectCell(t, term, 1, 0, 0, PageCellWideSpacerTail)
	tm4SExpectCell(t, term, 2, 0, 0x1F469, PageCellWideWide)
	tm4SExpectCursor(t, term, 4, 0, false)
}

// Zig: test "term51: print: many zero-width characters grow the grapheme storage"
func TestTm4_print_many_zero_width_characters_grow_the_grapheme_storage(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 2))
	defer term.Deinit()

	zigTry(t, term.Print('e'))
	// Stay below 256 codepoints: a larger append would trip the known
	// bitmap_allocator.zig:281 out-of-bounds defect.
	count := 200
	for i := 0; i < count; i++ {
		if i%2 == 0 {
			zigTry(t, term.Print(0x301))
		} else {
			zigTry(t, term.Print(0x302))
		}
	}
	lc := tm4SAt(t, term, 0, 0)
	cps := zigUnwrap(t, lc.node.data.LookupGrapheme(lc.cell))
	expectEqual(t, uint(count), uint(len(cps)))
	for i, cp := range cps {
		if i%2 == 0 {
			expectEqual(t, uint32(0x301), cp)
		} else {
			expectEqual(t, uint32(0x302), cp)
		}
	}
	tm4SExpectCursor(t, term, 1, 0, false)
}

// Zig: test "term51: print: clustering attaches combining marks and marks the row dirty"
func TestTm4_print_clustering_attaches_combining_marks_and_marks_the_row_dirty(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 2))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.Print('e'))
	term.clearDirty()
	zigTry(t, term.Print(0x301))
	tm4SExpectCell(t, term, 0, 0, 'e', PageCellWideNarrow)
	tm4SExpectGrapheme(t, term, 0, 0, []uint32{0x301})
	tm4SExpectCursor(t, term, 1, 0, false)
	tm4SExpectRowDirty(t, term, 0, true)
	tm4ExpectOptional(t, Some[uint32]('e'), term.previous_char)
}

// Zig: test "term51: print: clustering skips the grapheme path at column zero and for bytes"
func TestTm4_print_clustering_skips_the_grapheme_path_at_column_zero_and_for_bytes(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	// Zero-width at x == 0 is dropped (width 0 with 2027 returns).
	zigTry(t, term.Print(0x301))
	tm4SExpectCursor(t, term, 0, 0, false)
	tm4ExpectOptional(t, None[uint32](), term.previous_char)

	// Bytes (c <= 255) never cluster even when they could extend something.
	zigTry(t, term.Print('a'))
	zigTry(t, term.Print(0xAD))
	zigTry(t, term.Print(0x9F))
	tm4SExpectCell(t, term, 1, 0, 0xAD, PageCellWideNarrow)
	tm4SExpectCell(t, term, 2, 0, 0x9F, PageCellWideNarrow)
	tm4SExpectNoGrapheme(t, term, 0, 0)
	tm4SExpectCursor(t, term, 3, 0, false)
}

// Zig: test "term51: print: clustering a character after a grapheme break starts a new cell"
func TestTm4_print_clustering_a_character_after_a_grapheme_break_starts_a_new_cell(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.Print('a'))
	zigTry(t, term.Print(0x4E2D))
	zigTry(t, term.Print('b'))
	zigTry(t, term.Print(0x100))
	tm4SExpectCell(t, term, 0, 0, 'a', PageCellWideNarrow)
	tm4SExpectCell(t, term, 1, 0, 0x4E2D, PageCellWideWide)
	tm4SExpectCell(t, term, 2, 0, 0, PageCellWideSpacerTail)
	tm4SExpectCell(t, term, 3, 0, 'b', PageCellWideNarrow)
	tm4SExpectCell(t, term, 4, 0, 0x100, PageCellWideNarrow)
	tm4SExpectCursor(t, term, 5, 0, false)
	tm4ExpectOptional(t, Some[uint32](0x100), term.previous_char)
}

// Zig: test "term51: print: clustering ignores zero-width characters that do not join"
func TestTm4_print_clustering_ignores_zero_width_characters_that_do_not_join(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.Print('a'))
	// ZERO WIDTH SPACE is a break after 'a' and has width zero: dropped, not
	// attached (unlike the non-clustering path).
	zigTry(t, term.Print(0x200B))
	tm4SExpectNoGrapheme(t, term, 0, 0)
	tm4SExpectCursor(t, term, 1, 0, false)

	// After an empty cell a combining mark is dropped too.
	term.SetCursorPos(1, 4)
	zigTry(t, term.Print(0x301))
	tm4SExpectNoGrapheme(t, term, 2, 0)
	tm4SExpectCursor(t, term, 3, 0, false)
	tm4ExpectOptional(t, Some[uint32]('a'), term.previous_char)
}

// Zig: test "term51: print: clustering attaches to a wide character"
func TestTm4_print_clustering_attaches_to_a_wide_character(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.Print(0x4E2D))
	zigTry(t, term.Print(0x301))
	tm4SExpectCell(t, term, 0, 0, 0x4E2D, PageCellWideWide)
	tm4SExpectGrapheme(t, term, 0, 0, []uint32{0x301})
	tm4SExpectNoGrapheme(t, term, 1, 0)
	tm4SExpectCursor(t, term, 2, 0, false)
}

// Zig: test "term51: print: clustering attaches to the head of a wide character with pending wrap"
func TestTm4_print_clustering_attaches_to_the_head_of_a_wide_character_with_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(4, 2))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	term.SetCursorPos(1, 3)
	zigTry(t, term.Print(0x4E2D))
	tm4SExpectCursor(t, term, 3, 0, true)
	// pending wrap: the previous cell is the cursor cell (a spacer tail), so
	// the head is one further left.
	zigTry(t, term.Print(0x301))
	tm4SExpectGrapheme(t, term, 2, 0, []uint32{0x301})
	tm4SExpectNoGrapheme(t, term, 3, 0)
	tm4SExpectCursor(t, term, 3, 0, true)
}

// Zig: test "term51: print: clustering with pending wrap attaches to the last glyph"
func TestTm4_print_clustering_with_pending_wrap_attaches_to_the_last_glyph(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(3, 2))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.PrintString([]uint8("abc")))
	tm4SExpectCursor(t, term, 2, 0, true)
	zigTry(t, term.Print(0x301))
	tm4SExpectGrapheme(t, term, 2, 0, []uint32{0x301})
	tm4SExpectNoGrapheme(t, term, 1, 0)
	tm4SExpectCursor(t, term, 2, 0, true)

	// The next printable still wraps.
	zigTry(t, term.Print('d'))
	tm4SExpectCell(t, term, 0, 1, 'd', PageCellWideNarrow)
}

// Zig: test "term51: print: clustering without wraparound at the last column attaches by cell content"
func TestTm4_print_clustering_without_wraparound_at_the_last_column_attaches_by_cell_content(t *testing.T) {
	testStdCapacity(t)
	// Last column holds a glyph: attach to it.
	func() {
		term := tm4Init(t, terminalOptionsDefault(3, 2))
		defer term.Deinit()
		term.modes.Set(ModesModeGraphemeCluster, true)
		term.modes.Set(ModesModeWraparound, false)
		zigTry(t, term.PrintString([]uint8("abc")))
		zigTry(t, term.Print(0x301))
		tm4SExpectGrapheme(t, term, 2, 0, []uint32{0x301})
		tm4SExpectNoGrapheme(t, term, 1, 0)
	}()

	// Last column is empty (cursor parked): attach to the previous cell.
	func() {
		term := tm4Init(t, terminalOptionsDefault(3, 2))
		defer term.Deinit()
		term.modes.Set(ModesModeGraphemeCluster, true)
		term.modes.Set(ModesModeWraparound, false)
		zigTry(t, term.PrintString([]uint8("ab")))
		tm4SExpectCursor(t, term, 2, 0, false)
		zigTry(t, term.Print(0x301))
		tm4SExpectGrapheme(t, term, 1, 0, []uint32{0x301})
		tm4SExpectNoGrapheme(t, term, 2, 0)
	}()

	// Not at the last column: always the previous cell.
	func() {
		term := tm4Init(t, terminalOptionsDefault(5, 2))
		defer term.Deinit()
		term.modes.Set(ModesModeGraphemeCluster, true)
		term.modes.Set(ModesModeWraparound, false)
		zigTry(t, term.PrintString([]uint8("ab")))
		zigTry(t, term.Print(0x301))
		tm4SExpectGrapheme(t, term, 1, 0, []uint32{0x301})
	}()
}

// Zig: test "term51: print: clustering uses the cell left of the cursor when moved onto text"
func TestTm4_print_clustering_uses_the_cell_left_of_the_cursor_when_moved_onto_text(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.PrintString([]uint8("abc")))
	term.SetCursorPos(1, 3)
	zigTry(t, term.Print(0x301))
	// Attached to 'b', the cell left of the cursor, not to 'c'.
	tm4SExpectGrapheme(t, term, 1, 0, []uint32{0x301})
	tm4SExpectNoGrapheme(t, term, 2, 0)
	tm4SExpectCursor(t, term, 2, 0, false)
}

// Zig: test "term51: print: clustering ZWJ sequences stay in one wide cell"
func TestTm4_print_clustering_ZWJ_sequences_stay_in_one_wide_cell(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	tm4SPrintAll(t, term, []uint32{0x1F468, 0x200D, 0x1F469, 0x200D, 0x1F467})
	tm4SExpectCell(t, term, 0, 0, 0x1F468, PageCellWideWide)
	tm4SExpectGrapheme(t, term, 0, 0, []uint32{0x200D, 0x1F469, 0x200D, 0x1F467})
	tm4SExpectCell(t, term, 1, 0, 0, PageCellWideSpacerTail)
	tm4SExpectCursor(t, term, 2, 0, false)

	// The next emoji after a completed sequence is a new cell.
	zigTry(t, term.Print(0x1F600))
	tm4SExpectCell(t, term, 2, 0, 0x1F600, PageCellWideWide)
	tm4SExpectNoGrapheme(t, term, 2, 0)
	tm4SExpectCursor(t, term, 4, 0, false)
}

// Zig: test "term51: print: clustering regional indicators pair into flags"
func TestTm4_print_clustering_regional_indicators_pair_into_flags(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	// US and GB flags.
	tm4SPrintAll(t, term, []uint32{0x1F1FA, 0x1F1F8, 0x1F1EC, 0x1F1E7})
	tm4SExpectCell(t, term, 0, 0, 0x1F1FA, PageCellWideWide)
	tm4SExpectGrapheme(t, term, 0, 0, []uint32{0x1F1F8})
	tm4SExpectCell(t, term, 1, 0, 0, PageCellWideSpacerTail)
	tm4SExpectCell(t, term, 2, 0, 0x1F1EC, PageCellWideWide)
	tm4SExpectGrapheme(t, term, 2, 0, []uint32{0x1F1E7})
	tm4SExpectCell(t, term, 3, 0, 0, PageCellWideSpacerTail)
	tm4SExpectCursor(t, term, 4, 0, false)
}

// Zig: test "term51: print: clustering emoji modifier joins its base"
func TestTm4_print_clustering_emoji_modifier_joins_its_base(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	tm4SPrintAll(t, term, []uint32{0x1F44D, 0x1F3FB})
	tm4SExpectCell(t, term, 0, 0, 0x1F44D, PageCellWideWide)
	tm4SExpectGrapheme(t, term, 0, 0, []uint32{0x1F3FB})
	tm4SExpectCursor(t, term, 2, 0, false)
}

// Zig: test "term51: print: clustering a width-contributing codepoint makes the cell wide"
func TestTm4_print_clustering_a_width_contributing_codepoint_makes_the_cell_wide(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	// Devanagari conjunct: KA + VIRAMA + SSA.
	tm4SPrintAll(t, term, []uint32{0x915, 0x94D})
	tm4SExpectCell(t, term, 0, 0, 0x915, PageCellWideNarrow)
	tm4SExpectGrapheme(t, term, 0, 0, []uint32{0x94D})
	tm4SExpectCursor(t, term, 1, 0, false)
	zigTry(t, term.Print(0x937))
	tm4SExpectCell(t, term, 0, 0, 0x915, PageCellWideWide)
	tm4SExpectGrapheme(t, term, 0, 0, []uint32{0x94D, 0x937})
	tm4SExpectCell(t, term, 1, 0, 0, PageCellWideSpacerTail)
	tm4SExpectCursor(t, term, 2, 0, false)
}

// Zig: test "term51: print: clustering VS16 widens a narrow base mid-line"
func TestTm4_print_clustering_VS16_widens_a_narrow_base_mid_line(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.Print('a'))
	zigTry(t, term.Print('#'))
	term.clearDirty()
	zigTry(t, term.Print(0xFE0F))
	tm4SExpectCell(t, term, 1, 0, '#', PageCellWideWide)
	tm4SExpectGrapheme(t, term, 1, 0, []uint32{0xFE0F})
	tm4SExpectCell(t, term, 2, 0, 0, PageCellWideSpacerTail)
	tm4SExpectCursor(t, term, 3, 0, false)
	tm4SExpectRowDirty(t, term, 0, true)
	// previous_char is not updated by the selector.
	tm4ExpectOptional(t, Some[uint32]('#'), term.previous_char)
}

// Zig: test "term51: print: clustering VS16 widening to the exact end of the line pends a wrap"
func TestTm4_print_clustering_VS16_widening_to_the_exact_end_of_the_line_pends_a_wrap(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(3, 2))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	term.SetCursorPos(1, 2)
	zigTry(t, term.Print('#'))
	// Cursor is on the last column with no pending wrap.
	tm4SExpectCursor(t, term, 2, 0, false)
	zigTry(t, term.Print(0xFE0F))
	tm4SExpectCell(t, term, 1, 0, '#', PageCellWideWide)
	tm4SExpectCell(t, term, 2, 0, 0, PageCellWideSpacerTail)
	tm4SExpectCursor(t, term, 2, 0, true)
	expect(t, !tm4SAt(t, term, 0, 0).row.wrap, "!term51SAt(&t, 0, 0).row.wrap")
}

// Zig: test "term51: print: clustering VS16 on a base in the last column wraps it to the next line"
func TestTm4_print_clustering_VS16_on_a_base_in_the_last_column_wraps_it_to_the_next_line(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(4, 3))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	term.SetCursorPos(1, 4)
	zigTry(t, term.Print('#'))
	tm4SExpectCursor(t, term, 3, 0, true)
	zigTry(t, term.Print(0xFE0F))
	tm4SExpectCell(t, term, 3, 0, 0, PageCellWideSpacerHead)
	expect(t, tm4SAt(t, term, 0, 0).row.wrap, "term51SAt(&t, 0, 0).row.wrap")
	expect(t, tm4SAt(t, term, 0, 1).row.wrap_continuation, "term51SAt(&t, 0, 1).row.wrap_continuation")
	tm4SExpectCell(t, term, 0, 1, '#', PageCellWideWide)
	tm4SExpectGrapheme(t, term, 0, 1, []uint32{0xFE0F})
	tm4SExpectCell(t, term, 1, 1, 0, PageCellWideSpacerTail)
	tm4SExpectCursor(t, term, 2, 1, false)
}

// Zig: test "term51: print: clustering VS16 at a right margin wraps without a spacer head"
func TestTm4_print_clustering_VS16_at_a_right_margin_wraps_without_a_spacer_head(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(10, 3))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)
	term.modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(3, 5)

	term.SetCursorPos(1, 5)
	zigTry(t, term.Print('#'))
	tm4SExpectCursor(t, term, 4, 0, true)
	zigTry(t, term.Print(0xFE0F))
	tm4SExpectCell(t, term, 4, 0, 0, PageCellWideNarrow)
	expect(t, !tm4SAt(t, term, 0, 0).row.wrap, "!term51SAt(&t, 0, 0).row.wrap")
	tm4SExpectCell(t, term, 2, 1, '#', PageCellWideWide)
	tm4SExpectGrapheme(t, term, 2, 1, []uint32{0xFE0F})
	tm4SExpectCell(t, term, 3, 1, 0, PageCellWideSpacerTail)
	tm4SExpectCursor(t, term, 4, 1, false)
	expect(t, !tm4SAt(t, term, 0, 1).row.wrap_continuation, "!term51SAt(&t, 0, 1).row.wrap_continuation")
}

// Zig: test "term51: print: clustering VS16 on a last column base without wraparound does nothing"
func TestTm4_print_clustering_VS16_on_a_last_column_base_without_wraparound_does_nothing(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(3, 2))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)
	term.modes.Set(ModesModeWraparound, false)

	term.SetCursorPos(1, 3)
	zigTry(t, term.Print('#'))
	tm4SExpectCursor(t, term, 2, 0, true)
	zigTry(t, term.Print(0xFE0F))
	// The base remains narrow and does not get the selector.
	tm4SExpectCell(t, term, 2, 0, '#', PageCellWideNarrow)
	tm4SExpectNoGrapheme(t, term, 2, 0)
	tm4SExpectCursor(t, term, 2, 0, true)
}

// Zig: test "term51: print: clustering VS16 without wraparound widens when there is room"
func TestTm4_print_clustering_VS16_without_wraparound_widens_when_there_is_room(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(4, 2))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)
	term.modes.Set(ModesModeWraparound, false)

	term.SetCursorPos(1, 2)
	zigTry(t, term.Print('#'))
	zigTry(t, term.Print(0xFE0F))
	tm4SExpectCell(t, term, 1, 0, '#', PageCellWideWide)
	tm4SExpectCell(t, term, 2, 0, 0, PageCellWideSpacerTail)
	tm4SExpectCursor(t, term, 3, 0, false)

	// Parked on the last (empty) column: uses the previous cell.
	term2 := tm4Init(t, terminalOptionsDefault(3, 2))
	defer term2.Deinit()
	term2.modes.Set(ModesModeGraphemeCluster, true)
	term2.modes.Set(ModesModeWraparound, false)
	term2.SetCursorPos(1, 2)
	zigTry(t, term2.Print('#'))
	tm4SExpectCursor(t, term2, 2, 0, false)
	zigTry(t, term2.Print(0xFE0F))
	tm4SExpectCell(t, term2, 1, 0, '#', PageCellWideWide)
	tm4SExpectCell(t, term2, 2, 0, 0, PageCellWideSpacerTail)
	tm4SExpectCursor(t, term2, 2, 0, true)
}

// Zig: test "term51: print: clustering VS16 repeated on an already wide base only appends"
func TestTm4_print_clustering_VS16_repeated_on_an_already_wide_base_only_appends(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.Print('#'))
	zigTry(t, term.Print(0xFE0F))
	zigTry(t, term.Print(0xFE0F))
	tm4SExpectCell(t, term, 0, 0, '#', PageCellWideWide)
	tm4SExpectGrapheme(t, term, 0, 0, []uint32{0xFE0F, 0xFE0F})
	tm4SExpectCursor(t, term, 2, 0, false)
}

// Zig: test "term51: print: clustering variation selectors on non emoji bases are dropped"
func TestTm4_print_clustering_variation_selectors_on_non_emoji_bases_are_dropped(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.Print('a'))
	term.clearDirty()
	zigTry(t, term.Print(0xFE0F))
	zigTry(t, term.Print(0xFE0E))
	tm4SExpectNoGrapheme(t, term, 0, 0)
	tm4SExpectCell(t, term, 0, 0, 'a', PageCellWideNarrow)
	tm4SExpectCursor(t, term, 1, 0, false)
	tm4SExpectRowDirty(t, term, 0, false)

	// A wide CJK character is not a variation sequence base either.
	zigTry(t, term.Print(0x4E2D))
	zigTry(t, term.Print(0xFE0F))
	tm4SExpectNoGrapheme(t, term, 1, 0)
	tm4SExpectCursor(t, term, 3, 0, false)
}

// Zig: test "term51: print: clustering VS15 narrows a wide base and moves the cursor back"
func TestTm4_print_clustering_VS15_narrows_a_wide_base_and_moves_the_cursor_back(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.Print('a'))
	zigTry(t, term.Print(0x2614))
	tm4SExpectCell(t, term, 1, 0, 0x2614, PageCellWideWide)
	tm4SExpectCursor(t, term, 3, 0, false)
	zigTry(t, term.Print(0xFE0E))
	tm4SExpectCell(t, term, 1, 0, 0x2614, PageCellWideNarrow)
	tm4SExpectGrapheme(t, term, 1, 0, []uint32{0xFE0E})
	tm4SExpectCell(t, term, 2, 0, 0, PageCellWideNarrow)
	tm4SExpectCursor(t, term, 2, 0, false)
}

// Zig: test "term51: print: clustering VS15 on a wide base at the end of the line stays and unpends"
func TestTm4_print_clustering_VS15_on_a_wide_base_at_the_end_of_the_line_stays_and_unpends(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 2))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	term.SetCursorPos(1, 4)
	zigTry(t, term.Print(0x2614))
	tm4SExpectCursor(t, term, 4, 0, true)
	zigTry(t, term.Print(0xFE0E))
	tm4SExpectCell(t, term, 3, 0, 0x2614, PageCellWideNarrow)
	tm4SExpectGrapheme(t, term, 3, 0, []uint32{0xFE0E})
	tm4SExpectCell(t, term, 4, 0, 0, PageCellWideNarrow)
	tm4SExpectCursor(t, term, 4, 0, false)
}

// Zig: test "term51: print: clustering VS15 on a narrow base only appends"
func TestTm4_print_clustering_VS15_on_a_narrow_base_only_appends(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(8, 2))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.Print('#'))
	zigTry(t, term.Print(0xFE0E))
	tm4SExpectCell(t, term, 0, 0, '#', PageCellWideNarrow)
	tm4SExpectGrapheme(t, term, 0, 0, []uint32{0xFE0E})
	tm4SExpectCursor(t, term, 1, 0, false)
}

// Zig: test "term51: print: clustering VS16 transfers existing grapheme data when wrapping"
func TestTm4_print_clustering_VS16_transfers_existing_grapheme_data_when_wrapping(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(3, 3))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	term.SetCursorPos(1, 3)
	zigTry(t, term.Print('#'))
	zigTry(t, term.Print(0x301))
	tm4SExpectGrapheme(t, term, 2, 0, []uint32{0x301})
	zigTry(t, term.Print(0xFE0F))

	// The old cell became a spacer head with no graphemes and the row flag
	// was recomputed.
	tm4SExpectCell(t, term, 2, 0, 0, PageCellWideSpacerHead)
	tm4SExpectNoGrapheme(t, term, 2, 0)
	expect(t, !tm4SAt(t, term, 0, 0).row.grapheme, "!term51SAt(&t, 0, 0).row.grapheme")
	expect(t, tm4SAt(t, term, 0, 0).row.wrap, "term51SAt(&t, 0, 0).row.wrap")

	tm4SExpectCell(t, term, 0, 1, '#', PageCellWideWide)
	tm4SExpectGrapheme(t, term, 0, 1, []uint32{0x301, 0xFE0F})
	tm4SExpectCell(t, term, 1, 1, 0, PageCellWideSpacerTail)
	tm4SExpectCursor(t, term, 2, 1, false)
	expectEqual(t, uint(1), tm4SAt(t, term, 0, 0).node.data.GraphemeCount())
}

// Zig: test "term51: print: clustering VS16 transfers grapheme data across a page boundary"
func TestTm4_print_clustering_VS16_transfers_grapheme_data_across_a_page_boundary(t *testing.T) {
	testStdCapacity(t)
	rows := PageStdCapacity.rows
	cols := PageStdCapacity.cols
	term := tm4Init(t, terminalOptionsDefault(cols, rows))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	term.CursorDown(uint(rows - 1))
	for i := uint(rows); i < uint(term.screens.active.pages.pages.first.data.capacity.rows); i++ {
		zigTry(t, term.Index())
	}
	term.CursorRight(uint(cols - 1))

	zigTry(t, term.Print('#'))
	zigTry(t, term.Print(0x301))
	expect(t, term.screens.active.cursor.pending_wrap, "t.screens.active.cursor.pending_wrap")
	zigTry(t, term.Print(0xFE0F))

	tm4SExpectCell(t, term, uint(cols-1), uint(rows-2), 0, PageCellWideSpacerHead)
	tm4SExpectNoGrapheme(t, term, uint(cols-1), uint(rows-2))
	tm4SExpectCell(t, term, 0, uint(rows-1), '#', PageCellWideWide)
	tm4SExpectGrapheme(t, term, 0, uint(rows-1), []uint32{0x301, 0xFE0F})
	tm4SExpectCell(t, term, 1, uint(rows-1), 0, PageCellWideSpacerTail)
	tm4SExpectCursor(t, term, 2, uint(rows-1), false)
	// The old page no longer has any grapheme data.
	expectEqual(t, uint(0), tm4SAt(t, term, 0, uint(rows-2)).node.data.GraphemeCount())
}

// Zig: test "term51: print: clustering VS16 widening keeps style and hyperlink on the spacer tail"
func TestTm4_print_clustering_VS16_widening_keeps_style_and_hyperlink_on_the_spacer_tail(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.screens.active.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	zigTry(t, term.screens.active.StartHyperlink([]uint8("http://example.com"), None[[]uint8]()))
	zigTry(t, term.Print('#'))
	zigTry(t, term.Print(0xFE0F))
	head := tm4SAt(t, term, 0, 0)
	tail := tm4SAt(t, term, 1, 0)
	expectEqual(t, PageCellWideWide, head.cell.wide)
	expectEqual(t, PageCellWideSpacerTail, tail.cell.wide)
	expect(t, head.cell.hyperlink, "head.cell.hyperlink")
	expect(t, tail.cell.hyperlink, "tail.cell.hyperlink")
	expectEqual(t, head.cell.style_id, tail.cell.style_id)
	expect(t, head.cell.style_id != StyleDefaultId, "head.cell.style_id != style.default_id")
}

// Zig: test "term51: print: many cells with grapheme data grow the page capacity"
func TestTm4_print_many_cells_with_grapheme_data_grow_the_page_capacity(t *testing.T) {
	testStdCapacity(t)
	// 150 one-chunk graphemes exceed the 128 chunks of the default page, so
	// appendGrapheme has to increase the capacity (both clustering modes).
	for _, clustering := range []bool{false, true} {
		func() {
			// One spare column so the last cell is not left with a pending wrap.
			term := tm4Init(t, terminalOptionsDefault(151, 2))
			defer term.Deinit()
			term.modes.Set(ModesModeGraphemeCluster, clustering)

			for i := 0; i < 150; i++ {
				tm4SPrintAll(t, term, []uint32{'e', 0x301})
			}
			for x := uint(0); x < 150; x++ {
				tm4SExpectCell(t, term, x, 0, 'e', PageCellWideNarrow)
				tm4SExpectGrapheme(t, term, x, 0, []uint32{0x301})
			}
			expectEqual(t, uint(150), tm4SAt(t, term, 0, 0).node.data.GraphemeCount())
		}()
	}
}

// Zig: test "term51: print: clustering many combining marks grow the grapheme storage"
func TestTm4_print_clustering_many_combining_marks_grow_the_grapheme_storage(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 2))
	defer term.Deinit()
	term.modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.Print('e'))
	count := 200
	for i := 0; i < count; i++ {
		zigTry(t, term.Print(0x301))
	}
	lc := tm4SAt(t, term, 0, 0)
	expectEqual(t, uint(count), uint(len(zigUnwrap(t, lc.node.data.LookupGrapheme(lc.cell)))))
}

// Zig: test "term51: printCell: writes at the cursor without moving it or marking dirty"
func TestTm4_printCell_writes_at_the_cursor_without_moving_it_or_marking_dirty(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()

	term.SetCursorPos(2, 3)
	term.clearDirty()
	term.printCell('A', PageCellWideNarrow)
	tm4SExpectCell(t, term, 2, 1, 'A', PageCellWideNarrow)
	tm4SExpectCursor(t, term, 2, 1, false)
	tm4SExpectRowDirty(t, term, 1, false)
	tm4ExpectOptional(t, None[uint32](), term.previous_char)
	expectEqual(t, PageCellContentTagCodepoint, tm4SAt(t, term, 2, 1).cell.content_tag)

	// A zero codepoint with narrow writes an empty-looking cell.
	term.printCell(0, PageCellWideNarrow)
	tm4SExpectCell(t, term, 2, 1, 0, PageCellWideNarrow)

	// Writing again simply replaces the cell.
	term.printCell('B', PageCellWideNarrow)
	tm4SExpectCell(t, term, 2, 1, 'B', PageCellWideNarrow)
}

// Zig: test "term51: printCell: overwriting a spacer head just replaces it"
func TestTm4_printCell_overwriting_a_spacer_head_just_replaces_it(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(4, 3))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("abc")))
	zigTry(t, term.Print(0x4E2D))
	tm4SExpectCell(t, term, 3, 0, 0, PageCellWideSpacerHead)
	term.SetCursorPos(1, 4)
	term.printCell('x', PageCellWideNarrow)
	tm4SExpectCell(t, term, 3, 0, 'x', PageCellWideNarrow)
	// The wide char on the next row is untouched.
	tm4SExpectCell(t, term, 0, 1, 0x4E2D, PageCellWideWide)
	tm4SExpectCell(t, term, 1, 1, 0, PageCellWideSpacerTail)
}

// Zig: test "term51: printCell: replaces non-text cell content and takes pen attributes"
func TestTm4_printCell_replaces_non_text_cell_content_and_takes_pen_attributes(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()

	func() {
		lc := tm4SAt(t, term, 0, 0)
		*lc.cell = PageCell{
			content_tag: PageCellContentTagBgColorRgb,
			content:     PageCellContent{color_rgb: PageCellRGB{r: 1, g: 2, b: 3}},
		}
	}()
	term.screens.active.cursor.protected = true
	term.screens.active.cursor.semantic_content = PageCellSemanticContentInput
	term.printCell('Z', PageCellWideNarrow)
	cell := tm4SAt(t, term, 0, 0).cell
	expectEqual(t, PageCellContentTagCodepoint, cell.content_tag)
	expectEqual(t, uint32('Z'), cell.content.codepoint)
	expect(t, cell.protected, "cell.protected")
	expectEqual(t, PageCellSemanticContentInput, cell.semantic_content)
	expect(t, !cell.hyperlink, "!cell.hyperlink")
}

// Zig: test "term51: printCell: charset utf8 and ascii are identity, others map"
func TestTm4_printCell_charset_utf8_and_ascii_are_identity_others_map(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()

	term.ConfigureCharset(CharsetsSlotsG0, CharsetsCharsetUtf8)
	term.printCell(0x1F600, PageCellWideNarrow)
	tm4SExpectCell(t, term, 0, 0, 0x1F600, PageCellWideNarrow)
	term.ConfigureCharset(CharsetsSlotsG0, CharsetsCharsetAscii)
	term.printCell(0x1F600, PageCellWideNarrow)
	tm4SExpectCell(t, term, 0, 0, 0x1F600, PageCellWideNarrow)
	term.printCell('`', PageCellWideNarrow)
	tm4SExpectCell(t, term, 0, 0, '`', PageCellWideNarrow)

	term.ConfigureCharset(CharsetsSlotsG0, CharsetsCharsetDecSpecial)
	term.printCell('`', PageCellWideNarrow)
	tm4SExpectCell(t, term, 0, 0, 0x25C6, PageCellWideNarrow)
	// 255 is mapped (identity), 256 is outside the table: space.
	term.printCell(0xFF, PageCellWideNarrow)
	tm4SExpectCell(t, term, 0, 0, 0xFF, PageCellWideNarrow)
	term.printCell(0x100, PageCellWideNarrow)
	tm4SExpectCell(t, term, 0, 0, ' ', PageCellWideNarrow)

	term.ConfigureCharset(CharsetsSlotsG0, CharsetsCharsetBritish)
	term.printCell('#', PageCellWideNarrow)
	tm4SExpectCell(t, term, 0, 0, 0xA3, PageCellWideNarrow)
	term.printCell(0x1F600, PageCellWideNarrow)
	tm4SExpectCell(t, term, 0, 0, ' ', PageCellWideNarrow)
}

// Zig: test "term51: printCell: single shift is used once, even for the utf8 slot"
func TestTm4_printCell_single_shift_is_used_once_even_for_the_utf8_slot(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()

	term.ConfigureCharset(CharsetsSlotsG0, CharsetsCharsetDecSpecial)
	term.ConfigureCharset(CharsetsSlotsG2, CharsetsCharsetUtf8)
	term.screens.active.charset.single_shift = Some(CharsetsSlotsG2)
	term.printCell('q', PageCellWideNarrow)
	// The single shift slot is utf8 so no mapping happened.
	tm4SExpectCell(t, term, 0, 0, 'q', PageCellWideNarrow)
	tm4ExpectOptional(t, None[CharsetsSlots](), term.screens.active.charset.single_shift)
	term.printCell('q', PageCellWideNarrow)
	tm4SExpectCell(t, term, 0, 0, 0x2500, PageCellWideNarrow)
}

// Zig: test "term51: printCell: overwriting a wide head clears its tail"
func TestTm4_printCell_overwriting_a_wide_head_clears_its_tail(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()

	zigTry(t, term.Print(0x4E2D))
	term.SetCursorPos(1, 1)
	term.printCell('a', PageCellWideNarrow)
	tm4SExpectCell(t, term, 0, 0, 'a', PageCellWideNarrow)
	tm4SExpectCell(t, term, 1, 0, 0, PageCellWideNarrow)
	expect(t, tm4SAt(t, term, 1, 0).cell.IsEmpty(), "term51SAt(&t, 1, 0).cell.isEmpty()")
}

// Zig: test "term51: printCell: overwriting a spacer tail clears its head"
func TestTm4_printCell_overwriting_a_spacer_tail_clears_its_head(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()

	zigTry(t, term.Print(0x4E2D))
	term.SetCursorPos(1, 2)
	term.printCell('a', PageCellWideNarrow)
	tm4SExpectCell(t, term, 0, 0, 0, PageCellWideNarrow)
	tm4SExpectCell(t, term, 1, 0, 'a', PageCellWideNarrow)
}

// Zig: test "term51: printCell: overwriting a wide head near the left edge clears a stale spacer head above"
func TestTm4_printCell_overwriting_a_wide_head_near_the_left_edge_clears_a_stale_spacer_head_above(t *testing.T) {
	testStdCapacity(t)
	// Row 0 ends in a spacer head (wide char wrapped), row 1 starts with the
	// wide char. Overwriting at x = 0 or x = 1 clears the stale head.
	for _, overwrite_x := range []int{0, 1} {
		func() {
			term := tm4Init(t, terminalOptionsDefault(4, 3))
			defer term.Deinit()

			zigTry(t, term.PrintString([]uint8("abc")))
			zigTry(t, term.Print(0x4E2D))
			tm4SExpectCell(t, term, 3, 0, 0, PageCellWideSpacerHead)
			tm4SExpectCell(t, term, 0, 1, 0x4E2D, PageCellWideWide)

			if overwrite_x == 0 {
				// Overwrite the head itself.
				term.SetCursorPos(2, 1)
				term.printCell('z', PageCellWideNarrow)
				tm4SExpectCell(t, term, 0, 1, 'z', PageCellWideNarrow)
				tm4SExpectCell(t, term, 1, 1, 0, PageCellWideNarrow)
			} else {
				// Overwrite the tail.
				term.SetCursorPos(2, 2)
				term.printCell('z', PageCellWideNarrow)
				tm4SExpectCell(t, term, 0, 1, 0, PageCellWideNarrow)
				tm4SExpectCell(t, term, 1, 1, 'z', PageCellWideNarrow)
			}
			tm4SExpectCell(t, term, 3, 0, 0, PageCellWideNarrow)
		}()
	}
}

// Zig: test "term51: printCell: stale spacer head is kept when the overwritten wide char is further right"
func TestTm4_printCell_stale_spacer_head_is_kept_when_the_overwritten_wide_char_is_further_right(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 3))
	defer term.Deinit()

	// Row 0 ends in a spacer head, row 1 has a wide char at x=2..3.
	zigTry(t, term.PrintString([]uint8("abcde")))
	zigTry(t, term.Print(0x4E2D))
	tm4SExpectCell(t, term, 5, 0, 0, PageCellWideSpacerHead)
	term.SetCursorPos(2, 3)
	zigTry(t, term.Print(0x1F600))
	tm4SExpectCell(t, term, 2, 1, 0x1F600, PageCellWideWide)

	// Overwrite the head at x = 2: x <= 1 is false so row 0 is untouched.
	term.SetCursorPos(2, 3)
	term.printCell('z', PageCellWideNarrow)
	tm4SExpectCell(t, term, 2, 1, 'z', PageCellWideNarrow)
	tm4SExpectCell(t, term, 3, 1, 0, PageCellWideNarrow)
	tm4SExpectCell(t, term, 5, 0, 0, PageCellWideSpacerHead)
}

// Zig: test "term51: printCell: overwriting a wide head on the first row does not look above"
func TestTm4_printCell_overwriting_a_wide_head_on_the_first_row_does_not_look_above(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()

	zigTry(t, term.Print(0x4E2D))
	term.SetCursorPos(1, 1)
	// y == 0 so cursorCellEndOfPrev must not be consulted.
	term.printCell('q', PageCellWideNarrow)
	tm4SExpectCell(t, term, 0, 0, 'q', PageCellWideNarrow)
}

// Zig: test "term51: printCell: clears grapheme data of the overwritten cell"
func TestTm4_printCell_clears_grapheme_data_of_the_overwritten_cell(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()

	zigTry(t, term.Print('e'))
	zigTry(t, term.Print(0x301))
	tm4SExpectGrapheme(t, term, 0, 0, []uint32{0x301})
	term.SetCursorPos(1, 1)
	term.printCell('x', PageCellWideNarrow)
	tm4SExpectCell(t, term, 0, 0, 'x', PageCellWideNarrow)
	tm4SExpectNoGrapheme(t, term, 0, 0)
	expect(t, !tm4SAt(t, term, 0, 0).row.grapheme, "!term51SAt(&t, 0, 0).row.grapheme")
}

// Zig: test "term51: printCell: style reference counting only changes when the style id differs"
func TestTm4_printCell_style_reference_counting_only_changes_when_the_style_id_differs(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()

	zigTry(t, term.screens.active.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	id := term.screens.active.cursor.style_id
	page := &term.screens.active.cursor.page_pin.node.data
	expectEqual(t, SizeCellCountInt(1), page.styles.RefCount(id))

	term.printCell('a', PageCellWideNarrow)
	expectEqual(t, SizeCellCountInt(2), page.styles.RefCount(id))
	expect(t, term.screens.active.cursor.page_row.styled, "t.screens.active.cursor.page_row.styled")

	// Same style again: unchanged.
	term.printCell('b', PageCellWideNarrow)
	expectEqual(t, SizeCellCountInt(2), page.styles.RefCount(id))

	// Default style over styled: released.
	zigTry(t, term.screens.active.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))
	expectEqual(t, SizeCellCountInt(1), page.styles.RefCount(id))
	term.printCell('c', PageCellWideNarrow)
	expectEqual(t, SizeCellCountInt(0), page.styles.RefCount(id))
	expectEqual(t, StyleDefaultId, tm4SAt(t, term, 0, 0).cell.style_id)

	// Default over default: nothing to do.
	term.printCell('d', PageCellWideNarrow)
	expectEqual(t, StyleDefaultId, tm4SAt(t, term, 0, 0).cell.style_id)
}

// Zig: test "term51: printCell: hyperlink set, replaced and cleared"
func TestTm4_printCell_hyperlink_set_replaced_and_cleared(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 2))
	defer term.Deinit()

	// No link on a cell without a link: nothing happens.
	term.printCell('a', PageCellWideNarrow)
	expect(t, !tm4SAt(t, term, 0, 0).cell.hyperlink, "!term51SAt(&t, 0, 0).cell.hyperlink")
	expect(t, !tm4SAt(t, term, 0, 0).row.hyperlink, "!term51SAt(&t, 0, 0).row.hyperlink")

	// Active link is attached.
	zigTry(t, term.screens.active.StartHyperlink([]uint8("http://example.com"), None[[]uint8]()))
	term.printCell('b', PageCellWideNarrow)
	expect(t, tm4SAt(t, term, 0, 0).cell.hyperlink, "term51SAt(&t, 0, 0).cell.hyperlink")
	expect(t, tm4SAt(t, term, 0, 0).row.hyperlink, "term51SAt(&t, 0, 0).row.hyperlink")

	// Writing with no active link clears the old one and the row flag.
	term.screens.active.EndHyperlink()
	term.printCell('c', PageCellWideNarrow)
	expect(t, !tm4SAt(t, term, 0, 0).cell.hyperlink, "!term51SAt(&t, 0, 0).cell.hyperlink")
	expect(t, !tm4SAt(t, term, 0, 0).row.hyperlink, "!term51SAt(&t, 0, 0).row.hyperlink")
	expectEqual(t, uint(0), tm4SAt(t, term, 0, 0).node.data.hyperlink_set.Count())
}

// Zig: test "term51: printWrap: soft wraps at the screen edge"
func TestTm4_printWrap_soft_wraps_at_the_screen_edge(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(4, 3))
	defer term.Deinit()

	term.SetCursorPos(1, 4)
	zigTry(t, term.printWrap())
	tm4SExpectCursor(t, term, 0, 1, false)
	expect(t, tm4SAt(t, term, 0, 0).row.wrap, "term51SAt(&t, 0, 0).row.wrap")
	expect(t, tm4SAt(t, term, 0, 1).row.wrap_continuation, "term51SAt(&t, 0, 1).row.wrap_continuation")
	expect(t, !tm4SAt(t, term, 0, 1).row.wrap, "!term51SAt(&t, 0, 1).row.wrap")
	expect(t, !tm4SAt(t, term, 0, 0).row.wrap_continuation, "!term51SAt(&t, 0, 0).row.wrap_continuation")
}

// Zig: test "term51: printWrap: not at the screen edge there is no soft wrap"
func TestTm4_printWrap_not_at_the_screen_edge_there_is_no_soft_wrap(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(4, 3))
	defer term.Deinit()

	term.SetCursorPos(1, 3)
	term.screens.active.cursor.pending_wrap = true
	zigTry(t, term.printWrap())
	tm4SExpectCursor(t, term, 0, 1, false)
	expect(t, !tm4SAt(t, term, 0, 0).row.wrap, "!term51SAt(&t, 0, 0).row.wrap")
	expect(t, !tm4SAt(t, term, 0, 1).row.wrap_continuation, "!term51SAt(&t, 0, 1).row.wrap_continuation")
}

// Zig: test "term51: printWrap: goes to the left margin"
func TestTm4_printWrap_goes_to_the_left_margin(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(10, 3))
	defer term.Deinit()

	term.modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(4, 8)
	term.SetCursorPos(1, 8)
	zigTry(t, term.printWrap())
	tm4SExpectCursor(t, term, 3, 1, false)
	expect(t, !tm4SAt(t, term, 0, 0).row.wrap, "!term51SAt(&t, 0, 0).row.wrap")
	expect(t, !tm4SAt(t, term, 0, 1).row.wrap_continuation, "!term51SAt(&t, 0, 1).row.wrap_continuation")

	// At the screen edge outside the margins it still marks the wrap and
	// returns to the left margin.
	term.SetCursorPos(1, 10)
	zigTry(t, term.printWrap())
	tm4SExpectCursor(t, term, 3, 1, false)
	expect(t, tm4SAt(t, term, 0, 0).row.wrap, "term51SAt(&t, 0, 0).row.wrap")
	expect(t, tm4SAt(t, term, 0, 1).row.wrap_continuation, "term51SAt(&t, 0, 1).row.wrap_continuation")
}

// Zig: test "term51: printWrap: scrolls at the bottom of the region"
func TestTm4_printWrap_scrolls_at_the_bottom_of_the_region(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(3, 2))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("abc")))
	term.SetCursorPos(2, 3)
	zigTry(t, term.Print('x'))
	tm4SExpectCursor(t, term, 2, 1, true)
	zigTry(t, term.printWrap())
	tm4SExpectCursor(t, term, 0, 1, false)
	// "abc" scrolled into history and the row with the 'x' moved up.
	tm4SExpectCell(t, term, 2, 0, 'x', PageCellWideNarrow)
	expect(t, tm4SAt(t, term, 0, 0).row.wrap, "term51SAt(&t, 0, 0).row.wrap")
	expect(t, tm4SAt(t, term, 0, 1).row.wrap_continuation, "term51SAt(&t, 0, 1).row.wrap_continuation")
}

// Zig: test "term51: printWrap: semantic prompt state is carried to the new row"
func TestTm4_printWrap_semantic_prompt_state_is_carried_to_the_new_row(t *testing.T) {
	testStdCapacity(t)
	// Prompt: the new row is a prompt continuation and the state is kept.
	func() {
		term := tm4Init(t, terminalOptionsDefault(3, 3))
		defer term.Deinit()
		term.screens.active.cursor.semantic_content = PageCellSemanticContentPrompt
		term.SetCursorPos(1, 3)
		zigTry(t, term.printWrap())
		expectEqual(t, PageCellSemanticContentPrompt, term.screens.active.cursor.semantic_content)
		expect(t, !term.screens.active.cursor.semantic_content_clear_eol, "!t.screens.active.cursor.semantic_content_clear_eol")
		expectEqual(t, PageRowSemanticPromptPromptContinuation, tm4SAt(t, term, 0, 1).row.semantic_prompt)
	}()

	// Prompt with clear-eol: the index() inside resets the state but
	// printWrap restores both fields afterwards.
	func() {
		term := tm4Init(t, terminalOptionsDefault(3, 3))
		defer term.Deinit()
		term.screens.active.cursor.semantic_content = PageCellSemanticContentPrompt
		term.screens.active.cursor.semantic_content_clear_eol = true
		term.SetCursorPos(1, 3)
		zigTry(t, term.printWrap())
		expectEqual(t, PageCellSemanticContentPrompt, term.screens.active.cursor.semantic_content)
		expect(t, term.screens.active.cursor.semantic_content_clear_eol, "t.screens.active.cursor.semantic_content_clear_eol")
		expectEqual(t, PageRowSemanticPromptPromptContinuation, tm4SAt(t, term, 0, 1).row.semantic_prompt)
	}()

	// Input: state kept, and index() marks the continuation.
	func() {
		term := tm4Init(t, terminalOptionsDefault(3, 3))
		defer term.Deinit()
		term.screens.active.cursor.semantic_content = PageCellSemanticContentInput
		term.SetCursorPos(1, 3)
		zigTry(t, term.printWrap())
		expectEqual(t, PageCellSemanticContentInput, term.screens.active.cursor.semantic_content)
		expectEqual(t, PageRowSemanticPromptPromptContinuation, tm4SAt(t, term, 0, 1).row.semantic_prompt)
	}()

	// Input with clear-eol: no continuation mark, state restored.
	func() {
		term := tm4Init(t, terminalOptionsDefault(3, 3))
		defer term.Deinit()
		term.screens.active.cursor.semantic_content = PageCellSemanticContentInput
		term.screens.active.cursor.semantic_content_clear_eol = true
		term.SetCursorPos(1, 3)
		zigTry(t, term.printWrap())
		expectEqual(t, PageCellSemanticContentInput, term.screens.active.cursor.semantic_content)
		expect(t, term.screens.active.cursor.semantic_content_clear_eol, "t.screens.active.cursor.semantic_content_clear_eol")
		expectEqual(t, PageRowSemanticPromptNone, tm4SAt(t, term, 0, 1).row.semantic_prompt)
	}()

	// Output: nothing is marked.
	func() {
		term := tm4Init(t, terminalOptionsDefault(3, 3))
		defer term.Deinit()
		term.SetCursorPos(1, 3)
		zigTry(t, term.printWrap())
		expectEqual(t, PageCellSemanticContentOutput, term.screens.active.cursor.semantic_content)
		expectEqual(t, PageRowSemanticPromptNone, tm4SAt(t, term, 0, 1).row.semantic_prompt)
	}()
}

// Zig: test "term51: configureCharset: every slot takes every charset independently"
func TestTm4_configureCharset_every_slot_takes_every_charset_independently(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(4, 2))
	defer term.Deinit()
	term.clearDirty()

	cs := &term.screens.active.charset
	// Initial state: all utf8, GL = G0, GR = G2, no single shift.
	expectEqual(t, CharsetsCharsetUtf8, cs.charsets.Get(CharsetsSlotsG0))
	expectEqual(t, CharsetsCharsetUtf8, cs.charsets.Get(CharsetsSlotsG1))
	expectEqual(t, CharsetsCharsetUtf8, cs.charsets.Get(CharsetsSlotsG2))
	expectEqual(t, CharsetsCharsetUtf8, cs.charsets.Get(CharsetsSlotsG3))
	expectEqual(t, CharsetsSlotsG0, cs.gl)
	expectEqual(t, CharsetsSlotsG2, cs.gr)
	tm4ExpectOptional(t, None[CharsetsSlots](), cs.single_shift)

	slots := [...]CharsetsSlots{CharsetsSlotsG0, CharsetsSlotsG1, CharsetsSlotsG2, CharsetsSlotsG3}
	sets := [...]CharsetsCharset{CharsetsCharsetUtf8, CharsetsCharsetAscii, CharsetsCharsetBritish, CharsetsCharsetDecSpecial}
	for si, slot := range slots {
		for _, set := range sets {
			term.ConfigureCharset(slot, set)
			expectEqual(t, set, cs.charsets.Get(slot))
			// The other slots hold what we configured before.
			for oi, other := range slots {
				if oi == si {
					continue
				}
				expected := CharsetsCharsetUtf8
				if oi < si {
					expected = CharsetsCharsetDecSpecial
				}
				expectEqual(t, expected, cs.charsets.Get(other))
			}
		}
	}

	// No selection or single shift was touched and nothing became dirty.
	expectEqual(t, CharsetsSlotsG0, cs.gl)
	expectEqual(t, CharsetsSlotsG2, cs.gr)
	tm4ExpectOptional(t, None[CharsetsSlots](), cs.single_shift)
	tm4SExpectRowDirty(t, term, 0, false)
	tm4SExpectCursor(t, term, 0, 0, false)
}

// Zig: test "term51: configureCharset: state is per screen but copied on a screen switch"
func TestTm4_configureCharset_state_is_per_screen_but_copied_on_a_screen_switch(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(4, 2))
	defer term.Deinit()

	term.ConfigureCharset(CharsetsSlotsG1, CharsetsCharsetBritish)
	_, err := term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	// The alternate screen inherits the charset state.
	expectEqual(t, CharsetsCharsetBritish, term.screens.active.charset.charsets.Get(CharsetsSlotsG1))

	term.ConfigureCharset(CharsetsSlotsG1, CharsetsCharsetDecSpecial)
	alt_set := term.screens.active.charset.charsets.Get(CharsetsSlotsG1)
	expectEqual(t, CharsetsCharsetDecSpecial, alt_set)
	// The primary screen's own copy is unchanged until we switch back...
	expectEqual(t, CharsetsCharsetBritish, tm4Primary(t, term).charset.charsets.Get(CharsetsSlotsG1))
	// ...at which point the state of the screen we leave is carried over.
	_, err = term.SwitchScreen(ScreenSetKeyPrimary)
	zigTry(t, err)
	expectEqual(t, CharsetsCharsetDecSpecial, term.screens.active.charset.charsets.Get(CharsetsSlotsG1))
}

// Zig: test "term51: invokeCharset: locking shifts select GL and GR"
func TestTm4_invokeCharset_locking_shifts_select_GL_and_GR(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(4, 2))
	defer term.Deinit()

	cs := &term.screens.active.charset
	slots := [...]CharsetsSlots{CharsetsSlotsG0, CharsetsSlotsG1, CharsetsSlotsG2, CharsetsSlotsG3}
	for _, slot := range slots {
		term.InvokeCharset(CharsetsActiveSlotGL, slot, false)
		expectEqual(t, slot, cs.gl)
		// GR untouched.
		expectEqual(t, CharsetsSlotsG2, cs.gr)
	}
	term.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG0, false)
	for _, slot := range slots {
		term.InvokeCharset(CharsetsActiveSlotGR, slot, false)
		expectEqual(t, slot, cs.gr)
		// GL untouched.
		expectEqual(t, CharsetsSlotsG0, cs.gl)
	}
	tm4ExpectOptional(t, None[CharsetsSlots](), cs.single_shift)
}

// Zig: test "term51: invokeCharset: single shift only records the slot"
func TestTm4_invokeCharset_single_shift_only_records_the_slot(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(4, 2))
	defer term.Deinit()

	cs := &term.screens.active.charset
	slots := [...]CharsetsSlots{CharsetsSlotsG0, CharsetsSlotsG1, CharsetsSlotsG2, CharsetsSlotsG3}
	for _, slot := range slots {
		term.InvokeCharset(CharsetsActiveSlotGL, slot, true)
		tm4ExpectOptional(t, Some[CharsetsSlots](slot), cs.single_shift)
		// GL and GR remain at their defaults.
		expectEqual(t, CharsetsSlotsG0, cs.gl)
		expectEqual(t, CharsetsSlotsG2, cs.gr)
	}

	// A locking shift does not clear a pending single shift.
	term.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG1, false)
	tm4ExpectOptional(t, Some[CharsetsSlots](CharsetsSlotsG3), cs.single_shift)
	expectEqual(t, CharsetsSlotsG1, cs.gl)
	term.InvokeCharset(CharsetsActiveSlotGR, CharsetsSlotsG3, false)
	tm4ExpectOptional(t, Some[CharsetsSlots](CharsetsSlotsG3), cs.single_shift)
}

// Zig: test "term51: carriageReturn: at or right of the left margin goes to the left margin"
func TestTm4_carriageReturn_at_or_right_of_the_left_margin_goes_to_the_left_margin(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(10, 5))
	defer term.Deinit()
	term.scrolling_region.left = 2
	term.scrolling_region.right = 7

	for _, x := range []uint{2, 3, 5, 7, 8, 9} {
		term.screens.active.CursorAbsolute(SizeCellCountInt(x), 3)
		term.CarriageReturn()
		tm4SExpectCursor(t, term, 2, 3, false)
	}
}

// Zig: test "term51: carriageReturn: left of the left margin goes to column zero"
func TestTm4_carriageReturn_left_of_the_left_margin_goes_to_column_zero(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(10, 5))
	defer term.Deinit()
	term.scrolling_region.left = 2
	term.scrolling_region.right = 7

	for _, x := range []uint{0, 1} {
		term.screens.active.CursorAbsolute(SizeCellCountInt(x), 1)
		term.CarriageReturn()
		tm4SExpectCursor(t, term, 0, 1, false)
	}
}

// Zig: test "term51: carriageReturn: origin mode always uses the left margin"
func TestTm4_carriageReturn_origin_mode_always_uses_the_left_margin(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(10, 5))
	defer term.Deinit()
	term.scrolling_region.left = 2
	term.scrolling_region.right = 7
	term.modes.Set(ModesModeOrigin, true)

	for _, x := range []uint{0, 1, 2, 5, 9} {
		term.screens.active.CursorAbsolute(SizeCellCountInt(x), 2)
		term.CarriageReturn()
		tm4SExpectCursor(t, term, 2, 2, false)
	}
}

// Zig: test "term51: carriageReturn: without margins goes to column zero"
func TestTm4_carriageReturn_without_margins_goes_to_column_zero(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(10, 5))
	defer term.Deinit()

	term.screens.active.CursorAbsolute(6, 4)
	term.CarriageReturn()
	tm4SExpectCursor(t, term, 0, 4, false)
	// Already at zero is a no-op.
	term.CarriageReturn()
	tm4SExpectCursor(t, term, 0, 4, false)

	term.modes.Set(ModesModeOrigin, true)
	term.screens.active.CursorAbsolute(3, 1)
	term.CarriageReturn()
	tm4SExpectCursor(t, term, 0, 1, false)
}

// Zig: test "term51: carriageReturn: clears pending wrap without wrapping"
func TestTm4_carriageReturn_clears_pending_wrap_without_wrapping(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 3))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("abcde")))
	tm4SExpectCursor(t, term, 4, 0, true)
	term.CarriageReturn()
	tm4SExpectCursor(t, term, 0, 0, false)
	zigTry(t, term.Print('X'))
	tm4SExpectPlain(t, term, "Xbcde")
	expect(t, !tm4SAt(t, term, 0, 0).row.wrap, "!term51SAt(&t, 0, 0).row.wrap")
}

// Zig: test "term51: carriageReturn: uses the margin values even when mode 69 is off"
func TestTm4_carriageReturn_uses_the_margin_values_even_when_mode_69_is_off(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(10, 3))
	defer term.Deinit()

	term.modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(4, 8)
	term.modes.Set(ModesModeEnableLeftAndRightMargin, false)
	term.screens.active.CursorAbsolute(6, 1)
	term.CarriageReturn()
	tm4SExpectCursor(t, term, 3, 1, false)
}

// Zig: test "term51: linefeed: moves down keeping the column and clearing pending wrap"
func TestTm4_linefeed_moves_down_keeping_the_column_and_clearing_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 4))
	defer term.Deinit()

	term.screens.active.CursorAbsolute(3, 1)
	zigTry(t, term.Linefeed())
	tm4SExpectCursor(t, term, 3, 2, false)

	term.screens.active.CursorAbsolute(4, 2)
	term.screens.active.cursor.pending_wrap = true
	zigTry(t, term.Linefeed())
	tm4SExpectCursor(t, term, 4, 3, false)
}

// Zig: test "term51: linefeed: with linefeed mode also returns the carriage"
func TestTm4_linefeed_with_linefeed_mode_also_returns_the_carriage(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(10, 4))
	defer term.Deinit()
	term.modes.Set(ModesModeLinefeed, true)

	term.screens.active.CursorAbsolute(6, 0)
	zigTry(t, term.Linefeed())
	tm4SExpectCursor(t, term, 0, 1, false)

	// With a left margin the carriage returns to the margin when at or
	// right of it, otherwise to column zero.
	term.scrolling_region.left = 3
	term.scrolling_region.right = 8
	term.screens.active.CursorAbsolute(6, 1)
	zigTry(t, term.Linefeed())
	tm4SExpectCursor(t, term, 3, 2, false)
	term.screens.active.CursorAbsolute(1, 2)
	zigTry(t, term.Linefeed())
	tm4SExpectCursor(t, term, 0, 3, false)
}

// Zig: test "term51: linefeed: at the bottom of the screen scrolls"
func TestTm4_linefeed_at_the_bottom_of_the_screen_scrolls(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(4, 3))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("a\nb\nc")))
	tm4SExpectCursor(t, term, 1, 2, false)
	zigTry(t, term.Linefeed())
	tm4SExpectPlain(t, term, "b\nc")
	tm4SExpectCursor(t, term, 1, 2, false)
	// Without linefeed mode the column is preserved.
	zigTry(t, term.Print('d'))
	tm4SExpectPlain(t, term, "b\nc\n d")
}

// Zig: test "term51: linefeed: at the bottom of a scroll region scrolls only the region"
func TestTm4_linefeed_at_the_bottom_of_a_scroll_region_scrolls_only_the_region(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(3, 5))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("a\nb\nc\nd\ne")))
	term.scrolling_region.top = 1
	term.scrolling_region.bottom = 3
	term.screens.active.CursorAbsolute(0, 3)
	zigTry(t, term.Linefeed())
	tm4SExpectPlain(t, term, "a\nc\nd\n\ne")
	tm4SExpectCursor(t, term, 0, 3, false)
}

// Zig: test "term51: linefeed: outside the scroll region moves down without scrolling"
func TestTm4_linefeed_outside_the_scroll_region_moves_down_without_scrolling(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(3, 6))
	defer term.Deinit()
	term.scrolling_region.top = 1
	term.scrolling_region.bottom = 3

	// Above the region.
	term.screens.active.CursorAbsolute(1, 0)
	zigTry(t, term.Linefeed())
	tm4SExpectCursor(t, term, 1, 1, false)

	// Below the region.
	term.screens.active.CursorAbsolute(1, 4)
	zigTry(t, term.Linefeed())
	tm4SExpectCursor(t, term, 1, 5, false)

	// At the very bottom, below the region: nothing moves and nothing scrolls.
	zigTry(t, term.Print('x'))
	zigTry(t, term.Linefeed())
	tm4SExpectCursor(t, term, 2, 5, false)
	tm4SExpectPlain(t, term, "\n\n\n\n\n x")
}

// Zig: test "term51: linefeed: cursor right of the margins at the region bottom does not scroll"
func TestTm4_linefeed_cursor_right_of_the_margins_at_the_region_bottom_does_not_scroll(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(10, 3))
	defer term.Deinit()
	term.scrolling_region.left = 2
	term.scrolling_region.right = 7
	term.scrolling_region.top = 0
	term.scrolling_region.bottom = 1

	term.screens.active.CursorAbsolute(9, 1)
	zigTry(t, term.Linefeed())
	tm4SExpectCursor(t, term, 9, 1, false)
}

// Zig: test "term51: linefeed: left and right margins scroll only the columns between them"
func TestTm4_linefeed_left_and_right_margins_scroll_only_the_columns_between_them(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(6, 3))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("abcdef\nghijkl\nmnopqr")))
	term.scrolling_region.left = 1
	term.scrolling_region.right = 4
	term.screens.active.CursorAbsolute(2, 2)
	zigTry(t, term.Linefeed())
	tm4SExpectPlain(t, term, "ahijkf\ngnopql\nm    r")
	tm4SExpectCursor(t, term, 2, 2, false)
}

// Zig: test "term51: backspace: moves left one column and stops at column zero"
func TestTm4_backspace_moves_left_one_column_and_stops_at_column_zero(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 3))
	defer term.Deinit()

	term.screens.active.CursorAbsolute(2, 1)
	term.Backspace()
	tm4SExpectCursor(t, term, 1, 1, false)
	term.Backspace()
	tm4SExpectCursor(t, term, 0, 1, false)
	// Column zero without reverse wrap: nothing happens.
	term.Backspace()
	tm4SExpectCursor(t, term, 0, 1, false)
}

// Zig: test "term51: backspace: ignores margins and clears pending wrap"
func TestTm4_backspace_ignores_margins_and_clears_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(10, 3))
	defer term.Deinit()
	term.scrolling_region.left = 4
	term.scrolling_region.right = 8

	// Moves across the left margin without stopping.
	term.screens.active.CursorAbsolute(4, 0)
	term.Backspace()
	tm4SExpectCursor(t, term, 3, 0, false)

	// Pending wrap: moves from the last column (not decremented) in the
	// default mode.
	term.screens.active.CursorAbsolute(9, 1)
	term.screens.active.cursor.pending_wrap = true
	term.Backspace()
	tm4SExpectCursor(t, term, 8, 1, false)
}

// Zig: test "term51: backspace: with reverse wrap and pending wrap only clears the pending wrap"
func TestTm4_backspace_with_reverse_wrap_and_pending_wrap_only_clears_the_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 3))
	defer term.Deinit()
	term.modes.Set(ModesModeReverseWrap, true)

	zigTry(t, term.PrintString([]uint8("abcde")))
	tm4SExpectCursor(t, term, 4, 0, true)
	term.Backspace()
	tm4SExpectCursor(t, term, 4, 0, false)
	term.Backspace()
	tm4SExpectCursor(t, term, 3, 0, false)
}

// Zig: test "term51: backspace: reverse wrap goes to the end of a soft-wrapped previous line"
func TestTm4_backspace_reverse_wrap_goes_to_the_end_of_a_soft_wrapped_previous_line(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 3))
	defer term.Deinit()
	term.modes.Set(ModesModeReverseWrap, true)

	zigTry(t, term.PrintString([]uint8("abcdefg")))
	tm4SExpectCursor(t, term, 2, 1, false)
	term.CursorLeft(2)
	tm4SExpectCursor(t, term, 0, 1, false)
	term.Backspace()
	tm4SExpectCursor(t, term, 4, 0, false)
}

// Zig: test "term51: cursorUp: moves up and treats zero as one"
func TestTm4_cursorUp_moves_up_and_treats_zero_as_one(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 10))
	defer term.Deinit()

	term.screens.active.CursorAbsolute(3, 8)
	term.CursorUp(2)
	tm4SExpectCursor(t, term, 3, 6, false)
	term.CursorUp(0)
	tm4SExpectCursor(t, term, 3, 5, false)
	term.CursorUp(1)
	tm4SExpectCursor(t, term, 3, 4, false)
	// Larger than the distance to the top clamps.
	term.CursorUp(100)
	tm4SExpectCursor(t, term, 3, 0, false)
	term.CursorUp(1)
	tm4SExpectCursor(t, term, 3, 0, false)
	term.screens.active.CursorAbsolute(3, 9)
	term.CursorUp(math.MaxUint)
	tm4SExpectCursor(t, term, 3, 0, false)
	term.screens.active.CursorAbsolute(3, 9)
	term.CursorUp(9)
	tm4SExpectCursor(t, term, 3, 0, false)
	term.screens.active.CursorAbsolute(3, 9)
	term.CursorUp(10)
	tm4SExpectCursor(t, term, 3, 0, false)
}

// Zig: test "term51: cursorUp: stops at the top margin when inside the region"
func TestTm4_cursorUp_stops_at_the_top_margin_when_inside_the_region(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 10))
	defer term.Deinit()
	term.scrolling_region.top = 2
	term.scrolling_region.bottom = 6

	term.screens.active.CursorAbsolute(1, 5)
	term.CursorUp(2)
	tm4SExpectCursor(t, term, 1, 3, false)
	term.CursorUp(100)
	tm4SExpectCursor(t, term, 1, 2, false)
	// At the top margin: no movement.
	term.CursorUp(1)
	tm4SExpectCursor(t, term, 1, 2, false)

	// Exactly the distance to the margin.
	term.screens.active.CursorAbsolute(1, 6)
	term.CursorUp(4)
	tm4SExpectCursor(t, term, 1, 2, false)
}

// Zig: test "term51: cursorUp: above the top margin goes to the top of the screen"
func TestTm4_cursorUp_above_the_top_margin_goes_to_the_top_of_the_screen(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 10))
	defer term.Deinit()
	term.scrolling_region.top = 3
	term.scrolling_region.bottom = 6

	term.screens.active.CursorAbsolute(1, 2)
	term.CursorUp(1)
	tm4SExpectCursor(t, term, 1, 1, false)
	term.CursorUp(50)
	tm4SExpectCursor(t, term, 1, 0, false)
	term.CursorUp(1)
	tm4SExpectCursor(t, term, 1, 0, false)
}

// Zig: test "term51: cursorUp: always clears pending wrap and ignores origin mode"
func TestTm4_cursorUp_always_clears_pending_wrap_and_ignores_origin_mode(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 10))
	defer term.Deinit()

	term.screens.active.CursorAbsolute(4, 0)
	term.screens.active.cursor.pending_wrap = true
	// No movement possible, still clears.
	term.CursorUp(1)
	tm4SExpectCursor(t, term, 4, 0, false)

	term.scrolling_region.top = 2
	term.scrolling_region.bottom = 6
	term.modes.Set(ModesModeOrigin, true)
	term.screens.active.CursorAbsolute(4, 5)
	term.screens.active.cursor.pending_wrap = true
	term.CursorUp(1)
	tm4SExpectCursor(t, term, 4, 4, false)
}

// Zig: test "term51: cursorDown: moves down and treats zero as one"
func TestTm4_cursorDown_moves_down_and_treats_zero_as_one(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 10))
	defer term.Deinit()

	term.screens.active.CursorAbsolute(2, 1)
	term.CursorDown(2)
	tm4SExpectCursor(t, term, 2, 3, false)
	term.CursorDown(0)
	tm4SExpectCursor(t, term, 2, 4, false)
	term.CursorDown(1)
	tm4SExpectCursor(t, term, 2, 5, false)
	term.CursorDown(100)
	tm4SExpectCursor(t, term, 2, 9, false)
	term.CursorDown(1)
	tm4SExpectCursor(t, term, 2, 9, false)
	term.screens.active.CursorAbsolute(2, 0)
	term.CursorDown(math.MaxUint)
	tm4SExpectCursor(t, term, 2, 9, false)
	term.screens.active.CursorAbsolute(2, 0)
	term.CursorDown(9)
	tm4SExpectCursor(t, term, 2, 9, false)
	term.screens.active.CursorAbsolute(2, 0)
	term.CursorDown(10)
	tm4SExpectCursor(t, term, 2, 9, false)
}

// Zig: test "term51: cursorDown: stops at the bottom margin when inside the region"
func TestTm4_cursorDown_stops_at_the_bottom_margin_when_inside_the_region(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 10))
	defer term.Deinit()
	term.scrolling_region.top = 2
	term.scrolling_region.bottom = 6

	term.screens.active.CursorAbsolute(1, 3)
	term.CursorDown(2)
	tm4SExpectCursor(t, term, 1, 5, false)
	term.CursorDown(100)
	tm4SExpectCursor(t, term, 1, 6, false)
	// At the bottom margin: no movement.
	term.CursorDown(1)
	tm4SExpectCursor(t, term, 1, 6, false)

	// Exactly the distance to the margin.
	term.screens.active.CursorAbsolute(1, 2)
	term.CursorDown(4)
	tm4SExpectCursor(t, term, 1, 6, false)

	// Above the region, inside the check y <= bottom applies.
	term.screens.active.CursorAbsolute(1, 0)
	term.CursorDown(100)
	tm4SExpectCursor(t, term, 1, 6, false)
}

// Zig: test "term51: cursorDown: below the bottom margin goes to the last row"
func TestTm4_cursorDown_below_the_bottom_margin_goes_to_the_last_row(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 10))
	defer term.Deinit()
	term.scrolling_region.top = 2
	term.scrolling_region.bottom = 6

	term.screens.active.CursorAbsolute(1, 7)
	term.CursorDown(1)
	tm4SExpectCursor(t, term, 1, 8, false)
	term.CursorDown(50)
	tm4SExpectCursor(t, term, 1, 9, false)
	term.CursorDown(1)
	tm4SExpectCursor(t, term, 1, 9, false)
}

// Zig: test "term51: cursorDown: always clears pending wrap and ignores origin mode"
func TestTm4_cursorDown_always_clears_pending_wrap_and_ignores_origin_mode(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(5, 10))
	defer term.Deinit()

	term.screens.active.CursorAbsolute(4, 9)
	term.screens.active.cursor.pending_wrap = true
	term.CursorDown(1)
	tm4SExpectCursor(t, term, 4, 9, false)

	term.scrolling_region.top = 2
	term.scrolling_region.bottom = 6
	term.modes.Set(ModesModeOrigin, true)
	term.screens.active.CursorAbsolute(4, 3)
	term.screens.active.cursor.pending_wrap = true
	term.CursorDown(1)
	tm4SExpectCursor(t, term, 4, 4, false)
}

// Zig: test "term51: cursorRight: moves right and treats zero as one"
func TestTm4_cursorRight_moves_right_and_treats_zero_as_one(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(10, 3))
	defer term.Deinit()

	term.screens.active.CursorAbsolute(1, 1)
	term.CursorRight(2)
	tm4SExpectCursor(t, term, 3, 1, false)
	term.CursorRight(0)
	tm4SExpectCursor(t, term, 4, 1, false)
	term.CursorRight(1)
	tm4SExpectCursor(t, term, 5, 1, false)
	term.CursorRight(100)
	tm4SExpectCursor(t, term, 9, 1, false)
	term.CursorRight(1)
	tm4SExpectCursor(t, term, 9, 1, false)
	term.screens.active.CursorAbsolute(0, 1)
	term.CursorRight(math.MaxUint)
	tm4SExpectCursor(t, term, 9, 1, false)
	term.screens.active.CursorAbsolute(0, 1)
	term.CursorRight(9)
	tm4SExpectCursor(t, term, 9, 1, false)
	term.screens.active.CursorAbsolute(0, 1)
	term.CursorRight(10)
	tm4SExpectCursor(t, term, 9, 1, false)
}

// Zig: test "term51: cursorRight: stops at the right margin when at or left of it"
func TestTm4_cursorRight_stops_at_the_right_margin_when_at_or_left_of_it(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(10, 3))
	defer term.Deinit()
	term.scrolling_region.left = 2
	term.scrolling_region.right = 6

	term.screens.active.CursorAbsolute(3, 1)
	term.CursorRight(2)
	tm4SExpectCursor(t, term, 5, 1, false)
	term.CursorRight(100)
	tm4SExpectCursor(t, term, 6, 1, false)
	term.CursorRight(1)
	tm4SExpectCursor(t, term, 6, 1, false)

	// Left of the left margin also stops at the right margin.
	term.screens.active.CursorAbsolute(0, 1)
	term.CursorRight(100)
	tm4SExpectCursor(t, term, 6, 1, false)

	// Exactly the distance.
	term.screens.active.CursorAbsolute(2, 1)
	term.CursorRight(4)
	tm4SExpectCursor(t, term, 6, 1, false)
}

// Go-only: panic test: invokeCharset: single shift requires the GL active slot
func TestTm4_panic_invokeCharset_single_shift_requires_GL(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(4, 2))
	defer term.Deinit()

	// Zig: `if (single) { assert(active == .GL); ... }`.
	tm4ExpectPanic(t, func() {
		term.InvokeCharset(CharsetsActiveSlotGR, CharsetsSlotsG1, true)
	})
}

// Go-only: panic test: printCell: overwriting a spacer tail in column zero
func TestTm4_panic_printCell_spacer_tail_at_column_zero(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(4, 2))
	defer term.Deinit()

	// Zig: in the `.spacer_tail` branch, `assert(self.screens.active.cursor.x > 0)`.
	// A spacer tail in column zero is an invalid state that a test makes by
	// hand.
	term.screens.active.cursor.page_cell.wide = PageCellWideSpacerTail
	tm4ExpectPanic(t, func() {
		term.printCell('a', PageCellWideNarrow)
	})
}

// Go-only: panic test: printCell: a styled cell in a row not flagged as styled
func TestTm4_panic_printCell_styled_cell_in_unstyled_row(t *testing.T) {
	testStdCapacity(t)
	term := tm4Init(t, terminalOptionsDefault(4, 2))
	defer term.Deinit()

	// Zig: `if (cell.style_id != style.default_id) { assert(
	// self.screens.active.cursor.page_row.styled); ... }` when the style of
	// the cell differs from the cursor's. The cell is given a style id by
	// hand while the row has no styled flag.
	term.screens.active.cursor.page_cell.style_id = 1
	expect(t, !term.screens.active.cursor.page_row.styled, "!t.screens.active.cursor.page_row.styled")
	tm4ExpectPanic(t, func() {
		term.printCell('a', PageCellWideNarrow)
	})
}
