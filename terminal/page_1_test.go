// Tests translated one for one from src/terminal/page.zig (Ghostty v1.3.1
// plus the term51 additional tests), first third: the tests from
// "Page.layout can take a maxed capacity" up to and including
// "term51: Page.exactRowCapacity: a grapheme tag without map data adds
// nothing". Each Go test carries the exact Zig title in a "// Zig: test"
// comment. Helpers are prefixed pg1.
//
// Translation notes:
//   - `try testing.expect(x)` is expect(t, x, "<the Zig expression>"), and
//     expectEqual / expectEqualSlices / expectEqualStrings keep Zig's names
//     and Zig's (expected, actual) argument order. `try` on a call that
//     returns an error is zigTry(t, err); `x.?` is zigUnwrap.
//     `try testing.expectError(error.X, call)` is pg1ExpectError(t, ErrX,
//     <the error of the call>).
//   - `Page.init(.{ .cols = c, .rows = r, ... })` leaves the omitted fields
//     at Capacity's declared defaults; Go composite literals do not, so the
//     tests build the capacity with PageCapacityInit and assign the fields
//     the Zig literal sets (pg1Cap is that for the common
//     `.{ .cols, .rows, .styles }` literal). pg1Init is `try Page.init(cap)`.
//   - `std_capacity` is PageStdCapacity with grapheme_bytes 512: Zig uses 512
//     under builtin.is_test (see page.go), and these tests run under test.
//   - The `memory` / `base` arguments are gone (Go memory). `page.rows.ptr(
//     page.memory)[y]` is &page.rows[y] and `row.cells.ptr(page.memory)[x]`
//     is page.GetCells(row)[x]. Packed-struct bit casts of a Row compare the
//     PageRow structs; those of a Cell use PageCell.bitCast.
//   - `std.heap.page_size_min` is 4096 and @sizeOf(Cell), @sizeOf(Row) are 8
//     (see page.go).
//   - `if (std.valgrind.runningOnValgrind() > 0) return error.SkipZigTest;`
//     at the start of the verifyIntegrity tests is dropped with a
//     `// Zig-only:` comment in each test; `defer page.deinit()` is
//     `defer page.Deinit()`, the stub every test calls.
//   - Zig's `testing.allocator` / `alloc` arguments of verifyIntegrity are
//     dropped: VerifyIntegrity takes none.
//   - Tests that were not translated are listed in
//     scripts/not-translated.txt.

package terminal

import (
	"errors"
	"math"
	"testing"
)

// pg1ExpectError is Zig's `try testing.expectError(expected, actual)`.
func pg1ExpectError(t *testing.T, want error, got error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

// pg1ExpectPanic runs f and fails the test if f does not panic.
func pg1ExpectPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if recover() == nil {
			t.Fatalf("expected a panic, got none")
		}
	}()
	f()
}

// pg1PageSizeMin is std.heap.page_size_min as recorded in the layouts.
const pg1PageSizeMin = 4096

// pg1SizeOfCell and pg1SizeOfRow are @sizeOf(Cell) and @sizeOf(Row).
const (
	pg1SizeOfCell = 8
	pg1SizeOfRow  = 8
)

// pg1Cap is the Zig literal `.{ .cols = cols, .rows = rows, .styles =
// styles }`: every other field at its default.
func pg1Cap(cols SizeCellCountInt, rows SizeCellCountInt, styles SizeStyleCountInt) PageCapacity {
	c := PageCapacityInit(cols, rows)
	c.styles = styles
	return c
}

// pg1StdCapacity is std_capacity as the Zig tests see it
// (builtin.is_test: grapheme_bytes is 512, not 8192).
func pg1StdCapacity() PageCapacity {
	c := PageStdCapacity
	c.grapheme_bytes = 512
	return c
}

// pg1Init is `try Page.init(cap)`.
func pg1Init(t *testing.T, cap PageCapacity) *Page {
	t.Helper()
	page, err := PageInit(cap)
	zigTry(t, err)
	return &page
}

// pg1Clone is `try page.clone()`.
func pg1Clone(t *testing.T, page *Page) *Page {
	t.Helper()
	c, err := page.Clone()
	zigTry(t, err)
	return &c
}

// pg1StylesAdd is `try page.styles.add(page.memory, style)`.
func pg1StylesAdd(t *testing.T, page *Page, style Style) StyleID {
	t.Helper()
	id, err := page.styles.Add(style)
	zigTry(t, err)
	return id
}

// pg1InsertHyperlink is `try page.insertHyperlink(link)`.
func pg1InsertHyperlink(t *testing.T, page *Page, link Hyperlink) HyperlinkID {
	t.Helper()
	id, err := page.InsertHyperlink(link)
	zigTry(t, err)
	return id
}

// pg1Codepoint is the Zig literal `.{ .content_tag = .codepoint, .content =
// .{ .codepoint = cp } }`.
func pg1Codepoint(cp uint32) PageCell {
	return pageCellFields{
		content_tag: PageCellContentTagCodepoint,
		content:     pageCellContentFields{codepoint: cp}.pack(),
	}.pack()
}

// pg1Implicit is the Zig literal `.{ .implicit = n }` of Hyperlink.Id.
func pg1Implicit(n SizeOffsetInt) HyperlinkHyperlinkId {
	return HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdImplicit, Implicit: n}
}

// pg1Explicit is the Zig literal `.{ .explicit = id }` of Hyperlink.Id.
func pg1Explicit(id string) HyperlinkHyperlinkId {
	return HyperlinkHyperlinkId{Tag: HyperlinkHyperlinkIdExplicit, Explicit: []uint8(id)}
}

// ---------------------------------------------------------------------------
// Upstream tests.
// ---------------------------------------------------------------------------

// Zig: test "Page.layout can take a maxed capacity"
func TestPg1_Page_layout_can_take_a_maxed_capacity(t *testing.T) {
	// Our intention is for a maxed-out capacity to always fit
	// within a page layout without triggering runtime safety on any
	// overflow. This simplifies some of our handling downstream of the
	// call (relevant to: https://github.com/ghostty-org/ghostty/issues/10258)
	//
	// Zig sets every field of Capacity to the maximum of its type with an
	// `inline for` over the fields; the six fields are written out.
	cap := PageCapacity{
		cols:            math.MaxUint16,
		rows:            math.MaxUint16,
		styles:          math.MaxUint16,
		hyperlink_bytes: math.MaxUint16,
		grapheme_bytes:  math.MaxUint32,
		string_bytes:    math.MaxUint32,
	}

	// Note that a max capacity will exceed our max_page_size so we
	// can't init a page with it, but it should layout.
	_ = PageLayoutOf(cap)
}

// Zig: test "Cell is zero by default"
func TestPg1_Cell_is_zero_by_default(t *testing.T) {
	cell := PageCellInit(0)
	cell_int := cell.bitCast()
	expectEqual(t, uint64(0), cell_int)

	// The zero value should be output type for semantic content.
	// This is very important for our assumptions elsewhere.
	expectEqual(t, PageCellSemanticContentOutput, cell.SemanticContent())
}

// Zig: test "Page capacity adjust cols down"
func TestPg1_Page_capacity_adjust_cols_down(t *testing.T) {
	original := pg1StdCapacity()
	original_size := PageLayoutOf(original).total_size
	adjusted, err := original.Adjust(PageCapacityAdjustment{cols: Some(original.cols / 2)})
	zigTry(t, err)
	adjusted_size := PageLayoutOf(adjusted).total_size
	expectEqual(t, original_size, adjusted_size)
	// If we layout a page with 1 more row and it's still the same size
	// then adjust is not producing enough rows.
	bigger := adjusted
	bigger.rows += 1
	bigger_size := PageLayoutOf(bigger).total_size
	expect(t, bigger_size > original_size, "bigger_size > original_size")
}

// Zig: test "Page capacity adjust cols down to 1"
func TestPg1_Page_capacity_adjust_cols_down_to_1(t *testing.T) {
	original := pg1StdCapacity()
	original_size := PageLayoutOf(original).total_size
	adjusted, err := original.Adjust(PageCapacityAdjustment{cols: Some[SizeCellCountInt](1)})
	zigTry(t, err)
	adjusted_size := PageLayoutOf(adjusted).total_size
	expectEqual(t, original_size, adjusted_size)
	// If we layout a page with 1 more row and it's still the same size
	// then adjust is not producing enough rows.
	bigger := adjusted
	bigger.rows += 1
	bigger_size := PageLayoutOf(bigger).total_size
	expect(t, bigger_size > original_size, "bigger_size > original_size")
}

// Zig: test "Page capacity adjust cols up"
func TestPg1_Page_capacity_adjust_cols_up(t *testing.T) {
	original := pg1StdCapacity()
	original_size := PageLayoutOf(original).total_size
	adjusted, err := original.Adjust(PageCapacityAdjustment{cols: Some(original.cols * 2)})
	zigTry(t, err)
	adjusted_size := PageLayoutOf(adjusted).total_size
	expectEqual(t, original_size, adjusted_size)
	// If we layout a page with 1 more row and it's still the same size
	// then adjust is not producing enough rows.
	bigger := adjusted
	bigger.rows += 1
	bigger_size := PageLayoutOf(bigger).total_size
	expect(t, bigger_size > original_size, "bigger_size > original_size")
}

// Zig: test "Page capacity adjust cols sweep"
func TestPg1_Page_capacity_adjust_cols_sweep(t *testing.T) {
	cp := pg1StdCapacity()
	original_cols := cp.cols
	original_size := PageLayoutOf(cp).total_size
	for c := uint(1); c < uint(original_cols)*2; c++ {
		var err error
		cp, err = cp.Adjust(PageCapacityAdjustment{cols: Some(SizeCellCountInt(c))})
		zigTry(t, err)
		adjusted_size := PageLayoutOf(cp).total_size
		expectEqual(t, original_size, adjusted_size)
		// If we layout a page with 1 more row and it's still the same size
		// then adjust is not producing enough rows.
		bigger := cp
		bigger.rows += 1
		bigger_size := PageLayoutOf(bigger).total_size
		expect(t, bigger_size > original_size, "bigger_size > original_size")
	}
}

// Zig: test "Page capacity adjust cols too high"
func TestPg1_Page_capacity_adjust_cols_too_high(t *testing.T) {
	original := pg1StdCapacity()
	_, err := original.Adjust(PageCapacityAdjustment{cols: Some[SizeCellCountInt](math.MaxUint16)})
	pg1ExpectError(t, ErrOutOfMemory, err)
}

// Zig: test "Capacity maxCols basic"
func TestPg1_Capacity_maxCols_basic(t *testing.T) {
	cp := pg1StdCapacity()
	max := zigUnwrap(t, cp.MaxCols())

	// maxCols should be >= current cols (since current capacity is valid)
	expect(t, max >= cp.cols, "max >= cap.cols")

	// Adjusting to maxCols should succeed with at least 1 row
	adjusted, err := cp.Adjust(PageCapacityAdjustment{cols: Some(max)})
	zigTry(t, err)
	expect(t, adjusted.rows >= 1, "adjusted.rows >= 1")

	// Adjusting to maxCols + 1 should fail
	_, err = cp.Adjust(PageCapacityAdjustment{cols: Some(max + 1)})
	pg1ExpectError(t, ErrOutOfMemory, err)
}

// Zig: test "Capacity maxCols preserves total size"
func TestPg1_Capacity_maxCols_preserves_total_size(t *testing.T) {
	cp := pg1StdCapacity()
	original_size := PageLayoutOf(cp).total_size
	max := zigUnwrap(t, cp.MaxCols())
	adjusted, err := cp.Adjust(PageCapacityAdjustment{cols: Some(max)})
	zigTry(t, err)
	adjusted_size := PageLayoutOf(adjusted).total_size
	expectEqual(t, original_size, adjusted_size)
}

// Zig: test "Capacity maxCols with 1 row exactly"
func TestPg1_Capacity_maxCols_with_1_row_exactly(t *testing.T) {
	cp := pg1StdCapacity()
	max := zigUnwrap(t, cp.MaxCols())
	adjusted, err := cp.Adjust(PageCapacityAdjustment{cols: Some(max)})
	zigTry(t, err)
	expectEqual(t, SizeCellCountInt(1), adjusted.rows)
}

// Zig: test "Page init"
func TestPg1_Page_init(t *testing.T) {
	page := pg1Init(t, pg1Cap(120, 80, 32))
	defer page.Deinit()
}

// Zig: test "Page read and write cells"
func TestPg1_Page_read_and_write_cells(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()

	for y := range uint(page.capacity.rows) {
		rac := page.GetRowAndCell(1, y)
		*rac.cell = pg1Codepoint(uint32(y))
	}

	// Read it again
	for y := range uint(page.capacity.rows) {
		rac := page.GetRowAndCell(1, y)
		expectEqual(t, uint32(y), rac.cell.content().codepoint())
	}
}

// Zig: test "Page appendGrapheme small"
func TestPg1_Page_appendGrapheme_small(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()

	rac := page.GetRowAndCell(0, 0)
	*rac.cell = PageCellInit(0x09)

	// One
	zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 0x0A))
	expect(t, rac.row.grapheme, "rac.row.grapheme")
	expect(t, rac.cell.HasGrapheme(), "rac.cell.hasGrapheme()")
	expectEqualSlices(t, []uint32{0x0A}, zigUnwrap(t, page.LookupGrapheme(rac.cell)))

	// Two
	zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 0x0B))
	expect(t, rac.row.grapheme, "rac.row.grapheme")
	expect(t, rac.cell.HasGrapheme(), "rac.cell.hasGrapheme()")
	expectEqualSlices(t, []uint32{0x0A, 0x0B}, zigUnwrap(t, page.LookupGrapheme(rac.cell)))

	// Clear it
	page.ClearGrapheme(rac.cell)
	page.UpdateRowGraphemeFlag(rac.row)
	expect(t, !rac.row.grapheme, "!rac.row.grapheme")
	expect(t, !rac.cell.HasGrapheme(), "!rac.cell.hasGrapheme()")
}

// Zig: test "Page appendGrapheme larger than chunk"
func TestPg1_Page_appendGrapheme_larger_than_chunk(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()

	rac := page.GetRowAndCell(0, 0)
	*rac.cell = PageCellInit(0x09)

	const count = pageGraphemeChunkLen * 10
	for i := range uint(count) {
		zigTry(t, page.AppendGrapheme(rac.row, rac.cell, uint32(0x0A+i)))
	}

	cps := zigUnwrap(t, page.LookupGrapheme(rac.cell))
	expectEqual(t, uint(count), uint(len(cps)))
	for i := range uint(count) {
		expectEqual(t, uint32(0x0A+i), cps[i])
	}
}

// Zig: test "Page clearGrapheme not all cells"
func TestPg1_Page_clearGrapheme_not_all_cells(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()

	rac := page.GetRowAndCell(0, 0)
	*rac.cell = PageCellInit(0x09)
	zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 0x0A))

	rac2 := page.GetRowAndCell(1, 0)
	*rac2.cell = PageCellInit(0x09)
	zigTry(t, page.AppendGrapheme(rac2.row, rac2.cell, 0x0A))

	// Clear it
	page.ClearGrapheme(rac.cell)
	page.UpdateRowGraphemeFlag(rac.row)
	expect(t, rac.row.grapheme, "rac.row.grapheme")
	expect(t, !rac.cell.HasGrapheme(), "!rac.cell.hasGrapheme()")
	expect(t, rac2.cell.HasGrapheme(), "rac2.cell.hasGrapheme()")
}

// Zig: test "Page clone"
func TestPg1_Page_clone(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()

	// Write
	for y := range uint(page.capacity.rows) {
		rac := page.GetRowAndCell(1, y)
		*rac.cell = pg1Codepoint(uint32(y))
	}

	// Clone
	page2 := pg1Clone(t, page)
	defer page2.Deinit()
	expectEqual(t, page.capacity, page2.capacity)

	// Read it again
	for y := range uint(page2.capacity.rows) {
		rac := page2.GetRowAndCell(1, y)
		expectEqual(t, uint32(y), rac.cell.content().codepoint())
	}

	// Write again
	for y := range uint(page.capacity.rows) {
		rac := page.GetRowAndCell(1, y)
		*rac.cell = pg1Codepoint(0)
	}

	// Read it again, should be unchanged
	for y := range uint(page2.capacity.rows) {
		rac := page2.GetRowAndCell(1, y)
		expectEqual(t, uint32(y), rac.cell.content().codepoint())
	}

	// Read the original
	for y := range uint(page.capacity.rows) {
		rac := page.GetRowAndCell(1, y)
		expectEqual(t, uint32(0), rac.cell.content().codepoint())
	}
}

// Zig: test "Page clone graphemes"
func TestPg1_Page_clone_graphemes(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()

	// Append some graphemes
	{
		rac := page.GetRowAndCell(0, 0)
		*rac.cell = PageCellInit(0x09)
		zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 0x0A))
		zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 0x0B))
	}

	// Clone it
	page2 := pg1Clone(t, page)
	defer page2.Deinit()
	{
		rac := page2.GetRowAndCell(0, 0)
		expect(t, rac.row.grapheme, "rac.row.grapheme")
		expect(t, rac.cell.HasGrapheme(), "rac.cell.hasGrapheme()")
		expectEqualSlices(t, []uint32{0x0A, 0x0B}, zigUnwrap(t, page2.LookupGrapheme(rac.cell)))
	}
}

// Zig: test "Page clone styles"
func TestPg1_Page_clone_styles(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()

	// Write with some styles
	{
		id := pg1StylesAdd(t, page, Style{Flags: StyleFlags{Bold: true}})

		for x := range uint(page.size.cols) {
			rac := page.GetRowAndCell(x, 0)
			rac.row.styled = true
			*rac.cell = pageCellFields{
				content_tag: PageCellContentTagCodepoint,
				content:     pageCellContentFields{codepoint: uint32(x + 1)}.pack(),
				style_id:    id,
			}.pack()
			page.styles.Use(id)
		}
	}

	// Clone it
	page2 := pg1Clone(t, page)
	defer page2.Deinit()
	{
		id := func() uint16 {
			rac := page2.GetRowAndCell(0, 0)
			return rac.cell.StyleID()
		}()

		for x := range uint(page.size.cols) {
			rac := page.GetRowAndCell(x, 0)
			expect(t, rac.row.styled, "rac.row.styled")
			expectEqual(t, id, rac.cell.StyleID())
		}

		style := page.styles.Get(id)
		expect(t, Style{Flags: StyleFlags{Bold: true}}.Eql(*style), "(Style{ .flags = .{ .bold = true } }).eql(style.*)")
	}
}

// Zig: test "Page cloneFrom"
func TestPg1_Page_cloneFrom(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()

	// Write
	for y := range uint(page.capacity.rows) {
		rac := page.GetRowAndCell(1, y)
		*rac.cell = pg1Codepoint(uint32(y))
	}

	// Clone
	page2 := pg1Init(t, pg1Cap(10, 10, 8))
	defer page2.Deinit()
	zigTry(t, page2.CloneFrom(page, 0, uint(page.size.rows)))

	// Read it again
	for y := range uint(page2.capacity.rows) {
		rac := page2.GetRowAndCell(1, y)
		expectEqual(t, uint32(y), rac.cell.content().codepoint())
	}

	// Write again
	for y := range uint(page.capacity.rows) {
		rac := page.GetRowAndCell(1, y)
		*rac.cell = pg1Codepoint(0)
	}

	// Read it again, should be unchanged
	for y := range uint(page2.capacity.rows) {
		rac := page2.GetRowAndCell(1, y)
		expectEqual(t, uint32(y), rac.cell.content().codepoint())
	}

	// Read the original
	for y := range uint(page.capacity.rows) {
		rac := page.GetRowAndCell(1, y)
		expectEqual(t, uint32(0), rac.cell.content().codepoint())
	}
}

// Zig: test "Page cloneFrom shrink columns"
func TestPg1_Page_cloneFrom_shrink_columns(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()

	// Write
	for y := range uint(page.capacity.rows) {
		rac := page.GetRowAndCell(1, y)
		*rac.cell = pg1Codepoint(uint32(y))
	}

	// Clone
	page2 := pg1Init(t, pg1Cap(5, 10, 8))
	defer page2.Deinit()
	zigTry(t, page2.CloneFrom(page, 0, uint(page.size.rows)))
	expectEqual(t, SizeCellCountInt(5), page2.size.cols)

	// Read it again
	for y := range uint(page2.capacity.rows) {
		rac := page2.GetRowAndCell(1, y)
		expectEqual(t, uint32(y), rac.cell.content().codepoint())
	}
}

// Zig: test "Page cloneFrom partial"
func TestPg1_Page_cloneFrom_partial(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()

	// Write
	for y := range uint(page.capacity.rows) {
		rac := page.GetRowAndCell(1, y)
		*rac.cell = pg1Codepoint(uint32(y))
	}

	// Clone
	page2 := pg1Init(t, pg1Cap(10, 10, 8))
	defer page2.Deinit()
	zigTry(t, page2.CloneFrom(page, 0, 5))

	// Read it again
	for y := range uint(5) {
		rac := page2.GetRowAndCell(1, y)
		expectEqual(t, uint32(y), rac.cell.content().codepoint())
	}
	for y := uint(5); y < uint(page2.size.rows); y++ {
		rac := page2.GetRowAndCell(1, y)
		expectEqual(t, uint32(0), rac.cell.content().codepoint())
	}
}

// Zig: test "Page cloneFrom hyperlinks exact capacity"
func TestPg1_Page_cloneFrom_hyperlinks_exact_capacity(t *testing.T) {
	page := pg1Init(t, PageCapacityInit(50, 50))
	defer page.Deinit()

	// Ensure our page can accommodate the capacity.
	hyperlink_cap := page.HyperlinkCapacity()
	expect(t, hyperlink_cap <= uint(page.size.cols)*uint(page.size.rows), "hyperlink_cap <= page.size.cols * page.size.rows")

	// Create a hyperlink.
	hyperlink_id := pg1InsertHyperlink(t, page, Hyperlink{
		ID:  pg1Implicit(0),
		URI: []uint8("https://example.com"),
	})

	// Fill the exact cap with cells.
fill:
	for x := range uint(page.size.cols) {
		for y := range uint(page.size.rows) {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pg1Codepoint(42)
			zigTry(t, page.SetHyperlink(rac.row, rac.cell, hyperlink_id))
			page.hyperlink_set.Use(hyperlink_id)

			if page.HyperlinkCount() == hyperlink_cap {
				break fill
			}
		}
	}
	expectEqual(t, page.HyperlinkCount(), page.HyperlinkCapacity())

	// Clone the full page
	page2 := pg1Init(t, page.capacity)
	defer page2.Deinit()
	zigTry(t, page2.CloneFrom(page, 0, uint(page.size.rows)))

	// We should have the same number of hyperlinks
	expectEqual(t, page2.HyperlinkCount(), page.HyperlinkCount())
}

// Zig: test "Page cloneFrom graphemes"
func TestPg1_Page_cloneFrom_graphemes(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()

	// Write
	for y := range uint(page.capacity.rows) {
		rac := page.GetRowAndCell(1, y)
		*rac.cell = pg1Codepoint(uint32(y + 1))
		zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 0x0A))
	}

	// Clone
	page2 := pg1Init(t, pg1Cap(10, 10, 8))
	defer page2.Deinit()
	zigTry(t, page2.CloneFrom(page, 0, uint(page.size.rows)))

	// Read it again
	for y := range uint(page2.capacity.rows) {
		rac := page2.GetRowAndCell(1, y)
		expectEqual(t, uint32(y+1), rac.cell.content().codepoint())
		expect(t, rac.row.grapheme, "rac.row.grapheme")
		expect(t, rac.cell.HasGrapheme(), "rac.cell.hasGrapheme()")
		expectEqualSlices(t, []uint32{0x0A}, zigUnwrap(t, page2.LookupGrapheme(rac.cell)))
	}

	// Write again
	for y := range uint(page.capacity.rows) {
		rac := page.GetRowAndCell(1, y)
		page.ClearGrapheme(rac.cell)
		page.UpdateRowGraphemeFlag(rac.row)
		*rac.cell = pg1Codepoint(0)
	}

	// Read it again, should be unchanged
	for y := range uint(page2.capacity.rows) {
		rac := page2.GetRowAndCell(1, y)
		expectEqual(t, uint32(y+1), rac.cell.content().codepoint())
		expect(t, rac.row.grapheme, "rac.row.grapheme")
		expect(t, rac.cell.HasGrapheme(), "rac.cell.hasGrapheme()")
		expectEqualSlices(t, []uint32{0x0A}, zigUnwrap(t, page2.LookupGrapheme(rac.cell)))
	}

	// Read the original
	for y := range uint(page.capacity.rows) {
		rac := page.GetRowAndCell(1, y)
		expectEqual(t, uint32(0), rac.cell.content().codepoint())
	}
}

// Zig: test "Page cloneFrom frees dst graphemes"
func TestPg1_Page_cloneFrom_frees_dst_graphemes(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()
	for y := range uint(page.capacity.rows) {
		rac := page.GetRowAndCell(1, y)
		*rac.cell = pg1Codepoint(uint32(y + 1))
	}

	// Clone
	page2 := pg1Init(t, pg1Cap(10, 10, 8))
	defer page2.Deinit()
	for y := range uint(page2.capacity.rows) {
		rac := page2.GetRowAndCell(1, y)
		*rac.cell = pg1Codepoint(uint32(y + 1))
		zigTry(t, page2.AppendGrapheme(rac.row, rac.cell, 0x0A))
	}

	// Clone from page which has no graphemes.
	zigTry(t, page2.CloneFrom(page, 0, uint(page.size.rows)))

	// Read it again
	for y := range uint(page2.capacity.rows) {
		rac := page2.GetRowAndCell(1, y)
		expectEqual(t, uint32(y+1), rac.cell.content().codepoint())
		expect(t, !rac.row.grapheme, "!rac.row.grapheme")
		expect(t, !rac.cell.HasGrapheme(), "!rac.cell.hasGrapheme()")
	}
	expectEqual(t, uint(0), page2.GraphemeCount())
}

// Zig: test "Page cloneRowFrom partial"
func TestPg1_Page_cloneRowFrom_partial(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()

	// Write
	{
		const y = 0
		for x := range uint(page.size.cols) {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pg1Codepoint(uint32(x + 1))
		}
	}

	// Clone
	page2 := pg1Init(t, pg1Cap(10, 10, 8))
	defer page2.Deinit()
	zigTry(t, page2.ClonePartialRowFrom(
		page,
		page2.GetRow(0),
		page.GetRow(0),
		2,
		8,
	))

	// Read it again
	{
		const y = 0
		for x := range uint(page2.size.cols) {
			var expected uint32
			if x >= 2 && x < 8 {
				expected = uint32(x + 1)
			}
			rac := page2.GetRowAndCell(x, y)
			expectEqual(t, expected, rac.cell.content().codepoint())
		}
	}
}

// Zig: test "Page cloneRowFrom partial grapheme in non-copied source region"
func TestPg1_Page_cloneRowFrom_partial_grapheme_in_non_copied_source_region(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()

	// Write
	{
		const y = 0
		for x := range uint(page.size.cols) {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pg1Codepoint(uint32(x + 1))
		}
		{
			rac := page.GetRowAndCell(0, y)
			zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 0x0A))
		}
		{
			rac := page.GetRowAndCell(9, y)
			zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 0x0A))
		}
	}
	expectEqual(t, uint(2), page.GraphemeCount())

	// Clone
	page2 := pg1Init(t, pg1Cap(10, 10, 8))
	defer page2.Deinit()
	zigTry(t, page2.ClonePartialRowFrom(
		page,
		page2.GetRow(0),
		page.GetRow(0),
		2,
		8,
	))

	// Read it again
	{
		const y = 0
		for x := range uint(page2.size.cols) {
			var expected uint32
			if x >= 2 && x < 8 {
				expected = uint32(x + 1)
			}
			rac := page2.GetRowAndCell(x, y)
			expectEqual(t, expected, rac.cell.content().codepoint())
			expect(t, !rac.cell.HasGrapheme(), "!rac.cell.hasGrapheme()")
		}
		{
			rac := page2.GetRowAndCell(9, y)
			expect(t, !rac.row.grapheme, "!rac.row.grapheme")
		}
	}
	expectEqual(t, uint(0), page2.GraphemeCount())
}

// Zig: test "Page cloneRowFrom partial grapheme in non-copied dest region"
func TestPg1_Page_cloneRowFrom_partial_grapheme_in_non_copied_dest_region(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()

	// Write
	{
		const y = 0
		for x := range uint(page.size.cols) {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pg1Codepoint(uint32(x + 1))
		}
	}
	expectEqual(t, uint(0), page.GraphemeCount())

	// Clone
	page2 := pg1Init(t, pg1Cap(10, 10, 8))
	defer page2.Deinit()
	{
		const y = 0
		for x := range uint(page2.size.cols) {
			rac := page2.GetRowAndCell(x, y)
			*rac.cell = pg1Codepoint(0xBB)
		}
		{
			rac := page2.GetRowAndCell(0, y)
			zigTry(t, page2.AppendGrapheme(rac.row, rac.cell, 0x0A))
		}
		{
			rac := page2.GetRowAndCell(9, y)
			zigTry(t, page2.AppendGrapheme(rac.row, rac.cell, 0x0A))
		}
	}
	zigTry(t, page2.ClonePartialRowFrom(
		page,
		page2.GetRow(0),
		page.GetRow(0),
		2,
		8,
	))

	// Read it again
	{
		const y = 0
		for x := range uint(page2.size.cols) {
			var expected uint32 = 0xBB
			if x >= 2 && x < 8 {
				expected = uint32(x + 1)
			}
			rac := page2.GetRowAndCell(x, y)
			expectEqual(t, expected, rac.cell.content().codepoint())
		}
		{
			rac := page2.GetRowAndCell(9, y)
			expect(t, rac.row.grapheme, "rac.row.grapheme")
		}
	}
	expectEqual(t, uint(2), page2.GraphemeCount())
}

// Zig: test "Page cloneRowFrom partial hyperlink in same page copy"
func TestPg1_Page_cloneRowFrom_partial_hyperlink_in_same_page_copy(t *testing.T) {
	page := pg1Init(t, PageCapacityInit(10, 10))
	defer page.Deinit()

	// We need to create a hyperlink.
	hyperlink_id, err := page.hyperlink_set.AddContext(
		HyperlinkPageEntry{id: HyperlinkPageEntryId{Tag: HyperlinkPageEntryIdImplicit, Implicit: 0}},
		HyperlinkSetContext{page: page},
	)
	zigTry(t, err)

	// Write
	{
		const y = 0
		for x := range uint(page.size.cols) {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pg1Codepoint(uint32(x + 1))
		}

		// Hyperlink in a single cell
		{
			rac := page.GetRowAndCell(7, y)
			zigTry(t, page.SetHyperlink(rac.row, rac.cell, hyperlink_id))
		}
	}
	expectEqual(t, uint(1), page.HyperlinkCount())

	// Clone into the same page
	zigTry(t, page.ClonePartialRowFrom(
		page,
		page.GetRow(1),
		page.GetRow(0),
		2,
		8,
	))

	// Read it again
	{
		const y = 1
		for x := range uint(page.size.cols) {
			var expected uint32
			if x >= 2 && x < 8 {
				expected = uint32(x + 1)
			}
			rac := page.GetRowAndCell(x, y)
			expectEqual(t, expected, rac.cell.content().codepoint())
		}
		{
			rac := page.GetRowAndCell(7, y)
			expect(t, rac.row.hyperlink, "rac.row.hyperlink")
			expect(t, rac.cell.Hyperlink(), "rac.cell.hyperlink")
		}
	}
	expectEqual(t, uint(2), page.HyperlinkCount())
}

// Zig: test "Page cloneRowFrom partial hyperlink in same page omit"
func TestPg1_Page_cloneRowFrom_partial_hyperlink_in_same_page_omit(t *testing.T) {
	page := pg1Init(t, PageCapacityInit(10, 10))
	defer page.Deinit()

	// We need to create a hyperlink.
	hyperlink_id, err := page.hyperlink_set.AddContext(
		HyperlinkPageEntry{id: HyperlinkPageEntryId{Tag: HyperlinkPageEntryIdImplicit, Implicit: 0}},
		HyperlinkSetContext{page: page},
	)
	zigTry(t, err)

	// Write
	{
		const y = 0
		for x := range uint(page.size.cols) {
			rac := page.GetRowAndCell(x, y)
			*rac.cell = pg1Codepoint(uint32(x + 1))
		}

		// Hyperlink in a single cell
		{
			rac := page.GetRowAndCell(7, y)
			zigTry(t, page.SetHyperlink(rac.row, rac.cell, hyperlink_id))
		}
	}
	expectEqual(t, uint(1), page.HyperlinkCount())

	// Clone into the same page
	zigTry(t, page.ClonePartialRowFrom(
		page,
		page.GetRow(1),
		page.GetRow(0),
		2,
		6,
	))

	// Read it again
	{
		const y = 1
		for x := range uint(page.size.cols) {
			var expected uint32
			if x >= 2 && x < 6 {
				expected = uint32(x + 1)
			}
			rac := page.GetRowAndCell(x, y)
			expectEqual(t, expected, rac.cell.content().codepoint())
		}
		{
			rac := page.GetRowAndCell(7, y)
			expect(t, !rac.row.hyperlink, "!rac.row.hyperlink")
			expect(t, !rac.cell.Hyperlink(), "!rac.cell.hyperlink")
		}
	}
	expectEqual(t, uint(1), page.HyperlinkCount())
}

// Zig: test "Page moveCells text-only"
func TestPg1_Page_moveCells_text_only(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()

	// Write
	for x := range uint(page.capacity.cols) {
		rac := page.GetRowAndCell(x, 0)
		*rac.cell = pg1Codepoint(uint32(x + 1))
	}

	src := page.GetRow(0)
	dst := page.GetRow(1)
	page.MoveCells(src, 0, dst, 0, uint(page.capacity.cols))

	// New rows should have text
	for x := range uint(page.capacity.cols) {
		rac := page.GetRowAndCell(x, 1)
		expectEqual(t, uint32(x+1), rac.cell.content().codepoint())
	}

	// Old row should be blank
	for x := range uint(page.capacity.cols) {
		rac := page.GetRowAndCell(x, 0)
		expectEqual(t, uint32(0), rac.cell.content().codepoint())
	}
}

// Zig: test "Page moveCells graphemes"
func TestPg1_Page_moveCells_graphemes(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()

	// Write
	for x := range uint(page.size.cols) {
		rac := page.GetRowAndCell(x, 0)
		*rac.cell = pg1Codepoint(uint32(x + 1))
		zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 0x0A))
	}
	original_count := page.GraphemeCount()

	src := page.GetRow(0)
	dst := page.GetRow(1)
	page.MoveCells(src, 0, dst, 0, uint(page.size.cols))
	expectEqual(t, original_count, page.GraphemeCount())

	// New rows should have text
	for x := range uint(page.size.cols) {
		rac := page.GetRowAndCell(x, 1)
		expectEqual(t, uint32(x+1), rac.cell.content().codepoint())
		expectEqualSlices(t, []uint32{0x0A}, zigUnwrap(t, page.LookupGrapheme(rac.cell)))
	}

	// Old row should be blank
	for x := range uint(page.size.cols) {
		rac := page.GetRowAndCell(x, 0)
		expectEqual(t, uint32(0), rac.cell.content().codepoint())
	}
}

// Zig: test "Page verifyIntegrity graphemes good"
func TestPg1_Page_verifyIntegrity_graphemes_good(t *testing.T) {
	// Too slow, and not really necessary because the integrity tests are
	// only run in debug builds and unit tests verify they work well enough.
	// Zig-only: if (std.valgrind.runningOnValgrind() > 0) return error.SkipZigTest;

	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()

	// Write
	for x := range uint(page.size.cols) {
		rac := page.GetRowAndCell(x, 0)
		*rac.cell = pg1Codepoint(uint32(x + 1))
		zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 0x0A))
	}

	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "Page verifyIntegrity grapheme row not marked"
func TestPg1_Page_verifyIntegrity_grapheme_row_not_marked(t *testing.T) {
	// Too slow, and not really necessary because the integrity tests are
	// only run in debug builds and unit tests verify they work well enough.
	// Zig-only: if (std.valgrind.runningOnValgrind() > 0) return error.SkipZigTest;

	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()

	// Write
	for x := range uint(page.size.cols) {
		rac := page.GetRowAndCell(x, 0)
		*rac.cell = pg1Codepoint(uint32(x + 1))
		zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 0x0A))
	}

	// Make invalid by unmarking the row
	page.GetRow(0).grapheme = false

	pg1ExpectError(t, ErrUnmarkedGraphemeRow, page.VerifyIntegrity())
}

// Zig: test "Page verifyIntegrity styles good"
func TestPg1_Page_verifyIntegrity_styles_good(t *testing.T) {
	// Too slow, and not really necessary because the integrity tests are
	// only run in debug builds and unit tests verify they work well enough.
	// Zig-only: if (std.valgrind.runningOnValgrind() > 0) return error.SkipZigTest;

	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()

	// Upsert a style we'll use
	id := pg1StylesAdd(t, page, Style{Flags: StyleFlags{Bold: true}})

	// Write
	for x := range uint(page.size.cols) {
		rac := page.GetRowAndCell(x, 0)
		rac.row.styled = true
		*rac.cell = pageCellFields{
			content_tag: PageCellContentTagCodepoint,
			content:     pageCellContentFields{codepoint: uint32(x + 1)}.pack(),
			style_id:    id,
		}.pack()
		page.styles.Use(id)
	}

	// The original style add would have incremented the
	// ref count too, so release it to balance that out.
	page.styles.Release(id)

	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "Page verifyIntegrity styles ref count mismatch"
func TestPg1_Page_verifyIntegrity_styles_ref_count_mismatch(t *testing.T) {
	// Too slow, and not really necessary because the integrity tests are
	// only run in debug builds and unit tests verify they work well enough.
	// Zig-only: if (std.valgrind.runningOnValgrind() > 0) return error.SkipZigTest;

	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()

	// Upsert a style we'll use
	id := pg1StylesAdd(t, page, Style{Flags: StyleFlags{Bold: true}})

	// Write
	for x := range uint(page.size.cols) {
		rac := page.GetRowAndCell(x, 0)
		rac.row.styled = true
		*rac.cell = pageCellFields{
			content_tag: PageCellContentTagCodepoint,
			content:     pageCellContentFields{codepoint: uint32(x + 1)}.pack(),
			style_id:    id,
		}.pack()
		page.styles.Use(id)
	}

	// The original style add would have incremented the
	// ref count too, so release it to balance that out.
	page.styles.Release(id)

	// Miss a ref
	page.styles.Release(id)

	pg1ExpectError(t, ErrMismatchedStyleRef, page.VerifyIntegrity())
}

// Zig: test "Page verifyIntegrity zero rows"
func TestPg1_Page_verifyIntegrity_zero_rows(t *testing.T) {
	// Too slow, and not really necessary because the integrity tests are
	// only run in debug builds and unit tests verify they work well enough.
	// Zig-only: if (std.valgrind.runningOnValgrind() > 0) return error.SkipZigTest;

	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()
	page.size.rows = 0
	pg1ExpectError(t, ErrZeroRowCount, page.VerifyIntegrity())
}

// Zig: test "Page verifyIntegrity zero cols"
func TestPg1_Page_verifyIntegrity_zero_cols(t *testing.T) {
	// Too slow, and not really necessary because the integrity tests are
	// only run in debug builds and unit tests verify they work well enough.
	// Zig-only: if (std.valgrind.runningOnValgrind() > 0) return error.SkipZigTest;

	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()
	page.size.cols = 0
	pg1ExpectError(t, ErrZeroColCount, page.VerifyIntegrity())
}

// Zig: test "Page exactRowCapacity empty rows"
func TestPg1_Page_exactRowCapacity_empty_rows(t *testing.T) {
	cp := pg1Cap(10, 10, 8)
	cp.hyperlink_bytes = 32 * hyperlinkSetItemSizeOf
	cp.string_bytes = 512
	page := pg1Init(t, cp)
	defer page.Deinit()

	// Empty page: all capacity fields should be 0 (except cols/rows)
	got := page.ExactRowCapacity(0, 5)
	expectEqual(t, SizeCellCountInt(10), got.cols)
	expectEqual(t, SizeCellCountInt(5), got.rows)
	expectEqual(t, SizeStyleCountInt(0), got.styles)
	expectEqual(t, SizeGraphemeBytesInt(0), got.grapheme_bytes)
	expectEqual(t, SizeHyperlinkCountInt(0), got.hyperlink_bytes)
	expectEqual(t, SizeStringBytesInt(0), got.string_bytes)
}

// Zig: test "Page exactRowCapacity styles"
func TestPg1_Page_exactRowCapacity_styles(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()

	// No styles: capacity should be 0
	{
		got := page.ExactRowCapacity(0, 5)
		expectEqual(t, SizeStyleCountInt(0), got.styles)
	}

	// Add one style to a cell
	style1_id := pg1StylesAdd(t, page, Style{Flags: StyleFlags{Bold: true}})
	{
		rac := page.GetRowAndCell(0, 0)
		rac.row.styled = true
		rac.cell.setStyleID(style1_id)
	}

	// One unique style - capacity accounts for load factor
	cap_one_style := page.ExactRowCapacity(0, 5)
	{
		expectEqual(t, RefCountedSetCapacityForCount(1), uint(cap_one_style.styles))
	}

	// Add same style to another cell (duplicate) - capacity unchanged
	{
		rac := page.GetRowAndCell(1, 0)
		rac.cell.setStyleID(style1_id)
	}
	{
		got := page.ExactRowCapacity(0, 5)
		expectEqual(t, cap_one_style.styles, got.styles)
	}

	// Add a different style
	style2_id := pg1StylesAdd(t, page, Style{Flags: StyleFlags{Italic: true}})
	{
		rac := page.GetRowAndCell(2, 0)
		rac.cell.setStyleID(style2_id)
	}

	// Two unique styles - capacity accounts for load factor
	cap_two_styles := page.ExactRowCapacity(0, 5)
	{
		expectEqual(t, RefCountedSetCapacityForCount(2), uint(cap_two_styles.styles))
		expect(t, cap_two_styles.styles > cap_one_style.styles, "cap_two_styles.styles > cap_one_style.styles")
	}

	// Style outside the row range should not be counted
	{
		rac := page.GetRowAndCell(0, 7)
		rac.row.styled = true
		rac.cell.setStyleID(pg1StylesAdd(t, page, Style{Flags: StyleFlags{Underline: SGRAttributeUnderlineSingle}}))
	}
	{
		got := page.ExactRowCapacity(0, 5)
		expectEqual(t, cap_two_styles.styles, got.styles)
	}

	// Full range includes the new style
	{
		got := page.ExactRowCapacity(0, 10)
		expectEqual(t, RefCountedSetCapacityForCount(3), uint(got.styles))
	}

	// Verify clone works with exact capacity and produces same result
	{
		got := page.ExactRowCapacity(0, 5)
		cloned := pg1Init(t, got)
		defer cloned.Deinit()
		for y := range uint(5) {
			src_row := &page.rows[y]
			dst_row := &cloned.rows[y]
			zigTry(t, cloned.CloneRowFrom(page, dst_row, src_row))
		}
		cloned_cap := cloned.ExactRowCapacity(0, 5)
		expectEqual(t, got, cloned_cap)
	}
}

// Zig: test "Page exactRowCapacity single style clone"
func TestPg1_Page_exactRowCapacity_single_style_clone(t *testing.T) {
	// Regression test: verify a single style can be cloned with exact capacity.
	// This tests that capacityForCount properly accounts for ID 0 being reserved.
	page := pg1Init(t, pg1Cap(10, 2, 8))
	defer page.Deinit()

	// Add exactly one style to row 0
	style_id := pg1StylesAdd(t, page, Style{Flags: StyleFlags{Bold: true}})
	{
		rac := page.GetRowAndCell(0, 0)
		rac.row.styled = true
		rac.cell.setStyleID(style_id)
	}

	// exactRowCapacity for just row 0 should give capacity for 1 style
	got := page.ExactRowCapacity(0, 1)
	expectEqual(t, RefCountedSetCapacityForCount(1), uint(got.styles))

	// Create a new page with exact capacity and clone
	cloned := pg1Init(t, got)
	defer cloned.Deinit()

	src_row := &page.rows[0]
	dst_row := &cloned.rows[0]

	// This must not fail with StyleSetOutOfMemory
	zigTry(t, cloned.CloneRowFrom(page, dst_row, src_row))

	// Verify the style was cloned correctly
	cloned_cell := &cloned.GetCells(&cloned.rows[0])[0]
	expect(t, cloned_cell.StyleID() != StyleDefaultID, "cloned_cell.style_id != stylepkg.default_id")
}

// Zig: test "Page exactRowCapacity styles max single row"
func TestPg1_Page_exactRowCapacity_styles_max_single_row(t *testing.T) {
	cp := PageCapacityInit(math.MaxUint16, 1)
	cp.styles = math.MaxUint16
	page := pg1Init(t, cp)
	defer page.Deinit()

	// Style our first row
	row := &page.rows[0]
	row.styled = true

	// Fill cells with styles until we get OOM, but limit to a reasonable count
	// to avoid overflow when computing capacityForCount near maxInt
	cells := page.GetCells(row)[0:page.size.cols]
	count := uint(0)
	max_count := uint(1000) // Limit to avoid overflow in capacity calculation
	for i := range cells {
		cell := &cells[i]
		if count >= max_count {
			break
		}
		style_id, err := page.styles.Add(Style{
			FgColor: StyleColor{Tag: StyleColorRgb, Rgb: ColorRGB{
				R: uint8(i & 0xFF),
				G: uint8((i >> 8) & 0xFF),
				B: 0,
			}},
		})
		if err != nil {
			break
		}
		cell.setStyleID(style_id)
		count += 1
	}

	// Verify we added a meaningful number of styles
	expect(t, count > 0, "count > 0")

	// Capacity should be at least count (adjusted for load factor)
	got := page.ExactRowCapacity(0, 1)
	expectEqual(t, RefCountedSetCapacityForCount(count), uint(got.styles))
}

// Zig: test "Page exactRowCapacity grapheme_bytes"
func TestPg1_Page_exactRowCapacity_grapheme_bytes(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()

	// No graphemes: capacity should be 0
	{
		got := page.ExactRowCapacity(0, 5)
		expectEqual(t, SizeGraphemeBytesInt(0), got.grapheme_bytes)
	}

	// Add one grapheme (1 codepoint) to a cell - rounds up to grapheme_chunk
	{
		rac := page.GetRowAndCell(0, 0)
		*rac.cell = PageCellInit('a')
		zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 0x0301)) // combining acute accent
	}
	{
		got := page.ExactRowCapacity(0, 5)
		// 1 codepoint = 4 bytes, rounds up to grapheme_chunk (16)
		expectEqual(t, SizeGraphemeBytesInt(pageGraphemeChunk), got.grapheme_bytes)
	}

	// Add another grapheme to a different cell - should sum
	{
		rac := page.GetRowAndCell(1, 0)
		*rac.cell = PageCellInit('e')
		zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 0x0300)) // combining grave accent
	}
	{
		got := page.ExactRowCapacity(0, 5)
		// 2 graphemes, each 1 codepoint = 2 * grapheme_chunk
		expectEqual(t, SizeGraphemeBytesInt(pageGraphemeChunk*2), got.grapheme_bytes)
	}

	// Add a larger grapheme (multiple codepoints) that fits in one chunk
	{
		rac := page.GetRowAndCell(2, 0)
		*rac.cell = PageCellInit('o')
		zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 0x0301))
		zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 0x0302))
		zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 0x0303))
	}
	{
		got := page.ExactRowCapacity(0, 5)
		// First two cells: 2 * grapheme_chunk
		// Third cell: 3 codepoints = 12 bytes, rounds up to grapheme_chunk
		expectEqual(t, SizeGraphemeBytesInt(pageGraphemeChunk*3), got.grapheme_bytes)
	}

	// Grapheme outside the row range should not be counted
	{
		rac := page.GetRowAndCell(0, 7)
		*rac.cell = PageCellInit('x')
		zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 0x0304))
	}
	{
		got := page.ExactRowCapacity(0, 5)
		expectEqual(t, SizeGraphemeBytesInt(pageGraphemeChunk*3), got.grapheme_bytes)
	}

	// Full range includes the new grapheme
	{
		got := page.ExactRowCapacity(0, 10)
		expectEqual(t, SizeGraphemeBytesInt(pageGraphemeChunk*4), got.grapheme_bytes)
	}

	// Verify clone works with exact capacity and produces same result
	{
		got := page.ExactRowCapacity(0, 5)
		cloned := pg1Init(t, got)
		defer cloned.Deinit()
		for y := range uint(5) {
			src_row := &page.rows[y]
			dst_row := &cloned.rows[y]
			zigTry(t, cloned.CloneRowFrom(page, dst_row, src_row))
		}
		cloned_cap := cloned.ExactRowCapacity(0, 5)
		expectEqual(t, got, cloned_cap)
	}
}

// Zig: test "Page exactRowCapacity grapheme_bytes larger than chunk"
func TestPg1_Page_exactRowCapacity_grapheme_bytes_larger_than_chunk(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()

	// Add a grapheme larger than one chunk (grapheme_chunk_len = 4 codepoints)
	rac := page.GetRowAndCell(0, 0)
	*rac.cell = PageCellInit('a')

	// Add 6 codepoints - requires 2 chunks (6 * 4 = 24 bytes, rounds up to 32)
	for i := range uint(6) {
		zigTry(t, page.AppendGrapheme(rac.row, rac.cell, uint32(0x0300+i)))
	}

	got := page.ExactRowCapacity(0, 1)
	// 6 codepoints = 24 bytes, alignForward(24, 16) = 32
	expectEqual(t, SizeGraphemeBytesInt(32), got.grapheme_bytes)

	// Verify clone works with exact capacity and produces same result
	cloned := pg1Init(t, got)
	defer cloned.Deinit()
	src_row := &page.rows[0]
	dst_row := &cloned.rows[0]
	zigTry(t, cloned.CloneRowFrom(page, dst_row, src_row))
	cloned_cap := cloned.ExactRowCapacity(0, 1)
	expectEqual(t, got, cloned_cap)
}

// Zig: test "Page exactRowCapacity hyperlinks"
func TestPg1_Page_exactRowCapacity_hyperlinks(t *testing.T) {
	cp := pg1Cap(10, 10, 8)
	cp.hyperlink_bytes = 32 * hyperlinkSetItemSizeOf
	cp.string_bytes = 512
	page := pg1Init(t, cp)
	defer page.Deinit()

	// No hyperlinks: capacity should be 0
	{
		got := page.ExactRowCapacity(0, 5)
		expectEqual(t, SizeHyperlinkCountInt(0), got.hyperlink_bytes)
		expectEqual(t, SizeStringBytesInt(0), got.string_bytes)
	}

	// Add one hyperlink with implicit ID
	const uri1 = "https://example.com"
	id1 := func() HyperlinkID {
		rac := page.GetRowAndCell(0, 0)

		// Create and add hyperlink entry
		id := pg1InsertHyperlink(t, page, Hyperlink{
			ID:  pg1Implicit(1),
			URI: []uint8(uri1),
		})
		zigTry(t, page.SetHyperlink(rac.row, rac.cell, id))
		return id
	}()
	// 1 hyperlink - capacity accounts for load factor
	cap_one_link := page.ExactRowCapacity(0, 5)
	{
		expectEqual(t, RefCountedSetCapacityForCount(1)*hyperlinkSetItemSizeOf, uint(cap_one_link.hyperlink_bytes))
		// URI "https://example.com" = 19 bytes, rounds up to string_chunk (32)
		expectEqual(t, SizeStringBytesInt(pageStringChunk), cap_one_link.string_bytes)
	}

	// Add same hyperlink to another cell (duplicate ID) - capacity unchanged
	{
		rac := page.GetRowAndCell(1, 0)

		// Use the same hyperlink ID for another cell
		page.hyperlink_set.Use(id1)
		zigTry(t, page.SetHyperlink(rac.row, rac.cell, id1))
	}
	{
		got := page.ExactRowCapacity(0, 5)
		expectEqual(t, cap_one_link.hyperlink_bytes, got.hyperlink_bytes)
		expectEqual(t, cap_one_link.string_bytes, got.string_bytes)
	}

	// Add a different hyperlink with explicit ID
	const uri2 = "https://other.example.org/path"
	const explicit_id = "my-link-id"
	{
		rac := page.GetRowAndCell(2, 0)

		id := pg1InsertHyperlink(t, page, Hyperlink{
			ID:  pg1Explicit(explicit_id),
			URI: []uint8(uri2),
		})
		zigTry(t, page.SetHyperlink(rac.row, rac.cell, id))
	}
	// 2 hyperlinks - capacity accounts for load factor
	cap_two_links := page.ExactRowCapacity(0, 5)
	{
		expectEqual(t, RefCountedSetCapacityForCount(2)*hyperlinkSetItemSizeOf, uint(cap_two_links.hyperlink_bytes))
		// First URI: 19 bytes -> 32, Second URI: 30 bytes -> 32, Explicit ID: 10 bytes -> 32
		expectEqual(t, SizeStringBytesInt(pageStringChunk*3), cap_two_links.string_bytes)
	}

	// Hyperlink outside the row range should not be counted
	{
		rac := page.GetRowAndCell(0, 7) // row 7 is outside range [0, 5)

		id := pg1InsertHyperlink(t, page, Hyperlink{
			ID:  pg1Implicit(99),
			URI: []uint8("https://outside.example.com"),
		})
		zigTry(t, page.SetHyperlink(rac.row, rac.cell, id))
	}
	{
		got := page.ExactRowCapacity(0, 5)
		expectEqual(t, cap_two_links.hyperlink_bytes, got.hyperlink_bytes)
		expectEqual(t, cap_two_links.string_bytes, got.string_bytes)
	}

	// Full range includes the new hyperlink
	{
		got := page.ExactRowCapacity(0, 10)
		expectEqual(t, RefCountedSetCapacityForCount(3)*hyperlinkSetItemSizeOf, uint(got.hyperlink_bytes))
		// Third URI: 27 bytes -> 32
		expectEqual(t, SizeStringBytesInt(pageStringChunk*4), got.string_bytes)
	}

	// Verify clone works with exact capacity and produces same result
	{
		got := page.ExactRowCapacity(0, 5)
		cloned := pg1Init(t, got)
		defer cloned.Deinit()
		for y := range uint(5) {
			src_row := &page.rows[y]
			dst_row := &cloned.rows[y]
			zigTry(t, cloned.CloneRowFrom(page, dst_row, src_row))
		}
		cloned_cap := cloned.ExactRowCapacity(0, 5)
		expectEqual(t, got, cloned_cap)
	}
}

// Zig: test "Page exactRowCapacity single hyperlink clone"
func TestPg1_Page_exactRowCapacity_single_hyperlink_clone(t *testing.T) {
	// Regression test: verify a single hyperlink can be cloned with exact capacity.
	// This tests that capacityForCount properly accounts for ID 0 being reserved.
	cp := pg1Cap(10, 2, 8)
	cp.hyperlink_bytes = 32 * hyperlinkSetItemSizeOf
	cp.string_bytes = 512
	page := pg1Init(t, cp)
	defer page.Deinit()

	// Add exactly one hyperlink to row 0
	const uri = "https://example.com"
	{
		rac := page.GetRowAndCell(0, 0)
		link_id := pg1InsertHyperlink(t, page, Hyperlink{
			ID:  pg1Implicit(1),
			URI: []uint8(uri),
		})
		zigTry(t, page.SetHyperlink(rac.row, rac.cell, link_id))
	}

	// exactRowCapacity for just row 0 should give capacity for 1 hyperlink
	got := page.ExactRowCapacity(0, 1)
	expectEqual(t, RefCountedSetCapacityForCount(1)*hyperlinkSetItemSizeOf, uint(got.hyperlink_bytes))

	// Create a new page with exact capacity and clone
	cloned := pg1Init(t, got)
	defer cloned.Deinit()

	src_row := &page.rows[0]
	dst_row := &cloned.rows[0]

	// This must not fail with HyperlinkSetOutOfMemory
	zigTry(t, cloned.CloneRowFrom(page, dst_row, src_row))

	// Verify the hyperlink was cloned correctly
	cloned_cell := &cloned.GetCells(&cloned.rows[0])[0]
	expect(t, cloned_cell.Hyperlink(), "cloned_cell.hyperlink")
}

// Zig: test "Page exactRowCapacity hyperlink map capacity for many cells"
func TestPg1_Page_exactRowCapacity_hyperlink_map_capacity_for_many_cells(t *testing.T) {
	// A single hyperlink spanning many cells requires hyperlink_map capacity
	// based on cell count, not unique hyperlink count.
	const cols = 50
	cp := pg1Cap(cols, 2, 8)
	cp.hyperlink_bytes = 32 * hyperlinkSetItemSizeOf
	cp.string_bytes = 512
	page := pg1Init(t, cp)
	defer page.Deinit()

	// Add one hyperlink spanning all 50 columns in row 0
	const uri = "https://example.com"
	id := func() HyperlinkID {
		rac := page.GetRowAndCell(0, 0)
		link_id := pg1InsertHyperlink(t, page, Hyperlink{
			ID:  pg1Implicit(1),
			URI: []uint8(uri),
		})
		zigTry(t, page.SetHyperlink(rac.row, rac.cell, link_id))
		return link_id
	}()

	// Apply same hyperlink to remaining cells in row 0
	for x := uint(1); x < cols; x++ {
		rac := page.GetRowAndCell(x, 0)
		page.hyperlink_set.Use(id)
		zigTry(t, page.SetHyperlink(rac.row, rac.cell, id))
	}

	// exactRowCapacity must account for 50 hyperlink cells, not just 1 unique hyperlink
	got := page.ExactRowCapacity(0, 1)

	// The hyperlink_bytes must be large enough that layout() computes sufficient
	// hyperlink_map capacity. With hyperlink_cell_multiplier=16, we need at least
	// ceil(50/16) = 4 hyperlink entries worth of bytes for the map.
	min_for_map := uint((cols + pageHyperlinkCellMultiplier - 1) / pageHyperlinkCellMultiplier) // std.math.divCeil(usize, cols, hyperlink_cell_multiplier) catch 0
	min_hyperlink_bytes := min_for_map * hyperlinkSetItemSizeOf
	expect(t, uint(got.hyperlink_bytes) >= min_hyperlink_bytes, "cap.hyperlink_bytes >= min_hyperlink_bytes")

	// Create a new page with exact capacity and clone - must not fail
	cloned := pg1Init(t, got)
	defer cloned.Deinit()

	src_row := &page.rows[0]
	dst_row := &cloned.rows[0]

	// This must not fail with HyperlinkMapOutOfMemory
	zigTry(t, cloned.CloneRowFrom(page, dst_row, src_row))

	// Verify all hyperlinks were cloned correctly
	for x := range uint(cols) {
		cloned_cell := &cloned.GetCells(&cloned.rows[0])[x]
		expect(t, cloned_cell.Hyperlink(), "cloned_cell.hyperlink")
	}
}

// ---------------------------------------------------------------------------
// term51: additional tests (H). Not part of upstream Ghostty.
// ---------------------------------------------------------------------------

// pg1HItemSize is @sizeOf(term51HItem) = @sizeOf(hyperlink.Set.Item).
const pg1HItemSize = hyperlinkSetItemSizeOf

// pg1HBits is term51HBits.
func pg1HBits(cell PageCell) uint64 {
	return cell.bitCast()
}

// pg1HNew is term51HNew: a page with room for 5 distinct styles and 5
// distinct hyperlinks.
func pg1HNew(t *testing.T, cols SizeCellCountInt, rows SizeCellCountInt) *Page {
	t.Helper()
	cp := pg1Cap(cols, rows, 8)
	cp.hyperlink_bytes = 8 * pg1HItemSize
	return pg1Init(t, cp)
}

// pg1HFillRow is term51HFillRow: write the codepoint `base + x` into every
// cell of row y.
func pg1HFillRow(page *Page, y uint, base uint32) {
	for x := range uint(page.size.cols) {
		*page.GetRowAndCell(x, y).cell = PageCellInit(base + uint32(x))
	}
}

// pg1HGrapheme is term51HGrapheme: write a cell with `cp` and the extra
// grapheme codepoints.
func pg1HGrapheme(t *testing.T, page *Page, x uint, y uint, cp uint32, extra []uint32) {
	t.Helper()
	rac := page.GetRowAndCell(x, y)
	*rac.cell = PageCellInit(cp)
	for _, e := range extra {
		zigTry(t, page.AppendGrapheme(rac.row, rac.cell, e))
	}
}

// pg1HStyleCells is term51HStyleCells: add a style (distinguished by palette
// fg n) and put it on the cells. After this the style's reference count is
// exactly len(cells).
func pg1HStyleCells(t *testing.T, page *Page, n uint8, cells [][2]uint) StyleID {
	t.Helper()
	id := pg1StylesAdd(t, page, Style{FgColor: StyleColor{Tag: StyleColorPalette, Palette: n}})
	for _, c := range cells {
		rac := page.GetRowAndCell(c[0], c[1])
		rac.row.styled = true
		rac.cell.setStyleID(id)
		page.styles.Use(id)
	}
	page.styles.Release(id)
	return id
}

// pg1HLinkCells is term51HLinkCells: insert a hyperlink and put it on the
// cells. After this the link's reference count is exactly len(cells).
func pg1HLinkCells(t *testing.T, page *Page, link Hyperlink, cells [][2]uint) HyperlinkID {
	t.Helper()
	id := pg1InsertHyperlink(t, page, link)
	for _, c := range cells {
		rac := page.GetRowAndCell(c[0], c[1])
		page.hyperlink_set.Use(id)
		zigTry(t, page.SetHyperlink(rac.row, rac.cell, id))
	}
	page.hyperlink_set.Release(id)
	return id
}

// pg1HLink is term51HLink.
func pg1HLink(uri string) Hyperlink {
	return Hyperlink{ID: pg1Implicit(0), URI: []uint8(uri)}
}

// pg1HExpectRowsEqual is term51HExpectRowsEqual.
func pg1HExpectRowsEqual(t *testing.T, a *Page, b *Page, y uint) {
	t.Helper()
	ra := a.GetRow(y)
	rb := b.GetRow(y)
	for x := range uint(a.size.cols) {
		expectEqual(t, pg1HBits(a.GetCells(ra)[x]), pg1HBits(b.GetCells(rb)[x]))
	}
}

// pg1HFillAll is term51HFillAll: fill every row of a page with distinct
// text.
func pg1HFillAll(page *Page) {
	for y := range uint(page.size.rows) {
		pg1HFillRow(page, y, uint32(0x100+y*0x20))
	}
}

// pg1HExpectIntegrity is term51HExpectIntegrity.
func pg1HExpectIntegrity(t *testing.T, page *Page, expected error) {
	t.Helper()
	pg1ExpectError(t, expected, page.VerifyIntegrity())
}

// pg1HPopulate is term51HPopulate: populate a page with every kind of
// managed content.
func pg1HPopulate(t *testing.T, page *Page) {
	t.Helper()
	pg1HFillAll(page)
	pg1HGrapheme(t, page, 1, 0, 'e', []uint32{0x301, 0x302})
	pg1HGrapheme(t, page, 4, 2, 'f', []uint32{0x303, 0x304, 0x305, 0x306, 0x307})
	pg1HStyleCells(t, page, 7, [][2]uint{{2, 1}, {3, 1}})
	pg1HStyleCells(t, page, 9, [][2]uint{{0, 3}})
	pg1HLinkCells(t, page, pg1HLink("https://one.example/"), [][2]uint{{5, 4}, {6, 4}})
	pg1HLinkCells(t, page, Hyperlink{ID: pg1Explicit("my-id"), URI: []uint8("https://two.example/")}, [][2]uint{{7, 5}})
	page.GetRowAndCell(8, 1).cell.setWide(PageCellWideWide)
	page.GetRowAndCell(9, 1).cell.setWide(PageCellWideSpacerTail)
	page.GetRow(2).wrap = true
	page.GetRow(3).wrap_continuation = true
	page.GetRow(3).semantic_prompt = PageRowSemanticPromptPrompt
}

// ---- Page.init ------------------------------------------------------------

// Zig: test "term51: Page.init: dimensions, capacity and empty state"
func TestPg1_term51_Page_init_dimensions_capacity_and_empty_state(t *testing.T) {
	cp := PageCapacity{
		cols:            12,
		rows:            7,
		styles:          8,
		hyperlink_bytes: 8 * pg1HItemSize,
		grapheme_bytes:  1024,
		string_bytes:    2048,
	}
	page := pg1Init(t, cp)
	defer page.Deinit()

	expectEqual(t, SizeCellCountInt(12), page.size.cols)
	expectEqual(t, SizeCellCountInt(7), page.size.rows)
	expectEqual(t, cp, page.capacity)
	expect(t, !page.dirty, "!page.dirty")
	expect(t, !page.IsDirty(), "!page.isDirty()")
	expectEqual(t, uint(0), page.GraphemeCount())
	expectEqual(t, uint(0), page.HyperlinkCount())
	expectEqual(t, uint(0), page.styles.Count())
	expectEqual(t, uint(0), page.hyperlink_set.Count())
	expectEqual(t, uint64(0), page.string_alloc.UsedBytes())
	expectEqual(t, uint64(0), page.grapheme_alloc.UsedBytes())

	cells_base := page.cells
	for y := range uint(7) {
		row := page.GetRow(y)
		// Every row points at its own run of 12 cells, and has no flags.
		expect(t, &cells_base[y*12] == &page.GetCells(row)[0], "&cells_base[y * 12] == &row.cells.ptr(page.memory)[0]")
		expectEqual(t, PageRow{cells: row.cells}, *row)
		for _, cell := range page.GetCells(row) {
			expect(t, cell.IsZero(), "cell.isZero()")
		}
	}
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.init: minimum 1x1 page"
func TestPg1_term51_Page_init_minimum_1x1_page(t *testing.T) {
	page := pg1HNew(t, 1, 1)
	defer page.Deinit()
	expectEqual(t, SizeCellCountInt(1), page.size.cols)
	expectEqual(t, SizeCellCountInt(1), page.size.rows)
	rac := page.GetRowAndCell(0, 0)
	expect(t, rac.cell.IsZero(), "rac.cell.isZero()")
	expectEqual(t, rac.cell, &page.GetCells(rac.row)[0])
	expectEqual(t, uint(1), uint(len(page.GetCells(rac.row))))
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.init: maximum column count"
func TestPg1_term51_Page_init_maximum_column_count(t *testing.T) {
	page := pg1Init(t, pg1Cap(65535, 1, 8))
	defer page.Deinit()
	expectEqual(t, SizeCellCountInt(65535), page.size.cols)
	row := page.GetRow(0)
	cells := page.GetCells(row)
	expectEqual(t, uint(65535), uint(len(cells)))
	expect(t, cells[0].IsZero(), "cells[0].isZero()")
	expect(t, cells[65534].IsZero(), "cells[65534].isZero()")
	cells[65534] = PageCellInit('z')
	expectEqual(t, uint32('z'), page.GetRowAndCell(65534, 0).cell.Codepoint())
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.init: maximum row count"
func TestPg1_term51_Page_init_maximum_row_count(t *testing.T) {
	page := pg1Init(t, pg1Cap(1, 65535, 8))
	defer page.Deinit()
	expectEqual(t, SizeCellCountInt(65535), page.size.rows)
	cells_base := page.cells
	// First and last rows each own exactly one distinct cell.
	expectEqual(t, &cells_base[0], &page.GetCells(page.GetRow(0))[0])
	expectEqual(t, &cells_base[65534], &page.GetCells(page.GetRow(65534))[0])
	*page.GetRowAndCell(0, 65534).cell = PageCellInit('q')
	expectEqual(t, uint32('q'), page.GetRowAndCell(0, 65534).cell.Codepoint())
	expectEqual(t, uint32(0), page.GetRowAndCell(0, 65533).cell.Codepoint())
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.init: zero rows is constructible but fails integrity"
func TestPg1_term51_Page_init_zero_rows_is_constructible_but_fails_integrity(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 0, 8))
	defer page.Deinit()
	expectEqual(t, SizeCellCountInt(0), page.size.rows)
	expectEqual(t, SizeCellCountInt(10), page.size.cols)
	pg1ExpectError(t, ErrZeroRowCount, page.VerifyIntegrity())

	// Zero rows and zero columns: the row check wins.
	page2 := pg1Init(t, pg1Cap(0, 0, 8))
	defer page2.Deinit()
	pg1ExpectError(t, ErrZeroRowCount, page2.VerifyIntegrity())
}

// Zig: test "term51: Page.init: zero-sized optional capacities"
func TestPg1_term51_Page_init_zero_sized_optional_capacities(t *testing.T) {
	page := pg1Init(t, PageCapacity{
		cols:            4,
		rows:            4,
		styles:          0,
		hyperlink_bytes: 0,
		grapheme_bytes:  0,
		string_bytes:    0,
	})
	defer page.Deinit()
	expectEqual(t, uint(0), page.GraphemeCapacity())
	expectEqual(t, uint(0), page.GraphemeCount())
	expectEqual(t, uint(0), page.HyperlinkCapacity())
	expectEqual(t, uint(0), page.HyperlinkCount())
	expectEqual(t, uint(0), page.styles.layout.cap)
	expectEqual(t, uint(0), page.hyperlink_set.layout.cap)
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.init: layout offsets and memory size"
func TestPg1_term51_Page_init_layout_offsets_and_memory_size(t *testing.T) {
	cp := pg1Cap(9, 5, 8)
	l := PageLayoutOf(cp)
	page := pg1Init(t, cp)
	defer page.Deinit()

	// Zig-only: try term51Ht.expectEqual(l.total_size, page.memory.len);
	// The next Zig line is on page.memory.len, which equals l.total_size; it
	// is checked on the layout's total_size.
	// The Zig line: expectEqual(@as(usize, 0), page.memory.len % std.heap.page_size_min).
	expectEqual(t, uint64(0), l.total_size%pg1PageSizeMin)
	// Zig-only: try term51Ht.expect(std.mem.isAligned(@intFromPtr(page.memory.ptr), std.heap.page_size_min));
	// Zig-only: try term51Ht.expectEqual(@as(size.OffsetInt, @intCast(l.rows_start)), page.rows.offset);
	// Zig-only: try term51Ht.expectEqual(@as(size.OffsetInt, @intCast(l.cells_start)), page.cells.offset);
	expectEqual(t, uint64(9*5*pg1SizeOfCell), l.cells_size)
	expectEqual(t, uint64(5*pg1SizeOfRow), l.rows_size)
	// Cells are zeroed. Zig reads the cells as bytes, all of which must be
	// zero; each cell is one u64.
	for _, c := range page.cells[0 : 9*5] {
		expectEqual(t, uint64(0), c.bitCast())
	}
}

// Zig: test "term51: Page.init: independent pages do not share memory"
func TestPg1_term51_Page_init_independent_pages_do_not_share_memory(t *testing.T) {
	a := pg1HNew(t, 4, 4)
	defer a.Deinit()
	b := pg1HNew(t, 4, 4)
	defer b.Deinit()
	// Zig-only: try term51Ht.expect(a.memory.ptr != b.memory.ptr);
	*a.GetRowAndCell(1, 1).cell = PageCellInit('a')
	expect(t, b.GetRowAndCell(1, 1).cell.IsZero(), "b.getRowAndCell(1, 1).cell.isZero()")
}

// ---- Page.initBuf ---------------------------------------------------------
// The four Page.initBuf tests are not translated (no memory base in Go); see
// scripts/not-translated.txt.

// ---- Page.deinit ----------------------------------------------------------

// Zig: test "term51: Page.deinit: pages of many shapes can be created and released"
func TestPg1_term51_Page_deinit_pages_of_many_shapes_can_be_created_and_released(t *testing.T) {
	shapes := [][2]SizeCellCountInt{
		{1, 1}, {80, 24}, {215, 215}, {1, 300}, {300, 1},
	}
	for _, s := range shapes {
		page := pg1HNew(t, s[0], s[1])
		*page.GetRowAndCell(0, 0).cell = PageCellInit('x')
		expectEqual(t, uint32('x'), page.GetRowAndCell(0, 0).cell.Codepoint())
		page.Deinit()
	}
}

// ---- Page.reinit ----------------------------------------------------------

// Zig: test "term51: Page.reinit: wipes all content and restores the initial state"
func TestPg1_term51_Page_reinit_wipes_all_content_and_restores_the_initial_state(t *testing.T) {
	page := pg1HNew(t, 10, 6)
	defer page.Deinit()
	// Zig-only: const mem_ptr = page.memory.ptr;
	// Zig-only: const mem_len = page.memory.len;
	cp := page.capacity

	pg1HFillAll(page)
	pg1HGrapheme(t, page, 1, 1, 'e', []uint32{0x301, 0x302})
	pg1HStyleCells(t, page, 4, [][2]uint{{2, 2}, {3, 2}})
	pg1HLinkCells(t, page, pg1HLink("https://example.com"), [][2]uint{{4, 3}})
	page.GetRow(5).dirty = true
	page.GetRow(0).wrap = true
	page.dirty = true
	page.size.rows = 3
	page.size.cols = 8

	page.Reinit()

	// Zig-only: try term51Ht.expectEqual(mem_ptr, page.memory.ptr);
	// Zig-only: try term51Ht.expectEqual(mem_len, page.memory.len);
	expectEqual(t, cp, page.capacity)
	expectEqual(t, SizeCellCountInt(10), page.size.cols)
	expectEqual(t, SizeCellCountInt(6), page.size.rows)
	expect(t, !page.dirty, "!page.dirty")
	expect(t, !page.IsDirty(), "!page.isDirty()")
	expectEqual(t, uint(0), page.GraphemeCount())
	expectEqual(t, uint(0), page.HyperlinkCount())
	expectEqual(t, uint(0), page.styles.Count())
	expectEqual(t, uint(0), page.hyperlink_set.Count())
	expectEqual(t, uint64(0), page.grapheme_alloc.UsedBytes())
	expectEqual(t, uint64(0), page.string_alloc.UsedBytes())
	for y := range uint(6) {
		row := page.GetRow(y)
		expect(t, &page.cells[y*10] == &page.GetCells(row)[0], "&page.cells.ptr(page.memory)[y * 10] == &row.cells.ptr(page.memory)[0]")
		expectEqual(t, PageRow{cells: row.cells}, *row)
		for _, c := range page.GetCells(row) {
			expect(t, c.IsZero(), "c.isZero()")
		}
	}
	zigTry(t, page.VerifyIntegrity())

	// And the page is reusable, including the styles/hyperlinks tables.
	sid := pg1HStyleCells(t, page, 4, [][2]uint{{0, 0}})
	expectEqual(t, StyleID(1), sid)
	hid := pg1HLinkCells(t, page, pg1HLink("https://example.com"), [][2]uint{{1, 0}})
	expectEqual(t, HyperlinkID(1), hid)
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.reinit: on a fresh page changes nothing observable"
func TestPg1_term51_Page_reinit_on_a_fresh_page_changes_nothing_observable(t *testing.T) {
	page := pg1HNew(t, 5, 5)
	defer page.Deinit()
	page.Reinit()
	page.Reinit()
	expectEqual(t, SizeCellCountInt(5), page.size.cols)
	expectEqual(t, SizeCellCountInt(5), page.size.rows)
	expect(t, !page.IsDirty(), "!page.isDirty()")
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.reinit: resets the integrity pause counter"
func TestPg1_term51_Page_reinit_resets_the_integrity_pause_counter(t *testing.T) {
	page := pg1HNew(t, 3, 3)
	defer page.Deinit()
	page.PauseIntegrityChecks(true)
	page.PauseIntegrityChecks(true)
	page.Reinit()
	if pageSlowRuntimeSafety {
		expectEqual(t, uint64(0), page.pause_integrity_checks)
	}
	// Checks are active again.
	page.size.rows = 0
	pg1ExpectError(t, ErrZeroRowCount, page.VerifyIntegrity())
}

// ---- Page.pauseIntegrityChecks / Page.assertIntegrity ---------------------

// Zig: test "term51: Page.pauseIntegrityChecks: nests as a counter"
func TestPg1_term51_Page_pauseIntegrityChecks_nests_as_a_counter(t *testing.T) {
	page := pg1HNew(t, 3, 3)
	defer page.Deinit()

	if pageSlowRuntimeSafety {
		expectEqual(t, uint64(0), page.pause_integrity_checks)
		page.PauseIntegrityChecks(true)
		expectEqual(t, uint64(1), page.pause_integrity_checks)
		page.PauseIntegrityChecks(true)
		expectEqual(t, uint64(2), page.pause_integrity_checks)
		page.PauseIntegrityChecks(false)
		expectEqual(t, uint64(1), page.pause_integrity_checks)
		page.PauseIntegrityChecks(false)
		expectEqual(t, uint64(0), page.pause_integrity_checks)
	} else {
		// Without runtime safety the call is a no-op.
		page.PauseIntegrityChecks(true)
		page.PauseIntegrityChecks(false)
		page.PauseIntegrityChecks(false)
	}
}

// Zig: test "term51: Page.pauseIntegrityChecks: suppresses verifyIntegrity only while paused"
func TestPg1_term51_Page_pauseIntegrityChecks_suppresses_verifyIntegrity_only_while_paused(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	page.GetRowAndCell(1, 1).cell.setContentTag(PageCellContentTagCodepointGrapheme) // corrupt

	pg1ExpectError(t, ErrMissingGraphemeData, page.VerifyIntegrity())

	if pageSlowRuntimeSafety {
		page.PauseIntegrityChecks(true)
		zigTry(t, page.VerifyIntegrity())
		page.PauseIntegrityChecks(true)
		zigTry(t, page.VerifyIntegrity())
		// One unpause of two: still paused.
		page.PauseIntegrityChecks(false)
		zigTry(t, page.VerifyIntegrity())
		// Fully unpaused: the corruption is reported again.
		page.PauseIntegrityChecks(false)
		pg1ExpectError(t, ErrMissingGraphemeData, page.VerifyIntegrity())
	} else {
		page.PauseIntegrityChecks(true)
		pg1ExpectError(t, ErrMissingGraphemeData, page.VerifyIntegrity())
	}
}

// Zig: test "term51: Page.assertIntegrity: silent on consistent pages"
func TestPg1_term51_Page_assertIntegrity_silent_on_consistent_pages(t *testing.T) {
	page := pg1HNew(t, 6, 4)
	defer page.Deinit()
	page.AssertIntegrity()

	pg1HFillAll(page)
	pg1HGrapheme(t, page, 1, 0, 'e', []uint32{0x301})
	pg1HStyleCells(t, page, 2, [][2]uint{{0, 1}, {1, 1}})
	pg1HLinkCells(t, page, pg1HLink("https://x"), [][2]uint{{2, 2}})
	page.AssertIntegrity()
}

// Zig: test "term51: Page.assertIntegrity: honors pause and the runtime safety option"
func TestPg1_term51_Page_assertIntegrity_honors_pause_and_the_runtime_safety_option(t *testing.T) {
	page := pg1HNew(t, 6, 4)
	defer page.Deinit()
	page.size.rows = 0 // violates ZeroRowCount

	if pageSlowRuntimeSafety {
		// Would panic if checked; paused so it must not.
		page.PauseIntegrityChecks(true)
		page.AssertIntegrity()
		page.PauseIntegrityChecks(false)
	} else {
		// No-op without runtime safety, even when inconsistent.
		page.AssertIntegrity()
	}
	page.size.rows = 4
	page.AssertIntegrity()
}

// ---- Page.verifyIntegrity -------------------------------------------------

// Zig: test "term51: Page.verifyIntegrity: empty and fully featured pages are valid"
func TestPg1_term51_Page_verifyIntegrity_empty_and_fully_featured_pages_are_valid(t *testing.T) {
	page := pg1HNew(t, 10, 6)
	defer page.Deinit()
	zigTry(t, page.VerifyIntegrity())

	pg1HFillAll(page)
	pg1HGrapheme(t, page, 0, 0, 'e', []uint32{0x301})
	pg1HGrapheme(t, page, 9, 5, 'f', []uint32{0x301, 0x302, 0x303, 0x304, 0x305})
	pg1HStyleCells(t, page, 1, [][2]uint{{1, 1}, {2, 1}, {3, 4}})
	pg1HStyleCells(t, page, 2, [][2]uint{{5, 2}})
	pg1HLinkCells(t, page, pg1HLink("https://a.example"), [][2]uint{{1, 3}, {2, 3}})
	pg1HLinkCells(t, page, Hyperlink{ID: pg1Explicit("id"), URI: []uint8("https://b")}, [][2]uint{{4, 3}})
	// wide + tail
	page.GetRowAndCell(6, 2).cell.setWide(PageCellWideWide)
	page.GetRowAndCell(7, 2).cell.setWide(PageCellWideSpacerTail)
	// wide at the very end without its tail is accepted
	page.GetRowAndCell(9, 2).cell.setWide(PageCellWideWide)
	// wrapped row with a spacer head in the last column
	page.GetRow(0).wrap = true
	page.GetRowAndCell(9, 0).cell.setWide(PageCellWideSpacerHead)
	// bg-color-only cells are fine
	*page.GetRowAndCell(0, 4).cell = pageCellFields{
		content_tag: PageCellContentTagBgColorRgb,
		content:     pageCellContentFields{color_rgb: PageCellRGB{R: 1, G: 2, B: 3}}.pack(),
	}.pack()
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.verifyIntegrity: ZeroRowCount"
func TestPg1_term51_Page_verifyIntegrity_ZeroRowCount(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	page.size.rows = 0
	pg1HExpectIntegrity(t, page, ErrZeroRowCount)
}

// Zig: test "term51: Page.verifyIntegrity: ZeroColCount"
func TestPg1_term51_Page_verifyIntegrity_ZeroColCount(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	page.size.cols = 0
	pg1HExpectIntegrity(t, page, ErrZeroColCount)
}

// Zig: test "term51: Page.verifyIntegrity: zero rows and zero cols reports rows first"
func TestPg1_term51_Page_verifyIntegrity_zero_rows_and_zero_cols_reports_rows_first(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	page.size.rows = 0
	page.size.cols = 0
	pg1HExpectIntegrity(t, page, ErrZeroRowCount)
}

// Zig: test "term51: Page.verifyIntegrity: size smaller than capacity limits what is checked"
func TestPg1_term51_Page_verifyIntegrity_size_smaller_than_capacity_limits_what_is_checked(t *testing.T) {
	page := pg1HNew(t, 10, 10)
	defer page.Deinit()
	// Corrupt cells outside the logical size.
	page.GetRowAndCell(8, 2).cell.setContentTag(PageCellContentTagCodepointGrapheme)
	page.GetRowAndCell(3, 8).cell.setWide(PageCellWideSpacerTail)
	page.GetRowAndCell(8, 8).cell.setHyperlink(true)
	page.size.rows = 5
	page.size.cols = 5
	zigTry(t, page.VerifyIntegrity())

	// Growing the size back exposes the corruption (row 2 first).
	page.size.cols = 10
	pg1HExpectIntegrity(t, page, ErrMissingGraphemeData)
	page.GetRowAndCell(8, 2).cell.setContentTag(PageCellContentTagCodepoint)
	zigTry(t, page.VerifyIntegrity())
	page.size.rows = 10
	pg1HExpectIntegrity(t, page, ErrInvalidSpacerTailLocation)
}

// Zig: test "term51: Page.verifyIntegrity: MissingGraphemeData"
func TestPg1_term51_Page_verifyIntegrity_MissingGraphemeData(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	*page.GetRowAndCell(2, 1).cell = PageCellInit('a')
	page.GetRowAndCell(2, 1).cell.setContentTag(PageCellContentTagCodepointGrapheme)
	pg1HExpectIntegrity(t, page, ErrMissingGraphemeData)

	// Even with the row flagged, the data is still missing.
	page.GetRow(1).grapheme = true
	pg1HExpectIntegrity(t, page, ErrMissingGraphemeData)
}

// Zig: test "term51: Page.verifyIntegrity: UnmarkedGraphemeCell"
func TestPg1_term51_Page_verifyIntegrity_UnmarkedGraphemeCell(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	pg1HGrapheme(t, page, 1, 1, 'a', []uint32{0x301})
	pg1HGrapheme(t, page, 3, 3, 'b', []uint32{0x301})
	zigTry(t, page.VerifyIntegrity())

	// Un-mark one cell but leave its map entry behind.
	page.GetRowAndCell(3, 3).cell.setContentTag(PageCellContentTagCodepoint)
	pg1HExpectIntegrity(t, page, ErrUnmarkedGraphemeCell)
}

// Zig: test "term51: Page.verifyIntegrity: UnmarkedGraphemeRow"
func TestPg1_term51_Page_verifyIntegrity_UnmarkedGraphemeRow(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	pg1HGrapheme(t, page, 0, 2, 'a', []uint32{0x301})
	pg1HGrapheme(t, page, 3, 2, 'b', []uint32{0x301})
	zigTry(t, page.VerifyIntegrity())
	page.GetRow(2).grapheme = false
	pg1HExpectIntegrity(t, page, ErrUnmarkedGraphemeRow)
	// A row flagged without graphemes is just a false positive: valid.
	page.GetRow(2).grapheme = true
	page.GetRow(0).grapheme = true
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.verifyIntegrity: graphemes seen versus map count"
func TestPg1_term51_Page_verifyIntegrity_graphemes_seen_versus_map_count(t *testing.T) {
	page := pg1HNew(t, 6, 4)
	defer page.Deinit()
	// Map entries for cells outside the checked size are not "seen":
	// fewer seen than present is accepted.
	pg1HGrapheme(t, page, 5, 3, 'a', []uint32{0x301})
	pg1HGrapheme(t, page, 0, 0, 'b', []uint32{0x301})
	page.size.cols = 3
	page.size.rows = 2
	expectEqual(t, uint(2), page.GraphemeCount())
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.verifyIntegrity: UnmarkedStyleRow"
func TestPg1_term51_Page_verifyIntegrity_UnmarkedStyleRow(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	pg1HStyleCells(t, page, 1, [][2]uint{{1, 1}, {2, 1}})
	zigTry(t, page.VerifyIntegrity())
	page.GetRow(1).styled = false
	pg1HExpectIntegrity(t, page, ErrUnmarkedStyleRow)
	// The converse (flag without styles) is a valid false positive.
	page.GetRow(1).styled = true
	page.GetRow(3).styled = true
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.verifyIntegrity: MismatchedStyleRef"
func TestPg1_term51_Page_verifyIntegrity_MismatchedStyleRef(t *testing.T) {
	page := pg1HNew(t, 6, 4)
	defer page.Deinit()
	id := pg1HStyleCells(t, page, 3, [][2]uint{{0, 0}, {1, 0}, {2, 3}})
	expectEqual(t, uint16(3), page.styles.RefCount(id))
	zigTry(t, page.VerifyIntegrity())

	// Equal is fine, more references than cells is fine (not exact).
	page.styles.Use(id)
	zigTry(t, page.VerifyIntegrity())
	page.styles.Release(id)
	page.styles.Release(id) // ref 2, 3 cells
	pg1HExpectIntegrity(t, page, ErrMismatchedStyleRef)
}

// Zig: test "term51: Page.verifyIntegrity: style ref counts are checked per style id"
func TestPg1_term51_Page_verifyIntegrity_style_ref_counts_are_checked_per_style_id(t *testing.T) {
	page := pg1HNew(t, 6, 4)
	defer page.Deinit()
	a := pg1HStyleCells(t, page, 1, [][2]uint{{0, 0}, {1, 0}})
	b := pg1HStyleCells(t, page, 2, [][2]uint{{2, 0}, {3, 0}})
	expect(t, a != b, "a != b")
	zigTry(t, page.VerifyIntegrity())
	// Take a reference from one style only.
	page.styles.Release(b)
	pg1HExpectIntegrity(t, page, ErrMismatchedStyleRef)
}

// Zig: test "term51: Page.verifyIntegrity: MissingHyperlinkData"
func TestPg1_term51_Page_verifyIntegrity_MissingHyperlinkData(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	page.GetRowAndCell(1, 1).cell.setHyperlink(true)
	// Reported even when the row flag is also missing (data check is first).
	pg1HExpectIntegrity(t, page, ErrMissingHyperlinkData)
	page.GetRow(1).hyperlink = true
	pg1HExpectIntegrity(t, page, ErrMissingHyperlinkData)
}

// Zig: test "term51: Page.verifyIntegrity: UnmarkedHyperlinkRow"
func TestPg1_term51_Page_verifyIntegrity_UnmarkedHyperlinkRow(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	pg1HLinkCells(t, page, pg1HLink("https://a"), [][2]uint{{1, 1}})
	zigTry(t, page.VerifyIntegrity())
	page.GetRow(1).hyperlink = false
	pg1HExpectIntegrity(t, page, ErrUnmarkedHyperlinkRow)
}

// Zig: test "term51: Page.verifyIntegrity: MismatchedHyperlinkRef"
func TestPg1_term51_Page_verifyIntegrity_MismatchedHyperlinkRef(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	id := pg1HLinkCells(t, page, pg1HLink("https://a"), [][2]uint{{0, 0}, {1, 0}, {2, 0}})
	expectEqual(t, uint16(3), page.hyperlink_set.RefCount(id))
	zigTry(t, page.VerifyIntegrity())
	page.hyperlink_set.Use(id)
	zigTry(t, page.VerifyIntegrity()) // extra refs are fine
	page.hyperlink_set.Release(id)
	page.hyperlink_set.Release(id)
	pg1HExpectIntegrity(t, page, ErrMismatchedHyperlinkRef)
}

// Zig: test "term51: Page.verifyIntegrity: UnmarkedHyperlinkCell"
func TestPg1_term51_Page_verifyIntegrity_UnmarkedHyperlinkCell(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	pg1HLinkCells(t, page, pg1HLink("https://a"), [][2]uint{{1, 1}, {2, 1}})
	zigTry(t, page.VerifyIntegrity())
	// Clear the cell flag but leave the map entry.
	page.GetRowAndCell(2, 1).cell.setHyperlink(false)
	pg1HExpectIntegrity(t, page, ErrUnmarkedHyperlinkCell)
}

// Zig: test "term51: Page.verifyIntegrity: InvalidSpacerTailLocation at column zero"
func TestPg1_term51_Page_verifyIntegrity_InvalidSpacerTailLocation_at_column_zero(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	page.GetRowAndCell(0, 2).cell.setWide(PageCellWideSpacerTail)
	pg1HExpectIntegrity(t, page, ErrInvalidSpacerTailLocation)
}

// Zig: test "term51: Page.verifyIntegrity: InvalidSpacerTailLocation not following wide"
func TestPg1_term51_Page_verifyIntegrity_InvalidSpacerTailLocation_not_following_wide(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	// Previous cell narrow.
	page.GetRowAndCell(1, 0).cell.setWide(PageCellWideSpacerTail)
	pg1HExpectIntegrity(t, page, ErrInvalidSpacerTailLocation)
	// Previous cell wide: valid.
	page.GetRowAndCell(0, 0).cell.setWide(PageCellWideWide)
	zigTry(t, page.VerifyIntegrity())
	// A tail after a tail is not valid.
	page.GetRowAndCell(2, 0).cell.setWide(PageCellWideSpacerTail)
	pg1HExpectIntegrity(t, page, ErrInvalidSpacerTailLocation)
	// A tail after a wide cell that is itself a tail is not valid either.
	page.GetRowAndCell(1, 0).cell.setWide(PageCellWideNarrow)
	page.GetRowAndCell(0, 0).cell.setWide(PageCellWideNarrow)
	pg1HExpectIntegrity(t, page, ErrInvalidSpacerTailLocation)
}

// Zig: test "term51: Page.verifyIntegrity: spacer tail after wide in every column is valid"
func TestPg1_term51_Page_verifyIntegrity_spacer_tail_after_wide_in_every_column_is_valid(t *testing.T) {
	page := pg1HNew(t, 5, 1)
	defer page.Deinit()
	for x := range uint(4) {
		page.GetRowAndCell(x, 0).cell.setWide(PageCellWideWide)
		page.GetRowAndCell(x+1, 0).cell.setWide(PageCellWideSpacerTail)
		zigTry(t, page.VerifyIntegrity())
		page.GetRowAndCell(x, 0).cell.setWide(PageCellWideNarrow)
		page.GetRowAndCell(x+1, 0).cell.setWide(PageCellWideNarrow)
	}
}

// Zig: test "term51: Page.verifyIntegrity: InvalidSpacerHeadLocation"
func TestPg1_term51_Page_verifyIntegrity_InvalidSpacerHeadLocation(t *testing.T) {
	page := pg1HNew(t, 5, 3)
	defer page.Deinit()
	page.GetRow(1).wrap = true
	for _, x := range []uint{0, 1, 3} {
		page.GetRowAndCell(x, 1).cell.setWide(PageCellWideSpacerHead)
		pg1HExpectIntegrity(t, page, ErrInvalidSpacerHeadLocation)
		page.GetRowAndCell(x, 1).cell.setWide(PageCellWideNarrow)
	}
	// Both wrong: the location error wins over the missing wrap.
	page.GetRow(1).wrap = false
	page.GetRowAndCell(2, 1).cell.setWide(PageCellWideSpacerHead)
	pg1HExpectIntegrity(t, page, ErrInvalidSpacerHeadLocation)
}

// Zig: test "term51: Page.verifyIntegrity: UnwrappedSpacerHead"
func TestPg1_term51_Page_verifyIntegrity_UnwrappedSpacerHead(t *testing.T) {
	page := pg1HNew(t, 5, 3)
	defer page.Deinit()
	page.GetRowAndCell(4, 1).cell.setWide(PageCellWideSpacerHead)
	pg1HExpectIntegrity(t, page, ErrUnwrappedSpacerHead)
	page.GetRow(1).wrap = true
	zigTry(t, page.VerifyIntegrity())
	// wrap_continuation does not substitute for wrap.
	page.GetRow(1).wrap = false
	page.GetRow(1).wrap_continuation = true
	pg1HExpectIntegrity(t, page, ErrUnwrappedSpacerHead)
}

// Zig: test "term51: Page.verifyIntegrity: spacer head in a one column page"
func TestPg1_term51_Page_verifyIntegrity_spacer_head_in_a_one_column_page(t *testing.T) {
	page := pg1HNew(t, 1, 2)
	defer page.Deinit()
	page.GetRowAndCell(0, 0).cell.setWide(PageCellWideSpacerHead)
	pg1HExpectIntegrity(t, page, ErrUnwrappedSpacerHead)
	page.GetRow(0).wrap = true
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.verifyIntegrity: wide cells themselves are never rejected"
func TestPg1_term51_Page_verifyIntegrity_wide_cells_themselves_are_never_rejected(t *testing.T) {
	page := pg1HNew(t, 3, 1)
	defer page.Deinit()
	page.GetRowAndCell(0, 0).cell.setWide(PageCellWideWide)
	page.GetRowAndCell(1, 0).cell.setWide(PageCellWideWide)
	page.GetRowAndCell(2, 0).cell.setWide(PageCellWideWide)
	zigTry(t, page.VerifyIntegrity())
}

// Zig: test "term51: Page.verifyIntegrity: rows are checked top to bottom, first error wins"
func TestPg1_term51_Page_verifyIntegrity_rows_are_checked_top_to_bottom_first_error_wins(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	page.GetRowAndCell(0, 3).cell.setWide(PageCellWideSpacerTail) // InvalidSpacerTailLocation
	page.GetRowAndCell(1, 1).cell.setHyperlink(true)              // MissingHyperlinkData
	pg1HExpectIntegrity(t, page, ErrMissingHyperlinkData)
	page.GetRowAndCell(1, 1).cell.setHyperlink(false)
	pg1HExpectIntegrity(t, page, ErrInvalidSpacerTailLocation)
}

// Zig: test "term51: Page.verifyIntegrity: cells are checked left to right, first error wins"
func TestPg1_term51_Page_verifyIntegrity_cells_are_checked_left_to_right_first_error_wins(t *testing.T) {
	page := pg1HNew(t, 4, 1)
	defer page.Deinit()
	page.GetRowAndCell(3, 0).cell.setContentTag(PageCellContentTagCodepointGrapheme) // MissingGraphemeData
	page.GetRowAndCell(1, 0).cell.setHyperlink(true)                                 // MissingHyperlinkData
	pg1HExpectIntegrity(t, page, ErrMissingHyperlinkData)
	page.GetRowAndCell(1, 0).cell.setHyperlink(false)
	pg1HExpectIntegrity(t, page, ErrMissingGraphemeData)
}

// Zig: test "term51: Page.verifyIntegrity: per-cell check order is grapheme, style, hyperlink, wide"
func TestPg1_term51_Page_verifyIntegrity_per_cell_check_order_is_grapheme_style_hyperlink_wide(t *testing.T) {
	page := pg1HNew(t, 4, 1)
	defer page.Deinit()
	// A cell at x=0 that is wrong in all four ways.
	style_id := pg1HStyleCells(t, page, 1, [][2]uint{{0, 0}})
	page.GetRow(0).styled = false // UnmarkedStyleRow
	cell := page.GetRowAndCell(0, 0).cell
	cell.setContentTag(PageCellContentTagCodepointGrapheme) // MissingGraphemeData
	cell.setHyperlink(true)                                 // MissingHyperlinkData
	cell.setWide(PageCellWideSpacerTail)                    // InvalidSpacerTailLocation
	pg1HExpectIntegrity(t, page, ErrMissingGraphemeData)
	cell.setContentTag(PageCellContentTagCodepoint)
	pg1HExpectIntegrity(t, page, ErrUnmarkedStyleRow)
	page.GetRow(0).styled = true
	pg1HExpectIntegrity(t, page, ErrMissingHyperlinkData)
	cell.setHyperlink(false)
	pg1HExpectIntegrity(t, page, ErrInvalidSpacerTailLocation)
	cell.setWide(PageCellWideNarrow)
	zigTry(t, page.VerifyIntegrity())
	_ = style_id
}

// Zig: test "term51: Page.verifyIntegrity: row-level grapheme check runs after the row's cells"
func TestPg1_term51_Page_verifyIntegrity_row_level_grapheme_check_runs_after_the_row_s_cells(t *testing.T) {
	page := pg1HNew(t, 4, 2)
	defer page.Deinit()
	pg1HGrapheme(t, page, 0, 0, 'a', []uint32{0x301})
	page.GetRow(0).grapheme = false                               // UnmarkedGraphemeRow, reported after row 0's cells
	page.GetRowAndCell(1, 1).cell.setWide(PageCellWideSpacerTail) // x=1 follows narrow
	pg1HExpectIntegrity(t, page, ErrUnmarkedGraphemeRow)

	// A cell error later in the same row is found before the row check.
	page.GetRowAndCell(3, 0).cell.setWide(PageCellWideSpacerTail)
	pg1HExpectIntegrity(t, page, ErrInvalidSpacerTailLocation)
}

// Zig: test "term51: Page.verifyIntegrity: style ref check precedes hyperlink ref check"
func TestPg1_term51_Page_verifyIntegrity_style_ref_check_precedes_hyperlink_ref_check(t *testing.T) {
	page := pg1HNew(t, 4, 2)
	defer page.Deinit()
	sid := pg1HStyleCells(t, page, 1, [][2]uint{{0, 0}, {1, 0}})
	hid := pg1HLinkCells(t, page, pg1HLink("https://a"), [][2]uint{{2, 0}, {3, 0}})
	page.hyperlink_set.Release(hid)
	pg1HExpectIntegrity(t, page, ErrMismatchedHyperlinkRef)
	page.styles.Release(sid)
	pg1HExpectIntegrity(t, page, ErrMismatchedStyleRef)
}

// Not translated, Zig test "term51: Page.verifyIntegrity: temporary allocations come from the given allocator": std.testing.FailingAllocator has no Go counterpart; see
// scripts/not-translated.txt.

// Not translated, Zig test "term51: Page.verifyIntegrity: valid pages do not modify the page": memory-snapshot test (compares page.memory byte for byte);
// see scripts/not-translated.txt.

// ---- Page.clone / Page.cloneBuf -------------------------------------------

// Zig: test "term51: Page.clone: copies every cell, row and table"
func TestPg1_term51_Page_clone_copies_every_cell_row_and_table(t *testing.T) {
	page := pg1HNew(t, 10, 6)
	defer page.Deinit()
	pg1HPopulate(t, page)

	cpy := pg1Clone(t, page)
	defer cpy.Deinit()

	// Zig-only: try term51Ht.expect(copy.memory.ptr != page.memory.ptr);
	// Zig-only: try term51Ht.expectEqual(page.memory.len, copy.memory.len);
	expectEqual(t, page.capacity, cpy.capacity)
	expectEqual(t, page.size, cpy.size)
	expectEqual(t, page.dirty, cpy.dirty)
	expectEqual(t, page.GraphemeCount(), cpy.GraphemeCount())
	expectEqual(t, page.HyperlinkCount(), cpy.HyperlinkCount())
	expectEqual(t, page.styles.Count(), cpy.styles.Count())
	expectEqual(t, page.hyperlink_set.Count(), cpy.hyperlink_set.Count())
	expectEqual(t, page.grapheme_alloc.UsedBytes(), cpy.grapheme_alloc.UsedBytes())
	expectEqual(t, page.string_alloc.UsedBytes(), cpy.string_alloc.UsedBytes())
	for y := range uint(6) {
		expectEqual(t, *page.GetRow(y), *cpy.GetRow(y))
		pg1HExpectRowsEqual(t, page, cpy, y)
	}
	// Zig-only: // Byte for byte identical (everything is stored as offsets).
	// Zig-only: try term51Ht.expectEqualSlices(u8, page.memory, copy.memory);
	zigTry(t, cpy.VerifyIntegrity())

	// Managed data resolves inside the copy's own memory.
	expectEqualSlices(t, []uint32{0x301, 0x302}, zigUnwrap(t, cpy.LookupGrapheme(cpy.GetRowAndCell(1, 0).cell)))
	hid := zigUnwrap(t, cpy.LookupHyperlink(cpy.GetRowAndCell(7, 5).cell))
	entry := cpy.hyperlink_set.Get(hid)
	expectEqualStrings(t, "https://two.example/", string(entry.uri))
	expect(t, entry.id.Tag == HyperlinkPageEntryIdExplicit, "entry.id == .explicit")
	expectEqualStrings(t, "my-id", string(entry.id.Explicit))
	sid := cpy.GetRowAndCell(2, 1).cell.StyleID()
	expect(t, Style{FgColor: StyleColor{Tag: StyleColorPalette, Palette: 7}}.Eql(*cpy.styles.Get(sid)), "(Style{ .fg_color = .{ .palette = 7 } }).eql(copy.styles.get(copy.memory, sid).*)")
}

// Zig: test "term51: Page.clone: copy and original are independent"
func TestPg1_term51_Page_clone_copy_and_original_are_independent(t *testing.T) {
	page := pg1HNew(t, 10, 6)
	defer page.Deinit()
	pg1HPopulate(t, page)
	cpy := pg1Clone(t, page)
	defer cpy.Deinit()

	// Mutate the copy.
	cpy.ClearCells(cpy.GetRow(0), 0, 10)
	cpy.ClearCells(cpy.GetRow(1), 0, 10)
	cpy.ClearCells(cpy.GetRow(4), 0, 10)
	expectEqual(t, uint(1), cpy.GraphemeCount())
	expectEqual(t, uint(1), cpy.HyperlinkCount())

	// The original is untouched.
	expectEqual(t, uint(2), page.GraphemeCount())
	expectEqual(t, uint(3), page.HyperlinkCount())
	expectEqual(t, uint32('e'), page.GetRowAndCell(1, 0).cell.Codepoint())
	expectEqualSlices(t, []uint32{0x301, 0x302}, zigUnwrap(t, page.LookupGrapheme(page.GetRowAndCell(1, 0).cell)))
	zigTry(t, page.VerifyIntegrity())
	zigTry(t, cpy.VerifyIntegrity())

	// And the other direction.
	*page.GetRowAndCell(0, 5).cell = PageCellInit('Z')
	expect(t, cpy.GetRowAndCell(0, 5).cell.Codepoint() != 'Z', "copy.getRowAndCell(0, 5).cell.codepoint() != 'Z'")
}

// Zig: test "term51: Page.clone: keeps a reduced logical size and dirty flags"
func TestPg1_term51_Page_clone_keeps_a_reduced_logical_size_and_dirty_flags(t *testing.T) {
	page := pg1HNew(t, 10, 8)
	defer page.Deinit()
	pg1HFillAll(page)
	page.size.rows = 5
	page.size.cols = 7
	page.dirty = true
	page.GetRow(2).dirty = true

	cpy := pg1Clone(t, page)
	defer cpy.Deinit()
	expectEqual(t, SizeCellCountInt(5), cpy.size.rows)
	expectEqual(t, SizeCellCountInt(7), cpy.size.cols)
	expectEqual(t, SizeCellCountInt(10), cpy.capacity.cols)
	expectEqual(t, SizeCellCountInt(8), cpy.capacity.rows)
	expect(t, cpy.dirty, "copy.dirty")
	expect(t, cpy.GetRow(2).dirty, "copy.getRow(2).dirty")
	expect(t, !cpy.GetRow(1).dirty, "!copy.getRow(1).dirty")
	expect(t, cpy.IsDirty(), "copy.isDirty()")
}

// Not translated, Zig test "term51: Page.clone: an empty page clones to an empty page": memory-snapshot test (compares page.memory byte for byte);
// see scripts/not-translated.txt.

// Zig: test "term51: Page.clone: copies the integrity pause counter"
func TestPg1_term51_Page_clone_copies_the_integrity_pause_counter(t *testing.T) {
	page := pg1HNew(t, 3, 3)
	defer page.Deinit()
	page.PauseIntegrityChecks(true)
	cpy := pg1Clone(t, page)
	defer cpy.Deinit()
	if pageSlowRuntimeSafety {
		expectEqual(t, uint64(1), cpy.pause_integrity_checks)
	}
	page.PauseIntegrityChecks(false)
	cpy.PauseIntegrityChecks(false)
}

// The two Page.cloneBuf tests are not translated (no memory base in Go); see
// scripts/not-translated.txt.

// ---- Page.exactRowCapacity ------------------------------------------------

// Zig: test "term51: Page.exactRowCapacity: empty rows need nothing but the shape"
func TestPg1_term51_Page_exactRowCapacity_empty_rows_need_nothing_but_the_shape(t *testing.T) {
	page := pg1HNew(t, 7, 9)
	defer page.Deinit()
	ranges := [][2]uint{{0, 1}, {3, 7}, {8, 9}, {0, 9}}
	for _, r := range ranges {
		cp := page.ExactRowCapacity(r[0], r[1])
		expectEqual(t, PageCapacity{
			cols:            7,
			rows:            SizeCellCountInt(r[1] - r[0]),
			styles:          0,
			grapheme_bytes:  0,
			hyperlink_bytes: 0,
			string_bytes:    0,
		}, cp)
	}
}

// Zig: test "term51: Page.exactRowCapacity: columns come from the size, not the capacity"
func TestPg1_term51_Page_exactRowCapacity_columns_come_from_the_size_not_the_capacity(t *testing.T) {
	page := pg1HNew(t, 10, 4)
	defer page.Deinit()
	page.size.cols = 6
	cp := page.ExactRowCapacity(0, 2)
	expectEqual(t, SizeCellCountInt(6), cp.cols)
	expectEqual(t, SizeCellCountInt(2), cp.rows)
	// Cells beyond the size are not looked at.
	page.size.cols = 10
	pg1HStyleCells(t, page, 1, [][2]uint{{8, 0}})
	page.size.cols = 6
	expectEqual(t, SizeStyleCountInt(0), page.ExactRowCapacity(0, 2).styles)
}

// Zig: test "term51: Page.exactRowCapacity: styles are counted once per distinct id"
func TestPg1_term51_Page_exactRowCapacity_styles_are_counted_once_per_distinct_id(t *testing.T) {
	page := pg1HNew(t, 10, 6)
	defer page.Deinit()

	// One style on many cells: one unique style.
	pg1HStyleCells(t, page, 1, [][2]uint{{0, 0}, {1, 0}, {2, 0}, {0, 1}, {5, 1}})
	expectEqual(t, SizeStyleCountInt(RefCountedSetCapacityForCount(1)), page.ExactRowCapacity(0, 2).styles)
	expectEqual(t, SizeStyleCountInt(3), page.ExactRowCapacity(0, 2).styles)

	// A second and further styles.
	pg1HStyleCells(t, page, 2, [][2]uint{{0, 2}})
	expectEqual(t, SizeStyleCountInt(4), page.ExactRowCapacity(0, 3).styles)
	pg1HStyleCells(t, page, 3, [][2]uint{{0, 3}})
	expectEqual(t, SizeStyleCountInt(5), page.ExactRowCapacity(0, 4).styles)
	pg1HStyleCells(t, page, 4, [][2]uint{{0, 4}})
	pg1HStyleCells(t, page, 5, [][2]uint{{0, 5}})
	expectEqual(t, SizeStyleCountInt(8), page.ExactRowCapacity(0, 6).styles)
	expectEqual(t, SizeStyleCountInt(RefCountedSetCapacityForCount(5)), page.ExactRowCapacity(0, 6).styles)

	// Only rows inside the range count.
	expectEqual(t, SizeStyleCountInt(3), page.ExactRowCapacity(1, 2).styles)
	expectEqual(t, SizeStyleCountInt(4), page.ExactRowCapacity(1, 3).styles)
	expectEqual(t, SizeStyleCountInt(3), page.ExactRowCapacity(5, 6).styles)
}

// Zig: test "term51: Page.exactRowCapacity: styles are found without row flags"
func TestPg1_term51_Page_exactRowCapacity_styles_are_found_without_row_flags(t *testing.T) {
	page := pg1HNew(t, 4, 2)
	defer page.Deinit()
	pg1HStyleCells(t, page, 1, [][2]uint{{1, 0}})
	page.GetRow(0).styled = false
	expectEqual(t, SizeStyleCountInt(3), page.ExactRowCapacity(0, 1).styles)
}

// Zig: test "term51: Page.exactRowCapacity: grapheme bytes are chunk rounded per cell"
func TestPg1_term51_Page_exactRowCapacity_grapheme_bytes_are_chunk_rounded_per_cell(t *testing.T) {
	page := pg1HNew(t, 10, 6)
	defer page.Deinit()

	// 1 codepoint = 4 bytes -> 1 chunk of 16
	pg1HGrapheme(t, page, 0, 0, 'a', []uint32{0x301})
	expectEqual(t, SizeGraphemeBytesInt(16), page.ExactRowCapacity(0, 1).grapheme_bytes)
	// 4 codepoints = 16 bytes -> 1 chunk
	pg1HGrapheme(t, page, 1, 0, 'b', []uint32{1, 2, 3, 4})
	expectEqual(t, SizeGraphemeBytesInt(32), page.ExactRowCapacity(0, 1).grapheme_bytes)
	// 5 codepoints = 20 bytes -> 2 chunks
	pg1HGrapheme(t, page, 2, 0, 'c', []uint32{1, 2, 3, 4, 5})
	expectEqual(t, SizeGraphemeBytesInt(64), page.ExactRowCapacity(0, 1).grapheme_bytes)
	// 8 codepoints = 32 bytes -> 2 chunks; 9 -> 3 chunks
	pg1HGrapheme(t, page, 3, 0, 'd', []uint32{1, 2, 3, 4, 5, 6, 7, 8})
	expectEqual(t, SizeGraphemeBytesInt(96), page.ExactRowCapacity(0, 1).grapheme_bytes)
	pg1HGrapheme(t, page, 4, 0, 'e', []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9})
	expectEqual(t, SizeGraphemeBytesInt(144), page.ExactRowCapacity(0, 1).grapheme_bytes)

	// Cells outside the row range are not counted.
	pg1HGrapheme(t, page, 0, 3, 'x', []uint32{0x301})
	expectEqual(t, SizeGraphemeBytesInt(144), page.ExactRowCapacity(0, 3).grapheme_bytes)
	expectEqual(t, SizeGraphemeBytesInt(16), page.ExactRowCapacity(3, 4).grapheme_bytes)
	expectEqual(t, SizeGraphemeBytesInt(160), page.ExactRowCapacity(0, 4).grapheme_bytes)
	expectEqual(t, SizeGraphemeBytesInt(0), page.ExactRowCapacity(4, 6).grapheme_bytes)
}

// Zig: test "term51: Page.exactRowCapacity: a grapheme tag without map data adds nothing"
func TestPg1_term51_Page_exactRowCapacity_a_grapheme_tag_without_map_data_adds_nothing(t *testing.T) {
	page := pg1HNew(t, 4, 2)
	defer page.Deinit()
	*page.GetRowAndCell(1, 0).cell = PageCellInit('a')
	page.GetRowAndCell(1, 0).cell.setContentTag(PageCellContentTagCodepointGrapheme)
	expectEqual(t, SizeGraphemeBytesInt(0), page.ExactRowCapacity(0, 1).grapheme_bytes)
}

// ---------------------------------------------------------------------------
// Go-only panic tests: assert / unreachable / panic paths of the functions
// tested above that the Zig tests cannot reach (a Zig test that hit one
// would abort). The setup is outside pg1ExpectPanic so only the call under
// test can satisfy it.
// ---------------------------------------------------------------------------

// Go-only: panic test: PageCapacity.Adjust: the new row count does not fit the row count type
func TestPg1_panic_Capacity_adjust_new_row_count_overflows_u16(t *testing.T) {
	// Zig: `adjusted.rows = @intCast(new_rows)` is a safety-checked cast to
	// u16. A tall capacity adjusted to one column has room for far more
	// than 65535 rows.
	cp := PageCapacityInit(10, 65535)
	pg1ExpectPanic(t, func() {
		_, _ = cp.Adjust(PageCapacityAdjustment{cols: Some[SizeCellCountInt](1)})
	})
}

// Go-only: panic test: Page.ExactRowCapacity: y_start >= y_end
func TestPg1_panic_Page_exactRowCapacity_empty_range(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	pg1ExpectPanic(t, func() {
		page.ExactRowCapacity(2, 2)
	})
}

// Go-only: panic test: Page.ExactRowCapacity: y_start > y_end
func TestPg1_panic_Page_exactRowCapacity_reversed_range(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	pg1ExpectPanic(t, func() {
		page.ExactRowCapacity(3, 1)
	})
}

// Go-only: panic test: Page.ExactRowCapacity: y_end > size.rows
func TestPg1_panic_Page_exactRowCapacity_end_beyond_size(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	pg1ExpectPanic(t, func() {
		page.ExactRowCapacity(0, 5)
	})
}

// Go-only: panic test: Page.ExactRowCapacity: the style capacity does not fit the style count type
func TestPg1_panic_Page_exactRowCapacity_styles_capacity_overflows_u16(t *testing.T) {
	// Zig: `.styles = @intCast(styles_cap)` is a safety-checked cast to u16;
	// capacityForCount(n) exceeds 65535 once n + 1 > 0.8125 * 65535 (about
	// 53247 styles). exactRowCapacity only reads the cells' style ids.
	cp := PageCapacityInit(math.MaxUint16, 1)
	cp.styles = math.MaxUint16
	page := pg1Init(t, cp)
	defer page.Deinit()
	cells := page.GetCells(page.GetRow(0))
	for i := range 54000 {
		cells[i].setStyleID(StyleID(i + 1))
	}
	pg1ExpectPanic(t, func() {
		page.ExactRowCapacity(0, 1)
	})
}

// Go-only: panic test: Page.CloneFrom: y_start > y_end
func TestPg1_panic_Page_cloneFrom_reversed_range(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()
	page2 := pg1Init(t, pg1Cap(10, 10, 8))
	defer page2.Deinit()
	pg1ExpectPanic(t, func() {
		_ = page2.CloneFrom(page, 5, 3)
	})
}

// Go-only: panic test: Page.CloneFrom: y_end > other.size.rows
func TestPg1_panic_Page_cloneFrom_end_beyond_other_size(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()
	page2 := pg1Init(t, pg1Cap(10, 10, 8))
	defer page2.Deinit()
	pg1ExpectPanic(t, func() {
		_ = page2.CloneFrom(page, 0, 11)
	})
}

// Go-only: panic test: Page.CloneFrom: y_end - y_start > self.size.rows
func TestPg1_panic_Page_cloneFrom_range_larger_than_destination(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()
	page2 := pg1Init(t, pg1Cap(10, 5, 8))
	defer page2.Deinit()
	pg1ExpectPanic(t, func() {
		_ = page2.CloneFrom(page, 0, 10)
	})
}

// Go-only: panic test: Page.ClonePartialRowFrom: x_start > x_end
func TestPg1_panic_Page_clonePartialRowFrom_start_after_end(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()
	page2 := pg1Init(t, pg1Cap(10, 10, 8))
	defer page2.Deinit()
	pg1ExpectPanic(t, func() {
		_ = page2.ClonePartialRowFrom(page, page2.GetRow(0), page.GetRow(0), 8, 2)
	})
}

// Go-only: panic test: Page.ClonePartialRowFrom: source row without managed memory has a grapheme cell
func TestPg1_panic_Page_clonePartialRowFrom_unmanaged_source_row_with_grapheme_cell(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()
	page2 := pg1Init(t, pg1Cap(10, 10, 8))
	defer page2.Deinit()
	// The row is not flagged grapheme, so it has no managed memory.
	page.GetRowAndCell(3, 0).cell.setContentTag(PageCellContentTagCodepointGrapheme)
	pg1ExpectPanic(t, func() {
		_ = page2.ClonePartialRowFrom(page, page2.GetRow(0), page.GetRow(0), 0, 10)
	})
}

// Go-only: panic test: Page.ClonePartialRowFrom: source row without managed memory has a hyperlink cell
func TestPg1_panic_Page_clonePartialRowFrom_unmanaged_source_row_with_hyperlink_cell(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()
	page2 := pg1Init(t, pg1Cap(10, 10, 8))
	defer page2.Deinit()
	page.GetRowAndCell(3, 0).cell.setHyperlink(true)
	pg1ExpectPanic(t, func() {
		_ = page2.ClonePartialRowFrom(page, page2.GetRow(0), page.GetRow(0), 0, 10)
	})
}

// Go-only: panic test: Page.ClonePartialRowFrom: source row without managed memory has a styled cell
func TestPg1_panic_Page_clonePartialRowFrom_unmanaged_source_row_with_styled_cell(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()
	page2 := pg1Init(t, pg1Cap(10, 10, 8))
	defer page2.Deinit()
	page.GetRowAndCell(3, 0).cell.setStyleID(1)
	pg1ExpectPanic(t, func() {
		_ = page2.ClonePartialRowFrom(page, page2.GetRow(0), page.GetRow(0), 0, 10)
	})
}

// Go-only: panic test: Page.ClonePartialRowFrom: source grapheme cell without grapheme data
func TestPg1_panic_Page_clonePartialRowFrom_source_grapheme_cell_without_data(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()
	page2 := pg1Init(t, pg1Cap(10, 10, 8))
	defer page2.Deinit()
	// Zig: `other.lookupGrapheme(src_cell).?` on a null optional.
	page.GetRow(0).grapheme = true
	page.GetRowAndCell(3, 0).cell.setContentTag(PageCellContentTagCodepointGrapheme)
	pg1ExpectPanic(t, func() {
		_ = page2.ClonePartialRowFrom(page, page2.GetRow(0), page.GetRow(0), 0, 10)
	})
}

// Go-only: panic test: Page.ClonePartialRowFrom: source hyperlink cell without hyperlink data
func TestPg1_panic_Page_clonePartialRowFrom_source_hyperlink_cell_without_data(t *testing.T) {
	page := pg1Init(t, pg1Cap(10, 10, 8))
	defer page.Deinit()
	page2 := pg1Init(t, pg1Cap(10, 10, 8))
	defer page2.Deinit()
	// Zig: `other.lookupHyperlink(src_cell).?` on a null optional.
	page.GetRow(0).hyperlink = true
	page.GetRowAndCell(3, 0).cell.setHyperlink(true)
	pg1ExpectPanic(t, func() {
		_ = page2.ClonePartialRowFrom(page, page2.GetRow(0), page.GetRow(0), 0, 10)
	})
}

// Go-only: panic test: Page.GetRow: y >= size.rows
func TestPg1_panic_Page_getRow_y_out_of_range(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	pg1ExpectPanic(t, func() {
		page.GetRow(4)
	})
}

// Go-only: panic test: Page.GetRow: y inside the capacity but outside the size
func TestPg1_panic_Page_getRow_y_beyond_reduced_size(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	page.size.rows = 2
	pg1ExpectPanic(t, func() {
		page.GetRow(2)
	})
}

// Go-only: panic test: Page.GetRowAndCell: y >= size.rows
func TestPg1_panic_Page_getRowAndCell_y_out_of_range(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	pg1ExpectPanic(t, func() {
		page.GetRowAndCell(0, 4)
	})
}

// Go-only: panic test: Page.GetRowAndCell: x >= size.cols
func TestPg1_panic_Page_getRowAndCell_x_out_of_range(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	pg1ExpectPanic(t, func() {
		page.GetRowAndCell(4, 0)
	})
}

// Go-only: panic test: Page.GetRowAndCell: x inside the capacity but outside the size
func TestPg1_panic_Page_getRowAndCell_x_beyond_reduced_size(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	page.size.cols = 2
	pg1ExpectPanic(t, func() {
		page.GetRowAndCell(2, 0)
	})
}

// Go-only: panic test: Page.GetCells: row is not one of the page's rows
func TestPg1_panic_Page_getCells_row_not_in_page(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	// Zig (slow_runtime_safety): the row pointer must lie between the rows
	// and the cells.
	foreign := PageRow{}
	pg1ExpectPanic(t, func() {
		page.GetCells(&foreign)
	})
}

// Go-only: panic test: Page.GetCells: row belongs to another page
func TestPg1_panic_Page_getCells_row_of_another_page(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	other := pg1HNew(t, 4, 4)
	defer other.Deinit()
	pg1ExpectPanic(t, func() {
		page.GetCells(other.GetRow(0))
	})
}

// Go-only: panic test: Page.AppendGrapheme: cell with codepoint 0
func TestPg1_panic_Page_appendGrapheme_zero_codepoint_cell(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	// Zig (slow_runtime_safety): assert(cell.codepoint() != 0).
	rac := page.GetRowAndCell(0, 0)
	pg1ExpectPanic(t, func() {
		_ = page.AppendGrapheme(rac.row, rac.cell, 0x0A)
	})
}

// Go-only: panic test: Page.AppendGrapheme: grapheme cell in a row not marked grapheme
func TestPg1_panic_Page_appendGrapheme_grapheme_cell_in_unmarked_row(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	// Zig: assert(row.grapheme) once the cell already has graphemes.
	rac := page.GetRowAndCell(0, 0)
	*rac.cell = PageCellInit(0x09)
	rac.cell.setContentTag(PageCellContentTagCodepointGrapheme)
	pg1ExpectPanic(t, func() {
		_ = page.AppendGrapheme(rac.row, rac.cell, 0x0A)
	})
}

// Go-only: panic test: Page.AppendGrapheme: grapheme cell without grapheme data
func TestPg1_panic_Page_appendGrapheme_grapheme_cell_without_data(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	// Zig: `map.getPtr(cell_offset).?` on a null optional.
	rac := page.GetRowAndCell(0, 0)
	*rac.cell = PageCellInit(0x09)
	rac.cell.setContentTag(PageCellContentTagCodepointGrapheme)
	rac.row.grapheme = true
	pg1ExpectPanic(t, func() {
		_ = page.AppendGrapheme(rac.row, rac.cell, 0x0A)
	})
}

// Go-only: panic test: Page.ClearGrapheme: cell without graphemes
func TestPg1_panic_Page_clearGrapheme_cell_without_graphemes(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	// Zig (slow_runtime_safety): assert(cell.hasGrapheme()).
	rac := page.GetRowAndCell(0, 0)
	*rac.cell = PageCellInit(0x09)
	pg1ExpectPanic(t, func() {
		page.ClearGrapheme(rac.cell)
	})
}

// Go-only: panic test: Page.ClearGrapheme: grapheme cell without grapheme data
func TestPg1_panic_Page_clearGrapheme_grapheme_cell_without_data(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	// Zig: `map.getEntry(cell_offset).?` on a null optional.
	rac := page.GetRowAndCell(0, 0)
	*rac.cell = PageCellInit(0x09)
	rac.cell.setContentTag(PageCellContentTagCodepointGrapheme)
	pg1ExpectPanic(t, func() {
		page.ClearGrapheme(rac.cell)
	})
}

// Go-only: panic test: Page.SetGraphemes: cell with codepoint 0
func TestPg1_panic_Page_setGraphemes_zero_codepoint_cell(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	// Zig: assert(cell.codepoint() > 0).
	rac := page.GetRowAndCell(0, 0)
	pg1ExpectPanic(t, func() {
		_ = page.SetGraphemes(rac.row, rac.cell, []uint32{0x301})
	})
}

// Go-only: panic test: Page.SetGraphemes: cell already has graphemes
func TestPg1_panic_Page_setGraphemes_cell_already_has_graphemes(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	// Zig: assert(cell.content_tag == .codepoint).
	rac := page.GetRowAndCell(0, 0)
	*rac.cell = PageCellInit(0x09)
	zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 0x0A))
	pg1ExpectPanic(t, func() {
		_ = page.SetGraphemes(rac.row, rac.cell, []uint32{0x301})
	})
}

// Go-only: panic test: Page.MoveGrapheme: source cell without graphemes
func TestPg1_panic_Page_moveGrapheme_source_without_graphemes(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	// Zig (slow_runtime_safety): assert(src.hasGrapheme()).
	src := page.GetRowAndCell(0, 0)
	*src.cell = PageCellInit(0x09)
	dst := page.GetRowAndCell(1, 0)
	pg1ExpectPanic(t, func() {
		page.MoveGrapheme(src.cell, dst.cell)
	})
}

// Go-only: panic test: Page.MoveGrapheme: destination cell already has graphemes
func TestPg1_panic_Page_moveGrapheme_destination_has_graphemes(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	// Zig (slow_runtime_safety): assert(!dst.hasGrapheme()).
	pg1HGrapheme(t, page, 0, 0, 'a', []uint32{0x301})
	pg1HGrapheme(t, page, 1, 0, 'b', []uint32{0x302})
	src := page.GetRowAndCell(0, 0)
	dst := page.GetRowAndCell(1, 0)
	pg1ExpectPanic(t, func() {
		page.MoveGrapheme(src.cell, dst.cell)
	})
}

// Go-only: panic test: Page.moveHyperlink: source cell without a hyperlink
func TestPg1_panic_Page_moveHyperlink_source_without_hyperlink(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	// Zig: assert(src.hyperlink).
	src := page.GetRowAndCell(0, 0)
	dst := page.GetRowAndCell(1, 0)
	pg1ExpectPanic(t, func() {
		page.moveHyperlink(src.cell, dst.cell)
	})
}

// Go-only: panic test: Page.moveHyperlink: destination cell already has a hyperlink
func TestPg1_panic_Page_moveHyperlink_destination_has_hyperlink(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	// Zig: assert(!dst.hyperlink).
	pg1HLinkCells(t, page, pg1HLink("https://a"), [][2]uint{{0, 0}, {1, 0}})
	src := page.GetRowAndCell(0, 0)
	dst := page.GetRowAndCell(1, 0)
	pg1ExpectPanic(t, func() {
		page.moveHyperlink(src.cell, dst.cell)
	})
}

// Go-only: panic test: Page.MoveCells: source grapheme cell without grapheme data
func TestPg1_panic_Page_moveCells_source_grapheme_cell_without_data(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	// Zig: moveGrapheme does `map.getEntry(src_offset).?` on a null optional.
	rac := page.GetRowAndCell(0, 0)
	*rac.cell = PageCellInit('a')
	rac.cell.setContentTag(PageCellContentTagCodepointGrapheme)
	rac.row.grapheme = true
	pg1ExpectPanic(t, func() {
		page.MoveCells(page.GetRow(0), 0, page.GetRow(1), 0, 1)
	})
}

// Go-only: panic test: Page.MoveCells: source hyperlink cell without hyperlink data
func TestPg1_panic_Page_moveCells_source_hyperlink_cell_without_data(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	// Zig: moveHyperlink does `map.getEntry(src_offset).?` on a null optional.
	rac := page.GetRowAndCell(0, 0)
	rac.cell.setHyperlink(true)
	rac.row.hyperlink = true
	pg1ExpectPanic(t, func() {
		page.MoveCells(page.GetRow(0), 0, page.GetRow(1), 0, 1)
	})
}

// Go-only: panic test: Page.SetHyperlink: same hyperlink set again on a row not marked hyperlink
func TestPg1_panic_Page_setHyperlink_same_id_on_unmarked_row(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	// Zig: when the cell already maps to this id, assert(row.hyperlink).
	id := pg1HLinkCells(t, page, pg1HLink("https://a"), [][2]uint{{1, 1}})
	page.hyperlink_set.Use(id)
	page.hyperlink_set.Use(id)
	rac := page.GetRowAndCell(1, 1)
	rac.row.hyperlink = false
	pg1ExpectPanic(t, func() {
		_ = page.SetHyperlink(rac.row, rac.cell, id)
	})
}

// Go-only: panic test: Page.InsertHyperlink: empty URI
func TestPg1_panic_Page_insertHyperlink_empty_uri(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	// Zig: string_alloc.alloc asserts n > 0 for the URI.
	pg1ExpectPanic(t, func() {
		_, _ = page.InsertHyperlink(Hyperlink{ID: pg1Implicit(0)})
	})
}

// Go-only: panic test: Page.PauseIntegrityChecks: unpause without a pause
func TestPg1_panic_Page_pauseIntegrityChecks_unpause_below_zero(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	page := pg1HNew(t, 3, 3)
	defer page.Deinit()
	// Zig: `self.pause_integrity_checks -= 1` on a usize at 0 is a
	// safety-checked integer overflow.
	pg1ExpectPanic(t, func() {
		page.PauseIntegrityChecks(false)
	})
}

// Go-only: panic test: Page.AssertIntegrity: integrity violation
func TestPg1_panic_Page_assertIntegrity_violation(t *testing.T) {
	setPageSlowRuntimeSafety(t, true)
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	// Zig: @panic("page integrity violation").
	page.size.rows = 0
	pg1ExpectPanic(t, func() {
		page.AssertIntegrity()
	})
}

// Go-only: panic test: Page.VerifyIntegrity: cell style not in the style set
func TestPg1_panic_Page_verifyIntegrity_cell_style_not_in_set(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	// Zig: `self.styles.get` asserts the id is present in the set.
	rac := page.GetRowAndCell(1, 1)
	rac.row.styled = true
	rac.cell.setStyleID(5)
	pg1ExpectPanic(t, func() {
		_ = page.VerifyIntegrity()
	})
}

// Go-only: panic test: Page.VerifyIntegrity: hyperlink id of a cell not in the hyperlink set
func TestPg1_panic_Page_verifyIntegrity_hyperlink_id_not_in_set(t *testing.T) {
	page := pg1HNew(t, 4, 4)
	defer page.Deinit()
	// Zig: `self.hyperlink_set.get` asserts the id is present in the set.
	rac := page.GetRowAndCell(1, 1)
	rac.row.hyperlink = true
	rac.cell.setHyperlink(true)
	zigTry(t, page.hyperlink_map.Map().PutNoClobber(page.cellOffset(rac.cell), 3))
	pg1ExpectPanic(t, func() {
		_ = page.VerifyIntegrity()
	})
}
