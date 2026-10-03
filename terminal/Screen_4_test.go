// Tests translated one for one from src/terminal/Screen.zig (Ghostty v1.3.1
// plus the term51 additional tests), from `test "term51: select: untracked
// selection becomes tracked and keeps bounds"` to the end of the file. Each Go
// test carries the exact Zig title in a "// Zig: test" comment. Helpers are
// prefixed sc4.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualStrings keep Zig's names and Zig's (expected,
//     actual) argument order. `try` on a call that returns an error is
//     zigTry(t, err); `x.?` is zigUnwrap;
//     `try testing.expectError(error.X, call)` is sc4ExpectError(t, ErrX,
//     <the error of the call>).
//   - Zig's helpers term51R* are the unexported sc4R* functions below.
//     `Screen.init(alloc, ...)` is sc4Init (ScreenInit with the Zig defaults
//     written out, plus zigTry); the screen is a *Screen so that Zig's `&s`
//     is `s`. `s.testWriteString("...")` is sc4Write.
//   - Every test starts with testStdCapacity(t): Zig's unit tests run with
//     grapheme_bytes 512, and the page-boundary tests depend on the rows per
//     page.
//   - `defer map.deinit(alloc)` is `defer func() { map_.Deinit() }()`: Deinit
//     has a value receiver, so a plain `defer map_.Deinit()` would copy the
//     map before it is filled.
//   - A `defer s.deinit()` inside a loop body runs at the end of each
//     iteration in Zig; the loop bodies here are function literals so the Go
//     defer behaves the same way.
//   - `x[x.len]` on a `[:0]const u8` is sc4Sentinel (the slice needs spare
//     capacity covering the terminating zero byte, see docs/mapping.md).
//   - Zig-only: `const alloc = std.testing.allocator;`, `defer alloc.free(...)`
//     and the allocator arguments are allocator bookkeeping with no Go
//     counterpart; they are dropped without a per-test note.
//   - `std.Io.Writer.Allocating.toOwnedSlice` has no wireframe name: the
//     helper sc4RDump copies `written()` instead. `std.Io.Writer.fixed(&buf)`
//     is `oscWriter{buffer: buf[:]}` and `w.buffered()` is
//     `w.buffer[:w.end]` (osc.go).
//   - Tests that were not translated are listed in
//     scripts/not-translated-sc4.txt.

package terminal

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

// sc4ExpectError is Zig's `try testing.expectError(expected, actual)`.
func sc4ExpectError(t *testing.T, want error, got error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

// sc4ExpectPanic runs f and fails unless it panics. The wireframe's
// not-implemented stub does not count: a test that only reaches the stub must
// fail until the implementation panics for the right reason. Zig's tests
// cannot test an assert or `unreachable`: the test would abort.
func sc4ExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("expected a panic, got none")
		}
		if s, ok := r.(string); ok && s == notImplemented {
			t.Fatalf("panicked with the not-implemented stub, not with the expected panic")
		}
	}()
	f()
}

// sc4Sentinel is `s.ptr[s.len]`, the byte after the end of a [:0]const u8,
// which must be the NUL.
func sc4Sentinel(t *testing.T, s []uint8) uint8 {
	t.Helper()
	if cap(s) <= len(s) {
		t.Fatalf("no sentinel byte after a slice of length %d (capacity %d)", len(s), cap(s))
	}
	return s[:len(s)+1][len(s)]
}

// sc4Init is `try Screen.init(alloc, .{ .cols = cols, .rows = rows,
// .max_scrollback = max_scrollback })`: kitty_image_storage_limit is the Zig
// default, 320 * 1000 * 1000.
func sc4Init(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt, max_scrollback uint64) *Screen {
	t.Helper()
	s, err := ScreenInit(ScreenOptions{
		cols:                      cols,
		rows:                      rows,
		max_scrollback:            max_scrollback,
		kitty_image_storage_limit: 320 * 1000 * 1000,
	})
	zigTry(t, err)
	return &s
}

// sc4Write is `try s.testWriteString(text)`.
func sc4Write(t *testing.T, s *Screen, text string) {
	t.Helper()
	zigTry(t, s.TestWriteString([]uint8(text)))
}

// sc4R_boundary is term51R_boundary: the default word boundary codepoints
// used by Ghostty's config.
var sc4R_boundary = []uint32{
	0, ' ', '\t', '\'', '"', '│', '`', '|', ':', ';',
	',', '(', ')', '[', ']', '{', '}', '<', '>', '$',
}

// sc4RPin is term51RPin: the pin at the given screen coordinates.
func sc4RPin(t *testing.T, s *Screen, x SizeCellCountInt, y uint32) PageListPin {
	t.Helper()
	return zigUnwrap(t, s.pages.Pin(sc4ScreenPt(x, y)))
}

// sc4ScreenPt is the Zig literal `point.Point{ .screen = .{ .x = x, .y = y } }`.
func sc4ScreenPt(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagScreen, Screen: PointCoordinate{x: x, y: y}}
}

// sc4RPt is term51RPt: the screen point of a pin.
func sc4RPt(t *testing.T, s *Screen, p PageListPin) Point {
	t.Helper()
	return zigUnwrap(t, s.pages.PointFromPin(PointTagScreen, p))
}

// sc4RExpectSel is term51RExpectSel: expect a non-null, untracked,
// non-rectangle selection with the given screen coordinates for start and end.
func sc4RExpectSel(
	t *testing.T,
	s *Screen,
	sel_ Optional[Selection],
	x0 SizeCellCountInt,
	y0 uint32,
	x1 SizeCellCountInt,
	y1 uint32,
) {
	t.Helper()
	expect(t, sel_.Valid, "sel_ != null")
	sel := sel_.Value
	expect(t, !sel.Tracked(), "!sel.tracked()")
	expect(t, !sel.rectangle, "!sel.rectangle")
	expectEqual(t, sc4ScreenPt(x0, y0), sc4RPt(t, s, sel.Start()))
	expectEqual(t, sc4ScreenPt(x1, y1), sc4RPt(t, s, sel.End()))
}

// sc4RSelStr is term51RSelStr: selectionString for the given screen
// coordinates.
func sc4RSelStr(
	t *testing.T,
	s *Screen,
	x0 SizeCellCountInt,
	y0 uint32,
	x1 SizeCellCountInt,
	y1 uint32,
	rect bool,
	trim bool,
) []uint8 {
	t.Helper()
	text, err := s.SelectionString(ScreenSelectionString{
		sel: SelectionInit(
			sc4RPin(t, s, x0, y0),
			sc4RPin(t, s, x1, y1),
			rect,
		),
		trim: trim,
	})
	zigTry(t, err)
	return text
}

// sc4RFillFirstPage is term51RFillFirstPage: write enough newlines that the
// cursor sits on the last row of the first page, so the following writes land
// in a second page.
func sc4RFillFirstPage(t *testing.T, s *Screen) uint32 {
	t.Helper()
	first_page_size := uint32(s.pages.pages.first.data.capacity.rows)
	s.pages.pages.first.data.PauseIntegrityChecks(true)
	for i := uint32(0); i < first_page_size-1; i++ {
		sc4Write(t, s, "\n")
	}
	s.pages.pages.first.data.PauseIntegrityChecks(false)
	return first_page_size
}

// sc4RCursorTo is term51RCursorTo: move the cursor to the cell at screen row y.
func sc4RCursorTo(t *testing.T, s *Screen, x SizeCellCountInt, y uint32) {
	t.Helper()
	top := zigUnwrap(t, s.pages.PointFromPin(PointTagScreen, s.pages.GetTopLeft(PointTagActive))).Screen.y
	s.CursorAbsolute(x, SizeCellCountInt(y-top))
}

// sc4Prompt is the Zig literal `.{ .prompt = kind }` for cursorSetSemanticContent.
func sc4Prompt(kind OSCParsersSemanticPromptPromptKind) ScreenCursorSetSemanticContent {
	return ScreenCursorSetSemanticContent{Tag: ScreenCursorSetSemanticContentTagPrompt, Prompt: kind}
}

// sc4Input is the Zig literal `.{ .input = clear }` for cursorSetSemanticContent.
func sc4Input(clear ScreenCursorSetSemanticContentInput) ScreenCursorSetSemanticContent {
	return ScreenCursorSetSemanticContent{Tag: ScreenCursorSetSemanticContentTagInput, Input: clear}
}

// sc4Output is the Zig literal `.output` for cursorSetSemanticContent.
func sc4Output() ScreenCursorSetSemanticContent {
	return ScreenCursorSetSemanticContent{Tag: ScreenCursorSetSemanticContentTagOutput}
}

// sc4RWritePrompt is term51RWritePrompt: write a prompt followed by input,
// both with explicit semantic content.
func sc4RWritePrompt(t *testing.T, s *Screen, prompt string, input string) {
	t.Helper()
	s.CursorSetSemanticContent(sc4Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc4Write(t, s, prompt)
	s.CursorSetSemanticContent(sc4Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc4Write(t, s, input)
}

// sc4RMove is term51RMove.
func sc4RMove(left uint, right uint) ScreenPromptClickMove {
	return ScreenPromptClickMove{left: left, right: right}
}

// sc4Click is the Zig literal `.{ .cl = cl }` of a SemanticClick.
func sc4Click(cl OSCParsersSemanticPromptClick) ScreenSemanticPromptSemanticClick {
	return ScreenSemanticPromptSemanticClick{Tag: ScreenSemanticPromptSemanticClickCl, Cl: cl}
}

// sc4Line is `s.selectLine(.{ .pin = pin })`.
func sc4Line(s *Screen, pin PageListPin) Optional[Selection] {
	return s.SelectLine(screenSelectLineDefault(pin))
}

// sc4LineWS is `s.selectLine(.{ .pin = pin, .whitespace = ws })`.
func sc4LineWS(s *Screen, pin PageListPin, ws Optional[[]uint32]) Optional[Selection] {
	opts := screenSelectLineDefault(pin)
	opts.whitespace = ws
	return s.SelectLine(opts)
}

// sc4RDump is term51RDump: dumpString into an allocating writer and return
// the bytes.
func sc4RDump(t *testing.T, s *Screen, tl PageListPin, br Optional[PageListPin], unwrap bool) []uint8 {
	t.Helper()
	aw := oscWriterAllocatingInit()
	defer aw.deinit()
	zigTry(t, s.DumpString(&aw.writer, ScreenDumpStringOptions{tl: tl, br: br, unwrap: unwrap}))
	// toOwnedSlice
	return append([]uint8(nil), aw.written()...)
}

// sc4Tag is the Zig `.{ .screen = .{} }` family: a point with only a tag.
func sc4Tag(tag PointTag) Point {
	return Point{Tag: tag}
}

// Zig: test "term51: select: untracked selection becomes tracked and keeps bounds"
func TestSc4_select_untracked_selection_becomes_tracked_and_keeps_bounds(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 5, 0)
	defer s.Deinit()
	sc4Write(t, s, "hello\nworld")

	input := SelectionInit(sc4RPin(t, s, 3, 1), sc4RPin(t, s, 1, 0), false)
	expect(t, !input.Tracked(), "!input.tracked()")
	s.dirty.selection = false
	zigTry(t, s.Select(Some(input)))

	// The caller's value stays untracked, the screen holds a tracked copy.
	expect(t, !input.Tracked(), "!input.tracked()")
	sel := zigUnwrap(t, s.selection)
	expect(t, sel.Tracked(), "sel.tracked()")
	expect(t, !sel.rectangle, "!sel.rectangle")
	expect(t, s.dirty.selection, "s.dirty.selection")

	// Start/end order is preserved (a backwards selection stays backwards).
	expectEqual(t, sc4ScreenPt(3, 1), sc4RPt(t, s, sel.Start()))
	expectEqual(t, sc4ScreenPt(1, 0), sc4RPt(t, s, sel.End()))

	// The tracked pins are distinct objects from the input pins.
	expect(t, sel.bounds.Tracked.start != sel.bounds.Tracked.end, "sel.bounds.tracked.start != sel.bounds.tracked.end")
}

// Zig: test "term51: select: rectangle flag is preserved"
func TestSc4_select_rectangle_flag_is_preserved(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 5, 0)
	defer s.Deinit()
	sc4Write(t, s, "hello\nworld")

	zigTry(t, s.Select(Some(SelectionInit(sc4RPin(t, s, 0, 0), sc4RPin(t, s, 2, 1), true))))
	expect(t, zigUnwrap(t, s.selection).rectangle, "s.selection.?.rectangle")
	zigTry(t, s.Select(Some(SelectionInit(sc4RPin(t, s, 0, 0), sc4RPin(t, s, 2, 1), false))))
	expect(t, !zigUnwrap(t, s.selection).rectangle, "!s.selection.?.rectangle")
}

// Zig: test "term51: select: tracked selection is adopted without new pins"
func TestSc4_select_tracked_selection_is_adopted_without_new_pins(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 5, 0)
	defer s.Deinit()
	sc4Write(t, s, "hello")

	base := s.pages.CountTrackedPins()
	input := SelectionInit(sc4RPin(t, s, 0, 0), sc4RPin(t, s, 4, 0), false)
	tracked, err := input.Track(s)
	zigTry(t, err)
	expectEqual(t, base+2, s.pages.CountTrackedPins())

	s.dirty.selection = false
	zigTry(t, s.Select(Some(tracked)))
	expect(t, s.dirty.selection, "s.dirty.selection")

	// No additional pins, and the very same tracked pins are adopted.
	expectEqual(t, base+2, s.pages.CountTrackedPins())
	expect(t, zigUnwrap(t, s.selection).bounds.Tracked.start == tracked.bounds.Tracked.start, "s.selection.?.bounds.tracked.start == tracked.bounds.tracked.start")
	expect(t, zigUnwrap(t, s.selection).bounds.Tracked.end == tracked.bounds.Tracked.end, "s.selection.?.bounds.tracked.end == tracked.bounds.tracked.end")
}

// Zig: test "term51: select: replacing a selection untracks the previous one"
func TestSc4_select_replacing_a_selection_untracks_the_previous_one(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 5, 0)
	defer s.Deinit()
	sc4Write(t, s, "hello\nworld")

	base := s.pages.CountTrackedPins()
	zigTry(t, s.Select(Some(SelectionInit(sc4RPin(t, s, 0, 0), sc4RPin(t, s, 4, 0), false))))
	expectEqual(t, base+2, s.pages.CountTrackedPins())

	s.dirty.selection = false
	zigTry(t, s.Select(Some(SelectionInit(sc4RPin(t, s, 0, 1), sc4RPin(t, s, 4, 1), false))))
	expect(t, s.dirty.selection, "s.dirty.selection")
	expectEqual(t, base+2, s.pages.CountTrackedPins())
	expectEqual(t, sc4ScreenPt(0, 1), sc4RPt(t, s, zigUnwrap(t, s.selection).Start()))
	expectEqual(t, sc4ScreenPt(4, 1), sc4RPt(t, s, zigUnwrap(t, s.selection).End()))

	// Selecting identical bounds again still marks the screen dirty.
	s.dirty.selection = false
	zigTry(t, s.Select(Some(SelectionInit(sc4RPin(t, s, 0, 1), sc4RPin(t, s, 4, 1), false))))
	expect(t, s.dirty.selection, "s.dirty.selection")
	expectEqual(t, base+2, s.pages.CountTrackedPins())

	// Clean up leaves no tracked pins behind.
	zigTry(t, s.Select(None[Selection]()))
	expectEqual(t, base, s.pages.CountTrackedPins())
}

// term51: select: allocation failure leaves prior selection untouched: not
// translated (FailingAllocator), see scripts/not-translated-sc4.txt.

// Zig: test "term51: clearSelection: untracks pins and sets dirty"
func TestSc4_clearSelection_untracks_pins_and_sets_dirty(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 5, 0)
	defer s.Deinit()
	sc4Write(t, s, "hello")

	base := s.pages.CountTrackedPins()
	zigTry(t, s.Select(Some(SelectionInit(sc4RPin(t, s, 0, 0), sc4RPin(t, s, 4, 0), true))))
	expectEqual(t, base+2, s.pages.CountTrackedPins())

	s.dirty.selection = false
	s.ClearSelection()
	expect(t, !s.selection.Valid, "s.selection == null")
	expect(t, s.dirty.selection, "s.dirty.selection")
	expectEqual(t, base, s.pages.CountTrackedPins())
}

// Zig: test "term51: clearSelection: without selection does not set dirty"
func TestSc4_clearSelection_without_selection_does_not_set_dirty(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 5, 0)
	defer s.Deinit()

	base := s.pages.CountTrackedPins()
	s.ClearSelection()
	expect(t, !s.selection.Valid, "s.selection == null")
	expect(t, !s.dirty.selection, "!s.dirty.selection")
	expectEqual(t, base, s.pages.CountTrackedPins())

	// A dirty flag that is already set is not cleared.
	s.dirty.selection = true
	s.ClearSelection()
	expect(t, s.dirty.selection, "s.dirty.selection")
}

// Zig: test "term51: clearSelection: is idempotent"
func TestSc4_clearSelection_is_idempotent(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 5, 0)
	defer s.Deinit()
	sc4Write(t, s, "hello")

	base := s.pages.CountTrackedPins()
	zigTry(t, s.Select(Some(SelectionInit(sc4RPin(t, s, 0, 0), sc4RPin(t, s, 4, 0), false))))
	s.ClearSelection()
	s.dirty.selection = false
	s.ClearSelection()
	expect(t, !s.selection.Valid, "s.selection == null")
	expect(t, !s.dirty.selection, "!s.dirty.selection")
	expectEqual(t, base, s.pages.CountTrackedPins())
}

// Zig: test "term51: selectionString: basic single row is sentinel terminated"
func TestSc4_selectionString_basic_single_row_is_sentinel_terminated(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 20, 5, 0)
	defer s.Deinit()
	sc4Write(t, s, "hello world")

	text := sc4RSelStr(t, s, 0, 0, 4, 0, false, true)
	expectEqualStrings(t, "hello", string(text))
	expectEqual(t, uint8(0), sc4Sentinel(t, text))

	// Single cell.
	one := sc4RSelStr(t, s, 6, 0, 6, 0, false, true)
	expectEqualStrings(t, "w", string(one))
}

// Zig: test "term51: selectionString: reversed selection equals forward selection"
func TestSc4_selectionString_reversed_selection_equals_forward_selection(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 5, 0)
	defer s.Deinit()
	sc4Write(t, s, "abc\ndef\nghi")

	fwd := sc4RSelStr(t, s, 1, 0, 1, 2, false, true)
	expectEqualStrings(t, "bc\ndef\ngh", string(fwd))

	rev := sc4RSelStr(t, s, 1, 2, 1, 0, false, true)
	expectEqualStrings(t, "bc\ndef\ngh", string(rev))
}

// Zig: test "term51: selectionString: empty screen yields empty string"
func TestSc4_selectionString_empty_screen_yields_empty_string(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 5, 0)
	defer s.Deinit()

	text := sc4RSelStr(t, s, 0, 0, 4, 2, false, true)
	expectEqualStrings(t, "", string(text))
	expectEqual(t, uint8(0), sc4Sentinel(t, text))

	text2 := sc4RSelStr(t, s, 0, 0, 4, 2, false, false)
	expectEqualStrings(t, "", string(text2))
}

// Zig: test "term51: selectionString: trim removes trailing blanks only"
func TestSc4_selectionString_trim_removes_trailing_blanks_only(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 6, 5, 0)
	defer s.Deinit()
	sc4Write(t, s, " ab   \n  cd  \n      ")

	{
		text := sc4RSelStr(t, s, 0, 0, 5, 1, false, true)
		expectEqualStrings(t, " ab\n  cd", string(text))
	}
	{
		text := sc4RSelStr(t, s, 0, 0, 5, 1, false, false)
		expectEqualStrings(t, " ab   \n  cd  ", string(text))
	}

	// A selection of only spaces: trimmed it is empty, untrimmed it is kept.
	{
		text := sc4RSelStr(t, s, 0, 2, 5, 2, false, true)
		expectEqualStrings(t, "", string(text))
	}
	{
		text := sc4RSelStr(t, s, 0, 2, 5, 2, false, false)
		expectEqualStrings(t, "      ", string(text))
	}
}

// Zig: test "term51: selectionString: rectangle selection"
func TestSc4_selectionString_rectangle_selection(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 6, 5, 0)
	defer s.Deinit()
	sc4Write(t, s, "abcdef\nghijkl\nmnopqr")

	// Top-left to bottom-right.
	{
		text := sc4RSelStr(t, s, 1, 0, 3, 2, true, true)
		expectEqualStrings(t, "bcd\nhij\nnop", string(text))
	}
	// Bottom-right to top-left.
	{
		text := sc4RSelStr(t, s, 3, 2, 1, 0, true, true)
		expectEqualStrings(t, "bcd\nhij\nnop", string(text))
	}
	// Top-right to bottom-left.
	{
		text := sc4RSelStr(t, s, 3, 0, 1, 2, true, true)
		expectEqualStrings(t, "bcd\nhij\nnop", string(text))
	}
	// Bottom-left to top-right.
	{
		text := sc4RSelStr(t, s, 1, 2, 3, 0, true, true)
		expectEqualStrings(t, "bcd\nhij\nnop", string(text))
	}
	// Single column rectangle.
	{
		text := sc4RSelStr(t, s, 2, 0, 2, 2, true, true)
		expectEqualStrings(t, "c\ni\no", string(text))
	}
	// The same corners as a non-rectangle selection.
	{
		text := sc4RSelStr(t, s, 1, 0, 3, 2, false, true)
		expectEqualStrings(t, "bcdef\nghijkl\nmnop", string(text))
	}
}

// Zig: test "term51: selectionString: soft wrapped rows are unwrapped, hard breaks kept"
func TestSc4_selectionString_soft_wrapped_rows_are_unwrapped_hard_breaks_kept(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 5, 0)
	defer s.Deinit()
	sc4Write(t, s, "1ABCD2EFGH3I\nxy")

	{
		text := sc4RSelStr(t, s, 0, 0, 1, 3, false, true)
		expectEqualStrings(t, "1ABCD2EFGH3I\nxy", string(text))
	}
	// Selection that starts and ends in the middle of the wrapped line.
	{
		text := sc4RSelStr(t, s, 3, 0, 0, 2, false, true)
		expectEqualStrings(t, "CD2EFGH3", string(text))
	}
}

// Zig: test "term51: selectionString: wide characters"
func TestSc4_selectionString_wide_characters(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 6, 5, 0)
	defer s.Deinit()
	sc4Write(t, s, "a世b")

	// Whole row.
	{
		text := sc4RSelStr(t, s, 0, 0, 5, 0, false, true)
		expectEqualStrings(t, "a世b", string(text))
	}
	// Starting on the spacer tail includes the whole wide character.
	{
		text := sc4RSelStr(t, s, 2, 0, 3, 0, false, true)
		expectEqualStrings(t, "世b", string(text))
	}
	// Ending on the wide head or its spacer tail yields the character.
	{
		text := sc4RSelStr(t, s, 0, 0, 1, 0, false, true)
		expectEqualStrings(t, "a世", string(text))
	}
	{
		text := sc4RSelStr(t, s, 0, 0, 2, 0, false, true)
		expectEqualStrings(t, "a世", string(text))
	}
}

// Zig: test "term51: selectionString: wide character wrapped with spacer head"
func TestSc4_selectionString_wide_character_wrapped_with_spacer_head(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 5, 0)
	defer s.Deinit()
	sc4Write(t, s, "abcd世e")

	// Row 0 ends with a spacer head, the wide char moved to row 1.
	expectEqual(t, PageCellWideSpacerHead, sc4RPin(t, s, 4, 0).RowAndCell().cell.wide)
	{
		text := sc4RSelStr(t, s, 0, 0, 2, 1, false, true)
		expectEqualStrings(t, "abcd世e", string(text))
	}
}

// Zig: test "term51: selectionString: grapheme clusters are emitted"
func TestSc4_selectionString_grapheme_clusters_are_emitted(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 6, 5, 0)
	defer s.Deinit()
	sc4Write(t, s, "éx")

	text := sc4RSelStr(t, s, 0, 0, 1, 0, false, true)
	expectEqualStrings(t, "éx", string(text))
}

// Zig: test "term51: selectionString: map has one pin per output byte"
func TestSc4_selectionString_map_has_one_pin_per_output_byte(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 6, 5, 0)
	defer s.Deinit()
	sc4Write(t, s, "ab\ncd")

	var map_ StringMap
	opts := screenSelectionStringDefault(SelectionInit(sc4RPin(t, s, 0, 0), sc4RPin(t, s, 1, 1), false))
	opts.map_ = &map_
	text, err := s.SelectionString(opts)
	zigTry(t, err)
	defer func() { map_.Deinit() }()

	expectEqualStrings(t, "ab\ncd", string(text))
	expectEqualStrings(t, "ab\ncd", string(map_.string))
	expectEqual(t, uint8(0), sc4Sentinel(t, map_.string))
	// The map string is a copy, not the returned slice.
	expect(t, &map_.string[0] != &text[0], "map.string.ptr != text.ptr")
	expectEqual(t, len(text), len(map_.map_))

	expectEqual(t, sc4ScreenPt(0, 0), sc4RPt(t, s, map_.map_[0]))
	expectEqual(t, sc4ScreenPt(1, 0), sc4RPt(t, s, map_.map_[1]))
	expectEqual(t, sc4ScreenPt(0, 1), sc4RPt(t, s, map_.map_[3]))
	expectEqual(t, sc4ScreenPt(1, 1), sc4RPt(t, s, map_.map_[4]))
}

// Zig: test "term51: selectionString: map maps every byte of a multibyte character"
func TestSc4_selectionString_map_maps_every_byte_of_a_multibyte_character(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 6, 5, 0)
	defer s.Deinit()
	sc4Write(t, s, "a世b")

	var map_ StringMap
	opts := screenSelectionStringDefault(SelectionInit(sc4RPin(t, s, 0, 0), sc4RPin(t, s, 5, 0), false))
	opts.map_ = &map_
	text, err := s.SelectionString(opts)
	zigTry(t, err)
	defer func() { map_.Deinit() }()

	expectEqualStrings(t, "a世b", string(text))
	expectEqual(t, 5, len(map_.map_))
	expectEqual(t, sc4ScreenPt(0, 0), sc4RPt(t, s, map_.map_[0]))
	for i := 1; i < 4; i++ {
		expectEqual(t, sc4ScreenPt(1, 0), sc4RPt(t, s, map_.map_[i]))
	}
	expectEqual(t, sc4ScreenPt(3, 0), sc4RPt(t, s, map_.map_[4]))
}

// Zig: test "term51: selectionString: map is untouched when not requested and empty selection map"
func TestSc4_selectionString_map_is_untouched_when_not_requested_and_empty_selection_map(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 6, 5, 0)
	defer s.Deinit()

	var map_ StringMap
	opts := screenSelectionStringDefault(SelectionInit(sc4RPin(t, s, 0, 0), sc4RPin(t, s, 5, 0), false))
	opts.map_ = &map_
	text, err := s.SelectionString(opts)
	zigTry(t, err)
	defer func() { map_.Deinit() }()
	expectEqualStrings(t, "", string(text))
	expectEqualStrings(t, "", string(map_.string))
	expectEqual(t, 0, len(map_.map_))
}

// term51: selectionString: allocation failure while copying the map: not
// translated (tripwire fault injection), see scripts/not-translated-sc4.txt.

// term51: selectionString: allocation failures free everything: not
// translated (FailingAllocator), see scripts/not-translated-sc4.txt.

// Zig: test "term51: selectionString: selection across pages"
func TestSc4_selectionString_selection_across_pages(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 3, 2048)
	defer s.Deinit()

	first_page_size := sc4RFillFirstPage(t, s)
	sc4Write(t, s, "abc\ndef\nghi")

	// The rows really are in two different pages.
	expect(t,
		sc4RPin(t, s, 0, first_page_size-1).node != sc4RPin(t, s, 0, first_page_size).node,
		"term51RPin(&s, 0, first_page_size - 1).node != term51RPin(&s, 0, first_page_size).node",
	)

	text := sc4RSelStr(t, s, 1, first_page_size-1, 1, first_page_size+1, false, true)
	expectEqualStrings(t, "bc\ndef\ngh", string(text))

	// Reversed.
	rev := sc4RSelStr(t, s, 1, first_page_size+1, 1, first_page_size-1, false, true)
	expectEqualStrings(t, "bc\ndef\ngh", string(rev))

	// Map pins refer to the right pages.
	var map_ StringMap
	opts := screenSelectionStringDefault(SelectionInit(
		sc4RPin(t, s, 0, first_page_size-1),
		sc4RPin(t, s, 2, first_page_size+1),
		false,
	))
	opts.map_ = &map_
	t2, err := s.SelectionString(opts)
	zigTry(t, err)
	defer func() { map_.Deinit() }()
	expectEqualStrings(t, "abc\ndef\nghi", string(t2))
	expect(t, map_.map_[0].node != map_.map_[len(map_.map_)-1].node, "map.map[0].node != map.map[map.map.len - 1].node")
}

// Zig: test "term51: selectionString: soft wrap across a page boundary is unwrapped"
func TestSc4_selectionString_soft_wrap_across_a_page_boundary_is_unwrapped(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 3, 2048)
	defer s.Deinit()

	first_page_size := sc4RFillFirstPage(t, s)
	sc4Write(t, s, "aaaaabbbbbccccc")

	text := sc4RSelStr(t, s, 0, first_page_size-1, 4, first_page_size, false, true)
	expectEqualStrings(t, "aaaaabbbbbccccc", string(text))
}

// Zig: test "term51: selectionString: selection through scrollback history"
func TestSc4_selectionString_selection_through_scrollback_history(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 3, 2, 10)
	defer s.Deinit()
	sc4Write(t, s, "a\nb\nc\nd")

	// Screen row 0 is history ('a'), rows 2 and 3 are the active area.
	text := sc4RSelStr(t, s, 0, 0, 0, 3, false, true)
	expectEqualStrings(t, "a\nb\nc\nd", string(text))

	hist := sc4RSelStr(t, s, 0, 1, 2, 2, true, true)
	expectEqualStrings(t, "b\nc", string(hist))
}

// Zig: test "term51: selectLine: trims default whitespace and works from any column"
func TestSc4_selectLine_trims_default_whitespace_and_works_from_any_column(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 5, 0)
	defer s.Deinit()
	sc4Write(t, s, "  ab cd  \n\tef\t")

	// Click on every kind of cell in the first row: leading space, letter,
	// interior space, trailing space and a never-written cell.
	for _, x := range []SizeCellCountInt{0, 1, 2, 4, 6, 7, 8, 9} {
		sel := sc4Line(s, sc4RPin(t, s, x, 0))
		sc4RExpectSel(t, s, sel, 2, 0, 6, 0)
	}

	// Tabs are in the default whitespace set too.
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 2, 1)), 1, 1, 2, 1)
}

// Zig: test "term51: selectLine: custom whitespace sets"
func TestSc4_selectLine_custom_whitespace_sets(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 8, 5, 0)
	defer s.Deinit()
	sc4Write(t, s, "xxabxx\n  ab  \n  ab  ")

	// Only 'x' is whitespace.
	sc4RExpectSel(t, s, sc4LineWS(s, sc4RPin(t, s, 3, 0), Some([]uint32{'x'})), 2, 0, 3, 0)

	// With default whitespace 'x' is content.
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 3, 0)), 0, 0, 5, 0)

	// Spaces are content when only 'x' is whitespace.
	sc4RExpectSel(t, s, sc4LineWS(s, sc4RPin(t, s, 3, 1), Some([]uint32{'x'})), 0, 1, 5, 1)

	// An empty set trims nothing written, but cells that were never
	// written (no text) are still skipped at the ends.
	sc4RExpectSel(t, s, sc4LineWS(s, sc4RPin(t, s, 3, 2), Some([]uint32{})), 0, 2, 5, 2)

	// A row of only whitespace has nothing to select.
	expect(t, !sc4LineWS(s, sc4RPin(t, s, 3, 1), Some([]uint32{'a', 'b', ' '})).Valid,
		"s.selectLine(.{ .pin = term51RPin(&s, 3, 1), .whitespace = &.{ 'a', 'b', ' ' } }) == null")
}

// Zig: test "term51: selectLine: null whitespace selects the full row"
func TestSc4_selectLine_null_whitespace_selects_the_full_row(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 6, 4, 0)
	defer s.Deinit()
	sc4Write(t, s, "ab\n  \n")

	sc4RExpectSel(t, s, sc4LineWS(s, sc4RPin(t, s, 1, 0), None[[]uint32]()), 0, 0, 5, 0)

	// Row of written spaces.
	sc4RExpectSel(t, s, sc4LineWS(s, sc4RPin(t, s, 3, 1), None[[]uint32]()), 0, 1, 5, 1)

	// Never written row.
	sc4RExpectSel(t, s, sc4LineWS(s, sc4RPin(t, s, 0, 2), None[[]uint32]()), 0, 2, 5, 2)
}

// Zig: test "term51: selectLine: blank rows return null with default whitespace"
func TestSc4_selectLine_blank_rows_return_null_with_default_whitespace(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 6, 4, 0)
	defer s.Deinit()
	sc4Write(t, s, "ab\n   \n")

	// Row of spaces.
	expect(t, !sc4Line(s, sc4RPin(t, s, 1, 1)).Valid, "s.selectLine(.{ .pin = term51RPin(&s, 1, 1) }) == null")
	// Never written row.
	expect(t, !sc4Line(s, sc4RPin(t, s, 0, 3)).Valid, "s.selectLine(.{ .pin = term51RPin(&s, 0, 3) }) == null")
}

// Zig: test "term51: selectLine: single cell line and last column"
func TestSc4_selectLine_single_cell_line_and_last_column(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 4, 0)
	defer s.Deinit()
	sc4Write(t, s, "a\n    b\nabcde\nxy")

	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 0, 0)), 0, 0, 0, 0)
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 4, 1)), 4, 1, 4, 1)

	// A full row that is not soft wrapped selects only that row, even when
	// clicking the last column.
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 4, 2)), 0, 2, 4, 2)
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 0, 3)), 0, 3, 1, 3)
}

// Zig: test "term51: selectLine: soft wrapped line over three rows"
func TestSc4_selectLine_soft_wrapped_line_over_three_rows(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 6, 0)
	defer s.Deinit()
	sc4Write(t, s, "  abcdefg  \nxyz")

	// Whitespace is trimmed from the start of the first row and the end of
	// the last row; every click in the wrapped line gives the same result.
	for _, c := range [][2]SizeCellCountInt{
		{0, 0}, {3, 0}, {0, 1}, {2, 1}, {4, 1}, {0, 2},
	} {
		sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, c[0], uint32(c[1]))), 2, 0, 3, 1)
	}

	// The next hard line is separate.
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 1, 3)), 0, 3, 2, 3)
}

// Zig: test "term51: selectLine: soft wrapped line with null whitespace spans full rows"
func TestSc4_selectLine_soft_wrapped_line_with_null_whitespace_spans_full_rows(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 6, 0)
	defer s.Deinit()
	sc4Write(t, s, "abcdefg\nxyz")

	sc4RExpectSel(t, s, sc4LineWS(s, sc4RPin(t, s, 3, 1), None[[]uint32]()), 0, 0, 4, 1)
}

// Zig: test "term51: selectLine: wrap flag on the last row yields null"
func TestSc4_selectLine_wrap_flag_on_the_last_row_yields_null(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 3, 0)
	defer s.Deinit()
	sc4Write(t, s, "abc\ndef\nghi")

	// Normal case first.
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 1, 2)), 0, 2, 2, 2)

	// A soft wrap on the final row of the screen has no row to continue
	// into: the end of the line can't be found.
	sc4RPin(t, s, 0, 2).RowAndCell().row.wrap = true
	expect(t, !sc4Line(s, sc4RPin(t, s, 1, 2)).Valid, "s.selectLine(.{ .pin = term51RPin(&s, 1, 2) }) == null")
	expect(t, !sc4LineWS(s, sc4RPin(t, s, 1, 2), None[[]uint32]()).Valid, "s.selectLine(.{ .pin = term51RPin(&s, 1, 2), .whitespace = null }) == null")

	// A wrap on a middle row joins it with the next one.
	sc4RPin(t, s, 0, 2).RowAndCell().row.wrap = false
	sc4RPin(t, s, 0, 1).RowAndCell().row.wrap = true
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 1, 1)), 0, 1, 2, 2)
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 1, 2)), 0, 1, 2, 2)

	// A wrap flag on the first row of the screen: the start is simply
	// column zero of that row.
	sc4RPin(t, s, 0, 1).RowAndCell().row.wrap = false
	sc4RPin(t, s, 0, 0).RowAndCell().row.wrap = true
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 1, 0)), 0, 0, 2, 1)
}

// Zig: test "term51: selectLine: wide characters"
func TestSc4_selectLine_wide_characters(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 6, 4, 0)
	defer s.Deinit()
	sc4Write(t, s, "a世b\nab世")

	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 1, 0)), 0, 0, 3, 0)
	// Clicking on the spacer tail selects the line as well.
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 2, 0)), 0, 0, 3, 0)
	// A wide character at the end: the end is the wide head (the spacer
	// tail has no text).
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 0, 1)), 0, 1, 2, 1)
}

// Zig: test "term51: selectLine: wrapped line across a page boundary"
func TestSc4_selectLine_wrapped_line_across_a_page_boundary(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 3, 2048)
	defer s.Deinit()

	r := sc4RFillFirstPage(t, s)
	sc4Write(t, s, "aaaaabbbbbccccc\nzz")
	expect(t, sc4RPin(t, s, 0, r-1).node != sc4RPin(t, s, 0, r).node, "term51RPin(&s, 0, r - 1).node != term51RPin(&s, 0, r).node")

	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 3, r)), 0, r-1, 4, r)
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 3, r-1)), 0, r-1, 4, r)
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 1, r+1)), 0, r+1, 1, r+1)
}

// Zig: test "term51: selectLine: line in scrollback history"
func TestSc4_selectLine_line_in_scrollback_history(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 4, 2, 10)
	defer s.Deinit()
	sc4Write(t, s, " aa \n bb \n cc \n dd")

	// Screen rows 0 and 1 are history.
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 0, 0)), 1, 0, 2, 0)
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 3, 1)), 1, 1, 2, 1)
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 3, 3)), 1, 3, 2, 3)
}

// Zig: test "term51: selectLine: semantic boundary splits a single row into three"
func TestSc4_selectLine_semantic_boundary_splits_a_single_row_into_three(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 6, 3, 0)
	defer s.Deinit()
	s.CursorSetSemanticContent(sc4Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc4Write(t, s, "PP")
	s.CursorSetSemanticContent(sc4Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc4Write(t, s, "II")
	s.CursorSetSemanticContent(sc4Output())
	sc4Write(t, s, "OO")

	// Prompt cells.
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 0, 0)), 0, 0, 1, 0)
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 1, 0)), 0, 0, 1, 0)
	// Input cells.
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 2, 0)), 2, 0, 3, 0)
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 3, 0)), 2, 0, 3, 0)
	// Output cells.
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 4, 0)), 4, 0, 5, 0)
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 5, 0)), 4, 0, 5, 0)

	// With the boundary disabled the whole row is one line.
	for x := 0; x < 6; x++ {
		opts := screenSelectLineDefault(sc4RPin(t, s, SizeCellCountInt(x), 0))
		opts.semantic_prompt_boundary = false
		sc4RExpectSel(t, s, s.SelectLine(opts), 0, 0, 5, 0)
	}
}

// Zig: test "term51: selectLine: semantic boundary across soft wrapped rows"
func TestSc4_selectLine_semantic_boundary_across_soft_wrapped_rows(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 4, 4, 0)
	defer s.Deinit()
	s.CursorSetSemanticContent(sc4Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc4Write(t, s, "PP")
	s.CursorSetSemanticContent(sc4Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc4Write(t, s, "IIII")
	s.CursorSetSemanticContent(sc4Output())
	sc4Write(t, s, "OO")

	// Rows: "PPII" / "IIOO", wrapped.
	expect(t, sc4RPin(t, s, 0, 0).RowAndCell().row.wrap, "term51RPin(&s, 0, 0).rowAndCell().row.wrap")
	expect(t, sc4RPin(t, s, 0, 1).RowAndCell().row.wrap_continuation, "term51RPin(&s, 0, 1).rowAndCell().row.wrap_continuation")

	// Prompt: stops where the input starts.
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 0, 0)), 0, 0, 1, 0)
	// Input spans the wrap.
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 3, 0)), 2, 0, 1, 1)
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 0, 1)), 2, 0, 1, 1)
	// Output is only the end of the second row.
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 3, 1)), 2, 1, 3, 1)
}

// Zig: test "term51: selectLine: semantic boundary exactly at a row break"
func TestSc4_selectLine_semantic_boundary_exactly_at_a_row_break(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 4, 4, 0)
	defer s.Deinit()
	s.CursorSetSemanticContent(sc4Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc4Write(t, s, "PPPP")
	s.CursorSetSemanticContent(sc4Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc4Write(t, s, "IIII")

	// Prompt row ends at the row end even though the line continues
	// (first cell of next row differs).
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 1, 0)), 0, 0, 3, 0)
	// Input starts at the start of the second row.
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 2, 1)), 0, 1, 3, 1)
}

// Zig: test "term51: selectLine: semantic state equal over soft wrapped rows"
func TestSc4_selectLine_semantic_state_equal_over_soft_wrapped_rows(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 4, 4, 0)
	defer s.Deinit()
	s.CursorSetSemanticContent(sc4Output())
	sc4Write(t, s, "OOOOOOOO")

	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 1, 1)), 0, 0, 3, 1)
	sc4RExpectSel(t, s, sc4Line(s, sc4RPin(t, s, 2, 0)), 0, 0, 3, 1)
}

// Zig: test "term51: selectAll: empty and blank screens return null"
func TestSc4_selectAll_empty_and_blank_screens_return_null(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 6, 4, 0)
	defer s.Deinit()

	expect(t, !s.SelectAll().Valid, "s.selectAll() == null")

	// Only whitespace (space and tab) is also nothing.
	sc4Write(t, s, "   \n\t \t\n  ")
	expect(t, !s.SelectAll().Valid, "s.selectAll() == null")
}

// Zig: test "term51: selectAll: single cell"
func TestSc4_selectAll_single_cell(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 6, 4, 0)
	defer s.Deinit()
	sc4Write(t, s, "x")

	sc4RExpectSel(t, s, s.SelectAll(), 0, 0, 0, 0)

	// A lone character somewhere else on the screen.
	s2 := sc4Init(t, 6, 4, 0)
	defer s2.Deinit()
	s2.CursorAbsolute(4, 2)
	sc4Write(t, s2, "y")
	sc4RExpectSel(t, s2, s2.SelectAll(), 4, 2, 4, 2)
}

// Zig: test "term51: selectAll: skips surrounding blank rows and whitespace"
func TestSc4_selectAll_skips_surrounding_blank_rows_and_whitespace(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 6, 0)
	defer s.Deinit()
	sc4Write(t, s, "\n  \n \tab  c \n\n d \n")

	// First non-whitespace is 'a' (x=2,y=2), last is 'd' (x=1,y=4).
	sc4RExpectSel(t, s, s.SelectAll(), 2, 2, 1, 4)
}

// Zig: test "term51: selectAll: includes scrollback history"
func TestSc4_selectAll_includes_scrollback_history(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 4, 2, 10)
	defer s.Deinit()
	sc4Write(t, s, "  a\nb\nc\n d \ne")

	sc4RExpectSel(t, s, s.SelectAll(), 2, 0, 0, 4)

	text, err := s.SelectionString(screenSelectionStringDefault(zigUnwrap(t, s.SelectAll())))
	zigTry(t, err)
	expectEqualStrings(t, "a\nb\nc\n d\ne", string(text))
}

// Zig: test "term51: selectAll: spans multiple pages"
func TestSc4_selectAll_spans_multiple_pages(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 3, 2048)
	defer s.Deinit()

	s.CursorAbsolute(3, 0)
	sc4Write(t, s, "first")
	s.CursorAbsolute(0, 0)
	r := sc4RFillFirstPage(t, s)
	sc4Write(t, s, "\nlast\n\n")

	sel := zigUnwrap(t, s.SelectAll())
	expect(t, sel.Start().node != sel.End().node, "sel.start().node != sel.end().node")
	sc4RExpectSel(t, s, Some(sel), 3, 0, 3, r)
}

// Zig: test "term51: selectAll: wide character at the end"
func TestSc4_selectAll_wide_character_at_the_end(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 6, 3, 0)
	defer s.Deinit()
	sc4Write(t, s, "ab世")

	// The spacer tail has no text so the end is the wide head.
	sc4RExpectSel(t, s, s.SelectAll(), 0, 0, 2, 0)
}

// Zig: test "term51: selectAll: non-breaking space is content"
func TestSc4_selectAll_non_breaking_space_is_content(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 6, 3, 0)
	defer s.Deinit()
	// U+00A0 is not in the whitespace set, so it counts as content.
	sc4Write(t, s, "    x ")
	sc4RExpectSel(t, s, s.SelectAll(), 1, 0, 5, 0)
}

// Zig: test "term51: selectWordBetween: forward range finds the first word"
func TestSc4_selectWordBetween_forward_range_finds_the_first_word(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 20, 4, 0)
	defer s.Deinit()
	sc4Write(t, s, "foo bar baz")
	b := sc4R_boundary

	// Start on a word.
	sc4RExpectSel(t, s, s.SelectWordBetween(sc4RPin(t, s, 5, 0), sc4RPin(t, s, 10, 0), b), 4, 0, 6, 0)
	// The word is not clipped to the range.
	sc4RExpectSel(t, s, s.SelectWordBetween(sc4RPin(t, s, 0, 0), sc4RPin(t, s, 1, 0), b), 0, 0, 2, 0)
	// Start on whitespace (which is text): the whitespace run is selected.
	sc4RExpectSel(t, s, s.SelectWordBetween(sc4RPin(t, s, 3, 0), sc4RPin(t, s, 10, 0), b), 3, 0, 3, 0)
}

// Zig: test "term51: selectWordBetween: backward range finds the first word"
func TestSc4_selectWordBetween_backward_range_finds_the_first_word(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 20, 4, 0)
	defer s.Deinit()
	sc4Write(t, s, "foo bar baz")
	b := sc4R_boundary

	sc4RExpectSel(t, s, s.SelectWordBetween(sc4RPin(t, s, 10, 0), sc4RPin(t, s, 0, 0), b), 8, 0, 10, 0)
	sc4RExpectSel(t, s, s.SelectWordBetween(sc4RPin(t, s, 5, 0), sc4RPin(t, s, 0, 0), b), 4, 0, 6, 0)
	sc4RExpectSel(t, s, s.SelectWordBetween(sc4RPin(t, s, 3, 0), sc4RPin(t, s, 0, 0), b), 3, 0, 3, 0)
}

// Zig: test "term51: selectWordBetween: start equals end"
func TestSc4_selectWordBetween_start_equals_end(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 20, 4, 0)
	defer s.Deinit()
	sc4Write(t, s, "foo bar")
	b := sc4R_boundary

	sc4RExpectSel(t, s, s.SelectWordBetween(sc4RPin(t, s, 5, 0), sc4RPin(t, s, 5, 0), b), 4, 0, 6, 0)
	// Single empty cell: nothing to select.
	expect(t, !s.SelectWordBetween(sc4RPin(t, s, 15, 0), sc4RPin(t, s, 15, 0), b).Valid,
		"s.selectWordBetween(term51RPin(&s, 15, 0), term51RPin(&s, 15, 0), b) == null")
}

// Zig: test "term51: selectWordBetween: skips empty cells until text"
func TestSc4_selectWordBetween_skips_empty_cells_until_text(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 20, 4, 0)
	defer s.Deinit()
	s.CursorAbsolute(5, 1)
	sc4Write(t, s, "xy")
	b := sc4R_boundary

	// Forward from the top-left across an empty row into the text.
	sc4RExpectSel(t, s, s.SelectWordBetween(sc4RPin(t, s, 0, 0), sc4RPin(t, s, 19, 3), b), 5, 1, 6, 1)
	// Backward from the bottom-right to the text.
	sc4RExpectSel(t, s, s.SelectWordBetween(sc4RPin(t, s, 19, 3), sc4RPin(t, s, 0, 0), b), 5, 1, 6, 1)
	// Range ends before the text begins.
	expect(t, !s.SelectWordBetween(sc4RPin(t, s, 0, 0), sc4RPin(t, s, 4, 1), b).Valid,
		"s.selectWordBetween(term51RPin(&s, 0, 0), term51RPin(&s, 4, 1), b) == null")
	// Range starts after the text, going backward ends before it begins.
	expect(t, !s.SelectWordBetween(sc4RPin(t, s, 19, 1), sc4RPin(t, s, 7, 1), b).Valid,
		"s.selectWordBetween(term51RPin(&s, 19, 1), term51RPin(&s, 7, 1), b) == null")
	// Entirely empty ranges.
	expect(t, !s.SelectWordBetween(sc4RPin(t, s, 0, 0), sc4RPin(t, s, 19, 0), b).Valid,
		"s.selectWordBetween(term51RPin(&s, 0, 0), term51RPin(&s, 19, 0), b) == null")
	expect(t, !s.SelectWordBetween(sc4RPin(t, s, 19, 3), sc4RPin(t, s, 0, 2), b).Valid,
		"s.selectWordBetween(term51RPin(&s, 19, 3), term51RPin(&s, 0, 2), b) == null")
	// Range stops right at the first cell of the text.
	sc4RExpectSel(t, s, s.SelectWordBetween(sc4RPin(t, s, 0, 0), sc4RPin(t, s, 5, 1), b), 5, 1, 6, 1)
}

// Zig: test "term51: selectWordBetween: boundary codepoints are passed through"
func TestSc4_selectWordBetween_boundary_codepoints_are_passed_through(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 20, 4, 0)
	defer s.Deinit()
	sc4Write(t, s, "foo bar")

	// 'o' is the only boundary: from 'f' the word is just "f".
	only_o := []uint32{'o'}
	sc4RExpectSel(t, s, s.SelectWordBetween(sc4RPin(t, s, 0, 0), sc4RPin(t, s, 6, 0), only_o), 0, 0, 0, 0)
	// The first cell that has a selectable word is used: here every text
	// cell is selectable, so the first cell is chosen.
	none := []uint32{}
	sc4RExpectSel(t, s, s.SelectWordBetween(sc4RPin(t, s, 2, 0), sc4RPin(t, s, 6, 0), none), 0, 0, 6, 0)
}

// Zig: test "term51: selectWordBetween: wide character and spacer tail"
func TestSc4_selectWordBetween_wide_character_and_spacer_tail(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 4, 0)
	defer s.Deinit()
	sc4Write(t, s, "世 b")
	b := sc4R_boundary

	// Starting on the spacer tail (no text) moves on to the space.
	sc4RExpectSel(t, s, s.SelectWordBetween(sc4RPin(t, s, 1, 0), sc4RPin(t, s, 3, 0), b), 2, 0, 2, 0)
	// Starting on the wide head selects it.
	sc4RExpectSel(t, s, s.SelectWordBetween(sc4RPin(t, s, 0, 0), sc4RPin(t, s, 3, 0), b), 0, 0, 0, 0)
}

// Zig: test "term51: selectWord: every default boundary codepoint separates words"
func TestSc4_selectWord_every_default_boundary_codepoint_separates_words(t *testing.T) {
	testStdCapacity(t)

	for _, cp := range sc4R_boundary {
		// The empty cell (codepoint 0) is covered separately.
		if cp == 0 {
			continue
		}
		func() {
			s := sc4Init(t, 10, 3, 0)
			defer s.Deinit()
			line := "ab" + string(rune(cp)) + "cd"
			sc4Write(t, s, line)

			b := sc4R_boundary
			// Word before the boundary.
			sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 0, 0), b), 0, 0, 1, 0)
			sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 1, 0), b), 0, 0, 1, 0)
			// The boundary character alone.
			sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 2, 0), b), 2, 0, 2, 0)
			// Word after it.
			sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 3, 0), b), 3, 0, 4, 0)
			sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 4, 0), b), 3, 0, 4, 0)
		}()
	}
}

// Zig: test "term51: selectWord: adjacent boundary codepoints form one run"
func TestSc4_selectWord_adjacent_boundary_codepoints_form_one_run(t *testing.T) {
	testStdCapacity(t)

	// Every pair of consecutive non-NUL boundary codepoints.
	for i := 1; i+1 < len(sc4R_boundary); i++ {
		func() {
			s := sc4Init(t, 10, 3, 0)
			defer s.Deinit()
			text := "a" + string(rune(sc4R_boundary[i])) + string(rune(sc4R_boundary[i+1])) + "b"
			sc4Write(t, s, text)

			b := sc4R_boundary
			sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 1, 0), b), 1, 0, 2, 0)
			sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 2, 0), b), 1, 0, 2, 0)
			sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 0, 0), b), 0, 0, 0, 0)
			sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 3, 0), b), 3, 0, 3, 0)
		}()
	}
}

// Zig: test "term51: selectWord: non-boundary punctuation and symbols stay inside a word"
func TestSc4_selectWord_non_boundary_punctuation_and_symbols_stay_inside_a_word(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 40, 3, 0)
	defer s.Deinit()
	text := "foo-bar.baz/qux_1=2+3@4#5%6&7*8!9?~^\\xé"
	sc4Write(t, s, text)
	b := sc4R_boundary

	last := SizeCellCountInt(utf8.RuneCountInString(text))
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 0, 0), b), 0, 0, last-1, 0)
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 3, 0), b), 0, 0, last-1, 0)
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, last-1, 0), b), 0, 0, last-1, 0)
}

// Zig: test "term51: selectWord: whitespace runs and tabs"
func TestSc4_selectWord_whitespace_runs_and_tabs(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 20, 3, 0)
	defer s.Deinit()
	sc4Write(t, s, "ab   \t cd")
	b := sc4R_boundary

	// Spaces and a tab form one boundary run.
	for _, x := range []SizeCellCountInt{2, 3, 4, 5, 6} {
		sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, x, 0), b), 2, 0, 6, 0)
	}
}

// Zig: test "term51: selectWord: empty cells return null"
func TestSc4_selectWord_empty_cells_return_null(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 3, 0)
	defer s.Deinit()
	sc4Write(t, s, "ab")
	b := sc4R_boundary

	// Right of the text on the same row and a never-written row.
	expect(t, !s.SelectWord(sc4RPin(t, s, 2, 0), b).Valid, "s.selectWord(term51RPin(&s, 2, 0), b) == null")
	expect(t, !s.SelectWord(sc4RPin(t, s, 9, 0), b).Valid, "s.selectWord(term51RPin(&s, 9, 0), b) == null")
	expect(t, !s.SelectWord(sc4RPin(t, s, 0, 2), b).Valid, "s.selectWord(term51RPin(&s, 0, 2), b) == null")
	// Empty screen: nothing anywhere.
	s2 := sc4Init(t, 3, 1, 0)
	defer s2.Deinit()
	expect(t, !s2.SelectWord(sc4RPin(t, s2, 0, 0), b).Valid, "s2.selectWord(term51RPin(&s2, 0, 0), b) == null")
}

// Zig: test "term51: selectWord: an empty cell ends a word even if NUL is not a boundary"
func TestSc4_selectWord_an_empty_cell_ends_a_word_even_if_NUL_is_not_a_boundary(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 3, 0)
	defer s.Deinit()
	s.CursorAbsolute(2, 0)
	sc4Write(t, s, "abc")
	s.CursorAbsolute(7, 0)
	sc4Write(t, s, "de")

	only_colon := []uint32{':'}
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 3, 0), only_colon), 2, 0, 4, 0)
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 8, 0), only_colon), 7, 0, 8, 0)
}

// Zig: test "term51: selectWord: empty boundary set makes every text cell part of one word"
func TestSc4_selectWord_empty_boundary_set_makes_every_text_cell_part_of_one_word(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 20, 3, 0)
	defer s.Deinit()
	sc4Write(t, s, "ab cd:ef")

	none := []uint32{}
	for _, x := range []SizeCellCountInt{0, 2, 5, 7} {
		sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, x, 0), none), 0, 0, 7, 0)
	}
}

// Zig: test "term51: selectWord: custom boundary set and boundary-only text"
func TestSc4_selectWord_custom_boundary_set_and_boundary_only_text(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 20, 3, 0)
	defer s.Deinit()
	sc4Write(t, s, "a-b-c  d\n::::")

	dash := []uint32{'-'}
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 0, 0), dash), 0, 0, 0, 0)
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 1, 0), dash), 1, 0, 1, 0)
	// With only '-' as boundary, the spaces are word characters.
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 4, 0), dash), 4, 0, 7, 0)
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 6, 0), dash), 4, 0, 7, 0)

	// A row made only of boundary characters.
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 2, 1), sc4R_boundary), 0, 1, 3, 1)
}

// Zig: test "term51: selectWord: single cell word and word at the edges"
func TestSc4_selectWord_single_cell_word_and_word_at_the_edges(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 3, 2, 0)
	defer s.Deinit()
	sc4Write(t, s, "a\ncde")
	b := sc4R_boundary

	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 0, 0), b), 0, 0, 0, 0)
	// First cell of the screen and the last cell of the last row.
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 0, 1), b), 0, 1, 2, 1)
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 2, 1), b), 0, 1, 2, 1)
}

// Zig: test "term51: selectWord: word ends at a hard row break"
func TestSc4_selectWord_word_ends_at_a_hard_row_break(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 4, 0)
	defer s.Deinit()
	sc4Write(t, s, "abcde\nfgh")
	b := sc4R_boundary

	// Clicks that are not on the last column of the unwrapped row stop at it.
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 3, 0), b), 0, 0, 4, 0)
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 0, 0), b), 0, 0, 4, 0)
	// And the row below starts a new word.
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 0, 1), b), 0, 1, 2, 1)
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 2, 1), b), 0, 1, 2, 1)
}

// Zig: test "term51: selectWord: click on last column of an unwrapped row leaks into the next row"
func TestSc4_selectWord_click_on_last_column_of_an_unwrapped_row_leaks_into_the_next_row(t *testing.T) {
	// Upstream quirk, pinned: the pin cell itself is consumed before the
	// "end of unwrapped row" check runs, so a click on the last column of
	// a hard-broken row continues into the next row when that row starts
	// with the same class of character.
	testStdCapacity(t)
	s := sc4Init(t, 5, 4, 0)
	defer s.Deinit()
	sc4Write(t, s, "abcde\nfgh")
	b := sc4R_boundary

	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 4, 0), b), 0, 0, 2, 1)

	// If the next row starts with a different class, it does not leak.
	s2 := sc4Init(t, 5, 4, 0)
	defer s2.Deinit()
	sc4Write(t, s2, "abcde\n fgh")
	sc4RExpectSel(t, s2, s2.SelectWord(sc4RPin(t, s2, 4, 0), b), 0, 0, 4, 0)
}

// Zig: test "term51: selectWord: word across soft wrapped rows"
func TestSc4_selectWord_word_across_soft_wrapped_rows(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 5, 0)
	defer s.Deinit()
	sc4Write(t, s, "abcdefghijkl mn")
	b := sc4R_boundary

	for _, c := range [][2]SizeCellCountInt{{0, 0}, {4, 0}, {0, 1}, {4, 1}, {0, 2}, {1, 2}} {
		sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, c[0], uint32(c[1])), b), 0, 0, 1, 2)
	}
	// The space and the next word.
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 2, 2), b), 2, 2, 2, 2)
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 4, 2), b), 3, 2, 4, 2)
}

// Zig: test "term51: selectWord: boundary right at the wrapped row break"
func TestSc4_selectWord_boundary_right_at_the_wrapped_row_break(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 5, 0)
	defer s.Deinit()
	sc4Write(t, s, "abcde fgh")
	b := sc4R_boundary

	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 2, 0), b), 0, 0, 4, 0)
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 4, 0), b), 0, 0, 4, 0)
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 0, 1), b), 0, 1, 0, 1)
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 2, 1), b), 1, 1, 3, 1)
}

// Zig: test "term51: selectWord: wide characters"
func TestSc4_selectWord_wide_characters(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 3, 0)
	defer s.Deinit()
	sc4Write(t, s, "a世b")
	b := sc4R_boundary

	// The spacer tail has no text, so it acts like an empty cell and ends
	// the word: "b" is not part of the word of "a世".
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 0, 0), b), 0, 0, 1, 0)
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 1, 0), b), 0, 0, 1, 0)
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 3, 0), b), 3, 0, 3, 0)
	// Clicking the spacer tail itself selects nothing.
	expect(t, !s.SelectWord(sc4RPin(t, s, 2, 0), b).Valid, "s.selectWord(term51RPin(&s, 2, 0), b) == null")
}

// Zig: test "term51: selectWord: wide character wrapped with spacer head"
func TestSc4_selectWord_wide_character_wrapped_with_spacer_head(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 4, 0)
	defer s.Deinit()
	sc4Write(t, s, "abcd世e")
	b := sc4R_boundary

	expectEqual(t, PageCellWideSpacerHead, sc4RPin(t, s, 4, 0).RowAndCell().cell.wide)
	// The spacer head (no text) splits the word on the first row.
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 0, 0), b), 0, 0, 3, 0)
	expect(t, !s.SelectWord(sc4RPin(t, s, 4, 0), b).Valid, "s.selectWord(term51RPin(&s, 4, 0), b) == null")
	// And the wide character on the second row is its own word.
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 0, 1), b), 0, 1, 0, 1)
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 2, 1), b), 2, 1, 2, 1)
}

// Zig: test "term51: selectWord: grapheme cell counts as text"
func TestSc4_selectWord_grapheme_cell_counts_as_text(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 3, 0)
	defer s.Deinit()
	sc4Write(t, s, "éx y")
	b := sc4R_boundary

	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 0, 0), b), 0, 0, 1, 0)
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 1, 0), b), 0, 0, 1, 0)
}

// Zig: test "term51: selectWord: word wrapped across a page boundary"
func TestSc4_selectWord_word_wrapped_across_a_page_boundary(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 3, 2048)
	defer s.Deinit()

	r := sc4RFillFirstPage(t, s)
	sc4Write(t, s, "aaaaabbbbbccccc ddd")
	expect(t, sc4RPin(t, s, 0, r-1).node != sc4RPin(t, s, 0, r).node, "term51RPin(&s, 0, r - 1).node != term51RPin(&s, 0, r).node")
	b := sc4R_boundary

	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 3, r-1), b), 0, r-1, 4, r)
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 9, r-1), b), 0, r-1, 4, r)
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 0, r), b), 0, r-1, 4, r)
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 5, r), b), 5, r, 5, r)
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 7, r), b), 6, r, 8, r)
}

// Zig: test "term51: selectWord: word in scrollback history"
func TestSc4_selectWord_word_in_scrollback_history(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 8, 2, 10)
	defer s.Deinit()
	sc4Write(t, s, "foo bar\nbaz qux\nlast one\nend")

	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 5, 0), sc4R_boundary), 4, 0, 6, 0)
	sc4RExpectSel(t, s, s.SelectWord(sc4RPin(t, s, 1, 1), sc4R_boundary), 0, 1, 2, 1)
}

// Zig: test "term51: selectOutput: prompt and input cells return null"
func TestSc4_selectOutput_prompt_and_input_cells_return_null(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 6, 0)
	defer s.Deinit()
	s.CursorSetSemanticContent(sc4Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc4Write(t, s, "$ ")
	s.CursorSetSemanticContent(sc4Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc4Write(t, s, "ls")
	s.CursorSetSemanticContent(sc4Output())
	sc4Write(t, s, "\nout")

	expect(t, !s.SelectOutput(sc4RPin(t, s, 0, 0)).Valid, "s.selectOutput(term51RPin(&s, 0, 0)) == null")
	expect(t, !s.SelectOutput(sc4RPin(t, s, 1, 0)).Valid, "s.selectOutput(term51RPin(&s, 1, 0)) == null")
	expect(t, !s.SelectOutput(sc4RPin(t, s, 2, 0)).Valid, "s.selectOutput(term51RPin(&s, 2, 0)) == null")
	expect(t, !s.SelectOutput(sc4RPin(t, s, 3, 0)).Valid, "s.selectOutput(term51RPin(&s, 3, 0)) == null")
	// Output cell of the prompt's own output.
	sc4RExpectSel(t, s, s.SelectOutput(sc4RPin(t, s, 1, 1)), 0, 1, 2, 1)
}

// Zig: test "term51: selectOutput: no prompts anywhere returns null"
func TestSc4_selectOutput_no_prompts_anywhere_returns_null(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 4, 0)
	defer s.Deinit()
	sc4Write(t, s, "abc\ndef")

	expect(t, !s.SelectOutput(sc4RPin(t, s, 1, 0)).Valid, "s.selectOutput(term51RPin(&s, 1, 0)) == null")
	// Never written cell on an empty screen.
	s2 := sc4Init(t, 10, 4, 0)
	defer s2.Deinit()
	expect(t, !s2.SelectOutput(sc4RPin(t, s2, 0, 0)).Valid, "s2.selectOutput(term51RPin(&s2, 0, 0)) == null")
}

// Zig: test "term51: selectOutput: output before the first prompt"
func TestSc4_selectOutput_output_before_the_first_prompt(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 8, 0)
	defer s.Deinit()
	s.CursorSetSemanticContent(sc4Output())
	sc4Write(t, s, "out1\nout2\n\n\n")
	s.CursorSetSemanticContent(sc4Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc4Write(t, s, "$ ")
	s.CursorSetSemanticContent(sc4Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc4Write(t, s, "ls")

	// From the start of the screen to the last text cell before the prompt.
	sc4RExpectSel(t, s, s.SelectOutput(sc4RPin(t, s, 0, 0)), 0, 0, 3, 1)
	sc4RExpectSel(t, s, s.SelectOutput(sc4RPin(t, s, 9, 3)), 0, 0, 3, 1)
}

// Zig: test "term51: selectOutput: blank output before the first prompt"
func TestSc4_selectOutput_blank_output_before_the_first_prompt(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 8, 0)
	defer s.Deinit()
	s.CursorAbsolute(0, 2)
	s.CursorSetSemanticContent(sc4Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc4Write(t, s, "$ ")

	// Nothing to trim to: the selection collapses to the screen origin.
	sc4RExpectSel(t, s, s.SelectOutput(sc4RPin(t, s, 5, 0)), 0, 0, 0, 0)
}

// Zig: test "term51: selectOutput: output after a prompt trims trailing blank rows"
func TestSc4_selectOutput_output_after_a_prompt_trims_trailing_blank_rows(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 12, 0)
	defer s.Deinit()
	s.CursorSetSemanticContent(sc4Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc4Write(t, s, "$ ")
	s.CursorSetSemanticContent(sc4Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc4Write(t, s, "ls")
	s.CursorSetSemanticContent(sc4Output())
	sc4Write(t, s, "\no1\no2xx\n\n\n")
	s.CursorSetSemanticContent(sc4Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc4Write(t, s, "$ ")
	s.CursorSetSemanticContent(sc4Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc4Write(t, s, "x")

	// Output starts at the first output cell with text and ends at the last
	// output cell with text.
	for _, c := range [][2]SizeCellCountInt{{0, 1}, {5, 2}, {9, 4}, {0, 3}} {
		sc4RExpectSel(t, s, s.SelectOutput(sc4RPin(t, s, c[0], uint32(c[1]))), 0, 1, 3, 2)
	}
	// The blank (output-class) cell on the prompt row selects the same block.
	sc4RExpectSel(t, s, s.SelectOutput(sc4RPin(t, s, 6, 0)), 0, 1, 3, 2)
}

// Zig: test "term51: selectOutput: output of the last prompt reaches the end of the screen"
func TestSc4_selectOutput_output_of_the_last_prompt_reaches_the_end_of_the_screen(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 6, 0)
	defer s.Deinit()
	s.CursorSetSemanticContent(sc4Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc4Write(t, s, "$ ")
	s.CursorSetSemanticContent(sc4Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc4Write(t, s, "ls")
	s.CursorSetSemanticContent(sc4Output())
	sc4Write(t, s, "\none\ntwo\n\n")

	sc4RExpectSel(t, s, s.SelectOutput(sc4RPin(t, s, 1, 1)), 0, 1, 2, 2)
}

// Zig: test "term51: selectOutput: prompt without any output text returns null"
func TestSc4_selectOutput_prompt_without_any_output_text_returns_null(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 4, 0)
	defer s.Deinit()
	s.CursorSetSemanticContent(sc4Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc4Write(t, s, "$ ")
	s.CursorSetSemanticContent(sc4Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc4Write(t, s, "ls")

	// Blank (output-class) cell after the input of the only prompt.
	expect(t, !s.SelectOutput(sc4RPin(t, s, 8, 0)).Valid, "s.selectOutput(term51RPin(&s, 8, 0)) == null")
	expect(t, !s.SelectOutput(sc4RPin(t, s, 0, 3)).Valid, "s.selectOutput(term51RPin(&s, 0, 3)) == null")
}

// Zig: test "term51: selectOutput: wrapped output and multiple blocks"
func TestSc4_selectOutput_wrapped_output_and_multiple_blocks(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 12, 0)
	defer s.Deinit()
	s.CursorSetSemanticContent(sc4Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc4Write(t, s, "$ ")
	s.CursorSetSemanticContent(sc4Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc4Write(t, s, "a")
	s.CursorSetSemanticContent(sc4Output())
	sc4Write(t, s, "\nAAAAABBB\n")
	s.CursorSetSemanticContent(sc4Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc4Write(t, s, "$ ")
	s.CursorSetSemanticContent(sc4Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc4Write(t, s, "b")
	s.CursorSetSemanticContent(sc4Output())
	sc4Write(t, s, "\nCC")

	// First block: rows 1-2 (wrapped).
	sc4RExpectSel(t, s, s.SelectOutput(sc4RPin(t, s, 1, 2)), 0, 1, 2, 2)
	// Second block: row 4.
	sc4RExpectSel(t, s, s.SelectOutput(sc4RPin(t, s, 1, 4)), 0, 4, 1, 4)
	text, err := s.SelectionString(screenSelectionStringDefault(zigUnwrap(t, s.SelectOutput(sc4RPin(t, s, 0, 1)))))
	zigTry(t, err)
	expectEqualStrings(t, "AAAAABBB", string(text))
}

// Zig: test "term51: selectOutput: output spans a page boundary"
func TestSc4_selectOutput_output_spans_a_page_boundary(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 3, 2048)
	defer s.Deinit()

	r := sc4RFillFirstPage(t, s)
	// Cursor is on the last row of the first page.
	s.CursorSetSemanticContent(sc4Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc4Write(t, s, "$ ")
	s.CursorSetSemanticContent(sc4Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc4Write(t, s, "ls")
	s.CursorSetSemanticContent(sc4Output())
	sc4Write(t, s, "\nfoo\nbar\n")
	s.CursorSetSemanticContent(sc4Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc4Write(t, s, "$ ")

	sel := s.SelectOutput(sc4RPin(t, s, 0, r+1))
	sc4RExpectSel(t, s, sel, 0, r, 2, r+1)
	expect(t, zigUnwrap(t, sel).Start().node != sc4RPin(t, s, 0, r-1).node, "sel.?.start().node != term51RPin(&s, 0, r - 1).node")
}

// Zig: test "term51: promptClickMove: cursor must be on input content"
func TestSc4_promptClickMove_cursor_must_be_on_input_content(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 20, 5, 0)
	defer s.Deinit()
	s.semantic_prompt.click = sc4Click(OSCParsersSemanticPromptClickLine)
	sc4RWritePrompt(t, s, "> ", "hello")

	// Cursor on an input cell and in input mode.
	s.CursorAbsolute(2, 0)
	expectEqual(t, sc4RMove(0, 2), s.PromptClickMove(sc4RPin(t, s, 4, 0)))

	// Cursor cell is input even though the cursor itself is not in input
	// mode: still moves.
	s.CursorSetSemanticContent(sc4Output())
	expect(t, s.cursor.page_cell.semantic_content == PageCellSemanticContentInput, "s.cursor.page_cell.semantic_content == .input")
	expectEqual(t, sc4RMove(0, 2), s.PromptClickMove(sc4RPin(t, s, 4, 0)))

	// Cursor in neither input mode nor on an input cell: no movement.
	s.CursorAbsolute(10, 0)
	expect(t, s.cursor.page_cell.semantic_content == PageCellSemanticContentOutput, "s.cursor.page_cell.semantic_content == .output")
	expectEqual(t, ScreenPromptClickMoveZero, s.PromptClickMove(sc4RPin(t, s, 4, 0)))

	// Cursor in input mode but on a non-input cell: moves left over the
	// input cells between click and cursor.
	s.CursorSetSemanticContent(sc4Input(ScreenCursorSetSemanticContentInputClearExplicit))
	expectEqual(t, sc4RMove(3, 0), s.PromptClickMove(sc4RPin(t, s, 4, 0)))

	// Cursor in prompt mode on a prompt cell: no movement.
	s.CursorSetSemanticContent(sc4Prompt(OSCParsersSemanticPromptPromptKindInitial))
	s.CursorAbsolute(0, 0)
	expect(t, s.cursor.page_cell.semantic_content == PageCellSemanticContentPrompt, "s.cursor.page_cell.semantic_content == .prompt")
	expectEqual(t, ScreenPromptClickMoveZero, s.PromptClickMove(sc4RPin(t, s, 4, 0)))

	// Cursor on an output cell with output mode: no movement.
	s.CursorSetSemanticContent(sc4Output())
	s.CursorAbsolute(15, 2)
	expectEqual(t, ScreenPromptClickMoveZero, s.PromptClickMove(sc4RPin(t, s, 4, 0)))
}

// Zig: test "term51: promptClickMove: click mode selects the strategy"
func TestSc4_promptClickMove_click_mode_selects_the_strategy(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 20, 5, 0)
	defer s.Deinit()
	sc4RWritePrompt(t, s, "> ", "hello")
	s.CursorAbsolute(2, 0)

	// The default is disabled.
	expectEqual(t, ScreenSemanticPromptSemanticClick{Tag: ScreenSemanticPromptSemanticClickNone}, s.semantic_prompt.click)
	expectEqual(t, ScreenPromptClickMoveZero, s.PromptClickMove(sc4RPin(t, s, 4, 0)))

	s.semantic_prompt.click = ScreenSemanticPromptSemanticClick{Tag: ScreenSemanticPromptSemanticClickClickEvents}
	expectEqual(t, ScreenPromptClickMoveZero, s.PromptClickMove(sc4RPin(t, s, 4, 0)))
	expectEqual(t, ScreenPromptClickMoveZero, s.PromptClickMove(sc4RPin(t, s, 0, 0)))

	// Every line based click mode behaves the same (and left works too).
	modes := []ScreenSemanticPromptSemanticClick{
		sc4Click(OSCParsersSemanticPromptClickLine),
		sc4Click(OSCParsersSemanticPromptClickMultiple),
		sc4Click(OSCParsersSemanticPromptClickConservativeVertical),
		sc4Click(OSCParsersSemanticPromptClickSmartVertical),
	}
	for _, mode := range modes {
		s.semantic_prompt.click = mode
		s.CursorAbsolute(2, 0)
		expectEqual(t, sc4RMove(0, 2), s.PromptClickMove(sc4RPin(t, s, 4, 0)))
		s.CursorAbsolute(6, 0)
		expectEqual(t, sc4RMove(2, 0), s.PromptClickMove(sc4RPin(t, s, 4, 0)))
	}
}

// Zig: test "term51: promptClickMove: click on the cursor cell is no movement"
func TestSc4_promptClickMove_click_on_the_cursor_cell_is_no_movement(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 20, 5, 0)
	defer s.Deinit()
	s.semantic_prompt.click = sc4Click(OSCParsersSemanticPromptClickLine)
	sc4RWritePrompt(t, s, "> ", "hello")

	s.CursorAbsolute(4, 0)
	expectEqual(t, ScreenPromptClickMoveZero, s.PromptClickMove(sc4RPin(t, s, 4, 0)))
	// One cell either side is one step.
	expectEqual(t, sc4RMove(0, 1), s.PromptClickMove(sc4RPin(t, s, 5, 0)))
	expectEqual(t, sc4RMove(1, 0), s.PromptClickMove(sc4RPin(t, s, 3, 0)))
}

// Zig: test "term51: promptClickLine: input without a prompt"
func TestSc4_promptClickLine_input_without_a_prompt(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 20, 5, 0)
	defer s.Deinit()
	s.CursorSetSemanticContent(sc4Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc4Write(t, s, "hello")

	s.CursorAbsolute(0, 0)
	expectEqual(t, sc4RMove(0, 3), s.promptClickLine(sc4RPin(t, s, 3, 0)))
	expectEqual(t, sc4RMove(0, 4), s.promptClickLine(sc4RPin(t, s, 4, 0)))
	// Beyond the input: every remaining input cell plus the end cell.
	expectEqual(t, sc4RMove(0, 5), s.promptClickLine(sc4RPin(t, s, 5, 0)))
	expectEqual(t, sc4RMove(0, 5), s.promptClickLine(sc4RPin(t, s, 19, 0)))

	s.CursorAbsolute(4, 0)
	expectEqual(t, sc4RMove(4, 0), s.promptClickLine(sc4RPin(t, s, 0, 0)))
	expectEqual(t, sc4RMove(1, 0), s.promptClickLine(sc4RPin(t, s, 3, 0)))
}

// Zig: test "term51: promptClickLine: right of cursor on prompt plus input"
func TestSc4_promptClickLine_right_of_cursor_on_prompt_plus_input(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 20, 5, 0)
	defer s.Deinit()
	sc4RWritePrompt(t, s, "> ", "hello")
	s.CursorAbsolute(2, 0)

	expectEqual(t, sc4RMove(0, 1), s.promptClickLine(sc4RPin(t, s, 3, 0)))
	// Last input cell.
	expectEqual(t, sc4RMove(0, 4), s.promptClickLine(sc4RPin(t, s, 6, 0)))
	// The first blank cell after the input and further: input count plus one.
	expectEqual(t, sc4RMove(0, 5), s.promptClickLine(sc4RPin(t, s, 7, 0)))
	expectEqual(t, sc4RMove(0, 5), s.promptClickLine(sc4RPin(t, s, 19, 0)))
}

// Zig: test "term51: promptClickLine: cursor on the blank cell after the input"
func TestSc4_promptClickLine_cursor_on_the_blank_cell_after_the_input(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 20, 5, 0)
	defer s.Deinit()
	sc4RWritePrompt(t, s, "> ", "hello")

	// The cursor is naturally after the input.
	expectEqual(t, SizeCellCountInt(7), s.cursor.x)
	expect(t, s.cursor.page_cell.semantic_content == PageCellSemanticContentOutput, "s.cursor.page_cell.semantic_content == .output")

	// To the right there is nothing: no +1 because the cursor cell is
	// not an input cell.
	expectEqual(t, sc4RMove(0, 0), s.promptClickLine(sc4RPin(t, s, 12, 0)))
	// Left: target inclusive.
	expectEqual(t, sc4RMove(1, 0), s.promptClickLine(sc4RPin(t, s, 6, 0)))
	expectEqual(t, sc4RMove(4, 0), s.promptClickLine(sc4RPin(t, s, 3, 0)))
	expectEqual(t, sc4RMove(5, 0), s.promptClickLine(sc4RPin(t, s, 2, 0)))
	// Clicking a prompt cell: every input cell left of the cursor.
	expectEqual(t, sc4RMove(5, 0), s.promptClickLine(sc4RPin(t, s, 1, 0)))
	expectEqual(t, sc4RMove(5, 0), s.promptClickLine(sc4RPin(t, s, 0, 0)))
}

// Zig: test "term51: promptClickLine: left of cursor in the middle of input"
func TestSc4_promptClickLine_left_of_cursor_in_the_middle_of_input(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 20, 5, 0)
	defer s.Deinit()
	sc4RWritePrompt(t, s, "> ", "hello")
	s.CursorAbsolute(5, 0)

	expectEqual(t, sc4RMove(1, 0), s.promptClickLine(sc4RPin(t, s, 4, 0)))
	expectEqual(t, sc4RMove(3, 0), s.promptClickLine(sc4RPin(t, s, 2, 0)))
	// Prompt cells are never counted, and never found.
	expectEqual(t, sc4RMove(3, 0), s.promptClickLine(sc4RPin(t, s, 1, 0)))
	expectEqual(t, sc4RMove(3, 0), s.promptClickLine(sc4RPin(t, s, 0, 0)))
}

// Zig: test "term51: promptClickLine: right prompt after the input"
func TestSc4_promptClickLine_right_prompt_after_the_input(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 20, 5, 0)
	defer s.Deinit()
	sc4RWritePrompt(t, s, "> ", "ab")
	s.CursorAbsolute(15, 0)
	s.CursorSetSemanticContent(sc4Prompt(OSCParsersSemanticPromptPromptKindRight))
	sc4Write(t, s, "RP")

	// Cursor on 'a': click on the right prompt: remaining input + end cell.
	s.CursorAbsolute(2, 0)
	expectEqual(t, sc4RMove(0, 2), s.promptClickLine(sc4RPin(t, s, 16, 0)))
	// Cursor on the last input cell.
	s.CursorAbsolute(3, 0)
	expectEqual(t, sc4RMove(0, 1), s.promptClickLine(sc4RPin(t, s, 16, 0)))
	// Cursor on the right prompt, click on the input.
	s.CursorAbsolute(15, 0)
	expectEqual(t, sc4RMove(1, 0), s.promptClickLine(sc4RPin(t, s, 3, 0)))
	expectEqual(t, sc4RMove(2, 0), s.promptClickLine(sc4RPin(t, s, 2, 0)))
	expectEqual(t, sc4RMove(2, 0), s.promptClickLine(sc4RPin(t, s, 0, 0)))
}

// Zig: test "term51: promptClickLine: non-input cells between input are skipped"
func TestSc4_promptClickLine_non_input_cells_between_input_are_skipped(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 20, 5, 0)
	defer s.Deinit()
	sc4RWritePrompt(t, s, "> ", "a")
	s.CursorSetSemanticContent(sc4Output())
	sc4Write(t, s, "xx")
	s.CursorSetSemanticContent(sc4Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc4Write(t, s, "b")

	s.CursorAbsolute(2, 0)
	expectEqual(t, sc4RMove(0, 1), s.promptClickLine(sc4RPin(t, s, 5, 0)))
	// Clicking a gap cell can never match (only input cells are targets),
	// so it behaves like clicking beyond the input.
	expectEqual(t, sc4RMove(0, 2), s.promptClickLine(sc4RPin(t, s, 3, 0)))
	expectEqual(t, sc4RMove(0, 2), s.promptClickLine(sc4RPin(t, s, 4, 0)))

	s.CursorAbsolute(5, 0)
	expectEqual(t, sc4RMove(1, 0), s.promptClickLine(sc4RPin(t, s, 2, 0)))
	// Gap cell as the target when moving left: all input cells counted.
	expectEqual(t, sc4RMove(1, 0), s.promptClickLine(sc4RPin(t, s, 3, 0)))
}

// Zig: test "term51: promptClickLine: input on a hard line break is limited to the cursor line"
func TestSc4_promptClickLine_input_on_a_hard_line_break_is_limited_to_the_cursor_line(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 20, 5, 0)
	defer s.Deinit()
	sc4RWritePrompt(t, s, "> ", "ab")
	sc4Write(t, s, "\n")
	s.CursorSetSemanticContent(sc4Prompt(OSCParsersSemanticPromptPromptKindContinuation))
	sc4Write(t, s, ". ")
	s.CursorSetSemanticContent(sc4Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc4Write(t, s, "cd")

	expect(t, sc4RPin(t, s, 0, 1).RowAndCell().row.semantic_prompt == PageRowSemanticPromptPromptContinuation, "term51RPin(&s, 0, 1).rowAndCell().row.semantic_prompt == .prompt_continuation")
	expect(t, !sc4RPin(t, s, 0, 0).RowAndCell().row.wrap, "!term51RPin(&s, 0, 0).rowAndCell().row.wrap")
	expect(t, !sc4RPin(t, s, 0, 1).RowAndCell().row.wrap_continuation, "!term51RPin(&s, 0, 1).rowAndCell().row.wrap_continuation")

	// From the first line to the second: remaining input of the first line
	// plus the end cell, nothing from the second line.
	s.CursorAbsolute(2, 0)
	expectEqual(t, sc4RMove(0, 2), s.promptClickLine(sc4RPin(t, s, 3, 1)))
	s.CursorAbsolute(3, 0)
	expectEqual(t, sc4RMove(0, 1), s.promptClickLine(sc4RPin(t, s, 2, 1)))

	// From the second line back to the first: only the second line counts.
	s.CursorAbsolute(3, 1)
	expectEqual(t, sc4RMove(1, 0), s.promptClickLine(sc4RPin(t, s, 2, 1)))
	expectEqual(t, sc4RMove(1, 0), s.promptClickLine(sc4RPin(t, s, 3, 0)))
	expectEqual(t, sc4RMove(1, 0), s.promptClickLine(sc4RPin(t, s, 2, 0)))
}

// Zig: test "term51: promptClickLine: soft wrapped input over three rows"
func TestSc4_promptClickLine_soft_wrapped_input_over_three_rows(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 6, 0)
	defer s.Deinit()
	sc4RWritePrompt(t, s, "> ", "abcdefghijk")

	// Rows: "> abc" / "defgh" / "ijk".
	expect(t, sc4RPin(t, s, 0, 0).RowAndCell().row.wrap, "term51RPin(&s, 0, 0).rowAndCell().row.wrap")
	expect(t, sc4RPin(t, s, 0, 1).RowAndCell().row.wrap, "term51RPin(&s, 0, 1).rowAndCell().row.wrap")
	expect(t, sc4RPin(t, s, 0, 1).RowAndCell().row.wrap_continuation, "term51RPin(&s, 0, 1).rowAndCell().row.wrap_continuation")
	expect(t, sc4RPin(t, s, 0, 2).RowAndCell().row.wrap_continuation, "term51RPin(&s, 0, 2).rowAndCell().row.wrap_continuation")
	expect(t, sc4RPin(t, s, 0, 2).RowAndCell().row.semantic_prompt == PageRowSemanticPromptPromptContinuation, "term51RPin(&s, 0, 2).rowAndCell().row.semantic_prompt == .prompt_continuation")

	// Right: from 'a' to 'j' is b..j = 9 cells.
	s.CursorAbsolute(2, 0)
	expectEqual(t, sc4RMove(0, 2), s.promptClickLine(sc4RPin(t, s, 4, 0)))
	expectEqual(t, sc4RMove(0, 3), s.promptClickLine(sc4RPin(t, s, 0, 1)))
	expectEqual(t, sc4RMove(0, 9), s.promptClickLine(sc4RPin(t, s, 1, 2)))
	// Beyond the end of the last row: all 10 input cells plus the end cell.
	expectEqual(t, sc4RMove(0, 11), s.promptClickLine(sc4RPin(t, s, 4, 2)))
	expectEqual(t, sc4RMove(0, 11), s.promptClickLine(sc4RPin(t, s, 3, 2)))

	// Left: from 'j' back to 'a' is i..a = 9 cells.
	s.CursorAbsolute(1, 2)
	expectEqual(t, sc4RMove(1, 0), s.promptClickLine(sc4RPin(t, s, 0, 2)))
	expectEqual(t, sc4RMove(2, 0), s.promptClickLine(sc4RPin(t, s, 4, 1)))
	expectEqual(t, sc4RMove(9, 0), s.promptClickLine(sc4RPin(t, s, 2, 0)))
	// Prompt cell: all the input before the cursor.
	expectEqual(t, sc4RMove(9, 0), s.promptClickLine(sc4RPin(t, s, 0, 0)))
}

// Zig: test "term51: promptClickLine: cursor in the first and last column of wrapped rows"
func TestSc4_promptClickLine_cursor_in_the_first_and_last_column_of_wrapped_rows(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 4, 0)
	defer s.Deinit()
	sc4RWritePrompt(t, s, "> ", "abcdefgh")

	// Cursor in the last column of the first row: nothing left on that row.
	s.CursorAbsolute(4, 0)
	expectEqual(t, sc4RMove(0, 3), s.promptClickLine(sc4RPin(t, s, 2, 1)))
	expectEqual(t, sc4RMove(0, 1), s.promptClickLine(sc4RPin(t, s, 0, 1)))

	// Cursor in the first column of the second row: nothing left on that row.
	s.CursorAbsolute(0, 1)
	expectEqual(t, sc4RMove(2, 0), s.promptClickLine(sc4RPin(t, s, 3, 0)))
	expectEqual(t, sc4RMove(1, 0), s.promptClickLine(sc4RPin(t, s, 4, 0)))
}

// Zig: test "term51: promptClickLine: soft wrapped rows must be prompt continuations"
func TestSc4_promptClickLine_soft_wrapped_rows_must_be_prompt_continuations(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 4, 0)
	defer s.Deinit()
	sc4RWritePrompt(t, s, "> ", "abcdefgh")

	// Rows: "> abc" / "defgh".
	s.CursorAbsolute(2, 0)
	expectEqual(t, sc4RMove(0, 5), s.promptClickLine(sc4RPin(t, s, 2, 1)))

	// If the second row is no longer a prompt continuation the walk stops
	// after the cursor row, without the extra end cell.
	sc4RPin(t, s, 0, 1).RowAndCell().row.semantic_prompt = PageRowSemanticPromptNone
	expectEqual(t, sc4RMove(0, 2), s.promptClickLine(sc4RPin(t, s, 2, 1)))
	expectEqual(t, sc4RMove(0, 2), s.promptClickLine(sc4RPin(t, s, 4, 1)))

	// A row marked as an initial prompt is not a continuation either.
	sc4RPin(t, s, 0, 1).RowAndCell().row.semantic_prompt = PageRowSemanticPromptPrompt
	expectEqual(t, sc4RMove(0, 2), s.promptClickLine(sc4RPin(t, s, 2, 1)))
}

// Zig: test "term51: promptClickLine: end cell is only added when the line really ends"
func TestSc4_promptClickLine_end_cell_is_only_added_when_the_line_really_ends(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 6, 2, 0)
	defer s.Deinit()
	sc4RWritePrompt(t, s, "> ", "abcdef")

	// Rows: "> abcd" / "ef".
	s.CursorAbsolute(2, 0)
	expectEqual(t, sc4RMove(0, 6), s.promptClickLine(sc4RPin(t, s, 5, 1)))

	// Row 1 is the last row of the screen and not wrapped: end cell added.
	// With its wrap flag set, the iteration just runs out and nothing is added.
	sc4RPin(t, s, 0, 1).RowAndCell().row.wrap = true
	expectEqual(t, sc4RMove(0, 5), s.promptClickLine(sc4RPin(t, s, 5, 1)))
	sc4RPin(t, s, 0, 1).RowAndCell().row.wrap = false
	expectEqual(t, sc4RMove(0, 6), s.promptClickLine(sc4RPin(t, s, 5, 1)))
}

// Zig: test "term51: promptClickLine: wide characters count both cells"
func TestSc4_promptClickLine_wide_characters_count_both_cells(t *testing.T) {
	// Upstream pins: the spacer tail of a wide input character also carries
	// the input semantic content, so it is counted as an extra move.
	testStdCapacity(t)
	s := sc4Init(t, 20, 5, 0)
	defer s.Deinit()
	sc4RWritePrompt(t, s, "> ", "a世b")

	expect(t, sc4RPin(t, s, 4, 0).RowAndCell().cell.semantic_content == PageCellSemanticContentInput, "term51RPin(&s, 4, 0).rowAndCell().cell.semantic_content == .input")

	s.CursorAbsolute(2, 0)
	expectEqual(t, sc4RMove(0, 3), s.promptClickLine(sc4RPin(t, s, 5, 0)))
	expectEqual(t, sc4RMove(0, 1), s.promptClickLine(sc4RPin(t, s, 3, 0)))

	s.CursorAbsolute(6, 0)
	expectEqual(t, sc4RMove(4, 0), s.promptClickLine(sc4RPin(t, s, 2, 0)))
	expectEqual(t, sc4RMove(1, 0), s.promptClickLine(sc4RPin(t, s, 5, 0)))
}

// Zig: test "term51: promptClickLine: input wrapped across a page boundary"
func TestSc4_promptClickLine_input_wrapped_across_a_page_boundary(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 3, 2048)
	defer s.Deinit()

	r := sc4RFillFirstPage(t, s)
	sc4RWritePrompt(t, s, "> ", "abcdefghijkl")
	expect(t, sc4RPin(t, s, 0, r-1).node != sc4RPin(t, s, 0, r).node, "term51RPin(&s, 0, r - 1).node != term51RPin(&s, 0, r).node")

	// Rows: "> abcdefgh" (page 0) / "ijkl" (page 1).
	sc4RCursorTo(t, s, 2, r-1)
	expectEqual(t, sc4RMove(0, 9), s.promptClickLine(sc4RPin(t, s, 1, r)))
	expectEqual(t, sc4RMove(0, 7), s.promptClickLine(sc4RPin(t, s, 9, r-1)))
	expectEqual(t, sc4RMove(0, 12), s.promptClickLine(sc4RPin(t, s, 9, r)))

	sc4RCursorTo(t, s, 1, r)
	expectEqual(t, sc4RMove(9, 0), s.promptClickLine(sc4RPin(t, s, 2, r-1)))
	expectEqual(t, sc4RMove(1, 0), s.promptClickLine(sc4RPin(t, s, 0, r)))
}

// Zig: test "term51: promptClickLine: line wrap flag set on the cursor row of a hard line"
func TestSc4_promptClickLine_line_wrap_flag_set_on_the_cursor_row_of_a_hard_line(t *testing.T) {
	// A wrapped *first* row followed by a hard break: only the first logical
	// line is walked.
	testStdCapacity(t)
	s := sc4Init(t, 5, 5, 0)
	defer s.Deinit()
	sc4RWritePrompt(t, s, "> ", "abcde")
	sc4Write(t, s, "\n")
	s.CursorSetSemanticContent(sc4Prompt(OSCParsersSemanticPromptPromptKindContinuation))
	sc4Write(t, s, "..")
	s.CursorSetSemanticContent(sc4Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc4Write(t, s, "xy")

	// Rows: "> abc" / "de" (wrap continuation) / "..xy" (hard continuation).
	s.CursorAbsolute(2, 0)
	// Click in the last (hard) line: walk covers rows 0 and 1, the end cell
	// is added because row 1 is not wrapped.
	expectEqual(t, sc4RMove(0, 5), s.promptClickLine(sc4RPin(t, s, 3, 2)))
	// Cursor in the last line moving left only sees its own row.
	s.CursorAbsolute(4, 2)
	expectEqual(t, sc4RMove(1, 0), s.promptClickLine(sc4RPin(t, s, 3, 2)))
	expectEqual(t, sc4RMove(2, 0), s.promptClickLine(sc4RPin(t, s, 2, 2)))
	expectEqual(t, sc4RMove(2, 0), s.promptClickLine(sc4RPin(t, s, 1, 1)))
}

// Zig: test "term51: lineIterator: starts at the given pin"
func TestSc4_lineIterator_starts_at_the_given_pin(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 4, 3, 0)
	defer s.Deinit()

	start := sc4RPin(t, s, 2, 1)
	it := s.LineIterator(start)
	expect(t, it.screen == s, "it.screen == &s")
	expect(t, it.current.Valid, "it.current != null")
	expect(t, it.current.Value.Eql(start), "it.current.?.eql(start)")
}

// Zig: test "term51: LineIterator.next: default iterator has no current line"
func TestSc4_LineIterator_next_default_iterator_has_no_current_line(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 4, 3, 0)
	defer s.Deinit()

	it := ScreenLineIterator{screen: s}
	expect(t, !it.current.Valid, "it.current == null")
	expect(t, !it.Next().Valid, "it.next() == null")
	expect(t, !it.Next().Valid, "it.next() == null")
}

// Zig: test "term51: LineIterator.next: yields every row including blank ones, then null"
func TestSc4_LineIterator_next_yields_every_row_including_blank_ones_then_null(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 3, 4, 0)
	defer s.Deinit()
	sc4Write(t, s, "ab\n\n cd")

	it := s.LineIterator(sc4RPin(t, s, 0, 0))
	// Whitespace is not trimmed by the iterator.
	sc4RExpectSel(t, s, it.Next(), 0, 0, 2, 0)
	sc4RExpectSel(t, s, it.Next(), 0, 1, 2, 1)
	sc4RExpectSel(t, s, it.Next(), 0, 2, 2, 2)
	sc4RExpectSel(t, s, it.Next(), 0, 3, 2, 3)
	expect(t, !it.current.Valid, "it.current == null")
	expect(t, !it.Next().Valid, "it.next() == null")
	expect(t, !it.Next().Valid, "it.next() == null")
}

// Zig: test "term51: LineIterator.next: starting mid-row and mid-wrap yields the whole line"
func TestSc4_LineIterator_next_starting_mid_row_and_mid_wrap_yields_the_whole_line(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 4, 0)
	defer s.Deinit()
	sc4Write(t, s, "1ABCD2EFGH\nzz")

	// Start in the middle of the second row of the wrapped line.
	it := s.LineIterator(sc4RPin(t, s, 3, 1))
	sc4RExpectSel(t, s, it.Next(), 0, 0, 4, 1)
	sc4RExpectSel(t, s, it.Next(), 0, 2, 4, 2)
	sc4RExpectSel(t, s, it.Next(), 0, 3, 4, 3)
	expect(t, !it.Next().Valid, "it.next() == null")

	// Start at the first row.
	it2 := s.LineIterator(sc4RPin(t, s, 4, 0))
	sc4RExpectSel(t, s, it2.Next(), 0, 0, 4, 1)
	sc4RExpectSel(t, s, it2.Next(), 0, 2, 4, 2)
}

// Zig: test "term51: LineIterator.next: soft wrapped line content"
func TestSc4_LineIterator_next_soft_wrapped_line_content(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 3, 5, 0)
	defer s.Deinit()
	sc4Write(t, s, "abcdefgh\nij")

	it := s.LineIterator(sc4RPin(t, s, 0, 0))
	expected := []string{"abcdefgh", "ij"}
	for _, e := range expected {
		opts := screenSelectionStringDefault(zigUnwrap(t, it.Next()))
		opts.trim = true
		text, err := s.SelectionString(opts)
		zigTry(t, err)
		expectEqualStrings(t, e, string(text))
	}
}

// Zig: test "term51: LineIterator.next: ignores semantic prompt boundaries"
func TestSc4_LineIterator_next_ignores_semantic_prompt_boundaries(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 6, 3, 0)
	defer s.Deinit()
	sc4RWritePrompt(t, s, "$ ", "ls")
	s.CursorSetSemanticContent(sc4Output())
	sc4Write(t, s, "!!")

	it := s.LineIterator(sc4RPin(t, s, 3, 0))
	sc4RExpectSel(t, s, it.Next(), 0, 0, 5, 0)
}

// Zig: test "term51: LineIterator.next: stops when a line cannot be selected"
func TestSc4_LineIterator_next_stops_when_a_line_cannot_be_selected(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 4, 3, 0)
	defer s.Deinit()
	sc4Write(t, s, "ab\ncd\nef")

	// The last row claims to wrap but there is nothing after it.
	sc4RPin(t, s, 0, 2).RowAndCell().row.wrap = true

	it := s.LineIterator(sc4RPin(t, s, 0, 0))
	sc4RExpectSel(t, s, it.Next(), 0, 0, 3, 0)
	sc4RExpectSel(t, s, it.Next(), 0, 1, 3, 1)
	expect(t, it.current.Valid, "it.current != null")
	expect(t, !it.Next().Valid, "it.next() == null")
	expect(t, !it.current.Valid, "it.current == null")
	expect(t, !it.Next().Valid, "it.next() == null")
}

// Zig: test "term51: LineIterator.next: walks scrollback history and pages"
func TestSc4_LineIterator_next_walks_scrollback_history_and_pages(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 3, 2048)
	defer s.Deinit()

	r := sc4RFillFirstPage(t, s)
	sc4Write(t, s, "aaaaabbbbbccccc\nzz")

	it := s.LineIterator(sc4RPin(t, s, 0, r-2))
	sc4RExpectSel(t, s, it.Next(), 0, r-2, 9, r-2)
	// The wrapped line starts in page 0 and ends in page 1.
	sel := zigUnwrap(t, it.Next())
	sc4RExpectSel(t, s, Some(sel), 0, r-1, 9, r)
	expect(t, sel.Start().node != sel.End().node, "sel.start().node != sel.end().node")
	sc4RExpectSel(t, s, it.Next(), 0, r+1, 9, r+1)
	// Active area has 3 rows: the screen's last row is r + 1.
	expect(t, !it.Next().Valid, "it.next() == null")
}

// Zig: test "term51: dumpString: wrapped rows are joined only when unwrapping"
func TestSc4_dumpString_wrapped_rows_are_joined_only_when_unwrapping(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 3, 0)
	defer s.Deinit()
	sc4Write(t, s, "1ABCD2EFGH\nx")
	tl := s.pages.GetTopLeft(PointTagScreen)

	joined := sc4RDump(t, s, tl, None[PageListPin](), true)
	expectEqualStrings(t, "1ABCD2EFGH\nx", string(joined))

	visual := sc4RDump(t, s, tl, None[PageListPin](), false)
	expectEqualStrings(t, "1ABCD\n2EFGH\nx", string(visual))

	// The default of the option is to unwrap.
	aw := oscWriterAllocatingInit()
	defer aw.deinit()
	zigTry(t, s.DumpString(&aw.writer, screenDumpStringOptionsDefault(tl)))
	expectEqualStrings(t, "1ABCD2EFGH\nx", string(aw.written()))
}

// Zig: test "term51: dumpString: empty screen and single cell"
func TestSc4_dumpString_empty_screen_and_single_cell(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 3, 0)
	defer s.Deinit()
	tl := s.pages.GetTopLeft(PointTagScreen)

	empty := sc4RDump(t, s, tl, None[PageListPin](), true)
	expectEqualStrings(t, "", string(empty))

	sc4Write(t, s, "x")
	one := sc4RDump(t, s, tl, Some(tl), true)
	expectEqualStrings(t, "x", string(one))

	// The end pin defaults to the bottom right of the screen.
	all := sc4RDump(t, s, tl, None[PageListPin](), true)
	expectEqualStrings(t, "x", string(all))
}

// Zig: test "term51: dumpString: blank rows and trailing spaces are kept as written"
func TestSc4_dumpString_blank_rows_and_trailing_spaces_are_kept_as_written(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 5, 0)
	defer s.Deinit()
	sc4Write(t, s, "\n\nab  \ncd")
	tl := s.pages.GetTopLeft(PointTagScreen)

	// Leading blank rows give newlines, trailing spaces are written cells
	// and kept (no trimming), trailing never-written rows give nothing.
	out := sc4RDump(t, s, tl, None[PageListPin](), false)
	expectEqualStrings(t, "\n\nab  \ncd", string(out))
}

// Zig: test "term51: dumpString: row range and columns of the end pins"
func TestSc4_dumpString_row_range_and_columns_of_the_end_pins(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 4, 0)
	defer s.Deinit()
	sc4Write(t, s, "abcde\nfghij\nklmno\npqrst")

	// A single row.
	{
		out := sc4RDump(t, s, sc4RPin(t, s, 0, 1), Some(sc4RPin(t, s, 4, 1)), true)
		expectEqualStrings(t, "fghij", string(out))
	}
	// Several rows from column zero to the last column.
	{
		out := sc4RDump(t, s, sc4RPin(t, s, 0, 1), Some(sc4RPin(t, s, 4, 2)), true)
		expectEqualStrings(t, "fghij\nklmno", string(out))
	}
	// From the given row to the end of the screen.
	{
		out := sc4RDump(t, s, sc4RPin(t, s, 0, 2), None[PageListPin](), true)
		expectEqualStrings(t, "klmno\npqrst", string(out))
	}
	// The x of the pins is honoured (it is a non-rectangular selection),
	// in contrast to the doc comment saying it is ignored.
	{
		out := sc4RDump(t, s, sc4RPin(t, s, 2, 0), Some(sc4RPin(t, s, 1, 2)), true)
		expectEqualStrings(t, "cde\nfghij\nkl", string(out))
	}
	// Reversed pins give the same text.
	{
		out := sc4RDump(t, s, sc4RPin(t, s, 1, 2), Some(sc4RPin(t, s, 2, 0)), true)
		expectEqualStrings(t, "cde\nfghij\nkl", string(out))
	}
}

// Zig: test "term51: dumpString: wide characters and graphemes"
func TestSc4_dumpString_wide_characters_and_graphemes(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 6, 3, 0)
	defer s.Deinit()
	sc4Write(t, s, "a世b\néx")
	tl := s.pages.GetTopLeft(PointTagScreen)

	out := sc4RDump(t, s, tl, None[PageListPin](), false)
	expectEqualStrings(t, "a世b\néx", string(out))
}

// Zig: test "term51: dumpString: history and multiple pages"
func TestSc4_dumpString_history_and_multiple_pages(t *testing.T) {
	testStdCapacity(t)
	func() {
		s := sc4Init(t, 3, 2, 10)
		defer s.Deinit()
		sc4Write(t, s, "a\nb\nc\nd")

		all := sc4RDump(t, s, s.pages.GetTopLeft(PointTagScreen), None[PageListPin](), true)
		expectEqualStrings(t, "a\nb\nc\nd", string(all))

		tail := sc4RDump(t, s, sc4RPin(t, s, 0, 1), None[PageListPin](), true)
		expectEqualStrings(t, "b\nc\nd", string(tail))
	}()
	func() {
		s := sc4Init(t, 10, 3, 2048)
		defer s.Deinit()
		r := sc4RFillFirstPage(t, s)
		sc4Write(t, s, "abc\ndef")
		expect(t, sc4RPin(t, s, 0, r-1).node != sc4RPin(t, s, 0, r).node, "term51RPin(&s, 0, r - 1).node != term51RPin(&s, 0, r).node")

		out := sc4RDump(t, s, sc4RPin(t, s, 0, r-1), None[PageListPin](), true)
		expectEqualStrings(t, "abc\ndef", string(out))

		// Leading blank rows of the first page are emitted as newlines.
		lead := sc4RDump(t, s, s.pages.GetTopLeft(PointTagScreen), None[PageListPin](), true)
		expectEqual(t, int(r-1), strings.Count(string(lead), "\n")-1)
		expect(t, bytes.HasSuffix(lead, []uint8("abc\ndef")), "std.mem.endsWith(u8, lead, \"abc\\ndef\")")
	}()
}

// Zig: test "term51: dumpString: writer failure is propagated"
func TestSc4_dumpString_writer_failure_is_propagated(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 3, 0)
	defer s.Deinit()
	sc4Write(t, s, "hello world")

	var buf [2]uint8
	w := oscWriter{buffer: buf[:]}
	sc4ExpectError(t, ErrWriteFailed, s.DumpString(&w, screenDumpStringOptionsDefault(s.pages.GetTopLeft(PointTagScreen))))

	// A big enough fixed buffer succeeds and is exact.
	var big [64]uint8
	w2 := oscWriter{buffer: big[:]}
	zigTry(t, s.DumpString(&w2, screenDumpStringOptionsDefault(s.pages.GetTopLeft(PointTagScreen))))
	expectEqualStrings(t, "hello world", string(w2.buffer[:w2.end]))
}

// Zig: test "term51: dumpStringAlloc: point tags select the region and wrap visually"
func TestSc4_dumpStringAlloc_point_tags_select_the_region_and_wrap_visually(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 3, 2, 10)
	defer s.Deinit()
	sc4Write(t, s, "abcdefg\nh\ni")

	// Screen rows: abc / def / g / h / i. History: first three, active: h, i.
	{
		out, err := s.DumpStringAlloc(sc4Tag(PointTagScreen))
		zigTry(t, err)
		expectEqualStrings(t, "abc\ndef\ng\nh\ni", string(out))
	}
	{
		out, err := s.DumpStringAlloc(sc4Tag(PointTagHistory))
		zigTry(t, err)
		expectEqualStrings(t, "abc\ndef\ng", string(out))
	}
	{
		out, err := s.DumpStringAlloc(sc4Tag(PointTagActive))
		zigTry(t, err)
		expectEqualStrings(t, "h\ni", string(out))
	}
	{
		out, err := s.DumpStringAlloc(sc4Tag(PointTagViewport))
		zigTry(t, err)
		expectEqualStrings(t, "h\ni", string(out))
	}
	// Only the tag matters, not the coordinates.
	{
		out, err := s.DumpStringAlloc(Point{Tag: PointTagActive, Active: PointCoordinate{x: 2, y: 1}})
		zigTry(t, err)
		expectEqualStrings(t, "h\ni", string(out))
	}
	// Scrolled viewport.
	s.Scroll(ScreenScroll{Tag: ScreenScrollTop})
	{
		out, err := s.DumpStringAlloc(sc4Tag(PointTagViewport))
		zigTry(t, err)
		expectEqualStrings(t, "abc\ndef", string(out))
	}
}

// Zig: test "term51: dumpStringAlloc: history of a screen without history is an error"
func TestSc4_dumpStringAlloc_history_of_a_screen_without_history_is_an_error(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 3, 2, 10)
	defer s.Deinit()
	sc4Write(t, s, "a\nb")

	_, err := s.DumpStringAlloc(sc4Tag(PointTagHistory))
	sc4ExpectError(t, ErrUnknownPoint, err)
	out, err := s.DumpStringAlloc(sc4Tag(PointTagActive))
	zigTry(t, err)
	expectEqualStrings(t, "a\nb", string(out))
}

// Zig: test "term51: dumpStringAlloc: empty screen"
func TestSc4_dumpStringAlloc_empty_screen(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 3, 2, 0)
	defer s.Deinit()

	out, err := s.DumpStringAlloc(sc4Tag(PointTagScreen))
	zigTry(t, err)
	expectEqualStrings(t, "", string(out))
}

// term51: dumpStringAlloc: allocation failures surface as errors: not
// translated (FailingAllocator), see scripts/not-translated-sc4.txt.

// Zig: test "term51: dumpStringAllocUnwrapped: soft wrapped rows are joined"
func TestSc4_dumpStringAllocUnwrapped_soft_wrapped_rows_are_joined(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 3, 2, 10)
	defer s.Deinit()
	sc4Write(t, s, "abcdefg\nh\ni")

	// Wrapped rows joined; hard breaks kept.
	{
		out, err := s.DumpStringAllocUnwrapped(sc4Tag(PointTagScreen))
		zigTry(t, err)
		expectEqualStrings(t, "abcdefg\nh\ni", string(out))
	}
	{
		out, err := s.DumpStringAllocUnwrapped(sc4Tag(PointTagHistory))
		zigTry(t, err)
		expectEqualStrings(t, "abcdefg", string(out))
	}
	{
		out, err := s.DumpStringAllocUnwrapped(sc4Tag(PointTagActive))
		zigTry(t, err)
		expectEqualStrings(t, "h\ni", string(out))
	}
	{
		out, err := s.DumpStringAllocUnwrapped(Point{Tag: PointTagViewport, Viewport: PointCoordinate{x: 1, y: 1}})
		zigTry(t, err)
		expectEqualStrings(t, "h\ni", string(out))
	}
}

// Zig: test "term51: dumpStringAllocUnwrapped: errors"
func TestSc4_dumpStringAllocUnwrapped_errors(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 3, 2, 10)
	defer s.Deinit()
	sc4Write(t, s, "a\nb")

	_, err := s.DumpStringAllocUnwrapped(sc4Tag(PointTagHistory))
	sc4ExpectError(t, ErrUnknownPoint, err)

	// Zig-only: var fail_index: usize = 0;
	// Zig-only: while (fail_index < 50) : (fail_index += 1) {
	// Zig-only:     var failing = std.testing.FailingAllocator.init(std.testing.allocator, .{ .fail_index = fail_index });
	// Zig-only:     const result = s.dumpStringAllocUnwrapped(failing.allocator(), .{ .screen = .{} });
	// Zig-only:     if (result) |out| { ... return; } else |err| { try std.testing.expect(err == error.WriteFailed or err == error.OutOfMemory); }
	// Zig-only: }
	// Zig-only: return error.TestUnexpectedResult;
}

// Zig: test "term51: testWriteString: narrow cells, cursor advance and pending wrap"
func TestSc4_testWriteString_narrow_cells_cursor_advance_and_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 3, 0)
	defer s.Deinit()

	// Empty string is a no-op.
	sc4Write(t, s, "")
	expectEqual(t, SizeCellCountInt(0), s.cursor.x)
	expect(t, !s.cursor.pending_wrap, "!s.cursor.pending_wrap")

	sc4Write(t, s, "abcd")
	expectEqual(t, SizeCellCountInt(4), s.cursor.x)
	expect(t, !s.cursor.pending_wrap, "!s.cursor.pending_wrap")

	// Filling the row sets pending wrap but does not wrap yet.
	sc4Write(t, s, "e")
	expectEqual(t, SizeCellCountInt(4), s.cursor.x)
	expectEqual(t, SizeCellCountInt(0), s.cursor.y)
	expect(t, s.cursor.pending_wrap, "s.cursor.pending_wrap")
	expect(t, !sc4RPin(t, s, 0, 0).RowAndCell().row.wrap, "!term51RPin(&s, 0, 0).rowAndCell().row.wrap")
	for x, c := range []uint8("abcde") {
		cell := sc4RPin(t, s, SizeCellCountInt(x), 0).RowAndCell().cell
		expectEqual(t, uint32(c), cell.content.codepoint)
		expectEqual(t, PageCellWideNarrow, cell.wide)
		expectEqual(t, StyleDefaultId, cell.style_id)
	}

	// The next character wraps.
	sc4Write(t, s, "f")
	expectEqual(t, SizeCellCountInt(1), s.cursor.x)
	expectEqual(t, SizeCellCountInt(1), s.cursor.y)
	expect(t, !s.cursor.pending_wrap, "!s.cursor.pending_wrap")
	expect(t, sc4RPin(t, s, 0, 0).RowAndCell().row.wrap, "term51RPin(&s, 0, 0).rowAndCell().row.wrap")
	expect(t, !sc4RPin(t, s, 0, 0).RowAndCell().row.wrap_continuation, "!term51RPin(&s, 0, 0).rowAndCell().row.wrap_continuation")
	expect(t, sc4RPin(t, s, 0, 1).RowAndCell().row.wrap_continuation, "term51RPin(&s, 0, 1).rowAndCell().row.wrap_continuation")
	expect(t, !sc4RPin(t, s, 0, 1).RowAndCell().row.wrap, "!term51RPin(&s, 0, 1).rowAndCell().row.wrap")
	expectEqual(t, uint32('f'), sc4RPin(t, s, 0, 1).RowAndCell().cell.content.codepoint)
}

// Zig: test "term51: testWriteString: invalid UTF-8 is rejected before writing"
func TestSc4_testWriteString_invalid_UTF_8_is_rejected_before_writing(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 3, 0)
	defer s.Deinit()

	sc4ExpectError(t, ErrInvalidUtf8, s.TestWriteString([]uint8("\xff")))
	sc4ExpectError(t, ErrInvalidUtf8, s.TestWriteString([]uint8("ab\xc3")))
	sc4ExpectError(t, ErrInvalidUtf8, s.TestWriteString([]uint8("ab\x80cd")))
	// Surrogates and overlong encodings are rejected too.
	sc4ExpectError(t, ErrInvalidUtf8, s.TestWriteString([]uint8("\xed\xa0\x80")))
	sc4ExpectError(t, ErrInvalidUtf8, s.TestWriteString([]uint8("\xc0\x80")))

	// The whole string is validated first, so nothing was written.
	expectEqual(t, SizeCellCountInt(0), s.cursor.x)
	out, err := s.DumpStringAlloc(sc4Tag(PointTagScreen))
	zigTry(t, err)
	expectEqualStrings(t, "", string(out))
}

// Zig: test "term51: testWriteString: newline moves down, resets wrap and scrolls at the bottom"
func TestSc4_testWriteString_newline_moves_down_resets_wrap_and_scrolls_at_the_bottom(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 3, 2, 0)
	defer s.Deinit()

	sc4Write(t, s, "abc\n")
	// The explicit newline resets pending wrap and does not set wrap.
	expect(t, !s.cursor.pending_wrap, "!s.cursor.pending_wrap")
	expectEqual(t, SizeCellCountInt(0), s.cursor.x)
	expectEqual(t, SizeCellCountInt(1), s.cursor.y)
	expect(t, !sc4RPin(t, s, 0, 0).RowAndCell().row.wrap, "!term51RPin(&s, 0, 0).rowAndCell().row.wrap")

	sc4Write(t, s, "de\n")
	// Scrolled: with no scrollback the first row is gone.
	expectEqual(t, SizeCellCountInt(1), s.cursor.y)
	out, err := s.DumpStringAlloc(sc4Tag(PointTagScreen))
	zigTry(t, err)
	expectEqualStrings(t, "de", string(out))

	// A newline with a column offset goes back to column zero.
	sc4Write(t, s, "xy\nz")
	expectEqual(t, SizeCellCountInt(1), s.cursor.x)
}

// Zig: test "term51: testWriteString: carriage return and nul are plain cells"
func TestSc4_testWriteString_carriage_return_and_nul_are_plain_cells(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 6, 2, 0)
	defer s.Deinit()

	// Only '\n' is special. A '\r' is printed into a cell.
	sc4Write(t, s, "a\rb")
	expectEqual(t, SizeCellCountInt(3), s.cursor.x)
	expectEqual(t, uint32('\r'), sc4RPin(t, s, 1, 0).RowAndCell().cell.content.codepoint)
	expectEqual(t, uint32('b'), sc4RPin(t, s, 2, 0).RowAndCell().cell.content.codepoint)

	// NUL advances the cursor but leaves a cell without text.
	sc4Write(t, s, "\x00")
	expectEqual(t, SizeCellCountInt(4), s.cursor.x)
	expect(t, !sc4RPin(t, s, 3, 0).RowAndCell().cell.HasText(), "!term51RPin(&s, 3, 0).rowAndCell().cell.hasText()")
}

// Zig: test "term51: testWriteString: codepoints up to 0xFF always take one cell"
func TestSc4_testWriteString_codepoints_up_to_0xFF_always_take_one_cell(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 10, 2, 0)
	defer s.Deinit()

	// U+00E9, U+00FF, U+00AD (soft hyphen), U+0085 and U+00A0.
	sc4Write(t, s, "éÿ­\u0085 ")
	expectEqual(t, SizeCellCountInt(5), s.cursor.x)
	expected := []uint32{0xe9, 0xff, 0xad, 0x85, 0xa0}
	for x, cp := range expected {
		cell := sc4RPin(t, s, SizeCellCountInt(x), 0).RowAndCell().cell
		expectEqual(t, cp, cell.content.codepoint)
		expectEqual(t, PageCellWideNarrow, cell.wide)
	}

	// U+0100 is the first codepoint looked up in the width table.
	sc4Write(t, s, "Ā")
	expectEqual(t, SizeCellCountInt(6), s.cursor.x)
	expectEqual(t, PageCellWideNarrow, sc4RPin(t, s, 5, 0).RowAndCell().cell.wide)
}

// Zig: test "term51: testWriteString: wide characters write a head and a tail"
func TestSc4_testWriteString_wide_characters_write_a_head_and_a_tail(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 5, 3, 0)
	defer s.Deinit()

	sc4Write(t, s, "a世")
	expectEqual(t, SizeCellCountInt(3), s.cursor.x)
	{
		head := sc4RPin(t, s, 1, 0).RowAndCell().cell
		tail := sc4RPin(t, s, 2, 0).RowAndCell().cell
		expectEqual(t, PageCellWideWide, head.wide)
		expectEqual(t, uint32(0x4e16), head.content.codepoint)
		expectEqual(t, PageCellWideSpacerTail, tail.wide)
		expect(t, !tail.HasText(), "!tail.hasText()")
	}

	// An emoji beyond the BMP is wide as well.
	sc4Write(t, s, "\U0001f600")
	expectEqual(t, PageCellWideWide, sc4RPin(t, s, 3, 0).RowAndCell().cell.wide)
	expectEqual(t, PageCellWideSpacerTail, sc4RPin(t, s, 4, 0).RowAndCell().cell.wide)
	// The tail was in the last column, so the cursor waits to wrap.
	expectEqual(t, SizeCellCountInt(4), s.cursor.x)
	expect(t, s.cursor.pending_wrap, "s.cursor.pending_wrap")
}

// Zig: test "term51: testWriteString: wide character at the last column leaves a spacer head"
func TestSc4_testWriteString_wide_character_at_the_last_column_leaves_a_spacer_head(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 4, 3, 0)
	defer s.Deinit()

	sc4Write(t, s, "abc世")
	{
		head := sc4RPin(t, s, 3, 0).RowAndCell()
		expectEqual(t, PageCellWideSpacerHead, head.cell.wide)
		expect(t, !head.cell.HasText(), "!head.cell.hasText()")
		expect(t, head.row.wrap, "head.row.wrap")
	}
	expect(t, sc4RPin(t, s, 0, 1).RowAndCell().row.wrap_continuation, "term51RPin(&s, 0, 1).rowAndCell().row.wrap_continuation")
	expectEqual(t, PageCellWideWide, sc4RPin(t, s, 0, 1).RowAndCell().cell.wide)
	expectEqual(t, PageCellWideSpacerTail, sc4RPin(t, s, 1, 1).RowAndCell().cell.wide)
	expectEqual(t, SizeCellCountInt(2), s.cursor.x)
	expectEqual(t, SizeCellCountInt(1), s.cursor.y)

	// A wide character that exactly fits at the end of a row does not wrap.
	s2 := sc4Init(t, 4, 3, 0)
	defer s2.Deinit()
	sc4Write(t, s2, "ab世")
	expectEqual(t, PageCellWideWide, sc4RPin(t, s2, 2, 0).RowAndCell().cell.wide)
	expectEqual(t, PageCellWideSpacerTail, sc4RPin(t, s2, 3, 0).RowAndCell().cell.wide)
	expect(t, s2.cursor.pending_wrap, "s2.cursor.pending_wrap")
	expect(t, !sc4RPin(t, s2, 0, 0).RowAndCell().row.wrap, "!term51RPin(&s2, 0, 0).rowAndCell().row.wrap")

	// A wide character when the cursor is pending-wrap at the end.
	sc4Write(t, s2, "世")
	expect(t, sc4RPin(t, s2, 0, 0).RowAndCell().row.wrap, "term51RPin(&s2, 0, 0).rowAndCell().row.wrap")
	expectEqual(t, PageCellWideWide, sc4RPin(t, s2, 0, 1).RowAndCell().cell.wide)
}

// Zig: test "term51: testWriteString: zero width codepoints become graphemes"
func TestSc4_testWriteString_zero_width_codepoints_become_graphemes(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 8, 3, 0)
	defer s.Deinit()

	sc4Write(t, s, "é̂")
	// The cursor does not advance for zero width codepoints.
	expectEqual(t, SizeCellCountInt(1), s.cursor.x)
	{
		pin := sc4RPin(t, s, 0, 0)
		cell := pin.RowAndCell().cell
		expectEqual(t, PageCellContentTagCodepointGrapheme, cell.content_tag)
		expectEqual(t, uint32('e'), cell.content.codepoint)
		expectEqualSlices(t, []uint32{0x301, 0x302}, zigUnwrap(t, pin.Grapheme(cell)))
	}

	// After a wide character the grapheme attaches to the wide head, not
	// to its spacer tail.
	sc4Write(t, s, "世́")
	expectEqual(t, SizeCellCountInt(3), s.cursor.x)
	{
		pin := sc4RPin(t, s, 1, 0)
		head := pin.RowAndCell().cell
		expectEqual(t, PageCellWideWide, head.wide)
		expectEqual(t, PageCellContentTagCodepointGrapheme, head.content_tag)
		expectEqualSlices(t, []uint32{0x301}, zigUnwrap(t, pin.Grapheme(head)))
		tail := sc4RPin(t, s, 2, 0).RowAndCell().cell
		expectEqual(t, PageCellContentTagCodepoint, tail.content_tag)
	}
}

// Zig: test "term51: testWriteString: zero width codepoint at pending wrap attaches to the previous cell"
func TestSc4_testWriteString_zero_width_codepoint_at_pending_wrap_attaches_to_the_previous_cell(t *testing.T) {
	// Upstream quirk (test helper only): with pending wrap the cursor is
	// still on the last written cell, but the helper looks one cell to the
	// LEFT of the cursor, so the grapheme lands on the cell before it.
	testStdCapacity(t)
	s := sc4Init(t, 3, 3, 0)
	defer s.Deinit()

	sc4Write(t, s, "abc")
	expect(t, s.cursor.pending_wrap, "s.cursor.pending_wrap")
	sc4Write(t, s, "́")

	b := sc4RPin(t, s, 1, 0)
	expectEqual(t, PageCellContentTagCodepointGrapheme, b.RowAndCell().cell.content_tag)
	expectEqualSlices(t, []uint32{0x301}, zigUnwrap(t, b.Grapheme(b.RowAndCell().cell)))
	c := sc4RPin(t, s, 2, 0).RowAndCell().cell
	expectEqual(t, PageCellContentTagCodepoint, c.content_tag)
}

// Zig: test "term51: testWriteString: cells take the cursor style and protection"
func TestSc4_testWriteString_cells_take_the_cursor_style_and_protection(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 6, 3, 0)
	defer s.Deinit()

	zigTry(t, s.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	s.cursor.protected = true
	id := s.cursor.style_id
	expect(t, id != StyleDefaultId, "id != style.default_id")

	sc4Write(t, s, "a")
	{
		rac := sc4RPin(t, s, 0, 0).RowAndCell()
		expectEqual(t, id, rac.cell.style_id)
		expect(t, rac.cell.protected, "rac.cell.protected")
		expect(t, rac.row.styled, "rac.row.styled")
	}
	{
		page := &s.cursor.page_pin.node.data
		// The cursor plus one cell.
		expectEqual(t, SizeCellCountInt(2), page.styles.RefCount(id))
	}

	// A wide character references the style from both of its cells.
	sc4Write(t, s, "世")
	{
		head := sc4RPin(t, s, 1, 0).RowAndCell().cell
		tail := sc4RPin(t, s, 2, 0).RowAndCell().cell
		expectEqual(t, id, head.style_id)
		expect(t, head.protected, "head.protected")
		// The tail cell is protected but carries no style id itself.
		expectEqual(t, StyleDefaultId, tail.style_id)
		expect(t, tail.protected, "tail.protected")
		page := &s.cursor.page_pin.node.data
		expectEqual(t, SizeCellCountInt(4), page.styles.RefCount(id))
	}

	// Unstyled cells do not mark the row styled.
	s.cursor.protected = false
	zigTry(t, s.SetAttribute(SGRAttribute{Tag: SGRAttributeTagResetBold}))
	sc4Write(t, s, "\nz")
	expect(t, !sc4RPin(t, s, 0, 1).RowAndCell().row.styled, "!term51RPin(&s, 0, 1).rowAndCell().row.styled")
	expect(t, !sc4RPin(t, s, 0, 1).RowAndCell().cell.protected, "!term51RPin(&s, 0, 1).rowAndCell().cell.protected")
}

// Zig: test "term51: testWriteString: cells take the cursor hyperlink"
func TestSc4_testWriteString_cells_take_the_cursor_hyperlink(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 4, 3, 0)
	defer s.Deinit()

	sc4Write(t, s, "a")
	expect(t, !sc4RPin(t, s, 0, 0).RowAndCell().cell.hyperlink, "!term51RPin(&s, 0, 0).rowAndCell().cell.hyperlink")

	zigTry(t, s.StartHyperlink([]uint8("http://example.com"), None[[]uint8]()))
	sc4Write(t, s, "b世")
	expect(t, sc4RPin(t, s, 1, 0).RowAndCell().cell.hyperlink, "term51RPin(&s, 1, 0).rowAndCell().cell.hyperlink")
	expect(t, sc4RPin(t, s, 2, 0).RowAndCell().cell.hyperlink, "term51RPin(&s, 2, 0).rowAndCell().cell.hyperlink")
	expect(t, sc4RPin(t, s, 3, 0).RowAndCell().cell.hyperlink, "term51RPin(&s, 3, 0).rowAndCell().cell.hyperlink")
	expect(t, sc4RPin(t, s, 3, 0).RowAndCell().row.hyperlink, "term51RPin(&s, 3, 0).rowAndCell().row.hyperlink")
	s.EndHyperlink()
	sc4Write(t, s, "c")
	expect(t, !sc4RPin(t, s, 0, 1).RowAndCell().cell.hyperlink, "!term51RPin(&s, 0, 1).rowAndCell().cell.hyperlink")

	// The spacer head of a wide character that wraps is part of the link.
	// Upstream defect (test helper): the hyperlink is attached to the spacer
	// head before the row's wrap flag is set, so the page integrity check
	// inside cursorSetHyperlink panics in debug builds
	// (IntegrityError.UnwrappedSpacerHead). Pause the checks to pin the
	// resulting state.
	s2 := sc4Init(t, 3, 3, 0)
	defer s2.Deinit()
	zigTry(t, s2.StartHyperlink([]uint8("http://example.com"), None[[]uint8]()))
	s2.cursor.page_pin.node.data.PauseIntegrityChecks(true)
	sc4Write(t, s2, "ab世")
	s2.cursor.page_pin.node.data.PauseIntegrityChecks(false)
	expectEqual(t, PageCellWideSpacerHead, sc4RPin(t, s2, 2, 0).RowAndCell().cell.wide)
	expect(t, sc4RPin(t, s2, 2, 0).RowAndCell().cell.hyperlink, "term51RPin(&s2, 2, 0).rowAndCell().cell.hyperlink")
	expect(t, sc4RPin(t, s2, 0, 1).RowAndCell().cell.hyperlink, "term51RPin(&s2, 0, 1).rowAndCell().cell.hyperlink")
	expect(t, sc4RPin(t, s2, 1, 1).RowAndCell().cell.hyperlink, "term51RPin(&s2, 1, 1).rowAndCell().cell.hyperlink")
}

// Zig: test "term51: testWriteString: semantic content is stamped on cells"
func TestSc4_testWriteString_semantic_content_is_stamped_on_cells(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 4, 4, 0)
	defer s.Deinit()

	s.CursorSetSemanticContent(sc4Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc4Write(t, s, "p")
	s.CursorSetSemanticContent(sc4Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc4Write(t, s, "i世")
	expect(t, sc4RPin(t, s, 0, 0).RowAndCell().cell.semantic_content == PageCellSemanticContentPrompt, "term51RPin(&s, 0, 0).rowAndCell().cell.semantic_content == .prompt")
	expect(t, sc4RPin(t, s, 1, 0).RowAndCell().cell.semantic_content == PageCellSemanticContentInput, "term51RPin(&s, 1, 0).rowAndCell().cell.semantic_content == .input")
	expect(t, sc4RPin(t, s, 2, 0).RowAndCell().cell.semantic_content == PageCellSemanticContentInput, "term51RPin(&s, 2, 0).rowAndCell().cell.semantic_content == .input")
	expect(t, sc4RPin(t, s, 3, 0).RowAndCell().cell.semantic_content == PageCellSemanticContentInput, "term51RPin(&s, 3, 0).rowAndCell().cell.semantic_content == .input")

	// The spacer head carries it as well.
	s2 := sc4Init(t, 3, 3, 0)
	defer s2.Deinit()
	s2.CursorSetSemanticContent(sc4Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc4Write(t, s2, "ab世")
	expect(t, sc4RPin(t, s2, 2, 0).RowAndCell().cell.semantic_content == PageCellSemanticContentInput, "term51RPin(&s2, 2, 0).rowAndCell().cell.semantic_content == .input")
	expectEqual(t, PageCellWideSpacerHead, sc4RPin(t, s2, 2, 0).RowAndCell().cell.wide)
}

// Zig: test "term51: testWriteString: newline semantic prompt handling"
func TestSc4_testWriteString_newline_semantic_prompt_handling(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 4, 8, 0)
	defer s.Deinit()

	// Output: the next row is unmarked.
	s.CursorSetSemanticContent(sc4Output())
	sc4Write(t, s, "a\nb\n")
	expect(t, sc4RPin(t, s, 0, 1).RowAndCell().row.semantic_prompt == PageRowSemanticPromptNone, "term51RPin(&s, 0, 1).rowAndCell().row.semantic_prompt == .none")
	expect(t, sc4RPin(t, s, 0, 2).RowAndCell().row.semantic_prompt == PageRowSemanticPromptNone, "term51RPin(&s, 0, 2).rowAndCell().row.semantic_prompt == .none")
	expect(t, s.cursor.semantic_content == PageCellSemanticContentOutput, "s.cursor.semantic_content == .output")

	// Input without clear-eol: the next row becomes a prompt continuation
	// and the content stays input.
	s.CursorSetSemanticContent(sc4Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc4Write(t, s, "c\n")
	expect(t, sc4RPin(t, s, 0, 3).RowAndCell().row.semantic_prompt == PageRowSemanticPromptPromptContinuation, "term51RPin(&s, 0, 3).rowAndCell().row.semantic_prompt == .prompt_continuation")
	expect(t, s.cursor.semantic_content == PageCellSemanticContentInput, "s.cursor.semantic_content == .input")

	// Prompt: same.
	s.CursorSetSemanticContent(sc4Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc4Write(t, s, "d\n")
	expect(t, sc4RPin(t, s, 0, 4).RowAndCell().row.semantic_prompt == PageRowSemanticPromptPromptContinuation, "term51RPin(&s, 0, 4).rowAndCell().row.semantic_prompt == .prompt_continuation")
	expect(t, s.cursor.semantic_content == PageCellSemanticContentPrompt, "s.cursor.semantic_content == .prompt")

	// Input with clear-eol: the cursor reverts to output and the row is
	// left unmarked.
	s.CursorSetSemanticContent(sc4Input(ScreenCursorSetSemanticContentInputClearEol))
	sc4Write(t, s, "e\n")
	expect(t, sc4RPin(t, s, 0, 5).RowAndCell().row.semantic_prompt == PageRowSemanticPromptNone, "term51RPin(&s, 0, 5).rowAndCell().row.semantic_prompt == .none")
	expect(t, s.cursor.semantic_content == PageCellSemanticContentOutput, "s.cursor.semantic_content == .output")
	expect(t, !s.cursor.semantic_content_clear_eol, "!s.cursor.semantic_content_clear_eol")
}

// Zig: test "term51: testWriteString: soft wrap semantic prompt handling"
func TestSc4_testWriteString_soft_wrap_semantic_prompt_handling(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 2, 8, 0)
	defer s.Deinit()

	s.CursorSetSemanticContent(sc4Output())
	sc4Write(t, s, "abc")
	expect(t, sc4RPin(t, s, 0, 1).RowAndCell().row.semantic_prompt == PageRowSemanticPromptNone, "term51RPin(&s, 0, 1).rowAndCell().row.semantic_prompt == .none")

	s.CursorSetSemanticContent(sc4Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc4Write(t, s, "\nabc")
	expect(t, sc4RPin(t, s, 0, 2).RowAndCell().row.semantic_prompt == PageRowSemanticPromptPromptContinuation, "term51RPin(&s, 0, 2).rowAndCell().row.semantic_prompt == .prompt_continuation")
	expect(t, sc4RPin(t, s, 0, 3).RowAndCell().row.semantic_prompt == PageRowSemanticPromptPromptContinuation, "term51RPin(&s, 0, 3).rowAndCell().row.semantic_prompt == .prompt_continuation")

	s.CursorSetSemanticContent(sc4Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc4Write(t, s, "\nabc")
	expect(t, sc4RPin(t, s, 0, 4).RowAndCell().row.semantic_prompt == PageRowSemanticPromptPromptContinuation, "term51RPin(&s, 0, 4).rowAndCell().row.semantic_prompt == .prompt_continuation")
	expect(t, sc4RPin(t, s, 0, 5).RowAndCell().row.semantic_prompt == PageRowSemanticPromptPromptContinuation, "term51RPin(&s, 0, 5).rowAndCell().row.semantic_prompt == .prompt_continuation")
}

// Zig: test "term51: testWriteString: soft wrap scrolls at the bottom"
func TestSc4_testWriteString_soft_wrap_scrolls_at_the_bottom(t *testing.T) {
	testStdCapacity(t)
	s := sc4Init(t, 3, 2, 0)
	defer s.Deinit()

	sc4Write(t, s, "abcdefghi")
	out, err := s.DumpStringAlloc(sc4Tag(PointTagScreen))
	zigTry(t, err)
	expectEqualStrings(t, "def\nghi", string(out))
	expect(t, sc4RPin(t, s, 0, 0).RowAndCell().row.wrap, "term51RPin(&s, 0, 0).rowAndCell().row.wrap")
	expect(t, sc4RPin(t, s, 0, 1).RowAndCell().row.wrap_continuation, "term51RPin(&s, 0, 1).rowAndCell().row.wrap_continuation")
	expect(t, s.cursor.pending_wrap, "s.cursor.pending_wrap")
}

// Go-only: panic test: Screen.testWriteString: pending wrap with the cursor not in the last column
func TestSc4_panic_testWriteString_pending_wrap_not_in_last_column(t *testing.T) {
	// Zig: assert(self.cursor.x == self.pages.cols - 1) before wrapping.
	// No Deinit: the screen is left in the state the panic stopped it in.
	testStdCapacity(t)
	s := sc4Init(t, 5, 3, 0)
	s.cursor.pending_wrap = true
	sc4ExpectPanic(t, func() { _ = s.TestWriteString([]uint8("a")) })
}

// Go-only: panic test: Screen.testWriteString: zero width codepoint with the cursor in column 0
func TestSc4_panic_testWriteString_zero_width_codepoint_in_column_0(t *testing.T) {
	// Zig: cursorCellLeft(1) asserts self.cursor.x >= n.
	testStdCapacity(t)
	s := sc4Init(t, 5, 3, 0)
	sc4ExpectPanic(t, func() { _ = s.TestWriteString([]uint8("́")) })
}

// Go-only: panic test: Screen.testWriteString: zero width codepoint after a spacer head
func TestSc4_panic_testWriteString_zero_width_codepoint_after_a_spacer_head(t *testing.T) {
	// Zig: `.spacer_head => unreachable` when the cell left of the cursor is a
	// spacer head. Normal writes never put one there, so the cell is set
	// by hand.
	testStdCapacity(t)
	s := sc4Init(t, 5, 3, 0)
	sc4Write(t, s, "a")
	sc4RPin(t, s, 0, 0).RowAndCell().cell.wide = PageCellWideSpacerHead
	sc4ExpectPanic(t, func() { _ = s.TestWriteString([]uint8("́")) })
}
