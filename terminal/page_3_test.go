// Tests translated one for one from src/terminal/page.zig (Ghostty v1.3.1 plus
// the term51 additional tests), from `test "term51: Page.swapCells: plain
// cells can be swapped between rows"` to the end of the file. Each Go test
// carries the exact Zig title in a "// Zig: test" comment.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual keeps Zig's (expected, actual) argument order.
//   - Zig's helpers term51H* and term51I* are the unexported pg3H* and pg3I*
//     functions below. The page helpers return a *Page (Zig's `&page`);
//     errors the Zig helpers propagate with `try` stop the test (zigTry).
//   - `page.memory` is gone (Go memory): `defer page.deinit()` becomes
//     `defer page.Deinit()`, `x.ptr(page.memory)[i]` is the slice `page.rows`
//     or the result of GetCells, and the `base` argument of every set function
//     is dropped. `try page.verifyIntegrity(std.testing.allocator)` is
//     `zigTry(t, page.VerifyIntegrity())`.
//   - Zig-only: `const alloc = testing.allocator;`, `defer alloc.free(...)` and
//     the allocations that only exist to build an input are allocator
//     bookkeeping with no Go counterpart; they are dropped without a
//     per-test note.
//   - `@as(?T, null)` comparisons are `!x.Valid`: Optional of a slice is not
//     comparable with ==.
//   - `@bitCast` of a Cell into a u64 is PageCell.bitCast; `@bitCast` of a u64
//     into a Cell (term51IRaw) is pageCellFromBits. Likewise PageRow.bitCast
//     and pageRowFromBits for a Row.
//   - Zig-only: lines that compare pointers into page memory with the memory
//     range (`@intFromPtr(p) >= base` and the like) have no Go counterpart and
//     are dropped with a comment. Pointer identity between two slices of the
//     same page (`cps.ptr == ptr`) is kept as the identity of their first
//     elements.

package terminal

import (
	"errors"
	"math/bits"
	"testing"
)

// pg3ExpectError is Zig's `try testing.expectError(expected, actual)`.
func pg3ExpectError(t *testing.T, want error, got error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

// pg3ExpectPanic runs f and fails unless it panics. The wireframe's
// not-implemented stub does not count: a test that only reaches the stub must
// fail until the implementation panics for the right reason.
func pg3ExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("expected a panic, got none")
		}
		if s, ok := r.(string); ok && s == notImplemented {
			t.Fatalf("panicked with the not-implemented stub, not with the expected panic")
		}
	}()
	f()
}

// pg3AlignForward is std.mem.alignForward(u64, n, alignment).
func pg3AlignForward(n uint64, alignment uint64) uint64 {
	return (n + alignment - 1) / alignment * alignment
}

// pg3PageSizeMin is std.heap.page_size_min as recorded (4096).
const pg3PageSizeMin = 4096

// pg3Init is `try Page.init(cap)` and the address of the result.
func pg3Init(t *testing.T, cap PageCapacity) *Page {
	t.Helper()
	p, err := PageInit(cap)
	zigTry(t, err)
	return &p
}

// pg3StdCapacity is std_capacity under `builtin.is_test`: the wireframe's
// PageStdCapacity is the production value (8192 grapheme bytes), the unit
// tests ran with 512 (see the comment on PageStdCapacity).
func pg3StdCapacity() PageCapacity {
	c := PageStdCapacity
	c.grapheme_bytes = 512
	return c
}

// pg3MaxedCapacity is the Capacity with every field at the maximum of its
// type (`@field(cap, field.name) = std.math.maxInt(field.type)`).
func pg3MaxedCapacity() PageCapacity {
	return PageCapacity{
		cols:            0xFFFF,
		rows:            0xFFFF,
		styles:          0xFFFF,
		hyperlink_bytes: 0xFFFF,
		grapheme_bytes:  0xFFFFFFFF,
		string_bytes:    0xFFFFFFFF,
	}
}

// ---- term51H helpers ---------------------------------------------------------

// pg3HBits is term51HBits.
func pg3HBits(cell PageCell) uint64 {
	return cell.bitCast()
}

// pg3HNew is term51HNew: a page with room for 5 distinct styles and 5
// distinct hyperlinks.
func pg3HNew(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) *Page {
	t.Helper()
	cap := PageCapacityInit(cols, rows)
	cap.styles = 8
	cap.hyperlink_bytes = 8 * hyperlinkSetItemSizeOf
	return pg3Init(t, cap)
}

// pg3HFillRow is term51HFillRow: write the codepoint `base + x` into every
// cell of row y.
func pg3HFillRow(page *Page, y uint, base uint32) {
	for x := uint(0); x < uint(page.size.cols); x++ {
		*page.GetRowAndCell(x, y).cell = PageCellInit(base + uint32(x))
	}
}

// pg3HGrapheme is term51HGrapheme: write a cell with `cp` and the extra
// grapheme codepoints.
func pg3HGrapheme(t *testing.T, page *Page, x uint, y uint, cp uint32, extra []uint32) {
	t.Helper()
	rac := page.GetRowAndCell(x, y)
	*rac.cell = PageCellInit(cp)
	for _, e := range extra {
		zigTry(t, page.AppendGrapheme(rac.row, rac.cell, e))
	}
}

// pg3HStyleCells is term51HStyleCells: add a style (distinguished by palette
// fg n) and put it on the cells. After this the style's reference count is
// exactly len(cells).
func pg3HStyleCells(t *testing.T, page *Page, n uint8, cells [][2]uint) StyleID {
	t.Helper()
	id, err := page.styles.Add(Style{FgColor: StyleColor{Tag: StyleColorPalette, Palette: n}})
	zigTry(t, err)
	for _, c := range cells {
		rac := page.GetRowAndCell(c[0], c[1])
		rac.row.styled = true
		rac.cell.setStyleID(id)
		page.styles.Use(id)
	}
	page.styles.Release(id)
	return id
}

// pg3HLinkCells is term51HLinkCells: insert a hyperlink and put it on the
// cells. After this the link's reference count is exactly len(cells).
func pg3HLinkCells(t *testing.T, page *Page, link Hyperlink, cells [][2]uint) HyperlinkID {
	t.Helper()
	id, err := page.InsertHyperlink(link)
	zigTry(t, err)
	for _, c := range cells {
		rac := page.GetRowAndCell(c[0], c[1])
		page.hyperlink_set.Use(id)
		zigTry(t, page.SetHyperlink(rac.row, rac.cell, id))
	}
	page.hyperlink_set.Release(id)
	return id
}

// pg3HLink is term51HLink.
func pg3HLink(uri string) Hyperlink {
	return Hyperlink{
		ID:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: 0},
		URI: []uint8(uri),
	}
}

// pg3HFillAll is term51HFillAll: fill every row of a page with distinct text.
func pg3HFillAll(page *Page) {
	for y := uint(0); y < uint(page.size.rows); y++ {
		pg3HFillRow(page, y, uint32(0x100+y*0x20))
	}
}

// pg3HExpectCps is term51HExpectCps.
func pg3HExpectCps(t *testing.T, page *Page, y uint, expected []uint32) {
	t.Helper()
	expectEqual(t, uint(page.size.cols), uint(len(expected)))
	cells := page.GetCells(page.GetRow(y))
	for i, e := range expected {
		expectEqual(t, e, cells[i].Codepoint())
	}
}

// ---- term51I helpers ---------------------------------------------------------

// pg3ICap is term51ICap: 10x10 cells, 8 styles and the default hyperlink
// (192 bytes), grapheme (1024 bytes) and string (2048 bytes) capacities.
func pg3ICap() PageCapacity {
	c := PageCapacityInit(10, 10)
	c.styles = 8
	return c
}

// pg3IZeroMeta is term51IZeroMeta: a capacity with every managed-memory
// capacity set to zero.
func pg3IZeroMeta() PageCapacity {
	return PageCapacity{
		cols:            1,
		rows:            1,
		styles:          0,
		hyperlink_bytes: 0,
		grapheme_bytes:  0,
		string_bytes:    0,
	}
}

// pg3IItem is term51IItem: the size in bytes of one hyperlink set item.
// hyperlink_bytes is converted to a count of set items with this size.
const pg3IItem = hyperlinkSetItemSizeOf

// pg3IRaw is term51IRaw: `@bitCast` of a u64 to a Cell.
func pg3IRaw(bits uint64) PageCell {
	return pageCellFromBits(bits)
}

// pg3IBits is term51IBits.
func pg3IBits(cell PageCell) uint64 {
	return cell.bitCast()
}

// pg3ITagged is term51ITagged: a cell built from raw bits: the 2 tag bits and
// the 24 content bits. Everything else is zero.
func pg3ITagged(tag PageCellContentTag, content uint32) PageCell {
	return pg3IRaw(uint64(tag) | (uint64(content) << 2))
}

// pg3IBlank is term51IBlank: a default cell built at runtime the way
// Screen.blankCell returns it (`return .{}`).
func pg3IBlank() PageCell {
	return PageCell(0)
}

// pg3ISpelled is term51ISpelled: a cell with every default spelled out.
func pg3ISpelled() PageCell {
	return pageCellFields{
		content_tag:      PageCellContentTagCodepoint,
		content:          pageCellContentFields{codepoint: 0}.pack(),
		style_id:         0,
		wide:             PageCellWideNarrow,
		protected:        false,
		hyperlink:        false,
		semantic_content: PageCellSemanticContentOutput,
	}.pack()
}

// pg3IBgPalette is term51IBgPalette: a background-only cell built by struct
// literal.
func pg3IBgPalette(index uint8) PageCell {
	return pageCellFields{content_tag: PageCellContentTagBgColorPalette, content: pageCellContentFields{color_palette: index}.pack()}.pack()
}

// pg3IBgRgb is term51IBgRgb.
func pg3IBgRgb(r uint8, g uint8, b uint8) PageCell {
	return pageCellFields{
		content_tag: PageCellContentTagBgColorRgb,
		content:     pageCellContentFields{color_rgb: PageCellRGB{R: r, G: g, B: b}}.pack(),
	}.pack()
}

// pg3IRC is term51IRC.
type pg3IRC struct {
	row  *PageRow
	cell *PageCell
}

// pg3IPut is term51IPut: write a plain text cell at x,y and return it with
// its row.
func pg3IPut(page *Page, x uint, y uint, cp uint32) pg3IRC {
	rac := page.GetRowAndCell(x, y)
	*rac.cell = PageCellInit(cp)
	return pg3IRC{row: rac.row, cell: rac.cell}
}

// pg3ILinkErr is term51ILink: insert an implicit-id hyperlink (ref count 1 on
// success), returning the error for the tests that inspect it.
func pg3ILinkErr(page *Page, n uint32) (HyperlinkID, error) {
	return page.InsertHyperlink(Hyperlink{
		ID:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: n},
		URI: []uint8("https://example.com"),
	})
}

// pg3ILink is `try term51ILink(page, n)`.
func pg3ILink(t *testing.T, page *Page, n uint32) HyperlinkID {
	t.Helper()
	id, err := pg3ILinkErr(page, n)
	zigTry(t, err)
	return id
}

// pg3IRefs is term51IRefs.
func pg3IRefs(page *Page, id HyperlinkID) uint {
	return uint(page.hyperlink_set.RefCount(id))
}

// pg3IStrUsed is term51IStrUsed.
func pg3IStrUsed(page *Page) uint64 {
	return page.string_alloc.UsedBytes()
}

// pg3IGrUsed is term51IGrUsed.
func pg3IGrUsed(page *Page) uint64 {
	return page.grapheme_alloc.UsedBytes()
}

// pg3IChunk is term51IChunk: grapheme memory is allocated in chunks of 4
// codepoints (16 bytes).
const pg3IChunk = 16

// ---- Page.swapCells ----------------------------------------------------------

// Zig: test "term51: Page.swapCells: plain cells can be swapped between rows"
func TestPg3_Page_swapCells_plain_cells_can_be_swapped_between_rows(t *testing.T) {
	page := pg3HNew(t, 4, 2)
	defer page.Deinit()
	*page.GetRowAndCell(1, 0).cell = PageCellInit('a')
	*page.GetRowAndCell(2, 1).cell = PageCellInit('b')
	page.SwapCells(page.GetRowAndCell(1, 0).cell, page.GetRowAndCell(2, 1).cell)
	expectEqual(t, uint32('b'), page.GetRowAndCell(1, 0).cell.Codepoint())
	expectEqual(t, uint32('a'), page.GetRowAndCell(2, 1).cell.Codepoint())
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.swapCells: does not change row flags"
func TestPg3_Page_swapCells_does_not_change_row_flags(t *testing.T) {
	page := pg3HNew(t, 6, 1)
	defer page.Deinit()
	pg3HGrapheme(t, page, 1, 0, 'a', []uint32{0x301})
	pg3HStyleCells(t, page, 1, [][2]uint{{1, 0}})
	row := page.GetRow(0)
	row.wrap = true
	// term51HRowBits(row.*) compares all 64 bits of the row; PageRow is
	// compared field by field, which is the same comparison.
	flags := *row
	page.SwapCells(page.GetRowAndCell(1, 0).cell, page.GetRowAndCell(3, 0).cell)
	expectEqual(t, flags, *row)
}

// ---- Page.clearCells ---------------------------------------------------------

// Zig: test "term51: Page.clearCells: plain full row is zeroed"
func TestPg3_Page_clearCells_plain_full_row_is_zeroed(t *testing.T) {
	page := pg3HNew(t, 6, 2)
	defer page.Deinit()
	pg3HFillAll(page)
	cells := page.GetCells(page.GetRow(0))
	for i := range cells {
		c := &cells[i]
		c.setProtected(true)
		c.setSemanticContent(PageCellSemanticContentInput)
		c.setWide(PageCellWideNarrow)
	}
	*page.GetRowAndCell(1, 0).cell = pageCellFields{content_tag: PageCellContentTagBgColorRgb, content: pageCellContentFields{color_rgb: PageCellRGB{R: 9, G: 9, B: 9}}.pack()}.pack()
	*page.GetRowAndCell(2, 0).cell = pageCellFields{content_tag: PageCellContentTagBgColorPalette, content: pageCellContentFields{color_palette: 3}.pack()}.pack()

	page.ClearCells(page.GetRow(0), 0, 6)

	for _, c := range page.GetCells(page.GetRow(0)) {
		expect(t, c.IsZero(), "c.isZero()")
	}
	// Other rows untouched.
	pg3HExpectCps(t, page, 1, []uint32{0x120, 0x121, 0x122, 0x123, 0x124, 0x125})
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.clearCells: ranges are half open and bounded"
func TestPg3_Page_clearCells_ranges_are_half_open_and_bounded(t *testing.T) {
	page := pg3HNew(t, 8, 1)
	defer page.Deinit()
	row := page.GetRow(0)
	full := []uint32{0x100, 0x101, 0x102, 0x103, 0x104, 0x105, 0x106, 0x107}

	pg3HFillRow(page, 0, 0x100)
	page.ClearCells(row, 2, 5)
	pg3HExpectCps(t, page, 0, []uint32{0x100, 0x101, 0, 0, 0, 0x105, 0x106, 0x107})

	pg3HFillRow(page, 0, 0x100)
	page.ClearCells(row, 0, 1)
	pg3HExpectCps(t, page, 0, []uint32{0, 0x101, 0x102, 0x103, 0x104, 0x105, 0x106, 0x107})

	pg3HFillRow(page, 0, 0x100)
	page.ClearCells(row, 7, 8)
	pg3HExpectCps(t, page, 0, []uint32{0x100, 0x101, 0x102, 0x103, 0x104, 0x105, 0x106, 0})

	pg3HFillRow(page, 0, 0x100)
	page.ClearCells(row, 3, 3)
	pg3HExpectCps(t, page, 0, full)
	page.ClearCells(row, 8, 8)
	pg3HExpectCps(t, page, 0, full)
}

// Zig: test "term51: Page.clearCells: graphemes of a full row are freed and the row flag cleared"
func TestPg3_Page_clearCells_graphemes_of_a_full_row_are_freed_and_the_row_flag_cleared(t *testing.T) {
	page := pg3HNew(t, 10, 2)
	defer page.Deinit()
	pg3HFillAll(page)
	pg3HGrapheme(t, page, 0, 0, 'a', []uint32{0x301})
	pg3HGrapheme(t, page, 5, 0, 'b', []uint32{1, 2, 3, 4, 5})
	pg3HGrapheme(t, page, 1, 1, 'c', []uint32{0x301})

	page.ClearCells(page.GetRow(0), 0, 10)

	expectEqual(t, uint(1), page.GraphemeCount())
	expectEqual(t, uint64(16), page.grapheme_alloc.UsedBytes())
	expect(t, !page.GetRow(0).grapheme, "!page.getRow(0).grapheme")
	expect(t, page.GetRow(1).grapheme, "page.getRow(1).grapheme")
	expect(t, page.GetRowAndCell(0, 0).cell.IsZero(), "page.getRowAndCell(0, 0).cell.isZero()")
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.clearCells: a partial range keeps the grapheme flag while others remain"
func TestPg3_Page_clearCells_a_partial_range_keeps_the_grapheme_flag_while_others_remain(t *testing.T) {
	page := pg3HNew(t, 10, 1)
	defer page.Deinit()
	pg3HFillAll(page)
	pg3HGrapheme(t, page, 1, 0, 'a', []uint32{0x301})
	pg3HGrapheme(t, page, 8, 0, 'b', []uint32{0x301})
	row := page.GetRow(0)

	page.ClearCells(row, 0, 5)
	expectEqual(t, uint(1), page.GraphemeCount())
	expect(t, row.grapheme, "row.grapheme")
	expect(t, !page.GetRowAndCell(1, 0).cell.HasGrapheme(), "!page.getRowAndCell(1, 0).cell.hasGrapheme()")
	expect(t, page.GetRowAndCell(8, 0).cell.HasGrapheme(), "page.getRowAndCell(8, 0).cell.hasGrapheme()")

	// Clearing the last one with another partial range drops the flag.
	page.ClearCells(row, 6, 10)
	expectEqual(t, uint(0), page.GraphemeCount())
	expect(t, !row.grapheme, "!row.grapheme")
	expectEqual(t, uint64(0), page.grapheme_alloc.UsedBytes())
}

// Zig: test "term51: Page.clearCells: hyperlinks are released and unmapped"
func TestPg3_Page_clearCells_hyperlinks_are_released_and_unmapped(t *testing.T) {
	page := pg3HNew(t, 10, 2)
	defer page.Deinit()
	pg3HFillAll(page)
	a := pg3HLinkCells(t, page, pg3HLink("https://a"), [][2]uint{{1, 0}, {2, 0}, {8, 0}})
	b := pg3HLinkCells(t, page, pg3HLink("https://b"), [][2]uint{{3, 0}, {4, 1}})
	row := page.GetRow(0)

	page.ClearCells(row, 0, 5)

	expectEqual(t, uint16(1), page.hyperlink_set.RefCount(a))
	expectEqual(t, uint16(1), page.hyperlink_set.RefCount(b))
	expectEqual(t, uint(2), page.HyperlinkCount())
	expect(t, !page.LookupHyperlink(page.GetRowAndCell(1, 0).cell).Valid, "page.lookupHyperlink(page.getRowAndCell(1, 0).cell) == null")
	expect(t, row.hyperlink, "row.hyperlink") // (8,0) remains
	expectEqual(t, uint(2), page.hyperlink_set.Count())

	page.ClearCells(row, 5, 10)
	expect(t, !row.hyperlink, "!row.hyperlink")
	expectEqual(t, uint(1), page.HyperlinkCount())
	expectEqual(t, uint16(0), page.hyperlink_set.RefCount(a))
	expectEqual(t, uint(1), page.hyperlink_set.Count())
	// Other row untouched.
	expect(t, page.GetRow(1).hyperlink, "page.getRow(1).hyperlink")
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.clearCells: styles are released and the styled flag follows the remaining cells with a stale quirk"
func TestPg3_Page_clearCells_styles_are_released_and_the_styled_flag_follows_the_remaining_cells_with_a_stale_quirk(t *testing.T) {
	page := pg3HNew(t, 10, 1)
	defer page.Deinit()
	pg3HFillAll(page)
	a := pg3HStyleCells(t, page, 1, [][2]uint{{1, 0}, {2, 0}, {7, 0}})
	row := page.GetRow(0)

	page.ClearCells(row, 0, 3)
	expectEqual(t, uint16(1), page.styles.RefCount(a))
	expect(t, row.styled, "row.styled")
	expectEqual(t, StyleID(0), page.GetRowAndCell(1, 0).cell.StyleID())
	expectEqual(t, a, page.GetRowAndCell(7, 0).cell.StyleID())

	page.ClearCells(row, 3, 10)
	expectEqual(t, uint16(0), page.styles.RefCount(a))
	expectEqual(t, uint(0), page.styles.Count())
	// Upstream quirk: the flag is refreshed before the cells in the range
	// are zeroed, and releasing a style does not reset the cell's style_id,
	// so the flag stays set although no styled cell remains.
	expect(t, row.styled, "row.styled")
	expectEqual(t, StyleID(0), page.GetRowAndCell(7, 0).cell.StyleID())
	// A second pass over the (now zeroed) cells does clear it.
	page.ClearCells(row, 0, 2)
	expect(t, !row.styled, "!row.styled")
}

// Zig: test "term51: Page.clearCells: full row clears all managed flags unconditionally"
func TestPg3_Page_clearCells_full_row_clears_all_managed_flags_unconditionally(t *testing.T) {
	page := pg3HNew(t, 6, 1)
	defer page.Deinit()
	pg3HFillAll(page)
	row := page.GetRow(0)
	row.grapheme = true
	row.styled = true
	row.hyperlink = true
	page.ClearCells(row, 0, 6)
	expect(t, !row.grapheme && !row.styled && !row.hyperlink, "!row.grapheme and !row.styled and !row.hyperlink")
}

// Zig: test "term51: Page.clearCells: partial range refreshes stale flags"
func TestPg3_Page_clearCells_partial_range_refreshes_stale_flags(t *testing.T) {
	page := pg3HNew(t, 6, 1)
	defer page.Deinit()
	row := page.GetRow(0)
	row.grapheme = true
	row.styled = true
	row.hyperlink = true
	page.ClearCells(row, 1, 3)
	expect(t, !row.grapheme && !row.styled && !row.hyperlink, "!row.grapheme and !row.styled and !row.hyperlink")

	// An empty range also refreshes.
	row.grapheme = true
	row.styled = true
	row.hyperlink = true
	page.ClearCells(row, 2, 2)
	expect(t, !row.grapheme && !row.styled && !row.hyperlink, "!row.grapheme and !row.styled and !row.hyperlink")
}

// Zig: test "term51: Page.clearCells: a row without managed flags is left alone, flags stay false"
func TestPg3_Page_clearCells_a_row_without_managed_flags_is_left_alone_flags_stay_false(t *testing.T) {
	page := pg3HNew(t, 6, 1)
	defer page.Deinit()
	pg3HFillAll(page)
	row := page.GetRow(0)
	page.ClearCells(row, 1, 4)
	expect(t, !row.ManagedMemory(), "!row.managedMemory()")
}

// Zig: test "term51: Page.clearCells: a cell with all kinds of managed data"
func TestPg3_Page_clearCells_a_cell_with_all_kinds_of_managed_data(t *testing.T) {
	page := pg3HNew(t, 6, 1)
	defer page.Deinit()
	pg3HGrapheme(t, page, 2, 0, 'e', []uint32{0x301})
	sid := pg3HStyleCells(t, page, 1, [][2]uint{{2, 0}})
	hid := pg3HLinkCells(t, page, pg3HLink("https://a"), [][2]uint{{2, 0}})
	page.ClearCells(page.GetRow(0), 0, 6)
	expectEqual(t, uint(0), page.GraphemeCount())
	expectEqual(t, uint(0), page.HyperlinkCount())
	expectEqual(t, uint16(0), page.styles.RefCount(sid))
	expectEqual(t, uint16(0), page.hyperlink_set.RefCount(hid))
	expect(t, page.GetRowAndCell(2, 0).cell.IsZero(), "page.getRowAndCell(2, 0).cell.isZero()")
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.clearCells: wide characters can be cleared as a pair"
func TestPg3_Page_clearCells_wide_characters_can_be_cleared_as_a_pair(t *testing.T) {
	page := pg3HNew(t, 6, 1)
	defer page.Deinit()
	*page.GetRowAndCell(1, 0).cell = PageCellInit('w')
	page.GetRowAndCell(1, 0).cell.setWide(PageCellWideWide)
	page.GetRowAndCell(2, 0).cell.setWide(PageCellWideSpacerTail)
	page.ClearCells(page.GetRow(0), 1, 3)
	for _, c := range page.GetCells(page.GetRow(0)) {
		expect(t, c.IsZero(), "c.isZero()")
	}
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.clearCells: leaves wrap, prompt and dirty row metadata alone"
func TestPg3_Page_clearCells_leaves_wrap_prompt_and_dirty_row_metadata_alone(t *testing.T) {
	page := pg3HNew(t, 6, 1)
	defer page.Deinit()
	pg3HFillAll(page)
	row := page.GetRow(0)
	row.wrap = true
	row.wrap_continuation = true
	row.semantic_prompt = PageRowSemanticPromptPromptContinuation
	row.dirty = false
	page.dirty = false
	page.ClearCells(row, 0, 6)
	expect(t, row.wrap && row.wrap_continuation, "row.wrap and row.wrap_continuation")
	expectEqual(t, PageRowSemanticPromptPromptContinuation, row.semantic_prompt)
	expect(t, !row.dirty, "!row.dirty")
	expect(t, !page.dirty, "!page.dirty")
}

// Zig: test "term51: Page.clearCells: kitty placeholder row flag"
func TestPg3_Page_clearCells_kitty_placeholder_row_flag(t *testing.T) {
	page := pg3HNew(t, 6, 3)
	defer page.Deinit()

	// Row 0: flag set, no placeholder cell present: cleared by a full clear
	// (only with kitty graphics enabled).
	pg3HFillRow(page, 0, 0x100)
	page.GetRow(0).kitty_virtual_placeholder = true
	page.ClearCells(page.GetRow(0), 0, 6)
	// Zig: `if (comptime build_options.kitty_graphics) { expect(!flag) } else
	// { expect(flag) }`. kitty_graphics is false in the library build (see
	// the header of page.go), so only the else branch exists.
	expect(t, page.GetRow(0).kitty_virtual_placeholder, "page.getRow(0).kitty_virtual_placeholder")

	// Row 1: a placeholder cell is present, the check runs before the
	// cells are zeroed so the flag stays set.
	pg3HFillRow(page, 1, 0x100)
	*page.GetRowAndCell(3, 1).cell = PageCellInit(0x10EEEE)
	page.GetRow(1).kitty_virtual_placeholder = true
	page.ClearCells(page.GetRow(1), 0, 6)
	expect(t, page.GetRow(1).kitty_virtual_placeholder, "page.getRow(1).kitty_virtual_placeholder")
	expect(t, page.GetRowAndCell(3, 1).cell.IsZero(), "page.getRowAndCell(3, 1).cell.isZero()")

	// Row 2: partial clears never touch the flag.
	pg3HFillRow(page, 2, 0x100)
	page.GetRow(2).kitty_virtual_placeholder = true
	page.ClearCells(page.GetRow(2), 0, 5)
	expect(t, page.GetRow(2).kitty_virtual_placeholder, "page.getRow(2).kitty_virtual_placeholder")
}

// ---- Cell --------------------------------------------------------------------

// Zig: test "term51: Cell.init: layout of Cell bit fields"
func TestPg3_Cell_init_layout_of_Cell_bit_fields(t *testing.T) {
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 8), @sizeOf(Cell));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 64), @bitSizeOf(Cell));

	// Bit offsets of every field.
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 0), @bitOffsetOf(Cell, "content_tag"));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 2), @bitOffsetOf(Cell, "content"));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 26), @bitOffsetOf(Cell, "style_id"));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 42), @bitOffsetOf(Cell, "wide"));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 44), @bitOffsetOf(Cell, "protected"));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 45), @bitOffsetOf(Cell, "hyperlink"));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 46), @bitOffsetOf(Cell, "semantic_content"));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 48), @bitOffsetOf(Cell, "_padding"));

	// Bit widths. The content union is 24 bits wide: the widest member
	// (color_rgb) is 24 bits, a codepoint uses only 21 of them.
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 2), @bitSizeOf(Cell.ContentTag));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 24), @bitSizeOf(@FieldType(Cell, "content")));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 24), @bitSizeOf(Cell.RGB));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 16), @bitSizeOf(StyleId));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 2), @bitSizeOf(Cell.Wide));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 2), @bitSizeOf(Cell.SemanticContent));

	// Enum values.
	expectEqual(t, uint8(0), uint8(PageCellContentTagCodepoint))
	expectEqual(t, uint8(1), uint8(PageCellContentTagCodepointGrapheme))
	expectEqual(t, uint8(2), uint8(PageCellContentTagBgColorPalette))
	expectEqual(t, uint8(3), uint8(PageCellContentTagBgColorRgb))
	expectEqual(t, uint8(0), uint8(PageCellWideNarrow))
	expectEqual(t, uint8(1), uint8(PageCellWideWide))
	expectEqual(t, uint8(2), uint8(PageCellWideSpacerTail))
	expectEqual(t, uint8(3), uint8(PageCellWideSpacerHead))
	expectEqual(t, uint8(0), uint8(PageCellSemanticContentOutput))
	expectEqual(t, uint8(1), uint8(PageCellSemanticContentInput))
	expectEqual(t, uint8(2), uint8(PageCellSemanticContentPrompt))

	// Reading fields out of raw bits.
	expectEqual(t, PageCellContentTagCodepointGrapheme, pg3IRaw(1).ContentTag())
	expectEqual(t, PageCellContentTagBgColorPalette, pg3IRaw(2).ContentTag())
	expectEqual(t, PageCellContentTagBgColorRgb, pg3IRaw(3).ContentTag())
	expectEqual(t, uint32(0x1FFFFF), pg3IRaw(uint64(0x1FFFFF)<<2).content().codepoint())
	expectEqual(t, uint8(0xAB), pg3IRaw(uint64(0xAB)<<2).content().colorPalette())
	rgb := pg3IRaw(uint64(0x030201) << 2).content().colorRGB()
	expectEqual(t, uint8(1), rgb.R)
	expectEqual(t, uint8(2), rgb.G)
	expectEqual(t, uint8(3), rgb.B)
	expectEqual(t, StyleID(0xABCD), pg3IRaw(uint64(0xABCD)<<26).StyleID())
	expectEqual(t, PageCellWideWide, pg3IRaw(uint64(1)<<42).Wide())
	expectEqual(t, PageCellWideSpacerTail, pg3IRaw(uint64(2)<<42).Wide())
	expectEqual(t, PageCellWideSpacerHead, pg3IRaw(uint64(3)<<42).Wide())
	expect(t, pg3IRaw(uint64(1)<<44).Protected(), "term51IRaw(@as(u64, 1) << 44).protected")
	expect(t, !pg3IRaw(uint64(1)<<44).Hyperlink(), "!term51IRaw(@as(u64, 1) << 44).hyperlink")
	expect(t, pg3IRaw(uint64(1)<<45).Hyperlink(), "term51IRaw(@as(u64, 1) << 45).hyperlink")
	expect(t, !pg3IRaw(uint64(1)<<45).Protected(), "!term51IRaw(@as(u64, 1) << 45).protected")
	expectEqual(t, PageCellSemanticContentInput, pg3IRaw(uint64(1)<<46).SemanticContent())
	expectEqual(t, PageCellSemanticContentPrompt, pg3IRaw(uint64(2)<<46).SemanticContent())
	expectEqual(t, uint16(0xFFFF), pg3IRaw(uint64(0xFFFF)<<48).padding())

	// Writing whole fields sets exactly their bits.
	c := pg3IRaw(0)
	c.setContentTag(PageCellContentTagBgColorRgb)
	expectEqual(t, uint64(3), pg3IBits(c))
	c = pg3IRaw(0)
	c.setStyleID(0xFFFF)
	expectEqual(t, uint64(0xFFFF)<<26, pg3IBits(c))
	c = pg3IRaw(0)
	c.setWide(PageCellWideSpacerHead)
	expectEqual(t, uint64(3)<<42, pg3IBits(c))
	c = pg3IRaw(0)
	c.setProtected(true)
	expectEqual(t, uint64(1)<<44, pg3IBits(c))
	c = pg3IRaw(0)
	c.setHyperlink(true)
	expectEqual(t, uint64(1)<<45, pg3IBits(c))
	c = pg3IRaw(0)
	c.setSemanticContent(PageCellSemanticContentPrompt)
	expectEqual(t, uint64(2)<<46, pg3IBits(c))
	c = pg3IRaw(0)
	c.setPadding(0xFFFF)
	expectEqual(t, uint64(0xFFFF)<<48, pg3IBits(c))
}

// Zig: test "term51: Cell.init: sets the codepoint and nothing else"
func TestPg3_Cell_init_sets_the_codepoint_and_nothing_else(t *testing.T) {
	// Zero codepoint: the all-zero cell.
	zero := PageCellInit(0)
	expectEqual(t, uint64(0), pg3IBits(zero))
	expectEqual(t, PageCellContentTagCodepoint, zero.ContentTag())
	expectEqual(t, uint32(0), zero.content().codepoint())
	expectEqual(t, StyleID(0), zero.StyleID())
	expectEqual(t, PageCellWideNarrow, zero.Wide())
	expect(t, !zero.Protected(), "!zero.protected")
	expect(t, !zero.Hyperlink(), "!zero.hyperlink")
	expectEqual(t, PageCellSemanticContentOutput, zero.SemanticContent())
	expectEqual(t, uint16(0), zero.padding())

	// Smallest, a few boundaries of the UTF-8 / UTF-16 ranges, the largest
	// Unicode scalar and the largest u21: the codepoint is stored in bits
	// 2..22 and every other bit of the cell stays zero (in particular the
	// 3 unused bits of the content union).
	cps := []uint32{1, 'A', 0x7F, 0x80, 0x7FF, 0x800, 0xFFFF, 0x10000, 0x10FFFF, 0x10EEEE, 0x1FFFFF}
	for _, cp := range cps {
		c := PageCellInit(cp)
		expectEqual(t, uint64(cp)<<2, pg3IBits(c))
		expectEqual(t, PageCellContentTagCodepoint, c.ContentTag())
		expectEqual(t, cp, c.content().codepoint())
		expectEqual(t, cp, c.Codepoint())
		expectEqual(t, StyleID(0), c.StyleID())
		expectEqual(t, PageCellWideNarrow, c.Wide())
		expect(t, !c.Protected(), "!c.protected")
		expect(t, !c.Hyperlink(), "!c.hyperlink")
		expectEqual(t, PageCellSemanticContentOutput, c.SemanticContent())
		expectEqual(t, uint16(0), c.padding())
	}

	// The default cell, built at runtime, equals init(0).
	expectEqual(t, pg3IBits(pg3IBlank()), pg3IBits(PageCellInit(0)))
}

// Zig: test "term51: Cell.isZero: default cells are zero"
func TestPg3_Cell_isZero_default_cells_are_zero(t *testing.T) {
	// A default cell built at runtime, as Screen.blankCell returns it.
	expectEqual(t, uint64(0), pg3IBits(pg3IBlank()))
	expect(t, pg3IBlank().IsZero(), "term51IBlank().isZero()")
	expectEqual(t, uint64(0), pg3IBits(pg3ISpelled()))
	expect(t, pg3ISpelled().IsZero(), "term51ISpelled().isZero()")
	expect(t, PageCellInit(0).IsZero(), "Cell.init(0).isZero()")
	expect(t, pg3IRaw(0).IsZero(), "term51IRaw(0).isZero()")
	cell := PageCellInit(0)
	expect(t, cell.IsZero(), "cell.isZero()")
	// A zero cell that had a field set and reset is zero again.
	cell.setWide(PageCellWideWide)
	expect(t, !cell.IsZero(), "!cell.isZero()")
	cell.setWide(PageCellWideNarrow)
	expect(t, cell.IsZero(), "cell.isZero()")
}

// Zig: test "term51: Cell.isZero: any single non-default field is non-zero"
func TestPg3_Cell_isZero_any_single_non_default_field_is_non_zero(t *testing.T) {
	// Content tags (with zero content).
	expect(t, !pg3ITagged(PageCellContentTagCodepointGrapheme, 0).IsZero(), "!term51ITagged(.codepoint_grapheme, 0).isZero()")
	expect(t, !pg3ITagged(PageCellContentTagBgColorPalette, 0).IsZero(), "!term51ITagged(.bg_color_palette, 0).isZero()")
	expect(t, !pg3ITagged(PageCellContentTagBgColorRgb, 0).IsZero(), "!term51ITagged(.bg_color_rgb, 0).isZero()")

	// Content (with the codepoint tag), smallest and largest codepoint,
	// a palette-sized value and an rgb-sized value.
	expect(t, !PageCellInit(1).IsZero(), "!Cell.init(1).isZero()")
	expect(t, !PageCellInit(0x1FFFFF).IsZero(), "!Cell.init(0x1FFFFF).isZero()")
	expect(t, !pg3ITagged(PageCellContentTagCodepoint, 0xFF).IsZero(), "!term51ITagged(.codepoint, 0xFF).isZero()")
	expect(t, !pg3ITagged(PageCellContentTagCodepoint, 0xFFFFFF).IsZero(), "!term51ITagged(.codepoint, 0xFFFFFF).isZero()")

	// Style id.
	c := pg3IRaw(0)
	c.setStyleID(1)
	expect(t, !c.IsZero(), "!c.isZero()")
	c.setStyleID(0xFFFF)
	expect(t, !c.IsZero(), "!c.isZero()")

	// Wide.
	for _, w := range []PageCellWide{PageCellWideWide, PageCellWideSpacerTail, PageCellWideSpacerHead} {
		c = pg3IRaw(0)
		c.setWide(w)
		expect(t, !c.IsZero(), "!c.isZero()")
	}

	// Flags.
	c = pg3IRaw(0)
	c.setProtected(true)
	expect(t, !c.IsZero(), "!c.isZero()")
	c = pg3IRaw(0)
	c.setHyperlink(true)
	expect(t, !c.IsZero(), "!c.isZero()")

	// Semantic content.
	c = pg3IRaw(0)
	c.setSemanticContent(PageCellSemanticContentInput)
	expect(t, !c.IsZero(), "!c.isZero()")
	c.setSemanticContent(PageCellSemanticContentPrompt)
	expect(t, !c.IsZero(), "!c.isZero()")
	c.setSemanticContent(PageCellSemanticContentOutput)
	expect(t, c.IsZero(), "c.isZero()")
}

// Zig: test "term51: Cell.isZero: compares all 64 raw bits"
func TestPg3_Cell_isZero_compares_all_64_raw_bits(t *testing.T) {
	// Every single bit of the cell, including the padding and the unused
	// bits of the content union, makes the cell non-zero.
	for i := 0; i < 64; i++ {
		bits := uint64(1) << i
		c := pg3IRaw(bits)
		expect(t, !c.IsZero(), "!c.isZero()")
		// And clearing it again gives zero.
		expectEqual(t, uint64(0), pg3IBits(c)&^bits)
	}

	// All bits set except semantic_content=3 (an invalid enum value);
	// using 0x1 in that field.
	nearly_all := ^uint64(0) &^ (uint64(2) << 46)
	expect(t, !pg3IRaw(nearly_all).IsZero(), "!term51IRaw(nearly_all).isZero()")
}

// Zig: test "term51: Cell.isZero: leftover bits are not ignored (docs/cell-iszero.md)"
func TestPg3_Cell_isZero_leftover_bits_are_not_ignored(t *testing.T) {
	// Bits 23, 24, 25 are the part of the 24-bit content union a codepoint
	// does not use. A cell with only those bits set reads as an empty
	// codepoint cell (no text, codepoint 0, isEmpty) but is not zero:
	// isZero is a raw comparison, not a semantic one.
	for _, bits := range []uint64{1 << 23, 1 << 24, 1 << 25, 7 << 23} {
		c := pg3IRaw(bits)
		expectEqual(t, PageCellContentTagCodepoint, c.ContentTag())
		expectEqual(t, uint32(0), c.Codepoint())
		expect(t, !c.HasText(), "!c.hasText()")
		expect(t, c.IsEmpty(), "c.isEmpty()")
		expect(t, !c.IsZero(), "!c.isZero()")
	}

	// The same for the 16 padding bits.
	for _, bits := range []uint64{uint64(1) << 48, uint64(1) << 63, uint64(0xFFFF) << 48} {
		c := pg3IRaw(bits)
		expectEqual(t, uint32(0), c.Codepoint())
		expect(t, !c.HasText(), "!c.hasText()")
		expect(t, c.IsEmpty(), "c.isEmpty()")
		expectEqual(t, PageCellWideNarrow, c.Wide())
		expect(t, !c.IsZero(), "!c.isZero()")
	}

	// Cells that Cell.init or a literal builds have no leftover bits.
	expect(t, PageCellInit(0).IsZero(), "Cell.init(0).isZero()")
	expectEqual(t, uint64(0), pg3IBits(PageCellInit(0)))
}

// Zig: test "term51: Cell.isZero: background-only cells are never zero"
func TestPg3_Cell_isZero_background_only_cells_are_never_zero(t *testing.T) {
	// The blank cell a background colour produces (tag bg_color_palette or
	// bg_color_rgb) is never zero, whatever the colour, including palette
	// index 0 and rgb black. This is what the single library caller
	// (Terminal.index with left/right margins) relies on.
	for palette := uint16(0); palette <= 255; palette++ {
		expect(t, !pg3ITagged(PageCellContentTagBgColorPalette, uint32(palette)).IsZero(), "!term51ITagged(.bg_color_palette, @intCast(palette)).isZero()")
		lit := pg3IBgPalette(uint8(palette))
		expectEqual(t, PageCellContentTagBgColorPalette, lit.ContentTag())
		expectEqual(t, uint8(palette), lit.content().colorPalette())
		expect(t, !lit.IsZero(), "!lit.isZero()")
	}
	colors := []uint32{0x000000, 0x000001, 0x010000, 0x0000FF, 0xFFFFFF, 0x800000}
	for _, rgb := range colors {
		expect(t, !pg3ITagged(PageCellContentTagBgColorRgb, rgb).IsZero(), "!term51ITagged(.bg_color_rgb, rgb).isZero()")
		lit := pg3IBgRgb(uint8(rgb), uint8(rgb>>8), uint8(rgb>>16))
		expectEqual(t, PageCellContentTagBgColorRgb, lit.ContentTag())
		expectEqual(t, uint8(rgb), lit.content().colorRGB().R)
		expect(t, !lit.IsZero(), "!lit.isZero()")
	}
}

// Zig: test "term51: Cell.hasText: codepoint tag"
func TestPg3_Cell_hasText_codepoint_tag(t *testing.T) {
	expect(t, !PageCellInit(0).HasText(), "!Cell.init(0).hasText()")
	expect(t, PageCellInit(1).HasText(), "Cell.init(1).hasText()")
	expect(t, PageCellInit('A').HasText(), "Cell.init('A').hasText()")
	expect(t, PageCellInit(0x1FFFFF).HasText(), "Cell.init(0x1FFFFF).hasText()")

	// Kitty virtual placeholder (U+10EEEE): the doc comment says this
	// returns false, but the code only checks for a non-zero codepoint.
	expect(t, PageCellInit(0x10EEEE).HasText(), "Cell.init(0x10EEEE).hasText()")

	// Only the codepoint matters: wide, spacer, style, flags do not.
	c := PageCellInit(0)
	c.setWide(PageCellWideWide)
	expect(t, !c.HasText(), "!c.hasText()")
	c.setWide(PageCellWideSpacerTail)
	expect(t, !c.HasText(), "!c.hasText()")
	c.setWide(PageCellWideSpacerHead)
	expect(t, !c.HasText(), "!c.hasText()")
	c = PageCellInit(0)
	c.setStyleID(5)
	c.setProtected(true)
	c.setHyperlink(true)
	c.setSemanticContent(PageCellSemanticContentPrompt)
	expect(t, !c.HasText(), "!c.hasText()")
	c = PageCellInit('x')
	c.setWide(PageCellWideSpacerTail)
	expect(t, c.HasText(), "c.hasText()")

	// Bits 23..25 of the content union are not part of the codepoint.
	expect(t, !pg3ITagged(PageCellContentTagCodepoint, 0xE00000).HasText(), "!term51ITagged(.codepoint, 0xE00000).hasText()")
	expect(t, pg3ITagged(PageCellContentTagCodepoint, 0xE00001).HasText(), "term51ITagged(.codepoint, 0xE00001).hasText()")
}

// Zig: test "term51: Cell.hasText: grapheme tag"
func TestPg3_Cell_hasText_grapheme_tag(t *testing.T) {
	// The codepoint of a grapheme cell is its first codepoint.
	expect(t, pg3ITagged(PageCellContentTagCodepointGrapheme, 1).HasText(), "term51ITagged(.codepoint_grapheme, 1).hasText()")
	expect(t, pg3ITagged(PageCellContentTagCodepointGrapheme, 0x1FFFFF).HasText(), "term51ITagged(.codepoint_grapheme, 0x1FFFFF).hasText()")
	// A grapheme tag over a zero codepoint has no text.
	expect(t, !pg3ITagged(PageCellContentTagCodepointGrapheme, 0).HasText(), "!term51ITagged(.codepoint_grapheme, 0).hasText()")
	expect(t, !pg3ITagged(PageCellContentTagCodepointGrapheme, 0xE00000).HasText(), "!term51ITagged(.codepoint_grapheme, 0xE00000).hasText()")
}

// Zig: test "term51: Cell.hasText: background cells never have text"
func TestPg3_Cell_hasText_background_cells_never_have_text(t *testing.T) {
	contents := []uint32{0, 1, 0xFF, 0x1FFFFF, 0xFFFFFF}
	for _, content := range contents {
		expect(t, !pg3ITagged(PageCellContentTagBgColorPalette, content).HasText(), "!term51ITagged(.bg_color_palette, content).hasText()")
		expect(t, !pg3ITagged(PageCellContentTagBgColorRgb, content).HasText(), "!term51ITagged(.bg_color_rgb, content).hasText()")
	}
}

// Zig: test "term51: Cell.codepoint: codepoint and grapheme tags return the codepoint"
func TestPg3_Cell_codepoint_codepoint_and_grapheme_tags_return_the_codepoint(t *testing.T) {
	expectEqual(t, uint32(0), PageCellInit(0).Codepoint())
	expectEqual(t, uint32(1), PageCellInit(1).Codepoint())
	expectEqual(t, uint32(0x10FFFF), PageCellInit(0x10FFFF).Codepoint())
	expectEqual(t, uint32(0x1FFFFF), PageCellInit(0x1FFFFF).Codepoint())
	expectEqual(t, uint32(0x1FFFFF), pg3ITagged(PageCellContentTagCodepoint, 0x1FFFFF).Codepoint())

	// Grapheme tag: same.
	expectEqual(t, uint32(0), pg3ITagged(PageCellContentTagCodepointGrapheme, 0).Codepoint())
	expectEqual(t, uint32(0x41), pg3ITagged(PageCellContentTagCodepointGrapheme, 0x41).Codepoint())
	expectEqual(t, uint32(0x1FFFFF), pg3ITagged(PageCellContentTagCodepointGrapheme, 0x1FFFFF).Codepoint())

	// Only the 21 low content bits are the codepoint.
	expectEqual(t, uint32(0), pg3ITagged(PageCellContentTagCodepoint, 0xE00000).Codepoint())
	expectEqual(t, uint32(0x1FFFFF), pg3ITagged(PageCellContentTagCodepoint, 0xFFFFFF).Codepoint())
	expectEqual(t, uint32(0x41), pg3ITagged(PageCellContentTagCodepointGrapheme, 0xE00041).Codepoint())

	// Other fields do not matter.
	c := PageCellInit(0x42)
	c.setWide(PageCellWideWide)
	c.setStyleID(9)
	c.setProtected(true)
	c.setHyperlink(true)
	c.setSemanticContent(PageCellSemanticContentInput)
	expectEqual(t, uint32(0x42), c.Codepoint())
}

// Zig: test "term51: Cell.codepoint: background cells return 0"
func TestPg3_Cell_codepoint_background_cells_return_0(t *testing.T) {
	contents := []uint32{0, 1, 0xFF, 0x1FFFFF, 0x123456, 0xFFFFFF}
	for _, content := range contents {
		expectEqual(t, uint32(0), pg3ITagged(PageCellContentTagBgColorPalette, content).Codepoint())
		expectEqual(t, uint32(0), pg3ITagged(PageCellContentTagBgColorRgb, content).Codepoint())
	}
}

// Zig: test "term51: Cell.gridWidth: width per wide value"
func TestPg3_Cell_gridWidth_width_per_wide_value(t *testing.T) {
	var c PageCell
	expectEqual(t, uint8(1), c.GridWidth())
	c.setWide(PageCellWideNarrow)
	expectEqual(t, uint8(1), c.GridWidth())
	c.setWide(PageCellWideWide)
	expectEqual(t, uint8(2), c.GridWidth())
	c.setWide(PageCellWideSpacerTail)
	expectEqual(t, uint8(1), c.GridWidth())
	c.setWide(PageCellWideSpacerHead)
	expectEqual(t, uint8(1), c.GridWidth())

	// Independent of the content.
	tags := []PageCellContentTag{PageCellContentTagCodepoint, PageCellContentTagCodepointGrapheme, PageCellContentTagBgColorPalette, PageCellContentTagBgColorRgb}
	for _, tag := range tags {
		d := pg3ITagged(tag, 0x41)
		d.setWide(PageCellWideWide)
		expectEqual(t, uint8(2), d.GridWidth())
		d.setWide(PageCellWideNarrow)
		expectEqual(t, uint8(1), d.GridWidth())
	}
	w := PageCellInit(0x4E2D)
	w.setWide(PageCellWideWide)
	expectEqual(t, uint8(2), w.GridWidth())
}

// Zig: test "term51: Cell.hasStyling: style id other than the default"
func TestPg3_Cell_hasStyling_style_id_other_than_the_default(t *testing.T) {
	var c PageCell
	expect(t, !c.HasStyling(), "!c.hasStyling()")
	c.setStyleID(0)
	expect(t, !c.HasStyling(), "!c.hasStyling()")
	c.setStyleID(1)
	expect(t, c.HasStyling(), "c.hasStyling()")
	c.setStyleID(2)
	expect(t, c.HasStyling(), "c.hasStyling()")
	c.setStyleID(0x8000)
	expect(t, c.HasStyling(), "c.hasStyling()")
	c.setStyleID(0xFFFF)
	expect(t, c.HasStyling(), "c.hasStyling()")

	// Only the style id matters, not the content tag or other fields.
	tags := []PageCellContentTag{PageCellContentTagCodepoint, PageCellContentTagCodepointGrapheme, PageCellContentTagBgColorPalette, PageCellContentTagBgColorRgb}
	for _, tag := range tags {
		d := pg3ITagged(tag, 0xFF)
		d.setWide(PageCellWideWide)
		d.setProtected(true)
		d.setHyperlink(true)
		d.setSemanticContent(PageCellSemanticContentPrompt)
		expect(t, !d.HasStyling(), "!d.hasStyling()")
		d.setStyleID(1)
		expect(t, d.HasStyling(), "d.hasStyling()")
	}
}

// Zig: test "term51: Cell.isEmpty: textual cells"
func TestPg3_Cell_isEmpty_textual_cells(t *testing.T) {
	// Codepoint tag.
	expect(t, PageCellInit(0).IsEmpty(), "Cell.init(0).isEmpty()")
	expect(t, !PageCellInit(1).IsEmpty(), "!Cell.init(1).isEmpty()")
	expect(t, !PageCellInit(0x1FFFFF).IsEmpty(), "!Cell.init(0x1FFFFF).isEmpty()")

	// Zero codepoint but a wide-ness other than narrow is meaningful
	// (wide spacers), so not empty.
	c := PageCellInit(0)
	c.setWide(PageCellWideWide)
	expect(t, !c.IsEmpty(), "!c.isEmpty()")
	c.setWide(PageCellWideSpacerTail)
	expect(t, !c.IsEmpty(), "!c.isEmpty()")
	c.setWide(PageCellWideSpacerHead)
	expect(t, !c.IsEmpty(), "!c.isEmpty()")
	c.setWide(PageCellWideNarrow)
	expect(t, c.IsEmpty(), "c.isEmpty()")

	// A cell with text is not empty whatever its width.
	c = PageCellInit('a')
	c.setWide(PageCellWideWide)
	expect(t, !c.IsEmpty(), "!c.isEmpty()")
	c.setWide(PageCellWideSpacerTail)
	expect(t, !c.IsEmpty(), "!c.isEmpty()")

	// Grapheme tag follows the same rule.
	expect(t, pg3ITagged(PageCellContentTagCodepointGrapheme, 0).IsEmpty(), "term51ITagged(.codepoint_grapheme, 0).isEmpty()")
	expect(t, !pg3ITagged(PageCellContentTagCodepointGrapheme, 1).IsEmpty(), "!term51ITagged(.codepoint_grapheme, 1).isEmpty()")
	c = pg3ITagged(PageCellContentTagCodepointGrapheme, 0)
	c.setWide(PageCellWideSpacerHead)
	expect(t, !c.IsEmpty(), "!c.isEmpty()")

	// The unused content bits are not text.
	expect(t, pg3ITagged(PageCellContentTagCodepoint, 0xE00000).IsEmpty(), "term51ITagged(.codepoint, 0xE00000).isEmpty()")
}

// Zig: test "term51: Cell.isEmpty: ignores style, flags and semantic content"
func TestPg3_Cell_isEmpty_ignores_style_flags_and_semantic_content(t *testing.T) {
	// The doc comment says "no text or styling" but styling is not
	// checked: a styled blank narrow cell is empty.
	c := PageCellInit(0)
	c.setStyleID(7)
	expect(t, c.HasStyling(), "c.hasStyling()")
	expect(t, c.IsEmpty(), "c.isEmpty()")
	c = PageCellInit(0)
	c.setProtected(true)
	expect(t, c.IsEmpty(), "c.isEmpty()")
	c = PageCellInit(0)
	c.setHyperlink(true)
	expect(t, c.IsEmpty(), "c.isEmpty()")
	c = PageCellInit(0)
	c.setSemanticContent(PageCellSemanticContentPrompt)
	expect(t, c.IsEmpty(), "c.isEmpty()")
	c.setSemanticContent(PageCellSemanticContentInput)
	expect(t, c.IsEmpty(), "c.isEmpty()")
}

// Zig: test "term51: Cell.isEmpty: background cells are never empty"
func TestPg3_Cell_isEmpty_background_cells_are_never_empty(t *testing.T) {
	contents := []uint32{0, 1, 0xFF, 0xFFFFFF}
	for _, content := range contents {
		expect(t, !pg3ITagged(PageCellContentTagBgColorPalette, content).IsEmpty(), "!term51ITagged(.bg_color_palette, content).isEmpty()")
		expect(t, !pg3ITagged(PageCellContentTagBgColorRgb, content).IsEmpty(), "!term51ITagged(.bg_color_rgb, content).isEmpty()")
	}
}

// Zig: test "term51: Cell.hasGrapheme: only the grapheme tag"
func TestPg3_Cell_hasGrapheme_only_the_grapheme_tag(t *testing.T) {
	expect(t, !PageCellInit(0).HasGrapheme(), "!Cell.init(0).hasGrapheme()")
	expect(t, !PageCellInit('a').HasGrapheme(), "!Cell.init('a').hasGrapheme()")
	expect(t, pg3ITagged(PageCellContentTagCodepointGrapheme, 0).HasGrapheme(), "term51ITagged(.codepoint_grapheme, 0).hasGrapheme()")
	expect(t, pg3ITagged(PageCellContentTagCodepointGrapheme, 0x41).HasGrapheme(), "term51ITagged(.codepoint_grapheme, 0x41).hasGrapheme()")
	expect(t, !pg3ITagged(PageCellContentTagBgColorPalette, 0).HasGrapheme(), "!term51ITagged(.bg_color_palette, 0).hasGrapheme()")
	expect(t, !pg3ITagged(PageCellContentTagBgColorPalette, 1).HasGrapheme(), "!term51ITagged(.bg_color_palette, 1).hasGrapheme()")
	expect(t, !pg3ITagged(PageCellContentTagBgColorRgb, 0).HasGrapheme(), "!term51ITagged(.bg_color_rgb, 0).hasGrapheme()")
	expect(t, !pg3ITagged(PageCellContentTagBgColorRgb, 0xFFFFFF).HasGrapheme(), "!term51ITagged(.bg_color_rgb, 0xFFFFFF).hasGrapheme()")

	// Other fields do not matter.
	c := pg3ITagged(PageCellContentTagCodepointGrapheme, 0)
	c.setWide(PageCellWideWide)
	c.setStyleID(3)
	c.setHyperlink(true)
	expect(t, c.HasGrapheme(), "c.hasGrapheme()")
	c = PageCellInit('a')
	c.setWide(PageCellWideWide)
	c.setStyleID(3)
	c.setHyperlink(true)
	expect(t, !c.HasGrapheme(), "!c.hasGrapheme()")
}

// Zig: test "term51: Cell.hasTextAny: slices of cells"
func TestPg3_Cell_hasTextAny_slices_of_cells(t *testing.T) {
	// Empty slice.
	expect(t, !PageCellHasTextAny([]PageCell{}), "!Cell.hasTextAny(&.{})")

	// One cell.
	expect(t, !PageCellHasTextAny([]PageCell{PageCellInit(0)}), "!Cell.hasTextAny(&.{Cell.init(0)})")
	expect(t, PageCellHasTextAny([]PageCell{PageCellInit('a')}), "Cell.hasTextAny(&.{Cell.init('a')})")

	// All blank.
	var blank [5]PageCell
	for i := range blank {
		blank[i] = PageCellInit(0)
	}
	expect(t, !PageCellHasTextAny(blank[:]), "!Cell.hasTextAny(&blank)")

	// Text in the first, a middle, and the last cell.
	cells := blank
	cells[0] = PageCellInit('a')
	expect(t, PageCellHasTextAny(cells[:]), "Cell.hasTextAny(&cells)")
	cells = blank
	cells[2] = PageCellInit('b')
	expect(t, PageCellHasTextAny(cells[:]), "Cell.hasTextAny(&cells)")
	cells = blank
	cells[4] = PageCellInit(0x1FFFFF)
	expect(t, PageCellHasTextAny(cells[:]), "Cell.hasTextAny(&cells)")

	// Cells that are not text: backgrounds, wide spacers, styled blanks,
	// a grapheme tag over a zero codepoint.
	other := [5]PageCell{
		pg3ITagged(PageCellContentTagBgColorPalette, 0x41),
		pg3ITagged(PageCellContentTagBgColorRgb, 0x414141),
		pg3ITagged(PageCellContentTagCodepointGrapheme, 0),
		PageCellInit(0),
		PageCellInit(0),
	}
	other[3].setWide(PageCellWideSpacerTail)
	other[4].setStyleID(2)
	expect(t, !PageCellHasTextAny(other[:]), "!Cell.hasTextAny(&other)")

	// A grapheme cell with text counts.
	other[2] = pg3ITagged(PageCellContentTagCodepointGrapheme, 0x41)
	expect(t, PageCellHasTextAny(other[:]), "Cell.hasTextAny(&other)")

	// A subslice is what is examined.
	cells = blank
	cells[3] = PageCellInit('z')
	expect(t, !PageCellHasTextAny(cells[0:3]), "!Cell.hasTextAny(cells[0..3])")
	expect(t, PageCellHasTextAny(cells[3:4]), "Cell.hasTextAny(cells[3..4])")
	expect(t, PageCellHasTextAny(cells[2:]), "Cell.hasTextAny(cells[2..])")
	expect(t, !PageCellHasTextAny(cells[4:]), "!Cell.hasTextAny(cells[4..])")
}

// ---- Row ---------------------------------------------------------------------

// Zig: test "term51: Row.managedMemory: layout of Row bit fields"
func TestPg3_Row_managedMemory_layout_of_Row_bit_fields(t *testing.T) {
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 8), @sizeOf(Row));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 64), @bitSizeOf(Row));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 0), @bitOffsetOf(Row, "cells"));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 32), @bitOffsetOf(Row, "wrap"));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 33), @bitOffsetOf(Row, "wrap_continuation"));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 34), @bitOffsetOf(Row, "grapheme"));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 35), @bitOffsetOf(Row, "styled"));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 36), @bitOffsetOf(Row, "hyperlink"));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 37), @bitOffsetOf(Row, "semantic_prompt"));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 39), @bitOffsetOf(Row, "kitty_virtual_placeholder"));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 40), @bitOffsetOf(Row, "dirty"));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 41), @bitOffsetOf(Row, "_padding"));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 32), @bitSizeOf(Offset(Cell)));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 2), @bitSizeOf(Row.SemanticPrompt));
	// Zig-only: try term51Itesting.expectEqual(@as(usize, 23), @bitSizeOf(@FieldType(Row, "_padding")));
	expectEqual(t, uint8(0), uint8(PageRowSemanticPromptNone))
	expectEqual(t, uint8(1), uint8(PageRowSemanticPromptPrompt))
	expectEqual(t, uint8(2), uint8(PageRowSemanticPromptPromptContinuation))

	base := PageRow{cells: 0}
	expectEqual(t, uint64(0), base.bitCast())

	// Writing every field sets exactly its bits.
	r := base
	r.cells = 0xFFFFFFFF
	expectEqual(t, uint64(0xFFFFFFFF), r.bitCast())
	r = base
	r.wrap = true
	expectEqual(t, uint64(1)<<32, r.bitCast())
	r = base
	r.wrap_continuation = true
	expectEqual(t, uint64(1)<<33, r.bitCast())
	r = base
	r.grapheme = true
	expectEqual(t, uint64(1)<<34, r.bitCast())
	r = base
	r.styled = true
	expectEqual(t, uint64(1)<<35, r.bitCast())
	r = base
	r.hyperlink = true
	expectEqual(t, uint64(1)<<36, r.bitCast())
	r = base
	r.semantic_prompt = PageRowSemanticPromptPrompt
	expectEqual(t, uint64(1)<<37, r.bitCast())
	r.semantic_prompt = PageRowSemanticPromptPromptContinuation
	expectEqual(t, uint64(2)<<37, r.bitCast())
	r = base
	r.kitty_virtual_placeholder = true
	expectEqual(t, uint64(1)<<39, r.bitCast())
	r = base
	r.dirty = true
	expectEqual(t, uint64(1)<<40, r.bitCast())
	r = base
	r._padding = 0x7FFFFF
	expectEqual(t, uint64(0x7FFFFF)<<41, r.bitCast())

	// Reading fields out of raw bits.
	rr := pageRowFromBits(uint64(0x12345678) | (uint64(1) << 34) | (uint64(1) << 40))
	expectEqual(t, uint32(0x12345678), uint32(rr.cells))
	expect(t, rr.grapheme, "rr.grapheme")
	expect(t, rr.dirty, "rr.dirty")
	expect(t, !rr.wrap, "!rr.wrap")
	expect(t, !rr.styled, "!rr.styled")
	expect(t, !rr.hyperlink, "!rr.hyperlink")
}

// Zig: test "term51: Row.managedMemory: styled, hyperlink or grapheme"
func TestPg3_Row_managedMemory_styled_hyperlink_or_grapheme(t *testing.T) {
	type Case struct {
		styled    bool
		hyperlink bool
		grapheme  bool
		expected  bool
	}
	cases := []Case{
		{styled: false, hyperlink: false, grapheme: false, expected: false},
		{styled: true, hyperlink: false, grapheme: false, expected: true},
		{styled: false, hyperlink: true, grapheme: false, expected: true},
		{styled: false, hyperlink: false, grapheme: true, expected: true},
		{styled: true, hyperlink: true, grapheme: false, expected: true},
		{styled: true, hyperlink: false, grapheme: true, expected: true},
		{styled: false, hyperlink: true, grapheme: true, expected: true},
		{styled: true, hyperlink: true, grapheme: true, expected: true},
	}
	for _, c := range cases {
		row := PageRow{
			cells:     0,
			styled:    c.styled,
			hyperlink: c.hyperlink,
			grapheme:  c.grapheme,
		}
		expectEqual(t, c.expected, row.ManagedMemory())
	}

	// No other field makes a row have managed memory.
	quiet := PageRow{
		cells:                     0xFFFFFFFF,
		wrap:                      true,
		wrap_continuation:         true,
		semantic_prompt:           PageRowSemanticPromptPromptContinuation,
		kitty_virtual_placeholder: true,
		dirty:                     true,
		_padding:                  0x7FFFFF,
	}
	expect(t, !quiet.ManagedMemory(), "!quiet.managedMemory()")
	loud := quiet
	loud.styled = true
	expect(t, loud.ManagedMemory(), "loud.managedMemory()")
}

// ---- Hyperlinks --------------------------------------------------------------

// pg3Bytes is `alloc.alloc(u8, n)` filled with `@memset(buf, b)`.
func pg3Bytes(n int, b uint8) []uint8 {
	buf := make([]uint8, n)
	for i := range buf {
		buf[i] = b
	}
	return buf
}

// pg3Implicit is `.{ .id = .{ .implicit = n }, .uri = uri }`.
func pg3Implicit(n SizeOffsetInt, uri []uint8) Hyperlink {
	return Hyperlink{
		ID:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: n},
		URI: uri,
	}
}

// pg3Explicit is `.{ .id = .{ .explicit = id }, .uri = uri }`.
func pg3Explicit(id []uint8, uri []uint8) Hyperlink {
	return Hyperlink{
		ID:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdExplicit, Explicit: id},
		URI: uri,
	}
}

// Zig: test "term51: Page.lookupHyperlink: null without a link, id with one"
func TestPg3_Page_lookupHyperlink_null_without_a_link_id_with_one(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	// Fresh page: nothing anywhere.
	for y := uint(0); y < uint(page.size.rows); y++ {
		for x := uint(0); x < uint(page.size.cols); x++ {
			expectEqual(t,
				None[HyperlinkID](),
				page.LookupHyperlink(page.GetRowAndCell(x, y).cell),
			)
		}
	}

	id1 := pg3ILink(t, page, 1)
	id2 := pg3ILink(t, page, 2)
	expect(t, id1 != id2, "id1 != id2")

	a := pg3IPut(page, 3, 4, 'a')
	zigTry(t, page.SetHyperlink(a.row, a.cell, id1))
	b := pg3IPut(page, 9, 9, 'b')
	zigTry(t, page.SetHyperlink(b.row, b.cell, id2))

	// Exactly the two cells have links, each with its own id.
	expectEqual(t, Some(id1), page.LookupHyperlink(a.cell))
	expectEqual(t, Some(id2), page.LookupHyperlink(b.cell))
	for y := uint(0); y < uint(page.size.rows); y++ {
		for x := uint(0); x < uint(page.size.cols); x++ {
			cell := page.GetRowAndCell(x, y).cell
			if cell == a.cell || cell == b.cell {
				continue
			}
			expectEqual(t, None[HyperlinkID](), page.LookupHyperlink(cell))
		}
	}

	// Neighbours of the linked cell do not have the link.
	expectEqual(t, None[HyperlinkID](), page.LookupHyperlink(page.GetRowAndCell(2, 4).cell))
	expectEqual(t, None[HyperlinkID](), page.LookupHyperlink(page.GetRowAndCell(4, 4).cell))
	expectEqual(t, None[HyperlinkID](), page.LookupHyperlink(page.GetRowAndCell(3, 3).cell))
	expectEqual(t, None[HyperlinkID](), page.LookupHyperlink(page.GetRowAndCell(3, 5).cell))
}

// Zig: test "term51: Page.lookupHyperlink: consults the map, not the cell flag"
func TestPg3_Page_lookupHyperlink_consults_the_map_not_the_cell_flag(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	id := pg3ILink(t, page, 1)
	a := pg3IPut(page, 0, 0, 'a')
	zigTry(t, page.SetHyperlink(a.row, a.cell, id))
	expect(t, a.cell.Hyperlink(), "a.cell.hyperlink")

	// Clearing the flag by hand: the map entry is still found.
	a.cell.setHyperlink(false)
	expectEqual(t, Some(id), page.LookupHyperlink(a.cell))
	a.cell.setHyperlink(true)

	// Setting the flag on another cell by hand: no map entry, no id.
	b := pg3IPut(page, 1, 0, 'b')
	b.cell.setHyperlink(true)
	expectEqual(t, None[HyperlinkID](), page.LookupHyperlink(b.cell))
	b.cell.setHyperlink(false)

	// Rewriting the cell content does not drop the map entry either.
	*a.cell = PageCellInit('z')
	expectEqual(t, Some(id), page.LookupHyperlink(a.cell))
}

// Zig: test "term51: Page.clearHyperlink: releases the link, clears the cell flag"
func TestPg3_Page_clearHyperlink_releases_the_link_clears_the_cell_flag(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	id := pg3ILink(t, page, 1)
	a := pg3IPut(page, 0, 0, 'a')
	zigTry(t, page.SetHyperlink(a.row, a.cell, id))
	page.hyperlink_set.Use(id)
	b := pg3IPut(page, 1, 0, 'b')
	zigTry(t, page.SetHyperlink(b.row, b.cell, id))
	expectEqual(t, uint(2), pg3IRefs(page, id))
	expectEqual(t, uint(2), page.HyperlinkCount())

	// Clear one of two.
	page.ClearHyperlink(a.cell)
	expect(t, !a.cell.Hyperlink(), "!a.cell.hyperlink")
	expectEqual(t, None[HyperlinkID](), page.LookupHyperlink(a.cell))
	expectEqual(t, uint(1), page.HyperlinkCount())
	expectEqual(t, uint(1), pg3IRefs(page, id))
	expectEqual(t, uint(1), page.hyperlink_set.Count())
	// The other cell is untouched.
	expect(t, b.cell.Hyperlink(), "b.cell.hyperlink")
	expectEqual(t, Some(id), page.LookupHyperlink(b.cell))
	// The cell text is untouched.
	expectEqual(t, uint32('a'), a.cell.Codepoint())
	// The row flag is not updated by clearHyperlink.
	expect(t, a.row.hyperlink, "a.row.hyperlink")

	// Clearing again is a no-op.
	page.ClearHyperlink(a.cell)
	expectEqual(t, uint(1), page.HyperlinkCount())
	expectEqual(t, uint(1), pg3IRefs(page, id))

	// Clear the last user: the ref count reaches zero and the set has no
	// living item. The row flag still stays until updated.
	page.ClearHyperlink(b.cell)
	expect(t, !b.cell.Hyperlink(), "!b.cell.hyperlink")
	expectEqual(t, uint(0), page.HyperlinkCount())
	expectEqual(t, uint(0), pg3IRefs(page, id))
	expectEqual(t, uint(0), page.hyperlink_set.Count())
	expect(t, b.row.hyperlink, "b.row.hyperlink")
	page.UpdateRowHyperlinkFlag(b.row)
	expect(t, !b.row.hyperlink, "!b.row.hyperlink")
}

// Zig: test "term51: Page.clearHyperlink: cell without a link is untouched"
func TestPg3_Page_clearHyperlink_cell_without_a_link_is_untouched(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	id := pg3ILink(t, page, 1)
	a := pg3IPut(page, 0, 0, 'a')
	zigTry(t, page.SetHyperlink(a.row, a.cell, id))
	plain := pg3IPut(page, 5, 0, 'p')
	other_row := pg3IPut(page, 0, 3, 'q')

	before := pg3IBits(*plain.cell)
	page.ClearHyperlink(plain.cell)
	page.ClearHyperlink(other_row.cell)
	expectEqual(t, before, pg3IBits(*plain.cell))
	expectEqual(t, uint(1), page.HyperlinkCount())
	expectEqual(t, uint(1), pg3IRefs(page, id))
	expect(t, a.cell.Hyperlink(), "a.cell.hyperlink")
	expect(t, a.row.hyperlink, "a.row.hyperlink")
	expect(t, !other_row.row.hyperlink, "!other_row.row.hyperlink")

	// On a page with no hyperlinks at all.
	empty := pg3Init(t, pg3ICap())
	defer empty.Deinit()
	empty.ClearHyperlink(empty.GetRowAndCell(2, 2).cell)
	expectEqual(t, uint(0), empty.HyperlinkCount())
}

// Zig: test "term51: Page.updateRowHyperlinkFlag: clears the flag when no cell has a link"
func TestPg3_Page_updateRowHyperlinkFlag_clears_the_flag_when_no_cell_has_a_link(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	// Flag set but no cell with a link: cleared.
	row := page.GetRow(2)
	row.hyperlink = true
	page.UpdateRowHyperlinkFlag(row)
	expect(t, !row.hyperlink, "!row.hyperlink")

	// Flag already false: stays false.
	page.UpdateRowHyperlinkFlag(row)
	expect(t, !row.hyperlink, "!row.hyperlink")

	// A link in the first cell, the last cell, a middle cell: flag stays.
	id := pg3ILink(t, page, 1)
	n := 0
	for _, x := range []uint{0, 9, 4} {
		row3 := page.GetRow(3)
		cell := &page.GetCells(row3)[x]
		*cell = PageCellInit('x')
		if n > 0 {
			page.hyperlink_set.Use(id)
		}
		zigTry(t, page.SetHyperlink(row3, cell, id))
		n += 1
		row3.hyperlink = true
		page.UpdateRowHyperlinkFlag(row3)
		expect(t, row3.hyperlink, "row3.hyperlink")
	}

	// Clear them one by one: flag stays until the last one is gone.
	row3 := page.GetRow(3)
	cells := page.GetCells(row3)
	page.ClearHyperlink(&cells[0])
	page.UpdateRowHyperlinkFlag(row3)
	expect(t, row3.hyperlink, "row3.hyperlink")
	page.ClearHyperlink(&cells[9])
	page.UpdateRowHyperlinkFlag(row3)
	expect(t, row3.hyperlink, "row3.hyperlink")
	page.ClearHyperlink(&cells[4])
	page.UpdateRowHyperlinkFlag(row3)
	expect(t, !row3.hyperlink, "!row3.hyperlink")
}

// Zig: test "term51: Page.updateRowHyperlinkFlag: never sets the flag, other rows untouched"
func TestPg3_Page_updateRowHyperlinkFlag_never_sets_the_flag_other_rows_untouched(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	id := pg3ILink(t, page, 1)
	a := pg3IPut(page, 1, 1, 'a')
	zigTry(t, page.SetHyperlink(a.row, a.cell, id))
	expect(t, a.row.hyperlink, "a.row.hyperlink")

	// Row 5 has a flag but no link, row 1 has a link: updating row 5 does
	// not touch row 1.
	r5 := page.GetRow(5)
	r5.hyperlink = true
	page.UpdateRowHyperlinkFlag(r5)
	expect(t, !r5.hyperlink, "!r5.hyperlink")
	expect(t, a.row.hyperlink, "a.row.hyperlink")

	// The function looks at the cell flags, not at the map: with the cell
	// flag cleared by hand the row flag goes.
	a.cell.setHyperlink(false)
	page.UpdateRowHyperlinkFlag(a.row)
	expect(t, !a.row.hyperlink, "!a.row.hyperlink")
	// The flag is never set by update: a cell with the flag on a row
	// without the flag keeps the row flag false.
	a.cell.setHyperlink(true)
	page.UpdateRowHyperlinkFlag(a.row)
	expect(t, !a.row.hyperlink, "!a.row.hyperlink")
}

// Zig: test "term51: Page.updateRowHyperlinkFlag: scans only size.cols cells"
func TestPg3_Page_updateRowHyperlinkFlag_scans_only_size_cols_cells(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	id := pg3ILink(t, page, 1)
	a := pg3IPut(page, 7, 0, 'a')
	zigTry(t, page.SetHyperlink(a.row, a.cell, id))

	// Shrinking the page width hides the cell at x=7 from the scan.
	page.size.cols = 5
	page.UpdateRowHyperlinkFlag(a.row)
	expect(t, !a.row.hyperlink, "!a.row.hyperlink")

	// A cell inside the width is seen.
	page.size.cols = 10
	a.row.hyperlink = true
	page.size.cols = 8
	page.UpdateRowHyperlinkFlag(a.row)
	expect(t, a.row.hyperlink, "a.row.hyperlink")
}

// Zig: test "term51: Page.insertHyperlink: implicit id"
func TestPg3_Page_insertHyperlink_implicit_id(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	expectEqual(t, uint64(0), pg3IStrUsed(page))
	uri := []uint8("https://example.com/a")
	id, err := page.InsertHyperlink(pg3Implicit(7, uri))
	zigTry(t, err)
	// The first id handed out by the set is 1 (0 is reserved).
	expectEqual(t, HyperlinkID(1), id)
	expectEqual(t, uint(1), pg3IRefs(page, id))
	expectEqual(t, uint(1), page.hyperlink_set.Count())
	// Inserting does not touch the cell map.
	expectEqual(t, uint(0), page.HyperlinkCount())

	// The entry has the implicit id and a copy of the uri in page memory.
	entry := *page.hyperlink_set.Get(id)
	expectEqual(t, SizeOffsetInt(7), entry.id.Implicit)
	expectEqualStrings(t, "https://example.com/a", string(entry.uri))

	// One 32-byte string chunk was used for the uri.
	expectEqual(t, uint64(32), pg3IStrUsed(page))

	// The uri was copied, not referenced.
	expect(t, &entry.uri[0] != &uri[0], "uri_ptr != \"https://example.com/a\".ptr")
	// Zig-only: const base = @intFromPtr(page.memory.ptr);
	// Zig-only: try term51Itesting.expect(@intFromPtr(uri_ptr) >= base);
	// Zig-only: try term51Itesting.expect(@intFromPtr(uri_ptr) < base + page.memory.len);
}

// Zig: test "term51: Page.insertHyperlink: explicit id"
func TestPg3_Page_insertHyperlink_explicit_id(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	id, err := page.InsertHyperlink(pg3Explicit([]uint8("my-id"), []uint8("https://example.com/b")))
	zigTry(t, err)
	expectEqual(t, HyperlinkID(1), id)
	expectEqual(t, uint(1), pg3IRefs(page, id))

	entry := *page.hyperlink_set.Get(id)
	expectEqualStrings(t, "my-id", string(entry.id.Explicit))
	expectEqualStrings(t, "https://example.com/b", string(entry.uri))
	// The id is a separate allocation from the uri: 2 chunks of 32 bytes.
	expectEqual(t, uint64(64), pg3IStrUsed(page))
	// Zig: entry.id.explicit.offset.offset != entry.uri.offset.offset (the
	// offsets differ); the Go entries hold slices, so the identity of their
	// first elements stands for the offset.
	expect(t, &entry.id.Explicit[0] != &entry.uri[0], "entry.id.explicit.offset.offset != entry.uri.offset.offset")

	// Explicit ids are copied too.
	// Zig-only: const id_ptr = entry.id.explicit.slice(page.memory).ptr;
	// Zig-only: const base = @intFromPtr(page.memory.ptr);
	// Zig-only: try term51Itesting.expect(@intFromPtr(id_ptr) >= base);
	// Zig-only: try term51Itesting.expect(@intFromPtr(id_ptr) < base + page.memory.len);
}

// Zig: test "term51: Page.insertHyperlink: string bytes round up to 32-byte chunks"
func TestPg3_Page_insertHyperlink_string_bytes_round_up_to_32_byte_chunks(t *testing.T) {
	lens := []int{1, 31, 32, 33, 64, 65, 96, 1000, 2016}
	expected_chunks := []uint64{1, 1, 1, 2, 2, 3, 3, 32, 63}
	for i, length := range lens {
		chunks := expected_chunks[i]
		page := pg3Init(t, pg3ICap())
		defer page.Deinit()
		uri := pg3Bytes(length, 'u')
		id, err := page.InsertHyperlink(pg3Implicit(1, uri))
		zigTry(t, err)
		expectEqual(t, chunks*32, pg3IStrUsed(page))
		entry := *page.hyperlink_set.Get(id)
		expectEqualSlices(t, uri, entry.uri)
	}

	// The same rounding for an explicit id.
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()
	long_id := pg3Bytes(40, 'i')
	_, err := page.InsertHyperlink(pg3Explicit(long_id, []uint8("u")))
	zigTry(t, err)
	expectEqual(t, uint64(32+64), pg3IStrUsed(page))
}

// Zig: test "term51: Page.insertHyperlink: identical links share an id"
func TestPg3_Page_insertHyperlink_identical_links_share_an_id(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	link := pg3Explicit([]uint8("same"), []uint8("https://example.com/c"))
	id1, err := page.InsertHyperlink(link)
	zigTry(t, err)
	used := pg3IStrUsed(page)
	expectEqual(t, uint64(64), used)
	id2, err := page.InsertHyperlink(link)
	zigTry(t, err)
	expectEqual(t, id1, id2)
	expectEqual(t, uint(2), pg3IRefs(page, id1))
	expectEqual(t, uint(1), page.hyperlink_set.Count())
	// The strings allocated for the duplicate are freed again.
	expectEqual(t, used, pg3IStrUsed(page))

	// Same for implicit ids.
	imp := pg3Implicit(3, []uint8("https://example.com/d"))
	imp_id, err := page.InsertHyperlink(imp)
	zigTry(t, err)
	used2 := pg3IStrUsed(page)
	imp_again, err := page.InsertHyperlink(imp)
	zigTry(t, err)
	expectEqual(t, imp_id, imp_again)
	expectEqual(t, uint(2), pg3IRefs(page, imp_id))
	expectEqual(t, used2, pg3IStrUsed(page))
}

// Zig: test "term51: Page.insertHyperlink: different links get different ids"
func TestPg3_Page_insertHyperlink_different_links_get_different_ids(t *testing.T) {
	cap := PageCapacityInit(4, 4)
	cap.styles = 4
	cap.hyperlink_bytes = SizeHyperlinkCountInt(9 * pg3IItem)
	page := pg3Init(t, cap)
	defer page.Deinit()

	links := []Hyperlink{
		pg3Implicit(1, []uint8("u")),
		pg3Implicit(2, []uint8("u")),            // other implicit id
		pg3Implicit(1, []uint8("v")),            // other uri
		pg3Explicit([]uint8("1"), []uint8("u")), // explicit vs implicit
		pg3Explicit([]uint8("2"), []uint8("u")), // other explicit id
		pg3Implicit(0xFFFFFFFF, []uint8("u")),
	}
	ids := make([]HyperlinkID, len(links))
	for i, link := range links {
		id, err := page.InsertHyperlink(link)
		zigTry(t, err)
		ids[i] = id
		expectEqual(t, uint(1), pg3IRefs(page, ids[i]))
		// Ids are handed out in order.
		expectEqual(t, HyperlinkID(i+1), ids[i])
	}
	expectEqual(t, uint(len(links)), page.hyperlink_set.Count())

	// Reading them back.
	e := *page.hyperlink_set.Get(ids[5])
	expectEqual(t, SizeOffsetInt(0xFFFFFFFF), e.id.Implicit)
	// Inserting each again finds the existing id.
	for i, link := range links {
		again, err := page.InsertHyperlink(link)
		zigTry(t, err)
		expectEqual(t, ids[i], again)
		expectEqual(t, uint(2), pg3IRefs(page, ids[i]))
	}
}

// Zig: test "term51: Page.insertHyperlink: StringsOutOfMemory for the uri"
func TestPg3_Page_insertHyperlink_StringsOutOfMemory_for_the_uri(t *testing.T) {
	// No string memory at all.
	cap := PageCapacityInit(4, 4)
	cap.styles = 4
	cap.string_bytes = 0
	page := pg3Init(t, cap)
	defer page.Deinit()
	_, err := page.InsertHyperlink(pg3Implicit(1, []uint8("u")))
	pg3ExpectError(t, ErrStringsOutOfMemory, err)
	expectEqual(t, uint(0), page.hyperlink_set.Count())
	expectEqual(t, uint64(0), pg3IStrUsed(page))

	// Default string memory is 2048 bytes = 64 chunks. A uri that uses all
	// of them works (the id is implicit so it needs no string memory)...
	page2 := pg3Init(t, pg3ICap())
	defer page2.Deinit()
	uri := pg3Bytes(2048, 'u')
	id, err := page2.InsertHyperlink(pg3Implicit(1, uri))
	zigTry(t, err)
	expectEqual(t, uint64(2048), pg3IStrUsed(page2))
	// ... and the next string does not fit.
	_, err = page2.InsertHyperlink(pg3Implicit(2, []uint8("u")))
	pg3ExpectError(t, ErrStringsOutOfMemory, err)
	expectEqual(t, uint64(2048), pg3IStrUsed(page2))
	expectEqual(t, uint(1), page2.hyperlink_set.Count())
	expectEqual(t, uint(1), pg3IRefs(page2, id))
}

// Zig: test "term51: Page.insertHyperlink: StringsOutOfMemory for the id frees the uri"
func TestPg3_Page_insertHyperlink_StringsOutOfMemory_for_the_id_frees_the_uri(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	// A 63-chunk uri leaves exactly one free chunk.
	uri := pg3Bytes(2016, 'u')

	// A 32-byte id needs one chunk: fits.
	id1, err := page.InsertHyperlink(pg3Explicit(pg3Bytes(32, 'i'), uri))
	zigTry(t, err)
	expectEqual(t, uint64(2048), pg3IStrUsed(page))
	page.hyperlink_set.Release(id1)

	page2 := pg3Init(t, pg3ICap())
	defer page2.Deinit()
	// A 33-byte id needs two chunks: does not fit, and the uri that was
	// already allocated is freed again.
	_, err = page2.InsertHyperlink(pg3Explicit(pg3Bytes(33, 'i'), uri))
	pg3ExpectError(t, ErrStringsOutOfMemory, err)
	expectEqual(t, uint64(0), pg3IStrUsed(page2))
	expectEqual(t, uint(0), page2.hyperlink_set.Count())

	// With no string memory the uri fails first for an explicit id too.
	cap := PageCapacityInit(4, 4)
	cap.styles = 4
	cap.string_bytes = 0
	page3 := pg3Init(t, cap)
	defer page3.Deinit()
	_, err = page3.InsertHyperlink(pg3Explicit([]uint8("id"), []uint8("u")))
	pg3ExpectError(t, ErrStringsOutOfMemory, err)
}

// Zig: test "term51: Page.insertHyperlink: SetOutOfMemory frees the strings"
func TestPg3_Page_insertHyperlink_SetOutOfMemory_frees_the_strings(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	// The default hyperlink_bytes (4 items) leave room for 2 distinct links.
	id1 := pg3ILink(t, page, 1)
	id2 := pg3ILink(t, page, 2)
	expectEqual(t, HyperlinkID(1), id1)
	expectEqual(t, HyperlinkID(2), id2)
	used := pg3IStrUsed(page)
	expectEqual(t, uint64(64), used)

	_, err := page.InsertHyperlink(pg3Implicit(3, []uint8("https://example.com/other")))
	pg3ExpectError(t, ErrSetOutOfMemory, err)
	// The uri allocated for the failed insert was freed.
	expectEqual(t, used, pg3IStrUsed(page))
	expectEqual(t, uint(2), page.hyperlink_set.Count())

	// Same with an explicit id: both strings are freed.
	_, err = page.InsertHyperlink(pg3Explicit([]uint8("e"), []uint8("https://example.com/other")))
	pg3ExpectError(t, ErrSetOutOfMemory, err)
	expectEqual(t, used, pg3IStrUsed(page))

	// An existing link is still found when the set is full.
	expectEqual(t, id2, pg3ILink(t, page, 2))
	expectEqual(t, uint(2), pg3IRefs(page, id2))
}

// Zig: test "term51: Page.insertHyperlink: number of distinct links per hyperlink_bytes"
func TestPg3_Page_insertHyperlink_number_of_distinct_links_per_hyperlink_bytes(t *testing.T) {
	// hyperlink_bytes is divided by the size of a set item to get a count,
	// the set rounds the table up to a power of two and has a load factor of
	// 13/16, and id 0 is reserved: the usable number of distinct links is
	// less than the count.
	type Case struct{ items, links int }
	cases := []Case{
		{items: 0, links: 0},
		{items: 1, links: 0},
		{items: 2, links: 0},
		{items: 3, links: 2},
		{items: 4, links: 2},
		{items: 5, links: 5},
		{items: 8, links: 5},
		{items: 9, links: 12},
	}
	for _, c := range cases {
		cap := PageCapacityInit(4, 4)
		cap.styles = 4
		cap.hyperlink_bytes = SizeHyperlinkCountInt(c.items * pg3IItem)
		page := pg3Init(t, cap)
		defer page.Deinit()

		n := 0
		for {
			_, err := page.InsertHyperlink(pg3Implicit(SizeOffsetInt(n), []uint8("u")))
			if err != nil {
				pg3ExpectError(t, ErrSetOutOfMemory, err)
				break
			}
			n += 1
			expect(t, n < 100, "n < 100")
		}
		expectEqual(t, c.links, n)
	}

	// One byte short of three items is two items: no usable slot.
	cap := PageCapacityInit(4, 4)
	cap.styles = 4
	cap.hyperlink_bytes = SizeHyperlinkCountInt(3*pg3IItem - 1)
	page := pg3Init(t, cap)
	defer page.Deinit()
	_, err := pg3ILinkErr(page, 1)
	pg3ExpectError(t, ErrSetOutOfMemory, err)
	expectEqual(t, uint64(0), pg3IStrUsed(page))
}

// Zig: test "term51: Page.insertHyperlink: SetNeedsRehash after releasing a link below the last"
func TestPg3_Page_insertHyperlink_SetNeedsRehash_after_releasing_a_link_below_the_last(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	id1 := pg3ILink(t, page, 1)
	id2 := pg3ILink(t, page, 2)
	expectEqual(t, uint64(64), pg3IStrUsed(page))

	// Release the first link: it is dead but the second link is above it,
	// so its id cannot be reused without a rehash.
	page.hyperlink_set.Release(id1)
	expectEqual(t, uint(1), page.hyperlink_set.Count())
	_, err := page.InsertHyperlink(pg3Implicit(3, []uint8("https://example.com")))
	pg3ExpectError(t, ErrSetNeedsRehash, err)
	// The strings of the failed insert were freed.
	expectEqual(t, uint64(64), pg3IStrUsed(page))

	// Releasing the top link too lets the set trim both dead items (freeing
	// their strings) and the insert succeeds with the first id.
	page.hyperlink_set.Release(id2)
	id3, err := page.InsertHyperlink(pg3Implicit(3, []uint8("https://example.com")))
	zigTry(t, err)
	expectEqual(t, HyperlinkID(1), id3)
	expectEqual(t, uint64(32), pg3IStrUsed(page))
	expectEqual(t, uint(1), page.hyperlink_set.Count())
}

// Zig: test "term51: Page.setHyperlink: first link on a cell"
func TestPg3_Page_setHyperlink_first_link_on_a_cell(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	id := pg3ILink(t, page, 1)
	a := pg3IPut(page, 2, 3, 'a')
	expect(t, !a.row.hyperlink, "!a.row.hyperlink")
	zigTry(t, page.SetHyperlink(a.row, a.cell, id))

	expect(t, a.cell.Hyperlink(), "a.cell.hyperlink")
	expect(t, a.row.hyperlink, "a.row.hyperlink")
	expectEqual(t, Some(id), page.LookupHyperlink(a.cell))
	expectEqual(t, uint(1), page.HyperlinkCount())
	// The reference count is not incremented by setHyperlink.
	expectEqual(t, uint(1), pg3IRefs(page, id))
	// Text and other fields are untouched.
	expectEqual(t, uint32('a'), a.cell.Codepoint())

	// A second cell in the same row, then one in another row.
	page.hyperlink_set.Use(id)
	b := pg3IPut(page, 5, 3, 'b')
	zigTry(t, page.SetHyperlink(b.row, b.cell, id))
	expect(t, b.cell.Hyperlink(), "b.cell.hyperlink")
	expect(t, b.row.hyperlink, "b.row.hyperlink")
	expectEqual(t, uint(2), page.HyperlinkCount())
	page.hyperlink_set.Use(id)
	c := pg3IPut(page, 0, 9, 'c')
	zigTry(t, page.SetHyperlink(c.row, c.cell, id))
	expect(t, c.row.hyperlink, "c.row.hyperlink")
	expectEqual(t, uint(3), page.HyperlinkCount())
	expectEqual(t, uint(3), pg3IRefs(page, id))
	// A row with no links is not flagged.
	expect(t, !page.GetRow(0).hyperlink, "!page.getRow(0).hyperlink")
}

// Zig: test "term51: Page.setHyperlink: replacing a link releases the old one"
func TestPg3_Page_setHyperlink_replacing_a_link_releases_the_old_one(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	id1 := pg3ILink(t, page, 1)
	id2 := pg3ILink(t, page, 2)
	a := pg3IPut(page, 0, 0, 'a')
	zigTry(t, page.SetHyperlink(a.row, a.cell, id1))
	expectEqual(t, uint(1), pg3IRefs(page, id1))
	expectEqual(t, uint(2), page.hyperlink_set.Count())

	// id2 already has the reference from its insert; that moves to the cell.
	zigTry(t, page.SetHyperlink(a.row, a.cell, id2))
	expectEqual(t, Some(id2), page.LookupHyperlink(a.cell))
	expect(t, a.cell.Hyperlink(), "a.cell.hyperlink")
	expect(t, a.row.hyperlink, "a.row.hyperlink")
	expectEqual(t, uint(1), page.HyperlinkCount())
	expectEqual(t, uint(0), pg3IRefs(page, id1))
	expectEqual(t, uint(1), pg3IRefs(page, id2))
	expectEqual(t, uint(1), page.hyperlink_set.Count())
}

// Zig: test "term51: Page.setHyperlink: same id again keeps one reference per use"
func TestPg3_Page_setHyperlink_same_id_again_keeps_one_reference_per_use(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	id := pg3ILink(t, page, 1)
	a := pg3IPut(page, 0, 0, 'a')
	zigTry(t, page.SetHyperlink(a.row, a.cell, id))
	expectEqual(t, uint(1), pg3IRefs(page, id))

	// The caller increments the count for the new use (as for any new
	// link), the old reference is always released: net count unchanged.
	page.hyperlink_set.Use(id)
	expectEqual(t, uint(2), pg3IRefs(page, id))
	zigTry(t, page.SetHyperlink(a.row, a.cell, id))
	expectEqual(t, uint(1), pg3IRefs(page, id))
	expectEqual(t, uint(1), page.HyperlinkCount())
	expectEqual(t, Some(id), page.LookupHyperlink(a.cell))
	expect(t, a.cell.Hyperlink(), "a.cell.hyperlink")
	expect(t, a.row.hyperlink, "a.row.hyperlink")

	// The cell flag may be false while the map entry exists (printing over
	// a cell): setting the same id again sets the flag back.
	a.cell.setHyperlink(false)
	page.hyperlink_set.Use(id)
	zigTry(t, page.SetHyperlink(a.row, a.cell, id))
	expect(t, a.cell.Hyperlink(), "a.cell.hyperlink")
	expect(t, a.row.hyperlink, "a.row.hyperlink")
	expectEqual(t, uint(1), pg3IRefs(page, id))
	expectEqual(t, uint(1), page.HyperlinkCount())
}

// Zig: test "term51: Page.setHyperlink: HyperlinkMapOutOfMemory at the map capacity"
func TestPg3_Page_setHyperlink_HyperlinkMapOutOfMemory_at_the_map_capacity(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	// 4 hyperlink items * 16 cells per hyperlink: 64 cells with links.
	cap := page.HyperlinkCapacity()
	expectEqual(t, uint(64), cap)

	id := pg3ILink(t, page, 1)
	for i := uint(0); i < cap; i++ {
		x := i % 10
		y := i / 10
		rc := pg3IPut(page, x, y, 'a')
		if i > 0 {
			page.hyperlink_set.Use(id)
		}
		zigTry(t, page.SetHyperlink(rc.row, rc.cell, id))
		expectEqual(t, i+1, page.HyperlinkCount())
	}
	expectEqual(t, cap, page.HyperlinkCount())
	expectEqual(t, cap, pg3IRefs(page, id))

	// One more cell, in a row that has no link yet: error, nothing changed.
	extra := pg3IPut(page, 0, 7, 'z')
	pg3ExpectError(t, ErrHyperlinkMapOutOfMemory, page.SetHyperlink(extra.row, extra.cell, id))
	expect(t, !extra.cell.Hyperlink(), "!extra.cell.hyperlink")
	expect(t, !extra.row.hyperlink, "!extra.row.hyperlink")
	expectEqual(t, None[HyperlinkID](), page.LookupHyperlink(extra.cell))
	expectEqual(t, cap, page.HyperlinkCount())
	expectEqual(t, cap, pg3IRefs(page, id))

	// A cell that already has a link can still be changed at capacity.
	id2 := pg3ILink(t, page, 2)
	first := page.GetRowAndCell(0, 0)
	zigTry(t, page.SetHyperlink(first.row, first.cell, id2))
	expectEqual(t, Some(id2), page.LookupHyperlink(first.cell))
	expectEqual(t, cap, page.HyperlinkCount())
	expectEqual(t, cap-1, pg3IRefs(page, id))

	// Clearing a cell makes room again.
	page.ClearHyperlink(page.GetRowAndCell(1, 0).cell)
	expectEqual(t, cap-1, page.HyperlinkCount())
	page.hyperlink_set.Use(id)
	zigTry(t, page.SetHyperlink(extra.row, extra.cell, id))
	expectEqual(t, cap, page.HyperlinkCount())
	expect(t, extra.row.hyperlink, "extra.row.hyperlink")
}

// Zig: test "term51: Page.setHyperlink: no map memory"
func TestPg3_Page_setHyperlink_no_map_memory(t *testing.T) {
	// hyperlink_bytes smaller than one item gives a zero-capacity map.
	cap := PageCapacityInit(4, 4)
	cap.styles = 4
	cap.hyperlink_bytes = 0
	page := pg3Init(t, cap)
	defer page.Deinit()
	expectEqual(t, uint(0), page.HyperlinkCapacity())
	a := pg3IPut(page, 0, 0, 'a')
	// Any id will do: the map is full before the id is looked at.
	pg3ExpectError(t, ErrHyperlinkMapOutOfMemory, page.SetHyperlink(a.row, a.cell, 1))
	expect(t, !a.cell.Hyperlink(), "!a.cell.hyperlink")
	expect(t, !a.row.hyperlink, "!a.row.hyperlink")
	expectEqual(t, uint(0), page.HyperlinkCount())
}

// Zig: test "term51: Page.moveHyperlink: moves the map entry only"
func TestPg3_Page_moveHyperlink_moves_the_map_entry_only(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	id := pg3ILink(t, page, 1)
	src := pg3IPut(page, 1, 1, 'a')
	zigTry(t, page.SetHyperlink(src.row, src.cell, id))
	dst := pg3IPut(page, 6, 1, 'b')
	other := pg3IPut(page, 2, 2, 'c')
	page.hyperlink_set.Use(id)
	zigTry(t, page.SetHyperlink(other.row, other.cell, id))

	page.moveHyperlink(src.cell, dst.cell)

	// The entry moved: same id, count and reference count.
	expectEqual(t, None[HyperlinkID](), page.LookupHyperlink(src.cell))
	expectEqual(t, Some(id), page.LookupHyperlink(dst.cell))
	expectEqual(t, Some(id), page.LookupHyperlink(other.cell))
	expectEqual(t, uint(2), page.HyperlinkCount())
	expectEqual(t, uint(2), pg3IRefs(page, id))

	// The cell flags are NOT changed: the caller is responsible.
	expect(t, src.cell.Hyperlink(), "src.cell.hyperlink")
	expect(t, !dst.cell.Hyperlink(), "!dst.cell.hyperlink")

	// Once the caller fixes the flags the page is consistent again.
	src.cell.setHyperlink(false)
	dst.cell.setHyperlink(true)
	page.AssertIntegrity()
}

// Zig: test "term51: Page.moveHyperlink: across rows, chained, at capacity"
func TestPg3_Page_moveHyperlink_across_rows_chained_at_capacity(t *testing.T) {
	cap := PageCapacityInit(10, 10)
	cap.styles = 8
	cap.hyperlink_bytes = SizeHyperlinkCountInt(pg3IItem)
	page := pg3Init(t, cap)
	defer page.Deinit()
	// One item gives a map of 16 cells but no usable set slot, so build the
	// map entry by hand with a fabricated id. The map is filled to capacity:
	// moving an entry needs no free slot.
	expectEqual(t, uint(16), page.HyperlinkCapacity())

	m := page.hyperlink_map.Map()
	for i := 0; i < 16; i++ {
		cell := page.GetRowAndCell(uint(i%10), uint(i/10)).cell
		zigTry(t, m.Put(page.cellOffset(cell), HyperlinkID(100+i)))
	}
	expectEqual(t, uint(16), page.HyperlinkCount())
	expectEqual(t, page.HyperlinkCapacity(), page.HyperlinkCount())

	// Cell index 0 is (0,0); move it to (3,8): different row.
	src := page.GetRowAndCell(0, 0)
	dst := page.GetRowAndCell(3, 8)
	src.cell.setHyperlink(true)
	page.moveHyperlink(src.cell, dst.cell)
	expectEqual(t, None[HyperlinkID](), page.LookupHyperlink(src.cell))
	expectEqual(t, Some(HyperlinkID(100)), page.LookupHyperlink(dst.cell))
	expectEqual(t, uint(16), page.HyperlinkCount())

	// Chain it on to a third cell (the flag has to be moved by hand for the
	// assert in moveHyperlink).
	dst.cell.setHyperlink(true)
	third := page.GetRowAndCell(9, 9)
	page.moveHyperlink(dst.cell, third.cell)
	expectEqual(t, None[HyperlinkID](), page.LookupHyperlink(dst.cell))
	expectEqual(t, Some(HyperlinkID(100)), page.LookupHyperlink(third.cell))
	expectEqual(t, uint(16), page.HyperlinkCount())

	// Other entries are untouched.
	for i := 1; i < 16; i++ {
		cell := page.GetRowAndCell(uint(i%10), uint(i/10)).cell
		expectEqual(t, Some(HyperlinkID(100+i)), page.LookupHyperlink(cell))
	}
}

// Zig: test "term51: Page.hyperlinkCount: counts cells with a link"
func TestPg3_Page_hyperlinkCount_counts_cells_with_a_link(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()
	expectEqual(t, uint(0), page.HyperlinkCount())

	id1 := pg3ILink(t, page, 1)
	id2 := pg3ILink(t, page, 2)
	// The count is cells, not distinct links: three cells, one link.
	a := pg3IPut(page, 0, 0, 'a')
	zigTry(t, page.SetHyperlink(a.row, a.cell, id1))
	expectEqual(t, uint(1), page.HyperlinkCount())
	b := pg3IPut(page, 1, 0, 'b')
	page.hyperlink_set.Use(id1)
	zigTry(t, page.SetHyperlink(b.row, b.cell, id1))
	expectEqual(t, uint(2), page.HyperlinkCount())
	c := pg3IPut(page, 2, 0, 'c')
	page.hyperlink_set.Use(id1)
	zigTry(t, page.SetHyperlink(c.row, c.cell, id1))
	expectEqual(t, uint(3), page.HyperlinkCount())

	// Replacing the link of a cell does not change the count.
	zigTry(t, page.SetHyperlink(c.row, c.cell, id2))
	expectEqual(t, uint(3), page.HyperlinkCount())

	// Clearing decrements.
	page.ClearHyperlink(a.cell)
	expectEqual(t, uint(2), page.HyperlinkCount())
	page.ClearHyperlink(b.cell)
	page.ClearHyperlink(c.cell)
	expectEqual(t, uint(0), page.HyperlinkCount())
}

// Zig: test "term51: Page.hyperlinkCapacity: number of cells that can have a link"
func TestPg3_Page_hyperlinkCapacity_number_of_cells_that_can_have_a_link(t *testing.T) {
	// hyperlink_bytes / item size = number of items; the map holds 16 cells
	// per item rounded up to a power of two; no items, no map.
	type Case struct {
		bytes uint
		cap   uint
	}
	cases := []Case{
		{bytes: 0, cap: 0},
		{bytes: pg3IItem - 1, cap: 0},
		{bytes: pg3IItem, cap: 16},
		{bytes: 2*pg3IItem - 1, cap: 16},
		{bytes: 2 * pg3IItem, cap: 32},
		{bytes: 3 * pg3IItem, cap: 64},
		{bytes: 4 * pg3IItem, cap: 64},
		{bytes: 5 * pg3IItem, cap: 128},
		{bytes: 8 * pg3IItem, cap: 128},
		{bytes: 9 * pg3IItem, cap: 256},
		{bytes: 65535, cap: 32768},
	}
	for _, c := range cases {
		page := pg3Init(t, PageCapacity{
			cols:            1,
			rows:            1,
			styles:          0,
			hyperlink_bytes: SizeHyperlinkCountInt(c.bytes),
			grapheme_bytes:  0,
			string_bytes:    0,
		})
		defer page.Deinit()
		expectEqual(t, c.cap, page.HyperlinkCapacity())
		expectEqual(t, uint(0), page.HyperlinkCount())
	}

	// The default and the capacity does not change as links are added.
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()
	expectEqual(t, uint(64), page.HyperlinkCapacity())
	id := pg3ILink(t, page, 1)
	a := pg3IPut(page, 0, 0, 'a')
	zigTry(t, page.SetHyperlink(a.row, a.cell, id))
	expectEqual(t, uint(64), page.HyperlinkCapacity())
	expectEqual(t, SizeHyperlinkCountInt(192), PageCapacityInit(1, 1).hyperlink_bytes)
}

// ---- Graphemes ---------------------------------------------------------------

// Zig: test "term51: Page.setGraphemes: stores the codepoints on a text cell"
func TestPg3_Page_setGraphemes_stores_the_codepoints_on_a_text_cell(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	a := pg3IPut(page, 3, 2, 0x41)
	expect(t, !a.row.grapheme, "!a.row.grapheme")
	zigTry(t, page.SetGraphemes(a.row, a.cell, []uint32{0x301, 0x302, 0x303}))

	expectEqual(t, PageCellContentTagCodepointGrapheme, a.cell.ContentTag())
	expect(t, a.cell.HasGrapheme(), "a.cell.hasGrapheme()")
	expect(t, a.row.grapheme, "a.row.grapheme")
	// The first codepoint stays on the cell and is not part of the list.
	expectEqual(t, uint32(0x41), a.cell.Codepoint())
	expectEqualSlices(t, []uint32{0x301, 0x302, 0x303}, zigUnwrap(t, page.LookupGrapheme(a.cell)))
	expectEqual(t, uint(1), page.GraphemeCount())
	expectEqual(t, uint64(pg3IChunk), pg3IGrUsed(page))

	// Other cells and rows are untouched.
	expect(t, !page.GetRow(0).grapheme, "!page.getRow(0).grapheme")
	expect(t, !page.LookupGrapheme(page.GetRowAndCell(2, 2).cell).Valid, "page.lookupGrapheme(page.getRowAndCell(2, 2).cell) == null")
	page.AssertIntegrity()
}

// Zig: test "term51: Page.setGraphemes: chunk usage by length"
func TestPg3_Page_setGraphemes_chunk_usage_by_length(t *testing.T) {
	// 4 codepoints fit one 16-byte chunk, 5 need two, 9 need three.
	type Case struct {
		length int
		chunks uint64
	}
	cases := []Case{
		{length: 1, chunks: 1},
		{length: 2, chunks: 1},
		{length: 3, chunks: 1},
		{length: 4, chunks: 1},
		{length: 5, chunks: 2},
		{length: 8, chunks: 2},
		{length: 9, chunks: 3},
		{length: 16, chunks: 4},
		{length: 17, chunks: 5},
		{length: 100, chunks: 25},
	}
	var cps [100]uint32
	for i := range cps {
		cps[i] = uint32(0x300 + i)
	}

	for _, c := range cases {
		page := pg3Init(t, pg3ICap())
		defer page.Deinit()
		a := pg3IPut(page, 0, 0, 'e')
		zigTry(t, page.SetGraphemes(a.row, a.cell, cps[0:c.length]))
		expectEqual(t, c.chunks*pg3IChunk, pg3IGrUsed(page))
		expectEqualSlices(t, cps[0:c.length], zigUnwrap(t, page.LookupGrapheme(a.cell)))
		// The stored list is a copy.
		expect(t, &zigUnwrap(t, page.LookupGrapheme(a.cell))[0] != &cps[0], "page.lookupGrapheme(a.cell).?.ptr != &cps")
		page.AssertIntegrity()
	}
}

// Zig: test "term51: Page.setGraphemes: extreme codepoints and several cells"
func TestPg3_Page_setGraphemes_extreme_codepoints_and_several_cells(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	a := pg3IPut(page, 0, 0, 0x1FFFFF)
	zigTry(t, page.SetGraphemes(a.row, a.cell, []uint32{0, 0x10FFFF, 0x1FFFFF}))
	expectEqualSlices(t, []uint32{0, 0x10FFFF, 0x1FFFFF}, zigUnwrap(t, page.LookupGrapheme(a.cell)))
	expectEqual(t, uint32(0x1FFFFF), a.cell.Codepoint())

	b := pg3IPut(page, 9, 9, 'b')
	zigTry(t, page.SetGraphemes(b.row, b.cell, []uint32{0xFE0F}))
	c := pg3IPut(page, 4, 0, 'c')
	zigTry(t, page.SetGraphemes(c.row, c.cell, []uint32{0x1F3FB, 0x200D}))
	expectEqual(t, uint(3), page.GraphemeCount())
	expectEqualSlices(t, []uint32{0xFE0F}, zigUnwrap(t, page.LookupGrapheme(b.cell)))
	expectEqualSlices(t, []uint32{0x1F3FB, 0x200D}, zigUnwrap(t, page.LookupGrapheme(c.cell)))
	expectEqualSlices(t, []uint32{0, 0x10FFFF, 0x1FFFFF}, zigUnwrap(t, page.LookupGrapheme(a.cell)))
	expect(t, a.row.grapheme, "a.row.grapheme")
	expect(t, b.row.grapheme, "b.row.grapheme")
	expectEqual(t, uint64(3*pg3IChunk), pg3IGrUsed(page))
	page.AssertIntegrity()
}

// Zig: test "term51: Page.setGraphemes: only the content tag of the cell changes"
func TestPg3_Page_setGraphemes_only_the_content_tag_of_the_cell_changes(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	a := pg3IPut(page, 0, 0, 0x4E2D)
	a.cell.setWide(PageCellWideWide)
	a.cell.setProtected(true)
	a.cell.setSemanticContent(PageCellSemanticContentInput)
	before := pg3IBits(*a.cell)
	zigTry(t, page.SetGraphemes(a.row, a.cell, []uint32{0xFE0F}))
	// Only the 2 tag bits differ: 0 -> 1.
	expectEqual(t, before|1, pg3IBits(*a.cell))
	expectEqual(t, PageCellWideWide, a.cell.Wide())
	expect(t, a.cell.Protected(), "a.cell.protected")
	expectEqual(t, PageCellSemanticContentInput, a.cell.SemanticContent())
	expectEqual(t, uint32(0x4E2D), a.cell.Codepoint())
}

// Zig: test "term51: Page.setGraphemes: GraphemeAllocOutOfMemory"
func TestPg3_Page_setGraphemes_GraphemeAllocOutOfMemory(t *testing.T) {
	// No grapheme memory at all.
	cap := PageCapacityInit(10, 10)
	cap.styles = 8
	cap.grapheme_bytes = 0
	page := pg3Init(t, cap)
	defer page.Deinit()
	a := pg3IPut(page, 0, 0, 'a')
	pg3ExpectError(t, ErrGraphemeAllocOutOfMemory, page.SetGraphemes(a.row, a.cell, []uint32{0x301}))
	expectEqual(t, PageCellContentTagCodepoint, a.cell.ContentTag())
	expect(t, !a.row.grapheme, "!a.row.grapheme")
	expectEqual(t, uint(0), page.GraphemeCount())
	expectEqual(t, uint(0), page.GraphemeCapacity())

	// 1024 bytes = 64 chunks of 4 codepoints: 64 single-chunk graphemes fit
	// and the 65th does not.
	page2 := pg3Init(t, pg3ICap())
	defer page2.Deinit()
	expectEqual(t, uint(64), page2.GraphemeCapacity())
	for i := uint(0); i < 64; i++ {
		rc := pg3IPut(page2, i%10, i/10, 'a')
		zigTry(t, page2.SetGraphemes(rc.row, rc.cell, []uint32{0x301}))
	}
	expectEqual(t, uint(64), page2.GraphemeCount())
	expectEqual(t, uint64(1024), pg3IGrUsed(page2))
	extra := pg3IPut(page2, 4, 6, 'x')
	pg3ExpectError(t, ErrGraphemeAllocOutOfMemory, page2.SetGraphemes(extra.row, extra.cell, []uint32{0x301}))
	expectEqual(t, PageCellContentTagCodepoint, extra.cell.ContentTag())
	expectEqual(t, uint(64), page2.GraphemeCount())

	// Freeing one chunk makes room for a one-chunk grapheme, but not for a
	// two-chunk one.
	page2.ClearGrapheme(page2.GetRowAndCell(0, 0).cell)
	pg3ExpectError(t, ErrGraphemeAllocOutOfMemory, page2.SetGraphemes(extra.row, extra.cell, []uint32{1, 2, 3, 4, 5}))
	expectEqual(t, uint64(1008), pg3IGrUsed(page2))
	zigTry(t, page2.SetGraphemes(extra.row, extra.cell, []uint32{1, 2, 3, 4}))
	expectEqual(t, uint64(1024), pg3IGrUsed(page2))
	expectEqual(t, uint(64), page2.GraphemeCount())
	page2.AssertIntegrity()
}

// Zig: test "term51: Page.setGraphemes: GraphemeMapOutOfMemory frees the codepoints"
func TestPg3_Page_setGraphemes_GraphemeMapOutOfMemory_frees_the_codepoints(t *testing.T) {
	// 16 grapheme bytes: the allocator still has 64 chunks but the map can
	// hold only one cell.
	cap := PageCapacityInit(10, 10)
	cap.styles = 8
	cap.grapheme_bytes = 16
	page := pg3Init(t, cap)
	defer page.Deinit()
	expectEqual(t, uint(1), page.GraphemeCapacity())

	a := pg3IPut(page, 0, 0, 'a')
	zigTry(t, page.SetGraphemes(a.row, a.cell, []uint32{0x301, 0x302}))
	expectEqual(t, uint64(pg3IChunk), pg3IGrUsed(page))

	b := pg3IPut(page, 0, 1, 'b')
	pg3ExpectError(t, ErrGraphemeMapOutOfMemory, page.SetGraphemes(b.row, b.cell, []uint32{0x301, 0x302, 0x303, 0x304, 0x305}))
	// The allocation was given back, nothing on the cell or row changed.
	expectEqual(t, uint64(pg3IChunk), pg3IGrUsed(page))
	expectEqual(t, PageCellContentTagCodepoint, b.cell.ContentTag())
	expect(t, !b.row.grapheme, "!b.row.grapheme")
	expect(t, !page.LookupGrapheme(b.cell).Valid, "page.lookupGrapheme(b.cell) == null")
	expectEqual(t, uint(1), page.GraphemeCount())
	expectEqualSlices(t, []uint32{0x301, 0x302}, zigUnwrap(t, page.LookupGrapheme(a.cell)))

	// After clearing the first one the second fits.
	page.ClearGrapheme(a.cell)
	zigTry(t, page.SetGraphemes(b.row, b.cell, []uint32{0x301}))
	expectEqual(t, uint(1), page.GraphemeCount())
	expect(t, b.row.grapheme, "b.row.grapheme")
}

// Zig: test "term51: Page.appendGrapheme: first codepoint on a plain cell"
func TestPg3_Page_appendGrapheme_first_codepoint_on_a_plain_cell(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	a := pg3IPut(page, 1, 2, 'a')
	expect(t, !a.row.grapheme, "!a.row.grapheme")
	zigTry(t, page.AppendGrapheme(a.row, a.cell, 0x301))
	expect(t, a.cell.HasGrapheme(), "a.cell.hasGrapheme()")
	expect(t, a.row.grapheme, "a.row.grapheme")
	expectEqual(t, uint32('a'), a.cell.Codepoint())
	expectEqualSlices(t, []uint32{0x301}, zigUnwrap(t, page.LookupGrapheme(a.cell)))
	expectEqual(t, uint(1), page.GraphemeCount())
	expectEqual(t, uint64(pg3IChunk), pg3IGrUsed(page))

	// The largest codepoint is stored as is.
	b := pg3IPut(page, 2, 2, 'b')
	zigTry(t, page.AppendGrapheme(b.row, b.cell, 0x1FFFFF))
	expectEqualSlices(t, []uint32{0x1FFFFF}, zigUnwrap(t, page.LookupGrapheme(b.cell)))

	// Only the content tag of the cell changes.
	c := pg3IPut(page, 3, 2, 'c')
	c.cell.setWide(PageCellWideWide)
	c.cell.setProtected(true)
	before := pg3IBits(*c.cell)
	zigTry(t, page.AppendGrapheme(c.row, c.cell, 0x301))
	expectEqual(t, before|1, pg3IBits(*c.cell))
	page.AssertIntegrity()
}

// Zig: test "term51: Page.appendGrapheme: grows in place inside a chunk"
func TestPg3_Page_appendGrapheme_grows_in_place_inside_a_chunk(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	a := pg3IPut(page, 0, 0, 'a')
	zigTry(t, page.AppendGrapheme(a.row, a.cell, 0x301))
	first := zigUnwrap(t, page.LookupGrapheme(a.cell))
	ptr := &first[0]

	// 2nd, 3rd and 4th codepoint use the rest of the first chunk: same
	// memory, no new allocation.
	for n := 2; n < 5; n++ {
		zigTry(t, page.AppendGrapheme(a.row, a.cell, uint32(0x300+n)))
		cps := zigUnwrap(t, page.LookupGrapheme(a.cell))
		expectEqual(t, n, len(cps))
		expect(t, &cps[0] == ptr, "cps.ptr == ptr")
		expectEqual(t, uint64(pg3IChunk), pg3IGrUsed(page))
		expectEqual(t, uint(1), page.GraphemeCount())
	}
	expectEqualSlices(t, []uint32{0x301, 0x302, 0x303, 0x304}, zigUnwrap(t, page.LookupGrapheme(a.cell)))
	page.AssertIntegrity()
}

// Zig: test "term51: Page.appendGrapheme: grows into a larger allocation at a chunk boundary"
func TestPg3_Page_appendGrapheme_grows_into_a_larger_allocation_at_a_chunk_boundary(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	a := pg3IPut(page, 0, 0, 'a')
	var expected [20]uint32
	for i := 0; i < 20; i++ {
		expected[i] = uint32(0x300 + i)
		before := page.LookupGrapheme(a.cell)
		var old *uint32
		if before.Valid {
			old = &before.Value[0]
		}
		zigTry(t, page.AppendGrapheme(a.row, a.cell, expected[i]))
		cps := zigUnwrap(t, page.LookupGrapheme(a.cell))
		expectEqualSlices(t, expected[0:i+1], cps)

		length := uint64(i + 1)
		// Chunks in use: ceil(len / 4), the old allocation is freed.
		expectEqual(t, ((length+3)/4)*pg3IChunk, pg3IGrUsed(page))
		// The slice moves exactly when the previous length was a multiple
		// of 4 (the old chunk space was full).
		if before.Valid {
			if i%4 == 0 {
				expect(t, &cps[0] != old, "cps.ptr != old")
			} else {
				expect(t, &cps[0] == old, "cps.ptr == old")
			}
		}
		expectEqual(t, uint(1), page.GraphemeCount())
	}
	page.AssertIntegrity()
}

// Zig: test "term51: Page.appendGrapheme: OutOfMemory leaves the cell unchanged"
func TestPg3_Page_appendGrapheme_OutOfMemory_leaves_the_cell_unchanged(t *testing.T) {
	// No grapheme memory.
	cap := PageCapacityInit(10, 10)
	cap.styles = 8
	cap.grapheme_bytes = 0
	page := pg3Init(t, cap)
	defer page.Deinit()
	a := pg3IPut(page, 0, 0, 'a')
	before := pg3IBits(*a.cell)
	pg3ExpectError(t, ErrOutOfMemory, page.AppendGrapheme(a.row, a.cell, 0x301))
	expectEqual(t, before, pg3IBits(*a.cell))
	expect(t, !a.row.grapheme, "!a.row.grapheme")
	expectEqual(t, uint(0), page.GraphemeCount())

	// Map full (one entry) while the allocator has room: the allocation is
	// given back.
	cap2 := PageCapacityInit(10, 10)
	cap2.styles = 8
	cap2.grapheme_bytes = 16
	page2 := pg3Init(t, cap2)
	defer page2.Deinit()
	x := pg3IPut(page2, 0, 0, 'x')
	zigTry(t, page2.AppendGrapheme(x.row, x.cell, 0x301))
	y := pg3IPut(page2, 0, 1, 'y')
	ybefore := pg3IBits(*y.cell)
	pg3ExpectError(t, ErrOutOfMemory, page2.AppendGrapheme(y.row, y.cell, 0x301))
	expectEqual(t, ybefore, pg3IBits(*y.cell))
	expect(t, !y.row.grapheme, "!y.row.grapheme")
	expectEqual(t, uint64(pg3IChunk), pg3IGrUsed(page2))
	expectEqual(t, uint(1), page2.GraphemeCount())

	// Appending to the cell that has the entry still works (in place).
	zigTry(t, page2.AppendGrapheme(x.row, x.cell, 0x302))
	expectEqualSlices(t, []uint32{0x301, 0x302}, zigUnwrap(t, page2.LookupGrapheme(x.cell)))
}

// Zig: test "term51: Page.appendGrapheme: OutOfMemory when growing past a chunk"
func TestPg3_Page_appendGrapheme_OutOfMemory_when_growing_past_a_chunk(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	// Cell A takes the first chunk with 4 codepoints; 63 other cells take
	// the remaining chunks.
	a := pg3IPut(page, 0, 0, 'a')
	for i := 0; i < 4; i++ {
		zigTry(t, page.AppendGrapheme(a.row, a.cell, uint32(0x301+i)))
	}
	for i := uint(1); i < 64; i++ {
		rc := pg3IPut(page, i%10, i/10, 'o')
		zigTry(t, page.AppendGrapheme(rc.row, rc.cell, 0x301))
	}
	expectEqual(t, uint64(1024), pg3IGrUsed(page))
	expectEqual(t, uint(64), page.GraphemeCount())

	// Full: A cannot move to a 2-chunk allocation and stays as it was.
	pg3ExpectError(t, ErrOutOfMemory, page.AppendGrapheme(a.row, a.cell, 0x305))
	expectEqualSlices(t, []uint32{0x301, 0x302, 0x303, 0x304}, zigUnwrap(t, page.LookupGrapheme(a.cell)))
	expectEqual(t, uint64(1024), pg3IGrUsed(page))

	// One free chunk is not enough for 2 chunks.
	page.ClearGrapheme(page.GetRowAndCell(1, 0).cell)
	pg3ExpectError(t, ErrOutOfMemory, page.AppendGrapheme(a.row, a.cell, 0x305))
	expectEqualSlices(t, []uint32{0x301, 0x302, 0x303, 0x304}, zigUnwrap(t, page.LookupGrapheme(a.cell)))
	expectEqual(t, uint64(1008), pg3IGrUsed(page))

	// Two adjacent free chunks are: the new allocation takes them and the
	// old chunk is freed.
	page.ClearGrapheme(page.GetRowAndCell(2, 0).cell)
	zigTry(t, page.AppendGrapheme(a.row, a.cell, 0x305))
	expectEqualSlices(t, []uint32{0x301, 0x302, 0x303, 0x304, 0x305}, zigUnwrap(t, page.LookupGrapheme(a.cell)))
	expectEqual(t, uint64(1008), pg3IGrUsed(page))
	expectEqual(t, uint(62), page.GraphemeCount())
	// Cleanup flag/tag state of the cleared cells for the integrity check.
	page.UpdateRowGraphemeFlag(page.GetRow(0))
	page.AssertIntegrity()
}

// Zig: test "term51: Page.lookupGrapheme: null for a cell without graphemes"
func TestPg3_Page_lookupGrapheme_null_for_a_cell_without_graphemes(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	expect(t, !page.LookupGrapheme(page.GetRowAndCell(0, 0).cell).Valid, "page.lookupGrapheme(page.getRowAndCell(0, 0).cell) == null")
	a := pg3IPut(page, 1, 1, 'a')
	expect(t, !page.LookupGrapheme(a.cell).Valid, "page.lookupGrapheme(a.cell) == null")

	zigTry(t, page.AppendGrapheme(a.row, a.cell, 0x301))
	expectEqualSlices(t, []uint32{0x301}, zigUnwrap(t, page.LookupGrapheme(a.cell)))
	// The neighbours still have none.
	expect(t, !page.LookupGrapheme(page.GetRowAndCell(0, 1).cell).Valid, "page.lookupGrapheme(page.getRowAndCell(0, 1).cell) == null")
	expect(t, !page.LookupGrapheme(page.GetRowAndCell(2, 1).cell).Valid, "page.lookupGrapheme(page.getRowAndCell(2, 1).cell) == null")
	expect(t, !page.LookupGrapheme(page.GetRowAndCell(1, 0).cell).Valid, "page.lookupGrapheme(page.getRowAndCell(1, 0).cell) == null")
	expect(t, !page.LookupGrapheme(page.GetRowAndCell(1, 2).cell).Valid, "page.lookupGrapheme(page.getRowAndCell(1, 2).cell) == null")
}

// Zig: test "term51: Page.lookupGrapheme: consults the map, returns page memory"
func TestPg3_Page_lookupGrapheme_consults_the_map_returns_page_memory(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	a := pg3IPut(page, 1, 1, 'a')
	zigTry(t, page.AppendGrapheme(a.row, a.cell, 0x301))
	zigTry(t, page.AppendGrapheme(a.row, a.cell, 0x302))

	// The tag is not consulted: with the tag reset by hand the entry is
	// still found, and a tag set by hand does not make up an entry.
	a.cell.setContentTag(PageCellContentTagCodepoint)
	expectEqualSlices(t, []uint32{0x301, 0x302}, zigUnwrap(t, page.LookupGrapheme(a.cell)))
	a.cell.setContentTag(PageCellContentTagCodepointGrapheme)
	b := pg3IPut(page, 2, 1, 'b')
	b.cell.setContentTag(PageCellContentTagCodepointGrapheme)
	expect(t, !page.LookupGrapheme(b.cell).Valid, "page.lookupGrapheme(b.cell) == null")

	// The slice is a view of page memory: writing through it is visible to
	// later lookups.
	cps := zigUnwrap(t, page.LookupGrapheme(a.cell))
	cps[0] = 0x999
	expectEqualSlices(t, []uint32{0x999, 0x302}, zigUnwrap(t, page.LookupGrapheme(a.cell)))
	// Zig-only: const base = @intFromPtr(page.memory.ptr);
	// Zig-only: try term51Itesting.expect(@intFromPtr(cps.ptr) >= base);
	// Zig-only: try term51Itesting.expect(@intFromPtr(cps.ptr) < base + page.memory.len);
}

// Zig: test "term51: Page.moveGrapheme: moves the map entry and nothing else"
func TestPg3_Page_moveGrapheme_moves_the_map_entry_and_nothing_else(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	src := pg3IPut(page, 1, 1, 'a')
	zigTry(t, page.AppendGrapheme(src.row, src.cell, 0x301))
	zigTry(t, page.AppendGrapheme(src.row, src.cell, 0x302))
	dst := pg3IPut(page, 6, 1, 'b')
	slice_before := zigUnwrap(t, page.LookupGrapheme(src.cell))
	used := pg3IGrUsed(page)

	page.MoveGrapheme(src.cell, dst.cell)

	// The entry (same memory) is now found through the destination.
	expect(t, !page.LookupGrapheme(src.cell).Valid, "page.lookupGrapheme(src.cell) == null")
	moved := zigUnwrap(t, page.LookupGrapheme(dst.cell))
	expect(t, &moved[0] == &slice_before[0], "moved.ptr == slice_before.ptr")
	expectEqualSlices(t, []uint32{0x301, 0x302}, moved)
	expectEqual(t, uint(1), page.GraphemeCount())
	expectEqual(t, used, pg3IGrUsed(page))

	// The content tags are NOT changed (as the doc comment warns).
	expect(t, src.cell.HasGrapheme(), "src.cell.hasGrapheme()")
	expect(t, !dst.cell.HasGrapheme(), "!dst.cell.hasGrapheme()")
	expectEqual(t, uint32('a'), src.cell.Codepoint())
	expectEqual(t, uint32('b'), dst.cell.Codepoint())

	// Once the caller fixes the tags the page is consistent.
	src.cell.setContentTag(PageCellContentTagCodepoint)
	dst.cell.setContentTag(PageCellContentTagCodepointGrapheme)
	page.AssertIntegrity()
}

// Zig: test "term51: Page.moveGrapheme: across rows, chained, map at capacity"
func TestPg3_Page_moveGrapheme_across_rows_chained_map_at_capacity(t *testing.T) {
	// The map holds exactly one cell: moving needs no free slot.
	cap := PageCapacityInit(10, 10)
	cap.styles = 8
	cap.grapheme_bytes = 16
	page := pg3Init(t, cap)
	defer page.Deinit()
	expectEqual(t, uint(1), page.GraphemeCapacity())

	a := pg3IPut(page, 0, 0, 'a')
	zigTry(t, page.SetGraphemes(a.row, a.cell, []uint32{0x301, 0x302, 0x303, 0x304, 0x305}))
	expectEqual(t, uint(1), page.GraphemeCount())

	b := pg3IPut(page, 4, 5, 'b')
	page.MoveGrapheme(a.cell, b.cell)
	expect(t, !page.LookupGrapheme(a.cell).Valid, "page.lookupGrapheme(a.cell) == null")
	expectEqualSlices(t, []uint32{0x301, 0x302, 0x303, 0x304, 0x305}, zigUnwrap(t, page.LookupGrapheme(b.cell)))
	expectEqual(t, uint(1), page.GraphemeCount())
	expectEqual(t, uint64(2*pg3IChunk), pg3IGrUsed(page))

	// Chain on, with the tags fixed up by hand.
	a.cell.setContentTag(PageCellContentTagCodepoint)
	b.cell.setContentTag(PageCellContentTagCodepointGrapheme)
	c := pg3IPut(page, 9, 9, 'c')
	page.MoveGrapheme(b.cell, c.cell)
	expect(t, !page.LookupGrapheme(b.cell).Valid, "page.lookupGrapheme(b.cell) == null")
	expectEqualSlices(t, []uint32{0x301, 0x302, 0x303, 0x304, 0x305}, zigUnwrap(t, page.LookupGrapheme(c.cell)))

	// Move back to the very first cell.
	b.cell.setContentTag(PageCellContentTagCodepoint)
	c.cell.setContentTag(PageCellContentTagCodepointGrapheme)
	page.MoveGrapheme(c.cell, a.cell)
	expectEqualSlices(t, []uint32{0x301, 0x302, 0x303, 0x304, 0x305}, zigUnwrap(t, page.LookupGrapheme(a.cell)))
	expectEqual(t, uint(1), page.GraphemeCount())
}

// Zig: test "term51: Page.clearGrapheme: frees the codepoints and resets the tag"
func TestPg3_Page_clearGrapheme_frees_the_codepoints_and_resets_the_tag(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	a := pg3IPut(page, 0, 0, 'a')
	b := pg3IPut(page, 1, 0, 'b')
	for i := 0; i < 9; i++ {
		zigTry(t, page.AppendGrapheme(a.row, a.cell, uint32(0x301+i)))
	}
	zigTry(t, page.AppendGrapheme(b.row, b.cell, 0x301))
	// 9 codepoints take 3 chunks, 1 takes one.
	expectEqual(t, uint64(4*pg3IChunk), pg3IGrUsed(page))
	expectEqual(t, uint(2), page.GraphemeCount())

	a.cell.setWide(PageCellWideWide)
	page.ClearGrapheme(a.cell)
	expectEqual(t, PageCellContentTagCodepoint, a.cell.ContentTag())
	expect(t, !a.cell.HasGrapheme(), "!a.cell.hasGrapheme()")
	expect(t, !page.LookupGrapheme(a.cell).Valid, "page.lookupGrapheme(a.cell) == null")
	expectEqual(t, uint(1), page.GraphemeCount())
	expectEqual(t, uint64(pg3IChunk), pg3IGrUsed(page))
	// The first codepoint and other fields stay.
	expectEqual(t, uint32('a'), a.cell.Codepoint())
	expectEqual(t, PageCellWideWide, a.cell.Wide())
	// The row flag is not updated by clearGrapheme.
	expect(t, a.row.grapheme, "a.row.grapheme")
	// The other cell is untouched.
	expect(t, b.cell.HasGrapheme(), "b.cell.hasGrapheme()")
	expectEqualSlices(t, []uint32{0x301}, zigUnwrap(t, page.LookupGrapheme(b.cell)))

	// Clear the last one and the memory is entirely free again.
	page.ClearGrapheme(b.cell)
	expectEqual(t, uint(0), page.GraphemeCount())
	expectEqual(t, uint64(0), pg3IGrUsed(page))

	// The cell can take graphemes again.
	zigTry(t, page.AppendGrapheme(a.row, a.cell, 0x400))
	expectEqualSlices(t, []uint32{0x400}, zigUnwrap(t, page.LookupGrapheme(a.cell)))
	page.UpdateRowGraphemeFlag(a.row)
	expect(t, a.row.grapheme, "a.row.grapheme")
}

// Zig: test "term51: Page.updateRowGraphemeFlag: clears the flag when no cell has graphemes"
func TestPg3_Page_updateRowGraphemeFlag_clears_the_flag_when_no_cell_has_graphemes(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	row := page.GetRow(4)
	row.grapheme = true
	page.UpdateRowGraphemeFlag(row)
	expect(t, !row.grapheme, "!row.grapheme")
	// Never sets the flag.
	page.UpdateRowGraphemeFlag(row)
	expect(t, !row.grapheme, "!row.grapheme")

	// Graphemes in the first, last and a middle cell keep the flag until
	// the last is cleared.
	cells := page.GetCells(row)
	for _, x := range []int{0, 9, 5} {
		cells[x] = PageCellInit('a')
		zigTry(t, page.AppendGrapheme(row, &cells[x], 0x301))
	}
	expect(t, row.grapheme, "row.grapheme")
	for _, x := range []int{0, 9} {
		page.ClearGrapheme(&cells[x])
		page.UpdateRowGraphemeFlag(row)
		expect(t, row.grapheme, "row.grapheme")
	}
	page.ClearGrapheme(&cells[5])
	page.UpdateRowGraphemeFlag(row)
	expect(t, !row.grapheme, "!row.grapheme")
}

// Zig: test "term51: Page.updateRowGraphemeFlag: looks at the cell tags of this row only"
func TestPg3_Page_updateRowGraphemeFlag_looks_at_the_cell_tags_of_this_row_only(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	// A grapheme in row 1 does not keep the flag of row 2.
	a := pg3IPut(page, 0, 1, 'a')
	zigTry(t, page.AppendGrapheme(a.row, a.cell, 0x301))
	r2 := page.GetRow(2)
	r2.grapheme = true
	page.UpdateRowGraphemeFlag(r2)
	expect(t, !r2.grapheme, "!r2.grapheme")
	expect(t, a.row.grapheme, "a.row.grapheme")

	// The scan is by cell tag, not by map entry: a tag set by hand keeps the
	// flag, a missing tag with a map entry does not.
	b := pg3IPut(page, 3, 2, 'b')
	b.cell.setContentTag(PageCellContentTagCodepointGrapheme)
	r2.grapheme = true
	page.UpdateRowGraphemeFlag(r2)
	expect(t, r2.grapheme, "r2.grapheme")
	b.cell.setContentTag(PageCellContentTagCodepoint)
	a.cell.setContentTag(PageCellContentTagCodepoint)
	page.UpdateRowGraphemeFlag(a.row)
	expect(t, !a.row.grapheme, "!a.row.grapheme")
}

// Zig: test "term51: Page.updateRowGraphemeFlag: scans only size.cols cells"
func TestPg3_Page_updateRowGraphemeFlag_scans_only_size_cols_cells(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	a := pg3IPut(page, 7, 0, 'a')
	zigTry(t, page.AppendGrapheme(a.row, a.cell, 0x301))
	page.size.cols = 5
	page.UpdateRowGraphemeFlag(a.row)
	expect(t, !a.row.grapheme, "!a.row.grapheme")
	a.row.grapheme = true
	page.size.cols = 8
	page.UpdateRowGraphemeFlag(a.row)
	expect(t, a.row.grapheme, "a.row.grapheme")
}

// Zig: test "term51: Page.graphemeCount: counts cells with graphemes"
func TestPg3_Page_graphemeCount_counts_cells_with_graphemes(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()
	expectEqual(t, uint(0), page.GraphemeCount())

	a := pg3IPut(page, 0, 0, 'a')
	zigTry(t, page.AppendGrapheme(a.row, a.cell, 0x301))
	expectEqual(t, uint(1), page.GraphemeCount())
	// More codepoints on the same cell do not change the count.
	zigTry(t, page.AppendGrapheme(a.row, a.cell, 0x302))
	for i := 0; i < 10; i++ {
		zigTry(t, page.AppendGrapheme(a.row, a.cell, 0x303))
	}
	expectEqual(t, uint(1), page.GraphemeCount())
	b := pg3IPut(page, 1, 0, 'b')
	zigTry(t, page.SetGraphemes(b.row, b.cell, []uint32{1, 2, 3, 4, 5, 6}))
	expectEqual(t, uint(2), page.GraphemeCount())
	c := pg3IPut(page, 0, 5, 'c')
	zigTry(t, page.AppendGrapheme(c.row, c.cell, 0x301))
	expectEqual(t, uint(3), page.GraphemeCount())

	page.ClearGrapheme(b.cell)
	expectEqual(t, uint(2), page.GraphemeCount())
	page.ClearGrapheme(a.cell)
	page.ClearGrapheme(c.cell)
	expectEqual(t, uint(0), page.GraphemeCount())
}

// Zig: test "term51: Page.graphemeCapacity: number of cells that can have graphemes"
func TestPg3_Page_graphemeCapacity_number_of_cells_that_can_have_graphemes(t *testing.T) {
	// grapheme_bytes / 16 (chunk size) rounded up, then up to a power of two;
	// zero bytes give a zero capacity.
	type Case struct {
		bytes uint32
		cap   uint
	}
	cases := []Case{
		{bytes: 0, cap: 0},
		{bytes: 1, cap: 1},
		{bytes: 16, cap: 1},
		{bytes: 17, cap: 2},
		{bytes: 32, cap: 2},
		{bytes: 33, cap: 4},
		{bytes: 48, cap: 4},
		{bytes: 49, cap: 4},
		{bytes: 64, cap: 4},
		{bytes: 65, cap: 8},
		{bytes: 512, cap: 32},
		{bytes: 1024, cap: 64},
		{bytes: 1025, cap: 128},
		{bytes: 1040, cap: 128},
		{bytes: 1041, cap: 128},
		{bytes: 2048, cap: 128},
		{bytes: 2049, cap: 256},
		{bytes: 8192, cap: 512},
		{bytes: 8193, cap: 1024},
	}
	for _, c := range cases {
		page := pg3Init(t, PageCapacity{
			cols:            1,
			rows:            1,
			styles:          0,
			hyperlink_bytes: 0,
			grapheme_bytes:  c.bytes,
			string_bytes:    0,
		})
		defer page.Deinit()
		expectEqual(t, c.cap, page.GraphemeCapacity())
		expectEqual(t, uint(0), page.GraphemeCount())
	}

	// The capacity does not change as graphemes are added.
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()
	expectEqual(t, uint(64), page.GraphemeCapacity())
	a := pg3IPut(page, 0, 0, 'a')
	zigTry(t, page.AppendGrapheme(a.row, a.cell, 0x301))
	expectEqual(t, uint(64), page.GraphemeCapacity())
	expectEqual(t, SizeGraphemeBytesInt(1024), PageCapacityInit(1, 1).grapheme_bytes)
}

// ---- Row flags, dirty --------------------------------------------------------

// Zig: test "term51: Page.updateRowStyledFlag: clears the flag when no cell is styled"
func TestPg3_Page_updateRowStyledFlag_clears_the_flag_when_no_cell_is_styled(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	row := page.GetRow(6)
	row.styled = true
	page.UpdateRowStyledFlag(row)
	expect(t, !row.styled, "!row.styled")
	// Never sets it.
	page.UpdateRowStyledFlag(row)
	expect(t, !row.styled, "!row.styled")

	// A styled cell in the first, last and a middle position keeps it.
	cells := page.GetCells(row)
	for _, x := range []int{0, 9, 4} {
		row.styled = true
		cells[x].setStyleID(1)
		page.UpdateRowStyledFlag(row)
		expect(t, row.styled, "row.styled")
		cells[x].setStyleID(0)
		page.UpdateRowStyledFlag(row)
		expect(t, !row.styled, "!row.styled")
	}

	// Any non-default style id counts, including the largest.
	row.styled = true
	cells[3].setStyleID(0xFFFF)
	page.UpdateRowStyledFlag(row)
	expect(t, row.styled, "row.styled")
	cells[3].setStyleID(0)

	// A cell with only a background colour has no style id.
	row.styled = true
	cells[2] = pg3ITagged(PageCellContentTagBgColorRgb, 0xFFFFFF)
	page.UpdateRowStyledFlag(row)
	expect(t, !row.styled, "!row.styled")
}

// Zig: test "term51: Page.updateRowStyledFlag: this row only, size.cols only"
func TestPg3_Page_updateRowStyledFlag_this_row_only_size_cols_only(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	// A styled cell in another row does not matter.
	page.GetCells(page.GetRow(1))[0].setStyleID(1)
	r2 := page.GetRow(2)
	r2.styled = true
	page.UpdateRowStyledFlag(r2)
	expect(t, !r2.styled, "!r2.styled")

	// Cells beyond size.cols are not scanned.
	page.GetCells(r2)[7].setStyleID(1)
	r2.styled = true
	page.size.cols = 7
	page.UpdateRowStyledFlag(r2)
	expect(t, !r2.styled, "!r2.styled")
	r2.styled = true
	page.size.cols = 8
	page.UpdateRowStyledFlag(r2)
	expect(t, r2.styled, "r2.styled")
}

// Zig: test "term51: Page.isDirty: page flag or any row in size.rows"
func TestPg3_Page_isDirty_page_flag_or_any_row_in_size_rows(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()
	expect(t, !page.IsDirty(), "!page.isDirty()")

	// The page flag alone.
	page.dirty = true
	expect(t, page.IsDirty(), "page.isDirty()")
	page.dirty = false
	expect(t, !page.IsDirty(), "!page.isDirty()")

	// Any single row: first, a middle one, last.
	for _, y := range []uint{0, 5, 9} {
		page.GetRow(y).dirty = true
		expect(t, page.IsDirty(), "page.isDirty()")
		page.GetRow(y).dirty = false
		expect(t, !page.IsDirty(), "!page.isDirty()")
	}

	// Both together, then each cleared in turn.
	page.dirty = true
	page.GetRow(3).dirty = true
	expect(t, page.IsDirty(), "page.isDirty()")
	page.dirty = false
	expect(t, page.IsDirty(), "page.isDirty()")
	page.GetRow(3).dirty = false
	expect(t, !page.IsDirty(), "!page.isDirty()")

	// Other row flags are not dirtiness.
	for y := uint(0); y < 10; y++ {
		row := page.GetRow(y)
		row.wrap = true
		row.wrap_continuation = true
		row.grapheme = true
		row.styled = true
		row.hyperlink = true
		row.kitty_virtual_placeholder = true
		row.semantic_prompt = PageRowSemanticPromptPrompt
	}
	expect(t, !page.IsDirty(), "!page.isDirty()")

	// Content changes do not set the flags by themselves.
	pg3IPut(page, 3, 3, 'x')
	expect(t, !page.IsDirty(), "!page.isDirty()")
}

// Zig: test "term51: Page.isDirty: rows beyond size.rows are not looked at"
func TestPg3_Page_isDirty_rows_beyond_size_rows_are_not_looked_at(t *testing.T) {
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()

	page.size.rows = 5
	// Row index 5 and above are outside the page size: ignored.
	page.rows[5].dirty = true
	page.rows[9].dirty = true
	expect(t, !page.IsDirty(), "!page.isDirty()")
	// Row 4 is the last one inside.
	page.rows[4].dirty = true
	expect(t, page.IsDirty(), "page.isDirty()")
	page.rows[4].dirty = false
	expect(t, !page.IsDirty(), "!page.isDirty()")
	// The page flag always counts.
	page.dirty = true
	expect(t, page.IsDirty(), "page.isDirty()")
}

// ---- Page.layout and Capacity ------------------------------------------------

// pg3CeilPowerOfTwo is std.math.ceilPowerOfTwoAssert(usize, n).
func pg3CeilPowerOfTwo(n uint) uint {
	assert(n > 0)
	return uint(1) << bits.Len(n-1)
}

// pg3Cap is the Zig literal with all six fields spelled out, as the
// availableBitsForGrid / maxCols / adjust tables write it.
func pg3Cap(cols SizeCellCountInt, rows SizeCellCountInt, styles SizeStyleCountInt, hyperlink_bytes SizeHyperlinkCountInt, grapheme_bytes SizeGraphemeBytesInt, string_bytes SizeStringBytesInt) PageCapacity {
	return PageCapacity{
		cols:            cols,
		rows:            rows,
		styles:          styles,
		hyperlink_bytes: hyperlink_bytes,
		grapheme_bytes:  grapheme_bytes,
		string_bytes:    string_bytes,
	}
}

// pg3ICheckLayout is term51ICheckLayout: checks the structural relations
// every layout has: the sections follow each other in order with their
// alignments, and the total is a multiple of the page size that just covers
// the last section.
func pg3ICheckLayout(t *testing.T, cap PageCapacity) {
	t.Helper()
	l := PageLayoutOf(cap)
	rows := uint64(cap.rows)
	cols := uint64(cap.cols)

	expectEqual(t, cap, l.capacity)
	expectEqual(t, uint64(0), l.rows_start)
	expectEqual(t, rows*8, l.rows_size)
	// Rows are 8 bytes and cells are aligned to 8: no gap.
	expectEqual(t, rows*8, l.cells_start)
	expectEqual(t, rows*cols*8, l.cells_size)

	expectEqual(t, pg3AlignForward(l.cells_start+l.cells_size, 8), l.styles_start)
	expectEqual(t,
		pg3AlignForward(l.styles_start+l.styles_layout.total_size, 8),
		l.grapheme_alloc_start,
	)
	expectEqual(t,
		pg3AlignForward(l.grapheme_alloc_start+l.grapheme_alloc_layout.total_size, 8),
		l.grapheme_map_start,
	)
	expectEqual(t,
		pg3AlignForward(l.grapheme_map_start+l.grapheme_map_layout.total_size, 8),
		l.string_alloc_start,
	)
	expectEqual(t,
		pg3AlignForward(l.string_alloc_start+l.string_alloc_layout.total_size, 8),
		l.hyperlink_set_start,
	)
	expectEqual(t,
		pg3AlignForward(l.hyperlink_set_start+l.hyperlink_set_layout.total_size, 4),
		l.hyperlink_map_start,
	)
	end := l.hyperlink_map_start + l.hyperlink_map_layout.total_size
	expect(t, l.total_size >= end, "l.total_size >= end")
	expect(t, l.total_size-end < pg3PageSizeMin, "l.total_size - end < std.heap.page_size_min")
	expectEqual(t, uint64(0), l.total_size%pg3PageSizeMin)
}

// pg3IExpectLayoutCaps is term51IExpectLayoutCaps.
func pg3IExpectLayoutCaps(t *testing.T, l PageLayout, styles uint, galloc_bitmaps uint, gmap uint, str_bitmaps uint, hset uint, hmap uint) {
	t.Helper()
	expectEqual(t, styles, l.styles_layout.cap)
	expectEqual(t, galloc_bitmaps, l.grapheme_alloc_layout.bitmap_count)
	expectEqual(t, gmap, uint(l.grapheme_map_layout.capacity))
	expectEqual(t, str_bitmaps, l.string_alloc_layout.bitmap_count)
	expectEqual(t, hset, l.hyperlink_set_layout.cap)
	expectEqual(t, hmap, uint(l.hyperlink_map_layout.capacity))
}

// Zig: test "term51: Page.layout: layout of the smallest capacities"
func TestPg3_Page_layout_layout_of_the_smallest_capacities(t *testing.T) {
	// cols=1 rows=1, every managed capacity zero.
	l := PageLayoutOf(pg3IZeroMeta())
	expectEqual(t, uint64(4096), l.total_size)
	expectEqual(t, uint64(0), l.rows_start)
	expectEqual(t, uint64(8), l.rows_size)
	expectEqual(t, uint64(8), l.cells_start)
	expectEqual(t, uint64(8), l.cells_size)
	expectEqual(t, uint64(16), l.styles_start)
	expectEqual(t, uint64(0), l.styles_layout.total_size)
	expectEqual(t, uint64(16), l.grapheme_alloc_start)
	expectEqual(t, uint64(0), l.grapheme_alloc_layout.total_size)
	expectEqual(t, uint64(16), l.grapheme_map_start)
	// The two hash maps still need their 16-byte header at zero capacity.
	expectEqual(t, uint64(16), l.grapheme_map_layout.total_size)
	expectEqual(t, uint64(32), l.string_alloc_start)
	expectEqual(t, uint64(0), l.string_alloc_layout.total_size)
	expectEqual(t, uint64(32), l.hyperlink_set_start)
	expectEqual(t, uint64(0), l.hyperlink_set_layout.total_size)
	expectEqual(t, uint64(32), l.hyperlink_map_start)
	expectEqual(t, uint64(16), l.hyperlink_map_layout.total_size)
	pg3IExpectLayoutCaps(t, l, 0, 0, 0, 0, 0, 0)
	expectEqual(t, pg3IZeroMeta(), l.capacity)
	pg3ICheckLayout(t, pg3IZeroMeta())

	// cols=0 rows=0: no rows and no cells at all; still one page.
	zero := pg3IZeroMeta()
	zero.cols = 0
	zero.rows = 0
	z := PageLayoutOf(zero)
	expectEqual(t, uint64(4096), z.total_size)
	expectEqual(t, uint64(0), z.rows_size)
	expectEqual(t, uint64(0), z.cells_start)
	expectEqual(t, uint64(0), z.cells_size)
	expectEqual(t, uint64(0), z.styles_start)
	expectEqual(t, uint64(0), z.grapheme_map_start)
	expectEqual(t, uint64(16), z.string_alloc_start)
	expectEqual(t, uint64(16), z.hyperlink_set_start)
	expectEqual(t, uint64(16), z.hyperlink_map_start)
	pg3ICheckLayout(t, zero)

	// cols=0 with rows, and rows=0 with cols: no cells.
	c0 := pg3IZeroMeta()
	c0.cols = 0
	c0.rows = 5
	lc0 := PageLayoutOf(c0)
	expectEqual(t, uint64(40), lc0.rows_size)
	expectEqual(t, uint64(0), lc0.cells_size)
	expectEqual(t, uint64(40), lc0.cells_start)
	r0 := pg3IZeroMeta()
	r0.cols = 5
	r0.rows = 0
	lr0 := PageLayoutOf(r0)
	expectEqual(t, uint64(0), lr0.rows_size)
	expectEqual(t, uint64(0), lr0.cells_size)
}

// Zig: test "term51: Page.layout: layout of a 10x10 page with default capacities"
func TestPg3_Page_layout_layout_of_a_10x10_page_with_default_capacities(t *testing.T) {
	l := PageLayoutOf(pg3ICap())
	expectEqual(t, uint64(86016), l.total_size)
	expectEqual(t, uint64(0), l.rows_start)
	expectEqual(t, uint64(80), l.rows_size)
	expectEqual(t, uint64(80), l.cells_start)
	expectEqual(t, uint64(800), l.cells_size)
	expectEqual(t, uint64(880), l.styles_start)
	expectEqual(t, uint64(232), l.styles_layout.total_size)
	expectEqual(t, uint64(1112), l.grapheme_alloc_start)
	expectEqual(t, uint64(16392), l.grapheme_alloc_layout.total_size)
	expectEqual(t, uint64(17504), l.grapheme_map_start)
	expectEqual(t, uint64(1360), l.grapheme_map_layout.total_size)
	expectEqual(t, uint64(18864), l.string_alloc_start)
	expectEqual(t, uint64(65544), l.string_alloc_layout.total_size)
	expectEqual(t, uint64(84408), l.hyperlink_set_start)
	expectEqual(t, uint64(152), l.hyperlink_set_layout.total_size)
	expectEqual(t, uint64(84560), l.hyperlink_map_start)
	expectEqual(t, uint64(464), l.hyperlink_map_layout.total_size)
	pg3IExpectLayoutCaps(t, l, 6, 1, 64, 1, 3, 64)
	pg3ICheckLayout(t, pg3ICap())

	// A different capacity echoed back.
	expectEqual(t, pg3ICap(), l.capacity)
}

// Zig: test "term51: Page.layout: layout of std_capacity"
func TestPg3_Page_layout_layout_of_std_capacity(t *testing.T) {
	// Under test the standard capacity has 512 grapheme bytes
	// (pg3StdCapacity sets it; see the comment on PageStdCapacity).
	std_capacity := pg3StdCapacity()
	expectEqual(t, SizeCellCountInt(215), std_capacity.cols)
	expectEqual(t, SizeCellCountInt(215), std_capacity.rows)
	expectEqual(t, SizeStyleCountInt(128), std_capacity.styles)
	expectEqual(t, SizeGraphemeBytesInt(512), std_capacity.grapheme_bytes)
	expectEqual(t, SizeHyperlinkCountInt(192), std_capacity.hyperlink_bytes)
	expectEqual(t, SizeStringBytesInt(2048), std_capacity.string_bytes)

	l := PageLayoutOf(std_capacity)
	expectEqual(t, uint64(454656), l.total_size)
	expectEqual(t, uint64(1720), l.rows_size)
	expectEqual(t, uint64(1720), l.cells_start)
	expectEqual(t, uint64(369800), l.cells_size)
	expectEqual(t, uint64(371520), l.styles_start)
	expectEqual(t, uint64(4000), l.styles_layout.total_size)
	expectEqual(t, uint64(375520), l.grapheme_alloc_start)
	expectEqual(t, uint64(8200), l.grapheme_alloc_layout.total_size)
	expectEqual(t, uint64(383720), l.grapheme_map_start)
	expectEqual(t, uint64(688), l.grapheme_map_layout.total_size)
	expectEqual(t, uint64(384408), l.string_alloc_start)
	expectEqual(t, uint64(449952), l.hyperlink_set_start)
	expectEqual(t, uint64(450104), l.hyperlink_map_start)
	pg3IExpectLayoutCaps(t, l, 104, 1, 32, 1, 3, 64)
	pg3ICheckLayout(t, std_capacity)
}

// Zig: test "term51: Page.layout: layout of a maxed capacity"
func TestPg3_Page_layout_layout_of_a_maxed_capacity(t *testing.T) {
	cap := pg3MaxedCapacity()
	l := PageLayoutOf(cap)
	expectEqual(t, uint64(524280), l.rows_size)
	expectEqual(t, uint64(34358689800), l.cells_size)
	expectEqual(t, uint64(34359214080), l.styles_start)
	expectEqual(t, uint64(2048000), l.styles_layout.total_size)
	expectEqual(t, uint64(34361262080), l.grapheme_alloc_start)
	expectEqual(t, uint64(68753031168), l.grapheme_alloc_layout.total_size)
	expectEqual(t, uint64(103114293248), l.grapheme_map_start)
	expectEqual(t, uint64(5637144592), l.grapheme_map_layout.total_size)
	expectEqual(t, uint64(108751437840), l.string_alloc_start)
	expectEqual(t, uint64(137455730688), l.string_alloc_layout.total_size)
	expectEqual(t, uint64(246207168528), l.hyperlink_set_start)
	expectEqual(t, uint64(83968), l.hyperlink_set_layout.total_size)
	expectEqual(t, uint64(246207252496), l.hyperlink_map_start)
	expectEqual(t, uint64(229392), l.hyperlink_map_layout.total_size)
	expectEqual(t, uint64(246207483904), l.total_size)
	pg3IExpectLayoutCaps(t, l, 53248, 4194304, 268435456, 2097152, 1664, 32768)
	pg3ICheckLayout(t, cap)
}

// Zig: test "term51: Page.layout: layout by rows and columns"
func TestPg3_Page_layout_layout_by_rows_and_columns(t *testing.T) {
	// Rows and cells are 8 bytes each and are laid out first, so the sizes
	// are exact products.
	dims := []SizeCellCountInt{0, 1, 2, 3, 10, 80, 215, 1000, 65535}
	for _, rows := range dims {
		for _, cols := range dims {
			cap := pg3IZeroMeta()
			cap.rows = rows
			cap.cols = cols
			l := PageLayoutOf(cap)
			expectEqual(t, uint64(rows)*8, l.rows_size)
			expectEqual(t, uint64(rows)*8, l.cells_start)
			expectEqual(t, uint64(rows)*uint64(cols)*8, l.cells_size)
			expectEqual(t, l.cells_start+l.cells_size, l.styles_start)
			pg3ICheckLayout(t, cap)
		}
	}

	// Literal totals: the total is rounded up to a whole number of pages
	// and the zero-capacity metadata take 32 bytes.
	type Case struct {
		cols  SizeCellCountInt
		rows  SizeCellCountInt
		total uint64
	}
	cases := []Case{
		{cols: 1, rows: 1, total: 4096},
		{cols: 80, rows: 24, total: 4096 * 4},
		{cols: 100, rows: 1, total: 4096},
		{cols: 2, rows: 255, total: 8192},
		{cols: 1, rows: 254, total: 4096},
		{cols: 1, rows: 255, total: 8192},
		{cols: 1, rows: 32766, total: 524288},
		{cols: 1, rows: 32767, total: 528384},
		{cols: 1, rows: 65534, total: 1048576},
		{cols: 1, rows: 65535, total: 1052672},
	}
	for _, c := range cases {
		cap := pg3IZeroMeta()
		cap.cols = c.cols
		cap.rows = c.rows
		expectEqual(t, c.total, PageLayoutOf(cap).total_size)
	}
}

// Zig: test "term51: Page.layout: layout by styles"
func TestPg3_Page_layout_layout_by_styles(t *testing.T) {
	// styles -> style set layout: the table is the next power of two of
	// the count (u16 ids), 13/16 of it is usable and an item is 36 bytes.
	// Zig: @sizeOf(StyleSet.Item), which the wireframe keeps as
	// styleSetItemSizes.
	expectEqual(t, uint64(36), styleSetItemSizes.item_size)
	type Case struct {
		styles       SizeStyleCountInt
		cap          uint
		table_cap    uint
		items_start  uint64
		total        uint64
		galloc_start uint64
	}
	cases := []Case{
		{styles: 0, cap: 0, table_cap: 0, items_start: 0, total: 0, galloc_start: 16},
		{styles: 1, cap: 0, table_cap: 1, items_start: 4, total: 4, galloc_start: 24},
		{styles: 2, cap: 1, table_cap: 2, items_start: 4, total: 40, galloc_start: 56},
		{styles: 8, cap: 6, table_cap: 8, items_start: 16, total: 232, galloc_start: 248},
		{styles: 128, cap: 104, table_cap: 128, items_start: 256, total: 4000, galloc_start: 4016},
		{styles: 65535, cap: 53248, table_cap: 65536, items_start: 131072, total: 2048000, galloc_start: 2048016},
	}
	for _, c := range cases {
		cap := pg3IZeroMeta()
		cap.styles = c.styles
		l := PageLayoutOf(cap)
		expectEqual(t, c.cap, l.styles_layout.cap)
		expectEqual(t, c.table_cap, l.styles_layout.table_cap)
		expectEqual(t, uint64(0), l.styles_layout.table_start)
		expectEqual(t, c.items_start, l.styles_layout.items_start)
		expectEqual(t, c.total, l.styles_layout.total_size)
		expectEqual(t, uint64(16), l.styles_start)
		expectEqual(t, c.galloc_start, l.grapheme_alloc_start)
		pg3ICheckLayout(t, cap)
	}
	// The page total of the largest.
	cap := pg3IZeroMeta()
	cap.styles = 65535
	expectEqual(t, uint64(2052096), PageLayoutOf(cap).total_size)

	// Every count from 0 to 300 follows the rule.
	for n := uint(0); n < 301; n++ {
		cap.styles = SizeStyleCountInt(n)
		l := PageLayoutOf(cap)
		var table_cap uint
		if n != 0 {
			table_cap = pg3CeilPowerOfTwo(n)
		}
		expectEqual(t, table_cap, l.styles_layout.table_cap)
		expectEqual(t, table_cap*13/16, l.styles_layout.cap)
		pg3ICheckLayout(t, cap)
	}
}

// Zig: test "term51: Page.layout: layout by hyperlink_bytes"
func TestPg3_Page_layout_layout_by_hyperlink_bytes(t *testing.T) {
	// hyperlink_bytes / 48 = number of set items; the set table is rounded
	// to a power of two with 13/16 usable; the cell map holds 16 cells per
	// item rounded to a power of two.
	expectEqual(t, uint64(48), uint64(pg3IItem))
	type Case struct {
		bytes           SizeHyperlinkCountInt
		set_cap         uint
		set_table_cap   uint
		set_mask        uint
		set_items_start uint64
		set_total       uint64
		map_cap         uint
		map_total       uint64
		map_keys        uint64
		map_vals        uint64
		set_start       uint64
		map_start       uint64
		total           uint64
	}
	cases := []Case{
		{bytes: 0, set_cap: 0, set_table_cap: 0, set_mask: 0, set_items_start: 0, set_total: 0, map_cap: 0, map_total: 16, map_keys: 0, map_vals: 0, set_start: 32, map_start: 32, total: 4096},
		{bytes: 47, set_cap: 0, set_table_cap: 0, set_mask: 0, set_items_start: 0, set_total: 0, map_cap: 0, map_total: 16, map_keys: 0, map_vals: 0, set_start: 32, map_start: 32, total: 4096},
		{bytes: 48, set_cap: 0, set_table_cap: 1, set_mask: 0, set_items_start: 8, set_total: 8, map_cap: 16, map_total: 128, map_keys: 16, map_vals: 80, set_start: 32, map_start: 40, total: 4096},
		{bytes: 95, set_cap: 0, set_table_cap: 1, set_mask: 0, set_items_start: 8, set_total: 8, map_cap: 16, map_total: 128, map_keys: 16, map_vals: 80, set_start: 32, map_start: 40, total: 4096},
		{bytes: 96, set_cap: 1, set_table_cap: 2, set_mask: 1, set_items_start: 8, set_total: 56, map_cap: 32, map_total: 240, map_keys: 32, map_vals: 160, set_start: 32, map_start: 88, total: 4096},
		{bytes: 144, set_cap: 3, set_table_cap: 4, set_mask: 3, set_items_start: 8, set_total: 152, map_cap: 64, map_total: 464, map_keys: 64, map_vals: 320, set_start: 32, map_start: 184, total: 4096},
		{bytes: 192, set_cap: 3, set_table_cap: 4, set_mask: 3, set_items_start: 8, set_total: 152, map_cap: 64, map_total: 464, map_keys: 64, map_vals: 320, set_start: 32, map_start: 184, total: 4096},
		{bytes: 240, set_cap: 6, set_table_cap: 8, set_mask: 7, set_items_start: 16, set_total: 304, map_cap: 128, map_total: 912, map_keys: 128, map_vals: 640, set_start: 32, map_start: 336, total: 4096},
		{bytes: 65535, set_cap: 1664, set_table_cap: 2048, set_mask: 2047, set_items_start: 4096, set_total: 83968, map_cap: 32768, map_total: 229392, map_keys: 32768, map_vals: 163840, set_start: 32, map_start: 84000, total: 315392},
	}
	for _, c := range cases {
		cap := pg3IZeroMeta()
		cap.hyperlink_bytes = c.bytes
		l := PageLayoutOf(cap)
		expectEqual(t, c.set_cap, l.hyperlink_set_layout.cap)
		expectEqual(t, c.set_table_cap, l.hyperlink_set_layout.table_cap)
		expectEqual(t, c.set_mask, uint(l.hyperlink_set_layout.table_mask))
		expectEqual(t, uint64(0), l.hyperlink_set_layout.table_start)
		expectEqual(t, c.set_items_start, l.hyperlink_set_layout.items_start)
		expectEqual(t, c.set_total, l.hyperlink_set_layout.total_size)
		expectEqual(t, c.map_cap, uint(l.hyperlink_map_layout.capacity))
		expectEqual(t, c.map_total, l.hyperlink_map_layout.total_size)
		expectEqual(t, c.map_keys, l.hyperlink_map_layout.keys_start)
		expectEqual(t, c.map_vals, l.hyperlink_map_layout.vals_start)
		expectEqual(t, c.set_start, l.hyperlink_set_start)
		expectEqual(t, c.map_start, l.hyperlink_map_start)
		expectEqual(t, c.total, l.total_size)
		pg3ICheckLayout(t, cap)
	}

	// The rule for every byte count up to 1000 and a few larger ones.
	cap := pg3IZeroMeta()
	step := func(bytes uint) uint {
		if bytes < 1000 {
			return 1
		}
		return 25
	}
	for bytes := uint(0); bytes <= 1100; bytes += step(bytes) {
		cap.hyperlink_bytes = SizeHyperlinkCountInt(bytes)
		l := PageLayoutOf(cap)
		items := bytes / 48
		var table_cap uint
		if items != 0 {
			table_cap = pg3CeilPowerOfTwo(items)
		}
		expectEqual(t, table_cap*13/16, l.hyperlink_set_layout.cap)
		var map_cap uint
		if items != 0 {
			map_cap = pg3CeilPowerOfTwo(items * 16)
		}
		expectEqual(t, map_cap, uint(l.hyperlink_map_layout.capacity))
		pg3ICheckLayout(t, cap)
	}
}

// Zig: test "term51: Page.layout: layout by grapheme_bytes"
func TestPg3_Page_layout_layout_by_grapheme_bytes(t *testing.T) {
	// grapheme_bytes -> chunks of 16 bytes (rounded up); the allocator has a
	// bitmap word per 64 chunks; the map holds as many cells as there are
	// chunks, rounded to a power of two. The allocator layout reserves
	// 16 * grapheme_bytes of chunk memory (see the BitmapAllocator layout).
	type Case struct {
		bytes        SizeGraphemeBytesInt
		alloc_total  uint64
		bitmaps      uint
		chunks_start uint64
		map_cap      uint
		map_total    uint64
		map_keys     uint64
		map_vals     uint64
		map_start    uint64
		total        uint64
	}
	cases := []Case{
		{bytes: 0, alloc_total: 0, bitmaps: 0, chunks_start: 0, map_cap: 0, map_total: 16, map_keys: 0, map_vals: 0, map_start: 16, total: 4096},
		{bytes: 1, alloc_total: 264, bitmaps: 1, chunks_start: 8, map_cap: 1, map_total: 40, map_keys: 4, map_vals: 8, map_start: 280, total: 4096},
		{bytes: 16, alloc_total: 264, bitmaps: 1, chunks_start: 8, map_cap: 1, map_total: 40, map_keys: 4, map_vals: 8, map_start: 280, total: 4096},
		{bytes: 17, alloc_total: 520, bitmaps: 1, chunks_start: 8, map_cap: 2, map_total: 64, map_keys: 4, map_vals: 16, map_start: 536, total: 4096},
		{bytes: 32, alloc_total: 520, bitmaps: 1, chunks_start: 8, map_cap: 2, map_total: 64, map_keys: 4, map_vals: 16, map_start: 536, total: 4096},
		{bytes: 33, alloc_total: 776, bitmaps: 1, chunks_start: 8, map_cap: 4, map_total: 104, map_keys: 4, map_vals: 24, map_start: 792, total: 4096},
		{bytes: 48, alloc_total: 776, bitmaps: 1, chunks_start: 8, map_cap: 4, map_total: 104, map_keys: 4, map_vals: 24, map_start: 792, total: 4096},
		{bytes: 49, alloc_total: 1032, bitmaps: 1, chunks_start: 8, map_cap: 4, map_total: 104, map_keys: 4, map_vals: 24, map_start: 1048, total: 4096},
		{bytes: 64, alloc_total: 1032, bitmaps: 1, chunks_start: 8, map_cap: 4, map_total: 104, map_keys: 4, map_vals: 24, map_start: 1048, total: 4096},
		{bytes: 65, alloc_total: 1288, bitmaps: 1, chunks_start: 8, map_cap: 8, map_total: 184, map_keys: 8, map_vals: 40, map_start: 1304, total: 4096},
		{bytes: 1024, alloc_total: 16392, bitmaps: 1, chunks_start: 8, map_cap: 64, map_total: 1360, map_keys: 64, map_vals: 320, map_start: 16408, total: 20480},
		{bytes: 1040, alloc_total: 16656, bitmaps: 2, chunks_start: 16, map_cap: 128, map_total: 2704, map_keys: 128, map_vals: 640, map_start: 16672, total: 20480},
	}
	for _, c := range cases {
		cap := pg3IZeroMeta()
		cap.grapheme_bytes = c.bytes
		l := PageLayoutOf(cap)
		expectEqual(t, c.alloc_total, l.grapheme_alloc_layout.total_size)
		expectEqual(t, c.bitmaps, l.grapheme_alloc_layout.bitmap_count)
		expectEqual(t, uint64(0), l.grapheme_alloc_layout.bitmap_start)
		expectEqual(t, c.chunks_start, l.grapheme_alloc_layout.chunks_start)
		expectEqual(t, uint64(16), l.grapheme_alloc_start)
		expectEqual(t, c.map_cap, uint(l.grapheme_map_layout.capacity))
		expectEqual(t, c.map_total, l.grapheme_map_layout.total_size)
		expectEqual(t, c.map_keys, l.grapheme_map_layout.keys_start)
		expectEqual(t, c.map_vals, l.grapheme_map_layout.vals_start)
		expectEqual(t, c.map_start, l.grapheme_map_start)
		expectEqual(t, c.total, l.total_size)
		pg3ICheckLayout(t, cap)
	}

	// The largest value (nothing is allocated here, only laid out).
	big := pg3IZeroMeta()
	big.grapheme_bytes = 0xFFFFFFFF
	lb := PageLayoutOf(big)
	expectEqual(t, uint(4194304), lb.grapheme_alloc_layout.bitmap_count)
	expectEqual(t, uint64(33554432), lb.grapheme_alloc_layout.chunks_start)
	expectEqual(t, uint64(68753031168), lb.grapheme_alloc_layout.total_size)
	expectEqual(t, uint(268435456), uint(lb.grapheme_map_layout.capacity))
	expectEqual(t, uint64(5637144592), lb.grapheme_map_layout.total_size)
	expectEqual(t, uint64(68753031184), lb.grapheme_map_start)
	expectEqual(t, uint64(74390179840), lb.total_size)
	pg3ICheckLayout(t, big)

	// The rule for every byte count up to 5000.
	cap := pg3IZeroMeta()
	for n := uint64(0); n < 5001; n++ {
		cap.grapheme_bytes = SizeGraphemeBytesInt(n)
		l := PageLayoutOf(cap)
		chunks := (n + 15) / 16
		bitmaps := (chunks + 63) / 64
		expectEqual(t, uint(bitmaps), l.grapheme_alloc_layout.bitmap_count)
		expectEqual(t, bitmaps*8+chunks*16*16, l.grapheme_alloc_layout.total_size)
		var map_cap uint
		if n != 0 {
			map_cap = pg3CeilPowerOfTwo(uint(chunks))
		}
		expectEqual(t, map_cap, uint(l.grapheme_map_layout.capacity))
		pg3ICheckLayout(t, cap)
	}
}

// Zig: test "term51: Page.layout: layout by string_bytes"
func TestPg3_Page_layout_layout_by_string_bytes(t *testing.T) {
	// string_bytes -> chunks of 32 bytes (rounded up); a bitmap word per 64
	// chunks; the allocator layout reserves 32 * string_bytes of chunk
	// memory.
	type Case struct {
		bytes        SizeStringBytesInt
		total_alloc  uint64
		bitmaps      uint
		chunks_start uint64
		set_start    uint64
		total        uint64
	}
	cases := []Case{
		{bytes: 0, total_alloc: 0, bitmaps: 0, chunks_start: 0, set_start: 32, total: 4096},
		{bytes: 1, total_alloc: 1032, bitmaps: 1, chunks_start: 8, set_start: 1064, total: 4096},
		{bytes: 32, total_alloc: 1032, bitmaps: 1, chunks_start: 8, set_start: 1064, total: 4096},
		{bytes: 33, total_alloc: 2056, bitmaps: 1, chunks_start: 8, set_start: 2088, total: 4096},
		{bytes: 2048, total_alloc: 65544, bitmaps: 1, chunks_start: 8, set_start: 65576, total: 69632},
		{bytes: 2049, total_alloc: 66576, bitmaps: 2, chunks_start: 16, set_start: 66608, total: 69632},
	}
	for _, c := range cases {
		cap := pg3IZeroMeta()
		cap.string_bytes = c.bytes
		l := PageLayoutOf(cap)
		expectEqual(t, c.total_alloc, l.string_alloc_layout.total_size)
		expectEqual(t, c.bitmaps, l.string_alloc_layout.bitmap_count)
		expectEqual(t, uint64(0), l.string_alloc_layout.bitmap_start)
		expectEqual(t, c.chunks_start, l.string_alloc_layout.chunks_start)
		expectEqual(t, uint64(32), l.string_alloc_start)
		expectEqual(t, c.set_start, l.hyperlink_set_start)
		expectEqual(t, c.total, l.total_size)
		pg3ICheckLayout(t, cap)
	}

	// The rule for every byte count up to 5000.
	cap := pg3IZeroMeta()
	for n := uint64(0); n < 5001; n++ {
		cap.string_bytes = SizeStringBytesInt(n)
		l := PageLayoutOf(cap)
		chunks := (n + 31) / 32
		bitmaps := (chunks + 63) / 64
		expectEqual(t, uint(bitmaps), l.string_alloc_layout.bitmap_count)
		expectEqual(t, bitmaps*8+chunks*32*32, l.string_alloc_layout.total_size)
		pg3ICheckLayout(t, cap)
	}
}

// Zig: test "term51: Page.layout: layout of every capacity field combination"
func TestPg3_Page_layout_layout_of_every_capacity_field_combination(t *testing.T) {
	styles := []SizeStyleCountInt{0, 1, 16, 128, 1000}
	hyperlinks := []SizeHyperlinkCountInt{0, 48, 192, 1000}
	graphemes := []SizeGraphemeBytesInt{0, 1, 512, 1024, 1040, 8192}
	strings := []SizeStringBytesInt{0, 1, 2048, 2049, 4096}
	shapes := [][2]SizeCellCountInt{{0, 0}, {1, 1}, {80, 24}, {215, 215}, {1, 1000}, {1000, 1}}
	for _, shape := range shapes {
		for _, st := range styles {
			for _, hb := range hyperlinks {
				for _, gb := range graphemes {
					for _, sb := range strings {
						cap := PageCapacity{
							cols:            shape[0],
							rows:            shape[1],
							styles:          st,
							hyperlink_bytes: hb,
							grapheme_bytes:  gb,
							string_bytes:    sb,
						}
						pg3ICheckLayout(t, cap)
					}
				}
			}
		}
	}
}

// Zig: test "term51: Page.layout: layout is what Page.init allocates"
func TestPg3_Page_layout_layout_is_what_Page_init_allocates(t *testing.T) {
	// The page memory is the layout total, the capacity is echoed, and the
	// allocators and maps are set up from the layout.
	page := pg3Init(t, pg3ICap())
	defer page.Deinit()
	l := PageLayoutOf(pg3ICap())
	// Zig-only: try term51Itesting.expectEqual(l.total_size, page.memory.len);
	expectEqual(t, pg3ICap(), page.capacity)
	expectEqual(t, SizeCellCountInt(10), page.size.cols)
	expectEqual(t, SizeCellCountInt(10), page.size.rows)
	expectEqual(t, uint(64), page.GraphemeCapacity())
	expectEqual(t, uint(64), page.HyperlinkCapacity())
	expectEqual(t, l.grapheme_alloc_layout.bitmap_count, page.grapheme_alloc.bitmap_count)
	expectEqual(t, l.string_alloc_layout.bitmap_count, page.string_alloc.bitmap_count)
	expectEqual(t, l.hyperlink_set_layout.cap, page.hyperlink_set.layout.cap)
	expectEqual(t, l.styles_layout.cap, page.styles.layout.cap)
	expectEqual(t, uint64(64*16), page.grapheme_alloc.CapacityBytes())
	expectEqual(t, uint64(64*32), page.string_alloc.CapacityBytes())
}

// Zig: test "term51: Capacity.availableBitsForGrid: bits before the managed memory"
func TestPg3_Capacity_availableBitsForGrid_bits_before_the_managed_memory(t *testing.T) {
	// The meta sections are packed backwards from the end of the page: what
	// is left in front is the room for rows and cells.
	type Case struct {
		cap  PageCapacity
		bits uint64
	}
	maxed := pg3MaxedCapacity()
	big_gb := pg3IZeroMeta()
	big_gb.grapheme_bytes = 0xFFFFFFFF
	cases := []Case{
		{cap: pg3IZeroMeta(), bits: 32512},
		{cap: pg3Cap(0, 0, 0, 0, 0, 0), bits: 32512},
		{cap: pg3ICap(), bits: 14976},
		{cap: pg3StdCapacity(), bits: 3004864},
		{cap: pg3Cap(1, 1, 1, 0, 0, 0), bits: 32448},
		{cap: pg3Cap(1, 1, 2, 0, 0, 0), bits: 32192},
		{cap: pg3Cap(1, 1, 65535, 0, 0, 0), bits: 32512},
		{cap: pg3Cap(1, 1, 0, 47, 0, 0), bits: 32512},
		{cap: pg3Cap(1, 1, 0, 48, 0, 0), bits: 31552},
		{cap: pg3Cap(1, 1, 0, 96, 0, 0), bits: 30272},
		{cap: pg3Cap(1, 1, 0, 144, 0, 0), bits: 27712},
		{cap: pg3Cap(1, 1, 0, 192, 0, 0), bits: 27712},
		{cap: pg3Cap(1, 1, 0, 240, 0, 0), bits: 22912},
		{cap: pg3Cap(1, 1, 0, 65535, 0, 0), bits: 16128},
		{cap: pg3Cap(1, 1, 0, 0, 1, 0), bits: 30208},
		{cap: pg3Cap(1, 1, 0, 0, 16, 0), bits: 30208},
		{cap: pg3Cap(1, 1, 0, 0, 17, 0), bits: 27968},
		{cap: pg3Cap(1, 1, 0, 0, 33, 0), bits: 25600},
		{cap: pg3Cap(1, 1, 0, 0, 49, 0), bits: 23552},
		{cap: pg3Cap(1, 1, 0, 0, 65, 0), bits: 20864},
		{cap: pg3Cap(1, 1, 0, 0, 1024, 0), bits: 21696},
		{cap: pg3Cap(1, 1, 0, 0, 1040, 0), bits: 8832},
		{cap: big_gb, bits: 32512},
		{cap: pg3Cap(1, 1, 0, 0, 0, 1), bits: 24256},
		{cap: pg3Cap(1, 1, 0, 0, 0, 33), bits: 16064},
		{cap: pg3Cap(1, 1, 0, 0, 0, 2048), bits: 32448},
		{cap: pg3Cap(1, 1, 0, 0, 0, 2049), bits: 24192},
		{cap: pg3Cap(0, 0, 0, 0, 26721, 0), bits: 64},
		{cap: maxed, bits: 274873728768},
	}
	for _, c := range cases {
		expectEqual(t, c.bits, c.cap.availableBitsForGrid())
		// Always whole 8-byte words.
		expectEqual(t, uint64(0), c.bits%64)
	}
}

// Zig: test "term51: Capacity.availableBitsForGrid: depends on the page total, not the grid shape"
func TestPg3_Capacity_availableBitsForGrid_depends_on_the_page_total_not_the_grid_shape(t *testing.T) {
	// Everything that rounds to the same total gives the same answer.
	cap := pg3IZeroMeta()
	for _, cols := range []SizeCellCountInt{1, 2, 10, 100, 200, 250, 507} {
		cap.cols = cols
		expectEqual(t, uint64(4096), PageLayoutOf(cap).total_size)
		expectEqual(t, uint64(32512), cap.availableBitsForGrid())
	}
	// One cell more rounds the page up by one page and the room grows by a
	// whole page (4096 bytes = 32768 bits).
	cap.cols = 508
	expectEqual(t, uint64(8192), PageLayoutOf(cap).total_size)
	expectEqual(t, uint64(32512+32768), cap.availableBitsForGrid())

	// The room is at least what the rows and cells use, and less than that
	// plus a page.
	e := PageCapacityInit(80, 24)
	f := pg3Cap(5, 7, 100, 1000, 3000, 5000)
	caps := []PageCapacity{pg3ICap(), pg3StdCapacity(), pg3IZeroMeta(), e, f}
	for _, c := range caps {
		l := PageLayoutOf(c)
		avail_bytes := c.availableBitsForGrid() / 8
		expect(t, avail_bytes >= l.styles_start, "avail_bytes >= l.styles_start")
		expect(t, avail_bytes < l.styles_start+pg3PageSizeMin+64, "avail_bytes < l.styles_start + std.heap.page_size_min + 64")
		expect(t, avail_bytes <= l.total_size, "avail_bytes <= l.total_size")
	}
}

// Zig: test "term51: Capacity.maxCols: columns that fit at the same page size"
func TestPg3_Capacity_maxCols_columns_that_fit_at_the_same_page_size(t *testing.T) {
	type Case struct {
		cap  PageCapacity
		cols Optional[SizeCellCountInt]
	}
	cases := []Case{
		{cap: pg3IZeroMeta(), cols: Some(SizeCellCountInt(507))},
		{cap: pg3ICap(), cols: Some(SizeCellCountInt(233))},
		{cap: pg3StdCapacity(), cols: Some(SizeCellCountInt(46950))},
		{cap: pg3Cap(1, 1, 1, 0, 0, 0), cols: Some(SizeCellCountInt(506))},
		{cap: pg3Cap(1, 1, 2, 0, 0, 0), cols: Some(SizeCellCountInt(502))},
		{cap: pg3Cap(1, 1, 0, 48, 0, 0), cols: Some(SizeCellCountInt(492))},
		{cap: pg3Cap(1, 1, 0, 65535, 0, 0), cols: Some(SizeCellCountInt(251))},
		{cap: pg3Cap(1, 1, 0, 0, 1040, 0), cols: Some(SizeCellCountInt(137))},
		{cap: pg3Cap(1, 1, 0, 0, 0, 2049), cols: Some(SizeCellCountInt(377))},
		// Room for two columns (avail 192 bits) and for one (128 bits).
		{cap: pg3Cap(1, 1, 0, 0, 24688, 0), cols: Some(SizeCellCountInt(2))},
		{cap: pg3Cap(1, 1, 0, 0, 25697, 0), cols: Some(SizeCellCountInt(1))},
		// Only 64 bits left, the size of a row: nothing fits.
		{cap: pg3Cap(0, 0, 0, 0, 26721, 0), cols: None[SizeCellCountInt]()},
		{cap: pg3Cap(0, 1, 0, 0, 26721, 0), cols: None[SizeCellCountInt]()},
	}
	for _, c := range cases {
		expectEqual(t, c.cols, c.cap.MaxCols())
		// maxCols is (room - one row) / one cell, when there is any.
		if c.cols.Valid {
			expectEqual(t,
				(c.cap.availableBitsForGrid()-64)/64,
				uint64(c.cols.Value),
			)
		} else {
			expect(t, c.cap.availableBitsForGrid() <= 64, "case.cap.availableBitsForGrid() <= 64")
		}
	}
}

// Zig: test "term51: Capacity.maxCols: clamps to the largest column count"
func TestPg3_Capacity_maxCols_clamps_to_the_largest_column_count(t *testing.T) {
	// Just below the clamp: 128 pages of room give 65531 columns.
	cap := pg3IZeroMeta()
	cap.rows = 32766
	expectEqual(t, uint64(524288), PageLayoutOf(cap).total_size)
	expectEqual(t, Some(SizeCellCountInt(65531)), cap.MaxCols())

	// One more row needs a 129th page and the room exceeds the limit.
	cap.rows = 32767
	expectEqual(t, uint64(528384), PageLayoutOf(cap).total_size)
	expectEqual(t, Some(SizeCellCountInt(65535)), cap.MaxCols())
	cap.rows = 65534
	expectEqual(t, Some(SizeCellCountInt(65535)), cap.MaxCols())
	cap.rows = 65535
	expectEqual(t, Some(SizeCellCountInt(65535)), cap.MaxCols())

	maxed := pg3MaxedCapacity()
	expectEqual(t, Some(SizeCellCountInt(65535)), maxed.MaxCols())
}

// Zig: test "term51: Capacity.maxCols: the smallest answers"
func TestPg3_Capacity_maxCols_the_smallest_answers(t *testing.T) {
	// 128 bits of room: one row plus one column.
	one := pg3Cap(1, 1, 0, 0, 25697, 0)
	expectEqual(t, uint64(128), one.availableBitsForGrid())
	expectEqual(t, Some(SizeCellCountInt(1)), one.MaxCols())
	// 192 bits: two columns.
	two := pg3Cap(1, 1, 0, 0, 24688, 0)
	expectEqual(t, uint64(192), two.availableBitsForGrid())
	expectEqual(t, Some(SizeCellCountInt(2)), two.MaxCols())
	// 64 bits: not even one column (equal to a row is not enough).
	none := pg3Cap(0, 0, 0, 0, 26721, 0)
	expectEqual(t, uint64(64), none.availableBitsForGrid())
	expectEqual(t, None[SizeCellCountInt](), none.MaxCols())
	// Cap fields other than the meta ones do not matter.
	other := none
	other.cols = 1
	other.rows = 0
	expectEqual(t, None[SizeCellCountInt](), other.MaxCols())
}

// Zig: test "term51: Capacity.adjust: without a column request nothing changes"
func TestPg3_Capacity_adjust_without_a_column_request_nothing_changes(t *testing.T) {
	caps := []PageCapacity{pg3ICap(), pg3StdCapacity(), pg3IZeroMeta(), PageCapacityInit(80, 24)}
	for _, cap := range caps {
		same, err := cap.Adjust(PageCapacityAdjustment{})
		zigTry(t, err)
		expectEqual(t, cap, same)
		explicit_null, err := cap.Adjust(PageCapacityAdjustment{cols: None[SizeCellCountInt]()})
		zigTry(t, err)
		expectEqual(t, cap, explicit_null)
	}
	// The default adjustment has no columns.
	expectEqual(t, None[SizeCellCountInt](), PageCapacityAdjustment{}.cols)
}

// Zig: test "term51: Capacity.adjust: columns change the rows to keep the total size"
func TestPg3_Capacity_adjust_columns_change_the_rows_to_keep_the_total_size(t *testing.T) {
	// term51ICap has room for 14976 bits: rows = floor(14976 / (64 + 64 cols)).
	type Case struct {
		cols SizeCellCountInt
		rows SizeCellCountInt
	}
	cases := []Case{
		{cols: 0, rows: 234},
		{cols: 1, rows: 117},
		{cols: 2, rows: 78},
		{cols: 10, rows: 21},
		{cols: 20, rows: 11},
		{cols: 116, rows: 2},
		{cols: 117, rows: 1},
		{cols: 233, rows: 1},
	}
	icap := pg3ICap()
	original := PageLayoutOf(icap).total_size
	for _, c := range cases {
		adjusted, err := icap.Adjust(PageCapacityAdjustment{cols: Some(c.cols)})
		zigTry(t, err)
		expectEqual(t, c.cols, adjusted.cols)
		expectEqual(t, c.rows, adjusted.rows)
		// Everything else is carried over.
		expectEqual(t, icap.styles, adjusted.styles)
		expectEqual(t, icap.hyperlink_bytes, adjusted.hyperlink_bytes)
		expectEqual(t, icap.grapheme_bytes, adjusted.grapheme_bytes)
		expectEqual(t, icap.string_bytes, adjusted.string_bytes)
		// Same total size, and one more row would not fit.
		expectEqual(t, original, PageLayoutOf(adjusted).total_size)
		bigger := adjusted
		bigger.rows += 1
		expect(t, PageLayoutOf(bigger).total_size > original, "Page.layout(bigger).total_size > original")
	}

	// The page can still be created with the adjusted capacity.
	adjusted, err := icap.Adjust(PageCapacityAdjustment{cols: Some(SizeCellCountInt(50))})
	zigTry(t, err)
	page := pg3Init(t, adjusted)
	defer page.Deinit()
	expectEqual(t, SizeCellCountInt(50), page.size.cols)
	expectEqual(t, adjusted.rows, page.size.rows)
}

// Zig: test "term51: Capacity.adjust: the same columns can still gain rows"
func TestPg3_Capacity_adjust_the_same_columns_can_still_gain_rows(t *testing.T) {
	// Rows are recomputed from the room, so keeping the columns grows the
	// rows into the unused part of the last page.
	icap := pg3ICap()
	same, err := icap.Adjust(PageCapacityAdjustment{cols: Some(SizeCellCountInt(10))})
	zigTry(t, err)
	expectEqual(t, SizeCellCountInt(10), same.cols)
	expectEqual(t, SizeCellCountInt(21), same.rows)
	// And it is idempotent.
	again, err := same.Adjust(PageCapacityAdjustment{cols: Some(SizeCellCountInt(10))})
	zigTry(t, err)
	expectEqual(t, same, again)

	std_capacity := pg3StdCapacity()
	std_same, err := std_capacity.Adjust(PageCapacityAdjustment{cols: Some(SizeCellCountInt(215))})
	zigTry(t, err)
	expectEqual(t, SizeCellCountInt(217), std_same.rows)
	// std_capacity itself: 3004864 / (64 + 64 * 1) = 23475 rows of 1 column.
	std_one, err := std_capacity.Adjust(PageCapacityAdjustment{cols: Some(SizeCellCountInt(1))})
	zigTry(t, err)
	expectEqual(t, SizeCellCountInt(23475), std_one.rows)
	std_zero, err := std_capacity.Adjust(PageCapacityAdjustment{cols: Some(SizeCellCountInt(0))})
	zigTry(t, err)
	expectEqual(t, SizeCellCountInt(46951), std_zero.rows)
}

// Zig: test "term51: Capacity.adjust: OutOfMemory one column past maxCols"
func TestPg3_Capacity_adjust_OutOfMemory_one_column_past_maxCols(t *testing.T) {
	caps := []PageCapacity{
		pg3ICap(),
		pg3StdCapacity(),
		pg3IZeroMeta(),
		PageCapacityInit(80, 24),
		pg3Cap(1, 1, 0, 0, 25697, 0),
		pg3Cap(1, 1, 0, 0, 24688, 0),
	}
	for _, cap := range caps {
		max := zigUnwrap(t, cap.MaxCols())
		// Exactly maxCols fits one row.
		fit, err := cap.Adjust(PageCapacityAdjustment{cols: Some(max)})
		zigTry(t, err)
		expectEqual(t, max, fit.cols)
		expectEqual(t, SizeCellCountInt(1), fit.rows)
		expectEqual(t, PageLayoutOf(cap).total_size, PageLayoutOf(fit).total_size)
		// One more does not.
		_, err = cap.Adjust(PageCapacityAdjustment{cols: Some(max + 1)})
		pg3ExpectError(t, ErrOutOfMemory, err)
		// Neither does the largest column count.
		_, err = cap.Adjust(PageCapacityAdjustment{cols: Some(SizeCellCountInt(65535))})
		pg3ExpectError(t, ErrOutOfMemory, err)
	}

	// With room for one column only: 1 works, 2 does not; 0 columns works.
	one := pg3Cap(1, 1, 0, 0, 25697, 0)
	r1, err := one.Adjust(PageCapacityAdjustment{cols: Some(SizeCellCountInt(1))})
	zigTry(t, err)
	expectEqual(t, SizeCellCountInt(1), r1.rows)
	_, err = one.Adjust(PageCapacityAdjustment{cols: Some(SizeCellCountInt(2))})
	pg3ExpectError(t, ErrOutOfMemory, err)
	r0, err := one.Adjust(PageCapacityAdjustment{cols: Some(SizeCellCountInt(0))})
	zigTry(t, err)
	expectEqual(t, SizeCellCountInt(2), r0.rows)

	// A capacity where maxCols is null: no column fits, a row of zero
	// columns still does (one row of 64 bits).
	none := pg3Cap(0, 0, 0, 0, 26721, 0)
	_, err = none.Adjust(PageCapacityAdjustment{cols: Some(SizeCellCountInt(1))})
	pg3ExpectError(t, ErrOutOfMemory, err)
	zero_cols, err := none.Adjust(PageCapacityAdjustment{cols: Some(SizeCellCountInt(0))})
	zigTry(t, err)
	expectEqual(t, SizeCellCountInt(0), zero_cols.cols)
	expectEqual(t, SizeCellCountInt(1), zero_cols.rows)
}

// Zig: test "term51: Capacity.adjust: never grows the page, keeps its size while a row is small"
func TestPg3_Capacity_adjust_never_grows_the_page_keeps_its_size_while_a_row_is_small(t *testing.T) {
	caps := []PageCapacity{
		pg3ICap(),
		pg3StdCapacity(),
		pg3IZeroMeta(),
		pg3Cap(80, 24, 16, 1000, 3000, 5000),
		pg3Cap(1, 1, 0, 65535, 0, 0),
		pg3Cap(3, 3, 500, 0, 100, 0),
	}
	for _, cap := range caps {
		original := PageLayoutOf(cap).total_size
		avail := cap.availableBitsForGrid()
		max := uint(zigUnwrap(t, cap.MaxCols()))
		step := max / 400
		if step < 1 {
			step = 1
		}
		for cols := uint(0); cols <= max; cols += step {
			adjusted, err := cap.Adjust(PageCapacityAdjustment{cols: Some(SizeCellCountInt(cols))})
			zigTry(t, err)
			expectEqual(t, cols, uint(adjusted.cols))
			expectEqual(t, avail/(64+64*uint64(cols)), uint64(adjusted.rows))
			expect(t, adjusted.rows >= 1, "adjusted.rows >= 1")
			// The page never gets bigger. It keeps its size exactly while a
			// row is much smaller than a page.
			adjusted_total := PageLayoutOf(adjusted).total_size
			expect(t, adjusted_total <= original, "adjusted_total <= original")
			if 8+8*cols <= 2048 {
				expectEqual(t, original, adjusted_total)
			}
			// One more row would need more than the room there is.
			bigger := adjusted
			bigger.rows += 1
			expect(t, PageLayoutOf(bigger).total_size > original, "Page.layout(bigger).total_size > original")
		}
	}
}

// Zig: test "term51: Capacity.adjust: shrinks the page when the leftover is more than a page"
func TestPg3_Capacity_adjust_shrinks_the_page_when_the_leftover_is_more_than_a_page(t *testing.T) {
	// Rows are whole: with wide rows the room left over after the last row
	// can be more than a page, and the adjusted capacity then needs fewer
	// pages than the original (std_capacity is 111 pages).
	std_capacity := pg3StdCapacity()
	expectEqual(t, uint64(454656), PageLayoutOf(std_capacity).total_size)
	type Case struct {
		cols  SizeCellCountInt
		rows  SizeCellCountInt
		total uint64
	}
	cases := []Case{
		{cols: 545, rows: 85, total: 450560},
		{cols: 558, rows: 83, total: 450560},
		{cols: 634, rows: 73, total: 450560},
	}
	for _, c := range cases {
		adjusted, err := std_capacity.Adjust(PageCapacityAdjustment{cols: Some(c.cols)})
		zigTry(t, err)
		expectEqual(t, c.rows, adjusted.rows)
		expectEqual(t, c.total, PageLayoutOf(adjusted).total_size)
		bigger := adjusted
		bigger.rows += 1
		expect(t, PageLayoutOf(bigger).total_size > PageLayoutOf(std_capacity).total_size, "Page.layout(bigger).total_size > Page.layout(std_capacity).total_size")
	}
}

// Zig: test "term51: Capacity.adjust: largest row count that fits the row type"
func TestPg3_Capacity_adjust_largest_row_count_that_fits_the_row_type(t *testing.T) {
	// 1 column of 8-byte cells and an 8-byte row: with 256 pages of room
	// the rows come out at 65534, the last count below the u16 limit.
	cap := pg3IZeroMeta()
	cap.rows = 65534
	expectEqual(t, uint64(1048576), PageLayoutOf(cap).total_size)
	adjusted, err := cap.Adjust(PageCapacityAdjustment{cols: Some(SizeCellCountInt(1))})
	zigTry(t, err)
	expectEqual(t, SizeCellCountInt(1), adjusted.cols)
	expectEqual(t, SizeCellCountInt(65534), adjusted.rows)
	expectEqual(t, uint64(1048576), PageLayoutOf(adjusted).total_size)

	// Just below: 128 pages.
	cap.rows = 32766
	half, err := cap.Adjust(PageCapacityAdjustment{cols: Some(SizeCellCountInt(1))})
	zigTry(t, err)
	expectEqual(t, SizeCellCountInt(32766), half.rows)
	cap.rows = 32767
	half_plus, err := cap.Adjust(PageCapacityAdjustment{cols: Some(SizeCellCountInt(1))})
	zigTry(t, err)
	expectEqual(t, SizeCellCountInt(33022), half_plus.rows)
}

// ---- Go-only panic tests -----------------------------------------------------
//
// Zig cannot test an assert or a safety trap (the test would abort). Each test
// below reaches one of them through the Go API and expects the port to stop
// where upstream stops. pg3ExpectPanic does not accept the wireframe's
// not-implemented panic.

// Go-only: panic test: Page.setGraphemes: assert(cell.codepoint() > 0), cell with codepoint 0
func TestPg3_panic_Page_setGraphemes_cell_codepoint_is_zero(t *testing.T) {
	page := pg3HNew(t, 4, 1)
	defer page.Deinit()
	rac := page.GetRowAndCell(0, 0)
	pg3ExpectPanic(t, func() {
		_ = page.SetGraphemes(rac.row, rac.cell, []uint32{0x301})
	})
}

// Go-only: panic test: Page.setGraphemes: assert(cell.content_tag == .codepoint), cell that already has graphemes
func TestPg3_panic_Page_setGraphemes_cell_content_tag_is_not_codepoint(t *testing.T) {
	page := pg3HNew(t, 4, 1)
	defer page.Deinit()
	pg3HGrapheme(t, page, 0, 0, 'a', []uint32{0x301})
	rac := page.GetRowAndCell(0, 0)
	pg3ExpectPanic(t, func() {
		_ = page.SetGraphemes(rac.row, rac.cell, []uint32{0x302})
	})
}

// Go-only: panic test: Page.appendGrapheme: assert(cell.codepoint() != 0), cell with codepoint 0
func TestPg3_panic_Page_appendGrapheme_cell_codepoint_is_zero(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	page := pg3HNew(t, 4, 1)
	defer page.Deinit()
	rac := page.GetRowAndCell(0, 0)
	pg3ExpectPanic(t, func() {
		_ = page.AppendGrapheme(rac.row, rac.cell, 0x301)
	})
}

// Go-only: panic test: Page.appendGrapheme: assert(row.grapheme), cell tagged as grapheme in a row without the flag
func TestPg3_panic_Page_appendGrapheme_row_grapheme_flag_is_false(t *testing.T) {
	page := pg3HNew(t, 4, 1)
	defer page.Deinit()
	a := pg3IPut(page, 0, 0, 'a')
	a.cell.setContentTag(PageCellContentTagCodepointGrapheme)
	pg3ExpectPanic(t, func() {
		_ = page.AppendGrapheme(a.row, a.cell, 0x301)
	})
}

// Go-only: panic test: Page.appendGrapheme: map.getPtr(cell_offset).?, cell tagged as grapheme in a flagged row without a map entry
func TestPg3_panic_Page_appendGrapheme_no_map_entry(t *testing.T) {
	page := pg3HNew(t, 4, 1)
	defer page.Deinit()
	a := pg3IPut(page, 0, 0, 'a')
	a.cell.setContentTag(PageCellContentTagCodepointGrapheme)
	a.row.grapheme = true
	pg3ExpectPanic(t, func() {
		_ = page.AppendGrapheme(a.row, a.cell, 0x301)
	})
}

// Go-only: panic test: Page.moveGrapheme: assert(src.hasGrapheme()), source cell without graphemes
func TestPg3_panic_Page_moveGrapheme_src_has_no_grapheme(t *testing.T) {
	page := pg3HNew(t, 4, 1)
	defer page.Deinit()
	src := pg3IPut(page, 0, 0, 'a')
	dst := pg3IPut(page, 1, 0, 'b')
	pg3ExpectPanic(t, func() {
		page.MoveGrapheme(src.cell, dst.cell)
	})
}

// Go-only: panic test: Page.moveGrapheme: assert(!dst.hasGrapheme()), destination cell with graphemes
func TestPg3_panic_Page_moveGrapheme_dst_has_grapheme(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	page := pg3HNew(t, 4, 1)
	defer page.Deinit()
	pg3HGrapheme(t, page, 0, 0, 'a', []uint32{0x301})
	pg3HGrapheme(t, page, 1, 0, 'b', []uint32{0x302})
	src := page.GetRowAndCell(0, 0)
	dst := page.GetRowAndCell(1, 0)
	pg3ExpectPanic(t, func() {
		page.MoveGrapheme(src.cell, dst.cell)
	})
}

// Go-only: panic test: Page.moveGrapheme: map.getEntry(src_offset).?, source cell tagged as grapheme without a map entry
func TestPg3_panic_Page_moveGrapheme_no_map_entry(t *testing.T) {
	page := pg3HNew(t, 4, 1)
	defer page.Deinit()
	src := pg3IPut(page, 0, 0, 'a')
	src.cell.setContentTag(PageCellContentTagCodepointGrapheme)
	dst := pg3IPut(page, 1, 0, 'b')
	pg3ExpectPanic(t, func() {
		page.MoveGrapheme(src.cell, dst.cell)
	})
}

// Go-only: panic test: Page.clearGrapheme: assert(cell.hasGrapheme()), cell without graphemes
func TestPg3_panic_Page_clearGrapheme_cell_has_no_grapheme(t *testing.T) {
	page := pg3HNew(t, 4, 1)
	defer page.Deinit()
	a := pg3IPut(page, 0, 0, 'a')
	pg3ExpectPanic(t, func() {
		page.ClearGrapheme(a.cell)
	})
}

// Go-only: panic test: Page.clearGrapheme: map.getEntry(cell_offset).?, cell tagged as grapheme without a map entry
func TestPg3_panic_Page_clearGrapheme_no_map_entry(t *testing.T) {
	page := pg3HNew(t, 4, 1)
	defer page.Deinit()
	a := pg3IPut(page, 0, 0, 'a')
	a.cell.setContentTag(PageCellContentTagCodepointGrapheme)
	pg3ExpectPanic(t, func() {
		page.ClearGrapheme(a.cell)
	})
}

// Go-only: panic test: Page.moveHyperlink: assert(src.hyperlink), source cell without the hyperlink flag
func TestPg3_panic_Page_moveHyperlink_src_has_no_hyperlink(t *testing.T) {
	page := pg3HNew(t, 4, 1)
	defer page.Deinit()
	src := pg3IPut(page, 0, 0, 'a')
	dst := pg3IPut(page, 1, 0, 'b')
	pg3ExpectPanic(t, func() {
		page.moveHyperlink(src.cell, dst.cell)
	})
}

// Go-only: panic test: Page.moveHyperlink: assert(!dst.hyperlink), destination cell with the hyperlink flag
func TestPg3_panic_Page_moveHyperlink_dst_has_hyperlink(t *testing.T) {
	page := pg3HNew(t, 4, 1)
	defer page.Deinit()
	pg3HLinkCells(t, page, pg3HLink("https://a"), [][2]uint{{0, 0}, {1, 0}})
	src := page.GetRowAndCell(0, 0)
	dst := page.GetRowAndCell(1, 0)
	pg3ExpectPanic(t, func() {
		page.moveHyperlink(src.cell, dst.cell)
	})
}

// Go-only: panic test: Page.moveHyperlink: map.getEntry(src_offset).?, source cell with the hyperlink flag but no map entry
func TestPg3_panic_Page_moveHyperlink_no_map_entry(t *testing.T) {
	page := pg3HNew(t, 4, 1)
	defer page.Deinit()
	src := pg3IPut(page, 0, 0, 'a')
	src.cell.setHyperlink(true)
	dst := pg3IPut(page, 1, 0, 'b')
	pg3ExpectPanic(t, func() {
		page.moveHyperlink(src.cell, dst.cell)
	})
}

// Go-only: panic test: Page.setHyperlink: assert(row.hyperlink), same id set again on a cell whose row flag is false
func TestPg3_panic_Page_setHyperlink_row_hyperlink_flag_is_false(t *testing.T) {
	page := pg3HNew(t, 4, 1)
	defer page.Deinit()
	id := pg3ILink(t, page, 1)
	a := pg3IPut(page, 0, 0, 'a')
	zigTry(t, page.SetHyperlink(a.row, a.cell, id))
	a.row.hyperlink = false
	page.hyperlink_set.Use(id)
	pg3ExpectPanic(t, func() {
		_ = page.SetHyperlink(a.row, a.cell, id)
	})
}

// Go-only: panic test: Page.swapCells: map.getEntry(...).?, two cells tagged as grapheme without map entries
func TestPg3_panic_Page_swapCells_both_graphemes_no_map_entry(t *testing.T) {
	page := pg3HNew(t, 4, 1)
	defer page.Deinit()
	a := pg3IPut(page, 0, 0, 'a')
	a.cell.setContentTag(PageCellContentTagCodepointGrapheme)
	b := pg3IPut(page, 1, 0, 'b')
	b.cell.setContentTag(PageCellContentTagCodepointGrapheme)
	pg3ExpectPanic(t, func() {
		page.SwapCells(a.cell, b.cell)
	})
}

// Go-only: panic test: Page.swapCells: map.getEntry(...).?, two cells with the hyperlink flag but no map entries
func TestPg3_panic_Page_swapCells_both_hyperlinks_no_map_entry(t *testing.T) {
	page := pg3HNew(t, 4, 1)
	defer page.Deinit()
	a := pg3IPut(page, 0, 0, 'a')
	a.cell.setHyperlink(true)
	b := pg3IPut(page, 1, 0, 'b')
	b.cell.setHyperlink(true)
	pg3ExpectPanic(t, func() {
		page.SwapCells(a.cell, b.cell)
	})
}

// Go-only: panic test: Page.clearCells: row.cells.ptr(memory)[left..end], left greater than end (start index larger than end index)
func TestPg3_panic_Page_clearCells_left_greater_than_end(t *testing.T) {
	page := pg3HNew(t, 6, 1)
	defer page.Deinit()
	pg3HFillAll(page)
	row := page.GetRow(0)
	pg3ExpectPanic(t, func() {
		page.ClearCells(row, 3, 2)
	})
}

// Go-only: panic test: Page.lookupGrapheme: getOffset of a cell that is not in the page memory
func TestPg3_panic_Page_lookupGrapheme_cell_outside_the_page(t *testing.T) {
	page := pg3HNew(t, 4, 1)
	defer page.Deinit()
	var foreign PageCell
	pg3ExpectPanic(t, func() {
		page.LookupGrapheme(&foreign)
	})
}

// Go-only: panic test: Page.lookupHyperlink: getOffset of a cell that is not in the page memory
func TestPg3_panic_Page_lookupHyperlink_cell_outside_the_page(t *testing.T) {
	page := pg3HNew(t, 4, 1)
	defer page.Deinit()
	var foreign PageCell
	pg3ExpectPanic(t, func() {
		page.LookupHyperlink(&foreign)
	})
}

// Go-only: panic test: Capacity.adjust: @intCast(new_rows) to u16, a row count above 65535
func TestPg3_panic_Capacity_adjust_new_rows_does_not_fit_the_row_type(t *testing.T) {
	maxed := pg3MaxedCapacity()
	pg3ExpectPanic(t, func() {
		_, _ = maxed.Adjust(PageCapacityAdjustment{cols: Some(SizeCellCountInt(0))})
	})
}
