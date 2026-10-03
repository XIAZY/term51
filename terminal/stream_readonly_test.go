// Tests translated one for one from src/terminal/stream_readonly.zig (Ghostty
// v1.3.1 plus the term51 additional tests). Each Go test carries the exact Zig
// title in a "// Zig: test" comment. The helpers expect, expectEqual,
// expectEqualStrings, zigTry and zigUnwrap are defined in other test files of
// the package (Parser_test.go, dcs_test.go); the helpers of this file are
// prefixed sro.
//
// Translation notes:
//   - The Zig variable `t` (the Terminal) is `tm` in Go, because `t` is the
//     *testing.T. `var s: Stream = .initAlloc(testing.allocator, .init(&t))`
//     is `h := StreamReadonlyHandlerInit(&tm)` and
//     `s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)`;
//     `s.handler` is then the pointer &h. sroAllocator stands in for
//     testing.allocator (any non-nil value is "an allocator was provided").
//   - `h.vt(.tag, value)` is `h.Vt(tag, action)` with the whole StreamAction:
//     sroVt builds the call and fails the test on an error (Zig `try`), and
//     the sro<Name>Act functions build the action for each payload shape.
//   - Blocks `{ var t = ...; defer t.deinit(...); ... }`, and the bodies of
//     loops that create a terminal per iteration, are immediately invoked
//     function literals so that the deferred Deinit runs at the end of the
//     block as in Zig. `inline for` over comptime lists is an ordinary loop.
//   - term51DSnap is sroSnap. Zig's `text_hash: u64` (Wyhash of the screen
//     text) is the text itself (`text string`): the comparison is the same
//     and the Go standard library has no Wyhash. `modes`, `saved_modes` and
//     `flags`, which Zig compares as packed integers, are the structs.
//   - `@sizeOf(Handler)` is unsafe.Sizeof.
//   - Allocator and leak checks (`defer list.deinit(testing.allocator)`,
//     freeing the parser's command list) are dropped with `// Zig-only:`.

package terminal

import (
	"strings"
	"testing"
	"unsafe"
)

// sroAllocator stands in for std.testing.allocator.
var sroAllocator any = struct{}{}

// sroMaxCols is term51D_max_cols.
const sroMaxCols = 300

// sroSnap is term51DSnap: everything about a terminal that the readonly
// handler could possibly touch, as plain values so that two snapshots can be
// compared with expectEqual. The screen text is included (Zig: as a hash).
type sroSnap struct {
	x                     SizeCellCountInt
	y                     SizeCellCountInt
	cursor_style          ScreenCursorStyle
	pending_wrap          bool
	cursor_protected      bool
	cursor_style_value    Style
	hyperlink_id          HyperlinkId
	hyperlink_implicit_id SizeOffsetInt
	semantic_content      PageCellSemanticContent
	protected_mode        ANSIProtectedMode
	saved_cursor          Optional[ScreenSavedCursor]
	charset               ScreenCharsetState
	kitty_keyboard        KittyKeyFlagStack
	active_key            ScreenSetKey
	top                   SizeCellCountInt
	bottom                SizeCellCountInt
	left                  SizeCellCountInt
	right                 SizeCellCountInt
	cols                  SizeCellCountInt
	rows                  SizeCellCountInt
	modes                 ModesModePacked
	saved_modes           ModesModePacked
	flags                 TerminalFlags
	colors                TerminalColors
	status_display        ANSIStatusDisplay
	mouse_shape           MouseShape
	previous_char         Optional[uint32]
	pwd_len               uint
	tabs                  [sroMaxCols]bool
	text                  string
}

// sroSnapshot is term51DSnapshot.
func sroSnapshot(t *testing.T, tm *Terminal) sroSnap {
	t.Helper()
	scr := tm.screens.active
	var tabs [sroMaxCols]bool
	n := uint(tm.cols)
	if n > sroMaxCols {
		n = sroMaxCols
	}
	for i := uint(0); i < n; i++ {
		tabs[i] = tm.tabstops.Get(i)
	}
	str, err := tm.PlainString()
	zigTry(t, err)
	return sroSnap{
		x:                     scr.cursor.x,
		y:                     scr.cursor.y,
		cursor_style:          scr.cursor.cursor_style,
		pending_wrap:          scr.cursor.pending_wrap,
		cursor_protected:      scr.cursor.protected,
		cursor_style_value:    scr.cursor.style,
		hyperlink_id:          scr.cursor.hyperlink_id,
		hyperlink_implicit_id: scr.cursor.hyperlink_implicit_id,
		semantic_content:      scr.cursor.semantic_content,
		protected_mode:        scr.protected_mode,
		saved_cursor:          scr.saved_cursor,
		charset:               scr.charset,
		kitty_keyboard:        scr.kitty_keyboard,
		active_key:            tm.screens.active_key,
		top:                   tm.scrolling_region.top,
		bottom:                tm.scrolling_region.bottom,
		left:                  tm.scrolling_region.left,
		right:                 tm.scrolling_region.right,
		cols:                  tm.cols,
		rows:                  tm.rows,
		modes:                 tm.modes.values,
		saved_modes:           tm.modes.saved,
		flags:                 tm.flags,
		colors:                tm.colors,
		status_display:        tm.status_display,
		mouse_shape:           tm.mouse_shape,
		previous_char:         tm.previous_char,
		pwd_len:               uint(len(tm.pwd)),
		tabs:                  tabs,
		text:                  string(str),
	}
}

// sroModesWith is term51DModesWith: the mode bits of a snapshot with one mode
// changed.
func sroModesWith(bits ModesModePacked, mode ModesMode, value bool) ModesModePacked {
	state := ModesModeState{values: bits}
	state.Set(mode, value)
	return state.values
}

// sroVt is `try h.vt(tag, value)`: the action carries its own tag.
func sroVt(t *testing.T, h *StreamReadonlyHandler, a StreamAction) {
	t.Helper()
	zigTry(t, h.Vt(a.Tag, a))
}

// Constructors for the StreamAction of each payload shape.

// sroAct is an action without a payload (`{}`).
func sroAct(key StreamActionKey) StreamAction { return StreamAction{Tag: key} }

// sroPrintAct is `.print, .{ .cp = cp }`.
func sroPrintAct(cp uint32) StreamAction {
	return StreamAction{Tag: StreamActionKeyPrint, Print: StreamActionPrint{cp: cp}}
}

// sroModeAct is set_mode, reset_mode, save_mode, restore_mode or
// request_mode with `.{ .mode = mode }`.
func sroModeAct(key StreamActionKey, mode ModesMode) StreamAction {
	a := StreamAction{Tag: key}
	m := StreamActionMode{mode: mode}
	switch key {
	case StreamActionKeySetMode:
		a.SetMode = m
	case StreamActionKeyResetMode:
		a.ResetMode = m
	case StreamActionKeySaveMode:
		a.SaveMode = m
	case StreamActionKeyRestoreMode:
		a.RestoreMode = m
	case StreamActionKeyRequestMode:
		a.RequestMode = m
	default:
		panic("sroModeAct: not a mode action")
	}
	return a
}

// sroMarginAct is top_and_bottom_margin or left_and_right_margin with
// `.{ .top_left = a, .bottom_right = b }`.
func sroMarginAct(key StreamActionKey, top_left uint16, bottom_right uint16) StreamAction {
	a := StreamAction{Tag: key}
	m := StreamActionMargin{top_left: top_left, bottom_right: bottom_right}
	switch key {
	case StreamActionKeyTopAndBottomMargin:
		a.TopAndBottomMargin = m
	case StreamActionKeyLeftAndRightMargin:
		a.LeftAndRightMargin = m
	default:
		panic("sroMarginAct: not a margin action")
	}
	return a
}

// sroCountAct is an action whose payload is a usize: print_repeat,
// delete_chars, erase_chars, insert_lines, insert_blanks, delete_lines,
// scroll_up, scroll_down.
func sroCountAct(key StreamActionKey, n uint) StreamAction {
	a := StreamAction{Tag: key}
	switch key {
	case StreamActionKeyPrintRepeat:
		a.PrintRepeat = n
	case StreamActionKeyDeleteChars:
		a.DeleteChars = n
	case StreamActionKeyEraseChars:
		a.EraseChars = n
	case StreamActionKeyInsertLines:
		a.InsertLines = n
	case StreamActionKeyInsertBlanks:
		a.InsertBlanks = n
	case StreamActionKeyDeleteLines:
		a.DeleteLines = n
	case StreamActionKeyScrollUp:
		a.ScrollUp = n
	case StreamActionKeyScrollDown:
		a.ScrollDown = n
	default:
		panic("sroCountAct: not a usize action")
	}
	return a
}

// sroU16Act is an action whose payload is a u16: horizontal_tab,
// horizontal_tab_back, kitty_keyboard_pop, title_push, title_pop.
func sroU16Act(key StreamActionKey, n uint16) StreamAction {
	a := StreamAction{Tag: key}
	switch key {
	case StreamActionKeyHorizontalTab:
		a.HorizontalTab = n
	case StreamActionKeyHorizontalTabBack:
		a.HorizontalTabBack = n
	case StreamActionKeyKittyKeyboardPop:
		a.KittyKeyboardPop = n
	case StreamActionKeyTitlePush:
		a.TitlePush = n
	case StreamActionKeyTitlePop:
		a.TitlePop = n
	default:
		panic("sroU16Act: not a u16 action")
	}
	return a
}

// sroMoveAct is a cursor movement with `.{ .value = n }`.
func sroMoveAct(key StreamActionKey, n uint16) StreamAction {
	a := StreamAction{Tag: key}
	m := StreamActionCursorMovement{value: n}
	switch key {
	case StreamActionKeyCursorUp:
		a.CursorUp = m
	case StreamActionKeyCursorDown:
		a.CursorDown = m
	case StreamActionKeyCursorLeft:
		a.CursorLeft = m
	case StreamActionKeyCursorRight:
		a.CursorRight = m
	case StreamActionKeyCursorCol:
		a.CursorCol = m
	case StreamActionKeyCursorRow:
		a.CursorRow = m
	case StreamActionKeyCursorColRelative:
		a.CursorColRelative = m
	case StreamActionKeyCursorRowRelative:
		a.CursorRowRelative = m
	default:
		panic("sroMoveAct: not a cursor movement")
	}
	return a
}

// sroCursorPosAct is `.cursor_pos, .{ .row = row, .col = col }`.
func sroCursorPosAct(row uint16, col uint16) StreamAction {
	return StreamAction{Tag: StreamActionKeyCursorPos, CursorPos: StreamActionCursorPos{row: row, col: col}}
}

// sroBoolAct is an action whose payload is a bool: the erase_display_* and
// erase_line_* actions (the protected flag) and mouse_shift_capture.
func sroBoolAct(key StreamActionKey, b bool) StreamAction {
	a := StreamAction{Tag: key}
	switch key {
	case StreamActionKeyEraseDisplayBelow:
		a.EraseDisplayBelow = b
	case StreamActionKeyEraseDisplayAbove:
		a.EraseDisplayAbove = b
	case StreamActionKeyEraseDisplayComplete:
		a.EraseDisplayComplete = b
	case StreamActionKeyEraseDisplayScrollback:
		a.EraseDisplayScrollback = b
	case StreamActionKeyEraseDisplayScrollComplete:
		a.EraseDisplayScrollComplete = b
	case StreamActionKeyEraseLineRight:
		a.EraseLineRight = b
	case StreamActionKeyEraseLineLeft:
		a.EraseLineLeft = b
	case StreamActionKeyEraseLineComplete:
		a.EraseLineComplete = b
	case StreamActionKeyEraseLineRightUnlessPendingWrap:
		a.EraseLineRightUnlessPendingWrap = b
	case StreamActionKeyMouseShiftCapture:
		a.MouseShiftCapture = b
	default:
		panic("sroBoolAct: not a bool action")
	}
	return a
}

// sroKittyAct is kitty_keyboard_push, _set, _set_or or _set_not with
// `.{ .flags = flags }`.
func sroKittyAct(key StreamActionKey, flags KittyKeyFlags) StreamAction {
	a := StreamAction{Tag: key}
	f := StreamActionKittyKeyboardFlags{flags: flags}
	switch key {
	case StreamActionKeyKittyKeyboardPush:
		a.KittyKeyboardPush = f
	case StreamActionKeyKittyKeyboardSet:
		a.KittyKeyboardSet = f
	case StreamActionKeyKittyKeyboardSetOr:
		a.KittyKeyboardSetOr = f
	case StreamActionKeyKittyKeyboardSetNot:
		a.KittyKeyboardSetNot = f
	default:
		panic("sroKittyAct: not a kitty keyboard flags action")
	}
	return a
}

// sroSGRAct is `.set_attribute, attr`.
func sroSGRAct(tag SGRAttributeTag) StreamAction {
	return StreamAction{Tag: StreamActionKeySetAttribute, SetAttribute: SGRAttribute{Tag: tag}}
}

// sroConfigureCharsetAct is `.configure_charset, .{ .slot, .charset }`.
func sroConfigureCharsetAct(slot CharsetsSlots, charset CharsetsCharset) StreamAction {
	return StreamAction{
		Tag:              StreamActionKeyConfigureCharset,
		ConfigureCharset: StreamActionConfigureCharset{slot: slot, charset: charset},
	}
}

// sroInvokeCharsetAct is `.invoke_charset, .{ .bank, .charset, .locking }`.
func sroInvokeCharsetAct(bank CharsetsActiveSlot, charset CharsetsSlots, locking bool) StreamAction {
	return StreamAction{
		Tag:           StreamActionKeyInvokeCharset,
		InvokeCharset: StreamActionInvokeCharset{bank: bank, charset: charset, locking: locking},
	}
}

// sroSemanticPromptAct is `.semantic_prompt, cmd`.
func sroSemanticPromptAct(cmd OSCCommandSemanticPrompt) StreamAction {
	return StreamAction{Tag: StreamActionKeySemanticPrompt, SemanticPrompt: cmd}
}

// sroStartHyperlinkAct is `.start_hyperlink, .{ .uri = uri, .id = id }`.
func sroStartHyperlinkAct(uri string, id Optional[[]uint8]) StreamAction {
	return StreamAction{
		Tag:            StreamActionKeyStartHyperlink,
		StartHyperlink: StreamActionStartHyperlink{uri: []uint8(uri), id: id},
	}
}

// sroPrintString is `try t.printString(s)`.
func sroPrintString(t *testing.T, tm *Terminal, s string) {
	t.Helper()
	zigTry(t, tm.PrintString([]uint8(s)))
}

// sroExpectIgnored is term51DExpectIgnored: run a handler action that must
// not change anything observable.
func sroExpectIgnored(t *testing.T, tm *Terminal, h *StreamReadonlyHandler, a StreamAction) {
	t.Helper()
	before := sroSnapshot(t, tm)
	sroVt(t, h, a)
	after := sroSnapshot(t, tm)
	expectEqual(t, before, after)
}

// sroBusy is term51DBusy: a terminal with some state in every area the
// handler could touch, so that "nothing changed" assertions are meaningful.
func sroBusy(t *testing.T, tm *Terminal, h *StreamReadonlyHandler) {
	t.Helper()
	sroPrintString(t, tm, "hello\nworld")
	sroVt(t, h, sroModeAct(StreamActionKeySetMode, ModesModeCursorKeys))
	sroVt(t, h, sroModeAct(StreamActionKeySetMode, ModesModeMouseEventNormal))
	sroVt(t, h, sroModeAct(StreamActionKeySetMode, ModesModeMouseFormatSgr))
	sroVt(t, h, sroModeAct(StreamActionKeySetMode, ModesModeBracketedPaste))
	tm.colors.palette.Set(3, ColorRGB{r: 1, g: 2, b: 3})
	tm.colors.foreground.Set(ColorRGB{r: 4, g: 5, b: 6})
	tm.colors.cursor.Set(ColorRGB{r: 7, g: 8, b: 9})
	sroVt(t, h, sroKittyAct(StreamActionKeyKittyKeyboardPush, KittyKeyFlags{disambiguate: true}))
	sroVt(t, h, sroAct(StreamActionKeyTabSet))
	h.terminal.SetTopAndBottomMargin(2, 5)
	h.terminal.SetCursorPos(3, 4)
}

// sroAt is term51DAt.
func sroAt(tm *Terminal, x SizeCellCountInt, y SizeCellCountInt) {
	tm.screens.active.CursorAbsolute(x, y)
}

// sroExpectCursor is term51DExpectCursor.
func sroExpectCursor(t *testing.T, tm *Terminal, x uint, y uint) {
	t.Helper()
	expectEqual(t, x, uint(tm.screens.active.cursor.x))
	expectEqual(t, y, uint(tm.screens.active.cursor.y))
}

// sroExpectText is term51DExpectText.
func sroExpectText(t *testing.T, tm *Terminal, expected string) {
	t.Helper()
	str, err := tm.PlainString()
	zigTry(t, err)
	expectEqualStrings(t, expected, string(str))
}

// sroFill is term51DFill: fill a terminal of at least 10 columns and 5 rows
// with five distinct 10 character lines. The cursor ends pending wrap at the
// last cell.
func sroFill(t *testing.T, tm *Terminal) {
	t.Helper()
	sroPrintString(t, tm, "0123456789\nabcdefghij\nABCDEFGHIJ\nklmnopqrst\nKLMNOPQRST")
}

// sroScreen is `t.screens.get(key).?`.
func sroScreen(t *testing.T, tm *Terminal, key ScreenSetKey) *Screen {
	t.Helper()
	scr := tm.screens.Get(key)
	if scr == nil {
		t.Fatalf("unwrapped a null optional: screens.get(%d)", key)
	}
	return scr
}

// sroGetCell is `t.screens.active.pages.getCell(.{ .active = .{ .x = x, .y = y } }).?`.
func sroGetCell(t *testing.T, tm *Terminal, x SizeCellCountInt, y uint32) PageListCell {
	t.Helper()
	return zigUnwrap(t, tm.screens.active.pages.GetCell(Point{
		Tag:    PointTagActive,
		Active: PointCoordinate{x: x, y: y},
	}))
}

// sroDumpScreen is `t.screens.active.dumpStringAlloc(alloc, .{ .screen = .{} })`.
func sroDumpScreen(t *testing.T, tm *Terminal) string {
	t.Helper()
	str, err := tm.screens.active.DumpStringAlloc(Point{Tag: PointTagScreen})
	zigTry(t, err)
	return string(str)
}

// sroNew is term51DNew.
func sroNew(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) Terminal {
	t.Helper()
	tm, err := TerminalInit(terminalOptionsDefault(cols, rows))
	zigTry(t, err)
	return tm
}

// sroNewFilled is term51DNewFilled: a 10x5 terminal filled with the five lines
// of sroFill, cursor at the top left with no pending wrap.
func sroNewFilled(t *testing.T) Terminal {
	t.Helper()
	tm := sroNew(t, 10, 5)
	sroFill(t, &tm)
	sroAt(&tm, 0, 0)
	tm.screens.active.cursor.pending_wrap = false
	return tm
}

// sroFilled is term51D_filled.
const sroFilled = "0123456789\nabcdefghij\nABCDEFGHIJ\nklmnopqrst\nKLMNOPQRST"

// Zig: test "basic print"
func TestStreamReadonly_basic_print(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("Hello")))
	expectEqual(t, 5, tm.screens.active.cursor.x)
	expectEqual(t, 0, tm.screens.active.cursor.y)

	str, err := tm.PlainString()
	zigTry(t, err)
	// Zig-only: defer testing.allocator.free(str)
	expectEqualStrings(t, "Hello", string(str))
}

// Zig: test "cursor movement"
func TestStreamReadonly_cursor_movement(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Move cursor using escape sequences
	zigTry(t, s.NextSlice([]uint8("Hello\x1B[1;1H")))
	expectEqual(t, 0, tm.screens.active.cursor.x)
	expectEqual(t, 0, tm.screens.active.cursor.y)

	// Move to position 2,3
	zigTry(t, s.NextSlice([]uint8("\x1B[2;3H")))
	expectEqual(t, 2, tm.screens.active.cursor.x)
	expectEqual(t, 1, tm.screens.active.cursor.y)
}

// Zig: test "erase operations"
func TestStreamReadonly_erase_operations(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(20, 10))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Print some text
	zigTry(t, s.NextSlice([]uint8("Hello World")))
	expectEqual(t, 11, tm.screens.active.cursor.x)
	expectEqual(t, 0, tm.screens.active.cursor.y)

	// Move cursor to position 1,6 and erase from cursor to end of line
	zigTry(t, s.NextSlice([]uint8("\x1B[1;6H")))
	zigTry(t, s.NextSlice([]uint8("\x1B[K")))

	str, err := tm.PlainString()
	zigTry(t, err)
	// Zig-only: defer testing.allocator.free(str)
	expectEqualStrings(t, "Hello", string(str))
}

// Zig: test "tabs"
func TestStreamReadonly_tabs(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(80, 10))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("A\tB")))
	expectEqual(t, 9, tm.screens.active.cursor.x)

	str, err := tm.PlainString()
	zigTry(t, err)
	// Zig-only: defer testing.allocator.free(str)
	expectEqualStrings(t, "A       B", string(str))
}

// Zig: test "modes"
func TestStreamReadonly_modes(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(80, 24))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Test wraparound mode
	expect(t, tm.modes.Get(ModesModeWraparound), "t.modes.get(.wraparound)")
	zigTry(t, s.NextSlice([]uint8("\x1B[?7l"))) // Disable wraparound
	expect(t, !tm.modes.Get(ModesModeWraparound), "!t.modes.get(.wraparound)")
	zigTry(t, s.NextSlice([]uint8("\x1B[?7h"))) // Enable wraparound
	expect(t, tm.modes.Get(ModesModeWraparound), "t.modes.get(.wraparound)")
}

// Zig: test "scrolling regions"
func TestStreamReadonly_scrolling_regions(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(80, 24))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Set scrolling region from line 5 to 20
	zigTry(t, s.NextSlice([]uint8("\x1B[5;20r")))
	expectEqual(t, 4, tm.scrolling_region.top)
	expectEqual(t, 19, tm.scrolling_region.bottom)
	expectEqual(t, 0, tm.scrolling_region.left)
	expectEqual(t, 79, tm.scrolling_region.right)
}

// Zig: test "charsets"
func TestStreamReadonly_charsets(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(80, 24))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Configure G0 as DEC special graphics
	zigTry(t, s.NextSlice([]uint8("\x1B(0")))
	zigTry(t, s.NextSlice([]uint8("`"))) // Should print diamond character

	str, err := tm.PlainString()
	zigTry(t, err)
	// Zig-only: defer testing.allocator.free(str)
	expectEqualStrings(t, "◆", string(str))
}

// Zig: test "alt screen"
func TestStreamReadonly_alt_screen(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(10, 5))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Write to primary screen
	zigTry(t, s.NextSlice([]uint8("Primary")))
	expectEqual(t, ScreenSetKeyPrimary, tm.screens.active_key)

	// Switch to alt screen
	zigTry(t, s.NextSlice([]uint8("\x1B[?1049h")))
	expectEqual(t, ScreenSetKeyAlternate, tm.screens.active_key)

	// Write to alt screen
	zigTry(t, s.NextSlice([]uint8("Alt")))

	// Switch back to primary
	zigTry(t, s.NextSlice([]uint8("\x1B[?1049l")))
	expectEqual(t, ScreenSetKeyPrimary, tm.screens.active_key)

	str, err := tm.PlainString()
	zigTry(t, err)
	// Zig-only: defer testing.allocator.free(str)
	expectEqualStrings(t, "Primary", string(str))
}

// Zig: test "cursor save and restore"
func TestStreamReadonly_cursor_save_and_restore(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(80, 24))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Move cursor to 10,15
	zigTry(t, s.NextSlice([]uint8("\x1B[10;15H")))
	expectEqual(t, 14, tm.screens.active.cursor.x)
	expectEqual(t, 9, tm.screens.active.cursor.y)

	// Save cursor
	zigTry(t, s.NextSlice([]uint8("\x1B7")))

	// Move cursor elsewhere
	zigTry(t, s.NextSlice([]uint8("\x1B[1;1H")))
	expectEqual(t, 0, tm.screens.active.cursor.x)
	expectEqual(t, 0, tm.screens.active.cursor.y)

	// Restore cursor
	zigTry(t, s.NextSlice([]uint8("\x1B8")))
	expectEqual(t, 14, tm.screens.active.cursor.x)
	expectEqual(t, 9, tm.screens.active.cursor.y)
}

// Zig: test "attributes"
func TestStreamReadonly_attributes(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(80, 24))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Set bold and write text
	zigTry(t, s.NextSlice([]uint8("\x1B[1mBold\x1B[0m")))

	// Verify we can write attributes - just check the string was written
	str, err := tm.PlainString()
	zigTry(t, err)
	// Zig-only: defer testing.allocator.free(str)
	expectEqualStrings(t, "Bold", string(str))
}

// Zig: test "DECALN screen alignment"
func TestStreamReadonly_DECALN_screen_alignment(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(10, 3))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Run DECALN
	zigTry(t, s.NextSlice([]uint8("\x1B#8")))

	// Verify entire screen is filled with 'E'
	str, err := tm.PlainString()
	zigTry(t, err)
	// Zig-only: defer testing.allocator.free(str)
	expectEqualStrings(t, "EEEEEEEEEE\nEEEEEEEEEE\nEEEEEEEEEE", string(str))

	// Cursor should be at 1,1
	expectEqual(t, 0, tm.screens.active.cursor.x)
	expectEqual(t, 0, tm.screens.active.cursor.y)
}

// Zig: test "full reset"
func TestStreamReadonly_full_reset(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(80, 24))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Make some changes
	zigTry(t, s.NextSlice([]uint8("Hello")))
	zigTry(t, s.NextSlice([]uint8("\x1B[10;20H")))
	zigTry(t, s.NextSlice([]uint8("\x1B[5;20r"))) // Set scroll region
	zigTry(t, s.NextSlice([]uint8("\x1B[?7l")))   // Disable wraparound

	// Full reset
	zigTry(t, s.NextSlice([]uint8("\x1Bc")))

	// Verify reset state
	expectEqual(t, 0, tm.screens.active.cursor.x)
	expectEqual(t, 0, tm.screens.active.cursor.y)
	expectEqual(t, 0, tm.scrolling_region.top)
	expectEqual(t, 23, tm.scrolling_region.bottom)
	expect(t, tm.modes.Get(ModesModeWraparound), "t.modes.get(.wraparound)")
}

// Zig: test "ignores query actions"
func TestStreamReadonly_ignores_query_actions(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(80, 24))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// These should be ignored without error
	zigTry(t, s.NextSlice([]uint8("\x1B[c")))  // Device attributes
	zigTry(t, s.NextSlice([]uint8("\x1B[5n"))) // Device status report
	zigTry(t, s.NextSlice([]uint8("\x1B[6n"))) // Cursor position report

	// Terminal should still be functional
	zigTry(t, s.NextSlice([]uint8("Test")))
	str, err := tm.PlainString()
	zigTry(t, err)
	// Zig-only: defer testing.allocator.free(str)
	expectEqualStrings(t, "Test", string(str))
}

// Zig: test "OSC 4 set and reset palette"
func TestStreamReadonly_OSC_4_set_and_reset_palette(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Save default color
	default_color_0 := tm.colors.palette.original[0]

	// Set color 0 to red
	zigTry(t, s.NextSlice([]uint8("\x1b]4;0;rgb:ff/00/00\x1b\\")))
	expectEqual(t, uint8(0xff), tm.colors.palette.current[0].r)
	expectEqual(t, uint8(0x00), tm.colors.palette.current[0].g)
	expectEqual(t, uint8(0x00), tm.colors.palette.current[0].b)
	expect(t, tm.colors.palette.mask.isSet(0), "t.colors.palette.mask.isSet(0)")

	// Reset color 0
	zigTry(t, s.NextSlice([]uint8("\x1b]104;0\x1b\\")))
	expectEqual(t, default_color_0, tm.colors.palette.current[0])
	expect(t, !tm.colors.palette.mask.isSet(0), "!t.colors.palette.mask.isSet(0)")
}

// Zig: test "OSC 104 reset all palette colors"
func TestStreamReadonly_OSC_104_reset_all_palette_colors(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Set multiple colors
	zigTry(t, s.NextSlice([]uint8("\x1b]4;0;rgb:ff/00/00\x1b\\")))
	zigTry(t, s.NextSlice([]uint8("\x1b]4;1;rgb:00/ff/00\x1b\\")))
	zigTry(t, s.NextSlice([]uint8("\x1b]4;2;rgb:00/00/ff\x1b\\")))
	expect(t, tm.colors.palette.mask.isSet(0), "t.colors.palette.mask.isSet(0)")
	expect(t, tm.colors.palette.mask.isSet(1), "t.colors.palette.mask.isSet(1)")
	expect(t, tm.colors.palette.mask.isSet(2), "t.colors.palette.mask.isSet(2)")

	// Reset all palette colors
	zigTry(t, s.NextSlice([]uint8("\x1b]104\x1b\\")))
	expectEqual(t, tm.colors.palette.original[0], tm.colors.palette.current[0])
	expectEqual(t, tm.colors.palette.original[1], tm.colors.palette.current[1])
	expectEqual(t, tm.colors.palette.original[2], tm.colors.palette.current[2])
	expect(t, !tm.colors.palette.mask.isSet(0), "!t.colors.palette.mask.isSet(0)")
	expect(t, !tm.colors.palette.mask.isSet(1), "!t.colors.palette.mask.isSet(1)")
	expect(t, !tm.colors.palette.mask.isSet(2), "!t.colors.palette.mask.isSet(2)")
}

// Zig: test "OSC 10 set and reset foreground color"
func TestStreamReadonly_OSC_10_set_and_reset_foreground_color(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Initially unset
	expect(t, !tm.colors.foreground.Get().Valid, "t.colors.foreground.get() == null")

	// Set foreground to red
	zigTry(t, s.NextSlice([]uint8("\x1b]10;rgb:ff/00/00\x1b\\")))
	fg := zigUnwrap(t, tm.colors.foreground.Get())
	expectEqual(t, uint8(0xff), fg.r)
	expectEqual(t, uint8(0x00), fg.g)
	expectEqual(t, uint8(0x00), fg.b)

	// Reset foreground
	zigTry(t, s.NextSlice([]uint8("\x1b]110\x1b\\")))
	expect(t, !tm.colors.foreground.Get().Valid, "t.colors.foreground.get() == null")
}

// Zig: test "OSC 11 set and reset background color"
func TestStreamReadonly_OSC_11_set_and_reset_background_color(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Set background to green
	zigTry(t, s.NextSlice([]uint8("\x1b]11;rgb:00/ff/00\x1b\\")))
	bg := zigUnwrap(t, tm.colors.background.Get())
	expectEqual(t, uint8(0x00), bg.r)
	expectEqual(t, uint8(0xff), bg.g)
	expectEqual(t, uint8(0x00), bg.b)

	// Reset background
	zigTry(t, s.NextSlice([]uint8("\x1b]111\x1b\\")))
	expect(t, !tm.colors.background.Get().Valid, "t.colors.background.get() == null")
}

// Zig: test "OSC 12 set and reset cursor color"
func TestStreamReadonly_OSC_12_set_and_reset_cursor_color(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Set cursor to blue
	zigTry(t, s.NextSlice([]uint8("\x1b]12;rgb:00/00/ff\x1b\\")))
	cursor := zigUnwrap(t, tm.colors.cursor.Get())
	expectEqual(t, uint8(0x00), cursor.r)
	expectEqual(t, uint8(0x00), cursor.g)
	expectEqual(t, uint8(0xff), cursor.b)

	// Reset cursor
	zigTry(t, s.NextSlice([]uint8("\x1b]112\x1b\\")))
	// After reset, cursor might be null (using default)
}

// Zig: test "kitty color protocol set palette"
func TestStreamReadonly_kitty_color_protocol_set_palette(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Set palette color 5 to magenta using kitty protocol
	zigTry(t, s.NextSlice([]uint8("\x1b]21;5=rgb:ff/00/ff\x1b\\")))
	expectEqual(t, uint8(0xff), tm.colors.palette.current[5].r)
	expectEqual(t, uint8(0x00), tm.colors.palette.current[5].g)
	expectEqual(t, uint8(0xff), tm.colors.palette.current[5].b)
	expect(t, tm.colors.palette.mask.isSet(5), "t.colors.palette.mask.isSet(5)")
	expect(t, tm.flags.dirty.palette, "t.flags.dirty.palette")
}

// Zig: test "kitty color protocol reset palette"
func TestStreamReadonly_kitty_color_protocol_reset_palette(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Set and then reset palette color
	original := tm.colors.palette.original[7]
	zigTry(t, s.NextSlice([]uint8("\x1b]21;7=rgb:aa/bb/cc\x1b\\")))
	expect(t, tm.colors.palette.mask.isSet(7), "t.colors.palette.mask.isSet(7)")

	zigTry(t, s.NextSlice([]uint8("\x1b]21;7=\x1b\\")))
	expectEqual(t, original, tm.colors.palette.current[7])
	expect(t, !tm.colors.palette.mask.isSet(7), "!t.colors.palette.mask.isSet(7)")
}

// Zig: test "kitty color protocol set foreground"
func TestStreamReadonly_kitty_color_protocol_set_foreground(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Set foreground using kitty protocol
	zigTry(t, s.NextSlice([]uint8("\x1b]21;foreground=rgb:12/34/56\x1b\\")))
	fg := zigUnwrap(t, tm.colors.foreground.Get())
	expectEqual(t, uint8(0x12), fg.r)
	expectEqual(t, uint8(0x34), fg.g)
	expectEqual(t, uint8(0x56), fg.b)
}

// Zig: test "kitty color protocol set background"
func TestStreamReadonly_kitty_color_protocol_set_background(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Set background using kitty protocol
	zigTry(t, s.NextSlice([]uint8("\x1b]21;background=rgb:78/9a/bc\x1b\\")))
	bg := zigUnwrap(t, tm.colors.background.Get())
	expectEqual(t, uint8(0x78), bg.r)
	expectEqual(t, uint8(0x9a), bg.g)
	expectEqual(t, uint8(0xbc), bg.b)
}

// Zig: test "kitty color protocol set cursor"
func TestStreamReadonly_kitty_color_protocol_set_cursor(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Set cursor using kitty protocol
	zigTry(t, s.NextSlice([]uint8("\x1b]21;cursor=rgb:de/f0/12\x1b\\")))
	cursor := zigUnwrap(t, tm.colors.cursor.Get())
	expectEqual(t, uint8(0xde), cursor.r)
	expectEqual(t, uint8(0xf0), cursor.g)
	expectEqual(t, uint8(0x12), cursor.b)
}

// Zig: test "kitty color protocol reset foreground"
func TestStreamReadonly_kitty_color_protocol_reset_foreground(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Set and reset foreground
	zigTry(t, s.NextSlice([]uint8("\x1b]21;foreground=rgb:11/22/33\x1b\\")))
	expect(t, tm.colors.foreground.Get().Valid, "t.colors.foreground.get() != null")

	zigTry(t, s.NextSlice([]uint8("\x1b]21;foreground=\x1b\\")))
	// After reset, should be unset
	expect(t, !tm.colors.foreground.Get().Valid, "t.colors.foreground.get() == null")
}

// Zig: test "palette dirty flag set on color change"
func TestStreamReadonly_palette_dirty_flag_set_on_color_change(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Clear dirty flag
	tm.flags.dirty.palette = false

	// Setting palette color should set dirty flag
	zigTry(t, s.NextSlice([]uint8("\x1b]4;0;rgb:ff/00/00\x1b\\")))
	expect(t, tm.flags.dirty.palette, "t.flags.dirty.palette")

	// Clear and test reset
	tm.flags.dirty.palette = false
	zigTry(t, s.NextSlice([]uint8("\x1b]104;0\x1b\\")))
	expect(t, tm.flags.dirty.palette, "t.flags.dirty.palette")

	// Clear and test kitty protocol
	tm.flags.dirty.palette = false
	zigTry(t, s.NextSlice([]uint8("\x1b]21;1=rgb:00/ff/00\x1b\\")))
	expect(t, tm.flags.dirty.palette, "t.flags.dirty.palette")
}

// Zig: test "semantic prompt fresh line"
func TestStreamReadonly_semantic_prompt_fresh_line(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("Hello")))
	zigTry(t, s.NextSlice([]uint8("\x1b]133;L\x07")))
	expectEqual(t, 0, tm.screens.active.cursor.x)
	expectEqual(t, 1, tm.screens.active.cursor.y)
}

// Zig: test "semantic prompt fresh line new prompt"
func TestStreamReadonly_semantic_prompt_fresh_line_new_prompt(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Write some text and then send OSC 133;A (fresh_line_new_prompt)
	zigTry(t, s.NextSlice([]uint8("Hello")))
	zigTry(t, s.NextSlice([]uint8("\x1b]133;A\x07")))

	// Should do a fresh line (carriage return + index)
	expectEqual(t, 0, tm.screens.active.cursor.x)
	expectEqual(t, 1, tm.screens.active.cursor.y)

	// Should set cursor semantic_content to prompt
	expectEqual(t, PageCellSemanticContentPrompt, tm.screens.active.cursor.semantic_content)

	// Test with redraw option
	zigTry(t, s.NextSlice([]uint8("prompt$ ")))
	zigTry(t, s.NextSlice([]uint8("\x1b]133;A;redraw=1\x07")))
	expect(t, tm.flags.shell_redraws_prompt == OSCParsersSemanticPromptRedrawTrue, "t.flags.shell_redraws_prompt == .true")
}

// Zig: test "semantic prompt end of input, then start output"
func TestStreamReadonly_semantic_prompt_end_of_input_then_start_output(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Write some text and then send OSC 133;A (fresh_line_new_prompt)
	zigTry(t, s.NextSlice([]uint8("Hello")))
	zigTry(t, s.NextSlice([]uint8("\x1b]133;A\x07")))
	zigTry(t, s.NextSlice([]uint8("prompt$ ")))
	zigTry(t, s.NextSlice([]uint8("\x1b]133;B\x07")))
	expectEqual(t, PageCellSemanticContentInput, tm.screens.active.cursor.semantic_content)
	zigTry(t, s.NextSlice([]uint8("\x1b]133;C\x07")))
	expectEqual(t, PageCellSemanticContentOutput, tm.screens.active.cursor.semantic_content)
}

// Zig: test "semantic prompt prompt_start"
func TestStreamReadonly_semantic_prompt_prompt_start(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Write some text
	zigTry(t, s.NextSlice([]uint8("Hello")))

	// OSC 133;P marks the start of a prompt (without fresh line behavior)
	zigTry(t, s.NextSlice([]uint8("\x1b]133;P\x07")))
	expectEqual(t, PageCellSemanticContentPrompt, tm.screens.active.cursor.semantic_content)
	expectEqual(t, 5, tm.screens.active.cursor.x)
	expectEqual(t, 0, tm.screens.active.cursor.y)
}

// Zig: test "semantic prompt new_command"
func TestStreamReadonly_semantic_prompt_new_command(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Write some text
	zigTry(t, s.NextSlice([]uint8("Hello")))
	zigTry(t, s.NextSlice([]uint8("\x1b]133;N\x07")))

	// Should behave like fresh_line_new_prompt - cursor moves to column 0
	// on next line since we had content
	expectEqual(t, 0, tm.screens.active.cursor.x)
	expectEqual(t, 1, tm.screens.active.cursor.y)
	expectEqual(t, PageCellSemanticContentPrompt, tm.screens.active.cursor.semantic_content)
}

// Zig: test "semantic prompt new_command at column zero"
func TestStreamReadonly_semantic_prompt_new_command_at_column_zero(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// OSC 133;N when already at column 0 should stay on same line
	zigTry(t, s.NextSlice([]uint8("\x1b]133;N\x07")))
	expectEqual(t, 0, tm.screens.active.cursor.x)
	expectEqual(t, 0, tm.screens.active.cursor.y)
	expectEqual(t, PageCellSemanticContentPrompt, tm.screens.active.cursor.semantic_content)
}

// Zig: test "semantic prompt end_prompt_start_input_terminate_eol clears on linefeed"
func TestStreamReadonly_semantic_prompt_end_prompt_start_input_terminate_eol_clears_on_linefeed(t *testing.T) {
	testStdCapacity(t)
	tm, err := TerminalInit(terminalOptionsDefault(10, 10))
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Set input terminated by EOL
	zigTry(t, s.NextSlice([]uint8("\x1b]133;I\x07")))
	expectEqual(t, PageCellSemanticContentInput, tm.screens.active.cursor.semantic_content)

	// Linefeed should reset semantic content to output
	zigTry(t, s.NextSlice([]uint8("\n")))
	expectEqual(t, PageCellSemanticContentOutput, tm.screens.active.cursor.semantic_content)
}

// Zig: test "stream: CSI W with intermediate but no params"
func TestStreamReadonly_stream_CSI_W_with_intermediate_but_no_params(t *testing.T) {
	testStdCapacity(t)
	// Regression test from AFL++ crash. CSI ? W without
	// parameters caused an out-of-bounds access on input.params[0].
	opts := terminalOptionsDefault(80, 24)
	opts.max_scrollback = 100
	tm, err := TerminalInit(opts)
	zigTry(t, err)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("\x1b[?W")))
}

// ---------------------------------------------------------------------------
// term51: additional tests (D). Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

// Zig: test "term51: Handler.init: stores the terminal pointer and nothing else"
func TestStreamReadonly_term51_Handler_init_stores_the_terminal_pointer_and_nothing_else(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	expect(t, h.terminal == &tm, "h.terminal == &t")
	expectEqual(t, unsafe.Sizeof(&tm), unsafe.Sizeof(h))

	// Initializing a handler does not touch the terminal.
	fresh := sroNew(t, 10, 5)
	defer fresh.Deinit()
	expectEqual(t, sroSnapshot(t, &fresh), sroSnapshot(t, &tm))

	// Two handlers over the same terminal are independent values that
	// both point at it.
	h2 := StreamReadonlyHandlerInit(&tm)
	expect(t, h2.terminal == h.terminal, "h2.terminal == h.terminal")

	// The handler returned by Terminal.vtHandler is the same thing.
	h3 := tm.VtHandler()
	expect(t, h3.terminal == &tm, "h3.terminal == &t")
}

// Zig: test "term51: Handler.deinit: does nothing to the terminal and the handler stays usable"
func TestStreamReadonly_term51_Handler_deinit_does_nothing_to_the_terminal_and_the_handler_stays_usable(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()

	h := StreamReadonlyHandlerInit(&tm)
	sroVt(t, &h, sroPrintAct('A'))
	before := sroSnapshot(t, &tm)
	h.Deinit()
	expect(t, h.terminal == &tm, "h.terminal == &t")
	expectEqual(t, before, sroSnapshot(t, &tm))

	// Still usable after deinit.
	sroVt(t, &h, sroPrintAct('B'))
	sroExpectText(t, &tm, "AB")

	// And deinit again is harmless.
	h.Deinit()
	sroExpectText(t, &tm, "AB")
}

// Zig: test "term51: Handler.vt: DCS and APC actions are ignored"
func TestStreamReadonly_term51_Handler_vt_DCS_and_APC_actions_are_ignored(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 6)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	sroBusy(t, &tm, &h)

	sroExpectIgnored(t, &tm, &h, StreamAction{
		Tag:     StreamActionKeyDcsHook,
		DcsHook: ParserActionDCS{Final: 'q'},
	})
	sroExpectIgnored(t, &tm, &h, StreamAction{
		Tag: StreamActionKeyDcsHook,
		DcsHook: ParserActionDCS{
			Intermediates: []uint8("+"),
			Params:        []uint16{1, 2},
			Final:         'p',
		},
	})
	sroExpectIgnored(t, &tm, &h, StreamAction{Tag: StreamActionKeyDcsPut, DcsPut: 'a'})
	sroExpectIgnored(t, &tm, &h, StreamAction{Tag: StreamActionKeyDcsPut, DcsPut: 0x1b})
	sroExpectIgnored(t, &tm, &h, sroAct(StreamActionKeyDcsUnhook))
	sroExpectIgnored(t, &tm, &h, sroAct(StreamActionKeyApcStart))
	sroExpectIgnored(t, &tm, &h, StreamAction{Tag: StreamActionKeyApcPut, ApcPut: 'G'})
	sroExpectIgnored(t, &tm, &h, StreamAction{Tag: StreamActionKeyApcPut, ApcPut: 0})
	sroExpectIgnored(t, &tm, &h, sroAct(StreamActionKeyApcEnd))
}

// Zig: test "term51: Handler.vt: actions that need a reply are ignored"
func TestStreamReadonly_term51_Handler_vt_actions_that_need_a_reply_are_ignored(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 6)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	sroBusy(t, &tm, &h)

	sroExpectIgnored(t, &tm, &h, sroAct(StreamActionKeyEnquiry))
	sroExpectIgnored(t, &tm, &h, sroModeAct(StreamActionKeyRequestMode, ModesModeOrigin))
	sroExpectIgnored(t, &tm, &h, sroModeAct(StreamActionKeyRequestMode, ModesModeInsert))
	sroExpectIgnored(t, &tm, &h, StreamAction{
		Tag:                StreamActionKeyRequestModeUnknown,
		RequestModeUnknown: StreamActionRawMode{mode: 9999, ansi: false},
	})
	sroExpectIgnored(t, &tm, &h, StreamAction{
		Tag:                StreamActionKeyRequestModeUnknown,
		RequestModeUnknown: StreamActionRawMode{mode: 77, ansi: true},
	})
	sroExpectIgnored(t, &tm, &h, StreamAction{Tag: StreamActionKeySizeReport, SizeReport: CSISizeReportStyleCsi14T})
	sroExpectIgnored(t, &tm, &h, StreamAction{Tag: StreamActionKeySizeReport, SizeReport: CSISizeReportStyleCsi16T})
	sroExpectIgnored(t, &tm, &h, StreamAction{Tag: StreamActionKeySizeReport, SizeReport: CSISizeReportStyleCsi18T})
	sroExpectIgnored(t, &tm, &h, StreamAction{Tag: StreamActionKeySizeReport, SizeReport: CSISizeReportStyleCsi21T})
	sroExpectIgnored(t, &tm, &h, sroAct(StreamActionKeyXtversion))
	sroExpectIgnored(t, &tm, &h, StreamAction{Tag: StreamActionKeyDeviceAttributes, DeviceAttributes: ANSIDeviceAttributeReqPrimary})
	sroExpectIgnored(t, &tm, &h, StreamAction{Tag: StreamActionKeyDeviceAttributes, DeviceAttributes: ANSIDeviceAttributeReqSecondary})
	sroExpectIgnored(t, &tm, &h, StreamAction{Tag: StreamActionKeyDeviceAttributes, DeviceAttributes: ANSIDeviceAttributeReqTertiary})
	sroExpectIgnored(t, &tm, &h, StreamAction{
		Tag:          StreamActionKeyDeviceStatus,
		DeviceStatus: StreamActionDeviceStatus{request: DeviceStatusRequestOperatingStatus},
	})
	sroExpectIgnored(t, &tm, &h, StreamAction{
		Tag:          StreamActionKeyDeviceStatus,
		DeviceStatus: StreamActionDeviceStatus{request: DeviceStatusRequestCursorPosition},
	})
	sroExpectIgnored(t, &tm, &h, StreamAction{
		Tag:          StreamActionKeyDeviceStatus,
		DeviceStatus: StreamActionDeviceStatus{request: DeviceStatusRequestColorScheme},
	})
	sroExpectIgnored(t, &tm, &h, sroAct(StreamActionKeyKittyKeyboardQuery))
}

// Zig: test "term51: Handler.vt: UI and notification actions are ignored"
func TestStreamReadonly_term51_Handler_vt_UI_and_notification_actions_are_ignored(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 6)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	sroBusy(t, &tm, &h)

	sroExpectIgnored(t, &tm, &h, sroAct(StreamActionKeyBell))
	sroExpectIgnored(t, &tm, &h, StreamAction{
		Tag:         StreamActionKeyWindowTitle,
		WindowTitle: StreamActionWindowTitle{title: []uint8("a title")},
	})
	sroExpectIgnored(t, &tm, &h, StreamAction{
		Tag:         StreamActionKeyWindowTitle,
		WindowTitle: StreamActionWindowTitle{title: []uint8("")},
	})
	// The pwd reported by OSC 7 is not stored by the readonly handler.
	sroExpectIgnored(t, &tm, &h, StreamAction{
		Tag:       StreamActionKeyReportPwd,
		ReportPwd: StreamActionReportPwd{url: []uint8("file:///tmp")},
	})
	expectEqual(t, 0, len(tm.pwd))
	sroExpectIgnored(t, &tm, &h, StreamAction{
		Tag:                     StreamActionKeyShowDesktopNotification,
		ShowDesktopNotification: StreamActionShowDesktopNotification{title: []uint8("t"), body: []uint8("b")},
	})
	sroExpectIgnored(t, &tm, &h, StreamAction{
		Tag: StreamActionKeyProgressReport,
		ProgressReport: OSCCommandProgressReport{
			state:    OSCCommandProgressReportStateSet,
			progress: Some(uint8(50)),
		},
	})
	sroExpectIgnored(t, &tm, &h, StreamAction{
		Tag:            StreamActionKeyProgressReport,
		ProgressReport: OSCCommandProgressReport{state: OSCCommandProgressReportStateRemove},
	})
	sroExpectIgnored(t, &tm, &h, StreamAction{
		Tag:               StreamActionKeyClipboardContents,
		ClipboardContents: StreamActionClipboardContents{kind: 'c', data: []uint8("Zm9v")},
	})
	sroExpectIgnored(t, &tm, &h, sroU16Act(StreamActionKeyTitlePush, 0))
	sroExpectIgnored(t, &tm, &h, sroU16Act(StreamActionKeyTitlePush, 65535))
	sroExpectIgnored(t, &tm, &h, sroU16Act(StreamActionKeyTitlePop, 0))
	sroExpectIgnored(t, &tm, &h, sroU16Act(StreamActionKeyTitlePop, 65535))
}

// Zig: test "term51: Handler.vt: ignored sequences through a stream leave the terminal alone"
func TestStreamReadonly_term51_Handler_vt_ignored_sequences_through_a_stream_leave_the_terminal_alone(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 6)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()
	sroBusy(t, &tm, s.handler)

	before := sroSnapshot(t, &tm)
	zigTry(t, s.NextSlice([]uint8("\x07")))                                     // bell
	zigTry(t, s.NextSlice([]uint8("\x05")))                                     // enquiry
	zigTry(t, s.NextSlice([]uint8("\x1b[c\x1b[>c\x1b[=c")))                     // device attributes
	zigTry(t, s.NextSlice([]uint8("\x1b[5n\x1b[6n\x1b[?996n")))                 // device status
	zigTry(t, s.NextSlice([]uint8("\x1b[>q")))                                  // xtversion
	zigTry(t, s.NextSlice([]uint8("\x1b[14t\x1b[16t\x1b[18t\x1b[21t")))         // size reports
	zigTry(t, s.NextSlice([]uint8("\x1b[?u")))                                  // kitty keyboard query
	zigTry(t, s.NextSlice([]uint8("\x1b[?6$p\x1b[4$p\x1b[?9999$p")))            // DECRQM
	zigTry(t, s.NextSlice([]uint8("\x1b]0;a title\x07\x1b]7;file:///tmp\x07"))) // title, pwd
	zigTry(t, s.NextSlice([]uint8("\x1bP+q544e\x1b\\")))                        // DCS XTGETTCAP
	zigTry(t, s.NextSlice([]uint8("\x1b_Gi=1;AAAA\x1b\\")))                     // APC
	expectEqual(t, before, sroSnapshot(t, &tm))
}

// sroExpectTabTable is term51DExpectTabTable: run the private
// horizontalTab/horizontalTabBack from every column of the first row and
// compare the resulting column with the expected table.
func sroExpectTabTable(
	t *testing.T,
	tm *Terminal,
	h *StreamReadonlyHandler,
	back bool,
	count uint16,
	expected []uint8,
) {
	t.Helper()
	expectEqual(t, uint(tm.cols), uint(len(expected)))
	for x, want := range expected {
		sroAt(tm, SizeCellCountInt(x), 1)
		tm.screens.active.cursor.pending_wrap = false
		if back {
			h.horizontalTabBack(count)
		} else {
			h.horizontalTab(count)
		}
		if uint(tm.screens.active.cursor.x) != uint(want) {
			t.Fatalf("back=%v count=%d start x=%d: got %d, want %d", back, count, x, tm.screens.active.cursor.x, want)
		}
		// The row never changes.
		expectEqual(t, 1, tm.screens.active.cursor.y)
	}
}

// Zig: test "term51: Handler.horizontalTab: default tabstops, counts 0 1 2 3 65535 from every column"
func TestStreamReadonly_term51_Handler_horizontalTab_default_tabstops_counts_0_1_2_3_65535_from_every_column(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	// Default tabstops are at 8 and 16, the right margin is column 19.
	sroExpectTabTable(t, &tm, &h, false, 0, []uint8{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19})
	sroExpectTabTable(t, &tm, &h, false, 1, []uint8{8, 8, 8, 8, 8, 8, 8, 8, 16, 16, 16, 16, 16, 16, 16, 16, 19, 19, 19, 19})
	sroExpectTabTable(t, &tm, &h, false, 2, []uint8{16, 16, 16, 16, 16, 16, 16, 16, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19})
	sroExpectTabTable(t, &tm, &h, false, 3, []uint8{19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19})
	sroExpectTabTable(t, &tm, &h, false, 65535, []uint8{19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19})
}

// Zig: test "term51: Handler.horizontalTabBack: default tabstops, counts 0 1 2 3 65535 from every column"
func TestStreamReadonly_term51_Handler_horizontalTabBack_default_tabstops_counts_0_1_2_3_65535_from_every_column(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroExpectTabTable(t, &tm, &h, true, 0, []uint8{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19})
	sroExpectTabTable(t, &tm, &h, true, 1, []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 8, 8, 8, 8, 8, 8, 8, 8, 16, 16, 16})
	sroExpectTabTable(t, &tm, &h, true, 2, []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 8, 8, 8})
	sroExpectTabTable(t, &tm, &h, true, 3, []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	sroExpectTabTable(t, &tm, &h, true, 65535, []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
}

// Zig: test "term51: Handler.horizontalTab: custom tabstops and no tabstops"
func TestStreamReadonly_term51_Handler_horizontalTab_custom_tabstops_and_no_tabstops(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	// No tabstops at all: forward goes to the last column, back to the first.
	sroVt(t, &h, sroAct(StreamActionKeyTabClearAll))
	sroExpectTabTable(t, &tm, &h, false, 1, []uint8{19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19})
	sroExpectTabTable(t, &tm, &h, true, 1, []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})

	// Stops at 3 and 10 only.
	sroAt(&tm, 3, 0)
	sroVt(t, &h, sroAct(StreamActionKeyTabSet))
	sroAt(&tm, 10, 0)
	sroVt(t, &h, sroAct(StreamActionKeyTabSet))
	sroExpectTabTable(t, &tm, &h, false, 1, []uint8{3, 3, 3, 10, 10, 10, 10, 10, 10, 10, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19})
	sroExpectTabTable(t, &tm, &h, false, 2, []uint8{10, 10, 10, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19})
	sroExpectTabTable(t, &tm, &h, false, 65535, []uint8{19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19})
	sroExpectTabTable(t, &tm, &h, true, 1, []uint8{0, 0, 0, 0, 3, 3, 3, 3, 3, 3, 3, 10, 10, 10, 10, 10, 10, 10, 10, 10})
	sroExpectTabTable(t, &tm, &h, true, 2, []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 3, 3, 3, 3, 3, 3, 3, 3})
	sroExpectTabTable(t, &tm, &h, true, 65535, []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
}

// Zig: test "term51: Handler.horizontalTab: stops at the right margin and does not move beyond it"
func TestStreamReadonly_term51_Handler_horizontalTab_stops_at_the_right_margin_and_does_not_move_beyond_it(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	// Left margin column 4, right margin column 14 (0-indexed).
	zigTry(t, h.setMode(ModesModeEnableLeftAndRightMargin, true))
	sroVt(t, &h, sroMarginAct(StreamActionKeyLeftAndRightMargin, 5, 15))
	expectEqual(t, 4, tm.scrolling_region.left)
	expectEqual(t, 14, tm.scrolling_region.right)

	// The stop at 16 is beyond the margin, so 14 wins. From the margin or
	// beyond it the cursor does not move at all.
	sroExpectTabTable(t, &tm, &h, false, 0, []uint8{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19})
	sroExpectTabTable(t, &tm, &h, false, 1, []uint8{8, 8, 8, 8, 8, 8, 8, 8, 14, 14, 14, 14, 14, 14, 14, 15, 16, 17, 18, 19})
	sroExpectTabTable(t, &tm, &h, false, 2, []uint8{14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 15, 16, 17, 18, 19})
	sroExpectTabTable(t, &tm, &h, false, 65535, []uint8{14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 14, 15, 16, 17, 18, 19})
}

// Zig: test "term51: Handler.horizontalTabBack: left limit is column 0 without origin mode, the left margin with it"
func TestStreamReadonly_term51_Handler_horizontalTabBack_left_limit_is_column_0_without_origin_mode_the_left_margin_with_it(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	zigTry(t, h.setMode(ModesModeEnableLeftAndRightMargin, true))
	sroVt(t, &h, sroMarginAct(StreamActionKeyLeftAndRightMargin, 5, 15))

	// Origin mode off: margins are ignored going backwards.
	sroExpectTabTable(t, &tm, &h, true, 1, []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 8, 8, 8, 8, 8, 8, 8, 8, 16, 16, 16})
	sroExpectTabTable(t, &tm, &h, true, 65535, []uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})

	// Origin mode on (set directly so that the cursor is not homed): the
	// cursor stops at the left margin (column 4) and never moves when it
	// is at or left of it.
	tm.modes.Set(ModesModeOrigin, true)
	sroExpectTabTable(t, &tm, &h, true, 0, []uint8{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19})
	sroExpectTabTable(t, &tm, &h, true, 1, []uint8{0, 1, 2, 3, 4, 4, 4, 4, 4, 8, 8, 8, 8, 8, 8, 8, 8, 16, 16, 16})
	sroExpectTabTable(t, &tm, &h, true, 2, []uint8{0, 1, 2, 3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 8, 8, 8})
	sroExpectTabTable(t, &tm, &h, true, 65535, []uint8{0, 1, 2, 3, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4})
}

// Zig: test "term51: Handler.horizontalTab: tabstops beyond the preallocated 512 columns"
func TestStreamReadonly_term51_Handler_horizontalTab_tabstops_beyond_the_preallocated_512_columns(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 600, 2)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	// Default stops are every 8 columns up to (excluding) the last column.
	sroAt(&tm, 508, 0)
	h.horizontalTab(1)
	sroExpectCursor(t, &tm, 512, 0)
	h.horizontalTab(1)
	sroExpectCursor(t, &tm, 520, 0)
	sroAt(&tm, 585, 0)
	h.horizontalTab(1)
	sroExpectCursor(t, &tm, 592, 0)
	h.horizontalTab(1)
	sroExpectCursor(t, &tm, 599, 0)
	h.horizontalTab(1)
	sroExpectCursor(t, &tm, 599, 0)

	h.horizontalTabBack(1)
	sroExpectCursor(t, &tm, 592, 0)
	h.horizontalTabBack(2)
	sroExpectCursor(t, &tm, 576, 0)
	sroAt(&tm, 520, 0)
	h.horizontalTabBack(1)
	sroExpectCursor(t, &tm, 512, 0)
	h.horizontalTabBack(1)
	sroExpectCursor(t, &tm, 504, 0)

	sroAt(&tm, 0, 0)
	h.horizontalTab(65535)
	sroExpectCursor(t, &tm, 599, 0)
	h.horizontalTabBack(65535)
	sroExpectCursor(t, &tm, 0, 0)

	// Setting a stop in the dynamic region works as well.
	sroAt(&tm, 530, 0)
	sroVt(t, &h, sroAct(StreamActionKeyTabSet))
	sroAt(&tm, 521, 0)
	h.horizontalTab(1)
	sroExpectCursor(t, &tm, 528, 0)
	h.horizontalTab(1)
	sroExpectCursor(t, &tm, 530, 0)
	h.horizontalTab(1)
	sroExpectCursor(t, &tm, 536, 0)
}

// Zig: test "term51: Handler.horizontalTab: pending wrap is not cleared, and nothing is erased or printed"
func TestStreamReadonly_term51_Handler_horizontalTab_pending_wrap_is_not_cleared_and_nothing_is_erased_or_printed(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroPrintString(t, &tm, "0123456789")
	expect(t, tm.screens.active.cursor.pending_wrap, "t.screens.active.cursor.pending_wrap")
	sroExpectCursor(t, &tm, 9, 0)

	// At the right margin the loop does not move, so the flag remains.
	h.horizontalTab(1)
	sroExpectCursor(t, &tm, 9, 0)
	expect(t, tm.screens.active.cursor.pending_wrap, "t.screens.active.cursor.pending_wrap")
	h.horizontalTab(65535)
	sroExpectCursor(t, &tm, 9, 0)
	expect(t, tm.screens.active.cursor.pending_wrap, "t.screens.active.cursor.pending_wrap")
	sroExpectText(t, &tm, "0123456789")

	// Going backwards from a pending wrap state moves the cursor and
	// leaves the pending wrap flag set (upstream behaviour).
	h.horizontalTabBack(1)
	sroExpectCursor(t, &tm, 8, 0)
	expect(t, tm.screens.active.cursor.pending_wrap, "t.screens.active.cursor.pending_wrap")
	h.horizontalTabBack(1)
	sroExpectCursor(t, &tm, 0, 0)
	expect(t, tm.screens.active.cursor.pending_wrap, "t.screens.active.cursor.pending_wrap")
	sroExpectText(t, &tm, "0123456789")

	// A forward tab with the flag set and the cursor away from the margin
	// also keeps it.
	h.horizontalTab(1)
	sroExpectCursor(t, &tm, 8, 0)
	expect(t, tm.screens.active.cursor.pending_wrap, "t.screens.active.cursor.pending_wrap")
}

// Zig: test "term51: Handler.vt: horizontal_tab and horizontal_tab_back pass their count"
func TestStreamReadonly_term51_Handler_vt_horizontal_tab_and_horizontal_tab_back_pass_their_count(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 40, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroVt(t, &h, sroU16Act(StreamActionKeyHorizontalTab, 0))
	sroExpectCursor(t, &tm, 0, 0)
	sroVt(t, &h, sroU16Act(StreamActionKeyHorizontalTab, 1))
	sroExpectCursor(t, &tm, 8, 0)
	sroVt(t, &h, sroU16Act(StreamActionKeyHorizontalTab, 3))
	sroExpectCursor(t, &tm, 32, 0)
	sroVt(t, &h, sroU16Act(StreamActionKeyHorizontalTab, 65535))
	sroExpectCursor(t, &tm, 39, 0)
	sroVt(t, &h, sroU16Act(StreamActionKeyHorizontalTabBack, 0))
	sroExpectCursor(t, &tm, 39, 0)
	sroVt(t, &h, sroU16Act(StreamActionKeyHorizontalTabBack, 1))
	sroExpectCursor(t, &tm, 32, 0)
	sroVt(t, &h, sroU16Act(StreamActionKeyHorizontalTabBack, 2))
	sroExpectCursor(t, &tm, 16, 0)
	sroVt(t, &h, sroU16Act(StreamActionKeyHorizontalTabBack, 65535))
	sroExpectCursor(t, &tm, 0, 0)
}

// Zig: test "term51: Handler.vt: tabs through a stream (HT, CHT, CBT)"
func TestStreamReadonly_term51_Handler_vt_tabs_through_a_stream_HT_CHT_CBT(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 40, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("\t")))
	sroExpectCursor(t, &tm, 8, 0)
	zigTry(t, s.NextSlice([]uint8("\x1b[3I")))
	sroExpectCursor(t, &tm, 32, 0)
	zigTry(t, s.NextSlice([]uint8("\x1b[I")))
	sroExpectCursor(t, &tm, 39, 0)
	zigTry(t, s.NextSlice([]uint8("\x1b[Z")))
	sroExpectCursor(t, &tm, 32, 0)
	zigTry(t, s.NextSlice([]uint8("\x1b[2Z")))
	sroExpectCursor(t, &tm, 16, 0)
	// An explicit zero count is passed through and moves nothing (xterm
	// would treat it as 1).
	zigTry(t, s.NextSlice([]uint8("\x1b[0Z")))
	sroExpectCursor(t, &tm, 16, 0)
	zigTry(t, s.NextSlice([]uint8("\x1b[0I")))
	sroExpectCursor(t, &tm, 16, 0)
	zigTry(t, s.NextSlice([]uint8("\x1b[65535Z")))
	sroExpectCursor(t, &tm, 0, 0)
	zigTry(t, s.NextSlice([]uint8("\x1b[65535I")))
	sroExpectCursor(t, &tm, 39, 0)
}

// Zig: test "term51: Handler.vt: tab_set, tab_clear_current, tab_clear_all and tab_reset"
func TestStreamReadonly_term51_Handler_vt_tab_set_tab_clear_current_tab_clear_all_and_tab_reset(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	// Defaults.
	for i := uint(0); i < 20; i++ {
		expectEqual(t, i == 8 || i == 16, tm.tabstops.Get(i))
	}

	// tab_set at the cursor, idempotent, does not move the cursor.
	sroAt(&tm, 3, 2)
	sroVt(t, &h, sroAct(StreamActionKeyTabSet))
	expect(t, tm.tabstops.Get(3), "t.tabstops.get(3)")
	sroVt(t, &h, sroAct(StreamActionKeyTabSet))
	expect(t, tm.tabstops.Get(3), "t.tabstops.get(3)")
	sroExpectCursor(t, &tm, 3, 2)

	// tab_clear_current clears the stop under the cursor and only that one.
	sroVt(t, &h, sroAct(StreamActionKeyTabClearCurrent))
	expect(t, !tm.tabstops.Get(3), "!t.tabstops.get(3)")
	expect(t, tm.tabstops.Get(8), "t.tabstops.get(8)")
	expect(t, tm.tabstops.Get(16), "t.tabstops.get(16)")
	sroAt(&tm, 8, 2)
	sroVt(t, &h, sroAct(StreamActionKeyTabClearCurrent))
	for i := uint(0); i < 20; i++ {
		expectEqual(t, i == 16, tm.tabstops.Get(i))
	}
	sroExpectCursor(t, &tm, 8, 2)

	// tab_clear_all clears everything, regardless of the cursor.
	sroVt(t, &h, sroAct(StreamActionKeyTabSet))
	sroVt(t, &h, sroAct(StreamActionKeyTabClearAll))
	for i := uint(0); i < 20; i++ {
		expect(t, !tm.tabstops.Get(i), "!t.tabstops.get(i)")
	}
	sroVt(t, &h, sroAct(StreamActionKeyTabClearAll))
	for i := uint(0); i < 20; i++ {
		expect(t, !tm.tabstops.Get(i), "!t.tabstops.get(i)")
	}
	sroExpectCursor(t, &tm, 8, 2)

	// tab_reset restores the default stops every 8 columns, and drops
	// custom ones.
	sroVt(t, &h, sroAct(StreamActionKeyTabSet))
	sroAt(&tm, 5, 0)
	sroVt(t, &h, sroAct(StreamActionKeyTabSet))
	sroVt(t, &h, sroAct(StreamActionKeyTabReset))
	for i := uint(0); i < 20; i++ {
		expectEqual(t, i == 8 || i == 16, tm.tabstops.Get(i))
	}
	sroExpectCursor(t, &tm, 5, 0)

	// Resetting again changes nothing.
	sroVt(t, &h, sroAct(StreamActionKeyTabReset))
	for i := uint(0); i < 20; i++ {
		expectEqual(t, i == 8 || i == 16, tm.tabstops.Get(i))
	}
}

// Zig: test "term51: Handler.vt: tab_clear_current on a column without a stop sets one (upstream quirk)"
func TestStreamReadonly_term51_Handler_vt_tab_clear_current_on_a_column_without_a_stop_sets_one_upstream_quirk(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	// Tabstops.unset toggles the bit with an xor, so clearing a column
	// that has no stop creates one.
	sroAt(&tm, 3, 0)
	expect(t, !tm.tabstops.Get(3), "!t.tabstops.get(3)")
	sroVt(t, &h, sroAct(StreamActionKeyTabClearCurrent))
	expect(t, tm.tabstops.Get(3), "t.tabstops.get(3)")
	sroVt(t, &h, sroAct(StreamActionKeyTabClearCurrent))
	expect(t, !tm.tabstops.Get(3), "!t.tabstops.get(3)")

	// The cell is observable through horizontal tabs.
	sroVt(t, &h, sroAct(StreamActionKeyTabClearCurrent))
	sroAt(&tm, 0, 0)
	sroVt(t, &h, sroU16Act(StreamActionKeyHorizontalTab, 1))
	sroExpectCursor(t, &tm, 3, 0)
}

// Zig: test "term51: Handler.vt: tab stops are per terminal width and reset by a width change"
func TestStreamReadonly_term51_Handler_vt_tab_stops_are_per_terminal_width_and_reset_by_a_width_change(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroVt(t, &h, sroAct(StreamActionKeyTabClearAll))
	sroAt(&tm, 5, 0)
	sroVt(t, &h, sroAct(StreamActionKeyTabSet))

	// DECCOLM (with mode 40) resizes the terminal which rebuilds the
	// tabstops from the defaults.
	zigTry(t, h.setMode(ModesModeEnableMode3, true))
	zigTry(t, h.setMode(ModesMode132Column, true))
	expectEqual(t, 132, tm.cols)
	expect(t, !tm.tabstops.Get(5), "!t.tabstops.get(5)")
	expect(t, tm.tabstops.Get(8), "t.tabstops.get(8)")
	expect(t, tm.tabstops.Get(128), "t.tabstops.get(128)")
	expect(t, !tm.tabstops.Get(131), "!t.tabstops.get(131)")
	expect(t, !tm.tabstops.Get(130), "!t.tabstops.get(130)")
}

// sroCharsetDec is term51DCharsetDec.
const sroCharsetDec = CharsetsCharsetDecSpecial

// sroPlainModes is term51D_plain_modes: the modes that Handler.setMode only
// records (it falls in the `else` prong, or in a prong that explicitly does
// nothing).
var sroPlainModes = []ModesMode{
	ModesModeDisableKeyboard,
	ModesModeInsert,
	ModesModeSendReceiveMode,
	ModesModeLinefeed,
	ModesModeCursorKeys,
	ModesModeSlowScroll,
	ModesModeReverseColors,
	ModesModeAutorepeat,
	ModesModeWraparound,
	ModesModeCursorBlinking,
	ModesModeCursorVisible,
	ModesModeEnableMode3,
	ModesModeReverseWrap,
	ModesModeKeypadKeys,
	ModesModeFocusEvent,
	ModesModeMouseAlternateScroll,
	ModesModeIgnoreKeypadWithNumlock,
	ModesModeAltEscPrefix,
	ModesModeAltSendsEscape,
	ModesModeReverseWrapExtended,
	ModesModeBracketedPaste,
	ModesModeSynchronizedOutput,
	ModesModeGraphemeCluster,
	ModesModeReportColorScheme,
	ModesModeInBandSizeReports,
}

// sroSpecialModes is term51D_special_modes: the modes with a prong of their
// own in Handler.setMode that does more than record the bit.
var sroSpecialModes = []ModesMode{
	ModesModeOrigin,
	ModesModeEnableLeftAndRightMargin,
	ModesModeAltScreenLegacy,
	ModesModeAltScreen,
	ModesModeAltScreenSaveCursorClearEnter,
	ModesModeSaveCursor,
	ModesMode132Column,
	ModesModeMouseEventX10,
	ModesModeMouseEventNormal,
	ModesModeMouseEventButton,
	ModesModeMouseEventAny,
	ModesModeMouseFormatUtf8,
	ModesModeMouseFormatSgr,
	ModesModeMouseFormatUrxvt,
	ModesModeMouseFormatSgrPixels,
}

// Zig: test "term51: Handler.setMode: every mode of modes.zig is covered by the plain or special lists"
func TestStreamReadonly_term51_Handler_setMode_every_mode_of_modes_zig_is_covered_by_the_plain_or_special_lists(t *testing.T) {
	testStdCapacity(t)
	fields := ModesModeValues()
	expectEqual(t, 40, len(fields))
	expectEqual(t, len(fields), len(sroPlainModes)+len(sroSpecialModes))
	for _, f := range fields {
		n := 0
		for _, m := range sroPlainModes {
			if m.String() == f.String() {
				n++
			}
		}
		for _, m := range sroSpecialModes {
			if m.String() == f.String() {
				n++
			}
		}
		expectEqual(t, 1, n)
	}
}

// Zig: test "term51: Handler.setMode: plain modes only change their own bit"
func TestStreamReadonly_term51_Handler_setMode_plain_modes_only_change_their_own_bit(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 6)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	sroBusy(t, &tm, &h)

	for _, mode := range sroPlainModes {
		for _, enabled := range []bool{true, false, true, true, false, false} {
			before := sroSnapshot(t, &tm)
			zigTry(t, h.setMode(mode, enabled))
			expectEqual(t, enabled, tm.modes.Get(mode))
			expected := before
			expected.modes = sroModesWith(before.modes, mode, enabled)
			expectEqual(t, expected, sroSnapshot(t, &tm))
		}
	}
}

// Zig: test "term51: Handler.vt: set_mode and reset_mode on plain modes only change their own bit"
func TestStreamReadonly_term51_Handler_vt_set_mode_and_reset_mode_on_plain_modes_only_change_their_own_bit(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 6)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	sroBusy(t, &tm, &h)

	for _, mode := range sroPlainModes {
		before := sroSnapshot(t, &tm)
		sroVt(t, &h, sroModeAct(StreamActionKeySetMode, mode))
		expect(t, tm.modes.Get(mode), "t.modes.get(mode)")
		expected := before
		expected.modes = sroModesWith(before.modes, mode, true)
		expectEqual(t, expected, sroSnapshot(t, &tm))

		sroVt(t, &h, sroModeAct(StreamActionKeyResetMode, mode))
		expect(t, !tm.modes.Get(mode), "!t.modes.get(mode)")
		expected.modes = sroModesWith(before.modes, mode, false)
		expectEqual(t, expected, sroSnapshot(t, &tm))
	}
}

// Zig: test "term51: Handler.setMode: defaults of the plain modes"
func TestStreamReadonly_term51_Handler_setMode_defaults_of_the_plain_modes(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()

	// Only these start out enabled.
	for _, mode := range sroPlainModes {
		want := false
		switch mode {
		case ModesModeSendReceiveMode,
			ModesModeWraparound,
			ModesModeCursorVisible,
			ModesModeMouseAlternateScroll,
			ModesModeIgnoreKeypadWithNumlock,
			ModesModeAltEscPrefix:
			want = true
		}
		expectEqual(t, want, tm.modes.Get(mode))
	}
	for _, mode := range sroSpecialModes {
		expect(t, !tm.modes.Get(mode), "!t.modes.get(mode)")
	}
}

// Zig: test "term51: Handler.setMode: reverse_colors does not mark the terminal dirty"
func TestStreamReadonly_term51_Handler_setMode_reverse_colors_does_not_mark_the_terminal_dirty(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	zigTry(t, h.setMode(ModesModeReverseColors, true))
	expect(t, tm.modes.Get(ModesModeReverseColors), "t.modes.get(.reverse_colors)")
	expect(t, !tm.flags.dirty.reverse_colors, "!t.flags.dirty.reverse_colors")
	expect(t, !tm.flags.dirty.clear, "!t.flags.dirty.clear")
	expect(t, !tm.flags.dirty.palette, "!t.flags.dirty.palette")
	zigTry(t, h.setMode(ModesModeReverseColors, false))
	expect(t, !tm.modes.Get(ModesModeReverseColors), "!t.modes.get(.reverse_colors)")
	expect(t, !tm.flags.dirty.reverse_colors, "!t.flags.dirty.reverse_colors")
}

// Zig: test "term51: Handler.setMode: origin homes the cursor relative to the margins when set, absolute when reset"
func TestStreamReadonly_term51_Handler_setMode_origin_homes_the_cursor_relative_to_the_margins_when_set_absolute_when_reset(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 10)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	// No margins: both directions home to (0,0), pending wrap is cleared.
	sroPrintString(t, &tm, "0123456789")
	sroAt(&tm, 19, 7)
	tm.screens.active.cursor.pending_wrap = true
	zigTry(t, h.setMode(ModesModeOrigin, true))
	expect(t, tm.modes.Get(ModesModeOrigin), "t.modes.get(.origin)")
	sroExpectCursor(t, &tm, 0, 0)
	expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")

	// Top/bottom margins rows 3..8 (1-indexed).
	tm.modes.Set(ModesModeOrigin, false)
	tm.SetTopAndBottomMargin(3, 8)
	expectEqual(t, 2, tm.scrolling_region.top)
	expectEqual(t, 7, tm.scrolling_region.bottom)
	sroAt(&tm, 7, 5)
	tm.screens.active.cursor.pending_wrap = true
	zigTry(t, h.setMode(ModesModeOrigin, true))
	sroExpectCursor(t, &tm, 0, 2)
	expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")

	// Left/right margins as well: home is the margin corner.
	zigTry(t, h.setMode(ModesModeEnableLeftAndRightMargin, true))
	tm.SetLeftAndRightMargin(6, 15)
	sroExpectCursor(t, &tm, 5, 2)
	sroAt(&tm, 12, 6)
	zigTry(t, h.setMode(ModesModeOrigin, true))
	sroExpectCursor(t, &tm, 5, 2)

	// Resetting goes to the absolute top-left, ignoring the margins.
	sroAt(&tm, 12, 6)
	tm.screens.active.cursor.pending_wrap = true
	zigTry(t, h.setMode(ModesModeOrigin, false))
	expect(t, !tm.modes.Get(ModesModeOrigin), "!t.modes.get(.origin)")
	sroExpectCursor(t, &tm, 0, 0)
	expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")

	// The margins and the text are untouched.
	expectEqual(t, 2, tm.scrolling_region.top)
	expectEqual(t, 7, tm.scrolling_region.bottom)
	expectEqual(t, 5, tm.scrolling_region.left)
	expectEqual(t, 14, tm.scrolling_region.right)
	sroExpectText(t, &tm, "0123456789")
}

// Zig: test "term51: Handler.setMode: enable_left_and_right_margin reset restores full width margins"
func TestStreamReadonly_term51_Handler_setMode_enable_left_and_right_margin_reset_restores_full_width_margins(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 10)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	// Resetting when already default is harmless.
	zigTry(t, h.setMode(ModesModeEnableLeftAndRightMargin, false))
	expectEqual(t, 0, tm.scrolling_region.left)
	expectEqual(t, 19, tm.scrolling_region.right)

	// Setting does not touch the margins.
	tm.SetTopAndBottomMargin(3, 8)
	zigTry(t, h.setMode(ModesModeEnableLeftAndRightMargin, true))
	expect(t, tm.modes.Get(ModesModeEnableLeftAndRightMargin), "t.modes.get(.enable_left_and_right_margin)")
	expectEqual(t, 0, tm.scrolling_region.left)
	expectEqual(t, 19, tm.scrolling_region.right)
	tm.SetLeftAndRightMargin(5, 15)
	expectEqual(t, 4, tm.scrolling_region.left)
	expectEqual(t, 14, tm.scrolling_region.right)
	zigTry(t, h.setMode(ModesModeEnableLeftAndRightMargin, true))
	expectEqual(t, 4, tm.scrolling_region.left)
	expectEqual(t, 14, tm.scrolling_region.right)

	// Resetting restores left/right only; top/bottom and the cursor stay.
	sroAt(&tm, 10, 5)
	zigTry(t, h.setMode(ModesModeEnableLeftAndRightMargin, false))
	expect(t, !tm.modes.Get(ModesModeEnableLeftAndRightMargin), "!t.modes.get(.enable_left_and_right_margin)")
	expectEqual(t, 0, tm.scrolling_region.left)
	expectEqual(t, 19, tm.scrolling_region.right)
	expectEqual(t, 2, tm.scrolling_region.top)
	expectEqual(t, 7, tm.scrolling_region.bottom)
	sroExpectCursor(t, &tm, 10, 5)

	// With the mode off DECSLRM is ignored.
	before := sroSnapshot(t, &tm)
	sroVt(t, &h, sroMarginAct(StreamActionKeyLeftAndRightMargin, 3, 10))
	expectEqual(t, before, sroSnapshot(t, &tm))
}

// Zig: test "term51: Handler.setMode: alt_screen_legacy (47) switches screens and copies the cursor, no clearing"
func TestStreamReadonly_term51_Handler_setMode_alt_screen_legacy_47_switches_screens_and_copies_the_cursor_no_clearing(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroPrintString(t, &tm, "Primary")
	sroVt(t, &h, sroStartHyperlinkAct("http://a", None[[]uint8]()))
	expect(t, tm.screens.active.cursor.hyperlink_id != 0, "t.screens.active.cursor.hyperlink_id != 0")
	sroVt(t, &h, sroConfigureCharsetAct(CharsetsSlotsG0, CharsetsCharsetDecSpecial))
	sroExpectCursor(t, &tm, 7, 0)
	tm.flags.dirty.clear = false

	zigTry(t, h.setMode(ModesModeAltScreenLegacy, true))
	expect(t, tm.modes.Get(ModesModeAltScreenLegacy), "t.modes.get(.alt_screen_legacy)")
	expectEqual(t, ScreenSetKeyAlternate, tm.screens.active_key)
	expect(t, tm.flags.dirty.clear, "t.flags.dirty.clear")
	// Cursor position copied, alt screen starts blank, hyperlink ended on
	// the primary, charset state carried over.
	sroExpectCursor(t, &tm, 7, 0)
	sroExpectText(t, &tm, "")
	expectEqual(t, 0, sroScreen(t, &tm, ScreenSetKeyPrimary).cursor.hyperlink_id)
	expectEqual(t, 0, tm.screens.active.cursor.hyperlink_id)
	expectEqual(t, sroCharsetDec, tm.screens.active.charset.charsets.Get(CharsetsSlotsG0))

	// Print on the alt screen, then enable again: nothing happens.
	zigTry(t, tm.Print('X'))
	sroExpectCursor(t, &tm, 8, 0)
	before := sroSnapshot(t, &tm)
	zigTry(t, h.setMode(ModesModeAltScreenLegacy, true))
	expectEqual(t, before, sroSnapshot(t, &tm))
	sroExpectText(t, &tm, "       X")

	// Leaving copies the cursor back and keeps both screens' content.
	zigTry(t, h.setMode(ModesModeAltScreenLegacy, false))
	expect(t, !tm.modes.Get(ModesModeAltScreenLegacy), "!t.modes.get(.alt_screen_legacy)")
	expectEqual(t, ScreenSetKeyPrimary, tm.screens.active_key)
	sroExpectCursor(t, &tm, 8, 0)
	sroExpectText(t, &tm, "Primary")

	// Entering again does not clear the alt screen (47 never clears).
	zigTry(t, h.setMode(ModesModeAltScreenLegacy, true))
	expectEqual(t, ScreenSetKeyAlternate, tm.screens.active_key)
	sroExpectText(t, &tm, "       X")
	sroExpectCursor(t, &tm, 8, 0)

	// Leaving while on the primary screen does nothing to the screens.
	zigTry(t, h.setMode(ModesModeAltScreenLegacy, false))
	before_primary := sroSnapshot(t, &tm)
	zigTry(t, h.setMode(ModesModeAltScreenLegacy, false))
	expectEqual(t, before_primary, sroSnapshot(t, &tm))
}

// Zig: test "term51: Handler.setMode: alt_screen (1047) clears the alt screen only when leaving it"
func TestStreamReadonly_term51_Handler_setMode_alt_screen_1047_clears_the_alt_screen_only_when_leaving_it(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroPrintString(t, &tm, "Primary")
	zigTry(t, h.setMode(ModesModeAltScreen, true))
	expect(t, tm.modes.Get(ModesModeAltScreen), "t.modes.get(.alt_screen)")
	expectEqual(t, ScreenSetKeyAlternate, tm.screens.active_key)
	sroExpectCursor(t, &tm, 7, 0)
	sroExpectText(t, &tm, "")

	sroPrintString(t, &tm, "Al")
	sroExpectText(t, &tm, "       Al")

	// Entering again while on the alt screen does not clear it.
	zigTry(t, h.setMode(ModesModeAltScreen, true))
	expectEqual(t, ScreenSetKeyAlternate, tm.screens.active_key)
	sroExpectText(t, &tm, "       Al")

	// Leaving clears the alt screen, switches and copies the cursor.
	zigTry(t, h.setMode(ModesModeAltScreen, false))
	expect(t, !tm.modes.Get(ModesModeAltScreen), "!t.modes.get(.alt_screen)")
	expectEqual(t, ScreenSetKeyPrimary, tm.screens.active_key)
	sroExpectText(t, &tm, "Primary")
	expect(t, tm.screens.Get(ScreenSetKeyAlternate) != nil, "t.screens.get(.alternate) != null")

	// The alt screen really was cleared.
	zigTry(t, h.setMode(ModesModeAltScreenLegacy, true))
	sroExpectText(t, &tm, "")
	sroPrintString(t, &tm, "Z")
	zigTry(t, h.setMode(ModesModeAltScreenLegacy, false))
	sroExpectText(t, &tm, "Primary")

	// Resetting 1047 while on the primary screen does not erase anything.
	zigTry(t, h.setMode(ModesModeAltScreenLegacy, true))
	sroExpectText(t, &tm, "         Z")
	zigTry(t, h.setMode(ModesModeAltScreenLegacy, false))
	zigTry(t, h.setMode(ModesModeAltScreen, false))
	sroExpectText(t, &tm, "Primary")
	zigTry(t, h.setMode(ModesModeAltScreenLegacy, true))
	sroExpectText(t, &tm, "         Z")
}

// Zig: test "term51: Handler.setMode: alt_screen_save_cursor_clear_enter (1049) saves, clears on entry and restores on exit"
func TestStreamReadonly_term51_Handler_setMode_alt_screen_save_cursor_clear_enter_1049_saves_clears_on_entry_and_restores_on_exit(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	// Leave something behind on the alt screen to see it cleared on entry.
	zigTry(t, h.setMode(ModesModeAltScreenLegacy, true))
	sroPrintString(t, &tm, "stale")
	zigTry(t, h.setMode(ModesModeAltScreenLegacy, false))

	sroAt(&tm, 0, 0)
	sroPrintString(t, &tm, "Primary")
	sroAt(&tm, 4, 3)
	expect(t, !tm.screens.active.saved_cursor.Valid, "t.screens.active.saved_cursor == null")

	zigTry(t, h.setMode(ModesModeAltScreenSaveCursorClearEnter, true))
	expect(t, tm.modes.Get(ModesModeAltScreenSaveCursorClearEnter), "t.modes.get(.alt_screen_save_cursor_clear_enter)")
	expectEqual(t, ScreenSetKeyAlternate, tm.screens.active_key)
	// The cursor of the primary screen was saved, the alt screen is blank
	// and the cursor was copied over.
	saved := zigUnwrap(t, sroScreen(t, &tm, ScreenSetKeyPrimary).saved_cursor)
	expectEqual(t, 4, saved.x)
	expectEqual(t, 3, saved.y)
	sroExpectText(t, &tm, "")
	sroExpectCursor(t, &tm, 4, 3)

	// Entering again while on the alt screen saves the alt cursor on the
	// alt screen, clears it again and stays.
	sroPrintString(t, &tm, "again")
	sroAt(&tm, 2, 1)
	zigTry(t, h.setMode(ModesModeAltScreenSaveCursorClearEnter, true))
	expectEqual(t, ScreenSetKeyAlternate, tm.screens.active_key)
	sroExpectText(t, &tm, "")
	sroExpectCursor(t, &tm, 2, 1)
	alt_saved := zigUnwrap(t, tm.screens.active.saved_cursor)
	expectEqual(t, 2, alt_saved.x)
	expectEqual(t, 1, alt_saved.y)
	saved_again := zigUnwrap(t, sroScreen(t, &tm, ScreenSetKeyPrimary).saved_cursor)
	expectEqual(t, 4, saved_again.x)
	expectEqual(t, 3, saved_again.y)

	// Leaving restores the primary cursor and does not clear the alt screen.
	sroPrintString(t, &tm, "alt")
	zigTry(t, h.setMode(ModesModeAltScreenSaveCursorClearEnter, false))
	expect(t, !tm.modes.Get(ModesModeAltScreenSaveCursorClearEnter), "!t.modes.get(.alt_screen_save_cursor_clear_enter)")
	expectEqual(t, ScreenSetKeyPrimary, tm.screens.active_key)
	sroExpectCursor(t, &tm, 4, 3)
	sroExpectText(t, &tm, "Primary")
	zigTry(t, h.setMode(ModesModeAltScreenLegacy, true))
	sroExpectText(t, &tm, "\n  alt")
}

// Zig: test "term51: Handler.setMode: alt_screen_save_cursor_clear_enter reset on the primary screen restores a default cursor"
func TestStreamReadonly_term51_Handler_setMode_alt_screen_save_cursor_clear_enter_reset_on_the_primary_screen_restores_a_default_cursor(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroPrintString(t, &tm, "Primary")
	sroAt(&tm, 5, 3)
	tm.screens.active.cursor.pending_wrap = true
	// Nothing was saved, so the restore goes to the defaults.
	zigTry(t, h.setMode(ModesModeAltScreenSaveCursorClearEnter, false))
	expectEqual(t, ScreenSetKeyPrimary, tm.screens.active_key)
	sroExpectCursor(t, &tm, 0, 0)
	expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
	sroExpectText(t, &tm, "Primary")
	expect(t, tm.screens.Get(ScreenSetKeyAlternate) == nil, "t.screens.get(.alternate) == null")
}

// Zig: test "term51: Handler.setMode: save_cursor (1048) saves on set and restores on reset"
func TestStreamReadonly_term51_Handler_setMode_save_cursor_1048_saves_on_set_and_restores_on_reset(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 10)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	// Reset without a previous save restores the defaults.
	sroAt(&tm, 6, 6)
	sroVt(t, &h, sroSGRAct(SGRAttributeTagBold))
	tm.modes.Set(ModesModeOrigin, true)
	zigTry(t, h.setMode(ModesModeSaveCursor, false))
	expect(t, !tm.modes.Get(ModesModeSaveCursor), "!t.modes.get(.save_cursor)")
	sroExpectCursor(t, &tm, 0, 0)
	expect(t, !tm.screens.active.cursor.style.flags.bold, "!t.screens.active.cursor.style.flags.bold")
	expect(t, !tm.modes.Get(ModesModeOrigin), "!t.modes.get(.origin)")

	// Set saves, and does not move or change anything else.
	sroAt(&tm, 4, 2)
	sroVt(t, &h, sroSGRAct(SGRAttributeTagBold))
	tm.modes.Set(ModesModeOrigin, true)
	zigTry(t, h.setMode(ModesModeSaveCursor, true))
	expect(t, tm.modes.Get(ModesModeSaveCursor), "t.modes.get(.save_cursor)")
	sroExpectCursor(t, &tm, 4, 2)
	expect(t, tm.screens.active.cursor.style.flags.bold, "t.screens.active.cursor.style.flags.bold")
	saved := zigUnwrap(t, tm.screens.active.saved_cursor)
	expectEqual(t, 4, saved.x)
	expectEqual(t, 2, saved.y)
	expect(t, saved.origin, "saved.origin")
	expect(t, saved.style.flags.bold, "saved.style.flags.bold")

	// Move away, clear the attribute and origin mode, then restore.
	tm.modes.Set(ModesModeOrigin, false)
	sroAt(&tm, 0, 0)
	sroVt(t, &h, sroSGRAct(SGRAttributeTagResetBold))
	expect(t, !tm.screens.active.cursor.style.flags.bold, "!t.screens.active.cursor.style.flags.bold")
	zigTry(t, h.setMode(ModesModeSaveCursor, false))
	expect(t, !tm.modes.Get(ModesModeSaveCursor), "!t.modes.get(.save_cursor)")
	sroExpectCursor(t, &tm, 4, 2)
	expect(t, tm.screens.active.cursor.style.flags.bold, "t.screens.active.cursor.style.flags.bold")
	expect(t, tm.modes.Get(ModesModeOrigin), "t.modes.get(.origin)")

	// The saved state is kept after a restore.
	sroAt(&tm, 9, 9)
	zigTry(t, h.setMode(ModesModeSaveCursor, false))
	sroExpectCursor(t, &tm, 4, 2)

	// A second save overwrites the first.
	tm.modes.Set(ModesModeOrigin, false)
	sroAt(&tm, 1, 1)
	zigTry(t, h.setMode(ModesModeSaveCursor, true))
	sroAt(&tm, 2, 2)
	zigTry(t, h.setMode(ModesModeSaveCursor, true))
	sroAt(&tm, 3, 3)
	zigTry(t, h.setMode(ModesModeSaveCursor, false))
	sroExpectCursor(t, &tm, 2, 2)
}

// Zig: test "term51: Handler.setMode: save_cursor is per screen"
func TestStreamReadonly_term51_Handler_setMode_save_cursor_is_per_screen(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 10)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroAt(&tm, 4, 2)
	zigTry(t, h.setMode(ModesModeSaveCursor, true))
	zigTry(t, h.setMode(ModesModeAltScreenLegacy, true))
	// The alt screen has nothing saved.
	sroAt(&tm, 7, 7)
	zigTry(t, h.setMode(ModesModeSaveCursor, false))
	sroExpectCursor(t, &tm, 0, 0)
	zigTry(t, h.setMode(ModesModeAltScreenLegacy, false))
	zigTry(t, h.setMode(ModesModeSaveCursor, false))
	sroExpectCursor(t, &tm, 4, 2)
}

// Zig: test "term51: Handler.setMode: 132_column without mode 40 is ignored and the bit stays clear"
func TestStreamReadonly_term51_Handler_setMode_132_column_without_mode_40_is_ignored_and_the_bit_stays_clear(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroPrintString(t, &tm, "Hello")
	sroAt(&tm, 3, 2)
	before := sroSnapshot(t, &tm)

	zigTry(t, h.setMode(ModesMode132Column, true))
	expect(t, !tm.modes.Get(ModesMode132Column), "!t.modes.get(.@\"132_column\")")
	expectEqual(t, before, sroSnapshot(t, &tm))
	expectEqual(t, 20, tm.cols)

	zigTry(t, h.setMode(ModesMode132Column, false))
	expect(t, !tm.modes.Get(ModesMode132Column), "!t.modes.get(.@\"132_column\")")
	expectEqual(t, before, sroSnapshot(t, &tm))
	sroExpectText(t, &tm, "Hello")
	sroExpectCursor(t, &tm, 3, 2)
}

// Zig: test "term51: Handler.setMode: 132_column with mode 40 resizes to 132 or 80 columns, clears and homes"
func TestStreamReadonly_term51_Handler_setMode_132_column_with_mode_40_resizes_to_132_or_80_columns_clears_and_homes(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	// Enabling mode 40 alone does nothing visible.
	zigTry(t, h.setMode(ModesModeEnableMode3, true))
	expectEqual(t, 20, tm.cols)

	zigTry(t, h.setMode(ModesModeEnableLeftAndRightMargin, true))
	tm.SetTopAndBottomMargin(2, 4)
	tm.SetLeftAndRightMargin(3, 10)
	sroPrintString(t, &tm, "hello")
	sroAt(&tm, 6, 3)
	tm.flags.dirty.clear = false

	zigTry(t, h.setMode(ModesMode132Column, true))
	expect(t, tm.modes.Get(ModesMode132Column), "t.modes.get(.@\"132_column\")")
	expectEqual(t, 132, tm.cols)
	expectEqual(t, 5, tm.rows)
	expectEqual(t, 132, tm.screens.active.pages.cols)
	sroExpectText(t, &tm, "")
	sroExpectCursor(t, &tm, 0, 0)
	expect(t, tm.flags.dirty.clear, "t.flags.dirty.clear")
	// Margins are reset by the resize but the mode stays on.
	expectEqual(t, 0, tm.scrolling_region.top)
	expectEqual(t, 4, tm.scrolling_region.bottom)
	expectEqual(t, 0, tm.scrolling_region.left)
	expectEqual(t, 131, tm.scrolling_region.right)
	expect(t, tm.modes.Get(ModesModeEnableLeftAndRightMargin), "t.modes.get(.enable_left_and_right_margin)")
	// Tabstops follow the new width.
	expect(t, tm.tabstops.Get(8), "t.tabstops.get(8)")
	expect(t, tm.tabstops.Get(128), "t.tabstops.get(128)")
	expect(t, !tm.tabstops.Get(129), "!t.tabstops.get(129)")

	// Setting it again clears and homes again even though the size is
	// unchanged.
	sroPrintString(t, &tm, "abc")
	zigTry(t, h.setMode(ModesMode132Column, true))
	expect(t, tm.modes.Get(ModesMode132Column), "t.modes.get(.@\"132_column\")")
	expectEqual(t, 132, tm.cols)
	sroExpectText(t, &tm, "")
	sroExpectCursor(t, &tm, 0, 0)

	// Reset goes to 80 columns.
	sroPrintString(t, &tm, "abc")
	zigTry(t, h.setMode(ModesMode132Column, false))
	expect(t, !tm.modes.Get(ModesMode132Column), "!t.modes.get(.@\"132_column\")")
	expectEqual(t, 80, tm.cols)
	expectEqual(t, 80, tm.screens.active.pages.cols)
	expectEqual(t, 79, tm.scrolling_region.right)
	sroExpectText(t, &tm, "")
	sroExpectCursor(t, &tm, 0, 0)

	// Reset at 80 columns still clears and homes.
	sroPrintString(t, &tm, "abc")
	zigTry(t, h.setMode(ModesMode132Column, false))
	expectEqual(t, 80, tm.cols)
	sroExpectText(t, &tm, "")
	sroExpectCursor(t, &tm, 0, 0)

	// Resetting mode 40 does not undo the 132 state.
	zigTry(t, h.setMode(ModesMode132Column, true))
	zigTry(t, h.setMode(ModesModeEnableMode3, false))
	expect(t, tm.modes.Get(ModesMode132Column), "t.modes.get(.@\"132_column\")")
	expectEqual(t, 132, tm.cols)
	// But with mode 40 off a further change is ignored and clears the bit.
	zigTry(t, h.setMode(ModesMode132Column, false))
	expect(t, !tm.modes.Get(ModesMode132Column), "!t.modes.get(.@\"132_column\")")
	expectEqual(t, 132, tm.cols)
}

// Zig: test "term51: Handler.setMode: 132_column with mode 40 resizes both screens, and origin mode homes to 0,0"
func TestStreamReadonly_term51_Handler_setMode_132_column_with_mode_40_resizes_both_screens_and_origin_mode_homes_to_0_0(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	zigTry(t, h.setMode(ModesModeEnableMode3, true))
	zigTry(t, h.setMode(ModesModeAltScreenLegacy, true))
	expectEqual(t, ScreenSetKeyAlternate, tm.screens.active_key)
	sroPrintString(t, &tm, "alt")
	tm.modes.Set(ModesModeOrigin, true)

	zigTry(t, h.setMode(ModesMode132Column, true))
	expectEqual(t, 132, sroScreen(t, &tm, ScreenSetKeyPrimary).pages.cols)
	expectEqual(t, 132, sroScreen(t, &tm, ScreenSetKeyAlternate).pages.cols)
	expectEqual(t, ScreenSetKeyAlternate, tm.screens.active_key)
	sroExpectText(t, &tm, "")
	sroExpectCursor(t, &tm, 0, 0)
	expect(t, tm.modes.Get(ModesModeOrigin), "t.modes.get(.origin)")
}

// Zig: test "term51: Handler.setMode: mouse event modes set the event flag, reset clears it"
func TestStreamReadonly_term51_Handler_setMode_mouse_event_modes_set_the_event_flag_reset_clears_it(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	expectEqual(t, TerminalMouseEventsNone, tm.flags.mouse_event)
	for _, pair := range []struct {
		mode  ModesMode
		event TerminalMouseEvents
	}{
		{ModesModeMouseEventX10, TerminalMouseEventsX10},
		{ModesModeMouseEventNormal, TerminalMouseEventsNormal},
		{ModesModeMouseEventButton, TerminalMouseEventsButton},
		{ModesModeMouseEventAny, TerminalMouseEventsAny},
	} {
		zigTry(t, h.setMode(pair.mode, true))
		expect(t, tm.modes.Get(pair.mode), "t.modes.get(pair[0])")
		expectEqual(t, pair.event, tm.flags.mouse_event)
		expectEqual(t, TerminalMouseFormatX10, tm.flags.mouse_format)
		zigTry(t, h.setMode(pair.mode, true))
		expectEqual(t, pair.event, tm.flags.mouse_event)
		zigTry(t, h.setMode(pair.mode, false))
		expect(t, !tm.modes.Get(pair.mode), "!t.modes.get(pair[0])")
		expectEqual(t, TerminalMouseEventsNone, tm.flags.mouse_event)
		// Resetting when already off stays off.
		zigTry(t, h.setMode(pair.mode, false))
		expectEqual(t, TerminalMouseEventsNone, tm.flags.mouse_event)
	}
}

// Zig: test "term51: Handler.setMode: mouse event flag follows the last mode set, and any reset clears it"
func TestStreamReadonly_term51_Handler_setMode_mouse_event_flag_follows_the_last_mode_set_and_any_reset_clears_it(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	zigTry(t, h.setMode(ModesModeMouseEventX10, true))
	zigTry(t, h.setMode(ModesModeMouseEventNormal, true))
	expectEqual(t, TerminalMouseEventsNormal, tm.flags.mouse_event)
	zigTry(t, h.setMode(ModesModeMouseEventButton, true))
	expectEqual(t, TerminalMouseEventsButton, tm.flags.mouse_event)
	zigTry(t, h.setMode(ModesModeMouseEventAny, true))
	expectEqual(t, TerminalMouseEventsAny, tm.flags.mouse_event)
	expect(t, tm.modes.Get(ModesModeMouseEventX10), "t.modes.get(.mouse_event_x10)")
	expect(t, tm.modes.Get(ModesModeMouseEventNormal), "t.modes.get(.mouse_event_normal)")
	expect(t, tm.modes.Get(ModesModeMouseEventButton), "t.modes.get(.mouse_event_button)")
	expect(t, tm.modes.Get(ModesModeMouseEventAny), "t.modes.get(.mouse_event_any)")

	// Resetting a different mode than the one in effect still clears the
	// flag, even though the mode that is in effect is still set.
	zigTry(t, h.setMode(ModesModeMouseEventX10, false))
	expectEqual(t, TerminalMouseEventsNone, tm.flags.mouse_event)
	expect(t, tm.modes.Get(ModesModeMouseEventAny), "t.modes.get(.mouse_event_any)")

	// Even a mode that was never set.
	zigTry(t, h.setMode(ModesModeMouseEventNormal, true))
	expectEqual(t, TerminalMouseEventsNormal, tm.flags.mouse_event)
	zigTry(t, h.setMode(ModesModeMouseEventAny, false))
	expectEqual(t, TerminalMouseEventsNone, tm.flags.mouse_event)

	// Events and formats are tracked independently.
	zigTry(t, h.setMode(ModesModeMouseFormatSgr, true))
	zigTry(t, h.setMode(ModesModeMouseEventButton, true))
	expectEqual(t, TerminalMouseEventsButton, tm.flags.mouse_event)
	expectEqual(t, TerminalMouseFormatSgr, tm.flags.mouse_format)
	zigTry(t, h.setMode(ModesModeMouseEventButton, false))
	expectEqual(t, TerminalMouseFormatSgr, tm.flags.mouse_format)
}

// Zig: test "term51: Handler.setMode: mouse format modes set the format flag, reset goes back to x10"
func TestStreamReadonly_term51_Handler_setMode_mouse_format_modes_set_the_format_flag_reset_goes_back_to_x10(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	expectEqual(t, TerminalMouseFormatX10, tm.flags.mouse_format)
	for _, pair := range []struct {
		mode   ModesMode
		format TerminalMouseFormat
	}{
		{ModesModeMouseFormatUtf8, TerminalMouseFormatUtf8},
		{ModesModeMouseFormatSgr, TerminalMouseFormatSgr},
		{ModesModeMouseFormatUrxvt, TerminalMouseFormatUrxvt},
		{ModesModeMouseFormatSgrPixels, TerminalMouseFormatSgrPixels},
	} {
		zigTry(t, h.setMode(pair.mode, true))
		expect(t, tm.modes.Get(pair.mode), "t.modes.get(pair[0])")
		expectEqual(t, pair.format, tm.flags.mouse_format)
		expectEqual(t, TerminalMouseEventsNone, tm.flags.mouse_event)
		zigTry(t, h.setMode(pair.mode, false))
		expect(t, !tm.modes.Get(pair.mode), "!t.modes.get(pair[0])")
		expectEqual(t, TerminalMouseFormatX10, tm.flags.mouse_format)
	}

	// Last one set wins, and resetting any of them clears the format.
	zigTry(t, h.setMode(ModesModeMouseFormatUtf8, true))
	zigTry(t, h.setMode(ModesModeMouseFormatSgr, true))
	zigTry(t, h.setMode(ModesModeMouseFormatUrxvt, true))
	zigTry(t, h.setMode(ModesModeMouseFormatSgrPixels, true))
	expectEqual(t, TerminalMouseFormatSgrPixels, tm.flags.mouse_format)
	zigTry(t, h.setMode(ModesModeMouseFormatUtf8, false))
	expectEqual(t, TerminalMouseFormatX10, tm.flags.mouse_format)
	expect(t, tm.modes.Get(ModesModeMouseFormatSgrPixels), "t.modes.get(.mouse_format_sgr_pixels)")
	expect(t, tm.modes.Get(ModesModeMouseFormatUrxvt), "t.modes.get(.mouse_format_urxvt)")
	expect(t, tm.modes.Get(ModesModeMouseFormatSgr), "t.modes.get(.mouse_format_sgr)")
}

// Zig: test "term51: Handler.vt: set_mode and reset_mode run the side effects of special modes"
func TestStreamReadonly_term51_Handler_vt_set_mode_and_reset_mode_run_the_side_effects_of_special_modes(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 10)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroPrintString(t, &tm, "text")
	tm.SetTopAndBottomMargin(3, 8)
	sroAt(&tm, 9, 9)

	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeOrigin))
	expect(t, tm.modes.Get(ModesModeOrigin), "t.modes.get(.origin)")
	sroExpectCursor(t, &tm, 0, 2)
	sroVt(t, &h, sroModeAct(StreamActionKeyResetMode, ModesModeOrigin))
	expect(t, !tm.modes.Get(ModesModeOrigin), "!t.modes.get(.origin)")
	sroExpectCursor(t, &tm, 0, 0)

	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeAltScreenSaveCursorClearEnter))
	expectEqual(t, ScreenSetKeyAlternate, tm.screens.active_key)
	sroVt(t, &h, sroModeAct(StreamActionKeyResetMode, ModesModeAltScreenSaveCursorClearEnter))
	expectEqual(t, ScreenSetKeyPrimary, tm.screens.active_key)

	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeMouseEventAny))
	expectEqual(t, TerminalMouseEventsAny, tm.flags.mouse_event)
	sroVt(t, &h, sroModeAct(StreamActionKeyResetMode, ModesModeMouseEventAny))
	expectEqual(t, TerminalMouseEventsNone, tm.flags.mouse_event)

	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeMouseFormatUrxvt))
	expectEqual(t, TerminalMouseFormatUrxvt, tm.flags.mouse_format)
	sroVt(t, &h, sroModeAct(StreamActionKeyResetMode, ModesModeMouseFormatUrxvt))
	expectEqual(t, TerminalMouseFormatX10, tm.flags.mouse_format)

	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeEnableLeftAndRightMargin))
	tm.SetLeftAndRightMargin(2, 5)
	expectEqual(t, 4, tm.scrolling_region.right)
	sroVt(t, &h, sroModeAct(StreamActionKeyResetMode, ModesModeEnableLeftAndRightMargin))
	expectEqual(t, 19, tm.scrolling_region.right)
	expectEqual(t, 0, tm.scrolling_region.left)

	// 132 columns without mode 40.
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesMode132Column))
	expect(t, !tm.modes.Get(ModesMode132Column), "!t.modes.get(.@\"132_column\")")
	expectEqual(t, 20, tm.cols)
}

// Zig: test "term51: Handler.vt: save_mode and restore_mode round trip through setMode"
func TestStreamReadonly_term51_Handler_vt_save_mode_and_restore_mode_round_trip_through_setMode(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 10)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	// Saving changes only the saved bit, never the value.
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeCursorKeys))
	before := sroSnapshot(t, &tm)
	sroVt(t, &h, sroModeAct(StreamActionKeySaveMode, ModesModeCursorKeys))
	after := sroSnapshot(t, &tm)
	expect(t, tm.modes.saved.cursor_keys, "t.modes.saved.cursor_keys")
	expect(t, tm.modes.Get(ModesModeCursorKeys), "t.modes.get(.cursor_keys)")
	before.saved_modes = after.saved_modes
	expectEqual(t, before, after)

	// Change then restore.
	sroVt(t, &h, sroModeAct(StreamActionKeyResetMode, ModesModeCursorKeys))
	expect(t, !tm.modes.Get(ModesModeCursorKeys), "!t.modes.get(.cursor_keys)")
	sroVt(t, &h, sroModeAct(StreamActionKeyRestoreMode, ModesModeCursorKeys))
	expect(t, tm.modes.Get(ModesModeCursorKeys), "t.modes.get(.cursor_keys)")
	// Restoring twice gives the same value (the saved bit is kept).
	sroVt(t, &h, sroModeAct(StreamActionKeyResetMode, ModesModeCursorKeys))
	sroVt(t, &h, sroModeAct(StreamActionKeyRestoreMode, ModesModeCursorKeys))
	expect(t, tm.modes.Get(ModesModeCursorKeys), "t.modes.get(.cursor_keys)")

	// A second save replaces the first.
	sroVt(t, &h, sroModeAct(StreamActionKeyResetMode, ModesModeCursorKeys))
	sroVt(t, &h, sroModeAct(StreamActionKeySaveMode, ModesModeCursorKeys))
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeCursorKeys))
	sroVt(t, &h, sroModeAct(StreamActionKeyRestoreMode, ModesModeCursorKeys))
	expect(t, !tm.modes.Get(ModesModeCursorKeys), "!t.modes.get(.cursor_keys)")

	// Modes are saved independently.
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeBracketedPaste))
	sroVt(t, &h, sroModeAct(StreamActionKeySaveMode, ModesModeBracketedPaste))
	sroVt(t, &h, sroModeAct(StreamActionKeyResetMode, ModesModeBracketedPaste))
	sroVt(t, &h, sroModeAct(StreamActionKeyRestoreMode, ModesModeCursorKeys))
	expect(t, !tm.modes.Get(ModesModeBracketedPaste), "!t.modes.get(.bracketed_paste)")
	expect(t, !tm.modes.Get(ModesModeCursorKeys), "!t.modes.get(.cursor_keys)")
	sroVt(t, &h, sroModeAct(StreamActionKeyRestoreMode, ModesModeBracketedPaste))
	expect(t, tm.modes.Get(ModesModeBracketedPaste), "t.modes.get(.bracketed_paste)")

	// Restoring a mode that was never saved restores its default value.
	sroVt(t, &h, sroModeAct(StreamActionKeyResetMode, ModesModeWraparound))
	expect(t, !tm.modes.Get(ModesModeWraparound), "!t.modes.get(.wraparound)")
	sroVt(t, &h, sroModeAct(StreamActionKeyRestoreMode, ModesModeWraparound))
	expect(t, tm.modes.Get(ModesModeWraparound), "t.modes.get(.wraparound)")
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeInsert))
	sroVt(t, &h, sroModeAct(StreamActionKeyRestoreMode, ModesModeInsert))
	expect(t, !tm.modes.Get(ModesModeInsert), "!t.modes.get(.insert)")

	// Restore does not change anything but the mode.
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeKeypadKeys))
	sroVt(t, &h, sroModeAct(StreamActionKeySaveMode, ModesModeKeypadKeys))
	sroVt(t, &h, sroModeAct(StreamActionKeyResetMode, ModesModeKeypadKeys))
	before = sroSnapshot(t, &tm)
	sroVt(t, &h, sroModeAct(StreamActionKeyRestoreMode, ModesModeKeypadKeys))
	after = sroSnapshot(t, &tm)
	before.modes = sroModesWith(before.modes, ModesModeKeypadKeys, true)
	expectEqual(t, before, after)
}

// Zig: test "term51: Handler.vt: restore_mode runs the side effects of the restored value"
func TestStreamReadonly_term51_Handler_vt_restore_mode_runs_the_side_effects_of_the_restored_value(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 10)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	// Origin: save off, set on (cursor homes relative to the margin),
	// restore: off and the cursor homes to 0,0.
	tm.SetTopAndBottomMargin(3, 8)
	sroVt(t, &h, sroModeAct(StreamActionKeySaveMode, ModesModeOrigin))
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeOrigin))
	sroExpectCursor(t, &tm, 0, 2)
	sroAt(&tm, 5, 5)
	sroVt(t, &h, sroModeAct(StreamActionKeyRestoreMode, ModesModeOrigin))
	expect(t, !tm.modes.Get(ModesModeOrigin), "!t.modes.get(.origin)")
	sroExpectCursor(t, &tm, 0, 0)

	// Alternate screen: save while on primary, enter 1049, restore leaves.
	sroPrintString(t, &tm, "main")
	sroAt(&tm, 3, 3)
	sroVt(t, &h, sroModeAct(StreamActionKeySaveMode, ModesModeAltScreenSaveCursorClearEnter))
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeAltScreenSaveCursorClearEnter))
	expectEqual(t, ScreenSetKeyAlternate, tm.screens.active_key)
	sroAt(&tm, 8, 8)
	sroVt(t, &h, sroModeAct(StreamActionKeyRestoreMode, ModesModeAltScreenSaveCursorClearEnter))
	expect(t, !tm.modes.Get(ModesModeAltScreenSaveCursorClearEnter), "!t.modes.get(.alt_screen_save_cursor_clear_enter)")
	expectEqual(t, ScreenSetKeyPrimary, tm.screens.active_key)
	sroExpectCursor(t, &tm, 3, 3)

	// Restoring a saved "on" value enters the alt screen.
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeAltScreen))
	sroVt(t, &h, sroModeAct(StreamActionKeySaveMode, ModesModeAltScreen))
	sroVt(t, &h, sroModeAct(StreamActionKeyResetMode, ModesModeAltScreen))
	expectEqual(t, ScreenSetKeyPrimary, tm.screens.active_key)
	sroVt(t, &h, sroModeAct(StreamActionKeyRestoreMode, ModesModeAltScreen))
	expectEqual(t, ScreenSetKeyAlternate, tm.screens.active_key)
	expect(t, tm.modes.Get(ModesModeAltScreen), "t.modes.get(.alt_screen)")

	// Mouse flags.
	sroVt(t, &h, sroModeAct(StreamActionKeySaveMode, ModesModeMouseEventNormal))
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeMouseEventNormal))
	expectEqual(t, TerminalMouseEventsNormal, tm.flags.mouse_event)
	sroVt(t, &h, sroModeAct(StreamActionKeyRestoreMode, ModesModeMouseEventNormal))
	expectEqual(t, TerminalMouseEventsNone, tm.flags.mouse_event)
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeMouseFormatSgr))
	sroVt(t, &h, sroModeAct(StreamActionKeySaveMode, ModesModeMouseFormatSgr))
	sroVt(t, &h, sroModeAct(StreamActionKeyResetMode, ModesModeMouseFormatSgr))
	expectEqual(t, TerminalMouseFormatX10, tm.flags.mouse_format)
	sroVt(t, &h, sroModeAct(StreamActionKeyRestoreMode, ModesModeMouseFormatSgr))
	expectEqual(t, TerminalMouseFormatSgr, tm.flags.mouse_format)

	// Left and right margin mode: restoring "off" resets the margins.
	sroVt(t, &h, sroModeAct(StreamActionKeyResetMode, ModesModeAltScreen))
	sroVt(t, &h, sroModeAct(StreamActionKeySaveMode, ModesModeEnableLeftAndRightMargin))
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeEnableLeftAndRightMargin))
	tm.SetLeftAndRightMargin(3, 6)
	expectEqual(t, 5, tm.scrolling_region.right)
	sroVt(t, &h, sroModeAct(StreamActionKeyRestoreMode, ModesModeEnableLeftAndRightMargin))
	expect(t, !tm.modes.Get(ModesModeEnableLeftAndRightMargin), "!t.modes.get(.enable_left_and_right_margin)")
	expectEqual(t, 19, tm.scrolling_region.right)
}

// Zig: test "term51: Handler.vt: unknown modes and ANSI versus DEC mode numbers through a stream"
func TestStreamReadonly_term51_Handler_vt_unknown_modes_and_ANSI_versus_DEC_mode_numbers_through_a_stream(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 10)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()
	sroBusy(t, &tm, s.handler)

	// Unknown modes change nothing, either for set, reset, save or restore.
	before := sroSnapshot(t, &tm)
	zigTry(t, s.NextSlice([]uint8("\x1b[?9999h\x1b[?9999l\x1b[?0h\x1b[?0l")))
	zigTry(t, s.NextSlice([]uint8("\x1b[99h\x1b[99l\x1b[1h\x1b[1l\x1b[?20h\x1b[?20l")))
	zigTry(t, s.NextSlice([]uint8("\x1b[?9999s\x1b[?9999r")))
	expectEqual(t, before, sroSnapshot(t, &tm))

	// ANSI and DEC numbering are different modes.
	zigTry(t, s.NextSlice([]uint8("\x1b[4h")))
	expect(t, tm.modes.Get(ModesModeInsert), "t.modes.get(.insert)")
	expect(t, !tm.modes.Get(ModesModeSlowScroll), "!t.modes.get(.slow_scroll)")
	zigTry(t, s.NextSlice([]uint8("\x1b[4l\x1b[?4h")))
	expect(t, !tm.modes.Get(ModesModeInsert), "!t.modes.get(.insert)")
	expect(t, tm.modes.Get(ModesModeSlowScroll), "t.modes.get(.slow_scroll)")
	zigTry(t, s.NextSlice([]uint8("\x1b[12l")))
	expect(t, !tm.modes.Get(ModesModeSendReceiveMode), "!t.modes.get(.send_receive_mode)")
	expect(t, !tm.modes.Get(ModesModeCursorBlinking), "!t.modes.get(.cursor_blinking)")
	zigTry(t, s.NextSlice([]uint8("\x1b[?12h")))
	expect(t, tm.modes.Get(ModesModeCursorBlinking), "t.modes.get(.cursor_blinking)")
	zigTry(t, s.NextSlice([]uint8("\x1b[20h")))
	expect(t, tm.modes.Get(ModesModeLinefeed), "t.modes.get(.linefeed)")

	// Several modes in one sequence are applied in order.
	zigTry(t, s.NextSlice([]uint8("\x1b[?1000;1006;2004h")))
	expectEqual(t, TerminalMouseEventsNormal, tm.flags.mouse_event)
	expectEqual(t, TerminalMouseFormatSgr, tm.flags.mouse_format)
	expect(t, tm.modes.Get(ModesModeBracketedPaste), "t.modes.get(.bracketed_paste)")
	zigTry(t, s.NextSlice([]uint8("\x1b[?1000;1006;2004l")))
	expectEqual(t, TerminalMouseEventsNone, tm.flags.mouse_event)
	expectEqual(t, TerminalMouseFormatX10, tm.flags.mouse_format)
	expect(t, !tm.modes.Get(ModesModeBracketedPaste), "!t.modes.get(.bracketed_paste)")

	// XTSAVE/XTRESTORE.
	zigTry(t, s.NextSlice([]uint8("\x1b[?25l\x1b[?25s\x1b[?25h")))
	expect(t, tm.modes.Get(ModesModeCursorVisible), "t.modes.get(.cursor_visible)")
	zigTry(t, s.NextSlice([]uint8("\x1b[?25r")))
	expect(t, !tm.modes.Get(ModesModeCursorVisible), "!t.modes.get(.cursor_visible)")
}

// Zig: test "term51: Handler.vt: print writes the codepoint and advances the cursor"
func TestStreamReadonly_term51_Handler_vt_print_writes_the_codepoint_and_advances_the_cursor(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroVt(t, &h, sroPrintAct('A'))
	sroExpectText(t, &tm, "A")
	sroExpectCursor(t, &tm, 1, 0)
	expectEqual(t, Some(uint32('A')), tm.previous_char)

	// A wide character takes two cells.
	sroVt(t, &h, sroPrintAct(0x4E2D))
	sroExpectText(t, &tm, "A中")
	sroExpectCursor(t, &tm, 3, 0)
	expectEqual(t, Some(uint32(0x4E2D)), tm.previous_char)

	// The maximum codepoint is accepted.
	sroVt(t, &h, sroPrintAct('z'))
	sroExpectText(t, &tm, "A中z")
}

// Zig: test "term51: Handler.vt: print wraps at the last column, or overwrites it without wraparound"
func TestStreamReadonly_term51_Handler_vt_print_wraps_at_the_last_column_or_overwrites_it_without_wraparound(t *testing.T) {
	testStdCapacity(t)
	func() {
		tm := sroNew(t, 10, 3)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)
		for _, c := range []uint8("0123456789") {
			sroVt(t, &h, sroPrintAct(uint32(c)))
		}
		sroExpectCursor(t, &tm, 9, 0)
		expect(t, tm.screens.active.cursor.pending_wrap, "t.screens.active.cursor.pending_wrap")
		sroVt(t, &h, sroPrintAct('B'))
		sroExpectCursor(t, &tm, 1, 1)
		expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
		sroExpectText(t, &tm, "0123456789\nB")
	}()
	func() {
		tm := sroNew(t, 10, 3)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)
		sroVt(t, &h, sroModeAct(StreamActionKeyResetMode, ModesModeWraparound))
		for _, c := range []uint8("0123456789K") {
			sroVt(t, &h, sroPrintAct(uint32(c)))
		}
		sroExpectCursor(t, &tm, 9, 0)
		sroExpectText(t, &tm, "012345678K")
	}()
}

// Zig: test "term51: Handler.vt: print honours insert mode and the charset state"
func TestStreamReadonly_term51_Handler_vt_print_honours_insert_mode_and_the_charset_state(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroPrintString(t, &tm, "abc")
	sroAt(&tm, 0, 0)
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeInsert))
	sroVt(t, &h, sroPrintAct('X'))
	sroExpectText(t, &tm, "Xabc")
	sroExpectCursor(t, &tm, 1, 0)
	sroVt(t, &h, sroModeAct(StreamActionKeyResetMode, ModesModeInsert))
	sroVt(t, &h, sroPrintAct('Y'))
	sroExpectText(t, &tm, "XYbc")

	// DEC special graphics in G0.
	sroVt(t, &h, sroConfigureCharsetAct(CharsetsSlotsG0, CharsetsCharsetDecSpecial))
	sroVt(t, &h, sroPrintAct('`'))
	sroExpectText(t, &tm, "XY◆c")
}

// Zig: test "term51: Handler.vt: print_repeat repeats the previous character, a count of 0 acts as 1"
func TestStreamReadonly_term51_Handler_vt_print_repeat_repeats_the_previous_character_a_count_of_0_acts_as_1(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	// Nothing printed yet: no effect.
	sroVt(t, &h, sroCountAct(StreamActionKeyPrintRepeat, 5))
	sroExpectText(t, &tm, "")
	sroExpectCursor(t, &tm, 0, 0)

	sroVt(t, &h, sroPrintAct('x'))
	sroVt(t, &h, sroCountAct(StreamActionKeyPrintRepeat, 0))
	sroExpectText(t, &tm, "xx")
	sroVt(t, &h, sroCountAct(StreamActionKeyPrintRepeat, 1))
	sroExpectText(t, &tm, "xxx")
	sroVt(t, &h, sroCountAct(StreamActionKeyPrintRepeat, 3))
	sroExpectText(t, &tm, "xxxxxx")
	sroExpectCursor(t, &tm, 6, 0)

	// Wraps like normal printing.
	sroVt(t, &h, sroCountAct(StreamActionKeyPrintRepeat, 7))
	sroExpectText(t, &tm, "xxxxxxxxxx\nxxx")
	sroExpectCursor(t, &tm, 3, 1)
}

// Zig: test "term51: Handler.vt: print_repeat with a huge count scrolls"
func TestStreamReadonly_term51_Handler_vt_print_repeat_with_a_huge_count_scrolls(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroVt(t, &h, sroPrintAct('x'))
	sroVt(t, &h, sroCountAct(StreamActionKeyPrintRepeat, 65535))
	// 65536 characters in total: 6553 full rows and 6 more characters.
	sroExpectCursor(t, &tm, 6, 4)
	sroExpectText(t, &tm, "xxxxxxxxxx\nxxxxxxxxxx\nxxxxxxxxxx\nxxxxxxxxxx\nxxxxxx")
}

// Zig: test "term51: Handler.vt: backspace moves left, stops at column 0 and clears pending wrap"
func TestStreamReadonly_term51_Handler_vt_backspace_moves_left_stops_at_column_0_and_clears_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroAt(&tm, 3, 1)
	sroVt(t, &h, sroAct(StreamActionKeyBackspace))
	sroExpectCursor(t, &tm, 2, 1)
	sroVt(t, &h, sroAct(StreamActionKeyBackspace))
	sroVt(t, &h, sroAct(StreamActionKeyBackspace))
	sroExpectCursor(t, &tm, 0, 1)
	sroVt(t, &h, sroAct(StreamActionKeyBackspace))
	sroExpectCursor(t, &tm, 0, 1)

	// From the pending wrap state at the last column the cursor moves
	// to column 8 (wrap mode without reverse wrap) and the flag clears.
	sroPrintString(t, &tm, "0123456789")
	// ^ the row above is row 1, the print started at x=0 on row 1.
	expect(t, tm.screens.active.cursor.pending_wrap, "t.screens.active.cursor.pending_wrap")
	sroVt(t, &h, sroAct(StreamActionKeyBackspace))
	sroExpectCursor(t, &tm, 8, 1)
	expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")

	// Backspace does not erase.
	sroExpectText(t, &tm, "\n0123456789")
}

// Zig: test "term51: Handler.vt: carriage_return goes to column 0 or the left margin"
func TestStreamReadonly_term51_Handler_vt_carriage_return_goes_to_column_0_or_the_left_margin(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroAt(&tm, 5, 1)
	tm.screens.active.cursor.pending_wrap = true
	sroVt(t, &h, sroAct(StreamActionKeyCarriageReturn))
	sroExpectCursor(t, &tm, 0, 1)
	expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
	sroVt(t, &h, sroAct(StreamActionKeyCarriageReturn))
	sroExpectCursor(t, &tm, 0, 1)

	// Left margin at column 3: inside or at the margin goes to it, left of
	// it goes to column 0.
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeEnableLeftAndRightMargin))
	sroVt(t, &h, sroMarginAct(StreamActionKeyLeftAndRightMargin, 4, 8))
	sroAt(&tm, 5, 1)
	sroVt(t, &h, sroAct(StreamActionKeyCarriageReturn))
	sroExpectCursor(t, &tm, 3, 1)
	sroVt(t, &h, sroAct(StreamActionKeyCarriageReturn))
	sroExpectCursor(t, &tm, 3, 1)
	sroAt(&tm, 1, 1)
	sroVt(t, &h, sroAct(StreamActionKeyCarriageReturn))
	sroExpectCursor(t, &tm, 0, 1)
	// Right of the right margin as well.
	sroAt(&tm, 9, 1)
	sroVt(t, &h, sroAct(StreamActionKeyCarriageReturn))
	sroExpectCursor(t, &tm, 3, 1)

	// Origin mode always goes to the left margin.
	tm.modes.Set(ModesModeOrigin, true)
	sroAt(&tm, 1, 1)
	sroVt(t, &h, sroAct(StreamActionKeyCarriageReturn))
	sroExpectCursor(t, &tm, 3, 1)
}

// Zig: test "term51: Handler.vt: linefeed moves down, with newline mode also to column 0"
func TestStreamReadonly_term51_Handler_vt_linefeed_moves_down_with_newline_mode_also_to_column_0(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewFilled(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroAt(&tm, 5, 1)
	tm.screens.active.cursor.pending_wrap = true
	sroVt(t, &h, sroAct(StreamActionKeyLinefeed))
	sroExpectCursor(t, &tm, 5, 2)
	expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")

	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeLinefeed))
	sroVt(t, &h, sroAct(StreamActionKeyLinefeed))
	sroExpectCursor(t, &tm, 0, 3)
	sroVt(t, &h, sroModeAct(StreamActionKeyResetMode, ModesModeLinefeed))
	sroAt(&tm, 4, 3)
	sroVt(t, &h, sroAct(StreamActionKeyLinefeed))
	sroExpectCursor(t, &tm, 4, 4)

	// At the bottom the screen scrolls.
	sroVt(t, &h, sroAct(StreamActionKeyLinefeed))
	sroExpectCursor(t, &tm, 4, 4)
	sroExpectText(t, &tm, "abcdefghij\nABCDEFGHIJ\nklmnopqrst\nKLMNOPQRST")
}

// Zig: test "term51: Handler.vt: index moves down without a carriage return and scrolls at the bottom"
func TestStreamReadonly_term51_Handler_vt_index_moves_down_without_a_carriage_return_and_scrolls_at_the_bottom(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewFilled(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	// Index never returns the carriage, even with newline mode.
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeLinefeed))
	sroAt(&tm, 5, 1)
	sroVt(t, &h, sroAct(StreamActionKeyIndex))
	sroExpectCursor(t, &tm, 5, 2)

	// At the bottom of a scroll region only the region scrolls.
	tm.SetTopAndBottomMargin(2, 4)
	sroAt(&tm, 0, 3)
	sroVt(t, &h, sroAct(StreamActionKeyIndex))
	sroExpectCursor(t, &tm, 0, 3)
	sroExpectText(t, &tm, "0123456789\nABCDEFGHIJ\nklmnopqrst\n\nKLMNOPQRST")

	// Below the region the cursor is at the bottom of the screen: no move.
	sroAt(&tm, 0, 4)
	sroVt(t, &h, sroAct(StreamActionKeyIndex))
	sroExpectCursor(t, &tm, 0, 4)
	sroExpectText(t, &tm, "0123456789\nABCDEFGHIJ\nklmnopqrst\n\nKLMNOPQRST")

	// Above the region it just moves down.
	sroAt(&tm, 2, 0)
	sroVt(t, &h, sroAct(StreamActionKeyIndex))
	sroExpectCursor(t, &tm, 2, 1)
}

// Zig: test "term51: Handler.vt: next_line is an index followed by a carriage return"
func TestStreamReadonly_term51_Handler_vt_next_line_is_an_index_followed_by_a_carriage_return(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewFilled(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroAt(&tm, 5, 1)
	tm.screens.active.cursor.pending_wrap = true
	sroVt(t, &h, sroAct(StreamActionKeyNextLine))
	sroExpectCursor(t, &tm, 0, 2)
	expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")

	// At the bottom: scrolls and the cursor ends in column 0.
	sroAt(&tm, 6, 4)
	sroVt(t, &h, sroAct(StreamActionKeyNextLine))
	sroExpectCursor(t, &tm, 0, 4)
	sroExpectText(t, &tm, "abcdefghij\nABCDEFGHIJ\nklmnopqrst\nKLMNOPQRST")

	// With a left margin the carriage return goes to the margin.
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeEnableLeftAndRightMargin))
	sroVt(t, &h, sroMarginAct(StreamActionKeyLeftAndRightMargin, 4, 8))
	sroAt(&tm, 5, 1)
	sroVt(t, &h, sroAct(StreamActionKeyNextLine))
	sroExpectCursor(t, &tm, 3, 2)
	sroAt(&tm, 1, 1)
	sroVt(t, &h, sroAct(StreamActionKeyNextLine))
	sroExpectCursor(t, &tm, 0, 2)
}

// Zig: test "term51: Handler.vt: reverse_index moves up or scrolls down at the top of the region"
func TestStreamReadonly_term51_Handler_vt_reverse_index_moves_up_or_scrolls_down_at_the_top_of_the_region(t *testing.T) {
	testStdCapacity(t)
	func() {
		tm := sroNewFilled(t)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)

		sroAt(&tm, 4, 2)
		sroVt(t, &h, sroAct(StreamActionKeyReverseIndex))
		sroExpectCursor(t, &tm, 4, 1)
		sroVt(t, &h, sroAct(StreamActionKeyReverseIndex))
		sroExpectCursor(t, &tm, 4, 0)
		sroExpectText(t, &tm, sroFilled)

		// At the top of the screen the content scrolls down; the cursor
		// does not move.
		sroVt(t, &h, sroAct(StreamActionKeyReverseIndex))
		sroExpectCursor(t, &tm, 4, 0)
		sroExpectText(t, &tm, "\n0123456789\nabcdefghij\nABCDEFGHIJ\nklmnopqrst")
	}()
	func() {
		tm := sroNewFilled(t)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)

		// At the top margin only the region scrolls.
		tm.SetTopAndBottomMargin(2, 4)
		sroAt(&tm, 0, 1)
		sroVt(t, &h, sroAct(StreamActionKeyReverseIndex))
		sroExpectCursor(t, &tm, 0, 1)
		sroExpectText(t, &tm, "0123456789\n\nabcdefghij\nABCDEFGHIJ\nKLMNOPQRST")

		// Above the top margin it just moves up (and stops at row 0).
		sroAt(&tm, 0, 0)
		sroVt(t, &h, sroAct(StreamActionKeyReverseIndex))
		sroExpectCursor(t, &tm, 0, 0)
		sroExpectText(t, &tm, "0123456789\n\nabcdefghij\nABCDEFGHIJ\nKLMNOPQRST")
	}()
}

// Zig: test "term51: Handler.vt: cursor_up and cursor_down clamp, 0 acts as 1"
func TestStreamReadonly_term51_Handler_vt_cursor_up_and_cursor_down_clamp_0_acts_as_1(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	up := []struct {
		n uint16
		y uint
	}{
		{0, 1},
		{1, 1},
		{2, 0},
		{3, 0},
		{65535, 0},
	}
	for _, c := range up {
		sroAt(&tm, 5, 2)
		tm.screens.active.cursor.pending_wrap = true
		sroVt(t, &h, sroMoveAct(StreamActionKeyCursorUp, c.n))
		sroExpectCursor(t, &tm, 5, c.y)
		expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
	}
	down := []struct {
		n uint16
		y uint
	}{
		{0, 3},
		{1, 3},
		{2, 4},
		{3, 4},
		{65535, 4},
	}
	for _, c := range down {
		sroAt(&tm, 5, 2)
		tm.screens.active.cursor.pending_wrap = true
		sroVt(t, &h, sroMoveAct(StreamActionKeyCursorDown, c.n))
		sroExpectCursor(t, &tm, 5, c.y)
		expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
	}
}

// Zig: test "term51: Handler.vt: cursor_up and cursor_down respect the top and bottom margins"
func TestStreamReadonly_term51_Handler_vt_cursor_up_and_cursor_down_respect_the_top_and_bottom_margins(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroVt(t, &h, sroMarginAct(StreamActionKeyTopAndBottomMargin, 2, 4))
	// Inside the region: stops at the margins.
	sroAt(&tm, 0, 2)
	sroVt(t, &h, sroMoveAct(StreamActionKeyCursorUp, 65535))
	sroExpectCursor(t, &tm, 0, 1)
	sroAt(&tm, 0, 2)
	sroVt(t, &h, sroMoveAct(StreamActionKeyCursorDown, 65535))
	sroExpectCursor(t, &tm, 0, 3)
	// Above the region: up does not move, down may go to the bottom margin.
	sroAt(&tm, 0, 0)
	sroVt(t, &h, sroMoveAct(StreamActionKeyCursorUp, 3))
	sroExpectCursor(t, &tm, 0, 0)
	sroVt(t, &h, sroMoveAct(StreamActionKeyCursorDown, 65535))
	sroExpectCursor(t, &tm, 0, 3)
	// Below the region: down does not move, up stops at the top margin.
	sroAt(&tm, 0, 4)
	sroVt(t, &h, sroMoveAct(StreamActionKeyCursorDown, 3))
	sroExpectCursor(t, &tm, 0, 4)
	sroVt(t, &h, sroMoveAct(StreamActionKeyCursorUp, 65535))
	sroExpectCursor(t, &tm, 0, 1)
}

// Zig: test "term51: Handler.vt: cursor_left and cursor_right clamp, 0 acts as 1"
func TestStreamReadonly_term51_Handler_vt_cursor_left_and_cursor_right_clamp_0_acts_as_1(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	left := []struct {
		n uint16
		x uint
	}{
		{0, 4},
		{1, 4},
		{3, 2},
		{5, 0},
		{6, 0},
		{65535, 0},
	}
	for _, c := range left {
		sroAt(&tm, 5, 2)
		tm.screens.active.cursor.pending_wrap = true
		sroVt(t, &h, sroMoveAct(StreamActionKeyCursorLeft, c.n))
		sroExpectCursor(t, &tm, c.x, 2)
		expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
	}
	right := []struct {
		n uint16
		x uint
	}{
		{0, 6},
		{1, 6},
		{3, 8},
		{4, 9},
		{5, 9},
		{65535, 9},
	}
	for _, c := range right {
		sroAt(&tm, 5, 2)
		tm.screens.active.cursor.pending_wrap = true
		sroVt(t, &h, sroMoveAct(StreamActionKeyCursorRight, c.n))
		sroExpectCursor(t, &tm, c.x, 2)
		expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
	}
}

// Zig: test "term51: Handler.vt: cursor_right stops at the right margin, cursor_left ignores margins"
func TestStreamReadonly_term51_Handler_vt_cursor_right_stops_at_the_right_margin_cursor_left_ignores_margins(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeEnableLeftAndRightMargin))
	sroVt(t, &h, sroMarginAct(StreamActionKeyLeftAndRightMargin, 4, 8))
	sroAt(&tm, 5, 2)
	sroVt(t, &h, sroMoveAct(StreamActionKeyCursorRight, 65535))
	sroExpectCursor(t, &tm, 7, 2)
	sroVt(t, &h, sroMoveAct(StreamActionKeyCursorRight, 1))
	sroExpectCursor(t, &tm, 7, 2)
	sroAt(&tm, 0, 2)
	sroVt(t, &h, sroMoveAct(StreamActionKeyCursorRight, 65535))
	sroExpectCursor(t, &tm, 7, 2)
	// Right of the margin it can move to the edge of the screen only.
	sroAt(&tm, 8, 2)
	sroVt(t, &h, sroMoveAct(StreamActionKeyCursorRight, 65535))
	sroExpectCursor(t, &tm, 9, 2)
	sroVt(t, &h, sroMoveAct(StreamActionKeyCursorRight, 65535))
	sroExpectCursor(t, &tm, 9, 2)
	// Left goes to column 0 even across the left margin.
	sroAt(&tm, 5, 2)
	sroVt(t, &h, sroMoveAct(StreamActionKeyCursorLeft, 65535))
	sroExpectCursor(t, &tm, 0, 2)
}

// Zig: test "term51: Handler.vt: cursor_left with reverse wrap modes"
func TestStreamReadonly_term51_Handler_vt_cursor_left_with_reverse_wrap_modes(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	// Without reverse wrap the cursor stops at column 0.
	sroAt(&tm, 2, 2)
	sroVt(t, &h, sroMoveAct(StreamActionKeyCursorLeft, 5))
	sroExpectCursor(t, &tm, 0, 2)

	// Reverse wrap (mode 45) wraps to the previous line only if that
	// line was soft wrapped.
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeReverseWrap))
	sroAt(&tm, 2, 2)
	sroVt(t, &h, sroMoveAct(StreamActionKeyCursorLeft, 5))
	sroExpectCursor(t, &tm, 0, 2)
	sroVt(t, &h, sroModeAct(StreamActionKeyResetMode, ModesModeReverseWrap))

	// Extended reverse wrap (mode 1045) wraps regardless: 2 left to column
	// 0, the wrap itself costs one more, and the remaining 2 move left from
	// the last column of the previous line.
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeReverseWrapExtended))
	sroAt(&tm, 2, 2)
	sroVt(t, &h, sroMoveAct(StreamActionKeyCursorLeft, 5))
	sroExpectCursor(t, &tm, 7, 1)

	// Without wraparound mode there is no reverse wrapping at all.
	sroVt(t, &h, sroModeAct(StreamActionKeyResetMode, ModesModeWraparound))
	sroAt(&tm, 2, 2)
	sroVt(t, &h, sroMoveAct(StreamActionKeyCursorLeft, 5))
	sroExpectCursor(t, &tm, 0, 2)
}

// Zig: test "term51: Handler.vt: cursor_pos is 1-indexed, 0 acts as 1 and values clamp"
func TestStreamReadonly_term51_Handler_vt_cursor_pos_is_1_indexed_0_acts_as_1_and_values_clamp(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	cases := []struct {
		row uint16
		col uint16
		x   uint
		y   uint
	}{
		{1, 1, 0, 0},
		{0, 0, 0, 0},
		{3, 4, 3, 2},
		{0, 5, 4, 0},
		{5, 0, 0, 4},
		{5, 10, 9, 4},
		{6, 11, 9, 4},
		{65535, 65535, 9, 4},
		{65535, 1, 0, 4},
		{1, 65535, 9, 0},
	}
	for _, c := range cases {
		sroAt(&tm, 7, 3)
		tm.screens.active.cursor.pending_wrap = true
		sroVt(t, &h, sroCursorPosAct(c.row, c.col))
		sroExpectCursor(t, &tm, c.x, c.y)
		expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
	}
}

// Zig: test "term51: Handler.vt: cursor_pos is relative to the margins in origin mode"
func TestStreamReadonly_term51_Handler_vt_cursor_pos_is_relative_to_the_margins_in_origin_mode(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeEnableLeftAndRightMargin))
	sroVt(t, &h, sroMarginAct(StreamActionKeyLeftAndRightMargin, 4, 8))
	sroVt(t, &h, sroMarginAct(StreamActionKeyTopAndBottomMargin, 2, 4))
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeOrigin))

	cases := []struct {
		row uint16
		col uint16
		x   uint
		y   uint
	}{
		{1, 1, 3, 1},
		{0, 0, 3, 1},
		{2, 3, 5, 2},
		{3, 5, 7, 3},
		{4, 6, 7, 3},
		{65535, 65535, 7, 3},
	}
	for _, c := range cases {
		sroVt(t, &h, sroCursorPosAct(c.row, c.col))
		sroExpectCursor(t, &tm, c.x, c.y)
	}
}

// Zig: test "term51: Handler.vt: cursor_col and cursor_row keep the other coordinate"
func TestStreamReadonly_term51_Handler_vt_cursor_col_and_cursor_row_keep_the_other_coordinate(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	cols := []struct {
		v uint16
		x uint
	}{
		{0, 0},
		{1, 0},
		{4, 3},
		{10, 9},
		{11, 9},
		{65535, 9},
	}
	for _, c := range cols {
		sroAt(&tm, 5, 2)
		tm.screens.active.cursor.pending_wrap = true
		sroVt(t, &h, sroMoveAct(StreamActionKeyCursorCol, c.v))
		sroExpectCursor(t, &tm, c.x, 2)
		expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
	}
	rows := []struct {
		v uint16
		y uint
	}{
		{0, 0},
		{1, 0},
		{3, 2},
		{5, 4},
		{6, 4},
		{65535, 4},
	}
	for _, c := range rows {
		sroAt(&tm, 5, 2)
		tm.screens.active.cursor.pending_wrap = true
		sroVt(t, &h, sroMoveAct(StreamActionKeyCursorRow, c.v))
		sroExpectCursor(t, &tm, 5, c.y)
		expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
	}
}

// Zig: test "term51: Handler.vt: cursor_col_relative and cursor_row_relative, 0 does not move"
func TestStreamReadonly_term51_Handler_vt_cursor_col_relative_and_cursor_row_relative_0_does_not_move(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	cols := []struct {
		v uint16
		x uint
	}{
		{0, 5},
		{1, 6},
		{3, 8},
		{4, 9},
		{5, 9},
		{65535, 9},
	}
	for _, c := range cols {
		sroAt(&tm, 5, 2)
		tm.screens.active.cursor.pending_wrap = true
		sroVt(t, &h, sroMoveAct(StreamActionKeyCursorColRelative, c.v))
		sroExpectCursor(t, &tm, c.x, 2)
		expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
	}
	rows := []struct {
		v uint16
		y uint
	}{
		{0, 2},
		{1, 3},
		{2, 4},
		{3, 4},
		{65535, 4},
	}
	for _, c := range rows {
		sroAt(&tm, 5, 2)
		tm.screens.active.cursor.pending_wrap = true
		sroVt(t, &h, sroMoveAct(StreamActionKeyCursorRowRelative, c.v))
		sroExpectCursor(t, &tm, 5, c.y)
		expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
	}
}

// Zig: test "term51: Handler.vt: relative cursor moves are offset by the margins in origin mode (upstream quirk)"
func TestStreamReadonly_term51_Handler_vt_relative_cursor_moves_are_offset_by_the_margins_in_origin_mode_upstream_quirk(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeEnableLeftAndRightMargin))
	sroVt(t, &h, sroMarginAct(StreamActionKeyLeftAndRightMargin, 4, 8))
	sroVt(t, &h, sroMarginAct(StreamActionKeyTopAndBottomMargin, 2, 4))
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeOrigin))

	// The handler computes the absolute 1-indexed position of the cursor
	// and feeds both coordinates to setCursorPos, which adds the margin
	// offsets again (columns offset by 3, rows by 1, clamped to the margins).
	sroAt(&tm, 5, 2)
	sroVt(t, &h, sroMoveAct(StreamActionKeyCursorColRelative, 0))
	sroExpectCursor(t, &tm, 7, 3)
	sroAt(&tm, 4, 2)
	sroVt(t, &h, sroMoveAct(StreamActionKeyCursorColRelative, 0))
	sroExpectCursor(t, &tm, 7, 3)
	sroAt(&tm, 3, 2)
	sroVt(t, &h, sroMoveAct(StreamActionKeyCursorColRelative, 0))
	sroExpectCursor(t, &tm, 6, 3)
	sroAt(&tm, 5, 2)
	sroVt(t, &h, sroMoveAct(StreamActionKeyCursorRowRelative, 0))
	sroExpectCursor(t, &tm, 7, 3)
	sroAt(&tm, 5, 1)
	sroVt(t, &h, sroMoveAct(StreamActionKeyCursorRowRelative, 0))
	sroExpectCursor(t, &tm, 7, 2)

	// cursor_col and cursor_row are offset in the same way, and the
	// coordinate that is supposed to stay is offset too.
	sroAt(&tm, 5, 2)
	sroVt(t, &h, sroMoveAct(StreamActionKeyCursorCol, 1))
	sroExpectCursor(t, &tm, 3, 3)
	sroVt(t, &h, sroMoveAct(StreamActionKeyCursorRow, 1))
	sroExpectCursor(t, &tm, 6, 1)
}

// Zig: test "term51: Handler.vt: cursor_style sets the shape and the blinking mode"
func TestStreamReadonly_term51_Handler_vt_cursor_style_sets_the_shape_and_the_blinking_mode(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	cases := []struct {
		v     ANSICursorStyle
		style ScreenCursorStyle
		blink bool
	}{
		{ANSICursorStyleDefault, ScreenCursorStyleBlock, false},
		{ANSICursorStyleBlinkingBlock, ScreenCursorStyleBlock, true},
		{ANSICursorStyleSteadyBlock, ScreenCursorStyleBlock, false},
		{ANSICursorStyleBlinkingUnderline, ScreenCursorStyleUnderline, true},
		{ANSICursorStyleSteadyUnderline, ScreenCursorStyleUnderline, false},
		{ANSICursorStyleBlinkingBar, ScreenCursorStyleBar, true},
		{ANSICursorStyleSteadyBar, ScreenCursorStyleBar, false},
	}
	// Twice around so that every value is applied both from a blinking and
	// a steady state.
	twice := append(append([]struct {
		v     ANSICursorStyle
		style ScreenCursorStyle
		blink bool
	}{}, cases...), cases...)
	for _, c := range twice {
		sroVt(t, &h, StreamAction{Tag: StreamActionKeyCursorStyle, CursorStyle: c.v})
		expectEqual(t, c.style, tm.screens.active.cursor.cursor_style)
		expectEqual(t, c.blink, tm.modes.Get(ModesModeCursorBlinking))
	}
	for _, c := range cases {
		tm.modes.Set(ModesModeCursorBlinking, !c.blink)
		if c.style == ScreenCursorStyleBar {
			tm.screens.active.cursor.cursor_style = ScreenCursorStyleBlock
		} else {
			tm.screens.active.cursor.cursor_style = ScreenCursorStyleBar
		}
		sroVt(t, &h, StreamAction{Tag: StreamActionKeyCursorStyle, CursorStyle: c.v})
		expectEqual(t, c.style, tm.screens.active.cursor.cursor_style)
		expectEqual(t, c.blink, tm.modes.Get(ModesModeCursorBlinking))
	}

	// The cursor position and text are not affected.
	sroExpectCursor(t, &tm, 0, 0)
	sroExpectText(t, &tm, "")
}

// Zig: test "term51: Handler.vt: erase_display_below, above and complete"
func TestStreamReadonly_term51_Handler_vt_erase_display_below_above_and_complete(t *testing.T) {
	testStdCapacity(t)
	func() {
		tm := sroNewFilled(t)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)
		sroAt(&tm, 4, 2)
		tm.screens.active.cursor.pending_wrap = true
		sroVt(t, &h, sroBoolAct(StreamActionKeyEraseDisplayBelow, false))
		sroExpectText(t, &tm, "0123456789\nabcdefghij\nABCD")
		sroExpectCursor(t, &tm, 4, 2)
		expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
	}()
	func() {
		tm := sroNewFilled(t)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)
		sroAt(&tm, 4, 2)
		tm.screens.active.cursor.pending_wrap = true
		sroVt(t, &h, sroBoolAct(StreamActionKeyEraseDisplayAbove, false))
		sroExpectText(t, &tm, "\n\n     FGHIJ\nklmnopqrst\nKLMNOPQRST")
		sroExpectCursor(t, &tm, 4, 2)
		expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
	}()
	func() {
		tm := sroNewFilled(t)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)
		sroAt(&tm, 4, 2)
		tm.screens.active.cursor.pending_wrap = true
		tm.flags.dirty.clear = false
		sroVt(t, &h, sroBoolAct(StreamActionKeyEraseDisplayComplete, false))
		sroExpectText(t, &tm, "")
		sroExpectCursor(t, &tm, 4, 2)
		expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
		expect(t, tm.flags.dirty.clear, "t.flags.dirty.clear")
	}()
}

// Zig: test "term51: Handler.vt: erase_display_scrollback erases only the history"
func TestStreamReadonly_term51_Handler_vt_erase_display_scrollback_erases_only_the_history(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroPrintString(t, &tm, "1\n2\n3\n4\n5\n6")
	sroExpectText(t, &tm, "4\n5\n6")
	all := sroDumpScreen(t, &tm)
	// Zig-only: defer testing.allocator.free(all)
	expectEqualStrings(t, "1\n2\n3\n4\n5\n6", all)
	sroExpectCursor(t, &tm, 1, 2)

	sroVt(t, &h, sroBoolAct(StreamActionKeyEraseDisplayScrollback, false))
	sroExpectText(t, &tm, "4\n5\n6")
	sroExpectCursor(t, &tm, 1, 2)
	after := sroDumpScreen(t, &tm)
	// Zig-only: defer testing.allocator.free(after)
	expectEqualStrings(t, "4\n5\n6", after)

	// The protected flag makes no difference for the history.
	sroPrintString(t, &tm, "\n7\n8")
	sroVt(t, &h, sroBoolAct(StreamActionKeyEraseDisplayScrollback, true))
	after2 := sroDumpScreen(t, &tm)
	// Zig-only: defer testing.allocator.free(after2)
	expectEqualStrings(t, "6\n7\n8", after2)
}

// Zig: test "term51: Handler.vt: erase_display_scroll_complete moves the screen into the history and clears it"
func TestStreamReadonly_term51_Handler_vt_erase_display_scroll_complete_moves_the_screen_into_the_history_and_clears_it(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroPrintString(t, &tm, "a\nb\nc")
	tm.screens.active.cursor.pending_wrap = true
	sroVt(t, &h, sroBoolAct(StreamActionKeyEraseDisplayScrollComplete, false))
	sroExpectText(t, &tm, "")
	expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
	all := sroDumpScreen(t, &tm)
	// Zig-only: defer testing.allocator.free(all)
	expectEqualStrings(t, "a\nb\nc", all)
}

// sroProtected is term51DProtected: 10x3 terminal with "AB" protected and
// "CD" not on row 0, "EF" protected and "GH" not on row 1, using the given
// protection mode (the key of a protected_mode_* action).
func sroProtected(t *testing.T, h *StreamReadonlyHandler, key StreamActionKey) {
	t.Helper()
	tm := h.terminal
	sroVt(t, h, sroAct(key))
	sroPrintString(t, tm, "AB")
	sroVt(t, h, sroAct(StreamActionKeyProtectedModeOff))
	sroPrintString(t, tm, "CD")
	sroPrintString(t, tm, "\n")
	tm.CarriageReturn()
	sroVt(t, h, sroAct(key))
	sroPrintString(t, tm, "EF")
	sroVt(t, h, sroAct(StreamActionKeyProtectedModeOff))
	sroPrintString(t, tm, "GH")
	sroAt(tm, 0, 0)
}

// Zig: test "term51: Handler.vt: erase_display_* with the protected flag keep protected cells"
func TestStreamReadonly_term51_Handler_vt_erase_display_with_the_protected_flag_keep_protected_cells(t *testing.T) {
	testStdCapacity(t)
	// complete, below and above all respect protection when asked to.
	func() {
		tm := sroNew(t, 10, 3)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)
		sroProtected(t, &h, StreamActionKeyProtectedModeDec)
		sroExpectText(t, &tm, "ABCD\nEFGH")
		sroVt(t, &h, sroBoolAct(StreamActionKeyEraseDisplayComplete, true))
		sroExpectText(t, &tm, "AB\nEF")
	}()
	func() {
		tm := sroNew(t, 10, 3)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)
		sroProtected(t, &h, StreamActionKeyProtectedModeDec)
		sroVt(t, &h, sroBoolAct(StreamActionKeyEraseDisplayBelow, true))
		sroExpectText(t, &tm, "AB\nEF")
	}()
	func() {
		tm := sroNew(t, 10, 3)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)
		sroProtected(t, &h, StreamActionKeyProtectedModeDec)
		sroAt(&tm, 0, 2)
		sroVt(t, &h, sroBoolAct(StreamActionKeyEraseDisplayAbove, true))
		sroExpectText(t, &tm, "AB\nEF")
	}()
	// Without the flag, DEC protection is ignored.
	func() {
		tm := sroNew(t, 10, 3)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)
		sroProtected(t, &h, StreamActionKeyProtectedModeDec)
		sroVt(t, &h, sroBoolAct(StreamActionKeyEraseDisplayComplete, false))
		sroExpectText(t, &tm, "")
	}()
	func() {
		tm := sroNew(t, 10, 3)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)
		sroProtected(t, &h, StreamActionKeyProtectedModeDec)
		sroVt(t, &h, sroBoolAct(StreamActionKeyEraseDisplayBelow, false))
		sroExpectText(t, &tm, "")
	}()
	// ISO protection is respected even without the flag, and stays that
	// way after protected_mode_off.
	func() {
		tm := sroNew(t, 10, 3)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)
		sroProtected(t, &h, StreamActionKeyProtectedModeIso)
		sroVt(t, &h, sroBoolAct(StreamActionKeyEraseDisplayComplete, false))
		sroExpectText(t, &tm, "AB\nEF")
	}()
}

// Zig: test "term51: Handler.vt: erase_line_right, left and complete"
func TestStreamReadonly_term51_Handler_vt_erase_line_right_left_and_complete(t *testing.T) {
	testStdCapacity(t)
	func() {
		tm := sroNewFilled(t)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)
		sroAt(&tm, 4, 1)
		tm.screens.active.cursor.pending_wrap = true
		sroVt(t, &h, sroBoolAct(StreamActionKeyEraseLineRight, false))
		sroExpectText(t, &tm, "0123456789\nabcd\nABCDEFGHIJ\nklmnopqrst\nKLMNOPQRST")
		sroExpectCursor(t, &tm, 4, 1)
		expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
	}()
	func() {
		tm := sroNewFilled(t)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)
		sroAt(&tm, 4, 1)
		tm.screens.active.cursor.pending_wrap = true
		sroVt(t, &h, sroBoolAct(StreamActionKeyEraseLineLeft, false))
		sroExpectText(t, &tm, "0123456789\n     fghij\nABCDEFGHIJ\nklmnopqrst\nKLMNOPQRST")
		sroExpectCursor(t, &tm, 4, 1)
		expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
	}()
	func() {
		tm := sroNewFilled(t)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)
		sroAt(&tm, 4, 1)
		tm.screens.active.cursor.pending_wrap = true
		sroVt(t, &h, sroBoolAct(StreamActionKeyEraseLineComplete, false))
		sroExpectText(t, &tm, "0123456789\n\nABCDEFGHIJ\nklmnopqrst\nKLMNOPQRST")
		sroExpectCursor(t, &tm, 4, 1)
		expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
	}()
}

// Zig: test "term51: Handler.vt: erase_line_* with the protected flag keep protected cells"
func TestStreamReadonly_term51_Handler_vt_erase_line_with_the_protected_flag_keep_protected_cells(t *testing.T) {
	testStdCapacity(t)
	func() {
		tm := sroNew(t, 10, 3)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)
		sroProtected(t, &h, StreamActionKeyProtectedModeDec)
		sroVt(t, &h, sroBoolAct(StreamActionKeyEraseLineRight, true))
		sroExpectText(t, &tm, "AB\nEFGH")
	}()
	func() {
		tm := sroNew(t, 10, 3)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)
		sroProtected(t, &h, StreamActionKeyProtectedModeDec)
		sroAt(&tm, 3, 0)
		sroVt(t, &h, sroBoolAct(StreamActionKeyEraseLineLeft, true))
		sroExpectText(t, &tm, "AB\nEFGH")
	}()
	func() {
		tm := sroNew(t, 10, 3)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)
		sroProtected(t, &h, StreamActionKeyProtectedModeDec)
		sroVt(t, &h, sroBoolAct(StreamActionKeyEraseLineComplete, true))
		sroExpectText(t, &tm, "AB\nEFGH")
	}()
	func() {
		tm := sroNew(t, 10, 3)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)
		sroProtected(t, &h, StreamActionKeyProtectedModeDec)
		sroVt(t, &h, sroBoolAct(StreamActionKeyEraseLineRight, false))
		sroVt(t, &h, sroBoolAct(StreamActionKeyEraseLineComplete, false))
		sroExpectText(t, &tm, "\nEFGH")
	}()
	func() {
		tm := sroNew(t, 10, 3)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)
		sroProtected(t, &h, StreamActionKeyProtectedModeIso)
		sroVt(t, &h, sroBoolAct(StreamActionKeyEraseLineComplete, false))
		sroExpectText(t, &tm, "AB\nEFGH")
	}()
}

// Zig: test "term51: Handler.vt: protected_mode_off, iso and dec"
func TestStreamReadonly_term51_Handler_vt_protected_mode_off_iso_and_dec(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	expect(t, !tm.screens.active.cursor.protected, "!t.screens.active.cursor.protected")
	expectEqual(t, ANSIProtectedModeOff, tm.screens.active.protected_mode)
	// Off from the initial state changes nothing.
	sroVt(t, &h, sroAct(StreamActionKeyProtectedModeOff))
	expect(t, !tm.screens.active.cursor.protected, "!t.screens.active.cursor.protected")
	expectEqual(t, ANSIProtectedModeOff, tm.screens.active.protected_mode)

	sroVt(t, &h, sroAct(StreamActionKeyProtectedModeIso))
	expect(t, tm.screens.active.cursor.protected, "t.screens.active.cursor.protected")
	expectEqual(t, ANSIProtectedModeIso, tm.screens.active.protected_mode)
	// The last mode stays after switching protection off.
	sroVt(t, &h, sroAct(StreamActionKeyProtectedModeOff))
	expect(t, !tm.screens.active.cursor.protected, "!t.screens.active.cursor.protected")
	expectEqual(t, ANSIProtectedModeIso, tm.screens.active.protected_mode)

	sroVt(t, &h, sroAct(StreamActionKeyProtectedModeDec))
	expect(t, tm.screens.active.cursor.protected, "t.screens.active.cursor.protected")
	expectEqual(t, ANSIProtectedModeDec, tm.screens.active.protected_mode)
	sroVt(t, &h, sroAct(StreamActionKeyProtectedModeOff))
	expect(t, !tm.screens.active.cursor.protected, "!t.screens.active.cursor.protected")
	expectEqual(t, ANSIProtectedModeDec, tm.screens.active.protected_mode)

	// Printed cells carry the protection of the cursor.
	sroVt(t, &h, sroAct(StreamActionKeyProtectedModeDec))
	sroVt(t, &h, sroPrintAct('P'))
	sroVt(t, &h, sroAct(StreamActionKeyProtectedModeOff))
	sroVt(t, &h, sroPrintAct('U'))
	p := sroGetCell(t, &tm, 0, 0)
	u := sroGetCell(t, &tm, 1, 0)
	expect(t, p.cell.protected, "p.cell.protected")
	expect(t, !u.cell.protected, "!u.cell.protected")
}

// Zig: test "term51: Handler.vt: delete_chars shifts the rest of the line left"
func TestStreamReadonly_term51_Handler_vt_delete_chars_shifts_the_rest_of_the_line_left(t *testing.T) {
	testStdCapacity(t)
	const rest = "\nabcdefghij\nABCDEFGHIJ\nklmnopqrst\nKLMNOPQRST"
	cases := []struct {
		n   uint
		row string
	}{
		{0, "0123456789"},
		{1, "013456789"},
		{3, "0156789"},
		{8, "01"},
		{9, "01"},
		{65535, "01"},
	}
	for _, c := range cases {
		func() {
			tm := sroNewFilled(t)
			defer tm.Deinit()
			h := StreamReadonlyHandlerInit(&tm)
			sroAt(&tm, 2, 0)
			sroVt(t, &h, sroCountAct(StreamActionKeyDeleteChars, c.n))
			expected := c.row + rest
			sroExpectText(t, &tm, expected)
			sroExpectCursor(t, &tm, 2, 0)
		}()
	}
}

// Zig: test "term51: Handler.vt: delete_chars respects the left and right margins"
func TestStreamReadonly_term51_Handler_vt_delete_chars_respects_the_left_and_right_margins(t *testing.T) {
	testStdCapacity(t)
	const rest = "\nabcdefghij\nABCDEFGHIJ\nklmnopqrst\nKLMNOPQRST"
	// Margins are columns 3..7. Inside them only that part shifts, outside
	// nothing happens.
	for _, c := range []struct {
		x   SizeCellCountInt
		row string
	}{
		{4, "0123567 89"},
		{3, "0124567 89"},
		{7, "0123456 89"},
		{2, "0123456789"},
		{8, "0123456789"},
	} {
		func() {
			tm := sroNewFilled(t)
			defer tm.Deinit()
			h := StreamReadonlyHandlerInit(&tm)
			sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeEnableLeftAndRightMargin))
			sroVt(t, &h, sroMarginAct(StreamActionKeyLeftAndRightMargin, 4, 8))
			sroAt(&tm, c.x, 0)
			sroVt(t, &h, sroCountAct(StreamActionKeyDeleteChars, 1))
			sroExpectText(t, &tm, c.row+rest)
			sroExpectCursor(t, &tm, uint(c.x), 0)
		}()
	}
}

// Zig: test "term51: Handler.vt: delete_chars at the last column with pending wrap"
func TestStreamReadonly_term51_Handler_vt_delete_chars_at_the_last_column_with_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewFilled(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroAt(&tm, 9, 0)
	tm.screens.active.cursor.pending_wrap = true
	sroVt(t, &h, sroCountAct(StreamActionKeyDeleteChars, 1))
	sroExpectText(t, &tm, "012345678\nabcdefghij\nABCDEFGHIJ\nklmnopqrst\nKLMNOPQRST")
	sroExpectCursor(t, &tm, 9, 0)
	// A delete that happens clears the pending wrap state.
	expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")

	// A count of 0 does nothing, including leaving the flag alone.
	tm.screens.active.cursor.pending_wrap = true
	sroVt(t, &h, sroCountAct(StreamActionKeyDeleteChars, 0))
	expect(t, tm.screens.active.cursor.pending_wrap, "t.screens.active.cursor.pending_wrap")
	sroExpectText(t, &tm, "012345678\nabcdefghij\nABCDEFGHIJ\nklmnopqrst\nKLMNOPQRST")

	// So does a cursor outside the margins.
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeEnableLeftAndRightMargin))
	sroVt(t, &h, sroMarginAct(StreamActionKeyLeftAndRightMargin, 2, 5))
	sroAt(&tm, 9, 0)
	tm.screens.active.cursor.pending_wrap = true
	sroVt(t, &h, sroCountAct(StreamActionKeyDeleteChars, 1))
	expect(t, tm.screens.active.cursor.pending_wrap, "t.screens.active.cursor.pending_wrap")
}

// Zig: test "term51: Handler.vt: erase_chars blanks cells without shifting, 0 acts as 1"
func TestStreamReadonly_term51_Handler_vt_erase_chars_blanks_cells_without_shifting_0_acts_as_1(t *testing.T) {
	testStdCapacity(t)
	const rest = "\nabcdefghij\nABCDEFGHIJ\nklmnopqrst\nKLMNOPQRST"
	cases := []struct {
		n   uint
		row string
	}{
		{0, "01 3456789"},
		{1, "01 3456789"},
		{3, "01   56789"},
		{8, "01"},
		{9, "01"},
		{65535, "01"},
	}
	for _, c := range cases {
		func() {
			tm := sroNewFilled(t)
			defer tm.Deinit()
			h := StreamReadonlyHandlerInit(&tm)
			sroAt(&tm, 2, 0)
			tm.screens.active.cursor.pending_wrap = true
			sroVt(t, &h, sroCountAct(StreamActionKeyEraseChars, c.n))
			expected := c.row + rest
			sroExpectText(t, &tm, expected)
			sroExpectCursor(t, &tm, 2, 0)
		}()
	}
}

// Zig: test "term51: Handler.vt: erase_chars ignores DEC protection but respects ISO protection"
func TestStreamReadonly_term51_Handler_vt_erase_chars_ignores_DEC_protection_but_respects_ISO_protection(t *testing.T) {
	testStdCapacity(t)
	func() {
		tm := sroNew(t, 10, 3)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)
		sroProtected(t, &h, StreamActionKeyProtectedModeDec)
		sroVt(t, &h, sroCountAct(StreamActionKeyEraseChars, 4))
		sroExpectText(t, &tm, "\nEFGH")
	}()
	func() {
		tm := sroNew(t, 10, 3)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)
		sroProtected(t, &h, StreamActionKeyProtectedModeIso)
		sroVt(t, &h, sroCountAct(StreamActionKeyEraseChars, 4))
		sroExpectText(t, &tm, "AB\nEFGH")
	}()
}

// Zig: test "term51: Handler.vt: insert_blanks shifts the rest of the line right"
func TestStreamReadonly_term51_Handler_vt_insert_blanks_shifts_the_rest_of_the_line_right(t *testing.T) {
	testStdCapacity(t)
	const rest = "\nabcdefghij\nABCDEFGHIJ\nklmnopqrst\nKLMNOPQRST"
	cases := []struct {
		n   uint
		row string
	}{
		{0, "0123456789"},
		{1, "01 2345678"},
		{3, "01   23456"},
		{8, "01"},
		{9, "01"},
		{65535, "01"},
	}
	for _, c := range cases {
		func() {
			tm := sroNewFilled(t)
			defer tm.Deinit()
			h := StreamReadonlyHandlerInit(&tm)
			sroAt(&tm, 2, 0)
			tm.screens.active.cursor.pending_wrap = true
			sroVt(t, &h, sroCountAct(StreamActionKeyInsertBlanks, c.n))
			expected := c.row + rest
			sroExpectText(t, &tm, expected)
			sroExpectCursor(t, &tm, 2, 0)
			// Always clears pending wrap, even for a count of 0.
			expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
		}()
	}
}

// Zig: test "term51: Handler.vt: insert_blanks respects margins and clears pending wrap outside them"
func TestStreamReadonly_term51_Handler_vt_insert_blanks_respects_margins_and_clears_pending_wrap_outside_them(t *testing.T) {
	testStdCapacity(t)
	const rest = "\nabcdefghij\nABCDEFGHIJ\nklmnopqrst\nKLMNOPQRST"
	// Margins are columns 3..7.
	for _, c := range []struct {
		x   SizeCellCountInt
		row string
	}{
		{4, "0123 45689"},
		{3, "012 345689"},
		{7, "0123456 89"},
		{2, "0123456789"},
		{8, "0123456789"},
	} {
		func() {
			tm := sroNewFilled(t)
			defer tm.Deinit()
			h := StreamReadonlyHandlerInit(&tm)
			sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeEnableLeftAndRightMargin))
			sroVt(t, &h, sroMarginAct(StreamActionKeyLeftAndRightMargin, 4, 8))
			sroAt(&tm, c.x, 0)
			tm.screens.active.cursor.pending_wrap = true
			sroVt(t, &h, sroCountAct(StreamActionKeyInsertBlanks, 1))
			sroExpectText(t, &tm, c.row+rest)
			sroExpectCursor(t, &tm, uint(c.x), 0)
			expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
		}()
	}
}

// Zig: test "term51: Handler.vt: insert_lines pushes lines down and moves the cursor to the left margin"
func TestStreamReadonly_term51_Handler_vt_insert_lines_pushes_lines_down_and_moves_the_cursor_to_the_left_margin(t *testing.T) {
	testStdCapacity(t)
	cases := []struct {
		n    uint
		text string
		x    uint
	}{
		{0, sroFilled, 3},
		{1, "0123456789\n\nabcdefghij\nABCDEFGHIJ\nklmnopqrst", 0},
		{3, "0123456789\n\n\n\nabcdefghij", 0},
		{4, "0123456789", 0},
		{5, "0123456789", 0},
		{65535, "0123456789", 0},
	}
	for _, c := range cases {
		func() {
			tm := sroNewFilled(t)
			defer tm.Deinit()
			h := StreamReadonlyHandlerInit(&tm)
			sroAt(&tm, 3, 1)
			sroVt(t, &h, sroCountAct(StreamActionKeyInsertLines, c.n))
			sroExpectText(t, &tm, c.text)
			sroExpectCursor(t, &tm, c.x, 1)
		}()
	}
}

// Zig: test "term51: Handler.vt: insert_lines within a scroll region, outside it nothing happens"
func TestStreamReadonly_term51_Handler_vt_insert_lines_within_a_scroll_region_outside_it_nothing_happens(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewFilled(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroVt(t, &h, sroMarginAct(StreamActionKeyTopAndBottomMargin, 2, 4))
	sroAt(&tm, 3, 1)
	tm.screens.active.cursor.pending_wrap = true
	sroVt(t, &h, sroCountAct(StreamActionKeyInsertLines, 1))
	sroExpectText(t, &tm, "0123456789\n\nabcdefghij\nABCDEFGHIJ\nKLMNOPQRST")
	sroExpectCursor(t, &tm, 0, 1)
	expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")

	// Above and below the region: ignored, cursor untouched.
	before := sroSnapshot(t, &tm)
	sroAt(&tm, 3, 0)
	sroVt(t, &h, sroCountAct(StreamActionKeyInsertLines, 1))
	sroAt(&tm, 3, 4)
	sroVt(t, &h, sroCountAct(StreamActionKeyInsertLines, 1))
	sroAt(&tm, 0, 1)
	expectEqual(t, before, sroSnapshot(t, &tm))
}

// Zig: test "term51: Handler.vt: delete_lines pulls lines up and moves the cursor to the left margin"
func TestStreamReadonly_term51_Handler_vt_delete_lines_pulls_lines_up_and_moves_the_cursor_to_the_left_margin(t *testing.T) {
	testStdCapacity(t)
	cases := []struct {
		n    uint
		text string
		x    uint
	}{
		{0, sroFilled, 3},
		{1, "0123456789\nABCDEFGHIJ\nklmnopqrst\nKLMNOPQRST", 0},
		{3, "0123456789\nKLMNOPQRST", 0},
		{4, "0123456789", 0},
		{5, "0123456789", 0},
		{65535, "0123456789", 0},
	}
	for _, c := range cases {
		func() {
			tm := sroNewFilled(t)
			defer tm.Deinit()
			h := StreamReadonlyHandlerInit(&tm)
			sroAt(&tm, 3, 1)
			sroVt(t, &h, sroCountAct(StreamActionKeyDeleteLines, c.n))
			sroExpectText(t, &tm, c.text)
			sroExpectCursor(t, &tm, c.x, 1)
		}()
	}
}

// Zig: test "term51: Handler.vt: delete_lines within a scroll region, outside it nothing happens"
func TestStreamReadonly_term51_Handler_vt_delete_lines_within_a_scroll_region_outside_it_nothing_happens(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewFilled(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroVt(t, &h, sroMarginAct(StreamActionKeyTopAndBottomMargin, 2, 4))
	sroAt(&tm, 3, 1)
	tm.screens.active.cursor.pending_wrap = true
	sroVt(t, &h, sroCountAct(StreamActionKeyDeleteLines, 1))
	sroExpectText(t, &tm, "0123456789\nABCDEFGHIJ\nklmnopqrst\n\nKLMNOPQRST")
	sroExpectCursor(t, &tm, 0, 1)
	expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")

	before := sroSnapshot(t, &tm)
	sroAt(&tm, 3, 0)
	sroVt(t, &h, sroCountAct(StreamActionKeyDeleteLines, 1))
	sroAt(&tm, 3, 4)
	sroVt(t, &h, sroCountAct(StreamActionKeyDeleteLines, 1))
	sroAt(&tm, 0, 1)
	expectEqual(t, before, sroSnapshot(t, &tm))
}

// Zig: test "term51: Handler.vt: scroll_up moves the top line into the scrollback"
func TestStreamReadonly_term51_Handler_vt_scroll_up_moves_the_top_line_into_the_scrollback(t *testing.T) {
	testStdCapacity(t)
	cases := []struct {
		n    uint
		text string
	}{
		{0, sroFilled},
		{1, "abcdefghij\nABCDEFGHIJ\nklmnopqrst\nKLMNOPQRST"},
		{4, "KLMNOPQRST"},
		{5, ""},
		{6, ""},
		{65535, ""},
	}
	for _, c := range cases {
		func() {
			tm := sroNewFilled(t)
			defer tm.Deinit()
			h := StreamReadonlyHandlerInit(&tm)
			sroAt(&tm, 3, 2)
			tm.screens.active.cursor.pending_wrap = true
			sroVt(t, &h, sroCountAct(StreamActionKeyScrollUp, c.n))
			sroExpectText(t, &tm, c.text)
			// The cursor does not move, nor does the pending wrap state change.
			sroExpectCursor(t, &tm, 3, 2)
			expect(t, tm.screens.active.cursor.pending_wrap, "t.screens.active.cursor.pending_wrap")
			// Everything that scrolled out is in the history.
			all := sroDumpScreen(t, &tm)
			// Zig-only: defer testing.allocator.free(all)
			expectEqualStrings(t, sroFilled, strings.TrimRight(all, "\n "))
		}()
	}
}

// Zig: test "term51: Handler.vt: scroll_up within a scroll region does not create scrollback"
func TestStreamReadonly_term51_Handler_vt_scroll_up_within_a_scroll_region_does_not_create_scrollback(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewFilled(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroVt(t, &h, sroMarginAct(StreamActionKeyTopAndBottomMargin, 2, 4))
	sroAt(&tm, 3, 2)
	sroVt(t, &h, sroCountAct(StreamActionKeyScrollUp, 1))
	sroExpectText(t, &tm, "0123456789\nABCDEFGHIJ\nklmnopqrst\n\nKLMNOPQRST")
	sroExpectCursor(t, &tm, 3, 2)
	sroVt(t, &h, sroCountAct(StreamActionKeyScrollUp, 0))
	sroExpectText(t, &tm, "0123456789\nABCDEFGHIJ\nklmnopqrst\n\nKLMNOPQRST")
	sroVt(t, &h, sroCountAct(StreamActionKeyScrollUp, 65535))
	sroExpectText(t, &tm, "0123456789\n\n\n\nKLMNOPQRST")
	sroExpectCursor(t, &tm, 3, 2)
	all := sroDumpScreen(t, &tm)
	// Zig-only: defer testing.allocator.free(all)
	expectEqualStrings(t, "0123456789\n\n\n\nKLMNOPQRST", all)
}

// Zig: test "term51: Handler.vt: scroll_down moves lines down and keeps the cursor"
func TestStreamReadonly_term51_Handler_vt_scroll_down_moves_lines_down_and_keeps_the_cursor(t *testing.T) {
	testStdCapacity(t)
	cases := []struct {
		n    uint
		text string
	}{
		{0, sroFilled},
		{1, "\n0123456789\nabcdefghij\nABCDEFGHIJ\nklmnopqrst"},
		{4, "\n\n\n\n0123456789"},
		{5, ""},
		{6, ""},
		{65535, ""},
	}
	for _, c := range cases {
		func() {
			tm := sroNewFilled(t)
			defer tm.Deinit()
			h := StreamReadonlyHandlerInit(&tm)
			sroAt(&tm, 3, 2)
			tm.screens.active.cursor.pending_wrap = true
			sroVt(t, &h, sroCountAct(StreamActionKeyScrollDown, c.n))
			sroExpectText(t, &tm, c.text)
			sroExpectCursor(t, &tm, 3, 2)
			expect(t, tm.screens.active.cursor.pending_wrap, "t.screens.active.cursor.pending_wrap")
		}()
	}
}

// Zig: test "term51: Handler.vt: scroll_down within a scroll region"
func TestStreamReadonly_term51_Handler_vt_scroll_down_within_a_scroll_region(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewFilled(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroVt(t, &h, sroMarginAct(StreamActionKeyTopAndBottomMargin, 2, 4))
	sroAt(&tm, 3, 0)
	sroVt(t, &h, sroCountAct(StreamActionKeyScrollDown, 1))
	sroExpectText(t, &tm, "0123456789\n\nabcdefghij\nABCDEFGHIJ\nKLMNOPQRST")
	sroExpectCursor(t, &tm, 3, 0)
	sroVt(t, &h, sroCountAct(StreamActionKeyScrollDown, 65535))
	sroExpectText(t, &tm, "0123456789\n\n\n\nKLMNOPQRST")
}

// Zig: test "term51: Handler.vt: top_and_bottom_margin accepts only top < bottom and homes the cursor"
func TestStreamReadonly_term51_Handler_vt_top_and_bottom_margin_accepts_only_top_bottom_and_homes_the_cursor(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	type Case struct {
		top    uint16
		bottom uint16
		ok     bool
		t      SizeCellCountInt
		b      SizeCellCountInt
	}
	cases := []Case{
		{top: 2, bottom: 4, ok: true, t: 1, b: 3},
		{top: 0, bottom: 0, ok: true, t: 0, b: 4},
		{top: 1, bottom: 65535, ok: true, t: 0, b: 4},
		{top: 1, bottom: 5, ok: true, t: 0, b: 4},
		{top: 1, bottom: 6, ok: true, t: 0, b: 4},
		{top: 2, bottom: 0, ok: true, t: 1, b: 4},
		{top: 4, bottom: 5, ok: true, t: 3, b: 4},
		{top: 1, bottom: 2, ok: true, t: 0, b: 1},
		{top: 0, bottom: 3, ok: true, t: 0, b: 2},
		{top: 3, bottom: 3, ok: false, t: 0, b: 0},
		{top: 4, bottom: 2, ok: false, t: 0, b: 0},
		{top: 5, bottom: 5, ok: false, t: 0, b: 0},
		{top: 5, bottom: 0, ok: false, t: 0, b: 0},
		{top: 6, bottom: 0, ok: false, t: 0, b: 0},
		{top: 65535, bottom: 65535, ok: false, t: 0, b: 0},
		{top: 1, bottom: 1, ok: false, t: 0, b: 0},
	}
	// Establish a known region first for the rejected cases.
	for _, c := range cases {
		sroVt(t, &h, sroMarginAct(StreamActionKeyTopAndBottomMargin, 2, 4))
		sroAt(&tm, 6, 3)
		tm.screens.active.cursor.pending_wrap = true
		sroVt(t, &h, sroMarginAct(StreamActionKeyTopAndBottomMargin, c.top, c.bottom))
		if c.ok {
			expectEqual(t, c.t, tm.scrolling_region.top)
			expectEqual(t, c.b, tm.scrolling_region.bottom)
			sroExpectCursor(t, &tm, 0, 0)
			expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
		} else {
			expectEqual(t, 1, tm.scrolling_region.top)
			expectEqual(t, 3, tm.scrolling_region.bottom)
			sroExpectCursor(t, &tm, 6, 3)
			expect(t, tm.screens.active.cursor.pending_wrap, "t.screens.active.cursor.pending_wrap")
		}
		// Margins never change the left and right.
		expectEqual(t, 0, tm.scrolling_region.left)
		expectEqual(t, 9, tm.scrolling_region.right)
	}
}

// Zig: test "term51: Handler.vt: top_and_bottom_margin homes to the region corner in origin mode"
func TestStreamReadonly_term51_Handler_vt_top_and_bottom_margin_homes_to_the_region_corner_in_origin_mode(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	tm.modes.Set(ModesModeOrigin, true)
	sroVt(t, &h, sroMarginAct(StreamActionKeyTopAndBottomMargin, 3, 5))
	sroExpectCursor(t, &tm, 0, 2)
}

// Zig: test "term51: Handler.vt: left_and_right_margin needs mode 69 and accepts only left < right"
func TestStreamReadonly_term51_Handler_vt_left_and_right_margin_needs_mode_69_and_accepts_only_left_right(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	// Without the mode it is ignored.
	sroAt(&tm, 6, 3)
	sroVt(t, &h, sroMarginAct(StreamActionKeyLeftAndRightMargin, 2, 5))
	expectEqual(t, 0, tm.scrolling_region.left)
	expectEqual(t, 9, tm.scrolling_region.right)
	sroExpectCursor(t, &tm, 6, 3)

	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeEnableLeftAndRightMargin))
	type Case struct {
		l     uint16
		r     uint16
		ok    bool
		left  SizeCellCountInt
		right SizeCellCountInt
	}
	cases := []Case{
		{l: 4, r: 8, ok: true, left: 3, right: 7},
		{l: 0, r: 0, ok: true, left: 0, right: 9},
		{l: 1, r: 65535, ok: true, left: 0, right: 9},
		{l: 1, r: 10, ok: true, left: 0, right: 9},
		{l: 1, r: 11, ok: true, left: 0, right: 9},
		{l: 3, r: 0, ok: true, left: 2, right: 9},
		{l: 9, r: 10, ok: true, left: 8, right: 9},
		{l: 0, r: 2, ok: true, left: 0, right: 1},
		{l: 5, r: 5, ok: false, left: 0, right: 0},
		{l: 6, r: 5, ok: false, left: 0, right: 0},
		{l: 10, r: 10, ok: false, left: 0, right: 0},
		{l: 10, r: 0, ok: false, left: 0, right: 0},
		{l: 11, r: 0, ok: false, left: 0, right: 0},
		{l: 65535, r: 65535, ok: false, left: 0, right: 0},
	}
	for _, c := range cases {
		sroVt(t, &h, sroMarginAct(StreamActionKeyLeftAndRightMargin, 3, 7))
		expectEqual(t, 2, tm.scrolling_region.left)
		sroAt(&tm, 6, 3)
		tm.screens.active.cursor.pending_wrap = true
		sroVt(t, &h, sroMarginAct(StreamActionKeyLeftAndRightMargin, c.l, c.r))
		if c.ok {
			expectEqual(t, c.left, tm.scrolling_region.left)
			expectEqual(t, c.right, tm.scrolling_region.right)
			sroExpectCursor(t, &tm, 0, 0)
			expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
		} else {
			expectEqual(t, 2, tm.scrolling_region.left)
			expectEqual(t, 6, tm.scrolling_region.right)
			sroExpectCursor(t, &tm, 6, 3)
			expect(t, tm.screens.active.cursor.pending_wrap, "t.screens.active.cursor.pending_wrap")
		}
		// Top and bottom are not touched.
		expectEqual(t, 0, tm.scrolling_region.top)
		expectEqual(t, 4, tm.scrolling_region.bottom)
	}
}

// Zig: test "term51: Handler.vt: left_and_right_margin_ambiguous resets margins with mode 69, saves the cursor without"
func TestStreamReadonly_term51_Handler_vt_left_and_right_margin_ambiguous_resets_margins_with_mode_69_saves_the_cursor_without(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	// Without the mode it is DECSC.
	sroAt(&tm, 4, 2)
	expect(t, !tm.screens.active.saved_cursor.Valid, "t.screens.active.saved_cursor == null")
	sroVt(t, &h, sroAct(StreamActionKeyLeftAndRightMarginAmbiguous))
	sroExpectCursor(t, &tm, 4, 2)
	saved := zigUnwrap(t, tm.screens.active.saved_cursor)
	expectEqual(t, 4, saved.x)
	expectEqual(t, 2, saved.y)
	expectEqual(t, 0, tm.scrolling_region.left)
	expectEqual(t, 9, tm.scrolling_region.right)

	// With the mode it resets the margins and homes the cursor.
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeEnableLeftAndRightMargin))
	sroVt(t, &h, sroMarginAct(StreamActionKeyLeftAndRightMargin, 3, 7))
	expectEqual(t, 2, tm.scrolling_region.left)
	sroAt(&tm, 6, 3)
	sroVt(t, &h, sroAct(StreamActionKeyLeftAndRightMarginAmbiguous))
	expectEqual(t, 0, tm.scrolling_region.left)
	expectEqual(t, 9, tm.scrolling_region.right)
	sroExpectCursor(t, &tm, 0, 0)
	// The saved cursor was not overwritten.
	expectEqual(t, 4, zigUnwrap(t, tm.screens.active.saved_cursor).x)
}

// Zig: test "term51: Handler.vt: save_cursor and restore_cursor keep position, style and flags"
func TestStreamReadonly_term51_Handler_vt_save_cursor_and_restore_cursor_keep_position_style_and_flags(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	// Restoring without a save gives the defaults.
	sroAt(&tm, 5, 3)
	sroVt(t, &h, sroSGRAct(SGRAttributeTagBold))
	tm.screens.active.cursor.pending_wrap = true
	sroVt(t, &h, sroAct(StreamActionKeyProtectedModeDec))
	sroVt(t, &h, sroAct(StreamActionKeyRestoreCursor))
	sroExpectCursor(t, &tm, 0, 0)
	expect(t, !tm.screens.active.cursor.pending_wrap, "!t.screens.active.cursor.pending_wrap")
	expect(t, !tm.screens.active.cursor.protected, "!t.screens.active.cursor.protected")
	expect(t, !tm.screens.active.cursor.style.flags.bold, "!t.screens.active.cursor.style.flags.bold")

	// Save a rich state.
	sroAt(&tm, 4, 2)
	sroVt(t, &h, sroSGRAct(SGRAttributeTagBold))
	sroVt(t, &h, StreamAction{
		Tag:          StreamActionKeySetAttribute,
		SetAttribute: SGRAttribute{Tag: SGRAttributeTag8Fg, X8Fg: ColorNameRed},
	})
	tm.screens.active.cursor.pending_wrap = true
	sroVt(t, &h, sroAct(StreamActionKeyProtectedModeDec))
	tm.modes.Set(ModesModeOrigin, true)
	sroVt(t, &h, sroConfigureCharsetAct(CharsetsSlotsG0, CharsetsCharsetDecSpecial))
	sroVt(t, &h, sroInvokeCharsetAct(CharsetsActiveSlotGL, CharsetsSlotsG1, false))
	sroVt(t, &h, sroAct(StreamActionKeySaveCursor))

	// Change everything.
	tm.modes.Set(ModesModeOrigin, false)
	sroAt(&tm, 0, 0)
	tm.screens.active.cursor.pending_wrap = false
	sroVt(t, &h, sroSGRAct(SGRAttributeTagUnset))
	sroVt(t, &h, sroAct(StreamActionKeyProtectedModeOff))
	sroVt(t, &h, sroConfigureCharsetAct(CharsetsSlotsG0, CharsetsCharsetAscii))
	sroVt(t, &h, sroInvokeCharsetAct(CharsetsActiveSlotGL, CharsetsSlotsG0, false))

	sroVt(t, &h, sroAct(StreamActionKeyRestoreCursor))
	sroExpectCursor(t, &tm, 4, 2)
	expect(t, tm.screens.active.cursor.pending_wrap, "t.screens.active.cursor.pending_wrap")
	expect(t, tm.screens.active.cursor.protected, "t.screens.active.cursor.protected")
	expect(t, tm.modes.Get(ModesModeOrigin), "t.modes.get(.origin)")
	expect(t, tm.screens.active.cursor.style.flags.bold, "t.screens.active.cursor.style.flags.bold")
	expectEqual(t, TerminalColorsDefault.palette.current[0], tm.colors.palette.current[0])
	expectEqual(t, StyleColorPalette, tm.screens.active.cursor.style.fg_color.Tag)
	expectEqual(t, 1, tm.screens.active.cursor.style.fg_color.Palette)
	expectEqual(t, sroCharsetDec, tm.screens.active.charset.charsets.Get(CharsetsSlotsG0))
	expectEqual(t, CharsetsSlotsG1, tm.screens.active.charset.gl)

	// The saved state stays available.
	sroAt(&tm, 8, 4)
	sroVt(t, &h, sroAct(StreamActionKeyRestoreCursor))
	sroExpectCursor(t, &tm, 4, 2)

	// Saving again replaces it.
	sroAt(&tm, 7, 1)
	sroVt(t, &h, sroAct(StreamActionKeySaveCursor))
	sroAt(&tm, 1, 1)
	sroVt(t, &h, sroAct(StreamActionKeyRestoreCursor))
	sroExpectCursor(t, &tm, 7, 1)
}

// Zig: test "term51: Handler.vt: configure_charset sets every slot to every charset"
func TestStreamReadonly_term51_Handler_vt_configure_charset_sets_every_slot_to_every_charset(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	slots := []CharsetsSlots{CharsetsSlotsG0, CharsetsSlotsG1, CharsetsSlotsG2, CharsetsSlotsG3}
	charsets := []CharsetsCharset{
		CharsetsCharsetUtf8,
		CharsetsCharsetAscii,
		CharsetsCharsetBritish,
		CharsetsCharsetDecSpecial,
	}
	for _, slot := range slots {
		for _, cs := range charsets {
			before := sroSnapshot(t, &tm)
			sroVt(t, &h, sroConfigureCharsetAct(slot, cs))
			expectEqual(t, cs, tm.screens.active.charset.charsets.Get(slot))
			// Nothing else changed.
			expected := before
			expected.charset.charsets.Set(slot, cs)
			expectEqual(t, expected, sroSnapshot(t, &tm))
		}
	}
}

// Zig: test "term51: Handler.vt: invoke_charset: the locking field is inverted, false is a locking shift and true a single shift"
func TestStreamReadonly_term51_Handler_vt_invoke_charset_the_locking_field_is_inverted_false_is_a_locking_shift_and_true_a_single_shift(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	expectEqual(t, CharsetsSlotsG0, tm.screens.active.charset.gl)
	expectEqual(t, CharsetsSlotsG2, tm.screens.active.charset.gr)

	// Upstream names the third argument "locking" but the handler passes it
	// to Terminal.invokeCharset as "single": the stream sends locking=false
	// for SO/SI and the locking shifts, locking=true for SS2 and SS3.
	sroVt(t, &h, sroInvokeCharsetAct(CharsetsActiveSlotGL, CharsetsSlotsG1, false))
	expectEqual(t, CharsetsSlotsG1, tm.screens.active.charset.gl)
	expectEqual(t, CharsetsSlotsG2, tm.screens.active.charset.gr)
	sroVt(t, &h, sroInvokeCharsetAct(CharsetsActiveSlotGL, CharsetsSlotsG3, false))
	expectEqual(t, CharsetsSlotsG3, tm.screens.active.charset.gl)
	sroVt(t, &h, sroInvokeCharsetAct(CharsetsActiveSlotGR, CharsetsSlotsG1, false))
	expectEqual(t, CharsetsSlotsG1, tm.screens.active.charset.gr)
	expectEqual(t, CharsetsSlotsG3, tm.screens.active.charset.gl)
	sroVt(t, &h, sroInvokeCharsetAct(CharsetsActiveSlotGR, CharsetsSlotsG3, false))
	expectEqual(t, CharsetsSlotsG3, tm.screens.active.charset.gr)
	sroVt(t, &h, sroInvokeCharsetAct(CharsetsActiveSlotGL, CharsetsSlotsG0, false))
	expectEqual(t, CharsetsSlotsG0, tm.screens.active.charset.gl)
	expectEqual(t, None[CharsetsSlots](), tm.screens.active.charset.single_shift)

	// A single shift is recorded without changing GL and applies to the
	// next printed character only.
	sroVt(t, &h, sroConfigureCharsetAct(CharsetsSlotsG2, CharsetsCharsetDecSpecial))
	sroVt(t, &h, sroInvokeCharsetAct(CharsetsActiveSlotGL, CharsetsSlotsG2, true))
	expectEqual(t, Some(CharsetsSlotsG2), tm.screens.active.charset.single_shift)
	expectEqual(t, CharsetsSlotsG0, tm.screens.active.charset.gl)
	sroVt(t, &h, sroPrintAct('`'))
	sroVt(t, &h, sroPrintAct('`'))
	sroExpectText(t, &tm, "◆`")
	expectEqual(t, None[CharsetsSlots](), tm.screens.active.charset.single_shift)

	// A locking shift to a DEC special graphics slot.
	sroVt(t, &h, sroConfigureCharsetAct(CharsetsSlotsG1, CharsetsCharsetDecSpecial))
	sroVt(t, &h, sroInvokeCharsetAct(CharsetsActiveSlotGL, CharsetsSlotsG1, false))
	sroVt(t, &h, sroPrintAct('`'))
	sroVt(t, &h, sroInvokeCharsetAct(CharsetsActiveSlotGL, CharsetsSlotsG0, false))
	sroVt(t, &h, sroPrintAct('`'))
	sroExpectText(t, &tm, "◆`◆`")

	// British.
	sroVt(t, &h, sroConfigureCharsetAct(CharsetsSlotsG0, CharsetsCharsetBritish))
	sroVt(t, &h, sroPrintAct('#'))
	sroExpectText(t, &tm, "◆`◆`£")
}

// Zig: test "term51: Handler.vt: set_attribute applies attributes, ignores unknown ones"
func TestStreamReadonly_term51_Handler_vt_set_attribute_applies_attributes_ignores_unknown_ones(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	c := &tm.screens.active.cursor
	sroVt(t, &h, sroSGRAct(SGRAttributeTagBold))
	expect(t, c.style.flags.bold, "c.style.flags.bold")
	sroVt(t, &h, sroSGRAct(SGRAttributeTagItalic))
	expect(t, c.style.flags.italic, "c.style.flags.italic")
	sroVt(t, &h, sroSGRAct(SGRAttributeTagFaint))
	expect(t, c.style.flags.faint, "c.style.flags.faint")
	sroVt(t, &h, sroSGRAct(SGRAttributeTagInverse))
	expect(t, c.style.flags.inverse, "c.style.flags.inverse")
	sroVt(t, &h, StreamAction{
		Tag:          StreamActionKeySetAttribute,
		SetAttribute: SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: SGRAttributeUnderlineCurly},
	})
	expectEqual(t, SGRAttributeUnderlineCurly, c.style.flags.underline)
	sroVt(t, &h, StreamAction{
		Tag:          StreamActionKeySetAttribute,
		SetAttribute: SGRAttribute{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: ColorRGB{r: 1, g: 2, b: 3}},
	})
	expectEqual(t, StyleColorRgb, c.style.fg_color.Tag)
	expectEqual(t, uint8(2), c.style.fg_color.Rgb.g)
	sroVt(t, &h, StreamAction{
		Tag:          StreamActionKeySetAttribute,
		SetAttribute: SGRAttribute{Tag: SGRAttributeTag256Bg, X256Bg: 200},
	})
	expectEqual(t, StyleColorPalette, c.style.bg_color.Tag)
	expectEqual(t, uint8(200), c.style.bg_color.Palette)

	// Unknown attributes do nothing at all.
	before := sroSnapshot(t, &tm)
	sroVt(t, &h, StreamAction{
		Tag: StreamActionKeySetAttribute,
		SetAttribute: SGRAttribute{
			Tag:     SGRAttributeTagUnknown,
			Unknown: SGRAttributeUnknown{full: []uint16{38, 99}, partial: []uint16{99}},
		},
	})
	expectEqual(t, before, sroSnapshot(t, &tm))

	// Resets.
	sroVt(t, &h, sroSGRAct(SGRAttributeTagResetBold))
	expect(t, !c.style.flags.bold, "!c.style.flags.bold")
	expect(t, c.style.flags.italic, "c.style.flags.italic")

	// A printed cell carries the style.
	sroVt(t, &h, sroPrintAct('A'))
	cell := sroGetCell(t, &tm, 0, 0)
	expect(t, cell.cell.style_id != 0, "cell.cell.style_id != 0")

	// Unset clears everything.
	sroVt(t, &h, sroSGRAct(SGRAttributeTagUnset))
	expect(t, c.style.Default(), "c.style.default()")
	sroVt(t, &h, sroPrintAct('B'))
	cell2 := sroGetCell(t, &tm, 1, 0)
	expectEqual(t, 0, cell2.cell.style_id)
}

// Zig: test "term51: Handler.vt: mouse_shift_capture, active_status_display and mouse_shape"
func TestStreamReadonly_term51_Handler_vt_mouse_shift_capture_active_status_display_and_mouse_shape(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	expectEqual(t, TerminalFlagsMouseShiftCaptureNull, tm.flags.mouse_shift_capture)
	sroVt(t, &h, sroBoolAct(StreamActionKeyMouseShiftCapture, true))
	expectEqual(t, TerminalFlagsMouseShiftCaptureTrue, tm.flags.mouse_shift_capture)
	sroVt(t, &h, sroBoolAct(StreamActionKeyMouseShiftCapture, false))
	expectEqual(t, TerminalFlagsMouseShiftCaptureFalse, tm.flags.mouse_shift_capture)
	sroVt(t, &h, sroBoolAct(StreamActionKeyMouseShiftCapture, true))
	expectEqual(t, TerminalFlagsMouseShiftCaptureTrue, tm.flags.mouse_shift_capture)

	expectEqual(t, ANSIStatusDisplayMain, tm.status_display)
	sroVt(t, &h, StreamAction{Tag: StreamActionKeyActiveStatusDisplay, ActiveStatusDisplay: ANSIStatusDisplayStatusLine})
	expectEqual(t, ANSIStatusDisplayStatusLine, tm.status_display)
	sroVt(t, &h, StreamAction{Tag: StreamActionKeyActiveStatusDisplay, ActiveStatusDisplay: ANSIStatusDisplayMain})
	expectEqual(t, ANSIStatusDisplayMain, tm.status_display)

	expectEqual(t, MouseShapeText, tm.mouse_shape)
	sroVt(t, &h, StreamAction{Tag: StreamActionKeyMouseShape, MouseShape: MouseShapePointer})
	expectEqual(t, MouseShapePointer, tm.mouse_shape)
	sroVt(t, &h, StreamAction{Tag: StreamActionKeyMouseShape, MouseShape: MouseShapeDefault})
	expectEqual(t, MouseShapeDefault, tm.mouse_shape)
	sroVt(t, &h, StreamAction{Tag: StreamActionKeyMouseShape, MouseShape: MouseShapeText})
	expectEqual(t, MouseShapeText, tm.mouse_shape)
}

// Zig: test "term51: Handler.vt: modify_key_format sets modify_other_keys_2 only for other_keys_numeric"
func TestStreamReadonly_term51_Handler_vt_modify_key_format_sets_modify_other_keys_2_only_for_other_keys_numeric(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	formats := []ANSIModifyKeyFormat{
		ANSIModifyKeyFormatLegacy,
		ANSIModifyKeyFormatCursorKeys,
		ANSIModifyKeyFormatFunctionKeys,
		ANSIModifyKeyFormatOtherKeysNone,
		ANSIModifyKeyFormatOtherKeysNumericExcept,
		ANSIModifyKeyFormatOtherKeysNumeric,
	}
	expect(t, !tm.flags.modify_other_keys_2, "!t.flags.modify_other_keys_2")
	for _, v := range formats {
		want := v == ANSIModifyKeyFormatOtherKeysNumeric
		// From both previous states the result is the same.
		tm.flags.modify_other_keys_2 = false
		sroVt(t, &h, StreamAction{Tag: StreamActionKeyModifyKeyFormat, ModifyKeyFormat: v})
		expectEqual(t, want, tm.flags.modify_other_keys_2)
		tm.flags.modify_other_keys_2 = true
		sroVt(t, &h, StreamAction{Tag: StreamActionKeyModifyKeyFormat, ModifyKeyFormat: v})
		expectEqual(t, want, tm.flags.modify_other_keys_2)
	}
	expectEqual(t, 6, len(formats))
}

// Zig: test "term51: Handler.vt: kitty keyboard push, pop and set operate on the active screen's stack"
func TestStreamReadonly_term51_Handler_vt_kitty_keyboard_push_pop_and_set_operate_on_the_active_screen_s_stack(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	kb := &tm.screens.active.kitty_keyboard
	expectEqual(t, uint8(0), kb.Current().Int())

	sroVt(t, &h, sroKittyAct(StreamActionKeyKittyKeyboardPush, KittyKeyFlags{disambiguate: true}))
	expectEqual(t, uint8(1), kb.Current().Int())
	expectEqual(t, uint8(1), kb.idx)
	sroVt(t, &h, sroKittyAct(StreamActionKeyKittyKeyboardPush, KittyKeyFlags{report_events: true, report_all: true}))
	expectEqual(t, uint8(10), kb.Current().Int())
	expectEqual(t, uint8(2), kb.idx)

	// Pop zero entries does nothing, one goes back.
	sroVt(t, &h, sroU16Act(StreamActionKeyKittyKeyboardPop, 0))
	expectEqual(t, uint8(10), kb.Current().Int())
	sroVt(t, &h, sroU16Act(StreamActionKeyKittyKeyboardPop, 1))
	expectEqual(t, uint8(1), kb.Current().Int())
	expectEqual(t, uint8(1), kb.idx)

	// set / or / not apply to the current entry only.
	sroVt(t, &h, sroKittyAct(StreamActionKeyKittyKeyboardSet, KittyKeyFlags{report_all: true}))
	expectEqual(t, uint8(8), kb.Current().Int())
	sroVt(t, &h, sroKittyAct(StreamActionKeyKittyKeyboardSetOr, KittyKeyFlags{disambiguate: true, report_associated: true}))
	expectEqual(t, uint8(8|1|16), kb.Current().Int())
	sroVt(t, &h, sroKittyAct(StreamActionKeyKittyKeyboardSetNot, KittyKeyFlags{report_all: true, report_events: true}))
	expectEqual(t, uint8(1|16), kb.Current().Int())
	sroVt(t, &h, sroKittyAct(StreamActionKeyKittyKeyboardSet, KittyKeyFlags{}))
	expectEqual(t, uint8(0), kb.Current().Int())
	sroVt(t, &h, sroKittyAct(StreamActionKeyKittyKeyboardSet, KittyKeyFlagsTrue))
	expectEqual(t, uint8(31), kb.Current().Int())
	expectEqual(t, uint8(1), kb.idx)
	expectEqual(t, uint8(0), kb.flags[0].Int())
}

// Zig: test "term51: Handler.vt: kitty keyboard pop with large counts resets the stack, push wraps around"
func TestStreamReadonly_term51_Handler_vt_kitty_keyboard_pop_with_large_counts_resets_the_stack_push_wraps_around(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	kb := &tm.screens.active.kitty_keyboard

	// Pop on an empty stack keeps the flags disabled but the index wraps
	// around to the last entry (upstream behaviour).
	sroVt(t, &h, sroU16Act(StreamActionKeyKittyKeyboardPop, 1))
	expectEqual(t, uint8(0), kb.Current().Int())
	expectEqual(t, uint8(7), kb.idx)

	// Nine pushes overflow the 8 entry stack and evict the oldest.
	for i := uint8(1); i <= 9; i++ {
		sroVt(t, &h, sroKittyAct(StreamActionKeyKittyKeyboardPush, KittyKeyFlagsFromInt(i)))
	}
	expectEqual(t, uint8(9), kb.Current().Int())
	// The first push landed on index 0 because the index started at 7.
	expectEqual(t, uint8(0), kb.idx)
	// Popping 7 goes back around to the entry pushed second.
	sroVt(t, &h, sroU16Act(StreamActionKeyKittyKeyboardPop, 7))
	expectEqual(t, uint8(1), kb.idx)
	expectEqual(t, uint8(2), kb.Current().Int())

	// 8 or more clears the whole stack.
	sroVt(t, &h, sroKittyAct(StreamActionKeyKittyKeyboardPush, KittyKeyFlags{disambiguate: true}))
	sroVt(t, &h, sroU16Act(StreamActionKeyKittyKeyboardPop, 8))
	expectEqual(t, uint8(0), kb.idx)
	for _, f := range kb.flags {
		expectEqual(t, uint8(0), f.Int())
	}

	sroVt(t, &h, sroKittyAct(StreamActionKeyKittyKeyboardPush, KittyKeyFlags{disambiguate: true}))
	sroVt(t, &h, sroKittyAct(StreamActionKeyKittyKeyboardPush, KittyKeyFlags{disambiguate: true}))
	sroVt(t, &h, sroU16Act(StreamActionKeyKittyKeyboardPop, 65535))
	expectEqual(t, uint8(0), kb.idx)
	expectEqual(t, uint8(0), kb.Current().Int())
	for _, f := range kb.flags {
		expectEqual(t, uint8(0), f.Int())
	}
}

// Zig: test "term51: Handler.vt: kitty keyboard flags are per screen"
func TestStreamReadonly_term51_Handler_vt_kitty_keyboard_flags_are_per_screen(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroVt(t, &h, sroKittyAct(StreamActionKeyKittyKeyboardPush, KittyKeyFlags{disambiguate: true}))
	zigTry(t, h.setMode(ModesModeAltScreenLegacy, true))
	expectEqual(t, uint8(0), tm.screens.active.kitty_keyboard.Current().Int())
	sroVt(t, &h, sroKittyAct(StreamActionKeyKittyKeyboardPush, KittyKeyFlags{report_all: true}))
	expectEqual(t, uint8(8), tm.screens.active.kitty_keyboard.Current().Int())
	zigTry(t, h.setMode(ModesModeAltScreenLegacy, false))
	expectEqual(t, uint8(1), tm.screens.active.kitty_keyboard.Current().Int())
	sroVt(t, &h, sroU16Act(StreamActionKeyKittyKeyboardPop, 1))
	expectEqual(t, uint8(0), tm.screens.active.kitty_keyboard.Current().Int())
	zigTry(t, h.setMode(ModesModeAltScreenLegacy, true))
	expectEqual(t, uint8(8), tm.screens.active.kitty_keyboard.Current().Int())
}

// Zig: test "term51: Handler.vt: decaln fills with E, resets margins and origin, homes the cursor"
func TestStreamReadonly_term51_Handler_vt_decaln_fills_with_E_resets_margins_and_origin_homes_the_cursor(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroPrintString(t, &tm, "abc")
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeEnableLeftAndRightMargin))
	sroVt(t, &h, sroMarginAct(StreamActionKeyLeftAndRightMargin, 3, 6))
	sroVt(t, &h, sroMarginAct(StreamActionKeyTopAndBottomMargin, 2, 3))
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeOrigin))
	sroVt(t, &h, sroSGRAct(SGRAttributeTagBold))
	sroVt(t, &h, StreamAction{
		Tag:          StreamActionKeySetAttribute,
		SetAttribute: SGRAttribute{Tag: SGRAttributeTag8Bg, X8Bg: ColorNameBlue},
	})
	sroVt(t, &h, sroAct(StreamActionKeyProtectedModeDec))
	sroAt(&tm, 4, 2)

	sroVt(t, &h, sroAct(StreamActionKeyDecaln))
	sroExpectText(t, &tm, "EEEEEEEEEE\nEEEEEEEEEE\nEEEEEEEEEE")
	sroExpectCursor(t, &tm, 0, 0)
	expectEqual(t, 0, tm.scrolling_region.top)
	expectEqual(t, 2, tm.scrolling_region.bottom)
	expectEqual(t, 0, tm.scrolling_region.left)
	expectEqual(t, 9, tm.scrolling_region.right)
	expect(t, !tm.modes.Get(ModesModeOrigin), "!t.modes.get(.origin)")
	// The left/right margin mode itself is not touched.
	expect(t, tm.modes.Get(ModesModeEnableLeftAndRightMargin), "t.modes.get(.enable_left_and_right_margin)")
	// Attributes other than the colours are cleared; colours are kept.
	expect(t, !tm.screens.active.cursor.style.flags.bold, "!t.screens.active.cursor.style.flags.bold")
	expectEqual(t, StyleColorPalette, tm.screens.active.cursor.style.bg_color.Tag)
	expectEqual(t, uint8(4), tm.screens.active.cursor.style.bg_color.Palette)

	// Cells do not carry the protection.
	cell := sroGetCell(t, &tm, 3, 1)
	expect(t, !cell.cell.protected, "!cell.cell.protected")
	expectEqual(t, uint32('E'), cell.cell.content.codepoint)
}

// Zig: test "term51: Handler.vt: full_reset restores a pristine terminal but keeps colors and size"
func TestStreamReadonly_term51_Handler_vt_full_reset_restores_a_pristine_terminal_but_keeps_colors_and_size(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 6)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	fresh := sroNew(t, 20, 6)
	defer fresh.Deinit()
	pristine := sroSnapshot(t, &fresh)

	sroBusy(t, &tm, &h)
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeAltScreenSaveCursorClearEnter))
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeInsert))
	sroVt(t, &h, sroModeAct(StreamActionKeyResetMode, ModesModeWraparound))
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeOrigin))
	sroVt(t, &h, sroSGRAct(SGRAttributeTagBold))
	sroVt(t, &h, sroAct(StreamActionKeyProtectedModeIso))
	sroVt(t, &h, sroConfigureCharsetAct(CharsetsSlotsG0, CharsetsCharsetDecSpecial))
	sroVt(t, &h, StreamAction{Tag: StreamActionKeyActiveStatusDisplay, ActiveStatusDisplay: ANSIStatusDisplayStatusLine})
	sroVt(t, &h, sroBoolAct(StreamActionKeyMouseShiftCapture, true))
	sroVt(t, &h, StreamAction{Tag: StreamActionKeyModifyKeyFormat, ModifyKeyFormat: ANSIModifyKeyFormatOtherKeysNumeric})
	sroVt(t, &h, sroAct(StreamActionKeyTabClearAll))
	sroVt(t, &h, sroStartHyperlinkAct("http://a", None[[]uint8]()))
	sroVt(t, &h, sroModeAct(StreamActionKeySaveMode, ModesModeBracketedPaste))
	sroVt(t, &h, sroAct(StreamActionKeySaveCursor))
	sroVt(t, &h, sroPrintAct('x'))
	zigTry(t, tm.SetPwd([]uint8("/tmp")))
	colors_before := tm.colors
	tm.mouse_shape = MouseShapePointer

	sroVt(t, &h, sroAct(StreamActionKeyFullReset))

	expectEqual(t, ScreenSetKeyPrimary, tm.screens.active_key)
	expect(t, tm.screens.Get(ScreenSetKeyAlternate) == nil, "t.screens.get(.alternate) == null")
	expected := pristine
	// The renderer is told to redraw everything.
	flags := terminalFlagsDefault()
	flags.dirty.clear = true
	expected.flags = flags
	// Colors are not reset, neither is the mouse shape.
	expected.colors = colors_before
	expected.mouse_shape = MouseShapePointer
	expectEqual(t, expected, sroSnapshot(t, &tm))
	expectEqual(t, 0, len(tm.pwd))
	expectEqual(t, None[uint32](), tm.previous_char)
}

// Zig: test "term51: Handler.vt: full_reset keeps the size set by DECCOLM and reset modes return to the defaults"
func TestStreamReadonly_term51_Handler_vt_full_reset_keeps_the_size_set_by_DECCOLM_and_reset_modes_return_to_the_defaults(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesModeEnableMode3))
	sroVt(t, &h, sroModeAct(StreamActionKeySetMode, ModesMode132Column))
	expectEqual(t, 132, tm.cols)
	sroVt(t, &h, sroAct(StreamActionKeyFullReset))
	expectEqual(t, 132, tm.cols)
	expectEqual(t, 131, tm.scrolling_region.right)
	expect(t, !tm.modes.Get(ModesModeEnableMode3), "!t.modes.get(.enable_mode_3)")
	expect(t, !tm.modes.Get(ModesMode132Column), "!t.modes.get(.@\"132_column\")")
	expect(t, tm.modes.Get(ModesModeWraparound), "t.modes.get(.wraparound)")
	expect(t, tm.tabstops.Get(128), "t.tabstops.get(128)")
}

// Zig: test "term51: Handler.vt: start_hyperlink and end_hyperlink"
func TestStreamReadonly_term51_Handler_vt_start_hyperlink_and_end_hyperlink(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	c := &tm.screens.active.cursor
	expectEqual(t, 0, c.hyperlink_id)

	// Ending nothing is harmless.
	before := sroSnapshot(t, &tm)
	sroVt(t, &h, sroAct(StreamActionKeyEndHyperlink))
	expectEqual(t, before, sroSnapshot(t, &tm))

	// Explicit id.
	sroVt(t, &h, sroStartHyperlinkAct("http://a.example", Some([]uint8("myid"))))
	expect(t, c.hyperlink_id != 0, "c.hyperlink_id != 0")
	expectEqualStrings(t, "http://a.example", string(c.hyperlink.uri))
	expectEqual(t, HyperlinkHyperlinkIdExplicit, c.hyperlink.id.Tag)
	expectEqualStrings(t, "myid", string(c.hyperlink.id.Explicit))
	expectEqual(t, 0, c.hyperlink_implicit_id)
	sroVt(t, &h, sroPrintAct('A'))
	sroVt(t, &h, sroAct(StreamActionKeyEndHyperlink))
	expectEqual(t, 0, c.hyperlink_id)
	expect(t, c.hyperlink == nil, "c.hyperlink == null")
	sroVt(t, &h, sroPrintAct('B'))
	a := sroGetCell(t, &tm, 0, 0)
	b := sroGetCell(t, &tm, 1, 0)
	expect(t, a.cell.hyperlink, "a.cell.hyperlink")
	expect(t, !b.cell.hyperlink, "!b.cell.hyperlink")

	// Implicit ids count up.
	sroVt(t, &h, sroStartHyperlinkAct("http://b.example", None[[]uint8]()))
	expectEqual(t, HyperlinkHyperlinkIdImplicit, c.hyperlink.id.Tag)
	expectEqual(t, 0, c.hyperlink.id.Implicit)
	expectEqual(t, 1, c.hyperlink_implicit_id)
	// Starting another replaces the first.
	sroVt(t, &h, sroStartHyperlinkAct("http://c.example", None[[]uint8]()))
	expectEqualStrings(t, "http://c.example", string(c.hyperlink.uri))
	expectEqual(t, HyperlinkHyperlinkIdImplicit, c.hyperlink.id.Tag)
	expectEqual(t, 1, c.hyperlink.id.Implicit)
	expectEqual(t, 2, c.hyperlink_implicit_id)
	sroVt(t, &h, sroAct(StreamActionKeyEndHyperlink))
	expect(t, c.hyperlink == nil, "c.hyperlink == null")
}

// sroSP is `Action.SemanticPrompt.init(action)` (SP.init).
func sroSP(action OSCParsersSemanticPromptCommandAction) StreamAction {
	return sroSemanticPromptAct(OSCParsersSemanticPromptCommandInit(action))
}

// Zig: test "term51: Handler.vt: semantic_prompt actions"
func TestStreamReadonly_term51_Handler_vt_semantic_prompt_actions(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 5)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	// fresh_line does nothing at column 0, moves to the next line otherwise.
	sroVt(t, &h, sroSP(OSCParsersSemanticPromptCommandActionFreshLine))
	sroExpectCursor(t, &tm, 0, 0)
	sroPrintString(t, &tm, "abc")
	sroVt(t, &h, sroSP(OSCParsersSemanticPromptCommandActionFreshLine))
	sroExpectCursor(t, &tm, 0, 1)
	expectEqual(t, PageCellSemanticContentOutput, tm.screens.active.cursor.semantic_content)

	// fresh_line_new_prompt: fresh line plus prompt content, and the
	// redraw option.
	sroVt(t, &h, sroSP(OSCParsersSemanticPromptCommandActionFreshLineNewPrompt))
	sroExpectCursor(t, &tm, 0, 1)
	expectEqual(t, PageCellSemanticContentPrompt, tm.screens.active.cursor.semantic_content)
	expect(t, tm.flags.shell_redraws_prompt == OSCParsersSemanticPromptRedrawTrue, "t.flags.shell_redraws_prompt == .true")
	sroPrintString(t, &tm, "$ ")
	sroVt(t, &h, sroSemanticPromptAct(OSCParsersSemanticPromptCommand{
		action:              OSCParsersSemanticPromptCommandActionFreshLineNewPrompt,
		options_unvalidated: []uint8("redraw=0"),
	}))
	sroExpectCursor(t, &tm, 0, 2)
	expect(t, tm.flags.shell_redraws_prompt == OSCParsersSemanticPromptRedrawFalse, "t.flags.shell_redraws_prompt == .false")
	sroVt(t, &h, sroSemanticPromptAct(OSCParsersSemanticPromptCommand{
		action:              OSCParsersSemanticPromptCommandActionFreshLineNewPrompt,
		options_unvalidated: []uint8("redraw=1"),
	}))
	expect(t, tm.flags.shell_redraws_prompt == OSCParsersSemanticPromptRedrawTrue, "t.flags.shell_redraws_prompt == .true")

	// new_command behaves like fresh_line_new_prompt.
	sroPrintString(t, &tm, "x")
	sroVt(t, &h, sroSP(OSCParsersSemanticPromptCommandActionNewCommand))
	sroExpectCursor(t, &tm, 0, 3)
	expectEqual(t, PageCellSemanticContentPrompt, tm.screens.active.cursor.semantic_content)

	// end_prompt_start_input and the end-of-line variant.
	sroVt(t, &h, sroSP(OSCParsersSemanticPromptCommandActionEndPromptStartInput))
	expectEqual(t, PageCellSemanticContentInput, tm.screens.active.cursor.semantic_content)
	expect(t, !tm.screens.active.cursor.semantic_content_clear_eol, "!t.screens.active.cursor.semantic_content_clear_eol")
	sroVt(t, &h, sroSP(OSCParsersSemanticPromptCommandActionEndPromptStartInputTerminateEol))
	expectEqual(t, PageCellSemanticContentInput, tm.screens.active.cursor.semantic_content)
	expect(t, tm.screens.active.cursor.semantic_content_clear_eol, "t.screens.active.cursor.semantic_content_clear_eol")
	sroVt(t, &h, sroAct(StreamActionKeyLinefeed))
	expectEqual(t, PageCellSemanticContentOutput, tm.screens.active.cursor.semantic_content)

	// end_input_start_output and end_command.
	sroVt(t, &h, sroSP(OSCParsersSemanticPromptCommandActionPromptStart))
	sroExpectCursor(t, &tm, 0, 4)
	expectEqual(t, PageCellSemanticContentPrompt, tm.screens.active.cursor.semantic_content)
	sroVt(t, &h, sroSP(OSCParsersSemanticPromptCommandActionEndInputStartOutput))
	expectEqual(t, PageCellSemanticContentOutput, tm.screens.active.cursor.semantic_content)
	sroVt(t, &h, sroSP(OSCParsersSemanticPromptCommandActionPromptStart))
	sroVt(t, &h, sroSP(OSCParsersSemanticPromptCommandActionEndCommand))
	expectEqual(t, PageCellSemanticContentOutput, tm.screens.active.cursor.semantic_content)
	// None of these moved the cursor.
	sroExpectCursor(t, &tm, 0, 4)
}

var (
	sroRed   = ColorRGB{r: 255, g: 0, b: 0}        // term51D_red
	sroGreen = ColorRGB{r: 0, g: 255, b: 0}        // term51D_green
	sroBlue  = ColorRGB{r: 0, g: 0, b: 255}        // term51D_blue
	sroC1    = ColorRGB{r: 0x11, g: 0x22, b: 0x33} // term51D_c1
	sroC2    = ColorRGB{r: 0x44, g: 0x55, b: 0x66} // term51D_c2
)

// sroNewWithDefaults is term51DNewWithDefaults: a terminal with configured
// defaults for the dynamic colors and a custom base palette, so that resets
// are distinguishable from "unset".
func sroNewWithDefaults(t *testing.T) Terminal {
	t.Helper()
	palette := ColorDefault
	palette[1] = ColorRGB{r: 0x10, g: 0x20, b: 0x30}
	palette[200] = ColorRGB{r: 0x01, g: 0x02, b: 0x03}
	opts := terminalOptionsDefault(10, 3)
	opts.colors = TerminalColors{
		background: ColorDynamicRGBInit(ColorRGB{r: 0xb1, g: 0xb2, b: 0xb3}),
		foreground: ColorDynamicRGBInit(ColorRGB{r: 0xf1, g: 0xf2, b: 0xf3}),
		cursor:     ColorDynamicRGBInit(ColorRGB{r: 0xc1, g: 0xc2, b: 0xc3}),
		palette:    ColorDynamicPaletteInit(palette),
	}
	tm, err := TerminalInit(opts)
	zigTry(t, err)
	return tm
}

// The requests of osc_color.Request and the targets they name.

func sroTPalette(i uint8) OSCParsersColorTarget {
	return OSCParsersColorTarget{Tag: OSCParsersColorTargetPalette, Palette: i}
}

func sroTDynamic(d ColorDynamic) OSCParsersColorTarget {
	return OSCParsersColorTarget{Tag: OSCParsersColorTargetDynamic, Dynamic: d}
}

func sroTSpecial(s ColorSpecial) OSCParsersColorTarget {
	return OSCParsersColorTarget{Tag: OSCParsersColorTargetSpecial, Special: s}
}

func sroCSet(target OSCParsersColorTarget, color ColorRGB) OSCParsersColorRequest {
	return OSCParsersColorRequest{
		Tag: OSCParsersColorRequestSet,
		Set: OSCParsersColorColoredTarget{target: target, color: color},
	}
}

func sroCReset(target OSCParsersColorTarget) OSCParsersColorRequest {
	return OSCParsersColorRequest{Tag: OSCParsersColorRequestReset, Reset: target}
}

func sroCQuery(target OSCParsersColorTarget) OSCParsersColorRequest {
	return OSCParsersColorRequest{Tag: OSCParsersColorRequestQuery, Query: target}
}

var (
	sroCResetPalette = OSCParsersColorRequest{Tag: OSCParsersColorRequestResetPalette}
	sroCResetSpecial = OSCParsersColorRequest{Tag: OSCParsersColorRequestResetSpecial}
)

// The requests of kitty_color.OSC.Request and the keys they name.

func sroKPalette(i uint8) KittyColorKind {
	return KittyColorKind{Tag: KittyColorKindPalette, Palette: i}
}

func sroKSpecial(s KittyColorSpecial) KittyColorKind {
	return KittyColorKind{Tag: KittyColorKindSpecial, Special: s}
}

func sroKSet(key KittyColorKind, color ColorRGB) KittyColorOSCRequest {
	return KittyColorOSCRequest{
		Tag: KittyColorOSCRequestSet,
		Set: KittyColorOSCRequestSetPayload{key: key, color: color},
	}
}

func sroKReset(key KittyColorKind) KittyColorOSCRequest {
	return KittyColorOSCRequest{Tag: KittyColorOSCRequestReset, Reset: key}
}

func sroKQuery(key KittyColorKind) KittyColorOSCRequest {
	return KittyColorOSCRequest{Tag: KittyColorOSCRequestQuery, Query: key}
}

// sroColorOp is term51DColorOp: build a list of requests and run the private
// colorOperation.
func sroColorOp(t *testing.T, h *StreamReadonlyHandler, op OSCParsersColorOperation, items []OSCParsersColorRequest) {
	t.Helper()
	var list OSCParsersColorList
	// Zig-only: defer list.deinit(testing.allocator)
	for _, item := range items {
		zigTry(t, list.append(item))
	}
	zigTry(t, h.colorOperation(op, &list))
}

// sroKittyOp is term51DKittyOp: build a kitty color request list and run the
// private kittyColorOperation.
func sroKittyOp(t *testing.T, h *StreamReadonlyHandler, items []KittyColorOSCRequest) {
	t.Helper()
	osc := KittyColorOSC{}
	// Zig-only: defer osc.deinit(testing.allocator)
	osc.list = append(osc.list, items...)
	zigTry(t, h.kittyColorOperation(osc))
}

// sroExpectPaletteOnly is term51DExpectPaletteOnly.
func sroExpectPaletteOnly(t *testing.T, tm *Terminal, changed []uint8) {
	t.Helper()
	for i := 0; i < 256; i++ {
		is_changed := false
		for _, c := range changed {
			if int(c) == i {
				is_changed = true
			}
		}
		expectEqual(t, is_changed, tm.colors.palette.mask.isSet(i))
		if !is_changed {
			expectEqual(t, tm.colors.palette.original[i], tm.colors.palette.current[i])
		}
	}
	expectEqual(t, len(changed), tm.colors.palette.mask.count())
}

// Zig: test "term51: Handler.colorOperation: an empty request list changes nothing"
func TestStreamReadonly_term51_Handler_colorOperation_an_empty_request_list_changes_nothing(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewWithDefaults(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	sroBusy(t, &tm, &h)

	before := sroSnapshot(t, &tm)
	sroColorOp(t, &h, OSCParsersColorOperationOsc4, nil)
	sroColorOp(t, &h, OSCParsersColorOperationOsc104, nil)
	sroColorOp(t, &h, OSCParsersColorOperationOsc10, nil)
	expectEqual(t, before, sroSnapshot(t, &tm))
	expect(t, !tm.flags.dirty.palette, "!t.flags.dirty.palette")
}

// Zig: test "term51: Handler.colorOperation: set palette entries at the boundaries"
func TestStreamReadonly_term51_Handler_colorOperation_set_palette_entries_at_the_boundaries(t *testing.T) {
	testStdCapacity(t)
	for _, idx := range []uint8{0, 1, 15, 16, 127, 231, 232, 254, 255} {
		func() {
			tm := sroNewWithDefaults(t)
			defer tm.Deinit()
			h := StreamReadonlyHandlerInit(&tm)

			expect(t, !tm.flags.dirty.palette, "!t.flags.dirty.palette")
			sroColorOp(t, &h, OSCParsersColorOperationOsc4, []OSCParsersColorRequest{sroCSet(sroTPalette(idx), sroC1)})
			expectEqual(t, sroC1, tm.colors.palette.current[idx])
			sroExpectPaletteOnly(t, &tm, []uint8{idx})
			expect(t, tm.flags.dirty.palette, "t.flags.dirty.palette")
			// The "original" is not touched.
			expect(t, sroC1 != tm.colors.palette.original[idx], "!std.meta.eql(term51D_c1, t.colors.palette.original[idx])")
			// Nothing else was affected.
			expect(t, zigUnwrap(t, tm.colors.foreground.Get()).r == 0xf1, "t.colors.foreground.get().?.r == 0xf1")
			expectEqual(t, Optional[ColorRGB]{}, tm.colors.foreground.override)
			expectEqual(t, Optional[ColorRGB]{}, tm.colors.background.override)
			expectEqual(t, Optional[ColorRGB]{}, tm.colors.cursor.override)

			// Setting again replaces the color, the mask stays a single entry.
			sroColorOp(t, &h, OSCParsersColorOperationOsc4, []OSCParsersColorRequest{sroCSet(sroTPalette(idx), sroC2)})
			expectEqual(t, sroC2, tm.colors.palette.current[idx])
			sroExpectPaletteOnly(t, &tm, []uint8{idx})

			// Setting a color to the value it already has still marks it.
			orig := tm.colors.palette.original[idx]
			sroColorOp(t, &h, OSCParsersColorOperationOsc4, []OSCParsersColorRequest{sroCReset(sroTPalette(idx))})
			sroColorOp(t, &h, OSCParsersColorOperationOsc4, []OSCParsersColorRequest{sroCSet(sroTPalette(idx), orig)})
			expectEqual(t, orig, tm.colors.palette.current[idx])
			sroExpectPaletteOnly(t, &tm, []uint8{idx})
		}()
	}
}

// Zig: test "term51: Handler.colorOperation: the operation argument is ignored"
func TestStreamReadonly_term51_Handler_colorOperation_the_operation_argument_is_ignored(t *testing.T) {
	testStdCapacity(t)
	for op := OSCParsersColorOperation(0); op < oscParsersColorOperationCount; op++ {
		func() {
			tm := sroNewWithDefaults(t)
			defer tm.Deinit()
			h := StreamReadonlyHandlerInit(&tm)

			sroColorOp(t, &h, op, []OSCParsersColorRequest{
				sroCSet(sroTPalette(9), sroC1),
				sroCSet(sroTDynamic(ColorDynamicBackground), sroC2),
			})
			expectEqual(t, sroC1, tm.colors.palette.current[9])
			expectEqual(t, sroC2, zigUnwrap(t, tm.colors.background.Get()))
			sroColorOp(t, &h, op, []OSCParsersColorRequest{sroCReset(sroTPalette(9))})
			sroExpectPaletteOnly(t, &tm, []uint8{})
		}()
	}
}

// Zig: test "term51: Handler.colorOperation: set dynamic colors only for foreground, background and cursor"
func TestStreamReadonly_term51_Handler_colorOperation_set_dynamic_colors_only_for_foreground_background_and_cursor(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewWithDefaults(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroColorOp(t, &h, OSCParsersColorOperationOsc10, []OSCParsersColorRequest{sroCSet(sroTDynamic(ColorDynamicForeground), sroC1)})
	expectEqual(t, sroC1, zigUnwrap(t, tm.colors.foreground.Get()))
	expectEqual(t, Some(sroC1), tm.colors.foreground.override)
	// The configured default is untouched.
	expectEqual(t, Some(ColorRGB{r: 0xf1, g: 0xf2, b: 0xf3}), tm.colors.foreground.default_)
	expectEqual(t, Optional[ColorRGB]{}, tm.colors.background.override)
	expectEqual(t, Optional[ColorRGB]{}, tm.colors.cursor.override)

	sroColorOp(t, &h, OSCParsersColorOperationOsc11, []OSCParsersColorRequest{sroCSet(sroTDynamic(ColorDynamicBackground), sroC2)})
	expectEqual(t, sroC2, zigUnwrap(t, tm.colors.background.Get()))
	expectEqual(t, sroC1, zigUnwrap(t, tm.colors.foreground.Get()))

	sroColorOp(t, &h, OSCParsersColorOperationOsc12, []OSCParsersColorRequest{sroCSet(sroTDynamic(ColorDynamicCursor), sroRed)})
	expectEqual(t, sroRed, zigUnwrap(t, tm.colors.cursor.Get()))
	expectEqual(t, sroC2, zigUnwrap(t, tm.colors.background.Get()))
	expectEqual(t, sroC1, zigUnwrap(t, tm.colors.foreground.Get()))

	// Dynamic colors never touch the palette or its dirty flag.
	expect(t, !tm.flags.dirty.palette, "!t.flags.dirty.palette")
	sroExpectPaletteOnly(t, &tm, []uint8{})
}

// Zig: test "term51: Handler.colorOperation: the other dynamic colors and special colors are ignored"
func TestStreamReadonly_term51_Handler_colorOperation_the_other_dynamic_colors_and_special_colors_are_ignored(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewWithDefaults(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	sroBusy(t, &tm, &h)
	tm.flags.dirty.palette = false

	before := sroSnapshot(t, &tm)
	for _, d := range []ColorDynamic{
		ColorDynamicPointerForeground,
		ColorDynamicPointerBackground,
		ColorDynamicTektronixForeground,
		ColorDynamicTektronixBackground,
		ColorDynamicHighlightBackground,
		ColorDynamicTektronixCursor,
		ColorDynamicHighlightForeground,
	} {
		sroColorOp(t, &h, OSCParsersColorOperationOsc13, []OSCParsersColorRequest{sroCSet(sroTDynamic(d), sroC1)})
		sroColorOp(t, &h, OSCParsersColorOperationOsc113, []OSCParsersColorRequest{sroCReset(sroTDynamic(d))})
		sroColorOp(t, &h, OSCParsersColorOperationOsc13, []OSCParsersColorRequest{sroCQuery(sroTDynamic(d))})
	}
	for _, sp := range []ColorSpecial{
		ColorSpecialBold,
		ColorSpecialUnderline,
		ColorSpecialBlink,
		ColorSpecialReverse,
		ColorSpecialItalic,
	} {
		sroColorOp(t, &h, OSCParsersColorOperationOsc5, []OSCParsersColorRequest{sroCSet(sroTSpecial(sp), sroC1)})
		sroColorOp(t, &h, OSCParsersColorOperationOsc105, []OSCParsersColorRequest{sroCReset(sroTSpecial(sp))})
		sroColorOp(t, &h, OSCParsersColorOperationOsc5, []OSCParsersColorRequest{sroCQuery(sroTSpecial(sp))})
	}
	expectEqual(t, before, sroSnapshot(t, &tm))
	expect(t, !tm.flags.dirty.palette, "!t.flags.dirty.palette")
}

// Zig: test "term51: Handler.colorOperation: reset palette entry restores the original and clears the mask bit"
func TestStreamReadonly_term51_Handler_colorOperation_reset_palette_entry_restores_the_original_and_clears_the_mask_bit(t *testing.T) {
	testStdCapacity(t)
	for _, idx := range []uint8{0, 1, 200, 255} {
		func() {
			tm := sroNewWithDefaults(t)
			defer tm.Deinit()
			h := StreamReadonlyHandlerInit(&tm)

			sroColorOp(t, &h, OSCParsersColorOperationOsc4, []OSCParsersColorRequest{sroCSet(sroTPalette(idx), sroC1)})
			sroColorOp(t, &h, OSCParsersColorOperationOsc4, []OSCParsersColorRequest{sroCSet(sroTPalette(5), sroC2)})
			tm.flags.dirty.palette = false
			sroColorOp(t, &h, OSCParsersColorOperationOsc104, []OSCParsersColorRequest{sroCReset(sroTPalette(idx))})
			expectEqual(t, tm.colors.palette.original[idx], tm.colors.palette.current[idx])
			expect(t, tm.flags.dirty.palette, "t.flags.dirty.palette")
			// Only the entry that was reset changed back.
			sroExpectPaletteOnly(t, &tm, []uint8{5})
			expectEqual(t, sroC2, tm.colors.palette.current[5])
		}()
	}
	// The configured base palette is what a reset returns to.
	tm := sroNewWithDefaults(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	sroColorOp(t, &h, OSCParsersColorOperationOsc4, []OSCParsersColorRequest{sroCSet(sroTPalette(1), sroC1)})
	sroColorOp(t, &h, OSCParsersColorOperationOsc104, []OSCParsersColorRequest{sroCReset(sroTPalette(1))})
	expectEqual(t, ColorRGB{r: 0x10, g: 0x20, b: 0x30}, tm.colors.palette.current[1])
}

// Zig: test "term51: Handler.colorOperation: resetting an entry that was never set still marks the palette dirty"
func TestStreamReadonly_term51_Handler_colorOperation_resetting_an_entry_that_was_never_set_still_marks_the_palette_dirty(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewWithDefaults(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	expect(t, !tm.flags.dirty.palette, "!t.flags.dirty.palette")
	sroColorOp(t, &h, OSCParsersColorOperationOsc104, []OSCParsersColorRequest{sroCReset(sroTPalette(77))})
	expect(t, tm.flags.dirty.palette, "t.flags.dirty.palette")
	sroExpectPaletteOnly(t, &tm, []uint8{})
}

// Zig: test "term51: Handler.colorOperation: reset dynamic colors returns to the configured default or to unset"
func TestStreamReadonly_term51_Handler_colorOperation_reset_dynamic_colors_returns_to_the_configured_default_or_to_unset(t *testing.T) {
	testStdCapacity(t)
	// With configured defaults the override is set to the default.
	func() {
		tm := sroNewWithDefaults(t)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)

		sroColorOp(t, &h, OSCParsersColorOperationOsc10, []OSCParsersColorRequest{
			sroCSet(sroTDynamic(ColorDynamicForeground), sroC1),
			sroCSet(sroTDynamic(ColorDynamicBackground), sroC1),
			sroCSet(sroTDynamic(ColorDynamicCursor), sroC1),
		})
		sroColorOp(t, &h, OSCParsersColorOperationOsc110, []OSCParsersColorRequest{sroCReset(sroTDynamic(ColorDynamicForeground))})
		expectEqual(t, ColorRGB{r: 0xf1, g: 0xf2, b: 0xf3}, zigUnwrap(t, tm.colors.foreground.Get()))
		expectEqual(t, sroC1, zigUnwrap(t, tm.colors.background.Get()))
		expectEqual(t, sroC1, zigUnwrap(t, tm.colors.cursor.Get()))
		sroColorOp(t, &h, OSCParsersColorOperationOsc111, []OSCParsersColorRequest{sroCReset(sroTDynamic(ColorDynamicBackground))})
		expectEqual(t, ColorRGB{r: 0xb1, g: 0xb2, b: 0xb3}, zigUnwrap(t, tm.colors.background.Get()))
		expectEqual(t, sroC1, zigUnwrap(t, tm.colors.cursor.Get()))
		sroColorOp(t, &h, OSCParsersColorOperationOsc112, []OSCParsersColorRequest{sroCReset(sroTDynamic(ColorDynamicCursor))})
		expectEqual(t, ColorRGB{r: 0xc1, g: 0xc2, b: 0xc3}, zigUnwrap(t, tm.colors.cursor.Get()))
		expect(t, !tm.flags.dirty.palette, "!t.flags.dirty.palette")
	}()
	// Without defaults a reset leaves the color unset.
	func() {
		tm := sroNew(t, 10, 3)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)

		sroColorOp(t, &h, OSCParsersColorOperationOsc10, []OSCParsersColorRequest{
			sroCSet(sroTDynamic(ColorDynamicForeground), sroC1),
			sroCSet(sroTDynamic(ColorDynamicBackground), sroC1),
			sroCSet(sroTDynamic(ColorDynamicCursor), sroC1),
		})
		sroColorOp(t, &h, OSCParsersColorOperationOsc110, []OSCParsersColorRequest{sroCReset(sroTDynamic(ColorDynamicForeground))})
		expect(t, !tm.colors.foreground.Get().Valid, "t.colors.foreground.get() == null")
		expectEqual(t, sroC1, zigUnwrap(t, tm.colors.background.Get()))
		sroColorOp(t, &h, OSCParsersColorOperationOsc111, []OSCParsersColorRequest{sroCReset(sroTDynamic(ColorDynamicBackground))})
		expect(t, !tm.colors.background.Get().Valid, "t.colors.background.get() == null")
		expectEqual(t, sroC1, zigUnwrap(t, tm.colors.cursor.Get()))
		sroColorOp(t, &h, OSCParsersColorOperationOsc112, []OSCParsersColorRequest{sroCReset(sroTDynamic(ColorDynamicCursor))})
		expect(t, !tm.colors.cursor.Get().Valid, "t.colors.cursor.get() == null")
		// Resetting an unset color is fine.
		sroColorOp(t, &h, OSCParsersColorOperationOsc112, []OSCParsersColorRequest{sroCReset(sroTDynamic(ColorDynamicCursor))})
		expect(t, !tm.colors.cursor.Get().Valid, "t.colors.cursor.get() == null")
	}()
}

// Zig: test "term51: Handler.colorOperation: reset_palette resets exactly the masked entries"
func TestStreamReadonly_term51_Handler_colorOperation_reset_palette_resets_exactly_the_masked_entries(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewWithDefaults(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroColorOp(t, &h, OSCParsersColorOperationOsc4, []OSCParsersColorRequest{
		sroCSet(sroTPalette(3), sroC1),
		sroCSet(sroTPalette(17), sroC1),
		sroCSet(sroTPalette(1), sroC1),
		sroCSet(sroTPalette(255), sroC2),
		sroCSet(sroTPalette(0), sroC2),
	})
	sroExpectPaletteOnly(t, &tm, []uint8{0, 1, 3, 17, 255})
	// An entry that differs without being in the mask is left alone.
	tm.colors.palette.current[100] = sroBlue
	tm.flags.dirty.palette = false

	sroColorOp(t, &h, OSCParsersColorOperationOsc104, []OSCParsersColorRequest{sroCResetPalette})
	expect(t, tm.flags.dirty.palette, "t.flags.dirty.palette")
	expectEqual(t, 0, tm.colors.palette.mask.count())
	expectEqual(t, sroBlue, tm.colors.palette.current[100])
	for i := 0; i < 256; i++ {
		if i == 100 {
			continue
		}
		expectEqual(t, tm.colors.palette.original[i], tm.colors.palette.current[i])
	}
	expectEqual(t, ColorRGB{r: 0x10, g: 0x20, b: 0x30}, tm.colors.palette.current[1])

	// With an empty mask nothing is dirtied.
	tm.flags.dirty.palette = false
	sroColorOp(t, &h, OSCParsersColorOperationOsc104, []OSCParsersColorRequest{sroCResetPalette})
	expect(t, !tm.flags.dirty.palette, "!t.flags.dirty.palette")
	expectEqual(t, sroBlue, tm.colors.palette.current[100])
}

// Zig: test "term51: Handler.colorOperation: queries and reset_special do nothing"
func TestStreamReadonly_term51_Handler_colorOperation_queries_and_reset_special_do_nothing(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewWithDefaults(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	sroBusy(t, &tm, &h)
	tm.flags.dirty.palette = false

	before := sroSnapshot(t, &tm)
	sroColorOp(t, &h, OSCParsersColorOperationOsc4, []OSCParsersColorRequest{
		sroCQuery(sroTPalette(0)),
		sroCQuery(sroTPalette(3)),
		sroCQuery(sroTPalette(255)),
		sroCQuery(sroTDynamic(ColorDynamicForeground)),
		sroCQuery(sroTDynamic(ColorDynamicBackground)),
		sroCQuery(sroTDynamic(ColorDynamicCursor)),
		sroCQuery(sroTSpecial(ColorSpecialBold)),
		sroCResetSpecial,
	})
	sroColorOp(t, &h, OSCParsersColorOperationOsc105, []OSCParsersColorRequest{sroCResetSpecial})
	expectEqual(t, before, sroSnapshot(t, &tm))
	expect(t, !tm.flags.dirty.palette, "!t.flags.dirty.palette")
}

// Zig: test "term51: Handler.colorOperation: several requests apply in order"
func TestStreamReadonly_term51_Handler_colorOperation_several_requests_apply_in_order(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewWithDefaults(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	// Seven requests span several segments of the list.
	sroColorOp(t, &h, OSCParsersColorOperationOsc4, []OSCParsersColorRequest{
		sroCSet(sroTPalette(1), sroRed),
		sroCSet(sroTPalette(2), sroGreen),
		sroCReset(sroTPalette(1)),
		sroCSet(sroTPalette(3), sroBlue),
		sroCQuery(sroTPalette(4)),
		sroCSet(sroTPalette(2), sroC1),
		sroCSet(sroTDynamic(ColorDynamicForeground), sroC2),
	})
	sroExpectPaletteOnly(t, &tm, []uint8{2, 3})
	expectEqual(t, ColorRGB{r: 0x10, g: 0x20, b: 0x30}, tm.colors.palette.current[1])
	expectEqual(t, sroC1, tm.colors.palette.current[2])
	expectEqual(t, sroBlue, tm.colors.palette.current[3])
	expectEqual(t, sroC2, zigUnwrap(t, tm.colors.foreground.Get()))

	// A reset_palette in the middle only affects what was set before it.
	sroColorOp(t, &h, OSCParsersColorOperationOsc104, []OSCParsersColorRequest{
		sroCSet(sroTPalette(7), sroRed),
		sroCResetPalette,
		sroCSet(sroTPalette(8), sroGreen),
	})
	sroExpectPaletteOnly(t, &tm, []uint8{8})
	expectEqual(t, sroGreen, tm.colors.palette.current[8])
}

// Zig: test "term51: Handler.vt: color_operation runs the requests whatever the terminator"
func TestStreamReadonly_term51_Handler_vt_color_operation_runs_the_requests_whatever_the_terminator(t *testing.T) {
	testStdCapacity(t)
	for _, term := range []OSCTerminator{OSCTerminatorSt, OSCTerminatorBel} {
		func() {
			tm := sroNewWithDefaults(t)
			defer tm.Deinit()
			h := StreamReadonlyHandlerInit(&tm)

			var list OSCParsersColorList
			// Zig-only: defer list.deinit(testing.allocator)
			zigTry(t, list.append(sroCSet(sroTPalette(6), sroC1)))
			zigTry(t, list.append(sroCSet(sroTDynamic(ColorDynamicCursor), sroC2)))
			sroVt(t, &h, StreamAction{
				Tag:            StreamActionKeyColorOperation,
				ColorOperation: StreamActionColorOperation{op: OSCParsersColorOperationOsc4, requests: list, terminator: term},
			})
			expectEqual(t, sroC1, tm.colors.palette.current[6])
			expectEqual(t, sroC2, zigUnwrap(t, tm.colors.cursor.Get()))
			expect(t, tm.flags.dirty.palette, "t.flags.dirty.palette")

			// An empty list is accepted too.
			var empty OSCParsersColorList
			// Zig-only: defer empty.deinit(testing.allocator)
			sroVt(t, &h, StreamAction{
				Tag:            StreamActionKeyColorOperation,
				ColorOperation: StreamActionColorOperation{op: OSCParsersColorOperationOsc104, requests: empty, terminator: term},
			})
			sroExpectPaletteOnly(t, &tm, []uint8{6})
		}()
	}
}

// Zig: test "term51: Handler.kittyColorOperation: an empty request list changes nothing"
func TestStreamReadonly_term51_Handler_kittyColorOperation_an_empty_request_list_changes_nothing(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewWithDefaults(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	sroBusy(t, &tm, &h)
	tm.flags.dirty.palette = false

	before := sroSnapshot(t, &tm)
	sroKittyOp(t, &h, nil)
	expectEqual(t, before, sroSnapshot(t, &tm))
	expect(t, !tm.flags.dirty.palette, "!t.flags.dirty.palette")
}

// Zig: test "term51: Handler.kittyColorOperation: set palette entries at the boundaries"
func TestStreamReadonly_term51_Handler_kittyColorOperation_set_palette_entries_at_the_boundaries(t *testing.T) {
	testStdCapacity(t)
	for _, idx := range []uint8{0, 1, 15, 16, 127, 231, 232, 254, 255} {
		func() {
			tm := sroNewWithDefaults(t)
			defer tm.Deinit()
			h := StreamReadonlyHandlerInit(&tm)

			sroKittyOp(t, &h, []KittyColorOSCRequest{sroKSet(sroKPalette(idx), sroC1)})
			expectEqual(t, sroC1, tm.colors.palette.current[idx])
			sroExpectPaletteOnly(t, &tm, []uint8{idx})
			expect(t, tm.flags.dirty.palette, "t.flags.dirty.palette")
			expectEqual(t, Optional[ColorRGB]{}, tm.colors.foreground.override)
			expectEqual(t, Optional[ColorRGB]{}, tm.colors.background.override)
			expectEqual(t, Optional[ColorRGB]{}, tm.colors.cursor.override)

			sroKittyOp(t, &h, []KittyColorOSCRequest{sroKSet(sroKPalette(idx), sroC2)})
			expectEqual(t, sroC2, tm.colors.palette.current[idx])
			sroExpectPaletteOnly(t, &tm, []uint8{idx})
		}()
	}
}

// Zig: test "term51: Handler.kittyColorOperation: set foreground, background and cursor"
func TestStreamReadonly_term51_Handler_kittyColorOperation_set_foreground_background_and_cursor(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewWithDefaults(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	sroKittyOp(t, &h, []KittyColorOSCRequest{sroKSet(sroKSpecial(KittyColorSpecialForeground), sroC1)})
	expectEqual(t, sroC1, zigUnwrap(t, tm.colors.foreground.Get()))
	expectEqual(t, Optional[ColorRGB]{}, tm.colors.background.override)
	expectEqual(t, Optional[ColorRGB]{}, tm.colors.cursor.override)
	sroKittyOp(t, &h, []KittyColorOSCRequest{sroKSet(sroKSpecial(KittyColorSpecialBackground), sroC2)})
	expectEqual(t, sroC2, zigUnwrap(t, tm.colors.background.Get()))
	sroKittyOp(t, &h, []KittyColorOSCRequest{sroKSet(sroKSpecial(KittyColorSpecialCursor), sroRed)})
	expectEqual(t, sroRed, zigUnwrap(t, tm.colors.cursor.Get()))
	expectEqual(t, sroC1, zigUnwrap(t, tm.colors.foreground.Get()))
	expectEqual(t, sroC2, zigUnwrap(t, tm.colors.background.Get()))
	expect(t, !tm.flags.dirty.palette, "!t.flags.dirty.palette")
	sroExpectPaletteOnly(t, &tm, []uint8{})
}

// Zig: test "term51: Handler.kittyColorOperation: the other special keys are ignored for set, reset and query"
func TestStreamReadonly_term51_Handler_kittyColorOperation_the_other_special_keys_are_ignored_for_set_reset_and_query(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewWithDefaults(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	sroBusy(t, &tm, &h)
	tm.flags.dirty.palette = false

	before := sroSnapshot(t, &tm)
	for _, sp := range []KittyColorSpecial{
		KittyColorSpecialSelectionForeground,
		KittyColorSpecialSelectionBackground,
		KittyColorSpecialCursorText,
		KittyColorSpecialVisualBell,
		KittyColorSpecialSecondTransparentBackground,
	} {
		sroKittyOp(t, &h, []KittyColorOSCRequest{
			sroKSet(sroKSpecial(sp), sroC1),
			sroKReset(sroKSpecial(sp)),
			sroKQuery(sroKSpecial(sp)),
		})
	}
	expectEqual(t, before, sroSnapshot(t, &tm))
	expect(t, !tm.flags.dirty.palette, "!t.flags.dirty.palette")
}

// Zig: test "term51: Handler.kittyColorOperation: every special key, only foreground, background and cursor take effect"
func TestStreamReadonly_term51_Handler_kittyColorOperation_every_special_key_only_foreground_background_and_cursor_take_effect(t *testing.T) {
	testStdCapacity(t)
	for sp := KittyColorSpecial(0); sp < kittyColorSpecialCount; sp++ {
		func() {
			tm := sroNew(t, 10, 3)
			defer tm.Deinit()
			h := StreamReadonlyHandlerInit(&tm)

			sroKittyOp(t, &h, []KittyColorOSCRequest{sroKSet(sroKSpecial(sp), sroC1)})
			expectEqual(t, sp == KittyColorSpecialForeground, tm.colors.foreground.Get().Valid)
			expectEqual(t, sp == KittyColorSpecialBackground, tm.colors.background.Get().Valid)
			expectEqual(t, sp == KittyColorSpecialCursor, tm.colors.cursor.Get().Valid)
			expect(t, !tm.flags.dirty.palette, "!t.flags.dirty.palette")
			sroExpectPaletteOnly(t, &tm, []uint8{})

			sroKittyOp(t, &h, []KittyColorOSCRequest{sroKReset(sroKSpecial(sp))})
			expect(t, !tm.colors.foreground.Get().Valid, "t.colors.foreground.get() == null")
			expect(t, !tm.colors.background.Get().Valid, "t.colors.background.get() == null")
			expect(t, !tm.colors.cursor.Get().Valid, "t.colors.cursor.get() == null")
		}()
	}
	expectEqual(t, 8, kittyColorSpecialCount)
}

// Zig: test "term51: Handler.kittyColorOperation: reset palette entries"
func TestStreamReadonly_term51_Handler_kittyColorOperation_reset_palette_entries(t *testing.T) {
	testStdCapacity(t)
	for _, idx := range []uint8{0, 1, 200, 255} {
		func() {
			tm := sroNewWithDefaults(t)
			defer tm.Deinit()
			h := StreamReadonlyHandlerInit(&tm)

			sroKittyOp(t, &h, []KittyColorOSCRequest{
				sroKSet(sroKPalette(idx), sroC1),
				sroKSet(sroKPalette(5), sroC2),
			})
			tm.flags.dirty.palette = false
			sroKittyOp(t, &h, []KittyColorOSCRequest{sroKReset(sroKPalette(idx))})
			expectEqual(t, tm.colors.palette.original[idx], tm.colors.palette.current[idx])
			sroExpectPaletteOnly(t, &tm, []uint8{5})
			expect(t, tm.flags.dirty.palette, "t.flags.dirty.palette")
		}()
	}

	// Resetting an entry that was never set still marks the palette dirty.
	tm := sroNewWithDefaults(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	sroKittyOp(t, &h, []KittyColorOSCRequest{sroKReset(sroKPalette(99))})
	expect(t, tm.flags.dirty.palette, "t.flags.dirty.palette")
	sroExpectPaletteOnly(t, &tm, []uint8{})
}

// Zig: test "term51: Handler.kittyColorOperation: reset foreground, background and cursor"
func TestStreamReadonly_term51_Handler_kittyColorOperation_reset_foreground_background_and_cursor(t *testing.T) {
	testStdCapacity(t)
	func() {
		tm := sroNewWithDefaults(t)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)
		sroKittyOp(t, &h, []KittyColorOSCRequest{
			sroKSet(sroKSpecial(KittyColorSpecialForeground), sroC1),
			sroKSet(sroKSpecial(KittyColorSpecialBackground), sroC1),
			sroKSet(sroKSpecial(KittyColorSpecialCursor), sroC1),
		})
		sroKittyOp(t, &h, []KittyColorOSCRequest{sroKReset(sroKSpecial(KittyColorSpecialForeground))})
		expectEqual(t, ColorRGB{r: 0xf1, g: 0xf2, b: 0xf3}, zigUnwrap(t, tm.colors.foreground.Get()))
		expectEqual(t, sroC1, zigUnwrap(t, tm.colors.background.Get()))
		expectEqual(t, sroC1, zigUnwrap(t, tm.colors.cursor.Get()))
		sroKittyOp(t, &h, []KittyColorOSCRequest{sroKReset(sroKSpecial(KittyColorSpecialBackground))})
		expectEqual(t, ColorRGB{r: 0xb1, g: 0xb2, b: 0xb3}, zigUnwrap(t, tm.colors.background.Get()))
		expectEqual(t, sroC1, zigUnwrap(t, tm.colors.cursor.Get()))
		sroKittyOp(t, &h, []KittyColorOSCRequest{sroKReset(sroKSpecial(KittyColorSpecialCursor))})
		expectEqual(t, ColorRGB{r: 0xc1, g: 0xc2, b: 0xc3}, zigUnwrap(t, tm.colors.cursor.Get()))
		expect(t, !tm.flags.dirty.palette, "!t.flags.dirty.palette")
	}()
	func() {
		tm := sroNew(t, 10, 3)
		defer tm.Deinit()
		h := StreamReadonlyHandlerInit(&tm)
		sroKittyOp(t, &h, []KittyColorOSCRequest{
			sroKSet(sroKSpecial(KittyColorSpecialForeground), sroC1),
			sroKSet(sroKSpecial(KittyColorSpecialBackground), sroC1),
			sroKSet(sroKSpecial(KittyColorSpecialCursor), sroC1),
		})
		sroKittyOp(t, &h, []KittyColorOSCRequest{
			sroKReset(sroKSpecial(KittyColorSpecialForeground)),
			sroKReset(sroKSpecial(KittyColorSpecialBackground)),
			sroKReset(sroKSpecial(KittyColorSpecialCursor)),
		})
		expect(t, !tm.colors.foreground.Get().Valid, "t.colors.foreground.get() == null")
		expect(t, !tm.colors.background.Get().Valid, "t.colors.background.get() == null")
		expect(t, !tm.colors.cursor.Get().Valid, "t.colors.cursor.get() == null")
	}()
}

// Zig: test "term51: Handler.kittyColorOperation: queries do nothing and requests apply in order"
func TestStreamReadonly_term51_Handler_kittyColorOperation_queries_do_nothing_and_requests_apply_in_order(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewWithDefaults(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)

	before := sroSnapshot(t, &tm)
	sroKittyOp(t, &h, []KittyColorOSCRequest{
		sroKQuery(sroKPalette(0)),
		sroKQuery(sroKPalette(255)),
		sroKQuery(sroKSpecial(KittyColorSpecialForeground)),
		sroKQuery(sroKSpecial(KittyColorSpecialCursor)),
	})
	expectEqual(t, before, sroSnapshot(t, &tm))

	sroKittyOp(t, &h, []KittyColorOSCRequest{
		sroKSet(sroKPalette(1), sroRed),
		sroKSet(sroKPalette(2), sroGreen),
		sroKReset(sroKPalette(1)),
		sroKQuery(sroKPalette(2)),
		sroKSet(sroKPalette(3), sroBlue),
		sroKSet(sroKPalette(2), sroC1),
		sroKSet(sroKSpecial(KittyColorSpecialCursor), sroC2),
		sroKReset(sroKSpecial(KittyColorSpecialCursor)),
	})
	sroExpectPaletteOnly(t, &tm, []uint8{2, 3})
	expectEqual(t, sroC1, tm.colors.palette.current[2])
	expectEqual(t, sroBlue, tm.colors.palette.current[3])
	expectEqual(t, ColorRGB{r: 0xc1, g: 0xc2, b: 0xc3}, zigUnwrap(t, tm.colors.cursor.Get()))
}

// Zig: test "term51: Handler.vt: kitty_color_report runs the requests whatever the terminator"
func TestStreamReadonly_term51_Handler_vt_kitty_color_report_runs_the_requests_whatever_the_terminator(t *testing.T) {
	testStdCapacity(t)
	for _, term := range []OSCTerminator{OSCTerminatorSt, OSCTerminatorBel} {
		func() {
			tm := sroNewWithDefaults(t)
			defer tm.Deinit()
			h := StreamReadonlyHandlerInit(&tm)

			osc := KittyColorOSC{terminator: term}
			// Zig-only: defer osc.deinit(testing.allocator)
			osc.list = append(osc.list, sroKSet(sroKPalette(6), sroC1))
			osc.list = append(osc.list, sroKSet(sroKSpecial(KittyColorSpecialBackground), sroC2))
			sroVt(t, &h, StreamAction{Tag: StreamActionKeyKittyColorReport, KittyColorReport: osc})
			expectEqual(t, sroC1, tm.colors.palette.current[6])
			expectEqual(t, sroC2, zigUnwrap(t, tm.colors.background.Get()))
			expect(t, tm.flags.dirty.palette, "t.flags.dirty.palette")

			empty := KittyColorOSC{}
			sroVt(t, &h, StreamAction{Tag: StreamActionKeyKittyColorReport, KittyColorReport: empty})
			// Zig-only: empty.deinit(testing.allocator)
			sroExpectPaletteOnly(t, &tm, []uint8{6})
		}()
	}
}

// sroSend is term51DSend: feed bytes to a stream and then release the request
// list of a color operation that the parser still holds. Upstream never frees
// it (neither Parser.reset nor the readonly handler does), so lists that
// spilled out of the two preallocated entries would leak.
func sroSend(t *testing.T, s *StreamReadonlyStream, bytes string) {
	t.Helper()
	zigTry(t, s.NextSlice([]uint8(bytes)))
	// Zig-only: switch (s.parser.osc_parser.command) { .color_operation => |*v| v.requests.deinit(testing.allocator), else => {} }
	s.parser.osc_parser.command = OSCCommand{Tag: OSCCommandKeyInvalid}
}

// Zig: test "term51: Handler.colorOperation: the request list stays owned by the parser command (upstream leak)"
func TestStreamReadonly_term51_Handler_colorOperation_the_request_list_stays_owned_by_the_parser_command_upstream_leak(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Two requests fit the preallocated segment, three do not.
	zigTry(t, s.NextSlice([]uint8("\x1b]4;1;red;2;red\x07")))
	expectEqual(t, 2, s.parser.osc_parser.command.ColorOperation.requests.count())
	// Zig-only: try testing.expectEqual(@as(usize, 0), s.parser.osc_parser.command.color_operation.requests.dynamic_segments.len)
	zigTry(t, s.NextSlice([]uint8("\x1b]4;1;red;2;red;3;red\x07")))
	expectEqual(t, 3, s.parser.osc_parser.command.ColorOperation.requests.count())
	// The handler already applied them, and the heap segments are still
	// allocated, so the caller of the stream has to free them.
	expectEqual(t, sroRed, tm.colors.palette.current[3])
	// Zig-only: try testing.expect(s.parser.osc_parser.command.color_operation.requests.dynamic_segments.len > 0)
	// Zig-only: s.parser.osc_parser.command.color_operation.requests.deinit(testing.allocator)
	s.parser.osc_parser.command = OSCCommand{Tag: OSCCommandKeyInvalid}
}

// Zig: test "term51: Handler.colorOperation: OSC 4 through a stream"
func TestStreamReadonly_term51_Handler_colorOperation_OSC_4_through_a_stream(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewWithDefaults(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	sroSend(t, &s, "\x1b]4;1;rgb:ff/00/00;2;#00ff00\x1b\\")
	expectEqual(t, sroRed, tm.colors.palette.current[1])
	expectEqual(t, sroGreen, tm.colors.palette.current[2])
	sroExpectPaletteOnly(t, &tm, []uint8{1, 2})
	expect(t, tm.flags.dirty.palette, "t.flags.dirty.palette")

	// BEL terminator, a named color.
	sroSend(t, &s, "\x1b]4;3;rgb:00/00/ff\x07\x1b]4;9;red\x07")
	expectEqual(t, sroBlue, tm.colors.palette.current[3])
	expectEqual(t, sroRed, tm.colors.palette.current[9])
	sroExpectPaletteOnly(t, &tm, []uint8{1, 2, 3, 9})

	// Queries and special color indexes (256..260) have no effect.
	tm.flags.dirty.palette = false
	sroSend(t, &s, "\x1b]4;5;?\x07\x1b]4;256;red\x07\x1b]4;260;red\x07\x1b]4;258;?\x07")
	sroExpectPaletteOnly(t, &tm, []uint8{1, 2, 3, 9})
	expect(t, !tm.flags.dirty.palette, "!t.flags.dirty.palette")

	// A special entry does not stop the list, an unknown one (261) does.
	sroSend(t, &s, "\x1b]4;256;red;4;rgb:11/22/33\x07")
	expectEqual(t, sroC1, tm.colors.palette.current[4])
	sroSend(t, &s, "\x1b]4;261;red;5;rgb:11/22/33\x07")
	sroExpectPaletteOnly(t, &tm, []uint8{1, 2, 3, 4, 9})

	// An invalid color stops the list but keeps what came before.
	sroSend(t, &s, "\x1b]4;6;rgb:44/55/66;7;notacolor;8;red\x07")
	expectEqual(t, sroC2, tm.colors.palette.current[6])
	sroExpectPaletteOnly(t, &tm, []uint8{1, 2, 3, 4, 6, 9})

	// An invalid first color, a missing spec, an index out of range and a
	// non-numeric index all produce nothing.
	sroSend(t, &s, "\x1b]4;10;rgb:zz/00/00\x07")
	sroSend(t, &s, "\x1b]4;11\x07")
	sroSend(t, &s, "\x1b]4;999;red\x07")
	sroSend(t, &s, "\x1b]4;red;12;red\x07")
	sroSend(t, &s, "\x1b]4;-1;red\x07")
	sroSend(t, &s, "\x1b]4;12;rgb:ff/00\x07")
	sroSend(t, &s, "\x1b]4\x07")
	sroSend(t, &s, "\x1b]4;\x07")
	sroExpectPaletteOnly(t, &tm, []uint8{1, 2, 3, 4, 6, 9})

	// The one digit form is scaled, indexes with leading zeros are decimal.
	sroSend(t, &s, "\x1b]4;012;rgb:f/0/0\x07")
	expectEqual(t, sroRed, tm.colors.palette.current[12])
	// The highest index.
	sroSend(t, &s, "\x1b]4;255;rgb:11/22/33\x07")
	expectEqual(t, sroC1, tm.colors.palette.current[255])
}

// Zig: test "term51: Handler.colorOperation: OSC 5 special colors have no effect"
func TestStreamReadonly_term51_Handler_colorOperation_OSC_5_special_colors_have_no_effect(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewWithDefaults(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()
	sroBusy(t, &tm, s.handler)
	tm.flags.dirty.palette = false

	before := sroSnapshot(t, &tm)
	sroSend(t, &s, "\x1b]5;0;rgb:11/22/33;1;red;4;blue\x07")
	sroSend(t, &s, "\x1b]5;2;?\x07")
	expectEqual(t, before, sroSnapshot(t, &tm))
	expect(t, !tm.flags.dirty.palette, "!t.flags.dirty.palette")
}

// Zig: test "term51: Handler.colorOperation: OSC 10, 11 and 12 set successive dynamic colors"
func TestStreamReadonly_term51_Handler_colorOperation_OSC_10_11_and_12_set_successive_dynamic_colors(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 10, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	sroSend(t, &s, "\x1b]10;rgb:11/22/33\x1b\\")
	expectEqual(t, sroC1, zigUnwrap(t, tm.colors.foreground.Get()))
	expect(t, !tm.colors.background.Get().Valid, "t.colors.background.get() == null")
	expect(t, !tm.colors.cursor.Get().Valid, "t.colors.cursor.get() == null")

	sroSend(t, &s, "\x1b]11;rgb:44/55/66\x07")
	expectEqual(t, sroC2, zigUnwrap(t, tm.colors.background.Get()))
	sroSend(t, &s, "\x1b]12;#ff0000\x07")
	expectEqual(t, sroRed, zigUnwrap(t, tm.colors.cursor.Get()))

	// Several colors in one sequence continue with the next dynamic color.
	sroSend(t, &s, "\x1b]10;rgb:ff/00/00;rgb:00/ff/00;rgb:00/00/ff\x07")
	expectEqual(t, sroRed, zigUnwrap(t, tm.colors.foreground.Get()))
	expectEqual(t, sroGreen, zigUnwrap(t, tm.colors.background.Get()))
	expectEqual(t, sroBlue, zigUnwrap(t, tm.colors.cursor.Get()))
	sroSend(t, &s, "\x1b]11;rgb:11/22/33;rgb:44/55/66;rgb:ff/00/00\x07")
	expectEqual(t, sroRed, zigUnwrap(t, tm.colors.foreground.Get()))
	expectEqual(t, sroC1, zigUnwrap(t, tm.colors.background.Get()))
	expectEqual(t, sroC2, zigUnwrap(t, tm.colors.cursor.Get()))

	// The fourth color is the pointer foreground, which is ignored.
	sroSend(t, &s, "\x1b]10;rgb:00/00/01;rgb:00/00/02;rgb:00/00/03;rgb:00/00/04;rgb:00/00/05\x07")
	expectEqual(t, ColorRGB{r: 0, g: 0, b: 1}, zigUnwrap(t, tm.colors.foreground.Get()))
	expectEqual(t, ColorRGB{r: 0, g: 0, b: 2}, zigUnwrap(t, tm.colors.background.Get()))
	expectEqual(t, ColorRGB{r: 0, g: 0, b: 3}, zigUnwrap(t, tm.colors.cursor.Get()))

	// A query takes the place of a color without changing it.
	sroSend(t, &s, "\x1b]10;?;rgb:ff/00/00\x07")
	expectEqual(t, ColorRGB{r: 0, g: 0, b: 1}, zigUnwrap(t, tm.colors.foreground.Get()))
	expectEqual(t, sroRed, zigUnwrap(t, tm.colors.background.Get()))
	sroSend(t, &s, "\x1b]10;?\x07\x1b]11;?\x07\x1b]12;?\x07")
	expectEqual(t, sroRed, zigUnwrap(t, tm.colors.background.Get()))

	// An invalid spec stops the list.
	sroSend(t, &s, "\x1b]10;rgb:01/01/01;bogus;rgb:02/02/02\x07")
	expectEqual(t, ColorRGB{r: 1, g: 1, b: 1}, zigUnwrap(t, tm.colors.foreground.Get()))
	expectEqual(t, sroRed, zigUnwrap(t, tm.colors.background.Get()))
	expectEqual(t, ColorRGB{r: 0, g: 0, b: 3}, zigUnwrap(t, tm.colors.cursor.Get()))
	sroSend(t, &s, "\x1b]11;bogus;rgb:03/03/03\x07")
	expectEqual(t, sroRed, zigUnwrap(t, tm.colors.background.Get()))
	sroSend(t, &s, "\x1b]12;\x07\x1b]12;#12\x07")
	expectEqual(t, ColorRGB{r: 0, g: 0, b: 3}, zigUnwrap(t, tm.colors.cursor.Get()))

	// The palette is not involved.
	expect(t, !tm.flags.dirty.palette, "!t.flags.dirty.palette")
}

// Zig: test "term51: Handler.colorOperation: OSC 13 to 19 are ignored"
func TestStreamReadonly_term51_Handler_colorOperation_OSC_13_to_19_are_ignored(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewWithDefaults(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()
	sroBusy(t, &tm, s.handler)
	tm.flags.dirty.palette = false

	before := sroSnapshot(t, &tm)
	sroSend(t, &s, "\x1b]13;rgb:11/22/33\x07\x1b]14;red\x07\x1b]15;red\x07\x1b]16;red\x07")
	sroSend(t, &s, "\x1b]17;red\x07\x1b]18;red\x07\x1b]19;red\x07\x1b]13;?\x07")
	// OSC 13 with seven colors only reaches the tektronix and highlight colors.
	sroSend(t, &s, "\x1b]13;red;red;red;red;red;red;red\x07")
	expectEqual(t, before, sroSnapshot(t, &tm))
}

// Zig: test "term51: Handler.colorOperation: OSC 104 resets palette entries, all of them without parameters"
func TestStreamReadonly_term51_Handler_colorOperation_OSC_104_resets_palette_entries_all_of_them_without_parameters(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewWithDefaults(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	sroSend(t, &s, "\x1b]4;1;red;2;red;3;red;4;red;5;red\x07")
	sroExpectPaletteOnly(t, &tm, []uint8{1, 2, 3, 4, 5})

	sroSend(t, &s, "\x1b]104;1\x07")
	sroExpectPaletteOnly(t, &tm, []uint8{2, 3, 4, 5})
	expectEqual(t, ColorRGB{r: 0x10, g: 0x20, b: 0x30}, tm.colors.palette.current[1])

	// Several entries; empty parameters, out of range and non numeric ones
	// are skipped, special ones (256..260) have no effect.
	sroSend(t, &s, "\x1b]104;2;;x;3;999;256;260\x07")
	sroExpectPaletteOnly(t, &tm, []uint8{4, 5})
	tm.flags.dirty.palette = false
	sroSend(t, &s, "\x1b]104;256\x07")
	sroExpectPaletteOnly(t, &tm, []uint8{4, 5})
	expect(t, !tm.flags.dirty.palette, "!t.flags.dirty.palette")

	// Without parameters everything is reset.
	sroSend(t, &s, "\x1b]104\x07")
	sroExpectPaletteOnly(t, &tm, []uint8{})
	expect(t, tm.flags.dirty.palette, "t.flags.dirty.palette")

	// An empty parameter list also resets everything, and so does a list
	// with only invalid entries (upstream behaviour).
	sroSend(t, &s, "\x1b]4;1;red;2;red\x07")
	sroSend(t, &s, "\x1b]104;\x07")
	sroExpectPaletteOnly(t, &tm, []uint8{})
	sroSend(t, &s, "\x1b]4;1;red;2;red\x07")
	sroSend(t, &s, "\x1b]104;abc\x07")
	sroExpectPaletteOnly(t, &tm, []uint8{})
	sroSend(t, &s, "\x1b]4;1;red;2;red\x07")
	sroSend(t, &s, "\x1b]104;999\x07")
	sroExpectPaletteOnly(t, &tm, []uint8{})

	// OSC 105 (reset special colors) does nothing.
	sroSend(t, &s, "\x1b]4;1;red\x07")
	tm.flags.dirty.palette = false
	sroSend(t, &s, "\x1b]105\x07\x1b]105;0\x07")
	sroExpectPaletteOnly(t, &tm, []uint8{1})
	expect(t, !tm.flags.dirty.palette, "!t.flags.dirty.palette")
}

// Zig: test "term51: Handler.colorOperation: OSC 110, 111 and 112 reset the dynamic colors, 113 to 119 are ignored"
func TestStreamReadonly_term51_Handler_colorOperation_OSC_110_111_and_112_reset_the_dynamic_colors_113_to_119_are_ignored(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewWithDefaults(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	sroSend(t, &s, "\x1b]10;red;green;blue\x07")
	expectEqual(t, sroRed, zigUnwrap(t, tm.colors.foreground.Get()))
	sroSend(t, &s, "\x1b]110\x07")
	expectEqual(t, ColorRGB{r: 0xf1, g: 0xf2, b: 0xf3}, zigUnwrap(t, tm.colors.foreground.Get()))
	expectEqual(t, sroGreen, zigUnwrap(t, tm.colors.background.Get()))
	sroSend(t, &s, "\x1b]111\x1b\\")
	expectEqual(t, ColorRGB{r: 0xb1, g: 0xb2, b: 0xb3}, zigUnwrap(t, tm.colors.background.Get()))
	expectEqual(t, sroBlue, zigUnwrap(t, tm.colors.cursor.Get()))
	sroSend(t, &s, "\x1b]112\x07")
	expectEqual(t, ColorRGB{r: 0xc1, g: 0xc2, b: 0xc3}, zigUnwrap(t, tm.colors.cursor.Get()))

	// A parameter makes the whole request empty, so nothing is reset.
	sroSend(t, &s, "\x1b]10;red;green;blue\x07")
	sroSend(t, &s, "\x1b]110;1\x07\x1b]111;x\x07\x1b]112;\x07")
	expectEqual(t, sroRed, zigUnwrap(t, tm.colors.foreground.Get()))
	expectEqual(t, sroGreen, zigUnwrap(t, tm.colors.background.Get()))
	// The empty parameter is skipped by the tokenizer so this one resets.
	expectEqual(t, ColorRGB{r: 0xc1, g: 0xc2, b: 0xc3}, zigUnwrap(t, tm.colors.cursor.Get()))

	// The others change nothing.
	before := sroSnapshot(t, &tm)
	sroSend(t, &s, "\x1b]113\x07\x1b]114\x07\x1b]115\x07\x1b]116\x07\x1b]117\x07\x1b]118\x07\x1b]119\x07")
	expectEqual(t, before, sroSnapshot(t, &tm))
	expect(t, !tm.flags.dirty.palette, "!t.flags.dirty.palette")
}

// Zig: test "term51: Handler.colorOperation: without an allocator the color OSCs are dropped"
func TestStreamReadonly_term51_Handler_colorOperation_without_an_allocator_the_color_OSCs_are_dropped(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewWithDefaults(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInit[*StreamReadonlyHandler](&h)
	defer s.Deinit()
	sroBusy(t, &tm, s.handler)
	tm.flags.dirty.palette = false

	before := sroSnapshot(t, &tm)
	sroSend(t, &s, "\x1b]4;1;red\x07")
	sroSend(t, &s, "\x1b]104\x07\x1b]104;1\x07")
	sroSend(t, &s, "\x1b]10;red\x07\x1b]11;red\x07\x1b]12;red\x07")
	sroSend(t, &s, "\x1b]110\x07\x1b]111\x07\x1b]112\x07")
	sroSend(t, &s, "\x1b]21;1=rgb:ff/00/00;foreground=red\x07")
	expectEqual(t, before, sroSnapshot(t, &tm))
	expect(t, !tm.flags.dirty.palette, "!t.flags.dirty.palette")

	// Everything else still works.
	sroSend(t, &s, "\x1b[1;1HZ")
	sroExpectText(t, &tm, "Zello\nworld")
}

// Zig: test "term51: Handler.kittyColorOperation: OSC 21 through a stream"
func TestStreamReadonly_term51_Handler_kittyColorOperation_OSC_21_through_a_stream(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewWithDefaults(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("\x1b]21;1=rgb:ff/00/00;foreground=#112233\x1b\\")))
	expectEqual(t, sroRed, tm.colors.palette.current[1])
	expectEqual(t, sroC1, zigUnwrap(t, tm.colors.foreground.Get()))
	sroExpectPaletteOnly(t, &tm, []uint8{1})
	expect(t, tm.flags.dirty.palette, "t.flags.dirty.palette")

	zigTry(t, s.NextSlice([]uint8("\x1b]21;background=rgb:44/55/66;cursor=rgb:ff/00/00;255=rgb:00/00/ff\x07")))
	expectEqual(t, sroC2, zigUnwrap(t, tm.colors.background.Get()))
	expectEqual(t, sroRed, zigUnwrap(t, tm.colors.cursor.Get()))
	expectEqual(t, sroBlue, tm.colors.palette.current[255])
	sroExpectPaletteOnly(t, &tm, []uint8{1, 255})

	// Queries do nothing.
	tm.flags.dirty.palette = false
	zigTry(t, s.NextSlice([]uint8("\x1b]21;1=?;foreground=?;background=?;cursor=?;7=?\x07")))
	expectEqual(t, sroRed, tm.colors.palette.current[1])
	expectEqual(t, sroC1, zigUnwrap(t, tm.colors.foreground.Get()))
	expect(t, !tm.flags.dirty.palette, "!t.flags.dirty.palette")

	// Other special keys do nothing.
	before := sroSnapshot(t, &tm)
	zigTry(t, s.NextSlice([]uint8("\x1b]21;selection_foreground=#112233;selection_background=#112233;cursor_text=#112233\x07")))
	zigTry(t, s.NextSlice([]uint8("\x1b]21;visual_bell=#112233;second_transparent_background=#112233\x07")))
	zigTry(t, s.NextSlice([]uint8("\x1b]21;selection_foreground=;cursor_text=;visual_bell=\x07")))
	expectEqual(t, before, sroSnapshot(t, &tm))

	// An empty value resets.
	zigTry(t, s.NextSlice([]uint8("\x1b]21;1=\x07")))
	sroExpectPaletteOnly(t, &tm, []uint8{255})
	expectEqual(t, ColorRGB{r: 0x10, g: 0x20, b: 0x30}, tm.colors.palette.current[1])
	zigTry(t, s.NextSlice([]uint8("\x1b]21;foreground=;background=;cursor=\x07")))
	expectEqual(t, ColorRGB{r: 0xf1, g: 0xf2, b: 0xf3}, zigUnwrap(t, tm.colors.foreground.Get()))
	expectEqual(t, ColorRGB{r: 0xb1, g: 0xb2, b: 0xb3}, zigUnwrap(t, tm.colors.background.Get()))
	expectEqual(t, ColorRGB{r: 0xc1, g: 0xc2, b: 0xc3}, zigUnwrap(t, tm.colors.cursor.Get()))
}

// Zig: test "term51: Handler.kittyColorOperation: OSC 21 skips invalid keys and values but keeps going"
func TestStreamReadonly_term51_Handler_kittyColorOperation_OSC_21_skips_invalid_keys_and_values_but_keeps_going(t *testing.T) {
	testStdCapacity(t)
	tm := sroNewWithDefaults(t)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// Out of range index, unknown name, wrong case, bad color, missing key:
	// each is skipped and the following ones are still applied.
	zigTry(t, s.NextSlice([]uint8("\x1b]21;256=red;1=rgb:ff/00/00\x07")))
	expectEqual(t, sroRed, tm.colors.palette.current[1])
	sroExpectPaletteOnly(t, &tm, []uint8{1})
	zigTry(t, s.NextSlice([]uint8("\x1b]21;nothing=red;2=rgb:00/ff/00\x07")))
	sroExpectPaletteOnly(t, &tm, []uint8{1, 2})
	zigTry(t, s.NextSlice([]uint8("\x1b]21;Foreground=red;3=rgb:00/00/ff\x07")))
	sroExpectPaletteOnly(t, &tm, []uint8{1, 2, 3})
	expect(t, !tm.colors.foreground.override.Valid, "t.colors.foreground.override == null")
	zigTry(t, s.NextSlice([]uint8("\x1b]21;4=notacolor;5=rgb:11/22/33\x07")))
	sroExpectPaletteOnly(t, &tm, []uint8{1, 2, 3, 5})
	zigTry(t, s.NextSlice([]uint8("\x1b]21;=red;6=rgb:11/22/33\x07")))
	sroExpectPaletteOnly(t, &tm, []uint8{1, 2, 3, 5, 6})
	zigTry(t, s.NextSlice([]uint8("\x1b]21;\x07\x1b]21;;;\x07\x1b]21\x07")))
	sroExpectPaletteOnly(t, &tm, []uint8{1, 2, 3, 5, 6})

	// The value may have surrounding spaces.
	zigTry(t, s.NextSlice([]uint8("\x1b]21;7= rgb:11/22/33 \x07")))
	expectEqual(t, sroC1, tm.colors.palette.current[7])
}

// Zig: test "term51: Handler.vt: charset sequences through a stream"
func TestStreamReadonly_term51_Handler_vt_charset_sequences_through_a_stream(t *testing.T) {
	testStdCapacity(t)
	tm := sroNew(t, 20, 3)
	defer tm.Deinit()
	h := StreamReadonlyHandlerInit(&tm)
	s := StreamInitAlloc[*StreamReadonlyHandler](sroAllocator, &h)
	defer s.Deinit()

	// ESC ( 0 designates DEC special graphics into G0.
	zigTry(t, s.NextSlice([]uint8("\x1b(0`\x1b(B`")))
	sroExpectText(t, &tm, "◆`")
	// SO/SI are locking shifts to G1/G0.
	zigTry(t, s.NextSlice([]uint8("\x1b)0\x0e`\x0f`")))
	sroExpectText(t, &tm, "◆`◆`")
	expectEqual(t, CharsetsSlotsG0, tm.screens.active.charset.gl)
	// ESC N is a single shift to G2.
	zigTry(t, s.NextSlice([]uint8("\x1b*0\x1bN``")))
	sroExpectText(t, &tm, "◆`◆`◆`")
	expectEqual(t, CharsetsSlotsG0, tm.screens.active.charset.gl)
	// ESC n is a locking shift to G2, ESC o to G3.
	zigTry(t, s.NextSlice([]uint8("\x1bn`\x1bo`")))
	expectEqual(t, CharsetsSlotsG3, tm.screens.active.charset.gl)
	expectEqual(t, CharsetsSlotsG2, tm.screens.active.charset.gr)
}
