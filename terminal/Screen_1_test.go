// Tests translated one for one from src/terminal/Screen.zig (Ghostty v1.3.1
// plus the term51 additional tests), first part: the tests from "Screen read
// and write" up to and including "Screen: selectLine semantic all same
// content" (Zig lines 3348 to 8246). Each Go test carries the exact Zig title
// in a "// Zig: test" comment. Helpers are prefixed sc1.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualSlices / expectEqualStrings keep Zig's names
//     and Zig's (expected, actual) argument order. `try` on a call that
//     returns an error is zigTry(t, err); `x.?` is zigUnwrap.
//   - `try init(alloc, .{ ... })` / `Screen.init(alloc, .{ ... })` is
//     sc1Init(t, cols, rows, max_scrollback): ScreenInit with the Zig default
//     kitty_image_storage_limit, the screen returned by pointer (the methods
//     take a pointer receiver).
//   - `try s.testWriteString("...")` is sc1Write; `try s.dumpStringAlloc(alloc,
//     .{ .tag = .{} })` is sc1Dump, which returns a string. The
//     `defer alloc.free(str)` that follows each dumpStringAlloc is dropped
//     there (allocator/leak check).
//   - Zig's `.{ .active = .{ .x = x, .y = y } }` points are sc1Active,
//     sc1Screen, sc1History and sc1Viewport; Zig defaults a Go literal does
//     not give (Resize.reflow and prompt_redraw, SelectLine, SelectionString)
//     are written out through the shared helpers of zig_defaults_test.go.
//   - Every test starts with testStdCapacity(t): Zig's unit tests run with a
//     standard page capacity of grapheme_bytes 512.
//   - `defer s.deinit()` is `defer s.Deinit()`; `testing.allocator` and the
//     `alloc` arguments are dropped.
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt.

package terminal

import (
	"fmt"
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers (prefix sc1).
// ---------------------------------------------------------------------------

// sc1ExpectPanic runs f and fails the test if f does not panic.
func sc1ExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if recover() == nil {
			t.Fatalf("expected a panic, got none")
		}
	}()
	f()
}

// sc1KittyImageStorageLimit is the Zig default of Options.
// kitty_image_storage_limit.
const sc1KittyImageStorageLimit = 320 * 1000 * 1000

// sc1Init is `try Screen.init(alloc, .{ .cols = cols, .rows = rows,
// .max_scrollback = max_scrollback })`, the screen returned by pointer.
func sc1Init(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt, max_scrollback uint64) *Screen {
	t.Helper()
	s, err := ScreenInit(ScreenOptions{
		Cols:                   cols,
		Rows:                   rows,
		MaxScrollback:          max_scrollback,
		KittyImageStorageLimit: sc1KittyImageStorageLimit,
	})
	zigTry(t, err)
	return &s
}

// sc1Write is `try s.testWriteString(text)`.
func sc1Write(t *testing.T, s *Screen, text string) {
	t.Helper()
	zigTry(t, s.TestWriteString([]uint8(text)))
}

// sc1Dump is `try s.dumpStringAlloc(alloc, pt)` (followed by the dropped
// `defer alloc.free(str)`), as a string.
func sc1Dump(t *testing.T, s *Screen, pt Point) string {
	t.Helper()
	str, err := s.DumpStringAlloc(pt)
	zigTry(t, err)
	// Zig-only: defer alloc.free(str)
	return string(str)
}

// sc1Active, sc1Screen, sc1History and sc1Viewport are the Zig literals
// `.{ .active = .{ .x = x, .y = y } }` and so on.
func sc1Active(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagActive, Active: PointCoordinate{X: x, Y: y}}
}

func sc1Screen(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagScreen, Screen: PointCoordinate{X: x, Y: y}}
}

func sc1History(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagHistory, History: PointCoordinate{X: x, Y: y}}
}

func sc1Viewport(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagViewport, Viewport: PointCoordinate{X: x, Y: y}}
}

// sc1Pin is `s.pages.pin(pt).?`.
func sc1Pin(t *testing.T, s *Screen, pt Point) PageListPin {
	t.Helper()
	return zigUnwrap(t, s.Pages.Pin(pt))
}

// sc1Attr is the Zig literal `.{ .<tag> = {} }` of sgr.Attribute (a variant
// without a payload).
func sc1Attr(tag SGRAttributeTag) SGRAttribute {
	return SGRAttribute{Tag: tag}
}

// sc1SetAttr is `try s.setAttribute(.{ .<tag> = {} })`.
func sc1SetAttr(t *testing.T, s *Screen, tag SGRAttributeTag) {
	t.Helper()
	zigTry(t, s.SetAttribute(sc1Attr(tag)))
}

// sc1DirectColorBg is `.{ .direct_color_bg = .{ .r = r } }`.
func sc1DirectColorBg(r uint8) SGRAttribute {
	return SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: r}}
}

// sc1Select is `try s.select(sel)` with a non-null selection.
func sc1Select(t *testing.T, s *Screen, sel Selection) {
	t.Helper()
	zigTry(t, s.Select(Some(sel)))
}

// sc1Clone is `try s.clone(alloc, top, bot)`, the clone returned by pointer.
func sc1Clone(t *testing.T, s *Screen, top Point, bot Optional[Point]) *Screen {
	t.Helper()
	c, err := s.Clone(top, bot)
	zigTry(t, err)
	return &c
}

// sc1NoReflow is `.{ .cols = cols, .rows = rows, .reflow = false }` for
// resize: the Zig default prompt_redraw (.false) is written out.
func sc1NoReflow(cols SizeCellCountInt, rows SizeCellCountInt) ScreenResize {
	r := screenResizeDefault(cols, rows)
	r.Reflow = false
	return r
}

// sc1CursorCell is the repeated
// `s.pages.getCell(.{ .active = .{ .x = s.cursor.x, .y = s.cursor.y } }).?`.
func sc1CursorCell(t *testing.T, s *Screen) PageListCell {
	t.Helper()
	return zigUnwrap(t, s.Pages.GetCell(sc1Active(s.Cursor.X, uint32(s.Cursor.Y))))
}

// sc1PointFromPin is `s.pages.pointFromPin(tag, pin).?`.
func sc1PointFromPin(t *testing.T, s *Screen, tag PointTag, pin PageListPin) Point {
	t.Helper()
	return zigUnwrap(t, s.Pages.PointFromPin(tag, pin))
}

// sc1SemOutput, sc1SemPrompt and sc1SemInput are the Zig literals `.output`,
// `.{ .prompt = kind }` and `.{ .input = kind }` of cursorSetSemanticContent.
func sc1SemOutput() ScreenCursorSetSemanticContent {
	return ScreenCursorSetSemanticContent{Tag: ScreenCursorSetSemanticContentTagOutput}
}

func sc1SemPrompt(kind OSCParsersSemanticPromptPromptKind) ScreenCursorSetSemanticContent {
	return ScreenCursorSetSemanticContent{Tag: ScreenCursorSetSemanticContentTagPrompt, Prompt: kind}
}

func sc1SemInput(kind ScreenCursorSetSemanticContentInput) ScreenCursorSetSemanticContent {
	return ScreenCursorSetSemanticContent{Tag: ScreenCursorSetSemanticContentTagInput, Input: kind}
}

// sc1SelectionString is `try s.selectionString(alloc, opts)` (followed by the
// dropped `defer alloc.free(contents)`), as a string.
func sc1SelectionString(t *testing.T, s *Screen, opts ScreenSelectionString) string {
	t.Helper()
	str, err := s.SelectionString(opts)
	zigTry(t, err)
	// Zig-only: defer alloc.free(contents)
	return string(str)
}

// ---------------------------------------------------------------------------
// Tests.
// ---------------------------------------------------------------------------

// Zig: test "Screen read and write"
func TestSc1_Screen_read_and_write(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 80, 24, 1000)
	defer s.Deinit()
	expectEqual[StyleID](t, 0, s.Cursor.StyleID)

	sc1Write(t, s, "hello, world")
	str := sc1Dump(t, s, sc1Screen(0, 0))
	expectEqualStrings(t, "hello, world", str)
}

// Zig: test "Screen read and write newline"
func TestSc1_Screen_read_and_write_newline(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 80, 24, 1000)
	defer s.Deinit()
	expectEqual[StyleID](t, 0, s.Cursor.StyleID)

	sc1Write(t, s, "hello\nworld")
	str := sc1Dump(t, s, sc1Screen(0, 0))
	expectEqualStrings(t, "hello\nworld", str)
}

// Zig: test "Screen read and write scrollback"
func TestSc1_Screen_read_and_write_scrollback(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 80, 2, 1000)
	defer s.Deinit()

	sc1Write(t, s, "hello\nworld\ntest")
	{
		str := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "hello\nworld\ntest", str)
	}
	{
		str := sc1Dump(t, s, sc1Active(0, 0))
		expectEqualStrings(t, "world\ntest", str)
	}
}

// Zig: test "Screen read and write no scrollback small"
func TestSc1_Screen_read_and_write_no_scrollback_small(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 80, 2, 0)
	defer s.Deinit()

	sc1Write(t, s, "hello\nworld\ntest")
	{
		str := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "world\ntest", str)
	}
	{
		str := sc1Dump(t, s, sc1Active(0, 0))
		expectEqualStrings(t, "world\ntest", str)
	}
}

// Zig: test "Screen read and write no scrollback large"
func TestSc1_Screen_read_and_write_no_scrollback_large(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 80, 2, 0)
	defer s.Deinit()

	for i := 0; i < 1_000; i++ {
		str := fmt.Sprintf("%d\n", i)
		sc1Write(t, s, str)
	}
	sc1Write(t, s, "1000")

	{
		str := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "999\n1000", str)
	}
}

// Zig: test "Screen cursorCopy x/y"
func TestSc1_Screen_cursorCopy_x_y(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 10, 0)
	defer s.Deinit()
	s.CursorAbsolute(2, 3)
	expect(t, s.Cursor.X == 2, "s.cursor.x == 2")
	expect(t, s.Cursor.Y == 3, "s.cursor.y == 3")

	s2 := sc1Init(t, 10, 10, 0)
	defer s2.Deinit()
	zigTry(t, s2.CursorCopy(s.Cursor, screenCursorCopyOptionsDefault()))
	expect(t, s2.Cursor.X == 2, "s2.cursor.x == 2")
	expect(t, s2.Cursor.Y == 3, "s2.cursor.y == 3")
	sc1Write(t, s2, "Hello")

	{
		str := sc1Dump(t, s2, sc1Screen(0, 0))
		expectEqualStrings(t, "\n\n\n  Hello", str)
	}
}

// Zig: test "Screen cursorCopy style deref"
func TestSc1_Screen_cursorCopy_style_deref(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 10, 0)
	defer s.Deinit()

	s2 := sc1Init(t, 10, 10, 0)
	defer s2.Deinit()
	page := &s2.Cursor.PagePin.Node.data

	// Bold should create our style
	sc1SetAttr(t, s2, SGRAttributeTagBold)
	expectEqual[uint](t, 1, page.styles.Count())
	expect(t, s2.Cursor.Style.Flags.Bold, "s2.cursor.style.flags.bold")

	// Copy default style, should release our style
	zigTry(t, s2.CursorCopy(s.Cursor, screenCursorCopyOptionsDefault()))
	expect(t, !s2.Cursor.Style.Flags.Bold, "!s2.cursor.style.flags.bold")
	expectEqual[uint](t, 0, page.styles.Count())
}

// Zig: test "Screen cursorCopy style deref new page"
func TestSc1_Screen_cursorCopy_style_deref_new_page(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 10, 0)
	defer s.Deinit()

	s2 := sc1Init(t, 10, 10, 2048)
	defer s2.Deinit()

	// We need to get the cursor on a new page.
	first_page_size := s2.Pages.pages.first.data.capacity.rows

	// Fill the scrollback with blank lines until
	// there are only 5 rows left on the first page.
	s2.Pages.pages.first.data.PauseIntegrityChecks(true)
	for range first_page_size - 5 {
		sc1Write(t, s2, "\n")
	}
	s2.Pages.pages.first.data.PauseIntegrityChecks(false)

	sc1Write(t, s2, "1\n2\n3\n4\n5\n6\n7\n8\n9\n10")

	// s2.pages.diagram(...):
	//
	//      +----------+ = PAGE 0
	//  ... :          :
	//     +-------------+ ACTIVE
	// 4300 |1         | | 0
	// 4301 |2         | | 1
	// 4302 |3         | | 2
	// 4303 |4         | | 3
	// 4304 |5         | | 4
	//      +----------+ :
	//      +----------+ : = PAGE 1
	//    0 |6         | | 5
	//    1 |7         | | 6
	//    2 |8         | | 7
	//    3 |9         | | 8
	//    4 |10        | | 9
	//      :  ^       : : = PIN 0
	//      +----------+ :
	//     +-------------+

	// This should be PAGE 1
	page := &s2.Cursor.PagePin.Node.data

	// It should be the last page in the list.
	expect(t, &s2.Pages.pages.last.data == page, "&s2.pages.pages.last.?.data == page")
	// It should have a previous page.
	expect(t, s2.Cursor.PagePin.Node.prev != nil, "s2.cursor.page_pin.node.prev != null")

	// The cursor should be at 2, 9
	expect(t, s2.Cursor.X == 2, "s2.cursor.x == 2")
	expect(t, s2.Cursor.Y == 9, "s2.cursor.y == 9")

	// Bold should create our style in page 1.
	sc1SetAttr(t, s2, SGRAttributeTagBold)
	expectEqual[uint](t, 1, page.styles.Count())
	expect(t, s2.Cursor.Style.Flags.Bold, "s2.cursor.style.flags.bold")

	// Copy the cursor for the first screen. This should release
	// the style from page 1 and move the cursor back to page 0.
	zigTry(t, s2.CursorCopy(s.Cursor, screenCursorCopyOptionsDefault()))
	expect(t, !s2.Cursor.Style.Flags.Bold, "!s2.cursor.style.flags.bold")
	expectEqual[uint](t, 0, page.styles.Count())
	// The page after the page the cursor is now in should be page 1.
	expect(t, page == &s2.Cursor.PagePin.Node.next.data, "page == &s2.cursor.page_pin.node.next.?.data")
	// The cursor should be at 0, 0
	expect(t, s2.Cursor.X == 0, "s2.cursor.x == 0")
	expect(t, s2.Cursor.Y == 0, "s2.cursor.y == 0")
}

// Zig: test "Screen cursorCopy style copy"
func TestSc1_Screen_cursorCopy_style_copy(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 10, 0)
	defer s.Deinit()
	sc1SetAttr(t, s, SGRAttributeTagBold)

	s2 := sc1Init(t, 10, 10, 0)
	defer s2.Deinit()
	page := &s2.Cursor.PagePin.Node.data
	zigTry(t, s2.CursorCopy(s.Cursor, screenCursorCopyOptionsDefault()))
	expect(t, s2.Cursor.Style.Flags.Bold, "s2.cursor.style.flags.bold")
	expectEqual[uint](t, 1, page.styles.Count())
}

// Zig: test "Screen cursorCopy hyperlink deref"
func TestSc1_Screen_cursorCopy_hyperlink_deref(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 10, 0)
	defer s.Deinit()

	s2 := sc1Init(t, 10, 10, 0)
	defer s2.Deinit()
	page := &s2.Cursor.PagePin.Node.data

	// Create a hyperlink for the cursor.
	zigTry(t, s2.StartHyperlink([]uint8("https://example.com/"), None[[]uint8]()))
	expectEqual[uint](t, 1, page.hyperlink_set.Count())
	expect(t, s2.Cursor.HyperlinkID != 0, "s2.cursor.hyperlink_id != 0")

	// Copy a cursor with no hyperlink, should release our hyperlink.
	zigTry(t, s2.CursorCopy(s.Cursor, screenCursorCopyOptionsDefault()))
	expectEqual[uint](t, 0, page.hyperlink_set.Count())
	expect(t, s2.Cursor.HyperlinkID == 0, "s2.cursor.hyperlink_id == 0")
}

// Zig: test "Screen cursorCopy hyperlink deref new page"
func TestSc1_Screen_cursorCopy_hyperlink_deref_new_page(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 10, 0)
	defer s.Deinit()

	s2 := sc1Init(t, 10, 10, 2048)
	defer s2.Deinit()

	// We need to get the cursor on a new page.
	first_page_size := s2.Pages.pages.first.data.capacity.rows

	// Fill the scrollback with blank lines until
	// there are only 5 rows left on the first page.
	s2.Pages.pages.first.data.PauseIntegrityChecks(true)
	for range first_page_size - 5 {
		sc1Write(t, s2, "\n")
	}
	s2.Pages.pages.first.data.PauseIntegrityChecks(false)

	sc1Write(t, s2, "1\n2\n3\n4\n5\n6\n7\n8\n9\n10")

	// s2.pages.diagram(...):
	//
	//      +----------+ = PAGE 0
	//  ... :          :
	//     +-------------+ ACTIVE
	// 4300 |1         | | 0
	// 4301 |2         | | 1
	// 4302 |3         | | 2
	// 4303 |4         | | 3
	// 4304 |5         | | 4
	//      +----------+ :
	//      +----------+ : = PAGE 1
	//    0 |6         | | 5
	//    1 |7         | | 6
	//    2 |8         | | 7
	//    3 |9         | | 8
	//    4 |10        | | 9
	//      :  ^       : : = PIN 0
	//      +----------+ :
	//     +-------------+

	// This should be PAGE 1
	page := &s2.Cursor.PagePin.Node.data

	// It should be the last page in the list.
	expect(t, &s2.Pages.pages.last.data == page, "&s2.pages.pages.last.?.data == page")
	// It should have a previous page.
	expect(t, s2.Cursor.PagePin.Node.prev != nil, "s2.cursor.page_pin.node.prev != null")

	// The cursor should be at 2, 9
	expect(t, s2.Cursor.X == 2, "s2.cursor.x == 2")
	expect(t, s2.Cursor.Y == 9, "s2.cursor.y == 9")

	// Create a hyperlink for the cursor, should be in page 1.
	zigTry(t, s2.StartHyperlink([]uint8("https://example.com/"), None[[]uint8]()))
	expectEqual[uint](t, 1, page.hyperlink_set.Count())
	expect(t, s2.Cursor.HyperlinkID != 0, "s2.cursor.hyperlink_id != 0")

	// Copy the cursor for the first screen. This should release
	// the hyperlink from page 1 and move the cursor back to page 0.
	zigTry(t, s2.CursorCopy(s.Cursor, screenCursorCopyOptionsDefault()))
	expectEqual[uint](t, 0, page.hyperlink_set.Count())
	expect(t, s2.Cursor.HyperlinkID == 0, "s2.cursor.hyperlink_id == 0")
	// The page after the page the cursor is now in should be page 1.
	expect(t, page == &s2.Cursor.PagePin.Node.next.data, "page == &s2.cursor.page_pin.node.next.?.data")
	// The cursor should be at 0, 0
	expect(t, s2.Cursor.X == 0, "s2.cursor.x == 0")
	expect(t, s2.Cursor.Y == 0, "s2.cursor.y == 0")
}

// Zig: test "Screen cursorCopy hyperlink copy"
func TestSc1_Screen_cursorCopy_hyperlink_copy(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 10, 0)
	defer s.Deinit()

	// Create a hyperlink for the cursor.
	zigTry(t, s.StartHyperlink([]uint8("https://example.com/"), None[[]uint8]()))
	expectEqual[uint](t, 1, s.Cursor.PagePin.Node.data.hyperlink_set.Count())
	expect(t, s.Cursor.HyperlinkID != 0, "s.cursor.hyperlink_id != 0")

	s2 := sc1Init(t, 10, 10, 0)
	defer s2.Deinit()
	page := &s2.Cursor.PagePin.Node.data

	expectEqual[uint](t, 0, page.hyperlink_set.Count())
	expect(t, s2.Cursor.HyperlinkID == 0, "s2.cursor.hyperlink_id == 0")

	// Copy the cursor with the hyperlink.
	zigTry(t, s2.CursorCopy(s.Cursor, screenCursorCopyOptionsDefault()))
	expectEqual[uint](t, 1, page.hyperlink_set.Count())
	expect(t, s2.Cursor.HyperlinkID != 0, "s2.cursor.hyperlink_id != 0")
}

// Zig: test "Screen cursorCopy hyperlink copy disabled"
func TestSc1_Screen_cursorCopy_hyperlink_copy_disabled(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 10, 0)
	defer s.Deinit()

	// Create a hyperlink for the cursor.
	zigTry(t, s.StartHyperlink([]uint8("https://example.com/"), None[[]uint8]()))
	expectEqual[uint](t, 1, s.Cursor.PagePin.Node.data.hyperlink_set.Count())
	expect(t, s.Cursor.HyperlinkID != 0, "s.cursor.hyperlink_id != 0")

	s2 := sc1Init(t, 10, 10, 0)
	defer s2.Deinit()
	page := &s2.Cursor.PagePin.Node.data

	expectEqual[uint](t, 0, page.hyperlink_set.Count())
	expect(t, s2.Cursor.HyperlinkID == 0, "s2.cursor.hyperlink_id == 0")

	// Copy the cursor with the hyperlink.
	zigTry(t, s2.CursorCopy(s.Cursor, ScreenCursorCopyOptions{Hyperlink: false}))
	expectEqual[uint](t, 0, page.hyperlink_set.Count())
	expect(t, s2.Cursor.HyperlinkID == 0, "s2.cursor.hyperlink_id == 0")
}

// Zig: test "Screen style basics"
func TestSc1_Screen_style_basics(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 80, 24, 1000)
	defer s.Deinit()
	page := &s.Cursor.PagePin.Node.data
	expectEqual[uint](t, 0, page.styles.Count())

	// Set a new style
	sc1SetAttr(t, s, SGRAttributeTagBold)
	expect(t, s.Cursor.StyleID != 0, "s.cursor.style_id != 0")
	expectEqual[uint](t, 1, page.styles.Count())
	expect(t, s.Cursor.Style.Flags.Bold, "s.cursor.style.flags.bold")

	// Set another style, we should still only have one since it was unused
	sc1SetAttr(t, s, SGRAttributeTagItalic)
	expect(t, s.Cursor.StyleID != 0, "s.cursor.style_id != 0")
	expectEqual[uint](t, 1, page.styles.Count())
	expect(t, s.Cursor.Style.Flags.Italic, "s.cursor.style.flags.italic")
}

// Zig: test "Screen style reset to default"
func TestSc1_Screen_style_reset_to_default(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 80, 24, 1000)
	defer s.Deinit()
	page := &s.Cursor.PagePin.Node.data
	expectEqual[uint](t, 0, page.styles.Count())

	// Set a new style
	sc1SetAttr(t, s, SGRAttributeTagBold)
	expect(t, s.Cursor.StyleID != 0, "s.cursor.style_id != 0")
	expectEqual[uint](t, 1, page.styles.Count())

	// Reset to default
	sc1SetAttr(t, s, SGRAttributeTagResetBold)
	expect(t, s.Cursor.StyleID == 0, "s.cursor.style_id == 0")
	expectEqual[uint](t, 0, page.styles.Count())
}

// Zig: test "Screen style reset with unset"
func TestSc1_Screen_style_reset_with_unset(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 80, 24, 1000)
	defer s.Deinit()
	page := &s.Cursor.PagePin.Node.data
	expectEqual[uint](t, 0, page.styles.Count())

	// Set a new style
	sc1SetAttr(t, s, SGRAttributeTagBold)
	expect(t, s.Cursor.StyleID != 0, "s.cursor.style_id != 0")
	expectEqual[uint](t, 1, page.styles.Count())

	// Reset to default
	sc1SetAttr(t, s, SGRAttributeTagUnset)
	expect(t, s.Cursor.StyleID == 0, "s.cursor.style_id == 0")
	expectEqual[uint](t, 0, page.styles.Count())
}

// Zig: test "Screen clearRows active one line"
func TestSc1_Screen_clearRows_active_one_line(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 80, 24, 1000)
	defer s.Deinit()

	sc1Write(t, s, "hello, world")
	s.ClearRows(sc1Active(0, 0), None[Point](), false)
	expect(t, s.Pages.IsDirty(sc1Active(0, 0)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	str := sc1Dump(t, s, sc1Screen(0, 0))
	expectEqualStrings(t, "", str)
}

// Zig: test "Screen clearRows active multi line"
func TestSc1_Screen_clearRows_active_multi_line(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 80, 24, 1000)
	defer s.Deinit()

	sc1Write(t, s, "hello\nworld")
	s.ClearRows(sc1Active(0, 0), None[Point](), false)
	expect(t, s.Pages.IsDirty(sc1Active(0, 0)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, s.Pages.IsDirty(sc1Active(0, 1)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	str := sc1Dump(t, s, sc1Screen(0, 0))
	expectEqualStrings(t, "", str)
}

// Zig: test "Screen clearRows active styled line"
func TestSc1_Screen_clearRows_active_styled_line(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 80, 24, 1000)
	defer s.Deinit()

	sc1SetAttr(t, s, SGRAttributeTagBold)
	sc1Write(t, s, "hello world")
	sc1SetAttr(t, s, SGRAttributeTagUnset)

	// We should have one style
	page := &s.Cursor.PagePin.Node.data
	expectEqual[uint](t, 1, page.styles.Count())

	s.ClearRows(sc1Active(0, 0), None[Point](), false)

	// We should have none because active cleared it
	expectEqual[uint](t, 0, page.styles.Count())

	str := sc1Dump(t, s, sc1Screen(0, 0))
	expectEqualStrings(t, "", str)
}

// Zig: test "Screen clearRows protected"
func TestSc1_Screen_clearRows_protected(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 80, 24, 1000)
	defer s.Deinit()

	sc1Write(t, s, "UNPROTECTED")
	s.Cursor.Protected = true
	sc1Write(t, s, "PROTECTED")
	s.Cursor.Protected = false
	sc1Write(t, s, "UNPROTECTED")
	sc1Write(t, s, "\n")
	s.Cursor.Protected = true
	sc1Write(t, s, "PROTECTED")
	s.Cursor.Protected = false
	sc1Write(t, s, "UNPROTECTED")
	s.Cursor.Protected = true
	sc1Write(t, s, "PROTECTED")
	s.Cursor.Protected = false

	s.ClearRows(sc1Active(0, 0), None[Point](), true)

	str := sc1Dump(t, s, sc1Screen(0, 0))
	expectEqualStrings(t, "           PROTECTED\nPROTECTED           PROTECTED", str)
}

// Zig: test "Screen eraseRows history"
func TestSc1_Screen_eraseRows_history(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 5, 1000)
	defer s.Deinit()

	sc1Write(t, s, "1\n2\n3\n4\n5\n6")

	{
		str := sc1Dump(t, s, sc1Active(0, 0))
		expectEqualStrings(t, "2\n3\n4\n5\n6", str)
	}
	{
		str := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "1\n2\n3\n4\n5\n6", str)
	}

	s.EraseRows(sc1History(0, 0), None[Point]())

	{
		str := sc1Dump(t, s, sc1Active(0, 0))
		expectEqualStrings(t, "2\n3\n4\n5\n6", str)
	}
	{
		str := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "2\n3\n4\n5\n6", str)
	}
}

// Zig: test "Screen eraseRows history with more lines"
func TestSc1_Screen_eraseRows_history_with_more_lines(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 5, 1000)
	defer s.Deinit()

	sc1Write(t, s, "A\nB\nC\n1\n2\n3\n4\n5\n6")

	{
		str := sc1Dump(t, s, sc1Active(0, 0))
		expectEqualStrings(t, "2\n3\n4\n5\n6", str)
	}
	{
		str := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "A\nB\nC\n1\n2\n3\n4\n5\n6", str)
	}

	s.EraseRows(sc1History(0, 0), None[Point]())

	{
		str := sc1Dump(t, s, sc1Active(0, 0))
		expectEqualStrings(t, "2\n3\n4\n5\n6", str)
	}
	{
		str := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "2\n3\n4\n5\n6", str)
	}
}

// Zig: test "Screen eraseRows active partial"
func TestSc1_Screen_eraseRows_active_partial(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 5, 0)
	defer s.Deinit()

	sc1Write(t, s, "1\n2\n3")

	{
		str := sc1Dump(t, s, sc1Active(0, 0))
		expectEqualStrings(t, "1\n2\n3", str)
	}

	s.EraseRows(sc1Active(0, 0), Some(sc1Active(0, 1)))

	{
		str := sc1Dump(t, s, sc1Active(0, 0))
		expectEqualStrings(t, "3", str)
	}
	{
		str := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "3", str)
	}
}

// Zig: test "Screen: cursorDown across pages preserves style"
func TestSc1_Screen_cursorDown_across_pages_preserves_style(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 1)
	defer s.Deinit()

	// Scroll down enough to go to another page
	start_page := &s.Pages.pages.last.data
	rem := start_page.capacity.rows
	start_page.PauseIntegrityChecks(true)
	for range rem {
		zigTry(t, s.cursorDownOrScroll())
	}
	start_page.PauseIntegrityChecks(false)

	// We need our page to change for this test o make sense. If this
	// assertion fails then the bug is in the test: we should be scrolling
	// above enough for a new page to show up.
	{
		page := &s.Cursor.PagePin.Node.data
		expect(t, start_page != page, "start_page != page")
	}

	// Scroll back to the previous page
	s.CursorUp(1)
	{
		page := &s.Cursor.PagePin.Node.data
		expect(t, start_page == page, "start_page == page")
	}

	// Go back up, set a style
	sc1SetAttr(t, s, SGRAttributeTagBold)
	{
		page := &s.Cursor.PagePin.Node.data
		styleval := page.styles.Get(s.Cursor.StyleID)
		expect(t, styleval.Flags.Bold, "styleval.flags.bold")
	}

	// Go back down into the next page and we should have that style
	s.CursorDown(1)
	{
		page := &s.Cursor.PagePin.Node.data
		styleval := page.styles.Get(s.Cursor.StyleID)
		expect(t, styleval.Flags.Bold, "styleval.flags.bold")
	}
}

// Zig: test "Screen: cursorUp across pages preserves style"
func TestSc1_Screen_cursorUp_across_pages_preserves_style(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 1)
	defer s.Deinit()

	// Scroll down enough to go to another page
	start_page := &s.Pages.pages.last.data
	rem := start_page.capacity.rows
	start_page.PauseIntegrityChecks(true)
	for range rem {
		zigTry(t, s.cursorDownOrScroll())
	}
	start_page.PauseIntegrityChecks(false)

	// We need our page to change for this test o make sense. If this
	// assertion fails then the bug is in the test: we should be scrolling
	// above enough for a new page to show up.
	{
		page := &s.Cursor.PagePin.Node.data
		expect(t, start_page != page, "start_page != page")
	}

	// Go back up, set a style
	sc1SetAttr(t, s, SGRAttributeTagBold)
	{
		page := &s.Cursor.PagePin.Node.data
		styleval := page.styles.Get(s.Cursor.StyleID)
		expect(t, styleval.Flags.Bold, "styleval.flags.bold")
	}

	// Go back down into the prev page and we should have that style
	s.CursorUp(1)
	{
		page := &s.Cursor.PagePin.Node.data
		expect(t, start_page == page, "start_page == page")

		styleval := page.styles.Get(s.Cursor.StyleID)
		expect(t, styleval.Flags.Bold, "styleval.flags.bold")
	}
}

// Zig: test "Screen: cursorAbsolute across pages preserves style"
func TestSc1_Screen_cursorAbsolute_across_pages_preserves_style(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 1)
	defer s.Deinit()

	// Scroll down enough to go to another page
	start_page := &s.Pages.pages.last.data
	rem := start_page.capacity.rows
	start_page.PauseIntegrityChecks(true)
	for range rem {
		zigTry(t, s.cursorDownOrScroll())
	}
	start_page.PauseIntegrityChecks(false)

	// We need our page to change for this test o make sense. If this
	// assertion fails then the bug is in the test: we should be scrolling
	// above enough for a new page to show up.
	{
		page := &s.Cursor.PagePin.Node.data
		expect(t, start_page != page, "start_page != page")
	}

	// Go back up, set a style
	sc1SetAttr(t, s, SGRAttributeTagBold)
	{
		page := &s.Cursor.PagePin.Node.data
		styleval := page.styles.Get(s.Cursor.StyleID)
		expect(t, styleval.Flags.Bold, "styleval.flags.bold")
	}

	// Go back down into the prev page and we should have that style
	s.CursorAbsolute(1, 1)
	{
		page := &s.Cursor.PagePin.Node.data
		expect(t, start_page == page, "start_page == page")

		styleval := page.styles.Get(s.Cursor.StyleID)
		expect(t, styleval.Flags.Bold, "styleval.flags.bold")
	}
}

// Zig: test "Screen: cursorAbsolute to page with insufficient capacity"
func TestSc1_Screen_cursorAbsolute_to_page_with_insufficient_capacity(t *testing.T) {
	// This test checks for a very specific edge case
	// which previously resulted in memory corruption.
	//
	// The conditions for this edge case are as such:
	// - The cursor has an associated style or other managed memory.
	// - The cursor moves to a different page.
	// - The new page is at capacity and must have its capacity adjusted.
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 1)
	defer s.Deinit()

	// Scroll down enough to go to another page
	start_page := &s.Pages.pages.last.data
	rem := start_page.capacity.rows
	start_page.PauseIntegrityChecks(true)
	for range rem {
		zigTry(t, s.cursorDownOrScroll())
	}
	start_page.PauseIntegrityChecks(false)

	new_page := &s.Cursor.PagePin.Node.data

	// We need our page to change for this test to make sense. If this
	// assertion fails then the bug is in the test: we should be scrolling
	// above enough for a new page to show up.
	expect(t, start_page != new_page, "start_page != new_page")

	// Add styles to the start page until it reaches capacity.
	{
		// Pause integrity checks because they're slow and
		// we're not testing this, this is just setup.
		start_page.PauseIntegrityChecks(true)
		defer start_page.PauseIntegrityChecks(false)
		defer start_page.AssertIntegrity()

		for n := uint32(1); ; n++ {
			_, err := start_page.styles.Add(Style{BgColor: StyleColor{
				Tag: StyleColorRgb,
				Rgb: ColorRGB{R: uint8(n), G: uint8(n >> 8), B: uint8(n >> 16)},
			}})
			if err != nil {
				break
			}
		}
	}

	// Set a style on the cursor.
	sc1SetAttr(t, s, SGRAttributeTagBold)
	{
		styleval := new_page.styles.Get(s.Cursor.StyleID)
		expect(t, styleval.Flags.Bold, "styleval.flags.bold")
	}

	// Go back up into the start page and we should still have that style.
	s.CursorAbsolute(1, 1)
	{
		cur_page := &s.Cursor.PagePin.Node.data
		// The page we're on now should NOT equal start_page, since its
		// capacity should have been adjusted, which invalidates our ptr.
		expect(t, start_page != cur_page, "start_page != cur_page")
		// To make sure we DID change pages we check we're not on new_page.
		expect(t, new_page != cur_page, "new_page != cur_page")

		styleval := cur_page.styles.Get(s.Cursor.StyleID)
		expect(t, styleval.Flags.Bold, "styleval.flags.bold")
	}

	s.Cursor.PagePin.Node.data.AssertIntegrity()
	new_page.AssertIntegrity()
}

// Zig: test "Screen: scrolling"
func TestSc1_Screen_scrolling(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 0)
	defer s.Deinit()
	zigTry(t, s.SetAttribute(sc1DirectColorBg(155)))
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL")

	// Scroll down, should still be bottom
	zigTry(t, s.CursorDownScroll())
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "2EFGH\n3IJKL", contents)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Active(0, 2)))
		cell := list_cell.Cell
		expect(t, cell.content_tag == PageCellContentTagBgColorRgb, "cell.content_tag == .bg_color_rgb")
		expectEqual(t, PageCellRGB{R: 155, G: 0, B: 0}, cell.content.color_rgb)
	}

	// Everything is dirty because we have no scrollback
	expect(t, s.Pages.IsDirty(sc1Active(0, 0)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, s.Pages.IsDirty(sc1Active(0, 1)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, s.Pages.IsDirty(sc1Active(0, 2)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")

	// Scrolling to the bottom does nothing
	s.Scroll(ScreenScroll{Tag: ScreenScrollActive})

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "2EFGH\n3IJKL", contents)
	}
}

// Zig: test "Screen: scrolling with a single-row screen no scrollback"
func TestSc1_Screen_scrolling_with_a_single_row_screen_no_scrollback(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 1, 0)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD")

	// Scroll down, should still be bottom
	zigTry(t, s.CursorDownScroll())
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "", contents)
	}

	// Screen should be dirty
	expect(t, s.Pages.IsDirty(sc1Active(0, 0)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
}

// Zig: test "Screen: scrolling with a single-row screen with scrollback"
func TestSc1_Screen_scrolling_with_a_single_row_screen_with_scrollback(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 1, 1)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD")

	// Scroll down, should still be bottom
	zigTry(t, s.CursorDownScroll())
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "", contents)
	}

	// Active should be dirty
	expect(t, s.Pages.IsDirty(sc1Active(0, 0)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	// Scrollback also dirty because cursor moved from there
	expect(t, s.Pages.IsDirty(sc1Screen(0, 0)), "s.pages.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")

	s.Scroll(ScreenScroll{Tag: ScreenScrollDeltaRow, DeltaRow: -1})
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "1ABCD", contents)
	}
}

// Zig: test "Screen: scrolling across pages preserves style"
func TestSc1_Screen_scrolling_across_pages_preserves_style(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 1)
	defer s.Deinit()
	sc1SetAttr(t, s, SGRAttributeTagBold)
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL")
	start_page := &s.Pages.pages.last.data

	// Scroll down enough to go to another page
	rem := start_page.capacity.rows - start_page.size.rows + 1
	start_page.PauseIntegrityChecks(true)
	for range rem {
		zigTry(t, s.cursorDownOrScroll())
	}
	start_page.PauseIntegrityChecks(false)

	// We need our page to change for this test o make sense. If this
	// assertion fails then the bug is in the test: we should be scrolling
	// above enough for a new page to show up.
	page := &s.Pages.pages.last.data
	expect(t, start_page != page, "start_page != page")

	styleval := page.styles.Get(s.Cursor.StyleID)
	expect(t, styleval.Flags.Bold, "styleval.flags.bold")
}

// Zig: test "Screen: scroll down from 0"
func TestSc1_Screen_scroll_down_from_0(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 0)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL")

	// Scrolling up does nothing, but allows it
	s.Scroll(ScreenScroll{Tag: ScreenScrollDeltaRow, DeltaRow: -1})
	expect(t, s.Pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "1ABCD\n2EFGH\n3IJKL", contents)
	}
}

// Zig: test "Screen: scrollback various cases"
func TestSc1_Screen_scrollback_various_cases(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 1)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL")
	zigTry(t, s.CursorDownScroll())

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "2EFGH\n3IJKL", contents)
	}

	// Scrolling to the bottom
	s.Scroll(ScreenScroll{Tag: ScreenScrollActive})
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "2EFGH\n3IJKL", contents)
	}

	// Scrolling back should make it visible again
	s.Scroll(ScreenScroll{Tag: ScreenScrollDeltaRow, DeltaRow: -1})
	expect(t, s.Pages.viewport.Tag != PageListViewportActive, "s.pages.viewport != .active")
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "1ABCD\n2EFGH\n3IJKL", contents)
	}

	// Scrolling back again should do nothing
	s.Scroll(ScreenScroll{Tag: ScreenScrollDeltaRow, DeltaRow: -1})
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "1ABCD\n2EFGH\n3IJKL", contents)
	}

	// Scrolling to the bottom
	s.Scroll(ScreenScroll{Tag: ScreenScrollActive})
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "2EFGH\n3IJKL", contents)
	}

	// Scrolling forward with no grow should do nothing
	s.Scroll(ScreenScroll{Tag: ScreenScrollDeltaRow, DeltaRow: 1})
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "2EFGH\n3IJKL", contents)
	}

	// Scrolling to the top should work
	s.Scroll(ScreenScroll{Tag: ScreenScrollTop})
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "1ABCD\n2EFGH\n3IJKL", contents)
	}

	// Should be able to easily clear active area only
	s.ClearRows(sc1Active(0, 0), None[Point](), false)
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "1ABCD", contents)
	}

	// Scrolling to the bottom
	s.Scroll(ScreenScroll{Tag: ScreenScrollActive})
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "", contents)
	}
}

// Zig: test "Screen: scrollback with multi-row delta"
func TestSc1_Screen_scrollback_with_multi_row_delta(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 3)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL\n4ABCD\n5EFGH\n6IJKL")

	// Scroll to top
	s.Scroll(ScreenScroll{Tag: ScreenScrollTop})
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "1ABCD\n2EFGH\n3IJKL", contents)
	}

	// Scroll down multiple
	s.Scroll(ScreenScroll{Tag: ScreenScrollDeltaRow, DeltaRow: 5})
	expect(t, s.Pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "4ABCD\n5EFGH\n6IJKL", contents)
	}
}

// Zig: test "Screen: scrollback empty"
func TestSc1_Screen_scrollback_empty(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 50)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL")
	s.Scroll(ScreenScroll{Tag: ScreenScrollDeltaRow, DeltaRow: 1})
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "1ABCD\n2EFGH\n3IJKL", contents)
	}
}

// Zig: test "Screen: scrollback doesn't move viewport if not at bottom"
func TestSc1_Screen_scrollback_doesn_t_move_viewport_if_not_at_bottom(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 3)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL\n4ABCD\n5EFGH")

	// First test: we scroll up by 1, so we're not at the bottom anymore.
	s.Scroll(ScreenScroll{Tag: ScreenScrollDeltaRow, DeltaRow: -1})
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "2EFGH\n3IJKL\n4ABCD", contents)
	}

	// Next, we scroll back down by 1, this grows the scrollback but we
	// shouldn't move.
	zigTry(t, s.CursorDownScroll())
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "2EFGH\n3IJKL\n4ABCD", contents)
	}

	// Scroll again, this clears scrollback so we should move viewports
	// but still see the same thing since our original view fits.
	zigTry(t, s.CursorDownScroll())
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "2EFGH\n3IJKL\n4ABCD", contents)
	}
}

// Zig: test "Screen: scrolling moves selection"
func TestSc1_Screen_scrolling_moves_selection(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 1)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL")

	// Select a single line
	sc1Select(t, s, SelectionInit(
		sc1Pin(t, s, sc1Active(0, 1)),
		sc1Pin(t, s, sc1Active(s.Pages.cols-1, 1)),
		false,
	))

	// Scroll down, should still be bottom
	zigTry(t, s.CursorDownScroll())

	// Our selection should've moved up
	{
		sel := zigUnwrap(t, s.Selection)
		expectEqualPoint(t, sc1Active(0, 0), zigUnwrap(t, s.Pages.PointFromPin(PointTagActive, sel.Start())))
		expectEqualPoint(t, sc1Active(s.Pages.cols-1, 0), zigUnwrap(t, s.Pages.PointFromPin(PointTagActive, sel.End())))
	}

	{
		// Test our contents rotated
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "2EFGH\n3IJKL", contents)
	}

	// Scrolling to the bottom does nothing
	s.Scroll(ScreenScroll{Tag: ScreenScrollActive})

	// Our selection should've stayed the same
	{
		sel := zigUnwrap(t, s.Selection)
		expectEqualPoint(t, sc1Active(0, 0), zigUnwrap(t, s.Pages.PointFromPin(PointTagActive, sel.Start())))
		expectEqualPoint(t, sc1Active(s.Pages.cols-1, 0), zigUnwrap(t, s.Pages.PointFromPin(PointTagActive, sel.End())))
	}

	{
		// Test our contents rotated
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "2EFGH\n3IJKL", contents)
	}

	// Scroll up again
	zigTry(t, s.CursorDownScroll())

	{
		// Test our contents rotated
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "3IJKL", contents)
	}

	// Our selection should be null because it left the screen.
	{
		sel := zigUnwrap(t, s.Selection)
		expect(t, !s.Pages.PointFromPin(PointTagActive, sel.Start()).Valid, "s.pages.pointFromPin(.active, sel.start()) == null")
		expect(t, !s.Pages.PointFromPin(PointTagActive, sel.End()).Valid, "s.pages.pointFromPin(.active, sel.end()) == null")
	}
}

// Zig: test "Screen: scrolling moves viewport"
func TestSc1_Screen_scrolling_moves_viewport(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 1)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL\n")
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL")
	s.Scroll(ScreenScroll{Tag: ScreenScrollDeltaRow, DeltaRow: -2})

	{
		// Test our contents rotated
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "2EFGH\n3IJKL\n1ABCD", contents)
	}

	{
		expectEqualPoint(t, sc1Screen(0, 1), zigUnwrap(t, s.Pages.PointFromPin(PointTagScreen, s.Pages.GetTopLeft(PointTagViewport))))
	}
}

// Zig: test "Screen: scrolling when viewport is pruned"
func TestSc1_Screen_scrolling_when_viewport_is_pruned(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 215, 3, 1)
	defer s.Deinit()

	// Write some to create scrollback and move back into our scrollback.
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL\n")
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL")
	s.Scroll(ScreenScroll{Tag: ScreenScrollDeltaRow, DeltaRow: -2})

	// Our viewport is now somewhere pinned. Create so much scrollback
	// that we prune it.
	sc1Write(t, s, "\n")
	for i := 0; i < 1000; i++ {
		sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL\n")
	}
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL")

	{
		expectEqualPoint(t, sc1Screen(0, 0), zigUnwrap(t, s.Pages.PointFromPin(PointTagScreen, s.Pages.GetTopLeft(PointTagViewport))))
	}
}

// Zig: test "Screen: scroll and clear full screen"
func TestSc1_Screen_scroll_and_clear_full_screen(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 5)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL")

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "1ABCD\n2EFGH\n3IJKL", contents)
	}

	zigTry(t, s.ScrollClear())
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "", contents)
	}
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "1ABCD\n2EFGH\n3IJKL", contents)
	}
}

// Zig: test "Screen: scroll and clear partial screen"
func TestSc1_Screen_scroll_and_clear_partial_screen(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 5)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH")

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "1ABCD\n2EFGH", contents)
	}

	zigTry(t, s.ScrollClear())
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "", contents)
	}
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "1ABCD\n2EFGH", contents)
	}
}

// Zig: test "Screen: scroll and clear empty screen"
func TestSc1_Screen_scroll_and_clear_empty_screen(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 5)
	defer s.Deinit()
	zigTry(t, s.ScrollClear())
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "", contents)
	}
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "", contents)
	}
}

// Zig: test "Screen: scroll and clear ignore blank lines"
func TestSc1_Screen_scroll_and_clear_ignore_blank_lines(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 10)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH")
	zigTry(t, s.ScrollClear())
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "", contents)
	}

	// Move back to top-left
	s.CursorAbsolute(0, 0)

	// Write and clear
	sc1Write(t, s, "3ABCD\n")
	{
		contents := sc1Dump(t, s, sc1Active(0, 0))
		expectEqualStrings(t, "3ABCD", contents)
	}

	zigTry(t, s.ScrollClear())
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "", contents)
	}

	// Move back to top-left
	s.CursorAbsolute(0, 0)
	sc1Write(t, s, "X")

	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "1ABCD\n2EFGH\n3ABCD\nX", contents)
	}
}

// Zig: test "Screen: scroll above same page"
func TestSc1_Screen_scroll_above_same_page(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 10)
	defer s.Deinit()
	zigTry(t, s.SetAttribute(sc1DirectColorBg(155)))
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL")
	s.CursorAbsolute(0, 1)
	s.Pages.ClearDirty()

	// At this point:
	//  +-------------+ ACTIVE
	//   +----------+ : = PAGE 0
	// 0 |1ABCD00000| | 0
	// 1 |2EFGH00000| | 1
	//   :^         : : = PIN 0
	// 2 |3IJKL00000| | 2
	//   +----------+ :
	//  +-------------+

	zigTry(t, s.CursorScrollAbove())

	//   +----------+ = PAGE 0
	// 0 |1ABCD00000|
	//  +-------------+ ACTIVE
	// 1 |2EFGH00000| | 0
	// 2 |          | | 1
	//   :^         : : = PIN 0
	// 3 |3IJKL00000| | 2
	//   +----------+ :
	//  +-------------+

	// try s.pages.diagram(std.io.getStdErr().writer());

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "2EFGH\n\n3IJKL", contents)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Active(0, 1)))
		cell := list_cell.Cell
		expect(t, cell.content_tag == PageCellContentTagBgColorRgb, "cell.content_tag == .bg_color_rgb")
		expectEqual(t, PageCellRGB{R: 155, G: 0, B: 0}, cell.content.color_rgb)
	}

	// Page 0 row 1 (active row 0) is dirty because the cursor moved off of it.
	expect(t, s.Pages.IsDirty(sc1Active(0, 0)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	// Page 0 row 2 (active row 1) is dirty because it was cleared.
	expect(t, s.Pages.IsDirty(sc1Active(0, 1)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	// Page 0 row 3 (active row 2) is dirty because it's new.
	expect(t, s.Pages.IsDirty(sc1Active(0, 2)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")
}

// Zig: test "Screen: scroll above same page but cursor on previous page"
func TestSc1_Screen_scroll_above_same_page_but_cursor_on_previous_page(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 5, 10)
	defer s.Deinit()

	// We need to get the cursor to a new page
	first_page_size := s.Pages.pages.first.data.capacity.rows
	s.Pages.pages.first.data.PauseIntegrityChecks(true)
	for range first_page_size - 3 {
		sc1Write(t, s, "\n")
	}
	s.Pages.pages.first.data.PauseIntegrityChecks(false)

	zigTry(t, s.SetAttribute(sc1DirectColorBg(155)))
	sc1Write(t, s, "1A\n2B\n3C\n4D\n5E")
	s.CursorAbsolute(0, 1)
	s.Pages.ClearDirty()

	// Ensure we're still on the first page and have a second
	expect(t, s.Cursor.PagePin.Node == s.Pages.pages.first, "s.cursor.page_pin.node == s.pages.pages.first.?")
	expect(t, s.Pages.pages.first.next != nil, "s.pages.pages.first.?.next != null")

	// At this point:
	//      +----------+ = PAGE 0
	//  ... :          :
	//     +-------------+ ACTIVE
	// 4305 |1A00000000| | 0
	// 4306 |2B00000000| | 1
	//      :^         : : = PIN 0
	// 4307 |3C00000000| | 2
	//      +----------+ :
	//      +----------+ : = PAGE 1
	//    0 |4D00000000| | 3
	//    1 |5E00000000| | 4
	//      +----------+ :
	//     +-------------+

	zigTry(t, s.CursorScrollAbove())

	//      +----------+ = PAGE 0
	//  ... :          :
	// 4305 |1A00000000|
	//     +-------------+ ACTIVE
	// 4306 |2B00000000| | 0
	// 4307 |          | | 1
	//      :^         : : = PIN 0
	//      +----------+ :
	//      +----------+ : = PAGE 1
	//    0 |3C00000000| | 2
	//    1 |4D00000000| | 3
	//    2 |5E00000000| | 4
	//      +----------+ :
	//     +-------------+

	// try s.pages.diagram(std.io.getStdErr().writer());

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "2B\n\n3C\n4D\n5E", contents)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Active(0, 1)))
		cell := list_cell.Cell
		expect(t, cell.content_tag == PageCellContentTagBgColorRgb, "cell.content_tag == .bg_color_rgb")
		expectEqual(t, PageCellRGB{R: 155, G: 0, B: 0}, cell.content.color_rgb)
	}

	// Page 0's penultimate row is dirty because the cursor moved off of it.
	expect(t, s.Pages.IsDirty(sc1Active(0, 0)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	// The rest of the rows are dirty because they've been modified or are new.
	expect(t, s.Pages.IsDirty(sc1Active(0, 1)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, s.Pages.IsDirty(sc1Active(0, 2)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")
	expect(t, s.Pages.IsDirty(sc1Active(0, 3)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 3 } })")
	expect(t, s.Pages.IsDirty(sc1Active(0, 4)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 4 } })")
}

// Zig: test "Screen: scroll above same page but cursor on previous page last row"
func TestSc1_Screen_scroll_above_same_page_but_cursor_on_previous_page_last_row(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 5, 10)
	defer s.Deinit()

	// We need to get the cursor to a new page
	first_page_size := s.Pages.pages.first.data.capacity.rows
	s.Pages.pages.first.data.PauseIntegrityChecks(true)
	for range first_page_size - 2 {
		sc1Write(t, s, "\n")
	}
	s.Pages.pages.first.data.PauseIntegrityChecks(false)

	zigTry(t, s.SetAttribute(sc1DirectColorBg(155)))
	sc1Write(t, s, "1A\n2B\n3C\n4D\n5E")
	s.CursorAbsolute(0, 1)
	s.Pages.ClearDirty()

	// Ensure we're still on the first page and have a second
	expect(t, s.Cursor.PagePin.Node == s.Pages.pages.first, "s.cursor.page_pin.node == s.pages.pages.first.?")
	expect(t, s.Pages.pages.first.next != nil, "s.pages.pages.first.?.next != null")

	// At this point:
	//      +----------+ = PAGE 0
	//  ... :          :
	//     +-------------+ ACTIVE
	// 4306 |1A00000000| | 0
	// 4307 |2B00000000| | 1
	//      :^         : : = PIN 0
	//      +----------+ :
	//      +----------+ : = PAGE 1
	//    0 |3C00000000| | 2
	//    1 |4D00000000| | 3
	//    2 |5E00000000| | 4
	//      +----------+ :
	//     +-------------+

	zigTry(t, s.CursorScrollAbove())

	//      +----------+ = PAGE 0
	//  ... :          :
	// 4306 |1A00000000|
	//     +-------------+ ACTIVE
	// 4307 |2B00000000| | 0
	//      +----------+ :
	//      +----------+ : = PAGE 1
	//    0 |          | | 1
	//      :^         : : = PIN 0
	//    1 |3C00000000| | 2
	//    2 |4D00000000| | 3
	//    3 |5E00000000| | 4
	//      +----------+ :
	//     +-------------+

	// try s.pages.diagram(std.io.getStdErr().writer());

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "2B\n\n3C\n4D\n5E", contents)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Active(0, 1)))
		cell := list_cell.Cell
		expect(t, cell.content_tag == PageCellContentTagBgColorRgb, "cell.content_tag == .bg_color_rgb")
		expectEqual(t, PageCellRGB{R: 155, G: 0, B: 0}, cell.content.color_rgb)
	}

	// Page 0's final row is dirty because the cursor moved off of it.
	expect(t, s.Pages.IsDirty(sc1Active(0, 0)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	// Page 1's rows are all dirty because every row was moved.
	expect(t, s.Pages.IsDirty(sc1Active(0, 1)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, s.Pages.IsDirty(sc1Active(0, 2)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")
	expect(t, s.Pages.IsDirty(sc1Active(0, 3)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 3 } })")
	expect(t, s.Pages.IsDirty(sc1Active(0, 4)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 4 } })")

	// Attempt to clear the style from the cursor and
	// then assert the integrity of both of our pages.
	//
	// This catches a case of memory corruption where the cursor
	// is moved between pages without accounting for style refs.
	sc1SetAttr(t, s, SGRAttributeTagResetBg)
	s.Pages.pages.first.data.AssertIntegrity()
	s.Pages.pages.last.data.AssertIntegrity()
}

// Zig: test "Screen: scroll above creates new page"
func TestSc1_Screen_scroll_above_creates_new_page(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 10)
	defer s.Deinit()

	// We need to get the cursor to a new page
	first_page_size := s.Pages.pages.first.data.capacity.rows
	s.Pages.pages.first.data.PauseIntegrityChecks(true)
	for range first_page_size - 3 {
		sc1Write(t, s, "\n")
	}
	s.Pages.pages.first.data.PauseIntegrityChecks(false)

	zigTry(t, s.SetAttribute(sc1DirectColorBg(155)))
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL")
	s.CursorAbsolute(0, 1)
	s.Pages.ClearDirty()

	// Ensure we're still on the first page
	expect(t, s.Cursor.PagePin.Node == s.Pages.pages.first, "s.cursor.page_pin.node == s.pages.pages.first.?")

	// At this point:
	//      +----------+ = PAGE 0
	//  ... :          :
	//     +-------------+ ACTIVE
	// 4305 |1ABCD00000| | 0
	// 4306 |2EFGH00000| | 1
	//      :^         : : = PIN 0
	// 4307 |3IJKL00000| | 2
	//      +----------+ :
	//     +-------------+
	zigTry(t, s.CursorScrollAbove())

	//      +----------+ = PAGE 0
	//  ... :          :
	// 4305 |1ABCD00000|
	//     +-------------+ ACTIVE
	// 4306 |2EFGH00000| | 0
	// 4307 |          | | 1
	//      :^         : : = PIN 0
	//      +----------+ :
	//      +----------+ : = PAGE 1
	//    0 |3IJKL00000| | 2
	//      +----------+ :
	//     +-------------+

	// try s.pages.diagram(std.io.getStdErr().writer());

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "2EFGH\n\n3IJKL", contents)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Active(0, 1)))
		cell := list_cell.Cell
		expect(t, cell.content_tag == PageCellContentTagBgColorRgb, "cell.content_tag == .bg_color_rgb")
		expectEqual(t, PageCellRGB{R: 155, G: 0, B: 0}, cell.content.color_rgb)
	}

	// Page 0's penultimate row is dirty because the cursor moved off of it.
	expect(t, s.Pages.IsDirty(sc1Active(0, 0)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	// Page 0's final row is dirty because it was cleared.
	expect(t, s.Pages.IsDirty(sc1Active(0, 1)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	// Page 1's row is dirty because it's new.
	expect(t, s.Pages.IsDirty(sc1Active(0, 2)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")
}

// Zig: test "Screen: scroll above with cursor on non-final row"
func TestSc1_Screen_scroll_above_with_cursor_on_non_final_row(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 4, 10)
	defer s.Deinit()

	// Get the cursor to be 2 rows above a new page
	first_page_size := s.Pages.pages.first.data.capacity.rows
	s.Pages.pages.first.data.PauseIntegrityChecks(true)
	for range first_page_size - 3 {
		sc1Write(t, s, "\n")
	}
	s.Pages.pages.first.data.PauseIntegrityChecks(false)

	// Write 3 lines of text, forcing the last line into the first
	// row of a new page. Move our cursor onto the previous page.
	zigTry(t, s.SetAttribute(sc1DirectColorBg(155)))
	sc1Write(t, s, "1AB\n2BC\n3DE\n4FG")
	s.CursorAbsolute(0, 1)
	s.Pages.ClearDirty()

	// Ensure we're still on the first page. So our cursor is on the first
	// page but we have two pages of data.
	expect(t, s.Cursor.PagePin.Node == s.Pages.pages.first, "s.cursor.page_pin.node == s.pages.pages.first.?")

	//      +----------+ = PAGE 0
	//  ... :          :
	//     +-------------+ ACTIVE
	// 4305 |1AB0000000| | 0
	// 4306 |2BC0000000| | 1
	//      :^         : : = PIN 0
	// 4307 |3DE0000000| | 2
	//      +----------+ :
	//      +----------+ : = PAGE 1
	//    0 |4FG0000000| | 3
	//      +----------+ :
	//     +-------------+
	zigTry(t, s.CursorScrollAbove())

	//     +----------+ = PAGE 0
	//  ... :          :
	// 4305 |1AB0000000|
	//     +-------------+ ACTIVE
	// 4306 |2BC0000000| | 0
	// 4307 |          | | 1
	//      :^         : : = PIN 0
	//      +----------+ :
	//      +----------+ : = PAGE 1
	//    0 |3DE0000000| | 2
	//    1 |4FG0000000| | 3
	//      +----------+ :
	//     +-------------+
	// try s.pages.diagram(std.io.getStdErr().writer());

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "2BC\n\n3DE\n4FG", contents)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Active(0, 1)))
		cell := list_cell.Cell
		expect(t, cell.content_tag == PageCellContentTagBgColorRgb, "cell.content_tag == .bg_color_rgb")
		expectEqual(t, PageCellRGB{R: 155, G: 0, B: 0}, cell.content.color_rgb)
	}

	// Page 0's penultimate row is dirty because the cursor moved off of it.
	expect(t, s.Pages.IsDirty(sc1Active(0, 0)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	// Page 0's final row is dirty because it was cleared.
	expect(t, s.Pages.IsDirty(sc1Active(0, 1)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	// Page 1's row is dirty because it's new.
	expect(t, s.Pages.IsDirty(sc1Active(0, 2)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")
}

// Zig: test "Screen: scroll above no scrollback bottom of page"
func TestSc1_Screen_scroll_above_no_scrollback_bottom_of_page(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 0)
	defer s.Deinit()

	first_page_size := s.Pages.pages.first.data.capacity.rows
	s.Pages.pages.first.data.PauseIntegrityChecks(true)
	for range first_page_size - 3 {
		sc1Write(t, s, "\n")
	}
	s.Pages.pages.first.data.PauseIntegrityChecks(false)

	zigTry(t, s.SetAttribute(sc1DirectColorBg(155)))
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL")
	s.CursorAbsolute(0, 1)
	s.Pages.ClearDirty()

	// At this point:
	//  +-------------+ ACTIVE
	//   +----------+ : = PAGE 0
	// 0 |1ABCD00000| | 0
	// 1 |2EFGH00000| | 1
	//   :^         : : = PIN 0
	// 2 |3IJKL00000| | 2
	//   +----------+ :
	//  +-------------+

	zigTry(t, s.CursorScrollAbove())

	//   +----------+ = PAGE 0
	// 0 |1ABCD00000|
	//  +-------------+ ACTIVE
	// 1 |2EFGH00000| | 0
	// 2 |          | | 1
	//   :^         : : = PIN 0
	// 3 |3IJKL00000| | 2
	//   +----------+ :
	//  +-------------+

	//try s.pages.diagram(std.io.getStdErr().writer());

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "2EFGH\n\n3IJKL", contents)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Active(0, 1)))
		cell := list_cell.Cell
		expect(t, cell.content_tag == PageCellContentTagBgColorRgb, "cell.content_tag == .bg_color_rgb")
		expectEqual(t, PageCellRGB{R: 155, G: 0, B: 0}, cell.content.color_rgb)
	}

	// Page 0 row 1 (active row 0) is dirty because the cursor moved off of it.
	expect(t, s.Pages.IsDirty(sc1Active(0, 0)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	// Page 0 row 2 (active row 1) is dirty because it was cleared.
	expect(t, s.Pages.IsDirty(sc1Active(0, 1)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	// Page 0 row 3 (active row 2) is dirty because it is new.
	expect(t, s.Pages.IsDirty(sc1Active(0, 2)), "s.pages.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")
}

// Zig: test "Screen: clone"
func TestSc1_Screen_clone(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 10)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH")
	{
		contents := sc1Dump(t, s, sc1Active(0, 0))
		expectEqualStrings(t, "1ABCD\n2EFGH", contents)
	}
	expectEqual[uint](t, 5, uint(s.Cursor.X))
	expectEqual[uint](t, 1, uint(s.Cursor.Y))

	// Clone
	s2 := sc1Clone(t, s, sc1Active(0, 0), None[Point]())
	defer s2.Deinit()
	{
		contents := sc1Dump(t, s2, sc1Active(0, 0))
		expectEqualStrings(t, "1ABCD\n2EFGH", contents)
	}
	expectEqual[uint](t, 5, uint(s2.Cursor.X))
	expectEqual[uint](t, 1, uint(s2.Cursor.Y))

	// Write to s1, should not be in s2
	sc1Write(t, s, "\n34567")
	{
		contents := sc1Dump(t, s, sc1Active(0, 0))
		expectEqualStrings(t, "1ABCD\n2EFGH\n34567", contents)
	}
	{
		contents := sc1Dump(t, s2, sc1Active(0, 0))
		expectEqualStrings(t, "1ABCD\n2EFGH", contents)
	}
	expectEqual[uint](t, 5, uint(s2.Cursor.X))
	expectEqual[uint](t, 1, uint(s2.Cursor.Y))
}

// Zig: test "Screen: clone partial"
func TestSc1_Screen_clone_partial(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 10)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH")
	{
		contents := sc1Dump(t, s, sc1Active(0, 0))
		expectEqualStrings(t, "1ABCD\n2EFGH", contents)
	}
	expectEqual[uint](t, 5, uint(s.Cursor.X))
	expectEqual[uint](t, 1, uint(s.Cursor.Y))

	// Clone
	s2 := sc1Clone(t, s, sc1Active(0, 1), None[Point]())
	defer s2.Deinit()
	{
		contents := sc1Dump(t, s2, sc1Active(0, 0))
		expectEqualStrings(t, "2EFGH", contents)
	}

	// Cursor is shifted since we cloned partial
	expectEqual[uint](t, 5, uint(s2.Cursor.X))
	expectEqual[uint](t, 0, uint(s2.Cursor.Y))
}

// Zig: test "Screen: clone partial cursor out of bounds"
func TestSc1_Screen_clone_partial_cursor_out_of_bounds(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 10)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH")
	{
		contents := sc1Dump(t, s, sc1Active(0, 0))
		expectEqualStrings(t, "1ABCD\n2EFGH", contents)
	}
	expectEqual[uint](t, 5, uint(s.Cursor.X))
	expectEqual[uint](t, 1, uint(s.Cursor.Y))

	// Clone
	s2 := sc1Clone(t, s, sc1Active(0, 0), Some(sc1Active(0, 0)))
	defer s2.Deinit()
	{
		contents := sc1Dump(t, s2, sc1Active(0, 0))
		expectEqualStrings(t, "1ABCD", contents)
	}

	// Cursor is shifted since we cloned partial
	expectEqual[uint](t, 0, uint(s2.Cursor.X))
	expectEqual[uint](t, 0, uint(s2.Cursor.Y))
}

// Zig: test "Screen: clone contains full selection"
func TestSc1_Screen_clone_contains_full_selection(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 1)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL")

	// Select a single line
	sc1Select(t, s, SelectionInit(
		sc1Pin(t, s, sc1Active(0, 1)),
		sc1Pin(t, s, sc1Active(s.Pages.cols-1, 1)),
		false,
	))

	// Clone
	s2 := sc1Clone(t, s, sc1Active(0, 0), None[Point]())
	defer s2.Deinit()

	// Our selection should remain valid
	{
		sel := zigUnwrap(t, s2.Selection)
		expectEqualPoint(t, sc1Active(0, 1), zigUnwrap(t, s2.Pages.PointFromPin(PointTagActive, sel.Start())))
		expectEqualPoint(t, sc1Active(s2.Pages.cols-1, 1), zigUnwrap(t, s2.Pages.PointFromPin(PointTagActive, sel.End())))
	}
}

// Zig: test "Screen: clone contains none of selection"
func TestSc1_Screen_clone_contains_none_of_selection(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 1)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL")

	// Select a single line
	sc1Select(t, s, SelectionInit(
		sc1Pin(t, s, sc1Active(0, 0)),
		sc1Pin(t, s, sc1Active(s.Pages.cols-1, 0)),
		false,
	))

	// Clone
	s2 := sc1Clone(t, s, sc1Active(0, 1), None[Point]())
	defer s2.Deinit()

	// Our selection should be null
	expect(t, !s2.Selection.Valid, "s2.selection == null")
}

// Zig: test "Screen: clone contains selection start cutoff"
func TestSc1_Screen_clone_contains_selection_start_cutoff(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 1)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL")

	// Select a single line
	sc1Select(t, s, SelectionInit(
		sc1Pin(t, s, sc1Active(0, 0)),
		sc1Pin(t, s, sc1Active(s.Pages.cols-1, 1)),
		false,
	))

	// Clone
	s2 := sc1Clone(t, s, sc1Active(0, 1), None[Point]())
	defer s2.Deinit()

	// Our selection should remain valid
	{
		sel := zigUnwrap(t, s2.Selection)
		expectEqualPoint(t, sc1Active(0, 0), zigUnwrap(t, s2.Pages.PointFromPin(PointTagActive, sel.Start())))
		expectEqualPoint(t, sc1Active(s2.Pages.cols-1, 0), zigUnwrap(t, s2.Pages.PointFromPin(PointTagActive, sel.End())))
	}
}

// Zig: test "Screen: clone contains selection end cutoff"
func TestSc1_Screen_clone_contains_selection_end_cutoff(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 1)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL")

	// Select a single line
	sc1Select(t, s, SelectionInit(
		sc1Pin(t, s, sc1Active(0, 1)),
		sc1Pin(t, s, sc1Active(2, 2)),
		false,
	))

	// Clone
	s2 := sc1Clone(t, s, sc1Active(0, 0), Some(sc1Active(0, 1)))
	defer s2.Deinit()

	// Our selection should remain valid
	{
		sel := zigUnwrap(t, s2.Selection)
		expectEqualPoint(t, sc1Active(0, 1), zigUnwrap(t, s2.Pages.PointFromPin(PointTagActive, sel.Start())))
		expectEqualPoint(t, sc1Active(s2.Pages.cols-1, 2), zigUnwrap(t, s2.Pages.PointFromPin(PointTagActive, sel.End())))
	}
}

// Zig: test "Screen: clone contains selection end cutoff reversed"
func TestSc1_Screen_clone_contains_selection_end_cutoff_reversed(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 1)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL")

	// Select a single line
	sc1Select(t, s, SelectionInit(
		sc1Pin(t, s, sc1Active(2, 2)),
		sc1Pin(t, s, sc1Active(0, 1)),
		false,
	))

	// Clone
	s2 := sc1Clone(t, s, sc1Active(0, 0), Some(sc1Active(0, 1)))
	defer s2.Deinit()

	// Our selection should remain valid
	{
		sel := zigUnwrap(t, s2.Selection)
		expectEqualPoint(t, sc1Active(0, 1), zigUnwrap(t, s2.Pages.PointFromPin(PointTagActive, sel.Start())))
		expectEqualPoint(t, sc1Active(s2.Pages.cols-1, 2), zigUnwrap(t, s2.Pages.PointFromPin(PointTagActive, sel.End())))
	}
}

// Zig: test "Screen: clone contains subset of selection"
func TestSc1_Screen_clone_contains_subset_of_selection(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 4, 1)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL\n4ABCD")

	// Select the full screen
	sc1Select(t, s, SelectionInit(
		sc1Pin(t, s, sc1Active(0, 0)),
		sc1Pin(t, s, sc1Active(0, 3)),
		false,
	))

	// Clone
	s2 := sc1Clone(t, s, sc1Active(0, 1), Some(sc1Active(0, 2)))
	defer s2.Deinit()

	// Our selection should remain valid
	{
		sel := zigUnwrap(t, s2.Selection)
		expectEqualPoint(t, sc1Active(0, 0), zigUnwrap(t, s2.Pages.PointFromPin(PointTagActive, sel.Start())))
		expectEqualPoint(t, sc1Active(s2.Pages.cols-1, 3), zigUnwrap(t, s2.Pages.PointFromPin(PointTagActive, sel.End())))
	}
}

// Zig: test "Screen: clone contains subset of rectangle selection"
func TestSc1_Screen_clone_contains_subset_of_rectangle_selection(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 4, 1)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL\n4ABCD")

	// Select the full screen from x=1 to x=3
	sc1Select(t, s, SelectionInit(
		sc1Pin(t, s, sc1Active(1, 0)),
		sc1Pin(t, s, sc1Active(3, 3)),
		true,
	))

	// Clone
	s2 := sc1Clone(t, s, sc1Active(0, 1), Some(sc1Active(0, 2)))
	defer s2.Deinit()

	// Our selection should remain valid and be properly clipped
	// preserving the columns of the start and end points of the
	// selection.
	{
		sel := zigUnwrap(t, s2.Selection)
		expectEqualPoint(t, sc1Active(1, 0), zigUnwrap(t, s2.Pages.PointFromPin(PointTagActive, sel.Start())))
		expectEqualPoint(t, sc1Active(3, 3), zigUnwrap(t, s2.Pages.PointFromPin(PointTagActive, sel.End())))
	}
}

// Zig: test "Screen: clone basic"
func TestSc1_Screen_clone_basic(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 0)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL")

	{
		s2 := sc1Clone(t, s, sc1Active(0, 1), Some(sc1Active(0, 1)))
		defer s2.Deinit()

		// Test our contents rotated
		contents := sc1Dump(t, s2, sc1Active(0, 0))
		expectEqualStrings(t, "2EFGH", contents)
	}

	{
		s2 := sc1Clone(t, s, sc1Active(0, 1), Some(sc1Active(0, 2)))
		defer s2.Deinit()

		// Test our contents rotated
		contents := sc1Dump(t, s2, sc1Active(0, 0))
		expectEqualStrings(t, "2EFGH\n3IJKL", contents)
	}
}

// Zig: test "Screen: clone empty viewport"
func TestSc1_Screen_clone_empty_viewport(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 0)
	defer s.Deinit()

	{
		s2 := sc1Clone(t, s, sc1Viewport(0, 0), Some(sc1Viewport(0, 0)))
		defer s2.Deinit()

		// Test our contents rotated
		contents := sc1Dump(t, s2, sc1Viewport(0, 0))
		expectEqualStrings(t, "", contents)
	}
}

// Zig: test "Screen: clone one line viewport"
func TestSc1_Screen_clone_one_line_viewport(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 0)
	defer s.Deinit()
	sc1Write(t, s, "1ABC")

	{
		s2 := sc1Clone(t, s, sc1Viewport(0, 0), Some(sc1Viewport(0, 0)))
		defer s2.Deinit()

		// Test our contents
		contents := sc1Dump(t, s2, sc1Viewport(0, 0))
		expectEqualStrings(t, "1ABC", contents)
	}
}

// Zig: test "Screen: clone empty active"
func TestSc1_Screen_clone_empty_active(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 0)
	defer s.Deinit()

	{
		s2 := sc1Clone(t, s, sc1Active(0, 0), Some(sc1Active(0, 0)))
		defer s2.Deinit()

		// Test our contents rotated
		contents := sc1Dump(t, s2, sc1Active(0, 0))
		expectEqualStrings(t, "", contents)
	}
}

// Zig: test "Screen: clone one line active with extra space"
func TestSc1_Screen_clone_one_line_active_with_extra_space(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 0)
	defer s.Deinit()
	sc1Write(t, s, "1ABC")

	{
		s2 := sc1Clone(t, s, sc1Active(0, 0), None[Point]())
		defer s2.Deinit()

		// Test our contents rotated
		contents := sc1Dump(t, s2, sc1Active(0, 0))
		expectEqualStrings(t, "1ABC", contents)
	}
}

// Zig: test "Screen: clear history with no history"
func TestSc1_Screen_clear_history_with_no_history(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 3)
	defer s.Deinit()
	sc1Write(t, s, "4ABCD\n5EFGH\n6IJKL")
	expect(t, s.Pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")
	s.EraseRows(sc1History(0, 0), None[Point]())
	expect(t, s.Pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")
	{
		// Test our contents rotated
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "4ABCD\n5EFGH\n6IJKL", contents)
	}
	{
		// Test our contents rotated
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "4ABCD\n5EFGH\n6IJKL", contents)
	}
}

// Zig: test "Screen: clear history"
func TestSc1_Screen_clear_history(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 3)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL\n4ABCD\n5EFGH\n6IJKL")
	expect(t, s.Pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")

	// Scroll to top
	s.Scroll(ScreenScroll{Tag: ScreenScrollTop})
	{
		// Test our contents rotated
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "1ABCD\n2EFGH\n3IJKL", contents)
	}

	s.EraseRows(sc1History(0, 0), None[Point]())
	expect(t, s.Pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")
	{
		// Test our contents rotated
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "4ABCD\n5EFGH\n6IJKL", contents)
	}
	{
		// Test our contents rotated
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "4ABCD\n5EFGH\n6IJKL", contents)
	}
}

// Zig: test "Screen: clear above cursor"
func TestSc1_Screen_clear_above_cursor(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 10, 3)
	defer s.Deinit()
	sc1Write(t, s, "4ABCD\n5EFGH\n6IJKL")
	s.ClearRows(
		sc1Active(0, 0),
		Some(sc1Active(0, uint32(s.Cursor.Y-1))),
		false,
	)
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "\n\n6IJKL", contents)
	}
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "\n\n6IJKL", contents)
	}

	expectEqual[uint](t, 5, uint(s.Cursor.X))
	expectEqual[uint](t, 2, uint(s.Cursor.Y))
}

// Zig: test "Screen: clear above cursor with history"
func TestSc1_Screen_clear_above_cursor_with_history(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 3)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL\n")
	sc1Write(t, s, "4ABCD\n5EFGH\n6IJKL")
	s.ClearRows(
		sc1Active(0, 0),
		Some(sc1Active(0, uint32(s.Cursor.Y-1))),
		false,
	)
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "\n\n6IJKL", contents)
	}
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "1ABCD\n2EFGH\n3IJKL\n\n\n6IJKL", contents)
	}

	expectEqual[uint](t, 5, uint(s.Cursor.X))
	expectEqual[uint](t, 2, uint(s.Cursor.Y))
}

// Zig: test "Screen: resize (no reflow) more rows"
func TestSc1_Screen_resize_no_reflow_more_rows(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 0)
	defer s.Deinit()
	const str = "1ABCD\n2EFGH\n3IJKL"
	sc1Write(t, s, str)

	// Resize
	zigTry(t, s.Resize(sc1NoReflow(10, 10)))
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, str, contents)
	}
}

// Zig: test "Screen: resize (no reflow) less rows"
func TestSc1_Screen_resize_no_reflow_less_rows(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 0)
	defer s.Deinit()
	const str = "1ABCD\n2EFGH\n3IJKL"
	sc1Write(t, s, str)
	expectEqual[SizeCellCountInt](t, 5, s.Cursor.X)
	expectEqual[SizeCellCountInt](t, 2, s.Cursor.Y)
	zigTry(t, s.Resize(sc1NoReflow(10, 2)))

	// Since we shrunk, we should adjust our cursor
	expectEqual[SizeCellCountInt](t, 5, s.Cursor.X)
	expectEqual[SizeCellCountInt](t, 1, s.Cursor.Y)

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "2EFGH\n3IJKL", contents)
	}
}

// Zig: test "Screen: resize (no reflow) less rows trims blank lines"
func TestSc1_Screen_resize_no_reflow_less_rows_trims_blank_lines(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 0)
	defer s.Deinit()
	const str = "1ABCD"
	sc1Write(t, s, str)

	// Write only a background color into the remaining rows
	for y := 1; y < int(s.Pages.rows); y++ {
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Active(0, uint32(y))))
		*list_cell.Cell = PageCell{
			content_tag: PageCellContentTagBgColorRgb,
			content:     PageCellContent{color_rgb: PageCellRGB{R: 0xFF, G: 0, B: 0}},
		}
	}

	cursor := s.Cursor
	zigTry(t, s.Resize(sc1NoReflow(6, 2)))

	// Cursor should not move
	expectEqual(t, cursor.X, s.Cursor.X)
	expectEqual(t, cursor.Y, s.Cursor.Y)

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "1ABCD", contents)
	}
}

// Zig: test "Screen: resize (no reflow) more rows trims blank lines"
func TestSc1_Screen_resize_no_reflow_more_rows_trims_blank_lines(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 0)
	defer s.Deinit()
	const str = "1ABCD"
	sc1Write(t, s, str)

	// Write only a background color into the remaining rows
	for y := 1; y < int(s.Pages.rows); y++ {
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Active(0, uint32(y))))
		*list_cell.Cell = PageCell{
			content_tag: PageCellContentTagBgColorRgb,
			content:     PageCellContent{color_rgb: PageCellRGB{R: 0xFF, G: 0, B: 0}},
		}
	}

	cursor := s.Cursor
	zigTry(t, s.Resize(sc1NoReflow(10, 7)))

	// Cursor should not move
	expectEqual(t, cursor.X, s.Cursor.X)
	expectEqual(t, cursor.Y, s.Cursor.Y)

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "1ABCD", contents)
	}
}

// Zig: test "Screen: resize (no reflow) more cols"
func TestSc1_Screen_resize_no_reflow_more_cols(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 0)
	defer s.Deinit()
	const str = "1ABCD\n2EFGH\n3IJKL"
	sc1Write(t, s, str)
	zigTry(t, s.Resize(sc1NoReflow(20, 3)))

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, str, contents)
	}
}

// Zig: test "Screen: resize (no reflow) less cols"
func TestSc1_Screen_resize_no_reflow_less_cols(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 0)
	defer s.Deinit()
	const str = "1ABCD\n2EFGH\n3IJKL"
	sc1Write(t, s, str)
	zigTry(t, s.Resize(sc1NoReflow(4, 3)))

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "1ABC\n2EFG\n3IJK"
		expectEqualStrings(t, expected, contents)
	}
}

// Zig: test "Screen: resize (no reflow) more rows with scrollback cursor end"
func TestSc1_Screen_resize_no_reflow_more_rows_with_scrollback_cursor_end(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 7, 3, 2)
	defer s.Deinit()
	const str = "1ABCD\n2EFGH\n3IJKL\n4ABCD\n5EFGH"
	sc1Write(t, s, str)
	zigTry(t, s.Resize(sc1NoReflow(7, 10)))

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, str, contents)
	}
}

// Zig: test "Screen: resize (no reflow) less rows with scrollback"
func TestSc1_Screen_resize_no_reflow_less_rows_with_scrollback(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 7, 3, 2)
	defer s.Deinit()
	const str = "1ABCD\n2EFGH\n3IJKL\n4ABCD\n5EFGH"
	sc1Write(t, s, str)
	zigTry(t, s.Resize(sc1NoReflow(7, 2)))

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "4ABCD\n5EFGH"
		expectEqualStrings(t, expected, contents)
	}
}

// https://github.com/mitchellh/ghostty/issues/1030
//
// Zig: test "Screen: resize (no reflow) less rows with empty trailing"
func TestSc1_Screen_resize_no_reflow_less_rows_with_empty_trailing(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 5)
	defer s.Deinit()
	const str = "1\n2\n3\n4\n5\n6\n7\n8"
	sc1Write(t, s, str)
	zigTry(t, s.ScrollClear())
	s.CursorAbsolute(0, 0)
	sc1Write(t, s, "A\nB")

	cursor := s.Cursor
	zigTry(t, s.Resize(sc1NoReflow(5, 2)))
	expectEqual(t, cursor.X, s.Cursor.X)
	expectEqual(t, cursor.Y, s.Cursor.Y)

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "A\nB", contents)
	}
}

// Zig: test "Screen: resize (no reflow) more rows with soft wrapping"
func TestSc1_Screen_resize_no_reflow_more_rows_with_soft_wrapping(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 2, 3, 3)
	defer s.Deinit()
	const str = "1A2B\n3C4E\n5F6G"
	sc1Write(t, s, str)

	// Every second row should be wrapped
	for y := 0; y < 6; y++ {
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(0, uint32(y))))
		row := list_cell.Row
		wrapped := (y%2 == 0)
		expectEqual(t, wrapped, row.wrap)
	}

	// Resize
	zigTry(t, s.Resize(sc1NoReflow(2, 10)))
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "1A\n2B\n3C\n4E\n5F\n6G"
		expectEqualStrings(t, expected, contents)
	}

	// Every second row should be wrapped
	for y := 0; y < 6; y++ {
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(0, uint32(y))))
		row := list_cell.Row
		wrapped := (y%2 == 0)
		expectEqual(t, wrapped, row.wrap)
	}
}

// Zig: test "Screen: resize more rows no scrollback"
func TestSc1_Screen_resize_more_rows_no_scrollback(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 0)
	defer s.Deinit()
	const str = "1ABCD\n2EFGH\n3IJKL"
	sc1Write(t, s, str)
	cursor := s.Cursor
	zigTry(t, s.Resize(screenResizeDefault(5, 10)))

	// Cursor should not move
	expectEqual(t, cursor.X, s.Cursor.X)
	expectEqual(t, cursor.Y, s.Cursor.Y)

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, str, contents)
	}
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, str, contents)
	}
}

// Zig: test "Screen: resize more rows with empty scrollback"
func TestSc1_Screen_resize_more_rows_with_empty_scrollback(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 10)
	defer s.Deinit()
	const str = "1ABCD\n2EFGH\n3IJKL"
	sc1Write(t, s, str)
	cursor := s.Cursor
	zigTry(t, s.Resize(screenResizeDefault(5, 10)))

	// Cursor should not move
	expectEqual(t, cursor.X, s.Cursor.X)
	expectEqual(t, cursor.Y, s.Cursor.Y)

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, str, contents)
	}
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, str, contents)
	}
}

// Zig: test "Screen: resize more rows with populated scrollback"
func TestSc1_Screen_resize_more_rows_with_populated_scrollback(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 5)
	defer s.Deinit()
	const str = "1ABCD\n2EFGH\n3IJKL\n4ABCD\n5EFGH"
	sc1Write(t, s, str)
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "3IJKL\n4ABCD\n5EFGH"
		expectEqualStrings(t, expected, contents)
	}

	// Set our cursor to be on the "4"
	s.CursorAbsolute(0, 1)
	{
		list_cell := sc1CursorCell(t, s)
		expectEqual[uint32](t, '4', list_cell.Cell.content.codepoint)
	}

	// Resize
	zigTry(t, s.Resize(screenResizeDefault(5, 10)))

	// Cursor should still be on the "4"
	{
		list_cell := sc1CursorCell(t, s)
		expectEqual[uint32](t, '4', list_cell.Cell.content.codepoint)
	}

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "3IJKL\n4ABCD\n5EFGH"
		expectEqualStrings(t, expected, contents)
	}
}

// Zig: test "Screen: resize more cols no reflow"
func TestSc1_Screen_resize_more_cols_no_reflow(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 0)
	defer s.Deinit()
	const str = "1ABCD\n2EFGH\n3IJKL"
	sc1Write(t, s, str)

	cursor := s.Cursor
	zigTry(t, s.Resize(screenResizeDefault(10, 3)))

	// Cursor should not move
	expectEqual(t, cursor.X, s.Cursor.X)
	expectEqual(t, cursor.Y, s.Cursor.Y)

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, str, contents)
	}
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, str, contents)
	}
}

// https://github.com/mitchellh/ghostty/issues/272#issuecomment-1676038963
//
// Zig: test "Screen: resize more cols perfect split"
func TestSc1_Screen_resize_more_cols_perfect_split(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 0)
	defer s.Deinit()
	const str = "1ABCD2EFGH3IJKL"
	sc1Write(t, s, str)
	zigTry(t, s.Resize(screenResizeDefault(10, 3)))

	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "1ABCD2EFGH\n3IJKL", contents)
	}
}

// https://github.com/mitchellh/ghostty/issues/1159
//
// Zig: test "Screen: resize (no reflow) more cols with scrollback scrolled up"
func TestSc1_Screen_resize_no_reflow_more_cols_with_scrollback_scrolled_up(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 5)
	defer s.Deinit()
	const str = "1\n2\n3\n4\n5\n6\n7\n8"
	sc1Write(t, s, str)

	// Cursor at bottom
	expectEqual[SizeCellCountInt](t, 1, s.Cursor.X)
	expectEqual[SizeCellCountInt](t, 2, s.Cursor.Y)

	s.Scroll(ScreenScroll{Tag: ScreenScrollDeltaRow, DeltaRow: -4})
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "2\n3\n4", contents)
	}

	zigTry(t, s.Resize(screenResizeDefault(8, 3)))
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, str, contents)
	}

	// Cursor remains at bottom
	expectEqual[SizeCellCountInt](t, 1, s.Cursor.X)
	expectEqual[SizeCellCountInt](t, 2, s.Cursor.Y)
}

// https://github.com/mitchellh/ghostty/issues/1159
//
// Zig: test "Screen: resize (no reflow) less cols with scrollback scrolled up"
func TestSc1_Screen_resize_no_reflow_less_cols_with_scrollback_scrolled_up(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 5)
	defer s.Deinit()
	const str = "1\n2\n3\n4\n5\n6\n7\n8"
	sc1Write(t, s, str)

	// Cursor at bottom
	expectEqual[SizeCellCountInt](t, 1, s.Cursor.X)
	expectEqual[SizeCellCountInt](t, 2, s.Cursor.Y)

	s.Scroll(ScreenScroll{Tag: ScreenScrollDeltaRow, DeltaRow: -4})
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, "2\n3\n4", contents)
	}

	zigTry(t, s.Resize(screenResizeDefault(4, 3)))
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, str, contents)
	}
	{
		contents := sc1Dump(t, s, sc1Active(0, 0))
		expectEqualStrings(t, "6\n7\n8", contents)
	}

	// Cursor remains at bottom
	expectEqual[SizeCellCountInt](t, 1, s.Cursor.X)
	expectEqual[SizeCellCountInt](t, 2, s.Cursor.Y)

	// Old implementation doesn't do this but it makes sense to me:
	// {
	//     const contents = try s.dumpStringAlloc(alloc, .{ .viewport = .{} });
	//     defer alloc.free(contents);
	//     try testing.expectEqualStrings("2\n3\n4", contents);
	// }
}

// Zig: test "Screen: resize more cols no reflow preserves semantic prompt"
func TestSc1_Screen_resize_more_cols_no_reflow_preserves_semantic_prompt(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 0)
	defer s.Deinit()

	// Set one of the rows to be a prompt
	s.CursorSetSemanticContent(sc1SemOutput())
	sc1Write(t, s, "1ABCD\n")
	s.CursorSetSemanticContent(sc1SemPrompt(OSCParsersSemanticPromptPromptKindInitial))
	sc1Write(t, s, "2EFGH")
	s.CursorSetSemanticContent(sc1SemOutput())
	sc1Write(t, s, "\n3IJKL")

	zigTry(t, s.Resize(sc1NoReflow(10, 3)))

	const expected = "1ABCD\n2EFGH\n3IJKL"
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, expected, contents)
	}
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, expected, contents)
	}

	// Our one row should still be a semantic prompt, the others should not.
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Active(0, 0)))
		expect(t, list_cell.Row.semantic_prompt == PageRowSemanticPromptNone, "list_cell.row.semantic_prompt == .none")
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Active(0, 1)))
		expect(t, list_cell.Row.semantic_prompt == PageRowSemanticPromptPrompt, "list_cell.row.semantic_prompt == .prompt")
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Active(0, 2)))
		expect(t, list_cell.Row.semantic_prompt == PageRowSemanticPromptNone, "list_cell.row.semantic_prompt == .none")
	}
}

// Zig: test "Screen: resize more cols with reflow that fits full width"
func TestSc1_Screen_resize_more_cols_with_reflow_that_fits_full_width(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 0)
	defer s.Deinit()
	const str = "1ABCD2EFGH\n3IJKL"
	sc1Write(t, s, str)

	// Verify we soft wrapped
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "1ABCD\n2EFGH\n3IJKL"
		expectEqualStrings(t, expected, contents)
	}

	// Let's put our cursor on row 2, where the soft wrap is
	s.CursorAbsolute(0, 1)
	{
		list_cell := sc1CursorCell(t, s)
		expectEqual[uint32](t, '2', list_cell.Cell.content.codepoint)
	}

	// Resize and verify we undid the soft wrap because we have space now
	zigTry(t, s.Resize(screenResizeDefault(10, 3)))
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, str, contents)
	}

	// Our cursor should've moved
	expectEqual[uint](t, 5, uint(s.Cursor.X))
	expectEqual[uint](t, 0, uint(s.Cursor.Y))
}

// Zig: test "Screen: resize more cols with reflow that ends in newline"
func TestSc1_Screen_resize_more_cols_with_reflow_that_ends_in_newline(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 6, 3, 0)
	defer s.Deinit()
	const str = "1ABCD2EFGH\n3IJKL"
	sc1Write(t, s, str)

	// Verify we soft wrapped
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "1ABCD2\nEFGH\n3IJKL"
		expectEqualStrings(t, expected, contents)
	}

	// Let's put our cursor on the last row
	s.CursorAbsolute(0, 2)
	{
		list_cell := sc1CursorCell(t, s)
		expectEqual[uint32](t, '3', list_cell.Cell.content.codepoint)
	}

	// Resize and verify we undid the soft wrap because we have space now
	zigTry(t, s.Resize(screenResizeDefault(10, 3)))
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, str, contents)
	}

	// Our cursor should still be on the 3
	{
		list_cell := sc1CursorCell(t, s)
		expectEqual[uint32](t, '3', list_cell.Cell.content.codepoint)
	}
}

// Zig: test "Screen: resize more cols with reflow that forces more wrapping"
func TestSc1_Screen_resize_more_cols_with_reflow_that_forces_more_wrapping(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 0)
	defer s.Deinit()
	const str = "1ABCD2EFGH\n3IJKL"
	sc1Write(t, s, str)

	// Let's put our cursor on row 2, where the soft wrap is
	s.CursorAbsolute(0, 1)
	{
		list_cell := sc1CursorCell(t, s)
		expectEqual[uint32](t, '2', list_cell.Cell.content.codepoint)
	}

	// Verify we soft wrapped
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "1ABCD\n2EFGH\n3IJKL"
		expectEqualStrings(t, expected, contents)
	}

	// Resize and verify we undid the soft wrap because we have space now
	zigTry(t, s.Resize(screenResizeDefault(7, 3)))
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "1ABCD2E\nFGH\n3IJKL"
		expectEqualStrings(t, expected, contents)
	}

	// Our cursor should've moved
	expectEqual[SizeCellCountInt](t, 5, s.Cursor.X)
	expectEqual[SizeCellCountInt](t, 0, s.Cursor.Y)
}

// Zig: test "Screen: resize more cols with reflow that unwraps multiple times"
func TestSc1_Screen_resize_more_cols_with_reflow_that_unwraps_multiple_times(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 0)
	defer s.Deinit()
	const str = "1ABCD2EFGH3IJKL"
	sc1Write(t, s, str)

	// Let's put our cursor on row 2, where the soft wrap is
	s.CursorAbsolute(0, 2)
	{
		list_cell := sc1CursorCell(t, s)
		expectEqual[uint32](t, '3', list_cell.Cell.content.codepoint)
	}

	// Verify we soft wrapped
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "1ABCD\n2EFGH\n3IJKL"
		expectEqualStrings(t, expected, contents)
	}

	// Resize and verify we undid the soft wrap because we have space now
	zigTry(t, s.Resize(screenResizeDefault(15, 3)))
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "1ABCD2EFGH3IJKL"
		expectEqualStrings(t, expected, contents)
	}

	// Our cursor should've moved
	expectEqual[SizeCellCountInt](t, 10, s.Cursor.X)
	expectEqual[SizeCellCountInt](t, 0, s.Cursor.Y)
}

// Zig: test "Screen: resize more cols with populated scrollback"
func TestSc1_Screen_resize_more_cols_with_populated_scrollback(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 5)
	defer s.Deinit()
	const str = "1ABCD\n2EFGH\n3IJKL\n4ABCD5EFGH"
	sc1Write(t, s, str)
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "3IJKL\n4ABCD\n5EFGH"
		expectEqualStrings(t, expected, contents)
	}

	// // Set our cursor to be on the "5"
	s.CursorAbsolute(0, 2)
	{
		list_cell := sc1CursorCell(t, s)
		expectEqual[uint32](t, '5', list_cell.Cell.content.codepoint)
	}

	// Resize
	zigTry(t, s.Resize(screenResizeDefault(10, 3)))
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "2EFGH\n3IJKL\n4ABCD5EFGH"
		expectEqualStrings(t, expected, contents)
	}

	// Cursor should still be on the "5"
	{
		list_cell := sc1CursorCell(t, s)
		expectEqual[uint32](t, '5', list_cell.Cell.content.codepoint)
	}
}

// Zig: test "Screen: resize more cols with reflow"
func TestSc1_Screen_resize_more_cols_with_reflow(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 2, 3, 5)
	defer s.Deinit()
	const str = "1ABC\n2DEF\n3ABC\n4DEF"
	sc1Write(t, s, str)

	// Let's put our cursor on row 2, where the soft wrap is
	s.CursorAbsolute(0, 2)
	{
		list_cell := sc1CursorCell(t, s)
		expectEqual[uint32](t, 'E', list_cell.Cell.content.codepoint)
	}

	// Verify we soft wrapped
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "BC\n4D\nEF"
		expectEqualStrings(t, expected, contents)
	}

	// Resize and verify we undid the soft wrap because we have space now
	zigTry(t, s.Resize(screenResizeDefault(7, 3)))
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		const expected = "1ABC\n2DEF\n3ABC\n4DEF"
		expectEqualStrings(t, expected, contents)
	}

	// Our cursor should've moved
	expectEqual[SizeCellCountInt](t, 2, s.Cursor.X)
	expectEqual[SizeCellCountInt](t, 2, s.Cursor.Y)
}

// Zig: test "Screen: resize more rows and cols with wrapping"
func TestSc1_Screen_resize_more_rows_and_cols_with_wrapping(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 2, 4, 0)
	defer s.Deinit()
	const str = "1A2B\n3C4D"
	sc1Write(t, s, str)
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "1A\n2B\n3C\n4D"
		expectEqualStrings(t, expected, contents)
	}

	zigTry(t, s.Resize(screenResizeDefault(5, 10)))

	// Cursor should move due to wrapping
	expectEqual[SizeCellCountInt](t, 3, s.Cursor.X)
	expectEqual[SizeCellCountInt](t, 1, s.Cursor.Y)

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, str, contents)
	}
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, str, contents)
	}
}

// Zig: test "Screen: resize less rows no scrollback"
func TestSc1_Screen_resize_less_rows_no_scrollback(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 0)
	defer s.Deinit()
	const str = "1ABCD\n2EFGH\n3IJKL"
	sc1Write(t, s, str)

	s.CursorAbsolute(0, 0)
	cursor := s.Cursor
	zigTry(t, s.Resize(screenResizeDefault(5, 1)))

	// Cursor should not move
	expectEqual(t, cursor.X, s.Cursor.X)
	expectEqual(t, cursor.Y, s.Cursor.Y)

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "3IJKL"
		expectEqualStrings(t, expected, contents)
	}
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		const expected = "3IJKL"
		expectEqualStrings(t, expected, contents)
	}
}

// Zig: test "Screen: resize less rows moving cursor"
func TestSc1_Screen_resize_less_rows_moving_cursor(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 0)
	defer s.Deinit()
	const str = "1ABCD\n2EFGH\n3IJKL"
	sc1Write(t, s, str)

	// Put our cursor on the last line
	s.CursorAbsolute(1, 2)
	{
		list_cell := sc1CursorCell(t, s)
		expectEqual[uint32](t, 'I', list_cell.Cell.content.codepoint)
	}

	// Resize
	zigTry(t, s.Resize(screenResizeDefault(5, 1)))

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "3IJKL"
		expectEqualStrings(t, expected, contents)
	}
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		const expected = "3IJKL"
		expectEqualStrings(t, expected, contents)
	}

	// Cursor should be on the last line
	expectEqual[SizeCellCountInt](t, 1, s.Cursor.X)
	expectEqual[SizeCellCountInt](t, 0, s.Cursor.Y)
}

// Zig: test "Screen: resize less rows with empty scrollback"
func TestSc1_Screen_resize_less_rows_with_empty_scrollback(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 10)
	defer s.Deinit()
	const str = "1ABCD\n2EFGH\n3IJKL"
	sc1Write(t, s, str)
	zigTry(t, s.Resize(screenResizeDefault(5, 1)))

	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, str, contents)
	}
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "3IJKL"
		expectEqualStrings(t, expected, contents)
	}
}

// Zig: test "Screen: resize less rows with populated scrollback"
func TestSc1_Screen_resize_less_rows_with_populated_scrollback(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 5)
	defer s.Deinit()
	const str = "1ABCD\n2EFGH\n3IJKL\n4ABCD\n5EFGH"
	sc1Write(t, s, str)
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "3IJKL\n4ABCD\n5EFGH"
		expectEqualStrings(t, expected, contents)
	}

	// Resize
	zigTry(t, s.Resize(screenResizeDefault(5, 1)))

	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, str, contents)
	}
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "5EFGH"
		expectEqualStrings(t, expected, contents)
	}
}

// Zig: test "Screen: resize less rows with full scrollback"
func TestSc1_Screen_resize_less_rows_with_full_scrollback(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 3)
	defer s.Deinit()
	const str = "00000\n1ABCD\n2EFGH\n3IJKL\n4ABCD\n5EFGH"
	sc1Write(t, s, str)
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "3IJKL\n4ABCD\n5EFGH"
		expectEqualStrings(t, expected, contents)
	}

	expectEqual[SizeCellCountInt](t, 4, s.Cursor.X)
	expectEqual[SizeCellCountInt](t, 2, s.Cursor.Y)

	// Resize
	zigTry(t, s.Resize(screenResizeDefault(5, 2)))

	// Cursor should stay in the same relative place (bottom of the
	// screen, same character).
	expectEqual[SizeCellCountInt](t, 4, s.Cursor.X)
	expectEqual[SizeCellCountInt](t, 1, s.Cursor.Y)

	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		const expected = "00000\n1ABCD\n2EFGH\n3IJKL\n4ABCD\n5EFGH"
		expectEqualStrings(t, expected, contents)
	}
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "4ABCD\n5EFGH"
		expectEqualStrings(t, expected, contents)
	}
}

// Zig: test "Screen: resize less cols no reflow"
func TestSc1_Screen_resize_less_cols_no_reflow(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 0)
	defer s.Deinit()
	const str = "1AB\n2EF\n3IJ"
	sc1Write(t, s, str)

	s.CursorAbsolute(0, 0)
	cursor := s.Cursor
	zigTry(t, s.Resize(screenResizeDefault(3, 3)))

	// Cursor should not move
	expectEqual(t, cursor.X, s.Cursor.X)
	expectEqual(t, cursor.Y, s.Cursor.Y)

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, str, contents)
	}
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, str, contents)
	}
}

// Zig: test "Screen: resize less cols with reflow but row space"
func TestSc1_Screen_resize_less_cols_with_reflow_but_row_space(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 1)
	defer s.Deinit()
	const str = "1ABCD"
	sc1Write(t, s, str)

	// Put our cursor on the end
	s.CursorAbsolute(4, 0)
	{
		list_cell := sc1CursorCell(t, s)
		expectEqual[uint32](t, 'D', list_cell.Cell.content.codepoint)
	}

	zigTry(t, s.Resize(screenResizeDefault(3, 3)))
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "1AB\nCD"
		expectEqualStrings(t, expected, contents)
	}
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		const expected = "1AB\nCD"
		expectEqualStrings(t, expected, contents)
	}

	// Cursor should be on the last line
	expectEqual[SizeCellCountInt](t, 1, s.Cursor.X)
	expectEqual[SizeCellCountInt](t, 1, s.Cursor.Y)
}

// Zig: test "Screen: resize less cols with reflow with trimmed rows"
func TestSc1_Screen_resize_less_cols_with_reflow_with_trimmed_rows(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 0)
	defer s.Deinit()
	const str = "3IJKL\n4ABCD\n5EFGH"
	sc1Write(t, s, str)
	zigTry(t, s.Resize(screenResizeDefault(3, 3)))

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "CD\n5EF\nGH"
		expectEqualStrings(t, expected, contents)
	}
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		const expected = "CD\n5EF\nGH"
		expectEqualStrings(t, expected, contents)
	}
}

// Zig: test "Screen: resize less cols with reflow with trimmed rows and scrollback"
func TestSc1_Screen_resize_less_cols_with_reflow_with_trimmed_rows_and_scrollback(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 1)
	defer s.Deinit()
	const str = "3IJKL\n4ABCD\n5EFGH"
	sc1Write(t, s, str)
	zigTry(t, s.Resize(screenResizeDefault(3, 3)))

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "CD\n5EF\nGH"
		expectEqualStrings(t, expected, contents)
	}
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		const expected = "3IJ\nKL\n4AB\nCD\n5EF\nGH"
		expectEqualStrings(t, expected, contents)
	}
}

// Zig: test "Screen: resize less cols with reflow previously wrapped"
func TestSc1_Screen_resize_less_cols_with_reflow_previously_wrapped(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 0)
	defer s.Deinit()
	const str = "3IJKL4ABCD5EFGH"
	sc1Write(t, s, str)

	// Check
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		const expected = "3IJKL\n4ABCD\n5EFGH"
		expectEqualStrings(t, expected, contents)
	}

	zigTry(t, s.Resize(screenResizeDefault(3, 3)))

	// {
	//     const contents = try s.testString(alloc, .viewport);
	//     defer alloc.free(contents);
	//     const expected = "CD\n5EF\nGH";
	//     try testing.expectEqualStrings(expected, contents);
	// }
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		const expected = "ABC\nD5E\nFGH"
		expectEqualStrings(t, expected, contents)
	}
}

// Zig: test "Screen: resize less cols with reflow and scrollback"
func TestSc1_Screen_resize_less_cols_with_reflow_and_scrollback(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 5)
	defer s.Deinit()
	const str = "1A\n2B\n3C\n4D\n5E"
	sc1Write(t, s, str)

	// Put our cursor on the end
	s.CursorAbsolute(1, s.Pages.rows-1)
	{
		list_cell := sc1CursorCell(t, s)
		expectEqual[uint32](t, 'E', list_cell.Cell.content.codepoint)
	}

	zigTry(t, s.Resize(screenResizeDefault(3, 3)))

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "3C\n4D\n5E"
		expectEqualStrings(t, expected, contents)
	}

	// Cursor should be on the last line
	expectEqual[SizeCellCountInt](t, 1, s.Cursor.X)
	expectEqual[SizeCellCountInt](t, 2, s.Cursor.Y)
}

// Zig: test "Screen: resize less cols with reflow previously wrapped and scrollback"
func TestSc1_Screen_resize_less_cols_with_reflow_previously_wrapped_and_scrollback(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 2)
	defer s.Deinit()
	const str = "1ABCD2EFGH3IJKL4ABCD5EFGH"
	sc1Write(t, s, str)

	// Check
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "3IJKL\n4ABCD\n5EFGH"
		expectEqualStrings(t, expected, contents)
	}

	// Put our cursor on the end
	s.CursorAbsolute(s.Pages.cols-1, s.Pages.rows-1)
	{
		list_cell := sc1CursorCell(t, s)
		expectEqual[uint32](t, 'H', list_cell.Cell.content.codepoint)
	}

	zigTry(t, s.Resize(screenResizeDefault(3, 3)))

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "CD5\nEFG\nH"
		expectEqualStrings(t, expected, contents)
	}
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		const expected = "1AB\nCD2\nEFG\nH3I\nJKL\n4AB\nCD5\nEFG\nH"
		expectEqualStrings(t, expected, contents)
	}

	// Cursor should be on the last line
	expectEqual[SizeCellCountInt](t, 0, s.Cursor.X)
	expectEqual[SizeCellCountInt](t, 2, s.Cursor.Y)
	{
		list_cell := sc1CursorCell(t, s)
		expectEqual[uint32](t, 'H', list_cell.Cell.content.codepoint)
	}
}

// Zig: test "Screen: resize less cols with scrollback keeps cursor row"
func TestSc1_Screen_resize_less_cols_with_scrollback_keeps_cursor_row(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 5)
	defer s.Deinit()
	const str = "1A\n2B\n3C\n4D\n5E"
	sc1Write(t, s, str)

	// Lets do a scroll and clear operation
	zigTry(t, s.ScrollClear())

	// Move our cursor to the beginning
	s.CursorAbsolute(0, 0)

	zigTry(t, s.Resize(screenResizeDefault(3, 3)))

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = ""
		expectEqualStrings(t, expected, contents)
	}

	// Cursor should be on the last line
	expectEqual[SizeCellCountInt](t, 0, s.Cursor.X)
	expectEqual[SizeCellCountInt](t, 0, s.Cursor.Y)
}

// Zig: test "Screen: resize more rows, less cols with reflow with scrollback"
func TestSc1_Screen_resize_more_rows_less_cols_with_reflow_with_scrollback(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 3)
	defer s.Deinit()
	const str = "1ABCD\n2EFGH3IJKL\n4MNOP"
	sc1Write(t, s, str)

	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		const expected = "1ABCD\n2EFGH\n3IJKL\n4MNOP"
		expectEqualStrings(t, expected, contents)
	}
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "2EFGH\n3IJKL\n4MNOP"
		expectEqualStrings(t, expected, contents)
	}

	zigTry(t, s.Resize(screenResizeDefault(2, 10)))

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "BC\nD\n2E\nFG\nH3\nIJ\nKL\n4M\nNO\nP"
		expectEqualStrings(t, expected, contents)
	}
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		const expected = "1A\nBC\nD\n2E\nFG\nH3\nIJ\nKL\n4M\nNO\nP"
		expectEqualStrings(t, expected, contents)
	}
}

// This seems like it should work fine but for some reason in practice
// in the initial implementation I found this bug! This is a regression
// test for that.
//
// Zig: test "Screen: resize more rows then shrink again"
func TestSc1_Screen_resize_more_rows_then_shrink_again(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 10)
	defer s.Deinit()
	const str = "1ABC"
	sc1Write(t, s, str)

	// Grow
	zigTry(t, s.Resize(screenResizeDefault(5, 10)))
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, str, contents)
	}
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, str, contents)
	}

	// Shrink
	zigTry(t, s.Resize(screenResizeDefault(5, 3)))
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, str, contents)
	}
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, str, contents)
	}

	// Grow again
	zigTry(t, s.Resize(screenResizeDefault(5, 10)))
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		expectEqualStrings(t, str, contents)
	}
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, str, contents)
	}
}

// Zig: test "Screen: resize less cols to eliminate wide char"
func TestSc1_Screen_resize_less_cols_to_eliminate_wide_char(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 2, 1, 0)
	defer s.Deinit()
	const str = "😀"
	sc1Write(t, s, str)
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, str, contents)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideWide, cell.wide)
		expectEqual[uint32](t, '😀', cell.content.codepoint)
	}

	// Resize to 1 column can't fit a wide char. So it should be deleted.
	zigTry(t, s.Resize(screenResizeDefault(1, 1)))
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "", contents)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual[uint32](t, 0, cell.content.codepoint)
		expectEqual(t, PageCellWideNarrow, cell.wide)
	}
}

// Zig: test "Screen: resize less cols to wrap wide char"
func TestSc1_Screen_resize_less_cols_to_wrap_wide_char(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 3, 3, 0)
	defer s.Deinit()
	const str = "x😀"
	sc1Write(t, s, str)
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, str, contents)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(1, 0)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideWide, cell.wide)
		expectEqual[uint32](t, '😀', cell.content.codepoint)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(2, 0)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideSpacerTail, cell.wide)
	}

	zigTry(t, s.Resize(screenResizeDefault(2, 3)))
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "x\n😀", contents)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(1, 0)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideSpacerHead, cell.wide)
		expect(t, list_cell.Row.wrap, "list_cell.row.wrap")
	}
}

// Zig: test "Screen: resize less cols to eliminate wide char with row space"
func TestSc1_Screen_resize_less_cols_to_eliminate_wide_char_with_row_space(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 2, 2, 0)
	defer s.Deinit()
	const str = "😀"
	sc1Write(t, s, str)
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, str, contents)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideWide, cell.wide)
		expectEqual[uint32](t, '😀', cell.content.codepoint)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(1, 0)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideSpacerTail, cell.wide)
	}

	zigTry(t, s.Resize(screenResizeDefault(1, 2)))
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "", contents)
	}
}

// Zig: test "Screen: resize more cols with wide spacer head"
func TestSc1_Screen_resize_more_cols_with_wide_spacer_head(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 3, 2, 0)
	defer s.Deinit()
	const str = "  😀"
	sc1Write(t, s, str)
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "  \n😀", contents)
	}

	// So this is the key point: we end up with a wide spacer head at
	// the end of row 1, then the emoji, then a wide spacer tail on row 2.
	// We should expect that if we resize to more cols, the wide spacer
	// head is replaced with the emoji.
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(2, 0)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideSpacerHead, cell.wide)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(0, 1)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideWide, cell.wide)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(1, 1)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideSpacerTail, cell.wide)
	}

	zigTry(t, s.Resize(screenResizeDefault(4, 2)))
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, str, contents)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(2, 0)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideWide, cell.wide)
		expectEqual[uint32](t, '😀', cell.content.codepoint)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(3, 0)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideSpacerTail, cell.wide)
	}
}

// Zig: test "Screen: resize more cols with wide spacer head multiple lines"
func TestSc1_Screen_resize_more_cols_with_wide_spacer_head_multiple_lines(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 3, 3, 0)
	defer s.Deinit()
	const str = "xxxyy😀"
	sc1Write(t, s, str)
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "xxx\nyy\n😀", contents)
	}

	// Similar to the "wide spacer head" test, but this time we'er going
	// to increase our columns such that multiple rows are unwrapped.
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(2, 1)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideSpacerHead, cell.wide)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(0, 2)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideWide, cell.wide)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(1, 2)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideSpacerTail, cell.wide)
	}

	zigTry(t, s.Resize(screenResizeDefault(8, 2)))
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, str, contents)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(5, 0)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideWide, cell.wide)
		expectEqual[uint32](t, '😀', cell.content.codepoint)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(6, 0)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideSpacerTail, cell.wide)
	}
}

// Zig: test "Screen: resize more cols requiring a wide spacer head"
func TestSc1_Screen_resize_more_cols_requiring_a_wide_spacer_head(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 2, 2, 0)
	defer s.Deinit()
	const str = "xx😀"
	sc1Write(t, s, str)
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "xx\n😀", contents)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(0, 1)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideWide, cell.wide)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(1, 1)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideSpacerTail, cell.wide)
	}

	// This resizes to 3 columns, which isn't enough space for our wide
	// char to enter row 1. But we need to mark the wide spacer head on the
	// end of the first row since we're wrapping to the next row.
	zigTry(t, s.Resize(screenResizeDefault(3, 2)))
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "xx\n😀", contents)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(2, 0)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideSpacerHead, cell.wide)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(0, 1)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideWide, cell.wide)
		expectEqual[uint32](t, '😀', cell.content.codepoint)
	}
	{
		list_cell := zigUnwrap(t, s.Pages.GetCell(sc1Screen(1, 1)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideSpacerTail, cell.wide)
	}
}

// Zig: test "Screen: resize more cols with cursor at prompt"
func TestSc1_Screen_resize_more_cols_with_cursor_at_prompt(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 5)
	defer s.Deinit()

	// zig fmt: off
	sc1Write(t, s, "ABCDE\n")
	s.CursorSetSemanticContent(sc1SemPrompt(OSCParsersSemanticPromptPromptKindInitial))
	sc1Write(t, s, "> ")
	s.CursorSetSemanticContent(sc1SemInput(ScreenCursorSetSemanticContentInputClearEol))
	sc1Write(t, s, "echo")
	// zig fmt: on

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "ABCDE\n> echo"
		expectEqualStrings(t, expected, contents)
	}

	zigTry(t, s.Resize(ScreenResize{
		Cols:         20,
		Rows:         3,
		Reflow:       true,
		PromptRedraw: OSCParsersSemanticPromptRedrawTrue,
	}))

	// Cursor should not move
	expectEqual[SizeCellCountInt](t, 6, s.Cursor.X)
	expectEqual[SizeCellCountInt](t, 1, s.Cursor.Y)

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "ABCDE"
		expectEqualStrings(t, expected, contents)
	}
}

// Zig: test "Screen: resize more cols with cursor not at prompt"
func TestSc1_Screen_resize_more_cols_with_cursor_not_at_prompt(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 5)
	defer s.Deinit()

	// zig fmt: off
	sc1Write(t, s, "ABCDE\n")
	s.CursorSetSemanticContent(sc1SemPrompt(OSCParsersSemanticPromptPromptKindInitial))
	sc1Write(t, s, "> ")
	s.CursorSetSemanticContent(sc1SemInput(ScreenCursorSetSemanticContentInputClearEol))
	sc1Write(t, s, "echo\n")
	sc1Write(t, s, "output")
	// zig fmt: on

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "ABCDE\n> echo\noutput"
		expectEqualStrings(t, expected, contents)
	}

	zigTry(t, s.Resize(ScreenResize{
		Cols:         20,
		Rows:         3,
		Reflow:       true,
		PromptRedraw: OSCParsersSemanticPromptRedrawTrue,
	}))

	// Cursor should not move
	expectEqual[SizeCellCountInt](t, 6, s.Cursor.X)
	expectEqual[SizeCellCountInt](t, 2, s.Cursor.Y)

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "ABCDE\n> echo\noutput"
		expectEqualStrings(t, expected, contents)
	}
}

// Zig: test "Screen: resize with prompt_redraw last clears only one line"
func TestSc1_Screen_resize_with_prompt_redraw_last_clears_only_one_line(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 4, 5)
	defer s.Deinit()

	// zig fmt: off
	sc1Write(t, s, "ABCDE\n")
	s.CursorSetSemanticContent(sc1SemPrompt(OSCParsersSemanticPromptPromptKindInitial))
	sc1Write(t, s, "> ")
	s.CursorSetSemanticContent(sc1SemInput(ScreenCursorSetSemanticContentInputClearExplicit))
	sc1Write(t, s, "hello\n")
	sc1Write(t, s, "world")
	// zig fmt: on

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "ABCDE\n> hello\nworld"
		expectEqualStrings(t, expected, contents)
	}

	// Cursor is at end of "world" line with semantic_content = .input
	zigTry(t, s.Resize(ScreenResize{
		Cols:         20,
		Rows:         4,
		Reflow:       true,
		PromptRedraw: OSCParsersSemanticPromptRedrawLast,
	}))

	// With .last, only the current line where cursor is should be cleared
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "ABCDE\n> hello"
		expectEqualStrings(t, expected, contents)
	}
}

// Zig: test "Screen: resize with prompt_redraw last multiline prompt clears only last line"
func TestSc1_Screen_resize_with_prompt_redraw_last_multiline_prompt_clears_only_last_line(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 20, 5, 5)
	defer s.Deinit()

	// Create a 3-line prompt: 1 initial + 2 continuation lines
	// zig fmt: off
	s.CursorSetSemanticContent(sc1SemPrompt(OSCParsersSemanticPromptPromptKindInitial))
	sc1Write(t, s, "line1\n")
	s.CursorSetSemanticContent(sc1SemPrompt(OSCParsersSemanticPromptPromptKindContinuation))
	sc1Write(t, s, "line2\n")
	s.CursorSetSemanticContent(sc1SemPrompt(OSCParsersSemanticPromptPromptKindContinuation))
	sc1Write(t, s, "line3")
	// zig fmt: on

	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "line1\nline2\nline3"
		expectEqualStrings(t, expected, contents)
	}

	// Cursor is at end of line3 (the last continuation line)
	zigTry(t, s.Resize(ScreenResize{
		Cols:         30,
		Rows:         5,
		Reflow:       true,
		PromptRedraw: OSCParsersSemanticPromptRedrawLast,
	}))

	// With .last, only line3 (where cursor is) should be cleared
	{
		contents := sc1Dump(t, s, sc1Viewport(0, 0))
		const expected = "line1\nline2"
		expectEqualStrings(t, expected, contents)
	}
}

// Zig: test "Screen: select untracked"
func TestSc1_Screen_select_untracked(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 10, 0)
	defer s.Deinit()
	sc1Write(t, s, "ABC  DEF\n 123\n456")

	expect(t, !s.Selection.Valid, "s.selection == null")
	tracked := s.Pages.CountTrackedPins()
	sc1Select(t, s, SelectionInit(
		sc1Pin(t, s, sc1Active(0, 0)),
		sc1Pin(t, s, sc1Active(3, 0)),
		false,
	))
	expectEqual(t, tracked+2, s.Pages.CountTrackedPins())
	zigTry(t, s.Select(None[Selection]()))
	expectEqual(t, tracked, s.Pages.CountTrackedPins())
}

// Zig: test "Screen: selectAll"
func TestSc1_Screen_selectAll(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 10, 0)
	defer s.Deinit()

	{
		sc1Write(t, s, "ABC  DEF\n 123\n456")
		sel := zigUnwrap(t, s.SelectAll())
		defer sel.Deinit(s)
		expectEqualPoint(t, sc1Screen(0, 0), sc1PointFromPin(t, s, PointTagScreen, sel.Start()))
		expectEqualPoint(t, sc1Screen(2, 2), sc1PointFromPin(t, s, PointTagScreen, sel.End()))
	}

	{
		sc1Write(t, s, "\nFOO\n BAR\n BAZ\n QWERTY\n 12345678")
		sel := zigUnwrap(t, s.SelectAll())
		defer sel.Deinit(s)
		expectEqualPoint(t, sc1Screen(0, 0), sc1PointFromPin(t, s, PointTagScreen, sel.Start()))
		expectEqualPoint(t, sc1Screen(8, 7), sc1PointFromPin(t, s, PointTagScreen, sel.End()))
	}
}

// Zig: test "Screen: selectLine"
func TestSc1_Screen_selectLine(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 10, 0)
	defer s.Deinit()
	sc1Write(t, s, "ABC  DEF\n 123\n456")

	// Outside of active area
	// try testing.expect(s.selectLine(.{ .x = 13, .y = 0 }) == null);
	// try testing.expect(s.selectLine(.{ .x = 0, .y = 5 }) == null);

	// Going forward
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(0, 0)))))
		defer sel.Deinit(s)
		expectEqualPoint(t, sc1Screen(0, 0), sc1PointFromPin(t, s, PointTagScreen, sel.Start()))
		expectEqualPoint(t, sc1Screen(7, 0), sc1PointFromPin(t, s, PointTagScreen, sel.End()))
	}

	// Going backward
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(7, 0)))))
		defer sel.Deinit(s)
		expectEqualPoint(t, sc1Screen(0, 0), sc1PointFromPin(t, s, PointTagScreen, sel.Start()))
		expectEqualPoint(t, sc1Screen(7, 0), sc1PointFromPin(t, s, PointTagScreen, sel.End()))
	}

	// Going forward and backward
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(3, 0)))))
		defer sel.Deinit(s)
		expectEqualPoint(t, sc1Screen(0, 0), sc1PointFromPin(t, s, PointTagScreen, sel.Start()))
		expectEqualPoint(t, sc1Screen(7, 0), sc1PointFromPin(t, s, PointTagScreen, sel.End()))
	}

	// Outside active area
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(9, 0)))))
		defer sel.Deinit(s)
		expectEqualPoint(t, sc1Screen(0, 0), sc1PointFromPin(t, s, PointTagScreen, sel.Start()))
		expectEqualPoint(t, sc1Screen(7, 0), sc1PointFromPin(t, s, PointTagScreen, sel.End()))
	}
}

// Zig: test "Screen: selectLine across soft-wrap"
func TestSc1_Screen_selectLine_across_soft_wrap(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 10, 0)
	defer s.Deinit()
	sc1Write(t, s, " 12 34012   \n 123")

	// Going forward
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(1, 0)))))
		defer sel.Deinit(s)
		expectEqualPoint(t, sc1Screen(1, 0), sc1PointFromPin(t, s, PointTagScreen, sel.Start()))
		expectEqualPoint(t, sc1Screen(3, 1), sc1PointFromPin(t, s, PointTagScreen, sel.End()))
	}
}

// Zig: test "Screen: selectLine across full soft-wrap"
func TestSc1_Screen_selectLine_across_full_soft_wrap(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 5, 0)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD2EFGH\n3IJKL")

	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(2, 1)))))
		defer sel.Deinit(s)
		expectEqualPoint(t, sc1Screen(0, 0), sc1PointFromPin(t, s, PointTagScreen, sel.Start()))
		expectEqualPoint(t, sc1Screen(4, 1), sc1PointFromPin(t, s, PointTagScreen, sel.End()))
	}
}

// Zig: test "Screen: selectLine across soft-wrap ignores blank lines"
func TestSc1_Screen_selectLine_across_soft_wrap_ignores_blank_lines(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 10, 0)
	defer s.Deinit()
	sc1Write(t, s, " 12 34012             \n 123")

	// Going forward
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(1, 0)))))
		defer sel.Deinit(s)
		expectEqualPoint(t, sc1Screen(1, 0), sc1PointFromPin(t, s, PointTagScreen, sel.Start()))
		expectEqualPoint(t, sc1Screen(3, 1), sc1PointFromPin(t, s, PointTagScreen, sel.End()))
	}

	// Going backward
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(1, 1)))))
		defer sel.Deinit(s)
		expectEqualPoint(t, sc1Screen(1, 0), sc1PointFromPin(t, s, PointTagScreen, sel.Start()))
		expectEqualPoint(t, sc1Screen(3, 1), sc1PointFromPin(t, s, PointTagScreen, sel.End()))
	}

	// Going forward and backward
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(3, 0)))))
		defer sel.Deinit(s)
		expectEqualPoint(t, sc1Screen(1, 0), sc1PointFromPin(t, s, PointTagScreen, sel.Start()))
		expectEqualPoint(t, sc1Screen(3, 1), sc1PointFromPin(t, s, PointTagScreen, sel.End()))
	}
}

// Zig: test "Screen: selectLine disabled whitespace trimming"
func TestSc1_Screen_selectLine_disabled_whitespace_trimming(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 10, 0)
	defer s.Deinit()
	sc1Write(t, s, " 12 34012   \n 123")

	// Going forward
	{
		opts := screenSelectLineDefault(sc1Pin(t, s, sc1Active(1, 0)))
		opts.Whitespace = None[[]uint32]()
		sel := zigUnwrap(t, s.SelectLine(opts))
		defer sel.Deinit(s)
		expectEqualPoint(t, sc1Screen(0, 0), sc1PointFromPin(t, s, PointTagScreen, sel.Start()))
		expectEqualPoint(t, sc1Screen(4, 2), sc1PointFromPin(t, s, PointTagScreen, sel.End()))
	}

	// Non-wrapped
	{
		opts := screenSelectLineDefault(sc1Pin(t, s, sc1Active(1, 3)))
		opts.Whitespace = None[[]uint32]()
		sel := zigUnwrap(t, s.SelectLine(opts))
		defer sel.Deinit(s)
		expectEqualPoint(t, sc1Screen(0, 3), sc1PointFromPin(t, s, PointTagScreen, sel.Start()))
		expectEqualPoint(t, sc1Screen(4, 3), sc1PointFromPin(t, s, PointTagScreen, sel.End()))
	}
}

// Zig: test "Screen: selectLine with scrollback"
func TestSc1_Screen_selectLine_with_scrollback(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 2, 3, 5)
	defer s.Deinit()
	sc1Write(t, s, "1A\n2B\n3C\n4D\n5E")

	// Selecting first line
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(0, 0)))))
		defer sel.Deinit(s)
		expectEqualPoint(t, sc1Active(0, 0), sc1PointFromPin(t, s, PointTagActive, sel.Start()))
		expectEqualPoint(t, sc1Active(1, 0), sc1PointFromPin(t, s, PointTagActive, sel.End()))
	}

	// Selecting last line
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(0, 2)))))
		defer sel.Deinit(s)
		expectEqualPoint(t, sc1Active(0, 2), sc1PointFromPin(t, s, PointTagActive, sel.Start()))
		expectEqualPoint(t, sc1Active(1, 2), sc1PointFromPin(t, s, PointTagActive, sel.End()))
	}
}

// https://github.com/mitchellh/ghostty/issues/1329
//
// Zig: test "Screen: selectLine semantic prompt boundary"
func TestSc1_Screen_selectLine_semantic_prompt_boundary(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 10, 0)
	defer s.Deinit()
	sc1Write(t, s, "ABCDE\n")
	s.CursorSetSemanticContent(sc1SemPrompt(OSCParsersSemanticPromptPromptKindInitial))
	sc1Write(t, s, "A    ")
	s.CursorSetSemanticContent(sc1SemOutput())
	sc1Write(t, s, "> ")

	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "ABCDE\nA    \n> ", contents)
	}

	// Selecting output stops at the prompt even if soft-wrapped
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(1, 1)))))
		defer sel.Deinit(s)
		contents := sc1SelectionString(t, s, ScreenSelectionString{Sel: sel, Trim: false})
		const expected = "A"
		expectEqualStrings(t, expected, contents)
	}
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(1, 2)))))
		defer sel.Deinit(s)
		expectEqualPoint(t, sc1Active(0, 2), sc1PointFromPin(t, s, PointTagActive, sel.Start()))
		expectEqualPoint(t, sc1Active(0, 2), sc1PointFromPin(t, s, PointTagActive, sel.End()))
	}
}

// Zig: test "Screen: selectLine semantic prompt to input boundary"
func TestSc1_Screen_selectLine_semantic_prompt_to_input_boundary(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 5, 0)
	defer s.Deinit()

	// Write prompt followed by user input on same row: "$>command"
	// Using non-whitespace to avoid whitespace trimming affecting the test
	s.CursorSetSemanticContent(sc1SemPrompt(OSCParsersSemanticPromptPromptKindInitial))
	sc1Write(t, s, "$>")
	s.CursorSetSemanticContent(sc1SemInput(ScreenCursorSetSemanticContentInputClearExplicit))
	sc1Write(t, s, "command")

	// Selecting from prompt should only select prompt
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(0, 0)))))
		defer sel.Deinit(s)
		expectEqualPoint(t, sc1Active(0, 0), sc1PointFromPin(t, s, PointTagActive, sel.Start()))
		expectEqualPoint(t, sc1Active(1, 0), sc1PointFromPin(t, s, PointTagActive, sel.End()))
	}

	// Selecting from input should only select input
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(5, 0)))))
		defer sel.Deinit(s)
		expectEqualPoint(t, sc1Active(2, 0), sc1PointFromPin(t, s, PointTagActive, sel.Start()))
		expectEqualPoint(t, sc1Active(8, 0), sc1PointFromPin(t, s, PointTagActive, sel.End()))
	}
}

// Zig: test "Screen: selectLine semantic input to output boundary"
func TestSc1_Screen_selectLine_semantic_input_to_output_boundary(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 5, 0)
	defer s.Deinit()

	// Row 0: user input
	s.CursorSetSemanticContent(sc1SemInput(ScreenCursorSetSemanticContentInputClearExplicit))
	sc1Write(t, s, "ls -la\n")
	// Row 1: command output
	s.CursorSetSemanticContent(sc1SemOutput())
	sc1Write(t, s, "file.txt")

	// Selecting from input should only select input
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(2, 0)))))
		defer sel.Deinit(s)
		contents := sc1SelectionString(t, s, ScreenSelectionString{Sel: sel, Trim: false})
		expectEqualStrings(t, "ls -la", contents)
	}

	// Selecting from output should only select output
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(2, 1)))))
		defer sel.Deinit(s)
		contents := sc1SelectionString(t, s, ScreenSelectionString{Sel: sel, Trim: false})
		expectEqualStrings(t, "file.txt", contents)
	}
}

// Zig: test "Screen: selectLine semantic mid-row boundary"
func TestSc1_Screen_selectLine_semantic_mid_row_boundary(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 5, 0)
	defer s.Deinit()

	// Single row with output then prompt then input: "out$>cmd"
	// Using non-whitespace to avoid whitespace trimming affecting the test
	s.CursorSetSemanticContent(sc1SemOutput())
	sc1Write(t, s, "out")
	s.CursorSetSemanticContent(sc1SemPrompt(OSCParsersSemanticPromptPromptKindInitial))
	sc1Write(t, s, "$>")
	s.CursorSetSemanticContent(sc1SemInput(ScreenCursorSetSemanticContentInputClearExplicit))
	sc1Write(t, s, "cmd")

	// Selecting from output should stop at prompt
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(1, 0)))))
		defer sel.Deinit(s)
		expectEqualPoint(t, sc1Active(0, 0), sc1PointFromPin(t, s, PointTagActive, sel.Start()))
		expectEqualPoint(t, sc1Active(2, 0), sc1PointFromPin(t, s, PointTagActive, sel.End()))
	}

	// Selecting from prompt should only select prompt
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(3, 0)))))
		defer sel.Deinit(s)
		expectEqualPoint(t, sc1Active(3, 0), sc1PointFromPin(t, s, PointTagActive, sel.Start()))
		expectEqualPoint(t, sc1Active(4, 0), sc1PointFromPin(t, s, PointTagActive, sel.End()))
	}

	// Selecting from input should only select input
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(6, 0)))))
		defer sel.Deinit(s)
		expectEqualPoint(t, sc1Active(5, 0), sc1PointFromPin(t, s, PointTagActive, sel.Start()))
		expectEqualPoint(t, sc1Active(7, 0), sc1PointFromPin(t, s, PointTagActive, sel.End()))
	}
}

// Zig: test "Screen: selectLine semantic boundary soft-wrap with mid-row transition"
func TestSc1_Screen_selectLine_semantic_boundary_soft_wrap_with_mid_row_transition(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 5, 0)
	defer s.Deinit()

	// Row 0: prompt "$ " + input "cmd" (soft-wraps)
	// Row 1: input continues "12" + output "out"
	s.CursorSetSemanticContent(sc1SemPrompt(OSCParsersSemanticPromptPromptKindInitial))
	sc1Write(t, s, "$ ")
	s.CursorSetSemanticContent(sc1SemInput(ScreenCursorSetSemanticContentInputClearExplicit))
	sc1Write(t, s, "cmd12")
	s.CursorSetSemanticContent(sc1SemOutput())
	sc1Write(t, s, "out")

	// Verify layout
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "$ cmd\n12out", contents)
	}

	// Selecting from input on row 0 should get all input across soft-wrap
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(3, 0)))))
		defer sel.Deinit(s)
		contents := sc1SelectionString(t, s, ScreenSelectionString{Sel: sel, Trim: false})
		expectEqualStrings(t, "cmd12", contents)
	}

	// Selecting from input on row 1 should get all input across soft-wrap
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(0, 1)))))
		defer sel.Deinit(s)
		contents := sc1SelectionString(t, s, ScreenSelectionString{Sel: sel, Trim: false})
		expectEqualStrings(t, "cmd12", contents)
	}

	// Selecting from output should only get output
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(3, 1)))))
		defer sel.Deinit(s)
		contents := sc1SelectionString(t, s, ScreenSelectionString{Sel: sel, Trim: false})
		expectEqualStrings(t, "out", contents)
	}
}

// Zig: test "Screen: selectLine semantic boundary disabled"
func TestSc1_Screen_selectLine_semantic_boundary_disabled(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 5, 0)
	defer s.Deinit()

	// Write prompt followed by input
	s.CursorSetSemanticContent(sc1SemPrompt(OSCParsersSemanticPromptPromptKindInitial))
	sc1Write(t, s, "$ ")
	s.CursorSetSemanticContent(sc1SemInput(ScreenCursorSetSemanticContentInputClearExplicit))
	sc1Write(t, s, "command")

	// With semantic_prompt_boundary = false, should select entire line
	{
		opts := screenSelectLineDefault(sc1Pin(t, s, sc1Active(0, 0)))
		opts.SemanticPromptBoundary = false
		sel := zigUnwrap(t, s.SelectLine(opts))
		defer sel.Deinit(s)
		contents := sc1SelectionString(t, s, ScreenSelectionString{Sel: sel, Trim: false})
		expectEqualStrings(t, "$ command", contents)
	}
}

// Zig: test "Screen: selectLine semantic boundary first cell of row"
func TestSc1_Screen_selectLine_semantic_boundary_first_cell_of_row(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 5, 0)
	defer s.Deinit()

	// Row 0: input that soft-wraps
	// Row 1: output starts at first cell
	s.CursorSetSemanticContent(sc1SemInput(ScreenCursorSetSemanticContentInputClearExplicit))
	sc1Write(t, s, "12345")
	s.CursorSetSemanticContent(sc1SemOutput())
	sc1Write(t, s, "ABCDE")

	// Verify soft-wrap happened
	{
		pin := sc1Pin(t, s, sc1Active(0, 0))
		row := pin.RowAndCell().row
		expect(t, row.wrap, "row.wrap")
	}

	// Selecting from input should stop before output on row 1
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(2, 0)))))
		defer sel.Deinit(s)
		expectEqualPoint(t, sc1Active(0, 0), sc1PointFromPin(t, s, PointTagActive, sel.Start()))
		expectEqualPoint(t, sc1Active(4, 0), sc1PointFromPin(t, s, PointTagActive, sel.End()))
	}

	// Selecting from output should only get output
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(2, 1)))))
		defer sel.Deinit(s)
		expectEqualPoint(t, sc1Active(0, 1), sc1PointFromPin(t, s, PointTagActive, sel.Start()))
		expectEqualPoint(t, sc1Active(4, 1), sc1PointFromPin(t, s, PointTagActive, sel.End()))
	}
}

// Zig: test "Screen: selectLine semantic all same content"
func TestSc1_Screen_selectLine_semantic_all_same_content(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 5, 0)
	defer s.Deinit()

	// All prompt content that soft-wraps
	s.CursorSetSemanticContent(sc1SemPrompt(OSCParsersSemanticPromptPromptKindInitial))
	sc1Write(t, s, "prompt text")

	// Verify soft-wrap
	{
		contents := sc1Dump(t, s, sc1Screen(0, 0))
		expectEqualStrings(t, "promp\nt tex\nt", contents)
	}

	// Should select all prompt content across soft-wraps
	{
		sel := zigUnwrap(t, s.SelectLine(screenSelectLineDefault(sc1Pin(t, s, sc1Active(2, 1)))))
		defer sel.Deinit(s)
		contents := sc1SelectionString(t, s, ScreenSelectionString{Sel: sel, Trim: false})
		expectEqualStrings(t, "prompt text", contents)
	}
}

// ---------------------------------------------------------------------------
// Go-only panic tests: assert / unreachable paths of the functions under test
// that the Zig tests avoid (a Zig test cannot test a path that aborts).
// ---------------------------------------------------------------------------

// Go-only: panic test: Screen.cursorAbsolute: x >= cols
func TestSc1_panic_cursorAbsolute_x_out_of_range(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 0)
	defer s.Deinit()
	sc1ExpectPanic(t, func() { s.CursorAbsolute(10, 0) })
}

// Go-only: panic test: Screen.cursorAbsolute: y >= rows
func TestSc1_panic_cursorAbsolute_y_out_of_range(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 0)
	defer s.Deinit()
	sc1ExpectPanic(t, func() { s.CursorAbsolute(0, 3) })
}

// Go-only: panic test: Screen.cursorUp: n > cursor.y
func TestSc1_panic_cursorUp_above_top(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 0)
	defer s.Deinit()
	s.CursorAbsolute(0, 1)
	sc1ExpectPanic(t, func() { s.CursorUp(2) })
}

// Go-only: panic test: Screen.cursorDown: cursor.y + n >= rows
func TestSc1_panic_cursorDown_below_bottom(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 0)
	defer s.Deinit()
	s.CursorAbsolute(0, 1)
	sc1ExpectPanic(t, func() { s.CursorDown(2) })
}

// Go-only: panic test: Screen.cursorDownScroll: cursor is not on the last row
func TestSc1_panic_cursorDownScroll_cursor_not_on_last_row(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 0)
	defer s.Deinit()
	sc1ExpectPanic(t, func() { _ = s.CursorDownScroll() })
}

// Go-only: panic test: Screen.cursorCopy: other.x >= cols
func TestSc1_panic_cursorCopy_x_out_of_range(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 10, 0)
	defer s.Deinit()
	s2 := sc1Init(t, 5, 10, 0)
	defer s2.Deinit()
	other := s.Cursor
	other.X = 7
	sc1ExpectPanic(t, func() { _ = s2.CursorCopy(other, screenCursorCopyOptionsDefault()) })
}

// Go-only: panic test: Screen.cursorCopy: other.y >= rows
func TestSc1_panic_cursorCopy_y_out_of_range(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 10, 0)
	defer s.Deinit()
	s2 := sc1Init(t, 10, 5, 0)
	defer s2.Deinit()
	other := s.Cursor
	other.Y = 7
	sc1ExpectPanic(t, func() { _ = s2.CursorCopy(other, screenCursorCopyOptionsDefault()) })
}

// Go-only: panic test: Screen.clone: the selection is not tracked
func TestSc1_panic_clone_untracked_selection(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 5, 3, 1)
	defer s.Deinit()
	sc1Write(t, s, "1ABCD\n2EFGH\n3IJKL")

	// Set the selection directly (not through select) so that it stays
	// untracked.
	s.Selection = Some(SelectionInit(
		sc1Pin(t, s, sc1Active(0, 1)),
		sc1Pin(t, s, sc1Active(s.Pages.cols-1, 1)),
		false,
	))
	sc1ExpectPanic(t, func() { _, _ = s.Clone(sc1Active(0, 0), None[Point]()) })
}

// Go-only: panic test: Screen.cursorSetHyperlink: no hyperlink is active
func TestSc1_panic_cursorSetHyperlink_no_active_hyperlink(t *testing.T) {
	testStdCapacity(t)

	s := sc1Init(t, 10, 3, 0)
	defer s.Deinit()
	sc1ExpectPanic(t, func() { _ = s.CursorSetHyperlink() })
}
