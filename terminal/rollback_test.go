package terminal

// Tests of the rollback of partially done operations, reached through the
// CAPACITY errors that exist in Go (a full or maxed hyperlink set, a full
// style set, a fragmented grapheme store). They translate the Zig tests under
// the banner `term51: additional tests (AF)` at the end of Screen.zig,
// PageList.zig and Terminal.zig. The Zig tests of the same functions that
// inject allocation failures (FailingAllocator, tripwires, pools) are not
// translated, see docs/mapping.md and scripts/not-translated.txt.
//
// The Go port has no allocator that can fail, so every rollback that Zig
// reaches only through a failing allocation is unreachable here.

import (
	"errors"
	"testing"
)

// rbInit is `try init(alloc, .{ .cols = cols, .rows = rows, .max_scrollback = 0 })`.
func rbInit(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) *Screen {
	t.Helper()
	s, err := ScreenInit(ScreenOptions{
		Cols:                   cols,
		Rows:                   rows,
		MaxScrollback:          0,
		KittyImageStorageLimit: 320 * 1000 * 1000,
	})
	zigTry(t, err)
	return &s
}

// rbExpectError is `try testing.expectError(want, got)`.
func rbExpectError(t *testing.T, want error, got error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

// rbAttr is the SGR attribute literal `.bold` / `.italic`.
func rbAttr(tag SGRAttributeTag) SGRAttribute {
	return SGRAttribute{Tag: tag}
}

// rbWrite is `try s.testWriteString(text)`.
func rbWrite(t *testing.T, s *Screen, text string) {
	t.Helper()
	zigTry(t, s.TestWriteString([]uint8(text)))
}

// rbStartLink is `try s.startHyperlink(uri, id)`.
func rbStartLink(t *testing.T, s *Screen, uri string, id Optional[[]uint8]) {
	t.Helper()
	zigTry(t, s.StartHyperlink([]uint8(uri), id))
}

// rbMaxLinkStores is term51AFMaxLinkStores: grow the cursor page until its
// string store holds thousands of chunks and its hyperlink set has the
// largest capacity a page can have (hyperlink_bytes at the 16 bit limit).
func rbMaxLinkStores(t *testing.T, s *Screen) {
	t.Helper()
	for i := 0; i < 6; i++ {
		_, err := s.IncreaseCapacity(s.Cursor.PagePin.Node, Some(PageListIncreaseCapacityStringBytes))
		zigTry(t, err)
	}
	steps := 0
	for {
		_, err := s.IncreaseCapacity(s.Cursor.PagePin.Node, Some(PageListIncreaseCapacityHyperlinkBytes))
		if err != nil {
			rbExpectError(t, ErrOutOfSpace, err)
			break
		}
		steps++
		expect(t, steps < 64, "steps < 64")
	}
	expectEqual(t, SizeHyperlinkCountInt(1<<16-1), s.Cursor.PagePin.Node.data.capacity.hyperlink_bytes)
}

// rbFilled is term51AFFilled.
type rbFilled struct {
	count uint
	last  HyperlinkID
}

// rbFillLinks is term51AFFillLinks: insert distinct one-chunk links straight
// into the page (one reference each, no cells) until the hyperlink set is
// full.
func rbFillLinks(t *testing.T, page *Page) rbFilled {
	t.Helper()
	var n uint
	var last HyperlinkID
	for {
		id, err := page.InsertHyperlink(Hyperlink{
			ID:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: SizeOffsetInt(100 + n)},
			URI: []uint8("u"),
		})
		if err != nil {
			rbExpectError(t, ErrSetOutOfMemory, err)
			return rbFilled{count: n, last: last}
		}
		last = id
		expect(t, n < 100_000, "n < 100_000")
		n++
	}
}

// rbMaxStyles is term51AFMaxStyles: grow the cursor page's style capacity to
// the maximum.
func rbMaxStyles(s *Screen) {
	const max = 1<<16 - 1
	for s.Cursor.PagePin.Node.data.capacity.styles < max {
		_, err := s.IncreaseCapacity(s.Cursor.PagePin.Node, Some(PageListIncreaseCapacityStyles))
		if err != nil {
			break
		}
	}
}

// rbFillStyles is term51AFFillStyles: add distinct background-colour styles
// until the set refuses more.
func rbFillStyles(page *Page) uint {
	// Zig: n is a u24 that is @bitCast into the RGB of the background.
	var n uint32 = 1
	for {
		_, err := page.styles.Add(Style{
			BgColor: StyleColor{
				Tag: StyleColorRgb,
				Rgb: ColorRGB{R: uint8(n), G: uint8(n >> 8), B: uint8(n >> 16)},
			},
		})
		if err != nil {
			break
		}
		n += 1
	}
	return uint(n - 1)
}

// rbGraphemeCell is the element type of the Zig `time_order` array.
type rbGraphemeCell struct {
	x   uint
	cps []uint32
}

// rbRepeatCps is `[_]u21{cp} ** n`.
func rbRepeatCps(cp uint32, n int) []uint32 {
	r := make([]uint32, n)
	for i := range r {
		r[i] = cp
	}
	return r
}

// Zig: test "term51: Screen.startHyperlink: out of space rolls back the implicit id and leaves no link by a full hyperlink set"
func TestRollback_Screen_startHyperlink_out_of_space_rolls_back_the_implicit_id_and_leaves_no_link_by_a_full_hyperlink_set(t *testing.T) {
	testStdCapacity(t)
	s := rbInit(t, 10, 3)
	defer s.Deinit()
	rbMaxLinkStores(t, s)

	// An implicit link used by one cell, so that ending it frees no slot.
	rbStartLink(t, s, "https://old/", None[[]uint8]())
	rbWrite(t, s, "x")
	node := s.Cursor.PagePin.Node
	page := &node.data
	old_id := s.Cursor.HyperlinkID
	expectEqual(t, SizeOffsetInt(1), s.Cursor.HyperlinkImplicitID)
	expectEqual(t, 2, page.hyperlink_set.RefCount(old_id))

	filled := rbFillLinks(t, page)
	expect(t, filled.count > 100, "filled.count > 100")
	count := page.hyperlink_set.Count()
	strings := page.string_alloc.UsedBytes()
	cap := page.capacity

	// The set is full of live links and cannot grow: OutOfSpace, nothing is
	// started and the implicit counter is where it was.
	rbExpectError(t, ErrOutOfSpace, s.StartHyperlink([]uint8("https://new/"), None[[]uint8]()))
	expect(t, s.Cursor.PagePin.Node == node, "s.cursor.page_pin.node == node")
	expectEqual(t, cap, node.data.capacity)
	expectEqual(t, HyperlinkID(0), s.Cursor.HyperlinkID)
	expect(t, s.Cursor.Hyperlink == nil, "s.cursor.hyperlink == null")
	expectEqual(t, SizeOffsetInt(1), s.Cursor.HyperlinkImplicitID)
	// The previous link was ended: only its cell keeps it. The failed
	// insert freed its strings again.
	expectEqual(t, count, page.hyperlink_set.Count())
	expectEqual(t, 1, page.hyperlink_set.RefCount(old_id))
	expectEqual(t, strings, page.string_alloc.UsedBytes())
	expect(t, page.LookupHyperlink(&page.GetCells(page.GetRow(0))[0]).Valid, "page.lookupHyperlink(&page.getCells(page.getRow(0))[0]) != null")
	s.AssertIntegrity()
	page.AssertIntegrity()

	// An explicit link fails the same way; there is no counter to restore.
	rbExpectError(t, ErrOutOfSpace, s.StartHyperlink([]uint8("https://new/"), Some([]uint8("new"))))
	expectEqual(t, HyperlinkID(0), s.Cursor.HyperlinkID)
	expect(t, s.Cursor.Hyperlink == nil, "s.cursor.hyperlink == null")
	expectEqual(t, SizeOffsetInt(1), s.Cursor.HyperlinkImplicitID)
	expectEqual(t, count, page.hyperlink_set.Count())
	expectEqual(t, strings, page.string_alloc.UsedBytes())
	s.AssertIntegrity()
	page.AssertIntegrity()

	// Free the top slot of the set: the next implicit link succeeds and gets
	// the id the failed attempt did not consume.
	page.hyperlink_set.Release(filled.last)
	rbStartLink(t, s, "https://new/", None[[]uint8]())
	expectEqual(t, SizeOffsetInt(1), s.Cursor.Hyperlink.ID.Implicit)
	expectEqual(t, SizeOffsetInt(2), s.Cursor.HyperlinkImplicitID)
	expectEqualStrings(t, "https://new/", string(s.Cursor.Hyperlink.URI))
	expect(t, s.Cursor.PagePin.Node == node, "s.cursor.page_pin.node == node")
	s.AssertIntegrity()
	page.AssertIntegrity()
}

// Zig: test "term51: Screen.increaseCapacity: out of space leaves the cursor style and hyperlink untouched by a maxed hyperlink set"
func TestRollback_Screen_increaseCapacity_out_of_space_leaves_the_cursor_style_and_hyperlink_untouched_by_a_maxed_hyperlink_set(t *testing.T) {
	testStdCapacity(t)
	s := rbInit(t, 10, 3)
	defer s.Deinit()
	rbMaxLinkStores(t, s)
	zigTry(t, s.SetAttribute(rbAttr(SGRAttributeTagBold)))
	rbStartLink(t, s, "https://example.com/", Some([]uint8("id")))
	rbWrite(t, s, "ab")

	node := s.Cursor.PagePin.Node
	page := &node.data
	link := s.Cursor.Hyperlink
	style_id := s.Cursor.StyleID
	link_id := s.Cursor.HyperlinkID
	cap := page.capacity
	page_size := s.Pages.page_size
	serial := s.Pages.page_serial
	expectEqual(t, 3, page.styles.RefCount(style_id))
	expectEqual(t, 3, page.hyperlink_set.RefCount(link_id))

	// The failed call changed nothing: same node, same heap link, same ids
	// and reference counts.
	_, err := s.IncreaseCapacity(node, Some(PageListIncreaseCapacityHyperlinkBytes))
	rbExpectError(t, ErrOutOfSpace, err)
	expect(t, s.Cursor.PagePin.Node == node, "s.cursor.page_pin.node == node")
	expect(t, s.Pages.pages.first == node, "s.pages.pages.first == node")
	expect(t, s.Pages.pages.last == node, "s.pages.pages.last == node")
	expectEqual(t, cap, page.capacity)
	expectEqual(t, page_size, s.Pages.page_size)
	expectEqual(t, serial, s.Pages.page_serial)
	expect(t, s.Cursor.Hyperlink == link, "s.cursor.hyperlink.? == link")
	expectEqual(t, link_id, s.Cursor.HyperlinkID)
	expectEqual(t, style_id, s.Cursor.StyleID)
	expect(t, s.Cursor.Style.Flags.Bold, "s.cursor.style.flags.bold")
	expectEqual(t, 3, page.styles.RefCount(style_id))
	expectEqual(t, 3, page.hyperlink_set.RefCount(link_id))
	expectEqual(t, uint(1), page.hyperlink_set.Count())
	expectEqual(t, uint(2), page.HyperlinkCount())
	s.AssertIntegrity()
	page.AssertIntegrity()

	// Another dimension of the same page still grows and keeps the cursor
	// working: the style and the link are re-added to the new page.
	grown, err := s.IncreaseCapacity(node, Some(PageListIncreaseCapacityStringBytes))
	zigTry(t, err)
	expect(t, grown != node, "grown != node")
	expect(t, s.Cursor.PagePin.Node == grown, "s.cursor.page_pin.node == grown")
	expectEqualStrings(t, "https://example.com/", string(s.Cursor.Hyperlink.URI))
	expectEqualStrings(t, "id", string(s.Cursor.Hyperlink.ID.Explicit))
	expectEqual(t, 3, grown.data.styles.RefCount(s.Cursor.StyleID))
	expectEqual(t, 3, grown.data.hyperlink_set.RefCount(s.Cursor.HyperlinkID))
	s.AssertIntegrity()
	grown.data.AssertIntegrity()
}

// Zig: test "term51: Screen.cursorCopy: a style that does not fit restores the previous cursor by a full style set"
func TestRollback_Screen_cursorCopy_a_style_that_does_not_fit_restores_the_previous_cursor_by_a_full_style_set(t *testing.T) {
	testStdCapacity(t)
	s := rbInit(t, 10, 1)
	defer s.Deinit()
	other := rbInit(t, 10, 1)
	defer other.Deinit()
	zigTry(t, other.SetAttribute(rbAttr(SGRAttributeTagItalic)))
	expect(t, other.Cursor.StyleID != StyleDefaultID, "other.cursor.style_id != style.default_id")

	rbMaxStyles(s)
	zigTry(t, s.SetAttribute(rbAttr(SGRAttributeTagBold)))
	rbWrite(t, s, "a")
	node := s.Cursor.PagePin.Node
	page := &node.data
	bold_id := s.Cursor.StyleID
	rbFillStyles(page)
	expectEqual(t, 2, page.styles.RefCount(bold_id))
	count := page.styles.Count()

	// The one-row page cannot be split and the styles cannot grow.
	rbExpectError(t, ErrOutOfSpace, s.CursorCopy(other.Cursor, screenCursorCopyOptionsDefault()))

	// The cursor is back to what it was before: same position, same style
	// (taken from the old copy) and the same node.
	expect(t, s.Cursor.PagePin.Node == node, "s.cursor.page_pin.node == node")
	expectEqual(t, SizeCellCountInt(1), s.Cursor.X)
	expectEqual(t, SizeCellCountInt(0), s.Cursor.Y)
	expect(t, s.Cursor.Style.Flags.Bold, "s.cursor.style.flags.bold")
	expect(t, !s.Cursor.Style.Flags.Italic, "!s.cursor.style.flags.italic")
	expectEqual(t, bold_id, s.Cursor.StyleID)
	expect(t, s.Cursor.Hyperlink == nil, "s.cursor.hyperlink == null")
	// The other cursor and its page were not touched.
	expect(t, other.Cursor.Style.Flags.Italic, "other.cursor.style.flags.italic")
	opage := &other.Cursor.PagePin.Node.data
	expectEqual(t, 1, opage.styles.RefCount(other.Cursor.StyleID))

	// Upstream defect pinned: the update released the cursor's reference on
	// the old style before it failed, and restoring the old cursor does not
	// take it back: the cursor names an id it holds no reference on.
	expectEqual(t, 1, page.styles.RefCount(bold_id))
	expectEqual(t, count, page.styles.Count())
	page.AssertIntegrity()
}

// Zig: test "term51: Screen.clone: a page that does not fit when cloned fails and leaves the source untouched by a fragmented grapheme store"
func TestRollback_Screen_clone_a_page_that_does_not_fit_when_cloned_fails_and_leaves_the_source_untouched_by_a_fragmented_grapheme_store(t *testing.T) {
	testStdCapacity(t)
	// 3353 columns leave the page room to double its grapheme bytes twice
	// and still be a standard size (pooled) page.
	s := rbInit(t, 3353, 5)
	defer s.Deinit()
	_, err := s.IncreaseCapacity(s.Cursor.PagePin.Node, Some(PageListIncreaseCapacityGraphemeBytes))
	zigTry(t, err)
	_, err = s.IncreaseCapacity(s.Cursor.PagePin.Node, Some(PageListIncreaseCapacityGraphemeBytes))
	zigTry(t, err)
	page := &s.Cursor.PagePin.Node.data
	expectEqual(t, uint32(2048), page.capacity.grapheme_bytes)
	expectEqual(t, PageLayoutOf(PageStdCapacity).total_size, pageListMemoryLen(page))
	// 128 chunks of 16 bytes: two bitmaps of 64 chunks. A request of up to
	// 64 chunks never spans bitmaps.
	expectEqual(t, uint64(2048), page.grapheme_alloc.CapacityBytes())

	// Allocated in this order the four graphemes (40, 24, 40 and 24 chunks)
	// fill the store exactly: 40+24 in the first bitmap, 40+24 in the second.
	// They are on cells 2, 0, 3 and 1 of the first row.
	long := rbRepeatCps(0x301, 160)
	short := rbRepeatCps(0x302, 96)
	time_order := []rbGraphemeCell{
		{x: 2, cps: long},
		{x: 0, cps: short},
		{x: 3, cps: long},
		{x: 1, cps: short},
	}
	for _, e := range time_order {
		rac := page.GetRowAndCell(e.x, 0)
		*rac.cell = PageCellInit('a')
		zigTry(t, page.SetGraphemes(rac.row, rac.cell, e.cps))
	}
	expectEqual(t, uint64(2048), page.grapheme_alloc.UsedBytes())
	expectEqual(t, uint(4), page.GraphemeCount())
	page.AssertIntegrity()

	// A clone allocates them in cell order: 24, 24 (first bitmap, 16 chunks
	// left), 40 (second bitmap, 24 left) and the last 40 fits nowhere
	// although the 128 chunks are enough in total.
	pins := s.Pages.CountTrackedPins()
	rows := s.Pages.total_rows
	page_size := s.Pages.page_size
	serial := s.Pages.page_serial
	_, err = s.Clone(Point{Tag: PointTagScreen}, None[Point]())
	rbExpectError(t, ErrGraphemeAllocOutOfMemory, err)

	// The source is untouched.
	expectEqual(t, pins, s.Pages.CountTrackedPins())
	expectEqual(t, rows, s.Pages.total_rows)
	expectEqual(t, page_size, s.Pages.page_size)
	expectEqual(t, serial, s.Pages.page_serial)
	expectEqual(t, uint64(2048), page.grapheme_alloc.UsedBytes())
	expectEqual(t, uint(4), page.GraphemeCount())
	expectEqual(t, 160, len(zigUnwrap(t, page.LookupGrapheme(page.GetRowAndCell(2, 0).cell))))
	expectEqual(t, 96, len(zigUnwrap(t, page.LookupGrapheme(page.GetRowAndCell(1, 0).cell))))
	s.AssertIntegrity()
	page.AssertIntegrity()

	// Control: without the last grapheme the other three fit.
	s.ClearCells(page, page.GetRow(0), page.GetCells(page.GetRow(0))[3:4])
	expectEqual(t, uint(3), page.GraphemeCount())
	cp, err := s.Clone(Point{Tag: PointTagScreen}, None[Point]())
	zigTry(t, err)
	defer cp.Deinit()
	expectEqual(t, uint(3), cp.Cursor.PagePin.Node.data.GraphemeCount())
}

// Zig: test "term51: PageList.clone: a page that does not fit when cloned fails and leaves the source untouched by a fragmented grapheme store"
func TestRollback_PageList_clone_a_page_that_does_not_fit_when_cloned_fails_and_leaves_the_source_untouched_by_a_fragmented_grapheme_store(t *testing.T) {
	testStdCapacity(t)
	// 3353 columns leave a page room to double its grapheme bytes twice and
	// still be a standard size (pooled) page; 13 rows fit one page.
	l, err := PageListInit(3353, 20, None[uint64]())
	zigTry(t, err)
	s := &l
	defer s.Deinit()
	expectEqual(t, uint(2), s.TotalPages())
	second, err := s.IncreaseCapacity(s.pages.last, Some(PageListIncreaseCapacityGraphemeBytes))
	zigTry(t, err)
	last, err := s.IncreaseCapacity(second, Some(PageListIncreaseCapacityGraphemeBytes))
	zigTry(t, err)
	expect(t, s.pages.last == last, "s.pages.last.? == last")
	page := &last.data
	expectEqual(t, uint32(2048), page.capacity.grapheme_bytes)
	expectEqual(t, PageLayoutOf(PageStdCapacity).total_size, pageListMemoryLen(page))
	// 128 chunks of 16 bytes: two bitmaps of 64 chunks. A request of up to
	// 64 chunks never spans bitmaps.
	expectEqual(t, uint64(2048), page.grapheme_alloc.CapacityBytes())

	// Allocated in this order the four graphemes (40, 24, 40 and 24 chunks)
	// fill the store exactly: 40+24 in the first bitmap, 40+24 in the second.
	// They are on cells 2, 0, 3 and 1 of the first row of the last page.
	long := rbRepeatCps(0x301, 160)
	short := rbRepeatCps(0x302, 96)
	time_order := []rbGraphemeCell{
		{x: 2, cps: long},
		{x: 0, cps: short},
		{x: 3, cps: long},
		{x: 1, cps: short},
	}
	for _, e := range time_order {
		rac := page.GetRowAndCell(e.x, 0)
		*rac.cell = PageCellInit('a')
		zigTry(t, page.SetGraphemes(rac.row, rac.cell, e.cps))
	}
	expectEqual(t, uint64(2048), page.grapheme_alloc.UsedBytes())
	expectEqual(t, uint(4), page.GraphemeCount())
	page.AssertIntegrity()

	// A clone allocates them in cell order: 24, 24 (first bitmap, 16 chunks
	// left), 40 (second bitmap, 24 left) and the last 40 fits nowhere
	// although the 128 chunks are enough in total. The first page was cloned
	// already and is given back.
	pins := s.CountTrackedPins()
	total_rows := s.total_rows
	page_size := s.page_size
	serial := s.page_serial
	first := s.pages.first
	_, err = s.Clone(PageListClone{Top: Point{Tag: PointTagScreen}})
	rbExpectError(t, ErrGraphemeAllocOutOfMemory, err)

	// The source is untouched.
	expectEqual(t, uint(2), s.TotalPages())
	expect(t, s.pages.first == first, "s.pages.first.? == first")
	expect(t, s.pages.last == last, "s.pages.last.? == last")
	expectEqual(t, pins, s.CountTrackedPins())
	expectEqual(t, total_rows, s.total_rows)
	expectEqual(t, page_size, s.page_size)
	expectEqual(t, serial, s.page_serial)
	expectEqual(t, uint64(2048), page.grapheme_alloc.UsedBytes())
	expectEqual(t, uint(4), page.GraphemeCount())
	expectEqual(t, 160, len(zigUnwrap(t, page.LookupGrapheme(page.GetRowAndCell(2, 0).cell))))
	expectEqual(t, 96, len(zigUnwrap(t, page.LookupGrapheme(page.GetRowAndCell(1, 0).cell))))
	page.AssertIntegrity()

	// The same failure with a pin remap in the options: the remap is not
	// filled in for the failed clone and the tracked pins of the source stay.
	{
		remap := PageListCloneTrackedPinsRemap{}
		_, err = s.Clone(PageListClone{Top: Point{Tag: PointTagScreen}, TrackedPins: &remap})
		rbExpectError(t, ErrGraphemeAllocOutOfMemory, err)
		expectEqual(t, pins, s.CountTrackedPins())
	}

	// Control: without the last grapheme the other three fit.
	page.ClearCells(page.GetRow(0), 3, 4)
	expectEqual(t, uint(3), page.GraphemeCount())
	cp, err := s.Clone(PageListClone{Top: Point{Tag: PointTagScreen}})
	zigTry(t, err)
	defer cp.Deinit()
	expectEqual(t, uint(2), cp.TotalPages())
	expectEqual(t, uint(3), cp.pages.last.data.GraphemeCount())
}

// Zig: test "term51: Terminal.switchScreenMode: a cursor copy that runs out of space is ignored by a full style set"
func TestRollback_Terminal_switchScreenMode_a_cursor_copy_that_runs_out_of_space_is_ignored_by_a_full_style_set(t *testing.T) {
	testStdCapacity(t)
	// One row: the alternate page cannot be split to make room.
	term, err := TerminalInit(terminalOptionsDefault(10, 1))
	zigTry(t, err)
	defer term.Deinit()

	// Create the alternate screen and fill the style set of its page.
	_, err = term.SwitchScreen(ScreenSetKeyAlternate)
	zigTry(t, err)
	_, err = term.SwitchScreen(ScreenSetKeyPrimary)
	zigTry(t, err)
	alt := term.Screens.Get(ScreenSetKeyAlternate)
	expect(t, alt != nil, "t.screens.get(.alternate).? != null")
	const max_styles = 1<<16 - 1
	for alt.Cursor.PagePin.Node.data.capacity.styles < max_styles {
		_, err := alt.IncreaseCapacity(alt.Cursor.PagePin.Node, Some(PageListIncreaseCapacityStyles))
		if err != nil {
			break
		}
	}
	alt_page := &alt.Cursor.PagePin.Node.data
	expectEqual(t, SizeStyleCountInt(max_styles), alt_page.capacity.styles)
	rbFillStyles(alt_page)
	count := alt_page.styles.Count()

	// The primary cursor has a style and a position.
	zigTry(t, term.SetAttribute(rbAttr(SGRAttributeTagBold)))
	term.SetCursorPos(1, 4)
	primary := term.Screens.Active
	expectEqual(t, SizeCellCountInt(3), primary.Cursor.X)

	// Entering the alternate screen still works: the cursor copy fails with
	// OutOfSpace, which is only logged.
	zigTry(t, term.SwitchScreenMode(TerminalSwitchScreenMode47, true))
	expectEqual(t, ScreenSetKeyAlternate, term.Screens.ActiveKey)
	expect(t, term.Screens.Active == alt, "t.screens.active == alt")

	// The alternate cursor is what it was before the failed copy: neither the
	// position nor the style were taken over.
	expectEqual(t, SizeCellCountInt(0), alt.Cursor.X)
	expectEqual(t, SizeCellCountInt(0), alt.Cursor.Y)
	expectEqual(t, StyleDefaultID, alt.Cursor.StyleID)
	expect(t, alt.Cursor.Style.Default(), "alt.cursor.style.default()")
	expect(t, alt.Cursor.Hyperlink == nil, "alt.cursor.hyperlink == null")
	expectEqual(t, count, alt_page.styles.Count())
	alt.AssertIntegrity()
	alt_page.AssertIntegrity()

	// The primary cursor was not changed.
	expectEqual(t, SizeCellCountInt(3), primary.Cursor.X)
	expect(t, primary.Cursor.Style.Flags.Bold, "primary.cursor.style.flags.bold")
	expect(t, primary.Cursor.StyleID != StyleDefaultID, "primary.cursor.style_id != style.default_id")
	primary.AssertIntegrity()
}
