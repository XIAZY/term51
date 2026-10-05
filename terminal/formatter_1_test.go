// Tests translated one for one from src/terminal/formatter.zig (Ghostty
// v1.3.1 plus the term51 additional tests), first part: the first 61 tests,
// from "Page plain single line" up to and including "Screen plain
// multiline". Each Go test carries the exact Zig title in a "// Zig: test"
// comment. Helpers are prefixed fm1.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualStrings keep Zig's names and Zig's (expected,
//     actual) argument order. `try` on a call that returns an error is
//     zigTry(t, err); `x.?` is zigUnwrap.
//   - `const alloc = testing.allocator;` and every `alloc` argument are
//     Zig-only (Go memory); they are not repeated in each test. The
//     `defer point_map.deinit(alloc)` / `defer pin_map.deinit(alloc)` lines
//     are marked in each test with a `// Zig-only:` comment.
//   - `var builder: std.Io.Writer.Allocating = .init(alloc); defer
//     builder.deinit();` is fm1Builder (the deinit is registered with
//     t.Cleanup). `var t = try Terminal.init(alloc, .{ .cols, .rows });
//     defer t.deinit(alloc); var s = t.vtStream(); defer s.deinit();` is
//     fm1Terminal, which starts from terminalOptionsDefault (max_scrollback
//     10_000, default colors and modes) and registers both deinits with
//     t.Cleanup (stream first, terminal last, as the defers run in Zig).
//     `try s.nextSlice(x)` is fm1Next.
//   - `std.ArrayList(Coordinate)` / `std.ArrayList(Pin)` are slices
//     (`[]PointCoordinate`, `[]PageListPin`) passed to the formatter through
//     `FormatterPageFormatterPointMap{map_: &point_map}` and
//     `FormatterPinMap{map_: &pin_map}`; `.items.len` is len and
//     `clearRetainingCapacity` is `point_map = point_map[:0]`.
//   - Options start from the presets (FormatterOptionsPlain, ...Vt) and set
//     the fields the Zig literal sets, so `trim` and the empty non-null
//     codepoint map keep Zig's defaults.
//   - `std.mem.trimStart(u8, s, set)` is bytes.TrimLeft(s, set).
//   - Zig's `defer` inside a block runs at the end of the block; the Go
//     test wraps that block in a function literal so the defer runs there.
//   - Every test calls testStdCapacity first (Zig's unit tests run with
//     grapheme_bytes 512, which decides how many rows a page holds).
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt (none in this part).

package terminal

import (
	"bytes"
	"testing"
)

// fm1Builder is `var builder: std.Io.Writer.Allocating = .init(alloc); defer
// builder.deinit();`.
func fm1Builder(t *testing.T) *oscWriterAllocating {
	t.Helper()
	b := oscWriterAllocatingInit()
	t.Cleanup(b.deinit)
	return &b
}

// fm1ClearRetainingCapacity is std.Io.Writer.Allocating.clearRetainingCapacity.
func fm1ClearRetainingCapacity(b *oscWriterAllocating) {
	b.clearRetainingCapacity()
}

// fm1Terminal is `var t = try Terminal.init(alloc, .{ .cols = cols, .rows =
// rows }); defer t.deinit(alloc); var s = t.vtStream(); defer s.deinit();`.
func fm1Terminal(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) (*Terminal, *StreamReadonlyStream) {
	t.Helper()
	term, err := TerminalInit(terminalOptionsDefault(cols, rows))
	zigTry(t, err)
	tp := &term
	t.Cleanup(tp.Deinit)
	s := tp.VtStream()
	sp := &s
	t.Cleanup(sp.Deinit)
	return tp, sp
}

// fm1Next is `try s.nextSlice(input)`.
func fm1Next(t *testing.T, s *StreamReadonlyStream, input string) {
	t.Helper()
	zigTry(t, s.NextSlice([]uint8(input)))
}

// fm1SinglePage is the Zig preamble
//
//	const pages = &t.screens.active.pages;
//	try testing.expect(pages.pages.first != null);
//	try testing.expect(pages.pages.first == pages.pages.last);
//	const page = &pages.pages.last.?.data;
func fm1SinglePage(t *testing.T, term *Terminal) *Page {
	t.Helper()
	pages := &term.screens.active.pages
	expect(t, pages.pages.first != nil, "pages.pages.first != null")
	expect(t, pages.pages.first == pages.pages.last, "pages.pages.first == pages.pages.last")
	return &fm1UnwrapPtr(t, pages.pages.last).data
}

// fm1LastPage is `const page = &pages.pages.last.?.data;`.
func fm1LastPage(t *testing.T, term *Terminal) *Page {
	t.Helper()
	pages := &term.screens.active.pages
	return &fm1UnwrapPtr(t, pages.pages.last).data
}

// fm1UnwrapPtr is Zig's `p.?` on an optional pointer (nil is null).
func fm1UnwrapPtr[T any](t *testing.T, p *T) *T {
	t.Helper()
	if p == nil {
		t.Fatalf("unwrapped a null optional")
	}
	return p
}

// fm1Coord is `Coordinate{ .x = x, .y = y }`.
func fm1Coord(x int, y int) PointCoordinate {
	return PointCoordinate{x: SizeCellCountInt(x), y: uint32(y)}
}

// fm1PointScreen is `.{ .screen = .{ .x = x, .y = y } }`.
func fm1PointScreen(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagScreen, Screen: PointCoordinate{x: x, y: y}}
}

// fm1PointActive is `.{ .active = .{ .x = x, .y = y } }`.
func fm1PointActive(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagActive, Active: PointCoordinate{x: x, y: y}}
}

// fm1ExpectMapLen checks `point_map.items.len == output.len`.
func fm1ExpectMapLen(t *testing.T, output []uint8, point_map []PointCoordinate) {
	t.Helper()
	expectEqual(t, len(output), len(point_map))
}

// fm1ExpectPanic runs f and fails the test if f does not panic.
func fm1ExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if recover() == nil {
			t.Fatalf("expected a panic, got none")
		}
	}()
	f()
}

// fm1Lines joins expected output lines with "\n" (Zig multiline string
// literals `\\a\n\\b`).
func fm1Lines(lines ...string) string {
	var b bytes.Buffer
	for i, l := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(l)
	}
	return b.String()
}

// Zig: test "Page plain single line"
func TestFm1_Page_plain_single_line(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello, world")

	// Verify we have only a single page
	// Create the formatter
	page := fm1SinglePage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)

	// Test our point map.
	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	// Verify output
	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	expectEqualStrings(t, "hello, world", string(output))
	expectEqual(t, uint(page.size.rows), state.rows)
	expectEqual(t, uint(page.size.cols-12), state.cells)

	// Verify our point map
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < len(output); i++ {
		expectEqual(t, fm1Coord(i, 0), point_map[i])
	}
}

// Zig: test "Page plain single line soft-wrapped unwrapped"
func TestFm1_Page_plain_single_line_soft_wrapped_unwrapped(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 3, 5)

	fm1Next(t, s, "hello!")

	// Verify we have only a single page
	// Create the formatter
	page := fm1SinglePage(t, term)
	opts := FormatterOptionsPlain
	opts.unwrap = true
	formatter := FormatterPageFormatterInit(page, opts)

	// Test our point map.
	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	// Verify output
	// Note: we don't test the trailing state, which may have bugs
	// with unwrap...
	_, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	expectEqualStrings(t, "hello!", string(output))

	// Verify our point map
	fm1ExpectMapLen(t, output, point_map)
	expectEqual(t, PointCoordinate{x: 0, y: 0}, point_map[0])
	expectEqual(t, PointCoordinate{x: 1, y: 0}, point_map[1])
	expectEqual(t, PointCoordinate{x: 2, y: 0}, point_map[2])
	expectEqual(t, PointCoordinate{x: 0, y: 1}, point_map[3])
	expectEqual(t, PointCoordinate{x: 1, y: 1}, point_map[4])
	expectEqual(t, PointCoordinate{x: 2, y: 1}, point_map[5])
}

// Zig: test "Page plain single wide char"
func TestFm1_Page_plain_single_wide_char(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "1A⚡")

	// Verify we have only a single page
	// Create the formatter
	page := fm1SinglePage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)

	// Test our point map.
	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	// Full string
	{
		fm1ClearRetainingCapacity(builder)
		point_map = point_map[:0]
		state, err := formatter.FormatWithState(&builder.writer)
		zigTry(t, err)
		output := builder.writer.buffered()
		expectEqualStrings(t, "1A⚡", string(output))
		expectEqual(t, uint(page.size.rows), state.rows)
		expectEqual(t, uint(page.size.cols-4), state.cells)

		// Verify our point map
		fm1ExpectMapLen(t, output, point_map)
		for i := 2; i < len(output); i++ {
			expectEqual(t, PointCoordinate{x: 2, y: 0}, point_map[i])
		}
	}

	// Wide only (from start)
	{
		fm1ClearRetainingCapacity(builder)
		point_map = point_map[:0]

		formatter.start_x = 2
		state, err := formatter.FormatWithState(&builder.writer)
		zigTry(t, err)
		output := builder.writer.buffered()
		expectEqualStrings(t, "⚡", string(output))
		expectEqual(t, uint(page.size.rows), state.rows)
		expectEqual(t, uint(page.size.cols-4), state.cells)

		// Verify our point map
		fm1ExpectMapLen(t, output, point_map)
		for i := 0; i < len(output); i++ {
			expectEqual(t, PointCoordinate{x: 2, y: 0}, point_map[i])
		}
	}

	// Wide only (from tail)
	{
		fm1ClearRetainingCapacity(builder)
		point_map = point_map[:0]

		formatter.start_x = 3
		state, err := formatter.FormatWithState(&builder.writer)
		zigTry(t, err)
		output := builder.writer.buffered()
		expectEqualStrings(t, "⚡", string(output))
		expectEqual(t, uint(page.size.rows), state.rows)
		expectEqual(t, uint(page.size.cols-4), state.cells)

		// Verify our point map
		fm1ExpectMapLen(t, output, point_map)
		for i := 0; i < len(output); i++ {
			expectEqual(t, PointCoordinate{x: 2, y: 0}, point_map[i])
		}
	}
}

// Zig: test "Page plain single wide char soft-wrapped unwrapped"
func TestFm1_Page_plain_single_wide_char_soft_wrapped_unwrapped(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 3, 24)

	fm1Next(t, s, "1A⚡")

	// Verify we have only a single page
	// Create the formatter
	page := fm1SinglePage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	formatter.opts.unwrap = true

	// Test our point map.
	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	// Full string
	{
		fm1ClearRetainingCapacity(builder)
		point_map = point_map[:0]
		state, err := formatter.FormatWithState(&builder.writer)
		zigTry(t, err)
		output := builder.writer.buffered()
		expectEqualStrings(t, "1A⚡", string(output))
		expectEqual(t, uint(page.size.rows-1), state.rows)
		expectEqual(t, uint(page.size.cols-2), state.cells)

		// Verify our point map
		fm1ExpectMapLen(t, output, point_map)
		for i := 2; i < len(output); i++ {
			expectEqual(t, PointCoordinate{x: 0, y: 1}, point_map[i])
		}
	}

	// Full string (ending on spacer head)
	func() {
		fm1ClearRetainingCapacity(builder)
		point_map = point_map[:0]

		formatter.end_x = Some(SizeCellCountInt(2))
		formatter.end_y = Some(SizeCellCountInt(0))
		defer func() {
			formatter.end_x = None[SizeCellCountInt]()
			formatter.end_y = None[SizeCellCountInt]()
		}()

		_, err := formatter.FormatWithState(&builder.writer)
		zigTry(t, err)
		output := builder.writer.buffered()
		expectEqualStrings(t, "1A⚡", string(output))

		// Verify our point map
		fm1ExpectMapLen(t, output, point_map)
		for i := 2; i < len(output); i++ {
			expectEqual(t, PointCoordinate{x: 0, y: 1}, point_map[i])
		}
	}()

	// Wide only (from start)
	{
		fm1ClearRetainingCapacity(builder)
		point_map = point_map[:0]

		formatter.start_x = 2
		state, err := formatter.FormatWithState(&builder.writer)
		zigTry(t, err)
		output := builder.writer.buffered()
		expectEqualStrings(t, "⚡", string(output))
		expectEqual(t, uint(page.size.rows-1), state.rows)
		expectEqual(t, uint(page.size.cols-2), state.cells)

		// Verify our point map
		fm1ExpectMapLen(t, output, point_map)
		for i := 0; i < len(output); i++ {
			expectEqual(t, PointCoordinate{x: 0, y: 1}, point_map[i])
		}
	}

	// Wide only (from tail)
	{
		fm1ClearRetainingCapacity(builder)
		point_map = point_map[:0]

		formatter.start_y = 1
		formatter.start_x = 1
		state, err := formatter.FormatWithState(&builder.writer)
		zigTry(t, err)
		output := builder.writer.buffered()
		expectEqualStrings(t, "⚡", string(output))
		expectEqual(t, uint(page.size.rows-1), state.rows)
		expectEqual(t, uint(page.size.cols-2), state.cells)

		// Verify our point map
		fm1ExpectMapLen(t, output, point_map)
		for i := 0; i < len(output); i++ {
			expectEqual(t, PointCoordinate{x: 0, y: 1}, point_map[i])
		}
	}
}

// Zig: test "Page plain multiline"
func TestFm1_Page_plain_multiline(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello\r\nworld")

	// Verify we have only a single page
	// Create the formatter
	page := fm1SinglePage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	// Verify output
	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	expectEqualStrings(t, "hello\nworld", string(output))
	expectEqual(t, uint(page.size.rows-1), state.rows)
	expectEqual(t, uint(page.size.cols-5), state.cells)

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i, 0), point_map[i])
	}
	expectEqual(t, PointCoordinate{x: 4, y: 0}, point_map[5]) // \n
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i, 1), point_map[6+i])
	}
}

// Zig: test "Page plain multiline rectangle"
func TestFm1_Page_plain_multiline_rectangle(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello\r\nworld")

	// Verify we have only a single page
	// Create the formatter
	page := fm1SinglePage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	formatter.start_x = 1
	formatter.end_x = Some(SizeCellCountInt(3))
	formatter.rectangle = true

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	// Verify output
	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	expectEqualStrings(t, "ell\norl", string(output))
	expectEqual(t, uint(page.size.rows-1), state.rows)
	expectEqual(t, uint(0), state.cells)

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < 3; i++ {
		expectEqual(t, fm1Coord(i+1, 0), point_map[i])
	}
	expectEqual(t, PointCoordinate{x: 3, y: 0}, point_map[3]) // \n
	for i := 0; i < 3; i++ {
		expectEqual(t, fm1Coord(i+1, 1), point_map[4+i])
	}
}

// Zig: test "Page plain multi blank lines"
func TestFm1_Page_plain_multi_blank_lines(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello\r\n\r\n\r\nworld")

	// Verify we have only a single page
	// Create the formatter
	page := fm1SinglePage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	// Verify output
	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	expectEqualStrings(t, "hello\n\n\nworld", string(output))
	expectEqual(t, uint(page.size.rows-3), state.rows)
	expectEqual(t, uint(page.size.cols-5), state.cells)

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i, 0), point_map[i])
	}
	expectEqual(t, PointCoordinate{x: 4, y: 0}, point_map[5]) // \n after row 0
	expectEqual(t, PointCoordinate{x: 0, y: 1}, point_map[6]) // \n after blank row 1
	expectEqual(t, PointCoordinate{x: 0, y: 2}, point_map[7]) // \n after blank row 2
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i, 3), point_map[8+i])
	}
}

// Zig: test "Page plain trailing blank lines"
func TestFm1_Page_plain_trailing_blank_lines(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello\r\nworld\r\n\r\n")

	// Verify we have only a single page
	// Create the formatter
	page := fm1SinglePage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	// Verify output. We expect there to be no trailing newlines because
	// we can't differentiate trailing blank lines as being meaningful because
	// the page formatter can't see the cursor position.
	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	expectEqualStrings(t, "hello\nworld", string(output))
	expectEqual(t, uint(page.size.rows-1), state.rows)
	expectEqual(t, uint(page.size.cols-5), state.cells)

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i, 0), point_map[i])
	}
	expectEqual(t, PointCoordinate{x: 4, y: 0}, point_map[5]) // \n
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i, 1), point_map[6+i])
	}
}

// Zig: test "Page plain trailing whitespace"
func TestFm1_Page_plain_trailing_whitespace(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello   \r\nworld   ")

	// Verify we have only a single page
	// Create the formatter
	page := fm1SinglePage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	// Verify output. We expect there to be no trailing newlines because
	// we can't differentiate trailing blank lines as being meaningful because
	// the page formatter can't see the cursor position.
	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	expectEqualStrings(t, "hello\nworld", string(output))
	expectEqual(t, uint(page.size.rows-1), state.rows)
	expectEqual(t, uint(page.size.cols-5), state.cells)

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i, 0), point_map[i])
	}
	expectEqual(t, PointCoordinate{x: 4, y: 0}, point_map[5]) // \n
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i, 1), point_map[6+i])
	}
}

// Zig: test "Page plain trailing whitespace no trim"
func TestFm1_Page_plain_trailing_whitespace_no_trim(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello   \r\nworld  ")

	// Verify we have only a single page
	// Create the formatter
	page := fm1SinglePage(t, term)
	opts := FormatterOptionsPlain
	opts.trim = false
	formatter := FormatterPageFormatterInit(page, opts)

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	// Verify output. We expect there to be no trailing newlines because
	// we can't differentiate trailing blank lines as being meaningful because
	// the page formatter can't see the cursor position.
	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	expectEqualStrings(t, "hello   \nworld  ", string(output))
	expectEqual(t, uint(page.size.rows-1), state.rows)
	expectEqual(t, uint(page.size.cols-7), state.cells)

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < 8; i++ {
		expectEqual(t, fm1Coord(i, 0), point_map[i])
	}
	expectEqual(t, PointCoordinate{x: 7, y: 0}, point_map[8]) // \n
	for i := 0; i < 7; i++ {
		expectEqual(t, fm1Coord(i, 1), point_map[9+i])
	}
}

// Zig: test "Page plain with prior trailing state rows"
func TestFm1_Page_plain_with_prior_trailing_state_rows(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello")

	page := fm1SinglePage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	formatter.trailing_state = Some(FormatterPageFormatterTrailingState{rows: 2, cells: 0})

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	expectEqualStrings(t, "\n\nhello", string(output))
	expectEqual(t, uint(page.size.rows), state.rows)
	expectEqual(t, uint(page.size.cols-5), state.cells)

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	expectEqual(t, PointCoordinate{x: 0, y: 0}, point_map[0]) // \n first blank row
	expectEqual(t, PointCoordinate{x: 0, y: 1}, point_map[1]) // \n second blank row
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i, 0), point_map[2+i])
	}
}

// Zig: test "Page plain with prior trailing state cells no wrapped line"
func TestFm1_Page_plain_with_prior_trailing_state_cells_no_wrapped_line(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello")

	page := fm1SinglePage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	formatter.trailing_state = Some(FormatterPageFormatterTrailingState{rows: 0, cells: 3})

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	// Blank cells are reset when row is not a wrap continuation
	expectEqualStrings(t, "hello", string(output))
	expectEqual(t, uint(page.size.rows), state.rows)
	expectEqual(t, uint(page.size.cols-5), state.cells)

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i, 0), point_map[i])
	}
}

// Zig: test "Page plain with prior trailing state cells with wrap continuation"
func TestFm1_Page_plain_with_prior_trailing_state_cells_with_wrap_continuation(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "world")

	page := fm1SinglePage(t, term)

	// Surgically modify the first row to be a wrap continuation
	row := page.GetRow(0)
	row.wrap_continuation = true

	opts := FormatterOptionsPlain
	opts.unwrap = true
	formatter := FormatterPageFormatterInit(page, opts)
	formatter.trailing_state = Some(FormatterPageFormatterTrailingState{rows: 0, cells: 3})

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	// Blank cells are preserved when row is a wrap continuation with unwrap enabled
	expectEqualStrings(t, "   world", string(output))
	expectEqual(t, uint(page.size.rows), state.rows)
	expectEqual(t, uint(page.size.cols-5), state.cells)

	// Verify point map - 3 spaces from prior trailing state + "world"
	fm1ExpectMapLen(t, output, point_map)
	// The 3 blank cells can't go back beyond (0,0) so they all map to (0,0)
	expectEqual(t, PointCoordinate{x: 0, y: 0}, point_map[0]) // space
	expectEqual(t, PointCoordinate{x: 0, y: 0}, point_map[1]) // space
	expectEqual(t, PointCoordinate{x: 0, y: 0}, point_map[2]) // space
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i, 0), point_map[3+i])
	}
}

// Zig: test "Page plain soft-wrapped without unwrap"
func TestFm1_Page_plain_soft_wrapped_without_unwrap(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 10, 24)

	fm1Next(t, s, "hello world test")

	page := fm1SinglePage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	// Without unwrap, wrapped lines show as separate lines
	expectEqualStrings(t, "hello worl\nd test", string(output))
	expectEqual(t, uint(page.size.rows-1), state.rows)
	expectEqual(t, uint(page.size.cols-6), state.cells)

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < 10; i++ {
		expectEqual(t, fm1Coord(i, 0), point_map[i])
	}
	expectEqual(t, PointCoordinate{x: 9, y: 0}, point_map[10]) // \n
	for i := 0; i < 6; i++ {
		expectEqual(t, fm1Coord(i, 1), point_map[11+i])
	}
}

// Zig: test "Page plain soft-wrapped with unwrap"
func TestFm1_Page_plain_soft_wrapped_with_unwrap(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 10, 24)

	fm1Next(t, s, "hello world test")

	page := fm1SinglePage(t, term)
	opts := FormatterOptionsPlain
	opts.unwrap = true
	formatter := FormatterPageFormatterInit(page, opts)

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	// With unwrap, wrapped lines are joined together
	expectEqualStrings(t, "hello world test", string(output))
	expectEqual(t, uint(page.size.rows-1), state.rows)
	expectEqual(t, uint(page.size.cols-6), state.cells)

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < 10; i++ {
		expectEqual(t, fm1Coord(i, 0), point_map[i])
	}
	for i := 0; i < 6; i++ {
		expectEqual(t, fm1Coord(i, 1), point_map[10+i])
	}
}

// Zig: test "Page plain soft-wrapped 3 lines without unwrap"
func TestFm1_Page_plain_soft_wrapped_3_lines_without_unwrap(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 10, 24)

	fm1Next(t, s, "hello world this is a test")

	page := fm1SinglePage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	// Without unwrap, wrapped lines show as separate lines
	expectEqualStrings(t, "hello worl\nd this is\na test", string(output))
	expectEqual(t, uint(page.size.rows-2), state.rows)
	expectEqual(t, uint(page.size.cols-6), state.cells)

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < 10; i++ {
		expectEqual(t, fm1Coord(i, 0), point_map[i])
	}
	expectEqual(t, PointCoordinate{x: 9, y: 0}, point_map[10]) // \n
	for i := 0; i < 9; i++ {
		expectEqual(t, fm1Coord(i, 1), point_map[11+i])
	}
	expectEqual(t, PointCoordinate{x: 8, y: 1}, point_map[20]) // \n
	for i := 0; i < 6; i++ {
		expectEqual(t, fm1Coord(i, 2), point_map[21+i])
	}
}

// Zig: test "Page plain soft-wrapped 3 lines with unwrap"
func TestFm1_Page_plain_soft_wrapped_3_lines_with_unwrap(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 10, 24)

	fm1Next(t, s, "hello world this is a test")

	page := fm1SinglePage(t, term)
	opts := FormatterOptionsPlain
	opts.unwrap = true
	formatter := FormatterPageFormatterInit(page, opts)

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	// With unwrap, wrapped lines are joined together
	expectEqualStrings(t, "hello world this is a test", string(output))
	expectEqual(t, uint(page.size.rows-2), state.rows)
	expectEqual(t, uint(page.size.cols-6), state.cells)

	// Verify point map - unwrapped text spans 3 rows
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < 10; i++ {
		expectEqual(t, fm1Coord(i, 0), point_map[i])
	}
	for i := 0; i < 10; i++ {
		expectEqual(t, fm1Coord(i, 1), point_map[10+i])
	}
	for i := 0; i < 6; i++ {
		expectEqual(t, fm1Coord(i, 2), point_map[20+i])
	}
}

// Zig: test "Page plain start_y subset"
func TestFm1_Page_plain_start_y_subset(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello\r\nworld\r\ntest")

	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	formatter.start_y = 1

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	expectEqualStrings(t, "world\ntest", string(output))
	expectEqual(t, uint(page.size.rows-2), state.rows)
	expectEqual(t, uint(page.size.cols-4), state.cells)

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i, 1), point_map[i])
	}
	expectEqual(t, PointCoordinate{x: 4, y: 1}, point_map[5]) // \n
	for i := 0; i < 4; i++ {
		expectEqual(t, fm1Coord(i, 2), point_map[6+i])
	}
}

// Zig: test "Page plain end_y subset"
func TestFm1_Page_plain_end_y_subset(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello\r\nworld\r\ntest")

	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	formatter.end_y = Some(SizeCellCountInt(1))

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	expectEqualStrings(t, "hello\nworld", string(output))
	expectEqual(t, uint(1), state.rows)
	expectEqual(t, uint(page.size.cols-5), state.cells)

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i, 0), point_map[i])
	}
	expectEqual(t, PointCoordinate{x: 4, y: 0}, point_map[5]) // \n
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i, 1), point_map[6+i])
	}
}

// Zig: test "Page plain start_y and end_y range"
func TestFm1_Page_plain_start_y_and_end_y_range(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello\r\nworld\r\ntest\r\nfoo")

	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	formatter.start_y = 1
	formatter.end_y = Some(SizeCellCountInt(2))

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	expectEqualStrings(t, "world\ntest", string(output))
	expectEqual(t, uint(1), state.rows)
	expectEqual(t, uint(page.size.cols-4), state.cells)

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i, 1), point_map[i])
	}
	expectEqual(t, PointCoordinate{x: 4, y: 1}, point_map[5]) // \n
	for i := 0; i < 4; i++ {
		expectEqual(t, fm1Coord(i, 2), point_map[6+i])
	}
}

// Zig: test "Page plain start_y out of bounds"
func TestFm1_Page_plain_start_y_out_of_bounds(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello")

	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	formatter.start_y = 30

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	expectEqualStrings(t, "", string(output))
	expectEqual(t, uint(0), state.rows)
	expectEqual(t, uint(0), state.cells)

	// Verify point map is empty
	expectEqual(t, 0, len(point_map))
}

// Zig: test "Page plain end_y greater than rows"
func TestFm1_Page_plain_end_y_greater_than_rows(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello")

	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	formatter.end_y = Some(SizeCellCountInt(30))

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	// Should clamp to page.size.rows and work normally
	expectEqualStrings(t, "hello", string(output))
	expectEqual(t, uint(page.size.rows), state.rows)
	expectEqual(t, uint(page.size.cols-5), state.cells)

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i, 0), point_map[i])
	}
}

// Zig: test "Page plain end_y less than start_y"
func TestFm1_Page_plain_end_y_less_than_start_y(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello")

	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	formatter.start_y = 5
	formatter.end_y = Some(SizeCellCountInt(2))

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	expectEqualStrings(t, "", string(output))
	expectEqual(t, uint(0), state.rows)
	expectEqual(t, uint(0), state.cells)

	// Verify point map is empty
	expectEqual(t, 0, len(point_map))
}

// Zig: test "Page plain start_x on first row only"
func TestFm1_Page_plain_start_x_on_first_row_only(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello world")

	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	formatter.start_x = 6

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	expectEqualStrings(t, "world", string(output))
	expectEqual(t, uint(page.size.rows), state.rows)
	expectEqual(t, uint(page.size.cols-11), state.cells)

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i+6, 0), point_map[i])
	}
}

// Zig: test "Page plain end_x on last row only"
func TestFm1_Page_plain_end_x_on_last_row_only(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "first line\r\nsecond line\r\nthird line")

	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	formatter.end_y = Some(SizeCellCountInt(2))
	formatter.end_x = Some(SizeCellCountInt(4))

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	expectEqualStrings(t, "first line\nsecond line\nthird", string(output))
	expectEqual(t, uint(1), state.rows)
	expectEqual(t, uint(0), state.cells)

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < 10; i++ {
		expectEqual(t, fm1Coord(i, 0), point_map[i])
	}
	expectEqual(t, PointCoordinate{x: 9, y: 0}, point_map[10]) // \n
	for i := 0; i < 11; i++ {
		expectEqual(t, fm1Coord(i, 1), point_map[11+i])
	}
	expectEqual(t, PointCoordinate{x: 10, y: 1}, point_map[22]) // \n
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i, 2), point_map[23+i])
	}
}

// Zig: test "Page plain start_x and end_x multiline"
func TestFm1_Page_plain_start_x_and_end_x_multiline(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello world\r\ntest case\r\nfoo bar")

	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	formatter.start_x = 6
	formatter.end_y = Some(SizeCellCountInt(2))
	formatter.end_x = Some(SizeCellCountInt(2))

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	// First row: "world" (start_x=6 to end of row)
	// Second row: "test case" (full row)
	// Third row: "foo" (start to end_x=2, inclusive)
	expectEqualStrings(t, "world\ntest case\nfoo", string(output))
	expectEqual(t, uint(1), state.rows)
	expectEqual(t, uint(0), state.cells)

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i+6, 0), point_map[i])
	}
	expectEqual(t, PointCoordinate{x: 10, y: 0}, point_map[5]) // \n
	for i := 0; i < 9; i++ {
		expectEqual(t, fm1Coord(i, 1), point_map[6+i])
	}
	expectEqual(t, PointCoordinate{x: 8, y: 1}, point_map[15]) // \n
	for i := 0; i < 3; i++ {
		expectEqual(t, fm1Coord(i, 2), point_map[16+i])
	}
}

// Zig: test "Page plain start_x out of bounds"
func TestFm1_Page_plain_start_x_out_of_bounds(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello")

	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	formatter.start_x = 100

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	expectEqualStrings(t, "", string(output))
	expectEqual(t, uint(0), state.rows)
	expectEqual(t, uint(0), state.cells)

	// Verify point map is empty
	expectEqual(t, 0, len(point_map))
}

// Zig: test "Page plain end_x greater than cols"
func TestFm1_Page_plain_end_x_greater_than_cols(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello")

	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	formatter.end_x = Some(SizeCellCountInt(100))

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	expectEqualStrings(t, "hello", string(output))
	expectEqual(t, uint(page.size.rows), state.rows)
	expectEqual(t, uint(page.size.cols-5), state.cells)

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i, 0), point_map[i])
	}
}

// Zig: test "Page plain end_x less than start_x single row"
func TestFm1_Page_plain_end_x_less_than_start_x_single_row(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello")

	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	formatter.start_x = 10
	formatter.end_y = Some(SizeCellCountInt(0))
	formatter.end_x = Some(SizeCellCountInt(5))

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	expectEqualStrings(t, "", string(output))
	expectEqual(t, uint(0), state.rows)
	expectEqual(t, uint(0), state.cells)

	// Verify point map is empty
	expectEqual(t, 0, len(point_map))
}

// Zig: test "Page plain start_y non-zero ignores trailing state"
func TestFm1_Page_plain_start_y_non_zero_ignores_trailing_state(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello\r\nworld")

	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	formatter.start_y = 1
	formatter.trailing_state = Some(FormatterPageFormatterTrailingState{rows: 5, cells: 10})

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	// Should NOT output the 5 newlines from trailing_state because start_y is non-zero
	expectEqualStrings(t, "world", string(output))
	expectEqual(t, uint(page.size.rows-1), state.rows)
	expectEqual(t, uint(page.size.cols-5), state.cells)

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i, 1), point_map[i])
	}
}

// Zig: test "Page plain start_x non-zero ignores trailing state"
func TestFm1_Page_plain_start_x_non_zero_ignores_trailing_state(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello world")

	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	formatter.start_x = 6
	formatter.trailing_state = Some(FormatterPageFormatterTrailingState{rows: 2, cells: 8})

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	// Should NOT output the 2 newlines or 8 spaces from trailing_state because start_x is non-zero
	expectEqualStrings(t, "world", string(output))
	expectEqual(t, uint(page.size.rows), state.rows)
	expectEqual(t, uint(page.size.cols-11), state.cells)

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i+6, 0), point_map[i])
	}
}

// Zig: test "Page plain start_y and start_x zero uses trailing state"
func TestFm1_Page_plain_start_y_and_start_x_zero_uses_trailing_state(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello")

	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	formatter.start_y = 0
	formatter.start_x = 0
	formatter.trailing_state = Some(FormatterPageFormatterTrailingState{rows: 2, cells: 0})

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	// SHOULD output the 2 newlines from trailing_state because both start_y and start_x are 0
	expectEqualStrings(t, "\n\nhello", string(output))
	expectEqual(t, uint(page.size.rows), state.rows)
	expectEqual(t, uint(page.size.cols-5), state.cells)

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	expectEqual(t, PointCoordinate{x: 0, y: 0}, point_map[0]) // \n first blank row
	expectEqual(t, PointCoordinate{x: 0, y: 1}, point_map[1]) // \n second blank row
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i, 0), point_map[2+i])
	}
}

// Zig: test "Page plain single line with styling"
func TestFm1_Page_plain_single_line_with_styling(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello, \x1b[1mworld\x1b[0m")

	// Verify we have only a single page
	// Create the formatter
	page := fm1SinglePage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	// Verify output
	state, err := formatter.FormatWithState(&builder.writer)
	zigTry(t, err)
	output := builder.writer.buffered()
	expectEqualStrings(t, "hello, world", string(output))
	expectEqual(t, uint(page.size.rows), state.rows)
	expectEqual(t, uint(page.size.cols-12), state.cells)

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < 12; i++ {
		expectEqual(t, fm1Coord(i, 0), point_map[i])
	}
}

// Zig: test "Page VT single line plain text"
func TestFm1_Page_VT_single_line_plain_text(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello")

	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsVt)

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	zigTry(t, formatter.Format(&builder.writer))
	output := builder.writer.buffered()
	expectEqualStrings(t, "hello", string(output))

	// Verify point map
	fm1ExpectMapLen(t, output, point_map)
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i, 0), point_map[i])
	}
}

// Zig: test "Page VT single line with bold"
func TestFm1_Page_VT_single_line_with_bold(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "\x1b[1mhello\x1b[0m")

	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsVt)

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	zigTry(t, formatter.Format(&builder.writer))
	output := builder.writer.buffered()
	expectEqualStrings(t, "\x1b[0m\x1b[1mhello\x1b[0m", string(output))

	// Verify point map - style sequences should point to first character they style
	fm1ExpectMapLen(t, output, point_map)
	// \x1b[0m = 4 bytes, \x1b[1m = 4 bytes, total 8 bytes of style sequences
	// All style bytes should map to the first styled character at (0, 0)
	for i := 0; i < 8; i++ {
		expectEqual(t, PointCoordinate{x: 0, y: 0}, point_map[i])
	}
	// Then "hello" maps to its respective positions
	for i := 0; i < 5; i++ {
		expectEqual(t, fm1Coord(i, 0), point_map[8+i])
	}
}

// Zig: test "Page VT multiple styles"
func TestFm1_Page_VT_multiple_styles(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "\x1b[1mhello \x1b[3mworld\x1b[0m")

	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsVt)

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	zigTry(t, formatter.Format(&builder.writer))
	output := builder.writer.buffered()
	expectEqualStrings(t, "\x1b[0m\x1b[1mhello \x1b[0m\x1b[1m\x1b[3mworld\x1b[0m", string(output))

	// Verify point map matches output length
	fm1ExpectMapLen(t, output, point_map)
}

// Zig: test "Page VT with foreground color"
func TestFm1_Page_VT_with_foreground_color(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "\x1b[31mred\x1b[0m")

	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsVt)

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	zigTry(t, formatter.Format(&builder.writer))
	output := builder.writer.buffered()
	expectEqualStrings(t, "\x1b[0m\x1b[38;5;1mred\x1b[0m", string(output))

	// Verify point map - style sequences should point to first character they style
	fm1ExpectMapLen(t, output, point_map)
	// \x1b[0m = 4 bytes, \x1b[38;5;1m = 9 bytes, total 13 bytes of style sequences
	// All style bytes should map to the first styled character at (0, 0)
	for i := 0; i < 13; i++ {
		expectEqual(t, PointCoordinate{x: 0, y: 0}, point_map[i])
	}
	// Then "red" maps to its respective positions
	for i := 0; i < 3; i++ {
		expectEqual(t, fm1Coord(i, 0), point_map[13+i])
	}
}

// Zig: test "Page VT with background and foreground colors"
func TestFm1_Page_VT_with_background_and_foreground_colors(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello")

	page := fm1LastPage(t, term)

	opts := FormatterOptionsVt
	opts.background = Some(ColorRGB{r: 0x12, g: 0x34, b: 0x56})
	opts.foreground = Some(ColorRGB{r: 0xab, g: 0xcd, b: 0xef})
	formatter := FormatterPageFormatterInit(page, opts)

	zigTry(t, formatter.Format(&builder.writer))
	output := builder.writer.buffered()

	// Should emit OSC 10 for foreground, OSC 11 for background, then the text
	expectEqualStrings(t,
		"\x1b]10;rgb:ab/cd/ef\x1b\\\x1b]11;rgb:12/34/56\x1b\\hello",
		string(output),
	)
}

// Zig: test "Page VT multi-line with styles"
func TestFm1_Page_VT_multi_line_with_styles(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "\x1b[1mfirst\x1b[0m\r\n\x1b[3msecond\x1b[0m")

	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsVt)

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	zigTry(t, formatter.Format(&builder.writer))
	output := builder.writer.buffered()
	// Note: style is reset before newline to prevent background colors from
	// bleeding to the next line's leading cells.
	expectEqualStrings(t, "\x1b[0m\x1b[1mfirst\x1b[0m\r\n\x1b[0m\x1b[3msecond\x1b[0m", string(output))

	// Verify point map matches output length
	fm1ExpectMapLen(t, output, point_map)
}

// Zig: test "Page VT duplicate style not emitted twice"
func TestFm1_Page_VT_duplicate_style_not_emitted_twice(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "\x1b[1mhel\x1b[1mlo\x1b[0m")

	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsVt)

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	zigTry(t, formatter.Format(&builder.writer))
	output := builder.writer.buffered()
	expectEqualStrings(t, "\x1b[0m\x1b[1mhello\x1b[0m", string(output))

	// Verify point map matches output length
	fm1ExpectMapLen(t, output, point_map)
}

// Zig: test "PageList plain single line"
func TestFm1_PageList_plain_single_line(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello, world")

	var pin_map []PageListPin
	// Zig-only: defer pin_map.deinit(alloc);

	formatter := FormatterPageListFormatterInit(&term.screens.active.pages, FormatterOptionsPlain)
	formatter.pin_map = Some(FormatterPinMap{map_: &pin_map})
	zigTry(t, formatter.Format(&builder.writer))
	output := builder.writer.buffered()
	expectEqualStrings(t, "hello, world", string(output))

	// Verify pin map
	expectEqual(t, len(output), len(pin_map))
	node := fm1UnwrapPtr(t, term.screens.active.pages.pages.first)
	for i := 0; i < len(output); i++ {
		expectEqual(t, PageListPin{node: node, x: SizeCellCountInt(i), y: 0}, pin_map[i])
	}
}

// Zig: test "PageList plain spanning two pages"
func TestFm1_PageList_plain_spanning_two_pages(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	pages := &term.screens.active.pages
	first_page_rows := fm1UnwrapPtr(t, pages.pages.first).data.capacity.rows

	// Fill the first page almost completely
	for i := 0; i < int(first_page_rows)-1; i++ {
		fm1Next(t, s, "\r\n")
	}
	fm1Next(t, s, "page one")

	// Verify we're still on one page
	expect(t, pages.pages.first == pages.pages.last, "pages.pages.first == pages.pages.last")

	// Add one more newline to push content to a second page
	fm1Next(t, s, "\r\n")
	expect(t, pages.pages.first != pages.pages.last, "pages.pages.first != pages.pages.last")

	// Write content on the second page
	fm1Next(t, s, "page two")

	// Format the entire PageList
	var pin_map []PageListPin
	// Zig-only: defer pin_map.deinit(alloc);

	formatter := FormatterPageListFormatterInit(pages, FormatterOptionsPlain)
	formatter.pin_map = Some(FormatterPinMap{map_: &pin_map})
	zigTry(t, formatter.Format(&builder.writer))
	full_output := builder.writer.buffered()
	output := bytes.TrimLeft(full_output, "\n")
	expectEqualStrings(t, "page one\npage two", string(output))

	// Verify pin map
	expectEqual(t, len(full_output), len(pin_map))
	first_node := fm1UnwrapPtr(t, pages.pages.first)
	last_node := fm1UnwrapPtr(t, pages.pages.last)
	trimmed_count := len(full_output) - len(output)

	// First part (trimmed blank lines) maps to first node
	for i := 0; i < trimmed_count; i++ {
		expectEqual(t, first_node, pin_map[i].node)
	}

	// "page one" (8 chars) maps to first node
	for i := 0; i < 8; i++ {
		idx := trimmed_count + i
		expectEqual(t, first_node, pin_map[idx].node)
		expectEqual(t, SizeCellCountInt(i), pin_map[idx].x)
	}

	// \n - maps to last node as it represents the transition to new page
	expectEqual(t, last_node, pin_map[trimmed_count+8].node)

	// "page two" (8 chars) maps to last node
	for i := 0; i < 8; i++ {
		idx := trimmed_count + 9 + i
		expectEqual(t, last_node, pin_map[idx].node)
		expectEqual(t, SizeCellCountInt(i), pin_map[idx].x)
	}
}

// Zig: test "PageList soft-wrapped line spanning two pages without unwrap"
func TestFm1_PageList_soft_wrapped_line_spanning_two_pages_without_unwrap(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 10, 3)

	pages := &term.screens.active.pages
	first_page_rows := fm1UnwrapPtr(t, pages.pages.first).data.capacity.rows

	// Fill the first page with soft-wrapped content
	for i := 0; i < int(first_page_rows)-1; i++ {
		fm1Next(t, s, "\r\n")
	}
	fm1Next(t, s, "hello world test")

	// Verify we're on two pages due to wrapping
	expect(t, pages.pages.first != pages.pages.last, "pages.pages.first != pages.pages.last")

	// Format without unwrap - should show line breaks
	var pin_map []PageListPin
	// Zig-only: defer pin_map.deinit(alloc);

	formatter := FormatterPageListFormatterInit(pages, FormatterOptionsPlain)
	formatter.pin_map = Some(FormatterPinMap{map_: &pin_map})
	zigTry(t, formatter.Format(&builder.writer))
	full_output := builder.writer.buffered()
	output := bytes.TrimLeft(full_output, "\n")
	expectEqualStrings(t, "hello worl\nd test", string(output))

	// Verify pin map
	expectEqual(t, len(full_output), len(pin_map))
	first_node := fm1UnwrapPtr(t, pages.pages.first)
	last_node := fm1UnwrapPtr(t, pages.pages.last)
	trimmed_count := len(full_output) - len(output)

	// First part (trimmed blank lines) maps to first node
	for i := 0; i < trimmed_count; i++ {
		expectEqual(t, first_node, pin_map[i].node)
	}

	// First line maps to first node
	for i := 0; i < 10; i++ {
		idx := trimmed_count + i
		expectEqual(t, first_node, pin_map[idx].node)
	}

	// \n - maps to last node as it represents the transition to new page
	expectEqual(t, last_node, pin_map[trimmed_count+10].node)

	// "d test" (6 chars) maps to last node
	for i := 0; i < 6; i++ {
		idx := trimmed_count + 11 + i
		expectEqual(t, last_node, pin_map[idx].node)
	}
}

// Zig: test "PageList soft-wrapped line spanning two pages with unwrap"
func TestFm1_PageList_soft_wrapped_line_spanning_two_pages_with_unwrap(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 10, 3)

	pages := &term.screens.active.pages
	first_page_rows := fm1UnwrapPtr(t, pages.pages.first).data.capacity.rows

	// Fill the first page with soft-wrapped content
	for i := 0; i < int(first_page_rows)-1; i++ {
		fm1Next(t, s, "\r\n")
	}
	fm1Next(t, s, "hello world test")

	// Verify we're on two pages due to wrapping
	expect(t, pages.pages.first != pages.pages.last, "pages.pages.first != pages.pages.last")

	// Format with unwrap - should join the wrapped lines
	var pin_map []PageListPin
	// Zig-only: defer pin_map.deinit(alloc);

	opts := FormatterOptionsPlain
	opts.unwrap = true
	formatter := FormatterPageListFormatterInit(pages, opts)
	formatter.pin_map = Some(FormatterPinMap{map_: &pin_map})
	zigTry(t, formatter.Format(&builder.writer))
	full_output := builder.writer.buffered()
	output := bytes.TrimLeft(full_output, "\r\n")
	expectEqualStrings(t, "hello world test", string(output))

	// Verify pin map
	expectEqual(t, len(full_output), len(pin_map))
	first_node := fm1UnwrapPtr(t, pages.pages.first)
	last_node := fm1UnwrapPtr(t, pages.pages.last)
	trimmed_count := len(full_output) - len(output)

	// First part (trimmed blank lines) maps to first node
	for i := 0; i < trimmed_count; i++ {
		expectEqual(t, first_node, pin_map[i].node)
	}

	// First line from first page
	for i := 0; i < 10; i++ {
		idx := trimmed_count + i
		expectEqual(t, first_node, pin_map[idx].node)
	}

	// "d test" (6 chars) from last page
	for i := 0; i < 6; i++ {
		idx := trimmed_count + 10 + i
		expectEqual(t, last_node, pin_map[idx].node)
	}
}

// Zig: test "PageList VT spanning two pages"
func TestFm1_PageList_VT_spanning_two_pages(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	pages := &term.screens.active.pages
	first_page_rows := fm1UnwrapPtr(t, pages.pages.first).data.capacity.rows

	// Fill the first page almost completely
	for i := 0; i < int(first_page_rows)-1; i++ {
		fm1Next(t, s, "\r\n")
	}
	fm1Next(t, s, "\x1b[1mpage one")

	// Verify we're still on one page
	expect(t, pages.pages.first == pages.pages.last, "pages.pages.first == pages.pages.last")

	// Add one more newline to push content to a second page
	fm1Next(t, s, "\r\n")
	expect(t, pages.pages.first != pages.pages.last, "pages.pages.first != pages.pages.last")

	// New content is still styled
	fm1Next(t, s, "page two")

	// Format the entire PageList with VT
	var pin_map []PageListPin
	// Zig-only: defer pin_map.deinit(alloc);

	formatter := FormatterPageListFormatterInit(pages, FormatterOptionsVt)
	formatter.pin_map = Some(FormatterPinMap{map_: &pin_map})
	zigTry(t, formatter.Format(&builder.writer))
	full_output := builder.writer.buffered()
	output := bytes.TrimLeft(full_output, "\r\n")
	expectEqualStrings(t, "\x1b[0m\x1b[1mpage one\x1b[0m\r\n\x1b[0m\x1b[1mpage two\x1b[0m", string(output))

	// Verify pin map
	expectEqual(t, len(full_output), len(pin_map))
	first_node := fm1UnwrapPtr(t, pages.pages.first)
	last_node := fm1UnwrapPtr(t, pages.pages.last)

	// Just verify we have entries for both pages in the pin map
	var first_count uint = 0
	var last_count uint = 0
	for _, pin := range pin_map {
		if pin.node == first_node {
			first_count += 1
		}
		if pin.node == last_node {
			last_count += 1
		}
	}
	expect(t, first_count > 0, "first_count > 0")
	expect(t, last_count > 0, "last_count > 0")
}

// Zig: test "PageList plain with x offset on single page"
func TestFm1_PageList_plain_with_x_offset_on_single_page(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello world\r\ntest case\r\nfoo bar")

	pages := &term.screens.active.pages
	node := fm1UnwrapPtr(t, pages.pages.first)

	var pin_map []PageListPin
	// Zig-only: defer pin_map.deinit(alloc);

	formatter := FormatterPageListFormatterInit(pages, FormatterOptionsPlain)
	formatter.top_left = Some(PageListPin{node: node, y: 0, x: 6})
	formatter.bottom_right = Some(PageListPin{node: node, y: 2, x: 2})
	formatter.pin_map = Some(FormatterPinMap{map_: &pin_map})

	zigTry(t, formatter.Format(&builder.writer))
	output := builder.writer.buffered()
	expectEqualStrings(t, "world\ntest case\nfoo", string(output))

	// Verify pin map
	expectEqual(t, len(output), len(pin_map))
	for _, pin := range pin_map {
		expectEqual(t, node, pin.node)
	}

	// "world" starts at x=6, y=0
	for i := 0; i < 5; i++ {
		expectEqual(t, SizeCellCountInt(6+i), pin_map[i].x)
		expectEqual(t, SizeCellCountInt(0), pin_map[i].y)
	}
}

// Zig: test "PageList plain with x offset spanning two pages"
func TestFm1_PageList_plain_with_x_offset_spanning_two_pages(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	pages := &term.screens.active.pages
	first_page_rows := fm1UnwrapPtr(t, pages.pages.first).data.capacity.rows

	// Fill first page almost completely
	for i := 0; i < int(first_page_rows)-1; i++ {
		fm1Next(t, s, "\r\n")
	}
	fm1Next(t, s, "hello world")

	// Verify we're still on one page
	expect(t, pages.pages.first == pages.pages.last, "pages.pages.first == pages.pages.last")

	// Push to second page
	fm1Next(t, s, "\r\n")
	expect(t, pages.pages.first != pages.pages.last, "pages.pages.first != pages.pages.last")

	fm1Next(t, s, "foo bar test")

	first_node := fm1UnwrapPtr(t, pages.pages.first)
	last_node := fm1UnwrapPtr(t, pages.pages.last)

	var pin_map []PageListPin
	// Zig-only: defer pin_map.deinit(alloc);

	formatter := FormatterPageListFormatterInit(pages, FormatterOptionsPlain)
	formatter.top_left = Some(PageListPin{node: first_node, y: first_node.data.size.rows - 1, x: 6})
	formatter.bottom_right = Some(PageListPin{node: last_node, y: 1, x: 2})
	formatter.pin_map = Some(FormatterPinMap{map_: &pin_map})

	zigTry(t, formatter.Format(&builder.writer))
	full_output := builder.writer.buffered()
	output := bytes.TrimLeft(full_output, "\n")
	expectEqualStrings(t, "world\nfoo", string(output))

	// Verify pin map
	expectEqual(t, len(full_output), len(pin_map))
	trimmed_count := len(full_output) - len(output)

	// "world" (5 chars) from first page
	for i := 0; i < 5; i++ {
		idx := trimmed_count + i
		expectEqual(t, first_node, pin_map[idx].node)
		expectEqual(t, SizeCellCountInt(6+i), pin_map[idx].x)
	}

	// \n - maps to last node as it represents the transition to new page
	expectEqual(t, last_node, pin_map[trimmed_count+5].node)

	// "foo" (3 chars) from last page
	for i := 0; i < 3; i++ {
		idx := trimmed_count + 6 + i
		expectEqual(t, last_node, pin_map[idx].node)
		expectEqual(t, SizeCellCountInt(i), pin_map[idx].x)
	}
}

// Zig: test "PageList plain with start_x only"
func TestFm1_PageList_plain_with_start_x_only(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello world")

	pages := &term.screens.active.pages
	node := fm1UnwrapPtr(t, pages.pages.first)

	var pin_map []PageListPin
	// Zig-only: defer pin_map.deinit(alloc);

	formatter := FormatterPageListFormatterInit(pages, FormatterOptionsPlain)
	formatter.top_left = Some(PageListPin{node: node, y: 0, x: 6})
	formatter.pin_map = Some(FormatterPinMap{map_: &pin_map})

	zigTry(t, formatter.Format(&builder.writer))
	output := builder.writer.buffered()
	expectEqualStrings(t, "world", string(output))

	// Verify pin map
	expectEqual(t, len(output), len(pin_map))
	for i := 0; i < 5; i++ {
		expectEqual(t, node, pin_map[i].node)
		expectEqual(t, SizeCellCountInt(6+i), pin_map[i].x)
		expectEqual(t, SizeCellCountInt(0), pin_map[i].y)
	}
}

// Zig: test "PageList plain with end_x only"
func TestFm1_PageList_plain_with_end_x_only(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello world\r\ntest")

	pages := &term.screens.active.pages
	node := fm1UnwrapPtr(t, pages.pages.first)

	var pin_map []PageListPin
	// Zig-only: defer pin_map.deinit(alloc);

	formatter := FormatterPageListFormatterInit(pages, FormatterOptionsPlain)
	formatter.bottom_right = Some(PageListPin{node: node, y: 1, x: 2})
	formatter.pin_map = Some(FormatterPinMap{map_: &pin_map})

	zigTry(t, formatter.Format(&builder.writer))
	output := builder.writer.buffered()
	expectEqualStrings(t, "hello world\ntes", string(output))

	// Verify pin map
	expectEqual(t, len(output), len(pin_map))

	// "hello world" (11 chars) on y=0
	for i := 0; i < 11; i++ {
		expectEqual(t, node, pin_map[i].node)
		expectEqual(t, SizeCellCountInt(i), pin_map[i].x)
		expectEqual(t, SizeCellCountInt(0), pin_map[i].y)
	}

	// \n
	expectEqual(t, node, pin_map[11].node)

	// "tes" (3 chars) on y=1
	for i := 0; i < 3; i++ {
		expectEqual(t, node, pin_map[12+i].node)
		expectEqual(t, SizeCellCountInt(i), pin_map[12+i].x)
		expectEqual(t, SizeCellCountInt(1), pin_map[12+i].y)
	}
}

// Zig: test "PageList plain rectangle basic"
func TestFm1_PageList_plain_rectangle_basic(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 30, 5)

	fm1Next(t, s, "Lorem ipsum dolor\r\n")
	fm1Next(t, s, "sit amet, consectetur\r\n")
	fm1Next(t, s, "adipiscing elit, sed do\r\n")
	fm1Next(t, s, "eiusmod tempor incididunt\r\n")
	fm1Next(t, s, "ut labore et dolore")

	pages := &term.screens.active.pages

	formatter := FormatterPageListFormatterInit(pages, FormatterOptionsPlain)
	formatter.top_left = Some(zigUnwrap(t, pages.Pin(fm1PointScreen(2, 1))))
	formatter.bottom_right = Some(zigUnwrap(t, pages.Pin(fm1PointScreen(6, 3))))
	formatter.rectangle = true

	zigTry(t, formatter.Format(&builder.writer))
	output := builder.writer.buffered()

	expected := fm1Lines(
		"t ame",
		"ipisc",
		"usmod",
	)
	expectEqualStrings(t, expected, string(output))
}

// Zig: test "PageList plain rectangle with EOL"
func TestFm1_PageList_plain_rectangle_with_EOL(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 30, 5)

	fm1Next(t, s, "Lorem ipsum dolor\r\n")
	fm1Next(t, s, "sit amet, consectetur\r\n")
	fm1Next(t, s, "adipiscing elit, sed do\r\n")
	fm1Next(t, s, "eiusmod tempor incididunt\r\n")
	fm1Next(t, s, "ut labore et dolore")

	pages := &term.screens.active.pages

	formatter := FormatterPageListFormatterInit(pages, FormatterOptionsPlain)
	formatter.top_left = Some(zigUnwrap(t, pages.Pin(fm1PointScreen(12, 0))))
	formatter.bottom_right = Some(zigUnwrap(t, pages.Pin(fm1PointScreen(26, 4))))
	formatter.rectangle = true

	zigTry(t, formatter.Format(&builder.writer))
	output := builder.writer.buffered()

	expected := fm1Lines(
		"dolor",
		"nsectetur",
		"lit, sed do",
		"or incididunt",
		" dolore",
	)
	expectEqualStrings(t, expected, string(output))
}

// Zig: test "PageList plain rectangle more complex with breaks"
func TestFm1_PageList_plain_rectangle_more_complex_with_breaks(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 30, 8)

	fm1Next(t, s, "Lorem ipsum dolor\r\n")
	fm1Next(t, s, "sit amet, consectetur\r\n")
	fm1Next(t, s, "adipiscing elit, sed do\r\n")
	fm1Next(t, s, "eiusmod tempor incididunt\r\n")
	fm1Next(t, s, "ut labore et dolore\r\n")
	fm1Next(t, s, "\r\n")
	fm1Next(t, s, "magna aliqua. Ut enim\r\n")
	fm1Next(t, s, "ad minim veniam, quis")

	pages := &term.screens.active.pages

	formatter := FormatterPageListFormatterInit(pages, FormatterOptionsPlain)
	formatter.top_left = Some(zigUnwrap(t, pages.Pin(fm1PointScreen(11, 2))))
	formatter.bottom_right = Some(zigUnwrap(t, pages.Pin(fm1PointScreen(26, 7))))
	formatter.rectangle = true

	zigTry(t, formatter.Format(&builder.writer))
	output := builder.writer.buffered()

	expected := fm1Lines(
		"elit, sed do",
		"por incididunt",
		"t dolore",
		"",
		"a. Ut enim",
		"niam, quis",
	)
	expectEqualStrings(t, expected, string(output))
}

// Zig: test "TerminalFormatter plain no selection"
func TestFm1_TerminalFormatter_plain_no_selection(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello\r\nworld")

	formatter := FormatterTerminalFormatterInit(term, FormatterOptionsPlain)

	zigTry(t, formatter.Format(&builder.writer))
	expectEqualStrings(t, "hello\nworld", string(builder.writer.buffered()))
}

// Zig: test "TerminalFormatter vt with palette"
func TestFm1_TerminalFormatter_vt_with_palette(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	// Modify some palette colors using VT sequences
	fm1Next(t, s, "\x1b]4;0;rgb:12/34/56\x1b\\")
	fm1Next(t, s, "\x1b]4;1;rgb:ab/cd/ef\x1b\\")
	fm1Next(t, s, "\x1b]4;255;rgb:ff/00/ff\x1b\\")
	fm1Next(t, s, "test")

	formatter := FormatterTerminalFormatterInit(term, FormatterOptionsVt)

	zigTry(t, formatter.Format(&builder.writer))
	output := builder.writer.buffered()

	// Create a second terminal and apply the output
	t2, s2 := fm1Terminal(t, 80, 24)

	zigTry(t, s2.NextSlice(output))

	// Verify the palettes match
	expectEqual(t, term.colors.palette.current[0], t2.colors.palette.current[0])
	expectEqual(t, term.colors.palette.current[1], t2.colors.palette.current[1])
	expectEqual(t, term.colors.palette.current[255], t2.colors.palette.current[255])
}

// Zig: test "TerminalFormatter with selection"
func TestFm1_TerminalFormatter_with_selection(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "line1\r\nline2\r\nline3")

	formatter := FormatterTerminalFormatterInit(term, FormatterOptionsPlain)
	formatter.content = FormatterScreenFormatterContent{
		Tag: FormatterScreenFormatterContentSelection,
		Selection: Some(SelectionInit(
			zigUnwrap(t, term.screens.active.pages.Pin(fm1PointActive(0, 1))),
			zigUnwrap(t, term.screens.active.pages.Pin(fm1PointActive(4, 1))),
			false,
		)),
	}

	zigTry(t, formatter.Format(&builder.writer))
	expectEqualStrings(t, "line2", string(builder.writer.buffered()))
}

// Zig: test "TerminalFormatter plain with pin_map"
func TestFm1_TerminalFormatter_plain_with_pin_map(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello, world")

	var pin_map []PageListPin
	// Zig-only: defer pin_map.deinit(alloc);

	formatter := FormatterTerminalFormatterInit(term, FormatterOptionsPlain)
	formatter.pin_map = Some(FormatterPinMap{map_: &pin_map})

	zigTry(t, formatter.Format(&builder.writer))
	output := builder.writer.buffered()
	expectEqualStrings(t, "hello, world", string(output))

	// Verify pin map
	expectEqual(t, len(output), len(pin_map))
	node := fm1UnwrapPtr(t, term.screens.active.pages.pages.first)
	for i := 0; i < len(output); i++ {
		expectEqual(t, PageListPin{node: node, x: SizeCellCountInt(i), y: 0}, pin_map[i])
	}
}

// Zig: test "TerminalFormatter plain multiline with pin_map"
func TestFm1_TerminalFormatter_plain_multiline_with_pin_map(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello\r\nworld")

	var pin_map []PageListPin
	// Zig-only: defer pin_map.deinit(alloc);

	formatter := FormatterTerminalFormatterInit(term, FormatterOptionsPlain)
	formatter.pin_map = Some(FormatterPinMap{map_: &pin_map})

	zigTry(t, formatter.Format(&builder.writer))
	output := builder.writer.buffered()
	expectEqualStrings(t, "hello\nworld", string(output))

	// Verify pin map
	expectEqual(t, len(output), len(pin_map))
	node := fm1UnwrapPtr(t, term.screens.active.pages.pages.first)
	// "hello" (5 chars)
	for i := 0; i < 5; i++ {
		expectEqual(t, node, pin_map[i].node)
		expectEqual(t, SizeCellCountInt(i), pin_map[i].x)
		expectEqual(t, SizeCellCountInt(0), pin_map[i].y)
	}
	// "\n" maps to end of first line
	expectEqual(t, node, pin_map[5].node)
	// "world" (5 chars)
	for i := 0; i < 5; i++ {
		idx := 6 + i
		expectEqual(t, node, pin_map[idx].node)
		expectEqual(t, SizeCellCountInt(i), pin_map[idx].x)
		expectEqual(t, SizeCellCountInt(1), pin_map[idx].y)
	}
}

// Zig: test "TerminalFormatter vt with palette and pin_map"
func TestFm1_TerminalFormatter_vt_with_palette_and_pin_map(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	// Modify some palette colors using VT sequences
	fm1Next(t, s, "\x1b]4;0;rgb:12/34/56\x1b\\")
	fm1Next(t, s, "test")

	var pin_map []PageListPin
	// Zig-only: defer pin_map.deinit(alloc);

	formatter := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
	formatter.pin_map = Some(FormatterPinMap{map_: &pin_map})

	zigTry(t, formatter.Format(&builder.writer))
	output := builder.writer.buffered()

	// Verify pin map - palette bytes should be mapped to top left
	expectEqual(t, len(output), len(pin_map))
	node := fm1UnwrapPtr(t, term.screens.active.pages.pages.first)
	for i := 0; i < len(output); i++ {
		expectEqual(t, node, pin_map[i].node)
	}
}

// Zig: test "TerminalFormatter with selection and pin_map"
func TestFm1_TerminalFormatter_with_selection_and_pin_map(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "line1\r\nline2\r\nline3")

	var pin_map []PageListPin
	// Zig-only: defer pin_map.deinit(alloc);

	formatter := FormatterTerminalFormatterInit(term, FormatterOptionsPlain)
	formatter.content = FormatterScreenFormatterContent{
		Tag: FormatterScreenFormatterContentSelection,
		Selection: Some(SelectionInit(
			zigUnwrap(t, term.screens.active.pages.Pin(fm1PointActive(0, 1))),
			zigUnwrap(t, term.screens.active.pages.Pin(fm1PointActive(4, 1))),
			false,
		)),
	}
	formatter.pin_map = Some(FormatterPinMap{map_: &pin_map})

	zigTry(t, formatter.Format(&builder.writer))
	output := builder.writer.buffered()
	expectEqualStrings(t, "line2", string(output))

	// Verify pin map
	expectEqual(t, len(output), len(pin_map))
	node := fm1UnwrapPtr(t, term.screens.active.pages.pages.first)
	// "line2" (5 chars) from row 1
	for i := 0; i < 5; i++ {
		expectEqual(t, node, pin_map[i].node)
		expectEqual(t, SizeCellCountInt(i), pin_map[i].x)
		expectEqual(t, SizeCellCountInt(1), pin_map[i].y)
	}
}

// Zig: test "Screen plain single line"
func TestFm1_Screen_plain_single_line(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello, world")

	var pin_map []PageListPin
	// Zig-only: defer pin_map.deinit(alloc);

	formatter := FormatterScreenFormatterInit(term.screens.active, FormatterOptionsPlain)
	formatter.pin_map = Some(FormatterPinMap{map_: &pin_map})

	zigTry(t, formatter.Format(&builder.writer))
	output := builder.writer.buffered()
	expectEqualStrings(t, "hello, world", string(output))

	// Verify pin map
	expectEqual(t, len(output), len(pin_map))
	node := fm1UnwrapPtr(t, term.screens.active.pages.pages.first)
	for i := 0; i < len(output); i++ {
		expectEqual(t, PageListPin{node: node, x: SizeCellCountInt(i), y: 0}, pin_map[i])
	}
}

// Zig: test "Screen plain multiline"
func TestFm1_Screen_plain_multiline(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)

	fm1Next(t, s, "hello\r\nworld")

	var pin_map []PageListPin
	// Zig-only: defer pin_map.deinit(alloc);

	formatter := FormatterScreenFormatterInit(term.screens.active, FormatterOptionsPlain)
	formatter.pin_map = Some(FormatterPinMap{map_: &pin_map})

	zigTry(t, formatter.Format(&builder.writer))
	output := builder.writer.buffered()
	expectEqualStrings(t, "hello\nworld", string(output))

	// Verify pin map
	expectEqual(t, len(output), len(pin_map))
	node := fm1UnwrapPtr(t, term.screens.active.pages.pages.first)
	// "hello" (5 chars)
	for i := 0; i < 5; i++ {
		expectEqual(t, node, pin_map[i].node)
		expectEqual(t, SizeCellCountInt(i), pin_map[i].x)
		expectEqual(t, SizeCellCountInt(0), pin_map[i].y)
	}
	// "\n" maps to end of first line
	expectEqual(t, node, pin_map[5].node)
	// "world" (5 chars)
	for i := 0; i < 5; i++ {
		idx := 6 + i
		expectEqual(t, node, pin_map[idx].node)
		expectEqual(t, SizeCellCountInt(i), pin_map[idx].x)
		expectEqual(t, SizeCellCountInt(1), pin_map[idx].y)
	}
}

// Go-only: panic test: PageFormatter.formatStyleOpen: emit is plain (`.plain => unreachable`)
func TestFm1_panic_PageFormatter_formatStyleOpen_plain(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, _ := fm1Terminal(t, 80, 24)
	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	style := Style{}
	fm1ExpectPanic(t, func() {
		_ = formatter.formatStyleOpen(&builder.writer, &style)
	})
}

// Go-only: panic test: PageFormatter.formatHyperlinkOpen: emit is plain (`.plain, .vt => unreachable`)
func TestFm1_panic_PageFormatter_formatHyperlinkOpen_plain(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, _ := fm1Terminal(t, 80, 24)
	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	fm1ExpectPanic(t, func() {
		_ = formatter.formatHyperlinkOpen(&builder.writer, []uint8("https://example.com"))
	})
}

// Go-only: panic test: PageFormatter.formatHyperlinkOpen: emit is vt (`.plain, .vt => unreachable`)
func TestFm1_panic_PageFormatter_formatHyperlinkOpen_vt(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, _ := fm1Terminal(t, 80, 24)
	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsVt)
	fm1ExpectPanic(t, func() {
		_ = formatter.formatHyperlinkOpen(&builder.writer, []uint8("https://example.com"))
	})
}

// Go-only: panic test: PageFormatter.formatStyleClose: point_map set and empty (`assert(m.map.items.len > 0)`)
func TestFm1_panic_PageFormatter_formatStyleClose_empty_point_map(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, _ := fm1Terminal(t, 80, 24)
	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsVt)
	var point_map []PointCoordinate
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})
	fm1ExpectPanic(t, func() {
		_ = formatter.formatStyleClose(&builder.writer)
	})
}

// Go-only: panic test: PageFormatter.formatHyperlinkClose: html with point_map set and empty (`assert(m.map.items.len > 0)`)
func TestFm1_panic_PageFormatter_formatHyperlinkClose_empty_point_map(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, _ := fm1Terminal(t, 80, 24)
	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsHtml)
	var point_map []PointCoordinate
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})
	fm1ExpectPanic(t, func() {
		_ = formatter.formatHyperlinkClose(&builder.writer)
	})
}

// Go-only: panic test: PageFormatter.writeCodepointWithReplacement: string replacement that is not valid UTF-8 (`Utf8View.init(s) catch unreachable`)
func TestFm1_panic_PageFormatter_writeCodepointWithReplacement_invalid_utf8(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)
	fm1Next(t, s, "a")
	page := fm1LastPage(t, term)

	opts := FormatterOptionsPlain
	opts.codepoint_map = Some([]FormatterCodepointMap{{
		range_: [2]uint32{'a', 'a'},
		replacement: FormatterCodepointMapReplacement{
			Tag:    FormatterCodepointMapReplacementString,
			String: []uint8{0xff},
		},
	}})
	formatter := FormatterPageFormatterInit(page, opts)
	fm1ExpectPanic(t, func() {
		_ = formatter.Format(&builder.writer)
	})
}

// Go-only: panic test: PageFormatter.writeCell: codepoint_grapheme cell without grapheme data (`self.page.lookupGrapheme(cell).?`)
func TestFm1_panic_PageFormatter_writeCell_grapheme_without_data(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)
	fm1Next(t, s, "a")
	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	rac := page.GetRowAndCell(0, 0)
	fm1ExpectPanic(t, func() {
		_ = formatter.writeCell(PageCellContentTagCodepointGrapheme, &builder.writer, rac.cell)
	})
}

// Go-only: panic test: PageFormatter.formatWithState: rectangle with start_x > end_x on more than one row (slice `cells[row_start_x..row_end_x]` out of order)
func TestFm1_panic_PageFormatter_formatWithState_rectangle_start_x_after_end_x(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, s := fm1Terminal(t, 80, 24)
	fm1Next(t, s, "hello\r\nworld")
	page := fm1LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	formatter.start_x = 5
	formatter.end_x = Some(SizeCellCountInt(2))
	formatter.end_y = Some(SizeCellCountInt(1))
	formatter.rectangle = true
	fm1ExpectPanic(t, func() {
		_, _ = formatter.FormatWithState(&builder.writer)
	})
}

// Go-only: panic test: ScreenFormatter.format: charsets extra with gr set to G0 (`.G0 => unreachable, // GR can't be G0`)
func TestFm1_panic_ScreenFormatter_format_charset_gr_G0(t *testing.T) {
	testStdCapacity(t)
	builder := fm1Builder(t)
	term, _ := fm1Terminal(t, 80, 24)

	term.screens.active.charset.gr = CharsetsSlotsG0
	formatter := FormatterScreenFormatterInit(term.screens.active, FormatterOptionsVt)
	formatter.extra.charsets = true
	fm1ExpectPanic(t, func() {
		_ = formatter.Format(&builder.writer)
	})
}
