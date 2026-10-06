// Go-only tests for the project-lead ruling on the exactRowCapacity abort
// (no Zig test exists for it). Names are TestScreenSplitRuling_..., helpers
// are prefixed ssr.
//
// Upstream: when a style set of the maximum capacity (65535, 53,247 usable
// ids) is full, manualStyleUpdate's increaseCapacity returns OutOfSpace and
// splitForCapacity runs (Screen.zig:2043). If one of its two row ranges
// (rows 0..cursor, or cursor..end) references all 53,247 ids,
// Page.exactRowCapacity's `.styles = @intCast(styles_cap)` (page.zig:737)
// traps in a safe build: capacityForCount(53247) is 65536. Ordinary input
// reaches it: 53,247 cells with distinct SGR colours on a terminal wide
// enough for a non-standard page (46,708 columns, one more than the
// production standard capacity allows). term51 docs/upstream-crashes.md,
// upstream-findings.md section 15, case text/style_split
// full_set_rows_above_sgr.
//
// The port returns ErrOutOfSpace from splitForCapacity before anything
// changes (the page cannot be split), so manualStyleUpdate fails through its
// existing error path: Screen.setAttribute restores the previous style and
// returns the error, the stream handler ignores it, and printing continues
// with the previous style.
//
// The tests use the production page capacity (grapheme_bytes 8192, the
// default when testStdCapacity is not called), as the corpus does.

package terminal

import (
	"fmt"
	"testing"
)

const (
	// One more column than the production standard capacity fits in one row
	// (term51 data/layout/page_layout.txt, std_capacity_production
	// maxCols=46707): the page is non-standard, 215 rows of capacity.
	ssrCols = 46708
	// Every usable id of a style set of capacity 65535.
	ssrStyles = 53247
)

// ssrCell is the SGR of the i-th distinct style and the cell written with it:
// fg palette i%256, bg palette i/256 (the same bytes as the corpus case).
func ssrCell(i int) []byte {
	s := ""
	if i%256 == 0 {
		s = fmt.Sprintf("\x1b[48;5;%dm", i/256)
	}
	return []byte(s + fmt.Sprintf("\x1b[38;5;%dmx", i%256))
}

func ssrFeed(t *testing.T, stream *StreamReadonlyStream, b []byte) {
	t.Helper()
	if err := stream.NextSlice(b); err != nil {
		t.Fatalf("NextSlice: %v", err)
	}
}

func ssrPaletteStyle(fg, bg uint8) Style {
	return Style{
		fg_color: StyleColor{Tag: StyleColorPalette, Palette: fg},
		bg_color: StyleColor{Tag: StyleColorPalette, Palette: bg},
	}
}

func ssrCheckIntegrity(t *testing.T, term *Terminal) {
	t.Helper()
	for node := term.screens.active.pages.pages.first; node != nil; node = node.next {
		if err := node.data.VerifyIntegrity(); err != nil {
			t.Fatalf("page integrity: %v", err)
		}
	}
}

// Go-only: ruling (Screen.splitForCapacity returns ErrOutOfSpace where upstream aborts in Page.exactRowCapacity's style-capacity cast; term51 docs/upstream-crashes.md)
func TestScreenSplitRuling_full_style_set_in_the_rows_above_fails_the_style_change(t *testing.T) {
	opts := terminalOptionsDefault(ssrCols, 2)
	opts.max_scrollback = 0
	term, err := TerminalInit(opts)
	zigTry(t, err)
	defer term.Deinit()
	stream := term.VtStream()
	defer stream.Deinit()

	// 46,708 cells on row 0 and 6,539 on row 1, every one with its own style.
	var fill []byte
	for i := range ssrStyles {
		fill = append(fill, ssrCell(i)...)
	}
	ssrFeed(t, &stream, fill)

	s := term.screens.active
	node := s.pages.pages.first
	expect(t, node.next == nil, "one page")
	expect(t, node.data.capacity.styles == 65535, "style capacity at its maximum")
	expect(t, node.data.styles.Count() == ssrStyles, "every usable style id in use")
	expect(t, s.cursor.x == ssrStyles-ssrCols && s.cursor.y == 1, "cursor after the last cell")
	old_style := ssrPaletteStyle(254, 207)
	expect(t, s.cursor.style == old_style, "cursor has the last cell's style")
	old_id := s.cursor.style_id

	// A new style: the set is full, the page is at maximum capacity, and the
	// rows 0..1 above the cursor reference all 53,247 ids. Upstream traps
	// here; the port fails the style change.
	err = s.SetAttribute(SGRAttribute{Tag: SGRAttributeTag256Bg, X256Bg: 208})
	expect(t, err == ErrOutOfSpace, "setAttribute returns error.OutOfSpace")
	expect(t, s.cursor.style == old_style, "the previous style is restored")
	expect(t, s.cursor.style_id == old_id, "the previous style id is restored")
	expect(t, s.pages.pages.first == node && node.next == nil, "the page is not split")
	expect(t, node.data.size.rows == 2, "the page keeps both rows")
	expect(t, node.data.styles.Count() == ssrStyles, "no style added")
	ssrCheckIntegrity(t, &term)

	// Through the stream (the handler ignores the error) and on to printing:
	// the SGR has no effect and the text gets the previous style.
	ssrFeed(t, &stream, []byte("\x1b[48;5;208mAB"))
	expect(t, s.cursor.style == old_style, "the SGR had no effect")
	expect(t, s.cursor.x == ssrStyles-ssrCols+2, "two cells printed")
	for i, want := range "AB" {
		x := ssrStyles - ssrCols + i
		cell := node.data.GetRowAndCell(uint(x), 1).cell
		expect(t, cell.content.codepoint == uint32(want), fmt.Sprintf("cell %d holds %c", x, want))
		expect(t, cell.style_id == old_id, fmt.Sprintf("cell %d has the previous style", x))
	}
	ssrCheckIntegrity(t, &term)
}

// Go-only: ruling (as above; the range below the cursor, rows cursor..end, references every id)
func TestScreenSplitRuling_full_style_set_in_the_rows_below_fails_the_style_change(t *testing.T) {
	opts := terminalOptionsDefault(ssrCols, 2)
	opts.max_scrollback = 0
	term, err := TerminalInit(opts)
	zigTry(t, err)
	defer term.Deinit()
	stream := term.VtStream()
	defer stream.Deinit()

	// Row 0 is full and row 1 holds the rest; the cursor goes back to row 0,
	// so bytes_above (row 0) fits and bytes_below (rows 0..1) does not.
	var fill []byte
	for i := range ssrStyles {
		fill = append(fill, ssrCell(i)...)
	}
	fill = append(fill, "\x1b[1;1H"...)
	ssrFeed(t, &stream, fill)

	s := term.screens.active
	node := s.pages.pages.first
	expect(t, s.cursor.x == 0 && s.cursor.y == 0, "cursor at the top left")
	old_style := s.cursor.style
	old_id := s.cursor.style_id

	err = s.SetAttribute(SGRAttribute{Tag: SGRAttributeTag256Bg, X256Bg: 208})
	expect(t, err == ErrOutOfSpace, "setAttribute returns error.OutOfSpace")
	expect(t, s.cursor.style == old_style && s.cursor.style_id == old_id, "the previous style is restored")
	expect(t, s.pages.pages.first == node && node.next == nil, "the page is not split")
	ssrCheckIntegrity(t, &term)
}
