# mapping-pending: fix/stale-cursor

## Page-layer bookkeeping: the style-capacity cast in splitForCapacity (deviation, project lead's rule)

`Screen.splitForCapacity` sizes the rows above and below the cursor with
`Page.exactRowCapacity`. Its `.styles = @intCast(styles_cap)` (page.zig:737)
traps when a range references every id of a full style set of the maximum
capacity (capacityForCount(53247) = 65536). Terminal input reaches it (term51
suite `text/style_split`, cases `full_set_rows_above_sgr` and
`full_set_rows_below_sgr`; upstream-findings section 15, Screen item 32;
upstream-crashes.md). Rule: an error where upstream aborts.

- `Page.exactRowCapacityOrError` (page.go) is NOT a Zig declaration: the body of
  `exactRowCapacity`, with that one cast returning `ErrOutOfSpace` instead of
  trapping (the other casts stay assertions). `ExactRowCapacity` calls it and
  asserts the error is nil, so direct callers (and `PageList.Compact`) still
  panic where Zig traps (`TestPg1_panic_Page_exactRowCapacity_styles_capacity_overflows_u16`).
- `splitForCapacity` calls `exactRowCapacityOrError` for both ranges, in Zig's
  order, and returns the error before anything changes. `ErrOutOfSpace` is the
  error splitForCapacity already documents for "the page can't be split
  further", so every caller of `manualStyleUpdate` handles it as it handles
  that case upstream: `setAttribute`'s errdefer restores the previous style and
  returns the error, the stream handler drops it (`catch {}`), and printing
  continues with the previous style; `cursorChangePin`/`cursorReload`/
  `restoreCursor` fall back to the default style.
- The expected output is recorded from upstream with term51
  `patches-reference/0003` (the same behaviour in Zig: `splitForCapacity`
  returns `error.OutOfSpace`); the cases are `expect_crash`.
- Go-only tests: `Screen_split_rulings_test.go`
  (`TestScreenSplitRuling_full_style_set_in_the_rows_above_fails_the_style_change`,
  `TestScreenSplitRuling_full_style_set_in_the_rows_below_fails_the_style_change`).
  Rows for docs/go-only-tests.md:
  `| Screen_split_rulings_test.go | TestScreenSplitRuling_full_style_set_in_the_rows_above_fails_the_style_change | ruling (Screen.splitForCapacity returns ErrOutOfSpace where upstream aborts in Page.exactRowCapacity's style-capacity cast; docs/mapping.md "Page-layer bookkeeping") |`
  `| Screen_split_rulings_test.go | TestScreenSplitRuling_full_style_set_in_the_rows_below_fails_the_style_change | ruling (as above, the range below the cursor) |`

## Open (not implemented): stale cursor row pointers after a split that moves the cursor row

Upstream-findings section 15, Screen items 6 and 33; term51 case
`text/style_split/split_moves_cursor_row_erase` (`upstream_invalid`). After
`splitForCapacity` moves the cursor row to a new page, Zig's
`cursor.page_row`/`page_cell` still point into the old page and pointer
arithmetic on them (`cursorRight`, `cursorCellRight`, ...) keeps working there;
Go's `cellIndex` resolves the cell against the cursor's new page and panics
("cell is not in the page", `Screen.CursorRight`, Screen.go:1083, the first
print after the split). Awaiting the project lead's decision.
