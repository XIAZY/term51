package terminal

// Rollback specifications for four branches that are reachable in Go through
// a CAPACITY error (a fragmented grapheme store, a full style set, a full and
// maxed hyperlink set) but cannot be tested on upstream: upstream logs the
// failure with `log.err`, and the Zig test runner fails a test that logs an
// error. They are written before the implementation, against the wireframes.
// The expectations were read from the Zig source of v1.3.1
// (src/terminal/PageList.zig, Screen.zig) and each scenario was also run
// against a scratch copy of that source, where every assertion held and the
// only failure was the logged error itself.
//
// Shared with rollback_test.go (not edited): rbInit, rbExpectError, rbAttr,
// rbWrite, rbStartLink, rbMaxLinkStores, rbFillLinks, rbFillStyles,
// rbRepeatCps and rbGraphemeCell. Helpers of this file start with `rs`.

import "testing"

// rsFragment fills the first four cells of row y of the page with graphemes
// in this order: x=2 (40 chunks), x=0 (24 chunks), x=3 (40) and x=1 (24).
// In a store of two 64-chunk bitmaps (or more) they fit: 40+24 in the first
// bitmap, 40+24 in the second. A fresh store of two bitmaps filled in cell
// order (24, 24, 40, 40) fails on the last one: 40+24 and 16 chunks left in
// the first bitmap, 24 in the second, although 128 chunks are enough in total.
func rsFragment(t *testing.T, page *Page, y uint) {
	t.Helper()
	long := rbRepeatCps(0x301, 160)
	short := rbRepeatCps(0x302, 96)
	time_order := []rbGraphemeCell{
		{x: 2, cps: long},
		{x: 0, cps: short},
		{x: 3, cps: long},
		{x: 1, cps: short},
	}
	for _, e := range time_order {
		rac := page.GetRowAndCell(e.x, y)
		*rac.cell = PageCellInit('a')
		zigTry(t, page.SetGraphemes(rac.row, rac.cell, e.cps))
	}
	expectEqual(t, uint64(2048), page.grapheme_alloc.UsedBytes())
	expectEqual(t, uint(4), page.GraphemeCount())
	page.AssertIntegrity()
}

// rsTrack is `try s.trackPin(.{ .node = node, .y = y, .x = x })`.
func rsTrack(t *testing.T, s *PageList, node *PageListNode, y SizeCellCountInt, x SizeCellCountInt) *PageListPin {
	t.Helper()
	p, err := s.TrackPin(PageListPin{Node: node, Y: y, X: x})
	zigTry(t, err)
	return p
}

// rsExpectPin checks the node, row and column of a pin.
func rsExpectPin(t *testing.T, p *PageListPin, node *PageListNode, y SizeCellCountInt, x SizeCellCountInt) {
	t.Helper()
	expect(t, p.Node == node, "p.node == node")
	expectEqual(t, y, p.Y)
	expectEqual(t, x, p.X)
}

// rsMaxStylesNode grows the style capacity of the given page (which need not
// be the cursor page) to the maximum and returns its node.
func rsMaxStylesNode(s *Screen, node *PageListNode) *PageListNode {
	const max = 1<<16 - 1
	for node.data.capacity.styles < max {
		n, err := s.IncreaseCapacity(node, Some(PageListIncreaseCapacityStyles))
		if err != nil {
			break
		}
		node = n
	}
	return node
}

// rsChangeScene splits a 3 row screen into three one-row pages (the cursor
// stays in the first) and prepares the SECOND page, the one the cursor moves
// to with one step down. A one-row page cannot be split, so a failing style
// update on it has no way out. With fill_styles the page's style set is at
// the maximum capacity and full; with fill_links its string store and
// hyperlink set are at the maximum capacity and the set is full of live
// links. The capacities are grown first and the sets filled last, because a
// capacity increase does not copy references that no cell holds.
func rsChangeScene(t *testing.T, s *Screen, fill_styles bool, fill_links bool) *PageListNode {
	t.Helper()
	for y := uint32(1); y <= 2; y++ {
		pin := zigUnwrap(t, s.Pages.Pin(Point{Tag: PointTagActive, Active: PointCoordinate{X: 0, Y: y}}))
		zigTry(t, s.Pages.Split(pin))
	}
	expectEqual(t, uint(3), s.Pages.TotalPages())
	node := s.Pages.pages.first.next
	expect(t, s.Cursor.PagePin.Node == s.Pages.pages.first, "s.cursor.page_pin.node == s.pages.pages.first")

	if fill_styles {
		node = rsMaxStylesNode(s, node)
		expectEqual(t, SizeStyleCountInt(1<<16-1), node.data.capacity.styles)
	}
	if fill_links {
		for i := 0; i < 6; i++ {
			n, err := s.IncreaseCapacity(node, Some(PageListIncreaseCapacityStringBytes))
			zigTry(t, err)
			node = n
		}
		steps := 0
		for {
			n, err := s.IncreaseCapacity(node, Some(PageListIncreaseCapacityHyperlinkBytes))
			if err != nil {
				rbExpectError(t, ErrOutOfSpace, err)
				break
			}
			node = n
			steps++
			expect(t, steps < 64, "steps < 64")
		}
		expectEqual(t, SizeHyperlinkCountInt(1<<16-1), node.data.capacity.hyperlink_bytes)
		filled := rbFillLinks(t, &node.data)
		expect(t, filled.count > 100, "filled.count > 100")
	}
	if fill_styles {
		rbFillStyles(&node.data)
	}
	// Moving the cursor one row down must go to this page and no other.
	expect(t, s.Pages.pages.first.next == node, "s.pages.pages.first.next == node")
	expect(t, s.Pages.pages.last == node.next, "s.pages.pages.last == node.next")
	return node
}

// ---------------------------------------------------------------------------
// PageList.compact
// ---------------------------------------------------------------------------

// Go-only: rollback specification (not testable on upstream: PageList.compact logs an error at PageList.zig:2806). Follows PageList.zig:2774-2828 (the failing clone at 2797-2812).
func TestRollbackSpec_PageList_compact_clone_failure_pretends_no_compaction_was_needed(t *testing.T) {
	testStdCapacity(t)
	// 3353 columns, 13 rows: one page. Three doublings of the grapheme bytes
	// make it a non-standard page (4096 grapheme bytes, memory longer than
	// std_size), so compact does not return early.
	l, err := PageListInit(3353, 13, None[uint64]())
	zigTry(t, err)
	s := &l
	defer s.Deinit()
	expectEqual(t, uint(1), s.TotalPages())
	node := s.pages.first
	for i := 0; i < 3; i++ {
		node, err = s.IncreaseCapacity(node, Some(PageListIncreaseCapacityGraphemeBytes))
		zigTry(t, err)
	}
	page := &node.data
	expectEqual(t, uint32(4096), page.capacity.grapheme_bytes)
	expectEqual(t, SizeCellCountInt(13), page.size.rows)

	// The four graphemes fill 2048 bytes of the 4096 (one 64-chunk bitmap
	// has 40+24 chunks, the next one 40+24).
	rsFragment(t, page, 0)

	// The page is non-standard and the exact capacity is smaller, so compact
	// goes on to clone. The exact capacity has grapheme_bytes of 2048 (128
	// chunks, two bitmaps) and a standard size layout: the new page is a
	// pooled one. Cloning in cell order (24, 24, 40, 40) into it cannot place
	// the last grapheme although the capacity is "exact".
	req := page.ExactRowCapacity(0, uint(page.size.rows))
	expectEqual(t, uint32(2048), req.grapheme_bytes)
	req_size := PageLayoutOf(req).total_size
	old_len := pageListMemoryLen(page)
	expect(t, old_len > pageListStdSize(), "page.memory.len > std_size")
	expect(t, req_size < old_len, "new_size < old_size")
	expect(t, req_size <= pageListStdSize(), "new page is pooled")

	tracked := rsTrack(t, s, node, 5, 7)
	pins := s.CountTrackedPins()
	total_rows := s.total_rows
	page_size := s.page_size
	serial := s.page_serial
	cap := page.capacity

	// compact: createPage(req_cap) succeeds, cloneFrom fails with
	// GraphemeAllocOutOfMemory, the error is logged, the new node is
	// destroyed and the result is null with no error ("gracefully degrade by
	// pretending we didn't need to compact").
	res, err := s.Compact(node)
	zigTry(t, err)
	expect(t, res == nil, "(try s.compact(node)) == null")

	// The list is as it was: same node, same page, same pins.
	expectEqual(t, uint(1), s.TotalPages())
	expect(t, s.pages.first == node, "s.pages.first.? == node")
	expect(t, s.pages.last == node, "s.pages.last.? == node")
	expect(t, &s.pages.first.data == page, "&s.pages.first.?.data == page")
	rsExpectPin(t, tracked, node, 5, 7)
	expectEqual(t, pins, s.CountTrackedPins())
	expectEqual(t, total_rows, s.total_rows)
	expectEqual(t, cap, page.capacity)
	expectEqual(t, old_len, pageListMemoryLen(page))
	// createPage consumed a serial and compact does not give it back.
	expectEqual(t, serial+1, s.page_serial)
	// Upstream defect pinned: createPage adds the size of the pooled buffer
	// (std_size) to page_size and destroyNode takes the page's memory.len
	// (the layout size, smaller than std_size) off again, so the failed
	// compact leaves page_size larger by the difference.
	expectEqual(t, page_size+(pageListStdSize()-req_size), s.page_size)
	// The page content is untouched.
	expectEqual(t, uint(4), page.GraphemeCount())
	expectEqual(t, uint64(2048), page.grapheme_alloc.UsedBytes())
	expectEqual(t, 160, len(zigUnwrap(t, page.LookupGrapheme(page.GetRowAndCell(2, 0).cell))))
	expectEqual(t, 160, len(zigUnwrap(t, page.LookupGrapheme(page.GetRowAndCell(3, 0).cell))))
	expectEqual(t, 96, len(zigUnwrap(t, page.LookupGrapheme(page.GetRowAndCell(0, 0).cell))))
	expectEqual(t, 96, len(zigUnwrap(t, page.LookupGrapheme(page.GetRowAndCell(1, 0).cell))))
	s.AssertIntegrity()
	page.AssertIntegrity()

	// Control: without the last grapheme the exact capacity (1408 bytes)
	// clones and the page is replaced by the compact one, the pin moved.
	page.ClearCells(page.GetRow(0), 3, 4)
	expectEqual(t, uint(3), page.GraphemeCount())
	req = page.ExactRowCapacity(0, uint(page.size.rows))
	expectEqual(t, uint32(1408), req.grapheme_bytes)
	res, err = s.Compact(node)
	zigTry(t, err)
	expect(t, res != nil, "(try s.compact(node)) != null")
	expect(t, res != node, "res != node")
	expectEqual(t, uint(1), s.TotalPages())
	expect(t, s.pages.first == res, "s.pages.first.? == res")
	expect(t, s.pages.last == res, "s.pages.last.? == res")
	expectEqual(t, req, res.data.capacity)
	expectEqual(t, uint(3), res.data.GraphemeCount())
	rsExpectPin(t, tracked, res, 5, 7)
	expectEqual(t, pins, s.CountTrackedPins())
	expectEqual(t, total_rows, s.total_rows)
	s.AssertIntegrity()
	res.data.AssertIntegrity()
}

// ---------------------------------------------------------------------------
// PageList.split
// ---------------------------------------------------------------------------

// Go-only: rollback specification (not testable on upstream: PageList.split logs an error at PageList.zig:2883). Follows PageList.zig:2844-2919 (the failing clone at 2882-2892).
func TestRollbackSpec_PageList_split_clone_failure_returns_OutOfSpace_and_leaves_the_page_unsplit(t *testing.T) {
	testStdCapacity(t)
	// 3353 columns, 20 rows: two pages, the last with 7 rows. Two doublings
	// of its grapheme bytes keep it a standard size (pooled) page.
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
	expectEqual(t, SizeCellCountInt(7), page.size.rows)
	expectEqual(t, uint32(2048), page.capacity.grapheme_bytes)
	expectEqual(t, PageLayoutOf(PageStdCapacity).total_size, pageListMemoryLen(page))

	// The fragmented graphemes are on row 1. Splitting at row 1 copies rows 1
	// to 6 into a new page of the SAME capacity; the clone allocates the
	// graphemes in cell order into a fresh store and fails on the last one.
	rsFragment(t, page, 1)
	page.GetRow(1).dirty = false

	// Tracked pins: on the row that would move, below it, above it.
	p6 := rsTrack(t, s, last, 6, 9)
	p3 := rsTrack(t, s, last, 3, 7)
	p1 := rsTrack(t, s, last, 1, 4)
	p0 := rsTrack(t, s, last, 0, 2)
	pins := s.CountTrackedPins()
	total_rows := s.total_rows
	page_size := s.page_size
	serial := s.page_serial
	first := s.pages.first
	cap := page.capacity

	// split: createPage succeeds, cloneFrom fails with
	// GraphemeAllocOutOfMemory, the error is logged and OutOfSpace is
	// returned ("we couldn't split and let our callers gracefully handle
	// it"). The errdefer destroys the new node.
	rbExpectError(t, ErrOutOfSpace, s.Split(PageListPin{Node: last, Y: 1}))

	// Nothing was split: no new page, no rows moved, no pin moved.
	expectEqual(t, uint(2), s.TotalPages())
	expect(t, s.pages.first == first, "s.pages.first.? == first")
	expect(t, s.pages.last == last, "s.pages.last.? == last")
	expect(t, last.next == nil, "last.next == null")
	expectEqual(t, SizeCellCountInt(7), page.size.rows)
	expectEqual(t, total_rows, s.total_rows)
	rsExpectPin(t, p6, last, 6, 9)
	rsExpectPin(t, p3, last, 3, 7)
	rsExpectPin(t, p1, last, 1, 4)
	rsExpectPin(t, p0, last, 0, 2)
	expectEqual(t, pins, s.CountTrackedPins())
	// The new page was created (a serial was used and stays used) and given
	// back (the size accounting is balanced because the page is a standard
	// size page).
	expectEqual(t, serial+1, s.page_serial)
	expectEqual(t, page_size, s.page_size)
	// The source is untouched: the rows were not cleared, not dirtied.
	expectEqual(t, cap, page.capacity)
	expectEqual(t, uint(4), page.GraphemeCount())
	expectEqual(t, uint64(2048), page.grapheme_alloc.UsedBytes())
	expectEqual(t, 160, len(zigUnwrap(t, page.LookupGrapheme(page.GetRowAndCell(2, 1).cell))))
	expectEqual(t, 160, len(zigUnwrap(t, page.LookupGrapheme(page.GetRowAndCell(3, 1).cell))))
	expectEqual(t, 96, len(zigUnwrap(t, page.LookupGrapheme(page.GetRowAndCell(0, 1).cell))))
	expectEqual(t, 96, len(zigUnwrap(t, page.LookupGrapheme(page.GetRowAndCell(1, 1).cell))))
	expect(t, !page.GetRow(1).dirty, "!page.getRow(1).dirty")
	s.AssertIntegrity()
	page.AssertIntegrity()

	// Control: without the last grapheme the split works; the rows from 1 on
	// move to a new page after the old one and the pins follow.
	page.ClearCells(page.GetRow(1), 3, 4)
	expectEqual(t, uint(3), page.GraphemeCount())
	zigTry(t, s.Split(PageListPin{Node: last, Y: 1}))
	expectEqual(t, uint(3), s.TotalPages())
	target := last.next
	expect(t, target != nil, "last.next != null")
	expect(t, s.pages.last == target, "s.pages.last.? == last.next.?")
	expectEqual(t, SizeCellCountInt(1), last.data.size.rows)
	expectEqual(t, SizeCellCountInt(6), target.data.size.rows)
	expectEqual(t, uint(0), last.data.GraphemeCount())
	expectEqual(t, uint(3), target.data.GraphemeCount())
	rsExpectPin(t, p6, target, 5, 9)
	rsExpectPin(t, p3, target, 2, 7)
	rsExpectPin(t, p1, target, 0, 4)
	rsExpectPin(t, p0, last, 0, 2)
	expectEqual(t, total_rows, s.total_rows)
	s.AssertIntegrity()
}

// ---------------------------------------------------------------------------
// Screen.cursorCopy
// ---------------------------------------------------------------------------

// rsCopyState is what rsCopyScene records about the page of the target screen.
type rsCopyState struct {
	node    *PageListNode
	old_id  HyperlinkID
	count   uint
	strings uint64
	cap     PageCapacity
}

// rsCopyScene prepares a screen s whose cursor page has maxed hyperlink
// stores, an implicit link in use by one cell (so that ending it frees no
// slot) and a full hyperlink set, and a second screen `other` whose cursor is
// at x=4, y=1 with the cursor shape underline, a pending wrap and a
// hyperlink: explicit ("oid") or implicit (two links were started, so its
// implicit counter is 2).
func rsCopyScene(t *testing.T, s *Screen, other *Screen, implicit bool) rsCopyState {
	t.Helper()
	rbMaxLinkStores(t, s)
	rbStartLink(t, s, "https://old/", None[[]uint8]())
	rbWrite(t, s, "x")
	node := s.Cursor.PagePin.Node
	page := &node.data
	old_id := s.Cursor.HyperlinkID
	expectEqual(t, SizeOffsetInt(1), s.Cursor.HyperlinkImplicitID)
	expectEqual(t, 2, page.hyperlink_set.RefCount(old_id))

	if implicit {
		rbStartLink(t, other, "https://skip/", None[[]uint8]())
		rbStartLink(t, other, "https://other/", None[[]uint8]())
		expectEqual(t, SizeOffsetInt(2), other.Cursor.HyperlinkImplicitID)
	} else {
		rbStartLink(t, other, "https://other/", Some([]uint8("oid")))
		expectEqual(t, SizeOffsetInt(0), other.Cursor.HyperlinkImplicitID)
	}
	other.CursorAbsolute(4, 1)
	other.Cursor.PendingWrap = true
	other.Cursor.CursorStyle = ScreenCursorStyleUnderline

	filled := rbFillLinks(t, page)
	expect(t, filled.count > 100, "filled.count > 100")
	return rsCopyState{
		node:    node,
		old_id:  old_id,
		count:   page.hyperlink_set.Count(),
		strings: page.string_alloc.UsedBytes(),
		cap:     page.capacity,
	}
}

// rsExpectCopyFailedLink checks what stays after a cursorCopy whose
// startHyperlink failed: the position and the plain fields were taken over,
// no link is active, and the page lost nothing but the old link's slot
// reference (the old link was ended first, its cell keeps it).
func rsExpectCopyFailedLink(t *testing.T, s *Screen, other *Screen, st rsCopyState) {
	t.Helper()
	page := &st.node.data
	expect(t, s.Cursor.PagePin.Node == st.node, "s.cursor.page_pin.node == node")
	expectEqual(t, SizeCellCountInt(4), s.Cursor.X)
	expectEqual(t, SizeCellCountInt(1), s.Cursor.Y)
	expectEqual(t, SizeCellCountInt(4), s.Cursor.PagePin.X)
	expectEqual(t, SizeCellCountInt(1), s.Cursor.PagePin.Y)
	expect(t, s.Cursor.PageRow == page.GetRow(1), "s.cursor.page_row == page.getRow(1)")
	expect(t, s.Cursor.PageCell == &page.GetCells(page.GetRow(1))[4], "s.cursor.page_cell == &page.getCells(page.getRow(1))[4]")
	expect(t, s.Cursor.PendingWrap, "s.cursor.pending_wrap")
	expectEqual(t, ScreenCursorStyleUnderline, s.Cursor.CursorStyle)

	// The hyperlink was not copied, and no half state is left.
	expectEqual(t, HyperlinkID(0), s.Cursor.HyperlinkID)
	expect(t, s.Cursor.Hyperlink == nil, "s.cursor.hyperlink == null")
	expectEqual(t, st.cap, page.capacity)
	expectEqual(t, st.count, page.hyperlink_set.Count())
	expectEqual(t, st.strings, page.string_alloc.UsedBytes())
	// The previous link was ended: only its cell keeps it.
	expectEqual(t, 1, page.hyperlink_set.RefCount(st.old_id))
	expect(t, page.LookupHyperlink(&page.GetCells(page.GetRow(0))[0]).Valid, "page.lookupHyperlink(&page.getCells(page.getRow(0))[0]) != null")

	// The other cursor and its page were not touched.
	expect(t, other.Cursor.Hyperlink != nil, "other.cursor.hyperlink != null")
	expectEqual(t, SizeCellCountInt(4), other.Cursor.X)
	s.AssertIntegrity()
	page.AssertIntegrity()
}

// Go-only: rollback specification (not testable on upstream: Screen.cursorCopy logs an error at Screen.zig:1113). Follows Screen.zig:1059-1116 (the swallowed startHyperlink failure at 1109-1114).
func TestRollbackSpec_Screen_cursorCopy_explicit_hyperlink_failure_is_swallowed_and_the_cursor_is_copied_without_link(t *testing.T) {
	testStdCapacity(t)
	s := rbInit(t, 10, 3)
	defer s.Deinit()
	other := rbInit(t, 10, 3)
	defer other.Deinit()
	st := rsCopyScene(t, s, other, false)

	// cursorCopy: the old link is ended, the other cursor is taken over, the
	// hyperlink of the other cursor (an explicit link "oid") cannot be added
	// (the set is full and cannot grow: startHyperlink returns OutOfSpace),
	// the error is logged and swallowed: cursorCopy returns no error.
	zigTry(t, s.CursorCopy(other.Cursor, screenCursorCopyOptionsDefault()))

	rsExpectCopyFailedLink(t, s, other, st)
	// The default style of the other cursor was taken over.
	expectEqual(t, StyleDefaultID, s.Cursor.StyleID)
	expect(t, s.Cursor.Style.Default(), "s.cursor.style.default()")
	// The cursor struct was copied wholesale, including the implicit id
	// counter: it is the other cursor's (0), not this cursor's old one (1).
	// An explicit link has no counter to restore.
	expectEqual(t, SizeOffsetInt(0), s.Cursor.HyperlinkImplicitID)
}

// Go-only: rollback specification (not testable on upstream: Screen.cursorCopy logs an error at Screen.zig:1113). Follows Screen.zig:1059-1116 and startHyperlink at Screen.zig:2189-2238.
func TestRollbackSpec_Screen_cursorCopy_implicit_hyperlink_failure_restores_the_copied_implicit_counter(t *testing.T) {
	testStdCapacity(t)
	s := rbInit(t, 10, 3)
	defer s.Deinit()
	other := rbInit(t, 10, 3)
	defer other.Deinit()
	st := rsCopyScene(t, s, other, true)

	// The other cursor's link is implicit, so startHyperlink gets a NEW
	// implicit id from this cursor's counter, which `self.cursor = other`
	// has set to the other cursor's counter (2): it takes id 2, the counter
	// becomes 3, the insert fails and the errdefer takes it back to 2.
	zigTry(t, s.CursorCopy(other.Cursor, screenCursorCopyOptionsDefault()))

	rsExpectCopyFailedLink(t, s, other, st)
	expectEqual(t, StyleDefaultID, s.Cursor.StyleID)
	expectEqual(t, SizeOffsetInt(2), s.Cursor.HyperlinkImplicitID)
}

// Go-only: rollback specification (not testable on upstream: Screen.cursorCopy logs an error at Screen.zig:1113). Follows Screen.zig:1059-1116: manualStyleUpdate at 1091 has succeeded when the hyperlink fails at 1109.
func TestRollbackSpec_Screen_cursorCopy_hyperlink_failure_keeps_the_copied_style(t *testing.T) {
	testStdCapacity(t)
	s := rbInit(t, 10, 3)
	defer s.Deinit()
	other := rbInit(t, 10, 3)
	defer other.Deinit()
	zigTry(t, other.SetAttribute(rbAttr(SGRAttributeTagBold)))
	st := rsCopyScene(t, s, other, false)
	page := &st.node.data
	expectEqual(t, uint(0), page.styles.Count())

	// The style update comes first and succeeds: it is not rolled back when
	// the hyperlink then fails (the errdefer of cursorCopy only covers the
	// style update itself).
	zigTry(t, s.CursorCopy(other.Cursor, screenCursorCopyOptionsDefault()))

	rsExpectCopyFailedLink(t, s, other, st)
	expect(t, s.Cursor.Style.Flags.Bold, "s.cursor.style.flags.bold")
	expect(t, s.Cursor.StyleID != StyleDefaultID, "s.cursor.style_id != style.default_id")
	expectEqual(t, 1, page.styles.RefCount(s.Cursor.StyleID))
	expectEqual(t, uint(1), page.styles.Count())
	expectEqual(t, SizeOffsetInt(0), s.Cursor.HyperlinkImplicitID)
	// The other cursor keeps its own style on its own page.
	opage := &other.Cursor.PagePin.Node.data
	expectEqual(t, 1, opage.styles.RefCount(other.Cursor.StyleID))
}

// ---------------------------------------------------------------------------
// Screen.cursorChangePin
// ---------------------------------------------------------------------------

// rsChangeState is the state before the cursor moves onto the second page.
type rsChangeState struct {
	node0     *PageListNode
	node1     *PageListNode
	old_style StyleID
	old_link  HyperlinkID
	old_row   *PageRow
	new_row   *PageRow
	cap1      PageCapacity
	serial    uint64
	page_size uint64
}

// rsChangeStart gives the cursor (on the first page) a bold style and an
// implicit or explicit hyperlink, clears the dirty flags of the row it leaves
// and the row it enters, and returns the state before the move.
func rsChangeStart(t *testing.T, s *Screen, node1 *PageListNode, uri string, id Optional[[]uint8]) rsChangeState {
	t.Helper()
	node0 := s.Pages.pages.first
	expect(t, s.Cursor.PagePin.Node == node0, "s.cursor.page_pin.node == node0")
	zigTry(t, s.SetAttribute(rbAttr(SGRAttributeTagBold)))
	rbStartLink(t, s, uri, id)
	page0 := &node0.data
	st := rsChangeState{
		node0:     node0,
		node1:     node1,
		old_style: s.Cursor.StyleID,
		old_link:  s.Cursor.HyperlinkID,
		old_row:   s.Cursor.PageRow,
		new_row:   zigUnwrap(t, s.Cursor.PagePin.Down(1)).RowAndCell().row,
		cap1:      node1.data.capacity,
		serial:    s.Pages.page_serial,
		page_size: s.Pages.page_size,
	}
	expect(t, st.old_style != StyleDefaultID, "style_id != style.default_id")
	expectEqual(t, 1, page0.styles.RefCount(st.old_style))
	expectEqual(t, 1, page0.hyperlink_set.RefCount(st.old_link))
	st.old_row.dirty = false
	st.new_row.dirty = false
	node1.data.dirty = false
	return st
}

// rsExpectMoved checks what a cursor move onto the second page leaves
// whatever happened to the style and the link: the cursor is on the new row
// of the new page with all pointers refreshed, the old style and the old link
// reference were released on the old page, both rows were marked dirty (not
// the whole new page), and no page was split, grown or created.
func rsExpectMoved(t *testing.T, s *Screen, st rsChangeState) {
	t.Helper()
	page0 := &st.node0.data
	page1 := &st.node1.data
	expect(t, s.Cursor.PagePin.Node == st.node1, "s.cursor.page_pin.node == node1")
	expectEqual(t, SizeCellCountInt(0), s.Cursor.X)
	expectEqual(t, SizeCellCountInt(1), s.Cursor.Y)
	expectEqual(t, SizeCellCountInt(0), s.Cursor.PagePin.Y)
	expect(t, s.Cursor.PageRow == st.new_row, "s.cursor.page_row == new_row")
	expect(t, s.Cursor.PageCell == &page1.GetCells(st.new_row)[0], "s.cursor.page_cell == &page1.getCells(new_row)[0]")

	// The cursor's references on the old page are gone (the slots stay, as
	// refcounted sets keep a zero-reference item around).
	expectEqual(t, 0, page0.styles.RefCount(st.old_style))
	expectEqual(t, 0, page0.hyperlink_set.RefCount(st.old_link))

	expect(t, st.old_row.dirty, "old_row.dirty")
	expect(t, st.new_row.dirty, "new_row.dirty")
	expect(t, !page1.dirty, "!page1.dirty")

	// Nothing was split, grown or created: the three pages are as they
	// were, the failed attempts did not use a serial or any page memory.
	expectEqual(t, uint(3), s.Pages.TotalPages())
	expect(t, s.Pages.pages.first == st.node0, "s.pages.pages.first.? == node0")
	expect(t, s.Pages.pages.first.next == st.node1, "s.pages.pages.first.?.next.? == node1")
	expectEqual(t, st.cap1, page1.capacity)
	expectEqual(t, st.serial, s.Pages.page_serial)
	expectEqual(t, st.page_size, s.Pages.page_size)
}

// Go-only: rollback specification (not testable on upstream: Screen.cursorChangePin logs an error at Screen.zig:1177). Follows Screen.zig:1129-1204 (style at 1170-1181, hyperlink at 1184-1203) and manualStyleUpdate at Screen.zig:1995-2074.
func TestRollbackSpec_Screen_cursorChangePin_style_that_does_not_fit_degrades_to_the_default_style(t *testing.T) {
	testStdCapacity(t)
	s := rbInit(t, 10, 3)
	defer s.Deinit()
	node1 := rsChangeScene(t, s, true, false)
	st := rsChangeStart(t, s, node1, "https://example.com/", None[[]uint8]())
	page0 := &st.node0.data
	page1 := &node1.data
	expectEqual(t, SizeOffsetInt(1), s.Cursor.HyperlinkImplicitID)
	stcount := page1.styles.Count()
	lcount := page1.hyperlink_set.Count()

	// The cursor moves onto the second page. Its style is released on the
	// old page, then re-added with manualStyleUpdate on the new page: the
	// set is full, increaseCapacity says OutOfSpace (maxed), splitForCapacity
	// says OutOfSpace (one row). The error is logged and the cursor degrades
	// to the default style: style = .{}, style_id = 0.
	s.CursorDown(1)
	rsExpectMoved(t, s, st)

	expectEqual(t, StyleDefaultID, s.Cursor.StyleID)
	expect(t, s.Cursor.Style.Default(), "s.cursor.style.default()")
	// Nothing was added to the new page's style set.
	expectEqual(t, stcount, page1.styles.Count())

	// The hyperlink is migrated on its own: a NEW implicit id is taken from
	// the counter (1, the counter becomes 2) and the link is added to the new
	// page with one reference; the old reference was released.
	expect(t, s.Cursor.Hyperlink != nil, "s.cursor.hyperlink != null")
	expect(t, s.Cursor.HyperlinkID != 0, "s.cursor.hyperlink_id != 0")
	expectEqualStrings(t, "https://example.com/", string(s.Cursor.Hyperlink.URI))
	expect(t, s.Cursor.Hyperlink.ID.Tag == HyperlinkHyperlinkIdImplicit, "s.cursor.hyperlink.?.id == .implicit")
	expectEqual(t, SizeOffsetInt(1), s.Cursor.Hyperlink.ID.Implicit)
	expectEqual(t, SizeOffsetInt(2), s.Cursor.HyperlinkImplicitID)
	expectEqual(t, 1, page1.hyperlink_set.RefCount(s.Cursor.HyperlinkID))
	expectEqual(t, lcount+1, page1.hyperlink_set.Count())
	expectEqual(t, 0, page0.hyperlink_set.RefCount(st.old_link))
	s.AssertIntegrity()
	page0.AssertIntegrity()
	page1.AssertIntegrity()
}

// Go-only: rollback specification (not testable on upstream: Screen.cursorChangePin logs an error at Screen.zig:1197). Follows Screen.zig:1129-1204 (hyperlink at 1184-1203) and startHyperlink at Screen.zig:2189-2238.
func TestRollbackSpec_Screen_cursorChangePin_hyperlink_that_does_not_fit_degrades_to_no_link(t *testing.T) {
	testStdCapacity(t)
	s := rbInit(t, 10, 3)
	defer s.Deinit()
	node1 := rsChangeScene(t, s, false, true)
	st := rsChangeStart(t, s, node1, "https://example.com/", None[[]uint8]())
	page0 := &st.node0.data
	page1 := &node1.data
	expectEqual(t, SizeOffsetInt(1), s.Cursor.HyperlinkImplicitID)
	lcount := page1.hyperlink_set.Count()
	strings := page1.string_alloc.UsedBytes()

	// The style is re-added on the new page without trouble (its set has
	// room). The hyperlink then fails: the set is full and cannot grow, so
	// startHyperlink returns OutOfSpace (and takes its implicit id back), the
	// error is logged and the cursor has no hyperlink: id 0, no heap link.
	s.CursorDown(1)
	rsExpectMoved(t, s, st)

	expect(t, s.Cursor.Style.Flags.Bold, "s.cursor.style.flags.bold")
	expect(t, s.Cursor.StyleID != StyleDefaultID, "s.cursor.style_id != style.default_id")
	expectEqual(t, 1, page1.styles.RefCount(s.Cursor.StyleID))

	expectEqual(t, HyperlinkID(0), s.Cursor.HyperlinkID)
	expect(t, s.Cursor.Hyperlink == nil, "s.cursor.hyperlink == null")
	// The counter is where it was before the failed attempt.
	expectEqual(t, SizeOffsetInt(1), s.Cursor.HyperlinkImplicitID)
	// The failed insert left the new page's set and strings as they were.
	expectEqual(t, lcount, page1.hyperlink_set.Count())
	expectEqual(t, strings, page1.string_alloc.UsedBytes())
	expectEqual(t, 0, page0.hyperlink_set.RefCount(st.old_link))
	s.AssertIntegrity()
	page0.AssertIntegrity()
	page1.AssertIntegrity()
}

// Go-only: rollback specification (not testable on upstream: Screen.cursorChangePin logs an error at Screen.zig:1177 and Screen.zig:1197). Follows Screen.zig:1129-1204: the style degrades at 1172-1180 and then the hyperlink at 1190-1198.
func TestRollbackSpec_Screen_cursorChangePin_style_and_hyperlink_that_do_not_fit_both_degrade(t *testing.T) {
	testStdCapacity(t)
	s := rbInit(t, 10, 3)
	defer s.Deinit()
	node1 := rsChangeScene(t, s, true, true)
	// An explicit link this time: it has no counter.
	st := rsChangeStart(t, s, node1, "https://example.com/", Some([]uint8("id")))
	page0 := &st.node0.data
	page1 := &node1.data
	expectEqual(t, SizeOffsetInt(0), s.Cursor.HyperlinkImplicitID)
	stcount := page1.styles.Count()
	lcount := page1.hyperlink_set.Count()
	strings := page1.string_alloc.UsedBytes()

	// Both fail, each on its own: the style degrades to the default style and
	// the link to no link. The cursor is on the new page with a plain state.
	s.CursorDown(1)
	rsExpectMoved(t, s, st)

	expectEqual(t, StyleDefaultID, s.Cursor.StyleID)
	expect(t, s.Cursor.Style.Default(), "s.cursor.style.default()")
	expectEqual(t, stcount, page1.styles.Count())

	expectEqual(t, HyperlinkID(0), s.Cursor.HyperlinkID)
	expect(t, s.Cursor.Hyperlink == nil, "s.cursor.hyperlink == null")
	expectEqual(t, SizeOffsetInt(0), s.Cursor.HyperlinkImplicitID)
	expectEqual(t, lcount, page1.hyperlink_set.Count())
	expectEqual(t, strings, page1.string_alloc.UsedBytes())
	expectEqual(t, 0, page0.styles.RefCount(st.old_style))
	expectEqual(t, 0, page0.hyperlink_set.RefCount(st.old_link))
	s.AssertIntegrity()
	page0.AssertIntegrity()
	page1.AssertIntegrity()
}
