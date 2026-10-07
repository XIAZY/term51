// Tests translated one for one from src/terminal/formatter.zig (Ghostty
// v1.3.1 plus the term51 additional tests), lines 4544 to 7125: the
// ScreenFormatter and TerminalFormatter extras tests, the PageFormatter HTML,
// palette and codepoint_map tests, and the term51 tests of Format.styled, the
// init functions, ScreenFormatter.Extra.isSet, writeCodepoint,
// writeCodepointWithReplacement, writeCell, cellStyle, formatStyleOpen,
// formatStyleClose and formatHyperlinkOpen. Each Go test carries the exact Zig
// title in a "// Zig: test" comment. Helpers are prefixed fm2.
//
// Translation notes:
//   - The testing parameter is `t`; the Terminal that Zig's tests call `t` is
//     called `term`. `try x` on a call that returns an error is zigTry;
//     `x.?` is zigUnwrap (or a nil check for a pointer).
//   - `var builder: std.Io.Writer.Allocating = .init(alloc)` is
//     `builder := oscWriterAllocatingInit()`; `builder.writer.buffered()` is
//     the same call on the Go writer; `output` is converted with string() for
//     comparisons. `builder.clearRetainingCapacity()` is fm2Clear (the wireframe
//     has no such method: Zig's body is `writer.end = 0`).
//   - Zig's `.{ .emit = .html }` and the presets are FormatterOptionsHtml and
//     so on; `opts.codepoint_map = map` is `Some(map)`, and
//     `std.MultiArrayList(CodepointMap)` is a `[]FormatterCodepointMap`
//     (fm2MapInit; fm2ReplCodepoint and fm2ReplString build the two
//     Replacement variants).
//   - `.{ .alloc = alloc, .map = &x }` is `Some(FormatterPinMap{map_: &x})`
//     (and FormatterPageFormatterPointMap for the page formatter's point map).
//   - `term51Z*` helpers are the fm2 helpers below. `inline for` loops are
//     ordinary loops over the written-out values; the field loop of the
//     `ScreenFormatter.Extra.isSet` test is the table of the six fields.
//   - Every test calls testStdCapacity first. Leak-check lines
//     (`defer x.deinit(alloc)` of lists and maps) are dropped with
//     `// Zig-only:` comments.
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt (none in this range).

package terminal

import (
	"reflect"
	"strings"
	"testing"
)

// fm2ExpectPanic runs f, which must panic. A panic with the wireframe's
// "not implemented" value does not count: it only means the function under
// test does not exist yet. The setup is done outside f.
func fm2ExpectPanic(t *testing.T, f func()) {
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

// fm2Init is `try Terminal.init(alloc, .{ .cols = cols, .rows = rows })`;
// the terminal is returned by pointer (the methods take a pointer receiver).
func fm2Init(tt *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) *Terminal {
	tt.Helper()
	t, err := TerminalInit(terminalOptionsDefault(cols, rows))
	zigTry(tt, err)
	return &t
}

// fm2Active is the Zig literal `.{ .active = .{ .x = x, .y = y } }`.
func fm2Active(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagActive, Active: PointCoordinate{X: x, Y: y}}
}

// fm2Clear is `builder.clearRetainingCapacity()`.
func fm2Clear(b *oscWriterAllocating) {
	b.clearRetainingCapacity()
}

// fm2FirstNode is `t.screens.active.pages.pages.first.?`.
func fm2FirstNode(tt *testing.T, term *Terminal) *PageListNode {
	tt.Helper()
	node := term.Screens.Active.Pages.pages.first
	expect(tt, node != nil, "t.screens.active.pages.pages.first.?")
	return node
}

// fm2LastPage is `&t.screens.active.pages.pages.last.?.data`.
func fm2LastPage(tt *testing.T, term *Terminal) *Page {
	tt.Helper()
	node := term.Screens.Active.Pages.pages.last
	expect(tt, node != nil, "pages.pages.last.?")
	return &node.data
}

// fm2Write is term51ZWrite: feed data to the terminal through a vtStream.
func fm2Write(tt *testing.T, term *Terminal, data string) {
	tt.Helper()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(tt, s.NextSlice([]uint8(data)))
}

// fm2Page is term51ZPage: `&t.screens.active.pages.pages.first.?.data`.
func fm2Page(tt *testing.T, term *Terminal) *Page {
	tt.Helper()
	return &fm2FirstNode(tt, term).data
}

// fm2Formatter is "any formatter that has a format method".
type fm2Formatter interface {
	format(writer *oscWriter) error
}

// fm2PageOut is term51ZPageOut: format the first page of the terminal with
// a PageFormatter and return the output.
func fm2PageOut(tt *testing.T, term *Terminal, opts FormatterOptions) []uint8 {
	tt.Helper()
	builder := oscWriterAllocatingInit()
	defer builder.deinit()
	f := FormatterPageFormatterInit(fm2Page(tt, term), opts)
	zigTry(tt, f.format(&builder.writer))
	return append([]uint8(nil), builder.writer.buffered()...)
}

// fm2Out is term51ZOut: format with any formatter that has a format method
// and return the output.
func fm2Out(tt *testing.T, f fm2Formatter) []uint8 {
	tt.Helper()
	builder := oscWriterAllocatingInit()
	defer builder.deinit()
	zigTry(tt, f.format(&builder.writer))
	return append([]uint8(nil), builder.writer.buffered()...)
}

// fm2ReplCodepoint is `.{ .codepoint = cp }` of CodepointMap.Replacement.
func fm2ReplCodepoint(cp uint32) FormatterCodepointMapReplacement {
	return FormatterCodepointMapReplacement{Tag: FormatterCodepointMapReplacementCodepoint, Codepoint: cp}
}

// fm2ReplString is `.{ .string = s }` of CodepointMap.Replacement.
func fm2ReplString(s string) FormatterCodepointMapReplacement {
	return FormatterCodepointMapReplacement{Tag: FormatterCodepointMapReplacementString, String: []uint8(s)}
}

// fm2Entry is `.{ .range = .{ lo, hi }, .replacement = r }`.
func fm2Entry(lo uint32, hi uint32, r FormatterCodepointMapReplacement) FormatterCodepointMap {
	return FormatterCodepointMap{range_: [2]uint32{lo, hi}, replacement: r}
}

// fm2MapInit is term51ZMapInit: a codepoint map made of the entries.
func fm2MapInit(entries []FormatterCodepointMap) []FormatterCodepointMap {
	var m []FormatterCodepointMap
	for _, e := range entries {
		m = append(m, e)
	}
	return m
}

// fm2Opts is `.{ .emit = emit }` with the Zig defaults of Options.
func fm2Opts(emit FormatterFormat) FormatterOptions {
	return FormatterOptions{
		Emit:         emit,
		Trim:         true,
		CodepointMap: Optional[[]FormatterCodepointMap]{Valid: true},
	}
}

// fm2ExpectCp is term51ZExpectCp.
func fm2ExpectCp(tt *testing.T, f FormatterPageFormatter, cp uint32, expected string) {
	tt.Helper()
	b := oscWriterAllocatingInit()
	defer b.deinit()
	zigTry(tt, f.writeCodepoint(&b.writer, cp))
	expectEqualStrings(tt, expected, string(b.writer.buffered()))
}

// fm2ExpectReplaced is term51ZExpectReplaced.
func fm2ExpectReplaced(tt *testing.T, f FormatterPageFormatter, cp uint32, expected string) {
	tt.Helper()
	b := oscWriterAllocatingInit()
	defer b.deinit()
	zigTry(tt, f.writeCodepointWithReplacement(&b.writer, cp))
	expectEqualStrings(tt, expected, string(b.writer.buffered()))
}

// fm2ExpectStyleOpen is term51ZExpectStyleOpen.
func fm2ExpectStyleOpen(tt *testing.T, f FormatterPageFormatter, style Style, expected string) {
	tt.Helper()
	b := oscWriterAllocatingInit()
	defer b.deinit()
	zigTry(tt, f.formatStyleOpen(&b.writer, &style))
	expectEqualStrings(tt, expected, string(b.writer.buffered()))
}

// fm2Cell is term51ZCell: `term51ZPage(t).getRowAndCell(x, y).cell`.
func fm2Cell(tt *testing.T, term *Terminal, x uint, y uint) *PageCell {
	tt.Helper()
	return fm2Page(tt, term).GetRowAndCell(x, y).cell
}

// fm2Palette is the Zig literal `.{ .palette = n }` of Style.Color.
func fm2Palette(n uint8) StyleColor {
	return StyleColor{Tag: StyleColorPalette, Palette: n}
}

// fm2Rgb is the Zig literal `.{ .rgb = .{ .r = r, .g = g, .b = b } }` of
// Style.Color.
func fm2Rgb(r uint8, g uint8, b uint8) StyleColor {
	return StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{R: r, G: g, B: b}}
}

// Zig: test "Screen plain with selection"
func TestFm2_Screen_plain_with_selection(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("line1\r\nline2\r\nline3")))

	var pin_map []PageListPin
	// Zig-only: defer pin_map.deinit(alloc);

	formatter := FormatterScreenFormatterInit(term.Screens.Active, FormatterOptionsPlain)
	formatter.content = FormatterScreenFormatterContent{
		Tag: FormatterScreenFormatterContentSelection,
		Selection: Some(SelectionInit(
			zigUnwrap(t, term.Screens.Active.Pages.Pin(fm2Active(0, 1))),
			zigUnwrap(t, term.Screens.Active.Pages.Pin(fm2Active(4, 1))),
			false,
		)),
	}
	formatter.pin_map = Some(FormatterPinMap{map_: &pin_map})

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()
	expectEqualStrings(t, "line2", string(output))

	// Verify pin map
	expectEqual(t, len(output), len(pin_map))
	node := fm2FirstNode(t, term)
	// "line2" (5 chars) from row 1
	for i := 0; i < 5; i++ {
		expectEqual(t, node, pin_map[i].Node)
		expectEqual(t, SizeCellCountInt(i), pin_map[i].X)
		expectEqual(t, SizeCellCountInt(1), pin_map[i].Y)
	}
}

// Zig: test "Screen vt with cursor position"
func TestFm2_Screen_vt_with_cursor_position(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Position cursor at a specific location
	zigTry(t, s.NextSlice([]uint8("hello\r\nworld")))

	var pin_map []PageListPin
	// Zig-only: defer pin_map.deinit(alloc);

	formatter := FormatterScreenFormatterInit(term.Screens.Active, FormatterOptionsVt)
	formatter.extra.Cursor = true
	formatter.pin_map = Some(FormatterPinMap{map_: &pin_map})

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	// Create a second terminal and apply the output
	term2 := fm2Init(t, 80, 24)
	defer term2.Deinit()

	s2 := term2.VtStream()
	defer s2.Deinit()

	zigTry(t, s2.NextSlice(output))

	// Verify cursor positions match
	expectEqual(t, term.Screens.Active.Cursor.X, term2.Screens.Active.Cursor.X)
	expectEqual(t, term.Screens.Active.Cursor.Y, term2.Screens.Active.Cursor.Y)

	// Verify pin map - the extras should be mapped to the last pin
	expectEqual(t, len(output), len(pin_map))
	node := fm2FirstNode(t, term)
	content_len := len("hello\r\nworld")
	// Content bytes map to their positions
	for i := 0; i < content_len; i++ {
		expectEqual(t, node, pin_map[i].Node)
	}
	// Extra bytes (cursor position) map to last content pin
	for i := content_len; i < len(output); i++ {
		expectEqual(t, node, pin_map[i].Node)
	}
}

// Zig: test "Screen vt with style"
func TestFm2_Screen_vt_with_style(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Set some style attributes
	zigTry(t, s.NextSlice([]uint8("\x1b[1;31mhello")))

	var pin_map []PageListPin
	// Zig-only: defer pin_map.deinit(alloc);

	formatter := FormatterScreenFormatterInit(term.Screens.Active, FormatterOptionsVt)
	formatter.extra.Style = true
	formatter.pin_map = Some(FormatterPinMap{map_: &pin_map})

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	// Create a second terminal and apply the output
	term2 := fm2Init(t, 80, 24)
	defer term2.Deinit()

	s2 := term2.VtStream()
	defer s2.Deinit()

	zigTry(t, s2.NextSlice(output))

	// Verify styles match
	expect(t, term.Screens.Active.Cursor.Style.Eql(term2.Screens.Active.Cursor.Style),
		"t.screens.active.cursor.style.eql(t2.screens.active.cursor.style)")

	// Verify pin map
	expectEqual(t, len(output), len(pin_map))
	node := fm2FirstNode(t, term)
	for i := 0; i < len(output); i++ {
		expectEqual(t, node, pin_map[i].Node)
	}
}

// Zig: test "Screen vt with hyperlink"
func TestFm2_Screen_vt_with_hyperlink(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Set a hyperlink
	zigTry(t, s.NextSlice([]uint8("\x1b]8;;http://example.com\x1b\\hello")))

	var pin_map []PageListPin
	// Zig-only: defer pin_map.deinit(alloc);

	formatter := FormatterScreenFormatterInit(term.Screens.Active, FormatterOptionsVt)
	formatter.extra.Hyperlink = true
	formatter.pin_map = Some(FormatterPinMap{map_: &pin_map})

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	// Create a second terminal and apply the output
	term2 := fm2Init(t, 80, 24)
	defer term2.Deinit()

	s2 := term2.VtStream()
	defer s2.Deinit()

	zigTry(t, s2.NextSlice(output))

	// Verify hyperlinks match
	has_link1 := term.Screens.Active.Cursor.Hyperlink != nil
	has_link2 := term2.Screens.Active.Cursor.Hyperlink != nil
	expectEqual(t, has_link1, has_link2)

	if has_link1 {
		link1 := term.Screens.Active.Cursor.Hyperlink
		link2 := term2.Screens.Active.Cursor.Hyperlink
		expect(t, link2 != nil, "t2.screens.active.cursor.hyperlink.?")
		expectEqualStrings(t, string(link1.URI), string(link2.URI))
	}

	// Verify pin map
	expectEqual(t, len(output), len(pin_map))
	node := fm2FirstNode(t, term)
	for i := 0; i < len(output); i++ {
		expectEqual(t, node, pin_map[i].Node)
	}
}

// Zig: test "Screen vt with protection"
func TestFm2_Screen_vt_with_protection(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Enable protection mode
	zigTry(t, s.NextSlice([]uint8("\x1b[1\"qhello")))

	var pin_map []PageListPin
	// Zig-only: defer pin_map.deinit(alloc);

	formatter := FormatterScreenFormatterInit(term.Screens.Active, FormatterOptionsVt)
	formatter.extra.Protection = true
	formatter.pin_map = Some(FormatterPinMap{map_: &pin_map})

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	// Create a second terminal and apply the output
	term2 := fm2Init(t, 80, 24)
	defer term2.Deinit()

	s2 := term2.VtStream()
	defer s2.Deinit()

	zigTry(t, s2.NextSlice(output))

	// Verify protection state matches
	expectEqual(t, term.Screens.Active.Cursor.Protected, term2.Screens.Active.Cursor.Protected)

	// Verify pin map
	expectEqual(t, len(output), len(pin_map))
	node := fm2FirstNode(t, term)
	for i := 0; i < len(output); i++ {
		expectEqual(t, node, pin_map[i].Node)
	}
}

// Zig: test "Screen vt with kitty keyboard"
func TestFm2_Screen_vt_with_kitty_keyboard(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Set kitty keyboard flags (disambiguate + report_events = 3)
	zigTry(t, s.NextSlice([]uint8("\x1b[=3;1uhello")))

	var pin_map []PageListPin
	// Zig-only: defer pin_map.deinit(alloc);

	formatter := FormatterScreenFormatterInit(term.Screens.Active, FormatterOptionsVt)
	formatter.extra.KittyKeyboard = true
	formatter.pin_map = Some(FormatterPinMap{map_: &pin_map})

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	// Create a second terminal and apply the output
	term2 := fm2Init(t, 80, 24)
	defer term2.Deinit()

	s2 := term2.VtStream()
	defer s2.Deinit()

	zigTry(t, s2.NextSlice(output))

	// Verify kitty keyboard state matches
	flags1 := term.Screens.Active.KittyKeyboard.Current().Int()
	flags2 := term2.Screens.Active.KittyKeyboard.Current().Int()
	expectEqual(t, flags1, flags2)

	// Verify pin map
	expectEqual(t, len(output), len(pin_map))
	node := fm2FirstNode(t, term)
	for i := 0; i < len(output); i++ {
		expectEqual(t, node, pin_map[i].Node)
	}
}

// Zig: test "Screen vt with charsets"
func TestFm2_Screen_vt_with_charsets(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Set G0 to DEC special and shift to G1
	zigTry(t, s.NextSlice([]uint8("\x1b(0\x0ehello")))

	var pin_map []PageListPin
	// Zig-only: defer pin_map.deinit(alloc);

	formatter := FormatterScreenFormatterInit(term.Screens.Active, FormatterOptionsVt)
	formatter.extra.Charsets = true
	formatter.pin_map = Some(FormatterPinMap{map_: &pin_map})

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	// Create a second terminal and apply the output
	term2 := fm2Init(t, 80, 24)
	defer term2.Deinit()

	s2 := term2.VtStream()
	defer s2.Deinit()

	zigTry(t, s2.NextSlice(output))

	// Verify charset state matches
	expectEqual(t, term.Screens.Active.Charset.Gl, term2.Screens.Active.Charset.Gl)
	expectEqual(t, term.Screens.Active.Charset.Gr, term2.Screens.Active.Charset.Gr)
	expectEqual(t,
		term.Screens.Active.Charset.charsets.Get(CharsetsSlotsG0),
		term2.Screens.Active.Charset.charsets.Get(CharsetsSlotsG0),
	)

	// Verify pin map
	expectEqual(t, len(output), len(pin_map))
	node := fm2FirstNode(t, term)
	for i := 0; i < len(output); i++ {
		expectEqual(t, node, pin_map[i].Node)
	}
}

// Go-only: panic test: ScreenFormatter.format: charsets extra with the GR slot G0 ("GR can't be G0" is unreachable in Zig)
func TestFm2_panic_ScreenFormatter_format_charsets_gr_is_G0(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	term.Screens.Active.Charset.Gr = CharsetsSlotsG0

	formatter := FormatterScreenFormatterInit(term.Screens.Active, FormatterOptionsVt)
	formatter.extra.Charsets = true

	fm2ExpectPanic(t, func() {
		_ = formatter.format(&builder.writer)
	})
}

// Zig: test "Terminal vt with scrolling region"
func TestFm2_Terminal_vt_with_scrolling_region(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Set scrolling region: top=5, bottom=20
	zigTry(t, s.NextSlice([]uint8("\x1b[6;21rhello")))

	formatter := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
	formatter.extra.ScrollingRegion = true

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	// Create a second terminal and apply the output
	term2 := fm2Init(t, 80, 24)
	defer term2.Deinit()

	s2 := term2.VtStream()
	defer s2.Deinit()

	zigTry(t, s2.NextSlice(output))

	// Verify scrolling regions match
	expectEqual(t, term.ScrollingRegion.Top, term2.ScrollingRegion.Top)
	expectEqual(t, term.ScrollingRegion.Bottom, term2.ScrollingRegion.Bottom)
	expectEqual(t, term.ScrollingRegion.Left, term2.ScrollingRegion.Left)
	expectEqual(t, term.ScrollingRegion.Right, term2.ScrollingRegion.Right)
}

// Zig: test "Terminal vt with modes"
func TestFm2_Terminal_vt_with_modes(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Enable some modes that differ from defaults
	zigTry(t, s.NextSlice([]uint8("\x1b[?2004h"))) // Bracketed paste
	zigTry(t, s.NextSlice([]uint8("\x1b[?1000h"))) // Mouse event normal
	zigTry(t, s.NextSlice([]uint8("\x1b[?7l")))    // Disable wraparound (default is true)
	zigTry(t, s.NextSlice([]uint8("hello")))

	formatter := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
	formatter.extra.Modes = true

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	// Create a second terminal and apply the output
	term2 := fm2Init(t, 80, 24)
	defer term2.Deinit()

	s2 := term2.VtStream()
	defer s2.Deinit()

	zigTry(t, s2.NextSlice(output))

	// Verify modes match
	expectEqual(t, term.Modes.Get(ModesModeBracketedPaste), term2.Modes.Get(ModesModeBracketedPaste))
	expectEqual(t, term.Modes.Get(ModesModeMouseEventNormal), term2.Modes.Get(ModesModeMouseEventNormal))
	expectEqual(t, term.Modes.Get(ModesModeWraparound), term2.Modes.Get(ModesModeWraparound))
}

// Zig: test "Terminal vt with tabstops"
func TestFm2_Terminal_vt_with_tabstops(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Clear all tabs and set custom tabstops
	zigTry(t, s.NextSlice([]uint8("\x1b[3g")))       // Clear all tabs
	zigTry(t, s.NextSlice([]uint8("\x1b[5G\x1bH")))  // Set tab at column 5
	zigTry(t, s.NextSlice([]uint8("\x1b[15G\x1bH"))) // Set tab at column 15
	zigTry(t, s.NextSlice([]uint8("\x1b[30G\x1bH"))) // Set tab at column 30
	zigTry(t, s.NextSlice([]uint8("hello")))

	formatter := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
	formatter.extra.Tabstops = true

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	// Create a second terminal and apply the output
	term2 := fm2Init(t, 80, 24)
	defer term2.Deinit()

	s2 := term2.VtStream()
	defer s2.Deinit()

	zigTry(t, s2.NextSlice(output))

	// Verify tabstops match (columns are 0-indexed in the API)
	expectEqual(t, term.Tabstops.Get(4), term2.Tabstops.Get(4))
	expectEqual(t, term.Tabstops.Get(14), term2.Tabstops.Get(14))
	expectEqual(t, term.Tabstops.Get(29), term2.Tabstops.Get(29))
	expect(t, term2.Tabstops.Get(4), "t2.tabstops.get(4)")   // Column 5 (1-indexed)
	expect(t, term2.Tabstops.Get(14), "t2.tabstops.get(14)") // Column 15 (1-indexed)
	expect(t, term2.Tabstops.Get(29), "t2.tabstops.get(29)") // Column 30 (1-indexed)
	expect(t, !term2.Tabstops.Get(8), "!t2.tabstops.get(8)") // Not a tab
}

// Zig: test "Terminal vt with keyboard modes"
func TestFm2_Terminal_vt_with_keyboard_modes(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Set modify other keys mode 2
	zigTry(t, s.NextSlice([]uint8("\x1b[>4;2m")))
	zigTry(t, s.NextSlice([]uint8("hello")))

	formatter := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
	formatter.extra.Keyboard = true

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	// Create a second terminal and apply the output
	term2 := fm2Init(t, 80, 24)
	defer term2.Deinit()

	s2 := term2.VtStream()
	defer s2.Deinit()

	zigTry(t, s2.NextSlice(output))

	// Verify keyboard mode matches
	expectEqual(t, term.Flags.ModifyOtherKeys2, term2.Flags.ModifyOtherKeys2)
	expect(t, term2.Flags.ModifyOtherKeys2, "t2.flags.modify_other_keys_2")
}

// Zig: test "Terminal vt with pwd"
func TestFm2_Terminal_vt_with_pwd(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Set pwd using OSC 7
	zigTry(t, s.NextSlice([]uint8("\x1b]7;file://host/home/user\x1b\\hello")))

	formatter := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
	formatter.extra.Pwd = true

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	// Create a second terminal and apply the output
	term2 := fm2Init(t, 80, 24)
	defer term2.Deinit()

	s2 := term2.VtStream()
	defer s2.Deinit()

	zigTry(t, s2.NextSlice(output))

	// Verify pwd matches
	expectEqualStrings(t, string(term.Pwd), string(term2.Pwd))
}

// Zig: test "Page html with multiple styles"
func TestFm2_Page_html_with_multiple_styles(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Set bold, then italic, then reset
	zigTry(t, s.NextSlice([]uint8("\x1b[1mbold\x1b[3mitalic\x1b[0mnormal")))

	page := fm2LastPage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsHtml)

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	expectEqualStrings(t,
		"<div style=\"font-family: monospace; white-space: pre;\">"+
			"<div style=\"display: inline;font-weight: bold;\">bold</div>"+
			"<div style=\"display: inline;font-weight: bold;font-style: italic;\">italic</div>"+
			"normal"+
			"</div>",
		string(output),
	)
}

// Zig: test "Page html plain text"
func TestFm2_Page_html_plain_text(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("hello, world")))

	page := fm2LastPage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsHtml)

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	// Plain text without styles should be wrapped in monospace div
	expectEqualStrings(t,
		"<div style=\"font-family: monospace; white-space: pre;\">hello, world</div>",
		string(output),
	)
}

// Zig: test "Page html with colors"
func TestFm2_Page_html_with_colors(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Set red foreground, blue background
	zigTry(t, s.NextSlice([]uint8("\x1b[31;44mcolored")))

	page := fm2LastPage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsHtml)

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	expectEqualStrings(t,
		"<div style=\"font-family: monospace; white-space: pre;\">"+
			"<div style=\"display: inline;color: var(--vt-palette-1);background-color: var(--vt-palette-4);\">colored</div>"+
			"</div>",
		string(output),
	)
}

// Zig: test "TerminalFormatter html with palette"
func TestFm2_TerminalFormatter_html_with_palette(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Modify some palette colors
	zigTry(t, s.NextSlice([]uint8("\x1b]4;0;rgb:12/34/56\x1b\\")))
	zigTry(t, s.NextSlice([]uint8("\x1b]4;1;rgb:ab/cd/ef\x1b\\")))
	zigTry(t, s.NextSlice([]uint8("\x1b]4;255;rgb:ff/00/ff\x1b\\")))
	zigTry(t, s.NextSlice([]uint8("test")))

	formatter := FormatterTerminalFormatterInit(term, FormatterOptionsHtml)
	formatter.extra.Palette = true

	zigTry(t, formatter.format(&builder.writer))
	output := string(builder.writer.buffered())

	// Verify palette CSS variables are emitted
	expect(t, strings.Contains(output, "<style>:root{"), "std.mem.indexOf(u8, output, \"<style>:root{\") != null")
	expect(t, strings.Contains(output, "--vt-palette-0: #123456;"), "std.mem.indexOf(u8, output, \"--vt-palette-0: #123456;\") != null")
	expect(t, strings.Contains(output, "--vt-palette-1: #abcdef;"), "std.mem.indexOf(u8, output, \"--vt-palette-1: #abcdef;\") != null")
	expect(t, strings.Contains(output, "--vt-palette-255: #ff00ff;"), "std.mem.indexOf(u8, output, \"--vt-palette-255: #ff00ff;\") != null")
	expect(t, strings.Contains(output, "}</style>"), "std.mem.indexOf(u8, output, \"}</style>\") != null")
	expect(t, strings.Contains(output, "test"), "std.mem.indexOf(u8, output, \"test\") != null")
}

// Zig: test "Page html with background and foreground colors"
func TestFm2_Page_html_with_background_and_foreground_colors(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("hello")))

	page := fm2LastPage(t, term)
	opts := FormatterOptionsHtml
	opts.Background = Some(ColorRGB{R: 0x12, G: 0x34, B: 0x56})
	opts.Foreground = Some(ColorRGB{R: 0xab, G: 0xcd, B: 0xef})
	formatter := FormatterPageFormatterInit(page, opts)

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	expectEqualStrings(t,
		"<div style=\"font-family: monospace; white-space: pre;background-color: #123456;color: #abcdef;\">hello</div>",
		string(output),
	)
}

// Zig: test "Page html with escaping"
func TestFm2_Page_html_with_escaping(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("<tag>&\"'text")))

	page := fm2LastPage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsHtml)

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	expectEqualStrings(t,
		"<div style=\"font-family: monospace; white-space: pre;\">&lt;tag&gt;&amp;&quot;&#39;text</div>",
		string(output),
	)

	// Verify point map length matches output
	expectEqual(t, len(output), len(point_map))

	// Opening wrapper div
	wrapper_start := "<div style=\"font-family: monospace; white-space: pre;\">"
	wrapper_start_len := len(wrapper_start)
	for i := 0; i < wrapper_start_len; i++ {
		expectEqual(t, PointCoordinate{X: 0, Y: 0}, point_map[i])
	}

	// Verify each character maps correctly, accounting for escaping
	offset := wrapper_start_len
	// < (4 bytes: &lt;) -> x=0
	for i := 0; i < 4; i++ {
		expectEqual(t, PointCoordinate{X: 0, Y: 0}, point_map[offset+i])
	}
	// t (1 byte) -> x=1
	expectEqual(t, PointCoordinate{X: 1, Y: 0}, point_map[offset+4])
	// a (1 byte) -> x=2
	expectEqual(t, PointCoordinate{X: 2, Y: 0}, point_map[offset+5])
	// g (1 byte) -> x=3
	expectEqual(t, PointCoordinate{X: 3, Y: 0}, point_map[offset+6])
	// > (4 bytes: &gt;) -> x=4
	for i := 0; i < 4; i++ {
		expectEqual(t, PointCoordinate{X: 4, Y: 0}, point_map[offset+7+i])
	}
	// & (5 bytes: &amp;) -> x=5
	for i := 0; i < 5; i++ {
		expectEqual(t, PointCoordinate{X: 5, Y: 0}, point_map[offset+11+i])
	}
	// " (6 bytes: &quot;) -> x=6
	for i := 0; i < 6; i++ {
		expectEqual(t, PointCoordinate{X: 6, Y: 0}, point_map[offset+16+i])
	}
	// ' (5 bytes: &#39;) -> x=7
	for i := 0; i < 5; i++ {
		expectEqual(t, PointCoordinate{X: 7, Y: 0}, point_map[offset+22+i])
	}
	// t (1 byte) -> x=8
	expectEqual(t, PointCoordinate{X: 8, Y: 0}, point_map[offset+27])
	// e (1 byte) -> x=9
	expectEqual(t, PointCoordinate{X: 9, Y: 0}, point_map[offset+28])
	// x (1 byte) -> x=10
	expectEqual(t, PointCoordinate{X: 10, Y: 0}, point_map[offset+29])
	// t (1 byte) -> x=11
	expectEqual(t, PointCoordinate{X: 11, Y: 0}, point_map[offset+30])
}

// Zig: test "Page html with unicode as numeric entities"
func TestFm2_Page_html_with_unicode_as_numeric_entities(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Box drawing characters that caused issue #9426
	zigTry(t, s.NextSlice([]uint8("╰─ ❯")))

	page := fm2LastPage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsHtml)

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	// Expected: box drawing chars as numeric entities
	// ╰ = U+2570 = 9584, ─ = U+2500 = 9472, ❯ = U+276F = 10095
	expectEqualStrings(t,
		"<div style=\"font-family: monospace; white-space: pre;\">&#9584;&#9472; &#10095;</div>",
		string(output),
	)
}

// Zig: test "Page html ascii characters unchanged"
func TestFm2_Page_html_ascii_characters_unchanged(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("hello world")))

	page := fm2LastPage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsHtml)

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	// ASCII should be emitted directly
	expectEqualStrings(t,
		"<div style=\"font-family: monospace; white-space: pre;\">hello world</div>",
		string(output),
	)
}

// Zig: test "Page html mixed ascii and unicode"
func TestFm2_Page_html_mixed_ascii_and_unicode(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("test ╰─❯ ok")))

	page := fm2LastPage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsHtml)

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	// Mix of ASCII and Unicode entities
	expectEqualStrings(t,
		"<div style=\"font-family: monospace; white-space: pre;\">test &#9584;&#9472;&#10095; ok</div>",
		string(output),
	)
}

// Zig: test "Page VT with palette option emits RGB"
func TestFm2_Page_VT_with_palette_option_emits_RGB(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Set a custom palette color and use it
	zigTry(t, s.NextSlice([]uint8("\x1b]4;1;rgb:ab/cd/ef\x1b\\")))
	zigTry(t, s.NextSlice([]uint8("\x1b[31mred")))

	page := fm2LastPage(t, term)

	// Without palette option - should emit palette index
	{
		fm2Clear(&builder)
		formatter := FormatterPageFormatterInit(page, FormatterOptionsVt)
		zigTry(t, formatter.format(&builder.writer))
		output := builder.writer.buffered()
		expectEqualStrings(t, "\x1b[0m\x1b[38;5;1mred\x1b[0m", string(output))
	}

	// With palette option - should emit RGB directly
	{
		fm2Clear(&builder)
		opts := FormatterOptionsVt
		opts.Palette = &term.Colors.Palette.Current
		formatter := FormatterPageFormatterInit(page, opts)
		zigTry(t, formatter.format(&builder.writer))
		output := builder.writer.buffered()
		expectEqualStrings(t, "\x1b[0m\x1b[38;2;171;205;239mred\x1b[0m", string(output))
	}
}

// Zig: test "Page html with palette option emits RGB"
func TestFm2_Page_html_with_palette_option_emits_RGB(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Set a custom palette color and use it
	zigTry(t, s.NextSlice([]uint8("\x1b]4;1;rgb:ab/cd/ef\x1b\\")))
	zigTry(t, s.NextSlice([]uint8("\x1b[31mred")))

	page := fm2LastPage(t, term)

	// Without palette option - should emit CSS variable
	{
		fm2Clear(&builder)
		formatter := FormatterPageFormatterInit(page, FormatterOptionsHtml)
		zigTry(t, formatter.format(&builder.writer))
		output := builder.writer.buffered()
		expectEqualStrings(t,
			"<div style=\"font-family: monospace; white-space: pre;\">"+
				"<div style=\"display: inline;color: var(--vt-palette-1);\">red</div>"+
				"</div>",
			string(output),
		)
	}

	// With palette option - should emit RGB directly
	{
		fm2Clear(&builder)
		opts := FormatterOptionsHtml
		opts.Palette = &term.Colors.Palette.Current
		formatter := FormatterPageFormatterInit(page, opts)
		zigTry(t, formatter.format(&builder.writer))
		output := builder.writer.buffered()
		expectEqualStrings(t,
			"<div style=\"font-family: monospace; white-space: pre;\">"+
				"<div style=\"display: inline;color: rgb(171, 205, 239);\">red</div>"+
				"</div>",
			string(output),
		)
	}
}

// Zig: test "Page VT style reset properly closes styles"
func TestFm2_Page_VT_style_reset_properly_closes_styles(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Set bold, then reset with SGR 0
	zigTry(t, s.NextSlice([]uint8("\x1b[1mbold\x1b[0mnormal")))

	page := fm2LastPage(t, term)

	fm2Clear(&builder)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsVt)
	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	// The reset should properly close the bold style
	expectEqualStrings(t, "\x1b[0m\x1b[1mbold\x1b[0mnormal", string(output))
}

// Zig: test "Page codepoint_map single replacement"
func TestFm2_Page_codepoint_map_single_replacement(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("hello world")))

	page := fm2LastPage(t, term)

	// Replace 'o' with 'x'
	var map_ []FormatterCodepointMap
	// Zig-only: defer map.deinit(alloc);
	map_ = append(map_, FormatterCodepointMap{
		range_:      [2]uint32{'o', 'o'},
		replacement: fm2ReplCodepoint('x'),
	})

	opts := FormatterOptionsPlain
	opts.CodepointMap = Some(map_)
	formatter := FormatterPageFormatterInit(page, opts)

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()
	expectEqualStrings(t, "hellx wxrld", string(output))

	// Verify point map - each output byte should map to original cell position
	expectEqual(t, len(output), len(point_map))
	// "hello world" -> "hellx wxrld"
	// h e l l o   w o r l d
	// 0 1 2 3 4 5 6 7 8 9 10
	expectEqual(t, PointCoordinate{X: 0, Y: 0}, point_map[0])   // h
	expectEqual(t, PointCoordinate{X: 1, Y: 0}, point_map[1])   // e
	expectEqual(t, PointCoordinate{X: 2, Y: 0}, point_map[2])   // l
	expectEqual(t, PointCoordinate{X: 3, Y: 0}, point_map[3])   // l
	expectEqual(t, PointCoordinate{X: 4, Y: 0}, point_map[4])   // x (was o)
	expectEqual(t, PointCoordinate{X: 5, Y: 0}, point_map[5])   // space
	expectEqual(t, PointCoordinate{X: 6, Y: 0}, point_map[6])   // w
	expectEqual(t, PointCoordinate{X: 7, Y: 0}, point_map[7])   // x (was o)
	expectEqual(t, PointCoordinate{X: 8, Y: 0}, point_map[8])   // r
	expectEqual(t, PointCoordinate{X: 9, Y: 0}, point_map[9])   // l
	expectEqual(t, PointCoordinate{X: 10, Y: 0}, point_map[10]) // d
}

// Zig: test "Page codepoint_map conflicting replacement prefers last"
func TestFm2_Page_codepoint_map_conflicting_replacement_prefers_last(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("hello")))

	page := fm2LastPage(t, term)

	// Replace 'o' with 'x', then with 'y' - should prefer last
	var map_ []FormatterCodepointMap
	// Zig-only: defer map.deinit(alloc);
	map_ = append(map_, FormatterCodepointMap{
		range_:      [2]uint32{'o', 'o'},
		replacement: fm2ReplCodepoint('x'),
	})
	map_ = append(map_, FormatterCodepointMap{
		range_:      [2]uint32{'o', 'o'},
		replacement: fm2ReplCodepoint('y'),
	})

	opts := FormatterOptionsPlain
	opts.CodepointMap = Some(map_)
	formatter := FormatterPageFormatterInit(page, opts)

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()
	expectEqualStrings(t, "helly", string(output))
}

// Zig: test "Page codepoint_map replace with string"
func TestFm2_Page_codepoint_map_replace_with_string(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("hello")))

	page := fm2LastPage(t, term)

	// Replace 'o' with a multi-byte string
	var map_ []FormatterCodepointMap
	// Zig-only: defer map.deinit(alloc);
	map_ = append(map_, FormatterCodepointMap{
		range_:      [2]uint32{'o', 'o'},
		replacement: fm2ReplString("XYZ"),
	})

	opts := FormatterOptionsPlain
	opts.CodepointMap = Some(map_)
	formatter := FormatterPageFormatterInit(page, opts)

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()
	expectEqualStrings(t, "hellXYZ", string(output))

	// Verify point map - string replacements should all map to the original cell
	expectEqual(t, len(output), len(point_map))
	// "hello" -> "hellXYZ"
	// h e l l o
	// 0 1 2 3 4
	expectEqual(t, PointCoordinate{X: 0, Y: 0}, point_map[0]) // h
	expectEqual(t, PointCoordinate{X: 1, Y: 0}, point_map[1]) // e
	expectEqual(t, PointCoordinate{X: 2, Y: 0}, point_map[2]) // l
	expectEqual(t, PointCoordinate{X: 3, Y: 0}, point_map[3]) // l
	// All bytes of the replacement string "XYZ" should point to position 4 (where 'o' was)
	expectEqual(t, PointCoordinate{X: 4, Y: 0}, point_map[4]) // X
	expectEqual(t, PointCoordinate{X: 4, Y: 0}, point_map[5]) // Y
	expectEqual(t, PointCoordinate{X: 4, Y: 0}, point_map[6]) // Z
}

// Zig: test "Page codepoint_map range replacement"
func TestFm2_Page_codepoint_map_range_replacement(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("abcdefg")))

	page := fm2LastPage(t, term)

	// Replace 'b' through 'e' with 'X'
	var map_ []FormatterCodepointMap
	// Zig-only: defer map.deinit(alloc);
	map_ = append(map_, FormatterCodepointMap{
		range_:      [2]uint32{'b', 'e'},
		replacement: fm2ReplCodepoint('X'),
	})

	opts := FormatterOptionsPlain
	opts.CodepointMap = Some(map_)
	formatter := FormatterPageFormatterInit(page, opts)

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()
	expectEqualStrings(t, "aXXXXfg", string(output))
}

// Zig: test "Page codepoint_map multiple ranges"
func TestFm2_Page_codepoint_map_multiple_ranges(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("hello world")))

	page := fm2LastPage(t, term)

	// Replace 'a'-'m' with 'A' and 'n'-'z' with 'Z'
	var map_ []FormatterCodepointMap
	// Zig-only: defer map.deinit(alloc);
	map_ = append(map_, FormatterCodepointMap{
		range_:      [2]uint32{'a', 'm'},
		replacement: fm2ReplCodepoint('A'),
	})
	map_ = append(map_, FormatterCodepointMap{
		range_:      [2]uint32{'n', 'z'},
		replacement: fm2ReplCodepoint('Z'),
	})

	opts := FormatterOptionsPlain
	opts.CodepointMap = Some(map_)
	formatter := FormatterPageFormatterInit(page, opts)

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()
	// h e l l o   w o r l d
	// A A A A Z   Z Z Z A A
	expectEqualStrings(t, "AAAAZ ZZZAA", string(output))
}

// Zig: test "Page codepoint_map unicode replacement"
func TestFm2_Page_codepoint_map_unicode_replacement(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("hello ⚡ world")))

	page := fm2LastPage(t, term)

	// Replace lightning bolt with fire emoji
	var map_ []FormatterCodepointMap
	// Zig-only: defer map.deinit(alloc);
	map_ = append(map_, FormatterCodepointMap{
		range_:      [2]uint32{'⚡', '⚡'},
		replacement: fm2ReplString("🔥"),
	})

	opts := FormatterOptionsPlain
	opts.CodepointMap = Some(map_)
	formatter := FormatterPageFormatterInit(page, opts)

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()
	expectEqualStrings(t, "hello 🔥 world", string(output))

	// Verify point map
	expectEqual(t, len(output), len(point_map))
	// "hello ⚡ world"
	// h e l l o   ⚡   w o r l  d
	// 0 1 2 3 4 5 6   8 9 10 11 12
	// Note: ⚡ is a wide character occupying cells 6-7
	for i := 0; i < 6; i++ {
		expectEqual(t,
			PointCoordinate{X: SizeCellCountInt(i), Y: 0},
			point_map[i],
		)
	}
	// 🔥 is 4 UTF-8 bytes, all should map to cell 6 (where ⚡ was)
	fire_start := 6 // "hello " is 6 bytes
	for i := 0; i < 4; i++ {
		expectEqual(t,
			PointCoordinate{X: 6, Y: 0},
			point_map[fire_start+i],
		)
	}
	// " world" follows
	world_start := fire_start + 4
	for i := 0; i < 6; i++ {
		expectEqual(t,
			PointCoordinate{X: SizeCellCountInt(8 + i), Y: 0},
			point_map[world_start+i],
		)
	}
}

// Zig: test "Page codepoint_map with styled formats"
func TestFm2_Page_codepoint_map_with_styled_formats(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 10, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("\x1b[31mred text\x1b[0m")))

	page := fm2LastPage(t, term)

	// Replace 'e' with 'X' in styled text
	var map_ []FormatterCodepointMap
	// Zig-only: defer map.deinit(alloc);
	map_ = append(map_, FormatterCodepointMap{
		range_:      [2]uint32{'e', 'e'},
		replacement: fm2ReplCodepoint('X'),
	})

	opts := FormatterOptionsVt
	opts.CodepointMap = Some(map_)
	formatter := FormatterPageFormatterInit(page, opts)

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()
	// Should preserve styles while replacing text
	// "red text" becomes "rXd tXxt"
	// VT format uses \x1b[38;5;1m for palette color 1
	expectEqualStrings(t, "\x1b[0m\x1b[38;5;1mrXd tXxt\x1b[0m", string(output))
}

// Zig: test "Page codepoint_map empty map"
func TestFm2_Page_codepoint_map_empty_map(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("hello world")))

	page := fm2LastPage(t, term)

	// Empty map should not change anything
	var map_ []FormatterCodepointMap
	// Zig-only: defer map.deinit(alloc);

	opts := FormatterOptionsPlain
	opts.CodepointMap = Some(map_)
	formatter := FormatterPageFormatterInit(page, opts)

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()
	expectEqualStrings(t, "hello world", string(output))
}

// Zig: test "Page VT background color on trailing blank cells"
func TestFm2_Page_VT_background_color_on_trailing_blank_cells(t *testing.T) {
	// This test reproduces a bug where trailing cells with background color
	// but no text are emitted as plain spaces without SGR sequences.
	// This causes TUIs like htop to lose background colors on rehydration.
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 20, 5)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Simulate a TUI row: "CPU:" with text, then trailing cells with red background
	// to end of line (no text after the colored region).
	// \x1b[41m sets red background, then EL fills rest of row with that bg.
	zigTry(t, s.NextSlice([]uint8("CPU:\x1b[41m\x1b[K")))
	// Reset colors and move to next line with different content
	zigTry(t, s.NextSlice([]uint8("\x1b[0m\r\nline2")))

	page := fm2LastPage(t, term)

	formatter := FormatterPageFormatterInit(page, FormatterOptionsVt)
	formatter.opts.Trim = false // Don't trim so we can see the trailing behavior

	zigTry(t, formatter.format(&builder.writer))
	output := string(builder.writer.buffered())

	// The output should preserve the red background SGR for trailing cells on line 1.
	// Bug: the first row outputs "CPU:\r\n" only - losing the background color fill.
	// The red background should appear BEFORE the newline, not after.

	// Find position of CRLF
	crlf_pos := strings.Index(output, "\r\n")
	if crlf_pos < 0 {
		// No CRLF found, fail the test
		t.Fatalf("error.TestUnexpectedResult")
	}

	// Check that red background (48;5;1) appears BEFORE the newline (on line 1)
	line1 := output[0:crlf_pos]
	has_red_bg_line1 := strings.Contains(line1, "\x1b[41m") ||
		strings.Contains(line1, "\x1b[48;5;1m")

	// This should be true but currently fails due to the bug
	expect(t, has_red_bg_line1, "has_red_bg_line1")
}

// Zig: test "Page HTML with hyperlinks"
func TestFm2_Page_HTML_with_hyperlinks(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Start a hyperlink, write some text, end it
	zigTry(t, s.NextSlice([]uint8("\x1b]8;;https://example.com\x1b\\link text\x1b]8;;\x1b\\ normal")))

	page := fm2LastPage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsHtml)

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	expectEqualStrings(t,
		"<div style=\"font-family: monospace; white-space: pre;\">"+
			"<a href=\"https://example.com\">link text</a> normal"+
			"</div>",
		string(output),
	)
}

// Zig: test "Page HTML with multiple hyperlinks"
func TestFm2_Page_HTML_with_multiple_hyperlinks(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Two different hyperlinks
	zigTry(t, s.NextSlice([]uint8("\x1b]8;;https://first.com\x1b\\first\x1b]8;;\x1b\\ ")))
	zigTry(t, s.NextSlice([]uint8("\x1b]8;;https://second.com\x1b\\second\x1b]8;;\x1b\\")))

	page := fm2LastPage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsHtml)

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	expectEqualStrings(t,
		"<div style=\"font-family: monospace; white-space: pre;\">"+
			"<a href=\"https://first.com\">first</a>"+
			" "+
			"<a href=\"https://second.com\">second</a>"+
			"</div>",
		string(output),
	)
}

// Zig: test "Page HTML with hyperlink escaping"
func TestFm2_Page_HTML_with_hyperlink_escaping(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// URL with special characters that need escaping
	zigTry(t, s.NextSlice([]uint8("\x1b]8;;https://example.com?a=1&b=2\x1b\\link\x1b]8;;\x1b\\")))

	page := fm2LastPage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsHtml)

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	expectEqualStrings(t,
		"<div style=\"font-family: monospace; white-space: pre;\">"+
			"<a href=\"https://example.com?a=1&amp;b=2\">link</a>"+
			"</div>",
		string(output),
	)
}

// Zig: test "Page HTML with styled hyperlink"
func TestFm2_Page_HTML_with_styled_hyperlink(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Bold hyperlink
	zigTry(t, s.NextSlice([]uint8("\x1b]8;;https://example.com\x1b\\\x1b[1mbold link\x1b[0m\x1b]8;;\x1b\\")))

	page := fm2LastPage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsHtml)

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	expectEqualStrings(t,
		"<div style=\"font-family: monospace; white-space: pre;\">"+
			"<div style=\"display: inline;font-weight: bold;\">"+
			"<a href=\"https://example.com\">bold link</div></a>"+
			"</div>",
		string(output),
	)
}

// Zig: test "Page HTML hyperlink closes style before anchor"
func TestFm2_Page_HTML_hyperlink_closes_style_before_anchor(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	// Styled hyperlink followed by plain text
	zigTry(t, s.NextSlice([]uint8("\x1b]8;;https://example.com\x1b\\\x1b[1mbold\x1b[0m plain")))

	page := fm2LastPage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsHtml)

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	expectEqualStrings(t,
		"<div style=\"font-family: monospace; white-space: pre;\">"+
			"<div style=\"display: inline;font-weight: bold;\">"+
			"<a href=\"https://example.com\">bold</div> plain</a>"+
			"</div>",
		string(output),
	)
}

// Zig: test "Page HTML hyperlink point map maps closing to previous cell"
func TestFm2_Page_HTML_hyperlink_point_map_maps_closing_to_previous_cell(t *testing.T) {
	testStdCapacity(t)
	builder := oscWriterAllocatingInit()
	defer builder.deinit()

	term := fm2Init(t, 80, 24)
	defer term.Deinit()

	s := term.VtStream()
	defer s.Deinit()

	zigTry(t, s.NextSlice([]uint8("\x1b]8;;https://example.com\x1b\\link\x1b]8;;\x1b\\ normal")))

	page := fm2LastPage(t, term)
	formatter := FormatterPageFormatterInit(page, FormatterOptionsHtml)

	var point_map []PointCoordinate
	// Zig-only: defer point_map.deinit(alloc);
	formatter.point_map = Some(FormatterPageFormatterPointMap{map_: &point_map})

	zigTry(t, formatter.format(&builder.writer))
	output := builder.writer.buffered()

	const expected_output = "<div style=\"font-family: monospace; white-space: pre;\">" +
		"<a href=\"https://example.com\">link</a> normal" +
		"</div>"
	expectEqualStrings(t, expected_output, string(output))
	expectEqual(t, len(expected_output), len(point_map))

	// The </a> closing tag bytes should all map to the last cell of the link
	closing_idx := strings.Index(expected_output, "</a>")
	expect(t, closing_idx >= 0, "std.mem.indexOf(u8, expected_output, \"</a>\").?")
	expected_coord := point_map[closing_idx-1]
	for i := closing_idx; i < closing_idx+len("</a>"); i++ {
		expectEqual(t, expected_coord, point_map[i])
	}
}

// Zig: test "term51: Format.styled: only plain is unstyled"
func TestFm2_Format_styled_only_plain_is_unstyled(t *testing.T) {
	testStdCapacity(t)
	expect(t, !FormatterFormatPlain.Styled(), "!Format.plain.styled()")
	expect(t, FormatterFormatVt.Styled(), "Format.vt.styled()")
	expect(t, FormatterFormatHtml.Styled(), "Format.html.styled()")
	// Options presets carry the matching format.
	expectEqual(t, FormatterFormatPlain, FormatterOptionsPlain.Emit)
	expectEqual(t, FormatterFormatVt, FormatterOptionsVt.Emit)
	expectEqual(t, FormatterFormatHtml, FormatterOptionsHtml.Emit)
}

// Zig: test "term51: PageFormatter.init: defaults and option defaults"
func TestFm2_PageFormatter_init_defaults_and_option_defaults(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 5, 2)
	defer term.Deinit()

	f := FormatterPageFormatterInit(fm2Page(t, term), FormatterOptionsHtml)
	expect(t, f.page == fm2Page(t, term), "f.page == term51ZPage(&t)")
	expectEqual(t, FormatterFormatHtml, f.opts.Emit)
	expect(t, !f.opts.Unwrap, "!f.opts.unwrap")
	expect(t, f.opts.Trim, "f.opts.trim")
	expect(t, !f.opts.Background.Valid, "f.opts.background == null")
	expect(t, !f.opts.Foreground.Valid, "f.opts.foreground == null")
	expect(t, f.opts.Palette == nil, "f.opts.palette == null")
	// The codepoint map defaults to an empty (non-null) list.
	expect(t, f.opts.CodepointMap.Valid, "f.opts.codepoint_map != null")
	expectEqual(t, 0, len(f.opts.CodepointMap.Value))
	expectEqual(t, SizeCellCountInt(0), f.start_x)
	expectEqual(t, SizeCellCountInt(0), f.start_y)
	expect(t, !f.end_x.Valid, "f.end_x == null")
	expect(t, !f.end_y.Valid, "f.end_y == null")
	expect(t, !f.rectangle, "!f.rectangle")
	expect(t, !f.point_map.Valid, "f.point_map == null")
	expect(t, !f.trailing_state.Valid, "f.trailing_state == null")

	// TrailingState defaults.
	empty := FormatterPageFormatterTrailingStateEmpty
	expectEqual(t, uint(0), empty.rows)
	expectEqual(t, uint(0), empty.cells)
	dflt := FormatterPageFormatterTrailingState{}
	expectEqual(t, uint(0), dflt.rows)
	expectEqual(t, uint(0), dflt.cells)
}

// Zig: test "term51: PageListFormatter.init: defaults"
func TestFm2_PageListFormatter_init_defaults(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 5, 2)
	defer term.Deinit()

	pages := &term.Screens.Active.Pages
	f := FormatterPageListFormatterInit(pages, FormatterOptionsVt)
	expect(t, f.list == pages, "f.list == pages")
	expectEqual(t, FormatterFormatVt, f.opts.Emit)
	expect(t, !f.top_left.Valid, "f.top_left == null")
	expect(t, !f.bottom_right.Valid, "f.bottom_right == null")
	expect(t, !f.rectangle, "!f.rectangle")
	expect(t, !f.pin_map.Valid, "f.pin_map == null")
}

// Zig: test "term51: ScreenFormatter.init: defaults"
func TestFm2_ScreenFormatter_init_defaults(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 5, 2)
	defer term.Deinit()

	f := FormatterScreenFormatterInit(term.Screens.Active, FormatterOptionsVt)
	expect(t, f.screen == term.Screens.Active, "f.screen == t.screens.active")
	expectEqual(t, FormatterFormatVt, f.opts.Emit)
	expect(t, f.content.Tag == FormatterScreenFormatterContentSelection, "f.content == .selection")
	expect(t, !f.content.Selection.Valid, "f.content.selection == null")
	expectEqual(t, FormatterScreenFormatterExtraNone, f.extra)
	expect(t, !f.pin_map.Valid, "f.pin_map == null")

	// The presets.
	none := FormatterScreenFormatterExtraNone
	expect(t, !none.Cursor, "!none.cursor")
	expect(t, !none.Style, "!none.style")
	expect(t, !none.Hyperlink, "!none.hyperlink")
	expect(t, !none.Protection, "!none.protection")
	expect(t, !none.KittyKeyboard, "!none.kitty_keyboard")
	expect(t, !none.Charsets, "!none.charsets")
	styles := FormatterScreenFormatterExtraStyles
	expect(t, !styles.Cursor, "!styles.cursor")
	expect(t, styles.Style, "styles.style")
	expect(t, styles.Hyperlink, "styles.hyperlink")
	expect(t, !styles.Protection, "!styles.protection")
	expect(t, !styles.KittyKeyboard, "!styles.kitty_keyboard")
	expect(t, !styles.Charsets, "!styles.charsets")
	all := FormatterScreenFormatterExtraAll
	expect(t, all.Cursor, "all.cursor")
	expect(t, all.Style, "all.style")
	expect(t, all.Hyperlink, "all.hyperlink")
	expect(t, all.Protection, "all.protection")
	expect(t, all.KittyKeyboard, "all.kitty_keyboard")
	expect(t, all.Charsets, "all.charsets")
}

// Zig: test "term51: TerminalFormatter.init: defaults and extra presets"
func TestFm2_TerminalFormatter_init_defaults_and_extra_presets(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 5, 2)
	defer term.Deinit()

	f := FormatterTerminalFormatterInit(term, FormatterOptionsHtml)
	expect(t, f.terminal == term, "f.terminal == &t")
	expectEqual(t, FormatterFormatHtml, f.opts.Emit)
	expect(t, f.content.Tag == FormatterScreenFormatterContentSelection, "f.content == .selection")
	expect(t, !f.content.Selection.Valid, "f.content.selection == null")
	expect(t, !f.pin_map.Valid, "f.pin_map == null")
	// The default extra is `styles`.
	expectEqual(t, FormatterTerminalFormatterExtraStyles, f.extra)

	none := FormatterTerminalFormatterExtraNone
	expect(t, !none.Palette, "!none.palette")
	expect(t, !none.Modes, "!none.modes")
	expect(t, !none.ScrollingRegion, "!none.scrolling_region")
	expect(t, !none.Tabstops, "!none.tabstops")
	expect(t, !none.Pwd, "!none.pwd")
	expect(t, !none.Keyboard, "!none.keyboard")
	expectEqual(t, FormatterScreenFormatterExtraNone, none.Screen)

	styles := FormatterTerminalFormatterExtraStyles
	expect(t, styles.Palette, "styles.palette")
	expect(t, !styles.Modes, "!styles.modes")
	expect(t, !styles.ScrollingRegion, "!styles.scrolling_region")
	expect(t, !styles.Tabstops, "!styles.tabstops")
	expect(t, !styles.Pwd, "!styles.pwd")
	expect(t, !styles.Keyboard, "!styles.keyboard")
	expectEqual(t, FormatterScreenFormatterExtraStyles, styles.Screen)

	all := FormatterTerminalFormatterExtraAll
	expect(t, all.Palette, "all.palette")
	expect(t, all.Modes, "all.modes")
	expect(t, all.ScrollingRegion, "all.scrolling_region")
	expect(t, all.Tabstops, "all.tabstops")
	expect(t, all.Pwd, "all.pwd")
	expect(t, all.Keyboard, "all.keyboard")
	expectEqual(t, FormatterScreenFormatterExtraAll, all.Screen)
}

// Zig: test "term51: ScreenFormatter.Extra.isSet: every field alone, presets"
func TestFm2_ScreenFormatter_Extra_isSet_every_field_alone_presets(t *testing.T) {
	testStdCapacity(t)
	expect(t, !FormatterScreenFormatterExtraNone.isSet(), "!E.isSet(.none)")
	expect(t, FormatterScreenFormatterExtraStyles.isSet(), "E.isSet(.styles)")
	expect(t, FormatterScreenFormatterExtraAll.isSet(), "E.isSet(.all)")

	// Zig: inline for (@typeInfo(E).@"struct".fields). The six fields, by hand.
	fields := []struct {
		name string
		set  func(e *FormatterScreenFormatterExtra, v bool)
	}{
		{"cursor", func(e *FormatterScreenFormatterExtra, v bool) { e.Cursor = v }},
		{"style", func(e *FormatterScreenFormatterExtra, v bool) { e.Style = v }},
		{"hyperlink", func(e *FormatterScreenFormatterExtra, v bool) { e.Hyperlink = v }},
		{"protection", func(e *FormatterScreenFormatterExtra, v bool) { e.Protection = v }},
		{"kitty_keyboard", func(e *FormatterScreenFormatterExtra, v bool) { e.KittyKeyboard = v }},
		{"charsets", func(e *FormatterScreenFormatterExtra, v bool) { e.Charsets = v }},
	}
	for _, field := range fields {
		e := FormatterScreenFormatterExtraNone
		field.set(&e, true)
		expect(t, e.isSet(), "E.isSet(e) ("+field.name+")")

		// All set except this one is still set.
		rest := FormatterScreenFormatterExtraAll
		field.set(&rest, false)
		expect(t, rest.isSet(), "E.isSet(rest) ("+field.name+")")
	}

	// Exactly the six documented fields exist.
	expectEqual(t, 6, reflect.TypeOf(FormatterScreenFormatterExtra{}).NumField())
	expectEqual(t, 6, len(fields))
}

// Zig: test "term51: PageFormatter.writeCodepoint: plain and vt write UTF-8"
func TestFm2_PageFormatter_writeCodepoint_plain_and_vt_write_UTF_8(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 5, 2)
	defer term.Deinit()

	for _, emit := range []FormatterFormat{FormatterFormatPlain, FormatterFormatVt} {
		f := FormatterPageFormatterInit(fm2Page(t, term), fm2Opts(emit))
		fm2ExpectCp(t, f, 'a', "a")
		fm2ExpectCp(t, f, 0, "\x00")
		fm2ExpectCp(t, f, 0x1b, "\x1b")
		fm2ExpectCp(t, f, '\n', "\n")
		fm2ExpectCp(t, f, 0x7f, "\x7f")
		// No HTML escaping.
		fm2ExpectCp(t, f, '<', "<")
		fm2ExpectCp(t, f, '>', ">")
		fm2ExpectCp(t, f, '&', "&")
		fm2ExpectCp(t, f, '"', "\"")
		fm2ExpectCp(t, f, '\'', "'")
		// UTF-8 boundaries: 1, 2, 3 and 4 byte encodings.
		fm2ExpectCp(t, f, 0x80, "\xc2\x80")
		fm2ExpectCp(t, f, 0x7ff, "\xdf\xbf")
		fm2ExpectCp(t, f, 0x800, "\xe0\xa0\x80")
		fm2ExpectCp(t, f, 0xffff, "\xef\xbf\xbf")
		fm2ExpectCp(t, f, 0x10000, "\xf0\x90\x80\x80")
		fm2ExpectCp(t, f, 0x10ffff, "\xf4\x8f\xbf\xbf")
		// Unencodable values become U+FFFD: a surrogate half and a value
		// above U+10FFFF (u21 holds up to 0x1FFFFF).
		fm2ExpectCp(t, f, 0xd800, "\xef\xbf\xbd")
		fm2ExpectCp(t, f, 0xdfff, "\xef\xbf\xbd")
		fm2ExpectCp(t, f, 0x110000, "\xef\xbf\xbd")
		fm2ExpectCp(t, f, 0x1fffff, "\xef\xbf\xbd")
	}
}

// Zig: test "term51: PageFormatter.writeCodepoint: html escapes and numeric entities"
func TestFm2_PageFormatter_writeCodepoint_html_escapes_and_numeric_entities(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 5, 2)
	defer term.Deinit()

	f := FormatterPageFormatterInit(fm2Page(t, term), FormatterOptionsHtml)
	fm2ExpectCp(t, f, '<', "&lt;")
	fm2ExpectCp(t, f, '>', "&gt;")
	fm2ExpectCp(t, f, '&', "&amp;")
	fm2ExpectCp(t, f, '"', "&quot;")
	fm2ExpectCp(t, f, '\'', "&#39;")

	// ASCII (< 0x80) is emitted directly, including controls.
	fm2ExpectCp(t, f, 'a', "a")
	fm2ExpectCp(t, f, ' ', " ")
	fm2ExpectCp(t, f, '\n', "\n")
	fm2ExpectCp(t, f, 0, "\x00")
	fm2ExpectCp(t, f, 0x1b, "\x1b")
	fm2ExpectCp(t, f, 0x7f, "\x7f")

	// Everything at or above 0x80 is a decimal numeric entity, with no
	// validity check.
	fm2ExpectCp(t, f, 0x80, "&#128;")
	fm2ExpectCp(t, f, 0xe9, "&#233;")
	fm2ExpectCp(t, f, 0x4e2d, "&#20013;")
	fm2ExpectCp(t, f, 0xd800, "&#55296;")
	fm2ExpectCp(t, f, 0x10ffff, "&#1114111;")
	fm2ExpectCp(t, f, 0x1fffff, "&#2097151;")
}

// Zig: test "term51: PageFormatter.writeCodepointWithReplacement: no map and empty map pass through"
func TestFm2_PageFormatter_writeCodepointWithReplacement_no_map_and_empty_map_pass_through(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 5, 2)
	defer term.Deinit()

	// Null map.
	{
		opts := fm2Opts(FormatterFormatPlain)
		opts.CodepointMap = Optional[[]FormatterCodepointMap]{}
		f := FormatterPageFormatterInit(fm2Page(t, term), opts)
		fm2ExpectReplaced(t, f, 'a', "a")
		fm2ExpectReplaced(t, f, 0x4e2d, "\xe4\xb8\xad")
	}

	// Default (empty) map.
	{
		f := FormatterPageFormatterInit(fm2Page(t, term), FormatterOptionsPlain)
		fm2ExpectReplaced(t, f, 'a', "a")
		fm2ExpectReplaced(t, f, 0x4e2d, "\xe4\xb8\xad")
	}
}

// Zig: test "term51: PageFormatter.writeCodepointWithReplacement: range bounds are inclusive"
func TestFm2_PageFormatter_writeCodepointWithReplacement_range_bounds_are_inclusive(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 5, 2)
	defer term.Deinit()

	m := fm2MapInit([]FormatterCodepointMap{
		fm2Entry(0x41, 0x43, fm2ReplCodepoint('X')),
		fm2Entry('z', 'z', fm2ReplCodepoint('Y')),
	})
	// Zig-only: defer m.deinit(alloc);

	opts := fm2Opts(FormatterFormatPlain)
	opts.CodepointMap = Some(m)
	f := FormatterPageFormatterInit(fm2Page(t, term), opts)
	fm2ExpectReplaced(t, f, 0x40, "@")
	fm2ExpectReplaced(t, f, 0x41, "X")
	fm2ExpectReplaced(t, f, 0x42, "X")
	fm2ExpectReplaced(t, f, 0x43, "X")
	fm2ExpectReplaced(t, f, 0x44, "D")
	// A single-point range.
	fm2ExpectReplaced(t, f, 'y', "y")
	fm2ExpectReplaced(t, f, 'z', "Y")
	fm2ExpectReplaced(t, f, '{', "{")
}

// Zig: test "term51: PageFormatter.writeCodepointWithReplacement: last matching entry wins"
func TestFm2_PageFormatter_writeCodepointWithReplacement_last_matching_entry_wins(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 5, 2)
	defer term.Deinit()

	// Specific entry after the broad one.
	{
		m := fm2MapInit([]FormatterCodepointMap{
			fm2Entry('a', 'z', fm2ReplCodepoint('X')),
			fm2Entry('c', 'c', fm2ReplCodepoint('Y')),
		})
		// Zig-only: defer m.deinit(alloc);
		opts := fm2Opts(FormatterFormatPlain)
		opts.CodepointMap = Some(m)
		f := FormatterPageFormatterInit(fm2Page(t, term), opts)
		fm2ExpectReplaced(t, f, 'c', "Y")
		fm2ExpectReplaced(t, f, 'd', "X")
	}

	// Broad entry after the specific one hides it.
	{
		m := fm2MapInit([]FormatterCodepointMap{
			fm2Entry('c', 'c', fm2ReplCodepoint('Y')),
			fm2Entry('a', 'z', fm2ReplCodepoint('X')),
		})
		// Zig-only: defer m.deinit(alloc);
		opts := fm2Opts(FormatterFormatPlain)
		opts.CodepointMap = Some(m)
		f := FormatterPageFormatterInit(fm2Page(t, term), opts)
		fm2ExpectReplaced(t, f, 'c', "X")
	}

	// Replacements are applied once, not recursively.
	{
		m := fm2MapInit([]FormatterCodepointMap{
			fm2Entry('a', 'a', fm2ReplCodepoint('b')),
			fm2Entry('b', 'b', fm2ReplCodepoint('c')),
		})
		// Zig-only: defer m.deinit(alloc);
		opts := fm2Opts(FormatterFormatPlain)
		opts.CodepointMap = Some(m)
		f := FormatterPageFormatterInit(fm2Page(t, term), opts)
		fm2ExpectReplaced(t, f, 'a', "b")
		fm2ExpectReplaced(t, f, 'b', "c")
	}
}

// Zig: test "term51: PageFormatter.writeCodepointWithReplacement: string replacements"
func TestFm2_PageFormatter_writeCodepointWithReplacement_string_replacements(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 5, 2)
	defer term.Deinit()

	m := fm2MapInit([]FormatterCodepointMap{
		fm2Entry('a', 'a', fm2ReplString("xyz")),
		fm2Entry('b', 'b', fm2ReplString("")),
		fm2Entry('c', 'c', fm2ReplString("\xc3\xa9\xe4\xb8\xad\xf0\x9f\x98\x80")),
		fm2Entry('d', 'd', fm2ReplString("<&>")),
	})
	// Zig-only: defer m.deinit(alloc);

	{
		opts := fm2Opts(FormatterFormatPlain)
		opts.CodepointMap = Some(m)
		f := FormatterPageFormatterInit(fm2Page(t, term), opts)
		fm2ExpectReplaced(t, f, 'a', "xyz")
		// The empty string replaces with nothing at all.
		fm2ExpectReplaced(t, f, 'b', "")
		fm2ExpectReplaced(t, f, 'c', "\xc3\xa9\xe4\xb8\xad\xf0\x9f\x98\x80")
		fm2ExpectReplaced(t, f, 'd', "<&>")
		fm2ExpectReplaced(t, f, 'e', "e")
	}

	// For HTML each replacement codepoint is escaped on its own.
	{
		opts := fm2Opts(FormatterFormatHtml)
		opts.CodepointMap = Some(m)
		f := FormatterPageFormatterInit(fm2Page(t, term), opts)
		fm2ExpectReplaced(t, f, 'a', "xyz")
		fm2ExpectReplaced(t, f, 'b', "")
		fm2ExpectReplaced(t, f, 'c', "&#233;&#20013;&#128512;")
		fm2ExpectReplaced(t, f, 'd', "&lt;&amp;&gt;")
	}
}

// Go-only: panic test: PageFormatter.writeCodepointWithReplacement: a string replacement that is not valid UTF-8 (Utf8View.init(s) catch unreachable)
func TestFm2_panic_writeCodepointWithReplacement_string_is_not_valid_UTF_8(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 5, 2)
	defer term.Deinit()

	m := fm2MapInit([]FormatterCodepointMap{
		fm2Entry('a', 'a', fm2ReplString("\xff")),
	})
	opts := fm2Opts(FormatterFormatPlain)
	opts.CodepointMap = Some(m)
	f := FormatterPageFormatterInit(fm2Page(t, term), opts)
	b := oscWriterAllocatingInit()
	defer b.deinit()

	fm2ExpectPanic(t, func() {
		_ = f.writeCodepointWithReplacement(&b.writer, 'a')
	})
}

// Zig: test "term51: PageFormatter.writeCodepointWithReplacement: codepoint replacements per format"
func TestFm2_PageFormatter_writeCodepointWithReplacement_codepoint_replacements_per_format(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 5, 2)
	defer term.Deinit()

	m := fm2MapInit([]FormatterCodepointMap{
		fm2Entry('a', 'a', fm2ReplCodepoint('<')),
		fm2Entry('b', 'b', fm2ReplCodepoint(0x4e2d)),
		fm2Entry(0xe9, 0xe9, fm2ReplCodepoint('e')),
		fm2Entry(0, 0x1fffff, fm2ReplCodepoint('?')),
	})
	// Zig-only: defer m.deinit(alloc);

	// The catch-all last entry hides all of the earlier ones.
	{
		opts := fm2Opts(FormatterFormatVt)
		opts.CodepointMap = Some(m)
		f := FormatterPageFormatterInit(fm2Page(t, term), opts)
		fm2ExpectReplaced(t, f, 'a', "?")
		fm2ExpectReplaced(t, f, 0, "?")
		fm2ExpectReplaced(t, f, 0x1fffff, "?")
	}

	// Without the catch-all.
	m2 := fm2MapInit([]FormatterCodepointMap{
		fm2Entry('a', 'a', fm2ReplCodepoint('<')),
		fm2Entry('b', 'b', fm2ReplCodepoint(0x4e2d)),
		fm2Entry(0xe9, 0xe9, fm2ReplCodepoint('e')),
	})
	// Zig-only: defer m2.deinit(alloc);
	{
		opts := fm2Opts(FormatterFormatPlain)
		opts.CodepointMap = Some(m2)
		f := FormatterPageFormatterInit(fm2Page(t, term), opts)
		fm2ExpectReplaced(t, f, 'a', "<")
		fm2ExpectReplaced(t, f, 'b', "\xe4\xb8\xad")
		fm2ExpectReplaced(t, f, 0xe9, "e")
	}
	{
		opts := fm2Opts(FormatterFormatHtml)
		opts.CodepointMap = Some(m2)
		f := FormatterPageFormatterInit(fm2Page(t, term), opts)
		fm2ExpectReplaced(t, f, 'a', "&lt;")
		fm2ExpectReplaced(t, f, 'b', "&#20013;")
		fm2ExpectReplaced(t, f, 0xe9, "e")
		fm2ExpectReplaced(t, f, 'c', "c")
	}
}

// Zig: test "term51: PageFormatter.writeCell: blank, text and grapheme cells"
func TestFm2_PageFormatter_writeCell_blank_text_and_grapheme_cells(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 10, 2)
	defer term.Deinit()
	// x=0 'a', x=1 empty (skipped by CUF), x=2 'e' + combining acute, x=3 '<'.
	fm2Write(t, term, "a\x1b[1Ce\xcc\x81<")

	page := fm2Page(t, term)
	expectEqual(t, uint32('a'), fm2Cell(t, term, 0, 0).Codepoint())
	expectEqual(t, PageCellContentTagCodepoint, fm2Cell(t, term, 0, 0).content_tag)
	// x=1 was skipped by CUF so it is empty; x=2 is the grapheme cell.
	expect(t, !fm2Cell(t, term, 1, 0).HasText(), "!term51ZCell(&t, 1, 0).hasText()")
	expectEqual(t, PageCellContentTagCodepointGrapheme, fm2Cell(t, term, 2, 0).content_tag)

	plain := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	html := FormatterPageFormatterInit(page, FormatterOptionsHtml)

	b := oscWriterAllocatingInit()
	defer b.deinit()

	// Blank cell: a plain space, whatever the format.
	zigTry(t, plain.writeCell(PageCellContentTagCodepoint, &b.writer, fm2Cell(t, term, 1, 0)))
	zigTry(t, html.writeCell(PageCellContentTagCodepoint, &b.writer, fm2Cell(t, term, 1, 0)))
	expectEqualStrings(t, "  ", string(b.writer.buffered()))
	b.writer.end = 0

	// Text cell.
	zigTry(t, plain.writeCell(PageCellContentTagCodepoint, &b.writer, fm2Cell(t, term, 0, 0)))
	expectEqualStrings(t, "a", string(b.writer.buffered()))
	b.writer.end = 0
	zigTry(t, html.writeCell(PageCellContentTagCodepoint, &b.writer, fm2Cell(t, term, 3, 0)))
	expectEqualStrings(t, "&lt;", string(b.writer.buffered()))
	b.writer.end = 0

	// Grapheme cell with the grapheme tag: base then each extra codepoint.
	zigTry(t, plain.writeCell(PageCellContentTagCodepointGrapheme, &b.writer, fm2Cell(t, term, 2, 0)))
	expectEqualStrings(t, "e\xcc\x81", string(b.writer.buffered()))
	b.writer.end = 0
	zigTry(t, html.writeCell(PageCellContentTagCodepointGrapheme, &b.writer, fm2Cell(t, term, 2, 0)))
	expectEqualStrings(t, "e&#769;", string(b.writer.buffered()))
	b.writer.end = 0

	// The comptime tag decides: with `.codepoint` only the base is written.
	zigTry(t, plain.writeCell(PageCellContentTagCodepoint, &b.writer, fm2Cell(t, term, 2, 0)))
	expectEqualStrings(t, "e", string(b.writer.buffered()))
	b.writer.end = 0
}

// Go-only: panic test: PageFormatter.writeCell: the grapheme tag on a cell that has no grapheme data (page.lookupGrapheme(cell).? on null)
func TestFm2_panic_writeCell_grapheme_tag_without_grapheme_data(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 10, 2)
	defer term.Deinit()
	fm2Write(t, term, "a")

	// The cell is a plain text cell: the page has no grapheme for it.
	cell := fm2Cell(t, term, 0, 0)
	f := FormatterPageFormatterInit(fm2Page(t, term), FormatterOptionsPlain)
	b := oscWriterAllocatingInit()
	defer b.deinit()

	fm2ExpectPanic(t, func() {
		_ = f.writeCell(PageCellContentTagCodepointGrapheme, &b.writer, cell)
	})
}

// Zig: test "term51: PageFormatter.writeCell: replacement applies to graphemes and not to blanks"
func TestFm2_PageFormatter_writeCell_replacement_applies_to_graphemes_and_not_to_blanks(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 10, 2)
	defer term.Deinit()
	fm2Write(t, term, "e\xcc\x81\x1b[2Gx")
	page := fm2Page(t, term)

	m := fm2MapInit([]FormatterCodepointMap{
		fm2Entry(0, 0, fm2ReplCodepoint('N')),
		fm2Entry('e', 'e', fm2ReplString("EE")),
		fm2Entry(0x301, 0x301, fm2ReplCodepoint('!')),
	})
	// Zig-only: defer m.deinit(alloc);
	opts := fm2Opts(FormatterFormatPlain)
	opts.CodepointMap = Some(m)
	f := FormatterPageFormatterInit(page, opts)

	b := oscWriterAllocatingInit()
	defer b.deinit()

	// Both the base codepoint and the grapheme codepoints are mapped.
	zigTry(t, f.writeCell(PageCellContentTagCodepointGrapheme, &b.writer, fm2Cell(t, term, 0, 0)))
	expectEqualStrings(t, "EE!", string(b.writer.buffered()))
	b.writer.end = 0

	// An empty cell is a space even though codepoint 0 is mapped.
	zigTry(t, f.writeCell(PageCellContentTagCodepoint, &b.writer, fm2Cell(t, term, 5, 0)))
	expectEqualStrings(t, " ", string(b.writer.buffered()))
	b.writer.end = 0

	// Background-only cells (no text) are spaces as well.
	bg_cell := PageCellInit(0)
	bg_cell.content_tag = PageCellContentTagBgColorPalette
	bg_cell.content = PageCellContent{color_palette: 'e'}
	zigTry(t, f.writeCell(PageCellContentTagBgColorPalette, &b.writer, &bg_cell))
	expectEqualStrings(t, " ", string(b.writer.buffered()))
	b.writer.end = 0
	bg_cell.content_tag = PageCellContentTagBgColorRgb
	bg_cell.content = PageCellContent{color_rgb: PageCellRGB{R: 'e', G: 'e', B: 'e'}}
	zigTry(t, f.writeCell(PageCellContentTagBgColorRgb, &b.writer, &bg_cell))
	expectEqualStrings(t, " ", string(b.writer.buffered()))
}

// Zig: test "term51: PageFormatter.cellStyle: unstyled, styled and background cells"
func TestFm2_PageFormatter_cellStyle_unstyled_styled_and_background_cells(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 10, 2)
	defer term.Deinit()
	// x0 plain, x1 bold red + underline curly, x2 grapheme with italic,
	// then bg palette fill and bg rgb fill via EL on the next row.
	fm2Write(t, term, "a\x1b[1;31;4:3mb\x1b[0;3me\xcc\x81\x1b[0m")
	fm2Write(t, term, "\r\n\x1b[45m\x1b[K")
	fm2Write(t, term, "\x1b[2;3H\x1b[48;2;10;20;30m\x1b[K")

	f := FormatterPageFormatterInit(fm2Page(t, term), FormatterOptionsPlain)

	s0 := f.cellStyle(fm2Cell(t, term, 0, 0))
	expect(t, s0.Default(), "s0.default()")

	s1 := f.cellStyle(fm2Cell(t, term, 1, 0))
	expect(t, !s1.Default(), "!s1.default()")
	expect(t, s1.Flags.Bold, "s1.flags.bold")
	expectEqual(t, SGRAttributeUnderlineCurly, s1.Flags.Underline)
	expect(t, !s1.Flags.Italic, "!s1.flags.italic")
	expect(t, s1.FgColor.Eql(fm2Palette(1)), "s1.fg_color.eql(.{ .palette = 1 })")
	expect(t, s1.BgColor.Eql(StyleColor{}), "s1.bg_color.eql(.none)")

	s2 := f.cellStyle(fm2Cell(t, term, 2, 0))
	expectEqual(t, PageCellContentTagCodepointGrapheme, fm2Cell(t, term, 2, 0).content_tag)
	expect(t, s2.Flags.Italic, "s2.flags.italic")
	expect(t, !s2.Flags.Bold, "!s2.flags.bold")
	expect(t, s2.FgColor.Eql(StyleColor{}), "s2.fg_color.eql(.none)")

	// Background palette cell: only the bg color, nothing else.
	expectEqual(t, PageCellContentTagBgColorPalette, fm2Cell(t, term, 0, 1).content_tag)
	s3 := f.cellStyle(fm2Cell(t, term, 0, 1))
	expect(t, s3.Eql(Style{BgColor: fm2Palette(5)}), "s3.eql(.{ .bg_color = .{ .palette = 5 } })")

	// Background rgb cell.
	expectEqual(t, PageCellContentTagBgColorRgb, fm2Cell(t, term, 2, 1).content_tag)
	s4 := f.cellStyle(fm2Cell(t, term, 2, 1))
	expect(t, s4.Eql(Style{BgColor: fm2Rgb(10, 20, 30)}), "s4.eql(.{ .bg_color = .{ .rgb = .{ .r = 10, .g = 20, .b = 30 } } })")
	expect(t, s4.FgColor.Eql(StyleColor{}), "s4.fg_color.eql(.none)")
	expect(t, s4.underline_color.Eql(StyleColor{}), "s4.underline_color.eql(.none)")

	// A styled cell that is empty (e.g. erased with a background set) and
	// a styled blank are all still read via the style id.
	expect(t, f.cellStyle(fm2Cell(t, term, 9, 0)).Default(), "f.cellStyle(term51ZCell(&t, 9, 0)).default()")
}

// Zig: test "term51: PageFormatter.formatStyleOpen: vt flags"
func TestFm2_PageFormatter_formatStyleOpen_vt_flags(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 5, 2)
	defer term.Deinit()
	f := FormatterPageFormatterInit(fm2Page(t, term), FormatterOptionsVt)

	fm2ExpectStyleOpen(t, f, Style{}, "\x1b[0m")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Bold: true}}, "\x1b[0m\x1b[1m")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Faint: true}}, "\x1b[0m\x1b[2m")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Italic: true}}, "\x1b[0m\x1b[3m")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Blink: true}}, "\x1b[0m\x1b[5m")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Inverse: true}}, "\x1b[0m\x1b[7m")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Invisible: true}}, "\x1b[0m\x1b[8m")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Strikethrough: true}}, "\x1b[0m\x1b[9m")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Overline: true}}, "\x1b[0m\x1b[53m")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Underline: SGRAttributeUnderlineSingle}}, "\x1b[0m\x1b[4m")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Underline: SGRAttributeUnderlineDouble}}, "\x1b[0m\x1b[4:2m")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Underline: SGRAttributeUnderlineCurly}}, "\x1b[0m\x1b[4:3m")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Underline: SGRAttributeUnderlineDotted}}, "\x1b[0m\x1b[4:4m")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Underline: SGRAttributeUnderlineDashed}}, "\x1b[0m\x1b[4:5m")

	// All flags: fixed order regardless of how they were set.
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{
		Bold:          true,
		Italic:        true,
		Faint:         true,
		Blink:         true,
		Inverse:       true,
		Invisible:     true,
		Strikethrough: true,
		Overline:      true,
		Underline:     SGRAttributeUnderlineCurly,
	}}, "\x1b[0m\x1b[1m\x1b[2m\x1b[3m\x1b[5m\x1b[7m\x1b[8m\x1b[9m\x1b[53m\x1b[4:3m")
}

// Go-only: panic test: PageFormatter.formatStyleOpen: plain format (.plain => unreachable)
func TestFm2_panic_formatStyleOpen_plain_format(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 5, 2)
	defer term.Deinit()
	f := FormatterPageFormatterInit(fm2Page(t, term), FormatterOptionsPlain)
	b := oscWriterAllocatingInit()
	defer b.deinit()
	style := Style{Flags: StyleFlags{Bold: true}}

	fm2ExpectPanic(t, func() {
		_ = f.formatStyleOpen(&b.writer, &style)
	})
}

// Zig: test "term51: PageFormatter.formatStyleOpen: vt colors and palette option"
func TestFm2_PageFormatter_formatStyleOpen_vt_colors_and_palette_option(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 5, 2)
	defer term.Deinit()
	f := FormatterPageFormatterInit(fm2Page(t, term), FormatterOptionsVt)

	// Palette indices stay indices without a palette option.
	fm2ExpectStyleOpen(t, f, Style{FgColor: fm2Palette(0)}, "\x1b[0m\x1b[38;5;0m")
	fm2ExpectStyleOpen(t, f, Style{FgColor: fm2Palette(255)}, "\x1b[0m\x1b[38;5;255m")
	fm2ExpectStyleOpen(t, f, Style{BgColor: fm2Palette(7)}, "\x1b[0m\x1b[48;5;7m")
	fm2ExpectStyleOpen(t, f, Style{underline_color: fm2Palette(9)}, "\x1b[0m\x1b[58;5;9m")
	// RGB.
	fm2ExpectStyleOpen(t, f, Style{FgColor: fm2Rgb(1, 2, 3)}, "\x1b[0m\x1b[38;2;1;2;3m")
	fm2ExpectStyleOpen(t, f, Style{BgColor: fm2Rgb(255, 0, 128)}, "\x1b[0m\x1b[48;2;255;0;128m")
	fm2ExpectStyleOpen(t, f, Style{underline_color: fm2Rgb(4, 5, 6)}, "\x1b[0m\x1b[58;2;4;5;6m")
	// Order: flags, fg, bg, underline color.
	fm2ExpectStyleOpen(t, f, Style{
		FgColor:         fm2Palette(1),
		BgColor:         fm2Rgb(9, 8, 7),
		underline_color: fm2Palette(2),
		Flags:           StyleFlags{Bold: true},
	}, "\x1b[0m\x1b[1m\x1b[38;5;1m\x1b[48;2;9;8;7m\x1b[58;5;2m")

	// With a palette option, palette indices are resolved to RGB and the
	// other kinds are unchanged.
	palette := ColorDefault
	palette[3] = ColorRGB{R: 30, G: 31, B: 32}
	palette[200] = ColorRGB{R: 200, G: 201, B: 202}
	fp_opts := FormatterOptionsVt
	fp_opts.Palette = &palette
	fp := FormatterPageFormatterInit(fm2Page(t, term), fp_opts)
	fm2ExpectStyleOpen(t, fp, Style{FgColor: fm2Palette(3)}, "\x1b[0m\x1b[38;2;30;31;32m")
	fm2ExpectStyleOpen(t, fp, Style{BgColor: fm2Palette(200)}, "\x1b[0m\x1b[48;2;200;201;202m")
	fm2ExpectStyleOpen(t, fp, Style{underline_color: fm2Palette(3)}, "\x1b[0m\x1b[58;2;30;31;32m")
	fm2ExpectStyleOpen(t, fp, Style{FgColor: fm2Rgb(1, 2, 3)}, "\x1b[0m\x1b[38;2;1;2;3m")
	fm2ExpectStyleOpen(t, fp, Style{}, "\x1b[0m")
}

// Zig: test "term51: PageFormatter.formatStyleOpen: html flags"
func TestFm2_PageFormatter_formatStyleOpen_html_flags(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 5, 2)
	defer term.Deinit()
	f := FormatterPageFormatterInit(fm2Page(t, term), FormatterOptionsHtml)

	const open = "<div style=\"display: inline;"
	fm2ExpectStyleOpen(t, f, Style{}, open+"\">")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Bold: true}}, open+"font-weight: bold;\">")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Italic: true}}, open+"font-style: italic;\">")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Faint: true}}, open+"opacity: 0.5;\">")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Invisible: true}}, open+"visibility: hidden;\">")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Inverse: true}}, open+"filter: invert(100%);\">")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Blink: true}}, open+"text-decoration-line: blink;\">")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Strikethrough: true}}, open+"text-decoration-line: line-through;\">")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Overline: true}}, open+"text-decoration-line: overline;\">")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Underline: SGRAttributeUnderlineSingle}}, open+"text-decoration-line: underline;text-decoration-style: solid;\">")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Underline: SGRAttributeUnderlineDouble}}, open+"text-decoration-line: underline;text-decoration-style: double;\">")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Underline: SGRAttributeUnderlineCurly}}, open+"text-decoration-line: underline;text-decoration-style: wavy;\">")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Underline: SGRAttributeUnderlineDotted}}, open+"text-decoration-line: underline;text-decoration-style: dotted;\">")
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{Underline: SGRAttributeUnderlineDashed}}, open+"text-decoration-line: underline;text-decoration-style: dashed;\">")

	// Everything at once: line list in fixed order, then style, then the
	// simple properties.
	fm2ExpectStyleOpen(t, f, Style{Flags: StyleFlags{
		Bold:          true,
		Italic:        true,
		Faint:         true,
		Blink:         true,
		Inverse:       true,
		Invisible:     true,
		Strikethrough: true,
		Overline:      true,
		Underline:     SGRAttributeUnderlineDouble,
	}}, open+"text-decoration-line: underline line-through overline blink;"+
		"text-decoration-style: double;font-weight: bold;font-style: italic;"+
		"opacity: 0.5;visibility: hidden;filter: invert(100%);\">")
}

// Zig: test "term51: PageFormatter.formatStyleOpen: html colors and palette option"
func TestFm2_PageFormatter_formatStyleOpen_html_colors_and_palette_option(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 5, 2)
	defer term.Deinit()
	f := FormatterPageFormatterInit(fm2Page(t, term), FormatterOptionsHtml)

	const open = "<div style=\"display: inline;"
	fm2ExpectStyleOpen(t, f, Style{FgColor: fm2Palette(0)}, open+"color: var(--vt-palette-0);\">")
	fm2ExpectStyleOpen(t, f, Style{BgColor: fm2Palette(255)}, open+"background-color: var(--vt-palette-255);\">")
	fm2ExpectStyleOpen(t, f, Style{underline_color: fm2Palette(12)}, open+"text-decoration-color: var(--vt-palette-12);\">")
	fm2ExpectStyleOpen(t, f, Style{FgColor: fm2Rgb(1, 2, 3)}, open+"color: rgb(1, 2, 3);\">")
	fm2ExpectStyleOpen(t, f, Style{BgColor: fm2Rgb(255, 255, 0)}, open+"background-color: rgb(255, 255, 0);\">")
	fm2ExpectStyleOpen(t, f, Style{underline_color: fm2Rgb(0, 0, 9)}, open+"text-decoration-color: rgb(0, 0, 9);\">")
	// Colors come first (fg, bg, underline) then the flags.
	fm2ExpectStyleOpen(t, f, Style{
		FgColor:         fm2Palette(3),
		BgColor:         fm2Rgb(1, 2, 3),
		underline_color: fm2Rgb(4, 5, 6),
		Flags:           StyleFlags{Bold: true, Underline: SGRAttributeUnderlineSingle},
	}, open+"color: var(--vt-palette-3);background-color: rgb(1, 2, 3);"+
		"text-decoration-color: rgb(4, 5, 6);text-decoration-line: underline;"+
		"text-decoration-style: solid;font-weight: bold;\">")

	// Palette option resolves indices to rgb().
	palette := ColorDefault
	palette[3] = ColorRGB{R: 30, G: 31, B: 32}
	fp_opts := FormatterOptionsHtml
	fp_opts.Palette = &palette
	fp := FormatterPageFormatterInit(fm2Page(t, term), fp_opts)
	fm2ExpectStyleOpen(t, fp, Style{
		FgColor:         fm2Palette(3),
		BgColor:         fm2Palette(3),
		underline_color: fm2Palette(3),
	}, open+"color: rgb(30, 31, 32);background-color: rgb(30, 31, 32);"+
		"text-decoration-color: rgb(30, 31, 32);\">")
}

// Zig: test "term51: PageFormatter.formatStyleClose: per format and point map"
func TestFm2_PageFormatter_formatStyleClose_per_format_and_point_map(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 5, 2)
	defer term.Deinit()

	b := oscWriterAllocatingInit()
	defer b.deinit()

	// Plain writes nothing.
	{
		f := FormatterPageFormatterInit(fm2Page(t, term), FormatterOptionsPlain)
		zigTry(t, f.formatStyleClose(&b.writer))
		expectEqualStrings(t, "", string(b.writer.buffered()))
	}
	// VT.
	{
		f := FormatterPageFormatterInit(fm2Page(t, term), FormatterOptionsVt)
		zigTry(t, f.formatStyleClose(&b.writer))
		expectEqualStrings(t, "\x1b[0m", string(b.writer.buffered()))
		b.writer.end = 0
	}
	// HTML.
	{
		f := FormatterPageFormatterInit(fm2Page(t, term), FormatterOptionsHtml)
		zigTry(t, f.formatStyleClose(&b.writer))
		expectEqualStrings(t, "</div>", string(b.writer.buffered()))
		b.writer.end = 0
	}

	// Point map: the closing bytes repeat the previous entry.
	var pm []PointCoordinate
	// Zig-only: defer pm.deinit(alloc);
	pm = append(pm, PointCoordinate{X: 1, Y: 0})
	pm = append(pm, PointCoordinate{X: 3, Y: 2})
	{
		f := FormatterPageFormatterInit(fm2Page(t, term), FormatterOptionsVt)
		f.point_map = Some(FormatterPageFormatterPointMap{map_: &pm})
		zigTry(t, f.formatStyleClose(&b.writer))
		b.writer.end = 0
		expectEqual(t, 2+4, len(pm))
		for _, c := range pm[2:] {
			expectEqual(t, PointCoordinate{X: 3, Y: 2}, c)
		}
	}
	{
		f := FormatterPageFormatterInit(fm2Page(t, term), FormatterOptionsHtml)
		f.point_map = Some(FormatterPageFormatterPointMap{map_: &pm})
		zigTry(t, f.formatStyleClose(&b.writer))
		b.writer.end = 0
		expectEqual(t, 6+6, len(pm))
		for _, c := range pm[6:] {
			expectEqual(t, PointCoordinate{X: 3, Y: 2}, c)
		}
	}
	// Plain with a point map writes and maps nothing.
	{
		f := FormatterPageFormatterInit(fm2Page(t, term), FormatterOptionsPlain)
		f.point_map = Some(FormatterPageFormatterPointMap{map_: &pm})
		zigTry(t, f.formatStyleClose(&b.writer))
		expectEqualStrings(t, "", string(b.writer.buffered()))
		expectEqual(t, 12, len(pm))
	}
}

// Go-only: panic test: PageFormatter.formatStyleClose: a point map with no entries (assert(m.map.items.len > 0))
func TestFm2_panic_formatStyleClose_empty_point_map(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 5, 2)
	defer term.Deinit()
	b := oscWriterAllocatingInit()
	defer b.deinit()

	var pm []PointCoordinate
	f := FormatterPageFormatterInit(fm2Page(t, term), FormatterOptionsVt)
	f.point_map = Some(FormatterPageFormatterPointMap{map_: &pm})

	fm2ExpectPanic(t, func() {
		_ = f.formatStyleClose(&b.writer)
	})
}

// Zig: test "term51: PageFormatter.formatHyperlinkOpen: html anchors and escaping"
func TestFm2_PageFormatter_formatHyperlinkOpen_html_anchors_and_escaping(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 5, 2)
	defer term.Deinit()
	f := FormatterPageFormatterInit(fm2Page(t, term), FormatterOptionsHtml)

	b := oscWriterAllocatingInit()
	defer b.deinit()

	zigTry(t, f.formatHyperlinkOpen(&b.writer, []uint8("https://example.com/a")))
	expectEqualStrings(t, "<a href=\"https://example.com/a\">", string(b.writer.buffered()))
	b.writer.end = 0

	zigTry(t, f.formatHyperlinkOpen(&b.writer, []uint8("")))
	expectEqualStrings(t, "<a href=\"\">", string(b.writer.buffered()))
	b.writer.end = 0

	// Each of the five special characters is escaped.
	zigTry(t, f.formatHyperlinkOpen(&b.writer, []uint8("a<b>c&d\"e'f")))
	expectEqualStrings(t,
		"<a href=\"a&lt;b&gt;c&amp;d&quot;e&#39;f\">",
		string(b.writer.buffered()),
	)
	b.writer.end = 0

	// The URI is escaped per byte, not per codepoint: each byte of a
	// UTF-8 sequence becomes its own numeric entity.
	zigTry(t, f.formatHyperlinkOpen(&b.writer, []uint8("x\xc3\xa9")))
	expectEqualStrings(t, "<a href=\"x&#195;&#169;\">", string(b.writer.buffered()))
	b.writer.end = 0

	// Control characters below 0x80 pass through.
	zigTry(t, f.formatHyperlinkOpen(&b.writer, []uint8("a\tb\x01")))
	expectEqualStrings(t, "<a href=\"a\tb\x01\">", string(b.writer.buffered()))
}

// Go-only: panic test: PageFormatter.formatHyperlinkOpen: plain format (.plain, .vt => unreachable)
func TestFm2_panic_formatHyperlinkOpen_plain_format(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 5, 2)
	defer term.Deinit()
	f := FormatterPageFormatterInit(fm2Page(t, term), FormatterOptionsPlain)
	b := oscWriterAllocatingInit()
	defer b.deinit()

	fm2ExpectPanic(t, func() {
		_ = f.formatHyperlinkOpen(&b.writer, []uint8("https://example.com"))
	})
}

// Go-only: panic test: PageFormatter.formatHyperlinkOpen: vt format (.plain, .vt => unreachable)
func TestFm2_panic_formatHyperlinkOpen_vt_format(t *testing.T) {
	testStdCapacity(t)
	term := fm2Init(t, 5, 2)
	defer term.Deinit()
	f := FormatterPageFormatterInit(fm2Page(t, term), FormatterOptionsVt)
	b := oscWriterAllocatingInit()
	defer b.deinit()

	fm2ExpectPanic(t, func() {
		_ = f.formatHyperlinkOpen(&b.writer, []uint8("https://example.com"))
	})
}
