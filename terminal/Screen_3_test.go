// Tests translated one for one from src/terminal/Screen.zig (Ghostty v1.3.1
// plus the term51 additional tests), third part: the tests from "term51:
// cursorScrollAbove: cursor on the previous page moves to the next page" up
// to and including "term51: select: null with no selection does not set
// dirty" (Zig lines 13170 to 18245). Each Go test carries the exact Zig title
// in a "// Zig: test" comment. Helpers are prefixed sc3.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualSlices / expectEqualStrings keep Zig's names
//     and Zig's (expected, actual) argument order. `try` on a call that
//     returns an error is zigTry(t, err); `x.?` is zigUnwrap.
//   - `init(alloc, .{ ... })` is sc3Init, which calls ScreenInit with the Zig
//     defaults written out (kitty_image_storage_limit = 320 * 1000 * 1000)
//     and returns the screen by pointer (the methods take a pointer
//     receiver). The helpers of the Zig file (term51O*, term51Q*, term51R*)
//     are translated as sc3O*, sc3Q* and sc3R*.
//   - Every test starts with testStdCapacity(t): Zig's unit tests run with a
//     standard page capacity of grapheme_bytes 512.
//   - A Zig block `{ var s = ...; defer s.deinit(); ... }` (or a loop body
//     with a defer) is a Go function literal called at once, so the deferred
//     Deinit runs at the end of the block as in Zig.
//   - `const alloc = std.testing.allocator;` is dropped (a `// Zig-only:`
//     comment); allocator arguments are dropped.
//   - build_options.kitty_graphics is false in the library build, so the
//     helper term51QClearKittyDirty does nothing and term51QKittyDirty
//     returns true, exactly as in Zig.
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt.

package terminal

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers (prefix sc3).
// ---------------------------------------------------------------------------

// sc3ExpectPanic runs f and fails the test if f does not panic.
func sc3ExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if recover() == nil {
			t.Fatalf("expected a panic, got none")
		}
	}()
	f()
}

// sc3Init is `try init(alloc, .{ .cols = cols, .rows = rows, .max_scrollback =
// max_scrollback })`: ScreenInit with the Zig defaults of Options, the
// screen returned by pointer.
func sc3Init(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt, max_scrollback uint64) *Screen {
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

// sc3Active, sc3Screen, sc3History and sc3Viewport are the Zig literals
// `.{ .active = .{ .x = x, .y = y } }` and so on (`.{ .active = .{} }` is
// sc3Active(0, 0)).
func sc3Active(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagActive, Active: PointCoordinate{x: x, y: y}}
}

func sc3Screen(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagScreen, Screen: PointCoordinate{x: x, y: y}}
}

func sc3History(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagHistory, History: PointCoordinate{x: x, y: y}}
}

func sc3Viewport(x SizeCellCountInt, y uint32) Point {
	return Point{Tag: PointTagViewport, Viewport: PointCoordinate{x: x, y: y}}
}

// sc3A is the Zig literal of an sgr.Attribute without a payload (`.bold`,
// `.unset`, `.reset_bg`, ...).
func sc3A(tag SGRAttributeTag) SGRAttribute {
	return SGRAttribute{Tag: tag}
}

// sc3AttrBgRGB is `.{ .direct_color_bg = .{ .r = r, .g = g, .b = b } }`.
func sc3AttrBgRGB(r uint8, g uint8, b uint8) SGRAttribute {
	return SGRAttribute{Tag: SGRAttributeTagDirectColorBg, DirectColorBg: ColorRGB{r: r, g: g, b: b}}
}

// sc3AttrFgRGB is `.{ .direct_color_fg = .{ .r = r, .g = g, .b = b } }`.
func sc3AttrFgRGB(r uint8, g uint8, b uint8) SGRAttribute {
	return SGRAttribute{Tag: SGRAttributeTagDirectColorFg, DirectColorFg: ColorRGB{r: r, g: g, b: b}}
}

// sc3AttrUnderline is `.{ .underline = u }`.
func sc3AttrUnderline(u SGRAttributeUnderline) SGRAttribute {
	return SGRAttribute{Tag: SGRAttributeTagUnderline, Underline: u}
}

// sc3AttrBg256 is `.{ .@"256_bg" = idx }`.
func sc3AttrBg256(idx uint8) SGRAttribute {
	return SGRAttribute{Tag: SGRAttributeTag256Bg, X256Bg: idx}
}

// sc3AttrFg256 is `.{ .@"256_fg" = idx }`.
func sc3AttrFg256(idx uint8) SGRAttribute {
	return SGRAttribute{Tag: SGRAttributeTag256Fg, X256Fg: idx}
}

// sc3SetAttr is `try s.setAttribute(attr)`.
func sc3SetAttr(t *testing.T, s *Screen, attr SGRAttribute) {
	t.Helper()
	zigTry(t, s.SetAttribute(attr))
}

// sc3Write is `try s.testWriteString(text)`.
func sc3Write(t *testing.T, s *Screen, text string) {
	t.Helper()
	zigTry(t, s.TestWriteString([]uint8(text)))
}

// sc3StartLink is `try s.startHyperlink(uri, id)` with an id.
func sc3StartLink(t *testing.T, s *Screen, uri string, id string) {
	t.Helper()
	zigTry(t, s.StartHyperlink([]uint8(uri), Some([]uint8(id))))
}

// sc3StartLinkImplicit is `try s.startHyperlink(uri, null)`.
func sc3StartLinkImplicit(t *testing.T, s *Screen, uri string) {
	t.Helper()
	zigTry(t, s.StartHyperlink([]uint8(uri), None[[]uint8]()))
}

// sc3Wide is the Zig literal of a cell with a codepoint of 0 and the given
// wide state, `.{ .content_tag = .codepoint, .content = .{ .codepoint = cp },
// .wide = wide }`.
func sc3CellWide(cp uint32, wide PageCellWide) PageCell {
	return PageCell{
		content_tag: PageCellContentTagCodepoint,
		content:     PageCellContent{codepoint: cp},
		wide:        wide,
	}
}

// ---- term51O helpers ------------------------------------------------------

// sc3OPin is term51OPin: an (untracked) pin at the active point (x, y).
func sc3OPin(t *testing.T, s *Screen, x int, y int) PageListPin {
	t.Helper()
	return zigUnwrap(t, s.pages.Pin(sc3Active(SizeCellCountInt(x), uint32(y))))
}

// sc3OCell is term51OCell: the cell at the active point (x, y).
func sc3OCell(t *testing.T, s *Screen, x int, y int) *PageCell {
	t.Helper()
	return sc3OPin(t, s, x, y).RowAndCell().cell
}

// sc3ORow is term51ORow: the row at the active row y.
func sc3ORow(t *testing.T, s *Screen, y int) *PageRow {
	t.Helper()
	return sc3OPin(t, s, 0, y).RowAndCell().row
}

// sc3OSetCp is term51OSetCp: write a plain codepoint straight into the cell
// at active (x, y).
func sc3OSetCp(t *testing.T, s *Screen, x int, y int, cp uint32) {
	t.Helper()
	*sc3OCell(t, s, x, y) = PageCell{
		content_tag: PageCellContentTagCodepoint,
		content:     PageCellContent{codepoint: cp},
	}
}

// sc3OCp is term51OCp: the codepoint of the cell at active (x, y).
func sc3OCp(t *testing.T, s *Screen, x int, y int) uint32 {
	t.Helper()
	return sc3OCell(t, s, x, y).Codepoint()
}

// sc3OStyleRefs is term51OStyleRefs: the reference count of a style in a
// page.
func sc3OStyleRefs(page *Page, id StyleId) uint {
	return uint(page.styles.RefCount(id))
}

// sc3OLinkRefs is term51OLinkRefs: the reference count of a hyperlink in a
// page.
func sc3OLinkRefs(page *Page, id HyperlinkId) uint {
	return uint(page.hyperlink_set.RefCount(id))
}

// sc3ONodeCount is term51ONodeCount: the number of pages in the screen.
func sc3ONodeCount(s *Screen) uint {
	var n uint
	for it := s.pages.pages.first; it != nil; it = it.next {
		n++
	}
	return n
}

// sc3ODirtyRows is term51ODirtyRows: the row-level dirty flag of the first n
// active rows.
func sc3ODirtyRows(t *testing.T, s *Screen, n int) []bool {
	t.Helper()
	result := make([]bool, n)
	for y := 0; y < n; y++ {
		result[y] = sc3ORow(t, s, y).dirty
	}
	return result
}

// sc3OExpectPin is term51OExpectPin: expect that the pin sits at active
// (x, y).
func sc3OExpectPin(t *testing.T, s *Screen, pin PageListPin, x uint, y uint) {
	t.Helper()
	pt := zigUnwrap(t, s.pages.PointFromPin(PointTagActive, pin))
	expectEqual(t, x, uint(pt.Active.x))
	expectEqual(t, y, uint(pt.Active.y))
}

// sc3OExpectCursorConsistent is term51OExpectCursorConsistent: expect that
// the cursor cache (x, y, row, cell) agrees with the pin.
func sc3OExpectCursorConsistent(t *testing.T, s *Screen) {
	t.Helper()
	rac := s.cursor.page_pin.RowAndCell()
	expect(t, s.cursor.page_row == rac.row, "s.cursor.page_row == rac.row")
	expect(t, s.cursor.page_cell == rac.cell, "s.cursor.page_cell == rac.cell")
	sc3OExpectPin(t, s, *s.cursor.page_pin, uint(s.cursor.x), uint(s.cursor.y))
	expectEqual(t, s.cursor.x, s.cursor.page_pin.x)
}

// sc3OTwoPages is term51OTwoPages: a screen whose cursor is on the bottom row
// of the active area, which is the only row used on the second page; every
// other active row is one of the last rows of the first page. The first page
// is full.
func sc3OTwoPages(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) *Screen {
	t.Helper()
	s := sc3Init(t, cols, rows, 10)
	first := &s.pages.pages.first.data
	first.PauseIntegrityChecks(true)
	for i := uint(0); i < uint(first.capacity.rows); i++ {
		zigTry(t, s.cursorDownOrScroll())
	}
	first.PauseIntegrityChecks(false)
	expectEqual(t, 2, sc3ONodeCount(s))
	expect(t, s.cursor.page_pin.node == s.pages.pages.last, "s.cursor.page_pin.node == s.pages.pages.last.?")
	expectEqual(t, 1, s.pages.pages.last.data.size.rows)
	expectEqual(t, first.capacity.rows, first.size.rows)
	expectEqual(t, rows-1, s.cursor.y)
	return s
}

// sc3OMultiPage is term51OMultiPage: a screen with a very wide active area
// so that the active area itself spans at least three pages (no scrolling
// needed).
func sc3OMultiPage(t *testing.T, max_scrollback uint64) *Screen {
	t.Helper()
	s := sc3Init(t, 2000, 60, max_scrollback)
	expect(t, sc3ONodeCount(s) >= 3, "term51ONodeCount(&s) >= 3")
	return s
}

// sc3OScreenCell is term51OScreenCell: the cell at the screen point (x, y).
func sc3OScreenCell(t *testing.T, s *Screen, x int, y int) *PageCell {
	t.Helper()
	return zigUnwrap(t, s.pages.Pin(sc3Screen(SizeCellCountInt(x), uint32(y)))).RowAndCell().cell
}

// sc3OFullPage is term51OFullPage: a screen whose single page is full (all of
// its row capacity is in use) and whose cursor is on the bottom row.
func sc3OFullPage(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) *Screen {
	t.Helper()
	s := sc3Init(t, cols, rows, 10)
	page := &s.pages.pages.first.data
	page.PauseIntegrityChecks(true)
	for page.size.rows < page.capacity.rows {
		zigTry(t, s.cursorDownOrScroll())
	}
	page.PauseIntegrityChecks(false)
	expectEqual(t, 1, sc3ONodeCount(s))
	expectEqual(t, page.capacity.rows, page.size.rows)
	expectEqual(t, rows-1, s.cursor.y)
	return s
}

// sc3OExpectBlank is term51OExpectBlank: expect that the cell is entirely
// blank.
func sc3OExpectBlank(t *testing.T, cell *PageCell) {
	t.Helper()
	expect(t, cell.IsZero(), "cell.isZero()")
}

// sc3OExpectBgRow is term51OExpectBgRow: expect that the whole row is the
// background colour cell for rgb.
func sc3OExpectBgRow(t *testing.T, s *Screen, y int, rgb [3]uint8) {
	t.Helper()
	for x := 0; x < int(s.pages.cols); x++ {
		cell := sc3OCell(t, s, x, y)
		expectEqual(t, PageCellContentTagBgColorRgb, cell.content_tag)
		expectEqual(t, rgb[0], cell.content.color_rgb.r)
		expectEqual(t, rgb[1], cell.content.color_rgb.g)
		expectEqual(t, rgb[2], cell.content.color_rgb.b)
	}
}

// sc3OExpectMovedFrom is term51OExpectMovedFrom: expect that the cursor's
// style and hyperlink were moved off old_node (no references left there) and
// are accounted for exactly once on the page the cursor is on now. The
// cursor must have a style and an explicit-id link created by
// sc3OStyleAndLink.
func sc3OExpectMovedFrom(t *testing.T, s *Screen, old_node *PageListNode, old_style_id StyleId, old_link_id HyperlinkId) {
	t.Helper()
	expect(t, s.cursor.page_pin.node != old_node, "s.cursor.page_pin.node != old_node")
	expectEqual(t, 0, sc3OStyleRefs(&old_node.data, old_style_id))
	expectEqual(t, 0, sc3OLinkRefs(&old_node.data, old_link_id))

	page := &s.cursor.page_pin.node.data
	expect(t, s.cursor.style.flags.bold, "s.cursor.style.flags.bold")
	expect(t, s.cursor.style_id != StyleDefaultId, "s.cursor.style_id != style.default_id")
	expect(t, page.styles.Get(s.cursor.style_id).flags.bold, "page.styles.get(page.memory, s.cursor.style_id).flags.bold")
	expectEqual(t, 1, sc3OStyleRefs(page, s.cursor.style_id))
	expectEqual(t, 1, page.styles.Count())

	expect(t, s.cursor.hyperlink != nil, "s.cursor.hyperlink != null")
	expect(t, s.cursor.hyperlink_id != 0, "s.cursor.hyperlink_id != 0")
	expectEqual(t, 1, sc3OLinkRefs(page, s.cursor.hyperlink_id))
	expectEqual(t, 1, page.hyperlink_set.Count())
	link := s.cursor.hyperlink
	expectEqualStrings(t, "https://example.com/", string(link.uri))
	expectEqualStrings(t, "ID", string(link.id.Explicit))
	entry := page.hyperlink_set.Get(s.cursor.hyperlink_id)
	expectEqualStrings(t, "https://example.com/", string(entry.uri))
	expectEqualStrings(t, "ID", string(entry.id.Explicit))
}

// sc3OStyleAndLink is term51OStyleAndLink: give the cursor a bold style and
// an explicit-id hyperlink and check that each is accounted for once on the
// cursor's page.
func sc3OStyleAndLink(t *testing.T, s *Screen) {
	t.Helper()
	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	sc3StartLink(t, s, "https://example.com/", "ID")
	page := &s.cursor.page_pin.node.data
	expectEqual(t, 1, sc3OStyleRefs(page, s.cursor.style_id))
	expectEqual(t, 1, sc3OLinkRefs(page, s.cursor.hyperlink_id))
}

// sc3OWrapped is term51OWrapped: a 5x3 screen with a wide character that did
// not fit on row 0: row 0 is "abcd" followed by a spacer head, row 1 starts
// with the wide character (cursor at (2, 1)).
func sc3OWrapped(t *testing.T) *Screen {
	t.Helper()
	s := sc3Init(t, 5, 3, 0)
	sc3Write(t, s, "abcd\U0001F600")
	expectEqual(t, PageCellWideSpacerHead, sc3OCell(t, s, 4, 0).wide)
	expect(t, sc3ORow(t, s, 0).wrap, "term51ORow(&s, 0).wrap")
	expect(t, sc3ORow(t, s, 1).wrap_continuation, "term51ORow(&s, 1).wrap_continuation")
	expectEqual(t, PageCellWideWide, sc3OCell(t, s, 0, 1).wide)
	expectEqual(t, 2, s.cursor.x)
	expectEqual(t, 1, s.cursor.y)
	return s
}

// ---- term51Q helpers ------------------------------------------------------

// sc3QExpectDump is term51QExpectDump: assert that dumping the given region
// of the screen gives expected.
func sc3QExpectDump(t *testing.T, s *Screen, tl Point, expected string) {
	t.Helper()
	str, err := s.DumpStringAlloc(tl)
	zigTry(t, err)
	expectEqualStrings(t, expected, string(str))
}

// sc3QHistoryScreen is term51QHistoryScreen: a 5x3 screen with 1000 bytes of
// scrollback holding the ten rows a0..a9: a0..a6 are history, a7..a9 are the
// active area.
func sc3QHistoryScreen(t *testing.T) *Screen {
	t.Helper()
	s := sc3Init(t, 5, 3, 1000)
	sc3Write(t, s, "a0\na1\na2\na3\na4\na5\na6\na7\na8\na9")
	return s
}

// sc3QCell is term51QCell: the cell at an active-area coordinate.
func sc3QCell(t *testing.T, s *Screen, x SizeCellCountInt, y uint32) *PageCell {
	t.Helper()
	return zigUnwrap(t, s.pages.GetCell(sc3Active(x, y))).cell
}

// sc3QRow is term51QRow: the row at an active-area coordinate.
func sc3QRow(t *testing.T, s *Screen, y uint32) *PageRow {
	t.Helper()
	return zigUnwrap(t, s.pages.GetCell(sc3Active(0, y))).row
}

// sc3QMove is term51QMove: move the cursor and clear a pending wrap, like a
// terminal does.
func sc3QMove(s *Screen, x SizeCellCountInt, y SizeCellCountInt) {
	s.CursorAbsolute(x, y)
	s.cursor.pending_wrap = false
}

// sc3QPut is term51QPut: write one cell directly (no cursor movement) with a
// codepoint.
func sc3QPut(t *testing.T, s *Screen, x SizeCellCountInt, y uint32, cp uint32) *PageCell {
	t.Helper()
	c := sc3QCell(t, s, x, y)
	*c = PageCellInit(cp)
	return c
}

// sc3QBits is term51QBits: the bit pattern of a cell, to compare cells
// exactly.
func sc3QBits(c PageCell) uint64 {
	return c.bitCast()
}

// sc3QStyleUses is term51QStyleUses: how many cells in the page use the given
// (non-default) style id.
func sc3QStyleUses(page *Page, id StyleId) uint {
	var n uint
	rows := page.rows[0:page.size.rows]
	for i := range rows {
		cells := page.GetCells(&rows[i])[0:page.size.cols]
		for _, c := range cells {
			if c.style_id == id {
				n++
			}
		}
	}
	return n
}

// sc3QLinkUses is term51QLinkUses: how many cells in the page carry the
// given hyperlink id.
func sc3QLinkUses(page *Page, id HyperlinkId) uint {
	var n uint
	rows := page.rows[0:page.size.rows]
	for i := range rows {
		cells := page.GetCells(&rows[i])[0:page.size.cols]
		for j := range cells {
			c := &cells[j]
			if c.hyperlink {
				got := page.LookupHyperlink(c)
				if got.Valid && got.Value == id {
					n++
				}
			}
		}
	}
	return n
}

// sc3QLiveStyles is term51QLiveStyles: the number of live (reference count
// above zero) styles in a page, counted independently of `styles.count()`.
func sc3QLiveStyles(page *Page) uint {
	var n uint
	for id := StyleId(1); id < page.styles.next_id; id++ {
		if page.styles.RefCount(id) > 0 {
			n++
		}
	}
	return n
}

// sc3QLiveLinks is term51QLiveLinks: same as sc3QLiveStyles for the
// hyperlink set.
func sc3QLiveLinks(page *Page) uint {
	var n uint
	for id := HyperlinkId(1); id < page.hyperlink_set.next_id; id++ {
		if page.hyperlink_set.RefCount(id) > 0 {
			n++
		}
	}
	return n
}

// sc3QExpectExactRefs is term51QExpectExactRefs: verify the reference count
// of every style and hyperlink of every page exactly: a count equals the
// number of cells using the id, plus one for the cursor on the cursor's page
// when the cursor holds that id. Also verifies the cached cursor pointers and
// the live counters.
func sc3QExpectExactRefs(t *testing.T, s *Screen) {
	t.Helper()

	rac := s.cursor.page_pin.RowAndCell()
	expect(t, s.cursor.page_row == rac.row, "s.cursor.page_row == rac.row")
	expect(t, s.cursor.page_cell == rac.cell, "s.cursor.page_cell == rac.cell")
	expectEqual(t, s.cursor.x, s.cursor.page_pin.x)
	if s.cursor.style_id == StyleDefaultId {
		expect(t, s.cursor.style.Default(), "s.cursor.style.default()")
	} else {
		expect(t, !s.cursor.style.Default(), "!s.cursor.style.default()")
	}
	expectEqual(t, s.cursor.hyperlink_id == 0, s.cursor.hyperlink == nil)

	for node := s.pages.pages.first; node != nil; node = node.next {
		page := &node.data
		is_cursor := node == s.cursor.page_pin.node

		for id := StyleId(1); id < page.styles.next_id; id++ {
			expected := sc3QStyleUses(page, id)
			if is_cursor && s.cursor.style_id == id {
				expected += 1
			}
			expectEqual(t, expected, uint(page.styles.RefCount(id)))
		}
		expectEqual(t, sc3QLiveStyles(page), page.styles.Count())

		for lid := HyperlinkId(1); lid < page.hyperlink_set.next_id; lid++ {
			expected := sc3QLinkUses(page, lid)
			if is_cursor && s.cursor.hyperlink_id == lid {
				expected += 1
			}
			expectEqual(t, expected, uint(page.hyperlink_set.RefCount(lid)))
		}
		expectEqual(t, sc3QLiveLinks(page), page.hyperlink_set.Count())
	}

	// The cursor's style and hyperlink are stored on the cursor page.
	page := &s.cursor.page_pin.node.data
	if s.cursor.style_id != StyleDefaultId {
		expect(t, page.styles.Get(s.cursor.style_id).Eql(s.cursor.style), "page.styles.get(page.memory, s.cursor.style_id).eql(s.cursor.style)")
	}
	if link := s.cursor.hyperlink; link != nil {
		entry := page.hyperlink_set.Get(s.cursor.hyperlink_id)
		expectEqualStrings(t, string(link.uri), string(entry.uri))
		switch link.id.Tag {
		case HyperlinkHyperlinkIdExplicit:
			expect(t, entry.id.Tag == HyperlinkPageEntryIdExplicit, "entry.id == .explicit")
			expectEqualStrings(t, string(link.id.Explicit), string(entry.id.Explicit))
		case HyperlinkHyperlinkIdImplicit:
			expect(t, entry.id.Tag == HyperlinkPageEntryIdImplicit, "entry.id == .implicit")
			expectEqual(t, link.id.Implicit, entry.id.Implicit)
		}
	}
}

// sc3QClearKittyDirty is term51QClearKittyDirty: reset the kitty dirty flag
// when that feature is compiled in. It is compiled out in the library build.
func sc3QClearKittyDirty(s *Screen) {
	// Zig-only: if (comptime build_options.kitty_graphics) s.kitty_images.dirty = false;
}

// sc3QKittyDirty is term51QKittyDirty: the kitty dirty flag, or true when the
// feature is compiled out (so that callers can assert it uniformly).
func sc3QKittyDirty(s *Screen) bool {
	// Zig-only: if (comptime build_options.kitty_graphics) return s.kitty_images.dirty;
	return true
}

// sc3QCrossPageScreen is term51QCrossPageScreen: a `cols` x 3 screen whose
// active area spans two pages: active rows 0 and 1 are the last two rows of
// the first page and active row 2 is the only row of the second page. The
// cursor is at (0, 2).
func sc3QCrossPageScreen(t *testing.T, cols SizeCellCountInt) *Screen {
	t.Helper()
	s := sc3Init(t, cols, 3, 10*1024*1024)
	first := s.pages.pages.first
	cap := uint(first.data.capacity.rows)
	s.CursorAbsolute(0, 2)
	first.data.PauseIntegrityChecks(true)
	for i := uint(0); i < cap-2; i++ {
		zigTry(t, s.CursorDownScroll())
	}
	first.data.PauseIntegrityChecks(false)
	assert(s.pages.pages.first != s.pages.pages.last)
	assert(s.pages.pages.last.data.size.rows == 1)
	assert(s.cursor.page_pin.node == s.pages.pages.last)
	return s
}

// sc3QGrapheme is term51QGrapheme: put `cp` followed by one grapheme
// codepoint 0x301 in a cell of the cursor page's row `y` (directly, no cursor
// movement).
func sc3QGrapheme(t *testing.T, s *Screen, x SizeCellCountInt, y uint32, cp uint32) {
	t.Helper()
	c := zigUnwrap(t, s.pages.GetCell(sc3Active(x, y)))
	*c.cell = PageCellInit(cp)
	zigTry(t, c.node.data.AppendGrapheme(c.row, c.cell, 0x301))
}

// sc3QTextScreen is term51QTextScreen: a 6x3 screen (with the given
// scrollback) holding `text`.
func sc3QTextScreen(t *testing.T, text string, scrollback uint64) *Screen {
	t.Helper()
	s := sc3Init(t, 6, 3, scrollback)
	sc3Write(t, s, text)
	return s
}

// sc3QAttrCase is term51QAttrCase.
type sc3QAttrCase struct {
	attr     SGRAttribute
	expected Style
}

// sc3QFullStyle is term51QFullStyle: a style with every attribute set to a
// distinct non-default value.
func sc3QFullStyle() Style {
	return Style{
		fg_color:        StyleColor{Tag: StyleColorPalette, Palette: 200},
		bg_color:        StyleColor{Tag: StyleColorPalette, Palette: 201},
		underline_color: StyleColor{Tag: StyleColorPalette, Palette: 202},
		flags: styleFlags{
			bold:          true,
			italic:        true,
			faint:         true,
			blink:         true,
			inverse:       true,
			invisible:     true,
			strikethrough: true,
			overline:      true,
			underline:     SGRAttributeUnderlineCurly,
		},
	}
}

// sc3QApplyFull is term51QApplyFull: apply attributes that build
// sc3QFullStyle.
func sc3QApplyFull(t *testing.T, s *Screen) {
	t.Helper()
	attrs := []SGRAttribute{
		sc3A(SGRAttributeTagBold),
		sc3A(SGRAttributeTagItalic),
		sc3A(SGRAttributeTagFaint),
		sc3A(SGRAttributeTagBlink),
		sc3A(SGRAttributeTagInverse),
		sc3A(SGRAttributeTagInvisible),
		sc3A(SGRAttributeTagStrikethrough),
		sc3A(SGRAttributeTagOverline),
		sc3AttrUnderline(SGRAttributeUnderlineCurly),
		sc3AttrFg256(200),
		sc3AttrBg256(201),
		{Tag: SGRAttributeTag256UnderlineColor, X256UnderlineColor: 202},
	}
	for _, a := range attrs {
		zigTry(t, s.SetAttribute(a))
	}
}

// ---------------------------------------------------------------------------
// Tests.
// ---------------------------------------------------------------------------

// Zig: test "term51: cursorScrollAbove: cursor on the previous page moves to the next page"
func TestSc3_term51_cursorScrollAbove_cursor_on_the_previous_page_moves_to_the_next_page(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	s := sc3OTwoPages(t, 20, 3)
	defer s.Deinit()
	sc3OSetCp(t, s, 0, 0, 'A')
	sc3OSetCp(t, s, 0, 1, 'B')
	sc3OSetCp(t, s, 0, 2, 'C')
	s.CursorAbsolute(6, 1)
	sc3OStyleAndLink(t, s)
	first := s.cursor.page_pin.node
	old_style_id := s.cursor.style_id
	old_link_id := s.cursor.hyperlink_id
	expect(t, first == s.pages.pages.first, "first == s.pages.pages.first.?")
	s.pages.ClearDirty()
	total := s.pages.total_rows

	zigTry(t, s.CursorScrollAbove())

	// No new page was needed; the cursor's row is now the first row of the
	// second page, with its style and link moved over.
	expectEqual(t, 2, sc3ONodeCount(s))
	expectEqual(t, total+1, s.pages.total_rows)
	expect(t, s.cursor.page_pin.node == s.pages.pages.last, "s.cursor.page_pin.node == s.pages.pages.last.?")
	expectEqual(t, 0, s.cursor.page_pin.y)
	expectEqual(t, 6, s.cursor.x)
	expectEqual(t, 1, s.cursor.y)
	sc3OExpectMovedFrom(t, s, first, old_style_id, old_link_id)
	sc3OExpectCursorConsistent(t, s)

	expectEqual(t, 'B', sc3OCp(t, s, 0, 0))
	sc3OExpectBlank(t, sc3OCell(t, s, 0, 1))
	expectEqual(t, 'C', sc3OCp(t, s, 0, 2))
	// The page the cursor moved to is entirely dirty, the old page's row
	// the cursor left is dirty too.
	expect(t, s.cursor.page_pin.node.data.dirty, "s.cursor.page_pin.node.data.dirty")
	expect(t, first.data.IsDirty(), "first.data.isDirty()")
}

// Zig: test "term51: cursorScrollAbove: a full page makes a new page and copies the last row"
func TestSc3_term51_cursorScrollAbove_a_full_page_makes_a_new_page_and_copies_the_last_row(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	s := sc3OFullPage(t, 200, 3)
	defer s.Deinit()
	sc3OSetCp(t, s, 0, 0, 'A')
	sc3OSetCp(t, s, 0, 1, 'B')
	sc3OSetCp(t, s, 0, 2, 'C')
	s.CursorAbsolute(3, 1)
	sc3OStyleAndLink(t, s)
	first := s.cursor.page_pin.node
	style_id := s.cursor.style_id
	link_id := s.cursor.hyperlink_id
	s.pages.ClearDirty()

	zigTry(t, s.CursorScrollAbove())

	expectEqual(t, 2, sc3ONodeCount(s))
	last := s.pages.pages.last
	expectEqual(t, 1, last.data.size.rows)
	// The cursor stays on the first page: its accounting is untouched.
	expect(t, s.cursor.page_pin.node == first, "s.cursor.page_pin.node == first")
	expectEqual(t, style_id, s.cursor.style_id)
	expectEqual(t, link_id, s.cursor.hyperlink_id)
	expectEqual(t, 1, sc3OStyleRefs(&first.data, style_id))
	expectEqual(t, 1, sc3OLinkRefs(&first.data, link_id))
	expectEqual(t, 3, s.cursor.x)
	expectEqual(t, 1, s.cursor.y)
	sc3OExpectCursorConsistent(t, s)

	expectEqual(t, 'B', sc3OCp(t, s, 0, 0))
	sc3OExpectBlank(t, sc3OCell(t, s, 0, 1))
	expectEqual(t, 'C', sc3OCp(t, s, 0, 2))
	expectEqual(t, 'A', sc3OScreenCell(t, s, 0, int(first.data.size.rows)-3).Codepoint())
	expect(t, first.data.dirty, "first.data.dirty")
	expect(t, last.data.IsDirty(), "last.data.isDirty()")
}

// Zig: test "term51: cursorScrollAbove: across several pages, with and without a new page"
func TestSc3_term51_cursorScrollAbove_across_several_pages_with_and_without_a_new_page(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;

	// Last page has room: grow does not allocate.
	func() {
		s := sc3OMultiPage(t, 100*1024*1024)
		defer s.Deinit()
		for y := 0; y < 60; y++ {
			sc3OSetCp(t, s, 0, y, uint32(0x100+y))
		}
		s.CursorAbsolute(5, 5)
		nodes := sc3ONodeCount(s)
		last := s.pages.pages.last
		expect(t, last.data.size.rows < last.data.capacity.rows, "last.data.size.rows < last.data.capacity.rows")

		zigTry(t, s.CursorScrollAbove())
		expectEqual(t, nodes, sc3ONodeCount(s))
		expectEqual(t, 61, s.pages.total_rows)
		expectEqual(t, 5, s.cursor.x)
		expectEqual(t, 5, s.cursor.y)
		sc3OExpectCursorConsistent(t, s)
		for y := 0; y < 60; y++ {
			var want uint32
			if y < 5 {
				want = uint32(0x100 + y + 1)
			} else if y == 5 {
				want = 0
			} else {
				want = uint32(0x100 + y)
			}
			expectEqual(t, want, sc3OCp(t, s, 0, y))
		}
		expectEqual(t, 0x100, sc3OScreenCell(t, s, 0, 0).Codepoint())
		s.AssertIntegrity()
	}()

	// Last page full: a new page is created and rows ripple across pages.
	func() {
		probe := sc3Init(t, 2000, 1, 0)
		cap := probe.pages.pages.first.data.capacity.rows
		probe.Deinit()
		rows := cap * 2

		s := sc3Init(t, 2000, rows, 100*1024*1024)
		defer s.Deinit()
		expectEqual(t, 2, sc3ONodeCount(s))
		expectEqual(t, cap, s.pages.pages.last.data.size.rows)
		for y := 0; y < int(rows); y++ {
			sc3OSetCp(t, s, 0, y, uint32(0x100+y))
		}
		s.CursorAbsolute(5, 3)

		zigTry(t, s.CursorScrollAbove())
		expectEqual(t, 3, sc3ONodeCount(s))
		expectEqual(t, uint(rows)+1, s.pages.total_rows)
		expectEqual(t, 5, s.cursor.x)
		expectEqual(t, 3, s.cursor.y)
		sc3OExpectCursorConsistent(t, s)
		for y := 0; y < int(rows); y++ {
			var want uint32
			if y < 3 {
				want = uint32(0x100 + y + 1)
			} else if y == 3 {
				want = 0
			} else {
				want = uint32(0x100 + y)
			}
			expectEqual(t, want, sc3OCp(t, s, 0, y))
		}
		s.AssertIntegrity()
	}()
}

// Zig: test "term51: cursorScrollAboveRotate: new page after a grow, styled rows are copied"
func TestSc3_term51_cursorScrollAboveRotate_new_page_after_a_grow_styled_rows_are_copied(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	s := sc3OFullPage(t, 200, 3)
	defer s.Deinit()
	sc3OSetCp(t, s, 0, 0, 'A')
	sc3OSetCp(t, s, 0, 1, 'B')
	s.CursorAbsolute(0, 2)
	sc3SetAttr(t, s, sc3AttrBgRGB(1, 2, 3))
	sc3Write(t, s, "C")
	s.CursorAbsolute(4, 1)
	first := s.cursor.page_pin.node
	style_id := s.cursor.style_id
	// The styled 'C' and the cursor.
	expectEqual(t, 2, sc3OStyleRefs(&first.data, style_id))

	// What cursorScrollAbove does before calling us.
	new_node, err := s.pages.Grow()
	zigTry(t, err)
	expect(t, new_node != nil, "(try s.pages.grow()).? != null")
	expect(t, new_node == s.pages.pages.last, "new_node == s.pages.pages.last.?")
	zigTry(t, s.cursorScrollAboveRotate())

	expect(t, s.cursor.page_pin.node == first, "s.cursor.page_pin.node == first")
	expectEqual(t, 4, s.cursor.x)
	expectEqual(t, 1, s.cursor.y)
	sc3OExpectCursorConsistent(t, s)
	expectEqual(t, 'B', sc3OCp(t, s, 0, 0))
	// The cursor row is cleared with the cursor's background (clearCells
	// uses the blank cell of the cursor).
	sc3OExpectBgRow(t, s, 1, [3]uint8{1, 2, 3})
	expectEqual(t, 'C', sc3OCp(t, s, 0, 2))

	// The copy of the styled row lives in the new page with its own style
	// reference; the original row's reference was released.
	copy := sc3OCell(t, s, 0, 2)
	expectEqual(t, 1, new_node.data.styles.Count())
	expectEqual(t, 1, sc3OStyleRefs(&new_node.data, copy.style_id))
	expectEqual(t, 1, sc3OStyleRefs(&first.data, style_id))
	expect(t, first.data.dirty, "first.data.dirty")
	expect(t, new_node.data.dirty, "new_node.data.dirty")
}

// Zig: test "term51: cursorScrollAboveRotate: cursor moves onto the next page when it is on the last row of its page"
func TestSc3_term51_cursorScrollAboveRotate_cursor_moves_onto_the_next_page_when_it_is_on_the_last_row_of_its_page(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	s := sc3OTwoPages(t, 20, 3)
	defer s.Deinit()
	sc3OSetCp(t, s, 0, 0, 'A')
	sc3OSetCp(t, s, 0, 1, 'B')
	sc3OSetCp(t, s, 0, 2, 'C')
	s.CursorAbsolute(2, 1)
	sc3OStyleAndLink(t, s)
	first := s.cursor.page_pin.node
	old_style_id := s.cursor.style_id
	old_link_id := s.cursor.hyperlink_id

	grown, err := s.pages.Grow()
	zigTry(t, err)
	expect(t, grown == nil, "(try s.pages.grow()) == null")
	zigTry(t, s.cursorScrollAboveRotate())

	expect(t, s.cursor.page_pin.node == s.pages.pages.last, "s.cursor.page_pin.node == s.pages.pages.last.?")
	expectEqual(t, 0, s.cursor.page_pin.y)
	expectEqual(t, 2, s.cursor.x)
	expectEqual(t, 1, s.cursor.y)
	sc3OExpectMovedFrom(t, s, first, old_style_id, old_link_id)
	sc3OExpectCursorConsistent(t, s)
	expectEqual(t, 'B', sc3OCp(t, s, 0, 0))
	sc3OExpectBlank(t, sc3OCell(t, s, 0, 1))
	expectEqual(t, 'C', sc3OCp(t, s, 0, 2))
	expect(t, s.cursor.page_pin.node.data.dirty, "s.cursor.page_pin.node.data.dirty")
}

// Zig: test "term51: cursorScrollAboveRotate: several intermediate pages"
func TestSc3_term51_cursorScrollAboveRotate_several_intermediate_pages(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	s := sc3OMultiPage(t, 100*1024*1024)
	defer s.Deinit()
	for y := 0; y < 60; y++ {
		sc3OSetCp(t, s, 0, y, uint32(0x100+y))
	}
	s.CursorAbsolute(5, 5)
	s.pages.ClearDirty()

	grown, err := s.pages.Grow()
	zigTry(t, err)
	expect(t, grown == nil, "(try s.pages.grow()) == null")
	zigTry(t, s.cursorScrollAboveRotate())

	expectEqual(t, 5, s.cursor.x)
	expectEqual(t, 5, s.cursor.y)
	sc3OExpectCursorConsistent(t, s)
	for y := 0; y < 60; y++ {
		var want uint32
		if y < 5 {
			want = uint32(0x100 + y + 1)
		} else if y == 5 {
			want = 0
		} else {
			want = uint32(0x100 + y)
		}
		expectEqual(t, want, sc3OCp(t, s, 0, y))
	}
	// Every page from the cursor's to the end is wholly dirty.
	for node := s.cursor.page_pin.node; node != nil; node = node.next {
		expect(t, node.data.dirty, "node.data.dirty")
	}
	s.AssertIntegrity()
}

// Zig: test "term51: cursorDownOrScroll: moves down until the last row, then scrolls"
func TestSc3_term51_cursorDownOrScroll_moves_down_until_the_last_row_then_scrolls(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	s := sc3Init(t, 5, 3, 100)
	defer s.Deinit()

	zigTry(t, s.cursorDownOrScroll())
	expectEqual(t, 1, s.cursor.y)
	zigTry(t, s.cursorDownOrScroll())
	expectEqual(t, 2, s.cursor.y)
	expectEqual(t, 3, s.pages.total_rows)
	zigTry(t, s.cursorDownOrScroll())
	expectEqual(t, 2, s.cursor.y)
	expectEqual(t, 4, s.pages.total_rows)
	sc3OExpectCursorConsistent(t, s)

	// Without scrollback it shifts instead.
	s2 := sc3Init(t, 5, 2, 0)
	defer s2.Deinit()
	sc3Write(t, s2, "a")
	zigTry(t, s2.cursorDownOrScroll())
	expectEqual(t, 1, s2.cursor.y)
	sc3OSetCp(t, s2, 0, 1, 'b')
	zigTry(t, s2.cursorDownOrScroll())
	expectEqual(t, 1, s2.cursor.y)
	expectEqual(t, 2, s2.pages.total_rows)
	expectEqual(t, 'b', sc3OCp(t, s2, 0, 0))
	sc3OExpectBlank(t, sc3OCell(t, s2, 0, 1))

	// A one row screen always scrolls.
	s3 := sc3Init(t, 5, 1, 100)
	defer s3.Deinit()
	zigTry(t, s3.cursorDownOrScroll())
	expectEqual(t, 0, s3.cursor.y)
	expectEqual(t, 2, s3.pages.total_rows)
}

// Zig: test "term51: cursorCopy: copies cursor fields and keeps the destination's pin"
func TestSc3_term51_cursorCopy_copies_cursor_fields_and_keeps_the_destination_s_pin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	src := sc3Init(t, 20, 10, 0)
	defer src.Deinit()
	src.CursorAbsolute(5, 3)
	sc3SetAttr(t, src, sc3A(SGRAttributeTagItalic))
	src.cursor.cursor_style = ScreenCursorStyleBar
	src.cursor.protected = true
	src.cursor.pending_wrap = true
	src.cursor.hyperlink_implicit_id = 4
	src.CursorSetSemanticContent(ScreenCursorSetSemanticContent{
		Tag:   ScreenCursorSetSemanticContentTagInput,
		Input: ScreenCursorSetSemanticContentInputClearEol,
	})

	dst := sc3Init(t, 10, 5, 0)
	defer dst.Deinit()
	dst_pin := dst.cursor.page_pin
	sc3SetAttr(t, dst, sc3A(SGRAttributeTagBold))
	dst_page := &dst.cursor.page_pin.node.data

	zigTry(t, dst.CursorCopy(src.cursor, screenCursorCopyOptionsDefault()))

	expectEqual(t, 5, dst.cursor.x)
	expectEqual(t, 3, dst.cursor.y)
	expectEqual(t, CursorStyleBar, dst.cursor.cursor_style)
	expect(t, dst.cursor.protected, "dst.cursor.protected")
	expect(t, dst.cursor.pending_wrap, "dst.cursor.pending_wrap")
	expectEqual(t, 4, dst.cursor.hyperlink_implicit_id)
	expectEqual(t, PageCellSemanticContentInput, dst.cursor.semantic_content)
	expect(t, dst.cursor.semantic_content_clear_eol, "dst.cursor.semantic_content_clear_eol")
	// Style value copied, and registered on our page once; the old one gone.
	expect(t, dst.cursor.style.flags.italic, "dst.cursor.style.flags.italic")
	expect(t, !dst.cursor.style.flags.bold, "!dst.cursor.style.flags.bold")
	expect(t, dst.cursor.style_id != StyleDefaultId, "dst.cursor.style_id != style.default_id")
	expectEqual(t, 1, dst_page.styles.Count())
	expectEqual(t, 1, sc3OStyleRefs(dst_page, dst.cursor.style_id))
	// Our own pin and caches.
	expect(t, dst.cursor.page_pin == dst_pin, "dst.cursor.page_pin == dst_pin")
	expect(t, dst.cursor.page_pin != src.cursor.page_pin, "dst.cursor.page_pin != src.cursor.page_pin")
	sc3OExpectCursorConsistent(t, dst)
	dst.AssertIntegrity()

	// The source is untouched.
	expectEqual(t, 5, src.cursor.x)
	expect(t, src.cursor.style.flags.italic, "src.cursor.style.flags.italic")
	sc3OExpectCursorConsistent(t, src)
}

// Zig: test "term51: cursorCopy: bounds of the destination"
func TestSc3_term51_cursorCopy_bounds_of_the_destination(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	src := sc3Init(t, 10, 5, 0)
	defer src.Deinit()
	dst := sc3Init(t, 10, 5, 0)
	defer dst.Deinit()

	src.CursorAbsolute(9, 4)
	zigTry(t, dst.CursorCopy(src.cursor, screenCursorCopyOptionsDefault()))
	expectEqual(t, 9, dst.cursor.x)
	expectEqual(t, 4, dst.cursor.y)
	sc3OExpectCursorConsistent(t, dst)

	src.CursorAbsolute(0, 0)
	zigTry(t, dst.CursorCopy(src.cursor, screenCursorCopyOptionsDefault()))
	expectEqual(t, 0, dst.cursor.x)
	expectEqual(t, 0, dst.cursor.y)
	sc3OExpectCursorConsistent(t, dst)

	// A cursor from a bigger screen that still fits.
	big := sc3Init(t, 30, 20, 0)
	defer big.Deinit()
	big.CursorAbsolute(9, 4)
	zigTry(t, dst.CursorCopy(big.cursor, screenCursorCopyOptionsDefault()))
	expectEqual(t, 9, dst.cursor.x)
	expectEqual(t, 4, dst.cursor.y)
	sc3OExpectCursorConsistent(t, dst)
	dst.AssertIntegrity()
}

// Zig: test "term51: cursorCopy: explicit and implicit hyperlinks are re-created on our page"
func TestSc3_term51_cursorCopy_explicit_and_implicit_hyperlinks_are_re_created_on_our_page(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	src := sc3Init(t, 10, 5, 0)
	defer src.Deinit()
	dst := sc3Init(t, 10, 5, 0)
	defer dst.Deinit()

	// Explicit id
	sc3StartLink(t, src, "https://example.com/a", "my-id")
	zigTry(t, dst.CursorCopy(src.cursor, screenCursorCopyOptionsDefault()))
	expect(t, dst.cursor.hyperlink != nil, "dst.cursor.hyperlink != null")
	expect(t, dst.cursor.hyperlink != src.cursor.hyperlink, "dst.cursor.hyperlink.? != src.cursor.hyperlink.?")
	expectEqualStrings(t, "https://example.com/a", string(dst.cursor.hyperlink.uri))
	expectEqualStrings(t, "my-id", string(dst.cursor.hyperlink.id.Explicit))
	expect(t, dst.cursor.hyperlink_id != 0, "dst.cursor.hyperlink_id != 0")
	{
		page := &dst.cursor.page_pin.node.data
		expectEqual(t, 1, page.hyperlink_set.Count())
		expectEqual(t, 1, sc3OLinkRefs(page, dst.cursor.hyperlink_id))
	}

	// Implicit id: the destination allocates the next id from the copied
	// counter, it is not the source's id.
	sc3StartLinkImplicit(t, src, "https://example.com/b")
	expectEqual(t, 1, src.cursor.hyperlink_implicit_id)
	zigTry(t, dst.CursorCopy(src.cursor, screenCursorCopyOptionsDefault()))
	expectEqual(t, 2, dst.cursor.hyperlink_implicit_id)
	expectEqual(t, 1, dst.cursor.hyperlink.id.Implicit)
	expectEqual(t, 0, src.cursor.hyperlink.id.Implicit)
	expectEqualStrings(t, "https://example.com/b", string(dst.cursor.hyperlink.uri))
	{
		// The explicit one from before was released.
		page := &dst.cursor.page_pin.node.data
		expectEqual(t, 1, page.hyperlink_set.Count())
		expectEqual(t, 1, sc3OLinkRefs(page, dst.cursor.hyperlink_id))
	}

	// Copying a cursor with no hyperlink ends ours; with hyperlink = false
	// we never take the other's but end ours too.
	plain := sc3Init(t, 10, 5, 0)
	defer plain.Deinit()
	zigTry(t, dst.CursorCopy(plain.cursor, screenCursorCopyOptionsDefault()))
	expect(t, dst.cursor.hyperlink == nil, "dst.cursor.hyperlink == null")
	expectEqual(t, 0, dst.cursor.hyperlink_id)

	sc3StartLinkImplicit(t, dst, "https://example.com/c")
	zigTry(t, dst.CursorCopy(src.cursor, ScreenCursorCopyOptions{hyperlink: false}))
	expect(t, dst.cursor.hyperlink == nil, "dst.cursor.hyperlink == null")
	expectEqual(t, 0, dst.cursor.hyperlink_id)
	dst.AssertIntegrity()
}

// Zig: test "term51: cursorCopy: styles from a cursor with a style, replacing and removing ours"
func TestSc3_term51_cursorCopy_styles_from_a_cursor_with_a_style_replacing_and_removing_ours(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	src := sc3Init(t, 10, 5, 0)
	defer src.Deinit()
	dst := sc3Init(t, 10, 5, 0)
	defer dst.Deinit()
	page := &dst.cursor.page_pin.node.data

	// Both have the same style: the page still has exactly one reference.
	sc3SetAttr(t, src, sc3A(SGRAttributeTagBold))
	sc3SetAttr(t, dst, sc3A(SGRAttributeTagBold))
	zigTry(t, dst.CursorCopy(src.cursor, screenCursorCopyOptionsDefault()))
	expectEqual(t, 1, page.styles.Count())
	expectEqual(t, 1, sc3OStyleRefs(page, dst.cursor.style_id))

	// Different styles
	sc3SetAttr(t, src, sc3A(SGRAttributeTagItalic))
	zigTry(t, dst.CursorCopy(src.cursor, screenCursorCopyOptionsDefault()))
	expect(t, dst.cursor.style.flags.bold && dst.cursor.style.flags.italic, "dst.cursor.style.flags.bold and dst.cursor.style.flags.italic")
	expectEqual(t, 1, sc3OStyleRefs(page, dst.cursor.style_id))

	// Default style from the other cursor removes ours.
	plain := sc3Init(t, 10, 5, 0)
	defer plain.Deinit()
	zigTry(t, dst.CursorCopy(plain.cursor, screenCursorCopyOptionsDefault()))
	expectEqual(t, StyleDefaultId, dst.cursor.style_id)
	expect(t, dst.cursor.style.Default(), "dst.cursor.style.default()")
	expectEqual(t, 0, page.styles.Count())
}

// Zig: test "term51: cursorCopy: moves our cursor across pages carrying style and link"
func TestSc3_term51_cursorCopy_moves_our_cursor_across_pages_carrying_style_and_link(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	src := sc3Init(t, 20, 3, 0)
	defer src.Deinit()
	src.CursorAbsolute(5, 0)
	sc3SetAttr(t, src, sc3A(SGRAttributeTagBold))
	sc3StartLink(t, src, "https://example.com/", "ID")

	dst := sc3OTwoPages(t, 20, 3)
	defer dst.Deinit()
	last := dst.cursor.page_pin.node
	sc3SetAttr(t, dst, sc3A(SGRAttributeTagItalic))
	sc3StartLinkImplicit(t, dst, "https://old.example/")
	old_style_id := dst.cursor.style_id
	old_link_id := dst.cursor.hyperlink_id

	zigTry(t, dst.CursorCopy(src.cursor, screenCursorCopyOptionsDefault()))
	expect(t, dst.cursor.page_pin.node == dst.pages.pages.first, "dst.cursor.page_pin.node == dst.pages.pages.first.?")
	expectEqual(t, 5, dst.cursor.x)
	expectEqual(t, 0, dst.cursor.y)
	sc3OExpectMovedFrom(t, dst, last, old_style_id, old_link_id)
	sc3OExpectCursorConsistent(t, dst)
}

// Zig: test "term51: cursorChangePin: an equal pin changes and marks nothing"
func TestSc3_term51_cursorChangePin_an_equal_pin_changes_and_marks_nothing(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	s := sc3Init(t, 10, 3, 0)
	defer s.Deinit()
	s.CursorAbsolute(2, 1)
	sc3OStyleAndLink(t, s)
	s.pages.ClearDirty()
	pin := *s.cursor.page_pin
	before := s.cursor

	s.cursorChangePin(pin)
	expect(t, s.cursor.page_pin.Eql(pin), "s.cursor.page_pin.eql(pin)")
	expect(t, before == s.cursor, "std.meta.eql(before, s.cursor)")
	expectEqualSlices(t, []bool{false, false, false}, sc3ODirtyRows(t, s, 3))
	expectEqual(t, 1, sc3OStyleRefs(&s.cursor.page_pin.node.data, s.cursor.style_id))
}

// Zig: test "term51: cursorChangePin: another row of the same page only updates the pin"
func TestSc3_term51_cursorChangePin_another_row_of_the_same_page_only_updates_the_pin(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	s := sc3Init(t, 10, 3, 0)
	defer s.Deinit()
	s.CursorAbsolute(2, 0)
	sc3OStyleAndLink(t, s)
	style_id := s.cursor.style_id
	link_id := s.cursor.hyperlink_id
	link_ptr := s.cursor.hyperlink
	pin_ptr := s.cursor.page_pin
	s.pages.ClearDirty()

	// The caller sets the coordinates first.
	s.cursor.y = 2
	s.cursorChangePin(sc3OPin(t, s, 2, 2))
	expect(t, s.cursor.page_pin == pin_ptr, "s.cursor.page_pin == pin_ptr")
	expectEqual(t, 2, s.cursor.page_pin.y)
	expectEqual(t, 2, s.cursor.page_pin.x)
	// Style and link are not touched when the page stays the same, and the
	// cached row and cell are NOT updated here.
	expectEqual(t, style_id, s.cursor.style_id)
	expectEqual(t, link_id, s.cursor.hyperlink_id)
	expect(t, s.cursor.hyperlink == link_ptr, "s.cursor.hyperlink.? == link_ptr")
	expect(t, s.cursor.page_row == sc3ORow(t, s, 0), "s.cursor.page_row == term51ORow(&s, 0)")
	// The old row (via the cache) and the new row are dirty.
	expectEqualSlices(t, []bool{true, false, true}, sc3ODirtyRows(t, s, 3))

	// Only the x changes: same row, still different pin, so it is marked.
	s.pages.ClearDirty()
	s.cursor.page_row = sc3ORow(t, s, 2)
	s.cursor.x = 5
	s.cursorChangePin(sc3OPin(t, s, 5, 2))
	expectEqual(t, 5, s.cursor.page_pin.x)
	expectEqualSlices(t, []bool{false, false, true}, sc3ODirtyRows(t, s, 3))
}

// Zig: test "term51: cursorChangePin: another page moves style and hyperlink"
func TestSc3_term51_cursorChangePin_another_page_moves_style_and_hyperlink(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	s := sc3OTwoPages(t, 20, 3)
	defer s.Deinit()
	s.CursorAbsolute(4, 1)
	sc3OStyleAndLink(t, s)
	first := s.cursor.page_pin.node
	old_style_id := s.cursor.style_id
	old_link_id := s.cursor.hyperlink_id
	old_link_ptr := s.cursor.hyperlink
	pin_ptr := s.cursor.page_pin
	s.pages.ClearDirty()

	s.cursor.y = 2
	s.cursorChangePin(sc3OPin(t, s, 4, 2))

	expect(t, s.cursor.page_pin == pin_ptr, "s.cursor.page_pin == pin_ptr")
	expect(t, s.cursor.page_pin.node == s.pages.pages.last, "s.cursor.page_pin.node == s.pages.pages.last.?")
	expectEqual(t, 0, s.cursor.page_pin.y)
	sc3OExpectMovedFrom(t, s, first, old_style_id, old_link_id)
	// A new heap allocation for the link (the old one was freed).
	expect(t, s.cursor.hyperlink != old_link_ptr, "s.cursor.hyperlink.? != old_link_ptr")
	// Old row via the cache and the new row are dirty.
	expect(t, sc3ORow(t, s, 1).dirty, "term51ORow(&s, 1).dirty")
	expect(t, sc3ORow(t, s, 2).dirty, "term51ORow(&s, 2).dirty")
	expect(t, !sc3ORow(t, s, 0).dirty, "!term51ORow(&s, 0).dirty")

	// Repair the caches like callers do.
	rac := s.cursor.page_pin.RowAndCell()
	s.cursor.page_row = rac.row
	s.cursor.page_cell = rac.cell
	sc3OExpectCursorConsistent(t, s)
}

// Zig: test "term51: cursorChangePin: another page with only a style, only a link, or neither"
func TestSc3_term51_cursorChangePin_another_page_with_only_a_style_only_a_link_or_neither(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;

	// Only a style
	func() {
		s := sc3OTwoPages(t, 20, 3)
		defer s.Deinit()
		s.CursorAbsolute(4, 1)
		sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
		first := s.cursor.page_pin.node
		old_style_id := s.cursor.style_id
		s.cursor.y = 2
		s.cursorChangePin(sc3OPin(t, s, 4, 2))
		expect(t, s.cursor.page_pin.node == s.pages.pages.last, "s.cursor.page_pin.node == s.pages.pages.last.?")
		expectEqual(t, 0, sc3OStyleRefs(&first.data, old_style_id))
		expectEqual(t, 1, sc3OStyleRefs(&s.cursor.page_pin.node.data, s.cursor.style_id))
		expect(t, s.cursor.hyperlink == nil, "s.cursor.hyperlink == null")
		expectEqual(t, 0, s.cursor.hyperlink_id)
	}()

	// Only a hyperlink
	func() {
		s := sc3OTwoPages(t, 20, 3)
		defer s.Deinit()
		s.CursorAbsolute(4, 1)
		sc3StartLink(t, s, "https://example.com/", "ID")
		first := s.cursor.page_pin.node
		old_link_id := s.cursor.hyperlink_id
		s.cursor.y = 2
		s.cursorChangePin(sc3OPin(t, s, 4, 2))
		expect(t, s.cursor.page_pin.node == s.pages.pages.last, "s.cursor.page_pin.node == s.pages.pages.last.?")
		expectEqual(t, 0, sc3OLinkRefs(&first.data, old_link_id))
		expectEqual(t, 1, sc3OLinkRefs(&s.cursor.page_pin.node.data, s.cursor.hyperlink_id))
		expectEqual(t, StyleDefaultId, s.cursor.style_id)
		expect(t, s.cursor.style.Default(), "s.cursor.style.default()")
	}()

	// Neither
	func() {
		s := sc3OTwoPages(t, 20, 3)
		defer s.Deinit()
		s.CursorAbsolute(4, 1)
		s.cursor.y = 2
		s.cursorChangePin(sc3OPin(t, s, 4, 2))
		expect(t, s.cursor.page_pin.node == s.pages.pages.last, "s.cursor.page_pin.node == s.pages.pages.last.?")
		expectEqual(t, StyleDefaultId, s.cursor.style_id)
		expect(t, s.cursor.hyperlink == nil, "s.cursor.hyperlink == null")
		expectEqual(t, 0, s.cursor.page_pin.node.data.styles.Count())
		expectEqual(t, 0, s.cursor.page_pin.node.data.hyperlink_set.Count())
	}()
}

// Zig: test "term51: cursorChangePin: a target page with a full hyperlink set is replaced"
func TestSc3_term51_cursorChangePin_a_target_page_with_a_full_hyperlink_set_is_replaced(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	s := sc3OTwoPages(t, 20, 3)
	defer s.Deinit()
	s.CursorAbsolute(4, 2)
	sc3StartLink(t, s, "https://example.com/", "ID")
	last := s.cursor.page_pin.node
	old_link_id := s.cursor.hyperlink_id
	first := s.pages.pages.first

	// Fill the first page's hyperlink set (or its strings).
	func() {
		page := &first.data
		page.PauseIntegrityChecks(true)
		defer page.PauseIntegrityChecks(false)
		for n := 0; ; n++ {
			uri := fmt.Sprintf("https://filler.example/%d", n)
			_, err := page.InsertHyperlink(Hyperlink{
				id:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: SizeOffsetInt(n)},
				uri: []uint8(uri),
			})
			if err != nil {
				break
			}
		}
	}()

	s.cursor.y = 1
	s.cursorChangePin(sc3OPin(t, s, 4, 1))

	// The first page had to be replaced to make room, and the cursor
	// followed it to the new node.
	expect(t, s.cursor.page_pin.node != first, "s.cursor.page_pin.node != first")
	expect(t, s.cursor.page_pin.node != last, "s.cursor.page_pin.node != last")
	expect(t, s.cursor.page_pin.node == s.pages.pages.first, "s.cursor.page_pin.node == s.pages.pages.first.?")
	expect(t, s.cursor.hyperlink != nil, "s.cursor.hyperlink != null")
	expect(t, s.cursor.hyperlink_id != 0, "s.cursor.hyperlink_id != 0")
	page := &s.cursor.page_pin.node.data
	expectEqual(t, 1, sc3OLinkRefs(page, s.cursor.hyperlink_id))
	expectEqualStrings(t, "https://example.com/", string(s.cursor.hyperlink.uri))
	expectEqualStrings(t, "ID", string(s.cursor.hyperlink.id.Explicit))
	expectEqual(t, 0, sc3OLinkRefs(&last.data, old_link_id))
}

// Zig: test "term51: cursorMarkDirty: marks exactly the cached cursor row"
func TestSc3_term51_cursorMarkDirty_marks_exactly_the_cached_cursor_row(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	s := sc3Init(t, 5, 4, 0)
	defer s.Deinit()

	for y := 0; y < 4; y++ {
		s.CursorAbsolute(1, SizeCellCountInt(y))
		s.pages.ClearDirty()
		s.CursorMarkDirty()
		want := []bool{false, false, false, false}
		want[y] = true
		expectEqualSlices(t, want, sc3ODirtyRows(t, s, 4))
		expect(t, !s.cursor.page_pin.node.data.dirty, "!s.cursor.page_pin.node.data.dirty")

		// Idempotent
		s.CursorMarkDirty()
		expectEqualSlices(t, want, sc3ODirtyRows(t, s, 4))
	}

	// It uses the cached row pointer, not the pin.
	s.pages.ClearDirty()
	real := s.cursor.page_row
	s.cursor.page_row = sc3ORow(t, s, 0)
	s.CursorMarkDirty()
	s.cursor.page_row = real
	expectEqualSlices(t, []bool{true, false, false, false}, sc3ODirtyRows(t, s, 4))
}

// Zig: test "term51: cursorResetWrap: row without wrap only clears the pending wrap"
func TestSc3_term51_cursorResetWrap_row_without_wrap_only_clears_the_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	s := sc3Init(t, 5, 3, 0)
	defer s.Deinit()
	sc3Write(t, s, "ab")
	s.cursor.pending_wrap = true
	// Sentinels that an early return must not touch.
	sc3ORow(t, s, 1).wrap_continuation = true
	*sc3OCell(t, s, 4, 0) = sc3CellWide(0, PageCellWideSpacerHead)

	s.CursorResetWrap()
	expect(t, !s.cursor.pending_wrap, "!s.cursor.pending_wrap")
	expect(t, !sc3ORow(t, s, 0).wrap, "!term51ORow(&s, 0).wrap")
	expect(t, sc3ORow(t, s, 1).wrap_continuation, "term51ORow(&s, 1).wrap_continuation")
	// No wrap on the row: the spacer head stays.
	expectEqual(t, PageCellWideSpacerHead, sc3OCell(t, s, 4, 0).wide)
	expectEqual(t, 2, s.cursor.x)
	expectEqual(t, 0, s.cursor.y)

	// Calling again does nothing more.
	s.CursorResetWrap()
	expect(t, !s.cursor.pending_wrap, "!s.cursor.pending_wrap")
}

// Zig: test "term51: cursorResetWrap: wrapped row clears wrap, next continuation and the spacer head"
func TestSc3_term51_cursorResetWrap_wrapped_row_clears_wrap_next_continuation_and_the_spacer_head(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	s := sc3OWrapped(t)
	defer s.Deinit()
	s.CursorAbsolute(1, 0)
	s.cursor.pending_wrap = true

	s.CursorResetWrap()
	expect(t, !s.cursor.pending_wrap, "!s.cursor.pending_wrap")
	expect(t, !sc3ORow(t, s, 0).wrap, "!term51ORow(&s, 0).wrap")
	expect(t, !sc3ORow(t, s, 1).wrap_continuation, "!term51ORow(&s, 1).wrap_continuation")
	// The spacer head is gone, the rest of the rows untouched.
	sc3OExpectBlank(t, sc3OCell(t, s, 4, 0))
	expectEqual(t, 'd', sc3OCp(t, s, 3, 0))
	expectEqual(t, PageCellWideWide, sc3OCell(t, s, 0, 1).wide)
	expectEqual(t, 0x1F600, sc3OCp(t, s, 0, 1))
	expectEqual(t, 1, s.cursor.x)
	expectEqual(t, 0, s.cursor.y)
	sc3OExpectCursorConsistent(t, s)

	// The flag on the row itself is not reset again by a second call.
	sc3ORow(t, s, 1).wrap_continuation = true
	s.CursorResetWrap()
	expect(t, sc3ORow(t, s, 1).wrap_continuation, "term51ORow(&s, 1).wrap_continuation")
}

// Zig: test "term51: cursorResetWrap: spacer head is cleared with the background colour"
func TestSc3_term51_cursorResetWrap_spacer_head_is_cleared_with_the_background_colour(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	s := sc3OWrapped(t)
	defer s.Deinit()
	s.CursorAbsolute(0, 0)
	sc3SetAttr(t, s, sc3AttrBgRGB(4, 5, 6))

	s.CursorResetWrap()
	cell := sc3OCell(t, s, 4, 0)
	expectEqual(t, PageCellWideNarrow, cell.wide)
	expectEqual(t, PageCellContentTagBgColorRgb, cell.content_tag)
	expectEqual(t, 4, cell.content.color_rgb.r)
	expectEqual(t, 5, cell.content.color_rgb.g)
	expectEqual(t, 6, cell.content.color_rgb.b)
	expect(t, !sc3ORow(t, s, 0).wrap, "!term51ORow(&s, 0).wrap")
}

// Zig: test "term51: cursorResetWrap: a spacer head with a hyperlink releases it"
func TestSc3_term51_cursorResetWrap_a_spacer_head_with_a_hyperlink_releases_it(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	s := sc3Init(t, 5, 3, 0)
	defer s.Deinit()
	sc3StartLink(t, s, "https://example.com/", "id")
	sc3Write(t, s, "abcd")
	// Build the spacer head by hand: the test writer sets the hyperlink
	// before the row is marked as wrapped, which trips the page integrity
	// check in debug builds.
	sc3ORow(t, s, 0).wrap = true
	sc3ORow(t, s, 1).wrap_continuation = true
	*sc3OCell(t, s, 4, 0) = sc3CellWide(0, PageCellWideSpacerHead)
	expectEqual(t, 4, s.cursor.x)
	zigTry(t, s.CursorSetHyperlink())
	page := &s.cursor.page_pin.node.data
	link_id := s.cursor.hyperlink_id
	expect(t, sc3OCell(t, s, 4, 0).hyperlink, "term51OCell(&s, 4, 0).hyperlink")
	// abcd (4) + spacer head + the cursor
	expectEqual(t, 6, sc3OLinkRefs(page, link_id))

	s.CursorAbsolute(0, 0)
	s.CursorResetWrap()
	expect(t, !sc3OCell(t, s, 4, 0).hyperlink, "!term51OCell(&s, 4, 0).hyperlink")
	sc3OExpectBlank(t, sc3OCell(t, s, 4, 0))
	expectEqual(t, 5, sc3OLinkRefs(page, link_id))
	expect(t, !sc3ORow(t, s, 0).wrap, "!term51ORow(&s, 0).wrap")
	expect(t, !sc3ORow(t, s, 1).wrap_continuation, "!term51ORow(&s, 1).wrap_continuation")
	// The other cells of the row keep theirs and the row flag stays.
	expect(t, sc3OCell(t, s, 0, 0).hyperlink, "term51OCell(&s, 0, 0).hyperlink")
	expect(t, sc3ORow(t, s, 0).hyperlink, "term51ORow(&s, 0).hyperlink")
}

// Zig: test "term51: cursorResetWrap: wrapped row whose last cell is not a spacer head"
func TestSc3_term51_cursorResetWrap_wrapped_row_whose_last_cell_is_not_a_spacer_head(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;

	// Soft wrap of plain text.
	func() {
		s := sc3Init(t, 5, 3, 0)
		defer s.Deinit()
		sc3Write(t, s, "abcdef")
		expect(t, sc3ORow(t, s, 0).wrap, "term51ORow(&s, 0).wrap")
		expect(t, sc3ORow(t, s, 1).wrap_continuation, "term51ORow(&s, 1).wrap_continuation")
		s.CursorAbsolute(0, 0)
		s.CursorResetWrap()
		expect(t, !sc3ORow(t, s, 0).wrap, "!term51ORow(&s, 0).wrap")
		expect(t, !sc3ORow(t, s, 1).wrap_continuation, "!term51ORow(&s, 1).wrap_continuation")
		expectEqual(t, 'e', sc3OCp(t, s, 4, 0))
	}()

	// A wide character ending exactly at the last column (spacer tail).
	func() {
		s := sc3Init(t, 5, 3, 0)
		defer s.Deinit()
		sc3Write(t, s, "abc\U0001F600")
		expectEqual(t, PageCellWideSpacerTail, sc3OCell(t, s, 4, 0).wide)
		sc3ORow(t, s, 0).wrap = true
		s.CursorAbsolute(0, 0)
		s.CursorResetWrap()
		expect(t, !sc3ORow(t, s, 0).wrap, "!term51ORow(&s, 0).wrap")
		expectEqual(t, PageCellWideSpacerTail, sc3OCell(t, s, 4, 0).wide)
		expectEqual(t, 0x1F600, sc3OCp(t, s, 3, 0))
	}()
}

// Zig: test "term51: cursorResetWrap: wrapped row on the last row and across a page boundary"
func TestSc3_term51_cursorResetWrap_wrapped_row_on_the_last_row_and_across_a_page_boundary(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;

	// No row below.
	func() {
		s := sc3Init(t, 5, 2, 0)
		defer s.Deinit()
		s.CursorAbsolute(0, 1)
		sc3ORow(t, s, 1).wrap = true
		s.cursor.pending_wrap = true
		s.CursorResetWrap()
		expect(t, !sc3ORow(t, s, 1).wrap, "!term51ORow(&s, 1).wrap")
		expect(t, !s.cursor.pending_wrap, "!s.cursor.pending_wrap")
		expectEqual(t, 1, s.cursor.y)
	}()

	// The next row is the first row of the next page.
	func() {
		s := sc3OTwoPages(t, 20, 3)
		defer s.Deinit()
		s.CursorAbsolute(0, 1)
		expect(t, s.cursor.page_pin.node != s.pages.pages.last, "s.cursor.page_pin.node != s.pages.pages.last.?")
		sc3ORow(t, s, 1).wrap = true
		sc3ORow(t, s, 2).wrap_continuation = true
		s.CursorResetWrap()
		expect(t, !sc3ORow(t, s, 1).wrap, "!term51ORow(&s, 1).wrap")
		expect(t, !sc3ORow(t, s, 2).wrap_continuation, "!term51ORow(&s, 2).wrap_continuation")
	}()
}

// Zig: test "term51: cursorResetWrap: does not touch the next row's own wrap flag or the cursor position"
func TestSc3_term51_cursorResetWrap_does_not_touch_the_next_row_s_own_wrap_flag_or_the_cursor_position(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	s := sc3Init(t, 5, 3, 0)
	defer s.Deinit()
	s.CursorAbsolute(3, 0)
	sc3ORow(t, s, 0).wrap = true
	sc3ORow(t, s, 0).wrap_continuation = true
	sc3ORow(t, s, 1).wrap = true
	sc3ORow(t, s, 1).wrap_continuation = true

	s.CursorResetWrap()
	expect(t, !sc3ORow(t, s, 0).wrap, "!term51ORow(&s, 0).wrap")
	// Its own continuation flag (set by the previous row) is not touched.
	expect(t, sc3ORow(t, s, 0).wrap_continuation, "term51ORow(&s, 0).wrap_continuation")
	expect(t, sc3ORow(t, s, 1).wrap, "term51ORow(&s, 1).wrap")
	expect(t, !sc3ORow(t, s, 1).wrap_continuation, "!term51ORow(&s, 1).wrap_continuation")
	expectEqual(t, 3, s.cursor.x)
	expectEqual(t, 0, s.cursor.y)
}

// Zig: test "term51: scroll: top and active move the viewport and leave the cursor"
func TestSc3_term51_scroll_top_and_active_move_the_viewport_and_leave_the_cursor(t *testing.T) {
	testStdCapacity(t)
	s := sc3QHistoryScreen(t)
	defer s.Deinit()

	expect(t, s.pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a7\na8\na9")

	sc3QClearKittyDirty(s)
	s.Scroll(ScreenScroll{Tag: ScreenScrollTop})
	expect(t, s.pages.viewport.Tag == PageListViewportTop, "s.pages.viewport == .top")
	expect(t, sc3QKittyDirty(s), "term51QKittyDirty(&s)")
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a0\na1\na2")

	// Scrolling the top again is stable.
	s.Scroll(ScreenScroll{Tag: ScreenScrollTop})
	expect(t, s.pages.viewport.Tag == PageListViewportTop, "s.pages.viewport == .top")

	sc3QClearKittyDirty(s)
	s.Scroll(ScreenScroll{Tag: ScreenScrollActive})
	expect(t, s.pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")
	expect(t, sc3QKittyDirty(s), "term51QKittyDirty(&s)")
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a7\na8\na9")

	// Active again is stable.
	s.Scroll(ScreenScroll{Tag: ScreenScrollActive})
	expect(t, s.pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")

	// The cursor is not affected by viewport scrolling.
	expectEqual(t, 2, s.cursor.x)
	expectEqual(t, 2, s.cursor.y)
}

// Zig: test "term51: scroll: row"
func TestSc3_term51_scroll_row(t *testing.T) {
	testStdCapacity(t)
	s := sc3QHistoryScreen(t)
	defer s.Deinit()

	// total_rows = 10, rows = 3. Rows at or below 7 clamp to active.
	s.Scroll(ScreenScroll{Tag: ScreenScrollRow, Row: 3})
	expect(t, s.pages.viewport.Tag == PageListViewportPin, "s.pages.viewport == .pin")
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a3\na4\na5")

	// A pinned viewport with a known offset moves by the delta.
	s.Scroll(ScreenScroll{Tag: ScreenScrollRow, Row: 6})
	expect(t, s.pages.viewport.Tag == PageListViewportPin, "s.pages.viewport == .pin")
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a6\na7\na8")

	s.Scroll(ScreenScroll{Tag: ScreenScrollRow, Row: 7})
	expect(t, s.pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a7\na8\na9")

	// Row zero is the top.
	s.Scroll(ScreenScroll{Tag: ScreenScrollRow, Row: 0})
	expect(t, s.pages.viewport.Tag == PageListViewportTop, "s.pages.viewport == .top")
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a0\na1\na2")

	// From the top, a row in the second half of the rows uses the backwards
	// traversal.
	s.Scroll(ScreenScroll{Tag: ScreenScrollRow, Row: 6})
	expect(t, s.pages.viewport.Tag == PageListViewportPin, "s.pages.viewport == .pin")
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a6\na7\na8")

	// And a row in the first half the forwards one.
	s.Scroll(ScreenScroll{Tag: ScreenScrollActive})
	s.Scroll(ScreenScroll{Tag: ScreenScrollRow, Row: 1})
	expect(t, s.pages.viewport.Tag == PageListViewportPin, "s.pages.viewport == .pin")
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a1\na2\na3")

	// Far beyond the end clamps to active.
	s.Scroll(ScreenScroll{Tag: ScreenScrollRow, Row: 1000})
	expect(t, s.pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")

	// One before the active area is still a pin.
	s.Scroll(ScreenScroll{Tag: ScreenScrollRow, Row: 6})
	expect(t, s.pages.viewport.Tag == PageListViewportPin, "s.pages.viewport == .pin")
	s.Scroll(ScreenScroll{Tag: ScreenScrollRow, Row: ^uint(0)})
	expect(t, s.pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")

	expectEqual(t, 2, s.cursor.y)
}

// Zig: test "term51: scroll: delta_row"
func TestSc3_term51_scroll_delta_row(t *testing.T) {
	testStdCapacity(t)
	s := sc3QHistoryScreen(t)
	defer s.Deinit()
	deltaRow := func(n int64) ScreenScroll {
		return ScreenScroll{Tag: ScreenScrollDeltaRow, DeltaRow: n}
	}

	// Scrolling down or not at all from active stays active.
	s.Scroll(deltaRow(0))
	expect(t, s.pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")
	s.Scroll(deltaRow(1))
	expect(t, s.pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")
	s.Scroll(deltaRow(100))
	expect(t, s.pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")

	s.Scroll(deltaRow(-2))
	expect(t, s.pages.viewport.Tag == PageListViewportPin, "s.pages.viewport == .pin")
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a5\na6\na7")

	// Pinned: fast paths in both directions.
	s.Scroll(deltaRow(0))
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a5\na6\na7")
	s.Scroll(deltaRow(1))
	expect(t, s.pages.viewport.Tag == PageListViewportPin, "s.pages.viewport == .pin")
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a6\na7\na8")
	s.Scroll(deltaRow(-3))
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a3\na4\na5")

	// Moving down into the first active row makes the viewport active.
	s.Scroll(deltaRow(4))
	expect(t, s.pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a7\na8\na9")

	// Overflowing up from a pin is the top.
	s.Scroll(deltaRow(-1))
	expect(t, s.pages.viewport.Tag == PageListViewportPin, "s.pages.viewport == .pin")
	s.Scroll(deltaRow(-100))
	expect(t, s.pages.viewport.Tag == PageListViewportTop, "s.pages.viewport == .top")
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a0\na1\na2")

	// Scrolling up from the top does nothing.
	s.Scroll(deltaRow(0))
	expect(t, s.pages.viewport.Tag == PageListViewportTop, "s.pages.viewport == .top")
	s.Scroll(deltaRow(-1))
	expect(t, s.pages.viewport.Tag == PageListViewportTop, "s.pages.viewport == .top")

	// Scrolling down from the top.
	s.Scroll(deltaRow(2))
	expect(t, s.pages.viewport.Tag == PageListViewportPin, "s.pages.viewport == .pin")
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a2\na3\na4")

	// Overflowing down from a pin is active.
	s.Scroll(deltaRow(100))
	expect(t, s.pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")

	// Overflowing up from active (slow path) lands on the top.
	s.Scroll(deltaRow(-100))
	expect(t, s.pages.viewport.Tag == PageListViewportTop, "s.pages.viewport == .top")

	// Exactly to the top from active.
	s.Scroll(ScreenScroll{Tag: ScreenScrollActive})
	s.Scroll(deltaRow(-7))
	expect(t, s.pages.viewport.Tag == PageListViewportTop, "s.pages.viewport == .top")

	// From the top, large forward delta goes through the slow path and
	// ends in the active area.
	s.Scroll(deltaRow(7))
	expect(t, s.pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")
}

// Zig: test "term51: scroll: pin"
func TestSc3_term51_scroll_pin(t *testing.T) {
	testStdCapacity(t)
	s := sc3QHistoryScreen(t)
	defer s.Deinit()
	scrollPin := func(x SizeCellCountInt, y uint32) ScreenScroll {
		return ScreenScroll{Tag: ScreenScrollPin, Pin: zigUnwrap(t, s.pages.Pin(sc3Screen(x, y)))}
	}

	// A pin in the history.
	s.Scroll(scrollPin(0, 4))
	expect(t, s.pages.viewport.Tag == PageListViewportPin, "s.pages.viewport == .pin")
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a4\na5\na6")

	// PageList documents that the x of the pin is ignored, but the pin is
	// stored as given: the viewport pin keeps x=3 and the viewport dump
	// starts at that column of the first row (so "a2" is cut off).
	s.Scroll(scrollPin(3, 2))
	expect(t, s.pages.viewport.Tag == PageListViewportPin, "s.pages.viewport == .pin")
	expectEqual(t, 3, s.pages.viewport_pin.x)
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "\na3\na4")

	// The pin that is the first row of the active area is active.
	s.Scroll(scrollPin(0, 7))
	expect(t, s.pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")

	// A pin inside the active area is active.
	s.Scroll(scrollPin(0, 9))
	expect(t, s.pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")

	// One row before active is a pin.
	s.Scroll(scrollPin(0, 6))
	expect(t, s.pages.viewport.Tag == PageListViewportPin, "s.pages.viewport == .pin")
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a6\na7\na8")

	// The first row is the top.
	s.Scroll(scrollPin(0, 0))
	expect(t, s.pages.viewport.Tag == PageListViewportTop, "s.pages.viewport == .top")

	// One row below the top is a pin.
	s.Scroll(scrollPin(0, 1))
	expect(t, s.pages.viewport.Tag == PageListViewportPin, "s.pages.viewport == .pin")
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a1\na2\na3")
}

// Zig: test "term51: scroll: delta_prompt"
func TestSc3_term51_scroll_delta_prompt(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 1000)
	defer s.Deinit()
	deltaPrompt := func(n int64) ScreenScroll {
		return ScreenScroll{Tag: ScreenScrollDeltaPrompt, DeltaPrompt: n}
	}

	// Prompts on a2, a5 and a8 (a8 is in the active area).
	rows := []string{"a0", "a1", "a2", "a3", "a4", "a5", "a6", "a7", "a8", "a9"}
	for i, text := range rows {
		if i == 2 || i == 5 || i == 8 {
			s.CursorSetSemanticContent(ScreenCursorSetSemanticContent{
				Tag:    ScreenCursorSetSemanticContentTagPrompt,
				Prompt: OSCParsersSemanticPromptPromptKindInitial,
			})
		}
		sc3Write(t, s, text)
		s.CursorSetSemanticContent(ScreenCursorSetSemanticContent{Tag: ScreenCursorSetSemanticContentTagOutput})
		if i+1 < len(rows) {
			sc3Write(t, s, "\n")
		}
	}
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a7\na8\na9")

	// Zero is a no-op.
	s.Scroll(deltaPrompt(0))
	expect(t, s.pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")

	// Back one prompt.
	s.Scroll(deltaPrompt(-1))
	expect(t, s.pages.viewport.Tag == PageListViewportPin, "s.pages.viewport == .pin")
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a5\na6\na7")

	// Back more prompts than exist goes to the furthest.
	s.Scroll(deltaPrompt(-5))
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a2\na3\na4")

	// No more prompts further back: unchanged.
	s.Scroll(deltaPrompt(-1))
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a2\na3\na4")

	// Forward one prompt.
	s.Scroll(deltaPrompt(1))
	expect(t, s.pages.viewport.Tag == PageListViewportPin, "s.pages.viewport == .pin")
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a5\na6\na7")

	// Forward onto a prompt inside the active area is active.
	s.Scroll(deltaPrompt(1))
	expect(t, s.pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")

	// Forward from the top by two prompts.
	s.Scroll(ScreenScroll{Tag: ScreenScrollTop})
	s.Scroll(deltaPrompt(2))
	expect(t, s.pages.viewport.Tag == PageListViewportPin, "s.pages.viewport == .pin")
	sc3QExpectDump(t, s, sc3Viewport(0, 0), "a5\na6\na7")

	// Forward by many prompts ends in active.
	s.Scroll(deltaPrompt(100))
	expect(t, s.pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")
}

// Zig: test "term51: scroll: a screen without scrollback never scrolls"
func TestSc3_term51_scroll_a_screen_without_scrollback_never_scrolls(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 0)
	defer s.Deinit()
	sc3Write(t, s, "a0\na1\na2\na3\na4")

	s.Scroll(ScreenScroll{Tag: ScreenScrollTop})
	expect(t, s.pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")
	s.Scroll(ScreenScroll{Tag: ScreenScrollRow, Row: 0})
	expect(t, s.pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")
	s.Scroll(ScreenScroll{Tag: ScreenScrollDeltaRow, DeltaRow: -1})
	expect(t, s.pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")
	s.Scroll(ScreenScroll{Tag: ScreenScrollDeltaPrompt, DeltaPrompt: -1})
	expect(t, s.pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")
	s.Scroll(ScreenScroll{Tag: ScreenScrollPin, Pin: zigUnwrap(t, s.pages.Pin(sc3Screen(0, 0)))})
	expect(t, s.pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")
	expect(t, s.ViewportIsBottom(), "s.viewportIsBottom()")
}

// Zig: test "term51: viewportIsBottom: follows the viewport kind"
func TestSc3_term51_viewportIsBottom_follows_the_viewport_kind(t *testing.T) {
	testStdCapacity(t)
	s := sc3QHistoryScreen(t)
	defer s.Deinit()

	expect(t, s.ViewportIsBottom(), "s.viewportIsBottom()")
	s.Scroll(ScreenScroll{Tag: ScreenScrollTop})
	expect(t, !s.ViewportIsBottom(), "!s.viewportIsBottom()")
	s.Scroll(ScreenScroll{Tag: ScreenScrollActive})
	expect(t, s.ViewportIsBottom(), "s.viewportIsBottom()")
	s.Scroll(ScreenScroll{Tag: ScreenScrollRow, Row: 3})
	expect(t, !s.ViewportIsBottom(), "!s.viewportIsBottom()")
	s.Scroll(ScreenScroll{Tag: ScreenScrollDeltaRow, DeltaRow: 1})
	expect(t, !s.ViewportIsBottom(), "!s.viewportIsBottom()")
	s.Scroll(ScreenScroll{Tag: ScreenScrollDeltaRow, DeltaRow: 100})
	expect(t, s.ViewportIsBottom(), "s.viewportIsBottom()")

	// A screen with no history is always at the bottom.
	e := sc3Init(t, 5, 3, 1000)
	defer e.Deinit()
	expect(t, e.ViewportIsBottom(), "e.viewportIsBottom()")
	e.Scroll(ScreenScroll{Tag: ScreenScrollTop})
	expect(t, !e.ViewportIsBottom(), "!e.viewportIsBottom()")
	// The by-value receiver reads the same. (Zig passes the screen by value;
	// Go's method takes a pointer, so the copy is a copy of the struct.)
	copy := *e
	expectEqual(t, e.ViewportIsBottom(), copy.ViewportIsBottom())
}

// Zig: test "term51: scrollClear: moves content to history and resets the cursor into the active area"
func TestSc3_term51_scrollClear_moves_content_to_history_and_resets_the_cursor_into_the_active_area(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 1000)
	defer s.Deinit()
	sc3Write(t, s, "ab\ncd")
	expectEqual(t, 2, s.cursor.x)
	expectEqual(t, 1, s.cursor.y)
	expectEqual(t, 3, s.pages.total_rows)

	sc3QClearKittyDirty(s)
	zigTry(t, s.ScrollClear())
	expect(t, sc3QKittyDirty(s), "term51QKittyDirty(&s)")

	// Two rows were scrolled (the third row was empty).
	expectEqual(t, 5, s.pages.total_rows)
	sc3QExpectDump(t, s, sc3Active(0, 0), "")
	sc3QExpectDump(t, s, sc3Screen(0, 0), "ab\ncd")
	sc3QExpectDump(t, s, sc3History(0, 0), "ab\ncd")

	// The cursor row was pushed into history, so the cursor was reset to
	// the top-left of the active area.
	expectEqual(t, 0, s.cursor.x)
	expectEqual(t, 0, s.cursor.y)
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: scrollClear: cursor row below the content keeps its position relative to the screen"
func TestSc3_term51_scrollClear_cursor_row_below_the_content_keeps_its_position_relative_to_the_screen(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 1000)
	defer s.Deinit()
	sc3Write(t, s, "ab")
	s.CursorAbsolute(3, 2)

	// Only row 0 is non-empty: non_empty = 3 - 2 = 1 row is scrolled.
	zigTry(t, s.ScrollClear())
	expectEqual(t, 4, s.pages.total_rows)
	sc3QExpectDump(t, s, sc3History(0, 0), "ab")
	sc3QExpectDump(t, s, sc3Active(0, 0), "")

	// The tracked cursor pin follows its row, which is now at active y=1.
	expectEqual(t, 3, s.cursor.x)
	expectEqual(t, 1, s.cursor.y)
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: scrollClear: nothing to scroll for an empty active area"
func TestSc3_term51_scrollClear_nothing_to_scroll_for_an_empty_active_area(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 1000)
	defer s.Deinit()

	s.CursorAbsolute(3, 1)
	zigTry(t, s.ScrollClear())
	expectEqual(t, 3, s.pages.total_rows)
	expectEqual(t, 3, s.cursor.x)
	expectEqual(t, 1, s.cursor.y)

	// History but an empty active area also scrolls nothing.
	h := sc3QHistoryScreen(t)
	defer h.Deinit()
	h.ClearRows(sc3Active(0, 0), None[Point](), false)
	total := h.pages.total_rows
	zigTry(t, h.ScrollClear())
	expectEqual(t, total, h.pages.total_rows)
	sc3QExpectDump(t, h, sc3Active(0, 0), "")
}

// Zig: test "term51: scrollClear: a full active area scrolls every row"
func TestSc3_term51_scrollClear_a_full_active_area_scrolls_every_row(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 1000)
	defer s.Deinit()
	sc3Write(t, s, "a\nb\nc")

	zigTry(t, s.ScrollClear())
	expectEqual(t, 6, s.pages.total_rows)
	sc3QExpectDump(t, s, sc3Active(0, 0), "")
	sc3QExpectDump(t, s, sc3History(0, 0), "a\nb\nc")
	expectEqual(t, 0, s.cursor.y)
	expectEqual(t, 0, s.cursor.x)

	// A second clear has nothing left to scroll.
	zigTry(t, s.ScrollClear())
	expectEqual(t, 6, s.pages.total_rows)
}

// Zig: test "term51: scrollClear: a single non-empty cell in the last row"
func TestSc3_term51_scrollClear_a_single_non_empty_cell_in_the_last_row(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 1000)
	defer s.Deinit()
	s.CursorAbsolute(0, 2)
	sc3Write(t, s, "z")

	zigTry(t, s.ScrollClear())
	expectEqual(t, 6, s.pages.total_rows)
	sc3QExpectDump(t, s, sc3Active(0, 0), "")
	sc3QExpectDump(t, s, sc3History(0, 0), "\n\nz")
}

// Zig: test "term51: scrollClear: does not touch the viewport and keeps the cursor style"
func TestSc3_term51_scrollClear_does_not_touch_the_viewport_and_keeps_the_cursor_style(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 1000)
	defer s.Deinit()
	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	sc3Write(t, s, "ab\ncd")
	page := &s.cursor.page_pin.node.data
	expectEqual(t, 1, page.styles.Count())
	// 4 cells + the cursor.
	expectEqual(t, 5, page.styles.RefCount(s.cursor.style_id))

	s.Scroll(ScreenScroll{Tag: ScreenScrollTop})
	expect(t, s.pages.viewport.Tag == PageListViewportTop, "s.pages.viewport == .top")
	zigTry(t, s.ScrollClear())
	expect(t, s.pages.viewport.Tag == PageListViewportTop, "s.pages.viewport == .top")

	expect(t, s.cursor.style.flags.bold, "s.cursor.style.flags.bold")
	expect(t, s.cursor.style_id != StyleDefaultId, "s.cursor.style_id != style.default_id")
	page2 := &s.cursor.page_pin.node.data
	expectEqual(t, 1, page2.styles.Count())
	expectEqual(t, 5, page2.styles.RefCount(s.cursor.style_id))
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: scrollClear: screen without scrollback"
func TestSc3_term51_scrollClear_screen_without_scrollback(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 0)
	defer s.Deinit()
	sc3Write(t, s, "ab\ncd")

	zigTry(t, s.ScrollClear())
	sc3QExpectDump(t, s, sc3Active(0, 0), "")
	expectEqual(t, 0, s.cursor.y)
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: eraseRows: a bounded range of history"
func TestSc3_term51_eraseRows_a_bounded_range_of_history(t *testing.T) {
	testStdCapacity(t)
	s := sc3QHistoryScreen(t)
	defer s.Deinit()

	// a0, a1 and a2 are removed (the bottom point is inclusive).
	s.EraseRows(sc3History(0, 0), Some(sc3History(0, 2)))
	expectEqual(t, 7, s.pages.total_rows)
	sc3QExpectDump(t, s, sc3Screen(0, 0), "a3\na4\na5\na6\na7\na8\na9")
	sc3QExpectDump(t, s, sc3Active(0, 0), "a7\na8\na9")

	// The cursor stays on the same content.
	expectEqual(t, 2, s.cursor.x)
	expectEqual(t, 2, s.cursor.y)
	sc3QExpectExactRefs(t, s)

	// Erasing a single history row.
	s.EraseRows(sc3History(0, 0), Some(sc3History(0, 0)))
	expectEqual(t, 6, s.pages.total_rows)
	sc3QExpectDump(t, s, sc3Screen(0, 0), "a4\na5\na6\na7\na8\na9")

	// All remaining history.
	s.EraseRows(sc3History(0, 0), None[Point]())
	expectEqual(t, 3, s.pages.total_rows)
	sc3QExpectDump(t, s, sc3Screen(0, 0), "a7\na8\na9")
	expectEqual(t, 2, s.cursor.y)

	// Nothing left in history: erasing it again is not possible, but the
	// active area and cursor are untouched by the previous call.
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: eraseRows: releases styles, hyperlinks and graphemes of erased rows"
func TestSc3_term51_eraseRows_releases_styles_hyperlinks_and_graphemes_of_erased_rows(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 1000)
	defer s.Deinit()

	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	sc3StartLink(t, s, "https://example.com", "id")
	sc3Write(t, s, "x\nx\nx\nx\nx\nx")

	// One grapheme in the history (screen y=1) and one in the active area.
	for _, y := range []uint32{1, 4} {
		c := zigUnwrap(t, s.pages.GetCell(sc3Screen(0, y)))
		zigTry(t, c.node.data.AppendGrapheme(c.row, c.cell, 0x301))
	}

	{
		page := &s.cursor.page_pin.node.data
		// 6 cells + the cursor.
		expectEqual(t, 7, page.styles.RefCount(s.cursor.style_id))
		expectEqual(t, 7, page.hyperlink_set.RefCount(s.cursor.hyperlink_id))
		expectEqual(t, 2, page.GraphemeCount())
		expectEqual(t, 6, page.HyperlinkCount())
	}
	sc3QExpectExactRefs(t, s)

	s.EraseRows(sc3History(0, 0), None[Point]())

	page := &s.cursor.page_pin.node.data
	expectEqual(t, 3, s.pages.total_rows)
	expectEqual(t, 1, page.styles.Count())
	expectEqual(t, 4, page.styles.RefCount(s.cursor.style_id))
	expectEqual(t, 1, page.hyperlink_set.Count())
	expectEqual(t, 4, page.hyperlink_set.RefCount(s.cursor.hyperlink_id))
	expectEqual(t, 1, page.GraphemeCount())
	expectEqual(t, 3, page.HyperlinkCount())
	expectEqual(t, 1, s.cursor.x)
	expectEqual(t, 2, s.cursor.y)
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: eraseRows: active rows above the cursor shift it up"
func TestSc3_term51_eraseRows_active_rows_above_the_cursor_shift_it_up(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 5, 0)
	defer s.Deinit()
	sc3Write(t, s, "1\n2\n3")
	expectEqual(t, 1, s.cursor.x)
	expectEqual(t, 2, s.cursor.y)

	s.EraseRows(sc3Active(0, 0), Some(sc3Active(0, 1)))
	sc3QExpectDump(t, s, sc3Active(0, 0), "3")
	// The active area is regrown to its full size.
	expectEqual(t, 5, s.pages.total_rows)
	// The cursor stayed on row "3" (now at y=0), x is preserved.
	expectEqual(t, 1, s.cursor.x)
	expectEqual(t, 0, s.cursor.y)
	rac := s.cursor.page_pin.RowAndCell()
	expect(t, s.cursor.page_row == rac.row, "s.cursor.page_row == rac.row")
	expect(t, s.cursor.page_cell == rac.cell, "s.cursor.page_cell == rac.cell")
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: eraseRows: erasing the cursor row moves the cursor to the top-left"
func TestSc3_term51_eraseRows_erasing_the_cursor_row_moves_the_cursor_to_the_top_left(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 5, 0)
	defer s.Deinit()
	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	sc3Write(t, s, "1\n2\n3\n4")
	s.CursorAbsolute(3, 1)
	{
		page := &s.cursor.page_pin.node.data
		// 4 cells + the cursor.
		expectEqual(t, 5, page.styles.RefCount(s.cursor.style_id))
	}

	// Erase rows 0..1, the cursor is on row 1.
	s.EraseRows(sc3Active(0, 0), Some(sc3Active(0, 1)))
	sc3QExpectDump(t, s, sc3Active(0, 0), "3\n4")
	expectEqual(t, 0, s.cursor.x)
	expectEqual(t, 0, s.cursor.y)

	// The erased cells released their styles; the cursor kept its own.
	page := &s.cursor.page_pin.node.data
	expect(t, s.cursor.style.flags.bold, "s.cursor.style.flags.bold")
	expectEqual(t, 3, page.styles.RefCount(s.cursor.style_id))
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: eraseRows: the cursor pin is kept in place when all of the page is erased"
func TestSc3_term51_eraseRows_the_cursor_pin_is_kept_in_place_when_all_of_the_page_is_erased(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 5, 0)
	defer s.Deinit()
	sc3Write(t, s, "1\n2\n3")
	// The cursor must be on the first row: PageList regrows the active area
	// one row at a time after reinitializing the page and a tracked pin
	// below the first row is invalid in between (integrity panic).
	s.CursorAbsolute(1, 0)

	// Erasing the whole (only) page reinitializes it rather than erasing
	// the nodes, so the tracked pin is not moved.
	s.EraseRows(sc3Active(0, 0), None[Point]())
	expectEqual(t, 5, s.pages.total_rows)
	sc3QExpectDump(t, s, sc3Screen(0, 0), "")
	expectEqual(t, 1, s.cursor.x)
	expectEqual(t, 0, s.cursor.y)
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: clearRows: unprotected clears the cells and resets every row attribute"
func TestSc3_term51_clearRows_unprotected_clears_the_cells_and_resets_every_row_attribute(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 0)
	defer s.Deinit()

	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	sc3StartLink(t, s, "https://example.com", "id")
	sc3Write(t, s, "abcde")
	s.EndHyperlink()
	sc3SetAttr(t, s, sc3A(SGRAttributeTagUnset))
	// A grapheme on the first cell.
	{
		row := sc3QRow(t, s, 0)
		zigTry(t, s.cursor.page_pin.node.data.AppendGrapheme(row, sc3QCell(t, s, 0, 0), 0x301))
		row.wrap = true
		row.wrap_continuation = true
		row.semantic_prompt = PageRowSemanticPromptPrompt
		row.kitty_virtual_placeholder = false
	}
	// The next row is untouched content.
	sc3QMove(s, 0, 1)
	sc3Write(t, s, "xyz")

	page := &s.cursor.page_pin.node.data
	expectEqual(t, 1, page.styles.Count())
	expectEqual(t, 1, page.hyperlink_set.Count())
	expectEqual(t, 1, page.GraphemeCount())
	expectEqual(t, 5, page.HyperlinkCount())

	s.pages.ClearDirty()
	s.ClearRows(sc3Active(0, 0), Some(sc3Active(0, 0)), false)

	// Row 0 is blank and fully reset, except dirty is set.
	row0 := sc3QRow(t, s, 0)
	expect(t, row0.dirty, "row0.dirty")
	expect(t, !row0.wrap, "!row0.wrap")
	expect(t, !row0.wrap_continuation, "!row0.wrap_continuation")
	expect(t, !row0.grapheme, "!row0.grapheme")
	expect(t, !row0.styled, "!row0.styled")
	expect(t, !row0.hyperlink, "!row0.hyperlink")
	expectEqual(t, PageRowSemanticPromptNone, row0.semantic_prompt)
	for x := 0; x < 5; x++ {
		expectEqual(t, uint64(0), sc3QBits(*sc3QCell(t, s, SizeCellCountInt(x), 0)))
	}

	// Row 1 is untouched and not marked dirty.
	expect(t, !sc3QRow(t, s, 1).dirty, "!term51QRow(&s, 1).dirty")
	expectEqual(t, 'x', sc3QCell(t, s, 0, 1).content.codepoint)

	// All the managed memory of row 0 was released.
	expectEqual(t, 0, page.GraphemeCount())
	expectEqual(t, 0, page.HyperlinkCount())
	expectEqual(t, 0, page.hyperlink_set.Count())
	expectEqual(t, 0, page.styles.Count())
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: clearRows: protected keeps protected cells and every row attribute"
func TestSc3_term51_clearRows_protected_keeps_protected_cells_and_every_row_attribute(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 0)
	defer s.Deinit()

	sc3SetAttr(t, s, sc3A(SGRAttributeTagItalic))
	sc3Write(t, s, "abcde")
	sc3SetAttr(t, s, sc3A(SGRAttributeTagUnset))
	{
		row := sc3QRow(t, s, 0)
		row.wrap = true
		row.wrap_continuation = true
		row.semantic_prompt = PageRowSemanticPromptPromptContinuation
		// Cells 1 and 3 are protected.
		sc3QCell(t, s, 1, 0).protected = true
		sc3QCell(t, s, 3, 0).protected = true
	}

	page := &s.cursor.page_pin.node.data
	expectEqual(t, 1, page.styles.Count())
	s.pages.ClearDirty()
	s.ClearRows(sc3Active(0, 0), Some(sc3Active(0, 0)), true)

	row0 := sc3QRow(t, s, 0)
	expect(t, row0.dirty, "row0.dirty")
	expect(t, row0.wrap, "row0.wrap")
	expect(t, row0.wrap_continuation, "row0.wrap_continuation")
	expect(t, row0.styled, "row0.styled")
	expectEqual(t, PageRowSemanticPromptPromptContinuation, row0.semantic_prompt)

	expectEqual(t, uint64(0), sc3QBits(*sc3QCell(t, s, 0, 0)))
	expectEqual(t, 'b', sc3QCell(t, s, 1, 0).content.codepoint)
	expect(t, sc3QCell(t, s, 1, 0).protected, "term51QCell(&s, 1, 0).protected")
	expectEqual(t, uint64(0), sc3QBits(*sc3QCell(t, s, 2, 0)))
	expectEqual(t, 'd', sc3QCell(t, s, 3, 0).content.codepoint)
	expectEqual(t, uint64(0), sc3QBits(*sc3QCell(t, s, 4, 0)))

	// The two protected cells keep their style references.
	expectEqual(t, 1, page.styles.Count())
	expectEqual(t, 2, page.styles.RefCount(sc3QCell(t, s, 1, 0).style_id))
	sc3QExpectExactRefs(t, s)

	// Clearing protected again changes nothing; clearing unprotected
	// removes the rest.
	s.ClearRows(sc3Active(0, 0), Some(sc3Active(0, 0)), true)
	expectEqual(t, 2, page.styles.RefCount(sc3QCell(t, s, 1, 0).style_id))
	s.ClearRows(sc3Active(0, 0), Some(sc3Active(0, 0)), false)
	expectEqual(t, 0, page.styles.Count())
	expect(t, !row0.wrap, "!row0.wrap")
	expect(t, !row0.styled, "!row0.styled")
	expect(t, !sc3QCell(t, s, 1, 0).protected, "!term51QCell(&s, 1, 0).protected")
}

// Zig: test "term51: clearRows: bounds select the rows that are cleared"
func TestSc3_term51_clearRows_bounds_select_the_rows_that_are_cleared(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 4, 0)
	defer s.Deinit()
	sc3Write(t, s, "r0\nr1\nr2\nr3")

	// Only the middle rows.
	s.pages.ClearDirty()
	s.ClearRows(sc3Active(0, 1), Some(sc3Active(0, 2)), false)
	sc3QExpectDump(t, s, sc3Active(0, 0), "r0\n\n\nr3")
	expect(t, !sc3QRow(t, s, 0).dirty, "!term51QRow(&s, 0).dirty")
	expect(t, sc3QRow(t, s, 1).dirty, "term51QRow(&s, 1).dirty")
	expect(t, sc3QRow(t, s, 2).dirty, "term51QRow(&s, 2).dirty")
	expect(t, !sc3QRow(t, s, 3).dirty, "!term51QRow(&s, 3).dirty")

	// A null bottom clears to the end of the active area.
	s.pages.ClearDirty()
	s.ClearRows(sc3Active(0, 3), None[Point](), false)
	sc3QExpectDump(t, s, sc3Active(0, 0), "r0")
	expect(t, sc3QRow(t, s, 3).dirty, "term51QRow(&s, 3).dirty")
	expect(t, !sc3QRow(t, s, 2).dirty, "!term51QRow(&s, 2).dirty")

	// A single row where tl == bl.
	s.pages.ClearDirty()
	s.ClearRows(sc3Active(0, 0), Some(sc3Active(0, 0)), false)
	sc3QExpectDump(t, s, sc3Active(0, 0), "")
	expect(t, sc3QRow(t, s, 0).dirty, "term51QRow(&s, 0).dirty")
	expect(t, !sc3QRow(t, s, 1).dirty, "!term51QRow(&s, 1).dirty")
}

// Zig: test "term51: clearRows: history rows"
func TestSc3_term51_clearRows_history_rows(t *testing.T) {
	testStdCapacity(t)
	s := sc3QHistoryScreen(t)
	defer s.Deinit()

	s.ClearRows(sc3History(0, 0), Some(sc3History(0, 1)), false)
	// The rows are blanked but still exist.
	expectEqual(t, 10, s.pages.total_rows)
	sc3QExpectDump(t, s, sc3Screen(0, 0), "\n\na2\na3\na4\na5\na6\na7\na8\na9")
	expect(t, zigUnwrap(t, s.pages.GetCell(sc3Screen(0, 0))).row.dirty, "s.pages.getCell(.{ .screen = .{ .x = 0, .y = 0 } }).?.row.dirty")
	expectEqual(t, uint64(0), sc3QBits(*zigUnwrap(t, s.pages.GetCell(sc3Screen(0, 1))).cell))
}

// Zig: test "term51: clearRows: cleared cells get the cursor background"
func TestSc3_term51_clearRows_cleared_cells_get_the_cursor_background(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 4, 2, 0)
	defer s.Deinit()
	sc3Write(t, s, "abcd")
	sc3QCell(t, s, 1, 0).protected = true

	sc3SetAttr(t, s, sc3AttrBg256(9))
	s.ClearRows(sc3Active(0, 0), Some(sc3Active(0, 0)), true)

	// Unprotected cells are bg-only cells, the protected cell is intact.
	for _, x := range []SizeCellCountInt{0, 2, 3} {
		c := sc3QCell(t, s, x, 0)
		expectEqual(t, PageCellContentTagBgColorPalette, c.content_tag)
		expectEqual(t, 9, c.content.color_palette)
		expectEqual(t, StyleDefaultId, c.style_id)
	}
	expectEqual(t, 'b', sc3QCell(t, s, 1, 0).content.codepoint)

	// Unprotected clears everything including the protected cell.
	sc3SetAttr(t, s, sc3AttrBgRGB(1, 2, 3))
	s.ClearRows(sc3Active(0, 0), Some(sc3Active(0, 0)), false)
	for x := 0; x < 4; x++ {
		c := sc3QCell(t, s, SizeCellCountInt(x), 0)
		expectEqual(t, PageCellContentTagBgColorRgb, c.content_tag)
		expectEqual(t, 1, c.content.color_rgb.r)
		expectEqual(t, 2, c.content.color_rgb.g)
		expectEqual(t, 3, c.content.color_rgb.b)
	}
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: clearRows: across a page boundary"
func TestSc3_term51_clearRows_across_a_page_boundary(t *testing.T) {
	testStdCapacity(t)
	s := sc3QCrossPageScreen(t, 6)
	defer s.Deinit()

	// Rows 1 (first page) and 2 (second page) hold styled graphemes.
	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	for _, y := range []uint32{0, 1, 2} {
		s.CursorAbsolute(0, SizeCellCountInt(y))
		*s.cursor.page_cell = PageCell{
			content_tag: PageCellContentTagCodepoint,
			content:     PageCellContent{codepoint: 'x'},
			style_id:    s.cursor.style_id,
		}
		s.cursor.page_cell.style_id = s.cursor.style_id
		s.cursor.page_pin.node.data.styles.Use(s.cursor.style_id)
		s.cursor.page_row.styled = true
	}
	s.CursorAbsolute(0, 2)
	first := s.pages.pages.first
	last := s.pages.pages.last
	expect(t, s.cursor.page_pin.node == last, "s.cursor.page_pin.node == last")
	expectEqual(t, 1, last.data.styles.Count())
	expectEqual(t, 1, first.data.styles.Count())
	sc3QExpectExactRefs(t, s)

	s.ClearRows(sc3Active(0, 1), None[Point](), false)

	expectEqual(t, 'x', sc3QCell(t, s, 0, 0).content.codepoint)
	expectEqual(t, uint64(0), sc3QBits(*sc3QCell(t, s, 0, 1)))
	expectEqual(t, uint64(0), sc3QBits(*sc3QCell(t, s, 0, 2)))
	expect(t, sc3QRow(t, s, 1).dirty, "term51QRow(&s, 1).dirty")
	expect(t, sc3QRow(t, s, 2).dirty, "term51QRow(&s, 2).dirty")
	// Only the cursor's reference remains on the last page.
	expectEqual(t, 1, last.data.styles.RefCount(s.cursor.style_id))
	expectEqual(t, 1, first.data.styles.RefCount(1))
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: blankCell: default and non-background styles are the zero cell"
func TestSc3_term51_blankCell_default_and_non_background_styles_are_the_zero_cell(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 4, 2, 0)
	defer s.Deinit()

	expectEqual(t, uint64(0), sc3QBits(s.BlankCell()))

	// A style without a background colour.
	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	sc3SetAttr(t, s, sc3AttrFgRGB(1, 2, 3))
	sc3SetAttr(t, s, SGRAttribute{Tag: SGRAttributeTagUnderlineColor, UnderlineColor: ColorRGB{r: 4, g: 5, b: 6}})
	sc3SetAttr(t, s, sc3AttrUnderline(SGRAttributeUnderlineCurly))
	expect(t, s.cursor.style_id != StyleDefaultId, "s.cursor.style_id != style.default_id")
	expectEqual(t, uint64(0), sc3QBits(s.BlankCell()))

	// Resetting to default.
	sc3SetAttr(t, s, sc3A(SGRAttributeTagUnset))
	expectEqual(t, uint64(0), sc3QBits(s.BlankCell()))
}

// Zig: test "term51: blankCell: palette and rgb backgrounds"
func TestSc3_term51_blankCell_palette_and_rgb_backgrounds(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 4, 2, 0)
	defer s.Deinit()

	for _, idx := range []uint8{0, 1, 127, 255} {
		sc3SetAttr(t, s, sc3AttrBg256(idx))
		c := s.BlankCell()
		expectEqual(t, PageCellContentTagBgColorPalette, c.content_tag)
		expectEqual(t, idx, c.content.color_palette)
		expectEqual(t, StyleDefaultId, c.style_id)
		expectEqual(t, PageCellWideNarrow, c.wide)
		expect(t, !c.protected, "!c.protected")
		expect(t, !c.hyperlink, "!c.hyperlink")
	}

	rgbs := []SGRAttribute{
		sc3AttrBgRGB(0, 0, 0),
		sc3AttrBgRGB(255, 255, 255),
		sc3AttrBgRGB(1, 2, 3),
	}
	for _, attr := range rgbs {
		sc3SetAttr(t, s, attr)
		c := s.BlankCell()
		expectEqual(t, PageCellContentTagBgColorRgb, c.content_tag)
		expectEqual(t, attr.DirectColorBg.r, c.content.color_rgb.r)
		expectEqual(t, attr.DirectColorBg.g, c.content.color_rgb.g)
		expectEqual(t, attr.DirectColorBg.b, c.content.color_rgb.b)
	}

	// A named background.
	sc3SetAttr(t, s, SGRAttribute{Tag: SGRAttributeTag8Bg, X8Bg: ColorNameRed})
	expectEqual(t, 1, s.BlankCell().content.color_palette)

	// Resetting the background goes back to the zero cell, the style stays.
	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	sc3SetAttr(t, s, sc3A(SGRAttributeTagResetBg))
	expectEqual(t, uint64(0), sc3QBits(s.BlankCell()))
}

// Zig: test "term51: blankCell: keyed on the style id, not on the style value"
func TestSc3_term51_blankCell_keyed_on_the_style_id_not_on_the_style_value(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 4, 2, 0)
	defer s.Deinit()

	// The id is the default while the style value has a background: the
	// style is not consulted.
	s.cursor.style = Style{bg_color: StyleColor{Tag: StyleColorPalette, Palette: 3}}
	expectEqual(t, StyleDefaultId, s.cursor.style_id)
	expectEqual(t, uint64(0), sc3QBits(s.BlankCell()))

	// Restore a consistent state.
	s.cursor.style = Style{}

	// A non-default id with a default style value gives no background.
	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	saved := s.cursor.style
	s.cursor.style = Style{}
	expect(t, s.cursor.style_id != StyleDefaultId, "s.cursor.style_id != style.default_id")
	expectEqual(t, uint64(0), sc3QBits(s.BlankCell()))
	s.cursor.style = saved
}

// Zig: test "term51: clearCells: a plain row clears only the given range and nothing else"
func TestSc3_term51_clearCells_a_plain_row_clears_only_the_given_range_and_nothing_else(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 2, 0)
	defer s.Deinit()
	sc3Write(t, s, "abcde")

	page := &s.cursor.page_pin.node.data
	row := s.cursor.page_row
	row.wrap = true
	row.wrap_continuation = true
	row.semantic_prompt = PageRowSemanticPromptPrompt
	row.dirty = false
	cells := page.GetCells(row)

	// A partial range.
	s.ClearCells(page, row, cells[1:4])
	expectEqual(t, 'a', cells[0].content.codepoint)
	expectEqual(t, uint64(0), sc3QBits(cells[1]))
	expectEqual(t, uint64(0), sc3QBits(cells[2]))
	expectEqual(t, uint64(0), sc3QBits(cells[3]))
	expectEqual(t, 'e', cells[4].content.codepoint)

	// The row attributes are not touched, not even the dirty flag.
	expect(t, row.wrap, "row.wrap")
	expect(t, row.wrap_continuation, "row.wrap_continuation")
	expectEqual(t, PageRowSemanticPromptPrompt, row.semantic_prompt)
	expect(t, !row.dirty, "!row.dirty")

	// A single cell at each end.
	s.ClearCells(page, row, cells[0:1])
	s.ClearCells(page, row, cells[4:5])
	for _, c := range cells {
		expectEqual(t, uint64(0), sc3QBits(c))
	}

	// The whole row.
	sc3Write(t, s, "xy")
	sc3QMove(s, 0, 0)
	s.ClearCells(page, row, cells)
	for _, c := range cells {
		expectEqual(t, uint64(0), sc3QBits(c))
	}
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: clearCells: graphemes"
func TestSc3_term51_clearCells_graphemes(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 6, 2, 0)
	defer s.Deinit()
	sc3Write(t, s, "abcdef")

	page := &s.cursor.page_pin.node.data
	row := s.cursor.page_row
	cells := page.GetCells(row)
	sc3QGrapheme(t, s, 1, 0, 'b')
	sc3QGrapheme(t, s, 3, 0, 'd')
	expect(t, row.grapheme, "row.grapheme")
	expectEqual(t, 2, page.GraphemeCount())

	// A range with no grapheme leaves the flag and the data alone.
	s.ClearCells(page, row, cells[4:6])
	expect(t, row.grapheme, "row.grapheme")
	expectEqual(t, 2, page.GraphemeCount())

	// A partial range with one grapheme: the flag stays since another
	// grapheme remains in the row.
	s.ClearCells(page, row, cells[0:2])
	expectEqual(t, 1, page.GraphemeCount())
	expect(t, row.grapheme, "row.grapheme")
	expectEqual(t, uint64(0), sc3QBits(cells[1]))
	expect(t, cells[3].HasGrapheme(), "cells[3].hasGrapheme()")
	expectEqualSlices(t, []uint32{0x301}, zigUnwrap(t, page.LookupGrapheme(&cells[3])))

	// A partial range that removes the last grapheme clears the flag via
	// the page's update.
	s.ClearCells(page, row, cells[2:5])
	expectEqual(t, 0, page.GraphemeCount())
	expect(t, !row.grapheme, "!row.grapheme")
	expectEqual(t, PageCellContentTagCodepoint, cells[3].content_tag)

	// A full-row clear resets the flag directly.
	sc3QGrapheme(t, s, 5, 0, 'f')
	expect(t, row.grapheme, "row.grapheme")
	s.ClearCells(page, row, cells)
	expect(t, !row.grapheme, "!row.grapheme")
	expectEqual(t, 0, page.GraphemeCount())
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: clearCells: hyperlinks"
func TestSc3_term51_clearCells_hyperlinks(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 6, 2, 0)
	defer s.Deinit()

	sc3StartLink(t, s, "https://example.com", "id")
	sc3Write(t, s, "abcd")
	s.EndHyperlink()
	sc3Write(t, s, "ef")
	sc3QMove(s, 0, 1)

	page := &s.cursor.page_pin.node.data
	row := sc3QRow(t, s, 0)
	cells := page.GetCells(row)
	id := zigUnwrap(t, page.LookupHyperlink(&cells[0]))
	expectEqual(t, 4, page.hyperlink_set.RefCount(id))
	expectEqual(t, 4, page.HyperlinkCount())
	expect(t, row.hyperlink, "row.hyperlink")

	// A range with none.
	s.ClearCells(page, row, cells[4:6])
	expectEqual(t, 4, page.hyperlink_set.RefCount(id))
	expect(t, row.hyperlink, "row.hyperlink")

	// A partial range with two of them.
	s.ClearCells(page, row, cells[1:3])
	expectEqual(t, 2, page.hyperlink_set.RefCount(id))
	expectEqual(t, 2, page.HyperlinkCount())
	expect(t, !cells[1].hyperlink, "!cells[1].hyperlink")
	expect(t, !page.LookupHyperlink(&cells[1]).Valid, "page.lookupHyperlink(&cells[1]) == null")
	expect(t, cells[0].hyperlink, "cells[0].hyperlink")
	expect(t, row.hyperlink, "row.hyperlink")
	expectEqual(t, 1, page.hyperlink_set.Count())

	// The remaining ones through a partial range clears the row flag.
	s.ClearCells(page, row, cells[0:4])
	expectEqual(t, 0, page.hyperlink_set.RefCount(id))
	expectEqual(t, 0, page.HyperlinkCount())
	expect(t, !row.hyperlink, "!row.hyperlink")
	expectEqual(t, 0, page.hyperlink_set.Count())

	// A full row clear resets the flag directly.
	sc3StartLinkImplicit(t, s, "https://example.com/2")
	s.CursorAbsolute(0, 1)
	sc3Write(t, s, "xy")
	s.EndHyperlink()
	row1 := sc3QRow(t, s, 1)
	expect(t, row1.hyperlink, "row1.hyperlink")
	s.ClearCells(page, row1, page.GetCells(row1))
	expect(t, !row1.hyperlink, "!row1.hyperlink")
	expectEqual(t, 0, page.HyperlinkCount())
	expectEqual(t, 0, page.hyperlink_set.Count())
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: clearCells: styles"
func TestSc3_term51_clearCells_styles(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 6, 2, 0)
	defer s.Deinit()

	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	sc3Write(t, s, "abc")
	sc3SetAttr(t, s, sc3A(SGRAttributeTagUnset))
	sc3Write(t, s, "def")
	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	sc3QMove(s, 0, 1)

	page := &s.cursor.page_pin.node.data
	row := sc3QRow(t, s, 0)
	cells := page.GetCells(row)
	id := s.cursor.style_id
	expect(t, row.styled, "row.styled")
	// Three cells and the cursor.
	expectEqual(t, 4, page.styles.RefCount(id))

	// A range of default-style cells releases nothing.
	s.ClearCells(page, row, cells[3:6])
	expectEqual(t, 4, page.styles.RefCount(id))
	expect(t, row.styled, "row.styled")

	// One styled cell.
	s.ClearCells(page, row, cells[0:1])
	expectEqual(t, 3, page.styles.RefCount(id))
	expect(t, row.styled, "row.styled")

	// The remaining styled cells. The flag is "recomputed" before the cells
	// are blanked, while they still carry their style ids, so a partial
	// clear never resets it (a permitted false positive).
	s.ClearCells(page, row, cells[1:3])
	expectEqual(t, 1, page.styles.RefCount(id))
	expect(t, row.styled, "row.styled")
	for _, c := range cells {
		expectEqual(t, StyleDefaultId, c.style_id)
	}

	// The cursor's own reference is untouched.
	expect(t, s.cursor.style_id == id, "s.cursor.style_id == id")
	expectEqual(t, 1, page.styles.Count())

	// A full-row clear resets the flag directly.
	sc3Write(t, s, "ab")
	sc3QMove(s, 0, 1)
	row1 := sc3QRow(t, s, 1)
	expect(t, row1.styled, "row1.styled")
	expectEqual(t, 3, page.styles.RefCount(id))
	s.ClearCells(page, row1, page.GetCells(row1))
	expect(t, !row1.styled, "!row1.styled")
	expectEqual(t, 1, page.styles.RefCount(id))
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: clearCells: mixed managed memory in one range"
func TestSc3_term51_clearCells_mixed_managed_memory_in_one_range(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 4, 2, 0)
	defer s.Deinit()

	sc3SetAttr(t, s, sc3A(SGRAttributeTagItalic))
	sc3StartLink(t, s, "https://example.com", "id")
	sc3Write(t, s, "abcd")
	s.EndHyperlink()
	sc3SetAttr(t, s, sc3A(SGRAttributeTagUnset))
	sc3QMove(s, 0, 1)

	page := &s.cursor.page_pin.node.data
	row := sc3QRow(t, s, 0)
	cells := page.GetCells(row)
	zigTry(t, page.AppendGrapheme(row, &cells[2], 0x301))
	style_id := cells[0].style_id
	link_id := zigUnwrap(t, page.LookupHyperlink(&cells[0]))
	expect(t, row.grapheme && row.hyperlink && row.styled, "row.grapheme and row.hyperlink and row.styled")
	expectEqual(t, 4, page.styles.RefCount(style_id))
	expectEqual(t, 4, page.hyperlink_set.RefCount(link_id))
	sc3QExpectExactRefs(t, s)

	s.ClearCells(page, row, cells[1:3])
	expectEqual(t, 2, page.styles.RefCount(style_id))
	expectEqual(t, 2, page.hyperlink_set.RefCount(link_id))
	expectEqual(t, 0, page.GraphemeCount())
	expect(t, !row.grapheme, "!row.grapheme")
	expect(t, row.hyperlink && row.styled, "row.hyperlink and row.styled")
	sc3QExpectExactRefs(t, s)

	s.ClearCells(page, row, cells)
	expectEqual(t, 0, page.styles.RefCount(style_id))
	expectEqual(t, 0, page.hyperlink_set.RefCount(link_id))
	expect(t, !row.grapheme && !row.hyperlink && !row.styled, "!row.grapheme and !row.hyperlink and !row.styled")
	expectEqual(t, 0, page.styles.Count())
	expectEqual(t, 0, page.hyperlink_set.Count())
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: clearCells: cleared cells take the cursor background and drop every cell flag"
func TestSc3_term51_clearCells_cleared_cells_take_the_cursor_background_and_drop_every_cell_flag(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 4, 2, 0)
	defer s.Deinit()
	sc3Write(t, s, "abcd")

	page := &s.cursor.page_pin.node.data
	row := sc3QRow(t, s, 0)
	cells := page.GetCells(row)
	cells[1].protected = true
	cells[1].semantic_content = PageCellSemanticContentPrompt

	// Default cursor: zero cells.
	s.ClearCells(page, row, cells[1:2])
	expectEqual(t, uint64(0), sc3QBits(cells[1]))

	// Cursor with a palette background.
	sc3QMove(s, 0, 1)
	sc3SetAttr(t, s, SGRAttribute{Tag: SGRAttributeTag8Bg, X8Bg: ColorNameCyan})
	cells[0].protected = true
	cells[1] = PageCellInit('z')
	cells[1].semantic_content = PageCellSemanticContentPrompt
	s.ClearCells(page, row, cells[0:2])
	for _, c := range cells[0:2] {
		expectEqual(t, PageCellContentTagBgColorPalette, c.content_tag)
		expectEqual(t, 6, c.content.color_palette)
		expectEqual(t, StyleDefaultId, c.style_id)
		expect(t, !c.protected, "!c.protected")
		expectEqual(t, PageCellSemanticContentOutput, c.semantic_content)
	}
	expectEqual(t, 'c', cells[2].content.codepoint)

	// A non-background style gives zero cells.
	sc3SetAttr(t, s, sc3A(SGRAttributeTagResetBg))
	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	s.ClearCells(page, row, cells[2:4])
	expectEqual(t, uint64(0), sc3QBits(cells[2]))
	expectEqual(t, uint64(0), sc3QBits(cells[3]))
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: clearCells: a wide character and its spacer tail"
func TestSc3_term51_clearCells_a_wide_character_and_its_spacer_tail(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 2, 0)
	defer s.Deinit()
	sc3Write(t, s, "A橋B")

	page := &s.cursor.page_pin.node.data
	row := sc3QRow(t, s, 0)
	cells := page.GetCells(row)
	expectEqual(t, PageCellWideWide, cells[1].wide)
	expectEqual(t, PageCellWideSpacerTail, cells[2].wide)

	// Clearing both halves together leaves a consistent page.
	s.ClearCells(page, row, cells[1:3])
	expectEqual(t, uint64(0), sc3QBits(cells[1]))
	expectEqual(t, uint64(0), sc3QBits(cells[2]))
	expectEqual(t, 'A', cells[0].content.codepoint)
	expectEqual(t, 'B', cells[3].content.codepoint)
	page.AssertIntegrity()
}

// Zig: test "term51: clearUnprotectedCells: runs of unprotected cells between protected cells"
func TestSc3_term51_clearUnprotectedCells_runs_of_unprotected_cells_between_protected_cells(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 8, 2, 0)
	defer s.Deinit()
	sc3Write(t, s, "ABCDEFGH")

	page := &s.cursor.page_pin.node.data
	row := sc3QRow(t, s, 0)
	cells := page.GetCells(row)
	for _, i := range []int{2, 3, 6} {
		cells[i].protected = true
	}

	s.ClearUnprotectedCells(page, row, cells)
	expected := "\x00\x00CD\x00\x00G\x00"
	for i, c := range cells {
		expectEqual(t, uint32(expected[i]), c.content.codepoint)
		expectEqual(t, i == 2 || i == 3 || i == 6, c.protected)
	}
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: clearUnprotectedCells: none protected clears everything and resets row flags"
func TestSc3_term51_clearUnprotectedCells_none_protected_clears_everything_and_resets_row_flags(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 4, 2, 0)
	defer s.Deinit()
	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	sc3Write(t, s, "ab")
	sc3SetAttr(t, s, sc3A(SGRAttributeTagUnset))
	sc3Write(t, s, "cd")
	sc3QMove(s, 0, 1)

	page := &s.cursor.page_pin.node.data
	row := sc3QRow(t, s, 0)
	cells := page.GetCells(row)
	zigTry(t, page.AppendGrapheme(row, &cells[3], 0x301))
	expect(t, row.styled && row.grapheme, "row.styled and row.grapheme")

	s.ClearUnprotectedCells(page, row, cells)
	for _, c := range cells {
		expectEqual(t, uint64(0), sc3QBits(c))
	}
	expect(t, !row.styled, "!row.styled")
	expect(t, !row.grapheme, "!row.grapheme")
	expectEqual(t, 0, page.GraphemeCount())
	expectEqual(t, 0, page.styles.Count())
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: clearUnprotectedCells: all protected clears nothing"
func TestSc3_term51_clearUnprotectedCells_all_protected_clears_nothing(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 4, 2, 0)
	defer s.Deinit()
	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	sc3Write(t, s, "abcd")
	sc3SetAttr(t, s, sc3A(SGRAttributeTagUnset))
	sc3QMove(s, 0, 1)

	page := &s.cursor.page_pin.node.data
	row := sc3QRow(t, s, 0)
	cells := page.GetCells(row)
	zigTry(t, page.AppendGrapheme(row, &cells[0], 0x301))
	for i := range cells {
		cells[i].protected = true
	}

	s.ClearUnprotectedCells(page, row, cells)
	expectEqual(t, 'a', cells[0].content.codepoint)
	expectEqual(t, 'd', cells[3].content.codepoint)
	expect(t, row.styled && row.grapheme, "row.styled and row.grapheme")
	expectEqual(t, 1, page.GraphemeCount())
	expectEqual(t, 1, page.styles.Count())
	expectEqual(t, 4, page.styles.RefCount(cells[0].style_id))
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: clearUnprotectedCells: protection at the ends of the slice"
func TestSc3_term51_clearUnprotectedCells_protection_at_the_ends_of_the_slice(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 2, 0)
	defer s.Deinit()

	page := &s.cursor.page_pin.node.data
	row := sc3QRow(t, s, 0)
	cells := page.GetCells(row)

	// [P u u u u], [u u u u P], [P u u u P] and [u P u P u].
	patterns := []string{"Puuuu", "uuuuP", "PuuuP", "uPuPu"}
	for _, pat := range patterns {
		for i := range cells {
			cells[i] = PageCellInit('x')
			cells[i].protected = pat[i] == 'P'
		}
		s.ClearUnprotectedCells(page, row, cells)
		for i, c := range cells {
			if pat[i] == 'P' {
				expectEqual(t, 'x', c.content.codepoint)
				expect(t, c.protected, "c.protected")
			} else {
				expectEqual(t, uint64(0), sc3QBits(c))
			}
		}
	}
}

// Zig: test "term51: clearUnprotectedCells: empty and partial slices"
func TestSc3_term51_clearUnprotectedCells_empty_and_partial_slices(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 2, 0)
	defer s.Deinit()
	sc3Write(t, s, "abcde")

	page := &s.cursor.page_pin.node.data
	row := sc3QRow(t, s, 0)
	cells := page.GetCells(row)

	// An empty slice does nothing.
	s.ClearUnprotectedCells(page, row, cells[2:2])
	s.ClearUnprotectedCells(page, row, cells[0:0])
	expectEqual(t, 'a', cells[0].content.codepoint)
	expectEqual(t, 'c', cells[2].content.codepoint)

	// A sub-slice only touches its range.
	cells[3].protected = true
	s.ClearUnprotectedCells(page, row, cells[1:4])
	expectEqual(t, 'a', cells[0].content.codepoint)
	expectEqual(t, uint64(0), sc3QBits(cells[1]))
	expectEqual(t, uint64(0), sc3QBits(cells[2]))
	expectEqual(t, 'd', cells[3].content.codepoint)
	expectEqual(t, 'e', cells[4].content.codepoint)

	// A single unprotected and a single protected cell.
	s.ClearUnprotectedCells(page, row, cells[4:5])
	expectEqual(t, uint64(0), sc3QBits(cells[4]))
	s.ClearUnprotectedCells(page, row, cells[3:4])
	expectEqual(t, 'd', cells[3].content.codepoint)
}

// Zig: test "term51: clearUnprotectedCells: managed memory is released only for cleared cells"
func TestSc3_term51_clearUnprotectedCells_managed_memory_is_released_only_for_cleared_cells(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 6, 2, 0)
	defer s.Deinit()
	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	sc3StartLink(t, s, "https://example.com", "id")
	sc3Write(t, s, "abcdef")
	s.EndHyperlink()
	sc3SetAttr(t, s, sc3A(SGRAttributeTagUnset))
	sc3QMove(s, 0, 1)

	page := &s.cursor.page_pin.node.data
	row := sc3QRow(t, s, 0)
	cells := page.GetCells(row)
	zigTry(t, page.AppendGrapheme(row, &cells[1], 0x301))
	zigTry(t, page.AppendGrapheme(row, &cells[4], 0x302))
	cells[4].protected = true
	style_id := cells[0].style_id
	link_id := zigUnwrap(t, page.LookupHyperlink(&cells[0]))
	sc3QExpectExactRefs(t, s)

	s.ClearUnprotectedCells(page, row, cells)
	// Only cell 4 survives.
	expectEqual(t, 1, page.styles.RefCount(style_id))
	expectEqual(t, 1, page.hyperlink_set.RefCount(link_id))
	expectEqual(t, 1, page.GraphemeCount())
	expectEqualSlices(t, []uint32{0x302}, zigUnwrap(t, page.LookupGrapheme(&cells[4])))
	expect(t, row.styled && row.hyperlink && row.grapheme, "row.styled and row.hyperlink and row.grapheme")
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: clearUnprotectedCells: cleared cells take the cursor background"
func TestSc3_term51_clearUnprotectedCells_cleared_cells_take_the_cursor_background(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 4, 2, 0)
	defer s.Deinit()
	sc3Write(t, s, "abcd")
	page := &s.cursor.page_pin.node.data
	row := sc3QRow(t, s, 0)
	cells := page.GetCells(row)
	cells[2].protected = true
	sc3QMove(s, 0, 1)
	sc3SetAttr(t, s, sc3AttrBgRGB(9, 8, 7))

	s.ClearUnprotectedCells(page, row, cells)
	for _, i := range []int{0, 1, 3} {
		expectEqual(t, PageCellContentTagBgColorRgb, cells[i].content_tag)
		expectEqual(t, 9, cells[i].content.color_rgb.r)
	}
	expectEqual(t, 'c', cells[2].content.codepoint)
}

// Zig: test "term51: splitCellBoundary: the boundary after the last cell only matters for a wrapped spacer head"
func TestSc3_term51_splitCellBoundary_the_boundary_after_the_last_cell_only_matters_for_a_wrapped_spacer_head(t *testing.T) {
	testStdCapacity(t)

	// Not wrapped: nothing happens.
	func() {
		s := sc3QTextScreen(t, "ABCDEF", 0)
		defer s.Deinit()
		s.CursorAbsolute(0, 0)
		before := sc3QBits(*sc3QCell(t, s, 5, 0))
		s.SplitCellBoundary(6)
		expectEqual(t, before, sc3QBits(*sc3QCell(t, s, 5, 0)))
		expect(t, !s.cursor.page_row.wrap, "!s.cursor.page_row.wrap")
	}()

	// Wrapped, but the last cell is not a spacer head: nothing happens.
	func() {
		s := sc3QTextScreen(t, "ABCDEFG", 0)
		defer s.Deinit()
		s.CursorAbsolute(0, 0)
		expect(t, s.cursor.page_row.wrap, "s.cursor.page_row.wrap")
		s.SplitCellBoundary(6)
		expectEqual(t, 'F', sc3QCell(t, s, 5, 0).content.codepoint)
		expect(t, s.cursor.page_row.wrap, "s.cursor.page_row.wrap")
	}()

	// Wrapped with a spacer head: it is cleared, wrap state is not touched.
	func() {
		s := sc3QTextScreen(t, "ABCDE橋", 0)
		defer s.Deinit()
		expectEqual(t, PageCellWideSpacerHead, sc3QCell(t, s, 5, 0).wide)
		expectEqual(t, PageCellWideWide, sc3QCell(t, s, 0, 1).wide)
		s.CursorAbsolute(0, 0)
		s.SplitCellBoundary(6)
		expectEqual(t, uint64(0), sc3QBits(*sc3QCell(t, s, 5, 0)))
		expectEqual(t, 'E', sc3QCell(t, s, 4, 0).content.codepoint)
		expect(t, s.cursor.page_row.wrap, "s.cursor.page_row.wrap")
		expect(t, sc3QRow(t, s, 1).wrap_continuation, "term51QRow(&s, 1).wrap_continuation")
		expectEqual(t, PageCellWideWide, sc3QCell(t, s, 0, 1).wide)
		s.cursor.page_pin.node.data.AssertIntegrity()
	}()
}

// Zig: test "term51: splitCellBoundary: x of 0 and 1 on a wrapped row that starts with a wide character"
func TestSc3_term51_splitCellBoundary_x_of_0_and_1_on_a_wrapped_row_that_starts_with_a_wide_character(t *testing.T) {
	testStdCapacity(t)

	// x = 0: the previous row's spacer head is cleared and nothing else.
	func() {
		s := sc3QTextScreen(t, "ABCDE橋", 0)
		defer s.Deinit()
		s.CursorAbsolute(0, 1)
		expect(t, s.cursor.page_row.wrap_continuation, "s.cursor.page_row.wrap_continuation")
		s.SplitCellBoundary(0)
		expectEqual(t, uint64(0), sc3QBits(*sc3QCell(t, s, 5, 0)))
		expectEqual(t, PageCellWideWide, sc3QCell(t, s, 0, 1).wide)
		expectEqual(t, PageCellWideSpacerTail, sc3QCell(t, s, 1, 1).wide)
		expect(t, sc3QRow(t, s, 0).wrap, "term51QRow(&s, 0).wrap")
		expect(t, s.cursor.page_row.wrap_continuation, "s.cursor.page_row.wrap_continuation")
	}()

	// x = 1: additionally the wide character is split, so it is cleared.
	func() {
		s := sc3QTextScreen(t, "ABCDE橋", 0)
		defer s.Deinit()
		s.CursorAbsolute(0, 1)
		s.SplitCellBoundary(1)
		expectEqual(t, uint64(0), sc3QBits(*sc3QCell(t, s, 5, 0)))
		expectEqual(t, uint64(0), sc3QBits(*sc3QCell(t, s, 0, 1)))
		expectEqual(t, uint64(0), sc3QBits(*sc3QCell(t, s, 1, 1)))
		expectEqual(t, 'E', sc3QCell(t, s, 4, 0).content.codepoint)
		s.cursor.page_pin.node.data.AssertIntegrity()
	}()

	// x = 2 does not look at the previous row.
	func() {
		s := sc3QTextScreen(t, "ABCDE橋", 0)
		defer s.Deinit()
		s.CursorAbsolute(0, 1)
		s.SplitCellBoundary(2)
		// The left neighbour of x=2 is the spacer tail: nothing is cleared,
		// in particular not the spacer head of the previous row.
		expectEqual(t, PageCellWideSpacerHead, sc3QCell(t, s, 5, 0).wide)
		expectEqual(t, PageCellWideWide, sc3QCell(t, s, 0, 1).wide)
		expectEqual(t, PageCellWideSpacerTail, sc3QCell(t, s, 1, 1).wide)
	}()
}

// Zig: test "term51: splitCellBoundary: wrapped rows without a spacer head or without a wide first cell"
func TestSc3_term51_splitCellBoundary_wrapped_rows_without_a_spacer_head_or_without_a_wide_first_cell(t *testing.T) {
	testStdCapacity(t)

	// Narrow first cell on a continuation row.
	func() {
		s := sc3QTextScreen(t, "ABCDEFG", 0)
		defer s.Deinit()
		s.CursorAbsolute(0, 1)
		expect(t, s.cursor.page_row.wrap_continuation, "s.cursor.page_row.wrap_continuation")
		s.SplitCellBoundary(0)
		s.SplitCellBoundary(1)
		expectEqual(t, 'F', sc3QCell(t, s, 5, 0).content.codepoint)
		expectEqual(t, 'G', sc3QCell(t, s, 0, 1).content.codepoint)
	}()

	// Wide first cell on a continuation row, but the previous row's last
	// cell is a regular character (the wide character wrapped because of a
	// pending wrap, not because of a spacer head).
	func() {
		s := sc3QTextScreen(t, "ABCDEF橋", 0)
		defer s.Deinit()
		s.CursorAbsolute(0, 1)
		expectEqual(t, PageCellWideWide, sc3QCell(t, s, 0, 1).wide)
		expect(t, s.cursor.page_row.wrap_continuation, "s.cursor.page_row.wrap_continuation")
		s.SplitCellBoundary(0)
		expectEqual(t, 'F', sc3QCell(t, s, 5, 0).content.codepoint)
		expectEqual(t, PageCellWideWide, sc3QCell(t, s, 0, 1).wide)
		// x = 1 splits the wide character.
		s.SplitCellBoundary(1)
		expectEqual(t, 'F', sc3QCell(t, s, 5, 0).content.codepoint)
		expectEqual(t, uint64(0), sc3QBits(*sc3QCell(t, s, 0, 1)))
		expectEqual(t, uint64(0), sc3QBits(*sc3QCell(t, s, 1, 1)))
	}()

	// Wide first cell on a row that is not a continuation.
	func() {
		s := sc3QTextScreen(t, "橋CD", 0)
		defer s.Deinit()
		s.CursorAbsolute(0, 0)
		expect(t, !s.cursor.page_row.wrap_continuation, "!s.cursor.page_row.wrap_continuation")
		s.SplitCellBoundary(0)
		expectEqual(t, PageCellWideWide, sc3QCell(t, s, 0, 0).wide)
		s.SplitCellBoundary(1)
		expectEqual(t, uint64(0), sc3QBits(*sc3QCell(t, s, 0, 0)))
		expectEqual(t, uint64(0), sc3QBits(*sc3QCell(t, s, 1, 0)))
		expectEqual(t, 'C', sc3QCell(t, s, 2, 0).content.codepoint)
	}()

	// A continuation flag on the very first row of the screen: there is no
	// previous row to look at.
	func() {
		s := sc3QTextScreen(t, "橋CD", 0)
		defer s.Deinit()
		s.CursorAbsolute(0, 0)
		s.cursor.page_row.wrap_continuation = true
		s.SplitCellBoundary(0)
		expectEqual(t, PageCellWideWide, sc3QCell(t, s, 0, 0).wide)
		s.SplitCellBoundary(1)
		expectEqual(t, uint64(0), sc3QBits(*sc3QCell(t, s, 0, 0)))
	}()
}

// Zig: test "term51: splitCellBoundary: boundaries in the middle of the row"
func TestSc3_term51_splitCellBoundary_boundaries_in_the_middle_of_the_row(t *testing.T) {
	testStdCapacity(t)

	// A, wide (1,2), B, C, D.
	text := "A橋BCD"

	// Left of the boundary is narrow: nothing.
	for _, x := range []SizeCellCountInt{1, 4, 5} {
		func() {
			s := sc3QTextScreen(t, text, 0)
			defer s.Deinit()
			s.CursorAbsolute(0, 0)
			s.SplitCellBoundary(x)
			expectEqual(t, 'A', sc3QCell(t, s, 0, 0).content.codepoint)
			expectEqual(t, PageCellWideWide, sc3QCell(t, s, 1, 0).wide)
			expectEqual(t, PageCellWideSpacerTail, sc3QCell(t, s, 2, 0).wide)
			expectEqual(t, 'B', sc3QCell(t, s, 3, 0).content.codepoint)
		}()
	}

	// Left of the boundary is the spacer tail: nothing.
	func() {
		s := sc3QTextScreen(t, text, 0)
		defer s.Deinit()
		s.CursorAbsolute(0, 0)
		s.SplitCellBoundary(3)
		expectEqual(t, PageCellWideWide, sc3QCell(t, s, 1, 0).wide)
		expectEqual(t, PageCellWideSpacerTail, sc3QCell(t, s, 2, 0).wide)
	}()

	// Left of the boundary is the wide cell: both halves are cleared.
	func() {
		s := sc3QTextScreen(t, text, 0)
		defer s.Deinit()
		s.CursorAbsolute(0, 0)
		s.SplitCellBoundary(2)
		expectEqual(t, 'A', sc3QCell(t, s, 0, 0).content.codepoint)
		expectEqual(t, uint64(0), sc3QBits(*sc3QCell(t, s, 1, 0)))
		expectEqual(t, uint64(0), sc3QBits(*sc3QCell(t, s, 2, 0)))
		expectEqual(t, 'B', sc3QCell(t, s, 3, 0).content.codepoint)
		s.cursor.page_pin.node.data.AssertIntegrity()
	}()

	// The wide character in the last two columns, boundary at cols - 1.
	func() {
		s := sc3QTextScreen(t, "ABCD橋", 0)
		defer s.Deinit()
		s.CursorAbsolute(0, 0)
		expectEqual(t, PageCellWideWide, sc3QCell(t, s, 4, 0).wide)
		expectEqual(t, PageCellWideSpacerTail, sc3QCell(t, s, 5, 0).wide)
		s.SplitCellBoundary(5)
		expectEqual(t, 'D', sc3QCell(t, s, 3, 0).content.codepoint)
		expectEqual(t, uint64(0), sc3QBits(*sc3QCell(t, s, 4, 0)))
		expectEqual(t, uint64(0), sc3QBits(*sc3QCell(t, s, 5, 0)))
		// And x == cols afterwards is a no-op on an unwrapped row.
		s.SplitCellBoundary(6)
		s.cursor.page_pin.node.data.AssertIntegrity()
	}()

	// Cleared halves take the cursor's background.
	func() {
		s := sc3QTextScreen(t, text, 0)
		defer s.Deinit()
		s.CursorAbsolute(0, 0)
		sc3SetAttr(t, s, sc3AttrBg256(4))
		s.SplitCellBoundary(2)
		expectEqual(t, PageCellContentTagBgColorPalette, sc3QCell(t, s, 1, 0).content_tag)
		expectEqual(t, PageCellContentTagBgColorPalette, sc3QCell(t, s, 2, 0).content_tag)
		expectEqual(t, PageCellWideNarrow, sc3QCell(t, s, 1, 0).wide)
		sc3QExpectExactRefs(t, s)
	}()
}

// Zig: test "term51: splitCellBoundary: a spacer head on a previous row in another page"
func TestSc3_term51_splitCellBoundary_a_spacer_head_on_a_previous_row_in_another_page(t *testing.T) {
	testStdCapacity(t)
	s := sc3QCrossPageScreen(t, 6)
	defer s.Deinit()

	// Row 1 (first page) ends with a spacer head and wraps into row 2
	// (second page), which starts with a wide character.
	*sc3QCell(t, s, 5, 1) = sc3CellWide(0, PageCellWideSpacerHead)
	sc3QRow(t, s, 1).wrap = true
	*sc3QCell(t, s, 0, 2) = sc3CellWide(0x6A4B, PageCellWideWide)
	*sc3QCell(t, s, 1, 2) = sc3CellWide(0, PageCellWideSpacerTail)
	sc3QRow(t, s, 2).wrap_continuation = true

	s.CursorAbsolute(0, 2)
	expect(t, s.cursor.page_pin.node == s.pages.pages.last, "s.cursor.page_pin.node == s.pages.pages.last.?")
	s.SplitCellBoundary(0)

	expectEqual(t, uint64(0), sc3QBits(*sc3QCell(t, s, 5, 1)))
	expectEqual(t, PageCellWideWide, sc3QCell(t, s, 0, 2).wide)
	expect(t, sc3QRow(t, s, 1).wrap, "term51QRow(&s, 1).wrap")
	s.pages.pages.first.data.AssertIntegrity()
	s.pages.pages.last.data.AssertIntegrity()
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: setAttribute: every attribute from the default style"
func TestSc3_term51_setAttribute_every_attribute_from_the_default_style(t *testing.T) {
	testStdCapacity(t)
	flags := func(f styleFlags) Style { return Style{flags: f} }
	ul := func(u SGRAttributeUnderline) Style { return flags(styleFlags{underline: u}) }
	fgPal := func(n uint8) Style {
		return Style{fg_color: StyleColor{Tag: StyleColorPalette, Palette: n}}
	}
	bgPal := func(n uint8) Style {
		return Style{bg_color: StyleColor{Tag: StyleColorPalette, Palette: n}}
	}
	ulPal := func(n uint8) Style {
		return Style{underline_color: StyleColor{Tag: StyleColorPalette, Palette: n}}
	}
	rgb := func(r uint8, g uint8, b uint8) StyleColor {
		return StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{r: r, g: g, b: b}}
	}
	n8 := func(tag SGRAttributeTag, name ColorName) SGRAttribute {
		a := SGRAttribute{Tag: tag}
		switch tag {
		case SGRAttributeTag8Fg:
			a.X8Fg = name
		case SGRAttributeTag8Bg:
			a.X8Bg = name
		case SGRAttributeTag8BrightFg:
			a.X8BrightFg = name
		case SGRAttributeTag8BrightBg:
			a.X8BrightBg = name
		}
		return a
	}
	cases := []sc3QAttrCase{
		{attr: sc3A(SGRAttributeTagUnset), expected: Style{}},
		{attr: sc3A(SGRAttributeTagBold), expected: flags(styleFlags{bold: true})},
		{attr: sc3A(SGRAttributeTagResetBold), expected: Style{}},
		{attr: sc3A(SGRAttributeTagItalic), expected: flags(styleFlags{italic: true})},
		{attr: sc3A(SGRAttributeTagResetItalic), expected: Style{}},
		{attr: sc3A(SGRAttributeTagFaint), expected: flags(styleFlags{faint: true})},
		{attr: sc3AttrUnderline(SGRAttributeUnderlineNone), expected: Style{}},
		{attr: sc3AttrUnderline(SGRAttributeUnderlineSingle), expected: ul(SGRAttributeUnderlineSingle)},
		{attr: sc3AttrUnderline(SGRAttributeUnderlineDouble), expected: ul(SGRAttributeUnderlineDouble)},
		{attr: sc3AttrUnderline(SGRAttributeUnderlineCurly), expected: ul(SGRAttributeUnderlineCurly)},
		{attr: sc3AttrUnderline(SGRAttributeUnderlineDotted), expected: ul(SGRAttributeUnderlineDotted)},
		{attr: sc3AttrUnderline(SGRAttributeUnderlineDashed), expected: ul(SGRAttributeUnderlineDashed)},
		{
			attr:     SGRAttribute{Tag: SGRAttributeTagUnderlineColor, UnderlineColor: ColorRGB{r: 1, g: 2, b: 3}},
			expected: Style{underline_color: rgb(1, 2, 3)},
		},
		// Black is a real colour, not "none".
		{
			attr:     SGRAttribute{Tag: SGRAttributeTagUnderlineColor, UnderlineColor: ColorRGB{r: 0, g: 0, b: 0}},
			expected: Style{underline_color: rgb(0, 0, 0)},
		},
		{attr: SGRAttribute{Tag: SGRAttributeTag256UnderlineColor, X256UnderlineColor: 0}, expected: ulPal(0)},
		{attr: SGRAttribute{Tag: SGRAttributeTag256UnderlineColor, X256UnderlineColor: 255}, expected: ulPal(255)},
		{attr: sc3A(SGRAttributeTagResetUnderlineColor), expected: Style{}},
		{attr: sc3A(SGRAttributeTagOverline), expected: flags(styleFlags{overline: true})},
		{attr: sc3A(SGRAttributeTagResetOverline), expected: Style{}},
		{attr: sc3A(SGRAttributeTagBlink), expected: flags(styleFlags{blink: true})},
		{attr: sc3A(SGRAttributeTagResetBlink), expected: Style{}},
		{attr: sc3A(SGRAttributeTagInverse), expected: flags(styleFlags{inverse: true})},
		{attr: sc3A(SGRAttributeTagResetInverse), expected: Style{}},
		{attr: sc3A(SGRAttributeTagInvisible), expected: flags(styleFlags{invisible: true})},
		{attr: sc3A(SGRAttributeTagResetInvisible), expected: Style{}},
		{attr: sc3A(SGRAttributeTagStrikethrough), expected: flags(styleFlags{strikethrough: true})},
		{attr: sc3A(SGRAttributeTagResetStrikethrough), expected: Style{}},
		{
			attr:     sc3AttrFgRGB(255, 128, 0),
			expected: Style{fg_color: rgb(255, 128, 0)},
		},
		{
			attr:     sc3AttrBgRGB(0, 128, 255),
			expected: Style{bg_color: rgb(0, 128, 255)},
		},
		{attr: n8(SGRAttributeTag8Fg, ColorNameBlack), expected: fgPal(0)},
		{attr: n8(SGRAttributeTag8Fg, ColorNameRed), expected: fgPal(1)},
		{attr: n8(SGRAttributeTag8Fg, ColorNameGreen), expected: fgPal(2)},
		{attr: n8(SGRAttributeTag8Fg, ColorNameYellow), expected: fgPal(3)},
		{attr: n8(SGRAttributeTag8Fg, ColorNameBlue), expected: fgPal(4)},
		{attr: n8(SGRAttributeTag8Fg, ColorNameMagenta), expected: fgPal(5)},
		{attr: n8(SGRAttributeTag8Fg, ColorNameCyan), expected: fgPal(6)},
		{attr: n8(SGRAttributeTag8Fg, ColorNameWhite), expected: fgPal(7)},
		{attr: n8(SGRAttributeTag8Bg, ColorNameBlack), expected: bgPal(0)},
		{attr: n8(SGRAttributeTag8Bg, ColorNameRed), expected: bgPal(1)},
		{attr: n8(SGRAttributeTag8Bg, ColorNameGreen), expected: bgPal(2)},
		{attr: n8(SGRAttributeTag8Bg, ColorNameYellow), expected: bgPal(3)},
		{attr: n8(SGRAttributeTag8Bg, ColorNameBlue), expected: bgPal(4)},
		{attr: n8(SGRAttributeTag8Bg, ColorNameMagenta), expected: bgPal(5)},
		{attr: n8(SGRAttributeTag8Bg, ColorNameCyan), expected: bgPal(6)},
		{attr: n8(SGRAttributeTag8Bg, ColorNameWhite), expected: bgPal(7)},
		{attr: sc3A(SGRAttributeTagResetFg), expected: Style{}},
		{attr: sc3A(SGRAttributeTagResetBg), expected: Style{}},
		{attr: n8(SGRAttributeTag8BrightFg, ColorNameBrightBlack), expected: fgPal(8)},
		{attr: n8(SGRAttributeTag8BrightFg, ColorNameBrightRed), expected: fgPal(9)},
		{attr: n8(SGRAttributeTag8BrightFg, ColorNameBrightGreen), expected: fgPal(10)},
		{attr: n8(SGRAttributeTag8BrightFg, ColorNameBrightYellow), expected: fgPal(11)},
		{attr: n8(SGRAttributeTag8BrightFg, ColorNameBrightBlue), expected: fgPal(12)},
		{attr: n8(SGRAttributeTag8BrightFg, ColorNameBrightMagenta), expected: fgPal(13)},
		{attr: n8(SGRAttributeTag8BrightFg, ColorNameBrightCyan), expected: fgPal(14)},
		{attr: n8(SGRAttributeTag8BrightFg, ColorNameBrightWhite), expected: fgPal(15)},
		{attr: n8(SGRAttributeTag8BrightBg, ColorNameBrightBlack), expected: bgPal(8)},
		{attr: n8(SGRAttributeTag8BrightBg, ColorNameBrightRed), expected: bgPal(9)},
		{attr: n8(SGRAttributeTag8BrightBg, ColorNameBrightGreen), expected: bgPal(10)},
		{attr: n8(SGRAttributeTag8BrightBg, ColorNameBrightYellow), expected: bgPal(11)},
		{attr: n8(SGRAttributeTag8BrightBg, ColorNameBrightBlue), expected: bgPal(12)},
		{attr: n8(SGRAttributeTag8BrightBg, ColorNameBrightMagenta), expected: bgPal(13)},
		{attr: n8(SGRAttributeTag8BrightBg, ColorNameBrightCyan), expected: bgPal(14)},
		{attr: n8(SGRAttributeTag8BrightBg, ColorNameBrightWhite), expected: bgPal(15)},
		// The name is not validated: any palette index passes through.
		{attr: n8(SGRAttributeTag8Fg, ColorName(100)), expected: fgPal(100)},
		{attr: n8(SGRAttributeTag8BrightBg, ColorName(255)), expected: bgPal(255)},
		{attr: sc3AttrFg256(0), expected: fgPal(0)},
		{attr: sc3AttrFg256(255), expected: fgPal(255)},
		{attr: sc3AttrBg256(0), expected: bgPal(0)},
		{attr: sc3AttrBg256(255), expected: bgPal(255)},
		{
			attr: SGRAttribute{
				Tag:     SGRAttributeTagUnknown,
				Unknown: SGRAttributeUnknown{full: []uint16{}, partial: []uint16{}},
			},
			expected: Style{},
		},
	}

	for i, c := range cases {
		// Zig: errdefer std.debug.print("setAttribute case {} failed\n", .{i});
		t.Logf("setAttribute case %d", i)
		func() {
			s := sc3Init(t, 4, 2, 0)
			defer s.Deinit()
			sc3SetAttr(t, s, c.attr)
			expectEqual(t, c.expected, s.cursor.style)

			page := &s.cursor.page_pin.node.data
			if c.expected.Default() {
				expectEqual(t, StyleDefaultId, s.cursor.style_id)
				expectEqual(t, 0, page.styles.Count())
			} else {
				expect(t, s.cursor.style_id != StyleDefaultId, "s.cursor.style_id != style.default_id")
				expectEqual(t, 1, page.styles.Count())
				expectEqual(t, 1, page.styles.RefCount(s.cursor.style_id))
				expect(t, page.styles.Get(s.cursor.style_id).Eql(c.expected), "page.styles.get(page.memory, s.cursor.style_id).eql(c.expected)")
			}
			sc3QExpectExactRefs(t, s)
		}()
	}
}

// Zig: test "term51: setAttribute: every attribute from a style with everything set"
func TestSc3_term51_setAttribute_every_attribute_from_a_style_with_everything_set(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 4, 2, 0)
	defer s.Deinit()
	sc3QApplyFull(t, s)
	expectEqual(t, sc3QFullStyle(), s.cursor.style)

	type Case struct {
		attr     SGRAttribute
		expected Style
	}
	full := sc3QFullStyle()

	cases := func() []Case {
		var list []Case

		// reset_bold clears bold AND faint.
		e := full
		e.flags.bold = false
		e.flags.faint = false
		list = append(list, Case{attr: sc3A(SGRAttributeTagResetBold), expected: e})

		e = full
		e.flags.italic = false
		list = append(list, Case{attr: sc3A(SGRAttributeTagResetItalic), expected: e})

		// faint, bold, italic... set again change nothing.
		list = append(list, Case{attr: sc3A(SGRAttributeTagBold), expected: full})
		list = append(list, Case{attr: sc3A(SGRAttributeTagFaint), expected: full})
		list = append(list, Case{attr: sc3A(SGRAttributeTagItalic), expected: full})

		e = full
		e.flags.underline = SGRAttributeUnderlineNone
		list = append(list, Case{attr: sc3AttrUnderline(SGRAttributeUnderlineNone), expected: e})
		e = full
		e.flags.underline = SGRAttributeUnderlineDashed
		list = append(list, Case{attr: sc3AttrUnderline(SGRAttributeUnderlineDashed), expected: e})

		e = full
		e.underline_color = StyleColor{Tag: StyleColorNone}
		list = append(list, Case{attr: sc3A(SGRAttributeTagResetUnderlineColor), expected: e})
		e = full
		e.underline_color = StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{r: 9, g: 9, b: 9}}
		list = append(list, Case{
			attr:     SGRAttribute{Tag: SGRAttributeTagUnderlineColor, UnderlineColor: ColorRGB{r: 9, g: 9, b: 9}},
			expected: e,
		})
		e = full
		e.underline_color = StyleColor{Tag: StyleColorPalette, Palette: 7}
		list = append(list, Case{
			attr:     SGRAttribute{Tag: SGRAttributeTag256UnderlineColor, X256UnderlineColor: 7},
			expected: e,
		})

		e = full
		e.flags.overline = false
		list = append(list, Case{attr: sc3A(SGRAttributeTagResetOverline), expected: e})
		e = full
		e.flags.blink = false
		list = append(list, Case{attr: sc3A(SGRAttributeTagResetBlink), expected: e})
		e = full
		e.flags.inverse = false
		list = append(list, Case{attr: sc3A(SGRAttributeTagResetInverse), expected: e})
		e = full
		e.flags.invisible = false
		list = append(list, Case{attr: sc3A(SGRAttributeTagResetInvisible), expected: e})
		e = full
		e.flags.strikethrough = false
		list = append(list, Case{attr: sc3A(SGRAttributeTagResetStrikethrough), expected: e})

		e = full
		e.fg_color = StyleColor{Tag: StyleColorNone}
		list = append(list, Case{attr: sc3A(SGRAttributeTagResetFg), expected: e})
		e = full
		e.bg_color = StyleColor{Tag: StyleColorNone}
		list = append(list, Case{attr: sc3A(SGRAttributeTagResetBg), expected: e})

		// Colours replace colours of any kind, and only their own slot.
		e = full
		e.fg_color = StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{r: 1, g: 2, b: 3}}
		list = append(list, Case{attr: sc3AttrFgRGB(1, 2, 3), expected: e})
		e = full
		e.bg_color = StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{r: 4, g: 5, b: 6}}
		list = append(list, Case{attr: sc3AttrBgRGB(4, 5, 6), expected: e})
		e = full
		e.fg_color = StyleColor{Tag: StyleColorPalette, Palette: 3}
		list = append(list, Case{attr: SGRAttribute{Tag: SGRAttributeTag8Fg, X8Fg: ColorNameYellow}, expected: e})
		e = full
		e.bg_color = StyleColor{Tag: StyleColorPalette, Palette: 12}
		list = append(list, Case{attr: SGRAttribute{Tag: SGRAttributeTag8BrightBg, X8BrightBg: ColorNameBrightBlue}, expected: e})
		e = full
		e.fg_color = StyleColor{Tag: StyleColorPalette, Palette: 14}
		list = append(list, Case{attr: SGRAttribute{Tag: SGRAttributeTag8BrightFg, X8BrightFg: ColorNameBrightCyan}, expected: e})
		e = full
		e.bg_color = StyleColor{Tag: StyleColorPalette, Palette: 5}
		list = append(list, Case{attr: SGRAttribute{Tag: SGRAttributeTag8Bg, X8Bg: ColorNameMagenta}, expected: e})
		e = full
		e.fg_color = StyleColor{Tag: StyleColorPalette, Palette: 17}
		list = append(list, Case{attr: sc3AttrFg256(17), expected: e})
		e = full
		e.bg_color = StyleColor{Tag: StyleColorPalette, Palette: 18}
		list = append(list, Case{attr: sc3AttrBg256(18), expected: e})

		// unknown changes nothing.
		list = append(list, Case{
			attr: SGRAttribute{
				Tag:     SGRAttributeTagUnknown,
				Unknown: SGRAttributeUnknown{full: []uint16{38, 9}, partial: []uint16{9}},
			},
			expected: full,
		})

		// unset gives the default style.
		list = append(list, Case{attr: sc3A(SGRAttributeTagUnset), expected: Style{}})
		assert(len(list) == 27)
		return list
	}()
	for i, c := range cases {
		// Zig: errdefer std.debug.print("setAttribute full case {} failed\n", .{i});
		t.Logf("setAttribute full case %d", i)
		func() {
			ts := sc3Init(t, 4, 2, 0)
			defer ts.Deinit()
			sc3QApplyFull(t, ts)
			sc3SetAttr(t, ts, c.attr)
			expectEqual(t, c.expected, ts.cursor.style)
			page := &ts.cursor.page_pin.node.data
			if c.expected.Default() {
				expectEqual(t, StyleDefaultId, ts.cursor.style_id)
				expectEqual(t, 0, page.styles.Count())
			} else {
				expectEqual(t, 1, page.styles.Count())
				expectEqual(t, 1, page.styles.RefCount(ts.cursor.style_id))
			}
			sc3QExpectExactRefs(t, ts)
		}()
	}
}

// Zig: test "term51: setAttribute: faint does not clear bold and reset_bold clears both"
func TestSc3_term51_setAttribute_faint_does_not_clear_bold_and_reset_bold_clears_both(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 4, 2, 0)
	defer s.Deinit()

	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	sc3SetAttr(t, s, sc3A(SGRAttributeTagFaint))
	expect(t, s.cursor.style.flags.bold, "s.cursor.style.flags.bold")
	expect(t, s.cursor.style.flags.faint, "s.cursor.style.flags.faint")
	sc3SetAttr(t, s, sc3A(SGRAttributeTagResetBold))
	expect(t, !s.cursor.style.flags.bold, "!s.cursor.style.flags.bold")
	expect(t, !s.cursor.style.flags.faint, "!s.cursor.style.flags.faint")
	expectEqual(t, StyleDefaultId, s.cursor.style_id)

	// reset_bold on a faint-only style also clears faint.
	sc3SetAttr(t, s, sc3A(SGRAttributeTagFaint))
	expect(t, s.cursor.style_id != StyleDefaultId, "s.cursor.style_id != style.default_id")
	sc3SetAttr(t, s, sc3A(SGRAttributeTagResetBold))
	expect(t, !s.cursor.style.flags.faint, "!s.cursor.style.flags.faint")
	expectEqual(t, StyleDefaultId, s.cursor.style_id)
}

// Zig: test "term51: setAttribute: reference counts across changes"
func TestSc3_term51_setAttribute_reference_counts_across_changes(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 6, 2, 0)
	defer s.Deinit()
	page := &s.cursor.page_pin.node.data

	// Setting the same thing twice keeps one style with one reference.
	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	bold_id := s.cursor.style_id
	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	expectEqual(t, bold_id, s.cursor.style_id)
	expectEqual(t, 1, page.styles.Count())
	expectEqual(t, 1, page.styles.RefCount(bold_id))

	// Two cells use the style.
	sc3Write(t, s, "ab")
	expectEqual(t, 3, page.styles.RefCount(bold_id))

	// A different style: the old one stays alive for the cells.
	sc3SetAttr(t, s, sc3A(SGRAttributeTagItalic))
	both_id := s.cursor.style_id
	expect(t, both_id != bold_id, "both_id != bold_id")
	expectEqual(t, 2, page.styles.Count())
	expectEqual(t, 2, page.styles.RefCount(bold_id))
	expectEqual(t, 1, page.styles.RefCount(both_id))
	sc3QExpectExactRefs(t, s)

	// Back to bold only: the id is found again and the other one dies.
	sc3SetAttr(t, s, sc3A(SGRAttributeTagResetItalic))
	expectEqual(t, bold_id, s.cursor.style_id)
	expectEqual(t, 1, page.styles.Count())
	expectEqual(t, 3, page.styles.RefCount(bold_id))
	expectEqual(t, 0, page.styles.RefCount(both_id))
	sc3QExpectExactRefs(t, s)

	// unknown does not touch the references.
	sc3SetAttr(t, s, SGRAttribute{
		Tag:     SGRAttributeTagUnknown,
		Unknown: SGRAttributeUnknown{full: []uint16{99}, partial: []uint16{99}},
	})
	expectEqual(t, bold_id, s.cursor.style_id)
	expectEqual(t, 3, page.styles.RefCount(bold_id))

	// unset: the cursor lets go, the cells keep theirs.
	sc3SetAttr(t, s, sc3A(SGRAttributeTagUnset))
	expectEqual(t, StyleDefaultId, s.cursor.style_id)
	expectEqual(t, 1, page.styles.Count())
	expectEqual(t, 2, page.styles.RefCount(bold_id))

	// A no-op on the default style does not add anything.
	sc3SetAttr(t, s, sc3A(SGRAttributeTagResetBold))
	sc3SetAttr(t, s, sc3A(SGRAttributeTagResetFg))
	sc3SetAttr(t, s, sc3AttrUnderline(SGRAttributeUnderlineNone))
	expectEqual(t, StyleDefaultId, s.cursor.style_id)
	expectEqual(t, 1, page.styles.Count())

	// Writing the same style again by attribute gets the same id.
	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	expectEqual(t, bold_id, s.cursor.style_id)
	expectEqual(t, 3, page.styles.RefCount(bold_id))
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: setAttribute: colours distinguish palette, rgb and none"
func TestSc3_term51_setAttribute_colours_distinguish_palette_rgb_and_none(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 6, 2, 0)
	defer s.Deinit()
	page := &s.cursor.page_pin.node.data

	// The same number as palette and as rgb are different styles.
	sc3SetAttr(t, s, sc3AttrFg256(1))
	palette_id := s.cursor.style_id
	sc3Write(t, s, "a")
	sc3SetAttr(t, s, sc3AttrFgRGB(1, 0, 0))
	expect(t, s.cursor.style_id != palette_id, "s.cursor.style_id != palette_id")
	expectEqual(t, 2, page.styles.Count())

	// The foreground and background of the same value are different too.
	sc3SetAttr(t, s, sc3A(SGRAttributeTagResetFg))
	sc3SetAttr(t, s, sc3AttrBgRGB(1, 0, 0))
	expect(t, s.cursor.style.fg_color.Tag == StyleColorNone, "s.cursor.style.fg_color == .none")
	expect(t, s.cursor.style.bg_color.Tag == StyleColorRgb, "s.cursor.style.bg_color == .rgb")
	expect(t, page.styles.Get(s.cursor.style_id).Eql(Style{
		bg_color: StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{r: 1, g: 0, b: 0}},
	}), "page.styles.get(page.memory, s.cursor.style_id).eql(.{ .bg_color = .{ .rgb = .{ .r = 1, .g = 0, .b = 0 } } })")
	// The palette style of the written cell and the cursor's style.
	expectEqual(t, 2, page.styles.Count())
	sc3QExpectExactRefs(t, s)
}

// sc3QFillStyles is term51QFillStyles: add distinct background-colour styles
// to the page until the set refuses more. Returns how many were added; they
// have a reference count of one and are ids 1.. when the set was empty.
func sc3QFillStyles(page *Page) uint {
	// Zig: n is a u24 that is @bitCast into the RGB of the background.
	var n uint32 = 1
	for {
		_, err := page.styles.Add(Style{
			bg_color: StyleColor{
				Tag: StyleColorRgb,
				Rgb: ColorRGB{r: uint8(n), g: uint8(n >> 8), b: uint8(n >> 16)},
			},
		})
		if err != nil {
			break
		}
		n += 1
	}
	return uint(n - 1)
}

// sc3QMaxStyles is term51QMaxStyles: grow the cursor page's style capacity
// to the maximum.
func sc3QMaxStyles(s *Screen) {
	const max = 1<<16 - 1
	for s.cursor.page_pin.node.data.capacity.styles < max {
		_, err := s.IncreaseCapacity(s.cursor.page_pin.node, Some(PageListIncreaseCapacityStyles))
		if err != nil {
			break
		}
	}
}

// Zig: test "term51: manualStyleUpdate: follows manual changes of the cursor style"
func TestSc3_term51_manualStyleUpdate_follows_manual_changes_of_the_cursor_style(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 2, 0)
	defer s.Deinit()
	page := &s.cursor.page_pin.node.data

	// The default style needs no update.
	zigTry(t, s.ManualStyleUpdate())
	expectEqual(t, StyleDefaultId, s.cursor.style_id)
	expectEqual(t, 0, page.styles.Count())

	// A manual change is picked up.
	s.cursor.style = Style{flags: styleFlags{bold: true}}
	expectEqual(t, StyleDefaultId, s.cursor.style_id)
	zigTry(t, s.ManualStyleUpdate())
	expect(t, s.cursor.style_id != StyleDefaultId, "s.cursor.style_id != style.default_id")
	expectEqual(t, 1, page.styles.Count())
	expectEqual(t, 1, page.styles.RefCount(s.cursor.style_id))
	expect(t, page.styles.Get(s.cursor.style_id).Eql(s.cursor.style), "page.styles.get(page.memory, s.cursor.style_id).eql(s.cursor.style)")
	sc3QExpectExactRefs(t, s)

	// Updating again without a change keeps a single reference.
	id := s.cursor.style_id
	zigTry(t, s.ManualStyleUpdate())
	expectEqual(t, id, s.cursor.style_id)
	expectEqual(t, 1, page.styles.RefCount(id))
	expectEqual(t, 1, page.styles.Count())

	// A different style releases the previous one.
	s.cursor.style = Style{flags: styleFlags{italic: true}}
	zigTry(t, s.ManualStyleUpdate())
	expectEqual(t, 1, page.styles.Count())
	expect(t, page.styles.Get(s.cursor.style_id).Eql(s.cursor.style), "page.styles.get(page.memory, s.cursor.style_id).eql(s.cursor.style)")
	expectEqual(t, 1, page.styles.RefCount(s.cursor.style_id))
	sc3QExpectExactRefs(t, s)

	// Back to the default releases and cannot fail.
	s.cursor.style = Style{}
	zigTry(t, s.ManualStyleUpdate())
	expectEqual(t, StyleDefaultId, s.cursor.style_id)
	expectEqual(t, 0, page.styles.Count())
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: manualStyleUpdate: styles used by cells outlive the cursor's reference"
func TestSc3_term51_manualStyleUpdate_styles_used_by_cells_outlive_the_cursor_s_reference(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 2, 0)
	defer s.Deinit()
	page := &s.cursor.page_pin.node.data

	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	sc3Write(t, s, "ab")
	bold_id := s.cursor.style_id
	expectEqual(t, 3, page.styles.RefCount(bold_id))

	// Change manually: the old style loses only the cursor's reference.
	s.cursor.style = Style{flags: styleFlags{italic: true}}
	zigTry(t, s.ManualStyleUpdate())
	expect(t, s.cursor.style_id != bold_id, "s.cursor.style_id != bold_id")
	expectEqual(t, 2, page.styles.RefCount(bold_id))
	expectEqual(t, 2, page.styles.Count())
	sc3QExpectExactRefs(t, s)

	// The cursor style set back to the cells' style shares their id.
	s.cursor.style = Style{flags: styleFlags{bold: true}}
	zigTry(t, s.ManualStyleUpdate())
	expectEqual(t, bold_id, s.cursor.style_id)
	expectEqual(t, 3, page.styles.RefCount(bold_id))
	expectEqual(t, 1, page.styles.Count())
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: setAttribute: a full style set grows the page capacity and keeps every style"
func TestSc3_term51_setAttribute_a_full_style_set_grows_the_page_capacity_and_keeps_every_style(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 10, 5, 0)
	defer s.Deinit()

	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	sc3Write(t, s, "AB")
	sc3SetAttr(t, s, sc3A(SGRAttributeTagUnset))
	node_before := s.cursor.page_pin.node
	bold_id := zigUnwrap(t, node_before.data.styles.Lookup(Style{flags: styleFlags{bold: true}}))
	expectEqual(t, 2, node_before.data.styles.RefCount(bold_id))
	cap_before := node_before.data.capacity.styles
	filled := sc3QFillStyles(&node_before.data)
	expect(t, filled > 0, "filled > 0")
	expectEqual(t, filled+1, node_before.data.styles.Count())

	sc3SetAttr(t, s, sc3A(SGRAttributeTagItalic))

	// The page was replaced with one with doubled style capacity, and the
	// unused styles that filled the set were dropped.
	page := &s.cursor.page_pin.node.data
	expect(t, s.cursor.page_pin.node != node_before, "s.cursor.page_pin.node != node_before")
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
	expectEqual(t, cap_before*2, page.capacity.styles)
	expect(t, s.cursor.style.flags.italic, "s.cursor.style.flags.italic")
	expect(t, s.cursor.style_id != StyleDefaultId, "s.cursor.style_id != style.default_id")
	expectEqual(t, 2, page.styles.Count())
	expectEqual(t, 1, page.styles.RefCount(s.cursor.style_id))
	bold_new := zigUnwrap(t, page.styles.Lookup(Style{flags: styleFlags{bold: true}}))
	expectEqual(t, 2, page.styles.RefCount(bold_new))
	sc3QExpectDump(t, s, sc3Screen(0, 0), "AB")
	expectEqual(t, 2, s.cursor.x)
	expectEqual(t, 0, s.cursor.y)
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: setAttribute: a set full of dead items is rehashed at the same capacity"
func TestSc3_term51_setAttribute_a_set_full_of_dead_items_is_rehashed_at_the_same_capacity(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 10, 5, 0)
	defer s.Deinit()

	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	sc3Write(t, s, "AB")
	sc3SetAttr(t, s, sc3A(SGRAttributeTagUnset))
	node_before := s.cursor.page_pin.node
	cap_before := node_before.data.capacity.styles
	page_before := &node_before.data
	filled := sc3QFillStyles(page_before)
	expectEqual(t, page_before.styles.layout.cap, uint(page_before.styles.next_id))

	// Release every filler but the last one so that most of the ids are
	// dead (they cannot be trimmed from the end because the last lives).
	for id := StyleId(2); uint(id) < filled+1; id++ {
		page_before.styles.Release(id)
	}
	expectEqual(t, 2, page_before.styles.Count())

	sc3SetAttr(t, s, sc3A(SGRAttributeTagItalic))

	page := &s.cursor.page_pin.node.data
	expect(t, s.cursor.page_pin.node != node_before, "s.cursor.page_pin.node != node_before")
	expectEqual(t, cap_before, page.capacity.styles)
	expectEqual(t, 2, page.styles.Count())
	expect(t, s.cursor.style.flags.italic, "s.cursor.style.flags.italic")
	expect(t, s.cursor.style_id != StyleDefaultId, "s.cursor.style_id != style.default_id")
	sc3QExpectDump(t, s, sc3Screen(0, 0), "AB")
	sc3QExpectExactRefs(t, s)
}

// sc3PageHasRow reports whether row is one of the elements of page.rows: the
// slice-element form of Zig's check that a row pointer lies inside
// page.memory.
func sc3PageHasRow(page *Page, row *PageRow) bool {
	for i := range page.rows {
		if &page.rows[i] == row {
			return true
		}
	}
	return false
}

// sc3PageHasCell reports whether cell is one of the elements of page.cells:
// the slice-element form of Zig's check that a cell pointer lies inside
// page.memory.
func sc3PageHasCell(page *Page, cell *PageCell) bool {
	for i := range page.cells {
		if &page.cells[i] == cell {
			return true
		}
	}
	return false
}

// Zig: test "term51: manualStyleUpdate: out of space at the maximum capacity splits the page and moves the cursor row"
func TestSc3_term51_manualStyleUpdate_out_of_space_at_the_maximum_capacity_splits_the_page_and_moves_the_cursor_row(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 100, 30, 0)
	defer s.Deinit()
	sc3QMaxStyles(s)
	expectEqual(t, 65535, s.cursor.page_pin.node.data.capacity.styles)

	// Bold + link cells on row 1, italic + link cells on row 28.
	s.CursorAbsolute(0, 1)
	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	sc3StartLink(t, s, "https://example.com", "id")
	sc3Write(t, s, "ab")
	s.CursorAbsolute(0, 28)
	s.cursor.pending_wrap = false
	sc3SetAttr(t, s, sc3A(SGRAttributeTagUnset))
	sc3SetAttr(t, s, sc3A(SGRAttributeTagItalic))
	sc3Write(t, s, "cd")
	sc3SetAttr(t, s, sc3A(SGRAttributeTagUnset))
	s.CursorAbsolute(5, 27)
	s.cursor.pending_wrap = false

	old_node := s.cursor.page_pin.node
	old_page := &old_node.data
	bold_id := zigUnwrap(t, old_page.styles.Lookup(Style{flags: styleFlags{bold: true}}))
	sc3QFillStyles(old_page)
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")

	// A style that does not exist yet needs a slot: there are none, the
	// capacity is at the maximum, so the page is split below row 27.
	sc3SetAttr(t, s, sc3AttrFg256(5))

	expect(t, s.pages.pages.first != s.pages.pages.last, "s.pages.pages.first != s.pages.pages.last")
	expect(t, s.pages.pages.first == old_node, "s.pages.pages.first == old_node")
	first := s.pages.pages.first
	last := s.pages.pages.last
	expectEqual(t, 27, first.data.size.rows)
	expectEqual(t, 3, last.data.size.rows)
	expect(t, s.cursor.page_pin.node == last, "s.cursor.page_pin.node == last")
	expectEqual(t, 5, s.cursor.x)
	expectEqual(t, 27, s.cursor.y)
	expectEqual(t, 30, s.pages.total_rows)

	// The style took effect on the new page.
	expect(t, s.cursor.style_id != StyleDefaultId, "s.cursor.style_id != style.default_id")
	expect(t, s.cursor.style.fg_color.Tag == StyleColorPalette && s.cursor.style.fg_color.Palette == 5, "s.cursor.style.fg_color.palette == 5")
	expectEqual(t, 2, last.data.styles.Count())
	expectEqual(t, 1, last.data.styles.RefCount(s.cursor.style_id))
	italic_id := zigUnwrap(t, last.data.styles.Lookup(Style{flags: styleFlags{italic: true}}))
	expectEqual(t, 2, last.data.styles.RefCount(italic_id))

	// The cursor's hyperlink moved with it: two cells plus the cursor.
	expectEqual(t, 1, last.data.hyperlink_set.Count())
	expectEqual(t, 3, last.data.hyperlink_set.RefCount(s.cursor.hyperlink_id))
	expectEqualStrings(t, "https://example.com", string(s.cursor.hyperlink.uri))

	// The old page keeps the bold cells and holds one reference less on the
	// link now that the cursor left.
	expectEqual(t, 2, first.data.styles.RefCount(bold_id))
	old_link := zigUnwrap(t, first.data.LookupHyperlink(&first.data.GetCells(first.data.GetRow(1))[0]))
	expectEqual(t, 2, first.data.hyperlink_set.RefCount(old_link))

	// Changed by ruling (project lead, 2026-10-06; docs/mapping.md
	// "Page-layer bookkeeping", term51 docs/upstream-crashes.md): upstream
	// leaves the cached row and cell pointers in the old page (the Zig test
	// asserts that); the port reloads them at the end of splitForCapacity.
	// Zig-only lines (upstream's stale state):
	//   try testing.expect(s.cursor.page_row != rac.row);
	//   try testing.expect(s.cursor.page_cell != rac.cell);
	//   (and the @intFromPtr range checks against old_page.memory)
	rac := s.cursor.page_pin.RowAndCell()
	expect(t, s.cursor.page_row == rac.row, "s.cursor.page_row == rac.row (ruling)")
	expect(t, s.cursor.page_cell == rac.cell, "s.cursor.page_cell == rac.cell (ruling)")

	// A reload fixes them.
	s.CursorReload()
	expect(t, s.cursor.page_row == rac.row, "s.cursor.page_row == rac.row")
	expect(t, s.cursor.page_cell == rac.cell, "s.cursor.page_cell == rac.cell")
	first.data.AssertIntegrity()
	last.data.AssertIntegrity()
	expectEqual(t, 'a', sc3QCell(t, s, 0, 1).content.codepoint)
	expectEqual(t, 'd', sc3QCell(t, s, 1, 28).content.codepoint)
	expectEqual(t, 1, last.data.styles.RefCount(s.cursor.style_id))
}

// sc3ExpectError is Zig's `try testing.expectError(expected, actual)`.
func sc3ExpectError(t *testing.T, want error, got error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

// Zig: test "term51: manualStyleUpdate: out of space on a one-row page cannot split"
func TestSc3_term51_manualStyleUpdate_out_of_space_on_a_one_row_page_cannot_split(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 10, 1, 0)
	defer s.Deinit()
	sc3QMaxStyles(s)
	sc3QFillStyles(&s.cursor.page_pin.node.data)

	sc3ExpectError(t, ErrOutOfSpace, s.SetAttribute(sc3A(SGRAttributeTagBold)))

	// The failure restores the (default) previous style.
	expect(t, s.cursor.style.Default(), "s.cursor.style.default()")
	expectEqual(t, StyleDefaultId, s.cursor.style_id)
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")

	// Directly: the manual style is the requested one, the id the default
	// (the documentation says the style is "reverted back to the default"
	// but only the id is).
	s.cursor.style = Style{flags: styleFlags{italic: true}}
	sc3ExpectError(t, ErrOutOfSpace, s.ManualStyleUpdate())
	expectEqual(t, StyleDefaultId, s.cursor.style_id)
	expect(t, s.cursor.style.flags.italic, "s.cursor.style.flags.italic")
	s.cursor.style = Style{}
}

// Zig: test "term51: setAttribute: a failed update with a full set cannot restore the previous style either"
func TestSc3_term51_setAttribute_a_failed_update_with_a_full_set_cannot_restore_the_previous_style_either(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 10, 1, 0)
	defer s.Deinit()
	sc3QMaxStyles(s)
	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	bold_id := s.cursor.style_id
	page := &s.cursor.page_pin.node.data
	sc3QFillStyles(page)
	expectEqual(t, 1, page.styles.RefCount(bold_id))

	sc3ExpectError(t, ErrOutOfSpace, s.SetAttribute(sc3A(SGRAttributeTagItalic)))

	// The update released bold (now dead: lookups ignore dead items), and
	// re-adding it needs a free slot like any other style, so restoring
	// fails too and the cursor degrades to the default style.
	expect(t, s.cursor.style.Default(), "s.cursor.style.default()")
	expectEqual(t, StyleDefaultId, s.cursor.style_id)
	expectEqual(t, 0, page.styles.RefCount(bold_id))
	expect(t, page == &s.cursor.page_pin.node.data, "page == &s.cursor.page_pin.node.data")
}

// Zig: test "term51: splitForCapacity: splits at the pin or one row below, whichever leaves less in the pinned half"
func TestSc3_term51_splitForCapacity_splits_at_the_pin_or_one_row_below_whichever_leaves_less_in_the_pinned_half(t *testing.T) {
	testStdCapacity(t)

	// A pin near the top: more rows below, so the split moves down one row
	// and the pinned row stays in the first page.
	func() {
		s := sc3Init(t, 100, 30, 0)
		defer s.Deinit()
		zigTry(t, s.splitForCapacity(zigUnwrap(t, s.pages.Pin(sc3Active(0, 2)))))
		expect(t, s.pages.pages.first != s.pages.pages.last, "s.pages.pages.first != s.pages.pages.last")
		expectEqual(t, 3, s.pages.pages.first.data.size.rows)
		expectEqual(t, 27, s.pages.pages.last.data.size.rows)
		expectEqual(t, 30, s.pages.total_rows)
		// The cursor at y=0 did not move.
		expect(t, s.cursor.page_pin.node == s.pages.pages.first, "s.cursor.page_pin.node == s.pages.pages.first.?")
		sc3QExpectExactRefs(t, s)
	}()

	// A pin near the bottom: more rows above, so the split is at the pin
	// and the pinned row goes to the new page.
	func() {
		s := sc3Init(t, 100, 30, 0)
		defer s.Deinit()
		zigTry(t, s.splitForCapacity(zigUnwrap(t, s.pages.Pin(sc3Active(0, 27)))))
		expectEqual(t, 27, s.pages.pages.first.data.size.rows)
		expectEqual(t, 3, s.pages.pages.last.data.size.rows)
		expect(t, s.cursor.page_pin.node == s.pages.pages.first, "s.cursor.page_pin.node == s.pages.pages.first.?")
		sc3QExpectExactRefs(t, s)
	}()

	// The first row: split below it.
	func() {
		s := sc3Init(t, 100, 30, 0)
		defer s.Deinit()
		zigTry(t, s.splitForCapacity(zigUnwrap(t, s.pages.Pin(sc3Active(0, 0)))))
		expectEqual(t, 1, s.pages.pages.first.data.size.rows)
		expectEqual(t, 29, s.pages.pages.last.data.size.rows)
	}()

	// The last row: split at it.
	func() {
		s := sc3Init(t, 100, 30, 0)
		defer s.Deinit()
		zigTry(t, s.splitForCapacity(zigUnwrap(t, s.pages.Pin(sc3Active(0, 29)))))
		expectEqual(t, 29, s.pages.pages.first.data.size.rows)
		expectEqual(t, 1, s.pages.pages.last.data.size.rows)
	}()
}

// Zig: test "term51: splitForCapacity: managed memory in a half counts towards that half"
func TestSc3_term51_splitForCapacity_managed_memory_in_a_half_counts_towards_that_half(t *testing.T) {
	testStdCapacity(t)

	// Control: without extra data a pin at y=14 splits below row 14.
	func() {
		s := sc3Init(t, 100, 30, 0)
		defer s.Deinit()
		zigTry(t, s.splitForCapacity(zigUnwrap(t, s.pages.Pin(sc3Active(0, 14)))))
		expectEqual(t, 15, s.pages.pages.first.data.size.rows)
		expectEqual(t, 15, s.pages.pages.last.data.size.rows)
	}()

	// With many graphemes above the pin, the upper half is the larger one
	// so the pinned row goes with the lower half.
	func() {
		s := sc3Init(t, 100, 30, 0)
		defer s.Deinit()
		for y := 0; y < 14; y++ {
			s.CursorAbsolute(0, SizeCellCountInt(y))
			for x := 0; x < 80; x++ {
				_ = s.cursor.page_cell
				cell := s.CursorCellRight(SizeCellCountInt(x))
				*cell = PageCellInit('x')
				zigTry(t, s.AppendGrapheme(cell, 0x301))
			}
		}
		s.CursorAbsolute(0, 0)
		g := s.cursor.page_pin.node.data.GraphemeCount()
		expectEqual(t, 14*80, g)

		zigTry(t, s.splitForCapacity(zigUnwrap(t, s.pages.Pin(sc3Active(0, 14)))))
		expectEqual(t, 14, s.pages.pages.first.data.size.rows)
		expectEqual(t, 16, s.pages.pages.last.data.size.rows)
		expectEqual(t, g, s.pages.pages.first.data.GraphemeCount())
		expectEqual(t, 0, s.pages.pages.last.data.GraphemeCount())
		sc3QExpectExactRefs(t, s)
	}()
}

// Zig: test "term51: splitForCapacity: a cursor on a moved row switches pages with its style and hyperlink"
func TestSc3_term51_splitForCapacity_a_cursor_on_a_moved_row_switches_pages_with_its_style_and_hyperlink(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 100, 30, 0)
	defer s.Deinit()

	s.CursorAbsolute(0, 28)
	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	sc3StartLink(t, s, "https://example.com", "id")
	sc3Write(t, s, "xy")
	// Cells above too.
	s.CursorAbsolute(0, 3)
	s.cursor.pending_wrap = false
	sc3Write(t, s, "pq")
	s.CursorAbsolute(2, 28)
	s.cursor.pending_wrap = false
	sc3QExpectExactRefs(t, s)

	old_node := s.cursor.page_pin.node
	zigTry(t, s.splitForCapacity(*s.cursor.page_pin))

	// Split at the cursor row: rows 28 and 29 are in the new page.
	expect(t, s.pages.pages.first == old_node, "s.pages.pages.first == old_node")
	expect(t, s.pages.pages.last != old_node, "s.pages.pages.last != old_node")
	expectEqual(t, 28, old_node.data.size.rows)
	expectEqual(t, 2, s.pages.pages.last.data.size.rows)
	expect(t, s.cursor.page_pin.node == s.pages.pages.last, "s.cursor.page_pin.node == s.pages.pages.last.?")
	expectEqual(t, 2, s.cursor.x)
	expectEqual(t, 28, s.cursor.y)
	expect(t, s.cursor.style.flags.bold, "s.cursor.style.flags.bold")

	// Changed by ruling (project lead, 2026-10-06; see the test above):
	// upstream's pointers are stale until a reload; the port's are already
	// reloaded by splitForCapacity. Zig-only lines (upstream's stale state):
	//   try testing.expect(s.cursor.page_row != rac.row);
	//   try testing.expect(s.cursor.page_cell != rac.cell);
	rac := s.cursor.page_pin.RowAndCell()
	expect(t, s.cursor.page_row == rac.row, "s.cursor.page_row == rac.row (ruling)")
	expect(t, s.cursor.page_cell == rac.cell, "s.cursor.page_cell == rac.cell (ruling)")
	s.CursorReload()
	expect(t, s.cursor.page_row == rac.row, "s.cursor.page_row == rac.row")
	expect(t, s.cursor.page_cell == rac.cell, "s.cursor.page_cell == rac.cell")

	last := s.pages.pages.last
	// The new page holds the style for the cells x, y and the cursor, and
	// the link for the two cells and the cursor.
	expectEqual(t, 3, last.data.styles.RefCount(s.cursor.style_id))
	expectEqual(t, 3, last.data.hyperlink_set.RefCount(s.cursor.hyperlink_id))
	// The old page keeps the two cells above and lost the cursor's refs.
	expectEqual(t, 1, old_node.data.styles.Count())
	expectEqual(t, 2, old_node.data.styles.RefCount(1))
	expectEqual(t, 2, old_node.data.hyperlink_set.RefCount(1))
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: splitForCapacity: a one-row page cannot be split"
func TestSc3_term51_splitForCapacity_a_one_row_page_cannot_be_split(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 10, 1, 0)
	defer s.Deinit()
	sc3ExpectError(t, ErrOutOfSpace, s.splitForCapacity(*s.cursor.page_pin))
	expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
}

// Zig: test "term51: splitForCapacity: layout equal halves split at the pin and pin y of zero is a no-op"
func TestSc3_term51_splitForCapacity_layout_equal_halves_split_at_the_pin_and_pin_y_of_zero_is_a_no_op(t *testing.T) {
	testStdCapacity(t)

	// With 10 columns and 2 rows both halves round to the same page size,
	// which is not "less", so the split is at the pin itself.
	func() {
		s := sc3Init(t, 10, 2, 0)
		defer s.Deinit()
		zigTry(t, s.splitForCapacity(zigUnwrap(t, s.pages.Pin(sc3Active(0, 1)))))
		expectEqual(t, 1, s.pages.pages.first.data.size.rows)
		expectEqual(t, 1, s.pages.pages.last.data.size.rows)
	}()

	// At y=0 splitting at the pin does nothing at all.
	func() {
		s := sc3Init(t, 10, 2, 0)
		defer s.Deinit()
		zigTry(t, s.splitForCapacity(zigUnwrap(t, s.pages.Pin(sc3Active(0, 0)))))
		expect(t, s.pages.pages.first == s.pages.pages.last, "s.pages.pages.first == s.pages.pages.last")
		expectEqual(t, 2, s.pages.pages.first.data.size.rows)
	}()
}

// sc3QSaved is term51QSaved: a saved cursor at (x, y) with distinctive other
// fields.
func sc3QSaved(x SizeCellCountInt, y SizeCellCountInt, pending_wrap bool) ScreenSavedCursor {
	charset := screenCharsetStateDefault()
	charset.gl = CharsetsSlotsG1
	return ScreenSavedCursor{
		x:            x,
		y:            y,
		style:        Style{flags: styleFlags{bold: true}},
		protected:    true,
		pending_wrap: pending_wrap,
		origin:       true,
		charset:      charset,
	}
}

// Zig: test "term51: resize: Resize defaults"
func TestSc3_term51_resize_Resize_defaults(t *testing.T) {
	testStdCapacity(t)
	// Zig: const r: Resize = .{ .cols = 3, .rows = 4 }; (screenResizeDefault
	// writes the Zig defaults of the other fields.)
	r := screenResizeDefault(3, 4)
	expectEqual(t, 3, r.cols)
	expectEqual(t, 4, r.rows)
	expect(t, r.reflow, "r.reflow")
	expectEqual(t, OSCParsersSemanticPromptRedrawFalse, r.prompt_redraw)
}

// sc3Resize is `try s.resize(.{ .cols = cols, .rows = rows })` with the Zig
// defaults of Resize (reflow = true, prompt_redraw = .false).
func sc3Resize(t *testing.T, s *Screen, cols SizeCellCountInt, rows SizeCellCountInt) {
	t.Helper()
	zigTry(t, s.Resize(screenResizeDefault(cols, rows)))
}

// Zig: test "term51: resize: the cursor style is released and re-added with exact reference counts"
func TestSc3_term51_resize_the_cursor_style_is_released_and_re_added_with_exact_reference_counts(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 1000)
	defer s.Deinit()
	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	sc3Write(t, s, "abc")

	noReflow := screenResizeDefault(40, 2)
	noReflow.reflow = false
	steps := []ScreenResize{
		screenResizeDefault(10, 3), // new page for more columns
		screenResizeDefault(4, 3),  // fewer columns, "abc" still fits
		screenResizeDefault(4, 6),  // more rows
		screenResizeDefault(4, 2),  // fewer rows
		screenResizeDefault(4, 2),  // no change at all
		noReflow,
	}
	for i, step := range steps {
		// Zig: errdefer std.debug.print("resize step {} failed\n", .{i});
		t.Logf("resize step %d", i)
		zigTry(t, s.Resize(step))
		expect(t, s.cursor.style.flags.bold, "s.cursor.style.flags.bold")
		expect(t, s.cursor.style_id != StyleDefaultId, "s.cursor.style_id != style.default_id")
		page := &s.cursor.page_pin.node.data
		expectEqual(t, 1, page.styles.Count())
		// Three cells and the cursor.
		expectEqual(t, 4, page.styles.RefCount(s.cursor.style_id))
		expectEqual(t, 3, s.cursor.x)
		sc3QExpectExactRefs(t, s)
	}
}

// Zig: test "term51: resize: a default cursor style stays default and cells keep their styles"
func TestSc3_term51_resize_a_default_cursor_style_stays_default_and_cells_keep_their_styles(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 1000)
	defer s.Deinit()
	sc3SetAttr(t, s, sc3A(SGRAttributeTagItalic))
	sc3Write(t, s, "ab")
	sc3SetAttr(t, s, sc3A(SGRAttributeTagUnset))

	sc3Resize(t, s, 9, 3)
	expectEqual(t, StyleDefaultId, s.cursor.style_id)
	expect(t, s.cursor.style.Default(), "s.cursor.style.default()")
	page := &s.cursor.page_pin.node.data
	expectEqual(t, 1, page.styles.Count())
	expectEqual(t, 2, page.styles.RefCount(sc3QCell(t, s, 0, 0).style_id))
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: resize: cells dropped without reflow release their styles"
func TestSc3_term51_resize_cells_dropped_without_reflow_release_their_styles(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 1000)
	defer s.Deinit()
	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	sc3Write(t, s, "abcde")
	sc3SetAttr(t, s, sc3A(SGRAttributeTagUnset))
	{
		page := &s.cursor.page_pin.node.data
		expectEqual(t, 5, page.styles.RefCount(sc3QCell(t, s, 0, 0).style_id))
	}

	// Truncated to three columns: d and e are gone.
	r := screenResizeDefault(3, 3)
	r.reflow = false
	zigTry(t, s.Resize(r))
	sc3QExpectDump(t, s, sc3Screen(0, 0), "abc")
	page := &s.cursor.page_pin.node.data
	expectEqual(t, 1, page.styles.Count())
	expectEqual(t, 3, page.styles.RefCount(sc3QCell(t, s, 0, 0).style_id))
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: resize: an explicit hyperlink is re-added and merges with the cells' link"
func TestSc3_term51_resize_an_explicit_hyperlink_is_re_added_and_merges_with_the_cells_link(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 1000)
	defer s.Deinit()
	sc3StartLink(t, s, "https://example.com/", "myid")
	sc3Write(t, s, "ab")
	old_link := s.cursor.hyperlink
	expectEqual(t, 0, s.cursor.hyperlink_implicit_id)

	sc3Resize(t, s, 10, 3)

	// A new heap copy with the same URI and explicit ID.
	expect(t, s.cursor.hyperlink != nil, "s.cursor.hyperlink != null")
	_ = old_link
	expectEqualStrings(t, "https://example.com/", string(s.cursor.hyperlink.uri))
	expectEqualStrings(t, "myid", string(s.cursor.hyperlink.id.Explicit))
	expect(t, s.cursor.hyperlink_id != 0, "s.cursor.hyperlink_id != 0")
	// Explicit IDs do not touch the implicit counter.
	expectEqual(t, 0, s.cursor.hyperlink_implicit_id)

	// One entry used by two cells and the cursor.
	page := &s.cursor.page_pin.node.data
	expectEqual(t, 1, page.hyperlink_set.Count())
	expectEqual(t, 3, page.hyperlink_set.RefCount(s.cursor.hyperlink_id))
	expectEqual(t, s.cursor.hyperlink_id, zigUnwrap(t, page.LookupHyperlink(sc3QCell(t, s, 0, 0))))
	sc3QExpectExactRefs(t, s)

	s.EndHyperlink()
	expectEqual(t, 1, page.hyperlink_set.Count())
	expectEqual(t, 2, page.hyperlink_set.RefCount(zigUnwrap(t, page.LookupHyperlink(sc3QCell(t, s, 0, 0)))))
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: resize: an implicit hyperlink gets a new implicit id and no longer matches its cells"
func TestSc3_term51_resize_an_implicit_hyperlink_gets_a_new_implicit_id_and_no_longer_matches_its_cells(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 1000)
	defer s.Deinit()
	sc3StartLinkImplicit(t, s, "https://example.com/")
	expectEqual(t, 0, s.cursor.hyperlink.id.Implicit)
	expectEqual(t, 1, s.cursor.hyperlink_implicit_id)
	sc3Write(t, s, "ab")

	sc3Resize(t, s, 10, 3)

	// The cursor's link was recreated with the next implicit id.
	expectEqualStrings(t, "https://example.com/", string(s.cursor.hyperlink.uri))
	expectEqual(t, 1, s.cursor.hyperlink.id.Implicit)
	expectEqual(t, 2, s.cursor.hyperlink_implicit_id)

	// So there are two entries in the page now: the cells' (implicit 0)
	// with two references and the cursor's (implicit 1) with one.
	page := &s.cursor.page_pin.node.data
	expectEqual(t, 2, page.hyperlink_set.Count())
	cell_id := zigUnwrap(t, page.LookupHyperlink(sc3QCell(t, s, 0, 0)))
	expect(t, cell_id != s.cursor.hyperlink_id, "cell_id != s.cursor.hyperlink_id")
	expectEqual(t, 2, page.hyperlink_set.RefCount(cell_id))
	expectEqual(t, 1, page.hyperlink_set.RefCount(s.cursor.hyperlink_id))
	sc3QExpectExactRefs(t, s)

	// Every further resize bumps the implicit id again, even without a
	// change of size.
	sc3Resize(t, s, 10, 3)
	expectEqual(t, 2, s.cursor.hyperlink.id.Implicit)
	expectEqual(t, 3, s.cursor.hyperlink_implicit_id)
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: resize: a hyperlink without cells and no hyperlink at all"
func TestSc3_term51_resize_a_hyperlink_without_cells_and_no_hyperlink_at_all(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 1000)
	defer s.Deinit()

	// No hyperlink: stays none.
	sc3Resize(t, s, 6, 3)
	expectEqual(t, 0, s.cursor.hyperlink_id)
	expect(t, s.cursor.hyperlink == nil, "s.cursor.hyperlink == null")
	expectEqual(t, 0, s.cursor.page_pin.node.data.hyperlink_set.Count())

	// A hyperlink without any cell using it.
	sc3StartLink(t, s, "https://a.example/", "a")
	sc3Resize(t, s, 7, 4)
	page := &s.cursor.page_pin.node.data
	expectEqual(t, 1, page.hyperlink_set.Count())
	expectEqual(t, 1, page.hyperlink_set.RefCount(s.cursor.hyperlink_id))
	sc3QExpectExactRefs(t, s)
	s.EndHyperlink()
	expectEqual(t, 0, page.hyperlink_set.Count())
}

// Zig: test "term51: resize: the saved cursor follows reflowed content"
func TestSc3_term51_resize_the_saved_cursor_follows_reflowed_content(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 1000)
	defer s.Deinit()
	sc3Write(t, s, "ABCDEFGH")
	expectEqual(t, 3, s.cursor.x)
	expectEqual(t, 1, s.cursor.y)
	s.saved_cursor = Some(sc3QSaved(1, 1, false))
	pins_before := s.pages.CountTrackedPins()

	sc3Resize(t, s, 10, 3)
	sc3QExpectDump(t, s, sc3Screen(0, 0), "ABCDEFGH")

	// 'G' moved from (1, 1) to (6, 0), and the cursor after 'H' to (8, 0).
	expectEqual(t, 6, zigUnwrap(t, s.saved_cursor).x)
	expectEqual(t, 0, zigUnwrap(t, s.saved_cursor).y)
	expectEqual(t, 8, s.cursor.x)
	expectEqual(t, 0, s.cursor.y)

	// Everything else of the saved cursor is untouched.
	sc := zigUnwrap(t, s.saved_cursor)
	expect(t, sc.style.flags.bold, "sc.style.flags.bold")
	expect(t, sc.protected, "sc.protected")
	expect(t, sc.origin, "sc.origin")
	expect(t, !sc.pending_wrap, "!sc.pending_wrap")
	expectEqual(t, CharsetsSlotsG1, sc.charset.gl)

	// The temporary tracked pin was released.
	expectEqual(t, pins_before, s.pages.CountTrackedPins())

	// Shrinking wraps it again.
	sc3Resize(t, s, 5, 3)
	expectEqual(t, 1, zigUnwrap(t, s.saved_cursor).x)
	expectEqual(t, 1, zigUnwrap(t, s.saved_cursor).y)
	expectEqual(t, pins_before, s.pages.CountTrackedPins())
}

// Zig: test "term51: resize: a saved cursor with pending wrap"
func TestSc3_term51_resize_a_saved_cursor_with_pending_wrap(t *testing.T) {
	testStdCapacity(t)

	// The pending wrap is dropped and the cursor moves one cell right when
	// the saved cursor is no longer in the last column.
	func() {
		s := sc3Init(t, 5, 3, 1000)
		defer s.Deinit()
		sc3Write(t, s, "ABCDE")
		s.saved_cursor = Some(sc3QSaved(4, 0, true))
		sc3Resize(t, s, 10, 3)
		expectEqual(t, 5, zigUnwrap(t, s.saved_cursor).x)
		expectEqual(t, 0, zigUnwrap(t, s.saved_cursor).y)
		expect(t, !zigUnwrap(t, s.saved_cursor).pending_wrap, "!s.saved_cursor.?.pending_wrap")
	}()

	// It is kept when the saved cursor is still in the last column.
	func() {
		s := sc3Init(t, 5, 3, 1000)
		defer s.Deinit()
		sc3Write(t, s, "ABCDE")
		s.saved_cursor = Some(sc3QSaved(4, 0, true))
		sc3Resize(t, s, 5, 4)
		expectEqual(t, 4, zigUnwrap(t, s.saved_cursor).x)
		expectEqual(t, 0, zigUnwrap(t, s.saved_cursor).y)
		expect(t, zigUnwrap(t, s.saved_cursor).pending_wrap, "s.saved_cursor.?.pending_wrap")
	}()

	// Without pending wrap the x is not adjusted.
	func() {
		s := sc3Init(t, 5, 3, 1000)
		defer s.Deinit()
		sc3Write(t, s, "ABCDE")
		s.saved_cursor = Some(sc3QSaved(4, 0, false))
		sc3Resize(t, s, 10, 3)
		expectEqual(t, 4, zigUnwrap(t, s.saved_cursor).x)
		expect(t, !zigUnwrap(t, s.saved_cursor).pending_wrap, "!s.saved_cursor.?.pending_wrap")
	}()

	// The main cursor's pending wrap is not adjusted by resize.
	func() {
		s := sc3Init(t, 5, 3, 1000)
		defer s.Deinit()
		sc3Write(t, s, "ABCDE")
		expect(t, s.cursor.pending_wrap, "s.cursor.pending_wrap")
		sc3Resize(t, s, 10, 3)
		expectEqual(t, 4, s.cursor.x)
		expect(t, s.cursor.pending_wrap, "s.cursor.pending_wrap")
	}()
}

// Zig: test "term51: resize: a saved cursor pushed out of the active area goes to the top-left"
func TestSc3_term51_resize_a_saved_cursor_pushed_out_of_the_active_area_goes_to_the_top_left(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 5, 1000)
	defer s.Deinit()
	sc3Write(t, s, "1\n2\n3\n4\n5")
	s.saved_cursor = Some(sc3QSaved(3, 0, true))
	pins_before := s.pages.CountTrackedPins()

	sc3Resize(t, s, 5, 3)
	sc3QExpectDump(t, s, sc3Active(0, 0), "3\n4\n5")
	sc3QExpectDump(t, s, sc3Screen(0, 0), "1\n2\n3\n4\n5")
	expectEqual(t, 0, zigUnwrap(t, s.saved_cursor).x)
	expectEqual(t, 0, zigUnwrap(t, s.saved_cursor).y)
	expect(t, !zigUnwrap(t, s.saved_cursor).pending_wrap, "!s.saved_cursor.?.pending_wrap")
	expectEqual(t, pins_before, s.pages.CountTrackedPins())
	// The main cursor stayed on its row.
	expectEqual(t, 1, s.cursor.x)
	expectEqual(t, 2, s.cursor.y)

	// A saved cursor that stays inside is moved with its content: the row
	// "4" is now at y=1.
	s.saved_cursor = Some(sc3QSaved(0, 1, false))
	sc3Resize(t, s, 5, 2)
	sc3QExpectDump(t, s, sc3Active(0, 0), "4\n5")
	expectEqual(t, 0, zigUnwrap(t, s.saved_cursor).x)
	expectEqual(t, 0, zigUnwrap(t, s.saved_cursor).y)
}

// Zig: test "term51: resize: a saved cursor outside the screen is left alone"
func TestSc3_term51_resize_a_saved_cursor_outside_the_screen_is_left_alone(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 1000)
	defer s.Deinit()
	sc3Write(t, s, "ABCDE")
	pins_before := s.pages.CountTrackedPins()

	// x beyond the columns.
	s.saved_cursor = Some(sc3QSaved(9, 1, true))
	sc3Resize(t, s, 10, 3)
	expectEqual(t, 9, zigUnwrap(t, s.saved_cursor).x)
	expectEqual(t, 1, zigUnwrap(t, s.saved_cursor).y)
	expect(t, zigUnwrap(t, s.saved_cursor).pending_wrap, "s.saved_cursor.?.pending_wrap")

	// y beyond the rows.
	s.saved_cursor = Some(sc3QSaved(1, 7, true))
	sc3Resize(t, s, 10, 3)
	expectEqual(t, 1, zigUnwrap(t, s.saved_cursor).x)
	expectEqual(t, 7, zigUnwrap(t, s.saved_cursor).y)
	expect(t, zigUnwrap(t, s.saved_cursor).pending_wrap, "s.saved_cursor.?.pending_wrap")
	expectEqual(t, pins_before, s.pages.CountTrackedPins())

	// No saved cursor stays none.
	s.saved_cursor = None[ScreenSavedCursor]()
	sc3Resize(t, s, 12, 3)
	expect(t, !s.saved_cursor.Valid, "s.saved_cursor == null")
	expectEqual(t, pins_before, s.pages.CountTrackedPins())
}

// Zig: test "term51: resize: reflow false lays out rows as they are and clamps the saved cursor"
func TestSc3_term51_resize_reflow_false_lays_out_rows_as_they_are_and_clamps_the_saved_cursor(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 1000)
	defer s.Deinit()
	sc3Write(t, s, "ABCDEFGH")

	// No reflow: the soft-wrapped lines stay separate rows. Growing the
	// columns reallocates the page and that drops the wrap flags of the
	// rows, so a later reflowing resize cannot join them either.
	r := screenResizeDefault(10, 3)
	r.reflow = false
	zigTry(t, s.Resize(r))
	sc3QExpectDump(t, s, sc3Screen(0, 0), "ABCDE\nFGH")
	expect(t, !sc3QRow(t, s, 0).wrap, "!term51QRow(&s, 0).wrap")
	expect(t, !sc3QRow(t, s, 1).wrap_continuation, "!term51QRow(&s, 1).wrap_continuation")
	sc3Resize(t, s, 20, 3)
	sc3QExpectDump(t, s, sc3Screen(0, 0), "ABCDE\nFGH")

	// Shrinking without reflow truncates the rows and the tracked pins are
	// clamped to the last column, so the saved cursor ends up there.
	ts := sc3Init(t, 10, 3, 1000)
	defer ts.Deinit()
	sc3Write(t, ts, "ABCDEFGHIJ")
	ts.saved_cursor = Some(sc3QSaved(8, 0, false))
	r = screenResizeDefault(5, 3)
	r.reflow = false
	zigTry(t, ts.Resize(r))
	sc3QExpectDump(t, ts, sc3Screen(0, 0), "ABCDE")
	expectEqual(t, 4, zigUnwrap(t, ts.saved_cursor).x)
	expectEqual(t, 0, zigUnwrap(t, ts.saved_cursor).y)
	// The cursor (pending wrap at x=9) is clamped too.
	expectEqual(t, 4, ts.cursor.x)
	expectEqual(t, 0, ts.cursor.y)
	expect(t, ts.cursor.pending_wrap, "t.cursor.pending_wrap")
}

// sc3ResizeRedraw is `try s.resize(.{ .cols = cols, .rows = rows,
// .prompt_redraw = redraw })` with the Zig default reflow = true.
func sc3ResizeRedraw(t *testing.T, s *Screen, cols SizeCellCountInt, rows SizeCellCountInt, redraw OSCParsersSemanticPromptRedraw) {
	t.Helper()
	r := screenResizeDefault(cols, rows)
	r.prompt_redraw = redraw
	zigTry(t, s.Resize(r))
}

// sc3Prompt is `s.cursorSetSemanticContent(.{ .prompt = kind })`.
func sc3Prompt(s *Screen, kind OSCParsersSemanticPromptPromptKind) {
	s.CursorSetSemanticContent(ScreenCursorSetSemanticContent{
		Tag:    ScreenCursorSetSemanticContentTagPrompt,
		Prompt: kind,
	})
}

// sc3Output is `s.cursorSetSemanticContent(.output)`.
func sc3Output(s *Screen) {
	s.CursorSetSemanticContent(ScreenCursorSetSemanticContent{Tag: ScreenCursorSetSemanticContentTagOutput})
}

// sc3InputClearExplicit is `s.cursorSetSemanticContent(.{ .input =
// .clear_explicit })`.
func sc3InputClearExplicit(s *Screen) {
	s.CursorSetSemanticContent(ScreenCursorSetSemanticContent{
		Tag:   ScreenCursorSetSemanticContentTagInput,
		Input: ScreenCursorSetSemanticContentInputClearExplicit,
	})
}

// Zig: test "term51: resize: prompt_redraw false never clears"
func TestSc3_term51_resize_prompt_redraw_false_never_clears(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 10, 4, 5)
	defer s.Deinit()
	sc3Write(t, s, "out\n")
	sc3Prompt(s, OSCParsersSemanticPromptPromptKindInitial)
	sc3Write(t, s, "$ ")
	sc3InputClearExplicit(s)
	sc3Write(t, s, "cmd")

	sc3ResizeRedraw(t, s, 20, 4, OSCParsersSemanticPromptRedrawFalse)
	sc3QExpectDump(t, s, sc3Screen(0, 0), "out\n$ cmd")
}

// Zig: test "term51: resize: prompt_redraw last clears the cursor row only and releases its managed memory"
func TestSc3_term51_resize_prompt_redraw_last_clears_the_cursor_row_only_and_releases_its_managed_memory(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 10, 4, 5)
	defer s.Deinit()
	sc3Write(t, s, "out\n")
	sc3Prompt(s, OSCParsersSemanticPromptPromptKindInitial)
	sc3Write(t, s, "$ ")
	sc3InputClearExplicit(s)
	sc3SetAttr(t, s, sc3A(SGRAttributeTagBold))
	sc3StartLink(t, s, "https://example.com", "id")
	sc3Write(t, s, "cmd")
	s.EndHyperlink()
	sc3SetAttr(t, s, sc3A(SGRAttributeTagUnset))
	{
		c := zigUnwrap(t, s.pages.GetCell(sc3Active(0, 1)))
		zigTry(t, c.node.data.AppendGrapheme(c.row, c.cell, 0x301))
	}
	page := &s.cursor.page_pin.node.data
	expectEqual(t, 1, page.styles.Count())
	expectEqual(t, 1, page.hyperlink_set.Count())
	expectEqual(t, 1, page.GraphemeCount())

	sc3ResizeRedraw(t, s, 20, 4, OSCParsersSemanticPromptRedrawLast)

	sc3QExpectDump(t, s, sc3Screen(0, 0), "out")
	// The cursor does not move and the row keeps its prompt marking.
	expectEqual(t, 5, s.cursor.x)
	expectEqual(t, 1, s.cursor.y)
	expectEqual(t, PageRowSemanticPromptPrompt, sc3QRow(t, s, 1).semantic_prompt)
	page2 := &s.cursor.page_pin.node.data
	expectEqual(t, 0, page2.styles.Count())
	expectEqual(t, 0, page2.hyperlink_set.Count())
	expectEqual(t, 0, page2.GraphemeCount())
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: resize: prompt_redraw true clears from the first prompt line to the end of the screen"
func TestSc3_term51_resize_prompt_redraw_true_clears_from_the_first_prompt_line_to_the_end_of_the_screen(t *testing.T) {
	testStdCapacity(t)

	for _, mode := range []OSCParsersSemanticPromptRedraw{OSCParsersSemanticPromptRedrawTrue, OSCParsersSemanticPromptRedrawLast} {
		func() {
			s := sc3Init(t, 10, 5, 5)
			defer s.Deinit()
			sc3Write(t, s, "old\n")
			sc3Prompt(s, OSCParsersSemanticPromptPromptKindInitial)
			sc3Write(t, s, "$ a\n")
			sc3Prompt(s, OSCParsersSemanticPromptPromptKindContinuation)
			sc3Write(t, s, "> b")
			sc3Output(s)
			sc3Write(t, s, "\ntail")
			sc3QMove(s, 3, 2)
			sc3InputClearExplicit(s)
			sc3QExpectDump(t, s, sc3Screen(0, 0), "old\n$ a\n> b\ntail")

			sc3ResizeRedraw(t, s, 20, 5, mode)

			// `.true` clears the whole prompt (rows 1 and 2) and everything
			// below it including the output row after the cursor. `.last`
			// clears only the cursor row.
			switch mode {
			case OSCParsersSemanticPromptRedrawTrue:
				sc3QExpectDump(t, s, sc3Screen(0, 0), "old")
			case OSCParsersSemanticPromptRedrawLast:
				sc3QExpectDump(t, s, sc3Screen(0, 0), "old\n$ a\n\ntail")
			default:
				panic("unreachable")
			}
			expectEqual(t, 3, s.cursor.x)
			expectEqual(t, 2, s.cursor.y)
			sc3QExpectExactRefs(t, s)
		}()
	}
}

// Zig: test "term51: resize: prompt_redraw true walks up past unmarked rows to the prompt"
func TestSc3_term51_resize_prompt_redraw_true_walks_up_past_unmarked_rows_to_the_prompt(t *testing.T) {
	testStdCapacity(t)
	for _, mode := range []OSCParsersSemanticPromptRedraw{OSCParsersSemanticPromptRedrawTrue, OSCParsersSemanticPromptRedrawLast} {
		func() {
			s := sc3Init(t, 10, 4, 5)
			defer s.Deinit()
			sc3Prompt(s, OSCParsersSemanticPromptPromptKindInitial)
			sc3Write(t, s, "$ a")
			// The next row is not marked as a continuation.
			sc3Output(s)
			sc3Write(t, s, "\n")
			sc3InputClearExplicit(s)
			sc3Write(t, s, "b")
			expectEqual(t, PageRowSemanticPromptNone, sc3QRow(t, s, 1).semantic_prompt)

			sc3ResizeRedraw(t, s, 20, 4, mode)
			switch mode {
			case OSCParsersSemanticPromptRedrawTrue:
				sc3QExpectDump(t, s, sc3Screen(0, 0), "")
			case OSCParsersSemanticPromptRedrawLast:
				sc3QExpectDump(t, s, sc3Screen(0, 0), "$ a")
			default:
				panic("unreachable")
			}
		}()
	}
}

// Zig: test "term51: resize: prompt_redraw does nothing when the cursor is on command output"
func TestSc3_term51_resize_prompt_redraw_does_nothing_when_the_cursor_is_on_command_output(t *testing.T) {
	testStdCapacity(t)
	for _, mode := range []OSCParsersSemanticPromptRedraw{OSCParsersSemanticPromptRedrawTrue, OSCParsersSemanticPromptRedrawLast} {
		func() {
			s := sc3Init(t, 10, 4, 5)
			defer s.Deinit()
			sc3Prompt(s, OSCParsersSemanticPromptPromptKindInitial)
			sc3Write(t, s, "$ a")
			sc3Output(s)
			sc3Write(t, s, "\nout")

			sc3ResizeRedraw(t, s, 20, 4, mode)
			sc3QExpectDump(t, s, sc3Screen(0, 0), "$ a\nout")
		}()
	}
}

// Zig: test "term51: resize: prompt_redraw also applies when the cursor content is prompt"
func TestSc3_term51_resize_prompt_redraw_also_applies_when_the_cursor_content_is_prompt(t *testing.T) {
	testStdCapacity(t)
	for _, mode := range []OSCParsersSemanticPromptRedraw{OSCParsersSemanticPromptRedrawTrue, OSCParsersSemanticPromptRedrawLast} {
		func() {
			s := sc3Init(t, 10, 4, 5)
			defer s.Deinit()
			sc3Write(t, s, "out\n")
			sc3Prompt(s, OSCParsersSemanticPromptPromptKindInitial)
			sc3Write(t, s, "$ ")
			expectEqual(t, PageCellSemanticContentPrompt, s.cursor.semantic_content)

			sc3ResizeRedraw(t, s, 20, 4, mode)
			sc3QExpectDump(t, s, sc3Screen(0, 0), "out")
		}()
	}
}

// Zig: test "term51: resize: prompt_redraw true without any prompt row clears nothing"
func TestSc3_term51_resize_prompt_redraw_true_without_any_prompt_row_clears_nothing(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 10, 4, 5)
	defer s.Deinit()
	sc3Write(t, s, "abc\ndef")
	// The cursor claims to be on input but no row is marked as a prompt.
	s.cursor.semantic_content = PageCellSemanticContentInput

	sc3ResizeRedraw(t, s, 20, 4, OSCParsersSemanticPromptRedrawTrue)
	sc3QExpectDump(t, s, sc3Screen(0, 0), "abc\ndef")

	// `.last` does not need a prompt row.
	sc3ResizeRedraw(t, s, 21, 4, OSCParsersSemanticPromptRedrawLast)
	sc3QExpectDump(t, s, sc3Screen(0, 0), "abc")
}

// Zig: test "term51: resize: a screen without scrollback erases the history created by fewer rows"
func TestSc3_term51_resize_a_screen_without_scrollback_erases_the_history_created_by_fewer_rows(t *testing.T) {
	testStdCapacity(t)

	func() {
		s := sc3Init(t, 5, 5, 0)
		defer s.Deinit()
		sc3Write(t, s, "1\n2\n3\n4\n5")
		sc3Resize(t, s, 5, 3)
		sc3QExpectDump(t, s, sc3Screen(0, 0), "3\n4\n5")
		expectEqual(t, 3, s.pages.total_rows)
		expectEqual(t, 1, s.cursor.x)
		expectEqual(t, 2, s.cursor.y)
		sc3QExpectExactRefs(t, s)
	}()

	// With scrollback the history is kept.
	func() {
		s := sc3Init(t, 5, 5, 1000)
		defer s.Deinit()
		sc3Write(t, s, "1\n2\n3\n4\n5")
		sc3Resize(t, s, 5, 3)
		sc3QExpectDump(t, s, sc3Screen(0, 0), "1\n2\n3\n4\n5")
		expectEqual(t, 5, s.pages.total_rows)
	}()
}

// Zig: test "term51: resize: the cursor row is kept when rows and columns grow"
func TestSc3_term51_resize_the_cursor_row_is_kept_when_rows_and_columns_grow(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 1000)
	defer s.Deinit()
	sc3Write(t, s, "ab\ncd")

	sc3Resize(t, s, 8, 6)
	sc3QExpectDump(t, s, sc3Screen(0, 0), "ab\ncd")
	expectEqual(t, 2, s.cursor.x)
	expectEqual(t, 1, s.cursor.y)
	expectEqual(t, 6, s.pages.total_rows)
	rac := s.cursor.page_pin.RowAndCell()
	expect(t, s.cursor.page_row == rac.row, "s.cursor.page_row == rac.row")
	expect(t, s.cursor.page_cell == rac.cell, "s.cursor.page_cell == rac.cell")
}

// sc3QFillGraphemes is term51QFillGraphemes: fill the grapheme storage of the
// cursor page by giving single-codepoint graphemes to the cells of rows 1..
// until the page refuses. Returns the number of cells that were given a
// grapheme.
func sc3QFillGraphemes(t *testing.T, s *Screen) uint {
	t.Helper()
	page := &s.cursor.page_pin.node.data
	var n uint
	for y := uint32(1); y < uint32(s.pages.rows); y++ {
		row := sc3QRow(t, s, y)
		for x := 0; x < int(s.pages.cols); x++ {
			cell := sc3QCell(t, s, SizeCellCountInt(x), y)
			*cell = PageCellInit('x')
			if err := page.AppendGrapheme(row, cell, 0x301); err != nil {
				*cell = PageCell{}
				return n
			}
			n += 1
		}
	}
	return n
}

// Zig: test "term51: appendGrapheme: grows a cell's grapheme across chunk boundaries"
func TestSc3_term51_appendGrapheme_grows_a_cell_s_grapheme_across_chunk_boundaries(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 10, 2, 0)
	defer s.Deinit()
	sc3Write(t, s, "ab")
	page := &s.cursor.page_pin.node.data
	cell := sc3QCell(t, s, 0, 0)
	expect(t, !s.cursor.page_row.grapheme, "!s.cursor.page_row.grapheme")

	var expected [12]uint32
	for i := range expected {
		expected[i] = uint32(0x300 + i + 1)
	}

	// 1..12 codepoints: every multiple of the chunk length (4) needs a new,
	// larger chunk.
	for n := 1; n < 13; n++ {
		zigTry(t, s.AppendGrapheme(cell, expected[n-1]))
		expectEqualSlices(t, expected[0:n], zigUnwrap(t, page.LookupGrapheme(cell)))
		expect(t, cell.HasGrapheme(), "cell.hasGrapheme()")
		expectEqual(t, PageCellContentTagCodepointGrapheme, cell.content_tag)
		expectEqual(t, 'a', cell.content.codepoint)
		expect(t, s.cursor.page_row.grapheme, "s.cursor.page_row.grapheme")
		expectEqual(t, 1, page.GraphemeCount())
	}

	// The neighbour is untouched.
	expect(t, !sc3QCell(t, s, 1, 0).HasGrapheme(), "!term51QCell(&s, 1, 0).hasGrapheme()")
	expect(t, page == &s.cursor.page_pin.node.data, "page == &s.cursor.page_pin.node.data")
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: appendGrapheme: any cell of the cursor row, left of, at or right of the cursor"
func TestSc3_term51_appendGrapheme_any_cell_of_the_cursor_row_left_of_at_or_right_of_the_cursor(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 10, 2, 0)
	defer s.Deinit()
	sc3Write(t, s, "abcdef")
	s.CursorAbsolute(3, 0)
	page := &s.cursor.page_pin.node.data

	zigTry(t, s.AppendGrapheme(s.CursorCellLeft(3), 0x301))
	zigTry(t, s.AppendGrapheme(s.cursor.page_cell, 0x302))
	zigTry(t, s.AppendGrapheme(s.CursorCellRight(2), 0x303))
	expectEqual(t, 3, page.GraphemeCount())
	expectEqualSlices(t, []uint32{0x301}, zigUnwrap(t, page.LookupGrapheme(sc3QCell(t, s, 0, 0))))
	expectEqualSlices(t, []uint32{0x302}, zigUnwrap(t, page.LookupGrapheme(sc3QCell(t, s, 3, 0))))
	expectEqualSlices(t, []uint32{0x303}, zigUnwrap(t, page.LookupGrapheme(sc3QCell(t, s, 5, 0))))
	expect(t, !sc3QCell(t, s, 1, 0).HasGrapheme(), "!term51QCell(&s, 1, 0).hasGrapheme()")
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: appendGrapheme: out of grapheme storage grows the page and finds the cell again"
func TestSc3_term51_appendGrapheme_out_of_grapheme_storage_grows_the_page_and_finds_the_cell_again(t *testing.T) {
	testStdCapacity(t)

	// The cell is left of, at, and right of the cursor.
	for _, target := range []int{0, 2, 4} {
		func() {
			s := sc3Init(t, 10, 24, 0)
			defer s.Deinit()
			sc3Write(t, s, "abcde")
			s.CursorAbsolute(2, 0)
			s.cursor.pending_wrap = false

			node_before := s.cursor.page_pin.node
			bytes_before := node_before.data.capacity.grapheme_bytes
			filled := sc3QFillGraphemes(t, s)
			expect(t, filled > 0, "filled > 0")
			expectEqual(t, filled, node_before.data.GraphemeCount())

			// The page really is full: a direct append fails.
			sc3ExpectError(t, ErrOutOfMemory,
				node_before.data.AppendGrapheme(s.cursor.page_row, sc3QCell(t, s, SizeCellCountInt(target), 0), 0x301))

			zigTry(t, s.AppendGrapheme(sc3QCell(t, s, SizeCellCountInt(target), 0), 0x301))

			// A new page with doubled grapheme storage that has everything.
			page := &s.cursor.page_pin.node.data
			expect(t, s.cursor.page_pin.node != node_before, "s.cursor.page_pin.node != node_before")
			expectEqual(t, bytes_before*2, page.capacity.grapheme_bytes)
			expectEqual(t, filled+1, page.GraphemeCount())
			expectEqualSlices(t, []uint32{0x301}, zigUnwrap(t, page.LookupGrapheme(sc3QCell(t, s, SizeCellCountInt(target), 0))))
			expectEqual(t, uint32("abcde"[target]), sc3QCell(t, s, SizeCellCountInt(target), 0).content.codepoint)
			expectEqual(t, 2, s.cursor.x)
			expectEqual(t, 0, s.cursor.y)
			expect(t, s.cursor.page_row.grapheme, "s.cursor.page_row.grapheme")
			sc3QExpectExactRefs(t, s)
		}()
	}
}

// Zig: test "term51: appendGrapheme: out of storage while extending a full chunk"
func TestSc3_term51_appendGrapheme_out_of_storage_while_extending_a_full_chunk(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 10, 3, 0)
	defer s.Deinit()
	sc3Write(t, s, "ab")

	// The test configuration starts with 512 grapheme bytes, but the chunk
	// allocator is made of whole bitmaps of 64 chunks (1024 bytes), so the
	// first doubling would not add any chunk. Start from 1024 bytes.
	_, err := s.IncreaseCapacity(s.cursor.page_pin.node, Some(PageListIncreaseCapacityGraphemeBytes))
	zigTry(t, err)
	node_before := s.cursor.page_pin.node
	page := &node_before.data
	bytes_before := page.capacity.grapheme_bytes
	expectEqual(t, 1024, bytes_before)

	// A cell of the cursor row with exactly four codepoints: one full chunk.
	cell := sc3QCell(t, s, 0, 0)
	for _, cp := range []uint32{0x301, 0x302, 0x303, 0x304} {
		zigTry(t, s.AppendGrapheme(cell, cp))
	}
	expectEqual(t, 4, len(zigUnwrap(t, page.LookupGrapheme(cell))))

	// Exhaust the chunk allocator: give every cell of rows 1 and 2 as long
	// a grapheme as the page allows, one codepoint at a time.
	for y := 1; y < 3; y++ {
		for x := 0; x < 10; x++ {
			c := sc3QCell(t, s, SizeCellCountInt(x), uint32(y))
			*c = PageCellInit('y')
			var n uint
			for {
				if err := page.AppendGrapheme(sc3QRow(t, s, uint32(y)), c, 0x305); err != nil {
					break
				}
				n += 1
			}
			if n == 0 {
				*c = PageCell{}
			}
		}
	}

	// The fifth codepoint needs a two chunk allocation: the page can't.
	sc3ExpectError(t, ErrOutOfMemory, page.AppendGrapheme(s.cursor.page_row, cell, 0x307))
	expectEqual(t, 4, len(zigUnwrap(t, page.LookupGrapheme(cell))))
	graphemes := page.GraphemeCount()

	// The Screen grows the page and appends.
	zigTry(t, s.AppendGrapheme(cell, 0x307))
	new_page := &s.cursor.page_pin.node.data
	expect(t, s.cursor.page_pin.node != node_before, "s.cursor.page_pin.node != node_before")
	expectEqual(t, graphemes, new_page.GraphemeCount())
	expectEqualSlices(t,
		[]uint32{0x301, 0x302, 0x303, 0x304, 0x307},
		zigUnwrap(t, new_page.LookupGrapheme(sc3QCell(t, s, 0, 0))))
	expectEqual(t, bytes_before*2, new_page.capacity.grapheme_bytes)
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: startHyperlink: explicit id copies the strings and stores one reference"
func TestSc3_term51_startHyperlink_explicit_id_copies_the_strings_and_stores_one_reference(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 10, 3, 0)
	defer s.Deinit()
	page := &s.cursor.page_pin.node.data
	expectEqual(t, 0, s.cursor.hyperlink_id)
	expect(t, s.cursor.hyperlink == nil, "s.cursor.hyperlink == null")

	uri_buf := []uint8("https://example.com/a")
	id_buf := []uint8("link-id")
	zigTry(t, s.StartHyperlink(uri_buf, Some(id_buf)))
	uri_buf[0] = 'X'
	id_buf[0] = 'X'

	// The cursor has its own heap copy.
	link := s.cursor.hyperlink
	expectEqualStrings(t, "https://example.com/a", string(link.uri))
	expect(t, &link.uri[0] != &uri_buf[0], "link.uri.ptr != @as([*]const u8, &uri_buf)")
	expectEqualStrings(t, "link-id", string(link.id.Explicit))
	expect(t, &link.id.Explicit[0] != &id_buf[0], "link.id.explicit.ptr != @as([*]const u8, &id_buf)")
	expect(t, s.cursor.hyperlink_id != 0, "s.cursor.hyperlink_id != 0")

	// And the page has a third one with a single reference.
	entry := page.hyperlink_set.Get(s.cursor.hyperlink_id)
	expectEqualStrings(t, "https://example.com/a", string(entry.uri))
	expectEqualStrings(t, "link-id", string(entry.id.Explicit))
	expectEqual(t, 1, page.hyperlink_set.Count())
	expectEqual(t, 1, page.hyperlink_set.RefCount(s.cursor.hyperlink_id))
	expectEqual(t, 0, page.HyperlinkCount())

	// An explicit id never touches the implicit counter.
	expectEqual(t, 0, s.cursor.hyperlink_implicit_id)
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: startHyperlink: implicit ids count up, one per implicit start"
func TestSc3_term51_startHyperlink_implicit_ids_count_up_one_per_implicit_start(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 10, 3, 0)
	defer s.Deinit()
	page := &s.cursor.page_pin.node.data

	sc3StartLinkImplicit(t, s, "https://a/")
	expectEqual(t, 0, s.cursor.hyperlink.id.Implicit)
	expectEqual(t, 1, s.cursor.hyperlink_implicit_id)
	sc3QExpectExactRefs(t, s)

	// The same URI again is a new implicit link.
	sc3StartLinkImplicit(t, s, "https://a/")
	expectEqual(t, 1, s.cursor.hyperlink.id.Implicit)
	expectEqual(t, 2, s.cursor.hyperlink_implicit_id)
	// Nothing used the first link, so only one is alive.
	expectEqual(t, 1, page.hyperlink_set.Count())

	// An explicit link in between does not consume an implicit id.
	sc3StartLink(t, s, "https://b/", "x")
	expectEqual(t, 2, s.cursor.hyperlink_implicit_id)
	sc3StartLinkImplicit(t, s, "https://c/")
	expectEqual(t, 2, s.cursor.hyperlink.id.Implicit)
	expectEqual(t, 3, s.cursor.hyperlink_implicit_id)

	// The counter can be at its largest value minus one.
	s.cursor.hyperlink_implicit_id = ^SizeOffsetInt(0) - 1
	sc3StartLinkImplicit(t, s, "https://d/")
	expectEqual(t, ^SizeOffsetInt(0)-1, s.cursor.hyperlink.id.Implicit)
	expectEqual(t, ^SizeOffsetInt(0), s.cursor.hyperlink_implicit_id)
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: startHyperlink: a new link ends the previous one and cells keep theirs"
func TestSc3_term51_startHyperlink_a_new_link_ends_the_previous_one_and_cells_keep_theirs(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 10, 3, 0)
	defer s.Deinit()
	page := &s.cursor.page_pin.node.data

	sc3StartLink(t, s, "https://a/", "a")
	sc3Write(t, s, "xy")
	a_id := s.cursor.hyperlink_id
	expectEqual(t, 3, page.hyperlink_set.RefCount(a_id))

	sc3StartLink(t, s, "https://b/", "b")
	expect(t, s.cursor.hyperlink_id != a_id, "s.cursor.hyperlink_id != a_id")
	expectEqualStrings(t, "https://b/", string(s.cursor.hyperlink.uri))
	// The first link now only has the cells' references.
	expectEqual(t, 2, page.hyperlink_set.RefCount(a_id))
	expectEqual(t, 2, page.hyperlink_set.Count())
	sc3QExpectExactRefs(t, s)

	// Starting an identical explicit link finds the existing entry and adds
	// one reference for the cursor.
	sc3StartLink(t, s, "https://a/", "a")
	expectEqual(t, a_id, s.cursor.hyperlink_id)
	expectEqual(t, 3, page.hyperlink_set.RefCount(a_id))
	// The unused "b" link died when it was replaced.
	expectEqual(t, 1, page.hyperlink_set.Count())

	// Same URI but a different explicit id is a different link, and the
	// same id with a different URI too.
	sc3StartLink(t, s, "https://a/", "other")
	expect(t, s.cursor.hyperlink_id != a_id, "s.cursor.hyperlink_id != a_id")
	sc3StartLink(t, s, "https://z/", "a")
	expect(t, s.cursor.hyperlink_id != a_id, "s.cursor.hyperlink_id != a_id")
	expectEqual(t, 2, page.hyperlink_set.RefCount(a_id))
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: startHyperlink: a full hyperlink set grows hyperlink_bytes"
func TestSc3_term51_startHyperlink_a_full_hyperlink_set_grows_hyperlink_bytes(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 10, 3, 0)
	defer s.Deinit()
	node_before := s.cursor.page_pin.node
	bytes_before := node_before.data.capacity.hyperlink_bytes

	// The standard page has room for two hyperlinks (ids 1 and 2).
	sc3StartLink(t, s, "https://a/", "a")
	sc3Write(t, s, "x")
	sc3StartLink(t, s, "https://b/", "b")
	sc3Write(t, s, "y")
	expect(t, s.cursor.page_pin.node == node_before, "s.cursor.page_pin.node == node_before")
	expectEqual(t, bytes_before, node_before.data.capacity.hyperlink_bytes)

	// The third needs a larger set.
	sc3StartLink(t, s, "https://c/", "c")
	page := &s.cursor.page_pin.node.data
	expect(t, s.cursor.page_pin.node != node_before, "s.cursor.page_pin.node != node_before")
	expectEqual(t, bytes_before*2, page.capacity.hyperlink_bytes)
	expectEqual(t, 3, page.hyperlink_set.Count())
	expectEqualStrings(t, "https://c/", string(s.cursor.hyperlink.uri))
	expectEqual(t, 2, page.HyperlinkCount())
	sc3QExpectDump(t, s, sc3Screen(0, 0), "xy")
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: startHyperlink: a set full of dead links is rehashed at the same capacity"
func TestSc3_term51_startHyperlink_a_set_full_of_dead_links_is_rehashed_at_the_same_capacity(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 10, 3, 0)
	defer s.Deinit()
	node_before := s.cursor.page_pin.node
	bytes_before := node_before.data.capacity.hyperlink_bytes

	sc3StartLink(t, s, "https://a/", "a")
	sc3Write(t, s, "x")
	sc3StartLink(t, s, "https://b/", "b")
	sc3Write(t, s, "y")

	// Kill link "a" (id 1) by clearing its cell: it is not the last item so
	// it cannot be trimmed, and a new link finds the set full of mostly
	// dead items.
	{
		page := &s.cursor.page_pin.node.data
		expectEqual(t, 2, page.hyperlink_set.Count())
		s.ClearCells(page, sc3QRow(t, s, 0), page.GetCells(sc3QRow(t, s, 0))[0:1])
		expectEqual(t, 1, page.hyperlink_set.Count())
	}

	sc3StartLink(t, s, "https://c/", "c")
	page := &s.cursor.page_pin.node.data
	expect(t, s.cursor.page_pin.node != node_before, "s.cursor.page_pin.node != node_before")
	expectEqual(t, bytes_before, page.capacity.hyperlink_bytes)
	expectEqual(t, 2, page.hyperlink_set.Count())
	expectEqualStrings(t, "https://c/", string(s.cursor.hyperlink.uri))
	// Only "b" remains of the cells' links.
	expectEqual(t, 1, page.HyperlinkCount())
	expect(t, page.LookupHyperlink(sc3QCell(t, s, 1, 0)).Valid, "page.lookupHyperlink(term51QCell(&s, 1, 0)) != null")
	expect(t, !page.LookupHyperlink(sc3QCell(t, s, 0, 0)).Valid, "page.lookupHyperlink(term51QCell(&s, 0, 0)) == null")
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: startHyperlink: strings that do not fit grow string_bytes"
func TestSc3_term51_startHyperlink_strings_that_do_not_fit_grow_string_bytes(t *testing.T) {
	testStdCapacity(t)

	// uri (63 chunks) plus a one-chunk id exactly fit the 64 chunks.
	func() {
		s := sc3Init(t, 10, 3, 0)
		defer s.Deinit()
		node_before := s.cursor.page_pin.node
		uri := strings.Repeat("u", 2016)
		sc3StartLink(t, s, uri, "x")
		expect(t, s.cursor.page_pin.node == node_before, "s.cursor.page_pin.node == node_before")
		expectEqual(t, 2048, node_before.data.capacity.string_bytes)
		expectEqual(t, 2016, len(s.cursor.hyperlink.uri))
		sc3QExpectExactRefs(t, s)
	}()

	// With a two-chunk id the total is 65 chunks: the page is replaced by one
	// with twice the string bytes.
	func() {
		s := sc3Init(t, 10, 3, 0)
		defer s.Deinit()
		node_before := s.cursor.page_pin.node
		uri := strings.Repeat("u", 2016)
		id := strings.Repeat("i", 33)
		sc3StartLink(t, s, uri, id)
		expect(t, s.cursor.page_pin.node != node_before, "s.cursor.page_pin.node != node_before")
		page := &s.cursor.page_pin.node.data
		expectEqual(t, 4096, page.capacity.string_bytes)
		expectEqual(t, 1, page.hyperlink_set.Count())
		expectEqualStrings(t, uri, string(s.cursor.hyperlink.uri))
		expectEqualStrings(t, id, string(s.cursor.hyperlink.id.Explicit))
		sc3QExpectExactRefs(t, s)
	}()
}

// Zig: test "term51: startHyperlinkOnce: starts a link and reports each page capacity error"
func TestSc3_term51_startHyperlinkOnce_starts_a_link_and_reports_each_page_capacity_error(t *testing.T) {
	testStdCapacity(t)

	// Success: ends the previous link, stores a copy.
	func() {
		s := sc3Init(t, 10, 3, 0)
		defer s.Deinit()
		zigTry(t, s.startHyperlinkOnce(Hyperlink{
			id:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: 7},
			uri: []uint8("https://a/"),
		}))
		expectEqual(t, 7, s.cursor.hyperlink.id.Implicit)
		// It does not use the screen's implicit counter.
		expectEqual(t, 0, s.cursor.hyperlink_implicit_id)
		first := s.cursor.hyperlink_id
		page := &s.cursor.page_pin.node.data
		expectEqual(t, 1, page.hyperlink_set.RefCount(first))

		zigTry(t, s.startHyperlinkOnce(Hyperlink{
			id:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdExplicit, Explicit: []uint8("b")},
			uri: []uint8("https://b/"),
		}))
		expectEqualStrings(t, "https://b/", string(s.cursor.hyperlink.uri))
		expectEqual(t, 1, page.hyperlink_set.Count())
		sc3QExpectExactRefs(t, s)
	}()

	// SetOutOfMemory: two live links fill the set.
	func() {
		s := sc3Init(t, 10, 3, 0)
		defer s.Deinit()
		sc3StartLink(t, s, "https://a/", "a")
		sc3Write(t, s, "x")
		sc3StartLink(t, s, "https://b/", "b")
		sc3Write(t, s, "y")
		node := s.cursor.page_pin.node

		sc3ExpectError(t, ErrSetOutOfMemory, s.startHyperlinkOnce(Hyperlink{
			id:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdExplicit, Explicit: []uint8("c")},
			uri: []uint8("https://c/"),
		}))
		// The previous link was ended and nothing was added.
		expectEqual(t, 0, s.cursor.hyperlink_id)
		expect(t, s.cursor.hyperlink == nil, "s.cursor.hyperlink == null")
		expect(t, s.cursor.page_pin.node == node, "s.cursor.page_pin.node == node")
		expectEqual(t, 2, node.data.hyperlink_set.Count())
		sc3QExpectExactRefs(t, s)
	}()

	// SetNeedsRehash.
	func() {
		s := sc3Init(t, 10, 3, 0)
		defer s.Deinit()
		sc3StartLink(t, s, "https://a/", "a")
		sc3Write(t, s, "x")
		sc3StartLink(t, s, "https://b/", "b")
		sc3Write(t, s, "y")
		page := &s.cursor.page_pin.node.data
		s.ClearCells(page, sc3QRow(t, s, 0), page.GetCells(sc3QRow(t, s, 0))[0:1])

		sc3ExpectError(t, ErrSetNeedsRehash, s.startHyperlinkOnce(Hyperlink{
			id:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdExplicit, Explicit: []uint8("c")},
			uri: []uint8("https://c/"),
		}))
		expectEqual(t, 0, s.cursor.hyperlink_id)
		expect(t, s.cursor.hyperlink == nil, "s.cursor.hyperlink == null")
	}()

	// StringsOutOfMemory, from the uri and from the id.
	func() {
		s := sc3Init(t, 10, 3, 0)
		defer s.Deinit()
		page := &s.cursor.page_pin.node.data
		// Occupy 60 of the 64 string chunks.
		_, err := page.string_alloc.Alloc(60 * 32)
		zigTry(t, err)

		// The uri needs 5 chunks.
		sc3ExpectError(t, ErrStringsOutOfMemory, s.startHyperlinkOnce(Hyperlink{
			id:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: 0},
			uri: []uint8(strings.Repeat("u", 150)),
		}))
		expect(t, s.cursor.hyperlink == nil, "s.cursor.hyperlink == null")

		// The uri fits (4 chunks) but the id (1 chunk) then does not, and the
		// uri allocation is given back: a second attempt fails the same way.
		for i := 0; i < 3; i++ {
			sc3ExpectError(t, ErrStringsOutOfMemory, s.startHyperlinkOnce(Hyperlink{
				id:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdExplicit, Explicit: []uint8("id")},
				uri: []uint8(strings.Repeat("u", 128)),
			}))
		}
		// That uri alone (implicit id) fits exactly because it was freed.
		zigTry(t, s.startHyperlinkOnce(Hyperlink{
			id:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: 0},
			uri: []uint8(strings.Repeat("u", 128)),
		}))
		expect(t, s.cursor.hyperlink != nil, "s.cursor.hyperlink != null")
	}()

	// The heap allocation fails: no page changes.
	// Zig-only: var s = try init(testing.allocator, .{ .cols = 10, .rows = 3, .max_scrollback = 0 });
	// Zig-only: defer s.deinit();
	// Zig-only: var fa = std.testing.FailingAllocator.init(testing.allocator, .{ .fail_index = 0 });
	// Zig-only: s.alloc = fa.allocator();
	// Zig-only: try testing.expectError(error.OutOfMemory, s.startHyperlinkOnce(.{ .uri = "https://a/", .id = .{ .implicit = 0 } }));
	// Zig-only: s.alloc = testing.allocator;
	// Zig-only: try testing.expect(s.cursor.hyperlink == null);
	// Zig-only: try testing.expectEqual(@as(usize, 0), s.cursor.page_pin.node.data.hyperlink_set.count());
}

// Zig: test "term51: endHyperlink: releases only the cursor's reference and is idempotent"
func TestSc3_term51_endHyperlink_releases_only_the_cursor_s_reference_and_is_idempotent(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 10, 3, 0)
	defer s.Deinit()
	page := &s.cursor.page_pin.node.data

	// Nothing to end.
	s.EndHyperlink()
	s.EndHyperlink()
	expectEqual(t, 0, s.cursor.hyperlink_id)
	expect(t, s.cursor.hyperlink == nil, "s.cursor.hyperlink == null")

	sc3StartLink(t, s, "https://a/", "a")
	sc3Write(t, s, "xy")
	id := s.cursor.hyperlink_id
	expectEqual(t, 3, page.hyperlink_set.RefCount(id))

	s.EndHyperlink()
	expectEqual(t, 0, s.cursor.hyperlink_id)
	expect(t, s.cursor.hyperlink == nil, "s.cursor.hyperlink == null")
	expectEqual(t, 2, page.hyperlink_set.RefCount(id))
	expectEqual(t, 1, page.hyperlink_set.Count())

	// Safe to call again.
	s.EndHyperlink()
	expectEqual(t, 2, page.hyperlink_set.RefCount(id))

	// Writing after the end does not add hyperlinks.
	sc3Write(t, s, "z")
	expectEqual(t, 2, page.HyperlinkCount())

	// A link without cells dies.
	sc3StartLink(t, s, "https://b/", "b")
	s.EndHyperlink()
	expectEqual(t, 1, page.hyperlink_set.Count())
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: cursorSetHyperlink: marks the cursor cell and counts the reference"
func TestSc3_term51_cursorSetHyperlink_marks_the_cursor_cell_and_counts_the_reference(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 10, 3, 0)
	defer s.Deinit()
	page := &s.cursor.page_pin.node.data

	sc3StartLink(t, s, "https://a/", "a")
	id := s.cursor.hyperlink_id
	expect(t, !s.cursor.page_row.hyperlink, "!s.cursor.page_row.hyperlink")
	expect(t, !s.cursor.page_cell.hyperlink, "!s.cursor.page_cell.hyperlink")

	zigTry(t, s.CursorSetHyperlink())
	expect(t, s.cursor.page_cell.hyperlink, "s.cursor.page_cell.hyperlink")
	expect(t, s.cursor.page_row.hyperlink, "s.cursor.page_row.hyperlink")
	expectEqual(t, id, zigUnwrap(t, page.LookupHyperlink(s.cursor.page_cell)))
	expectEqual(t, 1, page.HyperlinkCount())
	expectEqual(t, 2, page.hyperlink_set.RefCount(id))
	sc3QExpectExactRefs(t, s)

	// Setting the same link on the same cell again changes nothing.
	zigTry(t, s.CursorSetHyperlink())
	expectEqual(t, 1, page.HyperlinkCount())
	expectEqual(t, 2, page.hyperlink_set.RefCount(id))
	sc3QExpectExactRefs(t, s)

	// Another cell.
	s.CursorRight(1)
	zigTry(t, s.CursorSetHyperlink())
	expectEqual(t, 2, page.HyperlinkCount())
	expectEqual(t, 3, page.hyperlink_set.RefCount(id))

	// A different link replaces the cell's link and releases the old one.
	sc3StartLink(t, s, "https://b/", "b")
	b_id := s.cursor.hyperlink_id
	s.CursorLeft(1)
	zigTry(t, s.CursorSetHyperlink())
	expectEqual(t, b_id, zigUnwrap(t, page.LookupHyperlink(s.cursor.page_cell)))
	expectEqual(t, 1, page.hyperlink_set.RefCount(id))
	expectEqual(t, 2, page.hyperlink_set.RefCount(b_id))
	expectEqual(t, 2, page.HyperlinkCount())
	sc3QExpectExactRefs(t, s)

	// On another row.
	s.CursorDown(1)
	zigTry(t, s.CursorSetHyperlink())
	expect(t, s.cursor.page_row.hyperlink, "s.cursor.page_row.hyperlink")
	expectEqual(t, 3, page.HyperlinkCount())
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: cursorSetHyperlink: a full cell map grows hyperlink_bytes and keeps every link"
func TestSc3_term51_cursorSetHyperlink_a_full_cell_map_grows_hyperlink_bytes_and_keeps_every_link(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 80, 24, 0)
	defer s.Deinit()
	node_before := s.cursor.page_pin.node
	cap_before := node_before.data.HyperlinkCapacity()
	bytes_before := node_before.data.capacity.hyperlink_bytes
	strings_before := node_before.data.capacity.string_bytes
	expectEqual(t, 64, cap_before)

	// An implicit link: the capacity growth must keep the very same id.
	sc3StartLinkImplicit(t, s, "https://a/")

	// 64 cells fill the map.
	for x := 0; x < 64; x++ {
		s.CursorAbsolute(SizeCellCountInt(x), 0)
		zigTry(t, s.CursorSetHyperlink())
	}
	expect(t, s.cursor.page_pin.node == node_before, "s.cursor.page_pin.node == node_before")
	expectEqual(t, 64, node_before.data.HyperlinkCount())

	// The 65th does not fit.
	s.CursorAbsolute(64, 0)
	zigTry(t, s.CursorSetHyperlink())
	expect(t, s.cursor.page_pin.node != node_before, "s.cursor.page_pin.node != node_before")
	page := &s.cursor.page_pin.node.data
	expectEqual(t, bytes_before*2, page.capacity.hyperlink_bytes)
	expectEqual(t, cap_before*2, page.HyperlinkCapacity())
	expectEqual(t, strings_before, page.capacity.string_bytes)
	expectEqual(t, 65, page.HyperlinkCount())
	// One entry: 65 cells and the cursor.
	expectEqual(t, 1, page.hyperlink_set.Count())
	expectEqual(t, 66, page.hyperlink_set.RefCount(s.cursor.hyperlink_id))
	// The capacity growth did not consume an implicit id.
	expectEqual(t, 1, s.cursor.hyperlink_implicit_id)
	expectEqual(t, 0, s.cursor.hyperlink.id.Implicit)
	expect(t, s.cursor.page_cell.hyperlink, "s.cursor.page_cell.hyperlink")
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: cursorSetHyperlink: a uri that fills the strings also grows string_bytes"
func TestSc3_term51_cursorSetHyperlink_a_uri_that_fills_the_strings_also_grows_string_bytes(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 80, 24, 0)
	defer s.Deinit()
	node_before := s.cursor.page_pin.node

	// 64 chunks of 32 bytes minus a bit: the page string storage is full
	// after this single link.
	uri := strings.Repeat("a", PageStringBytesDefault-8)
	sc3StartLinkImplicit(t, s, uri)
	expect(t, s.cursor.page_pin.node == node_before, "s.cursor.page_pin.node == node_before")
	expectEqual(t, 2048, node_before.data.capacity.string_bytes)

	for x := 0; x < 65; x++ {
		s.CursorAbsolute(SizeCellCountInt(x), 0)
		zigTry(t, s.CursorSetHyperlink())
	}

	page := &s.cursor.page_pin.node.data
	expectEqual(t, 4096, page.capacity.string_bytes)
	expectEqual(t, 384, page.capacity.hyperlink_bytes)
	expectEqual(t, 65, page.HyperlinkCount())
	expectEqual(t, 66, page.hyperlink_set.RefCount(s.cursor.hyperlink_id))
	expectEqualStrings(t, uri, string(s.cursor.hyperlink.uri))
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: cursorSetSemanticContent: output, input and prompt kinds"
func TestSc3_term51_cursorSetSemanticContent_output_input_and_prompt_kinds(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 10, 3, 0)
	defer s.Deinit()
	s.CursorAbsolute(2, 1)
	sc_click := s.semantic_prompt.click
	expect(t, !s.semantic_prompt.seen, "!s.semantic_prompt.seen")
	expectEqual(t, PageCellSemanticContentOutput, s.cursor.semantic_content)
	expect(t, !s.cursor.semantic_content_clear_eol, "!s.cursor.semantic_content_clear_eol")

	inputClearEol := ScreenCursorSetSemanticContent{
		Tag:   ScreenCursorSetSemanticContentTagInput,
		Input: ScreenCursorSetSemanticContentInputClearEol,
	}

	// input with an explicit clear.
	sc3InputClearExplicit(s)
	expectEqual(t, PageCellSemanticContentInput, s.cursor.semantic_content)
	expect(t, !s.cursor.semantic_content_clear_eol, "!s.cursor.semantic_content_clear_eol")
	expectEqual(t, PageRowSemanticPromptNone, s.cursor.page_row.semantic_prompt)
	expect(t, !s.semantic_prompt.seen, "!s.semantic_prompt.seen")

	// input that clears at the end of the line.
	s.CursorSetSemanticContent(inputClearEol)
	expectEqual(t, PageCellSemanticContentInput, s.cursor.semantic_content)
	expect(t, s.cursor.semantic_content_clear_eol, "s.cursor.semantic_content_clear_eol")
	expectEqual(t, PageRowSemanticPromptNone, s.cursor.page_row.semantic_prompt)
	expect(t, !s.semantic_prompt.seen, "!s.semantic_prompt.seen")

	// output resets the flag.
	sc3Output(s)
	expectEqual(t, PageCellSemanticContentOutput, s.cursor.semantic_content)
	expect(t, !s.cursor.semantic_content_clear_eol, "!s.cursor.semantic_content_clear_eol")
	expect(t, !s.semantic_prompt.seen, "!s.semantic_prompt.seen")

	// Every prompt kind.
	kinds := []struct {
		kind OSCParsersSemanticPromptPromptKind
		row  PageRowSemanticPrompt
	}{
		{kind: OSCParsersSemanticPromptPromptKindInitial, row: PageRowSemanticPromptPrompt},
		{kind: OSCParsersSemanticPromptPromptKindRight, row: PageRowSemanticPromptPrompt},
		{kind: OSCParsersSemanticPromptPromptKindContinuation, row: PageRowSemanticPromptPromptContinuation},
		{kind: OSCParsersSemanticPromptPromptKindSecondary, row: PageRowSemanticPromptPromptContinuation},
	}
	for _, k := range kinds {
		// Start from input with clear_eol so the reset is visible and from
		// a row that has the other marking.
		s.CursorSetSemanticContent(inputClearEol)
		if k.row == PageRowSemanticPromptPrompt {
			s.cursor.page_row.semantic_prompt = PageRowSemanticPromptPromptContinuation
		} else {
			s.cursor.page_row.semantic_prompt = PageRowSemanticPromptPrompt
		}
		s.semantic_prompt.seen = false

		sc3Prompt(s, k.kind)
		expect(t, s.semantic_prompt.seen, "s.semantic_prompt.seen")
		expectEqual(t, PageCellSemanticContentPrompt, s.cursor.semantic_content)
		expect(t, !s.cursor.semantic_content_clear_eol, "!s.cursor.semantic_content_clear_eol")
		expectEqual(t, k.row, s.cursor.page_row.semantic_prompt)
	}

	// The click mode, the cursor position and the other rows are untouched.
	expectEqual(t, sc_click, s.semantic_prompt.click)
	expectEqual(t, 2, s.cursor.x)
	expectEqual(t, 1, s.cursor.y)
	expectEqual(t, PageRowSemanticPromptNone, sc3QRow(t, s, 0).semantic_prompt)
	expectEqual(t, PageRowSemanticPromptNone, sc3QRow(t, s, 2).semantic_prompt)

	// Leaving the prompt keeps the row marking, only the cursor changes.
	sc3Output(s)
	expectEqual(t, PageRowSemanticPromptPromptContinuation, s.cursor.page_row.semantic_prompt)
	sc3InputClearExplicit(s)
	expectEqual(t, PageRowSemanticPromptPromptContinuation, s.cursor.page_row.semantic_prompt)
}

// Zig: test "term51: cursorSetSemanticContent: new cells take the cursor's content"
func TestSc3_term51_cursorSetSemanticContent_new_cells_take_the_cursor_s_content(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 10, 3, 0)
	defer s.Deinit()

	sc3Prompt(s, OSCParsersSemanticPromptPromptKindInitial)
	sc3Write(t, s, "$")
	sc3InputClearExplicit(s)
	sc3Write(t, s, "a")
	sc3Output(s)
	sc3Write(t, s, "o")
	expectEqual(t, PageCellSemanticContentPrompt, sc3QCell(t, s, 0, 0).semantic_content)
	expectEqual(t, PageCellSemanticContentInput, sc3QCell(t, s, 1, 0).semantic_content)
	expectEqual(t, PageCellSemanticContentOutput, sc3QCell(t, s, 2, 0).semantic_content)
}

// sc3ExpectPoint is Zig's `expectEqual` of two point.Point values: the tags
// are equal and so are the coordinates of the active field.
func sc3ExpectPoint(t *testing.T, want Point, got Point) {
	t.Helper()
	expectEqual(t, want.Tag, got.Tag)
	switch want.Tag {
	case PointTagActive:
		expectEqual(t, want.Active, got.Active)
	case PointTagViewport:
		expectEqual(t, want.Viewport, got.Viewport)
	case PointTagScreen:
		expectEqual(t, want.Screen, got.Screen)
	case PointTagHistory:
		expectEqual(t, want.History, got.History)
	}
}

// Zig: test "term51: resize: the selection and other tracked pins follow reflowed content"
func TestSc3_term51_resize_the_selection_and_other_tracked_pins_follow_reflowed_content(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 1000)
	defer s.Deinit()
	sc3Write(t, s, "ABCDEFGH")

	// B..H across the soft wrap, and a pin of our own on F.
	zigTry(t, s.Select(Some(SelectionInit(
		zigUnwrap(t, s.pages.Pin(sc3Active(1, 0))),
		zigUnwrap(t, s.pages.Pin(sc3Active(2, 1))),
		false,
	))))
	mine, err := s.pages.TrackPin(zigUnwrap(t, s.pages.Pin(sc3Active(0, 1))))
	zigTry(t, err)
	defer s.pages.UntrackPin(mine)
	pins_before := s.pages.CountTrackedPins()

	sc3Resize(t, s, 10, 3)
	{
		sel := zigUnwrap(t, s.selection)
		sc3ExpectPoint(t, sc3Active(1, 0), zigUnwrap(t, s.pages.PointFromPin(PointTagActive, sel.Start())))
		sc3ExpectPoint(t, sc3Active(7, 0), zigUnwrap(t, s.pages.PointFromPin(PointTagActive, sel.End())))
		sc3ExpectPoint(t, sc3Active(5, 0), zigUnwrap(t, s.pages.PointFromPin(PointTagActive, *mine)))
		expectEqual(t, pins_before, s.pages.CountTrackedPins())
	}

	// Four columns: ABCD / EFGH.
	sc3Resize(t, s, 4, 3)
	sc3QExpectDump(t, s, sc3Screen(0, 0), "ABCD\nEFGH")
	{
		sel := zigUnwrap(t, s.selection)
		sc3ExpectPoint(t, sc3Active(1, 0), zigUnwrap(t, s.pages.PointFromPin(PointTagActive, sel.Start())))
		sc3ExpectPoint(t, sc3Active(3, 1), zigUnwrap(t, s.pages.PointFromPin(PointTagActive, sel.End())))
		sc3ExpectPoint(t, sc3Active(1, 1), zigUnwrap(t, s.pages.PointFromPin(PointTagActive, *mine)))
		expectEqual(t, pins_before, s.pages.CountTrackedPins())
	}
	sc3QExpectExactRefs(t, s)
}

// Zig: test "term51: resize: a selection pushed into the history stays valid"
func TestSc3_term51_resize_a_selection_pushed_into_the_history_stays_valid(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 5, 1000)
	defer s.Deinit()
	sc3Write(t, s, "1\n2\n3\n4\n5")
	zigTry(t, s.Select(Some(SelectionInit(
		zigUnwrap(t, s.pages.Pin(sc3Active(0, 0))),
		zigUnwrap(t, s.pages.Pin(sc3Active(0, 1))),
		false,
	))))

	sc3Resize(t, s, 5, 3)
	sel := zigUnwrap(t, s.selection)
	expect(t, sel.Tracked(), "sel.tracked()")
	expect(t, !s.pages.PointFromPin(PointTagActive, sel.Start()).Valid, "s.pages.pointFromPin(.active, sel.start()) == null")
	sc3ExpectPoint(t, sc3Screen(0, 0), zigUnwrap(t, s.pages.PointFromPin(PointTagScreen, sel.Start())))
	sc3ExpectPoint(t, sc3Screen(0, 1), zigUnwrap(t, s.pages.PointFromPin(PointTagScreen, sel.End())))
}

// Zig: test "term51: resize: prompt clearing uses plain blank cells because the cursor style is released first"
func TestSc3_term51_resize_prompt_clearing_uses_plain_blank_cells_because_the_cursor_style_is_released_first(t *testing.T) {
	testStdCapacity(t)
	for _, mode := range []OSCParsersSemanticPromptRedraw{OSCParsersSemanticPromptRedrawTrue, OSCParsersSemanticPromptRedrawLast} {
		func() {
			s := sc3Init(t, 10, 4, 5)
			defer s.Deinit()
			sc3Prompt(s, OSCParsersSemanticPromptPromptKindInitial)
			sc3Write(t, s, "$ ")
			sc3SetAttr(t, s, sc3AttrBg256(4))

			sc3ResizeRedraw(t, s, 20, 4, mode)

			// The prompt row was cleared without the background colour...
			for x := 0; x < 4; x++ {
				expectEqual(t, uint64(0), sc3QBits(*sc3QCell(t, s, SizeCellCountInt(x), 0)))
			}
			// ...and the cursor got its style back on the (new) page.
			expect(t, s.cursor.style.bg_color.Tag == StyleColorPalette, "s.cursor.style.bg_color == .palette")
			expect(t, s.cursor.style_id != StyleDefaultId, "s.cursor.style_id != style.default_id")
			expectEqual(t, 1, s.cursor.page_pin.node.data.styles.Count())
			sc3QExpectExactRefs(t, s)
		}()
	}
}

// sc3RPin is term51RPin: the pin at the given screen coordinates.
func sc3RPin(t *testing.T, s *Screen, x SizeCellCountInt, y uint32) PageListPin {
	t.Helper()
	return zigUnwrap(t, s.pages.Pin(sc3Screen(x, y)))
}

// sc3RPt is term51RPt: the screen point of a pin.
func sc3RPt(t *testing.T, s *Screen, p PageListPin) Point {
	t.Helper()
	return zigUnwrap(t, s.pages.PointFromPin(PointTagScreen, p))
}

// Zig: test "term51: select: null clears existing selection and untracks its pins"
func TestSc3_term51_select_null_clears_existing_selection_and_untracks_its_pins(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	s := sc3Init(t, 10, 5, 0)
	defer s.Deinit()
	sc3Write(t, s, "hello")

	base := s.pages.CountTrackedPins()
	zigTry(t, s.Select(Some(SelectionInit(sc3RPin(t, s, 0, 0), sc3RPin(t, s, 4, 0), false))))
	expect(t, s.selection.Valid, "s.selection != null")
	expectEqual(t, base+2, s.pages.CountTrackedPins())

	s.dirty.selection = false
	zigTry(t, s.Select(None[Selection]()))
	expect(t, !s.selection.Valid, "s.selection == null")
	expect(t, s.dirty.selection, "s.dirty.selection")
	expectEqual(t, base, s.pages.CountTrackedPins())
}

// Zig: test "term51: select: null with no selection does not set dirty"
func TestSc3_term51_select_null_with_no_selection_does_not_set_dirty(t *testing.T) {
	testStdCapacity(t)
	// Zig-only: const alloc = std.testing.allocator;
	s := sc3Init(t, 10, 5, 0)
	defer s.Deinit()

	base := s.pages.CountTrackedPins()
	expect(t, !s.dirty.selection, "!s.dirty.selection")
	zigTry(t, s.Select(None[Selection]()))
	expect(t, !s.selection.Valid, "s.selection == null")
	expect(t, !s.dirty.selection, "!s.dirty.selection")
	expectEqual(t, base, s.pages.CountTrackedPins())
}

// ---------------------------------------------------------------------------
// Go-only panic tests: the assert / unreachable / unwrap-of-null paths of the
// functions under test that Zig's tests avoid (a Zig test cannot test a path
// that aborts the process).
// ---------------------------------------------------------------------------

// Go-only: panic test: Screen.splitCellBoundary: x greater than cols (assert(x <= cols))
func TestSc3_panic_splitCellBoundary_x_greater_than_cols(t *testing.T) {
	testStdCapacity(t)
	s := sc3QTextScreen(t, "ABCDEF", 0)
	// No Deinit: the aborted call leaves the page's integrity checks paused.
	s.CursorAbsolute(0, 0)
	sc3ExpectPanic(t, func() { s.SplitCellBoundary(7) })
}

// Go-only: panic test: Screen.splitCellBoundary: a spacer head in the middle of the row (unreachable)
func TestSc3_panic_splitCellBoundary_spacer_head_in_the_middle_of_the_row(t *testing.T) {
	testStdCapacity(t)
	s := sc3QTextScreen(t, "ABCDEF", 0)
	// No Deinit: the aborted call leaves the page's integrity checks paused.
	s.CursorAbsolute(0, 0)
	// The left neighbour of x = 3 is a spacer head, which never occurs in
	// the middle of a row.
	*sc3QCell(t, s, 2, 0) = sc3CellWide(0, PageCellWideSpacerHead)
	sc3ExpectPanic(t, func() { s.SplitCellBoundary(3) })
}

// Go-only: panic test: Screen.clearCells: row is not a row of the page (assert on the row range)
func TestSc3_panic_clearCells_row_is_not_a_row_of_the_page(t *testing.T) {
	testStdCapacity(t)
	s := sc3OTwoPages(t, 20, 3)
	// No Deinit: the aborted call leaves the page's integrity checks paused.
	first := &s.pages.pages.first.data
	other := &s.pages.pages.last.data
	cells := first.GetCells(&first.rows[0])
	sc3ExpectPanic(t, func() { s.ClearCells(first, &other.rows[0], cells[0:1]) })
}

// Go-only: panic test: Screen.clearCells: cells are not cells of the row (assert on the cell range)
func TestSc3_panic_clearCells_cells_are_not_cells_of_the_row(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 2, 0)
	// No Deinit: the aborted call leaves the page's integrity checks paused.
	sc3Write(t, s, "abcde")
	page := &s.cursor.page_pin.node.data
	row0 := sc3QRow(t, s, 0)
	cells1 := page.GetCells(sc3QRow(t, s, 1))
	sc3ExpectPanic(t, func() { s.ClearCells(page, row0, cells1[0:2]) })
}

// Go-only: panic test: Screen.clearCells: empty cells slice (cells[0] out of bounds)
func TestSc3_panic_clearCells_empty_cells_slice(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 2, 0)
	// No Deinit: the aborted call leaves the page's integrity checks paused.
	sc3Write(t, s, "abcde")
	page := &s.cursor.page_pin.node.data
	row := sc3QRow(t, s, 0)
	cells := page.GetCells(row)
	sc3ExpectPanic(t, func() { s.ClearCells(page, row, cells[2:2]) })
}

// Go-only: panic test: Screen.cursorCopy: other.x not within the destination columns (assert(other.x < self.pages.cols))
func TestSc3_panic_cursorCopy_other_x_not_within_the_destination_columns(t *testing.T) {
	testStdCapacity(t)
	src := sc3Init(t, 30, 20, 0)
	defer src.Deinit()
	dst := sc3Init(t, 10, 5, 0)
	defer dst.Deinit()
	src.CursorAbsolute(15, 0)
	sc3ExpectPanic(t, func() { _ = dst.CursorCopy(src.cursor, screenCursorCopyOptionsDefault()) })
}

// Go-only: panic test: Screen.cursorCopy: other.y not within the destination rows (assert(other.y < self.pages.rows))
func TestSc3_panic_cursorCopy_other_y_not_within_the_destination_rows(t *testing.T) {
	testStdCapacity(t)
	src := sc3Init(t, 30, 20, 0)
	defer src.Deinit()
	dst := sc3Init(t, 10, 5, 0)
	defer dst.Deinit()
	src.CursorAbsolute(0, 10)
	sc3ExpectPanic(t, func() { _ = dst.CursorCopy(src.cursor, screenCursorCopyOptionsDefault()) })
}

// Go-only: panic test: Screen.cursorSetHyperlink: no hyperlink is active (assert(self.cursor.hyperlink_id != 0))
func TestSc3_panic_cursorSetHyperlink_no_hyperlink_is_active(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 10, 3, 0)
	defer s.Deinit()
	sc3ExpectPanic(t, func() { _ = s.CursorSetHyperlink() })
}

// Go-only: panic test: Screen.endHyperlink: hyperlink_id is 0 but a hyperlink is set (assert(self.cursor.hyperlink == null))
func TestSc3_panic_endHyperlink_id_zero_with_a_hyperlink_set(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 10, 3, 0)
	defer s.Deinit()
	s.cursor.hyperlink = &Hyperlink{}
	defer func() { s.cursor.hyperlink = nil }()
	sc3ExpectPanic(t, func() { s.EndHyperlink() })
}

// Go-only: panic test: Screen.cursorScrollAboveRotate: the cursor is on the last row of the last page (cursor.page_pin.down(1).? is null)
func TestSc3_panic_cursorScrollAboveRotate_cursor_on_the_last_row_of_the_last_page(t *testing.T) {
	testStdCapacity(t)
	s := sc3Init(t, 5, 3, 100)
	// No Deinit: the aborted call may leave the screen in a partial state.
	s.CursorAbsolute(0, 2)
	sc3ExpectPanic(t, func() { _ = s.cursorScrollAboveRotate() })
}
