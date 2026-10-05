// Tests translated one for one from src/terminal/formatter.zig (Ghostty
// v1.3.1 plus the term51 additional tests), lines 7126 to the end of the file
// (tests 123 to 180): the PageFormatter.formatHyperlinkClose and
// formatWithState tests, the PageListFormatter, ScreenFormatter and
// TerminalFormatter tests, the point map tests and the replay tests. Each Go
// test carries the exact Zig title in a "// Zig: test" comment. Helpers are
// prefixed fm3 (the helpers of the Zig file's earlier part, term51Z*, that
// these tests use are redefined here under that prefix).
//
// Translation notes:
//   - `var t = try Terminal.init(alloc, .{ .cols = c, .rows = r })` is
//     `term := fm3Init(t, c, r)` (starting from terminalOptionsDefault); the
//     Terminal that Zig's tests call `t` is called `term` because the testing
//     parameter is `t`. `.default_modes = .{ ... }` modifies the default
//     modes.
//   - `term51ZWrite(&t, data)` is fm3Write(t, term, data); `term51ZPage` is
//     fm3Page; `term51ZCell` is fm3Cell; `term51ZPageOut`, `term51ZOut`,
//     `term51ZRun`, `term51ZExpectPage`, `term51ZTwoPages`, `term51ZPin`,
//     `term51ZSelOut`, `term51ZCellStyle`, `term51ZRgb`, `term51ZHex`,
//     `term51ZModes` and `term51ZHtmlOpen` are fm3PageOut, fm3Out, fm3Run,
//     fm3ExpectPage, fm3TwoPages, fm3Pin, fm3SelOut, fm3CellStyle, fm3Rgb,
//     fm3Hex, fm3Modes and fm3HtmlOpen. `term51ZMapInit` builds the codepoint
//     map: a plain slice.
//   - The Options of Zig's `.{ .emit = ... }` literals start from a preset
//     (fm3Opts) so that `trim` is true and `codepoint_map` is present.
//   - `.html` / `.plain` / `.vt` passed as Options are the presets
//     FormatterOptionsHtml / FormatterOptionsPlain / FormatterOptionsVt.
//   - `std.ArrayList(Coordinate)` / `std.ArrayList(Pin)` are `[]PointCoordinate`
//     / `[]PageListPin` variables referenced from the pin map / point map.
//   - Expected output that Zig builds with writer helpers (`splatByteAll`,
//     `**`) is built with strings.Repeat.
//   - Zig's `alloc` arguments are dropped; `defer alloc.free(...)`,
//     `defer r.deinit(alloc)` and `defer pm.deinit(alloc)` leak checks are
//     dropped with `// Zig-only:` comments.
//   - Every test calls testStdCapacity first (Zig's unit tests run with
//     grapheme_bytes 512).
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt. The Go-only panic tests are at the end.

package terminal

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

// fm3ExpectPanic runs f, which must panic. A panic with the wireframe's
// "not implemented" value does not count: it only means the function under
// test does not exist yet. The setup is done outside f.
func fm3ExpectPanic(t *testing.T, f func()) {
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

// fm3Init is `try Terminal.init(alloc, .{ .cols = cols, .rows = rows })`; the
// terminal is returned by pointer (the methods take a pointer receiver).
func fm3Init(tt *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) *Terminal {
	tt.Helper()
	t, err := TerminalInit(terminalOptionsDefault(cols, rows))
	zigTry(tt, err)
	return &t
}

// fm3Write is term51ZWrite.
func fm3Write(tt *testing.T, term *Terminal, data string) {
	tt.Helper()
	s := term.VtStream()
	defer s.Deinit()
	zigTry(tt, s.NextSlice([]uint8(data)))
}

// fm3Page is term51ZPage: the first page of the active screen.
func fm3Page(term *Terminal) *Page {
	return &term.screens.active.pages.pages.first.data
}

// fm3Cell is term51ZCell.
func fm3Cell(term *Terminal, x SizeCellCountInt, y SizeCellCountInt) *PageCell {
	return fm3Page(term).GetRowAndCell(uint(x), uint(y)).cell
}

// fm3Opts is `.{ .emit = emit }`: a preset with another format.
func fm3Opts(emit FormatterFormat) FormatterOptions {
	opts := FormatterOptionsPlain
	opts.emit = emit
	return opts
}

// fm3SomeCc is a non-null size.CellCountInt.
func fm3SomeCc(v uint) Optional[SizeCellCountInt] {
	return Some(SizeCellCountInt(v))
}

// fm3Trailing is `.{ .rows = rows, .cells = cells }` as the optional
// trailing_state field.
func fm3Trailing(rows uint, cells uint) Optional[FormatterPageFormatterTrailingState] {
	return Some(FormatterPageFormatterTrailingState{rows: rows, cells: cells})
}

// fm3Coord is `Coordinate{ .x = x, .y = y }`.
func fm3Coord(x SizeCellCountInt, y uint32) PointCoordinate {
	return PointCoordinate{x: x, y: y}
}

// fm3Formatter is "any formatter that has a `format` method".
type fm3Formatter interface {
	Format(writer *oscWriter) error
}

// fm3Out is term51ZOut: format with any formatter and return the output.
func fm3Out(tt *testing.T, f fm3Formatter) string {
	tt.Helper()
	builder := oscWriterAllocatingInit()
	defer builder.deinit()
	zigTry(tt, f.Format(&builder.writer))
	out, err := builder.toOwnedSlice()
	zigTry(tt, err)
	return string(out)
}

// fm3PageOut is term51ZPageOut: format the first page of the terminal with a
// PageFormatter.
func fm3PageOut(tt *testing.T, term *Terminal, opts FormatterOptions) string {
	tt.Helper()
	f := FormatterPageFormatterInit(fm3Page(term), opts)
	return fm3Out(tt, f)
}

// fm3Res is term51ZRes.
type fm3Res struct {
	out   string
	state FormatterPageFormatterTrailingState
}

// fm3Run is term51ZRun.
func fm3Run(tt *testing.T, f FormatterPageFormatter) fm3Res {
	tt.Helper()
	builder := oscWriterAllocatingInit()
	defer builder.deinit()
	state, err := f.FormatWithState(&builder.writer)
	zigTry(tt, err)
	out, err := builder.toOwnedSlice()
	zigTry(tt, err)
	return fm3Res{out: string(out), state: state}
}

// fm3ExpectPage is term51ZExpectPage.
func fm3ExpectPage(tt *testing.T, term *Terminal, opts FormatterOptions, expected string) {
	tt.Helper()
	out := fm3PageOut(tt, term, opts)
	expectEqualStrings(tt, expected, out)
}

// fm3HtmlOpen is term51ZHtmlOpen.
const fm3HtmlOpen = "<div style=\"font-family: monospace; white-space: pre;\">"

// fm3Rgb is term51ZRgb: "{d};{d};{d}".
func fm3Rgb(c ColorRGB) string {
	return fmt.Sprintf("%d;%d;%d", c.r, c.g, c.b)
}

// fm3Hex is term51ZHex: "{x:0>2}/{x:0>2}/{x:0>2}".
func fm3Hex(c ColorRGB) string {
	return fmt.Sprintf("%02x/%02x/%02x", c.r, c.g, c.b)
}

// fm3ExpectWriteFailed is `try std.testing.expectError(error.WriteFailed, err)`.
func fm3ExpectWriteFailed(tt *testing.T, err error) {
	tt.Helper()
	if err != ErrWriteFailed {
		tt.Fatalf("expected error WriteFailed, got %v", err)
	}
}

// fm3IndexOfNone is `std.mem.indexOfNone(u8, s, set)` (null is -1).
func fm3IndexOfNone(s string, set string) int {
	for i := 0; i < len(s); i++ {
		if !strings.ContainsRune(set, rune(s[i])) {
			return i
		}
	}
	return -1
}

// fm3TwoPages fills the first page so `first` lands on its last row and
// `second` on the first row of a second page. Returns the first page's row
// count.
func fm3TwoPages(tt *testing.T, term *Terminal, first string, second string) uint {
	tt.Helper()
	pages := &term.screens.active.pages
	first_page_rows := uint(pages.pages.first.data.capacity.rows)
	s := term.VtStream()
	defer s.Deinit()
	for i := uint(0); i < first_page_rows-1; i++ {
		zigTry(tt, s.NextSlice([]uint8("\r\n")))
	}
	zigTry(tt, s.NextSlice([]uint8(first)))
	expect(tt, pages.pages.first == pages.pages.last, "pages.pages.first == pages.pages.last")
	zigTry(tt, s.NextSlice([]uint8("\r\n")))
	expect(tt, pages.pages.first != pages.pages.last, "pages.pages.first != pages.pages.last")
	zigTry(tt, s.NextSlice([]uint8(second)))
	return first_page_rows
}

// fm3Pin is term51ZPin: `t.screens.active.pages.pin(.{ .active = .{ .x = x,
// .y = y } }).?`.
func fm3Pin(tt *testing.T, term *Terminal, x SizeCellCountInt, y uint32) PageListPin {
	tt.Helper()
	return zigUnwrap(tt, term.screens.active.pages.Pin(Point{
		Tag:    PointTagActive,
		Active: PointCoordinate{x: x, y: y},
	}))
}

// fm3ContentSelection is `.{ .selection = sel }`.
func fm3ContentSelection(sel Selection) FormatterScreenFormatterContent {
	return FormatterScreenFormatterContent{
		Tag:       FormatterScreenFormatterContentSelection,
		Selection: Some(sel),
	}
}

// fm3ContentNone is `.none`.
var fm3ContentNone = FormatterScreenFormatterContent{Tag: FormatterScreenFormatterContentNone}

// fm3SelOut is term51ZSelOut.
func fm3SelOut(tt *testing.T, term *Terminal, opts FormatterOptions, sel Selection) string {
	tt.Helper()
	f := FormatterScreenFormatterInit(term.screens.active, opts)
	f.content = fm3ContentSelection(sel)
	return fm3Out(tt, f)
}

// fm3SetExtraField is `@field(f.extra, name) = true` on a ScreenFormatter's
// extra.
func fm3SetExtraField(tt *testing.T, extra *FormatterScreenFormatterExtra, name string) {
	tt.Helper()
	switch name {
	case "cursor":
		extra.cursor = true
	case "style":
		extra.style = true
	case "hyperlink":
		extra.hyperlink = true
	case "protection":
		extra.protection = true
	case "kitty_keyboard":
		extra.kitty_keyboard = true
	case "charsets":
		extra.charsets = true
	default:
		tt.Fatalf("no such field: %s", name)
	}
}

// fm3Modes is term51ZModes: every mode with its CSI code, in declaration
// order.
var fm3Modes = []struct {
	mode ModesMode
	code string
}{
	{mode: ModesModeDisableKeyboard, code: "2"},
	{mode: ModesModeInsert, code: "4"},
	{mode: ModesModeSendReceiveMode, code: "12"},
	{mode: ModesModeLinefeed, code: "20"},
	{mode: ModesModeCursorKeys, code: "?1"},
	{mode: ModesMode132Column, code: "?3"},
	{mode: ModesModeSlowScroll, code: "?4"},
	{mode: ModesModeReverseColors, code: "?5"},
	{mode: ModesModeOrigin, code: "?6"},
	{mode: ModesModeWraparound, code: "?7"},
	{mode: ModesModeAutorepeat, code: "?8"},
	{mode: ModesModeMouseEventX10, code: "?9"},
	{mode: ModesModeCursorBlinking, code: "?12"},
	{mode: ModesModeCursorVisible, code: "?25"},
	{mode: ModesModeEnableMode3, code: "?40"},
	{mode: ModesModeReverseWrap, code: "?45"},
	{mode: ModesModeAltScreenLegacy, code: "?47"},
	{mode: ModesModeKeypadKeys, code: "?66"},
	{mode: ModesModeEnableLeftAndRightMargin, code: "?69"},
	{mode: ModesModeMouseEventNormal, code: "?1000"},
	{mode: ModesModeMouseEventButton, code: "?1002"},
	{mode: ModesModeMouseEventAny, code: "?1003"},
	{mode: ModesModeFocusEvent, code: "?1004"},
	{mode: ModesModeMouseFormatUtf8, code: "?1005"},
	{mode: ModesModeMouseFormatSgr, code: "?1006"},
	{mode: ModesModeMouseAlternateScroll, code: "?1007"},
	{mode: ModesModeMouseFormatUrxvt, code: "?1015"},
	{mode: ModesModeMouseFormatSgrPixels, code: "?1016"},
	{mode: ModesModeIgnoreKeypadWithNumlock, code: "?1035"},
	{mode: ModesModeAltEscPrefix, code: "?1036"},
	{mode: ModesModeAltSendsEscape, code: "?1039"},
	{mode: ModesModeReverseWrapExtended, code: "?1045"},
	{mode: ModesModeAltScreen, code: "?1047"},
	{mode: ModesModeSaveCursor, code: "?1048"},
	{mode: ModesModeAltScreenSaveCursorClearEnter, code: "?1049"},
	{mode: ModesModeBracketedPaste, code: "?2004"},
	{mode: ModesModeSynchronizedOutput, code: "?2026"},
	{mode: ModesModeGraphemeCluster, code: "?2027"},
	{mode: ModesModeReportColorScheme, code: "?2031"},
	{mode: ModesModeInBandSizeReports, code: "?2048"},
}

// fm3CellStyle is term51ZCellStyle.
func fm3CellStyle(term *Terminal, x SizeCellCountInt, y SizeCellCountInt) Style {
	p := fm3Page(term)
	c := fm3Cell(term, x, y)
	switch c.content_tag {
	case PageCellContentTagCodepoint, PageCellContentTagCodepointGrapheme:
		if c.style_id == 0 {
			return Style{}
		}
		return *p.styles.Get(c.style_id)
	case PageCellContentTagBgColorPalette:
		return Style{bg_color: StyleColor{Tag: StyleColorPalette, Palette: c.content.color_palette}}
	case PageCellContentTagBgColorRgb:
		return Style{bg_color: StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{
			r: c.content.color_rgb.r,
			g: c.content.color_rgb.g,
			b: c.content.color_rgb.b,
		}}}
	}
	panic("unreachable")
}

// Zig: test "term51: PageFormatter.formatHyperlinkClose: html only, with point map"
func TestFm3_term51_PageFormatter_formatHyperlinkClose_html_only_with_point_map(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 5, 2)
	defer term.Deinit()

	b := oscWriterAllocatingInit()
	defer b.deinit()

	// Plain and VT write nothing.
	for _, emit := range []FormatterFormat{FormatterFormatPlain, FormatterFormatVt} {
		f := FormatterPageFormatterInit(fm3Page(term), fm3Opts(emit))
		zigTry(t, f.formatHyperlinkClose(&b.writer))
		expectEqualStrings(t, "", string(b.writer.buffered()))
	}

	f := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsHtml)
	zigTry(t, f.formatHyperlinkClose(&b.writer))
	expectEqualStrings(t, "</a>", string(b.writer.buffered()))

	// Point map: plain/vt add nothing; html repeats the last entry 4 times.
	var pm []PointCoordinate
	// Zig-only: defer pm.deinit(alloc);
	pm = append(pm, fm3Coord(2, 1))
	for _, emit := range []FormatterFormat{FormatterFormatPlain, FormatterFormatVt} {
		fm := FormatterPageFormatterInit(fm3Page(term), fm3Opts(emit))
		fm.point_map = Some(FormatterPageFormatterPointMap{map_: &pm})
		zigTry(t, fm.formatHyperlinkClose(&b.writer))
		expectEqual(t, 1, len(pm))
	}
	fm := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsHtml)
	fm.point_map = Some(FormatterPageFormatterPointMap{map_: &pm})
	zigTry(t, fm.formatHyperlinkClose(&b.writer))
	expectEqual(t, 5, len(pm))
	for _, c := range pm[1:] {
		expectEqual(t, fm3Coord(2, 1), c)
	}
}

// Zig: test "term51: PageFormatter.formatWithState: empty page per format and header colors"
func TestFm3_term51_PageFormatter_formatWithState_empty_page_per_format_and_header_colors(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 3)
	defer term.Deinit()

	bg := ColorRGB{r: 1, g: 2, b: 3}
	fg := ColorRGB{r: 0xaa, g: 0xbb, b: 0xcc}

	// Plain: nothing, never any header. State counts all rows blank.
	{
		opts := fm3Opts(FormatterFormatPlain)
		opts.background = Some(bg)
		opts.foreground = Some(fg)
		f := FormatterPageFormatterInit(fm3Page(term), opts)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "", r.out)
		expectEqual(t, uint(3), r.state.rows)
		expectEqual(t, uint(0), r.state.cells)
	}

	// VT without colors emits nothing; with colors the OSC 10/11 header is
	// emitted even though the page is empty (foreground first).
	fm3ExpectPage(t, term, FormatterOptionsVt, "")
	{
		opts := fm3Opts(FormatterFormatVt)
		opts.foreground = Some(fg)
		fm3ExpectPage(t, term, opts, "\x1b]10;rgb:aa/bb/cc\x1b\\")
	}
	{
		opts := fm3Opts(FormatterFormatVt)
		opts.background = Some(bg)
		fm3ExpectPage(t, term, opts, "\x1b]11;rgb:01/02/03\x1b\\")
	}
	{
		opts := fm3Opts(FormatterFormatVt)
		opts.foreground = Some(fg)
		opts.background = Some(bg)
		fm3ExpectPage(t, term, opts, "\x1b]10;rgb:aa/bb/cc\x1b\\\x1b]11;rgb:01/02/03\x1b\\")
	}

	// HTML always has the wrapper div; background comes first, no space
	// separates the properties.
	fm3ExpectPage(t, term, FormatterOptionsHtml, fm3HtmlOpen+"</div>")
	{
		opts := fm3Opts(FormatterFormatHtml)
		opts.background = Some(bg)
		fm3ExpectPage(t, term, opts,
			"<div style=\"font-family: monospace; white-space: pre;background-color: #010203;\"></div>")
	}
	{
		opts := fm3Opts(FormatterFormatHtml)
		opts.foreground = Some(fg)
		fm3ExpectPage(t, term, opts,
			"<div style=\"font-family: monospace; white-space: pre;color: #aabbcc;\"></div>")
	}
	{
		opts := fm3Opts(FormatterFormatHtml)
		opts.background = Some(bg)
		opts.foreground = Some(fg)
		fm3ExpectPage(t, term, opts,
			"<div style=\"font-family: monospace; white-space: pre;"+
				"background-color: #010203;color: #aabbcc;\"></div>")
	}
}

// Zig: test "term51: PageFormatter.formatWithState: out of range bounds return an empty state"
func TestFm3_term51_PageFormatter_formatWithState_out_of_range_bounds_return_an_empty_state(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 3)
	defer term.Deinit()
	fm3Write(t, term, "abc")

	prior := fm3Trailing(4, 2)

	// Each of these early-returns before any output (not even the HTML
	// wrapper) and, since they all have a non-zero start, ignore the
	// prior trailing state.
	for _, emit := range []FormatterFormat{FormatterFormatPlain, FormatterFormatHtml, FormatterFormatVt} {
		base := FormatterPageFormatterInit(fm3Page(term), fm3Opts(emit))

		f := base
		f.trailing_state = prior
		f.start_x = 10 // == cols
		{
			r := fm3Run(t, f)
			// Zig-only: defer r.deinit(alloc);
			expectEqualStrings(t, "", r.out)
			expectEqual(t, uint(0), r.state.rows)
			expectEqual(t, uint(0), r.state.cells)
		}

		f = base
		f.trailing_state = prior
		f.start_y = 3 // == rows
		{
			r := fm3Run(t, f)
			// Zig-only: defer r.deinit(alloc);
			expectEqualStrings(t, "", r.out)
			expectEqual(t, uint(0), r.state.rows)
			expectEqual(t, uint(0), r.state.cells)
		}

		f = base
		f.trailing_state = prior
		f.start_y = 2
		f.end_y = fm3SomeCc(1) // start_y > end_y
		{
			r := fm3Run(t, f)
			// Zig-only: defer r.deinit(alloc);
			expectEqualStrings(t, "", r.out)
			expectEqual(t, uint(0), r.state.rows)
			expectEqual(t, uint(0), r.state.cells)
		}

		f = base
		f.trailing_state = prior
		f.start_y = 1
		f.end_y = fm3SomeCc(1)
		f.start_x = 5
		f.end_x = fm3SomeCc(4) // single row with start_x > end_x
		{
			r := fm3Run(t, f)
			// Zig-only: defer r.deinit(alloc);
			expectEqualStrings(t, "", r.out)
			expectEqual(t, uint(0), r.state.rows)
			expectEqual(t, uint(0), r.state.cells)
		}
	}
}

// Zig: test "term51: PageFormatter.formatWithState: bounds at the limits are accepted and clamped"
func TestFm3_term51_PageFormatter_formatWithState_bounds_at_the_limits_are_accepted_and_clamped(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 3)
	defer term.Deinit()
	fm3Write(t, term, "abcdefghij\r\nklm\r\nxyz")

	// start_x == cols-1 is the last valid value.
	{
		f := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsPlain)
		f.start_x = 9
		f.end_y = fm3SomeCc(0)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "j", r.out)
		// Only row 0 was in range.
		expectEqual(t, uint(1), r.state.rows)
		expectEqual(t, uint(0), r.state.cells)
	}

	// start_y == rows-1 is the last valid value.
	{
		f := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsPlain)
		f.start_y = 2
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "xyz", r.out)
		expectEqual(t, uint(1), r.state.rows)
		expectEqual(t, uint(7), r.state.cells)
	}

	// end_x and end_y are clamped to the page.
	{
		f := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsPlain)
		f.end_x = Some(SizeCellCountInt(math.MaxUint16))
		f.end_y = Some(SizeCellCountInt(math.MaxUint16))
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "abcdefghij\nklm\nxyz", r.out)
	}

	// end_x == cols-1 and end_y == rows-1 are equivalent to null.
	{
		f := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsPlain)
		f.end_x = fm3SomeCc(9)
		f.end_y = fm3SomeCc(2)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "abcdefghij\nklm\nxyz", r.out)
	}

	// A single cell: start == end.
	{
		f := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsPlain)
		f.start_x = 1
		f.end_x = fm3SomeCc(1)
		f.start_y = 1
		f.end_y = fm3SomeCc(1)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "l", r.out)
		// Row 1 counted, row 2 is outside the range.
		expectEqual(t, uint(1), r.state.rows)
		expectEqual(t, uint(0), r.state.cells)
	}

	// The same x range applies only to the first/last row when not a
	// rectangle: start_x on the first row, end_x on the last row.
	{
		f := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsPlain)
		f.start_x = 8
		f.end_x = fm3SomeCc(1)
		f.end_y = fm3SomeCc(2)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "ij\nklm\nxy", r.out)
	}

	// A row whose text lies outside the x range is a blank row.
	{
		f := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsPlain)
		f.start_x = 4
		f.end_x = fm3SomeCc(0)
		f.start_y = 1
		f.end_y = fm3SomeCc(2)
		// Row 1 from x=4 is empty; row 2 up to x=0 is "x".
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "\nx", r.out)
	}
}

// Zig: test "term51: PageFormatter.formatWithState: trailing state rows and cells"
func TestFm3_term51_PageFormatter_formatWithState_trailing_state_rows_and_cells(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 3)
	defer term.Deinit()

	// Blank page: the prior rows are carried and the page rows added.
	{
		f := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsPlain)
		f.trailing_state = fm3Trailing(2, 5)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "", r.out)
		expectEqual(t, uint(5), r.state.rows)
		expectEqual(t, uint(5), r.state.cells)
	}

	fm3Write(t, term, "x")

	// Prior blank rows are flushed as newlines once text appears. A prior
	// non-wrapped line resets the blank cell count.
	{
		f := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsPlain)
		f.trailing_state = fm3Trailing(2, 7)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "\n\nx", r.out)
		expectEqual(t, uint(3), r.state.rows)
		expectEqual(t, uint(9), r.state.cells)
	}
	// For VT and HTML the newline is \r\n and \n respectively, inside the
	// HTML wrapper.
	{
		f := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsVt)
		f.trailing_state = fm3Trailing(2, 0)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "\r\n\r\nx", r.out)
	}
	{
		f := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsHtml)
		f.trailing_state = fm3Trailing(2, 0)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, fm3HtmlOpen+"\n\nx</div>", r.out)
	}

	// A zero start with a non-null trailing state of zero is the same as
	// none.
	{
		f := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsPlain)
		f.trailing_state = Some(FormatterPageFormatterTrailingStateEmpty)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "x", r.out)
	}

	// A non-zero start_y or start_x ignores the trailing state.
	{
		f := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsPlain)
		f.trailing_state = fm3Trailing(2, 0)
		f.start_x = 1
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "", r.out)
		expectEqual(t, uint(3), r.state.rows)
	}
}

// Zig: test "term51: PageFormatter.formatWithState: trailing cells carry across a wrap continuation"
func TestFm3_term51_PageFormatter_formatWithState_trailing_cells_carry_across_a_wrap_continuation(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 5, 2)
	defer term.Deinit()
	// Row 0 "fghij" (wrapped), row 1 "k" (continuation).
	fm3Write(t, term, "fghijk")
	page := fm3Page(term)
	expect(t, page.GetRow(0).wrap, "page.getRow(0).wrap")
	expect(t, page.GetRow(1).wrap_continuation, "page.getRow(1).wrap_continuation")
	expect(t, !page.GetRow(0).wrap_continuation, "!page.getRow(0).wrap_continuation")

	// Pretend row 0 continues a wrapped row of a previous page.
	page.GetRow(0).wrap_continuation = true

	// Unwrapped: the continuation is ignored and prior cells dropped.
	{
		f := FormatterPageFormatterInit(page, FormatterOptionsPlain)
		f.trailing_state = fm3Trailing(0, 3)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "fghij\nk", r.out)
	}

	// Unwrapped with a continuation: the 3 blank cells of the prior page
	// are emitted as spaces before the first text.
	{
		opts := fm3Opts(FormatterFormatPlain)
		opts.unwrap = true
		f := FormatterPageFormatterInit(page, opts)
		f.trailing_state = fm3Trailing(0, 3)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "   fghijk", r.out)
		expectEqual(t, uint(1), r.state.rows)
		expectEqual(t, uint(4), r.state.cells)
	}

	// The same without prior cells.
	{
		opts := fm3Opts(FormatterFormatPlain)
		opts.unwrap = true
		f := FormatterPageFormatterInit(page, opts)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "fghijk", r.out)
	}

	// A row that is not a continuation drops the carried cells.
	page.GetRow(0).wrap_continuation = false
	{
		opts := fm3Opts(FormatterFormatPlain)
		opts.unwrap = true
		f := FormatterPageFormatterInit(page, opts)
		f.trailing_state = fm3Trailing(0, 3)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "fghijk", r.out)
	}
}

// Zig: test "term51: PageFormatter.formatWithState: soft wrap with spacer head at the end bound"
func TestFm3_term51_PageFormatter_formatWithState_soft_wrap_with_spacer_head_at_the_end_bound(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 5, 3)
	defer term.Deinit()
	// Row 0: a b c d [spacer_head], row 1: wide, tail, e.
	fm3Write(t, term, "abcd\xe4\xb8\xade")
	page := fm3Page(term)
	expectEqual(t, PageCellWideSpacerHead, fm3Cell(term, 4, 0).wide)
	expect(t, page.GetRow(0).wrap, "page.getRow(0).wrap")
	expectEqual(t, PageCellWideWide, fm3Cell(term, 0, 1).wide)
	expectEqual(t, PageCellWideSpacerTail, fm3Cell(term, 1, 1).wide)

	// Whole page, unwrap: no newline, spacer cells skipped.
	{
		opts := fm3Opts(FormatterFormatPlain)
		opts.unwrap = true
		f := FormatterPageFormatterInit(page, opts)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "abcd\xe4\xb8\xade", r.out)
		expectEqual(t, uint(2), r.state.rows)
		expectEqual(t, uint(2), r.state.cells)
	}
	// Whole page, no unwrap: a newline between the rows.
	{
		f := FormatterPageFormatterInit(page, FormatterOptionsPlain)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "abcd\n\xe4\xb8\xade", r.out)
	}

	// End on the spacer head with unwrap: the end moves to the first cell
	// of the next row, so the wide character is included.
	{
		opts := fm3Opts(FormatterFormatPlain)
		opts.unwrap = true
		f := FormatterPageFormatterInit(page, opts)
		f.end_y = fm3SomeCc(0)
		f.end_x = fm3SomeCc(4)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "abcd\xe4\xb8\xad", r.out)
	}
	// Same bound, end_x before the spacer head: no move.
	{
		opts := fm3Opts(FormatterFormatPlain)
		opts.unwrap = true
		f := FormatterPageFormatterInit(page, opts)
		f.end_y = fm3SomeCc(0)
		f.end_x = fm3SomeCc(3)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "abcd", r.out)
	}
	// Without unwrap there is no move.
	{
		f := FormatterPageFormatterInit(page, FormatterOptionsPlain)
		f.end_y = fm3SomeCc(0)
		f.end_x = fm3SomeCc(4)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "abcd", r.out)
	}
	// A rectangle never moves.
	{
		opts := fm3Opts(FormatterFormatPlain)
		opts.unwrap = true
		f := FormatterPageFormatterInit(page, opts)
		f.rectangle = true
		f.end_y = fm3SomeCc(0)
		f.end_x = fm3SomeCc(4)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "abcd", r.out)
	}

	// On the last row of the page the end cannot move: put a spacer head
	// at (4, 2) manually.
	fm3Write(t, term, "\x1b[3;1Hwxyz")
	fm3Cell(term, 4, 2).wide = PageCellWideSpacerHead
	{
		opts := fm3Opts(FormatterFormatPlain)
		opts.unwrap = true
		f := FormatterPageFormatterInit(page, opts)
		f.start_y = 2
		f.end_y = fm3SomeCc(2)
		f.end_x = fm3SomeCc(4)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "wxyz", r.out)
	}
}

// Zig: test "term51: PageFormatter.formatWithState: start on a spacer head skips the row"
func TestFm3_term51_PageFormatter_formatWithState_start_on_a_spacer_head_skips_the_row(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 5, 3)
	defer term.Deinit()
	fm3Write(t, term, "abcd\xe4\xb8\xade")
	page := fm3Page(term)

	// Starting at the spacer head of row 0 skips row 0 entirely, without
	// counting it as a blank row (so there is no leading newline).
	for _, unwrap := range []bool{false, true} {
		opts := fm3Opts(FormatterFormatPlain)
		opts.unwrap = unwrap
		f := FormatterPageFormatterInit(page, opts)
		f.start_x = 4
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "\xe4\xb8\xade", r.out)
		// Row 1 +1, row 2 +1.
		expectEqual(t, uint(2), r.state.rows)
	}

	// A rectangle skips the spacer head row in the same way: the
	// rectangle covers only x=4, which on row 1 is empty, so nothing
	// else is output either.
	{
		f := FormatterPageFormatterInit(page, FormatterOptionsPlain)
		f.rectangle = true
		f.start_x = 4
		f.end_x = fm3SomeCc(4)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "", r.out)
		// Row 0 skipped, rows 1 and 2 are blank.
		expectEqual(t, uint(2), r.state.rows)
	}
}

// Zig: test "term51: PageFormatter.formatWithState: start and end inside wide characters"
func TestFm3_term51_PageFormatter_formatWithState_start_and_end_inside_wide_characters(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 2)
	defer term.Deinit()
	// a [wide][tail] b [wide][tail] c
	fm3Write(t, term, "a\xe4\xb8\xadb\xe6\x96\x87c")
	page := fm3Page(term)
	expectEqual(t, PageCellWideWide, fm3Cell(term, 1, 0).wide)
	expectEqual(t, PageCellWideSpacerTail, fm3Cell(term, 2, 0).wide)
	expectEqual(t, PageCellWideWide, fm3Cell(term, 4, 0).wide)

	const zhong = "\xe4\xb8\xad"
	const wen = "\xe6\x96\x87"

	// start_x on the head, on the tail (backs up to the head), and after.
	{
		f := FormatterPageFormatterInit(page, FormatterOptionsPlain)
		f.start_x = 1
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, zhong+"b"+wen+"c", r.out)
	}
	{
		f := FormatterPageFormatterInit(page, FormatterOptionsPlain)
		f.start_x = 2
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, zhong+"b"+wen+"c", r.out)
	}
	{
		f := FormatterPageFormatterInit(page, FormatterOptionsPlain)
		f.start_x = 3
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "b"+wen+"c", r.out)
	}

	// end_x on the head, on the tail, and before: the head is included in
	// the first two, excluded in the last.
	{
		f := FormatterPageFormatterInit(page, FormatterOptionsPlain)
		f.end_x = fm3SomeCc(1)
		f.end_y = fm3SomeCc(0)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "a"+zhong, r.out)
	}
	{
		f := FormatterPageFormatterInit(page, FormatterOptionsPlain)
		f.end_x = fm3SomeCc(2)
		f.end_y = fm3SomeCc(0)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "a"+zhong, r.out)
	}
	{
		f := FormatterPageFormatterInit(page, FormatterOptionsPlain)
		f.end_x = fm3SomeCc(0)
		f.end_y = fm3SomeCc(0)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "a", r.out)
	}

	// Single-row selection that starts on a tail and ends on a head.
	{
		f := FormatterPageFormatterInit(page, FormatterOptionsPlain)
		f.start_x = 2
		f.end_x = fm3SomeCc(4)
		f.end_y = fm3SomeCc(0)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, zhong+"b"+wen, r.out)
	}

	// A selection that is only a tail: start on the tail pulls in the
	// head even though end_x == start_x.
	{
		f := FormatterPageFormatterInit(page, FormatterOptionsPlain)
		f.start_x = 2
		f.end_x = fm3SomeCc(2)
		f.end_y = fm3SomeCc(0)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, zhong, r.out)
	}

	// HTML and VT: numeric entity and raw UTF-8.
	{
		f := FormatterPageFormatterInit(page, FormatterOptionsHtml)
		f.start_x = 2
		f.end_x = fm3SomeCc(4)
		f.end_y = fm3SomeCc(0)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, fm3HtmlOpen+"&#20013;b&#25991;</div>", r.out)
	}

	// Point map: the three bytes of a wide character map to the head cell,
	// even when the selection started on the tail.
	{
		var pm []PointCoordinate
		// Zig-only: defer pm.deinit(alloc);
		f := FormatterPageFormatterInit(page, FormatterOptionsPlain)
		f.start_x = 2
		f.end_x = fm3SomeCc(4)
		f.end_y = fm3SomeCc(0)
		f.point_map = Some(FormatterPageFormatterPointMap{map_: &pm})
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqual(t, len(r.out), len(pm))
		expectEqualSlices(t, []PointCoordinate{
			{x: 1, y: 0}, {x: 1, y: 0}, {x: 1, y: 0},
			{x: 3, y: 0}, {x: 4, y: 0}, {x: 4, y: 0},
			{x: 4, y: 0},
		}, pm)
	}
}

// Zig: test "term51: PageFormatter.formatWithState: rectangle with wide characters per row"
func TestFm3_term51_PageFormatter_formatWithState_rectangle_with_wide_characters_per_row(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 3)
	defer term.Deinit()
	// Row 0: a [wide][tail] b ; row 1: x y [wide][tail] z
	fm3Write(t, term, "a\xe4\xb8\xadb\r\nxy\xe6\x96\x87z")
	page := fm3Page(term)

	f := FormatterPageFormatterInit(page, FormatterOptionsPlain)
	f.rectangle = true
	f.start_x = 2
	f.end_x = fm3SomeCc(3)
	f.end_y = fm3SomeCc(1)
	r := fm3Run(t, f)
	// Zig-only: defer r.deinit(alloc);
	// Row 0: start on tail backs up to x=1 so cells 1..3 = wide, tail, b.
	// Row 1: start on the wide head at x=2 so cells 2..3 = wide, tail.
	expectEqualStrings(t, "\xe4\xb8\xadb\n\xe6\x96\x87", r.out)
}

// Zig: test "term51: PageFormatter.formatWithState: grapheme clusters per format"
func TestFm3_term51_PageFormatter_formatWithState_grapheme_clusters_per_format(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 2)
	defer term.Deinit()
	// e + combining acute, then a ZWJ family pair (grapheme clustering on).
	fm3Write(t, term, "\x1b[?2027he\xcc\x81x\xf0\x9f\x91\xa8\xe2\x80\x8d\xf0\x9f\x91\xa9y")
	expectEqual(t, PageCellContentTagCodepointGrapheme, fm3Cell(term, 0, 0).content_tag)
	expectEqual(t, PageCellContentTagCodepointGrapheme, fm3Cell(term, 2, 0).content_tag)
	expectEqual(t, PageCellWideWide, fm3Cell(term, 2, 0).wide)

	fm3ExpectPage(t, term, FormatterOptionsPlain,
		"e\xcc\x81x\xf0\x9f\x91\xa8\xe2\x80\x8d\xf0\x9f\x91\xa9y")
	fm3ExpectPage(t, term, FormatterOptionsVt,
		"e\xcc\x81x\xf0\x9f\x91\xa8\xe2\x80\x8d\xf0\x9f\x91\xa9y")
	fm3ExpectPage(t, term, FormatterOptionsHtml,
		fm3HtmlOpen+"e&#769;x&#128104;&#8205;&#128105;y</div>")

	// Selecting only the tail of the wide grapheme still yields the whole
	// cluster.
	{
		f := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsPlain)
		f.start_x = 3
		f.end_x = fm3SomeCc(3)
		f.end_y = fm3SomeCc(0)
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "\xf0\x9f\x91\xa8\xe2\x80\x8d\xf0\x9f\x91\xa9", r.out)
	}

	// Point map covers each byte of the cluster with the head cell.
	{
		var pm []PointCoordinate
		// Zig-only: defer pm.deinit(alloc);
		f := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsPlain)
		f.point_map = Some(FormatterPageFormatterPointMap{map_: &pm})
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqual(t, len(r.out), len(pm))
		// e + U+0301 (1 + 2 bytes) at x=0.
		for _, c := range pm[0:3] {
			expectEqual(t, fm3Coord(0, 0), c)
		}
		// x at x=1.
		expectEqual(t, fm3Coord(1, 0), pm[3])
		// 4 + 3 + 4 bytes at x=2.
		for _, c := range pm[4:15] {
			expectEqual(t, fm3Coord(2, 0), c)
		}
		// y at x=4.
		expectEqual(t, fm3Coord(4, 0), pm[15])
	}
}

// Zig: test "term51: PageFormatter.formatWithState: vt and html style transitions"
func TestFm3_term51_PageFormatter_formatWithState_vt_and_html_style_transitions(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 2)
	defer term.Deinit()
	// a (default) b (bold) c (italic) d (default) e (italic) f (italic)
	fm3Write(t, term, "a\x1b[1mb\x1b[0;3mc\x1b[0md\x1b[3me\x1b[3mf")

	fm3ExpectPage(t, term, FormatterOptionsVt,
		"a\x1b[0m\x1b[1mb\x1b[0m\x1b[3mc\x1b[0md\x1b[0m\x1b[3mef\x1b[0m")
	fm3ExpectPage(t, term, FormatterOptionsHtml,
		fm3HtmlOpen+"a"+
			"<div style=\"display: inline;font-weight: bold;\">b</div>"+
			"<div style=\"display: inline;font-style: italic;\">c</div>"+
			"d"+
			"<div style=\"display: inline;font-style: italic;\">ef</div>"+
			"</div>")
	// Plain ignores styles.
	fm3ExpectPage(t, term, FormatterOptionsPlain, "abcdef")

	// Point map for VT: style bytes map to the styled cell and the closing
	// reset maps to the previous byte.
	{
		term2 := fm3Init(t, 10, 2)
		defer term2.Deinit()
		fm3Write(t, term2, "a\x1b[1mb\x1b[0mc")
		var pm []PointCoordinate
		// Zig-only: defer pm.deinit(alloc);
		f := FormatterPageFormatterInit(fm3Page(term2), FormatterOptionsVt)
		f.point_map = Some(FormatterPageFormatterPointMap{map_: &pm})
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "a\x1b[0m\x1b[1mb\x1b[0mc", r.out)
		expectEqual(t, 15, len(pm))
		expectEqual(t, fm3Coord(0, 0), pm[0])
		for _, c := range pm[1:14] {
			expectEqual(t, fm3Coord(1, 0), c)
		}
		expectEqual(t, fm3Coord(2, 0), pm[14])
	}
}

// Zig: test "term51: PageFormatter.formatWithState: every sgr attribute and color kind end to end"
func TestFm3_term51_PageFormatter_formatWithState_every_sgr_attribute_and_color_kind_end_to_end(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 40, 2)
	defer term.Deinit()

	// One cell per attribute, each reset afterwards.
	fm3Write(t, term, "\x1b[1mA\x1b[0m\x1b[2mB\x1b[0m\x1b[3mC\x1b[0m\x1b[4mD\x1b[0m"+
		"\x1b[5mE\x1b[0m\x1b[7mF\x1b[0m\x1b[8mG\x1b[0m\x1b[9mH\x1b[0m\x1b[53mI\x1b[0m")
	fm3ExpectPage(t, term, FormatterOptionsVt, "\x1b[0m\x1b[1mA"+
		"\x1b[0m\x1b[2mB"+
		"\x1b[0m\x1b[3mC"+
		"\x1b[0m\x1b[4mD"+
		"\x1b[0m\x1b[5mE"+
		"\x1b[0m\x1b[7mF"+
		"\x1b[0m\x1b[8mG"+
		"\x1b[0m\x1b[9mH"+
		"\x1b[0m\x1b[53mI\x1b[0m")

	term2 := fm3Init(t, 40, 2)
	defer term2.Deinit()
	// Underline variants and every color kind.
	fm3Write(t, term2, "\x1b[4:2mA\x1b[0m\x1b[4:3mB\x1b[0m\x1b[4:4mC\x1b[0m\x1b[4:5mD\x1b[0m"+
		"\x1b[38;5;200mE\x1b[0m\x1b[48;5;100mF\x1b[0m\x1b[38;2;1;2;3mG\x1b[0m"+
		"\x1b[48;2;4;5;6mH\x1b[0m\x1b[4m\x1b[58;5;7mI\x1b[0m\x1b[4m\x1b[58;2;8;9;10mJ\x1b[0m"+
		"\x1b[31mK\x1b[0m\x1b[44mL\x1b[0m\x1b[91mM\x1b[0m")
	fm3ExpectPage(t, term2, FormatterOptionsVt, "\x1b[0m\x1b[4:2mA"+
		"\x1b[0m\x1b[4:3mB"+
		"\x1b[0m\x1b[4:4mC"+
		"\x1b[0m\x1b[4:5mD"+
		"\x1b[0m\x1b[38;5;200mE"+
		"\x1b[0m\x1b[48;5;100mF"+
		"\x1b[0m\x1b[38;2;1;2;3mG"+
		"\x1b[0m\x1b[48;2;4;5;6mH"+
		"\x1b[0m\x1b[4m\x1b[58;5;7mI"+
		"\x1b[0m\x1b[4m\x1b[58;2;8;9;10mJ"+
		"\x1b[0m\x1b[38;5;1mK"+
		"\x1b[0m\x1b[48;5;4mL"+
		"\x1b[0m\x1b[38;5;9mM\x1b[0m")

	// The palette option resolves them to RGB.
	term51ZRgb100 := fm3Rgb(ColorDefault[100])
	term51ZRgb7 := fm3Rgb(ColorDefault[7])
	term51ZRgb4 := fm3Rgb(ColorDefault[4])
	term51ZRgb9 := fm3Rgb(ColorDefault[9])
	palette := ColorDefault
	palette[1] = ColorRGB{r: 11, g: 12, b: 13}
	palette[200] = ColorRGB{r: 21, g: 22, b: 23}
	palette_opts := fm3Opts(FormatterFormatVt)
	palette_opts.palette = &palette
	fm3ExpectPage(t, term2, palette_opts, "\x1b[0m\x1b[4:2mA"+
		"\x1b[0m\x1b[4:3mB"+
		"\x1b[0m\x1b[4:4mC"+
		"\x1b[0m\x1b[4:5mD"+
		"\x1b[0m\x1b[38;2;21;22;23mE"+
		"\x1b[0m\x1b[48;2;"+term51ZRgb100+"mF"+
		"\x1b[0m\x1b[38;2;1;2;3mG"+
		"\x1b[0m\x1b[48;2;4;5;6mH"+
		"\x1b[0m\x1b[4m\x1b[58;2;"+term51ZRgb7+"mI"+
		"\x1b[0m\x1b[4m\x1b[58;2;8;9;10mJ"+
		"\x1b[0m\x1b[38;2;11;12;13mK"+
		"\x1b[0m\x1b[48;2;"+term51ZRgb4+"mL"+
		"\x1b[0m\x1b[38;2;"+term51ZRgb9+"mM\x1b[0m")

	// HTML for the first set.
	fm3ExpectPage(t, term, FormatterOptionsHtml, fm3HtmlOpen+
		"<div style=\"display: inline;font-weight: bold;\">A</div>"+
		"<div style=\"display: inline;opacity: 0.5;\">B</div>"+
		"<div style=\"display: inline;font-style: italic;\">C</div>"+
		"<div style=\"display: inline;text-decoration-line: underline;text-decoration-style: solid;\">D</div>"+
		"<div style=\"display: inline;text-decoration-line: blink;\">E</div>"+
		"<div style=\"display: inline;filter: invert(100%);\">F</div>"+
		"<div style=\"display: inline;visibility: hidden;\">G</div>"+
		"<div style=\"display: inline;text-decoration-line: line-through;\">H</div>"+
		"<div style=\"display: inline;text-decoration-line: overline;\">I</div>"+
		"</div>")
}

// Zig: test "term51: PageFormatter.formatWithState: background only cells"
func TestFm3_term51_PageFormatter_formatWithState_background_only_cells(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 3)
	defer term.Deinit()
	// Text then a red background fill to the end of the row; next row text
	// with an rgb background fill; third row only background.
	fm3Write(t, term, "CPU:\x1b[41m\x1b[K\x1b[0m\r\nB\x1b[48;2;1;2;3m\x1b[K\x1b[0m\r\n\x1b[42m\x1b[K\x1b[0m")
	expectEqual(t, PageCellContentTagBgColorPalette, fm3Cell(term, 4, 0).content_tag)
	expectEqual(t, PageCellContentTagBgColorRgb, fm3Cell(term, 1, 1).content_tag)

	// Plain: bg cells have no text so they are trailing blanks; the row of
	// only background cells is a blank row.
	fm3ExpectPage(t, term, FormatterOptionsPlain, "CPU:\nB")

	// VT: the fill is emitted as styled spaces, closed before the newline.
	// The last row has only background cells, which count as no text, so
	// nothing is emitted for it.
	fm3ExpectPage(t, term, FormatterOptionsVt,
		"CPU:\x1b[0m\x1b[48;5;1m"+strings.Repeat(" ", 6)+"\x1b[0m\r\n"+
			"B\x1b[0m\x1b[48;2;1;2;3m"+strings.Repeat(" ", 9)+"\x1b[0m")
	fm3ExpectPage(t, term, FormatterOptionsHtml,
		fm3HtmlOpen+"CPU:"+
			"<div style=\"display: inline;background-color: var(--vt-palette-1);\">"+strings.Repeat(" ", 6)+"</div>\n"+
			"B<div style=\"display: inline;background-color: rgb(1, 2, 3);\">"+strings.Repeat(" ", 9)+"</div></div>")
}

// Zig: test "term51: PageFormatter.formatWithState: styled formats do not trim spaces"
func TestFm3_term51_PageFormatter_formatWithState_styled_formats_do_not_trim_spaces(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 3)
	defer term.Deinit()
	// "x" followed by two real spaces; another row with real spaces only;
	// then a row with "y".
	fm3Write(t, term, "x  \r\n  \r\ny")

	// Plain trims trailing spaces (and the space-only row is blank).
	fm3ExpectPage(t, term, FormatterOptionsPlain, "x\n\ny")
	// With trim off the spaces remain; the empty tail still does not.
	{
		opts := fm3Opts(FormatterFormatPlain)
		opts.trim = false
		fm3ExpectPage(t, term, opts, "x  \n  \ny")
	}
	// Styled formats keep every written space whatever `trim` says.
	fm3ExpectPage(t, term, FormatterOptionsVt, "x  \r\n  \r\ny")
	{
		opts := fm3Opts(FormatterFormatVt)
		opts.trim = true
		fm3ExpectPage(t, term, opts, "x  \r\n  \r\ny")
	}
	fm3ExpectPage(t, term, FormatterOptionsHtml, fm3HtmlOpen+"x  \n  \ny</div>")
}

// Zig: test "term51: PageFormatter.formatWithState: blank rows and cells per format"
func TestFm3_term51_PageFormatter_formatWithState_blank_rows_and_cells_per_format(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 6)
	defer term.Deinit()
	// Leading blank row, text, two blank rows, text with leading empty
	// cells and an interior gap, trailing blank rows.
	fm3Write(t, term, "\r\na\r\n\r\n\x1b[3Cb\x1b[2Cc")

	fm3ExpectPage(t, term, FormatterOptionsPlain, "\na\n\n   b  c")
	fm3ExpectPage(t, term, FormatterOptionsVt, "\r\na\r\n\r\n   b  c")
	fm3ExpectPage(t, term, FormatterOptionsHtml, fm3HtmlOpen+"\na\n\n   b  c</div>")

	// Point map of blank rows and blank cells.
	var pm []PointCoordinate
	// Zig-only: defer pm.deinit(alloc);
	f := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsPlain)
	f.point_map = Some(FormatterPageFormatterPointMap{map_: &pm})
	r := fm3Run(t, f)
	// Zig-only: defer r.deinit(alloc);
	expectEqualStrings(t, "\na\n\n   b  c", r.out)
	expectEqual(t, len(r.out), len(pm))
	// Row 0 is blank: its newline maps to the default (0,0).
	expectEqual(t, fm3Coord(0, 0), pm[0])
	// "a" at (0,1).
	expectEqual(t, fm3Coord(0, 1), pm[1])
	// Two newlines: the first inherits the previous entry, the second maps
	// to the row below it with x = 0.
	expectEqual(t, fm3Coord(0, 1), pm[2])
	expectEqual(t, fm3Coord(0, 2), pm[3])
	// Three blank cells before "b" (x=3): they are mapped walking
	// backwards from the cell.
	expectEqual(t, fm3Coord(2, 3), pm[4])
	expectEqual(t, fm3Coord(1, 3), pm[5])
	expectEqual(t, fm3Coord(0, 3), pm[6])
	expectEqual(t, fm3Coord(3, 3), pm[7])
	// Two blank cells before "c" (x=6).
	expectEqual(t, fm3Coord(5, 3), pm[8])
	expectEqual(t, fm3Coord(4, 3), pm[9])
	expectEqual(t, fm3Coord(6, 3), pm[10])
}

// Zig: test "term51: PageFormatter.formatWithState: header bytes map to the origin"
func TestFm3_term51_PageFormatter_formatWithState_header_bytes_map_to_the_origin(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 2)
	defer term.Deinit()
	fm3Write(t, term, "\r\nab")
	fg := ColorRGB{r: 1, g: 2, b: 3}

	// VT header: 19 bytes for OSC 10.
	{
		var pm []PointCoordinate
		// Zig-only: defer pm.deinit(alloc);
		opts := fm3Opts(FormatterFormatVt)
		opts.foreground = Some(fg)
		f := FormatterPageFormatterInit(fm3Page(term), opts)
		f.point_map = Some(FormatterPageFormatterPointMap{map_: &pm})
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		const header = "\x1b]10;rgb:01/02/03\x1b\\"
		expectEqualStrings(t, header+"\r\nab", r.out)
		expectEqual(t, len(r.out), len(pm))
		for _, c := range pm[0:len(header)] {
			expectEqual(t, fm3Coord(0, 0), c)
		}
		// Row 0 is blank: \r\n inherits the last map entry (the header's).
		expectEqual(t, fm3Coord(0, 0), pm[len(header)])
		expectEqual(t, fm3Coord(0, 0), pm[len(header)+1])
		expectEqual(t, fm3Coord(0, 1), pm[len(header)+2])
		expectEqual(t, fm3Coord(1, 1), pm[len(header)+3])
	}

	// HTML: header and footer.
	{
		var pm []PointCoordinate
		// Zig-only: defer pm.deinit(alloc);
		f := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsHtml)
		f.point_map = Some(FormatterPageFormatterPointMap{map_: &pm})
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, fm3HtmlOpen+"\nab</div>", r.out)
		expectEqual(t, len(r.out), len(pm))
		for _, c := range pm[0:len(fm3HtmlOpen)] {
			expectEqual(t, fm3Coord(0, 0), c)
		}
		// Footer maps to the last cell.
		for _, c := range pm[len(pm)-6:] {
			expectEqual(t, fm3Coord(1, 1), c)
		}
	}
}

// Zig: test "term51: PageFormatter.formatWithState: html hyperlinks"
func TestFm3_term51_PageFormatter_formatWithState_html_hyperlinks(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 20, 4)
	defer term.Deinit()

	// Two separate implicit links with the same URI are different links:
	// the first closes and the second opens, adjacent.
	fm3Write(t, term, "\x1b]8;;u\x1b\\a\x1b]8;;u\x1b\\b\x1b]8;;\x1b\\c\r\n")
	// The same explicit id and URI is one link, even across separate OSC 8.
	fm3Write(t, term, "\x1b]8;id=k;v\x1b\\d\x1b]8;id=k;v\x1b\\e\x1b]8;;\x1b\\\r\n")
	// A link covering two rows stays open over the newline.
	fm3Write(t, term, "\x1b]8;;w\x1b\\f\r\ng\x1b]8;;\x1b\\")

	fm3ExpectPage(t, term, FormatterOptionsHtml,
		fm3HtmlOpen+
			"<a href=\"u\">a</a><a href=\"u\">b</a>c\n"+
			"<a href=\"v\">de\n</a>"+
			"<a href=\"w\">f\ng</a></div>")

	// Plain and VT never emit hyperlinks.
	fm3ExpectPage(t, term, FormatterOptionsPlain, "abc\nde\nf\ng")
	fm3ExpectPage(t, term, FormatterOptionsVt, "abc\r\nde\r\nf\r\ng")

	// The URI escapes.
	term2 := fm3Init(t, 20, 2)
	defer term2.Deinit()
	fm3Write(t, term2, "\x1b]8;;a\"b'c<d>e&f\x1b\\x\x1b]8;;\x1b\\")
	fm3ExpectPage(t, term2, FormatterOptionsHtml,
		fm3HtmlOpen+"<a href=\"a&quot;b&#39;c&lt;d&gt;e&amp;f\">x</a></div>")
}

// Zig: test "term51: PageFormatter.formatWithState: codepoint map end to end"
func TestFm3_term51_PageFormatter_formatWithState_codepoint_map_end_to_end(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 2)
	defer term.Deinit()
	fm3Write(t, term, "a<b\xe4\xb8\xad e\xcc\x81")

	// term51ZMapInit: the entries as a codepoint map.
	m := Some([]FormatterCodepointMap{
		{range_: [2]uint32{'a', 'a'}, replacement: FormatterCodepointMapReplacement{
			Tag: FormatterCodepointMapReplacementString, String: []uint8("AA")}},
		{range_: [2]uint32{'<', '<'}, replacement: FormatterCodepointMapReplacement{
			Tag: FormatterCodepointMapReplacementCodepoint, Codepoint: '>'}},
		{range_: [2]uint32{0x4e2d, 0x4e2d}, replacement: FormatterCodepointMapReplacement{
			Tag: FormatterCodepointMapReplacementCodepoint, Codepoint: 'W'}},
		{range_: [2]uint32{0x301, 0x301}, replacement: FormatterCodepointMapReplacement{
			Tag: FormatterCodepointMapReplacementString, String: []uint8("")}},
	})
	// Zig-only: defer m.deinit(alloc);

	{
		opts := fm3Opts(FormatterFormatPlain)
		opts.codepoint_map = m
		fm3ExpectPage(t, term, opts, "AA>bW e")
	}
	{
		opts := fm3Opts(FormatterFormatVt)
		opts.codepoint_map = m
		fm3ExpectPage(t, term, opts, "AA>bW e")
	}
	{
		opts := fm3Opts(FormatterFormatHtml)
		opts.codepoint_map = m
		fm3ExpectPage(t, term, opts, fm3HtmlOpen+"AA&gt;bW e</div>")
	}

	// Replacements do not apply to blank cells or alter wide handling: the
	// wide cell is still skipped over its tail. Point map bytes follow the
	// replaced output.
	var pm []PointCoordinate
	// Zig-only: defer pm.deinit(alloc);
	opts := fm3Opts(FormatterFormatPlain)
	opts.codepoint_map = m
	f := FormatterPageFormatterInit(fm3Page(term), opts)
	f.point_map = Some(FormatterPageFormatterPointMap{map_: &pm})
	r := fm3Run(t, f)
	// Zig-only: defer r.deinit(alloc);
	expectEqualStrings(t, "AA>bW e", r.out)
	expectEqualSlices(t, []PointCoordinate{
		{x: 0, y: 0}, {x: 0, y: 0}, // AA
		{x: 1, y: 0}, // >
		{x: 2, y: 0}, // b
		{x: 3, y: 0}, // W
		{x: 5, y: 0}, // space
		{x: 6, y: 0}, // e
	}, pm)
}

// Zig: test "term51: PageFormatter.format: matches formatWithState output and discards the state"
func TestFm3_term51_PageFormatter_format_matches_formatWithState_output_and_discards_the_state(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 3)
	defer term.Deinit()
	fm3Write(t, term, "hello\r\nworld")

	for _, emit := range []FormatterFormat{FormatterFormatPlain, FormatterFormatVt, FormatterFormatHtml} {
		f := FormatterPageFormatterInit(fm3Page(term), fm3Opts(emit))
		a := fm3Out(t, f)
		// Zig-only: defer alloc.free(a);
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, r.out, a)
	}

	// Errors from the writer are propagated: a fixed writer that is too
	// small.
	var buf [4]uint8
	w := oscWriterFixed(buf[:])
	f := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsPlain)
	fm3ExpectWriteFailed(t, f.Format(&w))
}

// Zig: test "term51: PageFormatter.formatWithState: trim and wrapped rows with unwrap"
func TestFm3_term51_PageFormatter_formatWithState_trim_and_wrapped_rows_with_unwrap(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 5, 4)
	defer term.Deinit()
	// "ab" + explicit space cells run into a wrapped row: "ab   " wraps
	// (the cursor wraps only when more is printed) then "cd".
	fm3Write(t, term, "ab   cd\r\nz")
	page := fm3Page(term)
	expect(t, page.GetRow(0).wrap, "page.getRow(0).wrap")

	// Without unwrap the trailing spaces of row 0 are trimmed.
	fm3ExpectPage(t, term, FormatterOptionsPlain, "ab\ncd\nz")
	// With unwrap the carried blanks are emitted when the continuation
	// has text: explicit spaces are blank cells, so they are replayed.
	{
		opts := fm3Opts(FormatterFormatPlain)
		opts.unwrap = true
		fm3ExpectPage(t, term, opts, "ab   cd\nz")
	}
	// trim = false emits them in place instead.
	{
		opts := fm3Opts(FormatterFormatPlain)
		opts.trim = false
		fm3ExpectPage(t, term, opts, "ab   \ncd\nz")
	}
	{
		opts := fm3Opts(FormatterFormatPlain)
		opts.trim = false
		opts.unwrap = true
		fm3ExpectPage(t, term, opts, "ab   cd\nz")
	}
}

// Zig: test "term51: PageListFormatter.format: empty screen per format"
func TestFm3_term51_PageListFormatter_format_empty_screen_per_format(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 3)
	defer term.Deinit()
	pages := &term.screens.active.pages

	for _, c := range []struct {
		emit     FormatterFormat
		expected string
	}{
		{FormatterFormatPlain, ""},
		{FormatterFormatVt, ""},
		{FormatterFormatHtml, fm3HtmlOpen + "</div>"},
	} {
		f := FormatterPageListFormatterInit(pages, fm3Opts(c.emit))
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, c.expected, out)
	}

	// The pin map of an empty plain output is empty.
	var pm []PageListPin
	// Zig-only: defer pm.deinit(alloc);
	f := FormatterPageListFormatterInit(pages, FormatterOptionsPlain)
	f.pin_map = Some(FormatterPinMap{map_: &pm})
	out := fm3Out(t, f)
	_ = out
	// Zig-only: defer alloc.free(out);
	expectEqual(t, 0, len(pm))
}

// Zig: test "term51: PageListFormatter.format: html and vt emit a header per page"
func TestFm3_term51_PageListFormatter_format_html_and_vt_emit_a_header_per_page(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 3)
	defer term.Deinit()
	first_page_rows := fm3TwoPages(t, term, "page one", "page two")
	pages := &term.screens.active.pages

	// HTML: leading blank rows of the first page, then a separate wrapper
	// for each page. The newline separating the lines is emitted in the
	// second page's wrapper.
	{
		f := FormatterPageListFormatterInit(pages, FormatterOptionsHtml)
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);

		exp := fm3HtmlOpen + strings.Repeat("\n", int(first_page_rows-1)) +
			"page one</div>" + fm3HtmlOpen + "\npage two</div>"
		expectEqualStrings(t, exp, out)
	}

	// VT: the foreground/background header is repeated per page, and the
	// newline is \r\n.
	{
		opts := fm3Opts(FormatterFormatVt)
		opts.foreground = Some(ColorRGB{r: 1, g: 2, b: 3})
		f := FormatterPageListFormatterInit(pages, opts)
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);

		const hdr = "\x1b]10;rgb:01/02/03\x1b\\"
		exp := hdr + strings.Repeat("\r\n", int(first_page_rows-1)) +
			"page one" + hdr + "\r\npage two"
		expectEqualStrings(t, exp, out)
	}

	// Plain: no header and the same text.
	{
		f := FormatterPageListFormatterInit(pages, FormatterOptionsPlain)
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqual(t, int(first_page_rows-1), fm3IndexOfNone(out, "\n"))
		expectEqualStrings(t, "page one\npage two", out[first_page_rows-1:])
	}
}

// Zig: test "term51: PageListFormatter.format: bounds spanning pages"
func TestFm3_term51_PageListFormatter_format_bounds_spanning_pages(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 3)
	defer term.Deinit()
	first_page_rows := fm3TwoPages(t, term, "abcdefgh", "ijklmnop")
	pages := &term.screens.active.pages
	first_node := pages.pages.first
	last_node := pages.pages.last

	tl := PageListPin{node: first_node, x: 2, y: SizeCellCountInt(first_page_rows - 1)}
	br := PageListPin{node: last_node, x: 4, y: 0}

	// Non-rectangle: start_x on the first page only, end_x on the last.
	{
		f := FormatterPageListFormatterInit(pages, FormatterOptionsPlain)
		f.top_left = Some(tl)
		f.bottom_right = Some(br)
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "cdefgh\nijklm", out)
	}

	// Only a top-left: formats to the end of the screen.
	{
		f := FormatterPageListFormatterInit(pages, FormatterOptionsPlain)
		f.top_left = Some(tl)
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "cdefgh\nijklmnop", out)
	}

	// Only a bottom-right: from the top-left of the screen.
	{
		f := FormatterPageListFormatterInit(pages, FormatterOptionsPlain)
		f.bottom_right = Some(br)
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqual(t, int(first_page_rows-1), fm3IndexOfNone(out, "\n"))
		expectEqualStrings(t, "abcdefgh\nijklm", out[first_page_rows-1:])
	}

	// Rectangle: x bounds on every row of every page. Because the second
	// page's chunk starts with a non-zero start_x the carried newline is
	// not applied, so the two rows are joined.
	{
		f := FormatterPageListFormatterInit(pages, FormatterOptionsPlain)
		f.top_left = Some(tl)
		f.bottom_right = Some(br)
		f.rectangle = true
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "cdeklm", out)
	}

	// Rectangle starting at x=0 keeps the newline between the pages.
	{
		f := FormatterPageListFormatterInit(pages, FormatterOptionsPlain)
		f.top_left = Some(PageListPin{node: first_node, x: 0, y: SizeCellCountInt(first_page_rows - 1)})
		f.bottom_right = Some(br)
		f.rectangle = true
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "abcde\nijklm", out)
	}

	// Pin map across the page boundary.
	{
		var pm []PageListPin
		// Zig-only: defer pm.deinit(alloc);
		f := FormatterPageListFormatterInit(pages, FormatterOptionsPlain)
		f.top_left = Some(tl)
		f.bottom_right = Some(br)
		f.pin_map = Some(FormatterPinMap{map_: &pm})
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "cdefgh\nijklm", out)
		expectEqual(t, len(out), len(pm))
		y := SizeCellCountInt(first_page_rows - 1)
		for i := 0; i < 6; i++ {
			expectEqual(t, PageListPin{node: first_node, x: SizeCellCountInt(2 + i), y: y}, pm[i])
		}
		// The newline belongs to the second page's chunk.
		expectEqual(t, last_node, pm[6].node)
		for i := 0; i < 5; i++ {
			expectEqual(t, PageListPin{node: last_node, x: SizeCellCountInt(i), y: 0}, pm[7+i])
		}
	}
}

// Zig: test "term51: PageListFormatter.format: selection orders and rectangles through ScreenFormatter"
func TestFm3_term51_PageListFormatter_format_selection_orders_and_rectangles_through_ScreenFormatter(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 4)
	defer term.Deinit()
	fm3Write(t, term, "0123456789\r\nabcdefghij\r\nklmnopqrst\r\nuvwxyz")

	// Reverse order selection is normalized.
	{
		sel := SelectionInit(fm3Pin(t, term, 2, 2), fm3Pin(t, term, 7, 0), false)
		out := fm3SelOut(t, term, FormatterOptionsPlain, sel)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "789\nabcdefghij\nklm", out)
	}

	// Rectangles in all four directions give the same box.
	corners := [][2]PageListPin{
		{fm3Pin(t, term, 2, 0), fm3Pin(t, term, 4, 2)}, // forward
		{fm3Pin(t, term, 4, 2), fm3Pin(t, term, 2, 0)}, // reverse
		{fm3Pin(t, term, 4, 0), fm3Pin(t, term, 2, 2)}, // mirrored forward
		{fm3Pin(t, term, 2, 2), fm3Pin(t, term, 4, 0)}, // mirrored reverse
	}
	for _, c := range corners {
		sel := SelectionInit(c[0], c[1], true)
		out := fm3SelOut(t, term, FormatterOptionsPlain, sel)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "234\ncde\nmno", out)
	}

	// A rectangle spanning blank rows: rows with no text in the box are
	// blank lines in the middle and dropped at the end.
	{
		sel := SelectionInit(fm3Pin(t, term, 7, 2), fm3Pin(t, term, 9, 3), true)
		out := fm3SelOut(t, term, FormatterOptionsPlain, sel)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "rst", out)
	}
	{
		sel := SelectionInit(fm3Pin(t, term, 7, 3), fm3Pin(t, term, 9, 3), false)
		out := fm3SelOut(t, term, FormatterOptionsPlain, sel)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "", out)
	}
	// Leading blank rows are kept; trailing ones dropped.
	{
		term2 := fm3Init(t, 10, 5)
		defer term2.Deinit()
		fm3Write(t, term2, "\r\n\r\nabc")
		sel := SelectionInit(fm3Pin(t, term2, 0, 0), fm3Pin(t, term2, 9, 4), false)
		out := fm3SelOut(t, term2, FormatterOptionsPlain, sel)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\n\nabc", out)
	}
}

// Zig: test "term51: PageListFormatter.format: selection in wrapped lines and wide characters"
func TestFm3_term51_PageListFormatter_format_selection_in_wrapped_lines_and_wide_characters(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 5, 4)
	defer term.Deinit()
	// Rows: "hello" (wrap), " worl" (wrap), "d".
	fm3Write(t, term, "hello world")

	sel := SelectionInit(fm3Pin(t, term, 2, 0), fm3Pin(t, term, 0, 2), false)
	{
		out := fm3SelOut(t, term, FormatterOptionsPlain, sel)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "llo\n worl\nd", out)
	}
	{
		opts := fm3Opts(FormatterFormatPlain)
		opts.unwrap = true
		out := fm3SelOut(t, term, opts, sel)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "llo world", out)
	}
	// Selection that ends mid-way through a wrapped line.
	{
		sel2 := SelectionInit(fm3Pin(t, term, 3, 0), fm3Pin(t, term, 2, 1), false)
		opts := fm3Opts(FormatterFormatPlain)
		opts.unwrap = true
		out := fm3SelOut(t, term, opts, sel2)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "lo wo", out)
	}

	// Wide characters: a [wide][tail] b [wide][tail] c.
	term2 := fm3Init(t, 10, 2)
	defer term2.Deinit()
	fm3Write(t, term2, "a\xe4\xb8\xadb\xe6\x96\x87c")
	const zhong = "\xe4\xb8\xad"
	const wen = "\xe6\x96\x87"
	{
		// Starts on a tail, ends on a head.
		s2 := SelectionInit(fm3Pin(t, term2, 2, 0), fm3Pin(t, term2, 4, 0), false)
		out := fm3SelOut(t, term2, FormatterOptionsPlain, s2)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, zhong+"b"+wen, out)
	}
	{
		// Starts on a head and ends on a tail.
		s2 := SelectionInit(fm3Pin(t, term2, 1, 0), fm3Pin(t, term2, 5, 0), false)
		out := fm3SelOut(t, term2, FormatterOptionsPlain, s2)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, zhong+"b"+wen, out)
	}
	{
		// Ends just before a wide character.
		s2 := SelectionInit(fm3Pin(t, term2, 0, 0), fm3Pin(t, term2, 3, 0), false)
		out := fm3SelOut(t, term2, FormatterOptionsPlain, s2)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "a"+zhong+"b", out)
	}
}

// Zig: test "term51: ScreenFormatter.format: content none and extras per format"
func TestFm3_term51_ScreenFormatter_format_content_none_and_extras_per_format(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 5)
	defer term.Deinit()
	fm3Write(t, term, "hi")

	// No content, no extras.
	{
		f := FormatterScreenFormatterInit(term.screens.active, FormatterOptionsVt)
		f.content = fm3ContentNone
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "", out)
	}

	// Content with no extras for VT, with a trailing-blank trimmed result.
	{
		f := FormatterScreenFormatterInit(term.screens.active, FormatterOptionsVt)
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "hi", out)
	}

	// Plain and html ignore all extras.
	{
		f := FormatterScreenFormatterInit(term.screens.active, FormatterOptionsPlain)
		f.extra = FormatterScreenFormatterExtraAll
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "hi", out)
	}
	{
		f := FormatterScreenFormatterInit(term.screens.active, FormatterOptionsHtml)
		f.extra = FormatterScreenFormatterExtraAll
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, fm3HtmlOpen+"hi</div>", out)
	}
	{
		f := FormatterScreenFormatterInit(term.screens.active, FormatterOptionsPlain)
		f.content = fm3ContentNone
		f.extra = FormatterScreenFormatterExtraAll
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "", out)
	}
}

// Zig: test "term51: ScreenFormatter.format: each extra on a default screen"
func TestFm3_term51_ScreenFormatter_format_each_extra_on_a_default_screen(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 5)
	defer term.Deinit()

	type Case struct {
		field    string
		expected string
	}
	for _, c := range []Case{
		// The default style still emits a reset.
		{field: "style", expected: "\x1b[0m"},
		{field: "hyperlink", expected: ""},
		{field: "protection", expected: ""},
		{field: "kitty_keyboard", expected: ""},
		{field: "charsets", expected: ""},
		{field: "cursor", expected: "\x1b[1;1H"},
	} {
		f := FormatterScreenFormatterInit(term.screens.active, FormatterOptionsVt)
		f.content = fm3ContentNone
		f.extra = FormatterScreenFormatterExtraNone
		fm3SetExtraField(t, &f.extra, c.field)
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, c.expected, out)
	}
}

// Zig: test "term51: ScreenFormatter.format: style extra"
func TestFm3_term51_ScreenFormatter_format_style_extra(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 5)
	defer term.Deinit()

	f := FormatterScreenFormatterInit(term.screens.active, FormatterOptionsVt)
	f.content = fm3ContentNone
	f.extra.style = true

	fm3Write(t, term, "\x1b[1;3;4:2;9;38;5;9;48;2;4;5;6;58;2;7;8;9m")
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t,
			"\x1b[0m\x1b[1m\x1b[3m\x1b[9m\x1b[4:2m\x1b[38;5;9m\x1b[48;2;4;5;6m\x1b[58;2;7;8;9m",
			out)
	}

	// The palette option is not used by the screen's style extra: indices
	// stay indices.
	palette := ColorDefault
	palette[9] = ColorRGB{r: 1, g: 1, b: 1}
	{
		fp := f
		fp.opts.palette = &palette
		out := fm3Out(t, fp)
		// Zig-only: defer alloc.free(out);
		expect(t, strings.Contains(out, "\x1b[38;5;9m"), "std.mem.indexOf(u8, out, \"\\x1b[38;5;9m\") != null")
	}

	fm3Write(t, term, "\x1b[0m")
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b[0m", out)
	}
}

// Zig: test "term51: ScreenFormatter.format: hyperlink extra"
func TestFm3_term51_ScreenFormatter_format_hyperlink_extra(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 5)
	defer term.Deinit()

	f := FormatterScreenFormatterInit(term.screens.active, FormatterOptionsVt)
	f.content = fm3ContentNone
	f.extra.hyperlink = true

	// Implicit id: only the URI is emitted.
	fm3Write(t, term, "\x1b]8;;http://example.com/a?b=1\x1b\\")
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b]8;;http://example.com/a?b=1\x1b\\", out)
	}

	// Explicit id.
	fm3Write(t, term, "\x1b]8;id=foo;http://example.org\x1b\\")
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b]8;id=foo;http://example.org\x1b\\", out)
	}

	// Ended.
	fm3Write(t, term, "\x1b]8;;\x1b\\")
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "", out)
	}
}

// Zig: test "term51: ScreenFormatter.format: protection and kitty keyboard extras"
func TestFm3_term51_ScreenFormatter_format_protection_and_kitty_keyboard_extras(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 5)
	defer term.Deinit()

	fp := FormatterScreenFormatterInit(term.screens.active, FormatterOptionsVt)
	fp.content = fm3ContentNone
	fp.extra.protection = true
	fk := FormatterScreenFormatterInit(term.screens.active, FormatterOptionsVt)
	fk.content = fm3ContentNone
	fk.extra.kitty_keyboard = true

	// Protection: DECSCA 1 protects, 0 and 2 unprotect.
	fm3Write(t, term, "\x1b[1\"q")
	expect(t, term.screens.active.cursor.protected, "t.screens.active.cursor.protected")
	{
		out := fm3Out(t, fp)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b[1\"q", out)
	}
	fm3Write(t, term, "\x1b[2\"q")
	{
		out := fm3Out(t, fp)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "", out)
	}
	// ISO protection (SPA) sets the same cursor flag.
	term.screens.active.cursor.protected = true
	{
		out := fm3Out(t, fp)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b[1\"q", out)
	}

	// Kitty keyboard: nothing when disabled; the flags integer otherwise.
	{
		out := fm3Out(t, fk)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "", out)
	}
	fm3Write(t, term, "\x1b[=1;1u")
	{
		out := fm3Out(t, fk)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b[=1;1u", out)
	}
	fm3Write(t, term, "\x1b[=31;1u")
	{
		out := fm3Out(t, fk)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b[=31;1u", out)
	}
	fm3Write(t, term, "\x1b[=0;1u")
	{
		out := fm3Out(t, fk)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "", out)
	}
	// The flags of the top of the stack are used.
	fm3Write(t, term, "\x1b[=2;1u\x1b[>4u")
	{
		out := fm3Out(t, fk)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b[=4;1u", out)
	}
}

// Zig: test "term51: ScreenFormatter.format: charsets extra designations and invocations"
func TestFm3_term51_ScreenFormatter_format_charsets_extra_designations_and_invocations(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 5)
	defer term.Deinit()
	screen := term.screens.active

	f := FormatterScreenFormatterInit(screen, FormatterOptionsVt)
	f.content = fm3ContentNone
	f.extra.charsets = true

	// Defaults: all UTF-8, GL=G0, GR=G2: nothing.
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "", out)
	}

	// Each slot with each non-default charset.
	slots := []struct {
		slot  CharsetsSlots
		inter string
	}{
		{slot: CharsetsSlotsG0, inter: "("},
		{slot: CharsetsSlotsG1, inter: ")"},
		{slot: CharsetsSlotsG2, inter: "*"},
		{slot: CharsetsSlotsG3, inter: "+"},
	}
	sets := []struct {
		cs    CharsetsCharset
		final string
	}{
		{cs: CharsetsCharsetAscii, final: "B"},
		{cs: CharsetsCharsetBritish, final: "A"},
		{cs: CharsetsCharsetDecSpecial, final: "0"},
	}
	for _, s := range slots {
		for _, c := range sets {
			screen.charset.charsets.Set(s.slot, c.cs)
			out := fm3Out(t, f)
			// Zig-only: defer alloc.free(out);
			expectEqualStrings(t, "\x1b", out[0:1])
			expectEqualStrings(t, s.inter, out[1:2])
			expectEqualStrings(t, c.final, out[2:3])
			expectEqual(t, 3, len(out))
			screen.charset.charsets.Set(s.slot, CharsetsCharsetUtf8)
		}
	}

	// Multiple designations are emitted in slot order.
	screen.charset.charsets.Set(CharsetsSlotsG3, CharsetsCharsetBritish)
	screen.charset.charsets.Set(CharsetsSlotsG1, CharsetsCharsetDecSpecial)
	screen.charset.charsets.Set(CharsetsSlotsG0, CharsetsCharsetAscii)
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b(B\x1b)0\x1b+A", out)
	}
	screen.charset.charsets.Set(CharsetsSlotsG3, CharsetsCharsetUtf8)
	screen.charset.charsets.Set(CharsetsSlotsG1, CharsetsCharsetUtf8)
	screen.charset.charsets.Set(CharsetsSlotsG0, CharsetsCharsetUtf8)

	// GL invocations.
	gls := []struct {
		gl  CharsetsSlots
		seq string
	}{
		{gl: CharsetsSlotsG1, seq: "\x0e"},
		{gl: CharsetsSlotsG2, seq: "\x1bn"},
		{gl: CharsetsSlotsG3, seq: "\x1bo"},
	}
	for _, c := range gls {
		screen.charset.gl = c.gl
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, c.seq, out)
	}
	screen.charset.gl = CharsetsSlotsG0

	// GR invocations (G2 is the default).
	grs := []struct {
		gr  CharsetsSlots
		seq string
	}{
		{gr: CharsetsSlotsG1, seq: "\x1b~"},
		{gr: CharsetsSlotsG3, seq: "\x1b|"},
	}
	for _, c := range grs {
		screen.charset.gr = c.gr
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, c.seq, out)
	}

	// Everything together: designations, then GL, then GR.
	screen.charset.charsets.Set(CharsetsSlotsG2, CharsetsCharsetDecSpecial)
	screen.charset.gl = CharsetsSlotsG3
	screen.charset.gr = CharsetsSlotsG1
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b*0\x1bo\x1b~", out)
	}
}

// Zig: test "term51: ScreenFormatter.format: charsets set through the stream round trip"
func TestFm3_term51_ScreenFormatter_format_charsets_set_through_the_stream_round_trip(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 5)
	defer term.Deinit()
	fm3Write(t, term, "\x1b(0\x1b)A\x1b*B\x1b+0\x0e\x1b|")

	f := FormatterScreenFormatterInit(term.screens.active, FormatterOptionsVt)
	f.content = fm3ContentNone
	f.extra.charsets = true
	out := fm3Out(t, f)
	// Zig-only: defer alloc.free(out);
	expectEqualStrings(t, "\x1b(0\x1b)A\x1b*B\x1b+0\x0e\x1b|", out)

	term2 := fm3Init(t, 10, 5)
	defer term2.Deinit()
	fm3Write(t, term2, out)
	for _, slot := range []CharsetsSlots{CharsetsSlotsG0, CharsetsSlotsG1, CharsetsSlotsG2, CharsetsSlotsG3} {
		expectEqual(t,
			term.screens.active.charset.charsets.Get(slot),
			term2.screens.active.charset.charsets.Get(slot),
		)
	}
	expectEqual(t, term.screens.active.charset.gl, term2.screens.active.charset.gl)
	expectEqual(t, term.screens.active.charset.gr, term2.screens.active.charset.gr)
}

// Zig: test "term51: ScreenFormatter.format: cursor extra positions"
func TestFm3_term51_ScreenFormatter_format_cursor_extra_positions(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 5)
	defer term.Deinit()

	f := FormatterScreenFormatterInit(term.screens.active, FormatterOptionsVt)
	f.content = fm3ContentNone
	f.extra.cursor = true

	cases := []struct {
		input    string
		expected string
	}{
		{input: "\x1b[H", expected: "\x1b[1;1H"},
		{input: "\x1b[3;4H", expected: "\x1b[3;4H"},
		{input: "\x1b[5;10H", expected: "\x1b[5;10H"},
		{input: "\x1b[1;10H", expected: "\x1b[1;10H"},
		// Beyond the screen clamps.
		{input: "\x1b[99;99H", expected: "\x1b[5;10H"},
	}
	for _, c := range cases {
		fm3Write(t, term, c.input)
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, c.expected, out)
	}

	// A pending wrap still reports the last column.
	fm3Write(t, term, "\x1b[H0123456789")
	expect(t, term.screens.active.cursor.pending_wrap, "t.screens.active.cursor.pending_wrap")
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b[1;10H", out)
	}
}

// Zig: test "term51: ScreenFormatter.format: extras follow content in a fixed order"
func TestFm3_term51_ScreenFormatter_format_extras_follow_content_in_a_fixed_order(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 5)
	defer term.Deinit()
	fm3Write(t, term, "hi\r\nyo")
	fm3Write(t, term, "\x1b[1m\x1b]8;id=x;http://u\x1b\\\x1b[1\"q\x1b[=3;1u\x1b(0\x0e\x1b[2;2H")

	f := FormatterScreenFormatterInit(term.screens.active, FormatterOptionsVt)
	f.extra = FormatterScreenFormatterExtraAll
	out := fm3Out(t, f)
	// Zig-only: defer alloc.free(out);
	expectEqualStrings(t,
		"hi\r\nyo"+
			"\x1b[0m\x1b[1m"+ // style
			"\x1b]8;id=x;http://u\x1b\\"+ // hyperlink
			"\x1b[1\"q"+ // protection
			"\x1b[=3;1u"+ // kitty keyboard
			"\x1b(0\x0e"+ // charsets
			"\x1b[2;2H", // cursor
		out,
	)

	// Selection content with extras: the extras do not depend on the
	// selection.
	sel := SelectionInit(fm3Pin(t, term, 0, 0), fm3Pin(t, term, 1, 0), false)
	f.content = fm3ContentSelection(sel)
	out2 := fm3Out(t, f)
	// Zig-only: defer alloc.free(out2);
	expect(t, strings.HasPrefix(out2, "hi"), "std.mem.startsWith(u8, out2, \"hi\")")
	expectEqualStrings(t, out[len("hi\r\nyo"):], out2[len("hi"):])

	// Round trip.
	term2 := fm3Init(t, 10, 5)
	defer term2.Deinit()
	fm3Write(t, term2, out)
	s1 := term.screens.active
	s2 := term2.screens.active
	expectEqual(t, s1.cursor.x, s2.cursor.x)
	expectEqual(t, s1.cursor.y, s2.cursor.y)
	expect(t, s1.cursor.style.Eql(s2.cursor.style), "s1.cursor.style.eql(s2.cursor.style)")
	expect(t, s2.cursor.protected, "s2.cursor.protected")
	expect(t, s1.cursor.hyperlink != nil, "s1.cursor.hyperlink != null")
	expect(t, s2.cursor.hyperlink != nil, "s2.cursor.hyperlink != null")
	expectEqualStrings(t, string(s1.cursor.hyperlink.uri), string(s2.cursor.hyperlink.uri))
	expectEqual(t, s1.kitty_keyboard.Current().Int(), s2.kitty_keyboard.Current().Int())
	expectEqual(t, s1.charset.gl, s2.charset.gl)
}

// Zig: test "term51: ScreenFormatter.format: pin map for extras"
func TestFm3_term51_ScreenFormatter_format_pin_map_for_extras(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 5)
	defer term.Deinit()
	fm3Write(t, term, "hi\x1b[1m")
	node := term.screens.active.pages.pages.first

	// Extras after content map to the last content pin.
	{
		var pm []PageListPin
		// Zig-only: defer pm.deinit(alloc);
		f := FormatterScreenFormatterInit(term.screens.active, FormatterOptionsVt)
		f.extra.style = true
		f.extra.cursor = true
		f.pin_map = Some(FormatterPinMap{map_: &pm})
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "hi\x1b[0m\x1b[1m\x1b[1;3H", out)
		expectEqual(t, len(out), len(pm))
		expectEqual(t, PageListPin{node: node, x: 0, y: 0}, pm[0])
		expectEqual(t, PageListPin{node: node, x: 1, y: 0}, pm[1])
		for _, p := range pm[2:] {
			expectEqual(t, PageListPin{node: node, x: 1, y: 0}, p)
		}
	}

	// Without content the extras map to the top-left of the screen.
	{
		var pm []PageListPin
		// Zig-only: defer pm.deinit(alloc);
		f := FormatterScreenFormatterInit(term.screens.active, FormatterOptionsVt)
		f.content = fm3ContentNone
		f.extra.cursor = true
		f.pin_map = Some(FormatterPinMap{map_: &pm})
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b[1;3H", out)
		expectEqual(t, len(out), len(pm))
		for _, p := range pm {
			expectEqual(t, PageListPin{node: node, x: 0, y: 0}, p)
		}
	}

	// Plain and html with extras set do not add map entries.
	{
		var pm []PageListPin
		// Zig-only: defer pm.deinit(alloc);
		f := FormatterScreenFormatterInit(term.screens.active, FormatterOptionsPlain)
		f.extra = FormatterScreenFormatterExtraAll
		f.pin_map = Some(FormatterPinMap{map_: &pm})
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "hi", out)
		expectEqual(t, 2, len(pm))
	}

	// VT with no extras set: only the content.
	{
		var pm []PageListPin
		// Zig-only: defer pm.deinit(alloc);
		f := FormatterScreenFormatterInit(term.screens.active, FormatterOptionsVt)
		f.pin_map = Some(FormatterPinMap{map_: &pm})
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "hi", out)
		expectEqual(t, 2, len(pm))
	}
}

// Zig: test "term51: ScreenFormatter.format: content selection null covers the whole screen"
func TestFm3_term51_ScreenFormatter_format_content_selection_null_covers_the_whole_screen(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 6, 3)
	defer term.Deinit()
	fm3Write(t, term, "ab\r\ncd\r\nef")

	f := FormatterScreenFormatterInit(term.screens.active, FormatterOptionsPlain)
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "ab\ncd\nef", out)
	}
	f.content = FormatterScreenFormatterContent{Tag: FormatterScreenFormatterContentSelection}
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "ab\ncd\nef", out)
	}
	// The screen's own selection is not consulted.
	zigTry(t, term.screens.active.Select(Some(SelectionInit(fm3Pin(t, term, 0, 1), fm3Pin(t, term, 1, 1), false))))
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "ab\ncd\nef", out)
	}
}

// Zig: test "term51: TerminalFormatter.format: plain ignores every extra"
func TestFm3_term51_TerminalFormatter_format_plain_ignores_every_extra(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 3)
	defer term.Deinit()
	fm3Write(t, term, "\x1b[?2004h\x1b[2;3rhello\x1b[>4;2m\x1b]7;file://h/p\x1b\\")

	f := FormatterTerminalFormatterInit(term, FormatterOptionsPlain)
	f.extra = FormatterTerminalFormatterExtraAll
	out := fm3Out(t, f)
	// Zig-only: defer alloc.free(out);
	expectEqualStrings(t, "hello", out)

	f.content = fm3ContentNone
	out2 := fm3Out(t, f)
	// Zig-only: defer alloc.free(out2);
	expectEqualStrings(t, "", out2)
}

// Zig: test "term51: TerminalFormatter.format: palette extra for vt and html"
func TestFm3_term51_TerminalFormatter_format_palette_extra_for_vt_and_html(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 3)
	defer term.Deinit()
	fm3Write(t, term, "\x1b]4;0;rgb:12/34/56\x1b\\\x1b]4;17;rgb:0a/0b/0c\x1b\\\x1b]4;255;rgb:ff/00/ff\x1b\\")

	// VT: 256 OSC 4 sequences, one per palette entry, in index order.
	{
		f := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
		f.content = fm3ContentNone
		f.extra = FormatterTerminalFormatterExtraNone
		f.extra.palette = true
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);

		expect(t, strings.HasPrefix(out, "\x1b]4;0;rgb:12/34/56\x1b\\\x1b]4;1;rgb:"),
			"std.mem.startsWith(u8, out, \"\\x1b]4;0;rgb:12/34/56\\x1b\\\\\\x1b]4;1;rgb:\")")
		expect(t, strings.HasSuffix(out, "\x1b]4;254;rgb:"+
			fm3Hex(ColorDefault[254])+"\x1b\\\x1b]4;255;rgb:ff/00/ff\x1b\\"),
			"std.mem.endsWith(u8, out, \"\\x1b]4;254;rgb:\" ++ ... ++ \"\\x1b]4;255;rgb:ff/00/ff\\x1b\\\\\")")
		expect(t, strings.Contains(out, "\x1b]4;17;rgb:0a/0b/0c\x1b\\\x1b]4;18;"),
			"std.mem.indexOf(u8, out, \"\\x1b]4;17;rgb:0a/0b/0c\\x1b\\\\\\x1b]4;18;\") != null")
		expectEqual(t, 256, strings.Count(out, "\x1b]4;"))
		// 256 * 19 fixed bytes plus the digits of the indices (10 * 1,
		// 90 * 2, 156 * 3).
		expectEqual(t, 256*19+10+180+468, len(out))
	}

	// HTML: one style block of CSS variables.
	{
		f := FormatterTerminalFormatterInit(term, FormatterOptionsHtml)
		f.content = fm3ContentNone
		f.extra = FormatterTerminalFormatterExtraNone
		f.extra.palette = true
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);

		expect(t, strings.HasPrefix(out, "<style>:root{--vt-palette-0: #123456;--vt-palette-1: #"),
			"std.mem.startsWith(u8, out, \"<style>:root{--vt-palette-0: #123456;--vt-palette-1: #\")")
		expect(t, strings.HasSuffix(out, "--vt-palette-255: #ff00ff;}</style>"),
			"std.mem.endsWith(u8, out, \"--vt-palette-255: #ff00ff;}</style>\")")
		expect(t, strings.Contains(out, "--vt-palette-17: #0a0b0c;--vt-palette-18: #"),
			"std.mem.indexOf(u8, out, \"--vt-palette-17: #0a0b0c;--vt-palette-18: #\") != null")
		expectEqual(t, 256, strings.Count(out, "--vt-palette-"))
		// 256 * 23 fixed bytes + digits + "<style>:root{" + "}</style>".
		expectEqual(t, 256*23+10+180+468+13+9, len(out))
	}

	// Palette extra false: nothing, for both formats, even with content.
	for _, emit := range []FormatterFormat{FormatterFormatVt, FormatterFormatHtml} {
		f := FormatterTerminalFormatterInit(term, fm3Opts(emit))
		f.extra = FormatterTerminalFormatterExtraNone
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expect(t, !strings.Contains(out, "palette"), "std.mem.indexOf(u8, out, \"palette\") == null")
		expect(t, !strings.Contains(out, "\x1b]4;"), "std.mem.indexOf(u8, out, \"\\x1b]4;\") == null")
	}

	// The palette precedes the content.
	{
		fm3Write(t, term, "\x1b[Hxy")
		f := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expect(t, strings.HasSuffix(out, "\x1b]4;255;rgb:ff/00/ff\x1b\\xy\x1b[0m"),
			"std.mem.endsWith(u8, out, \"\\x1b]4;255;rgb:ff/00/ff\\x1b\\\\xy\\x1b[0m\")")
	}
}

// Zig: test "term51: TerminalFormatter.format: every mode individually"
func TestFm3_term51_TerminalFormatter_format_every_mode_individually(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 3)
	defer term.Deinit()

	// The table covers every mode of the enum exactly once.
	expectEqual(t, len(ModesModeValues()), len(fm3Modes))

	f := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
	f.content = fm3ContentNone
	f.extra = FormatterTerminalFormatterExtraNone
	f.extra.modes = true

	// No change from defaults: nothing.
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "", out)
	}

	for _, m := range fm3Modes {
		func() {
			dflt := term.modes.Get(m.mode)
			term.modes.Set(m.mode, !dflt)
			defer term.modes.Set(m.mode, dflt)

			out := fm3Out(t, f)
			// Zig-only: defer alloc.free(out);
			// A mode turned on is "h", turned off "l".
			suffix := "h"
			if dflt {
				suffix = "l"
			}
			expected := fmt.Sprintf("\x1b[%s%s", m.code, suffix)
			// Zig-only: defer alloc.free(expected);
			expectEqualStrings(t, expected, out)
		}()
	}
}

// Zig: test "term51: TerminalFormatter.format: all modes changed in declaration order"
func TestFm3_term51_TerminalFormatter_format_all_modes_changed_in_declaration_order(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 3)
	defer term.Deinit()

	for _, m := range fm3Modes {
		term.modes.Set(m.mode, !term.modes.Get(m.mode))
	}

	f := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
	f.content = fm3ContentNone
	f.extra = FormatterTerminalFormatterExtraNone
	f.extra.modes = true
	out := fm3Out(t, f)
	// Zig-only: defer alloc.free(out);
	expectEqualStrings(t,
		"\x1b[2h\x1b[4h\x1b[12l\x1b[20h"+
			"\x1b[?1h\x1b[?3h\x1b[?4h\x1b[?5h\x1b[?6h\x1b[?7l\x1b[?8h\x1b[?9h"+
			"\x1b[?12h\x1b[?25l\x1b[?40h\x1b[?45h\x1b[?47h\x1b[?66h\x1b[?69h"+
			"\x1b[?1000h\x1b[?1002h\x1b[?1003h\x1b[?1004h\x1b[?1005h\x1b[?1006h"+
			"\x1b[?1007l\x1b[?1015h\x1b[?1016h\x1b[?1035l\x1b[?1036l\x1b[?1039h"+
			"\x1b[?1045h\x1b[?1047h\x1b[?1048h\x1b[?1049h\x1b[?2004h\x1b[?2026h"+
			"\x1b[?2027h\x1b[?2031h\x1b[?2048h",
		out,
	)

	// Plain and html never emit modes.
	for _, emit := range []FormatterFormat{FormatterFormatPlain, FormatterFormatHtml} {
		fm := FormatterTerminalFormatterInit(term, fm3Opts(emit))
		fm.content = fm3ContentNone
		fm.extra = FormatterTerminalFormatterExtraNone
		fm.extra.modes = true
		o := fm3Out(t, fm)
		// Zig-only: defer alloc.free(o);
		expectEqualStrings(t, "", o)
	}
}

// Zig: test "term51: TerminalFormatter.format: modes are compared with the terminal's defaults"
func TestFm3_term51_TerminalFormatter_format_modes_are_compared_with_the_terminals_defaults(t *testing.T) {
	testStdCapacity(t)
	opts := terminalOptionsDefault(10, 3)
	opts.default_modes.bracketed_paste = true
	opts.default_modes.wraparound = false
	term_, err := TerminalInit(opts)
	zigTry(t, err)
	term := &term_
	defer term.Deinit()

	f := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
	f.content = fm3ContentNone
	f.extra = FormatterTerminalFormatterExtraNone
	f.extra.modes = true

	// Equal to the configured defaults: nothing.
	expect(t, term.modes.Get(ModesModeBracketedPaste), "t.modes.get(.bracketed_paste)")
	expect(t, !term.modes.Get(ModesModeWraparound), "!t.modes.get(.wraparound)")
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "", out)
	}

	// Differences from those defaults are emitted, in the reverse sense of
	// the built-in defaults.
	term.modes.Set(ModesModeBracketedPaste, false)
	term.modes.Set(ModesModeWraparound, true)
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b[?7h\x1b[?2004l", out)
	}
}

// Zig: test "term51: TerminalFormatter.format: scrolling region extra"
func TestFm3_term51_TerminalFormatter_format_scrolling_region_extra(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 6)
	defer term.Deinit()

	f := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
	f.content = fm3ContentNone
	f.extra = FormatterTerminalFormatterExtraNone
	f.extra.scrolling_region = true

	// Default: full screen, nothing.
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "", out)
	}

	// Top and bottom.
	fm3Write(t, term, "\x1b[2;4r")
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b[2;4r", out)
	}
	// Only the top differs from the full screen.
	fm3Write(t, term, "\x1b[3r")
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b[3;6r", out)
	}
	// Only the bottom differs.
	fm3Write(t, term, "\x1b[1;5r")
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b[1;5r", out)
	}
	// Reset to the full screen.
	fm3Write(t, term, "\x1b[r")
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "", out)
	}

	// Left and right margins (mode 69 itself is not part of this extra).
	fm3Write(t, term, "\x1b[?69h\x1b[3;8s")
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b[3;8s", out)
	}
	// Only the left differs.
	fm3Write(t, term, "\x1b[4s")
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b[4;10s", out)
	}
	// Only the right differs.
	fm3Write(t, term, "\x1b[1;9s")
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b[1;9s", out)
	}
	// Both, DECSTBM first.
	fm3Write(t, term, "\x1b[2;5r\x1b[2;3s")
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b[2;5r\x1b[2;3s", out)
	}

	// Round trip (needs mode 69 enabled on the receiving side first).
	term2 := fm3Init(t, 10, 6)
	defer term2.Deinit()
	fm3Write(t, term2, "\x1b[?69h")
	out := fm3Out(t, f)
	// Zig-only: defer alloc.free(out);
	fm3Write(t, term2, out)
	expectEqual(t, term.scrolling_region.top, term2.scrolling_region.top)
	expectEqual(t, term.scrolling_region.bottom, term2.scrolling_region.bottom)
	expectEqual(t, term.scrolling_region.left, term2.scrolling_region.left)
	expectEqual(t, term.scrolling_region.right, term2.scrolling_region.right)

	// Plain and html do not emit it.
	for _, emit := range []FormatterFormat{FormatterFormatPlain, FormatterFormatHtml} {
		fm := FormatterTerminalFormatterInit(term, fm3Opts(emit))
		fm.content = fm3ContentNone
		fm.extra = FormatterTerminalFormatterExtraNone
		fm.extra.scrolling_region = true
		o := fm3Out(t, fm)
		// Zig-only: defer alloc.free(o);
		expectEqualStrings(t, "", o)
	}
}

// Zig: test "term51: TerminalFormatter.format: tabstops extra"
func TestFm3_term51_TerminalFormatter_format_tabstops_extra(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 20, 3)
	defer term.Deinit()

	f := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
	f.content = fm3ContentNone
	f.extra = FormatterTerminalFormatterExtraNone
	f.extra.tabstops = true

	// The default interval of 8 gives stops at 8 and 16 (0-indexed).
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b[3g\x1b[9G\x1bH\x1b[17G\x1bH", out)
	}

	// All cleared: only the clear sequence.
	fm3Write(t, term, "\x1b[3g")
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b[3g", out)
	}

	// First and last columns.
	fm3Write(t, term, "\x1b[1G\x1bH\x1b[20G\x1bH")
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b[3g\x1b[1G\x1bH\x1b[20G\x1bH", out)
	}

	// Round trip.
	term2 := fm3Init(t, 20, 3)
	defer term2.Deinit()
	out := fm3Out(t, f)
	// Zig-only: defer alloc.free(out);
	fm3Write(t, term2, out)
	for col := uint(0); col < 20; col++ {
		expectEqual(t, term.tabstops.Get(col), term2.tabstops.Get(col))
	}
	expect(t, term2.tabstops.Get(0), "t2.tabstops.get(0)")
	expect(t, term2.tabstops.Get(19), "t2.tabstops.get(19)")
	expect(t, !term2.tabstops.Get(8), "!t2.tabstops.get(8)")

	// Plain and html do not emit it.
	for _, emit := range []FormatterFormat{FormatterFormatPlain, FormatterFormatHtml} {
		fm := FormatterTerminalFormatterInit(term, fm3Opts(emit))
		fm.content = fm3ContentNone
		fm.extra = FormatterTerminalFormatterExtraNone
		fm.extra.tabstops = true
		o := fm3Out(t, fm)
		// Zig-only: defer alloc.free(o);
		expectEqualStrings(t, "", o)
	}
}

// Zig: test "term51: TerminalFormatter.format: keyboard and pwd extras"
func TestFm3_term51_TerminalFormatter_format_keyboard_and_pwd_extras(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 3)
	defer term.Deinit()

	fk := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
	fk.content = fm3ContentNone
	fk.extra = FormatterTerminalFormatterExtraNone
	fk.extra.keyboard = true
	fp := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
	fp.content = fm3ContentNone
	fp.extra = FormatterTerminalFormatterExtraNone
	fp.extra.pwd = true

	// Defaults: nothing.
	{
		out := fm3Out(t, fk)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "", out)
	}
	{
		out := fm3Out(t, fp)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "", out)
	}

	// Only modifyOtherKeys level 2 is represented.
	fm3Write(t, term, "\x1b[>4;1m")
	expect(t, !term.flags.modify_other_keys_2, "!t.flags.modify_other_keys_2")
	{
		out := fm3Out(t, fk)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "", out)
	}
	fm3Write(t, term, "\x1b[>4;2m")
	{
		out := fm3Out(t, fk)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b[>4;2m", out)
	}

	// The read-only stream ignores OSC 7, so set the pwd directly.
	expectEqual(t, 0, len(term.pwd))
	fm3Write(t, term, "\x1b]7;file://ignored/x\x1b\\")
	expectEqual(t, 0, len(term.pwd))
	zigTry(t, term.SetPwd([]uint8("file://host/home/user")))
	{
		out := fm3Out(t, fp)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b]7;file://host/home/user\x1b\\", out)
	}

	// Plain and html do not emit either.
	for _, emit := range []FormatterFormat{FormatterFormatPlain, FormatterFormatHtml} {
		fm := FormatterTerminalFormatterInit(term, fm3Opts(emit))
		fm.content = fm3ContentNone
		fm.extra = FormatterTerminalFormatterExtraNone
		fm.extra.keyboard = true
		fm.extra.pwd = true
		o := fm3Out(t, fm)
		// Zig-only: defer alloc.free(o);
		expectEqualStrings(t, "", o)
	}
}

// Zig: test "term51: TerminalFormatter.format: all extras in order with round trip"
func TestFm3_term51_TerminalFormatter_format_all_extras_in_order_with_round_trip(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 3)
	defer term.Deinit()
	fm3Write(t, term, "\x1b]4;3;rgb:01/02/03\x1b\\\x1b[?2004h\x1b[2;3r")
	fm3Write(t, term, "\x1b[Hhi")
	fm3Write(t, term, "\x1b[=3;1u\x1b[>4;2m\x1b(0\x1b[1\"q")
	zigTry(t, term.SetPwd([]uint8("file://h/p")))
	fm3Write(t, term, "\x1b]8;;http://e.com\x1b\\\x1b[1m")

	f := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
	f.extra = FormatterTerminalFormatterExtraAll
	out := fm3Out(t, f)
	// Zig-only: defer alloc.free(out);

	// 256 palette sequences first.
	expect(t, strings.HasPrefix(out, "\x1b]4;0;rgb:"), "std.mem.startsWith(u8, out, \"\\x1b]4;0;rgb:\")")
	idx := strings.Index(out, "\x1b]4;255;rgb:")
	expect(t, idx >= 0, "std.mem.indexOf(u8, out, \"\\x1b]4;255;rgb:\") != null")
	palette_end := idx + len("\x1b]4;255;rgb:00/00/00\x1b\\")
	expect(t, strings.Contains(out, "\x1b]4;3;rgb:01/02/03\x1b\\"),
		"std.mem.indexOf(u8, out, \"\\x1b]4;3;rgb:01/02/03\\x1b\\\\\") != null")
	tail := out[palette_end:]
	expectEqualStrings(t,
		"\x1b[?2004h"+ // modes
			"hi"+ // content
			"\x1b[0m\x1b[1m"+ // style
			"\x1b]8;;http://e.com\x1b\\"+ // hyperlink
			"\x1b[1\"q"+ // protection
			"\x1b[=3;1u"+ // kitty
			"\x1b(0"+ // charsets
			"\x1b[1;3H"+ // cursor
			"\x1b[2;3r"+ // scrolling region
			"\x1b[3g\x1b[9G\x1bH"+ // tabstops (10 cols: only column 9)
			"\x1b[>4;2m"+ // keyboard
			"\x1b]7;file://h/p\x1b\\", // pwd
		tail,
	)

	// Round trip into a fresh terminal.
	term2 := fm3Init(t, 10, 3)
	defer term2.Deinit()
	fm3Write(t, term2, out)
	expectEqual(t, term.colors.palette.current[3], term2.colors.palette.current[3])
	expect(t, term2.modes.Get(ModesModeBracketedPaste), "t2.modes.get(.bracketed_paste)")
	expectEqual(t, term.scrolling_region.top, term2.scrolling_region.top)
	expectEqual(t, term.scrolling_region.bottom, term2.scrolling_region.bottom)
	expectEqual(t, term.flags.modify_other_keys_2, term2.flags.modify_other_keys_2)
	// The read-only stream does not apply OSC 7, so the pwd is not restored.
	expectEqual(t, 0, len(term2.pwd))
	expectEqual(t,
		term.screens.active.kitty_keyboard.Current().Int(),
		term2.screens.active.kitty_keyboard.Current().Int(),
	)
	expect(t, term2.screens.active.cursor.protected, "t2.screens.active.cursor.protected")
	expect(t, term2.screens.active.cursor.style.flags.bold, "t2.screens.active.cursor.style.flags.bold")
	// The cursor is NOT restored: the cursor position is emitted by the
	// screen formatter, but the scrolling region (DECSTBM homes the
	// cursor) and the tabstops (the last HTS was set at column 9) are
	// emitted after it and move it.
	expectEqual(t, SizeCellCountInt(2), term.screens.active.cursor.x)
	expectEqual(t, SizeCellCountInt(8), term2.screens.active.cursor.x)
	expectEqual(t, SizeCellCountInt(0), term2.screens.active.cursor.y)
}

// Zig: test "term51: TerminalFormatter.format: trailing extras move the replayed cursor"
func TestFm3_term51_TerminalFormatter_format_trailing_extras_move_the_replayed_cursor(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 4)
	defer term.Deinit()
	fm3Write(t, term, "\x1b[2;3r\x1b[3;5H")
	expectEqual(t, SizeCellCountInt(4), term.screens.active.cursor.x)
	expectEqual(t, SizeCellCountInt(2), term.screens.active.cursor.y)

	// The cursor then the region: the region resets the cursor to home.
	{
		f := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
		f.content = fm3ContentNone
		f.extra = FormatterTerminalFormatterExtraNone
		f.extra.screen.cursor = true
		f.extra.scrolling_region = true
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b[3;5H\x1b[2;3r", out)

		term2 := fm3Init(t, 10, 4)
		defer term2.Deinit()
		fm3Write(t, term2, out)
		expectEqual(t, SizeCellCountInt(0), term2.screens.active.cursor.x)
		expectEqual(t, SizeCellCountInt(0), term2.screens.active.cursor.y)
	}

	// The cursor then the tabstops: the cursor ends on the last stop.
	{
		f := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
		f.content = fm3ContentNone
		f.extra = FormatterTerminalFormatterExtraNone
		f.extra.screen.cursor = true
		f.extra.tabstops = true
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b[3;5H\x1b[3g\x1b[9G\x1bH", out)

		term2 := fm3Init(t, 10, 4)
		defer term2.Deinit()
		fm3Write(t, term2, out)
		expectEqual(t, SizeCellCountInt(8), term2.screens.active.cursor.x)
		expectEqual(t, SizeCellCountInt(2), term2.screens.active.cursor.y)
	}
}

// Zig: test "term51: TerminalFormatter.format: pin map covers palette, modes and trailing extras"
func TestFm3_term51_TerminalFormatter_format_pin_map_covers_palette_modes_and_trailing_extras(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 3)
	defer term.Deinit()
	fm3Write(t, term, "\x1b[?2004hab\x1b[>4;2m")
	node := term.screens.active.pages.pages.first

	var pm []PageListPin
	// Zig-only: defer pm.deinit(alloc);
	f := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
	f.extra = FormatterTerminalFormatterExtraNone
	f.extra.palette = true
	f.extra.modes = true
	f.extra.keyboard = true
	f.pin_map = Some(FormatterPinMap{map_: &pm})
	out := fm3Out(t, f)
	// Zig-only: defer alloc.free(out);

	const palette_len = 256*19 + 10 + 180 + 468
	const modes = "\x1b[?2004h"
	expectEqualStrings(t, modes+"ab"+"\x1b[>4;2m", out[palette_len:])
	expectEqual(t, len(out), len(pm))

	top_left := PageListPin{node: node, x: 0, y: 0}
	// Palette and modes map to the top-left pin.
	for _, p := range pm[0 : palette_len+len(modes)] {
		expectEqual(t, top_left, p)
	}
	// Content.
	expectEqual(t, top_left, pm[palette_len+len(modes)])
	expectEqual(t,
		PageListPin{node: node, x: 1, y: 0},
		pm[palette_len+len(modes)+1],
	)
	// The trailing keyboard sequence maps to the last content pin.
	for _, p := range pm[palette_len+len(modes)+2:] {
		expectEqual(t, PageListPin{node: node, x: 1, y: 0}, p)
	}

	// With no content at all, the trailing extras map to the final pin of
	// the earlier extras (here, the top-left).
	var pm2 []PageListPin
	// Zig-only: defer pm2.deinit(alloc);
	f2 := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
	f2.content = fm3ContentNone
	f2.extra = FormatterTerminalFormatterExtraNone
	f2.extra.keyboard = true
	f2.extra.pwd = true
	f2.pin_map = Some(FormatterPinMap{map_: &pm2})
	out2 := fm3Out(t, f2)
	// Zig-only: defer alloc.free(out2);
	expectEqualStrings(t, "\x1b[>4;2m", out2)
	expectEqual(t, len(out2), len(pm2))
	for _, p := range pm2 {
		expectEqual(t, top_left, p)
	}

	// HTML with the palette extra and a pin map: same length as output.
	var pm3 []PageListPin
	// Zig-only: defer pm3.deinit(alloc);
	f3 := FormatterTerminalFormatterInit(term, FormatterOptionsHtml)
	f3.pin_map = Some(FormatterPinMap{map_: &pm3})
	out3 := fm3Out(t, f3)
	// Zig-only: defer alloc.free(out3);
	expect(t, strings.HasPrefix(out3, "<style>:root{"), "std.mem.startsWith(u8, out3, \"<style>:root{\")")
	expectEqual(t, len(out3), len(pm3))
}

// Zig: test "term51: TerminalFormatter.format: formats only the active screen"
func TestFm3_term51_TerminalFormatter_format_formats_only_the_active_screen(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 3)
	defer term.Deinit()
	fm3Write(t, term, "primary")

	{
		f := FormatterTerminalFormatterInit(term, FormatterOptionsPlain)
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "primary", out)
	}

	// Entering the alternate screen keeps the cursor column (7), so the
	// text is preceded by spaces (blank cells before text are emitted).
	fm3Write(t, term, "\x1b[?1049halt")
	{
		f := FormatterTerminalFormatterInit(term, FormatterOptionsPlain)
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "       alt", out)
	}
	{
		f := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
		f.extra = FormatterTerminalFormatterExtraNone
		f.extra.modes = true
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "\x1b[?1049h"+"       alt", out)
	}

	fm3Write(t, term, "\x1b[?1049l")
	{
		f := FormatterTerminalFormatterInit(term, FormatterOptionsPlain)
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "primary", out)
	}
}

// Zig: test "term51: TerminalFormatter.format: selection content and styles extra for html"
func TestFm3_term51_TerminalFormatter_format_selection_content_and_styles_extra_for_html(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 3)
	defer term.Deinit()
	fm3Write(t, term, "\x1b[1mbold\x1b[0m plain\r\nsecond")

	// A selection restricted to the second row.
	f := FormatterTerminalFormatterInit(term, FormatterOptionsHtml)
	f.extra = FormatterTerminalFormatterExtraNone
	f.content = fm3ContentSelection(SelectionInit(fm3Pin(t, term, 0, 1), fm3Pin(t, term, 5, 1), false))
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, fm3HtmlOpen+"second</div>", out)
	}

	// Background and foreground options apply to the page wrapper.
	f.opts.background = Some(ColorRGB{r: 0, g: 0, b: 0})
	f.opts.foreground = Some(ColorRGB{r: 255, g: 255, b: 255})
	f.content = fm3ContentSelection(SelectionInit(fm3Pin(t, term, 0, 0), fm3Pin(t, term, 3, 0), false))
	{
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t,
			"<div style=\"font-family: monospace; white-space: pre;"+
				"background-color: #000000;color: #ffffff;\">"+
				"<div style=\"display: inline;font-weight: bold;\">bold</div></div>",
			out,
		)
	}
}

// Zig: test "term51: PageFormatter.formatWithState: point map of background cells and anchors"
func TestFm3_term51_PageFormatter_formatWithState_point_map_of_background_cells_and_anchors(t *testing.T) {
	testStdCapacity(t)

	// Background cells: one byte each, the style bytes map to the first
	// styled cell and the closing reset to the previous byte.
	{
		term := fm3Init(t, 6, 2)
		defer term.Deinit()
		fm3Write(t, term, "CPU:\x1b[41m\x1b[K\x1b[0m")
		var pm []PointCoordinate
		// Zig-only: defer pm.deinit(alloc);
		f := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsVt)
		f.point_map = Some(FormatterPageFormatterPointMap{map_: &pm})
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, "CPU:\x1b[0m\x1b[48;5;1m  \x1b[0m", r.out)
		expectEqual(t, len(r.out), len(pm))
		for i := 0; i < 4; i++ {
			expectEqual(t, fm3Coord(SizeCellCountInt(i), 0), pm[i])
		}
		// The style is 4 + 9 bytes, then the two spaces, then the 4 byte
		// reset which repeats the last entry.
		for _, c := range pm[4:18] {
			expectEqual(t, fm3Coord(4, 0), c)
		}
		expectEqual(t, fm3Coord(5, 0), pm[18])
		for _, c := range pm[19:] {
			expectEqual(t, fm3Coord(5, 0), c)
		}
	}

	// HTML anchor opening bytes map to the first linked cell.
	{
		term := fm3Init(t, 6, 2)
		defer term.Deinit()
		fm3Write(t, term, "\x1b[2C\x1b]8;;u\x1b\\a\x1b]8;;\x1b\\")
		var pm []PointCoordinate
		// Zig-only: defer pm.deinit(alloc);
		f := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsHtml)
		f.point_map = Some(FormatterPageFormatterPointMap{map_: &pm})
		r := fm3Run(t, f)
		// Zig-only: defer r.deinit(alloc);
		expectEqualStrings(t, fm3HtmlOpen+"  <a href=\"u\">a</a></div>", r.out)
		expectEqual(t, len(r.out), len(pm))
		h := len(fm3HtmlOpen)
		for _, c := range pm[0:h] {
			expectEqual(t, fm3Coord(0, 0), c)
		}
		// The two blank cells in front of the anchor.
		expectEqual(t, fm3Coord(1, 0), pm[h])
		expectEqual(t, fm3Coord(0, 0), pm[h+1])
		// `<a href="u">` (12 bytes), then the cell, then `</a>` and `</div>`.
		for _, c := range pm[h+2:] {
			expectEqual(t, fm3Coord(2, 0), c)
		}
	}
}

// Zig: test "term51: TerminalFormatter.format: allocation failure of the pin map and small writers"
func TestFm3_term51_TerminalFormatter_format_allocation_failure_of_the_pin_map_and_small_writers(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 6, 2)
	defer term.Deinit()
	fm3Write(t, term, "ab")

	// Zig-only: the fail_index loop with std.testing.FailingAllocator (an
	// allocation failure of the pin map; not translatable, see
	// scripts/not-translated.txt). The remaining half is translated.

	// A writer that is too small fails at the palette, at the content and at
	// the trailing extras: every smaller capacity fails, the exact one works.
	f := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
	f.extra = FormatterTerminalFormatterExtraAll
	out := fm3Out(t, f)
	// Zig-only: defer alloc.free(out);
	expect(t, len(out) > 5522, "out.len > 5522")
	buf := make([]uint8, len(out))
	// Zig-only: defer alloc.free(buf);
	for cap_ := 0; cap_ < len(out); cap_++ {
		w := oscWriterFixed(buf[0:cap_])
		fm3ExpectWriteFailed(t, f.Format(&w))
	}
	w := oscWriterFixed(buf)
	zigTry(t, f.Format(&w))
	expectEqualStrings(t, out, string(w.buffered()))
}

// Zig: test "term51: ScreenFormatter.format: small writers fail in content and extras"
func TestFm3_term51_ScreenFormatter_format_small_writers_fail_in_content_and_extras(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 6, 2)
	defer term.Deinit()
	fm3Write(t, term, "abcdef")

	f := FormatterScreenFormatterInit(term.screens.active, FormatterOptionsVt)
	f.extra = FormatterScreenFormatterExtraAll
	out := fm3Out(t, f)
	// Zig-only: defer alloc.free(out);

	// Every shorter limit fails and exact size succeeds.
	for cap_ := 0; cap_ < len(out); cap_++ {
		buf := make([]uint8, cap_)
		// Zig-only: defer alloc.free(buf);
		w := oscWriterFixed(buf)
		fm3ExpectWriteFailed(t, f.Format(&w))
	}
	buf := make([]uint8, len(out))
	// Zig-only: defer alloc.free(buf);
	w := oscWriterFixed(buf)
	zigTry(t, f.Format(&w))
	expectEqualStrings(t, out, string(w.buffered()))
}

// Zig: test "term51: TerminalFormatter.format: vt output replays into an equivalent terminal"
func TestFm3_term51_TerminalFormatter_format_vt_output_replays_into_an_equivalent_terminal(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 20, 6)
	defer term.Deinit()
	fm3Write(t, term, "\x1b[1mBold\x1b[0m \x1b[3;38;5;9mItal\x1b[0m\r\n"+
		"\x1b[4:3;58;2;1;2;3;48;2;9;8;7mcurly\x1b[0m   x\r\n"+
		"\r\n"+
		"a\xe4\xb8\xadb e\xcc\x81  \x1b[41mred\x1b[K\x1b[0m\r\n"+
		"\x1b[2;7;9;53mmany\x1b[0m end")

	f := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
	f.extra = FormatterTerminalFormatterExtraAll
	vt := fm3Out(t, f)
	// Zig-only: defer alloc.free(vt);

	term2 := fm3Init(t, 20, 6)
	defer term2.Deinit()
	fm3Write(t, term2, vt)

	// Same visible text.
	plain := FormatterTerminalFormatterInit(term, FormatterOptionsPlain)
	plain.extra = FormatterTerminalFormatterExtraNone
	p1 := fm3Out(t, plain)
	// Zig-only: defer alloc.free(p1);
	plain2 := FormatterTerminalFormatterInit(term2, FormatterOptionsPlain)
	plain2.extra = FormatterTerminalFormatterExtraNone
	p2 := fm3Out(t, plain2)
	// Zig-only: defer alloc.free(p2);
	expectEqualStrings(t, p1, p2)
	expectEqualStrings(t,
		"Bold Ital\ncurly   x\n\na\xe4\xb8\xadb e\xcc\x81  red\nmany end",
		p1,
	)

	// Same style on every cell that has text, and the same wide flags.
	for y := SizeCellCountInt(0); y < 6; y++ {
		for x := SizeCellCountInt(0); x < 20; x++ {
			c1 := fm3Cell(term, x, y)
			c2 := fm3Cell(term2, x, y)
			expectEqual(t, c1.wide, c2.wide)
			if c1.HasText() {
				expectEqual(t, c1.Codepoint(), c2.Codepoint())
				expect(t, fm3CellStyle(term, x, y).Eql(fm3CellStyle(term2, x, y)),
					"term51ZCellStyle(&t, x, y).eql(term51ZCellStyle(&t2, x, y))")
			}
		}
	}
	// The red fill cells came back as background cells.
	red := Style{bg_color: StyleColor{Tag: StyleColorPalette, Palette: 1}}
	expect(t, fm3CellStyle(term2, 12, 3).Eql(red), "term51ZCellStyle(&t2, 12, 3).eql(.{ .bg_color = .{ .palette = 1 } })")
	expect(t, fm3CellStyle(term2, 19, 3).Eql(red), "term51ZCellStyle(&t2, 19, 3).eql(.{ .bg_color = .{ .palette = 1 } })")

	// And formatting the replayed terminal gives the same VT again for
	// everything that does not depend on the replayed cursor (the cursor
	// ends up elsewhere, see the trailing extras test).
	g1 := FormatterTerminalFormatterInit(term, FormatterOptionsVt)
	vt1 := fm3Out(t, g1)
	// Zig-only: defer alloc.free(vt1);
	g2 := FormatterTerminalFormatterInit(term2, FormatterOptionsVt)
	vt2 := fm3Out(t, g2)
	// Zig-only: defer alloc.free(vt2);
	expectEqualStrings(t, vt1, vt2)
}

// Zig: test "term51: PageListFormatter.format: vt of a wrapped page replays with the same wrapping"
func TestFm3_term51_PageListFormatter_format_vt_of_a_wrapped_page_replays_with_the_same_wrapping(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 5, 4)
	defer term.Deinit()
	fm3Write(t, term, "hello world\r\nab")

	// Without unwrap each row is written with \r\n, so the replay has hard
	// line breaks. With unwrap the soft wraps are rejoined and the replay
	// wraps again by itself.
	{
		f := FormatterPageListFormatterInit(&term.screens.active.pages, FormatterOptionsVt)
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "hello\r\n worl\r\nd\r\nab", out)
	}
	{
		opts := fm3Opts(FormatterFormatVt)
		opts.unwrap = true
		f := FormatterPageListFormatterInit(&term.screens.active.pages, opts)
		out := fm3Out(t, f)
		// Zig-only: defer alloc.free(out);
		expectEqualStrings(t, "hello world\r\nab", out)

		term2 := fm3Init(t, 5, 4)
		defer term2.Deinit()
		fm3Write(t, term2, out)
		expect(t, fm3Page(term2).GetRow(0).wrap, "term51ZPage(&t2).getRow(0).wrap")
		expect(t, fm3Page(term2).GetRow(1).wrap, "term51ZPage(&t2).getRow(1).wrap")
		expect(t, fm3Page(term2).GetRow(1).wrap_continuation, "term51ZPage(&t2).getRow(1).wrap_continuation")
		expect(t, !fm3Page(term2).GetRow(2).wrap, "!term51ZPage(&t2).getRow(2).wrap")
	}
}

// Go-only: panic test: PageFormatter.formatHyperlinkClose: html with a point map that is empty (`assert(m.map.items.len > 0)`)
func TestFm3_panic_PageFormatter_formatHyperlinkClose_html_empty_point_map(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 5, 2)
	defer term.Deinit()
	b := oscWriterAllocatingInit()
	defer b.deinit()

	var pm []PointCoordinate
	f := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsHtml)
	f.point_map = Some(FormatterPageFormatterPointMap{map_: &pm})
	fm3ExpectPanic(t, func() { _ = f.formatHyperlinkClose(&b.writer) })
}

// Go-only: panic test: ScreenFormatter.format: charsets extra with GR set to G0 (`.G0 => unreachable, // GR can't be G0`)
func TestFm3_panic_ScreenFormatter_format_charsets_extra_gr_is_G0(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 5)
	defer term.Deinit()
	b := oscWriterAllocatingInit()
	defer b.deinit()

	f := FormatterScreenFormatterInit(term.screens.active, FormatterOptionsVt)
	f.content = fm3ContentNone
	f.extra.charsets = true
	term.screens.active.charset.gr = CharsetsSlotsG0
	fm3ExpectPanic(t, func() { _ = f.Format(&b.writer) })
}

// Go-only: panic test: PageFormatter.formatWithState: a codepoint_grapheme cell with no grapheme data (`self.page.lookupGrapheme(cell).?`)
func TestFm3_panic_PageFormatter_formatWithState_grapheme_cell_without_grapheme_data(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 2)
	defer term.Deinit()
	fm3Write(t, term, "x")
	fm3Cell(term, 0, 0).content_tag = PageCellContentTagCodepointGrapheme
	b := oscWriterAllocatingInit()
	defer b.deinit()

	f := FormatterPageFormatterInit(fm3Page(term), FormatterOptionsPlain)
	fm3ExpectPanic(t, func() { _, _ = f.FormatWithState(&b.writer) })
}

// Go-only: panic test: PageFormatter.formatWithState: codepoint map string replacement that is not valid UTF-8 (`std.unicode.Utf8View.init(s) catch unreachable`)
func TestFm3_panic_PageFormatter_formatWithState_replacement_string_not_valid_utf8(t *testing.T) {
	testStdCapacity(t)
	term := fm3Init(t, 10, 2)
	defer term.Deinit()
	fm3Write(t, term, "a")
	b := oscWriterAllocatingInit()
	defer b.deinit()

	opts := fm3Opts(FormatterFormatPlain)
	opts.codepoint_map = Some([]FormatterCodepointMap{
		{range_: [2]uint32{'a', 'a'}, replacement: FormatterCodepointMapReplacement{
			Tag: FormatterCodepointMapReplacementString, String: []uint8("\xff")}},
	})
	f := FormatterPageFormatterInit(fm3Page(term), opts)
	fm3ExpectPanic(t, func() { _, _ = f.FormatWithState(&b.writer) })
}
