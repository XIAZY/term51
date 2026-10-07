// Tests translated one for one from src/terminal/page.zig (Ghostty v1.3.1
// plus the term51 additional tests), lines 5036 to 7214 of the patched file:
// from `term51: Page.exactRowCapacity: hyperlink strings are chunk rounded`
// to `term51: Page.swapCells: swapping a cell with itself changes nothing`.
// Each Go test carries the exact Zig title in a "// Zig: test" comment.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualSlices / expectEqualStrings keep Zig's names
//     and Zig's (expected, actual) argument order. Like `try`, they stop the
//     test at the first failure. `try f()` on a call that returns an error is
//     zigTry(t, err); `x.?` is zigUnwrap; `try testing.expectError(error.X,
//     call)` is pg2ExpectError.
//   - Zig's test helpers (term51H...) are the pg2 helpers below. A page is
//     handled by pointer (`&page` in Zig is `page` here), which is what
//     pg2New and pg2Init return.
//   - The `memory` / `base` argument of every call is gone (Go memory):
//     `page.styles.refCount(page.memory, id)` is `page.styles.RefCount(id)`,
//     `entry.uri.slice(page.memory)` is `entry.uri`, and `x.ptr(page.memory)`
//     is the Go slice `page.rows` / `page.cells`.
//   - Zig-only: `defer page.deinit()` is kept as `defer page.Deinit()`. A Zig
//     `defer` inside a loop body runs at the end of each iteration; Go's
//     `defer` runs when the test function returns. The allocator argument of
//     verifyIntegrity (`std.testing.allocator`) is dropped without a
//     per-test note.
//   - `build_options.kitty_graphics` is false in the library build (see the
//     header of page.go): pg2KittyGraphics. `build_options.slow_runtime_safety`
//     is pageSlowRuntimeSafety.
//   - Where Zig compares byte addresses of page memory, the Go test compares
//     the element addresses of `page.rows` / `page.cells`, or indexes with
//     Page.rowIndex / Page.cellIndex; each such place says so.
//   - Go-only panic tests (titled `// Go-only: panic test: ...`) are at the
//     end of the file.

package terminal

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// pg2KittyGraphics is `build_options.kitty_graphics`, false in the library
// build.
const pg2KittyGraphics = false

// pg2Item is `@sizeOf(term51HItem)`, i.e. @sizeOf(hyperlink.Set.Item), as a
// size.HyperlinkCountInt.
const pg2Item SizeHyperlinkCountInt = hyperlinkSetItemSizeOf

// pg2ExpectError is Zig's `try testing.expectError(expected, actual)`.
func pg2ExpectError(t *testing.T, want error, got error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

// pg2ExpectPanic runs f, which must panic. A panic with the wireframe's
// "not implemented" value does not count: it only means the function under
// test does not exist yet. The setup of the page is done outside f.
func pg2ExpectPanic(t *testing.T, f func()) {
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

// pg2Bits is term51HBits.
func pg2Bits(cell PageCell) uint64 {
	return cell.bitCast()
}

// pg2Init is `try Page.init(cap)`.
func pg2Init(t *testing.T, capacity PageCapacity) *Page {
	t.Helper()
	p, err := PageInit(capacity)
	zigTry(t, err)
	return &p
}

// pg2New is term51HNew: a page with room for 5 distinct styles and 5
// distinct hyperlinks.
func pg2New(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) *Page {
	t.Helper()
	capacity := PageCapacityInit(cols, rows)
	capacity.styles = 8
	capacity.hyperlink_bytes = 8 * pg2Item
	return pg2Init(t, capacity)
}

// pg2FillRow is term51HFillRow: write the codepoint `base + x` into every
// cell of row y.
func pg2FillRow(page *Page, y uint, base uint32) {
	for x := range uint(page.size.cols) {
		*page.GetRowAndCell(x, y).cell = PageCellInit(base + uint32(x))
	}
}

// pg2Grapheme is term51HGrapheme: write a cell with `cp` and the extra
// grapheme codepoints.
func pg2Grapheme(t *testing.T, page *Page, x uint, y uint, cp uint32, extra []uint32) {
	t.Helper()
	rac := page.GetRowAndCell(x, y)
	*rac.cell = PageCellInit(cp)
	for _, e := range extra {
		zigTry(t, page.AppendGrapheme(rac.row, rac.cell, e))
	}
}

// pg2PaletteStyle is `Style{ .fg_color = .{ .palette = n } }`.
func pg2PaletteStyle(n uint8) Style {
	return Style{FgColor: StyleColor{Tag: StyleColorPalette, Palette: n}}
}

// pg2StyleCells is term51HStyleCells: add a style (distinguished by palette
// fg n) and put it on the cells. After this the style's reference count is
// exactly len(cells).
func pg2StyleCells(t *testing.T, page *Page, n uint8, cells [][2]uint) StyleID {
	t.Helper()
	id, err := page.styles.Add(pg2PaletteStyle(n))
	zigTry(t, err)
	for _, c := range cells {
		rac := page.GetRowAndCell(c[0], c[1])
		rac.row.styled = true
		rac.cell.style_id = id
		page.styles.Use(id)
	}
	page.styles.Release(id)
	return id
}

// pg2LinkCells is term51HLinkCells: insert a hyperlink and put it on the
// cells. After this the link's reference count is exactly len(cells).
func pg2LinkCells(t *testing.T, page *Page, link Hyperlink, cells [][2]uint) HyperlinkID {
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

// pg2Link is term51HLink: `.{ .id = .{ .implicit = 0 }, .uri = uri }`.
func pg2Link(uri string) Hyperlink {
	return Hyperlink{
		ID:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: 0},
		URI: []uint8(uri),
	}
}

// pg2LinkImplicit is `.{ .id = .{ .implicit = id }, .uri = uri }`.
func pg2LinkImplicit(id SizeOffsetInt, uri string) Hyperlink {
	return Hyperlink{
		ID:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: id},
		URI: []uint8(uri),
	}
}

// pg2LinkExplicit is `.{ .id = .{ .explicit = id }, .uri = uri }`.
func pg2LinkExplicit(id string, uri string) Hyperlink {
	return Hyperlink{
		ID:  HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdExplicit, Explicit: []uint8(id)},
		URI: []uint8(uri),
	}
}

// pg2FillAll is term51HFillAll: fill every row of a page with distinct text.
func pg2FillAll(page *Page) {
	for y := range uint(page.size.rows) {
		pg2FillRow(page, y, uint32(0x100+y*0x20))
	}
}

// pg2Populate is term51HPopulate: populate a page with every kind of managed
// content.
func pg2Populate(t *testing.T, page *Page) {
	t.Helper()
	pg2FillAll(page)
	pg2Grapheme(t, page, 1, 0, 'e', []uint32{0x301, 0x302})
	pg2Grapheme(t, page, 4, 2, 'f', []uint32{0x303, 0x304, 0x305, 0x306, 0x307})
	pg2StyleCells(t, page, 7, [][2]uint{{2, 1}, {3, 1}})
	pg2StyleCells(t, page, 9, [][2]uint{{0, 3}})
	pg2LinkCells(t, page, pg2Link("https://one.example/"), [][2]uint{{5, 4}, {6, 4}})
	pg2LinkCells(t, page, pg2LinkExplicit("my-id", "https://two.example/"), [][2]uint{{7, 5}})
	page.GetRowAndCell(8, 1).cell.wide = PageCellWideWide
	page.GetRowAndCell(9, 1).cell.wide = PageCellWideSpacerTail
	page.GetRow(2).wrap = true
	page.GetRow(3).wrap_continuation = true
	page.GetRow(3).semantic_prompt = PageRowSemanticPromptPrompt
}

// pg2ExpectRowsEqualModManaged is term51HExpectRowsEqualModManaged: rows
// equal in text and wide state (ids may be remapped across pages).
func pg2ExpectRowsEqualModManaged(t *testing.T, a *Page, b *Page, ya uint, yb uint) {
	t.Helper()
	ra := a.GetRow(ya)
	rb := b.GetRow(yb)
	for x := range uint(a.size.cols) {
		ca := a.GetCells(ra)[x]
		cb := b.GetCells(rb)[x]
		expectEqual(t, ca.Codepoint(), cb.Codepoint())
		expectEqual(t, ca.wide, cb.wide)
		expectEqual(t, ca.HasGrapheme(), cb.HasGrapheme())
		expectEqual(t, ca.hyperlink, cb.hyperlink)
		expectEqual(t, ca.style_id != 0, cb.style_id != 0)
	}
}

// pg2ExpectCps is term51HExpectCps.
func pg2ExpectCps(t *testing.T, page *Page, y uint, expected []uint32) {
	t.Helper()
	expectEqual(t, uint(page.size.cols), uint(len(expected)))
	for i, c := range page.GetCells(page.GetRow(y)) {
		expectEqual(t, expected[i], c.Codepoint())
	}
}

// pg2MarkRow is term51HMarkRow.
func pg2MarkRow(page *Page, y uint, cp uint32) {
	cells := page.GetCells(page.GetRow(y))
	for i := range cells {
		cells[i] = PageCellInit(cp)
	}
}

// pg2Repeat is `[_]u21{cp} ** n`.
func pg2Repeat(cp uint32, n int) []uint32 {
	out := make([]uint32, n)
	for i := range out {
		out[i] = cp
	}
	return out
}

// pg2FillLinkMap is term51HFillLinkMap.
func pg2FillLinkMap(t *testing.T, page *Page, uri string) HyperlinkID {
	t.Helper()
	id, err := page.InsertHyperlink(pg2Link(uri))
	zigTry(t, err)
	n := uint(0)
	for page.HyperlinkCount() < page.HyperlinkCapacity() {
		rac := page.GetRowAndCell(n%uint(page.size.cols), n/uint(page.size.cols))
		page.hyperlink_set.Use(id)
		zigTry(t, page.SetHyperlink(rac.row, rac.cell, id))
		n++
	}
	page.hyperlink_set.Release(id)
	return id
}

// pg2RowFirstCell stands for `row.cells.ptr(page.memory)`: the first cell of
// the row. Row.cells holds the byte offset upstream would have (see page.go),
// so the index in `page.cells` is (cells - layout.cells_start) / 8.
func pg2RowFirstCell(page *Page, row *PageRow) *PageCell {
	l := PageLayoutOf(page.capacity)
	return &page.cells[(uint64(row.cells)-l.cells_start)/8]
}

// ---- Page.exactRowCapacity ------------------------------------------------

// Zig: test "term51: Page.exactRowCapacity: hyperlink strings are chunk rounded"
func TestPg2_term51_Page_exactRowCapacity_hyperlink_strings_are_chunk_rounded(t *testing.T) {
	page := pg2New(t, 10, 10)
	defer page.Deinit()
	const item = pg2Item

	// URI lengths on both sides of the 32 byte chunk.
	uris := []struct {
		len   int
		bytes int
	}{
		{len: 1, bytes: 32},
		{len: 32, bytes: 32},
		{len: 33, bytes: 64},
		{len: 64, bytes: 64},
		{len: 65, bytes: 96},
	}
	for _, u := range uris {
		page.Reinit()
		// Zig-only: const uri = try std.testing.allocator.alloc(u8, u.len); defer std.testing.allocator.free(uri);
		// Zig-only: @memset(uri, 'u');
		uri := strings.Repeat("u", u.len)
		pg2LinkCells(t, page, pg2Link(uri), [][2]uint{{0, 0}})
		capacity := page.ExactRowCapacity(0, 1)
		expectEqual(t, SizeStringBytesInt(u.bytes), capacity.string_bytes)
		expectEqual(t, SizeHyperlinkCountInt(RefCountedSetCapacityForCount(1))*item, capacity.hyperlink_bytes)
	}
}

// Zig: test "term51: Page.exactRowCapacity: explicit ids add string bytes, implicit ids do not"
func TestPg2_term51_Page_exactRowCapacity_explicit_ids_add_string_bytes_implicit_ids_do_not(t *testing.T) {
	page := pg2New(t, 10, 4)
	defer page.Deinit()
	const item = pg2Item

	pg2LinkCells(t, page, pg2LinkImplicit(5, "https://a"), [][2]uint{{0, 0}})
	capacity := page.ExactRowCapacity(0, 1)
	expectEqual(t, SizeStringBytesInt(32), capacity.string_bytes)
	expectEqual(t, 3*item, capacity.hyperlink_bytes)

	pg2LinkCells(t, page, pg2LinkExplicit("abc", "https://b"), [][2]uint{{1, 0}})
	capacity = page.ExactRowCapacity(0, 1)
	// 32 (a) + 32 (b uri) + 32 (b id)
	expectEqual(t, SizeStringBytesInt(96), capacity.string_bytes)
	// two links need capacityForCount(2) = 4 set items
	expectEqual(t, SizeHyperlinkCountInt(RefCountedSetCapacityForCount(2))*item, capacity.hyperlink_bytes)

	// An explicit id longer than a chunk.
	page.Reinit()
	long_id := strings.Repeat("i", 40)
	pg2LinkCells(t, page, pg2LinkExplicit(long_id, "https://c"), [][2]uint{{0, 0}})
	capacity = page.ExactRowCapacity(0, 1)
	expectEqual(t, SizeStringBytesInt(32+64), capacity.string_bytes)
}

// Zig: test "term51: Page.exactRowCapacity: a link used by many cells counts once for strings"
func TestPg2_term51_Page_exactRowCapacity_a_link_used_by_many_cells_counts_once_for_strings(t *testing.T) {
	page := pg2New(t, 10, 10)
	defer page.Deinit()
	const item = pg2Item
	var cells [10][2]uint
	for x := range uint(10) {
		cells[x] = [2]uint{x, 0}
	}
	pg2LinkCells(t, page, pg2Link("https://a"), cells[:])
	capacity := page.ExactRowCapacity(0, 1)
	expectEqual(t, SizeStringBytesInt(32), capacity.string_bytes)
	expectEqual(t, 3*item, capacity.hyperlink_bytes)
}

// Zig: test "term51: Page.exactRowCapacity: hyperlink map minimum is cells / 16 rounded up"
func TestPg2_term51_Page_exactRowCapacity_hyperlink_map_minimum_is_cells_16_rounded_up(t *testing.T) {
	page := pg2New(t, 10, 10)
	defer page.Deinit()
	const item = pg2Item

	// One link (set capacity 3) on 48 cells: ceil(48/16) = 3, tie.
	var cells [100][2]uint
	for i := range uint(100) {
		cells[i] = [2]uint{i % 10, i / 10}
	}
	pg2LinkCells(t, page, pg2Link("https://a"), cells[0:48])
	expectEqual(t, 3*item, page.ExactRowCapacity(0, 10).hyperlink_bytes)

	// 49 cells: ceil(49/16) = 4 beats the set capacity of 3.
	page.Reinit()
	pg2LinkCells(t, page, pg2Link("https://a"), cells[0:49])
	expectEqual(t, 4*item, page.ExactRowCapacity(0, 10).hyperlink_bytes)

	// 100 cells: ceil(100/16) = 7.
	page.Reinit()
	pg2LinkCells(t, page, pg2Link("https://a"), cells[0:100])
	capacity := page.ExactRowCapacity(0, 10)
	expectEqual(t, 7*item, capacity.hyperlink_bytes)
	expectEqual(t, SizeStringBytesInt(32), capacity.string_bytes)
}

// Zig: test "term51: Page.exactRowCapacity: hyperlink cell flag without map data"
func TestPg2_term51_Page_exactRowCapacity_hyperlink_cell_flag_without_map_data(t *testing.T) {
	page := pg2New(t, 10, 2)
	defer page.Deinit()
	const item = pg2Item

	// 17 cells claim to be hyperlinks but have no map entry: no set items,
	// no strings, map minimum ceil(17/16) = 2.
	for x := range uint(10) {
		page.GetRowAndCell(x, 0).cell.hyperlink = true
	}
	for x := range uint(7) {
		page.GetRowAndCell(x, 1).cell.hyperlink = true
	}
	capacity := page.ExactRowCapacity(0, 2)
	expectEqual(t, 2*item, capacity.hyperlink_bytes)
	expectEqual(t, SizeStringBytesInt(0), capacity.string_bytes)

	// Exactly 16 cells -> 1.
	page.GetRowAndCell(6, 1).cell.hyperlink = false
	capacity = page.ExactRowCapacity(0, 2)
	expectEqual(t, 1*item, capacity.hyperlink_bytes)
}

// Zig: test "term51: Page.exactRowCapacity: distinct links add up"
func TestPg2_term51_Page_exactRowCapacity_distinct_links_add_up(t *testing.T) {
	page := pg2New(t, 10, 4)
	defer page.Deinit()
	const item = pg2Item
	pg2LinkCells(t, page, pg2Link("https://a"), [][2]uint{{0, 0}, {1, 0}})
	pg2LinkCells(t, page, pg2Link("https://b"), [][2]uint{{2, 1}})
	pg2LinkCells(t, page, pg2Link("https://c"), [][2]uint{{3, 2}})

	// Row 0 only: 1 link.
	capacity := page.ExactRowCapacity(0, 1)
	expectEqual(t, SizeStringBytesInt(32), capacity.string_bytes)
	expectEqual(t, 3*item, capacity.hyperlink_bytes)
	// Rows 0-1: 2 links -> set capacity 4.
	capacity = page.ExactRowCapacity(0, 2)
	expectEqual(t, SizeStringBytesInt(64), capacity.string_bytes)
	expectEqual(t, 4*item, capacity.hyperlink_bytes)
	// Rows 0-3: 3 links -> set capacity ceil(4/0.8125) = 5.
	capacity = page.ExactRowCapacity(0, 4)
	expectEqual(t, SizeStringBytesInt(96), capacity.string_bytes)
	expectEqual(t, 5*item, capacity.hyperlink_bytes)
	// Rows 2-3 only: 1 link.
	capacity = page.ExactRowCapacity(2, 4)
	expectEqual(t, SizeStringBytesInt(32), capacity.string_bytes)
}

// Zig: test "term51: Page.exactRowCapacity: style and hyperlink ids do not interfere"
func TestPg2_term51_Page_exactRowCapacity_style_and_hyperlink_ids_do_not_interfere(t *testing.T) {
	page := pg2New(t, 6, 2)
	defer page.Deinit()
	const item = pg2Item
	// Both get id 1 in their own sets.
	sid := pg2StyleCells(t, page, 1, [][2]uint{{0, 0}})
	hid := pg2LinkCells(t, page, pg2Link("https://a"), [][2]uint{{1, 0}})
	expectEqual(t, sid, hid)
	capacity := page.ExactRowCapacity(0, 1)
	expectEqual(t, SizeStyleCountInt(3), capacity.styles)
	expectEqual(t, 3*item, capacity.hyperlink_bytes)
	expectEqual(t, SizeStringBytesInt(32), capacity.string_bytes)
}

// Zig: test "term51: Page.exactRowCapacity: the capacity is exactly enough to clone the rows"
func TestPg2_term51_Page_exactRowCapacity_the_capacity_is_exactly_enough_to_clone_the_rows(t *testing.T) {
	page := pg2New(t, 10, 6)
	defer page.Deinit()
	pg2Populate(t, page)

	ranges := [][2]uint{{0, 6}, {0, 1}, {1, 4}, {3, 6}, {5, 6}}
	for _, r := range ranges {
		capacity := page.ExactRowCapacity(r[0], r[1])
		dst := pg2Init(t, capacity)
		defer dst.Deinit()
		zigTry(t, dst.CloneFrom(page, r[0], r[1]))
		zigTry(t, dst.VerifyIntegrity())
		for y := r[0]; y < r[1]; y++ {
			pg2ExpectRowsEqualModManaged(t, page, dst, y, y-r[0])
		}
		// Everything that was counted is in use in the clone.
		var graphemes uint
		var links uint
		for y := r[0]; y < r[1]; y++ {
			for _, c := range page.GetCells(page.GetRow(y)) {
				if c.HasGrapheme() {
					graphemes += 1
				}
				if c.hyperlink {
					links += 1
				}
			}
		}
		expectEqual(t, graphemes, dst.GraphemeCount())
		expectEqual(t, links, dst.HyperlinkCount())
	}
}

// ---- Page.cloneFrom -------------------------------------------------------

// Zig: test "term51: Page.cloneFrom: full range copies cells, row metadata and managed data"
func TestPg2_term51_Page_cloneFrom_full_range_copies_cells_row_metadata_and_managed_data(t *testing.T) {
	src := pg2New(t, 10, 6)
	defer src.Deinit()
	pg2Populate(t, src)
	dst := pg2New(t, 10, 6)
	defer dst.Deinit()

	zigTry(t, dst.CloneFrom(src, 0, 6))

	for y := range uint(6) {
		pg2ExpectRowsEqualModManaged(t, src, dst, y, y)
		s := *src.GetRow(y)
		d := *dst.GetRow(y)
		expectEqual(t, s.wrap, d.wrap)
		expectEqual(t, s.wrap_continuation, d.wrap_continuation)
		expectEqual(t, s.semantic_prompt, d.semantic_prompt)
		expectEqual(t, s.grapheme, d.grapheme)
		expectEqual(t, s.styled, d.styled)
		expectEqual(t, s.hyperlink, d.hyperlink)
	}
	expectEqual(t, src.GraphemeCount(), dst.GraphemeCount())
	expectEqual(t, src.HyperlinkCount(), dst.HyperlinkCount())
	expectEqual(t, src.styles.Count(), dst.styles.Count())
	expectEqual(t, src.hyperlink_set.Count(), dst.hyperlink_set.Count())
	zigTry(t, dst.VerifyIntegrity())
	zigTry(t, src.VerifyIntegrity())
}

// Zig: test "term51: Page.cloneFrom: row range shifts rows and leaves the rest untouched"
func TestPg2_term51_Page_cloneFrom_row_range_shifts_rows_and_leaves_the_rest_untouched(t *testing.T) {
	src := pg2New(t, 10, 6)
	defer src.Deinit()
	pg2FillAll(src)
	dst := pg2New(t, 10, 6)
	defer dst.Deinit()
	for y := range uint(6) {
		pg2MarkRow(dst, y, 0x900)
	}

	zigTry(t, dst.CloneFrom(src, 2, 5))

	for y := range uint(3) {
		pg2ExpectRowsEqualModManaged(t, src, dst, y+2, y)
	}
	for y := uint(3); y < 6; y++ {
		for _, c := range dst.GetCells(dst.GetRow(y)) {
			expectEqual(t, uint32(0x900), c.Codepoint())
		}
	}
}

// Zig: test "term51: Page.cloneFrom: an empty range copies nothing"
func TestPg2_term51_Page_cloneFrom_an_empty_range_copies_nothing(t *testing.T) {
	src := pg2New(t, 10, 4)
	defer src.Deinit()
	pg2FillAll(src)
	dst := pg2New(t, 10, 4)
	defer dst.Deinit()
	for y := range uint(4) {
		pg2MarkRow(dst, y, 0x900)
	}

	zigTry(t, dst.CloneFrom(src, 3, 3))
	zigTry(t, dst.CloneFrom(src, 0, 0))
	zigTry(t, dst.CloneFrom(src, 4, 4))

	for y := range uint(4) {
		for _, c := range dst.GetCells(dst.GetRow(y)) {
			expectEqual(t, uint32(0x900), c.Codepoint())
		}
	}
}

// Zig: test "term51: Page.cloneFrom: a taller destination keeps its extra rows"
func TestPg2_term51_Page_cloneFrom_a_taller_destination_keeps_its_extra_rows(t *testing.T) {
	src := pg2New(t, 10, 4)
	defer src.Deinit()
	pg2FillAll(src)
	dst := pg2New(t, 10, 8)
	defer dst.Deinit()
	for y := range uint(8) {
		pg2MarkRow(dst, y, 0x900)
	}

	zigTry(t, dst.CloneFrom(src, 0, 4))

	for y := range uint(4) {
		pg2ExpectRowsEqualModManaged(t, src, dst, y, y)
	}
	for y := uint(4); y < 8; y++ {
		for _, c := range dst.GetCells(dst.GetRow(y)) {
			expectEqual(t, uint32(0x900), c.Codepoint())
		}
	}
}

// Zig: test "term51: Page.cloneFrom: releases managed memory of overwritten destination rows"
func TestPg2_term51_Page_cloneFrom_releases_managed_memory_of_overwritten_destination_rows(t *testing.T) {
	src := pg2New(t, 10, 3)
	defer src.Deinit()
	pg2FillAll(src)
	dst := pg2New(t, 10, 3)
	defer dst.Deinit()
	pg2FillAll(dst)
	sid := pg2StyleCells(t, dst, 1, [][2]uint{{0, 0}, {1, 0}})
	hid := pg2LinkCells(t, dst, pg2Link("https://a"), [][2]uint{{2, 1}})
	pg2Grapheme(t, dst, 3, 2, 'e', []uint32{0x301})

	zigTry(t, dst.CloneFrom(src, 0, 3))

	expectEqual(t, uint16(0), dst.styles.RefCount(sid))
	expectEqual(t, uint16(0), dst.hyperlink_set.RefCount(hid))
	expectEqual(t, uint(0), dst.styles.Count())
	expectEqual(t, uint(0), dst.hyperlink_set.Count())
	expectEqual(t, uint(0), dst.HyperlinkCount())
	expectEqual(t, uint(0), dst.GraphemeCount())
	expectEqual(t, uint64(0), dst.grapheme_alloc.UsedBytes())
	for y := range uint(3) {
		row := dst.GetRow(y)
		expect(t, !row.grapheme && !row.styled && !row.hyperlink, "!row.grapheme and !row.styled and !row.hyperlink")
	}
}

// Zig: test "term51: Page.cloneFrom: fewer destination columns truncate and take the whole row metadata"
func TestPg2_term51_Page_cloneFrom_fewer_destination_columns_truncate_and_take_the_whole_row_metadata(t *testing.T) {
	src := pg2New(t, 10, 3)
	defer src.Deinit()
	pg2FillAll(src)
	src.GetRow(0).wrap = true
	src.GetRow(0).semantic_prompt = PageRowSemanticPromptPrompt
	src.GetRow(1).wrap_continuation = true
	src.GetRow(0).dirty = true
	// Wide cell ends up at the last destination column, its tail is cut.
	src.GetRowAndCell(4, 2).cell.wide = PageCellWideWide
	src.GetRowAndCell(5, 2).cell.wide = PageCellWideSpacerTail
	// A spacer head in the last source column is cut.
	src.GetRowAndCell(9, 0).cell.wide = PageCellWideSpacerHead
	dst := pg2New(t, 5, 3)
	defer dst.Deinit()

	zigTry(t, dst.CloneFrom(src, 0, 3))

	expectEqual(t, SizeCellCountInt(5), dst.size.cols)
	for y := range uint(3) {
		cells := dst.GetCells(dst.GetRow(y))
		expectEqual(t, uint(5), uint(len(cells)))
		for x, c := range cells {
			expectEqual(t, src.GetRowAndCell(uint(x), y).cell.Codepoint(), c.Codepoint())
		}
	}
	expect(t, dst.GetRow(0).wrap, "dst.getRow(0).wrap")
	expectEqual(t, PageRowSemanticPromptPrompt, dst.GetRow(0).semantic_prompt)
	expect(t, dst.GetRow(0).dirty, "dst.getRow(0).dirty")
	expect(t, dst.GetRow(1).wrap_continuation, "dst.getRow(1).wrap_continuation")
	expect(t, !dst.GetRow(1).wrap, "!dst.getRow(1).wrap")
	expectEqual(t, PageCellWideWide, dst.GetRowAndCell(4, 2).cell.wide)
	zigTry(t, dst.VerifyIntegrity())
}

// Zig: test "term51: Page.cloneFrom: managed data beyond the destination width is dropped, flags stay"
func TestPg2_term51_Page_cloneFrom_managed_data_beyond_the_destination_width_is_dropped_flags_stay(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2FillAll(src)
	pg2Grapheme(t, src, 7, 0, 'e', []uint32{0x301})
	pg2StyleCells(t, src, 1, [][2]uint{{8, 0}})
	pg2LinkCells(t, src, pg2Link("https://a"), [][2]uint{{9, 0}})
	dst := pg2New(t, 5, 1)
	defer dst.Deinit()

	zigTry(t, dst.CloneFrom(src, 0, 1))

	expectEqual(t, uint(0), dst.GraphemeCount())
	expectEqual(t, uint(0), dst.HyperlinkCount())
	expectEqual(t, uint(0), dst.styles.Count())
	// The row flags came from the source row and are now false positives.
	row := dst.GetRow(0)
	expect(t, row.grapheme, "row.grapheme")
	expect(t, row.styled, "row.styled")
	expect(t, row.hyperlink, "row.hyperlink")
	zigTry(t, dst.VerifyIntegrity())
}

// Zig: test "term51: Page.cloneFrom: more destination columns leave the extra cells and flags alone"
func TestPg2_term51_Page_cloneFrom_more_destination_columns_leave_the_extra_cells_and_flags_alone(t *testing.T) {
	src := pg2New(t, 5, 2)
	defer src.Deinit()
	pg2FillAll(src)
	src.GetRow(0).wrap = true
	src.GetRow(0).dirty = true
	src.GetRow(0).semantic_prompt = PageRowSemanticPromptPromptContinuation
	dst := pg2New(t, 10, 2)
	defer dst.Deinit()
	for y := range uint(2) {
		pg2MarkRow(dst, y, 0x900)
	}
	pg2Grapheme(t, dst, 7, 0, 'g', []uint32{0x301})
	dst.GetRow(1).wrap_continuation = true
	dst.GetRow(1).dirty = true
	dst.GetRow(1).semantic_prompt = PageRowSemanticPromptPrompt

	zigTry(t, dst.CloneFrom(src, 0, 2))

	// Copied part.
	for x := range uint(5) {
		expectEqual(t, src.GetRowAndCell(x, 0).cell.Codepoint(), dst.GetRowAndCell(x, 0).cell.Codepoint())
		expectEqual(t, src.GetRowAndCell(x, 1).cell.Codepoint(), dst.GetRowAndCell(x, 1).cell.Codepoint())
	}
	// Extra columns are NOT zeroed (despite upstream's doc comment).
	for x := uint(5); x < 10; x++ {
		expect(t, dst.GetRowAndCell(x, 1).cell.Codepoint() == 0x900, "dst.getRowAndCell(x, 1).cell.codepoint() == 0x900")
	}
	expectEqual(t, uint32('g'), dst.GetRowAndCell(7, 0).cell.Codepoint())
	expect(t, dst.GetRowAndCell(7, 0).cell.HasGrapheme(), "dst.getRowAndCell(7, 0).cell.hasGrapheme()")
	expectEqual(t, uint(1), dst.GraphemeCount())
	// Partial-row rules: dst flags kept, wrap not taken from the source.
	expect(t, dst.GetRow(0).grapheme, "dst.getRow(0).grapheme")
	expect(t, !dst.GetRow(0).wrap, "!dst.getRow(0).wrap")
	expect(t, dst.GetRow(0).dirty, "dst.getRow(0).dirty") // OR of both
	expectEqual(t, PageRowSemanticPromptPromptContinuation, dst.GetRow(0).semantic_prompt)
	expect(t, dst.GetRow(1).wrap_continuation, "dst.getRow(1).wrap_continuation")
	expect(t, dst.GetRow(1).dirty, "dst.getRow(1).dirty")
	// Semantic prompt always comes from the source.
	expectEqual(t, PageRowSemanticPromptNone, dst.GetRow(1).semantic_prompt)
	zigTry(t, dst.VerifyIntegrity())
}

// Zig: test "term51: Page.cloneFrom: a source spacer head is cleared when columns grow"
func TestPg2_term51_Page_cloneFrom_a_source_spacer_head_is_cleared_when_columns_grow(t *testing.T) {
	src := pg2New(t, 5, 2)
	defer src.Deinit()
	pg2FillAll(src)
	src.GetRow(0).wrap = true
	src.GetRowAndCell(4, 0).cell.wide = PageCellWideSpacerHead
	dst := pg2New(t, 10, 2)
	defer dst.Deinit()

	zigTry(t, dst.CloneFrom(src, 0, 2))

	expectEqual(t, PageCellWideNarrow, dst.GetRowAndCell(4, 0).cell.wide)
	// The text of the cell is kept; only its wide state changes.
	expectEqual(t, src.GetRowAndCell(4, 0).cell.Codepoint(), dst.GetRowAndCell(4, 0).cell.Codepoint())
	// The partial-row rule keeps the destination's wrap (false).
	expect(t, !dst.GetRow(0).wrap, "!dst.getRow(0).wrap")
	zigTry(t, dst.VerifyIntegrity())
}

// Zig: test "term51: Page.cloneFrom: spacer head is kept when the width is equal"
func TestPg2_term51_Page_cloneFrom_spacer_head_is_kept_when_the_width_is_equal(t *testing.T) {
	src := pg2New(t, 5, 2)
	defer src.Deinit()
	src.GetRow(0).wrap = true
	src.GetRowAndCell(4, 0).cell.wide = PageCellWideSpacerHead
	dst := pg2New(t, 5, 2)
	defer dst.Deinit()
	zigTry(t, dst.CloneFrom(src, 0, 2))
	expectEqual(t, PageCellWideSpacerHead, dst.GetRowAndCell(4, 0).cell.wide)
	expect(t, dst.GetRow(0).wrap, "dst.getRow(0).wrap")
}

// Zig: test "term51: Page.cloneFrom: only a source spacer head in the last column is cleared when growing"
func TestPg2_term51_Page_cloneFrom_only_a_source_spacer_head_in_the_last_column_is_cleared_when_growing(t *testing.T) {
	src := pg2New(t, 5, 2)
	defer src.Deinit()
	// wide at the last source column stays wide, narrow stays narrow
	src.GetRowAndCell(4, 0).cell.wide = PageCellWideWide
	src.GetRowAndCell(3, 1).cell.wide = PageCellWideWide
	src.GetRowAndCell(4, 1).cell.wide = PageCellWideSpacerTail
	dst := pg2New(t, 8, 2)
	defer dst.Deinit()
	zigTry(t, dst.CloneFrom(src, 0, 2))
	expectEqual(t, PageCellWideWide, dst.GetRowAndCell(4, 0).cell.wide)
	expectEqual(t, PageCellWideWide, dst.GetRowAndCell(3, 1).cell.wide)
	expectEqual(t, PageCellWideSpacerTail, dst.GetRowAndCell(4, 1).cell.wide)
	zigTry(t, dst.VerifyIntegrity())
}

// Zig: test "term51: Page.cloneFrom: the spacer head fix also applies to rows with managed memory"
func TestPg2_term51_Page_cloneFrom_the_spacer_head_fix_also_applies_to_rows_with_managed_memory(t *testing.T) {
	src := pg2New(t, 5, 1)
	defer src.Deinit()
	pg2FillAll(src)
	pg2Grapheme(t, src, 0, 0, 'e', []uint32{0x301})
	src.GetRow(0).wrap = true
	src.GetRowAndCell(4, 0).cell.wide = PageCellWideSpacerHead
	dst := pg2New(t, 9, 1)
	defer dst.Deinit()
	zigTry(t, dst.CloneFrom(src, 0, 1))
	expectEqual(t, PageCellWideNarrow, dst.GetRowAndCell(4, 0).cell.wide)
	expectEqual(t, uint(1), dst.GraphemeCount())
}

// Zig: test "term51: Page.cloneFrom: within one page copies rows with managed data"
func TestPg2_term51_Page_cloneFrom_within_one_page_copies_rows_with_managed_data(t *testing.T) {
	page := pg2New(t, 10, 6)
	defer page.Deinit()
	pg2FillAll(page)
	pg2Grapheme(t, page, 1, 4, 'e', []uint32{0x301, 0x302})
	sid := pg2StyleCells(t, page, 3, [][2]uint{{2, 4}})
	hid := pg2LinkCells(t, page, pg2Link("https://a"), [][2]uint{{3, 4}})

	zigTry(t, page.CloneFrom(page, 4, 5))

	// Row 4 is now also row 0.
	pg2ExpectRowsEqualModManaged(t, page, page, 4, 0)
	expectEqual(t, sid, page.GetRowAndCell(2, 0).cell.style_id)
	expectEqual(t, hid, zigUnwrap(t, page.LookupHyperlink(page.GetRowAndCell(3, 0).cell)))
	expectEqual(t, uint16(2), page.styles.RefCount(sid))
	expectEqual(t, uint16(2), page.hyperlink_set.RefCount(hid))
	expectEqual(t, uint(2), page.GraphemeCount())
	expectEqual(t, uint(2), page.HyperlinkCount())
	a := zigUnwrap(t, page.LookupGrapheme(page.GetRowAndCell(1, 0).cell))
	b := zigUnwrap(t, page.LookupGrapheme(page.GetRowAndCell(1, 4).cell))
	expectEqualSlices(t, b, a)
	expect(t, &a[0] != &b[0], "a.ptr != b.ptr")
	expect(t, page.GetRow(0).grapheme && page.GetRow(0).styled && page.GetRow(0).hyperlink, "page.getRow(0).grapheme and page.getRow(0).styled and page.getRow(0).hyperlink")
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.cloneFrom: an error stops the copy but keeps earlier rows"
func TestPg2_term51_Page_cloneFrom_an_error_stops_the_copy_but_keeps_earlier_rows(t *testing.T) {
	src := pg2New(t, 10, 3)
	defer src.Deinit()
	pg2FillAll(src)
	pg2StyleCells(t, src, 1, [][2]uint{{0, 0}})
	pg2StyleCells(t, src, 2, [][2]uint{{0, 1}})
	pg2StyleCells(t, src, 3, [][2]uint{{0, 2}})
	// Room for two styles only.
	capacity := PageCapacityInit(10, 3)
	capacity.styles = 4
	dst := pg2Init(t, capacity)
	defer dst.Deinit()

	pg2ExpectError(t, ErrStyleSetOutOfMemory, dst.CloneFrom(src, 0, 3))

	expectEqual(t, StyleID(1), dst.GetRowAndCell(0, 0).cell.style_id)
	expectEqual(t, StyleID(2), dst.GetRowAndCell(0, 1).cell.style_id)
	// The failing cell keeps the text but not the style.
	expectEqual(t, src.GetRowAndCell(0, 2).cell.Codepoint(), dst.GetRowAndCell(0, 2).cell.Codepoint())
	expectEqual(t, StyleID(0), dst.GetRowAndCell(0, 2).cell.style_id)
	expect(t, dst.GetRow(2).styled, "dst.getRow(2).styled")
	expectEqual(t, uint(2), dst.styles.Count())
	zigTry(t, dst.VerifyIntegrity())
}

// ---- Page.cloneRowFrom ----------------------------------------------------

// Zig: test "term51: Page.cloneRowFrom: copies the full row between different row indices"
func TestPg2_term51_Page_cloneRowFrom_copies_the_full_row_between_different_row_indices(t *testing.T) {
	src := pg2New(t, 10, 3)
	defer src.Deinit()
	pg2FillAll(src)
	src.GetRow(2).wrap = true
	src.GetRow(2).semantic_prompt = PageRowSemanticPromptPrompt
	pg2Grapheme(t, src, 4, 2, 'e', []uint32{0x301})
	dst := pg2New(t, 10, 3)
	defer dst.Deinit()
	pg2MarkRow(dst, 0, 0x900)

	zigTry(t, dst.CloneRowFrom(src, dst.GetRow(0), src.GetRow(2)))

	pg2ExpectRowsEqualModManaged(t, src, dst, 2, 0)
	expect(t, dst.GetRow(0).wrap, "dst.getRow(0).wrap")
	expect(t, dst.GetRow(0).grapheme, "dst.getRow(0).grapheme")
	expectEqual(t, PageRowSemanticPromptPrompt, dst.GetRow(0).semantic_prompt)
	expectEqual(t, uint(1), dst.GraphemeCount())
	// Other destination rows untouched.
	expectEqual(t, uint32(0), dst.GetRowAndCell(0, 1).cell.Codepoint())
}

// Zig: test "term51: Page.cloneRowFrom: narrower destination copies only its columns"
func TestPg2_term51_Page_cloneRowFrom_narrower_destination_copies_only_its_columns(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2FillAll(src)
	dst := pg2New(t, 4, 1)
	defer dst.Deinit()
	zigTry(t, dst.CloneRowFrom(src, dst.GetRow(0), src.GetRow(0)))
	pg2ExpectCps(t, dst, 0, []uint32{0x100, 0x101, 0x102, 0x103})
}

// Zig: test "term51: Page.cloneRowFrom: wider destination copies the source columns only"
func TestPg2_term51_Page_cloneRowFrom_wider_destination_copies_the_source_columns_only(t *testing.T) {
	src := pg2New(t, 4, 1)
	defer src.Deinit()
	pg2FillAll(src)
	dst := pg2New(t, 7, 1)
	defer dst.Deinit()
	pg2MarkRow(dst, 0, 0x900)
	zigTry(t, dst.CloneRowFrom(src, dst.GetRow(0), src.GetRow(0)))
	pg2ExpectCps(t, dst, 0, []uint32{0x100, 0x101, 0x102, 0x103, 0x900, 0x900, 0x900})
}

// Zig: test "term51: Page.cloneRowFrom: is clonePartialRowFrom over the destination width"
func TestPg2_term51_Page_cloneRowFrom_is_clonePartialRowFrom_over_the_destination_width(t *testing.T) {
	src := pg2New(t, 10, 6)
	defer src.Deinit()
	pg2Populate(t, src)
	a := pg2New(t, 10, 6)
	defer a.Deinit()
	b := pg2New(t, 10, 6)
	defer b.Deinit()
	for _, y := range []uint{0, 1} {
		zigTry(t, a.CloneRowFrom(src, a.GetRow(y), src.GetRow(y)))
		zigTry(t, b.ClonePartialRowFrom(src, b.GetRow(y), src.GetRow(y), 0, 10))
	}
	// Zig: expectEqualSlices(u8, a.memory, b.memory). Page.memory does not
	// exist in Go; the two whole pages (rows, cells and every table) are
	// compared instead.
	expect(t, reflect.DeepEqual(a, b), "a.memory == b.memory (the two pages are deeply equal)")
}

// Zig: test "term51: Page.cloneRowFrom: errors are the CloneFromError set"
func TestPg2_term51_Page_cloneRowFrom_errors_are_the_CloneFromError_set(t *testing.T) {
	src := pg2New(t, 4, 1)
	defer src.Deinit()
	pg2StyleCells(t, src, 1, [][2]uint{{0, 0}, {1, 0}, {2, 0}})
	// A set with room for two styles, both in use.
	capacity := PageCapacityInit(4, 2)
	capacity.styles = 4
	dst := pg2Init(t, capacity)
	defer dst.Deinit()
	pg2StyleCells(t, dst, 21, [][2]uint{{0, 1}})
	pg2StyleCells(t, dst, 22, [][2]uint{{1, 1}})
	pg2ExpectError(t, ErrStyleSetOutOfMemory, dst.CloneRowFrom(src, dst.GetRow(0), src.GetRow(0)))
}

// ---- Page.clonePartialRowFrom ---------------------------------------------

// Zig: test "term51: Page.clonePartialRowFrom: plain cells are copied bit for bit in the range only"
func TestPg2_term51_Page_clonePartialRowFrom_plain_cells_are_copied_bit_for_bit_in_the_range_only(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2FillRow(src, 0, 0x100)
	src.GetRowAndCell(2, 0).cell.protected = true
	src.GetRowAndCell(3, 0).cell.semantic_content = PageCellSemanticContentPrompt
	src.GetRowAndCell(4, 0).cell.semantic_content = PageCellSemanticContentInput
	*src.GetRowAndCell(5, 0).cell = PageCell{content_tag: PageCellContentTagBgColorPalette, content: PageCellContent{color_palette: 77}}
	*src.GetRowAndCell(6, 0).cell = PageCell{content_tag: PageCellContentTagBgColorRgb, content: PageCellContent{color_rgb: PageCellRGB{R: 9, G: 8, B: 7}}}
	src.GetRowAndCell(7, 0).cell.wide = PageCellWideWide
	src.GetRowAndCell(8, 0).cell.wide = PageCellWideSpacerTail
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	pg2MarkRow(dst, 0, 0x900)

	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 2, 9))

	for x := range uint(10) {
		d := *dst.GetRowAndCell(x, 0).cell
		if x >= 2 && x < 9 {
			expectEqual(t, pg2Bits(*src.GetRowAndCell(x, 0).cell), pg2Bits(d))
		} else {
			expectEqual(t, pg2Bits(PageCellInit(0x900)), pg2Bits(d))
		}
	}
}

// Zig: test "term51: Page.clonePartialRowFrom: range end is clamped to both column counts"
func TestPg2_term51_Page_clonePartialRowFrom_range_end_is_clamped_to_both_column_counts(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2FillRow(src, 0, 0x100)
	dst := pg2New(t, 6, 1)
	defer dst.Deinit()
	pg2MarkRow(dst, 0, 0x900)

	// Request far beyond: clamped to min(6, 10) = 6.
	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 2, 1000))
	pg2ExpectCps(t, dst, 0, []uint32{0x900, 0x900, 0x102, 0x103, 0x104, 0x105})

	// Source narrower than destination: clamped to 4.
	src2 := pg2New(t, 4, 1)
	defer src2.Deinit()
	pg2FillRow(src2, 0, 0x200)
	dst2 := pg2New(t, 8, 1)
	defer dst2.Deinit()
	pg2MarkRow(dst2, 0, 0x900)
	zigTry(t, dst2.ClonePartialRowFrom(src2, dst2.GetRow(0), src2.GetRow(0), 0, 1000))
	pg2ExpectCps(t, dst2, 0, []uint32{0x200, 0x201, 0x202, 0x203, 0x900, 0x900, 0x900, 0x900})
}

// Zig: test "term51: Page.clonePartialRowFrom: range end exactly at the last column and start at zero"
func TestPg2_term51_Page_clonePartialRowFrom_range_end_exactly_at_the_last_column_and_start_at_zero(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2FillRow(src, 0, 0x100)
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	pg2MarkRow(dst, 0, 0x900)
	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 9, 10))
	pg2ExpectCps(t, dst, 0, []uint32{0x900, 0x900, 0x900, 0x900, 0x900, 0x900, 0x900, 0x900, 0x900, 0x109})
	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 1))
	pg2ExpectCps(t, dst, 0, []uint32{0x100, 0x900, 0x900, 0x900, 0x900, 0x900, 0x900, 0x900, 0x900, 0x109})
}

// Zig: test "term51: Page.clonePartialRowFrom: empty range copies no cells"
func TestPg2_term51_Page_clonePartialRowFrom_empty_range_copies_no_cells(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2FillRow(src, 0, 0x100)
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	pg2MarkRow(dst, 0, 0x900)

	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 4, 4))
	pg2ExpectCps(t, dst, 0, pg2Repeat(0x900, 10))
	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 10, 10))
	pg2ExpectCps(t, dst, 0, pg2Repeat(0x900, 10))
	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 0))
	pg2ExpectCps(t, dst, 0, pg2Repeat(0x900, 10))
}

// Zig: test "term51: Page.clonePartialRowFrom: empty range still refreshes false positive dst flags"
func TestPg2_term51_Page_clonePartialRowFrom_empty_range_still_refreshes_false_positive_dst_flags(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	row := dst.GetRow(0)
	row.grapheme = true
	row.styled = true
	row.hyperlink = true

	zigTry(t, dst.ClonePartialRowFrom(src, row, src.GetRow(0), 3, 3))
	expect(t, !row.grapheme, "!row.grapheme")
	expect(t, !row.styled, "!row.styled")
	expect(t, !row.hyperlink, "!row.hyperlink")

	// With real managed data elsewhere in the row the flags stay.
	pg2Grapheme(t, dst, 0, 0, 'e', []uint32{0x301})
	pg2StyleCells(t, dst, 1, [][2]uint{{1, 0}})
	pg2LinkCells(t, dst, pg2Link("https://a"), [][2]uint{{2, 0}})
	zigTry(t, dst.ClonePartialRowFrom(src, row, src.GetRow(0), 5, 5))
	expect(t, row.grapheme, "row.grapheme")
	expect(t, row.styled, "row.styled")
	expect(t, row.hyperlink, "row.hyperlink")
	expectEqual(t, uint(1), dst.GraphemeCount())
}

// Zig: test "term51: Page.clonePartialRowFrom: a full range takes the source row metadata"
func TestPg2_term51_Page_clonePartialRowFrom_a_full_range_takes_the_source_row_metadata(t *testing.T) {
	src := pg2New(t, 10, 2)
	defer src.Deinit()
	dst := pg2New(t, 10, 2)
	defer dst.Deinit()
	pg2FillRow(src, 0, 0x100)
	srow := src.GetRow(0)
	srow.wrap = true
	srow.wrap_continuation = true
	srow.semantic_prompt = PageRowSemanticPromptPrompt
	srow.dirty = true
	srow.kitty_virtual_placeholder = true
	drow := dst.GetRow(1)
	cells_before := drow.cells

	zigTry(t, dst.ClonePartialRowFrom(src, drow, srow, 0, 10))

	expect(t, drow.wrap, "drow.wrap")
	expect(t, drow.wrap_continuation, "drow.wrap_continuation")
	expectEqual(t, PageRowSemanticPromptPrompt, drow.semantic_prompt)
	expect(t, drow.dirty, "drow.dirty")
	expect(t, drow.kitty_virtual_placeholder, "drow.kitty_virtual_placeholder")
	// Zig: cells_before.offset, drow.cells.offset (Row.cells is the offset)
	expectEqual(t, cells_before, drow.cells)
	pg2ExpectCps(t, dst, 1, []uint32{0x100, 0x101, 0x102, 0x103, 0x104, 0x105, 0x106, 0x107, 0x108, 0x109})
}

// Zig: test "term51: Page.clonePartialRowFrom: a full range overwrites destination metadata, even dirty"
func TestPg2_term51_Page_clonePartialRowFrom_a_full_range_overwrites_destination_metadata_even_dirty(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	drow := dst.GetRow(0)
	drow.wrap = true
	drow.wrap_continuation = true
	drow.dirty = true
	drow.semantic_prompt = PageRowSemanticPromptPromptContinuation
	drow.kitty_virtual_placeholder = true

	zigTry(t, dst.ClonePartialRowFrom(src, drow, src.GetRow(0), 0, 10))

	expect(t, !drow.wrap, "!drow.wrap")
	expect(t, !drow.wrap_continuation, "!drow.wrap_continuation")
	// The destination's dirty flag is replaced, not merged, for full rows.
	expect(t, !drow.dirty, "!drow.dirty")
	expectEqual(t, PageRowSemanticPromptNone, drow.semantic_prompt)
	expect(t, !drow.kitty_virtual_placeholder, "!drow.kitty_virtual_placeholder")
}

// Zig: test "term51: Page.clonePartialRowFrom: false positive source flags propagate on a full row"
func TestPg2_term51_Page_clonePartialRowFrom_false_positive_source_flags_propagate_on_a_full_row(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	srow := src.GetRow(0)
	srow.grapheme = true
	srow.styled = true
	srow.hyperlink = true
	pg2FillRow(src, 0, 0x100)

	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), srow, 0, 10))

	expect(t, dst.GetRow(0).grapheme, "dst.getRow(0).grapheme")
	expect(t, dst.GetRow(0).styled, "dst.getRow(0).styled")
	expect(t, dst.GetRow(0).hyperlink, "dst.getRow(0).hyperlink")
	pg2ExpectCps(t, dst, 0, []uint32{0x100, 0x101, 0x102, 0x103, 0x104, 0x105, 0x106, 0x107, 0x108, 0x109})
	zigTry(t, dst.VerifyIntegrity())
}

// Zig: test "term51: Page.clonePartialRowFrom: a partial range keeps destination flags and merges dirty"
func TestPg2_term51_Page_clonePartialRowFrom_a_partial_range_keeps_destination_flags_and_merges_dirty(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	pg2FillRow(src, 0, 0x100)
	srow := src.GetRow(0)
	srow.wrap = false
	srow.wrap_continuation = true
	srow.dirty = true
	srow.semantic_prompt = PageRowSemanticPromptPromptContinuation
	srow.kitty_virtual_placeholder = true
	drow := dst.GetRow(0)
	drow.wrap = true
	drow.wrap_continuation = false
	drow.semantic_prompt = PageRowSemanticPromptPrompt

	zigTry(t, dst.ClonePartialRowFrom(src, drow, srow, 1, 5))

	expect(t, drow.wrap, "drow.wrap")                                             // dst's
	expect(t, !drow.wrap_continuation, "!drow.wrap_continuation")                 // dst's
	expect(t, drow.dirty, "drow.dirty")                                           // src dirty
	expectEqual(t, PageRowSemanticPromptPromptContinuation, drow.semantic_prompt) // src's
	expect(t, drow.kitty_virtual_placeholder, "drow.kitty_virtual_placeholder")   // src's

	// dst dirty, src clean: still dirty. Both clean: clean.
	srow.dirty = false
	drow.dirty = true
	zigTry(t, dst.ClonePartialRowFrom(src, drow, srow, 1, 5))
	expect(t, drow.dirty, "drow.dirty")
	drow.dirty = false
	zigTry(t, dst.ClonePartialRowFrom(src, drow, srow, 1, 5))
	expect(t, !drow.dirty, "!drow.dirty")
}

// Zig: test "term51: Page.clonePartialRowFrom: full versus partial is decided by the range width"
func TestPg2_term51_Page_clonePartialRowFrom_full_versus_partial_is_decided_by_the_range_width(t *testing.T) {
	src := pg2New(t, 6, 1)
	defer src.Deinit()
	pg2FillRow(src, 0, 0x100)
	src.GetRow(0).wrap = true
	dst := pg2New(t, 6, 1)
	defer dst.Deinit()

	// Width 5 < 6 in both placements: partial, wrap not taken.
	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 5))
	expect(t, !dst.GetRow(0).wrap, "!dst.getRow(0).wrap")
	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 1, 6))
	expect(t, !dst.GetRow(0).wrap, "!dst.getRow(0).wrap")
	// Width 6: full.
	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 6))
	expect(t, dst.GetRow(0).wrap, "dst.getRow(0).wrap")

	// A one column page: the only possible range is the full row.
	s1 := pg2New(t, 1, 1)
	defer s1.Deinit()
	s1.GetRow(0).wrap = true
	d1 := pg2New(t, 1, 1)
	defer d1.Deinit()
	zigTry(t, d1.ClonePartialRowFrom(s1, d1.GetRow(0), s1.GetRow(0), 0, 1))
	expect(t, d1.GetRow(0).wrap, "d1.getRow(0).wrap")
}

// Zig: test "term51: Page.clonePartialRowFrom: partial copy clears destination managed data in the range only"
func TestPg2_term51_Page_clonePartialRowFrom_partial_copy_clears_destination_managed_data_in_the_range_only(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2FillRow(src, 0, 0x100)
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	pg2FillRow(dst, 0, 0x900)
	pg2Grapheme(t, dst, 0, 0, 'a', []uint32{0x301})                              // out
	pg2Grapheme(t, dst, 3, 0, 'b', []uint32{0x301})                              // in
	sid := pg2StyleCells(t, dst, 1, [][2]uint{{1, 0}, {4, 0}})                   // out, in
	hid := pg2LinkCells(t, dst, pg2Link("https://a"), [][2]uint{{6, 0}, {9, 0}}) // in, out
	drow := dst.GetRow(0)

	zigTry(t, dst.ClonePartialRowFrom(src, drow, src.GetRow(0), 2, 8))

	expectEqual(t, uint(1), dst.GraphemeCount())
	expect(t, dst.GetRowAndCell(0, 0).cell.HasGrapheme(), "dst.getRowAndCell(0, 0).cell.hasGrapheme()")
	expect(t, !dst.GetRowAndCell(3, 0).cell.HasGrapheme(), "!dst.getRowAndCell(3, 0).cell.hasGrapheme()")
	expect(t, drow.grapheme, "drow.grapheme")
	expectEqual(t, uint16(1), dst.styles.RefCount(sid))
	expectEqual(t, sid, dst.GetRowAndCell(1, 0).cell.style_id)
	expectEqual(t, StyleID(0), dst.GetRowAndCell(4, 0).cell.style_id)
	expect(t, drow.styled, "drow.styled")
	expectEqual(t, uint16(1), dst.hyperlink_set.RefCount(hid))
	expectEqual(t, uint(1), dst.HyperlinkCount())
	expect(t, !dst.GetRowAndCell(6, 0).cell.hyperlink, "!dst.getRowAndCell(6, 0).cell.hyperlink")
	expect(t, dst.GetRowAndCell(9, 0).cell.hyperlink, "dst.getRowAndCell(9, 0).cell.hyperlink")
	expect(t, drow.hyperlink, "drow.hyperlink")
	zigTry(t, dst.VerifyIntegrity())
}

// Zig: test "term51: Page.clonePartialRowFrom: partial copy over the only managed cells clears the grapheme and hyperlink row flags"
func TestPg2_term51_Page_clonePartialRowFrom_partial_copy_over_the_only_managed_cells_clears_the_grapheme_and_hyperlink_row_flags(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2FillRow(src, 0, 0x100)
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	pg2FillRow(dst, 0, 0x900)
	pg2Grapheme(t, dst, 3, 0, 'b', []uint32{0x301})
	pg2StyleCells(t, dst, 1, [][2]uint{{4, 0}})
	pg2LinkCells(t, dst, pg2Link("https://a"), [][2]uint{{6, 0}})
	drow := dst.GetRow(0)
	expect(t, drow.grapheme && drow.styled && drow.hyperlink, "drow.grapheme and drow.styled and drow.hyperlink")

	zigTry(t, dst.ClonePartialRowFrom(src, drow, src.GetRow(0), 2, 8))

	expect(t, !drow.grapheme, "!drow.grapheme")
	expect(t, !drow.hyperlink, "!drow.hyperlink")
	// clearCells refreshes the styled flag before it zeroes the cells, so
	// the released style cell still counts and the flag stays set (stale).
	expect(t, drow.styled, "drow.styled")
	expectEqual(t, uint(0), dst.GraphemeCount())
	expectEqual(t, uint(0), dst.HyperlinkCount())
	expectEqual(t, uint(0), dst.styles.Count())
}

// Zig: test "term51: Page.clonePartialRowFrom: full copy over managed destination clears and replaces"
func TestPg2_term51_Page_clonePartialRowFrom_full_copy_over_managed_destination_clears_and_replaces(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2FillRow(src, 0, 0x100)
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	pg2Grapheme(t, dst, 3, 0, 'b', []uint32{0x301})
	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 10))
	expectEqual(t, uint(0), dst.GraphemeCount())
	expect(t, !dst.GetRow(0).grapheme, "!dst.getRow(0).grapheme")
	pg2ExpectCps(t, dst, 0, []uint32{0x100, 0x101, 0x102, 0x103, 0x104, 0x105, 0x106, 0x107, 0x108, 0x109})
}

// Zig: test "term51: Page.clonePartialRowFrom: copies grapheme clusters between pages"
func TestPg2_term51_Page_clonePartialRowFrom_copies_grapheme_clusters_between_pages(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2FillRow(src, 0, 0x100)
	pg2Grapheme(t, src, 2, 0, 'a', []uint32{0x301, 0x302, 0x303})
	pg2Grapheme(t, src, 5, 0, 'b', []uint32{1, 2, 3, 4, 5, 6})
	pg2Grapheme(t, src, 8, 0, 'c', []uint32{0x301})
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()

	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 10))

	expectEqual(t, uint(3), dst.GraphemeCount())
	expect(t, dst.GetRow(0).grapheme, "dst.getRow(0).grapheme")
	expectEqualSlices(t, []uint32{0x301, 0x302, 0x303}, zigUnwrap(t, dst.LookupGrapheme(dst.GetRowAndCell(2, 0).cell)))
	expectEqualSlices(t, []uint32{1, 2, 3, 4, 5, 6}, zigUnwrap(t, dst.LookupGrapheme(dst.GetRowAndCell(5, 0).cell)))
	expectEqualSlices(t, []uint32{0x301}, zigUnwrap(t, dst.LookupGrapheme(dst.GetRowAndCell(8, 0).cell)))
	expectEqual(t, uint32('b'), dst.GetRowAndCell(5, 0).cell.Codepoint())
	// 1 chunk + 2 chunks + 1 chunk
	expectEqual(t, uint64(16+32+16), dst.grapheme_alloc.UsedBytes())
	// Independent copies.
	src.ClearGrapheme(src.GetRowAndCell(2, 0).cell)
	expectEqualSlices(t, []uint32{0x301, 0x302, 0x303}, zigUnwrap(t, dst.LookupGrapheme(dst.GetRowAndCell(2, 0).cell)))
	zigTry(t, dst.VerifyIntegrity())
}

// Zig: test "term51: Page.clonePartialRowFrom: partial copy takes only the graphemes in range"
func TestPg2_term51_Page_clonePartialRowFrom_partial_copy_takes_only_the_graphemes_in_range(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2FillRow(src, 0, 0x100)
	pg2Grapheme(t, src, 1, 0, 'a', []uint32{0x301})
	pg2Grapheme(t, src, 4, 0, 'b', []uint32{0x302})
	pg2Grapheme(t, src, 8, 0, 'c', []uint32{0x303})
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()

	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 3, 6))

	expectEqual(t, uint(1), dst.GraphemeCount())
	expect(t, dst.GetRowAndCell(4, 0).cell.HasGrapheme(), "dst.getRowAndCell(4, 0).cell.hasGrapheme()")
	expectEqualSlices(t, []uint32{0x302}, zigUnwrap(t, dst.LookupGrapheme(dst.GetRowAndCell(4, 0).cell)))
	expect(t, dst.GetRow(0).grapheme, "dst.getRow(0).grapheme")
	expect(t, dst.GetRowAndCell(1, 0).cell.IsZero(), "dst.getRowAndCell(1, 0).cell.isZero()")
}

// Zig: test "term51: Page.clonePartialRowFrom: grapheme copy within the same page"
func TestPg2_term51_Page_clonePartialRowFrom_grapheme_copy_within_the_same_page(t *testing.T) {
	page := pg2New(t, 10, 2)
	defer page.Deinit()
	pg2Grapheme(t, page, 2, 0, 'a', []uint32{0x301, 0x302})
	zigTry(t, page.ClonePartialRowFrom(page, page.GetRow(1), page.GetRow(0), 0, 10))
	expectEqual(t, uint(2), page.GraphemeCount())
	a := zigUnwrap(t, page.LookupGrapheme(page.GetRowAndCell(2, 0).cell))
	b := zigUnwrap(t, page.LookupGrapheme(page.GetRowAndCell(2, 1).cell))
	expectEqualSlices(t, a, b)
	expect(t, &a[0] != &b[0], "a.ptr != b.ptr")
	expect(t, page.GetRow(1).grapheme, "page.getRow(1).grapheme")
}

// Zig: test "term51: Page.clonePartialRowFrom: styles are copied by id within one page"
func TestPg2_term51_Page_clonePartialRowFrom_styles_are_copied_by_id_within_one_page(t *testing.T) {
	page := pg2New(t, 10, 2)
	defer page.Deinit()
	pg2FillAll(page)
	sid := pg2StyleCells(t, page, 4, [][2]uint{{1, 0}, {2, 0}, {3, 0}})
	zigTry(t, page.ClonePartialRowFrom(page, page.GetRow(1), page.GetRow(0), 2, 4))
	expectEqual(t, StyleID(0), page.GetRowAndCell(1, 1).cell.style_id)
	expectEqual(t, sid, page.GetRowAndCell(2, 1).cell.style_id)
	expectEqual(t, sid, page.GetRowAndCell(3, 1).cell.style_id)
	expectEqual(t, uint16(5), page.styles.RefCount(sid))
	expect(t, page.GetRow(1).styled, "page.getRow(1).styled")
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.clonePartialRowFrom: styles keep their id across pages when the slot is free"
func TestPg2_term51_Page_clonePartialRowFrom_styles_keep_their_id_across_pages_when_the_slot_is_free(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2FillRow(src, 0, 0x100)
	a := pg2StyleCells(t, src, 1, [][2]uint{{1, 0}, {2, 0}})
	b := pg2StyleCells(t, src, 2, [][2]uint{{3, 0}})
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()

	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 10))

	expectEqual(t, a, dst.GetRowAndCell(1, 0).cell.style_id)
	expectEqual(t, a, dst.GetRowAndCell(2, 0).cell.style_id)
	expectEqual(t, b, dst.GetRowAndCell(3, 0).cell.style_id)
	expectEqual(t, uint16(2), dst.styles.RefCount(a))
	expectEqual(t, uint16(1), dst.styles.RefCount(b))
	expect(t, pg2PaletteStyle(2).Eql(*dst.styles.Get(b)), "(Style{ .fg_color = .{ .palette = 2 } }).eql(dst.styles.get(dst.memory, b).*)")
	expect(t, dst.GetRow(0).styled, "dst.getRow(0).styled")
	zigTry(t, dst.VerifyIntegrity())
}

// Zig: test "term51: Page.clonePartialRowFrom: styles are remapped when the id is taken by another style"
func TestPg2_term51_Page_clonePartialRowFrom_styles_are_remapped_when_the_id_is_taken_by_another_style(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	sid := pg2StyleCells(t, src, 5, [][2]uint{{1, 0}})
	expectEqual(t, StyleID(1), sid)
	dst := pg2New(t, 10, 2)
	defer dst.Deinit()
	// Occupy id 1 in the destination with a different style.
	other := pg2StyleCells(t, dst, 9, [][2]uint{{0, 1}})
	expectEqual(t, StyleID(1), other)

	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 10))

	got := dst.GetRowAndCell(1, 0).cell.style_id
	expectEqual(t, StyleID(2), got)
	expect(t, pg2PaletteStyle(5).Eql(*dst.styles.Get(got)), "(Style{ .fg_color = .{ .palette = 5 } }).eql(dst.styles.get(dst.memory, got).*)")
	expect(t, pg2PaletteStyle(9).Eql(*dst.styles.Get(other)), "(Style{ .fg_color = .{ .palette = 9 } }).eql(dst.styles.get(dst.memory, other).*)")
	expectEqual(t, uint16(1), dst.styles.RefCount(got))
	expectEqual(t, uint16(1), dst.styles.RefCount(other))
	zigTry(t, dst.VerifyIntegrity())
}

// Zig: test "term51: Page.clonePartialRowFrom: an equal existing style is reused and counted"
func TestPg2_term51_Page_clonePartialRowFrom_an_equal_existing_style_is_reused_and_counted(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	// Burn id 1 so the style under test has id 2 in the source.
	pg2StyleCells(t, src, 3, [][2]uint{{0, 0}})
	s2 := pg2StyleCells(t, src, 5, [][2]uint{{1, 0}})
	expectEqual(t, StyleID(2), s2)
	dst := pg2New(t, 10, 2)
	defer dst.Deinit()
	// The destination has the same style, but at id 1.
	d1 := pg2StyleCells(t, dst, 5, [][2]uint{{0, 1}})
	expectEqual(t, StyleID(1), d1)

	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 1, 2))

	expectEqual(t, d1, dst.GetRowAndCell(1, 0).cell.style_id)
	expectEqual(t, uint16(2), dst.styles.RefCount(d1))
	expectEqual(t, uint(1), dst.styles.Count())
	zigTry(t, dst.VerifyIntegrity())

	// Same style at the same id: reference count increments too.
	dst2 := pg2New(t, 10, 2)
	defer dst2.Deinit()
	pg2StyleCells(t, dst2, 3, [][2]uint{{0, 1}})
	e2 := pg2StyleCells(t, dst2, 5, [][2]uint{{1, 1}})
	expectEqual(t, StyleID(2), e2)
	zigTry(t, dst2.ClonePartialRowFrom(src, dst2.GetRow(0), src.GetRow(0), 1, 2))
	expectEqual(t, e2, dst2.GetRowAndCell(1, 0).cell.style_id)
	expectEqual(t, uint16(2), dst2.styles.RefCount(e2))
	zigTry(t, dst2.VerifyIntegrity())
}

// Zig: test "term51: Page.clonePartialRowFrom: a dead style slot in the destination is reused for the id"
func TestPg2_term51_Page_clonePartialRowFrom_a_dead_style_slot_in_the_destination_is_reused_for_the_id(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	sid := pg2StyleCells(t, src, 5, [][2]uint{{1, 0}})
	dst := pg2New(t, 10, 2)
	defer dst.Deinit()
	// Add and fully release a different style: id 1 is now dead.
	x := pg2StyleCells(t, dst, 9, [][2]uint{{0, 1}})
	dst.ClearCells(dst.GetRow(1), 0, 10)
	expectEqual(t, uint16(0), dst.styles.RefCount(x))

	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 10))

	expectEqual(t, sid, dst.GetRowAndCell(1, 0).cell.style_id)
	expect(t, pg2PaletteStyle(5).Eql(*dst.styles.Get(sid)), "(Style{ .fg_color = .{ .palette = 5 } }).eql(dst.styles.get(dst.memory, sid).*)")
	expectEqual(t, uint16(1), dst.styles.RefCount(sid))
	expectEqual(t, uint(1), dst.styles.Count())
}

// Zig: test "term51: Page.clonePartialRowFrom: hyperlinks are copied by id within one page"
func TestPg2_term51_Page_clonePartialRowFrom_hyperlinks_are_copied_by_id_within_one_page(t *testing.T) {
	page := pg2New(t, 10, 2)
	defer page.Deinit()
	pg2FillAll(page)
	hid := pg2LinkCells(t, page, pg2Link("https://a"), [][2]uint{{1, 0}, {2, 0}})
	zigTry(t, page.ClonePartialRowFrom(page, page.GetRow(1), page.GetRow(0), 0, 10))
	expectEqual(t, hid, zigUnwrap(t, page.LookupHyperlink(page.GetRowAndCell(1, 1).cell)))
	expectEqual(t, hid, zigUnwrap(t, page.LookupHyperlink(page.GetRowAndCell(2, 1).cell)))
	expectEqual(t, uint16(4), page.hyperlink_set.RefCount(hid))
	expectEqual(t, uint(4), page.HyperlinkCount())
	expect(t, page.GetRow(1).hyperlink, "page.getRow(1).hyperlink")
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.clonePartialRowFrom: a new hyperlink is duplicated into the destination page"
func TestPg2_term51_Page_clonePartialRowFrom_a_new_hyperlink_is_duplicated_into_the_destination_page(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2FillRow(src, 0, 0x100)
	sid := pg2LinkCells(t, src, pg2LinkImplicit(7, "https://example.com/path"), [][2]uint{{3, 0}})
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()

	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 10))

	expect(t, dst.GetRowAndCell(3, 0).cell.hyperlink, "dst.getRowAndCell(3, 0).cell.hyperlink")
	expect(t, dst.GetRow(0).hyperlink, "dst.getRow(0).hyperlink")
	did := zigUnwrap(t, dst.LookupHyperlink(dst.GetRowAndCell(3, 0).cell))
	expectEqual(t, sid, did) // same id preferred
	entry := dst.hyperlink_set.Get(did)
	expectEqualStrings(t, "https://example.com/path", string(entry.uri))
	// Zig reads `entry.id.implicit`, which is safety-checked against the tag.
	expectEqual(t, HyperlinkPageEntryIdImplicit, entry.id.Tag)
	expectEqual(t, SizeOffsetInt(7), entry.id.Implicit)
	expectEqual(t, uint16(1), dst.hyperlink_set.RefCount(did))
	expectEqual(t, uint(1), dst.HyperlinkCount())
	expectEqual(t, uint64(32), dst.string_alloc.UsedBytes())
	// The destination owns its own string memory.
	expect(t, &entry.uri[0] != &src.hyperlink_set.Get(sid).uri[0], "entry.uri.slice(dst.memory).ptr != src.hyperlink_set.get(src.memory, sid).uri.slice(src.memory).ptr")
	zigTry(t, dst.VerifyIntegrity())
}

// Zig: test "term51: Page.clonePartialRowFrom: explicit hyperlink ids are duplicated too"
func TestPg2_term51_Page_clonePartialRowFrom_explicit_hyperlink_ids_are_duplicated_too(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2LinkCells(t, src, pg2LinkExplicit("link-1", "https://a"), [][2]uint{{0, 0}})
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 10))
	did := zigUnwrap(t, dst.LookupHyperlink(dst.GetRowAndCell(0, 0).cell))
	entry := dst.hyperlink_set.Get(did)
	// Zig reads `entry.id.explicit`, which is safety-checked against the tag.
	expectEqual(t, HyperlinkPageEntryIdExplicit, entry.id.Tag)
	expectEqualStrings(t, "link-1", string(entry.id.Explicit))
	expectEqualStrings(t, "https://a", string(entry.uri))
	expectEqual(t, uint64(64), dst.string_alloc.UsedBytes())
}

// Zig: test "term51: Page.clonePartialRowFrom: cells sharing a hyperlink share one destination entry"
func TestPg2_term51_Page_clonePartialRowFrom_cells_sharing_a_hyperlink_share_one_destination_entry(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2LinkCells(t, src, pg2Link("https://a"), [][2]uint{{0, 0}, {4, 0}, {9, 0}})
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 10))
	id := zigUnwrap(t, dst.LookupHyperlink(dst.GetRowAndCell(0, 0).cell))
	expectEqual(t, id, zigUnwrap(t, dst.LookupHyperlink(dst.GetRowAndCell(4, 0).cell)))
	expectEqual(t, id, zigUnwrap(t, dst.LookupHyperlink(dst.GetRowAndCell(9, 0).cell)))
	expectEqual(t, uint16(3), dst.hyperlink_set.RefCount(id))
	expectEqual(t, uint(3), dst.HyperlinkCount())
	expectEqual(t, uint(1), dst.hyperlink_set.Count())
	expectEqual(t, uint64(32), dst.string_alloc.UsedBytes())
}

// Zig: test "term51: Page.clonePartialRowFrom: an equal destination hyperlink is reused without new strings"
func TestPg2_term51_Page_clonePartialRowFrom_an_equal_destination_hyperlink_is_reused_without_new_strings(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2LinkCells(t, src, pg2Link("https://other"), [][2]uint{{8, 0}})
	sid := pg2LinkCells(t, src, pg2Link("https://a"), [][2]uint{{2, 0}})
	expectEqual(t, HyperlinkID(2), sid)
	dst := pg2New(t, 10, 2)
	defer dst.Deinit()
	did := pg2LinkCells(t, dst, pg2Link("https://a"), [][2]uint{{0, 1}})
	expectEqual(t, HyperlinkID(1), did)
	strings_before := dst.string_alloc.UsedBytes()

	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 5))

	expectEqual(t, did, zigUnwrap(t, dst.LookupHyperlink(dst.GetRowAndCell(2, 0).cell)))
	expectEqual(t, uint16(2), dst.hyperlink_set.RefCount(did))
	expectEqual(t, strings_before, dst.string_alloc.UsedBytes())
	expectEqual(t, uint(1), dst.hyperlink_set.Count())
	expectEqual(t, uint(2), dst.HyperlinkCount())
	// The link outside the range was not copied.
	expect(t, !dst.GetRowAndCell(8, 0).cell.hyperlink, "!dst.getRowAndCell(8, 0).cell.hyperlink")
	zigTry(t, dst.VerifyIntegrity())
}

// Zig: test "term51: Page.clonePartialRowFrom: hyperlink ids are remapped when the id is taken"
func TestPg2_term51_Page_clonePartialRowFrom_hyperlink_ids_are_remapped_when_the_id_is_taken(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	sid := pg2LinkCells(t, src, pg2Link("https://a"), [][2]uint{{1, 0}})
	expectEqual(t, HyperlinkID(1), sid)
	dst := pg2New(t, 10, 2)
	defer dst.Deinit()
	other := pg2LinkCells(t, dst, pg2Link("https://other"), [][2]uint{{0, 1}})
	expectEqual(t, HyperlinkID(1), other)

	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 10))

	got := zigUnwrap(t, dst.LookupHyperlink(dst.GetRowAndCell(1, 0).cell))
	expectEqual(t, HyperlinkID(2), got)
	expectEqualStrings(t, "https://a", string(dst.hyperlink_set.Get(got).uri))
	expectEqualStrings(t, "https://other", string(dst.hyperlink_set.Get(other).uri))
	expectEqual(t, uint16(1), dst.hyperlink_set.RefCount(got))
	zigTry(t, dst.VerifyIntegrity())
}

// Zig: test "term51: Page.clonePartialRowFrom: hyperlinks outside a partial range are skipped"
func TestPg2_term51_Page_clonePartialRowFrom_hyperlinks_outside_a_partial_range_are_skipped(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2LinkCells(t, src, pg2Link("https://a"), [][2]uint{{8, 0}})
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 1, 5))
	expectEqual(t, uint(0), dst.HyperlinkCount())
	expectEqual(t, uint(0), dst.hyperlink_set.Count())
	// The dst flag is kept (false) in a partial copy.
	expect(t, !dst.GetRow(0).hyperlink, "!dst.getRow(0).hyperlink")
}

// Zig: test "term51: Page.clonePartialRowFrom: a cell with grapheme, hyperlink and style copies all three"
func TestPg2_term51_Page_clonePartialRowFrom_a_cell_with_grapheme_hyperlink_and_style_copies_all_three(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2FillRow(src, 0, 0x100)
	pg2Grapheme(t, src, 4, 0, 'e', []uint32{0x301, 0x302})
	pg2StyleCells(t, src, 6, [][2]uint{{4, 0}})
	pg2LinkCells(t, src, pg2Link("https://a"), [][2]uint{{4, 0}})
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()

	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 10))

	c := dst.GetRowAndCell(4, 0).cell
	expect(t, c.HasGrapheme(), "c.hasGrapheme()")
	expect(t, c.hyperlink, "c.hyperlink")
	expect(t, c.style_id != 0, "c.style_id != 0")
	expectEqualSlices(t, []uint32{0x301, 0x302}, zigUnwrap(t, dst.LookupGrapheme(c)))
	expect(t, dst.LookupHyperlink(c).Valid, "dst.lookupHyperlink(c) != null")
	expect(t, pg2PaletteStyle(6).Eql(*dst.styles.Get(c.style_id)), "(Style{ .fg_color = .{ .palette = 6 } }).eql(dst.styles.get(dst.memory, c.style_id).*)")
	row := dst.GetRow(0)
	expect(t, row.grapheme && row.styled && row.hyperlink, "row.grapheme and row.styled and row.hyperlink")
	zigTry(t, dst.VerifyIntegrity())
}

// Zig: test "term51: Page.clonePartialRowFrom: wide and spacer cells inside the range are copied"
func TestPg2_term51_Page_clonePartialRowFrom_wide_and_spacer_cells_inside_the_range_are_copied(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2FillRow(src, 0, 0x100)
	src.GetRowAndCell(3, 0).cell.wide = PageCellWideWide
	src.GetRowAndCell(4, 0).cell.wide = PageCellWideSpacerTail
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 3, 5))
	expectEqual(t, PageCellWideWide, dst.GetRowAndCell(3, 0).cell.wide)
	expectEqual(t, PageCellWideSpacerTail, dst.GetRowAndCell(4, 0).cell.wide)
	zigTry(t, dst.VerifyIntegrity())
}

// Zig: test "term51: Page.clonePartialRowFrom: a spacer tail copied after an existing wide cell is valid"
func TestPg2_term51_Page_clonePartialRowFrom_a_spacer_tail_copied_after_an_existing_wide_cell_is_valid(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	*src.GetRowAndCell(3, 0).cell = PageCellInit('a')
	src.GetRowAndCell(3, 0).cell.wide = PageCellWideWide
	src.GetRowAndCell(4, 0).cell.wide = PageCellWideSpacerTail
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	// The destination already has the wide head at x=3.
	*dst.GetRowAndCell(3, 0).cell = PageCellInit('b')
	dst.GetRowAndCell(3, 0).cell.wide = PageCellWideWide
	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 4, 5))
	expectEqual(t, uint32('b'), dst.GetRowAndCell(3, 0).cell.Codepoint())
	expectEqual(t, PageCellWideSpacerTail, dst.GetRowAndCell(4, 0).cell.wide)
}

// Zig: test "term51: Page.clonePartialRowFrom: spacer head with the same width and a full row"
func TestPg2_term51_Page_clonePartialRowFrom_spacer_head_with_the_same_width_and_a_full_row(t *testing.T) {
	src := pg2New(t, 6, 1)
	defer src.Deinit()
	src.GetRow(0).wrap = true
	src.GetRowAndCell(5, 0).cell.wide = PageCellWideSpacerHead
	dst := pg2New(t, 6, 1)
	defer dst.Deinit()
	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 6))
	expectEqual(t, PageCellWideSpacerHead, dst.GetRowAndCell(5, 0).cell.wide)
	zigTry(t, dst.VerifyIntegrity())
}

// Zig: test "term51: Page.clonePartialRowFrom: growing columns clears the last source cell spacer head"
func TestPg2_term51_Page_clonePartialRowFrom_growing_columns_clears_the_last_source_cell_spacer_head(t *testing.T) {
	src := pg2New(t, 6, 1)
	defer src.Deinit()
	src.GetRow(0).wrap = true
	src.GetRowAndCell(5, 0).cell.wide = PageCellWideSpacerHead
	dst := pg2New(t, 9, 1)
	defer dst.Deinit()
	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 9))
	expectEqual(t, PageCellWideNarrow, dst.GetRowAndCell(5, 0).cell.wide)
	expect(t, !dst.GetRow(0).wrap, "!dst.getRow(0).wrap")
	zigTry(t, dst.VerifyIntegrity())
}

// Zig: test "term51: Page.clonePartialRowFrom: kitty placeholder cells and the row flag"
func TestPg2_term51_Page_clonePartialRowFrom_kitty_placeholder_cells_and_the_row_flag(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2FillRow(src, 0, 0x100)
	*src.GetRowAndCell(3, 0).cell = PageCellInit(0x10EEEE)
	*src.GetRowAndCell(8, 0).cell = PageCellInit(0x10EEEE)
	// Row has managed memory (a false positive flag) so the slow path runs,
	// but the row's own placeholder flag is not set.
	src.GetRow(0).grapheme = true
	dst := pg2New(t, 10, 2)
	defer dst.Deinit()

	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(1), src.GetRow(0), 4, 7))
	// The placeholder at x=3 is outside the range. No flag in either mode.
	expect(t, !dst.GetRow(1).kitty_virtual_placeholder, "!dst.getRow(1).kitty_virtual_placeholder")

	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(1), src.GetRow(0), 2, 5))
	if pg2KittyGraphics {
		expect(t, dst.GetRow(1).kitty_virtual_placeholder, "dst.getRow(1).kitty_virtual_placeholder")
	} else {
		expect(t, !dst.GetRow(1).kitty_virtual_placeholder, "!dst.getRow(1).kitty_virtual_placeholder")
	}

	// The source row flag itself is always copied, wherever the placeholder is.
	src.GetRow(0).kitty_virtual_placeholder = true
	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 4, 5))
	expect(t, dst.GetRow(0).kitty_virtual_placeholder, "dst.getRow(0).kitty_virtual_placeholder")
}

// Zig: test "term51: Page.clonePartialRowFrom: restores the integrity pause counter"
func TestPg2_term51_Page_clonePartialRowFrom_restores_the_integrity_pause_counter(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2Grapheme(t, src, 0, 0, 'e', []uint32{0x301})
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 10))
	if pageSlowRuntimeSafety {
		expectEqual(t, uint64(0), dst.pause_integrity_checks)
	}

	// And on the error path too.
	capacity := PageCapacityInit(10, 1)
	capacity.styles = 8
	capacity.grapheme_bytes = 0
	small := pg2Init(t, capacity)
	defer small.Deinit()
	pg2ExpectError(t, ErrGraphemeAllocOutOfMemory, small.ClonePartialRowFrom(src, small.GetRow(0), src.GetRow(0), 0, 10))
	if pageSlowRuntimeSafety {
		expectEqual(t, uint64(0), small.pause_integrity_checks)
	}
}

// ---- clonePartialRowFrom errors -------------------------------------------

// Zig: test "term51: Page.clonePartialRowFrom: StyleSetOutOfMemory when the style set is full"
func TestPg2_term51_Page_clonePartialRowFrom_StyleSetOutOfMemory_when_the_style_set_is_full(t *testing.T) {
	capacity := PageCapacityInit(10, 1)
	capacity.styles = 16
	src := pg2Init(t, capacity)
	defer src.Deinit()
	for i := range uint(6) {
		pg2StyleCells(t, src, uint8(i+1), [][2]uint{{i, 0}})
	}
	dst := pg2New(t, 10, 1) // ids 1..5 fit
	defer dst.Deinit()

	// Five distinct styles fit, the sixth does not.
	zigTry(t, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 5))
	expectEqual(t, uint(5), dst.styles.Count())
	pg2ExpectError(t, ErrStyleSetOutOfMemory, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 5, 6))
	expectEqual(t, StyleID(0), dst.GetRowAndCell(5, 0).cell.style_id)
	expectEqual(t, uint(5), dst.styles.Count())
}

// Zig: test "term51: Page.clonePartialRowFrom: StyleSetNeedsRehash when dead ids block the set"
func TestPg2_term51_Page_clonePartialRowFrom_StyleSetNeedsRehash_when_dead_ids_block_the_set(t *testing.T) {
	capacity := PageCapacityInit(10, 1)
	capacity.styles = 16
	src := pg2Init(t, capacity)
	defer src.Deinit()
	for i := range uint(6) {
		pg2StyleCells(t, src, uint8(i+1), [][2]uint{{i, 0}})
	}
	dst := pg2New(t, 10, 2)
	defer dst.Deinit()
	// Fill ids 1..5 on row 1, then free id 2 so it is dead but not trimmable.
	for i := range uint(5) {
		pg2StyleCells(t, dst, uint8(i+21), [][2]uint{{i, 1}})
	}
	dst.ClearCells(dst.GetRow(1), 1, 2)
	expectEqual(t, uint(4), dst.styles.Count())

	pg2ExpectError(t, ErrStyleSetNeedsRehash, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 5, 6))
	expectEqual(t, StyleID(0), dst.GetRowAndCell(5, 0).cell.style_id)
	expect(t, dst.GetRow(0).styled, "dst.getRow(0).styled")
}

// Zig: test "term51: Page.clonePartialRowFrom: HyperlinkMapOutOfMemory across pages"
func TestPg2_term51_Page_clonePartialRowFrom_HyperlinkMapOutOfMemory_across_pages(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2LinkCells(t, src, pg2Link("https://src"), [][2]uint{{1, 0}})
	capacity := PageCapacityInit(10, 8)
	capacity.styles = 8
	dst := pg2Init(t, capacity)
	defer dst.Deinit()
	filler := pg2FillLinkMap(t, dst, "https://filler")
	expectEqual(t, dst.HyperlinkCapacity(), dst.HyperlinkCount())
	string_bytes := dst.string_alloc.UsedBytes()

	pg2ExpectError(t, ErrHyperlinkMapOutOfMemory, dst.ClonePartialRowFrom(src, dst.GetRow(7), src.GetRow(0), 0, 10))
	expect(t, !dst.GetRowAndCell(1, 7).cell.hyperlink, "!dst.getRowAndCell(1, 7).cell.hyperlink")
	expectEqual(t, dst.HyperlinkCapacity(), dst.HyperlinkCount())
	expectEqual(t, string_bytes, dst.string_alloc.UsedBytes())
	_ = filler
}

// Zig: test "term51: Page.clonePartialRowFrom: a full hyperlink map rejects even a link the destination has"
func TestPg2_term51_Page_clonePartialRowFrom_a_full_hyperlink_map_rejects_even_a_link_the_destination_has(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2LinkCells(t, src, pg2Link("https://filler"), [][2]uint{{1, 0}})
	capacity := PageCapacityInit(10, 8)
	capacity.styles = 8
	dst := pg2Init(t, capacity)
	defer dst.Deinit()
	pg2FillLinkMap(t, dst, "https://filler")
	pg2ExpectError(t, ErrHyperlinkMapOutOfMemory, dst.ClonePartialRowFrom(src, dst.GetRow(7), src.GetRow(0), 0, 10))
}

// Zig: test "term51: Page.clonePartialRowFrom: HyperlinkMapOutOfMemory within one page leaks a reference"
func TestPg2_term51_Page_clonePartialRowFrom_HyperlinkMapOutOfMemory_within_one_page_leaks_a_reference(t *testing.T) {
	capacity := PageCapacityInit(10, 8)
	capacity.styles = 8
	page := pg2Init(t, capacity)
	defer page.Deinit()
	filler := pg2FillLinkMap(t, page, "https://filler")
	before := page.hyperlink_set.RefCount(filler)

	pg2ExpectError(t, ErrHyperlinkMapOutOfMemory, page.ClonePartialRowFrom(page, page.GetRow(7), page.GetRow(0), 0, 1))
	// Upstream calls `use` before setHyperlink fails and never releases.
	expectEqual(t, before+1, page.hyperlink_set.RefCount(filler))
	expect(t, !page.GetRowAndCell(0, 7).cell.hyperlink, "!page.getRowAndCell(0, 7).cell.hyperlink")
}

// Zig: test "term51: Page.clonePartialRowFrom: StringAllocOutOfMemory for the uri"
func TestPg2_term51_Page_clonePartialRowFrom_StringAllocOutOfMemory_for_the_uri(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2LinkCells(t, src, pg2Link("x"), [][2]uint{{1, 0}})
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	// Use all 64 chunks of the destination string table.
	all := dst.string_alloc.CapacityBytes()
	expectEqual(t, uint64(2048), all)
	_, err := dst.string_alloc.Alloc(uint(all))
	zigTry(t, err)

	pg2ExpectError(t, ErrStringAllocOutOfMemory, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 10))
	expect(t, !dst.GetRowAndCell(1, 0).cell.hyperlink, "!dst.getRowAndCell(1, 0).cell.hyperlink")
	expectEqual(t, uint(0), dst.HyperlinkCount())
	expectEqual(t, all, dst.string_alloc.UsedBytes())
}

// Zig: test "term51: Page.clonePartialRowFrom: StringAllocOutOfMemory for the id frees the uri again"
func TestPg2_term51_Page_clonePartialRowFrom_StringAllocOutOfMemory_for_the_id_frees_the_uri_again(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2LinkCells(t, src, pg2LinkExplicit("i", "u"), [][2]uint{{1, 0}})
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	all := dst.string_alloc.CapacityBytes()
	// Leave exactly one chunk free.
	_, err := dst.string_alloc.Alloc(uint(all - 32))
	zigTry(t, err)

	pg2ExpectError(t, ErrStringAllocOutOfMemory, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 10))
	expectEqual(t, all-32, dst.string_alloc.UsedBytes())
}

// Zig: test "term51: Page.clonePartialRowFrom: HyperlinkSetOutOfMemory leaks the duplicated strings"
func TestPg2_term51_Page_clonePartialRowFrom_HyperlinkSetOutOfMemory_leaks_the_duplicated_strings(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2LinkCells(t, src, pg2Link("https://a"), [][2]uint{{1, 0}})
	// A set that can hold no item at all.
	capacity := PageCapacityInit(10, 1)
	capacity.styles = 8
	capacity.hyperlink_bytes = 2 * pg2Item
	dst := pg2Init(t, capacity)
	defer dst.Deinit()
	expectEqual(t, uint(1), dst.hyperlink_set.layout.cap)

	pg2ExpectError(t, ErrHyperlinkSetOutOfMemory, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 10))
	expect(t, !dst.GetRowAndCell(1, 0).cell.hyperlink, "!dst.getRowAndCell(1, 0).cell.hyperlink")
	expectEqual(t, uint(0), dst.HyperlinkCount())
	// The duplicated uri chunk is never freed (upstream defect).
	expectEqual(t, uint64(32), dst.string_alloc.UsedBytes())
}

// Zig: test "term51: Page.clonePartialRowFrom: HyperlinkSetNeedsRehash"
func TestPg2_term51_Page_clonePartialRowFrom_HyperlinkSetNeedsRehash(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2LinkCells(t, src, pg2Link("https://1"), [][2]uint{{0, 0}})
	pg2LinkCells(t, src, pg2Link("https://2"), [][2]uint{{1, 0}})
	third := pg2LinkCells(t, src, pg2Link("https://3"), [][2]uint{{2, 0}})
	expectEqual(t, HyperlinkID(3), third)

	// Capacity for two links (ids 1 and 2).
	capacity := PageCapacityInit(10, 2)
	capacity.styles = 8
	capacity.hyperlink_bytes = 4 * pg2Item
	dst := pg2Init(t, capacity)
	defer dst.Deinit()
	expectEqual(t, uint(3), dst.hyperlink_set.layout.cap)
	a := pg2LinkCells(t, dst, pg2Link("https://x"), [][2]uint{{0, 1}})
	pg2LinkCells(t, dst, pg2Link("https://y"), [][2]uint{{1, 1}})
	dst.ClearCells(dst.GetRow(1), 0, 1)
	expectEqual(t, uint16(0), dst.hyperlink_set.RefCount(a))

	pg2ExpectError(t, ErrHyperlinkSetNeedsRehash, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 2, 3))
	expect(t, !dst.GetRowAndCell(2, 0).cell.hyperlink, "!dst.getRowAndCell(2, 0).cell.hyperlink")
}

// Zig: test "term51: Page.clonePartialRowFrom: GraphemeAllocOutOfMemory leaves no grapheme or style behind"
func TestPg2_term51_Page_clonePartialRowFrom_GraphemeAllocOutOfMemory_leaves_no_grapheme_or_style_behind(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2Grapheme(t, src, 1, 0, 'e', []uint32{0x301})
	pg2StyleCells(t, src, 1, [][2]uint{{1, 0}})
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	all := dst.grapheme_alloc.CapacityBytes()
	_, err := dst.grapheme_alloc.Alloc(uint(all / 4)) // @sizeOf(u21) is 4
	zigTry(t, err)

	pg2ExpectError(t, ErrGraphemeAllocOutOfMemory, dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 10))
	cell := dst.GetRowAndCell(1, 0).cell
	expect(t, !cell.HasGrapheme(), "!cell.hasGrapheme()")
	expectEqual(t, uint32('e'), cell.Codepoint())
	// The grapheme step fails before the style step.
	expectEqual(t, StyleID(0), cell.style_id)
	expectEqual(t, uint(0), dst.styles.Count())
	expectEqual(t, uint(0), dst.GraphemeCount())
	expectEqual(t, all, dst.grapheme_alloc.UsedBytes())
}

// Zig: test "term51: Page.clonePartialRowFrom: GraphemeMapOutOfMemory frees the allocation again"
func TestPg2_term51_Page_clonePartialRowFrom_GraphemeMapOutOfMemory_frees_the_allocation_again(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	pg2Grapheme(t, src, 1, 0, 'e', []uint32{0x301})
	// 80 bytes of grapheme space: the map holds 8 entries (5 chunks rounded
	// up to a power of two) while the bitmap allocator has 64 chunks.
	capacity := PageCapacityInit(10, 2)
	capacity.styles = 8
	capacity.grapheme_bytes = 80
	dst := pg2Init(t, capacity)
	defer dst.Deinit()
	expectEqual(t, uint(8), dst.GraphemeCapacity())
	for x := range uint(8) {
		rac := dst.GetRowAndCell(x, 0)
		*rac.cell = PageCellInit('a')
		zigTry(t, dst.SetGraphemes(rac.row, rac.cell, []uint32{0x301}))
	}
	expectEqual(t, uint(8), dst.GraphemeCount())
	used := dst.grapheme_alloc.UsedBytes()
	expectEqual(t, uint64(8*16), used)
	expect(t, used < dst.grapheme_alloc.CapacityBytes(), "used < dst.grapheme_alloc.capacityBytes()")

	pg2ExpectError(t, ErrGraphemeMapOutOfMemory, dst.ClonePartialRowFrom(src, dst.GetRow(1), src.GetRow(0), 0, 10))
	expect(t, !dst.GetRowAndCell(1, 1).cell.HasGrapheme(), "!dst.getRowAndCell(1, 1).cell.hasGrapheme()")
	expectEqual(t, used, dst.grapheme_alloc.UsedBytes())
	expectEqual(t, uint(8), dst.GraphemeCount())
}

// ---- Page.getRow / Page.getCells / Page.getRowAndCell ---------------------

// Zig: test "term51: Page.getRow: returns the row structure for each y"
func TestPg2_term51_Page_getRow_returns_the_row_structure_for_each_y(t *testing.T) {
	page := pg2New(t, 6, 5)
	defer page.Deinit()
	rows := page.rows
	for y := range uint(5) {
		expectEqual(t, &rows[y], page.GetRow(y))
		// Zig: y * @sizeOf(Row) == @intFromPtr(getRow(y)) - @intFromPtr(getRow(0)),
		// a byte distance. In Go it is the distance in rows, via rowIndex
		// (@sizeOf(Row) is 8).
		expectEqual(t, y, page.rowIndex(page.GetRow(y))-page.rowIndex(page.GetRow(0)))
	}
	// Writes through the pointer are seen by later lookups.
	page.GetRow(3).wrap = true
	page.GetRow(3).semantic_prompt = PageRowSemanticPromptPrompt
	expect(t, page.GetRow(3).wrap, "page.getRow(3).wrap")
	expect(t, !page.GetRow(2).wrap, "!page.getRow(2).wrap")
	expect(t, !page.GetRow(4).wrap, "!page.getRow(4).wrap")
	expectEqual(t, PageRowSemanticPromptPrompt, page.GetRow(3).semantic_prompt)
}

// Zig: test "term51: Page.getRow: honors a logical size smaller than the capacity"
func TestPg2_term51_Page_getRow_honors_a_logical_size_smaller_than_the_capacity(t *testing.T) {
	page := pg2New(t, 6, 10)
	defer page.Deinit()
	page.size.rows = 4
	// The last valid row is size.rows - 1 and is the same row as before.
	expectEqual(t, &page.rows[3], page.GetRow(3))
	// Each row still addresses its own cells, with stride capacity.cols.
	cells := page.cells
	expectEqual(t, &cells[3*6], pg2RowFirstCell(page, page.GetRow(3)))
}

// Zig: test "term51: Page.getCells: returns size.cols cells of the row"
func TestPg2_term51_Page_getCells_returns_size_cols_cells_of_the_row(t *testing.T) {
	page := pg2New(t, 7, 4)
	defer page.Deinit()
	for y := range uint(4) {
		row := page.GetRow(y)
		cells := page.GetCells(row)
		expectEqual(t, uint(7), uint(len(cells)))
		expectEqual(t, pg2RowFirstCell(page, row), &cells[0])
	}
	// Written cells are visible, per row.
	page.GetCells(page.GetRow(2))[5] = PageCellInit('x')
	expectEqual(t, uint32('x'), page.GetRowAndCell(5, 2).cell.Codepoint())
	expectEqual(t, uint32(0), page.GetRowAndCell(5, 1).cell.Codepoint())
	expectEqual(t, uint32(0), page.GetRowAndCell(5, 3).cell.Codepoint())
}

// Zig: test "term51: Page.getCells: length follows size.cols not capacity.cols"
func TestPg2_term51_Page_getCells_length_follows_size_cols_not_capacity_cols(t *testing.T) {
	page := pg2New(t, 10, 3)
	defer page.Deinit()
	page.size.cols = 4
	expectEqual(t, uint(4), uint(len(page.GetCells(page.GetRow(1)))))
	page.size.cols = 1
	expectEqual(t, uint(1), uint(len(page.GetCells(page.GetRow(0)))))
	page.size.cols = 10
	expectEqual(t, uint(10), uint(len(page.GetCells(page.GetRow(2)))))
}

// Zig: test "term51: Page.getCells: layout rows are spaced by the capacity columns"
func TestPg2_term51_Page_getCells_layout_rows_are_spaced_by_the_capacity_columns(t *testing.T) {
	page := pg2New(t, 10, 3)
	defer page.Deinit()
	page.size.cols = 4
	a := page.GetCells(page.GetRow(0))
	b := page.GetCells(page.GetRow(1))
	// Zig: @intFromPtr(a.ptr) + 10 * @sizeOf(Cell) == @intFromPtr(b.ptr), a
	// byte distance. In Go it is the distance in cells, via cellIndex.
	expectEqual(t, page.cellIndex(&a[0])+10, page.cellIndex(&b[0]))
}

// Zig: test "term51: Page.getRowAndCell: maps x and y to the right row and cell"
func TestPg2_term51_Page_getRowAndCell_maps_x_and_y_to_the_right_row_and_cell(t *testing.T) {
	page := pg2New(t, 7, 5)
	defer page.Deinit()
	for y := range uint(5) {
		for x := range uint(7) {
			rac := page.GetRowAndCell(x, y)
			expectEqual(t, page.GetRow(y), rac.row)
			expectEqual(t, &page.GetCells(rac.row)[x], rac.cell)
			*rac.cell = PageCellInit(uint32(x + y*100 + 1))
		}
	}
	for y := range uint(5) {
		for x := range uint(7) {
			expectEqual(t, uint32(x+y*100+1), page.GetRowAndCell(x, y).cell.Codepoint())
		}
	}
}

// Zig: test "term51: Page.getRowAndCell: corners and a logical size smaller than the capacity"
func TestPg2_term51_Page_getRowAndCell_corners_and_a_logical_size_smaller_than_the_capacity(t *testing.T) {
	page := pg2New(t, 10, 10)
	defer page.Deinit()
	page.size.cols = 3
	page.size.rows = 2
	first := page.GetRowAndCell(0, 0)
	last := page.GetRowAndCell(2, 1)
	expectEqual(t, &page.cells[0], first.cell)
	// Row 1 starts after the capacity's 10 columns of row 0.
	expectEqual(t, &page.cells[10+2], last.cell)
	expectEqual(t, &page.rows[1], last.row)
}

// ---- Page.moveCells -------------------------------------------------------

// Zig: test "term51: Page.moveCells: partial text move leaves the rest of both rows alone"
func TestPg2_term51_Page_moveCells_partial_text_move_leaves_the_rest_of_both_rows_alone(t *testing.T) {
	page := pg2New(t, 10, 2)
	defer page.Deinit()
	pg2FillRow(page, 0, 0x100)
	pg2FillRow(page, 1, 0x200)

	page.MoveCells(page.GetRow(0), 2, page.GetRow(1), 4, 4)

	// dst x=4..7 got src x=2..5.
	pg2ExpectCps(t, page, 1, []uint32{0x200, 0x201, 0x202, 0x203, 0x102, 0x103, 0x104, 0x105, 0x208, 0x209})
	// src x=2..5 zeroed.
	pg2ExpectCps(t, page, 0, []uint32{0x100, 0x101, 0, 0, 0, 0, 0x106, 0x107, 0x108, 0x109})
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.moveCells: whole row move"
func TestPg2_term51_Page_moveCells_whole_row_move(t *testing.T) {
	page := pg2New(t, 6, 3)
	defer page.Deinit()
	pg2FillRow(page, 0, 0x100)
	pg2FillRow(page, 1, 0x200)
	page.MoveCells(page.GetRow(0), 0, page.GetRow(1), 0, 6)
	pg2ExpectCps(t, page, 1, []uint32{0x100, 0x101, 0x102, 0x103, 0x104, 0x105})
	pg2ExpectCps(t, page, 0, []uint32{0, 0, 0, 0, 0, 0})
	for _, c := range page.GetCells(page.GetRow(0)) {
		expect(t, c.IsZero(), "c.isZero()")
	}
}

// Zig: test "term51: Page.moveCells: single cell at each edge"
func TestPg2_term51_Page_moveCells_single_cell_at_each_edge(t *testing.T) {
	page := pg2New(t, 5, 2)
	defer page.Deinit()
	pg2FillRow(page, 0, 0x100)
	page.MoveCells(page.GetRow(0), 0, page.GetRow(1), 4, 1)
	page.MoveCells(page.GetRow(0), 4, page.GetRow(1), 0, 1)
	pg2ExpectCps(t, page, 1, []uint32{0x104, 0, 0, 0, 0x100})
	pg2ExpectCps(t, page, 0, []uint32{0, 0x101, 0x102, 0x103, 0})
}

// Zig: test "term51: Page.moveCells: zero length moves nothing"
func TestPg2_term51_Page_moveCells_zero_length_moves_nothing(t *testing.T) {
	page := pg2New(t, 5, 2)
	defer page.Deinit()
	pg2FillRow(page, 0, 0x100)
	pg2FillRow(page, 1, 0x200)
	page.MoveCells(page.GetRow(0), 2, page.GetRow(1), 3, 0)
	pg2ExpectCps(t, page, 0, []uint32{0x100, 0x101, 0x102, 0x103, 0x104})
	pg2ExpectCps(t, page, 1, []uint32{0x200, 0x201, 0x202, 0x203, 0x204})
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.moveCells: within one row between disjoint ranges"
func TestPg2_term51_Page_moveCells_within_one_row_between_disjoint_ranges(t *testing.T) {
	page := pg2New(t, 10, 1)
	defer page.Deinit()
	pg2FillRow(page, 0, 0x100)
	row := page.GetRow(0)
	page.MoveCells(row, 0, row, 6, 3)
	pg2ExpectCps(t, page, 0, []uint32{0, 0, 0, 0x103, 0x104, 0x105, 0x100, 0x101, 0x102, 0x109})
}

// Zig: test "term51: Page.moveCells: full row move clears stale source flags, partial keeps them"
func TestPg2_term51_Page_moveCells_full_row_move_clears_stale_source_flags_partial_keeps_them(t *testing.T) {
	page := pg2New(t, 6, 3)
	defer page.Deinit()
	pg2FillRow(page, 0, 0x100)
	src := page.GetRow(0)
	src.grapheme = true
	src.styled = true
	src.hyperlink = true
	src.wrap = true

	// Partial: flags are retained.
	page.MoveCells(src, 0, page.GetRow(1), 0, 5)
	expect(t, src.grapheme && src.styled && src.hyperlink, "src.grapheme and src.styled and src.hyperlink")

	// Full: the managed memory flags are cleared; wrap is not a managed flag.
	page.MoveCells(src, 0, page.GetRow(2), 0, 6)
	expect(t, !src.grapheme, "!src.grapheme")
	expect(t, !src.styled, "!src.styled")
	expect(t, !src.hyperlink, "!src.hyperlink")
	expect(t, src.wrap, "src.wrap")
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.moveCells: moves graphemes with their cells"
func TestPg2_term51_Page_moveCells_moves_graphemes_with_their_cells(t *testing.T) {
	page := pg2New(t, 10, 2)
	defer page.Deinit()
	pg2FillRow(page, 0, 0x100)
	pg2Grapheme(t, page, 1, 0, 'a', []uint32{0x301, 0x302})
	pg2Grapheme(t, page, 6, 0, 'b', []uint32{1, 2, 3, 4, 5})
	used := page.grapheme_alloc.UsedBytes()

	page.MoveCells(page.GetRow(0), 0, page.GetRow(1), 3, 7)

	expectEqual(t, uint(2), page.GraphemeCount())
	expectEqual(t, used, page.grapheme_alloc.UsedBytes())
	a := page.GetRowAndCell(4, 1).cell
	b := page.GetRowAndCell(9, 1).cell
	expect(t, a.HasGrapheme() && b.HasGrapheme(), "a.hasGrapheme() and b.hasGrapheme()")
	expectEqual(t, uint32('a'), a.Codepoint())
	expectEqualSlices(t, []uint32{0x301, 0x302}, zigUnwrap(t, page.LookupGrapheme(a)))
	expectEqualSlices(t, []uint32{1, 2, 3, 4, 5}, zigUnwrap(t, page.LookupGrapheme(b)))
	expect(t, page.GetRow(1).grapheme, "page.getRow(1).grapheme")
	// The source cells are blank and no longer in the map.
	expect(t, page.GetRowAndCell(1, 0).cell.IsZero(), "page.getRowAndCell(1, 0).cell.isZero()")
	expect(t, !page.LookupGrapheme(page.GetRowAndCell(1, 0).cell).Valid, "page.lookupGrapheme(page.getRowAndCell(1, 0).cell) == null")
	expect(t, !page.LookupGrapheme(page.GetRowAndCell(6, 0).cell).Valid, "page.lookupGrapheme(page.getRowAndCell(6, 0).cell) == null")
	// The move was partial (7 of 10 cells) so the source row flag is stale.
	expect(t, page.GetRow(0).grapheme, "page.getRow(0).grapheme")
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.moveCells: a partial grapheme move keeps the source row flag"
func TestPg2_term51_Page_moveCells_a_partial_grapheme_move_keeps_the_source_row_flag(t *testing.T) {
	page := pg2New(t, 10, 2)
	defer page.Deinit()
	pg2Grapheme(t, page, 1, 0, 'a', []uint32{0x301})
	pg2Grapheme(t, page, 8, 0, 'b', []uint32{0x301})
	page.MoveCells(page.GetRow(0), 0, page.GetRow(1), 0, 4)
	expect(t, page.GetRow(0).grapheme, "page.getRow(0).grapheme") // stays: partial row move
	expect(t, page.GetRowAndCell(8, 0).cell.HasGrapheme(), "page.getRowAndCell(8, 0).cell.hasGrapheme()")
	expect(t, page.GetRow(1).grapheme, "page.getRow(1).grapheme")
	expectEqual(t, uint(2), page.GraphemeCount())
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.moveCells: clears the destination range first"
func TestPg2_term51_Page_moveCells_clears_the_destination_range_first(t *testing.T) {
	page := pg2New(t, 10, 2)
	defer page.Deinit()
	pg2FillAll(page)
	// Destination range [2,6) has all three kinds of managed data;
	// x=8 (outside the range) keeps its own.
	pg2Grapheme(t, page, 2, 1, 'g', []uint32{0x301})
	pg2Grapheme(t, page, 8, 1, 'h', []uint32{0x302})
	sid := pg2StyleCells(t, page, 4, [][2]uint{{3, 1}, {8, 1}})
	hid := pg2LinkCells(t, page, pg2Link("https://a"), [][2]uint{{4, 1}, {9, 1}})

	page.MoveCells(page.GetRow(0), 0, page.GetRow(1), 2, 4)

	expectEqual(t, uint(1), page.GraphemeCount())
	expectEqual(t, uint16(1), page.styles.RefCount(sid))
	expectEqual(t, uint16(1), page.hyperlink_set.RefCount(hid))
	expectEqual(t, uint(1), page.HyperlinkCount())
	expect(t, page.GetRowAndCell(8, 1).cell.HasGrapheme(), "page.getRowAndCell(8, 1).cell.hasGrapheme()")
	expect(t, page.GetRowAndCell(9, 1).cell.hyperlink, "page.getRowAndCell(9, 1).cell.hyperlink")
	expectEqual(t, sid, page.GetRowAndCell(8, 1).cell.style_id)
	expectEqual(t, StyleID(0), page.GetRowAndCell(3, 1).cell.style_id)
	expectEqual(t, uint32(0x100), page.GetRowAndCell(2, 1).cell.Codepoint())
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.moveCells: moves hyperlinks and keeps reference counts"
func TestPg2_term51_Page_moveCells_moves_hyperlinks_and_keeps_reference_counts(t *testing.T) {
	page := pg2New(t, 10, 2)
	defer page.Deinit()
	pg2FillRow(page, 0, 0x100)
	hid := pg2LinkCells(t, page, pg2Link("https://a"), [][2]uint{{1, 0}, {5, 0}})

	page.MoveCells(page.GetRow(0), 0, page.GetRow(1), 0, 10)

	expectEqual(t, uint16(2), page.hyperlink_set.RefCount(hid))
	expectEqual(t, uint(2), page.HyperlinkCount())
	expectEqual(t, hid, zigUnwrap(t, page.LookupHyperlink(page.GetRowAndCell(1, 1).cell)))
	expectEqual(t, hid, zigUnwrap(t, page.LookupHyperlink(page.GetRowAndCell(5, 1).cell)))
	expect(t, page.GetRowAndCell(1, 1).cell.hyperlink, "page.getRowAndCell(1, 1).cell.hyperlink")
	expect(t, page.GetRow(1).hyperlink, "page.getRow(1).hyperlink")
	expect(t, !page.GetRow(0).hyperlink, "!page.getRow(0).hyperlink")
	expect(t, !page.LookupHyperlink(page.GetRowAndCell(1, 0).cell).Valid, "page.lookupHyperlink(page.getRowAndCell(1, 0).cell) == null")
	expect(t, !page.GetRowAndCell(1, 0).cell.hyperlink, "!page.getRowAndCell(1, 0).cell.hyperlink")
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.moveCells: moves styles with the same ids and sets the destination row flag"
func TestPg2_term51_Page_moveCells_moves_styles_with_the_same_ids_and_sets_the_destination_row_flag(t *testing.T) {
	page := pg2New(t, 10, 2)
	defer page.Deinit()
	pg2FillRow(page, 0, 0x100)
	a := pg2StyleCells(t, page, 1, [][2]uint{{2, 0}, {3, 0}})
	b := pg2StyleCells(t, page, 2, [][2]uint{{7, 0}})

	page.MoveCells(page.GetRow(0), 0, page.GetRow(1), 0, 10)

	expectEqual(t, a, page.GetRowAndCell(2, 1).cell.style_id)
	expectEqual(t, a, page.GetRowAndCell(3, 1).cell.style_id)
	expectEqual(t, b, page.GetRowAndCell(7, 1).cell.style_id)
	expectEqual(t, uint16(2), page.styles.RefCount(a))
	expectEqual(t, uint16(1), page.styles.RefCount(b))
	expect(t, page.GetRow(1).styled, "page.getRow(1).styled")
	expect(t, !page.GetRow(0).styled, "!page.getRow(0).styled")
	expect(t, page.GetRowAndCell(2, 0).cell.IsZero(), "page.getRowAndCell(2, 0).cell.isZero()")
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.moveCells: partial style move keeps the source styled flag"
func TestPg2_term51_Page_moveCells_partial_style_move_keeps_the_source_styled_flag(t *testing.T) {
	page := pg2New(t, 10, 2)
	defer page.Deinit()
	pg2FillRow(page, 0, 0x100)
	a := pg2StyleCells(t, page, 1, [][2]uint{{2, 0}, {8, 0}})
	page.MoveCells(page.GetRow(0), 0, page.GetRow(1), 0, 5)
	expect(t, page.GetRow(0).styled, "page.getRow(0).styled")
	expect(t, page.GetRow(1).styled, "page.getRow(1).styled")
	expectEqual(t, uint16(2), page.styles.RefCount(a))
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.moveCells: destination styled flag is recomputed from the cells"
func TestPg2_term51_Page_moveCells_destination_styled_flag_is_recomputed_from_the_cells(t *testing.T) {
	page := pg2New(t, 10, 2)
	defer page.Deinit()
	pg2FillAll(page)
	// The destination row only has a style inside the range to be cleared.
	sid := pg2StyleCells(t, page, 1, [][2]uint{{3, 1}})
	expect(t, page.GetRow(1).styled, "page.getRow(1).styled")

	page.MoveCells(page.GetRow(0), 0, page.GetRow(1), 2, 3)

	// The style is released, but clearCells refreshes the flag before it
	// zeroes the cells, so the flag stays set (stale) here.
	expect(t, page.GetRow(1).styled, "page.getRow(1).styled")
	expectEqual(t, uint16(0), page.styles.RefCount(sid))
	expectEqual(t, StyleID(0), page.GetRowAndCell(3, 1).cell.style_id)
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.moveCells: an already styled destination row stays styled"
func TestPg2_term51_Page_moveCells_an_already_styled_destination_row_stays_styled(t *testing.T) {
	page := pg2New(t, 10, 2)
	defer page.Deinit()
	pg2FillAll(page)
	pg2StyleCells(t, page, 1, [][2]uint{{9, 1}})
	page.MoveCells(page.GetRow(0), 0, page.GetRow(1), 0, 3)
	expect(t, page.GetRow(1).styled, "page.getRow(1).styled")
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.moveCells: a cell with grapheme, hyperlink and style moves intact"
func TestPg2_term51_Page_moveCells_a_cell_with_grapheme_hyperlink_and_style_moves_intact(t *testing.T) {
	page := pg2New(t, 10, 2)
	defer page.Deinit()
	pg2Grapheme(t, page, 2, 0, 'e', []uint32{0x301, 0x302})
	sid := pg2StyleCells(t, page, 3, [][2]uint{{2, 0}})
	hid := pg2LinkCells(t, page, pg2Link("https://a"), [][2]uint{{2, 0}})

	page.MoveCells(page.GetRow(0), 2, page.GetRow(1), 7, 1)

	c := page.GetRowAndCell(7, 1).cell
	expect(t, c.HasGrapheme() && c.hyperlink, "c.hasGrapheme() and c.hyperlink")
	expectEqual(t, sid, c.style_id)
	expectEqual(t, hid, zigUnwrap(t, page.LookupHyperlink(c)))
	expectEqualSlices(t, []uint32{0x301, 0x302}, zigUnwrap(t, page.LookupGrapheme(c)))
	r := page.GetRow(1)
	expect(t, r.grapheme && r.styled && r.hyperlink, "r.grapheme and r.styled and r.hyperlink")
	expect(t, page.GetRowAndCell(2, 0).cell.IsZero(), "page.getRowAndCell(2, 0).cell.isZero()")
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.moveCells: cell attributes move with the cell"
func TestPg2_term51_Page_moveCells_cell_attributes_move_with_the_cell(t *testing.T) {
	page := pg2New(t, 6, 2)
	defer page.Deinit()
	*page.GetRowAndCell(0, 0).cell = PageCell{content_tag: PageCellContentTagBgColorPalette, content: PageCellContent{color_palette: 12}}
	*page.GetRowAndCell(1, 0).cell = PageCell{content_tag: PageCellContentTagBgColorRgb, content: PageCellContent{color_rgb: PageCellRGB{R: 1, G: 2, B: 3}}}
	*page.GetRowAndCell(2, 0).cell = PageCellInit('p')
	page.GetRowAndCell(2, 0).cell.protected = true
	page.GetRowAndCell(2, 0).cell.semantic_content = PageCellSemanticContentPrompt
	*page.GetRowAndCell(3, 0).cell = PageCellInit('w')
	page.GetRowAndCell(3, 0).cell.wide = PageCellWideWide
	page.GetRowAndCell(4, 0).cell.wide = PageCellWideSpacerTail
	var want [5]uint64
	for x := range uint(5) {
		want[x] = pg2Bits(*page.GetRowAndCell(x, 0).cell)
	}

	page.MoveCells(page.GetRow(0), 0, page.GetRow(1), 1, 5)

	for x := range uint(5) {
		expectEqual(t, want[x], pg2Bits(*page.GetRowAndCell(x+1, 1).cell))
	}
	for x := range uint(6) {
		expect(t, page.GetRowAndCell(x, 0).cell.IsZero(), "page.getRowAndCell(x, 0).cell.isZero()")
	}
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.moveCells: a spacer head can be moved to a wrapped row's last column"
func TestPg2_term51_Page_moveCells_a_spacer_head_can_be_moved_to_a_wrapped_row_s_last_column(t *testing.T) {
	page := pg2New(t, 6, 2)
	defer page.Deinit()
	page.GetRow(0).wrap = true
	page.GetRow(1).wrap = true
	page.GetRowAndCell(5, 0).cell.wide = PageCellWideSpacerHead
	page.MoveCells(page.GetRow(0), 5, page.GetRow(1), 5, 1)
	expectEqual(t, PageCellWideSpacerHead, page.GetRowAndCell(5, 1).cell.wide)
	expectEqual(t, PageCellWideNarrow, page.GetRowAndCell(5, 0).cell.wide)
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.moveCells: kitty placeholder row flags"
func TestPg2_term51_Page_moveCells_kitty_placeholder_row_flags(t *testing.T) {
	page := pg2New(t, 6, 3)
	defer page.Deinit()
	pg2FillRow(page, 0, 0x100)
	*page.GetRowAndCell(2, 0).cell = PageCellInit(0x10EEEE)
	// Put the source row on the slow path with a managed memory flag.
	page.GetRow(0).grapheme = true
	page.GetRow(0).kitty_virtual_placeholder = true

	page.MoveCells(page.GetRow(0), 0, page.GetRow(1), 0, 6)

	if pg2KittyGraphics {
		expect(t, page.GetRow(1).kitty_virtual_placeholder, "page.getRow(1).kitty_virtual_placeholder")
		expect(t, !page.GetRow(0).kitty_virtual_placeholder, "!page.getRow(0).kitty_virtual_placeholder")
	} else {
		expect(t, !page.GetRow(1).kitty_virtual_placeholder, "!page.getRow(1).kitty_virtual_placeholder")
		// Not touched without kitty graphics.
		expect(t, page.GetRow(0).kitty_virtual_placeholder, "page.getRow(0).kitty_virtual_placeholder")
	}
	expectEqual(t, uint32(0x10EEEE), page.GetRowAndCell(2, 1).cell.Codepoint())
}

// Zig: test "term51: Page.moveCells: leaves other row metadata alone"
func TestPg2_term51_Page_moveCells_leaves_other_row_metadata_alone(t *testing.T) {
	page := pg2New(t, 6, 2)
	defer page.Deinit()
	pg2FillRow(page, 0, 0x100)
	page.GetRow(0).wrap = true
	page.GetRow(0).semantic_prompt = PageRowSemanticPromptPrompt
	page.GetRow(0).dirty = false
	page.GetRow(1).wrap_continuation = true
	page.MoveCells(page.GetRow(0), 0, page.GetRow(1), 0, 6)
	expect(t, page.GetRow(0).wrap, "page.getRow(0).wrap")
	expectEqual(t, PageRowSemanticPromptPrompt, page.GetRow(0).semantic_prompt)
	expect(t, page.GetRow(1).wrap_continuation, "page.getRow(1).wrap_continuation")
	expect(t, !page.GetRow(1).wrap, "!page.getRow(1).wrap")
	expect(t, !page.GetRow(0).dirty, "!page.getRow(0).dirty")
	expect(t, !page.GetRow(1).dirty, "!page.getRow(1).dirty")
}

// ---- Page.swapCells -------------------------------------------------------

// Zig: test "term51: Page.swapCells: swaps plain cells"
func TestPg2_term51_Page_swapCells_swaps_plain_cells(t *testing.T) {
	page := pg2New(t, 6, 1)
	defer page.Deinit()
	*page.GetRowAndCell(1, 0).cell = PageCellInit('a')
	page.GetRowAndCell(1, 0).cell.protected = true
	*page.GetRowAndCell(4, 0).cell = PageCell{content_tag: PageCellContentTagBgColorRgb, content: PageCellContent{color_rgb: PageCellRGB{R: 5, G: 6, B: 7}}}
	a := pg2Bits(*page.GetRowAndCell(1, 0).cell)
	b := pg2Bits(*page.GetRowAndCell(4, 0).cell)

	page.SwapCells(page.GetRowAndCell(1, 0).cell, page.GetRowAndCell(4, 0).cell)

	expectEqual(t, b, pg2Bits(*page.GetRowAndCell(1, 0).cell))
	expectEqual(t, a, pg2Bits(*page.GetRowAndCell(4, 0).cell))
	// And back.
	page.SwapCells(page.GetRowAndCell(4, 0).cell, page.GetRowAndCell(1, 0).cell)
	expectEqual(t, a, pg2Bits(*page.GetRowAndCell(1, 0).cell))
	expectEqual(t, b, pg2Bits(*page.GetRowAndCell(4, 0).cell))
}

// Zig: test "term51: Page.swapCells: adjacent cells and empty cells"
func TestPg2_term51_Page_swapCells_adjacent_cells_and_empty_cells(t *testing.T) {
	page := pg2New(t, 4, 1)
	defer page.Deinit()
	*page.GetRowAndCell(1, 0).cell = PageCellInit('a')
	page.SwapCells(page.GetRowAndCell(1, 0).cell, page.GetRowAndCell(2, 0).cell)
	pg2ExpectCps(t, page, 0, []uint32{0, 0, 'a', 0})
	page.SwapCells(page.GetRowAndCell(0, 0).cell, page.GetRowAndCell(3, 0).cell)
	pg2ExpectCps(t, page, 0, []uint32{0, 0, 'a', 0})
}

// Zig: test "term51: Page.swapCells: grapheme in the source only"
func TestPg2_term51_Page_swapCells_grapheme_in_the_source_only(t *testing.T) {
	page := pg2New(t, 6, 1)
	defer page.Deinit()
	pg2Grapheme(t, page, 1, 0, 'a', []uint32{0x301, 0x302})
	*page.GetRowAndCell(4, 0).cell = PageCellInit('z')

	page.SwapCells(page.GetRowAndCell(1, 0).cell, page.GetRowAndCell(4, 0).cell)

	expectEqual(t, uint(1), page.GraphemeCount())
	expectEqual(t, uint32('z'), page.GetRowAndCell(1, 0).cell.Codepoint())
	expect(t, !page.GetRowAndCell(1, 0).cell.HasGrapheme(), "!page.getRowAndCell(1, 0).cell.hasGrapheme()")
	expect(t, !page.LookupGrapheme(page.GetRowAndCell(1, 0).cell).Valid, "page.lookupGrapheme(page.getRowAndCell(1, 0).cell) == null")
	expectEqual(t, uint32('a'), page.GetRowAndCell(4, 0).cell.Codepoint())
	expectEqualSlices(t, []uint32{0x301, 0x302}, zigUnwrap(t, page.LookupGrapheme(page.GetRowAndCell(4, 0).cell)))
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.swapCells: grapheme in the destination only"
func TestPg2_term51_Page_swapCells_grapheme_in_the_destination_only(t *testing.T) {
	page := pg2New(t, 6, 1)
	defer page.Deinit()
	*page.GetRowAndCell(1, 0).cell = PageCellInit('z')
	pg2Grapheme(t, page, 4, 0, 'a', []uint32{0x301})

	page.SwapCells(page.GetRowAndCell(1, 0).cell, page.GetRowAndCell(4, 0).cell)

	expectEqual(t, uint(1), page.GraphemeCount())
	expectEqual(t, uint32('a'), page.GetRowAndCell(1, 0).cell.Codepoint())
	expectEqualSlices(t, []uint32{0x301}, zigUnwrap(t, page.LookupGrapheme(page.GetRowAndCell(1, 0).cell)))
	expect(t, !page.LookupGrapheme(page.GetRowAndCell(4, 0).cell).Valid, "page.lookupGrapheme(page.getRowAndCell(4, 0).cell) == null")
	expectEqual(t, uint32('z'), page.GetRowAndCell(4, 0).cell.Codepoint())
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.swapCells: graphemes in both cells are exchanged"
func TestPg2_term51_Page_swapCells_graphemes_in_both_cells_are_exchanged(t *testing.T) {
	page := pg2New(t, 6, 1)
	defer page.Deinit()
	pg2Grapheme(t, page, 1, 0, 'a', []uint32{0x301, 0x302})
	pg2Grapheme(t, page, 4, 0, 'b', []uint32{1, 2, 3, 4, 5})
	used := page.grapheme_alloc.UsedBytes()

	page.SwapCells(page.GetRowAndCell(1, 0).cell, page.GetRowAndCell(4, 0).cell)

	expectEqual(t, uint(2), page.GraphemeCount())
	expectEqual(t, used, page.grapheme_alloc.UsedBytes())
	expectEqual(t, uint32('b'), page.GetRowAndCell(1, 0).cell.Codepoint())
	expectEqualSlices(t, []uint32{1, 2, 3, 4, 5}, zigUnwrap(t, page.LookupGrapheme(page.GetRowAndCell(1, 0).cell)))
	expectEqual(t, uint32('a'), page.GetRowAndCell(4, 0).cell.Codepoint())
	expectEqualSlices(t, []uint32{0x301, 0x302}, zigUnwrap(t, page.LookupGrapheme(page.GetRowAndCell(4, 0).cell)))
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.swapCells: hyperlink in one cell only"
func TestPg2_term51_Page_swapCells_hyperlink_in_one_cell_only(t *testing.T) {
	page := pg2New(t, 6, 1)
	defer page.Deinit()
	*page.GetRowAndCell(4, 0).cell = PageCellInit('z')
	*page.GetRowAndCell(1, 0).cell = PageCellInit('a')
	hid := pg2LinkCells(t, page, pg2Link("https://a"), [][2]uint{{1, 0}})

	page.SwapCells(page.GetRowAndCell(1, 0).cell, page.GetRowAndCell(4, 0).cell)

	expectEqual(t, uint(1), page.HyperlinkCount())
	expectEqual(t, uint16(1), page.hyperlink_set.RefCount(hid))
	expect(t, !page.GetRowAndCell(1, 0).cell.hyperlink, "!page.getRowAndCell(1, 0).cell.hyperlink")
	expect(t, !page.LookupHyperlink(page.GetRowAndCell(1, 0).cell).Valid, "page.lookupHyperlink(page.getRowAndCell(1, 0).cell) == null")
	expect(t, page.GetRowAndCell(4, 0).cell.hyperlink, "page.getRowAndCell(4, 0).cell.hyperlink")
	expectEqual(t, hid, zigUnwrap(t, page.LookupHyperlink(page.GetRowAndCell(4, 0).cell)))
	expectEqual(t, uint32('a'), page.GetRowAndCell(4, 0).cell.Codepoint())

	// Now with the link in the destination only.
	page.SwapCells(page.GetRowAndCell(1, 0).cell, page.GetRowAndCell(4, 0).cell)
	expect(t, page.GetRowAndCell(1, 0).cell.hyperlink, "page.getRowAndCell(1, 0).cell.hyperlink")
	expectEqual(t, hid, zigUnwrap(t, page.LookupHyperlink(page.GetRowAndCell(1, 0).cell)))
	expect(t, !page.LookupHyperlink(page.GetRowAndCell(4, 0).cell).Valid, "page.lookupHyperlink(page.getRowAndCell(4, 0).cell) == null")
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.swapCells: hyperlinks in both cells are exchanged"
func TestPg2_term51_Page_swapCells_hyperlinks_in_both_cells_are_exchanged(t *testing.T) {
	page := pg2New(t, 6, 1)
	defer page.Deinit()
	a := pg2LinkCells(t, page, pg2Link("https://a"), [][2]uint{{1, 0}})
	b := pg2LinkCells(t, page, pg2Link("https://b"), [][2]uint{{4, 0}})
	expect(t, a != b, "a != b")

	page.SwapCells(page.GetRowAndCell(1, 0).cell, page.GetRowAndCell(4, 0).cell)

	expectEqual(t, b, zigUnwrap(t, page.LookupHyperlink(page.GetRowAndCell(1, 0).cell)))
	expectEqual(t, a, zigUnwrap(t, page.LookupHyperlink(page.GetRowAndCell(4, 0).cell)))
	expectEqual(t, uint(2), page.HyperlinkCount())
	expectEqual(t, uint16(1), page.hyperlink_set.RefCount(a))
	expectEqual(t, uint16(1), page.hyperlink_set.RefCount(b))
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.swapCells: styles swap by id without touching reference counts"
func TestPg2_term51_Page_swapCells_styles_swap_by_id_without_touching_reference_counts(t *testing.T) {
	page := pg2New(t, 6, 1)
	defer page.Deinit()
	a := pg2StyleCells(t, page, 1, [][2]uint{{1, 0}, {2, 0}})
	b := pg2StyleCells(t, page, 2, [][2]uint{{4, 0}})
	page.SwapCells(page.GetRowAndCell(1, 0).cell, page.GetRowAndCell(4, 0).cell)
	expectEqual(t, b, page.GetRowAndCell(1, 0).cell.style_id)
	expectEqual(t, a, page.GetRowAndCell(4, 0).cell.style_id)
	expectEqual(t, uint16(2), page.styles.RefCount(a))
	expectEqual(t, uint16(1), page.styles.RefCount(b))
	// Swap with an unstyled cell.
	page.SwapCells(page.GetRowAndCell(0, 0).cell, page.GetRowAndCell(4, 0).cell)
	expectEqual(t, a, page.GetRowAndCell(0, 0).cell.style_id)
	expectEqual(t, StyleID(0), page.GetRowAndCell(4, 0).cell.style_id)
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.swapCells: grapheme and hyperlink combinations in one swap"
func TestPg2_term51_Page_swapCells_grapheme_and_hyperlink_combinations_in_one_swap(t *testing.T) {
	page := pg2New(t, 6, 1)
	defer page.Deinit()
	// x=1: grapheme only. x=4: hyperlink only.
	pg2Grapheme(t, page, 1, 0, 'a', []uint32{0x301})
	*page.GetRowAndCell(4, 0).cell = PageCellInit('b')
	hid := pg2LinkCells(t, page, pg2Link("https://a"), [][2]uint{{4, 0}})

	page.SwapCells(page.GetRowAndCell(1, 0).cell, page.GetRowAndCell(4, 0).cell)

	expect(t, page.GetRowAndCell(4, 0).cell.HasGrapheme(), "page.getRowAndCell(4, 0).cell.hasGrapheme()")
	expectEqual(t, uint32('a'), page.GetRowAndCell(4, 0).cell.Codepoint())
	expect(t, !page.GetRowAndCell(4, 0).cell.hyperlink, "!page.getRowAndCell(4, 0).cell.hyperlink")
	expect(t, page.GetRowAndCell(1, 0).cell.hyperlink, "page.getRowAndCell(1, 0).cell.hyperlink")
	expectEqual(t, hid, zigUnwrap(t, page.LookupHyperlink(page.GetRowAndCell(1, 0).cell)))
	expect(t, !page.GetRowAndCell(1, 0).cell.HasGrapheme(), "!page.getRowAndCell(1, 0).cell.hasGrapheme()")
	expectEqual(t, uint(1), page.GraphemeCount())
	expectEqual(t, uint(1), page.HyperlinkCount())
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.swapCells: cells with every kind of managed data"
func TestPg2_term51_Page_swapCells_cells_with_every_kind_of_managed_data(t *testing.T) {
	page := pg2New(t, 6, 1)
	defer page.Deinit()
	pg2Grapheme(t, page, 0, 0, 'a', []uint32{0x301})
	pg2Grapheme(t, page, 5, 0, 'b', []uint32{0x302, 0x303})
	s0 := pg2StyleCells(t, page, 1, [][2]uint{{0, 0}})
	s5 := pg2StyleCells(t, page, 2, [][2]uint{{5, 0}})
	h0 := pg2LinkCells(t, page, pg2Link("https://0"), [][2]uint{{0, 0}})
	h5 := pg2LinkCells(t, page, pg2Link("https://5"), [][2]uint{{5, 0}})

	page.SwapCells(page.GetRowAndCell(0, 0).cell, page.GetRowAndCell(5, 0).cell)

	l := page.GetRowAndCell(0, 0).cell
	r := page.GetRowAndCell(5, 0).cell
	expectEqual(t, uint32('b'), l.Codepoint())
	expectEqual(t, uint32('a'), r.Codepoint())
	expectEqual(t, s5, l.style_id)
	expectEqual(t, s0, r.style_id)
	expectEqual(t, h5, zigUnwrap(t, page.LookupHyperlink(l)))
	expectEqual(t, h0, zigUnwrap(t, page.LookupHyperlink(r)))
	expectEqualSlices(t, []uint32{0x302, 0x303}, zigUnwrap(t, page.LookupGrapheme(l)))
	expectEqualSlices(t, []uint32{0x301}, zigUnwrap(t, page.LookupGrapheme(r)))
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.swapCells: swapping a cell with itself changes nothing"
func TestPg2_term51_Page_swapCells_swapping_a_cell_with_itself_changes_nothing(t *testing.T) {
	page := pg2New(t, 4, 1)
	defer page.Deinit()
	pg2Grapheme(t, page, 1, 0, 'a', []uint32{0x301})
	hid := pg2LinkCells(t, page, pg2Link("https://a"), [][2]uint{{1, 0}})
	before := pg2Bits(*page.GetRowAndCell(1, 0).cell)
	cell := page.GetRowAndCell(1, 0).cell

	page.SwapCells(cell, cell)

	expectEqual(t, before, pg2Bits(*cell))
	expectEqualSlices(t, []uint32{0x301}, zigUnwrap(t, page.LookupGrapheme(cell)))
	expectEqual(t, hid, zigUnwrap(t, page.LookupHyperlink(cell)))
	expectEqual(t, uint(1), page.GraphemeCount())
	expectEqual(t, uint(1), page.HyperlinkCount())

	plain := page.GetRowAndCell(3, 0).cell
	*plain = PageCellInit('q')
	page.SwapCells(plain, plain)
	expectEqual(t, uint32('q'), plain.Codepoint())
}

// ---- Go-only panic tests --------------------------------------------------
//
// Zig tests cannot test an assert or a safety-checked trap (the test binary
// would abort), so upstream's tests avoid these paths. Each test below sets a
// page up outside pg2ExpectPanic and then makes the one call that must stop
// the program. The conditions are those of the functions in page.zig as
// listed in each title.

// Go-only: panic test: ExactRowCapacity: assert(y_start < y_end), an empty range
func TestPg2_panic_ExactRowCapacity_empty_range(t *testing.T) {
	page := pg2New(t, 4, 2)
	defer page.Deinit()
	pg2ExpectPanic(t, func() { page.ExactRowCapacity(1, 1) })
}

// Go-only: panic test: ExactRowCapacity: assert(y_start < y_end), start after end
func TestPg2_panic_ExactRowCapacity_start_after_end(t *testing.T) {
	page := pg2New(t, 4, 3)
	defer page.Deinit()
	pg2ExpectPanic(t, func() { page.ExactRowCapacity(2, 1) })
}

// Go-only: panic test: ExactRowCapacity: assert(y_end <= self.size.rows)
func TestPg2_panic_ExactRowCapacity_end_past_size_rows(t *testing.T) {
	page := pg2New(t, 4, 3)
	defer page.Deinit()
	page.size.rows = 2
	pg2ExpectPanic(t, func() { page.ExactRowCapacity(0, 3) })
}

// Go-only: panic test: CloneFrom: assert(y_start <= y_end)
func TestPg2_panic_CloneFrom_start_after_end(t *testing.T) {
	src := pg2New(t, 4, 4)
	defer src.Deinit()
	dst := pg2New(t, 4, 4)
	defer dst.Deinit()
	pg2ExpectPanic(t, func() { dst.CloneFrom(src, 2, 1) })
}

// Go-only: panic test: CloneFrom: assert(y_end <= other.size.rows)
func TestPg2_panic_CloneFrom_end_past_source_rows(t *testing.T) {
	src := pg2New(t, 4, 4)
	defer src.Deinit()
	src.size.rows = 2
	dst := pg2New(t, 4, 4)
	defer dst.Deinit()
	pg2ExpectPanic(t, func() { dst.CloneFrom(src, 0, 3) })
}

// Go-only: panic test: CloneFrom: assert(y_end - y_start <= self.size.rows)
func TestPg2_panic_CloneFrom_range_taller_than_destination(t *testing.T) {
	src := pg2New(t, 4, 4)
	defer src.Deinit()
	dst := pg2New(t, 4, 4)
	defer dst.Deinit()
	dst.size.rows = 2
	pg2ExpectPanic(t, func() { dst.CloneFrom(src, 0, 3) })
}

// Go-only: panic test: ClonePartialRowFrom: assert(x_start <= x_end), start after the requested end
func TestPg2_panic_ClonePartialRowFrom_start_after_end(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	pg2ExpectPanic(t, func() { dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 5, 3) })
}

// Go-only: panic test: ClonePartialRowFrom: assert(x_start <= x_end), start past the clamped end
func TestPg2_panic_ClonePartialRowFrom_start_past_clamped_end(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	dst := pg2New(t, 6, 1)
	defer dst.Deinit()
	// x_end is clamped to min(6, 10) = 6, which is below x_start.
	pg2ExpectPanic(t, func() { dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 7, 1000) })
}

// Go-only: panic test: ClonePartialRowFrom: source row without managed memory flags has a hyperlink cell (slow_runtime_safety)
func TestPg2_panic_ClonePartialRowFrom_unmarked_hyperlink_cell(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	src.GetRowAndCell(1, 0).cell.hyperlink = true
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	pg2ExpectPanic(t, func() { dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 10) })
}

// Go-only: panic test: ClonePartialRowFrom: source row without managed memory flags has a grapheme cell (slow_runtime_safety)
func TestPg2_panic_ClonePartialRowFrom_unmarked_grapheme_cell(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	src.GetRowAndCell(1, 0).cell.content_tag = PageCellContentTagCodepointGrapheme
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	pg2ExpectPanic(t, func() { dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 10) })
}

// Go-only: panic test: ClonePartialRowFrom: source row without managed memory flags has a styled cell (slow_runtime_safety)
func TestPg2_panic_ClonePartialRowFrom_unmarked_styled_cell(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	src.GetRowAndCell(1, 0).cell.style_id = 1
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	pg2ExpectPanic(t, func() { dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 10) })
}

// Go-only: panic test: ClonePartialRowFrom: other.lookupGrapheme(src_cell).? is null for a grapheme cell
func TestPg2_panic_ClonePartialRowFrom_grapheme_cell_without_data(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	src.GetRow(0).grapheme = true
	src.GetRowAndCell(1, 0).cell.content_tag = PageCellContentTagCodepointGrapheme
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	pg2ExpectPanic(t, func() { dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 10) })
}

// Go-only: panic test: ClonePartialRowFrom: other.lookupHyperlink(src_cell).? is null for a hyperlink cell
func TestPg2_panic_ClonePartialRowFrom_hyperlink_cell_without_data(t *testing.T) {
	src := pg2New(t, 10, 1)
	defer src.Deinit()
	src.GetRow(0).hyperlink = true
	src.GetRowAndCell(1, 0).cell.hyperlink = true
	dst := pg2New(t, 10, 1)
	defer dst.Deinit()
	pg2ExpectPanic(t, func() { dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 10) })
}

// Go-only: panic test: ClonePartialRowFrom: growing columns indexes cells[other.size.cols - 1] past a shorter copied range
func TestPg2_panic_ClonePartialRowFrom_spacer_head_index_past_range(t *testing.T) {
	src := pg2New(t, 4, 1)
	defer src.Deinit()
	dst := pg2New(t, 8, 1)
	defer dst.Deinit()
	// cells is [0..2] (two cells), the index is other.size.cols - 1 = 3.
	pg2ExpectPanic(t, func() { dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 2) })
}

// Go-only: panic test: GetRow: assert(y < self.size.rows)
func TestPg2_panic_GetRow_y_past_size_rows(t *testing.T) {
	page := pg2New(t, 4, 4)
	defer page.Deinit()
	page.size.rows = 2
	pg2ExpectPanic(t, func() { page.GetRow(2) })
}

// Go-only: panic test: GetCells: row is not one of the page's rows (slow_runtime_safety)
func TestPg2_panic_GetCells_row_of_another_page(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	page := pg2New(t, 4, 2)
	defer page.Deinit()
	other := pg2New(t, 4, 2)
	defer other.Deinit()
	row := other.GetRow(0)
	pg2ExpectPanic(t, func() { page.GetCells(row) })
}

// Go-only: panic test: GetRowAndCell: assert(y < self.size.rows)
func TestPg2_panic_GetRowAndCell_y_past_size_rows(t *testing.T) {
	page := pg2New(t, 4, 4)
	defer page.Deinit()
	page.size.rows = 2
	pg2ExpectPanic(t, func() { page.GetRowAndCell(0, 2) })
}

// Go-only: panic test: GetRowAndCell: assert(x < self.size.cols)
func TestPg2_panic_GetRowAndCell_x_past_size_cols(t *testing.T) {
	page := pg2New(t, 4, 4)
	defer page.Deinit()
	page.size.cols = 2
	pg2ExpectPanic(t, func() { page.GetRowAndCell(2, 0) })
}

// Go-only: panic test: MoveCells: assertIntegrity after a spacer head moved to a row that is not wrapped
func TestPg2_panic_MoveCells_spacer_head_to_unwrapped_row(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	page := pg2New(t, 6, 2)
	defer page.Deinit()
	page.GetRow(0).wrap = true
	page.GetRowAndCell(5, 0).cell.wide = PageCellWideSpacerHead
	pg2ExpectPanic(t, func() { page.MoveCells(page.GetRow(0), 5, page.GetRow(1), 5, 1) })
}

// Go-only: panic test: MoveCells: assertIntegrity after a spacer tail moved to the first column
func TestPg2_panic_MoveCells_spacer_tail_to_first_column(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	page := pg2New(t, 6, 2)
	defer page.Deinit()
	page.GetRowAndCell(0, 0).cell.wide = PageCellWideWide
	page.GetRowAndCell(1, 0).cell.wide = PageCellWideSpacerTail
	pg2ExpectPanic(t, func() { page.MoveCells(page.GetRow(0), 1, page.GetRow(1), 0, 1) })
}

// Go-only: panic test: SwapCells: assertIntegrity after a spacer tail is swapped to the first column
func TestPg2_panic_SwapCells_spacer_tail_to_first_column(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	page := pg2New(t, 6, 1)
	defer page.Deinit()
	page.GetRowAndCell(0, 0).cell.wide = PageCellWideWide
	page.GetRowAndCell(1, 0).cell.wide = PageCellWideSpacerTail
	pg2ExpectPanic(t, func() { page.SwapCells(page.GetRowAndCell(0, 0).cell, page.GetRowAndCell(1, 0).cell) })
}

// Go-only: panic test: SwapCells: map.getEntry(offset).? is null when both cells claim a grapheme
func TestPg2_panic_SwapCells_both_grapheme_cells_without_data(t *testing.T) {
	page := pg2New(t, 6, 1)
	defer page.Deinit()
	page.GetRow(0).grapheme = true
	page.GetRowAndCell(1, 0).cell.content_tag = PageCellContentTagCodepointGrapheme
	page.GetRowAndCell(4, 0).cell.content_tag = PageCellContentTagCodepointGrapheme
	pg2ExpectPanic(t, func() { page.SwapCells(page.GetRowAndCell(1, 0).cell, page.GetRowAndCell(4, 0).cell) })
}

// Go-only: panic test: SwapCells: map.getEntry(offset).? is null when both cells claim a hyperlink
func TestPg2_panic_SwapCells_both_hyperlink_cells_without_data(t *testing.T) {
	page := pg2New(t, 6, 1)
	defer page.Deinit()
	page.GetRow(0).hyperlink = true
	page.GetRowAndCell(1, 0).cell.hyperlink = true
	page.GetRowAndCell(4, 0).cell.hyperlink = true
	pg2ExpectPanic(t, func() { page.SwapCells(page.GetRowAndCell(1, 0).cell, page.GetRowAndCell(4, 0).cell) })
}

// Go-only: panic test: SwapCells: moveGrapheme finds no map entry for the source grapheme cell
func TestPg2_panic_SwapCells_grapheme_cell_without_data(t *testing.T) {
	page := pg2New(t, 6, 1)
	defer page.Deinit()
	page.GetRow(0).grapheme = true
	page.GetRowAndCell(1, 0).cell.content_tag = PageCellContentTagCodepointGrapheme
	pg2ExpectPanic(t, func() { page.SwapCells(page.GetRowAndCell(1, 0).cell, page.GetRowAndCell(4, 0).cell) })
}

// Go-only: panic test: SwapCells: moveHyperlink finds no map entry for the source hyperlink cell
func TestPg2_panic_SwapCells_hyperlink_cell_without_data(t *testing.T) {
	page := pg2New(t, 6, 1)
	defer page.Deinit()
	page.GetRow(0).hyperlink = true
	page.GetRowAndCell(1, 0).cell.hyperlink = true
	pg2ExpectPanic(t, func() { page.SwapCells(page.GetRowAndCell(1, 0).cell, page.GetRowAndCell(4, 0).cell) })
}
