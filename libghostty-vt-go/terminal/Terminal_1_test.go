// Tests translated one for one from src/terminal/Terminal.zig (Ghostty
// v1.3.1 plus the term51 additional tests), lines 3111 to 7155: the print,
// cursor movement, margin, insertLines, scrollUp and scrollDown tests. Each
// Go test carries the exact Zig title in a "// Zig: test" comment. Helpers
// are prefixed tm1.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(tt, x, "<the Zig expression>"), and
//     expectEqual / expectEqualStrings / expectEqualSlices keep Zig's names
//     and Zig's (expected, actual) argument order. `try` on a call that
//     returns an error is zigTry(tt, err); `x.?` is zigUnwrap. The testing
//     parameter is `t`, which scripts/check-test-mapping.py requires, so the
//     Terminal that Zig's tests call `t` is called `term`. (The helpers take
//     the testing parameter as `tt`.)
//   - `var t = try init(alloc, .{ .cols = c, .rows = r })` is `term := tm1Init(t, c, r)`,
//     which starts from terminalOptionsDefault (max_scrollback 10_000, the
//     default colors and modes); with a max_scrollback it is
//     tm1InitScrollback. `t.plainString(alloc)` is tm1PlainString(t, term).
//   - `.{ .screen = .{ .x = x, .y = y } }` and its `.active` and `.viewport`
//     forms are tm1Screen, tm1Active and tm1Viewport.
//   - `for ("hello") |c| try t.print(c)` ranges over the bytes of the string,
//     as Zig does.
//   - Zig's `alloc` arguments are dropped; `defer alloc.free(str)` leak
//     checks are dropped with `// Zig-only:` comments.
//   - Every test calls testStdCapacity first (Zig's unit tests run with
//     grapheme_bytes 512).
//   - `@embedFile("res/glitch.txt")` is the constant tm1GlitchText, the bytes
//     of that file.
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt.

package terminal

import "testing"

// tm1ExpectPanic runs f, which must panic. A panic with the wireframe's
// "not implemented" value does not count: it only means the function under
// test does not exist yet. The setup is done outside f.
func tm1ExpectPanic(t *testing.T, f func()) {
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

// tm1Init is `try Terminal.init(alloc, .{ .cols = cols, .rows = rows })`;
// the terminal is returned by pointer (the methods take a pointer receiver).
func tm1Init(tt *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) *Terminal {
	tt.Helper()
	t, err := TerminalInit(terminalOptionsDefault(cols, rows))
	zigTry(tt, err)
	return &t
}

// tm1InitScrollback is `try Terminal.init(alloc, .{ .cols = cols, .rows =
// rows, .max_scrollback = max_scrollback })`.
func tm1InitScrollback(tt *testing.T, cols SizeCellCountInt, rows SizeCellCountInt, max_scrollback uint64) *Terminal {
	tt.Helper()
	opts := terminalOptionsDefault(cols, rows)
	opts.MaxScrollback = max_scrollback
	t, err := TerminalInit(opts)
	zigTry(tt, err)
	return &t
}

// tm1PlainString is `try t.plainString(alloc)` as a Go string.
func tm1PlainString(tt *testing.T, t *Terminal) string {
	tt.Helper()
	str, err := t.PlainString()
	zigTry(tt, err)
	return string(str)
}

// tm1Int is the integer types the point helpers take, so that untyped
// constants, loop variables and size values all work as in Zig.
type tm1Int interface {
	~int | ~uint | ~uint16 | ~uint32
}

// tm1Screen, tm1Active, tm1Viewport and tm1History are the Zig literals
// `.{ .screen = .{ .x = x, .y = y } }` and so on.
func tm1Screen[X, Y tm1Int](x X, y Y) Point {
	return Point{Tag: PointTagScreen, Screen: PointCoordinate{X: SizeCellCountInt(x), Y: uint32(y)}}
}

func tm1Active[X, Y tm1Int](x X, y Y) Point {
	return Point{Tag: PointTagActive, Active: PointCoordinate{X: SizeCellCountInt(x), Y: uint32(y)}}
}

func tm1Viewport[X, Y tm1Int](x X, y Y) Point {
	return Point{Tag: PointTagViewport, Viewport: PointCoordinate{X: SizeCellCountInt(x), Y: uint32(y)}}
}

// tm1GlitchText is @embedFile("res/glitch.txt").
const tm1GlitchText = "" +
	"G\xcc\xb4\xcc\x84\xcd\x91\xcc\x8a\xcc\x8c\xcc\x87\xcc\x91\xcd\x92\xcd" +
	"\x8b\xcd\x91\xcc\x84\xcd\x9d\xcd\x9d\xcd\x84\xcd\x90\xcd\x84\xcd\x81\xcc" +
	"\xbd\xcd\x8c\xcd\x82\xcc\x8e\xcd\x98\xcc\x80\xcc\x94\xcd\x8b\xcc\x95\xcc" +
	"\x93\xcc\x81\xcc\x85\xcc\x8c\xcc\x87\xcd\x8e\xcd\x8d\xcc\x9c\xcd\x8e\xcd" +
	"\x94\xcd\x9c\xcd\x95\xcc\xa9\xcc\x97\xcd\x95\xcd\x96\xcc\x9f\xcc\x9ch" +
	"\xcc\xb6\xcd\x91\xcc\xbe\xcd\x9b\xcc\x92\xcc\x86\xcc\x92\xcc\x89\xcd\x98" +
	"\xcc\x92\xcc\x91\xcd\x82\xcc\x84\xcc\x9e\xcd\x85\xcc\xab\xcd\x89\xcc\xb3" +
	"\xcc\xac\xcc\x9c\xcc\xb1\xcc\xa1\xcc\xa5\xcd\x95o\xcc\xb7\xcd\xa0\xcd" +
	"\x92\xcc\x82\xcc\x91\xcd\x92\xcc\xa5\xcd\x94\xcd\x94\xcd\x96\xcc\xad\xcd" +
	"\x9a\xcd\x99\xcc\xa8\xcc\xa8\xcc\xaf\xcc\x9f\xcc\xad\xcc\x98\xcd\x87\xcc" +
	"\xab\xcc\xb0\xcd\x9a\xcc\xba\xcc\xb3\xcc\x99\xcd\x9c\xcc\xb3\xcc\x9f\xcd" +
	"\x9a\xcc\xab\xcc\xb1\xcc\xb9\xcd\x85\xcc\xb1s\xcc\xb4\xcc\x85\xcd\x92" +
	"\xcc\x80\xcc\x8f\xcd\x86\xcd\xa0\xcc\x95\xcc\x90\xcc\x8b\xcd\x82\xcd\x98" +
	"\xcd\x83\xcc\x9a\xcd\x8b\xcc\x83\xcc\x88\xcc\x94\xcd\x98\xcc\x82\xcc\x88" +
	"\xcd\x98\xcd\x9b\xcd\xa0\xcc\x90\xcc\xbf\xcc\x94\xcc\x81\xcc\x94\xcc\x84" +
	"\xcd\x91\xcd\x98\xcc\x87\xcd\x91\xcc\x8b\xcd\x84\xcd\x8c\xcd\x8b\xcc\xbe" +
	"\xcc\x83\xcc\xbd\xcd\x84\xcc\x96\xcc\xa7\xcc\xb0\xcc\x9c\xcc\xb1\xcc\xb9" +
	"\xcd\x99t\xcc\xb5\xcd\x8b\xcc\x8a\xcc\x80\xcd\x90\xcc\x8d\xcc\x85\xcc" +
	"\x80\xcc\x84\xcc\x9a\xcc\x83\xcd\x9d\xcd\x84\xcc\x94\xcc\x87\xcd\x84\xcc" +
	"\x84\xcc\x83\xcc\xbd\xcc\x82\xcc\x8c\xcc\x85\xcc\x84\xcc\x8b\xcd\x92\xcc" +
	"\x83\xcd\x84\xcc\x8d\xcd\x80\xcc\x8d\xcd\x9d\xcd\x9d\xcc\x87\xcc\xbd\xcc" +
	"\x90\xcd\x8a\xcc\xbe\xcd\x9d\xcc\x86\xcc\x85\xcc\x88\xcc\xbf\xcd\x83\xcd" +
	"\x92\xcc\x84\xcd\x9d\xcc\xbe\xcd\x8c\xcc\x9c\xcc\xa2\xcc\xb1\xcc\xa6\xcd" +
	"\x87\xcd\x89\xcc\xac\xcc\xae\xcc\xbc\xcd\x96\xcc\xb3\xcc\x97\xcc\xa5\xcc" +
	"\x9d\xcc\xac\xcd\x87\xcd\x95\xcc\xa5\xcc\x9c\xcd\x95\xcc\xb3\xcc\xb1\xcc" +
	"\xa5\xcc\xae\xcd\x89\xcc\xae\xcc\xa9\xcc\x98\xcc\xb0\xcc\xaa\xcc\xa4\xcd" +
	"\x89\xcd\x8e\xcc\xb2\xcd\x88\xcd\x8d\xcc\xb3\xcc\x9f\xcc\xa0\xcd\x88\xcc" +
	"\x9d\xcc\xabt\xcc\xb4\xcc\x8d\xcd\x86\xcd\x92\xcc\x8a\xcc\x92\xcc\x94" +
	"\xcc\x8a\xcd\x84\xcd\xa0\xcc\xbf\xcd\xa0\xcc\x8a\xcc\x85\xcd\x82\xcc\x90" +
	"\xcd\x8b\xcc\xbf\xcd\x82\xcc\x88\xcc\x92\xcc\x84\xcc\xa5\xcc\xbc\xcc\xb3" +
	"\xcc\x97\xcc\xac\xcc\xac\xcd\x94\xcd\x8e\xcd\x9c\xcc\xaf\xcd\x89\xcd\x87" +
	"\xcc\xae\xcc\xb0\xcd\x96\xcd\x87\xcc\x9d\xcd\x94\xcc\xb3\xcc\xb3\xcc\x97" +
	"\xcc\xb0\xcd\x87\xcd\x8e\xcd\x89\xcc\xac\xcd\x87\xcc\x9d\xcc\xba\xcc\xaf" +
	"\xcd\x8e\xcd\x96\xcd\x94y\xcc\xb4\xcc\x88\xcd\x9b\xcc\x92\xcc\x82\xcd" +
	"\x8c\xcc\x8c\xcd\x98\xcd\x98\xcc\x8f\xcd\xa0\xcc\x88\xcc\xbe\xcd\x83\xcd" +
	"\x84\xcc\xbf\xcd\x90\xcc\x95\xcc\x82\xcc\x93\xcc\x94\xcd\x83\xcc\x82\xcd" +
	"\x84\xcd\x91\xcd\x9d\xcd\x9b\xcd\x8a\xcd\x8b\xcc\x94\xcc\xbf\xcd\x98\xcc" +
	"\x8a\xcd\x91\xcd\x8c\xcc\x8a\xcc\x8f\xcc\x97\xcc\x9c\xcc\xa5\xcd\x87\xcd" +
	"\x96\xcc\xb0\xcd\x8e\xcc\x9d\xcc\xb9\xcc\x97\xcc\xaa\xcc\x99\xcc\x9e\xcc" +
	"\xa3\xcc\xb3\xcd\x8e\xcc\xaf\xcc\xb9\xcd\x9a\xcc\xb2\xcc\x9d\xcc\x97\xcc" +
	"\xb3\xcc\xb3\xcc\x97\xcc\x96\xcd\x8e\xcc\x97\xcc\xac\xcd\x88\xcd\x99\xcc" +
	"\x9d\xcc\x9f\xcd\x8d\xcc\xa5\xcc\xa4\xcc\xa2\xcd\x96\xcd\x87\xcc\xb0\xcd" +
	"\x88\xcc\xba\x0a"

// tm1FirstPage is `t.screens.active.pages.pages.first.?`.
func tm1FirstPage(tt *testing.T, t *Terminal) *PageListNode {
	tt.Helper()
	node := t.Screens.Active.Pages.pages.first
	expect(tt, node != nil, "t.screens.active.pages.pages.first.?")
	return node
}

// Zig: test "Terminal: input with no control characters"
func TestTm1_Terminal_input_with_no_control_characters(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 40, 40)
	defer term.Deinit()

	// Basic grid writing
	for _, c := range []uint8("hello") {
		zigTry(t, term.Print(uint32(c)))
	}
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 5, term.Screens.Active.Cursor.X)
	{
		str := tm1PlainString(t, term)
		// Zig-only: defer alloc.free(str);
		expectEqualStrings(t, "hello", str)
	}

	// The first row should be dirty
	expect(t, term.isDirty(tm1Screen(5, 0)), "t.isDirty(.{ .screen = .{ .x = 5, .y = 0 } })")
	expect(t, !term.isDirty(tm1Screen(5, 1)), "!t.isDirty(.{ .screen = .{ .x = 5, .y = 1 } })")
}

// Zig: test "Terminal: input with basic wraparound"
func TestTm1_Terminal_input_with_basic_wraparound(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 40)
	defer term.Deinit()

	// Basic grid writing
	for _, c := range []uint8("helloworldabc12") {
		zigTry(t, term.Print(uint32(c)))
	}
	expectEqual(t, 2, term.Screens.Active.Cursor.Y)
	expectEqual(t, 4, term.Screens.Active.Cursor.X)
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	{
		str := tm1PlainString(t, term)
		// Zig-only: defer alloc.free(str);
		expectEqualStrings(t, "hello\nworld\nabc12", str)
	}
}

// Zig: test "Terminal: input with basic wraparound dirty"
func TestTm1_Terminal_input_with_basic_wraparound_dirty(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 40)
	defer term.Deinit()

	for _, c := range []uint8("hello") {
		zigTry(t, term.Print(uint32(c)))
	}
	expect(t, term.isDirty(tm1Screen(4, 0)), "t.isDirty(.{ .screen = .{ .x = 4, .y = 0 } })")
	term.clearDirty()
	zigTry(t, term.Print('w'))

	// Old row is dirty because cursor moved from there
	expect(t, term.isDirty(tm1Screen(4, 0)), "t.isDirty(.{ .screen = .{ .x = 4, .y = 0 } })")
	expect(t, term.isDirty(tm1Screen(0, 1)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 1 } })")
}

// Zig: test "Terminal: input that forces scroll"
func TestTm1_Terminal_input_that_forces_scroll(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 1, 5)
	defer term.Deinit()

	// Basic grid writing
	for _, c := range []uint8("abcdef") {
		zigTry(t, term.Print(uint32(c)))
	}
	expectEqual(t, 4, term.Screens.Active.Cursor.Y)
	expectEqual(t, 0, term.Screens.Active.Cursor.X)
	{
		str := tm1PlainString(t, term)
		// Zig-only: defer alloc.free(str);
		expectEqualStrings(t, "b\nc\nd\ne\nf", str)
	}
}

// Zig: test "Terminal: input unique style per cell"
func TestTm1_Terminal_input_unique_style_per_cell(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 30, 30)
	defer term.Deinit()

	for y := 0; y < int(term.Rows); y++ {
		for x := 0; x < int(term.Cols); x++ {
			term.SetCursorPos(uint(y), uint(x))
			zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: uint8(x), G: uint8(y), B: 0}}))
			zigTry(t, term.Print('x'))
		}
	}
}

// Zig: test "Terminal: input glitch text"
func TestTm1_Terminal_input_glitch_text(t *testing.T) {
	testStdCapacity(t)
	glitch := tm1GlitchText
	term := tm1Init(t, 30, 30)
	defer term.Deinit()

	// Get our initial grapheme capacity.
	grapheme_cap := func() SizeGraphemeBytesInt {
		page := tm1FirstPage(t, term)
		return page.data.capacity.grapheme_bytes
	}()

	// Print glitch text until our capacity changes
	for {
		page := tm1FirstPage(t, term)
		if page.data.capacity.grapheme_bytes != grapheme_cap {
			break
		}
		zigTry(t, term.PrintString([]uint8(glitch)))
	}

	// We're testing to make sure that grapheme capacity gets increased.
	page := tm1FirstPage(t, term)
	expect(t, page.data.capacity.grapheme_bytes > grapheme_cap, "page.data.capacity.grapheme_bytes > grapheme_cap")
}

// Zig: test "Terminal: zero-width character at start"
func TestTm1_Terminal_zero_width_character_at_start(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	// This used to crash the terminal. This is not allowed so we should
	// just ignore it.
	zigTry(t, term.Print(0x200D))

	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 0, term.Screens.Active.Cursor.X)

	// Should not be dirty since we changed nothing.
	expect(t, !term.isDirty(tm1Screen(0, 0)), "!t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
}

// https://github.com/mitchellh/ghostty/issues/1400
// Zig: test "Terminal: print single very long line"
func TestTm1_Terminal_print_single_very_long_line(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	// This would crash for issue 1400. So the assertion here is
	// that we simply do not crash.
	for i := 0; i < 1000; i++ {
		zigTry(t, term.Print('x'))
	}
}

// Zig: test "Terminal: print wide char"
func TestTm1_Terminal_print_wide_char(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	zigTry(t, term.Print(0x1F600)) // Smiley face
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 2, term.Screens.Active.Cursor.X)

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0x1F600, cell.content().codepoint())
		expectEqual(t, PageCellWideWide, cell.Wide())
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(1, 0)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideSpacerTail, cell.Wide())
	}

	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
}

// Zig: test "Terminal: print wide char at edge creates spacer head"
func TestTm1_Terminal_print_wide_char_at_edge_creates_spacer_head(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 10, 10)
	defer term.Deinit()

	term.SetCursorPos(1, 10)
	zigTry(t, term.Print(0x1F600)) // Smiley face
	expectEqual(t, 1, term.Screens.Active.Cursor.Y)
	expectEqual(t, 2, term.Screens.Active.Cursor.X)

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(9, 0)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideSpacerHead, cell.Wide())
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 1)))
		cell := list_cell.Cell
		expectEqual(t, 0x1F600, cell.content().codepoint())
		expectEqual(t, PageCellWideWide, cell.Wide())
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(1, 1)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideSpacerTail, cell.Wide())
	}

	// Our first row just had a spacer head added which does not affect
	// rendering so only the place where the wide char was printed
	// should be marked.
	// BUT old row is dirty because cursor moved from there
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm1Screen(0, 1)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 1 } })")
}

// Zig: test "Terminal: print wide char with 1-column width"
func TestTm1_Terminal_print_wide_char_with_1_column_width(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 1, 2)
	defer term.Deinit()

	zigTry(t, term.Print('😀')) // 0x1F600

	// This prints a space so we should be dirty.
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
}

// Zig: test "Terminal: print wide char in single-width terminal"
func TestTm1_Terminal_print_wide_char_in_single_width_terminal(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 1, 80)
	defer term.Deinit()

	zigTry(t, term.Print(0x1F600)) // Smiley face
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 0, term.Screens.Active.Cursor.X)
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0, cell.content().codepoint())
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}

	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
}

// Zig: test "Terminal: print over wide char at 0,0"
func TestTm1_Terminal_print_over_wide_char_at_0_0(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	zigTry(t, term.Print(0x1F600)) // Smiley face
	term.SetCursorPos(0, 0)
	zigTry(t, term.Print('A'))

	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 1, term.Screens.Active.Cursor.X)

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, 'A', cell.content().codepoint())
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(1, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0, cell.content().codepoint())
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}

	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
	expect(t, !term.isDirty(tm1Screen(0, 1)), "!t.isDirty(.{ .screen = .{ .x = 0, .y = 1 } })")
}

// Zig: test "Terminal: print over wide char at col 0 corrupts previous row"
func TestTm1_Terminal_print_over_wide_char_at_col_0_corrupts_previous_row(t *testing.T) {
	testStdCapacity(t)
	// Crash found by AFL++ fuzzer (afl-out/stream/default/crashes/id:000002).
	//
	// printCell, when overwriting a wide cell with a narrow cell at x<=1
	// and y>0, sets the last cell of the previous row to .narrow — even
	// when that cell is a .spacer_tail rather than a .spacer_head. This
	// orphans the .wide cell at cols-2.
	term := tm1Init(t, 10, 3)
	defer term.Deinit()

	// Fill rows 0 and 1 with wide chars (5 per row on a 10-col terminal).
	for i := 0; i < 10; i++ {
		zigTry(t, term.Print(0x4E2D))
	}

	// Move cursor to row 1, col 0 (on top of a wide char) and print a
	// narrow character. This triggers printCell's .wide branch which
	// corrupts row 0's last cell: col 9 changes from .spacer_tail to
	// .narrow, orphaning the .wide at col 8.
	term.SetCursorPos(2, 1)
	zigTry(t, term.Print('A'))

	// Row 1, col 0 should be narrow (we just overwrote the wide char).
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 1)))
		expectEqual(t, PageCellWideNarrow, list_cell.Cell.Wide())
	}
	// Row 0, col 8 should still be .wide (the last wide char on the row).
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(8, 0)))
		expectEqual(t, PageCellWideWide, list_cell.Cell.Wide())
	}
	// Row 0, col 9 must remain .spacer_tail to pair with the .wide at col 8.
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(9, 0)))
		expectEqual(t, PageCellWideSpacerTail, list_cell.Cell.Wide())
	}
}

// Zig: test "Terminal: print over wide spacer tail"
func TestTm1_Terminal_print_over_wide_spacer_tail(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.Print('橋'))
	term.SetCursorPos(1, 2)
	zigTry(t, term.Print('X'))

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0, cell.content().codepoint())
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(1, 0)))
		cell := list_cell.Cell
		expectEqual(t, 'X', cell.content().codepoint())
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, " X", str)
	}

	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
}

// Zig: test "Terminal: print over wide char with bold"
func TestTm1_Terminal_print_over_wide_char_with_bold(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	zigTry(t, term.Print(0x1F600)) // Smiley face
	// verify we have styles in our style map
	{
		page := &term.Screens.Active.Cursor.PagePin.Node.data
		expectEqual(t, 1, page.styles.Count())
	}

	// Go back and overwrite with no style
	term.SetCursorPos(0, 0)
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))
	zigTry(t, term.Print('A')) // Smiley face

	// verify our style is gone
	{
		page := &term.Screens.Active.Cursor.PagePin.Node.data
		expectEqual(t, 0, page.styles.Count())
	}

	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
}

// Zig: test "Terminal: print over wide char with bg color"
func TestTm1_Terminal_print_over_wide_char_with_bg_color(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{R: 0xFF, G: 0, B: 0}}))
	zigTry(t, term.Print(0x1F600)) // Smiley face
	// verify we have styles in our style map
	{
		page := &term.Screens.Active.Cursor.PagePin.Node.data
		expectEqual(t, 1, page.styles.Count())
	}

	// Go back and overwrite with no style
	term.SetCursorPos(0, 0)
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))
	zigTry(t, term.Print('A')) // Smiley face

	// verify our style is gone
	{
		page := &term.Screens.Active.Cursor.PagePin.Node.data
		expectEqual(t, 0, page.styles.Count())
	}

	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
}

// Zig: test "Terminal: print multicodepoint grapheme, disabled mode 2027"
func TestTm1_Terminal_print_multicodepoint_grapheme_disabled_mode_2027(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	// https://github.com/mitchellh/ghostty/issues/289
	// This is: 👨‍👩‍👧 (which may or may not render correctly)
	zigTry(t, term.Print(0x1F468))
	zigTry(t, term.Print(0x200D))
	zigTry(t, term.Print(0x1F469))
	zigTry(t, term.Print(0x200D))
	zigTry(t, term.Print(0x1F467))

	// We should have 6 cells taken up
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 6, term.Screens.Active.Cursor.X)

	// Assert various properties about our screen to verify
	// we have all expected cells.
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0x1F468, cell.content().codepoint())
		expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
		expectEqual(t, PageCellWideWide, cell.Wide())
		cps := zigUnwrap(t, list_cell.Node.data.LookupGrapheme(cell))
		expectEqual(t, 1, len(cps))
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(1, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0, cell.content().codepoint())
		expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
		expectEqual(t, PageCellWideSpacerTail, cell.Wide())
		expect(t, !list_cell.Node.data.LookupGrapheme(cell).Valid, "list_cell.node.data.lookupGrapheme(cell) == null")
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(2, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0x1F469, cell.content().codepoint())
		expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
		expectEqual(t, PageCellWideWide, cell.Wide())
		cps := zigUnwrap(t, list_cell.Node.data.LookupGrapheme(cell))
		expectEqual(t, 1, len(cps))
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(3, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0, cell.content().codepoint())
		expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
		expectEqual(t, PageCellWideSpacerTail, cell.Wide())
		expect(t, !list_cell.Node.data.LookupGrapheme(cell).Valid, "list_cell.node.data.lookupGrapheme(cell) == null")
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(4, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0x1F467, cell.content().codepoint())
		expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
		expectEqual(t, PageCellWideWide, cell.Wide())
		expect(t, !list_cell.Node.data.LookupGrapheme(cell).Valid, "list_cell.node.data.lookupGrapheme(cell) == null")
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(5, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0, cell.content().codepoint())
		expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
		expectEqual(t, PageCellWideSpacerTail, cell.Wide())
		expect(t, !list_cell.Node.data.LookupGrapheme(cell).Valid, "list_cell.node.data.lookupGrapheme(cell) == null")
	}

	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
}

// Zig: test "Terminal: VS16 doesn't make character with 2027 disabled"
func TestTm1_Terminal_VS16_doesn_t_make_character_with_2027_disabled(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	// Disable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, false)

	zigTry(t, term.Print(0x2764)) // Heart
	zigTry(t, term.Print(0xFE0F)) // VS16 to make wide

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "❤️", str)
	}

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0x2764, cell.content().codepoint())
		expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
		expectEqual(t, PageCellWideNarrow, cell.Wide())
		cps := zigUnwrap(t, list_cell.Node.data.LookupGrapheme(cell))
		expectEqual(t, 1, len(cps))
	}
}

// Zig: test "Terminal: ignored VS16 doesn't mark dirty"
func TestTm1_Terminal_ignored_VS16_doesn_t_mark_dirty(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	// Disable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, false)

	zigTry(t, term.Print(0x2764)) // Heart
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")

	term.clearDirty()
	zigTry(t, term.Print(0xFE0F)) // VS16 to make wide
	expect(t, !term.isDirty(tm1Screen(0, 0)), "!t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
}

// Zig: test "Terminal: print invalid VS16 non-grapheme"
func TestTm1_Terminal_print_invalid_VS16_non_grapheme(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	// https://github.com/mitchellh/ghostty/issues/1482
	zigTry(t, term.Print('x'))
	zigTry(t, term.Print(0xFE0F))

	// We should have 1 narrow cell.
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 1, term.Screens.Active.Cursor.X)

	// Assert various properties about our screen to verify
	// we have all expected cells.
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, 'x', cell.content().codepoint())
		expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(1, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0, cell.content().codepoint())
	}
}

// Zig: test "Terminal: invalid VS16 doesn't mark dirty"
func TestTm1_Terminal_invalid_VS16_doesn_t_mark_dirty(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	// Disable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, false)

	zigTry(t, term.Print('x'))
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")

	term.clearDirty()
	zigTry(t, term.Print(0xFE0F)) // VS16 to make wide
	expect(t, !term.isDirty(tm1Screen(0, 0)), "!t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
}

// Zig: test "Terminal: print multicodepoint grapheme, mode 2027"
func TestTm1_Terminal_print_multicodepoint_grapheme_mode_2027(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	// https://github.com/mitchellh/ghostty/issues/289
	// This is: 👨‍👩‍👧 (which may or may not render correctly)
	zigTry(t, term.Print(0x1F468))
	zigTry(t, term.Print(0x200D))
	zigTry(t, term.Print(0x1F469))
	zigTry(t, term.Print(0x200D))
	zigTry(t, term.Print(0x1F467))

	// We should have 2 cells taken up. It is one character but "wide".
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 2, term.Screens.Active.Cursor.X)

	// Row should be dirty
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")

	// Assert various properties about our screen to verify
	// we have all expected cells.
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0x1F468, cell.content().codepoint())
		expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
		expectEqual(t, PageCellWideWide, cell.Wide())
		cps := zigUnwrap(t, list_cell.Node.data.LookupGrapheme(cell))
		expectEqual(t, 4, len(cps))
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(1, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0, cell.content().codepoint())
		expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
		expectEqual(t, PageCellWideSpacerTail, cell.Wide())
	}
}

// Zig: test "Terminal: keypad sequence VS15"
func TestTm1_Terminal_keypad_sequence_VS15(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	// This is: "#︎" (number sign with text presentation selector)
	zigTry(t, term.Print(0x23))   // # Number sign (valid base)
	zigTry(t, term.Print(0xFE0E)) // VS15 (text presentation selector)

	// VS15 should combine with the base character into a single grapheme cluster,
	// taking 1 cell (narrow character).
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 1, term.Screens.Active.Cursor.X)

	// Row should be dirty
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")

	// The base emoji should be in cell 0 with the skin tone as a grapheme
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0x23, cell.content().codepoint())
		expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}
}

// Zig: test "Terminal: keypad sequence VS16"
func TestTm1_Terminal_keypad_sequence_VS16(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	// This is: "#️" (number sign with emoji presentation selector)
	zigTry(t, term.Print(0x23))   // # Number sign (valid base)
	zigTry(t, term.Print(0xFE0F)) // VS16 (emoji presentation selector)

	// VS16 should combine with the base character into a single grapheme cluster,
	// taking 2 cells (wide character).
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 2, term.Screens.Active.Cursor.X)

	// Row should be dirty
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")

	// The base emoji should be in cell 0 with the skin tone as a grapheme
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0x23, cell.content().codepoint())
		expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
		expectEqual(t, PageCellWideWide, cell.Wide())
	}
}

// Zig: test "Terminal: Fitzpatrick skin tone next valid base"
func TestTm1_Terminal_Fitzpatrick_skin_tone_next_valid_base(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	// This is: "👋🏿" (waving hand with dark skin tone)
	zigTry(t, term.Print(0x1F44B)) // 👋 Waving hand (valid base)
	zigTry(t, term.Print(0x1F3FF)) // 🏿 Dark skin tone modifier

	// The skin tone should combine with the base emoji into a single grapheme cluster,
	// taking 2 cells (wide character).
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 2, term.Screens.Active.Cursor.X)

	// Row should be dirty
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")

	// The base emoji should be in cell 0 with the skin tone as a grapheme
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0x1F44B, cell.content().codepoint())
		expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
		expectEqual(t, PageCellWideWide, cell.Wide())
	}
}

// Zig: test "Terminal: Fitzpatrick skin tone next to non-base"
func TestTm1_Terminal_Fitzpatrick_skin_tone_next_to_non_base(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	// This is: "🏿" (which may not render correctly in your editor!)
	zigTry(t, term.Print(0x22))    // "
	zigTry(t, term.Print(0x1F3FF)) // Dark skin tone
	zigTry(t, term.Print(0x22))    // "

	// We should have 4 cells taken up. Importantly, the skin tone
	// should not join with the quotes.
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 4, term.Screens.Active.Cursor.X)

	// Row should be dirty
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")

	// Assert various properties about our screen to verify
	// we have all expected cells.
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0x22, cell.content().codepoint())
		expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(1, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0x1F3FF, cell.content().codepoint())
		expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
		expectEqual(t, PageCellWideWide, cell.Wide())
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(3, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0x22, cell.content().codepoint())
		expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}
}

// Zig: test "Terminal: multicodepoint grapheme marks dirty on every codepoint"
func TestTm1_Terminal_multicodepoint_grapheme_marks_dirty_on_every_codepoint(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	// https://github.com/mitchellh/ghostty/issues/289
	// This is: 👨‍👩‍👧 (which may or may not render correctly)
	zigTry(t, term.Print(0x1F468))
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
	term.clearDirty()
	zigTry(t, term.Print(0x200D))
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
	term.clearDirty()
	zigTry(t, term.Print(0x1F469))
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
	term.clearDirty()
	zigTry(t, term.Print(0x200D))
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
	term.clearDirty()
	zigTry(t, term.Print(0x1F467))
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")

	// We should have 2 cells taken up. It is one character but "wide".
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 2, term.Screens.Active.Cursor.X)
}

// Zig: test "Terminal: VS15 to make narrow character"
func TestTm1_Terminal_VS15_to_make_narrow_character(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.Print(0x2614)) // Umbrella with rain drops, width=2
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
	term.clearDirty()

	// We should have 2 cells taken up. It is one character but "wide".
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 2, term.Screens.Active.Cursor.X)

	zigTry(t, term.Print(0xFE0E)) // VS15 to make narrow
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
	term.clearDirty()

	// VS15 should send us back a cell since our char is no longer wide.
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 1, term.Screens.Active.Cursor.X)

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "☔︎", str)
	}

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0x2614, cell.content().codepoint())
		expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
		expectEqual(t, PageCellWideNarrow, cell.Wide())
		cps := zigUnwrap(t, list_cell.Node.data.LookupGrapheme(cell))
		expectEqual(t, 1, len(cps))
	}
}

// Zig: test "Terminal: VS15 on already narrow emoji"
func TestTm1_Terminal_VS15_on_already_narrow_emoji(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.Print(0x26C8)) // Thunder cloud and rain, width=1
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
	term.clearDirty()
	zigTry(t, term.Print(0xFE0E)) // VS15 to make narrow
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
	term.clearDirty()

	// Character takes up one cell
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 1, term.Screens.Active.Cursor.X)

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "⛈︎", str)
	}

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0x26C8, cell.content().codepoint())
		expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
		expectEqual(t, PageCellWideNarrow, cell.Wide())
		cps := zigUnwrap(t, list_cell.Node.data.LookupGrapheme(cell))
		expectEqual(t, 1, len(cps))
	}
}

// Zig: test "Terminal: print invalid VS15 following emoji is wide"
func TestTm1_Terminal_print_invalid_VS15_following_emoji_is_wide(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.Print('\U0001f9e0')) // 🧠
	zigTry(t, term.Print(0xFE0E))       // not valid with U+1F9E0 as base

	// We should have 2 cells taken up. It is one character but "wide".
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 2, term.Screens.Active.Cursor.X)

	// Assert various properties about our screen to verify
	// we have all expected cells.
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, '\U0001f9e0', cell.content().codepoint())
		expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
		expectEqual(t, PageCellWideWide, cell.Wide())
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(1, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0, cell.content().codepoint())
		expectEqual(t, PageCellWideSpacerTail, cell.Wide())
	}
}

// Zig: test "Terminal: print invalid VS15 in emoji ZWJ sequence"
func TestTm1_Terminal_print_invalid_VS15_in_emoji_ZWJ_sequence(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.Print('\U0001f469')) // 👩
	zigTry(t, term.Print(0xFE0E))       // not valid with U+1F469 as base
	zigTry(t, term.Print('\U0000200d')) // ZWJ
	zigTry(t, term.Print('\U0001f466')) // 👦

	// We should have 2 cells taken up. It is one character but "wide".
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 2, term.Screens.Active.Cursor.X)

	// Assert various properties about our screen to verify
	// we have all expected cells.
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, '\U0001f469', cell.content().codepoint())
		expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
		expectEqualSlices(t, []uint32{'\U0000200d', '\U0001f466'}, zigUnwrap(t, list_cell.Node.data.LookupGrapheme(cell)))
		expectEqual(t, PageCellWideWide, cell.Wide())
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(1, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0, cell.content().codepoint())
		expectEqual(t, PageCellWideSpacerTail, cell.Wide())
	}
}

// Zig: test "Terminal: VS15 to make narrow character with pending wrap"
func TestTm1_Terminal_VS15_to_make_narrow_character_with_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 4, 5)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	expect(t, term.Modes.Get(ModesModeWraparound), "t.modes.get(.wraparound)")

	zigTry(t, term.Print(0x1F34B)) // Lemon, width=2
	zigTry(t, term.Print(0x2614))  // Umbrella with rain drops, width=2
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
	term.clearDirty()

	// We only move to the end of the line because we're in a pending wrap
	// state.
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 3, term.Screens.Active.Cursor.X)
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")

	zigTry(t, term.Print(0xFE0E)) // VS15 to make narrow
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
	term.clearDirty()

	// VS15 should clear the pending wrap state
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 3, term.Screens.Active.Cursor.X)
	expect(t, !term.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "🍋☔︎", str)
	}

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(2, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0x2614, cell.content().codepoint())
		expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
		expectEqual(t, PageCellWideNarrow, cell.Wide())
		cps := zigUnwrap(t, list_cell.Node.data.LookupGrapheme(cell))
		expectEqual(t, 1, len(cps))
	}

	// VS15 should not affect the previous grapheme
	{
		lemon_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0))).Cell
		expectEqual(t, 0x1F34B, lemon_cell.content().codepoint())
		expectEqual(t, PageCellWideWide, lemon_cell.Wide())
		spacer_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(1, 0))).Cell
		expectEqual(t, 0, spacer_cell.content().codepoint())
		expectEqual(t, PageCellWideSpacerTail, spacer_cell.Wide())
	}
}

// Zig: test "Terminal: VS16 to make wide character on next line"
func TestTm1_Terminal_VS16_to_make_wide_character_on_next_line(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 3, 5)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	term.CursorRight(2)
	zigTry(t, term.Print('#'))
	expectEqual(t, 2, term.Screens.Active.Cursor.X)
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	expect(t, term.isDirty(tm1Screen(2, 0)), "t.isDirty(.{ .screen = .{ .x = 2, .y = 0 } })")
	term.clearDirty()

	zigTry(t, term.Print(0xFE0F)) // VS16 to make wide

	expect(t, term.isDirty(tm1Screen(2, 0)), "t.isDirty(.{ .screen = .{ .x = 2, .y = 0 } })")
	term.clearDirty()
	expectEqual(t, 1, term.Screens.Active.Cursor.Y)
	expectEqual(t, 2, term.Screens.Active.Cursor.X)
	expect(t, !term.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")

	{
		// Previous cell turns into spacer_head
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(2, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0, cell.content().codepoint())
		expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
		expectEqual(t, PageCellWideSpacerHead, cell.Wide())
	}
	{
		// '#' cell is wide
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 1)))
		cell := list_cell.Cell
		expectEqual(t, '#', cell.content().codepoint())
		expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
		expectEqualSlices(t, []uint32{0xFE0F}, zigUnwrap(t, list_cell.Node.data.LookupGrapheme(cell)))
		expectEqual(t, PageCellWideWide, cell.Wide())
	}
	{
		// spacer_tail
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(1, 1)))
		cell := list_cell.Cell
		expectEqual(t, 0, cell.content().codepoint())
		expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
		expectEqual(t, PageCellWideSpacerTail, cell.Wide())
	}
}

// Zig: test "Terminal: VS16 to make wide character on next line with hyperlink"
func TestTm1_Terminal_VS16_to_make_wide_character_on_next_line_with_hyperlink(t *testing.T) {
	testStdCapacity(t)
	// Regression test for the crash fixed in print's grapheme `.wide` path:
	// writing a spacer_head at the screen edge before row.wrap was set.
	term := tm1Init(t, 3, 5)
	defer term.Deinit()

	// Enable grapheme clustering and activate a hyperlink so printCell
	// calls cursorSetHyperlink (which runs page integrity checks).
	term.Modes.Set(ModesModeGraphemeCluster, true)
	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://example.com"), None[[]uint8]()))

	term.CursorRight(2)
	zigTry(t, term.Print('#'))
	expectEqual(t, 2, term.Screens.Active.Cursor.X)
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")

	// Without the fix, this panicked with UnwrappedSpacerHead.
	zigTry(t, term.Print(0xFE0F)) // VS16 to make wide

	expectEqual(t, 1, term.Screens.Active.Cursor.Y)
	expectEqual(t, 2, term.Screens.Active.Cursor.X)
	expect(t, !term.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")

	{
		// Previous cell turns into spacer_head and remains hyperlinked.
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(2, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0, cell.content().codepoint())
		expectEqual(t, PageCellWideSpacerHead, cell.Wide())
		expect(t, cell.Hyperlink(), "cell.hyperlink")
		expect(t, list_cell.Row.wrap, "list_cell.row.wrap")
	}
	{
		// '#' cell is now wide and still hyperlinked.
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 1)))
		cell := list_cell.Cell
		expectEqual(t, '#', cell.content().codepoint())
		expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
		expectEqualSlices(t, []uint32{0xFE0F}, zigUnwrap(t, list_cell.Node.data.LookupGrapheme(cell)))
		expectEqual(t, PageCellWideWide, cell.Wide())
		expect(t, cell.Hyperlink(), "cell.hyperlink")
	}
	{
		// spacer_tail inherits hyperlink as part of the same grapheme cell.
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(1, 1)))
		cell := list_cell.Cell
		expectEqual(t, 0, cell.content().codepoint())
		expectEqual(t, PageCellWideSpacerTail, cell.Wide())
		expect(t, cell.Hyperlink(), "cell.hyperlink")
	}
}

// Zig: test "Terminal: VS16 to make wide character with pending wrap"
func TestTm1_Terminal_VS16_to_make_wide_character_with_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 3, 5)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	term.CursorRight(1)
	zigTry(t, term.Print('#'))
	expectEqual(t, 2, term.Screens.Active.Cursor.X)
	expect(t, !term.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")

	zigTry(t, term.Print(0xFE0F)) // VS16 to make wide

	expectEqual(t, 2, term.Screens.Active.Cursor.X)
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")

	{
		// '#' cell is wide
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(1, 0)))
		cell := list_cell.Cell
		expectEqual(t, '#', cell.content().codepoint())
		expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
		expectEqualSlices(t, []uint32{0xFE0F}, zigUnwrap(t, list_cell.Node.data.LookupGrapheme(cell)))
		expectEqual(t, PageCellWideWide, cell.Wide())
	}
	{
		// spacer_tail
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(2, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0, cell.content().codepoint())
		expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
		expectEqual(t, PageCellWideSpacerTail, cell.Wide())
	}
}

// Zig: test "Terminal: VS16 to make wide character with mode 2027"
func TestTm1_Terminal_VS16_to_make_wide_character_with_mode_2027(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.Print(0x2764)) // Heart
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
	term.clearDirty()
	zigTry(t, term.Print(0xFE0F)) // VS16 to make wide
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
	term.clearDirty()

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "❤️", str)
	}

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0x2764, cell.content().codepoint())
		expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
		expectEqual(t, PageCellWideWide, cell.Wide())
		cps := zigUnwrap(t, list_cell.Node.data.LookupGrapheme(cell))
		expectEqual(t, 1, len(cps))
	}
}

// Zig: test "Terminal: VS16 repeated with mode 2027"
func TestTm1_Terminal_VS16_repeated_with_mode_2027(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.Print(0x2764)) // Heart
	zigTry(t, term.Print(0xFE0F)) // VS16 to make wide
	zigTry(t, term.Print(0x2764)) // Heart
	zigTry(t, term.Print(0xFE0F)) // VS16 to make wide

	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "❤️❤️", str)
	}

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0x2764, cell.content().codepoint())
		expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
		expectEqual(t, PageCellWideWide, cell.Wide())
		cps := zigUnwrap(t, list_cell.Node.data.LookupGrapheme(cell))
		expectEqual(t, 1, len(cps))
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(2, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0x2764, cell.content().codepoint())
		expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
		expectEqual(t, PageCellWideWide, cell.Wide())
		cps := zigUnwrap(t, list_cell.Node.data.LookupGrapheme(cell))
		expectEqual(t, 1, len(cps))
	}
}

// Zig: test "Terminal: print invalid VS16 grapheme"
func TestTm1_Terminal_print_invalid_VS16_grapheme(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	// https://github.com/mitchellh/ghostty/issues/1482
	zigTry(t, term.Print('x'))
	zigTry(t, term.Print(0xFE0F)) // invalid VS16

	// We should have 1 cells taken up, and narrow.
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 1, term.Screens.Active.Cursor.X)

	// Assert various properties about our screen to verify
	// we have all expected cells.
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, 'x', cell.content().codepoint())
		expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(1, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0, cell.content().codepoint())
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}
}

// Zig: test "Terminal: print invalid VS16 with second char"
func TestTm1_Terminal_print_invalid_VS16_with_second_char(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	// https://github.com/mitchellh/ghostty/issues/1482
	zigTry(t, term.Print('x'))
	zigTry(t, term.Print(0xFE0F))
	zigTry(t, term.Print('y'))

	// We should have 2 cells taken up, from two separate narrow characters.
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 2, term.Screens.Active.Cursor.X)

	// Assert various properties about our screen to verify
	// we have all expected cells.
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, 'x', cell.content().codepoint())
		expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(1, 0)))
		cell := list_cell.Cell
		expectEqual(t, 'y', cell.content().codepoint())
		expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}
}

// Zig: test "Terminal: print grapheme ò (o with nonspacing mark) should be narrow"
func TestTm1_Terminal_print_grapheme_o_o_with_nonspacing_mark_should_be_narrow(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.Print('o'))
	zigTry(t, term.Print(0x0300)) // combining grave accent

	// We should have 1 cell taken up.
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 1, term.Screens.Active.Cursor.X)

	// Assert various properties about our screen to verify
	// we have all expected cells.
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, 'o', cell.content().codepoint())
		expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
		expectEqualSlices(t, []uint32{0x0300}, zigUnwrap(t, list_cell.Node.data.LookupGrapheme(cell)))
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}
}

// Zig: test "Terminal: print Devanagari grapheme should be wide"
func TestTm1_Terminal_print_Devanagari_grapheme_should_be_wide(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	// क्‍ष
	zigTry(t, term.Print(0x0915))
	zigTry(t, term.Print(0x094D))
	zigTry(t, term.Print(0x200D))
	zigTry(t, term.Print(0x0937))

	// We should have 2 cells taken up. It is one character but "wide".
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 2, term.Screens.Active.Cursor.X)

	// Assert various properties about our screen to verify
	// we have all expected cells.
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0x0915, cell.content().codepoint())
		expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
		expectEqualSlices(t, []uint32{0x094D, 0x200D, 0x0937}, zigUnwrap(t, list_cell.Node.data.LookupGrapheme(cell)))
		expectEqual(t, PageCellWideWide, cell.Wide())
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(1, 0)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideSpacerTail, cell.Wide())
	}
}

// Zig: test "Terminal: print Devanagari grapheme should be wide on next line"
func TestTm1_Terminal_print_Devanagari_grapheme_should_be_wide_on_next_line(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 3, 5)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	term.CursorRight(2)

	// क्‍ष
	zigTry(t, term.Print(0x0915))
	zigTry(t, term.Print(0x094D))
	zigTry(t, term.Print(0x200D))
	expectEqual(t, 2, term.Screens.Active.Cursor.X)
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")

	// This one increases the width to wide
	zigTry(t, term.Print(0x0937))

	// We should have 2 cells taken up. It is one character but "wide".
	expectEqual(t, 1, term.Screens.Active.Cursor.Y)
	expectEqual(t, 2, term.Screens.Active.Cursor.X)
	expect(t, !term.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")

	{
		// Previous cell turns into spacer_head
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(2, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0, cell.content().codepoint())
		expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
		expectEqual(t, PageCellWideSpacerHead, cell.Wide())
	}
	{
		// Devanagari grapheme is wide
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 1)))
		cell := list_cell.Cell
		expectEqual(t, 0x0915, cell.content().codepoint())
		expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
		expectEqualSlices(t, []uint32{0x094D, 0x200D, 0x0937}, zigUnwrap(t, list_cell.Node.data.LookupGrapheme(cell)))
		expectEqual(t, PageCellWideWide, cell.Wide())
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(1, 1)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideSpacerTail, cell.Wide())
	}
}

// Zig: test "Terminal: print Devanagari grapheme should be wide on next page"
func TestTm1_Terminal_print_Devanagari_grapheme_should_be_wide_on_next_page(t *testing.T) {
	testStdCapacity(t)
	rows := PageStdCapacity.rows
	cols := PageStdCapacity.cols
	term := tm1Init(t, cols, rows)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	term.CursorDown(uint(rows - 1))

	for i, end := int(rows), int(tm1FirstPage(t, term).data.capacity.rows); i < end; i++ {
		zigTry(t, term.Index())
	}

	term.CursorRight(uint(cols - 1))

	expectEqual(t, cols-1, term.Screens.Active.Cursor.X)
	expectEqual(t, rows-1, term.Screens.Active.Cursor.Y)

	// क्‍ष
	zigTry(t, term.Print(0x0915))
	zigTry(t, term.Print(0x094D))
	zigTry(t, term.Print(0x200D))
	expectEqual(t, cols-1, term.Screens.Active.Cursor.X)
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")

	// This one increases the width to wide
	zigTry(t, term.Print(0x0937))

	// We should have 2 cells taken up. It is one character but "wide".
	expectEqual(t, rows-1, term.Screens.Active.Cursor.Y)
	expectEqual(t, 2, term.Screens.Active.Cursor.X)
	expect(t, !term.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")

	{
		// Previous cell turns into spacer_head
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Active(cols-1, rows-2)))
		cell := list_cell.Cell
		expectEqual(t, 0, cell.content().codepoint())
		expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
		expectEqual(t, PageCellWideSpacerHead, cell.Wide())
	}
	{
		// Devanagari grapheme is wide
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Active(0, rows-1)))
		cell := list_cell.Cell
		expectEqual(t, 0x0915, cell.content().codepoint())
		expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
		expectEqualSlices(t, []uint32{0x094D, 0x200D, 0x0937}, zigUnwrap(t, list_cell.Node.data.LookupGrapheme(cell)))
		expectEqual(t, PageCellWideWide, cell.Wide())
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Active(1, rows-1)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideSpacerTail, cell.Wide())
	}
}

// Zig: test "Terminal: print invalid VS16 with second char (combining)"
func TestTm1_Terminal_print_invalid_VS16_with_second_char_combining(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	// https://github.com/mitchellh/ghostty/issues/1482
	zigTry(t, term.Print('n'))
	zigTry(t, term.Print(0xFE0F)) // invalid VS16
	zigTry(t, term.Print(0x0303)) // combining tilde

	// We should have 1 cells taken up, and narrow.
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 1, term.Screens.Active.Cursor.X)

	// Assert various properties about our screen to verify
	// we have all expected cells.
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, 'n', cell.content().codepoint())
		expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
		expectEqualSlices(t, []uint32{'\U00000303'}, zigUnwrap(t, list_cell.Node.data.LookupGrapheme(cell)))
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(1, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0, cell.content().codepoint())
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}
}

// Zig: test "Terminal: overwrite grapheme should clear grapheme data"
func TestTm1_Terminal_overwrite_grapheme_should_clear_grapheme_data(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.Print(0x26C8)) // Thunder cloud and rain
	zigTry(t, term.Print(0xFE0E)) // VS15 to make narrow
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
	term.clearDirty()

	term.SetCursorPos(1, 1)
	zigTry(t, term.Print('A'))
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "A", str)
	}

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		cell := list_cell.Cell
		expectEqual(t, 'A', cell.content().codepoint())
		expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}
}

// Zig: test "Terminal: overwrite multicodepoint grapheme clears grapheme data"
func TestTm1_Terminal_overwrite_multicodepoint_grapheme_clears_grapheme_data(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 10, 10)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	// https://github.com/mitchellh/ghostty/issues/289
	// This is: 👨‍👩‍👧 (which may or may not render correctly)
	zigTry(t, term.Print(0x1F468))
	zigTry(t, term.Print(0x200D))
	zigTry(t, term.Print(0x1F469))
	zigTry(t, term.Print(0x200D))
	zigTry(t, term.Print(0x1F467))

	// We should have 2 cells taken up. It is one character but "wide".
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 2, term.Screens.Active.Cursor.X)

	// We should have one cell with graphemes
	page := &term.Screens.Active.Cursor.PagePin.Node.data
	expectEqual(t, 1, page.GraphemeCount())

	// Move back and overwrite wide
	term.SetCursorPos(1, 1)
	term.clearDirty()
	zigTry(t, term.Print('X'))
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")

	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 1, term.Screens.Active.Cursor.X)
	expectEqual(t, 0, page.GraphemeCount())

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "X", str)
	}
}

// Zig: test "Terminal: overwrite multicodepoint grapheme tail clears grapheme data"
func TestTm1_Terminal_overwrite_multicodepoint_grapheme_tail_clears_grapheme_data(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 10, 10)
	defer term.Deinit()

	// Enable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	// https://github.com/mitchellh/ghostty/issues/289
	// This is: 👨‍👩‍👧 (which may or may not render correctly)
	zigTry(t, term.Print(0x1F468))
	zigTry(t, term.Print(0x200D))
	zigTry(t, term.Print(0x1F469))
	zigTry(t, term.Print(0x200D))
	zigTry(t, term.Print(0x1F467))

	// We should have 2 cells taken up. It is one character but "wide".
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 2, term.Screens.Active.Cursor.X)

	// We should have one cell with graphemes
	page := &term.Screens.Active.Cursor.PagePin.Node.data
	expectEqual(t, 1, page.GraphemeCount())

	// Move back and overwrite wide
	term.SetCursorPos(1, 2)
	zigTry(t, term.Print('X'))

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, " X", str)
	}

	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 2, term.Screens.Active.Cursor.X)
	expectEqual(t, 0, page.GraphemeCount())
}

// Zig: test "Terminal: print breaks valid grapheme cluster with Prepend + ASCII for speed"
func TestTm1_Terminal_print_breaks_valid_grapheme_cluster_with_Prepend_ASCII_for_speed(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()
	term.Modes.Set(ModesModeGraphemeCluster, true)

	// Make sure we're not at cursor.x == 0 for the next char.
	zigTry(t, term.Print('_'))

	// U+0600 ARABIC NUMBER SIGN (Prepend)
	zigTry(t, term.Print(0x0600))
	zigTry(t, term.Print('1'))

	// We should have 3 cells taken up, each narrow. Note that this is
	// **incorrect** grapheme break behavior, since a Prepend code point should
	// not break with the one following it per UAX #29 GB9b. However, as an
	// optimization we assume a grapheme break when c <= 255, and note that
	// this deviation only affects these very uncommon scenarios (e.g. the
	// Arabic number sign should precede Arabic-script digits).
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 3, term.Screens.Active.Cursor.X)
	// This is what we'd expect if we did break correctly:
	//try testing.expectEqual(@as(usize, 2), t.screens.active.cursor.x);

	// Assert various properties about our screen to verify
	// we have all expected cells.
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(1, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0x0600, cell.content().codepoint())
		expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
		// This is what we'd expect if we did break correctly:
		//try testing.expect(cell.hasGrapheme());
		//try testing.expectEqualSlices(u21, &.{'1'}, list_cell.node.data.lookupGrapheme(cell).?);
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(2, 0)))
		cell := list_cell.Cell
		expectEqual(t, '1', cell.content().codepoint())
		// This is what we'd expect if we did break correctly:
		//try testing.expectEqual(@as(u21, 0), cell.content.codepoint);
		expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}
}

// Zig: test "Terminal: print writes to bottom if scrolled"
func TestTm1_Terminal_print_writes_to_bottom_if_scrolled(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 2)
	defer term.Deinit()

	// Basic grid writing
	for _, c := range []uint8("hello") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.SetCursorPos(0, 0)

	// Make newlines so we create scrollback
	// 3 pushes hello off the screen
	zigTry(t, term.Index())
	zigTry(t, term.Index())
	zigTry(t, term.Index())
	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "", str)
	}

	// Scroll to the top
	term.Screens.Active.Scroll(ScreenScroll{Tag: ScreenScrollTop})
	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "hello", str)
	}

	// Type
	zigTry(t, term.Print('A'))
	term.Screens.Active.Scroll(ScreenScroll{Tag: ScreenScrollActive})
	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\nA", str)
	}

	expect(t, term.isDirty(tm1Active(term.Screens.Active.Cursor.X, term.Screens.Active.Cursor.Y)), "t.isDirty(.{ .active = .{ .x = t.screens.active.cursor.x, .y = t.screens.active.cursor.y } })")
}

// Zig: test "Terminal: print charset"
func TestTm1_Terminal_print_charset(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	// G1 should have no effect
	term.ConfigureCharset(CharsetsSlotsG1, CharsetsCharsetDecSpecial)
	term.ConfigureCharset(CharsetsSlotsG2, CharsetsCharsetDecSpecial)
	term.ConfigureCharset(CharsetsSlotsG3, CharsetsCharsetDecSpecial)

	// No dirty to configure charset
	expect(t, !term.isDirty(tm1Screen(0, 0)), "!t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")

	// Basic grid writing
	zigTry(t, term.Print('`'))
	term.ConfigureCharset(CharsetsSlotsG0, CharsetsCharsetUtf8)
	zigTry(t, term.Print('`'))
	term.ConfigureCharset(CharsetsSlotsG0, CharsetsCharsetAscii)
	zigTry(t, term.Print('`'))
	term.ConfigureCharset(CharsetsSlotsG0, CharsetsCharsetDecSpecial)
	zigTry(t, term.Print('`'))
	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "```◆", str)
	}

	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
}

// Zig: test "Terminal: print charset outside of ASCII"
func TestTm1_Terminal_print_charset_outside_of_ASCII(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	// G1 should have no effect
	term.ConfigureCharset(CharsetsSlotsG1, CharsetsCharsetDecSpecial)
	term.ConfigureCharset(CharsetsSlotsG2, CharsetsCharsetDecSpecial)
	term.ConfigureCharset(CharsetsSlotsG3, CharsetsCharsetDecSpecial)

	// No dirty to configure charset
	expect(t, !term.isDirty(tm1Screen(0, 0)), "!t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")

	// Basic grid writing
	term.ConfigureCharset(CharsetsSlotsG0, CharsetsCharsetDecSpecial)
	zigTry(t, term.Print('`'))
	zigTry(t, term.Print(0x1F600))
	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "◆ ", str)
	}

	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
}

// Zig: test "Terminal: print invoke charset"
func TestTm1_Terminal_print_invoke_charset(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	term.ConfigureCharset(CharsetsSlotsG1, CharsetsCharsetDecSpecial)

	zigTry(t, term.Print('`'))

	// Invokecharset but should not mark dirty on its own
	term.clearDirty()
	term.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG1, false)
	expect(t, !term.isDirty(tm1Screen(0, 0)), "!t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
	zigTry(t, term.Print('`'))
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
	zigTry(t, term.Print('`'))
	term.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG0, false)
	zigTry(t, term.Print('`'))
	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "`◆◆`", str)
	}
}

// Zig: test "Terminal: print invoke charset single"
func TestTm1_Terminal_print_invoke_charset_single(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	term.ConfigureCharset(CharsetsSlotsG1, CharsetsCharsetDecSpecial)

	// Basic grid writing
	zigTry(t, term.Print('`'))
	term.InvokeCharset(CharsetsActiveSlotGL, CharsetsSlotsG1, true)
	zigTry(t, term.Print('`'))
	zigTry(t, term.Print('`'))
	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "`◆`", str)
	}
}

// Zig: test "Terminal: soft wrap"
func TestTm1_Terminal_soft_wrap(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 3, 80)
	defer term.Deinit()

	// Basic grid writing
	for _, c := range []uint8("hello") {
		zigTry(t, term.Print(uint32(c)))
	}
	expectEqual(t, 1, term.Screens.Active.Cursor.Y)
	expectEqual(t, 2, term.Screens.Active.Cursor.X)
	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "hel\nlo", str)
	}
}

// Zig: test "Terminal: soft wrap with semantic prompt"
func TestTm1_Terminal_soft_wrap_with_semantic_prompt(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 3, 80)
	defer term.Deinit()

	// Mark our prompt.
	zigTry(t, term.SemanticPrompt(OSCParsersSemanticPromptCommandInit(OSCParsersSemanticPromptCommandActionPromptStart)))
	// Should not make anything dirty on its own.
	expect(t, !term.isDirty(tm1Screen(0, 0)), "!t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")

	// Write and wrap
	for _, c := range []uint8("hello") {
		zigTry(t, term.Print(uint32(c)))
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 0)))
		expectEqual(t, PageRowSemanticPromptPrompt, list_cell.Row.semantic_prompt)
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 1)))
		expectEqual(t, PageRowSemanticPromptPromptContinuation, list_cell.Row.semantic_prompt)
	}
}

// Zig: test "Terminal: disabled wraparound with wide char and one space"
func TestTm1_Terminal_disabled_wraparound_with_wide_char_and_one_space(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	term.Modes.Set(ModesModeWraparound, false)

	// This puts our cursor at the end and there is NO SPACE for a
	// wide character.
	zigTry(t, term.PrintString([]uint8("AAAA")))
	term.clearDirty()
	zigTry(t, term.Print(0x1F6A8)) // Police car light
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 4, term.Screens.Active.Cursor.X)

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "AAAA", str)
	}

	// Make sure we printed nothing
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(4, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0, cell.content().codepoint())
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}

	// Should not be dirty since we didn't modify anything
	expect(t, !term.isDirty(tm1Screen(0, 0)), "!t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
}

// Zig: test "Terminal: disabled wraparound with wide char and no space"
func TestTm1_Terminal_disabled_wraparound_with_wide_char_and_no_space(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	term.Modes.Set(ModesModeWraparound, false)

	// This puts our cursor at the end and there is NO SPACE for a
	// wide character.
	zigTry(t, term.PrintString([]uint8("AAAAA")))
	term.clearDirty()
	zigTry(t, term.Print(0x1F6A8)) // Police car light
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 4, term.Screens.Active.Cursor.X)

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "AAAAA", str)
	}

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(4, 0)))
		cell := list_cell.Cell
		expectEqual(t, 'A', cell.content().codepoint())
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}

	// Should not be dirty since we didn't modify anything
	expect(t, !term.isDirty(tm1Screen(0, 0)), "!t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
}

// Zig: test "Terminal: disabled wraparound with wide grapheme and half space"
func TestTm1_Terminal_disabled_wraparound_with_wide_grapheme_and_half_space(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	term.Modes.Set(ModesModeGraphemeCluster, true)
	term.Modes.Set(ModesModeWraparound, false)

	// This puts our cursor at the end and there is NO SPACE for a
	// wide character.
	zigTry(t, term.PrintString([]uint8("AAAA")))
	zigTry(t, term.Print(0x2764)) // Heart
	term.clearDirty()
	zigTry(t, term.Print(0xFE0F)) // VS16 to make wide
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 4, term.Screens.Active.Cursor.X)

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "AAAA❤", str)
	}

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(4, 0)))
		cell := list_cell.Cell
		expectEqual(t, '❤', cell.content().codepoint())
		expectEqual(t, PageCellWideNarrow, cell.Wide())
	}

	// Should not be dirty since we didn't modify anything
	expect(t, !term.isDirty(tm1Screen(0, 0)), "!t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
}

// Zig: test "Terminal: print right margin wrap"
func TestTm1_Terminal_print_right_margin_wrap(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 10, 5)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("123456789")))
	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(3, 5)
	term.SetCursorPos(1, 5)
	zigTry(t, term.PrintString([]uint8("XY")))

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "1234X6789\n  Y", str)
	}

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Active(0, 0)))
		row := list_cell.Row
		expect(t, !row.wrap, "!row.wrap")
	}
}

// Zig: test "Terminal: print right margin wrap dirty tracking"
func TestTm1_Terminal_print_right_margin_wrap_dirty_tracking(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 10, 5)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("123456789")))
	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(3, 5)
	term.SetCursorPos(1, 5)

	// Writing our X on the first line should mark only that line dirty.
	term.clearDirty()
	zigTry(t, term.Print('X'))
	expect(t, term.isDirty(tm1Screen(4, 0)), "t.isDirty(.{ .screen = .{ .x = 4, .y = 0 } })")
	expect(t, !term.isDirty(tm1Screen(2, 1)), "!t.isDirty(.{ .screen = .{ .x = 2, .y = 1 } })")

	// Writing our Y should wrap. It marks both rows dirty because the
	// cursor moved.
	term.clearDirty()
	zigTry(t, term.Print('Y'))
	expect(t, term.isDirty(tm1Screen(4, 0)), "t.isDirty(.{ .screen = .{ .x = 4, .y = 0 } })")
	expect(t, term.isDirty(tm1Screen(2, 1)), "t.isDirty(.{ .screen = .{ .x = 2, .y = 1 } })")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "1234X6789\n  Y", str)
	}
}

// Zig: test "Terminal: print right margin outside"
func TestTm1_Terminal_print_right_margin_outside(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 10, 5)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("123456789")))
	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(3, 5)
	term.SetCursorPos(1, 6)
	term.clearDirty()
	zigTry(t, term.PrintString([]uint8("XY")))

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "12345XY89", str)
	}

	expect(t, term.isDirty(tm1Screen(5, 0)), "t.isDirty(.{ .screen = .{ .x = 5, .y = 0 } })")
}

// Zig: test "Terminal: print right margin outside wrap"
func TestTm1_Terminal_print_right_margin_outside_wrap(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 10, 5)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("123456789")))
	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(3, 5)
	term.SetCursorPos(1, 10)
	zigTry(t, term.PrintString([]uint8("XY")))

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "123456789X\n  Y", str)
	}
}

// Zig: test "Terminal: print wide char at right margin does not create spacer head"
func TestTm1_Terminal_print_wide_char_at_right_margin_does_not_create_spacer_head(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 10, 10)
	defer term.Deinit()

	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(3, 5)
	term.SetCursorPos(1, 5)
	zigTry(t, term.Print(0x1F600)) // Smiley face
	expectEqual(t, 1, term.Screens.Active.Cursor.Y)
	expectEqual(t, 4, term.Screens.Active.Cursor.X)

	// Both rows dirty because the cursor moved
	expect(t, term.isDirty(tm1Screen(4, 0)), "t.isDirty(.{ .screen = .{ .x = 4, .y = 0 } })")
	expect(t, term.isDirty(tm1Screen(4, 1)), "t.isDirty(.{ .screen = .{ .x = 4, .y = 1 } })")

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(4, 0)))
		cell := list_cell.Cell
		expectEqual(t, 0, cell.content().codepoint())
		expectEqual(t, PageCellWideNarrow, cell.Wide())

		row := list_cell.Row
		expect(t, !row.wrap, "!row.wrap")
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(2, 1)))
		cell := list_cell.Cell
		expectEqual(t, 0x1F600, cell.content().codepoint())
		expectEqual(t, PageCellWideWide, cell.Wide())
	}
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(3, 1)))
		cell := list_cell.Cell
		expectEqual(t, PageCellWideSpacerTail, cell.Wide())
	}
}

// Zig: test "Terminal: print with hyperlink"
func TestTm1_Terminal_print_with_hyperlink(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	// Setup our hyperlink and print
	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://example.com"), None[[]uint8]()))
	zigTry(t, term.PrintString([]uint8("123456")))

	// Verify all our cells have a hyperlink
	for x := 0; x < 6; x++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(x, 0)))
		row := list_cell.Row
		expect(t, row.hyperlink, "row.hyperlink")
		cell := list_cell.Cell
		expect(t, cell.Hyperlink(), "cell.hyperlink")
		id := zigUnwrap(t, list_cell.Node.data.LookupHyperlink(cell))
		expectEqual(t, 1, id)
	}

	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
}

// Zig: test "Terminal: print over cell with same hyperlink"
func TestTm1_Terminal_print_over_cell_with_same_hyperlink(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	// Setup our hyperlink and print
	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://example.com"), None[[]uint8]()))
	zigTry(t, term.PrintString([]uint8("123456")))
	term.SetCursorPos(1, 1)
	zigTry(t, term.PrintString([]uint8("123456")))

	// Verify all our cells have a hyperlink
	for x := 0; x < 6; x++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(x, 0)))
		row := list_cell.Row
		expect(t, row.hyperlink, "row.hyperlink")
		cell := list_cell.Cell
		expect(t, cell.Hyperlink(), "cell.hyperlink")
		id := zigUnwrap(t, list_cell.Node.data.LookupHyperlink(cell))
		expectEqual(t, 1, id)
	}

	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
}

// Zig: test "Terminal: print and end hyperlink"
func TestTm1_Terminal_print_and_end_hyperlink(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	// Setup our hyperlink and print
	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://example.com"), None[[]uint8]()))
	zigTry(t, term.PrintString([]uint8("123")))
	term.Screens.Active.EndHyperlink()
	zigTry(t, term.PrintString([]uint8("456")))

	// Verify all our cells have a hyperlink
	for x := 0; x < 3; x++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(x, 0)))
		row := list_cell.Row
		expect(t, row.hyperlink, "row.hyperlink")
		cell := list_cell.Cell
		expect(t, cell.Hyperlink(), "cell.hyperlink")
		id := zigUnwrap(t, list_cell.Node.data.LookupHyperlink(cell))
		expectEqual(t, 1, id)
	}
	for x := 3; x < 6; x++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(x, 0)))
		row := list_cell.Row
		expect(t, row.hyperlink, "row.hyperlink")
		cell := list_cell.Cell
		expect(t, !cell.Hyperlink(), "!cell.hyperlink")
	}

	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
}

// Zig: test "Terminal: print and change hyperlink"
func TestTm1_Terminal_print_and_change_hyperlink(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	// Setup our hyperlink and print
	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://one.example.com"), None[[]uint8]()))
	zigTry(t, term.PrintString([]uint8("123")))
	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://two.example.com"), None[[]uint8]()))
	zigTry(t, term.PrintString([]uint8("456")))

	// Verify all our cells have a hyperlink
	for x := 0; x < 3; x++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(x, 0)))
		cell := list_cell.Cell
		expect(t, cell.Hyperlink(), "cell.hyperlink")
		id := zigUnwrap(t, list_cell.Node.data.LookupHyperlink(cell))
		expectEqual(t, 1, id)
	}
	for x := 3; x < 6; x++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(x, 0)))
		cell := list_cell.Cell
		expect(t, cell.Hyperlink(), "cell.hyperlink")
		id := zigUnwrap(t, list_cell.Node.data.LookupHyperlink(cell))
		expectEqual(t, 2, id)
	}

	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
}

// Zig: test "Terminal: overwrite hyperlink"
func TestTm1_Terminal_overwrite_hyperlink(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	// Setup our hyperlink and print
	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://one.example.com"), None[[]uint8]()))
	zigTry(t, term.PrintString([]uint8("123")))
	term.SetCursorPos(1, 1)
	term.Screens.Active.EndHyperlink()
	zigTry(t, term.PrintString([]uint8("456")))

	// Verify all our cells have a hyperlink
	for x := 0; x < 3; x++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(x, 0)))
		page := &list_cell.Node.data
		row := list_cell.Row
		expect(t, !row.hyperlink, "!row.hyperlink")
		cell := list_cell.Cell
		expect(t, !cell.Hyperlink(), "!cell.hyperlink")
		expect(t, !page.LookupHyperlink(cell).Valid, "page.lookupHyperlink(cell) == null")
		expectEqual(t, 0, page.hyperlink_set.Count())
	}

	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
}

// Printing a wide char at the right edge with an active hyperlink causes
// printCell to write a spacer_head before printWrap sets the row wrap
// flag. The integrity check inside setHyperlink (or increaseCapacity)
// sees the unwrapped spacer head and panics. Found via fuzzing.
// Zig: test "Terminal: print wide char at right edge with hyperlink"
func TestTm1_Terminal_print_wide_char_at_right_edge_with_hyperlink(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 10, 5)
	defer term.Deinit()

	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://example.com"), None[[]uint8]()))

	// Move cursor to the last column (1-indexed)
	term.SetCursorPos(1, 10)

	// Print a wide character; this will call printCell(0, .spacer_head)
	// at the right edge before calling printWrap, triggering the
	// integrity violation.
	zigTry(t, term.Print(0x4E2D)) // U+4E2D '中'

	// Cursor wraps to row 2, after the wide char + spacer tail
	expectEqual(t, 1, term.Screens.Active.Cursor.Y)
	expectEqual(t, 2, term.Screens.Active.Cursor.X)

	// Row 0, col 9: spacer head with hyperlink
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(9, 0)))
		expectEqual(t, PageCellWideSpacerHead, list_cell.Cell.Wide())
		expect(t, list_cell.Cell.Hyperlink(), "list_cell.cell.hyperlink")
		expect(t, list_cell.Row.wrap, "list_cell.row.wrap")
	}
	// Row 1, col 0: the wide char with hyperlink
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(0, 1)))
		expectEqual(t, 0x4E2D, list_cell.Cell.content().codepoint())
		expectEqual(t, PageCellWideWide, list_cell.Cell.Wide())
		expect(t, list_cell.Cell.Hyperlink(), "list_cell.cell.hyperlink")
	}
	// Row 1, col 1: spacer tail with hyperlink
	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Screen(1, 1)))
		expectEqual(t, PageCellWideSpacerTail, list_cell.Cell.Wide())
		expect(t, list_cell.Cell.Hyperlink(), "list_cell.cell.hyperlink")
	}
}

// Zig: test "Terminal: linefeed and carriage return"
func TestTm1_Terminal_linefeed_and_carriage_return(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	// Print and CR.
	for _, c := range []uint8("hello") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.clearDirty()
	term.CarriageReturn()

	// CR should not mark row dirty because it doesn't change rendering.
	expect(t, !term.isDirty(tm1Screen(0, 0)), "!t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")

	zigTry(t, term.Linefeed())

	// LF marks row dirty due to cursor movement
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm1Screen(0, 1)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 1 } })")

	for _, c := range []uint8("world") {
		zigTry(t, term.Print(uint32(c)))
	}
	expectEqual(t, 1, term.Screens.Active.Cursor.Y)
	expectEqual(t, 5, term.Screens.Active.Cursor.X)
	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "hello\nworld", str)
	}
}

// Zig: test "Terminal: linefeed unsets pending wrap"
func TestTm1_Terminal_linefeed_unsets_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 80)
	defer term.Deinit()

	// Basic grid writing
	for _, c := range []uint8("hello") {
		zigTry(t, term.Print(uint32(c)))
	}
	expect(t, term.Screens.Active.Cursor.PendingWrap == true, "t.screens.active.cursor.pending_wrap == true")
	term.clearDirty()
	zigTry(t, term.Linefeed())
	expect(t, term.isDirty(tm1Screen(0, 0)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm1Screen(0, 1)), "t.isDirty(.{ .screen = .{ .x = 0, .y = 1 } })")
	expect(t, term.Screens.Active.Cursor.PendingWrap == false, "t.screens.active.cursor.pending_wrap == false")
}

// Zig: test "Terminal: linefeed mode automatic carriage return"
func TestTm1_Terminal_linefeed_mode_automatic_carriage_return(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 10, 10)
	defer term.Deinit()

	// Basic grid writing
	term.Modes.Set(ModesModeLinefeed, true)
	zigTry(t, term.PrintString([]uint8("123456")))
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('X'))
	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "123456\nX", str)
	}
}

// Zig: test "Terminal: carriage return unsets pending wrap"
func TestTm1_Terminal_carriage_return_unsets_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 80)
	defer term.Deinit()

	// Basic grid writing
	for _, c := range []uint8("hello") {
		zigTry(t, term.Print(uint32(c)))
	}
	expect(t, term.Screens.Active.Cursor.PendingWrap == true, "t.screens.active.cursor.pending_wrap == true")
	term.CarriageReturn()
	expect(t, term.Screens.Active.Cursor.PendingWrap == false, "t.screens.active.cursor.pending_wrap == false")
}

// Zig: test "Terminal: carriage return origin mode moves to left margin"
func TestTm1_Terminal_carriage_return_origin_mode_moves_to_left_margin(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 80)
	defer term.Deinit()

	term.Modes.Set(ModesModeOrigin, true)
	term.Screens.Active.Cursor.X = 0
	term.ScrollingRegion.Left = 2
	term.CarriageReturn()
	expectEqual(t, 2, term.Screens.Active.Cursor.X)
}

// Zig: test "Terminal: carriage return left of left margin moves to zero"
func TestTm1_Terminal_carriage_return_left_of_left_margin_moves_to_zero(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 80)
	defer term.Deinit()

	term.Screens.Active.Cursor.X = 1
	term.ScrollingRegion.Left = 2
	term.CarriageReturn()
	expectEqual(t, 0, term.Screens.Active.Cursor.X)
}

// Zig: test "Terminal: carriage return right of left margin moves to left margin"
func TestTm1_Terminal_carriage_return_right_of_left_margin_moves_to_left_margin(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 80)
	defer term.Deinit()

	term.Screens.Active.Cursor.X = 3
	term.ScrollingRegion.Left = 2
	term.CarriageReturn()
	expectEqual(t, 2, term.Screens.Active.Cursor.X)
}

// Zig: test "Terminal: backspace"
func TestTm1_Terminal_backspace(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	// BS
	for _, c := range []uint8("hello") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.Backspace()
	zigTry(t, term.Print('y'))
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)
	expectEqual(t, 5, term.Screens.Active.Cursor.X)
	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "helly", str)
	}
}

// Zig: test "Terminal: horizontal tabs"
func TestTm1_Terminal_horizontal_tabs(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 20, 5)
	defer term.Deinit()

	// HT
	zigTry(t, term.Print('1'))
	term.HorizontalTab()
	expectEqual(t, 8, term.Screens.Active.Cursor.X)

	// HT
	term.HorizontalTab()
	expectEqual(t, 16, term.Screens.Active.Cursor.X)

	// HT at the end
	term.HorizontalTab()
	expectEqual(t, 19, term.Screens.Active.Cursor.X)
	term.HorizontalTab()
	expectEqual(t, 19, term.Screens.Active.Cursor.X)
}

// Zig: test "Terminal: horizontal tabs starting on tabstop"
func TestTm1_Terminal_horizontal_tabs_starting_on_tabstop(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 20, 5)
	defer term.Deinit()

	term.SetCursorPos(uint(term.Screens.Active.Cursor.Y), 9)
	zigTry(t, term.Print('X'))
	term.SetCursorPos(uint(term.Screens.Active.Cursor.Y), 9)
	term.HorizontalTab()
	zigTry(t, term.Print('A'))

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "        X       A", str)
	}
}

// Zig: test "Terminal: horizontal tabs with right margin"
func TestTm1_Terminal_horizontal_tabs_with_right_margin(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 20, 5)
	defer term.Deinit()

	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 5
	term.SetCursorPos(uint(term.Screens.Active.Cursor.Y), 1)
	zigTry(t, term.Print('X'))
	term.HorizontalTab()
	zigTry(t, term.Print('A'))

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "X    A", str)
	}
}

// Zig: test "Terminal: horizontal tabs back"
func TestTm1_Terminal_horizontal_tabs_back(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 20, 5)
	defer term.Deinit()

	// Edge of screen
	term.SetCursorPos(uint(term.Screens.Active.Cursor.Y), 20)

	// HT
	term.HorizontalTabBack()
	expectEqual(t, 16, term.Screens.Active.Cursor.X)

	// HT
	term.HorizontalTabBack()
	expectEqual(t, 8, term.Screens.Active.Cursor.X)

	// HT
	term.HorizontalTabBack()
	expectEqual(t, 0, term.Screens.Active.Cursor.X)
	term.HorizontalTabBack()
	expectEqual(t, 0, term.Screens.Active.Cursor.X)
}

// Zig: test "Terminal: horizontal tabs back starting on tabstop"
func TestTm1_Terminal_horizontal_tabs_back_starting_on_tabstop(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 20, 5)
	defer term.Deinit()

	term.SetCursorPos(uint(term.Screens.Active.Cursor.Y), 9)
	zigTry(t, term.Print('X'))
	term.SetCursorPos(uint(term.Screens.Active.Cursor.Y), 9)
	term.HorizontalTabBack()
	zigTry(t, term.Print('A'))

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "A       X", str)
	}
}

// Zig: test "Terminal: horizontal tabs with left margin in origin mode"
func TestTm1_Terminal_horizontal_tabs_with_left_margin_in_origin_mode(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 20, 5)
	defer term.Deinit()

	term.Modes.Set(ModesModeOrigin, true)
	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 5
	term.SetCursorPos(1, 2)
	zigTry(t, term.Print('X'))
	term.HorizontalTabBack()
	zigTry(t, term.Print('A'))

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "  AX", str)
	}
}

// Zig: test "Terminal: horizontal tab back with cursor before left margin"
func TestTm1_Terminal_horizontal_tab_back_with_cursor_before_left_margin(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 20, 5)
	defer term.Deinit()

	term.Modes.Set(ModesModeOrigin, true)
	term.SaveCursor()
	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(5, 0)
	term.RestoreCursor()
	term.HorizontalTabBack()
	zigTry(t, term.Print('X'))

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "X", str)
	}
}

// Zig: test "Terminal: cursorPos resets wrap"
func TestTm1_Terminal_cursorPos_resets_wrap(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	for _, c := range []uint8("ABCDE") {
		zigTry(t, term.Print(uint32(c)))
	}
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	term.SetCursorPos(1, 1)
	expect(t, !term.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
	zigTry(t, term.Print('X'))

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "XBCDE", str)
	}
}

// Zig: test "Terminal: cursorPos off the screen"
func TestTm1_Terminal_cursorPos_off_the_screen(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	term.SetCursorPos(500, 500)
	zigTry(t, term.Print('X'))

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\n\n\n\n    X", str)
	}
}

// Zig: test "Terminal: cursorPos relative to origin"
func TestTm1_Terminal_cursorPos_relative_to_origin(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	term.ScrollingRegion.Top = 2
	term.ScrollingRegion.Bottom = 3
	term.Modes.Set(ModesModeOrigin, true)
	term.SetCursorPos(1, 1)
	zigTry(t, term.Print('X'))

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\n\nX", str)
	}
}

// Zig: test "Terminal: cursorPos relative to origin with left/right"
func TestTm1_Terminal_cursorPos_relative_to_origin_with_left_right(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	term.ScrollingRegion.Top = 2
	term.ScrollingRegion.Bottom = 3
	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 4
	term.Modes.Set(ModesModeOrigin, true)
	term.SetCursorPos(1, 1)
	zigTry(t, term.Print('X'))

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\n\n  X", str)
	}
}

// Zig: test "Terminal: cursorPos limits with full scroll region"
func TestTm1_Terminal_cursorPos_limits_with_full_scroll_region(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	term.ScrollingRegion.Top = 2
	term.ScrollingRegion.Bottom = 3
	term.ScrollingRegion.Left = 2
	term.ScrollingRegion.Right = 4
	term.Modes.Set(ModesModeOrigin, true)
	term.SetCursorPos(500, 500)
	zigTry(t, term.Print('X'))

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\n\n\n    X", str)
	}
}

// Probably outdated, but dates back to the original terminal implementation.
// Zig: test "Terminal: setCursorPos (original test)"
func TestTm1_Terminal_setCursorPos_original_test(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 80, 80)
	defer term.Deinit()

	expectEqual(t, 0, term.Screens.Active.Cursor.X)
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)

	// Setting it to 0 should keep it zero (1 based)
	term.SetCursorPos(0, 0)
	expectEqual(t, 0, term.Screens.Active.Cursor.X)
	expectEqual(t, 0, term.Screens.Active.Cursor.Y)

	// Should clamp to size
	term.SetCursorPos(81, 81)
	expectEqual(t, 79, term.Screens.Active.Cursor.X)
	expectEqual(t, 79, term.Screens.Active.Cursor.Y)

	// Should reset pending wrap
	term.SetCursorPos(0, 80)
	zigTry(t, term.Print('c'))
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	term.SetCursorPos(0, 80)
	expect(t, !term.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")

	// Origin mode
	term.Modes.Set(ModesModeOrigin, true)

	// No change without a scroll region
	term.SetCursorPos(81, 81)
	expectEqual(t, 79, term.Screens.Active.Cursor.X)
	expectEqual(t, 79, term.Screens.Active.Cursor.Y)

	// Set the scroll region
	term.SetTopAndBottomMargin(10, uint(term.Rows))
	term.SetCursorPos(0, 0)
	expectEqual(t, 0, term.Screens.Active.Cursor.X)
	expectEqual(t, 9, term.Screens.Active.Cursor.Y)

	term.SetCursorPos(1, 1)
	expectEqual(t, 0, term.Screens.Active.Cursor.X)
	expectEqual(t, 9, term.Screens.Active.Cursor.Y)

	term.SetCursorPos(100, 0)
	expectEqual(t, 0, term.Screens.Active.Cursor.X)
	expectEqual(t, 79, term.Screens.Active.Cursor.Y)

	term.SetTopAndBottomMargin(10, 11)
	term.SetCursorPos(2, 0)
	expectEqual(t, 0, term.Screens.Active.Cursor.X)
	expectEqual(t, 10, term.Screens.Active.Cursor.Y)
}

// Zig: test "Terminal: setTopAndBottomMargin simple"
func TestTm1_Terminal_setTopAndBottomMargin_simple(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI")))
	term.SetTopAndBottomMargin(0, 0)

	term.clearDirty()
	term.ScrollDown(1)

	// Mark the rows we moved as dirty.
	expect(t, term.isDirty(tm1Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm1Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, term.isDirty(tm1Active(0, 2)), "t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")
	expect(t, term.isDirty(tm1Active(0, 3)), "t.isDirty(.{ .active = .{ .x = 0, .y = 3 } })")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\nABC\nDEF\nGHI", str)
	}
}

// Zig: test "Terminal: setTopAndBottomMargin top only"
func TestTm1_Terminal_setTopAndBottomMargin_top_only(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI")))
	term.SetTopAndBottomMargin(2, 0)

	term.clearDirty()
	term.ScrollDown(1)

	// This is dirty because the cursor moves from this row
	expect(t, term.isDirty(tm1Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm1Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, term.isDirty(tm1Active(0, 2)), "t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")
	expect(t, term.isDirty(tm1Active(0, 3)), "t.isDirty(.{ .active = .{ .x = 0, .y = 3 } })")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABC\n\nDEF\nGHI", str)
	}
}

// Zig: test "Terminal: setTopAndBottomMargin top and bottom"
func TestTm1_Terminal_setTopAndBottomMargin_top_and_bottom(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI")))
	term.SetTopAndBottomMargin(1, 2)

	term.clearDirty()
	term.ScrollDown(1)

	expect(t, term.isDirty(tm1Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm1Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, !term.isDirty(tm1Active(0, 2)), "!t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\nABC\nGHI", str)
	}
}

// Zig: test "Terminal: setTopAndBottomMargin top equal to bottom"
func TestTm1_Terminal_setTopAndBottomMargin_top_equal_to_bottom(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI")))
	term.SetTopAndBottomMargin(2, 2)

	term.clearDirty()
	term.ScrollDown(1)

	expect(t, term.isDirty(tm1Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm1Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, term.isDirty(tm1Active(0, 2)), "t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")
	expect(t, term.isDirty(tm1Active(0, 3)), "t.isDirty(.{ .active = .{ .x = 0, .y = 3 } })")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\nABC\nDEF\nGHI", str)
	}
}

// Zig: test "Terminal: setLeftAndRightMargin simple"
func TestTm1_Terminal_setLeftAndRightMargin_simple(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI")))
	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(0, 0)

	term.clearDirty()
	term.EraseChars(1)

	expect(t, term.isDirty(tm1Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, !term.isDirty(tm1Active(0, 1)), "!t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, " BC\nDEF\nGHI", str)
	}
}

// Zig: test "Terminal: setLeftAndRightMargin left only"
func TestTm1_Terminal_setLeftAndRightMargin_left_only(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI")))
	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(2, 0)
	expectEqual(t, 1, term.ScrollingRegion.Left)
	expectEqual(t, term.Cols-1, term.ScrollingRegion.Right)
	term.SetCursorPos(1, 2)

	term.clearDirty()
	term.InsertLines(1)

	expect(t, term.isDirty(tm1Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm1Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, term.isDirty(tm1Active(0, 2)), "t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")
	expect(t, term.isDirty(tm1Active(0, 3)), "t.isDirty(.{ .active = .{ .x = 0, .y = 3 } })")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "A\nDBC\nGEF\n HI", str)
	}
}

// Zig: test "Terminal: setLeftAndRightMargin left and right"
func TestTm1_Terminal_setLeftAndRightMargin_left_and_right(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI")))
	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(1, 2)
	term.SetCursorPos(1, 2)

	term.clearDirty()
	term.InsertLines(1)

	expect(t, term.isDirty(tm1Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm1Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, term.isDirty(tm1Active(0, 2)), "t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")
	expect(t, term.isDirty(tm1Active(0, 3)), "t.isDirty(.{ .active = .{ .x = 0, .y = 3 } })")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "  C\nABF\nDEI\nGH", str)
	}
}

// Zig: test "Terminal: setLeftAndRightMargin left equal right"
func TestTm1_Terminal_setLeftAndRightMargin_left_equal_right(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI")))
	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(2, 2)
	term.SetCursorPos(1, 2)

	term.clearDirty()
	term.InsertLines(1)

	expect(t, term.isDirty(tm1Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm1Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, term.isDirty(tm1Active(0, 2)), "t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")
	expect(t, term.isDirty(tm1Active(0, 3)), "t.isDirty(.{ .active = .{ .x = 0, .y = 3 } })")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\nABC\nDEF\nGHI", str)
	}
}

// Zig: test "Terminal: setLeftAndRightMargin mode 69 unset"
func TestTm1_Terminal_setLeftAndRightMargin_mode_69_unset(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI")))
	term.Modes.Set(ModesModeEnableLeftAndRightMargin, false)
	term.SetLeftAndRightMargin(1, 2)
	term.SetCursorPos(1, 2)

	term.clearDirty()
	term.InsertLines(1)

	expect(t, term.isDirty(tm1Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm1Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, term.isDirty(tm1Active(0, 2)), "t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")
	expect(t, term.isDirty(tm1Active(0, 3)), "t.isDirty(.{ .active = .{ .x = 0, .y = 3 } })")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\nABC\nDEF\nGHI", str)
	}
}

// Zig: test "Terminal: insertLines simple"
func TestTm1_Terminal_insertLines_simple(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
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
	term.InsertLines(1)

	expect(t, !term.isDirty(tm1Active(0, 0)), "!t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm1Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, term.isDirty(tm1Active(0, 2)), "t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")
	expect(t, term.isDirty(tm1Active(0, 3)), "t.isDirty(.{ .active = .{ .x = 0, .y = 3 } })")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABC\n\nDEF\nGHI", str)
	}
}

// Zig: test "Terminal: insertLines colors with bg color"
func TestTm1_Terminal_insertLines_colors_with_bg_color(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
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
	term.InsertLines(1)

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABC\n\nDEF\nGHI", str)
	}

	for x := 0; x < int(term.Cols); x++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Active(x, 1)))
		expect(t, list_cell.Cell.ContentTag() == PageCellContentTagBgColorRgb, "list_cell.cell.content_tag == .bg_color_rgb")
		expectEqual(t, PageCellRGB{R: 0xFF, G: 0, B: 0}, list_cell.Cell.content().colorRGB())
	}
}

// Zig: test "Terminal: insertLines handles style refs"
func TestTm1_Terminal_insertLines_handles_style_refs(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 3)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())

	// For the line being deleted, create a refcounted style
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	zigTry(t, term.PrintString([]uint8("GHI")))
	zigTry(t, term.SetAttribute(SGRAttribute{Tag: SGRAttributeTagUnset}))

	// verify we have styles in our style map
	page := &term.Screens.Active.Cursor.PagePin.Node.data
	expectEqual(t, 1, page.styles.Count())

	term.SetCursorPos(2, 2)
	term.InsertLines(1)

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABC\n\nDEF", str)
	}

	// verify we have no styles in our style map
	expectEqual(t, 0, page.styles.Count())
}

// Zig: test "Terminal: insertLines outside of scroll region"
func TestTm1_Terminal_insertLines_outside_of_scroll_region(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI")))
	term.SetTopAndBottomMargin(3, 4)
	term.SetCursorPos(2, 2)

	term.clearDirty()
	term.InsertLines(1)

	expect(t, !term.isDirty(tm1Active(0, 0)), "!t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, !term.isDirty(tm1Active(0, 1)), "!t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, !term.isDirty(tm1Active(0, 2)), "!t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABC\nDEF\nGHI", str)
	}
}

// Zig: test "Terminal: insertLines top/bottom scroll region"
func TestTm1_Terminal_insertLines_top_bottom_scroll_region(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("123")))
	term.SetTopAndBottomMargin(1, 3)
	term.SetCursorPos(2, 2)

	term.clearDirty()
	term.InsertLines(1)

	expect(t, !term.isDirty(tm1Active(0, 0)), "!t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm1Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, term.isDirty(tm1Active(0, 2)), "t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")
	expect(t, !term.isDirty(tm1Active(0, 3)), "!t.isDirty(.{ .active = .{ .x = 0, .y = 3 } })")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABC\n\nDEF\n123", str)
	}
}

// Zig: test "Terminal: insertLines across page boundary marks all shifted rows dirty"
func TestTm1_Terminal_insertLines_across_page_boundary_marks_all_shifted_rows_dirty(t *testing.T) {
	testStdCapacity(t)
	term := tm1InitScrollback(t, 10, 5, 1024)
	defer term.Deinit()

	first_page := tm1FirstPage(t, term)
	first_page_nrows := first_page.data.capacity.rows

	// Fill up the first page minus 3 rows
	for i, end := 0, int(first_page_nrows-3); i < end; i++ {
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
	term.InsertLines(1)

	expect(t, term.isDirty(tm1Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm1Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, term.isDirty(tm1Active(0, 2)), "t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")
	expect(t, term.isDirty(tm1Active(0, 3)), "t.isDirty(.{ .active = .{ .x = 0, .y = 3 } })")
	expect(t, term.isDirty(tm1Active(0, 4)), "t.isDirty(.{ .active = .{ .x = 0, .y = 4 } })")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\n1AAAA\n2BBBB\n3CCCC\n4DDDD", str)
	}
}

// Zig: test "Terminal: insertLines (legacy test)"
func TestTm1_Terminal_insertLines_legacy_test(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 2, 5)
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
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('E'))

	// Move to row 2
	term.SetCursorPos(2, 1)

	// Insert two lines
	term.InsertLines(2)

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "A\n\n\nB\nC", str)
	}
}

// Zig: test "Terminal: insertLines zero"
func TestTm1_Terminal_insertLines_zero(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 2, 5)
	defer term.Deinit()

	// This should do nothing
	term.SetCursorPos(1, 1)
	term.InsertLines(0)
}

// Zig: test "Terminal: insertLines with scroll region"
func TestTm1_Terminal_insertLines_with_scroll_region(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 2, 6)
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
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('E'))

	term.SetTopAndBottomMargin(1, 2)
	term.SetCursorPos(1, 1)

	term.clearDirty()
	term.InsertLines(1)

	expect(t, term.isDirty(tm1Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm1Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, !term.isDirty(tm1Active(0, 2)), "!t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")

	zigTry(t, term.Print('X'))

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "X\nA\nC\nD\nE", str)
	}
}

// Zig: test "Terminal: insertLines more than remaining"
func TestTm1_Terminal_insertLines_more_than_remaining(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 2, 5)
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
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Print('E'))

	// Move to row 2
	term.SetCursorPos(2, 1)

	// Insert a bunch of  lines
	term.clearDirty()
	term.InsertLines(20)

	expect(t, !term.isDirty(tm1Active(0, 0)), "!t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm1Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, term.isDirty(tm1Active(0, 2)), "t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "A", str)
	}
}

// Zig: test "Terminal: insertLines resets pending wrap"
func TestTm1_Terminal_insertLines_resets_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	for _, c := range []uint8("ABCDE") {
		zigTry(t, term.Print(uint32(c)))
	}
	expect(t, term.Screens.Active.Cursor.PendingWrap, "t.screens.active.cursor.pending_wrap")
	term.InsertLines(1)
	expect(t, !term.Screens.Active.Cursor.PendingWrap, "!t.screens.active.cursor.pending_wrap")
	zigTry(t, term.Print('B'))

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "B\nABCDE", str)
	}
}

// Zig: test "Terminal: insertLines resets wrap"
func TestTm1_Terminal_insertLines_resets_wrap(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 3, 3)
	defer term.Deinit()

	zigTry(t, term.Print('1'))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	for _, c := range []uint8("ABCDEF") {
		zigTry(t, term.Print(uint32(c)))
	}
	term.SetCursorPos(1, 1)
	term.InsertLines(1)
	zigTry(t, term.Print('X'))

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "X\n1\nABC", str)
	}

	{
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Active(0, 2)))
		row := list_cell.Row
		expect(t, !row.wrap, "!row.wrap")
	}
}

// Zig: test "Terminal: insertLines multi-codepoint graphemes"
func TestTm1_Terminal_insertLines_multi_codepoint_graphemes(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	// Disable grapheme clustering
	term.Modes.Set(ModesModeGraphemeCluster, true)

	zigTry(t, term.PrintString([]uint8("ABC")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())

	// This is: 👨‍👩‍👧 (which may or may not render correctly)
	zigTry(t, term.Print(0x1F468))
	zigTry(t, term.Print(0x200D))
	zigTry(t, term.Print(0x1F469))
	zigTry(t, term.Print(0x200D))
	zigTry(t, term.Print(0x1F467))

	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI")))
	term.SetCursorPos(2, 2)
	term.InsertLines(1)

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABC\n\n👨‍👩‍👧\nGHI", str)
	}
}

// Zig: test "Terminal: insertLines left/right scroll region"
func TestTm1_Terminal_insertLines_left_right_scroll_region(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 10, 10)
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
	term.InsertLines(1)

	expect(t, !term.isDirty(tm1Active(0, 0)), "!t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm1Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, term.isDirty(tm1Active(0, 2)), "t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")
	expect(t, term.isDirty(tm1Active(0, 3)), "t.isDirty(.{ .active = .{ .x = 0, .y = 3 } })")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABC123\nD   56\nGEF489\n HI7", str)
	}
}

// Zig: test "Terminal: scrollUp simple"
func TestTm1_Terminal_scrollUp_simple(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI")))
	term.SetCursorPos(2, 2)

	cursor := term.Screens.Active.Cursor
	viewport_before := term.Screens.Active.Pages.GetTopLeft(PointTagViewport)
	zigTry(t, term.ScrollUp(1))
	expectEqual(t, cursor.X, term.Screens.Active.Cursor.X)
	expectEqual(t, cursor.Y, term.Screens.Active.Cursor.Y)

	// Viewport should have moved. Our entire page should've scrolled!
	// The viewport moving will cause our render state to make the full
	// frame as dirty.
	viewport_after := term.Screens.Active.Pages.GetTopLeft(PointTagViewport)
	expect(t, !viewport_before.Eql(viewport_after), "!viewport_before.eql(viewport_after)")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "DEF\nGHI", str)
	}
}

// Zig: test "Terminal: scrollUp moves hyperlink"
func TestTm1_Terminal_scrollUp_moves_hyperlink(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://example.com"), None[[]uint8]()))
	zigTry(t, term.PrintString([]uint8("DEF")))
	term.Screens.Active.EndHyperlink()
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI")))
	term.SetCursorPos(2, 2)
	zigTry(t, term.ScrollUp(1))

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "DEF\nGHI", str)
	}

	for x := 0; x < 3; x++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Viewport(x, 0)))
		row := list_cell.Row
		expect(t, row.hyperlink, "row.hyperlink")
		cell := list_cell.Cell
		expect(t, cell.Hyperlink(), "cell.hyperlink")
		id := zigUnwrap(t, list_cell.Node.data.LookupHyperlink(cell))
		expectEqual(t, 1, id)
		page := &list_cell.Node.data
		expectEqual(t, 1, page.hyperlink_set.Count())
	}
	for x := 0; x < 3; x++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Viewport(x, 1)))
		row := list_cell.Row
		expect(t, !row.hyperlink, "!row.hyperlink")
		cell := list_cell.Cell
		expect(t, !cell.Hyperlink(), "!cell.hyperlink")
		id := list_cell.Node.data.LookupHyperlink(cell)
		expect(t, !id.Valid, "id == null")
	}
}

// Zig: test "Terminal: scrollUp clears hyperlink"
func TestTm1_Terminal_scrollUp_clears_hyperlink(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://example.com"), None[[]uint8]()))
	zigTry(t, term.PrintString([]uint8("ABC")))
	term.Screens.Active.EndHyperlink()
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI")))
	term.SetCursorPos(2, 2)
	zigTry(t, term.ScrollUp(1))

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "DEF\nGHI", str)
	}

	for x := 0; x < 3; x++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Viewport(x, 0)))
		row := list_cell.Row
		expect(t, !row.hyperlink, "!row.hyperlink")
		cell := list_cell.Cell
		expect(t, !cell.Hyperlink(), "!cell.hyperlink")
		id := list_cell.Node.data.LookupHyperlink(cell)
		expect(t, !id.Valid, "id == null")
	}
}

// Zig: test "Terminal: scrollUp top/bottom scroll region"
func TestTm1_Terminal_scrollUp_top_bottom_scroll_region(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI")))
	term.SetTopAndBottomMargin(2, 3)
	term.SetCursorPos(1, 1)

	term.clearDirty()
	zigTry(t, term.ScrollUp(1))

	// This is dirty because the cursor moves from this row
	expect(t, term.isDirty(tm1Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm1Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, term.isDirty(tm1Active(0, 2)), "t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABC\nGHI", str)
	}
}

// Zig: test "Terminal: scrollUp left/right scroll region"
func TestTm1_Terminal_scrollUp_left_right_scroll_region(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 10, 10)
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

	cursor := term.Screens.Active.Cursor
	term.clearDirty()
	zigTry(t, term.ScrollUp(1))
	expectEqual(t, cursor.X, term.Screens.Active.Cursor.X)
	expectEqual(t, cursor.Y, term.Screens.Active.Cursor.Y)

	expect(t, term.isDirty(tm1Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm1Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, term.isDirty(tm1Active(0, 2)), "t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "AEF423\nDHI756\nG   89", str)
	}
}

// Zig: test "Terminal: scrollUp left/right scroll region hyperlink"
func TestTm1_Terminal_scrollUp_left_right_scroll_region_hyperlink(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 10, 10)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC123")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://example.com"), None[[]uint8]()))
	zigTry(t, term.PrintString([]uint8("DEF456")))
	term.Screens.Active.EndHyperlink()
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI789")))
	term.ScrollingRegion.Left = 1
	term.ScrollingRegion.Right = 3
	term.SetCursorPos(2, 2)
	zigTry(t, term.ScrollUp(1))

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "AEF423\nDHI756\nG   89", str)
	}

	// First row gets some hyperlinks
	{
		for x := 0; x < 1; x++ {
			list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Viewport(x, 0)))
			cell := list_cell.Cell
			expect(t, !cell.Hyperlink(), "!cell.hyperlink")
			id := list_cell.Node.data.LookupHyperlink(cell)
			expect(t, !id.Valid, "id == null")
		}
		for x := 1; x < 4; x++ {
			list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Viewport(x, 0)))
			row := list_cell.Row
			expect(t, row.hyperlink, "row.hyperlink")
			cell := list_cell.Cell
			expect(t, cell.Hyperlink(), "cell.hyperlink")
			id := zigUnwrap(t, list_cell.Node.data.LookupHyperlink(cell))
			expectEqual(t, 1, id)
			page := &list_cell.Node.data
			expectEqual(t, 1, page.hyperlink_set.Count())
		}
		for x := 4; x < 6; x++ {
			list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Viewport(x, 0)))
			cell := list_cell.Cell
			expect(t, !cell.Hyperlink(), "!cell.hyperlink")
			id := list_cell.Node.data.LookupHyperlink(cell)
			expect(t, !id.Valid, "id == null")
		}
	}

	// Second row preserves hyperlink where we didn't scroll
	{
		for x := 0; x < 1; x++ {
			list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Viewport(x, 1)))
			row := list_cell.Row
			expect(t, row.hyperlink, "row.hyperlink")
			cell := list_cell.Cell
			expect(t, cell.Hyperlink(), "cell.hyperlink")
			id := zigUnwrap(t, list_cell.Node.data.LookupHyperlink(cell))
			expectEqual(t, 1, id)
			page := &list_cell.Node.data
			expectEqual(t, 1, page.hyperlink_set.Count())
		}
		for x := 1; x < 4; x++ {
			list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Viewport(x, 1)))
			cell := list_cell.Cell
			expect(t, !cell.Hyperlink(), "!cell.hyperlink")
			id := list_cell.Node.data.LookupHyperlink(cell)
			expect(t, !id.Valid, "id == null")
		}
		for x := 4; x < 6; x++ {
			list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Viewport(x, 1)))
			row := list_cell.Row
			expect(t, row.hyperlink, "row.hyperlink")
			cell := list_cell.Cell
			expect(t, cell.Hyperlink(), "cell.hyperlink")
			id := zigUnwrap(t, list_cell.Node.data.LookupHyperlink(cell))
			expectEqual(t, 1, id)
			page := &list_cell.Node.data
			expectEqual(t, 1, page.hyperlink_set.Count())
		}
	}
}

// Zig: test "Terminal: scrollUp preserves pending wrap"
func TestTm1_Terminal_scrollUp_preserves_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	term.SetCursorPos(1, 5)
	zigTry(t, term.Print('A'))
	term.SetCursorPos(2, 5)
	zigTry(t, term.Print('B'))
	term.SetCursorPos(3, 5)
	zigTry(t, term.Print('C'))
	zigTry(t, term.ScrollUp(1))
	zigTry(t, term.Print('X'))

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "    B\n    C\n\nX", str)
	}
}

// Zig: test "Terminal: scrollUp full top/bottom region"
func TestTm1_Terminal_scrollUp_full_top_bottom_region(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("top")))
	term.SetCursorPos(5, 1)
	zigTry(t, term.PrintString([]uint8("ABCDE")))
	term.SetTopAndBottomMargin(2, 5)

	term.clearDirty()
	zigTry(t, term.ScrollUp(4))

	// This is dirty because the cursor moves from this row
	expect(t, term.isDirty(tm1Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	expect(t, term.isDirty(tm1Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "top", str)
	}
}

// Zig: test "Terminal: scrollUp full top/bottomleft/right scroll region"
func TestTm1_Terminal_scrollUp_full_top_bottomleft_right_scroll_region(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("top")))
	term.SetCursorPos(5, 1)
	zigTry(t, term.PrintString([]uint8("ABCDE")))
	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetTopAndBottomMargin(2, 5)
	term.SetLeftAndRightMargin(2, 4)

	term.clearDirty()
	zigTry(t, term.ScrollUp(4))

	// This is dirty because the cursor moves from this row
	expect(t, term.isDirty(tm1Active(0, 0)), "t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")
	for y := 1; y < 5; y++ {
		expect(t, term.isDirty(tm1Active(0, y)), "t.isDirty(.{ .active = .{ .x = 0, .y = @intCast(y) } })")
	}

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "top\n\n\n\nA   E", str)
	}
}

// Zig: test "Terminal: scrollUp creates scrollback in primary screen"
func TestTm1_Terminal_scrollUp_creates_scrollback_in_primary_screen(t *testing.T) {
	testStdCapacity(t)
	// When in primary screen with full-width scroll region at top,
	// scrollUp (CSI S) should push lines into scrollback like xterm.
	term := tm1InitScrollback(t, 5, 5, 10)
	defer term.Deinit()

	// Fill the screen with content
	zigTry(t, term.PrintString([]uint8("AAAAA")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("BBBBB")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("CCCCC")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DDDDD")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("EEEEE")))

	term.clearDirty()

	// Scroll up by 1, which should push "AAAAA" into scrollback
	zigTry(t, term.ScrollUp(1))

	// The cursor row (new empty row) should be dirty
	expect(t, term.Screens.Active.Cursor.PageRow.dirty, "t.screens.active.cursor.page_row.dirty")

	// The active screen should now show BBBBB through EEEEE plus one blank line
	{
		str := tm1PlainString(t, term)
		// Zig-only: defer alloc.free(str);
		expectEqualStrings(t, "BBBBB\nCCCCC\nDDDDD\nEEEEE", str)
	}

	// Now scroll to the top to see scrollback - AAAAA should be there
	term.Screens.Active.Scroll(ScreenScroll{Tag: ScreenScrollTop})
	{
		str := tm1PlainString(t, term)
		// Zig-only: defer alloc.free(str);
		// Should see AAAAA in scrollback
		expectEqualStrings(t, "AAAAA\nBBBBB\nCCCCC\nDDDDD\nEEEEE", str)
	}
}

// Zig: test "Terminal: scrollUp with max_scrollback zero"
func TestTm1_Terminal_scrollUp_with_max_scrollback_zero(t *testing.T) {
	testStdCapacity(t)
	// When max_scrollback is 0, scrollUp should still work but not retain history
	term := tm1InitScrollback(t, 5, 5, 0)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("AAAAA")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("BBBBB")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("CCCCC")))

	zigTry(t, term.ScrollUp(1))

	// Active screen should show scrolled content
	{
		str := tm1PlainString(t, term)
		// Zig-only: defer alloc.free(str);
		expectEqualStrings(t, "BBBBB\nCCCCC", str)
	}

	// Scroll to top - should be same as active since no scrollback
	term.Screens.Active.Scroll(ScreenScroll{Tag: ScreenScrollTop})
	{
		str := tm1PlainString(t, term)
		// Zig-only: defer alloc.free(str);
		expectEqualStrings(t, "BBBBB\nCCCCC", str)
	}
}

// Zig: test "Terminal: scrollUp with max_scrollback zero and top margin"
func TestTm1_Terminal_scrollUp_with_max_scrollback_zero_and_top_margin(t *testing.T) {
	testStdCapacity(t)
	// When max_scrollback is 0 and top margin is set, should use deleteLines path
	term := tm1InitScrollback(t, 5, 5, 0)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("AAAAA")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("BBBBB")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("CCCCC")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DDDDD")))

	// Set top margin (not at row 0)
	term.SetTopAndBottomMargin(2, 5)

	zigTry(t, term.ScrollUp(1))

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer alloc.free(str);
		// First row preserved, rest scrolled
		expectEqualStrings(t, "AAAAA\nCCCCC\nDDDDD", str)
	}
}

// Zig: test "Terminal: scrollUp with max_scrollback zero and left/right margin"
func TestTm1_Terminal_scrollUp_with_max_scrollback_zero_and_left_right_margin(t *testing.T) {
	testStdCapacity(t)
	// When max_scrollback is 0 with left/right margins, uses deleteLines path
	term := tm1InitScrollback(t, 10, 5, 0)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("AAAAABBBBB")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("CCCCCDDDDD")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("EEEEEFFFFF")))

	// Set left/right margins (columns 2-6, 1-indexed = indices 1-5)
	term.Modes.Set(ModesModeEnableLeftAndRightMargin, true)
	term.SetLeftAndRightMargin(2, 6)

	zigTry(t, term.ScrollUp(1))

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer alloc.free(str);
		// cols 1-5 scroll, col 0 and cols 6+ preserved
		expectEqualStrings(t, "ACCCCDBBBB\nCEEEEFDDDD\nE     FFFF", str)
	}
}

// Zig: test "Terminal: scrollDown simple"
func TestTm1_Terminal_scrollDown_simple(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI")))
	term.SetCursorPos(2, 2)

	cursor := term.Screens.Active.Cursor
	term.clearDirty()
	term.ScrollDown(1)
	expectEqual(t, cursor.X, term.Screens.Active.Cursor.X)
	expectEqual(t, cursor.Y, term.Screens.Active.Cursor.Y)

	for y := 0; y < 5; y++ {
		expect(t, term.isDirty(tm1Active(0, y)), "t.isDirty(.{ .active = .{ .x = 0, .y = @intCast(y) } })")
	}

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\nABC\nDEF\nGHI", str)
	}
}

// Zig: test "Terminal: scrollDown hyperlink moves"
func TestTm1_Terminal_scrollDown_hyperlink_moves(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://example.com"), None[[]uint8]()))
	zigTry(t, term.PrintString([]uint8("ABC")))
	term.Screens.Active.EndHyperlink()
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI")))
	term.SetCursorPos(2, 2)
	term.ScrollDown(1)

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "\nABC\nDEF\nGHI", str)
	}

	for x := 0; x < 3; x++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Viewport(x, 1)))
		row := list_cell.Row
		expect(t, row.hyperlink, "row.hyperlink")
		cell := list_cell.Cell
		expect(t, cell.Hyperlink(), "cell.hyperlink")
		id := zigUnwrap(t, list_cell.Node.data.LookupHyperlink(cell))
		expectEqual(t, 1, id)
		page := &list_cell.Node.data
		expectEqual(t, 1, page.hyperlink_set.Count())
	}
	for x := 0; x < 3; x++ {
		list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Viewport(x, 0)))
		row := list_cell.Row
		expect(t, !row.hyperlink, "!row.hyperlink")
		cell := list_cell.Cell
		expect(t, !cell.Hyperlink(), "!cell.hyperlink")
		id := list_cell.Node.data.LookupHyperlink(cell)
		expect(t, !id.Valid, "id == null")
	}
}

// Zig: test "Terminal: scrollDown outside of scroll region"
func TestTm1_Terminal_scrollDown_outside_of_scroll_region(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 5, 5)
	defer term.Deinit()

	zigTry(t, term.PrintString([]uint8("ABC")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI")))
	term.SetTopAndBottomMargin(3, 4)
	term.SetCursorPos(2, 2)

	cursor := term.Screens.Active.Cursor
	term.clearDirty()
	term.ScrollDown(1)
	expectEqual(t, cursor.X, term.Screens.Active.Cursor.X)
	expectEqual(t, cursor.Y, term.Screens.Active.Cursor.Y)

	expect(t, !term.isDirty(tm1Active(0, 0)), "!t.isDirty(.{ .active = .{ .x = 0, .y = 0 } })")

	// This is dirty because the cursor moves from this row
	expect(t, term.isDirty(tm1Active(0, 1)), "t.isDirty(.{ .active = .{ .x = 0, .y = 1 } })")
	expect(t, term.isDirty(tm1Active(0, 2)), "t.isDirty(.{ .active = .{ .x = 0, .y = 2 } })")
	expect(t, term.isDirty(tm1Active(0, 3)), "t.isDirty(.{ .active = .{ .x = 0, .y = 3 } })")

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "ABC\nDEF\n\nGHI", str)
	}
}

// Zig: test "Terminal: scrollDown left/right scroll region"
func TestTm1_Terminal_scrollDown_left_right_scroll_region(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 10, 10)
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

	cursor := term.Screens.Active.Cursor
	term.clearDirty()
	term.ScrollDown(1)
	expectEqual(t, cursor.X, term.Screens.Active.Cursor.X)
	expectEqual(t, cursor.Y, term.Screens.Active.Cursor.Y)

	for y := 0; y < 4; y++ {
		expect(t, term.isDirty(tm1Active(0, y)), "t.isDirty(.{ .active = .{ .x = 0, .y = @intCast(y) } })")
	}

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "A   23\nDBC156\nGEF489\n HI7", str)
	}
}

// Zig: test "Terminal: scrollDown left/right scroll region hyperlink"
func TestTm1_Terminal_scrollDown_left_right_scroll_region_hyperlink(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 10, 10)
	defer term.Deinit()

	zigTry(t, term.Screens.Active.StartHyperlink([]uint8("http://example.com"), None[[]uint8]()))
	zigTry(t, term.PrintString([]uint8("ABC123")))
	term.Screens.Active.EndHyperlink()
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("DEF456")))
	term.CarriageReturn()
	zigTry(t, term.Linefeed())
	zigTry(t, term.PrintString([]uint8("GHI789")))
	term.ScrollingRegion.Left = 1
	term.ScrollingRegion.Right = 3
	term.SetCursorPos(2, 2)
	term.ScrollDown(1)

	{
		str := tm1PlainString(t, term)
		// Zig-only: defer testing.allocator.free(str);
		expectEqualStrings(t, "A   23\nDBC156\nGEF489\n HI7", str)
	}

	// First row preserves hyperlink where we didn't scroll
	{
		for x := 0; x < 1; x++ {
			list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Viewport(x, 0)))
			row := list_cell.Row
			expect(t, row.hyperlink, "row.hyperlink")
			cell := list_cell.Cell
			expect(t, cell.Hyperlink(), "cell.hyperlink")
			id := zigUnwrap(t, list_cell.Node.data.LookupHyperlink(cell))
			expectEqual(t, 1, id)
			page := &list_cell.Node.data
			expectEqual(t, 1, page.hyperlink_set.Count())
		}
		for x := 1; x < 4; x++ {
			list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Viewport(x, 0)))
			cell := list_cell.Cell
			expect(t, !cell.Hyperlink(), "!cell.hyperlink")
			id := list_cell.Node.data.LookupHyperlink(cell)
			expect(t, !id.Valid, "id == null")
		}
		for x := 4; x < 6; x++ {
			list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Viewport(x, 0)))
			row := list_cell.Row
			expect(t, row.hyperlink, "row.hyperlink")
			cell := list_cell.Cell
			expect(t, cell.Hyperlink(), "cell.hyperlink")
			id := zigUnwrap(t, list_cell.Node.data.LookupHyperlink(cell))
			expectEqual(t, 1, id)
			page := &list_cell.Node.data
			expectEqual(t, 1, page.hyperlink_set.Count())
		}
	}

	// Second row gets some hyperlinks
	{
		for x := 0; x < 1; x++ {
			list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Viewport(x, 1)))
			cell := list_cell.Cell
			expect(t, !cell.Hyperlink(), "!cell.hyperlink")
			id := list_cell.Node.data.LookupHyperlink(cell)
			expect(t, !id.Valid, "id == null")
		}
		for x := 1; x < 4; x++ {
			list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Viewport(x, 1)))
			row := list_cell.Row
			expect(t, row.hyperlink, "row.hyperlink")
			cell := list_cell.Cell
			expect(t, cell.Hyperlink(), "cell.hyperlink")
			id := zigUnwrap(t, list_cell.Node.data.LookupHyperlink(cell))
			expectEqual(t, 1, id)
			page := &list_cell.Node.data
			expectEqual(t, 1, page.hyperlink_set.Count())
		}
		for x := 4; x < 6; x++ {
			list_cell := zigUnwrap(t, term.Screens.Active.Pages.GetCell(tm1Viewport(x, 1)))
			cell := list_cell.Cell
			expect(t, !cell.Hyperlink(), "!cell.hyperlink")
			id := list_cell.Node.data.LookupHyperlink(cell)
			expect(t, !id.Valid, "id == null")
		}
	}
}

// Go-only: panic test: invokeCharset: a single shift into GR (assert(active == .GL))
func TestTm1_panic_invokeCharset_single_shift_with_GR(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 10, 10)
	defer term.Deinit()

	tm1ExpectPanic(t, func() {
		term.InvokeCharset(CharsetsActiveSlotGR, CharsetsSlotsG1, true)
	})
}

// Go-only: panic test: index: semantic_content_clear_eol set while the cursor content is output (assert(!screen.cursor.semantic_content_clear_eol))
func TestTm1_panic_index_clear_eol_in_output_mode(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 10, 10)
	defer term.Deinit()

	term.Screens.Active.Cursor.SemanticContent = PageCellSemanticContentOutput
	term.Screens.Active.Cursor.SemanticContentClearEol = true
	tm1ExpectPanic(t, func() {
		_ = term.Index()
	})
}

// Go-only: panic test: printCell: overwriting a spacer tail in column 0 (assert(self.screens.active.cursor.x > 0))
func TestTm1_panic_printCell_spacer_tail_at_column_zero(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 10, 10)
	defer term.Deinit()

	term.Screens.Active.Cursor.PageCell.setWide(PageCellWideSpacerTail)
	tm1ExpectPanic(t, func() {
		term.printCell('A', PageCellWideNarrow)
	})
}

// Go-only: panic test: printCell: the cell has a style but its row is not flagged styled (assert(self.screens.active.cursor.page_row.styled))
func TestTm1_panic_printCell_styled_cell_in_unstyled_row(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 10, 10)
	defer term.Deinit()

	term.Screens.Active.Cursor.PageCell.setStyleID(1)
	tm1ExpectPanic(t, func() {
		term.printCell('A', PageCellWideNarrow)
	})
}

// Go-only: panic test: print: the previous cell holds grapheme data that has a break inside it (assert(!unicode.graphemeBreak(cp1, cp2, &state)))
func TestTm1_panic_print_previous_grapheme_has_a_break(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 10, 10)
	defer term.Deinit()

	term.Modes.Set(ModesModeGraphemeCluster, true)
	zigTry(t, term.Print('a'))
	page := &term.Screens.Active.Cursor.PagePin.Node.data
	rac := page.GetRowAndCell(0, uint(term.Screens.Active.Cursor.Y))
	// 'a' followed by 'b' is a grapheme break, which print never stores.
	zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 'b'))
	tm1ExpectPanic(t, func() {
		_ = term.Print(0x0300)
	})
}

// Go-only: panic test: rowWillBeShifted: a spacer tail in the left margin column 0 (cells[left - 1] overflows)
func TestTm1_panic_rowWillBeShifted_spacer_tail_at_left_zero(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 10, 10)
	defer term.Deinit()

	page := &term.Screens.Active.Cursor.PagePin.Node.data
	rac := page.GetRowAndCell(0, 0)
	rac.cell.setWide(PageCellWideSpacerTail)
	tm1ExpectPanic(t, func() {
		term.rowWillBeShifted(page, rac.row)
	})
}

// Go-only: panic test: insertLines: the scrolling region bottom is past the last row (page_pin.down(rem - 1).? is null)
func TestTm1_panic_insertLines_scrolling_region_bottom_past_screen(t *testing.T) {
	testStdCapacity(t)
	term := tm1Init(t, 10, 10)
	defer term.Deinit()

	term.ScrollingRegion.Bottom = 100
	tm1ExpectPanic(t, func() {
		term.InsertLines(1)
	})
}
