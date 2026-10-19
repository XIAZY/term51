// Tests translated one for one from src/terminal/Terminal.zig (Ghostty
// v1.3.1 plus the term51 additional tests), lines 7156 to 10446 of the
// patched file: from `Terminal: scrollDown outside of left/right scroll
// region` to `Terminal: deleteChars split wide character from wide` (the
// range holds no `term51:` test). Each Go test carries the exact Zig title
// in a "// Zig: test" comment. Helpers are prefixed tm2.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualStrings keep Zig's names and Zig's
//     (expected, actual) argument order. `try` on a call that returns an
//     error is zigTry(t, err); `x.?` is zigUnwrap.
//   - `var t = try init(alloc, .{ ... })` is `term := tm2Init(t, opts)` with
//     `opts` starting from terminalOptionsDefault (max_scrollback 10_000,
//     default colors and modes); the Terminal is held by pointer, `t` is the
//     *testing.T. Every test starts with testStdCapacity(t).
//   - `try t.plainString(alloc)`, `plainStringUnwrapped` and
//     `screens.active.dumpStringAlloc(alloc, ...)` are tm2PlainString,
//     tm2PlainStringUnwrapped and tm2DumpStringAlloc, which return a Go
//     string. `testing.allocator` leak checks (`const alloc =
//     testing.allocator`, `defer testing.allocator.free(str)`) are dropped
//     with `// Zig-only:` comments.
//   - `@as(usize, n)` compared with cursor.x / cursor.y is
//     SizeCellCountInt(n) (the field type); compared with a count it is
//     uint(n). `@as(u21, c)` is uint32(c); `Cell.RGB{...}` is PageCellRGB;
//     `cell.content.color_rgb` is the PageCellContent field of that name.
//   - `for ("ABC") |c| try t.print(c)` iterates the bytes of the string, as
//     Zig does: `for _, c := range []byte("ABC")`.
//   - `list_cell.node.data.lookupHyperlink(cell)` returns Optional[HyperlinkId]:
//     `== null` is `!id.Valid`. `page.styles.refCount(page.memory, id)` is
//     `page.styles.RefCount(id)` (no memory argument).
//   - Zig string escapes `\u{1F600}` are Go `\U0001F600`.
//   - Go-only panic tests are at the end of the file.

package terminal

import "testing"

// tm2Init is `try init(alloc, opts)`: the terminal is returned by pointer
// (its methods have pointer receivers).
func tm2Init(t *testing.T, opts TerminalOptions) *Terminal {
	t.Helper()
	term, err := TerminalInit(opts)
	zigTry(t, err)
	return &term
}

// tm2PlainString is `try t.plainString(alloc)`, as a Go string.
func tm2PlainString(t *testing.T, term *Terminal) string {
	t.Helper()
	str, err := term.PlainString()
	zigTry(t, err)
	return string(str)
}

// tm2PlainStringUnwrapped is `try t.plainStringUnwrapped(alloc)`, as a Go
// string.
func tm2PlainStringUnwrapped(t *testing.T, term *Terminal) string {
	t.Helper()
	str, err := term.PlainStringUnwrapped()
	zigTry(t, err)
	return string(str)
}

// tm2DumpStringAlloc is `try screen.dumpStringAlloc(alloc, tl)`, as a Go
// string.
func tm2DumpStringAlloc(t *testing.T, s *Screen, tl Point) string {
	t.Helper()
	str, err := s.DumpStringAlloc(tl)
	zigTry(t, err)
	return string(str)
}

// tm2Active, tm2Screen, tm2Viewport and tm2History are the Zig literals
// `.{ .active = .{ .x = x, .y = y } }` and so on.
func tm2Active(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagActive, Active: PointCoordinate{X: x, Y: y}}
}

func tm2Screen(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagScreen, Screen: PointCoordinate{X: x, Y: y}}
}

func tm2Viewport(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagViewport, Viewport: PointCoordinate{X: x, Y: y}}
}

func tm2History(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagHistory, History: PointCoordinate{X: x, Y: y}}
}

// tm2ExpectPanic runs f, which must panic. A panic with the wireframe's
// "not implemented" value does not count: it only means the function under
// test does not exist yet. The setup is done outside f.
func tm2ExpectPanic(t *testing.T, f func()) {
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

// Zig: test "Terminal: scrollDown outside of left/right scroll region"
func TestTm2_Terminal_scrollDown_outside_of_left_right_scroll_region(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(10, 10))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC123")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF456")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI789")))
	term.ScrollingRegion.Left = 1
	term.ScrollingRegion.Right = 3
	term.SetCursorPos(1, 1)

	cursor := term.Screens.Active.Cursor
	term.clearDirty()
	term.ScrollDown(1)
	expectEqual(t, cursor.X, term.Screens.Active.Cursor.X)
	expectEqual(t, cursor.Y, term.Screens.Active.Cursor.Y)

	for y := 0; y < 4; y++ {
		expect(t, term.isDirty(tm2Active(0, uint32(y))), "t.isDirty(.{ .active = .{ .x = 0, .y = @intCast(y) } })")
	}

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "A   23\nDBC156\nGEF489\n HI7", str)
	}
}

// Zig: test "Terminal: scrollDown preserves pending wrap"
func TestTm2_Terminal_scrollDown_preserves_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 10))
	defer term.Deinit()

	term.SetCursorPos(1, 5)
	zigTry(t, term.Print('A'))
	term.SetCursorPos(2, 5)
	zigTry(t, term.Print('B'))
	term.SetCursorPos(3, 5)
	zigTry(t, term.Print('C'))
	term.ScrollDown(1)
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\n    A\n    B\nX   C", str)
	}
}

// Zig: test "Terminal: eraseChars simple operation"
func TestTm2_Terminal_eraseChars_simple_operation(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	for _, c := range []byte("ABC") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.SetCursorPos(1, 1)
	term.clearDirty()
	term.EraseChars(2)
	zigTry(t, term.Print('X'))

	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, !term.isDirty(tm2Active(0, 1)), "!t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "X C", str)
	}
}

// Zig: test "Terminal: eraseChars minimum one"
func TestTm2_Terminal_eraseChars_minimum_one(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	for _, c := range []byte("ABC") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.SetCursorPos(1, 1)
	term.clearDirty()
	term.EraseChars(0)
	zigTry(t, term.Print('X'))
	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "XBC", str)
	}
}

// Zig: test "Terminal: eraseChars beyond screen edge"
func TestTm2_Terminal_eraseChars_beyond_screen_edge(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	for _, c := range []byte("  ABC") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.SetCursorPos(1, 4)
	term.EraseChars(10)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "  A", str)
	}
}

// Zig: test "Terminal: eraseChars wide character"
func TestTm2_Terminal_eraseChars_wide_character(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	zigTry(t, term.Print('橋'))
	for _, c := range []byte("BC") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.SetCursorPos(1, 1)
	term.EraseChars(1)
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "X BC", str)
	}
}

// Zig: test "Terminal: eraseChars resets pending wrap"
func TestTm2_Terminal_eraseChars_resets_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	for _, c := range []byte("ABCDE") {
		zigTry(t, term.Print(uint32(c)))
	}
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	term.EraseChars(1)
	expect(t, !term.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABCDX", str)
	}
}

// Zig: test "Terminal: eraseChars resets wrap"
func TestTm2_Terminal_eraseChars_resets_wrap(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	for _, c := range []byte("ABCDE123") {
		zigTry(t, term.Print(uint32(c)))
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Active(0, 0)))
		row := list_cell.Row
		expect(t, row.wrap, "row.wrap")
	}

	term.SetCursorPos(1, 1)
	term.EraseChars(1)

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Active(0, 0)))
		row := list_cell.Row
		expect(t, !row.wrap, "!row.wrap")
	}

	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "XBCDE\n123", str)
	}
}

// Zig: test "Terminal: eraseChars preserves background sgr"
func TestTm2_Terminal_eraseChars_preserves_background_sgr(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(10, 10))
	defer term.Deinit()

	for _, c := range []byte("ABC") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.SetCursorPos(1, 1)
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: 0xFF, G: 0, B: 0}}))
	term.EraseChars(2)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "  C", str)
		{
			list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Active(0, 0)))
			expect(t, list_cell.Cell.ContentTag() == PageCellContentTagBgColorRgb, "list_cell.cell.content_tag == .bg_color_rgb")
			expectEqual(t, PageCellRGB{R: 0xFF, G: 0, B: 0}, list_cell.Cell.content().colorRGB())
		}
		{
			list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Active(1, 0)))
			expect(t, list_cell.Cell.ContentTag() == PageCellContentTagBgColorRgb, "list_cell.cell.content_tag == .bg_color_rgb")
			expectEqual(t, PageCellRGB{R: 0xFF, G: 0, B: 0}, list_cell.Cell.content().colorRGB())
		}
	}
}

// Zig: test "Terminal: eraseChars handles refcounted styles"
func TestTm2_Terminal_eraseChars_handles_refcounted_styles(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(10, 10))
	defer term.Deinit()

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	zigTry(t, term.Print('A'))
	zigTry(t, term.Print('B'))
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))
	zigTry(t, term.Print('C'))

	// verify we have styles in our style map
	page := &term.Screens.Active.Cursor.PagePin.Node.data
	expectEqual(t, uint(1), page.styles.Count())

	term.SetCursorPos(1, 1)
	term.EraseChars(2)

	// verify we have no styles in our style map
	expectEqual(t, uint(0), page.styles.Count())
}

// Zig: test "Terminal: eraseChars protected attributes respected with iso"
func TestTm2_Terminal_eraseChars_protected_attributes_respected_with_iso(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.SetProtectedMode(ANSIProtectedModeIso)
	for _, c := range []byte("ABC") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.SetCursorPos(1, 1)
	term.EraseChars(2)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABC", str)
	}
}

// Zig: test "Terminal: eraseChars protected attributes ignored with dec most recent"
func TestTm2_Terminal_eraseChars_protected_attributes_ignored_with_dec_most_recent(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.SetProtectedMode(ANSIProtectedModeIso)
	for _, c := range []byte("ABC") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.SetProtectedMode(ANSIProtectedModeDec)
	term.SetProtectedMode(ANSIProtectedModeOff)
	term.SetCursorPos(1, 1)
	term.EraseChars(2)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "  C", str)
	}
}

// Zig: test "Terminal: eraseChars protected attributes ignored with dec set"
func TestTm2_Terminal_eraseChars_protected_attributes_ignored_with_dec_set(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.SetProtectedMode(ANSIProtectedModeDec)
	for _, c := range []byte("ABC") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.SetCursorPos(1, 1)
	term.EraseChars(2)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "  C", str)
	}
}

// Zig: test "Terminal: eraseChars wide char boundary conditions"
func TestTm2_Terminal_eraseChars_wide_char_boundary_conditions(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(8, 1))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("😀a😀b😀")))
	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "😀a😀b😀", str)
	}

	term.SetCursorPos(1, 2)
	term.EraseChars(3)
	term.Screens.Active.Cursor.PagePin.Node.data.AssertIntegrity()

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "     b😀", str)
	}
}

// Zig: test "Terminal: eraseChars wide char splits proper cell boundaries"
func TestTm2_Terminal_eraseChars_wide_char_splits_proper_cell_boundaries(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(30, 1))
	defer term.Deinit()

	// This is a test for a bug: https://github.com/ghostty-org/ghostty/issues/2817
	// To explain the setup:
	// (1) We need our wide characters starting on an even (1-based) column.
	// (2) We need our cursor to be in the middle somewhere.
	// (3) We need our count to be less than our cursor X and on a split cell.
	// The bug was that we split the wrong cell boundaries.

	zigTry(t, term.PrintString([]uint8("x食べて下さい")))
	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "x食べて下さい", str)
	}

	term.SetCursorPos(1, 6) // At: て
	term.EraseChars(4)      // Delete: て下
	term.Screens.Active.Cursor.PagePin.Node.data.AssertIntegrity()

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "x食べ    さい", str)
	}
}

// Zig: test "Terminal: eraseChars wide char wrap boundary conditions"
func TestTm2_Terminal_eraseChars_wide_char_wrap_boundary_conditions(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(8, 3))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8(".......😀abcde😀......")))
	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, ".......\n😀abcde\n😀......", str)

		unwrapped := tm2PlainStringUnwrapped(t, term)
		// Zig-only: defer testing.allocator.free(unwrapped);
		expectEqualStrings(t, ".......😀abcde😀......", unwrapped)
	}

	term.SetCursorPos(2, 2)
	term.EraseChars(3)
	term.Screens.Active.Cursor.PagePin.Node.data.AssertIntegrity()

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, ".......\n    cde\n😀......", str)

		unwrapped := tm2PlainStringUnwrapped(t, term)
		// Zig-only: defer testing.allocator.free(unwrapped);
		expectEqualStrings(t, ".......     cde\n😀......", unwrapped)
	}
}

// Zig: test "Terminal: reverseIndex"
func TestTm2_Terminal_reverseIndex(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(2, 5))
	defer term.Deinit()

	// Initial value
	zigTry(t, term.Print('A'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('B'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('C'))
	term.ReverseIndex()
	zigTry(t, term.Print('D'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	term.CarriageReturn()
	zigTry(t, term.Linefeed())

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "A\nBD\nC", str)
	}
}

// Zig: test "Terminal: reverseIndex from the top"
func TestTm2_Terminal_reverseIndex_from_the_top(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(2, 5))
	defer term.Deinit()

	zigTry(t, term.Print('A'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('B'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	term.CarriageReturn()
	zigTry(t, term.Linefeed())

	term.SetCursorPos(1, 1)
	term.ReverseIndex()
	zigTry(t, term.Print('D'))

	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	term.SetCursorPos(1, 1)
	term.ReverseIndex()
	zigTry(t, term.Print('E'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "E\nD\nA\nB", str)
	}
}

// Zig: test "Terminal: reverseIndex top of scrolling region"
func TestTm2_Terminal_reverseIndex_top_of_scrolling_region(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(2, 10))
	defer term.Deinit()

	// Initial value
	term.SetCursorPos(2, 1)
	zigTry(t, term.Print('A'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('B'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('C'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('D'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())

	// Set our scroll region
	term.SetTopAndBottomMargin(2, 5)
	term.SetCursorPos(2, 1)
	term.ReverseIndex()
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\nX\nA\nB\nC", str)
	}
}

// Zig: test "Terminal: reverseIndex top of screen"
func TestTm2_Terminal_reverseIndex_top_of_screen(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	zigTry(t, term.Print('A'))
	term.SetCursorPos(2, 1)
	zigTry(t, term.Print('B'))
	term.SetCursorPos(3, 1)
	zigTry(t, term.Print('C'))
	term.SetCursorPos(1, 1)
	term.ReverseIndex()
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "X\nA\nB\nC", str)
	}
}

// Zig: test "Terminal: reverseIndex not top of screen"
func TestTm2_Terminal_reverseIndex_not_top_of_screen(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	zigTry(t, term.Print('A'))
	term.SetCursorPos(2, 1)
	zigTry(t, term.Print('B'))
	term.SetCursorPos(3, 1)
	zigTry(t, term.Print('C'))
	term.SetCursorPos(2, 1)
	term.ReverseIndex()
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "X\nB\nC", str)
	}
}

// Zig: test "Terminal: reverseIndex top/bottom margins"
func TestTm2_Terminal_reverseIndex_top_bottom_margins(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	zigTry(t, term.Print('A'))
	term.SetCursorPos(2, 1)
	zigTry(t, term.Print('B'))
	term.SetCursorPos(3, 1)
	zigTry(t, term.Print('C'))
	term.SetTopAndBottomMargin(2, 3)
	term.SetCursorPos(2, 1)
	term.ReverseIndex()

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "A\n\nB", str)
	}
}

// Zig: test "Terminal: reverseIndex outside top/bottom margins"
func TestTm2_Terminal_reverseIndex_outside_top_bottom_margins(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	zigTry(t, term.Print('A'))
	term.SetCursorPos(2, 1)
	zigTry(t, term.Print('B'))
	term.SetCursorPos(3, 1)
	zigTry(t, term.Print('C'))
	term.SetTopAndBottomMargin(2, 3)
	term.SetCursorPos(1, 1)
	term.ReverseIndex()

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "A\nB\nC", str)
	}
}

// Zig: test "Terminal: reverseIndex left/right margins"
func TestTm2_Terminal_reverseIndex_left_right_margins(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC")))
	term.SetCursorPos(2, 1)
	zigTry(t, term.PrintString([]uint8("DEF")))
	term.SetCursorPos(3, 1)
	zigTry(t, term.PrintString([]uint8("GHI")))
	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(2, 3)
	term.SetCursorPos(1, 2)
	term.ReverseIndex()

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "A\nDBC\nGEF\n HI", str)
	}
}

// Zig: test "Terminal: reverseIndex outside left/right margins"
func TestTm2_Terminal_reverseIndex_outside_left_right_margins(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC")))
	term.SetCursorPos(2, 1)
	zigTry(t, term.PrintString([]uint8("DEF")))
	term.SetCursorPos(3, 1)
	zigTry(t, term.PrintString([]uint8("GHI")))
	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(2, 3)
	term.SetCursorPos(1, 1)
	term.ReverseIndex()

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABC\nDEF\nGHI", str)
	}
}

// Zig: test "Terminal: index"
func TestTm2_Terminal_index(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(2, 5))
	defer term.Deinit()

	zigTry(t, term.Index())
	zigTry(t, term.Print('A'))

	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm2Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\nA", str)
	}
}

// Zig: test "Terminal: index from the bottom"
func TestTm2_Terminal_index_from_the_bottom(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(2, 5))
	defer term.Deinit()

	term.SetCursorPos(5, 1)
	zigTry(t, term.Print('A'))
	term.CursorLeft(1) // undo moving right from 'A'

	term.clearDirty()
	zigTry(t, term.Index())
	zigTry(t, term.Print('B'))

	expect(t, term.isDirty(tm2Active(0, 3)), "t.isDirty(.{ .active = .{ .x = 0, .y = 3 } })")
	expect(t, term.isDirty(tm2Active(0, 4)), "t.isDirty(.{ .active = .{ .x = 0, .y = 4 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\n\n\nA\nB", str)
	}
}

// Zig: test "Terminal: index scrolling with hyperlink"
func TestTm2_Terminal_index_scrolling_with_hyperlink(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(2, 5))
	defer term.Deinit()

	term.SetCursorPos(5, 1)
	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://example.com"), None[[]uint8]()))
	zigTry(t, term.Print('A'))
	term.Screens.Active.EndHyperlink()
	term.CursorLeft(1) // undo moving right from 'A'
	zigTry(t, term.Index())
	zigTry(t, term.Print('B'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\n\n\nA\nB", str)
	}

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Viewport(0, 3)))
		row := list_cell.Row
		expect(t, row.hyperlink, "row.hyperlink")
		cell := list_cell.Cell
		expect(t, cell.Hyperlink(), "cell.hyperlink")
		id := zigUnwrap(t, list_cell.Node.data.LookupHyperlink(cell))
		expectEqual(t, HyperlinkID(1), id)
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Viewport(0, 4)))
		row := list_cell.Row
		expect(t, !row.hyperlink, "!row.hyperlink")
		cell := list_cell.Cell
		expect(t, !cell.Hyperlink(), "!cell.hyperlink")
		id := list_cell.Node.data.LookupHyperlink(cell)
		expect(t, !id.Valid, "id == null")
	}
}

// Zig: test "Terminal: index outside of scrolling region"
func TestTm2_Terminal_index_outside_of_scrolling_region(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(2, 5))
	defer term.Deinit()

	expectEqual(t, SizeCellCountInt(0), term.Screens.Active.Cursor.Y)
	term.SetTopAndBottomMargin(2, 5)
	zigTry(t, term.Index())
	expectEqual(t, SizeCellCountInt(1), term.Screens.Active.Cursor.Y)
}

// Zig: test "Terminal: index from the bottom outside of scroll region"
func TestTm2_Terminal_index_from_the_bottom_outside_of_scroll_region(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(2, 5))
	defer term.Deinit()

	term.SetTopAndBottomMargin(1, 2)
	term.SetCursorPos(5, 1)
	zigTry(t, term.Print('A'))
	term.clearDirty()
	zigTry(t, term.Index())
	zigTry(t, term.Print('B'))
	expect(t, term.isDirty(tm2Active(0, 4)), "t.isDirty(.{ .active = .{ .x = 0, .y = 4 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\n\n\n\nAB", str)
	}
}

// Zig: test "Terminal: index no scroll region, top of screen"
func TestTm2_Terminal_index_no_scroll_region_top_of_screen(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	zigTry(t, term.Print('A'))
	term.clearDirty()
	zigTry(t, term.Index())
	zigTry(t, term.Print('X'))

	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm2Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "A\n X", str)
	}
}

// Zig: test "Terminal: index bottom of primary screen"
func TestTm2_Terminal_index_bottom_of_primary_screen(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.SetCursorPos(5, 1)
	zigTry(t, term.Print('A'))
	term.clearDirty()
	zigTry(t, term.Index())
	zigTry(t, term.Print('X'))

	expect(t, term.isDirty(tm2Active(0, 3)), "t.isDirty(.{ .active = .{ .x = 0, .y = 3 } })")
	expect(t, term.isDirty(tm2Active(0, 4)), "t.isDirty(.{ .active = .{ .x = 0, .y = 4 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\n\n\nA\n X", str)
	}
}

// Zig: test "Terminal: index bottom of primary screen background sgr"
func TestTm2_Terminal_index_bottom_of_primary_screen_background_sgr(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.SetCursorPos(5, 1)
	zigTry(t, term.Print('A'))
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: 0xFF, G: 0, B: 0}}))
	zigTry(t, term.Index())

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\n\n\nA", str)
		for x := 0; x < 5; x++ {
			list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Active(SizeCellCountInt(x), 4)))
			expect(t, list_cell.Cell.ContentTag() == PageCellContentTagBgColorRgb, "list_cell.cell.content_tag == .bg_color_rgb")
			expectEqual(t, PageCellRGB{R: 0xFF, G: 0, B: 0}, list_cell.Cell.content().colorRGB())
		}
	}
}

// Zig: test "Terminal: index inside scroll region"
func TestTm2_Terminal_index_inside_scroll_region(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.SetTopAndBottomMargin(1, 3)
	zigTry(t, term.Print('A'))
	term.clearDirty()
	zigTry(t, term.Index())
	zigTry(t, term.Print('X'))

	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm2Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "A\n X", str)
	}
}

// Zig: test "Terminal: index bottom of scroll region with hyperlinks"
func TestTm2_Terminal_index_bottom_of_scroll_region_with_hyperlinks(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.SetTopAndBottomMargin(1, 2)
	zigTry(t, term.Print('A'))
	zigTry(t, term.Index())
	term.CarriageReturn()
	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://example.com"), None[[]uint8]()))
	zigTry(t, term.Print('B'))
	term.Screens.Active.EndHyperlink()
	zigTry(t, term.Index())
	term.CarriageReturn()
	zigTry(t, term.Print('C'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "B\nC", str)
	}

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Viewport(0, 0)))
		row := list_cell.Row
		expect(t, row.hyperlink, "row.hyperlink")
		cell := list_cell.Cell
		expect(t, cell.Hyperlink(), "cell.hyperlink")
		id := zigUnwrap(t, list_cell.Node.data.LookupHyperlink(cell))
		expectEqual(t, HyperlinkID(1), id)
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Viewport(0, 1)))
		row := list_cell.Row
		expect(t, !row.hyperlink, "!row.hyperlink")
		cell := list_cell.Cell
		expect(t, !cell.Hyperlink(), "!cell.hyperlink")
		id := list_cell.Node.data.LookupHyperlink(cell)
		expect(t, !id.Valid, "id == null")
	}
}

// Zig: test "Terminal: index bottom of scroll region clear hyperlinks"
func TestTm2_Terminal_index_bottom_of_scroll_region_clear_hyperlinks(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	opts := terminalOptionsDefault(5, 5)
	opts.MaxScrollback = 0
	term := tm2Init(t, opts)
	defer term.Deinit()

	term.SetTopAndBottomMargin(2, 3)
	term.SetCursorPos(2, 1)
	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://example.com"), None[[]uint8]()))
	zigTry(t, term.Print('A'))
	term.Screens.Active.EndHyperlink()
	zigTry(t, term.Index())
	term.CarriageReturn()
	zigTry(t, term.Print('B'))
	zigTry(t, term.Index())
	term.CarriageReturn()
	zigTry(t, term.Print('C'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\nB\nC", str)
	}

	for y := 1; y < 3; y++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Viewport(0, uint32(y))))
		row := list_cell.Row
		expect(t, !row.hyperlink, "!row.hyperlink")
		cell := list_cell.Cell
		expect(t, !cell.Hyperlink(), "!cell.hyperlink")
		id := list_cell.Node.data.LookupHyperlink(cell)
		expect(t, !id.Valid, "id == null")
		page := &list_cell.Node.data
		expectEqual(t, uint(0), page.hyperlink_set.Count())
	}
}

// Zig: test "Terminal: index bottom of scroll region with background SGR"
func TestTm2_Terminal_index_bottom_of_scroll_region_with_background_SGR(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.SetTopAndBottomMargin(1, 3)
	term.SetCursorPos(4, 1)
	zigTry(t, term.Print('B'))
	term.SetCursorPos(3, 1)
	zigTry(t, term.Print('A'))
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: 0xFF, G: 0, B: 0}}))
	zigTry(t, term.Index())

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\nA\n\nB", str)
	}

	for x := 0; x < int(term.Cols); x++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Active(SizeCellCountInt(x), 2)))
		expect(t, list_cell.Cell.ContentTag() == PageCellContentTagBgColorRgb, "list_cell.cell.content_tag == .bg_color_rgb")
		expectEqual(t, PageCellRGB{R: 0xFF, G: 0, B: 0}, list_cell.Cell.content().colorRGB())
	}
}

// Zig: test "Terminal: index bottom of primary screen with scroll region"
func TestTm2_Terminal_index_bottom_of_primary_screen_with_scroll_region(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.SetTopAndBottomMargin(1, 3)
	term.SetCursorPos(3, 1)
	zigTry(t, term.Print('A'))
	term.SetCursorPos(5, 1)
	term.clearDirty()
	zigTry(t, term.Index())
	zigTry(t, term.Index())
	zigTry(t, term.Index())
	zigTry(t, term.Print('X'))

	for y := 0; y < 4; y++ {
		expect(t, !term.isDirty(tm2Active(0, uint32(y))), "!t.isDirty(.{ .active = .{ .x = 0, .y = @intCast(y) } })")
	}
	expect(t, term.isDirty(tm2Active(0, 4)), "t.isDirty(.{ .active = .{ .x = 0, .y = 4 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\n\nA\n\nX", str)
	}
}

// Zig: test "Terminal: index outside left/right margin"
func TestTm2_Terminal_index_outside_left_right_margin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(10, 5))
	defer term.Deinit()

	term.SetTopAndBottomMargin(1, 3)
	term.ScrollingRegion.Left = 3
	term.ScrollingRegion.Right = 5
	term.SetCursorPos(3, 3)
	zigTry(t, term.Print('A'))
	term.SetCursorPos(3, 1)
	term.clearDirty()
	zigTry(t, term.Index())
	zigTry(t, term.Print('X'))

	expect(t, term.isDirty(tm2Active(0, 2)), "t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\n\nX A", str)
	}
}

// Zig: test "Terminal: index inside left/right margin"
func TestTm2_Terminal_index_inside_left_right_margin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(10, 5))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("AAAAAA")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("AAAAAA")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("AAAAAA")))
	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetTopAndBottomMargin(1, 3)
	term.SetLeftAndRightMargin(1, 3)
	term.SetCursorPos(3, 1)

	term.clearDirty()
	zigTry(t, term.Index())

	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm2Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, term.isDirty(tm2Active(0, 2)), "t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")

	expectEqual(t, SizeCellCountInt(2), term.Screens.Active.Cursor.Y)
	expectEqual(t, SizeCellCountInt(0), term.Screens.Active.Cursor.X)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "AAAAAA\nAAAAAA\n   AAA", str)
	}
}

// Zig: test "Terminal: index bottom of scroll region creates scrollback"
func TestTm2_Terminal_index_bottom_of_scroll_region_creates_scrollback(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.SetTopAndBottomMargin(1, 3)
	zigTry(t, term.PrintString([]uint8("1\n2\n3")))
	term.SetCursorPos(4, 1)
	zigTry(t, term.Print('X'))
	term.SetCursorPos(3, 1)
	zigTry(t, term.Index())
	zigTry(t, term.Print('Y'))

	{
		str := tm2DumpStringAlloc(t, term.Screens.Active, Point{Tag: PointTagViewport})
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "2\n3\nY\nX", str)
	}
	{
		str := tm2DumpStringAlloc(t, term.Screens.Active, Point{Tag: PointTagScreen})
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "1\n2\n3\nY\nX", str)
	}
}

// Zig: test "Terminal: index bottom of scroll region no scrollback"
func TestTm2_Terminal_index_bottom_of_scroll_region_no_scrollback(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	opts := terminalOptionsDefault(5, 5)
	opts.MaxScrollback = 0
	term := tm2Init(t, opts)
	defer term.Deinit()

	term.SetTopAndBottomMargin(1, 3)
	term.SetCursorPos(4, 1)
	zigTry(t, term.Print('B'))
	term.SetCursorPos(3, 1)
	zigTry(t, term.Print('A'))
	term.clearDirty()
	zigTry(t, term.Index())
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\nA\n X\nB", str)
	}
}

// Zig: test "Terminal: index bottom of scroll region blank line preserves SGR"
func TestTm2_Terminal_index_bottom_of_scroll_region_blank_line_preserves_SGR(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.SetTopAndBottomMargin(1, 3)
	zigTry(t, term.PrintString([]uint8("1\n2\n3")))
	term.SetCursorPos(4, 1)
	zigTry(t, term.Print('X'))
	term.SetCursorPos(3, 1)
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: 0xFF, G: 0, B: 0}}))
	zigTry(t, term.Index())

	{
		str := tm2DumpStringAlloc(t, term.Screens.Active, Point{Tag: PointTagViewport})
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "2\n3\n\nX", str)
	}
	{
		str := tm2DumpStringAlloc(t, term.Screens.Active, Point{Tag: PointTagScreen})
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "1\n2\n3\n\nX", str)
	}
	for x := 0; x < int(term.Cols); x++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Active(SizeCellCountInt(x), 2)))
		expect(t, list_cell.Cell.ContentTag() == PageCellContentTagBgColorRgb, "list_cell.cell.content_tag == .bg_color_rgb")
		expectEqual(t, PageCellRGB{R: 0xFF, G: 0, B: 0}, list_cell.Cell.content().colorRGB())
	}
}

// Zig: test "Terminal: cursorUp basic"
func TestTm2_Terminal_cursorUp_basic(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.SetCursorPos(3, 1)
	zigTry(t, term.Print('A'))
	term.CursorUp(10)
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, " X\n\nA", str)
	}
}

// Zig: test "Terminal: cursorUp below top scroll margin"
func TestTm2_Terminal_cursorUp_below_top_scroll_margin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.SetTopAndBottomMargin(2, 4)
	term.SetCursorPos(3, 1)
	zigTry(t, term.Print('A'))
	term.CursorUp(5)
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\n X\nA", str)
	}
}

// Zig: test "Terminal: cursorUp above top scroll margin"
func TestTm2_Terminal_cursorUp_above_top_scroll_margin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.SetTopAndBottomMargin(3, 5)
	term.SetCursorPos(3, 1)
	zigTry(t, term.Print('A'))
	term.SetCursorPos(2, 1)
	term.CursorUp(10)
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "X\n\nA", str)
	}
}

// Zig: test "Terminal: cursorUp resets wrap"
func TestTm2_Terminal_cursorUp_resets_wrap(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	for _, c := range []byte("ABCDE") {
		zigTry(t, term.Print(uint32(c)))
	}
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	term.CursorUp(1)
	expect(t, !term.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABCDX", str)
	}
}

// Zig: test "Terminal: cursorLeft no wrap"
func TestTm2_Terminal_cursorLeft_no_wrap(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(10, 5))
	defer term.Deinit()

	zigTry(t, term.Print('A'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('B'))
	term.CursorLeft(10)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "A\nB", str)
	}
}

// Zig: test "Terminal: cursorLeft unsets pending wrap state"
func TestTm2_Terminal_cursorLeft_unsets_pending_wrap_state(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	for _, c := range []byte("ABCDE") {
		zigTry(t, term.Print(uint32(c)))
	}
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	term.CursorLeft(1)
	expect(t, !term.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABCXE", str)
	}
}

// Zig: test "Terminal: cursorLeft unsets pending wrap state with longer jump"
func TestTm2_Terminal_cursorLeft_unsets_pending_wrap_state_with_longer_jump(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	for _, c := range []byte("ABCDE") {
		zigTry(t, term.Print(uint32(c)))
	}
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	term.CursorLeft(3)
	expect(t, !term.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "AXCDE", str)
	}
}

// Zig: test "Terminal: cursorLeft reverse wrap with pending wrap state"
func TestTm2_Terminal_cursorLeft_reverse_wrap_with_pending_wrap_state(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.Modes.Set(ModesModeWraparound, true)
	term.Modes.Set(ModesModeReverseWrap, true)

	for _, c := range []byte("ABCDE") {
		zigTry(t, term.Print(uint32(c)))
	}
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	term.CursorLeft(1)
	expect(t, !term.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABCDX", str)
	}
}

// Zig: test "Terminal: cursorLeft reverse wrap extended with pending wrap state"
func TestTm2_Terminal_cursorLeft_reverse_wrap_extended_with_pending_wrap_state(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.Modes.Set(ModesModeWraparound, true)
	term.Modes.Set(ModesModeReverseWrapExtended, true)

	for _, c := range []byte("ABCDE") {
		zigTry(t, term.Print(uint32(c)))
	}
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	term.CursorLeft(1)
	expect(t, !term.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABCDX", str)
	}
}

// Zig: test "Terminal: cursorLeft reverse wrap"
func TestTm2_Terminal_cursorLeft_reverse_wrap(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.Modes.Set(ModesModeWraparound, true)
	term.Modes.Set(ModesModeReverseWrap, true)

	for _, c := range []byte("ABCDE1") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.CursorLeft(2)
	zigTry(t, term.Print('X'))
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABCDX\n1", str)
	}
}

// Zig: test "Terminal: cursorLeft reverse wrap with no soft wrap"
func TestTm2_Terminal_cursorLeft_reverse_wrap_with_no_soft_wrap(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.Modes.Set(ModesModeWraparound, true)
	term.Modes.Set(ModesModeReverseWrap, true)

	for _, c := range []byte("ABCDE") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('1'))
	term.CursorLeft(2)
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABCDE\nX", str)
	}
}

// Zig: test "Terminal: cursorLeft reverse wrap before left margin"
func TestTm2_Terminal_cursorLeft_reverse_wrap_before_left_margin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.Modes.Set(ModesModeWraparound, true)
	term.Modes.Set(ModesModeReverseWrap, true)
	term.SetTopAndBottomMargin(3, 0)
	term.CursorLeft(1)
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\n\nX", str)
	}
}

// Zig: test "Terminal: cursorLeft extended reverse wrap"
func TestTm2_Terminal_cursorLeft_extended_reverse_wrap(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.Modes.Set(ModesModeWraparound, true)
	term.Modes.Set(ModesModeReverseWrapExtended, true)

	for _, c := range []byte("ABCDE") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('1'))
	term.CursorLeft(2)
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABCDX\n1", str)
	}
}

// Zig: test "Terminal: cursorLeft extended reverse wrap bottom wraparound"
func TestTm2_Terminal_cursorLeft_extended_reverse_wrap_bottom_wraparound(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 3))
	defer term.Deinit()

	term.Modes.Set(ModesModeWraparound, true)
	term.Modes.Set(ModesModeReverseWrapExtended, true)

	for _, c := range []byte("ABCDE") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('1'))
	term.CursorLeft(uint(1 + term.Cols + 1))
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABCDE\n1\n    X", str)
	}
}

// Zig: test "Terminal: cursorLeft extended reverse wrap is priority if both set"
func TestTm2_Terminal_cursorLeft_extended_reverse_wrap_is_priority_if_both_set(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 3))
	defer term.Deinit()

	term.Modes.Set(ModesModeWraparound, true)
	term.Modes.Set(ModesModeReverseWrap, true)
	term.Modes.Set(ModesModeReverseWrapExtended, true)

	for _, c := range []byte("ABCDE") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('1'))
	term.CursorLeft(uint(1 + term.Cols + 1))
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABCDE\n1\n    X", str)
	}
}

// Zig: test "Terminal: cursorLeft extended reverse wrap above top scroll region"
func TestTm2_Terminal_cursorLeft_extended_reverse_wrap_above_top_scroll_region(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.Modes.Set(ModesModeWraparound, true)
	term.Modes.Set(ModesModeReverseWrapExtended, true)

	term.SetTopAndBottomMargin(3, 0)
	term.SetCursorPos(2, 1)
	term.CursorLeft(1000)

	expectEqual(t, SizeCellCountInt(0), term.Screens.Active.Cursor.X)
	expectEqual(t, SizeCellCountInt(0), term.Screens.Active.Cursor.Y)
}

// Zig: test "Terminal: cursorLeft reverse wrap on first row"
func TestTm2_Terminal_cursorLeft_reverse_wrap_on_first_row(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.Modes.Set(ModesModeWraparound, true)
	term.Modes.Set(ModesModeReverseWrap, true)

	term.SetTopAndBottomMargin(3, 0)
	term.SetCursorPos(1, 2)
	term.CursorLeft(1000)

	expectEqual(t, SizeCellCountInt(0), term.Screens.Active.Cursor.X)
	expectEqual(t, SizeCellCountInt(0), term.Screens.Active.Cursor.Y)
}

// Zig: test "Terminal: cursorDown basic"
func TestTm2_Terminal_cursorDown_basic(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	zigTry(t, term.Print('A'))
	term.CursorDown(10)
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "A\n\n\n\n X", str)
	}
}

// Zig: test "Terminal: cursorDown above bottom scroll margin"
func TestTm2_Terminal_cursorDown_above_bottom_scroll_margin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.SetTopAndBottomMargin(1, 3)
	zigTry(t, term.Print('A'))
	term.CursorDown(10)
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "A\n\n X", str)
	}
}

// Zig: test "Terminal: cursorDown below bottom scroll margin"
func TestTm2_Terminal_cursorDown_below_bottom_scroll_margin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.SetTopAndBottomMargin(1, 3)
	zigTry(t, term.Print('A'))
	term.SetCursorPos(4, 1)
	term.CursorDown(10)
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "A\n\n\n\nX", str)
	}
}

// Zig: test "Terminal: cursorDown resets wrap"
func TestTm2_Terminal_cursorDown_resets_wrap(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	for _, c := range []byte("ABCDE") {
		zigTry(t, term.Print(uint32(c)))
	}
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	term.CursorDown(1)
	expect(t, !term.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABCDE\n    X", str)
	}
}

// Zig: test "Terminal: cursorRight resets wrap"
func TestTm2_Terminal_cursorRight_resets_wrap(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	for _, c := range []byte("ABCDE") {
		zigTry(t, term.Print(uint32(c)))
	}
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	term.CursorRight(1)
	expect(t, !term.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABCDX", str)
	}
}

// Zig: test "Terminal: cursorRight to the edge of screen"
func TestTm2_Terminal_cursorRight_to_the_edge_of_screen(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.CursorRight(100)
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "    X", str)
	}
}

// Zig: test "Terminal: cursorRight left of right margin"
func TestTm2_Terminal_cursorRight_left_of_right_margin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.ScrollingRegion.Right = 2
	term.CursorRight(100)
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "  X", str)
	}
}

// Zig: test "Terminal: cursorRight right of right margin"
func TestTm2_Terminal_cursorRight_right_of_right_margin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	term.ScrollingRegion.Right = 2
	term.SetCursorPos(1, 4)
	term.CursorRight(100)
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "    X", str)
	}
}

// Zig: test "Terminal: deleteLines simple"
func TestTm2_Terminal_deleteLines_simple(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI")))
	term.SetCursorPos(2, 2)

	term.clearDirty()
	term.DeleteLines(1)

	expect(t, !term.isDirty(tm2Active(0, 0)), "!t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm2Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, term.isDirty(tm2Active(0, 2)), "t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")
	expect(t, term.isDirty(tm2Active(0, 3)), "t.isDirty(.{ .active = .{ .x = 0, .y = 3 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABC\nGHI", str)
	}
}

// Zig: test "Terminal: deleteLines colors with bg color"
func TestTm2_Terminal_deleteLines_colors_with_bg_color(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI")))
	term.SetCursorPos(2, 2)

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: 0xFF, G: 0, B: 0}}))
	term.DeleteLines(1)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABC\nGHI", str)
	}

	for x := 0; x < int(term.Cols); x++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Active(SizeCellCountInt(x), 4)))
		expect(t, list_cell.Cell.ContentTag() == PageCellContentTagBgColorRgb, "list_cell.cell.content_tag == .bg_color_rgb")
		expectEqual(t, PageCellRGB{R: 0xFF, G: 0, B: 0}, list_cell.Cell.content().colorRGB())
	}
}

// Zig: test "Terminal: deleteLines across page boundary marks all shifted rows dirty"
func TestTm2_Terminal_deleteLines_across_page_boundary_marks_all_shifted_rows_dirty(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	opts := terminalOptionsDefault(10, 5)
	opts.MaxScrollback = 1024
	term := tm2Init(t, opts)
	defer term.Deinit()

	first_page := term.Screens.Active.Pages.pages.first
	expect(t, first_page != nil, "t.screens.active.pages.pages.first.?")
	first_page_nrows := first_page.data.capacity.rows

	// Fill up the first page minus 3 rows
	for i := 0; i < int(first_page_nrows)-3; i++ {
		zigTry(t, term.Linefeed())
	}

	// Add content that will cross a page boundary
	zigTry(t, term.PrintString([]uint8("1AAAA")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("2BBBB")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("3CCCC")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("4DDDD")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("5EEEE")))

	// Verify we now have a second page
	expect(t, first_page.next != nil, "first_page.next != null")

	term.SetCursorPos(1, 1)
	term.clearDirty()
	term.DeleteLines(1)

	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm2Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, term.isDirty(tm2Active(0, 2)), "t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")
	expect(t, term.isDirty(tm2Active(0, 3)), "t.isDirty(.{ .active = .{ .x = 0, .y = 3 } })")
	expect(t, term.isDirty(tm2Active(0, 4)), "t.isDirty(.{ .active = .{ .x = 0, .y = 4 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "2BBBB\n3CCCC\n4DDDD\n5EEEE", str)
	}
}

// Zig: test "Terminal: deleteLines (legacy)"
func TestTm2_Terminal_deleteLines_legacy(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(80, 80))
	defer term.Deinit()

	// Initial value
	zigTry(t, term.Print('A'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('B'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('C'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('D'))

	term.CursorUp(2)
	term.DeleteLines(1)

	zigTry(t, term.Print('E'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())

	// We should be
	expectEqual(t, SizeCellCountInt(0), term.Screens.Active.Cursor.X)
	expectEqual(t, SizeCellCountInt(2), term.Screens.Active.Cursor.Y)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "A\nE\nD", str)
	}
}

// Zig: test "Terminal: deleteLines with scroll region"
func TestTm2_Terminal_deleteLines_with_scroll_region(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(80, 80))
	defer term.Deinit()

	// Initial value
	zigTry(t, term.Print('A'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('B'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('C'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('D'))

	term.SetTopAndBottomMargin(1, 3)
	term.SetCursorPos(1, 1)

	term.clearDirty()
	term.DeleteLines(1)

	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm2Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, term.isDirty(tm2Active(0, 2)), "t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")
	expect(t, !term.isDirty(tm2Active(0, 3)), "!t.isDirty(.{ .active = .{ .x = 0, .y = 3 } })")

	zigTry(t, term.Print('E'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())

	// We should be
	// try testing.expectEqual(@as(usize, 0), t.screens.active.cursor.x);
	// try testing.expectEqual(@as(usize, 2), t.screens.active.cursor.y);

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "E\nC\n\nD", str)
	}
}

// Zig: test "Terminal: deleteLines with scroll region, large count"
func TestTm2_Terminal_deleteLines_with_scroll_region_large_count(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(80, 80))
	defer term.Deinit()

	// Initial value
	zigTry(t, term.Print('A'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('B'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('C'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('D'))

	term.SetTopAndBottomMargin(1, 3)
	term.SetCursorPos(1, 1)

	term.clearDirty()
	term.DeleteLines(5)

	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm2Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, term.isDirty(tm2Active(0, 2)), "t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")
	expect(t, !term.isDirty(tm2Active(0, 3)), "!t.isDirty(.{ .active = .{ .x = 0, .y = 3 } })")

	zigTry(t, term.Print('E'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())

	// We should be
	// try testing.expectEqual(@as(usize, 0), t.screens.active.cursor.x);
	// try testing.expectEqual(@as(usize, 2), t.screens.active.cursor.y);

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "E\n\n\nD", str)
	}
}

// Zig: test "Terminal: deleteLines with scroll region, cursor outside of region"
func TestTm2_Terminal_deleteLines_with_scroll_region_cursor_outside_of_region(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(80, 80))
	defer term.Deinit()

	// Initial value
	zigTry(t, term.Print('A'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('B'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('C'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('D'))

	term.SetTopAndBottomMargin(1, 3)
	term.SetCursorPos(4, 1)

	term.clearDirty()
	term.DeleteLines(1)

	for y := 0; y < 4; y++ {
		expect(t, !term.isDirty(tm2Active(0, uint32(y))), "!t.isDirty(.{ .active = .{ .x = 0, .y = @intCast(y) } })")
	}

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "A\nB\nC\nD", str)
	}
}

// Zig: test "Terminal: deleteLines resets pending wrap"
func TestTm2_Terminal_deleteLines_resets_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	for _, c := range []byte("ABCDE") {
		zigTry(t, term.Print(uint32(c)))
	}
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	term.DeleteLines(1)
	expect(t, !term.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
	zigTry(t, term.Print('B'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "B", str)
	}
}

// Zig: test "Terminal: deleteLines resets wrap"
func TestTm2_Terminal_deleteLines_resets_wrap(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(3, 3))
	defer term.Deinit()

	zigTry(t, term.Print('1'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	for _, c := range []byte("ABCDEF") {
		zigTry(t, term.Print(uint32(c)))
	}

	term.SetTopAndBottomMargin(1, 2)
	term.SetCursorPos(1, 1)
	term.DeleteLines(1)
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "XBC\n\nDEF", str)
	}

	for y := 0; y < int(term.Rows); y++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Active(0, uint32(y))))
		row := list_cell.Row
		expect(t, !row.wrap, "!row.wrap")
	}
}

// Zig: test "Terminal: deleteLines left/right scroll region"
func TestTm2_Terminal_deleteLines_left_right_scroll_region(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(10, 10))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC123")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF456")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI789")))
	term.ScrollingRegion.Left = 1
	term.ScrollingRegion.Right = 3
	term.SetCursorPos(2, 2)

	term.clearDirty()
	term.DeleteLines(1)

	expect(t, !term.isDirty(tm2Active(0, 0)), "!t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	for y := 1; y < 3; y++ {
		expect(t, term.isDirty(tm2Active(0, uint32(y))), "t.isDirty(.{ .active = .{ .x = 0, .y = @intCast(y) } })")
	}

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABC123\nDHI756\nG   89", str)
	}
}

// Zig: test "Terminal: deleteLines left/right scroll region from top"
func TestTm2_Terminal_deleteLines_left_right_scroll_region_from_top(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(10, 10))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC123")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF456")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI789")))
	term.ScrollingRegion.Left = 1
	term.ScrollingRegion.Right = 3
	term.SetCursorPos(1, 2)

	term.clearDirty()
	term.DeleteLines(1)

	for y := 0; y < 3; y++ {
		expect(t, term.isDirty(tm2Active(0, uint32(y))), "t.isDirty(.{ .active = .{ .x = 0, .y = @intCast(y) } })")
	}

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "AEF423\nDHI756\nG   89", str)
	}
}

// Zig: test "Terminal: deleteLines left/right scroll region high count"
func TestTm2_Terminal_deleteLines_left_right_scroll_region_high_count(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(10, 10))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC123")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF456")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI789")))
	term.ScrollingRegion.Left = 1
	term.ScrollingRegion.Right = 3
	term.SetCursorPos(2, 2)

	term.clearDirty()
	term.DeleteLines(100)

	expect(t, !term.isDirty(tm2Active(0, 0)), "!t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	for y := 1; y < 3; y++ {
		expect(t, term.isDirty(tm2Active(0, uint32(y))), "t.isDirty(.{ .active = .{ .x = 0, .y = @intCast(y) } })")
	}

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABC123\nD   56\nG   89", str)
	}
}

// Zig: test "Terminal: deleteLines wide character spacer head"
func TestTm2_Terminal_deleteLines_wide_character_spacer_head(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 3))
	defer term.Deinit()

	// Initial value
	// +-----+
	// |AAAAA| < Wrapped
	// |BBBB*| < Wrapped     (continued)
	// |WWCCC| < Non-wrapped (continued)
	// +-----+
	// where * represents a spacer head cell
	// and WW is the wide character.
	zigTry(t, term.PrintString([]uint8("AAAAABBBB\U0001F600CCC")))

	// Delete the top line
	// +-----+
	// |BBBB | < Non-wrapped
	// |WWCCC| < Non-wrapped
	// |     | < Non-wrapped
	// +-----+
	// This should convert the spacer head to
	// a regular empty cell, and un-set wrap.
	term.SetCursorPos(1, 1)
	term.DeleteLines(1)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		unwrapped_str := tm2PlainStringUnwrapped(t, term)
		// Zig-only: defer testing.allocator.free(unwrapped_str);
		expectEqualStrings(t, "BBBB\n\U0001F600CCC", str)
		expectEqualStrings(t, "BBBB\n\U0001F600CCC", unwrapped_str)
	}
}

// Zig: test "Terminal: deleteLines wide character spacer head left scroll margin"
func TestTm2_Terminal_deleteLines_wide_character_spacer_head_left_scroll_margin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 3))
	defer term.Deinit()

	// Initial value
	// +-----+
	// |AAAAA| < Wrapped
	// |BBBB*| < Wrapped     (continued)
	// |WWCCC| < Non-wrapped (continued)
	// +-----+
	// where * represents a spacer head cell
	// and WW is the wide character.
	zigTry(t, term.PrintString([]uint8("AAAAABBBB\U0001F600CCC")))

	term.ScrollingRegion.Left = 2

	// Delete the top line
	//    ###  <- scrolling region
	// +-----+
	// |AABB | < Wrapped
	// |BBCCC| < Wrapped     (continued)
	// |WW   | < Non-wrapped (continued)
	// +-----+
	// This should convert the spacer head to
	// a regular empty cell, but due to the
	// left scrolling margin, wrap state should
	// remain.
	term.SetCursorPos(1, 3)
	term.DeleteLines(1)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		unwrapped_str := tm2PlainStringUnwrapped(t, term)
		// Zig-only: defer testing.allocator.free(unwrapped_str);
		expectEqualStrings(t, "AABB\nBBCCC\n\U0001F600", str)
		expectEqualStrings(t, "AABB BBCCC\U0001F600", unwrapped_str)
	}
}

// Zig: test "Terminal: deleteLines wide character spacer head right scroll margin"
func TestTm2_Terminal_deleteLines_wide_character_spacer_head_right_scroll_margin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 3))
	defer term.Deinit()

	// Initial value
	// +-----+
	// |AAAAA| < Wrapped
	// |BBBB*| < Wrapped     (continued)
	// |WWCCC| < Non-wrapped (continued)
	// +-----+
	// where * represents a spacer head cell
	// and WW is the wide character.
	zigTry(t, term.PrintString([]uint8("AAAAABBBB\U0001F600CCC")))

	term.ScrollingRegion.Right = 3

	// Delete the top line
	//  ####   <- scrolling region
	// +-----+
	// |BBBBA| < Wrapped
	// |WWCC | < Wrapped     (continued)
	// |    C| < Non-wrapped (continued)
	// +-----+
	// This should convert the spacer head to
	// a regular empty cell, but due to the
	// right scrolling margin, wrap state should
	// remain.
	term.SetCursorPos(1, 1)
	term.DeleteLines(1)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		unwrapped_str := tm2PlainStringUnwrapped(t, term)
		// Zig-only: defer testing.allocator.free(unwrapped_str);
		expectEqualStrings(t, "BBBBA\n\U0001F600CC\n    C", str)
		expectEqualStrings(t, "BBBBA\U0001F600CC     C", unwrapped_str)
	}
}

// Zig: test "Terminal: deleteLines wide character spacer head left and right scroll margin"
func TestTm2_Terminal_deleteLines_wide_character_spacer_head_left_and_right_scroll_margin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 3))
	defer term.Deinit()

	// Initial value
	// +-----+
	// |AAAAA| < Wrapped
	// |BBBB*| < Wrapped     (continued)
	// |WWCCC| < Non-wrapped (continued)
	// +-----+
	// where * represents a spacer head cell
	// and WW is the wide character.
	zigTry(t, term.PrintString([]uint8("AAAAABBBB\U0001F600CCC")))

	term.ScrollingRegion.Right = 3
	term.ScrollingRegion.Left = 2

	// Delete the top line
	//    ##   <- scrolling region
	// +-----+
	// |AABBA| < Wrapped
	// |BBCC*| < Wrapped     (continued)
	// |WW  C| < Non-wrapped (continued)
	// +-----+
	// Because there is both a left scrolling
	// margin > 1 and a right scrolling margin
	// the spacer head should remain, and the
	// wrap state should be untouched.
	term.SetCursorPos(1, 3)
	term.DeleteLines(1)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		unwrapped_str := tm2PlainStringUnwrapped(t, term)
		// Zig-only: defer testing.allocator.free(unwrapped_str);
		expectEqualStrings(t, "AABBA\nBBCC\n\U0001F600  C", str)
		expectEqualStrings(t, "AABBABBCC\U0001F600  C", unwrapped_str)
	}
}

// Zig: test "Terminal: deleteLines wide character spacer head left (< 2) and right scroll margin"
func TestTm2_Terminal_deleteLines_wide_character_spacer_head_left_2_and_right_scroll_margin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 3))
	defer term.Deinit()

	// Initial value
	// +-----+
	// |AAAAA| < Wrapped
	// |BBBB*| < Wrapped     (continued)
	// |WWCCC| < Non-wrapped (continued)
	// +-----+
	// where * represents a spacer head cell
	// and WW is the wide character.
	zigTry(t, term.PrintString([]uint8("AAAAABBBB\U0001F600CCC")))

	term.ScrollingRegion.Right = 3
	term.ScrollingRegion.Left = 1

	// Delete the top line
	//   ###   <- scrolling region
	// +-----+
	// |ABBBA| < Wrapped
	// |B CC | < Wrapped     (continued)
	// |    C| < Non-wrapped (continued)
	// +-----+
	// Because the left margin is 1, the wide
	// char is split, and therefore removed,
	// along with the spacer head - however,
	// wrap state should be untouched.
	term.SetCursorPos(1, 2)
	term.DeleteLines(1)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		unwrapped_str := tm2PlainStringUnwrapped(t, term)
		// Zig-only: defer testing.allocator.free(unwrapped_str);
		expectEqualStrings(t, "ABBBA\nB CC\n    C", str)
		expectEqualStrings(t, "ABBBAB CC     C", unwrapped_str)
	}
}

// Zig: test "Terminal: deleteLines wide characters split by left/right scroll region boundaries"
func TestTm2_Terminal_deleteLines_wide_characters_split_by_left_right_scroll_region_boundaries(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 2))
	defer term.Deinit()

	// Initial value
	// +-----+
	// |AAAAA|
	// |WWBWW|
	// +-----+
	// where WW represents a wide character
	zigTry(t, term.PrintString([]uint8("AAAAA\n\U0001F600B\U0001F600")))

	term.ScrollingRegion.Right = 3
	term.ScrollingRegion.Left = 1

	// Delete the top line
	//   ###   <- scrolling region
	// +-----+
	// |A B A|
	// |     |
	// +-----+
	// The two wide chars, because they're
	// split by the edge of the scrolling
	// region, get removed.
	term.SetCursorPos(1, 2)
	term.DeleteLines(1)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "A B A", str)
	}
}

// Zig: test "Terminal: deleteLines zero"
func TestTm2_Terminal_deleteLines_zero(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(2, 5))
	defer term.Deinit()

	// This should do nothing
	term.SetCursorPos(1, 1)
	term.DeleteLines(0)
}

// Zig: test "Terminal: default style is empty"
func TestTm2_Terminal_default_style_is_empty(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	zigTry(t, term.Print('A'))

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, uint32('A'), cell.content().codepoint())
		expectEqual(t, StyleID(0), cell.StyleID())
	}
}

// Zig: test "Terminal: bold style"
func TestTm2_Terminal_bold_style(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	zigTry(t, term.Print('A'))

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, uint32('A'), cell.content().codepoint())
		expect(t, cell.StyleID() != 0, "cell.style_id != 0")
		page := &term.Screens.Active.Cursor.PagePin.Node.data
		expect(t, page.styles.RefCount(term.Screens.Active.Cursor.StyleID) > 1, "page.styles.refCount(page.memory, t.screens.active.cursor.style_id) > 1")
	}
}

// Zig: test "Terminal: garbage collect overwritten"
func TestTm2_Terminal_garbage_collect_overwritten(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	zigTry(t, term.Print('A'))
	term.SetCursorPos(1, 1)
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))
	zigTry(t, term.Print('B'))

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, uint32('B'), cell.content().codepoint())
		expect(t, cell.StyleID() == 0, "cell.style_id == 0")
	}

	// verify we have no styles in our style map
	page := &term.Screens.Active.Cursor.PagePin.Node.data
	expectEqual(t, uint(0), page.styles.Count())
}

// Zig: test "Terminal: do not garbage collect old styles in use"
func TestTm2_Terminal_do_not_garbage_collect_old_styles_in_use(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	zigTry(t, term.Print('A'))
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))
	zigTry(t, term.Print('B'))

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Screen(1, 0)))
		cell := list_cell.Cell
		expectEqual(t, uint32('B'), cell.content().codepoint())
		expect(t, cell.StyleID() == 0, "cell.style_id == 0")
	}

	// verify we have no styles in our style map
	page := &term.Screens.Active.Cursor.PagePin.Node.data
	expectEqual(t, uint(1), page.styles.Count())
}

// Zig: test "Terminal: print with style marks the row as styled"
func TestTm2_Terminal_print_with_style_marks_the_row_as_styled(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	zigTry(t, term.Print('A'))
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))
	zigTry(t, term.Print('B'))

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Screen(0, 0)))
		expect(t, list_cell.Row.styled, "list_cell.row.styled")
	}
}

// Zig: test "Terminal: DECALN"
func TestTm2_Terminal_DECALN(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(2, 2))
	defer term.Deinit()

	// Initial value
	zigTry(t, term.Print('A'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('B'))
	zigTry(t, term.Decaln())

	expectEqual(t, SizeCellCountInt(0), term.Screens.Active.Cursor.Y)
	expectEqual(t, SizeCellCountInt(0), term.Screens.Active.Cursor.X)

	for y := 0; y < int(term.Rows); y++ {
		expect(t, term.isDirty(tm2Active(0, uint32(y))), "t.isDirty(.{ .active = .{ .x = 0, .y = @intCast(y) } })")
	}

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "EE\nEE", str)
	}
}

// Zig: test "Terminal: decaln reset margins"
func TestTm2_Terminal_decaln_reset_margins(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(3, 3))
	defer term.Deinit()

	// Initial value
	term.Modes.Set(ModesModeOrigin, true)
	term.SetTopAndBottomMargin(2, 3)
	zigTry(t, term.Decaln())
	term.ScrollDown(1)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\nEEE\nEEE", str)
	}
}

// Zig: test "Terminal: decaln preserves color"
func TestTm2_Terminal_decaln_preserves_color(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(3, 3))
	defer term.Deinit()

	// Initial value
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: 0xFF, G: 0, B: 0}}))
	term.Modes.Set(ModesModeOrigin, true)
	term.SetTopAndBottomMargin(2, 3)
	zigTry(t, term.Decaln())
	term.ScrollDown(1)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\nEEE\nEEE", str)
	}

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Active(0, 0)))
		expect(t, list_cell.Cell.ContentTag() == PageCellContentTagBgColorRgb, "list_cell.cell.content_tag == .bg_color_rgb")
		expectEqual(t, PageCellRGB{R: 0xFF, G: 0, B: 0}, list_cell.Cell.content().colorRGB())
	}
}

// Zig: test "Terminal: DECALN resets graphemes with protected mode"
func TestTm2_Terminal_DECALN_resets_graphemes_with_protected_mode(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(3, 3))
	defer term.Deinit()

	// Add protected mode. A previous version of DECALN accidentally preserved
	// protected mode which left dangling managed memory.
	term.SetProtectedMode(ANSIProtectedModeIso)

	// This is: 👨‍👩‍👧 (which may or may not render correctly)
	term.Modes.Set(ModesModeGraphemeCluster, true)
	zigTry(t, term.Print(0x1F468))
	zigTry(t, term.Print(0x200D))
	zigTry(t, term.Print(0x1F469))
	zigTry(t, term.Print(0x200D))
	zigTry(t, term.Print(0x1F467))

	zigTry(t, term.Decaln())

	expectEqual(t, SizeCellCountInt(0), term.Screens.Active.Cursor.Y)
	expectEqual(t, SizeCellCountInt(0), term.Screens.Active.Cursor.X)
	expect(t, term.Screens.Active.Cursor.Protected, "t.screens.active.cursor.protected")
	expect(t, term.Screens.Active.ProtectedMode == ANSIProtectedModeIso, "t.screens.active.protected_mode == .iso")

	for y := 0; y < int(term.Rows); y++ {
		expect(t, term.isDirty(tm2Active(0, uint32(y))), "t.isDirty(.{ .active = .{ .x = 0, .y = @intCast(y) } })")
	}

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "EEE\nEEE\nEEE", str)
	}
}

// Zig: test "Terminal: insertBlanks zero"
func TestTm2_Terminal_insertBlanks_zero(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 2))
	defer term.Deinit()

	zigTry(t, term.Print('A'))
	zigTry(t, term.Print('B'))
	zigTry(t, term.Print('C'))
	term.SetCursorPos(1, 1)

	term.InsertBlanks(0)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABC", str)
	}
}

// Zig: test "Terminal: insertBlanks"
func TestTm2_Terminal_insertBlanks(t *testing.T) {
	testStdCapacity(t)
	// NOTE: this is not verified with conformance tests, so these
	// tests might actually be verifying wrong behavior.
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 2))
	defer term.Deinit()

	zigTry(t, term.Print('A'))
	zigTry(t, term.Print('B'))
	zigTry(t, term.Print('C'))
	term.SetCursorPos(1, 1)

	term.clearDirty()
	term.InsertBlanks(2)
	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, !term.isDirty(tm2Active(0, 1)), "!t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "  ABC", str)
	}
}

// Zig: test "Terminal: insertBlanks pushes off end"
func TestTm2_Terminal_insertBlanks_pushes_off_end(t *testing.T) {
	testStdCapacity(t)
	// NOTE: this is not verified with conformance tests, so these
	// tests might actually be verifying wrong behavior.
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(3, 2))
	defer term.Deinit()

	zigTry(t, term.Print('A'))
	zigTry(t, term.Print('B'))
	zigTry(t, term.Print('C'))
	term.SetCursorPos(1, 1)

	term.clearDirty()
	term.InsertBlanks(2)
	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "  A", str)
	}
}

// Zig: test "Terminal: insertBlanks more than size"
func TestTm2_Terminal_insertBlanks_more_than_size(t *testing.T) {
	testStdCapacity(t)
	// NOTE: this is not verified with conformance tests, so these
	// tests might actually be verifying wrong behavior.
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(3, 2))
	defer term.Deinit()

	zigTry(t, term.Print('A'))
	zigTry(t, term.Print('B'))
	zigTry(t, term.Print('C'))
	term.SetCursorPos(1, 1)

	term.clearDirty()
	term.InsertBlanks(5)
	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "", str)
	}
}

// Zig: test "Terminal: insertBlanks no scroll region, fits"
func TestTm2_Terminal_insertBlanks_no_scroll_region_fits(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(10, 10))
	defer term.Deinit()

	for _, c := range []byte("ABC") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.SetCursorPos(1, 1)

	term.clearDirty()
	term.InsertBlanks(2)
	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "  ABC", str)
	}
}

// Zig: test "Terminal: insertBlanks preserves background sgr"
func TestTm2_Terminal_insertBlanks_preserves_background_sgr(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(10, 10))
	defer term.Deinit()

	for _, c := range []byte("ABC") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.SetCursorPos(1, 1)
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: 0xFF, G: 0, B: 0}}))
	term.InsertBlanks(2)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "  ABC", str)
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Active(0, 0)))
		expect(t, list_cell.Cell.ContentTag() == PageCellContentTagBgColorRgb, "list_cell.cell.content_tag == .bg_color_rgb")
		expectEqual(t, PageCellRGB{R: 0xFF, G: 0, B: 0}, list_cell.Cell.content().colorRGB())
	}
}

// Zig: test "Terminal: insertBlanks shift off screen"
func TestTm2_Terminal_insertBlanks_shift_off_screen(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 10))
	defer term.Deinit()

	for _, c := range []byte("  ABC") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.SetCursorPos(1, 3)
	term.clearDirty()
	term.InsertBlanks(2)
	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "  X A", str)
	}
}

// Zig: test "Terminal: insertBlanks split multi-cell character"
func TestTm2_Terminal_insertBlanks_split_multi_cell_character(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 10))
	defer term.Deinit()

	for _, c := range []byte("123") {
		zigTry(t, term.Print(uint32(c)))
	}
	zigTry(t, term.Print('橋'))
	term.SetCursorPos(1, 1)
	term.clearDirty()
	term.InsertBlanks(1)
	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, " 123", str)
	}
}

// Zig: test "Terminal: insertBlanks inside left/right scroll region"
func TestTm2_Terminal_insertBlanks_inside_left_right_scroll_region(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(10, 10))
	defer term.Deinit()

	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 4
	term.SetCursorPos(1, 3)
	for _, c := range []byte("ABC") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.SetCursorPos(1, 3)

	term.clearDirty()
	term.InsertBlanks(2)
	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "  X A", str)
	}
}

// Zig: test "Terminal: insertBlanks outside left/right scroll region"
func TestTm2_Terminal_insertBlanks_outside_left_right_scroll_region(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(6, 10))
	defer term.Deinit()

	term.SetCursorPos(1, 4)
	for _, c := range []byte("ABC") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 4
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	term.clearDirty()
	term.InsertBlanks(2)
	expect(t, !term.isDirty(tm2Active(0, 0)), "!t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, !term.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "   ABX", str)
	}
}

// Zig: test "Terminal: insertBlanks left/right scroll region large count"
func TestTm2_Terminal_insertBlanks_left_right_scroll_region_large_count(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(10, 10))
	defer term.Deinit()

	term.Modes.Set(ModesModeOrigin, true)
	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(3, 5)
	term.SetCursorPos(1, 1)
	term.clearDirty()
	term.InsertBlanks(140)
	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "  X", str)
	}
}

// Zig: test "Terminal: insertBlanks deleting graphemes"
func TestTm2_Terminal_insertBlanks_deleting_graphemes(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	// Disable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.PrintString([]uint8("ABC")))

	// This is: 👨‍👩‍👧 (which may or may not render correctly)
	zigTry(t, term.Print(0x1F468))
	zigTry(t, term.Print(0x200D))
	zigTry(t, term.Print(0x1F469))
	zigTry(t, term.Print(0x200D))
	zigTry(t, term.Print(0x1F467))

	// We should have one cell with graphemes
	page := &term.Screens.Active.Cursor.PagePin.Node.data
	expectEqual(t, uint(1), page.GraphemeCount())

	term.SetCursorPos(1, 1)
	term.clearDirty()
	term.InsertBlanks(4)
	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "    A", str)
	}

	// We should have no graphemes
	expectEqual(t, uint(0), page.GraphemeCount())
}

// Zig: test "Terminal: insertBlanks shift graphemes"
func TestTm2_Terminal_insertBlanks_shift_graphemes(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.PrintString([]uint8("A")))

	// This is: 👨‍👩‍👧 (which may or may not render correctly)
	zigTry(t, term.Print(0x1F468))
	zigTry(t, term.Print(0x200D))
	zigTry(t, term.Print(0x1F469))
	zigTry(t, term.Print(0x200D))
	zigTry(t, term.Print(0x1F467))

	// We should have one cell with graphemes
	page := &term.Screens.Active.Cursor.PagePin.Node.data
	expectEqual(t, uint(1), page.GraphemeCount())

	term.SetCursorPos(1, 1)
	term.clearDirty()
	term.InsertBlanks(1)
	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, " A👨‍👩‍👧", str)
	}

	// We should have no graphemes
	expectEqual(t, uint(1), page.GraphemeCount())
}

// Zig: test "Terminal: insertBlanks split multi-cell character from tail"
func TestTm2_Terminal_insertBlanks_split_multi_cell_character_from_tail(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 10))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("橋123")))
	term.SetCursorPos(1, 2)
	term.InsertBlanks(1)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "   12", str)
	}
}

// Zig: test "Terminal: insertBlanks shifts hyperlinks"
func TestTm2_Terminal_insertBlanks_shifts_hyperlinks(t *testing.T) {
	testStdCapacity(t)
	// osc "8;;http://example.com"
	// printf "link"
	// printf "\r"
	// csi "3@"
	// echo
	//
	// link should be preserved, blanks should not be linked

	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(10, 2))
	defer term.Deinit()

	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://example.com"), None[[]uint8]()))
	zigTry(t, term.PrintString([]uint8("ABC")))
	term.SetCursorPos(1, 1)
	term.InsertBlanks(2)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "  ABC", str)
	}

	// Verify all our cells have a hyperlink
	for x := 2; x < 5; x++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Screen(SizeCellCountInt(x), 0)))
		row := list_cell.Row
		expect(t, row.hyperlink, "row.hyperlink")
		cell := list_cell.Cell
		expect(t, cell.Hyperlink(), "cell.hyperlink")
		id := zigUnwrap(t, list_cell.Node.data.LookupHyperlink(cell))
		expectEqual(t, HyperlinkID(1), id)
	}
	for x := 0; x < 2; x++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Screen(SizeCellCountInt(x), 0)))
		cell := list_cell.Cell
		expect(t, !cell.Hyperlink(), "!cell.hyperlink")
		id := list_cell.Node.data.LookupHyperlink(cell)
		expect(t, !id.Valid, "id == null")
	}
}

// Zig: test "Terminal: insertBlanks pushes hyperlink off end completely"
func TestTm2_Terminal_insertBlanks_pushes_hyperlink_off_end_completely(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(3, 2))
	defer term.Deinit()

	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://example.com"), None[[]uint8]()))
	zigTry(t, term.PrintString([]uint8("ABC")))
	term.SetCursorPos(1, 1)
	term.InsertBlanks(3)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "", str)
	}

	for x := 0; x < 3; x++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Screen(SizeCellCountInt(x), 0)))
		row := list_cell.Row
		expect(t, !row.hyperlink, "!row.hyperlink")
		cell := list_cell.Cell
		expect(t, !cell.Hyperlink(), "!cell.hyperlink")
		id := list_cell.Node.data.LookupHyperlink(cell)
		expect(t, !id.Valid, "id == null")
	}
}

// Zig: test "Terminal: insertBlanks wide char straddling right margin"
func TestTm2_Terminal_insertBlanks_wide_char_straddling_right_margin(t *testing.T) {
	testStdCapacity(t)
	// Crash found by AFL++ fuzzer.
	//
	// When a wide character straddles the right scroll margin (head at the
	// margin, spacer_tail just beyond it), insertBlanks shifts the wide head
	// away via swapCells but leaves the orphaned spacer_tail in place,
	// causing a page integrity violation.
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(10, 5))
	defer term.Deinit()

	// Fill row: A B C D 橋 _ _ _ _ _
	// Positions: 0 1 2 3 4W 5T 6 7 8 9
	term.SetCursorPos(1, 1)
	for _, c := range []byte("ABCD") {
		zigTry(t, term.Print(uint32(c)))
	}
	zigTry(t, term.Print('橋')) // wide char: head at 4, spacer_tail at 5

	// Set right margin so the wide head is AT the boundary and the
	// spacer_tail is just outside it.
	term.ScrollingRegion.Right = 4

	// Position cursor at x=2 (1-indexed col 3) and insert one blank.
	// This triggers the swap loop which displaces the wide head at
	// position 4 without clearing the spacer_tail at position 5.
	term.SetCursorPos(1, 3)
	term.InsertBlanks(1)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "AB CD", str)
	}
}

// Zig: test "Terminal: insertBlanks wide char spacer_tail orphaned beyond right margin"
func TestTm2_Terminal_insertBlanks_wide_char_spacer_tail_orphaned_beyond_right_margin(t *testing.T) {
	testStdCapacity(t)
	// Regression test for AFL++ crash.
	//
	// When insertBlanks clears the entire region from cursor to the right
	// margin (scroll_amount == 0), a wide character whose head is AT the
	// right margin gets cleared but its spacer_tail just beyond the margin
	// is left behind, causing a page integrity violation:
	//   "spacer tail not following wide"
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(10, 5))
	defer term.Deinit()

	// Fill cols 0–9 with wide chars: 中中中中中
	// Positions: 0W 1T 2W 3T 4W 5T 6W 7T 8W 9T
	for i := 0; i < 5; i++ {
		zigTry(t, term.Print(0x4E2D))
	}

	// Set left/right margins so that the last wide char (cols 8–9)
	// straddles the boundary: head at col 8 (inside), tail at col 9 (outside).
	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(1, 9) // 1-indexed: left=0, right=8

	// Cursor is now at (0, 0) after DECSLRM.  Print a narrow char to
	// advance cursor to col 1.
	zigTry(t, term.Print('a'))

	// ICH 8: insert 8 blanks at cursor x=1.
	// rem = right(8) - x(1) + 1 = 8, adjusted_count = 8, scroll_amount = 0.
	// The code clears cols 1–8 without noticing the spacer_tail at col 9.
	term.InsertBlanks(8)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "a", str)
	}
}

// Zig: test "Terminal: insert mode with space"
func TestTm2_Terminal_insert_mode_with_space(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(10, 2))
	defer term.Deinit()

	for _, c := range []byte("hello") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.SetCursorPos(1, 2)
	term.Modes.Set(ModesModeInsert, true)
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "hXello", str)
	}
}

// Zig: test "Terminal: insert mode doesn't wrap pushed characters"
func TestTm2_Terminal_insert_mode_doesn_t_wrap_pushed_characters(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 2))
	defer term.Deinit()

	for _, c := range []byte("hello") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.SetCursorPos(1, 2)
	term.Modes.Set(ModesModeInsert, true)
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "hXell", str)
	}
}

// Zig: test "Terminal: insert mode does nothing at the end of the line"
func TestTm2_Terminal_insert_mode_does_nothing_at_the_end_of_the_line(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 2))
	defer term.Deinit()

	for _, c := range []byte("hello") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.Modes.Set(ModesModeInsert, true)
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "hello\nX", str)
	}
}

// Zig: test "Terminal: insert mode with wide characters"
func TestTm2_Terminal_insert_mode_with_wide_characters(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 2))
	defer term.Deinit()

	for _, c := range []byte("hello") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.SetCursorPos(1, 2)
	term.Modes.Set(ModesModeInsert, true)
	zigTry(t, term.Print('😀')) // 0x1F600

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "h😀el", str)
	}
}

// Zig: test "Terminal: insert mode with wide characters at end"
func TestTm2_Terminal_insert_mode_with_wide_characters_at_end(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 2))
	defer term.Deinit()

	for _, c := range []byte("well") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.Modes.Set(ModesModeInsert, true)
	zigTry(t, term.Print('😀')) // 0x1F600

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "well\n😀", str)
	}
}

// Zig: test "Terminal: insert mode pushing off wide character"
func TestTm2_Terminal_insert_mode_pushing_off_wide_character(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 2))
	defer term.Deinit()

	for _, c := range []byte("123") {
		zigTry(t, term.Print(uint32(c)))
	}
	zigTry(t, term.Print('😀')) // 0x1F600
	term.Modes.Set(ModesModeInsert, true)
	term.SetCursorPos(1, 1)
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "X123", str)
	}
}

// Zig: test "Terminal: deleteChars"
func TestTm2_Terminal_deleteChars(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	for _, c := range []byte("ABCDE") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.SetCursorPos(1, 2)

	term.clearDirty()
	term.DeleteChars(2)
	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ADE", str)
	}
}

// Zig: test "Terminal: deleteChars zero count"
func TestTm2_Terminal_deleteChars_zero_count(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	for _, c := range []byte("ABCDE") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.SetCursorPos(1, 2)

	term.clearDirty()
	term.DeleteChars(0)
	expect(t, !term.isDirty(tm2Active(0, 0)), "!t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABCDE", str)
	}
}

// Zig: test "Terminal: deleteChars more than half"
func TestTm2_Terminal_deleteChars_more_than_half(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	for _, c := range []byte("ABCDE") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.SetCursorPos(1, 2)

	term.clearDirty()
	term.DeleteChars(3)
	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "AE", str)
	}
}

// Zig: test "Terminal: deleteChars more than line width"
func TestTm2_Terminal_deleteChars_more_than_line_width(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	for _, c := range []byte("ABCDE") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.SetCursorPos(1, 2)

	term.clearDirty()
	term.DeleteChars(10)
	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "A", str)
	}
}

// Zig: test "Terminal: deleteChars should shift left"
func TestTm2_Terminal_deleteChars_should_shift_left(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	for _, c := range []byte("ABCDE") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.SetCursorPos(1, 2)

	term.clearDirty()
	term.DeleteChars(1)
	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ACDE", str)
	}
}

// Zig: test "Terminal: deleteChars resets pending wrap"
func TestTm2_Terminal_deleteChars_resets_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	for _, c := range []byte("ABCDE") {
		zigTry(t, term.Print(uint32(c)))
	}
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	term.DeleteChars(1)
	expect(t, !term.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABCDX", str)
	}
}

// Zig: test "Terminal: deleteChars resets wrap"
func TestTm2_Terminal_deleteChars_resets_wrap(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	for _, c := range []byte("ABCDE123") {
		zigTry(t, term.Print(uint32(c)))
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Active(0, 0)))
		row := list_cell.Row
		expect(t, row.wrap, "row.wrap")
	}
	term.SetCursorPos(1, 1)
	term.DeleteChars(1)

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Active(0, 0)))
		row := list_cell.Row
		expect(t, !row.wrap, "!row.wrap")
	}

	zigTry(t, term.Print('X'))

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "XCDE\n123", str)
	}
}

// Zig: test "Terminal: deleteChars simple operation"
func TestTm2_Terminal_deleteChars_simple_operation(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(10, 10))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC123")))
	term.SetCursorPos(1, 3)

	term.clearDirty()
	term.DeleteChars(2)
	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "AB23", str)
	}
}

// Zig: test "Terminal: deleteChars preserves background sgr"
func TestTm2_Terminal_deleteChars_preserves_background_sgr(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(10, 10))
	defer term.Deinit()

	for _, c := range []byte("ABC123") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.SetCursorPos(1, 3)
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: 0xFF, G: 0, B: 0}}))
	term.DeleteChars(2)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "AB23", str)
	}
	for x := int(term.Cols) - 2; x < int(term.Cols); x++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Active(SizeCellCountInt(x), 0)))
		expect(t, list_cell.Cell.ContentTag() == PageCellContentTagBgColorRgb, "list_cell.cell.content_tag == .bg_color_rgb")
		expectEqual(t, PageCellRGB{R: 0xFF, G: 0, B: 0}, list_cell.Cell.content().colorRGB())
	}
}

// Zig: test "Terminal: deleteChars outside scroll region"
func TestTm2_Terminal_deleteChars_outside_scroll_region(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(6, 10))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC123")))
	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 4
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	term.clearDirty()
	term.DeleteChars(2)
	expect(t, !term.isDirty(tm2Active(0, 0)), "!t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABC123", str)
	}
}

// Zig: test "Terminal: deleteChars inside scroll region"
func TestTm2_Terminal_deleteChars_inside_scroll_region(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(6, 10))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC123")))
	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 4
	term.SetCursorPos(1, 4)

	term.clearDirty()
	term.DeleteChars(1)
	expect(t, term.isDirty(tm2Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABC2 3", str)
	}
}

// Zig: test "Terminal: deleteChars split wide character from spacer tail"
func TestTm2_Terminal_deleteChars_split_wide_character_from_spacer_tail(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(6, 10))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("A橋123")))
	term.SetCursorPos(1, 3)
	term.DeleteChars(1)

	{
		str := tm2PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "A 123", str)
	}
}

// Zig: test "Terminal: deleteChars split wide character from wide"
func TestTm2_Terminal_deleteChars_split_wide_character_from_wide(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = testing.allocator;
	term := tm2Init(t, terminalOptionsDefault(6, 10))
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("橋123")))
	term.SetCursorPos(1, 1)
	term.DeleteChars(1)

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, uint32(0), cell.content().codepoint())
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Screen(1, 0)))
		cell := list_cell.Cell
		expectEqual(t, uint32('1'), cell.content().codepoint())
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}
}

// Go-only: panic test: Terminal.index: semantic_content_clear_eol set while the cursor's semantic content is output
func TestTm2_panic_index_clear_eol_while_output(t *testing.T) {
	testStdCapacity(t)
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	// Zig: assert(!screen.cursor.semantic_content_clear_eol), in the
	// `else` branch taken when semantic_content == .output.
	term.Screens.Active.Cursor.SemanticContentClearEol = true
	tm2ExpectPanic(t, func() { _ = term.Index() })
}

// Go-only: panic test: Terminal.insertBlanks: cursor on a spacer tail at column 0
func TestTm2_panic_insertBlanks_spacer_tail_at_column_zero(t *testing.T) {
	testStdCapacity(t)
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	// Zig: assert(self.screens.active.cursor.x > 0), when the cursor cell
	// is a spacer tail.
	term.Screens.Active.Cursor.PageCell.setWide(PageCellWideSpacerTail)
	tm2ExpectPanic(t, func() { term.InsertBlanks(1) })
}

// Go-only: panic test: Terminal.insertBlanks: wide cell at the end of the shifted range is not followed by a spacer tail
func TestTm2_panic_insertBlanks_wide_without_spacer_tail(t *testing.T) {
	testStdCapacity(t)
	term := tm2Init(t, terminalOptionsDefault(10, 5))
	defer term.Deinit()

	// Zig: assert(end_multi[1].wide == .spacer_tail). With the cursor at
	// x = 0 and count 1, the last shifted cell is x = 8.
	lc := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Active(8, 0)))
	lc.Cell.setWide(PageCellWideWide)
	tm2ExpectPanic(t, func() { term.InsertBlanks(1) })
}

// Go-only: panic test: Terminal.deleteLines: spacer tail in the left margin column 0 (left - 1 overflows)
func TestTm2_panic_deleteLines_spacer_tail_at_left_margin_zero(t *testing.T) {
	testStdCapacity(t)
	term := tm2Init(t, terminalOptionsDefault(5, 5))
	defer term.Deinit()

	// Zig: rowWillBeShifted reads `cells[self.scrolling_region.left - 1]`
	// when the left margin cell is a spacer tail; with left = 0 the u16
	// subtraction overflows, which traps.
	lc := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm2Active(0, 0)))
	lc.Cell.setWide(PageCellWideSpacerTail)
	tm2ExpectPanic(t, func() { term.DeleteLines(1) })
}
