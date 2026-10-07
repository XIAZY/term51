package terminal

// Go-only tests for the project lead's ruling on pageSlowRuntimeSafety
// (page.go): the slow integrity checks follow upstream's RELEASE build by
// default. Upstream compiles the checks gated on
// `build_options.slow_runtime_safety` only into Debug builds; the reference
// build (the ReleaseSafe recorder of the corpus) and shipped Ghostty run
// without them and continue where a Debug build panics. Each test runs one
// such operation twice: with the option off it completes without a panic
// and leaves what upstream's release build leaves, with it on it panics as
// the Debug build does. Names are TestPageRuling_..., the page helpers are
// page_2_test.go's (pg2New, pg2FillRow, pg2ExpectError); prExpectPanic
// checks the panic value.

import "testing"

// prExpectPanic expects f to panic with the value want: "assertion failed"
// (assert, zig.go), "page integrity violation" (Page.AssertIntegrity) or
// "row is not in the page" (Page.rowIndex).
func prExpectPanic(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("expected the panic %q, got none", want)
		}
		if s, ok := r.(string); !ok || s != want {
			t.Fatalf("expected the panic %q, got: %v", want, r)
		}
	}()
	f()
}

// Go-only: ruling (slow_runtime_safety off by default: a partial clone starting at an orphaned spacer tail completes and leaves the tail, findings 14.10; on: the closing assertIntegrity panics with InvalidSpacerTailLocation)
func TestPageRuling_slow_runtime_safety_partial_clone_of_an_orphaned_spacer_tail(t *testing.T) {
	run := func() (*Page, error) {
		src := pg2New(t, 10, 1)
		t.Cleanup(src.Deinit)
		*src.GetRowAndCell(3, 0).cell = PageCellInit('a')
		src.GetRowAndCell(3, 0).cell.wide = PageCellWideWide
		src.GetRowAndCell(4, 0).cell.wide = PageCellWideSpacerTail
		dst := pg2New(t, 10, 1)
		t.Cleanup(dst.Deinit)
		// The destination has no wide cell at x=3.
		err := dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 4, 5)
		return dst, err
	}

	setPageSlowRuntimeSafety(t, false)
	dst, err := run()
	zigTry(t, err)
	expectEqual(t, PageCellWideNarrow, dst.GetRowAndCell(3, 0).cell.wide)
	expectEqual(t, PageCellWideSpacerTail, dst.GetRowAndCell(4, 0).cell.wide)
	// What the release build leaves is a page that fails verifyIntegrity
	// (which is not gated on the option).
	pg2ExpectError(t, ErrInvalidSpacerTailLocation, dst.VerifyIntegrity())

	setPageSlowRuntimeSafety(t, true)
	prExpectPanic(t, "page integrity violation", func() { _, _ = run() })
}

// Go-only: ruling (slow_runtime_safety off by default: appendGrapheme on a cell with codepoint 0 appends, page.zig:1487; on: assert(cell.codepoint() != 0) panics)
func TestPageRuling_slow_runtime_safety_appendGrapheme_on_a_codepoint_zero_cell(t *testing.T) {
	setPageSlowRuntimeSafety(t, false)
	page := pg2New(t, 10, 1)
	defer page.Deinit()
	rac := page.GetRowAndCell(0, 0)
	expectEqual(t, uint32(0), rac.cell.Codepoint())
	zigTry(t, page.AppendGrapheme(rac.row, rac.cell, 0x301))
	// The release build takes the first-grapheme path: the cell becomes a
	// grapheme cell whose codepoint is still 0 and the row is marked.
	expectEqual(t, PageCellContentTagCodepointGrapheme, rac.cell.content_tag)
	expectEqual(t, uint32(0), rac.cell.Codepoint())
	expect(t, rac.row.grapheme, "row.grapheme")
	cps := page.LookupGrapheme(rac.cell)
	expect(t, cps.Valid, "lookupGrapheme(cell) != null")
	expectEqualSlices(t, []uint32{0x301}, cps.Value)

	setPageSlowRuntimeSafety(t, true)
	page2 := pg2New(t, 10, 1)
	defer page2.Deinit()
	rac2 := page2.GetRowAndCell(0, 0)
	prExpectPanic(t, "assertion failed", func() { _ = page2.AppendGrapheme(rac2.row, rac2.cell, 0x301) })
}

// Go-only: ruling (slow_runtime_safety off by default: clonePartialRowFrom from a row whose flags say no managed memory copies styled and hyperlink cells bit for bit, page.zig:846-852; on: the cell asserts panic)
func TestPageRuling_slow_runtime_safety_clone_of_an_unmarked_row_with_managed_cells(t *testing.T) {
	run := func() (*Page, error) {
		src := pg2New(t, 10, 1)
		t.Cleanup(src.Deinit)
		pg2FillRow(src, 0, 'a')
		// Neither row.styled nor row.hyperlink is set.
		src.GetRowAndCell(1, 0).cell.style_id = 1
		src.GetRowAndCell(2, 0).cell.hyperlink = true
		dst := pg2New(t, 10, 1)
		t.Cleanup(dst.Deinit)
		err := dst.ClonePartialRowFrom(src, dst.GetRow(0), src.GetRow(0), 0, 10)
		return dst, err
	}

	setPageSlowRuntimeSafety(t, false)
	dst, err := run()
	zigTry(t, err)
	// fastmem.copy: the cells arrive unchanged, no style reference or
	// hyperlink entry is taken in the destination, and the destination
	// row takes the source's flags (a full-width copy).
	for x := range uint(10) {
		expectEqual(t, uint32('a')+uint32(x), dst.GetRowAndCell(x, 0).cell.Codepoint())
	}
	expectEqual(t, StyleID(1), dst.GetRowAndCell(1, 0).cell.style_id)
	expect(t, dst.GetRowAndCell(2, 0).cell.hyperlink, "cell(2).hyperlink")
	row := dst.GetRow(0)
	expect(t, !row.styled, "!row.styled")
	expect(t, !row.hyperlink, "!row.hyperlink")
	expectEqual(t, HashMapSize(0), dst.hyperlink_map.Map().Count())

	setPageSlowRuntimeSafety(t, true)
	prExpectPanic(t, "assertion failed", func() { _, _ = run() })
}

// Go-only: ruling (slow_runtime_safety off by default: getCells of another page's row returns this page's cells at that row's offset, page.zig:996-1004; on: the row address asserts panic)
func TestPageRuling_slow_runtime_safety_getCells_of_another_pages_row(t *testing.T) {
	setPageSlowRuntimeSafety(t, false)
	page := pg2New(t, 4, 2)
	defer page.Deinit()
	other := pg2New(t, 4, 2)
	defer other.Deinit()
	// Release: `row.cells.ptr(self.memory)`, the other row's offset into
	// this page's memory. With the same capacity that is this page's row 1.
	cells := page.GetCells(other.GetRow(1))
	own := page.GetCells(page.GetRow(1))
	expectEqual(t, len(own), len(cells))
	expect(t, &cells[0] == &own[0], "same cells as the page's own row 1")

	setPageSlowRuntimeSafety(t, true)
	prExpectPanic(t, "row is not in the page", func() { _ = page.GetCells(other.GetRow(1)) })
}
