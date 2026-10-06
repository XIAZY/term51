// Tests translated one for one from src/terminal/PageList.zig (Ghostty v1.3.1
// plus the term51 additional tests), part 4: the tests from "term51:
// ReflowCursor.reflowRow: background color cells are not trimmed" up to and
// including "term51: scrollbar: unscrolled screen fits exactly". Each Go test
// carries the exact Zig title in a "// Zig: test" comment. Helpers are
// prefixed pl4.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualSlices / expectEqualStrings keep Zig's names
//     and Zig's (expected, actual) argument order. `try` on a call that
//     returns an error is zigTry(t, err); `x.?` is zigUnwrap.
//     `try testing.expectError(error.X, call)` is pl4ExpectError(t, ErrX,
//     <the error of the call>).
//   - Zig-only, in every test: `const alloc = std.testing.allocator;` and
//     the allocator argument of `init` (allocators are dropped); `defer
//     s.deinit()` is `defer s.Deinit()`, the stub every test calls. They are
//     not repeated as `// Zig-only:` lines in each test. Every test starts
//     with testStdCapacity(t): Zig's unit tests run with a standard page
//     capacity of grapheme_bytes 512.
//   - `init(alloc, cols, rows, max_size)` is pl4Init(t, cols, rows, max):
//     pl4Max(n) is a max_size of n bytes, pl4NoMax() is `null`.
//   - Struct literals: PageList.Resize has reflow = true as its Zig default;
//     every literal writes reflow out. Optional fields left out of a Go
//     literal are `null`. A `Capacity` literal `.{ .cols = c, .rows = r }`
//     is PageCapacityInit(c, r) with the fields it sets assigned after.
//   - The Zig `Scroll` union literals are built with pl4ScrollRow,
//     pl4ScrollDeltaRow, pl4ScrollDeltaPrompt and pl4ScrollPin;
//     `.active` and `.top` are PageListScroll{Tag: ...}.
//   - Pointer comparisons of nodes are pointer comparisons; the byte length
//     of a page's memory (`page.memory.len`) is pageListMemoryLen(&page) and
//     `std_size` is pageListStdSize().
//   - `page.rows.ptr(page.memory)[y]` is page.rows[y], and
//     `row.cells.ptr(page.memory)[..]` is page.GetCells(row).
//   - Helpers of the Zig file used by these tests (term51K*, term51L*) are
//     translated here with the prefix pl4K and pl4L.
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt. Go-only panic tests follow the
//     translated tests.

package terminal

import (
	"errors"
	"math"
	"strings"
	"testing"
)

// pl4ExpectError is Zig's `try testing.expectError(expected, actual)`.
func pl4ExpectError(t *testing.T, want error, got error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

// pl4ExpectPanic runs f and fails the test if f does not panic.
func pl4ExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if recover() == nil {
			t.Fatalf("expected a panic, got none")
		}
	}()
	f()
}

// pl4Max is a `max_size` of n bytes, pl4NoMax is `null`.
func pl4Max(n uint64) Optional[uint64] { return Some(n) }
func pl4NoMax() Optional[uint64]       { return None[uint64]() }

// pl4Init is `try init(alloc, cols, rows, max_size)`.
func pl4Init(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt, max_size Optional[uint64]) *PageList {
	t.Helper()
	s, err := PageListInit(cols, rows, max_size)
	zigTry(t, err)
	return &s
}

// pl4S is a non-null `?size.CellCountInt`.
func pl4S(n SizeCellCountInt) Optional[SizeCellCountInt] { return Some(n) }

// pl4Cursor is `.{ .x = x, .y = y }` of Resize.Cursor, as the `cursor` option.
func pl4Cursor(x SizeCellCountInt, y SizeCellCountInt) Optional[PageListResizeCursor] {
	return Some(PageListResizeCursor{x: x, y: y})
}

// pl4ScreenPt, pl4ActivePt and pl4HistoryPt are `.{ .screen = .{ .x = x,
// .y = y } }` and the like.
func pl4ScreenPt(x uint, y uint) Point {
	return Point{Tag: PointTagScreen, Screen: PointCoordinate{x: SizeCellCountInt(x), y: uint32(y)}}
}

func pl4ActivePt(x uint, y uint) Point {
	return Point{Tag: PointTagActive, Active: PointCoordinate{x: SizeCellCountInt(x), y: uint32(y)}}
}

func pl4HistoryPt(x uint, y uint) Point {
	return Point{Tag: PointTagHistory, History: PointCoordinate{x: SizeCellCountInt(x), y: uint32(y)}}
}

// The Zig `Scroll` union literals with a payload.
func pl4ScrollRow(n uint) PageListScroll {
	return PageListScroll{Tag: PageListScrollRow, Row: n}
}

func pl4ScrollDeltaRow(n int64) PageListScroll {
	return PageListScroll{Tag: PageListScrollDeltaRow, DeltaRow: n}
}

func pl4ScrollDeltaPrompt(n int64) PageListScroll {
	return PageListScroll{Tag: PageListScrollDeltaPrompt, DeltaPrompt: n}
}

func pl4ScrollPin(p PageListPin) PageListScroll {
	return PageListScroll{Tag: PageListScrollPin, Pin: p}
}

// pl4Viewport is `.active`, `.top` or `.pin` as a Viewport value.
func pl4Viewport(tag PageListViewportTag) PageListViewport {
	return PageListViewport{Tag: tag}
}

// pl4Pin is `s.pin(pt).?`.
func pl4Pin(t *testing.T, s *PageList, pt Point) PageListPin {
	t.Helper()
	return zigUnwrap(t, s.Pin(pt))
}

// pl4Grow is `try s.grow()`.
func pl4Grow(t *testing.T, s *PageList) *PageListNode {
	t.Helper()
	n, err := s.Grow()
	zigTry(t, err)
	return n
}

// pl4GrowNode is `(try s.grow()).?`.
func pl4GrowNode(t *testing.T, s *PageList) *PageListNode {
	t.Helper()
	n := pl4Grow(t, s)
	if n == nil {
		t.Fatalf("unwrapped a null optional")
	}
	return n
}

// pl4IncreaseCapacity is `try s.increaseCapacity(node, adjustment)`.
func pl4IncreaseCapacity(t *testing.T, s *PageList, node *PageListNode, adjustment PageListIncreaseCapacity) *PageListNode {
	t.Helper()
	n, err := s.IncreaseCapacity(node, Some(adjustment))
	zigTry(t, err)
	return n
}

// pl4TrackPin is `try s.trackPin(p)`.
func pl4TrackPin(t *testing.T, s *PageList, p PageListPin) *PageListPin {
	t.Helper()
	tp, err := s.TrackPin(p)
	zigTry(t, err)
	return tp
}

// pl4InsertHyperlink is `try page.insertHyperlink(link)`.
func pl4InsertHyperlink(t *testing.T, page *Page, link Hyperlink) HyperlinkId {
	t.Helper()
	id, err := page.InsertHyperlink(link)
	zigTry(t, err)
	return id
}

// pl4StylesAdd is `try page.styles.add(page.memory, style)`.
func pl4StylesAdd(t *testing.T, page *Page, style Style) StyleId {
	t.Helper()
	id, err := page.styles.Add(style)
	zigTry(t, err)
	return id
}

// pl4Bold and pl4Italic are `.{ .flags = .{ .bold = true } }` and the like.
func pl4Bold() Style   { return Style{flags: styleFlags{bold: true}} }
func pl4Italic() Style { return Style{flags: styleFlags{italic: true}} }

// ---------------------------------------------------------------------------
// term51: additional tests (K) helpers.
// ---------------------------------------------------------------------------

// pl4KPin is term51KPin.
func pl4KPin(t *testing.T, s *PageList, x uint, y uint) PageListPin {
	t.Helper()
	return pl4Pin(t, s, pl4ScreenPt(x, y))
}

// pl4KLoad is term51KLoad: fill the existing screen rows of `s` from a text
// picture. Each line is one row: a leading `<` (wrap continuation) or `|`,
// one character per cell (`.` empty, `W` wide, `_` spacer tail, `^` spacer
// head, anything else is that codepoint), then `>` (wrapped) or `|`, then an
// optional `P` (prompt) or `C` (prompt continuation). This is the inverse of
// pl4KDump.
func pl4KLoad(t *testing.T, s *PageList, text string) {
	t.Helper()
	y := uint(0)
	for _, line := range strings.Split(text, "\n") {
		end := len(line)
		sp := PageRowSemanticPromptNone
		switch line[end-1] {
		case 'P':
			sp = PageRowSemanticPromptPrompt
			end -= 1
		case 'C':
			sp = PageRowSemanticPromptPromptContinuation
			end -= 1
		}
		wrap := line[end-1] == '>'
		cont := line[0] == '<'
		body := line[1 : end-1]
		if len(body) != int(s.cols) {
			t.Fatalf("TestUnexpectedResult")
		}

		p := pl4KPin(t, s, 0, y)
		row := p.RowAndCell().row
		row.wrap = wrap
		row.wrap_continuation = cont
		row.semantic_prompt = sp
		cells := p.node.data.GetCells(row)
		for x := 0; x < len(body); x++ {
			ch := body[x]
			switch ch {
			case '.':
				cells[x] = PageCell{}
			case 'W':
				cells[x] = PageCell{
					content_tag: PageCellContentTagCodepoint,
					content:     PageCellContent{codepoint: 0x1F600},
					wide:        PageCellWideWide,
				}
			case '_':
				cells[x] = PageCell{
					content_tag: PageCellContentTagCodepoint,
					content:     PageCellContent{codepoint: 0},
					wide:        PageCellWideSpacerTail,
				}
			case '^':
				cells[x] = PageCell{
					content_tag: PageCellContentTagCodepoint,
					content:     PageCellContent{codepoint: 0},
					wide:        PageCellWideSpacerHead,
				}
			default:
				cells[x] = PageCell{
					content_tag: PageCellContentTagCodepoint,
					content:     PageCellContent{codepoint: uint32(ch)},
				}
			}
		}
		y++
	}
}

// pl4KCh is term51KCh.
func pl4KCh(c PageCell) byte {
	switch c.content_tag {
	case PageCellContentTagBgColorPalette, PageCellContentTagBgColorRgb:
		return '#'
	}
	switch c.wide {
	case PageCellWideWide:
		return 'W'
	case PageCellWideSpacerTail:
		return '_'
	case PageCellWideSpacerHead:
		return '^'
	default:
		if c.content.codepoint == 0 {
			return '.'
		} else if c.content.codepoint < 128 {
			return byte(c.content.codepoint)
		}
		return '?'
	}
}

// pl4KDump is term51KDump: render every screen row of `s` in the format
// pl4KLoad reads.
func pl4KDump(t *testing.T, s *PageList) string {
	t.Helper()
	var buf []byte
	it := s.RowIterator(PageListDirectionRightDown, Point{Tag: PointTagScreen}, None[Point]())
	first := true
	for {
		o := it.Next()
		if !o.Valid {
			break
		}
		p := o.Value
		if !first {
			buf = append(buf, '\n')
		}
		first = false
		row := p.RowAndCell().row
		cells := p.node.data.GetCells(row)
		if len(cells) != int(s.cols) {
			t.Fatalf("TestUnexpectedResult")
		}
		if row.wrap_continuation {
			buf = append(buf, '<')
		} else {
			buf = append(buf, '|')
		}
		for _, c := range cells {
			buf = append(buf, pl4KCh(c))
		}
		if row.wrap {
			buf = append(buf, '>')
		} else {
			buf = append(buf, '|')
		}
		switch row.semantic_prompt {
		case PageRowSemanticPromptNone:
		case PageRowSemanticPromptPrompt:
			buf = append(buf, 'P')
		case PageRowSemanticPromptPromptContinuation:
			buf = append(buf, 'C')
		}
	}
	return string(buf)
}

// pl4KExpect is term51KExpect: assert the integrity of the list, the number
// of rows, and that the whole screen renders as `expected`.
func pl4KExpect(t *testing.T, s *PageList, expected string) {
	t.Helper()
	s.AssertIntegrity()
	got := pl4KDump(t, s)
	expectEqualStrings(t, expected, got)
	lines := uint(strings.Count(expected, "\n") + 1)
	expectEqual(t, lines, s.totalRows())
	expectEqual(t, lines, s.total_rows)
	for node := s.pages.first; node != nil; node = node.next {
		expectEqual(t, s.cols, node.data.size.cols)
		expect(t, node.data.size.rows > 0, "node.data.size.rows > 0")
	}
}

// pl4KExpectPin is term51KExpectPin: assert that a pin is at the given
// screen coordinate.
func pl4KExpectPin(t *testing.T, s *PageList, p PageListPin, x uint, y uint) {
	t.Helper()
	pt := zigUnwrap(t, s.PointFromPin(PointTagScreen, p))
	expectEqual(t, x, uint(pt.Screen.x))
	expectEqual(t, y, uint(pt.Screen.y))
}

// pl4KCell is term51KCell.
func pl4KCell(t *testing.T, s *PageList, x uint, y uint) PageCell {
	t.Helper()
	return *pl4KPin(t, s, x, y).RowAndCell().cell
}

// pl4KRow is term51KRow.
func pl4KRow(t *testing.T, s *PageList, y uint) *PageRow {
	t.Helper()
	return pl4KPin(t, s, 0, y).RowAndCell().row
}

// pl4KRowsPerPage is term51KRowsPerPage (the `out` buffer is the result).
func pl4KRowsPerPage(s *PageList) []uint {
	var out []uint
	for node := s.pages.first; node != nil; node = node.next {
		out = append(out, uint(node.data.size.rows))
	}
	return out
}

// pl4KAddNode is term51KAddNode: add a detached-in-spirit node (it IS in the
// list, after the last page) with `rows` used rows and keep the list's row
// cache consistent.
func pl4KAddNode(t *testing.T, s *PageList, cap PageCapacity, rows SizeCellCountInt) *PageListNode {
	t.Helper()
	node, err := s.createPage(cap)
	zigTry(t, err)
	node.data.size.rows = rows
	s.pages.Append(node)
	s.total_rows += uint(rows)
	return node
}

// pl4KReflow is term51KReflow: drives a ReflowCursor over every row of the
// list exactly as resizeCols does, but one row at a time, and without the
// post processing.
type pl4KReflow struct {
	t    *testing.T
	list *PageList
	rc   pageListReflowCursor
	it   PageListRowIterator
}

func pl4KReflowInit(t *testing.T, list *PageList, cols SizeCellCountInt, dst_cap Optional[PageCapacity]) *pl4KReflow {
	t.Helper()
	list.cols = cols
	var node *PageListNode
	{
		page := &list.pages.first.data
		var cap PageCapacity
		if dst_cap.Valid {
			cap = dst_cap.Value
		} else {
			c, err := page.capacity.Adjust(PageCapacityAdjustment{cols: Some(cols)})
			if err != nil {
				c = page.capacity
				c.cols = cols
				c.rows = min(page.size.rows, c.rows)
			}
			cap = c
		}
		n, err := list.createPage(cap)
		zigTry(t, err)
		n.data.size.rows = 1
		node = n
	}
	it := list.RowIterator(PageListDirectionRightDown, Point{Tag: PointTagScreen}, None[Point]())
	list.pages.first = node
	list.pages.last = node
	return &pl4KReflow{t: t, list: list, rc: pageListReflowCursorInit(node), it: it}
}

// step reflows the next source row, returns false when there are no more.
func (self *pl4KReflow) step() bool {
	self.t.Helper()
	o := self.it.Next()
	if !o.Valid {
		return false
	}
	row := o.Value
	zigTry(self.t, self.rc.reflowRow(self.list, row))
	if row.y == row.node.data.size.rows-1 {
		self.list.destroyNode(row.node)
	}
	return true
}

func (self *pl4KReflow) run() {
	self.t.Helper()
	for self.step() {
	}
}

// abort destroys the not yet consumed source pages (what resizeCols does on
// error).
func (self *pl4KReflow) abort() {
	var node_ *PageListNode
	if self.it.chunk.Valid {
		node_ = self.it.chunk.Value.node
	}
	for node_ != nil {
		node := node_
		node_ = node.next
		self.list.destroyNode(node)
	}
}

func (self *pl4KReflow) finish() {
	self.list.total_rows = self.rc.total_rows
}

// pl4KFillStyles is term51KFillStyles: fill every cell of row 0 of the page
// with a distinct style until the style set is full. Returns how many styles
// were added.
func pl4KFillStyles(page *Page) uint {
	n := uint(0)
	for x := uint(0); x < uint(page.size.cols); x++ {
		id, err := page.styles.Add(Style{bg_color: StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{
			r: uint8(x),
			g: uint8(x >> 8),
			b: 1,
		}}})
		if err != nil {
			break
		}
		rac := page.GetRowAndCell(x, 0)
		rac.row.styled = true
		*rac.cell = PageCell{
			content_tag: PageCellContentTagCodepoint,
			content:     PageCellContent{codepoint: 'X'},
			style_id:    id,
		}
		n++
	}
	return n
}

// pl4KFillStyleSet is term51KFillStyleSet: fill the style set of the page
// until it is full without using any cell.
func pl4KFillStyleSet(page *Page) uint {
	n := uint(0)
	for ; ; n++ {
		_, err := page.styles.Add(Style{bg_color: StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{
			r: uint8(n),
			g: uint8(n >> 8),
			b: uint8(n >> 16),
		}}})
		if err != nil {
			break
		}
	}
	return n
}

// pl4KHistory is term51KHistory: 2 columns, 3 rows and 2 history rows with
// markers a and b.
func pl4KHistory(t *testing.T, s *PageList) {
	t.Helper()
	zigTry(t, s.growRows(2))
	pl4KLoad(t, s, "|a.|\n|b.|\n|..|\n|..|\n|..|")
}

// pl4KFirstChunk is term51KFirstChunk.
func pl4KFirstChunk(t *testing.T, s *PageList) PageListPageIteratorChunk {
	t.Helper()
	it := s.PageIterator(PageListDirectionRightDown, Point{Tag: PointTagScreen}, None[Point]())
	return zigUnwrap(t, it.Next())
}

// pl4KGrowColsFixture is term51KGrowColsFixture: a list of 2 pages (200
// columns): the first page has 1 row and as many distinct styles on it (a
// full style set), the second has 3 rows and a style on the rows given by
// `styled`.
func pl4KGrowColsFixture(t *testing.T, s *PageList, styled [3]bool) {
	t.Helper()
	first := &s.pages.first.data
	expect(t, pl4KFillStyles(first) > 50, "try term51KFillStyles(first) > 50")
	_, err := first.styles.Add(Style{flags: styleFlags{faint: true}})
	pl4ExpectError(t, ErrOutOfMemory, err)
	cap := PageCapacityInit(200, 3)
	cap.styles = 128
	node := pl4KAddNode(t, s, cap, 3)
	page := &node.data
	id := pl4StylesAdd(t, page, pl4Bold())
	for y, v := range styled {
		rac := page.GetRowAndCell(0, uint(y))
		*rac.cell = PageCellInit(uint32('a' + y))
		if v {
			rac.cell.style_id = id
			rac.row.styled = true
			if y > 0 {
				page.styles.Use(id)
			}
		}
	}
	if !styled[0] && !styled[1] && !styled[2] {
		page.styles.Release(id)
	}
}

// pl4KTwoPages is term51KTwoPages: a list of 2 pages (the first one full)
// with `cols` columns and no scrollback.
func pl4KTwoPages(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) *PageList {
	t.Helper()
	s := pl4Init(t, cols, rows, pl4Max(0))
	page := &s.pages.first.data
	page.PauseIntegrityChecks(true)
	for i, end := uint(page.size.rows), uint(page.capacity.rows); i < end; i++ {
		pl4Grow(t, s)
	}
	page.PauseIntegrityChecks(false)
	zigTry(t, s.growRows(1))
	expectEqual(t, uint(2), s.TotalPages())
	return s
}

// ---------------------------------------------------------------------------
// term51: additional tests (L) helpers.
// ---------------------------------------------------------------------------

// pl4LGrowRows is term51LGrowRows: grow by n rows with the (expensive)
// per-page integrity check paused on the page that receives the row so that
// large grows stay fast.
func pl4LGrowRows(t *testing.T, s *PageList, n uint) {
	t.Helper()
	for i := uint(0); i < n; i++ {
		func() {
			last := s.pages.last
			last.data.PauseIntegrityChecks(true)
			defer last.data.PauseIntegrityChecks(false)
			pl4Grow(t, s)
		}()
	}
}

// pl4LFillLast is term51LFillLast: grow until the last page has size.rows ==
// capacity.rows.
func pl4LFillLast(t *testing.T, s *PageList) {
	t.Helper()
	last := s.pages.last
	pl4LGrowRows(t, s, uint(last.data.capacity.rows)-uint(last.data.size.rows))
}

// pl4LInitPages is term51LInitPages: three pages of 80 columns, 24 active
// rows: the first two are full (capacity.rows rows each) and the last has
// last_rows rows.
func pl4LInitPages(t *testing.T, last_rows uint) *PageList {
	t.Helper()
	s := pl4Init(t, 80, 24, pl4NoMax())
	pl4LFillLast(t, s)
	pl4GrowNode(t, s)
	pl4LFillLast(t, s)
	pl4GrowNode(t, s)
	pl4LGrowRows(t, s, last_rows-1)
	return s
}

// pl4LRowOf is term51LRowOf: absolute (screen) row of a pin.
func pl4LRowOf(t *testing.T, s *PageList, p PageListPin) uint {
	t.Helper()
	return uint(zigUnwrap(t, s.PointFromPin(PointTagScreen, p)).Screen.y)
}

// pl4LExpectViewportRow is term51LExpectViewportRow: expect the viewport to
// be a pin viewport whose pin is at absolute row n (x == 0), with the cached
// row offset (if any) equal to n and the scrollbar offset equal to n.
func pl4LExpectViewportRow(t *testing.T, s *PageList, n uint, cached bool) {
	t.Helper()
	want := pl4Pin(t, s, pl4ScreenPt(0, n))
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, want.node, s.viewport_pin.node)
	expectEqual(t, want.y, s.viewport_pin.y)
	expectEqual(t, SizeCellCountInt(0), s.viewport_pin.x)
	expectEqual(t, n, pl4LRowOf(t, s, *s.viewport_pin))
	if cached {
		expectEqual(t, Some(n), s.viewport_pin_row_offset)
	} else {
		expectEqual(t, None[uint](), s.viewport_pin_row_offset)
	}
	expectEqual(t, n, s.Scrollbar().offset)
}

// pl4LPinAt is term51LPinAt: put the viewport in pin mode at row n with the
// row offset cached.
func pl4LPinAt(t *testing.T, s *PageList, n uint) {
	t.Helper()
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	s.Scroll(pl4ScrollRow(n))
	pl4LExpectViewportRow(t, s, n, true)
}

// pl4LPromptList is term51LPromptList: total 50 rows, 10 active rows (active
// top is row 40), 80 columns.
func pl4LPromptList(t *testing.T) *PageList {
	t.Helper()
	s := pl4Init(t, 80, 10, pl4NoMax())
	pl4LGrowRows(t, s, 40)
	return s
}

// pl4LMark is term51LMark: set the semantic prompt state of the row at
// absolute row y.
func pl4LMark(t *testing.T, s *PageList, y uint, v PageRowSemanticPrompt) {
	t.Helper()
	pl4Pin(t, s, pl4ScreenPt(0, y)).RowAndCell().row.semantic_prompt = v
}

// pl4LText is term51LText: set a plain text cell at the given pin.
func pl4LText(p PageListPin, cp uint32) {
	*p.RowAndCell().cell = PageCell{
		content_tag: PageCellContentTagCodepoint,
		content:     PageCellContent{codepoint: cp},
	}
}

// pl4LMakeOversized is term51LMakeOversized: make the page of the given node
// larger than a standard page by doubling its style capacity. Returns the
// replacement node.
func pl4LMakeOversized(t *testing.T, s *PageList, node *PageListNode) *PageListNode {
	t.Helper()
	n := node
	for pageListMemoryLen(&n.data) <= pageListStdSize() {
		n = pl4IncreaseCapacity(t, s, n, PageListIncreaseCapacityStyles)
	}
	return n
}

// Zig: test "term51: ReflowCursor.reflowRow: background color cells are not trimmed"
func TestPl4_term51_ReflowCursor_reflowRow_background_color_cells_are_not_trimmed(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 2, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|a...|\n|....|")
	// Row 0: a, blank, bg palette, bg rgb. Row 1: only a bg cell.
	cells0 := pl4KPin(t, s, 0, 0).Cells(PageListPinCellSubsetAll)
	cells0[2] = PageCell{content_tag: PageCellContentTagBgColorPalette, content: PageCellContent{color_palette: 4}}
	cells0[3] = PageCell{content_tag: PageCellContentTagBgColorRgb, content: PageCellContent{color_rgb: PageCellRGB{r: 1, g: 2, b: 3}}}
	cells1 := pl4KPin(t, s, 0, 1).Cells(PageListPinCellSubsetAll)
	cells1[1] = PageCell{content_tag: PageCellContentTagBgColorPalette, content: PageCellContent{color_palette: 7}}

	r := pl4KReflowInit(t, s, 3, None[PageCapacity]())
	r.run()
	r.finish()
	pl4KExpect(t, s, "|a.#>\n<#..|\n|.#.|")
	expectEqual(t, uint8(4), pl4KCell(t, s, 2, 0).content.color_palette)
	expectEqual(t, PageCellContentTagBgColorRgb, pl4KCell(t, s, 0, 1).content_tag)
	expectEqual(t, uint8(3), pl4KCell(t, s, 0, 1).content.color_rgb.b)
	expectEqual(t, uint8(7), pl4KCell(t, s, 1, 2).content.color_palette)
}

// Zig: test "term51: ReflowCursor.reflowRow: cell flags survive the reflow"
func TestPl4_term51_ReflowCursor_reflowRow_cell_flags_survive_the_reflow(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 1, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|abcd|")
	cells := pl4KPin(t, s, 0, 0).Cells(PageListPinCellSubsetAll)
	cells[1].protected = true
	cells[2].semantic_content = PageCellSemanticContentInput
	cells[3].semantic_content = PageCellSemanticContentPrompt

	r := pl4KReflowInit(t, s, 2, None[PageCapacity]())
	r.run()
	r.finish()
	pl4KExpect(t, s, "|ab>\n<cd|")
	expect(t, pl4KCell(t, s, 1, 0).protected, "term51KCell(&s, 1, 0).protected")
	expect(t, !pl4KCell(t, s, 0, 0).protected, "!term51KCell(&s, 0, 0).protected")
	expectEqual(t, PageCellSemanticContentInput, pl4KCell(t, s, 0, 1).semantic_content)
	expectEqual(t, PageCellSemanticContentPrompt, pl4KCell(t, s, 1, 1).semantic_content)
}

// Zig: test "term51: ReflowCursor.reflowRow: graphemes follow their cell across the wrap"
func TestPl4_term51_ReflowCursor_reflowRow_graphemes_follow_their_cell_across_the_wrap(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 2, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|abcd|\n|efgh|")
	{
		page := &s.pages.first.data
		c := page.GetRowAndCell(2, 0)
		zigTry(t, page.AppendGrapheme(c.row, c.cell, 0x301))
		zigTry(t, page.AppendGrapheme(c.row, c.cell, 0x302))
		g := page.GetRowAndCell(0, 1)
		zigTry(t, page.AppendGrapheme(g.row, g.cell, 0x303))
	}

	r := pl4KReflowInit(t, s, 2, None[PageCapacity]())
	r.run()
	r.finish()
	pl4KExpect(t, s, "|ab>\n<cd|\n|ef>\n<gh|")
	page := &s.pages.first.data
	expectEqual(t, uint(2), page.GraphemeCount())
	// c is at x=0 of the second row, e at x=0 of the third.
	c := page.GetRowAndCell(0, 1)
	expect(t, c.row.grapheme, "c.row.grapheme")
	expectEqualSlices(t, []uint32{0x301, 0x302}, zigUnwrap(t, page.LookupGrapheme(c.cell)))
	e := page.GetRowAndCell(0, 2)
	expect(t, e.row.grapheme, "e.row.grapheme")
	expectEqualSlices(t, []uint32{0x303}, zigUnwrap(t, page.LookupGrapheme(e.cell)))
	// Rows without graphemes are not marked.
	expect(t, !page.GetRowAndCell(0, 0).row.grapheme, "!page.getRowAndCell(0, 0).row.grapheme")
	expect(t, !page.GetRowAndCell(0, 3).row.grapheme, "!page.getRowAndCell(0, 3).row.grapheme")
	page.AssertIntegrity()
}

// Zig: test "term51: ReflowCursor.reflowRow: styles follow their cells and are shared"
func TestPl4_term51_ReflowCursor_reflowRow_styles_follow_their_cells_and_are_shared(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 1, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|abcd|")
	{
		page := &s.pages.first.data
		bold := pl4StylesAdd(t, page, pl4Bold())
		rac0 := page.GetRowAndCell(0, 0)
		rac0.cell.style_id = bold
		rac0.row.styled = true
		page.styles.Use(bold)
		page.GetRowAndCell(3, 0).cell.style_id = bold
	}

	r := pl4KReflowInit(t, s, 2, None[PageCapacity]())
	r.run()
	r.finish()
	pl4KExpect(t, s, "|ab>\n<cd|")
	page := &s.pages.first.data
	a := page.GetRowAndCell(0, 0)
	d := page.GetRowAndCell(1, 1)
	expect(t, a.row.styled, "a.row.styled")
	expect(t, d.row.styled, "d.row.styled")
	expect(t, a.cell.style_id != 0, "a.cell.style_id != 0")
	expectEqual(t, a.cell.style_id, d.cell.style_id)
	expect(t, page.styles.Get(a.cell.style_id).flags.bold, "page.styles.get(page.memory, a.cell.style_id).flags.bold")
	expectEqual(t, SizeCellCountInt(2), page.styles.RefCount(a.cell.style_id))
	expectEqual(t, uint(1), page.styles.living)
	page.AssertIntegrity()
}

// Zig: test "term51: ReflowCursor.reflowRow: hyperlinks follow their cells and are shared"
func TestPl4_term51_ReflowCursor_reflowRow_hyperlinks_follow_their_cells_and_are_shared(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 1, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|abcd|")
	{
		page := &s.pages.first.data
		id := pl4InsertHyperlink(t, page, Hyperlink{
			id:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdExplicit, Explicit: []uint8("e")},
			uri: []uint8("https://x.example"),
		})
		c0 := page.GetRowAndCell(0, 0)
		zigTry(t, page.SetHyperlink(c0.row, c0.cell, id))
		c3 := page.GetRowAndCell(3, 0)
		page.hyperlink_set.Use(id)
		zigTry(t, page.SetHyperlink(c3.row, c3.cell, id))
	}

	r := pl4KReflowInit(t, s, 2, None[PageCapacity]())
	r.run()
	r.finish()
	pl4KExpect(t, s, "|ab>\n<cd|")
	page := &s.pages.first.data
	expect(t, page.GetRowAndCell(0, 0).row.hyperlink, "page.getRowAndCell(0, 0).row.hyperlink")
	expect(t, page.GetRowAndCell(0, 1).row.hyperlink, "page.getRowAndCell(0, 1).row.hyperlink")
	expectEqual(t, uint(2), page.HyperlinkCount())
	a := page.GetRowAndCell(0, 0).cell
	d := page.GetRowAndCell(1, 1).cell
	expect(t, a.hyperlink && d.hyperlink, "a.hyperlink and d.hyperlink")
	expect(t, !page.GetRowAndCell(1, 0).cell.hyperlink, "!page.getRowAndCell(1, 0).cell.hyperlink")
	id := zigUnwrap(t, page.LookupHyperlink(a))
	expectEqual(t, id, zigUnwrap(t, page.LookupHyperlink(d)))
	expectEqual(t, SizeCellCountInt(2), page.hyperlink_set.RefCount(id))
	e := page.hyperlink_set.Get(id)
	expectEqualStrings(t, "https://x.example", string(e.uri))
	expectEqualStrings(t, "e", string(e.id.Explicit))
	page.AssertIntegrity()
}

// Zig: test "term51: ReflowCursor.reflowRow: new pages are created when the page is full and pins follow"
func TestPl4_term51_ReflowCursor_reflowRow_new_pages_are_created_when_the_page_is_full_and_pins_follow(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 5, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|a...|\n|b...|\n|c...|\n|d...|\n|e...|")
	p_b := pl4TrackPin(t, s, pl4KPin(t, s, 0, 1))
	p_c := pl4TrackPin(t, s, pl4KPin(t, s, 0, 2))
	p_e := pl4TrackPin(t, s, pl4KPin(t, s, 0, 4))
	defer s.UntrackPin(p_b)
	defer s.UntrackPin(p_c)
	defer s.UntrackPin(p_e)

	// The first destination page only holds two rows.
	r := pl4KReflowInit(t, s, 4, Some(PageCapacityInit(4, 2)))
	r.run()
	r.finish()
	pl4KExpect(t, s, "|a...|\n|b...|\n|c...|\n|d...|\n|e...|")
	expectEqual(t, uint(2), pl4KRowsPerPage(s)[0])
	expectEqual(t, 2, len(pl4KRowsPerPage(s)))
	expectEqual(t, uint(3), pl4KRowsPerPage(s)[1])
	expectEqual(t, s.pages.first, p_b.node)
	expectEqual(t, s.pages.last, p_c.node)
	expectEqual(t, s.pages.last, p_e.node)
	expectEqual(t, SizeCellCountInt(1), p_b.y)
	expectEqual(t, SizeCellCountInt(0), p_c.y)
	expectEqual(t, SizeCellCountInt(2), p_e.y)
}

// Zig: test "term51: ReflowCursor.reflowRow: new pages inherit the capacity of the source page"
func TestPl4_term51_ReflowCursor_reflowRow_new_pages_inherit_the_capacity_of_the_source_page(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 3, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|a...|\n|b...|\n|c...|")
	// Give the source page more styles.
	pl4IncreaseCapacity(t, s, s.pages.first, PageListIncreaseCapacityStyles)
	pl4IncreaseCapacity(t, s, s.pages.first, PageListIncreaseCapacityGraphemeBytes)
	src_cap := s.pages.first.data.capacity
	expectEqual(t, SizeStyleCountInt(256), src_cap.styles)

	r := pl4KReflowInit(t, s, 2, Some(PageCapacityInit(2, 1)))
	r.run()
	r.finish()
	pl4KExpect(t, s, "|a.|\n|b.|\n|c.|")
	expectEqualSlices(t, []uint{1, 2}, pl4KRowsPerPage(s))
	n := uint(0)
	for node := s.pages.first; node != nil; node = node.next {
		n += 1
		if n == 1 {
			continue
		}
		expectEqual(t, SizeCellCountInt(2), node.data.capacity.cols)
		expectEqual(t, SizeStyleCountInt(256), node.data.capacity.styles)
		expectEqual(t, src_cap.grapheme_bytes, node.data.capacity.grapheme_bytes)
		expectEqual(t, src_cap.string_bytes, node.data.capacity.string_bytes)
		expectEqual(t, src_cap.hyperlink_bytes, node.data.capacity.hyperlink_bytes)
	}
	expectEqual(t, uint(2), n)
}

// Zig: test "term51: ReflowCursor.reflowRow: layout capacity of new pages falls back to the source row count"
func TestPl4_term51_ReflowCursor_reflowRow_layout_capacity_of_new_pages_falls_back_to_the_source_row_count(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 3, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|a...|\n|b...|\n|c...|")

	// A destination too wide for a standard page, with a single row.
	cols := zigUnwrap(t, PageStdCapacity.MaxCols()) + 1
	r := pl4KReflowInit(t, s, cols, Some(PageCapacityInit(cols, 1)))
	r.run()
	r.finish()
	// One page with the requested row, then a page with 3 rows of capacity
	// that holds the other two rows.
	expectEqualSlices(t, []uint{1, 2}, pl4KRowsPerPage(s))
	n := uint(0)
	for node := s.pages.first; node != nil; node = node.next {
		n += 1
		expectEqual(t, cols, node.data.capacity.cols)
		// min(source rows, standard rows) for the pages made by the reflow.
		if n > 1 {
			expectEqual(t, SizeCellCountInt(3), node.data.capacity.rows)
		}
	}
	expectEqual(t, uint32('c'), pl4KCell(t, s, 0, 2).content.codepoint)
}

// Zig: test "term51: ReflowCursor.reflowRow: layout capacity of new pages is capped to the standard row count"
func TestPl4_term51_ReflowCursor_reflowRow_layout_capacity_of_new_pages_is_capped_to_the_standard_row_count(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 300, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|a...|\n|b...|"+strings.Repeat("\n|....|", 298))

	cols := zigUnwrap(t, PageStdCapacity.MaxCols()) + 1
	r := pl4KReflowInit(t, s, cols, Some(PageCapacityInit(cols, 1)))
	// Only the two rows with text are reflowed.
	expect(t, r.step(), "try r.step()")
	expect(t, r.step(), "try r.step()")
	expectEqual(t, SizeCellCountInt(1), r.rc.page.size.rows)
	expectEqual(t, SizeCellCountInt(215), r.rc.page.capacity.rows)
	r.abort()
	r.finish()
	expectEqual(t, uint(2), s.TotalPages())
	s.AssertIntegrity()
}

// Zig: test "term51: ReflowCursor.reflowRow: out of space on the first row drops the style of the cell"
func TestPl4_term51_ReflowCursor_reflowRow_out_of_space_on_the_first_row_drops_the_style_of_the_cell(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 1, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|abc.|")
	{
		page := &s.pages.first.data
		bold := pl4StylesAdd(t, page, pl4Bold())
		c := page.GetRowAndCell(1, 0)
		c.cell.style_id = bold
		c.row.styled = true
	}

	// The destination cannot have more styles and the set is full.
	max := SizeStyleCountInt(math.MaxUint16)
	dst := PageCapacityInit(4, 2)
	dst.styles = max
	r := pl4KReflowInit(t, s, 4, Some(dst))
	expect(t, pl4KFillStyleSet(r.rc.page) > 1000, "term51KFillStyleSet(r.rc.page) > 1000")
	r.run()
	r.finish()

	// The text is kept, the style is lost and the row is not flagged.
	pl4KExpect(t, s, "|abc.|")
	expectEqual(t, SizeStyleCountInt(0), pl4KCell(t, s, 1, 0).style_id)
	expect(t, !pl4KRow(t, s, 0).styled, "!term51KRow(&s, 0).styled")
	expectEqual(t, uint(1), s.TotalPages())
	s.AssertIntegrity()
}

// Zig: test "term51: ReflowCursor.reflowRow: out of space after the first row moves the row to a new page"
func TestPl4_term51_ReflowCursor_reflowRow_out_of_space_after_the_first_row_moves_the_row_to_a_new_page(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 2, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|a...|\n|bcd.|")
	{
		page := &s.pages.first.data
		bold := pl4StylesAdd(t, page, pl4Bold())
		c := page.GetRowAndCell(1, 1)
		c.cell.style_id = bold
		c.row.styled = true
	}
	p := pl4TrackPin(t, s, pl4KPin(t, s, 1, 1))
	defer s.UntrackPin(p)

	max := SizeStyleCountInt(math.MaxUint16)
	dst := PageCapacityInit(4, 2)
	dst.styles = max
	r := pl4KReflowInit(t, s, 4, Some(dst))
	expect(t, pl4KFillStyleSet(r.rc.page) > 1000, "term51KFillStyleSet(r.rc.page) > 1000")
	r.run()
	r.finish()

	// The first page keeps the first row, the second row went to a page that
	// inherits the capacity of the source page and has room for the style.
	pl4KExpect(t, s, "|a...|\n|bcd.|")
	expectEqualSlices(t, []uint{1, 1}, pl4KRowsPerPage(s))
	expectEqual(t, SizeStyleCountInt(128), s.pages.last.data.capacity.styles)
	c := pl4KCell(t, s, 1, 1)
	expect(t, c.style_id != 0, "c.style_id != 0")
	expect(t, s.pages.last.data.styles.Get(c.style_id).flags.bold, "s.pages.last.?.data.styles.get(s.pages.last.?.data.memory, c.style_id).flags.bold")
	expect(t, pl4KRow(t, s, 1).styled, "term51KRow(&s, 1).styled")
	// The pin followed the row.
	expectEqual(t, s.pages.last, p.node)
	pl4KExpectPin(t, s, *p, 1, 1)
	s.AssertIntegrity()
}

// Zig: test "term51: resizeWithoutReflow: layout min_max_size is only recomputed without reflow"
func TestPl4_term51_resizeWithoutReflow_layout_min_max_size_is_only_recomputed_without_reflow(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 20, 3, pl4Max(0))
	defer s.Deinit()
	old := s.min_max_size

	// With reflow the caller (resize) is responsible: unchanged here.
	zigTry(t, s.resizeWithoutReflow(PageListResize{rows: pl4S(40), reflow: true}))
	expectEqual(t, old, s.min_max_size)
	expectEqual(t, uint(40), uint(s.rows))

	zigTry(t, s.resizeWithoutReflow(PageListResize{cols: pl4S(3000), rows: pl4S(41), reflow: false}))
	expectEqual(t, pageListMinMaxSize(3000, 41), s.min_max_size)
	expect(t, s.min_max_size > old, "s.min_max_size > old")
	expectEqual(t, uint(3000), uint(s.cols))
	expectEqual(t, uint(41), uint(s.rows))

	// Only one of the dimensions: the other one is the current one.
	zigTry(t, s.resizeWithoutReflow(PageListResize{rows: pl4S(5), reflow: false}))
	expectEqual(t, pageListMinMaxSize(3000, 5), s.min_max_size)
	zigTry(t, s.resizeWithoutReflow(PageListResize{cols: pl4S(20), reflow: false}))
	expectEqual(t, pageListMinMaxSize(20, 5), s.min_max_size)
}

// Zig: test "term51: resizeWithoutReflow: no change"
func TestPl4_term51_resizeWithoutReflow_no_change(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 3, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|abcd>\n<ef..|\n|g...|")

	zigTry(t, s.resizeWithoutReflow(PageListResize{reflow: true}))
	zigTry(t, s.resizeWithoutReflow(PageListResize{cols: pl4S(4), rows: pl4S(3), reflow: true}))
	pl4KExpect(t, s, "|abcd>\n<ef..|\n|g...|")
	expectEqual(t, uint(4), uint(s.cols))
	expectEqual(t, uint(3), uint(s.rows))
}

// Zig: test "term51: resizeWithoutReflow: fewer cols truncates rows and clamps pins"
func TestPl4_term51_resizeWithoutReflow_fewer_cols_truncates_rows_and_clamps_pins(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 6, 3, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|abcdef>\n<ghijkl|\n|mn....|")
	p5 := pl4TrackPin(t, s, pl4KPin(t, s, 5, 0))
	p3 := pl4TrackPin(t, s, pl4KPin(t, s, 3, 1))
	p1 := pl4TrackPin(t, s, pl4KPin(t, s, 1, 2))
	defer s.UntrackPin(p5)
	defer s.UntrackPin(p3)
	defer s.UntrackPin(p1)

	zigTry(t, s.resizeWithoutReflow(PageListResize{cols: pl4S(3), reflow: false}))
	expectEqual(t, uint(3), uint(s.cols))
	expectEqual(t, uint(3), uint(s.rows))
	pl4KExpect(t, s, "|abc>\n<ghi|\n|mn.|")
	pl4KExpectPin(t, s, *p5, 2, 0)
	pl4KExpectPin(t, s, *p3, 2, 1)
	pl4KExpectPin(t, s, *p1, 1, 2)
	// The page keeps the capacity for its old columns.
	expectEqual(t, SizeCellCountInt(6), s.pages.first.data.capacity.cols)
	expectEqual(t, SizeCellCountInt(3), s.pages.first.data.size.cols)
}

// Zig: test "term51: resizeWithoutReflow: fewer cols releases managed memory beyond the edge"
func TestPl4_term51_resizeWithoutReflow_fewer_cols_releases_managed_memory_beyond_the_edge(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 5, 2, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|abcde|\n|abcde|")
	{
		page := &s.pages.first.data
		// Row 0: grapheme at x=1 (kept) and x=3 (cut), style at x=4 (cut),
		// hyperlink at x=3 (cut).
		g1 := page.GetRowAndCell(1, 0)
		zigTry(t, page.AppendGrapheme(g1.row, g1.cell, 0x301))
		g3 := page.GetRowAndCell(3, 0)
		zigTry(t, page.AppendGrapheme(g3.row, g3.cell, 0x302))
		bold := pl4StylesAdd(t, page, pl4Bold())
		st := page.GetRowAndCell(4, 0)
		st.cell.style_id = bold
		st.row.styled = true
		link := pl4InsertHyperlink(t, page, Hyperlink{
			id:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: 1},
			uri: []uint8("https://a"),
		})
		zigTry(t, page.SetHyperlink(g3.row, g3.cell, link))

		// Row 1: a style that is kept.
		keep := pl4StylesAdd(t, page, pl4Italic())
		k := page.GetRowAndCell(0, 1)
		k.cell.style_id = keep
		k.row.styled = true
	}

	zigTry(t, s.resizeWithoutReflow(PageListResize{cols: pl4S(3), reflow: false}))
	page := &s.pages.first.data
	expectEqual(t, uint(1), page.GraphemeCount())
	expectEqual(t, uint(0), page.HyperlinkCount())
	expectEqual(t, uint(1), page.styles.living)
	r0 := page.GetRowAndCell(0, 0).row
	expect(t, r0.grapheme, "r0.grapheme")
	expect(t, !r0.hyperlink, "!r0.hyperlink")
	// The style of the cut cell was released but the flag is only updated
	// before the cells are zeroed, so it stays set on the row.
	expect(t, r0.styled, "r0.styled")
	expect(t, page.GetRowAndCell(0, 1).row.styled, "page.getRowAndCell(0, 1).row.styled")
	expectEqualSlices(t, []uint32{0x301}, zigUnwrap(t, page.LookupGrapheme(page.GetRowAndCell(1, 0).cell)))
	page.AssertIntegrity()
}

// Zig: test "term51: resizeWithoutReflow: fewer cols can split a wide character from its tail"
func TestPl4_term51_resizeWithoutReflow_fewer_cols_can_split_a_wide_character_from_its_tail(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 2, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|abW_|\n|ab.^>")
	// The spacer head sits in the last column of a wrapped row.
	zigTry(t, s.resizeWithoutReflow(PageListResize{cols: pl4S(3), reflow: false}))
	// The wide cell is now the last column without its tail, and the spacer
	// head was cut away.
	pl4KExpect(t, s, "|abW|\n|ab.>")
}

// Zig: test "term51: resizeWithoutReflow: fewer rows trims blank rows instead of making history"
func TestPl4_term51_resizeWithoutReflow_fewer_rows_trims_blank_rows_instead_of_making_history(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 2, 5, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|a.|\n|b.|\n|..|\n|..|\n|..|")

	zigTry(t, s.resizeWithoutReflow(PageListResize{rows: pl4S(3), reflow: true}))
	expectEqual(t, uint(3), uint(s.rows))
	pl4KExpect(t, s, "|a.|\n|b.|\n|..|")

	// One blank row is left before the text: only that one is trimmed.
	zigTry(t, s.resizeWithoutReflow(PageListResize{rows: pl4S(1), reflow: true}))
	expectEqual(t, uint(1), uint(s.rows))
	pl4KExpect(t, s, "|a.|\n|b.|")

	// No blank rows: history is made and nothing is removed.
	zigTry(t, s.resizeWithoutReflow(PageListResize{rows: pl4S(1), reflow: true}))
	pl4KExpect(t, s, "|a.|\n|b.|")
	pl4KExpectPin(t, s, s.GetTopLeft(PointTagActive), 0, 1)
}

// Zig: test "term51: resizeWithoutReflow: trimmed rows stop at a tracked pin"
func TestPl4_term51_resizeWithoutReflow_trimmed_rows_stop_at_a_tracked_pin(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 2, 5, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|a.|\n|b.|\n|..|\n|..|\n|..|")
	p := pl4TrackPin(t, s, pl4KPin(t, s, 0, 3))
	defer s.UntrackPin(p)

	zigTry(t, s.resizeWithoutReflow(PageListResize{rows: pl4S(1), reflow: true}))
	// Row 4 is trimmed, row 3 holds the pin so the rest stays as history.
	expectEqual(t, uint(1), uint(s.rows))
	pl4KExpect(t, s, "|a.|\n|b.|\n|..|\n|..|")
	pl4KExpectPin(t, s, *p, 0, 3)
}

// Zig: test "term51: resizeWithoutReflow: cols and rows together"
func TestPl4_term51_resizeWithoutReflow_cols_and_rows_together(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 3, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|abcd>\n<ef..|\n|....|")

	zigTry(t, s.resizeWithoutReflow(PageListResize{cols: pl4S(2), rows: pl4S(2), reflow: true}))
	pl4KExpect(t, s, "|ab>\n<ef|")
	expectEqual(t, uint(2), uint(s.cols))
	expectEqual(t, uint(2), uint(s.rows))

	zigTry(t, s.resizeWithoutReflow(PageListResize{cols: pl4S(3), rows: pl4S(3), reflow: true}))
	pl4KExpect(t, s, "|ab.>\n<ef.|\n|...|")
}

// Zig: test "term51: resizeWithoutReflow: more rows without a cursor pulls history into the active area"
func TestPl4_term51_resizeWithoutReflow_more_rows_without_a_cursor_pulls_history_into_the_active_area(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 2, 3, pl4NoMax())
	defer s.Deinit()
	pl4KHistory(t, s)

	zigTry(t, s.resizeWithoutReflow(PageListResize{rows: pl4S(5), reflow: true}))
	expectEqual(t, uint(5), uint(s.rows))
	pl4KExpect(t, s, "|a.|\n|b.|\n|..|\n|..|\n|..|")
	pl4KExpectPin(t, s, s.GetTopLeft(PointTagActive), 0, 0)
}

// Zig: test "term51: resizeWithoutReflow: more rows with the cursor at the bottom pulls history down"
func TestPl4_term51_resizeWithoutReflow_more_rows_with_the_cursor_at_the_bottom_pulls_history_down(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 2, 3, pl4NoMax())
	defer s.Deinit()
	pl4KHistory(t, s)

	// y == rows - 1
	zigTry(t, s.resizeWithoutReflow(PageListResize{rows: pl4S(5), reflow: true, cursor: pl4Cursor(0, 2)}))
	pl4KExpect(t, s, "|a.|\n|b.|\n|..|\n|..|\n|..|")
	pl4KExpectPin(t, s, s.GetTopLeft(PointTagActive), 0, 0)
}

// Zig: test "term51: resizeWithoutReflow: more rows with the cursor beyond the bottom pulls history down"
func TestPl4_term51_resizeWithoutReflow_more_rows_with_the_cursor_beyond_the_bottom_pulls_history_down(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 2, 3, pl4NoMax())
	defer s.Deinit()
	pl4KHistory(t, s)

	zigTry(t, s.resizeWithoutReflow(PageListResize{rows: pl4S(5), reflow: true, cursor: pl4Cursor(0, 9)}))
	pl4KExpect(t, s, "|a.|\n|b.|\n|..|\n|..|\n|..|")
	pl4KExpectPin(t, s, s.GetTopLeft(PointTagActive), 0, 0)
}

// Zig: test "term51: resizeWithoutReflow: more rows with the cursor above the bottom adds rows"
func TestPl4_term51_resizeWithoutReflow_more_rows_with_the_cursor_above_the_bottom_adds_rows(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 2, 3, pl4NoMax())
	defer s.Deinit()
	pl4KHistory(t, s)

	zigTry(t, s.resizeWithoutReflow(PageListResize{rows: pl4S(5), reflow: true, cursor: pl4Cursor(0, 0)}))
	expectEqual(t, uint(5), uint(s.rows))
	// History is kept as history: 2 rows were added.
	pl4KExpect(t, s, "|a.|\n|b.|\n|..|\n|..|\n|..|\n|..|\n|..|")
	pl4KExpectPin(t, s, s.GetTopLeft(PointTagActive), 0, 2)
}

// Zig: test "term51: resizeWithoutReflow: more rows than history plus active adds the difference"
func TestPl4_term51_resizeWithoutReflow_more_rows_than_history_plus_active_adds_the_difference(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 2, 3, pl4NoMax())
	defer s.Deinit()
	zigTry(t, s.growRows(1))
	pl4KLoad(t, s, "|a.|\n|..|\n|..|\n|..|")

	zigTry(t, s.resizeWithoutReflow(PageListResize{rows: pl4S(6), reflow: true}))
	pl4KExpect(t, s, "|a.|\n|..|\n|..|\n|..|\n|..|\n|..|")
	pl4KExpectPin(t, s, s.GetTopLeft(PointTagActive), 0, 0)
}

// Zig: test "term51: resizeWithoutReflow: more rows without history adds rows"
func TestPl4_term51_resizeWithoutReflow_more_rows_without_history_adds_rows(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 2, 3, pl4NoMax())
	defer s.Deinit()
	pl4KLoad(t, s, "|a.|\n|..|\n|..|")

	zigTry(t, s.resizeWithoutReflow(PageListResize{rows: pl4S(5), reflow: true}))
	pl4KExpect(t, s, "|a.|\n|..|\n|..|\n|..|\n|..|")
	s2 := pl4Init(t, 2, 3, pl4NoMax())
	defer s2.Deinit()
	zigTry(t, s2.resizeWithoutReflow(PageListResize{rows: pl4S(5), reflow: true, cursor: pl4Cursor(0, 0)}))
	expectEqual(t, uint(5), s2.totalRows())
}

// Zig: test "term51: resizeWithoutReflow: one row"
func TestPl4_term51_resizeWithoutReflow_one_row(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 3, 3, pl4NoMax())
	defer s.Deinit()
	pl4KLoad(t, s, "|abc|\n|d..|\n|..e|")

	zigTry(t, s.resizeWithoutReflow(PageListResize{rows: pl4S(1), cols: pl4S(1), reflow: true}))
	// The row with the cut text is blank and trimmed.
	pl4KExpect(t, s, "|a|\n|d|")
	expectEqual(t, uint(1), uint(s.rows))
	expectEqual(t, uint(1), uint(s.cols))
}

// Zig: test "term51: resizeWithoutReflow: more rows moves a pinned viewport into the active area"
func TestPl4_term51_resizeWithoutReflow_more_rows_moves_a_pinned_viewport_into_the_active_area(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 2, 4, pl4NoMax())
	defer s.Deinit()
	zigTry(t, s.growRows(8))
	s.Scroll(pl4ScrollPin(pl4Pin(t, s, Point{Tag: PointTagScreen, Screen: PointCoordinate{y: 7}})))
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")

	zigTry(t, s.resizeWithoutReflow(PageListResize{rows: pl4S(6), reflow: true}))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	// A viewport above stays where it is.
	s.Scroll(pl4ScrollPin(pl4Pin(t, s, Point{Tag: PointTagScreen, Screen: PointCoordinate{y: 2}})))
	zigTry(t, s.resizeWithoutReflow(PageListResize{rows: pl4S(8), reflow: true}))
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	zigTry(t, s.resizeWithoutReflow(PageListResize{rows: pl4S(11), reflow: true}))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
}

// Zig: test "term51: resizeWithoutReflowGrowCols: spare capacity just widens the page"
func TestPl4_term51_resizeWithoutReflowGrowCols_spare_capacity_just_widens_the_page(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 3, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|abcd>\n<ef..|\n|....|")
	zigTry(t, s.resizeWithoutReflow(PageListResize{cols: pl4S(2), reflow: false}))
	pl4KExpect(t, s, "|ab>\n<ef|\n|..|")

	node := s.pages.first
	tpin := pl4TrackPin(t, s, PageListPin{node: node, x: 1, y: 1})
	defer s.UntrackPin(tpin)
	expectEqual(t, uint(2), uint(s.cols))
	zigTry(t, s.resizeWithoutReflowGrowCols(4, pl4KFirstChunk(t, s)))
	// The same node: nothing was copied.
	expectEqual(t, node, s.pages.first)
	expectEqual(t, node, tpin.node)
	expectEqual(t, uint(4), uint(s.cols))
	expectEqual(t, SizeCellCountInt(4), node.data.size.cols)
	// The cut cells stay cleared.
	pl4KExpect(t, s, "|ab..>\n<ef..|\n|....|")
}

// Zig: test "term51: resizeWithoutReflowGrowCols: a spacer head at the old edge takes the slow path"
func TestPl4_term51_resizeWithoutReflowGrowCols_a_spacer_head_at_the_old_edge_takes_the_slow_path(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 2, pl4Max(0))
	defer s.Deinit()
	zigTry(t, s.resizeWithoutReflow(PageListResize{cols: pl4S(3), reflow: false}))
	pl4KLoad(t, s, "|ab^>\n|c..|")
	node := s.pages.first
	tpin := pl4TrackPin(t, s, PageListPin{node: node, x: 0, y: 1})
	defer s.UntrackPin(tpin)

	zigTry(t, s.resizeWithoutReflowGrowCols(4, pl4KFirstChunk(t, s)))
	// Capacity was enough but the row had to be cloned into a new page.
	expect(t, s.pages.first != node, "s.pages.first.? != node")
	expectEqual(t, s.pages.first, tpin.node)
	expectEqual(t, SizeCellCountInt(1), tpin.y)
	expectEqual(t, uint(1), s.TotalPages())
	// The spacer head became an empty narrow cell and the wrap was removed.
	pl4KExpect(t, s, "|ab..|\n|c...|")
}

// Zig: test "term51: resizeWithoutReflowGrowCols: a page without capacity is cloned into a wider one"
func TestPl4_term51_resizeWithoutReflowGrowCols_a_page_without_capacity_is_cloned_into_a_wider_one(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 3, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|abcd>P\n<ef..|C\n|g...|")
	{
		page := &s.pages.first.data
		g := page.GetRowAndCell(0, 2)
		zigTry(t, page.AppendGrapheme(g.row, g.cell, 0x301))
		bold := pl4StylesAdd(t, page, pl4Bold())
		b := page.GetRowAndCell(1, 2)
		*b.cell = PageCell{content_tag: PageCellContentTagCodepoint, content: PageCellContent{codepoint: 'h'}, style_id: bold}
		b.row.styled = true
		link := pl4InsertHyperlink(t, page, Hyperlink{
			id:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: 3},
			uri: []uint8("https://l"),
		})
		l := page.GetRowAndCell(2, 2)
		*l.cell = PageCellInit('i')
		zigTry(t, page.SetHyperlink(l.row, l.cell, link))
	}
	node := s.pages.first
	tpin := pl4TrackPin(t, s, PageListPin{node: node, x: 2, y: 1})
	defer s.UntrackPin(tpin)
	old_page_size := s.page_size

	expectEqual(t, SizeCellCountInt(4), node.data.capacity.cols)
	zigTry(t, s.resizeWithoutReflowGrowCols(6, pl4KFirstChunk(t, s)))
	expectEqual(t, uint(6), uint(s.cols))
	expect(t, s.pages.first != node, "s.pages.first.? != node")
	expectEqual(t, uint(1), s.TotalPages())
	page := &s.pages.first.data
	expectEqual(t, SizeCellCountInt(6), page.capacity.cols)
	expectEqual(t, SizeCellCountInt(6), page.size.cols)
	expectEqual(t, SizeCellCountInt(3), page.size.rows)
	expectEqual(t, old_page_size, s.page_size)
	// The cloned rows lose their wrap flags (the copy is treated as a partial
	// row copy because the source row has fewer cells) but keep the prompt.
	pl4KExpect(t, s, "|abcd..|P\n|ef....|C\n|ghi...|")
	// Managed memory moved along.
	expectEqual(t, uint(1), page.GraphemeCount())
	expectEqualSlices(t, []uint32{0x301}, zigUnwrap(t, page.LookupGrapheme(page.GetRowAndCell(0, 2).cell)))
	h := page.GetRowAndCell(1, 2).cell
	expect(t, page.styles.Get(h.style_id).flags.bold, "page.styles.get(page.memory, h.style_id).flags.bold")
	expect(t, page.GetRowAndCell(1, 2).row.styled, "page.getRowAndCell(1, 2).row.styled")
	l := page.GetRowAndCell(2, 2).cell
	e := page.hyperlink_set.Get(zigUnwrap(t, page.LookupHyperlink(l)))
	expectEqualStrings(t, "https://l", string(e.uri))
	// The tracked pin follows to the new page.
	expectEqual(t, s.pages.first, tpin.node)
	expectEqual(t, SizeCellCountInt(1), tpin.y)
	expectEqual(t, SizeCellCountInt(2), tpin.x)
	page.AssertIntegrity()
}

// Zig: test "term51: resizeWithoutReflowGrowCols: layout columns beyond a standard page use a custom capacity"
func TestPl4_term51_resizeWithoutReflowGrowCols_layout_columns_beyond_a_standard_page_use_a_custom_capacity(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 3, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|abcd|\n|e...|\n|....|")

	cols := zigUnwrap(t, PageStdCapacity.MaxCols()) + 1
	zigTry(t, s.resizeWithoutReflowGrowCols(cols, pl4KFirstChunk(t, s)))
	expectEqual(t, uint(1), s.TotalPages())
	page := &s.pages.first.data
	expectEqual(t, cols, page.capacity.cols)
	// rows = min(rows of the page, capacity rows).
	expectEqual(t, SizeCellCountInt(3), page.capacity.rows)
	expectEqual(t, SizeCellCountInt(3), page.size.rows)
	expectEqual(t, uint32('d'), pl4KCell(t, s, 3, 0).content.codepoint)
	expectEqual(t, uint32('e'), pl4KCell(t, s, 0, 1).content.codepoint)
	expectEqual(t, uint32(0), pl4KCell(t, s, 4, 0).content.codepoint)
	expectEqual(t, uint(3), s.totalRows())
	expect(t, pageListMemoryLen(page) > pageListStdSize(), "page.memory.len > std_size")
}

// Zig: test "term51: resizeWithoutReflowGrowCols: layout a page with too many rows is split and pins follow"
func TestPl4_term51_resizeWithoutReflowGrowCols_layout_a_page_with_too_many_rows_is_split_and_pins_follow(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 1000, 3, pl4NoMax())
	defer s.Deinit()
	zigTry(t, s.growRows(40))
	expectEqual(t, uint(1), s.TotalPages())
	for y := uint(0); y < 43; y++ {
		*pl4KPin(t, s, 0, y).RowAndCell().cell = PageCellInit('a' + uint32(y%26))
	}
	ys := []uint{0, 22, 23, 42}
	var pins [4]*PageListPin
	for i, y := range ys {
		pins[i] = pl4TrackPin(t, s, pl4KPin(t, s, 0, y))
	}
	defer func() {
		for _, p := range pins {
			s.UntrackPin(p)
		}
	}()

	// 2000 columns fit 23 rows per page.
	cap, err := PageStdCapacity.Adjust(PageCapacityAdjustment{cols: Some[SizeCellCountInt](2000)})
	zigTry(t, err)
	expectEqual(t, SizeCellCountInt(23), cap.rows)
	zigTry(t, s.resizeWithoutReflowGrowCols(2000, pl4KFirstChunk(t, s)))
	expectEqualSlices(t, []uint{23, 20}, pl4KRowsPerPage(s))
	expectEqual(t, uint(43), s.totalRows())
	for i, y := range ys {
		pl4KExpectPin(t, s, *pins[i], 0, y)
		expectEqual(t, 'a'+uint32(y%26), pins[i].RowAndCell().cell.content.codepoint)
	}
	expectEqual(t, s.pages.first, pins[0].node)
	expectEqual(t, s.pages.first, pins[1].node)
	expectEqual(t, s.pages.last, pins[2].node)
	expectEqual(t, SizeCellCountInt(0), pins[2].y)
	expectEqual(t, SizeCellCountInt(19), pins[3].y)
	for y := uint(0); y < 43; y++ {
		expectEqual(t, 'a'+uint32(y%26), pl4KCell(t, s, 0, y).content.codepoint)
	}
	s.AssertIntegrity()
}

// Zig: test "term51: resizeWithoutReflowGrowCols: layout rows are backfilled into the previous page when it has room"
func TestPl4_term51_resizeWithoutReflowGrowCols_layout_rows_are_backfilled_into_the_previous_page_when_it_has_room(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 1000, 3, pl4NoMax())
	defer s.Deinit()
	zigTry(t, s.growRows(60))
	expectEqualSlices(t, []uint{46, 17}, pl4KRowsPerPage(s))
	for y := uint(0); y < 63; y++ {
		*pl4KPin(t, s, 0, y).RowAndCell().cell = PageCellInit('a' + uint32(y%26))
	}
	// Free 10 rows in the first page.
	s.EraseRows(Point{Tag: PointTagHistory}, Some(pl4HistoryPt(0, 9)))
	expectEqualSlices(t, []uint{36, 17}, pl4KRowsPerPage(s))
	expectEqual(t, uint(53), s.totalRows())

	// Rows 36 and 45 are in the first 10 rows of the second page (copied to
	// the end of the previous page), rows 46 and 52 are in its remainder.
	ys := []uint{36, 45, 46, 52}
	var pins [4]*PageListPin
	for i, y := range ys {
		pins[i] = pl4TrackPin(t, s, pl4KPin(t, s, 0, y))
	}
	defer func() {
		for _, p := range pins {
			s.UntrackPin(p)
		}
	}()

	zigTry(t, s.resizeWithoutReflow(PageListResize{cols: pl4S(2000), reflow: false}))
	// First page makes pages of 23 and 13 rows, the second one fills the 10
	// missing rows of the last of those and keeps the other 7 in a new page.
	expectEqualSlices(t, []uint{23, 23, 7}, pl4KRowsPerPage(s))
	expectEqual(t, uint(53), s.totalRows())
	expectEqual(t, uint(53), s.total_rows)
	for i, y := range ys {
		pl4KExpectPin(t, s, *pins[i], 0, y)
		expectEqual(t, 'a'+uint32((y+10)%26), pins[i].RowAndCell().cell.content.codepoint)
	}
	expectEqual(t, s.pages.first.next, pins[0].node)
	expectEqual(t, SizeCellCountInt(13), pins[0].y)
	expectEqual(t, SizeCellCountInt(22), pins[1].y)
	expectEqual(t, s.pages.last, pins[2].node)
	expectEqual(t, SizeCellCountInt(0), pins[2].y)
	expectEqual(t, SizeCellCountInt(6), pins[3].y)
	for y := uint(0); y < 53; y++ {
		expectEqual(t, 'a'+uint32((y+10)%26), pl4KCell(t, s, 0, y).content.codepoint)
	}
	s.AssertIntegrity()
}

// Zig: test "term51: resizeWithoutReflowGrowCols: layout rows are not backfilled into a full previous page"
func TestPl4_term51_resizeWithoutReflowGrowCols_layout_rows_are_not_backfilled_into_a_full_previous_page(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 1000, 3, pl4NoMax())
	defer s.Deinit()
	zigTry(t, s.growRows(60))
	expectEqualSlices(t, []uint{46, 17}, pl4KRowsPerPage(s))

	zigTry(t, s.resizeWithoutReflow(PageListResize{cols: pl4S(2000), reflow: false}))
	// The 46 rows make two full pages of 23 and the 17 rows start a new one.
	expectEqualSlices(t, []uint{23, 23, 17}, pl4KRowsPerPage(s))
	expectEqual(t, uint(63), s.totalRows())
	s.AssertIntegrity()
}

// Zig: test "term51: resizeWithoutReflowGrowCols: a failed backfill falls back to new pages"
func TestPl4_term51_resizeWithoutReflowGrowCols_a_failed_backfill_falls_back_to_new_pages(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 200, 1, pl4NoMax())
	defer s.Deinit()
	pl4KGrowColsFixture(t, s, [3]bool{true, false, false})
	tpin := pl4TrackPin(t, s, PageListPin{node: s.pages.last, x: 0, y: 2})
	defer s.UntrackPin(tpin)

	zigTry(t, s.resizeWithoutReflow(PageListResize{cols: pl4S(201), reflow: false}))
	// The very first row cannot be copied to the previous page (the style set
	// is full) so nothing is copied there.
	expectEqualSlices(t, []uint{1, 3}, pl4KRowsPerPage(s))
	expectEqual(t, uint(4), s.totalRows())
	expectEqual(t, uint32('a'), pl4KCell(t, s, 0, 1).content.codepoint)
	expectEqual(t, uint32('c'), pl4KCell(t, s, 0, 3).content.codepoint)
	expect(t, pl4KCell(t, s, 0, 1).style_id != 0, "term51KCell(&s, 0, 1).style_id != 0")
	expectEqual(t, s.pages.last, tpin.node)
	expectEqual(t, SizeCellCountInt(2), tpin.y)
	s.AssertIntegrity()
}

// Zig: test "term51: resizeWithoutReflowGrowCols: a backfill that fails midway leaves pins in the copied rows dangling"
func TestPl4_term51_resizeWithoutReflowGrowCols_a_backfill_that_fails_midway_leaves_pins_in_the_copied_rows_dangling(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 200, 1, pl4NoMax())
	defer s.Deinit()
	pl4KGrowColsFixture(t, s, [3]bool{false, true, false})
	old_node := s.pages.last
	// One pin in the row that is copied to the previous page and one in a
	// row that is not.
	copied_pin := pl4TrackPin(t, s, PageListPin{node: old_node, x: 0, y: 0})
	defer s.UntrackPin(copied_pin)
	other_pin := pl4TrackPin(t, s, PageListPin{node: old_node, x: 0, y: 2})
	defer s.UntrackPin(other_pin)

	zigTry(t, s.resizeWithoutReflow(PageListResize{cols: pl4S(201), reflow: false}))
	// Row 0 went to the previous page, rows 1 and 2 to a new one.
	expectEqualSlices(t, []uint{2, 2}, pl4KRowsPerPage(s))
	expectEqual(t, uint(4), s.totalRows())
	expectEqual(t, uint32('a'), pl4KCell(t, s, 0, 1).content.codepoint)
	expectEqual(t, uint32('b'), pl4KCell(t, s, 0, 2).content.codepoint)
	expectEqual(t, uint32('c'), pl4KCell(t, s, 0, 3).content.codepoint)
	expectEqual(t, s.pages.last, other_pin.node)
	expectEqual(t, SizeCellCountInt(1), other_pin.y)

	// The pin in the copied row was not remapped when the copy stopped early
	// (the remap is skipped) so it still points to the destroyed page.
	expectEqual(t, old_node, copied_pin.node)
	found := false
	for node := s.pages.first; node != nil; node = node.next {
		if node == copied_pin.node {
			found = true
		}
	}
	expect(t, !found, "!found")

	// Repair it so that the list can be torn down.
	copied_pin.node = s.pages.first
	copied_pin.y = 1
}

// Zig: test "term51: trailingBlankLines: counts blank rows from the bottom up to max"
func TestPl4_term51_trailingBlankLines_counts_blank_rows_from_the_bottom_up_to_max(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 2, 5, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|a.|\n|b.|\n|..|\n|..|\n|..|")

	expectEqual(t, SizeCellCountInt(3), s.trailingBlankLines(10))
	expectEqual(t, SizeCellCountInt(3), s.trailingBlankLines(4))
	expectEqual(t, SizeCellCountInt(3), s.trailingBlankLines(3))
	expectEqual(t, SizeCellCountInt(2), s.trailingBlankLines(2))
	expectEqual(t, SizeCellCountInt(1), s.trailingBlankLines(1))
	// The count is incremented before it is compared with max so a max of 0
	// still counts one row.
	expectEqual(t, SizeCellCountInt(1), s.trailingBlankLines(0))
}

// Zig: test "term51: trailingBlankLines: text on the last row means none"
func TestPl4_term51_trailingBlankLines_text_on_the_last_row_means_none(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 2, 3, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|..|\n|..|\n|.x|")

	expectEqual(t, SizeCellCountInt(0), s.trailingBlankLines(10))
	expectEqual(t, SizeCellCountInt(0), s.trailingBlankLines(0))
}

// Zig: test "term51: trailingBlankLines: an entirely blank list is limited by its rows"
func TestPl4_term51_trailingBlankLines_an_entirely_blank_list_is_limited_by_its_rows(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 2, 3, pl4Max(0))
	defer s.Deinit()
	expectEqual(t, SizeCellCountInt(3), s.trailingBlankLines(3))
	expectEqual(t, SizeCellCountInt(3), s.trailingBlankLines(100))
	expectEqual(t, SizeCellCountInt(3), s.trailingBlankLines(math.MaxUint16))
}

// Zig: test "term51: trailingBlankLines: rows with only colors, styles or spacers are blank"
func TestPl4_term51_trailingBlankLines_rows_with_only_colors_styles_or_spacers_are_blank(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 3, 5, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|a..|\n|...|\n|...|\n|...|\n|...|")
	{
		page := &s.pages.first.data
		*page.GetRowAndCell(0, 1).cell = PageCell{content_tag: PageCellContentTagBgColorPalette, content: PageCellContent{color_palette: 1}}
		*page.GetRowAndCell(1, 2).cell = PageCell{content_tag: PageCellContentTagBgColorRgb, content: PageCellContent{color_rgb: PageCellRGB{r: 1, g: 2, b: 3}}}
		bold := pl4StylesAdd(t, page, pl4Bold())
		st := page.GetRowAndCell(2, 3)
		st.cell.style_id = bold
		st.row.styled = true
		*page.GetRowAndCell(2, 4).cell = PageCell{content_tag: PageCellContentTagCodepoint, content: PageCellContent{codepoint: 0}, wide: PageCellWideSpacerTail}
	}
	expectEqual(t, SizeCellCountInt(4), s.trailingBlankLines(10))
}

// Zig: test "term51: trailingBlankLines: text above stops the count and a grapheme counts as text"
func TestPl4_term51_trailingBlankLines_text_above_stops_the_count_and_a_grapheme_counts_as_text(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 2, 4, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|..|\n|a.|\n|..|\n|..|")
	expectEqual(t, SizeCellCountInt(2), s.trailingBlankLines(10))

	page := &s.pages.first.data
	c := page.GetRowAndCell(1, 2)
	*c.cell = PageCellInit('x')
	zigTry(t, page.AppendGrapheme(c.row, c.cell, 0x301))
	expectEqual(t, SizeCellCountInt(1), s.trailingBlankLines(10))
}

// Zig: test "term51: trailingBlankLines: layout counts across pages"
func TestPl4_term51_trailingBlankLines_layout_counts_across_pages(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 1000, 3, pl4NoMax())
	defer s.Deinit()
	zigTry(t, s.growRows(60))
	expectEqualSlices(t, []uint{46, 17}, pl4KRowsPerPage(s))

	// Text in the first page.
	*pl4KPin(t, s, 0, 40).RowAndCell().cell = PageCellInit('x')
	expectEqual(t, SizeCellCountInt(22), s.trailingBlankLines(100))
	expectEqual(t, SizeCellCountInt(20), s.trailingBlankLines(20))
	expectEqual(t, SizeCellCountInt(17), s.trailingBlankLines(17))
	expectEqual(t, SizeCellCountInt(18), s.trailingBlankLines(18))
	// Text in the last page.
	*pl4KPin(t, s, 5, 55).RowAndCell().cell = PageCellInit('y')
	expectEqual(t, SizeCellCountInt(7), s.trailingBlankLines(100))
}

// Zig: test "term51: trimTrailingBlankRows: removes blank rows up to max"
func TestPl4_term51_trimTrailingBlankRows_removes_blank_rows_up_to_max(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 2, 5, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|a.|\n|b.|\n|..|\n|..|\n|..|")

	expectEqual(t, SizeCellCountInt(2), s.trimTrailingBlankRows(2))
	// The caller owns the row cache.
	expectEqual(t, uint(5), s.total_rows)
	s.total_rows -= 2
	pl4KExpect(t, s, "|a.|\n|b.|\n|..|")

	expectEqual(t, SizeCellCountInt(1), s.trimTrailingBlankRows(1))
	s.total_rows -= 1
	pl4KExpect(t, s, "|a.|\n|b.|")
}

// Zig: test "term51: trimTrailingBlankRows: stops at text even when max allows more"
func TestPl4_term51_trimTrailingBlankRows_stops_at_text_even_when_max_allows_more(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 2, 5, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|a.|\n|b.|\n|..|\n|..|\n|..|")

	expectEqual(t, SizeCellCountInt(3), s.trimTrailingBlankRows(100))
	s.total_rows -= 3
	pl4KExpect(t, s, "|a.|\n|b.|")
	// Nothing more to trim.
	expectEqual(t, SizeCellCountInt(0), s.trimTrailingBlankRows(100))
	expectEqual(t, SizeCellCountInt(0), s.trimTrailingBlankRows(1))
}

// Zig: test "term51: trimTrailingBlankRows: max zero still trims one row"
func TestPl4_term51_trimTrailingBlankRows_max_zero_still_trims_one_row(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 2, 3, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|a.|\n|..|\n|..|")

	// trimmed is incremented before it is compared with max.
	expectEqual(t, SizeCellCountInt(1), s.trimTrailingBlankRows(0))
	s.total_rows -= 1
	pl4KExpect(t, s, "|a.|\n|..|")
}

// Zig: test "term51: trimTrailingBlankRows: a tracked pin protects its row"
func TestPl4_term51_trimTrailingBlankRows_a_tracked_pin_protects_its_row(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 2, 5, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|a.|\n|b.|\n|..|\n|..|\n|..|")
	p := pl4TrackPin(t, s, pl4KPin(t, s, 1, 3))
	defer s.UntrackPin(p)

	expectEqual(t, SizeCellCountInt(1), s.trimTrailingBlankRows(10))
	s.total_rows -= 1
	pl4KExpect(t, s, "|a.|\n|b.|\n|..|\n|..|")
	// A pin on the last row stops everything.
	p2 := pl4TrackPin(t, s, pl4KPin(t, s, 0, 3))
	defer s.UntrackPin(p2)
	expectEqual(t, SizeCellCountInt(0), s.trimTrailingBlankRows(10))
}

// Zig: test "term51: trimTrailingBlankRows: the viewport pin on the first row protects it"
func TestPl4_term51_trimTrailingBlankRows_the_viewport_pin_on_the_first_row_protects_it(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 2, 4, pl4Max(0))
	defer s.Deinit()

	// Entirely blank: the viewport pin tracks row 0.
	expectEqual(t, SizeCellCountInt(3), s.trimTrailingBlankRows(10))
	s.total_rows -= 3
	pl4KExpect(t, s, "|..|")
}

// Zig: test "term51: trimTrailingBlankRows: layout removes pages that become empty"
func TestPl4_term51_trimTrailingBlankRows_layout_removes_pages_that_become_empty(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 1000, 3, pl4NoMax())
	defer s.Deinit()
	zigTry(t, s.growRows(60))
	*pl4KPin(t, s, 0, 10).RowAndCell().cell = PageCellInit('x')
	expectEqualSlices(t, []uint{46, 17}, pl4KRowsPerPage(s))
	first := s.pages.first

	// The whole last page and 3 rows of the first one.
	expectEqual(t, SizeCellCountInt(20), s.trimTrailingBlankRows(20))
	s.total_rows -= 20
	expectEqual(t, uint(1), s.TotalPages())
	expectEqual(t, first, s.pages.last)
	expectEqual(t, SizeCellCountInt(43), first.data.size.rows)
	s.AssertIntegrity()

	// And the rest of the blank rows (the pin is on row 0).
	expectEqual(t, SizeCellCountInt(32), s.trimTrailingBlankRows(100))
	s.total_rows -= 32
	expectEqual(t, uint(11), s.totalRows())
	s.AssertIntegrity()
}

// Zig: test "term51: trimTrailingBlankRows: layout a pin in the last page keeps it and moves nothing"
func TestPl4_term51_trimTrailingBlankRows_layout_a_pin_in_the_last_page_keeps_it_and_moves_nothing(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 1000, 3, pl4NoMax())
	defer s.Deinit()
	zigTry(t, s.growRows(60))
	*pl4KPin(t, s, 0, 10).RowAndCell().cell = PageCellInit('x')
	p := pl4TrackPin(t, s, pl4KPin(t, s, 0, 51))
	defer s.UntrackPin(p)

	// Rows 62..52 are trimmed, row 51 holds the pin.
	expectEqual(t, SizeCellCountInt(11), s.trimTrailingBlankRows(100))
	s.total_rows -= 11
	expectEqualSlices(t, []uint{46, 6}, pl4KRowsPerPage(s))
	pl4KExpectPin(t, s, *p, 0, 51)
	s.AssertIntegrity()
}

// Zig: test "term51: trimTrailingBlankRows: layout only the last page is erased"
func TestPl4_term51_trimTrailingBlankRows_layout_only_the_last_page_is_erased(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 1000, 3, pl4NoMax())
	defer s.Deinit()
	zigTry(t, s.growRows(60))
	*pl4KPin(t, s, 0, 10).RowAndCell().cell = PageCellInit('x')

	// Exactly the 17 rows of the last page.
	expectEqual(t, SizeCellCountInt(17), s.trimTrailingBlankRows(17))
	s.total_rows -= 17
	expectEqualSlices(t, []uint{46}, pl4KRowsPerPage(s))
	s.AssertIntegrity()
}

// Zig: test "term51: trimTrailingBlankRows: rows with colors have no text and are trimmed but come back with grow"
func TestPl4_term51_trimTrailingBlankRows_rows_with_colors_have_no_text_and_are_trimmed_but_come_back_with_grow(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 2, 3, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|a.|\n|..|\n|..|")
	*pl4KPin(t, s, 1, 2).RowAndCell().cell = PageCell{
		content_tag: PageCellContentTagBgColorPalette,
		content:     PageCellContent{color_palette: 5},
	}

	expectEqual(t, SizeCellCountInt(2), s.trimTrailingBlankRows(10))
	s.total_rows -= 2
	pl4KExpect(t, s, "|a.|")

	// The trimmed row memory is not cleared: growing resurrects the color.
	pl4Grow(t, s)
	pl4Grow(t, s)
	pl4KExpect(t, s, "|a.|\n|..|\n|.#|")
}

// Zig: test "term51: resizeCols: layout hyperlinks that need more string bytes are kept when unwrapping across pages"
func TestPl4_term51_resizeCols_layout_hyperlinks_that_need_more_string_bytes_are_kept_when_unwrapping_across_pages(t *testing.T) {
	testStdCapacity(t)
	s := pl4KTwoPages(t, 2, 10)
	defer s.Deinit()
	total := s.totalRows()
	uri_a := strings.Repeat("a", PageStringBytesDefault-1)
	uri_b := strings.Repeat("b", PageStringBytesDefault-1)
	{
		page := &s.pages.first.data
		id := pl4InsertHyperlink(t, page, Hyperlink{
			id:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: 5},
			uri: []uint8(uri_a),
		})
		rac := page.GetRowAndCell(uint(page.size.cols)-1, uint(page.size.rows)-1)
		rac.row.wrap = true
		*rac.cell = PageCellInit('X')
		zigTry(t, page.SetHyperlink(rac.row, rac.cell, id))
	}
	{
		page := &s.pages.last.data
		id := pl4InsertHyperlink(t, page, Hyperlink{
			id:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: 6},
			uri: []uint8(uri_b),
		})
		rac := page.GetRowAndCell(0, 0)
		rac.row.wrap_continuation = true
		*rac.cell = PageCellInit('Y')
		zigTry(t, page.SetHyperlink(rac.row, rac.cell, id))
	}
	old_string_bytes := s.pages.last.data.capacity.string_bytes

	zigTry(t, s.Resize(PageListResize{cols: pl4S(3), reflow: true}))
	s.AssertIntegrity()
	// The two rows became one.
	expectEqual(t, total-1, s.totalRows())
	expectEqual(t, uint(2), s.TotalPages())
	last_row := s.totalRows() - 1
	expectEqual(t, uint32('X'), pl4KCell(t, s, 1, last_row).content.codepoint)
	expectEqual(t, uint32('Y'), pl4KCell(t, s, 2, last_row).content.codepoint)
	expect(t, !pl4KRow(t, s, last_row).wrap, "!term51KRow(&s, last_row).wrap")
	expect(t, !pl4KRow(t, s, last_row).wrap_continuation, "!term51KRow(&s, last_row).wrap_continuation")

	// Both links are in the last page whose string memory was increased.
	page := &s.pages.last.data
	expect(t, page.capacity.string_bytes > old_string_bytes, "page.capacity.string_bytes > old_string_bytes")
	expectEqual(t, uint(2), page.HyperlinkCount())
	p1 := pl4KPin(t, s, 1, last_row)
	e1 := page.hyperlink_set.Get(zigUnwrap(t, page.LookupHyperlink(p1.RowAndCell().cell)))
	expectEqualStrings(t, uri_a, string(e1.uri))
	expectEqual(t, uint(5), uint(e1.id.Implicit))
	p2 := pl4KPin(t, s, 2, last_row)
	e2 := page.hyperlink_set.Get(zigUnwrap(t, page.LookupHyperlink(p2.RowAndCell().cell)))
	expectEqualStrings(t, uri_b, string(e2.uri))
	expectEqual(t, uint(6), uint(e2.id.Implicit))
	// The other page has none.
	expectEqual(t, uint(0), s.pages.first.data.HyperlinkCount())
}

// Zig: test "term51: resizeCols: layout graphemes that need more bytes are kept when unwrapping across pages"
func TestPl4_term51_resizeCols_layout_graphemes_that_need_more_bytes_are_kept_when_unwrapping_across_pages(t *testing.T) {
	testStdCapacity(t)
	s := pl4KTwoPages(t, 2, 10)
	defer s.Deinit()
	total := s.totalRows()
	n := (PageGraphemeBytesDefault - 1) / 4 // @divFloor(grapheme_bytes_default - 1, @sizeOf(u21))
	cps_a := make([]uint32, n)
	cps_b := make([]uint32, n)
	for i := range cps_a {
		cps_a[i] = 'a'
		cps_b[i] = 'b'
	}
	{
		page := &s.pages.first.data
		rac := page.GetRowAndCell(uint(page.size.cols)-1, uint(page.size.rows)-1)
		rac.row.wrap = true
		*rac.cell = PageCellInit('X')
		zigTry(t, page.SetGraphemes(rac.row, rac.cell, cps_a))
	}
	{
		page := &s.pages.last.data
		rac := page.GetRowAndCell(0, 0)
		rac.row.wrap_continuation = true
		*rac.cell = PageCellInit('Y')
		zigTry(t, page.SetGraphemes(rac.row, rac.cell, cps_b))
	}
	old_bytes := s.pages.last.data.capacity.grapheme_bytes

	zigTry(t, s.Resize(PageListResize{cols: pl4S(3), reflow: true}))
	s.AssertIntegrity()
	expectEqual(t, total-1, s.totalRows())
	expectEqual(t, uint(2), s.TotalPages())
	last_row := s.totalRows() - 1
	page := &s.pages.last.data
	expect(t, page.capacity.grapheme_bytes > old_bytes, "page.capacity.grapheme_bytes > old_bytes")
	expectEqual(t, uint(2), page.GraphemeCount())
	expect(t, pl4KRow(t, s, last_row).grapheme, "term51KRow(&s, last_row).grapheme")
	x := pl4KPin(t, s, 1, last_row).RowAndCell().cell
	expectEqual(t, uint32('X'), x.content.codepoint)
	expectEqualSlices(t, cps_a, zigUnwrap(t, page.LookupGrapheme(x)))
	y := pl4KPin(t, s, 2, last_row).RowAndCell().cell
	expectEqual(t, uint32('Y'), y.content.codepoint)
	expectEqualSlices(t, cps_b, zigUnwrap(t, page.LookupGrapheme(y)))
	expectEqual(t, uint(0), s.pages.first.data.GraphemeCount())
}

// Zig: test "term51: resizeCols: layout styles that need a larger set are kept when unwrapping across pages"
func TestPl4_term51_resizeCols_layout_styles_that_need_a_larger_set_are_kept_when_unwrapping_across_pages(t *testing.T) {
	testStdCapacity(t)
	s := pl4KTwoPages(t, PageStdCapacity.styles-1, 10)
	defer s.Deinit()
	cols := uint(s.cols)
	n1 := uint(0)
	{
		page := &s.pages.first.data
		for x := uint(0); x < cols; x++ {
			id, err := page.styles.Add(Style{bg_color: StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{
				r: uint8(x),
				g: uint8(x >> 8),
				b: 0,
			}}})
			if err != nil {
				break
			}
			rac := page.GetRowAndCell(x, uint(page.size.rows)-1)
			rac.row.wrap = true
			rac.row.styled = true
			*rac.cell = PageCell{content_tag: PageCellContentTagCodepoint, content: PageCellContent{codepoint: 'X'}, style_id: id}
			n1 += 1
		}
	}
	n2 := uint(0)
	{
		page := &s.pages.last.data
		for x := uint(0); x < cols; x++ {
			id, err := page.styles.Add(Style{fg_color: StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{
				r: uint8(x),
				g: uint8(x >> 8),
				b: 0,
			}}})
			if err != nil {
				break
			}
			rac := page.GetRowAndCell(x, 0)
			rac.row.wrap_continuation = true
			rac.row.styled = true
			*rac.cell = PageCell{content_tag: PageCellContentTagCodepoint, content: PageCellContent{codepoint: 'X'}, style_id: id}
			n2 += 1
		}
	}
	// Both sets are full before the last row of the first page was written
	// completely and the total does not fit in one page.
	expect(t, n1 > 0 && n1 < cols, "n1 > 0 and n1 < cols")
	expect(t, n1+n2 > uint(PageStdCapacity.styles), "n1 + n2 > std_capacity.styles")

	zigTry(t, s.Resize(PageListResize{cols: pl4S(SizeCellCountInt(cols * 2)), reflow: true}))
	s.AssertIntegrity()
	page := &s.pages.last.data
	expect(t, page.capacity.styles > PageStdCapacity.styles, "page.capacity.styles > std_capacity.styles")
	expectEqual(t, n1+n2, page.styles.living)
	last_row := s.totalRows() - 1
	expect(t, pl4KRow(t, s, last_row).styled, "term51KRow(&s, last_row).styled")
	// First part has the bg styles, the second the fg ones.
	a := *pl4KPin(t, s, 0, last_row).RowAndCell().cell
	expect(t, page.styles.Get(a.style_id).bg_color.Tag == StyleColorRgb, "page.styles.get(page.memory, a.style_id).bg_color == .rgb")
	b := *pl4KPin(t, s, cols, last_row).RowAndCell().cell
	expect(t, page.styles.Get(b.style_id).fg_color.Tag == StyleColorRgb, "page.styles.get(page.memory, b.style_id).fg_color == .rgb")
	gap := *pl4KPin(t, s, n1, last_row).RowAndCell().cell
	expectEqual(t, SizeStyleCountInt(0), gap.style_id)
}

// Zig: test "term51: resizeCols: layout a line wrapped across two pages is joined and pins in both halves follow"
func TestPl4_term51_resizeCols_layout_a_line_wrapped_across_two_pages_is_joined_and_pins_in_both_halves_follow(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 1000, 3, pl4NoMax())
	defer s.Deinit()
	zigTry(t, s.growRows(60))
	expectEqualSlices(t, []uint{46, 17}, pl4KRowsPerPage(s))
	// Screen row 45 is the last row of the first page and wraps into the first
	// row of the second page.
	for y := uint(0); y < 45; y++ {
		*pl4KPin(t, s, 0, y).RowAndCell().cell = PageCellInit('n')
	}
	{
		a := pl4KPin(t, s, 0, 45).RowAndCell()
		a.row.wrap = true
		*a.cell = PageCellInit('p')
		*pl4KPin(t, s, 999, 45).RowAndCell().cell = PageCellInit('q')
		b := pl4KPin(t, s, 0, 46).RowAndCell()
		b.row.wrap_continuation = true
		*b.cell = PageCellInit('r')
	}
	pin_a := pl4TrackPin(t, s, pl4KPin(t, s, 999, 45))
	pin_b := pl4TrackPin(t, s, pl4KPin(t, s, 0, 46))
	pin_blank := pl4TrackPin(t, s, pl4KPin(t, s, 500, 46))
	defer s.UntrackPin(pin_a)
	defer s.UntrackPin(pin_b)
	defer s.UntrackPin(pin_blank)

	zigTry(t, s.Resize(PageListResize{cols: pl4S(2000), reflow: true}))
	s.AssertIntegrity()
	// Rows 0-44 and the joined row; the blank rows after it were never written.
	expectEqual(t, uint(46), s.totalRows())
	expectEqual(t, uint32('p'), pl4KCell(t, s, 0, 45).content.codepoint)
	expectEqual(t, uint32('q'), pl4KCell(t, s, 999, 45).content.codepoint)
	expectEqual(t, uint32('r'), pl4KCell(t, s, 1000, 45).content.codepoint)
	expect(t, !pl4KRow(t, s, 45).wrap, "!term51KRow(&s, 45).wrap")
	pl4KExpectPin(t, s, *pin_a, 999, 45)
	pl4KExpectPin(t, s, *pin_b, 1000, 45)
	// The pin in the blank part of the continuation row keeps its offset
	// from the start of that row and the row is extended to include it.
	pl4KExpectPin(t, s, *pin_blank, 1500, 45)
}

// Zig: test "term51: resize: nothing changes when the dimensions are the same"
func TestPl4_term51_resize_nothing_changes_when_the_dimensions_are_the_same(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 3, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|abcd>\n<ef..|\n|g...|")
	s.viewport_pin_row_offset = Some[uint](3)

	zigTry(t, s.Resize(PageListResize{reflow: true}))
	zigTry(t, s.Resize(PageListResize{cols: pl4S(4), rows: pl4S(3), reflow: true}))
	zigTry(t, s.Resize(PageListResize{cols: pl4S(4), rows: pl4S(3), reflow: false, cursor: pl4Cursor(1, 1)}))
	pl4KExpect(t, s, "|abcd>\n<ef..|\n|g...|")
	expectEqual(t, None[uint](), s.viewport_pin_row_offset)
}

// Zig: test "term51: resize: a single column unwraps when it grows"
func TestPl4_term51_resize_a_single_column_unwraps_when_it_grows(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 1, 4, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|a>\n<b>\n<c|\n|d|")

	zigTry(t, s.Resize(PageListResize{cols: pl4S(3), reflow: true}))
	pl4KExpect(t, s, "|abc|\n|d..|\n|...|\n|...|")
	// The active area (4 rows) is made up with blank rows.
	expectEqual(t, uint(4), uint(s.rows))
}

// Zig: test "term51: resize: a wide character at the edge round trips through a narrower width"
func TestPl4_term51_resize_a_wide_character_at_the_edge_round_trips_through_a_narrower_width(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 2, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|abW_|\n|....|")

	zigTry(t, s.Resize(PageListResize{cols: pl4S(3), reflow: true}))
	pl4KExpect(t, s, "|ab^>\n<W_.|")
	zigTry(t, s.Resize(PageListResize{cols: pl4S(4), reflow: true}))
	pl4KExpect(t, s, "|abW_|\n|....|")
}

// Zig: test "term51: resize: shrinking to one column destroys wide characters for good"
func TestPl4_term51_resize_shrinking_to_one_column_destroys_wide_characters_for_good(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 2, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|aW_b|\n|....|")

	zigTry(t, s.Resize(PageListResize{cols: pl4S(1), reflow: true}))
	pl4KExpect(t, s, "|a>\n<.>\n<b|")
	zigTry(t, s.Resize(PageListResize{cols: pl4S(4), reflow: true}))
	// The wide character became one empty cell.
	pl4KExpect(t, s, "|a.b.|\n|....|")
}

// Zig: test "term51: ReflowCursor.reflowRow: a blank wrapped row is written in full"
func TestPl4_term51_ReflowCursor_reflowRow_a_blank_wrapped_row_is_written_in_full(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 3, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|....>\n<a...|\n|....|")

	r := pl4KReflowInit(t, s, 4, None[PageCapacity]())
	r.run()
	r.finish()
	pl4KExpect(t, s, "|....>\n<a...|")

	s2 := pl4Init(t, 4, 2, pl4Max(0))
	defer s2.Deinit()
	pl4KLoad(t, s2, "|....>\n<....|")
	// Unwrapping keeps the blanks of the wrapped row, the blank
	// continuation is dropped.
	r2 := pl4KReflowInit(t, s2, 8, None[PageCapacity]())
	r2.run()
	r2.finish()
	pl4KExpect(t, s2, "|........|")
}

// Zig: test "term51: ReflowCursor.reflowRow: wrap flags are rebuilt from the content"
func TestPl4_term51_ReflowCursor_reflowRow_wrap_flags_are_rebuilt_from_the_content(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 2, pl4Max(0))
	defer s.Deinit()
	// The first row is a continuation of a row that is not there and the last
	// row says it wraps but nothing follows it.
	pl4KLoad(t, s, "<abcd|\n|efgh>")

	r := pl4KReflowInit(t, s, 8, None[PageCapacity]())
	r.run()
	r.finish()
	pl4KExpect(t, s, "|abcd....|\n|efgh....|")
}

// Zig: test "term51: ReflowCursor.reflowRow: dirty flags are not carried over"
func TestPl4_term51_ReflowCursor_reflowRow_dirty_flags_are_not_carried_over(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 2, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|ab..|\n|cd..|")
	pl4KRow(t, s, 0).dirty = true
	pl4KRow(t, s, 1).dirty = true

	r := pl4KReflowInit(t, s, 3, None[PageCapacity]())
	r.run()
	r.finish()
	pl4KExpect(t, s, "|ab.|\n|cd.|")
	expect(t, !pl4KRow(t, s, 0).dirty, "!term51KRow(&s, 0).dirty")
	expect(t, !pl4KRow(t, s, 1).dirty, "!term51KRow(&s, 1).dirty")
}

// Zig: test "term51: resizeWithoutReflow: dirty flags are kept by column changes"
func TestPl4_term51_resizeWithoutReflow_dirty_flags_are_kept_by_column_changes(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 4, 2, pl4Max(0))
	defer s.Deinit()
	pl4KLoad(t, s, "|ab..|\n|cd..|")
	pl4KRow(t, s, 0).dirty = true

	// Less columns clear in place.
	zigTry(t, s.resizeWithoutReflow(PageListResize{cols: pl4S(2), reflow: true}))
	expect(t, pl4KRow(t, s, 0).dirty, "term51KRow(&s, 0).dirty")
	expect(t, !pl4KRow(t, s, 1).dirty, "!term51KRow(&s, 1).dirty")
	// More columns than the capacity clone the rows into a new page.
	zigTry(t, s.resizeWithoutReflow(PageListResize{cols: pl4S(9), reflow: true}))
	expect(t, pl4KRow(t, s, 0).dirty, "term51KRow(&s, 0).dirty")
	expect(t, !pl4KRow(t, s, 1).dirty, "!term51KRow(&s, 1).dirty")
}

// Zig: test "term51: scroll: explicit max size 0 forces active for every variant"
func TestPl4_term51_scroll_explicit_max_size_0_forces_active_for_every_variant(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 24, pl4Max(0))
	defer s.Deinit()
	pl4LGrowRows(t, s, 10)
	expectEqual(t, uint(34), s.total_rows)

	first := s.pages.first
	mid := pl4Pin(t, s, pl4ScreenPt(0, 3))
	cases := []PageListScroll{
		{Tag: PageListScrollActive},
		{Tag: PageListScrollTop},
		pl4ScrollRow(0),
		pl4ScrollRow(5),
		pl4ScrollDeltaRow(-3),
		pl4ScrollDeltaRow(3),
		pl4ScrollDeltaPrompt(-1),
		pl4ScrollDeltaPrompt(1),
		pl4ScrollPin(mid),
	}

	for _, c := range cases {
		// From a top viewport.
		s.viewport = pl4Viewport(PageListViewportTop)
		s.Scroll(c)
		expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

		// From a pin viewport with a (bogus) cached offset: nothing is
		// touched except the viewport kind.
		s.viewport = pl4Viewport(PageListViewportPin)
		s.viewport_pin_row_offset = Some[uint](7)
		s.Scroll(c)
		expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
		expectEqual(t, Some[uint](7), s.viewport_pin_row_offset)
		s.viewport_pin_row_offset = None[uint]()

		// The viewport pin itself is never moved.
		expectEqual(t, first, s.viewport_pin.node)
		expectEqual(t, SizeCellCountInt(0), s.viewport_pin.y)
		expectEqual(t, SizeCellCountInt(0), s.viewport_pin.x)
	}
}

// Zig: test "term51: scroll: active and top variants"
func TestPl4_term51_scroll_active_and_top_variants(t *testing.T) {
	testStdCapacity(t)
	s := pl4LInitPages(t, 100)
	defer s.Deinit()
	total := s.total_rows
	expectEqual(t, 2*uint(s.pages.first.data.capacity.rows)+100, total)

	// Already active: stays active.
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	expectEqual(t, total-uint(s.rows), s.viewportRowOffset())

	// Active to top.
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
	expectEqual(t, s.pages.first, s.GetTopLeft(PointTagViewport).node)
	expectEqual(t, uint(0), s.viewportRowOffset())

	// Top to top.
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")

	// Top to active.
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	// Pin to top: the cached offset is left alone (stale but unused).
	pl4LPinAt(t, s, 100)
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
	expectEqual(t, Some[uint](100), s.viewport_pin_row_offset)

	// Pin to active: likewise.
	pl4LPinAt(t, s, 100)
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	expectEqual(t, Some[uint](100), s.viewport_pin_row_offset)
}

// Zig: test "term51: scroll: pin variant"
func TestPl4_term51_scroll_pin_variant(t *testing.T) {
	testStdCapacity(t)
	s := pl4LInitPages(t, 100)
	defer s.Deinit()
	c := uint(s.pages.first.data.capacity.rows)
	second := s.pages.first.next
	third := s.pages.last
	// Active top-left is third page row 76 (absolute 2c + 76).
	expectEqual(t, third, s.GetTopLeft(PointTagActive).node)
	expectEqual(t, SizeCellCountInt(76), s.GetTopLeft(PointTagActive).y)

	// Active top row: active. The viewport pin is not touched.
	vp_before := *s.viewport_pin
	s.Scroll(pl4ScrollPin(PageListPin{node: third, y: 76}))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	expectEqual(t, vp_before.node, s.viewport_pin.node)
	expectEqual(t, vp_before.y, s.viewport_pin.y)

	// Last row, with an x: active.
	s.Scroll(pl4ScrollPin(PageListPin{node: third, y: 99, x: 9}))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	// One above the active top: a pin viewport; the offset cache is reset
	// and the pin is copied including x.
	s.viewport_pin_row_offset = Some[uint](42)
	s.Scroll(pl4ScrollPin(PageListPin{node: third, y: 75}))
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, third, s.viewport_pin.node)
	expectEqual(t, SizeCellCountInt(75), s.viewport_pin.y)
	expectEqual(t, SizeCellCountInt(0), s.viewport_pin.x)
	expectEqual(t, None[uint](), s.viewport_pin_row_offset)
	expectEqual(t, 2*c+75, s.Scrollbar().offset)

	with_x := PageListPin{node: third, y: 75, x: 9}
	s.Scroll(pl4ScrollPin(with_x))
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, SizeCellCountInt(9), s.viewport_pin.x)
	expectEqual(t, None[uint](), s.viewport_pin_row_offset)

	// A pin in the second page (even row 0): a pin viewport, not top.
	s.Scroll(pl4ScrollPin(PageListPin{node: second, y: 0}))
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, second, s.viewport_pin.node)
	expectEqual(t, SizeCellCountInt(0), s.viewport_pin.y)
	expectEqual(t, c, s.Scrollbar().offset)

	// First page, row 1: pin viewport.
	s.Scroll(pl4ScrollPin(PageListPin{node: s.pages.first, y: 1}))
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, uint(1), s.Scrollbar().offset)

	// First page row 0 (any x): top. The viewport pin is not modified.
	s.Scroll(pl4ScrollPin(PageListPin{node: s.pages.first, y: 1}))
	keep := *s.viewport_pin
	s.Scroll(pl4ScrollPin(PageListPin{node: s.pages.first, y: 0, x: 9}))
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
	expectEqual(t, keep.node, s.viewport_pin.node)
	expectEqual(t, keep.y, s.viewport_pin.y)
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	s.Scroll(pl4ScrollPin(PageListPin{node: s.pages.first, y: 0}))
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")

	// From a pin viewport: pin in active goes active, pin at top goes top.
	s.Scroll(pl4ScrollPin(PageListPin{node: third, y: 75}))
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	s.Scroll(pl4ScrollPin(PageListPin{node: third, y: 76}))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	s.Scroll(pl4ScrollPin(PageListPin{node: third, y: 75}))
	s.Scroll(pl4ScrollPin(PageListPin{node: s.pages.first, y: 0}))
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
}

// Zig: test "term51: scroll: pin variant with active area spanning two pages"
func TestPl4_term51_scroll_pin_variant_with_active_area_spanning_two_pages(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 24, pl4NoMax())
	defer s.Deinit()
	pl4LFillLast(t, s)
	pl4GrowNode(t, s)
	pl4LGrowRows(t, s, 4)
	first := s.pages.first
	second := s.pages.last
	c := uint(first.data.capacity.rows)
	expectEqual(t, SizeCellCountInt(5), second.data.size.rows)

	// Active is the last 19 rows of the first page and all 5 of the last.
	expectEqual(t, first, s.GetTopLeft(PointTagActive).node)
	expectEqual(t, SizeCellCountInt(c-19), s.GetTopLeft(PointTagActive).y)

	s.Scroll(pl4ScrollPin(PageListPin{node: first, y: SizeCellCountInt(c - 19)}))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	s.Scroll(pl4ScrollPin(PageListPin{node: first, y: SizeCellCountInt(c - 1)}))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	s.Scroll(pl4ScrollPin(PageListPin{node: second, y: 0}))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	s.Scroll(pl4ScrollPin(PageListPin{node: second, y: 4}))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	s.Scroll(pl4ScrollPin(PageListPin{node: first, y: SizeCellCountInt(c - 20)}))
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, c-20, s.Scrollbar().offset)
}

// Zig: test "term51: scroll: row variant top and active boundaries"
func TestPl4_term51_scroll_row_variant_top_and_active_boundaries(t *testing.T) {
	testStdCapacity(t)
	s := pl4LInitPages(t, 100)
	defer s.Deinit()
	total := s.total_rows
	expectEqual(t, uint(1258), total)

	// Row 0 is top, whatever we were before. The cache is untouched.
	pl4LPinAt(t, s, 300)
	s.Scroll(pl4ScrollRow(0))
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
	expectEqual(t, Some[uint](300), s.viewport_pin_row_offset)

	// total - rows is the active top and goes to active.
	s.Scroll(pl4ScrollRow(total - uint(s.rows)))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	// Beyond too, up to the largest usize.
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	s.Scroll(pl4ScrollRow(total - uint(s.rows) + 1))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	s.Scroll(pl4ScrollRow(math.MaxUint))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	// One before the active top is a real pin.
	s.Scroll(pl4ScrollRow(total - uint(s.rows) - 1))
	pl4LExpectViewportRow(t, s, total-uint(s.rows)-1, true)
	expectEqual(t, s.pages.last, s.viewport_pin.node)
	expectEqual(t, SizeCellCountInt(75), s.viewport_pin.y)

	// Row 1 is the first real pin.
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	s.Scroll(pl4ScrollRow(1))
	pl4LExpectViewportRow(t, s, 1, true)
	expectEqual(t, s.pages.first, s.viewport_pin.node)
	expectEqual(t, SizeCellCountInt(1), s.viewport_pin.y)
}

// Zig: test "term51: scroll: row variant without scrollback"
func TestPl4_term51_scroll_row_variant_without_scrollback(t *testing.T) {
	testStdCapacity(t)
	// When total_rows == rows there is no history but .row = 0 still
	// selects the top viewport (n == 0 is checked first).
	s := pl4Init(t, 80, 24, pl4NoMax())
	defer s.Deinit()
	expectEqual(t, uint(24), s.total_rows)

	s.Scroll(pl4ScrollRow(0))
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	s.Scroll(pl4ScrollRow(1))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	s.Scroll(pl4ScrollRow(23))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
}

// Zig: test "term51: scroll: row variant slow path walks pages from either end"
func TestPl4_term51_scroll_row_variant_slow_path_walks_pages_from_either_end(t *testing.T) {
	testStdCapacity(t)
	s := pl4LInitPages(t, 100)
	defer s.Deinit()
	c := uint(s.pages.first.data.capacity.rows)
	second := s.pages.first.next
	third := s.pages.last
	total := s.total_rows
	// The midpoint decides the direction: n < 629 walks forward from the
	// first page, otherwise backward from the last.
	expectEqual(t, uint(629), total/2)

	rows := []uint{1, 100, c - 1, c, c + 1, 628, 629, 700, 2*c - 1, 2 * c, 2*c + 1, 1233}
	for _, n := range rows {
		s.Scroll(PageListScroll{Tag: PageListScrollActive})
		s.Scroll(pl4ScrollRow(n))
		pl4LExpectViewportRow(t, s, n, true)
	}

	// Spot-check the exact node/y at the page boundaries.
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	s.Scroll(pl4ScrollRow(c - 1))
	expectEqual(t, s.pages.first, s.viewport_pin.node)
	expectEqual(t, SizeCellCountInt(c-1), s.viewport_pin.y)
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	s.Scroll(pl4ScrollRow(c))
	expectEqual(t, second, s.viewport_pin.node)
	expectEqual(t, SizeCellCountInt(0), s.viewport_pin.y)
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	s.Scroll(pl4ScrollRow(628))
	expectEqual(t, second, s.viewport_pin.node)
	expectEqual(t, SizeCellCountInt(628-c), s.viewport_pin.y)
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	s.Scroll(pl4ScrollRow(629))
	expectEqual(t, second, s.viewport_pin.node)
	expectEqual(t, SizeCellCountInt(629-c), s.viewport_pin.y)
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	s.Scroll(pl4ScrollRow(2*c - 1))
	expectEqual(t, second, s.viewport_pin.node)
	expectEqual(t, SizeCellCountInt(c-1), s.viewport_pin.y)
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	s.Scroll(pl4ScrollRow(2 * c))
	expectEqual(t, third, s.viewport_pin.node)
	expectEqual(t, SizeCellCountInt(0), s.viewport_pin.y)
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	s.Scroll(pl4ScrollRow(1233))
	expectEqual(t, third, s.viewport_pin.node)
	expectEqual(t, SizeCellCountInt(75), s.viewport_pin.y)
}

// Zig: test "term51: scroll: row variant from top and from a pin without cache"
func TestPl4_term51_scroll_row_variant_from_top_and_from_a_pin_without_cache(t *testing.T) {
	testStdCapacity(t)
	s := pl4LInitPages(t, 100)
	defer s.Deinit()

	// From top: the stale cache is replaced.
	s.viewport_pin_row_offset = Some[uint](5)
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	s.Scroll(pl4ScrollRow(40))
	pl4LExpectViewportRow(t, s, 40, true)

	// From a pin viewport whose cache is null: slow path, cache set.
	s.Scroll(pl4ScrollPin(pl4Pin(t, s, pl4ScreenPt(0, 900))))
	expectEqual(t, None[uint](), s.viewport_pin_row_offset)
	s.Scroll(pl4ScrollRow(55))
	pl4LExpectViewportRow(t, s, 55, true)
}

// Zig: test "term51: scroll: row variant fast path from cached pin"
func TestPl4_term51_scroll_row_variant_fast_path_from_cached_pin(t *testing.T) {
	testStdCapacity(t)
	s := pl4LInitPages(t, 100)
	defer s.Deinit()

	pl4LPinAt(t, s, 100)

	// Down, across the page boundary.
	s.Scroll(pl4ScrollRow(700))
	pl4LExpectViewportRow(t, s, 700, true)

	// Up.
	s.Scroll(pl4ScrollRow(50))
	pl4LExpectViewportRow(t, s, 50, true)

	// Same row: nothing changes.
	node := s.viewport_pin.node
	s.Scroll(pl4ScrollRow(50))
	pl4LExpectViewportRow(t, s, 50, true)
	expectEqual(t, node, s.viewport_pin.node)

	// Last pin row before active.
	s.Scroll(pl4ScrollRow(s.total_rows - uint(s.rows) - 1))
	pl4LExpectViewportRow(t, s, s.total_rows-uint(s.rows)-1, true)

	// To row 0 and the active top via the early checks.
	s.Scroll(pl4ScrollRow(0))
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
	pl4LPinAt(t, s, 100)
	s.Scroll(pl4ScrollRow(s.total_rows - uint(s.rows)))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
}

// Zig: test "term51: scroll: row variant beyond 16 bit offsets falls back to active"
func TestPl4_term51_scroll_row_variant_beyond_16_bit_offsets_falls_back_to_active(t *testing.T) {
	testStdCapacity(t)
	// CellCountInt is u16 so a slow-path walk to a row that does not fit
	// it jumps to the active area (and leaves the offset cache stale).
	s := pl4Init(t, 1, 5, pl4NoMax())
	defer s.Deinit()
	pl4LGrowRows(t, s, 140_000-5)
	expectEqual(t, uint(140_000), s.total_rows)
	expect(t, s.pages.first != s.pages.last, "s.pages.first != s.pages.last")
	// midpoint is 70000: 65535 and 65536 are on the forward side, 74464
	// and 74465 are on the backward side (140000 - n must be <= 65535).

	// Forward walk: largest value that fits.
	s.Scroll(pl4ScrollRow(65535))
	pl4LExpectViewportRow(t, s, 65535, true)

	// Forward walk: one past.
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	s.Scroll(pl4ScrollRow(65536))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	expectEqual(t, Some[uint](65536), s.viewport_pin_row_offset)

	// Backward walk: largest remaining distance that fits.
	s.Scroll(pl4ScrollRow(140_000 - 65535))
	pl4LExpectViewportRow(t, s, 140_000-65535, true)

	// Backward walk: one past.
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	s.Scroll(pl4ScrollRow(140_000 - 65536))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	expectEqual(t, Some[uint](140_000-65536), s.viewport_pin_row_offset)

	// Forward, far past the limit.
	s.Scroll(pl4ScrollRow(69_999))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
}

// Zig: test "term51: scroll: row variant with inconsistent total rows falls back to active"
func TestPl4_term51_scroll_row_variant_with_inconsistent_total_rows_falls_back_to_active(t *testing.T) {
	testStdCapacity(t)
	// The "couldn't find the offset" fallthrough can only be reached when
	// total_rows disagrees with the pages, so fake that with the integrity
	// checks paused.
	s := pl4Init(t, 80, 24, pl4NoMax())
	defer s.Deinit()
	s.PauseIntegrityChecks(true)
	s.total_rows = 1000

	// Forward walk (400 < 500) runs off the end of the list.
	s.Scroll(pl4ScrollRow(400))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	expectEqual(t, Some[uint](400), s.viewport_pin_row_offset)

	// Backward walk (600 >= 500, remaining 400) runs off the start.
	s.viewport_pin_row_offset = None[uint]()
	s.Scroll(pl4ScrollRow(600))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	expectEqual(t, Some[uint](600), s.viewport_pin_row_offset)

	s.total_rows = 24
	s.viewport_pin_row_offset = None[uint]()
	s.PauseIntegrityChecks(false)
}

// Zig: test "term51: scroll: delta_row from pin moving up"
func TestPl4_term51_scroll_delta_row_from_pin_moving_up(t *testing.T) {
	testStdCapacity(t)
	s := pl4LInitPages(t, 100)
	defer s.Deinit()
	c := uint(s.pages.first.data.capacity.rows)

	// Within the page, cache follows.
	pl4LPinAt(t, s, 700)
	s.Scroll(pl4ScrollDeltaRow(-5))
	pl4LExpectViewportRow(t, s, 695, true)

	// Across a page boundary.
	pl4LPinAt(t, s, c+1)
	s.Scroll(pl4ScrollDeltaRow(-2))
	pl4LExpectViewportRow(t, s, c-1, true)
	expectEqual(t, s.pages.first, s.viewport_pin.node)

	// Up to exactly row 0: still a pin viewport (not converted to top).
	pl4LPinAt(t, s, 10)
	s.Scroll(pl4ScrollDeltaRow(-10))
	pl4LExpectViewportRow(t, s, 0, true)
	expectEqual(t, s.pages.first, s.viewport_pin.node)
	expectEqual(t, SizeCellCountInt(0), s.viewport_pin.y)

	// One more up: overflow, top. The pin and cache are left as they were.
	pl4LPinAt(t, s, 10)
	s.Scroll(pl4ScrollDeltaRow(-11))
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
	expectEqual(t, SizeCellCountInt(10), s.viewport_pin.y)
	expectEqual(t, Some[uint](10), s.viewport_pin_row_offset)

	// Largest magnitude.
	pl4LPinAt(t, s, 10)
	s.Scroll(pl4ScrollDeltaRow(-math.MaxInt64))
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")

	// Without a cache: the pin moves, the cache stays null.
	s.Scroll(pl4ScrollPin(pl4Pin(t, s, pl4ScreenPt(0, 900))))
	expectEqual(t, None[uint](), s.viewport_pin_row_offset)
	s.Scroll(pl4ScrollDeltaRow(-100))
	pl4LExpectViewportRow(t, s, 800, false)
}

// Zig: test "term51: scroll: delta_row from pin moving down"
func TestPl4_term51_scroll_delta_row_from_pin_moving_down(t *testing.T) {
	testStdCapacity(t)
	s := pl4LInitPages(t, 100)
	defer s.Deinit()
	c := uint(s.pages.first.data.capacity.rows)
	total := s.total_rows
	active_top := total - uint(s.rows)

	// Within the page.
	pl4LPinAt(t, s, 10)
	s.Scroll(pl4ScrollDeltaRow(5))
	pl4LExpectViewportRow(t, s, 15, true)

	// Across a page boundary.
	pl4LPinAt(t, s, c-1)
	s.Scroll(pl4ScrollDeltaRow(2))
	pl4LExpectViewportRow(t, s, c+1, true)
	expectEqual(t, s.pages.first.next, s.viewport_pin.node)

	// Down to the last row above active: stays a pin.
	pl4LPinAt(t, s, 1000)
	s.Scroll(pl4ScrollDeltaRow(int64(active_top - 1 - 1000)))
	pl4LExpectViewportRow(t, s, active_top-1, true)

	// Down to exactly the active top: active.
	pl4LPinAt(t, s, 1000)
	s.Scroll(pl4ScrollDeltaRow(int64(active_top - 1000)))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	// Down to the last row of the list (an offset, inside active): active.
	pl4LPinAt(t, s, 1000)
	s.Scroll(pl4ScrollDeltaRow(int64(total - 1 - 1000)))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	// One further: overflow, active.
	pl4LPinAt(t, s, 1000)
	s.Scroll(pl4ScrollDeltaRow(int64(total - 1000)))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	// Largest magnitude.
	pl4LPinAt(t, s, 1000)
	s.Scroll(pl4ScrollDeltaRow(math.MaxInt64))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	// Without a cache.
	s.Scroll(pl4ScrollPin(pl4Pin(t, s, pl4ScreenPt(0, 900))))
	s.Scroll(pl4ScrollDeltaRow(100))
	pl4LExpectViewportRow(t, s, 1000, false)
}

// Zig: test "term51: scroll: delta_row zero is a no-op in every viewport"
func TestPl4_term51_scroll_delta_row_zero_is_a_no_op_in_every_viewport(t *testing.T) {
	testStdCapacity(t)
	s := pl4LInitPages(t, 100)
	defer s.Deinit()

	s.Scroll(pl4ScrollDeltaRow(0))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	s.Scroll(pl4ScrollDeltaRow(0))
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")

	pl4LPinAt(t, s, 300)
	s.Scroll(pl4ScrollDeltaRow(0))
	pl4LExpectViewportRow(t, s, 300, true)
}

// Zig: test "term51: scroll: delta_row from top"
func TestPl4_term51_scroll_delta_row_from_top(t *testing.T) {
	testStdCapacity(t)
	s := pl4LInitPages(t, 100)
	defer s.Deinit()
	c := uint(s.pages.first.data.capacity.rows)
	total := s.total_rows
	active_top := total - uint(s.rows)

	// Backwards from the top: nowhere to go.
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	s.Scroll(pl4ScrollDeltaRow(-1))
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
	s.Scroll(pl4ScrollDeltaRow(-math.MaxInt64))
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")

	// Forward: a pin viewport (slow path), cache reset.
	s.viewport_pin_row_offset = Some[uint](9)
	s.Scroll(pl4ScrollDeltaRow(1))
	pl4LExpectViewportRow(t, s, 1, false)

	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	s.Scroll(pl4ScrollDeltaRow(int64(c)))
	pl4LExpectViewportRow(t, s, c, false)
	expectEqual(t, s.pages.first.next, s.viewport_pin.node)

	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	s.Scroll(pl4ScrollDeltaRow(int64(active_top - 1)))
	pl4LExpectViewportRow(t, s, active_top-1, false)

	// Exactly the active top, one past it, and overflow: active.
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	s.Scroll(pl4ScrollDeltaRow(int64(active_top)))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	s.Scroll(pl4ScrollDeltaRow(int64(active_top + 1)))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	s.Scroll(pl4ScrollDeltaRow(int64(total + 5)))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	s.Scroll(pl4ScrollDeltaRow(math.MaxInt64))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
}

// Zig: test "term51: scroll: delta_row from active"
func TestPl4_term51_scroll_delta_row_from_active(t *testing.T) {
	testStdCapacity(t)
	s := pl4LInitPages(t, 100)
	defer s.Deinit()
	total := s.total_rows
	active_top := total - uint(s.rows)

	// Forward from active: nothing to do.
	s.Scroll(pl4ScrollDeltaRow(1))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	s.Scroll(pl4ScrollDeltaRow(math.MaxInt64))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	// Backward: a pin viewport, cache reset.
	s.viewport_pin_row_offset = Some[uint](9)
	s.Scroll(pl4ScrollDeltaRow(-1))
	pl4LExpectViewportRow(t, s, active_top-1, false)

	// Back across a page boundary (active top is 76 rows into the last page).
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	s.Scroll(pl4ScrollDeltaRow(-77))
	pl4LExpectViewportRow(t, s, active_top-77, false)
	expectEqual(t, s.pages.first.next, s.viewport_pin.node)

	// To row 1, then row 0 which is converted to top.
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	s.Scroll(pl4ScrollDeltaRow(-int64(active_top - 1)))
	pl4LExpectViewportRow(t, s, 1, false)
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	s.Scroll(pl4ScrollDeltaRow(-int64(active_top)))
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")

	// Past the top: overflow end is the first row: top.
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	s.Scroll(pl4ScrollDeltaRow(-int64(active_top + 1)))
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	s.Scroll(pl4ScrollDeltaRow(-math.MaxInt64))
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
}

// Zig: test "term51: scroll: delta_row from active with little scrollback"
func TestPl4_term51_scroll_delta_row_from_active_with_little_scrollback(t *testing.T) {
	testStdCapacity(t)

	// No scrollback at all: moving up overflows to the first row, which is
	// inside the active area, so the viewport stays active (not top).
	{
		s := pl4Init(t, 80, 24, pl4NoMax())
		defer s.Deinit()
		s.Scroll(pl4ScrollDeltaRow(-5))
		expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	}

	// One row of scrollback: up one is row 0, which is top.
	{
		s := pl4Init(t, 80, 24, pl4NoMax())
		defer s.Deinit()
		pl4LGrowRows(t, s, 1)
		s.Scroll(pl4ScrollDeltaRow(-1))
		expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
	}

	// Two rows of scrollback: up one is a pin at row 1, up two is top.
	{
		s := pl4Init(t, 80, 24, pl4NoMax())
		defer s.Deinit()
		pl4LGrowRows(t, s, 2)
		s.Scroll(pl4ScrollDeltaRow(-1))
		pl4LExpectViewportRow(t, s, 1, false)
		s.Scroll(PageListScroll{Tag: PageListScrollActive})
		s.Scroll(pl4ScrollDeltaRow(-2))
		expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")
	}

	// Down from top with no scrollback is active (top == active).
	{
		s := pl4Init(t, 80, 24, pl4NoMax())
		defer s.Deinit()
		s.Scroll(PageListScroll{Tag: PageListScrollTop})
		s.Scroll(pl4ScrollDeltaRow(1))
		expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	}
}

// Zig: test "term51: scroll: delta_prompt delegates to scrollPrompt"
func TestPl4_term51_scroll_delta_prompt_delegates_to_scrollPrompt(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 10, pl4NoMax())
	defer s.Deinit()
	pl4LGrowRows(t, s, 40)
	pl4Pin(t, s, pl4ScreenPt(0, 20)).RowAndCell().row.semantic_prompt = PageRowSemanticPromptPrompt

	// Zero: nothing.
	s.Scroll(pl4ScrollDeltaPrompt(0))
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	// Up one prompt.
	s.Scroll(pl4ScrollDeltaPrompt(-1))
	pl4LExpectViewportRow(t, s, 20, false)

	// Down one prompt: none below: nothing changes.
	s.Scroll(pl4ScrollDeltaPrompt(1))
	pl4LExpectViewportRow(t, s, 20, true)
}

// Zig: test "term51: scrollPrompt: zero delta does nothing"
func TestPl4_term51_scrollPrompt_zero_delta_does_nothing(t *testing.T) {
	testStdCapacity(t)
	s := pl4LPromptList(t)
	defer s.Deinit()
	pl4LMark(t, s, 20, PageRowSemanticPromptPrompt)
	pl4LMark(t, s, 30, PageRowSemanticPromptPrompt)

	s.scrollPrompt(0)
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	s.scrollPrompt(0)
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")

	pl4LPinAt(t, s, 25)
	s.scrollPrompt(0)
	pl4LExpectViewportRow(t, s, 25, true)
}

// Zig: test "term51: scrollPrompt: backward from active"
func TestPl4_term51_scrollPrompt_backward_from_active(t *testing.T) {
	testStdCapacity(t)
	s := pl4LPromptList(t)
	defer s.Deinit()
	for _, y := range []uint{5, 12, 20, 30} {
		pl4LMark(t, s, y, PageRowSemanticPromptPrompt)
	}

	expected := []struct {
		delta int64
		row   uint
	}{
		{delta: -1, row: 30},
		{delta: -2, row: 20},
		{delta: -3, row: 12},
		{delta: -4, row: 5},
		// More than there are: the furthest one.
		{delta: -5, row: 5},
		{delta: -1000, row: 5},
	}
	for _, e := range expected {
		s.Scroll(PageListScroll{Tag: PageListScrollActive})
		s.scrollPrompt(e.delta)
		pl4LExpectViewportRow(t, s, e.row, false)
	}
}

// Zig: test "term51: scrollPrompt: backward resets the cached offset and sets x to 0"
func TestPl4_term51_scrollPrompt_backward_resets_the_cached_offset_and_sets_x_to_0(t *testing.T) {
	testStdCapacity(t)
	s := pl4LPromptList(t)
	defer s.Deinit()
	pl4LMark(t, s, 20, PageRowSemanticPromptPrompt)

	// A pin viewport with a computed cache and an x.
	s.Scroll(pl4ScrollPin(PageListPin{node: s.pages.first, y: 35, x: 7}))
	expectEqual(t, uint(35), s.Scrollbar().offset)
	expectEqual(t, Some[uint](35), s.viewport_pin_row_offset)

	s.scrollPrompt(-1)
	expectEqual(t, None[uint](), s.viewport_pin_row_offset)
	pl4LExpectViewportRow(t, s, 20, false)
}

// Zig: test "term51: scrollPrompt: backward skips a prompt on the viewport top row"
func TestPl4_term51_scrollPrompt_backward_skips_a_prompt_on_the_viewport_top_row(t *testing.T) {
	testStdCapacity(t)
	s := pl4LPromptList(t)
	defer s.Deinit()
	pl4LMark(t, s, 20, PageRowSemanticPromptPrompt)
	pl4LMark(t, s, 30, PageRowSemanticPromptPrompt)

	// Viewport top is the prompt at 30: the next one back is 20.
	s.Scroll(pl4ScrollPin(pl4Pin(t, s, pl4ScreenPt(0, 30))))
	s.scrollPrompt(-1)
	pl4LExpectViewportRow(t, s, 20, false)

	// Viewport top one below the prompt: it is found.
	s.Scroll(pl4ScrollPin(pl4Pin(t, s, pl4ScreenPt(0, 31))))
	s.scrollPrompt(-1)
	pl4LExpectViewportRow(t, s, 30, false)
}

// Zig: test "term51: scrollPrompt: backward with no prompt or at the top does nothing"
func TestPl4_term51_scrollPrompt_backward_with_no_prompt_or_at_the_top_does_nothing(t *testing.T) {
	testStdCapacity(t)
	s := pl4LPromptList(t)
	defer s.Deinit()

	// No prompts at all: active stays active.
	s.scrollPrompt(-1)
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	// At the top there is no row above: top stays top, even with a
	// prompt in the first row.
	pl4LMark(t, s, 0, PageRowSemanticPromptPrompt)
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	s.scrollPrompt(-1)
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")

	// Viewport pinned at row 1: the prompt at row 0 is found, as a pin
	// viewport at row 0 (it is not converted to top).
	s.Scroll(pl4ScrollPin(pl4Pin(t, s, pl4ScreenPt(0, 1))))
	expectEqual(t, uint(1), s.Scrollbar().offset)
	s.scrollPrompt(-1)
	pl4LExpectViewportRow(t, s, 0, false)
	expectEqual(t, s.pages.first, s.viewport_pin.node)

	// A pin viewport with nothing above it: unchanged and the cache kept.
	pl4LMark(t, s, 0, PageRowSemanticPromptNone)
	s.Scroll(pl4ScrollPin(pl4Pin(t, s, pl4ScreenPt(0, 5))))
	expectEqual(t, uint(5), s.Scrollbar().offset)
	s.scrollPrompt(-1)
	pl4LExpectViewportRow(t, s, 5, true)
}

// Zig: test "term51: scrollPrompt: backward over continuation lines"
func TestPl4_term51_scrollPrompt_backward_over_continuation_lines(t *testing.T) {
	testStdCapacity(t)
	s := pl4LPromptList(t)
	defer s.Deinit()
	pl4LMark(t, s, 10, PageRowSemanticPromptPrompt)
	pl4LMark(t, s, 11, PageRowSemanticPromptPromptContinuation)
	pl4LMark(t, s, 12, PageRowSemanticPromptPromptContinuation)

	// Viewport tops at 13, 12, 11: all find the prompt row 10.
	for _, top := range []uint{13, 12, 11} {
		s.Scroll(pl4ScrollPin(pl4Pin(t, s, pl4ScreenPt(0, top))))
		s.scrollPrompt(-1)
		pl4LExpectViewportRow(t, s, 10, false)
	}

	// Viewport top at the prompt itself: nothing above, unchanged.
	s.Scroll(pl4ScrollPin(pl4Pin(t, s, pl4ScreenPt(0, 10))))
	s.scrollPrompt(-1)
	pl4LExpectViewportRow(t, s, 10, false)

	// A multi-line prompt counts as one.
	pl4LMark(t, s, 5, PageRowSemanticPromptPrompt)
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	s.scrollPrompt(-1)
	pl4LExpectViewportRow(t, s, 10, false)
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	s.scrollPrompt(-2)
	pl4LExpectViewportRow(t, s, 5, false)
}

// Zig: test "term51: scrollPrompt: backward over cut-off continuation lines"
func TestPl4_term51_scrollPrompt_backward_over_cut_off_continuation_lines(t *testing.T) {
	testStdCapacity(t)

	// Continuation lines whose prompt row is missing, ended by a normal
	// row above: the topmost continuation row is the prompt.
	{
		s := pl4LPromptList(t)
		defer s.Deinit()
		pl4LMark(t, s, 5, PageRowSemanticPromptPromptContinuation)
		pl4LMark(t, s, 6, PageRowSemanticPromptPromptContinuation)
		s.Scroll(pl4ScrollPin(pl4Pin(t, s, pl4ScreenPt(0, 7))))
		s.scrollPrompt(-1)
		pl4LExpectViewportRow(t, s, 5, false)
	}

	// Continuation lines running into the first row of the screen:
	// pinned behaviour is that the row where the search started (row 1,
	// not the topmost row 0) is returned.
	{
		s := pl4LPromptList(t)
		defer s.Deinit()
		pl4LMark(t, s, 0, PageRowSemanticPromptPromptContinuation)
		pl4LMark(t, s, 1, PageRowSemanticPromptPromptContinuation)
		s.Scroll(pl4ScrollPin(pl4Pin(t, s, pl4ScreenPt(0, 3))))
		s.scrollPrompt(-1)
		pl4LExpectViewportRow(t, s, 1, false)
	}
}

// Zig: test "term51: scrollPrompt: forward from top"
func TestPl4_term51_scrollPrompt_forward_from_top(t *testing.T) {
	testStdCapacity(t)
	s := pl4LPromptList(t)
	defer s.Deinit()
	for _, y := range []uint{20, 30, 39} {
		pl4LMark(t, s, y, PageRowSemanticPromptPrompt)
	}
	// Row 45 is inside the active area.
	pl4LMark(t, s, 45, PageRowSemanticPromptPrompt)

	expected := []struct {
		delta int64
		row   Optional[uint]
	}{
		{delta: 1, row: Some[uint](20)},
		{delta: 2, row: Some[uint](30)},
		{delta: 3, row: Some[uint](39)},
		// Row 45 is in the active area: the viewport becomes active.
		{delta: 4, row: None[uint]()},
		{delta: 5, row: None[uint]()},
		{delta: 1000, row: None[uint]()},
	}
	for _, e := range expected {
		s.Scroll(PageListScroll{Tag: PageListScrollTop})
		s.scrollPrompt(e.delta)
		if e.row.Valid {
			pl4LExpectViewportRow(t, s, e.row.Value, false)
		} else {
			expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
		}
	}
}

// Zig: test "term51: scrollPrompt: forward skips continuation lines of the viewport prompt"
func TestPl4_term51_scrollPrompt_forward_skips_continuation_lines_of_the_viewport_prompt(t *testing.T) {
	testStdCapacity(t)
	s := pl4LPromptList(t)
	defer s.Deinit()
	pl4LMark(t, s, 20, PageRowSemanticPromptPrompt)
	pl4LMark(t, s, 21, PageRowSemanticPromptPromptContinuation)
	pl4LMark(t, s, 22, PageRowSemanticPromptPromptContinuation)
	pl4LMark(t, s, 30, PageRowSemanticPromptPrompt)
	pl4LMark(t, s, 31, PageRowSemanticPromptPromptContinuation)

	// Viewport top is the prompt row or one of its continuation rows: the
	// whole prompt is skipped.
	for _, top := range []uint{20, 21, 22} {
		s.Scroll(pl4ScrollPin(pl4Pin(t, s, pl4ScreenPt(0, top))))
		s.scrollPrompt(1)
		pl4LExpectViewportRow(t, s, 30, false)
	}

	// Viewport top just above the prompt: it is found.
	s.Scroll(pl4ScrollPin(pl4Pin(t, s, pl4ScreenPt(0, 19))))
	s.scrollPrompt(1)
	pl4LExpectViewportRow(t, s, 20, false)

	// A multi-line prompt counts once.
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	s.scrollPrompt(2)
	pl4LExpectViewportRow(t, s, 30, false)

	// Nothing after 30/31: unchanged and the cache is kept.
	s.Scroll(pl4ScrollPin(pl4Pin(t, s, pl4ScreenPt(0, 30))))
	expectEqual(t, uint(30), s.Scrollbar().offset)
	s.scrollPrompt(1)
	pl4LExpectViewportRow(t, s, 30, true)
}

// Zig: test "term51: scrollPrompt: forward from a non-prompt row counts a continuation row as a prompt"
func TestPl4_term51_scrollPrompt_forward_from_a_non_prompt_row_counts_a_continuation_row_as_a_prompt(t *testing.T) {
	testStdCapacity(t)
	s := pl4LPromptList(t)
	defer s.Deinit()
	pl4LMark(t, s, 20, PageRowSemanticPromptPromptContinuation)

	// The viewport row itself is a normal row, so no skipping happens
	// and the continuation row below is a prompt.
	s.Scroll(pl4ScrollPin(pl4Pin(t, s, pl4ScreenPt(0, 19))))
	s.scrollPrompt(1)
	pl4LExpectViewportRow(t, s, 20, false)
}

// Zig: test "term51: scrollPrompt: forward when continuation lines run to the end"
func TestPl4_term51_scrollPrompt_forward_when_continuation_lines_run_to_the_end(t *testing.T) {
	testStdCapacity(t)
	s := pl4LPromptList(t)
	defer s.Deinit()
	pl4LMark(t, s, 3, PageRowSemanticPromptPrompt)
	for y := uint(4); y < 50; y++ {
		pl4LMark(t, s, y, PageRowSemanticPromptPromptContinuation)
	}

	// The skip loop stops at the last row (which is still a continuation
	// row); that row is then reported as a prompt and is in the active
	// area.
	s.Scroll(pl4ScrollPin(pl4Pin(t, s, pl4ScreenPt(0, 3))))
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	s.scrollPrompt(1)
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
}

// Zig: test "term51: scrollPrompt: forward onto the active top or just above it"
func TestPl4_term51_scrollPrompt_forward_onto_the_active_top_or_just_above_it(t *testing.T) {
	testStdCapacity(t)
	s := pl4LPromptList(t)
	defer s.Deinit()

	// Prompt on the first active row: active.
	pl4LMark(t, s, 40, PageRowSemanticPromptPrompt)
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	s.scrollPrompt(1)
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	// Prompt on the row above: a pin viewport.
	pl4LMark(t, s, 40, PageRowSemanticPromptNone)
	pl4LMark(t, s, 39, PageRowSemanticPromptPrompt)
	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	s.scrollPrompt(1)
	pl4LExpectViewportRow(t, s, 39, false)
}

// Zig: test "term51: scrollPrompt: forward from active"
func TestPl4_term51_scrollPrompt_forward_from_active(t *testing.T) {
	testStdCapacity(t)
	s := pl4LPromptList(t)
	defer s.Deinit()
	pl4LMark(t, s, 45, PageRowSemanticPromptPrompt)

	// Already at the bottom: the prompt in the active area is found but
	// the viewport stays active.
	s.scrollPrompt(1)
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	// With no prompt below, a pin viewport is unchanged.
	pl4LMark(t, s, 45, PageRowSemanticPromptNone)
	pl4LPinAt(t, s, 20)
	s.scrollPrompt(1)
	pl4LExpectViewportRow(t, s, 20, true)
}

// Zig: test "term51: scrollPrompt: forward from the last row returns early"
func TestPl4_term51_scrollPrompt_forward_from_the_last_row_returns_early(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 1, pl4NoMax())
	defer s.Deinit()
	pl4LGrowRows(t, s, 5)
	pl4LMark(t, s, 5, PageRowSemanticPromptPrompt)

	// The single active row is the last row: nothing below it.
	s.scrollPrompt(1)
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")

	// Backward from there finds nothing either, there is no prompt above.
	s.scrollPrompt(-1)
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
	pl4LMark(t, s, 2, PageRowSemanticPromptPrompt)
	s.scrollPrompt(-1)
	pl4LExpectViewportRow(t, s, 2, false)
}

// Zig: test "term51: scrollPrompt: crosses page boundaries"
func TestPl4_term51_scrollPrompt_crosses_page_boundaries(t *testing.T) {
	testStdCapacity(t)
	s := pl4LInitPages(t, 100)
	defer s.Deinit()
	c := uint(s.pages.first.data.capacity.rows)
	pl4LMark(t, s, c-1, PageRowSemanticPromptPrompt)
	pl4LMark(t, s, c, PageRowSemanticPromptPrompt)
	pl4LMark(t, s, 2*c, PageRowSemanticPromptPrompt)

	s.scrollPrompt(-1)
	pl4LExpectViewportRow(t, s, 2*c, false)
	s.scrollPrompt(-1)
	pl4LExpectViewportRow(t, s, c, false)
	s.scrollPrompt(-1)
	pl4LExpectViewportRow(t, s, c-1, false)
	s.scrollPrompt(1)
	pl4LExpectViewportRow(t, s, c, false)
	s.scrollPrompt(1)
	pl4LExpectViewportRow(t, s, 2*c, false)
}

// Zig: test "term51: scrollClear: blank screen does nothing"
func TestPl4_term51_scrollClear_blank_screen_does_nothing(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 5, pl4NoMax())
	defer s.Deinit()

	zigTry(t, s.ScrollClear())
	expectEqual(t, uint(5), s.total_rows)
	expectEqual(t, uint(1), s.TotalPages())
}

// Zig: test "term51: scrollClear: scrolls up to the last non-empty row"
func TestPl4_term51_scrollClear_scrolls_up_to_the_last_non_empty_row(t *testing.T) {
	testStdCapacity(t)

	// Content at active row i scrolls i + 1 rows into history.
	for i := uint(0); i < 5; i++ {
		func() {
			s := pl4Init(t, 80, 5, pl4NoMax())
			defer s.Deinit()
			pl4LText(pl4Pin(t, s, pl4ActivePt(0, i)), 'A')
			zigTry(t, s.ScrollClear())
			expectEqual(t, 5+i+1, s.total_rows)
			expectEqual(t, 5+i+1, s.totalRows())

			// The content is now in history and the active area is blank.
			moved := pl4Pin(t, s, pl4ScreenPt(0, i))
			expectEqual(t, uint32('A'), moved.RowAndCell().cell.content.codepoint)
			expect(t, !s.PointFromPin(PointTagActive, moved).Valid, "s.pointFromPin(.active, moved) == null")
			for y := uint(0); y < 5; y++ {
				p := pl4Pin(t, s, pl4ActivePt(0, y))
				for _, cell := range p.Cells(PageListPinCellSubsetAll) {
					expect(t, cell.IsEmpty(), "cell.isEmpty()")
				}
			}

			// A second clear has nothing left to do.
			zigTry(t, s.ScrollClear())
			expectEqual(t, 5+i+1, s.total_rows)
		}()
	}
}

// Zig: test "term51: scrollClear: only the last non-empty row matters"
func TestPl4_term51_scrollClear_only_the_last_non_empty_row_matters(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 5, pl4NoMax())
	defer s.Deinit()
	pl4LText(pl4Pin(t, s, pl4ActivePt(0, 0)), 'A')
	pl4LText(pl4Pin(t, s, pl4ActivePt(79, 2)), 'B')

	zigTry(t, s.ScrollClear())
	expectEqual(t, uint(5+3), s.total_rows)
}

// Zig: test "term51: scrollClear: content only in history is not scrolled"
func TestPl4_term51_scrollClear_content_only_in_history_is_not_scrolled(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 5, pl4NoMax())
	defer s.Deinit()
	pl4LGrowRows(t, s, 10)
	expectEqual(t, uint(15), s.total_rows)

	// The last history row and a row further up.
	pl4LText(pl4Pin(t, s, pl4ScreenPt(0, 9)), 'A')
	pl4LText(pl4Pin(t, s, pl4ScreenPt(0, 3)), 'B')
	zigTry(t, s.ScrollClear())
	expectEqual(t, uint(15), s.total_rows)

	// The first active row does count.
	pl4LText(pl4Pin(t, s, pl4ScreenPt(0, 10)), 'C')
	zigTry(t, s.ScrollClear())
	expectEqual(t, uint(16), s.total_rows)
}

// Zig: test "term51: scrollClear: a blank single page taller than the active area"
func TestPl4_term51_scrollClear_a_blank_single_page_taller_than_the_active_area(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 5, pl4NoMax())
	defer s.Deinit()
	pl4LGrowRows(t, s, 10)

	zigTry(t, s.ScrollClear())
	expectEqual(t, uint(15), s.total_rows)
}

// Zig: test "term51: scrollClear: which cells count as non-empty"
func TestPl4_term51_scrollClear_which_cells_count_as_non_empty(t *testing.T) {
	testStdCapacity(t)

	// Colour-only cells are content.
	{
		s := pl4Init(t, 80, 5, pl4NoMax())
		defer s.Deinit()
		*pl4Pin(t, s, pl4ActivePt(10, 3)).RowAndCell().cell = PageCell{
			content_tag: PageCellContentTagBgColorPalette,
			content:     PageCellContent{color_palette: 1},
		}
		zigTry(t, s.ScrollClear())
		expectEqual(t, uint(5+4), s.total_rows)
	}
	{
		s := pl4Init(t, 80, 5, pl4NoMax())
		defer s.Deinit()
		*pl4Pin(t, s, pl4ActivePt(10, 1)).RowAndCell().cell = PageCell{
			content_tag: PageCellContentTagBgColorRgb,
			content:     PageCellContent{color_rgb: PageCellRGB{r: 1, g: 2, b: 3}},
		}
		zigTry(t, s.ScrollClear())
		expectEqual(t, uint(5+2), s.total_rows)
	}

	// A narrow cell with a zero codepoint is empty.
	{
		s := pl4Init(t, 80, 5, pl4NoMax())
		defer s.Deinit()
		*pl4Pin(t, s, pl4ActivePt(10, 4)).RowAndCell().cell = PageCell{
			content_tag: PageCellContentTagCodepoint,
			content:     PageCellContent{codepoint: 0},
		}
		zigTry(t, s.ScrollClear())
		expectEqual(t, uint(5), s.total_rows)
	}

	// A wide spacer is meaningful even with no text.
	{
		s := pl4Init(t, 80, 5, pl4NoMax())
		defer s.Deinit()
		node := s.pages.first
		node.data.PauseIntegrityChecks(true)
		defer node.data.PauseIntegrityChecks(false)
		*pl4Pin(t, s, pl4ActivePt(79, 2)).RowAndCell().cell = PageCell{
			content_tag: PageCellContentTagCodepoint,
			content:     PageCellContent{codepoint: 0},
			wide:        PageCellWideSpacerHead,
		}
		zigTry(t, s.ScrollClear())
		expectEqual(t, uint(5+3), s.total_rows)
	}
}

// Zig: test "term51: scrollClear: only the first cols cells of a row are examined"
func TestPl4_term51_scrollClear_only_the_first_cols_cells_of_a_row_are_examined(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 5, pl4NoMax())
	defer s.Deinit()
	pl4LText(pl4Pin(t, s, pl4ActivePt(5, 0)), 'A')

	// Pretend the screen is narrower than the page: column 5 is ignored.
	s.cols = 5
	zigTry(t, s.ScrollClear())
	expectEqual(t, uint(5), s.total_rows)

	// Wide enough to include column 5.
	s.cols = 6
	zigTry(t, s.ScrollClear())
	expectEqual(t, uint(6), s.total_rows)
	s.cols = 80
}

// Zig: test "term51: scrollClear: active area spanning two pages"
func TestPl4_term51_scrollClear_active_area_spanning_two_pages(t *testing.T) {
	testStdCapacity(t)
	const rows_in_last = 3

	// Active is the last 21 rows of the first page and 3 rows of the next.
	// Content on the last row of the first page: 3 blank rows follow.
	{
		s := pl4Init(t, 80, 24, pl4NoMax())
		defer s.Deinit()
		pl4LFillLast(t, s)
		pl4GrowNode(t, s)
		pl4LGrowRows(t, s, rows_in_last-1)
		c := uint(s.pages.first.data.capacity.rows)
		expectEqual(t, uint(rows_in_last), uint(s.pages.last.data.size.rows))
		total := s.total_rows

		pl4LText(pl4Pin(t, s, pl4ScreenPt(0, c-1)), 'A')
		zigTry(t, s.ScrollClear())
		expectEqual(t, total+21, s.total_rows)
	}

	// Content on the second row of the last page: 1 blank row follows.
	{
		s := pl4Init(t, 80, 24, pl4NoMax())
		defer s.Deinit()
		pl4LFillLast(t, s)
		pl4GrowNode(t, s)
		pl4LGrowRows(t, s, rows_in_last-1)
		c := uint(s.pages.first.data.capacity.rows)
		total := s.total_rows

		pl4LText(pl4Pin(t, s, pl4ScreenPt(0, c+1)), 'A')
		zigTry(t, s.ScrollClear())
		expectEqual(t, total+23, s.total_rows)
	}

	// Content on the first active row (in the first page): one row.
	// Content on the row above it: none.
	{
		s := pl4Init(t, 80, 24, pl4NoMax())
		defer s.Deinit()
		pl4LFillLast(t, s)
		pl4GrowNode(t, s)
		pl4LGrowRows(t, s, rows_in_last-1)
		total := s.total_rows
		active_top := total - 24

		pl4LText(pl4Pin(t, s, pl4ScreenPt(0, active_top-1)), 'A')
		zigTry(t, s.ScrollClear())
		expectEqual(t, total, s.total_rows)

		pl4LText(pl4Pin(t, s, pl4ScreenPt(0, active_top)), 'B')
		zigTry(t, s.ScrollClear())
		expectEqual(t, total+1, s.total_rows)
	}
}

// Zig: test "term51: scrollClear: does not change the viewport"
func TestPl4_term51_scrollClear_does_not_change_the_viewport(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 5, pl4NoMax())
	defer s.Deinit()
	pl4LGrowRows(t, s, 20)
	pl4LText(pl4Pin(t, s, pl4ActivePt(0, 4)), 'A')

	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	zigTry(t, s.ScrollClear())
	expectEqual(t, uint(30), s.total_rows)
	expect(t, s.viewport.Tag == PageListViewportTop, "s.viewport == .top")

	pl4LText(pl4Pin(t, s, pl4ActivePt(0, 4)), 'B')
	pl4LPinAt(t, s, 7)
	zigTry(t, s.ScrollClear())
	expectEqual(t, uint(35), s.total_rows)
	pl4LExpectViewportRow(t, s, 7, true)

	pl4LText(pl4Pin(t, s, pl4ActivePt(0, 0)), 'C')
	s.Scroll(PageListScroll{Tag: PageListScrollActive})
	zigTry(t, s.ScrollClear())
	expectEqual(t, uint(36), s.total_rows)
	expect(t, s.viewport.Tag == PageListViewportActive, "s.viewport == .active")
}

// Zig: test "term51: scrollClear: grows through the pruning path at the max size"
func TestPl4_term51_scrollClear_grows_through_the_pruning_path_at_the_max_size(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 5, pl4NoMax())
	defer s.Deinit()
	// Force pruning on every page allocation.
	s.explicit_max_size = 0
	s.min_max_size = 0

	// Two full pages.
	pl4LFillLast(t, s)
	pl4GrowNode(t, s)
	pl4LFillLast(t, s)
	c := uint(s.pages.first.data.capacity.rows)
	expectEqual(t, 2*c, s.total_rows)

	// Content on the last row: 5 rows to scroll. The first grow prunes the
	// first page and reuses it as the new last page, the other four fit.
	pl4LText(pl4Pin(t, s, pl4ActivePt(0, 4)), 'A')
	second := s.pages.last
	zigTry(t, s.ScrollClear())
	expectEqual(t, c+5, s.total_rows)
	expectEqual(t, uint(2), s.TotalPages())
	expectEqual(t, second, s.pages.first)
	expectEqual(t, SizeCellCountInt(5), s.pages.last.data.size.rows)
	expectEqual(
		t,
		uint32('A'),
		pl4Pin(t, s, pl4ScreenPt(0, c-1)).RowAndCell().cell.content.codepoint,
	)
}

// pl4Compact is `try s.compact(node)`.
func pl4Compact(t *testing.T, s *PageList, node *PageListNode) *PageListNode {
	t.Helper()
	n, err := s.Compact(node)
	zigTry(t, err)
	return n
}

// pl4CompactNode is `(try s.compact(node)).?`.
func pl4CompactNode(t *testing.T, s *PageList, node *PageListNode) *PageListNode {
	t.Helper()
	n := pl4Compact(t, s, node)
	if n == nil {
		t.Fatalf("unwrapped a null optional")
	}
	return n
}

// Zig: test "term51: compact: pages of standard size or smaller return null"
func TestPl4_term51_compact_pages_of_standard_size_or_smaller_return_null(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 24, pl4NoMax())
	defer s.Deinit()

	// A freshly created 80 column page is exactly the standard size, which
	// is the boundary of the "<=" check.
	node := s.pages.first
	expectEqual(t, pageListStdSize(), pageListMemoryLen(&node.data))
	page_size := s.page_size
	serial := s.page_serial

	expectEqual(t, (*PageListNode)(nil), pl4Compact(t, s, node))
	expectEqual(t, node, s.pages.first)
	expectEqual(t, node, s.pages.last)
	expectEqual(t, page_size, s.page_size)
	expectEqual(t, serial, s.page_serial)

	// A page whose layout is smaller than the standard size.
	small := pl4Init(t, 2, 2, pl4NoMax())
	defer small.Deinit()
	small_node := small.pages.first
	expect(t, pageListMemoryLen(&small_node.data) <= pageListStdSize(), "small_node.data.memory.len <= std_size")
	expectEqual(t, (*PageListNode)(nil), pl4Compact(t, small, small_node))
	expectEqual(t, small_node, small.pages.first)
}

// Zig: test "term51: compact: oversized page in the middle is replaced"
func TestPl4_term51_compact_oversized_page_in_the_middle_is_replaced(t *testing.T) {
	testStdCapacity(t)
	s := pl4LInitPages(t, 100)
	defer s.Deinit()
	c := uint(s.pages.first.data.capacity.rows)
	first := s.pages.first
	third := s.pages.last
	node := first.next

	// Contents and tracked pins on every page.
	pl4LText(PageListPin{node: node, x: 3, y: 7}, 'X')
	pl4LText(PageListPin{node: node, x: 79, y: SizeCellCountInt(c - 1)}, 'Y')
	pin_mid := pl4TrackPin(t, s, PageListPin{node: node, x: 5, y: 10})
	pin_last := pl4TrackPin(t, s, PageListPin{node: node, x: 79, y: SizeCellCountInt(c - 1)})
	pin_first := pl4TrackPin(t, s, PageListPin{node: first, x: 2, y: 3})
	pin_third := pl4TrackPin(t, s, PageListPin{node: third, x: 1, y: 0})
	s.Scroll(pl4ScrollPin(PageListPin{node: node, y: 20}))
	expectEqual(t, c+20, s.Scrollbar().offset)

	node = pl4LMakeOversized(t, s, node)
	expectEqual(t, node, pin_mid.node)
	node.data.dirty = true
	old_len := pageListMemoryLen(&node.data)
	old_size := node.data.size
	total_rows := s.total_rows
	serial := s.page_serial
	page_size := s.page_size
	old_cap := node.data.capacity
	expect(t, old_len > pageListStdSize(), "old_len > std_size")

	new := pl4CompactNode(t, s, node)

	// The page is smaller and still has the same size.
	expect(t, pageListMemoryLen(&new.data) < old_len, "new.data.memory.len < old_len")
	expectEqual(t, old_size.rows, new.data.size.rows)
	expectEqual(t, old_size.cols, new.data.size.cols)
	expect(t, new.data.dirty, "new.data.dirty")
	expectEqual(t, old_size.rows, new.data.capacity.rows)
	expectEqual(t, old_size.cols, new.data.capacity.cols)
	expect(t, new.data.capacity.styles < old_cap.styles, "new.data.capacity.styles < old_cap.styles")
	expectEqual(t, total_rows, s.total_rows)

	// A fresh serial was used.
	expectEqual(t, serial, new.serial)
	expectEqual(t, serial+1, s.page_serial)

	// Same place in the list.
	expectEqual(t, first, s.pages.first)
	expectEqual(t, third, s.pages.last)
	expectEqual(t, new, first.next)
	expectEqual(t, first, new.prev)
	expectEqual(t, third, new.next)
	expectEqual(t, new, third.prev)
	expectEqual(t, uint(3), s.TotalPages())

	// Tracked pins moved if they were on the page, untouched otherwise.
	expectEqual(t, new, pin_mid.node)
	expectEqual(t, SizeCellCountInt(5), pin_mid.x)
	expectEqual(t, SizeCellCountInt(10), pin_mid.y)
	expectEqual(t, new, pin_last.node)
	expectEqual(t, SizeCellCountInt(79), pin_last.x)
	expectEqual(t, SizeCellCountInt(c-1), pin_last.y)
	expectEqual(t, first, pin_first.node)
	expectEqual(t, third, pin_third.node)
	expectEqual(t, new, s.viewport_pin.node)
	expectEqual(t, SizeCellCountInt(20), s.viewport_pin.y)
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, c+20, s.Scrollbar().offset)

	// Contents.
	expectEqual(t, uint32('X'), new.data.GetRowAndCell(3, 7).cell.content.codepoint)
	expectEqual(t, uint32('Y'), new.data.GetRowAndCell(79, c-1).cell.content.codepoint)
	expect(t, new.data.GetRowAndCell(4, 7).cell.IsEmpty(), "new.data.getRowAndCell(4, 7).cell.isEmpty()")

	// Accounting: the new (pooled) page adds a standard page, the old
	// oversized one is subtracted by its length.
	expect(t, pageListMemoryLen(&new.data) <= pageListStdSize(), "new.data.memory.len <= std_size")
	expectEqual(t, page_size+pageListStdSize()-old_len, s.page_size)
}

// Zig: test "term51: compact: sole page becomes first and last"
func TestPl4_term51_compact_sole_page_becomes_first_and_last(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 24, pl4NoMax())
	defer s.Deinit()

	node := pl4LMakeOversized(t, s, s.pages.first)
	expectEqual(t, node, s.pages.first)
	expectEqual(t, node, s.pages.last)

	new := pl4CompactNode(t, s, node)
	expectEqual(t, new, s.pages.first)
	expectEqual(t, new, s.pages.last)
	expectEqual(t, (*PageListNode)(nil), new.prev)
	expectEqual(t, (*PageListNode)(nil), new.next)
	expectEqual(t, SizeCellCountInt(24), new.data.size.rows)
	expectEqual(t, uint(24), s.total_rows)

	// Tracked viewport pin followed the page.
	expectEqual(t, new, s.viewport_pin.node)
}

// Zig: test "term51: compact: first and last pages of a longer list"
func TestPl4_term51_compact_first_and_last_pages_of_a_longer_list(t *testing.T) {
	testStdCapacity(t)
	s := pl4LInitPages(t, 100)
	defer s.Deinit()
	second := s.pages.first.next

	// First page.
	{
		node := pl4LMakeOversized(t, s, s.pages.first)
		new := pl4CompactNode(t, s, node)
		expectEqual(t, new, s.pages.first)
		expectEqual(t, (*PageListNode)(nil), new.prev)
		expectEqual(t, second, new.next)
		expectEqual(t, new, second.prev)
		expectEqual(t, new, s.viewport_pin.node)
	}

	// Last page.
	{
		node := pl4LMakeOversized(t, s, s.pages.last)
		new := pl4CompactNode(t, s, node)
		expectEqual(t, new, s.pages.last)
		expectEqual(t, (*PageListNode)(nil), new.next)
		expectEqual(t, second, new.prev)
		expectEqual(t, new, second.next)
		expectEqual(t, SizeCellCountInt(100), new.data.size.rows)
	}
	expectEqual(t, uint(3), s.TotalPages())
}

// Zig: test "term51: compact: preserves styles graphemes and hyperlinks at exact capacity"
func TestPl4_term51_compact_preserves_styles_graphemes_and_hyperlinks_at_exact_capacity(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 20, 5, pl4NoMax())
	defer s.Deinit()

	{
		page := &s.pages.first.data

		// Two cells sharing one style and one with another.
		bold := pl4StylesAdd(t, page, pl4Bold())
		page.styles.Use(bold)
		italic := pl4StylesAdd(t, page, pl4Italic())
		for _, e := range []struct {
			x  uint
			y  uint
			id StyleId
		}{
			{x: 0, y: 0, id: bold},
			{x: 1, y: 0, id: bold},
			{x: 0, y: 1, id: italic},
		} {
			rac := page.GetRowAndCell(e.x, e.y)
			*rac.cell = PageCell{
				content_tag: PageCellContentTagCodepoint,
				content:     PageCellContent{codepoint: 'S'},
				style_id:    e.id,
			}
			rac.row.styled = true
		}

		// A grapheme cluster.
		{
			rac := page.GetRowAndCell(2, 0)
			*rac.cell = PageCell{
				content_tag: PageCellContentTagCodepoint,
				content:     PageCellContent{codepoint: 0x1F468},
			}
			zigTry(t, page.SetGraphemes(rac.row, rac.cell, []uint32{0x200D, 0x1F469}))
		}

		// A hyperlink.
		link := pl4InsertHyperlink(t, page, Hyperlink{
			id:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: 0},
			uri: []uint8("https://example.com"),
		})
		{
			rac := page.GetRowAndCell(3, 2)
			*rac.cell = PageCell{
				content_tag: PageCellContentTagCodepoint,
				content:     PageCellContent{codepoint: 'L'},
			}
			zigTry(t, page.SetHyperlink(rac.row, rac.cell, link))
		}
	}

	node := pl4LMakeOversized(t, s, s.pages.first)
	expected := node.data.ExactRowCapacity(0, uint(node.data.size.rows))
	new := pl4CompactNode(t, s, node)
	page := &new.data

	expectEqual(t, expected, page.capacity)
	expectEqual(t, uint(2), page.styles.Count())
	expectEqual(t, uint(1), page.GraphemeCount())
	expectEqual(t, uint(1), page.hyperlink_set.Count())

	{
		rac := page.GetRowAndCell(0, 0)
		expect(t, page.styles.Get(rac.cell.style_id).flags.bold, "page.styles.get(page.memory, rac.cell.style_id).flags.bold")
		expect(t, rac.row.styled, "rac.row.styled")
		rac2 := page.GetRowAndCell(1, 0)
		expectEqual(t, rac.cell.style_id, rac2.cell.style_id)
		rac3 := page.GetRowAndCell(0, 1)
		expect(t, page.styles.Get(rac3.cell.style_id).flags.italic, "page.styles.get(page.memory, rac3.cell.style_id).flags.italic")
	}
	{
		rac := page.GetRowAndCell(2, 0)
		expectEqual(t, uint32(0x1F468), rac.cell.content.codepoint)
		cps := zigUnwrap(t, page.LookupGrapheme(rac.cell))
		expectEqual(t, 2, len(cps))
		expectEqual(t, uint32(0x200D), cps[0])
		expectEqual(t, uint32(0x1F469), cps[1])
	}
	{
		rac := page.GetRowAndCell(3, 2)
		expect(t, rac.cell.hyperlink, "rac.cell.hyperlink")
		id := zigUnwrap(t, page.LookupHyperlink(rac.cell))
		entry := page.hyperlink_set.Get(id)
		expectEqualStrings(t, "https://example.com", string(entry.uri))
	}
}

// Zig: test "term51: compact: returns null when compacting would not save memory"
func TestPl4_term51_compact_returns_null_when_compacting_would_not_save_memory(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 24, pl4NoMax())
	defer s.Deinit()

	// Fill the page to its row capacity so the compacted page keeps all
	// of the rows, and make it need far more styles than a standard page
	// has so that it stays larger than the standard size when compacted.
	pl4LFillLast(t, s)
	node := s.pages.first
	for node.data.capacity.styles < 4096 {
		node = pl4IncreaseCapacity(t, s, node, PageListIncreaseCapacityStyles)
	}
	{
		page := &node.data
		for i := uint(0); i < 3000; i++ {
			id := pl4StylesAdd(t, page, Style{
				fg_color: StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{
					r: uint8(i & 0xFF),
					g: uint8(i >> 8),
					b: 7,
				}},
			})
			rac := page.GetRowAndCell(i%80, i/80)
			*rac.cell = PageCell{
				content_tag: PageCellContentTagCodepoint,
				content:     PageCellContent{codepoint: 'S'},
				style_id:    id,
			}
			rac.row.styled = true
		}
		expectEqual(t, uint(3000), page.styles.Count())
	}

	// The first compaction saves memory (graphemes, strings, spare style
	// capacity) but the result is still larger than a standard page.
	old_len := pageListMemoryLen(&node.data)
	compacted := pl4CompactNode(t, s, node)
	expect(t, pageListMemoryLen(&compacted.data) < old_len, "compacted.data.memory.len < old_len")
	expect(t, pageListMemoryLen(&compacted.data) > pageListStdSize(), "compacted.data.memory.len > std_size")

	// A second one has nothing to save: the capacity is already exact.
	page_size := s.page_size
	serial := s.page_serial
	expectEqual(t, (*PageListNode)(nil), pl4Compact(t, s, compacted))
	expectEqual(t, compacted, s.pages.first)
	expectEqual(t, compacted, s.pages.last)
	expectEqual(t, page_size, s.page_size)
	expectEqual(t, serial, s.page_serial)
	expectEqual(t, uint(3000), compacted.data.styles.Count())
}

// Zig: test "term51: split: a single row page cannot be split"
func TestPl4_term51_split_a_single_row_page_cannot_be_split(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 1, pl4NoMax())
	defer s.Deinit()
	serial := s.page_serial
	page_size := s.page_size

	// Even at row 0, which would otherwise be a no-op.
	pl4ExpectError(t, ErrOutOfSpace, s.Split(PageListPin{node: s.pages.first}))
	expectEqual(t, uint(1), s.TotalPages())
	expectEqual(t, serial, s.page_serial)
	expectEqual(t, page_size, s.page_size)
}

// Zig: test "term51: split: row 0 is a no-op"
func TestPl4_term51_split_row_0_is_a_no_op(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 10, pl4NoMax())
	defer s.Deinit()
	pl4LText(PageListPin{node: s.pages.first, y: 0}, 'A')
	serial := s.page_serial
	page_size := s.page_size

	zigTry(t, s.Split(PageListPin{node: s.pages.first, y: 0, x: 5}))
	expectEqual(t, uint(1), s.TotalPages())
	expectEqual(t, SizeCellCountInt(10), s.pages.first.data.size.rows)
	expectEqual(t, serial, s.page_serial)
	expectEqual(t, page_size, s.page_size)
	expectEqual(t, uint32('A'), s.pages.first.data.GetRowAndCell(0, 0).cell.content.codepoint)
}

// Zig: test "term51: split: rows from the pin down move to a new page"
func TestPl4_term51_split_rows_from_the_pin_down_move_to_a_new_page(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 10, pl4NoMax())
	defer s.Deinit()
	first := s.pages.first
	for y := uint(0); y < 10; y++ {
		pl4LText(PageListPin{node: first, y: SizeCellCountInt(y), x: 1}, uint32('a'+y))
	}
	serial := s.page_serial
	serial_min := s.page_serial_min

	zigTry(t, s.Split(PageListPin{node: first, y: 4, x: 3}))

	expectEqual(t, uint(2), s.TotalPages())
	second := s.pages.last
	expectEqual(t, first, s.pages.first)
	expectEqual(t, second, first.next)
	expectEqual(t, first, second.prev)
	expectEqual(t, uint(10), s.total_rows)
	expectEqual(t, SizeCellCountInt(4), first.data.size.rows)
	expectEqual(t, SizeCellCountInt(6), second.data.size.rows)
	expectEqual(t, first.data.capacity, second.data.capacity)
	expectEqual(t, first.data.size.cols, second.data.size.cols)

	// Moved rows are copied in order; the old copies are cleared.
	for y := uint(0); y < 4; y++ {
		expectEqual(t, uint32('a'+y), first.data.GetRowAndCell(1, y).cell.content.codepoint)
	}
	for y := uint(0); y < 6; y++ {
		expectEqual(t, uint32('a'+4+y), second.data.GetRowAndCell(1, y).cell.content.codepoint)
	}
	for y := 4; y < 10; y++ {
		row := &first.data.rows[y]
		for _, cell := range first.data.GetCells(row) {
			expect(t, cell.IsEmpty(), "cell.isEmpty()")
		}
	}

	// Serial numbers: the new page takes the next one.
	expectEqual(t, serial, second.serial)
	expectEqual(t, serial+1, s.page_serial)
	expectEqual(t, serial_min, s.page_serial_min)

	// The active area still ends at the last row.
	expectEqual(t, second, zigUnwrap(t, s.GetBottomRight(PointTagActive)).node)
}

// Zig: test "term51: split: row 1 and the last row"
func TestPl4_term51_split_row_1_and_the_last_row(t *testing.T) {
	testStdCapacity(t)

	{
		s := pl4Init(t, 80, 10, pl4NoMax())
		defer s.Deinit()
		zigTry(t, s.Split(PageListPin{node: s.pages.first, y: 1}))
		expectEqual(t, SizeCellCountInt(1), s.pages.first.data.size.rows)
		expectEqual(t, SizeCellCountInt(9), s.pages.last.data.size.rows)
	}
	{
		s := pl4Init(t, 80, 10, pl4NoMax())
		defer s.Deinit()
		zigTry(t, s.Split(PageListPin{node: s.pages.first, y: 9}))
		expectEqual(t, SizeCellCountInt(9), s.pages.first.data.size.rows)
		expectEqual(t, SizeCellCountInt(1), s.pages.last.data.size.rows)
	}
}

// Zig: test "term51: split: moves exactly the tracked pins at or after the split row"
func TestPl4_term51_split_moves_exactly_the_tracked_pins_at_or_after_the_split_row(t *testing.T) {
	testStdCapacity(t)
	s := pl4LInitPages(t, 100)
	defer s.Deinit()
	c := uint(s.pages.first.data.capacity.rows)
	first := s.pages.first
	second := first.next
	third := s.pages.last

	pin_first := pl4TrackPin(t, s, PageListPin{node: first, x: 9, y: 100})
	pin_before := pl4TrackPin(t, s, PageListPin{node: second, x: 1, y: 99})
	pin_at := pl4TrackPin(t, s, PageListPin{node: second, x: 2, y: 100})
	pin_after := pl4TrackPin(t, s, PageListPin{node: second, x: 3, y: 150})
	pin_end := pl4TrackPin(t, s, PageListPin{node: second, x: 79, y: SizeCellCountInt(c - 1)})
	pin_third := pl4TrackPin(t, s, PageListPin{node: third, x: 4, y: 100 - 1})
	pin_after.garbage = true

	// The viewport pin lives in the moved region too. Its absolute row
	// stays the same.
	s.Scroll(pl4ScrollPin(PageListPin{node: second, y: 200}))
	expectEqual(t, c+200, s.Scrollbar().offset)

	// Split at the position of a tracked pin. A fresh pin is built from its
	// fields: passing `pin_at.*` (even via a const copy) lets the argument
	// alias the tracked pin, which split then moves, corrupting the pins
	// handled after it (the integrity check fires).
	split_at := PageListPin{node: pin_at.node, y: pin_at.y, x: pin_at.x}
	zigTry(t, s.Split(split_at))

	new := second.next
	expectEqual(t, new, third.prev)
	expectEqual(t, SizeCellCountInt(100), second.data.size.rows)
	expectEqual(t, SizeCellCountInt(c-100), new.data.size.rows)

	expectEqual(t, first, pin_first.node)
	expectEqual(t, SizeCellCountInt(100), pin_first.y)
	expectEqual(t, second, pin_before.node)
	expectEqual(t, SizeCellCountInt(99), pin_before.y)
	expectEqual(t, SizeCellCountInt(1), pin_before.x)

	expectEqual(t, new, pin_at.node)
	expectEqual(t, SizeCellCountInt(0), pin_at.y)
	expectEqual(t, SizeCellCountInt(2), pin_at.x)
	expectEqual(t, new, pin_after.node)
	expectEqual(t, SizeCellCountInt(50), pin_after.y)
	expectEqual(t, SizeCellCountInt(3), pin_after.x)
	expect(t, pin_after.garbage, "pin_after.garbage")
	expectEqual(t, new, pin_end.node)
	expectEqual(t, SizeCellCountInt(c-1-100), pin_end.y)
	expectEqual(t, SizeCellCountInt(79), pin_end.x)
	expectEqual(t, third, pin_third.node)
	expectEqual(t, SizeCellCountInt(99), pin_third.y)

	expectEqual(t, new, s.viewport_pin.node)
	expectEqual(t, SizeCellCountInt(100), s.viewport_pin.y)
	expect(t, s.viewport.Tag == PageListViewportPin, "s.viewport == .pin")
	expectEqual(t, c+200, s.Scrollbar().offset)
}

// Zig: test "term51: split: position in the list"
func TestPl4_term51_split_position_in_the_list(t *testing.T) {
	testStdCapacity(t)
	s := pl4LInitPages(t, 100)
	defer s.Deinit()
	first := s.pages.first
	second := first.next
	third := s.pages.last

	// Split the first page: new page sits between first and second.
	zigTry(t, s.Split(PageListPin{node: first, y: 100}))
	a := first.next
	expect(t, a != second, "a != second")
	expectEqual(t, second, a.next)
	expectEqual(t, a, second.prev)
	expectEqual(t, first, s.pages.first)
	expectEqual(t, third, s.pages.last)

	// Split the last page: the new page becomes the last.
	zigTry(t, s.Split(PageListPin{node: third, y: 50}))
	b := s.pages.last
	expect(t, b != third, "b != third")
	expectEqual(t, third, b.prev)
	expectEqual(t, b, third.next)
	expectEqual(t, (*PageListNode)(nil), b.next)
	expectEqual(t, SizeCellCountInt(50), third.data.size.rows)
	expectEqual(t, SizeCellCountInt(50), b.data.size.rows)
	expectEqual(t, uint(5), s.TotalPages())
	expectEqual(t, 2*uint(first.data.capacity.rows)+100, s.total_rows)
}

// Zig: test "term51: split: row metadata is copied"
func TestPl4_term51_split_row_metadata_is_copied(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 10, pl4NoMax())
	defer s.Deinit()
	first := s.pages.first

	{
		rac := first.data.GetRowAndCell(0, 5)
		rac.row.wrap = true
		rac.row.semantic_prompt = PageRowSemanticPromptPrompt
		rac2 := first.data.GetRowAndCell(0, 6)
		rac2.row.wrap_continuation = true
		rac2.row.semantic_prompt = PageRowSemanticPromptPromptContinuation
	}

	zigTry(t, s.Split(PageListPin{node: first, y: 5}))
	second := s.pages.last

	expect(t, second.data.GetRowAndCell(0, 0).row.wrap, "second.data.getRowAndCell(0, 0).row.wrap")
	expectEqual(t, PageRowSemanticPromptPrompt, second.data.GetRowAndCell(0, 0).row.semantic_prompt)
	expect(t, !second.data.GetRowAndCell(0, 0).row.wrap_continuation, "!second.data.getRowAndCell(0, 0).row.wrap_continuation")
	expect(t, second.data.GetRowAndCell(0, 1).row.wrap_continuation, "second.data.getRowAndCell(0, 1).row.wrap_continuation")
	expectEqual(t, PageRowSemanticPromptPromptContinuation, second.data.GetRowAndCell(0, 1).row.semantic_prompt)
}

// Zig: test "term51: split: styles graphemes and hyperlinks move with their rows"
func TestPl4_term51_split_styles_graphemes_and_hyperlinks_move_with_their_rows(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 10, 10, pl4NoMax())
	defer s.Deinit()
	first := s.pages.first
	page := &first.data

	// A style used above and below the split: both pages need it.
	bold := pl4StylesAdd(t, page, pl4Bold())
	page.styles.Use(bold)
	for _, y := range []uint{2, 7} {
		rac := page.GetRowAndCell(0, y)
		*rac.cell = PageCell{
			content_tag: PageCellContentTagCodepoint,
			content:     PageCellContent{codepoint: 'S'},
			style_id:    bold,
		}
		rac.row.styled = true
	}

	// Grapheme and hyperlink below only.
	{
		rac := page.GetRowAndCell(1, 8)
		*rac.cell = PageCell{
			content_tag: PageCellContentTagCodepoint,
			content:     PageCellContent{codepoint: 0x1F468},
		}
		zigTry(t, page.SetGraphemes(rac.row, rac.cell, []uint32{0x200D}))
	}
	link := pl4InsertHyperlink(t, page, Hyperlink{
		id:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: 0},
		uri: []uint8("https://example.com"),
	})
	{
		rac := page.GetRowAndCell(2, 9)
		*rac.cell = PageCell{
			content_tag: PageCellContentTagCodepoint,
			content:     PageCellContent{codepoint: 'L'},
		}
		zigTry(t, page.SetHyperlink(rac.row, rac.cell, link))
	}

	zigTry(t, s.Split(PageListPin{node: first, y: 5}))
	second := s.pages.last

	expectEqual(t, uint(1), first.data.styles.Count())
	expectEqual(t, uint(1), second.data.styles.Count())
	expect(t, first.data.styles.Get(first.data.GetRowAndCell(0, 2).cell.style_id).flags.bold, "first.data.styles.get(first.data.memory, first.data.getRowAndCell(0, 2).cell.style_id).flags.bold")
	expect(t, second.data.styles.Get(second.data.GetRowAndCell(0, 2).cell.style_id).flags.bold, "second.data.styles.get(second.data.memory, second.data.getRowAndCell(0, 2).cell.style_id).flags.bold")

	expectEqual(t, uint(0), first.data.GraphemeCount())
	expectEqual(t, uint(1), second.data.GraphemeCount())
	expect(t, second.data.GetRowAndCell(1, 3).row.grapheme, "second.data.getRowAndCell(1, 3).row.grapheme")
	expectEqual(t, uint(0), first.data.hyperlink_set.Count())
	expectEqual(t, uint(1), second.data.hyperlink_set.Count())
	expect(t, second.data.GetRowAndCell(2, 4).cell.hyperlink, "second.data.getRowAndCell(2, 4).cell.hyperlink")
}

// Zig: test "term51: split: non-standard page keeps its capacity"
func TestPl4_term51_split_non_standard_page_keeps_its_capacity(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 10, pl4NoMax())
	defer s.Deinit()
	node := pl4LMakeOversized(t, s, s.pages.first)
	page_size := s.page_size
	len_ := pageListMemoryLen(&node.data)

	zigTry(t, s.Split(PageListPin{node: node, y: 3}))
	second := s.pages.last
	expectEqual(t, node.data.capacity, second.data.capacity)
	expectEqual(t, len_, pageListMemoryLen(&second.data))
	expectEqual(t, SizeCellCountInt(3), node.data.size.rows)
	expectEqual(t, SizeCellCountInt(7), second.data.size.rows)

	// The new page is not pooled, so it adds its exact length.
	expectEqual(t, page_size+len_, s.page_size)
}

// Zig: test "term51: split: layout page_size accounting for a standard page"
func TestPl4_term51_split_layout_page_size_accounting_for_a_standard_page(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 10, pl4NoMax())
	defer s.Deinit()
	page_size := s.page_size
	zigTry(t, s.Split(PageListPin{node: s.pages.first, y: 5}))
	expectEqual(t, page_size+pageListStdSize(), s.page_size)
}

// Zig: test "term51: Scrollbar.eql: compares every field"
func TestPl4_term51_Scrollbar_eql_compares_every_field(t *testing.T) {
	testStdCapacity(t)
	a := PageListScrollbar{total: 10, offset: 3, len: 5}
	expect(t, a.Eql(a), "a.eql(a)")
	expect(t, a.Eql(PageListScrollbar{total: 10, offset: 3, len: 5}), "a.eql(.{ .total = 10, .offset = 3, .len = 5 })")
	expect(t, !a.Eql(PageListScrollbar{total: 11, offset: 3, len: 5}), "!a.eql(.{ .total = 11, .offset = 3, .len = 5 })")
	expect(t, !a.Eql(PageListScrollbar{total: 10, offset: 4, len: 5}), "!a.eql(.{ .total = 10, .offset = 4, .len = 5 })")
	expect(t, !a.Eql(PageListScrollbar{total: 10, offset: 3, len: 6}), "!a.eql(.{ .total = 10, .offset = 3, .len = 6 })")
	expect(t, !a.Eql(PageListScrollbar{total: 3, offset: 10, len: 5}), "!a.eql(.{ .total = 3, .offset = 10, .len = 5 })")
	expect(t, !a.Eql(PageListScrollbarZero), "!a.eql(.zero)")
	expect(t, PageListScrollbarZero.Eql(PageListScrollbar{total: 0, offset: 0, len: 0}), "Scrollbar.zero.eql(.{ .total = 0, .offset = 0, .len = 0 })")

	// Symmetric.
	b := PageListScrollbar{total: 10, offset: 4, len: 5}
	expect(t, !b.Eql(a), "!b.eql(a)")
}

// Zig: test "term51: scrollbar: no scrollback reports only the active area"
func TestPl4_term51_scrollbar_no_scrollback_reports_only_the_active_area(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 24, pl4Max(0))
	defer s.Deinit()

	want := PageListScrollbar{total: 24, offset: 0, len: 24}
	expect(t, s.Scrollbar().Eql(want), "s.scrollbar().eql(want)")

	// Rows grown by the page layout are hidden.
	pl4LGrowRows(t, s, 10)
	expectEqual(t, uint(34), s.total_rows)
	expect(t, s.Scrollbar().Eql(want), "s.scrollbar().eql(want)")

	// Whatever the viewport says.
	s.viewport = pl4Viewport(PageListViewportTop)
	expect(t, s.Scrollbar().Eql(want), "s.scrollbar().eql(want)")
	s.viewport = pl4Viewport(PageListViewportActive)
}

// Zig: test "term51: scrollbar: reports total offset and rows for each viewport"
func TestPl4_term51_scrollbar_reports_total_offset_and_rows_for_each_viewport(t *testing.T) {
	testStdCapacity(t)
	s := pl4LInitPages(t, 100)
	defer s.Deinit()
	total := s.total_rows

	// Even a tiny non-zero max size is not the no-scrollback special case.
	s.explicit_max_size = 1

	expect(t, s.Scrollbar().Eql(PageListScrollbar{total: total, offset: total - 24, len: 24}), "s.scrollbar().eql(.{ .total = total, .offset = total - 24, .len = 24 })")

	s.Scroll(PageListScroll{Tag: PageListScrollTop})
	expect(t, s.Scrollbar().Eql(PageListScrollbar{total: total, offset: 0, len: 24}), "s.scrollbar().eql(.{ .total = total, .offset = 0, .len = 24 })")

	s.Scroll(pl4ScrollRow(500))
	expect(t, s.Scrollbar().Eql(PageListScrollbar{total: total, offset: 500, len: 24}), "s.scrollbar().eql(.{ .total = total, .offset = 500, .len = 24 })")

	s.Scroll(pl4ScrollPin(pl4Pin(t, s, pl4ScreenPt(0, 900))))
	expect(t, s.Scrollbar().Eql(PageListScrollbar{total: total, offset: 900, len: 24}), "s.scrollbar().eql(.{ .total = total, .offset = 900, .len = 24 })")
	s.explicit_max_size = math.MaxUint64
}

// Zig: test "term51: scrollbar: unscrolled screen fits exactly"
func TestPl4_term51_scrollbar_unscrolled_screen_fits_exactly(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 24, pl4NoMax())
	defer s.Deinit()
	expect(t, s.Scrollbar().Eql(PageListScrollbar{total: 24, offset: 0, len: 24}), "s.scrollbar().eql(.{ .total = 24, .offset = 0, .len = 24 })")
}

// Go-only: panic test: resize: cols is zero (assert(v > 0) under runtime safety)
func TestPl4_panic_resize_cols_is_zero(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 24, pl4NoMax())
	defer s.Deinit()
	pl4ExpectPanic(t, func() {
		_ = s.Resize(PageListResize{cols: pl4S(0), reflow: true})
	})
	pl4ExpectPanic(t, func() {
		_ = s.Resize(PageListResize{cols: pl4S(0), reflow: false})
	})
}

// Go-only: panic test: resize: rows is zero (assert(v > 0) under runtime safety)
func TestPl4_panic_resize_rows_is_zero(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 24, pl4NoMax())
	defer s.Deinit()
	pl4ExpectPanic(t, func() {
		_ = s.Resize(PageListResize{rows: pl4S(0), reflow: true})
	})
	pl4ExpectPanic(t, func() {
		_ = s.Resize(PageListResize{rows: pl4S(0), reflow: false})
	})
}

// Go-only: panic test: resizeCols: cols equals the current column count (assert(cols != self.cols))
func TestPl4_panic_resizeCols_cols_equals_the_current_column_count(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 24, pl4NoMax())
	defer s.Deinit()
	pl4ExpectPanic(t, func() {
		_ = s.resizeCols(80, None[PageListResizeCursor]())
	})
}

// Go-only: panic test: resizeWithoutReflowGrowCols: cols is not greater than the current column count (assert(cols > self.cols))
func TestPl4_panic_resizeWithoutReflowGrowCols_cols_is_not_greater_than_the_current_column_count(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 24, pl4NoMax())
	defer s.Deinit()
	chunk := pl4KFirstChunk(t, s)
	// Equal.
	pl4ExpectPanic(t, func() {
		_ = s.resizeWithoutReflowGrowCols(80, chunk)
	})
	// Smaller.
	pl4ExpectPanic(t, func() {
		_ = s.resizeWithoutReflowGrowCols(40, chunk)
	})
}

// Go-only: panic test: scroll: delta_row of the smallest isize from active (the negation overflows)
func TestPl4_panic_scroll_delta_row_of_the_smallest_isize_from_active(t *testing.T) {
	testStdCapacity(t)
	s := pl4LInitPages(t, 100)
	defer s.Deinit()
	pl4ExpectPanic(t, func() {
		s.Scroll(pl4ScrollDeltaRow(math.MinInt64))
	})
}

// Go-only: panic test: scroll: delta_row of the smallest isize from a pin viewport (the negation overflows)
func TestPl4_panic_scroll_delta_row_of_the_smallest_isize_from_a_pin_viewport(t *testing.T) {
	testStdCapacity(t)
	s := pl4LInitPages(t, 100)
	defer s.Deinit()
	pl4LPinAt(t, s, 100)
	pl4ExpectPanic(t, func() {
		s.Scroll(pl4ScrollDeltaRow(math.MinInt64))
	})
}

// Go-only: panic test: scrollPrompt: delta of the smallest isize (the negation overflows)
func TestPl4_panic_scrollPrompt_delta_of_the_smallest_isize(t *testing.T) {
	testStdCapacity(t)
	s := pl4LPromptList(t)
	defer s.Deinit()
	pl4ExpectPanic(t, func() {
		s.scrollPrompt(math.MinInt64)
	})
}

// Go-only: panic test: compact: page without rows (assert(page.size.rows > 0))
func TestPl4_panic_compact_page_without_rows(t *testing.T) {
	testStdCapacity(t)
	s := pl4Init(t, 80, 24, pl4NoMax())
	defer s.Deinit()
	node := s.pages.first
	rows := node.data.size.rows
	node.data.size.rows = 0
	pl4ExpectPanic(t, func() {
		_, _ = s.Compact(node)
	})
	node.data.size.rows = rows
}

// Go-only: panic test: split: pin row is beyond the rows of its page (assert(self.pinIsValid(p)))
func TestPl4_panic_split_pin_row_is_beyond_the_rows_of_its_page(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	testStdCapacity(t)
	s := pl4Init(t, 80, 10, pl4NoMax())
	defer s.Deinit()
	pl4ExpectPanic(t, func() {
		_ = s.Split(PageListPin{node: s.pages.first, y: 10})
	})
}

// Go-only: panic test: split: pin column is beyond the columns of its page (assert(self.pinIsValid(p)))
func TestPl4_panic_split_pin_column_is_beyond_the_columns_of_its_page(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	testStdCapacity(t)
	s := pl4Init(t, 80, 10, pl4NoMax())
	defer s.Deinit()
	pl4ExpectPanic(t, func() {
		_ = s.Split(PageListPin{node: s.pages.first, y: 5, x: 80})
	})
}

// Go-only: panic test: split: pin node is not in the list (assert(self.pinIsValid(p)))
func TestPl4_panic_split_pin_node_is_not_in_the_list(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	testStdCapacity(t)
	s := pl4Init(t, 80, 10, pl4NoMax())
	defer s.Deinit()
	other := pl4Init(t, 80, 10, pl4NoMax())
	defer other.Deinit()
	pl4ExpectPanic(t, func() {
		_ = s.Split(PageListPin{node: other.pages.first, y: 5})
	})
}
