// Tests translated one for one from src/terminal/Screen.zig (Ghostty v1.3.1
// plus the term51 additional tests), the tests from "Screen: selectWord"
// up to and including "term51: cursorScrollAbove: fills the new row only for
// a background colour". Each Go test carries the exact Zig title in a
// "// Zig: test" comment. Helpers are prefixed sc2.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualSlices / expectEqualStrings keep Zig's names
//     and Zig's (expected, actual) argument order. `try` on a call that
//     returns an error is zigTry(t, err); `x.?` is zigUnwrap.
//   - Every test starts with testStdCapacity(t): Zig's unit tests run with a
//     standard page capacity of grapheme_bytes 512.
//   - `init(alloc, .{ ... })` is sc2Init; the allocator parameters are gone,
//     `defer s.deinit()` is `defer s.Deinit()`, and std.testing.allocator
//     leak checking has no counterpart (nothing is written for it).
//   - `s.pages.pin(pt).?` is sc2Pin; a point literal
//     `.{ .active = .{ .x = x, .y = y } }` is sc2Active(x, y) (sc2Screen and
//     sc2Viewport likewise); `try s.testWriteString(str)` is sc2Write.
//   - Raw-pointer, pool and memory-offset assertions are dropped with a
//     `// Zig-only:` comment; pointer comparisons of nodes, rows and cells
//     stay pointer comparisons of the Go pointers.
//   - Zig's default option values are written out in every literal
//     (Resize.reflow = true, prompt_redraw = .false, trim = true, ...).
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt.

package terminal

import (
	"math"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers.
// ---------------------------------------------------------------------------

// sc2ExpectPanic runs f and fails the test if f does not panic.
func sc2ExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if recover() == nil {
			t.Fatalf("expected a panic, got none")
		}
	}()
	f()
}

// sc2Active is the Zig literal `.{ .active = .{ .x = x, .y = y } }`.
func sc2Active(x int, y int) Point {
	return Point{Tag: PointTagActive, Active: PointCoordinate{x: SizeCellCountInt(x), y: uint32(y)}}
}

// sc2Screen is the Zig literal `.{ .screen = .{ .x = x, .y = y } }`.
func sc2Screen(x int, y int) Point {
	return Point{Tag: PointTagScreen, Screen: PointCoordinate{x: SizeCellCountInt(x), y: uint32(y)}}
}

// sc2Viewport is the Zig literal `.{ .viewport = .{ .x = x, .y = y } }`.
func sc2Viewport(x int, y int) Point {
	return Point{Tag: PointTagViewport, Viewport: PointCoordinate{x: SizeCellCountInt(x), y: uint32(y)}}
}

// sc2Init is `try init(alloc, .{ .cols = cols, .rows = rows, .max_scrollback =
// max_scrollback })`, with Options.kitty_image_storage_limit at its Zig
// default. The screen is returned by pointer so the test can use it as `&s`.
func sc2Init(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt, max_scrollback uint64) *Screen {
	t.Helper()
	return sc2InitWith(t, ScreenOptions{
		cols:                      cols,
		rows:                      rows,
		max_scrollback:            max_scrollback,
		kitty_image_storage_limit: 320 * 1000 * 1000,
	})
}

// sc2InitWith is `try init(alloc, opts)`.
func sc2InitWith(t *testing.T, opts ScreenOptions) *Screen {
	t.Helper()
	s, err := ScreenInit(opts)
	zigTry(t, err)
	return &s
}

// sc2Write is `try s.testWriteString(str)`.
func sc2Write(t *testing.T, s *Screen, str string) {
	t.Helper()
	zigTry(t, s.TestWriteString([]uint8(str)))
}

// sc2Pin is `s.pages.pin(pt).?`.
func sc2Pin(t *testing.T, s *Screen, pt Point) PageListPin {
	t.Helper()
	return zigUnwrap(t, s.pages.Pin(pt))
}

// sc2PointFromPin is `s.pages.pointFromPin(tag, pin).?`.
func sc2PointFromPin(t *testing.T, s *Screen, tag PointTag, pin PageListPin) Point {
	t.Helper()
	return zigUnwrap(t, s.pages.PointFromPin(tag, pin))
}

// sc2SelectionString is `try s.selectionString(alloc, .{ .sel = sel, .trim =
// trim })`, as a string.
func sc2SelectionString(t *testing.T, s *Screen, sel Selection, trim bool) string {
	t.Helper()
	out, err := s.SelectionString(ScreenSelectionString{sel: sel, trim: trim})
	zigTry(t, err)
	return string(out)
}

// sc2DumpString is `try s.dumpStringAlloc(alloc, pt)`, as a string.
func sc2DumpString(t *testing.T, s *Screen, pt Point) string {
	t.Helper()
	out, err := s.DumpStringAlloc(pt)
	zigTry(t, err)
	return string(out)
}

// sc2SelectionInit is `Selection.init(<pin at a>, <pin at b>, rect)`.
func sc2SelectionInit(t *testing.T, s *Screen, a Point, b Point, rect bool) Selection {
	t.Helper()
	return SelectionInit(sc2Pin(t, s, a), sc2Pin(t, s, b), rect)
}

// sc2BoundaryCodepoints is the "default boundary codepoints for word
// selection" array of the selectWord tests.
func sc2BoundaryCodepoints() []uint32 {
	return []uint32{
		0, ' ', '\t', '\'', '"', '│', '`', '|', ':', ';',
		',', '(', ')', '[', ']', '{', '}', '<', '>', '$',
	}
}

// sc2Prompt is the Zig literal `.{ .prompt = kind }` for cursorSetSemanticContent.
func sc2Prompt(kind OSCParsersSemanticPromptPromptKind) ScreenCursorSetSemanticContent {
	return ScreenCursorSetSemanticContent{Tag: ScreenCursorSetSemanticContentTagPrompt, Prompt: kind}
}

// sc2Output is the Zig literal `.output` for cursorSetSemanticContent.
func sc2Output() ScreenCursorSetSemanticContent {
	return ScreenCursorSetSemanticContent{Tag: ScreenCursorSetSemanticContentTagOutput}
}

// sc2Input is the Zig literal `.{ .input = kind }` for cursorSetSemanticContent.
func sc2Input(kind ScreenCursorSetSemanticContentInput) ScreenCursorSetSemanticContent {
	return ScreenCursorSetSemanticContent{Tag: ScreenCursorSetSemanticContentTagInput, Input: kind}
}

// sc2ClickLine is the Zig literal `.{ .cl = .line }` for
// Screen.semantic_prompt.click.
func sc2ClickLine() ScreenSemanticPromptSemanticClick {
	return ScreenSemanticPromptSemanticClick{
		Tag: ScreenSemanticPromptSemanticClickCl,
		Cl:  OSCParsersSemanticPromptClickLine,
	}
}

// sc2Bold is `.bold` for setAttribute.
func sc2Bold() SGRAttribute {
	return SGRAttribute{Tag: SGRAttributeTagBold}
}

// sc2RgbBg is the style `.{ .bg_color = .{ .rgb = @bitCast(n) } }` where n is
// a u24: r is the low byte, b the high byte.
func sc2RgbBg(n uint32) Style {
	return Style{bg_color: StyleColor{
		Tag: StyleColorRgb,
		Rgb: ColorRGB{r: uint8(n & 0xFF), g: uint8((n >> 8) & 0xFF), b: uint8((n >> 16) & 0xFF)},
	}}
}

// sc2StartHyperlink is `try s.startHyperlink(uri, null)`.
func sc2StartHyperlink(t *testing.T, s *Screen, uri string) {
	t.Helper()
	zigTry(t, s.StartHyperlink([]uint8(uri), None[[]uint8]()))
}

// sc2Resize is `try s.resize(.{ .cols = cols, .rows = rows })`.
func sc2Resize(t *testing.T, s *Screen, cols SizeCellCountInt, rows SizeCellCountInt) {
	t.Helper()
	zigTry(t, s.Resize(screenResizeDefault(cols, rows)))
}

// sc2IncreaseCapacity is `try s.increaseCapacity(node, adjustment)`.
func sc2IncreaseCapacity(t *testing.T, s *Screen, node *PageListNode, adj PageListIncreaseCapacity) *PageListNode {
	t.Helper()
	n, err := s.IncreaseCapacity(node, Some(adj))
	zigTry(t, err)
	return n
}

// sc2ExpectSelection checks both ends of a selection against points of the
// given tag: the two `expectEqual(point.Point{ ... }, s.pages.pointFromPin(tag,
// sel.start()/sel.end()).?)` assertions.
func sc2ExpectSelection(t *testing.T, s *Screen, tag PointTag, sel Selection, start Point, end Point) {
	t.Helper()
	expectEqualPoint(t, start, sc2PointFromPin(t, s, tag, sel.Start()))
	expectEqualPoint(t, end, sc2PointFromPin(t, s, tag, sel.End()))
}

// ---------------------------------------------------------------------------
// Upstream tests.
// ---------------------------------------------------------------------------

// Zig: test "Screen: selectWord"
func TestSc2_Screen_selectWord(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 10, 0)
	defer s.Deinit()
	sc2Write(t, s, "ABC  DEF\n 123\n456")

	// Default boundary codepoints for word selection
	boundary_codepoints := sc2BoundaryCodepoints()

	// Outside of active area
	// try testing.expect(s.selectWord(.{ .x = 9, .y = 0 }) == null);
	// try testing.expect(s.selectWord(.{ .x = 0, .y = 5 }) == null);

	// Going forward
	{
		sel := zigUnwrap(t, s.SelectWord(sc2Pin(t, s, sc2Active(0, 0)), boundary_codepoints))
		defer sel.Deinit(s)
		sc2ExpectSelection(t, s, PointTagScreen, sel, sc2Screen(0, 0), sc2Screen(2, 0))
	}

	// Going backward
	{
		sel := zigUnwrap(t, s.SelectWord(sc2Pin(t, s, sc2Active(2, 0)), boundary_codepoints))
		defer sel.Deinit(s)
		sc2ExpectSelection(t, s, PointTagScreen, sel, sc2Screen(0, 0), sc2Screen(2, 0))
	}

	// Going forward and backward
	{
		sel := zigUnwrap(t, s.SelectWord(sc2Pin(t, s, sc2Active(1, 0)), boundary_codepoints))
		defer sel.Deinit(s)
		sc2ExpectSelection(t, s, PointTagScreen, sel, sc2Screen(0, 0), sc2Screen(2, 0))
	}

	// Whitespace
	{
		sel := zigUnwrap(t, s.SelectWord(sc2Pin(t, s, sc2Active(3, 0)), boundary_codepoints))
		defer sel.Deinit(s)
		sc2ExpectSelection(t, s, PointTagScreen, sel, sc2Screen(3, 0), sc2Screen(4, 0))
	}

	// Whitespace single char
	{
		sel := zigUnwrap(t, s.SelectWord(sc2Pin(t, s, sc2Active(0, 1)), boundary_codepoints))
		defer sel.Deinit(s)
		sc2ExpectSelection(t, s, PointTagScreen, sel, sc2Screen(0, 1), sc2Screen(0, 1))
	}

	// End of screen
	{
		sel := zigUnwrap(t, s.SelectWord(sc2Pin(t, s, sc2Active(1, 2)), boundary_codepoints))
		defer sel.Deinit(s)
		sc2ExpectSelection(t, s, PointTagScreen, sel, sc2Screen(0, 2), sc2Screen(2, 2))
	}
}

// Zig: test "Screen: selectWord across soft-wrap"
func TestSc2_Screen_selectWord_across_soft_wrap(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 10, 0)
	defer s.Deinit()
	sc2Write(t, s, " 1234012\n 123")

	// Default boundary codepoints for word selection
	boundary_codepoints := sc2BoundaryCodepoints()

	{
		contents := sc2DumpString(t, s, Point{Tag: PointTagScreen})
		expectEqualStrings(t, " 1234\n012\n 123", contents)
	}

	// Going forward
	{
		sel := zigUnwrap(t, s.SelectWord(sc2Pin(t, s, sc2Active(1, 0)), boundary_codepoints))
		defer sel.Deinit(s)
		sc2ExpectSelection(t, s, PointTagScreen, sel, sc2Screen(1, 0), sc2Screen(2, 1))
	}

	// Going backward
	{
		sel := zigUnwrap(t, s.SelectWord(sc2Pin(t, s, sc2Active(1, 1)), boundary_codepoints))
		defer sel.Deinit(s)
		sc2ExpectSelection(t, s, PointTagScreen, sel, sc2Screen(1, 0), sc2Screen(2, 1))
	}

	// Going forward and backward
	{
		sel := zigUnwrap(t, s.SelectWord(sc2Pin(t, s, sc2Active(3, 0)), boundary_codepoints))
		defer sel.Deinit(s)
		sc2ExpectSelection(t, s, PointTagScreen, sel, sc2Screen(1, 0), sc2Screen(2, 1))
	}
}

// Zig: test "Screen: selectWord whitespace across soft-wrap"
func TestSc2_Screen_selectWord_whitespace_across_soft_wrap(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 10, 0)
	defer s.Deinit()
	sc2Write(t, s, "1       1\n 123")

	// Default boundary codepoints for word selection
	boundary_codepoints := sc2BoundaryCodepoints()

	// Going forward
	{
		sel := zigUnwrap(t, s.SelectWord(sc2Pin(t, s, sc2Active(1, 0)), boundary_codepoints))
		defer sel.Deinit(s)
		sc2ExpectSelection(t, s, PointTagScreen, sel, sc2Screen(1, 0), sc2Screen(2, 1))
	}

	// Going backward
	{
		sel := zigUnwrap(t, s.SelectWord(sc2Pin(t, s, sc2Active(1, 1)), boundary_codepoints))
		defer sel.Deinit(s)
		sc2ExpectSelection(t, s, PointTagScreen, sel, sc2Screen(1, 0), sc2Screen(2, 1))
	}

	// Going forward and backward
	{
		sel := zigUnwrap(t, s.SelectWord(sc2Pin(t, s, sc2Active(3, 0)), boundary_codepoints))
		defer sel.Deinit(s)
		sc2ExpectSelection(t, s, PointTagScreen, sel, sc2Screen(1, 0), sc2Screen(2, 1))
	}
}

// Zig: test "Screen: selectWord with character boundary"
func TestSc2_Screen_selectWord_with_character_boundary(t *testing.T) {
	testStdCapacity(t)

	// Default boundary codepoints for word selection
	boundary_codepoints := sc2BoundaryCodepoints()

	cases := []string{
		" 'abc' \n123",
		" \"abc\" \n123",
		" │abc│ \n123",
		" `abc` \n123",
		" |abc| \n123",
		" :abc: \n123",
		" ;abc; \n123",
		" ,abc, \n123",
		" (abc( \n123",
		" )abc) \n123",
		" [abc[ \n123",
		" ]abc] \n123",
		" {abc{ \n123",
		" }abc} \n123",
		" <abc< \n123",
		" >abc> \n123",
		" $abc$ \n123",
	}

	for _, case_ := range cases {
		func() {
			s := sc2Init(t, 20, 10, 0)
			defer s.Deinit()
			sc2Write(t, s, case_)

			// Inside character forward
			{
				sel := zigUnwrap(t, s.SelectWord(sc2Pin(t, s, sc2Active(2, 0)), boundary_codepoints))
				defer sel.Deinit(s)
				sc2ExpectSelection(t, s, PointTagScreen, sel, sc2Screen(2, 0), sc2Screen(4, 0))
			}

			// Inside character backward
			{
				sel := zigUnwrap(t, s.SelectWord(sc2Pin(t, s, sc2Active(4, 0)), boundary_codepoints))
				defer sel.Deinit(s)
				sc2ExpectSelection(t, s, PointTagScreen, sel, sc2Screen(2, 0), sc2Screen(4, 0))
			}

			// Inside character bidirectional
			{
				sel := zigUnwrap(t, s.SelectWord(sc2Pin(t, s, sc2Active(3, 0)), boundary_codepoints))
				defer sel.Deinit(s)
				sc2ExpectSelection(t, s, PointTagScreen, sel, sc2Screen(2, 0), sc2Screen(4, 0))
			}

			// On quote
			// NOTE: this behavior is not ideal, so we can change this one day,
			// but I think its also not that important compared to the above.
			{
				sel := zigUnwrap(t, s.SelectWord(sc2Pin(t, s, sc2Active(1, 0)), boundary_codepoints))
				defer sel.Deinit(s)
				sc2ExpectSelection(t, s, PointTagScreen, sel, sc2Screen(0, 0), sc2Screen(1, 0))
			}
		}()
	}
}

// Zig: test "Screen: selectOutput"
func TestSc2_Screen_selectOutput(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 15, 0)
	defer s.Deinit()

	// Build content with cell-level semantic content:
	// Row 0-1: output1 (output)
	// Row 2: prompt2 (prompt)
	// Row 3: input2 (input)
	// Row 4-7: output2 (output, with overflow causing wrap)
	// Row 8: "$ " (prompt) + "input3" (input)
	// Row 9-11: output3 (output)
	s.CursorSetSemanticContent(sc2Output())
	sc2Write(t, s, "output1\n")
	sc2Write(t, s, "output1\n")
	s.CursorSetSemanticContent(sc2Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc2Write(t, s, "prompt2\n")
	s.CursorSetSemanticContent(sc2Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc2Write(t, s, "input2\n")
	s.CursorSetSemanticContent(sc2Output())
	sc2Write(t, s, "output2output2output2output2\n")
	sc2Write(t, s, "output2\n")
	s.CursorSetSemanticContent(sc2Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc2Write(t, s, "$ ")
	s.CursorSetSemanticContent(sc2Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc2Write(t, s, "input3\n")
	s.CursorSetSemanticContent(sc2Output())
	sc2Write(t, s, "output3\n")
	sc2Write(t, s, "output3\n")
	sc2Write(t, s, "output3")

	// First output block (rows 0-1), should select those rows
	{
		sel := zigUnwrap(t, s.SelectOutput(sc2Pin(t, s, sc2Active(1, 1))))
		defer sel.Deinit(s)
		contents := sc2SelectionString(t, s, sel, false)
		expectEqualStrings(t, "output1\noutput1", contents)
	}
	// Second output block (rows 4-7)
	{
		sel := zigUnwrap(t, s.SelectOutput(sc2Pin(t, s, sc2Active(3, 7))))
		defer sel.Deinit(s)
		contents := sc2SelectionString(t, s, sel, false)
		expectEqualStrings(t, "output2output2output2output2\noutput2", contents)
	}
	// Third output block (rows 9-11)
	{
		sel := zigUnwrap(t, s.SelectOutput(sc2Pin(t, s, sc2Active(2, 10))))
		defer sel.Deinit(s)
		sc2ExpectSelection(t, s, PointTagActive, sel, sc2Active(0, 9), sc2Active(6, 11))
	}
	// Click on prompt should return null
	{
		expect(t, !s.SelectOutput(sc2Pin(t, s, sc2Active(1, 8))).Valid,
			"s.selectOutput(s.pages.pin(.{ .active = .{ .x = 1, .y = 8 } }).?) == null")
	}
	// Click on input should return null
	{
		expect(t, !s.SelectOutput(sc2Pin(t, s, sc2Active(5, 8))).Valid,
			"s.selectOutput(s.pages.pin(.{ .active = .{ .x = 5, .y = 8 } }).?) == null")
	}
}

// Zig: test "Screen: selectionString basic"
func TestSc2_Screen_selectionString_basic(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 3, 0)
	defer s.Deinit()
	str := "1ABCD\n2EFGH\n3IJKL"
	sc2Write(t, s, str)

	{
		sel := SelectionInit(
			sc2Pin(t, s, sc2Screen(0, 1)),
			sc2Pin(t, s, sc2Screen(2, 2)),
			false,
		)
		contents := sc2SelectionString(t, s, sel, true)
		expected := "2EFGH\n3IJ"
		expectEqualStrings(t, expected, contents)
	}
}

// Zig: test "Screen: selectionString start outside of written area"
func TestSc2_Screen_selectionString_start_outside_of_written_area(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 10, 0)
	defer s.Deinit()
	str := "1ABCD\n2EFGH\n3IJKL"
	sc2Write(t, s, str)

	{
		sel := SelectionInit(
			sc2Pin(t, s, sc2Screen(0, 5)),
			sc2Pin(t, s, sc2Screen(2, 6)),
			false,
		)
		contents := sc2SelectionString(t, s, sel, true)
		expected := ""
		expectEqualStrings(t, expected, contents)
	}
}

// Zig: test "Screen: selectionString end outside of written area"
func TestSc2_Screen_selectionString_end_outside_of_written_area(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 10, 0)
	defer s.Deinit()
	str := "1ABCD\n2EFGH\n3IJKL"
	sc2Write(t, s, str)

	{
		sel := SelectionInit(
			sc2Pin(t, s, sc2Screen(0, 2)),
			sc2Pin(t, s, sc2Screen(2, 6)),
			false,
		)
		contents := sc2SelectionString(t, s, sel, true)
		expected := "3IJKL"
		expectEqualStrings(t, expected, contents)
	}
}

// Zig: test "Screen: selectionString trim space"
func TestSc2_Screen_selectionString_trim_space(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 3, 0)
	defer s.Deinit()
	str := "1AB  \n2EFGH\n3IJKL"
	sc2Write(t, s, str)

	sel := SelectionInit(
		sc2Pin(t, s, sc2Screen(0, 0)),
		sc2Pin(t, s, sc2Screen(2, 1)),
		false,
	)

	{
		contents := sc2SelectionString(t, s, sel, true)
		expected := "1AB\n2EF"
		expectEqualStrings(t, expected, contents)
	}

	// No trim
	{
		contents := sc2SelectionString(t, s, sel, false)
		expected := "1AB  \n2EF"
		expectEqualStrings(t, expected, contents)
	}
}

// Zig: test "Screen: selectionString trim empty line"
func TestSc2_Screen_selectionString_trim_empty_line(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 5, 0)
	defer s.Deinit()
	str := "1AB  \n\n2EFGH\n3IJKL"
	sc2Write(t, s, str)

	sel := SelectionInit(
		sc2Pin(t, s, sc2Screen(0, 0)),
		sc2Pin(t, s, sc2Screen(2, 2)),
		false,
	)

	{
		contents := sc2SelectionString(t, s, sel, true)
		expected := "1AB\n\n2EF"
		expectEqualStrings(t, expected, contents)
	}

	// No trim
	{
		contents := sc2SelectionString(t, s, sel, false)
		expected := "1AB  \n\n2EF"
		expectEqualStrings(t, expected, contents)
	}
}

// Zig: test "Screen: selectionString soft wrap"
func TestSc2_Screen_selectionString_soft_wrap(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 3, 0)
	defer s.Deinit()
	str := "1ABCD2EFGH3IJKL"
	sc2Write(t, s, str)

	{
		sel := SelectionInit(
			sc2Pin(t, s, sc2Screen(0, 1)),
			sc2Pin(t, s, sc2Screen(2, 2)),
			false,
		)
		contents := sc2SelectionString(t, s, sel, true)
		expected := "2EFGH3IJ"
		expectEqualStrings(t, expected, contents)
	}
}

// Zig: test "Screen: selectionString wide char"
func TestSc2_Screen_selectionString_wide_char(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 3, 0)
	defer s.Deinit()
	str := "1A⚡"
	sc2Write(t, s, str)

	{
		sel := SelectionInit(
			sc2Pin(t, s, sc2Screen(0, 0)),
			sc2Pin(t, s, sc2Screen(3, 0)),
			false,
		)
		contents := sc2SelectionString(t, s, sel, true)
		expected := str
		expectEqualStrings(t, expected, contents)
	}

	{
		sel := SelectionInit(
			sc2Pin(t, s, sc2Screen(0, 0)),
			sc2Pin(t, s, sc2Screen(2, 0)),
			false,
		)
		contents := sc2SelectionString(t, s, sel, true)
		expected := str
		expectEqualStrings(t, expected, contents)
	}

	{
		sel := SelectionInit(
			sc2Pin(t, s, sc2Screen(3, 0)),
			sc2Pin(t, s, sc2Screen(3, 0)),
			false,
		)
		contents := sc2SelectionString(t, s, sel, true)
		expected := "⚡"
		expectEqualStrings(t, expected, contents)
	}
}

// Zig: test "Screen: selectionString wide char with header"
func TestSc2_Screen_selectionString_wide_char_with_header(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 3, 0)
	defer s.Deinit()
	str := "1ABC⚡"
	sc2Write(t, s, str)

	{
		sel := SelectionInit(
			sc2Pin(t, s, sc2Screen(0, 0)),
			sc2Pin(t, s, sc2Screen(4, 0)),
			false,
		)
		contents := sc2SelectionString(t, s, sel, true)
		expected := str
		expectEqualStrings(t, expected, contents)
	}
}

// https://github.com/mitchellh/ghostty/issues/289
//
// Zig: test "Screen: selectionString empty with soft wrap"
func TestSc2_Screen_selectionString_empty_with_soft_wrap(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 2, 0)
	defer s.Deinit()

	// Let me describe the situation that caused this because this
	// test is not obvious. By writing an emoji below, we introduce
	// one cell with the emoji and one cell as a "wide char spacer".
	// We then soft wrap the line by writing spaces.
	//
	// By selecting only the tail, we'd select nothing and we had
	// a logic error that would cause a crash.
	sc2Write(t, s, "👨")
	sc2Write(t, s, "      ")

	{
		sel := SelectionInit(
			sc2Pin(t, s, sc2Screen(1, 0)),
			sc2Pin(t, s, sc2Screen(2, 0)),
			false,
		)
		contents := sc2SelectionString(t, s, sel, true)
		expected := "👨"
		expectEqualStrings(t, expected, contents)
	}
}

// Zig: test "Screen: selectionString with zero width joiner"
func TestSc2_Screen_selectionString_with_zero_width_joiner(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 1, 0)
	defer s.Deinit()
	str := "👨‍" // this has a ZWJ
	sc2Write(t, s, str)

	// Integrity check
	{
		pin := sc2Pin(t, s, Point{Tag: PointTagScreen, Screen: PointCoordinate{y: 0, x: 0}})
		cell := pin.RowAndCell().cell
		expectEqual(t, uint32(0x1F468), cell.content.codepoint)
		expectEqual(t, PageCellWideWide, cell.wide)
		cps := zigUnwrap(t, pin.node.data.LookupGrapheme(cell))
		expectEqual(t, 1, len(cps))
	}

	// The real test
	{
		sel := SelectionInit(
			sc2Pin(t, s, sc2Screen(0, 0)),
			sc2Pin(t, s, sc2Screen(1, 0)),
			false,
		)
		contents := sc2SelectionString(t, s, sel, true)
		expected := "👨‍"
		expectEqualStrings(t, expected, contents)
	}
}

// Zig: test "Screen: selectionString, rectangle, basic"
func TestSc2_Screen_selectionString_rectangle_basic(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 30, 5, 0)
	defer s.Deinit()
	str := "Lorem ipsum dolor\n" +
		"sit amet, consectetur\n" +
		"adipiscing elit, sed do\n" +
		"eiusmod tempor incididunt\n" +
		"ut labore et dolore"
	sel := SelectionInit(
		sc2Pin(t, s, sc2Screen(2, 1)),
		sc2Pin(t, s, sc2Screen(6, 3)),
		true,
	)
	expected := "t ame\n" +
		"ipisc\n" +
		"usmod"
	sc2Write(t, s, str)

	contents := sc2SelectionString(t, s, sel, true)
	expectEqualStrings(t, expected, contents)
}

// Zig: test "Screen: selectionString, rectangle, w/EOL"
func TestSc2_Screen_selectionString_rectangle_w_EOL(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 30, 5, 0)
	defer s.Deinit()
	str := "Lorem ipsum dolor\n" +
		"sit amet, consectetur\n" +
		"adipiscing elit, sed do\n" +
		"eiusmod tempor incididunt\n" +
		"ut labore et dolore"
	sel := SelectionInit(
		sc2Pin(t, s, sc2Screen(12, 0)),
		sc2Pin(t, s, sc2Screen(26, 4)),
		true,
	)
	expected := "dolor\n" +
		"nsectetur\n" +
		"lit, sed do\n" +
		"or incididunt\n" +
		" dolore"
	sc2Write(t, s, str)

	contents := sc2SelectionString(t, s, sel, true)
	expectEqualStrings(t, expected, contents)
}

// Zig: test "Screen: selectionString, rectangle, more complex w/breaks"
func TestSc2_Screen_selectionString_rectangle_more_complex_w_breaks(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 30, 8, 0)
	defer s.Deinit()
	str := "Lorem ipsum dolor\n" +
		"sit amet, consectetur\n" +
		"adipiscing elit, sed do\n" +
		"eiusmod tempor incididunt\n" +
		"ut labore et dolore\n" +
		"\n" +
		"magna aliqua. Ut enim\n" +
		"ad minim veniam, quis"
	sel := SelectionInit(
		sc2Pin(t, s, sc2Screen(11, 2)),
		sc2Pin(t, s, sc2Screen(26, 7)),
		true,
	)
	expected := "elit, sed do\n" +
		"por incididunt\n" +
		"t dolore\n" +
		"\n" +
		"a. Ut enim\n" +
		"niam, quis"
	sc2Write(t, s, str)

	contents := sc2SelectionString(t, s, sel, true)
	expectEqualStrings(t, expected, contents)
}

// Zig: test "Screen: selectionString multi-page"
func TestSc2_Screen_selectionString_multi_page(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 2048)
	defer s.Deinit()

	first_page_size := s.pages.pages.first.data.capacity.rows

	// Lazy way to seek to the first page boundary.
	s.pages.pages.first.data.PauseIntegrityChecks(true)
	for i := 0; i < int(first_page_size)-1; i++ {
		sc2Write(t, s, "\n")
	}
	s.pages.pages.first.data.PauseIntegrityChecks(false)

	sc2Write(t, s, "123456789\n!@#$%^&*(\n123456789")

	{
		sel := SelectionInit(
			sc2Pin(t, s, sc2Active(0, 0)),
			sc2Pin(t, s, sc2Active(2, 2)),
			false,
		)
		contents := sc2SelectionString(t, s, sel, true)
		expected := "123456789\n!@#$%^&*(\n123"
		expectEqualStrings(t, expected, contents)
	}
}

// Zig: test "Screen: lineIterator"
func TestSc2_Screen_lineIterator(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 5, 0)
	defer s.Deinit()
	str := "1ABCD\n2EFGH"
	sc2Write(t, s, str)

	// Test the line iterator
	iter := s.LineIterator(sc2Pin(t, s, sc2Viewport(0, 0)))
	{
		sel := zigUnwrap(t, iter.Next())
		actual := sc2SelectionString(t, s, sel, false)
		expectEqualStrings(t, "1ABCD", actual)
	}
	{
		sel := zigUnwrap(t, iter.Next())
		actual := sc2SelectionString(t, s, sel, false)
		expectEqualStrings(t, "2EFGH", actual)
	}
}

// Zig: test "Screen: lineIterator soft wrap"
func TestSc2_Screen_lineIterator_soft_wrap(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 5, 0)
	defer s.Deinit()
	str := "1ABCD2EFGH\n3ABCD"
	sc2Write(t, s, str)

	// Test the line iterator
	iter := s.LineIterator(sc2Pin(t, s, sc2Viewport(0, 0)))
	{
		sel := zigUnwrap(t, iter.Next())
		actual := sc2SelectionString(t, s, sel, false)
		expectEqualStrings(t, "1ABCD2EFGH", actual)
	}
	{
		sel := zigUnwrap(t, iter.Next())
		actual := sc2SelectionString(t, s, sel, false)
		expectEqualStrings(t, "3ABCD", actual)
	}
	// try testing.expect(iter.next() == null);
}

// Zig: test "Screen: hyperlink start/end"
func TestSc2_Screen_hyperlink_start_end(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 5, 0)
	defer s.Deinit()
	expect(t, s.cursor.hyperlink_id == 0, "s.cursor.hyperlink_id == 0")
	{
		page := &s.cursor.page_pin.node.data
		expectEqual(t, uint(0), page.hyperlink_set.Count())
	}

	sc2StartHyperlink(t, s, "http://example.com")
	expect(t, s.cursor.hyperlink_id != 0, "s.cursor.hyperlink_id != 0")
	{
		page := &s.cursor.page_pin.node.data
		expectEqual(t, uint(1), page.hyperlink_set.Count())
	}

	s.EndHyperlink()
	expect(t, s.cursor.hyperlink_id == 0, "s.cursor.hyperlink_id == 0")
	{
		page := &s.cursor.page_pin.node.data
		expectEqual(t, uint(0), page.hyperlink_set.Count())
	}
}

// Zig: test "Screen: hyperlink reuse"
func TestSc2_Screen_hyperlink_reuse(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 5, 0)
	defer s.Deinit()

	expect(t, s.cursor.hyperlink_id == 0, "s.cursor.hyperlink_id == 0")
	{
		page := &s.cursor.page_pin.node.data
		expectEqual(t, uint(0), page.hyperlink_set.Count())
	}

	// Use it for the first time
	sc2StartHyperlink(t, s, "http://example.com")
	expect(t, s.cursor.hyperlink_id != 0, "s.cursor.hyperlink_id != 0")
	id := s.cursor.hyperlink_id

	// Reuse the same hyperlink, expect we have the same ID
	sc2StartHyperlink(t, s, "http://example.com")
	expectEqual(t, id, s.cursor.hyperlink_id)
	{
		page := &s.cursor.page_pin.node.data
		expectEqual(t, uint(1), page.hyperlink_set.Count())
	}

	s.EndHyperlink()
	expect(t, s.cursor.hyperlink_id == 0, "s.cursor.hyperlink_id == 0")
	{
		page := &s.cursor.page_pin.node.data
		expectEqual(t, uint(0), page.hyperlink_set.Count())
	}
}

// Zig: test "Screen: hyperlink cursor state on resize"
func TestSc2_Screen_hyperlink_cursor_state_on_resize(t *testing.T) {
	testStdCapacity(t)

	// This test depends on underlying PageList implementation so
	// it may be invalid one day. It's here to document/verify the
	// current behavior.

	s := sc2Init(t, 5, 10, 0)
	defer s.Deinit()

	// Start a hyperlink
	sc2StartHyperlink(t, s, "http://example.com")
	expect(t, s.cursor.hyperlink_id != 0, "s.cursor.hyperlink_id != 0")
	{
		page := &s.cursor.page_pin.node.data
		expectEqual(t, uint(1), page.hyperlink_set.Count())
	}

	// Resize. Any column growth will trigger a page to be reallocated.
	sc2Resize(t, s, 10, 10)
	expect(t, s.cursor.hyperlink_id != 0, "s.cursor.hyperlink_id != 0")
	{
		page := &s.cursor.page_pin.node.data
		expectEqual(t, uint(1), page.hyperlink_set.Count())
	}

	s.EndHyperlink()
	expect(t, s.cursor.hyperlink_id == 0, "s.cursor.hyperlink_id == 0")
	{
		page := &s.cursor.page_pin.node.data
		expectEqual(t, uint(0), page.hyperlink_set.Count())
	}
}

// Zig: test "Screen: cursorSetHyperlink OOM + URI too large for string alloc"
func TestSc2_Screen_cursorSetHyperlink_OOM_URI_too_large_for_string_alloc(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 80, 24, 0)
	defer s.Deinit()

	// Start a hyperlink with a URI that just barely fits in the string alloc.
	// This will ensure that additional string alloc space is needed for the
	// redundant copy of the URI when the page is re-alloced.
	uri := strings.Repeat("a", int(PageStdCapacity.string_bytes)-8)
	sc2StartHyperlink(t, s, uri)

	// Figure out how many cells should can have hyperlinks in this page,
	// and write twice that number, to guarantee the capacity needs to be
	// increased at some point.
	base_capacity := s.cursor.page_pin.node.data.HyperlinkCapacity()
	base_string_bytes := s.cursor.page_pin.node.data.capacity.string_bytes
	for i := uint(0); i < base_capacity*2; i++ {
		zigTry(t, s.CursorSetHyperlink())
		if s.cursor.x >= s.pages.cols-1 {
			zigTry(t, s.cursorDownOrScroll())
			s.CursorHorizontalAbsolute(0)
		} else {
			s.CursorRight(1)
		}
	}

	// Make sure the capacity really did increase.
	expect(t, base_capacity < s.cursor.page_pin.node.data.HyperlinkCapacity(),
		"base_capacity < s.cursor.page_pin.node.data.hyperlinkCapacity()")
	// And that our string_bytes increased as well.
	expect(t, base_string_bytes < s.cursor.page_pin.node.data.capacity.string_bytes,
		"base_string_bytes < s.cursor.page_pin.node.data.capacity.string_bytes")
}

// Zig: test "Screen: increaseCapacity cursor style ref count preserved"
func TestSc2_Screen_increaseCapacity_cursor_style_ref_count_preserved(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 5, 0)
	defer s.Deinit()
	zigTry(t, s.SetAttribute(sc2Bold()))
	sc2Write(t, s, "1ABCD")

	// We should have one page and it should be our cursor page
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	expect(t, s.pages.pages.first == s.cursor.page_pin.node, "s.pages.pages.first == s.cursor.page_pin.node")

	old_style := s.cursor.style

	{
		page := &s.pages.pages.last.data
		// 5 chars + cursor = 6 refs
		expectEqual(t, SizeCellCountInt(6), page.styles.RefCount(s.cursor.style_id))
	}

	// This forces the page to change via increaseCapacity.
	new_node := sc2IncreaseCapacity(t, s, s.cursor.page_pin.node, PageListIncreaseCapacityGraphemeBytes)

	// Cursor's page_pin should now point to the new node
	expect(t, s.cursor.page_pin.node == new_node, "s.cursor.page_pin.node == new_node")

	// Verify cursor's page_cell and page_row are correctly reloaded from the pin
	page_rac := s.cursor.page_pin.RowAndCell()
	expect(t, s.cursor.page_row == page_rac.row, "s.cursor.page_row == page_rac.row")
	expect(t, s.cursor.page_cell == page_rac.cell, "s.cursor.page_cell == page_rac.cell")

	// Style should be preserved
	expectEqual(t, old_style, s.cursor.style)
	expect(t, s.cursor.style_id != StyleDefaultId, "s.cursor.style_id != style.default_id")

	// After increaseCapacity, the 5 chars are cloned (5 refs) and
	// the cursor's style is re-added (1 ref) = 6 total.
	{
		page := &s.pages.pages.last.data
		ref_count := page.styles.RefCount(s.cursor.style_id)
		expectEqual(t, SizeCellCountInt(6), ref_count)
	}
}

// Zig: test "Screen: increaseCapacity cursor hyperlink ref count preserved"
func TestSc2_Screen_increaseCapacity_cursor_hyperlink_ref_count_preserved(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 5, 0)
	defer s.Deinit()
	sc2StartHyperlink(t, s, "https://example.com/")
	sc2Write(t, s, "1ABCD")

	// We should have one page and it should be our cursor page
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	expect(t, s.pages.pages.first == s.cursor.page_pin.node, "s.pages.pages.first == s.cursor.page_pin.node")

	{
		page := &s.pages.pages.last.data
		// Cursor has the hyperlink active = 1 count in hyperlink_set
		expectEqual(t, uint(1), page.hyperlink_set.Count())
		expect(t, s.cursor.hyperlink_id != 0, "s.cursor.hyperlink_id != 0")
		expect(t, s.cursor.hyperlink != nil, "s.cursor.hyperlink != null")
	}

	// This forces the page to change via increaseCapacity.
	sc2IncreaseCapacity(t, s, s.cursor.page_pin.node, PageListIncreaseCapacityGraphemeBytes)

	// Hyperlink should be preserved with correct URI
	expect(t, s.cursor.hyperlink != nil, "s.cursor.hyperlink != null")
	expect(t, s.cursor.hyperlink_id != 0, "s.cursor.hyperlink_id != 0")
	expectEqualStrings(t, "https://example.com/", string(s.cursor.hyperlink.uri))

	// After increaseCapacity, the hyperlink is re-added to the new page.
	{
		page := &s.pages.pages.last.data
		expectEqual(t, uint(1), page.hyperlink_set.Count())
	}
}

// Zig: test "Screen: increaseCapacity cursor with both style and hyperlink preserved"
func TestSc2_Screen_increaseCapacity_cursor_with_both_style_and_hyperlink_preserved(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 5, 0)
	defer s.Deinit()

	// Set both a non-default style AND an active hyperlink.
	// Write one character first with bold to mark the row as styled,
	// then start the hyperlink and write more characters.
	zigTry(t, s.SetAttribute(sc2Bold()))
	sc2StartHyperlink(t, s, "https://example.com/")
	sc2Write(t, s, "1ABCD")

	// We should have one page and it should be our cursor page
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	expect(t, s.pages.pages.first == s.cursor.page_pin.node, "s.pages.pages.first == s.cursor.page_pin.node")

	old_style := s.cursor.style

	{
		page := &s.pages.pages.last.data
		// 5 chars + cursor = 6 refs for bold style
		expectEqual(t, SizeCellCountInt(6), page.styles.RefCount(s.cursor.style_id))
		// Cursor has the hyperlink active = 1 count in hyperlink_set
		expectEqual(t, uint(1), page.hyperlink_set.Count())
		expect(t, s.cursor.style_id != StyleDefaultId, "s.cursor.style_id != style.default_id")
		expect(t, s.cursor.hyperlink_id != 0, "s.cursor.hyperlink_id != 0")
		expect(t, s.cursor.hyperlink != nil, "s.cursor.hyperlink != null")
	}

	// This forces the page to change via increaseCapacity.
	sc2IncreaseCapacity(t, s, s.cursor.page_pin.node, PageListIncreaseCapacityGraphemeBytes)

	// Style should be preserved
	expectEqual(t, old_style, s.cursor.style)
	expect(t, s.cursor.style_id != StyleDefaultId, "s.cursor.style_id != style.default_id")

	// Hyperlink should be preserved with correct URI
	expect(t, s.cursor.hyperlink != nil, "s.cursor.hyperlink != null")
	expect(t, s.cursor.hyperlink_id != 0, "s.cursor.hyperlink_id != 0")
	expectEqualStrings(t, "https://example.com/", string(s.cursor.hyperlink.uri))

	// After increaseCapacity, both style and hyperlink are re-added to the new page.
	{
		page := &s.pages.pages.last.data
		ref_count := page.styles.RefCount(s.cursor.style_id)
		expectEqual(t, SizeCellCountInt(6), ref_count)
		expectEqual(t, uint(1), page.hyperlink_set.Count())
	}
}

// Test that calling increaseCapacity on a page that is NOT the cursor's
// page properly delegates to pages.increaseCapacity without doing the
// extra cursor accounting (style/hyperlink re-adding).
//
// Zig: test "Screen: increaseCapacity non-cursor page returns early"
func TestSc2_Screen_increaseCapacity_non_cursor_page_returns_early(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 80, 24, 10000)
	defer s.Deinit()

	// Set up a custom style and hyperlink on the cursor
	zigTry(t, s.SetAttribute(sc2Bold()))
	sc2StartHyperlink(t, s, "https://example.com/")
	sc2Write(t, s, "Hello")

	// Store cursor state before growing pages
	old_style := s.cursor.style
	old_style_id := s.cursor.style_id
	old_hyperlink := s.cursor.hyperlink
	old_hyperlink_id := s.cursor.hyperlink_id

	// The cursor is on the first (and only) page
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	expect(t, s.cursor.page_pin.node == s.pages.pages.first, "s.cursor.page_pin.node == s.pages.pages.first.?")

	// Grow pages until we have multiple pages. The cursor's pin stays on
	// the first page since we're just adding rows.
	first_page_node := s.pages.pages.first
	first_page_node.data.PauseIntegrityChecks(true)
	// Zig: for (0..n) evaluates n once.
	n := int(first_page_node.data.capacity.rows) - int(first_page_node.data.size.rows)
	for i := 0; i < n; i++ {
		_, err := s.pages.Grow()
		zigTry(t, err)
	}
	first_page_node.data.PauseIntegrityChecks(false)
	_, err := s.pages.Grow()
	zigTry(t, err)

	// Now we have two pages
	expect(t, s.pages.pages.first != s.pages.pages.last, "s.pages.pages.first != s.pages.pages.last")
	second_page := s.pages.pages.last

	// Cursor should still be on the first page (where it was created)
	expect(t, s.cursor.page_pin.node == s.pages.pages.first, "s.cursor.page_pin.node == s.pages.pages.first.?")
	expect(t, s.cursor.page_pin.node != second_page, "s.cursor.page_pin.node != second_page")

	second_page_styles_cap := second_page.data.capacity.styles
	cursor_page_styles_cap := s.cursor.page_pin.node.data.capacity.styles

	// Call increaseCapacity on the second page (NOT the cursor's page)
	new_second_page := sc2IncreaseCapacity(t, s, second_page, PageListIncreaseCapacityStyles)

	// The second page should have increased capacity
	expectEqual(t, second_page_styles_cap*2, new_second_page.data.capacity.styles)

	// The cursor's page (first page) should be unchanged
	expectEqual(t, cursor_page_styles_cap, s.cursor.page_pin.node.data.capacity.styles)

	// Cursor state should be completely unchanged since we didn't touch its page
	expectEqual(t, old_style, s.cursor.style)
	expectEqual(t, old_style_id, s.cursor.style_id)
	expectEqual(t, old_hyperlink, s.cursor.hyperlink)
	expectEqual(t, old_hyperlink_id, s.cursor.hyperlink_id)

	// Verify hyperlink is still valid
	expect(t, s.cursor.hyperlink != nil, "s.cursor.hyperlink != null")
	expectEqualStrings(t, "https://example.com/", string(s.cursor.hyperlink.uri))
}

// Regression test for https://github.com/ghostty-org/ghostty/issues/10282
//
// This test exposes a use-after-realloc bug in cursorDown (and similar
// cursor movement functions). The bug pattern:
//
//  1. cursorDown creates a by-value copy of the pin via page_pin.down(n)
//  2. cursorChangePin is called, which may trigger adjustCapacity
//     if the target page's style map is full
//  3. adjustCapacity frees the old page and creates a new one
//  4. The local pin copy still points to the freed page
//  5. rowAndCell() on the stale pin accesses freed memory
//
// Zig: test "Screen: cursorDown to page with insufficient capacity"
func TestSc2_Screen_cursorDown_to_page_with_insufficient_capacity(t *testing.T) {
	testStdCapacity(t)

	// Small screen to make page boundary crossing easy to set up
	s := sc2Init(t, 10, 3, 1)
	defer s.Deinit()

	// Scroll down enough to create a second page
	start_page := &s.pages.pages.last.data
	rem := start_page.capacity.rows
	start_page.PauseIntegrityChecks(true)
	for i := SizeCellCountInt(0); i < rem; i++ {
		zigTry(t, s.cursorDownOrScroll())
	}
	start_page.PauseIntegrityChecks(false)

	// Cursor should now be on a new page
	new_page := &s.cursor.page_pin.node.data
	expect(t, start_page != new_page, "start_page != new_page")

	// Fill new_page's style map to capacity. When we move INTO this page
	// with a style set, adjustCapacity will be triggered.
	func() {
		new_page.PauseIntegrityChecks(true)
		defer new_page.PauseIntegrityChecks(false)
		defer new_page.AssertIntegrity()

		for n := uint32(1); ; n++ {
			_, err := new_page.styles.Add(sc2RgbBg(n))
			if err != nil {
				break
			}
		}
	}()

	// Move cursor to start of active area and set a style
	s.CursorAbsolute(0, 0)
	zigTry(t, s.SetAttribute(sc2Bold()))
	expect(t, s.cursor.style.flags.bold, "s.cursor.style.flags.bold")
	expect(t, s.cursor.style_id != StyleDefaultId, "s.cursor.style_id != style.default_id")

	// Find the row just before the page boundary
	found := false
	for row := uint(0); row < uint(s.pages.rows)-1; row++ {
		s.CursorAbsolute(0, SizeCellCountInt(row))
		cur_node := s.cursor.page_pin.node
		if next := s.cursor.page_pin.Down(1); next.Valid {
			next_pin := next.Value
			if next_pin.node != cur_node {
				// Cursor is at 'row', moving down crosses to new_page
				expect(t, &next_pin.node.data == new_page, "&next_pin.node.data == new_page")

				// This cursorDown triggers the bug: the local page_pin copy
				// becomes stale after adjustCapacity, causing rowAndCell()
				// to access freed memory.
				s.CursorDown(1)

				// If the fix is applied, verify correct state
				expect(t, uint(s.cursor.y) == row+1, "s.cursor.y == row + 1")
				expect(t, s.cursor.style.flags.bold, "s.cursor.style.flags.bold")

				found = true
				break
			}
		}
	}
	if !found {
		// Didn't find boundary
		expect(t, false, "false")
	}
}

// Tests that setAttribute succeeds when the style map is full by
// increasing page capacity. When capacity is at max and increaseCapacity
// returns OutOfSpace, manualStyleUpdate will split the page instead.
//
// Zig: test "Screen setAttribute increases capacity when style map is full"
func TestSc2_Screen_setAttribute_increases_capacity_when_style_map_is_full(t *testing.T) {
	testStdCapacity(t)

	// Use a small screen with multiple rows
	s := sc2Init(t, 10, 5, 10)
	defer s.Deinit()

	// Write content to multiple rows
	sc2Write(t, s, "line1\nline2\nline3\nline4\nline5")

	// Get the page and fill its style map to capacity
	page := &s.cursor.page_pin.node.data
	original_styles_capacity := page.capacity.styles

	// Fill the style map to capacity using the StyleSet's layout capacity
	// which accounts for the load factor
	func() {
		page.PauseIntegrityChecks(true)
		defer page.PauseIntegrityChecks(false)
		defer page.AssertIntegrity()

		max_items := page.styles.layout.cap
		for n := uint(1); n < max_items; n++ {
			_, err := page.styles.Add(sc2RgbBg(uint32(n)))
			if err != nil {
				break
			}
		}
	}()

	// Now try to set a new unique attribute that would require a new style slot
	// This should succeed by increasing capacity (or splitting if at max capacity)
	zigTry(t, s.SetAttribute(sc2Bold()))

	// The style should have been applied (bold flag set)
	expect(t, s.cursor.style.flags.bold, "s.cursor.style.flags.bold")

	// The cursor should have a valid non-default style_id
	expect(t, s.cursor.style_id != StyleDefaultId, "s.cursor.style_id != style.default_id")

	// Either the capacity increased or the page was split/changed
	current_page := &s.cursor.page_pin.node.data
	capacity_increased := current_page.capacity.styles > original_styles_capacity
	page_changed := current_page != page
	expect(t, capacity_increased || page_changed, "capacity_increased or page_changed")
}

// Zig: test "Screen setAttribute splits page on OutOfSpace at max styles"
func TestSc2_Screen_setAttribute_splits_page_on_OutOfSpace_at_max_styles(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 10, 0)
	defer s.Deinit()

	// Write content to multiple rows so we have something to split
	sc2Write(t, s, "line1\nline2\nline3\nline4\nline5")

	// Remember the original node
	original_node := s.cursor.page_pin.node

	// Increase the page's style capacity to max by repeatedly calling increaseCapacity
	// Use Screen.increaseCapacity to properly maintain cursor state
	max_styles := SizeCellCountInt(math.MaxUint16)
	for s.cursor.page_pin.node.data.capacity.styles < max_styles {
		_, err := s.IncreaseCapacity(s.cursor.page_pin.node, Some(PageListIncreaseCapacityStyles))
		if err != nil {
			break
		}
	}

	// Get the page reference after increaseCapacity - cursor may have moved
	page := &s.cursor.page_pin.node.data
	expectEqual(t, max_styles, page.capacity.styles)

	// Fill the style map to capacity using the StyleSet's layout capacity
	// which accounts for the load factor
	func() {
		page.PauseIntegrityChecks(true)
		defer page.PauseIntegrityChecks(false)
		defer page.AssertIntegrity()

		max_items := page.styles.layout.cap
		for n := uint(1); n < max_items; n++ {
			_, err := page.styles.Add(sc2RgbBg(uint32(n)))
			if err != nil {
				break
			}
		}
	}()

	// Track the node before setAttribute
	node_before_set := s.cursor.page_pin.node

	// Now try to set a new unique attribute that would require a new style slot
	// At max capacity, increaseCapacity will return OutOfSpace, triggering page split
	zigTry(t, s.SetAttribute(sc2Bold()))

	// The style should have been applied (bold flag set)
	expect(t, s.cursor.style.flags.bold, "s.cursor.style.flags.bold")

	// The cursor should have a valid non-default style_id
	expect(t, s.cursor.style_id != StyleDefaultId, "s.cursor.style_id != style.default_id")

	// The page should have been split
	page_was_split := s.cursor.page_pin.node != node_before_set ||
		node_before_set.next != nil ||
		node_before_set.prev != nil ||
		s.cursor.page_pin.node != original_node
	expect(t, page_was_split, "page_was_split")
}

// Not translated: Zig test "selectionString map allocation failure cleanup"
// (selectionString_tw tripwire fault injection); see
// scripts/not-translated.txt.

// Zig: test "Screen: promptClickMove line right basic"
func TestSc2_Screen_promptClickMove_line_right_basic(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 20, 5, 0)
	defer s.Deinit()

	// Enable line click mode
	s.semantic_prompt.click = sc2ClickLine()

	// Write a prompt and input
	s.CursorSetSemanticContent(sc2Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc2Write(t, s, "> ")
	s.CursorSetSemanticContent(sc2Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc2Write(t, s, "hello")

	// Move cursor back to start of input (column 2, the 'h')
	s.CursorAbsolute(2, 0)

	// Click on first 'l' (column 4), should require 2 right movements (h->e->l)
	click_pin := sc2Pin(t, s, sc2Active(4, 0))
	result := s.PromptClickMove(click_pin)

	expectEqual(t, uint(2), result.right)
	expectEqual(t, uint(0), result.left)
}

// Zig: test "Screen: promptClickMove line right cursor not on input"
func TestSc2_Screen_promptClickMove_line_right_cursor_not_on_input(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 20, 5, 0)
	defer s.Deinit()

	// Enable line click mode
	s.semantic_prompt.click = sc2ClickLine()

	// Write a prompt and input
	s.CursorSetSemanticContent(sc2Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc2Write(t, s, "> ")
	s.CursorSetSemanticContent(sc2Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc2Write(t, s, "hello")
	s.CursorSetSemanticContent(sc2Output())

	// Move cursor back to prompt area (column 0, the '>')
	s.CursorAbsolute(0, 0)

	// Cursor is on prompt, not input - should return zero
	click_pin := sc2Pin(t, s, sc2Active(4, 0))
	result := s.PromptClickMove(click_pin)

	expectEqual(t, ScreenPromptClickMoveZero, result)
}

// Zig: test "Screen: promptClickMove line right click on same position"
func TestSc2_Screen_promptClickMove_line_right_click_on_same_position(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 20, 5, 0)
	defer s.Deinit()

	// Enable line click mode
	s.semantic_prompt.click = sc2ClickLine()

	// Write a prompt and input
	s.CursorSetSemanticContent(sc2Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc2Write(t, s, "> ")
	s.CursorSetSemanticContent(sc2Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc2Write(t, s, "hello")

	// Move cursor to column 4
	s.CursorAbsolute(4, 0)

	// Click on same position - no movement needed
	click_pin := sc2Pin(t, s, sc2Active(4, 0))
	result := s.PromptClickMove(click_pin)

	expectEqual(t, ScreenPromptClickMoveZero, result)
}

// Zig: test "Screen: promptClickMove line right skips non-input cells"
func TestSc2_Screen_promptClickMove_line_right_skips_non_input_cells(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 20, 5, 0)
	defer s.Deinit()

	// Enable line click mode
	s.semantic_prompt.click = sc2ClickLine()

	// Write: "> h" then output "X" then input "llo"
	s.CursorSetSemanticContent(sc2Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc2Write(t, s, "> ")
	s.CursorSetSemanticContent(sc2Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc2Write(t, s, "h")
	s.CursorSetSemanticContent(sc2Output())
	sc2Write(t, s, "X")
	s.CursorSetSemanticContent(sc2Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc2Write(t, s, "llo")

	// Move cursor to column 2 (the 'h')
	s.CursorAbsolute(2, 0)

	// Click on 'l' at column 5 - should skip the 'X' output cell
	// Movement: h (start) -> l (col 4) -> l (col 5) = 2 right movements
	click_pin := sc2Pin(t, s, sc2Active(5, 0))
	result := s.PromptClickMove(click_pin)

	expectEqual(t, uint(2), result.right)
	expectEqual(t, uint(0), result.left)
}

// Zig: test "Screen: promptClickMove line right soft-wrapped line"
func TestSc2_Screen_promptClickMove_line_right_soft_wrapped_line(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 5, 0)
	defer s.Deinit()

	// Enable line click mode
	s.semantic_prompt.click = sc2ClickLine()

	// Write a prompt and input that wraps
	s.CursorSetSemanticContent(sc2Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc2Write(t, s, "> ")
	s.CursorSetSemanticContent(sc2Input(ScreenCursorSetSemanticContentInputClearExplicit))
	// Write 8 chars of input, first row has 2 for prompt + 8 input = 10 cols
	sc2Write(t, s, "abcdefgh")
	// Continue on next row (soft-wrapped)
	sc2Write(t, s, "ij")

	// Verify soft wrap occurred
	{
		contents := sc2DumpString(t, s, Point{Tag: PointTagViewport})
		expectEqualStrings(t, "> abcdefgh\nij", contents)
	}

	// Move cursor to column 2 (the 'a')
	s.CursorAbsolute(2, 0)

	// Click on 'j' at column 1, row 1 - should count all input cells
	// Movement: a->b->c->d->e->f->g->h->i->j = 9 right movements
	click_pin := sc2Pin(t, s, sc2Active(1, 1))
	result := s.PromptClickMove(click_pin)

	expectEqual(t, uint(9), result.right)
	expectEqual(t, uint(0), result.left)
}

// Zig: test "Screen: promptClickMove disabled when click is none"
func TestSc2_Screen_promptClickMove_disabled_when_click_is_none(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 20, 5, 0)
	defer s.Deinit()

	// Click mode is .none by default (disabled)
	expectEqual(t, ScreenSemanticPromptSemanticClick{Tag: ScreenSemanticPromptSemanticClickNone}, s.semantic_prompt.click)

	// Write a prompt and input
	s.CursorSetSemanticContent(sc2Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc2Write(t, s, "> ")
	s.CursorSetSemanticContent(sc2Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc2Write(t, s, "hello")

	// Move cursor to start of input
	s.CursorAbsolute(2, 0)

	// Click should return zero since click mode is disabled
	click_pin := sc2Pin(t, s, sc2Active(4, 0))
	result := s.PromptClickMove(click_pin)

	expectEqual(t, ScreenPromptClickMoveZero, result)
}

// Zig: test "Screen: promptClickMove line right stops at hard wrap"
func TestSc2_Screen_promptClickMove_line_right_stops_at_hard_wrap(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 20, 5, 0)
	defer s.Deinit()

	// Enable line click mode
	s.semantic_prompt.click = sc2ClickLine()

	// Write prompt and input on first line
	s.CursorSetSemanticContent(sc2Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc2Write(t, s, "> ")
	s.CursorSetSemanticContent(sc2Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc2Write(t, s, "hello")
	// Hard wrap (newline)
	sc2Write(t, s, "\n")
	sc2Write(t, s, "world")

	// Move cursor to column 2 (the 'h')
	s.CursorAbsolute(2, 0)

	// Click on 'w' at column 0, row 1 - but line mode stops at hard wrap
	// Should only move to end of first line: h->e->l->l->o = 4 movements
	click_pin := sc2Pin(t, s, sc2Active(0, 1))
	result := s.PromptClickMove(click_pin)

	// Should stop at end of first line, not cross hard wrap
	expectEqual(t, uint(5), result.right)
	expectEqual(t, uint(0), result.left)
}

// Zig: test "Screen: promptClickMove line right stops at non-continuation row"
func TestSc2_Screen_promptClickMove_line_right_stops_at_non_continuation_row(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 20, 5, 0)
	defer s.Deinit()

	// Enable line click mode
	s.semantic_prompt.click = sc2ClickLine()

	// Row 0: PROMPT "> hello"
	s.CursorSetSemanticContent(sc2Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc2Write(t, s, "> ")
	s.CursorSetSemanticContent(sc2Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc2Write(t, s, "hello\n")

	// Row 1: CONTINUATION "world"
	s.CursorSetSemanticContent(sc2Prompt(OSCParsersSemanticPromptPromptKindContinuation))
	s.CursorSetSemanticContent(sc2Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc2Write(t, s, "world\n")

	// Row 2: NEW PROMPT "> again"
	s.CursorSetSemanticContent(sc2Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc2Write(t, s, "> ")
	s.CursorSetSemanticContent(sc2Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc2Write(t, s, "again")

	// Verify content
	{
		contents := sc2DumpString(t, s, Point{Tag: PointTagViewport})
		expectEqualStrings(t, "> hello\nworld\n> again", contents)
	}

	// Move cursor to 'w' at column 0, row 1
	s.CursorAbsolute(0, 1)

	// Click on 'a' at column 2, row 2 - but row 2 is a new prompt
	// Should stop at end of "world": w->o->r->l->d = 4 movements
	click_pin := sc2Pin(t, s, sc2Active(2, 2))
	result := s.PromptClickMove(click_pin)

	// Should stop at 'd' (end of world), not cross to new prompt
	expectEqual(t, uint(5), result.right)
	expectEqual(t, uint(0), result.left)
}

// Zig: test "Screen: promptClickMove line left basic"
func TestSc2_Screen_promptClickMove_line_left_basic(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 20, 5, 0)
	defer s.Deinit()

	// Enable line click mode
	s.semantic_prompt.click = sc2ClickLine()

	// Write a prompt and input
	s.CursorSetSemanticContent(sc2Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc2Write(t, s, "> ")
	s.CursorSetSemanticContent(sc2Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc2Write(t, s, "hello")

	// Cursor is at column 7 (after 'o'), move it to column 6 (the 'o')
	s.CursorAbsolute(6, 0)

	// Click on 'h' (column 2), should require 4 left movements (o->l->l->e->h)
	click_pin := sc2Pin(t, s, sc2Active(2, 0))
	result := s.PromptClickMove(click_pin)

	expectEqual(t, uint(4), result.left)
	expectEqual(t, uint(0), result.right)
}

// Zig: test "Screen: promptClickMove line left skips non-input cells"
func TestSc2_Screen_promptClickMove_line_left_skips_non_input_cells(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 20, 5, 0)
	defer s.Deinit()

	// Enable line click mode
	s.semantic_prompt.click = sc2ClickLine()

	// Write: "> h" then output "X" then input "llo"
	s.CursorSetSemanticContent(sc2Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc2Write(t, s, "> ")
	s.CursorSetSemanticContent(sc2Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc2Write(t, s, "h")
	s.CursorSetSemanticContent(sc2Output())
	sc2Write(t, s, "X")
	s.CursorSetSemanticContent(sc2Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc2Write(t, s, "llo")

	// Move cursor to column 6 (the 'o')
	s.CursorAbsolute(6, 0)

	// Click on 'h' at column 2 - should skip the 'X' output cell
	// Movement: o->l->l->h = 3 left movements (skipping X)
	click_pin := sc2Pin(t, s, sc2Active(2, 0))
	result := s.PromptClickMove(click_pin)

	expectEqual(t, uint(3), result.left)
	expectEqual(t, uint(0), result.right)
}

// Zig: test "Screen: promptClickMove line left soft-wrapped line"
func TestSc2_Screen_promptClickMove_line_left_soft_wrapped_line(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 5, 0)
	defer s.Deinit()

	// Enable line click mode
	s.semantic_prompt.click = sc2ClickLine()

	// Write a prompt and input that wraps
	s.CursorSetSemanticContent(sc2Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc2Write(t, s, "> ")
	s.CursorSetSemanticContent(sc2Input(ScreenCursorSetSemanticContentInputClearExplicit))
	// Write 8 chars of input, first row has 2 for prompt + 8 input = 10 cols
	sc2Write(t, s, "abcdefgh")
	// Continue on next row (soft-wrapped)
	sc2Write(t, s, "ij")

	// Verify soft wrap occurred
	{
		contents := sc2DumpString(t, s, Point{Tag: PointTagViewport})
		expectEqualStrings(t, "> abcdefgh\nij", contents)
	}

	// Cursor is at column 2, row 1 (after 'j'). Move to 'j' at column 1.
	s.CursorAbsolute(1, 1)

	// Click on 'a' at column 2, row 0 - should count all input cells backwards
	// Movement: j->i->h->g->f->e->d->c->b->a = 9 left movements
	click_pin := sc2Pin(t, s, sc2Active(2, 0))
	result := s.PromptClickMove(click_pin)

	expectEqual(t, uint(9), result.left)
	expectEqual(t, uint(0), result.right)
}

// Zig: test "Screen: promptClickMove line left stops at hard wrap"
func TestSc2_Screen_promptClickMove_line_left_stops_at_hard_wrap(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 20, 5, 0)
	defer s.Deinit()

	// Enable line click mode
	s.semantic_prompt.click = sc2ClickLine()

	// Write prompt and input on first line, then hard wrap
	s.CursorSetSemanticContent(sc2Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc2Write(t, s, "> ")
	s.CursorSetSemanticContent(sc2Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc2Write(t, s, "hello")
	// Hard wrap (newline)
	sc2Write(t, s, "\n")
	sc2Write(t, s, "world")

	// Move cursor to 'd' at column 4, row 1 (an actual input cell)
	s.CursorAbsolute(4, 1)

	// Click on 'h' at column 2, row 0 - but line mode stops at hard wrap
	// Should only move to start of second line, not cross to row 0
	click_pin := sc2Pin(t, s, sc2Active(2, 0))
	result := s.PromptClickMove(click_pin)

	// Should stop at start of second line: d->l->r->o->w = 4 movements
	expectEqual(t, uint(4), result.left)
	expectEqual(t, uint(0), result.right)
}

// Zig: test "Screen: promptClickMove click right of input same line"
func TestSc2_Screen_promptClickMove_click_right_of_input_same_line(t *testing.T) {
	testStdCapacity(t)

	// Set up: "> hello" where "> " is prompt and "hello" is input
	// Clicking to the right of the 'o' should move cursor past the input

	s := sc2Init(t, 20, 5, 0)
	defer s.Deinit()

	// Enable line click mode
	s.semantic_prompt.click = sc2ClickLine()

	// Write a prompt and input
	s.CursorSetSemanticContent(sc2Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc2Write(t, s, "> ")
	s.CursorSetSemanticContent(sc2Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc2Write(t, s, "hello")

	// Move cursor to start of input (column 2, the 'h')
	s.CursorAbsolute(2, 0)

	// Click beyond the input (column 15) - should move to one past the 'o'
	click_pin := sc2Pin(t, s, sc2Active(15, 0))
	result := s.PromptClickMove(click_pin)

	expectEqual(t, uint(5), result.right)
	expectEqual(t, uint(0), result.left)
}

// Zig: test "Screen: promptClickMove click right of input cursor at end"
func TestSc2_Screen_promptClickMove_click_right_of_input_cursor_at_end(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 20, 5, 0)
	defer s.Deinit()

	// Enable line click mode
	s.semantic_prompt.click = sc2ClickLine()

	// Write a prompt and input
	s.CursorSetSemanticContent(sc2Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc2Write(t, s, "> ")
	s.CursorSetSemanticContent(sc2Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc2Write(t, s, "hello")

	// Cursor is already at column 7 (one past 'o') after writing
	// Click beyond the input (column 15) - no movement needed since
	// cursor is already at the end position
	click_pin := sc2Pin(t, s, sc2Active(15, 0))
	result := s.PromptClickMove(click_pin)

	expectEqual(t, uint(0), result.right)
	expectEqual(t, uint(0), result.left)
}

// Zig: test "Screen: promptClickMove click right of input on lower line"
func TestSc2_Screen_promptClickMove_click_right_of_input_on_lower_line(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 20, 5, 0)
	defer s.Deinit()

	// Enable line click mode
	s.semantic_prompt.click = sc2ClickLine()

	// Write a prompt and input
	s.CursorSetSemanticContent(sc2Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc2Write(t, s, "> ")
	s.CursorSetSemanticContent(sc2Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc2Write(t, s, "hello")

	// Move cursor to start of input (column 2, the 'h')
	s.CursorAbsolute(2, 0)

	// Click on a lower line (row 1) - should move to end of input
	// This is outside the prompt area so should clamp to end
	click_pin := sc2Pin(t, s, sc2Active(5, 1))
	result := s.PromptClickMove(click_pin)

	// From 'h', we need to pass e, l, l, o (4 cells) + 1 past end = 5
	expectEqual(t, uint(5), result.right)
	expectEqual(t, uint(0), result.left)
}

// Zig: test "Screen: promptClickMove click right of input cursor at end lower line"
func TestSc2_Screen_promptClickMove_click_right_of_input_cursor_at_end_lower_line(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 20, 5, 0)
	defer s.Deinit()

	// Enable line click mode
	s.semantic_prompt.click = sc2ClickLine()

	// Write a prompt and input
	s.CursorSetSemanticContent(sc2Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc2Write(t, s, "> ")
	s.CursorSetSemanticContent(sc2Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc2Write(t, s, "hello")

	// Cursor is at column 7 after writing (one past 'o')
	// Click on a lower line (row 1) - cursor already at end, no movement needed
	click_pin := sc2Pin(t, s, sc2Active(5, 1))
	result := s.PromptClickMove(click_pin)

	expectEqual(t, uint(0), result.right)
	expectEqual(t, uint(0), result.left)
}

// Zig: test "Screen: promptClickMove click right of input cursor on last char"
func TestSc2_Screen_promptClickMove_click_right_of_input_cursor_on_last_char(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 20, 5, 0)
	defer s.Deinit()

	// Enable line click mode
	s.semantic_prompt.click = sc2ClickLine()

	// Write a prompt and input
	s.CursorSetSemanticContent(sc2Prompt(OSCParsersSemanticPromptPromptKindInitial))
	sc2Write(t, s, "> ")
	s.CursorSetSemanticContent(sc2Input(ScreenCursorSetSemanticContentInputClearExplicit))
	sc2Write(t, s, "hello")

	// Move cursor to last input char (column 6, the 'o')
	s.CursorAbsolute(6, 0)

	// Click beyond the input (column 15)
	click_pin := sc2Pin(t, s, sc2Active(15, 0))
	result := s.PromptClickMove(click_pin)

	expectEqual(t, uint(1), result.right)
	expectEqual(t, uint(0), result.left)
}

// ---------------------------------------------------------------------------
// term51 additional tests (O). Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

// sc2History is the Zig literal `.{ .history = .{ .x = x, .y = y } }`.
func sc2History(x int, y int) Point {
	return Point{Tag: PointTagHistory, History: PointCoordinate{x: SizeCellCountInt(x), y: uint32(y)}}
}

// sc2OPin is term51OPin: an (untracked) pin at the active point (x, y).
func sc2OPin(t *testing.T, s *Screen, x int, y int) PageListPin {
	t.Helper()
	return sc2Pin(t, s, sc2Active(x, y))
}

// sc2OCell is term51OCell: the cell at the active point (x, y).
func sc2OCell(t *testing.T, s *Screen, x int, y int) *PageCell {
	t.Helper()
	return sc2OPin(t, s, x, y).RowAndCell().cell
}

// sc2ORow is term51ORow: the row at the active row y.
func sc2ORow(t *testing.T, s *Screen, y int) *PageRow {
	t.Helper()
	return sc2OPin(t, s, 0, y).RowAndCell().row
}

// sc2OSetCp is term51OSetCp: write a plain codepoint straight into the cell
// at active (x, y).
func sc2OSetCp(t *testing.T, s *Screen, x int, y int, cp uint32) {
	t.Helper()
	*sc2OCell(t, s, x, y) = PageCell{
		content_tag: PageCellContentTagCodepoint,
		content:     PageCellContent{codepoint: cp},
	}
}

// sc2OCp is term51OCp: the codepoint of the cell at active (x, y).
func sc2OCp(t *testing.T, s *Screen, x int, y int) uint32 {
	t.Helper()
	return sc2OCell(t, s, x, y).Codepoint()
}

// sc2OStyleRefs is term51OStyleRefs: the reference count of a style in a page.
func sc2OStyleRefs(page *Page, id StyleId) uint {
	return uint(page.styles.RefCount(id))
}

// sc2OLinkRefs is term51OLinkRefs: the reference count of a hyperlink in a
// page.
func sc2OLinkRefs(page *Page, id HyperlinkId) uint {
	return uint(page.hyperlink_set.RefCount(id))
}

// sc2ONodeCount is term51ONodeCount: the number of pages in the screen.
func sc2ONodeCount(s *Screen) uint {
	var n uint
	for it := s.pages.pages.first; it != nil; it = it.next {
		n++
	}
	return n
}

// sc2ODirtyRows is term51ODirtyRows: the row-level dirty flag of the first n
// active rows.
func sc2ODirtyRows(t *testing.T, s *Screen, n int) []bool {
	t.Helper()
	result := make([]bool, n)
	for y := 0; y < n; y++ {
		result[y] = sc2ORow(t, s, y).dirty
	}
	return result
}

// sc2OExpectPin is term51OExpectPin: expect that the pin sits at active (x, y).
func sc2OExpectPin(t *testing.T, s *Screen, pin PageListPin, x int, y int) {
	t.Helper()
	pt := sc2PointFromPin(t, s, PointTagActive, pin)
	expectEqual(t, uint(x), uint(pt.Active.x))
	expectEqual(t, uint(y), uint(pt.Active.y))
}

// sc2OExpectCursorConsistent is term51OExpectCursorConsistent: expect that the
// cursor cache (x, y, row, cell) agrees with the pin.
func sc2OExpectCursorConsistent(t *testing.T, s *Screen) {
	t.Helper()
	rac := s.cursor.page_pin.RowAndCell()
	expect(t, s.cursor.page_row == rac.row, "s.cursor.page_row == rac.row")
	expect(t, s.cursor.page_cell == rac.cell, "s.cursor.page_cell == rac.cell")
	sc2OExpectPin(t, s, *s.cursor.page_pin, int(s.cursor.x), int(s.cursor.y))
	expectEqual(t, s.cursor.x, s.cursor.page_pin.x)
}

// sc2OTwoPages is term51OTwoPages: a screen whose cursor is on the bottom row
// of the active area, which is the only row used on the second page; every
// other active row is one of the last rows of the first page. The first page
// is full.
func sc2OTwoPages(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) *Screen {
	t.Helper()
	s := sc2Init(t, cols, rows, 10)
	// Zig: errdefer s.deinit() (a failed helper ends the test).
	first := &s.pages.pages.first.data
	first.PauseIntegrityChecks(true)
	for i := SizeCellCountInt(0); i < first.capacity.rows; i++ {
		zigTry(t, s.cursorDownOrScroll())
	}
	first.PauseIntegrityChecks(false)
	expectEqual(t, uint(2), sc2ONodeCount(s))
	expect(t, s.cursor.page_pin.node == s.pages.pages.last, "s.cursor.page_pin.node == s.pages.pages.last.?")
	expectEqual(t, SizeCellCountInt(1), s.pages.pages.last.data.size.rows)
	expectEqual(t, first.capacity.rows, first.size.rows)
	expectEqual(t, rows-1, s.cursor.y)
	return s
}

// sc2OMultiPage is term51OMultiPage: a screen with a very wide active area so
// that the active area itself spans at least three pages (no scrolling
// needed).
func sc2OMultiPage(t *testing.T, max_scrollback uint64) *Screen {
	t.Helper()
	s := sc2Init(t, 2000, 60, max_scrollback)
	// Zig: errdefer s.deinit() (a failed helper ends the test).
	expect(t, sc2ONodeCount(s) >= 3, "term51ONodeCount(&s) >= 3")
	return s
}

// Zig: test "term51: Cursor.deinit: without hyperlink it changes nothing"
func TestSc2_term51_Cursor_deinit_without_hyperlink_it_changes_nothing(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 4, 2, 0)
	defer s.Deinit()

	before := s.cursor
	s.cursor.Deinit()
	expect(t, before == s.cursor, "std.meta.eql(before, s.cursor)")
	expect(t, s.cursor.hyperlink == nil, "s.cursor.hyperlink == null")

	// Safe to call repeatedly
	s.cursor.Deinit()
	expect(t, before == s.cursor, "std.meta.eql(before, s.cursor)")
}

// Zig: test "term51: Cursor.deinit: frees an explicit-id hyperlink and its strings"
func TestSc2_term51_Cursor_deinit_frees_an_explicit_id_hyperlink_and_its_strings(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 4, 2, 0)
	defer s.Deinit()

	// Zig-only: any leak (uri, explicit id or the Hyperlink itself) fails the test.
	c := s.cursor
	src := Hyperlink{
		id:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdExplicit, Explicit: []uint8("my-id")},
		uri: []uint8("https://example.com/"),
	}
	// Zig-only: const link = try alloc.create(hyperlink.Hyperlink);
	link, err := src.Dupe()
	zigTry(t, err)
	c.hyperlink = &link
	c.Deinit()
}

// Zig: test "term51: Cursor.deinit: frees an implicit-id hyperlink"
func TestSc2_term51_Cursor_deinit_frees_an_implicit_id_hyperlink(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 4, 2, 0)
	defer s.Deinit()

	c := s.cursor
	src := Hyperlink{
		id:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: 7},
		uri: []uint8("https://example.com/implicit"),
	}
	// Zig-only: const link = try alloc.create(hyperlink.Hyperlink);
	link, err := src.Dupe()
	zigTry(t, err)
	c.hyperlink = &link
	c.Deinit()
}

// sc2Slots is `std.meta.tags(charsets.Slots)`.
func sc2Slots() []CharsetsSlots {
	return []CharsetsSlots{CharsetsSlotsG0, CharsetsSlotsG1, CharsetsSlotsG2, CharsetsSlotsG3}
}

// sc2Charsets is `std.meta.tags(charsets.Charset)`.
func sc2Charsets() []CharsetsCharset {
	return []CharsetsCharset{
		CharsetsCharsetUtf8,
		CharsetsCharsetAscii,
		CharsetsCharsetBritish,
		CharsetsCharsetDecSpecial,
	}
}

// Zig: test "term51: CharsetState.CharsetArray.get: defaults and every slot"
func TestSc2_term51_CharsetState_CharsetArray_get_defaults_and_every_slot(t *testing.T) {
	testStdCapacity(t)
	var a screenCharsetStateCharsetArray
	for _, slot := range sc2Slots() {
		expectEqual(t, CharsetsCharsetUtf8, a.Get(slot))
	}

	// Each slot is read from its own field.
	a.g0 = CharsetsCharsetAscii
	a.g1 = CharsetsCharsetBritish
	a.g2 = CharsetsCharsetDecSpecial
	a.g3 = CharsetsCharsetUtf8
	expectEqual(t, CharsetsCharsetAscii, a.Get(CharsetsSlotsG0))
	expectEqual(t, CharsetsCharsetBritish, a.Get(CharsetsSlotsG1))
	expectEqual(t, CharsetsCharsetDecSpecial, a.Get(CharsetsSlotsG2))
	expectEqual(t, CharsetsCharsetUtf8, a.Get(CharsetsSlotsG3))

	a.g3 = CharsetsCharsetDecSpecial
	a.g0 = CharsetsCharsetBritish
	expectEqual(t, CharsetsCharsetBritish, a.Get(CharsetsSlotsG0))
	expectEqual(t, CharsetsCharsetDecSpecial, a.Get(CharsetsSlotsG3))

	// A const pointer is enough
	p := &a
	expectEqual(t, CharsetsCharsetBritish, p.Get(CharsetsSlotsG1))
}

// Zig: test "term51: CharsetState.CharsetArray.set: every slot and charset, others untouched"
func TestSc2_term51_CharsetState_CharsetArray_set_every_slot_and_charset_others_untouched(t *testing.T) {
	testStdCapacity(t)
	for _, slot := range sc2Slots() {
		for _, cs := range sc2Charsets() {
			var a screenCharsetStateCharsetArray
			a.Set(slot, cs)
			for _, other := range sc2Slots() {
				want := CharsetsCharsetUtf8
				if other == slot {
					want = cs
				}
				expectEqual(t, want, a.Get(other))
			}
		}
	}

	// Setting again overwrites only that slot.
	var a screenCharsetStateCharsetArray
	a.Set(CharsetsSlotsG1, CharsetsCharsetAscii)
	a.Set(CharsetsSlotsG2, CharsetsCharsetBritish)
	a.Set(CharsetsSlotsG1, CharsetsCharsetDecSpecial)
	expectEqual(t, CharsetsCharsetUtf8, a.Get(CharsetsSlotsG0))
	expectEqual(t, CharsetsCharsetDecSpecial, a.Get(CharsetsSlotsG1))
	expectEqual(t, CharsetsCharsetBritish, a.Get(CharsetsSlotsG2))
	expectEqual(t, CharsetsCharsetUtf8, a.Get(CharsetsSlotsG3))
	expectEqual(t, CharsetsCharsetDecSpecial, a.g1)
	expectEqual(t, CharsetsCharsetBritish, a.g2)
}

// Zig: test "term51: init: Options.default and every field of a new screen"
func TestSc2_term51_init_Options_default_and_every_field_of_a_new_screen(t *testing.T) {
	testStdCapacity(t)

	expectEqual(t, SizeCellCountInt(80), ScreenOptionsDefault.cols)
	expectEqual(t, SizeCellCountInt(24), ScreenOptionsDefault.rows)
	expectEqual(t, uint64(0), ScreenOptionsDefault.max_scrollback)
	expectEqual(t, uint64(320*1000*1000), ScreenOptionsDefault.kitty_image_storage_limit)

	s := sc2InitWith(t, ScreenOptionsDefault)
	defer s.Deinit()

	// Zig-only: try std.testing.expect(s.alloc.ptr == alloc.ptr);
	expectEqual(t, SizeCellCountInt(80), s.pages.cols)
	expectEqual(t, SizeCellCountInt(24), s.pages.rows)
	expectEqual(t, uint(24), s.pages.total_rows)
	expect(t, s.no_scrollback, "s.no_scrollback")
	expectEqual(t, uint(1), sc2ONodeCount(s))
	// Viewport pin and cursor pin
	expectEqual(t, uint(2), s.pages.CountTrackedPins())

	// Cursor
	expectEqual(t, SizeCellCountInt(0), s.cursor.x)
	expectEqual(t, SizeCellCountInt(0), s.cursor.y)
	expectEqual(t, CursorStyleBlock, s.cursor.cursor_style)
	expect(t, !s.cursor.pending_wrap, "!s.cursor.pending_wrap")
	expect(t, !s.cursor.protected, "!s.cursor.protected")
	expect(t, s.cursor.style.Default(), "s.cursor.style.default()")
	expectEqual(t, StyleDefaultId, s.cursor.style_id)
	expectEqual(t, HyperlinkId(0), s.cursor.hyperlink_id)
	expectEqual(t, SizeOffsetInt(0), s.cursor.hyperlink_implicit_id)
	expect(t, s.cursor.hyperlink == nil, "s.cursor.hyperlink == null")
	expectEqual(t, PageCellSemanticContentOutput, s.cursor.semantic_content)
	expect(t, !s.cursor.semantic_content_clear_eol, "!s.cursor.semantic_content_clear_eol")
	expect(t, s.cursor.page_pin.node == s.pages.pages.first, "s.cursor.page_pin.node == s.pages.pages.first.?")
	expectEqual(t, SizeCellCountInt(0), s.cursor.page_pin.x)
	expectEqual(t, SizeCellCountInt(0), s.cursor.page_pin.y)
	sc2OExpectCursorConsistent(t, s)

	// Everything else
	expect(t, !s.saved_cursor.Valid, "s.saved_cursor == null")
	expect(t, !s.selection.Valid, "s.selection == null")
	expectEqual(t, ANSIProtectedModeOff, s.protected_mode)
	expectEqual(t, CharsetsSlotsG0, s.charset.gl)
	expectEqual(t, CharsetsSlotsG2, s.charset.gr)
	expect(t, !s.charset.single_shift.Valid, "s.charset.single_shift == null")
	for _, slot := range sc2Slots() {
		expectEqual(t, CharsetsCharsetUtf8, s.charset.charsets.Get(slot))
	}
	expectEqual(t, uint8(0), s.kitty_keyboard.idx)
	expectEqual(t, KittyKeyFlagsDisabled, s.kitty_keyboard.Current())
	expect(t, !s.semantic_prompt.seen, "!s.semantic_prompt.seen")
	expect(t, s.semantic_prompt.click.Tag == ScreenSemanticPromptSemanticClickNone, "s.semantic_prompt.click == .none")
	expect(t, !s.dirty.selection, "!s.dirty.selection")
	expect(t, !s.dirty.hyperlink_hover, "!s.dirty.hyperlink_hover")

	// A new screen is blank.
	expectEqual(t, uint32(0), sc2OCp(t, s, 0, 0))
	expectEqual(t, uint32(0), sc2OCp(t, s, 79, 23))
}

// Zig: test "term51: init: max_scrollback sets no_scrollback and the page list size limit"
func TestSc2_term51_init_max_scrollback_sets_no_scrollback_and_the_page_list_size_limit(t *testing.T) {
	testStdCapacity(t)

	// Zero: no scrollback (despite the Options doc comment saying
	// "unlimited"), the limit is the minimum the page list allows.
	{
		s := sc2Init(t, 10, 5, 0)
		defer s.Deinit()
		expect(t, s.no_scrollback, "s.no_scrollback")
		expectEqual(t, uint64(0), s.pages.explicit_max_size)
		expectEqual(t, s.pages.min_max_size, s.pages.MaxSize())
	}

	// One byte is enough to turn scrollback on, clamped to the minimum.
	{
		s := sc2Init(t, 10, 5, 1)
		defer s.Deinit()
		expect(t, !s.no_scrollback, "!s.no_scrollback")
		expectEqual(t, uint64(1), s.pages.explicit_max_size)
		expectEqual(t, s.pages.min_max_size, s.pages.MaxSize())
	}

	// A big limit is kept as is.
	{
		s := sc2Init(t, 10, 5, 100*1024*1024)
		defer s.Deinit()
		expect(t, !s.no_scrollback, "!s.no_scrollback")
		expectEqual(t, uint64(100*1024*1024), s.pages.MaxSize())
	}

	// The kitty storage limit does not influence anything observable when
	// kitty graphics are compiled out.
	{
		s := sc2InitWith(t, ScreenOptions{
			cols:                      10,
			rows:                      5,
			max_scrollback:            0,
			kitty_image_storage_limit: 0,
		})
		defer s.Deinit()
		expect(t, s.no_scrollback, "s.no_scrollback")
	}
}

// Zig: test "term51: init: dimensions from one cell to the maximum"
func TestSc2_term51_init_dimensions_from_one_cell_to_the_maximum(t *testing.T) {
	testStdCapacity(t)
	const max = SizeCellCountInt(math.MaxUint16)

	dims := [][2]SizeCellCountInt{
		{1, 1},
		{1, 2},
		{2, 1},
		{215, 215},
		{216, 216},
		{500, 5},
		{max, 2},
		{1, max},
	}
	for _, d := range dims {
		func() {
			s := sc2Init(t, d[0], d[1], 0)
			defer s.Deinit()
			expectEqual(t, d[0], s.pages.cols)
			expectEqual(t, d[1], s.pages.rows)
			expectEqual(t, uint(d[1]), s.pages.total_rows)

			// The pages hold exactly `rows` rows and every page has the width.
			var total uint
			for it := s.pages.pages.first; it != nil; it = it.next {
				total += uint(it.data.size.rows)
				expectEqual(t, d[0], it.data.size.cols)
			}
			expectEqual(t, uint(d[1]), total)

			expect(t, s.cursor.page_pin.node == s.pages.pages.first, "s.cursor.page_pin.node == s.pages.pages.first.?")
			sc2OExpectCursorConsistent(t, s)
			s.AssertIntegrity()

			// The last cell is addressable.
			s.CursorAbsolute(d[0]-1, d[1]-1)
			expectEqual(t, d[0]-1, s.cursor.x)
			expectEqual(t, d[1]-1, s.cursor.y)
			sc2OExpectCursorConsistent(t, s)
		}()
	}
}

// Not translated: Zig test "term51: init: out of memory at every allocation unwinds without leaking"
// (std.testing.FailingAllocator); see scripts/not-translated.txt.

// Zig: test "term51: deinit: releases hyperlink, selection, styles and every page"
func TestSc2_term51_deinit_releases_hyperlink_selection_styles_and_every_page(t *testing.T) {
	testStdCapacity(t)
	s := sc2OTwoPages(t, 20, 3)
	// Zig-only: any leak makes the test fail since this is std.testing.allocator.
	zigTry(t, s.SetAttribute(sc2Bold()))
	sc2StartHyperlink(t, s, "https://example.com/")
	sc2Write(t, s, "hello")
	zigTry(t, s.Select(Some(SelectionInit(
		sc2OPin(t, s, 0, 0),
		sc2OPin(t, s, 3, 2),
		false,
	))))
	expect(t, s.selection.Valid, "s.selection != null")
	expect(t, s.cursor.hyperlink != nil, "s.cursor.hyperlink != null")
	s.Deinit()
}

// Zig: test "term51: deinit: a fresh and a reset screen"
func TestSc2_term51_deinit_a_fresh_and_a_reset_screen(t *testing.T) {
	testStdCapacity(t)

	s := sc2Init(t, 3, 3, 0)
	s.Deinit()

	s2 := sc2Init(t, 3, 3, 10)
	zigTry(t, s2.StartHyperlink([]uint8("https://example.com/"), Some([]uint8("id"))))
	s2.Reset()
	s2.Deinit()
}

// Zig: test "term51: assertIntegrity: holds for every cursor position"
func TestSc2_term51_assertIntegrity_holds_for_every_cursor_position(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 4, 10)
	defer s.Deinit()

	for y := 0; y < 4; y++ {
		for x := 0; x < 5; x++ {
			s.CursorAbsolute(SizeCellCountInt(x), SizeCellCountInt(y))
			before := s.cursor
			s.AssertIntegrity()
			// It is read-only.
			expect(t, before == s.cursor, "std.meta.eql(before, s.cursor)")
		}
	}
}

// Zig: test "term51: assertIntegrity: holds with scrollback and cursor on a later page"
func TestSc2_term51_assertIntegrity_holds_with_scrollback_and_cursor_on_a_later_page(t *testing.T) {
	testStdCapacity(t)

	s := sc2OTwoPages(t, 20, 3)
	defer s.Deinit()
	s.AssertIntegrity()

	s.CursorAbsolute(19, 0)
	s.AssertIntegrity()
	s.CursorAbsolute(0, 2)
	s.AssertIntegrity()

	s2 := sc2OMultiPage(t, 0)
	defer s2.Deinit()
	s2.AssertIntegrity()
	s2.CursorAbsolute(1999, 59)
	s2.AssertIntegrity()
}

// Zig: test "term51: reset: restores every piece of screen state"
func TestSc2_term51_reset_restores_every_piece_of_screen_state(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 100)
	defer s.Deinit()

	zigTry(t, s.SetAttribute(sc2Bold()))
	sc2StartHyperlink(t, s, "https://example.com/")
	sc2Write(t, s, "ab\ncd\nef\ngh")
	expect(t, s.pages.total_rows > 3, "s.pages.total_rows > 3")
	expectEqual(t, SizeOffsetInt(1), s.cursor.hyperlink_implicit_id)

	s.cursor.cursor_style = ScreenCursorStyleBar
	s.cursor.protected = true
	s.cursor.pending_wrap = true
	s.CursorSetSemanticContent(sc2Prompt(OSCParsersSemanticPromptPromptKindInitial))
	s.cursor.semantic_content_clear_eol = true
	s.saved_cursor = Some(ScreenSavedCursor{
		x:            1,
		y:            1,
		style:        Style{},
		protected:    true,
		pending_wrap: true,
		origin:       true,
		charset:      screenCharsetStateDefault(),
	})
	s.charset.gl = CharsetsSlotsG1
	s.charset.gr = CharsetsSlotsG3
	s.charset.single_shift = Some(CharsetsSlotsG2)
	s.charset.charsets.Set(CharsetsSlotsG0, CharsetsCharsetAscii)
	s.charset.charsets.Set(CharsetsSlotsG3, CharsetsCharsetDecSpecial)
	s.kitty_keyboard.Push(KittyKeyFlags{disambiguate: true})
	s.protected_mode = ANSIProtectedModeDec
	s.semantic_prompt.click = ScreenSemanticPromptSemanticClick{Tag: ScreenSemanticPromptSemanticClickClickEvents}
	zigTry(t, s.Select(Some(SelectionInit(
		sc2OPin(t, s, 0, 0),
		sc2OPin(t, s, 3, 1),
		false,
	))))
	s.dirty = ScreenDirty{selection: false, hyperlink_hover: true}
	// Viewport, cursor, selection start and end
	expectEqual(t, uint(4), s.pages.CountTrackedPins())

	pin_ptr := s.cursor.page_pin
	s.Reset()

	// The tracked cursor pin is kept and moved to the top left.
	expect(t, s.cursor.page_pin == pin_ptr, "s.cursor.page_pin == pin_ptr")
	expect(t, s.cursor.page_pin.node == s.pages.pages.first, "s.cursor.page_pin.node == s.pages.pages.first.?")
	expectEqual(t, SizeCellCountInt(0), s.cursor.page_pin.x)
	expectEqual(t, SizeCellCountInt(0), s.cursor.page_pin.y)
	sc2OExpectCursorConsistent(t, s)
	// Selection pins are gone.
	expectEqual(t, uint(2), s.pages.CountTrackedPins())

	// Pages: only the active area remains.
	expectEqual(t, uint(3), s.pages.total_rows)
	expectEqual(t, uint(1), sc2ONodeCount(s))
	expectEqual(t, uint32(0), sc2OCp(t, s, 0, 0))
	expectEqual(t, uint32(0), sc2OCp(t, s, 0, 1))
	page := &s.cursor.page_pin.node.data
	expectEqual(t, uint(0), page.styles.Count())
	expectEqual(t, uint(0), page.hyperlink_set.Count())
	expect(t, !s.cursor.page_row.styled, "!s.cursor.page_row.styled")

	// The cursor is entirely fresh.
	expectEqual(t, SizeCellCountInt(0), s.cursor.x)
	expectEqual(t, SizeCellCountInt(0), s.cursor.y)
	expectEqual(t, CursorStyleBlock, s.cursor.cursor_style)
	expect(t, !s.cursor.pending_wrap, "!s.cursor.pending_wrap")
	expect(t, !s.cursor.protected, "!s.cursor.protected")
	expect(t, s.cursor.style.Default(), "s.cursor.style.default()")
	expectEqual(t, StyleDefaultId, s.cursor.style_id)
	expectEqual(t, HyperlinkId(0), s.cursor.hyperlink_id)
	expectEqual(t, SizeOffsetInt(0), s.cursor.hyperlink_implicit_id)
	expect(t, s.cursor.hyperlink == nil, "s.cursor.hyperlink == null")
	expectEqual(t, PageCellSemanticContentOutput, s.cursor.semantic_content)
	expect(t, !s.cursor.semantic_content_clear_eol, "!s.cursor.semantic_content_clear_eol")

	// Screen-level state.
	expect(t, !s.saved_cursor.Valid, "s.saved_cursor == null")
	expect(t, !s.selection.Valid, "s.selection == null")
	expectEqual(t, ANSIProtectedModeOff, s.protected_mode)
	expectEqual(t, CharsetsSlotsG0, s.charset.gl)
	expectEqual(t, CharsetsSlotsG2, s.charset.gr)
	expect(t, !s.charset.single_shift.Valid, "s.charset.single_shift == null")
	for _, slot := range sc2Slots() {
		expectEqual(t, CharsetsCharsetUtf8, s.charset.charsets.Get(slot))
	}
	expectEqual(t, uint8(0), s.kitty_keyboard.idx)
	expectEqual(t, KittyKeyFlagsDisabled, s.kitty_keyboard.Current())
	expect(t, !s.semantic_prompt.seen, "!s.semantic_prompt.seen")
	expect(t, s.semantic_prompt.click.Tag == ScreenSemanticPromptSemanticClickNone, "s.semantic_prompt.click == .none")

	// Clearing the selection flags the dirty selection bit; reset does not
	// otherwise touch the dirty flags.
	expect(t, s.dirty.selection, "s.dirty.selection")
	expect(t, s.dirty.hyperlink_hover, "s.dirty.hyperlink_hover")

	// Still fully functional afterwards.
	sc2Write(t, s, "xy")
	expectEqual(t, uint32('x'), sc2OCp(t, s, 0, 0))
	expectEqual(t, uint32('y'), sc2OCp(t, s, 1, 0))
}

// Zig: test "term51: reset: without a selection the dirty selection flag stays clear"
func TestSc2_term51_reset_without_a_selection_the_dirty_selection_flag_stays_clear(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 4, 2, 0)
	defer s.Deinit()

	sc2Write(t, s, "abc")
	s.Reset()
	expect(t, !s.dirty.selection, "!s.dirty.selection")
	expect(t, !s.dirty.hyperlink_hover, "!s.dirty.hyperlink_hover")

	// Dimensions and the no_scrollback flag are not part of the reset.
	expect(t, s.no_scrollback, "s.no_scrollback")
	expectEqual(t, SizeCellCountInt(4), s.pages.cols)
	expectEqual(t, SizeCellCountInt(2), s.pages.rows)
	// Zig-only: try std.testing.expect(s.alloc.ptr == alloc.ptr);

	// Twice in a row is fine.
	s.Reset()
	expectEqual(t, uint(2), s.pages.total_rows)
	sc2OExpectCursorConsistent(t, s)
}

// Zig: test "term51: reset: cursor on a later page goes back to the first page"
func TestSc2_term51_reset_cursor_on_a_later_page_goes_back_to_the_first_page(t *testing.T) {
	testStdCapacity(t)
	s := sc2OTwoPages(t, 20, 3)
	defer s.Deinit()

	zigTry(t, s.SetAttribute(sc2Bold()))
	zigTry(t, s.StartHyperlink([]uint8("https://example.com/"), Some([]uint8("id"))))
	sc2Write(t, s, "abc")
	expect(t, s.cursor.page_pin.node != s.pages.pages.first, "s.cursor.page_pin.node != s.pages.pages.first.?")

	s.Reset()
	expectEqual(t, uint(1), sc2ONodeCount(s))
	expect(t, s.cursor.page_pin.node == s.pages.pages.first, "s.cursor.page_pin.node == s.pages.pages.first.?")
	expectEqual(t, uint(3), s.pages.total_rows)
	expectEqual(t, uint(2), s.pages.CountTrackedPins())
	expectEqual(t, SizeCellCountInt(0), s.cursor.x)
	expectEqual(t, SizeCellCountInt(0), s.cursor.y)
	expect(t, s.cursor.style.Default(), "s.cursor.style.default()")
	expect(t, s.cursor.hyperlink == nil, "s.cursor.hyperlink == null")
	sc2OExpectCursorConsistent(t, s)
	s.AssertIntegrity()
}

// sc2OSelCase is term51OSelCase: a selection to clone, in screen coordinates
// of the 10 row source screen built by `sc2OCloneSel`.
type sc2OSelCase struct {
	sx   int
	sy   int
	ex   int
	ey   int
	rect bool
}

// sc2Clone is `try s.clone(alloc, top, bot)`.
func sc2Clone(t *testing.T, s *Screen, top Point, bot Optional[Point]) *Screen {
	t.Helper()
	s2, err := s.Clone(top, bot)
	zigTry(t, err)
	return &s2
}

// sc2OCloneSel is term51OCloneSel: build a 5 column, 6 row screen with 10
// lines of text (4 lines of history), select `c`, clone screen rows
// top_y..bot_y (inclusive) and return the cloned selection as screen
// coordinates {sx, sy, ex, ey} of the clone, or null when the clone has no
// selection.
func sc2OCloneSel(t *testing.T, c sc2OSelCase, top_y uint32, bot_y Optional[uint32]) Optional[[4]uint] {
	t.Helper()
	s := sc2Init(t, 5, 6, 1000)
	defer s.Deinit()
	sc2Write(t, s, "r0\nr1\nr2\nr3\nr4\nr5\nr6\nr7\nr8\nr9")
	expectEqual(t, uint(10), s.pages.total_rows)

	zigTry(t, s.Select(Some(SelectionInit(
		sc2Pin(t, s, sc2Screen(c.sx, c.sy)),
		sc2Pin(t, s, sc2Screen(c.ex, c.ey)),
		c.rect,
	))))

	bot := None[Point]()
	if bot_y.Valid {
		bot = Some(Point{Tag: PointTagScreen, Screen: PointCoordinate{y: bot_y.Value}})
	}
	s2 := sc2Clone(t, s, Point{Tag: PointTagScreen, Screen: PointCoordinate{y: top_y}}, bot)
	defer s2.Deinit()

	// The original selection is untouched and still tracked.
	expect(t, s.selection.Valid, "s.selection != null")
	expect(t, s.selection.Value.Tracked(), "s.selection.?.tracked()")

	if !s2.selection.Valid {
		return None[[4]uint]()
	}
	sel := s2.selection.Value
	expect(t, sel.Tracked(), "sel.tracked()")
	expectEqual(t, c.rect, sel.rectangle)
	a := sc2PointFromPin(t, s2, PointTagScreen, sel.Start()).Screen
	b := sc2PointFromPin(t, s2, PointTagScreen, sel.End()).Screen
	return Some([4]uint{uint(a.x), uint(a.y), uint(b.x), uint(b.y)})
}

// Zig: test "term51: clone: cursor, content and independence of a full clone"
func TestSc2_term51_clone_cursor_content_and_independence_of_a_full_clone(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 10)
	defer s.Deinit()
	sc2Write(t, s, "1ABCD\n2EFGH\n3IJ")

	s2 := sc2Clone(t, s, sc2Active(0, 0), None[Point]())
	defer s2.Deinit()

	expectEqual(t, SizeCellCountInt(3), s2.cursor.x)
	expectEqual(t, SizeCellCountInt(2), s2.cursor.y)
	sc2OExpectCursorConsistent(t, s2)
	// A different, tracked pin on the cloned pages.
	expect(t, s2.cursor.page_pin != s.cursor.page_pin, "s2.cursor.page_pin != s.cursor.page_pin")
	expect(t, s2.cursor.page_pin.node == s2.pages.pages.first, "s2.cursor.page_pin.node == s2.pages.pages.first.?")
	expect(t, s2.cursor.page_pin.node != s.pages.pages.first, "s2.cursor.page_pin.node != s.pages.pages.first.?")
	expectEqual(t, uint(2), s.pages.CountTrackedPins())

	expectEqual(t, SizeCellCountInt(10), s2.pages.cols)
	expectEqual(t, SizeCellCountInt(3), s2.pages.rows)
	expectEqual(t, uint(3), s2.pages.total_rows)
	// Zig-only: try std.testing.expect(s2.alloc.ptr == alloc.ptr);
	expect(t, !s2.no_scrollback, "!s2.no_scrollback")

	// Same text, independent memory.
	expectEqual(t, uint32('1'), sc2OCp(t, s2, 0, 0))
	expectEqual(t, uint32('J'), sc2OCp(t, s2, 2, 2))
	sc2OSetCp(t, s, 0, 0, 'Z')
	expectEqual(t, uint32('1'), sc2OCp(t, s2, 0, 0))
	s2.CursorAbsolute(0, 0)
	expectEqual(t, SizeCellCountInt(3), s.cursor.x)
	expectEqual(t, SizeCellCountInt(2), s.cursor.y)
	s2.AssertIntegrity()
	s.AssertIntegrity()
}

// Zig: test "term51: clone: only dimensions, content, cursor position and dirty flags are carried over"
func TestSc2_term51_clone_only_dimensions_content_cursor_position_and_dirty_flags_are_carried_over(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 10)
	defer s.Deinit()
	sc2Write(t, s, "abc")

	// Put every piece of non-copied state into a non-default value.
	zigTry(t, s.SetAttribute(sc2Bold()))
	zigTry(t, s.StartHyperlink([]uint8("https://example.com/"), Some([]uint8("id"))))
	s.cursor.cursor_style = ScreenCursorStyleUnderline
	s.cursor.protected = true
	s.cursor.pending_wrap = true
	s.cursor.hyperlink_implicit_id = 9
	s.CursorSetSemanticContent(sc2Prompt(OSCParsersSemanticPromptPromptKindInitial))
	s.cursor.semantic_content_clear_eol = true
	s.saved_cursor = Some(ScreenSavedCursor{
		x:            1,
		y:            1,
		style:        Style{},
		protected:    false,
		pending_wrap: false,
		origin:       false,
		charset:      screenCharsetStateDefault(),
	})
	s.charset.gl = CharsetsSlotsG1
	s.charset.charsets.Set(CharsetsSlotsG0, CharsetsCharsetBritish)
	s.protected_mode = ANSIProtectedModeIso
	s.kitty_keyboard.Push(KittyKeyFlags{report_all: true})
	s.semantic_prompt = ScreenSemanticPrompt{
		seen:  true,
		click: ScreenSemanticPromptSemanticClick{Tag: ScreenSemanticPromptSemanticClickClickEvents},
	}
	s.dirty = ScreenDirty{selection: true, hyperlink_hover: true}

	s2 := sc2Clone(t, s, sc2Active(0, 0), None[Point]())
	defer s2.Deinit()

	// Cursor position only
	expectEqual(t, SizeCellCountInt(3), s2.cursor.x)
	expectEqual(t, SizeCellCountInt(0), s2.cursor.y)
	expectEqual(t, CursorStyleBlock, s2.cursor.cursor_style)
	expect(t, !s2.cursor.pending_wrap, "!s2.cursor.pending_wrap")
	expect(t, !s2.cursor.protected, "!s2.cursor.protected")
	expect(t, s2.cursor.style.Default(), "s2.cursor.style.default()")
	expectEqual(t, StyleDefaultId, s2.cursor.style_id)
	expectEqual(t, HyperlinkId(0), s2.cursor.hyperlink_id)
	expectEqual(t, SizeOffsetInt(0), s2.cursor.hyperlink_implicit_id)
	expect(t, s2.cursor.hyperlink == nil, "s2.cursor.hyperlink == null")
	expectEqual(t, PageCellSemanticContentOutput, s2.cursor.semantic_content)
	expect(t, !s2.cursor.semantic_content_clear_eol, "!s2.cursor.semantic_content_clear_eol")

	// Screen state is not carried over...
	expect(t, !s2.saved_cursor.Valid, "s2.saved_cursor == null")
	expect(t, !s2.selection.Valid, "s2.selection == null")
	expectEqual(t, ANSIProtectedModeOff, s2.protected_mode)
	expectEqual(t, CharsetsSlotsG0, s2.charset.gl)
	expectEqual(t, CharsetsCharsetUtf8, s2.charset.charsets.Get(CharsetsSlotsG0))
	expectEqual(t, KittyKeyFlagsDisabled, s2.kitty_keyboard.Current())
	expect(t, !s2.semantic_prompt.seen, "!s2.semantic_prompt.seen")
	expect(t, s2.semantic_prompt.click.Tag == ScreenSemanticPromptSemanticClickNone, "s2.semantic_prompt.click == .none")

	// ... but the dirty flags are.
	expect(t, s2.dirty.selection, "s2.dirty.selection")
	expect(t, s2.dirty.hyperlink_hover, "s2.dirty.hyperlink_hover")

	// The page of the clone has no style or hyperlink refs of the cursor,
	// the cells' own references were cloned.
	page := &s2.cursor.page_pin.node.data
	expectEqual(t, uint(0), page.styles.Count())
	expectEqual(t, uint(0), page.hyperlink_set.Count())

	// And the original keeps all of it.
	expectEqual(t, CursorStyleUnderline, s.cursor.cursor_style)
	expect(t, s.cursor.pending_wrap, "s.cursor.pending_wrap")
	expect(t, s.cursor.hyperlink != nil, "s.cursor.hyperlink != null")
	expect(t, s.saved_cursor.Valid, "s.saved_cursor != null")
	expect(t, s.semantic_prompt.seen, "s.semantic_prompt.seen")
}

// Zig: test "term51: clone: dirty flags are copied as they are"
func TestSc2_term51_clone_dirty_flags_are_copied_as_they_are(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 4, 2, 0)
	defer s.Deinit()

	cases := []ScreenDirty{
		{selection: false, hyperlink_hover: false},
		{selection: true, hyperlink_hover: false},
		{selection: false, hyperlink_hover: true},
		{selection: true, hyperlink_hover: true},
	}
	for _, d := range cases {
		func() {
			s.dirty = d
			s2 := sc2Clone(t, s, sc2Active(0, 0), None[Point]())
			defer s2.Deinit()
			expectEqual(t, d, s2.dirty)
		}()
	}
}

// Zig: test "term51: clone: no_scrollback is copied"
func TestSc2_term51_clone_no_scrollback_is_copied(t *testing.T) {
	testStdCapacity(t)
	for _, no_sb := range []bool{true, false} {
		func() {
			var max_scrollback uint64 = 10
			if no_sb {
				max_scrollback = 0
			}
			s := sc2Init(t, 4, 2, max_scrollback)
			defer s.Deinit()
			expectEqual(t, no_sb, s.no_scrollback)
			s2 := sc2Clone(t, s, sc2Active(0, 0), None[Point]())
			defer s2.Deinit()
			expectEqual(t, no_sb, s2.no_scrollback)
		}()
	}
}

// Zig: test "term51: clone: cells keep their style and hyperlink, the cursor's own references do not"
func TestSc2_term51_clone_cells_keep_their_style_and_hyperlink_the_cursor_s_own_references_do_not(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	defer s.Deinit()
	zigTry(t, s.SetAttribute(sc2Bold()))
	zigTry(t, s.StartHyperlink([]uint8("https://example.com/"), Some([]uint8("id"))))
	sc2Write(t, s, "ab")

	page := &s.cursor.page_pin.node.data
	// Two cells and the cursor
	expectEqual(t, uint(3), sc2OStyleRefs(page, s.cursor.style_id))
	expectEqual(t, uint(3), sc2OLinkRefs(page, s.cursor.hyperlink_id))

	s2 := sc2Clone(t, s, sc2Active(0, 0), None[Point]())
	defer s2.Deinit()
	page2 := &s2.cursor.page_pin.node.data
	expectEqual(t, uint(1), page2.styles.Count())
	expectEqual(t, uint(1), page2.hyperlink_set.Count())

	cell := sc2OCell(t, s2, 0, 0)
	expectEqual(t, uint32('a'), cell.Codepoint())
	expect(t, cell.style_id != StyleDefaultId, "cell.style_id != style.default_id")
	expect(t, page2.styles.Get(cell.style_id).flags.bold, "page2.styles.get(page2.memory, cell.style_id).flags.bold")
	expectEqual(t, uint(2), sc2OStyleRefs(page2, cell.style_id))
	link_id := zigUnwrap(t, page2.LookupHyperlink(cell))
	expectEqual(t, uint(2), sc2OLinkRefs(page2, link_id))
	entry := page2.hyperlink_set.Get(link_id)
	expectEqualStrings(t, "https://example.com/", string(entry.uri))
	expectEqualStrings(t, "id", string(entry.id.Explicit))

	// The clone's cursor is not styled: writing there is plain.
	expectEqual(t, SizeCellCountInt(2), s2.cursor.x)
	sc2Write(t, s2, "Z")
	expectEqual(t, uint32('Z'), sc2OCp(t, s2, 2, 0))
	expectEqual(t, StyleDefaultId, sc2OCell(t, s2, 2, 0).style_id)
	expect(t, !sc2OCell(t, s2, 2, 0).hyperlink, "!term51OCell(&s2, 2, 0).hyperlink")
}

// Zig: test "term51: clone: bounds, history and padding with the cursor inside"
func TestSc2_term51_clone_bounds_history_and_padding_with_the_cursor_inside(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 3, 100)
	defer s.Deinit()
	// 6 lines in a 3 row screen: 3 history rows, cursor at (1, 2) -> screen y 5
	sc2Write(t, s, "r0\nr1\nr2\nr3\nr4\nr5")
	expectEqual(t, uint(6), s.pages.total_rows)
	expectEqual(t, SizeCellCountInt(2), s.cursor.x)
	expectEqual(t, SizeCellCountInt(2), s.cursor.y)

	// Whole screen including history: the cursor is relative to the
	// cloned active area.
	{
		s2 := sc2Clone(t, s, sc2Screen(0, 0), None[Point]())
		defer s2.Deinit()
		expectEqual(t, uint(6), s2.pages.total_rows)
		expectEqual(t, SizeCellCountInt(2), s2.cursor.x)
		expectEqual(t, SizeCellCountInt(2), s2.cursor.y)
		expectEqual(t, uint32('5'), sc2OCp(t, s2, 1, 2))
		top := sc2Pin(t, s2, sc2Screen(0, 0))
		expectEqual(t, uint32('r'), top.RowAndCell().cell.Codepoint())
		sc2OExpectCursorConsistent(t, s2)
	}

	// Only the history
	{
		s2 := sc2Clone(t, s, sc2History(0, 0), Some(sc2History(0, 2)))
		defer s2.Deinit()
		// 3 rows of history are exactly an active area.
		expectEqual(t, uint(3), s2.pages.total_rows)
		// The cursor was not in the clone: top-left.
		expectEqual(t, SizeCellCountInt(0), s2.cursor.x)
		expectEqual(t, SizeCellCountInt(0), s2.cursor.y)
		sc2OExpectCursorConsistent(t, s2)
		expectEqual(t, uint32('0'), sc2OCp(t, s2, 1, 0))
		expectEqual(t, uint32('2'), sc2OCp(t, s2, 1, 2))
	}

	// Cursor row only: padded with blank rows at the bottom (the cursor is
	// at the top of the clone).
	{
		s2 := sc2Clone(t, s, sc2Active(0, 2), Some(sc2Active(0, 2)))
		defer s2.Deinit()
		expectEqual(t, SizeCellCountInt(3), s2.pages.rows)
		expectEqual(t, uint(3), s2.pages.total_rows)
		expectEqual(t, SizeCellCountInt(2), s2.cursor.x)
		expectEqual(t, SizeCellCountInt(0), s2.cursor.y)
		expectEqual(t, uint32('5'), sc2OCp(t, s2, 1, 0))
		expectEqual(t, uint32(0), sc2OCp(t, s2, 0, 1))
		expectEqual(t, uint32(0), sc2OCp(t, s2, 0, 2))
		sc2OExpectCursorConsistent(t, s2)
	}

	// The bottom row is inclusive and the top row too.
	{
		s2 := sc2Clone(t, s, sc2Active(0, 1), Some(sc2Active(0, 2)))
		defer s2.Deinit()
		expectEqual(t, uint32('4'), sc2OCp(t, s2, 1, 0))
		expectEqual(t, uint32('5'), sc2OCp(t, s2, 1, 1))
		expectEqual(t, uint32(0), sc2OCp(t, s2, 1, 2))
		expectEqual(t, SizeCellCountInt(1), s2.cursor.y)
	}
}

// Zig: test "term51: clone: cursor outside the clone moves to the top left"
func TestSc2_term51_clone_cursor_outside_the_clone_moves_to_the_top_left(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 3, 100)
	defer s.Deinit()
	sc2Write(t, s, "1ABCD\n2EFGH\n3IJKL")
	s.CursorAbsolute(4, 1)

	// Clone starts below the cursor row.
	{
		s2 := sc2Clone(t, s, sc2Active(0, 2), None[Point]())
		defer s2.Deinit()
		expectEqual(t, SizeCellCountInt(0), s2.cursor.x)
		expectEqual(t, SizeCellCountInt(0), s2.cursor.y)
		expect(t, s2.cursor.page_pin.node == s2.pages.pages.first, "s2.cursor.page_pin.node == s2.pages.pages.first.?")
		expectEqual(t, SizeCellCountInt(0), s2.cursor.page_pin.x)
		expectEqual(t, SizeCellCountInt(0), s2.cursor.page_pin.y)
		sc2OExpectCursorConsistent(t, s2)
		expectEqual(t, uint32('3'), sc2OCp(t, s2, 0, 0))
	}

	// Clone ends above the cursor row.
	{
		s2 := sc2Clone(t, s, sc2Active(0, 0), Some(sc2Active(0, 0)))
		defer s2.Deinit()
		expectEqual(t, SizeCellCountInt(0), s2.cursor.x)
		expectEqual(t, SizeCellCountInt(0), s2.cursor.y)
		sc2OExpectCursorConsistent(t, s2)
	}

	// The cursor is the last row of the clone: inside.
	{
		s2 := sc2Clone(t, s, sc2Active(0, 0), Some(sc2Active(0, 1)))
		defer s2.Deinit()
		expectEqual(t, SizeCellCountInt(4), s2.cursor.x)
		expectEqual(t, SizeCellCountInt(1), s2.cursor.y)
	}
}

// Every tracked pin of the source that lies in the cloned rows is
// remapped into the clone, not only the cursor and selection pins: the
// source's (unused) viewport pin is copied too and stays tracked.
//
// Zig: test "term51: clone: tracked pins of the source inside the cloned rows are all copied"
func TestSc2_term51_clone_tracked_pins_of_the_source_inside_the_cloned_rows_are_all_copied(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 10)
	defer s.Deinit()
	sc2Write(t, s, "abc")
	expectEqual(t, uint(2), s.pages.CountTrackedPins())

	// Own viewport pin, copy of the source viewport pin and the cursor.
	{
		s2 := sc2Clone(t, s, sc2Active(0, 0), None[Point]())
		defer s2.Deinit()
		expectEqual(t, uint(3), s2.pages.CountTrackedPins())
	}

	// Both source pins are on row 0 which is cloned (and padded).
	{
		s2 := sc2Clone(t, s, sc2Active(0, 0), Some(sc2Active(0, 0)))
		defer s2.Deinit()
		expectEqual(t, uint(3), s2.pages.CountTrackedPins())
		expectEqual(t, SizeCellCountInt(3), s2.cursor.x)
	}

	// Neither is inside rows 1..2: own viewport pin and a new cursor pin.
	{
		s2 := sc2Clone(t, s, sc2Active(0, 1), None[Point]())
		defer s2.Deinit()
		expectEqual(t, uint(2), s2.pages.CountTrackedPins())
		expectEqual(t, SizeCellCountInt(0), s2.cursor.x)
	}

	// A selection adds its two pins when inside.
	zigTry(t, s.Select(Some(SelectionInit(sc2OPin(t, s, 0, 0), sc2OPin(t, s, 2, 1), false))))
	expectEqual(t, uint(4), s.pages.CountTrackedPins())
	{
		s2 := sc2Clone(t, s, sc2Active(0, 0), None[Point]())
		defer s2.Deinit()
		expectEqual(t, uint(5), s2.pages.CountTrackedPins())
	}
	// Selection entirely outside (no selection in the clone).
	{
		s2 := sc2Clone(t, s, sc2Active(0, 2), None[Point]())
		defer s2.Deinit()
		expect(t, !s2.selection.Valid, "s2.selection == null")
		expectEqual(t, uint(2), s2.pages.CountTrackedPins())
	}
}

// The fallback cursor pin is the first row of the first cloned page. When
// the clone has no more rows than the active area (so that row is
// active) that works; this pins the boundary: exactly `rows` rows.
//
// Zig: test "term51: clone: cursor outside a clone that has history is placed in the active area"
func TestSc2_term51_clone_cursor_outside_a_clone_that_has_history_is_placed_in_the_active_area(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 3, 100)
	defer s.Deinit()
	sc2Write(t, s, "r0\nr1\nr2\nr3\nr4\nr5\nr6\nr7")
	expectEqual(t, uint(8), s.pages.total_rows)

	s2 := sc2Clone(t, s, sc2Screen(0, 1), Some(sc2Screen(0, 3)))
	defer s2.Deinit()
	expectEqual(t, uint(3), s2.pages.total_rows)
	expectEqual(t, SizeCellCountInt(0), s2.cursor.x)
	expectEqual(t, SizeCellCountInt(0), s2.cursor.y)
	expectEqual(t, uint32('r'), sc2OCp(t, s2, 0, 0))
	expectEqual(t, uint32('1'), sc2OCp(t, s2, 1, 0))
	sc2OExpectCursorConsistent(t, s2)
}

// One row past the boundary pinned above: the clone has 4 rows, so its
// first row is history and the fallback cursor pin is outside the active
// area. Upstream aborts in assertIntegrity (Screen.zig:464-473, :554, :350;
// term51 docs/upstream-crashes.md, upstream-findings.md section 15, Screen
// item 1); the port returns ErrScreenCloneCursorOutside and leaves the
// source screen unchanged.
//
// Go-only: ruling (Screen.clone with the cursor outside the cloned region and more cloned rows than the active area returns ErrScreenCloneCursorOutside instead of upstream's abort; docs/mapping.md "Page-layer bookkeeping")
func TestSc2_go_only_clone_cursor_outside_a_clone_with_history_returns_an_error(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 3, 100)
	defer s.Deinit()
	sc2Write(t, s, "r0\nr1\nr2\nr3\nr4\nr5\nr6\nr7")
	expectEqual(t, uint(8), s.pages.total_rows)
	tracked := s.pages.CountTrackedPins()

	// The cursor (screen row 7) is outside rows 0..3.
	_, err := s.Clone(sc2Screen(0, 0), Some(sc2Screen(0, 3)))
	expect(t, err == ErrScreenCloneCursorOutside, "err == error.ScreenCloneCursorOutside")
	expectEqual(t, "ScreenCloneCursorOutside", err.Error())

	// The source is untouched.
	expectEqual(t, tracked, s.pages.CountTrackedPins())
	expectEqual(t, uint(8), s.pages.total_rows)
	expectEqual(t, SizeCellCountInt(2), s.cursor.x)
	expectEqual(t, SizeCellCountInt(2), s.cursor.y)
	s.AssertIntegrity()

	// The same region with the cursor inside it clones normally.
	s2 := sc2Clone(t, s, sc2Screen(0, 4), Some(sc2Screen(0, 7)))
	defer s2.Deinit()
	expectEqual(t, uint(4), s2.pages.total_rows)
	expectEqual(t, SizeCellCountInt(2), s2.cursor.x)
	expectEqual(t, SizeCellCountInt(2), s2.cursor.y)
	sc2OExpectCursorConsistent(t, s2)
}

// Zig: test "term51: clone: pages and cursor across a page boundary"
func TestSc2_term51_clone_pages_and_cursor_across_a_page_boundary(t *testing.T) {
	testStdCapacity(t)
	s := sc2OTwoPages(t, 20, 3)
	defer s.Deinit()
	sc2OSetCp(t, s, 0, 0, 'A')
	sc2OSetCp(t, s, 0, 1, 'B')
	sc2OSetCp(t, s, 0, 2, 'C')
	s.CursorAbsolute(7, 2)

	// Active only: two chunks, so two pages in the clone.
	{
		s2 := sc2Clone(t, s, sc2Active(0, 0), None[Point]())
		defer s2.Deinit()
		expectEqual(t, uint(2), sc2ONodeCount(s2))
		expectEqual(t, uint(3), s2.pages.total_rows)
		expectEqual(t, SizeCellCountInt(7), s2.cursor.x)
		expectEqual(t, SizeCellCountInt(2), s2.cursor.y)
		expect(t, s2.cursor.page_pin.node == s2.pages.pages.last, "s2.cursor.page_pin.node == s2.pages.pages.last.?")
		expectEqual(t, uint32('A'), sc2OCp(t, s2, 0, 0))
		expectEqual(t, uint32('B'), sc2OCp(t, s2, 0, 1))
		expectEqual(t, uint32('C'), sc2OCp(t, s2, 0, 2))
		sc2OExpectCursorConsistent(t, s2)
	}

	// The cursor's page only (a single row): padded by growing the page list.
	{
		s2 := sc2Clone(t, s, sc2Active(0, 2), None[Point]())
		defer s2.Deinit()
		expectEqual(t, uint(3), s2.pages.total_rows)
		expectEqual(t, SizeCellCountInt(7), s2.cursor.x)
		expectEqual(t, SizeCellCountInt(0), s2.cursor.y)
		expectEqual(t, uint32('C'), sc2OCp(t, s2, 0, 0))
		sc2OExpectCursorConsistent(t, s2)
	}

	// Everything including history: cursor is on the last page.
	{
		s2 := sc2Clone(t, s, sc2Screen(0, 0), None[Point]())
		defer s2.Deinit()
		expectEqual(t, s.pages.total_rows, s2.pages.total_rows)
		expectEqual(t, SizeCellCountInt(7), s2.cursor.x)
		expectEqual(t, SizeCellCountInt(2), s2.cursor.y)
		expect(t, s2.cursor.page_pin.node == s2.pages.pages.last, "s2.cursor.page_pin.node == s2.pages.pages.last.?")
		sc2OExpectCursorConsistent(t, s2)
	}
}

// sc2Some4 is `@as(?[4]usize, .{ a, b, c, d })`.
func sc2Some4(a uint, b uint, c uint, d uint) Optional[[4]uint] {
	return Some([4]uint{a, b, c, d})
}

// sc2Null4 is `@as(?[4]usize, null)`.
func sc2Null4() Optional[[4]uint] {
	return None[[4]uint]()
}

// Zig: test "term51: clone: selection fully inside, reversed and rectangular orders"
func TestSc2_term51_clone_selection_fully_inside_reversed_and_rectangular_orders(t *testing.T) {
	testStdCapacity(t)
	// The clone covers screen rows 2..7, which are 6 rows: no padding.
	// Clone row = source row - 2.

	// Forward, inside.
	expectEqual(t, sc2Some4(1, 1, 3, 3),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 3, ex: 3, ey: 5}, 2, Some(uint32(7))))

	// Single cell.
	expectEqual(t, sc2Some4(2, 2, 2, 2),
		sc2OCloneSel(t, sc2OSelCase{sx: 2, sy: 4, ex: 2, ey: 4}, 2, Some(uint32(7))))

	// Reverse: the clone is always stored with its top pin as start.
	expectEqual(t, sc2Some4(1, 1, 3, 3),
		sc2OCloneSel(t, sc2OSelCase{sx: 3, sy: 5, ex: 1, ey: 3}, 2, Some(uint32(7))))

	// Rectangle forward, reverse, mirrored forward and mirrored reverse.
	expectEqual(t, sc2Some4(1, 1, 3, 3),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 3, ex: 3, ey: 5, rect: true}, 2, Some(uint32(7))))
	expectEqual(t, sc2Some4(1, 1, 3, 3),
		sc2OCloneSel(t, sc2OSelCase{sx: 3, sy: 5, ex: 1, ey: 3, rect: true}, 2, Some(uint32(7))))
	// start top-right, end bottom-left
	expectEqual(t, sc2Some4(3, 1, 1, 3),
		sc2OCloneSel(t, sc2OSelCase{sx: 3, sy: 3, ex: 1, ey: 5, rect: true}, 2, Some(uint32(7))))
	// start bottom-left, end top-right
	expectEqual(t, sc2Some4(3, 1, 1, 3),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 5, ex: 3, ey: 3, rect: true}, 2, Some(uint32(7))))

	// First and last row of the clone are inside.
	expectEqual(t, sc2Some4(1, 0, 2, 5),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 2, ex: 2, ey: 7}, 2, Some(uint32(7))))
}

// Zig: test "term51: clone: selection start above the clone"
func TestSc2_term51_clone_selection_start_above_the_clone(t *testing.T) {
	testStdCapacity(t)

	// Non-rectangular: moved to the first column of the first row.
	expectEqual(t, sc2Some4(0, 0, 3, 2),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 0, ex: 3, ey: 4}, 2, Some(uint32(7))))
	// Starting one row above the clone.
	expectEqual(t, sc2Some4(0, 0, 3, 2),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 1, ex: 3, ey: 4}, 2, Some(uint32(7))))
	// Ending on the first row of the clone keeps one row.
	expectEqual(t, sc2Some4(0, 0, 3, 0),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 0, ex: 3, ey: 2}, 2, Some(uint32(7))))
	// Rectangle: the column of the start is preserved.
	expectEqual(t, sc2Some4(1, 0, 3, 2),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 0, ex: 3, ey: 4, rect: true}, 2, Some(uint32(7))))
	// Rectangle mirrored: top-right start above the clone keeps its column.
	expectEqual(t, sc2Some4(3, 0, 1, 2),
		sc2OCloneSel(t, sc2OSelCase{sx: 3, sy: 0, ex: 1, ey: 4, rect: true}, 2, Some(uint32(7))))
	// Reverse non-rectangular selection (the end is above).
	expectEqual(t, sc2Some4(0, 0, 3, 2),
		sc2OCloneSel(t, sc2OSelCase{sx: 3, sy: 4, ex: 1, ey: 0}, 2, Some(uint32(7))))
}

// Zig: test "term51: clone: selection end below the clone"
func TestSc2_term51_clone_selection_end_below_the_clone(t *testing.T) {
	testStdCapacity(t)

	// Non-rectangular: moved to the last column of the last row.
	expectEqual(t, sc2Some4(1, 1, 4, 5),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 3, ex: 3, ey: 9}, 2, Some(uint32(7))))
	// One row below the clone.
	expectEqual(t, sc2Some4(1, 1, 4, 5),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 3, ex: 3, ey: 8}, 2, Some(uint32(7))))
	// Starting on the last row of the clone.
	expectEqual(t, sc2Some4(1, 5, 4, 5),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 7, ex: 3, ey: 9}, 2, Some(uint32(7))))
	// Rectangle: the column of the end is preserved.
	expectEqual(t, sc2Some4(1, 1, 3, 5),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 3, ex: 3, ey: 9, rect: true}, 2, Some(uint32(7))))
	// Rectangle mirrored.
	expectEqual(t, sc2Some4(3, 1, 1, 5),
		sc2OCloneSel(t, sc2OSelCase{sx: 3, sy: 3, ex: 1, ey: 9, rect: true}, 2, Some(uint32(7))))
	// Reverse: the start is below.
	expectEqual(t, sc2Some4(1, 1, 4, 5),
		sc2OCloneSel(t, sc2OSelCase{sx: 3, sy: 9, ex: 1, ey: 3}, 2, Some(uint32(7))))
	// Mirrored reverse rectangle with the bottom-left start outside.
	expectEqual(t, sc2Some4(3, 1, 1, 5),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 9, ex: 3, ey: 3, rect: true}, 2, Some(uint32(7))))
}

// Zig: test "term51: clone: selection around the whole clone and entirely outside of it"
func TestSc2_term51_clone_selection_around_the_whole_clone_and_entirely_outside_of_it(t *testing.T) {
	testStdCapacity(t)

	// Both ends outside, selection covers the clone.
	expectEqual(t, sc2Some4(0, 0, 4, 5),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 0, ex: 3, ey: 9}, 2, Some(uint32(7))))
	// Covering it by exactly one row on each side.
	expectEqual(t, sc2Some4(0, 0, 4, 5),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 1, ex: 3, ey: 8}, 2, Some(uint32(7))))
	// Rectangle: columns preserved.
	expectEqual(t, sc2Some4(1, 0, 3, 5),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 0, ex: 3, ey: 9, rect: true}, 2, Some(uint32(7))))
	// Reverse covering selection.
	expectEqual(t, sc2Some4(0, 0, 4, 5),
		sc2OCloneSel(t, sc2OSelCase{sx: 3, sy: 9, ex: 1, ey: 0}, 2, Some(uint32(7))))

	// Entirely above, ending exactly one row above the clone.
	expectEqual(t, sc2Null4(),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 0, ex: 3, ey: 1}, 2, Some(uint32(7))))
	expectEqual(t, sc2Null4(),
		sc2OCloneSel(t, sc2OSelCase{sx: 3, sy: 1, ex: 1, ey: 0}, 2, Some(uint32(7))))
	// Entirely below, starting exactly one row below the clone.
	expectEqual(t, sc2Null4(),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 8, ex: 3, ey: 9}, 2, Some(uint32(7))))
	expectEqual(t, sc2Null4(),
		sc2OCloneSel(t, sc2OSelCase{sx: 3, sy: 9, ex: 1, ey: 8}, 2, Some(uint32(7))))
	// Single cells just outside.
	expectEqual(t, sc2Null4(),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 1, ex: 1, ey: 1}, 2, Some(uint32(7))))
	expectEqual(t, sc2Null4(),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 8, ex: 1, ey: 8}, 2, Some(uint32(7))))
	// Rectangles just outside.
	expectEqual(t, sc2Null4(),
		sc2OCloneSel(t, sc2OSelCase{sx: 3, sy: 0, ex: 1, ey: 1, rect: true}, 2, Some(uint32(7))))
	expectEqual(t, sc2Null4(),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 8, ex: 3, ey: 9, rect: true}, 2, Some(uint32(7))))
}

// Zig: test "term51: clone: selection with no bottom bound clones to the end of the screen"
func TestSc2_term51_clone_selection_with_no_bottom_bound_clones_to_the_end_of_the_screen(t *testing.T) {
	testStdCapacity(t)

	// Clone covers screen rows 4..9 (the active area).
	expectEqual(t, sc2Some4(1, 1, 3, 5),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 5, ex: 3, ey: 9}, 4, None[uint32]()))
	expectEqual(t, sc2Some4(0, 0, 3, 5),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 0, ex: 3, ey: 9}, 4, None[uint32]()))
	expectEqual(t, sc2Null4(),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 0, ex: 3, ey: 3}, 4, None[uint32]()))
}

// Zig: test "term51: clone: selection end of a padded clone moves to the padded last row"
func TestSc2_term51_clone_selection_end_of_a_padded_clone_moves_to_the_padded_last_row(t *testing.T) {
	testStdCapacity(t)

	// The clone is screen row 8 and 9 only, padded to 6 rows; an end pin
	// below the clone is moved to the last row of the padded page list.
	expectEqual(t, sc2Some4(1, 0, 3, 1),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 8, ex: 3, ey: 9}, 8, Some(uint32(9))))
	expectEqual(t, sc2Some4(1, 0, 4, 5),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 8, ex: 3, ey: 9}, 8, Some(uint32(8))))
	expectEqual(t, sc2Some4(0, 0, 3, 1),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 0, ex: 3, ey: 9}, 8, Some(uint32(9))))
	expectEqual(t, sc2Null4(),
		sc2OCloneSel(t, sc2OSelCase{sx: 1, sy: 0, ex: 3, ey: 7}, 8, Some(uint32(9))))
}

// Not translated: Zig test "term51: clone: out of memory at every allocation unwinds without leaking"
// (std.testing.FailingAllocator); see scripts/not-translated.txt.

// sc2PageHasRow reports whether row is an element of page.rows. It stands for
// Zig's address-range check against the page's memory (`addr >=
// @intFromPtr(first.memory.ptr)` and `addr < ... + first.memory.len`): a row
// of the page lives in the page's memory.
func sc2PageHasRow(page *Page, row *PageRow) bool {
	for i := range page.rows {
		if &page.rows[i] == row {
			return true
		}
	}
	return false
}

// sc2PageHasCell reports whether cell is an element of page.cells (see
// sc2PageHasRow).
func sc2PageHasCell(page *Page, cell *PageCell) bool {
	for i := range page.cells {
		if &page.cells[i] == cell {
			return true
		}
	}
	return false
}

// sc2OExpectMovedFrom is term51OExpectMovedFrom: expect that the cursor's
// style and hyperlink were moved off `old_node` (no references left there) and
// are accounted for exactly once on the page the cursor is on now. The cursor
// must have a style and an explicit-id link created by `sc2OStyleAndLink`.
func sc2OExpectMovedFrom(t *testing.T, s *Screen, old_node *PageListNode, old_style_id StyleId, old_link_id HyperlinkId) {
	t.Helper()
	expect(t, s.cursor.page_pin.node != old_node, "s.cursor.page_pin.node != old_node")
	expectEqual(t, uint(0), sc2OStyleRefs(&old_node.data, old_style_id))
	expectEqual(t, uint(0), sc2OLinkRefs(&old_node.data, old_link_id))

	page := &s.cursor.page_pin.node.data
	expect(t, s.cursor.style.flags.bold, "s.cursor.style.flags.bold")
	expect(t, s.cursor.style_id != StyleDefaultId, "s.cursor.style_id != style.default_id")
	expect(t, page.styles.Get(s.cursor.style_id).flags.bold, "page.styles.get(page.memory, s.cursor.style_id).flags.bold")
	expectEqual(t, uint(1), sc2OStyleRefs(page, s.cursor.style_id))
	expectEqual(t, uint(1), page.styles.Count())

	expect(t, s.cursor.hyperlink != nil, "s.cursor.hyperlink != null")
	expect(t, s.cursor.hyperlink_id != 0, "s.cursor.hyperlink_id != 0")
	expectEqual(t, uint(1), sc2OLinkRefs(page, s.cursor.hyperlink_id))
	expectEqual(t, uint(1), page.hyperlink_set.Count())
	link := s.cursor.hyperlink
	expectEqualStrings(t, "https://example.com/", string(link.uri))
	expectEqualStrings(t, "ID", string(link.id.Explicit))
	entry := page.hyperlink_set.Get(s.cursor.hyperlink_id)
	expectEqualStrings(t, "https://example.com/", string(entry.uri))
	expectEqualStrings(t, "ID", string(entry.id.Explicit))
}

// sc2OStyleAndLink is term51OStyleAndLink: give the cursor a bold style and an
// explicit-id hyperlink and check that each is accounted for once on the
// cursor's page.
func sc2OStyleAndLink(t *testing.T, s *Screen) {
	t.Helper()
	zigTry(t, s.SetAttribute(sc2Bold()))
	zigTry(t, s.StartHyperlink([]uint8("https://example.com/"), Some([]uint8("ID"))))
	page := &s.cursor.page_pin.node.data
	expectEqual(t, uint(1), sc2OStyleRefs(page, s.cursor.style_id))
	expectEqual(t, uint(1), sc2OLinkRefs(page, s.cursor.hyperlink_id))
}

// sc2OIncreaseCase is term51OIncreaseCase.
func sc2OIncreaseCase(t *testing.T, adj Optional[PageListIncreaseCapacity]) {
	t.Helper()
	s := sc2Init(t, 10, 3, 0)
	defer s.Deinit()
	zigTry(t, s.SetAttribute(sc2Bold()))
	zigTry(t, s.StartHyperlink([]uint8("https://example.com/"), Some([]uint8("id"))))
	sc2Write(t, s, "abc")
	s.CursorAbsolute(5, 1)
	s.cursor.pending_wrap = true

	old_node := s.cursor.page_pin.node
	old_cap := old_node.data.capacity
	old_link := s.cursor.hyperlink
	old_style := s.cursor.style
	old_style_id := s.cursor.style_id
	// Three cells and the cursor.
	expectEqual(t, uint(4), sc2OStyleRefs(&old_node.data, old_style_id))
	expectEqual(t, uint(4), sc2OLinkRefs(&old_node.data, s.cursor.hyperlink_id))

	new_node, err := s.IncreaseCapacity(old_node, adj)
	zigTry(t, err)

	// A brand new page replaced the old one in the same list position.
	expect(t, new_node != old_node, "new_node != old_node")
	expect(t, new_node == s.cursor.page_pin.node, "new_node == s.cursor.page_pin.node")
	expect(t, new_node == s.pages.pages.first, "new_node == s.pages.pages.first.?")
	expect(t, new_node == s.pages.pages.last, "new_node == s.pages.pages.last.?")

	// Only the requested dimension doubled.
	want := old_cap
	if adj.Valid {
		switch adj.Value {
		case PageListIncreaseCapacityStyles:
			want.styles *= 2
		case PageListIncreaseCapacityGraphemeBytes:
			want.grapheme_bytes *= 2
		case PageListIncreaseCapacityHyperlinkBytes:
			want.hyperlink_bytes *= 2
		case PageListIncreaseCapacityStringBytes:
			want.string_bytes *= 2
		}
	}
	expectEqual(t, want, new_node.data.capacity)

	// The cursor did not move and its caches are valid again.
	expectEqual(t, SizeCellCountInt(5), s.cursor.x)
	expectEqual(t, SizeCellCountInt(1), s.cursor.y)
	expect(t, s.cursor.pending_wrap, "s.cursor.pending_wrap")
	sc2OExpectCursorConsistent(t, s)
	s.AssertIntegrity()

	// Style: re-added so the page has the references of the cells (3) and
	// the cursor (1).
	page := &new_node.data
	expect(t, old_style == s.cursor.style, "std.meta.eql(old_style, s.cursor.style)")
	expect(t, s.cursor.style_id != StyleDefaultId, "s.cursor.style_id != style.default_id")
	expectEqual(t, uint(1), page.styles.Count())
	expectEqual(t, uint(4), sc2OStyleRefs(page, s.cursor.style_id))
	expectEqual(t, s.cursor.style_id, sc2OCell(t, s, 0, 0).style_id)

	// Hyperlink: re-added too, as a new heap allocation with equal content.
	expect(t, s.cursor.hyperlink != nil, "s.cursor.hyperlink != null")
	expect(t, s.cursor.hyperlink != old_link, "s.cursor.hyperlink.? != old_link")
	expectEqualStrings(t, "https://example.com/", string(s.cursor.hyperlink.uri))
	expectEqualStrings(t, "id", string(s.cursor.hyperlink.id.Explicit))
	expect(t, s.cursor.hyperlink_id != 0, "s.cursor.hyperlink_id != 0")
	expectEqual(t, uint(1), page.hyperlink_set.Count())
	expectEqual(t, uint(4), sc2OLinkRefs(page, s.cursor.hyperlink_id))
	expectEqual(t, s.cursor.hyperlink_id, zigUnwrap(t, page.LookupHyperlink(sc2OCell(t, s, 0, 0))))

	// Writing keeps working with the new page (a pending wrap would make
	// the test writer assert, so clear it).
	s.cursor.pending_wrap = false
	sc2Write(t, s, "Z")
	expectEqual(t, uint32('Z'), sc2OCp(t, s, 5, 1))
	expectEqual(t, s.cursor.style_id, sc2OCell(t, s, 5, 1).style_id)
	expectEqual(t, uint(5), sc2OStyleRefs(page, s.cursor.style_id))
	expectEqual(t, uint(5), sc2OLinkRefs(page, s.cursor.hyperlink_id))
}

// Zig: test "term51: increaseCapacity: cursor page with null adjustment rehashes at the same capacity"
func TestSc2_term51_increaseCapacity_cursor_page_with_null_adjustment_rehashes_at_the_same_capacity(t *testing.T) {
	testStdCapacity(t)
	sc2OIncreaseCase(t, None[PageListIncreaseCapacity]())
}

// Zig: test "term51: increaseCapacity: cursor page doubles styles"
func TestSc2_term51_increaseCapacity_cursor_page_doubles_styles(t *testing.T) {
	testStdCapacity(t)
	sc2OIncreaseCase(t, Some(PageListIncreaseCapacityStyles))
}

// Zig: test "term51: increaseCapacity: cursor page doubles grapheme bytes"
func TestSc2_term51_increaseCapacity_cursor_page_doubles_grapheme_bytes(t *testing.T) {
	testStdCapacity(t)
	sc2OIncreaseCase(t, Some(PageListIncreaseCapacityGraphemeBytes))
}

// Zig: test "term51: increaseCapacity: cursor page doubles hyperlink bytes"
func TestSc2_term51_increaseCapacity_cursor_page_doubles_hyperlink_bytes(t *testing.T) {
	testStdCapacity(t)
	sc2OIncreaseCase(t, Some(PageListIncreaseCapacityHyperlinkBytes))
}

// Zig: test "term51: increaseCapacity: cursor page doubles string bytes"
func TestSc2_term51_increaseCapacity_cursor_page_doubles_string_bytes(t *testing.T) {
	testStdCapacity(t)
	sc2OIncreaseCase(t, Some(PageListIncreaseCapacityStringBytes))
}

// Zig: test "term51: increaseCapacity: cursor without style or hyperlink"
func TestSc2_term51_increaseCapacity_cursor_without_style_or_hyperlink(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	defer s.Deinit()
	sc2Write(t, s, "abc")
	s.CursorAbsolute(4, 2)

	old_node := s.cursor.page_pin.node
	old_cap := old_node.data.capacity
	new_node := sc2IncreaseCapacity(t, s, old_node, PageListIncreaseCapacityStyles)
	expect(t, new_node != old_node, "new_node != old_node")
	expect(t, new_node == s.cursor.page_pin.node, "new_node == s.cursor.page_pin.node")
	expectEqual(t, old_cap.styles*2, new_node.data.capacity.styles)
	expectEqual(t, StyleDefaultId, s.cursor.style_id)
	expect(t, s.cursor.style.Default(), "s.cursor.style.default()")
	expectEqual(t, HyperlinkId(0), s.cursor.hyperlink_id)
	expect(t, s.cursor.hyperlink == nil, "s.cursor.hyperlink == null")
	expectEqual(t, uint(0), new_node.data.styles.Count())
	expectEqual(t, uint(0), new_node.data.hyperlink_set.Count())
	expectEqual(t, SizeCellCountInt(4), s.cursor.x)
	expectEqual(t, SizeCellCountInt(2), s.cursor.y)
	sc2OExpectCursorConsistent(t, s)
	expectEqual(t, uint32('b'), sc2OCp(t, s, 1, 0))
}

// Zig: test "term51: increaseCapacity: cursor style and link with no cell using them"
func TestSc2_term51_increaseCapacity_cursor_style_and_link_with_no_cell_using_them(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	defer s.Deinit()
	sc2OStyleAndLink(t, s)

	new_node := sc2IncreaseCapacity(t, s, s.cursor.page_pin.node, PageListIncreaseCapacityHyperlinkBytes)
	expect(t, new_node == s.cursor.page_pin.node, "new_node == s.cursor.page_pin.node")
	page := &new_node.data
	expectEqual(t, uint(1), sc2OStyleRefs(page, s.cursor.style_id))
	expectEqual(t, uint(1), sc2OLinkRefs(page, s.cursor.hyperlink_id))
	expect(t, s.cursor.style.flags.bold, "s.cursor.style.flags.bold")
	expectEqualStrings(t, "ID", string(s.cursor.hyperlink.id.Explicit))
}

// Zig: test "term51: increaseCapacity: implicit hyperlink id is preserved, counter untouched"
func TestSc2_term51_increaseCapacity_implicit_hyperlink_id_is_preserved_counter_untouched(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	defer s.Deinit()
	sc2StartHyperlink(t, s, "https://example.com/")
	sc2StartHyperlink(t, s, "https://example.org/")
	expectEqual(t, SizeOffsetInt(2), s.cursor.hyperlink_implicit_id)
	expectEqual(t, SizeOffsetInt(1), s.cursor.hyperlink.id.Implicit)
	sc2Write(t, s, "ab")

	new_node := sc2IncreaseCapacity(t, s, s.cursor.page_pin.node, PageListIncreaseCapacityStringBytes)
	expect(t, new_node == s.cursor.page_pin.node, "new_node == s.cursor.page_pin.node")
	expectEqual(t, SizeOffsetInt(2), s.cursor.hyperlink_implicit_id)
	expectEqual(t, SizeOffsetInt(1), s.cursor.hyperlink.id.Implicit)
	expectEqualStrings(t, "https://example.org/", string(s.cursor.hyperlink.uri))
	page := &new_node.data
	expectEqual(t, uint(3), sc2OLinkRefs(page, s.cursor.hyperlink_id))
	expectEqual(t, s.cursor.hyperlink_id, zigUnwrap(t, page.LookupHyperlink(sc2OCell(t, s, 0, 0))))
}

// Zig: test "term51: increaseCapacity: other page leaves the cursor accounting alone"
func TestSc2_term51_increaseCapacity_other_page_leaves_the_cursor_accounting_alone(t *testing.T) {
	testStdCapacity(t)
	s := sc2OTwoPages(t, 20, 3)
	defer s.Deinit()
	sc2OStyleAndLink(t, s)

	first := s.pages.pages.first
	cursor_node := s.cursor.page_pin.node
	expect(t, first != cursor_node, "first != cursor_node")
	old_cap := first.data.capacity
	style_id := s.cursor.style_id
	link_id := s.cursor.hyperlink_id
	link_ptr := s.cursor.hyperlink
	pin_ptr := s.cursor.page_pin
	row_ptr := s.cursor.page_row
	cell_ptr := s.cursor.page_cell

	new_node := sc2IncreaseCapacity(t, s, first, PageListIncreaseCapacityStyles)
	expect(t, new_node != first, "new_node != first")
	expect(t, new_node == s.pages.pages.first, "new_node == s.pages.pages.first.?")
	expect(t, new_node.next == cursor_node, "new_node.next == cursor_node")
	expectEqual(t, old_cap.styles*2, new_node.data.capacity.styles)

	// Nothing about the cursor changed.
	expect(t, s.cursor.page_pin == pin_ptr, "s.cursor.page_pin == pin_ptr")
	expect(t, s.cursor.page_pin.node == cursor_node, "s.cursor.page_pin.node == cursor_node")
	expect(t, s.cursor.page_row == row_ptr, "s.cursor.page_row == row_ptr")
	expect(t, s.cursor.page_cell == cell_ptr, "s.cursor.page_cell == cell_ptr")
	expectEqual(t, style_id, s.cursor.style_id)
	expectEqual(t, link_id, s.cursor.hyperlink_id)
	expect(t, s.cursor.hyperlink == link_ptr, "s.cursor.hyperlink.? == link_ptr")
	expectEqual(t, uint(1), sc2OStyleRefs(&cursor_node.data, style_id))
	expectEqual(t, uint(1), sc2OLinkRefs(&cursor_node.data, link_id))
	sc2OExpectCursorConsistent(t, s)
}

// Zig: test "term51: increaseCapacity: tracked pins of the cursor page follow, others stay"
func TestSc2_term51_increaseCapacity_tracked_pins_of_the_cursor_page_follow_others_stay(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	defer s.Deinit()
	sc2Write(t, s, "abc")
	zigTry(t, s.Select(Some(SelectionInit(sc2OPin(t, s, 0, 0), sc2OPin(t, s, 2, 1), false))))

	new_node, err := s.IncreaseCapacity(s.cursor.page_pin.node, None[PageListIncreaseCapacity]())
	zigTry(t, err)
	expect(t, s.selection.Value.Start().node == new_node, "s.selection.?.start().node == new_node")
	expect(t, s.selection.Value.End().node == new_node, "s.selection.?.end().node == new_node")
	expect(t, s.pages.viewport_pin.node == new_node, "s.pages.viewport_pin.node == new_node")
	sc2OExpectPin(t, s, s.selection.Value.Start(), 0, 0)
	sc2OExpectPin(t, s, s.selection.Value.End(), 2, 1)
}

// Zig: test "term51: increaseCapacity: out of space leaves the cursor state intact"
func TestSc2_term51_increaseCapacity_out_of_space_leaves_the_cursor_state_intact(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	defer s.Deinit()
	zigTry(t, s.SetAttribute(sc2Bold()))
	sc2Write(t, s, "a")

	node := s.cursor.page_pin.node
	steps := 0
	for {
		next, err := s.IncreaseCapacity(node, Some(PageListIncreaseCapacityStyles))
		if err != nil {
			expectEqual(t, error(ErrOutOfSpace), err)
			break
		}
		node = next
		steps++
		expect(t, steps < 64, "steps < 64")
	}
	expect(t, steps >= 1, "steps >= 1")
	expectEqual(t, SizeStyleCountInt(math.MaxUint16), node.data.capacity.styles)

	// The failed call changed nothing.
	expect(t, s.cursor.page_pin.node == node, "s.cursor.page_pin.node == node")
	expect(t, node == s.pages.pages.first, "node == s.pages.pages.first.?")
	expectEqual(t, uint(2), sc2OStyleRefs(&node.data, s.cursor.style_id))
	expect(t, s.cursor.style.flags.bold, "s.cursor.style.flags.bold")
	sc2OExpectCursorConsistent(t, s)
	s.AssertIntegrity()

	// And one more try fails the same way.
	_, err := s.IncreaseCapacity(node, Some(PageListIncreaseCapacityStyles))
	expectEqual(t, error(ErrOutOfSpace), err)
}

// Not translated: Zig test "term51: increaseCapacity: failing to re-add the hyperlink drops it, the cursor keeps working"
// (std.testing.FailingAllocator installed as Screen.alloc); see
// scripts/not-translated.txt.

// Zig: test "term51: cursorCellRight: returns the cell n columns right without moving"
func TestSc2_term51_cursorCellRight_returns_the_cell_n_columns_right_without_moving(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	defer s.Deinit()
	for x := 0; x < 10; x++ {
		sc2OSetCp(t, s, x, 1, uint32('A'+x))
	}
	s.CursorAbsolute(2, 1)
	s.pages.ClearDirty()
	before := s.cursor

	// n = 0 is the cursor's own cell.
	expect(t, s.CursorCellRight(0) == s.cursor.page_cell, "s.cursorCellRight(0) == s.cursor.page_cell")
	expectEqual(t, uint32('C'), s.CursorCellRight(0).Codepoint())
	expectEqual(t, uint32('D'), s.CursorCellRight(1).Codepoint())
	expectEqual(t, uint32('F'), s.CursorCellRight(3).Codepoint())
	// n = cols - 1 - x is the last cell
	expectEqual(t, uint32('J'), s.CursorCellRight(7).Codepoint())
	for n := 0; n < 8; n++ {
		expect(t, s.CursorCellRight(SizeCellCountInt(n)) == sc2OCell(t, s, 2+n, 1),
			"s.cursorCellRight(n) == term51OCell(&s, 2 + n, 1)")
	}

	// Nothing moved or got dirty.
	expect(t, before == s.cursor, "std.meta.eql(before, s.cursor)")
	expectEqualSlices(t, []bool{false, false, false}, sc2ODirtyRows(t, s, 3))
	expect(t, !s.cursor.page_pin.node.data.dirty, "!s.cursor.page_pin.node.data.dirty")

	// From the first column and from a last-column cursor.
	s.CursorAbsolute(0, 1)
	expectEqual(t, uint32('J'), s.CursorCellRight(9).Codepoint())
	s.CursorAbsolute(9, 1)
	expect(t, s.CursorCellRight(0) == sc2OCell(t, s, 9, 1), "s.cursorCellRight(0) == term51OCell(&s, 9, 1)")
}

// Zig: test "term51: cursorCellLeft: returns the cell n columns left without moving"
func TestSc2_term51_cursorCellLeft_returns_the_cell_n_columns_left_without_moving(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	defer s.Deinit()
	for x := 0; x < 10; x++ {
		sc2OSetCp(t, s, x, 1, uint32('A'+x))
	}
	s.CursorAbsolute(2, 1)
	s.pages.ClearDirty()
	before := s.cursor

	expect(t, s.CursorCellLeft(0) == s.cursor.page_cell, "s.cursorCellLeft(0) == s.cursor.page_cell")
	expectEqual(t, uint32('B'), s.CursorCellLeft(1).Codepoint())
	// n = x is the first cell
	expectEqual(t, uint32('A'), s.CursorCellLeft(2).Codepoint())
	for n := 0; n < 3; n++ {
		expect(t, s.CursorCellLeft(SizeCellCountInt(n)) == sc2OCell(t, s, 2-n, 1),
			"s.cursorCellLeft(n) == term51OCell(&s, 2 - n, 1)")
	}
	expect(t, before == s.cursor, "std.meta.eql(before, s.cursor)")
	expectEqualSlices(t, []bool{false, false, false}, sc2ODirtyRows(t, s, 3))

	s.CursorAbsolute(9, 1)
	expectEqual(t, uint32('A'), s.CursorCellLeft(9).Codepoint())
	expectEqual(t, uint32('I'), s.CursorCellLeft(1).Codepoint())
	s.CursorAbsolute(0, 1)
	expect(t, s.CursorCellLeft(0) == sc2OCell(t, s, 0, 1), "s.cursorCellLeft(0) == term51OCell(&s, 0, 1)")
}

// Zig: test "term51: cursorCellEndOfPrev: last cell of the row above, any column"
func TestSc2_term51_cursorCellEndOfPrev_last_cell_of_the_row_above_any_column(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	defer s.Deinit()
	sc2OSetCp(t, s, 9, 0, 'X')
	sc2OSetCp(t, s, 9, 1, 'Y')
	sc2OSetCp(t, s, 8, 0, 'x')
	s.pages.ClearDirty()

	for _, x := range []SizeCellCountInt{0, 3, 9} {
		s.CursorAbsolute(x, 1)
		s.pages.ClearDirty()
		before := s.cursor
		cell := s.CursorCellEndOfPrev()
		expect(t, cell == sc2OCell(t, s, 9, 0), "cell == term51OCell(&s, 9, 0)")
		expectEqual(t, uint32('X'), cell.Codepoint())
		expect(t, before == s.cursor, "std.meta.eql(before, s.cursor)")
		expectEqualSlices(t, []bool{false, false, false}, sc2ODirtyRows(t, s, 3))
	}

	s.CursorAbsolute(2, 2)
	expectEqual(t, uint32('Y'), s.CursorCellEndOfPrev().Codepoint())
	expect(t, s.CursorCellEndOfPrev() == sc2OCell(t, s, 9, 1), "s.cursorCellEndOfPrev() == term51OCell(&s, 9, 1)")
}

// Zig: test "term51: cursorCellEndOfPrev: single column and across a page boundary"
func TestSc2_term51_cursorCellEndOfPrev_single_column_and_across_a_page_boundary(t *testing.T) {
	testStdCapacity(t)

	{
		s := sc2Init(t, 1, 2, 0)
		defer s.Deinit()
		sc2OSetCp(t, s, 0, 0, 'Q')
		s.CursorAbsolute(0, 1)
		expectEqual(t, uint32('Q'), s.CursorCellEndOfPrev().Codepoint())
	}

	s := sc2OTwoPages(t, 20, 3)
	defer s.Deinit()
	sc2OSetCp(t, s, 19, 1, 'Z')
	s.CursorAbsolute(5, 2)
	expect(t, s.cursor.page_pin.node == s.pages.pages.last, "s.cursor.page_pin.node == s.pages.pages.last.?")

	before := s.cursor
	cell := s.CursorCellEndOfPrev()
	expectEqual(t, uint32('Z'), cell.Codepoint())
	expect(t, cell == sc2OCell(t, s, 19, 1), "cell == term51OCell(&s, 19, 1)")
	// The cell lives in the first page's memory.
	first := &s.pages.pages.first.data
	// Zig: addr >= @intFromPtr(first.memory.ptr) and addr < ... + first.memory.len,
	// as an element of the page's cells.
	expect(t, sc2PageHasCell(first, cell), "cell is in the first page's memory")
	expect(t, before == s.cursor, "std.meta.eql(before, s.cursor)")
}

// sc2OCursorMoveCase is the body shared by the cursorRight and cursorLeft
// tests: the cursor starts at (start_x, 1) with a style and a link, and every
// step must move x, the pin and the cell together and nothing else.
func sc2OCursorMoveCase(t *testing.T, start_x SizeCellCountInt, left bool, steps [][2]SizeCellCountInt) {
	t.Helper()
	s := sc2Init(t, 10, 3, 0)
	defer s.Deinit()
	s.CursorAbsolute(start_x, 1)
	sc2OStyleAndLink(t, s)
	node := s.cursor.page_pin.node
	style_id := s.cursor.style_id
	link_id := s.cursor.hyperlink_id
	row := s.cursor.page_row
	s.cursor.pending_wrap = true
	s.pages.ClearDirty()

	// n = 1, n = 0 and the largest n that stays inside the row.
	for _, st := range steps {
		if left {
			s.CursorLeft(st[0])
		} else {
			s.CursorRight(st[0])
		}
		expectEqual(t, st[1], s.cursor.x)
		expectEqual(t, st[1], s.cursor.page_pin.x)
		expectEqual(t, SizeCellCountInt(1), s.cursor.y)
		expect(t, s.cursor.page_cell == sc2OCell(t, s, int(st[1]), 1), "s.cursor.page_cell == term51OCell(&s, st[1], 1)")
		expect(t, s.cursor.page_row == row, "s.cursor.page_row == row")
		expect(t, s.cursor.page_pin.node == node, "s.cursor.page_pin.node == node")
		expect(t, s.cursor.pending_wrap, "s.cursor.pending_wrap")
		expectEqual(t, style_id, s.cursor.style_id)
		expectEqual(t, link_id, s.cursor.hyperlink_id)
		sc2OExpectCursorConsistent(t, s)
	}
	expectEqual(t, uint(1), sc2OStyleRefs(&node.data, style_id))
	expectEqual(t, uint(1), sc2OLinkRefs(&node.data, link_id))
	expectEqualSlices(t, []bool{false, false, false}, sc2ODirtyRows(t, s, 3))
}

// Zig: test "term51: cursorRight: moves x, pin and cell together and nothing else"
func TestSc2_term51_cursorRight_moves_x_pin_and_cell_together_and_nothing_else(t *testing.T) {
	testStdCapacity(t)
	sc2OCursorMoveCase(t, 0, false, [][2]SizeCellCountInt{{1, 1}, {0, 1}, {8, 9}})
}

// Zig: test "term51: cursorLeft: moves x, pin and cell together and nothing else"
func TestSc2_term51_cursorLeft_moves_x_pin_and_cell_together_and_nothing_else(t *testing.T) {
	testStdCapacity(t)
	sc2OCursorMoveCase(t, 9, true, [][2]SizeCellCountInt{{1, 8}, {0, 8}, {8, 0}})
}

// Zig: test "term51: cursorRight: single column screen can move by zero only"
func TestSc2_term51_cursorRight_single_column_screen_can_move_by_zero_only(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 1, 1, 0)
	defer s.Deinit()
	s.CursorRight(0)
	s.CursorLeft(0)
	expectEqual(t, SizeCellCountInt(0), s.cursor.x)
	sc2OExpectCursorConsistent(t, s)
}

// Zig: test "term51: cursorUp: same page marks old and new row dirty"
func TestSc2_term51_cursorUp_same_page_marks_old_and_new_row_dirty(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 4, 0)
	defer s.Deinit()
	s.CursorAbsolute(3, 3)
	s.pages.ClearDirty()

	s.CursorUp(1)
	expectEqual(t, SizeCellCountInt(3), s.cursor.x)
	expectEqual(t, SizeCellCountInt(2), s.cursor.y)
	sc2OExpectCursorConsistent(t, s)
	expectEqualSlices(t, []bool{false, false, true, true}, sc2ODirtyRows(t, s, 4))

	s.pages.ClearDirty()
	s.CursorUp(2)
	expectEqual(t, SizeCellCountInt(0), s.cursor.y)
	sc2OExpectCursorConsistent(t, s)
	expectEqualSlices(t, []bool{true, false, true, false}, sc2ODirtyRows(t, s, 4))

	// Zero rows: nothing moves, nothing dirty.
	s.pages.ClearDirty()
	s.CursorUp(0)
	expectEqual(t, SizeCellCountInt(0), s.cursor.y)
	expectEqualSlices(t, []bool{false, false, false, false}, sc2ODirtyRows(t, s, 4))
	expect(t, !s.cursor.page_pin.node.data.dirty, "!s.cursor.page_pin.node.data.dirty")

	// The pending wrap flag is not touched.
	s.CursorAbsolute(3, 2)
	s.cursor.pending_wrap = true
	s.CursorUp(2)
	expect(t, s.cursor.pending_wrap, "s.cursor.pending_wrap")
}

// Zig: test "term51: cursorDown: same page marks old and new row dirty"
func TestSc2_term51_cursorDown_same_page_marks_old_and_new_row_dirty(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 4, 0)
	defer s.Deinit()
	s.CursorAbsolute(3, 0)
	s.pages.ClearDirty()

	s.CursorDown(1)
	expectEqual(t, SizeCellCountInt(3), s.cursor.x)
	expectEqual(t, SizeCellCountInt(1), s.cursor.y)
	sc2OExpectCursorConsistent(t, s)
	expectEqualSlices(t, []bool{true, true, false, false}, sc2ODirtyRows(t, s, 4))

	s.pages.ClearDirty()
	s.CursorDown(2)
	expectEqual(t, SizeCellCountInt(3), s.cursor.y)
	sc2OExpectCursorConsistent(t, s)
	expectEqualSlices(t, []bool{false, true, false, true}, sc2ODirtyRows(t, s, 4))

	// Zero rows at the bottom row.
	s.pages.ClearDirty()
	s.CursorDown(0)
	expectEqual(t, SizeCellCountInt(3), s.cursor.y)
	expectEqualSlices(t, []bool{false, false, false, false}, sc2ODirtyRows(t, s, 4))
	expect(t, !s.cursor.page_pin.node.data.dirty, "!s.cursor.page_pin.node.data.dirty")

	// The pending wrap flag is not touched.
	s.CursorAbsolute(3, 0)
	s.cursor.pending_wrap = true
	s.CursorDown(3)
	expect(t, s.cursor.pending_wrap, "s.cursor.pending_wrap")
	expectEqual(t, SizeCellCountInt(3), s.cursor.y)
}

// Zig: test "term51: cursorDown: single row moves by zero only"
func TestSc2_term51_cursorDown_single_row_moves_by_zero_only(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 3, 1, 0)
	defer s.Deinit()
	s.CursorDown(0)
	s.CursorUp(0)
	expectEqual(t, SizeCellCountInt(0), s.cursor.y)
	sc2OExpectCursorConsistent(t, s)
}

// Zig: test "term51: cursorDown: style and hyperlink move to the next page"
func TestSc2_term51_cursorDown_style_and_hyperlink_move_to_the_next_page(t *testing.T) {
	testStdCapacity(t)
	s := sc2OTwoPages(t, 20, 3)
	defer s.Deinit()
	s.CursorAbsolute(4, 0)
	sc2OStyleAndLink(t, s)
	first := s.cursor.page_pin.node
	old_style_id := s.cursor.style_id
	old_link_id := s.cursor.hyperlink_id
	expect(t, first == s.pages.pages.first, "first == s.pages.pages.first.?")

	// Still on the first page: nothing moves between pages.
	s.CursorDown(1)
	expect(t, s.cursor.page_pin.node == first, "s.cursor.page_pin.node == first")
	expectEqual(t, old_style_id, s.cursor.style_id)
	expectEqual(t, old_link_id, s.cursor.hyperlink_id)
	expectEqual(t, uint(1), sc2OStyleRefs(&first.data, old_style_id))

	s.pages.ClearDirty()
	s.CursorDown(1)
	expectEqual(t, SizeCellCountInt(4), s.cursor.x)
	expectEqual(t, SizeCellCountInt(2), s.cursor.y)
	expect(t, s.cursor.page_pin.node == s.pages.pages.last, "s.cursor.page_pin.node == s.pages.pages.last.?")
	sc2OExpectMovedFrom(t, s, first, old_style_id, old_link_id)
	sc2OExpectCursorConsistent(t, s)
	// Both the old row and the new row are dirty.
	expect(t, sc2ORow(t, s, 1).dirty, "term51ORow(&s, 1).dirty")
	expect(t, sc2ORow(t, s, 2).dirty, "term51ORow(&s, 2).dirty")
	expect(t, !sc2ORow(t, s, 0).dirty, "!term51ORow(&s, 0).dirty")

	// Printing uses the migrated style and link.
	sc2Write(t, s, "q")
	cell := sc2OCell(t, s, 4, 2)
	expectEqual(t, s.cursor.style_id, cell.style_id)
	expectEqual(t, s.cursor.hyperlink_id, zigUnwrap(t, s.cursor.page_pin.node.data.LookupHyperlink(cell)))
}

// Zig: test "term51: cursorUp: style and hyperlink move to the previous page"
func TestSc2_term51_cursorUp_style_and_hyperlink_move_to_the_previous_page(t *testing.T) {
	testStdCapacity(t)
	s := sc2OTwoPages(t, 20, 3)
	defer s.Deinit()
	s.CursorAbsolute(4, 2)
	sc2OStyleAndLink(t, s)
	last := s.cursor.page_pin.node
	expect(t, last == s.pages.pages.last, "last == s.pages.pages.last.?")
	old_style_id := s.cursor.style_id
	old_link_id := s.cursor.hyperlink_id

	s.pages.ClearDirty()
	s.CursorUp(1)
	expectEqual(t, SizeCellCountInt(4), s.cursor.x)
	expectEqual(t, SizeCellCountInt(1), s.cursor.y)
	expect(t, s.cursor.page_pin.node == s.pages.pages.first, "s.cursor.page_pin.node == s.pages.pages.first.?")
	sc2OExpectMovedFrom(t, s, last, old_style_id, old_link_id)
	sc2OExpectCursorConsistent(t, s)
	expect(t, sc2ORow(t, s, 1).dirty, "term51ORow(&s, 1).dirty")
	expect(t, sc2ORow(t, s, 2).dirty, "term51ORow(&s, 2).dirty")
	expect(t, !sc2ORow(t, s, 0).dirty, "!term51ORow(&s, 0).dirty")

	// Two rows up from the last page lands on the first row of the active
	// area in the first page.
	s.CursorDown(1)
	expect(t, s.cursor.page_pin.node == s.pages.pages.last, "s.cursor.page_pin.node == s.pages.pages.last.?")
	sc2OExpectCursorConsistent(t, s)
	id_on_last := s.cursor.style_id
	s.CursorUp(2)
	expectEqual(t, SizeCellCountInt(0), s.cursor.y)
	expect(t, s.cursor.page_pin.node == s.pages.pages.first, "s.cursor.page_pin.node == s.pages.pages.first.?")
	sc2OExpectCursorConsistent(t, s)
	expectEqual(t, uint(1), sc2OStyleRefs(&s.cursor.page_pin.node.data, s.cursor.style_id))
	expectEqual(t, uint(0), sc2OStyleRefs(&last.data, id_on_last))
}

// Zig: test "term51: cursorUp: implicit hyperlink gets a new implicit id on another page"
func TestSc2_term51_cursorUp_implicit_hyperlink_gets_a_new_implicit_id_on_another_page(t *testing.T) {
	testStdCapacity(t)
	s := sc2OTwoPages(t, 20, 3)
	defer s.Deinit()
	sc2StartHyperlink(t, s, "https://example.com/")
	expectEqual(t, SizeOffsetInt(1), s.cursor.hyperlink_implicit_id)
	expectEqual(t, SizeOffsetInt(0), s.cursor.hyperlink.id.Implicit)
	last := s.cursor.page_pin.node
	old_link_id := s.cursor.hyperlink_id

	s.CursorUp(1)
	expect(t, s.cursor.page_pin.node != last, "s.cursor.page_pin.node != last")
	expectEqual(t, uint(0), sc2OLinkRefs(&last.data, old_link_id))
	expectEqual(t, uint(1), sc2OLinkRefs(&s.cursor.page_pin.node.data, s.cursor.hyperlink_id))
	// The re-added link is a different link (counter advanced) than the
	// one text was written with before the move.
	expectEqual(t, SizeOffsetInt(2), s.cursor.hyperlink_implicit_id)
	expectEqual(t, SizeOffsetInt(1), s.cursor.hyperlink.id.Implicit)
	expectEqualStrings(t, "https://example.com/", string(s.cursor.hyperlink.uri))
}

// Zig: test "term51: cursorUp and cursorDown: styles skip over middle pages"
func TestSc2_term51_cursorUp_and_cursorDown_styles_skip_over_middle_pages(t *testing.T) {
	testStdCapacity(t)
	s := sc2OMultiPage(t, 0)
	defer s.Deinit()
	s.CursorAbsolute(5, 59)
	last := s.cursor.page_pin.node
	expect(t, last == s.pages.pages.last, "last == s.pages.pages.last.?")
	sc2OStyleAndLink(t, s)
	old_style_id := s.cursor.style_id
	old_link_id := s.cursor.hyperlink_id

	s.CursorUp(59)
	expectEqual(t, SizeCellCountInt(5), s.cursor.x)
	expectEqual(t, SizeCellCountInt(0), s.cursor.y)
	expect(t, s.cursor.page_pin.node == s.pages.pages.first, "s.cursor.page_pin.node == s.pages.pages.first.?")
	sc2OExpectMovedFrom(t, s, last, old_style_id, old_link_id)
	sc2OExpectCursorConsistent(t, s)

	// The middle pages never saw the style.
	for it := s.pages.pages.first.next; it != nil; it = it.next {
		if it == last {
			break
		}
		expectEqual(t, uint(0), it.data.styles.Count())
		expectEqual(t, uint(0), it.data.hyperlink_set.Count())
	}

	first := s.cursor.page_pin.node
	style_id2 := s.cursor.style_id
	link_id2 := s.cursor.hyperlink_id
	s.CursorDown(59)
	expectEqual(t, SizeCellCountInt(59), s.cursor.y)
	expect(t, s.cursor.page_pin.node == last, "s.cursor.page_pin.node == last")
	sc2OExpectMovedFrom(t, s, first, style_id2, link_id2)
	sc2OExpectCursorConsistent(t, s)

	// One page at a time too.
	s.CursorUp(1)
	s.CursorUp(1)
	s.CursorDown(2)
	expectEqual(t, SizeCellCountInt(59), s.cursor.y)
	sc2OExpectCursorConsistent(t, s)
	s.AssertIntegrity()
}

// Zig: test "term51: cursorUp: target page with a full style set is replaced by a bigger one"
func TestSc2_term51_cursorUp_target_page_with_a_full_style_set_is_replaced_by_a_bigger_one(t *testing.T) {
	testStdCapacity(t)
	s := sc2OTwoPages(t, 20, 3)
	defer s.Deinit()
	first := s.pages.pages.first
	old_styles_cap := first.data.capacity.styles

	// Fill the style set of the first page.
	func() {
		old := &first.data
		old.PauseIntegrityChecks(true)
		defer old.PauseIntegrityChecks(false)
		for n := uint32(1); ; n++ {
			_, err := old.styles.Add(sc2RgbBg(n))
			if err != nil {
				break
			}
		}
	}()

	zigTry(t, s.SetAttribute(sc2Bold()))
	s.CursorUp(1)

	// The target page was replaced by a copy and the cursor follows it.
	expect(t, s.cursor.page_pin.node != first, "s.cursor.page_pin.node != first")
	expect(t, s.cursor.page_pin.node == s.pages.pages.first, "s.cursor.page_pin.node == s.pages.pages.first.?")
	expectEqual(t, SizeCellCountInt(1), s.cursor.y)
	expect(t, s.cursor.style.flags.bold, "s.cursor.style.flags.bold")
	page := &s.cursor.page_pin.node.data
	expect(t, page.capacity.styles >= old_styles_cap, "page.capacity.styles >= old_styles_cap")
	expect(t, page.styles.Get(s.cursor.style_id).flags.bold, "page.styles.get(page.memory, s.cursor.style_id).flags.bold")
	expectEqual(t, uint(1), sc2OStyleRefs(page, s.cursor.style_id))
	sc2OExpectCursorConsistent(t, s)
	s.AssertIntegrity()
}

// Zig: test "term51: cursorRowUp: returns the row above without moving the cursor"
func TestSc2_term51_cursorRowUp_returns_the_row_above_without_moving_the_cursor(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 4, 0)
	defer s.Deinit()
	s.CursorAbsolute(3, 3)
	s.pages.ClearDirty()
	before := s.cursor

	expect(t, s.CursorRowUp(0) == s.cursor.page_row, "s.cursorRowUp(0) == s.cursor.page_row")
	expect(t, s.CursorRowUp(1) == sc2ORow(t, s, 2), "s.cursorRowUp(1) == term51ORow(&s, 2)")
	expect(t, s.CursorRowUp(2) == sc2ORow(t, s, 1), "s.cursorRowUp(2) == term51ORow(&s, 1)")
	expect(t, s.CursorRowUp(3) == sc2ORow(t, s, 0), "s.cursorRowUp(3) == term51ORow(&s, 0)")

	// Writing through the pointer changes that row.
	s.CursorRowUp(2).wrap = true
	expect(t, sc2ORow(t, s, 1).wrap, "term51ORow(&s, 1).wrap")
	expect(t, !sc2ORow(t, s, 2).wrap, "!term51ORow(&s, 2).wrap")

	expect(t, before == s.cursor, "std.meta.eql(before, s.cursor)")
	expectEqualSlices(t, []bool{false, false, false, false}, sc2ODirtyRows(t, s, 4))
}

// Zig: test "term51: cursorRowUp: across page boundaries"
func TestSc2_term51_cursorRowUp_across_page_boundaries(t *testing.T) {
	testStdCapacity(t)
	s := sc2OTwoPages(t, 20, 3)
	defer s.Deinit()
	first := &s.pages.pages.first.data
	before := s.cursor

	row1 := s.CursorRowUp(1)
	expect(t, row1 == sc2ORow(t, s, 1), "row1 == term51ORow(&s, 1)")
	// Zig: addr >= @intFromPtr(first.memory.ptr) and addr < ... + first.memory.len,
	// as an element of the page's rows.
	expect(t, sc2PageHasRow(first, row1), "row1 is in the first page's memory")
	expect(t, s.CursorRowUp(2) == sc2ORow(t, s, 0), "s.cursorRowUp(2) == term51ORow(&s, 0)")
	expect(t, before == s.cursor, "std.meta.eql(before, s.cursor)")

	m := sc2OMultiPage(t, 0)
	defer m.Deinit()
	m.CursorAbsolute(0, 59)
	expect(t, m.CursorRowUp(59) == sc2ORow(t, m, 0), "m.cursorRowUp(59) == term51ORow(&m, 0)")
	expect(t, m.CursorRowUp(30) == sc2ORow(t, m, 29), "m.cursorRowUp(30) == term51ORow(&m, 29)")
	expectEqual(t, SizeCellCountInt(59), m.cursor.y)
}

// Zig: test "term51: cursorHorizontalAbsolute: sets x, pin and cell only"
func TestSc2_term51_cursorHorizontalAbsolute_sets_x_pin_and_cell_only(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	defer s.Deinit()
	s.CursorAbsolute(4, 1)
	sc2OStyleAndLink(t, s)
	node := s.cursor.page_pin.node
	style_id := s.cursor.style_id
	link_id := s.cursor.hyperlink_id
	row := s.cursor.page_row
	s.cursor.pending_wrap = true
	s.pages.ClearDirty()

	// Left, right, same, first and last column.
	for _, x := range []SizeCellCountInt{0, 9, 9, 3, 3, 0, 9} {
		s.CursorHorizontalAbsolute(x)
		expectEqual(t, x, s.cursor.x)
		expectEqual(t, x, s.cursor.page_pin.x)
		expectEqual(t, SizeCellCountInt(1), s.cursor.y)
		expect(t, s.cursor.page_cell == sc2OCell(t, s, int(x), 1), "s.cursor.page_cell == term51OCell(&s, x, 1)")
		expect(t, s.cursor.page_row == row, "s.cursor.page_row == row")
		expect(t, s.cursor.page_pin.node == node, "s.cursor.page_pin.node == node")
		expect(t, s.cursor.pending_wrap, "s.cursor.pending_wrap")
		expectEqual(t, style_id, s.cursor.style_id)
		expectEqual(t, link_id, s.cursor.hyperlink_id)
		sc2OExpectCursorConsistent(t, s)
	}
	expectEqual(t, uint(1), sc2OStyleRefs(&node.data, style_id))
	expectEqual(t, uint(1), sc2OLinkRefs(&node.data, link_id))
	expectEqualSlices(t, []bool{false, false, false}, sc2ODirtyRows(t, s, 3))
}

// Zig: test "term51: cursorAbsolute: up, down, same row and same position"
func TestSc2_term51_cursorAbsolute_up_down_same_row_and_same_position(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 4, 0)
	defer s.Deinit()
	s.cursor.pending_wrap = true

	// Down (y > cursor.y)
	s.pages.ClearDirty()
	s.CursorAbsolute(7, 2)
	expectEqual(t, SizeCellCountInt(7), s.cursor.x)
	expectEqual(t, SizeCellCountInt(2), s.cursor.y)
	sc2OExpectCursorConsistent(t, s)
	expectEqualSlices(t, []bool{true, false, true, false}, sc2ODirtyRows(t, s, 4))

	// Up (y < cursor.y)
	s.pages.ClearDirty()
	s.CursorAbsolute(1, 1)
	expectEqual(t, SizeCellCountInt(1), s.cursor.x)
	expectEqual(t, SizeCellCountInt(1), s.cursor.y)
	sc2OExpectCursorConsistent(t, s)
	expectEqualSlices(t, []bool{false, true, true, false}, sc2ODirtyRows(t, s, 4))

	// Same row, other column: the pin differs so the row is marked.
	s.pages.ClearDirty()
	s.CursorAbsolute(5, 1)
	expectEqual(t, SizeCellCountInt(5), s.cursor.x)
	expectEqual(t, SizeCellCountInt(1), s.cursor.y)
	sc2OExpectCursorConsistent(t, s)
	expectEqualSlices(t, []bool{false, true, false, false}, sc2ODirtyRows(t, s, 4))

	// Same position: equal pin, nothing is marked.
	s.pages.ClearDirty()
	s.CursorAbsolute(5, 1)
	expectEqualSlices(t, []bool{false, false, false, false}, sc2ODirtyRows(t, s, 4))
	expect(t, !s.cursor.page_pin.node.data.dirty, "!s.cursor.page_pin.node.data.dirty")

	// Corners
	s.CursorAbsolute(9, 3)
	sc2OExpectCursorConsistent(t, s)
	s.CursorAbsolute(0, 0)
	sc2OExpectCursorConsistent(t, s)
	expectEqual(t, SizeCellCountInt(0), s.cursor.page_pin.y)

	// The pending wrap flag is never touched.
	expect(t, s.cursor.pending_wrap, "s.cursor.pending_wrap")
}

// Zig: test "term51: cursorAbsolute: style and hyperlink follow across pages"
func TestSc2_term51_cursorAbsolute_style_and_hyperlink_follow_across_pages(t *testing.T) {
	testStdCapacity(t)
	s := sc2OTwoPages(t, 20, 3)
	defer s.Deinit()
	s.CursorAbsolute(3, 1)
	sc2OStyleAndLink(t, s)
	first := s.cursor.page_pin.node
	old_style_id := s.cursor.style_id
	old_link_id := s.cursor.hyperlink_id

	// Down to the second page, to a different column.
	s.CursorAbsolute(8, 2)
	expectEqual(t, SizeCellCountInt(8), s.cursor.x)
	expectEqual(t, SizeCellCountInt(2), s.cursor.y)
	expect(t, s.cursor.page_pin.node == s.pages.pages.last, "s.cursor.page_pin.node == s.pages.pages.last.?")
	sc2OExpectMovedFrom(t, s, first, old_style_id, old_link_id)
	sc2OExpectCursorConsistent(t, s)

	// And back up to the first.
	last := s.cursor.page_pin.node
	style_id2 := s.cursor.style_id
	link_id2 := s.cursor.hyperlink_id
	s.CursorAbsolute(0, 0)
	expect(t, s.cursor.page_pin.node == s.pages.pages.first, "s.cursor.page_pin.node == s.pages.pages.first.?")
	sc2OExpectMovedFrom(t, s, last, style_id2, link_id2)
	sc2OExpectCursorConsistent(t, s)

	// Same page, different row: no accounting changes.
	style_id3 := s.cursor.style_id
	s.CursorAbsolute(6, 1)
	expectEqual(t, style_id3, s.cursor.style_id)
	expectEqual(t, uint(1), sc2OStyleRefs(&s.cursor.page_pin.node.data, style_id3))
}

// Zig: test "term51: cursorAbsolute: across several pages in one move"
func TestSc2_term51_cursorAbsolute_across_several_pages_in_one_move(t *testing.T) {
	testStdCapacity(t)
	s := sc2OMultiPage(t, 0)
	defer s.Deinit()
	sc2OStyleAndLink(t, s)
	first := s.cursor.page_pin.node
	style_id := s.cursor.style_id
	link_id := s.cursor.hyperlink_id

	s.CursorAbsolute(1999, 59)
	expectEqual(t, SizeCellCountInt(1999), s.cursor.x)
	expectEqual(t, SizeCellCountInt(59), s.cursor.y)
	expect(t, s.cursor.page_pin.node == s.pages.pages.last, "s.cursor.page_pin.node == s.pages.pages.last.?")
	sc2OExpectMovedFrom(t, s, first, style_id, link_id)
	sc2OExpectCursorConsistent(t, s)

	// Every row's pin lines up with its active coordinate.
	for _, y := range []SizeCellCountInt{0, 1, 22, 23, 24, 40, 58, 59} {
		s.CursorAbsolute(y, y)
		expectEqual(t, y, s.cursor.y)
		sc2OExpectCursorConsistent(t, s)
	}
}

// Zig: test "term51: cursorReload: re-derives x, y and the caches from the tracked pin"
func TestSc2_term51_cursorReload_re_derives_x_y_and_the_caches_from_the_tracked_pin(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 4, 0)
	defer s.Deinit()
	s.cursor.pending_wrap = true
	s.cursor.cursor_style = ScreenCursorStyleBar

	// Move only the pin, as a pagelist operation could.
	s.cursor.page_pin.x = 4
	s.cursor.page_pin.y = 2
	s.CursorReload()
	expectEqual(t, SizeCellCountInt(4), s.cursor.x)
	expectEqual(t, SizeCellCountInt(2), s.cursor.y)
	sc2OExpectCursorConsistent(t, s)
	// Everything else is untouched.
	expect(t, s.cursor.pending_wrap, "s.cursor.pending_wrap")
	expectEqual(t, CursorStyleBar, s.cursor.cursor_style)

	// Corner
	s.cursor.page_pin.x = 9
	s.cursor.page_pin.y = 3
	s.CursorReload()
	expectEqual(t, SizeCellCountInt(9), s.cursor.x)
	expectEqual(t, SizeCellCountInt(3), s.cursor.y)
	sc2OExpectCursorConsistent(t, s)

	// Idempotent
	before := s.cursor
	s.CursorReload()
	expect(t, before == s.cursor, "std.meta.eql(before, s.cursor)")
}

// Zig: test "term51: cursorReload: pin above the active area resets to the active top left"
func TestSc2_term51_cursorReload_pin_above_the_active_area_resets_to_the_active_top_left(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 100)
	defer s.Deinit()
	sc2Write(t, s, "abcdef")
	expectEqual(t, SizeCellCountInt(6), s.cursor.x)

	// Growing the page list pushes the cursor's row into the history.
	_, err := s.pages.Grow()
	zigTry(t, err)
	expect(t, !s.pages.PointFromPin(PointTagActive, *s.cursor.page_pin).Valid,
		"s.pages.pointFromPin(.active, s.cursor.page_pin.*) == null")

	s.cursor.pending_wrap = true
	s.CursorReload()
	expectEqual(t, SizeCellCountInt(0), s.cursor.x)
	expectEqual(t, SizeCellCountInt(0), s.cursor.y)
	expectEqual(t, SizeCellCountInt(0), s.cursor.page_pin.x)
	expectEqual(t, SizeCellCountInt(1), s.cursor.page_pin.y)
	expect(t, s.cursor.page_pin.Eql(sc2Pin(t, s, sc2Active(0, 0))),
		"s.cursor.page_pin.eql(s.pages.pin(.{ .active = .{} }).?)")
	sc2OExpectCursorConsistent(t, s)
	expect(t, s.cursor.pending_wrap, "s.cursor.pending_wrap")
}

// Zig: test "term51: cursorReload: re-registers the cursor style on the page"
func TestSc2_term51_cursorReload_re_registers_the_cursor_style_on_the_page(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 100)
	defer s.Deinit()
	zigTry(t, s.SetAttribute(sc2Bold()))
	sc2Write(t, s, "abcdef")
	page := &s.cursor.page_pin.node.data
	// Six cells and the cursor.
	expectEqual(t, uint(7), sc2OStyleRefs(page, s.cursor.style_id))

	// Same page, moved pin.
	s.cursor.page_pin.x = 2
	s.cursor.page_pin.y = 1
	s.CursorReload()
	expectEqual(t, SizeCellCountInt(2), s.cursor.x)
	expectEqual(t, SizeCellCountInt(1), s.cursor.y)
	expect(t, s.cursor.style_id != StyleDefaultId, "s.cursor.style_id != style.default_id")
	expect(t, s.cursor.style.flags.bold, "s.cursor.style.flags.bold")
	expectEqual(t, uint(7), sc2OStyleRefs(page, s.cursor.style_id))
	expectEqual(t, uint(1), page.styles.Count())

	// Growing moves the active area down one row: the pin (page row 1) is
	// now the active top row and keeps its column.
	_, err := s.pages.Grow()
	zigTry(t, err)
	s.CursorReload()
	expectEqual(t, SizeCellCountInt(2), s.cursor.x)
	expectEqual(t, SizeCellCountInt(0), s.cursor.y)
	expectEqual(t, uint(7), sc2OStyleRefs(page, s.cursor.style_id))
	sc2OExpectCursorConsistent(t, s)

	// Growing again pushes the pin into the history: back to the top-left
	// of the active area, style still referenced once by the cursor.
	_, err = s.pages.Grow()
	zigTry(t, err)
	expect(t, !s.pages.PointFromPin(PointTagActive, *s.cursor.page_pin).Valid,
		"s.pages.pointFromPin(.active, s.cursor.page_pin.*) == null")
	s.CursorReload()
	expectEqual(t, SizeCellCountInt(0), s.cursor.x)
	expectEqual(t, SizeCellCountInt(0), s.cursor.y)
	expectEqual(t, SizeCellCountInt(2), s.cursor.page_pin.y)
	expectEqual(t, uint(7), sc2OStyleRefs(page, s.cursor.style_id))
	sc2OExpectCursorConsistent(t, s)
}

// Zig: test "term51: cursorReload: default style is left alone"
func TestSc2_term51_cursorReload_default_style_is_left_alone(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	defer s.Deinit()
	s.cursor.page_pin.y = 1
	s.CursorReload()
	expectEqual(t, StyleDefaultId, s.cursor.style_id)
	expectEqual(t, uint(0), s.cursor.page_pin.node.data.styles.Count())
}

// sc2OScreenCell is term51OScreenCell: the cell at the screen point (x, y).
func sc2OScreenCell(t *testing.T, s *Screen, x int, y int) *PageCell {
	t.Helper()
	return sc2Pin(t, s, sc2Screen(x, y)).RowAndCell().cell
}

// sc2OFullPage is term51OFullPage: a screen whose single page is full (all of
// its row capacity is in use) and whose cursor is on the bottom row.
func sc2OFullPage(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) *Screen {
	t.Helper()
	s := sc2Init(t, cols, rows, 10)
	// Zig: errdefer s.deinit() (a failed helper ends the test).
	page := &s.pages.pages.first.data
	page.PauseIntegrityChecks(true)
	for page.size.rows < page.capacity.rows {
		zigTry(t, s.cursorDownOrScroll())
	}
	page.PauseIntegrityChecks(false)
	expectEqual(t, uint(1), sc2ONodeCount(s))
	expectEqual(t, page.capacity.rows, page.size.rows)
	expectEqual(t, rows-1, s.cursor.y)
	return s
}

// sc2OExpectBlank is term51OExpectBlank: expect that the cell is entirely
// blank.
func sc2OExpectBlank(t *testing.T, cell *PageCell) {
	t.Helper()
	expect(t, cell.IsZero(), "cell.isZero()")
}

// sc2OExpectBgRow is term51OExpectBgRow: expect that the whole row is the
// background colour cell for `rgb`.
func sc2OExpectBgRow(t *testing.T, s *Screen, y int, rgb [3]uint8) {
	t.Helper()
	for x := 0; x < int(s.pages.cols); x++ {
		cell := sc2OCell(t, s, x, y)
		expectEqual(t, PageCellContentTagBgColorRgb, cell.content_tag)
		expectEqual(t, rgb[0], cell.content.color_rgb.r)
		expectEqual(t, rgb[1], cell.content.color_rgb.g)
		expectEqual(t, rgb[2], cell.content.color_rgb.b)
	}
}

// sc2DirectColorBg is `.{ .direct_color_bg = .{ .r = r, .g = g, .b = b } }`.
func sc2DirectColorBg(r uint8, g uint8, b uint8) SGRAttribute {
	return SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{r: r, g: g, b: b}}
}

// Zig: test "term51: cursorDownScroll: without scrollback and a single row clears the row in place"
func TestSc2_term51_cursorDownScroll_without_scrollback_and_a_single_row_clears_the_row_in_place(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 1, 0)
	defer s.Deinit()
	expect(t, s.no_scrollback, "s.no_scrollback")
	zigTry(t, s.SetAttribute(sc2Bold()))
	sc2Write(t, s, "abc")
	page := &s.cursor.page_pin.node.data
	expectEqual(t, uint(4), sc2OStyleRefs(page, s.cursor.style_id))
	pin_ptr := s.cursor.page_pin
	pin_before := *s.cursor.page_pin
	row := s.cursor.page_row
	row.wrap = true
	s.cursor.pending_wrap = true
	s.pages.ClearDirty()

	zigTry(t, s.CursorDownScroll())

	// The cursor did not move at all.
	expect(t, s.cursor.page_pin == pin_ptr, "s.cursor.page_pin == pin_ptr")
	expect(t, s.cursor.page_pin.Eql(pin_before), "s.cursor.page_pin.eql(pin_before)")
	expectEqual(t, SizeCellCountInt(3), s.cursor.x)
	expectEqual(t, SizeCellCountInt(0), s.cursor.y)
	sc2OExpectCursorConsistent(t, s)
	expect(t, s.cursor.pending_wrap, "s.cursor.pending_wrap")
	expectEqual(t, uint(1), s.pages.total_rows)

	// Cells are blank, their style references are gone, the row is dirty.
	for x := 0; x < 5; x++ {
		sc2OExpectBlank(t, sc2OCell(t, s, x, 0))
	}
	expectEqual(t, uint(1), sc2OStyleRefs(page, s.cursor.style_id))
	expect(t, !row.styled, "!row.styled")
	expect(t, row.dirty, "row.dirty")
	// Row attributes other than the managed-memory flags are kept.
	expect(t, row.wrap, "row.wrap")
}

// Zig: test "term51: cursorDownScroll: without scrollback and a single row fills with the background"
func TestSc2_term51_cursorDownScroll_without_scrollback_and_a_single_row_fills_with_the_background(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 1, 0)
	defer s.Deinit()
	zigTry(t, s.SetAttribute(sc2DirectColorBg(10, 20, 30)))
	sc2Write(t, s, "abc")
	zigTry(t, s.CursorDownScroll())
	expectEqual(t, SizeCellCountInt(3), s.cursor.x)
	sc2OExpectBgRow(t, s, 0, [3]uint8{10, 20, 30})
}

// Zig: test "term51: cursorDownScroll: without scrollback shifts all rows up and keeps the cursor"
func TestSc2_term51_cursorDownScroll_without_scrollback_shifts_all_rows_up_and_keeps_the_cursor(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 4, 0)
	defer s.Deinit()
	zigTry(t, s.SetAttribute(sc2Bold()))
	sc2Write(t, s, "r0\nr1\nr2\nr3")
	expectEqual(t, SizeCellCountInt(2), s.cursor.x)
	expectEqual(t, SizeCellCountInt(3), s.cursor.y)
	page := &s.cursor.page_pin.node.data
	// Eight styled cells and the cursor.
	expectEqual(t, uint(9), sc2OStyleRefs(page, s.cursor.style_id))
	pin_ptr := s.cursor.page_pin
	pin_before := *s.cursor.page_pin
	s.cursor.pending_wrap = true
	s.pages.ClearDirty()

	zigTry(t, s.CursorDownScroll())

	// The pin was restored to where it was.
	expect(t, s.cursor.page_pin == pin_ptr, "s.cursor.page_pin == pin_ptr")
	expect(t, s.cursor.page_pin.Eql(pin_before), "s.cursor.page_pin.eql(pin_before)")
	expectEqual(t, SizeCellCountInt(2), s.cursor.x)
	expectEqual(t, SizeCellCountInt(3), s.cursor.y)
	sc2OExpectCursorConsistent(t, s)
	expect(t, s.cursor.pending_wrap, "s.cursor.pending_wrap")
	expectEqual(t, uint(4), s.pages.total_rows)

	// r0 is gone, everything moved up, the last row is blank.
	expectEqual(t, uint32('r'), sc2OCp(t, s, 0, 0))
	expectEqual(t, uint32('1'), sc2OCp(t, s, 1, 0))
	expectEqual(t, uint32('2'), sc2OCp(t, s, 1, 1))
	expectEqual(t, uint32('3'), sc2OCp(t, s, 1, 2))
	for x := 0; x < 5; x++ {
		sc2OExpectBlank(t, sc2OCell(t, s, x, 3))
	}
	// The erased row's styled cells were released.
	expectEqual(t, uint(7), sc2OStyleRefs(page, s.cursor.style_id))
	expect(t, !sc2ORow(t, s, 3).styled, "!term51ORow(&s, 3).styled")
	expect(t, sc2ORow(t, s, 2).styled, "term51ORow(&s, 2).styled")

	// The whole page is flagged dirty.
	expect(t, page.dirty, "page.dirty")
	for y := 0; y < 4; y++ {
		expect(t, s.pages.IsDirty(sc2Active(0, y)), "s.pages.isDirty(.{ .active = .{ .y = y } })")
	}
}

// Zig: test "term51: cursorDownScroll: without scrollback the new bottom row inherits the old top row's wrap flags"
func TestSc2_term51_cursorDownScroll_without_scrollback_the_new_bottom_row_inherits_the_old_top_row_s_wrap_flags(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 3, 0)
	defer s.Deinit()
	s.CursorAbsolute(0, 2)
	sc2ORow(t, s, 0).wrap = true
	sc2ORow(t, s, 0).wrap_continuation = true
	sc2ORow(t, s, 1).wrap = false
	zigTry(t, s.CursorDownScroll())

	// eraseRow rotates the row structs and only clears cells, so the flags
	// that are not cleaned by clearCells end up on the new last row.
	expect(t, sc2ORow(t, s, 2).wrap, "term51ORow(&s, 2).wrap")
	expect(t, sc2ORow(t, s, 2).wrap_continuation, "term51ORow(&s, 2).wrap_continuation")
	expect(t, !sc2ORow(t, s, 0).wrap, "!term51ORow(&s, 0).wrap")
	expect(t, !sc2ORow(t, s, 1).wrap, "!term51ORow(&s, 1).wrap")
}

// Zig: test "term51: cursorDownScroll: without scrollback fills the new row with the background"
func TestSc2_term51_cursorDownScroll_without_scrollback_fills_the_new_row_with_the_background(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 3, 0)
	defer s.Deinit()
	sc2Write(t, s, "r0\nr1\nr2")
	zigTry(t, s.SetAttribute(sc2DirectColorBg(1, 2, 3)))
	zigTry(t, s.CursorDownScroll())

	sc2OExpectBgRow(t, s, 2, [3]uint8{1, 2, 3})
	// Other rows are not affected.
	expectEqual(t, uint32('1'), sc2OCp(t, s, 1, 0))
	expectEqual(t, uint32('2'), sc2OCp(t, s, 1, 1))
	sc2OExpectCursorConsistent(t, s)

	// A style without a background does not fill.
	s2 := sc2Init(t, 5, 3, 0)
	defer s2.Deinit()
	s2.CursorAbsolute(0, 2)
	zigTry(t, s2.SetAttribute(sc2Bold()))
	zigTry(t, s2.CursorDownScroll())
	for x := 0; x < 5; x++ {
		sc2OExpectBlank(t, sc2OCell(t, s2, x, 2))
	}
}

// Zig: test "term51: cursorDownScroll: without scrollback across several pages"
func TestSc2_term51_cursorDownScroll_without_scrollback_across_several_pages(t *testing.T) {
	testStdCapacity(t)
	s := sc2OMultiPage(t, 0)
	defer s.Deinit()
	expect(t, s.no_scrollback, "s.no_scrollback")
	for y := 0; y < 60; y++ {
		sc2OSetCp(t, s, 0, y, uint32(0x100+y))
	}
	s.CursorAbsolute(7, 59)
	pin_ptr := s.cursor.page_pin
	pin_before := *s.cursor.page_pin
	total := s.pages.total_rows

	zigTry(t, s.CursorDownScroll())
	expect(t, s.cursor.page_pin == pin_ptr, "s.cursor.page_pin == pin_ptr")
	expect(t, s.cursor.page_pin.Eql(pin_before), "s.cursor.page_pin.eql(pin_before)")
	expectEqual(t, SizeCellCountInt(7), s.cursor.x)
	expectEqual(t, SizeCellCountInt(59), s.cursor.y)
	expectEqual(t, total, s.pages.total_rows)
	sc2OExpectCursorConsistent(t, s)
	for y := 0; y < 59; y++ {
		expectEqual(t, uint32(0x100+y+1), sc2OCp(t, s, 0, y))
	}
	expectEqual(t, uint32(0), sc2OCp(t, s, 0, 59))
	s.AssertIntegrity()
}

// Zig: test "term51: cursorDownScroll: with scrollback keeps history, adds a blank row"
func TestSc2_term51_cursorDownScroll_with_scrollback_keeps_history_adds_a_blank_row(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 3, 100)
	defer s.Deinit()
	expect(t, !s.no_scrollback, "!s.no_scrollback")
	sc2Write(t, s, "r0\nr1\nr2")
	expectEqual(t, uint(3), s.pages.total_rows)
	s.cursor.pending_wrap = true
	s.pages.ClearDirty()
	old_row := s.cursor.page_row

	zigTry(t, s.CursorDownScroll())

	expectEqual(t, uint(4), s.pages.total_rows)
	expectEqual(t, SizeCellCountInt(2), s.cursor.x)
	expectEqual(t, SizeCellCountInt(2), s.cursor.y)
	expect(t, s.cursor.pending_wrap, "s.cursor.pending_wrap")
	sc2OExpectCursorConsistent(t, s)
	expect(t, s.cursor.page_row != old_row, "s.cursor.page_row != old_row")
	expectEqual(t, uint32('0'), sc2OScreenCell(t, s, 1, 0).Codepoint())
	expectEqual(t, uint32('1'), sc2OScreenCell(t, s, 1, 1).Codepoint())
	expectEqual(t, uint32('2'), sc2OScreenCell(t, s, 1, 2).Codepoint())
	for x := 0; x < 5; x++ {
		sc2OExpectBlank(t, sc2OScreenCell(t, s, x, 3))
	}

	// The old and the new cursor row are dirty, the rest is not.
	expectEqualSlices(t, []bool{false, true, true}, sc2ODirtyRows(t, s, 3))
	expect(t, !s.cursor.page_pin.node.data.dirty, "!s.cursor.page_pin.node.data.dirty")

	// Scrolling again and again keeps adding history.
	for i := 0; i < 5; i++ {
		zigTry(t, s.CursorDownScroll())
	}
	expectEqual(t, uint(9), s.pages.total_rows)
	expectEqual(t, SizeCellCountInt(2), s.cursor.y)
	sc2OExpectCursorConsistent(t, s)
}

// Zig: test "term51: cursorDownScroll: with scrollback fills the new row only for a background colour"
func TestSc2_term51_cursorDownScroll_with_scrollback_fills_the_new_row_only_for_a_background_colour(t *testing.T) {
	testStdCapacity(t)

	// RGB background, cursor not in the first column.
	{
		s := sc2Init(t, 5, 3, 100)
		defer s.Deinit()
		sc2Write(t, s, "r0\nr1\nr2")
		zigTry(t, s.SetAttribute(sc2DirectColorBg(9, 8, 7)))
		zigTry(t, s.CursorDownScroll())
		expectEqual(t, SizeCellCountInt(2), s.cursor.x)
		sc2OExpectBgRow(t, s, 2, [3]uint8{9, 8, 7})
		// The previous row is plain text.
		expectEqual(t, uint32('2'), sc2OCp(t, s, 1, 1))
		expect(t, !sc2ORow(t, s, 1).styled, "!term51ORow(&s, 1).styled")
	}

	// Palette background
	{
		s := sc2Init(t, 5, 2, 100)
		defer s.Deinit()
		zigTry(t, s.SetAttribute(SGRAttribute{Tag: SGRAttributeTag256Bg, X256Bg: 7}))
		s.CursorAbsolute(4, 1)
		zigTry(t, s.CursorDownScroll())
		for x := 0; x < 5; x++ {
			cell := sc2OCell(t, s, x, 1)
			expectEqual(t, PageCellContentTagBgColorPalette, cell.content_tag)
			expectEqual(t, uint8(7), cell.content.color_palette)
		}
	}

	// A style with no background leaves the new row blank.
	{
		s := sc2Init(t, 5, 2, 100)
		defer s.Deinit()
		zigTry(t, s.SetAttribute(SGRAttribute{Tag: SGRAttributeTagItalic}))
		s.CursorAbsolute(0, 1)
		zigTry(t, s.CursorDownScroll())
		for x := 0; x < 5; x++ {
			sc2OExpectBlank(t, sc2OCell(t, s, x, 1))
		}
	}
}

// Zig: test "term51: cursorDownScroll: with scrollback and a full page moves the cursor to a new page"
func TestSc2_term51_cursorDownScroll_with_scrollback_and_a_full_page_moves_the_cursor_to_a_new_page(t *testing.T) {
	testStdCapacity(t)
	s := sc2OFullPage(t, 200, 3)
	defer s.Deinit()
	s.CursorAbsolute(9, 2)
	sc2OStyleAndLink(t, s)
	first := s.cursor.page_pin.node
	old_style_id := s.cursor.style_id
	old_link_id := s.cursor.hyperlink_id
	s.cursor.pending_wrap = true
	s.pages.ClearDirty()

	zigTry(t, s.CursorDownScroll())

	expectEqual(t, uint(2), sc2ONodeCount(s))
	last := s.pages.pages.last
	expect(t, s.cursor.page_pin.node == last, "s.cursor.page_pin.node == last")
	expectEqual(t, SizeCellCountInt(1), last.data.size.rows)
	expectEqual(t, SizeCellCountInt(9), s.cursor.x)
	expectEqual(t, SizeCellCountInt(2), s.cursor.y)
	expectEqual(t, SizeCellCountInt(0), s.cursor.page_pin.y)
	expectEqual(t, SizeCellCountInt(9), s.cursor.page_pin.x)
	expect(t, s.cursor.pending_wrap, "s.cursor.pending_wrap")
	sc2OExpectMovedFrom(t, s, first, old_style_id, old_link_id)
	sc2OExpectCursorConsistent(t, s)
	expect(t, sc2ORow(t, s, 2).dirty, "term51ORow(&s, 2).dirty")
	expect(t, sc2ORow(t, s, 1).dirty, "term51ORow(&s, 1).dirty")
	expect(t, !sc2ORow(t, s, 0).dirty, "!term51ORow(&s, 0).dirty")
}

// Zig: test "term51: cursorDownScroll: pruning scrollback recycles the first page"
func TestSc2_term51_cursorDownScroll_pruning_scrollback_recycles_the_first_page(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 200, 3, 1)
	defer s.Deinit()
	expect(t, !s.no_scrollback, "!s.no_scrollback")

	// Fill the first page then start (and fill) the second page.
	p0 := &s.pages.pages.first.data
	p0.PauseIntegrityChecks(true)
	for s.pages.pages.last == s.pages.pages.first {
		zigTry(t, s.cursorDownOrScroll())
	}
	p0.PauseIntegrityChecks(false)
	second := s.pages.pages.last
	second.data.PauseIntegrityChecks(true)
	for second.data.size.rows < second.data.capacity.rows {
		zigTry(t, s.cursorDownOrScroll())
	}
	second.data.PauseIntegrityChecks(false)
	expectEqual(t, uint(2), sc2ONodeCount(s))
	expect(t, s.cursor.page_pin.node == second, "s.cursor.page_pin.node == second")
	expectEqual(t, uint64(0), s.pages.page_serial_min)

	s.CursorAbsolute(4, 2)
	sc2OStyleAndLink(t, s)
	old_style_id := s.cursor.style_id
	old_link_id := s.cursor.hyperlink_id
	rows_in_second := second.data.size.rows

	zigTry(t, s.CursorDownScroll())

	// The first page was pruned and its memory reused as the new last page.
	expectEqual(t, uint(2), sc2ONodeCount(s))
	expect(t, s.pages.pages.first == second, "s.pages.pages.first.? == second")
	expect(t, s.pages.pages.last != second, "s.pages.pages.last.? != second")
	expectEqual(t, uint64(1), s.pages.page_serial_min)
	expectEqual(t, uint(rows_in_second)+1, s.pages.total_rows)
	expect(t, s.cursor.page_pin.node == s.pages.pages.last, "s.cursor.page_pin.node == s.pages.pages.last.?")
	expectEqual(t, SizeCellCountInt(4), s.cursor.x)
	expectEqual(t, SizeCellCountInt(2), s.cursor.y)
	sc2OExpectMovedFrom(t, s, second, old_style_id, old_link_id)
	sc2OExpectCursorConsistent(t, s)
	s.AssertIntegrity()
}

// Zig: test "term51: cursorScrollAbove: cursor on the bottom row scrolls like cursorDownScroll"
func TestSc2_term51_cursorScrollAbove_cursor_on_the_bottom_row_scrolls_like_cursorDownScroll(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 3, 100)
	defer s.Deinit()
	sc2Write(t, s, "r0\nr1\nr2")
	s.pages.ClearDirty()

	zigTry(t, s.CursorScrollAbove())
	expectEqual(t, uint(4), s.pages.total_rows)
	expectEqual(t, SizeCellCountInt(2), s.cursor.x)
	expectEqual(t, SizeCellCountInt(2), s.cursor.y)
	sc2OExpectCursorConsistent(t, s)
	expectEqual(t, uint32('1'), sc2OCp(t, s, 1, 0))
	expectEqual(t, uint32('2'), sc2OCp(t, s, 1, 1))
	sc2OExpectBlank(t, sc2OCell(t, s, 1, 2))
	expectEqual(t, uint32('0'), sc2OScreenCell(t, s, 1, 0).Codepoint())
	expectEqualSlices(t, []bool{false, true, true}, sc2ODirtyRows(t, s, 3))
}

// Zig: test "term51: cursorScrollAbove: bottom row without scrollback and a single row"
func TestSc2_term51_cursorScrollAbove_bottom_row_without_scrollback_and_a_single_row(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 5, 3, 0)
	defer s.Deinit()
	sc2Write(t, s, "r0\nr1\nr2")
	zigTry(t, s.CursorScrollAbove())
	expectEqual(t, uint(3), s.pages.total_rows)
	expectEqual(t, uint32('1'), sc2OCp(t, s, 1, 0))
	expectEqual(t, uint32('2'), sc2OCp(t, s, 1, 1))
	sc2OExpectBlank(t, sc2OCell(t, s, 1, 2))
	sc2OExpectCursorConsistent(t, s)

	s1 := sc2Init(t, 5, 1, 0)
	defer s1.Deinit()
	sc2Write(t, s1, "ab")
	s1.pages.ClearDirty()
	zigTry(t, s1.CursorScrollAbove())
	sc2OExpectBlank(t, sc2OCell(t, s1, 0, 0))
	expect(t, sc2ORow(t, s1, 0).dirty, "term51ORow(&s1, 0).dirty")
	expectEqual(t, SizeCellCountInt(2), s1.cursor.x)
}

// Zig: test "term51: cursorScrollAbove: cursor above the bottom on the last page rotates in place"
func TestSc2_term51_cursorScrollAbove_cursor_above_the_bottom_on_the_last_page_rotates_in_place(t *testing.T) {
	testStdCapacity(t)

	// Cursor on the second to last row and on the top row.
	for _, cy := range []SizeCellCountInt{2, 0} {
		func() {
			s := sc2Init(t, 5, 4, 100)
			defer s.Deinit()
			sc2Write(t, s, "r0\nr1\nr2\nr3")
			s.CursorAbsolute(1, cy)
			s.pages.ClearDirty()
			pin_y := s.cursor.page_pin.y

			zigTry(t, s.CursorScrollAbove())

			// One more row, the cursor row pin moved down by one page row but
			// the active coordinates stayed.
			expectEqual(t, uint(5), s.pages.total_rows)
			expectEqual(t, SizeCellCountInt(1), s.cursor.x)
			expectEqual(t, cy, s.cursor.y)
			expectEqual(t, pin_y+1, s.cursor.page_pin.y)
			sc2OExpectCursorConsistent(t, s)

			// Rows at and above the cursor moved up by one (the top one goes to
			// the history), the cursor row is blank, rows below did not move.
			for y := 0; y < 4; y++ {
				var want uint32
				if y < int(cy) {
					want = uint32('0' + y + 1)
				} else if y == int(cy) {
					want = 0
				} else {
					want = uint32('0' + y)
				}
				expectEqual(t, want, sc2OCp(t, s, 1, y))
			}
			expectEqual(t, uint32('0'), sc2OScreenCell(t, s, 1, 0).Codepoint())
			// The whole page is dirty.
			expect(t, s.cursor.page_pin.node.data.dirty, "s.cursor.page_pin.node.data.dirty")
			for y := 0; y < 4; y++ {
				expect(t, s.pages.IsDirty(sc2Active(0, y)), "s.pages.isDirty(.{ .active = .{ .y = y } })")
			}
		}()
	}
}

// Zig: test "term51: cursorScrollAbove: fills the new row only for a background colour"
func TestSc2_term51_cursorScrollAbove_fills_the_new_row_only_for_a_background_colour(t *testing.T) {
	testStdCapacity(t)

	{
		s := sc2Init(t, 5, 4, 100)
		defer s.Deinit()
		sc2Write(t, s, "r0\nr1\nr2\nr3")
		s.CursorAbsolute(3, 1)
		zigTry(t, s.SetAttribute(sc2DirectColorBg(5, 6, 7)))
		zigTry(t, s.CursorScrollAbove())
		sc2OExpectBgRow(t, s, 1, [3]uint8{5, 6, 7})
		expectEqual(t, uint32('2'), sc2OCp(t, s, 1, 2))
	}
	{
		s := sc2Init(t, 5, 4, 100)
		defer s.Deinit()
		sc2Write(t, s, "r0\nr1\nr2\nr3")
		s.CursorAbsolute(3, 1)
		zigTry(t, s.SetAttribute(sc2Bold()))
		zigTry(t, s.CursorScrollAbove())
		for x := 0; x < 5; x++ {
			sc2OExpectBlank(t, sc2OCell(t, s, x, 1))
		}
	}
}

// ---------------------------------------------------------------------------
// Go-only panic tests: the assert / unreachable paths the Zig tests avoid.
// Each screen is built outside the closure, so a panic in the setup is a
// failure and only the call under test may panic.
// ---------------------------------------------------------------------------

// Go-only: panic test: cursorCellRight: x + n >= cols
func TestSc2_panic_cursorCellRight_beyond_the_last_column(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	sc2ExpectPanic(t, func() { s.CursorCellRight(10) })
}

// Go-only: panic test: cursorCellLeft: x < n
func TestSc2_panic_cursorCellLeft_before_the_first_column(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	s.CursorAbsolute(2, 1)
	sc2ExpectPanic(t, func() { s.CursorCellLeft(3) })
}

// Go-only: panic test: cursorCellEndOfPrev: y == 0
func TestSc2_panic_cursorCellEndOfPrev_on_the_top_row(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	sc2ExpectPanic(t, func() { s.CursorCellEndOfPrev() })
}

// Go-only: panic test: cursorRight: x + n >= cols
func TestSc2_panic_cursorRight_beyond_the_last_column(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	sc2ExpectPanic(t, func() { s.CursorRight(10) })
}

// Go-only: panic test: cursorLeft: x < n
func TestSc2_panic_cursorLeft_before_the_first_column(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	sc2ExpectPanic(t, func() { s.CursorLeft(1) })
}

// Go-only: panic test: cursorUp: y < n
func TestSc2_panic_cursorUp_above_the_top_row(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	sc2ExpectPanic(t, func() { s.CursorUp(1) })
}

// Go-only: panic test: cursorRowUp: y < n
func TestSc2_panic_cursorRowUp_above_the_top_row(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	sc2ExpectPanic(t, func() { s.CursorRowUp(1) })
}

// Go-only: panic test: cursorDown: y + n >= rows
func TestSc2_panic_cursorDown_below_the_bottom_row(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	sc2ExpectPanic(t, func() { s.CursorDown(3) })
}

// Go-only: panic test: cursorHorizontalAbsolute: x >= cols
func TestSc2_panic_cursorHorizontalAbsolute_beyond_the_last_column(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	sc2ExpectPanic(t, func() { s.CursorHorizontalAbsolute(10) })
}

// Go-only: panic test: cursorAbsolute: x >= cols
func TestSc2_panic_cursorAbsolute_x_beyond_the_last_column(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	sc2ExpectPanic(t, func() { s.CursorAbsolute(10, 0) })
}

// Go-only: panic test: cursorAbsolute: y >= rows
func TestSc2_panic_cursorAbsolute_y_below_the_bottom_row(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	sc2ExpectPanic(t, func() { s.CursorAbsolute(0, 3) })
}

// Go-only: panic test: cursorDownScroll: cursor not on the bottom row
func TestSc2_panic_cursorDownScroll_cursor_not_on_the_bottom_row(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	sc2ExpectPanic(t, func() { _ = s.CursorDownScroll() })
}

// Go-only: panic test: assertIntegrity: cursor.x >= cols
func TestSc2_panic_assertIntegrity_cursor_x_beyond_the_columns(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	s.cursor.x = 10
	sc2ExpectPanic(t, func() { s.AssertIntegrity() })
}

// Go-only: panic test: assertIntegrity: cursor.y >= rows
func TestSc2_panic_assertIntegrity_cursor_y_beyond_the_rows(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	s.cursor.y = 3
	sc2ExpectPanic(t, func() { s.AssertIntegrity() })
}

// Go-only: panic test: assertIntegrity: cursor.x does not match the pin
func TestSc2_panic_assertIntegrity_cursor_x_does_not_match_the_pin(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	s.cursor.x = 3
	sc2ExpectPanic(t, func() { s.AssertIntegrity() })
}

// Go-only: panic test: assertIntegrity: cursor.y does not match the pin
func TestSc2_panic_assertIntegrity_cursor_y_does_not_match_the_pin(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	s.cursor.y = 1
	sc2ExpectPanic(t, func() { s.AssertIntegrity() })
}

// Go-only: panic test: assertIntegrity: the cursor pin is outside the active area (unreachable)
func TestSc2_panic_assertIntegrity_cursor_pin_outside_the_active_area(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 100)
	_, err := s.pages.Grow()
	zigTry(t, err)
	expect(t, !s.pages.PointFromPin(PointTagActive, *s.cursor.page_pin).Valid,
		"s.pages.pointFromPin(.active, s.cursor.page_pin.*) == null")
	sc2ExpectPanic(t, func() { s.AssertIntegrity() })
}

// Go-only: panic test: clone: the selection is not tracked
func TestSc2_panic_clone_selection_is_not_tracked(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	sc2Write(t, s, "abc")
	s.selection = Some(SelectionInit(sc2OPin(t, s, 0, 0), sc2OPin(t, s, 2, 0), false))
	sc2ExpectPanic(t, func() { _, _ = s.Clone(sc2Active(0, 0), None[Point]()) })
}

// Go-only: panic test: endHyperlink: no hyperlink id but a hyperlink
func TestSc2_panic_endHyperlink_hyperlink_set_without_an_id(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	s.cursor.hyperlink = &Hyperlink{uri: []uint8("https://example.com/")}
	sc2ExpectPanic(t, func() { s.EndHyperlink() })
}

// Go-only: panic test: cursorSetHyperlink: hyperlink_id == 0
func TestSc2_panic_cursorSetHyperlink_without_a_hyperlink(t *testing.T) {
	testStdCapacity(t)
	s := sc2Init(t, 10, 3, 0)
	sc2ExpectPanic(t, func() { _ = s.CursorSetHyperlink() })
}
