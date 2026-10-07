// Tests translated one for one from src/terminal/Screen.zig (Ghostty v1.3.1
// plus the term51 additional tests), from the banner `term51: additional
// tests (AA)` to the end of the file, and from src/terminal/ScreenSet.zig,
// the tests under its banner `term51: additional tests (AA)`. Each Go test
// carries the exact Zig title in a "// Zig: test" comment. Helpers are
// prefixed sca. The Screen tests are TestSca_..., the ScreenSet tests are
// TestScaSet_....
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualStrings keep Zig's names and Zig's (expected,
//     actual) argument order. `try` on a call that returns an error is
//     zigTry(t, err); `x.?` is zigUnwrap or a nil check.
//   - Zig's helpers term51AA* are the unexported sca* functions below.
//     `init(alloc, ...)` is scaInit (ScreenInit with the Zig default of
//     kitty_image_storage_limit written out, plus zigTry); the screen is a
//     *Screen so that Zig's `&s` is `s`.
//   - Every test starts with testStdCapacity(t): Zig's unit tests run with
//     grapheme_bytes 512, and the multi-page tests depend on the rows per
//     page.
//   - `defer s.deinit()` inside a loop body runs at the end of each
//     iteration in Zig; those loop bodies are function literals here so the
//     Go defer behaves the same way.
//   - `std.meta.tags(term51AAMode)` is the slice scaModes, and
//     `std.meta.eql(before, s.cursor)` is `before == s.cursor` (ScreenCursor
//     is comparable, its pointers compare by address as in Zig).
//   - `@as(u64, @bitCast(cell.*))` is PageCell.bitCast, a row's is
//     PageRow.bitCast and `@as(Row, @bitCast(bits))` is pageRowFromBits.
//   - Raw-pointer arithmetic (the "layout pointer distance" tests) becomes a
//     slice-index comparison: Page.cellIndex of the returned cell.
//   - Zig-only: `const alloc = std.testing.allocator;` and the allocator
//     arguments are allocator bookkeeping with no Go counterpart; they are
//     dropped without a per-test note. `errdefer s.deinit()` in
//     term51AAMultiPage is dropped (a failed check stops the test).
//   - Page memory accessors take no `page.memory`: `page.styles.get(
//     page.memory, id)` is page.styles.Get(id), `refCount(page.memory, id)`
//     is RefCount(id), and `entry.uri.slice(page.memory)` is entry.uri.
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt. The asserts of the functions under
//     test (cursorUp, cursorLeft, cursorCellRight, cursorCellLeft,
//     cursorHorizontalAbsolute) already have panic tests in Screen_2_test.go
//     (TestSc2_panic_...); they are not repeated here.

package terminal

import "testing"

// scaMode is term51AAMode: what the cursor carries while the cursor tests
// below move it around.
type scaMode uint8

const (
	scaModeNone scaMode = iota
	scaModeStyle
	scaModeLinkExplicit
	scaModeLinkImplicit
	scaModeBoth
)

// scaModes is std.meta.tags(term51AAMode).
var scaModes = []scaMode{
	scaModeNone,
	scaModeStyle,
	scaModeLinkExplicit,
	scaModeLinkImplicit,
	scaModeBoth,
}

// scaBig is term51AABig: a scrollback limit that is large enough never to
// prune anything.
const scaBig uint64 = 100 * 1024 * 1024

// scaInit is `try init(alloc, .{ .cols = cols, .rows = rows,
// .max_scrollback = max_scrollback })`: kitty_image_storage_limit is the Zig
// default, 320 * 1000 * 1000.
func scaInit(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt, max_scrollback uint64) *Screen {
	t.Helper()
	s, err := ScreenInit(ScreenOptions{
		Cols:                   cols,
		Rows:                   rows,
		MaxScrollback:          max_scrollback,
		KittyImageStorageLimit: 320 * 1000 * 1000,
	})
	zigTry(t, err)
	return &s
}

// scaWrite is `try s.testWriteString(text)`.
func scaWrite(t *testing.T, s *Screen, text string) {
	t.Helper()
	zigTry(t, s.TestWriteString([]uint8(text)))
}

// scaPin is term51AAPin: an (untracked) pin at the active point (x, y).
func scaPin(t *testing.T, s *Screen, x uint, y uint) PageListPin {
	t.Helper()
	return zigUnwrap(t, s.Pages.Pin(Point{
		Tag:    PointTagActive,
		Active: PointCoordinate{X: SizeCellCountInt(x), Y: uint32(y)},
	}))
}

// scaCell is term51AACell.
func scaCell(t *testing.T, s *Screen, x uint, y uint) *PageCell {
	t.Helper()
	return scaPin(t, s, x, y).RowAndCell().cell
}

// scaRow is term51AARow.
func scaRow(t *testing.T, s *Screen, y uint) *PageRow {
	t.Helper()
	return scaPin(t, s, 0, y).RowAndCell().row
}

// scaSetCp is term51AASetCp.
func scaSetCp(t *testing.T, s *Screen, x uint, y uint, cp uint32) {
	t.Helper()
	*scaCell(t, s, x, y) = PageCell{
		content_tag: PageCellContentTagCodepoint,
		content:     PageCellContent{codepoint: cp},
	}
}

// scaCp is term51AACp.
func scaCp(t *testing.T, s *Screen, x uint, y uint) uint32 {
	t.Helper()
	return scaCell(t, s, x, y).Codepoint()
}

// scaStyleRefs is term51AAStyleRefs.
func scaStyleRefs(page *Page, id StyleID) uint {
	return uint(page.styles.RefCount(id))
}

// scaLinkRefs is term51AALinkRefs.
func scaLinkRefs(page *Page, id HyperlinkID) uint {
	return uint(page.hyperlink_set.RefCount(id))
}

// scaNodeCount is term51AANodeCount.
func scaNodeCount(s *Screen) uint {
	var n uint
	for node := s.Pages.pages.first; node != nil; node = node.next {
		n++
	}
	return n
}

// scaMultiPage is term51AAMultiPage: a 2000 column, 60 row screen whose
// active area spans at least three pages.
func scaMultiPage(t *testing.T, max_scrollback uint64) *Screen {
	t.Helper()
	s := scaInit(t, 2000, 60, max_scrollback)
	// Zig-only: errdefer s.deinit()
	expect(t, scaNodeCount(s) >= 3, "term51AANodeCount(&s) >= 3")
	return s
}

// scaBoundaries is term51AABoundaries: marks which active rows touch a page
// boundary (the last row of a page and the first row of the next one). At
// least two boundaries must exist.
func scaBoundaries(t *testing.T, s *Screen) [60]bool {
	t.Helper()
	var adj [60]bool
	var count uint
	for y := uint(1); y < uint(s.Pages.rows); y++ {
		if scaPin(t, s, 0, y).Node != scaPin(t, s, 0, y-1).Node {
			adj[y] = true
			adj[y-1] = true
			count++
		}
	}
	expect(t, count >= 2, "count >= 2")
	return adj
}

// scaApplyMode is term51AAApplyMode: put the style and/or hyperlink of
// `mode` on the cursor.
func scaApplyMode(t *testing.T, s *Screen, mode scaMode) {
	t.Helper()
	switch mode {
	case scaModeNone:
	case scaModeStyle:
		zigTry(t, s.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
	case scaModeLinkExplicit:
		zigTry(t, s.StartHyperlink([]uint8("https://example.com/"), Some([]uint8("ID"))))
	case scaModeLinkImplicit:
		zigTry(t, s.StartHyperlink([]uint8("https://example.com/"), None[[]uint8]()))
	case scaModeBoth:
		zigTry(t, s.SetAttribute(SGRAttribute{Tag: SGRAttributeTagBold}))
		zigTry(t, s.StartHyperlink([]uint8("https://example.com/"), Some([]uint8("ID"))))
	}
}

// scaExpectMode is term51AAExpectMode: expect that the cursor carries
// exactly what `mode` says, and that the cursor holds exactly one reference
// of each in its current page.
func scaExpectMode(t *testing.T, s *Screen, mode scaMode) {
	t.Helper()
	page := &s.Cursor.PagePin.Node.data
	has_style := mode == scaModeStyle || mode == scaModeBoth
	if has_style {
		expect(t, s.Cursor.Style.Flags.Bold, "s.cursor.style.flags.bold")
		expect(t, s.Cursor.StyleID != StyleDefaultID, "s.cursor.style_id != style.default_id")
		expect(t, page.styles.Get(s.Cursor.StyleID).Flags.Bold, "page.styles.get(page.memory, s.cursor.style_id).flags.bold")
		expectEqual[uint](t, 1, scaStyleRefs(page, s.Cursor.StyleID))
	} else {
		expectEqual[StyleID](t, StyleDefaultID, s.Cursor.StyleID)
		expect(t, !s.Cursor.Style.Flags.Bold, "!s.cursor.style.flags.bold")
	}

	has_link := mode == scaModeLinkExplicit || mode == scaModeLinkImplicit || mode == scaModeBoth
	if has_link {
		expect(t, s.Cursor.Hyperlink != nil, "s.cursor.hyperlink != null")
		expect(t, s.Cursor.HyperlinkID != 0, "s.cursor.hyperlink_id != 0")
		expectEqual[uint](t, 1, scaLinkRefs(page, s.Cursor.HyperlinkID))
		link := s.Cursor.Hyperlink
		expectEqualStrings(t, "https://example.com/", string(link.URI))
		switch link.ID.Tag {
		case HyperlinkHyperlinkIdExplicit:
			expect(t, mode != scaModeLinkImplicit, "mode != .link_implicit")
			expectEqualStrings(t, "ID", string(link.ID.Explicit))
		case HyperlinkHyperlinkIdImplicit:
			expect(t, mode == scaModeLinkImplicit, "mode == .link_implicit")
		}
		entry := page.hyperlink_set.Get(s.Cursor.HyperlinkID)
		expectEqualStrings(t, "https://example.com/", string(entry.uri))
	} else {
		expect(t, s.Cursor.Hyperlink == nil, "s.cursor.hyperlink == null")
		expectEqual[HyperlinkID](t, 0, s.Cursor.HyperlinkID)
	}
}

// scaExpectReleased is term51AAExpectReleased: expect that, if the cursor
// left `old_node`, the old page no longer holds the cursor's references.
func scaExpectReleased(
	t *testing.T,
	s *Screen,
	old_node *PageListNode,
	old_style_id StyleID,
	old_link_id HyperlinkID,
) {
	t.Helper()
	if s.Cursor.PagePin.Node == old_node {
		return
	}
	if old_style_id != StyleDefaultID {
		expectEqual[uint](t, 0, scaStyleRefs(&old_node.data, old_style_id))
	}
	if old_link_id != 0 {
		expectEqual[uint](t, 0, scaLinkRefs(&old_node.data, old_link_id))
	}
}

// scaExpectConsistent is term51AAExpectConsistent: expect that the cursor
// cache (x, y, row, cell) agrees with the pin.
func scaExpectConsistent(t *testing.T, s *Screen) {
	t.Helper()
	rac := s.Cursor.PagePin.RowAndCell()
	expect(t, s.Cursor.PageRow == rac.row, "s.cursor.page_row == rac.row")
	expect(t, s.Cursor.PageCell == rac.cell, "s.cursor.page_cell == rac.cell")
	pt := zigUnwrap(t, s.Pages.PointFromPin(PointTagActive, *s.Cursor.PagePin))
	expectEqual[uint](t, uint(s.Cursor.X), uint(pt.Active.X))
	expectEqual[uint](t, uint(s.Cursor.Y), uint(pt.Active.Y))
	expectEqual[SizeCellCountInt](t, s.Cursor.X, s.Cursor.PagePin.X)
}

// scaExpectDirty is term51AAExpectDirty: expect that exactly the active rows
// `a` and `b` (if given) have their row dirty flag set, and that no page is
// flagged dirty.
func scaExpectDirty(t *testing.T, s *Screen, a Optional[uint], b Optional[uint]) {
	t.Helper()
	for y := uint(0); y < uint(s.Pages.rows); y++ {
		want := (a.Valid && a.Value == y) || (b.Valid && b.Value == y)
		expectEqual[bool](t, want, scaRow(t, s, y).dirty)
	}
	for node := s.Pages.pages.first; node != nil; node = node.next {
		expect(t, !node.data.dirty, "!node.data.dirty")
	}
}

// scaUpCase is term51AAUpCase.
func scaUpCase(
	t *testing.T,
	max_scrollback uint64,
	mode scaMode,
	pending bool,
	all_n bool,
) {
	t.Helper()
	s := scaMultiPage(t, max_scrollback)
	defer s.Deinit()
	rows := uint(s.Pages.rows)
	adj := scaBoundaries(t, s)
	scaApplyMode(t, s, mode)

	for y := uint(0); y < rows; y++ {
		for n := uint(0); n < y+1; n++ {
			sparse := n <= 2 || n+1 >= y || n == y/2
			if !(sparse || (all_n && adj[y])) {
				continue
			}

			x := SizeCellCountInt((y*337 + n*71) % 2000)
			s.CursorAbsolute(x, SizeCellCountInt(y))
			s.Cursor.PendingWrap = pending
			s.Pages.ClearDirty()
			old_node := s.Cursor.PagePin.Node
			old_style_id := s.Cursor.StyleID
			old_link_id := s.Cursor.HyperlinkID

			s.CursorUp(SizeCellCountInt(n))

			expectEqual[SizeCellCountInt](t, x, s.Cursor.X)
			expectEqual[SizeCellCountInt](t, SizeCellCountInt(y-n), s.Cursor.Y)
			expectEqual[bool](t, pending, s.Cursor.PendingWrap)
			scaExpectConsistent(t, s)
			expect(t, s.Cursor.PagePin.Node == scaPin(t, s, uint(x), y-n).Node,
				"s.cursor.page_pin.node == term51AAPin(&s, x, y - n).node")
			scaExpectReleased(t, s, old_node, old_style_id, old_link_id)
			scaExpectMode(t, s, mode)
			if n == 0 {
				scaExpectDirty(t, s, None[uint](), None[uint]())
			} else {
				scaExpectDirty(t, s, Some(y), Some(y-n))
			}
		}
	}
	s.AssertIntegrity()
}

// Zig: test "term51: cursorUp: every row and distance of a multi-page screen, nothing on the cursor"
func TestSca_cursorUp_every_row_and_distance_of_a_multi_page_screen_nothing_on_the_cursor(t *testing.T) {
	testStdCapacity(t)
	scaUpCase(t, 0, scaModeNone, false, true)
}

// Zig: test "term51: cursorUp: every row and distance of a multi-page screen, with a style"
func TestSca_cursorUp_every_row_and_distance_of_a_multi_page_screen_with_a_style(t *testing.T) {
	testStdCapacity(t)
	scaUpCase(t, 0, scaModeStyle, false, true)
}

// Zig: test "term51: cursorUp: every row and distance of a multi-page screen, with an explicit hyperlink"
func TestSca_cursorUp_every_row_and_distance_of_a_multi_page_screen_with_an_explicit_hyperlink(t *testing.T) {
	testStdCapacity(t)
	scaUpCase(t, 0, scaModeLinkExplicit, false, true)
}

// Zig: test "term51: cursorUp: every row and distance of a multi-page screen, with an implicit hyperlink"
func TestSca_cursorUp_every_row_and_distance_of_a_multi_page_screen_with_an_implicit_hyperlink(t *testing.T) {
	testStdCapacity(t)
	scaUpCase(t, 0, scaModeLinkImplicit, false, true)
}

// Zig: test "term51: cursorUp: every row and distance of a multi-page screen, style and hyperlink without scrollback"
func TestSca_cursorUp_every_row_and_distance_of_a_multi_page_screen_style_and_hyperlink_without_scrollback(t *testing.T) {
	testStdCapacity(t)
	scaUpCase(t, 0, scaModeBoth, false, true)
}

// Zig: test "term51: cursorUp: every row and distance of a multi-page screen, style and hyperlink with scrollback"
func TestSca_cursorUp_every_row_and_distance_of_a_multi_page_screen_style_and_hyperlink_with_scrollback(t *testing.T) {
	testStdCapacity(t)
	scaUpCase(t, scaBig, scaModeBoth, false, true)
}

// Zig: test "term51: cursorUp: pending wrap is kept on a multi-page screen with style and hyperlink"
func TestSca_cursorUp_pending_wrap_is_kept_on_a_multi_page_screen_with_style_and_hyperlink(t *testing.T) {
	testStdCapacity(t)
	scaUpCase(t, 0, scaModeBoth, true, false)
	scaUpCase(t, scaBig, scaModeBoth, true, false)
	scaUpCase(t, 0, scaModeNone, true, false)
}

// Zig: test "term51: cursorUp: count zero, one and the largest count on a small screen"
func TestSca_cursorUp_count_zero_one_and_the_largest_count_on_a_small_screen(t *testing.T) {
	testStdCapacity(t)
	for _, sb := range []uint64{0, scaBig} {
		for _, pending := range []bool{false, true} {
			func() {
				s := scaInit(t, 4, 5, sb)
				defer s.Deinit()
				scaApplyMode(t, s, scaModeBoth)
				node := s.Cursor.PagePin.Node
				style_id := s.Cursor.StyleID
				link_id := s.Cursor.HyperlinkID

				// From the last row: the largest count is the cursor row (4).
				s.CursorAbsolute(3, 4)
				s.Cursor.PendingWrap = pending
				s.Pages.ClearDirty()
				s.CursorUp(0)
				expectEqual[SizeCellCountInt](t, 4, s.Cursor.Y)
				scaExpectDirty(t, s, None[uint](), None[uint]())
				s.CursorUp(1)
				expectEqual[SizeCellCountInt](t, 3, s.Cursor.Y)
				scaExpectDirty(t, s, Some[uint](3), Some[uint](4))
				s.CursorUp(3)
				expectEqual[SizeCellCountInt](t, 0, s.Cursor.Y)
				expectEqual[SizeCellCountInt](t, 3, s.Cursor.X)
				expectEqual[bool](t, pending, s.Cursor.PendingWrap)
				scaExpectConsistent(t, s)
				expect(t, s.Cursor.PagePin.Node == node, "s.cursor.page_pin.node == node")
				expectEqual[StyleID](t, style_id, s.Cursor.StyleID)
				expectEqual[HyperlinkID](t, link_id, s.Cursor.HyperlinkID)
				scaExpectMode(t, s, scaModeBoth)

				// At the top row only zero is possible.
				s.CursorUp(0)
				expectEqual[SizeCellCountInt](t, 0, s.Cursor.Y)
				scaExpectConsistent(t, s)
			}()
		}
	}
}

// Zig: test "term51: cursorUp: one row screen only accepts zero"
func TestSca_cursorUp_one_row_screen_only_accepts_zero(t *testing.T) {
	testStdCapacity(t)
	s := scaInit(t, 1, 1, 0)
	defer s.Deinit()
	scaApplyMode(t, s, scaModeBoth)
	s.Cursor.PendingWrap = true
	s.CursorUp(0)
	expectEqual[SizeCellCountInt](t, 0, s.Cursor.Y)
	expect(t, s.Cursor.PendingWrap, "s.cursor.pending_wrap")
	scaExpectConsistent(t, s)
	scaExpectMode(t, s, scaModeBoth)
}

// scaDownCase is term51AADownCase: moving down by one with
// `cursorDownOrScroll` from every row but the last one of a multi-page
// screen.
func scaDownCase(t *testing.T, max_scrollback uint64, mode scaMode, pending bool) {
	t.Helper()
	s := scaMultiPage(t, max_scrollback)
	defer s.Deinit()
	rows := uint(s.Pages.rows)
	scaApplyMode(t, s, mode)
	total := s.Pages.total_rows

	for y := uint(0); y < rows-1; y++ {
		x := SizeCellCountInt((y*337 + 5) % 2000)
		s.CursorAbsolute(x, SizeCellCountInt(y))
		s.Cursor.PendingWrap = pending
		s.Pages.ClearDirty()
		old_node := s.Cursor.PagePin.Node
		old_style_id := s.Cursor.StyleID
		old_link_id := s.Cursor.HyperlinkID

		zigTry(t, s.cursorDownOrScroll())

		expectEqual[SizeCellCountInt](t, x, s.Cursor.X)
		expectEqual[SizeCellCountInt](t, SizeCellCountInt(y+1), s.Cursor.Y)
		expectEqual[bool](t, pending, s.Cursor.PendingWrap)
		expectEqual[uint](t, total, s.Pages.total_rows)
		scaExpectConsistent(t, s)
		expect(t, s.Cursor.PagePin.Node == scaPin(t, s, uint(x), y+1).Node,
			"s.cursor.page_pin.node == term51AAPin(&s, x, y + 1).node")
		scaExpectReleased(t, s, old_node, old_style_id, old_link_id)
		scaExpectMode(t, s, mode)
		scaExpectDirty(t, s, Some(y), Some(y+1))
	}
	s.AssertIntegrity()
}

// Zig: test "term51: cursorDownOrScroll: moving down at every row of a multi-page screen, nothing on the cursor"
func TestSca_cursorDownOrScroll_moving_down_at_every_row_of_a_multi_page_screen_nothing_on_the_cursor(t *testing.T) {
	testStdCapacity(t)
	scaDownCase(t, 0, scaModeNone, false)
}

// Zig: test "term51: cursorDownOrScroll: moving down at every row of a multi-page screen, with a style"
func TestSca_cursorDownOrScroll_moving_down_at_every_row_of_a_multi_page_screen_with_a_style(t *testing.T) {
	testStdCapacity(t)
	scaDownCase(t, 0, scaModeStyle, false)
}

// Zig: test "term51: cursorDownOrScroll: moving down at every row of a multi-page screen, with hyperlinks"
func TestSca_cursorDownOrScroll_moving_down_at_every_row_of_a_multi_page_screen_with_hyperlinks(t *testing.T) {
	testStdCapacity(t)
	scaDownCase(t, 0, scaModeLinkExplicit, false)
	scaDownCase(t, 0, scaModeLinkImplicit, false)
}

// Zig: test "term51: cursorDownOrScroll: moving down at every row of a multi-page screen, style and hyperlink"
func TestSca_cursorDownOrScroll_moving_down_at_every_row_of_a_multi_page_screen_style_and_hyperlink(t *testing.T) {
	testStdCapacity(t)
	scaDownCase(t, 0, scaModeBoth, false)
	scaDownCase(t, scaBig, scaModeBoth, false)
}

// Zig: test "term51: cursorDownOrScroll: moving down keeps pending wrap on a multi-page screen"
func TestSca_cursorDownOrScroll_moving_down_keeps_pending_wrap_on_a_multi_page_screen(t *testing.T) {
	testStdCapacity(t)
	scaDownCase(t, 0, scaModeBoth, true)
	scaDownCase(t, scaBig, scaModeBoth, true)
	scaDownCase(t, 0, scaModeNone, true)
}

// scaMarkRows is term51AAMarkRows: marks every active row of a screen:
// column 0 of row y gets 0x100 + y.
func scaMarkRows(t *testing.T, s *Screen) {
	t.Helper()
	for y := uint(0); y < uint(s.Pages.rows); y++ {
		scaSetCp(t, s, 0, y, uint32(0x100+y))
	}
}

// scaExpectBlankRow is term51AAExpectBlankRow: expect that every cell of the
// active row y is blank.
func scaExpectBlankRow(t *testing.T, s *Screen, y uint) {
	t.Helper()
	for x := uint(0); x < uint(s.Pages.cols); x++ {
		cell := scaCell(t, s, x, y)
		expectEqual[uint64](t, 0, cell.bitCast())
	}
}

// scaScrollCase is term51AAScrollCase: scroll at the last row of a
// multi-page screen; `cursorDownOrScroll` at the bottom shifts the screen up
// by one row.
func scaScrollCase(t *testing.T, max_scrollback uint64, mode scaMode, pending bool) {
	t.Helper()
	s := scaMultiPage(t, max_scrollback)
	defer s.Deinit()
	rows := uint(s.Pages.rows)
	scaMarkRows(t, s)
	scaApplyMode(t, s, mode)
	x := SizeCellCountInt(1234)
	s.CursorAbsolute(x, SizeCellCountInt(rows-1))
	s.Cursor.PendingWrap = pending
	total := s.Pages.total_rows
	first_active := scaPin(t, s, 0, 0).Node
	s.Pages.ClearDirty()
	old_node := s.Cursor.PagePin.Node
	old_style_id := s.Cursor.StyleID
	old_link_id := s.Cursor.HyperlinkID

	zigTry(t, s.cursorDownOrScroll())

	expectEqual[SizeCellCountInt](t, x, s.Cursor.X)
	expectEqual[SizeCellCountInt](t, SizeCellCountInt(rows-1), s.Cursor.Y)
	expectEqual[bool](t, pending, s.Cursor.PendingWrap)
	scaExpectConsistent(t, s)
	scaExpectReleased(t, s, old_node, old_style_id, old_link_id)
	scaExpectMode(t, s, mode)
	for y := uint(0); y < rows-1; y++ {
		expectEqual[uint32](t, uint32(0x100+y+1), scaCp(t, s, 0, y))
	}
	scaExpectBlankRow(t, s, rows-1)

	if max_scrollback == 0 {
		// The rows were shifted inside the existing pages: nothing grew, the
		// cursor stays on its page and every page from the first active row
		// down is flagged dirty.
		expectEqual[uint](t, total, s.Pages.total_rows)
		expect(t, s.Cursor.PagePin.Node == old_node, "s.cursor.page_pin.node == old_node")
		for node := first_active; node != nil; node = node.next {
			expect(t, node.data.dirty, "node.data.dirty")
		}
	} else {
		// The old first row is now history.
		expectEqual[uint](t, total+1, s.Pages.total_rows)
		hist := zigUnwrap(t, s.Pages.Pin(Point{Tag: PointTagScreen, Screen: PointCoordinate{X: 0, Y: 0}}))
		expectEqual[uint32](t, 0x100, hist.RowAndCell().cell.Codepoint())
		// The old cursor row (now one up) and the new one are dirty.
		expect(t, scaRow(t, s, rows-2).dirty, "term51AARow(&s, rows - 2).dirty")
		expect(t, scaRow(t, s, rows-1).dirty, "term51AARow(&s, rows - 1).dirty")
	}
	s.AssertIntegrity()
}

// Zig: test "term51: cursorDownOrScroll: scrolling at the last row of a multi-page screen without scrollback"
func TestSca_cursorDownOrScroll_scrolling_at_the_last_row_of_a_multi_page_screen_without_scrollback(t *testing.T) {
	testStdCapacity(t)
	scaScrollCase(t, 0, scaModeNone, false)
	scaScrollCase(t, 0, scaModeStyle, false)
	scaScrollCase(t, 0, scaModeLinkExplicit, false)
	scaScrollCase(t, 0, scaModeLinkImplicit, false)
	scaScrollCase(t, 0, scaModeBoth, false)
}

// Zig: test "term51: cursorDownOrScroll: scrolling at the last row of a multi-page screen with scrollback"
func TestSca_cursorDownOrScroll_scrolling_at_the_last_row_of_a_multi_page_screen_with_scrollback(t *testing.T) {
	testStdCapacity(t)
	scaScrollCase(t, scaBig, scaModeNone, false)
	scaScrollCase(t, scaBig, scaModeStyle, false)
	scaScrollCase(t, scaBig, scaModeLinkExplicit, false)
	scaScrollCase(t, scaBig, scaModeLinkImplicit, false)
	scaScrollCase(t, scaBig, scaModeBoth, false)
}

// Zig: test "term51: cursorDownOrScroll: scrolling keeps pending wrap on a multi-page screen"
func TestSca_cursorDownOrScroll_scrolling_keeps_pending_wrap_on_a_multi_page_screen(t *testing.T) {
	testStdCapacity(t)
	scaScrollCase(t, 0, scaModeBoth, true)
	scaScrollCase(t, scaBig, scaModeBoth, true)
	scaScrollCase(t, 0, scaModeNone, true)
	scaScrollCase(t, scaBig, scaModeNone, true)
}

// Zig: test "term51: cursorDownOrScroll: scrolling until a new page is created moves style and hyperlink"
func TestSca_cursorDownOrScroll_scrolling_until_a_new_page_is_created_moves_style_and_hyperlink(t *testing.T) {
	testStdCapacity(t)
	for _, mode := range []scaMode{scaModeNone, scaModeStyle, scaModeLinkImplicit, scaModeBoth} {
		func() {
			s := scaMultiPage(t, scaBig)
			defer s.Deinit()
			rows := uint(s.Pages.rows)
			scaApplyMode(t, s, mode)
			s.CursorAbsolute(77, SizeCellCountInt(rows-1))
			nodes := scaNodeCount(s)
			limit := uint(s.Pages.pages.last.data.capacity.rows) + 2

			changed := false
			for i := uint(0); i < limit; i++ {
				old_node := s.Cursor.PagePin.Node
				old_style_id := s.Cursor.StyleID
				old_link_id := s.Cursor.HyperlinkID
				total := s.Pages.total_rows
				zigTry(t, s.cursorDownOrScroll())
				expectEqual[uint](t, total+1, s.Pages.total_rows)
				expectEqual[SizeCellCountInt](t, 77, s.Cursor.X)
				expectEqual[SizeCellCountInt](t, SizeCellCountInt(rows-1), s.Cursor.Y)
				scaExpectConsistent(t, s)
				scaExpectReleased(t, s, old_node, old_style_id, old_link_id)
				scaExpectMode(t, s, mode)
				expect(t, s.Cursor.PagePin.Node == s.Pages.pages.last,
					"s.cursor.page_pin.node == s.pages.pages.last.?")
				if s.Cursor.PagePin.Node != old_node {
					changed = true
					break
				}
			}
			expect(t, changed, "changed")
			expectEqual[uint](t, nodes+1, scaNodeCount(s))
			scaExpectBlankRow(t, s, rows-1)
			s.AssertIntegrity()
		}()
	}
}

// Zig: test "term51: cursorDownOrScroll: screens of one, two and three rows"
func TestSca_cursorDownOrScroll_screens_of_one_two_and_three_rows(t *testing.T) {
	testStdCapacity(t)
	for _, rows := range []SizeCellCountInt{1, 2, 3} {
		for _, sb := range []uint64{0, scaBig} {
			for _, pending := range []bool{false, true} {
				func() {
					s := scaInit(t, 5, rows, sb)
					defer s.Deinit()
					scaApplyMode(t, s, scaModeBoth)
					scaMarkRows(t, s)
					s.Cursor.PendingWrap = pending
					model := [3]uint32{0x100, 0x101, 0x102}
					var y SizeCellCountInt = 0

					for i := SizeCellCountInt(0); i < rows+3; i++ {
						total := s.Pages.total_rows
						zigTry(t, s.cursorDownOrScroll())
						if y+1 < rows {
							y += 1
							expectEqual[uint](t, total, s.Pages.total_rows)
						} else {
							for j := SizeCellCountInt(0); j < rows-1; j++ {
								model[j] = model[j+1]
							}
							model[rows-1] = 0
							var grown uint
							if sb == 0 {
								grown = 0
							} else {
								grown = 1
							}
							expectEqual[uint](t, total+grown, s.Pages.total_rows)
						}
						expectEqual[SizeCellCountInt](t, y, s.Cursor.Y)
						expectEqual[bool](t, pending, s.Cursor.PendingWrap)
						for yy := uint(0); yy < uint(rows); yy++ {
							expectEqual[uint32](t, model[yy], scaCp(t, s, 0, yy))
						}
						scaExpectConsistent(t, s)
						scaExpectMode(t, s, scaModeBoth)
					}
					expectEqual[SizeCellCountInt](t, rows-1, y)
				}()
			}
		}
	}
}

// Zig: test "term51: cursorDownOrScroll: references of cells in the scrolled out row"
func TestSca_cursorDownOrScroll_references_of_cells_in_the_scrolled_out_row(t *testing.T) {
	testStdCapacity(t)
	for _, sb := range []uint64{0, scaBig} {
		func() {
			s := scaInit(t, 5, 2, sb)
			defer s.Deinit()
			scaApplyMode(t, s, scaModeBoth)
			scaWrite(t, s, "ab")
			page := &s.Cursor.PagePin.Node.data
			style_id := s.Cursor.StyleID
			link_id := s.Cursor.HyperlinkID
			expectEqual[uint](t, 3, scaStyleRefs(page, style_id))
			expectEqual[uint](t, 3, scaLinkRefs(page, link_id))

			// Moving down from the first row changes nothing.
			zigTry(t, s.cursorDownOrScroll())
			expectEqual[SizeCellCountInt](t, 1, s.Cursor.Y)
			expectEqual[uint](t, 3, scaStyleRefs(page, style_id))

			// Scrolling: the row with "ab" is dropped without scrollback and
			// kept as history with it.
			zigTry(t, s.cursorDownOrScroll())
			expectEqual[StyleID](t, style_id, s.Cursor.StyleID)
			expectEqual[HyperlinkID](t, link_id, s.Cursor.HyperlinkID)
			var want uint
			if sb == 0 {
				want = 1
			} else {
				want = 3
			}
			expectEqual[uint](t, want, scaStyleRefs(page, style_id))
			expectEqual[uint](t, want, scaLinkRefs(page, link_id))
			expectEqual[uint32](t, 0, scaCp(t, s, 0, 0))
			expectEqual[uint32](t, 0, scaCp(t, s, 0, 1))
			s.AssertIntegrity()
		}()
	}
}

// Zig: test "term51: cursorDownOrScroll: a background color fills only the row that scrolls in"
func TestSca_cursorDownOrScroll_a_background_color_fills_only_the_row_that_scrolls_in(t *testing.T) {
	testStdCapacity(t)
	for _, palette := range []bool{false, true} {
		for _, rows := range []SizeCellCountInt{1, 2, 3} {
			for _, sb := range []uint64{0, scaBig} {
				func() {
					s := scaInit(t, 5, rows, sb)
					defer s.Deinit()
					if palette {
						zigTry(t, s.SetAttribute(SGRAttribute{Tag: SGRAttributeTag256Bg, X256Bg: 9}))
					} else {
						zigTry(t, s.SetAttribute(SGRAttribute{
							Tag:           SGRAttributeTagDirectColorBg,
							DirectColorBg: ColorRGB{R: 1, G: 2, B: 3},
						}))
					}
					scaMarkRows(t, s)

					// Moving down fills nothing.
					for i := SizeCellCountInt(0); i < rows-1; i++ {
						zigTry(t, s.cursorDownOrScroll())
						for x := uint(1); x < 5; x++ {
							expectEqual[PageCellContentTag](t,
								PageCellContentTagCodepoint,
								scaCell(t, s, x, uint(s.Cursor.Y)).content_tag,
							)
						}
					}

					s.CursorAbsolute(3, rows-1)
					zigTry(t, s.cursorDownOrScroll())
					expectEqual[SizeCellCountInt](t, 3, s.Cursor.X)
					for x := uint(0); x < 5; x++ {
						cell := scaCell(t, s, x, uint(rows-1))
						if palette {
							expectEqual[PageCellContentTag](t, PageCellContentTagBgColorPalette, cell.content_tag)
							expectEqual[uint8](t, 9, cell.content.color_palette)
						} else {
							expectEqual[PageCellContentTag](t, PageCellContentTagBgColorRgb, cell.content_tag)
							expectEqual[uint8](t, 1, cell.content.color_rgb.R)
							expectEqual[uint8](t, 2, cell.content.color_rgb.G)
							expectEqual[uint8](t, 3, cell.content.color_rgb.B)
						}
					}
					// The rows above hold shifted marks, not the fill.
					for y := uint(0); y < uint(rows)-1; y++ {
						expectEqual[uint32](t, uint32(0x100+y+1), scaCp(t, s, 0, y))
					}
					scaExpectConsistent(t, s)
				}()
			}
		}
	}
}

// scaHorizontalCase is term51AAHorizontalCase.
func scaHorizontalCase(t *testing.T, max_scrollback uint64, mode scaMode, pending bool) {
	t.Helper()
	s := scaMultiPage(t, max_scrollback)
	defer s.Deinit()
	rows := uint(s.Pages.rows)
	scaApplyMode(t, s, mode)

	for y := uint(0); y < rows; y++ {
		s.CursorAbsolute(1000, SizeCellCountInt(y))
		s.Cursor.PendingWrap = pending
		s.Pages.ClearDirty()
		node := s.Cursor.PagePin.Node
		row := s.Cursor.PageRow
		style_id := s.Cursor.StyleID
		link_id := s.Cursor.HyperlinkID

		for _, x := range []SizeCellCountInt{0, 1, 1, 1000, 1998, 1999, 1999, 0, 2} {
			s.CursorHorizontalAbsolute(x)
			expectEqual[SizeCellCountInt](t, x, s.Cursor.X)
			expectEqual[SizeCellCountInt](t, x, s.Cursor.PagePin.X)
			expectEqual[SizeCellCountInt](t, SizeCellCountInt(y), s.Cursor.Y)
			expect(t, s.Cursor.PageCell == scaCell(t, s, uint(x), y),
				"s.cursor.page_cell == term51AACell(&s, x, y)")
			expect(t, s.Cursor.PageRow == row, "s.cursor.page_row == row")
			expect(t, s.Cursor.PagePin.Node == node, "s.cursor.page_pin.node == node")
			expectEqual[bool](t, pending, s.Cursor.PendingWrap)
			expectEqual[StyleID](t, style_id, s.Cursor.StyleID)
			expectEqual[HyperlinkID](t, link_id, s.Cursor.HyperlinkID)
			scaExpectConsistent(t, s)
			scaExpectMode(t, s, mode)
		}
		scaExpectDirty(t, s, None[uint](), None[uint]())
	}
	s.AssertIntegrity()
}

// Zig: test "term51: cursorHorizontalAbsolute: every row of a multi-page screen, nothing on the cursor"
func TestSca_cursorHorizontalAbsolute_every_row_of_a_multi_page_screen_nothing_on_the_cursor(t *testing.T) {
	testStdCapacity(t)
	scaHorizontalCase(t, 0, scaModeNone, false)
}

// Zig: test "term51: cursorHorizontalAbsolute: every row of a multi-page screen, style and hyperlink"
func TestSca_cursorHorizontalAbsolute_every_row_of_a_multi_page_screen_style_and_hyperlink(t *testing.T) {
	testStdCapacity(t)
	scaHorizontalCase(t, 0, scaModeBoth, false)
	scaHorizontalCase(t, scaBig, scaModeBoth, false)
}

// Zig: test "term51: cursorHorizontalAbsolute: every row of a multi-page screen, pending wrap stays set"
func TestSca_cursorHorizontalAbsolute_every_row_of_a_multi_page_screen_pending_wrap_stays_set(t *testing.T) {
	testStdCapacity(t)
	scaHorizontalCase(t, 0, scaModeBoth, true)
	scaHorizontalCase(t, scaBig, scaModeNone, true)
}

// Zig: test "term51: cursorHorizontalAbsolute: every column pair, with and without pending wrap"
func TestSca_cursorHorizontalAbsolute_every_column_pair_with_and_without_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	for _, mode := range scaModes {
		for _, pending := range []bool{false, true} {
			func() {
				s := scaInit(t, 10, 3, 0)
				defer s.Deinit()
				s.CursorAbsolute(0, 2)
				scaApplyMode(t, s, mode)
				s.Cursor.PendingWrap = pending
				s.Pages.ClearDirty()
				for from := uint(0); from < 10; from++ {
					for to := uint(0); to < 10; to++ {
						s.CursorHorizontalAbsolute(SizeCellCountInt(from))
						expectEqual[SizeCellCountInt](t, SizeCellCountInt(from), s.Cursor.X)
						s.CursorHorizontalAbsolute(SizeCellCountInt(to))
						expectEqual[SizeCellCountInt](t, SizeCellCountInt(to), s.Cursor.X)
						expectEqual[SizeCellCountInt](t, SizeCellCountInt(to), s.Cursor.PagePin.X)
						expectEqual[SizeCellCountInt](t, 2, s.Cursor.Y)
						expect(t, s.Cursor.PageCell == scaCell(t, s, to, 2),
							"s.cursor.page_cell == term51AACell(&s, to, 2)")
						expectEqual[bool](t, pending, s.Cursor.PendingWrap)
					}
				}
				scaExpectConsistent(t, s)
				scaExpectMode(t, s, mode)
				scaExpectDirty(t, s, None[uint](), None[uint]())
			}()
		}
	}
}

// Zig: test "term51: cursorHorizontalAbsolute: one and two column screens"
func TestSca_cursorHorizontalAbsolute_one_and_two_column_screens(t *testing.T) {
	testStdCapacity(t)
	func() {
		s := scaInit(t, 1, 1, 0)
		defer s.Deinit()
		s.Cursor.PendingWrap = true
		s.CursorHorizontalAbsolute(0)
		expectEqual[SizeCellCountInt](t, 0, s.Cursor.X)
		expect(t, s.Cursor.PendingWrap, "s.cursor.pending_wrap")
		scaExpectConsistent(t, s)
	}()
	func() {
		s := scaInit(t, 2, 2, 0)
		defer s.Deinit()
		s.CursorAbsolute(0, 1)
		s.CursorHorizontalAbsolute(1)
		expectEqual[SizeCellCountInt](t, 1, s.Cursor.X)
		expect(t, s.Cursor.PageCell == scaCell(t, s, 1, 1), "s.cursor.page_cell == term51AACell(&s, 1, 1)")
		s.CursorHorizontalAbsolute(0)
		expectEqual[SizeCellCountInt](t, 0, s.Cursor.X)
		expect(t, s.Cursor.PageCell == scaCell(t, s, 0, 1), "s.cursor.page_cell == term51AACell(&s, 0, 1)")
		scaExpectConsistent(t, s)
	}()
}

// Zig: test "term51: cursorHorizontalAbsolute: leaves row flags and cell contents alone"
func TestSca_cursorHorizontalAbsolute_leaves_row_flags_and_cell_contents_alone(t *testing.T) {
	testStdCapacity(t)
	s := scaInit(t, 10, 3, 0)
	defer s.Deinit()
	scaWrite(t, s, "abcdefghij")
	expect(t, s.Cursor.PendingWrap, "s.cursor.pending_wrap")
	s.CursorAbsolute(4, 0)
	scaRow(t, s, 0).wrap = true
	scaRow(t, s, 1).wrap_continuation = true
	row0 := scaRow(t, s, 0).bitCast()
	row1 := scaRow(t, s, 1).bitCast()
	var cells [10]uint64
	for x := uint(0); x < 10; x++ {
		cells[x] = scaCell(t, s, x, 0).bitCast()
	}

	for _, x := range []SizeCellCountInt{0, 9, 4} {
		s.CursorHorizontalAbsolute(x)
		expectEqual[uint64](t, row0, scaRow(t, s, 0).bitCast())
		expectEqual[uint64](t, row1, scaRow(t, s, 1).bitCast())
		for cx := uint(0); cx < 10; cx++ {
			expectEqual[uint64](t, cells[cx], scaCell(t, s, cx, 0).bitCast())
		}
	}
	expectEqual[uint32](t, 'e', s.Cursor.PageCell.Codepoint())
}

// scaLeftCase is term51AALeftCase.
func scaLeftCase(t *testing.T, max_scrollback uint64, mode scaMode, pending bool) {
	t.Helper()
	s := scaMultiPage(t, max_scrollback)
	defer s.Deinit()
	rows := uint(s.Pages.rows)
	scaApplyMode(t, s, mode)

	type Step struct{ n, x SizeCellCountInt }
	steps := []Step{
		{n: 0, x: 1999},
		{n: 1, x: 1998},
		{n: 1, x: 1997},
		{n: 997, x: 1000},
		{n: 999, x: 1},
		{n: 1, x: 0},
		{n: 0, x: 0},
	}

	for y := uint(0); y < rows; y++ {
		s.CursorAbsolute(1999, SizeCellCountInt(y))
		s.Cursor.PendingWrap = pending
		s.Pages.ClearDirty()
		node := s.Cursor.PagePin.Node
		row := s.Cursor.PageRow
		style_id := s.Cursor.StyleID
		link_id := s.Cursor.HyperlinkID

		for _, st := range steps {
			s.CursorLeft(st.n)
			expectEqual[SizeCellCountInt](t, st.x, s.Cursor.X)
			expectEqual[SizeCellCountInt](t, st.x, s.Cursor.PagePin.X)
			expectEqual[SizeCellCountInt](t, SizeCellCountInt(y), s.Cursor.Y)
			expect(t, s.Cursor.PageCell == scaCell(t, s, uint(st.x), y),
				"s.cursor.page_cell == term51AACell(&s, st.x, y)")
			expect(t, s.Cursor.PageRow == row, "s.cursor.page_row == row")
			expect(t, s.Cursor.PagePin.Node == node, "s.cursor.page_pin.node == node")
			expectEqual[bool](t, pending, s.Cursor.PendingWrap)
			expectEqual[StyleID](t, style_id, s.Cursor.StyleID)
			expectEqual[HyperlinkID](t, link_id, s.Cursor.HyperlinkID)
			scaExpectConsistent(t, s)
			scaExpectMode(t, s, mode)
		}
		scaExpectDirty(t, s, None[uint](), None[uint]())
	}
	s.AssertIntegrity()
}

// Zig: test "term51: cursorLeft: every row of a multi-page screen, nothing on the cursor"
func TestSca_cursorLeft_every_row_of_a_multi_page_screen_nothing_on_the_cursor(t *testing.T) {
	testStdCapacity(t)
	scaLeftCase(t, 0, scaModeNone, false)
}

// Zig: test "term51: cursorLeft: every row of a multi-page screen, style and hyperlink"
func TestSca_cursorLeft_every_row_of_a_multi_page_screen_style_and_hyperlink(t *testing.T) {
	testStdCapacity(t)
	scaLeftCase(t, 0, scaModeBoth, false)
	scaLeftCase(t, scaBig, scaModeBoth, false)
}

// Zig: test "term51: cursorLeft: every row of a multi-page screen, pending wrap stays set"
func TestSca_cursorLeft_every_row_of_a_multi_page_screen_pending_wrap_stays_set(t *testing.T) {
	testStdCapacity(t)
	scaLeftCase(t, 0, scaModeBoth, true)
	scaLeftCase(t, scaBig, scaModeNone, true)
}

// Zig: test "term51: cursorLeft: every start column and distance without pending wrap"
func TestSca_cursorLeft_every_start_column_and_distance_without_pending_wrap(t *testing.T) {
	testStdCapacity(t)
	s := scaInit(t, 7, 2, 0)
	defer s.Deinit()
	s.CursorAbsolute(0, 1)
	s.Pages.ClearDirty()
	for x := uint(0); x < 7; x++ {
		for n := uint(0); n < x+1; n++ {
			s.CursorHorizontalAbsolute(SizeCellCountInt(x))
			s.CursorLeft(SizeCellCountInt(n))
			expectEqual[SizeCellCountInt](t, SizeCellCountInt(x-n), s.Cursor.X)
			expectEqual[SizeCellCountInt](t, SizeCellCountInt(x-n), s.Cursor.PagePin.X)
			expect(t, s.Cursor.PageCell == scaCell(t, s, x-n, 1),
				"s.cursor.page_cell == term51AACell(&s, x - n, 1)")
			expect(t, !s.Cursor.PendingWrap, "!s.cursor.pending_wrap")
			scaExpectConsistent(t, s)
		}
	}
	scaExpectDirty(t, s, None[uint](), None[uint]())
}

// Zig: test "term51: cursorLeft: one column screen"
func TestSca_cursorLeft_one_column_screen(t *testing.T) {
	testStdCapacity(t)
	s := scaInit(t, 1, 3, 0)
	defer s.Deinit()
	s.CursorAbsolute(0, 2)
	s.Cursor.PendingWrap = true
	s.CursorLeft(0)
	expectEqual[SizeCellCountInt](t, 0, s.Cursor.X)
	expect(t, s.Cursor.PendingWrap, "s.cursor.pending_wrap")
	scaExpectConsistent(t, s)
}

// Zig: test "term51: cursorCellRight: every start column and distance on small screens"
func TestSca_cursorCellRight_every_start_column_and_distance_on_small_screens(t *testing.T) {
	testStdCapacity(t)
	for _, cols := range []SizeCellCountInt{1, 2, 6} {
		func() {
			s := scaInit(t, cols, 2, 0)
			defer s.Deinit()
			for x := uint(0); x < uint(cols); x++ {
				scaSetCp(t, s, x, 1, uint32('a'+x))
			}
			for x := uint(0); x < uint(cols); x++ {
				s.CursorAbsolute(SizeCellCountInt(x), 1)
				s.Pages.ClearDirty()
				before := s.Cursor
				for n := uint(0); n < uint(cols)-x; n++ {
					cell := s.CursorCellRight(SizeCellCountInt(n))
					expect(t, cell == scaCell(t, s, x+n, 1), "cell == term51AACell(&s, x + n, 1)")
					expectEqual[uint32](t, uint32('a'+x+n), cell.Codepoint())
				}
				expect(t, before == s.Cursor, "std.meta.eql(before, s.cursor)")
				scaExpectDirty(t, s, None[uint](), None[uint]())
			}
		}()
	}
}

// Zig: test "term51: cursorCellLeft: every start column and distance on small screens"
func TestSca_cursorCellLeft_every_start_column_and_distance_on_small_screens(t *testing.T) {
	testStdCapacity(t)
	for _, cols := range []SizeCellCountInt{1, 2, 6} {
		func() {
			s := scaInit(t, cols, 2, 0)
			defer s.Deinit()
			for x := uint(0); x < uint(cols); x++ {
				scaSetCp(t, s, x, 1, uint32('a'+x))
			}
			for x := uint(0); x < uint(cols); x++ {
				s.CursorAbsolute(SizeCellCountInt(x), 1)
				s.Pages.ClearDirty()
				before := s.Cursor
				for n := uint(0); n < x+1; n++ {
					cell := s.CursorCellLeft(SizeCellCountInt(n))
					expect(t, cell == scaCell(t, s, x-n, 1), "cell == term51AACell(&s, x - n, 1)")
					expectEqual[uint32](t, uint32('a'+x-n), cell.Codepoint())
				}
				expect(t, before == s.Cursor, "std.meta.eql(before, s.cursor)")
				scaExpectDirty(t, s, None[uint](), None[uint]())
			}
		}()
	}
}

// Zig: test "term51: cursorCellRight: every row of a multi-page screen with pending wrap and a style"
func TestSca_cursorCellRight_every_row_of_a_multi_page_screen_with_pending_wrap_and_a_style(t *testing.T) {
	testStdCapacity(t)
	s := scaMultiPage(t, 0)
	defer s.Deinit()
	scaApplyMode(t, s, scaModeBoth)
	for y := uint(0); y < uint(s.Pages.rows); y++ {
		scaSetCp(t, s, 1999, y, uint32(0x200+y))
		for _, x := range []SizeCellCountInt{0, 1, 1000, 1998, 1999} {
			s.CursorAbsolute(x, SizeCellCountInt(y))
			s.Cursor.PendingWrap = true
			s.Pages.ClearDirty()
			before := s.Cursor
			for _, n := range []uint{0, 1, 1999 - uint(x)} {
				if uint(x)+n >= 2000 {
					continue
				}
				expect(t, s.CursorCellRight(SizeCellCountInt(n)) == scaCell(t, s, uint(x)+n, y),
					"s.cursorCellRight(n) == term51AACell(&s, x + n, y)")
			}
			expectEqual[uint32](t, uint32(0x200+y), s.CursorCellRight(1999-x).Codepoint())
			expect(t, before == s.Cursor, "std.meta.eql(before, s.cursor)")
			scaExpectDirty(t, s, None[uint](), None[uint]())
		}
	}
	scaExpectMode(t, s, scaModeBoth)
}

// Zig: test "term51: cursorCellLeft: every row of a multi-page screen with pending wrap and a style"
func TestSca_cursorCellLeft_every_row_of_a_multi_page_screen_with_pending_wrap_and_a_style(t *testing.T) {
	testStdCapacity(t)
	s := scaMultiPage(t, 0)
	defer s.Deinit()
	scaApplyMode(t, s, scaModeBoth)
	for y := uint(0); y < uint(s.Pages.rows); y++ {
		scaSetCp(t, s, 0, y, uint32(0x200+y))
		for _, x := range []SizeCellCountInt{0, 1, 1000, 1998, 1999} {
			s.CursorAbsolute(x, SizeCellCountInt(y))
			s.Cursor.PendingWrap = true
			s.Pages.ClearDirty()
			before := s.Cursor
			for _, n := range []uint{0, 1, uint(x)} {
				if n > uint(x) {
					continue
				}
				expect(t, s.CursorCellLeft(SizeCellCountInt(n)) == scaCell(t, s, uint(x)-n, y),
					"s.cursorCellLeft(n) == term51AACell(&s, x - n, y)")
			}
			expectEqual[uint32](t, uint32(0x200+y), s.CursorCellLeft(x).Codepoint())
			expect(t, before == s.Cursor, "std.meta.eql(before, s.cursor)")
			scaExpectDirty(t, s, None[uint](), None[uint]())
		}
	}
	scaExpectMode(t, s, scaModeBoth)
}

// Zig: test "term51: cursorCellRight: layout pointer distance is n cells"
func TestSca_cursorCellRight_layout_pointer_distance_is_n_cells(t *testing.T) {
	testStdCapacity(t)
	s := scaInit(t, 10, 1, 0)
	defer s.Deinit()
	s.CursorAbsolute(2, 0)
	// Go: the distance in bytes (n * @sizeOf(Cell)) is a distance in the
	// page's cells slice.
	page := &s.Cursor.PagePin.Node.data
	base := page.cellIndex(s.Cursor.PageCell)
	for n := uint(0); n < 8; n++ {
		expectEqual[uint](t, base+n, page.cellIndex(s.CursorCellRight(SizeCellCountInt(n))))
	}
}

// Zig: test "term51: cursorCellLeft: layout pointer distance is n cells"
func TestSca_cursorCellLeft_layout_pointer_distance_is_n_cells(t *testing.T) {
	testStdCapacity(t)
	s := scaInit(t, 10, 1, 0)
	defer s.Deinit()
	s.CursorAbsolute(7, 0)
	// Go: the distance in bytes (n * @sizeOf(Cell)) is a distance in the
	// page's cells slice.
	page := &s.Cursor.PagePin.Node.data
	base := page.cellIndex(s.Cursor.PageCell)
	for n := uint(0); n < 8; n++ {
		expectEqual[uint](t, base-n, page.cellIndex(s.CursorCellLeft(SizeCellCountInt(n))))
	}
}

// Zig: test "term51: cursorMarkDirty: every row of a multi-page screen marks the row and no page"
func TestSca_cursorMarkDirty_every_row_of_a_multi_page_screen_marks_the_row_and_no_page(t *testing.T) {
	testStdCapacity(t)
	s := scaMultiPage(t, 0)
	defer s.Deinit()
	scaApplyMode(t, s, scaModeBoth)
	for y := uint(0); y < uint(s.Pages.rows); y++ {
		s.CursorAbsolute(SizeCellCountInt((y*31)%2000), SizeCellCountInt(y))
		s.Cursor.PendingWrap = (y%2 == 0)
		s.Pages.ClearDirty()
		row_bits := s.Cursor.PageRow.bitCast()
		before := s.Cursor

		s.CursorMarkDirty()
		scaExpectDirty(t, s, Some(y), None[uint]())
		// Only the dirty bit changed in the row.
		expect_row := *s.Cursor.PageRow
		expect_row.dirty = false
		was_row := pageRowFromBits(row_bits)
		was_row.dirty = false
		expectEqual[uint64](t, was_row.bitCast(), expect_row.bitCast())
		expect(t, before == s.Cursor, "std.meta.eql(before, s.cursor)")

		// Marking again changes nothing.
		s.CursorMarkDirty()
		scaExpectDirty(t, s, Some(y), None[uint]())
	}
	scaExpectMode(t, s, scaModeBoth)
}

// Zig: test "term51: cursorMarkDirty: keeps rows that were already dirty"
func TestSca_cursorMarkDirty_keeps_rows_that_were_already_dirty(t *testing.T) {
	testStdCapacity(t)
	s := scaInit(t, 5, 6, 0)
	defer s.Deinit()
	s.CursorAbsolute(0, 3)
	s.Pages.ClearDirty()
	scaRow(t, s, 1).dirty = true
	scaRow(t, s, 4).dirty = true

	s.CursorMarkDirty()
	var got [6]bool
	for y := uint(0); y < 6; y++ {
		got[y] = scaRow(t, s, y).dirty
	}
	expectEqualSlices[bool](t, []bool{false, true, false, true, true, false}, got[:])

	// A row that is dirty already stays dirty.
	s.CursorMarkDirty()
	expect(t, scaRow(t, s, 3).dirty, "term51AARow(&s, 3).dirty")
	s.CursorAbsolute(0, 1)
	s.CursorMarkDirty()
	expect(t, scaRow(t, s, 1).dirty, "term51AARow(&s, 1).dirty")
}

// Zig: test "term51: viewportIsBottom: every viewport kind set directly"
func TestSca_viewportIsBottom_every_viewport_kind_set_directly(t *testing.T) {
	testStdCapacity(t)
	s := scaInit(t, 5, 3, 1000)
	defer s.Deinit()
	expect(t, s.Pages.viewport.Tag == PageListViewportActive, "s.pages.viewport == .active")
	expect(t, s.ViewportIsBottom(), "s.viewportIsBottom()")

	s.Pages.viewport = PageListViewport{Tag: PageListViewportTop}
	expect(t, !s.ViewportIsBottom(), "!s.viewportIsBottom()")
	s.Pages.viewport = PageListViewport{Tag: PageListViewportPin}
	expect(t, !s.ViewportIsBottom(), "!s.viewportIsBottom()")
	s.Pages.viewport = PageListViewport{Tag: PageListViewportActive}
	expect(t, s.ViewportIsBottom(), "s.viewportIsBottom()")
}

// Zig: test "term51: viewportIsBottom: output while scrolled up does not return to the bottom"
func TestSca_viewportIsBottom_output_while_scrolled_up_does_not_return_to_the_bottom(t *testing.T) {
	testStdCapacity(t)
	s := scaInit(t, 5, 3, 1000)
	defer s.Deinit()
	scaWrite(t, s, "a0\na1\na2\na3\na4\na5")
	expect(t, s.ViewportIsBottom(), "s.viewportIsBottom()")

	s.Scroll(ScreenScroll{Tag: ScreenScrollTop})
	expect(t, !s.ViewportIsBottom(), "!s.viewportIsBottom()")
	scaWrite(t, s, "\na6\na7")
	expect(t, !s.ViewportIsBottom(), "!s.viewportIsBottom()")

	s.Scroll(ScreenScroll{Tag: ScreenScrollActive})
	expect(t, s.ViewportIsBottom(), "s.viewportIsBottom()")
	scaWrite(t, s, "\na8")
	expect(t, s.ViewportIsBottom(), "s.viewportIsBottom()")
}

// Zig: test "term51: viewportIsBottom: reset returns to the bottom"
func TestSca_viewportIsBottom_reset_returns_to_the_bottom(t *testing.T) {
	testStdCapacity(t)
	s := scaInit(t, 5, 3, 1000)
	defer s.Deinit()
	scaWrite(t, s, "a0\na1\na2\na3\na4\na5")
	s.Scroll(ScreenScroll{Tag: ScreenScrollTop})
	expect(t, !s.ViewportIsBottom(), "!s.viewportIsBottom()")
	s.Reset()
	expect(t, s.ViewportIsBottom(), "s.viewportIsBottom()")
}

// scaExplicit is the Zig literal `.{ .uri = uri, .id = .{ .explicit = id } }`.
func scaExplicit(uri string, id string) Hyperlink {
	return Hyperlink{
		URI: []uint8(uri),
		ID:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdExplicit, Explicit: []uint8(id)},
	}
}

// Zig: test "term51: startHyperlinkOnce: an equal link reuses the page entry and counts references"
func TestSca_startHyperlinkOnce_an_equal_link_reuses_the_page_entry_and_counts_references(t *testing.T) {
	testStdCapacity(t)
	s := scaInit(t, 10, 3, 0)
	defer s.Deinit()
	page := &s.Cursor.PagePin.Node.data

	zigTry(t, s.startHyperlinkOnce(scaExplicit("https://a/", "a")))
	id := s.Cursor.HyperlinkID
	scaWrite(t, s, "xy")
	expectEqual[uint](t, 3, scaLinkRefs(page, id))

	// The cursor's own reference is released first, then re-added: the cells
	// keep theirs and the set still holds a single entry.
	zigTry(t, s.startHyperlinkOnce(scaExplicit("https://a/", "a")))
	expectEqual[HyperlinkID](t, id, s.Cursor.HyperlinkID)
	expectEqual[uint](t, 3, scaLinkRefs(page, id))
	expectEqual[uint](t, 1, page.hyperlink_set.Count())

	// A different link takes another id, the cells keep the first.
	zigTry(t, s.startHyperlinkOnce(scaExplicit("https://a/", "b")))
	expect(t, s.Cursor.HyperlinkID != id, "s.cursor.hyperlink_id != id")
	expectEqual[uint](t, 2, scaLinkRefs(page, id))
	expectEqual[uint](t, 2, page.hyperlink_set.Count())
	expectEqualStrings(t, "b", string(s.Cursor.Hyperlink.ID.Explicit))
	s.AssertIntegrity()
}

// Zig: test "term51: startHyperlinkOnce: the link goes into the cursor's page only"
func TestSca_startHyperlinkOnce_the_link_goes_into_the_cursor_s_page_only(t *testing.T) {
	testStdCapacity(t)
	s := scaMultiPage(t, 0)
	defer s.Deinit()
	s.CursorAbsolute(3, 59)
	node := s.Cursor.PagePin.Node
	expect(t, node != s.Pages.pages.first, "node != s.pages.pages.first.?")

	zigTry(t, s.startHyperlinkOnce(Hyperlink{
		URI: []uint8("https://a/"),
		ID:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: 5},
	}))
	expectEqual[uint](t, 1, scaLinkRefs(&node.data, s.Cursor.HyperlinkID))
	expectEqual[uint](t, 1, node.data.hyperlink_set.Count())
	for n := s.Pages.pages.first; n != nil; n = n.next {
		if n != node {
			expectEqual[uint](t, 0, n.data.hyperlink_set.Count())
		}
	}
	expectEqual[SizeOffsetInt](t, 5, s.Cursor.Hyperlink.ID.Implicit)
	// The implicit counter of the cursor is not used or advanced.
	expectEqual[SizeOffsetInt](t, 0, s.Cursor.HyperlinkImplicitID)
}

// Zig: test "term51: startHyperlinkOnce: the strings are copied"
func TestSca_startHyperlinkOnce_the_strings_are_copied(t *testing.T) {
	testStdCapacity(t)
	s := scaInit(t, 10, 3, 0)
	defer s.Deinit()
	uri := []uint8("https://a/")
	id := []uint8("id")
	zigTry(t, s.startHyperlinkOnce(Hyperlink{
		URI: uri,
		ID:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdExplicit, Explicit: id},
	}))
	uri[0] = 'X'
	id[0] = 'Y'
	link := s.Cursor.Hyperlink
	expectEqualStrings(t, "https://a/", string(link.URI))
	expectEqualStrings(t, "id", string(link.ID.Explicit))
	expect(t, &link.URI[0] != &uri[0], "link.uri.ptr != @as([*]const u8, &uri)")
	page := &s.Cursor.PagePin.Node.data
	entry := page.hyperlink_set.Get(s.Cursor.HyperlinkID)
	expectEqualStrings(t, "https://a/", string(entry.uri))
	expectEqualStrings(t, "id", string(entry.id.Explicit))
}
