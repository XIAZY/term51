// Translated Zig tests of src/terminal/Terminal.zig, from
// "Terminal: deleteChars split wide character from end" (line 10447) up to and
// including "term51: printRepeat: previous char survives cursor movement and is
// not set by zero-width input" (line 13617). Helpers carry the prefix tm3.
//
// Conventions (see docs/mapping.md):
//   - `var t = try init(alloc, .{ .cols = c, .rows = r })` is
//     `tm := tm3Init(t, terminalOptionsDefault(c, r))`; the Go terminal is
//     called `tm` because `t` is the *testing.T. Zig's `.{ .rows = r, .cols = c }`
//     fields not named stay at the Zig defaults the helper writes out.
//   - `try testing.expectEqualStrings(x, str)` after `plainString` is
//     `expectEqualStrings(t, x, tm3Plain(t, &tm))`; the allocator and the
//     `defer free(str)` are dropped.
//   - `for ("ABC") |c| try t.print(c)` is `tm3PrintEach(t, &tm, "ABC")`.
//   - A `defer` inside a Zig block runs at the end of the block: where the
//     block has one, the Go block is a function literal with the same defer.
//   - A Zig `.{ .active = .{ .x = x, .y = y } }` point is `tm3Active(x, y)`.
//   - Allocator and leak checks are dropped with `// Zig-only:` comments.
//   - testStdCapacity(t) is called first in every test (Zig's unit tests run
//     with grapheme_bytes 512).

package terminal

import (
	"errors"
	"math"
	"testing"
)

// tm3ExpectPanic runs f and fails the test unless f panics. A panic with the
// wireframe's "not implemented" message does not count: the panic has to be
// the one the Zig code has on the path.
func tm3ExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("expected a panic, got none")
		}
		if s, ok := r.(string); ok && s == notImplemented {
			t.Fatalf("expected the Zig panic, got the wireframe's %q", notImplemented)
		}
	}()
	f()
}

// tm3ExpectError is Zig's `try testing.expectError(want, got)`.
func tm3ExpectError(t *testing.T, want error, got error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

// tm3Init is `try init(alloc, opts)`.
func tm3Init(t *testing.T, opts TerminalOptions) Terminal {
	t.Helper()
	tm, err := TerminalInit(opts)
	zigTry(t, err)
	return tm
}

// tm3Active is the point `.{ .active = .{ .x = x, .y = y } }`.
func tm3Active(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagActive, Active: PointCoordinate{X: x, Y: y}}
}

// tm3Screen is the point `.{ .screen = .{ .x = x, .y = y } }`.
func tm3Screen(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagScreen, Screen: PointCoordinate{X: x, Y: y}}
}

// tm3GetCell is `t.screens.active.pages.getCell(pt).?`.
func tm3GetCell(t *testing.T, tm *Terminal, pt Point) PageListCell {
	t.Helper()
	return zigUnwrap(t, tm.Screens.Active.Pages.GetCell(pt))
}

// tm3At is the page list cell at the given active-area position:
// `t.screens.active.pages.getCell(.{ .active = .{ .x = x, .y = y } }).?`.
func tm3At(t *testing.T, tm *Terminal, x uint, y uint) PageListCell {
	t.Helper()
	return tm3GetCell(t, tm, tm3Active(SizeCellCountInt(x), uint32(y)))
}

// tm3Plain is `try t.plainString(alloc)` as a string.
func tm3Plain(t *testing.T, tm *Terminal) string {
	t.Helper()
	str, err := tm.PlainString()
	zigTry(t, err)
	return string(str)
}

// tm3PlainUnwrapped is `try t.plainStringUnwrapped(alloc)` as a string.
func tm3PlainUnwrapped(t *testing.T, tm *Terminal) string {
	t.Helper()
	str, err := tm.PlainStringUnwrapped()
	zigTry(t, err)
	return string(str)
}

// tm3Print is `try t.print(c)`.
func tm3Print(t *testing.T, tm *Terminal, c uint32) {
	t.Helper()
	zigTry(t, tm.Print(c))
}

// tm3PrintEach is `for (s) |c| try t.print(c)`.
func tm3PrintEach(t *testing.T, tm *Terminal, s string) {
	t.Helper()
	for _, c := range []byte(s) {
		zigTry(t, tm.Print(uint32(c)))
	}
}

// tm3PrintString is `try t.printString(s)`.
func tm3PrintString(t *testing.T, tm *Terminal, s string) {
	t.Helper()
	zigTry(t, tm.PrintString([]uint8(s)))
}

// tm3Linefeed is `try t.linefeed()`.
func tm3Linefeed(t *testing.T, tm *Terminal) {
	t.Helper()
	zigTry(t, tm.Linefeed())
}

// tm3SetAttribute is `try t.setAttribute(attr)`.
func tm3SetAttribute(t *testing.T, tm *Terminal, attr SGRAttribute) {
	t.Helper()
	zigTry(t, tm.SetAttribute(attr))
}

// tm3Attr is a payload-less attribute such as `.bold` or `.unset`.
func tm3Attr(tag SGRAttributeTag) SGRAttribute {
	return SGRAttribute{Tag: tag}
}

// tm3DirectBg is `.{ .direct_color_bg = .{ .r = r, .g = g, .b = b } }`.
func tm3DirectBg(r uint8, g uint8, b uint8) SGRAttribute {
	return SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: r, G: g, B: b}}
}

// tm3DirectFg is `.{ .direct_color_fg = .{ .r = r, .g = g, .b = b } }`.
func tm3DirectFg(r uint8, g uint8, b uint8) SGRAttribute {
	return SGRAttribute{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: ColorRGB{R: r, G: g, B: b}}
}

// tm3Prompt is a semantic prompt command `.{ .action = a, .options_unvalidated = opts }`.
func tm3Prompt(action OSCParsersSemanticPromptCommandAction, opts string) OSCCommandSemanticPrompt {
	return OSCCommandSemanticPrompt{Action: action, OptionsUnvalidated: []uint8(opts)}
}

// tm3SemanticPrompt is `try t.semanticPrompt(cmd)`.
func tm3SemanticPrompt(t *testing.T, tm *Terminal, cmd OSCCommandSemanticPrompt) {
	t.Helper()
	zigTry(t, tm.SemanticPrompt(cmd))
}

// tm3SwitchScreenMode is `try t.switchScreenMode(mode, enabled)`.
func tm3SwitchScreenMode(t *testing.T, tm *Terminal, mode TerminalSwitchScreenMode, enabled bool) {
	t.Helper()
	zigTry(t, tm.SwitchScreenMode(mode, enabled))
}

// tm3Resize is `try t.resize(alloc, cols, rows)`.
func tm3Resize(t *testing.T, tm *Terminal, cols SizeCellCountInt, rows SizeCellCountInt) {
	t.Helper()
	zigTry(t, tm.Resize(cols, rows))
}

// tm3Deccolm is `try t.deccolm(alloc, mode)`.
func tm3Deccolm(t *testing.T, tm *Terminal, mode TerminalDeccolmMode) {
	t.Helper()
	zigTry(t, tm.Deccolm(mode))
}

// tm3Dirty is `t.isDirty(.{ .active = .{ .x = 0, .y = y } })`.
func tm3Dirty(tm *Terminal, y uint32) bool {
	return tm.isDirty(tm3Active(0, y))
}

// tm3ExpectBgRGB is the pair of assertions that a cell is a background color
// cell of the given color.
func tm3ExpectBgRGB(t *testing.T, cell *PageCell, r uint8, g uint8, b uint8) {
	t.Helper()
	expect(t, cell.content_tag == PageCellContentTagBgColorRgb, "list_cell.cell.content_tag == .bg_color_rgb")
	expectEqual(t, PageCellRGB{R: r, G: g, B: b}, cell.content.color_rgb)
}

// tm3SAt is term51SAt: the page list cell at the given active-area position.
func tm3SAt(t *testing.T, tm *Terminal, x uint, y uint) PageListCell {
	t.Helper()
	return tm3At(t, tm, x, y)
}

// tm3SExpectCell is term51SExpectCell: expect the cell at (x, y) to hold
// exactly the codepoint and wide state.
func tm3SExpectCell(t *testing.T, tm *Terminal, x uint, y uint, cp uint32, wide PageCellWide) {
	t.Helper()
	lc := tm3SAt(t, tm, x, y)
	expectEqual(t, cp, lc.Cell.Codepoint())
	expectEqual(t, wide, lc.Cell.wide)
}

// tm3SExpectGrapheme is term51SExpectGrapheme: expect the cell at (x, y) to
// carry exactly this grapheme tail.
func tm3SExpectGrapheme(t *testing.T, tm *Terminal, x uint, y uint, expected []uint32) {
	t.Helper()
	lc := tm3SAt(t, tm, x, y)
	expect(t, lc.Cell.HasGrapheme(), "lc.cell.hasGrapheme()")
	expect(t, lc.Row.grapheme, "lc.row.grapheme")
	expectEqualSlices(t, expected, zigUnwrap(t, lc.Node.data.LookupGrapheme(lc.Cell)))
}

// tm3SExpectCursor is term51SExpectCursor: expect the cursor position and
// pending wrap state.
func tm3SExpectCursor(t *testing.T, tm *Terminal, x uint, y uint, pending_wrap bool) {
	t.Helper()
	expectEqual(t, x, uint(tm.Screens.Active.Cursor.X))
	expectEqual(t, y, uint(tm.Screens.Active.Cursor.Y))
	expectEqual(t, pending_wrap, tm.Screens.Active.Cursor.PendingWrap)
}

// tm3SExpectPlain is term51SExpectPlain: expect the plain string of the
// viewport.
func tm3SExpectPlain(t *testing.T, tm *Terminal, expected string) {
	t.Helper()
	expectEqualStrings(t, expected, tm3Plain(t, tm))
}

// Zig: test "Terminal: deleteChars split wide character from end"
func TestTm3_Terminal_deleteChars_split_wide_character_from_end(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(6, 10))
	defer tm.Deinit()

	tm3PrintString(t, &tm, "A橋123")
	tm.SetCursorPos(1, 1)
	tm.DeleteChars(1)

	{
		cell := tm3GetCell(t, &tm, tm3Screen(0, 0)).Cell
		expectEqual(t, uint32(0x6A4B), cell.content.codepoint)
		expectEqual(t, PageCellWideWide, cell.wide)
	}
	{
		cell := tm3GetCell(t, &tm, tm3Screen(1, 0)).Cell
		expectEqual(t, uint32(0), cell.content.codepoint)
		expectEqual(t, PageCellWideSpacerTail, cell.wide)
	}
}

// Zig: test "Terminal: deleteChars with a spacer head at the end"
func TestTm3_Terminal_deleteChars_with_a_spacer_head_at_the_end(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 10))
	defer tm.Deinit()

	tm3PrintString(t, &tm, "0123橋123")
	{
		list_cell := tm3GetCell(t, &tm, tm3Screen(4, 0))
		row := list_cell.Row
		cell := list_cell.Cell
		expectEqual(t, PageCellWideSpacerHead, cell.wide)
		expect(t, row.wrap, "row.wrap")
	}

	tm.SetCursorPos(1, 1)
	tm.DeleteChars(1)

	{
		cell := tm3GetCell(t, &tm, tm3Screen(3, 0)).Cell
		expectEqual(t, uint32(0), cell.content.codepoint)
		expectEqual(t, PageCellWideNarrow, cell.wide)
	}
}

// Zig: test "Terminal: deleteChars split wide character tail"
func TestTm3_Terminal_deleteChars_split_wide_character_tail(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm.SetCursorPos(1, uint(tm.Cols)-1)
	tm3Print(t, &tm, 0x6A4B) // 橋
	tm.CarriageReturn()
	tm.DeleteChars(uint(tm.Cols) - 1)
	tm3Print(t, &tm, '0')

	{
		expectEqualStrings(t, "0", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: deleteChars wide char boundary conditions"
func TestTm3_Terminal_deleteChars_wide_char_boundary_conditions(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(8, 1))
	defer tm.Deinit()

	// EXPLANATION(qwerasd):
	//
	// There are 3 or 4 boundaries to be concerned with in deleteChars,
	// depending on how you count them. Consider the following terminal:
	//
	//   +--------+
	// 0 |.ABCDEF.|
	//   : ^      : (^ = cursor)
	//   +--------+
	//
	// if we DCH 3 we get
	//
	//   +--------+
	// 0 |.DEF....|
	//   +--------+
	//
	// The boundaries exist at the following points then:
	//
	//   +--------+
	// 0 |.ABCDEF.|
	//   :11 22 33:
	//   +--------+
	//
	// I'm counting 2 for double since it's both the end of the deleted
	// content and the start of the content that is shifted in to place.
	//
	// Now consider wide characters (represented as `WW`) at these boundaries:
	//
	//   +--------+
	// 0 |WWaWWbWW|
	//   : ^      : (^ = cursor)
	//   : ^^^    : (^ = deleted by DCH 3)
	//   +--------+
	//
	// -> DCH 3
	// -> The first 2 wide characters are split & destroyed (verified in xterm)
	//
	//   +--------+
	// 0 |..bWW...|
	//   +--------+

	tm3PrintString(t, &tm, "😀a😀b😀")
	{
		expectEqualStrings(t, "😀a😀b😀", tm3Plain(t, &tm))
	}

	tm.SetCursorPos(1, 2)
	tm.DeleteChars(3)
	tm.Screens.Active.Cursor.PagePin.Node.data.AssertIntegrity()

	{
		expectEqualStrings(t, "  b😀", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: deleteChars wide char wrap boundary conditions"
func TestTm3_Terminal_deleteChars_wide_char_wrap_boundary_conditions(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(8, 3))
	defer tm.Deinit()

	// EXPLANATION(qwerasd):
	// (cont. from "Terminal: deleteChars wide char boundary conditions")
	//
	// Additionally consider soft-wrapped wide chars (`H` = spacer head):
	//
	//   +--------+
	// 0 |.......H…
	// 1 …WWabcdeH…
	//   : ^      : (^ = cursor)
	//   : ^^^    : (^ = deleted by DCH 3)
	// 2 …WW......|
	//   +--------+
	//
	// -> DCH 3
	// -> First wide character split and destroyed, including spacer head,
	//    second spacer head removed (verified in xterm).
	// -> Wrap state of row reset
	//
	//   +--------+
	// 0 |........|
	// 1 |.cde....|
	// 2 |WW......|
	//   +--------+
	//

	tm3PrintString(t, &tm, ".......😀abcde😀......")
	{
		expectEqualStrings(t, ".......\n😀abcde\n😀......", tm3Plain(t, &tm))
		expectEqualStrings(t, ".......😀abcde😀......", tm3PlainUnwrapped(t, &tm))
	}

	tm.SetCursorPos(2, 2)
	tm.DeleteChars(3)
	tm.Screens.Active.Cursor.PagePin.Node.data.AssertIntegrity()

	{
		expectEqualStrings(t, ".......\n cde\n😀......", tm3Plain(t, &tm))
		expectEqualStrings(t, ".......  cde\n😀......", tm3PlainUnwrapped(t, &tm))
	}
}

// Zig: test "Terminal: deleteChars wide char across right margin"
func TestTm3_Terminal_deleteChars_wide_char_across_right_margin(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(8, 3))
	defer tm.Deinit()

	// scroll region
	//    VVVVVV
	//  +-######-+
	//  |.abcdeWW|
	//  : ^      : (^ = cursor)
	//  +--------+
	//
	// DCH 1

	tm3PrintString(t, &tm, "123456橋")
	tm.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	tm.SetLeftAndRightMargin(2, 7)

	{
		expectEqualStrings(t, "123456橋", tm3Plain(t, &tm))
	}

	tm.SetCursorPos(1, 2)
	tm.DeleteChars(1)
	tm.Screens.Active.Cursor.PagePin.Node.data.AssertIntegrity()

	// NOTE: This behavior is slightly inconsistent with xterm. xterm
	// _visually_ splits the wide character (half the wide character shows
	// up in col 6 and half in col 8). In all other wide char split scenarios,
	// xterm clears the cell. Therefore, we've chosen to clear the cell here.
	// Given we have space, we also could actually preserve it, but I haven't
	// yet found a terminal that behaves that way. We should be open to
	// revisiting this behavior but for now we're going with the simpler
	// impl.
	{
		expectEqualStrings(t, "13456", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: saveCursor"
func TestTm3_Terminal_saveCursor(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(3, 3))
	defer tm.Deinit()

	tm3SetAttribute(t, &tm, tm3Attr(SGRAttributeTagBold))
	tm.Screens.Active.Charset.Gr = CharsetsSlotsG3
	tm.Modes.Set(ModesModeOrigin, true)
	tm.SaveCursor()
	tm.Screens.Active.Charset.Gr = CharsetsSlotsG0
	tm3SetAttribute(t, &tm, tm3Attr(SGRAttributeTagUnset))
	tm.Modes.Set(ModesModeOrigin, false)
	tm.RestoreCursor()
	expect(t, tm.Screens.Active.Cursor.Style.Flags.Bold, "t.screens.active.cursor.style.flags.bold")
	expect(t, tm.Screens.Active.Charset.Gr == CharsetsSlotsG3, "t.screens.active.charset.gr == .G3")
	expect(t, tm.Modes.Get(ModesModeOrigin), "t.modes.get(.origin)")
}

// Zig: test "Terminal: saveCursor position"
func TestTm3_Terminal_saveCursor_position(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	tm.SetCursorPos(1, 5)
	tm3Print(t, &tm, 'A')
	tm.SaveCursor()
	tm.SetCursorPos(1, 1)
	tm3Print(t, &tm, 'B')
	tm.RestoreCursor()
	tm3Print(t, &tm, 'X')

	{
		expectEqualStrings(t, "B   AX", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: saveCursor pending wrap state"
func TestTm3_Terminal_saveCursor_pending_wrap_state(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm.SetCursorPos(1, 5)
	tm3Print(t, &tm, 'A')
	tm.SaveCursor()
	tm.SetCursorPos(1, 1)
	tm3Print(t, &tm, 'B')
	tm.RestoreCursor()
	tm3Print(t, &tm, 'X')

	{
		expectEqualStrings(t, "B   A\nX", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: saveCursor origin mode"
func TestTm3_Terminal_saveCursor_origin_mode(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	tm.Modes.Set(ModesModeOrigin, true)
	tm.SaveCursor()
	tm.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	tm.SetLeftAndRightMargin(3, 5)
	tm.SetTopAndBottomMargin(2, 4)
	tm.RestoreCursor()
	tm3Print(t, &tm, 'X')

	{
		expectEqualStrings(t, "X", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: saveCursor resize"
func TestTm3_Terminal_saveCursor_resize(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	tm.SetCursorPos(1, 10)
	tm.SaveCursor()
	tm3Resize(t, &tm, 5, 5)
	tm.RestoreCursor()
	tm3Print(t, &tm, 'X')

	{
		expectEqualStrings(t, "    X", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: saveCursor protected pen"
func TestTm3_Terminal_saveCursor_protected_pen(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	tm.SetProtectedMode(ANSIProtectedModeIso)
	expect(t, tm.Screens.Active.Cursor.Protected, "t.screens.active.cursor.protected")
	tm.SetCursorPos(1, 10)
	tm.SaveCursor()
	tm.SetProtectedMode(ANSIProtectedModeOff)
	expect(t, !tm.Screens.Active.Cursor.Protected, "!t.screens.active.cursor.protected")
	tm.RestoreCursor()
	expect(t, tm.Screens.Active.Cursor.Protected, "t.screens.active.cursor.protected")
}

// Zig: test "Terminal: saveCursor doesn't modify hyperlink state"
func TestTm3_Terminal_saveCursor_doesn_t_modify_hyperlink_state(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(3, 3))
	defer tm.Deinit()

	zigTry(t, tm.Screens.Active.StartHyperlink([]uint8("http://example.com"), None[[]uint8]()))
	id := tm.Screens.Active.Cursor.HyperlinkID
	tm.SaveCursor()
	expectEqual(t, id, tm.Screens.Active.Cursor.HyperlinkID)
	tm.RestoreCursor()
	expectEqual(t, id, tm.Screens.Active.Cursor.HyperlinkID)
}

// Zig: test "Terminal: restoreCursor uses default style on OutOfSpace"
func TestTm3_Terminal_restoreCursor_uses_default_style_on_OutOfSpace(t *testing.T) {
	testStdCapacity(t)
	// Tests that restoreCursor falls back to default style when
	// manualStyleUpdate fails with OutOfSpace (can't split a 1-row page
	// and styles are at max capacity).

	// Use a single row so the page can't be split
	tm := tm3Init(t, terminalOptionsDefault(10, 1))
	defer tm.Deinit()

	// Set a style and save the cursor
	tm3SetAttribute(t, &tm, tm3Attr(SGRAttributeTagBold))
	tm.SaveCursor()

	// Clear the style
	tm3SetAttribute(t, &tm, tm3Attr(SGRAttributeTagUnset))
	expect(t, !tm.Screens.Active.Cursor.Style.Flags.Bold, "!t.screens.active.cursor.style.flags.bold")

	// Fill the style map to max capacity
	const max_styles = math.MaxUint16 // std.math.maxInt(size.CellCountInt)
	for tm.Screens.Active.Cursor.PagePin.Node.data.capacity.styles < max_styles {
		_, err := tm.Screens.Active.IncreaseCapacity(
			tm.Screens.Active.Cursor.PagePin.Node,
			Some(PageListIncreaseCapacityStyles),
		)
		if err != nil {
			break
		}
	}

	page := &tm.Screens.Active.Cursor.PagePin.Node.data
	expectEqual(t, SizeStyleCountInt(max_styles), page.capacity.styles)

	// Fill all style slots using the StyleSet's layout capacity which accounts
	// for the load factor. The capacity in the layout is the actual max number
	// of items that can be stored.
	func() {
		page.PauseIntegrityChecks(true)
		defer page.PauseIntegrityChecks(false)
		defer page.AssertIntegrity()

		max_items := page.styles.layout.cap
		for n := uint(1); n < max_items; n++ {
			_, err := page.styles.Add(
				// Zig: .{ .bg_color = .{ .rgb = @bitCast(@as(u24, @intCast(n))) } }
				Style{BgColor: StyleColor{
					Tag: StyleColorRgb,
					Rgb: ColorRGB{R: uint8(n), G: uint8(n >> 8), B: uint8(n >> 16)},
				}},
			)
			if err != nil {
				break
			}
		}
	}()

	// Restore cursor - should fall back to default style since page
	// can't be split (1 row) and styles are at max capacity
	tm.RestoreCursor()

	// The style should be reset to default because OutOfSpace occurred
	expect(t, !tm.Screens.Active.Cursor.Style.Flags.Bold, "!t.screens.active.cursor.style.flags.bold")
	expectEqual(t, StyleDefaultID, tm.Screens.Active.Cursor.StyleID)
}

// Zig: test "Terminal: setProtectedMode"
func TestTm3_Terminal_setProtectedMode(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(3, 3))
	defer tm.Deinit()

	expect(t, !tm.Screens.Active.Cursor.Protected, "!t.screens.active.cursor.protected")
	tm.SetProtectedMode(ANSIProtectedModeOff)
	expect(t, !tm.Screens.Active.Cursor.Protected, "!t.screens.active.cursor.protected")
	tm.SetProtectedMode(ANSIProtectedModeIso)
	expect(t, tm.Screens.Active.Cursor.Protected, "t.screens.active.cursor.protected")
	tm.SetProtectedMode(ANSIProtectedModeDec)
	expect(t, tm.Screens.Active.Cursor.Protected, "t.screens.active.cursor.protected")
	tm.SetProtectedMode(ANSIProtectedModeOff)
	expect(t, !tm.Screens.Active.Cursor.Protected, "!t.screens.active.cursor.protected")
}

// Zig: test "Terminal: eraseLine simple erase right"
func TestTm3_Terminal_eraseLine_simple_erase_right(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm3PrintEach(t, &tm, "ABCDE")
	tm.SetCursorPos(1, 3)
	tm.clearDirty()
	tm.EraseLine(CSIEraseLineRight, false)
	expect(t, tm3Dirty(&tm, 0), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		expectEqualStrings(t, "AB", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseLine resets pending wrap"
func TestTm3_Terminal_eraseLine_resets_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm3PrintEach(t, &tm, "ABCDE")
	expect(t, tm.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	tm.EraseLine(CSIEraseLineRight, false)
	expect(t, !tm.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
	tm3Print(t, &tm, 'B')

	{
		expectEqualStrings(t, "ABCDB", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseLine resets wrap"
func TestTm3_Terminal_eraseLine_resets_wrap(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm3PrintEach(t, &tm, "ABCDE123")
	{
		list_cell := tm3At(t, &tm, 0, 0)
		expect(t, list_cell.Row.wrap, "list_cell.row.wrap")
	}

	tm.SetCursorPos(1, 1)
	tm.EraseLine(CSIEraseLineRight, false)

	{
		list_cell := tm3At(t, &tm, 0, 0)
		expect(t, !list_cell.Row.wrap, "!list_cell.row.wrap")
	}
	tm3Print(t, &tm, 'X')

	{
		expectEqualStrings(t, "X\n123", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseLine right preserves background sgr"
func TestTm3_Terminal_eraseLine_right_preserves_background_sgr(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm3PrintEach(t, &tm, "ABCDE")
	tm.SetCursorPos(1, 2)
	tm3SetAttribute(t, &tm, tm3DirectBg(0xFF, 0, 0))
	tm.EraseLine(CSIEraseLineRight, false)

	{
		expectEqualStrings(t, "A", tm3Plain(t, &tm))
		for x := uint(1); x < 5; x++ {
			list_cell := tm3At(t, &tm, x, 0)
			tm3ExpectBgRGB(t, list_cell.Cell, 0xFF, 0, 0)
		}
	}
}

// Zig: test "Terminal: eraseLine right wide character"
func TestTm3_Terminal_eraseLine_right_wide_character(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	tm3PrintEach(t, &tm, "AB")
	tm3Print(t, &tm, '橋')
	tm3PrintEach(t, &tm, "DE")
	tm.SetCursorPos(1, 4)
	tm.clearDirty()
	tm.EraseLine(CSIEraseLineRight, false)
	expect(t, tm3Dirty(&tm, 0), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		expectEqualStrings(t, "AB", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseLine right protected attributes respected with iso"
func TestTm3_Terminal_eraseLine_right_protected_attributes_respected_with_iso(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm.SetProtectedMode(ANSIProtectedModeIso)
	tm3PrintEach(t, &tm, "ABC")
	tm.SetCursorPos(1, 1)
	tm.clearDirty()
	tm.EraseLine(CSIEraseLineRight, false)
	expect(t, tm3Dirty(&tm, 0), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		expectEqualStrings(t, "ABC", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseLine right protected attributes ignored with dec most recent"
func TestTm3_Terminal_eraseLine_right_protected_attributes_ignored_with_dec_most_recent(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm.SetProtectedMode(ANSIProtectedModeIso)
	tm3PrintEach(t, &tm, "ABC")
	tm.SetProtectedMode(ANSIProtectedModeDec)
	tm.SetProtectedMode(ANSIProtectedModeOff)
	tm.SetCursorPos(1, 2)
	tm.clearDirty()
	tm.EraseLine(CSIEraseLineRight, false)
	expect(t, tm3Dirty(&tm, 0), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		expectEqualStrings(t, "A", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseLine right protected attributes ignored with dec set"
func TestTm3_Terminal_eraseLine_right_protected_attributes_ignored_with_dec_set(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm.SetProtectedMode(ANSIProtectedModeDec)
	tm3PrintEach(t, &tm, "ABC")
	tm.SetCursorPos(1, 2)
	tm.clearDirty()
	tm.EraseLine(CSIEraseLineRight, false)
	expect(t, tm3Dirty(&tm, 0), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		expectEqualStrings(t, "A", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseLine right protected requested"
func TestTm3_Terminal_eraseLine_right_protected_requested(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	tm3PrintEach(t, &tm, "12345678")
	tm.SetCursorPos(uint(tm.Screens.Active.Cursor.Y)+1, 6)
	tm.SetProtectedMode(ANSIProtectedModeDec)
	tm3Print(t, &tm, 'X')
	tm.SetCursorPos(uint(tm.Screens.Active.Cursor.Y)+1, 4)
	tm.clearDirty()
	tm.EraseLine(CSIEraseLineRight, true)
	expect(t, tm3Dirty(&tm, 0), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		expectEqualStrings(t, "123  X", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseLine simple erase left"
func TestTm3_Terminal_eraseLine_simple_erase_left(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm3PrintEach(t, &tm, "ABCDE")
	tm.SetCursorPos(1, 3)
	tm.clearDirty()
	tm.EraseLine(CSIEraseLineLeft, false)
	expect(t, tm3Dirty(&tm, 0), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		expectEqualStrings(t, "   DE", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseLine left resets wrap"
func TestTm3_Terminal_eraseLine_left_resets_wrap(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm3PrintEach(t, &tm, "ABCDE")
	expect(t, tm.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	tm.clearDirty()
	tm.EraseLine(CSIEraseLineLeft, false)
	expect(t, tm3Dirty(&tm, 0), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, !tm.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
	tm3Print(t, &tm, 'B')

	{
		expectEqualStrings(t, "    B", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseLine left preserves background sgr"
func TestTm3_Terminal_eraseLine_left_preserves_background_sgr(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm3PrintEach(t, &tm, "ABCDE")
	tm.SetCursorPos(1, 2)
	tm3SetAttribute(t, &tm, tm3DirectBg(0xFF, 0, 0))
	tm.EraseLine(CSIEraseLineLeft, false)

	{
		expectEqualStrings(t, "  CDE", tm3Plain(t, &tm))
		for x := uint(0); x < 2; x++ {
			list_cell := tm3At(t, &tm, x, 0)
			tm3ExpectBgRGB(t, list_cell.Cell, 0xFF, 0, 0)
		}
	}
}

// Zig: test "Terminal: eraseLine left wide character"
func TestTm3_Terminal_eraseLine_left_wide_character(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	tm3PrintEach(t, &tm, "AB")
	tm3Print(t, &tm, '橋')
	tm3PrintEach(t, &tm, "DE")
	tm.SetCursorPos(1, 3)
	tm.clearDirty()
	tm.EraseLine(CSIEraseLineLeft, false)
	expect(t, tm3Dirty(&tm, 0), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		expectEqualStrings(t, "    DE", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseLine left protected attributes respected with iso"
func TestTm3_Terminal_eraseLine_left_protected_attributes_respected_with_iso(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm.SetProtectedMode(ANSIProtectedModeIso)
	tm3PrintEach(t, &tm, "ABC")
	tm.SetCursorPos(1, 1)
	tm.clearDirty()
	tm.EraseLine(CSIEraseLineLeft, false)
	expect(t, tm3Dirty(&tm, 0), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		expectEqualStrings(t, "ABC", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseLine left protected attributes ignored with dec most recent"
func TestTm3_Terminal_eraseLine_left_protected_attributes_ignored_with_dec_most_recent(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm.SetProtectedMode(ANSIProtectedModeIso)
	tm3PrintEach(t, &tm, "ABC")
	tm.SetProtectedMode(ANSIProtectedModeDec)
	tm.SetProtectedMode(ANSIProtectedModeOff)
	tm.SetCursorPos(1, 2)
	tm.clearDirty()
	tm.EraseLine(CSIEraseLineLeft, false)
	expect(t, tm3Dirty(&tm, 0), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		expectEqualStrings(t, "  C", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseLine left protected attributes ignored with dec set"
func TestTm3_Terminal_eraseLine_left_protected_attributes_ignored_with_dec_set(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm.SetProtectedMode(ANSIProtectedModeDec)
	tm3PrintEach(t, &tm, "ABC")
	tm.SetCursorPos(1, 2)
	tm.clearDirty()
	tm.EraseLine(CSIEraseLineLeft, false)
	expect(t, tm3Dirty(&tm, 0), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		expectEqualStrings(t, "  C", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseLine left protected requested"
func TestTm3_Terminal_eraseLine_left_protected_requested(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	tm3PrintEach(t, &tm, "123456789")
	tm.SetCursorPos(uint(tm.Screens.Active.Cursor.Y)+1, 6)
	tm.SetProtectedMode(ANSIProtectedModeDec)
	tm3Print(t, &tm, 'X')
	tm.SetCursorPos(uint(tm.Screens.Active.Cursor.Y)+1, 8)
	tm.clearDirty()
	tm.EraseLine(CSIEraseLineLeft, true)
	expect(t, tm3Dirty(&tm, 0), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		expectEqualStrings(t, "     X  9", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseLine complete preserves background sgr"
func TestTm3_Terminal_eraseLine_complete_preserves_background_sgr(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm3PrintEach(t, &tm, "ABCDE")
	tm.SetCursorPos(1, 2)
	tm3SetAttribute(t, &tm, tm3DirectBg(0xFF, 0, 0))
	tm.EraseLine(CSIEraseLineComplete, false)

	{
		expectEqualStrings(t, "", tm3Plain(t, &tm))
		for x := uint(0); x < 5; x++ {
			list_cell := tm3At(t, &tm, x, 0)
			tm3ExpectBgRGB(t, list_cell.Cell, 0xFF, 0, 0)
		}
	}
}

// Zig: test "Terminal: eraseLine complete protected attributes respected with iso"
func TestTm3_Terminal_eraseLine_complete_protected_attributes_respected_with_iso(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm.SetProtectedMode(ANSIProtectedModeIso)
	tm3PrintEach(t, &tm, "ABC")
	tm.SetCursorPos(1, 1)
	tm.clearDirty()
	tm.EraseLine(CSIEraseLineComplete, false)
	expect(t, tm3Dirty(&tm, 0), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		expectEqualStrings(t, "ABC", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseLine complete protected attributes ignored with dec most recent"
func TestTm3_Terminal_eraseLine_complete_protected_attributes_ignored_with_dec_most_recent(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm.SetProtectedMode(ANSIProtectedModeIso)
	tm3PrintEach(t, &tm, "ABC")
	tm.SetProtectedMode(ANSIProtectedModeDec)
	tm.SetProtectedMode(ANSIProtectedModeOff)
	tm.SetCursorPos(1, 2)
	tm.clearDirty()
	tm.EraseLine(CSIEraseLineComplete, false)
	expect(t, tm3Dirty(&tm, 0), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		expectEqualStrings(t, "", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseLine complete protected attributes ignored with dec set"
func TestTm3_Terminal_eraseLine_complete_protected_attributes_ignored_with_dec_set(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm.SetProtectedMode(ANSIProtectedModeDec)
	tm3PrintEach(t, &tm, "ABC")
	tm.SetCursorPos(1, 2)
	tm.clearDirty()
	tm.EraseLine(CSIEraseLineComplete, false)
	expect(t, tm3Dirty(&tm, 0), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		expectEqualStrings(t, "", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseLine complete protected requested"
func TestTm3_Terminal_eraseLine_complete_protected_requested(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	tm3PrintEach(t, &tm, "123456789")
	tm.SetCursorPos(uint(tm.Screens.Active.Cursor.Y)+1, 6)
	tm.SetProtectedMode(ANSIProtectedModeDec)
	tm3Print(t, &tm, 'X')
	tm.SetCursorPos(uint(tm.Screens.Active.Cursor.Y)+1, 8)
	tm.clearDirty()
	tm.EraseLine(CSIEraseLineComplete, true)
	expect(t, tm3Dirty(&tm, 0), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		expectEqualStrings(t, "     X", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: tabClear single"
func TestTm3_Terminal_tabClear_single(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(30, 5))
	defer tm.Deinit()

	tm.HorizontalTab()
	tm.TabClear(CSITabClearCurrent)
	expect(t, !tm3Dirty(&tm, 0), "!t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	tm.SetCursorPos(1, 1)
	tm.HorizontalTab()
	expectEqual(t, uint(16), uint(tm.Screens.Active.Cursor.X))
}

// Zig: test "Terminal: tabClear all"
func TestTm3_Terminal_tabClear_all(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(30, 5))
	defer tm.Deinit()

	tm.TabClear(CSITabClearAll)
	expect(t, !tm3Dirty(&tm, 0), "!t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	tm.SetCursorPos(1, 1)
	tm.HorizontalTab()
	expectEqual(t, uint(29), uint(tm.Screens.Active.Cursor.X))
}

// Zig: test "Terminal: printRepeat simple"
func TestTm3_Terminal_printRepeat_simple(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm3PrintString(t, &tm, "A")
	zigTry(t, tm.PrintRepeat(1))
	expect(t, tm3Dirty(&tm, 0), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		expectEqualStrings(t, "AA", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: printRepeat wrap"
func TestTm3_Terminal_printRepeat_wrap(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm3PrintString(t, &tm, "    A")
	zigTry(t, tm.PrintRepeat(1))
	expect(t, tm3Dirty(&tm, 0), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		expectEqualStrings(t, "    A\nA", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: printRepeat no previous character"
func TestTm3_Terminal_printRepeat_no_previous_character(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	zigTry(t, tm.PrintRepeat(1))
	expect(t, !tm3Dirty(&tm, 0), "!t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		expectEqualStrings(t, "", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: printAttributes"
func TestTm3_Terminal_printAttributes(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	var storage [64]uint8

	func() {
		tm3SetAttribute(t, &tm, tm3DirectFg(1, 2, 3))
		defer tm3SetAttribute(t, &tm, tm3Attr(SGRAttributeTagUnset))
		buf, err := tm.PrintAttributes(storage[:])
		zigTry(t, err)
		expectEqualStrings(t, "0;38:2::1:2:3", string(buf))
	}()

	func() {
		tm3SetAttribute(t, &tm, tm3Attr(SGRAttributeTagBold))
		tm3SetAttribute(t, &tm, tm3DirectBg(1, 2, 3))
		defer tm3SetAttribute(t, &tm, tm3Attr(SGRAttributeTagUnset))
		buf, err := tm.PrintAttributes(storage[:])
		zigTry(t, err)
		expectEqualStrings(t, "0;1;48:2::1:2:3", string(buf))
	}()

	func() {
		tm3SetAttribute(t, &tm, tm3Attr(SGRAttributeTagBold))
		tm3SetAttribute(t, &tm, tm3Attr(SGRAttributeTagFaint))
		tm3SetAttribute(t, &tm, tm3Attr(SGRAttributeTagItalic))
		tm3SetAttribute(t, &tm, SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineSingle})
		tm3SetAttribute(t, &tm, tm3Attr(SGRAttributeTagBlink))
		tm3SetAttribute(t, &tm, tm3Attr(SGRAttributeTagInverse))
		tm3SetAttribute(t, &tm, tm3Attr(SGRAttributeTagInvisible))
		tm3SetAttribute(t, &tm, tm3Attr(SGRAttributeTagStrikethrough))
		tm3SetAttribute(t, &tm, tm3DirectFg(100, 200, 255))
		tm3SetAttribute(t, &tm, tm3DirectBg(101, 102, 103))
		defer tm3SetAttribute(t, &tm, tm3Attr(SGRAttributeTagUnset))
		buf, err := tm.PrintAttributes(storage[:])
		zigTry(t, err)
		expectEqualStrings(t, "0;1;2;3;4;5;7;8;9;38:2::100:200:255;48:2::101:102:103", string(buf))
	}()

	func() {
		tm3SetAttribute(t, &tm, SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineSingle})
		defer tm3SetAttribute(t, &tm, tm3Attr(SGRAttributeTagUnset))
		buf, err := tm.PrintAttributes(storage[:])
		zigTry(t, err)
		expectEqualStrings(t, "0;4", string(buf))
	}()

	{
		buf, err := tm.PrintAttributes(storage[:])
		zigTry(t, err)
		expectEqualStrings(t, "0", string(buf))
	}
}

// Zig: test "Terminal: eraseDisplay simple erase below"
func TestTm3_Terminal_eraseDisplay_simple_erase_below(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm3PrintEach(t, &tm, "ABC")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "DEF")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "GHI")
	tm.SetCursorPos(2, 2)

	tm.clearDirty()
	tm.EraseDisplay(CSIEraseDisplayBelow, false)
	expect(t, !tm3Dirty(&tm, 0), "!t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, tm3Dirty(&tm, 1), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, tm3Dirty(&tm, 2), "t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")

	{
		expectEqualStrings(t, "ABC\nD", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseDisplay erase below preserves SGR bg"
func TestTm3_Terminal_eraseDisplay_erase_below_preserves_SGR_bg(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm3PrintEach(t, &tm, "ABC")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "DEF")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "GHI")
	tm.SetCursorPos(2, 2)

	tm3SetAttribute(t, &tm, tm3DirectBg(0xFF, 0, 0))
	tm.EraseDisplay(CSIEraseDisplayBelow, false)

	{
		expectEqualStrings(t, "ABC\nD", tm3Plain(t, &tm))
		for x := uint(1); x < 5; x++ {
			list_cell := tm3At(t, &tm, x, 1)
			tm3ExpectBgRGB(t, list_cell.Cell, 0xFF, 0, 0)
		}
	}
}

// Zig: test "Terminal: eraseDisplay below split multi-cell"
func TestTm3_Terminal_eraseDisplay_below_split_multi_cell(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm3PrintString(t, &tm, "AB橋C")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintString(t, &tm, "DE橋F")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintString(t, &tm, "GH橋I")
	tm.SetCursorPos(2, 4)
	tm.EraseDisplay(CSIEraseDisplayBelow, false)

	{
		expectEqualStrings(t, "AB橋C\nDE", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseDisplay below protected attributes respected with iso"
func TestTm3_Terminal_eraseDisplay_below_protected_attributes_respected_with_iso(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm.SetProtectedMode(ANSIProtectedModeIso)
	tm3PrintEach(t, &tm, "ABC")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "DEF")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "GHI")
	tm.SetCursorPos(2, 2)
	tm.EraseDisplay(CSIEraseDisplayBelow, false)

	{
		expectEqualStrings(t, "ABC\nDEF\nGHI", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseDisplay below protected attributes ignored with dec most recent"
func TestTm3_Terminal_eraseDisplay_below_protected_attributes_ignored_with_dec_most_recent(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm.SetProtectedMode(ANSIProtectedModeIso)
	tm3PrintEach(t, &tm, "ABC")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "DEF")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "GHI")
	tm.SetProtectedMode(ANSIProtectedModeDec)
	tm.SetProtectedMode(ANSIProtectedModeOff)
	tm.SetCursorPos(2, 2)
	tm.EraseDisplay(CSIEraseDisplayBelow, false)

	{
		expectEqualStrings(t, "ABC\nD", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseDisplay below protected attributes ignored with dec set"
func TestTm3_Terminal_eraseDisplay_below_protected_attributes_ignored_with_dec_set(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm.SetProtectedMode(ANSIProtectedModeDec)
	tm3PrintEach(t, &tm, "ABC")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "DEF")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "GHI")
	tm.SetCursorPos(2, 2)
	tm.EraseDisplay(CSIEraseDisplayBelow, false)

	{
		expectEqualStrings(t, "ABC\nD", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseDisplay below protected attributes respected with force"
func TestTm3_Terminal_eraseDisplay_below_protected_attributes_respected_with_force(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm.SetProtectedMode(ANSIProtectedModeDec)
	tm3PrintEach(t, &tm, "ABC")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "DEF")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "GHI")
	tm.SetCursorPos(2, 2)
	tm.EraseDisplay(CSIEraseDisplayBelow, true)

	{
		expectEqualStrings(t, "ABC\nDEF\nGHI", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseDisplay simple erase above"
func TestTm3_Terminal_eraseDisplay_simple_erase_above(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm3PrintEach(t, &tm, "ABC")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "DEF")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "GHI")
	tm.SetCursorPos(2, 2)

	tm.clearDirty()
	tm.EraseDisplay(CSIEraseDisplayAbove, false)
	expect(t, tm3Dirty(&tm, 0), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, tm3Dirty(&tm, 1), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, !tm3Dirty(&tm, 2), "!t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")

	{
		expectEqualStrings(t, "\n  F\nGHI", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseDisplay erase above preserves SGR bg"
func TestTm3_Terminal_eraseDisplay_erase_above_preserves_SGR_bg(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm3PrintEach(t, &tm, "ABC")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "DEF")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "GHI")
	tm.SetCursorPos(2, 2)

	tm3SetAttribute(t, &tm, tm3DirectBg(0xFF, 0, 0))
	tm.EraseDisplay(CSIEraseDisplayAbove, false)

	{
		expectEqualStrings(t, "\n  F\nGHI", tm3Plain(t, &tm))
		for x := uint(0); x < 2; x++ {
			list_cell := tm3At(t, &tm, x, 1)
			tm3ExpectBgRGB(t, list_cell.Cell, 0xFF, 0, 0)
		}
	}
}

// Zig: test "Terminal: eraseDisplay above split multi-cell"
func TestTm3_Terminal_eraseDisplay_above_split_multi_cell(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm3PrintString(t, &tm, "AB橋C")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintString(t, &tm, "DE橋F")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintString(t, &tm, "GH橋I")
	tm.SetCursorPos(2, 3)
	tm.EraseDisplay(CSIEraseDisplayAbove, false)

	{
		expectEqualStrings(t, "\n    F\nGH橋I", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseDisplay above protected attributes respected with iso"
func TestTm3_Terminal_eraseDisplay_above_protected_attributes_respected_with_iso(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm.SetProtectedMode(ANSIProtectedModeIso)
	tm3PrintEach(t, &tm, "ABC")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "DEF")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "GHI")
	tm.SetCursorPos(2, 2)
	tm.EraseDisplay(CSIEraseDisplayAbove, false)

	{
		expectEqualStrings(t, "ABC\nDEF\nGHI", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseDisplay above protected attributes ignored with dec most recent"
func TestTm3_Terminal_eraseDisplay_above_protected_attributes_ignored_with_dec_most_recent(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm.SetProtectedMode(ANSIProtectedModeIso)
	tm3PrintEach(t, &tm, "ABC")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "DEF")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "GHI")
	tm.SetProtectedMode(ANSIProtectedModeDec)
	tm.SetProtectedMode(ANSIProtectedModeOff)
	tm.SetCursorPos(2, 2)
	tm.EraseDisplay(CSIEraseDisplayAbove, false)

	{
		expectEqualStrings(t, "\n  F\nGHI", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseDisplay above protected attributes ignored with dec set"
func TestTm3_Terminal_eraseDisplay_above_protected_attributes_ignored_with_dec_set(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm.SetProtectedMode(ANSIProtectedModeDec)
	tm3PrintEach(t, &tm, "ABC")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "DEF")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "GHI")
	tm.SetCursorPos(2, 2)
	tm.EraseDisplay(CSIEraseDisplayAbove, false)

	{
		expectEqualStrings(t, "\n  F\nGHI", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseDisplay above protected attributes respected with force"
func TestTm3_Terminal_eraseDisplay_above_protected_attributes_respected_with_force(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm.SetProtectedMode(ANSIProtectedModeDec)
	tm3PrintEach(t, &tm, "ABC")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "DEF")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "GHI")
	tm.SetCursorPos(2, 2)
	tm.EraseDisplay(CSIEraseDisplayAbove, true)

	{
		expectEqualStrings(t, "ABC\nDEF\nGHI", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseDisplay protected complete"
func TestTm3_Terminal_eraseDisplay_protected_complete(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	tm3Print(t, &tm, 'A')
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "123456789")
	tm.SetCursorPos(uint(tm.Screens.Active.Cursor.Y)+1, 6)
	tm.SetProtectedMode(ANSIProtectedModeDec)
	tm3Print(t, &tm, 'X')
	tm.SetCursorPos(uint(tm.Screens.Active.Cursor.Y)+1, 4)

	tm.clearDirty()
	tm.EraseDisplay(CSIEraseDisplayComplete, true)
	for y := uint32(0); y < uint32(tm.Rows); y++ {
		expect(t, tm3Dirty(&tm, y), "t.isDirty(.{ .active = .{ .x = 0, .y = y } })")
	}

	{
		expectEqualStrings(t, "\n     X", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseDisplay protected below"
func TestTm3_Terminal_eraseDisplay_protected_below(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	tm3Print(t, &tm, 'A')
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "123456789")
	tm.SetCursorPos(uint(tm.Screens.Active.Cursor.Y)+1, 6)
	tm.SetProtectedMode(ANSIProtectedModeDec)
	tm3Print(t, &tm, 'X')
	tm.SetCursorPos(uint(tm.Screens.Active.Cursor.Y)+1, 4)
	tm.EraseDisplay(CSIEraseDisplayBelow, true)

	{
		expectEqualStrings(t, "A\n123  X", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseDisplay scroll complete"
func TestTm3_Terminal_eraseDisplay_scroll_complete(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	tm3Print(t, &tm, 'A')
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm.EraseDisplay(CSIEraseDisplayScrollComplete, false)

	{
		expectEqualStrings(t, "", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseDisplay protected above"
func TestTm3_Terminal_eraseDisplay_protected_above(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 3))
	defer tm.Deinit()

	tm3Print(t, &tm, 'A')
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "123456789")
	tm.SetCursorPos(uint(tm.Screens.Active.Cursor.Y)+1, 6)
	tm.SetProtectedMode(ANSIProtectedModeDec)
	tm3Print(t, &tm, 'X')
	tm.SetCursorPos(uint(tm.Screens.Active.Cursor.Y)+1, 8)
	tm.EraseDisplay(CSIEraseDisplayAbove, true)

	{
		expectEqualStrings(t, "\n     X  9", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: eraseDisplay complete preserves cursor"
func TestTm3_Terminal_eraseDisplay_complete_preserves_cursor(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	// Set our cursur
	tm3SetAttribute(t, &tm, tm3Attr(SGRAttributeTagBold))
	tm3PrintString(t, &tm, "AAAA")
	expect(t, tm.Screens.Active.Cursor.StyleID != StyleDefaultID, "t.screens.active.cursor.style_id != style.default_id")

	// Erasing the display may detect that our style is no longer in use
	// and prune our style, which we don't want because its still our
	// active cursor.
	tm.EraseDisplay(CSIEraseDisplayComplete, false)
	expect(t, tm.Screens.Active.Cursor.StyleID != StyleDefaultID, "t.screens.active.cursor.style_id != style.default_id")
}

// Zig: test "Terminal: semantic prompt"
func TestTm3_Terminal_semantic_prompt(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	// Prompt
	tm3SemanticPrompt(t, &tm, OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt))
	tm3PrintEach(t, &tm, "hello")
	expectEqual(t, SizeCellCountInt(0), tm.Screens.Active.Cursor.Y)
	expectEqual(t, SizeCellCountInt(5), tm.Screens.Active.Cursor.X)
	{
		list_cell := tm3At(t, &tm, uint(tm.Screens.Active.Cursor.X)-1, uint(tm.Screens.Active.Cursor.Y))
		cell := list_cell.Cell
		expectEqual(t, PageCellSemanticContentPrompt, cell.semantic_content)

		row := list_cell.Row
		expectEqual(t, PageRowSemanticPromptPrompt, row.semantic_prompt)
	}

	// Start input but end it on EOL
	tm3SemanticPrompt(t, &tm, OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionEndPromptStartInputTerminateEol))
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)

	// Write some output
	expectEqual(t, SizeCellCountInt(1), tm.Screens.Active.Cursor.Y)
	expectEqual(t, SizeCellCountInt(0), tm.Screens.Active.Cursor.X)
	tm3PrintEach(t, &tm, "world")
	{
		list_cell := tm3At(t, &tm, uint(tm.Screens.Active.Cursor.X)-1, uint(tm.Screens.Active.Cursor.Y))
		cell := list_cell.Cell
		expectEqual(t, PageCellSemanticContentOutput, cell.semantic_content)

		row := list_cell.Row
		expectEqual(t, PageRowSemanticPromptNone, row.semantic_prompt)
	}
}

// Zig: test "Terminal: semantic prompt continuations"
func TestTm3_Terminal_semantic_prompt_continuations(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	// Prompt
	tm3SemanticPrompt(t, &tm, OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt))
	tm3PrintEach(t, &tm, "hello")
	expectEqual(t, SizeCellCountInt(0), tm.Screens.Active.Cursor.Y)
	expectEqual(t, SizeCellCountInt(5), tm.Screens.Active.Cursor.X)
	{
		list_cell := tm3At(t, &tm, uint(tm.Screens.Active.Cursor.X)-1, uint(tm.Screens.Active.Cursor.Y))
		cell := list_cell.Cell
		expectEqual(t, PageCellSemanticContentPrompt, cell.semantic_content)

		row := list_cell.Row
		expectEqual(t, PageRowSemanticPromptPrompt, row.semantic_prompt)
	}

	// Start input but end it on EOL
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3SemanticPrompt(t, &tm, tm3Prompt(OSCParsersSemanticPromptCommandActionPromptStart, "k=c"))

	// Write some output
	expectEqual(t, SizeCellCountInt(1), tm.Screens.Active.Cursor.Y)
	expectEqual(t, SizeCellCountInt(0), tm.Screens.Active.Cursor.X)
	tm3PrintEach(t, &tm, "world")
	{
		list_cell := tm3At(t, &tm, uint(tm.Screens.Active.Cursor.X)-1, uint(tm.Screens.Active.Cursor.Y))
		cell := list_cell.Cell
		expectEqual(t, PageCellSemanticContentPrompt, cell.semantic_content)

		row := list_cell.Row
		expectEqual(t, PageRowSemanticPromptPromptContinuation, row.semantic_prompt)
	}
}

// Zig: test "Terminal: index in prompt mode marks new row as prompt continuation"
func TestTm3_Terminal_index_in_prompt_mode_marks_new_row_as_prompt_continuation(t *testing.T) {
	testStdCapacity(t)
	// This tests the Fish shell workaround: when in prompt mode and we get
	// a newline, assume the new row is a prompt continuation (since Fish
	// doesn't emit OSC133 k=s markers for continuation lines).
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	// Start a prompt
	tm3SemanticPrompt(t, &tm, OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionPromptStart))
	tm3PrintEach(t, &tm, "hello")

	// Verify first row is marked as prompt
	{
		list_cell := tm3At(t, &tm, 0, 0)
		expectEqual(t, PageRowSemanticPromptPrompt, list_cell.Row.semantic_prompt)
	}

	// Now do a linefeed while still in prompt mode
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)

	// The new row should automatically be marked as prompt continuation
	{
		list_cell := tm3At(t, &tm, 0, 1)
		expectEqual(t, PageRowSemanticPromptPromptContinuation, list_cell.Row.semantic_prompt)
	}

	// The cursor semantic content should still be prompt
	expectEqual(t, PageCellSemanticContentPrompt, tm.Screens.Active.Cursor.SemanticContent)
}

// Zig: test "Terminal: index in input mode does not mark new row as prompt"
func TestTm3_Terminal_index_in_input_mode_does_not_mark_new_row_as_prompt(t *testing.T) {
	testStdCapacity(t)
	// Input mode should NOT trigger prompt continuation on newline
	// (only prompt mode does, not input mode)
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	// Start a prompt then switch to input
	tm3SemanticPrompt(t, &tm, OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionPromptStart))
	tm3PrintEach(t, &tm, "$ ")
	tm3SemanticPrompt(t, &tm, OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionEndPromptStartInput))
	tm3PrintEach(t, &tm, "echo \\")

	// Linefeed while in input mode
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)

	// The new row should be marked as prompt continuation
	{
		list_cell := tm3At(t, &tm, 0, 1)
		expectEqual(t, PageRowSemanticPromptPromptContinuation, list_cell.Row.semantic_prompt)
	}

	// Our cursor should still be in input
	expectEqual(t, PageCellSemanticContentInput, tm.Screens.Active.Cursor.SemanticContent)
}

// Zig: test "Terminal: index in output mode does not mark new row as prompt"
func TestTm3_Terminal_index_in_output_mode_does_not_mark_new_row_as_prompt(t *testing.T) {
	testStdCapacity(t)
	// Output mode should NOT trigger prompt continuation
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	// Complete prompt cycle: prompt -> input -> output
	tm3SemanticPrompt(t, &tm, OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionPromptStart))
	tm3PrintEach(t, &tm, "$ ")
	tm3SemanticPrompt(t, &tm, OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionEndPromptStartInput))
	tm3PrintEach(t, &tm, "ls")
	tm3SemanticPrompt(t, &tm, OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionEndInputStartOutput))

	// Linefeed while in output mode
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)

	// The new row should NOT be marked as a prompt
	{
		list_cell := tm3At(t, &tm, 0, 1)
		expectEqual(t, PageRowSemanticPromptNone, list_cell.Row.semantic_prompt)
	}
}

// Zig: test "Terminal: OSC133C at x=0 on prompt row clears prompt mark"
func TestTm3_Terminal_OSC133C_at_x_eq_0_on_prompt_row_clears_prompt_mark(t *testing.T) {
	testStdCapacity(t)
	// This tests the second Fish heuristic: when Fish emits a newline
	// then immediately sends OSC133C (start output) at column 0, we
	// should clear the prompt continuation mark we just set.
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	// Start a prompt
	tm3SemanticPrompt(t, &tm, OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionPromptStart))
	tm3PrintEach(t, &tm, "$ echo \\")

	// Simulate Fish behavior: newline first (which marks next row as prompt)
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)

	// Verify the new row is marked as prompt continuation
	{
		list_cell := tm3At(t, &tm, 0, 1)
		expectEqual(t, PageRowSemanticPromptPromptContinuation, list_cell.Row.semantic_prompt)
	}

	// Now Fish sends OSC133C at column 0 (cursor is still at x=0)
	expectEqual(t, SizeCellCountInt(0), tm.Screens.Active.Cursor.X)
	tm3SemanticPrompt(t, &tm, OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionEndInputStartOutput))

	// The prompt continuation should be cleared
	{
		list_cell := tm3At(t, &tm, 0, 1)
		expectEqual(t, PageRowSemanticPromptNone, list_cell.Row.semantic_prompt)
	}
}

// Zig: test "Terminal: OSC133C at x>0 on prompt row does not clear prompt mark"
func TestTm3_Terminal_OSC133C_at_x_gt_0_on_prompt_row_does_not_clear_prompt_mark(t *testing.T) {
	testStdCapacity(t)
	// If we're not at column 0, we shouldn't clear the prompt mark
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	// Start a prompt on a row
	tm3SemanticPrompt(t, &tm, OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionPromptStart))
	tm3PrintEach(t, &tm, "$ ")

	// Move to a new line and mark it as prompt continuation manually
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3SemanticPrompt(t, &tm, tm3Prompt(OSCParsersSemanticPromptCommandActionPromptStart, "k=c"))
	tm3PrintEach(t, &tm, "> ")

	// Verify the row is marked as prompt continuation
	{
		list_cell := tm3At(t, &tm, 0, 1)
		expectEqual(t, PageRowSemanticPromptPromptContinuation, list_cell.Row.semantic_prompt)
	}

	// Now send OSC133C but cursor is NOT at column 0
	expect(t, tm.Screens.Active.Cursor.X > 0, "t.screens.active.cursor.x > 0")
	tm3SemanticPrompt(t, &tm, OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionEndInputStartOutput))

	// The prompt continuation should NOT be cleared (we're not at x=0)
	{
		list_cell := tm3At(t, &tm, 0, 1)
		expectEqual(t, PageRowSemanticPromptPromptContinuation, list_cell.Row.semantic_prompt)
	}
}

// Zig: test "Terminal: multiple newlines in prompt mode marks all rows"
func TestTm3_Terminal_multiple_newlines_in_prompt_mode_marks_all_rows(t *testing.T) {
	testStdCapacity(t)
	// Multiple newlines should each mark their row as prompt continuation
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	// Start a prompt
	tm3SemanticPrompt(t, &tm, OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionPromptStart))
	tm3PrintEach(t, &tm, "line1")

	// Multiple newlines
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "line2")
	tm.CarriageReturn()
	tm3Linefeed(t, &tm)
	tm3PrintEach(t, &tm, "line3")

	// First row should be prompt
	{
		list_cell := tm3At(t, &tm, 0, 0)
		expectEqual(t, PageRowSemanticPromptPrompt, list_cell.Row.semantic_prompt)
	}

	// Second and third rows should be prompt continuation
	{
		list_cell := tm3At(t, &tm, 0, 1)
		expectEqual(t, PageRowSemanticPromptPromptContinuation, list_cell.Row.semantic_prompt)
	}
	{
		list_cell := tm3At(t, &tm, 0, 2)
		expectEqual(t, PageRowSemanticPromptPromptContinuation, list_cell.Row.semantic_prompt)
	}
}

// Zig: test "Terminal: OSC133A click_events=1 sets click to click_events"
func TestTm3_Terminal_OSC133A_click_events_1_sets_click_to_click_events(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	// Verify default state is none
	expectEqual(t, ScreenSemanticPromptSemanticClick{Tag: ScreenSemanticPromptSemanticClickNone}, tm.Screens.Active.SemanticPrompt.Click)

	// OSC 133;A with click_events=1
	tm3SemanticPrompt(t, &tm, tm3Prompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "click_events=1"))

	expectEqual(t, ScreenSemanticPromptSemanticClick{Tag: ScreenSemanticPromptSemanticClickClickEvents}, tm.Screens.Active.SemanticPrompt.Click)
}

// Zig: test "Terminal: OSC133A click_events=0 does not set click_events"
func TestTm3_Terminal_OSC133A_click_events_0_does_not_set_click_events(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	// OSC 133;A with click_events=0
	tm3SemanticPrompt(t, &tm, tm3Prompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "click_events=0"))

	// Should remain none since click_events=0 doesn't activate anything
	expectEqual(t, ScreenSemanticPromptSemanticClick{Tag: ScreenSemanticPromptSemanticClickNone}, tm.Screens.Active.SemanticPrompt.Click)
}

// Zig: test "Terminal: OSC133A cl option sets click to cl value"
func TestTm3_Terminal_OSC133A_cl_option_sets_click_to_cl_value(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	// OSC 133;A with cl=m (multiple)
	tm3SemanticPrompt(t, &tm, tm3Prompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "cl=m"))

	expectEqual(t, ScreenSemanticPromptSemanticClick{Tag: ScreenSemanticPromptSemanticClickCl, Cl: OSCParsersSemanticPromptClickMultiple}, tm.Screens.Active.SemanticPrompt.Click)
}

// Zig: test "Terminal: OSC133A cl=line sets click to line"
func TestTm3_Terminal_OSC133A_cl_eq_line_sets_click_to_line(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	tm3SemanticPrompt(t, &tm, tm3Prompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "cl=line"))

	expectEqual(t, ScreenSemanticPromptSemanticClick{Tag: ScreenSemanticPromptSemanticClickCl, Cl: OSCParsersSemanticPromptClickLine}, tm.Screens.Active.SemanticPrompt.Click)
}

// Zig: test "Terminal: OSC133A click_events=1 takes priority over cl"
func TestTm3_Terminal_OSC133A_click_events_1_takes_priority_over_cl(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	// OSC 133;A with both click_events=1 and cl=m
	tm3SemanticPrompt(t, &tm, tm3Prompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "click_events=1;cl=m"))

	// click_events should take priority
	expectEqual(t, ScreenSemanticPromptSemanticClick{Tag: ScreenSemanticPromptSemanticClickClickEvents}, tm.Screens.Active.SemanticPrompt.Click)
}

// Zig: test "Terminal: OSC133A click_events=0 falls back to cl"
func TestTm3_Terminal_OSC133A_click_events_0_falls_back_to_cl(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	// OSC 133;A with click_events=0 and cl=v
	tm3SemanticPrompt(t, &tm, tm3Prompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "click_events=0;cl=v"))

	// Should fall back to cl since click_events is disabled
	expectEqual(t, ScreenSemanticPromptSemanticClick{Tag: ScreenSemanticPromptSemanticClickCl, Cl: OSCParsersSemanticPromptClickConservativeVertical}, tm.Screens.Active.SemanticPrompt.Click)
}

// Zig: test "Terminal: OSC133A no click options leaves click as none"
func TestTm3_Terminal_OSC133A_no_click_options_leaves_click_as_none(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 5))
	defer tm.Deinit()

	// OSC 133;A with no click-related options
	tm3SemanticPrompt(t, &tm, tm3Prompt(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt, "aid=123"))

	expectEqual(t, ScreenSemanticPromptSemanticClick{Tag: ScreenSemanticPromptSemanticClickNone}, tm.Screens.Active.SemanticPrompt.Click)
}

// Zig: test "Terminal: cursorIsAtPrompt"
func TestTm3_Terminal_cursorIsAtPrompt(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 3))
	defer tm.Deinit()

	expect(t, !tm.CursorIsAtPrompt(), "!t.cursorIsAtPrompt()")
	tm3SemanticPrompt(t, &tm, OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionPromptStart))
	expect(t, tm.CursorIsAtPrompt(), "t.cursorIsAtPrompt()")
	tm3PrintEach(t, &tm, "$ ")

	// Input is also a prompt
	tm3SemanticPrompt(t, &tm, OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionEndPromptStartInput))
	expect(t, tm.CursorIsAtPrompt(), "t.cursorIsAtPrompt()")
	tm3PrintEach(t, &tm, "ls")

	// But once we say we're starting output, we're not a prompt
	// (cursor is not at x=0, so the Fish heuristic doesn't trigger)
	tm3SemanticPrompt(t, &tm, OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionEndInputStartOutput))
	// Still a prompt because this line has a prompt
	expect(t, tm.CursorIsAtPrompt(), "t.cursorIsAtPrompt()")
	tm3Linefeed(t, &tm)
	expect(t, !tm.CursorIsAtPrompt(), "!t.cursorIsAtPrompt()")

	// Until we know we're at a prompt again
	tm3Linefeed(t, &tm)
	tm3SemanticPrompt(t, &tm, OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionPromptStart))
	expect(t, tm.CursorIsAtPrompt(), "t.cursorIsAtPrompt()")
}

// Zig: test "Terminal: cursorIsAtPrompt alternate screen"
func TestTm3_Terminal_cursorIsAtPrompt_alternate_screen(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(3, 2))
	defer tm.Deinit()

	expect(t, !tm.CursorIsAtPrompt(), "!t.cursorIsAtPrompt()")
	tm3SemanticPrompt(t, &tm, OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionPromptStart))
	expect(t, tm.CursorIsAtPrompt(), "t.cursorIsAtPrompt()")

	// Secondary screen is never a prompt
	tm3SwitchScreenMode(t, &tm, TerminalSwitchScreenMode1049, true)
	expect(t, !tm.CursorIsAtPrompt(), "!t.cursorIsAtPrompt()")
	tm3SemanticPrompt(t, &tm, OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionPromptStart))
	expect(t, !tm.CursorIsAtPrompt(), "!t.cursorIsAtPrompt()")
}

// Zig: test "Terminal: fullReset with a non-empty pen"
func TestTm3_Terminal_fullReset_with_a_non_empty_pen(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(80, 80))
	defer tm.Deinit()

	tm3SetAttribute(t, &tm, tm3DirectFg(0xFF, 0, 0x7F))
	tm3SetAttribute(t, &tm, tm3DirectBg(0xFF, 0, 0x7F))
	tm.Screens.Active.Cursor.SemanticContent = PageCellSemanticContentInput
	tm.FullReset()

	{
		list_cell := tm3At(t, &tm, uint(tm.Screens.Active.Cursor.X), uint(tm.Screens.Active.Cursor.Y))
		cell := list_cell.Cell
		expect(t, cell.style_id == 0, "cell.style_id == 0")
	}

	expectEqual(t, StyleID(0), tm.Screens.Active.Cursor.StyleID)
	expectEqual(t, PageCellSemanticContentOutput, tm.Screens.Active.Cursor.SemanticContent)
}

// Zig: test "Terminal: fullReset hyperlink"
func TestTm3_Terminal_fullReset_hyperlink(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(80, 80))
	defer tm.Deinit()

	zigTry(t, tm.Screens.Active.StartHyperlink([]uint8("http://example.com"), None[[]uint8]()))
	tm.FullReset()
	expectEqual(t, HyperlinkID(0), tm.Screens.Active.Cursor.HyperlinkID)
}

// Zig: test "Terminal: fullReset with a non-empty saved cursor"
func TestTm3_Terminal_fullReset_with_a_non_empty_saved_cursor(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(80, 80))
	defer tm.Deinit()

	tm3SetAttribute(t, &tm, tm3DirectFg(0xFF, 0, 0x7F))
	tm3SetAttribute(t, &tm, tm3DirectBg(0xFF, 0, 0x7F))
	tm.SaveCursor()
	tm.FullReset()

	{
		list_cell := tm3At(t, &tm, uint(tm.Screens.Active.Cursor.X), uint(tm.Screens.Active.Cursor.Y))
		cell := list_cell.Cell
		expect(t, cell.style_id == 0, "cell.style_id == 0")
	}

	expectEqual(t, StyleID(0), tm.Screens.Active.Cursor.StyleID)
}

// Zig: test "Terminal: fullReset origin mode"
func TestTm3_Terminal_fullReset_origin_mode(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 10))
	defer tm.Deinit()

	tm.SetCursorPos(3, 5)
	tm.Modes.Set(ModesModeOrigin, true)
	tm.FullReset()

	// Origin mode should be reset and the cursor should be moved
	expectEqual(t, SizeCellCountInt(0), tm.Screens.Active.Cursor.Y)
	expectEqual(t, SizeCellCountInt(0), tm.Screens.Active.Cursor.X)
	expect(t, !tm.Modes.Get(ModesModeOrigin), "!t.modes.get(.origin)")
}

// Zig: test "Terminal: fullReset status display"
func TestTm3_Terminal_fullReset_status_display(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 10))
	defer tm.Deinit()

	tm.StatusDisplay = ANSIStatusDisplayStatusLine
	tm.FullReset()
	expect(t, tm.StatusDisplay == ANSIStatusDisplayMain, "t.status_display == .main")
}

// https://github.com/mitchellh/ghostty/issues/1607
//
// Zig: test "Terminal: fullReset clears alt screen kitty keyboard state"
func TestTm3_Terminal_fullReset_clears_alt_screen_kitty_keyboard_state(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 10))
	defer tm.Deinit()

	tm3SwitchScreenMode(t, &tm, TerminalSwitchScreenMode1049, true)
	tm.Screens.Active.KittyKeyboard.Push(KittyKeyFlags{
		Disambiguate:     true,
		ReportEvents:     false,
		ReportAlternates: true,
		ReportAll:        true,
		ReportAssociated: true,
	})
	tm3SwitchScreenMode(t, &tm, TerminalSwitchScreenMode1049, false)

	tm.FullReset()
	expect(t, tm.Screens.Get(ScreenSetKeyAlternate) == nil, "t.screens.get(.alternate) == null")
}

// Zig: test "Terminal: fullReset default modes"
func TestTm3_Terminal_fullReset_default_modes(t *testing.T) {
	testStdCapacity(t)
	opts := terminalOptionsDefault(10, 10)
	opts.DefaultModes = ModesModePackedInit()
	sm1_modesFieldSet(&opts.DefaultModes, "grapheme_cluster", true)
	tm := tm3Init(t, opts)
	defer tm.Deinit()
	expect(t, tm.Modes.Get(ModesModeGraphemeCluster), "t.modes.get(.grapheme_cluster)")
	tm.FullReset()
	expect(t, tm.Modes.Get(ModesModeGraphemeCluster), "t.modes.get(.grapheme_cluster)")
}

// Zig: test "Terminal: fullReset tracked pins"
func TestTm3_Terminal_fullReset_tracked_pins(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(80, 80))
	defer tm.Deinit()

	// Create a tracked pin
	p, err := tm.Screens.Active.Pages.TrackPin(*tm.Screens.Active.Cursor.PagePin)
	zigTry(t, err)
	tm.FullReset()
	expect(t, tm.Screens.Active.Pages.PinIsValid(*p), "t.screens.active.pages.pinIsValid(p.*)")
}

// https://github.com/mitchellh/ghostty/issues/272
// This is also tested in depth in screen resize tests but I want to keep
// this test around to ensure we don't regress at multiple layers.
//
// Zig: test "Terminal: resize less cols with wide char then print"
func TestTm3_Terminal_resize_less_cols_with_wide_char_then_print(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(3, 3))
	defer tm.Deinit()

	tm3Print(t, &tm, 'x')
	tm3Print(t, &tm, '😀') // 0x1F600
	tm3Resize(t, &tm, 2, 3)
	tm.SetCursorPos(1, 2)
	tm3Print(t, &tm, '😀') // 0x1F600
}

// https://github.com/mitchellh/ghostty/issues/723
// This was found via fuzzing so its highly specific.
//
// Zig: test "Terminal: resize with left and right margin set"
func TestTm3_Terminal_resize_with_left_and_right_margin_set(t *testing.T) {
	testStdCapacity(t)
	const cols = 70
	const rows = 23
	tm := tm3Init(t, terminalOptionsDefault(cols, rows))
	defer tm.Deinit()

	tm.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	tm3Print(t, &tm, '0')
	tm.Modes.Set(ModesModeEnableMode3, true)
	tm3Resize(t, &tm, cols, rows)
	tm.SetLeftAndRightMargin(2, 0)
	zigTry(t, tm.PrintRepeat(1850))
	_ = tm.Modes.Restore(ModesModeEnableMode3)
	tm3Resize(t, &tm, cols, rows)
}

// https://github.com/mitchellh/ghostty/issues/1343
//
// Zig: test "Terminal: resize with wraparound off"
func TestTm3_Terminal_resize_with_wraparound_off(t *testing.T) {
	testStdCapacity(t)
	const cols = 4
	const rows = 2
	tm := tm3Init(t, terminalOptionsDefault(cols, rows))
	defer tm.Deinit()

	tm.Modes.Set(ModesModeWraparound, false)
	tm3Print(t, &tm, '0')
	tm3Print(t, &tm, '1')
	tm3Print(t, &tm, '2')
	tm3Print(t, &tm, '3')
	const new_cols = 2
	tm3Resize(t, &tm, new_cols, rows)

	str := tm3Plain(t, &tm)
	expectEqualStrings(t, "01", str)
}

// Zig: test "Terminal: resize with wraparound on"
func TestTm3_Terminal_resize_with_wraparound_on(t *testing.T) {
	testStdCapacity(t)
	const cols = 4
	const rows = 2
	tm := tm3Init(t, terminalOptionsDefault(cols, rows))
	defer tm.Deinit()

	tm.Modes.Set(ModesModeWraparound, true)
	tm3Print(t, &tm, '0')
	tm3Print(t, &tm, '1')
	tm3Print(t, &tm, '2')
	tm3Print(t, &tm, '3')
	const new_cols = 2
	tm3Resize(t, &tm, new_cols, rows)

	str := tm3Plain(t, &tm)
	expectEqualStrings(t, "01\n23", str)
}

// Zig: test "Terminal: resize with high unique style per cell"
func TestTm3_Terminal_resize_with_high_unique_style_per_cell(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(30, 30))
	defer tm.Deinit()

	for y := uint(0); y < uint(tm.Rows); y++ {
		for x := uint(0); x < uint(tm.Cols); x++ {
			tm.SetCursorPos(y, x)
			tm3SetAttribute(t, &tm, tm3DirectBg(uint8(x), uint8(y), 0))
			tm3Print(t, &tm, 'x')
		}
	}

	tm3Resize(t, &tm, 60, 30)
}

// Zig: test "Terminal: resize with high unique style per cell with wrapping"
func TestTm3_Terminal_resize_with_high_unique_style_per_cell_with_wrapping(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(30, 30))
	defer tm.Deinit()

	cell_count := uint16(tm.Rows * tm.Cols)
	for i := uint(0); i < uint(cell_count); i++ {
		r := uint8(i >> 8)
		g := uint8(i & 0xFF)

		tm3SetAttribute(t, &tm, tm3DirectBg(r, g, 0))
		tm3Print(t, &tm, 'x')
	}

	tm3Resize(t, &tm, 60, 30)
}

// Zig: test "Terminal: resize with reflow and saved cursor"
func TestTm3_Terminal_resize_with_reflow_and_saved_cursor(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(2, 3))
	defer tm.Deinit()
	tm3PrintString(t, &tm, "1A2B")
	tm.SetCursorPos(2, 2)
	{
		list_cell := tm3At(t, &tm, uint(tm.Screens.Active.Cursor.X), uint(tm.Screens.Active.Cursor.Y))
		cell := list_cell.Cell
		expectEqual(t, uint32('B'), cell.content.codepoint)
	}

	{
		expectEqualStrings(t, "1A\n2B", tm3Plain(t, &tm))
	}

	tm.SaveCursor()
	tm3Resize(t, &tm, 5, 3)
	tm.RestoreCursor()

	{
		expectEqualStrings(t, "1A2B", tm3Plain(t, &tm))
	}

	// Verify our cursor is still in the same place
	{
		list_cell := tm3At(t, &tm, uint(tm.Screens.Active.Cursor.X), uint(tm.Screens.Active.Cursor.Y))
		cell := list_cell.Cell
		expectEqual(t, uint32('B'), cell.content.codepoint)
	}
}

// Zig: test "Terminal: resize with reflow and saved cursor pending wrap"
func TestTm3_Terminal_resize_with_reflow_and_saved_cursor_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(2, 3))
	defer tm.Deinit()
	tm3PrintString(t, &tm, "1A2B")
	{
		list_cell := tm3At(t, &tm, uint(tm.Screens.Active.Cursor.X), uint(tm.Screens.Active.Cursor.Y))
		cell := list_cell.Cell
		expectEqual(t, uint32('B'), cell.content.codepoint)
	}

	{
		expectEqualStrings(t, "1A\n2B", tm3Plain(t, &tm))
	}

	tm.SaveCursor()
	tm3Resize(t, &tm, 5, 3)
	tm.RestoreCursor()

	{
		expectEqualStrings(t, "1A2B", tm3Plain(t, &tm))
	}

	// Pending wrap should be reset
	tm3Print(t, &tm, 'X')
	{
		expectEqualStrings(t, "1A2BX", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: DECCOLM without DEC mode 40"
func TestTm3_Terminal_DECCOLM_without_DEC_mode_40(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm.Modes.Set(ModesMode132Column, true)
	tm3Deccolm(t, &tm, TerminalDeccolmMode132Cols)
	expectEqual(t, SizeCellCountInt(5), tm.Cols)
	expectEqual(t, SizeCellCountInt(5), tm.Rows)
	expect(t, !tm.Modes.Get(ModesMode132Column), "!t.modes.get(.@\"132_column\")")
}

// Zig: test "Terminal: DECCOLM unset"
func TestTm3_Terminal_DECCOLM_unset(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm.Modes.Set(ModesModeEnableMode3, true)
	tm3Deccolm(t, &tm, TerminalDeccolmMode80Cols)
	expectEqual(t, SizeCellCountInt(80), tm.Cols)
	expectEqual(t, SizeCellCountInt(5), tm.Rows)
}

// Zig: test "Terminal: DECCOLM resets pending wrap"
func TestTm3_Terminal_DECCOLM_resets_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm3PrintEach(t, &tm, "ABCDE")
	expect(t, tm.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")

	tm.Modes.Set(ModesModeEnableMode3, true)
	tm3Deccolm(t, &tm, TerminalDeccolmMode80Cols)
	expectEqual(t, SizeCellCountInt(80), tm.Cols)
	expectEqual(t, SizeCellCountInt(5), tm.Rows)
	expect(t, !tm.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
}

// Zig: test "Terminal: DECCOLM preserves SGR bg"
func TestTm3_Terminal_DECCOLM_preserves_SGR_bg(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm3SetAttribute(t, &tm, tm3DirectBg(0xFF, 0, 0))
	tm.Modes.Set(ModesModeEnableMode3, true)
	tm3Deccolm(t, &tm, TerminalDeccolmMode80Cols)

	{
		list_cell := tm3At(t, &tm, 0, 0)
		tm3ExpectBgRGB(t, list_cell.Cell, 0xFF, 0, 0)
	}
}

// Zig: test "Terminal: DECCOLM resets scroll region"
func TestTm3_Terminal_DECCOLM_resets_scroll_region(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	tm.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	tm.SetTopAndBottomMargin(2, 3)
	tm.SetLeftAndRightMargin(3, 5)

	tm.Modes.Set(ModesModeEnableMode3, true)
	tm3Deccolm(t, &tm, TerminalDeccolmMode80Cols)

	expect(t, tm.Modes.Get(ModesModeEnableLeftAndRightMargin), "t.modes.get(.enable_left_and_right_margin)")
	expectEqual(t, SizeCellCountInt(0), tm.ScrollingRegion.Top)
	expectEqual(t, SizeCellCountInt(4), tm.ScrollingRegion.Bottom)
	expectEqual(t, SizeCellCountInt(0), tm.ScrollingRegion.Left)
	expectEqual(t, SizeCellCountInt(79), tm.ScrollingRegion.Right)
}

// Zig: test "Terminal: mode 47 alt screen plain"
func TestTm3_Terminal_mode_47_alt_screen_plain(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	// Print on primary screen
	tm3PrintString(t, &tm, "1A")

	// Go to alt screen with mode 47
	tm3SwitchScreenMode(t, &tm, TerminalSwitchScreenMode47, true)
	expectEqual(t, ScreenSetKeyAlternate, tm.Screens.ActiveKey)

	// Screen should be empty
	{
		expectEqualStrings(t, "", tm3Plain(t, &tm))
	}

	// Print on alt screen. This should be off center because
	// we copy the cursor over from the primary screen
	tm3PrintString(t, &tm, "2B")
	{
		expectEqualStrings(t, "  2B", tm3Plain(t, &tm))
	}

	// Go back to primary
	tm3SwitchScreenMode(t, &tm, TerminalSwitchScreenMode47, false)
	expectEqual(t, ScreenSetKeyPrimary, tm.Screens.ActiveKey)

	// Primary screen should still have the original content
	{
		expectEqualStrings(t, "1A", tm3Plain(t, &tm))
	}

	// Go back to alt screen with mode 47
	tm3SwitchScreenMode(t, &tm, TerminalSwitchScreenMode47, true)
	expectEqual(t, ScreenSetKeyAlternate, tm.Screens.ActiveKey)

	// Screen should retain content
	{
		expectEqualStrings(t, "  2B", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: mode 47 copies cursor both directions"
func TestTm3_Terminal_mode_47_copies_cursor_both_directions(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	// Color our cursor red
	tm3SetAttribute(t, &tm, tm3DirectFg(0xFF, 0, 0x7F))

	// Go to alt screen with mode 47
	tm3SwitchScreenMode(t, &tm, TerminalSwitchScreenMode47, true)
	expectEqual(t, ScreenSetKeyAlternate, tm.Screens.ActiveKey)

	// Verify that our style is set
	{
		expect(t, tm.Screens.Active.Cursor.StyleID != StyleDefaultID, "t.screens.active.cursor.style_id != style.default_id")
		page := &tm.Screens.Active.Cursor.PagePin.Node.data
		expectEqual(t, uint(1), page.styles.Count())
		expect(t, page.styles.RefCount(tm.Screens.Active.Cursor.StyleID) > 0, "page.styles.refCount(page.memory, t.screens.active.cursor.style_id) > 0")
	}

	// Set a new style
	tm3SetAttribute(t, &tm, tm3DirectFg(0, 0xFF, 0))

	// Go back to primary
	tm3SwitchScreenMode(t, &tm, TerminalSwitchScreenMode47, false)
	expectEqual(t, ScreenSetKeyPrimary, tm.Screens.ActiveKey)

	// Verify that our style is still set
	{
		expect(t, tm.Screens.Active.Cursor.StyleID != StyleDefaultID, "t.screens.active.cursor.style_id != style.default_id")
		page := &tm.Screens.Active.Cursor.PagePin.Node.data
		expectEqual(t, uint(1), page.styles.Count())
		expect(t, page.styles.RefCount(tm.Screens.Active.Cursor.StyleID) > 0, "page.styles.refCount(page.memory, t.screens.active.cursor.style_id) > 0")
	}
}

// Zig: test "Terminal: mode 1047 alt screen plain"
func TestTm3_Terminal_mode_1047_alt_screen_plain(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	// Print on primary screen
	tm3PrintString(t, &tm, "1A")

	// Go to alt screen with mode 47
	tm3SwitchScreenMode(t, &tm, TerminalSwitchScreenMode1047, true)
	expectEqual(t, ScreenSetKeyAlternate, tm.Screens.ActiveKey)

	// Screen should be empty
	{
		expectEqualStrings(t, "", tm3Plain(t, &tm))
	}

	// Print on alt screen. This should be off center because
	// we copy the cursor over from the primary screen
	tm3PrintString(t, &tm, "2B")
	{
		expectEqualStrings(t, "  2B", tm3Plain(t, &tm))
	}

	// Go back to primary
	tm3SwitchScreenMode(t, &tm, TerminalSwitchScreenMode1047, false)
	expectEqual(t, ScreenSetKeyPrimary, tm.Screens.ActiveKey)

	// Primary screen should still have the original content
	{
		expectEqualStrings(t, "1A", tm3Plain(t, &tm))
	}

	// Go back to alt screen with mode 1047
	tm3SwitchScreenMode(t, &tm, TerminalSwitchScreenMode1047, true)
	expectEqual(t, ScreenSetKeyAlternate, tm.Screens.ActiveKey)

	// Screen should be empty
	{
		expectEqualStrings(t, "", tm3Plain(t, &tm))
	}
}

// Zig: test "Terminal: mode 1047 copies cursor both directions"
func TestTm3_Terminal_mode_1047_copies_cursor_both_directions(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	// Color our cursor red
	tm3SetAttribute(t, &tm, tm3DirectFg(0xFF, 0, 0x7F))

	// Go to alt screen with mode 47
	tm3SwitchScreenMode(t, &tm, TerminalSwitchScreenMode1047, true)
	expectEqual(t, ScreenSetKeyAlternate, tm.Screens.ActiveKey)

	// Verify that our style is set
	{
		expect(t, tm.Screens.Active.Cursor.StyleID != StyleDefaultID, "t.screens.active.cursor.style_id != style.default_id")
		page := &tm.Screens.Active.Cursor.PagePin.Node.data
		expectEqual(t, uint(1), page.styles.Count())
		expect(t, page.styles.RefCount(tm.Screens.Active.Cursor.StyleID) > 0, "page.styles.refCount(page.memory, t.screens.active.cursor.style_id) > 0")
	}

	// Set a new style
	tm3SetAttribute(t, &tm, tm3DirectFg(0, 0xFF, 0))

	// Go back to primary
	tm3SwitchScreenMode(t, &tm, TerminalSwitchScreenMode1047, false)
	expectEqual(t, ScreenSetKeyPrimary, tm.Screens.ActiveKey)

	// Verify that our style is still set
	{
		expect(t, tm.Screens.Active.Cursor.StyleID != StyleDefaultID, "t.screens.active.cursor.style_id != style.default_id")
		page := &tm.Screens.Active.Cursor.PagePin.Node.data
		expectEqual(t, uint(1), page.styles.Count())
		expect(t, page.styles.RefCount(tm.Screens.Active.Cursor.StyleID) > 0, "page.styles.refCount(page.memory, t.screens.active.cursor.style_id) > 0")
	}
}

// Zig: test "Terminal: mode 1049 alt screen plain"
func TestTm3_Terminal_mode_1049_alt_screen_plain(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()

	// Print on primary screen
	tm3PrintString(t, &tm, "1A")

	// Go to alt screen with mode 47
	tm3SwitchScreenMode(t, &tm, TerminalSwitchScreenMode1049, true)
	expectEqual(t, ScreenSetKeyAlternate, tm.Screens.ActiveKey)

	// Screen should be empty
	{
		expectEqualStrings(t, "", tm3Plain(t, &tm))
	}

	// Print on alt screen. This should be off center because
	// we copy the cursor over from the primary screen
	tm3PrintString(t, &tm, "2B")
	{
		expectEqualStrings(t, "  2B", tm3Plain(t, &tm))
	}

	// Go back to primary
	tm3SwitchScreenMode(t, &tm, TerminalSwitchScreenMode1049, false)
	expectEqual(t, ScreenSetKeyPrimary, tm.Screens.ActiveKey)

	// Primary screen should still have the original content
	{
		expectEqualStrings(t, "1A", tm3Plain(t, &tm))
	}

	// Write, our cursor should be restored back.
	tm3PrintString(t, &tm, "C")
	{
		expectEqualStrings(t, "1AC", tm3Plain(t, &tm))
	}

	// Go back to alt screen with mode 1049
	tm3SwitchScreenMode(t, &tm, TerminalSwitchScreenMode1049, true)
	expectEqual(t, ScreenSetKeyAlternate, tm.Screens.ActiveKey)

	// Screen should be empty
	{
		expectEqualStrings(t, "", tm3Plain(t, &tm))
	}
}

// Reproduces a crash found by AFL++ fuzzer (afl-out/stream/default/crashes/
// id:000007,sig:06,src:004522). The crash is a page integrity violation
// "spacer tail not following wide" triggered during scrollUp -> deleteLines
// -> clearCells. When deleteLines count >= scroll region height, all rows
// are cleared (no shifting), so rowWillBeShifted is never called and wide
// characters straddling the right margin boundary leave orphaned spacer_tails.
//
// Zig: test "Terminal: deleteLines wide char at right margin with full clear"
func TestTm3_Terminal_deleteLines_wide_char_at_right_margin_with_full_clear(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(80, 24))
	defer tm.Deinit()

	// Place a wide character at col 39 (1-indexed) on several rows.
	// The wide cell lands at col 38 (0-indexed) with spacer_tail at col 39.
	tm.SetCursorPos(10, 39)
	tm3Print(t, &tm, 0x4E2D) // '中'

	// Set left/right scroll margins so scrolling_region.right = 38.
	// clearCells will clear cells[4..39], which includes the wide cell
	// at col 38 but NOT the spacer_tail at col 39.
	tm.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	tm.SetLeftAndRightMargin(5, 39)

	// scrollUp with count >= region height causes deleteLines to clear
	// ALL rows without any shifting, so rowWillBeShifted is never called
	// and the orphaned spacer_tail at col 39 triggers a page integrity
	// violation in clearCells.
	zigTry(t, tm.ScrollUp(uint(tm.Rows)))
}

// ---------------------------------------------------------------------------
// term51: additional tests (S). Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

// Zig: test "term51: MouseEvents.motion: every enum value"
func TestTm3_term51_MouseEvents_motion_every_enum_value(t *testing.T) {
	testStdCapacity(t)
	expect(t, !TerminalMouseEventsNone.Motion(), "!MouseEvents.none.motion()")
	expect(t, !TerminalMouseEventsX10.Motion(), "!MouseEvents.x10.motion()")
	expect(t, !TerminalMouseEventsNormal.Motion(), "!MouseEvents.normal.motion()")
	expect(t, TerminalMouseEventsButton.Motion(), "MouseEvents.button.motion()")
	expect(t, TerminalMouseEventsAny.Motion(), "MouseEvents.any.motion()")

	// The numeric values are part of the packed flags layout.
	expectEqual(t, uint8(0), uint8(TerminalMouseEventsNone))
	expectEqual(t, uint8(1), uint8(TerminalMouseEventsX10))
	expectEqual(t, uint8(2), uint8(TerminalMouseEventsNormal))
	expectEqual(t, uint8(3), uint8(TerminalMouseEventsButton))
	expectEqual(t, uint8(4), uint8(TerminalMouseEventsAny))
	// Zig: try std.testing.expectEqual(@as(usize, 5), @typeInfo(MouseEvents).@"enum".fields.len);
	// The field count is written out as the value list: five values, none..any.
	expectEqual(t, 5, len([]TerminalMouseEvents{
		TerminalMouseEventsNone,
		TerminalMouseEventsX10,
		TerminalMouseEventsNormal,
		TerminalMouseEventsButton,
		TerminalMouseEventsAny,
	}))
}

// Zig: test "term51: Options: defaults"
func TestTm3_term51_Options_defaults(t *testing.T) {
	testStdCapacity(t)
	opts := terminalOptionsDefault(7, 3)
	expectEqual(t, SizeCellCountInt(7), opts.Cols)
	expectEqual(t, SizeCellCountInt(3), opts.Rows)
	expectEqual(t, uint64(10_000), opts.MaxScrollback)
	expectEqual(t, TerminalColorsDefault.Background, opts.Colors.Background)
	expectEqual(t, TerminalColorsDefault.Foreground, opts.Colors.Foreground)
	expectEqual(t, TerminalColorsDefault.Cursor, opts.Colors.Cursor)
	expectEqual(t, TerminalColorsDefault.Palette.Current, opts.Colors.Palette.Current)
	expectEqual(t, ModesModePackedInit(), opts.DefaultModes)
	// Wraparound is the only default-on mode among those we care about.
	expect(t, sm1_modesField(&opts.DefaultModes, "wraparound"), "opts.default_modes.wraparound")
	expect(t, !sm1_modesField(&opts.DefaultModes, "insert"), "!opts.default_modes.insert")
	expect(t, !sm1_modesField(&opts.DefaultModes, "origin"), "!opts.default_modes.origin")
	expect(t, !sm1_modesField(&opts.DefaultModes, "grapheme_cluster"), "!opts.default_modes.grapheme_cluster")
}

// Zig: test "term51: init: initial state"
func TestTm3_term51_init_initial_state(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(20, 6))
	defer tm.Deinit()

	expectEqual(t, SizeCellCountInt(20), tm.Cols)
	expectEqual(t, SizeCellCountInt(6), tm.Rows)
	expectEqual(t, uint32(0), tm.WidthPx)
	expectEqual(t, uint32(0), tm.HeightPx)
	expectEqual(t, ANSIStatusDisplayMain, tm.StatusDisplay)
	expectEqual(t, None[uint32](), tm.PreviousChar)
	expectEqual(t, MouseShapeText, tm.MouseShape)

	// Scrolling region is the full screen.
	expectEqual(t, SizeCellCountInt(0), tm.ScrollingRegion.Top)
	expectEqual(t, SizeCellCountInt(5), tm.ScrollingRegion.Bottom)
	expectEqual(t, SizeCellCountInt(0), tm.ScrollingRegion.Left)
	expectEqual(t, SizeCellCountInt(19), tm.ScrollingRegion.Right)

	// Primary screen is active, the alternate screen does not exist yet.
	expectEqual(t, ScreenSetKeyPrimary, tm.Screens.ActiveKey)
	expect(t, tm.Screens.Get(ScreenSetKeyPrimary) != nil, "t.screens.get(.primary) != null")
	expect(t, tm.Screens.Get(ScreenSetKeyAlternate) == nil, "t.screens.get(.alternate) == null")
	expect(t, tm.Screens.Active == tm.Screens.Get(ScreenSetKeyPrimary), "t.screens.active == t.screens.get(.primary).?")
	tm3SExpectCursor(t, &tm, 0, 0, false)
	expect(t, !tm.Screens.Active.SavedCursor.Valid, "t.screens.active.saved_cursor == null")

	// No pwd.
	expectEqual(t, uint(0), uint(len(tm.Pwd)))
	expect(t, !tm.GetPwd().Valid, "t.getPwd() == null")

	// The flags are at their defaults.
	expectEqual(t, OSCParsersSemanticPromptRedrawTrue, tm.Flags.ShellRedrawsPrompt)
	expect(t, !tm.Flags.ModifyOtherKeys2, "!t.flags.modify_other_keys_2")
	expectEqual(t, TerminalMouseEventsNone, tm.Flags.MouseEvent)
	expectEqual(t, TerminalMouseFormatX10, tm.Flags.MouseFormat)
	expect(t, tm.Flags.Focused, "t.flags.focused")
	expect(t, !tm.Flags.PasswordInput, "!t.flags.password_input")
	expect(t, !tm.Flags.SelectionScroll, "!t.flags.selection_scroll")
	expect(t, !tm.Flags.SearchViewportDirty, "!t.flags.search_viewport_dirty")
	expect(t, !tm.Flags.Dirty.Palette, "!t.flags.dirty.palette")
	expect(t, !tm.Flags.Dirty.ReverseColors, "!t.flags.dirty.reverse_colors")
	expect(t, !tm.Flags.Dirty.Clear, "!t.flags.dirty.clear")
	expect(t, !tm.Flags.Dirty.Preedit, "!t.flags.dirty.preedit")

	// Empty screen.
	tm3SExpectPlain(t, &tm, "")
}

// Zig: test "term51: init: tabstops every 8 columns excluding column 0 and the last"
func TestTm3_term51_init_tabstops_every_8_columns_excluding_column_0_and_the_last(t *testing.T) {
	testStdCapacity(t)
	{
		tm := tm3Init(t, terminalOptionsDefault(20, 2))
		defer tm.Deinit()
		for col := uint(0); col < 20; col++ {
			expected := col == 8 || col == 16
			expectEqual(t, expected, tm.Tabstops.Get(col))
		}
	}

	// The stop loop is `i < cols - 1`: 16 is excluded when cols is 17 and
	// included when cols is 18.
	{
		tm := tm3Init(t, terminalOptionsDefault(17, 2))
		defer tm.Deinit()
		expect(t, tm.Tabstops.Get(8), "t.tabstops.get(8)")
		expect(t, !tm.Tabstops.Get(16), "!t.tabstops.get(16)")
	}
	{
		tm := tm3Init(t, terminalOptionsDefault(18, 2))
		defer tm.Deinit()
		expect(t, tm.Tabstops.Get(8), "t.tabstops.get(8)")
		expect(t, tm.Tabstops.Get(16), "t.tabstops.get(16)")
	}

	// Very narrow terminals have no tabstops at all.
	{
		tm := tm3Init(t, terminalOptionsDefault(8, 2))
		defer tm.Deinit()
		for col := uint(0); col < 8; col++ {
			expect(t, !tm.Tabstops.Get(col), "!t.tabstops.get(col)")
		}
	}
}

// Zig: test "term51: init: minimum size 1x1"
func TestTm3_term51_init_minimum_size_1x1(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(1, 1))
	defer tm.Deinit()

	expectEqual(t, SizeCellCountInt(1), tm.Cols)
	expectEqual(t, SizeCellCountInt(1), tm.Rows)
	expectEqual(t, SizeCellCountInt(0), tm.ScrollingRegion.Top)
	expectEqual(t, SizeCellCountInt(0), tm.ScrollingRegion.Bottom)
	expectEqual(t, SizeCellCountInt(0), tm.ScrollingRegion.Left)
	expectEqual(t, SizeCellCountInt(0), tm.ScrollingRegion.Right)
	tm3SExpectCursor(t, &tm, 0, 0, false)
}

// Zig: test "term51: init: colors option is copied into the terminal"
func TestTm3_term51_init_colors_option_is_copied_into_the_terminal(t *testing.T) {
	testStdCapacity(t)
	rgb := ColorRGB{R: 1, G: 2, B: 3}
	colors := TerminalColorsDefault
	colors.Background = ColorDynamicRGBInit(rgb)
	colors.Foreground = ColorDynamicRGB{Override: Some(ColorRGB{R: 9, G: 8, B: 7}), Default: None[ColorRGB]()}
	colors.Palette.Set(5, rgb)

	opts := terminalOptionsDefault(4, 4)
	opts.Colors = colors
	tm := tm3Init(t, opts)
	defer tm.Deinit()

	expectEqual(t, Some(rgb), tm.Colors.Background.Default)
	expectEqual(t, None[ColorRGB](), tm.Colors.Background.Override)
	expectEqual(t, Some(ColorRGB{R: 9, G: 8, B: 7}), tm.Colors.Foreground.Override)
	expectEqual(t, None[ColorRGB](), tm.Colors.Foreground.Default)
	expectEqual(t, ColorDynamicRGBUnset, tm.Colors.Cursor)
	expectEqual(t, rgb, tm.Colors.Palette.Current[5])
	expect(t, tm.Colors.Palette.Mask.isSet(5), "t.colors.palette.mask.isSet(5)")
	expect(t, !tm.Colors.Palette.Mask.isSet(4), "!t.colors.palette.mask.isSet(4)")
}

// Zig: test "term51: init: default colors when option omitted"
func TestTm3_term51_init_default_colors_when_option_omitted(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(4, 4))
	defer tm.Deinit()
	expectEqual(t, ColorDynamicRGBUnset, tm.Colors.Background)
	expectEqual(t, ColorDynamicRGBUnset, tm.Colors.Foreground)
	expectEqual(t, ColorDynamicRGBUnset, tm.Colors.Cursor)
	expectEqual(t, TerminalColorsDefault.Palette.Current, tm.Colors.Palette.Current)
	expectEqual(t, 0, tm.Colors.Palette.Mask.count())
}

// Zig: test "term51: init: default modes seed both values and defaults"
func TestTm3_term51_init_default_modes_seed_both_values_and_defaults(t *testing.T) {
	testStdCapacity(t)
	{
		tm := tm3Init(t, terminalOptionsDefault(4, 4))
		defer tm.Deinit()
		expect(t, tm.Modes.Get(ModesModeWraparound), "t.modes.get(.wraparound)")
		expect(t, !tm.Modes.Get(ModesModeInsert), "!t.modes.get(.insert)")
		expect(t, !tm.Modes.Get(ModesModeOrigin), "!t.modes.get(.origin)")
		expect(t, !tm.Modes.Get(ModesModeGraphemeCluster), "!t.modes.get(.grapheme_cluster)")
		expectEqual(t, ModesModePackedInit(), tm.Modes.Values)
		expectEqual(t, ModesModePackedInit(), tm.Modes.Default)
	}
	{
		dm := ModesModePackedInit()
		sm1_modesFieldSet(&dm, "insert", true)
		sm1_modesFieldSet(&dm, "wraparound", false)
		sm1_modesFieldSet(&dm, "grapheme_cluster", true)
		opts := terminalOptionsDefault(4, 4)
		opts.DefaultModes = dm
		tm := tm3Init(t, opts)
		defer tm.Deinit()
		expect(t, tm.Modes.Get(ModesModeInsert), "t.modes.get(.insert)")
		expect(t, !tm.Modes.Get(ModesModeWraparound), "!t.modes.get(.wraparound)")
		expect(t, tm.Modes.Get(ModesModeGraphemeCluster), "t.modes.get(.grapheme_cluster)")
		expectEqual(t, dm, tm.Modes.Values)
		expectEqual(t, dm, tm.Modes.Default)

		// The saved defaults are independent of later changes to values.
		tm.Modes.Set(ModesModeInsert, false)
		expect(t, sm1_modesField(&tm.Modes.Default, "insert"), "t.modes.default.insert")
	}
}

// Zig: test "term51: init: max_scrollback is forwarded to the screen"
func TestTm3_term51_init_max_scrollback_is_forwarded_to_the_screen(t *testing.T) {
	testStdCapacity(t)
	{
		tm := tm3Init(t, terminalOptionsDefault(4, 2))
		defer tm.Deinit()
		expectEqual(t, uint64(10_000), tm.Screens.Active.Pages.explicit_max_size)
		expect(t, !tm.Screens.Active.NoScrollback, "!t.screens.active.no_scrollback")
	}
	{
		opts := terminalOptionsDefault(4, 2)
		opts.MaxScrollback = 12345
		tm := tm3Init(t, opts)
		defer tm.Deinit()
		expectEqual(t, uint64(12345), tm.Screens.Active.Pages.explicit_max_size)
		expect(t, !tm.Screens.Active.NoScrollback, "!t.screens.active.no_scrollback")
	}
	{
		opts := terminalOptionsDefault(4, 2)
		opts.MaxScrollback = 0
		tm := tm3Init(t, opts)
		defer tm.Deinit()
		expectEqual(t, uint64(0), tm.Screens.Active.Pages.explicit_max_size)
		expect(t, tm.Screens.Active.NoScrollback, "t.screens.active.no_scrollback")
	}
}

// Zig: test "term51: deinit: frees pwd, tabstops and screens"
func TestTm3_term51_deinit_frees_pwd_tabstops_and_screens(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(300, 10))
	zigTry(t, tm.SetPwd([]uint8("file:///home/user/some/long/path")))
	expectEqualStrings(t, "file:///home/user/some/long/path", string(zigUnwrap(t, tm.GetPwd())))
	// Force the alternate screen to exist too so that both are freed.
	_, err := tm.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	tm3Print(t, &tm, 'x')
	tm.Deinit()
	// Zig-only: The leak detector of the testing allocator verifies the rest.
}

// Zig: test "term51: gpa: returns the allocator of the active screen"
func TestTm3_term51_gpa_returns_the_allocator_of_the_active_screen(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(4, 4))
	defer tm.Deinit()

	tm.gpa() // Zig: const a = t.gpa();
	// Zig-only: try std.testing.expect(a.ptr == std.testing.allocator.ptr);
	// Zig-only: try std.testing.expect(a.vtable == std.testing.allocator.vtable);
	// Zig-only: try std.testing.expect(a.ptr == t.screens.active.alloc.ptr);

	// The allocator is the same once the alternate screen is active.
	_, err := tm.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	tm.gpa() // Zig: const b = t.gpa();
	// Zig-only: try std.testing.expect(b.ptr == std.testing.allocator.ptr);
	// Zig-only: try std.testing.expect(b.vtable == std.testing.allocator.vtable);
}

// Zig: test "term51: vtHandler: handler wraps this terminal"
func TestTm3_term51_vtHandler_handler_wraps_this_terminal(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 3))
	defer tm.Deinit()

	h := tm.VtHandler()
	defer h.Deinit()
	expect(t, h.terminal == &tm, "h.terminal == &t")

	zigTry(t, h.Vt(StreamActionKeyPrint, &StreamAction{Tag: StreamActionKeyPrint, Print: StreamActionPrint{Cp: 'h'}}))
	zigTry(t, h.Vt(StreamActionKeyPrint, &StreamAction{Tag: StreamActionKeyPrint, Print: StreamActionPrint{Cp: 'i'}}))
	tm3SExpectPlain(t, &tm, "hi")
	tm3SExpectCursor(t, &tm, 2, 0, false)

	zigTry(t, h.Vt(StreamActionKeyCarriageReturn, &StreamAction{Tag: StreamActionKeyCarriageReturn}))
	zigTry(t, h.Vt(StreamActionKeyLinefeed, &StreamAction{Tag: StreamActionKeyLinefeed}))
	zigTry(t, h.Vt(StreamActionKeyPrint, &StreamAction{Tag: StreamActionKeyPrint, Print: StreamActionPrint{Cp: 'x'}}))
	tm3SExpectPlain(t, &tm, "hi\nx")
}

// Zig: test "term51: vtStream: processes VT bytes into terminal state"
func TestTm3_term51_vtStream_processes_VT_bytes_into_terminal_state(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 4))
	defer tm.Deinit()

	s := tm.VtStream()
	defer s.Deinit()
	expect(t, s.handler.terminal == &tm, "s.handler.terminal == &t")

	zigTry(t, s.NextSlice([]uint8("hello\r\nworld")))
	tm3SExpectPlain(t, &tm, "hello\nworld")
	tm3SExpectCursor(t, &tm, 5, 1, false)

	// Cursor movement and erase sequences.
	zigTry(t, s.NextSlice([]uint8("\x1b[1;1H\x1b[2C!")))
	tm3SExpectPlain(t, &tm, "he!lo\nworld")

	// SGR style then print.
	zigTry(t, s.NextSlice([]uint8("\x1b[2;1H\x1b[1mB")))
	{
		lc := tm3SAt(t, &tm, 0, 1)
		expectEqual(t, uint32('B'), lc.Cell.Codepoint())
		expect(t, lc.Cell.style_id != StyleDefaultID, "lc.cell.style_id != style.default_id")
		expect(t, lc.Node.data.styles.Get(lc.Cell.style_id).Flags.Bold, "lc.node.data.styles.get(lc.node.data.memory, lc.cell.style_id).flags.bold")
	}
}

// Zig: test "term51: vtStream: queries produce no output and no state change"
func TestTm3_term51_vtStream_queries_produce_no_output_and_no_state_change(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 4))
	defer tm.Deinit()

	s := tm.VtStream()
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("ab")))
	// DSR cursor position report, device attributes, XTVERSION: all ignored.
	zigTry(t, s.NextSlice([]uint8("\x1b[6n\x1b[c\x1b[>q")))
	tm3SExpectPlain(t, &tm, "ab")
	tm3SExpectCursor(t, &tm, 2, 0, false)
}

// Zig: test "term51: vtStream: the stream is given the terminal allocator for long OSC"
func TestTm3_term51_vtStream_the_stream_is_given_the_terminal_allocator_for_long_OSC(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 4))
	defer tm.Deinit()

	s := tm.VtStream()
	defer s.Deinit()
	// osc_parser.alloc is `any` in Go: nil is Zig's null, any other value is
	// "an allocator was provided".
	expect(t, s.parser.osc_parser.alloc, "s.parser.osc_parser.alloc != null")
	// Zig-only: try std.testing.expect(s.parser.osc_parser.alloc.?.ptr == std.testing.allocator.ptr);

	// An OSC payload far larger than the fixed OSC buffer must be accepted
	// (using the allocator) and not corrupt following output.
	var long [5000]uint8
	for i := range long {
		long[i] = 'a'
	}
	zigTry(t, s.NextSlice([]uint8("\x1b]2;")))
	zigTry(t, s.NextSlice(long[:]))
	zigTry(t, s.NextSlice([]uint8("\x07ok")))
	tm3SExpectPlain(t, &tm, "ok")
}

// Zig: test "term51: printString: ascii, empty and newline handling"
func TestTm3_term51_printString_ascii_empty_and_newline_handling(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 4))
	defer tm.Deinit()

	// Empty string is a no-op.
	tm3PrintString(t, &tm, "")
	tm3SExpectCursor(t, &tm, 0, 0, false)
	expectEqual(t, None[uint32](), tm.PreviousChar)

	tm3PrintString(t, &tm, "abc")
	tm3SExpectCursor(t, &tm, 3, 0, false)
	expectEqual(t, Some(uint32('c')), tm.PreviousChar)

	// '\n' is carriage return plus linefeed (no linefeed mode needed).
	tm3PrintString(t, &tm, "\nxy")
	tm3SExpectCursor(t, &tm, 2, 1, false)
	tm3SExpectPlain(t, &tm, "abc\nxy")

	// Consecutive newlines leave blank rows.
	tm3PrintString(t, &tm, "\n\nz")
	tm3SExpectPlain(t, &tm, "abc\nxy\n\nz")
	tm3SExpectCursor(t, &tm, 1, 3, false)

	// Newline at the bottom scrolls.
	tm3PrintString(t, &tm, "\nq")
	tm3SExpectPlain(t, &tm, "xy\n\nz\nq")
	tm3SExpectCursor(t, &tm, 1, 3, false)
}

// Zig: test "term51: printString: newline clears pending wrap and returns to column zero"
func TestTm3_term51_printString_newline_clears_pending_wrap_and_returns_to_column_zero(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(3, 4))
	defer tm.Deinit()

	tm3PrintString(t, &tm, "abc")
	tm3SExpectCursor(t, &tm, 2, 0, true)
	tm3PrintString(t, &tm, "\nd")
	tm3SExpectCursor(t, &tm, 1, 1, false)
	tm3SExpectPlain(t, &tm, "abc\nd")
	// The newline did not soft wrap the row.
	expect(t, !tm3SAt(t, &tm, 0, 0).Row.wrap, "!term51SAt(&t, 0, 0).row.wrap")
	expect(t, !tm3SAt(t, &tm, 0, 1).Row.wrap_continuation, "!term51SAt(&t, 0, 1).row.wrap_continuation")
}

// Zig: test "term51: printString: newline with a left margin returns to the left margin"
func TestTm3_term51_printString_newline_with_a_left_margin_returns_to_the_left_margin(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 4))
	defer tm.Deinit()

	tm.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	tm.SetLeftAndRightMargin(3, 8)
	tm.SetCursorPos(1, 5)
	tm3PrintString(t, &tm, "a\nb")
	tm3SExpectCell(t, &tm, 4, 0, 'a', PageCellWideNarrow)
	// carriageReturn moves to the left margin since x >= left.
	tm3SExpectCell(t, &tm, 2, 1, 'b', PageCellWideNarrow)
}

// Zig: test "term51: printString: carriage return and tab are printed as cells"
func TestTm3_term51_printString_carriage_return_and_tab_are_printed_as_cells(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 2))
	defer tm.Deinit()

	// Only '\n' is special: other control characters go through print and
	// occupy a cell with the control codepoint.
	tm3PrintString(t, &tm, "a\rb\tc")
	tm3SExpectCursor(t, &tm, 5, 0, false)
	tm3SExpectCell(t, &tm, 0, 0, 'a', PageCellWideNarrow)
	tm3SExpectCell(t, &tm, 1, 0, '\r', PageCellWideNarrow)
	tm3SExpectCell(t, &tm, 2, 0, 'b', PageCellWideNarrow)
	tm3SExpectCell(t, &tm, 3, 0, '\t', PageCellWideNarrow)
	tm3SExpectCell(t, &tm, 4, 0, 'c', PageCellWideNarrow)
}

// Zig: test "term51: printString: invalid utf8 is an error and prints nothing"
func TestTm3_term51_printString_invalid_utf8_is_an_error_and_prints_nothing(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 2))
	defer tm.Deinit()

	// The whole string is validated before anything is printed.
	tm3ExpectError(t, ErrInvalidUtf8, tm.PrintString([]uint8("ab\xFFcd")))
	tm3SExpectCursor(t, &tm, 0, 0, false)
	tm3SExpectPlain(t, &tm, "")

	// Truncated multi-byte sequence.
	tm3ExpectError(t, ErrInvalidUtf8, tm.PrintString([]uint8("a\xE4\xB8")))
	tm3SExpectPlain(t, &tm, "")

	// Lone continuation byte.
	tm3ExpectError(t, ErrInvalidUtf8, tm.PrintString([]uint8("\x80")))
	// Overlong encoding of '/'.
	tm3ExpectError(t, ErrInvalidUtf8, tm.PrintString([]uint8("\xC0\xAF")))
	// Encoded surrogate.
	tm3ExpectError(t, ErrInvalidUtf8, tm.PrintString([]uint8("\xED\xA0\x80")))
	tm3SExpectPlain(t, &tm, "")
}

// Zig: test "term51: printString: multibyte, wide and combining text"
func TestTm3_term51_printString_multibyte_wide_and_combining_text(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 2))
	defer tm.Deinit()

	// 2-byte, 3-byte (wide), 4-byte (wide) sequences.
	tm3PrintString(t, &tm, "é中\U0001F600")
	tm3SExpectCell(t, &tm, 0, 0, 0xE9, PageCellWideNarrow)
	tm3SExpectCell(t, &tm, 1, 0, 0x4E2D, PageCellWideWide)
	tm3SExpectCell(t, &tm, 2, 0, 0, PageCellWideSpacerTail)
	tm3SExpectCell(t, &tm, 3, 0, 0x1F600, PageCellWideWide)
	tm3SExpectCell(t, &tm, 4, 0, 0, PageCellWideSpacerTail)
	tm3SExpectCursor(t, &tm, 5, 0, false)

	// Combining mark (mode 2027 off) attaches to the previous cell.
	tm3PrintString(t, &tm, "é")
	tm3SExpectCell(t, &tm, 5, 0, 'e', PageCellWideNarrow)
	tm3SExpectGrapheme(t, &tm, 5, 0, []uint32{0x301})
	tm3SExpectCursor(t, &tm, 6, 0, false)
}

// Zig: test "term51: printRepeat: nothing printed before any character"
func TestTm3_term51_printRepeat_nothing_printed_before_any_character(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 2))
	defer tm.Deinit()

	zigTry(t, tm.PrintRepeat(5))
	tm3SExpectCursor(t, &tm, 0, 0, false)
	tm3SExpectPlain(t, &tm, "")
	expectEqual(t, None[uint32](), tm.PreviousChar)
}

// Zig: test "term51: printRepeat: count zero repeats once and counts repeat exactly"
func TestTm3_term51_printRepeat_count_zero_repeats_once_and_counts_repeat_exactly(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(20, 2))
	defer tm.Deinit()

	tm3Print(t, &tm, 'a')
	zigTry(t, tm.PrintRepeat(0))
	tm3SExpectPlain(t, &tm, "aa")
	zigTry(t, tm.PrintRepeat(1))
	tm3SExpectPlain(t, &tm, "aaa")
	zigTry(t, tm.PrintRepeat(4))
	tm3SExpectPlain(t, &tm, "aaaaaaa")
	tm3SExpectCursor(t, &tm, 7, 0, false)
	expectEqual(t, Some(uint32('a')), tm.PreviousChar)
}

// Zig: test "term51: printRepeat: wraps across lines"
func TestTm3_term51_printRepeat_wraps_across_lines(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(4, 4))
	defer tm.Deinit()

	tm3Print(t, &tm, 'z')
	zigTry(t, tm.PrintRepeat(9))
	tm3SExpectPlain(t, &tm, "zzzz\nzzzz\nzz")
	tm3SExpectCursor(t, &tm, 2, 2, false)
}

// Zig: test "term51: printRepeat: repeats a wide character as wide"
func TestTm3_term51_printRepeat_repeats_a_wide_character_as_wide(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(6, 3))
	defer tm.Deinit()

	tm3Print(t, &tm, 0x4E2D)
	zigTry(t, tm.PrintRepeat(2))
	tm3SExpectCell(t, &tm, 0, 0, 0x4E2D, PageCellWideWide)
	tm3SExpectCell(t, &tm, 1, 0, 0, PageCellWideSpacerTail)
	tm3SExpectCell(t, &tm, 2, 0, 0x4E2D, PageCellWideWide)
	tm3SExpectCell(t, &tm, 3, 0, 0, PageCellWideSpacerTail)
	tm3SExpectCell(t, &tm, 4, 0, 0x4E2D, PageCellWideWide)
	tm3SExpectCell(t, &tm, 5, 0, 0, PageCellWideSpacerTail)
	tm3SExpectCursor(t, &tm, 5, 0, true)
}

// Zig: test "term51: printRepeat: previous char survives cursor movement and is not set by zero-width input"
func TestTm3_term51_printRepeat_previous_char_survives_cursor_movement_and_is_not_set_by_zero_width_input(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(10, 3))
	defer tm.Deinit()

	tm3Print(t, &tm, 'q')
	tm.SetCursorPos(2, 3)
	zigTry(t, tm.PrintRepeat(2))
	tm3SExpectCell(t, &tm, 2, 1, 'q', PageCellWideNarrow)
	tm3SExpectCell(t, &tm, 3, 1, 'q', PageCellWideNarrow)

	// A combining mark (non-2027) is attached and does not replace
	// previous_char.
	tm3Print(t, &tm, 0x301)
	expectEqual(t, Some(uint32('q')), tm.PreviousChar)
	zigTry(t, tm.PrintRepeat(1))
	tm3SExpectCell(t, &tm, 4, 1, 'q', PageCellWideNarrow)
}

// Go-only: panic test: Terminal.init: cols is 0 (Zig: Tabstops.reset and
// scrolling_region.right compute `cols - 1`, an integer underflow that traps)
//
// Go: an error since the zero-size ruling (docs/public-api.md, "Wrong
// arguments"); the test keeps its name.
func TestTm3_panic_init_cols_zero(t *testing.T) {
	testStdCapacity(t)
	_, err := TerminalInit(terminalOptionsDefault(0, 5))
	if err := error(err); err != ErrZeroColCount {
		t.Fatalf("got %v, want %v", err, ErrZeroColCount)
	}
}

// Go-only: panic test: Terminal.init: rows is 0 (Zig: the page list has no
// first page, `page_list.first.?` of null, and scrolling_region.bottom computes
// `rows - 1`)
//
// Go: an error since the zero-size ruling; the test keeps its name.
func TestTm3_panic_init_rows_zero(t *testing.T) {
	testStdCapacity(t)
	_, err := TerminalInit(terminalOptionsDefault(5, 0))
	if err := error(err); err != ErrZeroRowCount {
		t.Fatalf("got %v, want %v", err, ErrZeroRowCount)
	}
}

// Go-only: panic test: Terminal.resize: cols is 0 (Zig: Tabstops.reset
// computes `cols - 1`, an integer underflow that traps)
func TestTm3_panic_resize_cols_zero(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()
	// Go: an error since the zero-size ruling; the test keeps its name.
	if err := error(tm.Resize(0, 5)); err != ErrZeroColCount {
		t.Fatalf("got %v, want %v", err, ErrZeroColCount)
	}
}

// Go-only: panic test: Terminal.resize: rows is 0 (Zig: the scrolling region
// computes `rows - 1`, an integer underflow that traps)
func TestTm3_panic_resize_rows_zero(t *testing.T) {
	testStdCapacity(t)
	tm := tm3Init(t, terminalOptionsDefault(5, 5))
	defer tm.Deinit()
	// Go: an error since the zero-size ruling; the test keeps its name.
	if err := error(tm.Resize(5, 0)); err != ErrZeroRowCount {
		t.Fatalf("got %v, want %v", err, ErrZeroRowCount)
	}
}

// Go-only: ruling (Terminal.setCursorPos adds in 64-bit usize on every platform: a huge column clamps to the right margin, also on 32-bit ARM)
func TestTm3_go_only_setCursorPos_huge_column_clamps_to_margin(t *testing.T) {
	testStdCapacity(t)
	term, err := TerminalInit(terminalOptionsDefault(20, 10))
	zigTry(t, err)
	defer term.Deinit()
	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(3, 8)
	term.SetTopAndBottomMargin(2, 6)
	term.Modes.Set(ModesModeOrigin, true)
	// 0xFFFFFFFF + left margin overflows a 32-bit uint but not the 64-bit
	// usize of the reference.
	term.SetCursorPos(1, 0xFFFFFFFF)
	expectEqual(t, SizeCellCountInt(7), term.Screens.Active.Cursor.X)
	expectEqual(t, SizeCellCountInt(1), term.Screens.Active.Cursor.Y)
}

// Go-only: panic test: Terminal.setPwd: pwd aliases the terminal's own pwd (Zig: @memcpy arguments alias)
func TestTm3_panic_setPwd_with_its_own_pwd(t *testing.T) {
	testStdCapacity(t)
	term, err := TerminalInit(terminalOptionsDefault(5, 5))
	zigTry(t, err)
	defer term.Deinit()
	zigTry(t, term.SetPwd([]uint8("/home/user")))
	tm3ExpectPanic(t, func() { _ = term.SetPwd(term.GetPwd().Value) })
	// A copy does not alias and works as in Zig.
	term2, err := TerminalInit(terminalOptionsDefault(5, 5))
	zigTry(t, err)
	defer term2.Deinit()
	zigTry(t, term2.SetPwd([]uint8("/home/user")))
	zigTry(t, term2.SetPwd(append([]uint8(nil), term2.GetPwd().Value...)))
	expectEqualStrings(t, "/home/user", string(term2.GetPwd().Value))
	// So does a shorter slice of a different buffer.
	zigTry(t, term2.SetPwd([]uint8("/tmp")))
	expectEqualStrings(t, "/tmp", string(term2.GetPwd().Value))
}
